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

func TestSweepExpiredComparesInstantsAcrossOffsets(t *testing.T) {
	store, _, cleanup := setupLeaseStore(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Now()
	for i, offset := range []int{-12 * 60 * 60, 14 * 60 * 60} {
		for j, delta := range []time.Duration{-time.Minute, time.Minute} {
			issue := int64(100 + 2*i + j)
			if _, err := store.ClaimIssue(ctx, issue, "reviewer", "m1", "codex", "s1", time.Minute); err != nil {
				t.Fatal(err)
			}
			expires := now.Add(delta).In(time.FixedZone("test", offset)).Format(time.RFC3339)
			if _, err := store.db.ExecContext(ctx, `UPDATE claims SET lease_expires=? WHERE issue_number=?`, expires, issue); err != nil {
				t.Fatal(err)
			}
		}
	}
	expired, err := store.SweepExpired(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[int64]bool{}
	for _, c := range expired {
		got[c.IssueNumber] = true
	}
	if len(expired) != 2 || !got[100] || !got[102] {
		t.Fatalf("expired=%+v; want expired instants 100 and 102 only", expired)
	}
}
