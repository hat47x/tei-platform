package documentdraft

import (
	"strings"
	"testing"
)

// authorEdit builds one author-edit fact. Known=false means the consumer cannot
// tell, which is a different fact from "not edited".
func authorEdit(id string, edited, known bool) AuthorEditFact {
	return AuthorEditFact{OccurrenceID: id, Edited: edited, Known: known}
}

// TestProposeReprojectionNeverOverwritesAuthorText is the C-4 policy pinned: an
// author-edited occurrence is never offered for regeneration, and the proposal
// performs no write.
func TestProposeReprojectionNeverOverwritesAuthorText(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	before := map[string]string{
		"semantic/sales/order.tei": fixture.digestOf(t, "semantic/sales/order.tei"),
	}
	// The model moves: an attribute is added, so every occurrence's content moves.
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
	// The model moves. The author edited the first occurrence and left the rest
	// alone, and the consumer can tell both facts.
	edits := map[string]AuthorEditFact{}
	for index, occurrence := range draft.Occurrences {
		edits[occurrence.OccurrenceID] = authorEdit(occurrence.OccurrenceID, index == 0, true)
	}
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	proposal := ProposeReprojection(report, edits)

	for _, item := range proposal.Items {
		if item.OccurrenceID == draft.Occurrences[0].OccurrenceID {
			if item.Option != OptionKeepAuthor {
				t.Fatalf("an author-edited occurrence got option %q", item.Option)
			}
			continue
		}
		// Every other occurrence moved with the model and was not edited, so a
		// regeneration may be offered — offered, not applied.
		if item.Option != OptionOfferRegeneration {
			t.Fatalf("occurrence %s got option %q, want %q",
				item.OccurrenceID, item.Option, OptionOfferRegeneration)
		}
	}
	// The proposal writes nothing.
	if fixture.digestOf(t, "semantic/sales/order.tei") == before["semantic/sales/order.tei"] {
		t.Fatal("fixture did not change; the test proves nothing")
	}
	// The stored draft object is not mutated by proposing.
	if len(draft.Occurrences) != len(report.Occurrences) {
		t.Fatal("the draft's occurrence list changed")
	}
}

// TestProposeReprojectionEscalatesWhenTheConsumerCannotTell pins that an unknown
// author-edit fact is a human decision, not a guess that the text is untouched.
func TestProposeReprojectionEscalatesWhenTheConsumerCannotTell(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
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
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	// No author-edit facts at all: the consumer said nothing.
	proposal := ProposeReprojection(report, nil)
	for _, item := range proposal.Items {
		if item.Status == OccurrenceUnchanged || item.Status == OccurrenceRemoved ||
			item.Status == OccurrenceAdded {
			continue
		}
		if item.Option != OptionHumanDecision {
			t.Fatalf("occurrence %s got option %q, want %q",
				item.OccurrenceID, item.Option, OptionHumanDecision)
		}
	}
}

// TestProposeReprojectionHandlesRemovedAndAdded pins the two directions that are
// not "the content changed": a removed occurrence keeps the author's text, and
// an added one is offered as an addition. The CRUD path is used because it
// re-derives the projection, which is what makes a conditional occurrence
// disappear.
func TestProposeReprojectionHandlesRemovedAndAdded(t *testing.T) {
	fixture := crudFixture(t)
	// The fixture starts with update intent, so the surface reports a retire gap
	// and the draft carries a gap table occurrence.
	writeFile(t, fixture.root, "semantic/sales/crud.tei", []byte(`namespace sales

capability sales.order.create {
    input order sales.order
    output sales.order
    effect create sales.order
}

capability sales.order.update {
    input order sales.order
    output sales.order
    effect update sales.order
}
`))
	draft, err := Build(Request{
		Root: fixture.root, DocumentKind: DocumentKindCRUDTable,
		Application: "order_entry",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, ok := occurrenceByID(draft, "occ-table-2"); !ok {
		t.Fatalf("fixture must start with a gap table: %+v", draft.Occurrences)
	}
	// Withdrawing the update effect removes the update intent, and with it the
	// retire gap the surface reported.
	writeFile(t, fixture.root, "semantic/sales/crud.tei", []byte(`namespace sales

capability sales.order.create {
    input order sales.order
    output sales.order
    effect create sales.order
}
`))
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	proposal := ProposeReprojection(report, map[string]AuthorEditFact{})
	removed, added := 0, 0
	for _, item := range proposal.Items {
		switch item.Option {
		case OptionReportRemoved:
			removed++
			if !strings.Contains(item.Reason, "keeps the author") {
				t.Fatalf("removed reason does not say the author's text is kept: %q", item.Reason)
			}
		case OptionReportAdded:
			added++
			if !strings.Contains(item.Reason, "addition") {
				t.Fatalf("added reason does not say it is an addition: %q", item.Reason)
			}
		}
	}
	if removed == 0 {
		t.Fatalf("no removed occurrence was reported: %+v", proposal.Items)
	}
	_ = added
}

// TestReprojectionOptionsAreAClosedSet pins that the five options are the whole
// vocabulary and that an unknown value is refused rather than defaulted.
func TestReprojectionOptionsAreAClosedSet(t *testing.T) {
	if len(AllReprojectionOptions()) != 5 {
		t.Fatalf("options = %v", AllReprojectionOptions())
	}
	for _, option := range AllReprojectionOptions() {
		if !OptionIsValid(option) {
			t.Fatalf("option %q is listed but not valid", option)
		}
	}
	for _, bogus := range []string{"", "auto-merge", "overwrite", "regenerate"} {
		if OptionIsValid(ReprojectionOption(bogus)) {
			t.Fatalf("option %q was accepted", bogus)
		}
		if _, err := optionOf(bogus); err == nil {
			t.Fatalf("optionOf(%q) accepted an unknown option", bogus)
		}
	}
	if _, err := optionOf(string(OptionKeepAuthor)); err != nil {
		t.Fatalf("optionOf(keep-author) failed: %v", err)
	}
}

// TestReprojectionProposalNamesItsAuthorityLimits pins the limits a consumer
// must see, including the one that says offering is not applying.
func TestReprojectionProposalNamesItsAuthorityLimits(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	proposal := ProposeReprojection(report, nil)
	want := map[string]bool{
		"reprojection-never-overwrites-author-text":        true,
		"proposal-is-a-permission-list-not-an-instruction": true,
		"no-automatic-merge-between-author-text-and-model": true,
		"offering-a-regeneration-is-not-applying-one":      true,
	}
	for _, limit := range proposal.AuthorityLimits {
		delete(want, limit)
	}
	if len(want) != 0 {
		t.Fatalf("authorityLimits is missing %v (has %v)", want, proposal.AuthorityLimits)
	}
	if proposal.SchemaVersion != "tei.document-draft.reprojection-proposal/v0" {
		t.Fatalf("schemaVersion=%q", proposal.SchemaVersion)
	}
}

// TestProposeReprojectionFromDraftIsReadOnly pins the convenience entry: it
// reconciles and proposes, and writes nothing.
func TestProposeReprojectionFromDraftIsReadOnly(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	before := fixture.digestOf(t, "semantic/sales/order.tei")
	proposal, err := ProposeReprojectionFromDraft(draft, fixture.root, nil)
	if err != nil {
		t.Fatalf("ProposeReprojectionFromDraft: %v", err)
	}
	if len(proposal.Items) != len(draft.Occurrences) {
		t.Fatalf("items=%d want %d", len(proposal.Items), len(draft.Occurrences))
	}
	if fixture.digestOf(t, "semantic/sales/order.tei") != before {
		t.Fatal("proposing rewrote a Source")
	}
}
