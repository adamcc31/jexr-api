package postgres

import (
	"context"
	"go-recruitment-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type jobRepo struct {
	db *pgxpool.Pool
}

func NewJobRepository(db *pgxpool.Pool) domain.JobRepository {
	return &jobRepo{db: db}
}

func (r *jobRepo) Create(ctx context.Context, job *domain.Job) error {
	query := `INSERT INTO jobs (company_id, title, description, salary_min, salary_max, location, company_status, employment_type, job_type, experience_level, qualifications, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id`
	err := r.db.QueryRow(ctx, query,
		job.CompanyID, job.Title, job.Description, job.SalaryMin, job.SalaryMax, job.Location, job.CompanyStatus,
		job.EmploymentType, job.JobType, job.ExperienceLevel, job.Qualifications,
		job.CreatedAt, job.UpdatedAt,
	).Scan(&job.ID)
	return err
}

func (r *jobRepo) GetByID(ctx context.Context, id int64) (*domain.Job, error) {
	query := `SELECT id, company_id, title, description, salary_min, salary_max, location, company_status, employment_type, job_type, experience_level, qualifications, created_at, updated_at FROM jobs WHERE id = $1`
	var job domain.Job
	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.CompanyID, &job.Title, &job.Description, &job.SalaryMin, &job.SalaryMax, &job.Location, &job.CompanyStatus,
		&job.EmploymentType, &job.JobType, &job.ExperienceLevel, &job.Qualifications,
		&job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// GetByIDWithCompany retrieves a job with company profile details
func (r *jobRepo) GetByIDWithCompany(ctx context.Context, id int64) (*domain.JobWithCompany, error) {
	query := `
		SELECT 
			j.id, j.company_id, j.title, j.description, j.salary_min, j.salary_max, 
			j.location, j.company_status, j.employment_type, j.job_type, 
			j.experience_level, j.qualifications, j.created_at, j.updated_at,
			COALESCE(cp.company_name, 'Unknown Company') as company_name,
			cp.logo_url,
			cp.website,
			cp.industry
		FROM jobs j
		LEFT JOIN company_profiles cp ON j.company_id = cp.id
		WHERE j.id = $1`

	var job domain.JobWithCompany
	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.CompanyID, &job.Title, &job.Description, &job.SalaryMin, &job.SalaryMax,
		&job.Location, &job.CompanyStatus, &job.EmploymentType, &job.JobType,
		&job.ExperienceLevel, &job.Qualifications, &job.CreatedAt, &job.UpdatedAt,
		&job.CompanyName, &job.CompanyLogoURL, &job.CompanyWebsite, &job.Industry,
	)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *jobRepo) Fetch(ctx context.Context, limit, offset int) ([]domain.Job, int64, error) {
	query := `SELECT id, company_id, title, description, salary_min, salary_max, location, company_status, employment_type, job_type, experience_level, qualifications, created_at, updated_at 
              FROM jobs ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var job domain.Job
		if err := rows.Scan(&job.ID, &job.CompanyID, &job.Title, &job.Description, &job.SalaryMin, &job.SalaryMax, &job.Location, &job.CompanyStatus, &job.EmploymentType, &job.JobType, &job.ExperienceLevel, &job.Qualifications, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs`).Scan(&total); err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// FetchWithCompany retrieves jobs with company profile data for public/candidate pages
func (r *jobRepo) FetchWithCompany(ctx context.Context, limit, offset int) ([]domain.JobWithCompany, int64, error) {
	query := `
		SELECT 
			j.id, j.company_id, j.title, j.description, j.salary_min, j.salary_max, 
			j.location, j.company_status, j.employment_type, j.job_type, 
			j.experience_level, j.qualifications, j.created_at, j.updated_at,
			COALESCE(cp.company_name, 'Unknown Company') as company_name,
			cp.logo_url,
			cp.website,
			cp.industry
		FROM jobs j
		LEFT JOIN company_profiles cp ON j.company_id = cp.id
		ORDER BY j.created_at DESC 
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []domain.JobWithCompany
	for rows.Next() {
		var job domain.JobWithCompany
		if err := rows.Scan(
			&job.ID, &job.CompanyID, &job.Title, &job.Description, &job.SalaryMin, &job.SalaryMax,
			&job.Location, &job.CompanyStatus, &job.EmploymentType, &job.JobType,
			&job.ExperienceLevel, &job.Qualifications, &job.CreatedAt, &job.UpdatedAt,
			&job.CompanyName, &job.CompanyLogoURL, &job.CompanyWebsite, &job.Industry,
		); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs`).Scan(&total); err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// FetchPublicActiveJobs retrieves only ACTIVE jobs with company data for public access
// SECURITY: This method hardcodes the 'active' filter - no client-side bypass possible
func (r *jobRepo) FetchPublicActiveJobs(ctx context.Context, limit, offset int) ([]domain.JobWithCompany, int64, error) {
	query := `
		SELECT 
			j.id, j.company_id, j.title, j.description, j.salary_min, j.salary_max, 
			j.location, j.company_status, j.employment_type, j.job_type, 
			j.experience_level, j.qualifications, j.created_at, j.updated_at,
			COALESCE(cp.company_name, 'Unknown Company') as company_name,
			cp.logo_url,
			cp.website,
			cp.industry
		FROM jobs j
		LEFT JOIN company_profiles cp ON j.company_id = cp.id
		WHERE j.company_status = 'active'
		ORDER BY j.created_at DESC 
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []domain.JobWithCompany
	for rows.Next() {
		var job domain.JobWithCompany
		if err := rows.Scan(
			&job.ID, &job.CompanyID, &job.Title, &job.Description, &job.SalaryMin, &job.SalaryMax,
			&job.Location, &job.CompanyStatus, &job.EmploymentType, &job.JobType,
			&job.ExperienceLevel, &job.Qualifications, &job.CreatedAt, &job.UpdatedAt,
			&job.CompanyName, &job.CompanyLogoURL, &job.CompanyWebsite, &job.Industry,
		); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE company_status = 'active'`).Scan(&total); err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// FetchByCompanyID retrieves jobs for a specific company (employer's jobs only)
func (r *jobRepo) FetchByCompanyID(ctx context.Context, companyID int64, limit, offset int) ([]domain.Job, int64, error) {
	query := `SELECT id, company_id, title, description, salary_min, salary_max, location, company_status, employment_type, job_type, experience_level, qualifications, created_at, updated_at 
              FROM jobs WHERE company_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, companyID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var job domain.Job
		if err := rows.Scan(&job.ID, &job.CompanyID, &job.Title, &job.Description, &job.SalaryMin, &job.SalaryMax, &job.Location, &job.CompanyStatus, &job.EmploymentType, &job.JobType, &job.ExperienceLevel, &job.Qualifications, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE company_id = $1`, companyID).Scan(&total); err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

func (r *jobRepo) Update(ctx context.Context, job *domain.Job) error {
	query := `UPDATE jobs SET 
		title = $2, 
		description = $3, 
		salary_min = $4, 
		salary_max = $5, 
		location = $6, 
		employment_type = $7, 
		job_type = $8, 
		experience_level = $9, 
		qualifications = $10, 
		updated_at = $11 
	WHERE id = $1`
	result, err := r.db.Exec(ctx, query,
		job.ID, job.Title, job.Description, job.SalaryMin, job.SalaryMax, job.Location,
		job.EmploymentType, job.JobType, job.ExperienceLevel, job.Qualifications,
		job.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *jobRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM jobs WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
