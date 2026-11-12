package documentdraft

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

// specFixture is one synthetic Workspace with two data elements, one relation
// between them and a capability, so a specification can bundle several chapters.
func specFixture(t *testing.T) fixture {
	t.Helper()
	base := newFixture(t)
	writeFile(t, base.root, "semantic/sales/order-line.tei", []byte(`namespace sales

data OrderLine {
    attribute orderLineId type std.text
    attribute lineNumber type std.text
}
`))
	return base
}

// TestSpecificationBundlesTheRequestedChapters is the first-pass proof: an
// ordered section list becomes one chapter per element, each with its own
// occurrences.
func TestSpecificationBundlesTheRequestedChapters(t *testing.T) {
	fixture := specFixture(t)
	draft, err := Build(Request{
		Root:         fixture.root,
		DocumentKind: DocumentKindSpecification,
		Sections: []SectionSpec{
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}, RelationID: "sales.order.customer"},
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order_line"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if draft.DocumentKind != DocumentKindSpecification {
		t.Fatalf("documentKind=%q", draft.DocumentKind)
	}
	if len(draft.Subjects) < 3 {
		t.Fatalf("subjects=%+v", draft.Subjects)
	}
	chapters := 0
	for _, occurrence := range draft.Occurrences {
		if occurrence.Kind == OccurrenceExplanation {
			chapters++
		}
	}
	if chapters != 2 {
		t.Fatalf("chapter count=%d want 2: %+v", chapters, draft.Occurrences)
	}
	// The relation inside the first chapter becomes a relation occurrence and a
	// relation entry, and its target is a subject too.
	if len(draft.Relations) != 1 || draft.Relations[0].Relation.ID != "sales.order.customer" {
		t.Fatalf("relations=%+v", draft.Relations)
	}
	foundRelation := false
	for _, occurrence := range draft.Occurrences {
		if occurrence.Kind == OccurrenceRelation &&
			occurrence.Content.Relation != nil &&
			occurrence.Content.Relation.Relation.ID == "sales.order.customer" {
			foundRelation = true
		}
	}
	if !foundRelation {
		t.Fatal("the chapter relation has no relation occurrence")
	}
	if draft.Scope == nil || len(draft.Scope.SkippedSubjects) != 0 {
		t.Fatalf("scope=%+v", draft.Scope)
	}
	for _, occurrence := range draft.Occurrences {
		if !strings.HasPrefix(occurrence.ContentDigest, "sha256:") {
			t.Fatalf("occurrence %s has no digest", occurrence.OccurrenceID)
		}
	}
}

// TestSpecificationIsDeterministicForTheSameInput pins that the same section list
// and the same read produce the same document, chapter order included.
func TestSpecificationIsDeterministicForTheSameInput(t *testing.T) {
	fixture := specFixture(t)
	sections := []SectionSpec{
		{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}},
		{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "customer.customer"}},
	}
	first, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindSpecification, Sections: sections})
	if err != nil {
		t.Fatalf("first Build: %v", err)
	}
	second, err := Build(Request{Root: fixture.root, DocumentKind: DocumentKindSpecification, Sections: sections})
	if err != nil {
		t.Fatalf("second Build: %v", err)
	}
	if first.DraftID != second.DraftID {
		t.Fatalf("draftId changed: %q vs %q", first.DraftID, second.DraftID)
	}
	if len(first.Occurrences) != len(second.Occurrences) {
		t.Fatalf("occurrence count changed: %d vs %d", len(first.Occurrences), len(second.Occurrences))
	}
	for i := range first.Occurrences {
		if first.Occurrences[i].OccurrenceID != second.Occurrences[i].OccurrenceID {
			t.Fatalf("occurrence %d id changed: %q vs %q", i,
				first.Occurrences[i].OccurrenceID, second.Occurrences[i].OccurrenceID)
		}
	}
}

