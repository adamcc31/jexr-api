package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORSMiddleware adds CORS headers for cross-origin requests
// This is required for the Next.js frontend (port 3000) to communicate
// with the Go backend (port 8080)
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Allow specific origins
		allowedOrigins := map[string]bool{
			// Production
			"https://www.jexpertrecruitment.com": true,
			"https://jexpertrecruitment.com":     true,
			// Development
			"http://localhost:3000":    true,
			"http://127.0.0.1:3000":    true,
			"http://localhost:3001":    true,
			"http://192.168.1.26:3000": true,
		}

		if allowedOrigins[origin] || origin == "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
