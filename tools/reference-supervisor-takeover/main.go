package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/resource"
)

type result struct {
	HostA                resource.SupervisorOwnership `json:"hostA"`
	HostB                resource.SupervisorOwnership `json:"hostB"`
	Takeover             bool                         `json:"takeover"`
	ActiveLease          resource.Lease               `json:"activeLease"`
	StaleRenewRejected   bool                         `json:"staleRenewRejected"`
	StaleRefreshRejected bool                         `json:"staleRefreshRejected"`
	StaleReleaseRejected bool                         `json:"staleReleaseRejected"`
	BoundaryNotes        []string                     `json:"boundaryNotes"`
}

func main() {
	ctx := context.Background()
	now := time.Date(2026, 8, 29, 22, 0, 0, 0, time.UTC)
	ownerships := &resource.MemoryOwnershipStore{Now: func() time.Time { return now }}
	secretID := core.MustCanonicalID("production.sales_core.migration_credential")
	broker := &resource.AdapterBroker{TTL: 10 * time.Minute, Now: func() time.Time { return now }, Adapters: []resource.Adapter{
		&resource.StaticAdapter{ResourceKind: permission.KindSecret, Entries: []resource.StaticResource{{ResourceID: secretID, SecretHandle: "opaque://not-durable"}}},
	}}
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"}
	req := resource.AcquireRequest{Environment: core.MustCanonicalID("production"), CapabilityKind: "deployment-executor", CapabilityID: core.MustCanonicalID("migration.atlas"), RuntimePolicyFingerprint: "reference-policy", Permissions: []permission.Spec{perm}, Bindings: []resource.Binding{{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: secretID}}, LeasePolicy: resource.LeasePolicy{MaxLifetime: "1h", RenewBefore: "9m", RevokeTimeout: "1s", RenewalFailure: resource.RenewalFailClosed}}
	lease, err := broker.Acquire(ctx, req)
	if err != nil {
		panic(err)
	}
	hostA, _, err := ownerships.Claim(ctx, lease.ID, "host-a", 30*time.Second)
	if err != nil {
		panic(err)
	}
	leaseA, err := broker.Fence(ctx, lease, hostA.FencingToken)
	if err != nil {
		panic(err)
	}

	now = now.Add(31 * time.Second)
	hostB, takeover, err := ownerships.Claim(ctx, lease.ID, "host-b", 30*time.Second)
	if err != nil {
		panic(err)
	}
	leaseB, err := broker.Fence(ctx, leaseA, hostB.FencingToken)
	if err != nil {
		panic(err)
	}

	staleRenewRejected := false
	if _, err := broker.Renew(ctx, req, leaseA); err != nil {
		staleRenewRejected = true
	}
	staleRefreshRejected := false
	if _, err := broker.Refresh(ctx, req, leaseA); err != nil {
		staleRefreshRejected = true
	}
	staleReleaseRejected := false
	if err := broker.Release(ctx, leaseA); err != nil {
		staleReleaseRejected = true
	}
	if err := broker.Release(ctx, leaseB); err != nil {
		panic(err)
	}
	if err := ownerships.Release(ctx, hostB); err != nil {
		panic(err)
	}

	out := result{HostA: hostA, HostB: hostB, Takeover: takeover, ActiveLease: leaseB, StaleRenewRejected: staleRenewRejected, StaleRefreshRejected: staleRefreshRejected, StaleReleaseRejected: staleReleaseRejected, BoundaryNotes: []string{
		"Ownership expiry and takeover are coordinated by a storage-neutral SupervisorOwnershipStore contract; this executable reference uses the in-memory implementation.",
		"The fencing token advances independently from resource lease revision, so supervisor generation is not confused with resource renewal/refresh history.",
		"The old host cannot Renew, Refresh, or Release after host-b installs the newer fencing token, even if host-a still holds an old lease copy.",
		"Existing process/session reattachment after host restart is intentionally deferred to a later Invocation Handle / Reattach contract.",
	}}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
