package targetprofile

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"time"
)

func sqliteHistory(root string) ([]HistoryEntry, error) {
	db, err := openSQLite(root)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT digest, content, created_at FROM artifacts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []HistoryEntry{}
	dbPath := filepath.Join(root, "artifacts.db")
	for rows.Next() {
		var digest string
		var content []byte
		var createdAt string
		if err := rows.Scan(&digest, &content, &createdAt); err != nil {
			return nil, err
		}
		entry := HistoryEntry{
			Digest: digest, Path: dbPath + "#" + digest, Bytes: int64(len(content)),
		}
		modified, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			entry.Reason = "created_at is invalid: " + err.Error()
			entries = append(entries, entry)
			continue
		}
		entry.Modified = modified
		if digestBytes(content) != digest {
			entry.Reason = "sqlite artifact digest does not match"
			entries = append(entries, entry)
			continue
		}
		entry.Verified = true
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Modified.Equal(entries[j].Modified) {
			return entries[i].Digest > entries[j].Digest
		}
		return entries[i].Modified.After(entries[j].Modified)
	})
	return entries, nil
}

func pruneSQLiteHistory(root string, keepLast int, dryRun bool) (HistoryPruneReceipt, error) {
	receipt := HistoryPruneReceipt{
		Schema: HistoryPruneSchema, Root: root, KeepLast: keepLast, DryRun: dryRun,
	}
	if !dryRun {
		if err := verifyHistoryAudit(root); err != nil {
			receipt.Reason = "existing history audit is invalid: " + err.Error()
			return receipt, nil
		}
	}
	entries, err := sqliteHistory(root)
	if err != nil {
		return receipt, err
	}
	for _, entry := range entries {
		if !entry.Verified {
			receipt.Reason = "refusing to prune unverified history: " + entry.Path
			return receipt, nil
		}
	}
	if len(entries) <= keepLast {
		receipt.Kept = entries
		receipt.Passed = true
		return receipt, nil
	}
	receipt.Kept = entries[:keepLast]
	toDelete := entries[keepLast:]

	db, err := openSQLite(root)
	if err != nil {
		return receipt, err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return receipt, err
	}
	defer tx.Rollback()
	current, err := stateValue(tx, "current")
	if err != nil {
		return receipt, err
	}
	previous, err := stateValue(tx, "previous")
	if err != nil {
		return receipt, err
	}
	for _, entry := range toDelete {
		if entry.Digest == current || entry.Digest == previous {
			receipt.Reason = "refusing to delete the current or previous sqlite artifact"
			return receipt, nil
		}
	}
	if dryRun {
		receipt.WouldDelete = toDelete
		receipt.Passed = true
		return receipt, nil
	}
	for _, entry := range toDelete {
		if _, err := tx.Exec(`DELETE FROM artifacts WHERE digest = ?`, entry.Digest); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
	}
	if err := verifySQLitePrune(tx, receipt.Kept, current, previous); err != nil {
		receipt.Reason = "sqlite prune verification failed: " + err.Error()
		return receipt, nil
	}
	if err := tx.Commit(); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	receipt.Deleted = toDelete
	auditPath, err := appendHistoryAudit(root, StorageSQLite, keepLast, receipt.Deleted)
	if err != nil {
		receipt.Reason = "sqlite history pruned but audit append failed: " + err.Error()
		return receipt, nil
	}
	receipt.AuditPath = auditPath
	receipt.Passed = true
	return receipt, nil
}

// verifySQLitePrune reads the actual transaction result, including trigger effects.
func verifySQLitePrune(tx *sql.Tx, kept []HistoryEntry, current, previous string) error {
	expected := make(map[string]HistoryEntry, len(kept))
	for _, entry := range kept {
		expected[entry.Digest] = entry
	}
	rows, err := tx.Query(`SELECT digest, content, created_at FROM artifacts`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var digest, createdAt string
		var content []byte
		if err := rows.Scan(&digest, &content, &createdAt); err != nil {
			return err
		}
		entry, ok := expected[digest]
		if !ok {
			return fmt.Errorf("artifact set does not match the prune plan")
		}
		modified, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil || !modified.Equal(entry.Modified) || digestBytes(content) != entry.Digest {
			return fmt.Errorf("retained artifact bytes or timestamp changed")
		}
		delete(expected, digest)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(expected) != 0 {
		return fmt.Errorf("retained artifact is missing")
	}
	for _, state := range []struct{ key, want string }{{"current", current}, {"previous", previous}} {
		got, err := stateValue(tx, state.key)
		if err != nil {
			return err
		}
		if got != state.want {
			return fmt.Errorf("sqlite %s changed during prune", state.key)
		}
	}
	return nil
}
