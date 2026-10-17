// Package typereorg lists what a proposed merge of semantic types would touch.
// It is read-only: it never rewrites a type reference, a mapping, or a
// Workspace, and it never decides that two types mean the same thing.
package typereorg

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/ir"
)

const Schema = "tei.reference.type-reorg-impact/v0"

// Merge proposes moving every use of From onto Into.
type Merge struct {
	From []string `json:"from"`
	Into string   `json:"into"`
}

type Use struct {
	Attribute string `json:"attribute"`
	Data      string `json:"data"`
	Type      string `json:"type"`
}

// ValidationUse records a Validation whose referenced attribute uses a merged type.
type ValidationUse struct {
	Validation string `json:"validation"`
	Data       string `json:"data"`
	Attribute  string `json:"attribute"`
	Type       string `json:"type"`
}

type KeyUse struct {
	Kind   string   `json:"kind"`
	ID     string   `json:"id"`
	Data   string   `json:"data"`
	Fields []string `json:"attributes"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

type Impact struct {
	Schema            string          `json:"schema"`
	StructureRevision string          `json:"structureRevision"`
	Merge             Merge           `json:"merge"`
	Attributes        []Use           `json:"attributes"`
	Validations       []ValidationUse `json:"validations"`
	AliasTypes        []string        `json:"aliasTypes"`
	Keys              []KeyUse        `json:"keys"`
	Diagnostics       []Diagnostic    `json:"diagnostics"`
	// ReviewRequired is always true: a merge is a meaning decision that a
	// matching representation never settles.
	ReviewRequired bool `json:"reviewRequired"`
	Applied        bool `json:"applied"`
}

func Analyze(world ir.World, m Merge) (Impact, error) {
	if m.Into == "" || len(m.From) == 0 {
		return Impact{}, fmt.Errorf("merge needs from and into")
	}
	raw, err := json.Marshal(struct {
		S, P interface{}
	}{world.Semantics, world.Persistence})
	if err != nil {
		return Impact{}, err
	}
	out := Impact{
		Schema: Schema, StructureRevision: fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), Merge: m,
		Attributes: []Use{}, Validations: []ValidationUse{}, AliasTypes: []string{}, Keys: []KeyUse{}, Diagnostics: []Diagnostic{},
		ReviewRequired: true,
	}
	from := map[string]bool{}
	for _, f := range m.From {
		from[f] = true
	}
	types := map[string]struct{ symbol, kind, underlying string }{}
	for _, ns := range world.Semantics {
		for _, t := range ns.Types {
			types[string(t.ID)] = struct{ symbol, kind, underlying string }{t.Symbol, string(t.Kind), string(t.Underlying)}
		}
	}
	diag := func(code, subject string) {
		out.Diagnostics = append(out.Diagnostics, Diagnostic{Code: code, Subject: subject})
	}
	into, ok := types[m.Into]
	if !ok {
		diag("unknown-target-type", m.Into)
	}
	for _, f := range m.From {
		t, known := types[f]
		switch {
		case f == m.Into:
			diag("merge-into-itself", f)
		case !known:
			diag("unknown-source-type", f)
		default:
			if t.kind == "primitive" {
				diag("source-is-primitive", f)
			}
			if ok && t.underlying != into.underlying {
				diag("underlying-representation-differs", f)
			}
			if ok && t.symbol != into.symbol {
				diag("symbol-differs-meaning-unconfirmed", f)
			}
		}
	}

	attrType := map[string]string{}
	attrData := map[string]string{}
	for _, ns := range world.Semantics {
		for _, d := range ns.Data {
			for _, a := range d.Attributes {
				attrType[string(a.ID)] = string(a.TypeRef)
				attrData[string(a.ID)] = string(d.ID)
				if from[string(a.TypeRef)] {
					out.Attributes = append(out.Attributes, Use{string(a.ID), string(d.ID), string(a.TypeRef)})
				}
			}
		}
	}
	for id, t := range types {
		if from[t.underlying] {
			out.AliasTypes = append(out.AliasTypes, id)
		}
	}
	sort.Strings(out.AliasTypes)

	// Validations whose referenced attribute uses a merged type are also impacted.
	for _, ns := range world.Semantics {
		for _, v := range ns.Validations {
			if from[attrType[string(v.Attribute)]] {
				out.Validations = append(out.Validations, ValidationUse{
					Validation: string(v.ID),
					Data:       attrData[string(v.Attribute)],
					Attribute:  string(v.Attribute),
					Type:       attrType[string(v.Attribute)],
				})
			}
		}
	}

	touched := func(ids []string) bool {
		for _, id := range ids {
			if from[attrType[id]] {
				return true
			}
		}
		return false
	}
	ids := func(in []string) []string { return in }
	for _, ns := range world.Semantics {
		for _, d := range ns.Data {
			for _, k := range d.CandidateKeys {
				a := make([]string, 0, len(k.Attributes))
				for _, x := range k.Attributes {
					a = append(a, string(x))
				}
				if touched(ids(a)) {
					out.Keys = append(out.Keys, KeyUse{"candidateKey", string(k.ID), string(d.ID), a})
				}
			}
			for _, f := range d.FunctionalDependencies {
				var a []string
				for _, x := range f.Determinants {
					a = append(a, string(x))
				}
				for _, x := range f.Dependents {
					a = append(a, string(x))
				}
				if touched(a) {
					out.Keys = append(out.Keys, KeyUse{"functionalDependency", string(f.ID), string(d.ID), a})
				}
			}
			for _, r := range d.Relations {
				for _, k := range r.Keys {
					s, t := string(k.SourceAttribute), string(k.TargetAttribute)
					if !touched([]string{s, t}) {
						continue
					}
					out.Keys = append(out.Keys, KeyUse{"relationKey", string(r.ID), string(d.ID), []string{s, t}})
					if attrType[s] != attrType[t] {
						// Types already differ across the key. A merge may make them
						// equal, which hides the difference instead of resolving it.
						diag("relation-key-type-already-differs", string(r.ID))
						after := func(id string) string {
							if from[id] {
								return m.Into
							}
							return id
						}
						if after(attrType[s]) == after(attrType[t]) {
							diag("relation-key-difference-would-be-hidden", string(r.ID))
						}
					}
				}
			}
		}
	}

	// Physical storage that differs between the merged types goes through the
	// migration safety gate, not through this analysis.
	storage := map[string]map[string]bool{}
	for _, pm := range world.Persistence {
		for _, c := range pm.Columns {
			if from[attrType[string(c.Attribute)]] {
				t := attrType[string(c.Attribute)]
				if storage[t] == nil {
					storage[t] = map[string]bool{}
				}
				storage[t][string(c.Storage)] = true
			}
		}
	}
	seen := map[string]bool{}
	for _, set := range storage {
		for s := range set {
			seen[s] = true
		}
	}
	if len(seen) > 1 {
		diag("physical-storage-differs", "attributes of merged types")
	}

	sort.Slice(out.Attributes, func(i, j int) bool { return out.Attributes[i].Attribute < out.Attributes[j].Attribute })
	sort.Slice(out.Keys, func(i, j int) bool {
		if out.Keys[i].Kind != out.Keys[j].Kind {
			return out.Keys[i].Kind < out.Keys[j].Kind
		}
		return out.Keys[i].ID < out.Keys[j].ID
	})
	sort.Slice(out.Validations, func(i, j int) bool {
		return out.Validations[i].Validation < out.Validations[j].Validation
	})
	sort.Slice(out.Diagnostics, func(i, j int) bool {
		if out.Diagnostics[i].Code != out.Diagnostics[j].Code {
			return out.Diagnostics[i].Code < out.Diagnostics[j].Code
		}
		return out.Diagnostics[i].Subject < out.Diagnostics[j].Subject
	})
	return out, nil
}