// TestSpecificationChapterCountsTheChaptersItHolds pins that the document never
// claims a chapter count the model does not support: a skipped section is not a
// chapter, so "全 N 章" counts the chapters that exist. A reader who trusts that
// number must not be told three when the document holds two.
func TestSpecificationChapterCountsTheChaptersItHolds(t *testing.T) {
	fixture := specFixture(t)
	draft, err := Build(Request{
		Root:         fixture.root,
		DocumentKind: DocumentKindSpecification,
		Sections: []SectionSpec{
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}},
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.nothing"}},
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "customer.customer"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, occurrence := range draft.Occurrences {
		if occurrence.Kind != OccurrenceExplanation {
			continue
		}
		first := strings.SplitN(occurrence.Content.Text, "\n", 2)[0]
		if !strings.Contains(first, "全 2 章") {
			t.Fatalf("chapter %s says %q; three sections were requested and one is undeclared",
				occurrence.OccurrenceID, first)
		}
		if strings.Contains(occurrence.Content.Text, "全 3 章") {
			t.Fatalf("chapter %s counts a section that produced no chapter", occurrence.OccurrenceID)
		}
	}
	// The stored draft and a rebuild agree, so a reconciliation of an unchanged
	// model reports nothing to reconfirm.
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(report.NeedsReconfirmation) != 0 {
		t.Fatalf("an unchanged model produced reconfirmation items: %v", report.NeedsReconfirmation)
	}
}

// TestSpecificationSkipsAnUndeclaredChapterInsteadOfWritingOne is S4-b: a subject
// the model does not declare is recorded in SkippedSubjects and produces no
// chapter, so a partial request never looks complete.
func TestSpecificationSkipsAnUndeclaredChapterInsteadOfWritingOne(t *testing.T) {
	fixture := specFixture(t)
	draft, err := Build(Request{
		Root:         fixture.root,
		DocumentKind: DocumentKindSpecification,
		Sections: []SectionSpec{
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}},
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.nothing"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if draft.Scope == nil || len(draft.Scope.SkippedSubjects) != 1 {
		t.Fatalf("skipped=%+v", draft.Scope)
	}
	skipped := draft.Scope.SkippedSubjects[0]
	if skipped.Subject.ID != "sales.nothing" || skipped.Reason != "subject-not-declared" {
		t.Fatalf("skipped=%+v", skipped)
	}
	// Only the declared element produced a chapter.
	chapters := 0
	for _, occurrence := range draft.Occurrences {
		if occurrence.Kind == OccurrenceExplanation {
			chapters++
		}
	}
	if chapters != 1 {
		t.Fatalf("chapter count=%d want 1", chapters)
	}
	// The document never mentions the undeclared subject as a chapter.
	for _, occurrence := range draft.Occurrences {
		if occurrence.Content.Text != "" && strings.Contains(occurrence.Content.Text, "sales.nothing") {
			t.Fatalf("occurrence %s mentions the undeclared subject", occurrence.OccurrenceID)
		}
	}
}

// TestSpecificationSkipsARelationTheElementDoesNotDeclare pins that a relation
// inside a chapter is skipped with its reason, while the chapter itself stays.
func TestSpecificationSkipsARelationTheElementDoesNotDeclare(t *testing.T) {
	fixture := specFixture(t)
	draft, err := Build(Request{
		Root:         fixture.root,
		DocumentKind: DocumentKindSpecification,
		Sections: []SectionSpec{
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"},
				RelationID: "sales.order.phantom"},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if draft.Scope == nil || len(draft.Scope.SkippedSubjects) != 1 {
		t.Fatalf("skipped=%+v", draft.Scope)
	}
	if draft.Scope.SkippedSubjects[0].Reason != "relation-not-declared:sales.order.phantom" {
		t.Fatalf("reason=%q", draft.Scope.SkippedSubjects[0].Reason)
	}
	// The chapter stayed: a skipped section is not a skipped chapter.
	chapters := 0
	for _, occurrence := range draft.Occurrences {
		if occurrence.Kind == OccurrenceExplanation {
			chapters++
		}
	}
	if chapters != 1 {
		t.Fatalf("chapter count=%d want 1", chapters)
	}
	if len(draft.Relations) != 0 {
		t.Fatalf("relations=%+v", draft.Relations)
	}
}

// TestSpecificationRejectsMixedInputs keeps the three input shapes apart.
func TestSpecificationRejectsMixedInputs(t *testing.T) {
	fixture := specFixture(t)
	cases := map[string]Request{
		"no sections": {Root: fixture.root, DocumentKind: DocumentKindSpecification},
		"application": {Root: fixture.root, DocumentKind: DocumentKindSpecification,
			Application: "order_entry",
			Sections:    []SectionSpec{{Subject: orderSubject()}}},
		"element selector": {Root: fixture.root, DocumentKind: DocumentKindSpecification,
			Subject:  orderSubject(),
			Sections: []SectionSpec{{Subject: orderSubject()}}},
		"top-level relation": {Root: fixture.root, DocumentKind: DocumentKindSpecification,
			RelationID: "sales.order.customer",
			Sections:   []SectionSpec{{Subject: orderSubject()}}},
		"blank section subject": {Root: fixture.root, DocumentKind: DocumentKindSpecification,
			Sections: []SectionSpec{{Subject: sourceindex.Subject{Kind: "semantic-data", ID: " "}}}},
	}
	for name, request := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Build(request)
			if err == nil {
				t.Fatalf("Build accepted %+v", request)
			}
			var draftErr *DraftError
			if !errors.As(err, &draftErr) || draftErr.Code != CodeInvalidArgument {
				t.Fatalf("code=%v want %q", err, CodeInvalidArgument)
			}
		})
	}
}

