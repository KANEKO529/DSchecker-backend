// intenal/handler/me.go
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (h *UserHandler) Me(c *gin.Context) {
	clerkUserIDValue, exists := c.Get("clerk_user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": "error",
			"error":  "clerk_user_id not found in context",
		})
		return
	}

	clerkUserID, ok := clerkUserIDValue.(string)
	if !ok || clerkUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": "error",
			"error":  "invalid clerk_user_id",
		})
		return
	}

	user, err := h.UserRepo.GetByClerkUserID(c.Request.Context(), clerkUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"status": "error",
				"error":  "user not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "failed to fetch user",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"user": user,
		},
	})
}