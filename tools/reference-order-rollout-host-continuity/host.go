package orderrollouthost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/core"
	"tei/internal/deployment/executor"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

// Host-side configuration passed through the environment of a host process.
// These are the concrete values the host's own secret manager, network and
// mount adapters hold. None of them may appear in any record or output.
const (
	EnvControlDSN   = "TEI_ORHC_CONTROL_DSN"
	EnvSecretHandle = "TEI_ORHC_SECRET_HANDLE"
	EnvSecretValue  = "TEI_ORHC_SECRET_VALUE"
	EnvEndpoint     = "TEI_ORHC_ENDPOINT"
	EnvMountHandle  = "TEI_ORHC_MOUNT_HANDLE"
	EnvWorkdir      = "TEI_ORHC_WORKDIR"
	// ChildArgsEnv lets a test binary act as the CLI or the tool executable.
	ChildArgsEnv = "TEI_ORHC_CHILD_ARGS"
)

// Logical resource identities of the production environment definition.
const (
	resCredential = "production.sales_core.migration_credential"
	resEndpoint   = "production.sales_core.database_endpoint"
	resWorkdir    = "production.sales_core.migration_workdir"
	mountTarget   = "/tei/migration"
)

// HostSecrets are the concrete values of the host adapters.
type HostSecrets struct {
	ControlDSN   string
	SecretHandle string
	SecretValue  string // the target database credential (a DSN with password)
	Endpoint     string
	MountHandle  string
	Workdir      string
}

// SecretsFromEnv reads the host adapter values from the environment.
func SecretsFromEnv() HostSecrets {
	return HostSecrets{ControlDSN: os.Getenv(EnvControlDSN), SecretHandle: os.Getenv(EnvSecretHandle), SecretValue: os.Getenv(EnvSecretValue),
		Endpoint: os.Getenv(EnvEndpoint), MountHandle: os.Getenv(EnvMountHandle), Workdir: os.Getenv(EnvWorkdir)}
}

// Env returns the values as environment entries for a host process.
func (s HostSecrets) Env() []string {
	return []string{EnvControlDSN + "=" + s.ControlDSN, EnvSecretHandle + "=" + s.SecretHandle, EnvSecretValue + "=" + s.SecretValue,
		EnvEndpoint + "=" + s.Endpoint, EnvMountHandle + "=" + s.MountHandle, EnvWorkdir + "=" + s.Workdir}
}

// Sensitive returns every concrete value that must never be recorded.
func (s HostSecrets) Sensitive() []string {
	return []string{s.ControlDSN, s.SecretHandle, s.SecretValue, s.Endpoint, s.MountHandle, s.Workdir}
}

// HostConfig configures one supervising host process.
type HostConfig struct {
	Role         string        `json:"role"` // "a" invokes; "b" takes over
	Owner        string        `json:"owner"`
	Paths        Paths         `json:"paths"`
	BinDir       string        `json:"-"` // directory holding the "atlas" executable
	ClockOffset  time.Duration `json:"clockOffset"`
	OwnershipTTL time.Duration `json:"ownershipTtl"`
	// The resource lease TTL is the plan's renew-before plus LeaseSlack, so
	// the product supervisor renews every LeaseSlack.
	LeaseSlack    time.Duration `json:"leaseSlack"`
	ApplyHoldMS   int           `json:"applyHoldMs"`
	FailRenew     bool          `json:"failRenew"`
	FailHeartbeat bool          `json:"failHeartbeat"`
	ProbeStale    bool          `json:"probeStale"`
	// Host B.
	ClaimPoll      time.Duration `json:"claimPoll"`
	ClaimTimeout   time.Duration `json:"claimTimeout"`
	ObserveWindow  time.Duration `json:"observeWindow"`
	ReleaseBarrier string        `json:"releaseBarrier"`
	// The child executable is the test binary when ChildArgs is set.
	ChildArgs bool `json:"-"`
}

func (c HostConfig) hostNow() time.Time { return time.Now().Add(c.ClockOffset) }

type referenceVerifier struct{}

