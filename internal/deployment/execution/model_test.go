package execution_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/deployment"
	execution "tei/internal/deployment/execution"
	"tei/internal/releaseplan"
	"tei/internal/workspace"
)

func TestReferenceCanaryDeploymentDrainsOldReleaseBeforeContract(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	var state deployment.Snapshot
	f, err := os.Open(filepath.Join(root, "reference", "observed", "production.deployment-state.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		t.Fatal(err)
	}

	var def releaseplan.Definition
	for _, x := range loaded.World.SolutionReleases {
		if x.ID == "order_rollout" && x.Version == "2026.8.27" {
			def = x
			break
		}
	}
	if def.ID == "" {
		t.Fatal("solution release not loaded")
	}
	compiled := releaseplan.CompileWithDeployment(def, loaded.World.ApplicationReleases, loaded.World.DatabaseReleases, state)
	if !compiled.Executable {
		t.Fatalf("compiled plan blocked: %v", compiled.Findings)
	}

	run := execution.Simulate(def, compiled, state, loaded.World.DatabaseReleases)
	if run.Status != execution.RunSucceeded {
		t.Fatalf("run blocked: %v", run.Findings)
	}
	if run.Final.IsActive("order_entry", "3.7.0") {
		t.Fatal("old application release should be retired")
	}
	if !run.Final.IsActive("order_entry", "3.8.0") {
		t.Fatal("new application release should be active")
	}

	var traffic []int
	var sawDrain, sawRetire bool
	for _, e := range run.Events {
		if e.Kind == execution.EventTrafficShifted && e.Application == "order_entry" && e.Version == "3.8.0" {
			traffic = append(traffic, e.TrafficBasisPoints)
		}
		if e.Kind == execution.EventDrainCompleted && e.Version == "3.7.0" {
			sawDrain = true
		}
		if e.Kind == execution.EventApplicationRetired && e.Version == "3.7.0" {
			sawRetire = true
		}
	}
	if len(traffic) != 3 || traffic[0] != 1000 || traffic[1] != 5000 || traffic[2] != 10000 {
		t.Fatalf("unexpected canary traffic sequence: %v", traffic)
	}
	if !sawDrain || !sawRetire {
		t.Fatalf("expected drain and retire events: %+v", run.Events)
	}
}

func TestSimulationBlocksRetireWhileTrafficRemains(t *testing.T) {
	def := releaseplan.Definition{ID: "s", Version: "1.0.0", Steps: []releaseplan.Step{
		{ID: "retire", Kind: releaseplan.StepApplicationRetire, Phase: releaseplan.PhaseTransition, Application: "a", Version: "1.0.0"},
	}}
	compiled := releaseplan.Compiled{ID: def.ID, Version: def.Version, Steps: def.Steps, Executable: true}
	initial := deployment.Snapshot{Environment: "prod", ActiveApplications: []deployment.ApplicationReleaseRef{{Application: "a", Version: "1.0.0"}}}
	run := execution.Simulate(def, compiled, initial, nil)
	if run.Status != execution.RunBlocked {
		t.Fatalf("expected blocked run, got %+v", run)
	}
}
