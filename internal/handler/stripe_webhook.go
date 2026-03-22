// internal/handler/stripe_webhook.go
package handler

import (
	"io"
	"net/http"

	"dscheckerapp/internal/lib"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
)

type StripeWebhookService interface {
	HandleEvent(event stripe.Event) error
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	event, err := webhook.ConstructEvent(
		payload,
		c.GetHeader("Stripe-Signature"),
		h.cfg.WebhookSecret,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stripe signature"})
		return
	}

	if err := h.service.HandleEvent(event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}