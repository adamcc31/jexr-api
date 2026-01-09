package security

import (
	"net/http"
	"strconv"
	"time"

	"go-recruitment-backend/internal/delivery/http/middleware"
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/security"

	"github.com/gin-gonic/gin"
)

// SecurityDashboardHandler handles HTTP requests for the security dashboard
type SecurityDashboardHandler struct {
	usecase     domain.SecurityDashboardUsecase
	authService *security.SecurityAuthService
}

// NewSecurityDashboardHandler creates a new security dashboard handler
func NewSecurityDashboardHandler(usecase domain.SecurityDashboardUsecase, authService *security.SecurityAuthService) *SecurityDashboardHandler {
	return &SecurityDashboardHandler{
		usecase:     usecase,
		authService: authService,
	}
}

// RegisterRoutes registers security dashboard routes on a separate router
// The path prefix should be non-discoverable (e.g., /v1/sec-ops-{random-hash})
// IMPORTANT: Hidden path is a NOISE LAYER only, NOT a security control
// Real security: IP Allowlist → MFA Auth → RBAC → Audit Log
func (h *SecurityDashboardHandler) RegisterRoutes(router *gin.RouterGroup) {
	// All routes require IP allowlist + auth
	router.Use(middleware.SecurityIPAllowlistMiddleware(h.authService))
	router.Use(middleware.SecurityAuditMiddleware())

	// Auth routes (no session required)
	auth := router.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/verify-totp", h.VerifyTOTP)
		auth.POST("/setup-totp", h.SetupTOTP)          // Generate TOTP secret + QR
		auth.POST("/confirm-totp", h.ConfirmTOTPSetup) // Verify and enable TOTP
	}

	// Protected routes (session required)
	protected := router.Group("")
	protected.Use(middleware.SecurityAuthMiddleware(h.authService))
	protected.Use(middleware.ReadOnlyModeMiddleware())
	{
		// Read-only routes (OBSERVER+)
		protected.GET("/auth/me", h.GetCurrentUser) // Get current authenticated user
		protected.GET("/stats", h.GetStats)
		protected.GET("/events", h.ListEvents)
		protected.GET("/heatmap", h.GetHeatmap)
		protected.GET("/timeline", h.GetTimeline)
		protected.GET("/integrity/status", h.GetIntegrityStatus)
		protected.POST("/logout", h.Logout)

		// Analyst routes (ANALYST+)
		analyst := protected.Group("")
		analyst.Use(middleware.SecurityRoleMiddleware(security.RoleSecurityAnalyst, security.RoleSecurityAdmin))
		{
			analyst.POST("/export/request", h.RequestExport)
			analyst.GET("/export/:id", h.GetExportRequest)
			analyst.GET("/export/:id/download", h.DownloadExport)
		}

		// Admin routes (ADMIN only)
		admin := protected.Group("")
		admin.Use(middleware.SecurityRoleMiddleware(security.RoleSecurityAdmin))
		{
			admin.GET("/export/pending", h.ListPendingExports)
			admin.POST("/export/:id/approve", h.ApproveExport)
			admin.POST("/export/:id/reject", h.RejectExport)
			admin.POST("/break-glass/activate", h.ActivateBreakGlass)
			admin.GET("/break-glass/status", h.GetBreakGlassStatus)
			admin.POST("/break-glass/revoke", h.RevokeBreakGlass)
			admin.POST("/integrity/verify", h.VerifyIntegrity)
		}
	}
}

// === Auth Handlers ===

// GetCurrentUser returns the currently authenticated user's info
// Used by frontend to verify session and populate user state
func (h *SecurityDashboardHandler) GetCurrentUser(c *gin.Context) {
	session, exists := c.Get("security_session")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "No active session", nil)
		return
	}

	s, ok := session.(*security.SecuritySession)
	if !ok {
		response.Error(c, http.StatusInternalServerError, "Invalid session type", nil)
		return
	}

	user, exists := c.Get("security_user")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not found", nil)
		return
	}

	u, ok := user.(*security.SecurityUser)
	if !ok {
		response.Error(c, http.StatusInternalServerError, "Invalid user type", nil)
		return
	}

	response.Success(c, http.StatusOK, "User retrieved", gin.H{
		"user": gin.H{
			"id":       u.ID,
			"username": u.Username,
			"email":    u.Email,
			"role":     u.Role,
		},
		"sessionId": s.ID,
		"expiresAt": s.ExpiresAt,
	})
}

