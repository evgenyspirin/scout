// Package authapp contains authentication use cases (login → JWT).
// The UserRepository and TokenSigner interfaces are declared here in the
// consumer package per Go convention.
package authapp

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"scout/internal/domain/user"
)

// invalidCredentialsError is a typed error (no global sentinel var).
type invalidCredentialsError struct{}

func (invalidCredentialsError) Error() string { return "invalid credentials" }

// IsInvalidCredentials reports whether err is an invalid-credentials error.
func IsInvalidCredentials(err error) bool {
	var e invalidCredentialsError
	return errors.As(err, &e)
}

// UserRepository looks up users by login.
type UserRepository interface {
	FindByLogin(ctx context.Context, login string) (user.User, bool)
}

// TokenSigner issues signed JWT access tokens.
type TokenSigner interface {
	Generate(userID, role string, ttl time.Duration) (string, error)
}

// Service implements the login use case.
type Service struct {
	users  UserRepository
	signer TokenSigner
	ttl    time.Duration
}

// NewService builds an auth Service.
func NewService(users UserRepository, signer TokenSigner, ttl time.Duration) *Service {
	return &Service{users: users, signer: signer, ttl: ttl}
}

// Login verifies credentials and returns a signed JWT on success.
func (s *Service) Login(ctx context.Context, login, password string) (string, error) {
	u, ok := s.users.FindByLogin(ctx, login)
	if !ok {
		// Run a bcrypt comparison against a dummy hash to reduce timing leaks.
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$invalidinvalidinvalidinvalidinvalidinvalidinv"), []byte(password))
		return "", invalidCredentialsError{}
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", invalidCredentialsError{}
	}
	return s.signer.Generate(u.ID, string(u.Role), s.ttl)
}
