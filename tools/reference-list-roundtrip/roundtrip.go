// Package listroundtrip exports a flat list of semantic elements for editing in
// a spreadsheet and turns the edited list back into a change proposal. It is
// read-only: a list never changes a Workspace, and a name is never trusted as
// an identity when it can match more than one element.
package listroundtrip

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"tei/internal/ir"
)

const ProposalSchema = "tei.reference.list-change-proposal/v0"

var header = []string{"kind", "id", "parent", "symbol", "new_symbol", "base_revision"}

type Row struct {
	Kind, ID, Parent, Symbol string
}

// Rows lists Data and Attribute elements in a stable order.
func Rows(world ir.World) []Row {
	var out []Row
	for _, ns := range world.Semantics {
		for _, d := range ns.Data {
			out = append(out, Row{"data", string(d.ID), string(ns.ID), d.Symbol})
			for _, a := range d.Attributes {
				out = append(out, Row{"attribute", string(a.ID), string(d.ID), a.Symbol})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func Revision(world ir.World) (string, error) {
	b, err := json.Marshal(world.Semantics)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(b)), nil
}

// Export writes the list. Every row carries the revision it was exported from.
func Export(w io.Writer, world ir.World) error {
	rev, err := Revision(world)
	if err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range Rows(world) {
		if err := cw.Write([]string{r.Kind, r.ID, r.Parent, r.Symbol, "", rev}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

type Edit struct {
	Line   int    `json:"line"`
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Before string `json:"before"`
	After  string `json:"after"`
	// Resolved says how the target was identified: "id" or "name".
	Resolved string `json:"resolved"`
}

type Diagnostic struct {
	Line    int    `json:"line"`
	Code    string `json:"code"`
	Subject string `json:"subject,omitempty"`
}

type Proposal struct {
	Schema       string       `json:"schema"`
	BaseRevision string       `json:"baseRevision"`
	Edits        []Edit       `json:"edits"`
	Diagnostics  []Diagnostic `json:"diagnostics"`
	Applied      bool         `json:"applied"`
}

// Propose reads an edited list. A row without new_symbol is ignored. Rows are
// matched by id when present; a name alone is accepted only when it matches
// exactly one element of that kind. Every problem is kept as a diagnostic and
// no row is silently skipped or re-targeted.
func Propose(world ir.World, r io.Reader) (Proposal, error) {
	rev, err := Revision(world)
	if err != nil {
		return Proposal{}, err
	}
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	records, err := cr.ReadAll()
	if err != nil {
		return Proposal{}, fmt.Errorf("read list: %w", err)
	}
	if len(records) == 0 {
		return Proposal{}, fmt.Errorf("list is empty")
	}
	col := map[string]int{}
	for i, h := range records[0] {
		col[strings.TrimSpace(h)] = i
	}
	for _, need := range []string{"kind", "symbol", "new_symbol"} {
		if _, ok := col[need]; !ok {
			return Proposal{}, fmt.Errorf("list has no %q column", need)
		}
	}
	get := func(rec []string, name string) string {
		if i, ok := col[name]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}

	byID := map[string]Row{}
	byName := map[string][]Row{}
	for _, row := range Rows(world) {
		byID[row.ID] = row
		k := row.Kind + "\x00" + row.Symbol
		byName[k] = append(byName[k], row)
	}

	p := Proposal{Schema: ProposalSchema, BaseRevision: rev, Edits: []Edit{}, Diagnostics: []Diagnostic{}}
	diag := func(line int, code, subject string) {
		p.Diagnostics = append(p.Diagnostics, Diagnostic{Line: line, Code: code, Subject: subject})
	}
	targeted := map[string]int{}
	for i, rec := range records[1:] {
		line := i + 2
		next := get(rec, "new_symbol")
		if next == "" {
			continue
		}
		if b := get(rec, "base_revision"); b != "" && b != rev {
			diag(line, "stale-list", b)
			continue
		}
		kind, id, name := get(rec, "kind"), get(rec, "id"), get(rec, "symbol")
		var target Row
		how := "name"
		if id != "" {
			row, ok := byID[id]
			if !ok {
				diag(line, "unknown-id", id)
				continue
			}
			if row.Kind != kind {
				diag(line, "kind-mismatch", id)
				continue
			}
			if name != "" && row.Symbol != name {
				diag(line, "symbol-differs-from-source", id)
				continue
			}
			target, how = row, "id"
		} else {
			matches := byName[kind+"\x00"+name]
			switch len(matches) {
			case 0:
				diag(line, "name-not-found", name)
				continue
			case 1:
				target = matches[0]
			default:
				diag(line, "ambiguous-name", name)
				continue
			}
		}
		if prev, dup := targeted[target.ID]; dup {
			diag(line, "duplicate-target", fmt.Sprintf("%s (line %d)", target.ID, prev))
			continue
		}
		targeted[target.ID] = line
		if next == target.Symbol {
			continue
		}
		p.Edits = append(p.Edits, Edit{Line: line, Kind: target.Kind, ID: target.ID, Before: target.Symbol, After: next, Resolved: how})
	}

	// A new name must stay unique among elements with the same kind and parent.
	after := map[string]string{}
	for _, e := range p.Edits {
		after[e.ID] = e.After
	}
	seen := map[string]string{}
	for _, row := range Rows(world) {
		sym := row.Symbol
		if v, ok := after[row.ID]; ok {
			sym = v
		}
		k := row.Kind + "\x00" + row.Parent + "\x00" + sym
		if other, dup := seen[k]; dup {
			_, aff1 := after[other]
			_, aff2 := after[row.ID]
			if aff1 || aff2 {
				diag(0, "symbol-collision", other+","+row.ID)
			}
		}
		seen[k] = row.ID
	}
	return p, nil
}
