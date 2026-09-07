package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

type d4O1AddressedBusinessStepProbe struct {
	LocalKey                    string                       `json:"localKey"`
	Body                        d4BusinessProcedureStepProbe `json:"body"`
	ProtectedApplicationPattern string                       `json:"protectedApplicationPattern,omitempty"`
}

type d4O1AddressedBusinessProcedureProbe struct {
	ID      string                                 `json:"id"`
	Purpose string                                 `json:"purpose"`
	Trigger candidateRef                           `json:"trigger"`
	Actor   candidateRef                           `json:"actor"`
	Steps   []d4O1AddressedBusinessStepProbe       `json:"steps"`
}

type d4O1NormalizedBusinessModelProbe struct {
	Procedures []d4O1AddressedBusinessProcedureProbe `json:"procedures"`
	Handoffs   []canonicalBusinessHandoffProbe       `json:"handoffs,omitempty"`
}

func TestD4O1NormalizedBusinessModelComposesCurrentMinimumCandidate(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, root)
	loaded, err := workspace.Load(filepath.Join(repoPath(), "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	model, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	if err := validateD4O1NormalizedBusinessModelProbe(loaded.World, model); err != nil {
		t.Fatalf("D4 + O1 + S4-A + H1/R3を一つのnormalized Business meaningとしてvalidateできません: %v", err)
	}
	if len(model.Procedures) != 2 || len(model.Handoffs) != 1 {
		t.Fatalf("unexpected normalized Business model: procedures=%d handoffs=%d", len(model.Procedures), len(model.Handoffs))
	}
	bp06, ok := findD4O1AddressedStepProbe(model, "order-entry.order-handling", "BP-06")
	if !ok {
		t.Fatal("BP-06がnormalized minimum modelにありません")
	}
	if bp06.ProtectedApplicationPattern != "order_entry.submit_order" {
		t.Fatalf("BP-06 protected target=%q", bp06.ProtectedApplicationPattern)
	}
	if len(bp06.Body.ApplicationPatterns) != 0 || !reflect.DeepEqual(bp06.Body.Capabilities, []string{"security.authorize"}) {
		t.Fatalf("BP-06 target / realization / capability separationを失いました: %+v", bp06)
	}
}

func TestD4O1SemanticAcceptanceFingerprintChangesOnProtectedTargetOnlyMeaning(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, root)

	before, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	changedTargets := currentProtectedTargetsD4O1Probe()
	changedTargets["order-entry.order-handling/BP-06"] = []string{"order_entry.order_record"}
	after, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, changedTargets)
	if err != nil {
		t.Fatal(err)
	}

	beforeStep, _ := findD4O1AddressedStepProbe(before, "order-entry.order-handling", "BP-06")
	afterStep, _ := findD4O1AddressedStepProbe(after, "order-entry.order-handling", "BP-06")
	if !reflect.DeepEqual(beforeStep.Body, afterStep.Body) {
		t.Fatal("protected target-only changeでD4 bodyまで変化しました")
	}
	if semanticFingerprintD4O1Probe(t, before) == semanticFingerprintD4O1Probe(t, after) {
		t.Fatal("protected target-only accepted meaning changeをsemantic Acceptance fingerprintが見逃しました")
	}
}

func TestD4O1SemanticAcceptanceFingerprintChangesOnHandoffOnlyMeaning(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, root)

	before, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	changed := set
	changed.Handoffs = append([]referenceSetHandoff(nil), set.Handoffs...)
	changed.Handoffs[0].Condition = "follow-up is required because unresolved external work remains"
	after, err := deriveD4O1NormalizedBusinessModelProbe(changed, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}

	if semanticFingerprintD4O1Probe(t, before) == semanticFingerprintD4O1Probe(t, after) {
		t.Fatal("handoff-only accepted meaning changeをsemantic Acceptance fingerprintが見逃しました")
	}
}

