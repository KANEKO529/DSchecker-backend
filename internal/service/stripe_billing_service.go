
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
	stripeConfig     *lib.StripeConfig
	stripeClient     *lib.StripeClient
	userRepo         BillingUserRepository
	subscriptionRepo BillingSubscriptionRepository
}

type PaymentMethodDTO struct {
	ID       string `json:"id"`
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth int64  `json:"exp_month"`
	ExpYear  int64  `json:"exp_year"`
}

func NewBillingService(
	stripeConfig *lib.StripeConfig,
	stripeClient *lib.StripeClient,
	userRepo BillingUserRepository,
	subscriptionRepo BillingSubscriptionRepository,
) *BillingService {
	return &BillingService{
		stripeConfig:     stripeConfig,
		stripeClient:     stripeClient,
		userRepo:         userRepo,
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
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
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

// clerkUserID で user を取得
// user.ID で subscription を取得
// subscription.stripe_subscription_id があれば Stripe で subscription を取得
// default_payment_method がなければ user.stripe_customer_id から customer を取得
// payment method を取得して返す
func (s *BillingService) GetCurrentPaymentMethod(ctx context.Context, clerkUserID string) (*PaymentMethodDTO, error) {
	user, err := s.userRepo.GetByClerkUserID(ctx, clerkUserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	var paymentMethodID string

	sub, err := s.subscriptionRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	if sub != nil && sub.StripeSubscriptionID != "" {
		stripeSub, err := s.stripeClient.GetSubscription(ctx, sub.StripeSubscriptionID)
		if err != nil {
			return nil, err
		}
		log.Println("subscription default_payment_method:", stripeSub.DefaultPaymentMethod)
	
		if stripeSub.DefaultPaymentMethod != nil {
			paymentMethodID = stripeSub.DefaultPaymentMethod.ID
		}
	}
	
	if paymentMethodID == "" && user.StripeCustomerID != nil && *user.StripeCustomerID != "" {
		stripeCus, err := s.stripeClient.GetCustomer(ctx, *user.StripeCustomerID)
		if err != nil {
			return nil, err
		}
		log.Println("customer default_payment_method:", stripeCus.InvoiceSettings.DefaultPaymentMethod)
	
		if stripeCus.InvoiceSettings != nil && stripeCus.InvoiceSettings.DefaultPaymentMethod != nil {
			paymentMethodID = stripeCus.InvoiceSettings.DefaultPaymentMethod.ID
		}
	}
	
	log.Println("resolved paymentMethodID:", paymentMethodID)


	if paymentMethodID == "" {
		return nil, nil
	}

	pm, err := s.stripeClient.GetPaymentMethod(ctx, paymentMethodID)

	log.Println("pm type:", pm.Type)
	log.Println("pm card:", pm.Card)
	
	if err != nil {
		return nil, err
	}
	if pm == nil || pm.Card == nil {
		return nil, nil
	}

	return &PaymentMethodDTO{
		ID:       pm.ID,
		Brand:    string(pm.Card.Brand),
		Last4:    pm.Card.Last4,
		ExpMonth: pm.Card.ExpMonth,
		ExpYear:  pm.Card.ExpYear,
	}, nil
}