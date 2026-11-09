package main

import (
	"errors"
	"fmt"
	"sort"
)

// WitnessAuditRequest is a read-only, fixture-level identity check. It does
// not authenticate receipts, verify independent expectations, or run tests.
// The required checks are fixed independently of the attempt observations.
type WitnessAuditRequest struct {
	Cases    []WitnessCase    `json:"cases"`
	Attempts []WitnessAttempt `json:"attempts"`
}

type WitnessCase struct {
	Lock                      Lock     `json:"lock"`
	ScopeRef                  string   `json:"scopeRef"`
	ScenarioDigest            string   `json:"scenarioDigest"`
	IndependentExpectedDigest string   `json:"independentExpectedDigest"`
	RequiredChecks            []string `json:"requiredChecks"`
}

type WitnessAttempt struct {
	AttemptID                 string         `json:"attemptId"`
	LockFingerprint           string         `json:"lockFingerprint"`
	SourceFingerprint         string         `json:"sourceFingerprint"`
	ArtifactDigest            string         `json:"artifactDigest"`
	Target                    string         `json:"target"`
	ScopeRef                  string         `json:"scopeRef"`
	ScenarioDigest            string         `json:"scenarioDigest"`
	IndependentExpectedDigest string         `json:"independentExpectedDigest"`
	ExpectedSteps             int            `json:"expectedSteps"`
	ObservedSteps             int            `json:"observedSteps"`
	Complete                  bool           `json:"complete"`
	Checks                    []WitnessCheck `json:"checks"`
}

type WitnessCheck struct {
	CheckID     string `json:"checkId"`
	Status      string `json:"status"`
	EvidenceRef string `json:"evidenceRef"`
}

type WitnessAuditRow struct {
	LockFingerprint string   `json:"lockFingerprint"`
	AttemptID       string   `json:"attemptId,omitempty"`
	Status          string   `json:"status"`
	Reasons         []string `json:"reasons,omitempty"`
}

type WitnessAuditReport struct {
	ClaimCeiling string            `json:"claimCeiling"`
	Qualified    bool              `json:"qualified"`
	AllReported  bool              `json:"allReported"`
	Rows         []WitnessAuditRow `json:"rows"`
}

