
// internal/service/stripe_billing_service.go
// Checkout Session 作成
package service

import (
	"context"
	"errors"
	"log"

	"dscheckerapp/internal/lib"
	"dscheckerapp/internal/model"

	"github.com/stripe/stripe-go/v84"
	stripeCheckoutSession "github.com/stripe/stripe-go/v84/checkout/session"
	stripeCustomer "github.com/stripe/stripe-go/v84/customer"
)

var ErrAlreadySubscribed = errors.New("already subscribed")

type BillingUserRepository interface {
	GetByClerkUserID(ctx context.Context, clerkUserID string) (*model.User, error)
	UpdateStripeCustomerID(ctx context.Context, userID int64, stripeCustomerID string) error
}

type BillingSubscriptionRepository interface {
	FindByUserID(ctx context.Context, userID int64) (*model.Subscription, error)
}

type BillingService struct {
	stripeConfig *lib.StripeConfig
	userRepo     BillingUserRepository
	subscriptionRepo BillingSubscriptionRepository
}

func NewBillingService(
	stripeConfig *lib.StripeConfig,
	userRepo BillingUserRepository,
	subscriptionRepo BillingSubscriptionRepository,
) *BillingService {
	return &BillingService{
		stripeConfig: stripeConfig,
		userRepo:     userRepo,
		subscriptionRepo: subscriptionRepo,
	}
}

// user 取得
// subscription 取得
// active / trialing ならエラー返す
// それ以外なら customer を再利用 or 新規作成
// session 作成
func (s *BillingService) CreateCheckoutSession(ctx context.Context, customerEmail string, clerkUserID string) (string, error) {
	
	// user 取得
	user, err := s.userRepo.GetByClerkUserID(ctx, clerkUserID)
	if err != nil {
		log.Printf("[CHECKOUT ERROR] failed to find user by clerk_user_id=%s: %v", clerkUserID, err)
		return "", err
	}

	// subscription 取得
	sub, err := s.subscriptionRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		log.Printf("[CHECKOUT ERROR] failed to find subscription by user_id=%d: %v", user.ID, err)
		return "", err
	}
	if sub != nil && (sub.Status == "active" || sub.Status == "trialing") {
		log.Printf("[CHECKOUT BLOCKED] already subscribed: user_id=%d status=%s", user.ID, sub.Status)
		return "", ErrAlreadySubscribed
	}

	// customer を再利用 or 新規作成
	var stripeCustomerID string

	if user.StripeCustomerID != nil && *user.StripeCustomerID != "" {
		stripeCustomerID = *user.StripeCustomerID
	} else {
		customerParams := &stripe.CustomerParams{}
		if customerEmail != "" {
			customerParams.Email = stripe.String(customerEmail)
		}

		c, err := stripeCustomer.New(customerParams) // 新規作成
		if err != nil {
			log.Printf("[STRIPE ERROR] failed to create customer: %v", err)
			return "", err
		}

		stripeCustomerID = c.ID
		// stripe_customer_id を users テーブルに保存（更新）
		if err := s.userRepo.UpdateStripeCustomerID(ctx, user.ID, stripeCustomerID); err != nil {
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