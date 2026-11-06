package access

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/core"
	"tei/internal/permission"
)

// Issued is a grant issued for one run. It is not a workspace declaration: it
// exists because some party decided this run may touch this scope, and it is
// recorded so the decision can be read back. Issuer names that party.
type Issued struct {
	RunKey     string          `json:"runKey"`
	Issuer     string          `json:"issuer"`
	Capability string          `json:"capability"`
	Permission permission.Spec `json:"permission"`
	Basis      string          `json:"basis"`
}

// IssueLog appends issued grants.
type IssueLog interface {
	AppendIssue(Issued) error
}

// Issue grants the scope to the capability for the run and records who issued
// it and why. An issuance without an issuer or a basis is refused: a record that
// does not say where the authority came from is not an issuance.
func Issue(log IssueLog, runKey, issuer string, capability core.CanonicalID, spec permission.Spec, basis string) (permission.Spec, error) {
	if log == nil {
		return permission.Spec{}, fmt.Errorf("no issue log is configured, so nothing is issued")
	}
	if issuer == "" {
		return permission.Spec{}, fmt.Errorf("no issuer is named, so nothing is issued")
	}
	if issuer == string(capability) {
		return permission.Spec{}, fmt.Errorf("%s cannot issue its own grant", capability)
	}
	if basis == "" {
		return permission.Spec{}, fmt.Errorf("no basis is given, so nothing is issued")
	}
	if err := spec.Validate(); err != nil {
		return permission.Spec{}, fmt.Errorf("issue %s: %w", spec.Key(), err)
	}
	issued := Issued{RunKey: runKey, Issuer: issuer, Capability: string(capability), Permission: spec, Basis: basis}
	if err := log.AppendIssue(issued); err != nil {
		return permission.Spec{}, err
	}
	return spec, nil
}

// IssueFile stores issued grants as JSON lines.
type IssueFile struct {
	Path string
}

// AppendIssue implements IssueLog.
func (f IssueFile) AppendIssue(record Issued) error {
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		return fmt.Errorf("prepare issue log %s: %w", f.Path, err)
	}
	file, err := os.OpenFile(f.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open issue log %s: %w", f.Path, err)
	}
	defer file.Close()
	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode issued grant: %w", err)
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write issue log %s: %w", f.Path, err)
	}
	return nil
}

// Issues reads the issued grants. A missing file is empty; a damaged line is an
// error and the records before it are returned.
func (f IssueFile) Issues() ([]Issued, error) {
	file, err := os.Open(f.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open issue log %s: %w", f.Path, err)
	}
	defer file.Close()
	var records []Issued
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		text := scanner.Bytes()
		if len(text) == 0 {
			continue
		}
		var record Issued
		if err := json.Unmarshal(text, &record); err != nil {
			return records, fmt.Errorf("issue log %s line %d: %w", f.Path, line, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("read issue log %s: %w", f.Path, err)
	}
	return records, nil
}
