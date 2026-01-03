package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-recruitment-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

type candidateRepository struct {
	db *pgxpool.Pool
}

func NewCandidateRepository(db *pgxpool.Pool) domain.CandidateRepository {
	return &candidateRepository{db: db}
}

func (r *candidateRepository) GetByUserID(ctx context.Context, userID string) (*domain.CandidateProfile, error) {
	// This is the legacy/simple getter, updated to fetch new fields
	query := `
		SELECT 
			id, user_id, title, bio, 
			COALESCE(highest_education, ''), COALESCE(major_field, ''), 
			COALESCE(desired_job_position, ''), COALESCE(desired_job_position_other, ''), 
			COALESCE(preferred_work_environment, ''), COALESCE(career_goals_3y, ''), 
			main_concerns_returning, COALESCE(special_message, ''), COALESCE(skills_other, ''),
			resume_url, created_at, updated_at 
		FROM candidate_profiles WHERE user_id = $1`

	var p domain.CandidateProfile
	var mainConcerns []string

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&p.ID, &p.UserID, &p.Title, &p.Bio,
		&p.HighestEducation, &p.MajorField,
		&p.DesiredJobPosition, &p.DesiredJobPositionOther,
		&p.PreferredWorkEnvironment, &p.CareerGoals3y,
		pq.Array(&mainConcerns), &p.SpecialMessage, &p.SkillsOther,
		&p.ResumeURL, &p.CreatedAt, &p.UpdatedAt,
	)
	p.MainConcernsReturning = mainConcerns

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *candidateRepository) Create(ctx context.Context, profile *domain.CandidateProfile) error {
	// Basic create, mostly for initial setup if needed
	query := `INSERT INTO candidate_profiles (user_id, title, bio, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`
	_, err := r.db.Exec(ctx, query, profile.UserID, profile.Title, profile.Bio)
	return err
}

func (r *candidateRepository) Update(ctx context.Context, profile *domain.CandidateProfile) error {
	// Simple update
	query := `UPDATE candidate_profiles SET title=$1, bio=$2, resume_url=$3, updated_at=NOW() WHERE user_id=$4`
	_, err := r.db.Exec(ctx, query, profile.Title, profile.Bio, profile.ResumeURL, profile.UserID)
	return err
}

// =================================================================================================
// Transactional Full Profile Operations
// =================================================================================================

