package postgres

import (
	"context"
	"fmt"
	"go-recruitment-backend/internal/domain"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type atsRepo struct {
	db *pgxpool.Pool
}

// NewATSRepository creates a new ATS repository instance
func NewATSRepository(db *pgxpool.Pool) domain.ATSRepository {
	return &atsRepo{db: db}
}

// SearchCandidates fetches candidates matching the filter criteria
func (r *atsRepo) SearchCandidates(ctx context.Context, filter domain.ATSFilter) ([]domain.ATSCandidate, int64, error) {
	// Build dynamic WHERE clause
	conditions := []string{"av.status IN ('VERIFIED', 'SUBMITTED')"}
	args := []interface{}{}
	argIndex := 1

	// Japanese Proficiency Group
	if len(filter.JapaneseLevels) > 0 {
		placeholders := make([]string, len(filter.JapaneseLevels))
		for i, level := range filter.JapaneseLevels {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, level)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("av.japanese_level IN (%s)", strings.Join(placeholders, ",")))
	}

	if filter.JapanExperienceMin != nil {
		conditions = append(conditions, fmt.Sprintf("av.japan_experience_duration >= $%d", argIndex))
		args = append(args, *filter.JapanExperienceMin)
		argIndex++
	}

	if filter.JapanExperienceMax != nil {
		conditions = append(conditions, fmt.Sprintf("av.japan_experience_duration <= $%d", argIndex))
		args = append(args, *filter.JapanExperienceMax)
		argIndex++
	}

	if filter.HasLPKTraining != nil {
		if *filter.HasLPKTraining {
			conditions = append(conditions, "av.lpk_id IS NOT NULL")
		} else {
			conditions = append(conditions, "(av.lpk_id IS NULL AND av.lpk_none = TRUE)")
		}
	}

	// Competency & Language Group
	if len(filter.EnglishCertTypes) > 0 {
		placeholders := make([]string, len(filter.EnglishCertTypes))
		for i, certType := range filter.EnglishCertTypes {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, certType)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("cc.certificate_type IN (%s)", strings.Join(placeholders, ",")))
	}

	if filter.EnglishMinScore != nil {
		conditions = append(conditions, fmt.Sprintf("cc.score_total >= $%d", argIndex))
		args = append(args, *filter.EnglishMinScore)
		argIndex++
	}

	if len(filter.TechnicalSkillIDs) > 0 || len(filter.ComputerSkillIDs) > 0 {
		allSkillIDs := append(filter.TechnicalSkillIDs, filter.ComputerSkillIDs...)
		placeholders := make([]string, len(allSkillIDs))
		for i, id := range allSkillIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, id)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("cs.skill_id IN (%s)", strings.Join(placeholders, ",")))
	}

	// Logistics & Availability Group - Age (convert to birth_date)
	if filter.AgeMin != nil {
		// Max birth date = today - min age years
		maxBirthDate := time.Now().AddDate(-*filter.AgeMin, 0, 0)
		conditions = append(conditions, fmt.Sprintf("av.birth_date <= $%d", argIndex))
		args = append(args, maxBirthDate)
		argIndex++
	}

	if filter.AgeMax != nil {
		// Min birth date = today - (max age + 1) years
		minBirthDate := time.Now().AddDate(-*filter.AgeMax-1, 0, 0)
		conditions = append(conditions, fmt.Sprintf("av.birth_date > $%d", argIndex))
		args = append(args, minBirthDate)
		argIndex++
	}

	if len(filter.Genders) > 0 {
		placeholders := make([]string, len(filter.Genders))
		for i, gender := range filter.Genders {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, gender)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("av.gender IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(filter.DomicileCities) > 0 {
		placeholders := make([]string, len(filter.DomicileCities))
		for i, city := range filter.DomicileCities {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, city)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("av.domicile_city IN (%s)", strings.Join(placeholders, ",")))
	}

	if filter.ExpectedSalaryMin != nil {
		conditions = append(conditions, fmt.Sprintf("av.expected_salary >= $%d", argIndex))
		args = append(args, *filter.ExpectedSalaryMin)
		argIndex++
	}

	if filter.ExpectedSalaryMax != nil {
		conditions = append(conditions, fmt.Sprintf("av.expected_salary <= $%d", argIndex))
		args = append(args, *filter.ExpectedSalaryMax)
		argIndex++
	}

	if filter.AvailableStartBefore != nil {
		conditions = append(conditions, fmt.Sprintf("av.available_start_date <= $%d", argIndex))
		args = append(args, *filter.AvailableStartBefore)
		argIndex++
	}

	// Education & Experience Group
	if len(filter.EducationLevels) > 0 {
		placeholders := make([]string, len(filter.EducationLevels))
		for i, level := range filter.EducationLevels {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, level)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("cp.highest_education IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(filter.MajorFields) > 0 {
		placeholders := make([]string, len(filter.MajorFields))
		for i, major := range filter.MajorFields {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, major)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("cp.major_field IN (%s)", strings.Join(placeholders, ",")))
	}

	if filter.TotalExperienceMin != nil {
		conditions = append(conditions, fmt.Sprintf("COALESCE(cp.total_experience_months, 0) >= $%d", argIndex))
		args = append(args, *filter.TotalExperienceMin)
		argIndex++
	}

	if filter.TotalExperienceMax != nil {
		conditions = append(conditions, fmt.Sprintf("COALESCE(cp.total_experience_months, 0) <= $%d", argIndex))
		args = append(args, *filter.TotalExperienceMax)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Sorting
	sortColumn := "av.verified_at"
	sortOrder := "DESC NULLS LAST"
	switch filter.SortBy {
	case "japanese_level":
		sortColumn = "av.japanese_level"
	case "age":
		sortColumn = "av.birth_date"
		sortOrder = "ASC NULLS LAST" // Older birth date = younger age
		if filter.SortOrder == "desc" {
			sortOrder = "DESC NULLS LAST"
		}
	case "expected_salary":
		sortColumn = "av.expected_salary"
	}
	if filter.SortOrder == "asc" && filter.SortBy != "age" {
		sortOrder = "ASC NULLS LAST"
	}

	orderClause := fmt.Sprintf("%s %s", sortColumn, sortOrder)

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT av.user_id)
		FROM account_verifications av
		LEFT JOIN candidate_profiles cp ON av.user_id = cp.user_id
		LEFT JOIN candidate_certificates cc ON av.user_id = cc.user_id
		LEFT JOIN candidate_skills cs ON av.user_id = cs.user_id
		WHERE %s
	`, whereClause)

	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	// Pagination
	pageSize := filter.PageSize
	if pageSize == 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	// Main query
	query := fmt.Sprintf(`
		SELECT DISTINCT ON (av.user_id)
			av.user_id,
			av.id AS verification_id,
			COALESCE(CONCAT(av.first_name, ' ', av.last_name), 'Unknown') AS full_name,
			av.profile_picture_url,
			EXTRACT(YEAR FROM AGE(av.birth_date))::INT AS age,
			av.gender,
			av.domicile_city,
			av.marital_status,
			av.japanese_level,
			av.japan_experience_duration,
			COALESCE(lpk.name, av.lpk_other_name) AS lpk_training_name,
			cc.certificate_type AS english_cert_type,
			cc.score_total AS english_score,
			cp.highest_education,
			cp.major_field,
			COALESCE(cp.total_experience_months, 0) AS total_experience_months,
			av.expected_salary,
			av.available_start_date,
			av.status AS verification_status,
			av.verified_at,
			av.submitted_at,
			(
				SELECT job_title FROM work_experiences 
				WHERE user_id = av.user_id 
				ORDER BY COALESCE(end_date, CURRENT_DATE) DESC, start_date DESC 
				LIMIT 1
			) AS last_position,
			(
				SELECT ARRAY_AGG(s.name) FROM candidate_skills cs2
				JOIN skills s ON cs2.skill_id = s.id
				WHERE cs2.user_id = av.user_id
			) AS skills
		FROM account_verifications av
		LEFT JOIN candidate_profiles cp ON av.user_id = cp.user_id
		LEFT JOIN lpk_list lpk ON av.lpk_id = lpk.id
		LEFT JOIN candidate_certificates cc ON av.user_id = cc.user_id AND cc.id = (
			SELECT id FROM candidate_certificates 
			WHERE user_id = av.user_id 
			ORDER BY score_total DESC NULLS LAST 
			LIMIT 1
		)
		LEFT JOIN candidate_skills cs ON av.user_id = cs.user_id
		WHERE %s
		ORDER BY av.user_id, %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var candidates []domain.ATSCandidate
	for rows.Next() {
		var c domain.ATSCandidate
		var skills []string

		err := rows.Scan(
			&c.UserID,
			&c.VerificationID,
			&c.FullName,
			&c.ProfilePictureURL,
			&c.Age,
			&c.Gender,
			&c.DomicileCity,
			&c.MaritalStatus,
			&c.JapaneseLevel,
			&c.JapanExperienceMonths,
			&c.LPKTrainingName,
			&c.EnglishCertType,
			&c.EnglishScore,
			&c.HighestEducation,
			&c.MajorField,
			&c.TotalExperienceMonths,
			&c.ExpectedSalary,
			&c.AvailableStartDate,
			&c.VerificationStatus,
			&c.VerifiedAt,
			&c.SubmittedAt,
			&c.LastPosition,
			&skills,
		)
		if err != nil {
			continue
		}
		c.Skills = skills
		candidates = append(candidates, c)
	}

	if candidates == nil {
		candidates = []domain.ATSCandidate{}
	}

	return candidates, total, nil
}

