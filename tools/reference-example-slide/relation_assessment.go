package exampleslide

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

// RelationFinding is a scope-limited assessment of a semantic relation.
// It does not assert that the business model itself is correct.
// Supported/contradiction/unresolved are read-only projection vocabulary.
type RelationFinding struct {
	RelationID       string   `json:"relationId"`
	SourceRecordID   string   `json:"sourceRecordId"`
	Result           string   `json:"result"`
	Reason           string   `json:"reason"`
	MatchedTargetIDs []string `json:"matchedTargetIds,omitempty"`
}

// assessRelations checks authored RelationKey tuples, not record IDs.
// "completeTables" is an explicit assertion about the example snapshot,
// NOT evidence that real production data are complete.
func assessRelations(catalog Catalog, state map[string]Record, completeTables map[string]bool) []RelationFinding {
	relations := append([]SemanticRelation(nil), catalog.Relations...)
	sort.Slice(relations, func(i, j int) bool { return relations[i].ID < relations[j].ID })
	sourceIDs := make([]string, 0, len(state))
	for id := range state {
		sourceIDs = append(sourceIDs, id)
	}
	sort.Strings(sourceIDs)
	var findings []RelationFinding
	for _, relation := range relations {
		for _, id := range sourceIDs {
			source := state[id]
			if source.Table != relation.SourceTable {
				continue
			}
			finding := RelationFinding{
				RelationID: relation.ID, SourceRecordID: id,
				Result: "unresolved", Reason: "relation-keys-not-declared",
			}
			if len(relation.Keys) == 0 {
				findings = append(findings, finding)
				continue
			}
			sourceBound := true
			for _, key := range relation.Keys {
				if !nonNullScalar(source.Values[key.SourceField]) {
					sourceBound = false
					break
				}
			}
			if !sourceBound {
				finding.Reason = "source-key-missing-or-unusable"
				findings = append(findings, finding)
				continue
			}
			possibleUnknown := false
			for _, targetID := range sourceIDs {
				target := state[targetID]
				if target.Table != relation.TargetTable {
					continue
				}
				match := true
				for _, key := range relation.Keys {
					value := target.Values[key.TargetField]
					if !nonNullScalar(value) {
						possibleUnknown = true
						match = false
						break
					}
					equal, comparable := equalScalar(source.Values[key.SourceField], value)
					if !comparable {
						possibleUnknown = true
						match = false
						break
					}
					if !equal {
						match = false
						break
					}
				}
				if match {
					finding.MatchedTargetIDs = append(finding.MatchedTargetIDs, targetID)
				}
			}
			switch {
			case relation.Cardinality == "one" && len(finding.MatchedTargetIDs) > 1:
				// Two visible matching targets prove a violation of "at most one",
				// even when the example omits other target records.
				finding.Result = "contradiction"
				finding.Reason = "multiple-targets-for-one"
			case possibleUnknown:
				finding.Reason = "unresolved-target-key"
			case !completeTables[relation.TargetTable]:
				finding.Reason = "target-table-is-partial"
			case len(finding.MatchedTargetIDs) == 0:
				// A zero-target match is not positive evidence of a record
				// correspondence for either one or many. The TEI Relation
				// does not separately specify a nonzero minimum cardinality.
				finding.Reason = "target-absence-requires-business-review"
			default:
				// Supports only the recorded join, under the stated complete
				// snapshot assumption. It is not domain-rule acceptance.
				finding.Result = "supported"
				finding.Reason = "keys-checked-in-claimed-complete-snapshot"
			}
			findings = append(findings, finding)
		}
	}
	return findings
}

func nonNullScalar(value json.RawMessage) bool {
	value = bytes.TrimSpace(value)
	if len(value) == 0 || bytes.Equal(value, []byte("null")) || !json.Valid(value) {
		return false
	}
	switch value[0] {
	case '{', '[':
		return false
	}
	return true
}

// equalScalar performs type-preserving scalar comparison. Decimal spellings
// such as 1, 1.0 and 1e0 are compared exactly using rational arithmetic,
// without float64 truncation of large identifiers. Different JSON types
// are *not* automatically coerced.
func equalScalar(a, b json.RawMessage) (equal bool, comparable bool) {
	if !nonNullScalar(a) || !nonNullScalar(b) {
		return false, false
	}
	var av, bv any
	decoder := json.NewDecoder(bytes.NewReader(a))
	decoder.UseNumber()
	if err := decoder.Decode(&av); err != nil {
		return false, false
	}
	decoder = json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	if err := decoder.Decode(&bv); err != nil {
		return false, false
	}
	switch left := av.(type) {
	case string:
		right, ok := bv.(string)
		return ok && left == right, ok
	case bool:
		right, ok := bv.(bool)
		return ok && left == right, ok
	case json.Number:
		right, ok := bv.(json.Number)
		if !ok || !boundedNumber(left.String()) || !boundedNumber(right.String()) {
			return false, false
		}
		lr, lok := new(big.Rat).SetString(left.String())
		rr, rok := new(big.Rat).SetString(right.String())
		if !lok || !rok {
			return false, false
		}
		return lr.Cmp(rr) == 0, true
	default:
		return false, false
	}
}

// Reject unreasonably long numbers and extreme exponents before allocating
// arbitrary precision rational values. Uncomparable keys become unresolved.
func boundedNumber(value string) bool {
	if len(value) > 128 {
		return false
	}
	at := strings.IndexAny(value, "eE")
	if at < 0 {
		return true
	}
	exponent := value[at+1:]
	if len(exponent) > 6 {
		return false
	}
	power, err := strconv.Atoi(exponent)
	return err == nil && power >= -1024 && power <= 1024
}

func completeTableSet(catalog Catalog, names []string) (map[string]bool, error) {
	out := make(map[string]bool, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("empty complete table ID")
		}
		if _, exists := catalog.Tables[name]; !exists {
			return nil, fmt.Errorf("unknown complete table %q", name)
		}
		if out[name] {
			return nil, fmt.Errorf("duplicate complete table %q", name)
		}
		out[name] = true
	}
	return out, nil
}
