package goemit_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"math/big"
	"sort"
	"strings"
	"testing"

	"tei/internal/codegen/goemit"
	"tei/internal/codegen/goemit/goemittest"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

func positiveRule(id, attribute core.CanonicalID) goemit.Rule {
	return goemit.Rule{
		Validation: semantic.Validation{
			ID: id, Kind: semantic.ValidationPositive, Attribute: attribute,
		},
		AttributeType: "std.decimal",
	}
}

var quantityRule = positiveRule("sales.order_line.quantity_positive", "sales.order_line.quantity")

// The standalone module is written by hand here, not by goemit: the test
// states what a caller outside TEI writes, and the expected answers are
// typed in, not taken from the generator.
const standaloneTest = `package generated

import "testing"

func TestCheckQuantityIsPositive(t *testing.T) {
	for _, c := range []struct {
		text string
		want Result
	}{
		{"2.5", Satisfied},
		{"0.01", Satisfied},
		{"0", NotSatisfied},
		{"-0", NotSatisfied},
		{"-1", NotSatisfied},
	} {
		value, err := ParseDecimal(c.text)
		if err != nil {
			t.Fatalf("%s: %v", c.text, err)
		}
		if got := CheckQuantityIsPositive(value); got != c.want {
			t.Fatalf("%s: got %s, want %s", c.text, got, c.want)
		}
		if got := QuantityIsPositive(value); got != (c.want == Satisfied) {
			t.Fatalf("%s: decision %t", c.text, got)
		}
	}
	if got := CheckQuantityIsPositive(nil); got != Undecided {
		t.Fatalf("missing value: got %s", got)
	}
	if got := CheckCustomerIDIsPositive(nil); got != Undecided {
		t.Fatalf("missing customer ID: got %s", got)
	}
}

func TestParseDecimalGrammar(t *testing.T) {
	for _, text := range []string{"0", "-0", "0.0", "-0.5", "300", "1.25"} {
		if _, err := ParseDecimal(text); err != nil {
			t.Errorf("%q refused: %v", text, err)
		}
	}
	for _, text := range []string{"", "-", "--1", "+1", ".5", "5.", "1.2.3", "007", "-007", "00.5", "1e3", "1/3", "1_000", "1,000", " 1", "1 "} {
		if _, err := ParseDecimal(text); err == nil {
			t.Errorf("%q accepted", text)
		}
	}
}
`

func TestEmittedPackageBuildsAndTestsInAStandaloneModule(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the emitted package with the go command")
	}
	out := goemit.Emit(goemit.Input{PackageName: "generated", Rules: []goemit.Rule{
		quantityRule,
		positiveRule("sales.customer.customer_id_positive", "sales.customer.customer_id"),
	}})
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	files := goemittest.Files(out)
	imports, err := goemittest.TEIImports(files)
	if err != nil || len(imports) != 0 {
		t.Fatalf("emitted Go imports TEI packages: %v %v", imports, err)
	}
	files["generated_test.go"] = []byte(standaloneTest)
	dir := goemittest.WriteModule(t, files)
	goemittest.RunGo(t, dir, "build", "./...")
	goemittest.RunGo(t, dir, "vet", "./...")
	goemittest.RunGo(t, dir, "test", "-count=1", "./...")
	deps := string(goemittest.RunGo(t, dir, "list", "-deps", "./..."))
	for _, dep := range strings.Fields(deps) {
		if dep == "tei" || strings.HasPrefix(dep, "tei/") {
			t.Fatalf("the standalone build depends on %s", dep)
		}
	}
}

// statefulNumber is a reader for Decide that reports one fixed state.
type statefulNumber struct {
	reading validationeval.NumberReading
}

func (s statefulNumber) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return s.reading, nil
}

func (s statefulNumber) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, fmt.Errorf("unexpected relation read")
}

// parityMain calls the generated check function for each case: with the
// parsed value when it is present, and with nil when it is null or was not
// fetched. It prints "<name>:<result>".
const parityMain = `package main

import "fmt"

func main() {
	for _, c := range []struct{ name, state, text string }{
%s	} {
		result := CheckQuantityIsPositive(nil)
		if c.state == "present" {
			value, err := ParseDecimal(c.text)
			if err != nil {
				panic(err)
			}
			result = CheckQuantityIsPositive(value)
		}
		fmt.Printf("%%s:%%s\n", c.name, result)
	}
}
`

