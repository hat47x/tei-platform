package driver_test

import (
	"context"
	"fmt"
	"testing"

	"tei/internal/driver"
)

func TestExecuteObservationReadKeepsExplicitIdentityNatureAndResult(t *testing.T) {
	calls := 0
	execution, err := driver.ExecuteObservationRead(
		context.Background(),
		"order_entry.order_status/read",
		driver.ObservationEvidenceExternalObserved,
		func(context.Context) (any, error) {
			calls++
			return map[string]any{"status": "submitted"}, nil
		},
	)
	if err != nil {
		t.Fatalf("Observation readを完了できません: %v", err)
	}
	if calls != 1 {
		t.Fatalf("Observation read calls=%d, want 1", calls)
	}
	if execution.ReadIdentity != "order_entry.order_status/read" {
		t.Fatalf("ReadIdentity=%q", execution.ReadIdentity)
	}
	if execution.Nature != driver.ObservationEvidenceExternalObserved {
		t.Fatalf("Nature=%q", execution.Nature)
	}
	result, ok := execution.Result.(map[string]any)
	if !ok || result["status"] != "submitted" {
		t.Fatalf("Result=%#v", execution.Result)
	}
}

func TestExecuteObservationReadKeepsIdentityWhenReadFails(t *testing.T) {
	execution, err := driver.ExecuteObservationRead(
		context.Background(),
		"synthetic.order_state.read",
		driver.ObservationEvidenceSynthetic,
		func(context.Context) (any, error) {
			return nil, fmt.Errorf("read unavailable")
		},
	)
	if err == nil {
		t.Fatal("Observation read failureがExecution Errorになっていません")
	}
	if execution.ReadIdentity != "synthetic.order_state.read" {
		t.Fatalf("ReadIdentity=%q", execution.ReadIdentity)
	}
	if execution.Nature != driver.ObservationEvidenceSynthetic {
		t.Fatalf("Nature=%q", execution.Nature)
	}
	if execution.Result != nil {
		t.Fatalf("失敗したObservationにresultがあります: %#v", execution.Result)
	}
}

func TestExecuteObservationReadRejectsMissingMetadataBeforeRead(t *testing.T) {
	tests := []struct {
		name     string
		identity string
		nature   driver.ObservationEvidenceNature
		read     driver.ObservationReadFunc
	}{
		{
			name:     "identityなし",
			identity: " ",
			nature:   driver.ObservationEvidenceSynthetic,
			read: func(context.Context) (any, error) {
				return "unexpected", nil
			},
		},
		{
			name:     "Evidence natureなし",
			identity: "synthetic.order_state.read",
			read: func(context.Context) (any, error) {
				return "unexpected", nil
			},
		},
		{
			name:     "read関数なし",
			identity: "synthetic.order_state.read",
			nature:   driver.ObservationEvidenceSynthetic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := driver.ExecuteObservationRead(context.Background(), tt.identity, tt.nature, tt.read)
			if err == nil {
				t.Fatal("不完全なObservation read指定を受理しています")
			}
		})
	}
}
