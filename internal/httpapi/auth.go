package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

type accessRole uint8

const (
	producerRole accessRole = iota
	adminRole
)

func (a *API) requireRole(role accessRole, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.authorized(w, r, role) {
			return
		}
		next(w, r)
	})
}

func (a *API) requireRoleFunc(role func(*http.Request) accessRole, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.authorized(w, r, role(r)) {
			return
		}
		next(w, r)
	})
}

func (a *API) authorized(w http.ResponseWriter, r *http.Request, required accessRole) bool {
	if r.Header.Get("Authorization") == "" {
		csrfToken, ok := a.adminSessionCSRFToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "admin session required")
			return false
		}
		if csrfProtectedMethod(r.Method) &&
			subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(csrfToken)) != 1 {
			writeError(w, http.StatusForbidden, "valid CSRF token required")
			return false
		}
		return true
	}
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeError(w, http.StatusUnauthorized, "bearer token required")
		return false
	}
	providedHash := sha256.Sum256([]byte(token))
	producerMatch := subtle.ConstantTimeCompare(providedHash[:], a.producerTokenHash[:])
	adminMatch := subtle.ConstantTimeCompare(providedHash[:], a.adminTokenHash[:])
	if adminMatch == 1 || (required == producerRole && producerMatch == 1) {
		return true
	}
	if producerMatch == 1 {
		writeError(w, http.StatusForbidden, "admin role required")
		return false
	}
	writeError(w, http.StatusUnauthorized, "invalid bearer token")
	return false
}

// csrfProtectedMethod reports whether a request can change application state.
func csrfProtectedMethod(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func (a *API) sourceRole(r *http.Request) accessRole {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/sources/"), "/")
	if len(parts) == 1 && r.Method == http.MethodGet {
		return producerRole
	}
	if len(parts) == 2 && parts[1] == "audio" &&
		(r.Method == http.MethodGet || r.Method == http.MethodHead) {
		return producerRole
	}
	return adminRole
}

