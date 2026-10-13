package sourceacceptance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

type currentReader struct {
	record Record
	err    error
	calls  int
}

func (r *currentReader) ReadCurrentAcceptedSource(_ context.Context, _ string) (Record, error) {
	r.calls++
	return r.record, r.err
}

func testArtifacts() []sourceindex.Artifact {
	a, _ := sourceindex.NewArtifact("applications/order-entry/application.tei", []byte("application order_entry {}\n"))
	b, _ := sourceindex.NewArtifact("semantic/sales/order.tei", []byte("data sales.order {}\n"))
	return []sourceindex.Artifact{a, b}
}

func fixture() (*currentReader, sourceindex.Index) {
	artifacts := testArtifacts()
	index := sourceindex.Index{Artifacts: append([]sourceindex.Artifact(nil), artifacts...)}
	return &currentReader{record: Record{
		ApplicationID: "order_entry", DecisionRef: "host://accepted/123", AuthorityRevision: "host-42",
		ModelRevision: sourceindex.ModelRevision(index), Artifacts: artifacts,
	}}, index
}

func expectRefusal(t *testing.T, r *currentReader, i sourceindex.Index, code string) {
	t.Helper()
	_, err := VerifyCurrent(context.Background(), r, "order_entry", i)
	if err == nil || !strings.Contains(err.Error(), code) {
		t.Fatalf("expected %s, got %v", code, err)
	}
}

func TestAcceptedR1AndOrderIndependence(t *testing.T) {
	r, index := fixture()
	index.Artifacts[0], index.Artifacts[1] = index.Artifacts[1], index.Artifacts[0]
	record, err := VerifyCurrent(context.Background(), r, "order_entry", index)
	if err != nil || r.calls != 1 {
		t.Fatalf("R1 must pass once: %+v %v", record, err)
	}
	record.Artifacts[0].Path = "modified-by-caller"
	if r.record.Artifacts[0].Path == "modified-by-caller" {
		t.Fatal("returned Host witness aliases backing slice")
	}
}

func TestR0AndSemanticEquivalentR2AreRefused(t *testing.T) {
	r, index := fixture()
	index.Artifacts[0].Fingerprint = sourceindex.Fingerprint([]byte("application order_entry { changed }\n"))
	expectRefusal(t, r, index, "SOURCE_FINGERPRINT_MISMATCH")
	r, index = fixture()
	index.Artifacts[0].Fingerprint = sourceindex.Fingerprint([]byte("application order_entry {}\n\n"))
	expectRefusal(t, r, index, "SOURCE_FINGERPRINT_MISMATCH")
}

func TestMembershipAndPathRejections(t *testing.T) {
	r, index := fixture()
	index.Artifacts = index.Artifacts[:1]
	expectRefusal(t, r, index, "SOURCE_MEMBERSHIP_MISMATCH")
	r, index = fixture()
	extra, _ := sourceindex.NewArtifact("unexpected.tei", []byte("extra"))
	index.Artifacts = append(index.Artifacts, extra)
	expectRefusal(t, r, index, "SOURCE_MEMBERSHIP_MISMATCH")
	r, index = fixture()
	index.Artifacts[0].Path = "../outside.tei"
	expectRefusal(t, r, index, "SOURCE_INDEX_INVALID")
	r, index = fixture()
	index.Artifacts = append(index.Artifacts, index.Artifacts[0])
	expectRefusal(t, r, index, "SOURCE_INDEX_INVALID")
}

func TestFailClosedForMissingOrRevokedHost(t *testing.T) {
	_, index := fixture()
	if _, err := VerifyCurrent(context.Background(), nil, "order_entry", index); err == nil || !strings.Contains(err.Error(), "SOURCE_ACCEPTANCE_READER_REQUIRED") {
		t.Fatal(err)
	}
	r, _ := fixture()
	r.err = errors.New("revoked")
	expectRefusal(t, r, index, "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE")
	if r.calls != 1 {
		t.Fatal("Host must be consulted")
	}
	r, _ = fixture()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := VerifyCurrent(ctx, r, "order_entry", index)
	if err == nil || !strings.Contains(err.Error(), "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE") || r.calls != 0 {
		t.Fatalf("cancelled read %v, calls %d", err, r.calls)
	}
}

func TestRejectForgedOrIncompleteWitness(t *testing.T) {
	cases := []struct {
		name, code string
		mutate     func(*Record)
	}{
		{"wrong app", "SOURCE_ACCEPTANCE_APPLICATION_MISMATCH", func(r *Record) { r.ApplicationID = "repair_intake" }},
		{"missing decision", "SOURCE_ACCEPTANCE_WITNESS_INVALID", func(r *Record) { r.DecisionRef = "" }},
		{"missing authority", "SOURCE_ACCEPTANCE_WITNESS_INVALID", func(r *Record) { r.AuthorityRevision = "" }},
		{"fake accepted revision", "SOURCE_ACCEPTANCE_WITNESS_INVALID", func(r *Record) { r.ModelRevision = sourceindex.Fingerprint([]byte("forged")) }},
		{"noncanonical path", "SOURCE_ACCEPTANCE_WITNESS_INVALID", func(r *Record) { r.Artifacts[0].Path = "../outside" }},
		{"duplicate member", "SOURCE_ACCEPTANCE_WITNESS_INVALID", func(r *Record) { r.Artifacts = append(r.Artifacts, r.Artifacts[0]) }},
		{"malformed digest", "SOURCE_ACCEPTANCE_WITNESS_INVALID", func(r *Record) { r.Artifacts[0].Fingerprint = "NOT_A_DIGEST" }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) { r, index := fixture(); tt.mutate(&r.record); expectRefusal(t, r, index, tt.code) })
	}
}
