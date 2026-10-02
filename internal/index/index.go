package index

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/dependency"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/releaseplan"
)

type SemanticElementEntry struct {
	Kind      string           `json:"kind"`
	ID        core.CanonicalID `json:"id"`
	Namespace core.CanonicalID `json:"namespace"`
	Parent    core.CanonicalID `json:"parent,omitempty"`
}

type SemanticDependency struct {
	FromKind string           `json:"fromKind"`
	From     core.CanonicalID `json:"from"`
	Relation string           `json:"relation"`
	ToKind   string           `json:"toKind"`
	To       core.CanonicalID `json:"to"`
}

type ApplicationDatabaseEdge struct {
	Application core.CanonicalID `json:"application"`
	Alias       core.LocalName   `json:"alias"`
	Database    core.CanonicalID `json:"database"`
	Access      string           `json:"access"`
}

type ApplicationPatternIndexEntry struct {
	Application   core.CanonicalID `json:"application"`
	Pattern       core.CanonicalID `json:"pattern"`
	Kind          string           `json:"kind"`
	Data          core.CanonicalID `json:"data,omitempty"`
	Relation      core.CanonicalID `json:"relation,omitempty"`
	Query         core.CanonicalID `json:"query,omitempty"`
	Capability    core.CanonicalID `json:"capability,omitempty"`
	Datasource    core.LocalName   `json:"datasource,omitempty"`
	Contract      core.CanonicalID `json:"contract,omitempty"`
	ContractQuery string           `json:"contractQuery,omitempty"`
}

type DeclaredDependency struct {
	FromKind string           `json:"fromKind"`
	From     core.CanonicalID `json:"from"`
	Relation string           `json:"relation"`
	ToKind   string           `json:"toKind"`
	To       core.CanonicalID `json:"to"`
	Alias    core.LocalName   `json:"alias,omitempty"`
}

type EnvironmentBinding struct {
	Environment core.CanonicalID `json:"environment"`
	Application core.CanonicalID `json:"application"`
	Alias       core.LocalName   `json:"alias"`
	Instance    core.CanonicalID `json:"instance"`
	Database    core.CanonicalID `json:"database"`
	Provider    string           `json:"provider"`
}

type DeploymentExecutorRouteEntry struct {
	Environment core.CanonicalID `json:"environment"`
	Action      string           `json:"action"`
	Scope       string           `json:"scope,omitempty"`
	Target      core.CanonicalID `json:"target,omitempty"`
	Executor    core.CanonicalID `json:"executor"`
}

type ObservationProviderRouteEntry struct {
	Environment core.CanonicalID `json:"environment"`
	Kind        string           `json:"kind"`
	Provider    core.CanonicalID `json:"provider"`
}

type EvidenceProviderRouteEntry struct {
	Environment core.CanonicalID `json:"environment"`
	Kind        string           `json:"kind"`
	Provider    core.CanonicalID `json:"provider"`
}

type PluginIndexEntry struct {
	ID                     core.CanonicalID `json:"id"`
	Version                string           `json:"version"`
	Runtime                string           `json:"runtime"`
	SandboxPolicy          string           `json:"sandboxPolicy,omitempty"`
	ArtifactDigest         string           `json:"artifactDigest,omitempty"`
	ArtifactSignatureRef   string           `json:"artifactSignatureRef,omitempty"`
	ArtifactProvenance     string           `json:"artifactProvenance,omitempty"`
	Capabilities           []string         `json:"capabilities,omitempty"`
	PermissionRequirements []string         `json:"permissionRequirements,omitempty"`
	ExternalRequirements   []string         `json:"externalRequirements,omitempty"`
}

type BundleIndexEntry struct {
	ID      core.CanonicalID `json:"id"`
	Version string           `json:"version"`
	Plugins []string         `json:"plugins,omitempty"`
}

type EnvironmentPluginResolutionEntry struct {
	Environment  core.CanonicalID `json:"environment"`
	Plugins      []string         `json:"plugins,omitempty"`
	Capabilities []string         `json:"capabilities,omitempty"`
}

