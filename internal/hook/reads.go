package hook

import (
	"context"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
)

// ReadsForは、宣言されたフックへ渡す読み取り値を、宣言の`reads`と照合する。
//
// フックが読む属性は宣言で決まる。宣言より広い値を実装へ渡すと、フックは
// 宣言外のデータを見たことになる。ここでは次を検査する。
//
//   - `reads`に載る属性のうち、値が観測されていないもの（`HOOK_READ_NOT_OBSERVED`）
//   - 観測された属性のうち、`reads`に載っていないもの（`HOOK_READ_UNDECLARED`）
//
// 未観測や宣言外は、空の値として黙って通さない。`Reads`の未設定は、`reads`が
// 空のフックだけ許す。
func ReadsFor(hook *application.Hook, reads map[core.CanonicalID]string) (map[core.CanonicalID]string, error) {
	if hook == nil {
		return nil, &Error{Code: "HOOK_UNDECLARED_CALL", Message: "reads were resolved without a hook declaration"}
	}
	declared := map[core.CanonicalID]struct{}{}
	for _, attribute := range hook.Reads {
		declared[attribute] = struct{}{}
		if _, observed := reads[attribute]; !observed {
			return nil, &Error{
				Code:    "HOOK_READ_NOT_OBSERVED",
				Message: fmt.Sprintf("hook %s declares read %s but no value was observed", hook.Phase, attribute),
			}
		}
	}
	for attribute := range reads {
		if _, ok := declared[attribute]; !ok {
			return nil, &Error{
				Code:    "HOOK_READ_UNDECLARED",
				Message: fmt.Sprintf("hook %s was given %s that it does not declare", hook.Phase, attribute),
			}
		}
	}
	// Return a copy so a caller cannot widen the hook's view after the check.
	resolved := make(map[core.CanonicalID]string, len(reads))
	for attribute, value := range reads {
		resolved[attribute] = value
	}
	return resolved, nil
}

// RunWithDeclaredReadsは、`ReadsFor`で絞った値を実装へ渡して実行する。
// 実装が`CallingImplementation`なら、宣言に制限した呼出し元も渡す。
// `on-failure warn`と不成立の扱いは`Runner.Run`と同じ規則を使う。
func (r Runner) RunWithDeclaredReads(ctx context.Context, invocation Invocation) error {
	pattern, err := r.pattern(invocation.Application, invocation.Pattern)
	if err != nil {
		return err
	}
	declared, err := r.declaration(pattern, invocation.Phase)
	if err != nil {
		return err
	}
	reads, err := ReadsFor(declared, invocation.Reads)
	if err != nil {
		return err
	}
	invocation.Reads = reads
	return r.runDeclared(ctx, invocation, declared)
}

// declarationは、Patternの位相の宣言を返す。
func (r Runner) declaration(pattern *application.Pattern, phase string) (*application.Hook, error) {
	for i := range pattern.Interaction.Hooks {
		if pattern.Interaction.Hooks[i].Phase == phase {
			return &pattern.Interaction.Hooks[i], nil
		}
	}
	return nil, &Error{Code: "HOOK_NOT_DECLARED", Message: fmt.Sprintf("pattern %s does not declare a %s hook", pattern.ID, phase)}
}
