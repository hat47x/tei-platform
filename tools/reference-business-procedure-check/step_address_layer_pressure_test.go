package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

type normalizedBusinessProcedureWithStepAddressProbe struct {
	ID      string                                          `json:"id"`
	Purpose string                                          `json:"purpose"`
	Trigger candidateRef                                    `json:"trigger"`
	Actor   candidateRef                                    `json:"actor"`
	Steps   []normalizedBusinessProcedureAddressedStepProbe `json:"steps"`
}

type normalizedBusinessProcedureAddressedStepProbe struct {
	LocalKey string                                       `json:"localKey"`
	Body     canonicalBusinessProcedureStepWithoutIDProbe `json:"body"`
}

func TestNormalizedStepAddressMetadataContainsNoSourceLocator(t *testing.T) {
	addressedType := reflect.TypeOf(normalizedBusinessProcedureAddressedStepProbe{})
	want := []string{"LocalKey", "Body"}
	if got := fieldNames(addressedType); !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized Step address metadataが膨張しました: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{"Path", "Fingerprint", "Range", "Revision", "ETag"} {
		if _, found := addressedType.FieldByName(forbidden); found {
			t.Fatalf("Canonical/normalized Step addressへSource locator field %qが侵入しました", forbidden)
		}
	}
}

func TestCanonicalHandoffCanResolveAgainstNormalizedProcedureAddressMetadataWithoutSourceIndex(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}

	procedures := map[string]normalizedBusinessProcedureWithStepAddressProbe{}
	for id, f := range fixtures {
		meaning, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(f)
		if err != nil {
			t.Fatal(err)
		}
		procedures[id] = meaning
	}

	handoffs, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	for _, handoff := range handoffs {
		if !normalizedStepAddressExists(procedures, handoff.From) {
			t.Fatalf("normalized meaningだけでhandoff sourceを解決できません: %+v", handoff.From)
		}
		if !normalizedStepAddressExists(procedures, handoff.To) {
			t.Fatalf("normalized meaningだけでhandoff targetを解決できません: %+v", handoff.To)
		}
	}
}

func TestStepLocalKeyRenameChangesAddressStructureButNotBusinessBodyMeaning(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	beforeBody, err := deriveCanonicalBusinessProcedureBodyWithoutStepIDProbe(f)
	if err != nil {
		t.Fatal(err)
	}
	beforeAddressed, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-04" {
			changed.Steps[i].ID = "BP-04-renamed"
		}
	}
	afterBody, err := deriveCanonicalBusinessProcedureBodyWithoutStepIDProbe(changed)
	if err != nil {
		t.Fatal(err)
	}
	afterAddressed, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(beforeBody, afterBody) {
		t.Fatal("Step local key renameだけでBusiness body meaningまで変更しました")
	}
	if reflect.DeepEqual(beforeAddressed, afterAddressed) {
		t.Fatal("Step local key renameをreferential/address structure changeとして観測できません")
	}
}

func TestIDFreeBodyAloneCannotResolveCanonicalHandoffEndpoint(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	body, err := deriveCanonicalBusinessProcedureBodyWithoutStepIDProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	bodyType := reflect.TypeOf(body.Steps[0])
	if _, found := bodyType.FieldByName("LocalKey"); found {
		t.Fatal("ID-free bodyへStep address metadataが混入しました")
	}
	if _, found := bodyType.FieldByName("ID"); found {
		t.Fatal("ID-free bodyへStep IDが混入しました")
	}

	// A Canonical handoff endpoint carries Procedure+Step address. Without any
	// normalized address metadata, resolving that endpoint would require an
	// external Source-side index. This test intentionally proves the absence of
	// such addressability in the body-only projection rather than silently
	// inventing an ordinal or similarity match.
}

func deriveNormalizedBusinessProcedureWithStepAddressProbe(f fixture) (normalizedBusinessProcedureWithStepAddressProbe, error) {
	body, err := deriveCanonicalBusinessProcedureBodyWithoutStepIDProbe(f)
	if err != nil {
		return normalizedBusinessProcedureWithStepAddressProbe{}, err
	}
	result := normalizedBusinessProcedureWithStepAddressProbe{
		ID:      body.ID,
		Purpose: body.Purpose,
		Trigger: body.Trigger,
		Actor:   body.Actor,
		Steps:   make([]normalizedBusinessProcedureAddressedStepProbe, 0, len(body.Steps)),
	}
	for i, item := range body.Steps {
		result.Steps = append(result.Steps, normalizedBusinessProcedureAddressedStepProbe{
			LocalKey: f.Steps[i].ID,
			Body:     item,
		})
	}
	return result, nil
}

func normalizedStepAddressExists(procedures map[string]normalizedBusinessProcedureWithStepAddressProbe, endpoint canonicalBusinessHandoffEndpointProbe) bool {
	procedure, ok := procedures[endpoint.Procedure]
	if !ok {
		return false
	}
	for _, item := range procedure.Steps {
		if item.LocalKey == endpoint.Step {
			return true
		}
	}
	return false
}
