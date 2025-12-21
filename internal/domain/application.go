package domain

import (
	"context"
	"time"
)

// Application status constants
const (
	ApplicationStatusApplied  = "applied"
	ApplicationStatusReviewed = "reviewed"
	ApplicationStatusAccepted = "accepted"
	ApplicationStatusRejected = "rejected"
)

// Application represents a job application from a candidate
type Application struct {
	ID                    int64     `json:"id"`
	JobID                 int64     `json:"job_id"`
	CandidateUserID       string    `json:"candidate_user_id"`
	AccountVerificationID *int64    `json:"account_verification_id,omitempty"`
	CvURL                 string    `json:"cv_url"` // Required
	CoverLetter           *string   `json:"cover_letter,omitempty"`
	Status                string    `json:"status"` // applied → reviewed → accepted / rejected
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`

	// Joined data for list responses
	CandidateName      *string `json:"candidate_name,omitempty"`
	CandidatePhoto     *string `json:"candidate_photo,omitempty"`
	VerificationStatus *string `json:"verification_status,omitempty"`
	JobTitle           *string `json:"job_title,omitempty"`
}

// ApplicationDetailResponse contains full application details including candidate profile
type ApplicationDetailResponse struct {
	Application  *Application          `json:"application"`
	Verification *AccountVerification  `json:"verification,omitempty"`
	Experiences  []JapanWorkExperience `json:"experiences,omitempty"`
}

// ApplicationRepository defines data access methods for applications
type ApplicationRepository interface {
	Create(ctx context.Context, app *Application) error
	GetByID(ctx context.Context, id int64) (*Application, error)
	GetByJobID(ctx context.Context, jobID int64) ([]Application, error)
	GetByUserID(ctx context.Context, userID string) ([]Application, error)
	CheckExists(ctx context.Context, jobID int64, userID string) (bool, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
}

// ApplicationUsecase defines business logic for applications
type ApplicationUsecase interface {
	// Candidate operations
	ApplyToJob(ctx context.Context, userID string, jobID int64, cvURL, coverLetter string) (*Application, error)
	GetMyApplications(ctx context.Context, userID string) ([]Application, error)

	// Employer operations
	ListByJobID(ctx context.Context, userID string, jobID int64) ([]Application, error)
	GetApplicationDetail(ctx context.Context, userID string, applicationID int64) (*ApplicationDetailResponse, error)
	UpdateApplicationStatus(ctx context.Context, userID string, applicationID int64, status string) error
}
