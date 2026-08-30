package runtime

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"tei/internal/core"
	"tei/internal/plugin"
)

// RuntimeLifecycle is an observation of the external process/container that
// executes one plugin invocation. It is operational state, not deployment
// semantic state.
type RuntimeLifecycle string

const (
	RuntimeStarting  RuntimeLifecycle = "starting"
	RuntimeRunning   RuntimeLifecycle = "running"
	RuntimeSucceeded RuntimeLifecycle = "succeeded"
	RuntimeFailed    RuntimeLifecycle = "failed"
	RuntimeMissing   RuntimeLifecycle = "missing"
	RuntimeUnknown   RuntimeLifecycle = "unknown"
)

func (s RuntimeLifecycle) terminal() bool { return s == RuntimeSucceeded || s == RuntimeFailed }

// DurableInvocationHandle is the minimum non-secret identity needed to
// reconcile an invocation after host restart/takeover. RuntimeHandle MUST be an
// opaque, non-secret handle. Commands, credentials, mount source handles and
// request payloads do not belong here.
type DurableInvocationHandle struct {
	FormatVersion            int              `json:"formatVersion"`
	Revision                 int              `json:"revision"`
	InvocationID             string           `json:"invocationId"`
	RunID                    string           `json:"runId,omitempty"`
	ActionID                 string           `json:"actionId,omitempty"`
	RequestID                string           `json:"requestId"`
	Environment              core.CanonicalID `json:"environment"`
	Plugin                   plugin.Identity  `json:"plugin"`
	CapabilityID             core.CanonicalID `json:"capabilityId"`
	DescriptorFingerprint    string           `json:"descriptorFingerprint"`
	PluginLockFingerprint    string           `json:"pluginLockFingerprint"`
	RuntimePolicyFingerprint string           `json:"runtimePolicyFingerprint"`
	ArtifactDigest           string           `json:"artifactDigest,omitempty"`
	RuntimeKind              string           `json:"runtimeKind"`
	RuntimeHandle            string           `json:"runtimeHandle"`
	ResourceLeaseID          string           `json:"resourceLeaseId,omitempty"`
	SupervisorOwnerID        string           `json:"supervisorOwnerId,omitempty"`
	FencingToken             uint64           `json:"fencingToken,omitempty"`
	RuntimeFencingToken      uint64           `json:"runtimeFencingToken,omitempty"`
	Lifecycle                RuntimeLifecycle `json:"lifecycle"`
	StartedAt                string           `json:"startedAt,omitempty"`
	LastObservedAt           string           `json:"lastObservedAt,omitempty"`
	LastMessage              string           `json:"lastMessage,omitempty"`
}

func (h DurableInvocationHandle) Validate() error {
	if h.FormatVersion != 1 {
		return fmt.Errorf("unsupported durable invocation handle format %d", h.FormatVersion)
	}
	if h.InvocationID == "" || h.RequestID == "" || h.Environment == "" || h.Plugin.ID == "" || h.Plugin.Version == "" || h.CapabilityID == "" {
		return fmt.Errorf("durable invocation handle identity is incomplete")
	}
	if h.DescriptorFingerprint == "" || h.PluginLockFingerprint == "" || h.RuntimePolicyFingerprint == "" {
		return fmt.Errorf("durable invocation handle runtime fingerprints are required")
	}
	if h.RuntimeKind == "" || h.RuntimeHandle == "" {
		return fmt.Errorf("durable invocation runtime identity is required")
	}
	switch h.Lifecycle {
	case RuntimeStarting, RuntimeRunning, RuntimeSucceeded, RuntimeFailed, RuntimeMissing, RuntimeUnknown:
	default:
		return fmt.Errorf("invalid runtime lifecycle %q", h.Lifecycle)
	}
	if h.Revision < 1 {
		return fmt.Errorf("durable invocation handle revision must be positive")
	}
	if h.RuntimeFencingToken > h.FencingToken {
		return fmt.Errorf("durable invocation runtime fencing token %d exceeds supervisor fencing token %d", h.RuntimeFencingToken, h.FencingToken)
	}
	return nil
}

