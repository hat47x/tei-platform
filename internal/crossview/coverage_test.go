package crossview

import "testing"

func TestEvaluateAgainstRequirements(t *testing.T) {
	req := []Requirement{{Subject: "function:customer", Relation: "read-contract", Owner: "validation"}}
	good := Check{Subject: "function:customer", Relation: "read-contract", Owner: "validation", Evidence: "source@rev1", Status: Applicable, Required: true}
	tests := []struct {
		name         string
		requirements []Requirement
		checks       []Check
		want         string
	}{
		{"complete declared coverage", req, []Check{good}, "eligible-for-proposal"},
		{"empty declared coverage", nil, []Check{good}, "blocked-unresolved"},
		{"missing required check", req, nil, "blocked-unresolved"},
		{"only unrelated check", req, []Check{{Subject: "other", Relation: "read-contract", Owner: "validation", Evidence: "r1", Status: Applicable, Required: true}}, "blocked-unresolved"},
		{"required marked optional", req, []Check{{Subject: good.Subject, Relation: good.Relation, Owner: good.Owner, Evidence: good.Evidence, Status: Applicable}}, "blocked-unresolved"},
		{"duplicate evidence", req, []Check{good, good}, "blocked-unresolved"},
		{"duplicate declaration", append(req, req...), []Check{good}, "blocked-unresolved"},
		{"invalid declaration", []Requirement{{Subject: "", Relation: "read-contract", Owner: "validation"}}, []Check{good}, "blocked-unresolved"},
		{"incompatible remains blocked", req, []Check{{Subject: good.Subject, Relation: good.Relation, Owner: good.Owner, Evidence: good.Evidence, Status: Incompatible, Required: true}}, "blocked-incompatible"},
		{"stale remains blocked", req, []Check{{Subject: good.Subject, Relation: good.Relation, Owner: good.Owner, Evidence: good.Evidence, Status: Stale, Required: true}}, "blocked-stale"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := EvaluateAgainstRequirements(tc.requirements, tc.checks).Outcome; got != tc.want {
				t.Fatalf("outcome %q, want %q", got, tc.want)
			}
		})
	}
}
