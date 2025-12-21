package postgres

import (
	"context"
	"go-recruitment-backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type companyProfileRepo struct {
	db *pgxpool.Pool
}

// NewCompanyProfileRepository creates a new company profile repository
func NewCompanyProfileRepository(db *pgxpool.Pool) domain.CompanyProfileRepository {
	return &companyProfileRepo{db: db}
}

// GetByUserID retrieves a company profile by the employer's user ID
func (r *companyProfileRepo) GetByUserID(ctx context.Context, userID string) (*domain.CompanyProfile, error) {
	query := `
		SELECT id, user_id, company_name, logo_url, location, company_story, 
		       founded, founder, headquarters, employee_count, website,
		       industry, description, hide_company_details,
		       gallery_image_1, gallery_image_2, gallery_image_3,
		       created_at, updated_at
		FROM company_profiles 
		WHERE user_id = $1`

	var profile domain.CompanyProfile
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&profile.ID, &profile.UserID, &profile.CompanyName,
		&profile.LogoURL, &profile.Location, &profile.CompanyStory,
		&profile.Founded, &profile.Founder, &profile.Headquarters,
		&profile.EmployeeCount, &profile.Website,
		&profile.Industry, &profile.Description, &profile.HideCompanyDetails,
		&profile.GalleryImage1, &profile.GalleryImage2, &profile.GalleryImage3,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &profile, nil
}

// GetByID retrieves a company profile by its ID (for public page)
func (r *companyProfileRepo) GetByID(ctx context.Context, id int64) (*domain.CompanyProfile, error) {
	query := `
		SELECT id, user_id, company_name, logo_url, location, company_story, 
		       founded, founder, headquarters, employee_count, website,
		       industry, description, hide_company_details,
		       gallery_image_1, gallery_image_2, gallery_image_3,
		       created_at, updated_at
		FROM company_profiles 
		WHERE id = $1`

	var profile domain.CompanyProfile
	err := r.db.QueryRow(ctx, query, id).Scan(
		&profile.ID, &profile.UserID, &profile.CompanyName,
		&profile.LogoURL, &profile.Location, &profile.CompanyStory,
		&profile.Founded, &profile.Founder, &profile.Headquarters,
		&profile.EmployeeCount, &profile.Website,
		&profile.Industry, &profile.Description, &profile.HideCompanyDetails,
		&profile.GalleryImage1, &profile.GalleryImage2, &profile.GalleryImage3,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &profile, nil
}

// Upsert creates or updates a company profile (1 profile per user)
func (r *companyProfileRepo) Upsert(ctx context.Context, profile *domain.CompanyProfile) error {
	now := time.Now()
	profile.UpdatedAt = now

	query := `
		INSERT INTO company_profiles (
			user_id, company_name, logo_url, location, company_story,
			founded, founder, headquarters, employee_count, website,
			industry, description, hide_company_details,
			gallery_image_1, gallery_image_2, gallery_image_3,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (user_id) DO UPDATE SET
			company_name = EXCLUDED.company_name,
			logo_url = EXCLUDED.logo_url,
			location = EXCLUDED.location,
			company_story = EXCLUDED.company_story,
			founded = EXCLUDED.founded,
			founder = EXCLUDED.founder,
			headquarters = EXCLUDED.headquarters,
			employee_count = EXCLUDED.employee_count,
			website = EXCLUDED.website,
			industry = EXCLUDED.industry,
			description = EXCLUDED.description,
			hide_company_details = EXCLUDED.hide_company_details,
			gallery_image_1 = EXCLUDED.gallery_image_1,
			gallery_image_2 = EXCLUDED.gallery_image_2,
			gallery_image_3 = EXCLUDED.gallery_image_3,
			updated_at = EXCLUDED.updated_at
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		profile.UserID, profile.CompanyName, profile.LogoURL, profile.Location, profile.CompanyStory,
		profile.Founded, profile.Founder, profile.Headquarters, profile.EmployeeCount, profile.Website,
		profile.Industry, profile.Description, profile.HideCompanyDetails,
		profile.GalleryImage1, profile.GalleryImage2, profile.GalleryImage3,
		now, now,
	).Scan(&profile.ID, &profile.CreatedAt)

	return err
}
