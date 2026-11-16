package validationgen_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationgen"
)

// This is a read-only I2M-0 observation of real generated Go. It records
// syntactic evidence and exact Go byte ranges, not accepted TEI meaning,
// complete control-flow equivalence, a public Capability, or a Build receipt.
type observedPositiveGo struct {
	ReadAttribute  string
	ReadCount      int
	StrictPositive bool
	TrueResult     bool
	FalseResult    bool
	SourceNilGuard bool
	ValueNilGuard  bool
	ReadErrorGuard bool
	ReadStart      int
	ReadEnd        int
	ConditionStart int
	ConditionEnd   int
}

func selectorName(expr ast.Expr, receiver, method string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != method {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == receiver
}

func observedReturn(expr ast.Expr, result string) bool {
	return selectorName(expr, "validationeval", result)
}

func bodyReturnsError(body *ast.BlockStmt) bool {
	for _, stmt := range body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 2 {
			continue
		}
		call, ok := ret.Results[1].(*ast.CallExpr)
		if ok && selectorName(call.Fun, "fmt", "Errorf") {
			return true
		}
	}
	return false
}

func observeGeneratedPositiveGo(src []byte) (observedPositiveGo, error) {
	var observed observedPositiveGo
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "generated.go", src, 0)
	if err != nil {
		return observed, err
	}
	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if candidate, ok := decl.(*ast.FuncDecl); ok && candidate.Name.Name == "EvaluateSelected" {
			if fn != nil {
				return observed, fmt.Errorf("more than one EvaluateSelected")
			}
			fn = candidate
		}
	}
	if fn == nil || fn.Body == nil {
		return observed, fmt.Errorf("generated EvaluateSelected function not found")
	}
	span := func(start, end token.Pos) (int, int, bool) {
		sf := fset.File(start)
		if sf == nil || fset.File(end) != sf {
			return 0, 0, false
		}
		a, b := sf.Offset(start), sf.Offset(end)
		return a, b, a >= 0 && a < b && b <= len(src)
	}
	var malformed error
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if malformed != nil {
			return false
		}
		switch n := node.(type) {
		case *ast.CallExpr:
			if !selectorName(n.Fun, "source", "Number") {
				break
			}
			observed.ReadCount++
			if len(n.Args) != 1 {
				malformed = fmt.Errorf("Source.Number must receive exactly one Attribute")
				return false
			}
			lit, ok := n.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				malformed = fmt.Errorf("Source.Number Attribute is not a string literal")
				return false
			}
			observed.ReadAttribute, malformed = strconv.Unquote(lit.Value)
			if malformed != nil {
				return false
			}
			var okSpan bool
			observed.ReadStart, observed.ReadEnd, okSpan = span(n.Pos(), n.End())
			if !okSpan {
				malformed = fmt.Errorf("Source.Number has no exact generated Go byte span")
				return false
			}
		case *ast.IfStmt:
			cmp, ok := n.Cond.(*ast.BinaryExpr)
			if !ok {
				break
			}
			if id, ok := cmp.X.(*ast.Ident); ok {
				if other, ok := cmp.Y.(*ast.Ident); ok && other.Name == "nil" && bodyReturnsError(n.Body) {
					switch {
					case id.Name == "source" && cmp.Op == token.EQL:
						observed.SourceNilGuard = true
					case id.Name == "value" && cmp.Op == token.EQL:
						observed.ValueNilGuard = true
					case id.Name == "err" && cmp.Op == token.NEQ:
						observed.ReadErrorGuard = true
					}
				}
			}
			if cmp.Op != token.GTR {
				break
			}
			call, ok := cmp.X.(*ast.CallExpr)
			if !ok || !selectorName(call.Fun, "value", "Sign") || len(call.Args) != 0 {
				break
			}
			zero, ok := cmp.Y.(*ast.BasicLit)
			if !ok || zero.Kind != token.INT || zero.Value != "0" {
				break
			}
			observed.StrictPositive = true
			var okSpan bool
			observed.ConditionStart, observed.ConditionEnd, okSpan = span(cmp.Pos(), cmp.End())
			if !okSpan {
				malformed = fmt.Errorf("positive condition has no exact Go byte span")
				return false
			}
			for _, stmt := range n.Body.List {
				if ret, ok := stmt.(*ast.ReturnStmt); ok && len(ret.Results) == 2 &&
					observedReturn(ret.Results[0], "ResultSatisfied") {
					observed.TrueResult = true
				}
			}
		case *ast.ReturnStmt:
			if len(n.Results) == 2 && observedReturn(n.Results[0], "ResultNotSatisfied") {
				observed.FalseResult = true
			}
		}
		return true
	})
	if malformed != nil {
		return observed, malformed
	}
	if observed.ReadCount != 1 || observed.ReadAttribute == "" ||
		!observed.StrictPositive || !observed.TrueResult || !observed.FalseResult ||
		!observed.SourceNilGuard || !observed.ValueNilGuard || !observed.ReadErrorGuard {
		return observed, fmt.Errorf("generated decision is not the observed single-read positive shape: %+v", observed)
	}
	if !bytes.Contains(src[observed.ReadStart:observed.ReadEnd], []byte("source.Number(")) ||
		string(src[observed.ConditionStart:observed.ConditionEnd]) != "value.Sign() > 0" {
		return observed, fmt.Errorf("AST byte spans do not match the generated read and condition")
	}
	return observed, nil
}

