package documentdraft

import (
	"errors"
	"testing"
)

// frozenFixture builds one synthetic Workspace with a data element, a relation
// and a capability, so an element draft can be frozen and later compared.
func frozenFixture(t *testing.T) fixture {
	t.Helper()
	base := newFixture(t)
	writeFile(t, base.root, "semantic/sales/submit.tei", []byte(`namespace sales

capability sales.order.submit {
    input order sales.order
    output sales.order
}
`))
	return base
}

// TestFrozenDeliveryPinsTheModelRevision is the first-pass proof: a frozen draft
// records the revision it was read at and says it will not be refreshed.
func TestFrozenDeliveryPinsTheModelRevision(t *testing.T) {
	fixture := frozenFixture(t)
	draft, err := Build(Request{
		Root: fixture.root, Subject: orderSubject(), Frozen: true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !draft.Frozen || draft.FrozenRevision == "" {
		t.Fatalf("pin=%+v", draft)
	}
	if draft.FrozenRevision != draft.Model.ModelRevision {
		t.Fatalf("pin %q does not match the model revision %q",
			draft.FrozenRevision, draft.Model.ModelRevision)
	}
	found := false
	for _, limit := range draft.AuthorityLimits {
		if limit == "frozen-delivery-is-not-refreshed-automatically" {
			found = true
		}
	}
	if !found {
		t.Fatalf("authorityLimits=%v", draft.AuthorityLimits)
	}
}

// TestFrozenDeliveryReportsExpiryAfterAModelChange pins that a changed model
// makes the delivery expire: the report says stale with a frozenRevision diff and
// never offers a refreshed projection.
func TestFrozenDeliveryReportsExpiryAfterAModelChange(t *testing.T) {
	fixture := frozenFixture(t)
	draft, err := Build(Request{
		Root: fixture.root, Subject: orderSubject(), Frozen: true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	before := fixture.digestOf(t, "semantic/sales/order.tei")
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
	if report.ModelReference.Status != ModelReferenceStale {
		t.Fatalf("model reference=%+v", report.ModelReference)
	}
	foundFrozenDiff := false
	for _, diff := range report.ModelReference.Diffs {
		if diff == "frozenRevision" {
			foundFrozenDiff = true
		}
	}
	if !foundFrozenDiff {
		t.Fatalf("diffs=%v do not name the frozen pin", report.ModelReference.Diffs)
	}
	foundLimit := false
	for _, limit := range report.AuthorityLimits {
		if limit == "frozen-delivery-is-not-refreshed-automatically" {
			foundLimit = true
		}
	}
	if !foundLimit {
		t.Fatalf("authorityLimits=%v", report.AuthorityLimits)
	}
	// The comparison wrote nothing: the source and the stored draft are intact.
	if fixture.digestOf(t, "semantic/sales/order.tei") == before {
		t.Fatal("fixture did not change; the test proves nothing")
	}
	if draft.FrozenRevision != draft.Model.ModelRevision {
		t.Fatal("the stored pin was rewritten in place")
	}
}

// TestFrozenDeliveryIsCurrentWhenTheModelDidNotMove pins the other direction: an
// unchanged model leaves the delivery valid.
func TestFrozenDeliveryIsCurrentWhenTheModelDidNotMove(t *testing.T) {
	fixture := frozenFixture(t)
	draft, err := Build(Request{
		Root: fixture.root, Subject: orderSubject(), Frozen: true,
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
}

// TestFrozenDeliveryIsNotADocumentKind keeps the pin and the kind apart: asking
// for `frozen-delivery` as a kind is refused rather than silently treated as a
// pin on whichever kind the implementation picked.
func TestFrozenDeliveryIsNotADocumentKind(t *testing.T) {
	fixture := frozenFixture(t)
	_, err := Build(Request{
		Root: fixture.root, Subject: orderSubject(),
		DocumentKind: DocumentKindFrozenDelivery,
	})
	if err == nil {
		t.Fatal("Build accepted frozen-delivery as a document kind")
	}
	var draftErr *DraftError
	if !errors.As(err, &draftErr) || draftErr.Code != CodeUnsupportedDocKind {
		t.Fatalf("code=%v want %q", err, CodeUnsupportedDocKind)
	}
}

// TestFrozenDeliveryAppliesToACRUDTable pins that the pin is orthogonal to the
// kind: a frozen CRUD delivery reports expiry the same way an element draft does.
func TestFrozenDeliveryAppliesToACRUDTable(t *testing.T) {
	fixture := crudFixture(t)
	draft, err := Build(Request{
		Root: fixture.root, DocumentKind: DocumentKindCRUDTable,
		Application: "order_entry", Frozen: true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !draft.Frozen || draft.FrozenRevision != draft.Model.ModelRevision {
		t.Fatalf("pin=%+v", draft)
	}
	report, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if report.ModelReference.Status != ModelReferenceCurrent {
		t.Fatalf("model reference=%+v", report.ModelReference)
	}
	fixture.rewrite(t, "databases/sales-core/persistence/order.tei", `persist sales_core.order {
    database sales_core
    semantic sales.order
    table public.orders_v2
    column sales.order.order_id -> order_id storage varchar required
    column sales.order.customer_id -> customer_id storage varchar required

    primary-key sales.order.order_id
}
`)
	expired, err := Reconcile(draft, fixture.root)
	if err != nil {
		t.Fatalf("Reconcile after change: %v", err)
	}
	if expired.ModelReference.Status != ModelReferenceStale {
		t.Fatalf("model reference=%+v", expired.ModelReference)
	}
	found := false
	for _, diff := range expired.ModelReference.Diffs {
		if diff == "frozenRevision" {
			found = true
		}
	}
	if !found {
		t.Fatalf("diffs=%v do not name the frozen pin", expired.ModelReference.Diffs)
	}
}

// TestUnfrozenDraftDoesNotPin pins the default: a draft without -frozen carries no
// pin, so a reader cannot mistake a tracking draft for a delivery artifact.
func TestUnfrozenDraftDoesNotPin(t *testing.T) {
	fixture := frozenFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if draft.Frozen || draft.FrozenRevision != "" {
		t.Fatalf("pin=%+v", draft)
	}
}

// TestFrozenDraftWithoutAPinIsRefused pins that the pin cannot be faked by hand:
// a draft that claims to be frozen without recording the revision is refused
// instead of compared, because its expiry would be undecidable.
func TestFrozenDraftWithoutAPinIsRefused(t *testing.T) {
	fixture := frozenFixture(t)
	draft, err := Build(Request{
		Root: fixture.root, Subject: orderSubject(), Frozen: true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	tampered := draft
	tampered.FrozenRevision = ""
	_, err = Reconcile(tampered, fixture.root)
	if err == nil {
		t.Fatal("Reconcile accepted a frozen draft without a pin")
	}
	var draftErr *DraftError
	if !errors.As(err, &draftErr) || draftErr.Code != CodeInvalidDraft {
		t.Fatalf("code=%v want %q", err, CodeInvalidDraft)
	}
}
