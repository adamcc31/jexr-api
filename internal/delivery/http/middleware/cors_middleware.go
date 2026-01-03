package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware adds CORS headers for cross-origin requests
// This is required for the Next.js frontend (port 3000) to communicate
// with the Go backend (port 8080)
//
// SECURITY: This middleware is strict about allowed origins:
// - Production: Only explicit production domains
// - Development: Allows localhost (disabled in production)
// - Vercel previews: Only jexpert-* prefixed subdomains
func CORSMiddleware() gin.HandlerFunc {
	// Determine if we're in production mode
	isProduction := os.Getenv("GIN_MODE") == "release"

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// === SECURITY: Strict Origin Whitelist ===
		// Production domains (always allowed)
		productionOrigins := map[string]bool{
			"https://www.jexpertrecruitment.com": true,
			"https://jexpertrecruitment.com":     true,
		}

		// Development domains (only in non-production mode)
		devOrigins := map[string]bool{
			"http://localhost:3000": true,
			"http://127.0.0.1:3000": true,
			"http://localhost:3001": true,
		}

		var isAllowed bool

		// Check production origins
		if productionOrigins[origin] {
			isAllowed = true
		}

		// Check development origins (ONLY in development mode)
		if !isProduction && devOrigins[origin] {
			isAllowed = true
		}

		// Allow Vercel preview deployments with strict validation
		// Pattern: jexpert-*.vercel.app or *-jexpert-*.vercel.app
		if !isAllowed && strings.HasSuffix(origin, ".vercel.app") {
			// Extract subdomain
			subdomain := strings.TrimPrefix(origin, "https://")
			subdomain = strings.TrimSuffix(subdomain, ".vercel.app")

			// Only allow if subdomain contains "jexpert" as a prefix or segment
			// This prevents malicious-jexpert.vercel.app type attacks
			if strings.HasPrefix(subdomain, "jexpert") ||
				strings.Contains(subdomain, "-jexpert-") {
				isAllowed = true
			}
		}

		// Empty origin (same-origin requests) - allow
		if origin == "" {
			isAllowed = true
		}

		// === SECURITY: Only set headers if origin is allowed ===
		if isAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
			c.Header("Access-Control-Max-Age", "86400") // 24 hours
		}
		// If not allowed, no CORS headers are sent - browser will block the request

		// Vary header to ensure caches differentiate by Origin
		c.Header("Vary", "Origin")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if isAllowed {
				c.AbortWithStatus(204)
			} else {
				c.AbortWithStatus(403)
			}
			return
		}

		c.Next()
	}
}
