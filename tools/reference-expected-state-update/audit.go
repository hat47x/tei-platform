package expectedstateupdate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// AuditSchema identifies one audit log line.
const AuditSchema = "tei.reference.expected-state-audit/v0"

// AuditEntry is one append-only audit line. It records who read or changed the
// observation store, not the values themselves.
type AuditEntry struct {
	Schema string `json:"schema"`
	At     string `json:"at"`
	Action string `json:"action"`
	RunID  string `json:"runId,omitempty"`
	Actor  string `json:"actor,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// AuditLog is an append-only JSON-lines audit log. It does not enforce
// authorization; the caller supplies the actor after its own check.
type AuditLog struct {
	Path string
}

// Append writes one entry and syncs the file. A missing path or action is an
// error, so an audit event is never silently dropped.
func (l AuditLog) Append(entry AuditEntry, now time.Time) error {
	if l.Path == "" {
		return fmt.Errorf("audit: path is required")
	}
	if entry.Action == "" {
		return fmt.Errorf("audit: action is required")
	}
	entry.Schema = AuditSchema
	entry.At = now.UTC().Format(time.RFC3339Nano)
	encoded, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("audit: open: %w", err)
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		_ = file.Close()
		return fmt.Errorf("audit: write: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("audit: sync: %w", err)
	}
	return file.Close()
}

// Read returns the entries in append order. A corrupt line or unknown schema is
// an error: the log is not silently truncated to the valid prefix.
func (l AuditLog) Read() ([]AuditEntry, error) {
	if l.Path == "" {
		return nil, fmt.Errorf("audit: path is required")
	}
	encoded, err := os.ReadFile(l.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("audit: read: %w", err)
	}
	var entries []AuditEntry
	for index, line := range bytes.Split(encoded, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var entry AuditEntry
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&entry); err != nil {
			return nil, fmt.Errorf("audit: line %d is not valid JSON: %w", index+1, err)
		}
		if entry.Schema != AuditSchema {
			return nil, fmt.Errorf("audit: line %d has unexpected schema %q", index+1, entry.Schema)
		}
		if entry.Action == "" || entry.At == "" {
			return nil, fmt.Errorf("audit: line %d is missing action or time", index+1)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