func TestD4O1SemanticAcceptanceFingerprintIgnoresPureProcedureArtifactLocatorMove(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, root)

	before, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	moved := set
	moved.Procedures = append([]referenceSetProcedure(nil), set.Procedures...)
	for i := range moved.Procedures {
		moved.Procedures[i].Artifact = "moved/" + moved.Procedures[i].Artifact
	}
	after, err := deriveD4O1NormalizedBusinessModelProbe(moved, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatal("pure Procedure artifact locator moveでnormalized Business meaningが変化しました")
	}
	if semanticFingerprintD4O1Probe(t, before) != semanticFingerprintD4O1Probe(t, after) {
		t.Fatal("pure locator moveでsemantic Acceptance fingerprintが変化しました")
	}
}

func TestD4O1SemanticAcceptanceFingerprintNormalizesSourceSetOrdering(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, root)

	before, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	reordered := set
	reordered.Procedures = append([]referenceSetProcedure(nil), set.Procedures...)
	for left, right := 0, len(reordered.Procedures)-1; left < right; left, right = left+1, right-1 {
		reordered.Procedures[left], reordered.Procedures[right] = reordered.Procedures[right], reordered.Procedures[left]
	}
	after, err := deriveD4O1NormalizedBusinessModelProbe(reordered, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	if semanticFingerprintD4O1Probe(t, before) != semanticFingerprintD4O1Probe(t, after) {
		t.Fatal("source setのProcedure locator記述順だけでsemantic Acceptance fingerprintが変化しました")
	}
}

func TestD4O1NormalizedBusinessModelDoesNotCarrySourceOrRuntimeInfrastructure(t *testing.T) {
	modelType := reflect.TypeOf(d4O1NormalizedBusinessModelProbe{})
	if got := fieldNames(modelType); !reflect.DeepEqual(got, []string{"Procedures", "Handoffs"}) {
		t.Fatalf("minimum normalized Business modelが膨張しました: got=%v", got)
	}
	for _, forbidden := range []string{"Source", "Path", "Artifact", "Fingerprint", "Revision", "Runtime", "Workflow", "Execution"} {
		if _, found := modelType.FieldByName(forbidden); found {
			t.Fatalf("normalized Business meaningへSource/runtime infrastructure %qを混入しました", forbidden)
		}
	}
}

func deriveD4O1NormalizedBusinessModelProbe(set referenceSet, fixtures map[string]fixture, targets map[string][]string) (d4O1NormalizedBusinessModelProbe, error) {
	ids := make([]string, 0, len(fixtures))
	for id := range fixtures {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	result := d4O1NormalizedBusinessModelProbe{
		Procedures: make([]d4O1AddressedBusinessProcedureProbe, 0, len(ids)),
	}
	for _, id := range ids {
		f := fixtures[id]
		perStep := map[string][]string{}
		prefix := id + "/"
		for key, value := range targets {
			if strings.HasPrefix(key, prefix) {
				perStep[strings.TrimPrefix(key, prefix)] = append([]string(nil), value...)
			}
		}
		body, err := deriveD4O1ProcedureProbe(f, perStep)
		if err != nil {
			return d4O1NormalizedBusinessModelProbe{}, fmt.Errorf("Procedure %s: %w", id, err)
		}
		procedure := d4O1AddressedBusinessProcedureProbe{
			ID:      body.ID,
			Purpose: body.Purpose,
			Trigger: body.Trigger,
			Actor:   body.Actor,
			Steps:   make([]d4O1AddressedBusinessStepProbe, 0, len(body.Steps)),
		}
		for i, step := range body.Steps {
			procedure.Steps = append(procedure.Steps, d4O1AddressedBusinessStepProbe{
				LocalKey:                    f.Steps[i].ID,
				Body:                        step.Body,
				ProtectedApplicationPattern: step.ProtectedApplicationPattern,
			})
		}
		result.Procedures = append(result.Procedures, procedure)
	}

	for key := range targets {
		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			return d4O1NormalizedBusinessModelProbe{}, fmt.Errorf("invalid protected target address %q", key)
		}
		if _, ok := fixtures[parts[0]]; !ok {
			return d4O1NormalizedBusinessModelProbe{}, fmt.Errorf("protected target Procedure %q not found", parts[0])
		}
	}

	handoffs, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		return d4O1NormalizedBusinessModelProbe{}, err
	}
	result.Handoffs = handoffs
	return result, nil
}

