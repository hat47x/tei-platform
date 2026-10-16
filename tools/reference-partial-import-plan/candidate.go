package partialimport

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// The Shared Acceptance reference currently accepts only additive operations.
// Candidate therefore carries `new` Data definitions as add-data and refuses
// every other status instead of approximating an update. The wire shape below
// mirrors tei.reference.change-candidate/v0; it is not a platform schema.

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
	Op        string    `json:"op"`
	Placement string    `json:"placement"`
	Data      *dataSpec `json:"data"`
}

type dataSpec struct {
	Namespace              string          `json:"namespace"`
	ID                     string          `json:"id"`
	Symbol                 string          `json:"symbol"`
	Attributes             []attributeSpec `json:"attributes,omitempty"`
	Relations              []relationSpec  `json:"relations,omitempty"`
	CandidateKeys          []keySpec       `json:"candidateKeys,omitempty"`
	FunctionalDependencies []fdSpec        `json:"functionalDependencies,omitempty"`
}

type attributeSpec struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

type relationSpec struct {
	ID          string            `json:"id"`
	Symbol      string            `json:"symbol"`
	Target      string            `json:"target"`
	Cardinality string            `json:"cardinality"`
	Keys        []relationKeySpec `json:"keys,omitempty"`
}

type relationKeySpec struct {
	SourceAttribute string `json:"sourceAttribute"`
	TargetAttribute string `json:"targetAttribute"`
}

type keySpec struct {
	Data       string   `json:"data"`
	ID         string   `json:"id"`
	Symbol     string   `json:"symbol"`
	Attributes []string `json:"attributes"`
}

type fdSpec struct {
	Data         string   `json:"data"`
	ID           string   `json:"id"`
	Symbol       string   `json:"symbol"`
	Determinants []string `json:"determinants"`
	Dependents   []string `json:"dependents"`
}

// Candidate builds a change candidate that adds every selected Data
// definition. It fails unless the plan is importable and every item is new.
// baseRevision is the target's model revision as reported by the Shared
// Acceptance `revision` command; this package cannot compute it.
func Candidate(plan Plan, source ir.World, baseRevision string) ([]byte, error) {
	if plan.Applied {
		return nil, fmt.Errorf("plan is marked applied")
	}
	if !revisionPattern.MatchString(baseRevision) {
		return nil, fmt.Errorf("base revision must be sha256:<64 hex digits>")
	}
	if !plan.Importable {
		return nil, fmt.Errorf("plan is not importable: resolve conflicts and unresolved dependencies first")
	}
	byID := map[string]semantic.Data{}
	ns := map[string]string{}
	for _, n := range source.Semantics {
		for _, d := range n.Data {
			byID[string(d.ID)] = d
			ns[string(d.ID)] = string(n.ID)
		}
	}
	items := append([]Item(nil), plan.Items...)
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	var delta []operation
	for _, it := range items {
		if it.Status != New {
			return nil, fmt.Errorf("%s is %s: the acceptance boundary only accepts additions", it.ID, it.Status)
		}
		d := byID[it.ID]
		spec := dataSpec{Namespace: ns[it.ID], ID: it.ID, Symbol: d.Symbol}
		for _, a := range d.Attributes {
			spec.Attributes = append(spec.Attributes, attributeSpec{string(a.ID), a.Symbol, string(a.TypeRef)})
		}
		for _, r := range d.Relations {
			rs := relationSpec{ID: string(r.ID), Symbol: r.Symbol, Target: string(r.Target), Cardinality: string(r.Cardinality)}
			for _, k := range r.Keys {
				rs.Keys = append(rs.Keys, relationKeySpec{string(k.SourceAttribute), string(k.TargetAttribute)})
			}
			spec.Relations = append(spec.Relations, rs)
		}
		for _, k := range d.CandidateKeys {
			spec.CandidateKeys = append(spec.CandidateKeys, keySpec{it.ID, string(k.ID), k.Symbol, idStrings(k.Attributes)})
		}
		for _, f := range d.FunctionalDependencies {
			spec.FunctionalDependencies = append(spec.FunctionalDependencies,
				fdSpec{it.ID, string(f.ID), f.Symbol, idStrings(f.Determinants), idStrings(f.Dependents)})
		}
		tail := strings.TrimPrefix(it.ID, ns[it.ID]+".")
		placement := "semantic/" + strings.ReplaceAll(ns[it.ID], "_", "-") + "/" + strings.ReplaceAll(tail, "_", "-") + ".tei"
		delta = append(delta, operation{Op: "add-data", Placement: placement, Data: &spec})
	}
	if len(delta) == 0 {
		return nil, fmt.Errorf("nothing to add")
	}
	c := candidate{
		Schema: candidateSchema, ID: "partial-import-" + plan.SourceRevision[7:19], BaseModelRevision: baseRevision,
		Origin: origin{Kind: "cli", Notes: map[string]string{"sourceRevision": plan.SourceRevision, "tool": "reference-partial-import-plan"}},
		Intent: "別のWorkspaceから選んだData定義を取り込む",
		Delta:  delta,
	}
	return json.MarshalIndent(c, "", "  ")
}

func idStrings[T ~string](in []T) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = string(v)
	}
	return out
}
