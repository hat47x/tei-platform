//go:build linux

package runtime

import (
	"context"
	"crypto/ed25519"
	"errors"
	"testing"
)

type testSignerAuthorizer struct {
	allow bool
	calls int
}

func (a *testSignerAuthorizer) AuthorizeSignerChange(_ context.Context, principal, operation, ref string) error {
	a.calls++
	if !a.allow || principal != "host-admin" || ref != "signed/tool" {
		return errors.New("denied")
	}
	if operation != "allow" && operation != "revoke" {
		return errors.New("unsupported operation")
	}
	return nil
}

func TestLinuxSignerAdminRequiresHostAuthorization(t *testing.T) {
	trust := NewLinuxHostSignerTrust()
	auth := &testSignerAuthorizer{}
	admin := LinuxSignerAdmin{Trust: trust, Authorizer: auth}
	req := LinuxSignerAdminRequest{SignerRef: "signed/tool", PublicKey: make(ed25519.PublicKey, ed25519.PublicKeySize)}
	if err := admin.Allow(context.Background(), "host-admin", req); err == nil {
		t.Fatal("unauthorized registration succeeded")
	}
	if trust.current(req.SignerRef, 1) {
		t.Fatal("denied registration mutated trust")
	}
	auth.allow = true
	if err := admin.Allow(context.Background(), "", req); err == nil {
		t.Fatal("missing verified principal was accepted")
	}
	if auth.calls != 1 {
		t.Fatal("missing principal should not call authorizer")
	}
	if err := admin.Allow(context.Background(), "host-admin", req); err != nil {
		t.Fatal(err)
	}
	_, epoch, err := trust.resolve(req.SignerRef)
	if err != nil || epoch == 0 {
		t.Fatalf("authorized registration failed: epoch=%d err=%v", epoch, err)
	}
	auth.allow = false
	if changed, err := admin.Revoke(context.Background(), "host-admin", req.SignerRef); err == nil || changed {
		t.Fatal("unauthorized revocation succeeded")
	}
	if !trust.current(req.SignerRef, epoch) {
		t.Fatal("denied revocation mutated trust")
	}
	auth.allow = true
	if changed, err := admin.Revoke(context.Background(), "host-admin", req.SignerRef); err != nil || !changed {
		t.Fatalf("authorized revocation failed: changed=%v err=%v", changed, err)
	}
	if trust.current(req.SignerRef, epoch) {
		t.Fatal("revocation left old authorization current")
	}
}

func TestLinuxSignerAdminFailsClosed(t *testing.T) {
	trust := NewLinuxHostSignerTrust()
	req := LinuxSignerAdminRequest{SignerRef: "signed/tool", PublicKey: make(ed25519.PublicKey, ed25519.PublicKeySize)}
	if err := (LinuxSignerAdmin{Trust: trust}).Allow(context.Background(), "host-admin", req); err == nil {
		t.Fatal("missing authorizer accepted")
	}
	if err := (LinuxSignerAdmin{Authorizer: &testSignerAuthorizer{allow: true}}).Allow(context.Background(), "host-admin", req); err == nil {
		t.Fatal("missing trust store accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (LinuxSignerAdmin{Trust: trust, Authorizer: &testSignerAuthorizer{allow: true}}).Allow(ctx, "host-admin", req); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context was not rejected: %v", err)
	}
	if _, _, err := trust.resolve(req.SignerRef); err == nil {
		t.Fatal("invalid requests modified trust")
	}
}