func TestGeneratedPositiveASTHasTraceableReadAndDecision(t *testing.T) {
	rule := semantic.Validation{
		ID:        "sales.order_line.quantity_positive",
		Kind:      semantic.ValidationPositive,
		Attribute: "sales.order_line.quantity",
	}
	generated, err := validationgen.GeneratePositive(rule, "std.decimal", "main")
	if err != nil {
		t.Fatal(err)
	}
	observed, err := observeGeneratedPositiveGo(generated)
	if err != nil {
		t.Fatal(err)
	}
	if observed.ReadAttribute != "sales.order_line.quantity" {
		t.Fatalf("Go AST observed the wrong read target: %+v", observed)
	}
	// A second typed Rule can produce an equally shaped Go function, but its
	// read target is different. Code formatting alone does not confer identity.
	rule.Attribute = "sales.order_line.discount"
	second, err := validationgen.GeneratePositive(rule, "std.decimal", "main")
	if err != nil {
		t.Fatal(err)
	}
	changed, err := observeGeneratedPositiveGo(second)
	if err != nil {
		t.Fatal(err)
	}
	if changed.ReadAttribute != "sales.order_line.discount" ||
		bytes.Equal(generated, second) {
		t.Fatalf("Rule target edit was not visible in actual Go AST: %+v", changed)
	}

	// A source comment is not a new branch or read. Offsets are recalculated
	// for the new bytes, while the observed behavior shape remains identical.
	commented := append([]byte("// display-only text\n"), generated...)
	afterComment, err := observeGeneratedPositiveGo(commented)
	if err != nil {
		t.Fatal(err)
	}
	withoutPositions := func(o observedPositiveGo) observedPositiveGo {
		o.ReadStart, o.ReadEnd, o.ConditionStart, o.ConditionEnd = 0, 0, 0, 0
		return o
	}
	if !reflect.DeepEqual(withoutPositions(observed), withoutPositions(afterComment)) ||
		afterComment.ReadStart <= observed.ReadStart {
		t.Fatal("comment-only change was mistaken for a new Go decision")
	}

	// The visible control-flow diagram must be derived from actual code.
	// It cannot keep asserting a strict-positive predicate after a >= mutant.
	mutated := bytes.Replace(generated, []byte("value.Sign() > 0"),
		[]byte("value.Sign() >= 0"), 1)
	if bytes.Equal(mutated, generated) {
		t.Fatal("positive branch mutant was not applied")
	}
	if _, err := observeGeneratedPositiveGo(mutated); err == nil {
		t.Fatal("modified threshold was incorrectly observed as strict positive")
	}

	// A syntactic nil check is not sufficient: the guarded branch must
	// actually return an error, otherwise a diagram would invent failure
	// behavior that the generated implementation no longer guarantees.
	droppedError := bytes.Replace(generated, []byte("fmt.Errorf("), []byte("fmt.Sprintf("), 1)
	if bytes.Equal(droppedError, generated) {
		t.Fatal("error-return mutant was not applied")
	}
	if _, err := observeGeneratedPositiveGo(droppedError); err == nil {
		t.Fatal("a nil guard without an error return was accepted as failure handling")
	}
}

func TestGeneratorSourceAndGeneratedDecisionRemainDistinct(t *testing.T) {
	// Observe TEI's handwritten generator separately from its output: an
	// implementation of a Generator is not the business Rule it emits.
	src, err := os.ReadFile("positive.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "positive.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	observedGenerator := false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "GeneratePositive" || fn.Body == nil {
			continue
		}
		observedGenerator = true
		start := fset.File(fn.Pos()).Offset(fn.Pos())
		end := fset.File(fn.End()).Offset(fn.End())
		if start < 0 || end <= start || end > len(src) ||
			!strings.Contains(string(src[start:end]), "format.Source") {
			t.Fatal("Generator AST was not bound to the real implementation byte range")
		}
	}
	if !observedGenerator {
		t.Fatal("the real GeneratePositive source was not observed")
	}
}
