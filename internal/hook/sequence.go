package hook

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
)

// PhaseOrderは、フックの位相を実行する順序である。同じ Pattern に複数の位相が
// 宣言されたとき、この順に実行する。宣言順や名前の文字列順では決めない。
var PhaseOrder = []string{
	"before-read",
	"before-create",
	"after-create",
	"before-update",
	"after-update",
	"before-delete",
	"after-delete",
}

// SequenceResultは、Runnerが順に実行した位相の記録である。
// どの位相まで進んだかを残し、途中で止まった位置を呼出し側が確かめられるようにする。
type SequenceResult struct {
	// Executedは、成功した位相である。
	Executed []string `json:"executed,omitempty"`
	// Warnedは、`on-failure warn`で失敗したが続行した位相である。
	Warned []string `json:"warned,omitempty"`
	// Refusedは、不成立または実行の失敗で止まった位相である。
	Refused string `json:"refused,omitempty"`
}

// RunSequenceは、Patternに宣言されたフックを位相の順序（`PhaseOrder`）で実行する。
// `invocation.Phase`は使いません。`Reads`は各位相へそのまま渡します。
//
// 不成立（`NotSatisfied`）または実行の失敗に当たった位相で止まり、そこまでの
// 記録を返します。`on-failure warn`の位相は記録して先へ進みます。宣言の無い位相は
// 飛ばします。どの位相も宣言されていなければ`HOOK_NOT_DECLARED`を返します。
//
// 返すエラーは止まった位相の失敗そのものです。`SequenceResult`と併せて読むことで、
// 何を実行してどこで止まったかを区別できます。
func (r Runner) RunSequence(ctx context.Context, invocation Invocation) (SequenceResult, error) {
	pattern, err := r.pattern(invocation.Application, invocation.Pattern)
	if err != nil {
		return SequenceResult{}, err
	}
	declared := map[string]*application.Hook{}
	for i := range pattern.Interaction.Hooks {
		phase := pattern.Interaction.Hooks[i].Phase
		if !knownPhase(phase) {
			// A phase the order does not name would be skipped silently, leaving a
			// declaration that never runs. Refuse it instead.
			return SequenceResult{}, &Error{
				Code:    "HOOK_PHASE_UNSUPPORTED",
				Message: fmt.Sprintf("pattern %s declares unsupported hook phase %s", invocation.Pattern, phase),
			}
		}
		// A scope this runner cannot apply is refused, not ignored: the sequence
		// runs one hook per record, so only the record scope and the pattern's own
		// unit (empty) are applied here. A row-scoped hook needs the row set, which
		// the sequence does not carry, and is pointed at RunRows. Running the hook
		// anyway would leave a declaration that is present and does nothing (D-9).
		switch scope := pattern.Interaction.Hooks[i].Scope; scope {
		case "", string(application.PatternRecord):
		case "row":
			return SequenceResult{}, &Error{
				Code:    "HOOK_SCOPE_NEEDS_ROWS",
				Message: fmt.Sprintf("pattern %s hook %s declares scope row, so it needs the row set; use RunRows", invocation.Pattern, phase),
			}
		default:
			return SequenceResult{}, &Error{
				Code:    "HOOK_SCOPE_UNAPPLIED",
				Message: fmt.Sprintf("pattern %s hook %s declares scope %s but the sequence runs one hook per record", invocation.Pattern, phase, scope),
			}
		}
		// Two hooks with the same phase are refused by validation. The first
		// declaration wins here so the runner does not invent an order between
		// them, and it never silently drops a phase it did not run.
		if _, exists := declared[phase]; exists {
			continue
		}
		declared[phase] = &pattern.Interaction.Hooks[i]
	}
	var result SequenceResult
	for _, phase := range PhaseOrder {
		declaration, declaredHere := declared[phase]
		if !declaredHere {
			continue
		}
		invocation.Phase = phase
		// Each phase sees only the reads its own declaration names. The caller
		// supplies one value set for the pattern, so the sequence narrows it per
		// phase: a narrower phase must not inherit the wider view of another, and
		// a value no phase declares is simply not passed on. A declared read that
		// has no value is still refused by ReadsFor.
		phaseReads := map[core.CanonicalID]string{}
		for _, attribute := range declaration.Reads {
			if value, observed := invocation.Reads[attribute]; observed {
				phaseReads[attribute] = value
			}
		}
		reads, readErr := ReadsFor(declaration, phaseReads)
		if readErr != nil {
			result.Refused = phase
			return result, readErr
		}
		invocation.Reads = reads
		err := r.runDeclared(ctx, invocation, declaration)
		if err == nil {
			result.Executed = append(result.Executed, phase)
			continue
		}
		var warning *Warning
		if errors.As(err, &warning) {
			result.Warned = append(result.Warned, phase)
			continue
		}
		result.Refused = phase
		return result, err
	}
	if len(result.Executed) == 0 && len(result.Warned) == 0 && result.Refused == "" {
		return SequenceResult{}, &Error{Code: "HOOK_NOT_DECLARED", Message: fmt.Sprintf("pattern %s declares no hook", invocation.Pattern)}
	}
	return result, nil
}

