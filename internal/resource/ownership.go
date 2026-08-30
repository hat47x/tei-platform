package resource

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SupervisorOwnership is the durable/shared coordination record for one
// resource lease supervisor. FencingToken is monotonic per lease and must be
// propagated to authority-bearing broker operations so a stale owner cannot
// mutate a lease after takeover.
type SupervisorOwnership struct {
	LeaseID        string `json:"leaseId"`
	OwnerID        string `json:"ownerId"`
	FencingToken   uint64 `json:"fencingToken"`
	ClaimedAt      string `json:"claimedAt,omitempty"`
	HeartbeatAt    string `json:"heartbeatAt,omitempty"`
	OwnerExpiresAt string `json:"ownerExpiresAt,omitempty"`
}

type OwnershipState string

const (
	OwnershipClaimed   OwnershipState = "ownership-claimed"
	OwnershipHeartbeat OwnershipState = "ownership-heartbeat"
	OwnershipTakeover  OwnershipState = "ownership-takeover"
	OwnershipLost      OwnershipState = "ownership-lost"
	OwnershipReleased  OwnershipState = "ownership-released"
)

type OwnershipRecord struct {
	State          OwnershipState `json:"state"`
	ObservedAt     string         `json:"observedAt,omitempty"`
	OwnerID        string         `json:"ownerId,omitempty"`
	FencingToken   uint64         `json:"fencingToken,omitempty"`
	OwnerExpiresAt string         `json:"ownerExpiresAt,omitempty"`
	Message        string         `json:"message,omitempty"`
}

// SupervisorOwnershipStore is deliberately storage-neutral. Production hosts
// may back it with PostgreSQL, etcd, Consul, or another linearizable/shared
// store. Claim must allocate a strictly increasing fencing token after expiry
// or release; Heartbeat must never resurrect an expired ownership implicitly.
type SupervisorOwnershipStore interface {
	Claim(context.Context, string, string, time.Duration) (SupervisorOwnership, bool, error)
	Heartbeat(context.Context, SupervisorOwnership, time.Duration) (SupervisorOwnership, error)
	Assert(context.Context, SupervisorOwnership) error
	Release(context.Context, SupervisorOwnership) error
}

// MemoryOwnershipStore is a deterministic reference implementation used by the
// M0 probe. Its semantics model a shared durable store, but the implementation
// itself is intentionally in-memory and is not claimed to survive host/process
// restart.
type MemoryOwnershipStore struct {
	Now func() time.Time

	mu        sync.Mutex
	active    map[string]SupervisorOwnership
	lastToken map[string]uint64
}

func (s *MemoryOwnershipStore) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func ownershipExpiry(o SupervisorOwnership) (time.Time, error) {
	if o.OwnerExpiresAt == "" {
		return time.Time{}, fmt.Errorf("lease %s ownership has no expiry", o.LeaseID)
	}
	t, err := time.Parse(time.RFC3339Nano, o.OwnerExpiresAt)
	if err != nil {
		if t2, err2 := time.Parse(time.RFC3339, o.OwnerExpiresAt); err2 == nil {
			return t2, nil
		}
		return time.Time{}, fmt.Errorf("lease %s ownership has invalid expiry %q: %w", o.LeaseID, o.OwnerExpiresAt, err)
	}
	return t, nil
}

