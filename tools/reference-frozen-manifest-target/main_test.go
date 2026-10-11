package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimesnapshot "tei/tools/reference-runtime-snapshot"
)

func pinnedFixture(t *testing.T) (string, runtimesnapshot.Identity) {
	t.Helper()
	raw := []byte(`{"applicationId":"order_entry","handoverBaselineId":"B","baselineRepositoryRevision":"R","targetBuildRevision":"T","unresolved":["not-yet-authorized"]}`)
	file := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(file, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return file, runtimesnapshot.Identity{
		ApplicationID: "order_entry", HandoverBaselineID: "B",
		BaselineRepositoryRevision: "R", TargetBuildRevision: "T",
		ManifestSHA256: hex.EncodeToString(sum[:]),
	}
}

func TestFrozenManifestEntryRequiresExplicitBinding(t *testing.T) {
	file, identity := pinnedFixture(t)
	cfg, err := parseOptions([]string{
		"-manifest", file, "-application", identity.ApplicationID,
		"-baseline-id", identity.HandoverBaselineID,
		"-baseline-revision", identity.BaselineRepositoryRevision,
		"-target-revision", identity.TargetBuildRevision,
		"-sha256", identity.ManifestSHA256, "-port", "8088",
	})
	if err != nil || cfg.manifest != file || cfg.port != 8088 ||
		cfg.identity.ManifestSHA256 != identity.ManifestSHA256 {
		t.Fatalf("valid pinned entry rejected: %+v / %v", cfg, err)
	}
	for _, args := range [][]string{
		nil, {"-manifest", file, "trailing"},
		{"-manifest", file, "-port", "0"},
		{"-manifest", file, "-port", "65536"},
		{"-manifest", file, "-port", "not-a-number"},
		{"-unknown", "value"},
	} {
		if _, err := parseOptions(args); err == nil {
			t.Fatalf("invalid command line accepted: %v", args)
		}
	}
}

func TestFrozenManifestEntryHasOnlyReadOnlyRoutes(t *testing.T) {
	file, identity := pinnedFixture(t)
	snapshot, err := loadSnapshot(file, identity)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/manifest.json", http.StatusOK},
		{http.MethodGet, "/healthz", http.StatusNoContent},
		{http.MethodGet, "/scenario/read", http.StatusNotFound},
		{http.MethodPost, "/scenario/submit", http.StatusNotFound},
		{http.MethodPost, "/manifest.json", http.StatusMethodNotAllowed},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(tc.method, tc.path, nil)
		snapshot.Handler().ServeHTTP(response, request)
		if response.Code != tc.status {
			t.Fatalf("%s %s: got %d, want %d", tc.method, tc.path, response.Code, tc.status)
		}
	}
}

func TestFrozenManifestEntryFailsClosedOnMismatchesAndOversize(t *testing.T) {
	file, identity := pinnedFixture(t)
	changed := identity
	changed.ApplicationID = "repair_intake"
	if _, err := loadSnapshot(file, changed); err == nil {
		t.Fatal("wrong application identity accepted")
	}
	changed = identity
	changed.ManifestSHA256 = strings.Repeat("0", 64)
	if _, err := loadSnapshot(file, changed); err == nil {
		t.Fatal("unpinned manifest bytes accepted")
	}
	changed = identity
	changed.TargetBuildRevision = ""
	if _, err := loadSnapshot(file, changed); err == nil {
		t.Fatal("missing independent target build revision accepted")
	}
	if err := os.WriteFile(file, []byte(strings.Repeat("x", maxManifestBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSnapshot(file, identity); err == nil ||
		!strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized manifest was accepted: %v", err)
	}
}
