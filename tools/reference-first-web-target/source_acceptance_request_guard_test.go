//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
)

// Test-only mutable Reader. It is not a Host publication, human approval,
// trusted revocation mechanism, or authorization to execute a Command.
type requestSourceReader struct {
	record sourceacceptance.Record
	err    error
	calls  int
}

func (s *requestSourceReader) ReadCurrentAcceptedSource(_ context.Context, _ string) (sourceacceptance.Record, error) {
	s.calls++
	return s.record, s.err
}

func TestCurrentAcceptedSourceHTTPRequestGuard(t *testing.T) {
	member, err := sourceindex.NewArtifact("applications/order-entry/application.tei", []byte("application order_entry {}\n"))
	if err != nil {
		t.Fatal(err)
	}
	index := sourceindex.Index{Artifacts: []sourceindex.Artifact{member}}
	startup := sourceacceptance.Record{
		ApplicationID: "order_entry", DecisionRef: "test-only://decision/R1",
		AuthorityRevision: "host-test-revision-1",
		ModelRevision: sourceindex.ModelRevision(index),
		Artifacts:     append([]sourceindex.Artifact(nil), index.Artifacts...),
	}
	reader := &requestSourceReader{record: startup}
	dispatches := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dispatches++
		w.WriteHeader(http.StatusNoContent)
	})
	handler := guardCurrentAcceptedSource(next, reader, "order_entry", index, startup)
	request := func() *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/scenario/submit", nil))
		return response
	}
	ok := request()
	if ok.Code != http.StatusNoContent || reader.calls != 1 || dispatches != 1 {
		t.Fatalf("current R1 should enter handler exactly once: code=%d calls=%d dispatch=%d", ok.Code, reader.calls, dispatches)
	}

	// Startup witness remains fixed even if the Host later issues an equally
	// valued new decision. The process must be restarted for that authority.
	reader.record.DecisionRef = "test-only://decision/R1-reissued"
	reissued := request()
	if reissued.Code != http.StatusServiceUnavailable || dispatches != 1 {
		t.Fatal("a replacement Host decision reached the Command handler")
	}
	reader.record = startup
	reader.record.AuthorityRevision = "host-test-revision-2"
	if request().Code != http.StatusServiceUnavailable || dispatches != 1 {
		t.Fatal("a changed Authority revision reached the Command handler")
	}

	reader.record = startup
	reader.err = errors.New("Host current acceptance was revoked")
	revoked := request()
	if revoked.Code != http.StatusServiceUnavailable || dispatches != 1 ||
		revoked.Header().Get("Cache-Control") != "no-store" ||
		!strings.Contains(revoked.Body.String(), "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE") ||
		strings.Contains(revoked.Body.String(), "Host current acceptance was revoked") {
		t.Fatalf("revocation must fail closed without leaking Host details: %d %q", revoked.Code, revoked.Body.String())
	}

	reader.err = nil
	reader.record = startup
	reader.record.Artifacts = append([]sourceindex.Artifact(nil), startup.Artifacts...)
	reader.record.Artifacts[0].Fingerprint = sourceindex.Fingerprint([]byte("application order_entry {}\n\n"))
	if request().Code != http.StatusServiceUnavailable || dispatches != 1 {
		t.Fatal("same-meaning Source R2 reached a Command")
	}

	reader.record = startup
	recovered := request()
	if recovered.Code != http.StatusNoContent || reader.calls != 6 || dispatches != 2 {
		t.Fatalf("original accepted R1 must resume only after current Host recheck: code=%d calls=%d dispatch=%d", recovered.Code, reader.calls, dispatches)
	}
}

func TestCurrentAcceptedSourceHTTPRequestGuardMissingReaderAndCancellation(t *testing.T) {
	member, err := sourceindex.NewArtifact("one.tei", []byte("x\n"))
	if err != nil {
		t.Fatal(err)
	}
	index := sourceindex.Index{Artifacts: []sourceindex.Artifact{member}}
	startup := sourceacceptance.Record{
		ApplicationID: "order_entry", DecisionRef: "test-only",
		AuthorityRevision: "test-only", ModelRevision: sourceindex.ModelRevision(index),
		Artifacts: index.Artifacts,
	}
	calls := 0
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ })
	noReader := guardCurrentAcceptedSource(next, nil, "order_entry", index, startup)
	response := httptest.NewRecorder()
	noReader.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusServiceUnavailable || calls != 0 {
		t.Fatal("missing current Reader reached HTTP dispatch")
	}
	reader := &requestSourceReader{record: startup}
	handler := guardCurrentAcceptedSource(next, reader, "order_entry", index, startup)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/scenario/submit", nil).WithContext(ctx))
	if response.Code != http.StatusServiceUnavailable || calls != 0 || reader.calls != 0 {
		t.Fatal("cancelled request reached Host or Command dispatch")
	}
}
