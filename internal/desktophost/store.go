package desktophost

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const maxStoredDraftMarkerBytes = 1 << 20

var (
	ErrDraftMarkerExists = errors.New("Desktop Draft marker already exists")
	ErrDraftStoreRefused = errors.New("Desktop Draft store refused access")
)

// DraftMarkerStore retains only immutable, non-authoritative Draft metadata in
// a Host-selected private directory. The caller must provision and authorize
// this directory OUTSIDE the authoritative Workspace; it is not derived from
// or selected by candidate data or a WebView-provided path.
//
// os.Root confines accesses to the open directory on native hosts, but it
// does not substitute for process isolation, OS secret storage, or an ACL
// check on Windows. GOOS=js has weaker Root TOCTOU guarantees.
type DraftMarkerStore struct {
	root *os.Root
}

// OpenDraftMarkerStore never creates a directory or silently relaxes its mode.
// Ownership and platform ACL policy remain the embedding Host's concern.
func OpenDraftMarkerStore(privateDir string) (*DraftMarkerStore, error) {
	if !filepath.IsAbs(privateDir) {
		return nil, fmt.Errorf("%w: store path must be absolute", ErrDraftStoreRefused)
	}
	info, err := os.Lstat(privateDir)
	if err != nil {
		return nil, fmt.Errorf("%w: state directory: %v", ErrDraftStoreRefused, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%w: state directory must not be a symlink", ErrDraftStoreRefused)
	}
	root, err := os.OpenRoot(privateDir)
	if err != nil {
		return nil, fmt.Errorf("%w: open state directory: %v", ErrDraftStoreRefused, err)
	}
	check, err := root.Stat(".")
	if err != nil || !check.IsDir() || (runtime.GOOS != "windows" && check.Mode().Perm()&0o077 != 0) {
		_ = root.Close()
		return nil, fmt.Errorf("%w: state directory must be private", ErrDraftStoreRefused)
	}
	return &DraftMarkerStore{root: root}, nil
}

func (s *DraftMarkerStore) Close() error {
	if s == nil || s.root == nil {
		return nil
	}
	return s.root.Close()
}

// draftMarkerKey binds an immutable marker identity to one physical Workspace
// and one candidate reference. This is a locator, NOT a bearer credential.
func draftMarkerKey(marker DraftMarker) string {
	sum := sha256.Sum256([]byte(marker.Baseline.WorkspaceRoot + "\x00" + marker.CandidateRef))
	return hex.EncodeToString(sum[:])
}

func draftMarkerFilename(key string) (string, error) {
	if len(key) != 64 {
		return "", ErrInvalidDraftMarker
	}
	for _, c := range key {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", ErrInvalidDraftMarker
		}
	}
	return "draft-" + key + ".json", nil
}

// Save writes an immutable Draft Marker once. It does not retain candidate
// content, approval, authorization, or external-effect instructions.
//
// Publishing a fully written private temp file via a same-directory hard link
// is atomic and create-only: a second window cannot overwrite the first
// marker for the same Workspace and candidate. On a filesystem without hard
// link support the operation fails closed rather than using an unsafe copy.
// File sync precedes publication; complete crash-durability (including the
// directory entry) is not guaranteed on every OS/filesystem.
func (s *DraftMarkerStore) Save(marker DraftMarker) (string, error) {
	if s == nil || s.root == nil {
		return "", ErrDraftStoreRefused
	}
	raw, err := EncodeDraftMarker(marker)
	if err != nil {
		return "", err
	}
	if len(raw) > maxStoredDraftMarkerBytes {
		return "", fmt.Errorf("%w: metadata exceeds 1 MiB", ErrInvalidDraftMarker)
	}
	key := draftMarkerKey(marker)
	filename, _ := draftMarkerFilename(key)
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("create Draft temporary identity: %w", err)
	}
	temp := ".draft-write-" + hex.EncodeToString(random[:])
	f, err := s.root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("%w: open private Draft temp: %v", ErrDraftStoreRefused, err)
	}
	defer s.root.Remove(temp)
	_, writeErr := f.Write(raw)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		return "", fmt.Errorf("%w: cannot safely finish Draft temp", ErrDraftStoreRefused)
	}
	if err := s.root.Link(temp, filename); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", ErrDraftMarkerExists
		}
		return "", fmt.Errorf("%w: cannot publish immutable Draft marker: %v", ErrDraftStoreRefused, err)
	}
	return key, nil
}

// Recover reads one previously published Marker and delegates the state
// classification to RecoverDraftMarker. No accepted Source is written, and
// no unknown external command is replayed. The Host supplies the authorized
// workspace root and its observation function again on every call.
func (s *DraftMarkerStore) Recover(key, authorizedWorkspaceRoot string, observe func(string) (SourceObservation, error)) (Recovery, error) {
	if s == nil || s.root == nil {
		return Recovery{}, ErrDraftStoreRefused
	}
	filename, err := draftMarkerFilename(key)
	if err != nil {
		return Recovery{}, err
	}
	info, err := s.root.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() {
		return Recovery{}, ErrDraftStoreRefused
	}
	file, err := s.root.Open(filename)
	if err != nil {
		return Recovery{}, ErrDraftStoreRefused
	}
	actual, err := file.Stat()
	if err != nil || !actual.Mode().IsRegular() ||
		(runtime.GOOS != "windows" && actual.Mode().Perm()&0o077 != 0) {
		_ = file.Close()
		return Recovery{}, ErrDraftStoreRefused
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, maxStoredDraftMarkerBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(raw) > maxStoredDraftMarkerBytes {
		return Recovery{}, ErrDraftStoreRefused
	}
	var marker DraftMarker
	// Decode through the existing strict Marker boundary. Do not trust the
	// persisted candidate or Workspace identity to select the read location.
	recovery, err := RecoverDraftMarker(raw, authorizedWorkspaceRoot, observe)
	if err != nil {
		return Recovery{}, err
	}
	marker = recovery.Marker
	if !strings.EqualFold(draftMarkerKey(marker), key) {
		return Recovery{}, ErrInvalidDraftMarker
	}
	return recovery, nil
}
