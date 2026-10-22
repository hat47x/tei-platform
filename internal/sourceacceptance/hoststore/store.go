// Package hoststore is a file-backed reference implementation of the
// sourceacceptance.Reader seam. It stores the Host's Accepted Source decisions
// and answers "which one is CURRENT" on every call by re-reading the file.
//
// It is a storage and state-transition component only. It never decides that a
// Source is accepted: Ingest requires a DecisionVerifier injected by the Host,
// and there is no default verifier. Who may issue a Decision, the signature
// scheme and the revocation policy are Host decisions this package does not
// make. A Store that holds a Decision is NOT evidence of human Acceptance.
//
// The Store is a single-writer design. Writers in one process are serialized
// by a mutex; writes replace the file atomically, so a concurrent reader (for
// example a running Web Target in another process) sees either the old or the
// new state, or fails closed. Concurrent writers in several processes are not
// supported.
package hoststore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
)

const schemaVersion = "tei.host-acceptance-store/v0"

// Decision states. Exactly one Decision per Application can be current.
const (
	StateCurrent    = "current"
	StateRevoked    = "revoked"
	StateSuperseded = "superseded"
)

var (
	// ErrNoCurrentAcceptance is returned when an Application has no current
	// Decision (never accepted, revoked, or the store is unreadable). The Reader
	// path reports it opaquely; sourceacceptance.VerifyCurrent redacts it again.
	ErrNoCurrentAcceptance = errors.New("HOST_STORE_NO_CURRENT_ACCEPTANCE")
	// ErrIngestRefused covers an unverified, malformed or conflicting Decision.
	ErrIngestRefused = errors.New("HOST_STORE_INGEST_REFUSED")
	// ErrVerifierRequired is returned when the Store was opened without a
	// DecisionVerifier. The Store never accepts a Decision on its own.
	ErrVerifierRequired = errors.New("HOST_STORE_VERIFIER_REQUIRED")
)

// Decision is what a human-authorized issuer hands to the Host. Evidence is an
// opaque, issuer-defined blob (for example a signature over Record); the Store
// stores it verbatim and does not interpret it.
type Decision struct {
	Record   sourceacceptance.Record `json:"record"`
	Issuer   string                  `json:"issuer"`
	Evidence json.RawMessage         `json:"evidence,omitempty"`
}

// DecisionVerifier is supplied by the Host. It must return nil only when the
// Decision is authentic and authorized under the Host's issuer policy.
type DecisionVerifier func(context.Context, Decision) error

// Store is a directory of per-Application acceptance files.
type Store struct {
	dir      string
	verifier DecisionVerifier
	mu       sync.Mutex
}

// Open returns a Store rooted at dir. A nil verifier yields a read-only Store:
// ReadCurrentAcceptedSource works, but Ingest returns ErrVerifierRequired.
// Revoke does not need a verifier because it only narrows authority.
func Open(dir string, verifier DecisionVerifier) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("HOST_STORE_DIR_REQUIRED")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("HOST_STORE_OPEN_FAILED")
	}
	return &Store{dir: dir, verifier: verifier}, nil
}

type entry struct {
	Decision Decision `json:"decision"`
	State    string   `json:"state"`
}

type file struct {
	Schema        string  `json:"schema"`
	ApplicationID string  `json:"applicationId"`
	Entries       []entry `json:"entries"`
}

// ReadCurrentAcceptedSource implements sourceacceptance.Reader. The file is
// re-read on every call; nothing is cached, so revocation takes effect on the
// next verification.
func (s *Store) ReadCurrentAcceptedSource(ctx context.Context, applicationID string) (sourceacceptance.Record, error) {
	if err := ctx.Err(); err != nil {
		return sourceacceptance.Record{}, err
	}
	state, err := s.load(applicationID)
	if err != nil {
		return sourceacceptance.Record{}, ErrNoCurrentAcceptance
	}
	for _, e := range state.Entries {
		if e.State == StateCurrent {
			return cloneRecord(e.Decision.Record), nil
		}
	}
	return sourceacceptance.Record{}, ErrNoCurrentAcceptance
}

