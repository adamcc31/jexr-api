package domain

import (
	"context"
	"time"
)

// ============================================================================
// LPK (Lembaga Pelatihan Kerja) - Training Center
// ============================================================================

type LPK struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// ============================================================================
// Candidate Interest (Step 1)
// ============================================================================

// InterestKey represents valid interest options
type InterestKey string

const (
	InterestTeacher    InterestKey = "teacher"
	InterestTranslator InterestKey = "translator"
	InterestAdmin      InterestKey = "admin"
	InterestNone       InterestKey = "none"
)

// ValidInterestKeys returns all valid interest keys
func ValidInterestKeys() []InterestKey {
	return []InterestKey{InterestTeacher, InterestTranslator, InterestAdmin, InterestNone}
}

// IsValid checks if the interest key is valid
func (k InterestKey) IsValid() bool {
	for _, valid := range ValidInterestKeys() {
		if k == valid {
			return true
		}
	}
	return false
}

type CandidateInterest struct {
	ID          int64       `json:"id"`
	UserID      string      `json:"user_id"`
	InterestKey InterestKey `json:"interest_key"`
	CreatedAt   time.Time   `json:"created_at"`
}

// ============================================================================
// Candidate Company Preference (Step 3)
// ============================================================================

// CompanyPreferenceKey represents valid company type preferences
type CompanyPreferenceKey string

const (
	CompanyPMA          CompanyPreferenceKey = "pma"           // 100% Japanese (PMA)
	CompanyJointVenture CompanyPreferenceKey = "joint_venture" // Joint Venture
	CompanyLocal        CompanyPreferenceKey = "local"         // 100% Indonesian (Local)
)

// ValidCompanyPreferenceKeys returns all valid preference keys
func ValidCompanyPreferenceKeys() []CompanyPreferenceKey {
	return []CompanyPreferenceKey{CompanyPMA, CompanyJointVenture, CompanyLocal}
}

// IsValid checks if the preference key is valid
func (k CompanyPreferenceKey) IsValid() bool {
	for _, valid := range ValidCompanyPreferenceKeys() {
		if k == valid {
			return true
		}
	}
	return false
}

type CandidateCompanyPreference struct {
	ID            int64                `json:"id"`
	UserID        string               `json:"user_id"`
	PreferenceKey CompanyPreferenceKey `json:"preference_key"`
	CreatedAt     time.Time            `json:"created_at"`
}

// ============================================================================
// Onboarding Data Transfer Objects
// ============================================================================

// OnboardingStatus represents the onboarding completion status
type OnboardingStatus struct {
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// LPKSelection represents the LPK choice from Step 2
type LPKSelection struct {
	LPKID     *int64  `json:"lpk_id,omitempty"`     // Selected from list
	OtherName *string `json:"other_name,omitempty"` // Manual entry ("Lainnya")
	None      bool    `json:"none"`                 // "Saya tidak belajar di LPK"
}

// OnboardingSubmitRequest is the request payload for completing onboarding
type OnboardingSubmitRequest struct {
	// Step 1: Interests
	Interests []InterestKey `json:"interests" validate:"required,min=1"`

	// Step 2: LPK Selection
	LPKSelection LPKSelection `json:"lpk_selection" validate:"required"`

	// Step 3: Company Preferences
	CompanyPreferences []CompanyPreferenceKey `json:"company_preferences" validate:"required,min=1"`
}

// ============================================================================
// Repository Interface
// ============================================================================

type OnboardingRepository interface {
	// LPK Search
	SearchLPK(ctx context.Context, query string, limit int) ([]LPK, error)
	GetLPKByID(ctx context.Context, id int64) (*LPK, error)

	// Onboarding Status
	GetOnboardingStatus(ctx context.Context, userID string) (*OnboardingStatus, error)

	// Save Onboarding Data (atomic transaction)
	SaveOnboardingData(ctx context.Context, userID string, req *OnboardingSubmitRequest) error
}

// ============================================================================
// Usecase Interface
// ============================================================================

type OnboardingUsecase interface {
	// LPK Search with debouncing handled client-side
	SearchLPK(ctx context.Context, query string) ([]LPK, error)

	// Check if user has completed onboarding
	GetOnboardingStatus(ctx context.Context, userID string) (*OnboardingStatus, error)

	// Validate and save onboarding data
	CompleteOnboarding(ctx context.Context, userID string, req *OnboardingSubmitRequest) error
}
