package command

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/applicationview"
	"tei/internal/core"
	"tei/internal/failure"
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
	// OutcomeUndetermined states why the executor did not return an outcome
	// although the capability declares some. A capability may declare several
	// outcomes, and no declared rule selects one of them; an executor that
	// picked one anyway would report a decision nobody made. Nil Outcome with
	// this set means "not determined", which is different from "the capability
	// declares no outcome".
	OutcomeUndetermined string `json:"outcomeUndetermined,omitempty"`
	// AuthorizationMode names the authorization decision that let this
	// execution run. It is set by the Invoker, never by an Executor.
	AuthorizationMode AuthorizationMode `json:"authorizationMode,omitempty"`
	// Applied lists the data operations the executor actually applied. An
	// executor that cannot enumerate them leaves it nil, and the record then
	// holds only the declared and reported effects.
	Applied []AppliedOperation `json:"applied,omitempty"`
	// Persistence reports whether an independent observer saw the declared
	// effect after Commit. The Invoker sets it after a successful execution;
	// an Executor must leave it empty.
	Persistence PersistenceState `json:"persistence,omitempty"`
}

// Executorは、宣言されたCommandを実際に実行する側の契約である。
// 実装はL4（Plugin・Provider）に置き、このパッケージは境界だけを持つ。
//
// 取引への参加は契約に含まない。InvokerはUnitOfWorkがある場合にBegin・
// Commit・Rollbackを呼ぶが、Executorが同じ取引へ参加することは要求しない。
// そのため、Executorが成功した後にdeclaredEffectと異なる効果・結末・適用
// 操作を返した場合、巻き戻し済みとは断定できない。Invokerはそのような
// 契約違反をEffectUnknownの付いたExecutionErrorとして返し、
// LifecycleInvokerはunknownとして記録する（rejectedとしては記録しない）。
// 適用の有無を確定させるには、Executor側が同じ取引へ参加するか、実行後に
// 独立した観測で確かめる必要がある。参照実装の`ReferenceExecutor`は
// プロセス内のStoreを書き換えるだけで、外部の永続化を持たない。
type Executor interface {
	Execute(ctx context.Context, request Request) (Result, error)
}

// Authorizerは、Commandの実行前に呼出しの可否を決める契約である。
// 認可の正本や権限の付与は提供せず、L4の実装へ委ねる。
type Authorizer interface {
	Authorize(ctx context.Context, request Request) error
}

// ReferenceAuthorizer is an explicit "allow every request" authorizer. A
// reference execution that has no real identity or grant source must still say
// that it chose to run without authorization; a nil Authorizer is refused.
type ReferenceAuthorizer struct{}

// Authorize allows the request. It records no grant and evaluates no rule.
func (ReferenceAuthorizer) Authorize(context.Context, Request) error { return nil }

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

// ReportedFailure returns the machine-readable failure an entrance should
// report for err, and whether it has one.
//
// Every entrance answers the same way, because the answer is a property of the
// error, not of the entrance: an ExecutionError already carries the reason the
// caller branches on, and the remedy differs by code (supply an input, open the
// period, obtain authority, retry after a commit failure). A per-entrance
// allow-list is therefore wrong twice over — it hides reasons the core already
// made machine-readable, and the lists drift apart (#3807: the CLI flattened 15
// of 21 codes and knew seven, the Studio knew one).
//
// The second return value is the message without the "CODE: " prefix, so a
// caller that reads code and message separately does not see the code twice.
func ReportedFailure(err error) (string, string, bool) {
	// "The effect may already be applied" dominates the specific code, because
	// it changes what the caller must do: a blind retry can apply the change
	// twice. Any ExecutionError can carry the flag, and so can a wrapped error
	// whose cause does, so this is checked before the code is reported.
	if EffectMayBeUnknown(err) {
		return "COMMAND_EFFECT_UNKNOWN", err.Error(), true
	}
	var execution *ExecutionError
	if errors.As(err, &execution) {
		return execution.Code, execution.Message, true
	}
	return "", "", false
}
func (e *ExecutionError) Unwrap() error { return e.Cause }

// FailureCode returns the machine-readable failure code.
func (e *ExecutionError) FailureCode() string { return e.Code }

