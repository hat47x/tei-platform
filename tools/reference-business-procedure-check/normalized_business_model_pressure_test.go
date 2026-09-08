package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type normalizedBusinessModelProbe struct {
	Procedures []normalizedBusinessProcedureWithStepAddressProbe `json:"procedures"`
	Handoffs   []canonicalBusinessHandoffProbe                    `json:"handoffs,omitempty"`
}

func TestNormalizedBusinessModelProbeKeepsProcedureAndHandoffWithoutSourceInfrastructure(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}

	model, err := deriveNormalizedBusinessModelProbe(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateNormalizedBusinessModelProbe(model); err != nil {
		t.Fatalf("normalized Business modelだけでreferential integrityを検査できません: %v", err)
	}
	if len(model.Procedures) != 2 || len(model.Handoffs) != 1 {
		t.Fatalf("unexpected normalized Business model: procedures=%d handoffs=%d", len(model.Procedures), len(model.Handoffs))
	}
}

func TestNormalizedBusinessModelProbeHasNoSourceOrRuntimeContainerFields(t *testing.T) {
	modelType := reflect.TypeOf(normalizedBusinessModelProbe{})
	want := []string{"Procedures", "Handoffs"}
	if got := fieldNames(modelType); !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized Business model boundaryが膨張しました: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{"Source", "Path", "Fingerprint", "Revision", "Runtime", "Execution", "Workflow"} {
		if _, found := modelType.FieldByName(forbidden); found {
			t.Fatalf("normalized Business modelへSource/runtime field %qが侵入しました", forbidden)
		}
	}
}

func TestNormalizedBusinessModelIsInvariantToReferenceArtifactLocatorMove(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveNormalizedBusinessModelProbe(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}

	moved := set
	moved.Procedures = append([]referenceSetProcedure(nil), set.Procedures...)
	for i := range moved.Procedures {
		moved.Procedures[i].Artifact = "moved/" + moved.Procedures[i].Artifact
	}
	after, err := deriveNormalizedBusinessModelProbe(moved, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("Reference artifact locatorだけでnormalized Business modelが変化しました: before=%+v after=%+v", before, after)
	}
}

func TestNormalizedBusinessModelRejectsStaleHandoffAfterLocalStepAddressRename(t *testing.T) {
	model := currentNormalizedBusinessModelProbe(t)
	changed := cloneNormalizedBusinessModelProbe(model)
	if !renameNormalizedStepLocalKey(&changed, "order-entry.order-handling", "BP-09", "BP-09-renamed") {
		t.Fatal("test setupがBP-09を見つけられません")
	}
	if err := validateNormalizedBusinessModelProbe(changed); err == nil {
		t.Fatal("normalized Step local key rename後のstale handoffをSource Indexや位置で補完しました")
	}
}

func TestNormalizedBusinessModelAcceptsCoordinatedLocalAddressAndHandoffRename(t *testing.T) {
	model := currentNormalizedBusinessModelProbe(t)
	changed := cloneNormalizedBusinessModelProbe(model)
	if !renameNormalizedStepLocalKey(&changed, "order-entry.order-handling", "BP-09", "BP-09-renamed") {
		t.Fatal("test setupがBP-09を見つけられません")
	}
	changed.Handoffs[0].From.Step = "BP-09-renamed"
	if err := validateNormalizedBusinessModelProbe(changed); err != nil {
		t.Fatalf("Procedure-scoped local addressとhandoff endpointを同時更新しても解決できません: %v", err)
	}
}

func TestNormalizedBusinessModelRejectsMissingHandoffEndpointWithoutSourceIndex(t *testing.T) {
	model := currentNormalizedBusinessModelProbe(t)
	changed := cloneNormalizedBusinessModelProbe(model)
	for i := range changed.Procedures {
		if changed.Procedures[i].ID != "order-entry.follow-up" {
			continue
		}
		steps := changed.Procedures[i].Steps[:0]
		for _, item := range changed.Procedures[i].Steps {
			if item.LocalKey != "FU-00" {
				steps = append(steps, item)
			}
		}
		changed.Procedures[i].Steps = steps
	}
	if err := validateNormalizedBusinessModelProbe(changed); err == nil {
		t.Fatal("存在しないnormalized Step endpointへのhandoffを受理しました")
	}
}

func TestNormalizedBusinessModelKeepsHandoffConditionAsSemanticRelationMeaning(t *testing.T) {
	before := currentNormalizedBusinessModelProbe(t)
	after := cloneNormalizedBusinessModelProbe(before)
	after.Handoffs[0].Condition = "follow-up is required because unresolved delivery work remains"
	if reflect.DeepEqual(before, after) {
		t.Fatal("handoff condition changeをnormalized Business model差として観測できません")
	}
	if err := validateNormalizedBusinessModelProbe(after); err != nil {
		t.Fatalf("自然言語conditionをBusiness meaningとして保持できません: %v", err)
	}
}

