
// internal/service/stripe_billing_service.go
// Checkout Session 作成
package service

import (
	"context"
	"errors"
	"log"
	"time"

	"dscheckerapp/internal/lib"
	"dscheckerapp/internal/model"

	"github.com/stripe/stripe-go/v84"
	stripeCheckoutSession "github.com/stripe/stripe-go/v84/checkout/session"
	stripeCustomer "github.com/stripe/stripe-go/v84/customer"
)

var ErrAlreadySubscribed = errors.New("already subscribed")
var ErrStripeCustomerNotFound = errors.New("stripe customer not found")


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

// 支払い方法取得用DTO
type PaymentMethodItem struct {
	ID        string `json:"id"`
	Brand     string `json:"brand"`
	Last4     string `json:"last4"`
	ExpMonth  int64  `json:"expMonth"`
	ExpYear   int64  `json:"expYear"`
	IsDefault bool   `json:"isDefault"`

	BillingDetails BillingDetails `json:"billingDetails"`
}

// invoice一覧用DTO
type InvoiceListItem struct {
	InvoiceID        string    `json:"invoiceId"`
	InvoiceNumber    string    `json:"invoiceNumber"`
	BilledAt         time.Time `json:"billedAt"`
	AmountPaid       int64     `json:"amountPaid"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	StatusLabel      string    `json:"statusLabel"`
	SubscriptionName string    `json:"subscriptionName"`
	HostedInvoiceURL string    `json:"hostedInvoiceUrl"`
	InvoicePDF       string    `json:"invoicePdf"`
}


type BillingDetails struct {
	Name    string         `json:"name"`
	Email   string         `json:"email"`
	Phone   string         `json:"phone"`
	Address BillingAddress `json:"address"`
}

type BillingAddress struct {
	Country    string `json:"country"`
	PostalCode string `json:"postalCode"`
	State      string `json:"state"`
	City       string `json:"city"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
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
		BillingAddressCollection: stripe.String("required"),
		PhoneNumberCollection: &stripe.CheckoutSessionPhoneNumberCollectionParams{
			Enabled: stripe.Bool(true),
		},
	}

	// 保存同意UIを出す
	params.AddExtra("saved_payment_method_options[payment_method_save]", "enabled")

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
func (s *BillingService) GetListPaymentMethods(ctx context.Context, clerkUserID string) ([]PaymentMethodItem, error) {
	user, err := s.userRepo.GetByClerkUserID(ctx, clerkUserID)
	if err != nil {
		return nil, err
	}
	if user.StripeCustomerID == nil || *user.StripeCustomerID == "" {
		return []PaymentMethodItem{}, nil
	}

	customerID := *user.StripeCustomerID

	customer, err := s.stripeClient.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	methods, err := s.stripeClient.ListCustomerPaymentMethods(ctx, customerID)
	if err != nil {
		return nil, err
	}

	defaultPMID := ""
	if customer.InvoiceSettings != nil && customer.InvoiceSettings.DefaultPaymentMethod != nil {
		defaultPMID = customer.InvoiceSettings.DefaultPaymentMethod.ID
	}

	items := make([]PaymentMethodItem, 0, len(methods))
	for _, pm := range methods {
		item := mapPaymentMethodToItem(pm, pm.ID == defaultPMID)
		if item != nil {
			items = append(items, *item)
		}
	}

	return items, nil
}

func mapPaymentMethodToItem(pm *stripe.PaymentMethod, isDefault bool) *PaymentMethodItem {
	if pm == nil || pm.Card == nil {
		return nil
	}

	item := &PaymentMethodItem{
		ID:        pm.ID,
		Brand:     string(pm.Card.Brand),
		Last4:     pm.Card.Last4,
		ExpMonth:  pm.Card.ExpMonth,
		ExpYear:   pm.Card.ExpYear,
		IsDefault: isDefault,
	}

	if pm.BillingDetails != nil {
		item.BillingDetails.Name = pm.BillingDetails.Name
		item.BillingDetails.Email = pm.BillingDetails.Email
		item.BillingDetails.Phone = pm.BillingDetails.Phone

		if pm.BillingDetails.Address != nil {
			item.BillingDetails.Address = BillingAddress{
				Country:    pm.BillingDetails.Address.Country,
				PostalCode: pm.BillingDetails.Address.PostalCode,
				State:      pm.BillingDetails.Address.State,
				City:       pm.BillingDetails.Address.City,
				Line1:      pm.BillingDetails.Address.Line1,
				Line2:      pm.BillingDetails.Address.Line2,
			}
		}
	}

	return item
}

func (s *BillingService) ListInvoices(ctx context.Context, clerkUserID string) ([]InvoiceListItem, error) {
	user, err := s.userRepo.GetByClerkUserID(ctx, clerkUserID)
	if err != nil {
		return nil, err
	}
	if user.StripeCustomerID == nil || *user.StripeCustomerID == "" {
		return []InvoiceListItem{}, nil
	}

	// ここで取得
	invoices, err := s.stripeClient.ListInvoicesByCustomer(ctx, *user.StripeCustomerID, 20)
	if err != nil {
		return nil, err
	}

	items := make([]InvoiceListItem, 0, len(invoices))
	for _, inv := range invoices {
		billedAt := time.Unix(inv.Created, 0)

		items = append(items, InvoiceListItem{
			InvoiceID:        inv.ID,
			InvoiceNumber:    inv.Number,
			BilledAt:         billedAt,
			AmountPaid:       inv.AmountPaid,
			Currency:         string(inv.Currency),
			Status:           string(inv.Status),
			StatusLabel:      invoiceStatusLabel(inv.Status),
			SubscriptionName: extractSubscriptionName(inv),
			HostedInvoiceURL: inv.HostedInvoiceURL,
			InvoicePDF:       inv.InvoicePDF,
		})
	}

	return items, nil
}

func invoiceStatusLabel(status stripe.InvoiceStatus) string {
	switch status {
	case stripe.InvoiceStatusPaid:
		return "支払い済み"
	case stripe.InvoiceStatusOpen:
		return "未払い"
	case stripe.InvoiceStatusDraft:
		return "下書き"
	case stripe.InvoiceStatusVoid:
		return "無効"
	case stripe.InvoiceStatusUncollectible:
		return "回収不能"
	default:
		return "不明"
	}
}

func extractSubscriptionName(inv *stripe.Invoice) string {
	if inv.Lines != nil {
		for _, line := range inv.Lines.Data {
			if line.Description != "" {
				return line.Description
			}
		}
	}
	return "Proプラン"
}