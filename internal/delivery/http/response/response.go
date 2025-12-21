package response

import (
	"github.com/gin-gonic/gin"
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
