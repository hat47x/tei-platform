package modelref

import (
	"fmt"
	"reflect"
	"sort"
)

// EdgeRetarget is an explicitly reviewed change of a reference endpoint.
// It cannot change the source, relation meaning, or the authoritative Source.
type EdgeRetarget struct {
	From     Key    `json:"from"`
	Relation string `json:"relation"`
	Old      Ref    `json:"old"`
	New      Ref    `json:"new"`
}

// MigrationReview is a disposable, read-only consistency projection, not a
// Shared Acceptance decision or permission to modify the source files.
type MigrationReview struct {
	BaselineRevision string       `json:"baselineRevision"`
	RuleFingerprint  string       `json:"ruleFingerprint"`
	Consistent       bool         `json:"consistent"`
	Diagnostics      []Diagnostic `json:"diagnostics,omitempty"`
	Changes          []Change     `json:"changes,omitempty"`
	Affected         []Affected   `json:"affected,omitempty"`
	AuthorityLimits  []string     `json:"authorityLimits"`
}

// ReviewMigration verifies that a proposed rekey/split/merge accounts for
// every incoming reference. In particular a split does not guess which target
// each consumer should use. Both graphs and every rebind are caller-supplied
// candidates; this function never mutates them or any authoritative source.
func ReviewMigration(before, after Graph, t Transition, retargets []EdgeRetarget, rules []Rule, baselineRevision string) MigrationReview {
	out := MigrationReview{
		BaselineRevision: baselineRevision,
		RuleFingerprint:  RuleFingerprint(rules),
		AuthorityLimits: []string{
			"migration-review-is-not-shared-acceptance",
			"baseline-revision-is-unverified-caller-input",
			"logical-retarget-is-not-source-write-authority",
		},
	}
	add := func(code, at string) { out.Diagnostics = append(out.Diagnostics, Diagnostic{Code: code, At: at}) }
	if baselineRevision == "" {
		add("unbound-baseline-revision", "baselineRevision")
	}
	if issue := ValidateTransition(before, after, t); issue != "" {
		add(issue, "transition")
	}
	for _, diagnostic := range Validate(before, rules) {
		add("before-"+diagnostic.Code, diagnostic.At)
	}
	for _, diagnostic := range Validate(after, rules) {
		add("after-"+diagnostic.Code, diagnostic.At)
	}
	out.Changes = Diff(before, after)
	out.Affected = Impact(before, out.Changes, rules)

	removed, created := make(map[Key]bool), make(map[Key]bool)
	for _, k := range t.From {
		removed[k] = true
	}
	for _, k := range t.To {
		created[k] = true
	}
	// A migration review must not authorize unrelated node edits. A separate
	// change proposal may cover them, but this narrow contract cannot.
	for _, change := range out.Changes {
		switch change.Kind {
		case "added":
			if !created[change.Key] {
				add("unrelated-node-change", keyString(change.Key))
			}
		case "removed":
			if !removed[change.Key] {
				add("unrelated-node-change", keyString(change.Key))
			}
		case "reference-changed", "reference-evidence-changed":
			// Explicit retargets cover references below; evidence-only edits
			// must be reviewed separately by the Shared Acceptance boundary.
		case "display-renamed", "physical-renamed", "version-changed", "evidence-changed":
			add("unrelated-node-change", keyString(change.Key))
		}
	}
	oldLookup, newLookup := newLookup(before.Nodes), newLookup(after.Nodes)

	// Identifying an edge by its logical source/relation/target avoids relying
	// on unstable array offsets. Duplicate identical edges are ambiguous.
	type edgeID struct {
		from     Key
		relation string
		to       Ref
	}
	oldCounts, newCounts := map[edgeID]int{}, map[edgeID]int{}
	// An edge signature may occur more than once. Keep the full multiset of
	// evidence; comparing only the last matching edge would hide a change.
	oldEvidence, newEvidence := map[edgeID][]string{}, map[edgeID][]string{}
	for _, e := range before.Edges {
		id := edgeID{e.From, e.Relation, e.To}
		oldCounts[id]++
		oldEvidence[id] = append(oldEvidence[id], evidenceKey(e.Evidence))
	}
	for _, e := range after.Edges {
		id := edgeID{e.From, e.Relation, e.To}
		newCounts[id]++
		newEvidence[id] = append(newEvidence[id], evidenceKey(e.Evidence))
	}
	for id := range oldEvidence {
		sort.Strings(oldEvidence[id])
	}
	for id := range newEvidence {
		sort.Strings(newEvidence[id])
	}
	claimed := make(map[edgeID]bool)
	for i, move := range retargets {
		at := fmt.Sprintf("retargets/%d", i)
		oldID, newID := edgeID{move.From, move.Relation, move.Old}, edgeID{move.From, move.Relation, move.New}
		oldTarget, oldIssue := oldLookup.resolve(move.Old)
		newTarget, newIssue := newLookup.resolve(move.New)
		switch {
		case oldIssue != "" || !removed[oldTarget]:
			add("retarget-old-target-invalid", at)
		case newIssue != "" || !created[newTarget]:
			add("retarget-new-target-invalid", at)
		case oldCounts[oldID] != 1 || newCounts[newID] != 1:
			add("retarget-edge-ambiguous-or-absent", at)
		case claimed[oldID]:
			add("duplicate-retarget", at)
		case !reflect.DeepEqual(oldEvidence[oldID], newEvidence[newID]):
			add("retarget-evidence-changed", at)
		default:
			claimed[oldID] = true
		}
	}
	for id, count := range oldCounts {
		at := keyString(id.from) + "/" + id.relation + "/" + id.to.Subject.Kind + ":" + id.to.Subject.ID
		if removed[id.from] {
			// Outgoing edges of a renamed source also need a source mapping,
			// which EdgeRetarget deliberately does not claim to authorize.
			add("outgoing-reference-requires-own-review", at)
			continue
		}
		target, issue := oldLookup.resolve(id.to)
		if issue != "" {
			continue
		} // reported by Validate(before)
		if removed[target] {
			if count != 1 || !claimed[id] {
				add("unreviewed-incoming-reference", at)
			}
		} else if newCounts[id] != count {
			add("unrelated-reference-changed", at)
		} else if !reflect.DeepEqual(oldEvidence[id], newEvidence[id]) {
			add("unreviewed-reference-evidence", at)
		}
	}
	for id, count := range newCounts {
		if _, existed := oldCounts[id]; existed {
			continue
		}
		matched := false
		for _, move := range retargets {
			if id == (edgeID{move.From, move.Relation, move.New}) && claimed[edgeID{move.From, move.Relation, move.Old}] {
				matched = true
				break
			}
		}
		if !matched || count != 1 {
			add("unrelated-reference-added", keyString(id.from)+"/"+id.relation)
		}
	}
	sort.Slice(out.Diagnostics, func(i, j int) bool {
		if out.Diagnostics[i].At != out.Diagnostics[j].At {
			return out.Diagnostics[i].At < out.Diagnostics[j].At
		}
		return out.Diagnostics[i].Code < out.Diagnostics[j].Code
	})
	out.Consistent = len(out.Diagnostics) == 0
	return out
}
