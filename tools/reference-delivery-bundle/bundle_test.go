package deliverybundle

import (
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestPackAndVerifySaaSBundle(t *testing.T) {
	root := repoRoot(t)
	profile := filepath.Join(root, "tools", "reference-delivery-profile", "testdata", "profiles", "saas.json")
	catalog := filepath.Join(root, "tools", "reference-delivery-profile", "testdata", "plugins")
	out := filepath.Join(t.TempDir(), "bundle.zip")
	packed, err := Pack(profile, catalog, out)
	if err != nil {
		t.Fatal(err)
	}
	if !packed.Passed || packed.Manifest.ProfileID != "saas-default" || len(packed.Manifest.Plugins) == 0 {
		t.Fatalf("unexpected pack receipt: %+v", packed)
	}
	verified, err := Verify(out)
	if err != nil {
		t.Fatal(err)
	}
	if !verified.Passed || verified.Manifest.CatalogDigest != packed.Manifest.CatalogDigest {
		t.Fatalf("unexpected verify receipt: %+v", verified)
	}
	if verified.PackageDigest != packed.PackageDigest {
		t.Fatalf("package digest changed: %s vs %s", verified.PackageDigest, packed.PackageDigest)
	}
}
