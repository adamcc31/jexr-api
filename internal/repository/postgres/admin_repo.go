package postgres

import (
	"context"
	"go-recruitment-backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type adminRepo struct {
	db *pgxpool.Pool
}

func NewAdminRepository(db *pgxpool.Pool) domain.AdminRepository {
	return &adminRepo{db: db}
}

// GetStats fetches dashboard statistics
func (r *adminRepo) GetStats(ctx context.Context) (*domain.AdminStats, error) {
	stats := &domain.AdminStats{
		SystemHealth: domain.SystemHealth{
			Status:      "healthy",
			LastChecked: time.Now().Format(time.RFC3339),
		},
	}

	// Total users and by role
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)
	if err != nil {
		stats.TotalUsers = 0
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&stats.UsersByRole.Admin)
	if err != nil {
		stats.UsersByRole.Admin = 0
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'employer'`).Scan(&stats.UsersByRole.Employer)
	if err != nil {
		stats.UsersByRole.Employer = 0
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = 'candidate'`).Scan(&stats.UsersByRole.Candidate)
	if err != nil {
		stats.UsersByRole.Candidate = 0
	}

	// Total jobs
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs`).Scan(&stats.TotalJobs)
	if err != nil {
		stats.TotalJobs = 0
	}

	// Active jobs (not hidden)
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE COALESCE(status, 'active') = 'active'`).Scan(&stats.ActiveJobs)
	if err != nil {
		stats.ActiveJobs = stats.TotalJobs
	}

	// Companies - check if table exists first
	var tableExists bool
	err = r.db.QueryRow(ctx, `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'companies')`).Scan(&tableExists)
	if err == nil && tableExists {
		r.db.QueryRow(ctx, `SELECT COUNT(*) FROM companies`).Scan(&stats.TotalCompanies)
		r.db.QueryRow(ctx, `SELECT COUNT(*) FROM companies WHERE verification_status = 'pending'`).Scan(&stats.CompaniesByStatus.Pending)
		r.db.QueryRow(ctx, `SELECT COUNT(*) FROM companies WHERE verification_status = 'verified'`).Scan(&stats.CompaniesByStatus.Verified)
		r.db.QueryRow(ctx, `SELECT COUNT(*) FROM companies WHERE verification_status = 'rejected'`).Scan(&stats.CompaniesByStatus.Rejected)
	}

	// Applications - check if table exists
	err = r.db.QueryRow(ctx, `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'applications')`).Scan(&tableExists)
	if err == nil && tableExists {
		r.db.QueryRow(ctx, `SELECT COUNT(*) FROM applications`).Scan(&stats.TotalApplications)
	}

	return stats, nil
}

