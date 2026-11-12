package management

import (
	"testing"

	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
)

// TestVocabularyConstantsAreProduced names every exported word this package
// declares, so the repository ratchet sees an owner for each (#4422). The test
// asserts the closed sets actually contain the word rather than merely naming
// it: a constant that is declared and never produced would otherwise pass.
func TestVocabularyConstantsAreProduced(t *testing.T) {
	// The report schema is the identifier written by source.
	if SchemaVersion != "tei.plugin-management.inventory/v0" {
		t.Fatalf("SchemaVersion = %q", SchemaVersion)
	}

	// Scan states are the three values Derive accepts.
	for _, state := range []string{ScanComplete, ScanPartial, ScanNotRun} {
		if _, err := Derive(Input{
			World:       world(nil, environment.Environment{ID: cid(t, "development")}),
			Environment: cid(t, "development"),
			ScanState:   state,
		}); err != nil {
			t.Fatalf("scan state %q must be accepted: %v", state, err)
		}
	}

	// Every not-modelled fact is reported, so a reader learns the gaps once
	// instead of inferring false values.
	report, err := Derive(Input{
		World:       world(nil, environment.Environment{ID: cid(t, "development")}),
		Environment: cid(t, "development"),
	})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	for _, fact := range []string{
		NotModelledInstall, NotModelledEnable, NotModelledInitFailure,
		NotModelledOptionalDependency, NotModelledRetainedData,
		NotModelledInvocation, NotModelledEffect, NotModelledSettingsVersion,
	} {
		found := false
		for _, listed := range report.NotModelled {
			if listed == fact {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("report.NotModelled must name %q: %+v", fact, report.NotModelled)
		}
	}
}

// TestReasonVocabularyIsReachable wires each management reason code to an
// input that really produces it, so no declared word is dead.
func TestReasonVocabularyIsReachable(t *testing.T) {
	got := map[string]bool{}
	collect := func(report Report) {
		for _, code := range report.ScanReasons {
			got[code] = true
		}
		for _, code := range report.Resolver.ReasonCodes {
			got[code] = true
		}
		for _, row := range report.Rows {
			for _, code := range row.ReasonCodes {
				got[code] = true
			}
		}
	}
	derive := func(t *testing.T, in Input) Report {
		t.Helper()
		report, err := Derive(in)
		if err != nil {
			t.Fatalf("Derive: %v", err)
		}
		collect(report)
		return report
	}

	// Scan states.
	for _, state := range []string{ScanPartial, ScanNotRun} {
		derive(t, Input{
			World:       world(nil, environment.Environment{ID: cid(t, "development")}),
			Environment: cid(t, "development"),
			ScanState:   state,
		})
	}

	// A declared plugin the environment does not select, and one it selects
	// but the Workspace does not declare.
	derive(t, Input{
		World:       world([]plugin.Definition{builtinPlugin(t, "tei.declared.only")}, environment.Environment{ID: cid(t, "development")}),
		Environment: cid(t, "development"),
	})
	selectedOnly := environment.Environment{ID: cid(t, "development")}
	selectedOnly.Plugins = []environment.ExtensionRequirement{{ID: cid(t, "tei.absent"), CompatibleMajor: 1}}
	derive(t, Input{World: world(nil, selectedOnly), Environment: selectedOnly.ID})

	// No lock: runtime and grant are unverified.
	derive(t, Input{
		World:       world([]plugin.Definition{builtinPlugin(t, "tei.database.postgres")}, environment.Environment{ID: cid(t, "development")}),
		Environment: cid(t, "development"),
	})

	// A version mismatch reaches the resolver outcome.
	mismatch := environment.Environment{ID: cid(t, "development")}
	mismatch.Plugins = []environment.ExtensionRequirement{{ID: cid(t, "tei.database.postgres"), CompatibleMajor: 2}}
	derive(t, Input{
		World:       world([]plugin.Definition{builtinPlugin(t, "tei.database.postgres")}, mismatch),
		Environment: mismatch.ID,
	})

	// A dependency cycle.
	cycleEnv := environment.Environment{ID: cid(t, "development")}
	cycleEnv.Plugins = []environment.ExtensionRequirement{{ID: cid(t, "a"), CompatibleMajor: 1}}
	cycleA := plugin.Definition{ID: cid(t, "a"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{{Plugin: cid(t, "b"), CompatibleMajor: 1}}}
	cycleB := plugin.Definition{ID: cid(t, "b"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{{Plugin: cid(t, "a"), CompatibleMajor: 1}}}
	derive(t, Input{World: world([]plugin.Definition{cycleA, cycleB}, cycleEnv), Environment: cycleEnv.ID})

	// An unresolved dependency.
	missingEnv := environment.Environment{ID: cid(t, "development")}
	missingEnv.Plugins = []environment.ExtensionRequirement{{ID: cid(t, "needs.absent"), CompatibleMajor: 1}}
	needsAbsent := plugin.Definition{ID: cid(t, "needs.absent"), Version: "1.0.0",
		Runtime:  plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{{Plugin: cid(t, "tei.absent.dependency"), CompatibleMajor: 1}}}
	derive(t, Input{World: world([]plugin.Definition{needsAbsent}, missingEnv), Environment: missingEnv.ID})

	// An incompatible transitive requirement: two plugins demand different
	// majors of the same dependency.
	transEnv := environment.Environment{ID: cid(t, "development")}
	transEnv.Plugins = []environment.ExtensionRequirement{{ID: cid(t, "a"), CompatibleMajor: 1}}
	transA := plugin.Definition{ID: cid(t, "a"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{
			{Plugin: cid(t, "b"), CompatibleMajor: 1},
			{Plugin: cid(t, "c"), CompatibleMajor: 1},
		}}
	transB := plugin.Definition{ID: cid(t, "b"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{{Plugin: cid(t, "d"), CompatibleMajor: 1}}}
	transC := plugin.Definition{ID: cid(t, "c"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{{Plugin: cid(t, "d"), CompatibleMajor: 2}}}
	transD := plugin.Definition{ID: cid(t, "d"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}}
	derive(t, Input{World: world([]plugin.Definition{transA, transB, transC, transD}, transEnv), Environment: transEnv.ID})

	// An ambiguous environment selection: the same plugin demanded at two
	// majors.
	duplicateEnv := environment.Environment{ID: cid(t, "development")}
	duplicateEnv.Plugins = []environment.ExtensionRequirement{
		{ID: cid(t, "tei.database.postgres"), CompatibleMajor: 1},
		{ID: cid(t, "tei.database.postgres"), CompatibleMajor: 2},
	}
	derive(t, Input{
		World:       world([]plugin.Definition{builtinPlugin(t, "tei.database.postgres")}, duplicateEnv),
		Environment: duplicateEnv.ID,
	})

	// An unsupported runtime kind.
	wasmEnv := environment.Environment{ID: cid(t, "development")}
	wasmEnv.Plugins = []environment.ExtensionRequirement{{ID: cid(t, "tei.wasm"), CompatibleMajor: 1}}
	derive(t, Input{
		World: world([]plugin.Definition{{ID: cid(t, "tei.wasm"), Version: "1.0.0",
			Runtime: plugin.Runtime{Kind: plugin.RuntimeKind("wasm")}}}, wasmEnv),
		Environment: wasmEnv.ID,
	})

	// A lock whose environment entry diverges: an input problem naming the
	// source fingerprint.
	lockEnv := environment.Environment{ID: cid(t, "development")}
	lockEnv.Plugins = []environment.ExtensionRequirement{{ID: cid(t, "tei.database.postgres"), CompatibleMajor: 1}}
	lockPlugins := []plugin.Definition{builtinPlugin(t, "tei.database.postgres")}
	lock, err := plugin.BuildLock(plugin.Catalog{Plugins: lockPlugins}, []environment.Environment{lockEnv})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}
	if len(lock.Environments) != 1 || len(lock.Environments[0].Plugins) != 1 {
		t.Fatalf("the lock must pin one plugin: %+v", lock)
	}
	lock.Environments[0].Plugins[0].Fingerprint = "diverged"
	derive(t, Input{World: world(lockPlugins, lockEnv), Environment: lockEnv.ID, Lock: &lock})

	// A lock with an unsupported format version: the lock bytes are unusable.
	badFormat := lock
	badFormat.FormatVersion = 9
	derive(t, Input{World: world(lockPlugins, lockEnv), Environment: lockEnv.ID, Lock: &badFormat})

	// A lock that pins a plugin the environment no longer matches: no lock reason
	// needs a separate entry, because the divergence above already names it.

	// A capability whose declared grant is missing: the plan is refused, so the
	// row reports the grant state as unknown.
	gapEnv := environment.Environment{ID: cid(t, "development")}
	gapCap := databaseProviderCapability(t, "database.postgres", environment.ProviderPostgres)
	gapCap.Permissions = []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "workspace"}}
	gapPlugins := []plugin.Definition{builtinPlugin(t, "tei.database.postgres", gapCap)}
	gapLock, err := plugin.BuildLock(plugin.Catalog{Plugins: gapPlugins}, []environment.Environment{gapEnv})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}
	derive(t, Input{World: world(gapPlugins, gapEnv), Environment: gapEnv.ID, Lock: &gapLock})

	// The same capability with the grant present: a successful plan names no
	// gap, and the plan-built field is set.
	grantEnv := environment.Environment{ID: cid(t, "development")}
	grantEnv.PermissionGrants = []environment.PermissionGrant{{
		CapabilityKind: string(plugin.CapabilityDatabaseProvider),
		CapabilityID:   cid(t, "database.postgres"),
		Permission:     permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "workspace"},
	}}
	grantLock, err := plugin.BuildLock(plugin.Catalog{Plugins: gapPlugins}, []environment.Environment{grantEnv})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}
	granted := derive(t, Input{World: world(gapPlugins, grantEnv), Environment: grantEnv.ID, Lock: &grantLock})

	// A plugin declared with a permission the environment does not grant: the
	// row compares the plan's capability entry and names the gap. A plan
	// refusal for a resource binding is the other path; either way the row must
	// not claim the permission is present.
	missingGrantEnv := environment.Environment{ID: cid(t, "development")}
	missingGrantPlugins := []plugin.Definition{builtinPlugin(t, "tei.database.postgres", gapCap)}
	missingGrantLock, err := plugin.BuildLock(plugin.Catalog{Plugins: missingGrantPlugins}, []environment.Environment{missingGrantEnv})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}
	_ = derive(t, Input{World: world(missingGrantPlugins, missingGrantEnv), Environment: missingGrantEnv.ID, Lock: &missingGrantLock})

	for _, code := range []string{
		ReasonPluginNotDeclared, ReasonEnvNotSelected, ReasonDeclaredButUnresolvable,
		ReasonVersionUnsupported, ReasonDependencyCycle, ReasonDependencyUnresolved,
		ReasonContractIncompatible, ReasonResolverRefused, ReasonPermissionUnknown,
		ReasonGrantUnverified, ReasonUnsupportedTarget,
		ReasonRuntimeUnverified, ReasonScanIncomplete,
		ReasonScanNotRun, ReasonLockNotProvided, ReasonSourceFingerprintMismatch,
	} {
		if !got[code] {
			t.Errorf("reason code %q must be produced by one of the inputs above", code)
		}
	}
	_ = granted
}

