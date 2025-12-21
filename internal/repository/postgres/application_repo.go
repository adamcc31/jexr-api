package postgres

import (
	"context"
	"go-recruitment-backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type applicationRepo struct {
	db *pgxpool.Pool
}

// NewApplicationRepository creates a new application repository
func NewApplicationRepository(db *pgxpool.Pool) domain.ApplicationRepository {
	return &applicationRepo{db: db}
}

// Create inserts a new application
func (r *applicationRepo) Create(ctx context.Context, app *domain.Application) error {
	query := `
		INSERT INTO applications (job_id, candidate_user_id, account_verification_id, cv_url, cover_letter, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	now := time.Now()
	app.CreatedAt = now
	app.UpdatedAt = now
	if app.Status == "" {
		app.Status = domain.ApplicationStatusApplied
	}

	return r.db.QueryRow(ctx, query,
		app.JobID,
		app.CandidateUserID,
		app.AccountVerificationID,
		app.CvURL,
		app.CoverLetter,
		app.Status,
		app.CreatedAt,
		app.UpdatedAt,
	).Scan(&app.ID)
}

// GetByID retrieves an application by ID with joined candidate data
func (r *applicationRepo) GetByID(ctx context.Context, id int64) (*domain.Application, error) {
	query := `
		SELECT 
			a.id, a.job_id, a.candidate_user_id, a.account_verification_id, 
			a.cv_url, a.cover_letter, a.status, a.created_at, a.updated_at,
			COALESCE(av.first_name || ' ' || av.last_name, u.email) as candidate_name,
			av.profile_picture_url as candidate_photo,
			av.status as verification_status,
			j.title as job_title
		FROM applications a
		LEFT JOIN users u ON a.candidate_user_id = u.id
		LEFT JOIN account_verifications av ON a.account_verification_id = av.id
		LEFT JOIN jobs j ON a.job_id = j.id
		WHERE a.id = $1`

	var app domain.Application
	err := r.db.QueryRow(ctx, query, id).Scan(
		&app.ID, &app.JobID, &app.CandidateUserID, &app.AccountVerificationID,
		&app.CvURL, &app.CoverLetter, &app.Status, &app.CreatedAt, &app.UpdatedAt,
		&app.CandidateName, &app.CandidatePhoto, &app.VerificationStatus, &app.JobTitle,
	)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// GetByJobID retrieves all applications for a job with joined candidate data
func (r *applicationRepo) GetByJobID(ctx context.Context, jobID int64) ([]domain.Application, error) {
	query := `
		SELECT 
			a.id, a.job_id, a.candidate_user_id, a.account_verification_id, 
			a.cv_url, a.cover_letter, a.status, a.created_at, a.updated_at,
			COALESCE(av.first_name || ' ' || av.last_name, u.email) as candidate_name,
			av.profile_picture_url as candidate_photo,
			av.status as verification_status
		FROM applications a
		LEFT JOIN users u ON a.candidate_user_id = u.id
		LEFT JOIN account_verifications av ON a.account_verification_id = av.id
		WHERE a.job_id = $1
		ORDER BY a.created_at DESC`

	rows, err := r.db.Query(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var applications []domain.Application
	for rows.Next() {
		var app domain.Application
		if err := rows.Scan(
			&app.ID, &app.JobID, &app.CandidateUserID, &app.AccountVerificationID,
			&app.CvURL, &app.CoverLetter, &app.Status, &app.CreatedAt, &app.UpdatedAt,
			&app.CandidateName, &app.CandidatePhoto, &app.VerificationStatus,
		); err != nil {
			return nil, err
		}
		applications = append(applications, app)
	}
	return applications, nil
}

// GetByUserID retrieves all applications for a user with job titles
func (r *applicationRepo) GetByUserID(ctx context.Context, userID string) ([]domain.Application, error) {
	query := `
		SELECT 
			a.id, a.job_id, a.candidate_user_id, a.account_verification_id, 
			a.cv_url, a.cover_letter, a.status, a.created_at, a.updated_at,
			j.title as job_title
		FROM applications a
		LEFT JOIN jobs j ON a.job_id = j.id
		WHERE a.candidate_user_id = $1
		ORDER BY a.created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var applications []domain.Application
	for rows.Next() {
		var app domain.Application
		if err := rows.Scan(
			&app.ID, &app.JobID, &app.CandidateUserID, &app.AccountVerificationID,
			&app.CvURL, &app.CoverLetter, &app.Status, &app.CreatedAt, &app.UpdatedAt,
			&app.JobTitle,
		); err != nil {
			return nil, err
		}
		applications = append(applications, app)
	}
	return applications, nil
}

// CheckExists checks if an application already exists for the job/user combination
func (r *applicationRepo) CheckExists(ctx context.Context, jobID int64, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM applications WHERE job_id = $1 AND candidate_user_id = $2)`
	var exists bool
	err := r.db.QueryRow(ctx, query, jobID, userID).Scan(&exists)
	return exists, err
}

// UpdateStatus updates the status of an application and sets updated_at
func (r *applicationRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE applications SET status = $2, updated_at = $3 WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id, status, time.Now())
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
