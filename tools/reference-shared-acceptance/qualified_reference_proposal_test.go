package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/knowledgecontext"
	"tei/internal/subject"
)

func qualifiedCustomerReference(t *testing.T, dir string) knowledgecontext.QualificationResult {
	t.Helper()
	snapshot, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := knowledgecontext.Build(
		snapshot.loaded.World,
		snapshot.index,
		subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
	)
	qualification, err := knowledgecontext.QualifyWithSources(
		ctx,
		snapshot.index,
		knowledgecontext.QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
			RequiredDependencies: []knowledgecontext.DependencyRequirement{{
				Direction: "outgoing",
				Relation:  "contract-query",
				Kind:      "database-contract-query",
				ID:        "master_data.customer/find_by_id",
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if qualification.Status != knowledgecontext.QualificationApplicable {
		t.Fatalf("qualification=%+v", qualification)
	}
	return qualification
}

func qualifiedReferenceReuseRequest() QualifiedReferenceReuse {
	return QualifiedReferenceReuse{
		Schema:      qualifiedReferenceReuseSchema,
		PatternID:   "repair_intake.customer_reference",
		Application: "repair_intake",
		Relation:    "repair_intake.intake.customer",
		Datasource:  "customer_master",
		Placement:   s4PatternPath,
	}
}

func secondApplicationBaseWithoutPattern(revision string) Candidate {
	candidate := secondApplicationCandidate(revision)
	candidate.Delta = append([]Operation(nil), candidate.Delta[:2]...)
	candidate.Evidence = []Evidence{{Subject: "customer.customer"}}
	return candidate
}

func TestQualifiedPatternBuildsExplicitDestinationReferenceProposal(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	revision := revisionOfDir(t, dir)
	base := secondApplicationBaseWithoutPattern(revision)
	qualification := qualifiedCustomerReference(t, dir)
	request := qualifiedReferenceReuseRequest()

	proposed, err := proposeQualifiedReferencePattern(
		dir,
		mustJSON(t, base),
		mustJSON(t, qualification),
		mustJSON(t, request),
	)
	if err != nil {
		t.Fatal(err)
	}

	got, gotFingerprint, err := normalizeCandidate(proposed)
	if err != nil {
		t.Fatal(err)
	}
	want, wantFingerprint, err := normalizeCandidate(secondApplicationCandidate(revision))
	if err != nil {
		t.Fatal(err)
	}
	if gotFingerprint != wantFingerprint || !reflect.DeepEqual(got.Delta, want.Delta) {
		t.Fatalf(
			"qualified Proposalが既知のS4 semantic deltaと一致しません:\ngot=%+v\nwant=%+v\nfp=%s/%s",
			got.Delta,
			want.Delta,
			gotFingerprint,
			wantFingerprint,
		)
	}

	var pattern *PatternSpec
	for _, operation := range got.Delta {
		if operation.Op == opAddReferencePattern {
			pattern = operation.Pattern
			break
		}
	}
	if pattern == nil {
		t.Fatal("qualified Proposalにadd-reference-patternがありません")
	}
	if pattern.ID != request.PatternID ||
		pattern.Application != request.Application ||
		pattern.Relation != request.Relation ||
		pattern.Datasource != request.Datasource {
		t.Fatalf("destination-owned fields were not preserved: %+v", pattern)
	}
	if pattern.Relation == "sales.order.customer" || pattern.Datasource == "master" {
		t.Fatalf("origin-owned Relation / datasource were copied: %+v", pattern)
	}
	if pattern.ContractQuery.Contract != "master_data.customer" ||
		pattern.ContractQuery.Query != "find_by_id" ||
		len(pattern.ContractQuery.Inputs) != 1 ||
		pattern.ContractQuery.Inputs[0].Attribute != "customer.customer.customer_id" ||
		pattern.ContractQuery.Inputs[0].Input != "id" {
		t.Fatalf("qualified shared Contract Query selection was not preserved: %+v", pattern.ContractQuery)
	}

	var grounded bool
	for _, evidence := range got.Evidence {
		if evidence.SubjectKind == "application-pattern" &&
			evidence.Subject == "order_entry.customer_reference" &&
			evidence.QualificationRequirements != nil {
			for _, dependency := range evidence.QualificationDependencyEvidence {
				if dependency.SubjectKind == "database-contract-query" &&
					dependency.Subject == "master_data.customer/find_by_id" {
					grounded = true
				}
			}
		}
	}
	if !grounded {
		t.Fatalf("origin Pattern / required Contract Query Evidence is missing: %+v", got.Evidence)
	}

	ev, err := evaluate(dir, mustJSON(t, got))
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("qualified Proposal did not enter Shared Acceptance: %+v", allDiagnostics(ev.preview))
	}
}

func TestQualifiedPatternProposalRequiresExplicitContractQueryQualification(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	snapshot, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := knowledgecontext.Build(
		snapshot.loaded.World,
		snapshot.index,
		subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
	)
	qualification, err := knowledgecontext.QualifyWithSources(
		ctx,
		snapshot.index,
		knowledgecontext.QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if qualification.Status != knowledgecontext.QualificationApplicable {
		t.Fatalf("qualification=%+v", qualification)
	}

	_, err = proposeQualifiedReferencePattern(
		dir,
		mustJSON(t, secondApplicationBaseWithoutPattern(snapshot.revision)),
		mustJSON(t, qualification),
		mustJSON(t, qualifiedReferenceReuseRequest()),
	)
	if err == nil || !strings.Contains(err.Error(), "requiredDependencies") {
		t.Fatalf("Contract Queryを明示qualificationせずProposalを生成しました: %v", err)
	}
}

func TestQualifiedPatternProposalRejectsStaleQualificationEvidence(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	revision := revisionOfDir(t, dir)
	qualification := qualifiedCustomerReference(t, dir)
	if len(qualification.Evidence) == 0 {
		t.Fatal("qualification Evidenceがありません")
	}
	qualification.Evidence[0].SourceFingerprint = "sha256:" + strings.Repeat("0", 64)

	_, err := proposeQualifiedReferencePattern(
		dir,
		mustJSON(t, secondApplicationBaseWithoutPattern(revision)),
		mustJSON(t, qualification),
		mustJSON(t, qualifiedReferenceReuseRequest()),
	)
	if err == nil || !strings.Contains(err.Error(), "exact Evidence") {
		t.Fatalf("stale qualificationからProposalを生成しました: %v", err)
	}
}

func TestQualifiedReferenceReuseRequestCannotInjectContractQuery(t *testing.T) {
	data := []byte(`{
	  "schema": "tei.reference.qualified-reference-reuse/v0",
	  "patternId": "repair_intake.customer_reference",
	  "application": "repair_intake",
	  "relation": "repair_intake.intake.customer",
	  "datasource": "customer_master",
	  "placement": "applications/repair-intake/patterns/customer-reference.tei",
	  "contractQuery": {"contract": "other.contract", "query": "other"}
	}`)
	if _, err := decodeQualifiedReferenceReuse(data); err == nil {
		t.Fatal("destination requestからContract Queryを注入できています")
	}
}

func TestKnowledgeQualifyCLIProducesExplicitReferenceProposal(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	snapshot, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	before := treeDigest(t, dir)
	temp := t.TempDir()
	writeJSON := func(name string, value any) string {
		t.Helper()
		path := filepath.Join(temp, name)
		if err := os.WriteFile(path, mustJSON(t, value), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
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
	requirementsPath := writeJSON("qualification-requirements.json", requirements)
	candidatePath := writeJSON("base-candidate.json", secondApplicationBaseWithoutPattern(snapshot.revision))
	requestPath := writeJSON("reference-reuse.json", qualifiedReferenceReuseRequest())
	qualificationPath := filepath.Join(temp, "qualification.json")

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
	if err := os.WriteFile(qualificationPath, qualificationOut.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := cliRun(
		t,
		"propose-qualified-reference-pattern",
		"-workspace", dir,
		"-candidate", candidatePath,
		"-qualification", qualificationPath,
		"-reference-reuse", requestPath,
	)
	if code != 0 {
		t.Fatalf("proposal CLI code=%d err=%s out=%s", code, errOut, out)
	}
	proposed, err := decodeCandidate([]byte(out))
	if err != nil {
		t.Fatalf("proposal CLI output is not a candidate: %v\n%s", err, out)
	}
	_, gotFingerprint, err := normalizeCandidate(proposed)
	if err != nil {
		t.Fatal(err)
	}
	_, wantFingerprint, err := normalizeCandidate(secondApplicationCandidate(snapshot.revision))
	if err != nil {
		t.Fatal(err)
	}
	if gotFingerprint != wantFingerprint {
		t.Fatalf("end-to-end candidate fingerprint=%s want=%s", gotFingerprint, wantFingerprint)
	}
	ev, err := evaluate(dir, mustJSON(t, proposed))
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("end-to-end Proposal did not pass Shared Acceptance Preview: %+v", allDiagnostics(ev.preview))
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
}

func TestProposeQualifiedReferencePatternCLILeavesWorkspaceReadOnly(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	revision := revisionOfDir(t, dir)
	before := treeDigest(t, dir)
	temp := t.TempDir()
	writeJSON := func(name string, value any) string {
		t.Helper()
		path := filepath.Join(temp, name)
		if err := os.WriteFile(path, mustJSON(t, value), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	candidatePath := writeJSON("candidate.json", secondApplicationBaseWithoutPattern(revision))
	qualificationPath := writeJSON("qualification.json", qualifiedCustomerReference(t, dir))
	requestPath := writeJSON("reference-reuse.json", qualifiedReferenceReuseRequest())

	code, out, errOut := cliRun(
		t,
		"propose-qualified-reference-pattern",
		"-workspace", dir,
		"-candidate", candidatePath,
		"-qualification", qualificationPath,
		"-reference-reuse", requestPath,
	)
	if code != 0 {
		t.Fatalf("propose-qualified-reference-pattern code=%d err=%s out=%s", code, errOut, out)
	}
	proposed, err := decodeCandidate([]byte(out))
	if err != nil {
		t.Fatalf("CLI output is not a candidate: %v\n%s", err, out)
	}
	_, gotFingerprint, err := normalizeCandidate(proposed)
	if err != nil {
		t.Fatal(err)
	}
	_, wantFingerprint, err := normalizeCandidate(secondApplicationCandidate(revision))
	if err != nil {
		t.Fatal(err)
	}
	if gotFingerprint != wantFingerprint {
		t.Fatalf("CLI Proposal fingerprint=%s want=%s", gotFingerprint, wantFingerprint)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
}
