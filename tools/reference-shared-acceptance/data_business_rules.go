package main

import "fmt"

// normalizeNewDataBusinessRules validates the extra logical assertions in an
// add-data candidate against *only* the attributes explicitly declared in the
// same new Data. A physical key, row ID or Relation key cannot supply them.
func normalizeNewDataBusinessRules(input DataSpec, out *DataSpec, seen map[string]bool) error {
	owned := make(map[string]bool, len(out.Attributes))
	for _, attr := range out.Attributes {
		owned[attr.ID] = true
	}
	checkFields := func(ruleName string, attrs []string) error {
		for _, attr := range attrs {
			if !owned[attr] {
				return fmt.Errorf("%s references %s, which is not an attribute of new Data %s", ruleName, attr, out.ID)
			}
		}
		return nil
	}
	checkIdentity := func(ruleName, id string) error {
		if seen[id] {
			return fmt.Errorf("%s duplicates Data member identity %s", ruleName, id)
		}
		seen[id] = true
		return nil
	}
	for i, key := range input.CandidateKeys {
		label := fmt.Sprintf("data.candidateKeys[%d]", i)
		normalized, err := normalizeAddCandidateKey(Operation{
			Op: opAddCandidateKey, AddCandidateKey: &key,
		})
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		k := *normalized.AddCandidateKey
		if k.Data != out.ID {
			return fmt.Errorf("%s is owned by %s, not the new Data %s", label, k.Data, out.ID)
		}
		if err := checkIdentity(label, k.ID); err != nil {
			return err
		}
		if err := checkFields(label, k.Attributes); err != nil {
			return err
		}
		out.CandidateKeys = append(out.CandidateKeys, k)
	}
	for i, fd := range input.FunctionalDependencies {
		label := fmt.Sprintf("data.functionalDependencies[%d]", i)
		normalized, err := normalizeAddFunctionalDependency(Operation{
			Op: opAddFunctionalDependency, AddFunctionalDependency: &fd,
		})
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		f := *normalized.AddFunctionalDependency
		if f.Data != out.ID {
			return fmt.Errorf("%s is owned by %s, not the new Data %s", label, f.Data, out.ID)
		}
		if err := checkIdentity(label, f.ID); err != nil {
			return err
		}
		if err := checkFields(label, f.Determinants); err != nil {
			return err
		}
		if err := checkFields(label, f.Dependents); err != nil {
			return err
		}
		out.FunctionalDependencies = append(out.FunctionalDependencies, f)
	}
	return nil
}