type PermissionGrantIndexEntry struct {
	Environment    core.CanonicalID `json:"environment"`
	CapabilityKind string           `json:"capabilityKind"`
	CapabilityID   core.CanonicalID `json:"capabilityId"`
	Permission     string           `json:"permission"`
}

type ResourceBindingIndexEntry struct {
	Environment    core.CanonicalID `json:"environment"`
	Kind           string           `json:"kind"`
	Scope          string           `json:"scope"`
	ResourceID     core.CanonicalID `json:"resourceId"`
	Refresh        string           `json:"refresh,omitempty"`
	SecretDelivery string           `json:"secretDelivery,omitempty"`
	SecretTarget   string           `json:"secretTarget,omitempty"`
}

type LeasePolicyIndexEntry struct {
	Environment    core.CanonicalID `json:"environment"`
	CapabilityKind string           `json:"capabilityKind"`
	CapabilityID   core.CanonicalID `json:"capabilityId"`
	MaxLifetime    string           `json:"maxLifetime"`
	RenewBefore    string           `json:"renewBefore"`
	RevokeTimeout  string           `json:"revokeTimeout"`
	RenewalFailure string           `json:"renewalFailure"`
}

type QueryIndexEntry struct {
	Query       core.CanonicalID `json:"query"`
	Application core.CanonicalID `json:"application"`
	Datasource  core.LocalName   `json:"datasource"`
	Dialect     string           `json:"dialect"`
}

type PersistenceIndexEntry struct {
	Mapping      core.CanonicalID `json:"mapping"`
	Database     core.CanonicalID `json:"database"`
	SemanticData core.CanonicalID `json:"semanticData"`
	Table        string           `json:"table"`
}

type MigrationProviderIndexEntry struct {
	Database core.CanonicalID `json:"database"`
	Tool     string           `json:"tool"`
	Mode     string           `json:"mode"`
	Source   string           `json:"source,omitempty"`
}

type MigrationIndexEntry struct {
	Database core.CanonicalID `json:"database"`
	ID       string           `json:"id"`
	Parents  []string         `json:"parents,omitempty"`
	Recovery string           `json:"recovery"`
	Steps    []string         `json:"steps,omitempty"`
}

type ApplicationReleaseDatabaseRequirementEntry struct {
	Alias     core.LocalName   `json:"alias"`
	Database  core.CanonicalID `json:"database"`
	Access    string           `json:"access"`
	Contracts []string         `json:"contracts,omitempty"`
}

type ApplicationReleaseIndexEntry struct {
	Application    core.CanonicalID                             `json:"application"`
	Version        string                                       `json:"version"`
	SourceRevision string                                       `json:"sourceRevision,omitempty"`
	Databases      []ApplicationReleaseDatabaseRequirementEntry `json:"databases,omitempty"`
}

type DatabaseReleaseIndexEntry struct {
	Database          core.CanonicalID `json:"database"`
	Version           string           `json:"version"`
	SchemaFingerprint string           `json:"schemaFingerprint,omitempty"`
	Migrations        []string         `json:"migrations,omitempty"`
	Contracts         []string         `json:"contracts,omitempty"`
	DataReadiness     string           `json:"dataReadiness"`
}

type SolutionReleaseIndexEntry struct {
	ID         core.CanonicalID `json:"id"`
	Version    string           `json:"version"`
	Steps      int              `json:"steps"`
	Executable bool             `json:"executable"`
	Findings   []string         `json:"findings,omitempty"`
}

type BindingIndexEntry struct {
	Application core.CanonicalID `json:"application,omitempty"`
	Capability  core.CanonicalID `json:"capability"`
	Adapter     string           `json:"adapter"`
	Hook        string           `json:"hook"`
}