// AuditWitnessBindings never constructs a test result by joining observations
// from different attempts. A "reported-pass" is just an internally consistent
// self-reported fixture, not a trusted Conformance or runtime claim.
func AuditWitnessBindings(input WitnessAuditRequest) (WitnessAuditReport, error) {
	out := WitnessAuditReport{
		ClaimCeiling: "fixture-identity-consistency-only",
		Qualified:    false, Rows: []WitnessAuditRow{},
	}
	if len(input.Cases) == 0 {
		return out, errors.New("at least one independently requested case is required")
	}
	requirements := map[string]WitnessCase{}
	for _, c := range input.Cases {
		fp := c.Lock.Fingerprint
		if !validToken(fp) || c.Lock.SourceFingerprint == "" || c.Lock.ArtifactDigest == "" ||
			c.Lock.Target == "" || c.ScopeRef == "" || c.ScenarioDigest == "" ||
			c.IndependentExpectedDigest == "" || len(c.RequiredChecks) == 0 {
			return out, errors.New("case missing exact lock, scope, scenario, expected, or checks")
		}
		if _, exists := requirements[fp]; exists {
			return out, fmt.Errorf("duplicate case lock %q", fp)
		}
		seen := map[string]bool{}
		for _, name := range c.RequiredChecks {
			if !validToken(name) || seen[name] {
				return out, fmt.Errorf("invalid or duplicate required check %q", name)
			}
			seen[name] = true
		}
		requirements[fp] = c
	}
	attempts := map[string][]WitnessAttempt{}
	ids := map[string]bool{}
	evidenceOwners := map[string]string{}
	for _, a := range input.Attempts {
		if !validToken(a.AttemptID) || ids[a.AttemptID] {
			return out, fmt.Errorf("invalid or duplicate attempt ID %q", a.AttemptID)
		}
		ids[a.AttemptID] = true
		if _, exists := requirements[a.LockFingerprint]; !exists {
			return out, fmt.Errorf("attempt %q references an unrequested lock", a.AttemptID)
		}
		checks := map[string]bool{}
		for _, check := range a.Checks {
			if !validToken(check.CheckID) || checks[check.CheckID] {
				return out, fmt.Errorf("duplicate or invalid check ID in attempt %q", a.AttemptID)
			}
			checks[check.CheckID] = true
			if check.Status != "pass" && check.Status != "fail" &&
				check.Status != "skip" && check.Status != "unknown" &&
				check.Status != "partial" && check.Status != "timeout" {
				return out, fmt.Errorf("unrecognized test outcome %q", check.Status)
			}
			// Evidence may be aggregated inside a single attempt but cannot be
			// silently reused as if it came from two different attempts.
			if check.EvidenceRef != "" {
				if prior, ok := evidenceOwners[check.EvidenceRef]; ok && prior != a.AttemptID {
					return out, fmt.Errorf("evidence reference reused across attempts: %q", check.EvidenceRef)
				}
				evidenceOwners[check.EvidenceRef] = a.AttemptID
			}
		}
		attempts[a.LockFingerprint] = append(attempts[a.LockFingerprint], a)
	}
	keys := make([]string, 0, len(requirements))
	for k := range requirements {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out.AllReported = true
	for _, fp := range keys {
		c := requirements[fp]
		row := WitnessAuditRow{LockFingerprint: fp, Status: "not-run"}
		aa := attempts[fp]
		if len(aa) == 0 {
			out.AllReported = false
			out.Rows = append(out.Rows, row)
			continue
		}
		if len(aa) != 1 {
			row.Status = "ambiguous-attempt"
			row.Reasons = []string{"multiple attempts: caller must pin an exact attempt"}
			out.AllReported = false
			out.Rows = append(out.Rows, row)
			continue
		}
		a := aa[0]
		row.AttemptID = a.AttemptID
		if a.SourceFingerprint != c.Lock.SourceFingerprint ||
			a.ArtifactDigest != c.Lock.ArtifactDigest || a.Target != c.Lock.Target ||
			a.ScopeRef != c.ScopeRef || a.ScenarioDigest != c.ScenarioDigest ||
			a.IndependentExpectedDigest != c.IndependentExpectedDigest {
			row.Status = "identity-mismatch"
			row.Reasons = append(row.Reasons, "attempt source/artifact/target/scope/scenario/expected differs from case")
		}
		if !a.Complete || a.ExpectedSteps < 1 || a.ObservedSteps < a.ExpectedSteps {
			row.Reasons = append(row.Reasons, "attempt incomplete or no positive step coverage")
			if row.Status != "identity-mismatch" {
				row.Status = "incomplete"
			}
		}
		actual := map[string]WitnessCheck{}
		for _, check := range a.Checks {
			actual[check.CheckID] = check
		}
		failed := false
		for _, required := range c.RequiredChecks {
			check, ok := actual[required]
			if !ok {
				row.Reasons = append(row.Reasons, "missing required check: "+required)
				if row.Status == "not-run" {
					row.Status = "incomplete"
				}
				continue
			}
			if check.Status == "fail" {
				failed = true
				row.Reasons = append(row.Reasons, "failed required check: "+required)
				continue
			}
			if check.Status != "pass" || check.EvidenceRef == "" {
				row.Reasons = append(row.Reasons, "unverified required check: "+required)
				if row.Status == "not-run" {
					row.Status = "incomplete"
				}
			}
		}
		if row.Status == "not-run" {
			if failed {
				row.Status = "reported-failure"
			} else if len(row.Reasons) > 0 {
				row.Status = "incomplete"
			} else {
				row.Status = "reported-pass"
			}
		}
		if row.Status != "reported-pass" {
			out.AllReported = false
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}
