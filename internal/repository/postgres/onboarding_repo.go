package postgres

import (
	"context"
	"fmt"
	"go-recruitment-backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type onboardingRepo struct {
	db *pgxpool.Pool
}

func NewOnboardingRepository(db *pgxpool.Pool) domain.OnboardingRepository {
	return &onboardingRepo{db: db}
}

// ============================================================================
// LPK Search
// ============================================================================

func (r *onboardingRepo) SearchLPK(ctx context.Context, query string, limit int) ([]domain.LPK, error) {
	if limit <= 0 || limit > 20 {
		limit = 20
	}

	// Use ILIKE for case-insensitive search with prefix matching
	// Falls back to trigram if pg_trgm extension is available
	sqlQuery := `
		SELECT id, name, created_at 
		FROM lpk_list 
		WHERE name ILIKE $1 
		ORDER BY name ASC 
		LIMIT $2
	`

	// Prepare search pattern: match anywhere in name
	searchPattern := "%" + query + "%"

	rows, err := r.db.Query(ctx, sqlQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search LPK: %w", err)
	}
	defer rows.Close()

	var results []domain.LPK
	for rows.Next() {
		var lpk domain.LPK
		if err := rows.Scan(&lpk.ID, &lpk.Name, &lpk.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan LPK row: %w", err)
		}
		results = append(results, lpk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating LPK rows: %w", err)
	}

	return results, nil
}

func (r *onboardingRepo) GetLPKByID(ctx context.Context, id int64) (*domain.LPK, error) {
	query := `SELECT id, name, created_at FROM lpk_list WHERE id = $1`

	var lpk domain.LPK
	err := r.db.QueryRow(ctx, query, id).Scan(&lpk.ID, &lpk.Name, &lpk.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("LPK not found with id: %d", id)
		}
		return nil, fmt.Errorf("failed to get LPK by ID: %w", err)
	}

	return &lpk, nil
}

// ============================================================================
// Onboarding Status
// ============================================================================

func (r *onboardingRepo) GetOnboardingStatus(ctx context.Context, userID string) (*domain.OnboardingStatus, error) {
	query := `
		SELECT onboarding_completed_at 
		FROM account_verifications 
		WHERE user_id = $1
	`

	var completedAt *time.Time
	err := r.db.QueryRow(ctx, query, userID).Scan(&completedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No verification record yet - onboarding not completed
			return &domain.OnboardingStatus{
				Completed:   false,
				CompletedAt: nil,
			}, nil
		}
		return nil, fmt.Errorf("failed to get onboarding status: %w", err)
	}

	return &domain.OnboardingStatus{
		Completed:   completedAt != nil,
		CompletedAt: completedAt,
	}, nil
}

// ============================================================================
// Save Onboarding Data (Atomic Transaction)
// ============================================================================

func (r *onboardingRepo) SaveOnboardingData(ctx context.Context, userID string, req *domain.OnboardingSubmitRequest) error {
	// Start transaction for atomicity
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// 1. Clear any existing interests for this user (idempotent)
	_, err = tx.Exec(ctx, `DELETE FROM candidate_interests WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to clear existing interests: %w", err)
	}

	// 2. Insert new interests
	for _, interest := range req.Interests {
		_, err = tx.Exec(ctx, `
			INSERT INTO candidate_interests (user_id, interest_key, created_at) 
			VALUES ($1, $2, NOW())
		`, userID, string(interest))
		if err != nil {
			return fmt.Errorf("failed to insert interest %s: %w", interest, err)
		}
	}

	// 3. Clear any existing company preferences for this user
	_, err = tx.Exec(ctx, `DELETE FROM candidate_company_preferences WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to clear existing company preferences: %w", err)
	}

	// 4. Insert new company preferences
	for _, pref := range req.CompanyPreferences {
		_, err = tx.Exec(ctx, `
			INSERT INTO candidate_company_preferences (user_id, preference_key, created_at) 
			VALUES ($1, $2, NOW())
		`, userID, string(pref))
		if err != nil {
			return fmt.Errorf("failed to insert company preference %s: %w", pref, err)
		}
	}

	// 5. Update account_verifications with LPK selection and completion timestamp
	// First, ensure the record exists
	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM account_verifications WHERE user_id = $1)`, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check verification record existence: %w", err)
	}

	if !exists {
		// Create a new verification record if it doesn't exist
		_, err = tx.Exec(ctx, `
			INSERT INTO account_verifications (user_id, lpk_id, lpk_other_name, lpk_none, onboarding_completed_at) 
			VALUES ($1, $2, $3, $4, NOW())
		`, userID, req.LPKSelection.LPKID, req.LPKSelection.OtherName, req.LPKSelection.None)
		if err != nil {
			return fmt.Errorf("failed to create verification record: %w", err)
		}
	} else {
		// Update existing record
		_, err = tx.Exec(ctx, `
			UPDATE account_verifications 
			SET lpk_id = $2, 
				lpk_other_name = $3, 
				lpk_none = $4, 
				onboarding_completed_at = NOW()
			WHERE user_id = $1
		`, userID, req.LPKSelection.LPKID, req.LPKSelection.OtherName, req.LPKSelection.None)
		if err != nil {
			return fmt.Errorf("failed to update verification record: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
