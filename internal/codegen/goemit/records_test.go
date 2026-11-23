package goemit_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"tei/internal/codegen/goemit"
	"tei/internal/codegen/goemit/goemittest"
	"tei/internal/codegen/record"
	"tei/internal/semantic"
)

// textRange finds a function by reading lines, without go/parser: from the
// "func name(" line up through the "//" lines above it, and down to the
// first line that is a lone "}".
func textRange(t *testing.T, content []byte, name string) [2]int {
	t.Helper()
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "func "+name+"(") {
			continue
		}
		first := i
		for first > 0 && strings.HasPrefix(lines[first-1], "//") {
			first--
		}
		for last := i; last < len(lines); last++ {
			if lines[last] == "}" {
				return [2]int{first + 1, last + 1}
			}
		}
	}
	t.Fatalf("no function %s in\n%s", name, content)
	return [2]int{}
}

func TestTraceLinesCoverEachFunctionWithItsDocComment(t *testing.T) {
	out := goemit.Emit(goemit.Input{PackageName: "rules", Rules: []goemit.Rule{
		quantityRule,
		positiveRule("sales.order_line.discount_positive", "sales.order_line.discount"),
	}})
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	lines, err := out.TraceLines("goemit")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, line := range lines {
		if err := line.Validate(); err != nil {
			t.Fatal(err)
		}
		if want := textRange(t, out.File(line.File), line.Symbol); line.Lines != want {
			t.Errorf("%s: lines %v, the text gives %v", line.Symbol, line.Lines, want)
		}
		got = append(got, fmt.Sprintf("%s %s %s %s %s", line.Element, line.File, line.Symbol, line.Role, line.Provider))
	}
	// Order follows the Trace (by element ID), decision before check.
	want := []string{
		"sales.order_line.discount_positive discount_is_positive.go DiscountIsPositive decision goemit",
		"sales.order_line.discount_positive discount_is_positive.go CheckDiscountIsPositive check goemit",
		"sales.order_line.quantity_positive quantity_is_positive.go QuantityIsPositive decision goemit",
		"sales.order_line.quantity_positive quantity_is_positive.go CheckQuantityIsPositive check goemit",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("trace lines\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// The emitted layout today: header, package, import, then the decision
	// (3 comment lines, 3 code lines) and the check (3 + 9).
	if lines[2].Lines != [2]int{7, 12} || lines[3].Lines != [2]int{14, 25} {
		t.Fatalf("quantity ranges %v %v", lines[2].Lines, lines[3].Lines)
	}
	again, err := out.TraceLines("goemit")
	if err != nil || !reflect.DeepEqual(again, lines) {
		t.Fatal("trace lines differ between calls")
	}
}

// The ranges are read from the content, so they follow the file when its
// comments change length.
func TestTraceLinesFollowTheFileContent(t *testing.T) {
	out := goemit.Emit(goemit.Input{PackageName: "rules", Rules: []goemit.Rule{quantityRule}})
	rewritten, err := goemittest.RewriteComments(out, func(string, string) string { return "x" })
	if err != nil {
		t.Fatal(err)
	}
	// Drop the doc comment of the decision function: its range then starts
	// at the func line.
	content := string(rewritten.File("quantity_is_positive.go"))
	content = strings.Replace(content, "// x\n// x\n// x\nfunc QuantityIsPositive(", "func QuantityIsPositive(", 1)
	for i := range rewritten.Files {
		if rewritten.Files[i].Path == "quantity_is_positive.go" {
			rewritten.Files[i].Content = []byte(content)
		}
	}
	lines, err := rewritten.TraceLines("")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		if want := textRange(t, []byte(content), line.Symbol); line.Lines != want || line.Provider != "" {
			t.Errorf("%s: lines %v provider %q, want %v", line.Symbol, line.Lines, line.Provider, want)
		}
	}
	if lines[0].Lines[0] != lines[0].Lines[1]-2 {
		t.Fatalf("a decision without doc comment still starts above its func line: %v", lines[0].Lines)
	}
}

func TestTraceLinesRefuseATraceTheFilesDoNotBackUp(t *testing.T) {
	out := goemit.Emit(goemit.Input{PackageName: "rules", Rules: []goemit.Rule{quantityRule}})
	missingFunc := out
	missingFunc.Trace = []goemit.Trace{out.Trace[0]}
	missingFunc.Trace[0].Check = "CheckSomethingElse"
	if _, err := missingFunc.TraceLines(""); err == nil || !strings.Contains(err.Error(), "CheckSomethingElse") {
		t.Fatalf("undeclared function: %v", err)
	}
	missingFile := out
	missingFile.Trace = []goemit.Trace{out.Trace[0]}
	missingFile.Trace[0].File = "other.go"
	if _, err := missingFile.TraceLines(""); err == nil || !strings.Contains(err.Error(), "other.go") {
		t.Fatalf("missing file: %v", err)
	}
	broken := out
	broken.Files = append([]goemit.File(nil), out.Files...)
	for i := range broken.Files {
		if broken.Files[i].Path == out.Trace[0].File {
			broken.Files[i].Content = []byte("package rules\nfunc (")
		}
	}
	if _, err := broken.TraceLines(""); err == nil {
		t.Fatal("a file that does not parse gave trace lines")
	}
}

func TestManifestListsEveryFileWithItsDigest(t *testing.T) {
	out := goemit.Emit(goemit.Input{PackageName: "rules", Rules: []goemit.Rule{
		quantityRule,
		positiveRule("sales.customer.customer_id_positive", "sales.customer.customer_id"),
	}})
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	m := out.Manifest(goemit.Writer)
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	if m.Schema != record.ManifestSchema || m.Input != "" || len(m.Files) != len(out.Files) {
		t.Fatalf("manifest %+v", m)
	}
	var paths []string
	for i, file := range m.Files {
		paths = append(paths, file.Path)
		sum := sha256.Sum256(out.File(file.Path))
		if file.SHA256 != hex.EncodeToString(sum[:]) || file.Writer != "goemit" || file.Hook != "" {
			t.Errorf("%d: %+v", i, file)
		}
	}
	if got := strings.Join(paths, ","); got != "customer_id_is_positive.go,decimal.go,quantity_is_positive.go,result.go" {
		t.Fatalf("manifest paths %s", got)
	}
	reordered := goemit.Emit(goemit.Input{PackageName: "rules", Rules: []goemit.Rule{
		positiveRule("sales.customer.customer_id_positive", "sales.customer.customer_id"),
		quantityRule,
	}})
	if !reflect.DeepEqual(reordered.Manifest(goemit.Writer), m) {
		t.Fatal("the order of the rules changed the manifest")
	}
}

// Every code names who fixes it, and every diagnostic is a valid record.
func TestDiagnosticsAreValidRecordsWithTheirFixer(t *testing.T) {
	kind := quantityRule
	kind.Validation.Kind = semantic.ValidationRelationUnique
	shape := quantityRule
	shape.Validation.Relation = "sales.order_line.items"
	text := quantityRule
	text.AttributeType = "std.text"
	cases := []struct {
		input       goemit.Input
		code, fixer string
	}{
		{goemit.Input{PackageName: "_"}, goemit.CodeInvalidPackageName, "developer"},
		{goemit.Input{PackageName: "main", Rules: []goemit.Rule{kind}}, goemit.CodeUnsupportedKind, "designer"},
		{goemit.Input{PackageName: "main", Rules: []goemit.Rule{shape}}, goemit.CodeInvalidRuleShape, "designer"},
		{goemit.Input{PackageName: "main", Rules: []goemit.Rule{text}}, goemit.CodeUnsupportedType, "designer"},
		{goemit.Input{PackageName: "main", Rules: []goemit.Rule{positiveRule("x.rule", "x.2nd")}}, goemit.CodeNameInvalid, "developer"},
		{goemit.Input{PackageName: "main", Rules: []goemit.Rule{positiveRule("x.rule", "x.type")}}, goemit.CodeNameReserved, "developer"},
		{goemit.Input{PackageName: "main", Rules: []goemit.Rule{
			positiveRule("a.r1", "a.quantity"), positiveRule("b.r2", "b.quantity"),
		}}, goemit.CodeNameCollision, "developer"},
	}
	for _, c := range cases {
		out := goemit.Emit(c.input)
		if len(out.Diagnostics) != 1 {
			t.Fatalf("%s: %+v", c.code, out.Diagnostics)
		}
		d := out.Diagnostics[0]
		if d.Code != c.code || d.Fixer != c.fixer || d.Severity != "error" {
			t.Errorf("got %s fixer %q severity %q, want %s fixer %q", d.Code, d.Fixer, d.Severity, c.code, c.fixer)
		}
		if err := d.Validate(); err != nil {
			t.Errorf("%s: %v", c.code, err)
		}
	}
}
