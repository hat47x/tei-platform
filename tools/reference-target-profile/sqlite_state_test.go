package targetprofile

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func sqliteStateSnapshot(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT 'artifact:' || digest || ':' || hex(content) || ':' || created_at FROM artifacts
UNION ALL SELECT 'state:' || key || ':' || value FROM state ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var row string
		if err := rows.Scan(&row); err != nil {
			t.Fatal(err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func runSQLiteStateOperation(t *testing.T, operation, root, target string) (bool, string) {
	t.Helper()
	if operation == "deploy" {
		var receipt DeployReceipt
		if err := deploySQLite(context.Background(), root, []byte("fresh candidate"), &receipt); err != nil {
			return false, err.Error()
		}
		return receipt.Installed, ""
	}
	var receipt RollbackReceipt
	var err error
	if operation == "rollback" {
		receipt, err = Rollback(root)
	} else {
		receipt, err = RollbackTo(root, target)
	}
	if err != nil {
		t.Fatal(err)
	}
	return receipt.Passed, receipt.Reason
}

func TestSQLiteStateChangesRejectUnexpectedEffects(t *testing.T) {
	for _, operation := range []string{"deploy", "rollback", "rollback-to"} {
		for _, effect := range []string{"ignore-current", "ignore-previous", "corrupt-target", "delete-backup", "remove-current"} {
			t.Run(operation+"/"+effect, func(t *testing.T) {
				root, digests := sqliteSafetyRoot(t)
				db, err := openSQLite(root)
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close()
				when, key, body := "AFTER", "current", ""
				switch effect {
				case "ignore-current", "ignore-previous":
					when, body = "BEFORE", "SELECT RAISE(IGNORE);"
					if effect == "ignore-previous" {
						key = "previous"
					}
				case "corrupt-target":
					body = "UPDATE artifacts SET content=x'00' WHERE digest=NEW.value;"
				case "delete-backup":
					body = "DELETE FROM artifacts WHERE digest=OLD.value;"
				case "remove-current":
					body = "DELETE FROM state WHERE key='current';"
				}
				if _, err := db.Exec("CREATE TRIGGER changed_result " + when + " UPDATE ON state WHEN NEW.key='" + key + "' BEGIN " + body + " END"); err != nil {
					t.Fatal(err)
				}
				before := sqliteStateSnapshot(t, db)
				passed, reason := runSQLiteStateOperation(t, operation, root, digests[0])
				after := sqliteStateSnapshot(t, db)
				if passed || reason == "" || !reflect.DeepEqual(before, after) {
					t.Fatalf("unexpected state change: passed=%v reason=%q before=%v after=%v", passed, reason, before, after)
				}
				if _, err := os.Stat(filepath.Join(root, ".update-lock")); !os.IsNotExist(err) {
					t.Fatalf("rejected operation left lock: %v", err)
				}
			})
		}
	}
}

func TestSQLiteRollbackRejectsInvalidRetainedCurrent(t *testing.T) {
	for _, operation := range []string{"rollback", "rollback-to"} {
		for _, failure := range []string{"corrupt", "missing"} {
			t.Run(operation+"/"+failure, func(t *testing.T) {
				root, digests := sqliteSafetyRoot(t)
				db, err := openSQLite(root)
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close()
				query := "UPDATE artifacts SET content=x'00' WHERE digest=?"
				if failure == "missing" {
					query = "DELETE FROM artifacts WHERE digest=?"
				}
				if _, err := db.Exec(query, digests[3]); err != nil {
					t.Fatal(err)
				}
				before := sqliteStateSnapshot(t, db)
				passed, reason := runSQLiteStateOperation(t, operation, root, digests[0])
				after := sqliteStateSnapshot(t, db)
				if passed || reason == "" || !reflect.DeepEqual(before, after) {
					t.Fatalf("invalid backup accepted or evidence changed: passed=%v reason=%q before=%v after=%v", passed, reason, before, after)
				}
			})
		}
	}
}

func TestSQLiteStateChangesAllowLoggingTrigger(t *testing.T) {
	for _, operation := range []string{"deploy", "rollback", "rollback-to"} {
		t.Run(operation, func(t *testing.T) {
			root, digests := sqliteSafetyRoot(t)
			db, err := openSQLite(root)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.Exec(`CREATE TABLE changes(key TEXT);
CREATE TRIGGER log_change AFTER UPDATE ON state BEGIN INSERT INTO changes VALUES(NEW.key); END`); err != nil {
				t.Fatal(err)
			}
			passed, reason := runSQLiteStateOperation(t, operation, root, digests[0])
			var count int
			if err := db.QueryRow("SELECT count(*) FROM changes").Scan(&count); err != nil || count != 2 || !passed {
				t.Fatalf("logging trigger blocked valid operation: passed=%v reason=%q count=%d err=%v", passed, reason, count, err)
			}
		})
	}
}

func TestSQLiteRollbackReturnsReverseDigest(t *testing.T) {
	for _, operation := range []string{"rollback", "rollback-to"} {
		t.Run(operation, func(t *testing.T) {
			root, digests := sqliteSafetyRoot(t)
			var receipt RollbackReceipt
			var err error
			if operation == "rollback" {
				receipt, err = Rollback(root)
			} else {
				receipt, err = RollbackTo(root, digests[0])
			}
			if err != nil || !receipt.Passed || receipt.PreviousDigest != digests[3] {
				t.Fatalf("verified backup is absent from receipt: %+v %v", receipt, err)
			}
			reverse, err := RollbackTo(root, receipt.PreviousDigest)
			if err != nil || !reverse.Passed || reverse.ArtifactDigest != digests[3] || reverse.PreviousDigest != receipt.ArtifactDigest {
				t.Fatalf("returned digest cannot support reverse rollback: %+v %v", reverse, err)
			}
		})
	}
}

func TestSQLiteStateChangesWithoutCurrent(t *testing.T) {
	for _, operation := range []string{"deploy", "rollback-to"} {
		for _, keepPrevious := range []bool{false, true} {
			t.Run(operation+"/"+map[bool]string{false: "no-previous", true: "retained-previous"}[keepPrevious], func(t *testing.T) {
				root, digests := sqliteSafetyRoot(t)
				db, err := openSQLite(root)
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close()
				if _, err := db.Exec("DELETE FROM state WHERE key='current'"); err != nil {
					t.Fatal(err)
				}
				wantPrevious := digests[2]
				if !keepPrevious {
					wantPrevious = ""
					if _, err := db.Exec("DELETE FROM state WHERE key='previous'"); err != nil {
						t.Fatal(err)
					}
				}
				var gotPrevious, target string
				if operation == "deploy" {
					var receipt DeployReceipt
					if err := deploySQLite(context.Background(), root, []byte("fresh candidate"), &receipt); err != nil || !receipt.Installed {
						t.Fatalf("initial deploy failed: %+v %v", receipt, err)
					}
					gotPrevious, target = receipt.PreviousDigest, receipt.ArtifactDigest
				} else {
					receipt, err := RollbackTo(root, digests[0])
					if err != nil || !receipt.Passed {
						t.Fatalf("initial rollback-to failed: %+v %v", receipt, err)
					}
					gotPrevious, target = receipt.PreviousDigest, digests[0]
				}
				var current, previous string
				if err := db.QueryRow("SELECT value FROM state WHERE key='current'").Scan(&current); err != nil {
					t.Fatal(err)
				}
				if err := db.QueryRow("SELECT COALESCE((SELECT value FROM state WHERE key='previous'),'')").Scan(&previous); err != nil {
					t.Fatal(err)
				}
				if current != target || previous != wantPrevious || gotPrevious != previous {
					t.Fatalf("initial transition lost previous binding: current=%s previous=%s receiptPrevious=%s", current, previous, gotPrevious)
				}
			})
		}
	}
}
