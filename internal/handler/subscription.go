// internal/handler/subscription.go
package handler

import (
	"net/http"
	"errors"

	"dscheckerapp/internal/repository"
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

// CancelMySubscription
func (h *SubscriptionHandler) CancelMySubscription(c *gin.Context) {
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

	err := h.subscriptionService.CancelMySubscriptionByClerkUserID(
		c.Request.Context(),
		clerkUserID,
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrSubscriptionNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "subscription not found",
			})
		case errors.Is(err, service.ErrSubscriptionNotCancelable):
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "subscription not cancelable",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "failed to cancel subscription",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "subscription will be canceled at period end",
	})
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

func (h *SubscriptionHandler) ResumeMySubscription(c *gin.Context) {
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

	err := h.subscriptionService.ResumeMySubscriptionByClerkUserID(
		c.Request.Context(),
		clerkUserID,
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrSubscriptionNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "subscription not found",
			})
		case errors.Is(err, service.ErrSubscriptionNotResumable):
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "subscription not resumable",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "failed to resume subscription",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "subscription resumed",
	})
}