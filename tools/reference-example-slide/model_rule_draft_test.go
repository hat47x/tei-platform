package exampleslide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/workspace"
)

func modelRuleDraftFixture(t *testing.T) (string, Input, ModelHypotheses, ModelRuleDraftIntent) {
	t.Helper()
	root, input, hypotheses := modelHypothesisAuditFixture(t)
	review, err := ReviewModelHypotheses(input, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := AuditLoadedWorkspaceModelHypothesisSourceEvidence(root, input, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	claims := []ModelRuleEvidenceClaim{{
		Kind:        "document",
		Ref:         "documents/unverified-work-instruction",
		ClaimedBy:   "designer",
		Fingerprint: "sha256:" + strings.Repeat("a", 64),
	}}
	intent := ModelRuleDraftIntent{
		Schema:                 modelRuleDraftIntentSchema,
		ModelReviewFingerprint: review.Fingerprint,
		SourceAuditFingerprint: audit.Fingerprint,
	}
	for _, issue := range review.Issues {
		intent.Requests = append(intent.Requests, ModelRuleDraftRequest{
			IssueID:         issue.ID,
			ProposedRuleID:  issue.Table + "." + strings.ReplaceAll(issue.HypothesisID, "_", "-"),
			Rationale:       "業務上の識別と決定関係の仮説を確認する",
			ClaimedEvidence: append([]ModelRuleEvidenceClaim(nil), claims...),
		})
	}
	return root, input, hypotheses, intent
}

func TestRuleDraftSeparatesTypedDeclarationFromAcceptance(t *testing.T) {
	root, input, hypotheses, intent := modelRuleDraftFixture(t)
	output, err := DraftLoadedWorkspaceModelRules(root, input, hypotheses, intent)
	if err != nil {
		t.Fatal(err)
	}
	if output.Schema != "tei.reference.model-rule-declaration-draft/v0" ||
		!strings.HasPrefix(output.Fingerprint, "sha256:") || len(output.Declarations) != 2 {
		t.Fatalf("malformed model declaration draft: %+v", output)
	}
	for _, d := range output.Declarations {
		if d.Table != "sales.child" || d.Status != "unverified-business-rule" ||
			d.ReviewResult != "unresolved" || len(d.SourceBindings) == 0 ||
			len(d.ClaimedEvidence) != 1 || len(d.Outstanding) < 2 {
			t.Fatalf("source anchor was mistaken for rule acceptance: %+v", d)
		}
		for _, binding := range d.SourceBindings {
			if binding.Status != "source-range-checked" || binding.SourceRange == nil {
				t.Fatalf("lost current source location: %+v", d)
			}
		}
		if d.Kind == "candidate-key" {
			if !reflect.DeepEqual(d.KeyFields, []string{"sales.child.parent_id"}) ||
				len(d.Determinants) != 0 || len(d.Dependents) != 0 {
				t.Fatalf("business identity was confused with FD: %+v", d)
			}
		} else if d.Kind == "functional-dependency" {
			if !reflect.DeepEqual(d.Determinants, []string{"sales.child.parent_id"}) ||
				!reflect.DeepEqual(d.Dependents, []string{"sales.child.parent_id"}) ||
				len(d.KeyFields) != 0 {
				t.Fatalf("FD was confused with candidate key: %+v", d)
			}
		} else {
			t.Fatalf("unknown typed rule kind: %+v", d)
		}
	}
	content, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "\"accepted\"") ||
		strings.Contains(string(content), "\"delta\"") ||
		strings.Contains(string(content), "\"authorized\"") {
		t.Fatalf("draft appeared to contain an accepted semantic mutation: %s", content)
	}
}

func TestRuleDraftKnownCounterexampleCannotBecomeEligible(t *testing.T) {
	root, input, hypotheses, _ := modelRuleDraftFixture(t)
	path := filepath.Join(root, "semantic", "model.tei")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Use a genuinely comparable type for this example. std.identifier
	// stays opaque and must NOT be turned into a fake collision.
	changed := strings.Replace(string(raw), "parentId type std.identifier", "parentId type std.text", 1)
	if err := os.WriteFile(path, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := CatalogFromWorld(loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	input.Catalog = catalog
	input.Example.ModelRevision = catalog.Revision
	second := input.Example.Base[0]
	second.ID = "second-child"
	input.Example.Base = append(input.Example.Base, second)
	review, err := ReviewModelHypotheses(input, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := AuditLoadedWorkspaceModelHypothesisSourceEvidence(root, input, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	intent := ModelRuleDraftIntent{
		Schema:                 modelRuleDraftIntentSchema,
		ModelReviewFingerprint: review.Fingerprint,
		SourceAuditFingerprint: audit.Fingerprint,
	}
	for _, issue := range review.Issues {
		intent.Requests = append(intent.Requests, ModelRuleDraftRequest{
			IssueID:        issue.ID,
			ProposedRuleID: issue.Table + "." + issue.HypothesisID,
			Rationale:      "業務の判断は後続の審査を必要とする",
		})
	}
	out, err := DraftLoadedWorkspaceModelRules(root, input, hypotheses, intent)
	if err != nil {
		t.Fatal(err)
	}
	if out.Declarations[0].Kind != "candidate-key" ||
		out.Declarations[0].ReviewResult != "contradiction" ||
		out.Declarations[0].Status != "blocked-counterexample" {
		t.Fatalf("counterexample did not block key proposal: %+v", out)
	}
}

func TestRuleDraftRejectsStaleReviewOrSourceAuditEvenWithClaimedEvidence(t *testing.T) {
	root, input, hypotheses, intent := modelRuleDraftFixture(t)
	intent.ModelReviewFingerprint = "sha256:" + strings.Repeat("a", 64)
	if _, err := DraftLoadedWorkspaceModelRules(root, input, hypotheses, intent); err == nil {
		t.Fatal("stale review fingerprint accepted")
	}
	root, input, hypotheses, intent = modelRuleDraftFixture(t)
	intent.SourceAuditFingerprint = "sha256:" + strings.Repeat("a", 64)
	if _, err := DraftLoadedWorkspaceModelRules(root, input, hypotheses, intent); err == nil {
		t.Fatal("stale source audit fingerprint accepted")
	}
	root, input, hypotheses, intent = modelRuleDraftFixture(t)
	path := filepath.Join(root, "semantic", "model.tei")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(raw), "attribute parentId", "attribute changedId", 1)
	if err := os.WriteFile(path, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := DraftLoadedWorkspaceModelRules(root, input, hypotheses, intent); err == nil {
		t.Fatal("previous draft was accepted after changing Workspace source")
	}
}

func TestRuleDraftRejectsDuplicateOrForgedRequest(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ModelRuleDraftIntent)
	}{
		{"schema", func(i *ModelRuleDraftIntent) { i.Schema = "not-this-schema" }},
		{"empty", func(i *ModelRuleDraftIntent) { i.Requests = nil }},
		{"unknown issue", func(i *ModelRuleDraftIntent) { i.Requests[0].IssueID = "forged-review-issue" }},
		{"duplicate issue", func(i *ModelRuleDraftIntent) { i.Requests[1].IssueID = i.Requests[0].IssueID }},
		{"duplicate rule", func(i *ModelRuleDraftIntent) { i.Requests[1].ProposedRuleID = i.Requests[0].ProposedRuleID }},
		{"unqualified", func(i *ModelRuleDraftIntent) { i.Requests[0].ProposedRuleID = "business_identity" }},
		{"cross-table", func(i *ModelRuleDraftIntent) { i.Requests[0].ProposedRuleID = "sales.other.business_identity" }},
		{"invalid name", func(i *ModelRuleDraftIntent) { i.Requests[0].ProposedRuleID = "sales.child.Not Valid" }},
		{"missing rationale", func(i *ModelRuleDraftIntent) { i.Requests[0].Rationale = "   " }},
		{"wrong claim", func(i *ModelRuleDraftIntent) { i.Requests[0].ClaimedEvidence[0].Kind = "verified-by-ai" }},
		{"empty claimant", func(i *ModelRuleDraftIntent) { i.Requests[0].ClaimedEvidence[0].ClaimedBy = " " }},
		{"bad claim hash", func(i *ModelRuleDraftIntent) { i.Requests[0].ClaimedEvidence[0].Fingerprint = "sha256:guessed" }},
		{"duplicate claim", func(i *ModelRuleDraftIntent) {
			i.Requests[0].ClaimedEvidence = append(i.Requests[0].ClaimedEvidence, i.Requests[0].ClaimedEvidence[0])
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, input, hypotheses, intent := modelRuleDraftFixture(t)
			tt.edit(&intent)
			if _, err := DraftLoadedWorkspaceModelRules(root, input, hypotheses, intent); err == nil {
				t.Fatal("invalid declaration candidate input accepted")
			}
		})
	}
}

func TestRuleDraftOrderingAndInputAreStable(t *testing.T) {
	root, input, hypotheses, intent := modelRuleDraftFixture(t)
	before, err := json.Marshal(struct {
		Input      Input
		Hypotheses ModelHypotheses
		Intent     ModelRuleDraftIntent
	}{input, hypotheses, intent})
	if err != nil {
		t.Fatal(err)
	}
	left, err := DraftLoadedWorkspaceModelRules(root, input, hypotheses, intent)
	if err != nil {
		t.Fatal(err)
	}
	intent.Requests[0], intent.Requests[1] = intent.Requests[1], intent.Requests[0]
	right, err := DraftLoadedWorkspaceModelRules(root, input, hypotheses, intent)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("order changed draft identity: %+v vs %+v", left, right)
	}
	intent.Requests[0], intent.Requests[1] = intent.Requests[1], intent.Requests[0]
	after, err := json.Marshal(struct {
		Input      Input
		Hypotheses ModelHypotheses
		Intent     ModelRuleDraftIntent
	}{input, hypotheses, intent})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("draft generation mutated input")
	}
}
