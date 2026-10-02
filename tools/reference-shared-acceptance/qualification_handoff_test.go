package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/knowledgecontext"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func TestApplicableQualificationAttachesExactEvidenceWithoutChangingCandidateMeaning(t *testing.T) {
	dir := scratchWorkspace(t)
	snapshot, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := knowledgecontext.Build(
		snapshot.loaded.World,
		snapshot.index,
		subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
	)
	qualification, err := knowledgecontext.Qualify(ctx, knowledgecontext.QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
		RequiredDependencies: []knowledgecontext.DependencyRequirement{{
			Direction: "outgoing",
			Relation:  "contract-query",
			Kind:      "database-contract-query",
			ID:        "master_data.customer/find_by_id",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if qualification.Status != knowledgecontext.QualificationApplicable || len(qualification.Evidence) == 0 {
		t.Fatalf("qualification=%+v", qualification)
	}

	candidate := unitPriceCandidate(snapshot.revision)
	_, beforeFingerprint, err := normalizeCandidate(candidate)
	if err != nil {
		t.Fatal(err)
	}
	grounded, err := attachQualificationEvidence(mustJSON(t, candidate), qualification)
	if err != nil {
		t.Fatal(err)
	}
	_, afterFingerprint, err := normalizeCandidate(grounded)
	if err != nil {
		t.Fatal(err)
	}
	if beforeFingerprint != afterFingerprint {
		t.Fatalf("qualification Evidence changed candidate meaning: before=%s after=%s", beforeFingerprint, afterFingerprint)
	}

	var foundExact bool
	for _, evidence := range grounded.Evidence {
		if evidence.SubjectKind == "application-pattern" &&
			evidence.Subject == "order_entry.customer_reference" &&
			evidence.Source != "" &&
			strings.HasPrefix(evidence.SourceFingerprint, "sha256:") {
			foundExact = true
		}
	}
	if !foundExact {
		t.Fatalf("exact qualification Evidence was not attached: %+v", grounded.Evidence)
	}
	var foundRequirements bool
	for _, evidence := range grounded.Evidence {
		if evidence.QualificationRequirements != nil &&
			evidence.QualificationRequirements.ExpectedKind == "application-pattern" &&
			evidence.QualificationRequirements.Application == "order_entry" {
			foundRequirements = true
		}
	}
	if !foundRequirements {
		t.Fatalf("qualification requirements were not retained: %+v", grounded.Evidence)
	}

	ev, err := evaluate(dir, mustJSON(t, grounded))
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("qualified candidate did not pass Shared Acceptance preview: %+v", ev.preview.Stages)
	}
}

func TestSharedAcceptanceRejectsTamperedQualificationEvidence(t *testing.T) {
	dir := scratchWorkspace(t)
	snapshot, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := knowledgecontext.Build(
		snapshot.loaded.World,
		snapshot.index,
		subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
	)
	qualification, err := knowledgecontext.Qualify(ctx, knowledgecontext.QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
	})
	if err != nil {
		t.Fatal(err)
	}
	grounded, err := attachQualificationEvidence(mustJSON(t, unitPriceCandidate(snapshot.revision)), qualification)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("wrong subject kind", func(t *testing.T) {
		bad := grounded
		bad.Evidence = append([]Evidence(nil), grounded.Evidence...)
		for i := range bad.Evidence {
			if bad.Evidence[i].SubjectKind != "" {
				bad.Evidence[i].SubjectKind = "semantic-data"
				break
			}
		}
		ev, err := evaluate(dir, mustJSON(t, bad))
		if err != nil {
			t.Fatal(err)
		}
		if ev.preview.Acceptable || !previewHasDiagnostic(ev.preview, "EVIDENCE_EXACT_SUBJECT_NOT_FOUND") {
			t.Fatalf("tampered subject kind was not rejected: %+v", ev.preview.Stages)
		}
	})

	t.Run("stale source fingerprint", func(t *testing.T) {
		bad := grounded
		bad.Evidence = append([]Evidence(nil), grounded.Evidence...)
		for i := range bad.Evidence {
			if bad.Evidence[i].SourceFingerprint != "" {
				bad.Evidence[i].SourceFingerprint = "sha256:" + strings.Repeat("0", 64)
				break
			}
		}
		ev, err := evaluate(dir, mustJSON(t, bad))
		if err != nil {
			t.Fatal(err)
		}
		if ev.preview.Acceptable || !previewHasDiagnostic(ev.preview, "EVIDENCE_SOURCE_FINGERPRINT_MISMATCH") {
			t.Fatalf("stale source fingerprint was not rejected: %+v", ev.preview.Stages)
		}
	})

	t.Run("requirements no longer applicable", func(t *testing.T) {
		bad := grounded
		bad.Evidence = append([]Evidence(nil), grounded.Evidence...)
		for i := range bad.Evidence {
			if bad.Evidence[i].QualificationRequirements == nil {
				continue
			}
			requirements := *bad.Evidence[i].QualificationRequirements
			requirements.Application = "repair_intake"
			bad.Evidence[i].QualificationRequirements = &requirements
			break
		}
		ev, err := evaluate(dir, mustJSON(t, bad))
		if err != nil {
			t.Fatal(err)
		}
		if ev.preview.Acceptable || !previewHasDiagnostic(ev.preview, "EVIDENCE_QUALIFICATION_NOT_APPLICABLE") {
			t.Fatalf("no-longer-applicable requirements were not rejected: %+v", ev.preview.Stages)
		}
	})
}

