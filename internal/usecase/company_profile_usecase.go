package usecase

import (
	"context"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
)

type companyProfileUsecase struct {
	profileRepo      domain.CompanyProfileRepository
	verificationRepo domain.VerificationRepository
}

// NewCompanyProfileUsecase creates a new company profile usecase
func NewCompanyProfileUsecase(
	profileRepo domain.CompanyProfileRepository,
	verificationRepo domain.VerificationRepository,
) domain.CompanyProfileUsecase {
	return &companyProfileUsecase{
		profileRepo:      profileRepo,
		verificationRepo: verificationRepo,
	}
}

// GetEmployerProfile retrieves the employer's own company profile
func (uc *companyProfileUsecase) GetEmployerProfile(ctx context.Context, userID string) (*domain.CompanyProfile, error) {
	profile, err := uc.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			// Return empty profile instead of error for new employers
			return &domain.CompanyProfile{
				UserID:             userID,
				HideCompanyDetails: true, // Default to hidden
			}, nil
		}
		return nil, err
	}
	return profile, nil
}

// UpdateEmployerProfile creates or updates the employer's company profile
func (uc *companyProfileUsecase) UpdateEmployerProfile(ctx context.Context, userID string, profile *domain.CompanyProfile) error {
	// Validate gallery images: must be exactly 0 or exactly 3
	galleryCount := 0
	if profile.GalleryImage1 != nil && *profile.GalleryImage1 != "" {
		galleryCount++
	}
	if profile.GalleryImage2 != nil && *profile.GalleryImage2 != "" {
		galleryCount++
	}
	if profile.GalleryImage3 != nil && *profile.GalleryImage3 != "" {
		galleryCount++
	}

	// Gallery must be either empty (0) or complete (exactly 3)
	if galleryCount > 0 && galleryCount != 3 {
		return apperror.BadRequest("Gallery must have exactly 3 images")
	}

	// Force user ID from context (security: prevent IDOR)
	profile.UserID = userID

	return uc.profileRepo.Upsert(ctx, profile)
}

// GetPublicProfile retrieves a company profile for public viewing with visibility rules
func (uc *companyProfileUsecase) GetPublicProfile(ctx context.Context, id int64, viewer *domain.ViewerInfo) (*domain.PublicCompanyProfile, error) {
	profile, err := uc.profileRepo.GetByID(ctx, id)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, apperror.NotFound("Company profile not found")
		}
		return nil, err
	}

	// Build public profile
	publicProfile := &domain.PublicCompanyProfile{
		ID:            profile.ID,
		CompanyName:   profile.CompanyName,
		LogoURL:       profile.LogoURL,
		Location:      profile.Location,
		CompanyStory:  profile.CompanyStory,
		GalleryImage1: profile.GalleryImage1,
		GalleryImage2: profile.GalleryImage2,
		GalleryImage3: profile.GalleryImage3,
	}

	// Apply visibility rules
	showDetails := viewer != nil && viewer.ShouldShowCompanyDetails(profile.HideCompanyDetails)

	if showDetails {
		publicProfile.Founded = profile.Founded
		publicProfile.Founder = profile.Founder
		publicProfile.Headquarters = profile.Headquarters
		publicProfile.EmployeeCount = profile.EmployeeCount
		publicProfile.Website = profile.Website
		publicProfile.DetailsHidden = false
	} else {
		// Details are hidden - set flag so frontend knows
		publicProfile.DetailsHidden = profile.HideCompanyDetails
	}

	return publicProfile, nil
}