type Derived struct {
	Plugins                      []PluginIndexEntry                 `json:"plugins,omitempty"`
	Bundles                      []BundleIndexEntry                 `json:"bundles,omitempty"`
	EnvironmentPluginResolution  []EnvironmentPluginResolutionEntry `json:"environmentPluginResolution,omitempty"`
	SemanticElements             []SemanticElementEntry             `json:"semanticElements"`
	SemanticDependencies         []SemanticDependency               `json:"semanticDependencies"`
	DeclaredDependencies         []DeclaredDependency               `json:"declaredDependencies"`
	ApplicationPatterns          []ApplicationPatternIndexEntry     `json:"applicationPatterns,omitempty"`
	ApplicationDatabases         []ApplicationDatabaseEdge          `json:"applicationDatabases"`
	EnvironmentBindings          []EnvironmentBinding               `json:"environmentBindings"`
	DeploymentExecutorRoutes     []DeploymentExecutorRouteEntry     `json:"deploymentExecutorRoutes,omitempty"`
	ObservationProviderRoutes    []ObservationProviderRouteEntry    `json:"observationProviderRoutes,omitempty"`
	EvidenceProviderRoutes       []EvidenceProviderRouteEntry       `json:"evidenceProviderRoutes,omitempty"`
	PermissionGrants             []PermissionGrantIndexEntry        `json:"permissionGrants,omitempty"`
	ResourceBindings             []ResourceBindingIndexEntry        `json:"resourceBindings,omitempty"`
	LeasePolicies                []LeasePolicyIndexEntry            `json:"leasePolicies,omitempty"`
	Queries                      []QueryIndexEntry                  `json:"queries"`
	CustomGoBindings             []BindingIndexEntry                `json:"customGoBindings"`
	PersistenceMappings          []PersistenceIndexEntry            `json:"persistenceMappings,omitempty"`
	Migrations                   []MigrationIndexEntry              `json:"migrations,omitempty"`
	MigrationProviders           []MigrationProviderIndexEntry      `json:"migrationProviders,omitempty"`
	ApplicationReleases          []ApplicationReleaseIndexEntry     `json:"applicationReleases,omitempty"`
	DatabaseReleases             []DatabaseReleaseIndexEntry        `json:"databaseReleases,omitempty"`
	SolutionReleases             []SolutionReleaseIndexEntry        `json:"solutionReleases,omitempty"`
	ObservedPhysicalDependencies []ObservedPhysicalDependency       `json:"observedPhysicalDependencies,omitempty"`
	EnvironmentRequirements      []dependency.Requirement           `json:"environmentRequirements,omitempty"`
	AnalyzedQueries              []AnalyzedQueryEntry               `json:"analyzedQueries,omitempty"`
	PhysicalSchemas              []physical.SchemaSnapshot          `json:"physicalSchemas,omitempty"`
	ExecutionPlans               []ExecutionPlanEntry               `json:"executionPlans,omitempty"`
}

