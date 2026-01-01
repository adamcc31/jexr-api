package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-recruitment-backend/config"
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authUC       domain.AuthUsecase
	onboardingUC domain.OnboardingUsecase
	config       *config.Config
}

func NewAuthHandler(public *gin.RouterGroup, protected *gin.RouterGroup, authUC domain.AuthUsecase, onboardingUC domain.OnboardingUsecase, paramsConfig *config.Config) {
	handler := &AuthHandler{
		authUC:       authUC,
		onboardingUC: onboardingUC,
		config:       paramsConfig,
	}

	// Public Routes
	publicAuth := public.Group("/auth")
	{
		publicAuth.POST("/login", handler.Login)
		publicAuth.POST("/register", handler.Register)
		publicAuth.POST("/forgot-password", handler.ForgotPassword)
		publicAuth.POST("/reset-password", handler.ResetPassword)
		// Note: Email verification is handled directly by Supabase via email link
	}

	// Protected Routes
	protectedAuth := protected.Group("/auth")
	{
		protectedAuth.POST("/sync", handler.SyncProfile)
		protectedAuth.GET("/me", handler.Me)
	}
}

// Note: Manual email verification endpoint removed.
// Supabase handles email verification via direct link clicks from the confirmation email.

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=6"`
	Role         string `json:"role" binding:"required,oneof=candidate employer"`
	CaptchaToken string `json:"captchaToken"` // Cloudflare Turnstile Token
}

// Register godoc
// @Summary      User Registration
// @Description  Register a new user with email, password, and role. Supports Turnstile Captcha.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        register  body      RegisterRequest  true  "Registration Details"
// @Success      201    {object}  response.Response
// @Failure      400    {object}  response.Response
// @Failure      409    {object}  response.Response
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	// 1. Prepare Request to Supabase Auth API
	// We use direct HTTP client to pass custom Captcha headers, which gotrue-go might not support directly per-request.
	supabaseURL := h.config.SupabaseUrl
	if len(supabaseURL) > 0 && supabaseURL[len(supabaseURL)-1] == '/' {
		supabaseURL = supabaseURL[:len(supabaseURL)-1]
	}
	signupURL := fmt.Sprintf("%s/auth/v1/signup", supabaseURL)

	// Build redirect URL for email confirmation
	emailRedirectTo := h.config.FrontendURL + "/auth/callback"

	reqBody := map[string]interface{}{
		"email":    req.Email,
		"password": req.Password,
		"data": map[string]interface{}{
			"role": req.Role,
		},
		// Pass redirect URL in options (this is Supabase's documented format)
		"options": map[string]interface{}{
			"emailRedirectTo": emailRedirectTo,
		},
	}
	// Add captcha token to request body (Supabase expects this in the body, not headers)
	if req.CaptchaToken != "" {
		reqBody["gotrue_meta_security"] = map[string]interface{}{
			"captcha_token": req.CaptchaToken,
		}
	}
	jsonBody, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", signupURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		c.Error(apperror.Internal(err))
		return
	}

	// 2. Set Headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", h.config.SupabaseKey)

	// Forward Client IP and User Agent - Critical for Captcha and Security
	httpReq.Header.Set("X-Forwarded-For", c.ClientIP())
	httpReq.Header.Set("User-Agent", c.Request.UserAgent())
	if origin := c.Request.Header.Get("Map-Origin"); origin != "" {
		httpReq.Header.Set("Origin", origin)
	}

	// Add Turnstile/Captcha Headers if token provided
	if req.CaptchaToken != "" {
		httpReq.Header.Set("cf-turnstile-response", req.CaptchaToken)
		// Supabase docs sometimes mention h-captcha-response as a generic bucket for captcha tokens
		httpReq.Header.Set("h-captcha-response", req.CaptchaToken)
	}

	// 3. Execute Request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("Supabase Request Error: %v\n", err)
		c.Error(apperror.New(http.StatusInternalServerError, "Registration service unavailable", err))
		return
	}
	defer resp.Body.Close()

	// 4. Handle Response
	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		fmt.Printf("Supabase Error Response: %v\n", errResp)

		msg := "Registration failed"
		if m, ok := errResp["msg"].(string); ok {
			msg = m
		} else if m, ok := errResp["error_description"].(string); ok {
			msg = m
		}

		c.Error(apperror.BadRequest(msg))
		return
	}

	var supabaseUser struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&supabaseUser); err != nil {
		c.Error(apperror.New(http.StatusInternalServerError, "Failed to parse response", err))
		return
	}

	// 5. Response - User will be synced to local DB on first login (after email verification)
	// This ensures email must be verified before the user exists in our database
	msg := "Registration successful. Please check your email to confirm."
	var data interface{} = nil

	if supabaseUser.AccessToken != "" {
		// Auto-verified case (e.g., email already confirmed or auto-confirm enabled)
		// Sync user now since they're already verified
		user := &domain.User{
			ID:    supabaseUser.ID,
			Email: req.Email,
			Role:  req.Role,
		}
		if err := h.authUC.EnsureUserExists(c.Request.Context(), user); err != nil {
			c.Error(err)
			return
		}
		msg = "Registration successful"
		data = gin.H{
			"token": supabaseUser.AccessToken,
			"user":  user,
		}
	}

	response.Success(c, http.StatusCreated, msg, data)

}

