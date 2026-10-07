package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"tei/internal/core"
	"tei/internal/deployment/executor"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	"tei/internal/resource"
)

const ProtocolVersion = 1

type Boundary string

const (
	BoundaryInProcess Boundary = "in-process"
	BoundaryProcess   Boundary = "process"
)

type TrustClass string

const (
	TrustHostTrusted     TrustClass = "host-trusted"
	TrustExternalProcess TrustClass = "external-process"
)

type PluginRuntime struct {
	Plugin                plugin.Identity      `json:"plugin"`
	DescriptorFingerprint string               `json:"descriptorFingerprint"`
	BuildPlanFingerprint  string               `json:"buildPlanFingerprint,omitempty"`
	Boundary              Boundary             `json:"boundary"`
	Trust                 TrustClass           `json:"trust"`
	Command               string               `json:"command,omitempty"`
	Args                  []string             `json:"args,omitempty"`
	SandboxPolicy         plugin.SandboxPolicy `json:"sandboxPolicy,omitempty"`
	Artifact              plugin.Artifact      `json:"artifact,omitempty"`
}

type CapabilityRuntime struct {
	Plugin           plugin.Identity      `json:"plugin"`
	Capability       plugin.Capability    `json:"capability"`
	Boundary         Boundary             `json:"boundary"`
	Permissions      []permission.Spec    `json:"permissionLease,omitempty"`
	ResourceBindings []resource.Binding   `json:"resourceBindings,omitempty"`
	LeasePolicy      resource.LeasePolicy `json:"leasePolicy,omitempty"`
}

type EnvironmentPlan struct {
	Environment              core.CanonicalID    `json:"environment"`
	PluginLockFingerprint    string              `json:"pluginLockFingerprint"`
	RuntimePolicyFingerprint string              `json:"runtimePolicyFingerprint"`
	Plugins                  []PluginRuntime     `json:"plugins"`
	Capabilities             []CapabilityRuntime `json:"capabilities,omitempty"`
}

func BuildPlan(c plugin.Catalog, env environment.Environment, lock plugin.Lock) (EnvironmentPlan, error) {
	resolved, err := c.ResolveEnvironmentWithLock(env, lock)
	if err != nil {
		return EnvironmentPlan{}, err
	}
	lockFP, err := plugin.EnvironmentLockFingerprint(lock, env.ID)
	if err != nil {
		return EnvironmentPlan{}, err
	}
	out := EnvironmentPlan{Environment: env.ID, PluginLockFingerprint: lockFP}
	byID := map[string]plugin.Definition{}
	for _, d := range c.Plugins {
		byID[string(d.ID)+"@"+d.Version] = d
	}
	for _, ident := range resolved.Plugins {
		d, ok := byID[string(ident.ID)+"@"+ident.Version]
		if !ok {
			return EnvironmentPlan{}, fmt.Errorf("resolved plugin %s@%s missing from catalog", ident.ID, ident.Version)
		}
		locked, ok := lockedPluginForEnvironment(lock, env.ID, ident)
		if !ok {
			return EnvironmentPlan{}, fmt.Errorf("resolved plugin %s@%s missing from plugin lock", ident.ID, ident.Version)
		}
		pr := PluginRuntime{Plugin: ident, DescriptorFingerprint: plugin.DefinitionFingerprint(d), BuildPlanFingerprint: locked.BuildPlanFingerprint, Command: d.Runtime.Command, Args: append([]string(nil), d.Runtime.Args...), SandboxPolicy: d.Runtime.Sandbox, Artifact: d.Artifact}
		switch d.Runtime.Kind {
		case plugin.RuntimeBuiltin:
			pr.Boundary, pr.Trust = BoundaryInProcess, TrustHostTrusted
		case plugin.RuntimeProcess:
			// A separate process is a trust boundary but is not an OS sandbox.
			pr.Boundary, pr.Trust = BoundaryProcess, TrustExternalProcess
		default:
			return EnvironmentPlan{}, fmt.Errorf("plugin %s@%s has unsupported runtime %s", ident.ID, ident.Version, d.Runtime.Kind)
		}
		out.Plugins = append(out.Plugins, pr)
	}
	pluginRuntime := map[string]Boundary{}
	for _, pr := range out.Plugins {
		pluginRuntime[string(pr.Plugin.ID)+"@"+pr.Plugin.Version] = pr.Boundary
	}
	for _, c := range resolved.Capabilities {
		perms := grantsFor(env, c.Capability)
		bindings, err := resource.MatchBindings(perms, env.ResourceBindings)
		if err != nil && len(perms) > 0 {
			return EnvironmentPlan{}, fmt.Errorf("capability %s %s resource binding: %w", c.Capability.Kind, c.Capability.ID, err)
		}
		leasePolicy := leasePolicyFor(env, c.Capability, len(perms) > 0)
		if len(perms) > 0 {
			if err := leasePolicy.Validate(); err != nil {
				return EnvironmentPlan{}, fmt.Errorf("capability %s %s lease policy: %w", c.Capability.Kind, c.Capability.ID, err)
			}
		}
		out.Capabilities = append(out.Capabilities, CapabilityRuntime{Plugin: c.Plugin, Capability: c.Capability, Boundary: pluginRuntime[string(c.Plugin.ID)+"@"+c.Plugin.Version], Permissions: perms, ResourceBindings: bindings, LeasePolicy: leasePolicy})
	}
	sort.Slice(out.Plugins, func(i, j int) bool { return out.Plugins[i].Plugin.ID < out.Plugins[j].Plugin.ID })
	sort.Slice(out.Capabilities, func(i, j int) bool {
		if out.Capabilities[i].Capability.Kind != out.Capabilities[j].Capability.Kind {
			return out.Capabilities[i].Capability.Kind < out.Capabilities[j].Capability.Kind
		}
		if out.Capabilities[i].Capability.ID != out.Capabilities[j].Capability.ID {
			return out.Capabilities[i].Capability.ID < out.Capabilities[j].Capability.ID
		}
		return out.Capabilities[i].Plugin.ID < out.Capabilities[j].Plugin.ID
	})
	out.RuntimePolicyFingerprint = PolicyFingerprint(out)
	return out, nil
}


