// DBを操作する場所
//
package repository

import (
	"context"

	"dscheckerapp/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	DB *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) GetUsers(ctx context.Context) ([]model.User, error) {
	rows, err := r.DB.Query(ctx, `
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

// 削除済みユーザーが、同じ clerk_user_id で再作成・再同期されたときに復活できるよう、deleted_at = NULL を入れる
// user.created / user.updated が来た時に、そのユーザーが論理削除済みでも再有効化
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
	_, err := r.DB.Exec(ctx, query,
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
		SET deleted_at = NOW(),
		    updated_at = NOW()
		WHERE clerk_user_id = $1
	`
	_, err := r.DB.Exec(ctx, query, clerkUserID)
	return err
}