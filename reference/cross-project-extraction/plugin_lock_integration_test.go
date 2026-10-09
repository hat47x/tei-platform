package crossprojectextraction_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

const (
	artifactV1 = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	artifactV2 = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	artifactOther = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

var sharedPluginID = core.MustCanonicalID("tei.reference.required-fields")

func sharedDefinition(version, artifactDigest string) plugin.Definition {
	return plugin.Definition{
		ID:      sharedPluginID,
		Version: version,
		Runtime: plugin.Runtime{
			Kind:    plugin.RuntimeProcess,
			Command: "required-fields-reference",
		},
		Artifact: plugin.Artifact{Digest: artifactDigest},
	}
}

func projectEnvironment(id string) environment.Environment {
	return environment.Environment{
		ID: core.MustCanonicalID(id),
		Plugins: []environment.ExtensionRequirement{{
			ID:              sharedPluginID,
			CompatibleMajor: 1,
		}},
	}
}

func sha256Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func buildPlanFor(t *testing.T, definition plugin.Definition) plugin.BuildPlan {
	t.Helper()

	source, err := os.ReadFile("shared/required-fields.mjs")
	if err != nil {
		t.Fatal(err)
	}
	return plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: definition.ID, Version: definition.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(definition),
		Sources: []plugin.BuildSource{{
			Kind:    "implementation",
			ID:      "required-fields",
			Locator: "reference/cross-project-extraction/shared/required-fields.mjs",
			Digest:  sha256Digest(source),
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.reference"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  artifactOther,
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "reference-module",
			Entrypoint: "shared/required-fields.mjs",
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "host-process",
		Profile:    "cross-project-extraction-reference",
	}
}

func onlyLockedPlugin(t *testing.T, lock plugin.Lock) plugin.LockedPlugin {
	t.Helper()
	if len(lock.Environments) != 1 || len(lock.Environments[0].Plugins) != 1 {
		t.Fatalf("unexpected plugin lock: %#v", lock)
	}
	return lock.Environments[0].Plugins[0]
}

func TestExistingPluginLockKeepsProjectAdoptionIndependent(t *testing.T) {
	v1 := sharedDefinition("1.0.0", artifactV1)
	v2 := sharedDefinition("1.1.0", artifactV2)
	order := projectEnvironment("order")
	invoice := projectEnvironment("invoice")

	initialCatalog := plugin.Catalog{Plugins: []plugin.Definition{v1}}
	orderV1Lock, err := plugin.BuildLock(initialCatalog, []environment.Environment{order})
	if err != nil {
		t.Fatal(err)
	}
	invoiceV1Lock, err := plugin.BuildLock(initialCatalog, []environment.Environment{invoice})
	if err != nil {
		t.Fatal(err)
	}

	upgradedCatalog := plugin.Catalog{Plugins: []plugin.Definition{v1, v2}}

	orderResolved, err := upgradedCatalog.ResolveEnvironmentWithLock(order, orderV1Lock)
	if err != nil {
		t.Fatal(err)
	}
	if got := orderResolved.Plugins[0].Version; got != "1.0.0" {
		t.Fatalf("order changed without adoption: got %s", got)
	}

	invoiceV2Lock, err := plugin.BuildLock(upgradedCatalog, []environment.Environment{invoice})
	if err != nil {
		t.Fatal(err)
	}
	invoiceResolved, err := upgradedCatalog.ResolveEnvironmentWithLock(invoice, invoiceV2Lock)
	if err != nil {
		t.Fatal(err)
	}
	if got := invoiceResolved.Plugins[0].Version; got != "1.1.0" {
		t.Fatalf("invoice did not adopt selected version: got %s", got)
	}

	orderResolvedAgain, err := upgradedCatalog.ResolveEnvironmentWithLock(order, orderV1Lock)
	if err != nil {
		t.Fatal(err)
	}
	if got := orderResolvedAgain.Plugins[0].Version; got != "1.0.0" {
		t.Fatalf("invoice adoption propagated to order: got %s", got)
	}

	invoiceRolledBack, err := upgradedCatalog.ResolveEnvironmentWithLock(invoice, invoiceV1Lock)
	if err != nil {
		t.Fatal(err)
	}
	if got := invoiceRolledBack.Plugins[0].Version; got != "1.0.0" {
		t.Fatalf("invoice rollback did not restore v1: got %s", got)
	}
}

func TestExistingBuildAndRuntimeLocksFailClosedOnDrift(t *testing.T) {
	definition := sharedDefinition("1.1.0", artifactV2)
	invoice := projectEnvironment("invoice")
	catalog := plugin.Catalog{Plugins: []plugin.Definition{definition}}

	runtimeLock, err := plugin.BuildLock(catalog, []environment.Environment{invoice})
	if err != nil {
		t.Fatal(err)
	}
	plan := buildPlanFor(t, definition)
	build, err := plugin.LockBuildOutput(plan, artifactV2)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := plugin.BindRuntimeBuild(runtimeLock, invoice.ID, build)
	if err != nil {
		t.Fatal(err)
	}
	locked := onlyLockedPlugin(t, bound)
	if err := plugin.VerifyRuntimePluginBuild(plan, build, definition, locked); err != nil {
		t.Fatal(err)
	}

	t.Run("source changes after the locked build", func(t *testing.T) {
		changed := plan
		changed.Sources = append([]plugin.BuildSource(nil), plan.Sources...)
		changed.Sources[0].Digest = artifactOther
		if err := plugin.VerifyRuntimePluginBuild(changed, build, definition, locked); err == nil {
			t.Fatal("expected source drift to invalidate the build binding")
		}
	})

	t.Run("artifact changes under the same logical version", func(t *testing.T) {
		changed := definition
		changed.Artifact = plugin.Artifact{Digest: artifactOther}
		changedCatalog := plugin.Catalog{Plugins: []plugin.Definition{changed}}
		if _, err := changedCatalog.ResolveEnvironmentWithLock(invoice, bound); err == nil {
			t.Fatal("expected descriptor/artifact drift to invalidate the runtime lock")
		}
	})

	t.Run("runtime lock cannot bind a different build artifact", func(t *testing.T) {
		changedBuild := build
		changedBuild.ArtifactDigest = artifactOther
		if _, err := plugin.BindRuntimeBuild(runtimeLock, invoice.ID, changedBuild); err == nil {
			t.Fatal("expected artifact mismatch to fail closed")
		}
	})
}
