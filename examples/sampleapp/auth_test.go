package sampleapp_test

import (
	"testing"

	"forgejo-test-evaluator/examples/sampleapp"
)

func TestAuthenticateToken_Success(t *testing.T) {
	session, err := sampleapp.AuthenticateToken("Bearer valid_admin_token")
	if err != nil {
		t.Fatalf("expected valid token authentication to succeed, got %v", err)
	}

	if session.UserID != "usr_admin" || session.Role != "admin" {
		t.Errorf("unexpected user session: %+v", session)
	}
}

func TestAuthenticateToken_InvalidToken(t *testing.T) {
	_, err := sampleapp.AuthenticateToken("invalid_token")
	if err == nil {
		t.Fatal("expected error for invalid token format, got nil")
	}

	_, err = sampleapp.AuthenticateToken("Bearer expired_token_123")
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestAuthorizeAccess_AdminOverride(t *testing.T) {
	session := &sampleapp.UserSession{
		UserID: "usr_admin",
		Role:   "admin",
	}

	if err := sampleapp.AuthorizeAccess(session, "user"); err != nil {
		t.Errorf("expected admin access to succeed, got %v", err)
	}
}
