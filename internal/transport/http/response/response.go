package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"owas-api/internal/domain"
)

// ContextKeyRequestID is the gin.Context key storing the per-request correlation ID.
const ContextKeyRequestID = "request_id"

// Envelope defines the uniform RESTful JSON response payload.
type Envelope struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"requestId,omitempty"`
	Data      any            `json:"data,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

func requestID(c *gin.Context) string {
	if v, ok := c.Get(ContextKeyRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// OK writes a 200 OK JSON envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{
		Code:      "OK",
		Message:   "success",
		RequestID: requestID(c),
		Data:      data,
	})
}

// Created writes a 201 Created JSON envelope.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{
		Code:      "OK",
		Message:   "created",
		RequestID: requestID(c),
		Data:      data,
	})
}

// Error maps a domain error into its corresponding RESTful HTTP status code and JSON envelope.
func Error(c *gin.Context, err error) {
	status, code, msg, details := classifyError(err)
	c.JSON(status, Envelope{
		Code:      code,
		Message:   msg,
		RequestID: requestID(c),
		Details:   details,
	})
}

// AbortWithError writes an error envelope and aborts subsequent middleware/handlers.
func AbortWithError(c *gin.Context, err error) {
	status, code, msg, details := classifyError(err)
	c.AbortWithStatusJSON(status, Envelope{
		Code:      code,
		Message:   msg,
		RequestID: requestID(c),
		Details:   details,
	})
}

// CustomError writes an explicit HTTP status and error code envelope (e.g., 404/405).
func CustomError(c *gin.Context, status int, code, message string) {
	c.JSON(status, Envelope{
		Code:      code,
		Message:   message,
		RequestID: requestID(c),
	})
}

func classifyError(err error) (status int, code, message string, details map[string]any) {
	var appErr *domain.AppError
	hasAppErr := errors.As(err, &appErr)

	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
		code = "INVALID_ARGUMENT"
	case errors.Is(err, domain.ErrUnauthorized):
		status = http.StatusUnauthorized
		code = "UNAUTHORIZED"
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		code = "NOT_FOUND"
	case errors.Is(err, domain.ErrUpstreamTimeout):
		status = http.StatusGatewayTimeout
		code = "UPSTREAM_TIMEOUT"
	case errors.Is(err, domain.ErrUpstream):
		status = http.StatusBadGateway
		code = "UPSTREAM_ERROR"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil
	}

	if hasAppErr {
		if appErr.Code != "" {
			code = appErr.Code
		}
		message = appErr.Message
		details = appErr.Details
	} else {
		message = err.Error()
	}
	return status, code, message, details
}
