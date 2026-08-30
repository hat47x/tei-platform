package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"tei/internal/core"
	"tei/internal/permission"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

type example struct {
	AcquireRequestFingerprint string                  `json:"acquireRequestFingerprint"`
	LeasePolicy               resource.LeasePolicy    `json:"leasePolicy"`
	InitialLease              resource.Lease          `json:"initialLease"`
	InitialSandboxProfile     pruntime.SandboxProfile `json:"initialSandboxProfile"`
	RefreshedLease            resource.Lease          `json:"refreshedLease"`
	RefreshedSandboxProfile   pruntime.SandboxProfile `json:"refreshedSandboxProfile"`
	RenewedLease              resource.Lease          `json:"renewedLease"`
	RenewedSandboxProfile     pruntime.SandboxProfile `json:"renewedSandboxProfile"`
	DurableResources          []resource.Record       `json:"durableResources"`
	Refresh                   resource.RefreshRecord  `json:"refresh"`
	Renewal                   resource.RenewalRecord  `json:"renewal"`
	ActiveLeasesAfterRelease  int                     `json:"activeLeasesAfterRelease"`
	BoundaryNotes             []string                `json:"boundaryNotes"`
}

func main() {
	zone := time.FixedZone("JST", 9*60*60)
	now := time.Date(2026, 8, 29, 17, 30, 0, 0, zone)
	secret := core.MustCanonicalID("production.sales_core.migration_credential")
	network := core.MustCanonicalID("production.sales_core.database_endpoint")
	workdir := core.MustCanonicalID("production.sales_core.migration_workdir")
	broker := &resource.AdapterBroker{TTL: 10 * time.Minute, Now: func() time.Time { return now }, Adapters: []resource.Adapter{
		&resource.ReferenceSecretManagerAdapter{Handles: map[core.CanonicalID]string{secret: "secret-manager://reference/sales-core"}},
		&resource.StaticAdapter{ResourceKind: permission.KindNetwork, Entries: []resource.StaticResource{{ResourceID: network, NetworkEndpoint: "sales-db.production.internal:5432"}}},
		&resource.StaticAdapter{ResourceKind: permission.KindFilesystem, Entries: []resource.StaticResource{{ResourceID: workdir, MountSourceHandle: "mount-source://reference/migration-workdir", MountPath: "/tei/migration"}}},
	}}
	perms := []permission.Spec{
		{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"},
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "database-endpoint"},
		{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: "migration-workdir"},
	}
	bindings := []resource.Binding{
		{Kind: permission.KindSecret, Scope: "database-credential", ResourceID: secret, SecretDelivery: resource.SecretDeliveryEnvironment, SecretTarget: "TEI_DATABASE_CREDENTIAL", Refresh: resource.RefreshLive},
		{Kind: permission.KindNetwork, Scope: "database-endpoint", ResourceID: network},
		{Kind: permission.KindFilesystem, Scope: "migration-workdir", ResourceID: workdir},
	}
	policy := resource.LeasePolicy{MaxLifetime: "45m", RenewBefore: "5m", RevokeTimeout: "15s", RenewalFailure: resource.RenewalFailClosed}
	req := resource.AcquireRequest{Environment: core.MustCanonicalID("production"), CapabilityKind: "deployment-executor", CapabilityID: core.MustCanonicalID("migration.atlas"), RuntimePolicyFingerprint: "reference-runtime-policy-m0.19", Permissions: perms, Bindings: bindings, LeasePolicy: policy}
	initial, err := broker.Acquire(context.Background(), req)
	if err != nil {
		panic(err)
	}
	refreshed, err := broker.Refresh(context.Background(), req, initial)
	if err != nil {
		panic(err)
	}
	refresh := resource.RefreshRecord{Revision: refreshed.Revision, RefreshedAt: now.Format(time.RFC3339), Scopes: []string{"database-credential"}}
	now = now.Add(5 * time.Minute)
	renewed, err := broker.Renew(context.Background(), req, refreshed)
	if err != nil {
		panic(err)
	}
	renewal := resource.RenewalRecord{Revision: renewed.Revision, RenewedAt: now.Format(time.RFC3339), ExpiresAt: renewed.ExpiresAt}
	if err := broker.Release(context.Background(), renewed); err != nil {
		panic(err)
	}
	out := example{
		AcquireRequestFingerprint: resource.AcquireRequestFingerprint(req), LeasePolicy: policy,
		InitialLease: initial, InitialSandboxProfile: pruntime.SandboxProfileForLease(&initial),
		RefreshedLease: refreshed, RefreshedSandboxProfile: pruntime.SandboxProfileForLease(&refreshed),
		RenewedLease: renewed, RenewedSandboxProfile: pruntime.SandboxProfileForLease(&renewed),
		DurableResources: resource.Records(renewed.Resources), Refresh: refresh, Renewal: renewal,
		ActiveLeasesAfterRelease: broker.ActiveLeaseCount(),
		BoundaryNotes: []string{
			"Renew extends time only; Refresh changes concrete resource view only and never extends expiry.",
			"Only the secret binding is refresh=live. Network endpoint and mount view remain unchanged.",
			"Host-side opaque handles/endpoints/mounts in this example are intentionally excluded from Durable Run provenance.",
			"M0.19 fail-closed semantics make an ignored Renew/Refresh failure fail the whole invocation.",
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
