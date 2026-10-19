// Package authn is a Reference authentication boundary. It does not issue or
// store credentials. A delivery plugin supplies a Verifier; this package only
// enforces the boundary around its result.
package authn

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnauthenticated  = errors.New("AUTHENTICATION_REQUIRED")
	ErrInvalidPrincipal = errors.New("INVALID_PRINCIPAL")
	ErrTenantMismatch   = errors.New("TENANT_MISMATCH")
)

type Principal struct {
	TenantID string   `json:"tenantId"`
	Subject  string   `json:"subject"`
	Roles    []string `json:"roles,omitempty"`
}

// Verifier is implemented by a delivery plugin (for example, an OIDC adapter).
type Verifier interface {
	Verify(ctx context.Context, credential string) (Principal, error)
}

// Authenticate rejects an absent credential, an absent tenant, a verifier
// failure, an incomplete principal, and a principal bound to another tenant.
// It never returns an anonymous principal.
func Authenticate(ctx context.Context, verifier Verifier, credential, tenantID string) (Principal, error) {
	credential = strings.TrimSpace(credential)
	tenantID = strings.TrimSpace(tenantID)
	if credential == "" || tenantID == "" {
		return Principal{}, ErrUnauthenticated
	}
	if verifier == nil {
		return Principal{}, fmt.Errorf("verifier is required")
	}
	principal, err := verifier.Verify(ctx, credential)
	if err != nil {
		return Principal{}, ErrUnauthenticated
	}
	if strings.TrimSpace(principal.TenantID) == "" || strings.TrimSpace(principal.Subject) == "" {
		return Principal{}, ErrInvalidPrincipal
	}
	if principal.TenantID != tenantID {
		return Principal{}, ErrTenantMismatch
	}
	return principal, nil
}
