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
	current, err := stateValue(tx, "current")
	if err != nil {
		return err
	}
	if current != "" {
		if err := putState(tx, "previous", current); err != nil {
			return err
		}
	}
	if err := putState(tx, "current", digest); err != nil {
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
	receipt.PreviousDigest = current
	receipt.Installed = true
	receipt.DatabasePath = filepath.Join(root, "artifacts.db")
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
	if err := tx.Commit(); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	receipt.ArtifactDigest = previous
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
	if current != "" {
		if err := putState(tx, "previous", current); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
	}
	if err := putState(tx, "current", sourceDigest); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	if err := tx.Commit(); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	receipt.ArtifactDigest = sourceDigest
	receipt.Passed = true
	return receipt, nil
}
