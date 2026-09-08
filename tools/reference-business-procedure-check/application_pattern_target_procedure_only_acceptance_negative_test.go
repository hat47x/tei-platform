package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

func TestProcedureOnlyAcceptanceIsBlindToSeparateO1OverlayChange(t *testing.T) {
	root := repoPath()
	probeRoot := prepareO1AcceptanceWorkspaceProbe(t, root)
	procedureRel := "procedures/order-handling.json"
	overlayRel := "procedure-overlays/order-handling-protected-target.json"

	beforeResult, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	beforeProcedure, err := acceptProcedureSource(probeRoot, procedureRel, &beforeResult, "")
	if err != nil {
		t.Fatal(err)
	}
	beforeOverlay, err := readApplicationPatternTargetOverlayProbe(filepath.Join(probeRoot, filepath.FromSlash(overlayRel)))
	if err != nil {
		t.Fatal(err)
	}

	changeO1OverlayTargetProbe(t, probeRoot, overlayRel, "order_entry.order_record")

	afterResult, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	// This deliberately uses the current Procedure-only acceptance path. It is
	// expected to succeed because the Procedure source bytes did not change.
	// The test records why a separate O1 source cannot rely on that fingerprint.
	afterProcedure, err := acceptProcedureSource(probeRoot, procedureRel, &afterResult, beforeProcedure.Artifact.Fingerprint)
	if err != nil {
		t.Fatalf("test premise changed: Procedure-only acceptance no longer remains blind to overlay-only changes: %v", err)
	}
	afterOverlay, err := readApplicationPatternTargetOverlayProbe(filepath.Join(probeRoot, filepath.FromSlash(overlayRel)))
	if err != nil {
		t.Fatal(err)
	}

	if beforeProcedure.Artifact.Fingerprint != afterProcedure.Artifact.Fingerprint {
		t.Fatal("test setup: Procedure bytes changed during O1 overlay-only edit")
	}
	if reflect.DeepEqual(beforeOverlay, afterOverlay) {
		t.Fatal("test setup: O1 overlay target did not change")
	}
	if reflect.DeepEqual(beforeOverlay.Steps[0].ApplicationPatternTargets, afterOverlay.Steps[0].ApplicationPatternTargets) {
		t.Fatal("test setup: protected target meaning did not change")
	}
}
