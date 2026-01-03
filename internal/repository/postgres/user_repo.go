package postgres

import (
	"context"
	"errors"
	"fmt"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgreSQL error codes
const (
	pgUniqueViolation = "23505"
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

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return apperror.Conflict("User with this email already exists")
		}
		return apperror.Internal(err)
	}
	return nil
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

// UpdateByEmail updates a user record by email, including changing the ID.
// This is used when user's Supabase ID changes (e.g., account recreation).
func (r *userRepo) UpdateByEmail(ctx context.Context, email string, user *domain.User) error {
	fmt.Printf("[DEBUG UpdateByEmail] Starting for email: %s, new ID: %s\n", email, user.ID)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		fmt.Printf("[DEBUG UpdateByEmail] Failed to begin transaction: %v\n", err)
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Get Old ID (needed to update non-FK tables that reference user_id by string)
	var oldID string
	err = tx.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&oldID)
	if err != nil {
		fmt.Printf("[DEBUG UpdateByEmail] Failed to get old ID: %v\n", err)
		return apperror.Internal(err)
	}
	fmt.Printf("[DEBUG UpdateByEmail] Old ID: %s\n", oldID)

	// 2. Update users table
	// This will automatically cascade to company_profiles, candidate_profiles, etc.
	// thanks to ON UPDATE CASCADE in migration 000016
	query := `UPDATE users SET id = $1, role = $2, updated_at = $3 WHERE email = $4`
	_, err = tx.Exec(ctx, query, user.ID, user.Role, user.UpdatedAt, email)
	if err != nil {
		fmt.Printf("[DEBUG UpdateByEmail] Failed to update users table: %v\n", err)
		return apperror.Internal(err)
	}
	fmt.Printf("[DEBUG UpdateByEmail] Updated users table successfully\n")

	// 3. Manually update tables without Foreign Key constraints (candidate onboarding data)
	// These tables use user_id as TEXT and don't cascade automatically

	// candidate_interests
	_, err = tx.Exec(ctx, "UPDATE candidate_interests SET user_id = $1 WHERE user_id = $2", user.ID, oldID)
	if err != nil {
		fmt.Printf("[DEBUG UpdateByEmail] Failed to update candidate_interests: %v\n", err)
		return apperror.Internal(err)
	}

	// candidate_company_preferences
	_, err = tx.Exec(ctx, "UPDATE candidate_company_preferences SET user_id = $1 WHERE user_id = $2", user.ID, oldID)
	if err != nil {
		fmt.Printf("[DEBUG UpdateByEmail] Failed to update candidate_company_preferences: %v\n", err)
		return apperror.Internal(err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Printf("[DEBUG UpdateByEmail] Failed to commit transaction: %v\n", err)
		return apperror.Internal(err)
	}
	fmt.Printf("[DEBUG UpdateByEmail] Transaction committed successfully!\n")
	return nil
}
