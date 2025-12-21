package postgres

import (
	"context"
	"errors"
	"go-recruitment-backend/internal/domain"

	// "go-recruitment-backend/pkg/database" // Not needed if using pgxpool directly for type

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type candidateRepository struct {
	db *pgxpool.Pool
}

func NewCandidateRepository(db *pgxpool.Pool) domain.CandidateRepository {
	return &candidateRepository{db: db}
}

func (r *candidateRepository) GetByUserID(ctx context.Context, userID string) (*domain.CandidateProfile, error) {
	query := `SELECT id, user_id, title, bio, skills, resume_url, created_at, updated_at 
	          FROM candidate_profiles WHERE user_id = $1`

	var p domain.CandidateProfile
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&p.ID, &p.UserID, &p.Title, &p.Bio, &p.Skills, &p.ResumeURL, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil if not found, let usecase handle 404
		}
		return nil, err
	}
	return &p, nil
}

func (r *candidateRepository) Create(ctx context.Context, profile *domain.CandidateProfile) error {
	// Implementation omitted for brevity as focus is on Get Endpoint
	return nil
}

func (r *candidateRepository) Update(ctx context.Context, profile *domain.CandidateProfile) error {
	// Implementation omitted
	return nil
}