func TestGeneratedCheckAgreesWithDecideOnAFixedTable(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the emitted package with the go command")
	}
	// The expected answers are written by hand. Neither the evaluator nor the
	// generated code is used to compute them.
	cases := []struct {
		name  string
		state validationeval.ValueState
		text  string
		want  string
	}{
		{"two-and-a-half", validationeval.ValuePresent, "2.5", "satisfied"},
		{"one-hundredth", validationeval.ValuePresent, "0.01", "satisfied"},
		{"three-hundred", validationeval.ValuePresent, "300", "satisfied"},
		{"zero", validationeval.ValuePresent, "0", "not-satisfied"},
		{"zero-point-zero", validationeval.ValuePresent, "0.0", "not-satisfied"},
		{"negative-zero", validationeval.ValuePresent, "-0", "not-satisfied"},
		{"minus-one-hundredth", validationeval.ValuePresent, "-0.01", "not-satisfied"},
		{"minus-one", validationeval.ValuePresent, "-1", "not-satisfied"},
		{"null", validationeval.ValueNull, "", "undecided"},
		{"not-fetched", validationeval.ValueNotFetched, "", "undecided"},
	}

	var table strings.Builder
	var want strings.Builder
	for _, c := range cases {
		fmt.Fprintf(&table, "\t\t{%q, %q, %q},\n", c.name, c.state, c.text)
		fmt.Fprintf(&want, "%s:%s\n", c.name, c.want)

		reading := validationeval.NumberReading{State: c.state}
		if c.state == validationeval.ValuePresent {
			value, err := semantic.ParseOperatorCaseValue(c.text)
			if err != nil {
				t.Fatal(err)
			}
			reading.Value = value
		}
		decision, err := validationeval.Decide(quantityRule.Validation, statefulNumber{reading}, validationeval.Operators{})
		if err != nil {
			t.Fatalf("%s: Decide: %v", c.name, err)
		}
		got := string(decision.Result)
		if decision.Undecided != nil {
			got = "undecided"
		}
		if got != c.want {
			t.Fatalf("%s: Decide answered %s, want %s", c.name, got, c.want)
		}
	}

	out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{quantityRule}})
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	files := goemittest.Files(out)
	files["main.go"] = []byte(fmt.Sprintf(parityMain, table.String()))
	dir := goemittest.WriteModule(t, files)
	if got := string(goemittest.RunGo(t, dir, "run", ".")); got != want.String() {
		t.Fatalf("generated check answered\n%s\nwant\n%s", got, want.String())
	}
}

func TestEmitIsDeterministicAndFormatted(t *testing.T) {
	rules := []goemit.Rule{
		positiveRule("sales.order_line.quantity_positive", "sales.order_line.quantity"),
		positiveRule("sales.order_line.discount_positive", "sales.order_line.discount"),
		positiveRule("sales.customer.customer_id_positive", "sales.customer.customer_id"),
	}
	reversed := []goemit.Rule{rules[2], rules[1], rules[0]}
	first := goemit.Emit(goemit.Input{PackageName: "rules", Rules: rules})
	second := goemit.Emit(goemit.Input{PackageName: "rules", Rules: rules})
	fromReversed := goemit.Emit(goemit.Input{PackageName: "rules", Rules: reversed})
	if err := first.Err(); err != nil {
		t.Fatal(err)
	}
	for _, other := range []goemit.Output{second, fromReversed} {
		if len(other.Files) != len(first.Files) {
			t.Fatal("a repeated or reordered input changed the emitted files")
		}
		for i := range first.Files {
			if first.Files[i].Path != other.Files[i].Path ||
				!bytes.Equal(first.Files[i].Content, other.Files[i].Content) {
				t.Fatalf("%s differs between runs", first.Files[i].Path)
			}
		}
		if fmt.Sprint(first.Trace) != fmt.Sprint(other.Trace) {
			t.Fatalf("trace differs between runs:\n%v\n%v", first.Trace, other.Trace)
		}
	}

	var paths []string
	for _, file := range first.Files {
		paths = append(paths, file.Path)
		once, err := format.Source(file.Content)
		if err != nil {
			t.Fatal(err)
		}
		twice, err := format.Source(once)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(once, file.Content) || !bytes.Equal(twice, once) {
			t.Fatalf("%s is not stable under gofmt", file.Path)
		}
		// The header names the element, never a time, a path or a counter.
		header := strings.SplitN(string(file.Content), "\n", 2)[0]
		if !strings.HasPrefix(header, "// Code generated by TEI") || !strings.HasSuffix(header, "; DO NOT EDIT.") {
			t.Fatalf("%s header %q", file.Path, header)
		}
	}
	wantPaths := []string{"customer_id_is_positive.go", "decimal.go", "discount_is_positive.go", "quantity_is_positive.go", "result.go"}
	if !sort.StringsAreSorted(paths) || strings.Join(paths, ",") != strings.Join(wantPaths, ",") {
		t.Fatalf("emitted paths %v, want %v", paths, wantPaths)
	}
	var elements []string
	for _, line := range first.Trace {
		elements = append(elements, string(line.Element))
	}
	if !sort.StringsAreSorted(elements) || len(elements) != 3 {
		t.Fatalf("trace is not ordered by element ID: %v", elements)
	}
	if got := first.Trace[2].String(); got != "sales.order_line.quantity_positive -> quantity_is_positive.go: QuantityIsPositive, CheckQuantityIsPositive (value of sales.order_line.quantity)" {
		t.Fatalf("trace line %q", got)
	}
	if got := string(first.File("quantity_is_positive.go")); !strings.HasPrefix(got,
		"// Code generated by TEI from sales.order_line.quantity_positive; DO NOT EDIT.\n\npackage rules\n") {
		t.Fatalf("rule file header:\n%s", got)
	}
}

