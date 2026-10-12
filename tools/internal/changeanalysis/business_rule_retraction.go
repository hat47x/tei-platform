package changeanalysis

import (
	"sort"

	"tei/internal/core"
	"tei/internal/ir"
)

// ImpactWithBefore extends the established read-only Impact report with
// historical edges for removed or changed business identity / FD rules.
// After-only analysis cannot show deleted links: silently treating them as
// no effect would be unsafe when a designer retracts a rule.
//
// "previous." edges describe the *old* declaration and never claim that the
// links exist in the new model. This function is not permission to remove a
// rule from canonical TEI source, nor proof of physical PK/FK dependencies.
func ImpactWithBefore(before, after ir.World, changes []SubjectChange) ([]ImpactEdge, []Unresolved) {
	edges, unresolved := Impact(after, changes)
	for _, change := range changes {
		if change.Kind != "candidate-key" && change.Kind != "functional-dependency" {
			continue
		}
		if change.Change != "removed" && change.Change != "changed" {
			continue
		}
		oldEdges := logicalRuleEdges(before, change.Kind, core.CanonicalID(change.ID))
		for _, old := range oldEdges {
			old.Via = "previous." + old.Via
			old.Reason = "Previous authored logical rule before " + change.Change + ": " + old.Reason
			edges = append(edges, old)
		}
		if len(oldEdges) == 0 {
			unresolved = append(unresolved, Unresolved{
				Code: "LOGICAL_RULE_PREVIOUS_DECLARATION_MISSING",
				Message: change.Key() + " was reportedly " + change.Change + " but its previous Data and attributes could not be resolved",
			})
		}
		unresolved = append(unresolved, Unresolved{
			Code: "LOGICAL_RULE_RETRACTION_REQUIRES_REVIEW",
			Message: change.Key() + " is " + change.Change + "; previous determinant / dependent / owner references require human review. Downstream business validity and physical PK/FK consequences are not verified",
		})
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		if edges[i].Via != edges[j].Via {
			return edges[i].Via < edges[j].Via
		}
		return edges[i].Reason < edges[j].Reason
	})
	return dedupeEdges(edges), unresolved
}
