package xeadsource

import "fmt"

// validateCrossViewIOTargetEvidence checks that a resolved target, or a
// target-resolution failure, belongs to exactly one source-backed FunctionIO.
// Missing target evidence is allowed only to the extent supported by the
// aggregate IOID=0 sentinel count. This cannot independently reconstruct
// the exact IOID of a source node that was not retained in the projection.
func validateCrossViewIOTargetEvidence(p CrossViewProjection) error {
	functionByIO := map[string]string{}
	for _, edge := range p.SourceEdges {
		if edge.Kind != "action-function-use" {
			continue
		}
		if len(edge.SourceTokens) != 3 {
			return fmt.Errorf("cross-view target evidence requires an action/function/IO source triple")
		}
		var ioToken string
		for _, token := range edge.SourceTokens {
			if token == edge.FromToken || token == edge.ToToken {
				continue
			}
			if ioToken != "" {
				return fmt.Errorf("cross-view target evidence has ambiguous FunctionIO source token")
			}
			ioToken = token
		}
		if ioToken == "" {
			return fmt.Errorf("cross-view target evidence lacks a FunctionIO source token")
		}
		if _, duplicate := functionByIO[ioToken]; duplicate {
			return fmt.Errorf("cross-view target evidence reuses a FunctionIO source token")
		}
		functionByIO[ioToken] = edge.ToToken
	}

	terminalByIO := map[string]struct{}{}
	for _, edge := range p.SourceEdges {
		if edge.Kind != "task-function-io-target" {
			continue
		}
		function, exists := functionByIO[edge.FromToken]
		if !exists {
			return fmt.Errorf("cross-view target has no source-backed FunctionIO use")
		}
		if !equalCrossViewTokens(edge.SourceTokens, normalizedTokens(edge.FromToken, function, edge.ToToken)) {
			return fmt.Errorf("cross-view target has incomplete FunctionIO/function/target evidence")
		}
		if _, duplicate := terminalByIO[edge.FromToken]; duplicate {
			return fmt.Errorf("cross-view FunctionIO has multiple target outcomes")
		}
		terminalByIO[edge.FromToken] = struct{}{}
	}
	for _, unresolved := range p.Unresolved {
		if unresolved.RelationKind != "task-function-io-target" {
			continue
		}
		if len(unresolved.SourceTokens) != 2 {
			return fmt.Errorf("cross-view unresolved target has invalid FunctionIO evidence")
		}
		ioToken, functionToken := "", ""
		for _, token := range unresolved.SourceTokens {
			if function, exists := functionByIO[token]; exists {
				ioToken, functionToken = token, function
				break
			}
		}
		if ioToken == "" || !equalCrossViewTokens(unresolved.SourceTokens, normalizedTokens(ioToken, functionToken)) {
			return fmt.Errorf("cross-view unresolved target has no supporting FunctionIO/function pair")
		}
		if _, duplicate := terminalByIO[ioToken]; duplicate {
			return fmt.Errorf("cross-view FunctionIO has multiple target outcomes")
		}
		terminalByIO[ioToken] = struct{}{}
	}

	// An unpaired FunctionIO can still have IOID=0. Its original IOID
	// value is intentionally not available from public source-backed edges.
	remainingResolved := len(functionByIO) - len(terminalByIO)
	remainingUnresolved := p.TaskFunctionIOCount - len(functionByIO)
	if remainingUnresolved < 0 ||
		p.TaskFunctionIONoTargetCount < remainingResolved ||
		p.TaskFunctionIONoTargetCount > remainingResolved+remainingUnresolved {
		return fmt.Errorf("cross-view IOID=0 count contradicts source-backed target outcomes")
	}
	return nil
}

func equalCrossViewTokens(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
