package effect

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestScopeLIFOFailureAndPanicDoNotSkipCleanup(t *testing.T) {
	scope := NewScope()
	first := errors.New("first failure")
	third := errors.New("third failure")
	var order []int
	for _, step := range []struct {
		index   int
		inverse func() error
	}{
		{1, func() error { return first }},
		{2, func() error { panic("inverse panic") }},
		{3, func() error { return third }},
	} {
		step := step
		if err := scope.Track(func() error {
			order = append(order, step.index)
			return step.inverse()
		}); err != nil {
			t.Fatal(err)
		}
	}
	err := scope.Close()
	if !errors.Is(err, first) || !errors.Is(err, third) ||
		!strings.Contains(err.Error(), "effect inverse panicked: inverse panic") {
		t.Fatalf("cleanup did not aggregate failures and recovered panic: %v", err)
	}
	if !reflect.DeepEqual(order, []int{3, 2, 1}) {
		t.Fatalf("cleanup order drifted: %v", order)
	}
	if got := scope.Close(); got != err {
		t.Fatalf("Close result was not cached: %v != %v", got, err)
	}
	if got := scope.Track(func() error { return nil }); !errors.Is(got, ErrScopeClosed) {
		t.Fatalf("Track succeeded after Close: %v", got)
	}
}

func TestScopeConcurrentCloseRunsInverseOnce(t *testing.T) {
	s := NewScope()
	entered := make(chan struct{})
	release := make(chan struct{})
	var count atomic.Int32
	if err := s.Track(func() error {
		count.Add(1)
		close(entered)
		<-release
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	go func() { results <- s.Close() }()
	<-entered

	started := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(started)
		results <- s.Close()
	}()
	<-started
	if got := s.Track(func() error { return nil }); !errors.Is(got, ErrScopeClosed) {
		t.Fatalf("closed Scope accepted a new effect: %v", got)
	}
	if count.Load() != 1 {
		t.Fatal("inverse was rerun while first cleanup was blocked")
	}
	close(release)
	wg.Wait()
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Fatalf("Close #%d failed: %v", i, err)
		}
	}
	if count.Load() != 1 {
		t.Fatalf("inverse ran %d times", count.Load())
	}
}

func TestScopeInvalidInputsFailClosed(t *testing.T) {
	var nilScope *Scope
	if err := nilScope.Track(func() error { return nil }); err == nil {
		t.Fatal("nil scope registered effect")
	}
	if err := nilScope.Close(); err == nil {
		t.Fatal("nil scope Close returned success")
	}
	if err := NewScope().Track(nil); err == nil {
		t.Fatal("nil cleanup was registered")
	}
	if err := NewScope().Close(); err != nil {
		t.Fatalf("empty scope failed: %v", err)
	}
}
