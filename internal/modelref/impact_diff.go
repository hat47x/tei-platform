package modelref

import "sort"

// ReviewImpactDiff compares two fully supplied candidate snapshots. Without
// an after snapshot an added subject cannot have its consumers reviewed;
// ReviewImpact intentionally rejects that case.
func ReviewImpactDiff(before, after Graph, rules []Rule) ImpactReview {
	result := ImpactReview{
		SchemaVersion:   "tei.modelref.impact-review/v0",
		RuleFingerprint: RuleFingerprint(rules),
		AuthorityLimits: []string{
			"typed-graphs-are-candidates-not-accepted-meaning",
			"snapshot-revisions-and-sources-are-unverified",
			"semantic-impact-does-not-prove-runtime-effects",
		},
	}
	for _, finding := range Validate(before, rules) {
		finding.At = "before/" + finding.At
		result.Diagnostics = append(result.Diagnostics, finding)
	}
	for _, finding := range Validate(after, rules) {
		finding.At = "after/" + finding.At
		result.Diagnostics = append(result.Diagnostics, finding)
	}
	if len(result.Diagnostics) != 0 {
		sort.Slice(result.Diagnostics, func(i, j int) bool {
			if result.Diagnostics[i].At != result.Diagnostics[j].At {
				return result.Diagnostics[i].At < result.Diagnostics[j].At
			}
			return result.Diagnostics[i].Code < result.Diagnostics[j].Code
		})
		return result
	}
	result.GraphValid = true
	result.Changes = Diff(before, after)
	// New subjects need the after graph. Removed subjects need the before
	// graph. For changed stable IDs, combine both so newly introduced
	// consumers are not silently omitted from the candidate review.
	var oldChanges, newChanges []Change
	for _, change := range result.Changes {
		switch change.Kind {
		case "added":
			newChanges = append(newChanges, change)
		case "removed":
			oldChanges = append(oldChanges, change)
		default:
			oldChanges = append(oldChanges, change)
			newChanges = append(newChanges, change)
		}
	}
	all := append(Impact(before, oldChanges, rules), Impact(after, newChanges, rules)...)
	seen := make(map[Affected]bool, len(all))
	for _, item := range all {
		if !seen[item] {
			result.Affected = append(result.Affected, item)
			seen[item] = true
		}
	}
	sort.Slice(result.Affected, func(i, j int) bool {
		a, b := result.Affected[i], result.Affected[j]
		if a.Changed != b.Changed {
			return keyString(a.Changed) < keyString(b.Changed)
		}
		if a.Consumer != b.Consumer {
			return keyString(a.Consumer) < keyString(b.Consumer)
		}
		return a.Reason < b.Reason
	})
	return result
}
