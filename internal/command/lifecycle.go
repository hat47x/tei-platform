package command

import (
	"context"
	"fmt"
)

// Stageは、Command実行の1回分の段階である（#2564 → #1129）。
//
// 実行はdispatchされ、その後observeされるかunknownになる。unknownは
// 「適用されたか分からない」状態であり、成功にも取消にも丸めない。
type Stage string

const (
	// StageDispatchedは、実行を送信したが結果を観測していない状態である。
	StageDispatched Stage = "dispatched"
	// StageObservedは、業務上の結末または明確な拒否を観測した状態である。
	StageObserved Stage = "observed"
	// StageUnknownは、適用有無が確定しない状態である。
	StageUnknown Stage = "unknown"
)

// 観測recordに残す結末の名前である。個別の業務Outcomeがある場合はその名前を使う。
const (
	OutcomeCommitted = "committed"
	OutcomeRejected  = "rejected"
)

// Attemptは、Commandの1回の実行試行である。
type Attempt struct {
	CommandID    string `json:"commandId"`
	Owner        string `json:"owner"`
	Generation   int    `json:"generation"`
	Stage        Stage  `json:"stage"`
	Outcome      string `json:"outcome,omitempty"`
	Reason       string `json:"reason,omitempty"`
	TakeoverFrom int    `json:"takeoverFrom,omitempty"`
}

// Lifecycleは、Commandごとの最新Attemptを保持し、unknownの再送を拒否する。
//
// この型は実行段階の規則だけを持ち、永続化・監査・署名は持たない。
// 保存先と保持期間は本番ポリシーとして別に決める（#2845）。
type Lifecycle struct {
	attempts map[string]Attempt
}

// NewLifecycleは、空のLifecycleを返す。
func NewLifecycle() *Lifecycle {
	return &Lifecycle{attempts: map[string]Attempt{}}
}

// LifecycleFromAttemptsは、永続化されたAttemptからin-memoryの段階を組み立てる。
// durable storeが同じ規則を使うための入口であり、規則を複製しない。
func LifecycleFromAttempts(attempts []Attempt) *Lifecycle {
	lifecycle := NewLifecycle()
	for _, attempt := range attempts {
		if attempt.CommandID != "" {
			lifecycle.attempts[attempt.CommandID] = attempt
		}
	}
	return lifecycle
}

// Attemptは、Commandの最新Attemptを返す。
func (l *Lifecycle) Attempt(commandID string) (Attempt, bool) {
	attempt, ok := l.attempts[commandID]
	return attempt, ok
}

// Attemptsは、保持している全Attemptを返す（観測recordの参照用）。
func (l *Lifecycle) Attempts() []Attempt {
	attempts := make([]Attempt, 0, len(l.attempts))
	for _, attempt := range l.attempts {
		attempts = append(attempts, attempt)
	}
	return attempts
}

// Dispatchは、新しい試行を記録する。
//
// observedのCommandは再送できない。unknownのCommandも、同じowner/generation
// では再送できない（Takeoverが必要）。dispatchedのCommandは二重送信になるため
// 拒否する。
func (l *Lifecycle) Dispatch(commandID, owner string, generation int) (Attempt, error) {
	if commandID == "" || owner == "" {
		return Attempt{}, fmt.Errorf("command lifecycle: command and owner are required")
	}
	if generation < 1 {
		return Attempt{}, fmt.Errorf("command lifecycle: generation must be positive")
	}
	if existing, ok := l.attempts[commandID]; ok {
		switch existing.Stage {
		case StageObserved:
			return Attempt{}, fmt.Errorf("command lifecycle: command %q is already observed; do not re-dispatch", commandID)
		case StageDispatched:
			return Attempt{}, fmt.Errorf("command lifecycle: command %q is already in flight (%s/%d)", commandID, existing.Owner, existing.Generation)
		case StageUnknown:
			return Attempt{}, fmt.Errorf("command lifecycle: command %q has an unknown outcome; replay requires a takeover with a higher generation", commandID)
		}
	}
	attempt := Attempt{CommandID: commandID, Owner: owner, Generation: generation, Stage: StageDispatched}
	l.attempts[commandID] = attempt
	return attempt, nil
}

// Takeoverは、unknownのCommandを新しいownerとより大きいgenerationで引き継ぐ。
//
// Takeover自体は成功ではない。段階はdispatchedのままで、Observeして初めて
// 結末を記録する。
func (l *Lifecycle) Takeover(commandID, owner string, generation int) (Attempt, error) {
	if commandID == "" || owner == "" {
		return Attempt{}, fmt.Errorf("command lifecycle: command and owner are required")
	}
	existing, ok := l.attempts[commandID]
	if !ok {
		return Attempt{}, fmt.Errorf("command lifecycle: command %q was not dispatched", commandID)
	}
	if existing.Stage != StageUnknown {
		return Attempt{}, fmt.Errorf("command lifecycle: takeover requires an unknown outcome, command %q is %s", commandID, existing.Stage)
	}
	if generation <= existing.Generation {
		return Attempt{}, fmt.Errorf("command lifecycle: takeover generation %d must exceed the current generation %d", generation, existing.Generation)
	}
	if owner == existing.Owner {
		return Attempt{}, fmt.Errorf("command lifecycle: takeover requires a new owner, command %q is still owned by %q", commandID, owner)
	}
	attempt := Attempt{
		CommandID:    commandID,
		Owner:        owner,
		Generation:   generation,
		Stage:        StageDispatched,
		TakeoverFrom: existing.Generation,
	}
	l.attempts[commandID] = attempt
	return attempt, nil
}

