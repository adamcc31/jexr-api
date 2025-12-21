package domain

import (
	"context"
	"time"
)

// VerificationStatus constants
const (
	VerificationStatusPending   = "PENDING"
	VerificationStatusSubmitted = "SUBMITTED"
	VerificationStatusVerified  = "VERIFIED"
	VerificationStatusRejected  = "REJECTED"
)

// MaritalStatus constants
const (
	MaritalStatusSingle   = "SINGLE"
	MaritalStatusMarried  = "MARRIED"
	MaritalStatusDivorced = "DIVORCED"
)

// ValidMaritalStatuses for validation
var ValidMaritalStatuses = []string{MaritalStatusSingle, MaritalStatusMarried, MaritalStatusDivorced}

// JapaneseSpeakingLevel constants
const (
	JapaneseSpeakingNative  = "NATIVE"
	JapaneseSpeakingFluent  = "FLUENT"
	JapaneseSpeakingBasic   = "BASIC"
	JapaneseSpeakingPassive = "PASSIVE"
)

// ValidJapaneseSpeakingLevels for validation
var ValidJapaneseSpeakingLevels = []string{JapaneseSpeakingNative, JapaneseSpeakingFluent, JapaneseSpeakingBasic, JapaneseSpeakingPassive}

// AccountVerification represents a verification record
type AccountVerification struct {
	ID          int64      `json:"id"`
	UserID      string     `json:"user_id"`
	UserEmail   string     `json:"user_email,omitempty"` // Populated via join often
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	SubmittedAt time.Time  `json:"submitted_at"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	VerifiedBy  *string    `json:"verified_by,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Personal Profile Fields
	FirstName               *string `json:"first_name"`
	LastName                *string `json:"last_name"`
	ProfilePictureURL       *string `json:"profile_picture_url"`
	Occupation              *string `json:"occupation"`
	Phone                   *string `json:"phone"`
	WebsiteURL              *string `json:"website_url"`
	Intro                   *string `json:"intro"`
	JapanExperienceDuration *int    `json:"japan_experience_duration"` // In months
	JapaneseCertificateURL  *string `json:"japanese_certificate_url"`
	CvURL                   *string `json:"cv_url"`         // CV/Resume document URL
	JapaneseLevel           *string `json:"japanese_level"` // N5, N4, N3, N2, N1

	// HR Candidate Data: Identity & Demographics
	BirthDate     *time.Time `json:"birth_date,omitempty"`
	DomicileCity  *string    `json:"domicile_city,omitempty"`
	MaritalStatus *string    `json:"marital_status,omitempty"` // SINGLE, MARRIED, DIVORCED
	ChildrenCount *int       `json:"children_count,omitempty"`

	// HR Candidate Data: Core Competencies
	MainJobFields         []string `json:"main_job_fields,omitempty"`
	GoldenSkill           *string  `json:"golden_skill,omitempty"`
	JapaneseSpeakingLevel *string  `json:"japanese_speaking_level,omitempty"` // NATIVE, FLUENT, BASIC, PASSIVE

	// HR Candidate Data: Expectations & Availability
	ExpectedSalary      *int64     `json:"expected_salary,omitempty"` // Netto/THP in raw number
	JapanReturnDate     *time.Time `json:"japan_return_date,omitempty"`
	AvailableStartDate  *time.Time `json:"available_start_date,omitempty"`
	PreferredLocations  []string   `json:"preferred_locations,omitempty"`
	PreferredIndustries []string   `json:"preferred_industries,omitempty"`

	// HR Candidate Data: Supporting Documents
	SupportingCertificatesURL []string `json:"supporting_certificates_url,omitempty"`

	// Additional data for display
	UserProfile *UserProfileSummary `json:"user_profile,omitempty"`
}

// JapanWorkExperience represents a work entry in Japan
type JapanWorkExperience struct {
	ID                    int64      `json:"id"`
	AccountVerificationID int64      `json:"account_verification_id"`
	CompanyName           string     `json:"company_name"`
	JobTitle              string     `json:"job_title"`
	StartDate             time.Time  `json:"start_date"`
	EndDate               *time.Time `json:"end_date"` // Nullable if currently working
	Description           *string    `json:"description"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// VerificationResponse aggregates profile and experiences for API response
type VerificationResponse struct {
	Verification *AccountVerification  `json:"verification"`
	Experiences  []JapanWorkExperience `json:"experiences"`
}

// UserProfileSummary holds minimal profile info for the admin table
type UserProfileSummary struct {
	Name        string `json:"name"`
	CompanyName string `json:"company_name,omitempty"`
}

// VerificationFilter defines filtering options
type VerificationFilter struct {
	Role   string `json:"role,omitempty"`
	Status string `json:"status,omitempty"`
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
}

// VerificationRepository interface
type VerificationRepository interface {
	GetByUserID(ctx context.Context, userID string) (*AccountVerification, error)
	GetByID(ctx context.Context, id int64) (*AccountVerification, error)
	List(ctx context.Context, filter VerificationFilter) ([]AccountVerification, int64, error)
	UpdateStatus(ctx context.Context, id int64, status string, verifiedBy string, notes string) error

	// New methods for candidate profile
	Create(ctx context.Context, verification *AccountVerification) (int64, error)
	UpdateProfile(ctx context.Context, verification *AccountVerification, experiences []JapanWorkExperience) error
	GetWorkExperiences(ctx context.Context, verificationID int64) ([]JapanWorkExperience, error)
}

// VerificationUsecase interface
type VerificationUsecase interface {
	GetPendingVerifications(ctx context.Context, page, limit int) ([]AccountVerification, int64, error)
	ListVerifications(ctx context.Context, filter VerificationFilter) ([]AccountVerification, int64, error)
	VerifyUser(ctx context.Context, adminID string, verificationID int64, action string, notes string) error
	GetVerificationStatus(ctx context.Context, userID string) (*VerificationResponse, error)
	GetVerificationByID(ctx context.Context, id int64) (*VerificationResponse, error) // For admin detail view
	UpdateCandidateProfile(ctx context.Context, userID string, verification *AccountVerification, experiences []JapanWorkExperience) error
}
