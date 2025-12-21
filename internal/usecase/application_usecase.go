package usecase

import (
	"context"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
)

type applicationUsecase struct {
	applicationRepo  domain.ApplicationRepository
	jobRepo          domain.JobRepository
	verificationRepo domain.VerificationRepository
}

// NewApplicationUsecase creates a new application usecase
func NewApplicationUsecase(
	appRepo domain.ApplicationRepository,
	jobRepo domain.JobRepository,
	verificationRepo domain.VerificationRepository,
) domain.ApplicationUsecase {
	return &applicationUsecase{
		applicationRepo:  appRepo,
		jobRepo:          jobRepo,
		verificationRepo: verificationRepo,
	}
}

// ApplyToJob allows a verified candidate to apply to an active job
func (uc *applicationUsecase) ApplyToJob(ctx context.Context, userID string, jobID int64, cvURL, coverLetter string) (*domain.Application, error) {
	// 1. Validate CV is provided (required)
	if cvURL == "" {
		return nil, apperror.BadRequest("CV is required to submit an application")
	}

	// 2. Validate job exists and is active
	job, err := uc.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, apperror.NotFound("Job not found")
	}
	if job.CompanyStatus != "active" {
		return nil, apperror.BadRequest("Cannot apply to inactive job")
	}

	// 3. Validate candidate is verified
	verification, err := uc.verificationRepo.GetByUserID(ctx, userID)
	if err != nil || verification == nil {
		return nil, apperror.Forbidden("Complete your profile before applying")
	}
	if verification.Status != domain.VerificationStatusVerified {
		return nil, apperror.Forbidden("Your profile must be verified before you can apply")
	}

	// 4. Check for duplicate application
	exists, err := uc.applicationRepo.CheckExists(ctx, jobID, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if exists {
		return nil, apperror.BadRequest("You have already applied to this job")
	}

	// 5. Create application
	var coverLetterPtr *string
	if coverLetter != "" {
		coverLetterPtr = &coverLetter
	}

	app := &domain.Application{
		JobID:                 jobID,
		CandidateUserID:       userID,
		AccountVerificationID: &verification.ID,
		CvURL:                 cvURL,
		CoverLetter:           coverLetterPtr,
		Status:                domain.ApplicationStatusApplied,
	}

	if err := uc.applicationRepo.Create(ctx, app); err != nil {
		return nil, apperror.Internal(err)
	}

	return app, nil
}

// GetMyApplications returns all applications for the current user
func (uc *applicationUsecase) GetMyApplications(ctx context.Context, userID string) ([]domain.Application, error) {
	return uc.applicationRepo.GetByUserID(ctx, userID)
}

// ListByJobID returns all applications for a job (employer only, validated by ownership)
func (uc *applicationUsecase) ListByJobID(ctx context.Context, userID string, jobID int64) ([]domain.Application, error) {
	// 1. Validate employer owns this job
	if err := uc.validateJobOwnership(ctx, userID, jobID); err != nil {
		return nil, err
	}

	// 2. Fetch applications
	return uc.applicationRepo.GetByJobID(ctx, jobID)
}

// GetApplicationDetail returns full application details with candidate profile
func (uc *applicationUsecase) GetApplicationDetail(ctx context.Context, userID string, applicationID int64) (*domain.ApplicationDetailResponse, error) {
	// 1. Get application
	app, err := uc.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, apperror.NotFound("Application not found")
	}

	// 2. Validate employer owns the job
	if err := uc.validateJobOwnership(ctx, userID, app.JobID); err != nil {
		return nil, err
	}

	// 3. Get verification data and experiences
	var verification *domain.AccountVerification
	var experiences []domain.JapanWorkExperience

	if app.AccountVerificationID != nil {
		verification, _ = uc.verificationRepo.GetByID(ctx, *app.AccountVerificationID)
		if verification != nil {
			experiences, _ = uc.verificationRepo.GetWorkExperiences(ctx, verification.ID)
		}
	}

	return &domain.ApplicationDetailResponse{
		Application:  app,
		Verification: verification,
		Experiences:  experiences,
	}, nil
}

// UpdateApplicationStatus allows employer to update application status
// Status flow: applied → reviewed → accepted / rejected
func (uc *applicationUsecase) UpdateApplicationStatus(ctx context.Context, userID string, applicationID int64, status string) error {
	// 1. Validate status
	validStatuses := map[string]bool{
		domain.ApplicationStatusReviewed: true,
		domain.ApplicationStatusAccepted: true,
		domain.ApplicationStatusRejected: true,
	}
	if !validStatuses[status] {
		return apperror.BadRequest("Invalid status. Must be: reviewed, accepted, or rejected")
	}

	// 2. Get application
	app, err := uc.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return apperror.NotFound("Application not found")
	}

	// 3. Validate employer owns the job
	if err := uc.validateJobOwnership(ctx, userID, app.JobID); err != nil {
		return err
	}

	// 4. Update status (also updates updated_at in repository)
	return uc.applicationRepo.UpdateStatus(ctx, applicationID, status)
}

// validateJobOwnership checks if the user can access the job's applications
// For now, we simply verify the job exists since company_profiles linking is not yet implemented
// TODO: When company_profiles are properly linked, validate job.company_id matches employer's company
func (uc *applicationUsecase) validateJobOwnership(ctx context.Context, userID string, jobID int64) error {
	// Verify job exists
	_, err := uc.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return apperror.NotFound("Job not found")
	}

	// TODO: Implement proper company ownership validation when company_profiles table is linked
	// For now, allow any employer to view any job's applications
	// This should be updated to check:
	// 1. Get employer's company_profile_id from a company_profiles table
	// 2. Compare with job.company_id
	// 3. Only allow access if they match

	_ = userID // Will be used for proper validation later

	return nil
}
