// Characterization of partial developer-context observations and local decision
// delivery. These tests deliberately preserve failures of inference, not claim
// discovery completeness, noninterference, or model quality.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoveryScanBudgetMayHideRelatedDocument(t *testing.T) {
	root := t.TempDir()
	target := "internal/crossview/qualification.go"
	writeCoverageFixture(t, root, target, "package crossview\nfunc Evaluate() {}\n")
	for i := 0; i < 1000; i++ {
		path := fmt.Sprintf("docs/a%04d.md", i)
		writeCoverageFixture(t, root, path, "# Unrelated\nno source path here\n")
	}
	// W0: 1,000 unrelated docs, no matches.
	without, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(without.Candidates) != 0 {
		t.Fatalf("W0 candidates should be empty: %+v", without.Candidates)
	}

	// W1 differs only by an explicit, lexically related document after the
	// first 1,000 Markdown entries in WalkDir's name ordering.
	related := "docs/z-related.md"
	writeCoverageFixture(t, root, related,
		"# Source reference\ninternal/crossview/qualification.go\n")
	after, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(without, after) {
		t.Fatalf("reports should be observationally identical despite a new later doc:\nW0=%+v\nW1=%+v", without, after)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(related)))
	if err != nil || !strings.Contains(string(raw), target) {
		t.Fatalf("fixture must contain an independent, exact path mention: %v", err)
	}
	// Report has no scanned/complete field. An empty candidate set cannot
	// prove no matching document exists, even when maxCandidates is not hit.
}

func TestDiscoveryUnreadableDocumentIsNotNoDocument(t *testing.T) {
	root := t.TempDir()
	target := "internal/crossview/qualification.go"
	writeCoverageFixture(t, root, target, "package crossview\n")
	// W2: no documents.
	without, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	// W3: an existing Markdown is too large for readInside, so Discover
	// silently skips it. This is a portable file-size refusal fixture,
	// not a simulation of OS permission denial.
	related := "docs/related.md"
	body := "internal/crossview/qualification.go\n" +
		strings.Repeat("x", int(maxFileBytes))
	writeCoverageFixture(t, root, related, body)
	stat, err := os.Stat(filepath.Join(root, filepath.FromSlash(related)))
	if err != nil || stat.Size() <= maxFileBytes {
		t.Fatalf("expected an existing oversized fixture: %v", err)
	}
	if _, readErr := readInside(root, related); readErr == nil {
		t.Fatal("oversized file was unexpectedly readable")
	}
	withFile, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(without, withFile) {
		t.Fatalf("existing unreadable document should not appear as a candidate:\nW2=%+v\nW3=%+v", without, withFile)
	}
	// No document and an unreadable document are distinguishable by an
	// independent file observation, but not by this report alone.
}

func TestRerankModelFailureDoesNotUndoLocalRequestDelivery(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	candidate := "internal/validation/validation_test.go"
	writeCoverageFixture(t, root, target, "package validation\n")
	writeCoverageFixture(t, root, candidate, "package validation_test\n")

	// A disallowed endpoint fails before HTTP delivery.
	notSent, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	Rerank(context.Background(), &notSent,
		"http://example.invalid/v1/systemone", "fixture-model")
	if notSent.Evaluation.State != "model-failed" {
		t.Fatalf("expected invalid endpoint failure: %+v", notSent.Evaluation)
	}

	// A permitted loopback endpoint can receive the entire request but
	// respond with an error, causing the same deterministic candidate order
	// to be retained. A read-only Report flag is not a network prohibition.
	received := 0
	var delivered []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/systemone" {
			t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
		}
		var err error
		delivered, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
		}
		http.Error(w, "fixture failure", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	sentButFailed, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	Rerank(context.Background(), &sentButFailed,
		server.URL+"/v1/systemone", "fixture-model")
	if received != 1 || len(delivered) == 0 {
		t.Fatalf("expected exactly one locally delivered request, got %d", received)
	}
	if !strings.Contains(string(delivered), target) ||
		!strings.Contains(string(delivered), candidate) {
		t.Fatalf("fixture paths absent in delivered request: %s", delivered)
	}
	if sentButFailed.Evaluation.State != "model-failed" ||
		!strings.Contains(sentButFailed.Evaluation.Diagnostic, "HTTP 503") {
		t.Fatalf("expected model-failed after delivered request: %+v", sentButFailed.Evaluation)
	}
	if !sentButFailed.ReadOnly || !notSent.ReadOnly ||
		!reflect.DeepEqual(sentButFailed.Candidates, notSent.Candidates) {
		t.Fatalf("model failure must preserve deterministic candidates, not undo delivery")
	}
	// Local delivery was observed. We do not assert an external egress, a
	// model inferred anything, or a real listener had side effects.
}
