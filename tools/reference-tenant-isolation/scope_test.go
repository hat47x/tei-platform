package tenantisolation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newScope(t *testing.T, tenant string) (*Scope, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "seed.txt"), []byte("seed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	scope, err := Open(tenant, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = scope.Close() })
	return scope, root
}

func TestReadWriteStayInsideTenantRoot(t *testing.T) {
	scope, root := newScope(t, "tenant-a")
	if err := scope.Write("notes/a.txt", []byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	content, err := scope.Read("notes/a.txt")
	if err != nil || string(content) != "hello\n" {
		t.Fatalf("read=%q err=%v", content, err)
	}
	if _, err := os.Stat(filepath.Join(root, "notes", "a.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestRequireTenantRejectsAnotherTenant(t *testing.T) {
	scope, _ := newScope(t, "tenant-a")
	err := scope.RequireTenant("tenant-b")
	denial, ok := err.(*Denial)
	if !ok || denial.Code != CodeCrossTenant {
		t.Fatalf("want CROSS_TENANT_DENIED, got %v", err)
	}
	if err := scope.RequireTenant("tenant-a"); err != nil {
		t.Fatalf("same tenant was rejected: %v", err)
	}
	if len(scope.Denials()) != 1 {
		t.Fatalf("denials=%+v", scope.Denials())
	}
}

func TestRequireTenantRejectsEmptyRequest(t *testing.T) {
	scope, _ := newScope(t, "tenant-a")
	err := scope.RequireTenant("  ")
	denial, ok := err.(*Denial)
	if !ok || denial.Code != CodeTenantRequired {
		t.Fatalf("want TENANT_REQUIRED, got %v", err)
	}
}

func TestPathTraversalAndAbsolutePathsAreRejected(t *testing.T) {
	scope, _ := newScope(t, "tenant-a")
	for _, path := range []string{"../tenant-b/secret.txt", "/etc/passwd"} {
		if _, err := scope.Read(path); err == nil {
			t.Fatalf("path %q was accepted", path)
		}
	}
	for _, denial := range scope.Denials() {
		if denial.Code != CodeOutsideTenantRoot {
			t.Fatalf("unexpected denial: %+v", denial)
		}
	}
}

func TestSymlinkToAnotherTenantIsRejected(t *testing.T) {
	scope, root := newScope(t, "tenant-a")
	other, otherRoot := newScope(t, "tenant-b")
	if err := other.Write("secret.txt", []byte("tenant-b-secret\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(otherRoot, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := scope.Read("link/secret.txt"); err == nil {
		t.Fatal("read through a cross-tenant symlink was accepted")
	}
	if err := scope.Write("link/written.txt", []byte("x")); err == nil {
		t.Fatal("write through a cross-tenant symlink was accepted")
	}
}

func TestMissingFileIsNotABoundaryDenial(t *testing.T) {
	scope, _ := newScope(t, "tenant-a")
	if _, err := scope.Read("missing.txt"); !os.IsNotExist(err) {
		t.Fatalf("want not-exist, got %v", err)
	}
	if len(scope.Denials()) != 0 {
		t.Fatalf("missing file was recorded as a tenant denial: %+v", scope.Denials())
	}
}

func TestInvalidTenantIDIsRejected(t *testing.T) {
	for _, tenant := range []string{"", "Tenant-A", "../a", "a/b"} {
		if _, err := Open(tenant, t.TempDir()); err == nil || !strings.Contains(err.Error(), "invalid") {
			t.Fatalf("tenant %q was accepted: %v", tenant, err)
		}
	}
}
