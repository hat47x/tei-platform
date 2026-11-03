package targetprofile

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS artifacts (
  digest TEXT PRIMARY KEY,
  content BLOB NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS state (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);`

func sqliteRoot(root string) bool {
	_, err := os.Stat(filepath.Join(root, "artifacts.db"))
	return err == nil
}

func openSQLite(root string) (*sql.DB, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(root, "artifacts.db"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(sqliteSchema); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func stateValue(tx *sql.Tx, key string) (string, error) {
	var value string
	err := tx.QueryRow(`SELECT value FROM state WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func putState(tx *sql.Tx, key, value string) error {
	_, err := tx.Exec(`INSERT INTO state(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func deploySQLite(ctx context.Context, root string, artifact []byte, receipt *DeployReceipt) error {
	db, err := openSQLite(root)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	digest := digestBytes(artifact)
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO artifacts(digest, content, created_at) VALUES(?, ?, ?)`,
		digest, artifact, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if err := verifySQLiteArtifact(tx, digest); err != nil {
		return err
	}
	current, err := stateValue(tx, "current")
	if err != nil {
		return err
	}
	previous, err := stateValue(tx, "previous")
	if err != nil {
		return err
	}
	if current != "" {
		if err := verifySQLiteArtifact(tx, current); err != nil {
			return err
		}
		previous = current
		if err := putState(tx, "previous", previous); err != nil {
			return err
		}
	}
	if err := putState(tx, "current", digest); err != nil {
		return err
	}
	if err := verifySQLitePointers(tx, digest, previous); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	var stored []byte
	if err := db.QueryRow(`SELECT content FROM artifacts WHERE digest = ?`, digest).Scan(&stored); err != nil {
		return err
	}
	if digestBytes(stored) != digest {
		return fmt.Errorf("stored sqlite artifact digest does not match")
	}
	receipt.ArtifactDigest = digest
	receipt.PreviousDigest = previous
	receipt.Installed = true
	receipt.DatabasePath = filepath.Join(root, "artifacts.db")
	return nil
}

// verifySQLiteArtifact checks retained bytes inside the state-change transaction.
func verifySQLiteArtifact(tx *sql.Tx, digest string) error {
	var content []byte
	if err := tx.QueryRow(`SELECT content FROM artifacts WHERE digest = ?`, digest).Scan(&content); err != nil {
		return fmt.Errorf("sqlite artifact is unreadable: %w", err)
	}
	if digestBytes(content) != digest {
		return fmt.Errorf("stored sqlite artifact digest does not match")
	}
	return nil
}

// verifySQLitePointers checks both final pointers and their bytes before commit.
func verifySQLitePointers(tx *sql.Tx, current, previous string) error {
	for _, state := range []struct{ key, want string }{{"current", current}, {"previous", previous}} {
		got, err := stateValue(tx, state.key)
		if err != nil {
			return err
		}
		if got != state.want {
			return fmt.Errorf("sqlite %s pointer does not match planned digest", state.key)
		}
		if got != "" {
			if err := verifySQLiteArtifact(tx, got); err != nil {
				return fmt.Errorf("sqlite %s artifact verification failed: %w", state.key, err)
			}
		}
	}
	return nil
}

func rollbackSQLite(ctx context.Context, root string) (RollbackReceipt, error) {
	receipt := RollbackReceipt{
		Schema: RollbackSchema, GrantsAuthority: false, ProvesRealTarget: false,
	}
	db, err := openSQLite(root)
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	defer tx.Rollback()
	current, err := stateValue(tx, "current")
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	previous, err := stateValue(tx, "previous")
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if current == "" || previous == "" || current == previous {
		receipt.Reason = "current and previous sqlite artifacts are required"
		return receipt, nil
	}
	if err := verifySQLiteArtifact(tx, current); err != nil {
		receipt.Reason = "current sqlite artifact is invalid: " + err.Error()
		return receipt, nil
	}
	var content []byte
	if err := tx.QueryRow(`SELECT content FROM artifacts WHERE digest = ?`, previous).Scan(&content); err != nil {
		receipt.Reason = "previous sqlite artifact is unavailable: " + err.Error()
		return receipt, nil
	}
	if digestBytes(content) != previous {
		receipt.Reason = "previous sqlite artifact digest does not match"
		return receipt, nil
	}
	if err := putState(tx, "current", previous); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if err := putState(tx, "previous", current); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if err := verifySQLitePointers(tx, previous, current); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if err := tx.Commit(); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	receipt.ArtifactDigest = previous
	receipt.PreviousDigest = current
	receipt.CurrentSlot = ""
	receipt.PreviousSlot = ""
	receipt.Passed = true
	return receipt, nil
}

func rollbackToSQLite(ctx context.Context, root, sourceDigest string) (RollbackReceipt, error) {
	receipt := RollbackReceipt{
		Schema: RollbackSchema, RolledBackTo: sourceDigest,
		GrantsAuthority: false, ProvesRealTarget: false,
	}
	if !validDigest(sourceDigest) {
		receipt.Reason = "source digest must be sha256:<64 hex>"
		return receipt, nil
	}
	db, err := openSQLite(root)
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	defer tx.Rollback()
	var content []byte
	if err := tx.QueryRow(`SELECT content FROM artifacts WHERE digest = ?`, sourceDigest).Scan(&content); err != nil {
		receipt.Reason = "sqlite history artifact is unavailable: " + err.Error()
		return receipt, nil
	}
	if digestBytes(content) != sourceDigest {
		receipt.Reason = "sqlite history artifact digest does not match"
		return receipt, nil
	}
	current, err := stateValue(tx, "current")
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	previous, err := stateValue(tx, "previous")
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if current != "" {
		if err := verifySQLiteArtifact(tx, current); err != nil {
			receipt.Reason = "current sqlite artifact is invalid: " + err.Error()
			return receipt, nil
		}
		previous = current
		if err := putState(tx, "previous", previous); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
	}
	if err := putState(tx, "current", sourceDigest); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if err := verifySQLitePointers(tx, sourceDigest, previous); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if err := tx.Commit(); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	receipt.ArtifactDigest = sourceDigest
	receipt.PreviousDigest = previous
	receipt.Passed = true
	return receipt, nil
}
