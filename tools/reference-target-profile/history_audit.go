package targetprofile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const HistoryAuditSchema = "tei.reference.target-profile-history-prune-audit/v0"

type historyAuditRecord struct {
	Schema         string   `json:"schema"`
	Operation      string   `json:"operation"`
	RecordedAt     string   `json:"recordedAt"`
	Storage        string   `json:"storage"`
	KeepLast       int      `json:"keepLast"`
	Deleted        []string `json:"deleted"`
	PreviousDigest string   `json:"previousDigest,omitempty"`
	RecordDigest   string   `json:"recordDigest"`
}

type HistoryAuditReceipt struct {
	Schema     string               `json:"schema"`
	Root       string               `json:"root"`
	Count      int                  `json:"count"`
	HeadDigest string               `json:"headDigest,omitempty"`
	Records    []historyAuditRecord `json:"records"`
	Passed     bool                 `json:"passed"`
	Reason     string               `json:"reason,omitempty"`
}

const HistoryAuditReceiptSchema = "tei.reference.target-profile-history-audit-receipt/v0"

func HistoryAudit(root string) (HistoryAuditReceipt, error) {
	receipt := HistoryAuditReceipt{Schema: HistoryAuditReceiptSchema, Root: root, Records: []historyAuditRecord{}}
	records, err := readHistoryAudit(root)
	if err != nil {
		receipt.Reason = err.Error()
		return receipt, nil
	}
	receipt.Records = records
	receipt.Count = len(records)
	if len(records) > 0 {
		receipt.HeadDigest = records[len(records)-1].RecordDigest
	}
	receipt.Passed = true
	return receipt, nil
}

func historyAuditPath(root string) string {
	return filepath.Join(root, "history-prune-audit.jsonl")
}

func historyAuditDigest(record historyAuditRecord) (string, error) {
	record.RecordDigest = ""
	encoded, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	return digestBytes(encoded), nil
}

func readHistoryAudit(root string) ([]historyAuditRecord, error) {
	raw, err := os.ReadFile(historyAuditPath(root))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	records := []historyAuditRecord{}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var record historyAuditRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("invalid history audit record: %w", err)
		}
		if record.Schema != HistoryAuditSchema {
			return nil, fmt.Errorf("unexpected history audit schema %q", record.Schema)
		}
		expected, err := historyAuditDigest(record)
		if err != nil {
			return nil, err
		}
		if record.RecordDigest != expected {
			return nil, fmt.Errorf("history audit record digest does not match")
		}
		if len(records) > 0 && record.PreviousDigest != records[len(records)-1].RecordDigest {
			return nil, fmt.Errorf("history audit previous digest does not match")
		}
		if len(records) == 0 && record.PreviousDigest != "" {
			return nil, fmt.Errorf("first history audit record has a previous digest")
		}
		records = append(records, record)
	}
	return records, nil
}

func verifyHistoryAudit(root string) error {
	_, err := readHistoryAudit(root)
	return err
}

func appendHistoryAudit(root, storage string, keepLast int, deleted []HistoryEntry) (string, error) {
	records, err := readHistoryAudit(root)
	if err != nil {
		return "", err
	}
	previous := ""
	if len(records) > 0 {
		previous = records[len(records)-1].RecordDigest
	}
	digests := make([]string, 0, len(deleted))
	for _, entry := range deleted {
		digests = append(digests, entry.Digest)
	}
	record := historyAuditRecord{
		Schema: HistoryAuditSchema, Operation: "prune-history",
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Storage:    storage, KeepLast: keepLast, Deleted: digests,
		PreviousDigest: previous,
	}
	record.RecordDigest, err = historyAuditDigest(record)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	path := historyAuditPath(root)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return "", err
	}
	return path, nil
}
