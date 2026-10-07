package executor

import (
	"context"

	"tei/internal/core"
	"tei/internal/permission"
	"tei/internal/resource"
)

// Command is the provider-neutral intent handed to an external deployment
// executor. A successful Execute call means only that the executor accepted or
// processed the intent; durable completion is still established by TEI through
// observed-state reconciliation.
type Command struct {
	ID                 string           `json:"id"`
	ActionID           string           `json:"actionId"`
	IdempotencyKey     string           `json:"idempotencyKey"`
	Kind               string           `json:"kind"`
	Application        core.CanonicalID `json:"application,omitempty"`
	Version            string           `json:"version,omitempty"`
	Database           core.CanonicalID `json:"database,omitempty"`
	TrafficBasisPoints int              `json:"trafficBasisPoints,omitempty"`
}

type Outcome string

const (
	OutcomeAccepted Outcome = "accepted"
	OutcomeRejected Outcome = "rejected"
	OutcomeUnknown  Outcome = "unknown"
)

// Result is an executor-side acknowledgement. It never constitutes proof that
// the intended external state exists; that proof comes from reconciliation.
type PluginInvocation struct {
	PluginID                  core.CanonicalID             `json:"pluginId"`
	PluginVersion             string                       `json:"pluginVersion"`
	DescriptorFingerprint     string                       `json:"descriptorFingerprint"`
	BuildPlanFingerprint      string                       `json:"buildPlanFingerprint,omitempty"`
	RealizationDigest         string                       `json:"realizationDigest,omitempty"`
	PluginLockFingerprint     string                       `json:"pluginLockFingerprint"`
	RuntimePolicyFingerprint  string                       `json:"runtimePolicyFingerprint,omitempty"`
	ArtifactDigest            string                       `json:"artifactDigest,omitempty"`
	ArtifactSignatureRef      string                       `json:"artifactSignatureRef,omitempty"`
	ArtifactProvenance        string                       `json:"artifactProvenance,omitempty"`
	Boundary                  string                       `json:"boundary"`
	Sandboxed                 bool                         `json:"sandboxed"`
	PermissionLease           []permission.Spec            `json:"permissionLease,omitempty"`
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
	VerifierID                string                       `json:"verifierId,omitempty"`
	DigestVerified            bool                         `json:"digestVerified,omitempty"`
	SignatureVerified         bool                         `json:"signatureVerified,omitempty"`
	ProvenanceChecked         bool                         `json:"provenanceChecked,omitempty"`
}

type Result struct {
	ExecutorID       string            `json:"executorId"`
	AttemptID        string            `json:"attemptId"`
	ActionID         string            `json:"actionId"`
	IdempotencyKey   string            `json:"idempotencyKey"`
	Outcome          Outcome           `json:"outcome"`
	Retryable        bool              `json:"retryable,omitempty"`
	Message          string            `json:"message,omitempty"`
	ExternalRef      string            `json:"externalRef,omitempty"`
	ObservedAt       string            `json:"observedAt,omitempty"`
	PluginInvocation *PluginInvocation `json:"pluginInvocation,omitempty"`
}

// Executor is the narrow SPI used by TEI deployment execution. Concrete
// Kubernetes, cloud, VM, traffic-manager, or migration-tool adapters live
// outside the durable-run model.
type Executor interface {
	ID() string
	Supports(Command) bool
	Execute(context.Context, Command) (Result, error)
}
