package main

import "testing"

func TestValidateCandidateMode(t *testing.T) {
	tests := []struct {
		name         string
		split        bool
		candidateOut string
		wantError    bool
	}{
		{name: "read-only split", split: true},
		{name: "merge impact", split: false},
		{name: "merge candidate", split: false, candidateOut: "candidate.json"},
		{name: "split candidate is refused", split: true, candidateOut: "candidate.json", wantError: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCandidateMode(tc.split, tc.candidateOut)
			if (err != nil) != tc.wantError {
				t.Fatalf("validateCandidateMode(split=%v, candidateOut=%q) error=%v, wantError=%v", tc.split, tc.candidateOut, err, tc.wantError)
			}
		})
	}
}