func (s *MemoryOwnershipStore) Claim(ctx context.Context, leaseID, ownerID string, ttl time.Duration) (SupervisorOwnership, bool, error) {
	if err := ctx.Err(); err != nil {
		return SupervisorOwnership{}, false, err
	}
	if leaseID == "" || ownerID == "" {
		return SupervisorOwnership{}, false, fmt.Errorf("lease id and supervisor owner id are required")
	}
	if ttl <= 0 {
		return SupervisorOwnership{}, false, fmt.Errorf("supervisor ownership ttl must be positive")
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == nil {
		s.active = map[string]SupervisorOwnership{}
	}
	if s.lastToken == nil {
		s.lastToken = map[string]uint64{}
	}
	previous, exists := s.active[leaseID]
	takeover := false
	if exists {
		exp, err := ownershipExpiry(previous)
		if err != nil {
			return SupervisorOwnership{}, false, err
		}
		if now.Before(exp) {
			if previous.OwnerID != ownerID {
				return SupervisorOwnership{}, false, fmt.Errorf("resource lease %s supervisor is owned by %s token %d until %s", leaseID, previous.OwnerID, previous.FencingToken, previous.OwnerExpiresAt)
			}
			previous.HeartbeatAt = now.Format(time.RFC3339Nano)
			previous.OwnerExpiresAt = now.Add(ttl).Format(time.RFC3339Nano)
			s.active[leaseID] = previous
			return previous, false, nil
		}
		takeover = previous.OwnerID != ownerID
	}
	token := s.lastToken[leaseID] + 1
	if exists && previous.FencingToken >= token {
		token = previous.FencingToken + 1
	}
	s.lastToken[leaseID] = token
	o := SupervisorOwnership{LeaseID: leaseID, OwnerID: ownerID, FencingToken: token, ClaimedAt: now.Format(time.RFC3339Nano), HeartbeatAt: now.Format(time.RFC3339Nano), OwnerExpiresAt: now.Add(ttl).Format(time.RFC3339Nano)}
	s.active[leaseID] = o
	return o, takeover, nil
}

func (s *MemoryOwnershipStore) Heartbeat(ctx context.Context, ownership SupervisorOwnership, ttl time.Duration) (SupervisorOwnership, error) {
	if err := ctx.Err(); err != nil {
		return SupervisorOwnership{}, err
	}
	if ttl <= 0 {
		return SupervisorOwnership{}, fmt.Errorf("supervisor ownership ttl must be positive")
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.active[ownership.LeaseID]
	if !ok {
		return SupervisorOwnership{}, fmt.Errorf("resource lease %s has no active supervisor ownership", ownership.LeaseID)
	}
	if current.OwnerID != ownership.OwnerID || current.FencingToken != ownership.FencingToken {
		return SupervisorOwnership{}, fmt.Errorf("resource lease %s supervisor ownership was fenced: current=%s/%d attempted=%s/%d", ownership.LeaseID, current.OwnerID, current.FencingToken, ownership.OwnerID, ownership.FencingToken)
	}
	exp, err := ownershipExpiry(current)
	if err != nil {
		return SupervisorOwnership{}, err
	}
	if !now.Before(exp) {
		return SupervisorOwnership{}, fmt.Errorf("resource lease %s supervisor ownership expired at %s", ownership.LeaseID, current.OwnerExpiresAt)
	}
	current.HeartbeatAt = now.Format(time.RFC3339Nano)
	current.OwnerExpiresAt = now.Add(ttl).Format(time.RFC3339Nano)
	s.active[ownership.LeaseID] = current
	return current, nil
}

func (s *MemoryOwnershipStore) Assert(ctx context.Context, ownership SupervisorOwnership) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.active[ownership.LeaseID]
	if !ok {
		return fmt.Errorf("resource lease %s has no active supervisor ownership", ownership.LeaseID)
	}
	if current.OwnerID != ownership.OwnerID || current.FencingToken != ownership.FencingToken {
		return fmt.Errorf("resource lease %s supervisor ownership was fenced: current=%s/%d attempted=%s/%d", ownership.LeaseID, current.OwnerID, current.FencingToken, ownership.OwnerID, ownership.FencingToken)
	}
	exp, err := ownershipExpiry(current)
	if err != nil {
		return err
	}
	if !now.Before(exp) {
		return fmt.Errorf("resource lease %s supervisor ownership expired at %s", ownership.LeaseID, current.OwnerExpiresAt)
	}
	return nil
}

func (s *MemoryOwnershipStore) Release(ctx context.Context, ownership SupervisorOwnership) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.active[ownership.LeaseID]
	if !ok {
		return nil
	}
	if current.OwnerID != ownership.OwnerID || current.FencingToken != ownership.FencingToken {
		return fmt.Errorf("resource lease %s supervisor ownership release was fenced", ownership.LeaseID)
	}
	delete(s.active, ownership.LeaseID)
	if s.lastToken == nil {
		s.lastToken = map[string]uint64{}
	}
	if ownership.FencingToken > s.lastToken[ownership.LeaseID] {
		s.lastToken[ownership.LeaseID] = ownership.FencingToken
	}
	return nil
}