// Login handles initial username/password authentication
func (h *SecurityDashboardHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	ip := c.GetString("client_ip")
	userAgent := c.GetHeader("User-Agent")

	user, err := h.authService.Authenticate(c.Request.Context(), req.Username, req.Password, ip, userAgent)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	if !user.TOTPEnabled {
		response.Error(c, http.StatusForbidden, "MFA not configured - contact administrator", nil)
		return
	}

	// Return partial auth state - TOTP required
	response.Success(c, http.StatusOK, "TOTP verification required", gin.H{
		"userId":       user.ID,
		"username":     user.Username,
		"totpRequired": true,
	})
}

// VerifyTOTP completes MFA and creates session
func (h *SecurityDashboardHandler) VerifyTOTP(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		TOTPCode string `json:"totpCode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	ip := c.GetString("client_ip")
	userAgent := c.GetHeader("User-Agent")

	// Re-authenticate
	user, err := h.authService.Authenticate(c.Request.Context(), req.Username, req.Password, ip, userAgent)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	// Validate TOTP
	valid, err := h.authService.ValidateTOTP(c.Request.Context(), user, req.TOTPCode)
	if err != nil || !valid {
		response.Error(c, http.StatusUnauthorized, "Invalid TOTP code", nil)
		return
	}

	// Create session
	session, token, err := h.authService.CreateSession(c.Request.Context(), user.ID, ip, userAgent)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create session", nil)
		return
	}

	// Set session cookie
	c.SetCookie("security_session", token, int(session.ExpiresAt.Sub(time.Now()).Seconds()), "/", "", true, true)

	response.Success(c, http.StatusOK, "Authentication successful", gin.H{
		"sessionId": session.ID,
		"expiresAt": session.ExpiresAt,
		"role":      user.Role,
	})
}

// Logout revokes the current session
func (h *SecurityDashboardHandler) Logout(c *gin.Context) {
	session, exists := c.Get("security_session")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "No active session", nil)
		return
	}

	s, ok := session.(*security.SecuritySession)
	if !ok {
		response.Error(c, http.StatusInternalServerError, "Invalid session type", nil)
		return
	}

	if err := h.authService.RevokeSession(c.Request.Context(), s.ID, "user_logout"); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to revoke session", nil)
		return
	}

	// Clear cookie
	c.SetCookie("security_session", "", -1, "/", "", true, true)

	response.Success(c, http.StatusOK, "Logged out successfully", nil)
}

// SetupTOTP generates a new TOTP secret for first-time setup
// This requires valid credentials but allows users without TOTP to enroll
func (h *SecurityDashboardHandler) SetupTOTP(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	ip := c.GetString("client_ip")
	userAgent := c.GetHeader("User-Agent")

	// Authenticate without TOTP requirement
	user, err := h.authService.Authenticate(c.Request.Context(), req.Username, req.Password, ip, userAgent)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	// Check if TOTP already enabled
	if user.TOTPEnabled {
		response.Error(c, http.StatusBadRequest, "TOTP is already configured. Contact administrator to reset.", nil)
		return
	}

	// Generate new TOTP secret
	secret, qrCodeURL, err := h.authService.GenerateTOTPSecret(user.Username)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate TOTP secret", nil)
		return
	}

	// Store temp secret in DB for confirmation
	if err := h.authService.StoreTempTOTPSecret(c.Request.Context(), user.ID, secret); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to store TOTP secret", nil)
		return
	}

	response.Success(c, http.StatusOK, "TOTP setup initiated", gin.H{
		"secret":     secret,
		"qrCodeUrl":  qrCodeURL, // otpauth:// URL for QR code
		"userId":     user.ID,
		"username":   user.Username,
		"issuer":     "J-Expert Security",
		"setupGuide": "Scan QR code with Google Authenticator, Authy, or 1Password. Then confirm with a code.",
	})
}

// ConfirmTOTPSetup verifies the TOTP code and enables MFA for the user
func (h *SecurityDashboardHandler) ConfirmTOTPSetup(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		TOTPCode string `json:"totpCode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	ip := c.GetString("client_ip")
	userAgent := c.GetHeader("User-Agent")

	// Authenticate
	user, err := h.authService.Authenticate(c.Request.Context(), req.Username, req.Password, ip, userAgent)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	// Get temp secret and verify
	tempSecret, err := h.authService.GetTempTOTPSecret(c.Request.Context(), user.ID)
	if err != nil || tempSecret == "" {
		response.Error(c, http.StatusBadRequest, "No pending TOTP setup. Call /setup-totp first.", nil)
		return
	}

	// Verify the TOTP code and enable
	if err := h.authService.EnableTOTP(c.Request.Context(), user.ID, tempSecret, req.TOTPCode); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "TOTP berhasil diaktifkan! Silakan login kembali.", gin.H{
		"enabled":  true,
		"username": user.Username,
	})
}

