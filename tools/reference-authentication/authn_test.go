package authn

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fixtureVerifier struct {
	principal Principal
	err       error
}

func (f fixtureVerifier) Verify(context.Context, string) (Principal, error) {
	return f.principal, f.err
}

func TestAuthenticateValidPrincipal(t *testing.T) {
	principal, err := Authenticate(context.Background(), fixtureVerifier{
		principal: Principal{TenantID: "tenant-a", Subject: "user-1"},
	}, "fixture-token", "tenant-a")
	if err != nil || principal.Subject != "user-1" {
		t.Fatalf("principal=%+v err=%v", principal, err)
	}
}

func TestMissingCredentialOrTenantIsRejected(t *testing.T) {
	verifier := fixtureVerifier{principal: Principal{TenantID: "tenant-a", Subject: "user-1"}}
	for _, tc := range []struct{ credential, tenant string }{
		{"", "tenant-a"}, {"fixture-token", ""}, {"  ", "  "},
	} {
		if _, err := Authenticate(context.Background(), verifier, tc.credential, tc.tenant); !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("credential=%q tenant=%q err=%v", tc.credential, tc.tenant, err)
		}
	}
}

func TestVerifierFailureDoesNotLeakCredential(t *testing.T) {
	_, err := Authenticate(context.Background(), fixtureVerifier{err: errors.New("provider detail")}, "secret-token", "tenant-a")
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("want AUTHENTICATION_REQUIRED, got %v", err)
	}
	if strings.Contains(err.Error(), "secret-token") || strings.Contains(err.Error(), "provider detail") {
		t.Fatalf("error leaked verifier detail: %v", err)
	}
}

func TestPrincipalFromAnotherTenantIsRejected(t *testing.T) {
	_, err := Authenticate(context.Background(), fixtureVerifier{
		principal: Principal{TenantID: "tenant-b", Subject: "user-1"},
	}, "fixture-token", "tenant-a")
	if !errors.Is(err, ErrTenantMismatch) {
		t.Fatalf("want TENANT_MISMATCH, got %v", err)
	}
}

func TestIncompletePrincipalIsRejected(t *testing.T) {
	for _, principal := range []Principal{
		{TenantID: "tenant-a"},
		{Subject: "user-1"},
	} {
		if _, err := Authenticate(context.Background(), fixtureVerifier{principal: principal}, "fixture-token", "tenant-a"); !errors.Is(err, ErrInvalidPrincipal) {
			t.Fatalf("principal=%+v err=%v", principal, err)
		}
	}
}

func TestNilVerifierIsRejected(t *testing.T) {
	if _, err := Authenticate(context.Background(), nil, "fixture-token", "tenant-a"); err == nil {
		t.Fatal("nil verifier was accepted")
	}
}
