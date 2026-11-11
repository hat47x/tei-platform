package documentdraft

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

// TestBuildProjectsOneElementAndOneRelation is the first-pass proof: one
// accepted element plus one declared relation becomes a draft that carries the
// model reference, both subjects, the relation resolution and the occurrences
// in a deterministic order.
func TestBuildProjectsOneElementAndOneRelation(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{
		Root:       fixture.root,
		Subject:    orderSubject(),
		RelationID: "sales.order.customer",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if draft.SchemaVersion != SchemaVersion || draft.Effect != EffectReadOnlyProjection {
		t.Fatalf("unexpected draft header: %+v", draft)
	}
	if draft.DocumentKind != DocumentKindElementExplanation {
		t.Fatalf("documentKind=%q", draft.DocumentKind)
	}
	if draft.DraftID == "" || !strings.HasPrefix(draft.DraftID, "sha256:") {
		t.Fatalf("draftId=%q", draft.DraftID)
	}
	if draft.Title != "sales.order — 要素説明" {
		t.Fatalf("title=%q", draft.Title)
	}
	if len(draft.Subjects) != 2 {
		t.Fatalf("subjects=%+v", draft.Subjects)
	}
	if draft.Subjects[1].Subject != (sourceindex.Subject{Kind: "semantic-data", ID: "customer.customer"}) {
		t.Fatalf("relation target subject=%+v", draft.Subjects[1])
	}
	if len(draft.Relations) != 1 {
		t.Fatalf("relations=%+v", draft.Relations)
	}
	relation := draft.Relations[0]
	if relation.Relation != (sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.customer"}) ||
		relation.From != orderSubject() ||
		relation.To != (sourceindex.Subject{Kind: "semantic-data", ID: "customer.customer"}) ||
		relation.Resolution != ResolutionResolved || relation.Cardinality != "one" {
		t.Fatalf("relation=%+v", relation)
	}
	if len(relation.Keys) != 1 ||
		relation.Keys[0].SourceAttribute != "sales.order.customer_id" ||
		relation.Keys[0].TargetAttribute != "customer.customer.customer_id" {
		t.Fatalf("relation keys=%+v", relation.Keys)
	}
	if len(draft.Subjects[0].Sources) == 0 {
		t.Fatal("element carries no source binding")
	}
	artifact := draft.Subjects[0].Sources[0]
	if artifact.ArtifactPath != "semantic/sales/order.tei" ||
		!strings.HasPrefix(artifact.Fingerprint, "sha256:") || len(artifact.Ranges) == 0 {
		t.Fatalf("source binding=%+v", artifact)
	}
	if artifact.Fingerprint != fixture.digestOf(t, "semantic/sales/order.tei") {
		t.Fatalf("source fingerprint does not match the artifact bytes: %+v", artifact)
	}
	wantKinds := []string{
		OccurrenceExplanation, OccurrenceRelation, OccurrenceTable, OccurrenceCitation,
	}
	gotKinds := kindsOf(draft)
	if strings.Join(gotKinds, ",") != strings.Join(wantKinds, ",") {
		t.Fatalf("occurrence kinds=%v want %v", gotKinds, wantKinds)
	}
	for _, occurrence := range draft.Occurrences {
		if occurrence.ContentDigest == "" || !strings.HasPrefix(occurrence.ContentDigest, "sha256:") {
			t.Fatalf("occurrence %s has no content digest", occurrence.OccurrenceID)
		}
		if occurrence.GenerationConditions.Generator != GeneratorName ||
			occurrence.GenerationConditions.GeneratorVersion != GeneratorVersion ||
			occurrence.GenerationConditions.Rule == "" {
			t.Fatalf("occurrence %s generation conditions=%+v",
				occurrence.OccurrenceID, occurrence.GenerationConditions)
		}
		if occurrence.GenerationConditions.InputDigests["modelRevision"] == "" {
			t.Fatalf("occurrence %s has no model revision", occurrence.OccurrenceID)
		}
	}
	table, ok := occurrenceByID(draft, "occ-table-1")
	if !ok || table.Content.Table == nil {
		t.Fatalf("attribute table occurrence missing: %+v", draft.Occurrences)
	}
	if len(table.Content.Table.Header) != 5 {
		t.Fatalf("table header=%v", table.Content.Table.Header)
	}
	// Three attributes plus two declared relations are listed; nothing is
	// invented for an undeclared flag.
	if len(table.Content.Table.Rows) != 5 {
		t.Fatalf("table rows=%+v", table.Content.Table.Rows)
	}
	citation, ok := occurrenceByID(draft, "occ-citation-1")
	if !ok || citation.Content.Citation == nil {
		t.Fatalf("citation occurrence missing: %+v", draft.Occurrences)
	}
	if citation.Content.Citation.Resolution != "recorded-not-resolved" {
		t.Fatalf("citation resolution=%q", citation.Content.Citation.Resolution)
	}
	if len(draft.AuthorityLimits) == 0 {
		t.Fatal("draft states no authority limit")
	}
}

// TestBuildIsDeterministicForTheSameRead pins that the same read produces the
// same draft identity and the same occurrence digests. A change in either is a
// change a consumer would have to notice.
func TestBuildIsDeterministicForTheSameRead(t *testing.T) {
	fixture := newFixture(t)
	first, err := Build(Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer"})
	if err != nil {
		t.Fatalf("first Build: %v", err)
	}
	second, err := Build(Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer"})
	if err != nil {
		t.Fatalf("second Build: %v", err)
	}
	if first.DraftID != second.DraftID {
		t.Fatalf("draftId changed between identical reads: %q vs %q", first.DraftID, second.DraftID)
	}
	if len(first.Occurrences) != len(second.Occurrences) {
		t.Fatalf("occurrence count changed: %d vs %d", len(first.Occurrences), len(second.Occurrences))
	}
	for i := range first.Occurrences {
		if first.Occurrences[i].ContentDigest != second.Occurrences[i].ContentDigest {
			t.Fatalf("occurrence %s digest changed", first.Occurrences[i].OccurrenceID)
		}
	}
}

// TestBuildWithoutRelationOmitsTheRelationOccurrence keeps the relation opt-in:
// a caller that did not ask for one must not receive a projected relation.
func TestBuildWithoutRelationOmitsTheRelationOccurrence(t *testing.T) {
	fixture := newFixture(t)
	draft, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(draft.Relations) != 0 {
		t.Fatalf("relations=%+v", draft.Relations)
	}
	if _, ok := occurrenceByID(draft, "occ-relation-1"); ok {
		t.Fatal("relation occurrence present without a requested relation")
	}
	if len(draft.Subjects) != 1 {
		t.Fatalf("subjects=%+v", draft.Subjects)
	}
}

// TestBuildRejectsUnknownAndUnsupportedInputs separates an element that does not
// exist from a kind this projection does not read and from a document kind that
// is not part of the first integration.
func TestBuildRejectsUnknownAndUnsupportedInputs(t *testing.T) {
	fixture := newFixture(t)
	cases := []struct {
		name    string
		request Request
		code    string
	}{
		{
			name:    "missing element",
			request: Request{Root: fixture.root, Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.nothing"}},
			code:    CodeSubjectNotFound,
		},
		{
			name:    "unsupported element kind",
			request: Request{Root: fixture.root, Subject: sourceindex.Subject{Kind: "application", ID: "order_entry"}},
			code:    CodeUnsupportedKind,
		},
		{
			name:    "unsupported document kind",
			request: Request{Root: fixture.root, Subject: orderSubject(), DocumentKind: "three-element-diagram"},
			code:    CodeUnsupportedDocKind,
		},
		{
			name:    "relation of another element",
			request: Request{Root: fixture.root, Subject: sourceindex.Subject{Kind: "semantic-data", ID: "customer.customer"}, RelationID: "sales.order.customer"},
			code:    CodeRelationNotFound,
		},
		{
			name:    "unknown relation",
			request: Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.phantom"},
			code:    CodeRelationNotFound,
		},
		{
			name:    "empty workspace",
			request: Request{Root: " ", Subject: orderSubject()},
			code:    CodeInvalidArgument,
		},
		{
			name:    "blank subject id",
			request: Request{Root: fixture.root, Subject: sourceindex.Subject{Kind: "semantic-data", ID: " "}},
			code:    CodeInvalidArgument,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Build(tc.request)
			if err == nil {
				t.Fatalf("Build accepted %+v", tc.request)
			}
			var draftErr *DraftError
			if !errors.As(err, &draftErr) {
				t.Fatalf("error %v is not a DraftError", err)
			}
			if draftErr.Code != tc.code {
				t.Fatalf("code=%q want %q (%v)", draftErr.Code, tc.code, draftErr.Message)
			}
		})
	}
}

// TestBuildKeepsAnUnresolvedRelationTargetUnresolved pins that a declared
// relation to an undeclared data still projects, with the target reported as
// unresolved instead of being dropped or guessed.
func TestBuildKeepsAnUnresolvedRelationTargetUnresolved(t *testing.T) {
	fixture := newFixture(t)
	fixture.rewrite(t, "semantic/sales/order.tei", `namespace sales

data Order {
    attribute orderId type sales.order_id
    relation carrier to logistics.carrier many
}
`)
	draft, err := Build(Request{
		Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.carrier",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(draft.Relations) != 1 || draft.Relations[0].Resolution != ResolutionUnresolved {
		t.Fatalf("relation=%+v", draft.Relations)
	}
	if len(draft.Subjects) != 2 || len(draft.Subjects[1].Sources) != 0 {
		t.Fatalf("unresolved target subject=%+v", draft.Subjects)
	}
}

// TestBuildReportsAnAmbiguousTargetSeparatelyFromAnUnresolvedOne keeps the two
// answers apart: declared twice and declared zero times are different facts.
func TestBuildReportsAnAmbiguousTargetSeparatelyFromAnUnresolvedOne(t *testing.T) {
	fixture := newFixture(t)
	fixture.rewrite(t, "semantic/customer/customer-copy.tei", `namespace customer

data Customer {
    attribute customerId type customer.customer_id
}
`)
	draft, err := Build(Request{
		Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(draft.Relations) != 1 || draft.Relations[0].Resolution != ResolutionAmbiguous {
		t.Fatalf("relation=%+v", draft.Relations)
	}
}

// TestBuildRejectsAnElementDeclaredTwice keeps an ambiguous element out of a
// draft instead of picking whichever declaration the loader listed first.
func TestBuildRejectsAnElementDeclaredTwice(t *testing.T) {
	fixture := newFixture(t)
	fixture.rewrite(t, "semantic/sales/order-copy.tei", `namespace sales

data Order {
    attribute orderId type sales.order_id
}
`)
	_, err := Build(Request{Root: fixture.root, Subject: orderSubject()})
	if err == nil {
		t.Fatal("Build accepted an ambiguous element")
	}
	var draftErr *DraftError
	if !errors.As(err, &draftErr) || draftErr.Code != CodeSubjectAmbiguous {
		t.Fatalf("code=%v (%v)", err, draftErr)
	}
}

// TestBuildProjectsATypeAndACapability covers the other two supported element
// kinds with the same occurrence discipline.
func TestBuildProjectsATypeAndACapability(t *testing.T) {
	fixture := newFixture(t)
	typeDraft, err := Build(Request{
		Root:    fixture.root,
		Subject: sourceindex.Subject{Kind: "semantic-type", ID: "sales.order_id"},
	})
	if err != nil {
		t.Fatalf("type Build: %v", err)
	}
	if _, ok := occurrenceByID(typeDraft, "occ-table-1"); !ok {
		t.Fatalf("type table missing: %+v", typeDraft.Occurrences)
	}
	capabilityDraft, err := Build(Request{
		Root:    fixture.root,
		Subject: sourceindex.Subject{Kind: "semantic-capability", ID: "sales.order.submit"},
	})
	if err != nil {
		t.Fatalf("capability Build: %v", err)
	}
	if _, ok := occurrenceByID(capabilityDraft, "occ-table-1"); !ok {
		t.Fatalf("capability table missing: %+v", capabilityDraft.Occurrences)
	}
}

// TestBuildDoesNotWriteAnything is the read-only boundary of the projection: the
// Workspace bytes are identical after a Build.
func TestBuildDoesNotWriteAnything(t *testing.T) {
	fixture := newFixture(t)
	before := fixture.digestOf(t, "semantic/sales/order.tei")
	if _, err := Build(Request{Root: fixture.root, Subject: orderSubject(), RelationID: "sales.order.customer"}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if after := fixture.digestOf(t, "semantic/sales/order.tei"); after != before {
		t.Fatalf("Build changed the source: %s -> %s", before, after)
	}
}
