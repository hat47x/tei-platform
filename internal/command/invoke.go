package command

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/applicationview"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// Requestは、Commandを実行する呼出しの入力である。
type Request struct {
	Application core.CanonicalID  `json:"application"`
	Pattern     core.CanonicalID  `json:"pattern"`
	Capability  core.CanonicalID  `json:"capability"`
	Inputs      map[string]string `json:"inputs,omitempty"`
}

// Resultは、Commandの正常終了で得られる業務上の結末と効果である。
// 実行そのものの失敗はResultではなくerrorで返す。
type Result struct {
	Outcome *semantic.Outcome    `json:"outcome,omitempty"`
	Effect  *semantic.DataEffect `json:"effect,omitempty"`
}

// Executorは、宣言されたCommandを実際に実行する側の契約である。
// 実装はL4（Plugin・Provider）に置き、このパッケージは境界だけを持つ。
type Executor interface {
	Execute(ctx context.Context, request Request) (Result, error)
}

// Authorizerは、Commandの実行前に呼出しの可否を決める契約である。
// 認可の正本や権限の付与は提供せず、L4の実装へ委ねる。
type Authorizer interface {
	Authorize(ctx context.Context, request Request) error
}

// ExecutionErrorは、業務上の結末ではなく、呼出しの前提・実行環境・契約の不備を表す。
type ExecutionError struct {
	Code    string
	Message string
	Cause   error
	// EffectUnknownは、呼出しが失敗してもデータ効果が適用された可能性がある
	// ことを示す。Commit失敗やRollback失敗など、適用有無を確定できない場合に
	// 立てる。Lifecycleはこの印を見てunknownとして記録し、再送を拒否する。
	EffectUnknown bool
}

func (e *ExecutionError) Error() string { return e.Code + ": " + e.Message }
func (e *ExecutionError) Unwrap() error { return e.Cause }

// NewUnknownExecutionErrorは、適用有無が確定しない実行失敗を作る。
// L4（Plugin・Provider）が部分適用の可能性を報告するために使う。
func NewUnknownExecutionError(code, message string, cause error) *ExecutionError {
	return &ExecutionError{Code: code, Message: message, Cause: cause, EffectUnknown: true}
}

// EffectMayBeUnknownは、失敗した呼出しのデータ効果が適用された可能性があるかを
// 返す。未知のerrorは「適用されていない」と決めつけない。
func EffectMayBeUnknown(err error) bool {
	var executionError *ExecutionError
	if !errors.As(err, &executionError) {
		return false
	}
	return executionError.EffectUnknown
}

// Invokerは、宣言とExecutorの間に立ち、入力の検証・実行・結末の照合・取引を行う。
type Invoker struct {
	World      ir.World
	Executor   Executor
	UnitOfWork UnitOfWork
	Periods    PeriodState
	Authorizer Authorizer
	Recorder   Recorder
}

