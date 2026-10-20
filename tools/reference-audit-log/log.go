// Package auditlog is a Reference append-only audit log. Each entry is chained
// to the previous entry, so modifying an existing entry is detectable. This is
// a foundation an application can use; it is not a tamper-proof external store.
package auditlog

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var genesisHash = "sha256:" + strings.Repeat("0", 64)

type Entry struct {
	Seq      int64  `json:"seq"`
	TenantID string `json:"tenantId"`
	Actor    string `json:"actor,omitempty"`
	Action   string `json:"action"`
	Subject  string `json:"subject,omitempty"`
	At       string `json:"at"`
	PrevHash string `json:"prevHash"`
	Hash     string `json:"hash"`
}

type Log struct {
	path string
	mu   sync.Mutex
}

func Open(path string) (*Log, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("audit log path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	return &Log{path: path}, file.Close()
}

func entryHash(entry Entry) string {
	hash := sha256.New()
	for _, part := range []string{
		entry.PrevHash,
		strconv.FormatInt(entry.Seq, 10),
		entry.TenantID,
		entry.Actor,
		entry.Action,
		entry.Subject,
		entry.At,
	} {
		hash.Write([]byte(part))
		hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func readEntries(path string) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var entries []Entry
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("audit entry is not valid JSON: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (l *Log) Entries() ([]Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return readEntries(l.path)
}

// Verify checks the sequence and hash chain. It does not detect deletion of a
// suffix; callers that need that must pin the latest hash elsewhere.
func (l *Log) Verify() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	entries, err := readEntries(l.path)
	if err != nil {
		return err
	}
	previous := genesisHash
	for i, entry := range entries {
		wantSeq := int64(i + 1)
		if entry.Seq != wantSeq {
			return fmt.Errorf("audit entry %d has sequence %d", i, entry.Seq)
		}
		if entry.PrevHash != previous {
			return fmt.Errorf("audit entry %d has a broken previous hash", i)
		}
		if entry.Hash != entryHash(entry) {
			return fmt.Errorf("audit entry %d has a mismatched hash", i)
		}
		previous = entry.Hash
	}
	return nil
}

// Append adds one entry. It refuses to append to a log whose existing chain is
// already broken.
func (l *Log) Append(tenantID, actor, action, subject string) (Entry, error) {
	tenantID = strings.TrimSpace(tenantID)
	action = strings.TrimSpace(action)
	if tenantID == "" {
		return Entry{}, fmt.Errorf("tenant id is required")
	}
	if action == "" {
		return Entry{}, fmt.Errorf("audit action is required")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	entries, err := readEntries(l.path)
	if err != nil {
		return Entry{}, err
	}
	previous := genesisHash
	seq := int64(1)
	if len(entries) > 0 {
		last := entries[len(entries)-1]
		if last.Hash != entryHash(last) {
			return Entry{}, fmt.Errorf("audit log is tampered; refusing to append")
		}
		previous = last.Hash
		seq = last.Seq + 1
	}
	entry := Entry{
		Seq: seq, TenantID: tenantID, Actor: strings.TrimSpace(actor), Action: action,
		Subject: strings.TrimSpace(subject), At: time.Now().UTC().Format(time.RFC3339Nano),
		PrevHash: previous,
	}
	entry.Hash = entryHash(entry)
	raw, err := json.Marshal(entry)
	if err != nil {
		return Entry{}, err
	}
	file, err := os.OpenFile(l.path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return Entry{}, err
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		file.Close()
		return Entry{}, err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return Entry{}, err
	}
	if err := file.Close(); err != nil {
		return Entry{}, err
	}
	return entry, nil
}

// Head returns the latest entry hash after verifying the chain. An empty log
// returns the genesis hash. Pin the head outside the log to detect suffix
// deletion.
func (l *Log) Head() (string, error) {
	if err := l.Verify(); err != nil {
		return "", err
	}
	entries, err := readEntries(l.path)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return genesisHash, nil
	}
	return entries[len(entries)-1].Hash, nil
}

// VerifyHead verifies the chain and requires the last hash to equal expected.
// This detects deletion of a suffix when the caller has pinned the head.
func (l *Log) VerifyHead(expected string) error {
	if strings.TrimSpace(expected) == "" {
		return fmt.Errorf("expected head is required")
	}
	head, err := l.Head()
	if err != nil {
		return err
	}
	if head != expected {
		return fmt.Errorf("audit head mismatch: expected %s, actual %s", expected, head)
	}
	return nil
}
