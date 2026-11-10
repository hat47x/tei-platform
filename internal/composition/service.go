package composition

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Service attaches typed executable instances to host-owned generation bindings.
// It neither defines business contracts nor grants artifact or runtime authority.
// Only the host may construct, replace or withdraw a Service.
type Service[I, O any] struct {
	registry  *Registry
	route     *Coordinator
	mu        sync.RWMutex
	instances map[Binding]func(context.Context, I) (O, error)
}

type invocationBindingKey struct{}

// InvocationBinding returns the pinned instance identity for host adapters.
// It is trace data, not an authorization grant. Hosts must recheck current
// permission, expiry and fencing at the concrete effect boundary.
func InvocationBinding(ctx context.Context) (Binding, bool) {
	if ctx == nil {
		return Binding{}, false
	}
	b, ok := ctx.Value(invocationBindingKey{}).(Binding)
	return b, ok
}

// NewService prepares and checks the initial instance before making it callable.
// The guard is a current, nonblocking host check; it must not reenter Registry.
func NewService[I, O any](ctx context.Context, registry *Registry, selected Selection, binding Binding, guard func(Binding) error, invoke func(context.Context, I) (O, error), lifecycle Lifecycle) (service *Service[I, O], err error) {
	if ctx == nil || registry == nil || guard == nil || invoke == nil || lifecycle.Prepare == nil || lifecycle.Ready == nil {
		return nil, ErrCoordinatorInvalid
	}
	s := &Service[I, O]{registry: registry, instances: make(map[Binding]func(context.Context, I) (O, error))}
	scope := NewScope()
	defer func() {
		if p := recover(); p != nil {
			service = nil
			err = errors.Join(fmt.Errorf("service startup panicked: %v", p), scope.Close())
		}
	}()
	fail := func(err error) (*Service[I, O], error) { return nil, errors.Join(err, scope.Close()) }
	if err := s.prepare(ctx, binding, invoke, lifecycle, scope); err != nil {
		return fail(err)
	}
	if err := lifecycle.Ready(ctx, binding); err != nil {
		return fail(err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	route, err := NewSelectedCoordinator(registry, selected, binding, scope, guard)
	if err != nil {
		return fail(err)
	}
	s.route = route
	return s, nil
}

func (s *Service[I, O]) prepare(ctx context.Context, binding Binding, invoke func(context.Context, I) (O, error), lifecycle Lifecycle, scope *Scope) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := lifecycle.Prepare(ctx, scope); err != nil {
		return err
	}
	s.mu.Lock()
	if _, exists := s.instances[binding]; exists {
		s.mu.Unlock()
		return ErrStaleGeneration
	}
	s.instances[binding] = invoke
	s.mu.Unlock()
	undo := func() error {
		s.mu.Lock()
		delete(s.instances, binding)
		s.mu.Unlock()
		return nil
	}
	if err := scope.Track(undo); err != nil {
		_ = undo()
		return err
	}
	return nil
}

// Invoke preserves the pinned implementation through completion, including
// errors and panics. The returned binding contains trace identity, never input.
func (s *Service[I, O]) Invoke(ctx context.Context, input I) (O, Binding, error) {
	var zero O
	if s == nil || ctx == nil {
		return zero, Binding{}, ErrCoordinatorInvalid
	}
	if err := ctx.Err(); err != nil {
		return zero, Binding{}, err
	}
	lease, err := s.route.Acquire()
	if err != nil {
		return zero, Binding{}, err
	}
	defer lease.Release()
	s.mu.RLock()
	invoke := s.instances[lease.Binding]
	s.mu.RUnlock()
	if invoke == nil {
		return zero, lease.Binding, ErrUnbound
	}
	value, err := invoke(context.WithValue(ctx, invocationBindingKey{}, lease.Binding), input)
	return value, lease.Binding, err
}

// Replace stages a distinct instance; only a successful selected cutover makes
// it visible. Old instances remain until every pinned call has finished.
func (s *Service[I, O]) Replace(ctx context.Context, selected Selection, expected uint64, binding Binding, invoke func(context.Context, I) (O, error), lifecycle Lifecycle) error {
	if s == nil || invoke == nil || lifecycle.Prepare == nil || lifecycle.Ready == nil {
		return ErrCoordinatorInvalid
	}
	return s.route.ReplaceSelected(ctx, s.registry, selected, expected, binding, Lifecycle{
		Prepare: func(ctx context.Context, scope *Scope) error {
			return s.prepare(ctx, binding, invoke, lifecycle, scope)
		},
		Ready: lifecycle.Ready,
	})
}

// Restore attaches a fresh typed instance after an explicit withdrawal.
// It uses Coordinator's selected declaration, host guard and increasing
// generation checks. It never retries a business operation.
func (s *Service[I, O]) Restore(ctx context.Context, selected Selection, expected uint64, binding Binding, invoke func(context.Context, I) (O, error), lifecycle Lifecycle) error {
	if s == nil || invoke == nil || lifecycle.Prepare == nil || lifecycle.Ready == nil {
		return ErrCoordinatorInvalid
	}
	return s.route.RestoreSelected(ctx, s.registry, selected, expected, binding, Lifecycle{
		Prepare: func(ctx context.Context, scope *Scope) error {
			return s.prepare(ctx, binding, invoke, lifecycle, scope)
		},
		Ready: lifecycle.Ready,
	})
}

// Withdraw stops new requests. Reap must then observe completion and cleanup;
// business writes are never undone by disposal of an executable instance.
func (s *Service[I, O]) Withdraw(expected uint64) error { return s.route.Withdraw(expected) }
func (s *Service[I, O]) Reap(ctx context.Context) error { return s.route.Reap(ctx) }

// Shutdown stops new calls and reports both withdrawal and retirement failures.
// A cancelled drain retains its retirement for a later explicit Reap; no
// business operation or failed inverse is retried by this method.
func (s *Service[I, O]) Shutdown(ctx context.Context, expected uint64) error {
	if s == nil || ctx == nil {
		return ErrCoordinatorInvalid
	}
	err := s.Withdraw(expected)
	return errors.Join(err, s.Reap(ctx))
}
