package validationgen_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"

	"tei/internal/codegen/goemit"
	"tei/internal/codegen/goemit/goemittest"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationgen"
)

// This is a read-only I2M-0 observation of real generated Go. It records
// syntactic evidence and exact Go byte ranges, not accepted TEI meaning,
// complete control-flow equivalence, a public Capability, or a Build receipt.
//
// The generated code reads nothing itself: the caller passes the value. What
// the code is bound to is the structured Trace of goemit.Emit, which names the
// Attribute, the check function and its parameter. The observation holds only
// when the parsed file declares that check function with that parameter.
// BoundAttribute comes from the Trace; the Binding span is the parameter in
// the check function's signature. Comments are never read: rewording them
// leaves the observation as it is.
type observedPositiveGo struct {
	BoundAttribute    string
	BindingCount      int
	Decision          string
	Check             string
	Parameter         string
	StrictPositive    bool
	NilGuardUndecided bool
	TrueResult        bool
	FalseResult       bool
	BindingStart      int
	BindingEnd        int
	ConditionStart    int
	ConditionEnd      int
}

// tracedPositive lowers rule with GeneratePositive and returns the rule file
// with the Trace that goemit.Emit gives for the same input. It fails when the
// two disagree on the bytes.
func tracedPositive(rule semantic.Validation, attributeType core.CanonicalID, packageName string) ([]byte, goemit.Trace, error) {
	src, err := validationgen.GeneratePositive(rule, attributeType, packageName)
	if err != nil {
		return nil, goemit.Trace{}, err
	}
	out := goemit.Emit(goemit.Input{PackageName: packageName, Rules: []goemit.Rule{{Validation: rule, AttributeType: attributeType}}})
	if err := out.Err(); err != nil {
		return nil, goemit.Trace{}, err
	}
	if len(out.Trace) != 1 || !bytes.Equal(src, out.File(out.Trace[0].File)) {
		return nil, goemit.Trace{}, fmt.Errorf("GeneratePositive and goemit.Emit disagree: %+v", out.Trace)
	}
	return src, out.Trace[0], nil
}

// ratParameter returns the name of the single *big.Rat parameter of fn and
// the name of its single result type, or "" when fn has another shape.
func ratParameter(fn *ast.FuncDecl) (string, string) {
	params, results := fn.Type.Params, fn.Type.Results
	if params == nil || len(params.List) != 1 || len(params.List[0].Names) != 1 ||
		results == nil || len(results.List) != 1 {
		return "", ""
	}
	star, ok := params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return "", ""
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Rat" {
		return "", ""
	}
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "big" {
		return "", ""
	}
	result, ok := results.List[0].Type.(*ast.Ident)
	if !ok {
		return "", ""
	}
	return params.List[0].Names[0].Name, result.Name
}

