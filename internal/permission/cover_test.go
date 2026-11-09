package permission_test

import (
	"testing"

	"tei/internal/permission"
)

func spec(kind permission.Kind, access permission.Access, scope string) permission.Spec {
	return permission.Spec{Kind: kind, Access: access, Scope: scope}
}

func TestCoversMatchesKindAccessAndScope(t *testing.T) {
	readWrite := spec(permission.KindFilesystem, permission.AccessReadWrite, "/var/data")
	cases := []struct {
		name   string
		grant  permission.Spec
		req    permission.Spec
		covers bool
	}{
		{"read-write covers read", readWrite, spec(permission.KindFilesystem, permission.AccessRead, "/var/data"), true},
		{"read-write covers write", readWrite, spec(permission.KindFilesystem, permission.AccessWrite, "/var/data"), true},
		{"read does not cover write", spec(permission.KindFilesystem, permission.AccessRead, "/var/data"), spec(permission.KindFilesystem, permission.AccessWrite, "/var/data"), false},
		{"kind must match", spec(permission.KindNetwork, permission.AccessConnect, "db:5432"), spec(permission.KindFilesystem, permission.AccessRead, "db:5432"), false},
		{"child scope is covered", readWrite, spec(permission.KindFilesystem, permission.AccessRead, "/var/data/invoices"), true},
		{"sibling name is not covered", readWrite, spec(permission.KindFilesystem, permission.AccessRead, "/var/database"), false},
		{"connect is exact", spec(permission.KindNetwork, permission.AccessConnect, "db:5432"), spec(permission.KindNetwork, permission.AccessConnect, "db:5432"), true},
		{"empty scope covers nothing", spec(permission.KindFilesystem, permission.AccessRead, ""), spec(permission.KindFilesystem, permission.AccessRead, "/var/data"), false},
	}
	for _, test := range cases {
		if got := permission.Covers(test.grant, test.req); got != test.covers {
			t.Errorf("%s: Covers = %v, want %v", test.name, got, test.covers)
		}
	}
}

func TestMissingKeepsEveryUncoveredRequirement(t *testing.T) {
	granted := []permission.Spec{
		spec(permission.KindFilesystem, permission.AccessRead, "/var/data"),
	}
	required := []permission.Spec{
		spec(permission.KindFilesystem, permission.AccessRead, "/var/data/invoices"),
		spec(permission.KindNetwork, permission.AccessConnect, "db:5432"),
	}
	missing := permission.Missing(granted, required)
	if len(missing) != 1 {
		t.Fatalf("only the network requirement is missing: %+v", missing)
	}
	if missing[0].Kind != permission.KindNetwork {
		t.Fatalf("unexpected missing permission: %+v", missing[0])
	}
}

func TestMissingIsEmptyWhenEveryRequirementIsGranted(t *testing.T) {
	required := []permission.Spec{spec(permission.KindFilesystem, permission.AccessRead, "/var/data/invoices")}
	granted := []permission.Spec{spec(permission.KindFilesystem, permission.AccessReadWrite, "/var/data")}
	if missing := permission.Missing(granted, required); len(missing) != 0 {
		t.Fatalf("nothing is missing: %+v", missing)
	}
}
