package xeadsource

import (
	"fmt"
	"regexp"
)

var crossViewPublicationOperations = regexp.MustCompile(`^C?R?U?D?$`)

// validateCrossViewPublication rejects malformed aggregate keys and stale
// projections. A matching fingerprint is an integrity check, not a signature
// or a claim that the underlying source meaning was accepted.
func validateCrossViewPublication(p CrossViewProjection) error {
	if p.SchemaVersion != 1 ||
		!crossViewFingerprintPattern.MatchString(p.SourceArtifactDigest) ||
		!crossViewFingerprintPattern.MatchString(p.MappingContractFingerprint) ||
		p.DerivationPolicyRevision != CrossViewDerivationPolicyRevision ||
		p.DerivationPolicyFingerprint != crossViewDerivationPolicyFingerprint() ||
		!crossViewFingerprintPattern.MatchString(p.ProjectionFingerprint) {
		return fmt.Errorf("cross-view publication requires valid source, contract, and rule fingerprints")
	}
	copyProjection := p
	copyProjection.ProjectionFingerprint = ""
	if crossViewFingerprint(copyProjection) != p.ProjectionFingerprint {
		return fmt.Errorf("cross-view projection fingerprint does not match its content")
	}
	if p.TaskCount < 0 || p.TaskFunctionIOCount < 0 ||
		p.TaskFunctionIONoTargetCount < 0 ||
		p.TaskFunctionIONoTargetCount > p.TaskFunctionIOCount ||
		p.IOTableCount < 0 {
		return fmt.Errorf("cross-view publication contains invalid counts")
	}
	for _, action := range p.Actions {
		if !crossViewFingerprintPattern.MatchString(action.ActionToken) ||
			(action.TaskToken != "" && !crossViewFingerprintPattern.MatchString(action.TaskToken)) ||
			action.FunctionIOCount < 0 {
			return fmt.Errorf("cross-view publication contains invalid action identity")
		}
	}
	for _, e := range p.SourceEdges {
		if e.Nature != CrossViewNatureSource ||
			e.DerivationRuleFingerprint != "" ||
			e.Kind == "action-function-table-potential-operation" {
			return fmt.Errorf("cross-view publication contains invalid source relation")
		}
		if err := validateCrossViewPublicationEdge(e); err != nil {
			return err
		}
	}
	for _, e := range p.DerivedEdges {
		if e.Nature != CrossViewNatureDerived ||
			e.Kind != "action-function-table-potential-operation" ||
			e.DerivationRuleFingerprint != p.DerivationPolicyFingerprint {
			return fmt.Errorf("cross-view publication contains invalid derived relation")
		}
		if err := validateCrossViewPublicationEdge(e); err != nil {
			return err
		}
	}
	for _, u := range p.Unresolved {
		if !crossViewPublicationRelation(u.RelationKind) ||
			!crossViewPublicationUnresolved(u.Code) || len(u.SourceTokens) == 0 {
			return fmt.Errorf("cross-view publication contains unsupported unresolved vocabulary")
		}
		for _, token := range u.SourceTokens {
			if !crossViewFingerprintPattern.MatchString(token) {
				return fmt.Errorf("cross-view publication contains invalid unresolved token")
			}
		}
	}
	if err := validateCrossViewSourceCounts(p); err != nil {
		return err
	}
	if err := validateCrossViewDerivedLineage(p); err != nil {
		return err
	}
	return nil
}

