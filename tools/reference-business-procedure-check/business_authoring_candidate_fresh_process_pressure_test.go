package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

type businessFreshProcessReloadDescriptorProbe struct {
	ProcedurePaths []string `json:"procedurePaths"`
	HandoffPath    string   `json:"handoffPath"`
	WorldWorkspace string   `json:"worldWorkspace"`
	ResultPath     string   `json:"resultPath"`
}

func TestBusinessCandidateOnlyFreshProcessReloadPreservesMeaningWithoutLegacyInput(t *testing.T) {
	repoRoot := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	want, err := deriveBusinessAuthoringBundleProbe(businessAuthoringBundleProbe(candidate))
	if err != nil {
		t.Fatal(err)
	}

	diskRoot := t.TempDir()
	descriptor := businessFreshProcessReloadDescriptorProbe{
		WorldWorkspace: filepath.Join(repoRoot, "reference", "workspace"),
		ResultPath:     filepath.Join(diskRoot, "fresh-process-result.json"),
	}
	for _, procedure := range candidate.Procedures {
		path := filepath.Join(diskRoot, "candidate", procedure.Procedure.ID+".json")
		writeBusinessCandidateDiskSourceProbe(t, path, procedure)
		descriptor.ProcedurePaths = append(descriptor.ProcedurePaths, path)
	}
	descriptor.HandoffPath = filepath.Join(diskRoot, "candidate", "handoffs.json")
	writeBusinessCandidateDiskSourceProbe(t, descriptor.HandoffPath, candidate.Handoffs)

	// A legacy-looking artifact remains on disk, but its path is deliberately not
	// passed to the helper process. The child receives only candidate source paths
	// plus the current World workspace used for cross-view Conformance.
	legacyDecoy := filepath.Join(diskRoot, "legacy", "order-entry-protected-target-overlay.json")
	if err := os.MkdirAll(filepath.Dir(legacyDecoy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyDecoy, []byte(`{"legacy":"must-not-be-read"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	descriptorPath := filepath.Join(diskRoot, "fresh-process-descriptor.json")
	writeBusinessCandidateDiskSourceProbe(t, descriptorPath, descriptor)

	cmd := exec.Command(os.Args[0], "-test.run=^TestBusinessCandidateOnlyFreshProcessReloadHelper$")
	cmd.Env = append(os.Environ(), "TEI_BUSINESS_FRESH_PROCESS_DESCRIPTOR="+descriptorPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("candidate-only fresh-process helper failed: %v\n%s", err, output)
	}

	raw, err := os.ReadFile(descriptor.ResultPath)
	if err != nil {
		t.Fatalf("fresh-process helper did not write normalized Business result: %v", err)
	}
	var got d4O1NormalizedBusinessModelProbe
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("fresh-process normalized Business result is invalid: %v", err)
	}
	if semanticFingerprintD4O1Probe(t, got) != semanticFingerprintD4O1Probe(t, want) {
		t.Fatalf("fresh-process candidate-only reload changed normalized Business semantic fingerprint: got=%+v want=%+v", got, want)
	}
}

// TestBusinessCandidateOnlyFreshProcessReloadHelper is executed only as a child
// test process. It intentionally has no migration path and no legacy source
// locator: all Business authority inputs are the candidate source paths supplied
// by the parent descriptor.
func TestBusinessCandidateOnlyFreshProcessReloadHelper(t *testing.T) {
	descriptorPath := os.Getenv("TEI_BUSINESS_FRESH_PROCESS_DESCRIPTOR")
	if descriptorPath == "" {
		t.Skip("fresh-process helper only")
	}

	raw, err := os.ReadFile(descriptorPath)
	if err != nil {
		t.Fatal(err)
	}
	var descriptor businessFreshProcessReloadDescriptorProbe
	if err := json.Unmarshal(raw, &descriptor); err != nil {
		t.Fatal(err)
	}
	if len(descriptor.ProcedurePaths) == 0 || descriptor.HandoffPath == "" || descriptor.WorldWorkspace == "" || descriptor.ResultPath == "" {
		t.Fatal("fresh-process descriptor is incomplete")
	}

	reloaded := businessAuthoringBundleProbe{}
	for _, path := range descriptor.ProcedurePaths {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		procedure, decodeErr := decodeInlineO1FixtureProbe(file)
		closeErr := file.Close()
		if decodeErr != nil {
			t.Fatalf("candidate Procedure fresh-process reload %s: %v", path, decodeErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		reloaded.Procedures = append(reloaded.Procedures, procedure)
	}

	handoffFile, err := os.Open(descriptor.HandoffPath)
	if err != nil {
		t.Fatal(err)
	}
	reloaded.Handoffs, err = decodeStandaloneHandoffSourceProbe(handoffFile)
	closeErr := handoffFile.Close()
	if err != nil {
		t.Fatalf("candidate Handoff fresh-process reload: %v", err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}

	model, err := deriveBusinessAuthoringBundleProbe(reloaded)
	if err != nil {
		t.Fatalf("candidate-only fresh-process sources could not normalize: %v", err)
	}
	loaded, err := workspace.Load(descriptor.WorldWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateD4O1NormalizedBusinessModelProbe(loaded.World, model); err != nil {
		t.Fatalf("candidate-only fresh-process reload does not conform to current World: %v", err)
	}

	encoded, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(descriptor.ResultPath, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}
