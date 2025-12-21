package domain

import (
	"context"
	"errors"
	"time"
)

// Common domain errors
var ErrNotFound = errors.New("resource not found")

type Job struct {
	ID              int64     `json:"id"`
	CompanyID       int64     `json:"company_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	SalaryMin       float64   `json:"salary_min"`
	SalaryMax       float64   `json:"salary_max"`
	Location        string    `json:"location"`
	CompanyStatus   string    `json:"company_status"`
	EmploymentType  *string   `json:"employment_type"`
	JobType         *string   `json:"job_type"`
	ExperienceLevel *string   `json:"experience_level"`
	Qualifications  *string   `json:"qualifications"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// JobWithCompany extends Job with company profile information
type JobWithCompany struct {
	Job
	CompanyName    string  `json:"company_name"`
	CompanyLogoURL *string `json:"company_logo_url"`
	CompanyWebsite *string `json:"company_website"`
	Industry       *string `json:"industry"`
}

type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id int64) (*Job, error)
	GetByIDWithCompany(ctx context.Context, id int64) (*JobWithCompany, error)
	Fetch(ctx context.Context, limit, offset int) ([]Job, int64, error)
	FetchWithCompany(ctx context.Context, limit, offset int) ([]JobWithCompany, int64, error)
	FetchPublicActiveJobs(ctx context.Context, limit, offset int) ([]JobWithCompany, int64, error)
	FetchByCompanyID(ctx context.Context, companyID int64, limit, offset int) ([]Job, int64, error)
	Update(ctx context.Context, job *Job) error
	Delete(ctx context.Context, id int64) error
}

type JobUsecase interface {
	CreateJob(ctx context.Context, userID string, job *Job) error
	GetJobDetails(ctx context.Context, id int64) (*Job, error)
	GetJobDetailsWithCompany(ctx context.Context, id int64) (*JobWithCompany, error)
	ListJobs(ctx context.Context, page, pageSize int) ([]Job, int64, error)
	ListJobsWithCompany(ctx context.Context, page, pageSize int) ([]JobWithCompany, int64, error)
	ListPublicActiveJobs(ctx context.Context, page, pageSize int) ([]JobWithCompany, int64, error)
	ListJobsByEmployer(ctx context.Context, userID string, page, pageSize int) ([]Job, int64, error)
	UpdateJob(ctx context.Context, job *Job) error
	DeleteJob(ctx context.Context, id int64) error
}