func lockedPluginForEnvironment(lock plugin.Lock, environmentID core.CanonicalID, ident plugin.Identity) (plugin.LockedPlugin, bool) {
	for _, environmentLock := range lock.Environments {
		if environmentLock.Environment != environmentID {
			continue
		}
		for _, locked := range environmentLock.Plugins {
			if locked.ID == ident.ID && locked.Version == ident.Version {
				return locked, true
			}
		}
	}
	return plugin.LockedPlugin{}, false
}

func PolicyFingerprint(plan EnvironmentPlan) string {
	payload := struct {
		Environment           core.CanonicalID    `json:"environment"`
		PluginLockFingerprint string              `json:"pluginLockFingerprint"`
		Plugins               []PluginRuntime     `json:"plugins"`
		Capabilities          []CapabilityRuntime `json:"capabilities"`
	}{Environment: plan.Environment, PluginLockFingerprint: plan.PluginLockFingerprint, Plugins: append([]PluginRuntime(nil), plan.Plugins...), Capabilities: append([]CapabilityRuntime(nil), plan.Capabilities...)}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func leasePolicyFor(env environment.Environment, capability plugin.Capability, hasResources bool) resource.LeasePolicy {
	if !hasResources {
		return resource.LeasePolicy{}
	}
	for _, p := range env.LeasePolicies {
		if p.CapabilityKind == string(capability.Kind) && p.CapabilityID == capability.ID {
			return p.Policy.WithDefaults()
		}
	}
	return resource.DefaultLeasePolicy()
}

func grantsFor(env environment.Environment, capability plugin.Capability) []permission.Spec {
	var out []permission.Spec
	for _, g := range env.PermissionGrants {
		if g.CapabilityKind == string(capability.Kind) && g.CapabilityID == capability.ID {
			out = append(out, g.Permission)
		}
	}
	return permission.Normalize(out)
}

func hasRequiredPermissions(c CapabilityRuntime) error {
	granted := map[string]struct{}{}
	for _, p := range c.Permissions {
		granted[p.Key()] = struct{}{}
	}
	for _, req := range c.Capability.Permissions {
		if _, ok := granted[req.Key()]; !ok {
			return fmt.Errorf("capability %s %s requires permission %s but environment did not grant it", c.Capability.Kind, c.Capability.ID, req.Key())
		}
	}
	return nil
}

func (p EnvironmentPlan) Capability(kind plugin.CapabilityKind, id core.CanonicalID) (CapabilityRuntime, bool) {
	var found *CapabilityRuntime
	for i := range p.Capabilities {
		c := p.Capabilities[i]
		if c.Capability.Kind == kind && c.Capability.ID == id {
			if found != nil {
				return CapabilityRuntime{}, false
			}
			x := c
			found = &x
		}
	}
	if found == nil {
		return CapabilityRuntime{}, false
	}
	return *found, true
}

type ResourceLeaseController interface {
	Current() resource.Lease
	Renew(context.Context) (resource.Lease, error)
	Refresh(context.Context) (resource.Lease, error)
}

type Request struct {
	ProtocolVersion       int                     `json:"protocolVersion"`
	RequestID             string                  `json:"requestId"`
	Plugin                plugin.Identity         `json:"plugin"`
	CapabilityKind        plugin.CapabilityKind   `json:"capabilityKind"`
	CapabilityID          core.CanonicalID        `json:"capabilityId"`
	DescriptorFingerprint string                  `json:"descriptorFingerprint"`
	PluginLockFingerprint string                  `json:"pluginLockFingerprint"`
	Permissions           []permission.Spec       `json:"permissionLease,omitempty"`
	ResourceLease         *resource.Lease         `json:"resourceLease,omitempty"`
	LeaseController       ResourceLeaseController `json:"-"`
	Operation             string                  `json:"operation"`
	Payload               json.RawMessage         `json:"payload,omitempty"`
}

type ResponseStatus string

const (
	ResponseOK    ResponseStatus = "ok"
	ResponseError ResponseStatus = "error"
)

type Response struct {
	ProtocolVersion       int              `json:"protocolVersion"`
	RequestID             string           `json:"requestId"`
	Plugin                plugin.Identity  `json:"plugin"`
	CapabilityID          core.CanonicalID `json:"capabilityId"`
	DescriptorFingerprint string           `json:"descriptorFingerprint"`
	Status                ResponseStatus   `json:"status"`
	Payload               json.RawMessage  `json:"payload,omitempty"`
	Error                 string           `json:"error,omitempty"`
}

type Builtin interface {
	Identity() plugin.Identity
	DescriptorFingerprint() string
	Invoke(context.Context, Request) (Response, error)
}

// BuiltinRegistration is Host-owned acceptance metadata for an in-process
// realization. Build-bound Runtime locks are checked against this object rather
// than trusting the Builtin implementation to self-report its own provenance.
type BuiltinRegistration struct {
	Builtin              Builtin `json:"-"`
	BuildPlanFingerprint string  `json:"buildPlanFingerprint"`
	RealizationDigest    string  `json:"realizationDigest"`
}

// ProcessCommand preserves executable and argv as separate values across the
// Host boundary. It is never a shell command line and must not be reparsed by
// ProcessLauncher implementations.
type ProcessCommand struct {
	Executable string   `json:"executable"`
	Args       []string `json:"args,omitempty"`
}

// ProcessLauncher is supplied by the TEI host. It receives a structured
// executable/argv pair so hosts cannot silently introduce shell semantics.
type ProcessLauncher interface {
	Invoke(context.Context, ProcessCommand, Request) (Response, error)
}

type SandboxMount struct {
	Scope        string            `json:"scope"`
	ResourceID   core.CanonicalID  `json:"resourceId"`
	SourceHandle string            `json:"sourceHandle,omitempty"`
	Target       string            `json:"target"`
	Access       permission.Access `json:"access"`
}

type SecretInjection struct {
	Scope      string                  `json:"scope"`
	ResourceID core.CanonicalID        `json:"resourceId"`
	Handle     string                  `json:"handle"`
	Delivery   resource.SecretDelivery `json:"delivery"`
	Target     string                  `json:"target,omitempty"`
}

type SandboxProfile struct {
	DenyNetworkByDefault    bool              `json:"denyNetworkByDefault"`
	DenyFilesystemByDefault bool              `json:"denyFilesystemByDefault"`
	NetworkAllowlist        []string          `json:"networkAllowlist,omitempty"`
	Mounts                  []SandboxMount    `json:"mounts,omitempty"`
	SecretHandles           []string          `json:"secretHandles,omitempty"`
	SecretInjections        []SecretInjection `json:"secretInjections,omitempty"`
}

type SandboxSpec struct {
	Command         ProcessCommand          `json:"command"`
	Plugin          plugin.Identity         `json:"plugin"`
	Artifact        plugin.Artifact         `json:"artifact,omitempty"`
	Permissions     []permission.Spec       `json:"permissionLease,omitempty"`
	ResourceLease   *resource.Lease         `json:"resourceLease,omitempty"`
	LeaseController ResourceLeaseController `json:"-"`
	LeasePolicy     resource.LeasePolicy    `json:"leasePolicy,omitempty"`
	Profile         SandboxProfile          `json:"profile"`
}

type SandboxLauncher interface {
	Invoke(context.Context, SandboxSpec, Request) (Response, error)
}

type ArtifactVerification struct {
	VerifierID        string `json:"verifierId"`
	DigestVerified    bool   `json:"digestVerified"`
	SignatureVerified bool   `json:"signatureVerified"`
	ProvenanceChecked bool   `json:"provenanceChecked"`
}

type ArtifactVerifier interface {
	Verify(context.Context, plugin.Identity, plugin.Artifact) (ArtifactVerification, error)
}

type InvocationProvenance struct {
	RequestID                 string                       `json:"requestId"`
	Environment               core.CanonicalID             `json:"environment"`
	Plugin                    plugin.Identity              `json:"plugin"`
	CapabilityKind            plugin.CapabilityKind        `json:"capabilityKind"`
	CapabilityID              core.CanonicalID             `json:"capabilityId"`
	DescriptorFingerprint     string                       `json:"descriptorFingerprint"`
	BuildPlanFingerprint      string                       `json:"buildPlanFingerprint,omitempty"`
	RealizationDigest         string                       `json:"realizationDigest,omitempty"`
	PluginLockFingerprint     string                       `json:"pluginLockFingerprint"`
	Artifact                  plugin.Artifact              `json:"artifact,omitempty"`
	Verification              *ArtifactVerification        `json:"verification,omitempty"`
	Boundary                  Boundary                     `json:"boundary"`
	Sandboxed                 bool                         `json:"sandboxed"`
	Permissions               []permission.Spec            `json:"permissionLease,omitempty"`
	ResourceLeaseID           string                       `json:"resourceLeaseId,omitempty"`
	ResourceLeaseExpiresAt    string                       `json:"resourceLeaseExpiresAt,omitempty"`
	Resources                 []resource.Record            `json:"resources,omitempty"`
	ResourceLeaseReleased     bool                         `json:"resourceLeaseReleased,omitempty"`
	ResourceLeaseRevision     int                          `json:"resourceLeaseRevision,omitempty"`
	ResourceLeaseFencingToken uint64                       `json:"resourceLeaseFencingToken,omitempty"`
	ResourceLeaseRenewals     []resource.RenewalRecord     `json:"resourceLeaseRenewals,omitempty"`
	ResourceLeaseRefreshes    []resource.RefreshRecord     `json:"resourceLeaseRefreshes,omitempty"`
	ResourceLeaseSupervision  []resource.SupervisionRecord `json:"resourceLeaseSupervision,omitempty"`
	ResourceLeaseOwnership    []resource.OwnershipRecord   `json:"resourceLeaseOwnership,omitempty"`
}

type Invocation struct {
	Response   Response             `json:"response"`
	Provenance InvocationProvenance `json:"provenance"`
}

type Registry struct {
	Builtins               map[string]Builtin
	BuiltinRegistrations   map[string]BuiltinRegistration
	Processes              ProcessLauncher
	Sandboxes              SandboxLauncher
	Verifier               ArtifactVerifier
	Resources              resource.Broker
	SupervisorOwnerships   resource.SupervisorOwnershipStore
	SupervisorOwnerID      string
	SupervisorOwnershipTTL time.Duration
}

func identityKey(i plugin.Identity) string { return string(i.ID) + "@" + i.Version }


func (r Registry) resolveBuiltinRegistration(pr PluginRuntime) (Builtin, string, error) {
	key := identityKey(pr.Plugin)
	if pr.BuildPlanFingerprint == "" {
		impl, ok := r.Builtins[key]
		if !ok {
			return nil, "", fmt.Errorf("builtin plugin %s@%s is locked but not registered", pr.Plugin.ID, pr.Plugin.Version)
		}
		return impl, "", nil
	}
	if pr.Artifact.Digest == "" {
		return nil, "", fmt.Errorf("build-bound builtin plugin %s@%s has no realization digest", pr.Plugin.ID, pr.Plugin.Version)
	}
	if pr.Artifact.SignatureRef != "" || pr.Artifact.Provenance != "" {
		return nil, "", fmt.Errorf("build-bound builtin plugin %s@%s uses unsupported artifact signature/provenance metadata", pr.Plugin.ID, pr.Plugin.Version)
	}
	registration, ok := r.BuiltinRegistrations[key]
	if !ok || registration.Builtin == nil {
		return nil, "", fmt.Errorf("build-bound builtin plugin %s@%s has no accepted realization registration", pr.Plugin.ID, pr.Plugin.Version)
	}
	if registration.BuildPlanFingerprint != pr.BuildPlanFingerprint {
		return nil, "", fmt.Errorf("builtin realization build plan mismatch for %s@%s", pr.Plugin.ID, pr.Plugin.Version)
	}
	if registration.RealizationDigest != pr.Artifact.Digest {
		return nil, "", fmt.Errorf("builtin realization digest mismatch for %s@%s", pr.Plugin.ID, pr.Plugin.Version)
	}
	return registration.Builtin, registration.RealizationDigest, nil
}

func (r Registry) Invoke(ctx context.Context, plan EnvironmentPlan, kind plugin.CapabilityKind, id core.CanonicalID, operation, requestID string, payload any) (Response, error) {
	inv, err := r.InvokeWithProvenance(ctx, plan, kind, id, operation, requestID, payload)
	return inv.Response, err
}

func SandboxProfileForLease(lease *resource.Lease) SandboxProfile { return sandboxProfile(lease) }

func sandboxProfile(lease *resource.Lease) SandboxProfile {
	p := SandboxProfile{DenyNetworkByDefault: true, DenyFilesystemByDefault: true}
	if lease == nil {
		return p
	}
	for _, r := range resource.NormalizeResolved(lease.Resources) {
		switch r.Kind {
		case permission.KindNetwork:
			p.NetworkAllowlist = append(p.NetworkAllowlist, r.NetworkEndpoint)
		case permission.KindFilesystem:
			p.Mounts = append(p.Mounts, SandboxMount{Scope: r.Scope, ResourceID: r.ResourceID, SourceHandle: r.MountSourceHandle, Target: r.MountPath, Access: r.Access})
		case permission.KindSecret:
			p.SecretHandles = append(p.SecretHandles, r.SecretHandle)
			p.SecretInjections = append(p.SecretInjections, SecretInjection{Scope: r.Scope, ResourceID: r.ResourceID, Handle: r.SecretHandle, Delivery: r.SecretDelivery, Target: r.SecretTarget})
		}
	}
	sort.Strings(p.NetworkAllowlist)
	sort.Strings(p.SecretHandles)
	sort.Slice(p.SecretInjections, func(i, j int) bool {
		if p.SecretInjections[i].Scope != p.SecretInjections[j].Scope {
			return p.SecretInjections[i].Scope < p.SecretInjections[j].Scope
		}
		return p.SecretInjections[i].ResourceID < p.SecretInjections[j].ResourceID
	})
	sort.Slice(p.Mounts, func(i, j int) bool {
		if p.Mounts[i].Target != p.Mounts[j].Target {
			return p.Mounts[i].Target < p.Mounts[j].Target
		}
		return p.Mounts[i].ResourceID < p.Mounts[j].ResourceID
	})
	return p
}

func (r Registry) InvokeWithProvenance(ctx context.Context, plan EnvironmentPlan, kind plugin.CapabilityKind, id core.CanonicalID, operation, requestID string, payload any) (inv Invocation, retErr error) {
	cr, ok := plan.Capability(kind, id)
	if !ok {
		return Invocation{}, fmt.Errorf("capability %s %s is not uniquely resolved", kind, id)
	}
	if err := hasRequiredPermissions(cr); err != nil {
		return Invocation{}, err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return Invocation{}, err
	}
	var pr *PluginRuntime
	for i := range plan.Plugins {
		if plan.Plugins[i].Plugin == cr.Plugin {
			x := plan.Plugins[i]
			pr = &x
			break
		}
	}
	if pr == nil || pr.DescriptorFingerprint == "" {
		return Invocation{}, fmt.Errorf("plugin runtime descriptor missing for %s@%s", cr.Plugin.ID, cr.Plugin.Version)
	}

	prov := InvocationProvenance{RequestID: requestID, Environment: plan.Environment, Plugin: cr.Plugin, CapabilityKind: kind, CapabilityID: id, DescriptorFingerprint: pr.DescriptorFingerprint, BuildPlanFingerprint: pr.BuildPlanFingerprint, PluginLockFingerprint: plan.PluginLockFingerprint, Artifact: pr.Artifact, Boundary: cr.Boundary, Permissions: permission.Normalize(cr.Permissions)}

	// Trust and execution-boundary checks happen before a resource lease is
	// acquired. An unverifiable artifact or unavailable required sandbox must
	// never receive even a short-lived secret/network/filesystem lease.
	if pr.Artifact.Digest != "" && !(cr.Boundary == BoundaryInProcess && pr.BuildPlanFingerprint != "") {
		if r.Verifier == nil {
			return Invocation{}, fmt.Errorf("plugin %s@%s declares artifact %s but host has no artifact verifier", cr.Plugin.ID, cr.Plugin.Version, pr.Artifact.Digest)
		}
		verified, err := r.Verifier.Verify(ctx, cr.Plugin, pr.Artifact)
		if err != nil {
			return Invocation{}, fmt.Errorf("plugin artifact verification failed for %s@%s: %w", cr.Plugin.ID, cr.Plugin.Version, err)
		}
		if !verified.DigestVerified {
			return Invocation{}, fmt.Errorf("plugin artifact digest was not verified for %s@%s", cr.Plugin.ID, cr.Plugin.Version)
		}
		if pr.Artifact.SignatureRef != "" && !verified.SignatureVerified {
			return Invocation{}, fmt.Errorf("plugin artifact signature was not verified for %s@%s", cr.Plugin.ID, cr.Plugin.Version)
		}
		if pr.Artifact.Provenance != "" && !verified.ProvenanceChecked {
			return Invocation{}, fmt.Errorf("plugin artifact provenance was not checked for %s@%s", cr.Plugin.ID, cr.Plugin.Version)
		}
		prov.Verification = &verified
	}
	switch cr.Boundary {
	case BoundaryInProcess:
		impl, realizationDigest, err := r.resolveBuiltinRegistration(*pr)
		if err != nil {
			return Invocation{}, err
		}
		if impl.Identity() != cr.Plugin || impl.DescriptorFingerprint() != pr.DescriptorFingerprint {
			return Invocation{}, fmt.Errorf("builtin registry identity/descriptor mismatch for %s@%s", cr.Plugin.ID, cr.Plugin.Version)
		}
		prov.RealizationDigest = realizationDigest
	case BoundaryProcess:
		policy := pr.SandboxPolicy
		if policy == "" {
			policy = plugin.SandboxNone
		}
		if policy == plugin.SandboxRequired && r.Sandboxes == nil {
			return Invocation{}, fmt.Errorf("process plugin %s@%s requires sandbox launcher", cr.Plugin.ID, cr.Plugin.Version)
		}
		if policy != plugin.SandboxRequired && r.Sandboxes == nil && r.Processes == nil {
			return Invocation{}, fmt.Errorf("process plugin %s@%s has no host process or sandbox launcher", cr.Plugin.ID, cr.Plugin.Version)
		}
	default:
		return Invocation{}, fmt.Errorf("unsupported runtime boundary %s", cr.Boundary)
	}

	var lease *resource.Lease
	var leaseControl *leaseController
	if len(cr.Permissions) > 0 {
		if r.Resources == nil {
			return Invocation{}, fmt.Errorf("capability %s %s requires resources but host has no resource broker", kind, id)
		}
		acq := resource.AcquireRequest{Environment: plan.Environment, CapabilityKind: string(kind), CapabilityID: id, RuntimePolicyFingerprint: plan.RuntimePolicyFingerprint, Permissions: permission.Normalize(cr.Permissions), Bindings: resource.NormalizeBindings(cr.ResourceBindings), LeasePolicy: cr.LeasePolicy}
		l, err := r.Resources.Acquire(ctx, acq)
		if err != nil {
			return Invocation{}, fmt.Errorf("resource lease acquisition failed for capability %s: %w", id, err)
		}
		if err := resource.ValidateLease(acq, l); err != nil {
			_ = r.Resources.Release(context.WithoutCancel(ctx), l)
			return Invocation{}, fmt.Errorf("resource broker returned invalid lease for capability %s: %w", id, err)
		}
		leaseControl = newLeaseController(r.Resources, acq, l)
		if r.SupervisorOwnerships != nil {
			if r.SupervisorOwnerID == "" {
				_ = r.Resources.Release(context.WithoutCancel(ctx), l)
				return Invocation{}, fmt.Errorf("supervisor ownership store configured without supervisor owner id")
			}
			ttl := r.SupervisorOwnershipTTL
			if ttl <= 0 {
				ttl = 30 * time.Second
			}
			ownership, takeover, claimErr := r.SupervisorOwnerships.Claim(ctx, l.ID, r.SupervisorOwnerID, ttl)
			if claimErr != nil {
				_ = r.Resources.Release(context.WithoutCancel(ctx), l)
				return Invocation{}, fmt.Errorf("resource lease supervisor ownership claim failed: %w", claimErr)
			}
			fenced, fenceErr := leaseControl.installFence(ctx, ownership.FencingToken)
			if fenceErr != nil {
				_ = r.SupervisorOwnerships.Release(context.WithoutCancel(ctx), ownership)
				_ = r.Resources.Release(context.WithoutCancel(ctx), l)
				return Invocation{}, fmt.Errorf("resource lease supervisor fencing failed: %w", fenceErr)
			}
			l = fenced
			leaseControl.attachOwnership(r.SupervisorOwnerships, ownership, ttl, takeover)
		}
		lease = leaseControl.leasePointer()
		prov.ResourceLeaseID = l.ID
		prov.ResourceLeaseExpiresAt = l.ExpiresAt
		prov.ResourceLeaseRevision = normalizedLeaseRevision(l)
		prov.ResourceLeaseFencingToken = l.FencingToken
		prov.Resources = resource.Records(l.Resources)
		defer func() {
			// Cleanup must outlive cancellation of the invocation context; otherwise
			// a canceled plugin call could strand a credential/network/mount lease.
			current := leaseControl.Current()
			prov.ResourceLeaseID = current.ID
			prov.ResourceLeaseExpiresAt = current.ExpiresAt
			prov.ResourceLeaseRevision = normalizedLeaseRevision(current)
			prov.ResourceLeaseFencingToken = current.FencingToken
			prov.ResourceLeaseRenewals = leaseControl.Renewals()
			prov.ResourceLeaseRefreshes = leaseControl.Refreshes()
			prov.Resources = resource.Records(current.Resources)
			leaseControl.recordSupervision(resource.SupervisionRevokePending, "resource lease cleanup started")
			releaseCtx := context.WithoutCancel(ctx)
			var cancel context.CancelFunc
			if d := cr.LeasePolicy.RevokeTimeoutDuration(); d > 0 {
				releaseCtx, cancel = context.WithTimeout(releaseCtx, d)
				defer cancel()
			}
			err := r.Resources.Release(releaseCtx, current)
			if err == nil {
				prov.ResourceLeaseReleased = true
				leaseControl.recordSupervision(resource.SupervisionRevoked, "resource lease revoked")
			} else {
				leaseControl.recordSupervision(resource.SupervisionRevokeFailed, err.Error())
				retErr = errors.Join(retErr, fmt.Errorf("resource lease %s release failed: %w", current.ID, err))
			}
			if ownershipErr := leaseControl.ReleaseOwnership(context.WithoutCancel(ctx)); ownershipErr != nil {
				retErr = errors.Join(retErr, fmt.Errorf("resource lease supervisor ownership release failed: %w", ownershipErr))
			}
			prov.ResourceLeaseSupervision = leaseControl.Supervision()
			prov.ResourceLeaseOwnership = leaseControl.OwnershipRecords()
			inv.Provenance = prov
		}()
	}

	req := Request{ProtocolVersion: ProtocolVersion, RequestID: requestID, Plugin: cr.Plugin, CapabilityKind: kind, CapabilityID: id, DescriptorFingerprint: pr.DescriptorFingerprint, PluginLockFingerprint: plan.PluginLockFingerprint, Permissions: prov.Permissions, ResourceLease: lease, LeaseController: leaseControl, Operation: operation, Payload: b}
	var resp Response
	switch cr.Boundary {
	case BoundaryInProcess:
		impl, _, resolveErr := r.resolveBuiltinRegistration(*pr)
		if resolveErr != nil {
			return Invocation{}, resolveErr
		}
		if impl.Identity() != cr.Plugin {
			return Invocation{}, fmt.Errorf("builtin registry identity mismatch for %s@%s", cr.Plugin.ID, cr.Plugin.Version)
		}
		if impl.DescriptorFingerprint() != pr.DescriptorFingerprint {
			return Invocation{}, fmt.Errorf("builtin registry descriptor fingerprint mismatch for %s@%s", cr.Plugin.ID, cr.Plugin.Version)
		}
		resp, err = impl.Invoke(ctx, req)
	case BoundaryProcess:
		policy := pr.SandboxPolicy
		if policy == "" {
			policy = plugin.SandboxNone
		}
		processCommand := ProcessCommand{Executable: pr.Command, Args: append([]string(nil), pr.Args...)}
		if r.Sandboxes != nil && (policy == plugin.SandboxRequired || policy == plugin.SandboxPreferred) {
			prov.Sandboxed = true
			resp, err = r.Sandboxes.Invoke(ctx, SandboxSpec{Command: processCommand, Plugin: cr.Plugin, Artifact: pr.Artifact, Permissions: req.Permissions, ResourceLease: lease, LeaseController: leaseControl, LeasePolicy: cr.LeasePolicy, Profile: sandboxProfile(lease)}, req)
		} else {
			if policy == plugin.SandboxRequired {
				return Invocation{}, fmt.Errorf("process plugin %s@%s requires sandbox launcher", cr.Plugin.ID, cr.Plugin.Version)
			}
			if r.Processes == nil {
				return Invocation{}, fmt.Errorf("process plugin %s@%s has no host process launcher", cr.Plugin.ID, cr.Plugin.Version)
			}
			resp, err = r.Processes.Invoke(ctx, processCommand, req)
		}
	default:
		return Invocation{}, fmt.Errorf("unsupported runtime boundary %s", cr.Boundary)
	}
	if leaseControl != nil {
		if ferr := leaseControl.FatalError(); ferr != nil {
			err = errors.Join(err, ferr)
		}
	}
	if err != nil {
		return Invocation{Provenance: prov}, err
	}
	if resp.ProtocolVersion == 0 {
		resp.ProtocolVersion = ProtocolVersion
	}
	if resp.RequestID == "" {
		resp.RequestID = requestID
	}
	if resp.Plugin.ID == "" {
		resp.Plugin = cr.Plugin
	}
	if resp.CapabilityID == "" {
		resp.CapabilityID = id
	}
	if resp.DescriptorFingerprint == "" {
		resp.DescriptorFingerprint = pr.DescriptorFingerprint
	}
	inv = Invocation{Response: resp, Provenance: prov}
	if resp.ProtocolVersion != ProtocolVersion || resp.RequestID != requestID || resp.Plugin != cr.Plugin || resp.CapabilityID != id || resp.DescriptorFingerprint != pr.DescriptorFingerprint {
		return inv, fmt.Errorf("plugin runtime response identity mismatch for capability %s", id)
	}
	if resp.Status == ResponseError {
		return inv, fmt.Errorf("plugin capability %s failed: %s", id, resp.Error)
	}
	if resp.Status != ResponseOK {
		return inv, fmt.Errorf("plugin capability %s returned invalid status %s", id, resp.Status)
	}
	return inv, nil
}

// ValidateDeploymentInvocation verifies provenance attached to a deployment
// executor result against the exact locked runtime plan. It is intentionally
// separate from executor acknowledgement validation: executor ID/action/key
// prove routing identity, while this proves the Plugin implementation/trust
// context that produced the acknowledgement.
func ValidateDeploymentInvocation(plan EnvironmentPlan, capabilityID core.CanonicalID, pi *executor.PluginInvocation) error {
	if pi == nil {
		return nil // legacy/non-runtime acknowledgement; caller may enforce presence by policy later.
	}
	cr, ok := plan.Capability(plugin.CapabilityDeploymentExecutor, capabilityID)
	if !ok {
		return fmt.Errorf("deployment executor capability %s is not uniquely resolved", capabilityID)
	}
	if err := hasRequiredPermissions(cr); err != nil {
		return err
	}
	var pr *PluginRuntime
	for i := range plan.Plugins {
		if plan.Plugins[i].Plugin == cr.Plugin {
			x := plan.Plugins[i]
			pr = &x
			break
		}
	}
	if pr == nil {
		return fmt.Errorf("plugin runtime missing for deployment executor %s", capabilityID)
	}
	if pi.PluginID != cr.Plugin.ID || pi.PluginVersion != cr.Plugin.Version {
		return fmt.Errorf("deployment attempt plugin identity %s@%s does not match locked %s@%s", pi.PluginID, pi.PluginVersion, cr.Plugin.ID, cr.Plugin.Version)
	}
	if pi.DescriptorFingerprint != pr.DescriptorFingerprint {
		return fmt.Errorf("deployment attempt descriptor fingerprint does not match locked plugin %s@%s", cr.Plugin.ID, cr.Plugin.Version)
	}
	if pi.BuildPlanFingerprint != pr.BuildPlanFingerprint {
		return fmt.Errorf("deployment attempt build plan fingerprint does not match locked plugin %s@%s", cr.Plugin.ID, cr.Plugin.Version)
	}
	if cr.Boundary == BoundaryInProcess && pr.BuildPlanFingerprint != "" {
		if pi.RealizationDigest == "" || pi.RealizationDigest != pr.Artifact.Digest {
			return fmt.Errorf("deployment attempt builtin realization digest does not match locked plugin %s@%s", cr.Plugin.ID, cr.Plugin.Version)
		}
	} else if pi.RealizationDigest != "" {
		return fmt.Errorf("deployment attempt reports builtin realization digest for non-build-bound builtin plugin %s@%s", cr.Plugin.ID, cr.Plugin.Version)
	}
	if pi.PluginLockFingerprint != plan.PluginLockFingerprint {
		return fmt.Errorf("deployment attempt plugin lock fingerprint does not match environment lock")
	}
	if pi.RuntimePolicyFingerprint != plan.RuntimePolicyFingerprint {
		return fmt.Errorf("deployment attempt runtime policy fingerprint does not match environment runtime policy")
	}
	if pi.ArtifactDigest != pr.Artifact.Digest || pi.ArtifactSignatureRef != pr.Artifact.SignatureRef || pi.ArtifactProvenance != pr.Artifact.Provenance {
		return fmt.Errorf("deployment attempt artifact provenance does not match locked plugin %s@%s", cr.Plugin.ID, cr.Plugin.Version)
	}
	if pi.Boundary != string(cr.Boundary) {
		return fmt.Errorf("deployment attempt runtime boundary %s does not match planned %s", pi.Boundary, cr.Boundary)
	}
	if pr.SandboxPolicy == plugin.SandboxRequired && !pi.Sandboxed {
		return fmt.Errorf("deployment attempt for %s requires sandbox but provenance reports unsandboxed execution", capabilityID)
	}
	wantPerms := permission.Normalize(cr.Permissions)
	gotPerms := permission.Normalize(pi.PermissionLease)
	if len(wantPerms) != len(gotPerms) {
		return fmt.Errorf("deployment attempt permission lease differs from environment grant")
	}
	for i := range wantPerms {
		if wantPerms[i].Key() != gotPerms[i].Key() {
			return fmt.Errorf("deployment attempt permission lease differs from environment grant")
		}
	}
	if len(wantPerms) > 0 {
		if pi.ResourceLeaseID == "" {
			return fmt.Errorf("deployment attempt with permissions has no resource lease id")
		}
		if !pi.ResourceLeaseReleased {
			return fmt.Errorf("deployment attempt resource lease was not released")
		}
		revision := pi.ResourceLeaseRevision
		if revision == 0 {
			revision = 1
		}
		if revision < 1 {
			return fmt.Errorf("deployment attempt resource lease revision is invalid")
		}
		if len(pi.ResourceLeaseRenewals)+len(pi.ResourceLeaseRefreshes) != revision-1 {
			return fmt.Errorf("deployment attempt resource lease history does not match revision %d", revision)
		}
		var revisions []int
		for _, rr := range pi.ResourceLeaseRenewals {
			revisions = append(revisions, rr.Revision)
		}
		for _, rr := range pi.ResourceLeaseRefreshes {
			revisions = append(revisions, rr.Revision)
		}
		sort.Ints(revisions)
		for i, rev := range revisions {
			if rev != i+2 {
				return fmt.Errorf("deployment attempt resource lease history is not sequential")
			}
		}
		if len(pi.ResourceLeaseSupervision) > 0 {
			final := pi.ResourceLeaseSupervision[len(pi.ResourceLeaseSupervision)-1]
			if pi.ResourceLeaseReleased && final.State != resource.SupervisionRevoked {
				return fmt.Errorf("deployment attempt released resource lease without final revoked supervision state")
			}
			for _, sr := range pi.ResourceLeaseSupervision {
				switch sr.State {
				case resource.SupervisionRenewalFailed, resource.SupervisionRenewalDeadlineMissed, resource.SupervisionRefreshFailed, resource.SupervisionRevokeFailed:
					return fmt.Errorf("deployment attempt contains failed resource supervision state %s", sr.State)
				}
			}
		}
		bindings, err := resource.MatchBindings(wantPerms, cr.ResourceBindings)
		if err != nil {
			return err
		}
		wantRecords := make([]resource.Record, 0, len(wantPerms))
		for _, perm := range wantPerms {
			for _, b := range bindings {
				if b.Kind == perm.Kind && b.Scope == perm.Scope {
					wantRecords = append(wantRecords, resource.Record{Kind: perm.Kind, Access: perm.Access, Scope: perm.Scope, ResourceID: b.ResourceID, SecretDelivery: b.SecretDelivery, SecretTarget: b.SecretTarget})
					break
				}
			}
		}
		gotRecords := append([]resource.Record(nil), pi.Resources...)
		sort.Slice(wantRecords, func(i, j int) bool {
			return wantRecords[i].Kind+permission.Kind(wantRecords[i].Scope) < wantRecords[j].Kind+permission.Kind(wantRecords[j].Scope)
		})
		sort.Slice(gotRecords, func(i, j int) bool {
			return gotRecords[i].Kind+permission.Kind(gotRecords[i].Scope) < gotRecords[j].Kind+permission.Kind(gotRecords[j].Scope)
		})
		if len(wantRecords) != len(gotRecords) {
			return fmt.Errorf("deployment attempt resource lease differs from environment bindings")
		}
		for i := range wantRecords {
			if wantRecords[i] != gotRecords[i] {
				return fmt.Errorf("deployment attempt resource lease differs from environment bindings")
			}
		}
	} else if pi.ResourceLeaseID != "" || len(pi.Resources) > 0 {
		return fmt.Errorf("deployment attempt reports resource lease for capability without permissions")
	}
	if len(pi.ResourceLeaseOwnership) > 0 {
		if pi.ResourceLeaseFencingToken == 0 {
			return fmt.Errorf("deployment attempt has supervisor ownership history without fencing token")
		}
		for _, or := range pi.ResourceLeaseOwnership {
			if or.FencingToken != pi.ResourceLeaseFencingToken {
				return fmt.Errorf("deployment attempt supervisor ownership history has inconsistent fencing token")
			}
			if or.State == resource.OwnershipLost {
				return fmt.Errorf("deployment attempt lost supervisor ownership during invocation")
			}
		}
		if final := pi.ResourceLeaseOwnership[len(pi.ResourceLeaseOwnership)-1]; final.State != resource.OwnershipReleased {
			return fmt.Errorf("deployment attempt supervisor ownership was not released")
		}
	}
	if pr.Artifact.Digest != "" && !(cr.Boundary == BoundaryInProcess && pr.BuildPlanFingerprint != "") && !pi.DigestVerified {
		return fmt.Errorf("deployment attempt artifact digest was not verified")
	}
	if pr.Artifact.SignatureRef != "" && !pi.SignatureVerified {
		return fmt.Errorf("deployment attempt artifact signature was not verified")
	}
	if pr.Artifact.Provenance != "" && !pi.ProvenanceChecked {
		return fmt.Errorf("deployment attempt artifact provenance was not checked")
	}
	return nil
}