func identNamed(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

// returnsIdent reports whether body is exactly "return <name>".
func returnsIdent(body *ast.BlockStmt, name string) bool {
	if body == nil || len(body.List) != 1 {
		return false
	}
	ret, ok := body.List[0].(*ast.ReturnStmt)
	return ok && len(ret.Results) == 1 && identNamed(ret.Results[0], name)
}

func observeGeneratedPositiveGo(src []byte, trace goemit.Trace) (observedPositiveGo, error) {
	var observed observedPositiveGo
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "generated.go", src, parser.ParseComments)
	if err != nil {
		return observed, err
	}
	span := func(start, end token.Pos) (int, int, bool) {
		sf := fset.File(start)
		if sf == nil || fset.File(end) != sf {
			return 0, 0, false
		}
		a, b := sf.Offset(start), sf.Offset(end)
		return a, b, a >= 0 && a < b && b <= len(src)
	}
	var decision, check *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		switch _, result := ratParameter(fn); result {
		case "bool":
			if decision != nil {
				return observed, fmt.Errorf("more than one decision function")
			}
			decision = fn
		case "Result":
			if check != nil {
				return observed, fmt.Errorf("more than one check function")
			}
			check = fn
		}
	}
	if decision == nil || check == nil {
		return observed, fmt.Errorf("generated decision and check functions not found")
	}
	observed.Decision, observed.Check = decision.Name.Name, check.Name.Name
	observed.Parameter, _ = ratParameter(check)
	if parameter, _ := ratParameter(decision); parameter != observed.Parameter {
		return observed, fmt.Errorf("decision and check functions take different values")
	}

	// Decision: exactly "return <p>.Sign() > 0".
	if len(decision.Body.List) == 1 {
		if ret, ok := decision.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
			if cmp, ok := ret.Results[0].(*ast.BinaryExpr); ok && cmp.Op == token.GTR {
				call, okCall := cmp.X.(*ast.CallExpr)
				zero, okZero := cmp.Y.(*ast.BasicLit)
				if okCall && okZero && len(call.Args) == 0 &&
					zero.Kind == token.INT && zero.Value == "0" {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Sign" &&
						identNamed(sel.X, observed.Parameter) {
						var okSpan bool
						observed.ConditionStart, observed.ConditionEnd, okSpan = span(cmp.Pos(), cmp.End())
						if !okSpan {
							return observed, fmt.Errorf("positive condition has no exact Go byte span")
						}
						observed.StrictPositive = true
					}
				}
			}
		}
	}

	// Check: nil guard, decision call, final answer.
	if len(check.Body.List) == 3 {
		if guard, ok := check.Body.List[0].(*ast.IfStmt); ok && guard.Init == nil && guard.Else == nil {
			if cmp, ok := guard.Cond.(*ast.BinaryExpr); ok && cmp.Op == token.EQL &&
				identNamed(cmp.X, observed.Parameter) && identNamed(cmp.Y, "nil") {
				observed.NilGuardUndecided = returnsIdent(guard.Body, "Undecided")
			}
		}
		if decide, ok := check.Body.List[1].(*ast.IfStmt); ok && decide.Init == nil && decide.Else == nil {
			if call, ok := decide.Cond.(*ast.CallExpr); ok && identNamed(call.Fun, observed.Decision) &&
				len(call.Args) == 1 && identNamed(call.Args[0], observed.Parameter) {
				observed.TrueResult = returnsIdent(decide.Body, "Satisfied")
			}
		}
		if ret, ok := check.Body.List[2].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
			observed.FalseResult = identNamed(ret.Results[0], "NotSatisfied")
		}
	}

	// Binding: the Trace names the check function, its parameter and the
	// Attribute whose value that parameter receives. It counts only when the
	// declared functions are the ones the Trace names.
	if trace.Check == observed.Check && trace.Decision == observed.Decision &&
		trace.Parameter == observed.Parameter && trace.Attribute != "" {
		observed.BindingCount++
		observed.BoundAttribute = string(trace.Attribute)
		parameter := check.Type.Params.List[0].Names[0]
		var okSpan bool
		observed.BindingStart, observed.BindingEnd, okSpan = span(parameter.Pos(), parameter.End())
		if !okSpan {
			return observed, fmt.Errorf("bound parameter has no exact Go byte span")
		}
	}

	if observed.BindingCount != 1 || observed.BoundAttribute == "" ||
		!observed.StrictPositive || !observed.NilGuardUndecided ||
		!observed.TrueResult || !observed.FalseResult {
		return observed, fmt.Errorf("generated decision is not the observed bound positive shape: %+v (trace %s)", observed, trace)
	}
	if string(src[observed.BindingStart:observed.BindingEnd]) != observed.Parameter ||
		string(src[observed.ConditionStart:observed.ConditionEnd]) != observed.Parameter+".Sign() > 0" {
		return observed, fmt.Errorf("AST byte spans do not match the generated binding and condition")
	}
	return observed, nil
}

