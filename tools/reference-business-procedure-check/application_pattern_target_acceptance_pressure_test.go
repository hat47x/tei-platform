package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

type applicationPatternTargetAcceptanceSourceSetProbe struct {
	ProcedureArtifact sourceindex.Artifact
	OverlayArtifact   sourceindex.Artifact
	Fingerprint       string
}

func TestProcedureOnlyFingerprintCannotProtectFileNativeO1Target(t *testing.T) {
	root := repoPath()
	probeRoot := prepareO1AcceptanceWorkspaceProbe(t, root)
	procedureRel := "procedures/order-handling.json"
	overlayRel := "procedure-overlays/order-handling-protected-target.json"

	before, err := readO1AcceptanceSourceSetProbe(probeRoot, procedureRel, overlayRel)
	if err != nil {
		t.Fatal(err)
	}
	changeO1OverlayTargetProbe(t, probeRoot, overlayRel, "order_entry.order_record")
	after, err := readO1AcceptanceSourceSetProbe(probeRoot, procedureRel, overlayRel)
	if err != nil {
		t.Fatal(err)
	}

	if before.ProcedureArtifact.Fingerprint != after.ProcedureArtifact.Fingerprint {
		t.Fatal("test setup: O1 target-only changeでProcedure sourceまで変化しました")
	}
	if before.OverlayArtifact.Fingerprint == after.OverlayArtifact.Fingerprint {
		t.Fatal("O1 target-only changeをoverlay fingerprint差として観測できません")
	}
	if before.Fingerprint == after.Fingerprint {
		t.Fatal("O1 target-only changeをcomposite Acceptance fingerprint差として観測できません")
	}
}

func TestO1CompositeAcceptanceFingerprintRejectsStaleTargetChange(t *testing.T) {
	root := repoPath()
	probeRoot := prepareO1AcceptanceWorkspaceProbe(t, root)
	procedureRel := "procedures/order-handling.json"
	overlayRel := "procedure-overlays/order-handling-protected-target.json"

	before, err := readO1AcceptanceSourceSetProbe(probeRoot, procedureRel, overlayRel)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acceptO1SourceSetProbe(probeRoot, procedureRel, overlayRel, before.Fingerprint); err != nil {
		t.Fatalf("baseline O1 source setをacceptance targetとして解決できません: %v", err)
	}

	changeO1OverlayTargetProbe(t, probeRoot, overlayRel, "order_entry.order_record")
	if _, err := acceptO1SourceSetProbe(probeRoot, procedureRel, overlayRel, before.Fingerprint); err == nil {
		t.Fatal("O1 target-only change後も旧composite fingerprintを受理しました")
	}

	current, err := readO1AcceptanceSourceSetProbe(probeRoot, procedureRel, overlayRel)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := acceptO1SourceSetProbe(probeRoot, procedureRel, overlayRel, current.Fingerprint)
	if err != nil {
		t.Fatalf("fresh composite fingerprintでO1 source setを受理できません: %v", err)
	}
	candidate := accepted["BP-06"]
	if !reflect.DeepEqual(candidate.ApplicationPatternTargets, []string{"order_entry.order_record"}) {
		t.Fatalf("fresh acceptance targetがtarget-only changeを保持しません: %+v", candidate.ApplicationPatternTargets)
	}
	if len(candidate.Body.ApplicationPatterns) != 0 {
		t.Fatalf("fresh acceptance targetがprotected targetをrealizationへ変換しました: %+v", candidate.Body.ApplicationPatterns)
	}
}

