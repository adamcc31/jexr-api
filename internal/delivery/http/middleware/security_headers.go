package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware adds essential security headers to all responses.
// These headers protect against common web vulnerabilities:
// - MITM attacks (HSTS)
// - XSS attacks (X-XSS-Protection, X-Content-Type-Options)
// - Clickjacking (X-Frame-Options)
// - Information leakage (Referrer-Policy, Permissions-Policy)
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// HTTP Strict Transport Security (HSTS)
		// Forces browsers to only use HTTPS for this domain
		// max-age=63072000 = 2 years, includeSubDomains covers all subdomains
		c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")

		// Prevent MIME type sniffing
		// Stops browsers from incorrectly interpreting files as different MIME types
		c.Header("X-Content-Type-Options", "nosniff")

		// Legacy XSS protection (for older browsers)
		// Modern browsers use CSP instead, but this doesn't hurt
		c.Header("X-XSS-Protection", "1; mode=block")

		// Prevent clickjacking by disallowing framing
		// DENY = never allow framing, SAMEORIGIN = only same origin can frame
		c.Header("X-Frame-Options", "DENY")

		// Control referrer information sent with requests
		// strict-origin-when-cross-origin = send full URL to same origin, only origin to cross-origin
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Restrict browser features access
		// Empty values = disable the feature entirely
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		// Content Security Policy (basic policy)
		// This is a baseline CSP - adjust based on actual resource usage
		// For APIs, this primarily affects error pages and any HTML responses
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: https://*.supabase.co; "+
				"font-src 'self'; "+
				"connect-src 'self' https://*.supabase.co; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")

		// Prevent caching of sensitive data
		// This is especially important for authenticated API responses
		if c.GetHeader("Authorization") != "" {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}
