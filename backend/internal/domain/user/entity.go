// Package user contains the User entity and roles.
package user

// Role enumerates authorization roles.
type Role string

const (
	// RoleUser can browse photos.
	RoleUser Role = "user"
	// RoleAdmin can additionally create upload links and read ops endpoints.
	RoleAdmin Role = "admin"
)

// User is an authenticated principal.
type User struct {
	ID           string
	Login        string
	PasswordHash string
	Role         Role
}
