package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/workspace"
)

type actorContextR2HumanReviewProbe struct {
	Intent                       string                           `json:"intent"`
	ReferenceFingerprint         string                           `json:"referenceFingerprint"`
	RequiredInputFingerprints    []actorContextR0InputFingerprint `json:"requiredInputFingerprints"`
	CurrentMeaning               []string                         `json:"currentMeaning"`
	ConflictCodes                []string                         `json:"conflictCodes,omitempty"`
	UnresolvedDimensions         []string                         `json:"unresolvedDimensions,omitempty"`
	PotentiallyAffectedAuthority []string                         `json:"potentiallyAffectedAuthority,omitempty"`
	ReviewQuestions              []string                         `json:"reviewQuestions"`
	DrillDownRefs                []string                         `json:"drillDownRefs"`
	AuthorityLimits              []string                         `json:"authorityLimits"`
}

func TestActorContextR2BuildsHighDensityHumanReviewWithoutChoosingResolution(t *testing.T) {
	root := repoPath()
	workspacePath := filepath.Join(root, "reference", "workspace")
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	conflictPath := filepath.Join(root, "reference", "business-system-design", "order-entry-customer-edit-conflict.json")

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceIdx, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}
	r1, err := deriveActorContextR1ConstraintProbe(loaded.World, sourceIdx, f, workspacePath, fixturePath, conflictPath)
	if err != nil {
		t.Fatal(err)
	}

	got := deriveActorContextR2HumanReviewProbe(r1)

	if got.ReferenceFingerprint != r1.ReferenceFingerprint {
		t.Fatalf("R2 did not preserve the R1 Business Reference fingerprint: R1=%s R2=%s", r1.ReferenceFingerprint, got.ReferenceFingerprint)
	}
	if len(got.RequiredInputFingerprints) != len(r1.RequiredInputFingerprints) {
		t.Fatalf("R2 changed the R1 freshness dependency set: R1=%+v R2=%+v", r1.RequiredInputFingerprints, got.RequiredInputFingerprints)
	}
	if !actorContextR0Contains(got.ConflictCodes, "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT") {
		t.Fatalf("R2 lost the decisive current-path conflict: %+v", got)
	}
	for _, unresolved := range []string{
		"customer-update-ownership",
		"write-authority-path",
		"business-responsibility-expansion",
		"capability-or-handoff-design",
	} {
		if !actorContextR0Contains(got.UnresolvedDimensions, unresolved) {
			t.Fatalf("R2 lost unresolved dimension %q: %+v", unresolved, got)
		}
	}
	if !reflect.DeepEqual(got.DrillDownRefs, actorContextR2InputArtifacts(r1.RequiredInputFingerprints)) {
		t.Fatalf("R2 drill-down refs diverged from the R1 provenance inputs: inputs=%+v refs=%+v", r1.RequiredInputFingerprints, got.DrillDownRefs)
	}
	for _, source := range []string{actorContextR0ProcedureArtifact, actorContextR0ConflictArtifact} {
		if !actorContextR0Contains(got.DrillDownRefs, source) {
			t.Fatalf("R2 cannot drill down to decisive Reference input %q: %+v", source, got.DrillDownRefs)
		}
	}
	if len(got.ReviewQuestions) < 4 {
		t.Fatalf("R2 collapsed unresolved design into too few review questions: %+v", got.ReviewQuestions)
	}

	assertActorContextR2NoReviewDecisionShortcut(t, reflect.TypeOf(actorContextR2HumanReviewProbe{}))
}

func TestActorContextR2MarksPotentialAuthorityAsConditionalRatherThanMustChange(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceIdx, err := workspace.BuildSourceIndex(filepath.Join(root, "reference", "workspace"), loaded)
	if err != nil {
		t.Fatal(err)
	}
	r1, err := deriveActorContextR1ConstraintProbe(
		loaded.World,
		sourceIdx,
		f,
		filepath.Join(root, "reference", "workspace"),
		fixturePath,
		filepath.Join(root, "reference", "business-system-design", "order-entry-customer-edit-conflict.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	got := deriveActorContextR2HumanReviewProbe(r1)
	for _, item := range got.PotentiallyAffectedAuthority {
		if item == "must-change:order_entry.master-access" ||
			item == "must-change:master_data.customer" ||
			item == "must-change:BP-02" {
			t.Fatalf("R2 converted a conditional review scope into a mandatory source change: %q", item)
		}
	}
}

func deriveActorContextR2HumanReviewProbe(r1 actorContextR1ConstraintProbe) actorContextR2HumanReviewProbe {
	return actorContextR2HumanReviewProbe{
		Intent:                    r1.ChangeRequest,
		ReferenceFingerprint:      r1.ReferenceFingerprint,
		RequiredInputFingerprints: append([]actorContextR0InputFingerprint(nil), r1.RequiredInputFingerprints...),
		CurrentMeaning: []string{
			"business:" + r1.BusinessStep + ":" + r1.BusinessAction,
			"pattern:" + r1.Pattern + ":" + r1.PatternKind,
			"relation:" + r1.Relation,
			"datasource:" + r1.Datasource + ":access=" + r1.ApplicationAccess,
			"contract:" + r1.Contract + ":query=" + r1.ContractQuery,
			"contract-writes:none",
		},
		ConflictCodes:        append([]string(nil), r1.ConflictCodes...),
		UnresolvedDimensions: append([]string(nil), r1.UnresolvedDimensions...),
		PotentiallyAffectedAuthority: []string{
			"conditional:business-responsibility:" + r1.BusinessStep,
			"conditional:application-datasource-access:order_entry/" + r1.Datasource,
			"conditional:database-contract:" + r1.Contract,
			"conditional:authorization-and-capability-boundary",
		},
		ReviewQuestions: []string{
			"Which subsystem owns customer update responsibility?",
			"Should order_entry receive write authority, or should it hand off?",
			"Which accepted Contract or Capability would authorize mutation?",
			"What Observation or Evidence would establish that the update actually completed?",
		},
		DrillDownRefs: actorContextR2InputArtifacts(r1.RequiredInputFingerprints),
		AuthorityLimits: []string{
			"human-review-projection-is-derived",
			"potentially-affected-does-not-mean-must-change",
			"review-question-does-not-imply-answer",
			"summary-does-not-replace-authoritative-source",
		},
	}
}

func actorContextR2InputArtifacts(values []actorContextR0InputFingerprint) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value.Artifact == "" {
			continue
		}
		if _, ok := seen[value.Artifact]; ok {
			continue
		}
		seen[value.Artifact] = struct{}{}
		out = append(out, value.Artifact)
	}
	sort.Strings(out)
	return out
}

func assertActorContextR2NoReviewDecisionShortcut(t *testing.T, typ reflect.Type) {
	t.Helper()
	for _, forbidden := range []string{
		"Approved",
		"Rejected",
		"Safe",
		"Score",
		"Confidence",
		"Decision",
		"MustChange",
		"AutoApprove",
		"RecommendedResolution",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R2 human review probe introduced forbidden decision shortcut field %q", forbidden)
		}
	}
}
