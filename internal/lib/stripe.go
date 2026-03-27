// internal/lib/stripe.go
// Stripe設定読込
// stripe.Key 初期化
package lib

import (
	"fmt"
	"os"
	"context"

	"github.com/stripe/stripe-go/v84"
	stripeSub "github.com/stripe/stripe-go/v84/subscription"
	stripeCustomer "github.com/stripe/stripe-go/v84/customer"
	stripePaymentMethod "github.com/stripe/stripe-go/v84/paymentmethod"
)

// type StripeSubscriptionClient interface {
// 	CancelAtPeriodEnd(ctx context.Context, stripeSubscriptionID string) error
// 	ResumeSubscription(ctx context.Context, stripeSubscriptionID string) error
// }

type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	PriceID       string
	AppURL        string
}

type StripeClient struct {

}

func NewStripeClient() *StripeClient {
	return &StripeClient{}
}


func LoadStripeConfig() (*StripeConfig, error) {
	cfg := &StripeConfig{
		SecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		PriceID:       os.Getenv("STRIPE_PRICE_ID"),
		AppURL:        os.Getenv("CORS_ORIGIN"),
	}

	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY is required")
	}
	if cfg.WebhookSecret == "" {
		return nil, fmt.Errorf("STRIPE_WEBHOOK_SECRET is required")
	}
	if cfg.PriceID == "" {
		return nil, fmt.Errorf("STRIPE_PRICE_ID is required")
	}
	if cfg.AppURL == "" {
		return nil, fmt.Errorf("CORS_ORIGIN is required")
	}

	return cfg, nil
}

func InitStripe(cfg *StripeConfig) {
	stripe.Key = cfg.SecretKey
}

// 外部API通信
func (c *StripeClient) CancelAtPeriodEnd(ctx context.Context, stripeSubscriptionID string) error {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	params.Context = ctx

	_, err := stripeSub.Update(stripeSubscriptionID, params)
	return err
}

// 解約取り消し
func (c *StripeClient) ResumeSubscription(ctx context.Context, stripeSubscriptionID string) error {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}
	params.Context = ctx

	_, err := stripeSub.Update(stripeSubscriptionID, params)
	return err
}

func (c *StripeClient) GetSubscription(ctx context.Context, stripeSubscriptionID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{}
	params.Context = ctx
	return stripeSub.Get(stripeSubscriptionID, params)
}
func (c *StripeClient) GetCustomer(ctx context.Context, stripeCustomerID string) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{}
	params.Context = ctx
	return stripeCustomer.Get(stripeCustomerID, params)
}

func (c *StripeClient) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*stripe.PaymentMethod, error) {
	params := &stripe.PaymentMethodParams{}
	params.Context = ctx

	// 👇これ追加
	params.AddExpand("card")

	return stripePaymentMethod.Get(paymentMethodID, params)
}

func (c *StripeClient) ListCustomerPaymentMethods(ctx context.Context, stripeCustomerID string) ([]*stripe.PaymentMethod, error) {
	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(stripeCustomerID),
		Type:     stripe.String("card"),
	}
	params.Context = ctx

	iter := stripePaymentMethod.List(params)

	var methods []*stripe.PaymentMethod
	for iter.Next() {
		methods = append(methods, iter.PaymentMethod())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return methods, nil
}