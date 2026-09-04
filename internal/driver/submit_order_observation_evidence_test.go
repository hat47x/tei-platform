package driver_test

import (
	"context"
	"fmt"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
)

// syntheticSubmitOrderObservationEvidenceは、Gate BでCommandとObservationの境界を確認するためだけのtest-local Evidenceです。
// production向けの汎用Observation schemaではなく、実外部状態を確認したEvidenceでもありません。
type syntheticSubmitOrderObservationEvidence struct {
	Nature          driver.ObservationEvidenceNature
	CommandIdentity core.CanonicalID
	ReadBoundary    string
	ObservedState   string
	ExpectedState   string
	Matched         bool
}

func observeSyntheticSubmitOrderState(
	ctx context.Context,
	commandIdentity core.CanonicalID,
	reader *syntheticOrderStateReader,
	orderID string,
	expectedState string,
) (syntheticSubmitOrderObservationEvidence, error) {
	evidence := syntheticSubmitOrderObservationEvidence{
		Nature:          driver.ObservationEvidenceSynthetic,
		CommandIdentity: commandIdentity,
		ReadBoundary:    string(reader.Identity()),
		ExpectedState:   expectedState,
	}

	execution, err := driver.ExecuteObservationRead(
		ctx,
		evidence.ReadBoundary,
		evidence.Nature,
		func(ctx context.Context) (any, error) {
			return reader.Read(ctx, orderID)
		},
	)
	if err != nil {
		return evidence, err
	}

	observedState, ok := execution.Result.(string)
	if !ok {
		return evidence, fmt.Errorf("Observation read resultの型が不正です: %T", execution.Result)
	}
	evidence.ObservedState = observedState
	evidence.Matched = observedState == expectedState
	return evidence, nil
}

func TestSyntheticSubmitOrderObservationEvidenceSeparatesReadResultFromExpectation(t *testing.T) {
	store := &syntheticOrderStateStore{states: map[string]string{
		"order-1001": "processing",
	}}
	reader := &syntheticOrderStateReader{store: store}

	evidence, err := observeSyntheticSubmitOrderState(
		context.Background(),
		"sales.order.submit",
		reader,
		"order-1001",
		"submitted",
	)
	if err != nil {
		t.Fatalf("Observation read自体は成功するはずです: %v", err)
	}
	if reader.calls != 1 {
		t.Fatalf("Observation readの呼出回数=%d, want 1", reader.calls)
	}
	if evidence.Nature != driver.ObservationEvidenceSynthetic {
		t.Fatalf("Evidenceの性質=%q, want synthetic", evidence.Nature)
	}
	if evidence.CommandIdentity != "sales.order.submit" {
		t.Fatalf("Command identity=%s", evidence.CommandIdentity)
	}
	if evidence.ReadBoundary != "synthetic.order_state.read" {
		t.Fatalf("read boundary=%s", evidence.ReadBoundary)
	}
	if evidence.ObservedState != "processing" {
		t.Fatalf("Observed state=%q, want processing", evidence.ObservedState)
	}
	if evidence.ExpectedState != "submitted" {
		t.Fatalf("Expected state=%q, want submitted", evidence.ExpectedState)
	}
	if evidence.Matched {
		t.Fatal("read成功を期待状態との一致として扱っています")
	}
}

func TestSyntheticSubmitOrderObservationEvidenceKeepsReadErrorSeparateFromExpectation(t *testing.T) {
	reader := &syntheticOrderStateReader{
		store: &syntheticOrderStateStore{states: map[string]string{}},
		err:   fmt.Errorf("synthetic read unavailable"),
	}

	evidence, err := observeSyntheticSubmitOrderState(
		context.Background(),
		"sales.order.submit",
		reader,
		"order-1001",
		"submitted",
	)
	if err == nil {
		t.Fatal("Observation Execution Errorが失われています")
	}
	if evidence.Nature != driver.ObservationEvidenceSynthetic || evidence.ReadBoundary != "synthetic.order_state.read" {
		t.Fatalf("Observation失敗時のEvidence identity=%#v", evidence)
	}
	if evidence.ObservedState != "" || evidence.Matched {
		t.Fatalf("read失敗をObserved stateまたは期待一致として扱っています: %#v", evidence)
	}
}
