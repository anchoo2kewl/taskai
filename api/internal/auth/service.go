package auth

import (
	"time"
)

// Service provides authentication operations
type Service struct {
	jwtSecret string
	expiry    time.Duration
}

// NewService creates a new auth service
func NewService(jwtSecret string, expiry time.Duration) *Service {
	return &Service{
		jwtSecret: jwtSecret,
		expiry:    expiry,
	}
}

// GenerateToken creates a new JWT token for a user using service config
func (s *Service) GenerateToken(userID int64, email string) (string, error) {
	return GenerateToken(userID, email, s.jwtSecret, s.expiry)
}

// ValidateToken validates a JWT token and returns the claims using service config
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	return ValidateToken(tokenString, s.jwtSecret)
}
