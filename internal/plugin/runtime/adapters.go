package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/evidence"
	"tei/internal/deployment/executor"
	"tei/internal/deployment/observation"
	"tei/internal/plugin"
)

type DeploymentExecutor struct {
	Plan         EnvironmentPlan
	Registry     Registry
	CapabilityID core.CanonicalID
}

func (a DeploymentExecutor) ID() string { return string(a.CapabilityID) }
func (a DeploymentExecutor) Supports(c executor.Command) bool {
	cr, ok := a.Plan.Capability(plugin.CapabilityDeploymentExecutor, a.CapabilityID)
	if !ok {
		return false
	}
	return cr.Capability.SupportsAction(deployment.ActionKind(c.Kind))
}
func (a DeploymentExecutor) Execute(ctx context.Context, c executor.Command) (executor.Result, error) {
	if !a.Supports(c) {
		return executor.Result{}, fmt.Errorf("capability %s does not support action %s", a.CapabilityID, c.Kind)
	}
	inv, err := a.Registry.InvokeWithProvenance(ctx, a.Plan, plugin.CapabilityDeploymentExecutor, a.CapabilityID, "execute", c.ActionID, c)
	if err != nil {
		return executor.Result{}, err
	}
	var out executor.Result
	if err := json.Unmarshal(inv.Response.Payload, &out); err != nil {
		return out, err
	}
	out.ExecutorID = string(a.CapabilityID)
	if out.ActionID == "" {
		out.ActionID = c.ActionID
	}
	if out.IdempotencyKey == "" {
		out.IdempotencyKey = c.IdempotencyKey
	}
	pv := inv.Provenance
	pi := &executor.PluginInvocation{PluginID: pv.Plugin.ID, PluginVersion: pv.Plugin.Version, DescriptorFingerprint: pv.DescriptorFingerprint, BuildPlanFingerprint: pv.BuildPlanFingerprint, RealizationDigest: pv.RealizationDigest, PluginLockFingerprint: pv.PluginLockFingerprint, RuntimePolicyFingerprint: a.Plan.RuntimePolicyFingerprint, ArtifactDigest: pv.Artifact.Digest, ArtifactSignatureRef: pv.Artifact.SignatureRef, ArtifactProvenance: pv.Artifact.Provenance, Boundary: string(pv.Boundary), Sandboxed: pv.Sandboxed, PermissionLease: pv.Permissions, ResourceLeaseID: pv.ResourceLeaseID, ResourceLeaseExpiresAt: pv.ResourceLeaseExpiresAt, Resources: pv.Resources, ResourceLeaseReleased: pv.ResourceLeaseReleased, ResourceLeaseRevision: pv.ResourceLeaseRevision, ResourceLeaseFencingToken: pv.ResourceLeaseFencingToken, ResourceLeaseRenewals: pv.ResourceLeaseRenewals, ResourceLeaseRefreshes: pv.ResourceLeaseRefreshes, ResourceLeaseSupervision: pv.ResourceLeaseSupervision, ResourceLeaseOwnership: pv.ResourceLeaseOwnership}
	if pv.Verification != nil {
		pi.VerifierID = pv.Verification.VerifierID
		pi.DigestVerified = pv.Verification.DigestVerified
		pi.SignatureVerified = pv.Verification.SignatureVerified
		pi.ProvenanceChecked = pv.Verification.ProvenanceChecked
	}
	out.PluginInvocation = pi
	return out, nil
}

type ObservationProvider struct {
	Plan         EnvironmentPlan
	Registry     Registry
	CapabilityID core.CanonicalID
}

func (a ObservationProvider) ID() string { return string(a.CapabilityID) }
func (a ObservationProvider) Supports(k deployment.ObservationKind) bool {
	cr, ok := a.Plan.Capability(plugin.CapabilityObservationProvider, a.CapabilityID)
	return ok && cr.Capability.SupportsObservation(k)
}
func (a ObservationProvider) Observe(ctx context.Context, req observation.Request) (observation.Fragment, error) {
	if !a.Supports(req.Kind) {
		return observation.Fragment{}, fmt.Errorf("capability %s does not support observation %s", a.CapabilityID, req.Kind)
	}
	resp, err := a.Registry.Invoke(ctx, a.Plan, plugin.CapabilityObservationProvider, a.CapabilityID, "observe", string(req.Kind), req)
	if err != nil {
		return observation.Fragment{}, err
	}
	var out observation.Fragment
	if err := json.Unmarshal(resp.Payload, &out); err != nil {
		return out, err
	}
	out.ProviderID = a.CapabilityID
	return out, nil
}

type EvidenceProvider struct {
	Plan         EnvironmentPlan
	Registry     Registry
	CapabilityID core.CanonicalID
}

func (a EvidenceProvider) ID() string { return string(a.CapabilityID) }
func (a EvidenceProvider) Supports(k deployment.EvidenceKind) bool {
	cr, ok := a.Plan.Capability(plugin.CapabilityEvidenceProvider, a.CapabilityID)
	return ok && cr.Capability.SupportsEvidence(k)
}
func (a EvidenceProvider) Collect(ctx context.Context, req evidence.Request) (durable.Evidence, error) {
	if !a.Supports(req.Kind) {
		return durable.Evidence{}, fmt.Errorf("capability %s does not support evidence %s", a.CapabilityID, req.Kind)
	}
	resp, err := a.Registry.Invoke(ctx, a.Plan, plugin.CapabilityEvidenceProvider, a.CapabilityID, "collect", req.ActionID, req)
	if err != nil {
		return durable.Evidence{}, err
	}
	var out durable.Evidence
	if err := json.Unmarshal(resp.Payload, &out); err != nil {
		return out, err
	}
	if out.ActionID == "" {
		out.ActionID = req.ActionID
	}
	if out.Kind == "" {
		out.Kind = req.Kind
	}
	return out, nil
}