// InvocationHandleStore is intentionally storage-neutral. A production host
// can implement it on PostgreSQL/etcd/etc. Save uses optimistic concurrency so
// two recovery hosts cannot silently overwrite one another's observations.
type InvocationHandleStore interface {
	Load(context.Context, string) (DurableInvocationHandle, bool, error)
	Save(context.Context, DurableInvocationHandle, int) (DurableInvocationHandle, error)
}

type MemoryInvocationHandleStore struct {
	mu   sync.Mutex
	byID map[string]DurableInvocationHandle
}

func NewMemoryInvocationHandleStore() *MemoryInvocationHandleStore {
	return &MemoryInvocationHandleStore{byID: map[string]DurableInvocationHandle{}}
}

func (s *MemoryInvocationHandleStore) Load(_ context.Context, id string) (DurableInvocationHandle, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.byID[id]
	return h, ok, nil
}

func (s *MemoryInvocationHandleStore) Save(_ context.Context, h DurableInvocationHandle, expectedRevision int) (DurableInvocationHandle, error) {
	if err := h.Validate(); err != nil {
		return DurableInvocationHandle{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, exists := s.byID[h.InvocationID]
	if !exists {
		if expectedRevision != 0 {
			return DurableInvocationHandle{}, fmt.Errorf("invocation %s does not exist at expected revision %d", h.InvocationID, expectedRevision)
		}
		h.Revision = 1
	} else {
		if cur.Revision != expectedRevision {
			return DurableInvocationHandle{}, fmt.Errorf("invocation %s revision conflict: current=%d expected=%d", h.InvocationID, cur.Revision, expectedRevision)
		}
		h.Revision = cur.Revision + 1
	}
	s.byID[h.InvocationID] = h
	return h, nil
}

// RuntimeObservation never carries request payloads or resource material.
type RuntimeObservation struct {
	Lifecycle  RuntimeLifecycle `json:"lifecycle"`
	ObservedAt string           `json:"observedAt,omitempty"`
	Message    string           `json:"message,omitempty"`
}

func (o RuntimeObservation) normalize() RuntimeObservation {
	if o.ObservedAt == "" {
		o.ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	return o
}

// ReattachableRuntime is a host adapter for an external process/container
// runtime. Observe is non-authoritative. Reattach is authority-bearing and MUST
// fence stale tokens at the runtime boundary as well as at the resource broker.
type ReattachableRuntime interface {
	ObserveInvocation(context.Context, string) (RuntimeObservation, error)
	ReattachInvocation(context.Context, string, uint64) (RuntimeObservation, error)
}

type InvocationRecoveryOutcome string

const (
	RecoveryReattached       InvocationRecoveryOutcome = "reattached"
	RecoveryAlreadySucceeded InvocationRecoveryOutcome = "already-succeeded"
	RecoveryAlreadyFailed    InvocationRecoveryOutcome = "already-failed"
	RecoveryRuntimeLost      InvocationRecoveryOutcome = "runtime-lost"
	RecoveryWaitingObserve   InvocationRecoveryOutcome = "waiting-observation"
)

type InvocationRecoveryResult struct {
	InvocationID string                    `json:"invocationId"`
	Outcome      InvocationRecoveryOutcome `json:"outcome"`
	Handle       DurableInvocationHandle   `json:"handle"`
	Observation  RuntimeObservation        `json:"observation"`
	Findings     []string                  `json:"findings,omitempty"`
}

func validateHandleAgainstPlan(h DurableInvocationHandle, plan EnvironmentPlan) error {
	if h.Environment != plan.Environment {
		return fmt.Errorf("invocation %s environment changed: handle=%s plan=%s", h.InvocationID, h.Environment, plan.Environment)
	}
	if h.PluginLockFingerprint != plan.PluginLockFingerprint {
		return fmt.Errorf("invocation %s plugin lock fingerprint changed", h.InvocationID)
	}
	if h.RuntimePolicyFingerprint != plan.RuntimePolicyFingerprint {
		return fmt.Errorf("invocation %s runtime policy fingerprint changed", h.InvocationID)
	}
	cr, ok := plan.Capability(plugin.CapabilityDeploymentExecutor, h.CapabilityID)
	if !ok {
		return fmt.Errorf("invocation %s capability %s is not uniquely resolved", h.InvocationID, h.CapabilityID)
	}
	if cr.Plugin != h.Plugin {
		return fmt.Errorf("invocation %s plugin changed: handle=%s@%s plan=%s@%s", h.InvocationID, h.Plugin.ID, h.Plugin.Version, cr.Plugin.ID, cr.Plugin.Version)
	}
	for _, pr := range plan.Plugins {
		if pr.Plugin == h.Plugin {
			if pr.DescriptorFingerprint != h.DescriptorFingerprint {
				return fmt.Errorf("invocation %s plugin descriptor changed", h.InvocationID)
			}
			if h.ArtifactDigest != "" && pr.Artifact.Digest != h.ArtifactDigest {
				return fmt.Errorf("invocation %s artifact digest changed", h.InvocationID)
			}
			return nil
		}
	}
	return fmt.Errorf("invocation %s plugin runtime is not present in environment plan", h.InvocationID)
}

// ReconcileDurableInvocation observes an external runtime after host restart or
// supervisor takeover. It never re-executes a missing invocation. A running
// invocation is reattached only after the caller supplies the new monotonic
// fencing token acquired from the SupervisorOwnershipStore.
func ReconcileDurableInvocation(ctx context.Context, store InvocationHandleStore, rt ReattachableRuntime, plan EnvironmentPlan, invocationID, ownerID string, fencingToken uint64) (InvocationRecoveryResult, error) {
	if store == nil || rt == nil {
		return InvocationRecoveryResult{}, fmt.Errorf("invocation recovery requires handle store and reattachable runtime")
	}
	if ownerID == "" || fencingToken == 0 {
		return InvocationRecoveryResult{}, fmt.Errorf("invocation recovery requires supervisor owner id and fencing token")
	}
	h, ok, err := store.Load(ctx, invocationID)
	if err != nil {
		return InvocationRecoveryResult{}, err
	}
	if !ok {
		return InvocationRecoveryResult{}, fmt.Errorf("unknown durable invocation %s", invocationID)
	}
	if err := h.Validate(); err != nil {
		return InvocationRecoveryResult{}, err
	}
	if err := validateHandleAgainstPlan(h, plan); err != nil {
		return InvocationRecoveryResult{}, err
	}
	if h.FencingToken > fencingToken {
		return InvocationRecoveryResult{}, fmt.Errorf("invocation %s recovery uses stale fencing token %d; handle has %d", invocationID, fencingToken, h.FencingToken)
	}
	if h.FencingToken == fencingToken && h.SupervisorOwnerID != "" && h.SupervisorOwnerID != ownerID {
		return InvocationRecoveryResult{}, fmt.Errorf("invocation %s fencing token %d is already owned by %s", invocationID, fencingToken, h.SupervisorOwnerID)
	}

	obs, err := rt.ObserveInvocation(ctx, h.RuntimeHandle)
	if err != nil {
		return InvocationRecoveryResult{}, fmt.Errorf("observe invocation %s: %w", invocationID, err)
	}
	obs = obs.normalize()
	outcome := RecoveryWaitingObserve
	findings := []string{}

	switch obs.Lifecycle {
	case RuntimeRunning, RuntimeStarting:
		attached, err := rt.ReattachInvocation(ctx, h.RuntimeHandle, fencingToken)
		if err != nil {
			return InvocationRecoveryResult{}, fmt.Errorf("reattach invocation %s: %w", invocationID, err)
		}
		obs = attached.normalize()
		if obs.Lifecycle == RuntimeRunning || obs.Lifecycle == RuntimeStarting {
			outcome = RecoveryReattached
		} else if obs.Lifecycle == RuntimeSucceeded {
			outcome = RecoveryAlreadySucceeded
		} else if obs.Lifecycle == RuntimeFailed {
			outcome = RecoveryAlreadyFailed
		} else if obs.Lifecycle == RuntimeMissing {
			outcome = RecoveryRuntimeLost
			findings = append(findings, "runtime disappeared during reattach; do not re-execute automatically")
		}
	case RuntimeSucceeded:
		outcome = RecoveryAlreadySucceeded
	case RuntimeFailed:
		outcome = RecoveryAlreadyFailed
	case RuntimeMissing:
		outcome = RecoveryRuntimeLost
		findings = append(findings, "runtime handle is missing; automatic replay is forbidden because external side effects are unknown")
	case RuntimeUnknown:
		outcome = RecoveryWaitingObserve
		findings = append(findings, "runtime state is unknown; keep the deployment action unresolved")
		findings = append(findings, fmt.Sprintf("runtime fencing authority is unresolved; supervisor generation is %d while runtime-confirmed generation remains %d", fencingToken, h.RuntimeFencingToken))
	default:
		return InvocationRecoveryResult{}, fmt.Errorf("runtime returned invalid lifecycle %q", obs.Lifecycle)
	}

	next := h
	next.SupervisorOwnerID = ownerID
	next.FencingToken = fencingToken
	if outcome == RecoveryReattached {
		next.RuntimeFencingToken = fencingToken
	}
	next.Lifecycle = obs.Lifecycle
	next.LastObservedAt = obs.ObservedAt
	next.LastMessage = obs.Message
	saved, err := store.Save(ctx, next, h.Revision)
	if err != nil {
		return InvocationRecoveryResult{}, err
	}
	sort.Strings(findings)
	return InvocationRecoveryResult{InvocationID: invocationID, Outcome: outcome, Handle: saved, Observation: obs, Findings: findings}, nil
}

// ReferenceReattachableRuntime is a deterministic host adapter used by tests
// and reference examples. It demonstrates runtime-side fencing without claiming
// to be a production process/container implementation.
type ReferenceReattachableRuntime struct {
	mu     sync.Mutex
	states map[string]RuntimeObservation
	tokens map[string]uint64
}

func NewReferenceReattachableRuntime() *ReferenceReattachableRuntime {
	return &ReferenceReattachableRuntime{states: map[string]RuntimeObservation{}, tokens: map[string]uint64{}}
}

func (r *ReferenceReattachableRuntime) Put(handle string, state RuntimeLifecycle, token uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states[handle] = RuntimeObservation{Lifecycle: state, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	r.tokens[handle] = token
}

func (r *ReferenceReattachableRuntime) ObserveInvocation(_ context.Context, handle string) (RuntimeObservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.states[handle]
	if !ok {
		return RuntimeObservation{Lifecycle: RuntimeMissing, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), Message: "runtime handle not found"}, nil
	}
	o.ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return o, nil
}

func (r *ReferenceReattachableRuntime) ReattachInvocation(_ context.Context, handle string, token uint64) (RuntimeObservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.states[handle]
	if !ok {
		return RuntimeObservation{Lifecycle: RuntimeMissing}, nil
	}
	current := r.tokens[handle]
	if token < current {
		return RuntimeObservation{}, fmt.Errorf("runtime handle %s rejected stale fencing token %d; active=%d", handle, token, current)
	}
	if token > current {
		r.tokens[handle] = token
	}
	o.ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
	o.Message = fmt.Sprintf("reattached with fencing token %d", token)
	return o, nil
}
