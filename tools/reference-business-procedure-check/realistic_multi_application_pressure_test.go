package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/workspace"
)

func TestBusinessProcedureCanSpanTwoApplicationsManualAndExternalWork(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	addSyntheticSecondApplication(t, probeRoot)

	loaded, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	f.Steps = append(f.Steps,
		step{
			ID:             "BP-XA",
			BusinessAction: "review whether the accepted order can proceed through fulfillment",
			Realization:    "application-use",
			ApplicationPatterns: []string{
				"fulfillment_console.order_record",
			},
		},
		step{
			ID:             "BP-XM",
			BusinessAction: "coordinate physical preparation of the order with warehouse staff",
			Realization:    "manual-external-unresolved",
		},
		step{
			ID:             "BP-XE",
			BusinessAction: "continue shipment-related work with the external party responsible for delivery when required",
			Realization:    "manual-external-unresolved",
		},
	)

	result := evaluate(loaded.World, f)
	for _, item := range result.Diagnostics {
		if item.Step == "BP-XA" || item.Step == "BP-XM" || item.Step == "BP-XE" {
			t.Fatalf("multi-Application/manual/external Business Procedureがdiagnosticになりました: %+v", item)
		}
	}

	meaning, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(f)
	if err != nil {
		t.Fatal(err)
	}
	fulfillment, ok := canonicalMeaningStepWithoutRealization(meaning, "BP-XA")
	if !ok {
		t.Fatal("fulfillment Application stepがthin meaningから失われました")
	}
	if !reflect.DeepEqual(fulfillment.ApplicationPatterns, []string{"fulfillment_console.order_record"}) {
		t.Fatalf("fulfillment Application Pattern refを保持できません: %+v", fulfillment)
	}

	for _, id := range []string{"BP-XM", "BP-XE"} {
		manual, ok := canonicalMeaningStepWithoutRealization(meaning, id)
		if !ok {
			t.Fatalf("%s manual/external Business Actionがthin meaningから失われました", id)
		}
		if manual.BusinessAction == "" {
			t.Fatalf("%s Business Action wordingが空です", id)
		}
		if len(manual.ApplicationPatterns) != 0 || len(manual.Capabilities) != 0 {
			t.Fatalf("%s manual/external workへfake Function / Capabilityを付与しました: %+v", id, manual)
		}
	}

	applications := applicationIDsFromPatternRefs(meaning)
	wantApplications := []string{"fulfillment_console", "order_entry"}
	if !reflect.DeepEqual(applications, wantApplications) {
		t.Fatalf("Procedureが参照するApplication集合をqualified Pattern refsから導出できません: got=%v want=%v", applications, wantApplications)
	}
}

func TestRealisticCrossApplicationPressureDoesNotRequireApplicationOwnerInCanonicalBody(t *testing.T) {
	meaningType := reflect.TypeOf(canonicalBusinessProcedureMeaningWithoutRealizationProbe{})
	if _, found := meaningType.FieldByName("Application"); found {
		t.Fatal("multi-Application/manual/external Procedureを単一Application ownerへ押し込みました")
	}
}

func TestRealisticCrossApplicationPressureKeepsBusinessActorSeparateFromSecurity(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	meaning, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(f)
	if err != nil {
		t.Fatal(err)
	}
	if meaning.Actor.Label != "sales operator" || meaning.Actor.Status != "candidate" {
		t.Fatalf("Business actor candidateが変質しました: %+v", meaning.Actor)
	}
}

func applicationIDsFromPatternRefs(meaning canonicalBusinessProcedureMeaningWithoutRealizationProbe) []string {
	set := map[string]struct{}{}
	for _, item := range meaning.Steps {
		for _, ref := range item.ApplicationPatterns {
			for i := 0; i < len(ref); i++ {
				if ref[i] == '.' {
					if i > 0 {
						set[ref[:i]] = struct{}{}
					}
					break
				}
			}
		}
	}
	result := make([]string, 0, len(set))
	for id := range set {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
