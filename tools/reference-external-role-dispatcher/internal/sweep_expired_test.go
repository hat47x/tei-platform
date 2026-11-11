package dispatch

import (
	"context"
	"testing"
	"time"
)

// A sweep must report exactly the claims it expired in that call, and nothing
// that a previous sweep already expired. The previous implementation expired rows
// with an UPDATE and then re-queried them by `updated_at = datetime('now')`, which
// has second granularity: a sweep that crossed a second boundary returned zero
// rows although the work succeeded, and a second sweep within the same second
// reported the earlier rows again. A repeat run reproduced the first shape 1 time
// in 40 (#4252).
func TestSweepExpiredReportsOnlyWhatItExpired(t *testing.T) {
	store, _, cleanup := setupLeaseStore(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := store.ClaimIssue(ctx, 11, "implementer", "m1", "dsh", "s1", 1*time.Minute); err != nil {
		t.Fatalf("ClaimIssue: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE claims SET lease_expires = datetime('now', '-1 minute') WHERE issue_number = ?`, 11); err != nil {
		t.Fatalf("expire claim: %v", err)
	}

	first, err := store.SweepExpired(ctx)
	if err != nil {
		t.Fatalf("first SweepExpired: %v", err)
	}
	if len(first) != 1 || first[0].IssueNumber != 11 {
		t.Fatalf("first sweep = %+v, want exactly issue 11", first)
	}

	// The same second: the old query would still match this row, so a second
	// sweep would report it again instead of reporting nothing.
	second, err := store.SweepExpired(ctx)
	if err != nil {
		t.Fatalf("second SweepExpired: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second sweep = %+v, want none: a sweep must report only newly expired claims", second)
	}
}

// An active claim is not swept, so the count reflects expiry rather than every
// row in the table.
func TestSweepExpiredLeavesActiveClaimsAlone(t *testing.T) {
	store, _, cleanup := setupLeaseStore(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := store.ClaimIssue(ctx, 12, "implementer", "m1", "dsh", "s1", 10*time.Minute); err != nil {
		t.Fatalf("ClaimIssue: %v", err)
	}
	expired, err := store.SweepExpired(ctx)
	if err != nil {
		t.Fatalf("SweepExpired: %v", err)
	}
	if len(expired) != 0 {
		t.Fatalf("sweep = %+v, want none: the lease is still active", expired)
	}
	if got, err := store.GetClaim(ctx, 12); err != nil || got == nil || got.State != "working" {
		t.Fatalf("GetClaim = %+v, %v; want the claim still working", got, err)
	}
}
