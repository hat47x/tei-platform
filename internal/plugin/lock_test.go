package plugin_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestLockPinsVersionEvenWhenNewerCompatibleAppears(t *testing.T) {
	base := plugin.Definition{ID: core.MustCanonicalID("tei.x"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}}
	env := environment.Environment{ID: core.MustCanonicalID("prod"), Plugins: []environment.ExtensionRequirement{{ID: base.ID, CompatibleMajor: 1}}}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{base}}
	lock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	catalog.Plugins = append(catalog.Plugins, plugin.Definition{ID: base.ID, Version: "1.9.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}})
	r, err := catalog.ResolveEnvironmentWithLock(env, lock)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Plugins) != 1 || r.Plugins[0].Version != "1.0.0" {
		t.Fatalf("locked resolution: %#v", r.Plugins)
	}
}

func TestLockDetectsDescriptorMutation(t *testing.T) {
	base := plugin.Definition{ID: core.MustCanonicalID("tei.x"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}}
	env := environment.Environment{ID: core.MustCanonicalID("prod"), Plugins: []environment.ExtensionRequirement{{ID: base.ID, CompatibleMajor: 1}}}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{base}}
	lock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	catalog.Plugins[0].Runtime = plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "changed"}
	if _, err := catalog.ResolveEnvironmentWithLock(env, lock); err == nil {
		t.Fatal("expected fingerprint divergence")
	}
}

func TestDefinitionFingerprintIgnoresCapabilityOrdering(t *testing.T) {
	a := plugin.Definition{ID: core.MustCanonicalID("tei.x"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: core.MustCanonicalID("runtime.x"), Actions: []deployment.ActionKind{deployment.ActionTrafficShift, deployment.ActionApplicationStart}}}}
	b := a
	b.Provides = []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: core.MustCanonicalID("runtime.x"), Actions: []deployment.ActionKind{deployment.ActionApplicationStart, deployment.ActionTrafficShift}}}
	if plugin.DefinitionFingerprint(a) != plugin.DefinitionFingerprint(b) {
		t.Fatal("semantic reordering changed fingerprint")
	}
}

func TestEnvironmentLockFingerprintIgnoresPluginOrdering(t *testing.T) {
	l1 := plugin.Lock{FormatVersion: plugin.LockFormatVersion, Environments: []plugin.EnvironmentLock{{Environment: "prod", Plugins: []plugin.LockedPlugin{{ID: "tei.a", Version: "1.0.0", Fingerprint: "a"}, {ID: "tei.b", Version: "1.0.0", Fingerprint: "b"}}}}}
	l2 := plugin.Lock{FormatVersion: plugin.LockFormatVersion, Environments: []plugin.EnvironmentLock{{Environment: "prod", Plugins: []plugin.LockedPlugin{{ID: "tei.b", Version: "1.0.0", Fingerprint: "b"}, {ID: "tei.a", Version: "1.0.0", Fingerprint: "a"}}}}}
	a, err := plugin.EnvironmentLockFingerprint(l1, "prod")
	if err != nil {
		t.Fatal(err)
	}
	b, err := plugin.EnvironmentLockFingerprint(l2, "prod")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("ordering changed environment lock fingerprint: %s != %s", a, b)
	}
}


func TestLegacyFormat2PluginLockRemainsReadable(t *testing.T) {
	def := plugin.Definition{
		ID:      core.MustCanonicalID("tei.legacy"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
	}
	env := environment.Environment{
		ID: core.MustCanonicalID("legacy"),
		Plugins: []environment.ExtensionRequirement{{
			ID:              def.ID,
			CompatibleMajor: 1,
		}},
	}
	legacy := plugin.Lock{
		FormatVersion: 2,
		Environments: []plugin.EnvironmentLock{{
			Environment: env.ID,
			Plugins: []plugin.LockedPlugin{{
				ID:          def.ID,
				Version:     def.Version,
				Fingerprint: plugin.DefinitionFingerprint(def),
			}},
		}},
	}
	if _, err := plugin.EnvironmentLockFingerprint(legacy, env.ID); err != nil {
		t.Fatal(err)
	}
	resolved, err := (plugin.Catalog{Plugins: []plugin.Definition{def}}).ResolveEnvironmentWithLock(env, legacy)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Plugins) != 1 || resolved.Plugins[0].ID != def.ID {
		t.Fatalf("resolved legacy lock: %#v", resolved.Plugins)
	}
}

func TestBuildTraceFormat3PluginLockIsReadable(t *testing.T) {
	def := plugin.Definition{
		ID:      core.MustCanonicalID("tei.traced"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "traced"},
		Artifact: plugin.Artifact{
			Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
	env := environment.Environment{
		ID: core.MustCanonicalID("traced"),
		Plugins: []environment.ExtensionRequirement{{
			ID:              def.ID,
			CompatibleMajor: 1,
		}},
	}
	lock := plugin.Lock{
		FormatVersion: plugin.BuildTraceLockFormatVersion,
		Environments: []plugin.EnvironmentLock{{
			Environment: env.ID,
			Plugins: []plugin.LockedPlugin{{
				ID:                   def.ID,
				Version:              def.Version,
				Fingerprint:          plugin.DefinitionFingerprint(def),
				ArtifactDigest:       def.Artifact.Digest,
				BuildPlanFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			}},
		}},
	}
	if _, err := plugin.EnvironmentLockFingerprint(lock, env.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := (plugin.Catalog{Plugins: []plugin.Definition{def}}).ResolveEnvironmentWithLock(env, lock); err != nil {
		t.Fatal(err)
	}
}

func TestUnsupportedPluginLockFormatFails(t *testing.T) {
	lock := plugin.Lock{FormatVersion: 99}
	if _, err := plugin.EnvironmentLockFingerprint(lock, core.MustCanonicalID("x")); err == nil {
		t.Fatal("expected unsupported lock format to fail")
	}
	env := environment.Environment{ID: core.MustCanonicalID("x")}
	if _, err := (plugin.Catalog{}).ResolveEnvironmentWithLock(env, lock); err == nil {
		t.Fatal("expected unsupported lock format to fail")
	}
}