// FailureMessage returns the human-readable failure message.
func (e *ExecutionError) FailureMessage() string { return e.Message }

// FailureCause returns the underlying error, if any.
func (e *ExecutionError) FailureCause() error { return e.Cause }

// Kind classifies the business meaning of the failure.
func (e *ExecutionError) Kind() failure.Kind {
	switch e.Code {
	case "COMMAND_NOT_DECLARED",
		"INVALID_COMMAND_INPUT",
		"COMMAND_EXECUTOR_UNAVAILABLE",
		"COMMAND_AUTHORIZER_UNAVAILABLE",
		"COMMAND_UNAUTHORIZED",
		"COMMAND_PERIOD_CLOSED",
		"COMMAND_BUSINESS_DENIED",
		"COMMAND_RESOURCE_CONFLICT":
		return failure.BusinessRejection
	case "COMMAND_TRANSACTION_BEGIN_FAILED",
		"COMMAND_ROLLBACK_FAILED",
		"COMMAND_COMMIT_FAILED",
		"COMMAND_PERIOD_STATE_UNAVAILABLE",
		"COMMAND_PERIOD_SCOPE_UNKNOWN",
		"COMMAND_PERIOD_STATE_FAILED",
		"COMMAND_PERIOD_SCOPE_UNAVAILABLE",
		"COMMAND_EXECUTION_FAILED",
		"COMMAND_RECORD_FAILED",
		"UNDECLARED_COMMAND_EFFECT",
		"UNDECLARED_COMMAND_OUTCOME",
		"TIMEOUT":
		return failure.TechnicalFailure
	default:
		return failure.TechnicalFailure
	}
}

// Retryable reports whether the caller may safely retry the same operation
// identity after this failure.
func (e *ExecutionError) Retryable() failure.RetryPolicy {
	base := failure.NotRetryable
	switch e.Code {
	case "COMMAND_NOT_DECLARED",
		"INVALID_COMMAND_INPUT",
		"COMMAND_AUTHORIZER_UNAVAILABLE",
		"COMMAND_UNAUTHORIZED",
		"COMMAND_PERIOD_CLOSED",
		"COMMAND_PERIOD_STATE_UNAVAILABLE",
		"COMMAND_PERIOD_SCOPE_UNKNOWN",
		"COMMAND_PERIOD_STATE_FAILED",
		"COMMAND_PERIOD_SCOPE_UNAVAILABLE",
		"UNDECLARED_COMMAND_EFFECT",
		"UNDECLARED_COMMAND_OUTCOME",
		"COMMAND_BUSINESS_DENIED",
		"COMMAND_RESOURCE_CONFLICT":
		base = failure.NotRetryable
	case "COMMAND_EXECUTOR_UNAVAILABLE",
		"COMMAND_TRANSACTION_BEGIN_FAILED",
		"COMMAND_ROLLBACK_FAILED",
		"COMMAND_COMMIT_FAILED",
		"COMMAND_EXECUTION_FAILED",
		"COMMAND_RECORD_FAILED",
		"PROVIDER_PARTIAL",
		"TIMEOUT":
		base = failure.Retryable
	default:
		base = failure.NotRetryable
	}
	if base == failure.Retryable && e.EffectUnknown {
		return failure.LookupFirst
	}
	return base
}

// HasEffectUnknown reports whether the failure may have left partial effects
// applied.
func (e *ExecutionError) HasEffectUnknown() bool { return e.EffectUnknown }

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

// effectContractViolationは、Executorが実行された後に宣言から外れた結果を
// 返したことを表す。データ効果が適用された可能性があるため、常にunknownの印を
// 立てる。Executor契約はUnitOfWorkへの取引参加を要求していないので、参照用の
// UnitOfWorkがあっても巻き戻し済みとは断定できない。
func effectContractViolation(code, message string) *ExecutionError {
	return &ExecutionError{Code: code, Message: message, EffectUnknown: true}
}

