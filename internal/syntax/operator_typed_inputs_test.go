package syntax_test

import (
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// Road item 5's typed-value change added four input kinds to the vocabulary.
// Each kind is a word an author writes and the meaning has to carry, so each is
// checked here: writing it declares an operator whose input keeps the word, and
// converting the declaration carries it into the meaning. A kind that is parsed
// but dropped would leave a contract that cannot be checked.
func TestTheFourTypedInputKindsAreCarriedIntoTheDeclaration(t *testing.T) {
	cases := []struct {
		word string
		kind semantic.OperatorInput
	}{
		{"date", semantic.OperatorInputDate},
		{"text", semantic.OperatorInputText},
		{"truth", semantic.OperatorInputTruth},
		{"any", semantic.OperatorInputAny},
	}
	for _, tc := range cases {
		t.Run(tc.word, func(t *testing.T) {
			source := "namespace sales\noperator sales.op:\n  takes value: " + tc.word + "\n"
			file, err := syntax.ParseRuleAuthoringFile(source)
			if err != nil {
				t.Fatalf("%s was refused: %v", tc.word, err)
			}
			if len(file.Operators) != 1 {
				t.Fatalf("operators=%d, want the one declaration", len(file.Operators))
			}
			operator := file.Operators[0]
			if len(operator.Inputs) != 1 {
				t.Fatalf("inputs=%d, want the one input", len(operator.Inputs))
			}
			if got := string(operator.Inputs[0].Word); got != tc.word {
				t.Fatalf("word=%q, want %q", got, tc.word)
			}
			// The declaration has to carry the word into the meaning, because
			// a contract the evaluator cannot read answers nothing.
			namespace, issues := syntax.RuleNamespace(file)
			if len(issues) != 0 {
				t.Fatalf("the declaration was refused: %+v", issues)
			}
			if len(namespace.Operators) != 1 {
				t.Fatalf("the meaning carries %d operators, want the one", len(namespace.Operators))
			}
			declared := namespace.Operators[0]
			if len(declared.Inputs) != 1 || declared.Inputs[0].Type != tc.kind {
				t.Fatalf("the meaning carries %+v, want the input %s", declared.Inputs, tc.kind)
			}
		})
	}
}
