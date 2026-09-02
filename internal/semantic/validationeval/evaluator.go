package validationeval

import (
	"fmt"
	"math/big"

	"tei/internal/core"
	"tei/internal/semantic"
)

// Resultは、業務Validationを正常に評価できたときの結末を表す。
// 評価そのものを完了できなかった場合はResultへ畳まず、Go errorとして返す。
type Result string

const (
	ResultSatisfied    Result = "satisfied"
	ResultNotSatisfied Result = "not-satisfied"
)

// Sourceは、Canonical Validationが参照する値を実データから取り出すための最小境界である。
//
// EvaluatorはApplication、Driver、Database、Web固有のデータ表現を知らない。
// 呼び出し側は、対象Attributeの数値や、Relation scope内で比較するAttribute値の
// 安定した等値比較キーだけをこの境界へ渡す。
type Source interface {
	Number(attribute core.CanonicalID) (*big.Rat, error)
	RelationKeys(relation, attribute core.CanonicalID) ([]string, error)
}

// Evaluateは、Canonical Validationのrule本文を正本として業務Validationを評価する。
//
// NotSatisfiedは正常に評価できた業務上の不成立であり、errorではない。
// ruleの解決に必要な入力がない、値を評価できない、未対応kindである等の場合だけerrorを返す。
func Evaluate(rule semantic.Validation, source Source) (Result, error) {
	if source == nil {
		return "", fmt.Errorf("semantic validation %sを評価するSourceが指定されていません", rule.ID)
	}

	switch rule.Kind {
	case semantic.ValidationPositive:
		value, err := source.Number(rule.Attribute)
		if err != nil {
			return "", fmt.Errorf("semantic validation %sのattribute %sを評価できません: %w", rule.ID, rule.Attribute, err)
		}
		if value == nil {
			return "", fmt.Errorf("semantic validation %sのattribute %sに評価可能な数値がありません", rule.ID, rule.Attribute)
		}
		if value.Sign() > 0 {
			return ResultSatisfied, nil
		}
		return ResultNotSatisfied, nil

	case semantic.ValidationRelationUnique:
		keys, err := source.RelationKeys(rule.Relation, rule.Attribute)
		if err != nil {
			return "", fmt.Errorf("semantic validation %sのrelation %s / attribute %sを評価できません: %w", rule.ID, rule.Relation, rule.Attribute, err)
		}
		seen := make(map[string]struct{}, len(keys))
		for _, key := range keys {
			if _, exists := seen[key]; exists {
				return ResultNotSatisfied, nil
			}
			seen[key] = struct{}{}
		}
		return ResultSatisfied, nil

	default:
		return "", fmt.Errorf("semantic validation %sのkind %qはEvaluatorで扱えません", rule.ID, rule.Kind)
	}
}
