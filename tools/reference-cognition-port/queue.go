// Bounded preparation queue for cognition requests.
//
// #5808 describes the path a background preparation takes: an event, debounce
// and coalesce, a bounded queue, inference, a currentness re-check, and a
// read-only display. This file is the queue part, deliberately separated from
// the rest, because it is the only piece whose failure mode is *blocking the
// caller*: a queue that admits unbounded work, or that lets two jobs for the
// same subject run at once, turns a background hint into duplicated work and a
// UI stall.
//
// The queue coordinates; it does not execute. It owns no goroutine and reads no
// clock: Submit either admits a job and hands back a handle, or reports that it
// could not, and the caller runs the work and reports completion. A queue that
// ran the work itself would need a goroutine, a cancellation policy, and an
// error path — all of which belong to the Host that owns the scheduler, not to
// a reference primitive.
//
// It is not a scheduler, not a Host, and not a concurrency limit for anything
// else.
package cognitionport

import (
	"fmt"
	"sync"
)

// DefaultQueueDepth bounds how many jobs may run at once. A background hint
// that is worth showing is worth one slot; a deeper queue only delays the
// answer the user is looking at.
const DefaultQueueDepth = 1

// Suppression reasons a job is not admitted.
const (
	// SuppressedDepth: the queue is at its bound.
	SuppressedDepth = "depth"
	// SuppressedKey: a job for the same key is already pending or running.
	SuppressedKey = "key"
	// SuppressedClosed: the queue was closed.
	SuppressedClosed = "closed"
)

// QueueStats reports what the queue did with one submission, so a caller can
// measure suppression instead of guessing at it.
type QueueStats struct {
	// Admitted is true when a job was handed back for the caller to run.
	Admitted bool `json:"admitted"`
	// Suppressed names the reason a job was not admitted: one of the
	// Suppressed* constants. It is empty when Admitted is true.
	Suppressed string `json:"suppressed,omitempty"`
	// Pending is how many jobs were admitted but not yet finished when this
	// submission was decided.
	Pending int `json:"pending"`
	// Depth is the bound this queue enforces.
	Depth int `json:"depth"`
	// Deduplicated counts how many submissions this queue has refused because
	// an identical key was already in flight, since it was created.
	Deduplicated int64 `json:"deduplicated"`
	// DepthExceeded counts how many submissions it has refused because the
	// queue was full, since it was created.
	DepthExceeded int64 `json:"depthExceeded"`
}

// Queue admits bounded, de-duplicated preparation jobs.
//
// The zero value is not usable; NewQueue makes one. A Queue is safe for
// concurrent use, because the callers it models — an editor firing on every
// keystroke, a build finishing — do not serialise among themselves.
type Queue struct {
	mu       sync.Mutex
	depth    int
	active   map[string]struct{}
	dedup    int64
	depthHit int64
	closed   bool
}

// NewQueue returns a queue that admits at most depth concurrent jobs. A depth
// below one is refused rather than silently replaced with a default: a caller
// that asks for zero concurrency has a different intention than a caller that
// did not think about it.
func NewQueue(depth int) (*Queue, error) {
	if depth < 1 {
		return nil, fmt.Errorf("queue depth must be at least 1, got %d", depth)
	}
	return &Queue{depth: depth, active: map[string]struct{}{}}, nil
}

// Job is one admitted unit of work. The caller runs it and then calls Finish
// exactly once; the key is not available again until it does.
type Job struct {
	queue *Queue
	key   string
	once  sync.Once
}

// Finish releases the job's key so a later submission for the same subject can
// be admitted. It is idempotent, because a caller that finishes on a defer path
// and again on an error path must not double-release.
func (j *Job) Finish() {
	j.once.Do(func() {
		j.queue.mu.Lock()
		defer j.queue.mu.Unlock()
		delete(j.queue.active, j.key)
	})
}

// Submit offers one job for admission.
//
// A job is admitted when the queue has room and no job for the same key is
// already active. Coalescing on the key is the point: an edit that changes
// nothing about the subject should not queue a second inference behind the
// first. An empty key is refused rather than treated as a wildcard, because an
// empty key would coalesce every job into one and silently drop work the
// caller asked for.
func (q *Queue) Submit(key string) (*Job, QueueStats) {
	q.mu.Lock()
	defer q.mu.Unlock()
	stats := QueueStats{Pending: len(q.active), Depth: q.depth, Deduplicated: q.dedup, DepthExceeded: q.depthHit}
	switch {
	case q.closed:
		stats.Suppressed = SuppressedClosed
		return nil, stats
	case key == "":
		stats.Suppressed = SuppressedKey
		return nil, stats
	}
	if _, busy := q.active[key]; busy {
		q.dedup++
		stats.Suppressed = SuppressedKey
		stats.Deduplicated = q.dedup
		return nil, stats
	}
	if len(q.active) >= q.depth {
		q.depthHit++
		stats.Suppressed = SuppressedDepth
		stats.DepthExceeded = q.depthHit
		return nil, stats
	}
	q.active[key] = struct{}{}
	stats.Admitted = true
	return &Job{queue: q, key: key}, stats
}

// Close refuses further submissions. Active work is not cancelled: this queue
// has no goroutine to cancel, and a caller that wants cancellation cancels its
// own work.
func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
}
