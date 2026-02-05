package response

import (
	"strings"

	"go-recruitment-backend/pkg/validation"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Response standardizes the API JSON response
type Response struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     interface{} `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// Success sends a success response
func Success(c *gin.Context, code int, message string, data interface{}) {
	reqID, _ := c.Get("RequestID")
	idStr, _ := reqID.(string) // Safe type assertion

	c.JSON(code, Response{
		Success:   true,
		Message:   message,
		Data:      data,
		RequestID: idStr,
	})
}

// Error sends an error response
func Error(c *gin.Context, code int, message string, err interface{}) {
	reqID, _ := c.Get("RequestID")
	idStr, _ := reqID.(string)

	c.JSON(code, Response{
		Success:   false,
		Message:   message,
		Error:     err,
		RequestID: idStr,
	})
}

// ValidationError sends a user-friendly validation error response
// It detects validator.ValidationErrors and formats them with proper Indonesian field labels
func ValidationError(c *gin.Context, err error) {
	reqID, _ := c.Get("RequestID")
	idStr, _ := reqID.(string)

	// Try to extract validator.ValidationErrors
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		messages := validation.FormatValidationErrors(validationErrs)
		c.JSON(400, Response{
			Success:   false,
			Message:   "Validasi gagal: " + strings.Join(messages, "; "),
			Error:     messages,
			RequestID: idStr,
		})
		return
	}

	// Fallback for non-validation errors (e.g., JSON parse errors)
	c.JSON(400, Response{
		Success:   false,
		Message:   "Data tidak valid: " + err.Error(),
		Error:     err.Error(),
		RequestID: idStr,
	})
}
