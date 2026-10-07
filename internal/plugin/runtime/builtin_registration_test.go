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
	builtinEngineDigest      = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

func builtinRuntimeEngine() pruntime.RuntimeEngineIdentity {
	return pruntime.RuntimeEngineIdentity{
		ID:                   core.MustCanonicalID("tei.runtime.engine.static_response.reference"),
		Version:              "1.0.0",
		ContractVersion:      "tei.runtime-engine/static-response-v1",
		ImplementationDigest: builtinEngineDigest,
		Target:               "host/reference",
	}
}

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



type countingBuiltin struct {
	id      plugin.Identity
	fp      string
	invoked *int
}

func (b countingBuiltin) Identity() plugin.Identity { return b.id }
func (b countingBuiltin) DescriptorFingerprint() string { return b.fp }
func (b countingBuiltin) Invoke(_ context.Context, _ pruntime.Request) (pruntime.Response, error) {
	(*b.invoked)++
	raw, _ := json.Marshal(executor.Result{AttemptID: "replacement", Outcome: executor.OutcomeAccepted})
	return pruntime.Response{Status: pruntime.ResponseOK, Payload: raw}, nil
}

type swappingBuiltin struct {
	id           plugin.Identity
	fp           string
	key          string
	registrations map[string]pruntime.BuiltinRegistration
	replacement  pruntime.BuiltinRegistration
	invoked      *int
}