func Build(w ir.World) Derived {
	var d Derived
	for _, p := range w.Plugins {
		e := PluginIndexEntry{ID: p.ID, Version: p.Version, Runtime: string(p.Runtime.Kind), SandboxPolicy: string(p.Runtime.Sandbox), ArtifactDigest: p.Artifact.Digest, ArtifactSignatureRef: p.Artifact.SignatureRef, ArtifactProvenance: p.Artifact.Provenance}
		for _, c := range p.Provides {
			e.Capabilities = append(e.Capabilities, string(c.Kind)+":"+string(c.ID))
			for _, perm := range c.Permissions {
				e.PermissionRequirements = append(e.PermissionRequirements, string(c.Kind)+":"+string(c.ID)+":"+perm.Key())
			}
		}
		for _, r := range p.External {
			e.ExternalRequirements = append(e.ExternalRequirements, r.Scope+":"+r.Kind+":"+r.Name)
		}
		d.Plugins = append(d.Plugins, e)
	}
	for _, b := range w.Bundles {
		e := BundleIndexEntry{ID: b.ID, Version: b.Version}
		for _, r := range b.Plugins {
			e.Plugins = append(e.Plugins, string(r.Plugin)+"@^"+fmt.Sprintf("%d", r.CompatibleMajor))
		}
		d.Bundles = append(d.Bundles, e)
	}
	catalog := plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}
	for _, env := range w.Environments {
		r, err := catalog.ResolveEnvironment(env)
		if err != nil {
			continue
		}
		e := EnvironmentPluginResolutionEntry{Environment: env.ID}
		for _, p := range r.Plugins {
			e.Plugins = append(e.Plugins, string(p.ID)+"@"+p.Version)
		}
		for _, c := range r.Capabilities {
			e.Capabilities = append(e.Capabilities, string(c.Capability.Kind)+":"+string(c.Capability.ID)+"<-"+string(c.Plugin.ID)+"@"+c.Plugin.Version)
		}
		d.EnvironmentPluginResolution = append(d.EnvironmentPluginResolution, e)
	}
	for _, ns := range w.Semantics {
		d.SemanticElements = append(d.SemanticElements, SemanticElementEntry{Kind: "namespace", ID: ns.ID, Namespace: ns.ID})
		for _, typ := range ns.Types {
			d.SemanticElements = append(d.SemanticElements, SemanticElementEntry{Kind: "type", ID: typ.ID, Namespace: ns.ID})
			if typ.Underlying != "" {
				d.SemanticDependencies = append(d.SemanticDependencies, SemanticDependency{FromKind: "semantic-type", From: typ.ID, Relation: "underlying", ToKind: "semantic-type", To: typ.Underlying})
			}
		}
		for _, data := range ns.Data {
			d.SemanticElements = append(d.SemanticElements, SemanticElementEntry{Kind: "data", ID: data.ID, Namespace: ns.ID})
			for _, a := range data.Attributes {
				d.SemanticElements = append(d.SemanticElements, SemanticElementEntry{Kind: "attribute", ID: a.ID, Namespace: ns.ID, Parent: data.ID})
				d.SemanticDependencies = append(d.SemanticDependencies, SemanticDependency{FromKind: "semantic-attribute", From: a.ID, Relation: "type", ToKind: "semantic-type", To: a.TypeRef})
			}
			for _, r := range data.Relations {
				d.SemanticElements = append(d.SemanticElements, SemanticElementEntry{Kind: "relation", ID: r.ID, Namespace: ns.ID, Parent: data.ID})
				d.SemanticDependencies = append(d.SemanticDependencies, SemanticDependency{FromKind: "semantic-relation", From: r.ID, Relation: "targets", ToKind: "semantic-data", To: r.Target})
			}
		}
		for _, v := range ns.Validations {
			d.SemanticElements = append(d.SemanticElements, SemanticElementEntry{Kind: "validation", ID: v.ID, Namespace: ns.ID})
			if v.Attribute != "" {
				d.SemanticDependencies = append(d.SemanticDependencies, SemanticDependency{FromKind: "semantic-validation", From: v.ID, Relation: "attribute", ToKind: "semantic-attribute", To: v.Attribute})
			}
			if v.Relation != "" {
				d.SemanticDependencies = append(d.SemanticDependencies, SemanticDependency{FromKind: "semantic-validation", From: v.ID, Relation: "relation", ToKind: "semantic-relation", To: v.Relation})
			}
		}
		for _, c := range ns.Capabilities {
			d.SemanticElements = append(d.SemanticElements, SemanticElementEntry{Kind: "capability", ID: c.ID, Namespace: ns.ID})
			for _, in := range c.Inputs {
				d.SemanticDependencies = append(d.SemanticDependencies, SemanticDependency{FromKind: "semantic-capability", From: c.ID, Relation: "input", ToKind: "semantic-element", To: in.Type})
			}
			if c.Output != "" {
				d.SemanticDependencies = append(d.SemanticDependencies, SemanticDependency{FromKind: "semantic-capability", From: c.ID, Relation: "output", ToKind: "semantic-element", To: c.Output})
			}
		}
	}
	for _, a := range w.Applications {
		for _, s := range a.SemanticUses {
			d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "application", From: a.ID, Relation: "uses-semantic", ToKind: "semantic", To: s})
		}
		for _, v := range a.Validations {
			d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "application", From: a.ID, Relation: "uses-validation", ToKind: "semantic-validation", To: v})
		}
		for _, p := range a.Patterns {
			e := ApplicationPatternIndexEntry{
				Application: a.ID,
				Pattern:     p.ID,
				Kind:        string(p.Kind),
				Data:        p.Data,
				Relation:    p.Relation,
				Query:       p.Query,
				Capability:  p.Capability,
				Datasource:  p.Datasource,
			}
			d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "application", From: a.ID, Relation: "owns-pattern", ToKind: "application-pattern", To: p.ID})
			if p.Data != "" {
				d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "application-pattern", From: p.ID, Relation: "data", ToKind: "semantic-data", To: p.Data})
			}
			if p.Relation != "" {
				d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "application-pattern", From: p.ID, Relation: "relation", ToKind: "semantic-relation", To: p.Relation})
			}
			if p.Query != "" {
				d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "application-pattern", From: p.ID, Relation: "query", ToKind: "query", To: p.Query})
			}
			if p.Capability != "" {
				d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "application-pattern", From: p.ID, Relation: "capability", ToKind: "semantic-capability", To: p.Capability})
			}
			if p.ContractQuery != nil {
				e.Contract = p.ContractQuery.Contract
				e.ContractQuery = string(p.ContractQuery.Contract) + "/" + string(p.ContractQuery.Query)
				d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "application-pattern", From: p.ID, Relation: "contract", ToKind: "database-contract", To: p.ContractQuery.Contract})
				d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "application-pattern", From: p.ID, Relation: "contract-query", ToKind: "database-contract-query", To: core.CanonicalID(e.ContractQuery)})
			}
			d.ApplicationPatterns = append(d.ApplicationPatterns, e)
		}
		for _, r := range a.Databases {
			d.ApplicationDatabases = append(d.ApplicationDatabases, ApplicationDatabaseEdge{Application: a.ID, Alias: r.Alias, Database: r.TargetDatabase, Access: string(r.Access)})
			d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "application", From: a.ID, Relation: "targets-database", ToKind: "database", To: r.TargetDatabase, Alias: r.Alias})
			for _, cr := range r.RequiredContracts {
				d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "application", From: a.ID, Relation: "requires-contract", ToKind: "database-contract", To: cr.ContractID, Alias: r.Alias})
			}
		}
	}
	for _, db := range w.Databases {
		for _, s := range db.SemanticUses {
			d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{FromKind: "database", From: db.ID, Relation: "uses-semantic", ToKind: "semantic", To: s})
		}
		for _, c := range db.Contracts {
			for _, ref := range c.Reads {
				d.SemanticDependencies = append(d.SemanticDependencies, SemanticDependency{FromKind: "database-contract", From: c.ID, Relation: "reads", ToKind: "semantic-element", To: ref})
			}
			for _, ref := range c.Writes {
				d.SemanticDependencies = append(d.SemanticDependencies, SemanticDependency{FromKind: "database-contract", From: c.ID, Relation: "writes", ToKind: "semantic-element", To: ref})
			}
		}
	}
	for _, m := range w.Persistence {
		d.PersistenceMappings = append(d.PersistenceMappings, PersistenceIndexEntry{Mapping: m.ID, Database: m.Database, SemanticData: m.SemanticData, Table: m.Table})
	}
	for _, cfg := range w.MigrationProviders {
		d.MigrationProviders = append(d.MigrationProviders, MigrationProviderIndexEntry{Database: cfg.Database, Tool: string(cfg.Tool), Mode: string(cfg.Mode), Source: cfg.Source})
	}
	for _, m := range w.Migrations {
		e := MigrationIndexEntry{Database: m.Database, ID: m.ID, Parents: append([]string(nil), m.Parents...), Recovery: string(m.Recovery)}
		for _, step := range m.Steps {
			e.Steps = append(e.Steps, string(step.Kind))
		}
		d.Migrations = append(d.Migrations, e)
	}
	for _, r := range w.ApplicationReleases {
		e := ApplicationReleaseIndexEntry{Application: r.Application, Version: r.Version, SourceRevision: r.SourceRevision}
		for _, req := range r.Normalize().Databases {
			db := ApplicationReleaseDatabaseRequirementEntry{Alias: req.Alias, Database: req.TargetDatabase, Access: string(req.Access)}
			for _, cr := range req.RequiredContracts {
				db.Contracts = append(db.Contracts, string(cr.ContractID)+"@^"+fmt.Sprintf("%d", cr.CompatibleMajor))
			}
			e.Databases = append(e.Databases, db)
		}
		d.ApplicationReleases = append(d.ApplicationReleases, e)
	}
	for _, r := range w.DatabaseReleases {
		e := DatabaseReleaseIndexEntry{Database: r.Database, Version: r.Version, SchemaFingerprint: r.SchemaFingerprint, Migrations: append([]string(nil), r.Migrations...), DataReadiness: string(r.DataReadiness)}
		for _, c := range r.Provides {
			e.Contracts = append(e.Contracts, string(c.ID)+"@"+fmt.Sprintf("%d.%d", c.Version.Major, c.Version.Minor))
		}
		d.DatabaseReleases = append(d.DatabaseReleases, e)
	}
	for _, r := range w.SolutionReleases {
		c := releaseplan.Compile(r, w.ApplicationReleases, w.DatabaseReleases)
		d.SolutionReleases = append(d.SolutionReleases, SolutionReleaseIndexEntry{ID: r.ID, Version: r.Version, Steps: len(r.Steps), Executable: c.Executable, Findings: append([]string(nil), c.Findings...)})
	}
	for _, e := range w.Environments {
		for _, rb := range e.ResourceBindings {
			d.ResourceBindings = append(d.ResourceBindings, ResourceBindingIndexEntry{Environment: e.ID, Kind: string(rb.Kind), Scope: rb.Scope, ResourceID: rb.ResourceID, Refresh: string(rb.Refresh), SecretDelivery: string(rb.SecretDelivery), SecretTarget: rb.SecretTarget})
		}
		for _, g := range e.PermissionGrants {
			d.PermissionGrants = append(d.PermissionGrants, PermissionGrantIndexEntry{Environment: e.ID, CapabilityKind: g.CapabilityKind, CapabilityID: g.CapabilityID, Permission: g.Permission.Key()})
		}
		for _, lp := range e.LeasePolicies {
			p := lp.Policy.WithDefaults()
			d.LeasePolicies = append(d.LeasePolicies, LeasePolicyIndexEntry{Environment: e.ID, CapabilityKind: lp.CapabilityKind, CapabilityID: lp.CapabilityID, MaxLifetime: p.MaxLifetime, RenewBefore: p.RenewBefore, RevokeTimeout: p.RevokeTimeout, RenewalFailure: string(p.RenewalFailure)})
		}
		instances := map[core.CanonicalID]struct {
			db       core.CanonicalID
			provider string
		}{}
		for _, inst := range e.Instances {
			instances[inst.ID] = struct {
				db       core.CanonicalID
				provider string
			}{inst.Definition, string(inst.Provider)}
		}
		for _, b := range e.Bindings {
			inst := instances[b.Instance]
			d.EnvironmentBindings = append(d.EnvironmentBindings, EnvironmentBinding{Environment: e.ID, Application: b.Application, Alias: b.Alias, Instance: b.Instance, Database: inst.db, Provider: inst.provider})
		}
		for _, b := range e.DeploymentExecutors {
			d.DeploymentExecutorRoutes = append(d.DeploymentExecutorRoutes, DeploymentExecutorRouteEntry{Environment: e.ID, Action: string(b.Action), Scope: string(b.Scope), Target: b.Target, Executor: b.Executor})
		}
		for _, b := range e.ObservationProviders {
			d.ObservationProviderRoutes = append(d.ObservationProviderRoutes, ObservationProviderRouteEntry{Environment: e.ID, Kind: string(b.Kind), Provider: b.Provider})
		}
		for _, b := range e.EvidenceProviders {
			d.EvidenceProviderRoutes = append(d.EvidenceProviderRoutes, EvidenceProviderRouteEntry{Environment: e.ID, Kind: string(b.Kind), Provider: b.Provider})
		}
	}
	for _, q := range w.Queries {
		d.Queries = append(d.Queries, QueryIndexEntry{Query: q.ID, Application: q.Application, Datasource: q.Datasource, Dialect: string(q.Dialect)})
	}
	for _, b := range w.Bindings {
		d.CustomGoBindings = append(d.CustomGoBindings, BindingIndexEntry{Application: b.Application, Capability: b.Capability, Adapter: b.ExpectedAdapterFilename(), Hook: b.ExpectedHookName()})
	}

	d.EnvironmentRequirements = dependency.Derive(w)

	sort.Slice(d.Plugins, func(i, j int) bool {
		if d.Plugins[i].ID == d.Plugins[j].ID {
			return d.Plugins[i].Version < d.Plugins[j].Version
		}
		return d.Plugins[i].ID < d.Plugins[j].ID
	})
	sort.Slice(d.Bundles, func(i, j int) bool {
		if d.Bundles[i].ID == d.Bundles[j].ID {
			return d.Bundles[i].Version < d.Bundles[j].Version
		}
		return d.Bundles[i].ID < d.Bundles[j].ID
	})
	sort.Slice(d.EnvironmentPluginResolution, func(i, j int) bool {
		return d.EnvironmentPluginResolution[i].Environment < d.EnvironmentPluginResolution[j].Environment
	})
	sort.Slice(d.SemanticElements, func(i, j int) bool {
		if d.SemanticElements[i].ID != d.SemanticElements[j].ID {
			return d.SemanticElements[i].ID < d.SemanticElements[j].ID
		}
		return d.SemanticElements[i].Kind < d.SemanticElements[j].Kind
	})
	sort.Slice(d.SemanticDependencies, func(i, j int) bool {
		a, b := d.SemanticDependencies[i], d.SemanticDependencies[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Relation != b.Relation {
			return a.Relation < b.Relation
		}
		return a.To < b.To
	})
	sort.Slice(d.DeclaredDependencies, func(i, j int) bool {
		a, b := d.DeclaredDependencies[i], d.DeclaredDependencies[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Relation != b.Relation {
			return a.Relation < b.Relation
		}
		if a.Alias != b.Alias {
			return a.Alias < b.Alias
		}
		return a.To < b.To
	})
	sort.Slice(d.ApplicationPatterns, func(i, j int) bool {
		if d.ApplicationPatterns[i].Application == d.ApplicationPatterns[j].Application {
			return d.ApplicationPatterns[i].Pattern < d.ApplicationPatterns[j].Pattern
		}
		return d.ApplicationPatterns[i].Application < d.ApplicationPatterns[j].Application
	})
	sort.Slice(d.ApplicationDatabases, func(i, j int) bool {
		return d.ApplicationDatabases[i].Application < d.ApplicationDatabases[j].Application || (d.ApplicationDatabases[i].Application == d.ApplicationDatabases[j].Application && d.ApplicationDatabases[i].Alias < d.ApplicationDatabases[j].Alias)
	})
	sort.Slice(d.EnvironmentBindings, func(i, j int) bool {
		return d.EnvironmentBindings[i].Environment < d.EnvironmentBindings[j].Environment || (d.EnvironmentBindings[i].Environment == d.EnvironmentBindings[j].Environment && d.EnvironmentBindings[i].Alias < d.EnvironmentBindings[j].Alias)
	})
	sort.Slice(d.DeploymentExecutorRoutes, func(i, j int) bool {
		a, b := d.DeploymentExecutorRoutes[i], d.DeploymentExecutorRoutes[j]
		if a.Environment == b.Environment {
			if a.Action == b.Action {
				if a.Scope == b.Scope {
					return a.Target < b.Target
				}
				return a.Scope < b.Scope
			}
			return a.Action < b.Action
		}
		return a.Environment < b.Environment
	})
	sort.Slice(d.ObservationProviderRoutes, func(i, j int) bool {
		a, b := d.ObservationProviderRoutes[i], d.ObservationProviderRoutes[j]
		if a.Environment == b.Environment {
			return a.Kind < b.Kind
		}
		return a.Environment < b.Environment
	})
	sort.Slice(d.EvidenceProviderRoutes, func(i, j int) bool {
		a, b := d.EvidenceProviderRoutes[i], d.EvidenceProviderRoutes[j]
		if a.Environment == b.Environment {
			return a.Kind < b.Kind
		}
		return a.Environment < b.Environment
	})
	sort.Slice(d.PermissionGrants, func(i, j int) bool {
		a, b := d.PermissionGrants[i], d.PermissionGrants[j]
		if a.Environment != b.Environment {
			return a.Environment < b.Environment
		}
		if a.CapabilityKind != b.CapabilityKind {
			return a.CapabilityKind < b.CapabilityKind
		}
		if a.CapabilityID != b.CapabilityID {
			return a.CapabilityID < b.CapabilityID
		}
		return a.Permission < b.Permission
	})
	sort.Slice(d.ResourceBindings, func(i, j int) bool {
		a, b := d.ResourceBindings[i], d.ResourceBindings[j]
		if a.Environment != b.Environment {
			return a.Environment < b.Environment
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		return a.ResourceID < b.ResourceID
	})
	sort.Slice(d.LeasePolicies, func(i, j int) bool {
		a, b := d.LeasePolicies[i], d.LeasePolicies[j]
		if a.Environment != b.Environment {
			return a.Environment < b.Environment
		}
		if a.CapabilityKind != b.CapabilityKind {
			return a.CapabilityKind < b.CapabilityKind
		}
		return a.CapabilityID < b.CapabilityID
	})
	sort.Slice(d.Queries, func(i, j int) bool { return d.Queries[i].Query < d.Queries[j].Query })
	sort.Slice(d.CustomGoBindings, func(i, j int) bool { return d.CustomGoBindings[i].Capability < d.CustomGoBindings[j].Capability })
	sort.Slice(d.PersistenceMappings, func(i, j int) bool { return d.PersistenceMappings[i].Mapping < d.PersistenceMappings[j].Mapping })
	sort.Slice(d.MigrationProviders, func(i, j int) bool { return d.MigrationProviders[i].Database < d.MigrationProviders[j].Database })
	sort.Slice(d.Migrations, func(i, j int) bool {
		if d.Migrations[i].Database == d.Migrations[j].Database {
			return d.Migrations[i].ID < d.Migrations[j].ID
		}
		return d.Migrations[i].Database < d.Migrations[j].Database
	})
	sort.Slice(d.ApplicationReleases, func(i, j int) bool {
		if d.ApplicationReleases[i].Application == d.ApplicationReleases[j].Application {
			return d.ApplicationReleases[i].Version < d.ApplicationReleases[j].Version
		}
		return d.ApplicationReleases[i].Application < d.ApplicationReleases[j].Application
	})
	sort.Slice(d.SolutionReleases, func(i, j int) bool {
		if d.SolutionReleases[i].ID == d.SolutionReleases[j].ID {
			return d.SolutionReleases[i].Version < d.SolutionReleases[j].Version
		}
		return d.SolutionReleases[i].ID < d.SolutionReleases[j].ID
	})
	sort.Slice(d.DatabaseReleases, func(i, j int) bool {
		if d.DatabaseReleases[i].Database == d.DatabaseReleases[j].Database {
			return d.DatabaseReleases[i].Version < d.DatabaseReleases[j].Version
		}
		return d.DatabaseReleases[i].Database < d.DatabaseReleases[j].Database
	})
	return d
}
