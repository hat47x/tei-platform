package xeadsource

import "fmt"

// validateCrossViewSourceCounts checks source-observation coverage independently
// of the projection fingerprint. It does not claim semantic acceptance or
// infer runtime effects from structural counts.
func validateCrossViewSourceCounts(p CrossViewProjection) error {
	actions := make(map[string]CrossViewActionObservation, len(p.Actions))
	for _, action := range p.Actions {
		if _, duplicate := actions[action.ActionToken]; duplicate {
			return fmt.Errorf("cross-view source accounting has duplicate action observations")
		}
		if action.ManualOrExternalCandidate != (action.FunctionIOCount == 0) {
			return fmt.Errorf("cross-view source accounting has contradictory manual action status")
		}
		actions[action.ActionToken] = action
	}

	taskActions := map[string]int{}
	functionIOByAction := map[string]int{}
	functionIOCount := 0
	ioTableCount := 0
	for _, edge := range p.SourceEdges {
		switch edge.Kind {
		case "task-action":
			obs, exists := actions[edge.ToToken]
			if !exists || obs.TaskToken != edge.FromToken {
				return fmt.Errorf("cross-view source accounting has unmatched task action")
			}
			taskActions[edge.ToToken]++
		case "action-function-use":
			if _, exists := actions[edge.FromToken]; !exists {
				return fmt.Errorf("cross-view source accounting has function use without action")
			}
			functionIOCount++
			functionIOByAction[edge.FromToken]++
		case "function-table-operation":
			ioTableCount++
		}
	}

	orphanActions := map[string]int{}
	for _, unresolved := range p.Unresolved {
		if !crossViewUnresolvedRelationMatches(unresolved.Code, unresolved.RelationKind) {
			return fmt.Errorf("cross-view source accounting has contradictory unresolved relation")
		}
		switch unresolved.RelationKind {
		case "task-action":
			if len(unresolved.SourceTokens) != 1 {
				return fmt.Errorf("cross-view source accounting has malformed orphan action evidence")
			}
			orphanActions[unresolved.SourceTokens[0]]++
		case "action-function-use":
			functionIOCount++
			if unresolved.Code == "task-function-io-without-task-action" {
				continue
			}
			owner := ""
			for _, token := range unresolved.SourceTokens {
				if _, exists := actions[token]; exists {
					if owner != "" {
						return fmt.Errorf("cross-view source accounting has ambiguous unresolved action owner")
					}
					owner = token
				}
			}
			if owner == "" {
				return fmt.Errorf("cross-view source accounting has unresolved function use without its action")
			}
			functionIOByAction[owner]++
		case "function-table-operation":
			ioTableCount++
		}
	}

	distinctTasks := map[string]struct{}{}
	for _, action := range p.Actions {
		if action.TaskToken == "" {
			if taskActions[action.ActionToken] != 0 || orphanActions[action.ActionToken] != 1 {
				return fmt.Errorf("cross-view source accounting has incomplete orphan action evidence")
			}
		} else {
			if taskActions[action.ActionToken] != 1 || orphanActions[action.ActionToken] != 0 {
				return fmt.Errorf("cross-view source accounting has incomplete task action evidence")
			}
			distinctTasks[action.TaskToken] = struct{}{}
		}
		if functionIOByAction[action.ActionToken] != action.FunctionIOCount {
			return fmt.Errorf("cross-view source accounting has contradictory per-action function I/O count")
		}
	}
	if len(distinctTasks) > p.TaskCount || functionIOCount != p.TaskFunctionIOCount ||
		ioTableCount != p.IOTableCount {
		return fmt.Errorf("cross-view source accounting does not match its declared source totals")
	}
	return nil
}

func crossViewUnresolvedRelationMatches(code, relation string) bool {
	switch code {
	case "task-action-without-task":
		return relation == "task-action"
	case "task-function-io-without-task-action",
		"task-function-io-function-id-missing",
		"task-function-io-function-not-found",
		"task-function-io-function-ambiguous":
		return relation == "action-function-use"
	case "task-function-io-target-id-missing",
		"task-function-io-target-not-found",
		"task-function-io-target-ambiguous":
		return relation == "task-function-io-target"
	case "iotable-without-function",
		"iotable-table-id-missing",
		"iotable-table-not-found",
		"iotable-table-ambiguous",
		"iotable-operation-marker-unsupported",
		"iotable-operation-missing":
		return relation == "function-table-operation"
	default:
		return false
	}
}