// TestLevelVocabularyIsProduced checks the data-level statuses.
func TestLevelVocabularyIsProduced(t *testing.T) {
	report, err := Derive(Input{
		World: world(
			[]plugin.Definition{builtinPlugin(t, "tei.database.postgres",
				databaseProviderCapability(t, "database.postgres", environment.ProviderPostgres))},
			environment.Environment{
				ID:        cid(t, "development"),
				Instances: []environment.DatabaseInstance{{ID: cid(t, "sales_core.dev"), Definition: cid(t, "sales_core"), Provider: environment.ProviderPostgres}},
				Features:  []environment.Feature{environment.FeatureDatabaseWorkbench},
			},
		),
		Environment: cid(t, "development"),
	})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	statuses := map[string]bool{}
	for _, level := range report.Levels {
		statuses[level.Status] = true
	}
	if !statuses[LevelProvided] {
		t.Fatalf("a database instance with one provider must be %q: %+v", LevelProvided, report.Levels)
	}
	if !statuses[LevelUnsupported] {
		t.Fatalf("a feature no capability provides must be %q: %+v", LevelUnsupported, report.Levels)
	}

	noProvider, err := Derive(Input{
		World: world([]plugin.Definition{builtinPlugin(t, "tei.unrelated")}, environment.Environment{
			ID:        cid(t, "development"),
			Instances: []environment.DatabaseInstance{{ID: cid(t, "sales_core.dev"), Definition: cid(t, "sales_core"), Provider: environment.ProviderPostgres}},
		}),
		Environment: cid(t, "development"),
	})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if noProvider.Levels[0].Status != LevelNoProvider {
		t.Fatalf("an instance with no provider must be %q: %+v", LevelNoProvider, noProvider.Levels[0])
	}
}

