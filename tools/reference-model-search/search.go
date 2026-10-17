// Package modelsearch searches the authored Symbol text of an accepted-structure
// World and turns a replacement into a reviewable proposal. It is read-only:
// it never changes a Workspace, canonical identifiers, or acceptance state.
package modelsearch

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/ir"
)

const (
	SearchSchema   = "tei.reference.model-search/v0"
	ProposalSchema = "tei.reference.model-replace-proposal/v0"
)

// Kinds of element whose Symbol can be matched.
const (
	KindData       = "data"
	KindAttribute  = "attribute"
	KindRelation   = "relation"
	KindCandidate  = "candidateKey"
	KindDependency = "functionalDependency"
)

type Query struct {
	Text          string   `json:"text"`
	CaseSensitive bool     `json:"caseSensitive"`
	Exact         bool     `json:"exact"`
	Kinds         []string `json:"kinds,omitempty"`
}

// Hit locates one Symbol. Field is always "symbol": canonical IDs are never
// a search or replacement target.
type Hit struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Parent string `json:"parent,omitempty"`
	Field  string `json:"field"`
	Value  string `json:"value"`
}

type Result struct {
	Schema            string `json:"schema"`
	StructureRevision string `json:"structureRevision"`
	Query             Query  `json:"query"`
	Hits              []Hit  `json:"hits"`
}

func Search(world ir.World, q Query) (Result, error) {
	if q.Text == "" {
		return Result{}, fmt.Errorf("search text is required")
	}
	rev, err := revision(world)
	if err != nil {
		return Result{}, err
	}
	kinds := map[string]bool{}
	for _, k := range q.Kinds {
		kinds[k] = true
	}
	out := Result{Schema: SearchSchema, StructureRevision: rev, Query: q, Hits: []Hit{}}
	for _, e := range elements(world) {
		if len(kinds) > 0 && !kinds[e.Kind] {
			continue
		}
		if matches(e.Symbol, q) {
			out.Hits = append(out.Hits, Hit{Kind: e.Kind, ID: e.ID, Parent: e.Parent, Field: "symbol", Value: e.Symbol})
		}
		if e.Description != "" && matches(e.Description, q) {
			out.Hits = append(out.Hits, Hit{Kind: e.Kind, ID: e.ID, Parent: e.Parent, Field: "description", Value: e.Description})
		}
	}
	return out, nil
}

type Edit struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

// Proposal is never applied by this package. BaseRevision lets a later
// acceptance step reject it when the Source has moved.
type Proposal struct {
	Schema       string       `json:"schema"`
	BaseRevision string       `json:"baseRevision"`
	Edits        []Edit       `json:"edits"`
	Diagnostics  []Diagnostic `json:"diagnostics"`
	Applied      bool         `json:"applied"`
}

// Propose builds a replacement proposal for the selected hits. selected holds
// element IDs; nil selects every hit. A result whose revision no longer matches
// the world is refused rather than silently re-targeted.
func Propose(world ir.World, res Result, replacement string, selected []string) (Proposal, error) {
	rev, err := revision(world)
	if err != nil {
		return Proposal{}, err
	}
	if rev != res.StructureRevision {
		return Proposal{}, fmt.Errorf("search result is stale: structure revision changed")
	}
	if strings.TrimSpace(replacement) == "" {
		return Proposal{}, fmt.Errorf("replacement symbol is required")
	}
	pick := map[string]bool{}
	for _, id := range selected {
		pick[id] = true
	}
	p := Proposal{Schema: ProposalSchema, BaseRevision: rev, Edits: []Edit{}, Diagnostics: []Diagnostic{}}
	after := map[string]string{}
	for _, h := range res.Hits {
		if selected != nil && !pick[h.ID] {
			continue
		}
		// Description hits are search-only in this increment: replacing a
		// description is not a set-symbol operation and needs a different
		// change kind. They are reported as hits but not turned into edits.
		if h.Field != "symbol" {
			continue
		}
		next := replace(h.Value, res.Query, replacement)
		if next == h.Value {
			continue
		}
		p.Edits = append(p.Edits, Edit{Kind: h.Kind, ID: h.ID, Field: "symbol", Before: h.Value, After: next})
		after[h.ID] = next
	}
	// A replaced Symbol must stay unique among siblings of the same kind.
	type key struct{ kind, parent, symbol string }
	seen := map[key]string{}
	for _, e := range elements(world) {
		sym := e.Symbol
		if v, ok := after[e.ID]; ok {
			sym = v
		}
		k := key{e.Kind, e.Parent, sym}
		if other, dup := seen[k]; dup {
			p.Diagnostics = append(p.Diagnostics, Diagnostic{Code: "symbol-collision", Subject: other + "," + e.ID})
		}
		seen[k] = e.ID
	}
	sort.Slice(p.Diagnostics, func(i, j int) bool { return p.Diagnostics[i].Subject < p.Diagnostics[j].Subject })
	return p, nil
}

type element struct {
	Kind        string
	ID          string
	Parent      string
	Symbol      string
	Description string
}

func elements(world ir.World) []element {
	var out []element
	for _, ns := range world.Semantics {
		for _, d := range ns.Data {
			did := string(d.ID)
			out = append(out, element{Kind: KindData, ID: did, Parent: string(ns.ID), Symbol: d.Symbol, Description: d.Description})
			for _, a := range d.Attributes {
				out = append(out, element{Kind: KindAttribute, ID: string(a.ID), Parent: did, Symbol: a.Symbol, Description: a.Description})
			}
			for _, r := range d.Relations {
				out = append(out, element{Kind: KindRelation, ID: string(r.ID), Parent: did, Symbol: r.Symbol})
			}
			for _, k := range d.CandidateKeys {
				out = append(out, element{Kind: KindCandidate, ID: string(k.ID), Parent: did, Symbol: k.Symbol})
			}
			for _, f := range d.FunctionalDependencies {
				out = append(out, element{Kind: KindDependency, ID: string(f.ID), Parent: did, Symbol: f.Symbol})
			}
		}
	}
	return out
}

func matches(value string, q Query) bool {
	v, t := value, q.Text
	if !q.CaseSensitive {
		v, t = strings.ToLower(v), strings.ToLower(t)
	}
	if q.Exact {
		return v == t
	}
	return strings.Contains(v, t)
}

func replace(value string, q Query, with string) string {
	if q.Exact {
		return with
	}
	if q.CaseSensitive {
		return strings.ReplaceAll(value, q.Text, with)
	}
	lv, lt := strings.ToLower(value), strings.ToLower(q.Text)
	var b strings.Builder
	for i := 0; i < len(value); {
		if strings.HasPrefix(lv[i:], lt) {
			b.WriteString(with)
			i += len(lt)
			continue
		}
		b.WriteByte(value[i])
		i++
	}
	return b.String()
}

func revision(world ir.World) (string, error) {
	b, err := json.Marshal(struct {
		Semantics interface{} `json:"semantics"`
	}{world.Semantics})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(b)), nil
}
