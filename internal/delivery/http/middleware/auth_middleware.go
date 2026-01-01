package middleware

import (
	"fmt"
	"go-recruitment-backend/config"
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/auth"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwksProvider *auth.Provider, cfg *config.Config, authUC domain.AuthUsecase) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		var tokenString string

		// 1. Try to get token from Header
		if authHeader != "" {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// 2. Try to get token from Cookie
			cookie, err := c.Cookie("auth_token")
			if err == nil && cookie != "" {
				tokenString = cookie
			}
		}

		if tokenString == "" {
			response.Error(c, http.StatusUnauthorized, "Authorization header or auth_token cookie required", nil)
			c.Abort()
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Check signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
				// HS256 - Use Secret
				if cfg.SupabaseJWTSecret == "" {
					return nil, fmt.Errorf("HS256 token received but SUPABASE_JWT_KEY is not configured")
				}
				return []byte(cfg.SupabaseJWTSecret), nil
			}

			if _, ok := token.Method.(*jwt.SigningMethodRSA); ok {
				// RS256 - Use JWKS
				return jwksProvider.KeyFunc(token)
			}

			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		})

		if err != nil || !token.Valid {
			fmt.Printf("Token validation failed: %v\n", err)
			response.Error(c, http.StatusUnauthorized, "Invalid token", err.Error())
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "Invalid claims", nil)
			c.Abort()
			return
		}

		// Extract Supabase standard claims
		sub, _ := claims["sub"].(string)
		email, _ := claims["email"].(string)

		// Fetch fresh user data from DB to get the correct Role
		// We do NOT rely on the JWT role claim as it might be 'authenticated' or stale
		user, err := authUC.GetCurrentUser(c.Request.Context(), sub)
		if err != nil {
			// If user not found in local DB but has valid token, we might need to sync or unauthorized
			// For strictness, if not in DB, unauthorized
			response.Error(c, http.StatusUnauthorized, "User not found", nil)
			c.Abort()
			return
		}

		role := user.Role
		if role == "" {
			role = "candidate" // Fallback
		}

		c.Set(string(domain.KeyUserID), sub)
		c.Set(string(domain.KeyUserEmail), email)
		c.Set(string(domain.KeyUserRole), role)

		c.Next()
	}
}
