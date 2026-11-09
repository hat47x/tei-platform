package listroundtrip

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"tei/internal/core"
)

// Candidate converts a list Proposal into a Shared Acceptance change candidate
// made of set-symbol operations. The wire shape mirrors
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

// acceptanceBaseRevision is the model revision reported by the Shared
// Acceptance `revision` command. It differs from the list's own revision.
func Candidate(p Proposal, acceptanceBaseRevision string) ([]byte, error) {
	if p.Applied {
		return nil, fmt.Errorf("proposal is marked applied")
	}
	if !revisionPattern.MatchString(acceptanceBaseRevision) {
		return nil, fmt.Errorf("base revision must be sha256:<64 hex digits>")
	}
	if len(p.Diagnostics) > 0 {
		d := p.Diagnostics[0]
		return nil, fmt.Errorf("list has %d diagnostic(s); resolve them first (line %d: %s)", len(p.Diagnostics), d.Line, d.Code)
	}
	if len(p.Edits) == 0 {
		return nil, fmt.Errorf("list has no edits")
	}
	var delta []operation
	for _, e := range p.Edits {
		if e.Kind != "data" && e.Kind != "attribute" {
			return nil, fmt.Errorf("line %d: kind %q has no set-symbol operation", e.Line, e.Kind)
		}
		seg, err := core.SymbolToSegment(e.After)
		if err != nil {
			return nil, fmt.Errorf("line %d: new symbol %q: %w", e.Line, e.After, err)
		}
		if last := e.ID[strings.LastIndex(e.ID, ".")+1:]; seg != last {
			return nil, fmt.Errorf("line %d: new symbol %q would change the identifier (segment %q, expected %q)", e.Line, e.After, seg, last)
		}
		delta = append(delta, operation{Op: "set-symbol", Kind: e.Kind, ID: e.ID, Symbol: e.After})
	}
	c := candidate{
		Schema: candidateSchema, ID: "list-roundtrip-" + p.BaseRevision[7:19], BaseModelRevision: acceptanceBaseRevision,
		Origin: origin{Kind: "cli", Notes: map[string]string{"listRevision": p.BaseRevision, "tool": "reference-list-roundtrip"}},
		Intent: "編集した一覧の名称の変更を表示名の変更として反映する",
		Delta:  delta,
	}
	return json.MarshalIndent(c, "", "  ")
}