func (r *candidateRepository) GetFullProfile(ctx context.Context, userID string) (*domain.CandidateWithFullDetails, error) {
	// 1. Get Core Profile
	profile, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, nil // Not found
	}

	result := &domain.CandidateWithFullDetails{
		Profile:         *profile,
		WorkExperiences: []domain.WorkExperience{},
		Certificates:    []domain.CandidateCertificate{},
		Skills:          []domain.Skill{},
		SkillIDs:        []int{},
	}

	// 2. Get Dictionary Details
	detailsQuery := `SELECT soft_skills_description, applied_work_values, major_achievements 
	                 FROM candidate_details WHERE user_id = $1`
	var details domain.CandidateDetail
	details.UserID = userID
	err = r.db.QueryRow(ctx, detailsQuery, userID).Scan(
		&details.SoftSkillsDescription, &details.AppliedWorkValues, &details.MajorAchievements,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to fetch details: %w", err)
	}
	// If no rows, we leave empty strings (default)
	result.Details = details

	// 3. Get Work Experiences
	workQuery := `SELECT id, user_id, country_code, experience_type, company_name, job_title, start_date, end_date, description 
	              FROM work_experiences WHERE user_id = $1 ORDER BY start_date DESC`
	rows, err := r.db.Query(ctx, workQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch work exp: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var w domain.WorkExperience
		var startDate, endDate *time.Time
		err := rows.Scan(
			&w.ID, &w.UserID, &w.CountryCode, &w.ExperienceType, &w.CompanyName, &w.JobTitle,
			&startDate, &endDate, &w.Description,
		)
		if err != nil {
			return nil, err
		}
		// Format dates YYYY-MM-DD
		if startDate != nil {
			w.StartDate = startDate.Format("2006-01-02")
		}
		if endDate != nil {
			ed := endDate.Format("2006-01-02")
			w.EndDate = &ed
		}
		result.WorkExperiences = append(result.WorkExperiences, w)
	}

	// 4. Get Skills (Pivot + Master)
	skillsQuery := `
		SELECT s.id, s.name, s.category 
		FROM candidate_skills cs
		JOIN skills s ON cs.skill_id = s.id
		WHERE cs.user_id = $1`

	sRows, err := r.db.Query(ctx, skillsQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skills: %w", err)
	}
	defer sRows.Close()

	for sRows.Next() {
		var s domain.Skill
		if err := sRows.Scan(&s.ID, &s.Name, &s.Category); err != nil {
			return nil, err
		}
		result.Skills = append(result.Skills, s)
		result.SkillIDs = append(result.SkillIDs, s.ID)
	}

	// 5. Get Certificates (TOEFL, IELTS, TOEIC, OTHER - NOT JLPT)
	certQuery := `SELECT id, user_id, certificate_type, COALESCE(certificate_name, ''), 
	              score_total, score_details, issued_date, expires_date, document_file_path
	              FROM candidate_certificates WHERE user_id = $1 ORDER BY created_at DESC`
	certRows, err := r.db.Query(ctx, certQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch certificates: %w", err)
	}
	defer certRows.Close()

	for certRows.Next() {
		var c domain.CandidateCertificate
		var issuedDate, expiresDate *time.Time
		err := certRows.Scan(
			&c.ID, &c.UserID, &c.CertificateType, &c.CertificateName,
			&c.ScoreTotal, &c.ScoreDetails, &issuedDate, &expiresDate, &c.DocumentFilePath,
		)
		if err != nil {
			return nil, err
		}
		if issuedDate != nil {
			s := issuedDate.Format("2006-01-02")
			c.IssuedDate = &s
		}
		if expiresDate != nil {
			s := expiresDate.Format("2006-01-02")
			c.ExpiresDate = &s
		}
		result.Certificates = append(result.Certificates, c)
	}

	return result, nil
}

