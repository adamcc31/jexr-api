package postgres

import (
	"context"
	"errors"
	"fmt"
	"go-recruitment-backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type verificationRepo struct {
	db *pgxpool.Pool
}

func NewVerificationRepository(db *pgxpool.Pool) domain.VerificationRepository {
	return &verificationRepo{db: db}
}

func (r *verificationRepo) GetByUserID(ctx context.Context, userID string) (*domain.AccountVerification, error) {
	// Updated query to include new fields
	query := `
		SELECT 
			id, user_id, role, status, submitted_at, verified_at, verified_by, notes, created_at, updated_at,
			first_name, last_name, profile_picture_url, occupation, phone, website_url, intro, japan_experience_duration, japanese_certificate_url, cv_url, portfolio_url, japanese_level,
			birth_date, domicile_city, marital_status, children_count,
			main_job_fields, golden_skill, japanese_speaking_level,
			expected_salary, japan_return_date, available_start_date, preferred_locations, preferred_industries,
			supporting_certificates_url
		FROM account_verifications
		WHERE user_id = $1
	`
	var v domain.AccountVerification
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&v.ID, &v.UserID, &v.Role, &v.Status, &v.SubmittedAt, &v.VerifiedAt, &v.VerifiedBy, &v.Notes, &v.CreatedAt, &v.UpdatedAt,
		&v.FirstName, &v.LastName, &v.ProfilePictureURL, &v.Occupation, &v.Phone, &v.WebsiteURL, &v.Intro, &v.JapanExperienceDuration, &v.JapaneseCertificateURL, &v.CvURL, &v.PortfolioURL, &v.JapaneseLevel,
		&v.BirthDate, &v.DomicileCity, &v.MaritalStatus, &v.ChildrenCount,
		&v.MainJobFields, &v.GoldenSkill, &v.JapaneseSpeakingLevel,
		&v.ExpectedSalary, &v.JapanReturnDate, &v.AvailableStartDate, &v.PreferredLocations, &v.PreferredIndustries,
		&v.SupportingCertificatesURL,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found is not an error, just return nil
		}
		return nil, err
	}
	return &v, nil
}

