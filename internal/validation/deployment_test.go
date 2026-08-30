package validation_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/deployment"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestReferenceDeploymentSnapshotValidates(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(root, "reference", "observed", "production.deployment-state.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var state deployment.Snapshot
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		t.Fatal(err)
	}
	ds := validation.ValidateDeployment(loaded.World, state)
	if validation.HasErrors(ds) {
		t.Fatalf("deployment validation errors: %#v", ds)
	}
}

func TestDeploymentSnapshotRejectsUnknownActiveRelease(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	state := deployment.Snapshot{Environment: "production", ActiveApplications: []deployment.ApplicationReleaseRef{{Application: "order_entry", Version: "9.9.9"}}}
	ds := validation.ValidateDeployment(loaded.World, state)
	if !validation.HasErrors(ds) {
		t.Fatalf("expected deployment error, got %#v", ds)
	}
}

func TestDeploymentSnapshotAllowsExplicitCanaryTrafficAcrossTwoActiveReleases(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(root, "reference", "observed", "production.canary.deployment-state.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var state deployment.Snapshot
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		t.Fatal(err)
	}
	ds := validation.ValidateDeployment(loaded.World, state)
	if validation.HasErrors(ds) {
		t.Fatalf("canary deployment validation errors: %#v", ds)
	}
}

func TestDeploymentSnapshotRejectsTrafficOverallocation(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	state := deployment.Snapshot{
		Environment:        "production",
		ActiveApplications: []deployment.ApplicationReleaseRef{{Application: "order_entry", Version: "3.7.0"}, {Application: "order_entry", Version: "3.8.0"}},
		ApplicationRuntime: []deployment.ApplicationRuntimeState{
			{Application: "order_entry", Version: "3.7.0", Lifecycle: deployment.LifecycleServing, TrafficBasisPoints: 7000, AcceptingWork: true},
			{Application: "order_entry", Version: "3.8.0", Lifecycle: deployment.LifecycleServing, TrafficBasisPoints: 4000, AcceptingWork: true},
		},
	}
	ds := validation.ValidateDeployment(loaded.World, state)
	if !validation.HasErrors(ds) {
		t.Fatalf("expected traffic overallocation error, got %#v", ds)
	}
}