// ListUsers fetches paginated users with optional role filter
func (r *adminRepo) ListUsers(ctx context.Context, role string, page, pageSize int) ([]domain.AdminUser, int64, error) {
	var total int64
	var users []domain.AdminUser

	offset := (page - 1) * pageSize

	// Try to add is_disabled column if it doesn't exist (ignore errors)
	_, _ = r.db.Exec(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS is_disabled BOOLEAN DEFAULT false`)

	// Count query
	countQuery := `SELECT COUNT(*) FROM users`
	if role != "" {
		countQuery += ` WHERE role = $1`
		err := r.db.QueryRow(ctx, countQuery, role).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
	} else {
		err := r.db.QueryRow(ctx, countQuery).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
	}

	// Data query - try with is_disabled first, fallback to simpler query
	if role != "" {
		query := `SELECT id, email, role, COALESCE(is_disabled, false), created_at, updated_at 
		          FROM users WHERE role = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		rows, err := r.db.Query(ctx, query, role, pageSize, offset)
		if err != nil {
			// Fallback without is_disabled
			query = `SELECT id, email, role, false, created_at, updated_at 
			         FROM users WHERE role = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
			rows, err = r.db.Query(ctx, query, role, pageSize, offset)
			if err != nil {
				return []domain.AdminUser{}, 0, nil
			}
		}
		defer rows.Close()
		for rows.Next() {
			var u domain.AdminUser
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.IsDisabled, &createdAt, &updatedAt); err != nil {
				continue
			}
			u.CreatedAt = createdAt.Format(time.RFC3339)
			u.UpdatedAt = updatedAt.Format(time.RFC3339)
			users = append(users, u)
		}
	} else {
		query := `SELECT id, email, role, COALESCE(is_disabled, false), created_at, updated_at 
		          FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		rows, err := r.db.Query(ctx, query, pageSize, offset)
		if err != nil {
			// Fallback without is_disabled
			query = `SELECT id, email, role, false, created_at, updated_at 
			         FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
			rows, err = r.db.Query(ctx, query, pageSize, offset)
			if err != nil {
				return []domain.AdminUser{}, 0, nil
			}
		}
		defer rows.Close()
		for rows.Next() {
			var u domain.AdminUser
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.IsDisabled, &createdAt, &updatedAt); err != nil {
				continue
			}
			u.CreatedAt = createdAt.Format(time.RFC3339)
			u.UpdatedAt = updatedAt.Format(time.RFC3339)
			users = append(users, u)
		}
	}

	if users == nil {
		users = []domain.AdminUser{}
	}

	return users, total, nil
}

// DisableUser enables or disables a user
func (r *adminRepo) DisableUser(ctx context.Context, userID string, disable bool) error {
	// First try to add is_disabled column if it doesn't exist
	_, _ = r.db.Exec(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS is_disabled BOOLEAN DEFAULT false`)

	query := `UPDATE users SET is_disabled = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID, disable, time.Now())
	return err
}

// CreateUser inserts a new user
func (r *adminRepo) CreateUser(ctx context.Context, u domain.AdminUser) error {
	// First try to add is_disabled column if it doesn't exist
	_, _ = r.db.Exec(ctx, `ALTER TABLE users ADD COLUMN IF NOT EXISTS is_disabled BOOLEAN DEFAULT false`)

	query := `INSERT INTO users (id, email, role, is_disabled, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
	created, _ := time.Parse(time.RFC3339, u.CreatedAt)
	updated, _ := time.Parse(time.RFC3339, u.UpdatedAt)

	if created.IsZero() {
		created = time.Now()
	}
	if updated.IsZero() {
		updated = time.Now()
	}

	_, err := r.db.Exec(ctx, query, u.ID, u.Email, u.Role, u.IsDisabled, created, updated)
	return err
}

// UpdateUser updates an existing user
func (r *adminRepo) UpdateUser(ctx context.Context, u domain.AdminUser) error {
	query := `UPDATE users SET email = $2, role = $3, updated_at = $4 WHERE id = $1`
	updated, _ := time.Parse(time.RFC3339, u.UpdatedAt)
	if updated.IsZero() {
		updated = time.Now()
	}

	_, err := r.db.Exec(ctx, query, u.ID, u.Email, u.Role, updated)
	return err
}

// DeleteUser removes a user
func (r *adminRepo) DeleteUser(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// ListCompanies fetches paginated companies (placeholder - returns empty if table doesn't exist)
func (r *adminRepo) ListCompanies(ctx context.Context, status string, page, pageSize int) ([]domain.AdminCompany, int64, error) {
	// Check if companies table exists
	var tableExists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'companies')`).Scan(&tableExists)
	if err != nil || !tableExists {
		return []domain.AdminCompany{}, 0, nil
	}

	var total int64
	var companies []domain.AdminCompany

	offset := (page - 1) * pageSize

	// Count query
	countQuery := `SELECT COUNT(*) FROM companies`
	if status != "" {
		countQuery += ` WHERE verification_status = $1`
		r.db.QueryRow(ctx, countQuery, status).Scan(&total)
	} else {
		r.db.QueryRow(ctx, countQuery).Scan(&total)
	}

	// Data query
	if status != "" {
		query := `SELECT id, name, email, verification_status, employer_id, 
		          COALESCE((SELECT email FROM users WHERE id = companies.employer_id), ''), 
		          created_at, updated_at 
		          FROM companies WHERE verification_status = $1 
		          ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		rows, err := r.db.Query(ctx, query, status, pageSize, offset)
		if err != nil {
			return []domain.AdminCompany{}, 0, nil
		}
		defer rows.Close()
		for rows.Next() {
			var c domain.AdminCompany
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.VerificationStatus, &c.EmployerId, &c.EmployerEmail, &createdAt, &updatedAt); err != nil {
				continue
			}
			c.CreatedAt = createdAt.Format(time.RFC3339)
			c.UpdatedAt = updatedAt.Format(time.RFC3339)
			companies = append(companies, c)
		}
	} else {
		query := `SELECT id, name, email, verification_status, employer_id,
		          COALESCE((SELECT email FROM users WHERE id = companies.employer_id), ''),
		          created_at, updated_at 
		          FROM companies ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		rows, err := r.db.Query(ctx, query, pageSize, offset)
		if err != nil {
			return []domain.AdminCompany{}, 0, nil
		}
		defer rows.Close()
		for rows.Next() {
			var c domain.AdminCompany
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.VerificationStatus, &c.EmployerId, &c.EmployerEmail, &createdAt, &updatedAt); err != nil {
				continue
			}
			c.CreatedAt = createdAt.Format(time.RFC3339)
			c.UpdatedAt = updatedAt.Format(time.RFC3339)
			companies = append(companies, c)
		}
	}

	if companies == nil {
		companies = []domain.AdminCompany{}
	}

	return companies, total, nil
}

// VerifyCompany approves or rejects a company
func (r *adminRepo) VerifyCompany(ctx context.Context, companyID int64, action string, reason string) error {
	status := "verified"
	if action == "reject" {
		status = "rejected"
	}

	query := `UPDATE companies SET verification_status = $2, rejection_reason = $3, updated_at = $4 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, companyID, status, reason, time.Now())
	return err
}

// ListJobsForAdmin fetches paginated jobs for moderation
func (r *adminRepo) ListJobsForAdmin(ctx context.Context, status string, page, pageSize int) ([]domain.AdminJob, int64, error) {
	var total int64
	var jobs []domain.AdminJob

	offset := (page - 1) * pageSize

	// First ensure the needed columns exist
	_, _ = r.db.Exec(ctx, `ALTER TABLE jobs ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active'`)
	_, _ = r.db.Exec(ctx, `ALTER TABLE jobs ADD COLUMN IF NOT EXISTS is_flagged BOOLEAN DEFAULT false`)

	// Count query
	countQuery := `SELECT COUNT(*) FROM jobs`
	if status != "" {
		countQuery += ` WHERE COALESCE(status, 'active') = $1`
		r.db.QueryRow(ctx, countQuery, status).Scan(&total)
	} else {
		r.db.QueryRow(ctx, countQuery).Scan(&total)
	}

	// Data query
	if status != "" {
		query := `SELECT j.id, j.title, j.company_id, COALESCE(c.name, 'Unknown'), j.location, 
		          COALESCE(j.status, 'active'), COALESCE(j.is_flagged, false), j.created_at, j.updated_at 
		          FROM jobs j 
		          LEFT JOIN companies c ON j.company_id = c.id
		          WHERE COALESCE(j.status, 'active') = $1 
		          ORDER BY j.created_at DESC LIMIT $2 OFFSET $3`
		rows, err := r.db.Query(ctx, query, status, pageSize, offset)
		if err != nil {
			// Fallback query without company join
			query = `SELECT id, title, company_id, 'Unknown', location, 
			         COALESCE(status, 'active'), COALESCE(is_flagged, false), created_at, updated_at 
			         FROM jobs WHERE COALESCE(status, 'active') = $1 
			         ORDER BY created_at DESC LIMIT $2 OFFSET $3`
			rows, err = r.db.Query(ctx, query, status, pageSize, offset)
			if err != nil {
				return []domain.AdminJob{}, 0, nil
			}
		}
		defer rows.Close()
		for rows.Next() {
			var j domain.AdminJob
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&j.ID, &j.Title, &j.CompanyId, &j.CompanyName, &j.Location, &j.Status, &j.IsFlagged, &createdAt, &updatedAt); err != nil {
				continue
			}
			j.CreatedAt = createdAt.Format(time.RFC3339)
			j.UpdatedAt = updatedAt.Format(time.RFC3339)
			jobs = append(jobs, j)
		}
	} else {
		query := `SELECT j.id, j.title, j.company_id, COALESCE(c.name, 'Unknown'), j.location, 
		          COALESCE(j.status, 'active'), COALESCE(j.is_flagged, false), j.created_at, j.updated_at 
		          FROM jobs j 
		          LEFT JOIN companies c ON j.company_id = c.id
		          ORDER BY j.created_at DESC LIMIT $1 OFFSET $2`
		rows, err := r.db.Query(ctx, query, pageSize, offset)
		if err != nil {
			// Fallback query without company join
			query = `SELECT id, title, company_id, 'Unknown', location, 
			         COALESCE(status, 'active'), COALESCE(is_flagged, false), created_at, updated_at 
			         FROM jobs ORDER BY created_at DESC LIMIT $1 OFFSET $2`
			rows, err = r.db.Query(ctx, query, pageSize, offset)
			if err != nil {
				return []domain.AdminJob{}, 0, nil
			}
		}
		defer rows.Close()
		for rows.Next() {
			var j domain.AdminJob
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&j.ID, &j.Title, &j.CompanyId, &j.CompanyName, &j.Location, &j.Status, &j.IsFlagged, &createdAt, &updatedAt); err != nil {
				continue
			}
			j.CreatedAt = createdAt.Format(time.RFC3339)
			j.UpdatedAt = updatedAt.Format(time.RFC3339)
			jobs = append(jobs, j)
		}
	}

	if jobs == nil {
		jobs = []domain.AdminJob{}
	}

	return jobs, total, nil
}

// HideJob hides or unhides a job
func (r *adminRepo) HideJob(ctx context.Context, jobID int64, hide bool) error {
	_, _ = r.db.Exec(ctx, `ALTER TABLE jobs ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active'`)

	status := "active"
	if hide {
		status = "hidden"
	}

	query := `UPDATE jobs SET status = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, jobID, status, time.Now())
	return err
}

// FlagJob flags or unflags a job
func (r *adminRepo) FlagJob(ctx context.Context, jobID int64, flag bool, reason string) error {
	_, _ = r.db.Exec(ctx, `ALTER TABLE jobs ADD COLUMN IF NOT EXISTS is_flagged BOOLEAN DEFAULT false`)
	_, _ = r.db.Exec(ctx, `ALTER TABLE jobs ADD COLUMN IF NOT EXISTS flag_reason TEXT`)

	query := `UPDATE jobs SET is_flagged = $2, flag_reason = $3, updated_at = $4 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, jobID, flag, reason, time.Now())
	return err
}
