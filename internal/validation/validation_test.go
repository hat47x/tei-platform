package validation_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/migration"
	"tei/internal/database/persistence"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/permission"
	"tei/internal/plugin"
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/semantic"
	"tei/internal/stdlib"
	"tei/internal/validation"
)

func cid(s string) core.CanonicalID { return core.MustCanonicalID(s) }
func local(s string) core.LocalName { return core.MustLocalName(s) }

func referenceWorld() ir.World {
	salesContract := database.Contract{
		ID:      cid("sales_core.order"),
		Version: database.ContractVersion{Major: 4, Minor: 2},
		Reads:   []core.CanonicalID{cid("sales.order")},
		Writes:  []core.CanonicalID{cid("sales.order.status")},
		Queries: []database.QueryContract{{Name: "find_by_id"}},
	}
	masterContract := database.Contract{
		ID:      cid("master_data.customer"),
		Version: database.ContractVersion{Major: 2, Minor: 4},
		Reads:   []core.CanonicalID{cid("customer.customer")},
		Queries: []database.QueryContract{{Name: "find_by_id"}},
	}

	app := application.Application{
		ID:           cid("order_entry"),
		SemanticUses: []core.CanonicalID{cid("sales"), cid("customer")},
		Databases: []application.DatabaseRequirement{
			{
				Alias:             local("sales"),
				TargetDatabase:    cid("sales_core"),
				Access:            database.AccessReadWrite,
				RequiredContracts: []database.ContractRequirement{{ContractID: cid("sales_core.order"), CompatibleMajor: 4}},
			},
			{
				Alias:             local("master"),
				TargetDatabase:    cid("master_data"),
				Access:            database.AccessRead,
				RequiredContracts: []database.ContractRequirement{{ContractID: cid("master_data.customer"), CompatibleMajor: 2}},
			},
		},
	}

	env := environment.Environment{
		ID:       cid("development"),
		Plugins:  []environment.ExtensionRequirement{{ID: cid("tei.test.reference"), CompatibleMajor: 1}},
		Features: []environment.Feature{environment.FeatureSQLAnalysis, environment.FeatureDatabaseWorkbench},
		Instances: []environment.DatabaseInstance{
			{ID: cid("sales_core.dev"), Definition: cid("sales_core"), Provider: environment.ProviderPostgres},
			{ID: cid("master_data.dev"), Definition: cid("master_data"), Provider: environment.ProviderOracle},
		},
		Bindings: []environment.ApplicationDatabaseBinding{
			{Application: cid("order_entry"), Alias: local("sales"), Instance: cid("sales_core.dev")},
			{Application: cid("order_entry"), Alias: local("master"), Instance: cid("master_data.dev")},
		},
	}

	q := query.EmbeddedSQL{
		ID:          cid("order_entry.recent_orders"),
		Application: cid("order_entry"),
		Datasource:  local("sales"),
		Dialect:     environment.ProviderPostgres,
		Source:      "select order_id from orders",
	}

	return ir.World{
		Plugins: []plugin.Definition{{
			ID: cid("tei.test.reference"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
			Provides: []plugin.Capability{
				{Kind: plugin.CapabilityEditorIntelligence, ID: cid("editor.test"), Features: []environment.Feature{environment.FeatureSQLAnalysis, environment.FeatureDatabaseWorkbench}},
				{Kind: plugin.CapabilityDatabaseProvider, ID: cid("database.test"), DatabaseProviders: []environment.DatabaseProvider{environment.ProviderPostgres, environment.ProviderOracle}},
				{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.test"), Actions: []deployment.ActionKind{deployment.ActionApplicationStart}},
			},
		}},
		Semantics: []semantic.Namespace{
			stdlib.SemanticNamespace(),
			{ID: cid("sales"), Types: []semantic.Type{{ID: cid("sales.order_status"), Symbol: "OrderStatus", Kind: semantic.TypeAlias, Underlying: cid("std.code")}}, Data: []semantic.Data{{ID: cid("sales.order"), Symbol: "Order", Attributes: []semantic.Attribute{{ID: cid("sales.order.status"), Symbol: "status", TypeRef: cid("sales.order_status")}}, Relations: []semantic.Relation{{ID: cid("sales.order.customer"), Symbol: "customer", Target: cid("customer.customer"), Cardinality: semantic.CardinalityOne}}}}},
			{ID: cid("customer"), Data: []semantic.Data{{ID: cid("customer.customer"), Symbol: "Customer"}}},
		},
		Applications: []application.Application{app},
		Databases: []database.Database{
			{ID: cid("sales_core"), SemanticUses: []core.CanonicalID{cid("sales")}, Kind: "relational", Ownership: database.OwnershipManagedHere, Contracts: []database.Contract{salesContract}},
			{ID: cid("master_data"), SemanticUses: []core.CanonicalID{cid("customer")}, Kind: "relational", Ownership: database.OwnershipExternalObserved, Contracts: []database.Contract{masterContract}},
		},
		Environments: []environment.Environment{env},
		Queries:      []query.EmbeddedSQL{q},
	}
}

func TestReferenceWorldPasses(t *testing.T) {
	ds := validation.Validate(referenceWorld())
	for _, d := range ds {
		if d.Severity == validation.Error {
			t.Fatalf("unexpected error: %s: %s", d.Code, d.Message)
		}
	}
}

func TestApplicationSupportsMultipleDatabases(t *testing.T) {
	w := referenceWorld()
	if got := len(w.Applications[0].Databases); got != 2 {
		t.Fatalf("expected 2 database requirements, got %d", got)
	}
	if _, ok := w.Applications[0].DatabaseByAlias(local("sales")); !ok {
		t.Fatal("sales datasource not found")
	}
	if _, ok := w.Applications[0].DatabaseByAlias(local("master")); !ok {
		t.Fatal("master datasource not found")
	}
}

func TestDialectMismatchIsError(t *testing.T) {
	w := referenceWorld()
	w.Queries[0].Dialect = environment.ProviderOracle
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "SQL_DIALECT_TARGET_MISMATCH" && d.Severity == validation.Error {
			return
		}
	}
	t.Fatalf("expected SQL_DIALECT_TARGET_MISMATCH, got %#v", ds)
}

func TestUnknownDatasourceIsError(t *testing.T) {
	w := referenceWorld()
	w.Queries[0].Datasource = local("unknown")
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "UNKNOWN_DATASOURCE_ALIAS" {
			return
		}
	}
	t.Fatalf("expected UNKNOWN_DATASOURCE_ALIAS, got %#v", ds)
}

