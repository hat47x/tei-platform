package main

import (
	"reflect"
	"testing"
)

func auditBase() WitnessAuditRequest {
	return WitnessAuditRequest{
		Cases: []WitnessCase{
			{
				Lock:     Lock{Fingerprint: "lock-a", SourceFingerprint: "src-a", ArtifactDigest: "sha-a", Target: "go-static"},
				ScopeRef: "tenant-a/function-x", ScenarioDigest: "scenario-1",
				IndependentExpectedDigest: "independent-expected-1",
				RequiredChecks:            []string{"contract", "integration"},
			},
			{
				Lock:     Lock{Fingerprint: "lock-b", SourceFingerprint: "src-b", ArtifactDigest: "sha-b", Target: "go-static"},
				ScopeRef: "tenant-a/function-x", ScenarioDigest: "scenario-1",
				IndependentExpectedDigest: "independent-expected-1",
				RequiredChecks:            []string{"contract", "integration"},
			},
		},
		Attempts: []WitnessAttempt{
			{
				AttemptID: "attempt-a", LockFingerprint: "lock-a",
				SourceFingerprint: "src-a", ArtifactDigest: "sha-a", Target: "go-static",
				ScopeRef: "tenant-a/function-x", ScenarioDigest: "scenario-1",
				IndependentExpectedDigest: "independent-expected-1",
				ExpectedSteps:             2, ObservedSteps: 2, Complete: true,
				Checks: []WitnessCheck{
					{CheckID: "contract", Status: "pass", EvidenceRef: "e-a-contract"},
					{CheckID: "integration", Status: "pass", EvidenceRef: "e-a-integration"},
				},
			},
		},
	}
}

func TestWitnessAuditNoPromotionAndNoCrossCaseInference(t *testing.T) {
	out, err := AuditWitnessBindings(auditBase())
	if err != nil {
		t.Fatal(err)
	}
	if out.Qualified || out.AllReported || out.ClaimCeiling != "fixture-identity-consistency-only" {
		t.Fatalf("improper qualification %+v", out)
	}
	if len(out.Rows) != 2 || out.Rows[0].Status != "reported-pass" || out.Rows[1].Status != "not-run" {
		t.Fatalf("incorrect case coverage %+v", out)
	}
	reversed := auditBase()
	reversed.Cases[0], reversed.Cases[1] = reversed.Cases[1], reversed.Cases[0]
	out2, err := AuditWitnessBindings(reversed)
	if err != nil || !reflect.DeepEqual(out, out2) {
		t.Fatalf("case order changes output: %+v %v", out2, err)
	}
}

func TestWitnessAuditWrongArtifactSourceTargetScopeAndExpected(t *testing.T) {
	mutations := []struct {
		name   string
		change func(*WitnessAttempt)
	}{
		{"artifact", func(a *WitnessAttempt) { a.ArtifactDigest = "old-sha" }},
		{"source", func(a *WitnessAttempt) { a.SourceFingerprint = "other-source" }},
		{"target", func(a *WitnessAttempt) { a.Target = "web-wasm" }},
		{"tenant", func(a *WitnessAttempt) { a.ScopeRef = "tenant-b/function-x" }},
		{"scenario", func(a *WitnessAttempt) { a.ScenarioDigest = "scenario-old" }},
		{"expected", func(a *WitnessAttempt) { a.IndependentExpectedDigest = "expected-old" }},
	}
	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			in := auditBase()
			m.change(&in.Attempts[0])
			out, err := AuditWitnessBindings(in)
			if err != nil {
				t.Fatal(err)
			}
			if out.Rows[0].Status != "identity-mismatch" || out.Qualified || out.AllReported {
				t.Fatalf("accepted mismatched witness %+v", out)
			}
		})
	}
}

