package hook

import (
	"context"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// CallOutcome is the record of one capability a hook declared in `uses` and the
// boundary then called.
//
// A hook's `uses` is a promise that the capability is called. Until this type
// existed, nothing called it: the hosts ran the hook with no caller configured
// and reported the phase as executed. The record keeps the call, its refusal,
// and its result visible beside the phase.
type CallOutcome struct {
	Capability core.CanonicalID `json:"capability"`
	// Called is true when the boundary was reached. A false value means the
	// call did not happen, and Reason says why.
	Called bool `json:"called"`
	// Reason states why a declared capability was not called.
	Reason string `json:"reason,omitempty"`
	// Error is the boundary's refusal or failure, when it returned one.
	Error string `json:"error,omitempty"`
}

// ReasonNoCaller は、呼出し元が設定されていないため宣言した能力を呼べなかった
// ことを示す理由である。
const ReasonNoCaller = "no caller is configured for this hook boundary, so the declared capability was not called"

// ReasonNoInputs は、能力へ渡す入力を決められなかったことを示す理由である。
const ReasonNoInputs = "the hook declaration does not bind inputs for the declared capability, so the call was not made"

// RunDeclaredCalls は、フックが `uses` で宣言した能力を、呼出し元を通して呼ぶ。
//
// 呼出し元は宣言で制限済み（`Declared`）なので、宣言外の能力へは届きません。
// 入力は呼出し側が与えます。宣言が能力への結び付けを持たない場合は、推測して
// 呼ぶのではなく、理由つきで「呼ばなかった」と返します。
//
// 戻り値は呼出しごとの記録である。一件でも拒否または失敗があれば、最初のそれを
// エラーとして返します。呼ばなかった呼出しはエラーではありません（記録に残して
// 先へ進みます）。理由つきで残すためです。
func (r Runner) RunDeclaredCalls(ctx context.Context, applicationID, patternID core.CanonicalID, phase string, inputs map[core.CanonicalID]map[string]string, caller Caller) ([]CallOutcome, error) {
	pattern, err := r.findPatternForCalls(applicationID, patternID)
	if err != nil {
		return nil, err
	}
	declared := declarationForPhase(pattern, phase)
	if declared == nil {
		return nil, &Error{Code: "HOOK_NOT_DECLARED", Message: fmt.Sprintf("pattern %s does not declare a %s hook", patternID, phase)}
	}
	if len(declared.Uses) == 0 {
		return nil, nil
	}
	restricted := Declared{Inner: caller, Hook: declared}
	outcomes := make([]CallOutcome, 0, len(declared.Uses))
	var firstErr error
	for _, capability := range declared.Uses {
		outcome := CallOutcome{Capability: capability}
		bound, ok := inputs[capability]
		if !ok {
			outcome.Reason = ReasonNoInputs
			outcomes = append(outcomes, outcome)
			continue
		}
		outcome.Called = true
		_, callErr := restricted.Call(ctx, capability, bound)
		if callErr != nil {
			outcome.Error = callErr.Error()
			if firstErr == nil {
				firstErr = callErr
			}
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes, firstErr
}

// findPatternForCalls resolves the pattern a sequence names, without requiring
// it to be a record pattern with an interaction: a caller that already knows the
// pattern should not be told the declaration is missing when the pattern is
// simply of another kind. The hook declaration, not the kind, is what this path
// needs.
func (r Runner) findPatternForCalls(applicationID, patternID core.CanonicalID) (*application.Pattern, error) {
	for i := range r.World.Applications {
		if r.World.Applications[i].ID != applicationID {
			continue
		}
		for j := range r.World.Applications[i].Patterns {
			if r.World.Applications[i].Patterns[j].ID == patternID {
				return &r.World.Applications[i].Patterns[j], nil
			}
		}
	}
	return nil, &Error{Code: "HOOK_PATTERN_NOT_FOUND", Message: fmt.Sprintf("pattern %s is not defined in application %s", patternID, applicationID)}
}

// declarationForPhase returns the hook declaration of a phase, or nil.
func declarationForPhase(pattern *application.Pattern, phase string) *application.Hook {
	if pattern == nil || pattern.Interaction == nil {
		return nil
	}
	for i := range pattern.Interaction.Hooks {
		if pattern.Interaction.Hooks[i].Phase == phase {
			return &pattern.Interaction.Hooks[i]
		}
	}
	return nil
}

// DeclaredCalls resolves the command pattern for each capability a hook
// declares, so a host can supply the binding `RunDeclaredCalls` needs without
// guessing the target.
//
// The command pattern is resolved through `CommandPatternFor`, which refuses an
// undeclared or ambiguous capability.
//
// The inputs a hook passes are derived from its own `reads` by the binding the
// declaration already carries: the record pattern's `command <capability> {
// input <name> <attribute> }` says which attribute feeds which capability
// parameter. A hook read of that attribute therefore supplies that parameter.
// Nothing is invented here — a parameter whose attribute the hook does not read
// is simply absent, and the call is reported as not made rather than called with
// a guessed value.
func DeclaredCalls(world ir.World, applicationID, patternID core.CanonicalID, phase string, reads map[core.CanonicalID]string) (map[core.CanonicalID]map[string]string, error) {
	pattern := findPhasePattern(world, applicationID, patternID)
	declared := declarationForPhase(pattern, phase)
	if declared == nil {
		return nil, &Error{Code: "HOOK_NOT_DECLARED", Message: fmt.Sprintf("pattern %s does not declare a %s hook", patternID, phase)}
	}
	bound := make(map[core.CanonicalID]map[string]string, len(declared.Uses))
	for _, capability := range declared.Uses {
		// The command pattern is resolved so an undeclared or ambiguous
		// capability is refused before any call is attempted. Its identity is
		// not what carries the binding: a command pattern has no interaction,
		// so the binding lives on this pattern's own interaction command.
		if _, err := CommandPatternFor(world, applicationID, capability); err != nil {
			return nil, err
		}
		bound[capability] = BindInputsFromReads(pattern, capability, reads)
	}
	return bound, nil
}

// BindInputsFromReads maps a hook's reads to the capability parameters that the
// pattern's interaction command binds.
//
// A binding `input <name> <attribute>` inside `command <capability> { ... }`
// states the correspondence: the parameter <name> is fed by the attribute
// <attribute>. A hook read of that attribute supplies that parameter. A
// parameter whose attribute the hook does not read is left out, so the caller
// can report a missing value instead of passing the wrong one.
func BindInputsFromReads(pattern *application.Pattern, capability core.CanonicalID, reads map[core.CanonicalID]string) map[string]string {
	if pattern == nil || pattern.Interaction == nil {
		return nil
	}
	bindings := map[string]core.CanonicalID{}
	for _, command := range pattern.Interaction.Commands {
		if command.Capability != capability {
			continue
		}
		for _, binding := range command.Inputs {
			if _, exists := bindings[binding.Name]; exists {
				continue
			}
			bindings[binding.Name] = binding.Attribute
		}
	}
	if len(bindings) == 0 {
		return nil
	}
	bound := make(map[string]string, len(bindings))
	for name, attribute := range bindings {
		value, observed := reads[attribute]
		if !observed {
			continue
		}
		bound[name] = value
	}
	if len(bound) == 0 {
		return nil
	}
	return bound
}

// findPhasePattern returns the pattern without an error, for the paths that only
// need to read a declaration.
func findPhasePattern(world ir.World, applicationID, patternID core.CanonicalID) *application.Pattern {
	for i := range world.Applications {
		if world.Applications[i].ID != applicationID {
			continue
		}
		for j := range world.Applications[i].Patterns {
			if world.Applications[i].Patterns[j].ID == patternID {
				return &world.Applications[i].Patterns[j]
			}
		}
	}
	return nil
}
