package util

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the standard JSON error format
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Custom error types
type ErrorCode string

const (
	ErrPayloadTooLarge ErrorCode = "payload_too_large"
	ErrValidation      ErrorCode = "validation_error"
	ErrNotFound        ErrorCode = "not_found"
	ErrUnauthorized    ErrorCode = "unauthorized"
	ErrInternal        ErrorCode = "internal_error"
)

// HTTPError represents an error with HTTP status and error code
type HTTPError struct {
	StatusCode int
	Code       ErrorCode
	Message    string
}

// NewHTTPError creates a new HTTPError
func NewHTTPError(statusCode int, code ErrorCode, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	}
}

// PayloadTooLarge returns 413 error
func PayloadTooLarge(message string) *HTTPError {
	if message == "" {
		message = "Request body exceeds 1MB limit"
	}
	return NewHTTPError(http.StatusRequestEntityTooLarge, ErrPayloadTooLarge, message)
}

// ValidationError returns 400 error
func ValidationError(message string) *HTTPError {
	if message == "" {
		message = "Invalid request"
	}
	return NewHTTPError(http.StatusBadRequest, ErrValidation, message)
}

// NotFound returns 404 error
func NotFound(message string) *HTTPError {
	if message == "" {
		message = "Resource not found"
	}
	return NewHTTPError(http.StatusNotFound, ErrNotFound, message)
}

// Unauthorized returns 401 error
func Unauthorized(message string) *HTTPError {
	if message == "" {
		message = "Unauthorized"
	}
	return NewHTTPError(http.StatusUnauthorized, ErrUnauthorized, message)
}

// InternalError returns 500 error
func InternalError(message string) *HTTPError {
	if message == "" {
		message = "Internal server error"
	}
	return NewHTTPError(http.StatusInternalServerError, ErrInternal, message)
}

// WriteError writes an HTTPError as JSON to the response writer
func WriteError(w http.ResponseWriter, err *HTTPError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode)

	errResp := ErrorResponse{
		Error:   string(err.Code),
		Message: err.Message,
	}
	json.NewEncoder(w).Encode(errResp)
}
