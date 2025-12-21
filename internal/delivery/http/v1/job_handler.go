package v1

import (
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type JobHandler struct {
	jobUC domain.JobUsecase
}

func NewJobHandler(public *gin.RouterGroup, protected *gin.RouterGroup, jobUC domain.JobUsecase) {
	handler := &JobHandler{jobUC: jobUC}

	// PUBLIC routes - no authentication required
	// These endpoints only return active jobs (server-side enforced)
	publicJobs := public.Group("/jobs")
	{
		publicJobs.GET("/public", handler.PublicList)           // List active jobs only
		publicJobs.GET("/public/:id", handler.PublicGetDetails) // Get active job details
	}

	// PROTECTED routes - authentication required
	protectedJobs := protected.Group("/jobs")
	{
		protectedJobs.GET("", handler.List) // Full job list for authenticated users
		protectedJobs.GET("/:id", handler.GetDetails)
		protectedJobs.POST("", handler.Create)
		protectedJobs.PUT("/:id", handler.Update)
		protectedJobs.DELETE("/:id", handler.Delete)
	}

	// Employer-specific job routes (only shows employer's own jobs)
	employers := protected.Group("/employers")
	{
		employers.GET("/jobs", handler.ListByEmployer)
	}
}

type CreateJobRequest struct {
	Title           string  `json:"title" binding:"required"`
	Description     string  `json:"description" binding:"required"`
	SalaryMin       float64 `json:"salary_min" binding:"required,gt=0"`
	SalaryMax       float64 `json:"salary_max" binding:"required,gt=0,gtefield=SalaryMin"`
	Location        string  `json:"location" binding:"required"`
	EmploymentType  string  `json:"employment_type"`
	JobType         string  `json:"job_type"`
	ExperienceLevel string  `json:"experience_level"`
	Qualifications  string  `json:"qualifications"`
}

type UpdateJobRequest struct {
	Title           string  `json:"title" binding:"required"`
	Description     string  `json:"description" binding:"required"`
	SalaryMin       float64 `json:"salary_min" binding:"required,gt=0"`
	SalaryMax       float64 `json:"salary_max" binding:"required,gt=0,gtefield=SalaryMin"`
	Location        string  `json:"location" binding:"required"`
	EmploymentType  string  `json:"employment_type"`
	JobType         string  `json:"job_type"`
	ExperienceLevel string  `json:"experience_level"`
	Qualifications  string  `json:"qualifications"`
}

// CreateJob godoc
// @Summary      Create a new job
// @Description  Create a new job posting (Employer only)
// @Tags         jobs
// @Accept       json
// @Produce      json
// @Param        job  body      CreateJobRequest  true  "Job JSON"
// @Success      201  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /jobs [post]
// @Security     BearerAuth
func (h *JobHandler) Create(c *gin.Context) {
	// 1. Role Check
	role := c.GetString(string(domain.KeyUserRole))
	if role != "employer" && role != "admin" {
		c.Error(apperror.Forbidden("Only employers or admins can create jobs"))
		return
	}

	// 2. Bind JSON
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	// 3. Get User ID from context (AuthMiddleware)
	userID := c.GetString(string(domain.KeyUserID))

	// 4. Helper to convert empty string to nil pointer
	toPtr := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}

	job := &domain.Job{
		Title:           req.Title,
		Description:     req.Description,
		SalaryMin:       req.SalaryMin,
		SalaryMax:       req.SalaryMax,
		Location:        req.Location,
		EmploymentType:  toPtr(req.EmploymentType),
		JobType:         toPtr(req.JobType),
		ExperienceLevel: toPtr(req.ExperienceLevel),
		Qualifications:  toPtr(req.Qualifications),
		CompanyStatus:   "active",
	}

	if err := h.jobUC.CreateJob(c, userID, job); err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusCreated, "Job created", job)
}

