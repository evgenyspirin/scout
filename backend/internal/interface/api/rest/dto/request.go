// Package dto holds request DTOs.
package dto

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// UploadLinkRequest is the body of POST /photos/{id}/upload-link.
type UploadLinkRequest struct {
	ContentType string `json:"contentType"`
}
