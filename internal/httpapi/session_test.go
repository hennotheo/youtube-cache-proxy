package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRememberedAdminSession(t *testing.T) {
	for _, test := range []struct {
		name, body string
		ttl        time.Duration
	}{
		{"default", `{"token":"admin"}`, 8 * time.Hour},
		{"remember", `{"token":"admin","remember":true}`, 30 * 24 * time.Hour},
		{"unchecked", `{"token":"admin","remember":false}`, 8 * time.Hour},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := NewAPI(nil, nil, nil, "producer", "admin", "secret")
			router := NewRouter(a)
			r := httptest.NewRequest("POST", "https://cache.test/session/login", strings.NewReader(test.body))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != 204 {
				t.Fatalf("login: %d %s", w.Code, w.Body)
			}
			cookies := w.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatal(cookies)
			}
			c := cookies[0]
			if c.MaxAge != int(test.ttl.Seconds()) || time.Until(c.Expires) < test.ttl-time.Minute || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode {
				t.Fatalf("cookie: %+v", c)
			}
			if strings.Contains(c.Value, "admin") {
				t.Fatal("bearer stored in cookie")
			}
			// A new API instance restores the same session, as after server/browser reopening.
			reopened := NewRouter(NewAPI(nil, nil, nil, "producer", "admin", "secret"))
			r = httptest.NewRequest("GET", "https://cache.test/session/csrf", nil)
			r.AddCookie(c)
			w = httptest.NewRecorder()
			reopened.ServeHTTP(w, r)
			if w.Code != 200 {
				t.Fatalf("restore: %d %s", w.Code, w.Body)
			}
		})
	}
}
func TestRememberDoesNotBypassAdminAuthentication(t *testing.T) {
	router := NewRouter(NewAPI(nil, nil, nil, "producer", "admin", "secret"))
	r := httptest.NewRequest("POST", "https://cache.test/session/login", strings.NewReader(`{"token":"producer","remember":true}`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 401 || len(w.Result().Cookies()) != 0 {
		t.Fatal(w.Code, w.Result().Cookies())
	}
}
