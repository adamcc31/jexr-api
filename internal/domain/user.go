package domain

import (
	"context"
	"time"
)

type User struct {
	ID                  string    `json:"id"` // Supabase UUID
	Email               string    `json:"email"`
	Role                string    `json:"role"`
	OnboardingCompleted *bool     `json:"onboarding_completed,omitempty"` // Computed field, not in users table
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
}

type AuthUsecase interface {
	EnsureUserExists(ctx context.Context, user *User) error
	AssignRole(ctx context.Context, userID string, role string) error
	GetCurrentUser(ctx context.Context, id string) (*User, error)
	CheckEmailExists(ctx context.Context, email string) (bool, error)
}
