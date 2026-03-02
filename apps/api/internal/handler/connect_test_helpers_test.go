package handler

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
)

// requireConnectCode asserts that err is a *connect.Error with the given code.
func requireConnectCode(t *testing.T, err error, want connect.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %v, got nil", want)
	}
	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != want {
		t.Errorf("code = %v, want %v", ce.Code(), want)
	}
}