func TestEmittedRuleFileHasTheTwoLayers(t *testing.T) {
	rule := quantityRule
	rule.DisplayName = "数量は正の数"
	out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{rule}})
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	src := out.File("quantity_is_positive.go")
	for _, want := range []string{
		"import \"math/big\"\n",
		"func QuantityIsPositive(quantity *big.Rat) bool {\n\treturn quantity.Sign() > 0\n}\n",
		"func CheckQuantityIsPositive(quantity *big.Rat) Result {\n\tif quantity == nil {\n\t\treturn Undecided\n\t}\n\tif QuantityIsPositive(quantity) {\n\t\treturn Satisfied\n\t}\n\treturn NotSatisfied\n}\n",
		"// QuantityIsPositive は、規則「数量は正の数」（sales.order_line.quantity_positive）の判定です。\n",
		"// quantity には属性 sales.order_line.quantity の値を渡します。\n",
	} {
		if !bytes.Contains(src, []byte(want)) {
			t.Fatalf("rule file lacks %q:\n%s", want, src)
		}
	}
	// Identifiers come from the attribute's words only: no element ID, hash
	// or sequence number reaches them.
	allowed := map[string]bool{
		"main": true, "QuantityIsPositive": true, "CheckQuantityIsPositive": true,
		"quantity": true, "big": true, "Rat": true, "Sign": true, "bool": true,
		"Result": true, "nil": true, "Undecided": true, "Satisfied": true, "NotSatisfied": true,
	}
	file, err := parser.ParseFile(token.NewFileSet(), "rule.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && !allowed[id.Name] {
			t.Errorf("unexpected identifier %s", id.Name)
		}
		return true
	})
	bindings, err := goemittest.Bindings(src)
	if err != nil || len(bindings) != 1 ||
		bindings[0] != (goemittest.Binding{Check: "CheckQuantityIsPositive", Parameter: "quantity", Attribute: "sales.order_line.quantity"}) {
		t.Fatalf("binding sentence: %+v %v", bindings, err)
	}
}

func TestGeneratedResultValuesMatchTheEvaluator(t *testing.T) {
	out := goemit.Emit(goemit.Input{PackageName: "main"})
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	src := string(out.File("result.go"))
	for _, want := range []string{
		`Satisfied Result = "` + string(validationeval.ResultSatisfied) + `"`,
		`NotSatisfied Result = "` + string(validationeval.ResultNotSatisfied) + `"`,
		`Undecided Result = "undecided"`,
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("result.go lacks %s:\n%s", want, src)
		}
	}
}

func diagnosticCodes(out goemit.Output) []string {
	var codes []string
	for _, diagnostic := range out.Diagnostics {
		if diagnostic.Severity != goemit.SeverityError {
			return []string{"unexpected severity " + diagnostic.Severity}
		}
		codes = append(codes, diagnostic.Code)
	}
	return codes
}

func TestRejectedInputGivesItsDiagnosticAndNoFiles(t *testing.T) {
	relation := quantityRule
	relation.Validation.Relation = "sales.order_line.items"
	kind := quantityRule
	kind.Validation.Kind = semantic.ValidationRelationUnique
	text := quantityRule
	text.AttributeType = "std.text"
	cases := []struct {
		name    string
		input   goemit.Input
		code    string
		message string
	}{
		{"package", goemit.Input{PackageName: "not-a-go-identifier", Rules: []goemit.Rule{quantityRule}},
			"INVALID_PACKAGE_NAME", "invalid Go package name"},
		{"kind", goemit.Input{PackageName: "main", Rules: []goemit.Rule{kind}},
			"UNSUPPORTED_KIND", "unsupported generated kind"},
		{"relation", goemit.Input{PackageName: "main", Rules: []goemit.Rule{relation}},
			"INVALID_RULE_SHAPE", "positive validation must have an ID, one Attribute, and no Relation"},
		{"type", goemit.Input{PackageName: "main", Rules: []goemit.Rule{text}},
			"UNSUPPORTED_TYPE", "requires resolved std.decimal"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := goemit.Emit(c.input)
			if got := diagnosticCodes(out); len(got) != 1 || got[0] != c.code {
				t.Fatalf("codes %v, want %s", got, c.code)
			}
			if !strings.Contains(out.Err().Error(), c.message) || len(out.Files) != 0 || len(out.Trace) != 0 {
				t.Fatalf("rejection emitted files or lost its message: %v %d", out.Err(), len(out.Files))
			}
		})
	}
}

