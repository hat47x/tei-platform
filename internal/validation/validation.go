package validation

import (
	"fmt"
	"strings"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/release"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/releaseplan"
	"tei/internal/semantic"
	"tei/internal/subject"
)

type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

type Diagnostic struct {
	Code     string           `json:"code"`
	Severity Severity         `json:"severity"`
	Message  string           `json:"message"`
	Subject  *subject.Subject `json:"subject,omitempty"`
}

func Validate(w ir.World) []Diagnostic {
	out := validateApplicationPatterns(w)
	out = append(out, validateQueryResultMappings(w)...)
	out = append(out, validateSemanticBusinessValidations(w)...)
	out = append(out, validateCapabilityOutcomes(w)...)
	out = append(out, validateBusinessValidationReferences(w)...)

	pluginCatalog := plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}
	seenPluginVersions := map[string]struct{}{}
	for _, p := range w.Plugins {
		key := string(p.ID) + "@" + p.Version
		if _, dup := seenPluginVersions[key]; dup {
			out = append(out, errd("DUPLICATE_PLUGIN_VERSION", "plugin %s version %s is declared more than once", p.ID, p.Version))
		}
		seenPluginVersions[key] = struct{}{}
		if _, err := plugin.ParseVersion(p.Version); err != nil {
			out = append(out, errd("INVALID_PLUGIN_VERSION", "plugin %s: %v", p.ID, err))
		}
		if !p.Runtime.Kind.Valid() {
			out = append(out, errFor(subject.Subject{Kind: "plugin", ID: string(p.ID) + "@" + p.Version}, "INVALID_PLUGIN_RUNTIME", "plugin %s has invalid runtime %s", p.ID, p.Runtime.Kind))
		}
		if p.Runtime.Kind == plugin.RuntimeProcess && strings.TrimSpace(p.Runtime.Command) == "" {
			out = append(out, errd("PLUGIN_PROCESS_COMMAND_REQUIRED", "plugin %s process runtime requires a command", p.ID))
		}
		if !p.Runtime.Sandbox.Valid() {
			out = append(out, errd("INVALID_PLUGIN_SANDBOX_POLICY", "plugin %s has invalid sandbox policy %s", p.ID, p.Runtime.Sandbox))
		}
		if p.Runtime.Kind != plugin.RuntimeProcess && p.Runtime.Sandbox != "" && p.Runtime.Sandbox != plugin.SandboxNone {
			out = append(out, errd("PLUGIN_SANDBOX_REQUIRES_PROCESS_RUNTIME", "plugin %s sandbox policy %s requires process runtime", p.ID, p.Runtime.Sandbox))
		}
		if err := p.Artifact.Validate(); err != nil {
			out = append(out, errd("INVALID_PLUGIN_ARTIFACT", "plugin %s: %v", p.ID, err))
		}
		for _, ext := range p.External {
			validScope := ext.Scope == "editor" || ext.Scope == "build" || ext.Scope == "test" || ext.Scope == "runtime" || ext.Scope == "migration"
			validKind := ext.Kind == "toolchain" || ext.Kind == "runtime" || ext.Kind == "language-package" || ext.Kind == "tool"
			if !validScope {
				out = append(out, errd("INVALID_PLUGIN_EXTERNAL_SCOPE", "plugin %s external requirement %s has invalid scope %s", p.ID, ext.Name, ext.Scope))
			}
			if !validKind {
				out = append(out, errd("INVALID_PLUGIN_EXTERNAL_KIND", "plugin %s external requirement %s has invalid kind %s", p.ID, ext.Name, ext.Kind))
			}
			if strings.TrimSpace(ext.Name) == "" {
				out = append(out, errd("PLUGIN_EXTERNAL_NAME_REQUIRED", "plugin %s has external requirement with empty name", p.ID))
			}
		}
		seenCaps := map[string]struct{}{}
		for _, c := range p.Provides {
			if !c.Kind.Valid() {
				out = append(out, errd("INVALID_PLUGIN_CAPABILITY_KIND", "plugin %s capability %s has invalid kind %s", p.ID, c.ID, c.Kind))
			}
			ck := string(c.Kind) + "|" + string(c.ID)
			if _, dup := seenCaps[ck]; dup {
				out = append(out, errd("DUPLICATE_PLUGIN_CAPABILITY", "plugin %s provides %s %s more than once", p.ID, c.Kind, c.ID))
			}
			seenCaps[ck] = struct{}{}
			for _, a := range c.Actions {
				if !a.Valid() {
					out = append(out, errd("INVALID_PLUGIN_ACTION_SUPPORT", "plugin %s capability %s lists invalid action %s", p.ID, c.ID, a))
				}
			}
			for _, o := range c.Observations {
				if !o.Valid() {
					out = append(out, errd("INVALID_PLUGIN_OBSERVATION_SUPPORT", "plugin %s capability %s lists invalid observation %s", p.ID, c.ID, o))
				}
			}
			for _, e := range c.Evidence {
				if !e.Valid() {
					out = append(out, errd("INVALID_PLUGIN_EVIDENCE_SUPPORT", "plugin %s capability %s lists invalid evidence %s", p.ID, c.ID, e))
				}
			}
			for _, dbp := range c.DatabaseProviders {
				if !dbp.Known() || dbp == environment.ProviderGeneric {
					out = append(out, errd("INVALID_PLUGIN_DATABASE_PROVIDER_SUPPORT", "plugin %s capability %s lists invalid database provider %s", p.ID, c.ID, dbp))
				}
			}
			seenPermissions := map[string]struct{}{}
			for _, perm := range c.Permissions {
				if err := perm.Validate(); err != nil {
					out = append(out, errd("INVALID_PLUGIN_PERMISSION", "plugin %s capability %s: %v", p.ID, c.ID, err))
				}
				if _, dup := seenPermissions[perm.Key()]; dup {
					out = append(out, errd("DUPLICATE_PLUGIN_PERMISSION", "plugin %s capability %s declares permission %s more than once", p.ID, c.ID, perm.Key()))
				}
				seenPermissions[perm.Key()] = struct{}{}
			}
		}
	}
	seenBundleVersions := map[string]struct{}{}
	for _, b := range w.Bundles {
		key := string(b.ID) + "@" + b.Version
		if _, dup := seenBundleVersions[key]; dup {
			out = append(out, errd("DUPLICATE_BUNDLE_VERSION", "bundle %s version %s is declared more than once", b.ID, b.Version))
		}
		seenBundleVersions[key] = struct{}{}
		if _, err := plugin.ParseVersion(b.Version); err != nil {
			out = append(out, errd("INVALID_BUNDLE_VERSION", "bundle %s: %v", b.ID, err))
		}
	}

	semanticNamespaces := map[core.CanonicalID]semantic.Namespace{}
	semanticElements := map[core.CanonicalID]string{}
	semanticTypes := map[core.CanonicalID]semantic.Type{}
	semanticData := map[core.CanonicalID]struct{}{}
	semanticAttributes := []semantic.Attribute{}
	semanticAttributeByID := map[core.CanonicalID]semantic.Attribute{}
	semanticRelations := []semantic.Relation{}
	semanticCapabilities := []semantic.Capability{}
	semanticTypeDefs := []semantic.Type{}

	addSemanticElement := func(id core.CanonicalID, kind string) {
		if prior, exists := semanticElements[id]; exists {
			out = append(out, errd("DUPLICATE_SEMANTIC_ELEMENT", "semantic element %s is declared as both %s and %s", id, prior, kind))
			return
		}
		semanticElements[id] = kind
	}

	for _, ns := range w.Semantics {
		if _, exists := semanticNamespaces[ns.ID]; exists {
			out = append(out, errd("DUPLICATE_SEMANTIC_NAMESPACE", "duplicate semantic namespace %s", ns.ID))
		}
		semanticNamespaces[ns.ID] = ns
		for _, typ := range ns.Types {
			if !strings.HasPrefix(string(typ.ID), string(ns.ID)+".") {
				out = append(out, errFor(subject.Subject{Kind: "semantic-type", ID: string(typ.ID)}, "SEMANTIC_TYPE_NAMESPACE_MISMATCH", "type %s is declared in semantic namespace %s", typ.ID, ns.ID))
			}
			if !typ.Kind.Valid() {
				out = append(out, errd("INVALID_SEMANTIC_TYPE_KIND", "type %s has invalid kind %q", typ.ID, typ.Kind))
			}
			addSemanticElement(typ.ID, "type")
			semanticTypes[typ.ID] = typ
			semanticTypeDefs = append(semanticTypeDefs, typ)
		}
		for _, d := range ns.Data {
			if !strings.HasPrefix(string(d.ID), string(ns.ID)+".") {
				out = append(out, errFor(subject.Subject{Kind: "semantic-data", ID: string(d.ID)}, "SEMANTIC_DATA_NAMESPACE_MISMATCH", "data %s is declared in semantic namespace %s", d.ID, ns.ID))
			}
			addSemanticElement(d.ID, "data")
			semanticData[d.ID] = struct{}{}
			for _, a := range d.Attributes {
				if !strings.HasPrefix(string(a.ID), string(d.ID)+".") {
					out = append(out, errFor(subject.Subject{Kind: "semantic-attribute", ID: string(a.ID)}, "SEMANTIC_ATTRIBUTE_PARENT_MISMATCH", "attribute %s is declared under data %s", a.ID, d.ID))
				}
				addSemanticElement(a.ID, "attribute")
				semanticAttributes = append(semanticAttributes, a)
				semanticAttributeByID[a.ID] = a
			}
			for _, r := range d.Relations {
				if !strings.HasPrefix(string(r.ID), string(d.ID)+".") {
					out = append(out, errd("SEMANTIC_RELATION_PARENT_MISMATCH", "relation %s is declared under data %s", r.ID, d.ID))
				}
				if !r.Cardinality.Valid() {
					out = append(out, errFor(subject.Subject{Kind: "semantic-relation", ID: string(r.ID)}, "INVALID_SEMANTIC_RELATION_CARDINALITY", "relation %s has invalid cardinality %q", r.ID, r.Cardinality))
				}
				addSemanticElement(r.ID, "relation")
				semanticRelations = append(semanticRelations, r)
			}
		}
		for _, c := range ns.Capabilities {
			if !strings.HasPrefix(string(c.ID), string(ns.ID)+".") {
				out = append(out, errd("SEMANTIC_CAPABILITY_NAMESPACE_MISMATCH", "capability %s is declared in semantic namespace %s", c.ID, ns.ID))
			}
			addSemanticElement(c.ID, "capability")
			semanticCapabilities = append(semanticCapabilities, c)
		}
	}
	for _, typ := range semanticTypeDefs {
		if typ.Kind == semantic.TypeAlias {
			if typ.Underlying == "" {
				out = append(out, errd("SEMANTIC_ALIAS_TYPE_MISSING_UNDERLYING", "alias type %s has no underlying type", typ.ID))
			} else if _, ok := semanticTypes[typ.Underlying]; !ok {
				out = append(out, errd("UNKNOWN_SEMANTIC_TYPE_UNDERLYING", "type %s refers to unknown underlying type %s", typ.ID, typ.Underlying))
			}
		}
	}
	// Alias types form a semantic refinement chain and must not recurse.
	visitState := map[core.CanonicalID]int{}
	reportedTypeCycles := map[core.CanonicalID]struct{}{}
	var visitType func(core.CanonicalID)
	visitType = func(id core.CanonicalID) {
		if visitState[id] == 2 {
			return
		}
		visitState[id] = 1
		typ := semanticTypes[id]
		if typ.Kind == semantic.TypeAlias && typ.Underlying != "" {
			if under, ok := semanticTypes[typ.Underlying]; ok && under.Kind == semantic.TypeAlias {
				if visitState[typ.Underlying] == 1 {
					if _, done := reportedTypeCycles[id]; !done {
						out = append(out, errd("SEMANTIC_TYPE_ALIAS_CYCLE", "semantic type alias cycle reaches %s -> %s", id, typ.Underlying))
						reportedTypeCycles[id] = struct{}{}
					}
				} else {
					visitType(typ.Underlying)
				}
			}
		}
		visitState[id] = 2
	}
	for id := range semanticTypes {
		if visitState[id] == 0 {
			visitType(id)
		}
	}

	for _, a := range semanticAttributes {
		if _, ok := semanticTypes[a.TypeRef]; !ok {
			out = append(out, errd("UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE", "attribute %s refers to unknown semantic type %s", a.ID, a.TypeRef))
		}
	}
	for _, r := range semanticRelations {
		if _, ok := semanticData[r.Target]; !ok {
			out = append(out, errd("UNKNOWN_SEMANTIC_RELATION_TARGET", "relation %s targets unknown semantic data %s", r.ID, r.Target))
		}
	}
	semanticValueTypeExists := func(id core.CanonicalID) bool {
		if _, ok := semanticData[id]; ok {
			return true
		}
		_, ok := semanticTypes[id]
		return ok
	}
	for _, c := range semanticCapabilities {
		for _, in := range c.Inputs {
			if !semanticValueTypeExists(in.Type) {
				out = append(out, errd("UNKNOWN_SEMANTIC_CAPABILITY_INPUT", "capability %s input %s refers to unknown semantic value type %s", c.ID, in.Name, in.Type))
			}
		}
		if c.Output != "" && !semanticValueTypeExists(c.Output) {
			out = append(out, errd("UNKNOWN_SEMANTIC_CAPABILITY_OUTPUT", "capability %s output refers to unknown semantic value type %s", c.ID, c.Output))
		}

	}

	apps := map[core.CanonicalID]application.Application{}
	dbs := map[core.CanonicalID]database.Database{}
	contracts := map[core.CanonicalID]database.Contract{}
	contractOwners := map[core.CanonicalID]core.CanonicalID{}
	envs := map[core.CanonicalID]environment.Environment{}

	for _, a := range w.Applications {
		if _, exists := apps[a.ID]; exists {
			out = append(out, errFor(subject.Subject{Kind: "application", ID: string(a.ID)}, "DUPLICATE_APPLICATION", "duplicate application %s", a.ID))
		}
		apps[a.ID] = a
		seenAlias := map[core.LocalName]struct{}{}
		for _, r := range a.Databases {
			if _, exists := seenAlias[r.Alias]; exists {
				out = append(out, errd("DUPLICATE_DATASOURCE_ALIAS", "application %s defines datasource alias %s more than once", a.ID, r.Alias))
			}
			seenAlias[r.Alias] = struct{}{}
			if !r.Access.Valid() {
				out = append(out, errFor(subject.Subject{Kind: "application", ID: string(a.ID)}, "INVALID_DATABASE_ACCESS", "application %s datasource %s has invalid access mode %q", a.ID, r.Alias, r.Access))
			}
		}
	}

	for _, a := range w.Applications {
		for _, ns := range a.SemanticUses {
			if _, ok := semanticNamespaces[ns]; !ok {
				out = append(out, errd("UNKNOWN_APPLICATION_SEMANTIC_NAMESPACE", "application %s uses unknown semantic namespace %s", a.ID, ns))
			}
		}
	}

	for _, d := range w.Databases {
		if _, exists := dbs[d.ID]; exists {
			out = append(out, errd("DUPLICATE_DATABASE", "duplicate database %s", d.ID))
		}
		dbs[d.ID] = d
		if !d.Ownership.Valid() {
			out = append(out, errFor(subject.Subject{Kind: "database", ID: string(d.ID)}, "INVALID_DATABASE_OWNERSHIP", "database %s has invalid ownership %q", d.ID, d.Ownership))
		}
		semanticUseSet := map[core.CanonicalID]struct{}{}
		for _, ns := range d.SemanticUses {
			semanticUseSet[ns] = struct{}{}
			if _, ok := semanticNamespaces[ns]; !ok {
				out = append(out, errd("UNKNOWN_DATABASE_SEMANTIC_NAMESPACE", "database %s uses unknown semantic namespace %s", d.ID, ns))
			}
		}
		for _, c := range d.Contracts {
			if _, exists := contracts[c.ID]; exists {
				out = append(out, errd("DUPLICATE_CONTRACT", "duplicate database contract %s", c.ID))
			}
			contracts[c.ID] = c
			contractOwners[c.ID] = d.ID
			if !strings.HasPrefix(string(c.ID), string(d.ID)+".") {
				out = append(out, errFor(subject.Subject{Kind: "database-contract", ID: string(c.ID)}, "DATABASE_CONTRACT_NAMESPACE_MISMATCH", "contract %s is declared by database %s but is outside its namespace", c.ID, d.ID))
			}
			if c.Version.Major < 1 || c.Version.Minor < 0 {
				out = append(out, errd("INVALID_DATABASE_CONTRACT_VERSION", "contract %s has invalid version %d.%d", c.ID, c.Version.Major, c.Version.Minor))
			}
			for _, group := range []struct {
				relation string
				refs     []core.CanonicalID
			}{{"read", c.Reads}, {"write", c.Writes}} {
				for _, ref := range group.refs {
					if _, ok := semanticElements[ref]; !ok {
						out = append(out, errd("UNKNOWN_DATABASE_CONTRACT_SEMANTIC_REF", "contract %s %s refers to unknown semantic element %s", c.ID, group.relation, ref))
						continue
					}
					ns := core.CanonicalID(strings.SplitN(string(ref), ".", 2)[0])
					if _, ok := semanticUseSet[ns]; !ok {
						out = append(out, errd("DATABASE_CONTRACT_UNDECLARED_SEMANTIC_DEPENDENCY", "contract %s refers to semantic element %s but database %s does not declare uses semantic %s", c.ID, ref, d.ID, ns))
					}
				}
			}
		}
	}

	seenPersistence := map[core.CanonicalID]struct{}{}
	for _, m := range w.Persistence {
		if _, dup := seenPersistence[m.ID]; dup {
			out = append(out, errd("DUPLICATE_PERSISTENCE_MAPPING", "duplicate persistence mapping %s", m.ID))
		}
		seenPersistence[m.ID] = struct{}{}
		db, ok := dbs[m.Database]
		if !ok {
			out = append(out, errd("UNKNOWN_PERSISTENCE_DATABASE", "persistence mapping %s targets unknown database %s", m.ID, m.Database))
			continue
		}
		if !strings.HasPrefix(string(m.ID), string(m.Database)+".") {
			out = append(out, errd("PERSISTENCE_MAPPING_DATABASE_NAMESPACE_MISMATCH", "persistence mapping %s is owned by database %s but is outside its namespace", m.ID, m.Database))
		}
		if db.Kind != "relational" {
			out = append(out, errd("PERSISTENCE_MAPPING_NON_RELATIONAL_DATABASE", "persistence mapping %s targets non-relational database %s", m.ID, m.Database))
		}
		if _, ok := semanticData[m.SemanticData]; !ok {
			out = append(out, errd("UNKNOWN_PERSISTENCE_SEMANTIC_DATA", "persistence mapping %s refers to unknown semantic data %s", m.ID, m.SemanticData))
		}
		ns := core.CanonicalID(strings.SplitN(string(m.SemanticData), ".", 2)[0])
		declared := false
		for _, used := range db.SemanticUses {
			if used == ns {
				declared = true
				break
			}
		}
		if !declared {
			out = append(out, errd("PERSISTENCE_UNDECLARED_SEMANTIC_DEPENDENCY", "persistence mapping %s refers to %s but database %s does not declare uses semantic %s", m.ID, m.SemanticData, m.Database, ns))
		}
		if strings.TrimSpace(m.Table) == "" {
			out = append(out, errd("PERSISTENCE_TABLE_REQUIRED", "persistence mapping %s has no target table", m.ID))
		}
		seenColumns := map[string]struct{}{}
		seenAttrs := map[core.CanonicalID]struct{}{}
		for _, c := range m.Columns {
			if _, dup := seenAttrs[c.Attribute]; dup {
				out = append(out, errd("DUPLICATE_PERSISTENCE_ATTRIBUTE", "persistence mapping %s maps attribute %s more than once", m.ID, c.Attribute))
			}
			seenAttrs[c.Attribute] = struct{}{}
			if _, dup := seenColumns[strings.ToLower(c.Column)]; dup {
				out = append(out, errd("DUPLICATE_PERSISTENCE_COLUMN", "persistence mapping %s maps physical column %s more than once", m.ID, c.Column))
			}
			seenColumns[strings.ToLower(c.Column)] = struct{}{}
			if _, ok := semanticAttributeByID[c.Attribute]; !ok {
				out = append(out, errd("UNKNOWN_PERSISTENCE_ATTRIBUTE", "persistence mapping %s refers to unknown semantic attribute %s", m.ID, c.Attribute))
			} else if !strings.HasPrefix(string(c.Attribute), string(m.SemanticData)+".") {
				out = append(out, errd("PERSISTENCE_ATTRIBUTE_PARENT_MISMATCH", "persistence mapping %s attribute %s does not belong to semantic data %s", m.ID, c.Attribute, m.SemanticData))
			}
			if !c.Storage.Valid() {
				out = append(out, errd("INVALID_PERSISTENCE_STORAGE_TYPE", "persistence mapping %s attribute %s has invalid storage type %q", m.ID, c.Attribute, c.Storage))
			}
			if !c.Nullability.Valid() {
				out = append(out, errd("INVALID_PERSISTENCE_NULLABILITY", "persistence mapping %s attribute %s has invalid nullability %q", m.ID, c.Attribute, c.Nullability))
			}
			if strings.TrimSpace(c.Column) == "" {
				out = append(out, errd("PERSISTENCE_COLUMN_REQUIRED", "persistence mapping %s attribute %s has no physical column", m.ID, c.Attribute))
			}
		}
		for _, pk := range m.PrimaryKey {
			if _, ok := seenAttrs[pk]; !ok {
				out = append(out, errd("PERSISTENCE_PRIMARY_KEY_UNMAPPED_ATTRIBUTE", "persistence mapping %s primary key attribute %s is not mapped", m.ID, pk))
			}
		}
		seenIndexes := map[string]struct{}{}
		for _, idx := range m.Indexes {
			key := strings.ToLower(idx.Name)
			if _, dup := seenIndexes[key]; dup {
				out = append(out, errd("DUPLICATE_PERSISTENCE_INDEX", "persistence mapping %s index %s is declared more than once", m.ID, idx.Name))
			}
			seenIndexes[key] = struct{}{}
			for _, attr := range idx.Attributes {
				if _, ok := seenAttrs[attr]; !ok {
					out = append(out, errd("PERSISTENCE_INDEX_UNMAPPED_ATTRIBUTE", "persistence mapping %s index %s refers to unmapped attribute %s", m.ID, idx.Name, attr))
				}
			}
		}
	}

	seenMigrationProvider := map[core.CanonicalID]struct{}{}
	for _, cfg := range w.MigrationProviders {
		if _, dup := seenMigrationProvider[cfg.Database]; dup {
			out = append(out, errd("DUPLICATE_MIGRATION_PROVIDER", "database %s declares migration provider more than once", cfg.Database))
		}
		seenMigrationProvider[cfg.Database] = struct{}{}
		db, ok := dbs[cfg.Database]
		if !ok {
			out = append(out, errd("UNKNOWN_MIGRATION_PROVIDER_DATABASE", "migration provider targets unknown database %s", cfg.Database))
			continue
		}
		if db.Ownership != database.OwnershipManagedHere {
			out = append(out, errd("MIGRATION_PROVIDER_DATABASE_NOT_MANAGED_HERE", "migration provider for database %s requires managed-here ownership, got %s", cfg.Database, db.Ownership))
		}
		if !cfg.Tool.Valid() {
			out = append(out, errd("UNKNOWN_MIGRATION_TOOL", "database %s uses unknown migration tool %q", cfg.Database, cfg.Tool))
		}
		if !cfg.Mode.Valid() {
			out = append(out, errd("INVALID_MIGRATION_PROVIDER_MODE", "database %s uses invalid migration provider mode %q", cfg.Database, cfg.Mode))
		}
	}

	migrationIDs := map[core.CanonicalID]map[string]struct{}{}
	for _, m := range w.Migrations {
		db, ok := dbs[m.Database]
		if !ok {
			out = append(out, errd("UNKNOWN_MIGRATION_DATABASE", "migration %s targets unknown database %s", m.ID, m.Database))
			continue
		}
		if db.Ownership != database.OwnershipManagedHere {
			out = append(out, errd("MIGRATION_DATABASE_NOT_MANAGED_HERE", "migration %s targets database %s with ownership %s", m.ID, m.Database, db.Ownership))
		}
		if strings.TrimSpace(m.ID) == "" {
			out = append(out, errd("MIGRATION_ID_REQUIRED", "database %s has a migration with empty id", m.Database))
		}
		if _, ok := migrationIDs[m.Database]; !ok {
			migrationIDs[m.Database] = map[string]struct{}{}
		}
		if _, dup := migrationIDs[m.Database][m.ID]; dup {
			out = append(out, errd("DUPLICATE_MIGRATION", "database %s declares migration %s more than once", m.Database, m.ID))
		}
		migrationIDs[m.Database][m.ID] = struct{}{}
		if !m.Recovery.Valid() {
			out = append(out, errd("INVALID_MIGRATION_RECOVERY", "migration %s has invalid recovery strategy %q", m.ID, m.Recovery))
		}
		for i, step := range m.Steps {
			if !step.Kind.Valid() {
				out = append(out, errd("INVALID_MIGRATION_STEP_KIND", "migration %s step %d has invalid kind %q", m.ID, i+1, step.Kind))
			}
			if step.SQL != nil {
				if !step.SQL.Provider.Known() || step.SQL.Provider == environment.ProviderGeneric {
					out = append(out, errd("UNKNOWN_MIGRATION_SQL_PROVIDER", "migration %s step %d uses unsupported SQL provider %q", m.ID, i+1, step.SQL.Provider))
				}
				if strings.TrimSpace(step.SQL.Source) == "" {
					out = append(out, errd("MIGRATION_SQL_REQUIRED", "migration %s step %d has empty SQL", m.ID, i+1))
				}
			}
		}
	}
	for _, m := range w.Migrations {
		ids := migrationIDs[m.Database]
		for _, parent := range m.Parents {
			if _, ok := ids[parent]; !ok {
				out = append(out, errd("UNKNOWN_MIGRATION_PARENT", "migration %s parent %s does not exist in database %s", m.ID, parent, m.Database))
			}
		}
	}

	applicationReleaseVersions := map[core.CanonicalID]map[string]struct{}{}
	for _, r := range w.ApplicationReleases {
		if !core.ValidNumericVersion(r.Version) {
			out = append(out, errd("INVALID_APPLICATION_RELEASE_VERSION", "application %s release version %q must be numeric major.minor.patch", r.Application, r.Version))
		}
		if _, ok := applicationReleaseVersions[r.Application]; !ok {
			applicationReleaseVersions[r.Application] = map[string]struct{}{}
		}
		if _, dup := applicationReleaseVersions[r.Application][r.Version]; dup {
			out = append(out, errd("DUPLICATE_APPLICATION_RELEASE", "application %s declares release %s more than once", r.Application, r.Version))
		}
		applicationReleaseVersions[r.Application][r.Version] = struct{}{}
		if _, ok := apps[r.Application]; !ok {
			out = append(out, warnd("APPLICATION_RELEASE_NOT_IN_CURRENT_DEFINITION", "application release %s@%s has no current application definition; historical release remains inspectable", r.Application, r.Version))
		}
		seenAlias := map[core.LocalName]struct{}{}
		for _, req := range r.Databases {
			if _, dup := seenAlias[req.Alias]; dup {
				out = append(out, errd("DUPLICATE_APPLICATION_RELEASE_DATASOURCE_ALIAS", "application release %s@%s defines datasource alias %s more than once", r.Application, r.Version, req.Alias))
			}
			seenAlias[req.Alias] = struct{}{}
			if !req.Access.Valid() {
				out = append(out, errd("INVALID_APPLICATION_RELEASE_DATABASE_ACCESS", "application release %s@%s datasource %s has invalid access mode %q", r.Application, r.Version, req.Alias, req.Access))
			}
			if _, ok := dbs[req.TargetDatabase]; !ok {
				out = append(out, warnd("APPLICATION_RELEASE_DATABASE_NOT_CURRENT", "application release %s@%s targets database %s which has no current definition", r.Application, r.Version, req.TargetDatabase))
			}
			seenContract := map[core.CanonicalID]struct{}{}
			for _, cr := range req.RequiredContracts {
				if _, dup := seenContract[cr.ContractID]; dup {
					out = append(out, errd("DUPLICATE_APPLICATION_RELEASE_CONTRACT_REQUIREMENT", "application release %s@%s datasource %s requires contract %s more than once", r.Application, r.Version, req.Alias, cr.ContractID))
				}
				seenContract[cr.ContractID] = struct{}{}
				if cr.CompatibleMajor < 1 {
					out = append(out, errd("INVALID_APPLICATION_RELEASE_CONTRACT_VERSION", "application release %s@%s requires %s with invalid compatible major %d", r.Application, r.Version, cr.ContractID, cr.CompatibleMajor))
				}
				if _, ok := contracts[cr.ContractID]; !ok {
					out = append(out, warnd("APPLICATION_RELEASE_CONTRACT_NOT_CURRENT", "application release %s@%s requires historical contract %s which has no current definition", r.Application, r.Version, cr.ContractID))
					continue
				}
				if owner := contractOwners[cr.ContractID]; owner != req.TargetDatabase {
					out = append(out, warnd("APPLICATION_RELEASE_CONTRACT_CURRENT_OWNER_MISMATCH", "application release %s@%s recorded %s on database %s; current definition is owned by %s", r.Application, r.Version, cr.ContractID, req.TargetDatabase, owner))
				}
			}
		}
	}

	releaseVersions := map[core.CanonicalID]map[string]struct{}{}
	for _, r := range w.DatabaseReleases {
		db, ok := dbs[r.Database]
		if !ok {
			out = append(out, errd("UNKNOWN_DATABASE_RELEASE_DATABASE", "database release %s targets unknown database %s", r.Version, r.Database))
			continue
		}
		if db.Ownership != database.OwnershipManagedHere {
			out = append(out, errd("DATABASE_RELEASE_NOT_MANAGED_HERE", "database release %s targets database %s with ownership %s", r.Version, r.Database, db.Ownership))
		}
		if !release.ValidVersion(r.Version) {
			out = append(out, errd("INVALID_DATABASE_RELEASE_VERSION", "database %s release version %q must be numeric major.minor.patch", r.Database, r.Version))
		}
		if _, ok := releaseVersions[r.Database]; !ok {
			releaseVersions[r.Database] = map[string]struct{}{}
		}
		if _, dup := releaseVersions[r.Database][r.Version]; dup {
			out = append(out, errd("DUPLICATE_DATABASE_RELEASE", "database %s declares release %s more than once", r.Database, r.Version))
		}
		releaseVersions[r.Database][r.Version] = struct{}{}
		if !r.DataReadiness.Valid() {
			out = append(out, errd("INVALID_DATABASE_RELEASE_DATA_READINESS", "database %s release %s has invalid data readiness %q", r.Database, r.Version, r.DataReadiness))
		}
		seenMigrations := map[string]struct{}{}
		for _, mid := range r.Migrations {
			if _, dup := seenMigrations[mid]; dup {
				out = append(out, errd("DUPLICATE_DATABASE_RELEASE_MIGRATION", "database %s release %s includes migration %s more than once", r.Database, r.Version, mid))
			}
			seenMigrations[mid] = struct{}{}
			if _, ok := migrationIDs[r.Database][mid]; !ok {
				out = append(out, errd("UNKNOWN_DATABASE_RELEASE_MIGRATION", "database %s release %s includes unknown migration %s", r.Database, r.Version, mid))
			}
		}
		seenContracts := map[core.CanonicalID]struct{}{}
		for _, pc := range r.Provides {
			if _, dup := seenContracts[pc.ID]; dup {
				out = append(out, errd("DUPLICATE_DATABASE_RELEASE_CONTRACT", "database %s release %s provides contract %s more than once", r.Database, r.Version, pc.ID))
			}
			seenContracts[pc.ID] = struct{}{}
			c, ok := contracts[pc.ID]
			if !ok {
				out = append(out, warnd("DATABASE_RELEASE_CONTRACT_NOT_CURRENT", "database %s release %s provides historical contract %s which has no current definition", r.Database, r.Version, pc.ID))
			} else {
				if contractOwners[pc.ID] != r.Database {
					out = append(out, warnd("DATABASE_RELEASE_CONTRACT_CURRENT_OWNER_MISMATCH", "database %s release %s recorded contract %s; current owner is %s", r.Database, r.Version, pc.ID, contractOwners[pc.ID]))
				}
				if pc.Version.Major > c.Version.Major || (pc.Version.Major == c.Version.Major && pc.Version.Minor > c.Version.Minor) {
					out = append(out, warnd("DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION", "database %s release %s provides %s v%d.%d while current definition is v%d.%d", r.Database, r.Version, pc.ID, pc.Version.Major, pc.Version.Minor, c.Version.Major, c.Version.Minor))
				}
			}
			if pc.Version.Major < 1 || pc.Version.Minor < 0 {
				out = append(out, errd("INVALID_DATABASE_RELEASE_CONTRACT_VERSION", "database %s release %s has invalid contract version for %s", r.Database, r.Version, pc.ID))
			}
		}
	}

	solutionVersions := map[core.CanonicalID]map[string]struct{}{}
	for _, r := range w.SolutionReleases {
		if !core.ValidNumericVersion(r.Version) {
			out = append(out, errd("INVALID_SOLUTION_RELEASE_VERSION", "solution release %s version %q must be numeric major.minor.patch", r.ID, r.Version))
		}
		if _, ok := solutionVersions[r.ID]; !ok {
			solutionVersions[r.ID] = map[string]struct{}{}
		}
		if _, dup := solutionVersions[r.ID][r.Version]; dup {
			out = append(out, errd("DUPLICATE_SOLUTION_RELEASE", "solution release %s@%s is declared more than once", r.ID, r.Version))
		}
		solutionVersions[r.ID][r.Version] = struct{}{}
		stepIDs := map[core.LocalName]struct{}{}
		for _, st := range r.Steps {
			if _, dup := stepIDs[st.ID]; dup {
				out = append(out, errd("DUPLICATE_SOLUTION_RELEASE_STEP", "solution release %s@%s defines step %s more than once", r.ID, r.Version, st.ID))
			}
			stepIDs[st.ID] = struct{}{}
			if !st.Kind.Valid() {
				out = append(out, errd("INVALID_SOLUTION_RELEASE_STEP_KIND", "solution release %s@%s step %s has invalid kind %q", r.ID, r.Version, st.ID, st.Kind))
			}
			if !st.Phase.Valid() {
				out = append(out, errd("INVALID_SOLUTION_RELEASE_PHASE", "solution release %s@%s step %s has invalid phase %q", r.ID, r.Version, st.ID, st.Phase))
			}
			if !st.RolloutStrategy.Valid() {
				out = append(out, errd("INVALID_ROLLOUT_STRATEGY", "solution release %s@%s step %s has invalid rollout strategy %q", r.ID, r.Version, st.ID, st.RolloutStrategy))
			}
			if len(st.TrafficSteps) > 0 || st.RolloutStrategy != "" {
				if st.Kind != releaseplan.StepApplicationRelease {
					out = append(out, errd("ROLLOUT_POLICY_ON_NON_APPLICATION_STEP", "solution release %s@%s step %s has rollout policy but is %s", r.ID, r.Version, st.ID, st.Kind))
				} else {
					prev := -1
					for _, bps := range st.EffectiveTrafficSteps() {
						if bps <= prev || bps < 0 || bps > 10000 {
							out = append(out, errd("INVALID_ROLLOUT_TRAFFIC_SEQUENCE", "solution release %s@%s step %s traffic sequence must be strictly increasing within 0..10000", r.ID, r.Version, st.ID))
							break
						}
						prev = bps
					}
					steps := st.EffectiveTrafficSteps()
					if len(steps) == 0 || steps[len(steps)-1] != 10000 {
						out = append(out, errd("ROLLOUT_TRAFFIC_MUST_END_AT_FULL", "solution release %s@%s step %s rollout must end at 10000 basis points", r.ID, r.Version, st.ID))
					}
				}
			}
			switch st.Kind {
			case releaseplan.StepDatabaseRelease:
				if st.Database == "" || st.Version == "" {
					out = append(out, errd("INCOMPLETE_DATABASE_RELEASE_STEP", "solution release %s@%s step %s requires database and version", r.ID, r.Version, st.ID))
				}
			case releaseplan.StepApplicationRelease:
				if st.Application == "" || st.Version == "" {
					out = append(out, errd("INCOMPLETE_APPLICATION_RELEASE_STEP", "solution release %s@%s step %s requires application and version", r.ID, r.Version, st.ID))
				}
			case releaseplan.StepApplicationRetire:
				if st.Application == "" || st.Version == "" {
					out = append(out, errd("INCOMPLETE_APPLICATION_RETIRE_STEP", "solution release %s@%s step %s requires application and version", r.ID, r.Version, st.ID))
				}
			case releaseplan.StepDatabaseGate:
				if st.Database == "" || st.Contract == "" || st.CompatibleMajor < 1 {
					out = append(out, errd("INCOMPLETE_DATABASE_GATE_STEP", "solution release %s@%s step %s requires database, contract, and compatible major", r.ID, r.Version, st.ID))
				}
			}
		}
		for _, dep := range r.After {
			if _, ok := stepIDs[dep.Step]; !ok {
				out = append(out, errd("UNKNOWN_SOLUTION_RELEASE_STEP", "solution release %s@%s ordering refers to unknown step %s", r.ID, r.Version, dep.Step))
			}
			if _, ok := stepIDs[dep.After]; !ok {
				out = append(out, errd("UNKNOWN_SOLUTION_RELEASE_PREREQUISITE", "solution release %s@%s ordering refers to unknown prerequisite %s", r.ID, r.Version, dep.After))
			}
		}
		c := releaseplan.Compile(r, w.ApplicationReleases, w.DatabaseReleases)
		if !c.Executable {
			for _, f := range c.Findings {
				out = append(out, errd("SOLUTION_RELEASE_NOT_EXECUTABLE", "solution release %s@%s: %s", r.ID, r.Version, f))
			}
		}
	}

	for _, e := range w.Environments {
		if _, exists := envs[e.ID]; exists {
			out = append(out, errd("DUPLICATE_ENVIRONMENT", "duplicate environment %s", e.ID))
		}
		envs[e.ID] = e
	}

	for _, a := range w.Applications {
		for _, r := range a.Databases {
			if _, ok := dbs[r.TargetDatabase]; !ok {
				out = append(out, errd("UNKNOWN_DATABASE", "application %s datasource %s targets unknown database %s", a.ID, r.Alias, r.TargetDatabase))
				continue
			}
			for _, cr := range r.RequiredContracts {
				c, ok := contracts[cr.ContractID]
				if !ok {
					out = append(out, errd("UNKNOWN_DATABASE_CONTRACT", "application %s requires unknown contract %s", a.ID, cr.ContractID))
					continue
				}
				if owner := contractOwners[cr.ContractID]; owner != r.TargetDatabase {
					out = append(out, errd("DATABASE_CONTRACT_TARGET_MISMATCH", "application %s datasource %s targets database %s but contract %s is provided by %s", a.ID, r.Alias, r.TargetDatabase, cr.ContractID, owner))
				}
				if cr.CompatibleMajor < 1 {
					out = append(out, errd("INVALID_CONTRACT_REQUIREMENT_VERSION", "application %s requires %s with invalid compatible major %d", a.ID, cr.ContractID, cr.CompatibleMajor))
				} else if c.Version.Major != cr.CompatibleMajor {
					out = append(out, errd("DATABASE_CONTRACT_VERSION_UNSATISFIED", "application %s requires %s compatible major = %d but provider has %d", a.ID, cr.ContractID, cr.CompatibleMajor, c.Version.Major))
				}
			}
		}
	}

	for _, e := range w.Environments {
		resolvedPlugins, resolveErr := pluginCatalog.ResolveEnvironment(e)
		seenResourceBindings := map[string]struct{}{}
		for _, rb := range e.ResourceBindings {
			if err := rb.Validate(); err != nil {
				out = append(out, errd("INVALID_ENVIRONMENT_RESOURCE_BINDING", "environment %s: %v", e.ID, err))
			}
			if _, dup := seenResourceBindings[rb.Key()]; dup {
				out = append(out, errd("DUPLICATE_ENVIRONMENT_RESOURCE_BINDING", "environment %s maps resource scope %s more than once", e.ID, rb.Key()))
			}
			seenResourceBindings[rb.Key()] = struct{}{}
		}
		if resolveErr != nil {
			out = append(out, errd("PLUGIN_RESOLUTION_FAILED", "environment %s: %v", e.ID, resolveErr))
		}
		seenLeasePolicy := map[string]struct{}{}
		for _, lp := range e.LeasePolicies {
			key := lp.CapabilityKind + "|" + string(lp.CapabilityID)
			if _, dup := seenLeasePolicy[key]; dup {
				out = append(out, errd("DUPLICATE_ENVIRONMENT_LEASE_POLICY", "environment %s declares lease policy for %s more than once", e.ID, key))
			}
			seenLeasePolicy[key] = struct{}{}
			if err := lp.Policy.Validate(); err != nil {
				out = append(out, errd("INVALID_ENVIRONMENT_LEASE_POLICY", "environment %s capability %s: %v", e.ID, lp.CapabilityID, err))
			}
			ck := plugin.CapabilityKind(lp.CapabilityKind)
			if !ck.Valid() {
				out = append(out, errd("INVALID_ENVIRONMENT_LEASE_POLICY_CAPABILITY_KIND", "environment %s lease policy for %s uses invalid capability kind %s", e.ID, lp.CapabilityID, lp.CapabilityKind))
			} else if resolveErr == nil {
				if _, ok := resolvedPlugins.Capability(ck, lp.CapabilityID); !ok {
					out = append(out, errd("LEASE_POLICY_CAPABILITY_NOT_RESOLVED", "environment %s lease policy targets unresolved capability %s %s", e.ID, ck, lp.CapabilityID))
				}
			}
		}
		seenGrant := map[string]struct{}{}
		for _, g := range e.PermissionGrants {
			if err := g.Permission.Validate(); err != nil {
				out = append(out, errd("INVALID_ENVIRONMENT_PERMISSION_GRANT", "environment %s capability %s: %v", e.ID, g.CapabilityID, err))
			}
			resourceKey := string(g.Permission.Kind) + "|" + g.Permission.Scope
			if _, ok := seenResourceBindings[resourceKey]; !ok {
				out = append(out, errd("PERMISSION_RESOURCE_BINDING_MISSING", "environment %s grants permission %s to capability %s but no resource binding exists for %s", e.ID, g.Permission.Key(), g.CapabilityID, resourceKey))
			}
			capKind := plugin.CapabilityKind(g.CapabilityKind)
			if !capKind.Valid() {
				out = append(out, errd("INVALID_ENVIRONMENT_PERMISSION_CAPABILITY_KIND", "environment %s grant for %s uses invalid capability kind %s", e.ID, g.CapabilityID, g.CapabilityKind))
			}
			key := g.CapabilityKind + "|" + string(g.CapabilityID) + "|" + g.Permission.Key()
			if _, dup := seenGrant[key]; dup {
				out = append(out, errd("DUPLICATE_ENVIRONMENT_PERMISSION_GRANT", "environment %s grants %s more than once", e.ID, key))
			}
			seenGrant[key] = struct{}{}
			if resolveErr == nil && capKind.Valid() {
				rc, ok := resolvedPlugins.Capability(capKind, g.CapabilityID)
				if !ok {
					out = append(out, errd("PERMISSION_GRANT_CAPABILITY_NOT_RESOLVED", "environment %s grants permission to unresolved capability %s %s", e.ID, capKind, g.CapabilityID))
				} else {
					found := false
					for _, req := range rc.Capability.Permissions {
						if req.Key() == g.Permission.Key() {
							found = true
							break
						}
					}
					if !found {
						out = append(out, errd("ENVIRONMENT_PERMISSION_OVERGRANT", "environment %s grants %s to capability %s but plugin does not request it", e.ID, g.Permission.Key(), g.CapabilityID))
					}
				}
			}
		}
		checkCapabilityPermissions := func(rc plugin.ResolvedCapability, usage string) {
			granted := map[string]struct{}{}
			for _, g := range e.PermissionGrants {
				if g.CapabilityKind == string(rc.Capability.Kind) && g.CapabilityID == rc.Capability.ID {
					granted[g.Permission.Key()] = struct{}{}
				}
			}
			for _, req := range rc.Capability.Permissions {
				if _, ok := granted[req.Key()]; !ok {
					out = append(out, errd("CAPABILITY_PERMISSION_NOT_GRANTED", "environment %s %s uses capability %s %s which requires permission %s", e.ID, usage, rc.Capability.Kind, rc.Capability.ID, req.Key()))
				}
			}
		}
		seenPluginReq := map[core.CanonicalID]struct{}{}
		for _, req := range e.Plugins {
			if _, dup := seenPluginReq[req.ID]; dup {
				out = append(out, errd("DUPLICATE_ENVIRONMENT_PLUGIN_REQUIREMENT", "environment %s requires plugin %s more than once", e.ID, req.ID))
			}
			seenPluginReq[req.ID] = struct{}{}
			if req.CompatibleMajor < 1 {
				out = append(out, errd("INVALID_ENVIRONMENT_PLUGIN_VERSION", "environment %s plugin %s has invalid major %d", e.ID, req.ID, req.CompatibleMajor))
			}
		}
		seenBundleReq := map[core.CanonicalID]struct{}{}
		for _, req := range e.Bundles {
			if _, dup := seenBundleReq[req.ID]; dup {
				out = append(out, errd("DUPLICATE_ENVIRONMENT_BUNDLE_REQUIREMENT", "environment %s requires bundle %s more than once", e.ID, req.ID))
			}
			seenBundleReq[req.ID] = struct{}{}
			if req.CompatibleMajor < 1 {
				out = append(out, errd("INVALID_ENVIRONMENT_BUNDLE_VERSION", "environment %s bundle %s has invalid major %d", e.ID, req.ID, req.CompatibleMajor))
			}
		}
		seenFeatures := map[environment.Feature]struct{}{}
		for _, feature := range e.Features {
			if _, dup := seenFeatures[feature]; dup {
				out = append(out, errd("DUPLICATE_ENVIRONMENT_FEATURE", "environment %s enables feature %s more than once", e.ID, feature))
			}
			seenFeatures[feature] = struct{}{}
			if !feature.Known() {
				out = append(out, errd("UNKNOWN_ENVIRONMENT_FEATURE", "environment %s enables unknown feature %s", e.ID, feature))
			}
			if resolveErr == nil && feature == environment.FeatureSQLAnalysis {
				providers := 0
				for _, rc := range resolvedPlugins.Capabilities {
					if rc.Capability.Kind == plugin.CapabilityEditorIntelligence && rc.Capability.SupportsFeature(feature) {
						providers++
					}
				}
				if providers == 0 {
					out = append(out, errd("ENVIRONMENT_FEATURE_PLUGIN_MISSING", "environment %s feature %s has no resolved editor-intelligence plugin capability", e.ID, feature))
				}
				if providers > 1 {
					out = append(out, errd("ENVIRONMENT_FEATURE_PLUGIN_AMBIGUOUS", "environment %s feature %s is provided by %d resolved plugin capabilities", e.ID, feature, providers))
				}
				if providers == 1 {
					for _, rc := range resolvedPlugins.Capabilities {
						if rc.Capability.Kind == plugin.CapabilityEditorIntelligence && rc.Capability.SupportsFeature(feature) {
							checkCapabilityPermissions(rc, "feature "+string(feature))
							break
						}
					}
				}
			}
		}
		instanceIDs := map[core.CanonicalID]struct{}{}
		for _, inst := range e.Instances {
			if _, dup := instanceIDs[inst.ID]; dup {
				out = append(out, errd("DUPLICATE_DATABASE_INSTANCE", "environment %s defines database instance %s more than once", e.ID, inst.ID))
			}
			instanceIDs[inst.ID] = struct{}{}
			if _, ok := dbs[inst.Definition]; !ok {
				out = append(out, errd("UNKNOWN_DATABASE_INSTANCE_DEFINITION", "environment %s instance %s refers to unknown database %s", e.ID, inst.ID, inst.Definition))
			}
			if !inst.Provider.Known() || inst.Provider == environment.ProviderGeneric {
				out = append(out, errd("UNKNOWN_DATABASE_PROVIDER", "environment %s instance %s uses unsupported provider %q", e.ID, inst.ID, inst.Provider))
			}
			if resolveErr == nil && inst.Provider.Known() && inst.Provider != environment.ProviderGeneric {
				providers := 0
				for _, rc := range resolvedPlugins.Capabilities {
					if rc.Capability.Kind == plugin.CapabilityDatabaseProvider && rc.Capability.SupportsDatabaseProvider(inst.Provider) {
						providers++
					}
				}
				if providers == 0 {
					out = append(out, errd("DATABASE_PROVIDER_PLUGIN_MISSING", "environment %s instance %s provider %s has no resolved database-provider plugin", e.ID, inst.ID, inst.Provider))
				}
				if providers > 1 {
					out = append(out, errd("DATABASE_PROVIDER_PLUGIN_AMBIGUOUS", "environment %s instance %s provider %s is provided by %d resolved plugins", e.ID, inst.ID, inst.Provider, providers))
				}
				if providers == 1 {
					for _, rc := range resolvedPlugins.Capabilities {
						if rc.Capability.Kind == plugin.CapabilityDatabaseProvider && rc.Capability.SupportsDatabaseProvider(inst.Provider) {
							checkCapabilityPermissions(rc, "database instance "+string(inst.ID))
							break
						}
					}
				}
			}
			if resolveErr == nil {
				for _, mp := range w.MigrationProviders {
					if mp.Database != inst.Definition || mp.Tool == "native" {
						continue
					}
					providers := 0
					for _, rc := range resolvedPlugins.Capabilities {
						if rc.Capability.Kind == plugin.CapabilityMigrationProvider && rc.Capability.SupportsMigrationTool(mp.Tool) {
							providers++
						}
					}
					if providers == 0 {
						out = append(out, errd("MIGRATION_PROVIDER_PLUGIN_MISSING", "environment %s database %s migration tool %s has no resolved migration-provider plugin", e.ID, inst.Definition, mp.Tool))
					}
					if providers > 1 {
						out = append(out, errd("MIGRATION_PROVIDER_PLUGIN_AMBIGUOUS", "environment %s database %s migration tool %s is provided by %d resolved plugins", e.ID, inst.Definition, mp.Tool, providers))
					}
					if providers == 1 {
						for _, rc := range resolvedPlugins.Capabilities {
							if rc.Capability.Kind == plugin.CapabilityMigrationProvider && rc.Capability.SupportsMigrationTool(mp.Tool) {
								checkCapabilityPermissions(rc, "migration provider "+string(mp.Tool))
								break
							}
						}
					}
				}
			}
		}

		seenExecutors := map[string]struct{}{}
		for _, b := range e.DeploymentExecutors {
			if !b.Action.Valid() {
				out = append(out, errd("UNKNOWN_DEPLOYMENT_ACTION_KIND", "environment %s binds unsupported deployment action %s", e.ID, b.Action))
			}
			if b.Scope == "" {
				b.Scope = environment.ExecutorScopeGlobal
			}
			if !b.Scope.Valid() {
				out = append(out, errd("UNKNOWN_DEPLOYMENT_EXECUTOR_SCOPE", "environment %s action %s has invalid executor scope %s", e.ID, b.Action, b.Scope))
			}
			if b.Scope == environment.ExecutorScopeGlobal && b.Target != "" {
				out = append(out, errd("GLOBAL_EXECUTOR_BINDING_HAS_TARGET", "environment %s action %s global binding cannot target %s", e.ID, b.Action, b.Target))
			}
			if b.Scope != environment.ExecutorScopeGlobal && b.Target == "" {
				out = append(out, errd("SCOPED_EXECUTOR_BINDING_TARGET_REQUIRED", "environment %s action %s scope %s requires a target", e.ID, b.Action, b.Scope))
			}
			if b.Executor == "" {
				out = append(out, errd("DEPLOYMENT_EXECUTOR_ID_REQUIRED", "environment %s action %s has no executor id", e.ID, b.Action))
			}
			if resolveErr == nil && b.Executor != "" {
				rc, ok := resolvedPlugins.Capability(plugin.CapabilityDeploymentExecutor, b.Executor)
				if !ok {
					out = append(out, errd("DEPLOYMENT_EXECUTOR_PLUGIN_MISSING_OR_AMBIGUOUS", "environment %s executor %s is not provided uniquely by resolved plugins", e.ID, b.Executor))
				} else if !rc.Capability.SupportsAction(b.Action) {
					out = append(out, errd("DEPLOYMENT_EXECUTOR_CAPABILITY_UNSUPPORTED", "environment %s executor %s does not support action %s", e.ID, b.Executor, b.Action))
				} else {
					checkCapabilityPermissions(rc, "deployment executor "+string(b.Executor))
				}
			}
			key := fmt.Sprintf("%s|%s|%s", b.Action, b.Scope, b.Target)
			if _, dup := seenExecutors[key]; dup {
				out = append(out, errd("DUPLICATE_DEPLOYMENT_EXECUTOR_BINDING", "environment %s binds deployment action %s scope %s target %s more than once", e.ID, b.Action, b.Scope, b.Target))
			}
			seenExecutors[key] = struct{}{}
		}
		seenObservers := map[deployment.ObservationKind]struct{}{}
		for _, b := range e.ObservationProviders {
			if !b.Kind.Valid() {
				out = append(out, errd("UNKNOWN_OBSERVATION_KIND", "environment %s binds unsupported observation kind %s", e.ID, b.Kind))
			}
			if b.Provider == "" {
				out = append(out, errd("OBSERVATION_PROVIDER_ID_REQUIRED", "environment %s observation kind %s has no provider id", e.ID, b.Kind))
			}
			if resolveErr == nil && b.Provider != "" {
				rc, ok := resolvedPlugins.Capability(plugin.CapabilityObservationProvider, b.Provider)
				if !ok {
					out = append(out, errd("OBSERVATION_PROVIDER_PLUGIN_MISSING_OR_AMBIGUOUS", "environment %s observation provider %s is not provided uniquely by resolved plugins", e.ID, b.Provider))
				} else if !rc.Capability.SupportsObservation(b.Kind) {
					out = append(out, errd("OBSERVATION_PROVIDER_CAPABILITY_UNSUPPORTED", "environment %s provider %s does not support observation %s", e.ID, b.Provider, b.Kind))
				} else {
					checkCapabilityPermissions(rc, "observation provider "+string(b.Provider))
				}
			}
			if _, dup := seenObservers[b.Kind]; dup {
				out = append(out, errd("DUPLICATE_OBSERVATION_PROVIDER_BINDING", "environment %s binds observation kind %s more than once", e.ID, b.Kind))
			}
			seenObservers[b.Kind] = struct{}{}
		}
		seenEvidence := map[deployment.EvidenceKind]struct{}{}
		for _, b := range e.EvidenceProviders {
			if !b.Kind.Valid() {
				out = append(out, errd("UNKNOWN_EVIDENCE_KIND", "environment %s binds unsupported evidence kind %s", e.ID, b.Kind))
			}
			if b.Provider == "" {
				out = append(out, errd("EVIDENCE_PROVIDER_ID_REQUIRED", "environment %s evidence kind %s has no provider id", e.ID, b.Kind))
			}
			if resolveErr == nil && b.Provider != "" {
				rc, ok := resolvedPlugins.Capability(plugin.CapabilityEvidenceProvider, b.Provider)
				if !ok {
					out = append(out, errd("EVIDENCE_PROVIDER_PLUGIN_MISSING_OR_AMBIGUOUS", "environment %s evidence provider %s is not provided uniquely by resolved plugins", e.ID, b.Provider))
				} else if !rc.Capability.SupportsEvidence(b.Kind) {
					out = append(out, errd("EVIDENCE_PROVIDER_CAPABILITY_UNSUPPORTED", "environment %s provider %s does not support evidence %s", e.ID, b.Provider, b.Kind))
				} else {
					checkCapabilityPermissions(rc, "evidence provider "+string(b.Provider))
				}
			}
			if _, dup := seenEvidence[b.Kind]; dup {
				out = append(out, errd("DUPLICATE_EVIDENCE_PROVIDER_BINDING", "environment %s binds evidence kind %s more than once", e.ID, b.Kind))
			}
			seenEvidence[b.Kind] = struct{}{}
		}

		seenBindings := map[string]struct{}{}
		for _, b := range e.Bindings {
			key := fmt.Sprintf("%s/%s", b.Application, b.Alias)
			if _, dup := seenBindings[key]; dup {
				out = append(out, errd("DUPLICATE_ENVIRONMENT_BINDING", "environment %s binds %s.%s more than once", e.ID, b.Application, b.Alias))
			}
			seenBindings[key] = struct{}{}

			app, ok := apps[b.Application]
			if !ok {
				out = append(out, errd("UNKNOWN_BINDING_APPLICATION", "environment %s binding refers to unknown application %s", e.ID, b.Application))
				continue
			}
			if !app.HasDatabaseAlias(b.Alias) {
				out = append(out, errd("UNKNOWN_BINDING_DATASOURCE_ALIAS", "environment %s binds unknown datasource alias %s.%s", e.ID, b.Application, b.Alias))
				continue
			}
			inst, ok := e.InstanceByID(b.Instance)
			if !ok {
				out = append(out, errd("UNKNOWN_DATABASE_INSTANCE", "environment %s binding %s.%s points to unknown instance %s", e.ID, b.Application, b.Alias, b.Instance))
				continue
			}
			req, ok := app.DatabaseByAliasExact(b.Alias)
			if !ok {
				continue
			}
			if inst.Definition != req.TargetDatabase {
				out = append(out, errd("DATABASE_BINDING_TARGET_MISMATCH", "environment %s binds %s.%s to database %s, expected %s", e.ID, b.Application, b.Alias, inst.Definition, req.TargetDatabase))
			}
		}
	}

	seenQueries := map[core.CanonicalID]struct{}{}
	for _, q := range w.Queries {
		if _, dup := seenQueries[q.ID]; dup {
			out = append(out, errd("DUPLICATE_QUERY", "duplicate query %s", q.ID))
		}
		seenQueries[q.ID] = struct{}{}
		a, ok := apps[q.Application]
		if !ok {
			out = append(out, errd("UNKNOWN_QUERY_APPLICATION", "query %s belongs to unknown application %s", q.ID, q.Application))
			continue
		}
		if !strings.HasPrefix(string(q.ID), string(q.Application)+".") {
			out = append(out, errd("QUERY_UNIT_NAMESPACE_MISMATCH", "query %s is declared in application %s but is outside its namespace", q.ID, q.Application))
		}
		if !a.HasDatabaseAlias(q.Datasource) {
			out = append(out, errd("UNKNOWN_DATASOURCE_ALIAS", "query %s uses unknown datasource alias %s", q.ID, q.Datasource))
			continue
		}
		if !q.Dialect.Known() {
			out = append(out, errd("UNKNOWN_SQL_DIALECT", "query %s uses unsupported SQL dialect %q", q.ID, q.Dialect))
		}

		bound := false
		for _, e := range envs {
			for _, b := range e.Bindings {
				if b.Application != a.ID || b.Alias != q.Datasource {
					continue
				}
				bound = true
				inst, ok := e.InstanceByID(b.Instance)
				if !ok {
					continue
				}
				if q.Dialect != environment.ProviderGeneric && inst.Provider != q.Dialect {
					out = append(out, errd("SQL_DIALECT_TARGET_MISMATCH", "query %s uses %s SQL but environment %s binds datasource %s to %s", q.ID, q.Dialect, e.ID, q.Datasource, inst.Provider))
				}
			}
		}
		if !bound {
			out = append(out, Diagnostic{Code: "UNBOUND_DATASOURCE", Severity: Warning, Message: fmt.Sprintf("query %s datasource %s is not bound in any environment", q.ID, q.Datasource)})
		}
	}

	seenBindings := map[string]struct{}{}
	for _, b := range w.Bindings {
		key := fmt.Sprintf("%s/%s", b.Application, b.Capability)
		if _, dup := seenBindings[key]; dup {
			out = append(out, errd("DUPLICATE_CUSTOM_GO_BINDING", "application %s declares custom Go realization for %s more than once", b.Application, b.Capability))
		}
		seenBindings[key] = struct{}{}
		if _, ok := apps[b.Application]; !ok {
			out = append(out, errd("UNKNOWN_REALIZATION_APPLICATION", "custom Go realization for %s belongs to unknown application %s", b.Capability, b.Application))
		}
	}

	return out
}

func HasErrors(ds []Diagnostic) bool {
	for _, d := range ds {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

func errd(code, format string, args ...any) Diagnostic {
	return Diagnostic{Code: code, Severity: Error, Message: fmt.Sprintf(format, args...)}
}

func errFor(s subject.Subject, code, format string, args ...any) Diagnostic {
	return Diagnostic{Code: code, Severity: Error, Message: fmt.Sprintf(format, args...), Subject: &s}
}

func warnd(code, format string, args ...any) Diagnostic {
	return Diagnostic{Code: code, Severity: Warning, Message: fmt.Sprintf(format, args...)}
}

func warnFor(s subject.Subject, code, format string, args ...any) Diagnostic {
	return Diagnostic{Code: code, Severity: Warning, Message: fmt.Sprintf(format, args...), Subject: &s}
}