// === Dashboard Handlers ===

// GetStats returns dashboard statistics
func (h *SecurityDashboardHandler) GetStats(c *gin.Context) {
	stats, err := h.usecase.GetStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get stats", nil)
		return
	}
	response.Success(c, http.StatusOK, "Stats retrieved", stats)
}

// ListEvents returns filtered security events
func (h *SecurityDashboardHandler) ListEvents(c *gin.Context) {
	filter := domain.SecurityEventFilter{
		Limit:  50,
		Offset: 0,
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			filter.Limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			filter.Offset = o
		}
	}
	if startStr := c.Query("startTime"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = &t
		}
	}
	if endStr := c.Query("endTime"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = &t
		}
	}
	if ip := c.Query("ip"); ip != "" {
		filter.SearchIP = ip
	}
	if user := c.Query("user"); user != "" {
		filter.SearchUser = user
	}

	events, total, err := h.usecase.ListEvents(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list events", nil)
		return
	}

	response.Success(c, http.StatusOK, "Events retrieved", gin.H{
		"events": events,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetHeatmap returns auth failure heatmap data
func (h *SecurityDashboardHandler) GetHeatmap(c *gin.Context) {
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	if startStr := c.Query("startTime"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = t
		}
	}
	if endStr := c.Query("endTime"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = t
		}
	}

	heatmap, err := h.usecase.GetAuthFailureHeatmap(c.Request.Context(), startTime, endTime)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get heatmap", nil)
		return
	}

	response.Success(c, http.StatusOK, "Heatmap retrieved", heatmap)
}

// GetTimeline returns privileged action timeline
func (h *SecurityDashboardHandler) GetTimeline(c *gin.Context) {
	page := 1
	pageSize := 50

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if sizeStr := c.Query("pageSize"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}

	actions, total, err := h.usecase.GetPrivilegedActionTimeline(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get timeline", nil)
		return
	}

	response.Success(c, http.StatusOK, "Timeline retrieved", gin.H{
		"actions":  actions,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// === Export Handlers ===

// RequestExport creates a new export request
func (h *SecurityDashboardHandler) RequestExport(c *gin.Context) {
	var req domain.CreateExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	user := c.MustGet("security_user").(*security.SecurityUser)

	export, err := h.usecase.RequestExport(c.Request.Context(), user.ID, req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create export request", nil)
		return
	}

	response.Success(c, http.StatusCreated, "Export request created", export)
}

// GetExportRequest returns export request details
func (h *SecurityDashboardHandler) GetExportRequest(c *gin.Context) {
	exportID := c.Param("id")

	// Implementation would fetch export details
	response.Success(c, http.StatusOK, "Export request retrieved", gin.H{"id": exportID})
}

// ListPendingExports lists pending export requests (admin only)
func (h *SecurityDashboardHandler) ListPendingExports(c *gin.Context) {
	// Implementation would list pending exports
	response.Success(c, http.StatusOK, "Pending exports listed", gin.H{"exports": []interface{}{}})
}

// ApproveExport approves an export request (admin only)
func (h *SecurityDashboardHandler) ApproveExport(c *gin.Context) {
	exportID := c.Param("id")
	user := c.MustGet("security_user").(*security.SecurityUser)

	if err := h.usecase.ApproveExport(c.Request.Context(), exportID, user.ID); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to approve export", nil)
		return
	}

	response.Success(c, http.StatusOK, "Export approved", gin.H{"id": exportID})
}

// RejectExport rejects an export request (admin only)
func (h *SecurityDashboardHandler) RejectExport(c *gin.Context) {
	exportID := c.Param("id")
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Rejection reason required", nil)
		return
	}

	user := c.MustGet("security_user").(*security.SecurityUser)

	if err := h.usecase.RejectExport(c.Request.Context(), exportID, user.ID, req.Reason); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to reject export", nil)
		return
	}

	response.Success(c, http.StatusOK, "Export rejected", gin.H{"id": exportID})
}

