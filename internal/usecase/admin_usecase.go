package usecase

import (
	"context"
	"errors"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"math"
	"time"

	"github.com/google/uuid"
)

type adminUsecase struct {
	adminRepo domain.AdminRepository
}

func NewAdminUsecase(adminRepo domain.AdminRepository) domain.AdminUsecase {
	return &adminUsecase{adminRepo: adminRepo}
}

// GetStats returns dashboard statistics
func (u *adminUsecase) GetStats(ctx context.Context) (*domain.AdminStats, error) {
	// Check admin role from context
	if err := u.requireAdmin(ctx); err != nil {
		return nil, err
	}

	stats, err := u.adminRepo.GetStats(ctx)
	if err != nil {
		return nil, apperror.Internal(errors.New("Failed to fetch statistics: " + err.Error()))
	}

	return stats, nil
}

// ListUsers returns paginated users
func (u *adminUsecase) ListUsers(ctx context.Context, role string, page, pageSize int) (*domain.PaginatedResult[domain.AdminUser], error) {
	if err := u.requireAdmin(ctx); err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	users, total, err := u.adminRepo.ListUsers(ctx, role, page, pageSize)
	if err != nil {
		return nil, apperror.Internal(errors.New("Failed to fetch users: " + err.Error()))
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &domain.PaginatedResult[domain.AdminUser]{
		Data:       users,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// DisableUser enables or disables a user
func (u *adminUsecase) DisableUser(ctx context.Context, userID string, disable bool) (*domain.AdminUser, error) {
	if err := u.requireAdmin(ctx); err != nil {
		return nil, err
	}

	if userID == "" {
		return nil, apperror.BadRequest("User ID is required")
	}

	err := u.adminRepo.DisableUser(ctx, userID, disable)
	if err != nil {
		return nil, apperror.Internal(errors.New("Failed to update user: " + err.Error()))
	}

	// Fetch updated user
	users, _, err := u.adminRepo.ListUsers(ctx, "", 1, 1)
	if err != nil || len(users) == 0 {
		// Return minimal response
		return &domain.AdminUser{ID: userID, IsDisabled: disable}, nil
	}

	// Find the specific user
	for _, u := range users {
		if u.ID == userID {
			return &u, nil
		}
	}

	return &domain.AdminUser{ID: userID, IsDisabled: disable}, nil
}

// CreateUser creates a new user (DB only)
func (u *adminUsecase) CreateUser(ctx context.Context, req domain.CreateUserRequest) (*domain.AdminUser, error) {
	if err := u.requireAdmin(ctx); err != nil {
		return nil, err
	}

	user := domain.AdminUser{
		ID:         uuid.NewString(), // Generate random UUID
		Email:      req.Email,
		Role:       req.Role,
		IsDisabled: false,
		CreatedAt:  time.Now().Format(time.RFC3339),
		UpdatedAt:  time.Now().Format(time.RFC3339),
	}

	err := u.adminRepo.CreateUser(ctx, user)
	if err != nil {
		return nil, apperror.Internal(errors.New("Failed to create user: " + err.Error()))
	}

	return &user, nil
}

// UpdateUser updates an existing user
func (u *adminUsecase) UpdateUser(ctx context.Context, userID string, req domain.UpdateUserRequest) (*domain.AdminUser, error) {
	if err := u.requireAdmin(ctx); err != nil {
		return nil, err
	}

	// Note: Repo UpdateUser expects full object (PUT semantics)
	// If fields are missing in req, they will be overwritten with empty strings if we aren't careful.
	// We assume frontend sends all fields for now.

	user := domain.AdminUser{
		ID:        userID,
		Email:     req.Email,
		Role:      req.Role,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	err := u.adminRepo.UpdateUser(ctx, user)
	if err != nil {
		return nil, apperror.Internal(errors.New("Failed to update user: " + err.Error()))
	}

	return &user, nil
}

// DeleteUser deletes a user
func (u *adminUsecase) DeleteUser(ctx context.Context, userID string) error {
	if err := u.requireAdmin(ctx); err != nil {
		return err
	}

	err := u.adminRepo.DeleteUser(ctx, userID)
	if err != nil {
		return apperror.Internal(errors.New("Failed to delete user: " + err.Error()))
	}
	return nil
}

// ListCompanies returns paginated companies
func (u *adminUsecase) ListCompanies(ctx context.Context, status string, page, pageSize int) (*domain.PaginatedResult[domain.AdminCompany], error) {
	if err := u.requireAdmin(ctx); err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	companies, total, err := u.adminRepo.ListCompanies(ctx, status, page, pageSize)
	if err != nil {
		return nil, apperror.Internal(errors.New("Failed to fetch companies: " + err.Error()))
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &domain.PaginatedResult[domain.AdminCompany]{
		Data:       companies,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// VerifyCompany approves or rejects a company
func (u *adminUsecase) VerifyCompany(ctx context.Context, companyID int64, action string, reason string) (*domain.AdminCompany, error) {
	if err := u.requireAdmin(ctx); err != nil {
		return nil, err
	}

	if action != "approve" && action != "reject" {
		return nil, apperror.BadRequest("Action must be 'approve' or 'reject'")
	}

	err := u.adminRepo.VerifyCompany(ctx, companyID, action, reason)
	if err != nil {
		return nil, apperror.Internal(errors.New("Failed to verify company: " + err.Error()))
	}

	status := "verified"
	if action == "reject" {
		status = "rejected"
	}

	return &domain.AdminCompany{ID: companyID, VerificationStatus: status}, nil
}

// ListJobs returns paginated jobs for moderation
func (u *adminUsecase) ListJobs(ctx context.Context, status string, page, pageSize int) (*domain.PaginatedResult[domain.AdminJob], error) {
	if err := u.requireAdmin(ctx); err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	jobs, total, err := u.adminRepo.ListJobsForAdmin(ctx, status, page, pageSize)
	if err != nil {
		return nil, apperror.Internal(errors.New("Failed to fetch jobs: " + err.Error()))
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &domain.PaginatedResult[domain.AdminJob]{
		Data:       jobs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// HideJob hides or unhides a job
func (u *adminUsecase) HideJob(ctx context.Context, jobID int64, hide bool) (*domain.AdminJob, error) {
	if err := u.requireAdmin(ctx); err != nil {
		return nil, err
	}

	err := u.adminRepo.HideJob(ctx, jobID, hide)
	if err != nil {
		return nil, apperror.Internal(errors.New("Failed to update job: " + err.Error()))
	}

	status := "active"
	if hide {
		status = "hidden"
	}

	return &domain.AdminJob{ID: jobID, Status: status}, nil
}

// FlagJob flags or unflags a job
func (u *adminUsecase) FlagJob(ctx context.Context, jobID int64, flag bool, reason string) (*domain.AdminJob, error) {
	if err := u.requireAdmin(ctx); err != nil {
		return nil, err
	}

	err := u.adminRepo.FlagJob(ctx, jobID, flag, reason)
	if err != nil {
		return nil, apperror.Internal(errors.New("Failed to flag job: " + err.Error()))
	}

	return &domain.AdminJob{ID: jobID, IsFlagged: flag}, nil
}

// requireAdmin checks if the current user has admin role
// Works with both Gin context (c.Set) and standard context.WithValue
func (u *adminUsecase) requireAdmin(ctx context.Context) error {
	var role string

	// First try Gin context string key (from c.Set)
	if r, ok := ctx.Value(string(domain.KeyUserRole)).(string); ok {
		role = r
	}

	// Fallback to CtxKey type (from context.WithValue)
	if role == "" {
		if r, ok := ctx.Value(domain.KeyUserRole).(string); ok {
			role = r
		}
	}

	if role != "admin" {
		return apperror.Forbidden("Admin access required")
	}
	return nil
}
