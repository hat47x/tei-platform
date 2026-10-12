package composition

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrCoordinatorInvalid = errors.New("invalid composition coordinator operation")
	ErrReapBusy           = errors.New("retirement reaper already running")
)

// Lifecycle describes the initialization and readiness checks of one new
// runtime generation. Prepare must register all acquired transient resources
// in the supplied scope, so an unsuccessful stage can safely release them.
// Neither function is permitted to perform irreversible business operations.
type Lifecycle struct {
	Prepare func(context.Context, *Scope) error
	Ready   func(context.Context, Binding) error
}

// Coordinator serializes host-local lifecycle changes while leaving requests
// on the Router independently routable. A retained retiring generation remains
// the coordinator's responsibility until Reap observes its drain and cleanup.
// It is not a cluster-wide scheduler and does not itself verify artifact bytes.
type Coordinator struct {
	mu      sync.Mutex
	router  *Router
	guard   func(Binding) error
	retired []*Retired
	reaping bool
}

func NewCoordinator(router *Router, guard func(Binding) error) (*Coordinator, error) {
	if router == nil || guard == nil {
		return nil, ErrCoordinatorInvalid
	}
	return &Coordinator{router: router, guard: guard}, nil
}

// Acquire always applies the host-owned permission/lease/fencing guard.
func (c *Coordinator) Acquire() (*Lease, error) {
	if c == nil {
		return nil, ErrCoordinatorInvalid
	}
	return c.router.Acquire(c.guard)
}

// Replace stages and validates a new generation before publishing it. Failure
// before the cutover leaves the previous provider serving. The old generation
// is queued for Drain, never torn down while it has in-flight calls.
func (c *Coordinator) Replace(ctx context.Context, expected uint64, next Binding, lifecycle Lifecycle) error {
	return c.stage(ctx, expected, next, lifecycle, false)
}

// Restore starts a fresh generation after the previous provider was withdrawn.
// This is explicitly host-authorized, never an automatic registry restart.
func (c *Coordinator) Restore(ctx context.Context, expected uint64, next Binding, lifecycle Lifecycle) error {
	return c.stage(ctx, expected, next, lifecycle, true)
}

func (c *Coordinator) stage(ctx context.Context, expected uint64, next Binding, lifecycle Lifecycle, restore bool) error {
	if c == nil || ctx == nil || lifecycle.Prepare == nil || lifecycle.Ready == nil || !next.valid() {
		return ErrCoordinatorInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	scope := NewScope()
	var err error
	if err = callPrepare(ctx, lifecycle.Prepare, scope); err == nil {
		err = callReady(ctx, lifecycle.Ready, next)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		// Never hold the coordination lock while running plugin Prepare/Ready.
		// Withdrawal must remain possible during a hung or slow stage.
		c.mu.Lock()
		if err = ctx.Err(); err == nil {
			if restore {
				err = c.router.Activate(expected, next, scope, c.guard)
			} else {
				var old *Retired
				old, err = c.router.Publish(expected, next, scope, c.guard)
				if err == nil {
					c.retired = append(c.retired, old)
				}
			}
		}
		c.mu.Unlock()
	}
	if err != nil {
		return errors.Join(err, scope.Close())
	}
	return nil
}

// Withdraw immediately prevents new invocations and queues the old instance
// for draining. Existing leases remain pinned to the withdrawn generation.
func (c *Coordinator) Withdraw(expected uint64) error {
	if c == nil {
		return ErrCoordinatorInvalid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	old, err := c.router.Withdraw(expected)
	if err != nil {
		return err
	}
	c.retired = append(c.retired, old)
	return nil
}

// Reap drains and cleans up retired generations in order. The coordinator
// lock is released while waiting for in-flight work or running disposers, so
// emergency Withdraw and later replacements are not blocked by cleanup.
// Only one Reap may run at a time; cancellation retains unfinished retirement.
func (c *Coordinator) Reap(ctx context.Context) error {
	if c == nil || ctx == nil {
		return ErrCoordinatorInvalid
	}
	c.mu.Lock()
	if c.reaping {
		c.mu.Unlock()
		return ErrReapBusy
	}
	c.reaping = true
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.reaping = false
		c.mu.Unlock()
	}()
	var failures []error
	for {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		c.mu.Lock()
		if len(c.retired) == 0 {
			c.mu.Unlock()
			return errors.Join(failures...)
		}
		old := c.retired[0]
		c.mu.Unlock()
		err := old.Drain(ctx)
		if ctx.Err() != nil {
			return errors.Join(append(failures, ctx.Err())...)
		}
		c.mu.Lock()
		// Completed cleanup (including a failed inverse) is not retried.
		c.retired[0] = nil
		c.retired = c.retired[1:]
		c.mu.Unlock()
		if err != nil {
			failures = append(failures, err)
		}
	}
}

func (c *Coordinator) PendingRetirements() int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.retired)
}

func callPrepare(ctx context.Context, f func(context.Context, *Scope) error, s *Scope) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("prepare panicked: %v", p)
		}
	}()
	return f(ctx, s)
}

func callReady(ctx context.Context, f func(context.Context, Binding) error, b Binding) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("readiness panicked: %v", p)
		}
	}()
	return f(ctx, b)
}
