package modelsearch

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"tei/internal/core"
)

// Candidate converts a replacement Proposal into a Shared Acceptance change
// candidate made of set-symbol operations. The wire shape mirrors
// tei.reference.change-candidate/v0; it is not a platform schema.
//
// set-symbol changes a display symbol only. A new symbol whose canonical
// segment differs from the last segment of the subject ID would change the
// identifier, so it is refused here instead of being sent to the boundary.

const candidateSchema = "tei.reference.change-candidate/v0"

var revisionPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

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
	Op     string `json:"op"`
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
}

var setSymbolKind = map[string]string{
	KindData:       "data",
	KindAttribute:  "attribute",
	KindRelation:   "relation",
	KindCandidate:  "candidate-key",
	KindDependency: "functional-dependency",
}

// acceptanceBaseRevision is the target's model revision reported by the
// Shared Acceptance `revision` command; it differs from the proposal's own
// structure revision, which this package cannot translate.
func Candidate(p Proposal, acceptanceBaseRevision string) ([]byte, error) {
	if p.Applied {
		return nil, fmt.Errorf("proposal is marked applied")
	}
	if !revisionPattern.MatchString(acceptanceBaseRevision) {
		return nil, fmt.Errorf("base revision must be sha256:<64 hex digits>")
	}
	if len(p.Diagnostics) > 0 {
		return nil, fmt.Errorf("proposal has %d diagnostic(s); resolve them first (%s)", len(p.Diagnostics), p.Diagnostics[0].Code)
	}
	if len(p.Edits) == 0 {
		return nil, fmt.Errorf("proposal has no edits")
	}
	var delta []operation
	for _, e := range p.Edits {
		kind, ok := setSymbolKind[e.Kind]
		if !ok {
			return nil, fmt.Errorf("%s: kind %q has no set-symbol operation", e.ID, e.Kind)
		}
		seg, err := core.SymbolToSegment(e.After)
		if err != nil {
			return nil, fmt.Errorf("%s: new symbol %q: %w", e.ID, e.After, err)
		}
		if last := e.ID[strings.LastIndex(e.ID, ".")+1:]; seg != last {
			return nil, fmt.Errorf("%s: new symbol %q would change the identifier (segment %q, expected %q)", e.ID, e.After, seg, last)
		}
		delta = append(delta, operation{Op: "set-symbol", Kind: kind, ID: e.ID, Symbol: e.After})
	}
	c := candidate{
		Schema: candidateSchema, ID: "model-search-" + p.BaseRevision[7:19], BaseModelRevision: acceptanceBaseRevision,
		Origin: origin{Kind: "cli", Notes: map[string]string{"proposalRevision": p.BaseRevision, "tool": "reference-model-search"}},
		Intent: "名称の置換案を表示名の変更として反映する",
		Delta:  delta,
	}
	return json.MarshalIndent(c, "", "  ")
}
