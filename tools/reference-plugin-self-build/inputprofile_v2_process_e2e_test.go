package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/inputprofile"
	"tei/internal/plugin/transform"
)

// TestOptInV2InputWithExactV1ProcessEngine exercises a genuine, separately
// compiled deterministic-transform-v1 process. The shipped SB-1 builder Source,
// wire protocol and process engine instruction set are left unchanged.
//
// This is reference evidence of explicit ingress selection, not a production
// Host authorization or a new self-build stage.
func TestOptInV2InputWithExactV1ProcessEngine(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go executable unavailable: %v", err)
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(repo, filepath.FromSlash(transformPackageSourcePath))
	sourceRaw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	source, err := transform.ParseSourceJSON(sourceRaw)
	if err != nil {
		t.Fatal(err)
	}
	lockedSourceFingerprint, err := transform.SourceFingerprint(source)
	if err != nil {
		t.Fatal(err)
	}

	engineExecutable := filepath.Join(t.TempDir(), "exact-transform-v1")
	if runtime.GOOS == "windows" {
		engineExecutable += ".exe"
	}
	compile := exec.Command(goPath, "build", "-trimpath", "-buildvcs=false", "-mod=readonly", "-o", engineExecutable, "./tools/reference-transform-runtime-engine")
	compile.Dir = repo
	compile.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off")
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("build exact v1 transform process engine: %v: %s", err, output)
	}
	engineDigest, err := hashFile(engineExecutable)
	if err != nil {
		t.Fatal(err)
	}

	const fixtureDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const builderDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	tests := []struct {
		name       string
		capability bool
		raw        string
	}{
		{
			name: "explicit empty top-level arrays and reordered JSON",
			raw:  `{"externalRequirements":[],"requires":[],"provides":[],"version":"1.0.0","runtime":{"args":[],"kind":"builtin"},"id":"tei.inputprofile.process"}`,
		},
		{
			name: "explicit nested capability arrays and reordered JSON",
			capability: true,
			raw:        `{"runtime":{"args":[],"kind":"builtin"},"provides":[{"permissions":[],"actions":[],"id":"runtime.inputprofile","kind":"deployment-executor"}],"externalRequirements":[],"version":"1.0.0","id":"tei.inputprofile.process","requires":[]}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			definition := plugin.Definition{
				ID:      core.MustCanonicalID("tei.inputprofile.process"),
				Version: "1.0.0",
				Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
			}
			if tc.capability {
				definition.Provides = []plugin.Capability{{
					Kind: plugin.CapabilityDeploymentExecutor,
					ID:   core.MustCanonicalID("runtime.inputprofile"),
				}}
			}
			plan := plugin.BuildPlan{
				FormatVersion:               plugin.BuildPlanFormatVersion,
				Plugin:                      plugin.Identity{ID: definition.ID, Version: definition.Version},
				PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(definition),
				Sources:                     []plugin.BuildSource{{Kind: "plugin-definition", ID: "definition", Locator: "definition.tei", Digest: fixtureDigest}},
				Builder: plugin.BuilderIdentity{
					ID:              core.MustCanonicalID("tei.builder.plugin_package"),
					Version:         "1.0.0",
					ContractVersion: "tei.plugin-transform/v1",
					ArtifactDigest:  builderDigest,
				},
				Recipe:     plugin.BuildRecipe{Kind: "tei-plugin-package", Entrypoint: "definition.tei"},
				OutputKind: plugin.BuildOutputPluginPackage,
				Target:     "plugin-package",
				Profile:    "reference",
			}
			oracle, err := plugin.MaterializePluginPackage(definition, plan)
			if err != nil {
				t.Fatalf("Go SB-1 materialization rejected canonical fixture: %v", err)
			}
			planValue, err := transformValueFromJSON(plan)
			if err != nil {
				t.Fatal(err)
			}

			// Legacy ingress is unchanged: the real physical v1 process must
			// reject this noncanonical definition rather than silently repair it.
			legacyValue, err := transform.ParseJSON([]byte(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := evaluateWithExactEngine(engineExecutable, engineDigest, sourcePath, source, map[string]transform.Value{
				"buildPlan":        planValue,
				"pluginDefinition": legacyValue,
			}); err == nil {
				t.Fatal("unqualified legacy process unexpectedly accepted explicit empty arrays")
			}

			// Only an explicitly selected v2 ingress profile, bound to the
			// approved exact Source fingerprint, may normalize before invoking
			// the very same existing v1 process.
			canonical, evidence, err := inputprofile.CanonicalizePluginDefinition(
				inputprofile.PluginDefinitionJSONV2, source, lockedSourceFingerprint, []byte(tc.raw),
			)
			if err != nil {
				t.Fatalf("opt-in canonical ingress rejected valid definition: %v", err)
			}
			if evidence.Profile != inputprofile.PluginDefinitionJSONV2 ||
				evidence.SourceFingerprint != lockedSourceFingerprint ||
				evidence.OriginalInputDigest == evidence.CanonicalInputDigest ||
				!strings.HasPrefix(evidence.OriginalInputDigest, "sha256:") ||
				!strings.HasPrefix(evidence.CanonicalInputDigest, "sha256:") {
				t.Fatalf("missing exact ingress provenance: %+v", evidence)
			}
			var previous []byte
			for run := 0; run < 2; run++ {
				got, err := evaluateWithExactEngine(engineExecutable, engineDigest, sourcePath, source, map[string]transform.Value{
					"buildPlan":        planValue,
					"pluginDefinition": canonical,
				})
				if err != nil {
					t.Fatalf("exact v1 process with v2 ingress failed (run %d): %v", run+1, err)
				}
				if got.Output.Kind != transform.ValueBytes || !bytes.Equal(got.Output.Bytes, oracle) {
					t.Fatalf("exact process SB-1 output differed from Go oracle (run %d)", run+1)
				}
				if previous != nil && !bytes.Equal(previous, got.Output.Bytes) {
					t.Fatal("repeated exact process execution changed bytes")
				}
				previous = got.Output.Bytes
			}
			var manifest plugin.PluginPackageManifest
			if err := json.Unmarshal(previous, &manifest); err != nil {
				t.Fatalf("exact process emitted invalid SB-1 manifest: %v", err)
			}
			planFingerprint, err := plugin.BuildPlanFingerprint(plan)
			if err != nil {
				t.Fatal(err)
			}
			if manifest.BuildPlanFingerprint != planFingerprint ||
				manifest.PluginDefinitionFingerprint != plan.PluginDefinitionFingerprint {
				t.Fatal("exact process package lost canonical Definition or BuildPlan fingerprint")
			}
			if _, _, err := inputprofile.CanonicalizePluginDefinition(
				inputprofile.PluginDefinitionJSONV2, source, "different-locked-source", []byte(tc.raw),
			); err == nil {
				t.Fatal("unapproved Source fingerprint accepted at Host ingress")
			}
		})
	}
}