// Ingest stores a Decision verified by the Host-supplied verifier and makes it
// the Application's current acceptance. A previous current Decision becomes
// superseded in the same atomic write (replace). The Record is checked for
// internal consistency (canonical members, ModelRevision matches members); this
// is not a check against any Workspace.
func (s *Store) Ingest(ctx context.Context, d Decision) error {
	if s.verifier == nil {
		return ErrVerifierRequired
	}
	if err := checkRecord(d.Record); err != nil || strings.TrimSpace(d.Issuer) == "" {
		return ErrIngestRefused
	}
	if err := s.verifier(ctx, d); err != nil {
		return ErrIngestRefused
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load(d.Record.ApplicationID)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return ErrIngestRefused
		}
		state = file{Schema: schemaVersion, ApplicationID: d.Record.ApplicationID}
	}
	for _, e := range state.Entries {
		if e.Decision.Record.DecisionRef == d.Record.DecisionRef {
			// A DecisionRef identifies one decision forever; reusing it for
			// different content would rewrite history.
			return ErrIngestRefused
		}
	}
	for i := range state.Entries {
		if state.Entries[i].State == StateCurrent {
			state.Entries[i].State = StateSuperseded
		}
	}
	d.Record = cloneRecord(d.Record)
	state.Entries = append(state.Entries, entry{Decision: d, State: StateCurrent})
	return s.save(state)
}

// Revoke makes the Application's current Decision (identified by decisionRef)
// non-current. The next ReadCurrentAcceptedSource fails closed.
func (s *Store) Revoke(applicationID, decisionRef string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.load(applicationID)
	if err != nil {
		return ErrNoCurrentAcceptance
	}
	for i := range state.Entries {
		if state.Entries[i].State == StateCurrent && state.Entries[i].Decision.Record.DecisionRef == decisionRef {
			state.Entries[i].State = StateRevoked
			return s.save(state)
		}
	}
	return ErrNoCurrentAcceptance
}

// path hashes the Application ID so no caller-supplied string reaches the
// filesystem as a path component.
func (s *Store) path(applicationID string) string {
	sum := sha256.Sum256([]byte(applicationID))
	return filepath.Join(s.dir, hex.EncodeToString(sum[:])+".json")
}

func (s *Store) load(applicationID string) (file, error) {
	if strings.TrimSpace(applicationID) == "" {
		return file{}, errors.New("application required")
	}
	raw, err := os.ReadFile(s.path(applicationID))
	if err != nil {
		return file{}, err
	}
	var state file
	if err := json.Unmarshal(raw, &state); err != nil {
		return file{}, err
	}
	if state.Schema != schemaVersion || state.ApplicationID != applicationID {
		return file{}, errors.New("store file does not belong to this application")
	}
	current := 0
	for _, e := range state.Entries {
		if e.State == StateCurrent {
			current++
		}
		if e.Decision.Record.ApplicationID != applicationID {
			return file{}, errors.New("entry for another application")
		}
	}
	if current > 1 {
		// Two current Decisions means the file was edited outside this
		// package. Fail closed instead of picking one.
		return file{}, errors.New("more than one current decision")
	}
	return state, nil
}

func (s *Store) save(state file) error {
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	target := s.path(state.ApplicationID)
	tmp, err := os.CreateTemp(s.dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

func checkRecord(r sourceacceptance.Record) error {
	if strings.TrimSpace(r.ApplicationID) == "" || strings.TrimSpace(r.DecisionRef) == "" ||
		strings.TrimSpace(r.AuthorityRevision) == "" || len(r.Artifacts) == 0 {
		return errors.New("incomplete record")
	}
	seen := map[string]bool{}
	for _, a := range r.Artifacts {
		normalized, err := sourceindex.NormalizeWorkspacePath(a.Path)
		if err != nil || normalized != a.Path || seen[a.Path] {
			return errors.New("noncanonical or duplicate member")
		}
		seen[a.Path] = true
	}
	if sourceindex.ModelRevision(sourceindex.Index{Artifacts: r.Artifacts}) != r.ModelRevision {
		return errors.New("ModelRevision does not match members")
	}
	return nil
}

func cloneRecord(r sourceacceptance.Record) sourceacceptance.Record {
	r.Artifacts = append([]sourceindex.Artifact(nil), r.Artifacts...)
	return r
}

var _ sourceacceptance.Reader = (*Store)(nil)