func TestCustomGoBindingNaming(t *testing.T) {
	b := realization.CustomGoBinding{Capability: cid("order_entry.compose_order_summary")}
	if got, want := b.ExpectedAdapterFilename(), "order_entry__compose_order_summary.bind.go"; got != want {
		t.Fatalf("filename: got %q want %q", got, want)
	}
	if got, want := b.ExpectedHookName(), "TeiBindOrderEntryComposeOrderSummary"; got != want {
		t.Fatalf("hook: got %q want %q", got, want)
	}
}

func TestContractMustBelongToTargetDatabase(t *testing.T) {
	w := referenceWorld()
	w.Applications[0].Databases[0].RequiredContracts[0].ContractID = cid("master_data.customer")
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "DATABASE_CONTRACT_TARGET_MISMATCH" {
			return
		}
	}
	t.Fatalf("expected DATABASE_CONTRACT_TARGET_MISMATCH, got %#v", ds)
}

func TestEnvironmentBindingAliasMustExist(t *testing.T) {
	w := referenceWorld()
	w.Environments[0].Bindings[0].Alias = local("missing")
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "UNKNOWN_BINDING_DATASOURCE_ALIAS" {
			return
		}
	}
	t.Fatalf("expected UNKNOWN_BINDING_DATASOURCE_ALIAS, got %#v", ds)
}

func TestDatabaseContractSemanticReferenceMustExist(t *testing.T) {
	w := referenceWorld()
	w.Databases[0].Contracts[0].Reads[0] = cid("sales.missing")
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "UNKNOWN_DATABASE_CONTRACT_SEMANTIC_REF" {
			return
		}
	}
	t.Fatalf("expected UNKNOWN_DATABASE_CONTRACT_SEMANTIC_REF, got %#v", ds)
}

func TestSemanticRelationTargetMustExist(t *testing.T) {
	w := referenceWorld()
	for i := range w.Semantics {
		if w.Semantics[i].ID == cid("sales") {
			w.Semantics[i].Data[0].Relations = []semantic.Relation{{ID: cid("sales.order.missing"), Symbol: "missing", Target: cid("unknown.data"), Cardinality: semantic.CardinalityOne}}
			break
		}
	}
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "UNKNOWN_SEMANTIC_RELATION_TARGET" {
			return
		}
	}
	t.Fatalf("expected UNKNOWN_SEMANTIC_RELATION_TARGET, got %#v", ds)
}

func TestUnknownSemanticAttributeTypeIsError(t *testing.T) {
	w := referenceWorld()
	for i := range w.Semantics {
		if w.Semantics[i].ID == cid("sales") {
			w.Semantics[i].Data[0].Attributes[0].TypeRef = cid("sales.missing_type")
		}
	}
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE" {
			return
		}
	}
	t.Fatalf("expected UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE, got %#v", ds)
}

func TestUnknownEnvironmentFeatureIsError(t *testing.T) {
	w := referenceWorld()
	w.Environments[0].Features = append(w.Environments[0].Features, environment.Feature("editor.unknown"))
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "UNKNOWN_ENVIRONMENT_FEATURE" {
			return
		}
	}
	t.Fatalf("expected UNKNOWN_ENVIRONMENT_FEATURE, got %#v", ds)
}

