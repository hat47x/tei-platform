//go:build !tei_repair_only && !tei_frozen_world

package main

import (
	"context"
	"testing"

	"tei/internal/composition"
)

func TestPostgresScenarioBatchObservationIsReadOnlyAndSeparatelyGranted(t *testing.T) {
	s := postgresTestState(t)
	ctx := context.Background()
	service, err := wireBatchObservation(ctx, composition.NewRegistry(), s)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Withdraw(1); _ = service.Reap(ctx) })
	if _, err := s.db.ExecContext(ctx, "UPDATE public.orders SET status='submitted' WHERE order_id=1001; UPDATE public.tei_reference_execution_grants SET revoked=true WHERE provider_id IN ('order-submit','order-batch')"); err != nil {
		t.Fatal(err)
	}
	var before, after string
	if err := s.db.QueryRowContext(ctx, "SELECT xmin::text FROM public.orders WHERE order_id=1001").Scan(&before); err != nil {
		t.Fatal(err)
	}
	result, binding, err := service.Invoke(ctx, []int64{1003, 1001})
	if err != nil || len(result.Items) != 2 || binding.ContractID != batchObservationContract.ID ||
		result.Items[0].OrderID != 1003 || result.Items[0].Observation.Result != "draft" || result.Items[1].Observation.Result != "submitted" || result.Items[1].Observation.Nature != "external-observed" {
		t.Fatalf("observation: %+v %v", result, err)
	}
	if err := s.db.QueryRowContext(ctx, "SELECT xmin::text FROM public.orders WHERE order_id=1001").Scan(&after); err != nil || before != after {
		t.Fatal("observation changed order")
	}
	result, _, err = service.Invoke(ctx, []int64{1001, 9999})
	if err == nil || len(result.Items) != 0 {
		t.Fatal("missing order returned partial success")
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE public.tei_reference_execution_grants SET revoked=true WHERE provider_id='order-batch-observe'"); err != nil {
		t.Fatal(err)
	}
	result, _, err = service.Invoke(ctx, []int64{1001})
	if err == nil || len(result.Items) != 0 {
		t.Fatal("revoked observer returned state")
	}
}
