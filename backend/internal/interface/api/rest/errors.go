// Package rest contains the HTTP interface layer: controllers, middleware,
// DTOs and centralized error shaping.
package rest

import "github.com/gofiber/fiber/v2"

// ErrorCode is a machine-readable error code.
type ErrorCode string

const (
	CodeValidation ErrorCode = "ValidationError"
	CodeAuth       ErrorCode = "AuthenticationRequired"
	CodeForbidden  ErrorCode = "Forbidden"
	CodeNotFound   ErrorCode = "NotFound"
	CodeBadRequest ErrorCode = "BadRequest"
	CodeInternal   ErrorCode = "InternalServerError"
)

// FieldError describes one invalid request field.
type FieldError struct {
	Field string
	Issue string
}

// APIError is the single typed error shape used across the API.
// Returning it from a handler lets the central ErrorHandler render it.
type APIError struct {
	Status     int
	Code       ErrorCode
	Message    string
	ResourceID string
	Details    []FieldError
}

func (e *APIError) Error() string { return e.Message }

// Constructors (no global sentinel variables).

func NewBadRequest(msg string) *APIError {
	return &APIError{Status: fiber.StatusBadRequest, Code: CodeBadRequest, Message: msg}
}

func NewValidation(msg string, details []FieldError) *APIError {
	return &APIError{Status: fiber.StatusBadRequest, Code: CodeValidation, Message: msg, Details: details}
}

func NewAuth(msg string) *APIError {
	return &APIError{Status: fiber.StatusUnauthorized, Code: CodeAuth, Message: msg}
}

func NewNotFound(msg, resourceID string) *APIError {
	return &APIError{Status: fiber.StatusNotFound, Code: CodeNotFound, Message: msg, ResourceID: resourceID}
}