func currentNormalizedBusinessModelProbe(t *testing.T) normalizedBusinessModelProbe {
	t.Helper()
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	model, err := deriveNormalizedBusinessModelProbe(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func deriveNormalizedBusinessModelProbe(set referenceSet, fixtures map[string]fixture) (normalizedBusinessModelProbe, error) {
	ids := make([]string, 0, len(fixtures))
	for id := range fixtures {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	result := normalizedBusinessModelProbe{
		Procedures: make([]normalizedBusinessProcedureWithStepAddressProbe, 0, len(ids)),
		Handoffs:   make([]canonicalBusinessHandoffProbe, 0, len(set.Handoffs)),
	}
	for _, id := range ids {
		procedure, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(fixtures[id])
		if err != nil {
			return normalizedBusinessModelProbe{}, err
		}
		result.Procedures = append(result.Procedures, procedure)
	}
	for _, item := range set.Handoffs {
		result.Handoffs = append(result.Handoffs, canonicalBusinessHandoffProbe{
			From: canonicalBusinessHandoffEndpointProbe{
				Procedure: item.From.Procedure,
				Step:      item.From.Step,
			},
			To: canonicalBusinessHandoffEndpointProbe{
				Procedure: item.To.Procedure,
				Step:      item.To.Step,
			},
			Condition: item.Condition,
		})
	}
	sort.Slice(result.Handoffs, func(i, j int) bool {
		return normalizedHandoffKey(result.Handoffs[i]) < normalizedHandoffKey(result.Handoffs[j])
	})
	return result, nil
}

func validateNormalizedBusinessModelProbe(model normalizedBusinessModelProbe) error {
	addresses := map[string]map[string]struct{}{}
	for _, procedure := range model.Procedures {
		if strings.TrimSpace(procedure.ID) == "" {
			return fmt.Errorf("normalized Business Procedure ID is empty")
		}
		if _, duplicate := addresses[procedure.ID]; duplicate {
			return fmt.Errorf("duplicate normalized Business Procedure %q", procedure.ID)
		}
		steps := map[string]struct{}{}
		for _, item := range procedure.Steps {
			if strings.TrimSpace(item.LocalKey) == "" {
				return fmt.Errorf("normalized Business Procedure %q has empty Step local address", procedure.ID)
			}
			if _, duplicate := steps[item.LocalKey]; duplicate {
				return fmt.Errorf("duplicate normalized Step address %q/%q", procedure.ID, item.LocalKey)
			}
			steps[item.LocalKey] = struct{}{}
		}
		addresses[procedure.ID] = steps
	}

	seenHandoffs := map[string]struct{}{}
	for _, handoff := range model.Handoffs {
		if !normalizedEndpointExists(addresses, handoff.From) {
			return fmt.Errorf("normalized handoff source %q/%q not found", handoff.From.Procedure, handoff.From.Step)
		}
		if !normalizedEndpointExists(addresses, handoff.To) {
			return fmt.Errorf("normalized handoff target %q/%q not found", handoff.To.Procedure, handoff.To.Step)
		}
		if strings.TrimSpace(handoff.Condition) == "" {
			return fmt.Errorf("normalized handoff %s has no Business condition", normalizedHandoffKey(handoff))
		}
		key := normalizedHandoffKey(handoff)
		if _, duplicate := seenHandoffs[key]; duplicate {
			return fmt.Errorf("duplicate normalized Business handoff %s", key)
		}
		seenHandoffs[key] = struct{}{}
	}
	return nil
}

func normalizedEndpointExists(addresses map[string]map[string]struct{}, endpoint canonicalBusinessHandoffEndpointProbe) bool {
	steps, ok := addresses[endpoint.Procedure]
	if !ok {
		return false
	}
	_, ok = steps[endpoint.Step]
	return ok
}

func normalizedHandoffKey(handoff canonicalBusinessHandoffProbe) string {
	return handoff.From.Procedure + "/" + handoff.From.Step + "->" + handoff.To.Procedure + "/" + handoff.To.Step + ":" + handoff.Condition
}

func cloneNormalizedBusinessModelProbe(model normalizedBusinessModelProbe) normalizedBusinessModelProbe {
	result := normalizedBusinessModelProbe{
		Procedures: append([]normalizedBusinessProcedureWithStepAddressProbe(nil), model.Procedures...),
		Handoffs:   append([]canonicalBusinessHandoffProbe(nil), model.Handoffs...),
	}
	for i := range result.Procedures {
		result.Procedures[i].Steps = append([]normalizedBusinessProcedureAddressedStepProbe(nil), model.Procedures[i].Steps...)
	}
	return result
}

func renameNormalizedStepLocalKey(model *normalizedBusinessModelProbe, procedureID, from, to string) bool {
	for i := range model.Procedures {
		if model.Procedures[i].ID != procedureID {
			continue
		}
		for j := range model.Procedures[i].Steps {
			if model.Procedures[i].Steps[j].LocalKey == from {
				model.Procedures[i].Steps[j].LocalKey = to
				return true
			}
		}
	}
	return false
}
