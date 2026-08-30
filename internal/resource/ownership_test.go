package resource_test

import (
	"context"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/resource"
)

func TestSupervisorOwnershipTakeoverIssuesMonotonicFencingToken(t *testing.T) {
	now := time.Date(2026, 8, 29, 22, 0, 0, 0, time.UTC)
	store := &resource.MemoryOwnershipStore{Now: func() time.Time { return now }}
	a, takeover, err := store.Claim(context.Background(), "lease-1", "host-a", 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if takeover || a.FencingToken != 1 {
		t.Fatalf("first ownership=%#v takeover=%v", a, takeover)
	}
	if _, _, err := store.Claim(context.Background(), "lease-1", "host-b", 30*time.Second); err == nil {
		t.Fatal("expected live ownership to block second host")
	}
	now = now.Add(31 * time.Second)
	b, takeover, err := store.Claim(context.Background(), "lease-1", "host-b", 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !takeover || b.FencingToken != 2 {
		t.Fatalf("takeover ownership=%#v takeover=%v", b, takeover)
	}
	if err := store.Assert(context.Background(), a); err == nil {
		t.Fatal("expected old ownership to be fenced")
	}
	if _, err := store.Heartbeat(context.Background(), a, 30*time.Second); err == nil {
		t.Fatal("expected old heartbeat to be fenced")
	}
	if err := store.Release(context.Background(), a); err == nil {
		t.Fatal("expected old release to be fenced")
	}
	if err := store.Assert(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}

func TestAdapterBrokerRejectsStaleSupervisorAfterTakeoverFence(t *testing.T) {
	now := time.Date(2026, 8, 29, 22, 0, 0, 0, time.UTC)
	secretID := core.MustCanonicalID("production.sales_core.migration_credential")
	adapter := &resource.StaticAdapter{ResourceKind: permission.KindSecret, Entries: []resource.StaticResource{{ResourceID: secretID, SecretHandle: "secret://opaque"}}}
	broker := &resource.AdapterBroker{Adapters: []resource.Adapter{adapter}, TTL: 10 * time.Minute, Now: func() time.Time { return now }}
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"}
	req := resource.AcquireRequest{Environment: core.MustCanonicalID("production"), CapabilityKind: "deployment-executor", CapabilityID: core.MustCanonicalID("migration.atlas"), RuntimePolicyFingerprint: "policy", Permissions: []permission.Spec{perm}, Bindings: []resource.Binding{{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: secretID}}, LeasePolicy: resource.LeasePolicy{MaxLifetime: "1h", RenewBefore: "9m", RevokeTimeout: "1s", RenewalFailure: resource.RenewalFailClosed}}
	lease, err := broker.Acquire(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	hostA, err := broker.Fence(context.Background(), lease, 1)
	if err != nil {
		t.Fatal(err)
	}
	hostB, err := broker.Fence(context.Background(), hostA, 2)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute) // inside the deliberately broad renew window
	if _, err := broker.Renew(context.Background(), req, hostA); err == nil {
		t.Fatal("expected stale host renew to be fenced")
	}
	if err := broker.Release(context.Background(), hostA); err == nil {
		t.Fatal("expected stale host release to be fenced")
	}
	renewed, err := broker.Renew(context.Background(), req, hostB)
	if err != nil {
		t.Fatal(err)
	}
	if renewed.FencingToken != 2 {
		t.Fatalf("fencing token changed during renewal: %#v", renewed)
	}
	if err := broker.Release(context.Background(), renewed); err != nil {
		t.Fatal(err)
	}
}
