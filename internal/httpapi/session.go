package httpapi

import (
	"crypto/hmac"
	"crypto/rand"
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
	csrfToken, err := newCSRFCredential()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create admin session")
		return
	}
	expires := time.Now().Add(adminSessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name: adminSessionCookie, Value: a.signSession(expires.Unix(), csrfToken), Path: "/",
		Expires: expires, MaxAge: int(adminSessionTTL.Seconds()), HttpOnly: true,
		Secure: true, SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// csrfToken returns the CSRF credential bound to the authenticated session.
func (a *API) csrfToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	token, ok := a.adminSessionCSRFToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "admin session required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
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

// adminSessionCSRFToken validates the session and returns its bound CSRF credential.
func (a *API) adminSessionCSRFToken(r *http.Request) (string, bool) {
	if len(a.sessionSecret) == 0 {
		return "", false
	}
	cookie, err := r.Cookie(adminSessionCookie)
	if err != nil {
		return "", false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 {
		return "", false
	}
	expiresAt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() >= expiresAt {
		return "", false
	}
	providedSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", false
	}
	expectedSignature := a.sessionSignature(parts[0] + "." + parts[1])
	if !hmac.Equal(providedSignature, expectedSignature) {
		return "", false
	}
	return parts[1], true
}

// signSession encodes an expiry timestamp, CSRF credential, and HMAC signature.
func (a *API) signSession(expiresAt int64, csrfToken string) string {
	expiry := strconv.FormatInt(expiresAt, 10)
	value := expiry + "." + csrfToken
	signature := base64.RawURLEncoding.EncodeToString(a.sessionSignature(value))
	return value + "." + signature
}

// newCSRFCredential creates an unguessable value bound to an admin session.
func newCSRFCredential() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

// sessionSignature returns the HMAC-SHA256 signature for a session value.
func (a *API) sessionSignature(value string) []byte {
	signature := hmac.New(sha256.New, a.sessionSecret)
	_, _ = signature.Write([]byte(value))
	return signature.Sum(nil)
}

