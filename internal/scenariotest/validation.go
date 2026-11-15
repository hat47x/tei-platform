package scenariotest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"

	"tei/internal/semantic"
)

// ValidationFixture is test-only input for existing Canonical Validation rules.
// It contains data, never a second list of rule bodies or rule-selection authority.
type ValidationFixture struct {
	Numbers      map[string]string    `json:"numbers"`
	RelationKeys []ValidationRelation `json:"relationKeys"`
	// NumberStates names numeric Attributes that have no value and says why:
	// "null" (the record holds no value) or "not-fetched" (the Attribute was
	// not read). An Attribute is stated in Numbers or here, never in both.
	NumberStates map[string]string `json:"numberStates,omitempty"`
	// Dates holds calendar dates of Attributes, written YYYY-MM-DD.
	Dates map[string]string `json:"dates,omitempty"`
	// DateStates names date Attributes that have no value, with the words of
	// NumberStates. An Attribute is stated in Dates or here, never in both.
	DateStates map[string]string `json:"dateStates,omitempty"`
}

// The words a Fixture may use for a value that is not there, and for how much
// of a relation was read.
const (
	ValidationStateNull        = "null"
	ValidationStateNotFetched  = "not-fetched"
	ValidationCoverageComplete = "complete"
	ValidationCoveragePartial  = "partial"
)

type ValidationRelation struct {
	Relation  string   `json:"relation"`
	Attribute string   `json:"attribute"`
	Keys      []string `json:"keys"`
	// Coverage is "complete" (also when omitted) or "partial": whether Keys
	// and the two counts below cover every row of the relation.
	Coverage string `json:"coverage,omitempty"`
	// NullKeys counts rows that were read and hold no value for Attribute.
	NullKeys int `json:"nullKeys,omitempty"`
	// NotFetchedKeys counts rows whose Attribute was not read.
	NotFetchedKeys int `json:"notFetchedKeys,omitempty"`
}

func ParseValidationFixture(source []byte) (ValidationFixture, error) {
	if err := RejectDuplicateJSONMembers(source); err != nil {
		return ValidationFixture{}, fmt.Errorf("decode Validation Fixture: %w", err)
	}
	var f ValidationFixture
	dec := json.NewDecoder(bytes.NewReader(source))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return ValidationFixture{}, fmt.Errorf("decode Validation Fixture: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return ValidationFixture{}, fmt.Errorf("trailing Validation Fixture content")
	}
	if f.Numbers == nil || f.RelationKeys == nil {
		return ValidationFixture{}, fmt.Errorf("Validation Fixture requires numbers and relationKeys")
	}
	for id, value := range f.Numbers {
		if strings.TrimSpace(id) == "" {
			return ValidationFixture{}, fmt.Errorf("Validation number has empty Attribute")
		}
		n, ok := new(big.Rat).SetString(value)
		if !ok || n == nil {
			return ValidationFixture{}, fmt.Errorf("Validation number %s has invalid rational value", id)
		}
	}
	for id, state := range f.NumberStates {
		if strings.TrimSpace(id) == "" {
			return ValidationFixture{}, fmt.Errorf("Validation number state has empty Attribute")
		}
		if state != ValidationStateNull && state != ValidationStateNotFetched {
			return ValidationFixture{}, fmt.Errorf("Validation number state %s is %q; write null or not-fetched", id, state)
		}
		if _, both := f.Numbers[id]; both {
			return ValidationFixture{}, fmt.Errorf("Validation number %s has both a value and the state %s", id, state)
		}
	}
	for id, value := range f.Dates {
		if strings.TrimSpace(id) == "" {
			return ValidationFixture{}, fmt.Errorf("Validation date has empty Attribute")
		}
		if _, err := semantic.ParseCalendarDate(value); err != nil {
			return ValidationFixture{}, fmt.Errorf("Validation date %s: %w", id, err)
		}
	}
	for id, state := range f.DateStates {
		if strings.TrimSpace(id) == "" {
			return ValidationFixture{}, fmt.Errorf("Validation date state has empty Attribute")
		}
		if state != ValidationStateNull && state != ValidationStateNotFetched {
			return ValidationFixture{}, fmt.Errorf("Validation date state %s is %q; write null or not-fetched", id, state)
		}
		if _, both := f.Dates[id]; both {
			return ValidationFixture{}, fmt.Errorf("Validation date %s has both a value and the state %s", id, state)
		}
	}
	seen := make(map[string]bool, len(f.RelationKeys))
	for _, relation := range f.RelationKeys {
		if relation.Relation == "" || relation.Attribute == "" || relation.Keys == nil {
			return ValidationFixture{}, fmt.Errorf("Validation relation/attribute/keys must be explicit")
		}
		if relation.Coverage != "" && relation.Coverage != ValidationCoverageComplete && relation.Coverage != ValidationCoveragePartial {
			return ValidationFixture{}, fmt.Errorf("Validation relation %s/%s has coverage %q; write complete or partial", relation.Relation, relation.Attribute, relation.Coverage)
		}
		if relation.NullKeys < 0 || relation.NotFetchedKeys < 0 {
			return ValidationFixture{}, fmt.Errorf("Validation relation %s/%s has a negative key count", relation.Relation, relation.Attribute)
		}
		key := relation.Relation + "\000" + relation.Attribute
		if seen[key] {
			return ValidationFixture{}, fmt.Errorf("duplicate Validation relation/attribute: %s/%s", relation.Relation, relation.Attribute)
		}
		seen[key] = true
	}
	return f, nil
}
