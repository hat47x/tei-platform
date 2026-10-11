package composition

import (
	"errors"
	"fmt"
	"sync"
)

var ErrScopeClosed = errors.New("effect scope is closed")

// Scope owns transient effects. Durable business effects are not reversible by Scope.
type Scope struct {
	mu     sync.Mutex
	closed bool
	undo   []func() error
	done   chan struct{}
	result error
}

func NewScope() *Scope { return &Scope{done: make(chan struct{})} }

// Track registers the inverse of an effect after the effect has been established.
// Registration after Close is rejected; callers must undo failed registration themselves.
func (s *Scope) Track(inverse func() error) error {
	if s == nil || inverse == nil {
		return errors.New("scope and inverse are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrScopeClosed
	}
	s.undo = append(s.undo, inverse)
	return nil
}

// Close executes every inverse in reverse registration order, even if an inverse fails.
// Concurrent Close calls wait for the same cleanup and receive the same error.
func (s *Scope) Close() error {
	if s == nil {
		return errors.New("scope is required")
	}
	s.mu.Lock()
	if s.closed {
		done := s.done
		s.mu.Unlock()
		<-done
		s.mu.Lock()
		result := s.result
		s.mu.Unlock()
		return result
	}
	s.closed = true
	inverses := s.undo
	s.undo = nil
	s.mu.Unlock()

	var failures []error
	for i := len(inverses) - 1; i >= 0; i-- {
		if err := invokeInverse(inverses[i]); err != nil {
			failures = append(failures, err)
		}
	}
	s.mu.Lock()
	s.result = errors.Join(failures...)
	close(s.done)
	s.mu.Unlock()
	return s.result
}

func invokeInverse(inverse func() error) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("effect inverse panicked: %v", p)
		}
	}()
	return inverse()
}