// Login godoc
// @Summary      User Login
// @Description  Login with email and password via Supabase
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        login  body      LoginRequest  true  "Login Credentials"
// @Success      200    {object}  response.Response
// @Failure      400    {object}  response.Response
// @Failure      401    {object}  response.Response
// @Router       /auth/login [post]

// Login godoc
// @Summary      User Login
// @Description  Login with email and password via Supabase
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        login  body      LoginRequest  true  "Login Credentials"
// @Success      200    {object}  response.Response
// @Failure      400    {object}  response.Response
// @Failure      401    {object}  response.Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	// Use direct HTTP call to Supabase /token/grant endpoint (OTP/Password)
	// Actually for email/password it's /auth/v1/token?grant_type=password
	// Ref: https://supabase.com/docs/reference/api/auth-token

	supabaseURL := h.config.SupabaseUrl
	if len(supabaseURL) > 0 && supabaseURL[len(supabaseURL)-1] == '/' {
		supabaseURL = supabaseURL[:len(supabaseURL)-1]
	}
	// For password login: POST /token?grant_type=password
	loginURL := fmt.Sprintf("%s/auth/v1/token?grant_type=password", supabaseURL)

	reqBody := map[string]interface{}{
		"email":    req.Email,
		"password": req.Password,
	}
	jsonBody, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		c.Error(apperror.Internal(err))
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", h.config.SupabaseKey)

	// Forward Client IP and User Agent
	httpReq.Header.Set("X-Forwarded-For", c.ClientIP())
	httpReq.Header.Set("User-Agent", c.Request.UserAgent())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("Supabase Login Error: %v\n", err)
		c.Error(apperror.New(http.StatusInternalServerError, "Login service unavailable", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		fmt.Printf("Supabase Login Error: response status code %d: %s\n", resp.StatusCode, debugJSON(errResp)) // Helper or just stringify

		msg := "Wrong Password Or Account Not Found!"
		// If captcha failed, be specific if possible, though usually it's just 400
		if m, ok := errResp["msg"].(string); ok {
			// e.g. "captcha verification process failed"
			if m == "captcha verification process failed" {
				msg = m
			} else if m == "Invalid login credentials" {
				msg = "Wrong Password Or Account Not Found!" // Keep generic
			} else {
				// Keep other messages generic or pass through?
				// "Email not confirmed" is another common one.
				if m == "Email not confirmed" {
					msg = m
				}
			}
		}

		c.Error(apperror.Unauthorized(msg))
		return
	}

	var supabaseUser struct {
		User        domain.User `json:"user"`
		AccessToken string      `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&supabaseUser); err != nil {
		c.Error(apperror.New(http.StatusInternalServerError, "Failed to parse login response", err))
		return
	}

	// Sync User
	user := &domain.User{
		ID:    supabaseUser.User.ID,
		Email: supabaseUser.User.Email,
		// Role: Leave empty so EnsureUserExists doesn't overwrite existing role.
		// If user doesn't exist, EnsureUserExists will default it to 'candidate'.
	}

	if err := h.authUC.EnsureUserExists(c.Request.Context(), user); err != nil {
		c.Error(err)
		return
	}

	// Fetch the actual user from DB to get the correct Role (e.g. if it's 'admin')
	// EnsureUserExists might have created it as 'candidate' or left it as 'admin'
	actualUser, err := h.authUC.GetCurrentUser(c.Request.Context(), user.ID)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Login successful", gin.H{
		"token": supabaseUser.AccessToken,
		"user":  actualUser,
	})
}

// debugJSON Helper
func debugJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (h *AuthHandler) SyncProfile(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))
	email := c.GetString(string(domain.KeyUserEmail))
	role := c.GetString(string(domain.KeyUserRole))

	user := &domain.User{
		ID:    userID,
		Email: email,
		Role:  role,
	}

	if err := h.authUC.EnsureUserExists(c, user); err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Profile synced", user)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))
	user, err := h.authUC.GetCurrentUser(c, userID)
	if err != nil {
		c.Error(err)
		return
	}

	// For candidates, check onboarding status
	var onboardingCompleted *bool
	if user.Role == "candidate" { // Todo: Use domain constant if available
		status, err := h.onboardingUC.GetOnboardingStatus(c, userID)
		if err == nil && status != nil {
			onboardingCompleted = &status.Completed
		} else {
			// If error or nil, assume false to be safe (or nil if we want to show unknown)
			val := false
			onboardingCompleted = &val
		}
	}

	response.Success(c, http.StatusOK, "User details", gin.H{
		"user":                 user,
		"onboarding_completed": onboardingCompleted,
	})
}

// ForgotPasswordRequest for requesting password reset email
type ForgotPasswordRequest struct {
	Email        string `json:"email" binding:"required,email"`
	CaptchaToken string `json:"captchaToken" binding:"required"`
}

// ForgotPassword godoc
// @Summary      Request Password Reset
// @Description  Send password reset email to user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      ForgotPasswordRequest  true  "Email address and captcha"
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	// SECURITY: Track start time for constant-time response (timing attack mitigation)
	start := time.Now()

	// Target response time - should match the slowest path (valid email + Supabase call)
	// This prevents attackers from using response time to determine if email exists
	const targetDuration = 2 * time.Second

	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	// SECURITY: Always return the same response whether email exists or not.
	// This prevents email enumeration attacks where attackers probe to find valid emails.
	// The actual password reset email will only be sent if the email exists.

	successMessage := "If an account with that email exists, a password reset link has been sent."

	// 1. Check if email exists in local database (silently)
	exists, err := h.authUC.CheckEmailExists(c.Request.Context(), req.Email)
	if err != nil {
		// Log internally but don't expose to user
		fmt.Printf("ForgotPassword check error (non-fatal): %v\n", err)
		// Apply artificial delay to maintain constant response time
		h.simulateDelay(start, targetDuration)
		response.Success(c, http.StatusOK, successMessage, nil)
		return
	}

	if !exists {
		// Email doesn't exist - return fake success (no email sent)
		// SECURITY: Apply artificial delay to make response time identical to valid email path
		// This prevents timing attacks where attackers measure response time to enumerate emails
		h.simulateDelay(start, targetDuration)
		response.Success(c, http.StatusOK, successMessage, nil)
		return
	}

	// 2. Email exists - actually send the reset email via Supabase
	supabaseURL := h.config.SupabaseUrl
	if len(supabaseURL) > 0 && supabaseURL[len(supabaseURL)-1] == '/' {
		supabaseURL = supabaseURL[:len(supabaseURL)-1]
	}

	// Build recovery URL with redirect as query parameter
	// Supabase GoTrue API /recover endpoint requires redirect_to as a QUERY PARAMETER
	redirectURL := h.config.FrontendURL + "/auth/update-password"

	// Safely build the URL with query parameters
	u, _ := url.Parse(supabaseURL + "/auth/v1/recover")
	q := u.Query()
	q.Set("redirect_to", redirectURL)
	u.RawQuery = q.Encode()

	recoveryURL := u.String()

	reqBody := map[string]interface{}{
		"email": req.Email,
	}
	// Add captcha token to request body
	if req.CaptchaToken != "" {
		reqBody["gotrue_meta_security"] = map[string]interface{}{
			"captcha_token": req.CaptchaToken,
		}
	}
	jsonBody, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("POST", recoveryURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		// Log internally but return same success message
		fmt.Printf("ForgotPassword request creation error: %v\n", err)
		h.simulateDelay(start, targetDuration) // Ensure constant timing
		response.Success(c, http.StatusOK, successMessage, nil)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", h.config.SupabaseKey)

	// Forward client headers for captcha verification
	httpReq.Header.Set("X-Forwarded-For", c.ClientIP())
	httpReq.Header.Set("User-Agent", c.Request.UserAgent())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		// Log internally but don't reveal failure to user
		fmt.Printf("Supabase Recovery Error: %v\n", err)
		h.simulateDelay(start, targetDuration) // Ensure constant timing
		response.Success(c, http.StatusOK, successMessage, nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		// Log the actual error internally
		fmt.Printf("Supabase Recovery Error Response (non-fatal): %v\n", errResp)
		// Still return success to user - don't reveal if email exists or if there's a backend issue
	}

	// SECURITY: Apply delay even after successful Supabase call
	// to ensure ALL paths take the same amount of time
	h.simulateDelay(start, targetDuration)
	response.Success(c, http.StatusOK, successMessage, nil)
}

// simulateDelay ensures the response takes at least targetDuration from start time.
// This prevents timing attacks by making response times constant regardless of code path.
// If the actual processing already took longer than targetDuration, no delay is added.
func (h *AuthHandler) simulateDelay(start time.Time, targetDuration time.Duration) {
	elapsed := time.Since(start)
	if elapsed < targetDuration {
		time.Sleep(targetDuration - elapsed)
	}
}

// ResetPasswordRequest for setting new password
type ResetPasswordRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ResetPassword godoc
// @Summary      Reset Password
// @Description  Set new password using reset token from email link
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      ResetPasswordRequest  true  "Reset password details"
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	supabaseURL := h.config.SupabaseUrl
	if len(supabaseURL) > 0 && supabaseURL[len(supabaseURL)-1] == '/' {
		supabaseURL = supabaseURL[:len(supabaseURL)-1]
	}

	// Supabase user update endpoint - requires the access token from the reset link
	updateURL := fmt.Sprintf("%s/auth/v1/user", supabaseURL)

	reqBody := map[string]interface{}{
		"password": req.NewPassword,
	}
	jsonBody, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		c.Error(apperror.Internal(err))
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", h.config.SupabaseKey)
	// Use the access token from the password reset link
	httpReq.Header.Set("Authorization", "Bearer "+req.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("Supabase Password Update Error: %v\n", err)
		c.Error(apperror.New(http.StatusInternalServerError, "Password update service unavailable", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		fmt.Printf("Supabase Password Update Error Response: %v\n", errResp)

		msg := "Password reset failed"
		if m, ok := errResp["msg"].(string); ok {
			msg = m
		} else if m, ok := errResp["error_description"].(string); ok {
			msg = m
		}
		c.Error(apperror.BadRequest(msg))
		return
	}

	response.Success(c, http.StatusOK, "Password has been reset successfully. You can now login with your new password.", nil)
}
