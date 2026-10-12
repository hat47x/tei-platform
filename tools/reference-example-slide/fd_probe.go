package exampleslide

import (
	"fmt"
	"sort"
)

// FDHypothesis is a question about a functional dependency inside one
// illustrative table, not an accepted semantic key or a physical constraint.
type FDHypothesis struct {
	ID           string   `json:"id"`
	Table        string   `json:"table"`
	Determinants []string `json:"determinants"`
	Dependents   []string `json:"dependents"`
}

// FDRecordPair identifies a counterexample from the illustrative base snapshot.
type FDRecordPair struct {
	LeftID  string `json:"leftId"`
	RightID string `json:"rightId"`
}

type FDFinding struct {
	HypothesisID       string         `json:"hypothesisId"`
	Table              string         `json:"table"`
	Result             string         `json:"result"`
	Reason             string         `json:"reason"`
	Counterexamples    []FDRecordPair `json:"counterexamples,omitempty"`
	ConsistentPairs    int            `json:"consistentPairs"`
	IndeterminatePairs int            `json:"indeterminatePairs"`
}

// ProbeBaseFunctionalDependencies searches the *base example only* for pairs
// with equal determinants but different dependents. It never infers a key,
// claims a dependency is established, or changes Canonical Source.
// Even a complete illustrative sample cannot prove a universal business rule.
func ProbeBaseFunctionalDependencies(catalog Catalog, example Example, hypotheses []FDHypothesis) ([]FDFinding, error) {
	state, err := validatedBaseProbeState(catalog, example)
	if err != nil {
		return nil, err
	}
	ordered := append([]FDHypothesis(nil), hypotheses...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	seen := map[string]bool{}
	var findings []FDFinding
	for _, hypothesis := range ordered {
		if hypothesis.ID == "" || seen[hypothesis.ID] {
			return nil, fmt.Errorf("duplicate or empty hypothesis ID %q", hypothesis.ID)
		}
		seen[hypothesis.ID] = true
		table, found := catalog.Tables[hypothesis.Table]
		if !found {
			return nil, fmt.Errorf("%s: unknown table %q", hypothesis.ID, hypothesis.Table)
		}
		if err := validateFDFields(table, hypothesis.Determinants); err != nil {
			return nil, fmt.Errorf("%s determinants: %w", hypothesis.ID, err)
		}
		if err := validateFDFields(table, hypothesis.Dependents); err != nil {
			return nil, fmt.Errorf("%s dependents: %w", hypothesis.ID, err)
		}

		ids := make([]string, 0, len(state))
		for id, record := range state {
			if record.Table == hypothesis.Table {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		finding := FDFinding{
			HypothesisID: hypothesis.ID, Table: hypothesis.Table,
			Result: "unresolved", Reason: "no-counterexample-in-example",
		}
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				left, right := state[ids[i]], state[ids[j]]
				switch compareFDFields(table, left, right, hypothesis.Determinants) {
				case "different":
					continue // A known determinant difference rules out this pair.
				case "unknown":
					finding.IndeterminatePairs++
					continue
				}
				switch compareFDFields(table, left, right, hypothesis.Dependents) {
				case "different":
					finding.Counterexamples = append(finding.Counterexamples, FDRecordPair{
						LeftID: ids[i], RightID: ids[j],
					})
				case "unknown":
					finding.IndeterminatePairs++
				default:
					finding.ConsistentPairs++
				}
			}
		}
		if len(finding.Counterexamples) != 0 {
			finding.Result = "contradiction"
			finding.Reason = "equal-determinants-different-dependents"
		}
		findings = append(findings, finding)
	}
	return findings, nil
}

// validatedBaseProbeState applies the same revision, catalog, ID and value
// checks to FD and candidate-key probes. It does not accept any business key.
func validatedBaseProbeState(catalog Catalog, example Example) (map[string]Record, error) {
	if catalog.Revision == "" || catalog.Revision != example.ModelRevision || example.ID == "" {
		return nil, fmt.Errorf("missing or stale example revision")
	}
	if err := validateCatalog(catalog); err != nil {
		return nil, err
	}
	state := make(map[string]Record, len(example.Base))
	for _, record := range example.Base {
		if record.ID == "" || record.Table == "" {
			return nil, fmt.Errorf("record ID and table required")
		}
		if _, exists := state[record.ID]; exists {
			return nil, fmt.Errorf("duplicate record %q", record.ID)
		}
		state[record.ID] = record
	}
	if err := validateState(catalog, state); err != nil {
		return nil, fmt.Errorf("base: %w", err)
	}
	return state, nil
}

func validateFDFields(table Table, names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("at least one attribute is required")
	}
	seen := map[string]bool{}
	for _, name := range names {
		if name == "" || seen[name] || !hasField(table, name) {
			return fmt.Errorf("unknown, empty, or repeated attribute %q", name)
		}
		seen[name] = true
	}
	return nil
}

// A mismatch in any comparable tuple component wins over missing components.
// Opaque fields remain unknown: matching bytes are not domain-level evidence.
func compareFDFields(table Table, left, right Record, names []string) string {
	unknown := false
	for _, name := range names {
		var fieldType string
		for _, field := range table.Fields {
			if field.ID == name {
				fieldType = field.Type
				break
			}
		}
		if fieldType == "opaque" {
			unknown = true
			continue
		}
		equal, comparable := equalScalar(left.Values[name], right.Values[name])
		if !comparable {
			unknown = true
		} else if !equal {
			return "different"
		}
	}
	if unknown {
		return "unknown"
	}
	return "equal"
}
