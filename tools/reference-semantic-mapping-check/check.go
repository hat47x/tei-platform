// Package semanticmappingcheck is a read-only reference for #2095. It checks
// the mapping between business meaning definitions and physical elements
// against three invariants: every element reaches at most one accepted
// definition, an element without an accepted definition is listed as unfilled
// (never filled by a guess), and adding a definition changes only the elements
// bound to it.
//
// The data in testdata is a small synthetic example. The check never creates,
// accepts, or suggests a definition.
package semanticmappingcheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	InputSchema  = "tei.reference.semantic-mapping-input/v0"
	ReportSchema = "tei.reference.semantic-mapping-report/v0"
	ImpactSchema = "tei.reference.semantic-mapping-impact/v0"
)

const (
	MappingAccepted  = "accepted"
	MappingCandidate = "candidate"
)

const (
	OriginHuman = "human"
	OriginAI    = "ai"
)

// Definition is a business meaning definition. The check treats its ID as the
// only identity.
type Definition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Mapping links an element to a definition. A candidate is a proposal and has
// no effect on the result. An accepted mapping needs a named accepter, so an
// AI proposal cannot become accepted by itself (#267).
type Mapping struct {
	Definition string `json:"definition"`
	Status     string `json:"status"`
	Origin     string `json:"origin"`
	AcceptedBy string `json:"acceptedBy,omitempty"`
}

// Element is a physical element (a column, an attribute). ObservedIn records
// where the implementation shows the element ("what"). It says nothing about
// the business meaning ("why"), which only an accepted definition gives.
type Element struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	ObservedIn string    `json:"observedIn,omitempty"`
	Mappings   []Mapping `json:"mappings,omitempty"`
}

type Input struct {
	Schema      string       `json:"schema"`
	Definitions []Definition `json:"definitions"`
	Elements    []Element    `json:"elements"`
}

const (
	StateDefined   = "defined"
	StateUnfilled  = "unfilled"
	StateAmbiguous = "ambiguous"
)

const (
	StatusComplete   = "complete"
	StatusIncomplete = "incomplete"
	StatusRejected   = "rejected"
)

// Resolution is the result for one element.
type Resolution struct {
	ElementID string `json:"elementId"`
	State     string `json:"state"`
	// What is "observed" when the implementation location is recorded.
	What string `json:"what"`
	// Why lists the accepted definitions: one for defined, several for
	// ambiguous, none for unfilled.
	Why []string `json:"why"`
	// Candidates lists proposed definitions that are not accepted.
	Candidates []string `json:"candidates,omitempty"`
}

type Report struct {
	Schema     string       `json:"schema"`
	Status     string       `json:"status"`
	Defined    int          `json:"defined"`
	Unfilled   []string     `json:"unfilled"`
	Ambiguous  []string     `json:"ambiguous"`
	Elements   []Resolution `json:"elements"`
	Note       string       `json:"note"`
	Definition int          `json:"definitionCount"`
}

const reportNote = "候補の提案は定義として扱わず、未記入のまま残す。定義の作成・受理・提案はしない。" +
	"合成した入力での検査結果であり、実在のデータ辞書やカタログとの一致は確認していない。"

// LoadInput reads and validates an input file.
func LoadInput(path string) (Input, error) {
	var in Input
	data, err := os.ReadFile(path)
	if err != nil {
		return in, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		return in, fmt.Errorf("%s: %w", path, err)
	}
	return in, in.Validate()
}

// Validate rejects malformed input. It does not judge the semantics.
func (in Input) Validate() error {
	if in.Schema != InputSchema {
		return fmt.Errorf("schema must be %s", InputSchema)
	}
	defs := map[string]bool{}
	for _, def := range in.Definitions {
		if strings.TrimSpace(def.ID) == "" {
			return fmt.Errorf("definition id is empty")
		}
		if defs[def.ID] {
			return fmt.Errorf("duplicate definition id %q", def.ID)
		}
		defs[def.ID] = true
	}
	seen := map[string]bool{}
	for _, el := range in.Elements {
		if strings.TrimSpace(el.ID) == "" {
			return fmt.Errorf("element id is empty")
		}
		if seen[el.ID] {
			return fmt.Errorf("duplicate element id %q", el.ID)
		}
		seen[el.ID] = true
		pairs := map[string]bool{}
		for _, m := range el.Mappings {
			if !defs[m.Definition] {
				return fmt.Errorf("element %q maps to unknown definition %q", el.ID, m.Definition)
			}
			if pairs[m.Definition] {
				return fmt.Errorf("element %q maps to definition %q twice", el.ID, m.Definition)
			}
			pairs[m.Definition] = true
			if m.Origin != OriginHuman && m.Origin != OriginAI {
				return fmt.Errorf("element %q mapping %q: origin must be %s or %s", el.ID, m.Definition, OriginHuman, OriginAI)
			}
			switch m.Status {
			case MappingCandidate:
			case MappingAccepted:
				if strings.TrimSpace(m.AcceptedBy) == "" {
					return fmt.Errorf("element %q mapping %q: accepted needs acceptedBy", el.ID, m.Definition)
				}
			default:
				return fmt.Errorf("element %q mapping %q: status must be %s or %s", el.ID, m.Definition, MappingAccepted, MappingCandidate)
			}
		}
	}
	return nil
}

