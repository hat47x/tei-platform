//go:build linux

package runtime

import (
	"context"
	"errors"
)

// ErrLinuxSignerOperationReplay means an operation ID was previously reserved
// for this Host. A caller must reconcile the earlier outcome instead of
// starting a second trust-store mutation.
var ErrLinuxSignerOperationReplay = errors.New("Host signer operation ID was already reserved")

// ErrLinuxSignerOperationReservationUnavailable means the Host did not
// confirm a unique reservation. No trust-store mutation is started in this
// case; the caller must not interpret the reservation failure as success.
var ErrLinuxSignerOperationReservationUnavailable = errors.New("Host signer operation reservation is unavailable")

// LinuxSignerOperationReservation is constructed after Host authorization.
// The key fingerprint binds an allow request's content without including
// the Ed25519 public key bytes; revoke operations use an empty fingerprint.
type LinuxSignerOperationReservation struct {
	HostID       string
	OperationID  string
	Principal    string
	SignerRef    string
	Action       string
	KeySHA256    string
}

// LinuxSignerOperationLedger must atomically and durably reserve the unique
// (HostID, OperationID) pair BEFORE the Host changes signer trust.
//
// Return (true, nil) only after durable confirmation that this ID was
// unreserved. Return (false, nil) for ANY previously reserved ID, including
// a different action, signer, principal or public key. On storage failure,
// return an error: the caller will refuse to mutate trust.
//
// A Host requiring restart-safe replay protection must use a durable ledger
// with unique constraints. An in-memory implementation is test-only and
// cannot establish safety across process restarts or multiple Hosts.
//
// Reservations are deliberately not released on a later audit or operation
// failure: such IDs must be reconciled explicitly, never blindly retried.
type LinuxSignerOperationLedger interface {
	ReserveSignerOperation(ctx context.Context, reservation LinuxSignerOperationReservation) (bool, error)
}
