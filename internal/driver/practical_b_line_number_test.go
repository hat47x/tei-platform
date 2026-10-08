package driver_test

import (
	"math/big"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// 実証記録Bの規則 sales.order_line.line_number_positive は、確認用Scenarioのsourceから行番号を読めなければ評価できない。
// このtestは、sourceが行番号を数値として渡し、規則の判定はvalidationevalが行うことを確かめる。
func TestPracticalBLineNumberPositiveEvaluatesLineNumberThroughDriverSource(t *testing.T) {
	rule := semantic.Validation{
		ID:        "sales.order_line.line_number_positive",
		Kind:      semantic.ValidationPositive,
		Attribute: "sales.order_line.line_number",
	}
	cases := []struct {
		name       string
		lineNumber *big.Rat
		want       validationeval.Result
	}{
		{"line number 1 is satisfied", big.NewRat(1, 1), validationeval.ResultSatisfied},
		{"line number 0 is not satisfied", big.NewRat(0, 1), validationeval.ResultNotSatisfied},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validationeval.Evaluate(rule, submitOrderValidationSource{lineNumber: tc.lineNumber})
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("result=%s want=%s", got, tc.want)
			}
		})
	}

	t.Run("missing line number is an error, not a business result", func(t *testing.T) {
		if _, err := validationeval.Evaluate(rule, submitOrderValidationSource{}); err == nil {
			t.Fatal("行番号が無い評価を業務上の結果として返しています")
		}
	})
}
