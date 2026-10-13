package main

import (
	"errors"
	"fmt"
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
	// The metadata is only a diagnostic hint for manual recovery, not a
	// capability token or lease. No PID reuse or wall-clock expiry can grant
	// authority to steal this lock.
	if _, err := fmt.Fprintf(handle, "tei.reference.source-accept-lock/v0\npid=%d\n", os.Getpid()); err != nil {
		return nil, errors.Join(err, guard.release())
	}
	if err := handle.Sync(); err != nil {
		return nil, errors.Join(err, guard.release())
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
	if err := guard.files.Remove(sourceAcceptLockName); err != nil {
		return fmt.Errorf("remove owned Workspace Acceptance lock: %w", err)
	}
	return nil
}
