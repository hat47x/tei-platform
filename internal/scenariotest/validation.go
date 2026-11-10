package scenariotest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
)

// ValidationFixture is test-only input for existing Canonical Validation rules.
// It contains data, never a second list of rule bodies or rule-selection authority.
type ValidationFixture struct {
	Numbers      map[string]string    `json:"numbers"`
	RelationKeys []ValidationRelation `json:"relationKeys"`
}

type ValidationRelation struct {
	Relation  string   `json:"relation"`
	Attribute string   `json:"attribute"`
	Keys      []string `json:"keys"`
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
	seen := make(map[string]bool, len(f.RelationKeys))
	for _, relation := range f.RelationKeys {
		if relation.Relation == "" || relation.Attribute == "" || relation.Keys == nil {
			return ValidationFixture{}, fmt.Errorf("Validation relation/attribute/keys must be explicit")
		}
		key := relation.Relation + "\000" + relation.Attribute
		if seen[key] {
			return ValidationFixture{}, fmt.Errorf("duplicate Validation relation/attribute: %s/%s", relation.Relation, relation.Attribute)
		}
		seen[key] = true
	}
	return f, nil
}
