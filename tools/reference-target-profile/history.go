package targetprofile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	HistorySchema      = "tei.reference.target-profile-history/v0"
	HistoryPruneSchema = "tei.reference.target-profile-history-prune/v0"
)

type HistoryEntry struct {
	Digest   string    `json:"digest"`
	Path     string    `json:"path"`
	Bytes    int64     `json:"bytes"`
	Modified time.Time `json:"modified"`
	Verified bool      `json:"verified"`
	Reason   string    `json:"reason,omitempty"`
}

type HistoryReceipt struct {
	Schema  string         `json:"schema"`
	Root    string         `json:"root"`
	Entries []HistoryEntry `json:"entries"`
	Passed  bool           `json:"passed"`
	Reason  string         `json:"reason,omitempty"`
}

type HistoryPruneReceipt struct {
	Schema      string         `json:"schema"`
	Root        string         `json:"root"`
	KeepLast    int            `json:"keepLast"`
	DryRun      bool           `json:"dryRun"`
	Kept        []HistoryEntry `json:"kept"`
	Deleted     []HistoryEntry `json:"deleted"`
	WouldDelete []HistoryEntry `json:"wouldDelete,omitempty"`
	Passed      bool           `json:"passed"`
	Reason      string         `json:"reason,omitempty"`
}

// History lists and verifies artifacts retained under history/<digest>.
func History(root string) (HistoryReceipt, error) {
	receipt := HistoryReceipt{Schema: HistorySchema, Root: root}
	entries, err := readHistoryEntries(root)
	if err != nil {
		return receipt, err
	}
	receipt.Entries = entries
	for _, entry := range entries {
		if !entry.Verified {
			receipt.Reason = "one or more history artifacts failed verification"
			return receipt, nil
		}
	}
	receipt.Passed = true
	return receipt, nil
}

// PruneHistory deletes all but the newest keepLast verified history artifacts.
// It refuses to delete when any retained artifact is unreadable or altered.
func PruneHistory(root string, keepLast int, dryRun bool) (HistoryPruneReceipt, error) {
	receipt := HistoryPruneReceipt{
		Schema: HistoryPruneSchema, Root: root, KeepLast: keepLast, DryRun: dryRun,
	}
	if keepLast < 1 {
		receipt.Reason = "keepLast must be at least 1"
		return receipt, nil
	}
	entries, err := readHistoryEntries(root)
	if err != nil {
		return receipt, err
	}
	for _, entry := range entries {
		if !entry.Verified {
			receipt.Reason = "refusing to prune unverified history: " + entry.Path
			return receipt, nil
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Modified.Equal(entries[j].Modified) {
			return entries[i].Digest > entries[j].Digest
		}
		return entries[i].Modified.After(entries[j].Modified)
	})
	if len(entries) <= keepLast {
		receipt.Kept = entries
		receipt.Passed = true
		return receipt, nil
	}
	receipt.Kept = entries[:keepLast]
	receipt.Deleted = entries[keepLast:]
	if dryRun {
		receipt.WouldDelete = receipt.Deleted
		receipt.Deleted = nil
		receipt.Passed = true
		return receipt, nil
	}
	for _, entry := range receipt.Deleted {
		current, err := os.ReadFile(entry.Path)
		if err != nil {
			receipt.Reason = "history artifact is unreadable: " + err.Error()
			return receipt, nil
		}
		if digestBytes(current) != entry.Digest {
			receipt.Reason = "history artifact changed before deletion"
			return receipt, nil
		}
	}
	for _, entry := range receipt.Deleted {
		if err := os.Remove(entry.Path); err != nil {
			receipt.Reason = err.Error()
			return receipt, nil
		}
	}
	receipt.Passed = true
	return receipt, nil
}

func readHistoryEntries(root string) ([]HistoryEntry, error) {
	dir := filepath.Join(root, "history")
	dirEntries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []HistoryEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	entries := make([]HistoryEntry, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			return nil, fmt.Errorf("history contains a directory: %s", dirEntry.Name())
		}
		name := dirEntry.Name()
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		entry := HistoryEntry{Path: path, Bytes: info.Size(), Modified: info.ModTime()}
		if len(name) != 64 {
			entry.Reason = "history file name is not a 64-hex digest"
			entries = append(entries, entry)
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			entry.Reason = "history artifact is unreadable: " + err.Error()
			entries = append(entries, entry)
			continue
		}
		entry.Digest = digestBytes(content)
		if !strings.EqualFold(strings.TrimPrefix(entry.Digest, "sha256:"), name) {
			entry.Reason = "history file name does not match its digest"
			entries = append(entries, entry)
			continue
		}
		entry.Verified = true
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Digest < entries[j].Digest
	})
	return entries, nil
}
