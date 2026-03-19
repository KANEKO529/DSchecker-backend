// intenal/handler/me.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *UserHandler) Me(c *gin.Context) {
	clerkUserID, _ := c.Get("clerk_user_id")
	sessionID, _ := c.Get("session_id")

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"clerk_user_id": clerkUserID,
			"session_id":    sessionID,
		},
	})
}