// GetFilterOptions returns all available filter options
func (r *atsRepo) GetFilterOptions(ctx context.Context) (*domain.ATSFilterOptions, error) {
	options := &domain.ATSFilterOptions{
		JapaneseLevels:   []string{domain.JLPTN1, domain.JLPTN2, domain.JLPTN3, domain.JLPTN4, domain.JLPTN5, domain.JLPTNonCertified},
		Genders:          []string{domain.GenderMale, domain.GenderFemale},
		EducationLevels:  []string{domain.EducationHighSchool, domain.EducationDiploma, domain.EducationBachelor, domain.EducationMaster},
		EnglishCertTypes: []string{domain.EnglishCertTOEFL, domain.EnglishCertIELTS, domain.EnglishCertTOEIC},
	}

	// Get domicile cities
	cities, err := r.GetDistinctDomicileCities(ctx)
	if err == nil {
		options.DomicileCities = cities
	}

	// Get major fields
	majors, err := r.GetDistinctMajorFields(ctx)
	if err == nil {
		options.MajorFields = majors
	}

	// Get technical skills
	rows, err := r.db.Query(ctx, `SELECT id, name, category FROM skills WHERE category = 'TECHNICAL' ORDER BY name`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s domain.Skill
			if err := rows.Scan(&s.ID, &s.Name, &s.Category); err == nil {
				options.TechnicalSkills = append(options.TechnicalSkills, s)
			}
		}
	}

	// Get computer skills
	rows, err = r.db.Query(ctx, `SELECT id, name, category FROM skills WHERE category = 'COMPUTER' ORDER BY name`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s domain.Skill
			if err := rows.Scan(&s.ID, &s.Name, &s.Category); err == nil {
				options.ComputerSkills = append(options.ComputerSkills, s)
			}
		}
	}

	return options, nil
}

// GetDistinctDomicileCities returns unique domicile cities from verified candidates
func (r *atsRepo) GetDistinctDomicileCities(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT domicile_city 
		FROM account_verifications 
		WHERE domicile_city IS NOT NULL AND domicile_city != ''
		ORDER BY domicile_city
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cities []string
	for rows.Next() {
		var city string
		if err := rows.Scan(&city); err == nil {
			cities = append(cities, city)
		}
	}

	return cities, nil
}

// GetDistinctMajorFields returns unique major fields from candidates
func (r *atsRepo) GetDistinctMajorFields(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT major_field 
		FROM candidate_profiles 
		WHERE major_field IS NOT NULL AND major_field != ''
		ORDER BY major_field
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var majors []string
	for rows.Next() {
		var major string
		if err := rows.Scan(&major); err == nil {
			majors = append(majors, major)
		}
	}

	return majors, nil
}
