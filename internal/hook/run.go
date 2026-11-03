package hook

import (
	"context"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
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
	if r.Implementation == nil {
		return &Error{Code: "HOOK_IMPLEMENTATION_UNAVAILABLE", Message: "no implementation is configured for this hook boundary"}
	}
	if err := r.Implementation.Run(ctx, invocation); err != nil {
		if declared.Failure == "warn" {
			return &Warning{Err: err}
		}
		return &Error{Code: "HOOK_REJECTED", Message: err.Error(), Cause: err}
	}
	return nil
}