// wrapTimeout wraps an ExecutionError with a TIMEOUT code if the underlying
// cause is a context deadline exceeded. It preserves the EffectUnknown flag.
func wrapTimeout(err error) error {
	var executionError *ExecutionError
	if errors.As(err, &executionError) {
		if errors.Is(executionError.Cause, context.DeadlineExceeded) {
			executionError.Code = "TIMEOUT"
			executionError.Message = "operation timed out: " + executionError.Message
			return executionError
		}
	}
	return err
}

// Invokerは、宣言とExecutorの間に立ち、入力の検証・実行・結末の照合・取引を行う。
type Invoker struct {
	World      ir.World
	Executor   Executor
	UnitOfWork UnitOfWork
	Periods    PeriodState
	Authorizer Authorizer
	Recorder   Recorder
	// PersistenceObserver, when set, performs an independent read after a
	// successful Commit. A nil observer leaves persistence as not-observed;
	// it never means "not persisted".
	PersistenceObserver PersistenceObserver
	// DurableStore is the composition's claim that this boundary has a durable
	// store behind it. The Invoker does not verify it. false makes persistence
	// not-applicable (a reference run), so "not observed" is not printed when
	// there is nothing that could have stored the effect.
	DurableStore bool
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
	if i.Executor == nil {
		return Result{}, &ExecutionError{Code: "COMMAND_EXECUTOR_UNAVAILABLE", Message: "no executor is configured for this command boundary"}
	}
	// Authorization is an explicit decision, not an optional decoration. A
	// reference execution must pass ReferenceAuthorizer and say so, so an
	// unconfigured Host cannot silently execute without a decision. Authorization
	// precedes the period check because the period state can disclose business
	// facts (whether a period is closed) to a caller that is not permitted to run
	// the command at all.
	if i.Authorizer == nil {
		return Result{}, &ExecutionError{Code: "COMMAND_AUTHORIZER_UNAVAILABLE", Message: "no authorizer is configured; command execution requires an explicit authorization decision"}
	}
	if err := i.Authorizer.Authorize(ctx, request); err != nil {
		return Result{}, &ExecutionError{Code: "COMMAND_UNAUTHORIZED", Message: err.Error(), Cause: err}
	}
	mode := authorizationModeOf(i.Authorizer)
	var plannedEffect *semantic.DataEffect
	if plan.Effect != nil {
		plannedEffect = &semantic.DataEffect{Kind: semantic.DataEffectKind(plan.Effect.Kind), Data: plan.Effect.Data}
	}
	if err := checkDatasourceAccess(i.World, request, plannedEffect); err != nil {
		return Result{}, err
	}
	if err := i.checkPeriod(ctx, plan); err != nil {
		return Result{}, err
	}
	if i.UnitOfWork == nil {
		result, err := i.execute(ctx, request, plan)
		if err != nil {
			return result, wrapTimeout(err)
		}
		result.Persistence = i.observePersistence(ctx, request, declaredEffect(plan))
		result.AuthorizationMode = mode
		return result, nil
	}
	transaction, err := i.UnitOfWork.Begin(ctx)
	if err != nil {
		return Result{}, wrapTimeout(&ExecutionError{Code: "COMMAND_TRANSACTION_BEGIN_FAILED", Message: err.Error(), Cause: err})
	}
	result, err := i.execute(ctx, request, plan)
	if err != nil {
		if rollbackErr := transaction.Rollback(ctx); rollbackErr != nil {
			return Result{}, wrapTimeout(&ExecutionError{
				Code:          "COMMAND_ROLLBACK_FAILED",
				Message:       fmt.Sprintf("%s (rollback also failed: %s)", err.Error(), rollbackErr.Error()),
				Cause:         err,
				EffectUnknown: true,
			})
		}
		return Result{}, wrapTimeout(err)
	}
	if err := transaction.Commit(ctx); err != nil {
		_ = transaction.Rollback(ctx)
		return Result{}, wrapTimeout(&ExecutionError{
			Code:          "COMMAND_COMMIT_FAILED",
			Message:       err.Error(),
			Cause:         err,
			EffectUnknown: true,
		})
	}
	result.Persistence = i.observePersistence(ctx, request, declaredEffect(plan))
	result.AuthorizationMode = mode
	return result, nil
}

