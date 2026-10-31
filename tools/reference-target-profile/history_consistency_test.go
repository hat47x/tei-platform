package targetprofile

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoryApprovalRespectsHeldUpdateLock(t *testing.T) {
	for _, storage := range []string{StorageFilesystem, StorageSQLite} {
		t.Run(storage, func(t *testing.T) {
			root := t.TempDir()
			if storage == StorageSQLite {
				root, _ = sqliteSafetyRoot(t)
			}
			if err := os.Mkdir(filepath.Join(root, ".update-lock"), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Run("plan", func(t *testing.T) {
				_, err := BuildHistoryPruneApproval(root, 2)
				if err == nil || !strings.Contains(err.Error(), "local update lock unavailable") {
					t.Fatalf("plan ignored held lock: %v", err)
				}
			})
			t.Run("approved-prune", func(t *testing.T) {
				// A held root must be rejected before reading or verifying these inputs.
				r, err := PruneHistoryApproved(context.Background(), root, 2, false, "unread-approval", "unread-signature", "unread-signers", "signer")
				if err != nil || r.Passed || !strings.Contains(r.Reason, "local update lock unavailable") {
					t.Fatalf("approval was read before lock: %+v %v", r, err)
				}
			})
			if _, err := os.Stat(filepath.Join(root, ".update-lock")); err != nil {
				t.Fatalf("another owner's lock was removed: %v", err)
			}
		})
	}
}

func TestSQLiteHistoryOrdersParsedInstants(t *testing.T) {
	root, digests := sqliteSafetyRoot(t)
	db, err := openSQLite(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// RFC3339Nano strings .12Z and .123Z sort opposite to their instants.
	for i, stamp := range map[int]string{0: "2026-10-29T00:00:00.12Z", 1: "2026-10-29T00:00:00.123Z", 3: "2026-10-29T00:00:00.2Z"} {
		if _, err := db.Exec(`UPDATE artifacts SET created_at=? WHERE digest=?`, stamp, digests[i]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`DELETE FROM artifacts WHERE digest=?`, digests[2]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE state SET value=? WHERE key='previous'`, digests[1]); err != nil {
		t.Fatal(err)
	}
	history, err := History(root)
	if err != nil || !history.Passed || len(history.Entries) != 3 {
		t.Fatalf("history unavailable: %+v %v", history, err)
	}
	if history.Entries[0].Digest != digests[3] || history.Entries[1].Digest != digests[1] || history.Entries[2].Digest != digests[0] {
		t.Fatalf("history used lexical timestamps: %+v", history.Entries)
	}
	plan, err := BuildHistoryPruneApproval(root, 2)
	if err != nil || len(plan.Deleted) != 1 || plan.Deleted[0] != digests[0] {
		t.Fatalf("plan uses a different time order: %+v %v", plan, err)
	}
	r, err := PruneHistory(root, 2, false)
	if err != nil || !r.Passed || len(r.Deleted) != 1 || r.Deleted[0].Digest != digests[0] {
		t.Fatalf("prune did not retain the newest instants: %+v %v", r, err)
	}
}

func TestSQLiteHistoryOrdersOffsetsAndEqualInstants(t *testing.T) {
	root, digests := sqliteSafetyRoot(t)
	db, err := openSQLite(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	stamps := []string{"2026-10-29T01:00:00+01:00", "2026-10-29T00:00:00Z", "2026-10-28T23:30:00-01:00", "2026-10-29T00:15:00Z"}
	for i, stamp := range stamps {
		if _, err := db.Exec(`UPDATE artifacts SET created_at=? WHERE digest=?`, stamp, digests[i]); err != nil {
			t.Fatal(err)
		}
	}
	history, err := History(root)
	if err != nil || !history.Passed || len(history.Entries) != 4 {
		t.Fatalf("history unavailable: %+v %v", history, err)
	}
	// 00:30Z, 00:15Z, then equal 00:00Z instants with descending digests.
	want := []string{digests[2], digests[3], digests[0], digests[1]}
	for i, entry := range history.Entries {
		if entry.Digest != want[i] {
			t.Fatalf("offset/tie order mismatch: %+v", history.Entries)
		}
	}
}

func TestHistoryAuditCheckpointRejectsInvalidInputWithoutMetadata(t *testing.T) {
	for _, kind := range []string{"directory", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path := historyAuditPath(root)
			if kind == "directory" {
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(path, []byte("broken\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			r, err := BuildHistoryAuditCheckpoint(root)
			if err != nil || r.Passed || r.Reason == "" || r.Count != 0 || r.HeadDigest != "" || r.AuditDigest != "" {
				t.Fatalf("invalid checkpoint reported metadata: %+v %v", r, err)
			}
		})
	}
}
