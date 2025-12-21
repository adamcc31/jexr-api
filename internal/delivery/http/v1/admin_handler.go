package v1

import (
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminUC domain.AdminUsecase
}

func NewAdminHandler(protected *gin.RouterGroup, adminUC domain.AdminUsecase) {
	handler := &AdminHandler{adminUC: adminUC}

	admin := protected.Group("/admin")
	{
		// Dashboard stats
		admin.GET("/stats", handler.GetStats)

		// User management
		admin.GET("/users", handler.ListUsers)
		admin.POST("/users", handler.CreateUser)
		admin.PUT("/users/:id", handler.UpdateUser)
		admin.DELETE("/users/:id", handler.DeleteUser)
		admin.PATCH("/users/:id/disable", handler.DisableUser)

		// Company verification
		admin.GET("/companies", handler.ListCompanies)
		admin.PATCH("/companies/:id/verify", handler.VerifyCompany)

		// Job moderation
		admin.GET("/jobs", handler.ListJobs)
		admin.PATCH("/jobs/:id/hide", handler.HideJob)
		admin.PATCH("/jobs/:id/flag", handler.FlagJob)
	}
}

// GetStats godoc
// @Summary      Get admin dashboard statistics
// @Description  Returns counts for users, companies, jobs, and applications
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /admin/stats [get]
func (h *AdminHandler) GetStats(c *gin.Context) {
	stats, err := h.adminUC.GetStats(c)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusOK, "Dashboard statistics", stats)
}

// ListUsers godoc
// @Summary      List all users
// @Description  Returns paginated list of users with optional role filter
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        role     query     string  false  "Filter by role (admin, employer, candidate)"
// @Param        page     query     int     false  "Page number"
// @Param        pageSize query     int     false  "Items per page"
// @Success      200      {object}  response.Response
// @Failure      403      {object}  response.Response
// @Router       /admin/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
	role := c.Query("role")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	result, err := h.adminUC.ListUsers(c, role, page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusOK, "Users list", result)
}

// DisableUser godoc
// @Summary      Disable or enable a user
// @Description  Toggles user disabled status
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string  true   "User ID"
// @Param        body     body      object  true   "{ disable: bool }"
// @Success      200      {object}  response.Response
// @Failure      403      {object}  response.Response
// @Router       /admin/users/{id}/disable [patch]
func (h *AdminHandler) DisableUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.Error(apperror.BadRequest("User ID is required"))
		return
	}

	var body struct {
		Disable bool `json:"disable"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(apperror.BadRequest("Invalid request body"))
		return
	}

	user, err := h.adminUC.DisableUser(c, userID, body.Disable)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusOK, "User updated", user)
}

// CreateUser godoc
// @Summary      Create a new user
// @Description  Creates a new user record in the database
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body     body      domain.CreateUserRequest  true   "User details"
// @Success      201      {object}  response.Response
// @Failure      403      {object}  response.Response
// @Router       /admin/users [post]
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	user, err := h.adminUC.CreateUser(c, req)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusCreated, "User created", user)
}

// UpdateUser godoc
// @Summary      Update a user
// @Description  Updates an existing user record
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string  true   "User ID"
// @Param        body     body      domain.UpdateUserRequest  true   "User details"
// @Success      200      {object}  response.Response
// @Failure      403      {object}  response.Response
// @Router       /admin/users/{id} [put]
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.Error(apperror.BadRequest("User ID is required"))
		return
	}

	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	user, err := h.adminUC.UpdateUser(c, userID, req)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusOK, "User updated", user)
}

// DeleteUser godoc
// @Summary      Delete a user
// @Description  Permanently deletes a user record
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string  true   "User ID"
// @Success      200      {object}  response.Response
// @Failure      403      {object}  response.Response
// @Router       /admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.Error(apperror.BadRequest("User ID is required"))
		return
	}

	if err := h.adminUC.DeleteUser(c, userID); err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusOK, "User deleted", nil)
}

// ListCompanies godoc
// @Summary      List all companies
// @Description  Returns paginated list of companies with optional status filter
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        verificationStatus  query  string  false  "Filter by status (pending, verified, rejected)"
// @Param        page                query  int     false  "Page number"
// @Param        pageSize            query  int     false  "Items per page"
// @Success      200                 {object}  response.Response
// @Failure      403                 {object}  response.Response
// @Router       /admin/companies [get]
func (h *AdminHandler) ListCompanies(c *gin.Context) {
	status := c.Query("verificationStatus")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	result, err := h.adminUC.ListCompanies(c, status, page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusOK, "Companies list", result)
}

// VerifyCompany godoc
// @Summary      Verify a company
// @Description  Approves or rejects a company verification
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int     true   "Company ID"
// @Param        body  body      object  true   "{ action: 'approve'|'reject', reason?: string }"
// @Success      200   {object}  response.Response
// @Failure      403   {object}  response.Response
// @Router       /admin/companies/{id}/verify [patch]
func (h *AdminHandler) VerifyCompany(c *gin.Context) {
	companyIDStr := c.Param("id")
	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid company ID"))
		return
	}

	var body struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(apperror.BadRequest("Invalid request body"))
		return
	}

	company, err := h.adminUC.VerifyCompany(c, companyID, body.Action, body.Reason)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusOK, "Company verified", company)
}

// ListJobs godoc
// @Summary      List all jobs for moderation
// @Description  Returns paginated list of jobs with optional status filter
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        status    query  string  false  "Filter by status (active, hidden, flagged)"
// @Param        page      query  int     false  "Page number"
// @Param        pageSize  query  int     false  "Items per page"
// @Success      200       {object}  response.Response
// @Failure      403       {object}  response.Response
// @Router       /admin/jobs [get]
func (h *AdminHandler) ListJobs(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	result, err := h.adminUC.ListJobs(c, status, page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusOK, "Jobs list", result)
}

// HideJob godoc
// @Summary      Hide or unhide a job
// @Description  Toggles job visibility
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int     true   "Job ID"
// @Param        body  body      object  true   "{ hide: bool, reason?: string }"
// @Success      200   {object}  response.Response
// @Failure      403   {object}  response.Response
// @Router       /admin/jobs/{id}/hide [patch]
func (h *AdminHandler) HideJob(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid job ID"))
		return
	}

	var body struct {
		Hide   bool   `json:"hide"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(apperror.BadRequest("Invalid request body"))
		return
	}

	job, err := h.adminUC.HideJob(c, jobID, body.Hide)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusOK, "Job updated", job)
}

// FlagJob godoc
// @Summary      Flag or unflag a job
// @Description  Marks a job as suspicious
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int     true   "Job ID"
// @Param        body  body      object  true   "{ flag: bool, reason?: string }"
// @Success      200   {object}  response.Response
// @Failure      403   {object}  response.Response
// @Router       /admin/jobs/{id}/flag [patch]
func (h *AdminHandler) FlagJob(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		c.Error(apperror.BadRequest("Invalid job ID"))
		return
	}

	var body struct {
		Flag   bool   `json:"flag"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(apperror.BadRequest("Invalid request body"))
		return
	}

	job, err := h.adminUC.FlagJob(c, jobID, body.Flag, body.Reason)
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusOK, "Job flagged", job)
}
