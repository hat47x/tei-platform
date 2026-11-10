package hook

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"tei/internal/core"
)

// ReferenceRuleは、規則Pluginの参照実装である。宣言した読み取り値だけを使い、
// 同じ入力に対して同じ結果を返す。業務規則そのものではなく、フック境界の
// 「不成立」と「実行の失敗」の区別を確かめるために置く。
//
// 判定対象は`Required`と`Positive`で宣言し、`Invocation.Reads`の値だけを見ます。
// 宣言に無い属性は判定しないので、規則の範囲はこの値で表されます。実データの
// 読取りは呼出し側の責務であり、この規則は渡された値を評価するだけです。
type ReferenceRule struct {
	// IDは規則の識別子である。記録と試験の照合に使う。
	ID string
	// Requiredは、値が観測されていなければ不成立とする属性である。
	Required []core.CanonicalID
	// Positiveは、値が正の数でなければ不成立とする属性である。
	Positive []core.CanonicalID
}

// Runは、呼出し元を受け取らない経路のためのImplementation実装である。
func (r ReferenceRule) Run(_ context.Context, invocation Invocation) error {
	return r.evaluate(invocation)
}

// RunWithCallerは、呼出し元を渡されて実行する。この規則は能力を呼ばないため
// 呼出し元を使いませんが、`CallingImplementation`としてRunnerから渡される
// 契約は満たします。能力を呼ぶ規則は別の実装が担います。
func (r ReferenceRule) RunWithCaller(_ context.Context, invocation Invocation, _ Caller) error {
	return r.evaluate(invocation)
}

// evaluateは、宣言した属性を属性名の昇順に判定する。不成立が複数あっても、
// 同じ入力に対して同じ順で報告する。
func (r ReferenceRule) evaluate(invocation Invocation) error {
	for _, attribute := range sortedAttributes(append(append([]core.CanonicalID{}, r.Required...), r.Positive...)) {
		value, observed := invocation.Reads[attribute]
		if !observed {
			return &NotSatisfied{
				Reason:  fmt.Sprintf("attribute %s was not observed", attribute),
				Outcome: OutcomeUnmet,
			}
		}
		if !r.requiresPositive(attribute) {
			continue
		}
		number, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return &NotSatisfied{
				Reason:  fmt.Sprintf("attribute %s is not a number: %s", attribute, value),
				Outcome: OutcomeValueRejected,
				Cause:   err,
			}
		}
		if number <= 0 {
			return &NotSatisfied{
				Reason:  fmt.Sprintf("attribute %s is not positive: %s", attribute, value),
				Outcome: OutcomeValueRejected,
			}
		}
	}
	return nil
}

// requiresPositiveは、その属性が`Positive`の一覧に載っているかを返す。
func (r ReferenceRule) requiresPositive(attribute core.CanonicalID) bool {
	for _, positive := range r.Positive {
		if positive == attribute {
			return true
		}
	}
	return false
}

// sortedAttributesは、重複を除いて属性名の昇順に並べる。
func sortedAttributes(attributes []core.CanonicalID) []core.CanonicalID {
	seen := map[core.CanonicalID]bool{}
	unique := make([]core.CanonicalID, 0, len(attributes))
	for _, attribute := range attributes {
		if seen[attribute] {
			continue
		}
		seen[attribute] = true
		unique = append(unique, attribute)
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
	return unique
}
