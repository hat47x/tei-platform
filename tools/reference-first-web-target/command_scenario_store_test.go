//go:build !tei_repair_only

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// fakeSubmitStore counts writes without a database. It is not PostgreSQL evidence.
type fakeSubmitStore struct {
	orderIDs []int64
	err      error
}

func (f *fakeSubmitStore) SubmitOrder(_ context.Context, orderID int64) (int64, error) {
	f.orderIDs = append(f.orderIDs, orderID)
	return 7, f.err
}

func storeBackedRunner(t *testing.T, store orderSubmitStore) *referenceCommandScenarioRunner {
	t.Helper()
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newReferenceCommandScenarioRunner(loaded.World,
		legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	runner.submitStore = store
	return runner
}

var storeScenarioInput = commandScenarioInput{
	LoginID: "legacy-order-editor", Credential: "editor-secret", From: "2026-09-18T08:30:00Z", RowIndex: 0,
}

func TestSubmitStoreIsWrittenOnceAfterAuthorizationAndKeepsObservationSynthetic(t *testing.T) {
	store := &fakeSubmitStore{}
	granted := 0
	runner := storeBackedRunner(t, store)
	runner.hostEffect = func(context.Context, string) error { granted++; return nil }
	result, err := runner.RunCommand(context.Background(), storeScenarioInput)
	if err != nil || result.Stage != "completed" {
		t.Fatalf("stage=%s err=%v", result.Stage, err)
	}
	if len(store.orderIDs) != 1 || store.orderIDs[0] != 1001 || granted != 1 {
		t.Fatalf("store writes=%v grants=%d", store.orderIDs, granted)
	}
	if result.Observation == nil || result.Observation.Nature != "synthetic" {
		t.Fatalf("a store must not turn the in-process Observation into real evidence: %+v", result.Observation)
	}
}

// S05: the Host effect Grant is checked at the Adapter, directly before the
// store. A denied Grant leaves the store untouched (dispatch 0 at the store).
func TestSubmitStoreIsNotWrittenWithoutACurrentHostEffectGrant(t *testing.T) {
	store := &fakeSubmitStore{}
	runner := storeBackedRunner(t, store)
	runner.hostEffect = func(context.Context, string) error { return errors.New("revoked") }
	result, err := runner.RunCommand(context.Background(), storeScenarioInput)
	if err == nil || !strings.Contains(err.Error(), "HOST_EFFECT_GRANT_UNAVAILABLE") {
		t.Fatalf("expected the opaque grant refusal, got %v", err)
	}
	if len(store.orderIDs) != 0 || result.Command != nil || result.Observation != nil {
		t.Fatalf("effect happened without a Grant: writes=%v command=%+v", store.orderIDs, result.Command)
	}
}

func TestSubmitStoreFailureIsAnExecutionErrorWithoutCommandOrObservation(t *testing.T) {
	store := &fakeSubmitStore{err: errors.New("connection reset")}
	runner := storeBackedRunner(t, store)
	result, err := runner.RunCommand(context.Background(), storeScenarioInput)
	if err == nil {
		t.Fatal("a failed store write was reported as success")
	}
	if result.Command != nil || result.Observation != nil || result.Stage == "completed" {
		t.Fatalf("partial result presented as success: %+v", result)
	}
}

// With no store the runner keeps its previous, purely synthetic behaviour.
func TestWithoutAStoreTheRunnerIsUnchanged(t *testing.T) {
	runner := storeBackedRunner(t, nil)
	result, err := runner.RunCommand(context.Background(), storeScenarioInput)
	if err != nil || result.Stage != "completed" {
		t.Fatalf("stage=%s err=%v", result.Stage, err)
	}
}
