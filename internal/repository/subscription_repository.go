// internal/repository/subscription_repository.go
package repository

import (
	"context"

	"dscheckerapp/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionRepository struct {
	db *pgxpool.Pool
}

func NewSubscriptionRepository(db *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) FindUserIDByStripeCustomerID(customerID string) (int64, error) {
	ctx := context.Background()

	var userID int64
	err := r.db.QueryRow(ctx, `
		select id
		from public.users
		where stripe_customer_id = $1
		  and deleted_at is null
	`, customerID).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (r *SubscriptionRepository) UpsertSubscription(input model.UpsertSubscriptionInput) error {
	ctx := context.Background()

	_, err := r.db.Exec(ctx, `
		insert into public.subscriptions (
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
		values (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		on conflict (stripe_subscription_id)
		do update set
			user_id = excluded.user_id,
			stripe_customer_id = excluded.stripe_customer_id,
			stripe_price_id = excluded.stripe_price_id,
			status = excluded.status,
			current_period_start = excluded.current_period_start,
			current_period_end = excluded.current_period_end,
			cancel_at_period_end = excluded.cancel_at_period_end,
			canceled_at = excluded.canceled_at,
			ended_at = excluded.ended_at,
			latest_event_id = excluded.latest_event_id,
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