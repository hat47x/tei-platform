package hook

import (
	"context"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
)

// Producedは、フックが提案する項目値である。宣言の`produces`に載る属性だけを
// 受け取り、それ以外は拒否する。値の採用（確定）は呼出し側の判断であり、この型は
// 提案を運ぶだけである。
type Produced map[core.CanonicalID]string

// ProducingImplementationは、値の提案を返すImplementationである。読み取りだけの
// フックは`Implementation`のままでよく、値を作るフックだけがこの契約を使う。
type ProducingImplementation interface {
	Implementation
	RunProducing(ctx context.Context, invocation Invocation) (Produced, error)
}

// ProducesForは、フックが提案した値を、宣言の`produces`と照合する。
//
// 宣言していない属性の値を提案すると、フックは宣言より広い項目を書いたことに
// なる。ここでは、`produces`に載っていない属性を`HOOK_PRODUCE_UNDECLARED`で
// 拒否する。提案が空でも、それは「何も作らない」であり、拒否ではない。
func ProducesFor(hook *application.Hook, produced Produced) (Produced, error) {
	if hook == nil {
		return nil, &Error{Code: "HOOK_UNDECLARED_CALL", Message: "produced values were resolved without a hook declaration"}
	}
	declared := map[core.CanonicalID]struct{}{}
	for _, attribute := range hook.Produces {
		declared[attribute] = struct{}{}
	}
	for attribute := range produced {
		if _, ok := declared[attribute]; !ok {
			return nil, &Error{
				Code:    "HOOK_PRODUCE_UNDECLARED",
				Message: fmt.Sprintf("hook %s produced %s that it does not declare", hook.Phase, attribute),
			}
		}
	}
	// Return a copy so a caller cannot widen the proposal after the check.
	resolved := make(Produced, len(produced))
	for attribute, value := range produced {
		resolved[attribute] = value
	}
	return resolved, nil
}

// RunWithDeclaredValuesは、宣言に合わせて値を絞り、実装を実行し、提案された値を
// 照合して返す。読み取りは`ReadsFor`で絞り、値の提案は`ProducesFor`で照合する。
//
// 実装が`ProducingImplementation`なら提案を受け取り、そうでなければ提案は空である。
// `on-failure warn`と不成立の扱いは`Runner.Run`と同じ規則を使う。
func (r Runner) RunWithDeclaredValues(ctx context.Context, invocation Invocation) (Produced, error) {
	pattern, err := r.pattern(invocation.Application, invocation.Pattern)
	if err != nil {
		return nil, err
	}
	declared, err := r.declaration(pattern, invocation.Phase)
	if err != nil {
		return nil, err
	}
	reads, err := ReadsFor(declared, invocation.Reads)
	if err != nil {
		return nil, err
	}
	invocation.Reads = reads
	implementation := r.implementationFor(invocation.Phase)
	if implementation == nil {
		return nil, &Error{Code: "HOOK_IMPLEMENTATION_UNAVAILABLE", Message: "no implementation is configured for this hook boundary"}
	}
	var proposed Produced
	if producing, ok := implementation.(ProducingImplementation); ok {
		proposed, err = producing.RunProducing(ctx, invocation)
		// A producing hook is still a hook: its failure takes the same kind as a
		// plain one, so on-failure warn stays a warning and a refusal stays a
		// refusal.
		if err != nil {
			return nil, wrapFailure(err, declared)
		}
	} else {
		if err := r.runDeclared(ctx, invocation, declared); err != nil {
			return nil, err
		}
	}
	return ProducesFor(declared, proposed)
}
