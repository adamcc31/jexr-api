package domain

import (
	"context"
	"time"
)

// ============================================================================
// ATS Filter Constants
// ============================================================================

// JLPTLevel constants for filtering
const (
	JLPTN1           = "N1"
	JLPTN2           = "N2"
	JLPTN3           = "N3"
	JLPTN4           = "N4"
	JLPTN5           = "N5"
	JLPTNonCertified = "NON_CERTIFIED"
)

// Gender constants
const (
	GenderMale   = "MALE"
	GenderFemale = "FEMALE"
)

// EducationLevel constants
const (
	EducationHighSchool = "HIGH_SCHOOL"
	EducationDiploma    = "DIPLOMA"
	EducationBachelor   = "BACHELOR"
	EducationMaster     = "MASTER"
)

// EnglishCertType constants
const (
	EnglishCertTOEFL = "TOEFL"
	EnglishCertIELTS = "IELTS"
	EnglishCertTOEIC = "TOEIC"
)

// ============================================================================
// ATS Filter Request
// ============================================================================

// ATSFilter represents all possible filter parameters for ATS search
type ATSFilter struct {
	// Japanese Proficiency Group
	JapaneseLevels     []string `json:"japanese_levels,omitempty"`      // N1, N2, N3, N4, N5, NON_CERTIFIED
	JapanExperienceMin *int     `json:"japan_experience_min,omitempty"` // Months
	JapanExperienceMax *int     `json:"japan_experience_max,omitempty"` // Months
	HasLPKTraining     *bool    `json:"has_lpk_training,omitempty"`     // true/false/nil(any)

	// Competency & Language Group
	EnglishCertTypes  []string `json:"english_cert_types,omitempty"`  // TOEFL, IELTS, TOEIC
	EnglishMinScore   *float64 `json:"english_min_score,omitempty"`   // Minimum score
	TechnicalSkillIDs []int    `json:"technical_skill_ids,omitempty"` // Skill IDs (category=TECHNICAL)
	ComputerSkillIDs  []int    `json:"computer_skill_ids,omitempty"`  // Skill IDs (category=COMPUTER)

	// Logistics & Availability Group
	AgeMin               *int       `json:"age_min,omitempty"`                // Minimum age (converted to birth_date range)
	AgeMax               *int       `json:"age_max,omitempty"`                // Maximum age (converted to birth_date range)
	Genders              []string   `json:"genders,omitempty"`                // MALE, FEMALE
	DomicileCities       []string   `json:"domicile_cities,omitempty"`        // City names
	ExpectedSalaryMin    *int64     `json:"expected_salary_min,omitempty"`    // Minimum salary (IDR)
	ExpectedSalaryMax    *int64     `json:"expected_salary_max,omitempty"`    // Maximum salary (IDR)
	AvailableStartBefore *time.Time `json:"available_start_before,omitempty"` // Available start date <=

	// Education & Experience Group
	EducationLevels    []string `json:"education_levels,omitempty"`     // HIGH_SCHOOL, DIPLOMA, BACHELOR, MASTER
	MajorFields        []string `json:"major_fields,omitempty"`         // Major field names
	TotalExperienceMin *int     `json:"total_experience_min,omitempty"` // Months
	TotalExperienceMax *int     `json:"total_experience_max,omitempty"` // Months

	// Pagination & Sorting
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	SortBy    string `json:"sort_by,omitempty"`    // verified_at, japanese_level, age, expected_salary
	SortOrder string `json:"sort_order,omitempty"` // asc, desc
}

// ============================================================================
// ATS Candidate Result
// ============================================================================