// Observeは、現在の試行の結末を記録する。ownerとgenerationは送信時と一致する
// 必要がある。observedのCommandは再度observeできない。
func (l *Lifecycle) Observe(commandID, owner string, generation int, outcome string) error {
	existing, ok := l.attempts[commandID]
	if !ok {
		return fmt.Errorf("command lifecycle: command %q was not dispatched", commandID)
	}
	if existing.Stage != StageDispatched {
		return fmt.Errorf("command lifecycle: command %q is %s, not dispatched", commandID, existing.Stage)
	}
	if existing.Owner != owner || existing.Generation != generation {
		return fmt.Errorf("command lifecycle: observation owner/generation does not match the dispatched attempt")
	}
	if outcome == "" {
		return fmt.Errorf("command lifecycle: outcome is required")
	}
	existing.Stage = StageObserved
	existing.Outcome = outcome
	l.attempts[commandID] = existing
	return nil
}

// MarkUnknownは、適用有無が確定しない試行を記録する。
func (l *Lifecycle) MarkUnknown(commandID, owner string, generation int, reason string) error {
	existing, ok := l.attempts[commandID]
	if !ok {
		return fmt.Errorf("command lifecycle: command %q was not dispatched", commandID)
	}
	if existing.Stage != StageDispatched {
		return fmt.Errorf("command lifecycle: command %q is %s, not dispatched", commandID, existing.Stage)
	}
	if existing.Owner != owner || existing.Generation != generation {
		return fmt.Errorf("command lifecycle: unknown marker owner/generation does not match the dispatched attempt")
	}
	existing.Stage = StageUnknown
	existing.Reason = reason
	l.attempts[commandID] = existing
	return nil
}

// MayReplayは、同じcommandを新たにDispatchできるかを返す。
// dispatchedは既に実行中、unknownは不明な副作用があり、observedは
// 結末を記録済みのため、いずれも再送できない。
func (l *Lifecycle) MayReplay(commandID string) bool {
	_, alreadyDispatched := l.attempts[commandID]
	return !alreadyDispatched
}

// StageLifecycleは、Command試行の段階を保持する境界である。in-memoryの
// Lifecycleと、再起動をまたぐdurable storeの両方が実装する。
type StageLifecycle interface {
	Attempt(commandID string) (Attempt, bool)
	Dispatch(commandID, owner string, generation int) (Attempt, error)
	Takeover(commandID, owner string, generation int) (Attempt, error)
	Observe(commandID, owner string, generation int, outcome string) error
	MarkUnknown(commandID, owner string, generation int, reason string) error
	MayReplay(commandID string) bool
}

// LifecycleInvokerは、Invokerの実行をLifecycleの段階へ接続する（#1129）。
type LifecycleInvoker struct {
	Invoker   Invoker
	Lifecycle StageLifecycle
}

// DispatchOptionsは、1回の試行を識別する。
type DispatchOptions struct {
	CommandID  string
	Owner      string
	Generation int
	// Takeoverは、unknownのCommandを引き継ぐ場合にtrueにする。
	Takeover bool
}

// Invokeは、Commandを実行し、送信・観測・不明の段階を記録する。
//
// 適用有無が確定しないExecutionError（commit失敗など）はunknownとして記録し、
// 同じowner/generationの再送を拒否する。TakeoverはObserveを経るまで成功に
// ならない。
func (l LifecycleInvoker) Invoke(ctx context.Context, request Request, options DispatchOptions) (Result, error) {
	if l.Lifecycle == nil {
		return Result{}, &ExecutionError{
			Code:    "COMMAND_LIFECYCLE_UNAVAILABLE",
			Message: "no command lifecycle is configured for this command boundary",
		}
	}
	var err error
	if options.Takeover {
		_, err = l.Lifecycle.Takeover(options.CommandID, options.Owner, options.Generation)
	} else {
		_, err = l.Lifecycle.Dispatch(options.CommandID, options.Owner, options.Generation)
	}
	if err != nil {
		return Result{}, &ExecutionError{
			Code:    "COMMAND_LIFECYCLE_REFUSED",
			Message: err.Error(),
			Cause:   err,
		}
	}

	result, invokeErr := l.Invoker.Invoke(ctx, request)
	if invokeErr != nil {
		if EffectMayBeUnknown(invokeErr) {
			if markErr := l.Lifecycle.MarkUnknown(options.CommandID, options.Owner, options.Generation, invokeErr.Error()); markErr != nil {
				return result, lifecycleRecordError(markErr, invokeErr)
			}
			return result, invokeErr
		}
		if observeErr := l.Lifecycle.Observe(options.CommandID, options.Owner, options.Generation, OutcomeRejected); observeErr != nil {
			return result, lifecycleRecordError(observeErr, invokeErr)
		}
		return result, invokeErr
	}

	outcome := OutcomeCommitted
	if result.Outcome != nil {
		outcome = string(result.Outcome.Name)
	}
	if observeErr := l.Lifecycle.Observe(options.CommandID, options.Owner, options.Generation, outcome); observeErr != nil {
		return result, lifecycleRecordError(observeErr, nil)
	}
	return result, nil
}

// lifecycleRecordErrorは、業務結果と段階記録の食い違いを表す。
// Executorが効果を適用した後で段階の記録だけが失敗した可能性があるため、
// clean rejectionへ変換せずunknownを呼出し元へ伝える。
func lifecycleRecordError(recordErr, cause error) error {
	var inner error
	if cause != nil {
		inner = cause
	}
	return &ExecutionError{
		Code:          "COMMAND_LIFECYCLE_RECORD_FAILED",
		Message:       recordErr.Error(),
		Cause:         inner,
		EffectUnknown: true,
	}
}
