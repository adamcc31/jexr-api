package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

// CandidateProfile represents the core profile data in 'candidate_profiles' table
type CandidateProfile struct {
	ID                       int64          `json:"id"`
	UserID                   string         `json:"user_id" validate:"required"`
	Title                    string         `json:"title" validate:"required,min=3,max=100"` // Keeping for backward compat or strictly display title?
	Bio                      string         `json:"bio" validate:"max=500"`
	HighestEducation         string         `json:"highest_education"`
	MajorField               string         `json:"major_field"`
	DesiredJobPosition       string         `json:"desired_job_position"`       // Code/ID if master exists
	DesiredJobPositionOther  string         `json:"desired_job_position_other"` // Free text
	PreferredWorkEnvironment string         `json:"preferred_work_environment"`
	CareerGoals3y            string         `json:"career_goals_3y"`
	MainConcernsReturning    pq.StringArray `json:"main_concerns_returning" swaggertype:"array,string"`
	SpecialMessage           string         `json:"special_message"`
	SkillsOther              string         `json:"skills_other"` // Newline separated or JSON
	ResumeURL                string         `json:"resume_url" validate:"omitempty,url"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`

	// Legacy or Computed
	Skills []string `json:"skills" swaggertype:"array,string"` // From old schema, maybe computed from candidate_skills now?
}

// CandidateDetail represents the narrative data in 'candidate_details' table
type CandidateDetail struct {
	UserID                string    `json:"user_id"`
	SoftSkillsDescription string    `json:"soft_skills_description"`
	AppliedWorkValues     string    `json:"applied_work_values"`
	MajorAchievements     string    `json:"major_achievements"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// WorkExperience represents an entry in 'work_experiences' table
type WorkExperience struct {
	ID             int64     `json:"id"`
	UserID         string    `json:"user_id"`
	CountryCode    string    `json:"country_code" validate:"required,len=2"` // ISO-2
	ExperienceType string    `json:"experience_type" validate:"required,oneof=LOCAL OVERSEAS"`
	CompanyName    string    `json:"company_name" validate:"required"`
	JobTitle       string    `json:"job_title" validate:"required"`
	StartDate      string    `json:"start_date" validate:"required"` // Format: YYYY-MM-DD
	EndDate        *string   `json:"end_date"`                       // Nullable, Format: YYYY-MM-DD
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at" swaggerignore:"true"`
	UpdatedAt      time.Time `json:"updated_at" swaggerignore:"true"`
}

// Skill represents a master skill
type Skill struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

// CandidateCertificate represents a language/skill certificate
// Note: JLPT is handled separately in AccountVerification
type CandidateCertificate struct {
	ID               int64           `json:"id"`
	UserID           string          `json:"user_id"`
	CertificateType  string          `json:"certificate_type" validate:"required,oneof=TOEFL IELTS TOEIC OTHER"`
	CertificateName  string          `json:"certificate_name,omitempty"` // For 'OTHER' type
	ScoreTotal       *float64        `json:"score_total,omitempty"`
	ScoreDetails     json.RawMessage `json:"score_details,omitempty" swaggertype:"object"`
	IssuedDate       *string         `json:"issued_date,omitempty"`
	ExpiresDate      *string         `json:"expires_date,omitempty"`
	DocumentFilePath string          `json:"document_file_path" validate:"required"`
	CreatedAt        time.Time       `json:"created_at" swaggerignore:"true"`
	UpdatedAt        time.Time       `json:"updated_at" swaggerignore:"true"`
}

// CandidateWithFullDetails is the aggregate structure for API Request/Response
type CandidateWithFullDetails struct {
	Profile         CandidateProfile       `json:"profile"`
	Details         CandidateDetail        `json:"details"`
	WorkExperiences []WorkExperience       `json:"work_experiences"`
	Certificates    []CandidateCertificate `json:"certificates"`
	SkillIDs        []int                  `json:"skill_ids"` // For updates
	Skills          []Skill                `json:"skills"`    // For responses
}

type CandidateRepository interface {
	GetByUserID(ctx context.Context, userID string) (*CandidateProfile, error)
	Create(ctx context.Context, profile *CandidateProfile) error
	Update(ctx context.Context, profile *CandidateProfile) error

	// New Transactional Methods
	GetFullProfile(ctx context.Context, userID string) (*CandidateWithFullDetails, error)
	UpsertFullProfile(ctx context.Context, fullProfile *CandidateWithFullDetails) error

	// Master Data Helpers
	GetAllSkills(ctx context.Context) ([]Skill, error)
}

type CandidateUsecase interface {
	GetProfile(ctx context.Context, userID string) (*CandidateProfile, error)
	UpdateProfile(ctx context.Context, profile *CandidateProfile) error

	// New Usecase Methods
	GetFullProfile(ctx context.Context, userID string) (*CandidateWithFullDetails, error)
	UpdateFullProfile(ctx context.Context, userID string, req *CandidateWithFullDetails) error
	GetMasterSkills(ctx context.Context) ([]Skill, error)
}
