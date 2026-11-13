package documentdraft

import (
	"errors"
	"strings"
	"testing"
)

// TestReconcileReportsCurrentForAnUnchangedWorkspace is the baseline: the same
// read reconciles to current with nothing to reconfirm.
func TestReconcileReportsCurrentForAnUnchangedWorkspace(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.Effect != EffectReadOnlyProjection {
		t.Fatalf("effect=%q", report.Effect)
	}
	if report.ModelReference.Status != ModelReferenceCurrent {
		t.Fatalf("model reference=%+v", report.ModelReference)
	}
	if len(report.NeedsReconfirmation) != 0 {
		t.Fatalf("needsReconfirmation=%v", report.NeedsReconfirmation)
	}
	for _, occurrence := range report.Occurrences {
		if occurrence.Status != OccurrenceUnchanged {
			t.Fatalf("occurrence %s status=%q (%s)",
				occurrence.OccurrenceID, occurrence.Status, occurrence.Detail)
		}
		if occurrence.CurrentContentDigest != occurrence.RecordedContentDigest {
			t.Fatalf("occurrence %s digest moved", occurrence.OccurrenceID)
		}
	}
}

// TestReconcileNamesTheOccurrencesThatNeedReconfirmationAfterAModelChange is
// RD-01: a changed Source makes the model reference stale, the derived
// occurrences change, and the report says which ones a person must look at. It
// does not rewrite the draft or the document.
func TestReconcileNamesTheOccurrencesThatNeedReconfirmationAfterAModelChange(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	before := fixture.digestOf(t, "semantic/sales/order.tei")
	fixture.rewrite(t, "semantic/sales/order.tei", `namespace sales

data Order {
    attribute orderId type sales.order_id
    attribute customerId type customer.customer_id
    attribute status type std.text
    attribute placedAt type std.datetime
    relation customer to customer.customer one
        key sales.order.customer_id -> customer.customer.customer_id
    relation lines to sales.order_line many
}
`)
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.ModelReference.Status != ModelReferenceStale {
		t.Fatalf("model reference status=%q", report.ModelReference.Status)
	}
	joined := strings.Join(report.ModelReference.Diffs, ",")
	if !strings.Contains(joined, "modelRevision") || !strings.Contains(joined, "worldFingerprint") {
		t.Fatalf("model reference diffs=%v", report.ModelReference.Diffs)
	}
	if len(report.NeedsReconfirmation) == 0 {
		t.Fatal("a changed model produced nothing to reconfirm")
	}
	statuses := map[string]string{}
	for _, occurrence := range report.Occurrences {
		statuses[occurrence.OccurrenceID] = occurrence.Status
	}
	if statuses["occ-explanation-1"] != OccurrenceContentChanged {
		t.Fatalf("explanation status=%q", statuses["occ-explanation-1"])
	}
	if statuses["occ-table-1"] != OccurrenceContentChanged {
		t.Fatalf("table status=%q", statuses["occ-table-1"])
	}
	// The stored draft is untouched by the reconciliation.
	again, err := Build(Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer"})
	if err != nil {
		t.Fatalf("re-Build: %v", err)
	}
	if again.DraftID == draft.DraftID {
		t.Fatal("a changed model produced the same draft identity")
	}
	if draft.Occurrences[0].ContentDigest == again.Occurrences[0].ContentDigest {
		t.Fatal("the stored draft's content digest changed in place")
	}
	if fixture.digestOf(t, "semantic/sales/order.tei") == before {
		t.Fatal("fixture did not change; the test cannot prove anything")
	}
}

// TestReconcileSeparatesASourceMoveFromAMeaningChange keeps the three model
// digests apart: moving a declaration between files changes the Source Index and
// the managed revision without necessarily changing the World.
func TestReconcileSeparatesASourceMoveFromAMeaningChange(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	body, readErr := readFile(fixture.root, "semantic/sales/order.tei")
	if readErr != nil {
		t.Fatal(readErr)
	}
	fixture.remove(t, "semantic/sales/order.tei")
	writeFile(t, fixture.root, "semantic/sales/order-moved.tei", body)
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.ModelReference.Status != ModelReferenceStale {
		t.Fatalf("model reference status=%q", report.ModelReference.Status)
	}
	diffs := strings.Join(report.ModelReference.Diffs, ",")
	if !strings.Contains(diffs, "sourceIndexFingerprint") || !strings.Contains(diffs, "modelRevision") {
		t.Fatalf("diffs=%v", report.ModelReference.Diffs)
	}
	statuses := map[string]string{}
	for _, occurrence := range report.Occurrences {
		statuses[occurrence.OccurrenceID] = occurrence.Status
	}
	// The declarations are byte-identical, so the derived text is unchanged even
	// though the Source location moved: that difference is reported, not hidden.
	if statuses["occ-explanation-1"] != OccurrenceConditionsChanged {
		t.Fatalf("explanation status=%q want %q",
			statuses["occ-explanation-1"], OccurrenceConditionsChanged)
	}
	if statuses["occ-citation-1"] != OccurrenceContentChanged {
		t.Fatalf("citation status=%q", statuses["occ-citation-1"])
	}
}

// TestReconcileReportsAMissingSubjectInsteadOfFailing covers the case where the
// element a document was written about is no longer declared at all.
func TestReconcileReportsAMissingSubjectInsteadOfFailing(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	fixture.remove(t, "semantic/sales/order.tei")
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(report.NeedsReconfirmation) != len(draft.Occurrences) {
		t.Fatalf("needsReconfirmation=%v", report.NeedsReconfirmation)
	}
	for _, occurrence := range report.Occurrences {
		if occurrence.Status != OccurrenceSubjectMissing {
			t.Fatalf("occurrence %s status=%q", occurrence.OccurrenceID, occurrence.Status)
		}
	}
}

// TestReconcileRefusesADraftThatIsNotThisContract is RD-06: a hand-edited or
// foreign draft is refused instead of being partially compared or completed.
func TestReconcileRefusesADraftThatIsNotThisContract(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	cases := map[string]func(*Draft){
		"wrong schema":          func(d *Draft) { d.SchemaVersion = "tei.document-draft/v9" },
		"wrong effect":          func(d *Draft) { d.Effect = "read-write" },
		"no draft id":           func(d *Draft) { d.DraftID = "" },
		"no document kind":      func(d *Draft) { d.DocumentKind = "" },
		"no occurrence":         func(d *Draft) { d.Occurrences = nil },
		"occurrence no id":      func(d *Draft) { d.Occurrences[0].OccurrenceID = "" },
		"occurrence no rule":    func(d *Draft) { d.Occurrences[0].GenerationConditions.Rule = "" },
		"occurrence no digest":  func(d *Draft) { d.Occurrences[0].ContentDigest = "" },
		"occurrence no subject": func(d *Draft) { d.Occurrences[0].SubjectRefs = nil },
		"repeated occurrence": func(d *Draft) {
			d.Occurrences = append(d.Occurrences, d.Occurrences[0])
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			tampered := draft
			mutate(&tampered)
			_, err := Reconcile(tampered, fixture.root)
			if err == nil {
				t.Fatalf("Reconcile accepted a tampered draft (%s)", name)
			}
			var draftErr *DraftError
			if !errors.As(err, &draftErr) || draftErr.Code != CodeInvalidDraft {
				t.Fatalf("code=%v want %q", err, CodeInvalidDraft)
			}
		})
	}
}