// runDeclaredは、一つの宣言を実行する。`Run`と同じ境界と失敗の種類を使う。
func (r Runner) runDeclared(ctx context.Context, invocation Invocation, declared *application.Hook) error {
	implementation := r.implementationFor(invocation.Phase)
	if implementation == nil {
		return &Error{Code: "HOOK_IMPLEMENTATION_UNAVAILABLE", Message: "no implementation is configured for this hook boundary"}
	}
	var err error
	if calling, ok := implementation.(CallingImplementation); ok {
		err = calling.RunWithCaller(ctx, invocation, declaredCaller(r.Caller, declared))
	} else {
		err = implementation.Run(ctx, invocation)
	}
	return wrapFailure(err, declared)
}

// patternは、ApplicationとPatternの宣言を解決する。`Run`と同じ束縛を使う。
func (r Runner) pattern(applicationID, patternID core.CanonicalID) (*application.Pattern, error) {
	for i := range r.World.Applications {
		if r.World.Applications[i].ID != applicationID {
			continue
		}
		for j := range r.World.Applications[i].Patterns {
			pattern := &r.World.Applications[i].Patterns[j]
			if pattern.ID != patternID {
				continue
			}
			if pattern.Kind != application.PatternRecord || pattern.Interaction == nil {
				return nil, &Error{Code: "HOOK_PATTERN_NOT_RECORD", Message: fmt.Sprintf("pattern %s is not a record pattern with interaction", patternID)}
			}
			return pattern, nil
		}
	}
	return nil, &Error{Code: "HOOK_PATTERN_NOT_FOUND", Message: fmt.Sprintf("pattern %s is not defined in application %s", patternID, applicationID)}
}

// knownPhaseは、位相がPhaseOrderに載っているかを返す。
func knownPhase(phase string) bool {
	for _, known := range PhaseOrder {
		if known == phase {
			return true
		}
	}
	return false
}

// wrapFailureは、実装の失敗を、宣言に応じた種類へ揃える。
// 不成立は実装の答えとしてそのまま返し、`warn`は警告として包み、それ以外は拒否とする。
func wrapFailure(err error, declared *application.Hook) error {
	if err == nil {
		return nil
	}
	if isNotSatisfied(err) {
		// An unsatisfied hook may name a business outcome. The name is checked
		// here, where every refusal passes, so an implementation cannot widen
		// the vocabulary by reporting a name it invented.
		if outcome, ok := OutcomeOf(err); ok {
			if outcomeErr := ValidateOutcome(outcome); outcomeErr != nil {
				return outcomeErr
			}
		}
		return err
	}
	// A warning can be continued only when the failed hook did not leave an
	// unresolved effect. The called Command/Provider may have committed before
	// its acknowledgement was lost; continuing here would permit another effect.
	if declared != nil && declared.Failure == "warn" {
		if _, uncertain := uncertainFailure(err); !uncertain {
			return &Warning{Err: err}
		}
	}
	return &Error{Code: "HOOK_REJECTED", Message: err.Error(), Cause: err}
}
