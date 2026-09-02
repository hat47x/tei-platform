package validationeval_test

import (
	"errors"
	"math/big"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
)

type fixtureSource struct {
	numbers      map[core.CanonicalID]*big.Rat
	relationKeys map[string][]string
}

func (s fixtureSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	value, ok := s.numbers[attribute]
	if !ok {
		return nil, errors.New("対象Attributeの数値がありません")
	}
	return value, nil
}

func (s fixtureSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	keys, ok := s.relationKeys[string(relation)+"|"+string(attribute)]
	if !ok {
		return nil, errors.New("対象Relation / Attributeの値がありません")
	}
	return keys, nil
}

func TestReferencePositiveValidationUsesCanonicalAttribute(t *testing.T) {
	rule := referenceValidation(t, "sales.order_line.quantity_positive")

	for _, tt := range []struct {
		name string
		text string
		want validationeval.Result
	}{
		{name: "正の数量", text: "2.5", want: validationeval.ResultSatisfied},
		{name: "ゼロ", text: "0", want: validationeval.ResultNotSatisfied},
		{name: "負の数量", text: "-1", want: validationeval.ResultNotSatisfied},
	} {
		t.Run(tt.name, func(t *testing.T) {
			value, ok := new(big.Rat).SetString(tt.text)
			if !ok {
				t.Fatalf("test fixtureの数値%qを解釈できません", tt.text)
			}
			got, err := validationeval.Evaluate(rule, fixtureSource{
				numbers: map[core.CanonicalID]*big.Rat{
					"sales.order_line.quantity": value,
				},
			})
			if err != nil {
				t.Fatalf("quantity_positiveの評価に失敗しました: %v", err)
			}
			if got != tt.want {
				t.Fatalf("result=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestReferenceRelationUniqueValidationUsesCanonicalRelationScope(t *testing.T) {
	rule := referenceValidation(t, "sales.order.lines.line_number_unique")
	key := "sales.order.lines|sales.order_line.line_number"

	for _, tt := range []struct {
		name string
		keys []string
		want validationeval.Result
	}{
		{name: "明細番号が一意", keys: []string{"1", "2", "3"}, want: validationeval.ResultSatisfied},
		{name: "明細番号が重複", keys: []string{"1", "2", "2"}, want: validationeval.ResultNotSatisfied},
		{name: "明細が空", keys: nil, want: validationeval.ResultSatisfied},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validationeval.Evaluate(rule, fixtureSource{
				relationKeys: map[string][]string{key: tt.keys},
			})
			if err != nil {
				t.Fatalf("line_number_uniqueの評価に失敗しました: %v", err)
			}
			if got != tt.want {
				t.Fatalf("result=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestEvaluationFailureIsNotNotSatisfied(t *testing.T) {
	rule := referenceValidation(t, "sales.order_line.quantity_positive")

	got, err := validationeval.Evaluate(rule, fixtureSource{})
	if err == nil {
		t.Fatalf("評価入力がないのにerrorになりませんでした: result=%q", got)
	}
	if got == validationeval.ResultNotSatisfied {
		t.Fatalf("評価不能がNotSatisfiedへ畳まれています: result=%q err=%v", got, err)
	}
}

func referenceValidation(t *testing.T, id core.CanonicalID) semantic.Validation {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test fileの位置を取得できません")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}
	for _, ns := range loaded.World.Semantics {
		for _, rule := range ns.Validations {
			if rule.ID == id {
				return rule
			}
		}
	}
	t.Fatalf("Semantic Validation %sがReference Workspaceに見つかりません", id)
	return semantic.Validation{}
}
