package plugin

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/permission"
)

type CapabilityKind string

const (
	CapabilityDeploymentExecutor    CapabilityKind = "deployment-executor"
	CapabilityObservationProvider   CapabilityKind = "observation-provider"
	CapabilityEvidenceProvider      CapabilityKind = "evidence-provider"
	CapabilityDatabaseProvider      CapabilityKind = "database-provider"
	CapabilityContractQueryExecutor CapabilityKind = "contract-query-executor"
	CapabilityMigrationProvider     CapabilityKind = "migration-provider"
	CapabilityEditorIntelligence    CapabilityKind = "editor-intelligence"
)

func (k CapabilityKind) Valid() bool {
	switch k {
	case CapabilityDeploymentExecutor, CapabilityObservationProvider, CapabilityEvidenceProvider,
		CapabilityDatabaseProvider, CapabilityContractQueryExecutor, CapabilityMigrationProvider, CapabilityEditorIntelligence:
		return true
	default:
		return false
	}
}

type RuntimeKind string

const (
	RuntimeBuiltin RuntimeKind = "builtin"
	RuntimeProcess RuntimeKind = "process"
)

func (k RuntimeKind) Valid() bool { return k == RuntimeBuiltin || k == RuntimeProcess }

type SandboxPolicy string

const (
	SandboxNone      SandboxPolicy = "none"
	SandboxPreferred SandboxPolicy = "preferred"
	SandboxRequired  SandboxPolicy = "required"
)

func (p SandboxPolicy) Valid() bool {
	return p == "" || p == SandboxNone || p == SandboxPreferred || p == SandboxRequired
}

type Runtime struct {
	Kind    RuntimeKind   `json:"kind"`
	Command string        `json:"command,omitempty"`
	Args    []string      `json:"args,omitempty"`
	Sandbox SandboxPolicy `json:"sandbox,omitempty"`
}

type Artifact struct {
	Digest       string `json:"digest,omitempty"`
	SignatureRef string `json:"signatureRef,omitempty"`
	Provenance   string `json:"provenance,omitempty"`
}

func (a Artifact) Validate() error {
	if a.Digest == "" {
		if a.SignatureRef != "" || a.Provenance != "" {
			return fmt.Errorf("artifact signature/provenance requires a digest")
		}
		return nil
	}
	if !strings.HasPrefix(a.Digest, "sha256:") {
		return fmt.Errorf("artifact digest must use sha256:<64-hex>")
	}
	hexPart := strings.TrimPrefix(a.Digest, "sha256:")
	if len(hexPart) != 64 {
		return fmt.Errorf("artifact sha256 digest must contain 64 hex characters")
	}
	for _, r := range hexPart {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return fmt.Errorf("artifact sha256 digest contains non-hex character")
		}
	}
	return nil
}

type Capability struct {
	Kind              CapabilityKind                 `json:"kind"`
	ID                core.CanonicalID               `json:"id"`
	Actions           []deployment.ActionKind        `json:"actions,omitempty"`
	Observations      []deployment.ObservationKind   `json:"observations,omitempty"`
	Evidence          []deployment.EvidenceKind      `json:"evidence,omitempty"`
	DatabaseProviders []environment.DatabaseProvider `json:"databaseProviders,omitempty"`
	MigrationTools    []migration.Tool               `json:"migrationTools,omitempty"`
	Features          []environment.Feature          `json:"features,omitempty"`
	Permissions       []permission.Spec              `json:"permissions,omitempty"`
}

type Requirement struct {
	Plugin          core.CanonicalID `json:"plugin"`
	CompatibleMajor int              `json:"compatibleMajor"`
}

type ExternalRequirement struct {
	Scope string `json:"scope"`
	Kind  string `json:"kind"`
	Name  string `json:"name"`
}

type Definition struct {
	ID       core.CanonicalID      `json:"id"`
	Version  string                `json:"version"`
	Runtime  Runtime               `json:"runtime"`
	Artifact Artifact              `json:"artifact,omitempty"`
	Provides []Capability          `json:"provides,omitempty"`
	Requires []Requirement         `json:"requires,omitempty"`
	External []ExternalRequirement `json:"externalRequirements,omitempty"`
}

