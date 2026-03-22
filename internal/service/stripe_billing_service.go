
// internal/service/stripe_billing_service.go
// Checkout Session 作成
package service

import (
	"dscheckerapp/internal/lib"
	"log"

	"github.com/stripe/stripe-go/v84"
	stripeCheckoutSession "github.com/stripe/stripe-go/v84/checkout/session"
)

type BillingService struct {
	stripeConfig *lib.StripeConfig
}

func NewBillingService(stripeConfig *lib.StripeConfig) *BillingService {
	return &BillingService{
		stripeConfig: stripeConfig,
	}
}

func (s *BillingService) CreateCheckoutSession(customerEmail string, userID string) (string, error) {
	priceID := s.stripeConfig.PriceID
	appURL := s.stripeConfig.AppURL

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(appURL + "/billing/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(appURL + "/billing/cancel"),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		ClientReferenceID: stripe.String(userID),
	}

	if customerEmail != "" {
		params.CustomerEmail = stripe.String(customerEmail)
	}

	session, err := stripeCheckoutSession.New(params)
	if err != nil {
		return "", err
	}

	log.Printf("[STRIPE SUCCESS] checkout session created: session_id=%s user_id=%s", session.ID, userID)

	return session.URL, nil
}