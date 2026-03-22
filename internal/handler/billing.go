// internal/handler/billing.go
// user情報取得
// service呼び出し
// checkout_url を返す
package handler

import (
	"net/http"
	"log"

	"github.com/gin-gonic/gin"

	"dscheckerapp/internal/service"
)

type BillingHandler struct {
	billingService *service.BillingService
}

func NewBillingHandler(billingService *service.BillingService) *BillingHandler {
	return &BillingHandler{
		billingService: billingService,
	}
}

func (h *BillingHandler) CreateCheckoutSession(c *gin.Context) {
	userIDValue, ok := c.Get("clerk_user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "clerk_user_id not found"})
		return
	}

	emailValue, _ := c.Get("email")

	userID := toString(userIDValue)
	email := toString(emailValue)

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}

	checkoutURL, err := h.billingService.CreateCheckoutSession(email, userID)

	if err != nil {
		log.Printf("[CHECKOUT ERROR] %+v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"checkout_url": checkoutURL,
	})
}

func toString(v interface{}) string {
	s, _ := v.(string)
	return s
}