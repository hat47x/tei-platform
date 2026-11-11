package dispatch

import (
	"context"
	"os"
	"testing"
	"time"
)

func setupLeaseStore(t *testing.T) (*LeaseStore, string, func()) {
	t.Helper()
	path := t.TempDir() + "/lease.db"
	store, err := NewLeaseStore(path)
	if err != nil {
		t.Fatalf("NewLeaseStore: %v", err)
	}
	return store, path, func() { store.Close(); _ = os.Remove(path) }
}

func TestClaimAndRelease(t *testing.T) {
	store, _, cleanup := setupLeaseStore(t)
	defer cleanup()

	ctx := context.Background()
	claim, err := store.ClaimIssue(ctx, 1, "implementer", "m1", "dsh", "s1", 10*time.Minute)
	if err != nil {
		t.Fatalf("ClaimIssue: %v", err)
	}
	if claim.IssueNumber != 1 {
		t.Errorf("ClaimIssue.IssueNumber = %d, want 1", claim.IssueNumber)
	}
	if claim.ModelID != "m1" {
		t.Errorf("ClaimIssue.ModelID = %s, want m1", claim.ModelID)
	}

	// Second claim should fail.
	_, err = store.ClaimIssue(ctx, 1, "implementer", "m2", "dsh", "s2", 10*time.Minute)
	if err != ErrAlreadyClaimed {
		t.Errorf("ClaimIssue second = %v, want ErrAlreadyClaimed", err)
	}

	// Release and re-claim.
	if err := store.Release(ctx, 1); err != nil {
		t.Fatalf("Release: %v", err)
	}
	claim2, err := store.ClaimIssue(ctx, 1, "implementer", "m2", "dsh", "s2", 10*time.Minute)
	if err != nil {
		t.Fatalf("ClaimIssue after release: %v", err)
	}
	if claim2.ModelID != "m2" {
		t.Errorf("ClaimIssue after release.ModelID = %s, want m2", claim2.ModelID)
	}
}

func TestSweepExpired(t *testing.T) {
	store, _, cleanup := setupLeaseStore(t)
	defer cleanup()

	ctx := context.Background()
	_, err := store.ClaimIssue(ctx, 2, "implementer", "m1", "dsh", "s1", 1*time.Minute)
	if err != nil {
		t.Fatalf("ClaimIssue: %v", err)
	}

	// Manually expire the claim.
	_, err = store.db.ExecContext(ctx, `UPDATE claims SET lease_expires = datetime('now', '-1 minute') WHERE issue_number = ?`, 2)
	if err != nil {
		t.Fatalf("expire claim: %v", err)
	}

	expired, err := store.SweepExpired(ctx)
	if err != nil {
		t.Fatalf("SweepExpired: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("SweepExpired len = %d, want 1", len(expired))
	}
	if expired[0].IssueNumber != 2 {
		t.Errorf("SweepExpired[0].IssueNumber = %d, want 2", expired[0].IssueNumber)
	}
	if expired[0].State != "expired" {
		t.Errorf("SweepExpired[0].State = %s, want expired", expired[0].State)
	}
}

func TestHeartbeat(t *testing.T) {
	store, _, cleanup := setupLeaseStore(t)
	defer cleanup()

	ctx := context.Background()
	claim, err := store.ClaimIssue(ctx, 3, "implementer", "m1", "dsh", "s1", 10*time.Minute)
	if err != nil {
		t.Fatalf("ClaimIssue: %v", err)
	}

	if err := store.Heartbeat(ctx, 3, 20*time.Minute); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	got, err := store.GetClaim(ctx, 3)
	if err != nil {
		t.Fatalf("GetClaim: %v", err)
	}
	if got == nil {
		t.Fatal("GetClaim = nil, want claim")
	}
	if !got.LeaseExpires.After(claim.LeaseExpires) {
		t.Errorf("LeaseExpires not extended: got %v, want > %v", got.LeaseExpires, claim.LeaseExpires)
	}
}

func TestParallelClaim(t *testing.T) {
	store, _, cleanup := setupLeaseStore(t)
	defer cleanup()

	ctx := context.Background()
	type result struct {
		index int
		err   error
	}
	results := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func(idx int) {
			_, err := store.ClaimIssue(ctx, 42, "implementer", "m1", "dsh", "", 10*time.Minute)
			results <- result{index: idx, err: err}
		}(i)
	}
	var successes int
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err == nil {
			successes++
		} else if r.err != ErrAlreadyClaimed {
			t.Errorf("unexpected error from worker %d: %v", r.index, r.err)
		}
	}
	if successes != 1 {
		t.Errorf("parallel claims succeeded = %d, want 1", successes)
	}
}

func TestExpiredLeaseRecovery(t *testing.T) {
	store, _, cleanup := setupLeaseStore(t)
	defer cleanup()

	ctx := context.Background()
	_, err := store.ClaimIssue(ctx, 7, "implementer", "m1", "dsh", "s1", 1*time.Minute)
	if err != nil {
		t.Fatalf("ClaimIssue: %v", err)
	}

	// Expire the lease.
	_, err = store.db.ExecContext(ctx, `UPDATE claims SET lease_expires = datetime('now', '-1 minute') WHERE issue_number = ?`, 7)
	if err != nil {
		t.Fatalf("expire claim: %v", err)
	}

	// Sweep should mark it expired.
	expired, err := store.SweepExpired(ctx)
	if err != nil {
		t.Fatalf("SweepExpired: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("SweepExpired len = %d, want 1", len(expired))
	}

	// After sweep, the issue should be reclaimable.
	claim, err := store.ClaimIssue(ctx, 7, "implementer", "m2", "dsh", "s2", 10*time.Minute)
	if err != nil {
		t.Fatalf("ClaimIssue after sweep: %v", err)
	}
	if claim.ModelID != "m2" {
		t.Errorf("ClaimIssue after sweep.ModelID = %s, want m2", claim.ModelID)
	}
}
