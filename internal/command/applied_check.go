package command

import (
	"fmt"

	"tei/internal/applicationview"
	"tei/internal/core"
	"tei/internal/semantic"
)

// DifferenceCode names why an applied operation and a declared effect do not
// line up.
type DifferenceCode string

const (
	// DifferenceAdditionalScope is an operation of the declared kind on another
	// data. A cascade delete produces these: the declaration names the target,
	// and the relation policy reaches the related rows. The declaration does not
	// say how many rows it will reach, so the count stays unresolved.
	DifferenceAdditionalScope DifferenceCode = "additional-scope"
)

// Difference is one unresolved gap between what a capability declared and what
// an executor applied. It is kept as evidence rather than smoothed away: a
// reader can see that the declaration did not describe the whole effect.
type Difference struct {
	Code     DifferenceCode      `json:"code"`
	Declared semantic.DataEffect `json:"declared"`
	Applied  AppliedOperation    `json:"applied"`
	Message  string              `json:"message"`
}

// AppliedDiff holds the applied operations that the declared effect does not
// describe, grouped so a reader can count them without re-reading the list.
type AppliedDiff struct {
	Details []Difference `json:"details"`
}

// checkAppliedOperations compares the operations an executor reports as applied
// with the effect the capability declares.
//
// An operation of another kind than the declaration is refused, because the
// declaration and the execution disagree about what happened. An operation of
// the declared kind on another data is a legitimate consequence of a declared
// policy (a cascade), so it is kept as an unresolved difference rather than
// refused. A declaration with no matching operation is also refused, because
// then the declaration did not describe what the execution did.
func checkAppliedOperations(plan applicationview.CommandPlan, result Result) ([]Difference, error) {
	if len(result.Applied) == 0 {
		return nil, nil
	}
	if plan.Effect == nil {
		return nil, &ExecutionError{
			Code:    "UNDECLARED_COMMAND_EFFECT",
			Message: fmt.Sprintf("executor applied %s on %s but the capability declares no effect", result.Applied[0].Kind, result.Applied[0].Data),
		}
	}
	declared := semantic.DataEffect{Kind: semantic.DataEffectKind(plan.Effect.Kind), Data: plan.Effect.Data}
	var differences []Difference
	found := false
	for _, operation := range result.Applied {
		if operation.MatchesDeclared(&declared) {
			found = true
			continue
		}
		if operation.Kind == declared.Kind {
			differences = append(differences, Difference{
				Code:     DifferenceAdditionalScope,
				Declared: declared,
				Applied:  operation,
				Message:  fmt.Sprintf("capability declares %s on %s and the execution also applied it on %s", declared.Kind, declared.Data, operation.Data),
			})
			continue
		}
		return nil, &ExecutionError{
			Code:    "UNDECLARED_COMMAND_EFFECT",
			Message: fmt.Sprintf("executor applied %s on %s while the capability declares %s on %s", operation.Kind, operation.Data, declared.Kind, declared.Data),
		}
	}
	if !found {
		return nil, &ExecutionError{
			Code:    "MISSING_COMMAND_EFFECT",
			Message: fmt.Sprintf("executor reported no operation for the declared effect %s on %s", declared.Kind, declared.Data),
		}
	}
	return differences, nil
}

// ScopeCount counts the additional data an execution reached beyond the
// declared effect, so a record can say "declared one target, applied three".
func ScopeCount(differences []Difference) map[core.CanonicalID]int {
	if len(differences) == 0 {
		return nil
	}
	out := map[core.CanonicalID]int{}
	for _, difference := range differences {
		out[difference.Applied.Data]++
	}
	return out
}
