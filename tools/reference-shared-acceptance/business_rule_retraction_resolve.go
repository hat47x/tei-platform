package main

import (
	"fmt"

	"tei/internal/sourceindex"
)

// resolveBusinessRuleRetraction requires an existing, exact-kind authored rule
// and pins the source file bytes before any preview or authorized edit.
// RoleDefinition proves only that a source declaration exists, not the rule's
// business truth or the operator's independent institutional authority.
func resolveBusinessRuleRetraction(s snapshot, op Operation) []Diagnostic {
	spec := op.RemoveBusinessRule
	kind := businessRuleRetractionKind(op)
	key := operationKey(op)
	fail := func(code, format string, args ...any) []Diagnostic {
		return []Diagnostic{{Code: code, Message: fmt.Sprintf(format, args...), Subject: key}}
	}
	var sourceFound bool
	for _, ns := range s.loaded.World.Semantics {
		for _, data := range ns.Data {
			if string(data.ID) != spec.Data {
				continue
			}
			if kind == "candidate-key" {
				for _, rule := range data.CandidateKeys {
					if string(rule.ID) == spec.ID && rule.Symbol == spec.Symbol {
						sourceFound = true
					}
				}
			} else if kind == "functional-dependency" {
				for _, rule := range data.FunctionalDependencies {
					if string(rule.ID) == spec.ID && rule.Symbol == spec.Symbol {
						sourceFound = true
					}
				}
			}
		}
	}
	if !sourceFound {
		return fail("BUSINESS_RULE_NOT_FOUND", "%s %s was not explicitly declared with that kind, Data and symbol", kind, spec.ID)
	}
	sub := sourceindex.Subject{Kind: "semantic-" + kind, ID: spec.ID}
	refs := s.index.ReferencesFor(sub)
	if len(refs) != 1 || refs[0].Role != sourceindex.RoleDefinition {
		return fail("BUSINESS_RULE_SOURCE_AMBIGUOUS", "expected exactly one canonical definition anchor for %s", spec.ID)
	}
	location, ok := s.loaded.Sources["semantic-data:"+spec.Data]
	if !ok || refs[0].ArtifactPath != location.File {
		return fail("BUSINESS_RULE_SOURCE_MISMATCH", "rule %s is not anchored in its owning Data source", spec.ID)
	}
	current, ok := s.fingerprints()[location.File]
	if !ok || current != spec.SourceFingerprint {
		return fail("BUSINESS_RULE_SOURCE_STALE", "source fingerprint for %s is missing or stale", spec.ID)
	}
	return nil
}