// Verify accepts the plan's artifact without checking the executable: there
// is no real migration.atlas executable, and the stand-in is not the pinned
// artifact. The verifier id says so.
func (referenceVerifier) Verify(context.Context, plugin.Identity, plugin.Artifact) (pruntime.ArtifactVerification, error) {
	return pruntime.ArtifactVerification{VerifierID: "orhc-reference-verifier-not-checking-the-stand-in", DigestVerified: true, SignatureVerified: true, ProvenanceChecked: true}, nil
}

// hostAdapters are the host enforcement adapters. Network policy is recorded
// only (no OS enforcement); the filesystem adapter maps the opaque mount
// source handle to the working directory; the secret injector resolves the
// opaque handle in the host's secret manager.
type hostAdapters struct {
	secrets HostSecrets
	mu      sync.Mutex
	env     []string
	dir     string
	allow   int
}

func (h *hostAdapters) ApplyNetworkPolicy(_ context.Context, p pruntime.NetworkPolicy) (pruntime.Cleanup, error) {
	h.mu.Lock()
	h.allow = len(p.Allowlist)
	h.mu.Unlock()
	return func(context.Context) error { return nil }, nil
}

func (h *hostAdapters) ApplyFilesystemPolicy(_ context.Context, p pruntime.FilesystemPolicy) (pruntime.Cleanup, error) {
	for _, m := range p.Mounts {
		if m.SourceHandle != h.secrets.MountHandle {
			return nil, fmt.Errorf("unknown mount source for %s", m.Scope)
		}
		h.mu.Lock()
		h.dir = h.secrets.Workdir
		h.mu.Unlock()
	}
	return func(context.Context) error { h.mu.Lock(); h.dir = ""; h.mu.Unlock(); return nil }, nil
}

func (h *hostAdapters) InjectSecrets(_ context.Context, xs []pruntime.SecretInjection) (pruntime.Cleanup, error) {
	var env []string
	for _, x := range xs {
		if x.Handle != h.secrets.SecretHandle {
			return nil, fmt.Errorf("secret manager has no value for %s", x.Scope)
		}
		if x.Delivery != resource.SecretDeliveryEnvironment || x.Target == "" {
			return nil, fmt.Errorf("unsupported secret delivery for %s", x.Scope)
		}
		env = append(env, x.Target+"="+h.secrets.SecretValue)
	}
	h.mu.Lock()
	h.env = env
	h.mu.Unlock()
	return func(context.Context) error { h.mu.Lock(); h.env = nil; h.mu.Unlock(); return nil }, nil
}

func (h *hostAdapters) RefreshSecrets(context.Context, []pruntime.SecretInjection, []pruntime.SecretInjection) error {
	return nil
}

// atlasRunner saves the durable invocation handle from the request the host
// is about to dispatch, then runs the tool executable as a child process with
// an explicitly assembled environment.
type atlasRunner struct {
	cfg      HostConfig
	start    Start
	adapters *hostAdapters
	handles  *PgHandleStore
	control  *pgxpool.Pool
	sales    *pgxpool.Pool

	mu            sync.Mutex
	runtimeHandle string
	acceptedSeq   int64
}

