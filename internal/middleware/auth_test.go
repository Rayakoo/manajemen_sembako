package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret-key"

	setupRouter := func() *gin.Engine {
		r := gin.New()
		r.GET("/protected", AuthRequired(secret), func(c *gin.Context) {
			uid, ok := GetUserID(c)
			if !ok {
				c.Status(http.StatusInternalServerError)
				return
			}
			c.JSON(http.StatusOK, gin.H{"user_id": uid})
		})
		return r
	}

	makeToken := func(userID uint, exp time.Time, signKey string) string {
		claims := Claims{
			UserID: userID,
			Email:  "user@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(exp),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}
		tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(signKey))
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}
		return tok
	}

	t.Run("missing header returns 401", func(t *testing.T) {
		r := setupRouter()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("wrong secret returns 401", func(t *testing.T) {
		r := setupRouter()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+makeToken(42, time.Now().Add(time.Hour), "other-secret"))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("expired token returns 401", func(t *testing.T) {
		r := setupRouter()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+makeToken(42, time.Now().Add(-time.Minute), secret))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("valid token passes with user_id", func(t *testing.T) {
		r := setupRouter()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+makeToken(42, time.Now().Add(time.Hour), secret))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}
