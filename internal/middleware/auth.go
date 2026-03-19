// internal/middleware/auth.go
package middleware

import (
	"net/http"
	"log"

	"dscheckerapp/internal/auth"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	verifier *auth.ClerkVerifier
}

func NewAuthMiddleware(verifier *auth.ClerkVerifier) *AuthMiddleware {
	return &AuthMiddleware{
		verifier: verifier,
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := auth.GetSessionTokenFromRequest(c.Request)
		if token == "" {
			log.Println("[AUTH] token missing")
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "error",
				"error":  "authentication token is missing",
			})
			c.Abort()
			return
		}

		claims, err := m.verifier.VerifySessionToken(c.Request.Context(), token)
		if err != nil {
			log.Printf("[AUTH] token invalid: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "error",
				"error":  "invalid or expired token",
			})
			c.Abort()
			return
		}

		// ✅ 成功ログ（ここがポイント）
		log.Printf("[AUTH SUCCESS] user=%s session=%s", claims.Subject, claims.SessionID)

		c.Set("clerk_user_id", claims.Subject)
		c.Set("session_id", claims.SessionID)
		c.Set("session_claims", claims)

		c.Next()
	}
}