package resource_test

import (
	"context"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/resource"
)

func TestAdapterBrokerRenewalKeepsLogicalAuthorityAndMayRotateHandle(t *testing.T) {
	now := time.Date(2026, 8, 29, 15, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	secretID := core.MustCanonicalID("production.sales_core.migration_credential")
	adapter := &resource.StaticAdapter{ResourceKind: permission.KindSecret, Entries: []resource.StaticResource{{ResourceID: secretID, SecretHandle: "vault-handle://sales-core"}}}
	broker := &resource.AdapterBroker{Adapters: []resource.Adapter{adapter}, TTL: 10 * time.Minute, Now: func() time.Time { return now }}
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"}
	binding := resource.Binding{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: secretID, SecretDelivery: resource.SecretDeliveryEnvironment, SecretTarget: "TEI_DATABASE_CREDENTIAL"}
	req := resource.AcquireRequest{Environment: core.MustCanonicalID("production"), CapabilityKind: "deployment-executor", CapabilityID: core.MustCanonicalID("migration.atlas"), RuntimePolicyFingerprint: "policy-1", Permissions: []permission.Spec{perm}, Bindings: []resource.Binding{binding}}

	lease1, err := broker.Acquire(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if lease1.Revision != 1 || lease1.Resources[0].SecretHandle != "vault-handle://sales-core" {
		t.Fatalf("initial lease: %#v", lease1)
	}
	now = now.Add(5 * time.Minute)
	lease2, err := broker.Renew(context.Background(), req, lease1)
	if err != nil {
		t.Fatal(err)
	}
	if lease2.ID != lease1.ID || lease2.Revision != 2 {
		t.Fatalf("renewed lease: before=%#v after=%#v", lease1, lease2)
	}
	if lease2.Resources[0].SecretHandle != lease1.Resources[0].SecretHandle {
		t.Fatalf("in-place renewal changed concrete secret handle: before=%#v after=%#v", lease1, lease2)
	}
	if got, want := resource.StableRecords(lease2), resource.StableRecords(lease1); len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("logical authority changed: before=%#v after=%#v", want, got)
	}

	badReq := req
	badReq.Bindings = []resource.Binding{{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: core.MustCanonicalID("production.other.secret"), SecretDelivery: resource.SecretDeliveryEnvironment, SecretTarget: "TEI_DATABASE_CREDENTIAL"}}
	if _, err := broker.Renew(context.Background(), badReq, lease2); err == nil {
		t.Fatal("expected renewal with changed resource identity to be rejected")
	}
	if _, err := broker.Renew(context.Background(), req, lease1); err == nil {
		t.Fatal("expected stale lease revision to be rejected")
	}

	rotating := &resource.StaticAdapter{ResourceKind: permission.KindSecret, RenewSuffix: "#r", Entries: []resource.StaticResource{{ResourceID: secretID, SecretHandle: "vault-handle://sales-core"}}}
	rotBroker := &resource.AdapterBroker{Adapters: []resource.Adapter{rotating}, TTL: 10 * time.Minute, Now: func() time.Time { return now }}
	rotLease, err := rotBroker.Acquire(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if _, err := rotBroker.Renew(context.Background(), req, rotLease); err == nil {
		t.Fatal("expected live renewal that rotates concrete secret handle to be rejected")
	}
	if err := rotBroker.Release(context.Background(), rotLease); err != nil {
		t.Fatal(err)
	}

	if err := broker.Release(context.Background(), lease2); err != nil {
		t.Fatal(err)
	}
	if broker.ActiveLeaseCount() != 0 {
		t.Fatal("lease was not released")
	}
}

func TestLeasePolicyRenewWindowMaxLifetimeAndLiveRefresh(t *testing.T) {
	zone := time.FixedZone("JST", 9*60*60)
	now := time.Date(2026, 8, 29, 17, 0, 0, 0, zone)
	secretID := core.MustCanonicalID("production.sales_core.rotating_credential")
	adapter := &resource.StaticAdapter{ResourceKind: permission.KindSecret, Entries: []resource.StaticResource{{ResourceID: secretID, SecretHandle: "vault://v1"}}, RefreshSuffix: "#v"}
	broker := &resource.AdapterBroker{Adapters: []resource.Adapter{adapter}, TTL: 10 * time.Minute, Now: func() time.Time { return now }}
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"}
	binding := resource.Binding{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: secretID, SecretDelivery: resource.SecretDeliveryEnvironment, SecretTarget: "TEI_DATABASE_CREDENTIAL", Refresh: resource.RefreshLive}
	policy := resource.LeasePolicy{MaxLifetime: "20m", RenewBefore: "3m", RevokeTimeout: "2s", RenewalFailure: resource.RenewalFailClosed}
	req := resource.AcquireRequest{Environment: core.MustCanonicalID("production"), CapabilityKind: "deployment-executor", CapabilityID: core.MustCanonicalID("migration.atlas"), RuntimePolicyFingerprint: "policy-19", Permissions: []permission.Spec{perm}, Bindings: []resource.Binding{binding}, LeasePolicy: policy}
	lease1, err := broker.Acquire(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got := lease1.ExpiresAt; got != now.Add(10*time.Minute).Format(time.RFC3339) {
		t.Fatalf("initial expiry=%s", got)
	}

	now = now.Add(5 * time.Minute)
	if _, err := broker.Renew(context.Background(), req, lease1); err == nil {
		t.Fatal("expected renewal before renew window to fail")
	}

	// Refresh rotates the concrete handle but preserves expiry and logical authority.
	refreshed, err := broker.Refresh(context.Background(), req, lease1)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.ExpiresAt != lease1.ExpiresAt || refreshed.Resources[0].SecretHandle == lease1.Resources[0].SecretHandle {
		t.Fatalf("refresh=%#v", refreshed)
	}

	now = time.Date(2026, 8, 29, 17, 8, 0, 0, zone)
	renewed, err := broker.Renew(context.Background(), req, refreshed)
	if err != nil {
		t.Fatal(err)
	}
	if renewed.ExpiresAt != time.Date(2026, 8, 29, 17, 18, 0, 0, zone).Format(time.RFC3339) {
		t.Fatalf("renewed expiry=%s", renewed.ExpiresAt)
	}

	now = time.Date(2026, 8, 29, 17, 17, 0, 0, zone)
	final, err := broker.Renew(context.Background(), req, renewed)
	if err != nil {
		t.Fatal(err)
	}
	if final.ExpiresAt != time.Date(2026, 8, 29, 17, 20, 0, 0, zone).Format(time.RFC3339) {
		t.Fatalf("max-lifetime cap not applied: %s", final.ExpiresAt)
	}

	now = time.Date(2026, 8, 29, 17, 19, 0, 0, zone)
	if _, err := broker.Renew(context.Background(), req, final); err == nil {
		t.Fatal("expected max-lifetime renewal rejection")
	}
	if err := broker.Release(context.Background(), final); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshRequiresExplicitLiveBinding(t *testing.T) {
	now := time.Date(2026, 8, 29, 17, 0, 0, 0, time.UTC)
	secretID := core.MustCanonicalID("production.static_credential")
	adapter := &resource.StaticAdapter{ResourceKind: permission.KindSecret, Entries: []resource.StaticResource{{ResourceID: secretID, SecretHandle: "vault://v1"}}, RefreshSuffix: "#v"}
	broker := &resource.AdapterBroker{Adapters: []resource.Adapter{adapter}, TTL: 5 * time.Minute, Now: func() time.Time { return now }}
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "credential"}
	req := resource.AcquireRequest{Environment: core.MustCanonicalID("production"), CapabilityKind: "deployment-executor", CapabilityID: core.MustCanonicalID("migration.atlas"), RuntimePolicyFingerprint: "policy", Permissions: []permission.Spec{perm}, Bindings: []resource.Binding{{Kind: permission.KindSecret, Scope: "credential", ResourceID: secretID}}, LeasePolicy: resource.DefaultLeasePolicy()}
	lease, err := broker.Acquire(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := broker.Refresh(context.Background(), req, lease); err == nil {
		t.Fatal("expected refresh without refresh live to fail")
	}
	if err := broker.Release(context.Background(), lease); err != nil {
		t.Fatal(err)
	}
}
