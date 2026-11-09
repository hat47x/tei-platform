package targetprofile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sqliteSafetyRoot(t *testing.T) (string, []string) {
	t.Helper()
	root := t.TempDir()
	db, err := openSQLite(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var digests []string
	for i, body := range []string{"old", "middle", "previous", "current"} {
		digest := digestBytes([]byte(body))
		digests = append(digests, digest)
		if _, err := db.Exec(`INSERT INTO artifacts(digest,content,created_at) VALUES(?,?,?)`, digest, []byte(body), time.Unix(int64(100+i), 0).UTC().Format(time.RFC3339Nano)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO state(key,value) VALUES('current',?),('previous',?)`, digests[3], digests[2]); err != nil {
		t.Fatal(err)
	}
	return root, digests
}

func TestSQLiteWritersRespectUpdateLock(t *testing.T) {
	root, digests := sqliteSafetyRoot(t)
	before, err := os.ReadFile(filepath.Join(root, "artifacts.db"))
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, ".update-lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		t.Fatal(err)
	}
	check := func(passed bool, reason string, err error) {
		t.Helper()
		if err != nil || passed || !strings.Contains(reason, "local update lock unavailable") {
			t.Fatalf("sqlite writer bypassed lock: passed=%v reason=%q err=%v", passed, reason, err)
		}
		now, err := os.ReadFile(filepath.Join(root, "artifacts.db"))
		if err != nil || string(now) != string(before) {
			t.Fatal("blocked writer changed SQLite bytes")
		}
		if info, err := os.Stat(lock); err != nil || !info.IsDir() {
			t.Fatalf("writer released another owner's lock: %v", err)
		}
	}
	r, err := Rollback(root)
	check(r.Passed, r.Reason, err)
	r, err = RollbackTo(root, digests[0])
	check(r.Passed, r.Reason, err)
	for _, dryRun := range []bool{false, true} {
		r, err := PruneHistory(root, 2, dryRun)
		check(r.Passed, r.Reason, err)
		if len(r.Deleted) != 0 || len(r.WouldDelete) != 0 {
			t.Fatal("blocked prune reported work")
		}
	}
	_, _, _, candidates := selectionPaths(t)
	profile, err := Load(filepath.Join(repoRoot(t), "tools/reference-target-profile/testdata/desktop-replace.json"))
	if err != nil {
		t.Fatal(err)
	}
	profile.Storage = StorageSQLite
	raw, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(t.TempDir(), "profile.json")
	if err := os.WriteFile(profilePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := Deploy(context.Background(), DeployOptions{ProfilePath: profilePath, CandidatesDir: candidates, RootDir: root})
	check(d.Passed, d.Reason, err)
}

func TestSQLiteDeployValidatesBeforeStateChange(t *testing.T) {
	for _, state := range []string{"corrupt-target", "corrupt-current", "missing-current"} {
		t.Run(state, func(t *testing.T) {
			root, digests := sqliteSafetyRoot(t)
			db, err := openSQLite(root)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			artifact := []byte("fresh candidate")
			switch state {
			case "corrupt-target":
				artifact = []byte("old")
				_, err = db.Exec(`UPDATE artifacts SET content=? WHERE digest=?`, []byte("damaged evidence"), digests[0])
			case "corrupt-current":
				_, err = db.Exec(`UPDATE artifacts SET content=? WHERE digest=?`, []byte("damaged evidence"), digests[3])
			case "missing-current":
				_, err = db.Exec(`DELETE FROM artifacts WHERE digest=?`, digests[3])
			}
			if err != nil {
				t.Fatal(err)
			}
			receipt := DeployReceipt{}
			if err := deploySQLite(context.Background(), root, artifact, &receipt); err == nil {
				t.Fatal("invalid stored artifact was accepted")
			}
			var current, previous string
			if err := db.QueryRow(`SELECT value FROM state WHERE key='current'`).Scan(&current); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRow(`SELECT value FROM state WHERE key='previous'`).Scan(&previous); err != nil {
				t.Fatal(err)
			}
			if current != digests[3] || previous != digests[2] || receipt.Installed {
				t.Fatalf("failed deploy moved pointers: current=%s previous=%s receipt=%+v", current, previous, receipt)
			}
			var count int
			if err := db.QueryRow(`SELECT count(*) FROM artifacts WHERE digest=?`, digestBytes([]byte("fresh candidate"))).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("rejected deploy left inserted candidate")
			}
		})
	}
}

func TestSQLitePruneReportsOnlyCommittedDeletions(t *testing.T) {
	for _, failure := range []string{"protected", "delete-trigger"} {
		t.Run(failure, func(t *testing.T) {
			root, digests := sqliteSafetyRoot(t)
			db, err := openSQLite(root)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			keep := 1
			if failure == "delete-trigger" {
				keep = 2
				if _, err := db.Exec(`CREATE TRIGGER stop_delete BEFORE DELETE ON artifacts WHEN OLD.digest='` + digests[0] + `' BEGIN SELECT RAISE(ABORT,'synthetic delete failure'); END`); err != nil {
					t.Fatal(err)
				}
			}
			receipt, err := PruneHistory(root, keep, false)
			if err != nil || receipt.Passed || receipt.Reason == "" || len(receipt.Deleted) != 0 {
				t.Fatalf("uncommitted deletion was reported: %+v %v", receipt, err)
			}
			var count int
			if err := db.QueryRow(`SELECT count(*) FROM artifacts`).Scan(&count); err != nil || count != 4 {
				t.Fatalf("failed prune changed retained artifacts: count=%d err=%v", count, err)
			}
			if _, err := os.Stat(filepath.Join(root, "history-prune-audit.jsonl")); !os.IsNotExist(err) {
				t.Fatalf("failed prune appended success audit: %v", err)
			}
		})
	}
}

func TestSQLitePruneRetainsCommittedResultWhenAuditAppendFails(t *testing.T) {
	root, _ := sqliteSafetyRoot(t)
	audit := historyAuditPath(root)
	if err := os.WriteFile(audit, nil, 0o400); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(audit, 0o600)
	if f, err := os.OpenFile(audit, os.O_WRONLY|os.O_APPEND, 0); err == nil {
		f.Close()
		t.Skip("file permissions are bypassed by this user/filesystem")
	}
	receipt, err := PruneHistory(root, 2, false)
	if err != nil || receipt.Passed || len(receipt.Deleted) != 2 || !strings.Contains(receipt.Reason, "audit append failed") {
		t.Fatalf("committed change and audit failure conflated: %+v %v", receipt, err)
	}
	history, err := History(root)
	if err != nil || !history.Passed || len(history.Entries) != 2 {
		t.Fatalf("committed prune was lost: %+v %v", history, err)
	}
	// A valid empty chain is not evidence that no deletion happened.
	chain, err := HistoryAudit(root)
	if err != nil || !chain.Passed || chain.Count != 0 {
		t.Fatalf("unexpected remaining chain: %+v %v", chain, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".update-lock")); !os.IsNotExist(err) {
		t.Fatalf("audit failure left lock: %v", err)
	}
}
