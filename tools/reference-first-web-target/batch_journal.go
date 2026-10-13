//go:build !tei_repair_only

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// A host progress log, not a transaction receipt or permission to resume.
type batchJournalEvent struct {
	Schema      string            `json:"schema"`
	Sequence    uint64            `json:"sequence"`
	Kind        string            `json:"kind"`
	Planned     []int64           `json:"planned,omitempty"`
	OrderID     int64             `json:"orderId,omitempty"`
	Item        *batchItem        `json:"item,omitempty"`
	Receipt     *batchReceipt     `json:"receipt,omitempty"`
	Composition *compositionTrace `json:"composition,omitempty"`
}

type batchJournal interface{ Append(batchJournalEvent) error }

type fileBatchJournal struct {
	file     *os.File
	sequence uint64
	failed   bool
}

func artifactOutsideModel(root, path string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	path = filepath.Join(parent, filepath.Base(path))
	rel, err := filepath.Rel(root, path)
	if err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return "", fmt.Errorf("artifact destination must be outside the model workspace")
	}
	return path, nil
}

func openBatchJournal(root, path string) (*fileBatchJournal, error) {
	path, err := artifactOutsideModel(root, path)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot exclusively create batch journal")
	}
	return &fileBatchJournal{file: file}, nil
}

func (j *fileBatchJournal) Append(event batchJournalEvent) error {
	if j.failed {
		return fmt.Errorf("batch journal is unusable after a write failure")
	}
	event.Schema = "tei.reference.batch-progress/v1"
	event.Sequence = j.sequence + 1
	raw, err := json.Marshal(event)
	if err != nil {
		j.failed = true
		return fmt.Errorf("cannot encode batch journal")
	}
	raw = append(raw, '\n')
	n, err := j.file.Write(raw)
	if err == nil && n == len(raw) {
		err = j.file.Sync()
	}
	if err != nil || n != len(raw) {
		j.failed = true
		return fmt.Errorf("cannot sync batch journal; inspect database state before retry")
	}
	j.sequence++
	return nil
}

func appendBatchJournal(j batchJournal, event batchJournalEvent) error {
	if j == nil {
		return nil
	}
	return j.Append(event)
}
