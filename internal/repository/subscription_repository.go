// internal/repository/subscription_repository.go
package repository

import (
	"context"
	"errors"

	"dscheckerapp/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

type SubscriptionRepository struct {
	db *pgxpool.Pool
}

func NewSubscriptionRepository(db *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// checkout 用
// FindByUserID returns the current subscription for the user.
// If no current subscription exists, it returns nil, nil.
func (r *SubscriptionRepository) FindByUserID(ctx context.Context, userID int64) (*model.Subscription, error) {
	sub, err := r.FindCurrentByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return sub, nil
}

// 最新のsubscriptionレコードを取得
func (r *SubscriptionRepository) FindCurrentByUserID(ctx context.Context, userID int64) (*model.Subscription, error) {
	// 全レコードの中から優先順位をつけて1件選ぶ
	// active / trialing → 優先度 0
	// 解約予約中だけどまだ有効期間内 → 優先度 1
	// それ以外（終了済み・キャンセル済み・失敗など） → 優先度 2

	// const q = `
	// 	SELECT
	// 		id,
	// 		user_id,
	// 		stripe_customer_id,
	// 		stripe_subscription_id,
	// 		stripe_price_id,
	// 		status,
	// 		current_period_start,
	// 		current_period_end,
	// 		cancel_at_period_end,
	// 		canceled_at,
	// 		ended_at,
	// 		latest_event_id,
	// 		created_at,
	// 		updated_at
	// 	FROM subscriptions
	// 	WHERE user_id = $1
	// 	ORDER BY
	// 		CASE
	// 			WHEN status IN ('active', 'trialing') THEN 0
	// 			WHEN cancel_at_period_end = true
	// 			     AND current_period_end IS NOT NULL
	// 			     AND current_period_end > now() THEN 1
	// 			ELSE 2
	// 		END,
	// 		current_period_end DESC NULLS LAST,
	// 		created_at DESC
	// 	LIMIT 1
	// `

	// 有効候補だけに絞ってから1件選ぶ
	// まずWHEREで,active/trialing/解約予約中だがまだ期限内/ だけに絞る
	// 後に,その候補の中で,current_period_end が新しい順→created_at が新しい順　で1件返す
	const q = `
		SELECT
			id,
			user_id,
			stripe_customer_id,
			stripe_subscription_id,
			stripe_price_id,
			status,
			current_period_start,
			current_period_end,
			cancel_at_period_end,
			canceled_at,
			ended_at,
			latest_event_id,
			created_at,
			updated_at
		FROM subscriptions
		WHERE user_id = $1
		AND (
			status IN ('active', 'trialing')
			OR (
			cancel_at_period_end = true
			AND current_period_end IS NOT NULL
			AND current_period_end > now()
			)
		)
		ORDER BY current_period_end DESC NULLS LAST, created_at DESC
		LIMIT 1
	`

	var sub model.Subscription
	err := r.db.QueryRow(ctx, q, userID).Scan(
		&sub.ID,
		&sub.UserID,
		&sub.StripeCustomerID,
		&sub.StripeSubscriptionID,
		&sub.StripePriceID,
		&sub.Status,
		&sub.CurrentPeriodStart,
		&sub.CurrentPeriodEnd,
		&sub.CancelAtPeriodEnd,
		&sub.CanceledAt,
		&sub.EndedAt,
		&sub.LatestEventID,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}

	return &sub, nil
}

// me/subscription 用 使わない
// func (r *SubscriptionRepository) FindLatestByUserID(ctx context.Context, userID int64) (*model.Subscription, error) {
// 	const q = `
// 		SELECT
// 			id,
// 			user_id,
// 			stripe_customer_id,
// 			stripe_subscription_id,
// 			stripe_price_id,
// 			status,
// 			current_period_start,
// 			current_period_end,
// 			cancel_at_period_end,
// 			canceled_at,
// 			ended_at,
// 			latest_event_id,
// 			created_at,
// 			updated_at
// 		FROM subscriptions
// 		WHERE user_id = $1
// 		ORDER BY updated_at DESC
// 		LIMIT 1
// 	`

// 	var sub model.Subscription
// 	err := r.db.QueryRow(ctx, q, userID).Scan(
// 		&sub.ID,
// 		&sub.UserID,
// 		&sub.StripeCustomerID,
// 		&sub.StripeSubscriptionID,
// 		&sub.StripePriceID,
// 		&sub.Status,
// 		&sub.CurrentPeriodStart,
// 		&sub.CurrentPeriodEnd,
// 		&sub.CancelAtPeriodEnd,
// 		&sub.CanceledAt,
// 		&sub.EndedAt,
// 		&sub.LatestEventID,
// 		&sub.CreatedAt,
// 		&sub.UpdatedAt,
// 	)
// 	if err != nil {
// 		if errors.Is(err, pgx.ErrNoRows) {
// 			return nil, ErrSubscriptionNotFound
// 		}
// 		return nil, err
// 	}

// 	return &sub, nil
// }

func (r *SubscriptionRepository) FindUserIDByStripeCustomerID(ctx context.Context, customerID string) (int64, error) {
	var userID int64
	err := r.db.QueryRow(ctx, `
		SELECT id
		FROM public.users
		WHERE stripe_customer_id = $1
		  AND deleted_at IS NULL
	`, customerID).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (r *SubscriptionRepository) UpsertSubscription(ctx context.Context, input model.UpsertSubscriptionInput) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO public.subscriptions (
			user_id,
			stripe_customer_id,
			stripe_subscription_id,
			stripe_price_id,
			status,
			current_period_start,
			current_period_end,
			cancel_at_period_end,
			canceled_at,
			ended_at,
			latest_event_id
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (stripe_subscription_id)
		DO UPDATE SET
			user_id = EXCLUDED.user_id,
			stripe_customer_id = EXCLUDED.stripe_customer_id,
			stripe_price_id = EXCLUDED.stripe_price_id,
			status = EXCLUDED.status,
			current_period_start = EXCLUDED.current_period_start,
			current_period_end = EXCLUDED.current_period_end,
			cancel_at_period_end = EXCLUDED.cancel_at_period_end,
			canceled_at = EXCLUDED.canceled_at,
			ended_at = EXCLUDED.ended_at,
			latest_event_id = EXCLUDED.latest_event_id,
			updated_at = now()
	`,
		input.UserID,
		input.StripeCustomerID,
		input.StripeSubscriptionID,
		input.StripePriceID,
		string(input.Status),
		input.CurrentPeriodStart,
		input.CurrentPeriodEnd,
		input.CancelAtPeriodEnd,
		input.CanceledAt,
		input.EndedAt,
		input.LatestEventID,
	)
	return err
}
