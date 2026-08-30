package runtime

import (
	"context"
	"fmt"
	"testing"
	"time"

	"tei/internal/resource"
)

type heartbeatFailStore struct{}

func (heartbeatFailStore) Claim(context.Context, string, string, time.Duration) (resource.SupervisorOwnership, bool, error) {
	return resource.SupervisorOwnership{}, false, nil
}
func (heartbeatFailStore) Heartbeat(context.Context, resource.SupervisorOwnership, time.Duration) (resource.SupervisorOwnership, error) {
	return resource.SupervisorOwnership{}, fmt.Errorf("shared ownership store unavailable")
}
func (heartbeatFailStore) Assert(context.Context, resource.SupervisorOwnership) error  { return nil }
func (heartbeatFailStore) Release(context.Context, resource.SupervisorOwnership) error { return nil }

func TestOwnershipHeartbeatFailureIsFailClosed(t *testing.T) {
	now := time.Now().UTC()
	c := &leaseController{current: resource.Lease{ID: "lease-1", Revision: 1, FencingToken: 1, ExpiresAt: now.Add(time.Minute).Format(time.RFC3339Nano)}}
	c.attachOwnership(heartbeatFailStore{}, resource.SupervisorOwnership{LeaseID: "lease-1", OwnerID: "host-a", FencingToken: 1, ClaimedAt: now.Format(time.RFC3339Nano), HeartbeatAt: now.Format(time.RFC3339Nano), OwnerExpiresAt: now.Add(30 * time.Millisecond).Format(time.RFC3339Nano)}, 30*time.Millisecond, false)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := runOwnershipHeartbeat(ctx, c); err == nil {
		t.Fatal("expected ownership heartbeat failure")
	}
	if c.FatalError() == nil {
		t.Fatal("ownership heartbeat failure did not mark fail-closed fatal error")
	}
	events := c.OwnershipRecords()
	if len(events) < 2 || events[len(events)-1].State != resource.OwnershipLost {
		t.Fatalf("ownership events=%#v", events)
	}
}

type cancelAwareHeartbeatStore struct {
	started chan struct{}
}

func (s cancelAwareHeartbeatStore) Claim(context.Context, string, string, time.Duration) (resource.SupervisorOwnership, bool, error) {
	return resource.SupervisorOwnership{}, false, nil
}
func (s cancelAwareHeartbeatStore) Heartbeat(ctx context.Context, ownership resource.SupervisorOwnership, _ time.Duration) (resource.SupervisorOwnership, error) {
	select {
	case s.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return resource.SupervisorOwnership{}, ctx.Err()
}
func (cancelAwareHeartbeatStore) Assert(context.Context, resource.SupervisorOwnership) error {
	return nil
}
func (cancelAwareHeartbeatStore) Release(context.Context, resource.SupervisorOwnership) error {
	return nil
}

func TestOwnershipHeartbeatCancellationDuringShutdownIsNotFailure(t *testing.T) {
	now := time.Now().UTC()
	started := make(chan struct{}, 1)
	c := &leaseController{
		current: resource.Lease{ID: "lease-1", Revision: 1, FencingToken: 1, ExpiresAt: now.Add(time.Minute).Format(time.RFC3339Nano)},
	}
	c.attachOwnership(
		cancelAwareHeartbeatStore{started: started},
		resource.SupervisorOwnership{
			LeaseID:        "lease-1",
			OwnerID:        "host-a",
			FencingToken:   1,
			ClaimedAt:      now.Format(time.RFC3339Nano),
			HeartbeatAt:    now.Format(time.RFC3339Nano),
			OwnerExpiresAt: now.Add(30 * time.Millisecond).Format(time.RFC3339Nano),
		},
		30*time.Millisecond,
		false,
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runOwnershipHeartbeat(ctx, c)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("heartbeat did not start")
	}
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("shutdown cancellation became heartbeat failure: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("heartbeat loop did not stop after cancellation")
	}

	if err := c.FatalError(); err != nil {
		t.Fatalf("shutdown cancellation marked fail-closed error: %v", err)
	}
	for _, event := range c.OwnershipRecords() {
		if event.State == resource.OwnershipLost {
			t.Fatalf("shutdown cancellation was recorded as ownership loss: %#v", c.OwnershipRecords())
		}
	}
}
