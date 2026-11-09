package dataimportcheck

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func world() ir.World {
	return ir.World{Semantics: []semantic.Namespace{{ID: "s",
		Types: []semantic.Type{
			{ID: "std.integer", Kind: semantic.TypePrimitive},
			{ID: "std.decimal", Kind: semantic.TypePrimitive},
			{ID: "std.date", Kind: semantic.TypePrimitive},
			{ID: "std.text", Kind: semantic.TypePrimitive},
			{ID: "s.qty", Kind: semantic.TypeAlias, Underlying: "std.decimal"},
		},
		Data: []semantic.Data{
			{ID: "s.line", Symbol: "Line", Attributes: []semantic.Attribute{
				{ID: "s.line.no", Symbol: "no", TypeRef: "std.integer"},
				{ID: "s.line.qty", Symbol: "qty", TypeRef: "s.qty"},
				{ID: "s.line.on", Symbol: "on", TypeRef: "std.date"},
				{ID: "s.line.note", Symbol: "note", TypeRef: "std.text"},
			}, CandidateKeys: []semantic.CandidateKey{{ID: "s.line.k", Symbol: "k", Attributes: []core.CanonicalID{"s.line.no"}}}},
			{ID: "s.free", Symbol: "Free", Attributes: []semantic.Attribute{{ID: "s.free.a", Symbol: "a", TypeRef: "std.text"}}},
		}}}}
}

func check(t *testing.T, dataID, tsv string) Report {
	t.Helper()
	r, err := Check(world(), dataID, strings.NewReader(tsv))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func codes(r Report) []string {
	var out []string
	for _, f := range r.Findings {
		out = append(out, f.Code)
	}
	return out
}

func TestCleanRowsAreShapeOKButStillListWhatIsUnchecked(t *testing.T) {
	r := check(t, "s.line", "no\tqty\ton\tnote\n1\t2.50\t2026-10-18\thello\n2\t3\t2026-10-19\t\n")
	if !r.ShapeOK || r.Rows != 2 || len(r.Findings) != 0 {
		t.Fatalf("report = %+v", r)
	}
	if len(r.Unchecked) < 3 {
		t.Fatalf("unchecked = %v", r.Unchecked)
	}
}

func TestHeaderProblems(t *testing.T) {
	r := check(t, "s.line", "no\tno\tbogus\n1\t1\tx\n")
	got := strings.Join(codes(r), ",")
	for _, want := range []string{"duplicate-column", "unknown-column"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s in %s", want, got)
		}
	}
	r = check(t, "s.line", "qty\n1\n")
	if !strings.Contains(strings.Join(codes(r), ","), "missing-key-column") {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestTypeKeyAndShapeFindings(t *testing.T) {
	r := check(t, "s.line", "no\tqty\ton\n1\tabc\t2026-13-40\n\t1\t2026-10-18\n1\t1\t2026-10-18\n1\t1\t2026-10-18\nextra\n")
	want := map[string]bool{"type-mismatch": true, "missing-key-value": true, "duplicate-key-of-row-2": true, "field-count-mismatch": true}
	got := map[string]bool{}
	for _, c := range codes(r) {
		got[c] = true
	}
	for c := range want {
		if !got[c] {
			t.Fatalf("missing %s in %+v", c, r.Findings)
		}
	}
	if r.ShapeOK {
		t.Fatal("report with findings is ShapeOK")
	}
}

func TestAliasChainAndExponentAreChecked(t *testing.T) {
	r := check(t, "s.line", "no\tqty\n1\t1e3\n2\t2.5\n")
	if len(r.Findings) != 1 || r.Findings[0].Row != 2 || r.Findings[0].Column != "qty" {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestFindingsNeverCarryTheValue(t *testing.T) {
	secret := "SECRET-VALUE-123"
	r := check(t, "s.line", "no\tqty\n"+secret+"\t"+secret+"\n")
	if len(r.Findings) == 0 {
		t.Fatal("expected findings")
	}
	for _, f := range r.Findings {
		if strings.Contains(f.Code+f.Column, secret) {
			t.Fatalf("finding leaks a value: %+v", f)
		}
	}
}

func TestNoCandidateKeyIsStatedNotGuessed(t *testing.T) {
	r := check(t, "s.free", "a\nx\nx\n")
	if !r.ShapeOK || len(r.Key) != 0 {
		t.Fatalf("report = %+v", r)
	}
	if !strings.Contains(strings.Join(r.Unchecked, ";"), "no-candidate-key") {
		t.Fatalf("unchecked = %v", r.Unchecked)
	}
}

func TestHeaderMayUseAttributeIDAndBOM(t *testing.T) {
	r := check(t, "s.line", "\ufeffs.line.no\tnote\n1\tx\n")
	if !r.ShapeOK {
		t.Fatalf("findings = %+v", r.Findings)
	}
}

func TestUnknownDataAndEmptyInputAreErrors(t *testing.T) {
	if _, err := Check(world(), "s.none", strings.NewReader("a\n")); err == nil {
		t.Fatal("unknown Data accepted")
	}
	if _, err := Check(world(), "s.line", strings.NewReader("")); err == nil {
		t.Fatal("empty input accepted")
	}
}
