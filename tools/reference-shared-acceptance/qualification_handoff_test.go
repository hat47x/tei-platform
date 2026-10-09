package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
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
	qualification, err := knowledgecontext.QualifyWithSources(ctx, snapshot.index, knowledgecontext.QualificationRequirements{
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
	var foundDependencyEvidence bool
	for _, evidence := range grounded.Evidence {
		for _, dependency := range evidence.QualificationDependencyEvidence {
			if dependency.SubjectKind == "database-contract-query" &&
				dependency.Subject == "master_data.customer/find_by_id" &&
				dependency.Source != "" &&
				strings.HasPrefix(dependency.SourceFingerprint, "sha256:") {
				foundDependencyEvidence = true
			}
		}
	}
	if !foundDependencyEvidence {
		t.Fatalf("required dependency Evidence was not retained: %+v", grounded.Evidence)
	}

	ev, err := evaluate(dir, mustJSON(t, grounded))
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("qualified candidate did not pass Shared Acceptance preview: %+v", ev.preview.Stages)
	}
}

func TestBuilderReviewHandoffQualificationReusesExistingSharedAcceptanceEvidencePath(t *testing.T) {
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
	requirements := knowledgecontext.QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
		RequiredDependencies: []knowledgecontext.DependencyRequirement{{
			Direction: "outgoing",
			Relation:  "contract-query",
			Kind:      "database-contract-query",
			ID:        "master_data.customer/find_by_id",
		}},
	}

	direct, err := knowledgecontext.QualifyWithSources(ctx, snapshot.index, requirements)
	if err != nil {
		t.Fatal(err)
	}
	handoff, err := knowledgecontext.BuildBuilderReviewHandoff(ctx, snapshot.index, requirements)
	if err != nil {
		t.Fatal(err)
	}
	if handoff.Status != "qualified-for-disclosure" {
		t.Fatalf("builder review handoff status=%q", handoff.Status)
	}
	if !bytes.Equal(mustJSON(t, direct), mustJSON(t, handoff.Qualification)) {
		t.Fatalf(
			"builder review handoff changed existing QualificationResult:\ndirect=%+v\nhandoff=%+v",
			direct,
			handoff.Qualification,
		)
	}

	candidateData := mustJSON(t, unitPriceCandidate(snapshot.revision))
	directGrounded, err := attachQualificationEvidence(candidateData, direct)
	if err != nil {
		t.Fatal(err)
	}
	handoffGrounded, err := attachQualificationEvidence(candidateData, handoff.Qualification)
	if err != nil {
		t.Fatal(err)
	}
	directNormalized, directMeaning, err := normalizeCandidate(directGrounded)
	if err != nil {
		t.Fatal(err)
	}
	handoffNormalized, handoffMeaning, err := normalizeCandidate(handoffGrounded)
	if err != nil {
		t.Fatal(err)
	}
	if directMeaning != handoffMeaning {
		t.Fatalf(
			"builder review path changed candidate meaning: direct=%s handoff=%s",
			directMeaning,
			handoffMeaning,
		)
	}
	if !bytes.Equal(mustJSON(t, directNormalized), mustJSON(t, handoffNormalized)) {
		t.Fatalf(
			"builder review path produced a different grounded candidate:\ndirect=%+v\nhandoff=%+v",
			directNormalized.Evidence,
			handoffNormalized.Evidence,
		)
	}

	directEvaluation, err := evaluate(dir, mustJSON(t, directNormalized))
	if err != nil {
		t.Fatal(err)
	}
	handoffEvaluation, err := evaluate(dir, mustJSON(t, handoffNormalized))
	if err != nil {
		t.Fatal(err)
	}
	if !directEvaluation.preview.Acceptable || !handoffEvaluation.preview.Acceptable {
		t.Fatalf(
			"equivalent qualification paths diverged at Shared Acceptance preview: direct=%v handoff=%v",
			directEvaluation.preview.Stages,
			handoffEvaluation.preview.Stages,
		)
	}
	if directEvaluation.preview.CandidateFingerprint != handoffEvaluation.preview.CandidateFingerprint ||
		directEvaluation.preview.EvidenceFingerprint != handoffEvaluation.preview.EvidenceFingerprint {
		t.Fatalf(
			"builder review bridge changed Shared Acceptance identity: direct=(%s,%s) handoff=(%s,%s)",
			directEvaluation.preview.CandidateFingerprint,
			directEvaluation.preview.EvidenceFingerprint,
			handoffEvaluation.preview.CandidateFingerprint,
			handoffEvaluation.preview.EvidenceFingerprint,
		)
	}
}

