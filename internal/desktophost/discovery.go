package desktophost

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
)

const maxDraftStoreEntries = 4096

// DiscoverKeys finds immutable Draft Marker locators in a Host-authorized,
// dedicated private store after process restart. No key from the prior process
// is required. Keys are NOT authorization or approval tokens.
//
// The embedding Host must authorize access to the private store before calling
// this method, and must NOT forward the result directly to an untrusted WebView
// or remote service. Recover(key, authorizedWorkspaceRoot, observe) still makes
// a separate Workspace/Source check for each selected key.
//
// Discovery never opens candidate data, Source, or other Workspace paths.
// Temporary files left by an interrupted Save are ignored, never replayed.
func (s *DraftMarkerStore) DiscoverKeys() ([]string, error) {
	if s == nil || s.root == nil {
		return nil, ErrDraftStoreRefused
	}
	dir, err := s.root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("%w: open state inventory: %v", ErrDraftStoreRefused, err)
	}
	defer dir.Close()

	// Bound memory and work when a damaged store contains unexpected entries.
	entries, readErr := dir.ReadDir(maxDraftStoreEntries + 1)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return nil, fmt.Errorf("%w: read state inventory: %v", ErrDraftStoreRefused, readErr)
	}
	if len(entries) > maxDraftStoreEntries {
		return nil, fmt.Errorf("%w: too many state entries", ErrDraftStoreRefused)
	}
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".draft-write-") {
			// A process may have terminated during Save. Temp files are not
			// published Markers: never load, promote, or replay them.
			if len(name) != len(".draft-write-")+32 || !lowercaseHex(name[len(".draft-write-"):]) {
				return nil, fmt.Errorf("%w: unrecognized Draft temporary entry", ErrDraftStoreRefused)
			}
			continue
		}
		if !strings.HasPrefix(name, "draft-") || !strings.HasSuffix(name, ".json") {
			return nil, fmt.Errorf("%w: unrelated file in dedicated Draft store", ErrDraftStoreRefused)
		}
		key := strings.TrimSuffix(strings.TrimPrefix(name, "draft-"), ".json")
		filename, err := draftMarkerFilename(key)
		if err != nil || filename != name {
			return nil, fmt.Errorf("%w: invalid stored Draft key", ErrDraftStoreRefused)
		}
		// Validate metadata but do not trust it to authorize a Workspace.
		// Recover rechecks and opens the file through os.Root on use.
		info, err := s.root.Lstat(filename)
		if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maxStoredDraftMarkerBytes {
			return nil, fmt.Errorf("%w: invalid stored Draft file", ErrDraftStoreRefused)
		}
		if err := privateDraftFileMode(info.Mode()); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

func lowercaseHex(value string) bool {
	if value == "" {
		return false
	}
	for _, c := range value {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// privateDraftFileMode centralizes the Unix permission policy used by
// discovery. Windows ACL checks remain the responsibility of the Host.
func privateDraftFileMode(mode os.FileMode) error {
	// os.Root's permission policy is identical to Recover's: Windows ACLs
	// cannot be inferred from Unix permission bits.
	if runtime.GOOS != "windows" && mode.Perm()&0o077 != 0 {
		return ErrDraftStoreRefused
	}
	return nil
}
