package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/database"
	"tei/internal/workspace"
)

func TestFixtureApplicationContextIsNotThinCanonicalBusinessMeaning(t *testing.T) {
	path := filepath.Join(repoPath(), "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	changed := f
	changed.Application = "some-other-reference-context"
	after, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(changed)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(before, after) {
		t.Fatal("fixture-level Application contextをthin Canonical Business meaningへ混ぜました")
	}
}

func TestFixtureApplicationContextCanBeAbsentWithoutBlockingCrossViewEvaluation(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	f.Application = ""

	result := evaluate(loaded.World, f)
	if err := checkExpectedDiagnostics(result.Diagnostics, ""); err != nil {
		t.Fatalf("primary Application contextなしでblocking cross-view diagnosticが出ました: %v\n%+v", err, result.Diagnostics)
	}
	if result.Application != "" {
		t.Fatalf("absent reporting contextを別Applicationへ補完しました: %q", result.Application)
	}

	rows := patternRowsForStep(result, "BP-01")
	if len(rows) != 1 || rows[0].Data != "sales.order" {
		t.Fatalf("primary Application contextなしでexplicit Pattern refからData meaningを導出できません: %+v", rows)
	}
}

func TestUnknownFixtureApplicationContextIsNotSemanticOwnershipAuthority(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	f.Application = "reporting-context-not-present-in-world"

	result := evaluate(loaded.World, f)
	if err := checkExpectedDiagnostics(result.Diagnostics, ""); err != nil {
		t.Fatalf("reporting contextをsemantic ownerとしてblocking validationしました: %v\n%+v", err, result.Diagnostics)
	}
	if result.Application != f.Application {
		t.Fatalf("reporting context labelを暗黙補正しました: got=%q want=%q", result.Application, f.Application)
	}
}

func TestCrossApplicationAccessProbeDoesNotDependOnFixturePrimaryApplication(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	addSyntheticSecondApplicationWithWritableMaster(t, probeRoot)
	loaded, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}

	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	f.Application = "fulfillment_console"
	f.Steps = append(f.Steps, step{
		ID:             "BP-XACCESS-CONTEXT",
		BusinessAction: "update customer information using the Pattern owning application's accepted access",
		Realization:    "application-use",
		ApplicationPatterns: []string{
			"fulfillment_console.customer_reference",
		},
	})

	result := evaluate(loaded.World, f)
	applyAccessProbe(&result, loaded.World, f, accessProbe{
		SchemaVersion:      probeSchemaVersion,
		Procedure:          f.Procedure.ID,
		Step:               "BP-XACCESS-CONTEXT",
		ApplicationPattern: "fulfillment_console.customer_reference",
		RequestedAccess:    accessIntent{Required: database.AccessWrite},
		Reason:             "fixture primary context must not substitute for Pattern ownership",
	})
	for _, item := range result.Diagnostics {
		if item.Step == "BP-XACCESS-CONTEXT" {
			t.Fatalf("fixture-level Application contextをcross-view access authorityとして使用しました: %+v", item)
		}
	}
}

func TestFixtureApplicationContextCanExistForReferenceReportingWithoutBecomingProcedureOwner(t *testing.T) {
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
	f.Steps = append(f.Steps, step{
		ID:             "BP-XA-CONTEXT",
		BusinessAction: "review the order from the fulfillment application",
		Realization:    "application-use",
		ApplicationPatterns: []string{
			"fulfillment_console.order_record",
		},
	})

	result := evaluate(loaded.World, f)
	if result.Application != "order_entry" {
		t.Fatalf("Reference reporting context=%q want=order_entry", result.Application)
	}
	if _, _, ok := findPatternOwner(loaded.World, "fulfillment_console.order_record"); !ok {
		t.Fatal("second Application Pattern ownershipをnormalized Worldで解決できません")
	}

	meaning, err := deriveCanonicalBusinessProcedureMeaningWithoutRealizationProbe(f)
	if err != nil {
		t.Fatal(err)
	}
	meaningType := reflect.TypeOf(meaning)
	if _, found := meaningType.FieldByName("Application"); found {
		t.Fatal("Reference reporting contextをProcedure owner fieldへ昇格しました")
	}
}
