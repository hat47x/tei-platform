package derivednamecheck

import (
	"strings"
	"testing"
)

func loadRules(t *testing.T) Rules {
	t.Helper()
	rules, err := LoadRules("testdata/rules.json")
	if err != nil {
		t.Fatal(err)
	}
	return rules
}

func input(elements ...Element) Input {
	return Input{Schema: NamesSchema, Elements: elements}
}

func run(t *testing.T, in Input, targets ...string) Report {
	t.Helper()
	report, err := Check(loadRules(t), in, targets)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func find(report Report, kind, layer string) *Violation {
	for i := range report.Violations {
		if report.Violations[i].Kind == kind && report.Violations[i].Layer == layer {
			return &report.Violations[i]
		}
	}
	return nil
}

func ids(v *Violation) []string {
	var out []string
	for _, o := range v.Occurrences {
		out = append(out, o.ElementID)
	}
	return out
}

func TestPositiveInputIsNotRejected(t *testing.T) {
	in, err := LoadInput("testdata/input-ok.json")
	if err != nil {
		t.Fatal(err)
	}
	report := run(t, in, "postgres", "go", "windows-files")
	if report.Status != StatusPassed || len(report.Violations) != 0 {
		t.Fatalf("positive input rejected: %+v", report.Violations)
	}
	if len(report.Evaluated) != 3 {
		t.Fatalf("evaluated = %v", report.Evaluated)
	}
}

func TestNoTargetIsUnevaluatedNotPassed(t *testing.T) {
	report := run(t, input(Element{"a", "x"}))
	if report.Status != StatusUnevaluated || len(report.Evaluated) != 0 {
		t.Fatalf("status = %s", report.Status)
	}
}

func TestUnknownTargetIsError(t *testing.T) {
	_, err := Check(loadRules(t), input(Element{"a", "x"}), []string{"oracle"})
	if err == nil {
		t.Fatal("unknown target accepted")
	}
}

func TestCaseOnlyCollisionReportsBothElementsAndLayer(t *testing.T) {
	report := run(t, input(Element{"ent-1", "Customer"}, Element{"ent-2", "customer"}), "postgres")
	v := find(report, KindCollision, "postgres-table")
	if v == nil || strings.Join(ids(v), ",") != "ent-1,ent-2" || v.LayerVersion != "1" || v.Target != "postgres" {
		t.Fatalf("violations = %+v", report.Violations)
	}
	if report.Status != StatusRejected {
		t.Fatalf("status = %s", report.Status)
	}
}

func TestCaseOnlyCollisionIsLayerSpecific(t *testing.T) {
	// The Windows file layer folds case too, so use a case-sensitive probe:
	// only layers that fold case may report this pair.
	rules := loadRules(t)
	for i := range rules.Layers {
		rules.Layers[i].FoldCase = false
	}
	report, err := Check(rules, input(Element{"ent-1", "Customer"}, Element{"ent-2", "customer"}), []string{"postgres"})
	if err != nil || report.Status != StatusPassed {
		t.Fatalf("case-sensitive layer rejected case-only difference: %v %+v", err, report.Violations)
	}
}

func TestSeparatorOnlyCollisionOnlyWhereSeparatorsIgnored(t *testing.T) {
	in := input(Element{"ent-1", "order_line"}, Element{"ent-2", "orderline"})
	if report := run(t, in, "postgres"); report.Status != StatusPassed {
		t.Fatalf("postgres layer keeps underscores distinct: %+v", report.Violations)
	}
	v := find(run(t, in, "go"), KindCollision, "go-type")
	if v == nil || strings.Join(ids(v), ",") != "ent-1,ent-2" {
		t.Fatal("go layer did not report the separator collision")
	}
}

func TestSuffixCollisionWithDerivedName(t *testing.T) {
	report := run(t, input(Element{"ent-1", "invoice"}, Element{"ent-2", "invoice_key"}), "postgres")
	v := find(report, KindCollision, "postgres-table")
	if v == nil {
		t.Fatalf("violations = %+v", report.Violations)
	}
	var viaDerivation string
	for _, o := range v.Occurrences {
		if o.ElementID == "ent-1" {
			viaDerivation = o.Derivation
		}
	}
	if viaDerivation != "unique-key" {
		t.Fatalf("derivation = %q", viaDerivation)
	}
}

func TestReservedWordIsSingleElementViolation(t *testing.T) {
	report := run(t, input(Element{"ent-1", "Order"}), "postgres")
	v := find(report, KindReserved, "postgres-table")
	if v == nil || len(v.Occurrences) != 1 || v.Occurrences[0].ElementID != "ent-1" {
		t.Fatalf("violations = %+v", report.Violations)
	}
	// The same name is fine for a target whose layer does not reserve it.
	if r := run(t, input(Element{"ent-1", "Order"}), "windows-files"); r.Status != StatusPassed {
		t.Fatalf("other target rejected: %+v", r.Violations)
	}
}

func TestMaxLengthCountsDerivedName(t *testing.T) {
	ok := strings.Repeat("a", 58) // 58 + len("_pkey") = 63
	long := strings.Repeat("b", 59)
	report := run(t, input(Element{"ent-1", ok}, Element{"ent-2", long}), "postgres")
	var tooLong []*Violation
	for i := range report.Violations {
		if report.Violations[i].Kind == KindTooLong {
			tooLong = append(tooLong, &report.Violations[i])
		}
	}
	if len(tooLong) == 0 {
		t.Fatal("overlong derived name not reported")
	}
	for _, v := range tooLong {
		if v.Occurrences[0].ElementID != "ent-2" {
			t.Fatalf("wrong element reported: %+v", v)
		}
	}
}

func TestInvalidCharacters(t *testing.T) {
	report := run(t, input(Element{"ent-1", "売上"}), "postgres")
	if find(report, KindInvalidChars, "postgres-table") == nil {
		t.Fatalf("violations = %+v", report.Violations)
	}
}

func TestRenameIsCheckedWithSameRules(t *testing.T) {
	in := input(Element{"ent-1", "customer"}, Element{"ent-2", "client"})
	if run(t, in, "postgres").Status != StatusPassed {
		t.Fatal("before rename should pass")
	}
	in.Renames = []Element{{"ent-2", "CUSTOMER"}}
	report := run(t, in, "postgres")
	if find(report, KindCollision, "postgres-table") == nil {
		t.Fatalf("rename collision not reported: %+v", report.Violations)
	}
	if len(report.Renamed) != 1 || report.Renamed[0] != "ent-2" {
		t.Fatalf("renamed = %v", report.Renamed)
	}
}

func TestCheckDoesNotChangeInput(t *testing.T) {
	in := input(Element{"ent-1", "Customer"}, Element{"ent-2", "customer"})
	in.Renames = []Element{{"ent-2", "other"}}
	run(t, in, "postgres")
	if in.Elements[1].Name != "customer" || in.Elements[0].ID != "ent-1" {
		t.Fatal("input mutated")
	}
}

func TestInvalidInputsAndRules(t *testing.T) {
	if _, err := Check(loadRules(t), input(Element{"a", "x"}, Element{"a", "y"}), []string{"go"}); err == nil {
		t.Fatal("duplicate element id accepted")
	}
	bad := input(Element{"a", "x"})
	bad.Renames = []Element{{"missing", "y"}}
	if _, err := Check(loadRules(t), bad, []string{"go"}); err == nil {
		t.Fatal("rename of unknown element accepted")
	}
	rules := loadRules(t)
	rules.Layers[0].Derivations = append(rules.Layers[0].Derivations, Derivation{ID: "dup", Suffix: "_KEY"})
	if err := rules.Validate(); err == nil {
		t.Fatal("derivations building the same name accepted")
	}
}
