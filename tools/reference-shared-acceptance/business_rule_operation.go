package main

import (
	"fmt"
	"strings"
)

// These operation normalizers define source-authored assertions only.
// Business truth, key minimality, actor authority and physical DDL are not
// inferred or granted by normalizing an explicit logical rule.
func normalizeBusinessRuleIdentity(prefix, dataName, idName, symbol string) (string, string, string, error) {
	data, err := canonical(prefix+".data", dataName)
	if err != nil {
		return "", "", "", err
	}
	id, err := canonical(prefix+".id", idName)
	if err != nil {
		return "", "", "", err
	}
	segment, err := segmentOf(symbol)
	if err != nil {
		return "", "", "", fmt.Errorf("%s.symbol: %w", prefix, err)
	}
	if string(id) != string(data)+"."+segment {
		return "", "", "", fmt.Errorf("%s.id %s must belong to Data %s and match symbol %s", prefix, id, data, symbol)
	}
	return string(data), string(id), symbol, nil
}

func normalizeBusinessRuleAttributes(prefix, data string, attributes []string) ([]string, error) {
	if len(attributes) == 0 {
		return nil, fmt.Errorf("%s must contain at least one declared attribute", prefix)
	}
	seen := make(map[string]bool, len(attributes))
	out := make([]string, 0, len(attributes))
	for i, raw := range attributes {
		id, err := canonical(fmt.Sprintf("%s[%d]", prefix, i), raw)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(string(id), data+".") {
			return nil, fmt.Errorf("%s[%d] %s is outside Data %s", prefix, i, id, data)
		}
		if seen[string(id)] {
			return nil, fmt.Errorf("%s[%d] repeats attribute %s", prefix, i, id)
		}
		seen[string(id)] = true
		out = append(out, string(id))
	}
	return out, nil
}

func normalizeAddCandidateKey(op Operation) (Operation, error) {
	if err := onlyFields(op, "addCandidateKey"); err != nil {
		return op, err
	}
	if strings.TrimSpace(op.Placement) != "" {
		return op, fmt.Errorf("%s does not accept placement; the Data's source file owns the rule", op.Op)
	}
	if op.AddCandidateKey == nil {
		return op, fmt.Errorf("%s requires addCandidateKey", op.Op)
	}
	spec := *op.AddCandidateKey
	data, id, symbol, err := normalizeBusinessRuleIdentity("addCandidateKey", spec.Data, spec.ID, spec.Symbol)
	if err != nil {
		return op, err
	}
	fields, err := normalizeBusinessRuleAttributes("addCandidateKey.attributes", data, spec.Attributes)
	if err != nil {
		return op, err
	}
	return Operation{Op: opAddCandidateKey, AddCandidateKey: &AddCandidateKeySpec{
		Data: data, ID: id, Symbol: symbol, Attributes: fields,
	}}, nil
}

func normalizeAddFunctionalDependency(op Operation) (Operation, error) {
	if err := onlyFields(op, "addFunctionalDependency"); err != nil {
		return op, err
	}
	if strings.TrimSpace(op.Placement) != "" {
		return op, fmt.Errorf("%s does not accept placement; the Data's source file owns the rule", op.Op)
	}
	if op.AddFunctionalDependency == nil {
		return op, fmt.Errorf("%s requires addFunctionalDependency", op.Op)
	}
	spec := *op.AddFunctionalDependency
	data, id, symbol, err := normalizeBusinessRuleIdentity("addFunctionalDependency", spec.Data, spec.ID, spec.Symbol)
	if err != nil {
		return op, err
	}
	determinants, err := normalizeBusinessRuleAttributes("addFunctionalDependency.determinants", data, spec.Determinants)
	if err != nil {
		return op, err
	}
	dependents, err := normalizeBusinessRuleAttributes("addFunctionalDependency.dependents", data, spec.Dependents)
	if err != nil {
		return op, err
	}
	// A trivial FD is legal: the same attribute may appear once in each
	// group. Its utility and truth are outside syntactic normalization.
	return Operation{Op: opAddFunctionalDependency, AddFunctionalDependency: &AddFunctionalDependencySpec{
		Data: data, ID: id, Symbol: symbol, Determinants: determinants, Dependents: dependents,
	}}, nil
}

func businessRuleOperationSubject(op Operation) (data, id string, attributes []string) {
	switch op.Op {
	case opAddCandidateKey:
		return op.AddCandidateKey.Data, op.AddCandidateKey.ID, append([]string(nil), op.AddCandidateKey.Attributes...)
	case opAddFunctionalDependency:
		spec := op.AddFunctionalDependency
		attrs := append([]string(nil), spec.Determinants...)
		attrs = append(attrs, spec.Dependents...)
		return spec.Data, spec.ID, attrs
	}
	return "", "", nil
}
