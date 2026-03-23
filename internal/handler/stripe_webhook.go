// internal/handler/stripe_webhook.go
package handler

import (
	"io"
	"net/http"
	"context"
	"log"
	"dscheckerapp/internal/lib"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
)

type StripeWebhookService interface {
	HandleEvent(ctx context.Context, event stripe.Event) error
}

type StripeWebhookHandler struct {
	service StripeWebhookService
	cfg     *lib.StripeConfig
}

func NewStripeWebhookHandler(
	service StripeWebhookService,
	cfg *lib.StripeConfig,
) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		service: service,
		cfg:     cfg,
	}
}

func (h *StripeWebhookHandler) Handle(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[STRIPE WEBHOOK ERROR] failed to read request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	event, err := webhook.ConstructEvent(
		payload,
		c.GetHeader("Stripe-Signature"),
		h.cfg.WebhookSecret,
	)
	if err != nil {
		log.Printf("[STRIPE WEBHOOK ERROR] invalid signature: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stripe signature"})
		return
	}

	log.Printf("[STRIPE WEBHOOK] received event: id=%s type=%s", event.ID, event.Type)

	if err := h.service.HandleEvent(c.Request.Context(), event); err != nil {
		log.Printf("[STRIPE WEBHOOK ERROR] failed to handle event: id=%s type=%s err=%v", event.ID, event.Type, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[STRIPE WEBHOOK] handled successfully: id=%s type=%s", event.ID, event.Type)
	c.JSON(http.StatusOK, gin.H{"received": true})
}