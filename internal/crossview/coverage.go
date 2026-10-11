package crossview

import "strings"

// Requirement identifies a check that the caller explicitly requires.
// This is a caller-declared coverage contract, NOT proof that the caller
// discovered all relevant Business, Data or Function dependencies.
type Requirement struct {
	Subject  string `json:"subject"`
	Relation string `json:"relation"`
	Owner    string `json:"owner"`
}

func requirementKey(subject, relation, owner string) string {
	return subject + "\x00" + relation + "\x00" + owner
}

// EvaluateAgainstRequirements fails closed if any declared requirement is
// missing, optional-only, duplicated, or lacks a well-formed identity.
// It does not validate source revisions or authorize proposal/acceptance.
func EvaluateAgainstRequirements(required []Requirement, checks []Check) Result {
	result := Evaluate(checks)
	if len(required) == 0 {
		result.Outcome = "blocked-unresolved"
		return result
	}
	expected := make(map[string]struct{}, len(required))
	invalid := false
	for _, req := range required {
		if strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.Relation) == "" || strings.TrimSpace(req.Owner) == "" {
			invalid = true
			continue
		}
		key := requirementKey(req.Subject, req.Relation, req.Owner)
		if _, exists := expected[key]; exists {
			invalid = true
		}
		expected[key] = struct{}{}
	}
	counts := make(map[string]int, len(checks))
	for _, check := range checks {
		key := requirementKey(check.Subject, check.Relation, check.Owner)
		if _, exists := expected[key]; exists {
			counts[key]++
			if !check.Required {
				invalid = true
			}
		}
	}
	for key := range expected {
		if counts[key] != 1 {
			invalid = true
		}
	}
	if invalid && result.Outcome == "eligible-for-proposal" {
		result.Outcome = "blocked-unresolved"
	}
	return result
}
