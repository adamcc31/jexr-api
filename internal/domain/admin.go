package domain

import "context"

// AdminStats contains dashboard statistics
type AdminStats struct {
	TotalUsers        int64             `json:"totalUsers"`
	UsersByRole       UsersByRole       `json:"usersByRole"`
	TotalCompanies    int64             `json:"totalCompanies"`
	CompaniesByStatus CompaniesByStatus `json:"companiesByStatus"`
	TotalJobs         int64             `json:"totalJobs"`
	ActiveJobs        int64             `json:"activeJobs"`
	TotalApplications int64             `json:"totalApplications"`
	SystemHealth      SystemHealth      `json:"systemHealth"`
}

type UsersByRole struct {
	Admin     int64 `json:"admin"`
	Employer  int64 `json:"employer"`
	Candidate int64 `json:"candidate"`
}

type CompaniesByStatus struct {
	Pending  int64 `json:"pending"`
	Verified int64 `json:"verified"`
	Rejected int64 `json:"rejected"`
}

type SystemHealth struct {
	Status      string `json:"status"`      // "healthy", "degraded", "down"
	LastChecked string `json:"lastChecked"` // ISO8601 timestamp
}

// AdminUser represents a user for admin management
type AdminUser struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	IsDisabled bool   `json:"isDisabled"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// AdminCompany represents a company for admin verification
type AdminCompany struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Email              string `json:"email"`
	VerificationStatus string `json:"verificationStatus"` // pending, verified, rejected
	EmployerId         string `json:"employerId"`
	EmployerEmail      string `json:"employerEmail"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

// AdminJob represents a job for admin moderation
type AdminJob struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	CompanyId   int64  `json:"companyId"`
	CompanyName string `json:"companyName"`
	Location    string `json:"location"`
	Status      string `json:"status"` // active, hidden, flagged
	IsFlagged   bool   `json:"isFlagged"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// Request structs for User CRUD
type CreateUserRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=candidate employer"`
}

type UpdateUserRequest struct {
	Email string `json:"email" binding:"omitempty,email"`
	Role  string `json:"role" binding:"omitempty,oneof=candidate employer"`
}

// PaginatedResult for list responses
type PaginatedResult[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
}

// AdminRepository defines admin-specific data access
type AdminRepository interface {
	// Stats
	GetStats(ctx context.Context) (*AdminStats, error)

	// Users
	ListUsers(ctx context.Context, role string, page, pageSize int) ([]AdminUser, int64, error)
	DisableUser(ctx context.Context, userID string, disable bool) error
	CreateUser(ctx context.Context, user AdminUser) error
	UpdateUser(ctx context.Context, user AdminUser) error
	DeleteUser(ctx context.Context, userID string) error

	// Companies (placeholder - returns empty for now if table doesn't exist)
	ListCompanies(ctx context.Context, status string, page, pageSize int) ([]AdminCompany, int64, error)
	VerifyCompany(ctx context.Context, companyID int64, action string, reason string) error

	// Jobs
	ListJobsForAdmin(ctx context.Context, status string, page, pageSize int) ([]AdminJob, int64, error)
	HideJob(ctx context.Context, jobID int64, hide bool) error
	FlagJob(ctx context.Context, jobID int64, flag bool, reason string) error
}

// AdminUsecase defines admin business logic
type AdminUsecase interface {
	// Stats
	GetStats(ctx context.Context) (*AdminStats, error)

	// Users
	ListUsers(ctx context.Context, role string, page, pageSize int) (*PaginatedResult[AdminUser], error)
	DisableUser(ctx context.Context, userID string, disable bool) (*AdminUser, error)
	CreateUser(ctx context.Context, req CreateUserRequest) (*AdminUser, error)
	UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*AdminUser, error)
	DeleteUser(ctx context.Context, userID string) error

	// Companies
	ListCompanies(ctx context.Context, status string, page, pageSize int) (*PaginatedResult[AdminCompany], error)
	VerifyCompany(ctx context.Context, companyID int64, action string, reason string) (*AdminCompany, error)

	// Jobs
	ListJobs(ctx context.Context, status string, page, pageSize int) (*PaginatedResult[AdminJob], error)
	HideJob(ctx context.Context, jobID int64, hide bool) (*AdminJob, error)
	FlagJob(ctx context.Context, jobID int64, flag bool, reason string) (*AdminJob, error)
}