func (r *candidateRepository) UpsertFullProfile(ctx context.Context, full *domain.CandidateWithFullDetails) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	userID := full.Profile.UserID
	if userID == "" {
		return errors.New("user_id is required")
	}

	// 1. Update Core Profile
	// Using UPSERT simulation (Update, if 0 rows then Insert, though usually user exists)
	// We assume profile row exists for candidates. If not, we should probably INSERT.
	// Let's try Update first.
	updateProfileQuery := `
		UPDATE candidate_profiles SET
			title = $1, bio = $2,
			highest_education = $3, major_field = $4,
			desired_job_position = $5, desired_job_position_other = $6,
			preferred_work_environment = $7, career_goals_3y = $8,
			main_concerns_returning = $9, special_message = $10,
			skills_other = $11, resume_url = $12,
			updated_at = NOW()
		WHERE user_id = $13`

	cmdTag, err := tx.Exec(ctx, updateProfileQuery,
		full.Profile.Title, full.Profile.Bio,
		full.Profile.HighestEducation, full.Profile.MajorField,
		full.Profile.DesiredJobPosition, full.Profile.DesiredJobPositionOther,
		full.Profile.PreferredWorkEnvironment, full.Profile.CareerGoals3y,
		pq.Array(full.Profile.MainConcernsReturning), full.Profile.SpecialMessage,
		full.Profile.SkillsOther, full.Profile.ResumeURL,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		// Attempt Insert
		insertQuery := `
			INSERT INTO candidate_profiles (
				user_id, title, bio, highest_education, major_field, 
				desired_job_position, desired_job_position_other,
				preferred_work_environment, career_goals_3y,
				main_concerns_returning, special_message, skills_other, resume_url
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
		_, err := tx.Exec(ctx, insertQuery,
			userID, full.Profile.Title, full.Profile.Bio,
			full.Profile.HighestEducation, full.Profile.MajorField,
			full.Profile.DesiredJobPosition, full.Profile.DesiredJobPositionOther,
			full.Profile.PreferredWorkEnvironment, full.Profile.CareerGoals3y,
			pq.Array(full.Profile.MainConcernsReturning), full.Profile.SpecialMessage,
			full.Profile.SkillsOther, full.Profile.ResumeURL,
		)
		if err != nil {
			return fmt.Errorf("failed to insert profile: %w", err)
		}
	}

	// 2. Upsert Details
	detailsQuery := `
		INSERT INTO candidate_details (user_id, soft_skills_description, applied_work_values, major_achievements, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			soft_skills_description = EXCLUDED.soft_skills_description,
			applied_work_values = EXCLUDED.applied_work_values,
			major_achievements = EXCLUDED.major_achievements,
			updated_at = NOW()`
	_, err = tx.Exec(ctx, detailsQuery, userID, full.Details.SoftSkillsDescription, full.Details.AppliedWorkValues, full.Details.MajorAchievements)
	if err != nil {
		return fmt.Errorf("failed to upsert details: %w", err)
	}

	// 3. Work Experiences (Delete All -> Insert)
	_, err = tx.Exec(ctx, `DELETE FROM work_experiences WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete work exp: %w", err)
	}

	if len(full.WorkExperiences) > 0 {
		// Bulk insert or loop? Loop is easier to write, pgx has CopyFrom for bulk but CopyFrom requires simple types.
		// Prepare statement isn't necessary for small batch.
		weInsert := `
			INSERT INTO work_experiences (
				user_id, country_code, experience_type, company_name, job_title, start_date, end_date, description
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

		for _, we := range full.WorkExperiences {
			// Parse Dates
			start, _ := time.Parse("2006-01-02", we.StartDate)
			var end *time.Time
			if we.EndDate != nil && *we.EndDate != "" {
				t, _ := time.Parse("2006-01-02", *we.EndDate)
				end = &t
			}

			_, err := tx.Exec(ctx, weInsert, userID, we.CountryCode, we.ExperienceType, we.CompanyName, we.JobTitle, start, end, we.Description)
			if err != nil {
				return fmt.Errorf("failed to insert work exp: %w", err)
			}
		}
	}

	// 4. Skills (Delete Pivot -> Insert New)
	_, err = tx.Exec(ctx, `DELETE FROM candidate_skills WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to clean skills: %w", err)
	}

	if len(full.SkillIDs) > 0 {
		skillInsert := `INSERT INTO candidate_skills (user_id, skill_id) VALUES ($1, $2)`
		for _, sid := range full.SkillIDs {
			_, err := tx.Exec(ctx, skillInsert, userID, sid)
			if err != nil {
				return fmt.Errorf("failed to insert skill %d: %w", sid, err)
			}
		}
	}

	// 5. Certificates (Delete All -> Insert New)
	_, err = tx.Exec(ctx, `DELETE FROM candidate_certificates WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete certificates: %w", err)
	}

	if len(full.Certificates) > 0 {
		certInsert := `
			INSERT INTO candidate_certificates (
				user_id, certificate_type, certificate_name, score_total, score_details,
				issued_date, expires_date, document_file_path
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

		for _, cert := range full.Certificates {
			var issuedDate, expiresDate *time.Time
			if cert.IssuedDate != nil && *cert.IssuedDate != "" {
				t, _ := time.Parse("2006-01-02", *cert.IssuedDate)
				issuedDate = &t
			}
			if cert.ExpiresDate != nil && *cert.ExpiresDate != "" {
				t, _ := time.Parse("2006-01-02", *cert.ExpiresDate)
				expiresDate = &t
			}

			_, err := tx.Exec(ctx, certInsert,
				userID, cert.CertificateType, cert.CertificateName,
				cert.ScoreTotal, cert.ScoreDetails, issuedDate, expiresDate, cert.DocumentFilePath,
			)
			if err != nil {
				return fmt.Errorf("failed to insert certificate: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *candidateRepository) GetAllSkills(ctx context.Context) ([]domain.Skill, error) {
	query := `SELECT id, name, category FROM skills ORDER BY category, name`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []domain.Skill
	for rows.Next() {
		var s domain.Skill
		if err := rows.Scan(&s.ID, &s.Name, &s.Category); err != nil {
			return nil, err
		}
		skills = append(skills, s)
	}
	return skills, nil
}
