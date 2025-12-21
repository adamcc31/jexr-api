package usecase

import (
	"context"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"time"
)

type jobUsecase struct {
	jobRepo            domain.JobRepository
	companyProfileRepo domain.CompanyProfileRepository
}

func NewJobUsecase(jobRepo domain.JobRepository, companyProfileRepo domain.CompanyProfileRepository) domain.JobUsecase {
	return &jobUsecase{
		jobRepo:            jobRepo,
		companyProfileRepo: companyProfileRepo,
	}
}

func (u *jobUsecase) CreateJob(ctx context.Context, userID string, job *domain.Job) error {
	// Removed context.WithTimeout to fix DOS vulnerability

	// Get employer's company profile to set CompanyID
	companyProfile, err := u.companyProfileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return apperror.NotFound("Employer profile not found. Please create a company profile first.")
	}
	job.CompanyID = companyProfile.ID

	// Business Validation
	if job.SalaryMin > job.SalaryMax {
		return apperror.BadRequest("SalaryMin cannot be greater than SalaryMax")
	}
	if job.Title == "" {
		return apperror.BadRequest("Title is required")
	}

	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	return u.jobRepo.Create(ctx, job)
}

func (u *jobUsecase) GetJobDetails(ctx context.Context, id int64) (*domain.Job, error) {
	// Removed context.WithTimeout
	job, err := u.jobRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// GetJobDetailsWithCompany returns job with company profile data
func (u *jobUsecase) GetJobDetailsWithCompany(ctx context.Context, id int64) (*domain.JobWithCompany, error) {
	job, err := u.jobRepo.GetByIDWithCompany(ctx, id)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (u *jobUsecase) ListJobs(ctx context.Context, page, pageSize int) ([]domain.Job, int64, error) {
	// Removed context.WithTimeout
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	return u.jobRepo.Fetch(ctx, pageSize, offset)
}

// ListJobsWithCompany returns jobs with company profile data for public/candidate pages
func (u *jobUsecase) ListJobsWithCompany(ctx context.Context, page, pageSize int) ([]domain.JobWithCompany, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	return u.jobRepo.FetchWithCompany(ctx, pageSize, offset)
}

// ListPublicActiveJobs returns only active jobs for public access
// SECURITY: This enforces server-side filtering - client cannot bypass
func (u *jobUsecase) ListPublicActiveJobs(ctx context.Context, page, pageSize int) ([]domain.JobWithCompany, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	return u.jobRepo.FetchPublicActiveJobs(ctx, pageSize, offset)
}

// ListJobsByEmployer returns jobs belonging to a specific employer based on their user ID
func (u *jobUsecase) ListJobsByEmployer(ctx context.Context, userID string, page, pageSize int) ([]domain.Job, int64, error) {
	// Get employer's company profile to find company ID
	companyProfile, err := u.companyProfileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, 0, apperror.NotFound("Employer profile not found. Please create a company profile first.")
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	return u.jobRepo.FetchByCompanyID(ctx, companyProfile.ID, pageSize, offset)
}

func (u *jobUsecase) UpdateJob(ctx context.Context, job *domain.Job) error {
	// Business Validation
	if job.SalaryMin > job.SalaryMax {
		return apperror.BadRequest("SalaryMin cannot be greater than SalaryMax")
	}
	if job.Title == "" {
		return apperror.BadRequest("Title is required")
	}

	job.UpdatedAt = time.Now()

	return u.jobRepo.Update(ctx, job)
}

func (u *jobUsecase) DeleteJob(ctx context.Context, id int64) error {
	return u.jobRepo.Delete(ctx, id)
}