func TestGeneratedPositiveASTHasTraceableReadAndDecision(t *testing.T) {
	rule := semantic.Validation{
		ID:        "sales.order_line.quantity_positive",
		Kind:      semantic.ValidationPositive,
		Attribute: "sales.order_line.quantity",
	}
	generated, trace, err := tracedPositive(rule, "std.decimal", "main")
	if err != nil {
		t.Fatal(err)
	}
	observed, err := observeGeneratedPositiveGo(generated, trace)
	if err != nil {
		t.Fatal(err)
	}
	if observed.BoundAttribute != "sales.order_line.quantity" || observed.Check != "CheckQuantityIsPositive" ||
		observed.Parameter != "quantity" {
		t.Fatalf("Go AST observed the wrong bound attribute: %+v", observed)
	}
	// A second typed Rule can produce an equally shaped Go function, but it
	// is bound to a different attribute. Code formatting alone does not
	// confer identity.
	rule.Attribute = "sales.order_line.discount"
	second, secondTrace, err := tracedPositive(rule, "std.decimal", "main")
	if err != nil {
		t.Fatal(err)
	}
	changed, err := observeGeneratedPositiveGo(second, secondTrace)
	if err != nil {
		t.Fatal(err)
	}
	if changed.BoundAttribute != "sales.order_line.discount" || changed.Check != "CheckDiscountIsPositive" ||
		bytes.Equal(generated, second) {
		t.Fatalf("Rule target edit was not visible in actual Go AST: %+v", changed)
	}
	// The Trace binds only the functions the file declares: the discount
	// Trace does not bind the quantity file.
	if _, err := observeGeneratedPositiveGo(generated, secondTrace); err == nil {
		t.Fatal("a Trace naming functions the file does not declare was accepted as its binding")
	}

	withoutPositions := func(o observedPositiveGo) observedPositiveGo {
		o.BindingStart, o.BindingEnd, o.ConditionStart, o.ConditionEnd = 0, 0, 0, 0
		return o
	}
	// A source comment is not a new branch or binding. Offsets are
	// recalculated for the new bytes, while the observed shape is identical.
	commented := append([]byte("// display-only text\n"), generated...)
	afterComment, err := observeGeneratedPositiveGo(commented, trace)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(withoutPositions(observed), withoutPositions(afterComment)) ||
		afterComment.BindingStart <= observed.BindingStart {
		t.Fatal("comment-only change was mistaken for a new Go decision")
	}
	// Rewriting every comment, the doc sentence that names the attribute
	// among them, leaves the observation as it is: nothing is read from
	// comment text.
	reworded, err := goemittest.RewriteComments(goemit.Output{
		Files: []goemit.File{{Path: trace.File, Content: generated}},
		Trace: []goemit.Trace{trace},
	}, func(string, string) string { return "説明だけを書き換えた。" })
	if err != nil {
		t.Fatal(err)
	}
	rewordedSrc := reworded.File(trace.File)
	if bytes.Contains(rewordedSrc, []byte("sales.order_line.quantity")) {
		t.Fatalf("the comment rewrite left the attribute in the file:\n%s", rewordedSrc)
	}
	afterReword, err := observeGeneratedPositiveGo(rewordedSrc, trace)
	if err != nil {
		t.Fatalf("rewording comments broke the observation: %v", err)
	}
	if !reflect.DeepEqual(withoutPositions(observed), withoutPositions(afterReword)) {
		t.Fatalf("rewording comments changed the observation: %+v / %+v", observed, afterReword)
	}

	// The visible control-flow diagram must be derived from actual code.
	// It cannot keep asserting a strict-positive predicate after a >= mutant.
	mutated := bytes.Replace(generated, []byte("quantity.Sign() > 0"),
		[]byte("quantity.Sign() >= 0"), 1)
	if bytes.Equal(mutated, generated) {
		t.Fatal("positive branch mutant was not applied")
	}
	if _, err := observeGeneratedPositiveGo(mutated, trace); err == nil {
		t.Fatal("modified threshold was incorrectly observed as strict positive")
	}

	// A syntactic nil check is not sufficient: the guarded branch must
	// actually answer Undecided. A guard that answers NotSatisfied would
	// read a missing value as a failed rule.
	missingAsFailure := bytes.Replace(generated, []byte("return Undecided"), []byte("return NotSatisfied"), 1)
	if bytes.Equal(missingAsFailure, generated) {
		t.Fatal("nil-guard mutant was not applied")
	}
	if _, err := observeGeneratedPositiveGo(missingAsFailure, trace); err == nil {
		t.Fatal("a nil guard without Undecided was accepted as missing-value handling")
	}
}

func TestGeneratorSourceAndGeneratedDecisionRemainDistinct(t *testing.T) {
	// Observe TEI's handwritten generator separately from its output: an
	// implementation of a Generator is not the business Rule it emits.
	// GeneratePositive forwards to goemit; the Go is built in goemit.
	observe := func(path, function, want string) {
		t.Helper()
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		observedGenerator := false
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != function || fn.Body == nil {
				continue
			}
			observedGenerator = true
			start := fset.File(fn.Pos()).Offset(fn.Pos())
			end := fset.File(fn.End()).Offset(fn.End())
			if start < 0 || end <= start || end > len(src) ||
				!strings.Contains(string(src[start:end]), want) {
				t.Fatalf("Generator AST %s was not bound to the real implementation byte range", function)
			}
		}
		if !observedGenerator {
			t.Fatalf("the real %s source was not observed", function)
		}
	}
	observe("positive.go", "GeneratePositive", "goemit.Emit(")
	observe("../../codegen/goemit/positive.go", "positiveFile", "printer.Fprint(")
}
