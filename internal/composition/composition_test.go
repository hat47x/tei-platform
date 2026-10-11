package composition

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
)

func b(gen uint64, provider string) Binding {
	return Binding{ContractID: "std.test.echo", ContractVersion: "1.0.0", ProviderID: provider, ArtifactDigest: "sha256:example", Generation: gen}
}

func allow(Binding) error { return nil }

func TestScopeLIFOAndErrorAggregation(t *testing.T) {
	s := NewScope()
	var mu sync.Mutex
	order := []int{}
	for i := 1; i <= 3; i++ {
		n := i
		if err := s.Track(func() error {
			mu.Lock()
			order = append(order, n)
			mu.Unlock()
			if n == 2 {
				return errors.New("cleanup failed")
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	err := s.Close()
	if err == nil {
		t.Fatal("missing cleanup failure")
	}
	if !reflect.DeepEqual(order, []int{3, 2, 1}) {
		t.Fatalf("order %v", order)
	}
	if s.Close() == nil {
		t.Fatal("concurrent/idempotent cleanup must preserve failure")
	}
	if !errors.Is(s.Track(func() error { return nil }), ErrScopeClosed) {
		t.Fatal("registration after Close")
	}
}

func TestPublishKeepsOldInvocationUntilDrain(t *testing.T) {
	oldScope, nextScope := NewScope(), NewScope()
	clean := 0
	if err := oldScope.Track(func() error { clean++; return nil }); err != nil {
		t.Fatal(err)
	}
	r, err := NewRouter(b(1, "v1"), oldScope)
	if err != nil {
		t.Fatal(err)
	}
	old, err := r.Acquire(allow)
	if err != nil {
		t.Fatal(err)
	}
	retired, err := r.Publish(1, b(2, "v2"), nextScope, allow)
	if err != nil {
		t.Fatal(err)
	}
	now, err := r.Acquire(allow)
	if err != nil {
		t.Fatal(err)
	}
	if now.Binding.ProviderID != "v2" || old.Binding.ProviderID != "v1" {
		t.Fatal("lease changed generations")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(retired.Drain(ctx), context.Canceled) {
		t.Fatal("in-flight drain must not complete")
	}
	if clean != 0 {
		t.Fatal("effect disposed while v1 in-flight")
	}
	now.Release()
	old.Release()
	old.Release()
	if err := retired.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
	if clean != 1 {
		t.Fatalf("cleanup count %d", clean)
	}
}

func TestRejectStaleOrUnauthorizedPublish(t *testing.T) {
	r, err := NewRouter(b(1, "v1"), NewScope())
	if err != nil {
		t.Fatal(err)
	}
	deny := errors.New("denied")
	if _, err = r.Publish(1, b(2, "v2"), NewScope(), func(Binding) error { return deny }); !errors.Is(err, deny) {
		t.Fatalf("authorization: %v", err)
	}
	if _, err = r.Publish(99, b(2, "v2"), NewScope(), allow); !errors.Is(err, ErrStaleGeneration) {
		t.Fatal("missing stale generation rejection")
	}
	l, err := r.Acquire(allow)
	if err != nil {
		t.Fatal(err)
	}
	if l.Binding.ProviderID != "v1" {
		t.Fatal("old generation overwritten after failed publish")
	}
	l.Release()
}

func TestHostAuthorityAndWithdrawal(t *testing.T) {
	r, err := NewRouter(b(1, "v1"), NewScope())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Acquire(nil); !errors.Is(err, ErrGuardRequired) {
		t.Fatal("no host gate")
	}
	deny := errors.New("lease revoked")
	if _, err = r.Acquire(func(Binding) error { return deny }); !errors.Is(err, deny) {
		t.Fatal("stale Authority accepted")
	}
	l, err := r.Acquire(allow)
	if err != nil {
		t.Fatal(err)
	}
	retired, err := r.Withdraw(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Acquire(allow); !errors.Is(err, ErrUnbound) {
		t.Fatal("acquired withdrawn provider")
	}
	l.Release()
	if err := retired.Drain(context.Background()); err != nil {
		t.Fatal(err)
	}
}
