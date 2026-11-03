package designoutput

import (
	"strings"
	"testing"

	accessmatrix "tei/tools/reference-access-matrix"
)

func declaredMatrix() accessmatrix.Matrix {
	return accessmatrix.Matrix{
		Schema:            accessmatrix.Schema,
		StructureRevision: "sha256:fixture",
		Cells: []accessmatrix.Cell{
			{Application: "app", Data: "data", Access: []string{"create", "read"}},
		},
	}
}

func TestUnobservedEffectsReportKeepsDeclaredOnly(t *testing.T) {
	effects, err := buildEffects(declaredMatrix(), false, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if effects.Observed {
		t.Fatal("effects were reported as observed")
	}
	if len(effects.Declared) != 1 || effects.Declared[0].Application != "app" {
		t.Fatalf("declared=%+v", effects.Declared)
	}
	if len(effects.Differences) != 0 {
		t.Fatalf("unobserved differences=%v", effects.Differences)
	}
	if !strings.Contains(effects.Note, "観測していません") {
		t.Fatalf("note=%q", effects.Note)
	}
}

func TestObservedEffectsWithoutObservationFlagIsRejected(t *testing.T) {
	_, err := buildEffects(declaredMatrix(), false, "", []ObservedEffect{{Application: "app", Data: "data", Access: "create"}})
	if err == nil || !strings.Contains(err.Error(), "without marking the observation") {
		t.Fatalf("err=%v", err)
	}
}

func TestObservedCreateMatchesDeclaredCreate(t *testing.T) {
	effects, err := buildEffects(declaredMatrix(), true, "audit-log", []ObservedEffect{
		{Application: "app", Data: "data", Access: "create"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(effects.Differences) != 0 {
		t.Fatalf("differences=%v", effects.Differences)
	}
	// Read is not a side effect, so it is not expected in the observation.
	if len(effects.ObservedEffects) != 1 {
		t.Fatalf("observed=%+v", effects.ObservedEffects)
	}
}

func TestUndeclaredObservedUpdateIsReported(t *testing.T) {
	effects, err := buildEffects(declaredMatrix(), true, "audit-log", []ObservedEffect{
		{Application: "app", Data: "data", Access: "update"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(effects.Differences) != 2 {
		t.Fatalf("differences=%v", effects.Differences)
	}
	joined := strings.Join(effects.Differences, "\n")
	if !strings.Contains(joined, "undeclared-effect:app:data:update") {
		t.Fatalf("missing undeclared effect: %v", effects.Differences)
	}
	if !strings.Contains(joined, "unobserved-declared-effect:app:data:create") {
		t.Fatalf("missing unobserved declared effect: %v", effects.Differences)
	}
}

func TestDeclaredEffectWithoutObservationIsReported(t *testing.T) {
	effects, err := buildEffects(declaredMatrix(), true, "audit-log", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(effects.Differences) != 1 || effects.Differences[0] != "unobserved-declared-effect:app:data:create" {
		t.Fatalf("differences=%v", effects.Differences)
	}
}

func TestEffectsReportSeparatesSections(t *testing.T) {
	effects, err := buildEffects(declaredMatrix(), false, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	report := renderEffectsReport(effects, "sha256:fixture")
	for _, want := range []string{"## 宣言された利用", "## 観測された副作用", "未観測です", "## 差"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestEffectsReportShowsObservedAndDifferences(t *testing.T) {
	effects, err := buildEffects(declaredMatrix(), true, "audit-log", []ObservedEffect{
		{Application: "app", Data: "data", Access: "update"},
	})
	if err != nil {
		t.Fatal(err)
	}
	report := renderEffectsReport(effects, "sha256:fixture")
	for _, want := range []string{"audit-log", "undeclared-effect:app:data:update", "unobserved-declared-effect:app:data:create"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestDeclaredAndObservedAreNotMerged(t *testing.T) {
	effects, err := buildEffects(declaredMatrix(), true, "audit-log", []ObservedEffect{
		{Application: "app", Data: "data", Access: "update"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, cell := range effects.Declared {
		for _, access := range cell.Access {
			if access == "update" {
				t.Fatalf("the observed update leaked into the declared access: %+v", cell)
			}
		}
	}
}
