package usecase

import (
	"context"
	"fmt"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"log"
	"time"

	"github.com/go-playground/validator/v10"
)

type candidateUsecase struct {
	repo             domain.CandidateRepository
	verificationRepo domain.VerificationRepository
	validate         *validator.Validate
}

func NewCandidateUsecase(repo domain.CandidateRepository, verificationRepo domain.VerificationRepository, validate *validator.Validate) domain.CandidateUsecase {
	return &candidateUsecase{
		repo:             repo,
		verificationRepo: verificationRepo,
		validate:         validate,
	}
}

func (u *candidateUsecase) GetProfile(ctx context.Context, userID string) (*domain.CandidateProfile, error) {
	// Authorization
	authID, _ := ctx.Value(domain.KeyUserID).(string)
	if authID == "" {
		return nil, apperror.Unauthorized("Not authenticated")
	}
	if authID != userID {
		return nil, apperror.Forbidden("Access denied")
	}

	return u.repo.GetByUserID(ctx, userID)
}

func (u *candidateUsecase) UpdateProfile(ctx context.Context, profile *domain.CandidateProfile) error {
	authID, _ := ctx.Value(domain.KeyUserID).(string)
	if authID == "" {
		return apperror.Unauthorized("Not authenticated")
	}
	profile.UserID = authID

	if err := u.validate.Struct(profile); err != nil {
		return apperror.BadRequest(err.Error())
	}
	return u.repo.Update(ctx, profile)
}

// ============================================================================
// Full Profile Operations
// ============================================================================

func (u *candidateUsecase) GetFullProfile(ctx context.Context, userID string) (*domain.CandidateWithFullDetails, error) {
	authID, ok := ctx.Value(domain.KeyUserID).(string)

	if !ok || authID == "" {
		return nil, apperror.Unauthorized("Authentication required")
	}

	if authID != userID {
		return nil, apperror.Forbidden("You can only view your own profile")
	}

	fullProfile, err := u.repo.GetFullProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if fullProfile == nil {
		return nil, apperror.NotFound("Profile not found")
	}

	return fullProfile, nil
}

func (u *candidateUsecase) UpdateFullProfile(ctx context.Context, userID string, req *domain.CandidateWithFullDetails) error {
	authID, ok := ctx.Value(domain.KeyUserID).(string)
	if !ok || authID == "" {
		return apperror.Unauthorized("Authentication required")
	}
	if authID != userID {
		return apperror.Forbidden("You can only update your own profile")
	}

	// Force UserID consistency
	req.Profile.UserID = authID
	req.Details.UserID = authID
	for i := range req.WorkExperiences {
		req.WorkExperiences[i].UserID = authID
	}

	// Validation
	if err := u.validate.Struct(&req.Profile); err != nil {
		return apperror.BadRequest("Profile Validation: " + err.Error())
	}
	if err := u.validate.Struct(&req.Details); err != nil {
		return apperror.BadRequest("Details Validation: " + err.Error())
	}
	for i, we := range req.WorkExperiences {
		if err := u.validate.Struct(we); err != nil {
			return apperror.BadRequest(fmt.Sprintf("WorkExperience[%d]: %s", i, err.Error()))
		}
	}

	err := u.repo.UpsertFullProfile(ctx, req)
	if err != nil {
		return err
	}

	// Update verification submitted_at timestamp for correct admin sorting
	if u.verificationRepo != nil {
		if err := u.verificationRepo.UpdateSubmittedAt(ctx, authID, time.Now()); err != nil {
			// Log but don't fail - profile was saved successfully
			log.Printf("Warning: failed to update verification submitted_at: %v", err)
		}
	}

	return nil
}

func (u *candidateUsecase) GetMasterSkills(ctx context.Context) ([]domain.Skill, error) {
	return u.repo.GetAllSkills(ctx)
}
