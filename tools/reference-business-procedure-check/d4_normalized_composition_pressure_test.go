package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type d4AddressedBusinessProcedureProbe struct {
	ID      string                         `json:"id"`
	Purpose string                         `json:"purpose"`
	Trigger candidateRef                   `json:"trigger"`
	Actor   candidateRef                   `json:"actor"`
	Steps   []d4AddressedBusinessStepProbe `json:"steps"`
}

type d4AddressedBusinessStepProbe struct {
	LocalKey string                       `json:"localKey"`
	Body     d4BusinessProcedureStepProbe `json:"body"`
}

type d4NormalizedBusinessModelProbe struct {
	Procedures []d4AddressedBusinessProcedureProbe `json:"procedures"`
	Handoffs   []canonicalBusinessHandoffProbe     `json:"handoffs,omitempty"`
}

func TestD4ComposesWithS4AAndH1WithoutGrowingStepBody(t *testing.T) {
	model := currentD4NormalizedBusinessModelProbe(t)
	if err := validateD4NormalizedBusinessModelProbe(model); err != nil {
		t.Fatalf("D4 + S4-A + H1/R3 compositionをnormalized modelだけでvalidateできません: %v", err)
	}

	bodyType := reflect.TypeOf(d4BusinessProcedureStepProbe{})
	if _, found := bodyType.FieldByName("LocalKey"); found {
		t.Fatal("S4-A Step local addressをD4 bodyへ混入しました")
	}
	addressedType := reflect.TypeOf(d4AddressedBusinessStepProbe{})
	if got := fieldNames(addressedType); !reflect.DeepEqual(got, []string{"LocalKey", "Body"}) {
		t.Fatalf("S4-A address layerが膨張しました: got=%v", got)
	}
}

func TestD4CompositionRejectsStaleHandoffAfterLocalAddressRename(t *testing.T) {
	model := currentD4NormalizedBusinessModelProbe(t)
	changed := cloneD4NormalizedBusinessModelProbe(model)
	if !renameD4LocalKey(&changed, "order-entry.order-handling", "BP-09", "BP-09-renamed") {
		t.Fatal("test setupがBP-09を見つけられません")
	}
	if err := validateD4NormalizedBusinessModelProbe(changed); err == nil {
		t.Fatal("D4 bodyを変えずStep local addressだけrenameした後のstale handoffを受理しました")
	}
}

func TestD4CompositionAcceptsCoordinatedAddressAndHandoffRename(t *testing.T) {
	model := currentD4NormalizedBusinessModelProbe(t)
	changed := cloneD4NormalizedBusinessModelProbe(model)
	if !renameD4LocalKey(&changed, "order-entry.order-handling", "BP-09", "BP-09-renamed") {
		t.Fatal("test setupがBP-09を見つけられません")
	}
	changed.Handoffs[0].From.Step = "BP-09-renamed"
	if err := validateD4NormalizedBusinessModelProbe(changed); err != nil {
		t.Fatalf("D4 bodyを変更せずaddress + handoffを協調renameできません: %v", err)
	}
}

func TestD4CompositionKeepsHandoffConditionAsIndependentMeaning(t *testing.T) {
	before := currentD4NormalizedBusinessModelProbe(t)
	after := cloneD4NormalizedBusinessModelProbe(before)
	after.Handoffs[0].Condition = "follow-up is required because unresolved external work remains"
	if reflect.DeepEqual(before, after) {
		t.Fatal("D4 bodyから独立したhandoff condition changeを失いました")
	}
	if err := validateD4NormalizedBusinessModelProbe(after); err != nil {
		t.Fatalf("natural-language handoff conditionをthin normalized relationとして保持できません: %v", err)
	}
}

func TestD4CompositionRejectsDuplicateHandoffAtNormalizedLayer(t *testing.T) {
	model := currentD4NormalizedBusinessModelProbe(t)
	if len(model.Handoffs) == 0 {
		t.Fatal("test setupにhandoffがありません")
	}
	changed := cloneD4NormalizedBusinessModelProbe(model)
	changed.Handoffs = append(changed.Handoffs, changed.Handoffs[0])
	if err := validateD4NormalizedBusinessModelProbe(changed); err == nil {
		t.Fatal("normalize後に混入したduplicate Business handoff relationを受理しました")
	}
}

func TestD4CompositionStillFailsClosedBeforeAddressLayerWhenNewRefKindAppears(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	primary := fixtures["order-entry.order-handling"]
	primary.Steps = append([]step(nil), primary.Steps...)
	for i := range primary.Steps {
		if primary.Steps[i].ID == "BP-04" {
			primary.Steps[i].Data = []string{"sales.order"}
		}
	}
	fixtures["order-entry.order-handling"] = primary
	if _, err := deriveD4NormalizedBusinessModelProbe(set, fixtures); err == nil {
		t.Fatal("new direct Data refをaddress/handoff layerで隠してD4 normalized modelへ進めました")
	}
}

