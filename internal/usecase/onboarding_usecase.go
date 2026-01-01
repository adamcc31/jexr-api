package usecase

import (
	"context"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type onboardingUsecase struct {
	repo     domain.OnboardingRepository
	validate *validator.Validate
}

func NewOnboardingUsecase(repo domain.OnboardingRepository, validate *validator.Validate) domain.OnboardingUsecase {
	return &onboardingUsecase{
		repo:     repo,
		validate: validate,
	}
}

// ============================================================================
// LPK Search
// ============================================================================

func (u *onboardingUsecase) SearchLPK(ctx context.Context, query string) ([]domain.LPK, error) {
	if query == "" {
		return []domain.LPK{}, nil
	}

	// Limit results to 20 for autocomplete
	results, err := u.repo.SearchLPK(ctx, query, 20)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "Failed to search LPK: "+err.Error(), err)
	}

	return results, nil
}

// ============================================================================
// Onboarding Status
// ============================================================================

func (u *onboardingUsecase) GetOnboardingStatus(ctx context.Context, userID string) (*domain.OnboardingStatus, error) {
	// Security: Verify context user matches requested user
	ctxUserID, ok := ctx.Value(domain.KeyUserID).(string)
	if !ok || ctxUserID == "" {
		return nil, apperror.Unauthorized("User not authenticated")
	}

	if ctxUserID != userID {
		return nil, apperror.Forbidden("You can only check your own onboarding status")
	}

	status, err := u.repo.GetOnboardingStatus(ctx, userID)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "Failed to get onboarding status: "+err.Error(), err)
	}

	return status, nil
}

// ============================================================================
// Complete Onboarding
// ============================================================================

func (u *onboardingUsecase) CompleteOnboarding(ctx context.Context, userID string, req *domain.OnboardingSubmitRequest) error {
	// Security: Verify context user matches requested user
	ctxUserID, ok := ctx.Value(domain.KeyUserID).(string)
	if !ok || ctxUserID == "" {
		return apperror.Unauthorized("User not authenticated")
	}

	if ctxUserID != userID {
		return apperror.Forbidden("You can only complete your own onboarding")
	}

	// Validate request
	if err := u.validate.Struct(req); err != nil {
		return apperror.BadRequest("Validation failed: " + err.Error())
	}

	// Business Logic: Validate interests
	if err := u.validateInterests(req.Interests); err != nil {
		return err
	}

	// Business Logic: Validate LPK selection
	if err := u.validateLPKSelection(ctx, &req.LPKSelection); err != nil {
		return err
	}

	// Business Logic: Validate company preferences
	if err := u.validateCompanyPreferences(req.CompanyPreferences); err != nil {
		return err
	}

	// Save all data atomically
	if err := u.repo.SaveOnboardingData(ctx, userID, req); err != nil {
		return apperror.New(http.StatusInternalServerError, "Failed to save onboarding data: "+err.Error(), err)
	}

	return nil
}

// validateInterests ensures interest selection is valid
func (u *onboardingUsecase) validateInterests(interests []domain.InterestKey) error {
	if len(interests) == 0 {
		return apperror.BadRequest("At least one interest must be selected")
	}

	// Check all interests are valid
	for _, interest := range interests {
		if !interest.IsValid() {
			return apperror.BadRequest("Invalid interest key: " + string(interest))
		}
	}

	// Check mutual exclusivity: if "none" is selected, no other interests allowed
	hasNone := false
	hasOthers := false
	for _, interest := range interests {
		if interest == domain.InterestNone {
			hasNone = true
		} else {
			hasOthers = true
		}
	}

	if hasNone && hasOthers {
		return apperror.BadRequest("Cannot select 'none' along with other interests")
	}

	return nil
}

// validateLPKSelection ensures LPK selection is mutually exclusive
func (u *onboardingUsecase) validateLPKSelection(ctx context.Context, sel *domain.LPKSelection) error {
	// Count how many options are set
	count := 0
	if sel.LPKID != nil {
		count++
	}
	if sel.OtherName != nil && *sel.OtherName != "" {
		count++
	}
	if sel.None {
		count++
	}

	if count == 0 {
		return apperror.BadRequest("LPK selection is required")
	}

	if count > 1 {
		return apperror.BadRequest("LPK selection must be mutually exclusive: choose list, other, or none")
	}

	// If LPK ID is provided, verify it exists
	if sel.LPKID != nil {
		lpk, err := u.repo.GetLPKByID(ctx, *sel.LPKID)
		if err != nil || lpk == nil {
			return apperror.BadRequest("Selected LPK not found")
		}
	}

	// If "other" is selected, name must not be empty
	if sel.OtherName != nil && *sel.OtherName == "" {
		return apperror.BadRequest("LPK name is required when selecting 'Lainnya'")
	}

	return nil
}

// validateCompanyPreferences ensures company preferences are valid
func (u *onboardingUsecase) validateCompanyPreferences(prefs []domain.CompanyPreferenceKey) error {
	if len(prefs) == 0 {
		return apperror.BadRequest("At least one company preference must be selected")
	}

	// Check all preferences are valid
	for _, pref := range prefs {
		if !pref.IsValid() {
			return apperror.BadRequest("Invalid company preference key: " + string(pref))
		}
	}

	return nil
}
