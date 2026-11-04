package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	adminSessionCookie = "ycp_admin_session"
	adminSessionTTL    = 8 * time.Hour
)

// sessionLoginRequest contains the admin credential exchanged for a session.
type sessionLoginRequest struct {
	Token string "json:\"token\""
}

// loginSession exchanges the admin token for a time-limited signed cookie.
func (a *API) loginSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request sessionLoginRequest
	if len(a.sessionSecret) == 0 {
		writeError(w, http.StatusInternalServerError, "session service unavailable")
		return
	}
	if decodeJSON(w, r, &request) != nil {
		writeError(w, http.StatusBadRequest, "invalid session login request")
		return
	}
	if !a.isAdminToken(request.Token) {
		writeError(w, http.StatusUnauthorized, "invalid admin token")
		return
	}
	expires := time.Now().Add(adminSessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name: adminSessionCookie, Value: a.signSession(expires.Unix()), Path: "/",
		Expires: expires, MaxAge: int(adminSessionTTL.Seconds()), HttpOnly: true,
		Secure: true, SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// logoutSession expires the current admin session cookie.
func (a *API) logoutSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: adminSessionCookie, Value: "", Path: "/", MaxAge: -1,
		Expires: time.Unix(1, 0), HttpOnly: true, Secure: true,
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// hasAdminSession validates the signature and expiry carried by the cookie.
func (a *API) hasAdminSession(r *http.Request) bool {
	if len(a.sessionSecret) == 0 {
		return false
	}
	cookie, err := r.Cookie(adminSessionCookie)
	if err != nil {
		return false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return false
	}
	expiresAt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() >= expiresAt {
		return false
	}
	providedSignature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	expectedSignature := a.sessionSignature(parts[0])
	return hmac.Equal(providedSignature, expectedSignature)
}

// signSession encodes an expiry timestamp and its HMAC signature.
func (a *API) signSession(expiresAt int64) string {
	expiry := strconv.FormatInt(expiresAt, 10)
	signature := base64.RawURLEncoding.EncodeToString(a.sessionSignature(expiry))
	return expiry + "." + signature
}

// sessionSignature returns the HMAC-SHA256 signature for a session value.
func (a *API) sessionSignature(value string) []byte {
	signature := hmac.New(sha256.New, a.sessionSecret)
	_, _ = signature.Write([]byte(value))
	return signature.Sum(nil)
}