// declaredEffect projects the plan's declared effect for the persistence
// observer. It returns nil when the capability declares no effect.
func declaredEffect(plan applicationview.CommandPlan) *semantic.DataEffect {
	if plan.Effect == nil {
		return nil
	}
	return &semantic.DataEffect{Kind: semantic.DataEffectKind(plan.Effect.Kind), Data: plan.Effect.Data}
}

func (i Invoker) checkPeriod(ctx context.Context, plan applicationview.CommandPlan) error {
	if plan.Period == nil {
		return nil
	}
	if i.Periods == nil {
		return &ExecutionError{Code: "COMMAND_PERIOD_STATE_UNAVAILABLE", Message: fmt.Sprintf("period %s is declared but no period state is configured", plan.Period.ID)}
	}
	if plan.Effect == nil {
		return nil
	}
	scope, scopeErr := applicationview.ScopeFor(*plan.Period, businessDateOf(plan, *plan.Period))
	if scopeErr != nil {
		return &ExecutionError{Code: "COMMAND_PERIOD_SCOPE_UNKNOWN", Message: scopeErr.Error(), Cause: scopeErr}
	}
	closed, answered, err := scopeClosed(ctx, i.Periods, scope)
	if err != nil {
		return &ExecutionError{Code: "COMMAND_PERIOD_STATE_FAILED", Message: err.Error(), Cause: err}
	}
	if !answered {
		// A period split by a closing day cannot be answered by a state that only
		// knows whole months: reading the month's closure as the half's closure
		// would refuse or allow the wrong half.
		return &ExecutionError{Code: "COMMAND_PERIOD_SCOPE_UNAVAILABLE", Message: fmt.Sprintf("period %s is split by a closing day but the period state cannot answer for a half month", plan.Period.ID)}
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

// businessDateOf finds the input that fills the period's business date, so the
// scope follows the date the caller actually supplied rather than the plan's
// declaration.
func businessDateOf(plan applicationview.CommandPlan, period applicationview.CommandPeriodRef) string {
	for _, input := range plan.Inputs {
		if input.Attribute == period.BusinessDate {
			return input.Value
		}
	}
	return ""
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
			return Result{}, effectContractViolation(
				"UNDECLARED_COMMAND_EFFECT",
				fmt.Sprintf("executor returned effect %s on %s that the capability does not declare", result.Effect.Kind, result.Effect.Data),
			)
		}
	}
	// The executor has already run when applied operations are checked. Even
	// with a UnitOfWork, the Executor contract does not require it to join that
	// transaction, so a mismatch cannot be recorded as a clean rejection.
	differences, err := checkAppliedOperations(plan, result)
	if err != nil {
		var contractError *ExecutionError
		if errors.As(err, &contractError) {
			return Result{}, effectContractViolation(contractError.Code, contractError.Message)
		}
		return Result{}, effectContractViolation("UNDECLARED_COMMAND_EFFECT", err.Error())
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
			return Result{}, effectContractViolation(
				"UNDECLARED_COMMAND_OUTCOME",
				fmt.Sprintf("executor returned outcome %s that the capability does not declare", result.Outcome.Name),
			)
		}
	}
	if i.Recorder != nil {
		record := OperationRecord{
			Request:               request,
			Reported:              result.Effect,
			Outcome:               result.Outcome,
			OutcomeUndetermined:   result.OutcomeUndetermined,
			Applied:               result.Applied,
			Differences:           differences,
			ReportedEffectMissing: plan.Effect != nil && result.Effect == nil,
			// A command that declares no effect has nothing to compare, so an
			// empty Applied list is not an unobserved difference check. Only a
			// declared effect makes the empty list "the executor could not say".
			AppliedUnobserved: plan.Effect != nil && len(result.Applied) == 0,
		}
		if plan.Effect != nil {
			record.Declared = &semantic.DataEffect{Kind: semantic.DataEffectKind(plan.Effect.Kind), Data: plan.Effect.Data}
		}
		if err := i.Recorder.Record(ctx, record); err != nil {
			// The executor has already returned. Its effects need not share our
			// UnitOfWork transaction; a failed operation record cannot prove
			// the provider did nothing, even when rollback returns nil.
			return Result{}, NewUnknownExecutionError("COMMAND_RECORD_FAILED", err.Error(), err)
		}
	}
	return result, nil
}
