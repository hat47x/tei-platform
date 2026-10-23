//go:build !tei_repair_only

package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The Command has completed, but its subsequent independent Observation failed.
// That failure cannot be used as evidence that the write was rolled back.
func TestSubmitObservationFailureAfterCommandDoesNotExposeProviderError(t *testing.T) {
	const privateError = "postgres://account:password@internal-db.example/private: connection reset"
	result := commandScenarioResult{
		ApplicationID: "order_entry",
		Stage:         "observation",
		Command:       &businessResultProjection{Capability: "sales.order.submit"},
	}
	w := httptest.NewRecorder()
	(&webTargetServer{}).writeCommandScenarioHTTPResult(w, result, errors.New(privateError))

	// Preserve this Reference's existing error HTTP status, without implying rollback.
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
	}
	var got webCommandScenarioErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	const want = "action completion unconfirmed; inspect authoritative state before retrying"
	if got.ExecutionError != want || got.Result.Stage != "observation" ||
		got.Result.Command == nil || got.Result.Observation != nil {
		t.Fatalf("observation failure was misrepresented: %+v", got)
	}
	if strings.Contains(w.Body.String(), privateError) || strings.Contains(w.Body.String(), "internal-db.example") {
		t.Fatal("provider connection detail leaked to the HTTP caller")
	}
}
