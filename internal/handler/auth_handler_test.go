package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ayam-batu-api/internal/service"

	"github.com/gin-gonic/gin"
)

func TestGoogleOAuthRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("google login returns 503 when not configured", func(t *testing.T) {
		svc := service.NewAuthService(nil, "test-secret", 24)
		h := NewAuthHandler(svc)

		r := gin.New()
		r.GET("/api/v1/auth/google/login", h.GoogleLogin)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("google login redirects to Google with state cookie when configured", func(t *testing.T) {
		svc := service.NewAuthService(nil, "test-secret", 24, service.GoogleOAuthOptions{
			ClientID:     "client-id-123",
			ClientSecret: "client-secret-456",
			RedirectURL:  "http://localhost:8081/api/v1/auth/google/callback",
		})
		h := NewAuthHandler(svc)

		r := gin.New()
		r.GET("/api/v1/auth/google/login", h.GoogleLogin)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusTemporaryRedirect {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusTemporaryRedirect)
		}
		loc := rec.Header().Get("Location")
		if !strings.Contains(loc, "accounts.google.com") || !strings.Contains(loc, "state=") {
			t.Fatalf("unexpected redirect location: %q", loc)
		}
		if len(rec.Result().Cookies()) == 0 {
			t.Fatalf("expected oauth_state cookie to be set")
		}
	})

	t.Run("google callback rejects mismatched oauth state", func(t *testing.T) {
		svc := service.NewAuthService(nil, "test-secret", 24, service.GoogleOAuthOptions{
			ClientID:     "client-id-123",
			ClientSecret: "client-secret-456",
			RedirectURL:  "http://localhost:8081/api/v1/auth/google/callback",
		})
		h := NewAuthHandler(svc)

		r := gin.New()
		r.GET("/api/v1/auth/google/callback", h.GoogleCallback)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?state=bad&code=abc", nil)
		req.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: "different"})
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}
