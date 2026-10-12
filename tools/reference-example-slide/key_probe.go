package exampleslide

import (
	"fmt"
	"sort"
)

// CandidateKeyHypothesis is a proposed business identifier for an illustrative
// table. It is neither a declared semantic key nor a physical PK/UI row ID.
// Minimality and universality cannot be established from an example.
type CandidateKeyHypothesis struct {
	ID     string   `json:"id"`
	Table  string   `json:"table"`
	Fields []string `json:"fields"`
}

// CandidateKeyFinding records definite collisions separately from unusable
// values and pairs whose identity cannot be established from the example.
type CandidateKeyFinding struct {
	HypothesisID      string         `json:"hypothesisId"`
	Table             string         `json:"table"`
	Result            string         `json:"result"`
	Reason            string         `json:"reason"`
	Collisions        []FDRecordPair `json:"collisions,omitempty"`
	UnusableRecordIDs []string       `json:"unusableRecordIds,omitempty"`
	IndeterminatePairs int           `json:"indeterminatePairs"`
	DistinctPairs     int            `json:"distinctPairs"`
}

// ProbeBaseCandidateKeys finds *definite* duplicate candidate-key tuples in
// one illustrative base snapshot. No absence of collisions can establish a
// candidate key, its minimality or its mandatory/non-null business meaning.
// Every finding is read-only and never changes the accepted World or DB schema.
func ProbeBaseCandidateKeys(catalog Catalog, example Example, hypotheses []CandidateKeyHypothesis) ([]CandidateKeyFinding, error) {
	state, err := validatedBaseProbeState(catalog, example)
	if err != nil {
		return nil, err
	}
	ordered := append([]CandidateKeyHypothesis(nil), hypotheses...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	seen := map[string]bool{}
	findings := make([]CandidateKeyFinding, 0, len(ordered))
	for _, hypothesis := range ordered {
		if hypothesis.ID == "" || seen[hypothesis.ID] {
			return nil, fmt.Errorf("duplicate or empty candidate key hypothesis ID %q", hypothesis.ID)
		}
		seen[hypothesis.ID] = true
		table, found := catalog.Tables[hypothesis.Table]
		if !found {
			return nil, fmt.Errorf("%s: unknown table %q", hypothesis.ID, hypothesis.Table)
		}
		if err := validateFDFields(table, hypothesis.Fields); err != nil {
			return nil, fmt.Errorf("%s candidate key fields: %w", hypothesis.ID, err)
		}
		ids := make([]string, 0, len(state))
		for id, record := range state {
			if record.Table == hypothesis.Table {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		finding := CandidateKeyFinding{
			HypothesisID: hypothesis.ID, Table: hypothesis.Table,
			Result: "unresolved", Reason: "no-confirmed-key-collision-in-example",
		}
		for _, id := range ids {
			if !allKeyComponentsComparable(table, state[id], hypothesis.Fields) {
				finding.UnusableRecordIDs = append(finding.UnusableRecordIDs, id)
			}
		}
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				switch compareFDFields(table, state[ids[i]], state[ids[j]], hypothesis.Fields) {
				case "equal":
					finding.Collisions = append(finding.Collisions, FDRecordPair{
						LeftID: ids[i], RightID: ids[j],
					})
				case "different":
					finding.DistinctPairs++
				default:
					finding.IndeterminatePairs++
				}
			}
		}
		if len(finding.Collisions) > 0 {
			finding.Result = "contradiction"
			finding.Reason = "duplicate-candidate-key-in-example"
		}
		findings = append(findings, finding)
	}
	return findings, nil
}

func allKeyComponentsComparable(table Table, record Record, names []string) bool {
	for _, name := range names {
		for _, field := range table.Fields {
			if field.ID == name {
				if field.Type == "opaque" || !nonNullScalar(record.Values[name]) {
					return false
				}
				break
			}
		}
	}
	return true
}