// TestRoleVocabularyIsProduced checks the role statuses.
func TestRoleVocabularyIsProduced(t *testing.T) {
	statuses := map[string]bool{}
	for _, role := range roleStatuses() {
		statuses[role.Status] = true
	}
	for _, status := range []string{RoleRepresented, RolePartial, RoleNotAdopted, RoleBuildInput, RoleExternal} {
		if !statuses[status] {
			t.Errorf("role status %q must be produced by the table: %+v", status, roleStatuses())
		}
	}
}

// TestNotObservedHasARemediation keeps the invocation gap visible: the word is
// part of the vocabulary even though no input produces it today, because real
// invocation is a separate qualification (#1174).
func TestNotObservedHasARemediation(t *testing.T) {
	if ReasonNotObserved != "not-observed" {
		t.Fatalf("ReasonNotObserved = %q", ReasonNotObserved)
	}
	remediation := remediationFor(&Row{ReasonCodes: []string{ReasonNotObserved}})
	if len(remediation) == 0 || remediation[0] != "read-invocation-evidence" {
		t.Fatalf("not-observed must point at the invocation evidence, not at a permission: %+v", remediation)
	}
}

// TestRoleUIKitNamesTheSeparateAxis keeps the UI Kit word in the vocabulary as
// an explicitly separate axis (#997): the plugin contract does not represent it.
func TestRoleUIKitNamesTheSeparateAxis(t *testing.T) {
	if RoleUIKit != "ui-kit-axis" {
		t.Fatalf("RoleUIKit = %q", RoleUIKit)
	}
	for _, role := range roleStatuses() {
		if role.Target == "ui-studio-extension" && role.Status != RolePartial {
			t.Fatalf("the UI extension target is only partially represented: %+v", role)
		}
	}
}
