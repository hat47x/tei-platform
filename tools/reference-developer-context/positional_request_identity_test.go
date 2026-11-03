// Actual Rerank request binding: the positional cN labels belong to one
// ordered candidate snapshot, not a stable document identity across attempts.
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureC0Instruction(t *testing.T, report *Report) string {
	t.Helper()
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Questions map[string]struct {
				Instructions string `json:"instructions"`
			} `json:"questions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Errorf("decode rerank request: %v", err)
			http.Error(w, "invalid", 400)
			return
		}
		captured = input.Questions["c0"].Instructions
		replies := make(map[string]any, len(input.Questions))
		for key := range input.Questions {
			replies[key] = map[string]any{
				"type":   "choice",
				"choice": "uncertain",
				"probabilities": map[string]float64{
					"relevant":     0.1,
					"uncertain":    0.8,
					"not_relevant": 0.1,
				},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"answers": replies}); err != nil {
			t.Errorf("encode reply: %v", err)
		}
	}))
	defer server.Close()
	Rerank(context.Background(), report, server.URL+"/v1/systemone", "fixture-only-model")
	if report.Evaluation.State != "model-ranked" || captured == "" {
		t.Fatalf("expected captured current request, got state=%+v c0=%q", report.Evaluation, captured)
	}
	return captured
}

func TestRerankQuestionC0BindsDifferentCandidatesAcrossAttempts(t *testing.T) {
	root, target, direct := coverageFixture(t)
	before, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	// Capture actual HTTP request, not only the pre-sorted Report state.
	instruction0 := captureC0Instruction(t, &before)
	if !strings.Contains(instruction0, direct) {
		t.Fatalf("initial c0 did not refer to direct test: %q", instruction0)
	}
	added := "internal/validation/a00_test.go"
	dest := filepath.Join(root, filepath.FromSlash(added))
	if err := os.WriteFile(dest, []byte("package validation\n"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := Discover(root, target)
	if err != nil {
		t.Fatal(err)
	}
	instruction1 := captureC0Instruction(t, &after)
	if !strings.Contains(instruction1, added) || strings.Contains(instruction1, direct) {
		t.Fatalf("changed c0 mapped to wrong candidate: %q", instruction1)
	}
	if instruction0 == instruction1 {
		t.Fatal("same c0 question label unexpectedly held same instructions across attempts")
	}
	// A method application must pair the model's answer with that attempt's
	// Candidate Path+SHA+Basis and actual request, not with bare c0.
}
