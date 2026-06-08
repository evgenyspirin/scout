// Package validator centralizes request validation, returning typed field errors.
package validator

import (
	"strings"

	"scout/internal/interface/api/rest/dto"
)

// FieldError is a validation failure for a single field.
type FieldError struct {
	Field string
	Issue string
}

// ValidateLogin checks the login request body.
func ValidateLogin(r dto.LoginRequest) []FieldError {
	var errs []FieldError
	if strings.TrimSpace(r.Login) == "" {
		errs = append(errs, FieldError{Field: "login", Issue: "login is required"})
	}
	if strings.TrimSpace(r.Password) == "" {
		errs = append(errs, FieldError{Field: "password", Issue: "password is required"})
	}
	return errs
}

// ValidateUploadLink checks the upload-link request body.
func ValidateUploadLink(r dto.UploadLinkRequest) []FieldError {
	var errs []FieldError
	ct := strings.TrimSpace(r.ContentType)
	if ct == "" {
		errs = append(errs, FieldError{Field: "contentType", Issue: "contentType is required"})
	} else if !strings.HasPrefix(ct, "image/") {
		errs = append(errs, FieldError{Field: "contentType", Issue: "contentType must be an image/* type"})
	}
	return errs
}

// ValidateListQuery checks list pagination/filter query parameters.
func ValidateListQuery(limit int, minConfidence *float64) []FieldError {
	var errs []FieldError
	if limit < 1 || limit > 200 {
		errs = append(errs, FieldError{Field: "limit", Issue: "limit must be between 1 and 200"})
	}
	if minConfidence != nil && (*minConfidence < 0 || *minConfidence > 1) {
		errs = append(errs, FieldError{Field: "minConfidence", Issue: "minConfidence must be between 0 and 1"})
	}
	return errs
}
