package targetprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	HistoryPruneApprovalSchema = "tei.reference.target-profile-history-prune-approval/v0"
	historyPruneNamespace      = "tei-history-prune"
)

type HistoryPruneApproval struct {
	Schema   string   `json:"schema"`
	Root     string   `json:"root"`
	Storage  string   `json:"storage"`
	KeepLast int      `json:"keepLast"`
	Deleted  []string `json:"deleted"`
}

func historyPrunePlan(root string, keepLast int) (string, []HistoryEntry, error) {
	if keepLast < 1 {
		return "", nil, fmt.Errorf("keepLast must be at least 1")
	}
	if sqliteRoot(root) {
		entries, err := sqliteHistory(root)
		if err != nil {
			return "", nil, err
		}
		for _, entry := range entries {
			if !entry.Verified {
				return "", nil, fmt.Errorf("refusing to prune unverified history: %s", entry.Path)
			}
		}
		if len(entries) <= keepLast {
			return StorageSQLite, nil, nil
		}
		deleted := entries[keepLast:]
		db, err := openSQLite(root)
		if err != nil {
			return "", nil, err
		}
		defer db.Close()
		tx, err := db.Begin()
		if err != nil {
			return "", nil, err
		}
		defer tx.Rollback()
		current, err := stateValue(tx, "current")
		if err != nil {
			return "", nil, err
		}
		previous, err := stateValue(tx, "previous")
		if err != nil {
			return "", nil, err
		}
		for _, entry := range deleted {
			if entry.Digest == current || entry.Digest == previous {
				return "", nil, fmt.Errorf("refusing to delete the current or previous sqlite artifact")
			}
		}
		return StorageSQLite, deleted, nil
	}
	entries, err := readHistoryEntries(root)
	if err != nil {
		return "", nil, err
	}
	for _, entry := range entries {
		if !entry.Verified {
			return "", nil, fmt.Errorf("refusing to prune unverified history: %s", entry.Path)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Modified.Equal(entries[j].Modified) {
			return entries[i].Digest > entries[j].Digest
		}
		return entries[i].Modified.After(entries[j].Modified)
	})
	if len(entries) <= keepLast {
		return StorageFilesystem, nil, nil
	}
	return StorageFilesystem, entries[keepLast:], nil
}

func BuildHistoryPruneApproval(root string, keepLast int) (HistoryPruneApproval, error) {
	storage, deleted, err := historyPrunePlan(root, keepLast)
	if err != nil {
		return HistoryPruneApproval{}, err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return HistoryPruneApproval{}, err
	}
	approval := HistoryPruneApproval{
		Schema: HistoryPruneApprovalSchema, Root: absRoot, Storage: storage, KeepLast: keepLast,
		Deleted: []string{},
	}
	for _, entry := range deleted {
		approval.Deleted = append(approval.Deleted, entry.Digest)
	}
	return approval, nil
}

func verifyHistoryPruneApproval(ctx context.Context, root string, keepLast int, approvalPath, signaturePath, allowedSigners, identity string) error {
	raw, err := os.ReadFile(approvalPath)
	if err != nil {
		return err
	}
	verify := exec.CommandContext(ctx, "ssh-keygen", "-Y", "verify",
		"-f", allowedSigners, "-I", identity, "-n", historyPruneNamespace, "-s", signaturePath)
	verify.Stdin = strings.NewReader(string(raw))
	if output, err := verify.CombinedOutput(); err != nil {
		return fmt.Errorf("approval signature verification failed: %s", strings.TrimSpace(string(output)))
	}
	var approval HistoryPruneApproval
	if err := json.Unmarshal(raw, &approval); err != nil {
		return fmt.Errorf("approval is invalid: %w", err)
	}
	if approval.Schema != HistoryPruneApprovalSchema {
		return fmt.Errorf("unexpected approval schema %q", approval.Schema)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if approval.Root != absRoot {
		return fmt.Errorf("approval root does not match")
	}
	if approval.KeepLast != keepLast {
		return fmt.Errorf("approval keepLast does not match")
	}
	storage, deleted, err := historyPrunePlan(root, keepLast)
	if err != nil {
		return err
	}
	if approval.Storage != storage {
		return fmt.Errorf("approval storage does not match")
	}
	if len(approval.Deleted) != len(deleted) {
		return fmt.Errorf("approval deleted set does not match")
	}
	for index, entry := range deleted {
		if approval.Deleted[index] != entry.Digest {
			return fmt.Errorf("approval deleted digest does not match")
		}
	}
	return nil
}

func PruneHistoryApproved(ctx context.Context, root string, keepLast int, dryRun bool, approvalPath, signaturePath, allowedSigners, identity string) (HistoryPruneReceipt, error) {
	receipt := HistoryPruneReceipt{Schema: HistoryPruneSchema, Root: root, KeepLast: keepLast, DryRun: dryRun}
	if approvalPath == "" || signaturePath == "" || allowedSigners == "" || identity == "" {
		receipt.Reason = "approval, signature, allowed-signers, and identity are required together"
		return receipt, nil
	}
	if err := verifyHistoryPruneApproval(ctx, root, keepLast, approvalPath, signaturePath, allowedSigners, identity); err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	return PruneHistory(root, keepLast, dryRun)
}