func TestWitnessAuditIncompleteAndUnknownCannotPass(t *testing.T) {
	mutations := []struct {
		name   string
		change func(*WitnessAttempt)
	}{
		{"zero_steps", func(a *WitnessAttempt) { a.ExpectedSteps = 0; a.ObservedSteps = 0 }},
		{"partial_steps", func(a *WitnessAttempt) { a.ObservedSteps = 1 }},
		{"incomplete", func(a *WitnessAttempt) { a.Complete = false }},
		{"skipped", func(a *WitnessAttempt) { a.Checks[1].Status = "skip" }},
		{"unknown", func(a *WitnessAttempt) { a.Checks[1].Status = "unknown" }},
		{"timeout", func(a *WitnessAttempt) { a.Checks[1].Status = "timeout" }},
		{"missing_evidence", func(a *WitnessAttempt) { a.Checks[1].EvidenceRef = "" }},
		{"missing_check", func(a *WitnessAttempt) { a.Checks = a.Checks[:1] }},
	}
	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			in := auditBase()
			m.change(&in.Attempts[0])
			out, err := AuditWitnessBindings(in)
			if err != nil {
				t.Fatal(err)
			}
			if out.Rows[0].Status != "incomplete" || out.AllReported {
				t.Fatalf("unverified witness passed %+v", out)
			}
		})
	}
}

func TestWitnessAuditFailureIsDistinct(t *testing.T) {
	in := auditBase()
	in.Attempts[0].Checks[0].Status = "fail"
	out, err := AuditWitnessBindings(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Rows[0].Status != "reported-failure" || out.AllReported {
		t.Fatalf("failure classified as pass %+v", out)
	}
}

func TestWitnessAuditRejectsMultipleAttemptsInsteadOfMerging(t *testing.T) {
	in := auditBase()
	second := in.Attempts[0]
	second.AttemptID = "attempt-b"
	second.Checks = []WitnessCheck{{CheckID: "integration", Status: "pass", EvidenceRef: "e-b-integration"}}
	in.Attempts[0].Checks = []WitnessCheck{{CheckID: "contract", Status: "pass", EvidenceRef: "e-a-contract"}}
	in.Attempts = append(in.Attempts, second)
	out, err := AuditWitnessBindings(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Rows[0].Status != "ambiguous-attempt" || out.AllReported {
		t.Fatalf("combined two partial runs %+v", out)
	}
}

func TestWitnessAuditRejectsReusedEvidenceOrUnknownAttempt(t *testing.T) {
	in := auditBase()
	second := in.Attempts[0]
	second.AttemptID = "attempt-other"
	second.LockFingerprint = "lock-b"
	second.SourceFingerprint = "src-b"
	second.ArtifactDigest = "sha-b"
	in.Attempts = append(in.Attempts, second)
	if _, err := AuditWitnessBindings(in); err == nil {
		t.Fatal("same evidence reused across two attempts")
	}
	in = auditBase()
	in.Attempts[0].LockFingerprint = "phantom-lock"
	if _, err := AuditWitnessBindings(in); err == nil {
		t.Fatal("unknown lock witness accepted")
	}
	in = auditBase()
	in.Attempts[0].Checks[0].Status = "invented-status"
	if _, err := AuditWitnessBindings(in); err == nil {
		t.Fatal("unrecognized status accepted")
	}
}

func TestWitnessAuditRejectsUnboundedOrDuplicateObligations(t *testing.T) {
	in := auditBase()
	in.Cases[0].RequiredChecks = []string{"contract", "contract"}
	if _, err := AuditWitnessBindings(in); err == nil {
		t.Fatal("duplicate required check accepted")
	}
	in = auditBase()
	in.Cases[0].IndependentExpectedDigest = ""
	if _, err := AuditWitnessBindings(in); err == nil {
		t.Fatal("missing independently pinned expected accepted")
	}
	in = auditBase()
	in.Attempts = nil
	out, err := AuditWitnessBindings(in)
	if err != nil || out.Rows[0].Status != "not-run" || out.Rows[1].Status != "not-run" {
		t.Fatalf("missing attempts treated as run: %+v %v", out, err)
	}
}
