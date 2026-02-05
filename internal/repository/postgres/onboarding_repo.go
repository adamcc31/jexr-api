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
	// Check if user has any data in candidate_interests table
	// This is the source of truth for whether onboarding is completed
	query := `
		SELECT EXISTS(
			SELECT 1 FROM candidate_interests WHERE user_id = $1
		)
	`

	var hasInterests bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&hasInterests)
	if err != nil {
		return nil, fmt.Errorf("failed to check onboarding status: %w", err)
	}

	// Also get the completion timestamp if it exists
	var completedAt *time.Time
	if hasInterests {
		_ = r.db.QueryRow(ctx, `
			SELECT onboarding_completed_at 
			FROM account_verifications 
			WHERE user_id = $1
		`, userID).Scan(&completedAt)
	}

	return &domain.OnboardingStatus{
		Completed:   hasInterests,
		CompletedAt: completedAt,
	}, nil
}

// ============================================================================
// Get Onboarding Data
// ============================================================================

func (r *onboardingRepo) GetOnboardingData(ctx context.Context, userID string) (*domain.OnboardingData, error) {
	data := &domain.OnboardingData{
		Interests:          []domain.InterestKey{},
		CompanyPreferences: []domain.CompanyPreferenceKey{},
		LPKSelection: domain.LPKSelection{
			None: false,
		},
	}

	// 1. Get interests
	interestRows, err := r.db.Query(ctx, `
		SELECT interest_key FROM candidate_interests WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get interests: %w", err)
	}
	defer interestRows.Close()

	for interestRows.Next() {
		var key string
		if err := interestRows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan interest: %w", err)
		}
		data.Interests = append(data.Interests, domain.InterestKey(key))
	}

	// 2. Get company preferences
	prefRows, err := r.db.Query(ctx, `
		SELECT preference_key FROM candidate_company_preferences WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get company preferences: %w", err)
	}
	defer prefRows.Close()

	for prefRows.Next() {
		var key string
		if err := prefRows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan company preference: %w", err)
		}
		data.CompanyPreferences = append(data.CompanyPreferences, domain.CompanyPreferenceKey(key))
	}

	// 3. Get LPK selection, completion status, and interview willingness
	var lpkID *int64
	var lpkOtherName *string
	var lpkNone *bool
	var completedAt *time.Time
	var lpkName *string
	var willingToInterviewOnsite *bool

	err = r.db.QueryRow(ctx, `
		SELECT av.lpk_id, av.lpk_other_name, av.lpk_none, av.onboarding_completed_at, lpk.name, av.willing_to_interview_onsite
		FROM account_verifications av
		LEFT JOIN lpk_list lpk ON av.lpk_id = lpk.id
		WHERE av.user_id = $1
	`, userID).Scan(&lpkID, &lpkOtherName, &lpkNone, &completedAt, &lpkName, &willingToInterviewOnsite)

	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get LPK selection: %w", err)
	}

	if err != pgx.ErrNoRows {
		data.LPKSelection.LPKID = lpkID
		data.LPKSelection.OtherName = lpkOtherName
		if lpkNone != nil {
			data.LPKSelection.None = *lpkNone
		}
		data.LPKName = lpkName
		data.CompletedAt = completedAt
		data.WillingToInterviewOnsite = willingToInterviewOnsite
	}

	return data, nil
}

// ============================================================================
// Save Onboarding Data (Atomic Transaction)
// ============================================================================

func (r *onboardingRepo) SaveOnboardingData(ctx context.Context, userID string, req *domain.OnboardingSubmitRequest) error {
	// Log onboarding submission attempt (Railway-traceable, no sensitive data)
	lpkType := "list"
	if req.LPKSelection.None {
		lpkType = "none"
	} else if req.LPKSelection.OtherName != nil && *req.LPKSelection.OtherName != "" {
		lpkType = "other"
	}
	fmt.Printf("[Onboarding] userID=%s interests=%d lpk_type=%s company_prefs=%d\n",
		userID, len(req.Interests), lpkType, len(req.CompanyPreferences))

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
		// Default role to 'CANDIDATE' as only candidates go through onboarding
		// IMPORTANT: Must be uppercase to match CHECK constraint: role IN ('ADMIN', 'EMPLOYER', 'CANDIDATE')
		_, err = tx.Exec(ctx, `
			INSERT INTO account_verifications (
				user_id, role, lpk_id, lpk_other_name, lpk_none, willing_to_interview_onsite,
				first_name, last_name, phone, gender, birth_date, onboarding_completed_at
			) 
			VALUES ($1, 'CANDIDATE', $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		`, userID, req.LPKSelection.LPKID, req.LPKSelection.OtherName, req.LPKSelection.None,
			req.WillingToInterviewOnsite, req.FirstName, req.LastName, req.Phone, req.Gender, req.BirthDate)
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
				willing_to_interview_onsite = $5,
				first_name = COALESCE($6, first_name),
				last_name = COALESCE($7, last_name),
				phone = COALESCE($8, phone),
				gender = COALESCE($9, gender),
				birth_date = COALESCE($10, birth_date),
				onboarding_completed_at = NOW()
			WHERE user_id = $1
		`, userID, req.LPKSelection.LPKID, req.LPKSelection.OtherName, req.LPKSelection.None,
			req.WillingToInterviewOnsite, req.FirstName, req.LastName, req.Phone, req.Gender, req.BirthDate)
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