func (b swappingBuiltin) Identity() plugin.Identity { return b.id }
func (b swappingBuiltin) DescriptorFingerprint() string {
	b.registrations[b.key] = b.replacement
	return b.fp
}
func (b swappingBuiltin) Invoke(_ context.Context, _ pruntime.Request) (pruntime.Response, error) {
	*b.invoked++
	raw, _ := json.Marshal(executor.Result{AttemptID: "accepted-original", Outcome: executor.OutcomeAccepted})
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
			Locator: "plugin-implementations/tei-native-runtime-reference/1.0.0/implementation.tei",
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
			Entrypoint: "plugin-implementations/tei-native-runtime-reference/1.0.0/implementation.tei",
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "interpreted-builtin/host",
		Profile:    "reference",
		Parameters: []plugin.BuildParameter{
			{Name: "execution-model", Value: "interpreted-builtin"},
		},
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
	engine := builtinRuntimeEngine()
	engineLock := pruntime.RuntimeEngineLock{
		FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
		Bindings: []pruntime.RuntimeEngineBinding{{
			Plugin: plugin.Identity{ID: realized.ID, Version: realized.Version},
			Engine: engine,
		}},
	}
	plan, err = pruntime.BindRuntimeEngineLock(plan, engineLock)
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
			RuntimeEngine:        engine,
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
		{"runtime engine", func(r *pruntime.BuiltinRegistration) {
			r.RuntimeEngine.ImplementationDigest = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
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
	engineFingerprint, err := pruntime.RuntimeEngineIdentityFingerprint(fx.reg.RuntimeEngine)
	if err != nil {
		t.Fatal(err)
	}
	if result.PluginInvocation.RuntimeEngineFingerprint != engineFingerprint {
		t.Fatalf(
			"runtime engine fingerprint: got %s want %s",
			result.PluginInvocation.RuntimeEngineFingerprint,
			engineFingerprint,
		)
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

	tamperedEngine := *result.PluginInvocation
	tamperedEngine.RuntimeEngineFingerprint = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if err := pruntime.ValidateDeploymentInvocation(fx.plan, core.MustCanonicalID("runtime.native_reference"), &tamperedEngine); err == nil {
		t.Fatal("expected tampered runtime engine provenance to fail")
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


func TestBuildBoundBuiltinExecutesExactPreflightAcceptedInstance(t *testing.T) {
	fx := newBuildBoundBuiltinFixture(t)
	key := "tei.native.runtime.reference@1.0.0"
	registrations := map[string]pruntime.BuiltinRegistration{}
	originalInvoked := 0
	replacementInvoked := 0

	replacement := pruntime.BuiltinRegistration{
		Builtin: countingBuiltin{
			id:      fx.reg.Builtin.Identity(),
			fp:      fx.reg.Builtin.DescriptorFingerprint(),
			invoked: &replacementInvoked,
		},
		BuildPlanFingerprint: fx.reg.BuildPlanFingerprint,
		RealizationDigest:    fx.reg.RealizationDigest,
		RuntimeEngine:        fx.reg.RuntimeEngine,
	}
	original := swappingBuiltin{
		id:            fx.reg.Builtin.Identity(),
		fp:            fx.reg.Builtin.DescriptorFingerprint(),
		key:           key,
		registrations: registrations,
		replacement:   replacement,
		invoked:       &originalInvoked,
	}
	registrations[key] = pruntime.BuiltinRegistration{
		Builtin:              original,
		BuildPlanFingerprint: fx.reg.BuildPlanFingerprint,
		RealizationDigest:    fx.reg.RealizationDigest,
		RuntimeEngine:        fx.reg.RuntimeEngine,
	}

	adapter := pruntime.DeploymentExecutor{
		Plan: fx.plan,
		Registry: pruntime.Registry{
			BuiltinRegistrations: registrations,
		},
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
	if originalInvoked != 1 || replacementInvoked != 0 {
		t.Fatalf("invocation counts: original=%d replacement=%d", originalInvoked, replacementInvoked)
	}
	if result.AttemptID != "accepted-original" {
		t.Fatalf("unexpected builtin instance executed: %#v", result)
	}
}


func TestBuildBoundBuiltinRequiresRuntimeEngineBinding(t *testing.T) {
	fx := newBuildBoundBuiltinFixture(t)
	for i := range fx.plan.Plugins {
		if fx.plan.Plugins[i].Plugin.ID == core.MustCanonicalID("tei.native.runtime.reference") {
			fx.plan.Plugins[i].RuntimeEngine = nil
		}
	}
	fx.plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(fx.plan)
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
		t.Fatal("expected missing runtime engine binding to fail")
	}
}

func TestRuntimeEngineBindingChangesRuntimePolicyFingerprint(t *testing.T) {
	fx := newBuildBoundBuiltinFixture(t)
	withEngine := fx.plan.RuntimePolicyFingerprint
	withoutEngine := fx.plan
	for i := range withoutEngine.Plugins {
		if withoutEngine.Plugins[i].Plugin.ID == core.MustCanonicalID("tei.native.runtime.reference") {
			withoutEngine.Plugins[i].RuntimeEngine = nil
		}
	}
	withoutFingerprint := pruntime.PolicyFingerprint(withoutEngine)
	if withEngine == withoutFingerprint {
		t.Fatal("runtime engine identity did not affect runtime policy fingerprint")
	}
}


func TestBindRuntimeEngineLockDoesNotMutateInputPlan(t *testing.T) {
	fx := newBuildBoundBuiltinFixture(t)
	input := fx.plan
	for i := range input.Plugins {
		if input.Plugins[i].Plugin.ID == core.MustCanonicalID("tei.native.runtime.reference") {
			input.Plugins[i].RuntimeEngine = nil
		}
	}
	input.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(input)
	before := input.RuntimePolicyFingerprint

	lock := pruntime.RuntimeEngineLock{
		FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
		Bindings: []pruntime.RuntimeEngineBinding{{
			Plugin: plugin.Identity{ID: fx.definition.ID, Version: fx.definition.Version},
			Engine: builtinRuntimeEngine(),
		}},
	}
	bound, err := pruntime.BindRuntimeEngineLock(input, lock)
	if err != nil {
		t.Fatal(err)
	}
	for _, pr := range input.Plugins {
		if pr.Plugin.ID == fx.definition.ID && pr.RuntimeEngine != nil {
			t.Fatal("binding mutated input runtime plan")
		}
	}
	if input.RuntimePolicyFingerprint != before {
		t.Fatal("binding mutated input runtime policy fingerprint")
	}
	var found bool
	for _, pr := range bound.Plugins {
		if pr.Plugin.ID != fx.definition.ID {
			continue
		}
		found = true
		if pr.RuntimeEngine == nil || *pr.RuntimeEngine != builtinRuntimeEngine() {
			t.Fatalf("bound runtime engine: %#v", pr.RuntimeEngine)
		}
	}
	if !found {
		t.Fatal("bound plugin missing")
	}
	if bound.RuntimePolicyFingerprint == before {
		t.Fatal("runtime engine binding did not change runtime policy fingerprint")
	}
}

func TestBindRuntimeEngineLockRejectsRebindingDifferentEngine(t *testing.T) {
	fx := newBuildBoundBuiltinFixture(t)
	other := builtinRuntimeEngine()
	other.ImplementationDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	lock := pruntime.RuntimeEngineLock{
		FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
		Bindings: []pruntime.RuntimeEngineBinding{{
			Plugin: plugin.Identity{ID: fx.definition.ID, Version: fx.definition.Version},
			Engine: other,
		}},
	}
	if _, err := pruntime.BindRuntimeEngineLock(fx.plan, lock); err == nil {
		t.Fatal("expected different runtime engine rebinding to fail")
	}
}

func TestRuntimeEngineIdentityRejectsMalformedDigest(t *testing.T) {
	engine := builtinRuntimeEngine()
	engine.ImplementationDigest = "sha256:not-a-digest"
	if _, err := pruntime.RuntimeEngineIdentityFingerprint(engine); err == nil {
		t.Fatal("expected malformed runtime engine digest to fail")
	}
}
