package typereorg

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
)

// Candidate converts a merge Impact into a Shared Acceptance change candidate
// made of set-attribute-type operations, one per attribute that uses a merged
// type. The wire shape mirrors tei.reference.change-candidate/v0; it is not a
// platform schema.
//
// The boundary only accepts type changes between types with the same
// representation. Any diagnostic that means existing data would need a
// conversion, or that the merge itself is malformed, blocks the candidate here
// instead of being sent on. Diagnostics that ask for human review (a different
// symbol, a key type difference that would be hidden) do not block: the
// candidate still goes through an explicit Accept.

const candidateSchema = "tei.reference.change-candidate/v0"

var revisionPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

var blocking = map[string]bool{
	"split-candidate-unsupported":       true,
	"underlying-representation-differs": true,
	"physical-storage-differs":          true,
	"unknown-target-type":               true,
	"unknown-source-type":               true,
	"merge-into-itself":                 true,
	"source-is-primitive":               true,
}

type candidate struct {
	Schema            string      `json:"schema"`
	ID                string      `json:"id"`
	BaseModelRevision string      `json:"baseModelRevision"`
	Origin            origin      `json:"origin"`
	Intent            string      `json:"intent"`
	Delta             []operation `json:"delta"`
}

type origin struct {
	Kind  string            `json:"kind"`
	Notes map[string]string `json:"notes,omitempty"`
}

type operation struct {
	Op            string `json:"op"`
	ID            string `json:"id"`
	AttributeType string `json:"attributeType"`
}

// acceptanceBaseRevision is the model revision reported by the Shared
// Acceptance `revision` command; it differs from the impact's own revision.
func Candidate(imp Impact, acceptanceBaseRevision string) ([]byte, error) {
	if imp.Applied {
		return nil, fmt.Errorf("impact is marked applied")
	}
	if !revisionPattern.MatchString(acceptanceBaseRevision) {
		return nil, fmt.Errorf("base revision must be sha256:<64 hex digits>")
	}
	for _, d := range imp.Diagnostics {
		if blocking[d.Code] {
			return nil, fmt.Errorf("merge cannot become a candidate: %s (%s)", d.Code, d.Subject)
		}
	}
	if len(imp.AliasTypes) > 0 {
		return nil, fmt.Errorf("merge cannot become a candidate: %d alias type(s) are based on a merged type", len(imp.AliasTypes))
	}
	if len(imp.Attributes) == 0 {
		return nil, fmt.Errorf("no attribute uses the merged types")
	}
	var delta []operation
	for _, a := range imp.Attributes {
		delta = append(delta, operation{Op: "set-attribute-type", ID: a.Attribute, AttributeType: imp.Merge.Into})
	}
	sort.Slice(delta, func(i, j int) bool { return delta[i].ID < delta[j].ID })
	c := candidate{
		Schema: candidateSchema, ID: "type-merge-" + imp.StructureRevision[7:19], BaseModelRevision: acceptanceBaseRevision,
		Origin: origin{Kind: "cli", Notes: map[string]string{"impactRevision": imp.StructureRevision, "tool": "reference-type-reorg-impact"}},
		Intent: "型の統合案を属性の型の付け替えとして反映する",
		Delta:  delta,
	}
	return json.MarshalIndent(c, "", "  ")
}