func (r *atlasRunner) InvokeSandboxed(ctx context.Context, cmd pruntime.ProcessCommand, req pruntime.Request) (pruntime.Response, error) {
	if req.ResourceLease == nil {
		return pruntime.Response{}, fmt.Errorf("no resource lease")
	}
	var artifact string
	for _, p := range r.start.Plan.Plugins {
		if p.Plugin == req.Plugin {
			artifact = p.Artifact.Digest
		}
	}
	handle := "runtime://orhc/atlas/" + randomHex(6)
	h := pruntime.DurableInvocationHandle{FormatVersion: 1, InvocationID: r.start.InvocationID, RunID: r.start.RunID, ActionID: r.start.Command.ActionID,
		RequestID: req.RequestID, Environment: r.start.Plan.Environment, Plugin: req.Plugin, CapabilityID: req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint, PluginLockFingerprint: req.PluginLockFingerprint, RuntimePolicyFingerprint: req.ResourceLease.RuntimePolicyFingerprint,
		ArtifactDigest: artifact, RuntimeKind: "process", RuntimeHandle: handle, ResourceLeaseID: req.ResourceLease.ID,
		SupervisorOwnerID: r.cfg.Owner, FencingToken: req.ResourceLease.FencingToken, RuntimeFencingToken: req.ResourceLease.FencingToken,
		Lifecycle: pruntime.RuntimeStarting, StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if _, err := r.handles.Save(ctx, h, 0); err != nil {
		return pruntime.Response{}, fmt.Errorf("save durable invocation handle: %w", err)
	}
	r.mu.Lock()
	r.runtimeHandle = handle
	r.mu.Unlock()
	callEvent(ctx, r.control, r.start.InvocationID, r.cfg.Owner, "handle-saved", map[string]any{"leaseId": req.ResourceLease.ID, "fencingToken": req.ResourceLease.FencingToken})

	additional := map[string]string{EnvRuntimeHandle: handle, EnvApplyHold: strconv.Itoa(r.cfg.ApplyHoldMS)}
	if r.cfg.ChildArgs {
		additional[ChildArgsEnv] = `["atlas"]`
	}
	r.adapters.mu.Lock()
	dir, secretEnv := r.adapters.dir, append([]string(nil), r.adapters.env...)
	r.adapters.mu.Unlock()
	if dir == "" || len(secretEnv) == 0 {
		return pruntime.Response{}, fmt.Errorf("sandbox mount or secret injection was not applied")
	}
	profile := pruntime.HostProcessEnvironmentProfile{SearchDirectories: []string{r.cfg.BinDir}, HomeDirectory: dir, TemporaryDirectory: dir, Additional: additional}
	base, err := profile.Environment()
	if err != nil {
		return pruntime.Response{}, err
	}
	launcher := pruntime.StdioProcessLauncher{Env: append(base, secretEnv...), Dir: dir, MaxExecutionDuration: 10 * time.Minute}

	watchCtx, stopWatch := context.WithCancel(ctx)
	defer stopWatch()
	go r.watchAcceptance(watchCtx, handle)
	return launcher.InvokeSandboxed(ctx, cmd, req)
}

// watchAcceptance records the tool's acceptance notice (the job registered)
// in the call record. The notice is not evidence of the target state.
func (r *atlasRunner) watchAcceptance(ctx context.Context, handle string) {
	for {
		var accepted bool
		err := r.sales.QueryRow(ctx, `SELECT accepted_at IS NOT NULL FROM tei_atlas.job WHERE runtime_handle=$1`, handle).Scan(&accepted)
		if err == nil && accepted {
			seq := callEvent(ctx, r.control, r.start.InvocationID, r.cfg.Owner, "accepted-notice", map[string]any{"runtimeHandle": handle})
			r.mu.Lock()
			r.acceptedSeq = seq
			r.mu.Unlock()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func callEvent(ctx context.Context, pool *pgxpool.Pool, invocation, actor, kind string, detail map[string]any) int64 {
	raw, _ := json.Marshal(detail)
	var seq int64
	_ = pool.QueryRow(context.WithoutCancel(ctx), `INSERT INTO orhc.call_event(invocation_id,actor,kind,detail) VALUES ($1,$2,$3,$4) RETURNING seq`, invocation, actor, kind, raw).Scan(&seq)
	return seq
}

// Probe is one operation a stale host attempted after the takeover.
type Probe struct {
	Op       string `json:"op"`
	Rejected bool   `json:"rejected"`
	Message  string `json:"message,omitempty"`
}

// CallRecord is the output of step 2 (host A).
type CallRecord struct {
	Identity       Identity                   `json:"identity"`
	Owner          string                     `json:"owner"`
	LeaseID        string                     `json:"leaseId"`
	FencingToken   uint64                     `json:"fencingToken"`
	RuntimeHandle  string                     `json:"runtimeHandle"`
	AcceptedSeq    int64                      `json:"acceptedNoticeSeq"`
	CallFailed     bool                       `json:"callFailed"`
	CallError      string                     `json:"callError,omitempty"`
	Result         *executor.Result           `json:"result,omitempty"`
	Invocation     *executor.PluginInvocation `json:"invocation,omitempty"`
	StaleProbes    []Probe                    `json:"staleProbes,omitempty"`
	NetworkAllowed int                        `json:"networkAllowlistEntries"`
}

func invocationRecord(plan pruntime.EnvironmentPlan, pv pruntime.InvocationProvenance) *executor.PluginInvocation {
	return &executor.PluginInvocation{PluginID: pv.Plugin.ID, PluginVersion: pv.Plugin.Version, DescriptorFingerprint: pv.DescriptorFingerprint,
		PluginLockFingerprint: pv.PluginLockFingerprint, RuntimePolicyFingerprint: plan.RuntimePolicyFingerprint, ArtifactDigest: pv.Artifact.Digest,
		Boundary: string(pv.Boundary), Sandboxed: pv.Sandboxed, PermissionLease: pv.Permissions, ResourceLeaseID: pv.ResourceLeaseID,
		ResourceLeaseExpiresAt: pv.ResourceLeaseExpiresAt, Resources: pv.Resources, ResourceLeaseReleased: pv.ResourceLeaseReleased,
		ResourceLeaseRevision: pv.ResourceLeaseRevision, ResourceLeaseFencingToken: pv.ResourceLeaseFencingToken, ResourceLeaseRenewals: pv.ResourceLeaseRenewals,
		ResourceLeaseRefreshes: pv.ResourceLeaseRefreshes, ResourceLeaseSupervision: pv.ResourceLeaseSupervision, ResourceLeaseOwnership: pv.ResourceLeaseOwnership}
}

func hostParts(ctx context.Context, cfg HostConfig, sec HostSecrets, start Start) (*pgxpool.Pool, *pgxpool.Pool, *PgOwnershipStore, *PgBroker, error) {
	control, err := pgxpool.New(ctx, sec.ControlDSN)
	if err != nil {
		return nil, nil, nil, nil, errors.New("control database configuration rejected")
	}
	sales, err := pgxpool.New(ctx, sec.SecretValue)
	if err != nil {
		control.Close()
		return nil, nil, nil, nil, errors.New("target database configuration rejected")
	}
	cr, ok := start.Plan.Capability(plugin.CapabilityDeploymentExecutor, core.CanonicalID(CapabilityID))
	if !ok {
		return nil, nil, nil, nil, fmt.Errorf("capability %s not in plan", CapabilityID)
	}
	own := &PgOwnershipStore{Pool: control, Actor: cfg.Owner, HostNow: cfg.hostNow, FailHeartbeat: cfg.FailHeartbeat}
	broker := &PgBroker{Pool: control, Actor: cfg.Owner, Now: cfg.hostNow, FailRenew: cfg.FailRenew,
		TTL: cr.LeasePolicy.WithDefaults().RenewBeforeDuration() + cfg.LeaseSlack,
		Adapters: []resource.Adapter{
			&resource.ReferenceSecretManagerAdapter{Handles: map[core.CanonicalID]string{core.MustCanonicalID(resCredential): sec.SecretHandle}},
			&resource.StaticAdapter{ResourceKind: permission.KindNetwork, Entries: []resource.StaticResource{{ResourceID: core.MustCanonicalID(resEndpoint), NetworkEndpoint: sec.Endpoint}}},
			&resource.StaticAdapter{ResourceKind: permission.KindFilesystem, Entries: []resource.StaticResource{{ResourceID: core.MustCanonicalID(resWorkdir), MountSourceHandle: sec.MountHandle, MountPath: mountTarget}}},
		}}
	return control, sales, own, broker, nil
}

// RunHostA is step 2: one invocation of the first command through the
// product registry with the isolation launcher, the persistent lease broker
// and the persistent ownership store. With ProbeStale it then attempts every
// authority-bearing operation again with its own (possibly stale) copies.
func RunHostA(ctx context.Context, cfg HostConfig, sec HostSecrets) (CallRecord, error) {
	start, err := LoadStart(cfg.Paths)
	if err != nil {
		return CallRecord{}, err
	}
	control, sales, own, broker, err := hostParts(ctx, cfg, sec, start)
	if err != nil {
		return CallRecord{}, err
	}
	defer control.Close()
	defer sales.Close()
	adapters := &hostAdapters{secrets: sec}
	runner := &atlasRunner{cfg: cfg, start: start, adapters: adapters, handles: &PgHandleStore{Pool: control}, control: control, sales: sales}
	reg := pruntime.Registry{
		Sandboxes: pruntime.HostSandboxLauncher{Network: adapters, Filesystem: adapters, Secrets: adapters, Runner: runner, Now: cfg.hostNow},
		Verifier:  referenceVerifier{}, Resources: broker, SupervisorOwnerships: own, SupervisorOwnerID: cfg.Owner, SupervisorOwnershipTTL: cfg.OwnershipTTL,
	}
	callEvent(ctx, control, start.InvocationID, cfg.Owner, "invoke-start", map[string]any{"actionId": start.Command.ActionID, "requestId": start.RequestID})
	inv, callErr := reg.InvokeWithProvenance(ctx, start.Plan, plugin.CapabilityDeploymentExecutor, core.CanonicalID(CapabilityID), "execute", start.RequestID, start.Command)
	rec := CallRecord{Identity: start.Identity(), Owner: cfg.Owner, Invocation: invocationRecord(start.Plan, inv.Provenance), NetworkAllowed: adapters.allow}
	rec.Identity.RequestID = inv.Provenance.RequestID
	rec.LeaseID = inv.Provenance.ResourceLeaseID
	rec.FencingToken = inv.Provenance.ResourceLeaseFencingToken
	runner.mu.Lock()
	rec.RuntimeHandle, rec.AcceptedSeq = runner.runtimeHandle, runner.acceptedSeq
	runner.mu.Unlock()
	if callErr != nil {
		rec.CallFailed = true
		rec.CallError = callErr.Error()
	} else {
		var out executor.Result
		if err := json.Unmarshal(inv.Response.Payload, &out); err == nil {
			out.ExecutorID = CapabilityID
			rec.Result = &out
		}
	}
	callEvent(ctx, control, start.InvocationID, cfg.Owner, "invoke-end", map[string]any{"failed": rec.CallFailed})
	if cfg.ProbeStale && rec.LeaseID != "" {
		rec.StaleProbes = staleProbes(ctx, cfg, start, control, sales, own, broker, rec)
	}
	return rec, nil
}

func probe(op string, err error) Probe {
	p := Probe{Op: op, Rejected: err != nil}
	if err != nil {
		p.Message = err.Error()
	}
	return p
}

func staleProbes(ctx context.Context, cfg HostConfig, start Start, control, sales *pgxpool.Pool, own *PgOwnershipStore, broker *PgBroker, rec CallRecord) []Probe {
	o := resource.SupervisorOwnership{LeaseID: rec.LeaseID, OwnerID: cfg.Owner, FencingToken: rec.FencingToken}
	rev, _, _, _ := LeaseAuthority(ctx, control, rec.LeaseID)
	stale := resource.Lease{ID: rec.LeaseID, Revision: rev, FencingToken: rec.FencingToken}
	var out []Probe
	_, err := own.Heartbeat(ctx, o, cfg.OwnershipTTL)
	out = append(out, probe("ownership-heartbeat", err))
	_, _, err = own.Claim(ctx, rec.LeaseID, cfg.Owner, cfg.OwnershipTTL)
	out = append(out, probe("ownership-reclaim", err))
	out = append(out, probe("ownership-release", own.Release(ctx, o)))
	_, err = broker.Renew(ctx, resource.AcquireRequest{}, stale)
	out = append(out, probe("lease-renew", err))
	out = append(out, probe("lease-release", broker.Release(ctx, stale)))
	_, err = broker.Fence(ctx, stale, rec.FencingToken)
	out = append(out, probe("lease-fence", err))
	jobs := &JobRuntime{Pool: sales, ObserveWindow: cfg.ObserveWindow}
	if rec.RuntimeHandle != "" {
		_, err = jobs.ReattachInvocation(ctx, rec.RuntimeHandle, rec.FencingToken)
		out = append(out, probe("runtime-reattach", err))
	}
	_, err = pruntime.ReconcileDurableInvocation(ctx, &PgHandleStore{Pool: control}, jobs, start.Plan, start.InvocationID, cfg.Owner, rec.FencingToken)
	out = append(out, probe("invocation-reconcile", err))
	return out
}

// TakeoverRecord is the output of steps 3 and 4 (host B).
type TakeoverRecord struct {
	Identity           Identity                         `json:"identity"`
	HandleRevisionSeen int                              `json:"handleRevisionSeen"`
	PreviousOwner      string                           `json:"previousOwner"`
	PreviousToken      uint64                           `json:"previousToken"`
	LeaseID            string                           `json:"leaseId"`
	Owner              string                           `json:"owner"`
	FencingToken       uint64                           `json:"fencingToken"`
	RefusedClaims      int                              `json:"refusedClaims"`
	NaiveAtRefusals    []bool                           `json:"naiveHostClockTakeoverAtRefusals"`
	NaiveAtTakeover    bool                             `json:"naiveHostClockTakeoverAtTakeover"`
	TakeoverWaitMS     int64                            `json:"takeoverWaitMs"`
	PlanMatchesRun     bool                             `json:"planMatchesRun"`
	ObservesBefore     int64                            `json:"runtimeObservesBeforeReconcile"`
	Observes           int64                            `json:"runtimeObserves"`
	Reattaches         int64                            `json:"runtimeReattaches"`
	Recovery           *pruntime.InvocationRecoveryResult `json:"recovery,omitempty"`
	RecoveryError      string                           `json:"recoveryError,omitempty"`
	ReconciledSeq      int64                            `json:"reconciledSeq"`
	LeaseReleased      bool                             `json:"leaseReleased"`
	LeaseReleaseError  string                           `json:"leaseReleaseError,omitempty"`
	OwnershipReleased  bool                             `json:"ownershipReleased"`
	LeaseStateAfter    string                           `json:"leaseStateAfter"`
	OwnershipRowsAfter int                              `json:"ownershipRowsAfter"`
}

// RunHostB is steps 3 and 4: wait for the durable invocation handle written
// in step 2, take over the supervisor ownership after it stood still, fence
// the lease, reconcile the invocation, wait for the release barrier and
// return the lease and the ownership.
func RunHostB(ctx context.Context, cfg HostConfig, sec HostSecrets) (TakeoverRecord, error) {
	start, err := LoadStart(cfg.Paths)
	if err != nil {
		return TakeoverRecord{}, err
	}
	control, sales, own, broker, err := hostParts(ctx, cfg, sec, start)
	if err != nil {
		return TakeoverRecord{}, err
	}
	defer control.Close()
	defer sales.Close()
	handles := &PgHandleStore{Pool: control}
	var h pruntime.DurableInvocationHandle
	for {
		var ok bool
		if h, ok, err = handles.Load(ctx, start.InvocationID); err != nil {
			return TakeoverRecord{}, err
		} else if ok {
			break
		}
		select {
		case <-ctx.Done():
			return TakeoverRecord{}, ctx.Err()
		case <-time.After(cfg.ClaimPoll):
		}
	}
	rec := TakeoverRecord{Identity: Identity{RunID: h.RunID, StartRecordDigest: start.StartRecordDigest, PlanFingerprint: start.PlanFingerprint,
		PluginLockFingerprint: h.PluginLockFingerprint, RuntimePolicyFingerprint: h.RuntimePolicyFingerprint, ActionID: h.ActionID,
		IdempotencyKey: start.Command.IdempotencyKey, InvocationID: h.InvocationID, RequestID: h.RequestID},
		HandleRevisionSeen: h.Revision, PreviousOwner: h.SupervisorOwnerID, PreviousToken: h.FencingToken, LeaseID: h.ResourceLeaseID, Owner: cfg.Owner,
		PlanMatchesRun: start.RuntimePolicyMatches && start.PluginLockMatches}
	began := time.Now()
	deadline := began.Add(cfg.ClaimTimeout)
	var o resource.SupervisorOwnership
	for {
		var takeover bool
		o, takeover, err = own.Claim(ctx, h.ResourceLeaseID, cfg.Owner, cfg.OwnershipTTL)
		var held *LeaseHeldError
		if err == nil {
			if !takeover {
				return rec, fmt.Errorf("claim succeeded without a takeover")
			}
			break
		}
		if !errors.As(err, &held) {
			return rec, err
		}
		rec.RefusedClaims++
		rec.NaiveAtRefusals = append(rec.NaiveAtRefusals, held.NaiveHostClockTakeover)
		if time.Now().After(deadline) {
			return rec, fmt.Errorf("no takeover within %s", cfg.ClaimTimeout)
		}
		select {
		case <-ctx.Done():
			return rec, ctx.Err()
		case <-time.After(cfg.ClaimPoll):
		}
	}
	rec.TakeoverWaitMS = time.Since(began).Milliseconds()
	rec.FencingToken = o.FencingToken
	_ = control.QueryRow(ctx, `SELECT naive_host_clock_takeover FROM orhc.ownership_event WHERE lease_id=$1 AND op='takeover' ORDER BY seq DESC LIMIT 1`, h.ResourceLeaseID).Scan(&rec.NaiveAtTakeover)
	callEvent(ctx, control, start.InvocationID, cfg.Owner, "takeover", map[string]any{"leaseId": h.ResourceLeaseID, "fencingToken": o.FencingToken})

	// Keep the new ownership alive while it is held.
	hbCtx, stopHB := context.WithCancel(ctx)
	hbDone := make(chan struct{})
	go func() {
		defer close(hbDone)
		t := time.NewTicker(cfg.OwnershipTTL / 3)
		defer t.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-t.C:
				_, _ = own.Heartbeat(hbCtx, o, cfg.OwnershipTTL)
			}
		}
	}()
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(func() { stopHB(); <-hbDone }) }
	defer stop()

	fenced, err := broker.Fence(ctx, resource.Lease{ID: h.ResourceLeaseID}, o.FencingToken)
	if err != nil {
		return rec, fmt.Errorf("fence lease: %w", err)
	}
	jobs := &JobRuntime{Pool: sales, ObserveWindow: cfg.ObserveWindow}
	rec.ObservesBefore = jobs.Observes.Load()
	res, rerr := pruntime.ReconcileDurableInvocation(ctx, handles, jobs, start.Plan, start.InvocationID, cfg.Owner, o.FencingToken)
	rec.Observes, rec.Reattaches = jobs.Observes.Load(), jobs.Reattachs.Load()
	if rerr != nil {
		rec.RecoveryError = rerr.Error()
	} else {
		rec.Recovery = &res
	}
	rec.ReconciledSeq = callEvent(ctx, control, start.InvocationID, cfg.Owner, "reconciled", map[string]any{"failed": rerr != nil})

	if cfg.ReleaseBarrier != "" {
		for {
			var n int
			if err := control.QueryRow(ctx, `SELECT count(*) FROM orhc.barrier WHERE name=$1`, cfg.ReleaseBarrier).Scan(&n); err != nil {
				return rec, err
			}
			if n > 0 {
				break
			}
			select {
			case <-ctx.Done():
				return rec, ctx.Err()
			case <-time.After(cfg.ClaimPoll):
			}
		}
	}
	rev, _, _, err := LeaseAuthority(ctx, control, h.ResourceLeaseID)
	if err != nil {
		return rec, err
	}
	if err := broker.Release(ctx, resource.Lease{ID: fenced.ID, Revision: rev, FencingToken: o.FencingToken}); err != nil {
		rec.LeaseReleaseError = err.Error()
	} else {
		rec.LeaseReleased = true
	}
	stop()
	rec.OwnershipReleased = own.Release(ctx, o) == nil
	_, _, rec.LeaseStateAfter, _ = LeaseAuthority(ctx, control, h.ResourceLeaseID)
	_ = control.QueryRow(ctx, `SELECT count(*) FROM orhc.supervisor_ownership WHERE lease_id=$1`, h.ResourceLeaseID).Scan(&rec.OwnershipRowsAfter)
	callEvent(ctx, control, start.InvocationID, cfg.Owner, "released", map[string]any{"lease": rec.LeaseReleased, "ownership": rec.OwnershipReleased})
	return rec, nil
}

// OpenBarrier lets a waiting host B proceed to the release.
func OpenBarrier(ctx context.Context, pool *pgxpool.Pool, name string) error {
	_, err := pool.Exec(ctx, `INSERT INTO orhc.barrier(name) VALUES ($1) ON CONFLICT DO NOTHING`, name)
	return err
}

// writeJSON writes v to path atomically.
func writeJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Clean(path))
}

