package helpers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error  string `json:"error"`  
	Message string `json:"message"`
}

func HTTPStatusText(code int) string {
	return http.StatusText(code)
}

func RespondWithError(c *gin.Context, statusCode int, customMessage string) {
	c.JSON(statusCode, ErrorResponse{
		Error:  HTTPStatusText(statusCode),
		Message: customMessage,
	})
}
