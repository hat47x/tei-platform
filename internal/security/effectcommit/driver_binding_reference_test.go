package effectcommit

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/realization"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

// referenceEffectAdapter is test-only wiring for an existing Canonical Binding
// and the existing Runner. It is NOT a production Caller or trusted Grant issuer.
// Its output payload is synthetic; X-TEA business meaning is not accepted here.
type referenceEffectAdapter struct {
	runner Runner
	grant  legacyauthorization.AuthorizationGrant
	last   Result
	calls  int
}

func (a *referenceEffectAdapter) Invoke(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	req, ok := input.(Request)
	if !ok {
		return capabilityinvoke.BusinessResult{}, errors.New("synthetic effect fixture requires Request")
	}
	a.calls++
	got, err := a.runner.Execute(ctx, req, a.grant)
	a.last = got
	if err != nil {
		return capabilityinvoke.BusinessResult{}, err
	}
	switch got.Status {
	case StatusCommitted, StatusReplayed:
		// The Canonical output mode is checked; this fixture is NOT a sales.Order.
		return capabilityinvoke.BusinessResult{
			Capability: capability,
			Mode:       capabilityinvoke.ResultOutput,
			Payload:    map[string]string{"orderId": req.Scope.Resource},
		}, nil
	default:
		// Denial/conflict cannot be fabricated into undeclared output-only Outcomes.
		// No production execution-disposition mapping is chosen by this test.
		return capabilityinvoke.BusinessResult{}, fmt.Errorf("synthetic effect status %s needs an owner-approved disposition", got.Status)
	}
}

type referenceEffectBindingResolver struct{ adapter capabilityinvoke.Adapter }

func (r referenceEffectBindingResolver) ResolveAdapter(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
	if binding.Identity() != "order_entry/sales.order.submit" {
		return nil, fmt.Errorf("synthetic Adapter does not implement binding %s", binding.Identity())
	}
	return r.adapter, nil
}

func effectDriverFixture(t *testing.T, db *memoryProvider) (driver.Resolver, *referenceEffectAdapter, Request) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test fixture")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	req := fixtureRequest()
	req.Scope.Application = "order_entry"
	req.Scope.Capability = "sales.order.submit"
	req.Scope.Resource = "order_entry.submit_order"
	req.Scope.Action = "execute"
	adapter := &referenceEffectAdapter{
		runner: Runner{Provider: db},
		grant:  fixtureGrant(req), // Synthetic Grant; a real Adapter must obtain a trusted current grant.
	}
	return driver.NewResolver(loaded.World), adapter, req
}

func invokeSyntheticEffect(ctx context.Context, resolver driver.Resolver, adapter *referenceEffectAdapter, req Request) (driver.CapabilityInvocation, error) {
	return resolver.InvokeCapability(ctx, "order_entry", "sales.order.submit", referenceEffectBindingResolver{adapter: adapter}, req)
}

func TestDriverBindingReachesExistingEffectCommitRunnerWithoutInventingOutcome(t *testing.T) {
	db := newMemoryProvider()
	resolver, adapter, req := effectDriverFixture(t, db)
	ctx := context.Background()

	first, err := invokeSyntheticEffect(ctx, resolver, adapter, req)
	if err != nil || first.Binding.Binding.Identity() != "order_entry/sales.order.submit" ||
		first.Result.Mode != capabilityinvoke.ResultOutput || adapter.last.Status != StatusCommitted {
		t.Fatalf("first invocation/binding: %+v, runner=%+v, err=%v", first, adapter.last, err)
	}
	if db.version != 1 || len(db.records) != 1 || adapter.calls != 1 {
		t.Fatalf("first write/receipt count: version=%d receipts=%d calls=%d", db.version, len(db.records), adapter.calls)
	}

	replay, err := invokeSyntheticEffect(ctx, resolver, adapter, req)
	if err != nil || replay.Result.Mode != capabilityinvoke.ResultOutput || adapter.last.Status != StatusReplayed ||
		db.version != 1 || len(db.records) != 1 || adapter.calls != 2 {
		t.Fatalf("replay changed effect: %+v runner=%+v err=%v version=%d receipts=%d", replay, adapter.last, err, db.version, len(db.records))
	}

	changed := req
	changed.Payload = []byte("different-payload")
	conflict, err := invokeSyntheticEffect(ctx, resolver, adapter, changed)
	if err == nil || conflict.Result.Mode != "" || adapter.last.Status != StatusConflict ||
		adapter.last.Receipt != (Receipt{}) || db.version != 1 || len(db.records) != 1 {
		t.Fatalf("conflict became business output or effect: %+v runner=%+v err=%v", conflict, adapter.last, err)
	}

	db.enabled = false
	denied, err := invokeSyntheticEffect(ctx, resolver, adapter, req)
	if err == nil || denied.Result.Mode != "" || adapter.last.Status != StatusDenied ||
		adapter.last.Receipt != (Receipt{}) || db.version != 1 || len(db.records) != 1 {
		t.Fatalf("revoked grant exposed business output or receipt: %+v runner=%+v err=%v", denied, adapter.last, err)
	}
}

