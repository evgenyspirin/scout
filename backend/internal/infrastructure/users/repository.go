// Package users provides an in-memory, code-seeded user repository.
// The two required users (insect/admin) are seeded with bcrypt-hashed
// passwords at construction time.
package users

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"scout/internal/domain/user"
)

// Repository is an in-memory user store keyed by login.
type Repository struct {
	byLogin map[string]user.User
}

// Credential describes a user to seed.
type Credential struct {
	Login    string
	Password string
	Role     user.Role
}

// NewRepository seeds the repository with the given credentials.
func NewRepository(creds []Credential) (*Repository, error) {
	byLogin := make(map[string]user.User, len(creds))
	for _, c := range creds {
		hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		byLogin[c.Login] = user.User{
			ID:           c.Login,
			Login:        c.Login,
			PasswordHash: string(hash),
			Role:         c.Role,
		}
	}
	return &Repository{byLogin: byLogin}, nil
}

// DefaultCredentials returns the two users required by the assignment.
func DefaultCredentials() []Credential {
	return []Credential{
		{Login: "insect", Password: "insect123", Role: user.RoleUser},
		{Login: "admin", Password: "admin123", Role: user.RoleAdmin},
	}
}

// FindByLogin returns the user with the given login.
func (r *Repository) FindByLogin(_ context.Context, login string) (user.User, bool) {
	u, ok := r.byLogin[login]
	return u, ok
}
