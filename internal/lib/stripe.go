// internal/lib/stripe.go
// Stripe設定読込
// stripe.Key 初期化
package lib

import (
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v84"
)

type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	PriceID       string
	AppURL        string
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