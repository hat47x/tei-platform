package orderrollouthost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pruntime "tei/internal/plugin/runtime"
)

// PgHandleStore implements pruntime.InvocationHandleStore on PostgreSQL with
// optimistic concurrency on the handle revision. Every saved revision is
// also appended to orhc.handle_event.
type PgHandleStore struct {
	Pool *pgxpool.Pool
}

var _ pruntime.InvocationHandleStore = (*PgHandleStore)(nil)

func (s *PgHandleStore) Load(ctx context.Context, id string) (pruntime.DurableInvocationHandle, bool, error) {
	var raw []byte
	err := s.Pool.QueryRow(ctx, `SELECT handle FROM orhc.invocation_handle WHERE invocation_id=$1`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return pruntime.DurableInvocationHandle{}, false, nil
	}
	if err != nil {
		return pruntime.DurableInvocationHandle{}, false, err
	}
	var h pruntime.DurableInvocationHandle
	if err := json.Unmarshal(raw, &h); err != nil {
		return pruntime.DurableInvocationHandle{}, false, err
	}
	return h, true, nil
}

func (s *PgHandleStore) Save(ctx context.Context, h pruntime.DurableInvocationHandle, expected int) (pruntime.DurableInvocationHandle, error) {
	h.Revision = expected + 1
	if err := h.Validate(); err != nil {
		return pruntime.DurableInvocationHandle{}, err
	}
	raw, err := json.Marshal(h)
	if err != nil {
		return pruntime.DurableInvocationHandle{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return pruntime.DurableInvocationHandle{}, err
	}
	defer tx.Rollback(ctx)
	// Storage-side fencing: a handle is written only by the current
	// supervisor of its lease. ReconcileDurableInvocation compares the
	// caller's token with the token in the handle only, which still holds
	// the old owner's token when the new owner's reconciliation was refused.
	if h.ResourceLeaseID != "" {
		var owner string
		var token int64
		err := tx.QueryRow(ctx, `SELECT owner_id, fencing_token FROM orhc.supervisor_ownership WHERE lease_id=$1 FOR SHARE`, h.ResourceLeaseID).Scan(&owner, &token)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return pruntime.DurableInvocationHandle{}, err
		}
		if owner != h.SupervisorOwnerID || uint64(token) != h.FencingToken {
			return pruntime.DurableInvocationHandle{}, fmt.Errorf("%w: invocation %s handle written by %s token %d; current supervisor %s token %d", ErrFenced, h.InvocationID, h.SupervisorOwnerID, h.FencingToken, owner, token)
		}
	}
	if expected == 0 {
		tag, err := tx.Exec(ctx, `INSERT INTO orhc.invocation_handle VALUES ($1,1,$2) ON CONFLICT DO NOTHING`, h.InvocationID, raw)
		if err != nil {
			return pruntime.DurableInvocationHandle{}, err
		}
		if tag.RowsAffected() == 0 {
			return pruntime.DurableInvocationHandle{}, fmt.Errorf("invocation %s already exists", h.InvocationID)
		}
	} else {
		tag, err := tx.Exec(ctx, `UPDATE orhc.invocation_handle SET revision=$2, handle=$3 WHERE invocation_id=$1 AND revision=$4`, h.InvocationID, h.Revision, raw, expected)
		if err != nil {
			return pruntime.DurableInvocationHandle{}, err
		}
		if tag.RowsAffected() == 0 {
			return pruntime.DurableInvocationHandle{}, fmt.Errorf("invocation %s revision conflict at expected revision %d", h.InvocationID, expected)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO orhc.handle_event(invocation_id,revision,handle) VALUES ($1,$2,$3)`, h.InvocationID, h.Revision, raw); err != nil {
		return pruntime.DurableInvocationHandle{}, err
	}
	return h, tx.Commit(ctx)
}

// JobRuntime is the host adapter (pruntime.ReattachableRuntime) for the
// migration tool's execution registry (tei_atlas.job in the target
// database). The registry is written by the tool process itself; the host
// connects with the credential of its own secret manager.
//
// ObserveInvocation reads the lifecycle. A non-terminal job is "running" only
// if its heartbeat advances within ObserveWindow (measured on the observer's
// monotonic clock); otherwise its state is "unknown". A job that is not in the
// registry is "missing". ReattachInvocation installs the new runtime fencing
// token with a compare-and-set and rejects older tokens.
type JobRuntime struct {
	Pool          *pgxpool.Pool
	ObserveWindow time.Duration

	Observes  atomic.Int64
	Reattachs atomic.Int64
}

var _ pruntime.ReattachableRuntime = (*JobRuntime)(nil)

func lifecycleOf(s string) pruntime.RuntimeLifecycle {
	switch s {
	case "starting":
		return pruntime.RuntimeStarting
	case "running", "applying":
		return pruntime.RuntimeRunning
	case "succeeded":
		return pruntime.RuntimeSucceeded
	case "failed":
		return pruntime.RuntimeFailed
	}
	return pruntime.RuntimeUnknown
}

func (r *JobRuntime) read(ctx context.Context, handle string) (string, int64, bool, error) {
	var lc string
	var seq int64
	err := r.Pool.QueryRow(ctx, `SELECT lifecycle, heartbeat_seq FROM tei_atlas.job WHERE runtime_handle=$1`, handle).Scan(&lc, &seq)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, false, nil
	}
	return lc, seq, err == nil, err
}

func (r *JobRuntime) ObserveInvocation(ctx context.Context, handle string) (pruntime.RuntimeObservation, error) {
	r.Observes.Add(1)
	lc, seq, ok, err := r.read(ctx, handle)
	if err != nil {
		return pruntime.RuntimeObservation{}, fmt.Errorf("execution registry unavailable")
	}
	now := func() string { return time.Now().UTC().Format(time.RFC3339Nano) }
	if !ok {
		return pruntime.RuntimeObservation{Lifecycle: pruntime.RuntimeMissing, ObservedAt: now(), Message: "runtime handle is not in the execution registry"}, nil
	}
	state := lifecycleOf(lc)
	if state == pruntime.RuntimeSucceeded || state == pruntime.RuntimeFailed {
		return pruntime.RuntimeObservation{Lifecycle: state, ObservedAt: now(), Message: "execution registry reports " + lc}, nil
	}
	start := time.Now()
	for time.Since(start) < r.ObserveWindow {
		select {
		case <-ctx.Done():
			return pruntime.RuntimeObservation{}, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
		lc2, seq2, ok2, err := r.read(ctx, handle)
		if err != nil {
			return pruntime.RuntimeObservation{}, fmt.Errorf("execution registry unavailable")
		}
		if !ok2 {
			return pruntime.RuntimeObservation{Lifecycle: pruntime.RuntimeMissing, ObservedAt: now(), Message: "runtime handle left the execution registry"}, nil
		}
		if s2 := lifecycleOf(lc2); s2 == pruntime.RuntimeSucceeded || s2 == pruntime.RuntimeFailed {
			return pruntime.RuntimeObservation{Lifecycle: s2, ObservedAt: now(), Message: "execution registry reports " + lc2}, nil
		}
		if seq2 != seq {
			return pruntime.RuntimeObservation{Lifecycle: state, ObservedAt: now(), Message: fmt.Sprintf("execution heartbeat advanced (%s)", lc2)}, nil
		}
	}
	return pruntime.RuntimeObservation{Lifecycle: pruntime.RuntimeUnknown, ObservedAt: now(),
		Message: fmt.Sprintf("registry says %s but the execution heartbeat did not advance within %s; side effects unknown", lc, r.ObserveWindow)}, nil
}

func (r *JobRuntime) ReattachInvocation(ctx context.Context, handle string, token uint64) (pruntime.RuntimeObservation, error) {
	r.Reattachs.Add(1)
	var lc string
	err := r.Pool.QueryRow(ctx, `UPDATE tei_atlas.job SET runtime_token=$2 WHERE runtime_handle=$1 AND runtime_token <= $2 RETURNING lifecycle`, handle, int64(token)).Scan(&lc)
	if errors.Is(err, pgx.ErrNoRows) {
		var cur int64
		if err := r.Pool.QueryRow(ctx, `SELECT runtime_token FROM tei_atlas.job WHERE runtime_handle=$1`, handle).Scan(&cur); errors.Is(err, pgx.ErrNoRows) {
			return pruntime.RuntimeObservation{Lifecycle: pruntime.RuntimeMissing, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
		}
		return pruntime.RuntimeObservation{}, fmt.Errorf("runtime rejected stale fencing token %d; active=%d", token, cur)
	}
	if err != nil {
		return pruntime.RuntimeObservation{}, fmt.Errorf("execution registry unavailable")
	}
	return pruntime.RuntimeObservation{Lifecycle: lifecycleOf(lc), ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), Message: fmt.Sprintf("reattached with fencing token %d", token)}, nil
}
