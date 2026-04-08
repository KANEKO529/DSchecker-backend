// internal/repository/user_repository.go
// dbを操作する場所
package repository

import (
	"context"
	"fmt"

	"dscheckerapp/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(userID int64) (*model.User, error) {
	var user model.User

	err := r.db.QueryRow(context.Background(), `
		select
			id,
			clerk_user_id,
			role,
			user_name,
			email,
			stripe_customer_id,
			status,
			created_at,
			updated_at,
			deleted_at
		from public.users
		where id = $1
	`, userID).Scan(
		&user.ID,
		&user.ClerkUserID,
		&user.Role,
		&user.UserName,
		&user.Email,
		&user.StripeCustomerID,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdateStripeCustomerID(ctx context.Context, userID int64, stripeCustomerID string) error {
	// DBの実行結果は ct に入っている
	ct, err := r.db.Exec(ctx, `
		update public.users
		set stripe_customer_id = $1,
		    updated_at = now()
		where id = $2
	`, stripeCustomerID, userID)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return fmt.Errorf("user not found: user_id=%d", userID)
	}
	return nil
}

func (r *UserRepository) GetByClerkUserID(ctx context.Context, clerkUserID string) (*model.User, error) {
	query := `
		SELECT
			id,
			clerk_user_id,
			role,
			user_name,
			email,
			stripe_customer_id,
			status,
			created_at,
			updated_at,
			deleted_at
		FROM public.users
		WHERE clerk_user_id = $1
		  AND deleted_at IS NULL
		LIMIT 1
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, clerkUserID).Scan(
		&user.ID,
		&user.ClerkUserID,
		&user.Role,
		&user.UserName,
		&user.Email,
		&user.StripeCustomerID,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUsers(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.Query(ctx, `
		select
			id,
			clerk_user_id,
			role,
			user_name,
			email,
			status,
			created_at,
			updated_at,
			deleted_at
		from public.users
		order by id asc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []model.User{}

	for rows.Next() {
		var user model.User
		if err := rows.Scan(
			&user.ID,
			&user.ClerkUserID,
			&user.Role,
			&user.UserName,
			&user.Email,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.DeletedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// 同じ clerk_user_id のユーザーを新規作成 or 更新する
// 削除済みユーザーが、同じ clerk_user_id で再作成・再同期されたときに復活できるよう、deleted_at = NULL を入れる
// user.created / user.updated が来た時に、そのユーザーが論理削除済みでも再有効化
// user.created の初回同期
// user.updated の同期
// 何らかの理由で同じ Clerk ユーザーを再同期したいとき
func (r *UserRepository) UpsertUser(ctx context.Context, user model.User) error {
	query := `
		INSERT INTO public.users (
			clerk_user_id, email, user_name, role, status, created_at, updated_at, deleted_at
		)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), NULL)
		ON CONFLICT (clerk_user_id)
		DO UPDATE SET
			email = EXCLUDED.email,
			user_name = EXCLUDED.user_name,
			role = EXCLUDED.role,
			status = EXCLUDED.status,
			deleted_at = NULL,
			updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query,
		user.ClerkUserID,
		user.Email,
		user.UserName,
		user.Role,
		user.Status,
	)
	return err
}

// Clerk の user.deleted が来ても、レコード自体は残る
func (r *UserRepository) DeleteUserByClerkID(ctx context.Context, clerkUserID string) error {
	query := `
		UPDATE public.users
		SET 
			status = 'deleted',
			deleted_at = NOW(),
		    updated_at = NOW()
		WHERE clerk_user_id = $1
	`
	_, err := r.db.Exec(ctx, query, clerkUserID)
	return err
}