func TestSemanticTypeAliasCycleIsError(t *testing.T) {
	w := referenceWorld()
	for i := range w.Semantics {
		if w.Semantics[i].ID == cid("sales") {
			w.Semantics[i].Types = append(w.Semantics[i].Types,
				semantic.Type{ID: cid("sales.a"), Symbol: "A", Kind: semantic.TypeAlias, Underlying: cid("sales.b")},
				semantic.Type{ID: cid("sales.b"), Symbol: "B", Kind: semantic.TypeAlias, Underlying: cid("sales.a")},
			)
		}
	}
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "SEMANTIC_TYPE_ALIAS_CYCLE" {
			return
		}
	}
	t.Fatalf("expected SEMANTIC_TYPE_ALIAS_CYCLE, got %#v", ds)
}

func TestMigrationCannotBeOwnedByExternalObservedDatabase(t *testing.T) {
	w := referenceWorld()
	w.Migrations = append(w.Migrations, migration.Definition{ID: "20260827_001", Database: cid("master_data"), Recovery: migration.RecoveryForwardFix})
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "MIGRATION_DATABASE_NOT_MANAGED_HERE" {
			return
		}
	}
	t.Fatalf("expected MIGRATION_DATABASE_NOT_MANAGED_HERE, got %#v", ds)
}

func TestPersistenceAttributeMustBelongToSemanticData(t *testing.T) {
	w := referenceWorld()
	for i := range w.Semantics {
		if w.Semantics[i].ID == cid("customer") {
			w.Semantics[i].Data[0].Attributes = append(w.Semantics[i].Data[0].Attributes, semantic.Attribute{ID: cid("customer.customer.customer_id"), Symbol: "customerId", TypeRef: cid("std.identifier")})
		}
	}
	w.Persistence = append(w.Persistence, persistence.Mapping{ID: cid("sales_core.order"), Database: cid("sales_core"), SemanticData: cid("sales.order"), Table: "public.orders", Columns: []persistence.ColumnMapping{{Attribute: cid("customer.customer.customer_id"), Column: "customer_id", Storage: persistence.StorageInt64, Nullability: persistence.Required}}})
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "PERSISTENCE_ATTRIBUTE_PARENT_MISMATCH" {
			return
		}
	}
	t.Fatalf("expected PERSISTENCE_ATTRIBUTE_PARENT_MISMATCH, got %#v", ds)
}

func TestCapabilityPermissionMustBeExplicitlyGranted(t *testing.T) {
	w := referenceWorld()
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"}
	w.Plugins[0].Provides = append(w.Plugins[0].Provides, plugin.Capability{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.secure"), Actions: []deployment.ActionKind{deployment.ActionApplicationStart}, Permissions: []permission.Spec{perm}})
	w.Environments[0].DeploymentExecutors = []environment.DeploymentExecutorBinding{{Action: deployment.ActionApplicationStart, Scope: environment.ExecutorScopeApplication, Target: cid("order_entry"), Executor: cid("runtime.secure")}}
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "CAPABILITY_PERMISSION_NOT_GRANTED" {
			return
		}
	}
	t.Fatalf("expected CAPABILITY_PERMISSION_NOT_GRANTED, got %#v", ds)
}

func TestEnvironmentPermissionOvergrantIsError(t *testing.T) {
	w := referenceWorld()
	w.Environments[0].PermissionGrants = []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("runtime.test"), Permission: permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "database-endpoint"}}}
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "ENVIRONMENT_PERMISSION_OVERGRANT" {
			return
		}
	}
	t.Fatalf("expected ENVIRONMENT_PERMISSION_OVERGRANT, got %#v", ds)
}

func TestPermissionGrantRequiresResourceBinding(t *testing.T) {
	w := referenceWorld()
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"}
	w.Plugins[0].Provides = append(w.Plugins[0].Provides, plugin.Capability{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.secure_resource"), Actions: []deployment.ActionKind{deployment.ActionApplicationStart}, Permissions: []permission.Spec{perm}})
	w.Environments[0].DeploymentExecutors = []environment.DeploymentExecutorBinding{{Action: deployment.ActionApplicationStart, Scope: environment.ExecutorScopeApplication, Target: cid("order_entry"), Executor: cid("runtime.secure_resource")}}
	w.Environments[0].PermissionGrants = []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("runtime.secure_resource"), Permission: perm}}
	w.Environments[0].ResourceBindings = nil
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "PERMISSION_RESOURCE_BINDING_MISSING" {
			return
		}
	}
	t.Fatalf("expected PERMISSION_RESOURCE_BINDING_MISSING, got %#v", ds)
}

func TestDuplicateEnvironmentResourceBindingIsError(t *testing.T) {
	w := referenceWorld()
	env := &w.Environments[0]
	b := environment.ResourceBinding{Kind: permission.KindSecret, Scope: "named-secret", ResourceID: cid("test.named_secret")}
	env.ResourceBindings = []environment.ResourceBinding{b, b}
	ds := validation.Validate(w)
	for _, d := range ds {
		if d.Code == "DUPLICATE_ENVIRONMENT_RESOURCE_BINDING" {
			return
		}
	}
	t.Fatalf("expected DUPLICATE_ENVIRONMENT_RESOURCE_BINDING, got %#v", ds)
}