func TestNamesFollowTheAttributeWords(t *testing.T) {
	cases := []struct {
		attribute                        core.CanonicalID
		decision, check, parameter, path string
	}{
		{"sales.order_line.quantity", "QuantityIsPositive", "CheckQuantityIsPositive", "quantity", "quantity_is_positive.go"},
		{"sales.order_line.second_quantity", "SecondQuantityIsPositive", "CheckSecondQuantityIsPositive", "secondQuantity", "second_quantity_is_positive.go"},
		{"sales.order_line.secondQuantity", "SecondQuantityIsPositive", "CheckSecondQuantityIsPositive", "secondQuantity", "second_quantity_is_positive.go"},
		{"sales.customer.customer_id", "CustomerIDIsPositive", "CheckCustomerIDIsPositive", "customerID", "customer_id_is_positive.go"},
		{"web.call.api_url_count", "APIURLCountIsPositive", "CheckAPIURLCountIsPositive", "apiURLCount", "api_url_count_is_positive.go"},
		{"web.call.HTTPRetryLimit", "HTTPRetryLimitIsPositive", "CheckHTTPRetryLimitIsPositive", "httpRetryLimit", "http_retry_limit_is_positive.go"},
		{"sales.order_line.line2_amount", "Line2AmountIsPositive", "CheckLine2AmountIsPositive", "line2Amount", "line2_amount_is_positive.go"},
	}
	for _, c := range cases {
		out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{positiveRule("x.rule", c.attribute)}})
		if err := out.Err(); err != nil {
			t.Fatalf("%s: %v", c.attribute, err)
		}
		got := out.Trace[0]
		if got.Decision != c.decision || got.Check != c.check || got.File != c.path {
			t.Fatalf("%s: got %+v", c.attribute, got)
		}
		if !bytes.Contains(out.File(c.path), []byte("func "+c.decision+"("+c.parameter+" *big.Rat) bool")) {
			t.Fatalf("%s: parameter is not %s:\n%s", c.attribute, c.parameter, out.File(c.path))
		}
	}
}

func TestTwoRulesWithOneNameGiveNameCollision(t *testing.T) {
	out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{
		positiveRule("sales.order_line.quantity_positive", "sales.order_line.quantity"),
		positiveRule("inventory.item.quantity_positive", "inventory.item.quantity"),
	}})
	if got := diagnosticCodes(out); len(got) != 1 || got[0] != "NAME_COLLISION" {
		t.Fatalf("codes %v", got)
	}
	message := out.Diagnostics[0].Message
	if !strings.Contains(message, "inventory.item.quantity_positive") ||
		!strings.Contains(message, "sales.order_line.quantity_positive") ||
		!strings.Contains(message, "QuantityIsPositive") || len(out.Files) != 0 {
		t.Fatalf("collision does not name both elements or still emitted: %s", message)
	}
	// No suffix is invented to separate them.
	if strings.Contains(message, "QuantityIsPositive2") {
		t.Fatal("a numbered name was proposed")
	}
}

func TestReservedOrInvalidNamesAreRefused(t *testing.T) {
	cases := []struct {
		attribute core.CanonicalID
		code      string
	}{
		{"sales.order_line.type", "NAME_RESERVED"},
		{"sales.order_line.len", "NAME_RESERVED"},
		{"sales.order_line.big", "NAME_RESERVED"},
		{"sales.order_line.2nd_quantity", "NAME_INVALID"},
		{"sales.order_line.数量", "NAME_INVALID"},
	}
	for _, c := range cases {
		out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{positiveRule("x.rule", c.attribute)}})
		if got := diagnosticCodes(out); len(got) != 1 || got[0] != c.code || len(out.Files) != 0 {
			t.Fatalf("%s: codes %v, want %s", c.attribute, got, c.code)
		}
		if out.Diagnostics[0].Element != "x.rule" || !strings.Contains(out.Diagnostics[0].Message, string(c.attribute)) {
			t.Fatalf("%s: diagnostic does not name the rule and attribute: %+v", c.attribute, out.Diagnostics[0])
		}
	}
}

// sameNumber keeps the comparison on values, not on how they print.
func sameNumber(a, b *big.Rat) bool { return a.Cmp(b) == 0 }