func TestSharedAcceptanceRejectsSourceThatCurrentQualificationDidNotAuthorize(t *testing.T) {
	dir := scratchWorkspace(t)
	snapshot, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}

	target := subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"}
	ctx := knowledgecontext.Build(snapshot.loaded.World, snapshot.index, target)
	qualification, err := knowledgecontext.Qualify(ctx, knowledgecontext.QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
	})
	if err != nil {
		t.Fatal(err)
	}
	grounded, err := attachQualificationEvidence(mustJSON(t, unitPriceCandidate(snapshot.revision)), qualification)
	if err != nil {
		t.Fatal(err)
	}

	var alternate sourceindex.Artifact
	for _, artifact := range snapshot.index.Artifacts {
		if artifact.Path != qualification.Evidence[0].Source {
			alternate = artifact
			break
		}
	}
	if alternate.Path == "" {
		t.Fatal("alternate managed source was not found")
	}

	// Simulate a non-authoritative Reference for the same exact subject.
	// Exact subject/source lookup and fingerprint lookup both succeed, but
	// current qualification must not return this implementation-only source
	// as authoritative Evidence.
	snapshot.index.References = append(snapshot.index.References, sourceindex.Reference{
		Subject:      target,
		ArtifactPath: alternate.Path,
		Role:         sourceindex.RoleImplementation,
	})

	bad := grounded
	bad.Evidence = append([]Evidence(nil), grounded.Evidence...)
	changed := false
	for i := range bad.Evidence {
		if bad.Evidence[i].QualificationRequirements == nil {
			continue
		}
		bad.Evidence[i].Source = alternate.Path
		bad.Evidence[i].SourceFingerprint = alternate.Fingerprint
		changed = true
		break
	}
	if !changed {
		t.Fatal("qualification Evidence was not found")
	}

	diagnostics := resolveCandidate(snapshot, bad)
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "EVIDENCE_QUALIFICATION_PROVENANCE_MISMATCH" {
			return
		}
	}
	t.Fatalf("non-authoritative substituted source was not rejected: %+v", diagnostics)
}

func TestAttachQualificationRejectsNonApplicableResult(t *testing.T) {
	candidate := unitPriceCandidate("sha256:" + strings.Repeat("a", 64))
	for _, status := range []knowledgecontext.QualificationStatus{
		knowledgecontext.QualificationIncompatible,
		knowledgecontext.QualificationUnknown,
	} {
		_, err := attachQualificationEvidence(mustJSON(t, candidate), knowledgecontext.QualificationResult{
			Candidate: subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
			Status:    status,
		})
		if err == nil {
			t.Fatalf("status %s was attached as reusable Evidence", status)
		}
	}
}

func TestCandidateRejectsQualificationRequirementsWithoutExactProvenance(t *testing.T) {
	candidate := unitPriceCandidate("sha256:" + strings.Repeat("a", 64))
	candidate.Evidence = []Evidence{{
		SubjectKind: "application-pattern",
		Subject:     "order_entry.customer_reference",
		QualificationRequirements: &knowledgecontext.QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
		},
	}}
	if _, _, err := normalizeCandidate(candidate); err == nil {
		t.Fatal("qualification requirements without source fingerprint were accepted")
	}
}

func TestAttachQualificationCLIProducesCandidateWithoutWorkspaceMutation(t *testing.T) {
	dir := scratchWorkspace(t)
	snapshot, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := knowledgecontext.Build(
		snapshot.loaded.World,
		snapshot.index,
		subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
	)
	qualification, err := knowledgecontext.Qualify(ctx, knowledgecontext.QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
	})
	if err != nil {
		t.Fatal(err)
	}

	temp := t.TempDir()
	candidatePath := filepath.Join(temp, "candidate.json")
	qualificationPath := filepath.Join(temp, "qualification.json")
	if err := os.WriteFile(candidatePath, mustJSON(t, unitPriceCandidate(snapshot.revision)), 0o644); err != nil {
		t.Fatal(err)
	}
	qualificationJSON, err := json.Marshal(qualification)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(qualificationPath, qualificationJSON, 0o644); err != nil {
		t.Fatal(err)
	}

	before := treeDigest(t, dir)
	var stdout, stderr bytes.Buffer
	code := runCLI(
		[]string{"attach-qualification", "-candidate", candidatePath, "-qualification", qualificationPath},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var grounded Candidate
	if err := json.Unmarshal(stdout.Bytes(), &grounded); err != nil {
		t.Fatal(err)
	}
	if len(grounded.Evidence) == 0 {
		t.Fatalf("CLI output lacks qualification Evidence: %s", stdout.String())
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
}

func previewHasDiagnostic(preview Preview, code string) bool {
	for _, stage := range preview.Stages {
		for _, diagnostic := range stage.Diagnostics {
			if diagnostic.Code == code {
				return true
			}
		}
	}
	return false
}
