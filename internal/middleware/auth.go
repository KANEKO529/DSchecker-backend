// internal/middleware/auth.go
package middleware

import (
	"log"
	"net/http"

	"dscheckerapp/internal/auth"
	"dscheckerapp/internal/repository"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	verifier *auth.ClerkVerifier
	UserRepo *repository.UserRepository
}

func NewAuthMiddleware(verifier *auth.ClerkVerifier, userRepo *repository.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{
		verifier: verifier,
		UserRepo: userRepo,
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

		user, err := m.UserRepo.GetByClerkUserID(c.Request.Context(), claims.Subject)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "error",
				"error": gin.H{
					"code":    "USER_RESOLUTION_FAILED",
					"message": "failed to resolve local user",
				},
			})
			c.Abort()
			return
		}
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "error",
				"error": gin.H{
					"code":    "USER_NOT_FOUND",
					"message": "local user not found",
				},
			})
			c.Abort()
			return
		}
		c.Set("user_id", user.ID)

		c.Next()
	}
}

func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := auth.GetSessionTokenFromRequest(c.Request)

		if token == "" {
			log.Printf("[AUTH OPTIONAL] no token -> continue as guest")
			c.Next()
			return
		}

		claims, err := m.verifier.VerifySessionToken(c.Request.Context(), token)

		if err != nil {
			log.Printf("[AUTH OPTIONAL] verify failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "error",
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "invalid or expired token",
				},
			})
			c.Abort()
			return
		}

		c.Set("clerk_user_id", claims.Subject)
		c.Set("session_id", claims.SessionID)
		c.Set("session_claims", claims)

		user, err := m.UserRepo.GetByClerkUserID(c.Request.Context(), claims.Subject)

		if user != nil {
			c.Set("user_id", user.ID)
		}
		c.Next()
	}
}
