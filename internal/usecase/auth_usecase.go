package usecase

import (
	"context"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"time"
)

type authUsecase struct {
	userRepo domain.UserRepository
}

func NewAuthUsecase(userRepo domain.UserRepository) domain.AuthUsecase {
	return &authUsecase{userRepo: userRepo}
}

func (u *authUsecase) EnsureUserExists(ctx context.Context, user *domain.User) error {
	existing, err := u.userRepo.GetByID(ctx, user.ID)
	// If exists, check if we need to sync fields (e.g. Role)
	if existing != nil && err == nil {
		if user.Role != "" && existing.Role != user.Role {
			existing.Role = user.Role
			existing.UpdatedAt = time.Now()
			return u.userRepo.Update(ctx, existing)
		}
		return nil // Already exists and up to date
	}

	// Default to 'candidate' if no role
	if user.Role == "" {
		user.Role = "candidate"
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	return u.userRepo.Create(ctx, user)
}

// SyncUserFromAuth synchronizes a Supabase auth user to local database.
// This is called after successful login to ensure local DB is in sync.
// CRITICAL: This method is idempotent - it handles ID mismatches gracefully
// and will NEVER fail on existing users due to duplicate email constraints.
func (u *authUsecase) SyncUserFromAuth(ctx context.Context, user *domain.User) error {
	// Step 1: Try to find by ID (happy path - normal returning user)
	existing, err := u.userRepo.GetByID(ctx, user.ID)
	if existing != nil && err == nil {
		// Found by ID - user exists with matching ID
		// Only update if role sync is needed (and role was provided)
		if user.Role != "" && existing.Role != user.Role {
			existing.Role = user.Role
			existing.UpdatedAt = time.Now()
			return u.userRepo.Update(ctx, existing)
		}
		return nil // User is in sync, nothing to do
	}

	// Step 2: ID not found - check by email (handles ID mismatch / Supabase re-registration)
	existingByEmail, err := u.userRepo.GetByEmail(ctx, user.Email)
	if existingByEmail != nil && err == nil {
		// Email exists but with different ID - user re-registered in Supabase (new UUID)
		// Update the record with new Supabase ID to re-link accounts
		// IMPORTANT: Use UpdateByEmail since the ID in DB is the OLD one
		existingByEmail.ID = user.ID
		existingByEmail.UpdatedAt = time.Now()
		// Only sync role if explicitly provided and different
		if user.Role != "" && existingByEmail.Role != user.Role {
			existingByEmail.Role = user.Role
		}
		return u.userRepo.UpdateByEmail(ctx, user.Email, existingByEmail)
	}

	// Step 3: Truly new user - neither ID nor email exists, create record
	if user.Role == "" {
		user.Role = "candidate"
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	return u.userRepo.Create(ctx, user)
}

func (u *authUsecase) AssignRole(ctx context.Context, userID string, role string) error {
	// Security: Only admin can assign roles
	ctxRole, ok := ctx.Value(domain.KeyUserRole).(string)
	if !ok || ctxRole != "admin" {
		return apperror.Forbidden("Only admins can assign roles")
	}

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return apperror.NotFound("User not found")
	}

	user.Role = role
	user.UpdatedAt = time.Now()
	return u.userRepo.Update(ctx, user)
}

func (u *authUsecase) GetCurrentUser(ctx context.Context, id string) (*domain.User, error) {
	return u.userRepo.GetByID(ctx, id)
}

func (u *authUsecase) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// If error is "no rows", email doesn't exist
		if err.Error() == "no rows in result set" {
			return false, nil
		}
		return false, err
	}
	return user != nil, nil
}
