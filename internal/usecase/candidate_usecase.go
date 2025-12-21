package usecase

import (
	"context"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"

	"github.com/go-playground/validator/v10"
)

type candidateUsecase struct {
	repo     domain.CandidateRepository
	validate *validator.Validate
}

func NewCandidateUsecase(repo domain.CandidateRepository, validate *validator.Validate) domain.CandidateUsecase {
	return &candidateUsecase{
		repo:     repo,
		validate: validate,
	}
}

func (u *candidateUsecase) GetProfile(ctx context.Context, userID string) (*domain.CandidateProfile, error) {
	// Security: Ownership Check
	ctxUserID, ok := ctx.Value(domain.KeyUserID).(string)
	if !ok || ctxUserID == "" {
		return nil, apperror.Unauthorized("User not authenticated")
	}

	// Allow admin to bypass this check (optional, but good for admin dashboard)
	// For now, strict ownership:
	if ctxUserID != userID {
		// Check if user is admin if we want to allow admins
		/*
			role, _ := ctx.Value(domain.KeyUserRole).(string)
			if role != "admin" {
				return nil, apperror.Forbidden("You can only view your own profile")
			}
		*/
		return nil, apperror.Forbidden("You can only view your own profile")
	}

	profile, err := u.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, apperror.NotFound("Candidate profile not found")
	}
	return profile, nil
}

func (u *candidateUsecase) UpdateProfile(ctx context.Context, profile *domain.CandidateProfile) error {
	// Security: Verify context user matches profile user (IDOR prevention on update)
	ctxUserID, ok := ctx.Value(domain.KeyUserID).(string)
	if !ok || ctxUserID == "" {
		return apperror.Unauthorized("User not authenticated")
	}

	// Force the UserID to be the context user, ensuring they can't update someone else's profile
	profile.UserID = ctxUserID

	// Validation
	if err := u.validate.Struct(profile); err != nil {
		return apperror.BadRequest(err.Error())
	}

	return u.repo.Update(ctx, profile)
}