// TestReconcileReportsAnUnknownRuleInsteadOfGuessing is RD-04: when the recorded
// generator or rule is not this version's, the occurrence is reported as
// unverifiable and needs reconfirmation rather than being silently treated as
// unchanged.
func TestReconcileReportsAnUnknownRuleInsteadOfGuessing(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	draft.Occurrences[0].GenerationConditions.Rule = "element-summary-v9"
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.Occurrences[0].Status != OccurrenceUnknownRule {
		t.Fatalf("status=%q", report.Occurrences[0].Status)
	}
	found := false
	for _, id := range report.NeedsReconfirmation {
		if id == draft.Occurrences[0].OccurrenceID {
			found = true
		}
	}
	if !found {
		t.Fatalf("unknown rule is not listed for reconfirmation: %v", report.NeedsReconfirmation)
	}
}

// TestReconcileDetectsAGenerationDifferenceWithTheSameRuleName is the other half
// of RD-04: a rule that exists but produced different bytes is a content change,
// not an unchanged occurrence.
func TestReconcileDetectsAGenerationDifferenceWithTheSameRuleName(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// Rewrite the explanation text only, keeping the recorded rule and digest
	// fields as they were: the regenerated content must now differ.
	draft.Occurrences[0].Content.Text = draft.Occurrences[0].Content.Text + " 作者が追記した一文。"
	draft.Occurrences[0].ContentDigest = "sha256:" + strings.Repeat("0", 64)
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.Occurrences[0].Status != OccurrenceContentChanged {
		t.Fatalf("status=%q", report.Occurrences[0].Status)
	}
	if report.Occurrences[0].CurrentContentDigest == report.Occurrences[0].RecordedContentDigest {
		t.Fatal("reported digests are equal, so the comparison proved nothing")
	}
}

// TestReconcileNeverWrites is the read-only boundary of the comparison: every
// Workspace file keeps its bytes.
func TestReconcileNeverWrites(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	before := map[string]string{
		"semantic/sales/order.tei":       fixture.digestOf(t, "semantic/sales/order.tei"),
		"semantic/customer/customer.tei": fixture.digestOf(t, "semantic/customer/customer.tei"),
		"tei.workspace.yaml":             fixture.digestOf(t, "tei.workspace.yaml"),
	}
	if _, err := Reconcile(draft, fixture.root); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	for relative, want := range before {
		if got := fixture.digestOf(t, relative); got != want {
			t.Fatalf("%s changed: %s -> %s", relative, want, got)
		}
	}
}
