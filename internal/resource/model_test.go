package resource_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/resource"
)

func TestValidateLeaseUsesOpaqueLeastAuthorityResources(t *testing.T) {
	perms := []permission.Spec{
		{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"},
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "database-endpoint"},
		{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: "migration-workdir"},
	}
	bindings := []resource.Binding{
		{Kind: permission.KindSecret, Scope: "database-credential", ResourceID: core.MustCanonicalID("prod.db.secret")},
		{Kind: permission.KindNetwork, Scope: "database-endpoint", ResourceID: core.MustCanonicalID("prod.db.endpoint")},
		{Kind: permission.KindFilesystem, Scope: "migration-workdir", ResourceID: core.MustCanonicalID("prod.db.workdir")},
	}
	req := resource.AcquireRequest{Environment: core.MustCanonicalID("production"), CapabilityKind: "deployment-executor", CapabilityID: core.MustCanonicalID("migration.atlas"), Permissions: perms, Bindings: bindings}
	lease := resource.Lease{ID: "lease-1", Resources: []resource.Resolved{
		{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential", ResourceID: core.MustCanonicalID("prod.db.secret"), SecretHandle: "secret-handle://lease-1/credential"},
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "database-endpoint", ResourceID: core.MustCanonicalID("prod.db.endpoint"), NetworkEndpoint: "db.internal:5432"},
		{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: "migration-workdir", ResourceID: core.MustCanonicalID("prod.db.workdir"), MountPath: "/tei/migration"},
	}}
	if err := resource.ValidateLease(req, lease); err != nil {
		t.Fatal(err)
	}
	records := resource.Records(lease.Resources)
	if len(records) != 3 {
		t.Fatalf("records: %#v", records)
	}
}

func TestValidateLeaseRejectsExtraOrWrongConcreteResource(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"}
	req := resource.AcquireRequest{Environment: core.MustCanonicalID("production"), CapabilityKind: "deployment-executor", CapabilityID: core.MustCanonicalID("migration.atlas"), Permissions: []permission.Spec{perm}, Bindings: []resource.Binding{{Kind: perm.Kind, Scope: perm.Scope, ResourceID: core.MustCanonicalID("prod.db.secret")}}}
	bad := resource.Lease{ID: "lease-1", Resources: []resource.Resolved{{Kind: perm.Kind, Access: perm.Access, Scope: perm.Scope, ResourceID: core.MustCanonicalID("prod.db.secret"), SecretHandle: "opaque", NetworkEndpoint: "should-not-exist"}}}
	if err := resource.ValidateLease(req, bad); err == nil {
		t.Fatal("expected secret resource shape rejection")
	}
}

func TestCatalogBrokerAcquiresAndRevokesInvocationScopedLease(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "database-endpoint"}
	binding := resource.Binding{Kind: permission.KindNetwork, Scope: perm.Scope, ResourceID: core.MustCanonicalID("prod.db.endpoint")}
	b := &resource.CatalogBroker{Entries: []resource.CatalogEntry{{ResourceID: binding.ResourceID, Kind: permission.KindNetwork, NetworkEndpoint: "db.internal:5432"}}}
	req := resource.AcquireRequest{Environment: core.MustCanonicalID("production"), CapabilityKind: "deployment-executor", CapabilityID: core.MustCanonicalID("migration.atlas"), Permissions: []permission.Spec{perm}, Bindings: []resource.Binding{binding}}
	lease, err := b.Acquire(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if b.ActiveLeaseCount() != 1 || lease.Resources[0].NetworkEndpoint != "db.internal:5432" {
		t.Fatalf("lease: %#v active=%d", lease, b.ActiveLeaseCount())
	}
	if err := b.Release(context.Background(), lease); err != nil {
		t.Fatal(err)
	}
	if b.ActiveLeaseCount() != 0 {
		t.Fatalf("lease was not revoked")
	}
}
