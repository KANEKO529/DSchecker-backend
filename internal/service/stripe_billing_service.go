
// internal/service/stripe_billing_service.go
// Checkout Session 作成
package service

import (
	"context"
	"log"

	"dscheckerapp/internal/lib"
	"dscheckerapp/internal/model"

	"github.com/stripe/stripe-go/v84"
	stripeCheckoutSession "github.com/stripe/stripe-go/v84/checkout/session"
	stripeCustomer "github.com/stripe/stripe-go/v84/customer"
)

type BillingUserRepository interface {
	GetByClerkUserID(ctx context.Context, clerkUserID string) (*model.User, error)
	UpdateStripeCustomerID(userID int64, stripeCustomerID string) error
}

type BillingService struct {
	stripeConfig *lib.StripeConfig
	userRepo     BillingUserRepository
}

func NewBillingService(
	stripeConfig *lib.StripeConfig,
	userRepo BillingUserRepository,
) *BillingService {
	return &BillingService{
		stripeConfig: stripeConfig,
		userRepo:     userRepo,
	}
}

func (s *BillingService) CreateCheckoutSession(customerEmail string, clerkUserID string) (string, error) {
	ctx := context.Background()

	user, err := s.userRepo.GetByClerkUserID(ctx, clerkUserID)
	if err != nil {
		log.Printf("[CHECKOUT ERROR] failed to find user by clerk_user_id=%s: %v", clerkUserID, err)
		return "", err
	}

	stripeCustomerID := user.StripeCustomerID
	if stripeCustomerID == "" {
		customerParams := &stripe.CustomerParams{}
		if customerEmail != "" {
			customerParams.Email = stripe.String(customerEmail)
		}

		c, err := stripeCustomer.New(customerParams)
		if err != nil {
			log.Printf("[STRIPE ERROR] failed to create customer: %v", err)
			return "", err
		}

		stripeCustomerID = c.ID

		if err := s.userRepo.UpdateStripeCustomerID(user.ID, stripeCustomerID); err != nil {
			log.Printf("[CHECKOUT ERROR] failed to save stripe_customer_id for user_id=%d: %v", user.ID, err)
			return "", err
		}
	}

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
		ClientReferenceID: stripe.String(clerkUserID),
		Customer:          stripe.String(stripeCustomerID),
	}

	session, err := stripeCheckoutSession.New(params)
	if err != nil {
		log.Printf("[STRIPE ERROR] failed to create checkout session: %v", err)
		return "", err
	}

	log.Printf(
		"[STRIPE SUCCESS] checkout session created: session_id=%s clerk_user_id=%s internal_user_id=%d customer_id=%s",
		session.ID, clerkUserID, user.ID, stripeCustomerID,
	)

	return session.URL, nil
}