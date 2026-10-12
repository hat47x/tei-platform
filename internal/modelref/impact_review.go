package modelref

import (
	"fmt"
	"sort"
)

// ImpactReview makes unresolved evidence visible instead of interpreting an
// empty Impact result as proof that no consumer is affected.
type ImpactReview struct {
	SchemaVersion   string       `json:"schemaVersion"`
	GraphValid      bool         `json:"graphValid"`
	RuleFingerprint string       `json:"ruleFingerprint"`
	Affected        []Affected   `json:"affected,omitempty"`
	Diagnostics     []Diagnostic `json:"diagnostics,omitempty"`
	AuthorityLimits []string     `json:"authorityLimits"`
}

// ReviewImpact validates the complete snapshot and the supplied change set
// before presenting consumers. Invalid candidate graphs never return an
// apparently complete (empty) list of consumers. Even GraphValid=true does
// not prove Shared Acceptance, provenance validity or runtime effects.
func ReviewImpact(before Graph, changes []Change, rules []Rule) ImpactReview {
	review := ImpactReview{
		SchemaVersion: "tei.modelref.impact-review/v0",
		RuleFingerprint: RuleFingerprint(rules),
		AuthorityLimits: []string{
			"typed-graph-is-a-candidate-not-accepted-meaning",
			"no-source-revision-or-acceptance-bound",
			"impact-is-review-candidate-not-observed-runtime-effect",
		},
		Diagnostics: Validate(before, rules),
	}
	counts := make(map[Key]int, len(before.Nodes))
	for _, node := range before.Nodes {
		counts[node.Key]++
	}
	for i, change := range changes {
		at := fmt.Sprintf("changes/%d", i)
		switch change.Kind {
		case "added":
			if !validKey(change.Key) {
				review.Diagnostics = append(review.Diagnostics, Diagnostic{Code: "invalid-change-key", At: at})
			}
		case "removed", "display-renamed", "physical-renamed", "version-changed",
			"evidence-changed", "reference-changed", "reference-evidence-changed":
			if !validKey(change.Key) || counts[change.Key] != 1 {
				review.Diagnostics = append(review.Diagnostics, Diagnostic{Code: "unresolved-change-key", At: at})
			}
		default:
			review.Diagnostics = append(review.Diagnostics, Diagnostic{Code: "unknown-change-kind", At: at})
		}
	}
	sort.Slice(review.Diagnostics, func(i, j int) bool {
		if review.Diagnostics[i].At != review.Diagnostics[j].At {
			return review.Diagnostics[i].At < review.Diagnostics[j].At
		}
		return review.Diagnostics[i].Code < review.Diagnostics[j].Code
	})
	if len(review.Diagnostics) != 0 {
		return review
	}
	review.GraphValid = true
	review.Affected = Impact(before, changes, rules)
	return review
}
