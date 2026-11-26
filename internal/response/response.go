package response

import "github.com/gin-gonic/gin"

// Envelope defines a standard JSON API response wrapper.
type Envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// JSON sends a successful response with data.
func JSON(c *gin.Context, status int, message string, data any) {
	c.JSON(status, Envelope{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error sends an error response and aborts the Gin handler chain if needed.
func Error(c *gin.Context, status int, errMsg string) {
	c.JSON(status, Envelope{
		Success: false,
		Error:   errMsg,
	})
}
