package typereorg

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/ir"
)

// Split proposes moving uses of one semantic type into several target types.
// Reassignment maps a source attribute ID to the target type ID it should move to.
// It is read-only: it never rewrites a type reference, a mapping, or a Workspace.
type Split struct {
	From         string            `json:"from"`
	Into         []string          `json:"into"`
	Reassignment map[string]string `json:"reassignment,omitempty"`
}

// SplitImpact lists what a proposed split of a semantic type would touch.
// The output shape mirrors Impact so callers can reuse the same rendering.
func SplitImpact(world ir.World, s Split) (Impact, error) {
	if s.From == "" || len(s.Into) == 0 {
		return Impact{}, fmt.Errorf("split needs from and into")
	}
	raw, err := json.Marshal(struct {
		S, P interface{}
	}{world.Semantics, world.Persistence})
	if err != nil {
		return Impact{}, err
	}
	out := Impact{
		Schema: Schema, StructureRevision: fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), Merge: Merge{From: []string{s.From}, Into: s.Into[0]},
		Attributes: []Use{}, Validations: []ValidationUse{}, Capabilities: []CapabilityUse{},
		AliasTypes: []string{}, Keys: []KeyUse{}, Diagnostics: []Diagnostic{},
		ReviewRequired: true,
	}

	from := map[string]bool{s.From: true}
	into := map[string]bool{}
	for _, t := range s.Into {
		into[t] = true
	}
	if s.Reassignment == nil {
		s.Reassignment = map[string]string{}
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

	// SplitImpact is an analysis only. Its Impact.Merge field is a legacy
	// rendering shape and must never be interpreted as a merge candidate.
	// Block Candidate even if every attribute has an explicit reassignment.
	diag("split-candidate-unsupported", s.From)

	// Target types must exist.
	for _, t := range s.Into {
		if _, ok := types[t]; !ok {
			diag("unknown-target-type", t)
		}
	}
	if _, ok := types[s.From]; !ok {
		diag("unknown-source-type", s.From)
	}

	attrType := map[string]string{}
	attrData := map[string]string{}
	for _, ns := range world.Semantics {
		for _, d := range ns.Data {
			for _, a := range d.Attributes {
				attrType[string(a.ID)] = string(a.TypeRef)
				attrData[string(a.ID)] = string(d.ID)
				if from[string(a.TypeRef)] {
					target := s.Reassignment[string(a.ID)]
					if target == "" {
						diag("split-attribute-unassigned", string(a.ID))
					} else if !into[target] {
						diag("split-attribute-unknown-target", string(a.ID))
					}
					out.Attributes = append(out.Attributes, Use{Attribute: string(a.ID), Data: string(d.ID), Type: string(a.TypeRef), Target: target})
				}
			}
		}
	}

	// Keys whose attributes would be split across different target types are flagged.
	for _, ns := range world.Semantics {
		for _, d := range ns.Data {
			for _, k := range d.CandidateKeys {
				a := make([]string, 0, len(k.Attributes))
				for _, x := range k.Attributes {
					a = append(a, string(x))
				}
				if !anyFrom(a, from, attrType) {
					continue
				}
				out.Keys = append(out.Keys, KeyUse{"candidateKey", string(k.ID), string(d.ID), a})
				if splitCrossesTargets(a, s.Reassignment) {
					diag("split-key-crosses-targets", string(k.ID))
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
				if !anyFrom(a, from, attrType) {
					continue
				}
				out.Keys = append(out.Keys, KeyUse{"functionalDependency", string(f.ID), string(d.ID), a})
				if splitCrossesTargets(a, s.Reassignment) {
					diag("split-dependency-crosses-targets", string(f.ID))
				}
			}
			for _, r := range d.Relations {
				for _, k := range r.Keys {
					sa, ta := string(k.SourceAttribute), string(k.TargetAttribute)
					if !anyFrom([]string{sa, ta}, from, attrType) {
						continue
					}
					out.Keys = append(out.Keys, KeyUse{"relationKey", string(r.ID), string(d.ID), []string{sa, ta}})
				}
			}
		}
	}

	// Validations and capabilities using the source type.
	for _, ns := range world.Semantics {
		for _, v := range ns.Validations {
			if from[attrType[string(v.Attribute)]] {
				out.Validations = append(out.Validations, ValidationUse{
					Validation: string(v.ID), Data: attrData[string(v.Attribute)],
					Attribute: string(v.Attribute), Type: attrType[string(v.Attribute)],
				})
			}
		}
		for _, c := range ns.Capabilities {
			for _, in := range c.Inputs {
				if from[string(in.Type)] {
					out.Capabilities = append(out.Capabilities, CapabilityUse{
						Capability: string(c.ID), Direction: "input", Name: string(in.Name), Type: string(in.Type),
					})
				}
			}
			if from[string(c.Output)] {
				out.Capabilities = append(out.Capabilities, CapabilityUse{
					Capability: string(c.ID), Direction: "output", Type: string(c.Output),
				})
			}
		}
	}

	// Alias types whose underlying is the source type.
	for id, t := range types {
		if from[t.underlying] {
			out.AliasTypes = append(out.AliasTypes, id)
		}
	}
	sort.Strings(out.AliasTypes)

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
	sort.Slice(out.Capabilities, func(i, j int) bool {
		if out.Capabilities[i].Capability != out.Capabilities[j].Capability {
			return out.Capabilities[i].Capability < out.Capabilities[j].Capability
		}
		return out.Capabilities[i].Direction < out.Capabilities[j].Direction
	})
	sort.Slice(out.Diagnostics, func(i, j int) bool {
		if out.Diagnostics[i].Code != out.Diagnostics[j].Code {
			return out.Diagnostics[i].Code < out.Diagnostics[j].Code
		}
		return out.Diagnostics[i].Subject < out.Diagnostics[j].Subject
	})
	return out, nil
}

func anyFrom(ids []string, from map[string]bool, attrType map[string]string) bool {
	for _, id := range ids {
		if from[attrType[id]] {
			return true
		}
	}
	return false
}

func splitCrossesTargets(ids []string, reassignment map[string]string) bool {
	targets := map[string]bool{}
	for _, id := range ids {
		t := reassignment[id]
		if t == "" {
			return false
		}
		targets[t] = true
	}
	return len(targets) > 1
}
