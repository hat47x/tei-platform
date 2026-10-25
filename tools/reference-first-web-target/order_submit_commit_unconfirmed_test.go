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

// This fake Store models lost acknowledgement after a possible Commit.
// It is synthetic; it does not provide a durable receipt or prove a DB write.
type uncertainCommitOrderStore struct {
	submissions int
	reads       int
}

func (s *uncertainCommitOrderStore) Submit(context.Context, int64) error {
	s.submissions++
	// A low-level error may contain a credential; it must not reach HTTP.
	return fmt.Errorf("postgres://fixture-secret@internal-db: %w", errOrderSubmitCommitUnconfirmed)
}

func (s *uncertainCommitOrderStore) ReadStatus(context.Context, int64) (string, error) {
	s.reads++
	return "", errors.New("unexpected observation after uncertain commit")
}

func TestSubmitCommitUnconfirmedRetainsCategoryAndDoesNotRetry(t *testing.T) {
	ctx := context.Background()
	direct := &uncertainCommitOrderStore{}
	result, err := newOrderSubmitRunner(t, direct).RunCommand(ctx, orderSubmitInput)
	if !errors.Is(err, errOrderSubmitCommitUnconfirmed) ||
		result.Stage != "command-commit-unconfirmed" ||
		result.Command != nil || result.Observation != nil ||
		direct.submissions != 1 || direct.reads != 0 {
		t.Fatalf("lost commit acknowledgement changed category or retried: result=%+v err=%v submits=%d reads=%d",
			result, err, direct.submissions, direct.reads)
	}
	if strings.Contains(err.Error(), "fixture-secret") {
		t.Fatal("private Provider detail escaped the Adapter")
	}

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
	httpStore := &uncertainCommitOrderStore{}
	server, err := newWebTargetServer(
		projection, newReferenceReadScenarioRunner(loaded.World),
		newOrderSubmitRunner(t, httpStore),
	)
	if err != nil {
		t.Fatal(err)
	}

	body := `{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder,
		httptest.NewRequest(http.MethodPost, "/scenario/submit", strings.NewReader(body)))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("uncertain Commit must not be HTTP 422: HTTP=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got webCommandScenarioErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Result.Stage != "command-commit-unconfirmed" ||
		got.Result.Command != nil || got.Result.Observation != nil ||
		!strings.Contains(got.ExecutionError, errOrderSubmitCommitUnconfirmed.Error()) ||
		httpStore.submissions != 1 || httpStore.reads != 0 {
		t.Fatalf("uncertain Commit is misreported or retried: response=%+v submits=%d reads=%d",
			got, httpStore.submissions, httpStore.reads)
	}
	if strings.Contains(recorder.Body.String(), "fixture-secret") ||
		strings.Contains(recorder.Body.String(), "postgres://") ||
		strings.Contains(recorder.Body.String(), "order submit effect failed") {
		t.Fatalf("HTTP response leaked Provider details or claimed failure: %s", recorder.Body.String())
	}
}