// DownloadExport streams the approved export data
func (h *SecurityDashboardHandler) DownloadExport(c *gin.Context) {
	exportID := c.Param("id")
	user := c.MustGet("security_user").(*security.SecurityUser)

	events, err := h.usecase.GetExportData(c.Request.Context(), exportID, user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get export data", nil)
		return
	}

	// Return as JSON for now - could be CSV/Excel
	c.JSON(http.StatusOK, gin.H{
		"exportId": exportID,
		"events":   events,
		"count":    len(events),
	})
}

// === Break-Glass Handlers ===

// ActivateBreakGlass activates a time-limited DEVELOPER_ROOT session
func (h *SecurityDashboardHandler) ActivateBreakGlass(c *gin.Context) {
	var req domain.BreakGlassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	user := c.MustGet("security_user").(*security.SecurityUser)

	result, err := h.usecase.ActivateBreakGlass(c.Request.Context(), user.ID, req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Break-glass activated", result)
}

// GetBreakGlassStatus returns current break-glass session status
func (h *SecurityDashboardHandler) GetBreakGlassStatus(c *gin.Context) {
	user := c.MustGet("security_user").(*security.SecurityUser)

	result, err := h.usecase.GetActiveBreakGlass(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get status", nil)
		return
	}

	if result == nil {
		response.Success(c, http.StatusOK, "No active break-glass session", gin.H{"active": false})
		return
	}

	response.Success(c, http.StatusOK, "Break-glass status", gin.H{
		"active":  true,
		"session": result,
	})
}

// RevokeBreakGlass revokes an active break-glass session
func (h *SecurityDashboardHandler) RevokeBreakGlass(c *gin.Context) {
	var req struct {
		SessionID string `json:"sessionId" binding:"required"`
		Reason    string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	if err := h.usecase.RevokeBreakGlass(c.Request.Context(), req.SessionID, req.Reason); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to revoke", nil)
		return
	}

	response.Success(c, http.StatusOK, "Break-glass revoked", nil)
}

// === Integrity Handlers ===

// GetIntegrityStatus returns current log integrity status
func (h *SecurityDashboardHandler) GetIntegrityStatus(c *gin.Context) {
	status, lastAnchor, err := h.usecase.GetIntegrityStatus(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get integrity status", nil)
		return
	}

	response.Success(c, http.StatusOK, "Integrity status", gin.H{
		"status":     status,
		"lastAnchor": lastAnchor,
	})
}

// VerifyIntegrity performs a full integrity check (admin only)
func (h *SecurityDashboardHandler) VerifyIntegrity(c *gin.Context) {
	var req domain.IntegrityVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid start date", nil)
		return
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid end date", nil)
		return
	}

	report, err := h.usecase.VerifyIntegrity(c.Request.Context(), startDate, endDate)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Verification failed", nil)
		return
	}

	response.Success(c, http.StatusOK, "Integrity verification complete", report)
}
