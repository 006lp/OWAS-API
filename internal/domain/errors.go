package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors defining domain-level failure categories.
var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrNotFound        = errors.New("resource not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrUpstream        = errors.New("upstream error")
	ErrUpstreamTimeout = errors.New("upstream timeout")
)

// AppError carries a stable machine-readable code, human-readable message,
// optional diagnostic details, and an underlying sentinel/root cause error.
type AppError struct {
	Code    string
	Message string
	Details map[string]any
	Cause   error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap allows errors.Is and errors.As to inspect the wrapped cause.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewInvalidInputError creates a domain validation error.
func NewInvalidInputError(message string, details map[string]any) *AppError {
	return &AppError{
		Code:    "INVALID_ARGUMENT",
		Message: message,
		Details: details,
		Cause:   ErrInvalidInput,
	}
}

// NewNotFoundError creates a resource not found error.
func NewNotFoundError(message string) *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: message,
		Cause:   ErrNotFound,
	}
}

// NewUnauthorizedError creates an authentication/authorization rejection error.
func NewUnauthorizedError(message string, details map[string]any) *AppError {
	return &AppError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Details: details,
		Cause:   ErrUnauthorized,
	}
}

// NewUpstreamError creates an error representing an upstream service or protocol failure.
func NewUpstreamError(message string, cause error, details map[string]any) *AppError {
	wrapped := ErrUpstream
	if cause != nil {
		wrapped = fmt.Errorf("%w: %v", ErrUpstream, cause)
	}
	return &AppError{
		Code:    "UPSTREAM_ERROR",
		Message: message,
		Details: details,
		Cause:   wrapped,
	}
}

// NewUpstreamTimeoutError creates an error representing an upstream network timeout.
func NewUpstreamTimeoutError(message string, cause error) *AppError {
	wrapped := ErrUpstreamTimeout
	if cause != nil {
		wrapped = fmt.Errorf("%w: %v", ErrUpstreamTimeout, cause)
	}
	return &AppError{
		Code:    "UPSTREAM_TIMEOUT",
		Message: message,
		Cause:   wrapped,
	}
}