func validateD4O1NormalizedBusinessModelProbe(world ir.World, model d4O1NormalizedBusinessModelProbe) error {
	addresses := map[string]map[string]struct{}{}
	for _, procedure := range model.Procedures {
		if strings.TrimSpace(procedure.ID) == "" {
			return fmt.Errorf("D4+O1 normalized Procedure ID is empty")
		}
		if _, duplicate := addresses[procedure.ID]; duplicate {
			return fmt.Errorf("duplicate D4+O1 Procedure %q", procedure.ID)
		}
		steps := map[string]struct{}{}
		for _, step := range procedure.Steps {
			if strings.TrimSpace(step.LocalKey) == "" {
				return fmt.Errorf("Procedure %q has empty Step local address", procedure.ID)
			}
			if _, duplicate := steps[step.LocalKey]; duplicate {
				return fmt.Errorf("duplicate Step local address %q/%q", procedure.ID, step.LocalKey)
			}
			steps[step.LocalKey] = struct{}{}
			if step.ProtectedApplicationPattern != "" {
				diagnostics := applicationPatternTargetDiagnostics(world, step.LocalKey, []string{step.ProtectedApplicationPattern})
				if err := checkExpectedDiagnostics(diagnostics, ""); err != nil {
					return fmt.Errorf("protected target %q/%q is invalid: %w", procedure.ID, step.LocalKey, err)
				}
			}
		}
		addresses[procedure.ID] = steps
	}

	seen := map[string]struct{}{}
	for _, handoff := range model.Handoffs {
		if !d4O1EndpointExistsProbe(addresses, handoff.From) || !d4O1EndpointExistsProbe(addresses, handoff.To) {
			return fmt.Errorf("handoff endpoint is stale: %s", normalizedHandoffKey(handoff))
		}
		if strings.TrimSpace(handoff.Condition) == "" {
			return fmt.Errorf("handoff condition is empty: %s", normalizedHandoffKey(handoff))
		}
		key := normalizedHandoffKey(handoff)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate normalized Business handoff %s", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func d4O1EndpointExistsProbe(addresses map[string]map[string]struct{}, endpoint canonicalBusinessHandoffEndpointProbe) bool {
	steps, ok := addresses[endpoint.Procedure]
	if !ok {
		return false
	}
	_, ok = steps[endpoint.Step]
	return ok
}

func semanticFingerprintD4O1Probe(t *testing.T, model d4O1NormalizedBusinessModelProbe) string {
	t.Helper()
	content, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	return sourceindex.Fingerprint(content)
}

func currentProtectedTargetsD4O1Probe() map[string][]string {
	return map[string][]string{
		"order-entry.order-handling/BP-06": {"order_entry.submit_order"},
	}
}

func currentReferenceSetAndFixturesD4O1Probe(t *testing.T, root string) (referenceSet, map[string]fixture) {
	t.Helper()
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	return set, fixtures
}

func findD4O1AddressedStepProbe(model d4O1NormalizedBusinessModelProbe, procedureID, stepID string) (d4O1AddressedBusinessStepProbe, bool) {
	for _, procedure := range model.Procedures {
		if procedure.ID != procedureID {
			continue
		}
		for _, step := range procedure.Steps {
			if step.LocalKey == stepID {
				return step, true
			}
		}
	}
	return d4O1AddressedBusinessStepProbe{}, false
}
