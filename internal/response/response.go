package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response representa la estructura estándar de respuesta de la API
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ErrorResponse representa una respuesta de error
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    int    `json:"code"`
}

// SuccessResponse envía una respuesta exitosa
func SuccessResponse(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// ErrorResponseWithMessage envía una respuesta de error con mensaje personalizado
func ErrorResponseWithMessage(c *gin.Context, status int, message string) {
	c.JSON(status, ErrorResponse{
		Success: false,
		Error:   message,
		Code:    status,
	})
}

// BadRequestError envía un error 400
func BadRequestError(c *gin.Context, message string) {
	ErrorResponseWithMessage(c, http.StatusBadRequest, message)
}

// NotFoundError envía un error 404
func NotFoundError(c *gin.Context, message string) {
	ErrorResponseWithMessage(c, http.StatusNotFound, message)
}

// InternalServerError envía un error 500
func InternalServerError(c *gin.Context, message string) {
	ErrorResponseWithMessage(c, http.StatusInternalServerError, message)
}

// UnauthorizedError envía un error 401
func UnauthorizedError(c *gin.Context, message string) {
	ErrorResponseWithMessage(c, http.StatusUnauthorized, message)
}

// ConflictError envía un error 409
func ConflictError(c *gin.Context, message string) {
	ErrorResponseWithMessage(c, http.StatusConflict, message)
}
