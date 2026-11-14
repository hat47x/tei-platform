package hook

import (
	"errors"
	"fmt"

	"tei/internal/command"
)

// 結末名は、フックが「業務上の条件を満たさない」と答えたときに名乗る名前である。
const (
	// OutcomeUnmet は、フックが条件を満たさないとだけ答えた場合の結末名である。
	// Command 側に対応する拒否符号が無いため、Command の結末として報告しない。
	OutcomeUnmet = "not-satisfied"
	// OutcomeValueRejected は、値が宣言した範囲に無い結末名である。Command の
	// `value-rejected` と同じ名前で、`command.FailureOutcome` が返す。
	OutcomeValueRejected = "value-rejected"
)

// NotSatisfiedOutcomes は、`NotSatisfied.Outcome` が取りうる結末名である。
//
// `NotSatisfied.Outcome` は自由な文字列だったため、同じ「値が受け付けられない」
// を指すのに `value-rejected` と `valueRejected` が混在しうる状態でした。Command
// 側の `command.FailureOutcome` が返す名前を正とし、ここに載る名前だけを業務上の
// 結末として扱います。
//
// `not-satisfied` は Command 側に対応する拒否符号が無いため、フックが「条件を
// 満たさない」とだけ答えた場合の名前です。これは Command の拒否理由ではないので、
// 呼出し側は Command の結末として報告せず、フックの判定として報告します。
var NotSatisfiedOutcomes = []string{
	"not-satisfied",
	"not-found",
	"already-exists",
	"value-rejected",
	"rule-rejected",
	"undeclared",
}

// ValidateOutcome は、実装が名乗った結末名が語彙に載っているかを確かめる。
//
// `hook.ReadsFor`が宣言外の読み取りを拒否し、`hook.ProducesFor`が宣言外の提案を
// 拒否するのと同じ立場です。語彙は`NotSatisfiedOutcomes`が一つだけ持ち、ここでは
// それを照合するだけで、新しい名前を足しません。名乗らなかった（空の）結末名は
// 拒否しません。理由だけを返す不成立は正当だからです。
func ValidateOutcome(outcome string) error {
	if outcome == "" {
		return nil
	}
	for _, declared := range NotSatisfiedOutcomes {
		if declared == outcome {
			return nil
		}
	}
	return &Error{
		Code:    "HOOK_OUTCOME_UNDECLARED",
		Message: fmt.Sprintf("hook reported outcome %q that is not in the declared vocabulary", outcome),
	}
}

// CommandOutcomeFor は、フックの結末名を、Command の結末名へ対応づける。
//
// 対応があるのは、フックが Command 側と同じ拒否符号を指した場合だけです。
// 対応が無ければ false を返し、呼出し側はフックの結末として報告します。ここで
// 無理に対応づけると、フックの「条件を満たさない」が Command の拒否理由として
// 読まれてしまいます。
func CommandOutcomeFor(hookOutcome string) (string, bool) {
	switch hookOutcome {
	case "not-found":
		return commandOutcomeName(command.ReferenceRecordMissing), true
	case "already-exists":
		return commandOutcomeName(command.ReferenceRecordExists), true
	case "value-rejected":
		return commandOutcomeName(command.ReferenceValueRejected), true
	case "rule-rejected":
		return commandOutcomeName(command.ReferenceRuleRejected), true
	case "undeclared":
		return commandOutcomeName(command.ReferenceUndeclared), true
	}
	return "", false
}

// commandOutcomeName は、Command の拒否符号を結末名へ写す。写しの規則は
// `command.FailureOutcome` が持つ一つだけを使い、ここでは発明しません。
func commandOutcomeName(code command.ReferenceFailureCode) string {
	name, ok := command.FailureOutcome(&command.ReferenceFailure{Code: code})
	if !ok {
		return string(code)
	}
	return name
}

// OutcomeOf は、フックのエラーが持つ結末名を返す。`NotSatisfied` でなければ
// false を返し、実行の失敗を業務上の結末として読ませません。
func OutcomeOf(err error) (string, bool) {
	// An uncertain Command effect is never a clean business outcome, even
	// when an inner NotSatisfied is still reachable through error unwrapping.
	if _, uncertain := uncertainFailure(err); uncertain {
		return "", false
	}
	var notSatisfied *NotSatisfied
	if !errors.As(err, &notSatisfied) {
		return "", false
	}
	if notSatisfied.Outcome == "" {
		return "", false
	}
	return notSatisfied.Outcome, true
}
