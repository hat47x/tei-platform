package runtime_test

import (
	"strings"
	"testing"

	"tei/internal/permission"
	pruntime "tei/internal/plugin/runtime"
)

func TestProfileWithinDeclarationsHasNoViolations(t *testing.T) {
	perms := []permission.Spec{
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"},
		{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:db"},
	}
	profile, _ := pruntime.ProfileFromPermissions(perms)
	if violations := pruntime.ProfileExceedsDeclarations(profile, perms); len(violations) != 0 {
		t.Fatalf("a derived profile must stay within the declarations: %+v", violations)
	}
}

func TestProfileWithAnUndeclaredDestinationIsReported(t *testing.T) {
	perms := []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}}
	profile := pruntime.SandboxProfile{
		DenyNetworkByDefault:    true,
		DenyFilesystemByDefault: true,
		NetworkAllowlist:        []string{"db:5432", "evil:6379"},
	}
	violations := pruntime.ProfileExceedsDeclarations(profile, perms)
	if len(violations) != 1 || !strings.Contains(violations[0], "evil:6379") {
		t.Fatalf("the undeclared destination must be reported: %+v", violations)
	}
}

func TestProfileWithoutDenyByDefaultIsReported(t *testing.T) {
	violations := pruntime.ProfileExceedsDeclarations(pruntime.SandboxProfile{}, nil)
	if len(violations) != 2 {
		t.Fatalf("both policies must deny by default: %+v", violations)
	}
}

func TestProfileWithAnUndeclaredMountIsReported(t *testing.T) {
	profile := pruntime.SandboxProfile{
		DenyNetworkByDefault:    true,
		DenyFilesystemByDefault: true,
		Mounts:                  []pruntime.SandboxMount{{Scope: "/var/secrets"}},
	}
	if violations := pruntime.ProfileExceedsDeclarations(profile, nil); len(violations) != 1 || !strings.Contains(violations[0], "/var/secrets") {
		t.Fatalf("the undeclared mount must be reported: %+v", violations)
	}
}
