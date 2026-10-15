// Package partialimport classifies what importing selected Data definitions
// from one World into another would do. It is read-only: it never writes a
// definition, and a matching identifier is never taken as proof that two
// definitions mean the same thing.
package partialimport

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/ir"
	"tei/internal/semantic"
)

const Schema = "tei.reference.partial-import-plan/v0"

// Status of one selected Data definition against the target.
const (
	New       = "new"
	Identical = "identical"
	Update    = "update"
	Conflict  = "conflict"
)

type Item struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	// Changes lists structural differences for an update.
	Changes []string `json:"changes,omitempty"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
	Detail  string `json:"detail,omitempty"`
}

type Plan struct {
	Schema         string       `json:"schema"`
	SourceRevision string       `json:"sourceRevision"`
	TargetRevision string       `json:"targetRevision"`
	Items          []Item       `json:"items"`
	Unresolved     []Diagnostic `json:"unresolved"`
	Diagnostics    []Diagnostic `json:"diagnostics"`
	// Importable is false while any item conflicts or any dependency is missing.
	Importable bool `json:"importable"`
	Applied    bool `json:"applied"`
}

// Analyze plans importing the selected Data IDs. Dependencies that exist in
// neither the target nor the selection are kept as unresolved, not dropped.
func Analyze(source, target ir.World, selected []string) (Plan, error) {
	if len(selected) == 0 {
		return Plan{}, fmt.Errorf("select at least one Data definition")
	}
	srcRev, err := revision(source)
	if err != nil {
		return Plan{}, err
	}
	tgtRev, err := revision(target)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Schema: Schema, SourceRevision: srcRev, TargetRevision: tgtRev,
		Items: []Item{}, Unresolved: []Diagnostic{}, Diagnostics: []Diagnostic{}}

	srcData, srcNS := dataOf(source)
	tgtData, tgtNS := dataOf(target)
	tgtTypes, srcTypes := typesOf(target), typesOf(source)
	pick := map[string]bool{}
	for _, id := range selected {
		pick[id] = true
	}
	ids := append([]string(nil), selected...)
	sort.Strings(ids)

	// Symbols in the target, to detect a different element taking a used name.
	tgtBySymbol := map[string]string{}
	for id, d := range tgtData {
		tgtBySymbol[tgtNS[id]+"\x00"+d.Symbol] = id
	}

	for _, id := range ids {
		d, ok := srcData[id]
		if !ok {
			p.Diagnostics = append(p.Diagnostics, Diagnostic{Code: "not-in-source", Subject: id})
			continue
		}
		existing, inTarget := tgtData[id]
		item := Item{ID: id}
		switch {
		case !inTarget:
			item.Status = New
			if other, clash := tgtBySymbol[srcNS[id]+"\x00"+d.Symbol]; clash {
				item.Status = Conflict
				p.Diagnostics = append(p.Diagnostics, Diagnostic{Code: "symbol-used-by-other-id", Subject: id, Detail: other})
			}
		case tgtNS[id] != srcNS[id]:
			item.Status = Conflict
			p.Diagnostics = append(p.Diagnostics, Diagnostic{Code: "same-id-different-namespace", Subject: id})
		default:
			item.Changes = diff(existing, d)
			if len(item.Changes) == 0 {
				item.Status = Identical
			} else {
				item.Status = Update
				p.Diagnostics = append(p.Diagnostics, dependents(target, id, existing, d)...)
			}
		}
		p.Items = append(p.Items, item)

		for _, a := range d.Attributes {
			t := string(a.TypeRef)
			if _, ok := tgtTypes[t]; !ok {
				if _, inSrc := srcTypes[t]; inSrc {
					p.Unresolved = append(p.Unresolved, Diagnostic{Code: "type-only-in-source", Subject: string(a.ID), Detail: t})
				} else {
					p.Unresolved = append(p.Unresolved, Diagnostic{Code: "type-missing", Subject: string(a.ID), Detail: t})
				}
			}
		}
		for _, r := range d.Relations {
			t := string(r.Target)
			if _, ok := tgtData[t]; !ok && !pick[t] {
				p.Unresolved = append(p.Unresolved, Diagnostic{Code: "relation-target-missing", Subject: string(r.ID), Detail: t})
			}
		}
	}

	sortDiag(p.Unresolved)
	sortDiag(p.Diagnostics)
	p.Importable = len(p.Unresolved) == 0
	for _, it := range p.Items {
		if it.Status == Conflict {
			p.Importable = false
		}
	}
	for _, d := range p.Diagnostics {
		if d.Code == "not-in-source" || d.Code == "dependent-definition-affected" {
			p.Importable = false
		}
	}
	return p, nil
}

func dataOf(w ir.World) (map[string]semantic.Data, map[string]string) {
	d, ns := map[string]semantic.Data{}, map[string]string{}
	for _, n := range w.Semantics {
		for _, x := range n.Data {
			d[string(x.ID)] = x
			ns[string(x.ID)] = string(n.ID)
		}
	}
	return d, ns
}

func typesOf(w ir.World) map[string]bool {
	out := map[string]bool{}
	for _, n := range w.Semantics {
		for _, t := range n.Types {
			out[string(t.ID)] = true
		}
	}
	return out
}

// diff compares structure that carries meaning: attribute set and types,
// relations and their keys, candidate keys. Symbols are compared too, since a
// rename is a change a person has to see.
func diff(old, next semantic.Data) []string {
	var out []string
	if old.Symbol != next.Symbol {
		out = append(out, "symbol-changed")
	}
	oa, na := map[string]semantic.Attribute{}, map[string]semantic.Attribute{}
	for _, a := range old.Attributes {
		oa[string(a.ID)] = a
	}
	for _, a := range next.Attributes {
		na[string(a.ID)] = a
	}
	for id, a := range na {
		o, ok := oa[id]
		switch {
		case !ok:
			out = append(out, "attribute-added:"+id)
		case o.TypeRef != a.TypeRef:
			out = append(out, "attribute-type-changed:"+id)
		case o.Symbol != a.Symbol:
			out = append(out, "attribute-symbol-changed:"+id)
		}
	}
	for id := range oa {
		if _, ok := na[id]; !ok {
			out = append(out, "attribute-removed:"+id)
		}
	}
	if enc(old.Relations) != enc(next.Relations) {
		out = append(out, "relations-changed")
	}
	if enc(old.CandidateKeys) != enc(next.CandidateKeys) {
		out = append(out, "candidate-keys-changed")
	}
	if enc(old.FunctionalDependencies) != enc(next.FunctionalDependencies) {
		out = append(out, "functional-dependencies-changed")
	}
	sort.Strings(out)
	return out
}

// dependents reports target definitions that refer to attributes the update
// removes or retypes, which the import must not change silently.
func dependents(target ir.World, id string, old, next semantic.Data) []Diagnostic {
	na := map[string]semantic.Attribute{}
	for _, a := range next.Attributes {
		na[string(a.ID)] = a
	}
	touched := map[string]bool{}
	for _, a := range old.Attributes {
		n, ok := na[string(a.ID)]
		if !ok || n.TypeRef != a.TypeRef {
			touched[string(a.ID)] = true
		}
	}
	var out []Diagnostic
	for _, ns := range target.Semantics {
		for _, d := range ns.Data {
			for _, r := range d.Relations {
				for _, k := range r.Keys {
					if touched[string(k.SourceAttribute)] || touched[string(k.TargetAttribute)] {
						out = append(out, Diagnostic{Code: "dependent-definition-affected", Subject: string(r.ID), Detail: id})
					}
				}
			}
			for _, k := range d.CandidateKeys {
				for _, a := range k.Attributes {
					if touched[string(a)] && string(d.ID) != id {
						out = append(out, Diagnostic{Code: "dependent-definition-affected", Subject: string(k.ID), Detail: id})
					}
				}
			}
		}
	}
	return out
}

func enc(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func revision(w ir.World) (string, error) {
	b, err := json.Marshal(w.Semantics)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(b)), nil
}

func sortDiag(d []Diagnostic) {
	sort.Slice(d, func(i, j int) bool {
		if d[i].Code != d[j].Code {
			return d[i].Code < d[j].Code
		}
		if d[i].Subject != d[j].Subject {
			return d[i].Subject < d[j].Subject
		}
		return d[i].Detail < d[j].Detail
	})
}
