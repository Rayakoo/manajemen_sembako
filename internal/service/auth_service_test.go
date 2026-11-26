package service

import "testing"

func TestOAuthStateLifecycle(t *testing.T) {
	svc := NewAuthService(nil, "test-jwt-secret", 24)

	state, err := svc.GenerateOAuthState()
	if err != nil {
		t.Fatalf("GenerateOAuthState() error = %v", err)
	}
	if state == "" {
		t.Fatalf("expected non-empty state")
	}

	if err := svc.ValidateOAuthState(state, state); err != nil {
		t.Fatalf("ValidateOAuthState() with matching state failed: %v", err)
	}

	if err := svc.ValidateOAuthState(state, state+"tampered"); err == nil {
		t.Fatalf("expected mismatch error, got nil")
	}

	otherSvc := NewAuthService(nil, "different-secret", 24)
	if err := otherSvc.ValidateOAuthState(state, state); err == nil {
		t.Fatalf("expected invalid signature error, got nil")
	}
}
