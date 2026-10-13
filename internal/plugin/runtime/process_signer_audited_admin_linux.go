//go:build linux

package runtime

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

// ErrLinuxSignerAuditUnavailable means the required Host audit record could
// not be written. A failure to record "prepared" rejects mutation entirely.
var ErrLinuxSignerAuditUnavailable = errors.New("Host signer audit is unavailable")

// ErrLinuxSignerAuditOutcomeUnrecorded means the trust operation may have
// changed state, but its final audit result was not durably acknowledged.
// Callers must not blindly retry; reconciliation is required.
var ErrLinuxSignerAuditOutcomeUnrecorded = errors.New("Host signer change outcome was not audited")

// LinuxSignerAuditEvent records only Host-managed identifiers, trust epochs,
// and public-key fingerprints. It must never include the public key bytes,
// private key material, or plugin-provided request data beyond SignerRef.
type LinuxSignerAuditEvent struct {
	OperationID  string
	HostID       string
	Principal    string
	SignerRef    string
	Action       string // allow | revoke
	Phase        string // denied | prepared | applied | unchanged | failed
	ReasonCode   string
	OldEpoch     uint64
	NewEpoch     uint64
	OldKeySHA256 string
	NewKeySHA256 string
	RecordedAt   time.Time
}

// LinuxSignerAuditSink must acknowledge durable recording before returning
// success. Persistence, tamper resistance, and recovery remain the Host's
// responsibility; an in-memory sink is suitable for tests only.
type LinuxSignerAuditSink interface {
	RecordSignerChange(ctx context.Context, event LinuxSignerAuditEvent) error
}

// LinuxSignerAuditedAdmin composes the separately authorized Host admin with
// a required, Host-selected audit sink. The mutex orders calls through this
// particular instance; direct Trust.Allow/Revoke and other adapters can
// bypass this ordering, so epochs are observations rather than a transaction.
type LinuxSignerAuditedAdmin struct {
	mu     sync.Mutex
	Admin  LinuxSignerAdmin
	Sink   LinuxSignerAuditSink
	HostID string
}

// A typed-nil pointer inside an interface is not equal to nil. Reject such
// dependencies before dispatching to user-provided Host components.
func missingLinuxSignerAdminDependency(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func (a *LinuxSignerAuditedAdmin) preflight(ctx context.Context, principal, operationID string) error {
	if a == nil || ctx == nil || a.Admin.Trust == nil ||
		missingLinuxSignerAdminDependency(a.Admin.Authorizer) ||
		missingLinuxSignerAdminDependency(a.Sink) ||
		a.HostID == "" || principal == "" || operationID == "" ||
		strings.TrimSpace(a.HostID) != a.HostID ||
		strings.TrimSpace(operationID) != operationID ||
		len(a.HostID) > 256 || len(operationID) > 256 {
		return errors.New("Linux signer audit requires Host, operation, principal, authorizer, trust and sink")
	}
	return ctx.Err()
}

func linuxSignerKeyFingerprint(key ed25519.PublicKey) string {
	if len(key) != ed25519.PublicKeySize {
		return ""
	}
	sum := sha256.Sum256(key)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (a *LinuxSignerAuditedAdmin) snapshot(ref string) (uint64, string) {
	public, epoch, err := a.Admin.Trust.resolve(ref)
	if err != nil {
		return 0, ""
	}
	return epoch, linuxSignerKeyFingerprint(public)
}

func (a *LinuxSignerAuditedAdmin) record(ctx context.Context, event LinuxSignerAuditEvent) error {
	event.RecordedAt = time.Now().UTC()
	if err := a.Sink.RecordSignerChange(ctx, event); err != nil {
		// Do not leak sink-provided diagnostics into the plugin or admin API.
		return fmt.Errorf("%w during %s", ErrLinuxSignerAuditUnavailable, event.Phase)
	}
	return nil
}

func (a *LinuxSignerAuditedAdmin) finalize(ctx context.Context, event LinuxSignerAuditEvent, operationErr error) error {
	// Finalize even when the caller's context was canceled after mutation.
	// Host audit implementations should additionally impose their own deadline.
	if err := a.record(context.WithoutCancel(ctx), event); err != nil {
		return errors.Join(operationErr, ErrLinuxSignerAuditOutcomeUnrecorded, err)
	}
	return operationErr
}

func (a *LinuxSignerAuditedAdmin) authorize(ctx context.Context, event LinuxSignerAuditEvent) error {
	if err := a.Admin.authorize(ctx, event.Principal, event.Action, event.SignerRef); err != nil {
		event.Phase = "denied"
		event.ReasonCode = "authorization_denied"
		if auditErr := a.record(context.WithoutCancel(ctx), event); auditErr != nil {
			return errors.Join(err, auditErr)
		}
		return err
	}
	return nil
}

// Allow requires a durable "prepared" audit record before changing trust.
// Failure to record "applied" reports an ambiguous result: mutation might
// have succeeded even though this method returned an error.
func (a *LinuxSignerAuditedAdmin) Allow(ctx context.Context, principal, operationID string, req LinuxSignerAdminRequest) error {
	if err := a.preflight(ctx, principal, operationID); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	event := LinuxSignerAuditEvent{
		OperationID: operationID, HostID: a.HostID, Principal: principal,
		SignerRef: req.SignerRef, Action: "allow",
		NewKeySHA256: linuxSignerKeyFingerprint(req.PublicKey),
	}
	if err := a.authorize(ctx, event); err != nil {
		return err
	}
	if event.NewKeySHA256 == "" {
		event.Phase, event.ReasonCode = "denied", "invalid_public_key"
		return a.finalize(ctx, event, errors.New("invalid Host signer public key"))
	}
	event.OldEpoch, event.OldKeySHA256 = a.snapshot(req.SignerRef)
	event.Phase = "prepared"
	if err := a.record(ctx, event); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		event.Phase, event.ReasonCode = "failed", "context_canceled"
		return a.finalize(ctx, event, err)
	}
	err := a.Admin.Trust.Allow(req.SignerRef, req.PublicKey)
	event.NewEpoch, event.NewKeySHA256 = a.snapshot(req.SignerRef)
	if err != nil {
		event.Phase, event.ReasonCode = "failed", "trust_change_failed"
	} else {
		event.Phase = "applied"
	}
	return a.finalize(ctx, event, err)
}

// Revoke reports changed=true together with an error if revocation happened
// but the final outcome record could not be saved.
func (a *LinuxSignerAuditedAdmin) Revoke(ctx context.Context, principal, operationID, ref string) (bool, error) {
	if err := a.preflight(ctx, principal, operationID); err != nil {
		return false, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	event := LinuxSignerAuditEvent{
		OperationID: operationID, HostID: a.HostID, Principal: principal,
		SignerRef: ref, Action: "revoke",
	}
	if err := a.authorize(ctx, event); err != nil {
		return false, err
	}
	event.OldEpoch, event.OldKeySHA256 = a.snapshot(ref)
	event.Phase = "prepared"
	if err := a.record(ctx, event); err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		event.Phase, event.ReasonCode = "failed", "context_canceled"
		return false, a.finalize(ctx, event, err)
	}
	changed := a.Admin.Trust.Revoke(ref)
	event.NewEpoch, event.NewKeySHA256 = a.snapshot(ref)
	if changed {
		event.Phase = "applied"
	} else {
		event.Phase, event.ReasonCode = "unchanged", "signer_not_found"
	}
	return changed, a.finalize(ctx, event, nil)
}
