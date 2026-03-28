// internal/handler/billing.go
// user情報取得
// service呼び出し
// checkout_url を返す
package handler

import (
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"dscheckerapp/internal/service"
)


type BillingHandler struct {
	billingService *service.BillingService
}

type BillingSummaryResponse struct {
	PaymentMethods []service.PaymentMethodItem `json:"payment_methods"`
	Invoices       []service.InvoiceListItem   `json:"invoices"`
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

	clerkUserID := toString(userIDValue)
	email := toString(emailValue)

	checkoutURL, err := h.billingService.CreateCheckoutSession(c.Request.Context(), email, clerkUserID)
	if err != nil {
		log.Printf("[CHECKOUT ERROR] %+v", err)

		if errors.Is(err, service.ErrAlreadySubscribed) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "already subscribed",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create checkout session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"checkout_url": checkoutURL,
		},
	})
}

func toString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func (h *BillingHandler) GetBillingSummary(c *gin.Context) {
	clerkUserIDValue, exists := c.Get("clerk_user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	clerkUserID, ok := clerkUserIDValue.(string)
	if !ok || clerkUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid clerk user id"})
		return
	}

	ctx := c.Request.Context()

	var (
		wg       sync.WaitGroup
		pms      []service.PaymentMethodItem
		invoices []service.InvoiceListItem
		err1     error
		err2     error
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		pms, err1 = h.billingService.GetListPaymentMethods(ctx, clerkUserID)
	}()

	go func() {
		defer wg.Done()
		invoices, err2 = h.billingService.ListInvoices(ctx, clerkUserID)
	}()

	wg.Wait()

	if err1 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get payment methods"})
		return
	}

	if err2 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch invoices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": BillingSummaryResponse{
			PaymentMethods: pms,
			Invoices:       invoices,
		},
	})
}

// 3/28 API統合のため廃止
// func (h *BillingHandler) GetPaymentMethods(c *gin.Context) {
// 	clerkUserIDValue, exists := c.Get("clerk_user_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
// 		return
// 	}

// 	clerkUserID, ok := clerkUserIDValue.(string)
// 	if !ok || clerkUserID == "" {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid clerk user id"})
// 		return
// 	}

// 	pms, err := h.billingService.GetListPaymentMethods(c.Request.Context(), clerkUserID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get payment methods"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status": "success",
// 		"data": gin.H{
// 			"payment_methods": pms,
// 		},
// 	})
// }

// 3/28 API統合のため廃止
// func (h *BillingHandler) GetListInvoices(c *gin.Context) {
// 	clerkUserIDValue, exists := c.Get("clerk_user_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
// 		return
// 	}

// 	clerkUserID, ok := clerkUserIDValue.(string)
// 	if !ok {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
// 		return
// 	}

// 	invoices, err := h.billingService.ListInvoices(c.Request.Context(), clerkUserID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch invoices"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"invoices": invoices,
// 	})
// }

func (h *BillingHandler) CreateCustomerPortal(c *gin.Context) {
	clerkUserIDValue, exists := c.Get("clerk_user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	clerkUserID, ok := clerkUserIDValue.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}

	url, err := h.billingService.CreateCustomerPortalSession(c.Request.Context(), clerkUserID)
	if err != nil {
		if errors.Is(err, service.ErrStripeCustomerNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "stripe customer not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer portal session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}