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
	} {
		// Only code is matched here, never comment wording. Comment text
		// is free to change; the binding is checked through the Trace
		// below (see TestBindingAndBehaviourDoNotDependOnCommentText).
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
	// The binding is structured data in the Trace, checked against the
	// declared function, not read from the comment.
	bindings, err := goemittest.Bindings(out)
	if err != nil || len(bindings) != 1 ||
		bindings[0] != (goemittest.Binding{Check: "CheckQuantityIsPositive", Parameter: "quantity", Attribute: "sales.order_line.quantity"}) {
		t.Fatalf("binding: %+v %v", bindings, err)
	}
}

// Comment text is prose for people, never a contract. Every comment of every
// emitted file is replaced with text that names no attribute and no rule, and
// the bindings, the observed syntax and the run of the generated code stay the
// same. A test that reads its binding from the doc comment fails here.
func TestBindingAndBehaviourDoNotDependOnCommentText(t *testing.T) {
	out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{
		quantityRule,
		positiveRule("sales.order_line.discount_positive", "sales.order_line.discount"),
	}})
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	rewritten, err := goemittest.RewriteComments(out, func(path, _ string) string {
		return "unrelated text in " + path
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range rewritten.Files {
		for _, gone := range []string{"sales.order_line.", "DO NOT EDIT"} {
			if bytes.Contains(file.Content, []byte(gone)) {
				t.Fatalf("%s still holds %q after the comment rewrite:\n%s", file.Path, gone, file.Content)
			}
		}
		if bytes.Equal(file.Content, out.File(file.Path)) {
			t.Fatalf("%s was not rewritten", file.Path)
		}
	}
	before, errBefore := goemittest.Bindings(out)
	after, errAfter := goemittest.Bindings(rewritten)
	if errBefore != nil || errAfter != nil || len(after) != 2 || fmt.Sprint(before) != fmt.Sprint(after) {
		t.Fatalf("binding changed with comment text: %+v %v / %+v %v", before, errBefore, after, errAfter)
	}
	// The code itself is the same: only comments were touched.
	stripped := func(o goemit.Output) map[string]string {
		code := map[string]string{}
		for _, file := range o.Files {
			fset := token.NewFileSet()
			parsed, err := parser.ParseFile(fset, file.Path, file.Content, 0)
			if err != nil {
				t.Fatal(err)
			}
			var b bytes.Buffer
			if err := format.Node(&b, fset, parsed); err != nil {
				t.Fatal(err)
			}
			code[file.Path] = b.String()
		}
		return code
	}
	if fmt.Sprint(stripped(out)) != fmt.Sprint(stripped(rewritten)) {
		t.Fatal("the comment rewrite changed code")
	}
	// The rewritten files still build and give the same answers.
	single := func(o goemit.Output) goemit.Output {
		o.Trace = o.Trace[:1]
		return o
	}
	cases := []goemittest.Case{
		{Name: "positive", Row: map[string]goemittest.Cell{"sales.order_line.discount": {Text: "2"}}},
		{Name: "zero", Row: map[string]goemittest.Cell{"sales.order_line.discount": {Text: "0"}}},
		{Name: "missing", Row: map[string]goemittest.Cell{}},
	}
	if got, want := goemittest.Run(t, single(rewritten), cases), goemittest.Run(t, single(out), cases); got != want {
		t.Fatalf("comment rewrite changed the run:\n%s\nwant:\n%s", got, want)
	}

	// A Trace naming a function the file does not declare is refused.
	broken := out
	broken.Trace = append([]goemit.Trace(nil), out.Trace...)
	broken.Trace[0].Check = "CheckSomethingElse"
	if _, err := goemittest.Bindings(broken); err == nil {
		t.Fatal("a trace naming an undeclared check function was accepted")
	}
	// So is a Trace whose check function does not pass its value to the
	// decision function the Trace names.
	broken.Trace[0] = out.Trace[0]
	broken.Trace[0].Decision = out.Trace[1].Decision
	if _, err := goemittest.Bindings(broken); err == nil {
		t.Fatal("a trace naming another rule's decision function was accepted")
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
		{"blank package", goemit.Input{PackageName: "_", Rules: []goemit.Rule{quantityRule}},
			"INVALID_PACKAGE_NAME", "a package named _ cannot be built"},
		{"kind", goemit.Input{PackageName: "main", Rules: []goemit.Rule{kind}},
			"UNSUPPORTED_KIND", "unsupported generated kind"},
		{"relation", goemit.Input{PackageName: "main", Rules: []goemit.Rule{relation}},
			"INVALID_RULE_SHAPE", "positive validation must have an ID, one Attribute, and no Relation"},
		{"type", goemit.Input{PackageName: "main", Rules: []goemit.Rule{text}},
			"UNSUPPORTED_TYPE", "requires resolved std.decimal"},
	}
	// Text written into comments must not end the comment. Each of these
	// once made Emit panic in format.Source.
	for _, bad := range []struct {
		name string
		edit func(*goemit.Rule)
	}{
		{"id newline", func(r *goemit.Rule) { r.Validation.ID = "sales.order_line.quantity_positive\nfunc init() {}" }},
		{"id carriage return", func(r *goemit.Rule) { r.Validation.ID = "sales.order_line.quantity_positive\r" }},
		{"display name newline", func(r *goemit.Rule) { r.DisplayName = "数量は\n正の数" }},
		{"display name tab", func(r *goemit.Rule) { r.DisplayName = "数量は\t正の数" }},
		{"display name invalid UTF-8", func(r *goemit.Rule) { r.DisplayName = "\xff" }},
		{"display name line separator", func(r *goemit.Rule) { r.DisplayName = "数量は\u2028正の数" }},
		{"attribute newline", func(r *goemit.Rule) { r.Validation.Attribute = "sales.order\n_line.quantity" }},
	} {
		rule := quantityRule
		bad.edit(&rule)
		cases = append(cases, struct {
			name    string
			input   goemit.Input
			code    string
			message string
		}{bad.name, goemit.Input{PackageName: "main", Rules: []goemit.Rule{rule}},
			"INVALID_RULE_SHAPE", "cannot be written into a Go comment"})
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
		// An initialism followed by a plural 's' at the end of a word stays
		// one word, in camel and snake case alike.
		{"sales.order.userIDs", "UserIDsIsPositive", "CheckUserIDsIsPositive", "userIDs", "user_ids_is_positive.go"},
		{"sales.order.user_ids", "UserIDsIsPositive", "CheckUserIDsIsPositive", "userIDs", "user_ids_is_positive.go"},
		{"sales.order.userIDsCount", "UserIDsCountIsPositive", "CheckUserIDsCountIsPositive", "userIDsCount", "user_ids_count_is_positive.go"},
		{"web.call.apiURLCount", "APIURLCountIsPositive", "CheckAPIURLCountIsPositive", "apiURLCount", "api_url_count_is_positive.go"},
		// Documented odd spellings: iD splits at its case boundary, and
		// HTTPSUrl gives the HTTPS initialism followed by url.
		{"sales.order.iD", "IDIsPositive", "CheckIDIsPositive", "iD", "i_d_is_positive.go"},
		{"sales.order.iDs", "IDsIsPositive", "CheckIDsIsPositive", "iDs", "i_ds_is_positive.go"},
		// The plural rule covers the listed initialisms only: any other
		// capital run before a final 's' splits before its last capital.
		{"sales.order.userABCs", "UserAbCsIsPositive", "CheckUserAbCsIsPositive", "userAbCs", "user_ab_cs_is_positive.go"},
		{"sales.order.apiURLs", "APIURLsIsPositive", "CheckAPIURLsIsPositive", "apiURLs", "api_urls_is_positive.go"},
		{"web.call.HTTPSUrl", "HTTPSURLIsPositive", "CheckHTTPSURLIsPositive", "httpsURL", "https_url_is_positive.go"},
		// A lower-case run after "s" is not a plural: IDsum splits as before.
		{"sales.order.userIDsum", "UserIDsumIsPositive", "CheckUserIDsumIsPositive", "userIDsum", "user_i_dsum_is_positive.go"},
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

// A decision name of one rule can equal the check name of another:
// attribute check_quantity gives the decision CheckQuantityIsPositive, which
// is the check function of attribute quantity. Both orders must be refused.
func TestDecisionNameEqualToAnotherCheckNameGivesNameCollision(t *testing.T) {
	quantity := positiveRule("a.r1", "a.quantity")
	checkQuantity := positiveRule("a.r2", "a.check_quantity")
	for _, rules := range [][]goemit.Rule{{quantity, checkQuantity}, {checkQuantity, quantity}} {
		out := goemit.Emit(goemit.Input{PackageName: "rules", Rules: rules})
		if got := diagnosticCodes(out); len(got) != 1 || got[0] != "NAME_COLLISION" {
			t.Fatalf("codes %v", got)
		}
		message := out.Diagnostics[0].Message
		if !strings.Contains(message, "a.r1") || !strings.Contains(message, "a.r2") ||
			!strings.Contains(message, "CheckQuantityIsPositive") || len(out.Files) != 0 {
			t.Fatalf("collision does not name both elements and the name, or still emitted: %s", message)
		}
	}
}

// A file name of 200 bytes is emitted; one byte more gives NAME_INVALID.
func TestOverlongFileNameGivesNameInvalid(t *testing.T) {
	const suffix = "_is_positive.go"
	fits := core.CanonicalID("sales.order_line." + strings.Repeat("q", 200-len(suffix)))
	out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{positiveRule("x.rule", fits)}})
	if err := out.Err(); err != nil || len(out.Trace) != 1 || len(out.Trace[0].File) != 200 {
		t.Fatalf("a 200-byte file name was refused: %v %+v", err, out.Trace)
	}
	tooLong := fits + "q"
	out = goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{positiveRule("x.rule", tooLong)}})
	if got := diagnosticCodes(out); len(got) != 1 || got[0] != goemit.CodeNameInvalid || len(out.Files) != 0 {
		t.Fatalf("a 201-byte file name gave codes %v", got)
	}
	if !strings.Contains(out.Diagnostics[0].Message, "201 bytes") || out.Diagnostics[0].Element != "x.rule" {
		t.Fatalf("diagnostic does not name the length: %+v", out.Diagnostics[0])
	}
	// Underscores between words count too: the limit is on the file name,
	// not on the attribute.
	words := core.CanonicalID("sales.order_line." + strings.Repeat("ab_", 62) + "ab")
	out = goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{positiveRule("x.rule", words)}})
	if got := diagnosticCodes(out); len(got) != 1 || got[0] != goemit.CodeNameInvalid {
		t.Fatalf("a %d-byte file name from words gave codes %v", 62*3+2+len(suffix), got)
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

// The doc comments are free prose, so no test matches their wording. What a
// reader needs from them is still checked: the comment on each emitted
// function names the rule it comes from, the comment on the check function
// names the attribute whose value the caller passes, and a display name, when
// given, is shown. The facts are taken from the Trace, not from the comments.
func TestDocCommentsNameTheRuleTheAttributeAndTheDisplayName(t *testing.T) {
	rule := quantityRule
	rule.DisplayName = "数量は正の数"
	other := positiveRule("sales.order_line.discount_positive", "sales.order_line.discount")
	out := goemit.Emit(goemit.Input{PackageName: "rules", Rules: []goemit.Rule{rule, other}})
	if err := out.Err(); err != nil {
		t.Fatal(err)
	}
	display := map[core.CanonicalID]string{rule.Validation.ID: rule.DisplayName}
	for _, trace := range out.Trace {
		file, err := parser.ParseFile(token.NewFileSet(), trace.File, out.File(trace.File), parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		docs := map[string]string{}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Doc != nil {
				docs[fn.Name.Name] = fn.Doc.Text()
			}
		}
		for _, name := range []string{trace.Decision, trace.Check} {
			doc, ok := docs[name]
			if !ok {
				t.Fatalf("%s: function %s has no doc comment", trace.File, name)
			}
			if !namesID(doc, trace.Element) {
				t.Errorf("%s: doc comment of %s does not name rule %s:\n%s", trace.File, name, trace.Element, doc)
			}
			if want := display[trace.Element]; want != "" && !strings.Contains(doc, want) {
				t.Errorf("%s: doc comment of %s does not show display name %q:\n%s", trace.File, name, want, doc)
			}
		}
		if doc := docs[trace.Check]; !namesID(doc, trace.Attribute) {
			t.Errorf("%s: doc comment of %s does not name attribute %s:\n%s", trace.File, trace.Check, trace.Attribute, doc)
		}
	}
}

// namesID reports whether text names id as a whole identifier. Plain
// substring matching would accept the rule ID sales.order_line.quantity_positive
// as naming the attribute sales.order_line.quantity.
func namesID(text string, id core.CanonicalID) bool {
	isIDChar := func(b byte) bool {
		return b == '.' || b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
	}
	s := string(id)
	for from := 0; ; {
		i := strings.Index(text[from:], s)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(s)
		// A trailing '.' ends a sentence; it does not continue the ID.
		if (start == 0 || !isIDChar(text[start-1])) &&
			(end == len(text) || !isIDChar(text[end]) || text[end] == '.' && (end+1 == len(text) || !isIDChar(text[end+1]))) {
			return true
		}
		from = start + 1
	}
}
