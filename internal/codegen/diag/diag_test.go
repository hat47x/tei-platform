package diag_test

import (
	"encoding/json"
	"testing"

	"tei/internal/codegen/diag"
)

func valid() diag.Record {
	return diag.Record{Code: "NAME_COLLISION", Severity: diag.SeverityError, Message: "two rules give one name"}
}

func TestValidateAcceptsCompleteAndMinimalRecords(t *testing.T) {
	full := diag.Record{
		Code: "goemit/NAME_COLLISION", Severity: diag.SeverityWarning,
		File: "rules/quantity_is_positive.go", Range: &diag.Range{Start: 3, End: 3},
		Line: 1, Column: 4, Element: "sales.order_line.quantity_positive",
		Message: "m", Fixer: diag.FixerDeveloper, Provider: "goemit",
		Related: []diag.Related{{File: "a.tei", Range: &diag.Range{Start: 0, End: 9}, Role: "conflict"}},
	}
	for _, r := range []diag.Record{valid(), full} {
		if err := r.Validate(); err != nil {
			t.Fatalf("%+v: %v", r, err)
		}
	}
	for _, code := range []string{"A", "PARSE_ERROR", "X9_", "tei.codegen/E", "a.b-c.d/CODE_2"} {
		r := valid()
		r.Code = code
		if err := r.Validate(); err != nil {
			t.Errorf("code %q refused: %v", code, err)
		}
	}
	for _, severity := range []string{diag.SeverityError, diag.SeverityWarning, diag.SeverityInfo} {
		r := valid()
		r.Severity = severity
		if err := r.Validate(); err != nil {
			t.Errorf("severity %q refused: %v", severity, err)
		}
	}
	for _, fixer := range []string{"", diag.FixerDesigner, diag.FixerDeveloper, diag.FixerProvider} {
		r := valid()
		r.Fixer = fixer
		if err := r.Validate(); err != nil {
			t.Errorf("fixer %q refused: %v", fixer, err)
		}
	}
}

func TestValidateRefusesBrokenRecords(t *testing.T) {
	cases := map[string]func(*diag.Record){
		"empty code":           func(r *diag.Record) { r.Code = "" },
		"lower-case code":      func(r *diag.Record) { r.Code = "name_collision" },
		"code starts in digit": func(r *diag.Record) { r.Code = "9LIVES" },
		"upper-case namespace": func(r *diag.Record) { r.Code = "GoEmit/NAME" },
		"empty namespace":      func(r *diag.Record) { r.Code = "/NAME" },
		"namespace hyphen":     func(r *diag.Record) { r.Code = "go-emit/NAME" },
		"no severity":          func(r *diag.Record) { r.Severity = "" },
		"unknown severity":     func(r *diag.Record) { r.Severity = "fatal" },
		"capital severity":     func(r *diag.Record) { r.Severity = "Error" },
		"unknown fixer":        func(r *diag.Record) { r.Fixer = "operator" },
		"reversed range":       func(r *diag.Record) { r.Range = &diag.Range{Start: 5, End: 4} },
		"negative range":       func(r *diag.Record) { r.Range = &diag.Range{Start: -1, End: 4} },
		"negative line":        func(r *diag.Record) { r.Line = -1 },
		"empty message":        func(r *diag.Record) { r.Message = "" },
		"related without role": func(r *diag.Record) { r.Related = []diag.Related{{File: "a"}} },
		"related reversed range": func(r *diag.Record) {
			r.Related = []diag.Related{{Role: "x", Range: &diag.Range{Start: 2, End: 1}}}
		},
	}
	for name, edit := range cases {
		r := valid()
		edit(&r)
		if err := r.Validate(); err == nil {
			t.Errorf("%s: %+v accepted", name, r)
		}
	}
}

// The JSON names are part of the record's contract: lower case as named, and
// optional fields left out when empty.
func TestJSONFieldNames(t *testing.T) {
	minimal, err := json.Marshal(valid())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(minimal), `{"code":"NAME_COLLISION","severity":"error","message":"two rules give one name"}`; got != want {
		t.Fatalf("minimal record\n got %s\nwant %s", got, want)
	}
	full, err := json.Marshal(diag.Record{
		Code: "C", Severity: "info", File: "f", Range: &diag.Range{Start: 0, End: 1},
		Line: 1, Column: 2, Element: "e", Message: "m",
		Related: []diag.Related{{File: "g", Range: &diag.Range{Start: 1, End: 2}, Element: "h", Role: "r"}},
		Fixer:   "designer", Provider: "p",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"code":"C","severity":"info","file":"f","range":{"start":0,"end":1},"line":1,"column":2,"element":"e","message":"m",` +
		`"related":[{"file":"g","range":{"start":1,"end":2},"element":"h","role":"r"}],"fixer":"designer","provider":"p"}`
	if string(full) != want {
		t.Fatalf("full record\n got %s\nwant %s", full, want)
	}
}
