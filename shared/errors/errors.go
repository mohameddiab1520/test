package errors

import (
	"fmt"
	"net/http"
)

// Error represents a custom application error
type Error struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
	Details    interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// New creates a new custom error
func New(code, message string, statusCode int) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// WithDetails adds details to an error
func (e *Error) WithDetails(details interface{}) *Error {
	e.Details = details
	return e
}

// Common error codes
const (
	ErrCodeInternal         = "INTERNAL_ERROR"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeBadRequest       = "BAD_REQUEST"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeConflict         = "CONFLICT"
	ErrCodeValidation       = "VALIDATION_ERROR"
	ErrCodeDatabaseError    = "DATABASE_ERROR"
	ErrCodeExternalService  = "EXTERNAL_SERVICE_ERROR"
)

// Predefined errors
var (
	ErrInternal = &Error{
		Code:       ErrCodeInternal,
		Message:    "Internal server error",
		StatusCode: http.StatusInternalServerError,
	}

	ErrNotFound = &Error{
		Code:       ErrCodeNotFound,
		Message:    "Resource not found",
		StatusCode: http.StatusNotFound,
	}

	ErrBadRequest = &Error{
		Code:       ErrCodeBadRequest,
		Message:    "Bad request",
		StatusCode: http.StatusBadRequest,
	}

	ErrUnauthorized = &Error{
		Code:       ErrCodeUnauthorized,
		Message:    "Unauthorized",
		StatusCode: http.StatusUnauthorized,
	}

	ErrForbidden = &Error{
		Code:       ErrCodeForbidden,
		Message:    "Forbidden",
		StatusCode: http.StatusForbidden,
	}

	ErrConflict = &Error{
		Code:       ErrCodeConflict,
		Message:    "Resource conflict",
		StatusCode: http.StatusConflict,
	}

	ErrValidation = &Error{
		Code:       ErrCodeValidation,
		Message:    "Validation error",
		StatusCode: http.StatusBadRequest,
	}

	ErrDatabase = &Error{
		Code:       ErrCodeDatabaseError,
		Message:    "Database error",
		StatusCode: http.StatusInternalServerError,
	}

	ErrExternalService = &Error{
		Code:       ErrCodeExternalService,
		Message:    "External service error",
		StatusCode: http.StatusBadGateway,
	}
)

// IsAppError checks if an error is an application error
func IsAppError(err error) bool {
	_, ok := err.(*Error)
	return ok
}

// GetAppError converts an error to an application error
func GetAppError(err error) *Error {
	if appErr, ok := err.(*Error); ok {
		return appErr
	}
	return ErrInternal
}
