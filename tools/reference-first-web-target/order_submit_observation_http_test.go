//go:build !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
)

const privateObservationError = "postgres://account:password@internal-db.example/private: connection reset"

type observationReadFailureStore struct{ submissions int }

// Submit stands for a successful earlier write; the independent read then fails.
// This is a synthetic transport fixture, not evidence of a real DB commit.
func (s *observationReadFailureStore) Submit(_ context.Context, orderID int64) error {
	if orderID != 1001 {
		return fmt.Errorf("unexpected test order %d", orderID)
	}
	s.submissions++
	return nil
}
func (*observationReadFailureStore) ReadStatus(context.Context, int64) (string, error) {
	return "", errors.New(privateObservationError)
}

func TestSubmitObservationFailureAfterCommandDoesNotExposeProviderError(t *testing.T) {
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	projection, err := buildWebTargetProjection(loaded.World, projectionConfig{
		HandoverBaselineID:         testBaselineID,
		BaselineRepositoryRevision: testBaselineRevision,
		TargetBuildRevision:        testTargetRevision,
		ApplicationID:              core.MustCanonicalID("order_entry"),
	})
	if err != nil {
		t.Fatal(err)
	}
	store := &observationReadFailureStore{}
	runner := newOrderSubmitRunner(t, store)
	server, err := newWebTargetServer(projection, newReferenceReadScenarioRunner(loaded.World), runner)
	if err != nil {
		t.Fatal(err)
	}

	body := `{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/scenario/submit", strings.NewReader(body)))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("HTTP=%d body=%s", w.Code, w.Body.String())
	}
	var got webCommandScenarioErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	const want = "action completion unconfirmed; inspect authoritative state before retrying"
	if got.ExecutionError != want || got.Result.Stage != "observation" ||
		got.Result.Command == nil || got.Result.Observation != nil || store.submissions != 1 {
		t.Fatalf("post-command Observation failure was misrepresented: response=%+v submissions=%d", got, store.submissions)
	}
	if strings.Contains(w.Body.String(), privateObservationError) || strings.Contains(w.Body.String(), "internal-db.example") {
		t.Fatal("provider connection detail leaked to the HTTP caller")
	}
}
