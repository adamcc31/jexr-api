package domain

import (
	"context"
	"time"
)

type CandidateProfile struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id" validate:"required"`
	Title     string    `json:"title" validate:"required,min=3,max=100"`
	Bio       string    `json:"bio" validate:"max=500"`
	Skills    []string  `json:"skills" validate:"required,min=1"`
	ResumeURL string    `json:"resume_url" validate:"omitempty,url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CandidateRepository interface {
	GetByUserID(ctx context.Context, userID string) (*CandidateProfile, error)
	Create(ctx context.Context, profile *CandidateProfile) error
	Update(ctx context.Context, profile *CandidateProfile) error
}

type CandidateUsecase interface {
	GetProfile(ctx context.Context, userID string) (*CandidateProfile, error)
	UpdateProfile(ctx context.Context, profile *CandidateProfile) error
}
