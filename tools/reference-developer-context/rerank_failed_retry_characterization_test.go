// Characterization of model annotations on an in-process Report reused after
// a successful Rerank. The normal CLI performs one Rerank per fresh Discover.
// This test does not add a fallback contract or establish real-LLM behavior.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRerankFailedRetryLeavesPriorModelChoiceOnReusedReport(t *testing.T) {
	root := t.TempDir()
	target := "internal/validation/validation.go"
	direct := "internal/validation/validation_test.go"
	other := "internal/validation/other_test.go"
	writeCoverageFixture(t, root, target, "package validation\n")
	writeCoverageFixture(t, root, direct, "package validation_test\n")
	writeCoverageFixture(t, root, other, "package validation\n")

	report, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates) != 2 || report.Candidates[0].Path != direct ||
		report.Candidates[1].Path != other {
		t.Fatalf("unexpected baseline: %+v", report.Candidates)
	}

	// An accepted response deliberately reverses the deterministic ordering.
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"answers":{"c0":{"type":"choice","choice":"not_relevant","probabilities":{"relevant":0,"uncertain":0,"not_relevant":1}},"c1":{"type":"choice","choice":"relevant","probabilities":{"relevant":1,"uncertain":0,"not_relevant":0}}}}`)
	}))
	defer good.Close()

	ctx := context.Background()
	model := "fixture-only-model"
	Rerank(ctx, &report, good.URL+"/v1/systemone", model)
	if report.Evaluation.State != "model-ranked" || report.Candidates[0].Path != other ||
		report.Candidates[0].ModelChoice != "relevant" {
		t.Fatalf("expected initial model ordering: %+v", report)
	}

	// A pre-send rejection changes Evaluation only, not the prior annotations.
	Rerank(ctx, &report, "http://example.invalid/v1/systemone", model)
	if report.Evaluation.State != "model-failed" || report.Candidates[0].Path != other ||
		report.Candidates[0].ModelChoice != "relevant" ||
		report.Candidates[1].ModelChoice != "not_relevant" {
		t.Fatalf("expected stale model fields after pre-send rejection: %+v", report)
	}

	// A POST that reaches the loopback recipient may still result in failure.
	// Capture the actual request, not an inferred "no delivery" from the state.
	requests := make(chan []byte, 2)
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("request read: %v", readErr)
		}
		requests <- body
		http.Error(w, "fixture unavailable", http.StatusServiceUnavailable)
	}))
	defer failing.Close()

	Rerank(ctx, &report, failing.URL+"/v1/systemone", model)
	reusedBody := <-requests
	var reusedRequest struct {
		State struct {
			Candidates []Candidate `json:"candidates"`
		} `json:"state"`
	}
	if err := json.Unmarshal(reusedBody, &reusedRequest); err != nil {
		t.Fatal(err)
	}
	if report.Evaluation.State != "model-failed" || report.Candidates[0].Path != other ||
		report.Candidates[0].ModelChoice != "relevant" ||
		len(reusedRequest.State.Candidates) != 2 ||
		reusedRequest.State.Candidates[0].Path != other ||
		reusedRequest.State.Candidates[0].ModelChoice != "relevant" {
		t.Fatalf("reused report must retain prior choice, including in POST: report=%+v request=%+v",
			report, reusedRequest)
	}

	// The same failing HTTP endpoint on a fresh Discover must neither
	// inherit nor transmit a prior attempt's model choice or sorted position.
	fresh, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	Rerank(ctx, &fresh, failing.URL+"/v1/systemone", model)
	freshBody := <-requests
	var freshRequest struct {
		State struct {
			Candidates []Candidate `json:"candidates"`
		} `json:"state"`
	}
	if err := json.Unmarshal(freshBody, &freshRequest); err != nil {
		t.Fatal(err)
	}
	if fresh.Evaluation.State != "model-failed" || len(fresh.Candidates) != 2 ||
		fresh.Candidates[0].Path != direct || fresh.Candidates[0].ModelChoice != "" ||
		len(freshRequest.State.Candidates) != 2 ||
		freshRequest.State.Candidates[0].Path != direct ||
		freshRequest.State.Candidates[0].ModelChoice != "" {
		t.Fatalf("fresh failed attempt must retain deterministic candidates: report=%+v request=%+v",
			fresh, freshRequest)
	}

	// This demonstrates an evaluator reuse hazard, not a bug in the ordinary
	// single-shot CLI. Both failed attempts are accounted for separately,
	// and only the loopback POST attempts have observed delivery receipts.
}
