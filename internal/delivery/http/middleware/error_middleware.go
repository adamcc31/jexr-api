package middleware

import (
	"errors"
	"fmt"
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/pkg/apperror"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are errors appended to the context
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			var appErr *apperror.AppError
			if errors.As(err, &appErr) {
				response.Error(c, appErr.Code, appErr.Message, nil)
			} else {
				// SECURITY: Never expose internal error details to clients.
				// Log the actual error server-side for debugging, but send a
				// generic message to the user to prevent information disclosure.
				fmt.Printf("[ERROR] Internal Server Error: %v\n", err)
				response.Error(c, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.", nil)
			}
		}
	}
}