// Invokeは、Commandを実行する。入力の不備、Executorの不在、実行の失敗、
// 宣言されていない結末、取引の失敗は、いずれもExecutionErrorとして返す。
func (i Invoker) Invoke(ctx context.Context, request Request) (Result, error) {
	plan, err := applicationview.BuildCommandPlan(i.World, request.Application, request.Pattern, request.Capability, request.Inputs)
	if err != nil {
		return Result{}, &ExecutionError{Code: "COMMAND_NOT_DECLARED", Message: err.Error(), Cause: err}
	}
	if !plan.Ready {
		message := "command input is not ready"
		if len(plan.Missing) > 0 {
			message = fmt.Sprintf("missing inputs: %v", plan.Missing)
		} else if len(plan.Invalid) > 0 {
			message = fmt.Sprintf("invalid input %s: %s", plan.Invalid[0].Input, plan.Invalid[0].Message)
		}
		return Result{}, &ExecutionError{Code: "INVALID_COMMAND_INPUT", Message: message}
	}
	if err := i.checkPeriod(ctx, plan); err != nil {
		return Result{}, err
	}
	if i.Authorizer != nil {
		if err := i.Authorizer.Authorize(ctx, request); err != nil {
			return Result{}, &ExecutionError{Code: "COMMAND_UNAUTHORIZED", Message: err.Error(), Cause: err}
		}
	}
	if i.Executor == nil {
		return Result{}, &ExecutionError{Code: "COMMAND_EXECUTOR_UNAVAILABLE", Message: "no executor is configured for this command boundary"}
	}
	if i.UnitOfWork == nil {
		return i.execute(ctx, request, plan)
	}
	transaction, err := i.UnitOfWork.Begin(ctx)
	if err != nil {
		return Result{}, &ExecutionError{Code: "COMMAND_TRANSACTION_BEGIN_FAILED", Message: err.Error(), Cause: err}
	}
	result, err := i.execute(ctx, request, plan)
	if err != nil {
		if rollbackErr := transaction.Rollback(ctx); rollbackErr != nil {
			return Result{}, &ExecutionError{
				Code:          "COMMAND_ROLLBACK_FAILED",
				Message:       fmt.Sprintf("%s (rollback also failed: %s)", err.Error(), rollbackErr.Error()),
				Cause:         err,
				EffectUnknown: true,
			}
		}
		return Result{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		_ = transaction.Rollback(ctx)
		return Result{}, &ExecutionError{
			Code:          "COMMAND_COMMIT_FAILED",
			Message:       err.Error(),
			Cause:         err,
			EffectUnknown: true,
		}
	}
	return result, nil
}

func (i Invoker) checkPeriod(ctx context.Context, plan applicationview.CommandPlan) error {
	if plan.Period == nil || i.Periods == nil || plan.Effect == nil {
		return nil
	}
	closed, err := i.Periods.Closed(ctx, plan.Period.ID)
	if err != nil {
		return &ExecutionError{Code: "COMMAND_PERIOD_STATE_FAILED", Message: err.Error(), Cause: err}
	}
	if !closed {
		return nil
	}
	for _, rejected := range plan.Period.ClosedRejects {
		if rejected == plan.Effect.Kind {
			return &ExecutionError{Code: "COMMAND_PERIOD_CLOSED", Message: fmt.Sprintf("period %s is closed for %s", plan.Period.ID, plan.Effect.Kind)}
		}
	}
	return nil
}

func (i Invoker) execute(ctx context.Context, request Request, plan applicationview.CommandPlan) (Result, error) {
	result, err := i.Executor.Execute(ctx, request)
	if err != nil {
		// A Provider that cannot tell whether its write landed reports an
		// unknown effect. Preserve that marker across the boundary so the
		// lifecycle does not record the call as a clean rejection.
		return Result{}, &ExecutionError{
			Code:          "COMMAND_EXECUTION_FAILED",
			Message:       err.Error(),
			Cause:         err,
			EffectUnknown: EffectMayBeUnknown(err),
		}
	}
	if result.Effect != nil {
		if plan.Effect == nil || string(result.Effect.Kind) != plan.Effect.Kind || result.Effect.Data != plan.Effect.Data {
			return Result{}, &ExecutionError{
				Code:    "UNDECLARED_COMMAND_EFFECT",
				Message: fmt.Sprintf("executor returned effect %s on %s that the capability does not declare", result.Effect.Kind, result.Effect.Data),
			}
		}
	}
	if result.Outcome != nil {
		declared := false
		for _, outcome := range plan.Outcomes {
			if outcome.Name == string(result.Outcome.Name) {
				declared = true
				break
			}
		}
		if !declared {
			return Result{}, &ExecutionError{
				Code:    "UNDECLARED_COMMAND_OUTCOME",
				Message: fmt.Sprintf("executor returned outcome %s that the capability does not declare", result.Outcome.Name),
			}
		}
	}
	if i.Recorder != nil {
		record := OperationRecord{Request: request, Reported: result.Effect, Outcome: result.Outcome}
		if plan.Effect != nil {
			record.Declared = &semantic.DataEffect{Kind: semantic.DataEffectKind(plan.Effect.Kind), Data: plan.Effect.Data}
		}
		if err := i.Recorder.Record(ctx, record); err != nil {
			return Result{}, &ExecutionError{Code: "COMMAND_RECORD_FAILED", Message: err.Error(), Cause: err}
		}
	}
	return result, nil
}
