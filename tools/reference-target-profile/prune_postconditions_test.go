package targetprofile

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPrunePreservesUnterminatedAuditChain(t *testing.T) {
	for _, storage := range []string{StorageFilesystem, StorageSQLite} {
		for _, ending := range []string{"", "\r", " \t"} {
			t.Run(storage+"/"+ending, func(t *testing.T) {
				root := t.TempDir()
				if storage == StorageSQLite {
					root, _ = sqliteSafetyRoot(t)
				} else {
					for i, body := range []string{"old", "middle", "previous", "current"} {
						writeHistoryArtifact(t, root, body, time.Unix(int64(100+i), 0))
					}
				}
				first, err := PruneHistory(root, 3, false)
				if err != nil || !first.Passed {
					t.Fatalf("initial prune: %+v %v", first, err)
				}
				path := historyAuditPath(root)
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				before := append(bytes.TrimSuffix(raw, []byte("\n")), []byte(ending)...)
				if err := os.WriteFile(path, before, 0o600); err != nil {
					t.Fatal(err)
				}
				audit, err := HistoryAudit(root)
				if err != nil || !audit.Passed || audit.Count != 1 {
					t.Fatalf("accepted input must be a valid chain: %+v %v", audit, err)
				}
				second, err := PruneHistory(root, 2, false)
				if err != nil || !second.Passed || len(second.Deleted) != 1 {
					t.Fatalf("second prune: %+v %v", second, err)
				}
				after, err := os.ReadFile(path)
				if err != nil || !bytes.HasPrefix(after, before) {
					t.Fatalf("existing audit bytes changed: %v", err)
				}
				audit, err = HistoryAudit(root)
				if err != nil || !audit.Passed || audit.Count != 2 || audit.Records[1].PreviousDigest != audit.Records[0].RecordDigest {
					t.Fatalf("successful prune damaged audit chain: %+v %v", audit, err)
				}
				checkpoint, err := BuildHistoryAuditCheckpoint(root)
				if err != nil || !checkpoint.Passed || checkpoint.Count != 2 || checkpoint.AuditDigest != digestBytes(after) {
					t.Fatalf("checkpoint does not match appended bytes: %+v %v", checkpoint, err)
				}
			})
		}
	}
}

func TestSQLitePruneRejectsUnexpectedEffects(t *testing.T) {
	for _, effect := range []string{"ignore", "reinsert", "delete-kept", "corrupt-kept", "retime-kept", "move-current", "remove-previous", "insert-extra"} {
		t.Run(effect, func(t *testing.T) {
			root, digests := sqliteSafetyRoot(t)
			db, err := openSQLite(root)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			when, body := "AFTER", ""
			switch effect {
			case "ignore":
				when, body = "BEFORE", "SELECT RAISE(IGNORE);"
			case "reinsert":
				body = "INSERT INTO artifacts VALUES(OLD.digest,OLD.content,OLD.created_at);"
			case "delete-kept":
				body = "DELETE FROM artifacts WHERE digest='" + digests[3] + "';"
			case "corrupt-kept":
				body = "UPDATE artifacts SET content=x'00' WHERE digest='" + digests[2] + "';"
			case "retime-kept":
				body = "UPDATE artifacts SET created_at='2030-01-01T00:00:00Z' WHERE digest='" + digests[2] + "';"
			case "move-current":
				body = "UPDATE state SET value='" + digests[2] + "' WHERE key='current';"
			case "remove-previous":
				body = "DELETE FROM state WHERE key='previous';"
			case "insert-extra":
				body = "INSERT INTO artifacts VALUES('" + digestBytes([]byte("extra")) + "',CAST('extra' AS BLOB),'2026-01-01T00:00:00Z');"
			}
			// The oldest row is processed last: an earlier delete must roll back too.
			if _, err := db.Exec("CREATE TRIGGER unexpected_effect " + when + " DELETE ON artifacts WHEN OLD.digest='" + digests[0] + "' BEGIN " + body + " END"); err != nil {
				t.Fatal(err)
			}
			receipt, err := PruneHistory(root, 2, false)
			if err != nil || receipt.Passed || receipt.Reason == "" || len(receipt.Deleted) != 0 {
				t.Fatalf("unexpected effects reported as committed prune: %+v %v", receipt, err)
			}
			entries, err := sqliteHistory(root)
			if err != nil || len(entries) != 4 {
				t.Fatalf("rejected prune did not restore artifacts: %+v %v", entries, err)
			}
			for i, entry := range entries {
				if !entry.Verified || entry.Digest != digests[3-i] || !entry.Modified.Equal(time.Unix(int64(103-i), 0)) {
					t.Fatalf("rejected prune changed retained metadata/bytes: %+v", entries)
				}
			}
			for key, want := range map[string]string{"current": digests[3], "previous": digests[2]} {
				var got string
				if err := db.QueryRow("SELECT value FROM state WHERE key=?", key).Scan(&got); err != nil || got != want {
					t.Fatalf("rejected prune changed %s: %q %v", key, got, err)
				}
			}
			for _, path := range []string{historyAuditPath(root), filepath.Join(root, ".update-lock")} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("rejected prune left audit or lock: %s %v", path, err)
				}
			}
		})
	}
}

func TestSQLitePruneAllowsLoggingTrigger(t *testing.T) {
	root, _ := sqliteSafetyRoot(t)
	db, err := openSQLite(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE deletion_log(digest TEXT);
CREATE TRIGGER log_delete AFTER DELETE ON artifacts BEGIN INSERT INTO deletion_log VALUES(OLD.digest); END`); err != nil {
		t.Fatal(err)
	}
	receipt, err := PruneHistory(root, 2, false)
	if err != nil || !receipt.Passed || len(receipt.Deleted) != 2 {
		t.Fatalf("logging trigger blocked valid prune: %+v %v", receipt, err)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM deletion_log").Scan(&count); err != nil || count != 2 {
		t.Fatalf("logging trigger did not commit with prune: count=%d err=%v", count, err)
	}
}
