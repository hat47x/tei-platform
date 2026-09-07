package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestD3PreservesSocialBusinessWordingWithoutRealizationKind(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	model, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	fixtureSteps := stepsByID(f)
	modelSteps := addressedD3StepsByLocalKey(model)
	for _, id := range []string{"BP-04", "BP-06", "BP-08", "BP-09"} {
		source, ok := fixtureSteps[id]
		if !ok {
			t.Fatalf("Reference fixtureから%sが失われました", id)
		}
		body, ok := modelSteps[id]
		if !ok {
			t.Fatalf("D3 normalized candidateから%sのProcedure-scoped addressが失われました", id)
		}
		if body.BusinessAction != source.BusinessAction || body.BusinessAction == "" {
			t.Fatalf("%sのaccepted Business wordingをrealization subtractionで失いました: source=%q body=%q", id, source.BusinessAction, body.BusinessAction)
		}
		if len(body.ApplicationPatterns) != 0 {
			t.Fatalf("%sへfake Application Patternを補いました: %+v", id, body)
		}
	}

	for _, id := range []string{"BP-04", "BP-08", "BP-09"} {
		if len(modelSteps[id].Capabilities) != 0 {
			t.Fatalf("residual Business step %sへfake Capabilityを補いました: %+v", id, modelSteps[id])
		}
	}
	if !reflect.DeepEqual(modelSteps["BP-06"].Capabilities, []string{"security.authorize"}) {
		t.Fatalf("security boundaryからexisting semantic Capabilityへのexplicit refがD3で失われました: %+v", modelSteps["BP-06"])
	}

	submit, ok := modelSteps["BP-07"]
	if !ok {
		t.Fatal("D3 normalized candidateからBP-07が失われました")
	}
	if !reflect.DeepEqual(submit.ApplicationPatterns, []string{"order_entry.submit_order"}) {
		t.Fatalf("Command interactionへのexplicit Pattern refがD3から失われました: %+v", submit)
	}
}

func TestD3DoesNotTurnAuthoringRealizationLabelsIntoCanonicalKinds(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Steps = append([]step(nil), f.Steps...)
	for i := range changed.Steps {
		switch changed.Steps[i].ID {
		case "BP-06":
			changed.Steps[i].Realization = "security-review-label-reworded"
		case "BP-08":
			changed.Steps[i].Realization = "observation-review-label-reworded"
		case "BP-09":
			changed.Steps[i].Realization = "external-continuation-review-label-reworded"
		}
	}
	after, err := deriveNormalizedBusinessProcedureWithStepAddressProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatal("authoring realization labelだけをD3 Canonical/normalized kindへ昇格しました")
	}
}

func TestD3DoesNotPretendTypedSecurityOrObservationConformanceExists(t *testing.T) {
	bodyType := reflect.TypeOf(canonicalBusinessProcedureStepWithoutIDProbe{})
	for _, forbidden := range []string{"Realization", "Kind", "SecurityBoundary", "ObservationBoundary", "ExternalContinuation"} {
		if _, found := bodyType.FieldByName(forbidden); found {
			t.Fatalf("current D3 bodyへ未検証のtyped boundary field %qを追加しました", forbidden)
		}
	}
}

func addressedD3StepsByLocalKey(model normalizedBusinessProcedureWithStepAddressProbe) map[string]canonicalBusinessProcedureStepWithoutIDProbe {
	result := make(map[string]canonicalBusinessProcedureStepWithoutIDProbe, len(model.Steps))
	for _, item := range model.Steps {
		result[item.LocalKey] = item.Body
	}
	return result
}
