package effectfence

import (
	"context"
	"errors"
	"testing"
)

func TestRequirementStopsSideEffectBeforeCommit(t *testing.T) {
	m := setup(t)
	p := mustPrepare(t, m, request("guarded-commit"))
	calls := 0
	apply := func(ctx context.Context) (Result, error) {
		calls++
		return m.Commit(ctx, p)
	}
	out, err := CommitWithRequirement(context.Background(), m.BoundaryContract(), GuaranteeDurable, apply)
	var unsupported *GuaranteeUnavailableError
	if !errors.As(err, &unsupported) || out.Disposition != "" || calls != 0 {
		t.Fatalf("unsupported guarantee reached write: %+v %+v calls=%d", out, err, calls)
	}
	if _, version := m.Observe(scope); version != 0 {
		t.Fatal("unsupported guarantee produced effect")
	}
	out, err = CommitWithRequirement(context.Background(), m.BoundaryContract(), GuaranteeProcess, apply)
	if err != nil || out.Disposition != Committed || calls != 1 {
		t.Fatalf("explicit synthetic test guarantee not admitted: %+v %v calls=%d", out, err, calls)
	}
}

func TestRequirementNeverSilentlyDowngrades(t *testing.T) {
	for _, required := range []Guarantee{"", GuaranteeDurable, "invented"} {
		called := false
		_, err := CommitWithRequirement(context.Background(), BoundaryContract{}, required, func(context.Context) (Result, error) {
			called = true
			return Result{Disposition: Committed}, nil
		})
		if err == nil || called {
			t.Fatalf("unsupported requirement was not blocked: %q %v", required, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := CommitWithRequirement(ctx, NewMemoryStore().BoundaryContract(), GuaranteeProcess, func(context.Context) (Result, error) {
		t.Fatal("canceled effect was invoked")
		return Result{}, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected canceled state: %v", err)
	}
}

func TestDurableRequirementNeedsIndependentEvidence(t *testing.T) {
	declaration := BoundaryContract{
		Boundary:                BoundaryTransaction,
		AuthorityCheckedAtWrite: true,
		ResourceCheckedAtWrite:  true,
		ReceiptStoredWithWrite:  true,
		SubjectLinksCovered:     true,
		DurableReceipt:          true,
		RecoveryLookup:          true,
		Proof:                   ProofProvider,
		ProofReference:          "test-fixture-provider",
	}
	called := 0
	apply := func(context.Context) (Result, error) {
		called++
		return Result{Disposition: Committed}, nil
	}
	for _, verify := range []EvidenceVerifier{nil, func(BoundaryContract) bool { return false }} {
		_, err := CommitWithVerifiedRequirement(context.Background(), declaration, GuaranteeDurable, verify, apply)
		var unavailable *GuaranteeUnavailableError
		if !errors.As(err, &unavailable) || called != 0 {
			t.Fatalf("unverified provider was allowed: %v called=%d", err, called)
		}
	}
	result, err := CommitWithVerifiedRequirement(context.Background(), declaration, GuaranteeDurable, func(c BoundaryContract) bool {
		return c.ProofReference == "test-fixture-provider"
	}, apply)
	if err != nil || result.Disposition != Committed || called != 1 {
		t.Fatalf("trusted proof was not used: %+v err=%v called=%d", result, err, called)
	}
}
