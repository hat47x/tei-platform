//go:build linux

package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// LinuxFileSignerOperationLedger persists reservations in a Host-owned,
// private local directory. It never deletes a reservation on failure.
// The directory and its parent path must be controlled by the Host; use a
// local filesystem that implements O_EXCL and fsync durability semantics.
type LinuxFileSignerOperationLedger struct {
	Directory string
}

// NewLinuxFileSignerOperationLedger requires an existing, non-symlink
// directory owned by the running Host account and inaccessible to others.
func NewLinuxFileSignerOperationLedger(directory string) (*LinuxFileSignerOperationLedger, error) {
	if err := validateLinuxSignerLedgerDirectory(directory); err != nil {
		return nil, err
	}
	return &LinuxFileSignerOperationLedger{Directory: directory}, nil
}

func validateLinuxSignerLedgerDirectory(directory string) error {
	if !filepath.IsAbs(directory) {
		return errors.New("Linux signer ledger requires an absolute Host-controlled directory")
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return fmt.Errorf("Linux signer ledger directory unavailable: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0700 {
		return errors.New("Linux signer ledger directory must be a private non-symlink directory (0700)")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return errors.New("Linux signer ledger directory must be owned by the Host account")
	}
	return nil
}

// ReserveSignerOperation uses one exclusive file creation per Host/operation
// pair. The filename contains only the SHA256 of the pair; its content omits
// key bytes. It syncs both the file and the directory before reporting success.
// If any step fails, the caller must not mutate trust. Partial reservations
// are intentionally retained to prevent uncertain outcomes being replayed.
func (l *LinuxFileSignerOperationLedger) ReserveSignerOperation(ctx context.Context, reservation LinuxSignerOperationReservation) (bool, error) {
	if ctx == nil {
		return false, errors.New("Linux signer ledger requires a context")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if l == nil {
		return false, errors.New("Linux signer ledger is missing")
	}
	if err := validateLinuxSignerLedgerDirectory(l.Directory); err != nil {
		return false, err
	}
	if reservation.HostID == "" || reservation.OperationID == "" || reservation.Principal == "" ||
		reservation.SignerRef == "" || (reservation.Action != "allow" && reservation.Action != "revoke") {
		return false, errors.New("Linux signer ledger requires a bounded Host operation")
	}
	if len(reservation.HostID) > 256 || len(reservation.OperationID) > 256 ||
		len(reservation.Principal) > 512 || len(reservation.SignerRef) > 512 {
		return false, errors.New("Linux signer ledger operation is too long")
	}
	// Separating the fields with a NUL avoids ambiguous pair collisions.
	sum := sha256.Sum256([]byte(reservation.HostID + "\x00" + reservation.OperationID))
	filename := filepath.Join(l.Directory, hex.EncodeToString(sum[:])+".json")
	data, err := json.Marshal(reservation)
	if err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// A failed write/sync/close must NOT remove the reservation.
	// A partially created file still blocks replay and needs reconciliation.
	_, writeErr := file.Write(data)
	if writeErr == nil {
		_, writeErr = io.WriteString(file, "\n")
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return false, writeErr
	}
	if closeErr != nil {
		return false, closeErr
	}
	dir, err := os.Open(l.Directory)
	if err != nil {
		return false, err
	}
	syncErr := dir.Sync()
	closeErr = dir.Close()
	if syncErr != nil {
		return false, syncErr
	}
	if closeErr != nil {
		return false, closeErr
	}
	if err := ctx.Err(); err != nil {
		// Reservation may already be durable. Refuse the trust mutation and
		// keep the reservation for an explicit recovery decision.
		return false, err
	}
	return true, nil
}
