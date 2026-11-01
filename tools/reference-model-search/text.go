package modelsearch

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/ir"
)

// TextProposalSchema is a free-text replacement proposal. It is a different
// typed change from the Symbol replacement (ProposalSchema / set-symbol): it
// never changes a Symbol or a canonical identifier.
const TextProposalSchema = "tei.reference.model-text-replace-proposal/v0"

// TextFields are the free-text fields a text-replace proposal may change.
var TextFields = []string{"description", "display", "script"}

// ProposeText builds a free-text replacement proposal for the selected hits.
// It changes only the named free-text field. A Symbol change stays a separate
// typed change (Propose / set-symbol), so a display or description change
// cannot be mistaken for a logical identifier change.
func ProposeText(world ir.World, res Result, field, replacement string, selected []string) (Proposal, error) {
	if !isTextField(field) {
		return Proposal{}, fmt.Errorf("field %q is not a free-text field; use the symbol proposal", field)
	}
	rev, err := revision(world)
	if err != nil {
		return Proposal{}, err
	}
	if rev != res.StructureRevision {
		return Proposal{}, fmt.Errorf("search result is stale: structure revision changed")
	}
	if strings.TrimSpace(replacement) == "" {
		return Proposal{}, fmt.Errorf("replacement text is required")
	}

	pick := map[string]bool{}
	var requested []string
	for _, id := range selected {
		pick[id] = true
		requested = append(requested, id)
	}

	present := map[string]bool{}
	edited := map[string]bool{}
	p := Proposal{Schema: TextProposalSchema, BaseRevision: rev, Edits: []Edit{}, Diagnostics: []Diagnostic{}}
	for _, h := range res.Hits {
		if h.Field != field {
			continue
		}
		present[h.ID] = true
		if selected != nil && !pick[h.ID] {
			continue
		}
		next := replace(h.Value, res.Query, replacement)
		if next == h.Value {
			continue
		}
		if edited[h.ID] {
			continue
		}
		edited[h.ID] = true
		p.Edits = append(p.Edits, Edit{Kind: h.Kind, ID: h.ID, Field: h.Field, Before: h.Value, After: next})
	}

	if selected != nil {
		for _, id := range requested {
			if !present[id] {
				return Proposal{}, fmt.Errorf("selected id %q is not a hit in field %q", id, field)
			}
		}
	}

	sort.Slice(p.Edits, func(i, j int) bool {
		if p.Edits[i].ID == p.Edits[j].ID {
			return p.Edits[i].Field < p.Edits[j].Field
		}
		return p.Edits[i].ID < p.Edits[j].ID
	})
	return p, nil
}

func isTextField(field string) bool {
	for _, candidate := range TextFields {
		if candidate == field {
			return true
		}
	}
	return false
}