func TestO1CompositeAcceptanceFingerprintIsPureMoveInvariant(t *testing.T) {
	root := repoPath()
	probeRoot := prepareO1AcceptanceWorkspaceProbe(t, root)
	procedureRel := "procedures/order-handling.json"
	overlayRel := "procedure-overlays/order-handling-protected-target.json"

	before, err := readO1AcceptanceSourceSetProbe(probeRoot, procedureRel, overlayRel)
	if err != nil {
		t.Fatal(err)
	}

	movedRel := "procedure-overlays/sales/order-handling-protected-target.json"
	movedPath := filepath.Join(probeRoot, filepath.FromSlash(movedRel))
	if err := os.MkdirAll(filepath.Dir(movedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(probeRoot, filepath.FromSlash(overlayRel)), movedPath); err != nil {
		t.Fatal(err)
	}

	after, err := readO1AcceptanceSourceSetProbe(probeRoot, procedureRel, movedRel)
	if err != nil {
		t.Fatal(err)
	}
	if before.OverlayArtifact.Path == after.OverlayArtifact.Path {
		t.Fatal("test setup: overlay sourceをmoveできていません")
	}
	if before.OverlayArtifact.Fingerprint != after.OverlayArtifact.Fingerprint {
		t.Fatal("pure overlay file moveでcontent fingerprintが変化しました")
	}
	if before.Fingerprint != after.Fingerprint {
		t.Fatal("pure overlay file moveでcomposite Acceptance fingerprintが変化しました")
	}
	if _, err := acceptO1SourceSetProbe(probeRoot, procedureRel, movedRel, before.Fingerprint); err != nil {
		t.Fatalf("pure file move後に既存Acceptance targetを解決できません: %v", err)
	}
}

func TestO1CompositeAcceptanceRejectsFreshButInvalidTarget(t *testing.T) {
	root := repoPath()
	probeRoot := prepareO1AcceptanceWorkspaceProbe(t, root)
	procedureRel := "procedures/order-handling.json"
	overlayRel := "procedure-overlays/order-handling-protected-target.json"

	changeO1OverlayTargetProbe(t, probeRoot, overlayRel, "order_entry.missing_target")
	current, err := readO1AcceptanceSourceSetProbe(probeRoot, procedureRel, overlayRel)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acceptO1SourceSetProbe(probeRoot, procedureRel, overlayRel, current.Fingerprint); err == nil {
		t.Fatal("fresh fingerprintだけを理由にmissing protected targetをacceptance可能にしました")
	}
}

func TestO1CompositeAcceptanceTargetHasSemanticSourceRolesNotPaths(t *testing.T) {
	fingerprintA := o1CompositeAcceptanceFingerprintProbe("sha256:procedure", "sha256:overlay")
	fingerprintB := o1CompositeAcceptanceFingerprintProbe("sha256:procedure", "sha256:overlay")
	if fingerprintA != fingerprintB {
		t.Fatal("同じsemantic source roles/fingerprintsからdeterministic compositeを生成できません")
	}
	if fingerprintA == o1CompositeAcceptanceFingerprintProbe("sha256:overlay", "sha256:procedure") {
		t.Fatal("Procedure/overlay source role順序を失ったcomposite fingerprintです")
	}
}

func prepareO1AcceptanceWorkspaceProbe(t *testing.T, repoRoot string) string {
	t.Helper()
	root := prepareProcedureAuthoringWorkspace(t, repoRoot)
	overlayDir := filepath.Join(root, "procedure-overlays")
	if err := os.MkdirAll(overlayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(repoRoot, "reference", "business-system-design", "order-entry-protected-target-overlay.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlayDir, "order-handling-protected-target.json"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func readO1AcceptanceSourceSetProbe(root, procedureRel, overlayRel string) (applicationPatternTargetAcceptanceSourceSetProbe, error) {
	procedureContent, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(procedureRel)))
	if err != nil {
		return applicationPatternTargetAcceptanceSourceSetProbe{}, err
	}
	overlayContent, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(overlayRel)))
	if err != nil {
		return applicationPatternTargetAcceptanceSourceSetProbe{}, err
	}
	procedureArtifact, err := sourceindex.NewArtifact(procedureRel, procedureContent)
	if err != nil {
		return applicationPatternTargetAcceptanceSourceSetProbe{}, err
	}
	overlayArtifact, err := sourceindex.NewArtifact(overlayRel, overlayContent)
	if err != nil {
		return applicationPatternTargetAcceptanceSourceSetProbe{}, err
	}
	return applicationPatternTargetAcceptanceSourceSetProbe{
		ProcedureArtifact: procedureArtifact,
		OverlayArtifact:   overlayArtifact,
		Fingerprint:       o1CompositeAcceptanceFingerprintProbe(procedureArtifact.Fingerprint, overlayArtifact.Fingerprint),
	}, nil
}

func o1CompositeAcceptanceFingerprintProbe(procedureFingerprint, overlayFingerprint string) string {
	content := []byte("business-procedure-source=" + procedureFingerprint + "\nprotected-target-overlay=" + overlayFingerprint + "\n")
	return sourceindex.Fingerprint(content)
}

func acceptO1SourceSetProbe(root, procedureRel, overlayRel, expectedFingerprint string) (map[string]applicationPatternTargetAwareStepProbe, error) {
	sources, err := readO1AcceptanceSourceSetProbe(root, procedureRel, overlayRel)
	if err != nil {
		return nil, err
	}
	if sources.Fingerprint != expectedFingerprint {
		return nil, fmt.Errorf("stale O1 acceptance target: have %s want %s", sources.Fingerprint, expectedFingerprint)
	}
	procedureContent, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(procedureRel)))
	if err != nil {
		return nil, err
	}
	f, err := decodeFixtureBytes(procedureContent)
	if err != nil {
		return nil, err
	}
	overlay, err := readApplicationPatternTargetOverlayProbe(filepath.Join(root, filepath.FromSlash(overlayRel)))
	if err != nil {
		return nil, err
	}
	candidates, err := applyApplicationPatternTargetOverlayProbe(f, overlay)
	if err != nil {
		return nil, err
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return nil, err
	}
	for stepID, candidate := range candidates {
		if err := checkExpectedDiagnostics(applicationPatternTargetDiagnostics(loaded.World, stepID, candidate.ApplicationPatternTargets), ""); err != nil {
			return nil, fmt.Errorf("protected target source is not acceptable for %s: %w", stepID, err)
		}
	}
	return candidates, nil
}

func changeO1OverlayTargetProbe(t *testing.T, root, overlayRel, target string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(overlayRel))
	overlay, err := readApplicationPatternTargetOverlayProbe(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(overlay.Steps) != 1 {
		t.Fatalf("test setup: expected one O1 overlay step, got %d", len(overlay.Steps))
	}
	overlay.Steps[0].ApplicationPatternTargets = []string{target}
	content, err := json.MarshalIndent(overlay, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}