func currentD4NormalizedBusinessModelProbe(t *testing.T) d4NormalizedBusinessModelProbe {
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
	model, err := deriveD4NormalizedBusinessModelProbe(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func deriveD4NormalizedBusinessModelProbe(set referenceSet, fixtures map[string]fixture) (d4NormalizedBusinessModelProbe, error) {
	ids := make([]string, 0, len(fixtures))
	for id := range fixtures {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	result := d4NormalizedBusinessModelProbe{
		Procedures: make([]d4AddressedBusinessProcedureProbe, 0, len(ids)),
	}
	for _, id := range ids {
		f := fixtures[id]
		body, err := deriveD4BusinessProcedureProbe(f)
		if err != nil {
			return d4NormalizedBusinessModelProbe{}, err
		}
		if len(body.Steps) != len(f.Steps) {
			return d4NormalizedBusinessModelProbe{}, fmt.Errorf("D4 source/body step count mismatch for %q", id)
		}
		procedure := d4AddressedBusinessProcedureProbe{
			ID:      body.ID,
			Purpose: body.Purpose,
			Trigger: body.Trigger,
			Actor:   body.Actor,
			Steps:   make([]d4AddressedBusinessStepProbe, 0, len(body.Steps)),
		}
		for i, stepBody := range body.Steps {
			procedure.Steps = append(procedure.Steps, d4AddressedBusinessStepProbe{
				LocalKey: f.Steps[i].ID,
				Body:     stepBody,
			})
		}
		result.Procedures = append(result.Procedures, procedure)
	}

	handoffs, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		return d4NormalizedBusinessModelProbe{}, err
	}
	result.Handoffs = handoffs
	return result, nil
}

func validateD4NormalizedBusinessModelProbe(model d4NormalizedBusinessModelProbe) error {
	addresses := map[string]map[string]struct{}{}
	for _, procedure := range model.Procedures {
		if strings.TrimSpace(procedure.ID) == "" {
			return fmt.Errorf("D4 normalized Procedure ID is empty")
		}
		if _, duplicate := addresses[procedure.ID]; duplicate {
			return fmt.Errorf("duplicate D4 normalized Procedure %q", procedure.ID)
		}
		steps := map[string]struct{}{}
		for _, item := range procedure.Steps {
			if strings.TrimSpace(item.LocalKey) == "" {
				return fmt.Errorf("D4 Procedure %q has empty Step local address", procedure.ID)
			}
			if _, duplicate := steps[item.LocalKey]; duplicate {
				return fmt.Errorf("duplicate D4 Step local address %q/%q", procedure.ID, item.LocalKey)
			}
			steps[item.LocalKey] = struct{}{}
		}
		addresses[procedure.ID] = steps
	}

	seenHandoffs := map[string]struct{}{}
	for _, handoff := range model.Handoffs {
		if !d4EndpointExists(addresses, handoff.From) {
			return fmt.Errorf("D4 handoff source %q/%q not found", handoff.From.Procedure, handoff.From.Step)
		}
		if !d4EndpointExists(addresses, handoff.To) {
			return fmt.Errorf("D4 handoff target %q/%q not found", handoff.To.Procedure, handoff.To.Step)
		}
		if strings.TrimSpace(handoff.Condition) == "" {
			return fmt.Errorf("D4 handoff condition is empty")
		}
		key := normalizedHandoffKey(handoff)
		if _, duplicate := seenHandoffs[key]; duplicate {
			return fmt.Errorf("duplicate D4 normalized Business handoff %s", key)
		}
		seenHandoffs[key] = struct{}{}
	}
	return nil
}

func d4EndpointExists(addresses map[string]map[string]struct{}, endpoint canonicalBusinessHandoffEndpointProbe) bool {
	steps, ok := addresses[endpoint.Procedure]
	if !ok {
		return false
	}
	_, ok = steps[endpoint.Step]
	return ok
}

func cloneD4NormalizedBusinessModelProbe(model d4NormalizedBusinessModelProbe) d4NormalizedBusinessModelProbe {
	result := d4NormalizedBusinessModelProbe{
		Procedures: append([]d4AddressedBusinessProcedureProbe(nil), model.Procedures...),
		Handoffs:   append([]canonicalBusinessHandoffProbe(nil), model.Handoffs...),
	}
	for i := range result.Procedures {
		result.Procedures[i].Steps = append([]d4AddressedBusinessStepProbe(nil), model.Procedures[i].Steps...)
	}
	return result
}

func renameD4LocalKey(model *d4NormalizedBusinessModelProbe, procedureID, from, to string) bool {
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