func TestDriverBindingPreservesCommitUncertainErrorAndSameOperationReplay(t *testing.T) {
	db := newMemoryProvider()
	db.commitApplied = true // Acknowledge failure AFTER applying the memory transaction.
	db.commitError = errors.New("synthetic commit acknowledgement lost")
	resolver, adapter, req := effectDriverFixture(t, db)

	invocation, err := invokeSyntheticEffect(context.Background(), resolver, adapter, req)
	var uncertain *CommitUncertainError
	if !errors.As(err, &uncertain) || invocation.Result.Mode != "" ||
		db.version != 1 || len(db.records) != 1 || adapter.calls != 1 {
		t.Fatalf("uncertain Commit was lost across Driver: result=%+v err=%v version=%d receipts=%d calls=%d", invocation, err, db.version, len(db.records), adapter.calls)
	}

	// Recovery is a NEW explicit invocation using the SAME immutable operation.
	// The test does not claim the Driver automatically retries or runs Lookup.
	db.commitError = nil
	replay, err := invokeSyntheticEffect(context.Background(), resolver, adapter, req)
	if err != nil || replay.Result.Mode != capabilityinvoke.ResultOutput || adapter.last.Status != StatusReplayed ||
		db.version != 1 || len(db.records) != 1 || adapter.calls != 2 {
		t.Fatalf("same-operation replay duplicated effect: result=%+v runner=%+v err=%v version=%d receipts=%d calls=%d", replay, adapter.last, err, db.version, len(db.records), adapter.calls)
	}
}
// postCommitInvalidResultAdapter reproduces the dangerous current ordering:
// the real Runner commits, then the Adapter returns a result which violates the
// existing output-only Canonical contract. This is NOT a production Adapter.
type postCommitInvalidResultAdapter struct{ committed *referenceEffectAdapter }

func (a postCommitInvalidResultAdapter) Invoke(ctx context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	result, err := a.committed.Invoke(ctx, capability, input)
	if err != nil {
		return result, err
	}
	result.Mode = capabilityinvoke.ResultOutcome
	result.Outcome = "denied" // Never declared for output-only sales.order.submit.
	return result, nil
}

func TestDriverRejectsPostCommitInvalidBusinessResultWithoutErasingEffect(t *testing.T) {
	db := newMemoryProvider()
	resolver, committed, req := effectDriverFixture(t, db)
	ctx := context.Background()
	adapters := referenceEffectBindingResolver{
		adapter: postCommitInvalidResultAdapter{committed: committed},
	}

	// A valid request and synthetic Grant let the existing Runner COMMIT once.
	// Only afterward does capabilityinvoke.ValidateBusinessResult reject mode.
	invocation, err := resolver.InvokeCapability(
		ctx, "order_entry", "sales.order.submit", adapters, req,
	)
	if err == nil || invocation.Result.Mode != "" {
		t.Fatalf("invalid post-commit result must be a Driver Execution Error: result=%+v err=%v", invocation, err)
	}
	if committed.last.Status != StatusCommitted || committed.calls != 1 ||
		db.version != 1 || db.value != string(req.Payload) || len(db.records) != 1 {
		t.Fatalf("Execution Error incorrectly implies rollback: status=%s calls=%d version=%d receipts=%d", committed.last.Status, committed.calls, db.version, len(db.records))
	}

	// The same immutable operation can be looked up without another write.
	// This test demonstrates evidence still exists; production recovery needs
	// trusted Grant, Owner meaning and an explicit execution disposition.
	lookedUp, lookupErr := committed.runner.Lookup(ctx, req, committed.grant)
	if lookupErr != nil || lookedUp.Status != StatusReplayed ||
		lookedUp.Receipt.OperationID != req.OperationID ||
		db.version != 1 || committed.calls != 1 {
		t.Fatalf("post-commit evidence cannot be recovered safely: lookup=%+v err=%v", lookedUp, lookupErr)
	}
}
