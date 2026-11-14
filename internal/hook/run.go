package hook

import (
	"context"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/failure"
	"tei/internal/ir"
)

// Invocationは、宣言されたフックへ渡す入力である。
type Invocation struct {
	Application core.CanonicalID            `json:"application"`
	Pattern     core.CanonicalID            `json:"pattern"`
	Phase       string                      `json:"phase"`
	Reads       map[core.CanonicalID]string `json:"reads,omitempty"`
}

// Implementationは、フックの本体を実装する側の契約である。
// 実装はL4（規則Plugin・Provider）に置き、このパッケージは境界だけを持つ。
type Implementation interface {
	Run(ctx context.Context, invocation Invocation) error
}

// Errorは、フックの前提・実装・拒否を表す。
type Error struct {
	Code    string
	Message string
	Cause   error
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }
func (e *Error) Unwrap() error { return e.Cause }

// FailureCode returns the machine-readable failure code.
func (e *Error) FailureCode() string { return e.Code }

// FailureMessage returns the human-readable failure message.
func (e *Error) FailureMessage() string { return e.Message }

// FailureCause returns the underlying error, if any.
func (e *Error) FailureCause() error { return e.Cause }

// Kind classifies the business meaning of the failure.
func (e *Error) Kind() failure.Kind {
	// The called capability may already have applied an effect. An outer hook
	// refusal must not reclassify that uncertainty as a clean business rejection.
	if underlying, uncertain := uncertainFailure(e.Cause); uncertain {
		return underlying.Kind()
	}
	switch e.Code {
	case "HOOK_PATTERN_NOT_FOUND",
		"HOOK_PATTERN_NOT_RECORD",
		"HOOK_NOT_DECLARED",
		"HOOK_SCOPE_NEEDS_ROWS",
		"HOOK_SCOPE_UNAPPLIED",
		"HOOK_IMPLEMENTATION_UNAVAILABLE",
		"HOOK_CALL_REFUSED":
		return failure.BusinessRejection
	default:
		return failure.TechnicalFailure
	}
}

// Retryable reports whether the caller may safely retry the same operation
// identity after this failure.
func (e *Error) Retryable() failure.RetryPolicy {
	if _, uncertain := uncertainFailure(e.Cause); uncertain {
		return failure.LookupFirst
	}
	switch e.Code {
	case "HOOK_PATTERN_NOT_FOUND",
		"HOOK_PATTERN_NOT_RECORD",
		"HOOK_NOT_DECLARED",
		"HOOK_SCOPE_NEEDS_ROWS",
		"HOOK_SCOPE_UNAPPLIED",
		"HOOK_IMPLEMENTATION_UNAVAILABLE",
		"HOOK_CALL_REFUSED":
		return failure.NotRetryable
	default:
		return failure.NotRetryable
	}
}

// HasEffectUnknown returns the cause's uncertainty when an invoked capability
// may already have applied an effect. Hook-local refusals without such a cause
// are still clean, and retain the existing false default.
func (e *Error) HasEffectUnknown() bool {
	underlying, uncertain := uncertainFailure(e.Cause)
	return uncertain && underlying.HasEffectUnknown()
}

// uncertainFailure inspects the existing shared failure contract; it is not a
// second failure classifier. A lookup-first instruction also blocks retry even
// if the implementation did not mark its effect as unknown.
func uncertainFailure(err error) (failure.Failure, bool) {
	underlying, ok := failure.AsFailure(err)
	if !ok {
		return nil, false
	}
	return underlying, underlying.HasEffectUnknown() || underlying.Retryable() == failure.LookupFirst
}

// Warningは、on-failure warnのフックが失敗したことを表す。
// 呼出し側はWarningを確認して処理を続けるかを決める。
type Warning struct {
	Err error
}

func (w *Warning) Error() string { return "HOOK_WARNING: " + w.Err.Error() }
func (w *Warning) Unwrap() error { return w.Err }

