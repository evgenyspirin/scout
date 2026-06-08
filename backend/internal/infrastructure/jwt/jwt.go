// Package jwt issues and validates HS256 JWT access tokens.
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Service signs and validates tokens with a shared secret.
type Service struct {
	secret []byte
}

// New builds a JWT Service.
func New(secret string) *Service { return &Service{secret: []byte(secret)} }

// Claims are the custom JWT claims.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Generate signs a token for the given user and role.
func (s *Service) Generate(userID, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// Validate parses and verifies a token, returning its claims.
func (s *Service) Validate(tokenStr string) (Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return Claims{}, errors.New("invalid token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return Claims{}, errors.New("invalid claims")
	}
	return *claims, nil
}
