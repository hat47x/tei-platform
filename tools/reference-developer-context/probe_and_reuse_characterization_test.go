// Limited Go characterization for T7 and freshness of model comparison trials.
// These tests observe existing Discover/Rerank code; they do not add a probing
// Tool, prove all repository docs were explored, or evaluate a real LLM.
package main

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestDiscoveryTwelveTestsHideDocumentEvenIfDocumentIsAdded(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	writeCoverageFixture(t, root, target, "package validation\nfunc Apply() {}\n")
	for i := 0; i < maxCandidates; i++ {
		path := fmt.Sprintf("internal/validation/a%02d_test.go", i)
		writeCoverageFixture(t, root, path, "package validation\n")
	}

	// W0: candidate budget fills before docs scanning.
	without, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(without.Candidates) != maxCandidates {
		t.Fatalf("expected full budget in W0, got %d", len(without.Candidates))
	}

	// W1: same target/tests and one additional exact-path Markdown. Reports
	// remain identical because Discover does not enter the docs walk.
	related := "docs/target-reference.md"
	writeCoverageFixture(t, root, related, "# Exact source reference\n"+target+"\n")
	withDoc, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(without, withDoc) {
		t.Fatalf("W0/W1 must yield the same partial report:\nW0=%+v\nW1=%+v", without, withDoc)
	}
	if _, found := candidateOf(withDoc, related); found {
		t.Fatal("document unexpectedly survived the already-full test budget")
	}

	// Separate, fixture-scoped read by the test (NOT by the Discover CLI).
	// It proves a lexically matching document exists, not its meaning.
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(related)))
	if err != nil || !strings.Contains(string(content), target) {
		t.Fatalf("independent fixture witness missing: %v", err)
	}
}

func TestRerankReusedReportCarriesPreviousModelOutputIntoNextRequest(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	direct := "internal/validation/validation_test.go"
	other := "internal/validation/another_test.go"
	writeCoverageFixture(t, root, target, "package validation\n")
	writeCoverageFixture(t, root, direct, "package validation_test\n")
	writeCoverageFixture(t, root, other, "package validation\n")

	base, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(base.Candidates) != 2 || base.Candidates[0].Path != direct ||
		base.Candidates[1].Path != other {
		t.Fatalf("unexpected deterministic candidates: %+v", base.Candidates)
	}

	var requests [][]byte
	var candidateOrders [][]Candidate
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("model request read: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		requests = append(requests, append([]byte(nil), body...))
		var got struct {
			State struct {
				Candidates []Candidate `json:"candidates"`
			} `json:"state"`
			Questions map[string]struct {
				Instructions string `json:"instructions"`
			} `json:"questions"`
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("decode model request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(got.State.Candidates) != 2 || len(got.Questions) != 2 {
			t.Errorf("unexpected model request shape: %+v", got)
		}
		candidateOrders = append(candidateOrders, got.State.Candidates)
		// Always prefer the position c1. This intentionally makes the first
		// model call change the candidate order in the original Report.
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"answers":{"c0":{"type":"choice","choice":"not_relevant","probabilities":{"relevant":0,"uncertain":0,"not_relevant":1}},"c1":{"type":"choice","choice":"relevant","probabilities":{"relevant":1,"uncertain":0,"not_relevant":0}}}}`)
	}))
	defer server.Close()

	reused := base
	Rerank(context.Background(), &reused, server.URL+"/v1/systemone", "fixture-only-model")
	if reused.Evaluation.State != "model-ranked" || reused.Candidates[0].Path != other {
		t.Fatalf("first model choice did not reorder candidates: %+v", reused)
	}
	// A Go struct copy is not a deep copy: its Candidates slice still shares
	// backing storage with base. The original Evaluation field still says
	// "baseline", although its candidate order and choice were mutated.
	if base.Evaluation.State != "baseline" || base.Candidates[0].Path != other ||
		base.Candidates[0].ModelChoice == "" {
		t.Fatalf("expected a shallow-copy baseline to be contaminated: %+v", base)
	}
	Rerank(context.Background(), &reused, server.URL+"/v1/systemone", "fixture-only-model")
	if reused.Evaluation.State != "model-ranked" {
		t.Fatalf("reused report rerank failed: %+v", reused.Evaluation)
	}

	// A fresh new Report with the same unmodified filesystem must send the
	// original byte-for-byte request, not the previous trial's model answer.
	fresh, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	Rerank(context.Background(), &fresh, server.URL+"/v1/systemone", "fixture-only-model")
	if fresh.Evaluation.State != "model-ranked" || len(requests) != 3 ||
		len(candidateOrders) != 3 {
		t.Fatalf("unexpected delivery count / rerank state: %d %d %+v", len(requests), len(candidateOrders), fresh.Evaluation)
	}
	if candidateOrders[0][0].Path != direct ||
		candidateOrders[1][0].Path != other ||
		candidateOrders[2][0].Path != direct {
		t.Fatalf("trial-local c0 binding not preserved: %+v", candidateOrders)
	}
	if candidateOrders[0][0].ModelChoice != "" ||
		candidateOrders[1][0].ModelChoice == "" ||
		candidateOrders[2][0].ModelChoice != "" {
		t.Fatalf("prior model judgment unexpectedly absent/present: %+v", candidateOrders)
	}
	if bytes.Equal(requests[0], requests[1]) ||
		!bytes.Equal(requests[0], requests[2]) {
		t.Fatal("reused report contaminated the next request, fresh report did not reproduce clean first request")
	}
	// Single normal CLI entrypoint uses one Rerank call; this is an external
	// evaluator/comparison hazard, not proof of an ordinary CLI bug.
}
