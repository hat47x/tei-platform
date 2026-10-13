//go:build linux

package runtime

import (
	"context"
	"crypto/ed25519"
	"errors"
)

// LinuxSignerAdminRequest is an operation request for a Host-owned trust store.
// The caller must not supply the authenticated principal as part of this request.
type LinuxSignerAdminRequest struct {
	SignerRef string
	PublicKey ed25519.PublicKey
}

// LinuxSignerAdminAuthorizer makes a Host-local authorization decision.
// A Host must construct principal from its verified authentication context,
// not from a plugin manifest or untrusted request field.
type LinuxSignerAdminAuthorizer interface {
	AuthorizeSignerChange(ctx context.Context, principal string, operation string, ref string) error
}

// LinuxSignerAdmin restricts trust-store mutations to a Host-owned policy
// boundary. Existing low-level Allow/Revoke remain available for Host internals;
// applications should expose only this adapter to their management endpoints.
type LinuxSignerAdmin struct {
	Trust      *LinuxHostSignerTrust
	Authorizer LinuxSignerAdminAuthorizer
}

func (a LinuxSignerAdmin) authorize(ctx context.Context, principal, operation, ref string) error {
	if a.Trust == nil || a.Authorizer == nil || principal == "" || ctx == nil {
		return errors.New("Linux signer administration requires a trusted Host principal and authorizer")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateLinuxSignatureRef(ref); err != nil {
		return err
	}
	return a.Authorizer.AuthorizeSignerChange(ctx, principal, operation, ref)
}

// Allow performs a Host-authorized registration or replacement. Every
// successful registration advances the trust-store epoch.
func (a LinuxSignerAdmin) Allow(ctx context.Context, principal string, req LinuxSignerAdminRequest) error {
	if err := a.authorize(ctx, principal, "allow", req.SignerRef); err != nil {
		return err
	}
	return a.Trust.Allow(req.SignerRef, req.PublicKey)
}

// Revoke rejects unauthorized requests without revealing whether a signer
// exists. The boolean reports whether the authorized operation changed trust.
func (a LinuxSignerAdmin) Revoke(ctx context.Context, principal, ref string) (bool, error) {
	if err := a.authorize(ctx, principal, "revoke", ref); err != nil {
		return false, err
	}
	return a.Trust.Revoke(ref), nil
}
