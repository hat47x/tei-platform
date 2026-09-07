package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

const (
	applicationPatternTargetOverlaySchemaVersion = "business-procedure-pattern-target-overlay-v0"
	applicationPatternTargetOverlayArtifactRole  = "reference-pressure-overlay"
)

type applicationPatternTargetOverlayProbe struct {
	SchemaVersion string                                     `json:"schemaVersion"`
	ArtifactRole  string                                     `json:"artifactRole"`
	Canonical     bool                                       `json:"canonical"`
	Procedure     string                                     `json:"procedure"`
	Steps         []applicationPatternTargetOverlayStepProbe `json:"steps"`
}

type applicationPatternTargetOverlayStepProbe struct {
	Step                      string   `json:"step"`
	ApplicationPatternTargets []string `json:"applicationPatternTargets"`
}

func TestApplicationPatternTargetFileNativeOverlayPreservesProtectedTargetWithoutChangingRealization(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := readApplicationPatternTargetOverlayProbe(filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	candidates, err := applyApplicationPatternTargetOverlayProbe(f, overlay)
	if err != nil {
		t.Fatal(err)
	}
	candidate, ok := candidates["BP-06"]
	if !ok {
		t.Fatal("BP-06 protected-target overlayがありません")
	}
	if len(candidate.Body.ApplicationPatterns) != 0 {
		t.Fatalf("file-native protected targetをdirect Function realizationへ変換しました: %+v", candidate.Body.ApplicationPatterns)
	}
	if !reflect.DeepEqual(candidate.Body.Capabilities, []string{"security.authorize"}) {
		t.Fatalf("BP-06 supporting capabilityが失われました: %+v", candidate.Body.Capabilities)
	}
	if !reflect.DeepEqual(candidate.ApplicationPatternTargets, []string{"order_entry.submit_order"}) {
		t.Fatalf("protected targetがfile-native sourceから保持されません: %+v", candidate.ApplicationPatternTargets)
	}
	if err := checkExpectedDiagnostics(applicationPatternTargetDiagnostics(loaded.World, "BP-06", candidate.ApplicationPatternTargets), ""); err != nil {
		t.Fatalf("file-native protected targetをnormalized Worldでexact resolutionできません: %v", err)
	}

	result := evaluate(loaded.World, f)
	row, ok := targetPressureFunctionRowByStep(result, "BP-06")
	if !ok || !row.NoApplicationFunction {
		t.Fatalf("protected target overlayがBP-06 realization classificationへ逆流しました: %+v", row)
	}
}

func TestApplicationPatternTargetOverlayRoundTripsLosslessly(t *testing.T) {
	root := repoPath()
	overlay, err := readApplicationPatternTargetOverlayProbe(filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(overlay)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := decodeApplicationPatternTargetOverlayProbe(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reloaded, overlay) {
		t.Fatalf("protected-target overlay round-tripでmeaningを失いました: before=%+v after=%+v", overlay, reloaded)
	}
}

func TestApplicationPatternTargetOverlayFailsClosedOnStaleProcedureOrStep(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := readApplicationPatternTargetOverlayProbe(filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"))
	if err != nil {
		t.Fatal(err)
	}

	staleProcedure := overlay
	staleProcedure.Procedure = "order-entry.renamed-order-handling"
	if _, err := applyApplicationPatternTargetOverlayProbe(f, staleProcedure); err == nil {
		t.Fatal("stale Procedure identityをprotected-target overlayで黙って受理しました")
	}

	staleStep := overlay
	staleStep.Steps = append([]applicationPatternTargetOverlayStepProbe(nil), overlay.Steps...)
	staleStep.Steps[0].Step = "BP-06-renamed"
	if _, err := applyApplicationPatternTargetOverlayProbe(f, staleStep); err == nil {
		t.Fatal("stale Step addressをprotected-target overlayで黙って受理しました")
	}
}

func TestApplicationPatternTargetOverlayRejectsDuplicateStepAndUnknownFields(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := readApplicationPatternTargetOverlayProbe(filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"))
	if err != nil {
		t.Fatal(err)
	}
	duplicate := overlay
	duplicate.Steps = append(append([]applicationPatternTargetOverlayStepProbe(nil), overlay.Steps...), overlay.Steps[0])
	if _, err := applyApplicationPatternTargetOverlayProbe(f, duplicate); err == nil {
		t.Fatal("duplicate Step protected-target sourceを黙って受理しました")
	}

	_, err = decodeApplicationPatternTargetOverlayProbe(bytes.NewBufferString(`{
		"schemaVersion":"business-procedure-pattern-target-overlay-v0",
		"artifactRole":"reference-pressure-overlay",
		"canonical":false,
		"procedure":"order-entry.order-handling",
		"steps":[],
		"role":"protects"
	}`))
	if err == nil {
		t.Fatal("generic role fieldをprotected-target overlayへ黙って受理しました")
	}
}

func readApplicationPatternTargetOverlayProbe(path string) (applicationPatternTargetOverlayProbe, error) {
	file, err := os.Open(path)
	if err != nil {
		return applicationPatternTargetOverlayProbe{}, fmt.Errorf("protected-target overlay %sを読み込めません: %w", path, err)
	}
	defer file.Close()
	overlay, err := decodeApplicationPatternTargetOverlayProbe(file)
	if err != nil {
		return applicationPatternTargetOverlayProbe{}, fmt.Errorf("protected-target overlay %sが不正です: %w", path, err)
	}
	return overlay, nil
}

func decodeApplicationPatternTargetOverlayProbe(r io.Reader) (applicationPatternTargetOverlayProbe, error) {
	var overlay applicationPatternTargetOverlayProbe
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&overlay); err != nil {
		return applicationPatternTargetOverlayProbe{}, err
	}
	if err := requireEOF(decoder); err != nil {
		return applicationPatternTargetOverlayProbe{}, err
	}
	if overlay.SchemaVersion != applicationPatternTargetOverlaySchemaVersion {
		return applicationPatternTargetOverlayProbe{}, fmt.Errorf("unsupported protected-target overlay schemaVersion %q", overlay.SchemaVersion)
	}
	if overlay.ArtifactRole != applicationPatternTargetOverlayArtifactRole {
		return applicationPatternTargetOverlayProbe{}, fmt.Errorf("unsupported protected-target overlay artifactRole %q", overlay.ArtifactRole)
	}
	if overlay.Canonical {
		return applicationPatternTargetOverlayProbe{}, fmt.Errorf("protected-target pressure overlay must not claim Canonical authority")
	}
	if overlay.Procedure == "" {
		return applicationPatternTargetOverlayProbe{}, fmt.Errorf("protected-target overlay Procedure identity is required")
	}
	return overlay, nil
}

func applyApplicationPatternTargetOverlayProbe(f fixture, overlay applicationPatternTargetOverlayProbe) (map[string]applicationPatternTargetAwareStepProbe, error) {
	if overlay.Procedure != f.Procedure.ID {
		return nil, fmt.Errorf("protected-target overlay Procedure %s does not match %s", overlay.Procedure, f.Procedure.ID)
	}
	result := make(map[string]applicationPatternTargetAwareStepProbe, len(overlay.Steps))
	for _, source := range overlay.Steps {
		if source.Step == "" {
			return nil, fmt.Errorf("protected-target overlay Step address is required")
		}
		if _, exists := result[source.Step]; exists {
			return nil, fmt.Errorf("duplicate protected-target overlay Step %s", source.Step)
		}
		matches := 0
		for _, procedureStep := range f.Steps {
			if procedureStep.ID == source.Step {
				matches++
			}
		}
		if matches != 1 {
			return nil, fmt.Errorf("protected-target overlay Step %s resolves %d times", source.Step, matches)
		}
		if len(source.ApplicationPatternTargets) == 0 {
			return nil, fmt.Errorf("protected-target overlay Step %s has no target", source.Step)
		}
		candidate, err := deriveApplicationPatternTargetAwareStepProbe(f, source.Step, source.ApplicationPatternTargets)
		if err != nil {
			return nil, err
		}
		result[source.Step] = candidate
	}
	return result, nil
}
