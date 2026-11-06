package command

import (
	"fmt"

	"tei/internal/applicationview"
	"tei/internal/semantic"
)

// checkAppliedOperations compares the operations an executor reports as
// applied with the effect the capability declares. The declared effect must
// appear among them. A cascade delete legitimately applies the declared kind to
// related data, so additional operations are accepted only when their kind is
// the declared kind. Any other operation is undeclared, and a declared effect
// with no applied operation is a difference worth reporting.
func checkAppliedOperations(plan applicationview.CommandPlan, result Result) error {
	if len(result.Applied) == 0 {
		return nil
	}
	if plan.Effect == nil {
		return &ExecutionError{
			Code:    "UNDECLARED_COMMAND_EFFECT",
			Message: fmt.Sprintf("executor applied %s on %s but the capability declares no effect", result.Applied[0].Kind, result.Applied[0].Data),
		}
	}
	declared := semantic.DataEffect{Kind: semantic.DataEffectKind(plan.Effect.Kind), Data: plan.Effect.Data}
	found := false
	for _, operation := range result.Applied {
		if operation.MatchesDeclared(&declared) {
			found = true
			continue
		}
		if operation.Kind == declared.Kind {
			continue
		}
		return &ExecutionError{
			Code:    "UNDECLARED_COMMAND_EFFECT",
			Message: fmt.Sprintf("executor applied %s on %s while the capability declares %s on %s", operation.Kind, operation.Data, declared.Kind, declared.Data),
		}
	}
	if !found {
		return &ExecutionError{
			Code:    "MISSING_COMMAND_EFFECT",
			Message: fmt.Sprintf("executor reported no operation for the declared effect %s on %s", declared.Kind, declared.Data),
		}
	}
	return nil
}
