package runtime_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/plugin/runtime"
	"tei/internal/resource"
)

// The sandbox profile is the contract handed to a process sandbox, so it must
// be deny-by-default and grant only what the lease names. A profile that let
// network or filesystem through when the lease said nothing would hand every
// plugin the host's own access, which is the opposite of what a lease is for.
func TestASandboxProfileDeniesByDefault(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lease *resource.Lease
	}{
		{name: "no lease", lease: nil},
		{name: "an empty lease", lease: &resource.Lease{ID: "lease-1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := runtime.SandboxProfileForLease(tc.lease)
			if !p.DenyNetworkByDefault {
				t.Fatal("network must be denied by default")
			}
			if !p.DenyFilesystemByDefault {
				t.Fatal("the filesystem must be denied by default")
			}
			if len(p.NetworkAllowlist) != 0 || len(p.Mounts) != 0 ||
				len(p.SecretHandles) != 0 || len(p.SecretInjections) != 0 {
				t.Fatalf("nothing may be granted without a resource: %+v", p)
			}
		})
	}
}

// Each leased resource becomes exactly one allowance of its own kind, and only
// its own kind: a network grant must not produce a mount, and a filesystem
// grant must not open the network.
func TestSandboxProfileGrantsOnlyTheLeasedKinds(t *testing.T) {
	lease := &resource.Lease{
		ID: "lease-1",
		Resources: []resource.Resolved{
			{
				Kind:            permission.KindNetwork,
				Access:          permission.AccessConnect,
				Scope:           "service",
				ResourceID:      core.MustCanonicalID("api.example"),
				NetworkEndpoint: "api.example:443",
			},
			{
				Kind:              permission.KindFilesystem,
				Access:            permission.AccessRead,
				Scope:             "data",
				ResourceID:        core.MustCanonicalID("data.orders"),
				MountSourceHandle: "handle-1",
				MountPath:         "/data/orders",
			},
			{
				Kind:         permission.KindSecret,
				Access:       permission.AccessRead,
				Scope:        "service",
				ResourceID:   core.MustCanonicalID("api.token"),
				SecretHandle: "secret-1",
				SecretTarget: "API_TOKEN",
			},
		},
	}
	p := runtime.SandboxProfileForLease(lease)

	if got := p.NetworkAllowlist; len(got) != 1 || got[0] != "api.example:443" {
		t.Fatalf("network allowlist = %v", got)
	}
	if len(p.Mounts) != 1 {
		t.Fatalf("want one mount, got %+v", p.Mounts)
	}
	if p.Mounts[0].Target != "/data/orders" || p.Mounts[0].Access != permission.AccessRead {
		t.Fatalf("mount = %+v", p.Mounts[0])
	}
	if got := p.SecretHandles; len(got) != 1 || got[0] != "secret-1" {
		t.Fatalf("secret handles = %v", got)
	}
	if len(p.SecretInjections) != 1 || p.SecretInjections[0].Target != "API_TOKEN" {
		t.Fatalf("secret injections = %+v", p.SecretInjections)
	}

	// Deny-by-default stays set even when something is granted: the profile
	// describes the sandbox's baseline, and the lists are its exceptions.
	if !p.DenyNetworkByDefault || !p.DenyFilesystemByDefault {
		t.Fatal("a profile with grants must still deny by default")
	}

	// A filesystem grant must not have opened the network beyond its own entry.
	if len(p.NetworkAllowlist) != 1 {
		t.Fatalf("a filesystem grant added a network allowance: %v", p.NetworkAllowlist)
	}
}

// The profile is assembled from a sorted view, so two leases that list the same
// resources in different orders produce the same profile. Without that, the
// sandbox would differ by input order alone.
func TestSandboxProfileIsOrderIndependent(t *testing.T) {
	a := &resource.Lease{ID: "l", Resources: []resource.Resolved{
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, ResourceID: core.MustCanonicalID("alpha"), NetworkEndpoint: "zed:443"},
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, ResourceID: core.MustCanonicalID("zulu"), NetworkEndpoint: "alpha:443"},
	}}
	b := &resource.Lease{ID: "l", Resources: []resource.Resolved{
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, ResourceID: core.MustCanonicalID("zulu"), NetworkEndpoint: "alpha:443"},
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, ResourceID: core.MustCanonicalID("alpha"), NetworkEndpoint: "zed:443"},
	}}
	pa := runtime.SandboxProfileForLease(a)
	pb := runtime.SandboxProfileForLease(b)
	if len(pa.NetworkAllowlist) != 2 || len(pb.NetworkAllowlist) != 2 {
		t.Fatalf("allowlists = %v / %v", pa.NetworkAllowlist, pb.NetworkAllowlist)
	}
	for i := range pa.NetworkAllowlist {
		if pa.NetworkAllowlist[i] != pb.NetworkAllowlist[i] {
			t.Fatalf("the profile depends on resource order: %v vs %v", pa.NetworkAllowlist, pb.NetworkAllowlist)
		}
	}

	// The two inputs normalize to the same order, so the comparison above cannot
	// tell whether the endpoint list is sorted on its own. Assert that directly:
	// the resource IDs order alpha before zulu, while the endpoints are
	// zed:443 and alpha:443, so an unsorted list would keep zed:443 first.
	want := []string{"alpha:443", "zed:443"}
	for i := range want {
		if pa.NetworkAllowlist[i] != want[i] {
			t.Fatalf("the allowlist must be sorted by endpoint, got %v", pa.NetworkAllowlist)
		}
	}
}

// A kind the profile does not model grants nothing. An unmodelled permission
// must not be silently widened into network or filesystem access.
func TestAnUnmodelledResourceGrantsNothing(t *testing.T) {
	lease := &resource.Lease{ID: "l", Resources: []resource.Resolved{{
		Kind:       permission.Kind("unknown"),
		Access:     permission.AccessRead,
		ResourceID: core.MustCanonicalID("thing"),
	}}}
	p := runtime.SandboxProfileForLease(lease)
	if len(p.NetworkAllowlist) != 0 || len(p.Mounts) != 0 || len(p.SecretHandles) != 0 {
		t.Fatalf("an unmodelled kind produced an allowance: %+v", p)
	}
	if !p.DenyNetworkByDefault || !p.DenyFilesystemByDefault {
		t.Fatal("an unmodelled kind must not relax the default denial")
	}
}
