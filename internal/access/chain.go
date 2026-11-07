package access

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// chainGenesis is the previous hash of the first record. It is a fixed value, so
// an empty log and a log whose first record was replaced are different files.
const chainGenesis = "sha256:" + "0000000000000000000000000000000000000000000000000000000000000000"

// chainVersion names the rule that produced the hashes in a journal.
//
// It is written into every record and into the hash input, so a journal written
// under another rule is rejected with a clear mismatch instead of quietly
// verifying by accident. Changing how a record is hashed is therefore a change
// this journal can see, which is the point of a version marker.
const chainVersion = "tei.access.chain/v2"

// Chained is one record in a chain-linked access journal. The embedded Record is
// the decision itself; the extra fields make a later edit detectable.
//
// The fields are inline in the JSON (embedded structs are flattened), so a reader
// that only knows Record — including Records() and the plain File reader — still
// reads a chained journal without knowing about the chain.
type Chained struct {
	Record
	ChainVersion string `json:"chainVersion"`
	PrevHash     string `json:"prevHash"`
	Hash         string `json:"hash"`
}

// recordHash binds one decision to the one before it.
//
// The decision is hashed as its own JSON encoding rather than as a hand-listed
// set of fields. The hand-listed version left a trap: a field added to Record
// later would not be hashed, so a change to it would pass verification. Hashing
// the encoded form means every field is covered by construction.
func recordHash(previous string, record Record, index int) string {
	encoded, err := json.Marshal(record)
	if err != nil {
		// The Record is a plain struct of strings, a bool, and a string slice;
		// encoding it cannot fail. Falling back to a value that never matches
		// keeps a failed encode from producing a hash that verifies.
		return "sha256:" + hex.EncodeToString([]byte("unencodable:"+err.Error()))
	}
	hash := sha256.New()
	for _, part := range []string{chainVersion, previous, strconv.Itoa(index), string(encoded)} {
		hash.Write([]byte(part))
		hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

// ChainFile keeps access decisions in JSON lines, each linked to the previous
// one. A plain File records decisions; this one also makes a later change to a
// recorded decision visible.
//
// It does not make the file trustworthy on its own. It detects edits, reordering,
// and (when the head is pinned elsewhere) removal of a suffix; it cannot stop
// someone who can rewrite the whole file and recompute every hash. The difference
// between a plain journal and this one is that a partial, careless edit no longer
// passes unnoticed.
type ChainFile struct {
	Path string
}

// Append implements Journal. The record is linked to the last stored record, so a
// record appended out of order does not verify.
func (f ChainFile) Append(record Record) error {
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		return fmt.Errorf("prepare access journal %s: %w", f.Path, err)
	}
	chained, err := f.next(record)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(chained)
	if err != nil {
		return fmt.Errorf("encode access record: %w", err)
	}
	file, err := os.OpenFile(f.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open access journal %s: %w", f.Path, err)
	}
	defer file.Close()
	if _, err := file.Write(append(raw, '\n')); err != nil {
		return fmt.Errorf("write access journal %s: %w", f.Path, err)
	}
	return nil
}

// next builds the record that follows the last stored one. The existing chain is
// verified first: appending to a journal that has already been changed would put
// a valid link after an invalid one, which makes the whole file unreadable as
// evidence while still looking appended.
func (f ChainFile) next(record Record) (Chained, error) {
	records, err := f.chained()
	if err != nil {
		return Chained{}, err
	}
	if err := verifyChain(f.Path, records); err != nil {
		return Chained{}, err
	}
	previous := chainGenesis
	if len(records) > 0 {
		previous = records[len(records)-1].Hash
	}
	index := len(records) + 1
	return Chained{Record: record, ChainVersion: chainVersion, PrevHash: previous, Hash: recordHash(previous, record, index)}, nil
}

// chained reads the file as chained records.
func (f ChainFile) chained() ([]Chained, error) {
	file, err := os.Open(f.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open access journal %s: %w", f.Path, err)
	}
	defer file.Close()
	var records []Chained
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		text := scanner.Bytes()
		if len(text) == 0 {
			continue
		}
		var record Chained
		if err := json.Unmarshal(text, &record); err != nil {
			return records, fmt.Errorf("access journal %s line %d: %w", f.Path, line, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("read access journal %s: %w", f.Path, err)
	}
	return records, nil
}

// Records reads the decisions without the chain, so a caller that only wants the
// decisions does not have to know the journal is chained. A damaged line is an
// error and the records before it are returned.
func (f ChainFile) Records() ([]Record, error) {
	chained, err := f.chained()
	records := make([]Record, 0, len(chained))
	for _, record := range chained {
		records = append(records, record.Record)
	}
	return records, err
}

// Verify checks the chain from the first record to the last. It reports the first
// record that does not match, so a reader can say where the journal was changed.
// Without a pinned head, removal of a suffix is invisible: a shortened chain is
// still a valid chain.
func (f ChainFile) Verify() error {
	return f.VerifyHead("")
}

// VerifyHead verifies the chain and, when expected is not empty, requires the
// last hash to equal it. Pinning the head is what makes removal of a suffix
// detectable; a caller that never pins it can only detect edits, not deletions.
func (f ChainFile) VerifyHead(expected string) error {
	records, err := f.chained()
	if err != nil {
		return err
	}
	if err := verifyChain(f.Path, records); err != nil {
		return err
	}
	head := chainGenesis
	if len(records) > 0 {
		head = records[len(records)-1].Hash
	}
	if expected != "" && head != expected {
		return fmt.Errorf("access journal %s head mismatch: expected %s, actual %s", f.Path, expected, head)
	}
	return nil
}

// verifyChain walks the records and reports the first one that does not match.
//
// The version is checked before the hash: a record written under another rule
// would fail the hash comparison anyway, but the message would say "mismatched
// hash", which reads like tampering. Naming the version difference says what
// actually happened.
func verifyChain(path string, records []Chained) error {
	previous := chainGenesis
	for index, record := range records {
		if record.ChainVersion != chainVersion {
			return fmt.Errorf("access journal %s record %d was written under chain %q, not %q",
				path, index+1, record.ChainVersion, chainVersion)
		}
		if record.PrevHash != previous {
			return fmt.Errorf("access journal %s record %d does not follow the record before it", path, index+1)
		}
		if record.Hash != recordHash(previous, record.Record, index+1) {
			return fmt.Errorf("access journal %s record %d has a mismatched hash", path, index+1)
		}
		previous = record.Hash
	}
	return nil
}

// Head returns the last hash after verifying the chain. An empty journal returns
// the genesis value. The caller is expected to store it somewhere the journal
// cannot rewrite; otherwise a rewritten journal simply reports its own new head.
func (f ChainFile) Head() (string, error) {
	if err := f.Verify(); err != nil {
		return "", err
	}
	records, err := f.chained()
	if err != nil {
		return "", err
	}
	if len(records) == 0 {
		return chainGenesis, nil
	}
	return records[len(records)-1].Hash, nil
}
