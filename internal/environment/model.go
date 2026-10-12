package environment

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/permission"
	"tei/internal/resource"
)

type DatabaseProvider string

const (
	ProviderGeneric    DatabaseProvider = "generic"
	ProviderPostgres   DatabaseProvider = "postgres"
	ProviderOracle     DatabaseProvider = "oracle"
	ProviderMySQL      DatabaseProvider = "mysql"
	ProviderMariaDB    DatabaseProvider = "mariadb"
	ProviderSQLServer  DatabaseProvider = "sqlserver"
	ProviderSQLite     DatabaseProvider = "sqlite"
	ProviderDB2        DatabaseProvider = "db2"
	ProviderClickHouse DatabaseProvider = "clickhouse"
)

func (p DatabaseProvider) Known() bool {
	switch p {
	case ProviderGeneric, ProviderPostgres, ProviderOracle, ProviderMySQL, ProviderMariaDB,
		ProviderSQLServer, ProviderSQLite, ProviderDB2, ProviderClickHouse:
		return true
	default:
		return false
	}
}

type Feature core.CanonicalID

const (
	FeatureSQLAnalysis       Feature = "editor.sql_analysis"
	FeatureDatabaseWorkbench Feature = "editor.database_workbench"
)

func (f Feature) Known() bool {
	return f == FeatureSQLAnalysis || f == FeatureDatabaseWorkbench
}

type DatabaseInstance struct {
	ID         core.CanonicalID `json:"id"`
	Definition core.CanonicalID `json:"definition"`
	Provider   DatabaseProvider `json:"provider"`
}

type ApplicationDatabaseBinding struct {
	Application core.CanonicalID `json:"application"`
	Alias       core.LocalName   `json:"alias"`
	Instance    core.CanonicalID `json:"instance"`
}

// DeploymentExecutorBinding selects one executor deterministically for one
// provider-neutral action kind. No runtime ranking or technology inference is
// performed by TEI.
type ExecutorScope string

const (
	ExecutorScopeGlobal      ExecutorScope = "global"
	ExecutorScopeApplication ExecutorScope = "application"
	ExecutorScopeDatabase    ExecutorScope = "database"
)

func (s ExecutorScope) Valid() bool {
	return s == ExecutorScopeGlobal || s == ExecutorScopeApplication || s == ExecutorScopeDatabase
}

type DeploymentExecutorBinding struct {
	Action   deployment.ActionKind `json:"action"`
	Scope    ExecutorScope         `json:"scope,omitempty"`
	Target   core.CanonicalID      `json:"target,omitempty"`
	Executor core.CanonicalID      `json:"executor"`
}

type ObservationProviderBinding struct {
	Kind     deployment.ObservationKind `json:"kind"`
	Provider core.CanonicalID           `json:"provider"`
}

type EvidenceProviderBinding struct {
	Kind     deployment.EvidenceKind `json:"kind"`
	Provider core.CanonicalID        `json:"provider"`
}

type PermissionGrant = permission.Grant
type ResourceBinding = resource.Binding

type CapabilityLeasePolicy struct {
	CapabilityKind string               `json:"capabilityKind"`
	CapabilityID   core.CanonicalID     `json:"capabilityId"`
	Policy         resource.LeasePolicy `json:"policy"`
}

type ExtensionRequirement struct {
	ID              core.CanonicalID `json:"id"`
	CompatibleMajor int              `json:"compatibleMajor"`
}

// ValidateID rejects invalid IDs as supplied; ParseCanonicalID alone trims
// whitespace, so equality with the original value is essential.
func (r ExtensionRequirement) ValidateID() error {
	id, err := core.ParseCanonicalID(string(r.ID))
	if err != nil || id != r.ID {
		return fmt.Errorf("invalid canonical id %q", r.ID)
	}
	return nil
}

// Validate checks the local shape of one environment selection, without
// resolving it against a Plugin or Bundle catalog.
func (r ExtensionRequirement) Validate() error {
	if err := r.ValidateID(); err != nil {
		return err
	}
	if r.CompatibleMajor < 1 {
		return fmt.Errorf("compatible major must be >= 1, got %d", r.CompatibleMajor)
	}
	return nil
}

// ValidateExtensionRequirements rejects malformed entries consistently even
// when the declaration order differs. Cross-bundle shared dependencies and
// repeated references are separate concerns.
func ValidateExtensionRequirements(requires []ExtensionRequirement) error {
	ordered := append([]ExtensionRequirement(nil), requires...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].ID != ordered[j].ID {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].CompatibleMajor < ordered[j].CompatibleMajor
	})
	for _, req := range ordered {
		if err := req.Validate(); err != nil {
			return fmt.Errorf("extension requirement %q: %w", req.ID, err)
		}
	}
	return nil
}

type Environment struct {
	ID                   core.CanonicalID             `json:"id"`
	Plugins              []ExtensionRequirement       `json:"plugins,omitempty"`
	Bundles              []ExtensionRequirement       `json:"bundles,omitempty"`
	Features             []Feature                    `json:"features,omitempty"`
	Instances            []DatabaseInstance           `json:"instances,omitempty"`
	Bindings             []ApplicationDatabaseBinding `json:"bindings,omitempty"`
	DeploymentExecutors  []DeploymentExecutorBinding  `json:"deploymentExecutors,omitempty"`
	ObservationProviders []ObservationProviderBinding `json:"observationProviders,omitempty"`
	EvidenceProviders    []EvidenceProviderBinding    `json:"evidenceProviders,omitempty"`
	PermissionGrants     []PermissionGrant            `json:"permissionGrants,omitempty"`
	ResourceBindings     []ResourceBinding            `json:"resourceBindings,omitempty"`
	LeasePolicies        []CapabilityLeasePolicy      `json:"leasePolicies,omitempty"`
}

func (e Environment) InstanceByID(id core.CanonicalID) (DatabaseInstance, bool) {
	for _, v := range e.Instances {
		if v.ID == id {
			return v, true
		}
	}
	return DatabaseInstance{}, false
}
