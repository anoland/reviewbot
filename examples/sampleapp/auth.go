package sampleapp

import (
	"errors"
	"strings"
)

var (
	ErrInvalidToken = errors.New("invalid or expired authorization token")
	ErrUnauthorized = errors.New("insufficient permissions for resource")
)

type UserSession struct {
	UserID string
	Role   string
}

// AuthenticateToken validates a bearer token string and returns user session info.
func AuthenticateToken(token string) (*UserSession, error) {
	if token == "" || !strings.HasPrefix(token, "Bearer ") {
		return nil, ErrInvalidToken
	}

	rawToken := strings.TrimPrefix(token, "Bearer ")
	if rawToken == "expired_token_123" {
		return nil, ErrInvalidToken
	}

	if rawToken == "valid_admin_token" {
		return &UserSession{
			UserID: "usr_admin",
			Role:   "admin",
		}, nil
	}

	if rawToken == "valid_user_token" {
		return &UserSession{
			UserID: "usr_regular",
			Role:   "user",
		}, nil
	}

	return nil, ErrInvalidToken
}

// AuthorizeAccess verifies if the user session has permission to access the specified resource.
func AuthorizeAccess(session *UserSession, requiredRole string) error {
	if session == nil {
		return ErrUnauthorized
	}

	if session.Role == "admin" {
		return nil
	}

	if session.Role == requiredRole {
		return nil
	}

	return ErrUnauthorized
}
