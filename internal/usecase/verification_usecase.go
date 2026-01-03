package usecase

import (
	"context"
	"errors"
	"go-recruitment-backend/internal/domain"
	"slices"
	"strings"
	"time"
)

type verificationUsecase struct {
	verificationRepo domain.VerificationRepository
	userRepo         domain.UserRepository // If needed for status updates on user table?
}

func NewVerificationUsecase(repo domain.VerificationRepository, uRepo domain.UserRepository) domain.VerificationUsecase {
	return &verificationUsecase{
		verificationRepo: repo,
		userRepo:         uRepo,
	}
}

func (uc *verificationUsecase) GetPendingVerifications(ctx context.Context, page, limit int) ([]domain.AccountVerification, int64, error) {
	filter := domain.VerificationFilter{
		Status: domain.VerificationStatusPending,
		Page:   page,
		Limit:  limit,
	}
	return uc.verificationRepo.List(ctx, filter)
}

func (uc *verificationUsecase) ListVerifications(ctx context.Context, filter domain.VerificationFilter) ([]domain.AccountVerification, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	return uc.verificationRepo.List(ctx, filter)
}

func (uc *verificationUsecase) VerifyUser(ctx context.Context, adminID string, verificationID int64, action string, notes string) error {
	// 1. Get current verification
	v, err := uc.verificationRepo.GetByID(ctx, verificationID)
	if err != nil {
		return err
	}
	if v == nil {
		return errors.New("verification record not found")
	}

	// 2. Validate action
	action = strings.ToUpper(action)
	var newStatus string
	if action == "APPROVE" {
		newStatus = domain.VerificationStatusVerified
	} else if action == "REJECT" {
		newStatus = domain.VerificationStatusRejected
	} else {
		return errors.New("invalid action: must be APPROVE or REJECT")
	}

	// 3. Update status
	return uc.verificationRepo.UpdateStatus(ctx, verificationID, newStatus, adminID, notes)
}

func (uc *verificationUsecase) GetVerificationStatus(ctx context.Context, userID string) (*domain.VerificationResponse, error) {
	v, err := uc.verificationRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil // Or return a "not found" error or empty struct
	}

	experiences, err := uc.verificationRepo.GetWorkExperiences(ctx, v.ID)
	if err != nil {
		return nil, err
	}

	return &domain.VerificationResponse{
		Verification: v,
		Experiences:  experiences,
	}, nil
}

func (uc *verificationUsecase) GetVerificationByID(ctx context.Context, id int64) (*domain.VerificationResponse, error) {
	v, err := uc.verificationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}

	experiences, err := uc.verificationRepo.GetWorkExperiences(ctx, v.ID)
	if err != nil {
		return nil, err
	}

	return &domain.VerificationResponse{
		Verification: v,
		Experiences:  experiences,
	}, nil
}

func (uc *verificationUsecase) UpdateCandidateProfile(ctx context.Context, userID string, verification *domain.AccountVerification, experiences []domain.JapanWorkExperience) error {
	// 1. Validate enum fields (MANDATORY backend validation)
	if verification.MaritalStatus != nil && *verification.MaritalStatus != "" {
		if !slices.Contains(domain.ValidMaritalStatuses, *verification.MaritalStatus) {
			return errors.New("invalid marital_status: must be SINGLE, MARRIED, or DIVORCED")
		}
	}
	if verification.JapaneseSpeakingLevel != nil && *verification.JapaneseSpeakingLevel != "" {
		if !slices.Contains(domain.ValidJapaneseSpeakingLevels, *verification.JapaneseSpeakingLevel) {
			return errors.New("invalid japanese_speaking_level: must be NATIVE, FLUENT, BASIC, or PASSIVE")
		}
	}

	// 2. Check existence
	existing, err := uc.verificationRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// 3. Set up the verification record
	verification.UserID = userID
	verification.Role = "CANDIDATE"

	if existing == nil {
		// Create a new verification record
		verification.Status = domain.VerificationStatusPending
		id, err := uc.verificationRepo.Create(ctx, verification)
		if err != nil {
			return err
		}
		verification.ID = id
	} else {
		// Use existing ID
		verification.ID = existing.ID
	}

	// =========================================================================
	// Mandatory Field Validation for Verification Completion
	// =========================================================================
	// MANDATORY fields (all must be filled for SUBMITTED status):
	// - Profile Picture, First Name, Last Name, Occupation
	// - Phone, Birth Date, Domicile City
	// - Japan Experience Duration, CV Document
	// - At least one Japan Work Experience
	//
	// OPTIONAL fields (do NOT affect completion status):
	// - JLPT Certificate, Portfolio URL, Website URL, Intro
	// =========================================================================

	isComplete := true

	// Identity & Profile (Mandatory)
	if verification.ProfilePictureURL == nil || *verification.ProfilePictureURL == "" {
		isComplete = false
	}
	if verification.FirstName == nil || *verification.FirstName == "" {
		isComplete = false
	}
	if verification.LastName == nil || *verification.LastName == "" {
		isComplete = false
	}
	if verification.Occupation == nil || *verification.Occupation == "" {
		isComplete = false
	}
	if verification.Phone == nil || *verification.Phone == "" {
		isComplete = false
	}

	// Demographics (Mandatory)
	if verification.BirthDate == nil {
		isComplete = false
	}
	if verification.DomicileCity == nil || *verification.DomicileCity == "" {
		isComplete = false
	}

	// Experience (Mandatory)
	if verification.JapanExperienceDuration == nil {
		isComplete = false
	}

	// CV Document Upload (MANDATORY - not JLPT certificate)
	if verification.CvURL == nil || *verification.CvURL == "" {
		isComplete = false
	}

	// NOTE: JapaneseCertificateURL (JLPT) is now OPTIONAL - removed from mandatory checks
	// NOTE: PortfolioURL is OPTIONAL - not checked

	// NOTE: Japan Work Experience (experiences) is no longer required via this form.
	// Work experience is now captured in the unified work_experiences table through
	// the Professional Profile form. The old japan_work_experiences is deprecated.
	// Removed: if len(experiences) == 0 { isComplete = false }

	if isComplete {
		verification.Status = domain.VerificationStatusSubmitted
		verification.SubmittedAt = time.Now() // Reset submitted time on full submission
	} else {
		verification.Status = domain.VerificationStatusPending // Keep status as pending until complete
	}

	// Keep existing ID, UserID, CreatedAt, etc. The repository update query handles the updated fields.

	return uc.verificationRepo.UpdateProfile(ctx, verification, experiences)
}

func (uc *verificationUsecase) GetComprehensiveVerificationByID(ctx context.Context, id int64) (*domain.ComprehensiveVerificationResponse, error) {
	return uc.verificationRepo.GetComprehensiveByID(ctx, id)
}
