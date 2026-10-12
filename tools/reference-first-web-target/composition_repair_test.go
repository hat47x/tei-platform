//go:build !tei_order_only

package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/workspace"
)

type wrongApplicationLookup struct{ customerLookupRunner }

func (*wrongApplicationLookup) ApplicationID() string { return "order_entry" }

func TestCompositionRepairPreservesResultsErrorsAndPermission(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	loaded, err := workspace.Load(filepath.Join(filepath.Dir(file), "..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	runner := newReferenceCustomerLookupScenarioRunner(loaded.World)
	var denied atomic.Bool
	digest, err := runningExecutableDigest()
	if err != nil {
		t.Fatal(err)
	}
	guard := referenceInstanceGuard(digest, map[string]composition.Contract{"repair-customer": repairLookupContract})
	composed, err := wireComposedCustomerLookup(context.Background(), composition.NewRegistry(), runner, nil, digest, func(b composition.Binding) error {
		if denied.Load() {
			return errors.New("host revoked")
		}
		return guard(b)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = composed.service.Withdraw(1); _ = composed.service.Reap(context.Background()) })
	input := customerLookupInput{CustomerID: "11"}
	want, err := runner.RunCustomerLookup(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	got, err := composed.RunCustomerLookup(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Composition == nil || got.Composition.ContractID != repairLookupContract.ID || got.Composition.ArtifactDigest != digest {
		t.Fatal("missing repair composition provenance")
	}
	got.Composition = nil
	if !reflect.DeepEqual(want, got) {
		t.Fatal("repair service changed Driver result")
	}
	if _, err := composed.RunCustomerLookup(context.Background(), customerLookupInput{CustomerID: "invalid"}); !errors.Is(err, errCustomerLookupInput) {
		t.Fatalf("changed typed input failure: %v", err)
	}
	projection, err := buildWebTargetProjection(loaded.World, projectionConfig{ApplicationID: core.MustCanonicalID("repair_intake"), HandoverBaselineID: "B", BaselineRepositoryRevision: "reference", TargetBuildRevision: "test"})
	if err != nil {
		t.Fatal(err)
	}
	server, err := newWebTargetServer(projection, nil, nil, withCustomerLookup(composed))
	if err != nil {
		t.Fatal(err)
	}
	reply := httptest.NewRecorder()
	server.Handler().ServeHTTP(reply, httptest.NewRequest("GET", "/scenario/customer-lookup?customerId=11", nil))
	if reply.Code != 200 {
		t.Fatalf("composed HTTP: %d %s", reply.Code, reply.Body.String())
	}
	denied.Store(true)
	if _, err := composed.RunCustomerLookup(context.Background(), input); err == nil {
		t.Fatal("revoked instance still executed")
	}
	denied.Store(false)
	if err := composed.service.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if _, err := composed.RunCustomerLookup(context.Background(), input); err == nil {
		t.Fatal("withdrawn instance still executed")
	}
	if _, err := wireComposedCustomerLookup(context.Background(), composition.NewRegistry(), &wrongApplicationLookup{runner}, nil, digest, guard); err == nil {
		t.Fatal("wrong Application runner accepted")
	}
}

func TestReferenceInstanceGuardPinsExactHostConfiguration(t *testing.T) {
	offers := map[string]composition.Contract{"repair-customer": repairLookupContract}
	guard := referenceInstanceGuard("sha256:reference", offers)
	delete(offers, "repair-customer")
	b := composition.Binding{ContractID: repairLookupContract.ID, ContractVersion: "1", ProviderID: "repair-customer", ArtifactDigest: "sha256:reference", Generation: 1}
	if err := guard(b); err != nil {
		t.Fatal("caller mutation changed captured host selection")
	}
	for _, mutate := range []func(*composition.Binding){
		func(b *composition.Binding) { b.ProviderID = "order-read" },
		func(b *composition.Binding) { b.ContractVersion = "2" },
		func(b *composition.Binding) { b.ArtifactDigest = "sha256:other" },
		func(b *composition.Binding) { b.Generation = 2 },
	} {
		bad := b
		mutate(&bad)
		if guard(bad) == nil {
			t.Fatal("mismatched instance admitted")
		}
	}
}
