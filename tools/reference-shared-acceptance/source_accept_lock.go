package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

const sourceAcceptLockName = ".tei-accept.lock"

// errSourceAcceptBusy means another cooperating Acceptance (or a stale
// unreviewed lock) already owns the named Workspace. It does not identify or
// authenticate the owning actor.
var errSourceAcceptBusy = errors.New("Workspace Source Acceptance lock is occupied")

// sourceAcceptLock is an intentionally conservative, cross-PROCESS
// cooperative guard. Only callers using this same Shared Acceptance entry
// participate. A direct filesystem editor, Git checkout, plugin, or host that
// ignores the guard is NOT excluded.
//
// The lock lives outside managed Source so acquisition must not mutate the
// model revision. It is create-only and is never auto-stolen by PID/time
// heuristics: after a crash, an operator must confirm no accept is running
// before explicitly removing a stale lock.
type sourceAcceptLock struct {
	files  *os.Root
	handle *os.File
	marker []byte
}

// acquireSourceAcceptLock must execute BEFORE evaluate/Review and remain
// held through mutation, conformance checking, and rollback. The caller pins
// the Workspace with os.OpenRoot before calling this function.
func acquireSourceAcceptLock(files *os.Root) (*sourceAcceptLock, error) {
	if files == nil {
		return nil, errors.New("Acceptance Workspace root is not open")
	}
	handle, err := files.OpenFile(sourceAcceptLockName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, errSourceAcceptBusy
		}
		return nil, fmt.Errorf("create Workspace Acceptance lock: %w", err)
	}
	guard := &sourceAcceptLock{files: files, handle: handle}
	// A random ownership marker distinguishes a replaced lock even if its
	// inode number is immediately recycled by a filesystem. This marker is
	// for release verification, NOT permission or authority to Accept.
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		_ = handle.Close()
		// Leave the uninitialized lock in place rather than risk removing
		// a file another actor replaced after our exclusive create.
		return nil, fmt.Errorf("initialize Acceptance lock: inspect and clear the lock manually: %w", err)
	}
	guard.marker = []byte(fmt.Sprintf("tei.reference.source-accept-lock/v0\npid=%d\nnonce=%s\n", os.Getpid(), hex.EncodeToString(nonce[:])))
	if _, err := handle.Write(guard.marker); err != nil {
		_ = handle.Close()
		return nil, fmt.Errorf("write Acceptance lock identity: inspect and clear the lock manually: %w", err)
	}
	if err := handle.Sync(); err != nil {
		_ = handle.Close()
		return nil, fmt.Errorf("sync Acceptance lock identity: inspect and clear the lock manually: %w", err)
	}
	return guard, nil
}

// release never removes a lock whose inode changed while the owner ran.
// This identity check is best-effort: a malicious non-cooperating writer
// still can race Lstat vs Remove; this is not an OS-managed atomic mutex.
func (guard *sourceAcceptLock) release() error {
	if guard == nil || guard.files == nil || guard.handle == nil {
		return errors.New("no owned Workspace Acceptance lock to release")
	}
	owned, statErr := guard.handle.Stat()
	closeErr := guard.handle.Close()
	guard.handle = nil
	if statErr != nil || closeErr != nil {
		return fmt.Errorf("close owned Acceptance lock safely: %w", errors.Join(statErr, closeErr))
	}
	current, err := guard.files.Lstat(sourceAcceptLockName)
	if err != nil {
		return fmt.Errorf("verify Workspace Acceptance lock before release: %w", err)
	}
	if !current.Mode().IsRegular() || !os.SameFile(owned, current) {
		return errors.New("Workspace Acceptance lock identity changed; refusing to remove another actor's lock")
	}
	opened, err := guard.files.Open(sourceAcceptLockName)
	if err != nil {
		return fmt.Errorf("read owned Acceptance lock before release: %w", err)
	}
	content, readErr := io.ReadAll(io.LimitReader(opened, int64(len(guard.marker)+1)))
	closeReadErr := opened.Close()
	if readErr != nil || closeReadErr != nil {
		return fmt.Errorf("verify owned Acceptance lock content: %w", errors.Join(readErr, closeReadErr))
	}
	if !bytes.Equal(content, guard.marker) {
		return errors.New("Workspace Acceptance lock marker changed; refusing to remove another actor's lock")
	}
	if err := guard.files.Remove(sourceAcceptLockName); err != nil {
		return fmt.Errorf("remove owned Workspace Acceptance lock: %w", err)
	}
	return nil
}
