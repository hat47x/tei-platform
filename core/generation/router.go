package generation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"tei/core/effect"
)

var (
	ErrUnbound         = errors.New("no serving provider")
	ErrStaleGeneration = errors.New("generation changed")
	ErrGuardRequired   = errors.New("host guard is required")
)

// Binding is a host-local, immutable service selection. It is not an Authority grant.
type Binding struct {
	ContractID      string
	ContractVersion string
	ProviderID      string
	ArtifactDigest  string
	Generation      uint64
}

func (b Binding) valid() bool {
	if b.ContractID == "" || b.ContractVersion == "" || b.ProviderID == "" || b.Generation == 0 || !strings.HasPrefix(b.ArtifactDigest, "sha256:") {
		return false
	}
	hex := strings.TrimPrefix(b.ArtifactDigest, "sha256:")
	if len(hex) != 64 {
		return false
	}
	for _, c := range hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

type generation struct {
	binding  Binding
	scope    *effect.Scope
	inFlight uint64
	retiring bool
	drained  chan struct{}
}

// Router uses one local mutex as its linearization point for Acquire/Publish/Withdraw.
// Cross-host updates and current execution Authority remain owned by the host.
type Router struct {
	mu        sync.Mutex
	current   *generation
	last      Binding
	lastScope *effect.Scope
}

func NewRouter(first Binding, scope *effect.Scope) (*Router, error) {
	if !first.valid() || scope == nil {
		return nil, errors.New("initial binding and scope are required")
	}
	return &Router{current: &generation{binding: first, scope: scope, drained: make(chan struct{})}, last: first, lastScope: scope}, nil
}

// Acquire pins a generation for the lifetime of one invocation. The host's current
// permission, lease and fencing check MUST be supplied and must not do blocking I/O.
func (r *Router) Acquire(guard func(Binding) error) (*Lease, error) {
	if guard == nil {
		return nil, ErrGuardRequired
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil {
		return nil, ErrUnbound
	}
	if err := guard(r.current.binding); err != nil {
		return nil, err
	}
	g := r.current
	g.inFlight++
	return &Lease{router: r, gen: g, Binding: g.binding}, nil
}

type Lease struct {
	Binding Binding
	router  *Router
	gen     *generation
	once    sync.Once
}

func (l *Lease) Release() {
	if l == nil {
		return
	}
	l.once.Do(func() {
		l.router.mu.Lock()
		l.gen.inFlight--
		if l.gen.inFlight == 0 && l.gen.retiring {
			close(l.gen.drained)
		}
		l.router.mu.Unlock()
	})
}

// Publish performs one host-local compare-and-swap from expected to next.
// The new generation must already be staged and ready. The host guard is called
// at the cutover. It is not replaced by plugin-supplied metadata or readiness.
func (r *Router) Publish(expected uint64, next Binding, scope *effect.Scope, guard func(Binding) error) (*Retired, error) {
	if guard == nil {
		return nil, ErrGuardRequired
	}
	if !next.valid() || scope == nil {
		return nil, errors.New("next binding and scope are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil || r.current.binding.Generation != expected {
		return nil, ErrStaleGeneration
	}
	old := r.current
	if next.Generation <= expected || next.ContractID != old.binding.ContractID || next.ContractVersion != old.binding.ContractVersion || scope == old.scope {
		return nil, errors.New("generation, contract or scope is incompatible")
	}
	if err := guard(next); err != nil {
		return nil, fmt.Errorf("publish guard: %w", err)
	}
	r.current = &generation{binding: next, scope: scope, drained: make(chan struct{})}
	r.last, r.lastScope = next, scope
	return retire(old), nil
}

// Activate restores a withdrawn route using a strictly newer generation. The
// caller must have prepared and verified a distinct new instance. Old in-flight
// requests are still pinned to the withdrawn generation until they release.
func (r *Router) Activate(expected uint64, next Binding, scope *effect.Scope, guard func(Binding) error) error {
	if guard == nil {
		return ErrGuardRequired
	}
	if !next.valid() || scope == nil {
		return errors.New("next binding and scope are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current != nil || r.last.Generation != expected {
		return ErrStaleGeneration
	}
	if next.Generation <= expected || next.ContractID != r.last.ContractID || next.ContractVersion != r.last.ContractVersion || scope == r.lastScope {
		return errors.New("generation, contract or scope is incompatible")
	}
	if err := guard(next); err != nil {
		return fmt.Errorf("activate guard: %w", err)
	}
	r.current = &generation{binding: next, scope: scope, drained: make(chan struct{})}
	r.last, r.lastScope = next, scope
	return nil
}

// Withdraw invalidates all new acquisitions without pretending that in-flight
// operations can be cancelled or rolled back automatically.
func (r *Router) Withdraw(expected uint64) (*Retired, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil || r.current.binding.Generation != expected {
		return nil, ErrStaleGeneration
	}
	old := r.current
	r.current = nil
	return retire(old), nil
}

func retire(g *generation) *Retired {
	g.retiring = true
	if g.inFlight == 0 {
		close(g.drained)
	}
	return &Retired{gen: g}
}

type Retired struct{ gen *generation }

// Drain never disposes an active generation. A timeout preserves its effects
// for explicit recovery/fencing, and is NOT evidence of successful cleanup.
func (r *Retired) Drain(ctx context.Context) error {
	if r == nil || r.gen == nil || ctx == nil {
		return errors.New("retired generation and context required")
	}
	select {
	case <-r.gen.drained:
		return r.gen.scope.Close()
	case <-ctx.Done():
		return ctx.Err()
	}
}
