package v1

import (
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CompanyProfileHandler struct {
	profileUC      domain.CompanyProfileUsecase
	verificationUC domain.VerificationUsecase
}

// NewCompanyProfileHandler registers company profile routes
func NewCompanyProfileHandler(
	public *gin.RouterGroup,
	protected *gin.RouterGroup,
	profileUC domain.CompanyProfileUsecase,
	verificationUC domain.VerificationUsecase,
) {
	handler := &CompanyProfileHandler{
		profileUC:      profileUC,
		verificationUC: verificationUC,
	}

	// Public routes
	public.GET("/companies/:id", handler.GetPublicProfile)

	// Protected employer routes
	employers := protected.Group("/employers")
	{
		employers.GET("/company-profile", handler.GetOwnProfile)
		employers.PUT("/company-profile", handler.UpdateProfile)
	}
}

// CompanyProfileRequest matches the frontend input for company profile updates
type CompanyProfileRequest struct {
	CompanyName        string  `json:"company_name" binding:"required"`
	LogoURL            *string `json:"logo_url"`
	Location           *string `json:"location"`
	CompanyStory       *string `json:"company_story"`
	Founded            *string `json:"founded"`
	Founder            *string `json:"founder"`
	Headquarters       *string `json:"headquarters"`
	EmployeeCount      *string `json:"employee_count"`
	Website            *string `json:"website"`
	Industry           *string `json:"industry"`
	Description        *string `json:"description"`
	HideCompanyDetails bool    `json:"hide_company_details"`
	GalleryImage1      *string `json:"gallery_image_1"`
	GalleryImage2      *string `json:"gallery_image_2"`
	GalleryImage3      *string `json:"gallery_image_3"`
}

// GetOwnProfile godoc
// @Summary Get employer's own company profile
// @Description Retrieve the employer's company profile for editing
// @Tags Company Profile
// @Produce json
// @Success 200 {object} domain.CompanyProfile
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /employers/company-profile [get]
// @Security BearerAuth
func (h *CompanyProfileHandler) GetOwnProfile(c *gin.Context) {
	// Check role
	role := c.GetString(string(domain.KeyUserRole))
	if role != "employer" && role != "admin" {
		c.Error(apperror.Forbidden("Only employers can access company profiles"))
		return
	}

	userID := c.GetString(string(domain.KeyUserID))
	if userID == "" {
		c.Error(apperror.Unauthorized("User not authenticated"))
		return
	}

	profile, err := h.profileUC.GetEmployerProfile(c, userID)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Company profile retrieved", profile)
}

// UpdateProfile godoc
// @Summary Create or update company profile
// @Description Create or update the employer's company profile
// @Tags Company Profile
// @Accept json
// @Produce json
// @Param request body UpdateProfileRequest true "Profile data"
// @Success 200 {object} domain.CompanyProfile
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /employers/company-profile [put]
// @Security BearerAuth
func (h *CompanyProfileHandler) UpdateProfile(c *gin.Context) {
	// Check role
	role := c.GetString(string(domain.KeyUserRole))
	if role != "employer" && role != "admin" {
		c.Error(apperror.Forbidden("Only employers can update company profiles"))
		return
	}

	userID := c.GetString(string(domain.KeyUserID))
	if userID == "" {
		c.Error(apperror.Unauthorized("User not authenticated"))
		return
	}

	var req CompanyProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	profile := &domain.CompanyProfile{
		CompanyName:        req.CompanyName,
		LogoURL:            req.LogoURL,
		Location:           req.Location,
		CompanyStory:       req.CompanyStory,
		Founded:            req.Founded,
		Founder:            req.Founder,
		Headquarters:       req.Headquarters,
		EmployeeCount:      req.EmployeeCount,
		Website:            req.Website,
		Industry:           req.Industry,
		Description:        req.Description,
		HideCompanyDetails: req.HideCompanyDetails,
		GalleryImage1:      req.GalleryImage1,
		GalleryImage2:      req.GalleryImage2,
		GalleryImage3:      req.GalleryImage3,
	}

	if err := h.profileUC.UpdateEmployerProfile(c, userID, profile); err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Company profile updated", profile)
}

// GetPublicProfile godoc
// @Summary Get public company profile
// @Description Retrieve a company profile for public viewing with visibility rules
// @Tags Company Profile
// @Produce json
// @Param id path int true "Company ID"
// @Success 200 {object} domain.PublicCompanyProfile
// @Failure 404 {object} response.Response
// @Router /companies/{id} [get]
func (h *CompanyProfileHandler) GetPublicProfile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid company ID"))
		return
	}

	// Build viewer info from context (may be nil for unauthenticated)
	viewer := &domain.ViewerInfo{
		IsAuthenticated: false,
	}

	userID := c.GetString(string(domain.KeyUserID))
	if userID != "" {
		viewer.IsAuthenticated = true
		viewer.Role = c.GetString(string(domain.KeyUserRole))

		// Get verification status for candidates
		if viewer.Role == "candidate" {
			verification, err := h.verificationUC.GetVerificationStatus(c, userID)
			if err == nil && verification != nil && verification.Verification != nil {
				viewer.VerificationStatus = verification.Verification.Status
			}
		}
	}

	profile, err := h.profileUC.GetPublicProfile(c, id, viewer)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Company profile", profile)
}
