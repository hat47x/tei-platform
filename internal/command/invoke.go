package command

import (
	"context"
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

// ExecutionErrorは、業務上の結末ではなく、呼出しの前提・実行環境・契約の不備を表す。
type ExecutionError struct {
	Code    string
	Message string
	Cause   error
}

func (e *ExecutionError) Error() string { return e.Code + ": " + e.Message }
func (e *ExecutionError) Unwrap() error { return e.Cause }

// Invokerは、宣言とExecutorの間に立ち、入力の検証・実行・結末の照合・取引を行う。
type Invoker struct {
	World      ir.World
	Executor   Executor
	UnitOfWork UnitOfWork
	Periods    PeriodState
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
		_ = transaction.Rollback(ctx)
		return Result{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		_ = transaction.Rollback(ctx)
		return Result{}, &ExecutionError{Code: "COMMAND_COMMIT_FAILED", Message: err.Error(), Cause: err}
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
		return Result{}, &ExecutionError{Code: "COMMAND_EXECUTION_FAILED", Message: err.Error(), Cause: err}
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
	return result, nil
}
