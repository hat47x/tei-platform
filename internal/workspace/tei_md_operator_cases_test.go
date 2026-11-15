package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

// The examples of an operator are read by the product Load from a table and
// attached to the operator, wherever the operator is declared.

const teiMDOperatorCases = "semantic/sales/a-cases.tei.md"

func teiMDOperatorCasesWorkspace(t *testing.T) string {
	t.Helper()
	root := teiMDOperatorWorkspace(t)
	rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.rule-authoring/v0", "tei.authoring/v1")
	// This document sorts before the one that declares the operator.
	writeTEIMDTestFile(t, root, teiMDOperatorCases,
		"# 値引きの検査の例\n\n"+
			"<!-- tei: cases of sales.non_negative -->\n"+
			"| value | expect | note |\n| --- | --- | --- |\n"+
			"| 300 | satisfied | |\n| 0 | satisfied | 値引きなしは正しい入力 |\n| -0.01 | not-satisfied | |\n")
	return root
}

func TestTEIMDOperatorCasesAreAttachedToTheDeclaredOperator(t *testing.T) {
	root := teiMDOperatorCasesWorkspace(t)
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics := validation.Validate(loaded.World); len(diagnostics) != 0 {
		t.Fatalf("validator rejects the examples: %+v", diagnostics)
	}
	want := semantic.Operator{
		ID: "sales.non_negative", Symbol: "non_negative", Input: semantic.OperatorInputNumber,
		Cases: []semantic.OperatorCase{
			{Value: "300", Satisfied: true}, {Value: "0", Satisfied: true}, {Value: "-0.01", Satisfied: false},
		},
	}
	declared := operatorsOf(loaded)
	if len(declared) != 1 || !reflect.DeepEqual(declared[0], want) {
		t.Fatalf("operators=%+v, want %+v", declared, want)
	}
	if len(loaded.pendingOperatorCases) != 0 {
		t.Fatal("a returned Result still holds unattached case tables")
	}

	// A Workspace that states no example serializes none.
	plain, err := Load(teiMDOperatorWorkspace(t))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(plain.World.Semantics)
	if strings.Contains(string(raw), `"cases"`) {
		t.Fatalf("an operator without examples serializes a cases field: %s", raw)
	}
}

func TestTEIMDOperatorCasesNegatives(t *testing.T) {
	sourceErrorAt := func(t *testing.T, root, path, text, message string) {
		t.Helper()
		_, err := Load(root)
		var sourceError *SourceError
		if !errors.As(err, &sourceError) {
			t.Fatalf("want a located error, got %v", err)
		}
		if !strings.Contains(sourceError.Message, message) {
			t.Fatalf("message=%q, want %q", sourceError.Message, message)
		}
		document := readTEIMDTestFile(t, root, path)
		if bytes.Count(document, []byte(text)) != 1 {
			t.Fatalf("%q must occur once in %s", text, path)
		}
		if want := bytes.Index(document, []byte(text)); sourceError.Range.Start.Offset != want {
			t.Fatalf("error offset=%d, want %d", sourceError.Range.Start.Offset, want)
		}
	}

	t.Run("examples for an operator no source declares", func(t *testing.T) {
		root := teiMDOperatorCasesWorkspace(t)
		rewriteTEIMDTestFile(t, root, teiMDOperatorCases, "cases of sales.non_negative", "cases of sales.non_negativ")
		sourceErrorAt(t, root, teiMDOperatorCases, "sales.non_negativ ", "no source declares that operator")
	})

	t.Run("examples for an attribute", func(t *testing.T) {
		root := teiMDOperatorCasesWorkspace(t)
		rewriteTEIMDTestFile(t, root, teiMDOperatorCases, "cases of sales.non_negative", "cases of sales.order_line.discount")
		sourceErrorAt(t, root, teiMDOperatorCases, "sales.order_line.discount ", "no source declares that operator")
	})

	t.Run("two case tables for one operator", func(t *testing.T) {
		root := teiMDOperatorCasesWorkspace(t)
		const second = "semantic/sales/b-more-cases.tei.md"
		writeTEIMDTestFile(t, root, second,
			"<!-- tei: cases of sales.non_negative -->\n| value | expect |\n| --- | --- |\n| 7 | satisfied |\n")
		sourceErrorAt(t, root, second, "sales.non_negative ", "already has a case table at "+teiMDOperatorCases)
	})

	t.Run("mark this language does not know", func(t *testing.T) {
		root := teiMDOperatorCasesWorkspace(t)
		rewriteTEIMDTestFile(t, root, teiMDOperatorCases, "cases of sales.non_negative", "examples of sales.non_negative")
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "unknown tei mark") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("the fence-only language refuses the case table", func(t *testing.T) {
		root := teiMDOperatorCasesWorkspace(t)
		rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.authoring/v1", "tei.rule-authoring/v0")
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "comment marks are not read") {
			t.Fatalf("err=%v", err)
		}
	})
}

// The validator checks examples that reach it without passing the table
// reader, such as a World given as JSON.
func TestSemanticOperatorCaseCodesAreProduced(t *testing.T) {
	valid := func() ir.World {
		return ir.World{Semantics: []semantic.Namespace{{
			ID: "sales",
			Operators: []semantic.Operator{{
				ID: "sales.non_negative", Symbol: "non_negative", Input: semantic.OperatorInputNumber,
				Cases: []semantic.OperatorCase{{Value: "0", Satisfied: true}, {Value: "-1", Satisfied: false}},
			}},
		}}}
	}
	if diagnostics := validation.Validate(valid()); len(diagnostics) != 0 {
		t.Fatalf("baseline world is rejected: %+v", diagnostics)
	}
	for name, tc := range map[string]struct {
		mutate func(*ir.World)
		want   string
	}{
		"value that is not a decimal": {func(w *ir.World) { w.Semantics[0].Operators[0].Cases[0].Value = "1e5" }, "INVALID_SEMANTIC_OPERATOR_CASE"},
		"empty value":                 {func(w *ir.World) { w.Semantics[0].Operators[0].Cases[0].Value = "" }, "INVALID_SEMANTIC_OPERATOR_CASE"},
		"same number twice": {func(w *ir.World) {
			w.Semantics[0].Operators[0].Cases = append(w.Semantics[0].Operators[0].Cases, semantic.OperatorCase{Value: "0.0", Satisfied: false})
		}, "DUPLICATE_SEMANTIC_OPERATOR_CASE"},
	} {
		t.Run(name, func(t *testing.T) {
			world := valid()
			tc.mutate(&world)
			var codes []string
			found := false
			for _, diagnostic := range validation.Validate(world) {
				codes = append(codes, diagnostic.Code)
				found = found || diagnostic.Code == tc.want && diagnostic.Subject != nil && diagnostic.Subject.ID == "sales.non_negative"
			}
			if !found {
				t.Fatalf("want %s on sales.non_negative, got %v", tc.want, codes)
			}
		})
	}
}
