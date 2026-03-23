// internal/handler/subscription.go
package handler

import (
	"net/http"

	"dscheckerapp/internal/service"
	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	subscriptionService *service.SubscriptionService
}

func NewSubscriptionHandler(subscriptionService *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionService: subscriptionService,
	}
}

func (h *SubscriptionHandler) GetMySubscription(c *gin.Context) {
	clerkUserIDValue, exists := c.Get("clerk_user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "unauthorized",
		})
		return
	}

	clerkUserID, ok := clerkUserIDValue.(string)
	if !ok || clerkUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "invalid user context",
		})
		return
	}

	subscription, err := h.subscriptionService.GetMySubscriptionByClerkUserID(
		c.Request.Context(),
		clerkUserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "failed to fetch subscription",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"subscription": subscription,
		},
	})
}