package v1

import (
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ApplicationHandler struct {
	applicationUC domain.ApplicationUsecase
}

// NewApplicationHandler registers application routes
func NewApplicationHandler(r *gin.RouterGroup, applicationUC domain.ApplicationUsecase) {
	handler := &ApplicationHandler{applicationUC: applicationUC}

	// Candidate routes
	candidates := r.Group("/candidates")
	{
		candidates.POST("/jobs/:jobId/apply", handler.ApplyToJob)
		candidates.GET("/applications", handler.GetMyApplications)
	}

	// Employer routes
	employers := r.Group("/employers")
	{
		employers.GET("/jobs/:jobId/applications", handler.ListJobApplications)
		employers.GET("/applications/:id", handler.GetApplicationDetail)
		employers.PATCH("/applications/:id", handler.UpdateApplicationStatus)
	}
}

// ApplyToJobRequest is the request payload for applying to a job
type ApplyToJobRequest struct {
	CvURL       string `json:"cv_url" binding:"required"`
	CoverLetter string `json:"cover_letter"`
}

// ApplyToJob godoc
// @Summary      Apply to a job
// @Description  Submit an application for a job (Candidate only, must be verified)
// @Tags         applications
// @Accept       json
// @Produce      json
// @Param        jobId  path      int                true  "Job ID"
// @Param        body   body      ApplyToJobRequest  true  "Application data"
// @Success      201    {object}  response.Response{data=domain.Application}
// @Failure      400    {object}  response.Response
// @Failure      403    {object}  response.Response
// @Router       /candidates/jobs/{jobId}/apply [post]
// @Security     BearerAuth
func (h *ApplicationHandler) ApplyToJob(c *gin.Context) {
	// 1. Get user from context
	userID := c.GetString(string(domain.KeyUserID))
	role := c.GetString(string(domain.KeyUserRole))

	// Only candidates can apply
	if role != "candidate" {
		c.Error(apperror.Forbidden("Only candidates can apply to jobs"))
		return
	}

	// 2. Parse job ID
	jobIDStr := c.Param("jobId")
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid job ID"))
		return
	}

	// 3. Bind request
	var req ApplyToJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	// 4. Apply
	app, err := h.applicationUC.ApplyToJob(c, userID, jobID, req.CvURL, req.CoverLetter)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusCreated, "Application submitted successfully", app)
}

// GetMyApplications godoc
// @Summary      Get my applications
// @Description  Get all applications submitted by the current candidate
// @Tags         applications
// @Produce      json
// @Success      200  {object}  response.Response{data=[]domain.Application}
// @Failure      401  {object}  response.Response
// @Router       /candidates/applications [get]
// @Security     BearerAuth
func (h *ApplicationHandler) GetMyApplications(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))

	applications, err := h.applicationUC.GetMyApplications(c, userID)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Applications retrieved", applications)
}

// ListJobApplications godoc
// @Summary      List applications for a job
// @Description  Get all applications for a specific job (Employer only)
// @Tags         applications
// @Produce      json
// @Param        jobId  path      int  true  "Job ID"
// @Success      200    {object}  response.Response{data=[]domain.Application}
// @Failure      403    {object}  response.Response
// @Failure      404    {object}  response.Response
// @Router       /employers/jobs/{jobId}/applications [get]
// @Security     BearerAuth
func (h *ApplicationHandler) ListJobApplications(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))
	role := c.GetString(string(domain.KeyUserRole))

	// Only employers can view applications
	if role != "employer" && role != "admin" {
		c.Error(apperror.Forbidden("Only employers can view job applications"))
		return
	}

	// Parse job ID
	jobIDStr := c.Param("jobId")
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid job ID"))
		return
	}

	applications, err := h.applicationUC.ListByJobID(c, userID, jobID)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Applications retrieved", applications)
}

// GetApplicationDetail godoc
// @Summary      Get application detail
// @Description  Get full application details including candidate profile (Employer only)
// @Tags         applications
// @Produce      json
// @Param        id  path      int  true  "Application ID"
// @Success      200 {object}  response.Response{data=domain.ApplicationDetailResponse}
// @Failure      403 {object}  response.Response
// @Failure      404 {object}  response.Response
// @Router       /employers/applications/{id} [get]
// @Security     BearerAuth
func (h *ApplicationHandler) GetApplicationDetail(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))
	role := c.GetString(string(domain.KeyUserRole))

	if role != "employer" && role != "admin" {
		c.Error(apperror.Forbidden("Only employers can view application details"))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid application ID"))
		return
	}

	detail, err := h.applicationUC.GetApplicationDetail(c, userID, id)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Application detail retrieved", detail)
}

// UpdateStatusRequest is the request payload for updating application status
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=reviewed accepted rejected"`
}

// UpdateApplicationStatus godoc
// @Summary      Update application status
// @Description  Update the status of an application (Employer only)
// @Tags         applications
// @Accept       json
// @Produce      json
// @Param        id    path      int                  true  "Application ID"
// @Param        body  body      UpdateStatusRequest  true  "Status update"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      403   {object}  response.Response
// @Failure      404   {object}  response.Response
// @Router       /employers/applications/{id} [patch]
// @Security     BearerAuth
func (h *ApplicationHandler) UpdateApplicationStatus(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))
	role := c.GetString(string(domain.KeyUserRole))

	if role != "employer" && role != "admin" {
		c.Error(apperror.Forbidden("Only employers can update application status"))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid application ID"))
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	if err := h.applicationUC.UpdateApplicationStatus(c, userID, id, req.Status); err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Application status updated", nil)
}
