package middleware

import (
	"net/http"
	"strings"

	"ayam-batu-api/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const userIDContextKey = "user_id"

// Claims defines the JWT payload stored in access tokens.
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// AuthRequired validates the Bearer JWT token and stores user_id in Gin context.
func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "authorization header is required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			response.Error(c, http.StatusUnauthorized, "authorization header must be Bearer <token>")
			c.Abort()
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid || claims.UserID == 0 {
			response.Error(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(userIDContextKey, claims.UserID)
		c.Next()
	}
}

// GetUserID retrieves the authenticated user's ID from Gin context.
func GetUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get(userIDContextKey)
	if !exists {
		return 0, false
	}
	id, ok := val.(uint)
	return id, ok && id > 0
}
