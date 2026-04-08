package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"dscheckerapp/internal/model"
	"dscheckerapp/internal/service"

	"github.com/gin-gonic/gin"
	svix "github.com/svix/svix-webhooks/go"
)

type ClerkWebhookHandler struct {
	Service *service.ClerkWebhookService
}

func NewClerkWebhookHandler(s *service.ClerkWebhookService) *ClerkWebhookHandler {
	return &ClerkWebhookHandler{Service: s}
}

func (h *ClerkWebhookHandler) Handle(c *gin.Context) {
	secret := os.Getenv("CLERK_WEBHOOK_SIGNING_SECRET")
	if secret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing webhook secret"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	wh, err := svix.NewWebhook(secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to init verifier"})
		return
	}

	if err := wh.Verify(body, c.Request.Header); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var evt model.ClerkWebhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	switch evt.Type {
	case "user.created":
		var payload model.ClerkUserPayload
		if err := json.Unmarshal(evt.Data, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user.created payload"})
			return
		}

		if err := h.Service.HandleUserCreated(c.Request.Context(), payload); err != nil {
			log.Printf("[CLERK WEBHOOK] failed to process user.created: err=%v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process user.created"})
			return
		}

	case "user.updated":
		var payload model.ClerkUserPayload
		if err := json.Unmarshal(evt.Data, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user.updated payload"})
			return
		}
		if err := h.Service.HandleUserUpdated(c.Request.Context(), payload); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process user.updated"})
			return
		}

	case "user.deleted":
		var payload model.ClerkDeletedUserPayload
		if err := json.Unmarshal(evt.Data, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user.deleted payload"})
			return
		}
		if err := h.Service.HandleUserDeleted(c.Request.Context(), payload.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process user.deleted"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook processed successfully"})
}
