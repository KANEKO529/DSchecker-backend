// internal/repository/usage_log_repository.go
package repository

import (
	"context"
	"dscheckerapp/internal/model"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UsageLogRepository struct {
	db *pgxpool.Pool
}

func NewUsageLogRepository(db *pgxpool.Pool) *UsageLogRepository {
	return &UsageLogRepository{db: db}
}

// 直近1時間の件数を数える
func (r *UsageLogRepository) CountRecentByUserID(
	ctx context.Context,
	userID int64,
	actionType string,
	since time.Time,
) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM usage_logs
		WHERE user_id = $1
		  AND action_type = $2
		  AND used_at >= $3
	`

	var count int
	err := r.db.QueryRow(ctx, q, userID, actionType, since).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// 直近1時間の件数を数える
func (r *UsageLogRepository) CountRecentByGuestIdentifier(
	ctx context.Context,
	guestIdentifier string,
	actionType string,
	since time.Time,
) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM usage_logs
		WHERE guest_identifier = $1
		  AND action_type = $2
		  AND used_at >= $3
	`

	var count int
	err := r.db.QueryRow(ctx, q, guestIdentifier, actionType, since).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// ログを1件追加する
func (r *UsageLogRepository) Create(ctx context.Context, usageLog model.UsageLog) error {
	const q = `
		INSERT INTO usage_logs (
			user_id,
			guest_identifier,
			action_type,
			used_at
		)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Exec(
		ctx,
		q,
		usageLog.UserID,
		usageLog.GuestIdentifier,
		usageLog.ActionType,
		usageLog.UsedAt,
	)
	return err
}