// Check resolves every element. Only accepted mappings count.
func Check(in Input) (Report, error) {
	if err := in.Validate(); err != nil {
		return Report{}, err
	}
	report := Report{
		Schema:     ReportSchema,
		Unfilled:   []string{},
		Ambiguous:  []string{},
		Elements:   []Resolution{},
		Note:       reportNote,
		Definition: len(in.Definitions),
	}
	for _, el := range in.Elements {
		res := resolve(el)
		report.Elements = append(report.Elements, res)
		switch res.State {
		case StateDefined:
			report.Defined++
		case StateUnfilled:
			report.Unfilled = append(report.Unfilled, el.ID)
		case StateAmbiguous:
			report.Ambiguous = append(report.Ambiguous, el.ID)
		}
	}
	sort.Slice(report.Elements, func(i, j int) bool { return report.Elements[i].ElementID < report.Elements[j].ElementID })
	sort.Strings(report.Unfilled)
	sort.Strings(report.Ambiguous)
	switch {
	case len(report.Ambiguous) > 0:
		report.Status = StatusRejected
	case len(report.Unfilled) > 0:
		report.Status = StatusIncomplete
	default:
		report.Status = StatusComplete
	}
	return report, nil
}

func resolve(el Element) Resolution {
	res := Resolution{ElementID: el.ID, What: "not-observed", Why: []string{}}
	if el.ObservedIn != "" {
		res.What = "observed"
	}
	for _, m := range el.Mappings {
		if m.Status == MappingAccepted {
			res.Why = append(res.Why, m.Definition)
		} else {
			res.Candidates = append(res.Candidates, m.Definition)
		}
	}
	sort.Strings(res.Why)
	sort.Strings(res.Candidates)
	switch len(res.Why) {
	case 0:
		res.State = StateUnfilled
	case 1:
		res.State = StateDefined
	default:
		res.State = StateAmbiguous
	}
	return res
}

// Change is an element whose resolution differs between two inputs.
type Change struct {
	ElementID string   `json:"elementId"`
	Before    string   `json:"before"`
	After     string   `json:"after"`
	WhyBefore []string `json:"whyBefore"`
	WhyAfter  []string `json:"whyAfter"`
}

type Impact struct {
	Schema           string   `json:"schema"`
	Status           string   `json:"status"`
	AddedDefinitions []string `json:"addedDefinitions"`
	Changed          []Change `json:"changed"`
	// Outside lists changed elements that are not bound to an added
	// definition. A non-empty list rejects the change.
	Outside []string `json:"outsideScope"`
	Note    string   `json:"note"`
}

const (
	ImpactWithinScope = "within-scope"
	ImpactRejected    = "rejected"
)

const impactNote = "定義の追加で変わる要素が、その定義に結び付く要素に限られるかを調べた結果。" +
	"要素の集合が前後で異なる入力、定義の削除を含む入力は比較せずに拒否する。"

// CompareImpact compares the resolution before and after adding definitions.
// Elements must be the same set; definitions may only be added.
func CompareImpact(before, after Input) (Impact, error) {
	if err := before.Validate(); err != nil {
		return Impact{}, fmt.Errorf("before: %w", err)
	}
	if err := after.Validate(); err != nil {
		return Impact{}, fmt.Errorf("after: %w", err)
	}
	beforeDefs := map[string]bool{}
	for _, def := range before.Definitions {
		beforeDefs[def.ID] = true
	}
	afterDefs := map[string]bool{}
	added := []string{}
	for _, def := range after.Definitions {
		afterDefs[def.ID] = true
		if !beforeDefs[def.ID] {
			added = append(added, def.ID)
		}
	}
	for id := range beforeDefs {
		if !afterDefs[id] {
			return Impact{}, fmt.Errorf("definition %q was removed; only additions are compared", id)
		}
	}
	sort.Strings(added)
	addedSet := map[string]bool{}
	for _, id := range added {
		addedSet[id] = true
	}
	beforeEls := map[string]Resolution{}
	for _, el := range before.Elements {
		beforeEls[el.ID] = resolve(el)
	}
	if len(before.Elements) != len(after.Elements) {
		return Impact{}, fmt.Errorf("element sets differ: %d before, %d after", len(before.Elements), len(after.Elements))
	}
	impact := Impact{Schema: ImpactSchema, AddedDefinitions: added, Changed: []Change{}, Outside: []string{}, Note: impactNote}
	for _, el := range after.Elements {
		prev, ok := beforeEls[el.ID]
		if !ok {
			return Impact{}, fmt.Errorf("element %q exists only after", el.ID)
		}
		next := resolve(el)
		if prev.State == next.State && strings.Join(prev.Why, ",") == strings.Join(next.Why, ",") {
			continue
		}
		impact.Changed = append(impact.Changed, Change{
			ElementID: el.ID, Before: prev.State, After: next.State, WhyBefore: prev.Why, WhyAfter: next.Why,
		})
		bound := false
		for _, id := range next.Why {
			if addedSet[id] {
				bound = true
			}
		}
		if !bound {
			impact.Outside = append(impact.Outside, el.ID)
		}
	}
	sort.Slice(impact.Changed, func(i, j int) bool { return impact.Changed[i].ElementID < impact.Changed[j].ElementID })
	sort.Strings(impact.Outside)
	impact.Status = ImpactWithinScope
	if len(impact.Outside) > 0 {
		impact.Status = ImpactRejected
	}
	return impact, nil
}
