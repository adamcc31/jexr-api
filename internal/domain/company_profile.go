package domain

import (
	"context"
	"time"
)

// CompanyProfile represents an employer's company profile
type CompanyProfile struct {
	ID                 int64     `json:"id"`
	UserID             string    `json:"user_id"`
	CompanyName        string    `json:"company_name"`
	LogoURL            *string   `json:"logo_url"`
	Location           *string   `json:"location"`
	CompanyStory       *string   `json:"company_story"`
	Founded            *string   `json:"founded"`
	Founder            *string   `json:"founder"`
	Headquarters       *string   `json:"headquarters"`
	EmployeeCount      *string   `json:"employee_count"`
	Website            *string   `json:"website"`
	Industry           *string   `json:"industry"`
	Description        *string   `json:"description"`
	HideCompanyDetails bool      `json:"hide_company_details"`
	GalleryImage1      *string   `json:"gallery_image_1"`
	GalleryImage2      *string   `json:"gallery_image_2"`
	GalleryImage3      *string   `json:"gallery_image_3"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// PublicCompanyProfile is the public-facing version with conditional fields
type PublicCompanyProfile struct {
	ID            int64   `json:"id"`
	CompanyName   string  `json:"company_name"`
	LogoURL       *string `json:"logo_url"`
	Location      *string `json:"location"`
	CompanyStory  *string `json:"company_story"`
	GalleryImage1 *string `json:"gallery_image_1"`
	GalleryImage2 *string `json:"gallery_image_2"`
	GalleryImage3 *string `json:"gallery_image_3"`
	// Conditional fields - only shown if viewer is verified or hide_company_details is false
	Founded       *string `json:"founded,omitempty"`
	Founder       *string `json:"founder,omitempty"`
	Headquarters  *string `json:"headquarters,omitempty"`
	EmployeeCount *string `json:"employee_count,omitempty"`
	Website       *string `json:"website,omitempty"`
	// Flag to indicate if details are hidden
	DetailsHidden bool `json:"details_hidden"`
}

// ViewerInfo contains information about the viewer for visibility logic
type ViewerInfo struct {
	IsAuthenticated    bool
	Role               string
	VerificationStatus string
}

// ShouldShowCompanyDetails determines if company details should be visible
func (v *ViewerInfo) ShouldShowCompanyDetails(hideCompanyDetails bool) bool {
	// If company opts to show details, always show
	if !hideCompanyDetails {
		return true
	}
	// If hide is enabled, only show to verified candidates
	if !v.IsAuthenticated {
		return false
	}
	if v.Role == "admin" || v.Role == "employer" {
		return true
	}
	// For candidates, must be verified
	return v.VerificationStatus == VerificationStatusVerified
}

// CompanyProfileRepository defines storage operations
type CompanyProfileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*CompanyProfile, error)
	GetByID(ctx context.Context, id int64) (*CompanyProfile, error)
	Upsert(ctx context.Context, profile *CompanyProfile) error
}

// CompanyProfileUsecase defines business logic operations
type CompanyProfileUsecase interface {
	// Employer operations (protected)
	GetEmployerProfile(ctx context.Context, userID string) (*CompanyProfile, error)
	UpdateEmployerProfile(ctx context.Context, userID string, profile *CompanyProfile) error
	// Public operations
	GetPublicProfile(ctx context.Context, id int64, viewer *ViewerInfo) (*PublicCompanyProfile, error)
}