// PublicListJobs godoc
// @Summary      List active jobs (public)
// @Description  Get a list of active jobs for public access (no auth required)
// @Tags         jobs
// @Produce      json
// @Param        page       query     int  false  "Page number"
// @Param        page_size  query     int  false  "Page size"
// @Success      200        {object}  response.Response
// @Router       /jobs/public [get]
func (h *JobHandler) PublicList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// SECURITY: Always return only active jobs - no client-side bypass possible
	jobs, total, err := h.jobUC.ListPublicActiveJobs(c, page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Public job list", gin.H{
		"jobs":      jobs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// PublicGetDetails godoc
// @Summary      Get active job details (public)
// @Description  Get detailed info of an active job (no auth required)
// @Tags         jobs
// @Produce      json
// @Param        id   path      int  true  "Job ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /jobs/public/{id} [get]
func (h *JobHandler) PublicGetDetails(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid ID format"))
		return
	}

	// Return job with company profile data
	job, err := h.jobUC.GetJobDetailsWithCompany(c, id)
	if err != nil {
		c.Error(err)
		return
	}

	// SECURITY: Only return active jobs via public endpoint
	if job.CompanyStatus != "active" {
		c.Error(apperror.NotFound("Job not found"))
		return
	}

	response.Success(c, http.StatusOK, "Job details", job)
}

// ListJobs godoc
// @Summary      List jobs
// @Description  Get a list of jobs with pagination and company info
// @Tags         jobs
// @Produce      json
// @Param        page       query     int  false  "Page number"
// @Param        page_size  query     int  false  "Page size"
// @Success      200        {object}  response.Response
// @Router       /jobs [get]
// @Security     BearerAuth
func (h *JobHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// Return jobs with company profile data for public/candidate access
	jobs, total, err := h.jobUC.ListJobsWithCompany(c, page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Job list", gin.H{
		"jobs":      jobs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ListByEmployer godoc
// @Summary      List employer's own jobs
// @Description  Get a list of jobs belonging to the logged-in employer only
// @Tags         employers
// @Produce      json
// @Param        page       query     int  false  "Page number"
// @Param        page_size  query     int  false  "Page size"
// @Success      200        {object}  response.Response
// @Failure      401        {object}  response.Response
// @Failure      403        {object}  response.Response
// @Router       /employers/jobs [get]
// @Security     BearerAuth
func (h *JobHandler) ListByEmployer(c *gin.Context) {
	// Check role - only employers can access
	role := c.GetString(string(domain.KeyUserRole))
	if role != "employer" && role != "admin" {
		c.Error(apperror.Forbidden("Only employers can access their job list"))
		return
	}

	userID := c.GetString(string(domain.KeyUserID))
	if userID == "" {
		c.Error(apperror.Unauthorized("User not authenticated"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	jobs, total, err := h.jobUC.ListJobsByEmployer(c, userID, page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Employer job list", gin.H{
		"jobs":      jobs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetJobDetails godoc
// @Summary      Get job details
// @Description  Get detailed info of a job with company profile
// @Tags         jobs
// @Produce      json
// @Param        id   path      int  true  "Job ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /jobs/{id} [get]
// @Security     BearerAuth
func (h *JobHandler) GetDetails(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid ID format"))
		return
	}

	// Return job with company profile data
	job, err := h.jobUC.GetJobDetailsWithCompany(c, id)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Job details", job)
}

// DeleteJob godoc
// @Summary      Delete a job
// @Description  Permanently delete a job posting
// @Tags         jobs
// @Produce      json
// @Param        id   path      int  true  "Job ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /jobs/{id} [delete]
// @Security     BearerAuth
func (h *JobHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid ID format"))
		return
	}

	err = h.jobUC.DeleteJob(c, id)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Job deleted successfully", nil)
}

// UpdateJob godoc
// @Summary      Update a job
// @Description  Update an existing job posting
// @Tags         jobs
// @Accept       json
// @Produce      json
// @Param        id   path      int              true  "Job ID"
// @Param        job  body      UpdateJobRequest true  "Job JSON"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /jobs/{id} [put]
// @Security     BearerAuth
func (h *JobHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid ID format"))
		return
	}

	var req UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	job := &domain.Job{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		SalaryMin:   req.SalaryMin,
		SalaryMax:   req.SalaryMax,
		Location:    req.Location,
	}

	// Set optional fields (convert empty to nil)
	if req.EmploymentType != "" {
		job.EmploymentType = &req.EmploymentType
	}
	if req.JobType != "" {
		job.JobType = &req.JobType
	}
	if req.ExperienceLevel != "" {
		job.ExperienceLevel = &req.ExperienceLevel
	}
	if req.Qualifications != "" {
		job.Qualifications = &req.Qualifications
	}

	err = h.jobUC.UpdateJob(c, job)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Job updated successfully", job)
}