// TestElementAndCRUDKindsRejectSections pins the other direction: a kind that does
// not use a section list refuses it rather than ignoring it.
func TestElementAndCRUDKindsRejectSections(t *testing.T) {
	sections := []SectionSpec{{Subject: orderSubject()}}
	t.Run("element kind", func(t *testing.T) {
		fixture := newFixture(t)
		_, err := Build(Request{Root: fixture.root, Subject: orderSubject(), Sections: sections})
		if err == nil {
			t.Fatal("Build accepted sections for an element draft")
		}
		var draftErr *DraftError
		if !errors.As(err, &draftErr) || draftErr.Code != CodeInvalidArgument {
			t.Fatalf("code=%v want %q", err, CodeInvalidArgument)
		}
	})
	t.Run("crud kind", func(t *testing.T) {
		fixture := crudFixture(t)
		_, err := Build(Request{
			Root: fixture.root, DocumentKind: DocumentKindCRUDTable,
			Application: "order_entry", Sections: sections,
		})
		if err == nil {
			t.Fatal("Build accepted sections for a CRUD draft")
		}
		var draftErr *DraftError
		if !errors.As(err, &draftErr) || draftErr.Code != CodeInvalidArgument {
			t.Fatalf("code=%v want %q", err, CodeInvalidArgument)
		}
	})
}

// TestSpecificationReconcileReportsTheChaptersThatMoved pins that a changed model
// moves only the chapters it actually affects.
func TestSpecificationReconcileReportsTheChaptersThatMoved(t *testing.T) {
	fixture := specFixture(t)
	draft, err := Build(Request{
		Root:         fixture.root,
		DocumentKind: DocumentKindSpecification,
		Sections: []SectionSpec{
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}},
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "customer.customer"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.ModelReference.Status != ModelReferenceCurrent {
		t.Fatalf("model reference=%+v", report.ModelReference)
	}
	if len(report.NeedsReconfirmation) != 0 {
		t.Fatalf("needsReconfirmation=%v", report.NeedsReconfirmation)
	}

	// A change to one element moves its chapter only.
	fixture.rewrite(t, "semantic/sales/order.tei", `namespace sales

data Order {
    attribute orderId type std.text
    attribute customerId type std.text
    attribute status type std.text
    attribute placedAt type std.datetime
    relation customer to customer.customer one
        key sales.order.customer_id -> customer.customer.customer_id
    relation lines to sales.order_line many
}
`)
	changed, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile after change: %v", err)
	}
	if changed.ModelReference.Status != ModelReferenceStale {
		t.Fatalf("model reference=%+v", changed.ModelReference)
	}
	if len(changed.NeedsReconfirmation) == 0 {
		t.Fatal("a changed model produced nothing to reconfirm")
	}
	// A change to one element moves its chapter's content. The other chapter's
	// content is identical, so it is reported as a generation-conditions change:
	// the model reference moved, not the words. That distinction is the whole
	// point of comparing occurrences rather than the whole document.
	statuses := map[string]string{}
	for _, occurrence := range changed.Occurrences {
		statuses[occurrence.OccurrenceID] = occurrence.Status
	}
	contentChanged := 0
	conditionsChanged := 0
	unchanged := 0
	for _, status := range statuses {
		switch status {
		case OccurrenceContentChanged:
			contentChanged++
		case OccurrenceConditionsChanged:
			conditionsChanged++
		case OccurrenceUnchanged:
			unchanged++
		}
	}
	if contentChanged == 0 {
		t.Fatalf("no occurrence reported a content change: %+v", statuses)
	}
	if conditionsChanged == 0 {
		t.Fatalf("no occurrence reported a generation-conditions change: %+v", statuses)
	}
	if unchanged != 0 {
		t.Fatalf("an unchanged occurrence survived a model change: %+v", statuses)
	}
}

// TestSpecificationReconcileReportsARemovedChapter covers the removal direction:
// an element the model stopped declaring must not leave a chapter that claims it
// still exists.
func TestSpecificationReconcileReportsARemovedChapter(t *testing.T) {
	fixture := specFixture(t)
	draft, err := Build(Request{
		Root:         fixture.root,
		DocumentKind: DocumentKindSpecification,
		Sections: []SectionSpec{
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}},
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "customer.customer"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	fixture.remove(t, "semantic/customer/customer.tei")
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	removed := 0
	for _, occurrence := range report.Occurrences {
		if occurrence.Status == OccurrenceRemoved {
			removed++
		}
	}
	if removed == 0 {
		t.Fatalf("a removed element produced no removed occurrence: %+v", report.Occurrences)
	}
}

// TestSpecificationNeverWrites is the read-only boundary of the specification
// path.
func TestSpecificationNeverWrites(t *testing.T) {
	fixture := specFixture(t)
	draft, err := Build(Request{
		Root:         fixture.root,
		DocumentKind: DocumentKindSpecification,
		Sections: []SectionSpec{
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}},
		},
	})
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

// TestSpecificationRejectsADraftWithNoSubjects keeps a hand-edited specification
// from being compared against nothing.
func TestSpecificationRejectsADraftWithNoSubjects(t *testing.T) {
	fixture := specFixture(t)
	draft, err := Build(Request{
		Root:         fixture.root,
		DocumentKind: DocumentKindSpecification,
		Sections: []SectionSpec{
			{Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	tampered := draft
	tampered.Subjects = nil
	_, err = Reconcile(tampered, fixture.root)
	if err == nil {
		t.Fatal("Reconcile accepted a specification with no subject")
	}
	var draftErr *DraftError
	if !errors.As(err, &draftErr) || draftErr.Code != CodeInvalidDraft {
		t.Fatalf("code=%v want %q", err, CodeInvalidDraft)
	}
}
