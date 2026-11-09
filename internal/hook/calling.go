package hook

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/application"
)

// CallingImplementationは、宣言に制限されたCallerを受け取って実行するImplementationである。
// フック本体が能力を呼ぶ場合はこの契約を使う。宣言外の呼出しは、実装の判断ではなく
// Runnerが渡すDeclaredの層で拒否される。
//
// Implementationを埋め込むのは、既存のImplementationとしてRunnerへ渡せるようにするためである。
// Runnerはこの契約を持つ実装ではRunWithCallerだけを呼び、Runは呼ばない。
type CallingImplementation interface {
	Implementation
	RunWithCaller(ctx context.Context, invocation Invocation, caller Caller) error
}

// NotSatisfiedは、フックが業務上の条件を満たさないと判断したことを表す。
// 実行の失敗ではないため、ErrorやWarningとして包まない。呼出し側は業務上の
// 結末として扱い、実行の失敗と同じ経路で再試行しない。
type NotSatisfied struct {
	// Reasonは、判定の理由である。秘密や非公開の規則をそのまま観測へ出さない
	// ため、利用者へ示す文言は呼出し側が決める。
	Reason string
	// Outcomeは、対応する業務上の結末の名前である（任意）。
	Outcome string
	// Causeは、判断の根拠になったエラー（任意）。
	Cause error
}

func (n *NotSatisfied) Error() string {
	if n.Outcome != "" {
		return fmt.Sprintf("HOOK_NOT_SATISFIED: %s: %s", n.Outcome, n.Reason)
	}
	return "HOOK_NOT_SATISFIED: " + n.Reason
}

func (n *NotSatisfied) Unwrap() error { return n.Cause }

// isNotSatisfiedは、エラーが業務上の不成立かを返す。
func isNotSatisfied(err error) bool {
	var notSatisfied *NotSatisfied
	return errors.As(err, &notSatisfied)
}

// declaredCallerは、Runnerが実装へ渡す呼出し元を、そのフックの宣言で包む。
// 宣言に載っていない能力は、この層で拒否される。呼出し先が未設定でも
// 「許可」ではなく拒否として返る。
func declaredCaller(inner Caller, declared *application.Hook) Caller {
	return Declared{Inner: inner, Hook: declared}
}
