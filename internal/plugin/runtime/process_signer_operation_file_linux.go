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
	"strings"
	"sync"
	"syscall"
	"time"
)

// LinuxFileSignerOperationLedger is a Host-local reference implementation
// of the operation reservation contract. Its directory must be provisioned
// under a Host-trusted parent, on a local filesystem with POSIX O_EXCL and
// fsync guarantees. It is not a multi-Host coordination service.
//
// Every reservation is a new, mode-0600 file whose name is derived from the
// Host ID and operation ID. The record is written and fsynced, followed by an
// fsync of the containing directory, before success is acknowledged.
// Failed writes deliberately leave any created entry in place. This may
// require manual reconciliation, but prevents accidental replay.
//
// The ledger never deletes or overwrites reservations.
type LinuxFileSignerOperationLedger struct {
	mu        sync.RWMutex
	directory *os.File
}

// NewLinuxFileSignerOperationLedger opens an existing private directory.
// The Host must provision it with permissions that exclude all other users.
// O_NOFOLLOW rejects a symlink as the directory's final component; all
// ancestors must themselves be trusted by the Host.
func NewLinuxFileSignerOperationLedger(directory string) (*LinuxFileSignerOperationLedger, error) {
	if directory == "" || strings.TrimSpace(directory) != directory || !filepath.IsAbs(directory) {
		return nil, errors.New("Host operation ledger requires an absolute directory")
	}
	fd, err := syscall.Open(directory, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open Host operation ledger directory: %w", err)
	}
	dir := os.NewFile(uintptr(fd), directory)
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil {
		_ = dir.Close()
		return nil, fmt.Errorf("stat Host operation ledger directory: %w", err)
	}
	if st.Mode&syscall.S_IFMT != syscall.S_IFDIR || st.Uid != uint32(os.Geteuid()) || st.Mode&0077 != 0 {
		_ = dir.Close()
		return nil, errors.New("Host operation ledger directory is not private to its owner")
	}
	return &LinuxFileSignerOperationLedger{directory: dir}, nil
}

type linuxFileSignerReservationRecord struct {
	Version    int
	ReservedAt time.Time
	LinuxSignerOperationReservation
}

func validateLinuxFileSignerReservation(r LinuxSignerOperationReservation) error {
	if r.HostID == "" || r.OperationID == "" || r.Principal == "" ||
		len(r.HostID) > 256 || len(r.OperationID) > 256 || len(r.Principal) > 256 ||
		strings.TrimSpace(r.HostID) != r.HostID ||
		strings.TrimSpace(r.OperationID) != r.OperationID ||
		strings.TrimSpace(r.Principal) != r.Principal {
		return errors.New("invalid Host operation ledger identifiers")
	}
	if err := validateLinuxSignatureRef(r.SignerRef); err != nil {
		return err
	}
	switch r.Action {
	case "allow":
		if len(r.KeySHA256) != 71 || !strings.HasPrefix(r.KeySHA256, "sha256:") {
			return errors.New("allow reservation requires a SHA-256 public key fingerprint")
		}
		for _, c := range r.KeySHA256[7:] {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				return errors.New("allow reservation has an invalid SHA-256 fingerprint")
			}
		}
	case "revoke":
		if r.KeySHA256 != "" {
			return errors.New("revoke reservation must not contain a public key")
		}
	default:
		return errors.New("unknown Host operation ledger action")
	}
	return nil
}

// ReserveSignerOperation atomically claims the (HostID, OperationID) pair.
// An existing entry blocks every replay, including altered request content.
// On any I/O uncertainty it returns an error, and never removes the entry.
// The Host's audited admin translates this into a fail-closed result.
func (l *LinuxFileSignerOperationLedger) ReserveSignerOperation(ctx context.Context, r LinuxSignerOperationReservation) (bool, error) {
	if ctx == nil {
		return false, errors.New("Host operation ledger requires a context")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := validateLinuxFileSignerReservation(r); err != nil {
		return false, err
	}
	if l == nil {
		return false, errors.New("Host operation ledger is missing")
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.directory == nil {
		return false, errors.New("Host operation ledger has been closed")
	}

	identity, err := json.Marshal(struct {
		HostID      string
		OperationID string
	}{HostID: r.HostID, OperationID: r.OperationID})
	if err != nil {
		return false, err
	}
	sum := sha256.Sum256(identity)
	name := hex.EncodeToString(sum[:]) + ".json"
	record, err := json.Marshal(linuxFileSignerReservationRecord{
		Version: 1, ReservedAt: time.Now().UTC(),
		LinuxSignerOperationReservation: r,
	})
	if err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	fd, err := syscall.Openat(int(l.directory.Fd()), name,
		syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if errors.Is(err, syscall.EEXIST) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("create Host operation reservation: %w", err)
	}

	f := os.NewFile(uintptr(fd), name)
	if _, err := io.Copy(f, strings.NewReader(string(append(record, '\n')))); err != nil {
		_ = f.Close()
		return false, fmt.Errorf("write Host operation reservation: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return false, fmt.Errorf("sync Host operation reservation: %w", err)
	}
	if err := f.Close(); err != nil {
		return false, fmt.Errorf("close Host operation reservation: %w", err)
	}
	if err := l.directory.Sync(); err != nil {
		return false, fmt.Errorf("sync Host operation reservation directory: %w", err)
	}
	return true, nil
}

// Close releases the directory handle. It waits for outstanding reservations
// but does not remove or invalidate any reservation recorded on disk.
func (l *LinuxFileSignerOperationLedger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.directory == nil {
		return nil
	}
	err := l.directory.Close()
	l.directory = nil
	return err
}
