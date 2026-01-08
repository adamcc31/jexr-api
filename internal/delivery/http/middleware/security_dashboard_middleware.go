package middleware

import (
	"net/http"
	"strings"

	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/pkg/security"

	"github.com/gin-gonic/gin"
)

// SecurityMiddlewareConfig holds configuration for security middleware
type SecurityMiddlewareConfig struct {
	AuthService *security.SecurityAuthService
	Logger      *security.SecurityLogger
}

// SecurityIPAllowlistMiddleware validates that the request IP is in the allowed list
// This is the PRIMARY security control for the security dashboard
// The hidden route provides ZERO security guarantees - this is the real gate
func SecurityIPAllowlistMiddleware(authService *security.SecurityAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := getClientIP(c)

		allowed, err := authService.ValidateIP(c.Request.Context(), ip)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "IP validation failed", nil)
			c.Abort()
			return
		}

		if !allowed {
			// Log is already done in ValidateIP
			response.Error(c, http.StatusForbidden, "Access denied", nil)
			c.Abort()
			return
		}

		c.Set("client_ip", ip)
		c.Next()
	}
}

// SecurityAuthMiddleware validates the security dashboard session
// Requires MFA to be completed
func SecurityAuthMiddleware(authService *security.SecurityAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session token from cookie or header
		token := extractSecurityToken(c)
		if token == "" {
			response.Error(c, http.StatusUnauthorized, "Security session required", nil)
			c.Abort()
			return
		}

		ip := c.GetString("client_ip")
		if ip == "" {
			ip = getClientIP(c)
		}

		session, user, err := authService.ValidateSession(c.Request.Context(), token, ip)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "Invalid or expired session", nil)
			c.Abort()
			return
		}

		// Set user context
		c.Set("security_user", user)
		c.Set("security_session", session)
		c.Set("security_role", user.Role)

		c.Next()
	}
}

// SecurityRoleMiddleware enforces role-based access control
// roles parameter lists minimum required roles (any match allows access)
func SecurityRoleMiddleware(minRoles ...security.SecurityRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("security_role")
		if !exists {
			response.Error(c, http.StatusUnauthorized, "Role not determined", nil)
			c.Abort()
			return
		}

		userRole, ok := roleValue.(security.SecurityRole)
		if !ok {
			response.Error(c, http.StatusInternalServerError, "Invalid role type", nil)
			c.Abort()
			return
		}

		// Check if user role meets minimum requirement
		if !hasMinimumRole(userRole, minRoles) {
			security.DefaultLogger().Log(c.Request.Context(), security.SecurityEvent{
				Event:        security.EventUnauthorizedAccess,
				SubjectType:  "role",
				SubjectValue: string(userRole),
				IP:           c.GetString("client_ip"),
				Details: map[string]interface{}{
					"required_roles": rolesToStrings(minRoles),
					"endpoint":       c.Request.URL.Path,
				},
			})
			response.Error(c, http.StatusForbidden, "Insufficient permissions", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// BreakGlassRequiredMiddleware ensures an active break-glass session exists
// for DEVELOPER_ROOT level operations
func BreakGlassRequiredMiddleware(authService *security.SecurityAuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userValue, exists := c.Get("security_user")
		if !exists {
			response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
			c.Abort()
			return
		}

		user, ok := userValue.(*security.SecurityUser)
		if !ok {
			response.Error(c, http.StatusInternalServerError, "Invalid user type", nil)
			c.Abort()
			return
		}

		// Check for active break-glass session
		session, active, err := authService.CheckBreakGlassActive(c.Request.Context(), user.ID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Break-glass check failed", nil)
			c.Abort()
			return
		}

		if !active {
			security.DefaultLogger().Log(c.Request.Context(), security.SecurityEvent{
				Event:        security.EventUnauthorizedAccess,
				SubjectType:  "user_id",
				SubjectValue: security.HashValue(user.ID),
				IP:           c.GetString("client_ip"),
				Details: map[string]interface{}{
					"reason":   "break_glass_required",
					"endpoint": c.Request.URL.Path,
				},
			})
			response.Error(c, http.StatusForbidden, "Break-glass session required for this operation", nil)
			c.Abort()
			return
		}

		// Set break-glass context
		c.Set("break_glass_session", session)
		c.Next()
	}
}

// ReadOnlyModeMiddleware enforces read-only mode for specific roles
// SECURITY_OBSERVER can only perform GET requests
func ReadOnlyModeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("security_role")
		if !exists {
			c.Next()
			return
		}

		userRole, ok := roleValue.(security.SecurityRole)
		if !ok {
			c.Next()
			return
		}

		// Observers are read-only
		if userRole == security.RoleSecurityObserver {
			if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
				response.Error(c, http.StatusForbidden, "Observer role is read-only", nil)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// SecurityAuditMiddleware logs all security dashboard access
func SecurityAuditMiddleware() gin.HandlerFunc {
	logger := security.DefaultLogger()

	return func(c *gin.Context) {
		// Log request start
		userID := "anonymous"
		if user, exists := c.Get("security_user"); exists {
			if u, ok := user.(*security.SecurityUser); ok {
				userID = u.ID
			}
		}

		c.Next()

		// Log request completion
		logger.Log(c.Request.Context(), security.SecurityEvent{
			Event:        "security_dashboard_access",
			SubjectType:  "user_id",
			SubjectValue: security.HashValue(userID),
			IP:           c.GetString("client_ip"),
			UserAgent:    c.GetHeader("User-Agent"),
			Details: map[string]interface{}{
				"method":      c.Request.Method,
				"path":        c.Request.URL.Path,
				"status_code": c.Writer.Status(),
			},
		})
	}
}

// Helper functions

func extractSecurityToken(c *gin.Context) string {
	// Try cookie first
	if token, err := c.Cookie("security_session"); err == nil && token != "" {
		return token
	}

	// Try Authorization header
	authHeader := c.GetHeader("X-Security-Token")
	if authHeader != "" {
		return authHeader
	}

	// Try Bearer token as fallback
	bearerHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(bearerHeader, "SecurityBearer ") {
		return strings.TrimPrefix(bearerHeader, "SecurityBearer ")
	}

	return ""
}

func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header (for reverse proxy setups)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain (original client)
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return c.ClientIP()
}

// hasMinimumRole checks if user role meets minimum requirement
// Role hierarchy: OBSERVER < ANALYST < ADMIN
func hasMinimumRole(userRole security.SecurityRole, minRoles []security.SecurityRole) bool {
	roleHierarchy := map[security.SecurityRole]int{
		security.RoleSecurityObserver: 1,
		security.RoleSecurityAnalyst:  2,
		security.RoleSecurityAdmin:    3,
	}

	userLevel, ok := roleHierarchy[userRole]
	if !ok {
		return false
	}

	for _, minRole := range minRoles {
		minLevel, ok := roleHierarchy[minRole]
		if !ok {
			continue
		}
		if userLevel >= minLevel {
			return true
		}
	}

	return false
}

func rolesToStrings(roles []security.SecurityRole) []string {
	result := make([]string, len(roles))
	for i, r := range roles {
		result[i] = string(r)
	}
	return result
}
