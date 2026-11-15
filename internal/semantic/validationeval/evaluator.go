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
		return "", fmt.Errorf("no Source is specified to evaluate semantic validation %s", rule.ID)
	}
	decision, err := decide(rule, legacySource{source}, Operators{}, false)
	if err != nil {
		return "", err
	}
	return decision.Result, nil
}
