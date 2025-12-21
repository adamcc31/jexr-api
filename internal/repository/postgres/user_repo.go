package postgres

import (
	"context"
	"go-recruitment-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) domain.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (id, email, role, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.Exec(ctx, query, user.ID, user.Email, user.Role, user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, role, created_at, updated_at FROM users WHERE id = $1`
	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, role, created_at, updated_at FROM users WHERE email = $1`
	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) Update(ctx context.Context, user *domain.User) error {
	query := `UPDATE users SET email = $2, role = $3, updated_at = $4 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, user.ID, user.Email, user.Role, user.UpdatedAt)
	return err
}
