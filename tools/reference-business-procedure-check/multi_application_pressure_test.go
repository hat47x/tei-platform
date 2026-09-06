package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

func TestBusinessProcedureCanReferencePatternsAcrossApplications(t *testing.T) {
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
		ID:             "BP-XA",
		BusinessAction: "inspect the same order from a separate fulfillment application context",
		Realization:    "application-use",
		ApplicationPatterns: []string{
			"fulfillment_console.order_record",
		},
	})

	result := evaluate(loaded.World, f)
	for _, item := range result.Diagnostics {
		if item.Step == "BP-XA" {
			t.Fatalf("cross-Application Pattern referenceがdiagnosticになりました: %+v", item)
		}
	}

	rows := patternRowsForStep(result, "BP-XA")
	if len(rows) != 1 || rows[0].Pattern != "fulfillment_console.order_record" || rows[0].Data != "sales.order" {
		t.Fatalf("second Application Patternのmeaningをcross-view導出できません: %+v", rows)
	}
}

func TestThinCanonicalBusinessMeaningDoesNotCopySingleApplicationContainer(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	meaning, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		t.Fatal(err)
	}

	meaningType := reflect.TypeOf(meaning)
	if _, found := meaningType.FieldByName("Application"); found {
		t.Fatal("thin Canonical Business meaningを単一Application containerへ固定しました")
	}
}

func TestCrossApplicationPatternReferenceRemainsExplicitQualifiedIdentity(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	f.Steps = append(f.Steps, step{
		ID:             "BP-XA",
		BusinessAction: "inspect order in fulfillment context",
		Realization:    "application-use",
		ApplicationPatterns: []string{
			"fulfillment_console.order_record",
		},
	})
	meaning, err := deriveCanonicalBusinessProcedureMeaningProbe(f)
	if err != nil {
		t.Fatal(err)
	}
	item, ok := canonicalMeaningStep(meaning, "BP-XA")
	if !ok {
		t.Fatal("cross-Application Business stepがthin meaningから失われました")
	}
	want := []string{"fulfillment_console.order_record"}
	if !reflect.DeepEqual(item.ApplicationPatterns, want) {
		t.Fatalf("cross-Application Pattern identityを明示参照として保持できません: got=%v want=%v", item.ApplicationPatterns, want)
	}
}

func addSyntheticSecondApplication(t *testing.T, workspaceRoot string) {
	t.Helper()
	unit := filepath.Join(workspaceRoot, "applications", "fulfillment-console")
	patterns := filepath.Join(unit, "patterns")
	if err := os.MkdirAll(patterns, 0o755); err != nil {
		t.Fatal(err)
	}
	applicationSource := `application fulfillment_console {
    uses semantic sales

    database sales {
        target sales_core
        access read
        requires {
            contract sales_core.order ^4
        }
    }
}
`
	if err := os.WriteFile(filepath.Join(unit, "application.tei"), []byte(applicationSource), 0o600); err != nil {
		t.Fatal(err)
	}
	patternSource := `pattern fulfillment_console.order_record record {
    data sales.order
}
`
	if err := os.WriteFile(filepath.Join(patterns, "order-record.tei"), []byte(patternSource), 0o600); err != nil {
		t.Fatal(err)
	}
}

func patternRowsForStep(value report, stepID string) []patternMeaningRow {
	var result []patternMeaningRow
	for _, item := range value.BusinessActionPatternMeaningRows {
		if item.StepID == stepID {
			result = append(result, item)
		}
	}
	return result
}
