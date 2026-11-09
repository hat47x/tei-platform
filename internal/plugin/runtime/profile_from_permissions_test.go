package runtime_test

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/permission"
	pruntime "tei/internal/plugin/runtime"
)

func TestProfileFromPermissionsDerivesNetworkAndSecrets(t *testing.T) {
	profile, notes := pruntime.ProfileFromPermissions([]permission.Spec{
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"},
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "api:443"},
		{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:db"},
	})
	if !profile.DenyNetworkByDefault || !profile.DenyFilesystemByDefault {
		t.Fatalf("both policies must deny by default: %+v", profile)
	}
	if want := []string{"api:443", "db:5432"}; !reflect.DeepEqual(profile.NetworkAllowlist, want) {
		t.Fatalf("allowlist = %v, want %v", profile.NetworkAllowlist, want)
	}
	if want := []string{"vault:db"}; !reflect.DeepEqual(profile.SecretHandles, want) {
		t.Fatalf("secret handles = %v, want %v", profile.SecretHandles, want)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "secret injections") {
		t.Fatalf("secret injections must be reported as underived: %+v", notes)
	}
}

func TestProfileFromPermissionsReportsFilesystemAsUnderived(t *testing.T) {
	profile, notes := pruntime.ProfileFromPermissions([]permission.Spec{
		{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/var/reports"},
	})
	if len(profile.Mounts) != 0 {
		t.Fatalf("a permission cannot describe a mount: %+v", profile.Mounts)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "mounts are not derived") {
		t.Fatalf("the underived mount must be reported: %+v", notes)
	}
}

func TestProfileFromPermissionsWithNoDeclarationsDeniesAll(t *testing.T) {
	profile, notes := pruntime.ProfileFromPermissions(nil)
	if !profile.DenyNetworkByDefault || !profile.DenyFilesystemByDefault {
		t.Fatalf("no declarations means deny: %+v", profile)
	}
	if len(profile.NetworkAllowlist) != 0 || len(profile.SecretHandles) != 0 || len(notes) != 0 {
		t.Fatalf("nothing to allow and nothing to report: %+v %+v", profile, notes)
	}
}