// Runnerは、宣言とImplementationの間に立ち、位相の解決と失敗時の扱いを行う。
type Runner struct {
	World          ir.World
	Implementation Implementation
	// ByPhaseは、位相ごとに別の実装を供給する。規則Pluginは位相ごとに本体を
	// 持ちうるため、一つの実装ですべての位相を賄う前提を置かない。
	//
	// 位相が載っていればその実装を、載っていなければImplementationを使う。
	// どちらも無ければHOOK_IMPLEMENTATION_UNAVAILABLEで止まる（黙って成功に
	// しない）。
	ByPhase map[string]Implementation
	// Callerは、フックから能力を呼ぶ下位の境界である（`CommandCaller`など）。
	// 宣言に制限されていない境界をそのまま実装へ渡さず、必ずDeclaredで包む。
	Caller Caller
	// Rows is the row set a row-scoped hook is applied to. It is empty for a run
	// that handles one record, and a row-scoped hook is then reported as needing
	// the row set rather than run once for the record.
	Rows []map[core.CanonicalID]string
}

// implementationForは、位相に与えられた実装を返す。
//
// 位相ごとの指定が先で、全体のImplementationが後である。位相の名前は
// PhaseOrderが持つものだけを鍵にする（呼出し側が任意の文字列で別の実装を
// 差し込めないよう、宣言された位相だけを見る）。
func (r Runner) implementationFor(phase string) Implementation {
	if impl, ok := r.ByPhase[phase]; ok && impl != nil {
		return impl
	}
	return r.Implementation
}

// Runは、Patternの対話に宣言されたフックを実行する。
func (r Runner) Run(ctx context.Context, invocation Invocation) error {
	var pattern *application.Pattern
	for i := range r.World.Applications {
		if r.World.Applications[i].ID != invocation.Application {
			continue
		}
		for j := range r.World.Applications[i].Patterns {
			if r.World.Applications[i].Patterns[j].ID == invocation.Pattern {
				pattern = &r.World.Applications[i].Patterns[j]
				break
			}
		}
		break
	}
	if pattern == nil {
		return &Error{Code: "HOOK_PATTERN_NOT_FOUND", Message: fmt.Sprintf("pattern %s is not defined in application %s", invocation.Pattern, invocation.Application)}
	}
	if pattern.Kind != application.PatternRecord || pattern.Interaction == nil {
		return &Error{Code: "HOOK_PATTERN_NOT_RECORD", Message: fmt.Sprintf("pattern %s is not a record pattern with interaction", invocation.Pattern)}
	}
	var declared *application.Hook
	for i := range pattern.Interaction.Hooks {
		if pattern.Interaction.Hooks[i].Phase == invocation.Phase {
			declared = &pattern.Interaction.Hooks[i]
			break
		}
	}
	if declared == nil {
		return &Error{Code: "HOOK_NOT_DECLARED", Message: fmt.Sprintf("pattern %s does not declare a %s hook", invocation.Pattern, invocation.Phase)}
	}
	// Run is the single-record entry point, so it applies a record scope and the
	// pattern's own unit. A row-scoped hook needs the row set, which this entry
	// point does not carry: RunRows does that. Any other scope is declared but not
	// applied, which is the X-TEA defect this boundary exists to avoid (D-9: a
	// definition present in the source and inert at run time).
	switch declared.Scope {
	case "", string(application.PatternRecord):
	case "row":
		if len(r.Rows) > 0 {
			return r.RunRows(ctx, invocation, r.Rows)
		}
		return &Error{
			Code:    "HOOK_SCOPE_NEEDS_ROWS",
			Message: fmt.Sprintf("hook %s declares scope row, so it needs the row set; set Runner.Rows or use RunRows", invocation.Phase),
		}
	default:
		return &Error{
			Code:    "HOOK_SCOPE_UNAPPLIED",
			Message: fmt.Sprintf("hook %s declares scope %s but this runner applies one hook per record", invocation.Phase, declared.Scope),
		}
	}
	implementation := r.implementationFor(invocation.Phase)
	if implementation == nil {
		return &Error{Code: "HOOK_IMPLEMENTATION_UNAVAILABLE", Message: "no implementation is configured for this hook boundary"}
	}
	var err error
	if calling, ok := implementation.(CallingImplementation); ok {
		// A hook that calls capabilities is only given the caller its own
		// declaration permits. An unset caller is not a licence: Declared refuses
		// every call without an inner boundary.
		err = calling.RunWithCaller(ctx, invocation, declaredCaller(r.Caller, declared))
	} else {
		err = implementation.Run(ctx, invocation)
	}
	return wrapFailure(err, declared)
}
