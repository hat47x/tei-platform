package runtime

import (
	"testing"
	"time"

	"tei/internal/resource"
)

// The heartbeat interval decides how often a leased capability renews its
// supervisor ownership. A nil controller has no ownership, and a controller
// with no attached ownership has nothing to renew, so both answer zero: a
// non-zero answer there would schedule a heartbeat for a claim that does not
// exist.
func TestOwnershipHeartbeatIntervalIsZeroWithoutAClaim(t *testing.T) {
	var nilController *leaseController
	if got := nilController.OwnershipHeartbeatInterval(); got != 0 {
		t.Fatalf("a nil controller answered %v", got)
	}

	bare := &leaseController{}
	if got := bare.OwnershipHeartbeatInterval(); got != 0 {
		t.Fatalf("a controller with no ownership answered %v", got)
	}
}

// A TTL that cannot produce a positive third answers the 1ms floor rather than
// zero. Zero means "no heartbeat"; a tiny but positive interval means "renew
// this quickly", and the two must not be confused, or a claim with a 1ns TTL
// would never be renewed.
func TestOwnershipHeartbeatIntervalFloorsAtOneMillisecond(t *testing.T) {
	for _, ttl := range []time.Duration{1, 2, time.Nanosecond * 2} {
		c := &leaseController{ownership: &resource.SupervisorOwnership{}, ownershipTTL: ttl}
		if got := c.OwnershipHeartbeatInterval(); got != time.Millisecond {
			t.Fatalf("ttl %v answered %v, want the 1ms floor", ttl, got)
		}
	}
}

// A non-positive TTL is "no lease duration", which is the same as no claim: a
// heartbeat cannot be scheduled against it.
func TestOwnershipHeartbeatIntervalIsZeroForANonPositiveTTL(t *testing.T) {
	for _, ttl := range []time.Duration{0, -time.Second} {
		c := &leaseController{ownership: &resource.SupervisorOwnership{}, ownershipTTL: ttl}
		if got := c.OwnershipHeartbeatInterval(); got != 0 {
			t.Fatalf("ttl %v answered %v, want 0", ttl, got)
		}
	}
}

// A usable TTL renews at a third of it, so three missed beats exhaust the lease.
// The exact divisor is the contract: a longer interval could let the ownership
// expire between heartbeats, and a shorter one multiplies store traffic.
func TestOwnershipHeartbeatIntervalIsAThirdOfTheTTL(t *testing.T) {
	cases := []struct {
		ttl  time.Duration
		want time.Duration
	}{
		{ttl: 3 * time.Second, want: time.Second},
		{ttl: 30 * time.Second, want: 10 * time.Second},
		{ttl: time.Minute, want: 20 * time.Second},
		{ttl: 15 * time.Millisecond, want: 5 * time.Millisecond},
	}
	for _, tc := range cases {
		c := &leaseController{ownership: &resource.SupervisorOwnership{}, ownershipTTL: tc.ttl}
		if got := c.OwnershipHeartbeatInterval(); got != tc.want {
			t.Errorf("ttl %v answered %v, want %v", tc.ttl, got, tc.want)
		}
	}
}

// The answer is read under the same lock the controller uses elsewhere, so a
// caller that asks twice without an attach in between sees the same value.
func TestOwnershipHeartbeatIntervalIsStable(t *testing.T) {
	c := &leaseController{ownership: &resource.SupervisorOwnership{}, ownershipTTL: 9 * time.Second}
	first := c.OwnershipHeartbeatInterval()
	for i := 0; i < 5; i++ {
		if got := c.OwnershipHeartbeatInterval(); got != first {
			t.Fatalf("call %d answered %v, first answered %v", i, got, first)
		}
	}
	if first != 3*time.Second {
		t.Fatalf("answered %v, want 3s", first)
	}
}