// validateCrossViewDerivedLineage ensures every published potential path has
// its complete source-backed action/function and function/table evidence.
// Recomputing the projection fingerprint alone cannot establish this link.
func validateCrossViewDerivedLineage(p CrossViewProjection) error {
	actions := make(map[string]CrossViewActionObservation, len(p.Actions))
	for _, action := range p.Actions {
		if _, duplicate := actions[action.ActionToken]; duplicate {
			return fmt.Errorf("cross-view publication has duplicate action observations")
		}
		actions[action.ActionToken] = action
	}
	tablesByFunction := map[string][]CrossViewEdge{}
	for _, edge := range p.SourceEdges {
		switch edge.Kind {
		case "task-action":
			action, exists := actions[edge.ToToken]
			if !exists || action.TaskToken != edge.FromToken {
				return fmt.Errorf("cross-view publication has a task/action relation without a matching observation")
			}
		case "action-function-use":
			if _, exists := actions[edge.FromToken]; !exists {
				return fmt.Errorf("cross-view publication has a function use without an observed action")
			}
		case "function-table-operation":
			tablesByFunction[edge.FromToken] = append(tablesByFunction[edge.FromToken], edge)
		}
	}
	actual := make(map[string]int, len(p.DerivedEdges))
	for _, edge := range p.DerivedEdges {
		actual[crossViewEdgeKey(edge)]++
	}
	for _, pair := range actionFunctionPairs(p.SourceEdges) {
		for _, tableEdge := range tablesByFunction[pair.functionToken] {
			expected := CrossViewEdge{
				Kind:         "action-function-table-potential-operation",
				Nature:       CrossViewNatureDerived,
				FromToken:    pair.actionToken,
				ToToken:      tableEdge.ToToken,
				Operations:   tableEdge.Operations,
				SourceTokens: normalizedTokens(append(pair.sourceTokens, tableEdge.SourceTokens...)...),
			}
			key := crossViewEdgeKey(expected)
			if actual[key] == 0 {
				return fmt.Errorf("cross-view publication is missing a source-supported potential path")
			}
			actual[key]--
		}
	}
	for _, count := range actual {
		if count != 0 {
			return fmt.Errorf("cross-view publication contains a potential path without complete source evidence")
		}
	}
	return nil
}

func validateCrossViewPublicationEdge(e CrossViewEdge) error {
	if !crossViewPublicationEdgeKind(e.Kind) ||
		!crossViewFingerprintPattern.MatchString(e.FromToken) ||
		!crossViewFingerprintPattern.MatchString(e.ToToken) ||
		len(e.SourceTokens) < 2 {
		return fmt.Errorf("cross-view publication contains unsupported relation or tokens")
	}
	from, to := false, false
	for _, token := range e.SourceTokens {
		if !crossViewFingerprintPattern.MatchString(token) {
			return fmt.Errorf("cross-view publication contains invalid relation token")
		}
		from = from || token == e.FromToken
		to = to || token == e.ToToken
	}
	if !from || !to {
		return fmt.Errorf("cross-view publication lacks source evidence for relation endpoints")
	}
	switch e.Kind {
	case "task-function-io-target":
		if e.TargetKind != "IOPanel" && e.TargetKind != "IOSpool" && e.TargetKind != "IOTable" {
			return fmt.Errorf("cross-view publication contains unsupported target kind")
		}
		if e.Operations != "" {
			return fmt.Errorf("cross-view publication contains unexpected interaction operation")
		}
	case "function-table-operation", "action-function-table-potential-operation":
		if e.TargetKind != "" || e.Operations == "" ||
			!crossViewPublicationOperations.MatchString(e.Operations) {
			return fmt.Errorf("cross-view publication contains unsupported CRUD operation")
		}
	default:
		if e.TargetKind != "" || e.Operations != "" {
			return fmt.Errorf("cross-view publication contains unexpected relation attributes")
		}
	}
	return nil
}

func crossViewPublicationEdgeKind(kind string) bool {
	switch kind {
	case "task-action", "action-function-use", "task-function-io-target",
		"function-table-operation", "action-function-table-potential-operation":
		return true
	default:
		return false
	}
}

func crossViewPublicationRelation(kind string) bool {
	switch kind {
	case "task-action", "action-function-use", "task-function-io-target", "function-table-operation":
		return true
	default:
		return false
	}
}

func crossViewPublicationUnresolved(code string) bool {
	switch code {
	case "task-action-without-task",
		"task-function-io-without-task-action",
		"task-function-io-function-id-missing",
		"task-function-io-function-not-found",
		"task-function-io-function-ambiguous",
		"task-function-io-target-id-missing",
		"task-function-io-target-not-found",
		"task-function-io-target-ambiguous",
		"iotable-without-function",
		"iotable-table-id-missing",
		"iotable-table-not-found",
		"iotable-table-ambiguous",
		"iotable-operation-marker-unsupported",
		"iotable-operation-missing":
		return true
	default:
		return false
	}
}