// ATSCandidate represents a candidate row in the ATS listing
type ATSCandidate struct {
	// Identity
	UserID            string  `json:"user_id"`
	VerificationID    int64   `json:"verification_id"`
	FullName          string  `json:"full_name"`
	ProfilePictureURL *string `json:"profile_picture_url,omitempty"`

	// Demographics
	Age           *int    `json:"age,omitempty"`
	Gender        *string `json:"gender,omitempty"`
	DomicileCity  *string `json:"domicile_city,omitempty"`
	MaritalStatus *string `json:"marital_status,omitempty"`

	// Japanese Proficiency
	JapaneseLevel         *string `json:"japanese_level,omitempty"`
	JapanExperienceMonths *int    `json:"japan_experience_months,omitempty"`
	LPKTrainingName       *string `json:"lpk_training_name,omitempty"` // LPK name if has training, null otherwise

	// Competency
	EnglishCertType *string  `json:"english_cert_type,omitempty"`
	EnglishScore    *float64 `json:"english_score,omitempty"`
	Skills          []string `json:"skills,omitempty"`

	// Education & Experience
	HighestEducation      *string `json:"highest_education,omitempty"`
	MajorField            *string `json:"major_field,omitempty"`
	TotalExperienceMonths *int    `json:"total_experience_months,omitempty"`
	LastPosition          *string `json:"last_position,omitempty"`

	// Availability
	ExpectedSalary     *int64     `json:"expected_salary,omitempty"`
	AvailableStartDate *time.Time `json:"available_start_date,omitempty"`

	// Metadata
	VerificationStatus string     `json:"verification_status"`
	VerifiedAt         *time.Time `json:"verified_at,omitempty"`
	SubmittedAt        time.Time  `json:"submitted_at"`
}

// ============================================================================
// ATS Export Request
// ============================================================================

// ATSExportRequest represents the export configuration
type ATSExportRequest struct {
	Filter  ATSFilter `json:"filter"`
	Columns []string  `json:"columns"` // Selected columns for export
	Format  string    `json:"format"`  // "xlsx" or "csv"
}

// ExportableColumns lists all columns that can be exported
var ExportableColumns = []string{
	"full_name",
	"age",
	"gender",
	"domicile_city",
	"marital_status",
	"japanese_level",
	"japan_experience_months",
	"lpk_training_name",
	"english_cert_type",
	"english_score",
	"skills",
	"highest_education",
	"major_field",
	"total_experience_months",
	"last_position",
	"expected_salary",
	"available_start_date",
	"verification_status",
	"verified_at",
}

// ============================================================================
// ATS Filter Options (Reference Data)
// ============================================================================

// ATSFilterOptions contains all available filter options for the UI
type ATSFilterOptions struct {
	JapaneseLevels   []string `json:"japanese_levels"`
	Genders          []string `json:"genders"`
	EducationLevels  []string `json:"education_levels"`
	EnglishCertTypes []string `json:"english_cert_types"`
	DomicileCities   []string `json:"domicile_cities"`
	MajorFields      []string `json:"major_fields"`
	TechnicalSkills  []Skill  `json:"technical_skills"`
	ComputerSkills   []Skill  `json:"computer_skills"`
}

// ============================================================================
// Repository & Usecase Interfaces
// ============================================================================

// ATSRepository defines data access for ATS feature
type ATSRepository interface {
	// Search candidates with filters
	SearchCandidates(ctx context.Context, filter ATSFilter) ([]ATSCandidate, int64, error)

	// Get filter options (reference data)
	GetFilterOptions(ctx context.Context) (*ATSFilterOptions, error)

	// Get distinct domicile cities from verified candidates
	GetDistinctDomicileCities(ctx context.Context) ([]string, error)

	// Get distinct major fields from candidates
	GetDistinctMajorFields(ctx context.Context) ([]string, error)
}

// ATSUsecase defines business logic for ATS feature
type ATSUsecase interface {
	// Search candidates with validation
	SearchCandidates(ctx context.Context, filter ATSFilter) (*PaginatedResult[ATSCandidate], error)

	// Get filter options for UI dropdowns
	GetFilterOptions(ctx context.Context) (*ATSFilterOptions, error)

	// Export candidates as file bytes
	ExportCandidates(ctx context.Context, req ATSExportRequest) ([]byte, string, error)
}
