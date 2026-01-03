package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"go-recruitment-backend/internal/delivery/http/response"

	"github.com/gin-gonic/gin"
)

const (
	// CSRFTokenCookieName is the name of the cookie that stores the CSRF token
	CSRFTokenCookieName = "csrf_token"
	// CSRFTokenHeaderName is the name of the header that must contain the CSRF token
	CSRFTokenHeaderName = "X-CSRF-Token"
	// CSRFTokenLength is the length of the generated token in bytes (32 bytes = 64 hex chars)
	CSRFTokenLength = 32
	// CSRFTokenExpiry is how long the token is valid
	CSRFTokenExpiry = 24 * time.Hour
)

// generateCSRFToken creates a cryptographically secure random token
func generateCSRFToken() (string, error) {
	bytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CSRFMiddleware implements Double-Submit Cookie pattern for CSRF protection
//
// How it works:
// 1. On any request, if no csrf_token cookie exists, generate one and set it
// 2. For state-changing requests (POST, PUT, DELETE, PATCH), validate that:
//   - The X-CSRF-Token header exists
//   - The header value matches the csrf_token cookie value
//
// The frontend must:
// 1. Read the csrf_token cookie value
// 2. Include it in X-CSRF-Token header for all mutating requests
//
// This works because:
// - Cookies are automatically sent with requests (even cross-origin with credentials)
// - But attackers cannot read cookie values from a different origin
// - So they cannot forge the X-CSRF-Token header
//
// EXEMPTIONS:
// - Public auth routes are exempt because users don't have a session/cookie yet
// - These routes are protected by rate limiting instead
func CSRFMiddleware() gin.HandlerFunc {
	// Routes that are exempt from CSRF protection
	// These are public endpoints where users don't have a session yet
	// or endpoints that have their own authentication (like file uploads with JWT)
	csrfExemptPaths := map[string]bool{
		"/v1/auth/login":           true,
		"/v1/auth/register":        true,
		"/v1/auth/forgot-password": true,
		"/v1/auth/reset-password":  true,
		"/v1/contact":              true, // Public contact form
		"/v1/health":               true, // Health check
		"/v1/upload":               true, // File upload (protected by JWT auth instead)
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Check if path is exempt
		if csrfExemptPaths[path] {
			// Still set the cookie for future requests, but don't validate
			csrfCookie, err := c.Cookie(CSRFTokenCookieName)
			if err != nil || csrfCookie == "" {
				newToken, _ := generateCSRFToken()
				if newToken != "" {
					c.SetSameSite(http.SameSiteLaxMode)
					c.SetCookie(
						CSRFTokenCookieName,
						newToken,
						int(CSRFTokenExpiry.Seconds()),
						"/",
						"",
						true,
						false,
					)
				}
			}
			c.Next()
			return
		}

		// Get or generate CSRF token
		csrfCookie, err := c.Cookie(CSRFTokenCookieName)

		// Generate new token if none exists
		if err != nil || csrfCookie == "" {
			newToken, err := generateCSRFToken()
			if err != nil {
				response.Error(c, http.StatusInternalServerError, "Failed to generate security token", nil)
				c.Abort()
				return
			}

			// Set cookie with security attributes
			// SameSite=Lax allows the cookie to be sent on top-level navigations
			// but not on cross-site subrequests (forms, iframes)
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie(
				CSRFTokenCookieName,
				newToken,
				int(CSRFTokenExpiry.Seconds()),
				"/",
				"",    // Domain (empty = current domain)
				true,  // Secure (HTTPS only)
				false, // HttpOnly = false so JS can read it
			)
			csrfCookie = newToken
		}

		// For safe methods, no validation needed
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			c.Next()
			return
		}

		// For state-changing methods, validate CSRF token
		headerToken := c.GetHeader(CSRFTokenHeaderName)

		if headerToken == "" {
			response.Error(c, http.StatusForbidden, "Missing CSRF token", nil)
			c.Abort()
			return
		}

		if headerToken != csrfCookie {
			response.Error(c, http.StatusForbidden, "Invalid CSRF token", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// CSRFExempt creates a route group that bypasses CSRF checks
// Use sparingly and only for specific cases like:
// - Webhook endpoints that receive external POST requests
// - Public API endpoints used by non-browser clients
func CSRFExempt(r *gin.RouterGroup) *gin.RouterGroup {
	exempt := r.Group("")
	exempt.Use(func(c *gin.Context) {
		c.Set("csrf_exempt", true)
		c.Next()
	})
	return exempt
}

// CSRFMiddlewareWithExemption is the same as CSRFMiddleware but respects exemptions
func CSRFMiddlewareWithExemption() gin.HandlerFunc {
	baseMiddleware := CSRFMiddleware()

	return func(c *gin.Context) {
		// Check if route is exempt
		if exempt, exists := c.Get("csrf_exempt"); exists && exempt.(bool) {
			c.Next()
			return
		}

		baseMiddleware(c)
	}
}