func TestAttachQualificationIsIdempotentAfterEvidenceNormalization(t *testing.T) {
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
	qualification, err := knowledgecontext.QualifyWithSources(ctx, snapshot.index, knowledgecontext.QualificationRequirements{
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
	first, err := attachQualificationEvidence(
		mustJSON(t, unitPriceCandidate(snapshot.revision)),
		qualification,
	)
	if err != nil {
		t.Fatal(err)
	}

	var qualificationEvidence Evidence
	found := false
	for _, evidence := range first.Evidence {
		if evidence.QualificationRequirements != nil {
			qualificationEvidence = evidence
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("qualification Evidence was not attached: %+v", first.Evidence)
	}
	qualificationEvidence.Source = " " + qualificationEvidence.Source + " "
	if len(qualificationEvidence.QualificationDependencyEvidence) > 0 {
		qualificationEvidence.QualificationDependencyEvidence = append(
			qualificationEvidence.QualificationDependencyEvidence,
			qualificationEvidence.QualificationDependencyEvidence[0],
		)
	}
	perturbed := first
	perturbed.Evidence = append(append([]Evidence(nil), first.Evidence...), qualificationEvidence)

	second, err := attachQualificationEvidence(mustJSON(t, perturbed), qualification)
	if err != nil {
		t.Fatal(err)
	}
	normalizedFirst, _, err := normalizeCandidate(first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(mustJSON(t, normalizedFirst), mustJSON(t, second)) {
		t.Fatalf("equivalent qualification reattachment was not idempotent:\nfirst=%+v\nsecond=%+v", normalizedFirst.Evidence, second.Evidence)
	}
}

func TestKnowledgeQualifyCLIOutputFlowsThroughAttachmentAndPreview(t *testing.T) {
	dir := scratchWorkspace(t)
	snapshot, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	before := treeDigest(t, dir)

	temp := t.TempDir()
	requirementsPath := filepath.Join(temp, "qualification-requirements.json")
	candidatePath := filepath.Join(temp, "candidate.json")
	qualificationPath := filepath.Join(temp, "qualification.json")
	groundedPath := filepath.Join(temp, "grounded-candidate.json")

	requirements := knowledgecontext.QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
		RequiredDependencies: []knowledgecontext.DependencyRequirement{{
			Direction: "outgoing",
			Relation:  "contract-query",
			Kind:      "database-contract-query",
			ID:        "master_data.customer/find_by_id",
		}},
	}
	if err := os.WriteFile(requirementsPath, mustJSON(t, requirements), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidatePath, mustJSON(t, unitPriceCandidate(snapshot.revision)), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		"go", "run", "./cmd/tei", "knowledge-qualify",
		"-workspace", dir,
		"-subject-kind", "application-pattern",
		"-subject-id", "order_entry.customer_reference",
		"-qualification-requirements", requirementsPath,
	)
	cmd.Dir = repositoryRoot(t)
	cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
	var qualificationOut, qualificationErr bytes.Buffer
	cmd.Stdout = &qualificationOut
	cmd.Stderr = &qualificationErr
	if err := cmd.Run(); err != nil {
		t.Fatalf("tei knowledge-qualify failed: %v\nstderr=%s", err, qualificationErr.String())
	}
	qualification, err := decodeQualificationResult(qualificationOut.Bytes())
	if err != nil {
		t.Fatalf("actual tei knowledge-qualify output is not accepted by handoff decoder: %v\noutput=%s", err, qualificationOut.String())
	}
	if qualification.Status != knowledgecontext.QualificationApplicable || len(qualification.Evidence) == 0 {
		t.Fatalf("actual qualification lacks applicable exact Evidence: %+v", qualification)
	}
	if err := os.WriteFile(qualificationPath, qualificationOut.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	var attachOut, attachErr bytes.Buffer
	if code := runCLI(
		[]string{"attach-qualification", "-candidate", candidatePath, "-qualification", qualificationPath},
		&attachOut,
		&attachErr,
	); code != 0 {
		t.Fatalf("attach-qualification code=%d stderr=%s", code, attachErr.String())
	}
	grounded, err := decodeCandidate(attachOut.Bytes())
	if err != nil {
		t.Fatalf("attach-qualification output is not a candidate: %v\noutput=%s", err, attachOut.String())
	}
	if len(grounded.Evidence) == 0 {
		t.Fatalf("grounded candidate lacks Evidence: %+v", grounded)
	}
	if err := os.WriteFile(groundedPath, attachOut.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	var previewOut, previewErr bytes.Buffer
	if code := runCLI(
		[]string{"preview", "-workspace", dir, "-candidate", groundedPath},
		&previewOut,
		&previewErr,
	); code != 0 {
		t.Fatalf("preview code=%d stderr=%s output=%s", code, previewErr.String(), previewOut.String())
	}
	var preview Preview
	if err := json.Unmarshal(previewOut.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if !preview.Acceptable {
		t.Fatalf("actual qualification handoff did not survive Shared Acceptance preview: %+v", preview.Stages)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
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
		// kindの食い違いは、候補の正規化（qualificationRequirementsとの整合検査）が先に止める。
		// 整合を保ったまま改ざんしてもApplication scopeの検査に落ちるため、どちらの層でも受理されなければよい。
		if ev.preview.Acceptable ||
			!(previewHasDiagnostic(ev.preview, "CANDIDATE_NORMALIZE") || previewHasDiagnostic(ev.preview, "EVIDENCE_EXACT_SUBJECT_NOT_FOUND")) {
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

func TestSharedAcceptanceRejectsStaleRequiredDependencyEvidence(t *testing.T) {
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
	qualification, err := knowledgecontext.QualifyWithSources(ctx, snapshot.index, knowledgecontext.QualificationRequirements{
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
	if qualification.Status != knowledgecontext.QualificationApplicable || len(qualification.DependencyEvidence) == 0 {
		t.Fatalf("qualification=%+v", qualification)
	}
	grounded, err := attachQualificationEvidence(mustJSON(t, unitPriceCandidate(snapshot.revision)), qualification)
	if err != nil {
		t.Fatal(err)
	}

	bad := grounded
	bad.Evidence = append([]Evidence(nil), grounded.Evidence...)
	changed := false
	for i := range bad.Evidence {
		if len(bad.Evidence[i].QualificationDependencyEvidence) == 0 {
			continue
		}
		bad.Evidence[i].QualificationDependencyEvidence = append(
			[]knowledgecontext.QualificationEvidence(nil),
			bad.Evidence[i].QualificationDependencyEvidence...,
		)
		bad.Evidence[i].QualificationDependencyEvidence[0].SourceFingerprint = "sha256:" + strings.Repeat("0", 64)
		changed = true
		break
	}
	if !changed {
		t.Fatal("qualification dependency Evidence was not found")
	}

	ev, err := evaluate(dir, mustJSON(t, bad))
	if err != nil {
		t.Fatal(err)
	}
	if ev.preview.Acceptable || !previewHasDiagnostic(ev.preview, "EVIDENCE_QUALIFICATION_DEPENDENCY_STALE") {
		t.Fatalf("stale required dependency Evidence was not rejected: %+v", ev.preview.Stages)
	}
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

func TestAttachQualificationRejectsMalformedRequirements(t *testing.T) {
	candidate := unitPriceCandidate("sha256:" + strings.Repeat("a", 64))
	qualification := knowledgecontext.QualificationResult{
		Candidate: subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
		Status:    knowledgecontext.QualificationApplicable,
		Requirements: knowledgecontext.QualificationRequirements{
			ExpectedKind: "",
		},
		Evidence: []knowledgecontext.QualificationEvidence{{
			SubjectKind:       "application-pattern",
			Subject:           "order_entry.customer_reference",
			Source:            "applications/order-entry/patterns/customer-reference.tei",
			SourceFingerprint: "sha256:" + strings.Repeat("b", 64),
		}},
	}
	if _, err := attachQualificationEvidence(mustJSON(t, candidate), qualification); err == nil {
		t.Fatal("malformed qualification requirements were attached")
	}
}

func TestCandidateRejectsQualificationKindMismatch(t *testing.T) {
	candidate := unitPriceCandidate("sha256:" + strings.Repeat("a", 64))
	candidate.Evidence = []Evidence{{
		SubjectKind:       "application-pattern",
		Subject:           "order_entry.customer_reference",
		Source:            "applications/order-entry/patterns/customer-reference.tei",
		SourceFingerprint: "sha256:" + strings.Repeat("b", 64),
		QualificationRequirements: &knowledgecontext.QualificationRequirements{
			ExpectedKind: "semantic-data",
		},
	}}
	if _, _, err := normalizeCandidate(candidate); err == nil {
		t.Fatal("qualification expectedKind mismatch was accepted")
	}
}

func TestCandidateRejectsDependencyEvidenceOutsideExplicitRequirements(t *testing.T) {
	base := Evidence{
		SubjectKind:       "application-pattern",
		Subject:           "order_entry.customer_reference",
		Source:            "applications/order_entry/patterns/customer_reference.tei",
		SourceFingerprint: "sha256:" + strings.Repeat("b", 64),
		QualificationRequirements: &knowledgecontext.QualificationRequirements{
			ExpectedKind: "application-pattern",
			RequiredDependencies: []knowledgecontext.DependencyRequirement{{
				Direction: "outgoing",
				Relation:  "contract-query",
				Kind:      "database-contract-query",
				ID:        "master_data.customer/find_by_id",
			}},
		},
	}
	candidate := unitPriceCandidate("sha256:" + strings.Repeat("a", 64))
	candidate.Evidence = []Evidence{base}
	if _, _, err := normalizeCandidate(candidate); err == nil {
		t.Fatal("required dependency without authoritative Evidence was normalized")
	}

	extra := base
	extra.QualificationDependencyEvidence = []knowledgecontext.QualificationEvidence{{
		SubjectKind:       "semantic-data",
		Subject:           "customer.customer",
		Source:            "semantic/customer/customer.tei",
		SourceFingerprint: "sha256:" + strings.Repeat("c", 64),
	}}
	candidate.Evidence = []Evidence{extra}
	if _, _, err := normalizeCandidate(candidate); err == nil {
		t.Fatal("dependency Evidence outside explicit requirements was normalized")
	}
}

func TestCandidateNormalizesQualificationRequirements(t *testing.T) {
	candidate := unitPriceCandidate("sha256:" + strings.Repeat("a", 64))
	candidate.Evidence = []Evidence{{
		SubjectKind:       "application-pattern",
		Subject:           "order_entry.customer_reference",
		Source:            "applications/order-entry/patterns/customer-reference.tei",
		SourceFingerprint: "sha256:" + strings.Repeat("b", 64),
		QualificationRequirements: &knowledgecontext.QualificationRequirements{
			ExpectedKind: " application-pattern ",
			Application:  " order_entry ",
			RequiredDependencies: []knowledgecontext.DependencyRequirement{
				{
					Direction: " OUTGOING ",
					Relation:  " contract-query ",
					Kind:      " database-contract-query ",
					ID:        " z.query ",
					Alias:     " z ",
				},
				{
					Direction: " incoming ",
					Relation:  " depends-on ",
					Kind:      " semantic-data ",
					ID:        " a.data ",
				},
			},
		},
		QualificationDependencyEvidence: []knowledgecontext.QualificationEvidence{
			{
				SubjectKind:       " database-contract-query ",
				Subject:           " z.query ",
				Source:            " database-contracts/z.tei ",
				SourceFingerprint: "sha256:" + strings.Repeat("c", 64),
			},
			{
				SubjectKind:       " semantic-data ",
				Subject:           " a.data ",
				Source:            " semantic/a/data.tei ",
				SourceFingerprint: "sha256:" + strings.Repeat("d", 64),
			},
		},
	}}

	normalized, _, err := normalizeCandidate(candidate)
	if err != nil {
		t.Fatal(err)
	}
	requirements := normalized.Evidence[0].QualificationRequirements
	if requirements == nil {
		t.Fatal("qualification requirements disappeared during normalization")
	}
	if requirements.ExpectedKind != "application-pattern" || requirements.Application != "order_entry" {
		t.Fatalf("requirements were not trimmed: %+v", requirements)
	}
	if len(requirements.RequiredDependencies) != 2 {
		t.Fatalf("dependencies=%+v", requirements.RequiredDependencies)
	}
	if len(normalized.Evidence[0].QualificationDependencyEvidence) != 2 {
		t.Fatalf("dependency Evidence=%+v", normalized.Evidence[0].QualificationDependencyEvidence)
	}
	for _, evidence := range normalized.Evidence[0].QualificationDependencyEvidence {
		if strings.TrimSpace(evidence.SubjectKind) != evidence.SubjectKind ||
			strings.TrimSpace(evidence.Subject) != evidence.Subject ||
			strings.TrimSpace(evidence.Source) != evidence.Source {
			t.Fatalf("dependency Evidence was not normalized: %+v", evidence)
		}
	}
	first, second := requirements.RequiredDependencies[0], requirements.RequiredDependencies[1]
	if first.Direction != "incoming" ||
		first.Relation != "depends-on" ||
		first.Kind != "semantic-data" ||
		first.ID != "a.data" ||
		second.Direction != "outgoing" ||
		second.Relation != "contract-query" ||
		second.Kind != "database-contract-query" ||
		second.ID != "z.query" ||
		second.Alias != "z" {
		t.Fatalf("qualification requirements were not normalized deterministically: %+v", requirements.RequiredDependencies)
	}
}

func TestAttachQualificationRequiresExactEvidenceForEveryExplicitDependency(t *testing.T) {
	candidate := unitPriceCandidate("sha256:" + strings.Repeat("a", 64))
	base := knowledgecontext.QualificationResult{
		Candidate: subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
		Status:    knowledgecontext.QualificationApplicable,
		Requirements: knowledgecontext.QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
			RequiredDependencies: []knowledgecontext.DependencyRequirement{{
				Direction: "outgoing",
				Relation:  "contract-query",
				Kind:      "database-contract-query",
				ID:        "master_data.customer/find_by_id",
			}},
		},
		Evidence: []knowledgecontext.QualificationEvidence{{
			SubjectKind:       "application-pattern",
			Subject:           "order_entry.customer_reference",
			Source:            "applications/order_entry/patterns/customer_reference.tei",
			SourceFingerprint: "sha256:" + strings.Repeat("b", 64),
		}},
	}
	if _, err := attachQualificationEvidence(mustJSON(t, candidate), base); err == nil {
		t.Fatal("required dependency without authoritative Evidence was attached")
	}

	extra := base
	extra.Requirements.RequiredDependencies = nil
	extra.DependencyEvidence = []knowledgecontext.QualificationEvidence{{
		SubjectKind:       "database-contract-query",
		Subject:           "master_data.customer/find_by_id",
		Source:            "database-contracts/master_data/customer.tei",
		SourceFingerprint: "sha256:" + strings.Repeat("c", 64),
	}}
	if _, err := attachQualificationEvidence(mustJSON(t, candidate), extra); err == nil {
		t.Fatal("dependency Evidence outside explicit requirements was attached")
	}
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

func TestCandidateExactEvidenceAllowsSourceIndexCompositeIdentity(t *testing.T) {
	candidate := unitPriceCandidate("sha256:" + strings.Repeat("a", 64))
	candidate.Evidence = []Evidence{{
		SubjectKind: "database-contract-query",
		Subject:     "master_data.customer/find_by_id",
	}}
	if _, _, err := normalizeCandidate(candidate); err != nil {
		t.Fatalf("exact Source Index identity was rejected: %v", err)
	}

	candidate.Evidence[0].SubjectKind = ""
	if _, _, err := normalizeCandidate(candidate); err == nil {
		t.Fatal("legacy subject without kind bypassed Canonical ID validation")
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
