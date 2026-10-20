package deliverybundle

import (
	"path/filepath"
	"testing"
	"time"
)

func activeState(t *testing.T) string {
	t.Helper()
	statePath := filepath.Join(t.TempDir(), "state.json")
	if _, err := Activate(catalogWith(t, "audit"), statePath); err != nil {
		t.Fatal(err)
	}
	return statePath
}

func TestHealthUsesActiveDeployment(t *testing.T) {
	statePath := activeState(t)
	receipt, err := Health(statePath, `test -n "$TEI_CATALOG" && test -n "$TEI_ACTIVE_DIGEST" && test -n "$TEI_STATE"`, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Healthy || receipt.ExitCode != 0 || receipt.OutputDigest == "" {
		t.Fatalf("unexpected health receipt: %+v", receipt)
	}
}

func TestHealthFailureIsReported(t *testing.T) {
	statePath := activeState(t)
	receipt, err := Health(statePath, "exit 3", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Healthy || receipt.ExitCode != 3 {
		t.Fatalf("unexpected health receipt: %+v", receipt)
	}
}

func TestHealthTimeoutIsReported(t *testing.T) {
	statePath := activeState(t)
	receipt, err := Health(statePath, "sleep 2", 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Healthy || receipt.Error != "timeout" {
		t.Fatalf("unexpected health receipt: %+v", receipt)
	}
}

func TestHealthRequiresValidState(t *testing.T) {
	if _, err := Health(filepath.Join(t.TempDir(), "missing.json"), "exit 0", time.Second); err == nil {
		t.Fatal("missing state was accepted")
	}
}