func (r *verificationRepo) GetByID(ctx context.Context, id int64) (*domain.AccountVerification, error) {
	query := `
		SELECT 
			id, user_id, role, status, submitted_at, verified_at, verified_by, notes, created_at, updated_at,
			first_name, last_name, profile_picture_url, occupation, phone, website_url, intro, japan_experience_duration, japanese_certificate_url, cv_url, portfolio_url, japanese_level,
			birth_date, domicile_city, marital_status, children_count,
			main_job_fields, golden_skill, japanese_speaking_level,
			expected_salary, japan_return_date, available_start_date, preferred_locations, preferred_industries,
			supporting_certificates_url
		FROM account_verifications
		WHERE id = $1
	`
	var v domain.AccountVerification
	err := r.db.QueryRow(ctx, query, id).Scan(
		&v.ID, &v.UserID, &v.Role, &v.Status, &v.SubmittedAt, &v.VerifiedAt, &v.VerifiedBy, &v.Notes, &v.CreatedAt, &v.UpdatedAt,
		&v.FirstName, &v.LastName, &v.ProfilePictureURL, &v.Occupation, &v.Phone, &v.WebsiteURL, &v.Intro, &v.JapanExperienceDuration, &v.JapaneseCertificateURL, &v.CvURL, &v.PortfolioURL, &v.JapaneseLevel,
		&v.BirthDate, &v.DomicileCity, &v.MaritalStatus, &v.ChildrenCount,
		&v.MainJobFields, &v.GoldenSkill, &v.JapaneseSpeakingLevel,
		&v.ExpectedSalary, &v.JapanReturnDate, &v.AvailableStartDate, &v.PreferredLocations, &v.PreferredIndustries,
		&v.SupportingCertificatesURL,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *verificationRepo) List(ctx context.Context, filter domain.VerificationFilter) ([]domain.AccountVerification, int64, error) {
	// Build query dynamically
	baseQuery := `
		SELECT 
			av.id, av.user_id, av.role, av.status, av.submitted_at, av.verified_at, av.verified_by, av.notes, av.created_at, av.updated_at,
			av.first_name, av.last_name, av.profile_picture_url, av.occupation, av.phone, av.website_url, av.intro, av.japan_experience_duration, av.japanese_certificate_url, av.cv_url, av.portfolio_url, av.japanese_level,
			av.birth_date, av.domicile_city, av.marital_status, av.children_count,
			av.main_job_fields, av.golden_skill, av.japanese_speaking_level,
			av.expected_salary, av.japan_return_date, av.available_start_date, av.preferred_locations, av.preferred_industries,
			av.supporting_certificates_url,
			u.email,
			CASE 
				WHEN av.role = 'CANDIDATE' THEN COALESCE(av.first_name || ' ' || av.last_name, cp.title) -- Fallback to legacy title or concat name
				WHEN av.role = 'EMPLOYER' THEN comp.company_name
				ELSE ''
			END as profile_name
		FROM account_verifications av
		JOIN users u ON av.user_id = u.id
		LEFT JOIN candidate_profiles cp ON av.user_id = cp.user_id AND av.role = 'CANDIDATE'
		LEFT JOIN company_profiles comp ON av.user_id = comp.user_id AND av.role = 'EMPLOYER'
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM account_verifications WHERE 1=1`

	args := []interface{}{}
	argCounter := 1

	if filter.Role != "" {
		baseQuery += fmt.Sprintf(" AND av.role = $%d", argCounter)
		countQuery += fmt.Sprintf(" AND role = $%d", argCounter)
		args = append(args, filter.Role)
		argCounter++
	}

	if filter.Status != "" {
		baseQuery += fmt.Sprintf(" AND av.status = $%d", argCounter)
		countQuery += fmt.Sprintf(" AND status = $%d", argCounter)
		args = append(args, filter.Status)
		argCounter++
	}

	// Count total
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	baseQuery += fmt.Sprintf(" ORDER BY av.submitted_at DESC LIMIT $%d OFFSET $%d", argCounter, argCounter+1)
	offset := (filter.Page - 1) * filter.Limit
	args = append(args, filter.Limit, offset)

	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []domain.AccountVerification
	for rows.Next() {
		var v domain.AccountVerification
		var profileName string
		// Scan matches baseQuery select order
		err := rows.Scan(
			&v.ID, &v.UserID, &v.Role, &v.Status, &v.SubmittedAt, &v.VerifiedAt, &v.VerifiedBy, &v.Notes, &v.CreatedAt, &v.UpdatedAt,
			&v.FirstName, &v.LastName, &v.ProfilePictureURL, &v.Occupation, &v.Phone, &v.WebsiteURL, &v.Intro, &v.JapanExperienceDuration, &v.JapaneseCertificateURL, &v.CvURL, &v.PortfolioURL, &v.JapaneseLevel,
			&v.BirthDate, &v.DomicileCity, &v.MaritalStatus, &v.ChildrenCount,
			&v.MainJobFields, &v.GoldenSkill, &v.JapaneseSpeakingLevel,
			&v.ExpectedSalary, &v.JapanReturnDate, &v.AvailableStartDate, &v.PreferredLocations, &v.PreferredIndustries,
			&v.SupportingCertificatesURL,
			&v.UserEmail, &profileName,
		)
		if err != nil {
			continue // Or return error
		}

		v.UserProfile = &domain.UserProfileSummary{
			Name: profileName,
		}
		if v.Role == "EMPLOYER" {
			v.UserProfile.CompanyName = profileName
		}

		results = append(results, v)
	}

	if results == nil {
		results = []domain.AccountVerification{}
	}

	return results, total, nil
}

func (r *verificationRepo) UpdateStatus(ctx context.Context, id int64, status string, verifiedBy string, notes string) error {
	query := `
		UPDATE account_verifications
		SET status = $2, verified_by = $3, notes = $4, verified_at = $5, updated_at = $5
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, status, verifiedBy, notes, time.Now())
	return err
}

func (r *verificationRepo) Create(ctx context.Context, v *domain.AccountVerification) (int64, error) {
	query := `
		INSERT INTO account_verifications (
			user_id, role, status, submitted_at, created_at, updated_at,
			first_name, last_name, profile_picture_url, occupation, phone, website_url, intro, japan_experience_duration, japanese_certificate_url, cv_url, portfolio_url, japanese_level,
			birth_date, domicile_city, marital_status, children_count,
			main_job_fields, golden_skill, japanese_speaking_level,
			expected_salary, japan_return_date, available_start_date, preferred_locations, preferred_industries,
			supporting_certificates_url
		) VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30)
		RETURNING id
	`
	var id int64
	err := r.db.QueryRow(ctx, query,
		v.UserID, v.Role, v.Status, time.Now(), time.Now(),
		v.FirstName, v.LastName, v.ProfilePictureURL, v.Occupation, v.Phone, v.WebsiteURL, v.Intro, v.JapanExperienceDuration, v.JapaneseCertificateURL, v.CvURL, v.PortfolioURL, v.JapaneseLevel,
		v.BirthDate, v.DomicileCity, v.MaritalStatus, v.ChildrenCount,
		v.MainJobFields, v.GoldenSkill, v.JapaneseSpeakingLevel,
		v.ExpectedSalary, v.JapanReturnDate, v.AvailableStartDate, v.PreferredLocations, v.PreferredIndustries,
		v.SupportingCertificatesURL,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create verification: %w", err)
	}
	return id, nil
}

func (r *verificationRepo) UpdateProfile(ctx context.Context, v *domain.AccountVerification, experiences []domain.JapanWorkExperience) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Update account_verifications
	updateQuery := `
		UPDATE account_verifications
		SET 
			first_name = $2,
			last_name = $3,
			profile_picture_url = $4,
			occupation = $5,
			phone = $6,
			website_url = $7,
			intro = $8,
			japan_experience_duration = $9,
			japanese_certificate_url = $10,
			cv_url = $11,
			portfolio_url = $12,
			japanese_level = $13,
			status = $14,
			submitted_at = $15,
			birth_date = $16,
			domicile_city = $17,
			marital_status = $18,
			children_count = $19,
			main_job_fields = $20,
			golden_skill = $21,
			japanese_speaking_level = $22,
			expected_salary = $23,
			japan_return_date = $24,
			available_start_date = $25,
			preferred_locations = $26,
			preferred_industries = $27,
			supporting_certificates_url = $28,
			updated_at = $29
		WHERE id = $1
	`
	_, err = tx.Exec(ctx, updateQuery,
		v.ID,
		v.FirstName,
		v.LastName,
		v.ProfilePictureURL,
		v.Occupation,
		v.Phone,
		v.WebsiteURL,
		v.Intro,
		v.JapanExperienceDuration,
		v.JapaneseCertificateURL,
		v.CvURL,
		v.PortfolioURL,
		v.JapaneseLevel,
		v.Status,
		v.SubmittedAt,
		v.BirthDate,
		v.DomicileCity,
		v.MaritalStatus,
		v.ChildrenCount,
		v.MainJobFields,
		v.GoldenSkill,
		v.JapaneseSpeakingLevel,
		v.ExpectedSalary,
		v.JapanReturnDate,
		v.AvailableStartDate,
		v.PreferredLocations,
		v.PreferredIndustries,
		v.SupportingCertificatesURL,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	// 2. Delete existing work experiences
	_, err = tx.Exec(ctx, `DELETE FROM japan_work_experiences WHERE account_verification_id = $1`, v.ID)
	if err != nil {
		return fmt.Errorf("failed to delete old experiences: %w", err)
	}

	// 3. Insert new work experiences
	if len(experiences) > 0 {
		insertQuery := `
			INSERT INTO japan_work_experiences (
				account_verification_id, company_name, job_title, start_date, end_date, description, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		`
		for _, exp := range experiences {
			_, err = tx.Exec(ctx, insertQuery,
				v.ID,
				exp.CompanyName,
				exp.JobTitle,
				exp.StartDate,
				exp.EndDate,
				exp.Description,
				time.Now(),
			)
			if err != nil {
				return fmt.Errorf("failed to insert experience: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *verificationRepo) GetWorkExperiences(ctx context.Context, verificationID int64) ([]domain.JapanWorkExperience, error) {
	query := `
		SELECT id, account_verification_id, company_name, job_title, start_date, end_date, description, created_at, updated_at
		FROM japan_work_experiences
		WHERE account_verification_id = $1
		ORDER BY start_date DESC
	`
	rows, err := r.db.Query(ctx, query, verificationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var experiences []domain.JapanWorkExperience
	for rows.Next() {
		var exp domain.JapanWorkExperience
		err := rows.Scan(
			&exp.ID, &exp.AccountVerificationID, &exp.CompanyName, &exp.JobTitle,
			&exp.StartDate, &exp.EndDate, &exp.Description, &exp.CreatedAt, &exp.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		experiences = append(experiences, exp)
	}

	if experiences == nil {
		experiences = []domain.JapanWorkExperience{}
	}

	return experiences, nil
}

// GetComprehensiveByID fetches ALL candidate data for admin verification review
func (r *verificationRepo) GetComprehensiveByID(ctx context.Context, id int64) (*domain.ComprehensiveVerificationResponse, error) {
	// 1. Get account_verification by ID
	v, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}
	if v == nil {
		return nil, nil
	}

	userID := v.UserID
	response := &domain.ComprehensiveVerificationResponse{
		Verification: v,
	}

	// 2. Get japan_work_experiences (legacy)
	experiences, err := r.GetWorkExperiences(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get japan work experiences: %w", err)
	}
	response.Experiences = experiences

	// 3. Get candidate_profile
	profileQuery := `
		SELECT id, user_id, title, bio, highest_education, major_field, 
		       desired_job_position, desired_job_position_other, preferred_work_environment,
		       career_goals_3y, main_concerns_returning, special_message, skills_other, resume_url,
		       created_at, updated_at
		FROM candidate_profiles
		WHERE user_id = $1
	`
	var profile domain.CandidateProfile
	err = r.db.QueryRow(ctx, profileQuery, userID).Scan(
		&profile.ID, &profile.UserID, &profile.Title, &profile.Bio,
		&profile.HighestEducation, &profile.MajorField,
		&profile.DesiredJobPosition, &profile.DesiredJobPositionOther, &profile.PreferredWorkEnvironment,
		&profile.CareerGoals3y, &profile.MainConcernsReturning, &profile.SpecialMessage,
		&profile.SkillsOther, &profile.ResumeURL,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err == nil {
		response.CandidateProfile = &profile
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to get candidate profile: %w", err)
	}

	// 4. Get candidate_details
	detailsQuery := `
		SELECT user_id, soft_skills_description, applied_work_values, major_achievements,
		       created_at, updated_at
		FROM candidate_details
		WHERE user_id = $1
	`
	var details domain.CandidateDetail
	err = r.db.QueryRow(ctx, detailsQuery, userID).Scan(
		&details.UserID, &details.SoftSkillsDescription, &details.AppliedWorkValues,
		&details.MajorAchievements, &details.CreatedAt, &details.UpdatedAt,
	)
	if err == nil {
		response.CandidateDetails = &details
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to get candidate details: %w", err)
	}

	// 5. Get work_experiences (unified)
	workExpQuery := `
		SELECT id, user_id, country_code, experience_type, company_name, job_title,
		       start_date, end_date, description, created_at, updated_at
		FROM work_experiences
		WHERE user_id = $1
		ORDER BY start_date DESC
	`
	workRows, err := r.db.Query(ctx, workExpQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get work experiences: %w", err)
	}
	defer workRows.Close()

	var workExperiences []domain.WorkExperience
	for workRows.Next() {
		var we domain.WorkExperience
		var startDate, endDate *time.Time
		err := workRows.Scan(
			&we.ID, &we.UserID, &we.CountryCode, &we.ExperienceType, &we.CompanyName, &we.JobTitle,
			&startDate, &endDate, &we.Description, &we.CreatedAt, &we.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if startDate != nil {
			we.StartDate = startDate.Format("2006-01-02")
		}
		if endDate != nil {
			end := endDate.Format("2006-01-02")
			we.EndDate = &end
		}
		workExperiences = append(workExperiences, we)
	}
	if workExperiences == nil {
		workExperiences = []domain.WorkExperience{}
	}
	response.WorkExperiences = workExperiences

	// 6. Get skills (via candidate_skills join)
	skillsQuery := `
		SELECT s.id, s.name, s.category
		FROM skills s
		INNER JOIN candidate_skills cs ON s.id = cs.skill_id
		WHERE cs.user_id = $1
		ORDER BY s.category, s.name
	`
	skillRows, err := r.db.Query(ctx, skillsQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}
	defer skillRows.Close()

	var skills []domain.Skill
	for skillRows.Next() {
		var s domain.Skill
		if err := skillRows.Scan(&s.ID, &s.Name, &s.Category); err != nil {
			continue
		}
		skills = append(skills, s)
	}
	if skills == nil {
		skills = []domain.Skill{}
	}
	response.Skills = skills

	// 7. Get candidate_certificates
	certsQuery := `
		SELECT id, user_id, certificate_type, certificate_name, score_total,
		       score_details, issued_date, expires_date, document_file_path,
		       created_at, updated_at
		FROM candidate_certificates
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	certRows, err := r.db.Query(ctx, certsQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificates: %w", err)
	}
	defer certRows.Close()

	var certificates []domain.CandidateCertificate
	for certRows.Next() {
		var c domain.CandidateCertificate
		var issuedDate, expiresDate *time.Time
		if err := certRows.Scan(
			&c.ID, &c.UserID, &c.CertificateType, &c.CertificateName, &c.ScoreTotal,
			&c.ScoreDetails, &issuedDate, &expiresDate, &c.DocumentFilePath,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			continue
		}
		if issuedDate != nil {
			issued := issuedDate.Format("2006-01-02")
			c.IssuedDate = &issued
		}
		if expiresDate != nil {
			expires := expiresDate.Format("2006-01-02")
			c.ExpiresDate = &expires
		}
		certificates = append(certificates, c)
	}
	if certificates == nil {
		certificates = []domain.CandidateCertificate{}
	}
	response.Certificates = certificates

	// 8. Get onboarding data
	onboardingData := &domain.OnboardingData{}
	hasOnboardingData := false

	// 8a. Get interests
	interestsQuery := `SELECT interest_key FROM candidate_interests WHERE user_id = $1`
	interestRows, err := r.db.Query(ctx, interestsQuery, userID)
	if err == nil {
		defer interestRows.Close()
		for interestRows.Next() {
			var key string
			if err := interestRows.Scan(&key); err == nil {
				onboardingData.Interests = append(onboardingData.Interests, domain.InterestKey(key))
				hasOnboardingData = true
			}
		}
	}

	// 8b. Get LPK selection from account_verifications
	lpkQuery := `
		SELECT lpk_id, lpk_other_name, lpk_none, onboarding_completed_at
		FROM account_verifications
		WHERE id = $1
	`
	var lpkID *int64
	var lpkOtherName *string
	var lpkNone bool
	var onboardingCompletedAt *time.Time
	err = r.db.QueryRow(ctx, lpkQuery, id).Scan(&lpkID, &lpkOtherName, &lpkNone, &onboardingCompletedAt)
	if err == nil {
		onboardingData.LPKSelection = domain.LPKSelection{
			LPKID:     lpkID,
			OtherName: lpkOtherName,
			None:      lpkNone,
		}
		onboardingData.CompletedAt = onboardingCompletedAt
		hasOnboardingData = true

		// Resolve LPK name if ID is present
		if lpkID != nil {
			var lpkName string
			if err := r.db.QueryRow(ctx, "SELECT name FROM lpk_list WHERE id = $1", *lpkID).Scan(&lpkName); err == nil {
				onboardingData.LPKName = &lpkName
			}
		}
	}

	// 8c. Get company preferences
	prefsQuery := `SELECT preference_key FROM candidate_company_preferences WHERE user_id = $1`
	prefRows, err := r.db.Query(ctx, prefsQuery, userID)
	if err == nil {
		defer prefRows.Close()
		for prefRows.Next() {
			var key string
			if err := prefRows.Scan(&key); err == nil {
				onboardingData.CompanyPreferences = append(onboardingData.CompanyPreferences, domain.CompanyPreferenceKey(key))
				hasOnboardingData = true
			}
		}
	}

	if hasOnboardingData {
		response.OnboardingData = onboardingData
	}

	return response, nil
}
