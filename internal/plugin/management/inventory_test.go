package management

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
)

func cid(t *testing.T, value string) core.CanonicalID {
	t.Helper()
	return core.MustCanonicalID(value)
}

func builtinPlugin(t *testing.T, id string, provides ...plugin.Capability) plugin.Definition {
	t.Helper()
	return plugin.Definition{
		ID:       cid(t, id),
		Version:  "1.0.0",
		Runtime:  plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: provides,
	}
}

func databaseProviderCapability(t *testing.T, id string, provider environment.DatabaseProvider) plugin.Capability {
	t.Helper()
	return plugin.Capability{
		Kind:              plugin.CapabilityDatabaseProvider,
		ID:                cid(t, id),
		DatabaseProviders: []environment.DatabaseProvider{provider},
	}
}

func baseEnvironment(t *testing.T, plugins ...string) environment.Environment {
	t.Helper()
	env := environment.Environment{ID: cid(t, "development")}
	for _, id := range plugins {
		env.Plugins = append(env.Plugins, environment.ExtensionRequirement{ID: cid(t, id), CompatibleMajor: 1})
	}
	return env
}

func world(plugins []plugin.Definition, env environment.Environment) World {
	return World{Plugins: plugins, Environments: []environment.Environment{env}}
}

// TestDeriveKeepsEveryFactApart is the central negative: installation, the
// enabled setting, resolution, runtime and invocation must be separate fields,
// and a row must never claim "enabled".
func TestDeriveKeepsEveryFactApart(t *testing.T) {
	env := baseEnvironment(t, "tei.database.postgres")
	plugins := []plugin.Definition{
		builtinPlugin(t, "tei.database.postgres", databaseProviderCapability(t, "database.postgres", environment.ProviderPostgres)),
		builtinPlugin(t, "tei.unselected"),
	}
	report, err := Derive(Input{World: world(plugins, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}

	var selected, unselected Row
	for _, row := range report.Rows {
		switch row.PluginID {
		case "tei.database.postgres":
			selected = row
		case "tei.unselected":
			unselected = row
		}
	}
	if !selected.Declared || !selected.EnvSelected || !selected.Resolved {
		t.Fatalf("selected row must be declared, env-selected and resolved: %+v", selected)
	}
	if !unselected.Declared {
		t.Fatalf("unselected declared plugin must still be declared: %+v", unselected)
	}
	if unselected.EnvSelected || unselected.Resolved {
		t.Fatalf("unselected plugin must not be env-selected or resolved: %+v", unselected)
	}
	if !contains(unselected.ReasonCodes, ReasonEnvNotSelected) {
		t.Fatalf("unselected plugin must report %s: %+v", ReasonEnvNotSelected, unselected.ReasonCodes)
	}
	// No row may carry an "enabled" field, and every row names the gaps.
	for _, row := range report.Rows {
		if !contains(row.NotModelled, NotModelledEnable) || !contains(row.NotModelled, NotModelledInstall) {
			t.Fatalf("row %s must name the install and enable gaps: %+v", row.PluginID, row.NotModelled)
		}
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), `"enabled"`) {
		t.Fatalf("the inventory must not carry an enabled field:\n%s", raw)
	}
	if report.SettingsVersion != nil {
		t.Fatalf("settingsVersion must be nil when TEI has no source for it: %v", *report.SettingsVersion)
	}
}

// TestPartialScanNeverReportsZeroOrConfirmed pins PM1-09.
func TestPartialScanNeverReportsZeroOrConfirmed(t *testing.T) {
	env := baseEnvironment(t, "tei.database.postgres")
	plugins := []plugin.Definition{builtinPlugin(t, "tei.database.postgres", databaseProviderCapability(t, "database.postgres", environment.ProviderPostgres))}
	report, err := Derive(Input{World: world(plugins, env), Environment: env.ID, ScanState: ScanPartial})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if report.Confirmed {
		t.Fatalf("a partial scan must not be confirmed")
	}
	if report.Totals.PluginsTotal != nil {
		t.Fatalf("a partial scan must leave the total unknown, got %d", *report.Totals.PluginsTotal)
	}
	if !contains(report.ScanReasons, ReasonScanIncomplete) {
		t.Fatalf("partial scan must name %s: %+v", ReasonScanIncomplete, report.ScanReasons)
	}
	for _, row := range report.Rows {
		if !contains(row.Remaining, "rescan-required") {
			t.Fatalf("row %s must require a rescan: %+v", row.PluginID, row.Remaining)
		}
	}
}

func TestNotRunScanIsNotZero(t *testing.T) {
	env := baseEnvironment(t)
	report, err := Derive(Input{World: world(nil, env), Environment: env.ID, ScanState: ScanNotRun})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if report.Confirmed {
		t.Fatalf("a not-run scan must not be confirmed")
	}
	if report.Totals.PluginsTotal != nil {
		t.Fatalf("a not-run scan must leave the total unknown")
	}
	if !contains(report.ScanReasons, ReasonScanNotRun) {
		t.Fatalf("not-run scan must name %s", ReasonScanNotRun)
	}
}

// TestDeclaredButNotSelectedIsNotNotInstalled pins PM1-04 and PM1-05: a
// declared plugin the environment does not select is not "not installed", and
// it is not "enabled".
func TestDeclaredButNotSelectedIsNotNotInstalled(t *testing.T) {
	env := baseEnvironment(t)
	plugins := []plugin.Definition{builtinPlugin(t, "tei.declared.only")}
	report, err := Derive(Input{World: world(plugins, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("want one row, got %+v", report.Rows)
	}
	row := report.Rows[0]
	if !row.Declared {
		t.Fatalf("row must be declared")
	}
	if contains(row.ReasonCodes, ReasonPluginNotDeclared) {
		t.Fatalf("a declared plugin must not report %s", ReasonPluginNotDeclared)
	}
	if !contains(row.ReasonCodes, ReasonEnvNotSelected) {
		t.Fatalf("want %s, got %+v", ReasonEnvNotSelected, row.ReasonCodes)
	}
}

// TestMissingDeclarationIsReported pins PM1-04: the environment selects a
// plugin the Workspace does not declare. It is selected but not declared, which
// is a different fact from "not installed".
func TestMissingDeclarationIsReported(t *testing.T) {
	env := baseEnvironment(t, "tei.absent")
	report, err := Derive(Input{World: world(nil, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("the selected but undeclared plugin still needs a row: %+v", report.Rows)
	}
	row := report.Rows[0]
	if row.Declared {
		t.Fatalf("the plugin is not declared")
	}
	if !row.EnvSelected {
		t.Fatalf("the environment does select the plugin: %+v", row)
	}
	if row.Resolved {
		t.Fatalf("an undeclared plugin cannot resolve: %+v", row)
	}
	if !contains(row.ReasonCodes, ReasonPluginNotDeclared) {
		t.Fatalf("want %s, got %+v", ReasonPluginNotDeclared, row.ReasonCodes)
	}
}

// TestDependencyChainIsVisiblePerRow pins PM1-01 for the read-only part: the
// inventory shows each declared dependency; the candidate set belongs to
// #4377.
func TestDependencyChainIsVisiblePerRow(t *testing.T) {
	env := baseEnvironment(t, "a")
	a := plugin.Definition{
		ID: cid(t, "a"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{{Plugin: cid(t, "b"), CompatibleMajor: 1}},
	}
	b := plugin.Definition{
		ID: cid(t, "b"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{{Plugin: cid(t, "c"), CompatibleMajor: 1}},
	}
	c := builtinPlugin(t, "c")
	report, err := Derive(Input{World: world([]plugin.Definition{a, b, c}, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(report.Rows) != 3 {
		t.Fatalf("the resolver selects the whole chain, got %+v", report.Rows)
	}
	for _, row := range report.Rows {
		switch row.PluginID {
		case "a":
			if len(row.Requires) != 1 || row.Requires[0].Plugin != "b" {
				t.Fatalf("a must declare a dependency on b: %+v", row.Requires)
			}
		case "b":
			if len(row.Requires) != 1 || row.Requires[0].Plugin != "c" {
				t.Fatalf("b must declare a dependency on c: %+v", row.Requires)
			}
		case "c":
			if len(row.Requires) != 0 {
				t.Fatalf("c must declare no dependency: %+v", row.Requires)
			}
		}
	}
}

// TestUnsupportedVersionMapsToVersionUnsupported pins PM1-03.
func TestUnsupportedVersionMapsToVersionUnsupported(t *testing.T) {
	env := environment.Environment{ID: cid(t, "development")}
	env.Plugins = []environment.ExtensionRequirement{{ID: cid(t, "tei.database.postgres"), CompatibleMajor: 2}}
	plugins := []plugin.Definition{builtinPlugin(t, "tei.database.postgres", databaseProviderCapability(t, "database.postgres", environment.ProviderPostgres))}
	report, err := Derive(Input{World: world(plugins, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive must report a refusal, not fail: %v", err)
	}
	if report.Resolver.Resolved {
		t.Fatalf("a major mismatch must not resolve")
	}
	if !contains(report.Resolver.ReasonCodes, ReasonVersionUnsupported) {
		t.Fatalf("want %s, got %+v", ReasonVersionUnsupported, report.Resolver.ReasonCodes)
	}
	var found bool
	for _, reason := range report.Resolver.Reasons {
		if reason.ResolverCode == plugin.DiagnosticNoCompatiblePlugin {
			found = true
		}
	}
	if !found {
		t.Fatalf("the resolver's own code must be reported too: %+v", report.Resolver.Reasons)
	}
}

// TestDependencyCycleIsDistinct pins PM1-06.
func TestDependencyCycleIsDistinct(t *testing.T) {
	env := baseEnvironment(t, "a")
	a := plugin.Definition{
		ID: cid(t, "a"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{{Plugin: cid(t, "b"), CompatibleMajor: 1}},
	}
	b := plugin.Definition{
		ID: cid(t, "b"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{{Plugin: cid(t, "a"), CompatibleMajor: 1}},
	}
	report, err := Derive(Input{World: world([]plugin.Definition{a, b}, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if report.Resolver.Resolved {
		t.Fatalf("a cycle must not resolve")
	}
	if !contains(report.Resolver.ReasonCodes, ReasonDependencyCycle) {
		t.Fatalf("want %s, got %+v", ReasonDependencyCycle, report.Resolver.ReasonCodes)
	}
	if report.Confirmed {
		t.Fatalf("a cycle must not be confirmed")
	}
}

// TestLockPinsRuntimeAndArtifact pins the lock projection: without a lock the
// runtime boundary is unverified; with a lock the boundary and artifact come
// from the existing plan.
func TestLockPinsRuntimeAndArtifact(t *testing.T) {
	env := baseEnvironment(t, "tei.database.postgres")
	plugins := []plugin.Definition{builtinPlugin(t, "tei.database.postgres", databaseProviderCapability(t, "database.postgres", environment.ProviderPostgres))}

	report, err := Derive(Input{World: world(plugins, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	for _, row := range report.Rows {
		if !contains(row.ReasonCodes, ReasonRuntimeUnverified) {
			t.Fatalf("without a lock the runtime boundary is unverified: %+v", row.ReasonCodes)
		}
		if row.LockPinned != nil {
			t.Fatalf("without a lock LockPinned must be nil, not false")
		}
	}

	lock, err := plugin.BuildLock(plugin.Catalog{Plugins: plugins}, []environment.Environment{env})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}
	report, err = Derive(Input{World: world(plugins, env), Environment: env.ID, Lock: &lock})
	if err != nil {
		t.Fatalf("Derive with lock: %v", err)
	}
	if !report.Resolver.Resolved {
		t.Fatalf("the lock must resolve: %+v", report.Resolver)
	}
	for _, row := range report.Rows {
		if row.LockPinned == nil || !*row.LockPinned {
			t.Fatalf("the lock pins this plugin: %+v", row)
		}
		if row.RuntimeBoundary != string(runtimeBoundary(plugin.RuntimeBuiltin)) {
			// The boundary is reported by BuildPlan; in-process is the builtin
			// boundary. A different value would be a second definition.
			t.Fatalf("builtin boundary must come from BuildPlan, got %q", row.RuntimeBoundary)
		}
	}
}

func runtimeBoundary(kind plugin.RuntimeKind) string { return "in-process" }

// TestLockMismatchIsAnInputProblem not a resolution outcome.
func TestLockMismatchIsAnInputProblem(t *testing.T) {
	env := baseEnvironment(t, "tei.database.postgres")
	plugins := []plugin.Definition{builtinPlugin(t, "tei.database.postgres", databaseProviderCapability(t, "database.postgres", environment.ProviderPostgres))}
	lock, err := plugin.BuildLock(plugin.Catalog{Plugins: plugins}, []environment.Environment{env})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}
	lock.Environments[0].Plugins[0].Fingerprint = "deadbeef"
	report, err := Derive(Input{World: world(plugins, env), Environment: env.ID, Lock: &lock})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if report.Resolver.Resolved {
		t.Fatalf("a diverged lock must not resolve")
	}
	if !contains(report.Resolver.ReasonCodes, ReasonSourceFingerprintMismatch) {
		t.Fatalf("want %s, got %+v", ReasonSourceFingerprintMismatch, report.Resolver.ReasonCodes)
	}
}

// TestPermissionGapNeedsALock pins PM1-07: without a lock the inventory says
// grant-unverified; with a lock a missing grant is permission-missing.
func TestPermissionGapNeedsALock(t *testing.T) {
	env := baseEnvironment(t, "tei.database.postgres")
	capability := databaseProviderCapability(t, "database.postgres", environment.ProviderPostgres)
	capability.Permissions = []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "workspace"}}
	plugins := []plugin.Definition{builtinPlugin(t, "tei.database.postgres", capability)}

	report, err := Derive(Input{World: world(plugins, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	for _, row := range report.Rows {
		if !contains(row.ReasonCodes, ReasonGrantUnverified) {
			t.Fatalf("without a lock the grant state is unverified: %+v", row.ReasonCodes)
		}
		if contains(row.ReasonCodes, ReasonPermissionMissing) {
			t.Fatalf("without a lock a missing grant must not be claimed: %+v", row.ReasonCodes)
		}
	}

	lock, err := plugin.BuildLock(plugin.Catalog{Plugins: plugins}, []environment.Environment{env})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}
	report, err = Derive(Input{World: world(plugins, env), Environment: env.ID, Lock: &lock})
	if err != nil {
		t.Fatalf("Derive with lock: %v", err)
	}
	// The resolver still resolves the lock; the missing grant is a capability
	// gap the plan carries, and the invocation is what would refuse it.
	if !report.Resolver.Resolved {
		t.Fatalf("the lock itself still resolves: %+v", report.Resolver)
	}
	if report.Resolver.PlanBuilt == nil {
		t.Fatalf("a lock was supplied, so the plan state must be reported: %+v", report.Resolver)
	}
	var found bool
	for _, row := range report.Rows {
		if contains(row.ReasonCodes, ReasonPermissionMissing) {
			found = true
		}
	}
	if !found {
		t.Fatalf("the row must report %s: %+v", ReasonPermissionMissing, report.Rows)
	}
}

// TestUnsupportedTargetIsReported pins PM1-08: an unknown runtime kind is
// reported instead of falling back to another implementation.
func TestUnsupportedTargetIsReported(t *testing.T) {
	env := baseEnvironment(t, "tei.wasm")
	def := plugin.Definition{
		ID: cid(t, "tei.wasm"), Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeKind("wasm")},
	}
	report, err := Derive(Input{World: world([]plugin.Definition{def}, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("want one row: %+v", report.Rows)
	}
	row := report.Rows[0]
	if !contains(row.ReasonCodes, ReasonUnsupportedTarget) {
		t.Fatalf("want %s, got %+v", ReasonUnsupportedTarget, row.ReasonCodes)
	}
}

// TestMultipleProvidersAndDatabaseLevels pins the data-level projection: two
// providers for one database instance are visible, and one provider is not
// counted twice.
func TestMultipleProvidersAndDatabaseLevels(t *testing.T) {
	env := baseEnvironment(t, "tei.database.postgres", "tei.database.postgres.alt")
	env.Instances = []environment.DatabaseInstance{{
		ID: cid(t, "sales_core.dev"), Definition: cid(t, "sales_core"), Provider: environment.ProviderPostgres,
	}}
	inside := builtinPlugin(t, "tei.database.postgres", databaseProviderCapability(t, "database.postgres", environment.ProviderPostgres))
	alt := builtinPlugin(t, "tei.database.postgres.alt", databaseProviderCapability(t, "database.postgres.alt", environment.ProviderPostgres))
	report, err := Derive(Input{World: world([]plugin.Definition{inside, alt}, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(report.Levels) != 1 {
		t.Fatalf("want one database level: %+v", report.Levels)
	}
	level := report.Levels[0]
	if level.Status != LevelMultipleProviders {
		t.Fatalf("want %s, got %+v", LevelMultipleProviders, level)
	}
	if len(level.Providers) != 2 {
		t.Fatalf("both providers must be listed: %+v", level.Providers)
	}
}

func TestUnknownEnvironmentIsAnError(t *testing.T) {
	env := baseEnvironment(t)
	_, err := Derive(Input{World: world(nil, env), Environment: cid(t, "production")})
	if err == nil {
		t.Fatalf("an unknown environment must be refused")
	}
}

func TestUnknownScanStateIsRejected(t *testing.T) {
	env := baseEnvironment(t)
	_, err := Derive(Input{World: world(nil, env), Environment: env.ID, ScanState: "half"})
	if err == nil {
		t.Fatalf("an unknown scan state must be refused")
	}
}

// TestDiagnosisIsDeterministicAndTextIndependent pins part of PM1-11 and the
// display contract: the same input yields the same codes, and every row has a
// message.
func TestDiagnosisIsDeterministicAndTextIndependent(t *testing.T) {
	env := baseEnvironment(t, "tei.database.postgres", "tei.absent")
	plugins := []plugin.Definition{builtinPlugin(t, "tei.database.postgres", databaseProviderCapability(t, "database.postgres", environment.ProviderPostgres))}
	first, err := Derive(Input{World: world(plugins, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	second, err := Derive(Input{World: world(plugins, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive again: %v", err)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatalf("the same input must yield the same report:\n%s\n%s", a, b)
	}
	for _, row := range first.Rows {
		for _, code := range row.ReasonCodes {
			if _, ok := first.Messages[code]; !ok {
				t.Fatalf("reason code %q has no message", code)
			}
		}
	}
	if first.Messages[ReasonPluginNotDeclared] == "" {
		t.Fatalf("the message catalogue must cover a code the report emits")
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