type Bundle struct {
	ID      core.CanonicalID `json:"id"`
	Version string           `json:"version"`
	Plugins []Requirement    `json:"plugins"`
}

type Identity struct {
	ID      core.CanonicalID `json:"id"`
	Version string           `json:"version"`
}

type ResolvedSet struct {
	Environment  core.CanonicalID     `json:"environment"`
	Plugins      []Identity           `json:"plugins"`
	Capabilities []ResolvedCapability `json:"capabilities,omitempty"`
}

type ResolvedCapability struct {
	Plugin     Identity   `json:"plugin"`
	Capability Capability `json:"capability"`
}

func ParseVersion(v string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("version %q must be major.minor.patch", v)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, fmt.Errorf("version %q contains invalid numeric component", v)
		}
		out[i] = n
	}
	return out, nil
}

func CompatibleMajor(v string, major int) bool {
	p, err := ParseVersion(v)
	return err == nil && p[0] == major
}

func CompareVersion(a, b string) int {
	av, _ := ParseVersion(a)
	bv, _ := ParseVersion(b)
	for i := 0; i < 3; i++ {
		if av[i] < bv[i] {
			return -1
		}
		if av[i] > bv[i] {
			return 1
		}
	}
	return 0
}

func NormalizeDefinition(d Definition) Definition {
	for i := range d.Provides {
		c := &d.Provides[i]
		sort.Slice(c.Actions, func(a, b int) bool { return c.Actions[a] < c.Actions[b] })
		sort.Slice(c.Observations, func(a, b int) bool { return c.Observations[a] < c.Observations[b] })
		sort.Slice(c.Evidence, func(a, b int) bool { return c.Evidence[a] < c.Evidence[b] })
		sort.Slice(c.DatabaseProviders, func(a, b int) bool { return c.DatabaseProviders[a] < c.DatabaseProviders[b] })
		sort.Slice(c.MigrationTools, func(a, b int) bool { return c.MigrationTools[a] < c.MigrationTools[b] })
		sort.Slice(c.Features, func(a, b int) bool { return c.Features[a] < c.Features[b] })
		c.Permissions = permission.Normalize(c.Permissions)
	}
	sort.Slice(d.Provides, func(i, j int) bool {
		if d.Provides[i].Kind != d.Provides[j].Kind {
			return d.Provides[i].Kind < d.Provides[j].Kind
		}
		return d.Provides[i].ID < d.Provides[j].ID
	})
	sort.Slice(d.Requires, func(i, j int) bool { return d.Requires[i].Plugin < d.Requires[j].Plugin })
	sort.Slice(d.External, func(i, j int) bool {
		if d.External[i].Scope != d.External[j].Scope {
			return d.External[i].Scope < d.External[j].Scope
		}
		if d.External[i].Kind != d.External[j].Kind {
			return d.External[i].Kind < d.External[j].Kind
		}
		return d.External[i].Name < d.External[j].Name
	})
	return d
}

func (c Capability) SupportsAction(k deployment.ActionKind) bool {
	for _, v := range c.Actions {
		if v == k {
			return true
		}
	}
	return false
}
func (c Capability) SupportsObservation(k deployment.ObservationKind) bool {
	for _, v := range c.Observations {
		if v == k {
			return true
		}
	}
	return false
}
func (c Capability) SupportsEvidence(k deployment.EvidenceKind) bool {
	for _, v := range c.Evidence {
		if v == k {
			return true
		}
	}
	return false
}
func (c Capability) SupportsDatabaseProvider(p environment.DatabaseProvider) bool {
	for _, v := range c.DatabaseProviders {
		if v == p {
			return true
		}
	}
	return false
}
func (c Capability) SupportsMigrationTool(t migration.Tool) bool {
	for _, v := range c.MigrationTools {
		if v == t {
			return true
		}
	}
	return false
}
func (c Capability) SupportsFeature(f environment.Feature) bool {
	for _, v := range c.Features {
		if v == f {
			return true
		}
	}
	return false
}
