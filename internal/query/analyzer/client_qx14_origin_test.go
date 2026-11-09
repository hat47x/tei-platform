package analyzer

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"tei/internal/core"
)

// This uses the existing Go subprocess helper rather than SQLGlot. It
// demonstrates a missing input-origin proof; it does not assert that the
// current identity/protocol checks are erroneous or that a real worker lies.
func TestQX14SameIdentityResponseDoesNotBindSQLSchemaOrDialect(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEI_SQL_ANALYZER_TEST_HELPER", "1")
	original := Request{
		ProtocolVersion: 1, Operation: "match", Query: core.MustCanonicalID("app.qx14.origin"),
		Dialect: "postgres",
		Source:  "SELECT id FROM public.orders WHERE active = 1",
		Schema:  &SchemaContext{Tables: []SchemaTable{{Name: "public.orders", Columns: []string{"id", "active"}}}},
	}
	run := func(req Request) Result {
		t.Helper()
		result, err := RunPythonWorker(context.Background(), exe, "-test.run=^TestSQLAnalyzerWorkerHelper$", req)
		if err != nil {
			t.Fatalf("identity/protocol-matched response unexpectedly rejected: %v", err)
		}
		if result.Status != "ok" {
			t.Fatalf("unexpected helper status: %+v", result)
		}
		return result
	}
	baseline := run(original)
	cases := []struct {
		name   string
		modify func(*Request)
	}{
		{"different-source-same-ID", func(r *Request) { r.Source = "SELECT id FROM public.orders WHERE active = 0" }},
		{"different-schema-same-ID", func(r *Request) {
			r.Schema = &SchemaContext{Tables: []SchemaTable{{Name: "public.orders", Columns: []string{"id"}}}}
		}},
		{"different-dialect-same-ID", func(r *Request) { r.Dialect = "sqlite" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changed := original
			tc.modify(&changed)
			originalBytes, err := json.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			changedBytes, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			oldDigest, newDigest := sha256.Sum256(originalBytes), sha256.Sum256(changedBytes)
			if oldDigest == newDigest {
				t.Fatal("distinct analysis requests must have distinct test digests")
			}
			result := run(changed)
			if !reflect.DeepEqual(result, baseline) {
				t.Fatalf("helper should return the same unbound response for these requests: %+v versus %+v", result, baseline)
			}
			// A request-local digest can reject rebinding an OLD invocation
			// to new inputs; an echo-only response does not attest the worker
			// analyzed the NEW source/schema/dialect during the new call.
			oldInvocationMatchesNewRequest := oldDigest == newDigest
			if oldInvocationMatchesNewRequest {
				t.Fatal("stale invocation incorrectly rebound")
			}
			if result.Query != changed.Query || result.ProtocolVersion != changed.ProtocolVersion {
				t.Fatal("existing identity/protocol checks were not preserved")
			}
		})
	}
}

// An exact-request digest can detect mismatched request bytes, but it cannot
// witness changes between two equal byte sequences (ABA). This is
// a fixture-only Consumer qualification; no production epoch API is introduced.
func TestQX14RequestDigestDoesNotProveNoABA(t *testing.T) {
	fingerprint := func(r Request) [32]byte {
		t.Helper()
		payload, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		return sha256.Sum256(payload)
	}
	original := Request{
		ProtocolVersion: 1, Operation: "match", Query: core.MustCanonicalID("app.qx14.origin"),
		Dialect: "postgres", Source: "SELECT id FROM public.orders WHERE active = 1",
		Schema: &SchemaContext{Tables: []SchemaTable{{Name: "public.orders", Columns: []string{"id", "active"}}}},
	}
	changed := original
	changed.Source = "SELECT id FROM public.orders WHERE active = 0"
	reverted := original
	if fingerprint(original) == fingerprint(changed) {
		t.Fatal("a changed SQL request cannot be treated as the old request")
	}
	if fingerprint(original) != fingerprint(reverted) {
		t.Fatal("returning to the same bytes must have the same byte fingerprint")
	}
	// Revision is supplied only as a synthetic fixture fact; neither the
	// Analyzer Result nor a self-echoed digest attests to a trusted revision.
	type localObservation struct {
		requestHash      [32]byte
		observedRevision string
	}
	oldTicket := localObservation{requestHash: fingerprint(original), observedRevision: "r1"}
	mayReuse := func(ticket localObservation, current Request, currentRevision string) bool {
		return ticket.observedRevision != "" && currentRevision != "" &&
			ticket.requestHash == fingerprint(current) && ticket.observedRevision == currentRevision
	}
	if !mayReuse(oldTicket, original, "r1") {
		t.Fatal("identical synthetic request and revision should match")
	}
	if mayReuse(oldTicket, changed, "r2") {
		t.Fatal("changed source must invalidate an old request ticket")
	}
	if mayReuse(oldTicket, reverted, "r3") {
		t.Fatal("same bytes after ABA are not evidence of an unchanged revision")
	}
	if mayReuse(oldTicket, reverted, "") {
		t.Fatal("missing trusted current revision must not qualify reuse")
	}
}
