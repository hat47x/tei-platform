package runtime_test

import (
	"context"
	"encoding/json"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/environment"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

const (
	builtinRealizationDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	builtinBuilderDigest     = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	builtinSourceDigest      = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

type registeredBuiltin struct {
	id plugin.Identity
	fp string
}

func (b registeredBuiltin) Identity() plugin.Identity      { return b.id }
func (b registeredBuiltin) DescriptorFingerprint() string { return b.fp }
func (b registeredBuiltin) Invoke(_ context.Context, _ pruntime.Request) (pruntime.Response, error) {
	raw, _ := json.Marshal(executor.Result{AttemptID: "tei-native-reference", Outcome: executor.OutcomeAccepted})
	return pruntime.Response{Status: pruntime.ResponseOK, Payload: raw}, nil
}

type buildBoundBuiltinFixture struct {
	definition plugin.Definition
	env        environment.Environment
	plan       pruntime.EnvironmentPlan
	build      plugin.LockedBuild
	reg        pruntime.BuiltinRegistration
}

func newBuildBoundBuiltinFixture(t *testing.T) buildBoundBuiltinFixture {
	t.Helper()

	source := plugin.Definition{
		ID:      core.MustCanonicalID("tei.native.runtime.reference"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{{
			Kind:    plugin.CapabilityDeploymentExecutor,
			ID:      core.MustCanonicalID("runtime.native_reference"),
			Actions: []deployment.ActionKind{deployment.ActionApplicationStart},
		}},
	}
	buildPlan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: source.ID, Version: source.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(source),
		Sources: []plugin.BuildSource{{
			Kind:    "plugin-implementation",
			ID:      "tei-native-runtime-reference",
			Locator: "plugins/tei-native-runtime-reference/1.0.0/implementation.tei",
			Digest:  builtinSourceDigest,
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.native.seed"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  builtinBuilderDigest,
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "tei-native-static-response",
			Entrypoint: "plugins/tei-native-runtime-reference/1.0.0/implementation.tei",
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "builtin/host",
		Profile:    "reference",
	}
	realized, err := plugin.RealizeBuiltDefinition(source, builtinRealizationDigest)
	if err != nil {
		t.Fatal(err)
	}
	env := environment.Environment{
		ID: core.MustCanonicalID("native_reference"),
		Plugins: []environment.ExtensionRequirement{{
			ID:              realized.ID,
			CompatibleMajor: 1,
		}},
		DeploymentExecutors: []environment.DeploymentExecutorBinding{{
			Action:   deployment.ActionApplicationStart,
			Scope:    environment.ExecutorScopeGlobal,
			Executor: core.MustCanonicalID("runtime.native_reference"),
		}},
	}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{realized}}
	lock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	lockedBuild, err := plugin.LockBuildOutput(buildPlan, builtinRealizationDigest)
	if err != nil {
		t.Fatal(err)
	}
	lock, err = plugin.BindRuntimeBuild(lock, env.ID, lockedBuild)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := pruntime.BuildPlan(catalog, env, lock)
	if err != nil {
		t.Fatal(err)
	}

	impl := registeredBuiltin{
		id: plugin.Identity{ID: realized.ID, Version: realized.Version},
		fp: plugin.DefinitionFingerprint(realized),
	}
	return buildBoundBuiltinFixture{
		definition: realized,
		env:        env,
		plan:       plan,
		build:      lockedBuild,
		reg: pruntime.BuiltinRegistration{
			Builtin:              impl,
			BuildPlanFingerprint: lockedBuild.PlanFingerprint,
			RealizationDigest:    builtinRealizationDigest,
		},
	}
}

func TestBuildBoundBuiltinRequiresAcceptedRegistration(t *testing.T) {
	fx := newBuildBoundBuiltinFixture(t)
	adapter := pruntime.DeploymentExecutor{
		Plan:         fx.plan,
		Registry:     pruntime.Registry{},
		CapabilityID: core.MustCanonicalID("runtime.native_reference"),
	}
	_, err := adapter.Execute(context.Background(), executor.Command{
		ActionID:       "app/start",
		IdempotencyKey: "k-native",
		Kind:           string(deployment.ActionApplicationStart),
	})
	if err == nil {
		t.Fatal("expected missing accepted builtin registration")
	}
}

func TestBuildBoundBuiltinRejectsRegistrationMismatch(t *testing.T) {
	tests := []struct {
		name string
		edit func(*pruntime.BuiltinRegistration)
	}{
		{"build plan", func(r *pruntime.BuiltinRegistration) {
			r.BuildPlanFingerprint = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
		}},
		{"realization digest", func(r *pruntime.BuiltinRegistration) {
			r.RealizationDigest = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
		}},
		{"descriptor identity", func(r *pruntime.BuiltinRegistration) {
			r.Builtin = registeredBuiltin{
				id: plugin.Identity{ID: core.MustCanonicalID("tei.native.runtime.reference"), Version: "1.0.0"},
				fp: "wrong",
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := newBuildBoundBuiltinFixture(t)
			registration := fx.reg
			tt.edit(&registration)
			registry := pruntime.Registry{BuiltinRegistrations: map[string]pruntime.BuiltinRegistration{
				"tei.native.runtime.reference@1.0.0": registration,
			}}
			adapter := pruntime.DeploymentExecutor{
				Plan:         fx.plan,
				Registry:     registry,
				CapabilityID: core.MustCanonicalID("runtime.native_reference"),
			}
			if _, err := adapter.Execute(context.Background(), executor.Command{
				ActionID:       "app/start",
				IdempotencyKey: "k-native",
				Kind:           string(deployment.ActionApplicationStart),
			}); err == nil {
				t.Fatal("expected mismatched builtin registration to be rejected")
			}
		})
	}
}

func TestBuildBoundBuiltinUsesRegistrationWithoutExternalArtifactVerifier(t *testing.T) {
	fx := newBuildBoundBuiltinFixture(t)
	registry := pruntime.Registry{BuiltinRegistrations: map[string]pruntime.BuiltinRegistration{
		"tei.native.runtime.reference@1.0.0": fx.reg,
	}}
	adapter := pruntime.DeploymentExecutor{
		Plan:         fx.plan,
		Registry:     registry,
		CapabilityID: core.MustCanonicalID("runtime.native_reference"),
	}
	result, err := adapter.Execute(context.Background(), executor.Command{
		ActionID:       "app/start",
		IdempotencyKey: "k-native",
		Kind:           string(deployment.ActionApplicationStart),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AttemptID != "tei-native-reference" || result.Outcome != executor.OutcomeAccepted {
		t.Fatalf("result: %#v", result)
	}
	if result.PluginInvocation == nil {
		t.Fatal("missing plugin invocation provenance")
	}
	if result.PluginInvocation.BuildPlanFingerprint != fx.build.PlanFingerprint {
		t.Fatalf("build plan fingerprint: got %s want %s", result.PluginInvocation.BuildPlanFingerprint, fx.build.PlanFingerprint)
	}
	if result.PluginInvocation.RealizationDigest != builtinRealizationDigest {
		t.Fatalf("realization digest: got %s", result.PluginInvocation.RealizationDigest)
	}
	if result.PluginInvocation.DigestVerified {
		t.Fatal("in-process accepted realization must not pretend external artifact verification occurred")
	}
	if err := pruntime.ValidateDeploymentInvocation(fx.plan, core.MustCanonicalID("runtime.native_reference"), result.PluginInvocation); err != nil {
		t.Fatal(err)
	}

	tampered := *result.PluginInvocation
	tampered.RealizationDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if err := pruntime.ValidateDeploymentInvocation(fx.plan, core.MustCanonicalID("runtime.native_reference"), &tampered); err == nil {
		t.Fatal("expected tampered realization provenance to fail")
	}
}


func TestBuildBoundBuiltinRejectsUnsupportedSignatureOrProvenanceMetadata(t *testing.T) {
	tests := []struct {
		name string
		edit func(*plugin.Artifact)
	}{
		{"signature", func(a *plugin.Artifact) {
			a.SignatureRef = "sigstore://reference"
		}},
		{"provenance", func(a *plugin.Artifact) {
			a.Provenance = "source://reference"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := newBuildBoundBuiltinFixture(t)
			for i := range fx.plan.Plugins {
				if fx.plan.Plugins[i].Plugin.ID == core.MustCanonicalID("tei.native.runtime.reference") {
					tt.edit(&fx.plan.Plugins[i].Artifact)
				}
			}
			registry := pruntime.Registry{BuiltinRegistrations: map[string]pruntime.BuiltinRegistration{
				"tei.native.runtime.reference@1.0.0": fx.reg,
			}}
			adapter := pruntime.DeploymentExecutor{
				Plan:         fx.plan,
				Registry:     registry,
				CapabilityID: core.MustCanonicalID("runtime.native_reference"),
			}
			if _, err := adapter.Execute(context.Background(), executor.Command{
				ActionID:       "app/start",
				IdempotencyKey: "k-native",
				Kind:           string(deployment.ActionApplicationStart),
			}); err == nil {
				t.Fatal("expected unsupported builtin artifact trust metadata to fail closed")
			}
		})
	}
}
