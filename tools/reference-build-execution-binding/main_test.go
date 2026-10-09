package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

const (
	digestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	digestC = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	digestD = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

func uppercaseDigest(value string) string {
	return "sha256:" + strings.ToUpper(strings.TrimPrefix(value, "sha256:"))
}

func referencePluginBuildPlan(id string, definitionFingerprint string) plugin.BuildPlan {
	return plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: core.MustCanonicalID(id), Version: "1.0.0"},
		PluginDefinitionFingerprint: definitionFingerprint,
		Sources: []plugin.BuildSource{{
			Kind:    "reference-source",
			ID:      "shared-source",
			Locator: "reference://shared-source",
			Digest:  digestA,
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.reference"),
			Version:         "1.0.0",
			ContractVersion: "tei.reference.build/v1",
			ArtifactDigest:  digestB,
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "deterministic-reference",
			Entrypoint: "shared-source",
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "host/reference",
		Parameters: []plugin.BuildParameter{
			{Name: "mode", Value: "deterministic"},
		},
	}
}

func TestPluginBuildProjectionSeparatesExecutionFromLogicalSubject(t *testing.T) {
	planA := referencePluginBuildPlan("tei.reference.plugin_a", strings.Repeat("a", 64))
	planB := referencePluginBuildPlan("tei.reference.plugin_b", strings.Repeat("b", 64))
	buildA, err := plugin.LockBuildOutput(planA, digestC)
	if err != nil {
		t.Fatal(err)
	}
	buildB, err := plugin.LockBuildOutput(planB, digestC)
	if err != nil {
		t.Fatal(err)
	}

	executionA, setA, resultA, bindingA, err := projectPluginBuild(planA, buildA)
	if err != nil {
		t.Fatal(err)
	}
	executionB, setB, resultB, bindingB, err := projectPluginBuild(planB, buildB)
	if err != nil {
		t.Fatal(err)
	}

	executionFPA, _ := buildExecutionPlanFingerprint(executionA)
	executionFPB, _ := buildExecutionPlanFingerprint(executionB)
	if executionFPA != executionFPB {
		t.Fatal("Plugin identity leaked into domain-neutral build execution identity")
	}
	setFPA, _ := artifactSetFingerprint(setA)
	setFPB, _ := artifactSetFingerprint(setB)
	if setFPA != setFPB {
		t.Fatal("same physical output bytes produced different Artifact Set identity")
	}
	resultFPA, _ := buildExecutionResultFingerprint(resultA)
	resultFPB, _ := buildExecutionResultFingerprint(resultB)
	if resultFPA != resultFPB {
		t.Fatal("same execution and same physical outputs produced different build result identity")
	}
	bindingFPA, _ := pluginSubjectBindingFingerprint(bindingA)
	bindingFPB, _ := pluginSubjectBindingFingerprint(bindingB)
	if bindingFPA == bindingFPB {
		t.Fatal("different logical Plugins collapsed into one subject binding identity")
	}
}

func TestSameArtifactBytesDoNotCollapseDifferentBuildExecutions(t *testing.T) {
	planA := referencePluginBuildPlan("tei.reference.plugin", strings.Repeat("a", 64))
	planB := planA
	planB.Sources = append([]plugin.BuildSource(nil), planA.Sources...)
	planB.Sources[0].Digest = digestD

	buildA, err := plugin.LockBuildOutput(planA, digestC)
	if err != nil {
		t.Fatal(err)
	}
	buildB, err := plugin.LockBuildOutput(planB, digestC)
	if err != nil {
		t.Fatal(err)
	}

	executionA, setA, resultA, _, err := projectPluginBuild(planA, buildA)
	if err != nil {
		t.Fatal(err)
	}
	executionB, setB, resultB, _, err := projectPluginBuild(planB, buildB)
	if err != nil {
		t.Fatal(err)
	}

	executionFPA, _ := buildExecutionPlanFingerprint(executionA)
	executionFPB, _ := buildExecutionPlanFingerprint(executionB)
	if executionFPA == executionFPB {
		t.Fatal("different Source closures collapsed into one execution identity")
	}
	setFPA, _ := artifactSetFingerprint(setA)
	setFPB, _ := artifactSetFingerprint(setB)
	if setFPA != setFPB {
		t.Fatal("same physical bytes should retain the same content-set identity")
	}
	resultFPA, _ := buildExecutionResultFingerprint(resultA)
	resultFPB, _ := buildExecutionResultFingerprint(resultB)
	if resultFPA == resultFPB {
		t.Fatal("different executions with the same bytes collapsed into one result identity")
	}
}

func TestArtifactSetIdentityIsOrderIndependentAndLocatorIndependent(t *testing.T) {
	left := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{
			{Role: "manifest", Format: "application/json", Digest: digestA, Locator: "/tmp/a/manifest.json"},
			{Role: "executable", Format: "application/octet-stream", Digest: digestB, Locator: "/tmp/a/app"},
		},
	}
	right := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{
			{Role: "executable", Format: "application/octet-stream", Digest: digestB, Locator: "store://other/app"},
			{Role: "manifest", Format: "application/json", Digest: digestA, Locator: "store://other/manifest"},
		},
	}
	leftFP, err := artifactSetFingerprint(left)
	if err != nil {
		t.Fatal(err)
	}
	rightFP, err := artifactSetFingerprint(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftFP != rightFP {
		t.Fatal("Artifact Set identity depends on declaration order or locator")
	}

	changed := right
	changed.Members = append([]ArtifactMember(nil), right.Members...)
	changed.Members[0].Digest = digestC
	changedFP, err := artifactSetFingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedFP == leftFP {
		t.Fatal("Artifact Set identity ignored member digest change")
	}
}

func TestMultiOutputBindingSelectsRolesWithoutUsingFilesystemPaths(t *testing.T) {
	plan := BuildExecutionPlan{
		FormatVersion: buildExecutionPlanFormatVersion,
		Sources: []plugin.BuildSource{{
			Kind: "source", ID: "app", Locator: "source://app", Digest: digestA,
		}},
		Builder: plugin.BuilderIdentity{
			ID: core.MustCanonicalID("tei.builder.web_reference"), Version: "1.0.0",
			ContractVersion: "tei.reference.build/v1", ArtifactDigest: digestB,
		},
		Recipe: plugin.BuildRecipe{Kind: "web-reference", Entrypoint: "app"},
		OutputContract: "web-artifact-set",
		Target: "web/reference",
	}
	executionFP, err := buildExecutionPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	set := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{
			{Role: "web-script", Format: "text/javascript", Digest: digestC, Locator: "/tmp/site/app.js"},
			{Role: "resource-manifest", Format: "application/json", Digest: digestD, Locator: "/tmp/site/manifest.json"},
		},
	}
	setFP, err := artifactSetFingerprint(set)
	if err != nil {
		t.Fatal(err)
	}
	binding := PluginSubjectBinding{
		FormatVersion: pluginSubjectBindingFormatVersion,
		Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.reference.web_plugin"), Version: "1.0.0"},
		PluginDefinitionFingerprint: strings.Repeat("e", 64),
		ExecutionPlanFingerprint: executionFP,
		ArtifactSetFingerprint: setFP,
		OutputRoles: []string{"resource-manifest", "web-script"},
	}
	if err := verifyPluginSubjectBinding(binding, plan, set); err != nil {
		t.Fatal(err)
	}

	binding.OutputRoles = []string{"missing-role"}
	if err := verifyPluginSubjectBinding(binding, plan, set); err == nil {
		t.Fatal("binding accepted an output role absent from the Artifact Set")
	}
}

func TestOutputContractRemainsPartOfExecutionIdentity(t *testing.T) {
	plan := referencePluginBuildPlan("tei.reference.plugin", strings.Repeat("a", 64))
	runtimeExecution, err := executionPlanFromPluginBuildPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	packagePlan := plan
	packagePlan.OutputKind = plugin.BuildOutputPluginPackage
	packageExecution, err := executionPlanFromPluginBuildPlan(packagePlan)
	if err != nil {
		t.Fatal(err)
	}
	runtimeFP, _ := buildExecutionPlanFingerprint(runtimeExecution)
	packageFP, _ := buildExecutionPlanFingerprint(packageExecution)
	if runtimeFP == packageFP {
		t.Fatal("BuildPlan output semantics were lost in execution projection")
	}
}

func TestArtifactSetRejectsDuplicateRole(t *testing.T) {
	set := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{
			{Role: "primary", Format: "application/octet-stream", Digest: digestA},
			{Role: "primary", Format: "application/json", Digest: digestB},
		},
	}
	if _, err := artifactSetFingerprint(set); err == nil {
		t.Fatal("Artifact Set accepted duplicate role in the first Reference profile")
	}
}


func TestExecutionIdentityIgnoresSourceLocatorWhenExactBytesMatch(t *testing.T) {
	planA := referencePluginBuildPlan("tei.reference.plugin", strings.Repeat("a", 64))
	planB := planA
	planB.Sources = append([]plugin.BuildSource(nil), planA.Sources...)
	planB.Sources[0].Locator = "store://alternate/shared-source"

	executionA, err := executionPlanFromPluginBuildPlan(planA)
	if err != nil {
		t.Fatal(err)
	}
	executionB, err := executionPlanFromPluginBuildPlan(planB)
	if err != nil {
		t.Fatal(err)
	}
	fpA, err := buildExecutionPlanFingerprint(executionA)
	if err != nil {
		t.Fatal(err)
	}
	fpB, err := buildExecutionPlanFingerprint(executionB)
	if err != nil {
		t.Fatal(err)
	}
	if fpA != fpB {
		t.Fatal("source locator leaked into execution input identity despite identical exact bytes")
	}
}


func TestCompareArtifactSetsPreservesMissingExtraAndChangedEvidence(t *testing.T) {
	first := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{
			{Role: "executable", Format: "application/octet-stream", Digest: digestA},
			{Role: "manifest", Format: "application/json", Digest: digestB},
			{Role: "symbols", Format: "application/octet-stream", Digest: digestC},
		},
	}
	second := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{
			{Role: "executable", Format: "application/octet-stream", Digest: digestD},
			{Role: "manifest", Format: "application/json", Digest: digestB},
			{Role: "resource-map", Format: "application/json", Digest: digestA},
		},
	}
	comparison, err := compareArtifactSets(first, second)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.Reproducible {
		t.Fatal("different Artifact Sets were reported as reproducible")
	}
	if len(comparison.MissingFromSecond) != 1 || comparison.MissingFromSecond[0] != "symbols" {
		t.Fatalf("missing roles: %#v", comparison.MissingFromSecond)
	}
	if len(comparison.ExtraInSecond) != 1 || comparison.ExtraInSecond[0] != "resource-map" {
		t.Fatalf("extra roles: %#v", comparison.ExtraInSecond)
	}
	if len(comparison.Changed) != 1 || comparison.Changed[0].Role != "executable" {
		t.Fatalf("changed members: %#v", comparison.Changed)
	}
	if comparison.Changed[0].FirstDigest != digestA || comparison.Changed[0].SecondDigest != digestD {
		t.Fatalf("changed digest evidence: %#v", comparison.Changed[0])
	}
}

func TestCompareArtifactSetsIgnoresLocatorOnlyChanges(t *testing.T) {
	first := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{{
			Role: "primary", Format: "application/octet-stream", Digest: digestA, Locator: "/tmp/first",
		}},
	}
	second := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{{
			Role: "primary", Format: "application/octet-stream", Digest: digestA, Locator: "store://second",
		}},
	}
	comparison, err := compareArtifactSets(first, second)
	if err != nil {
		t.Fatal(err)
	}
	if !comparison.Reproducible {
		t.Fatalf("locator-only change was treated as output content drift: %#v", comparison)
	}
}


func TestOneExecutionCanBindMultiplePluginsWithoutDuplicatingArtifactResult(t *testing.T) {
	plan := BuildExecutionPlan{
		FormatVersion: buildExecutionPlanFormatVersion,
		Sources: []plugin.BuildSource{{
			Kind: "source", ID: "shared", Locator: "source://shared", Digest: digestA,
		}},
		Builder: plugin.BuilderIdentity{
			ID: core.MustCanonicalID("tei.builder.shared_reference"), Version: "1.0.0",
			ContractVersion: "tei.reference.build/v1", ArtifactDigest: digestB,
		},
		Recipe: plugin.BuildRecipe{Kind: "shared-reference", Entrypoint: "shared"},
		OutputContract: "shared-runtime-artifact",
		Target: "host/reference",
	}
	executionFP, err := buildExecutionPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	set := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{{
			Role: "primary", Format: "application/octet-stream", Digest: digestC,
		}},
	}
	setFP, err := artifactSetFingerprint(set)
	if err != nil {
		t.Fatal(err)
	}
	result := BuildExecutionResult{
		FormatVersion: buildExecutionResultFormatVersion,
		ExecutionPlanFingerprint: executionFP,
		ArtifactSetFingerprint: setFP,
	}
	if _, err := buildExecutionResultFingerprint(result); err != nil {
		t.Fatal(err)
	}

	bindings := []PluginSubjectBinding{
		{
			FormatVersion: pluginSubjectBindingFormatVersion,
			Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.reference.plugin_a"), Version: "1.0.0"},
			PluginDefinitionFingerprint: strings.Repeat("a", 64),
			ExecutionPlanFingerprint: executionFP,
			ArtifactSetFingerprint: setFP,
			OutputRoles: []string{"primary"},
		},
		{
			FormatVersion: pluginSubjectBindingFormatVersion,
			Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.reference.plugin_b"), Version: "1.0.0"},
			PluginDefinitionFingerprint: strings.Repeat("b", 64),
			ExecutionPlanFingerprint: executionFP,
			ArtifactSetFingerprint: setFP,
			OutputRoles: []string{"primary"},
		},
	}
	for _, binding := range bindings {
		if err := verifyPluginSubjectBinding(binding, plan, set); err != nil {
			t.Fatal(err)
		}
	}
	fpA, _ := pluginSubjectBindingFingerprint(bindings[0])
	fpB, _ := pluginSubjectBindingFingerprint(bindings[1])
	if fpA == fpB {
		t.Fatal("multiple logical Plugins collapsed into one binding identity")
	}
}

func TestRuntimeEngineSubjectBindingDoesNotRequirePluginIdentity(t *testing.T) {
	plan := BuildExecutionPlan{
		FormatVersion: buildExecutionPlanFormatVersion,
		Sources: []plugin.BuildSource{{
			Kind: "go-source-closure", ID: "transform-engine", Locator: "source://engine", Digest: digestA,
		}},
		Builder: plugin.BuilderIdentity{
			ID: core.MustCanonicalID("tei.builder.go_reference"), Version: "1.0.0",
			ContractVersion: "tei.reference.build/v1", ArtifactDigest: digestB,
		},
		Recipe: plugin.BuildRecipe{Kind: "runtime-engine", Entrypoint: "transform-engine"},
		OutputContract: "runtime-engine-executable",
		Target: "host/reference",
	}
	executionFP, err := buildExecutionPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	set := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{{
			Role: "runtime-engine", Format: "application/octet-stream", Digest: digestC,
		}},
	}
	setFP, err := artifactSetFingerprint(set)
	if err != nil {
		t.Fatal(err)
	}
	binding := RuntimeEngineSubjectBinding{
		FormatVersion: runtimeEngineSubjectBindingFormatVersion,
		Engine: pruntime.RuntimeEngineIdentity{
			ID: core.MustCanonicalID("tei.runtime_engine.reference"),
			Version: "1.0.0",
			ContractVersion: "tei.transform-runtime/v1",
			ImplementationDigest: digestC,
			Target: "host/reference",
		},
		ExecutionPlanFingerprint: executionFP,
		ArtifactSetFingerprint: setFP,
		OutputRole: "runtime-engine",
	}
	if err := verifyRuntimeEngineSubjectBinding(binding, plan, set); err != nil {
		t.Fatal(err)
	}
	if _, err := runtimeEngineSubjectBindingFingerprint(binding); err != nil {
		t.Fatal(err)
	}

	binding.Engine.ImplementationDigest = digestD
	if err := verifyRuntimeEngineSubjectBinding(binding, plan, set); err == nil {
		t.Fatal("runtime engine binding accepted an artifact member with a different implementation digest")
	}
}


func TestReferenceEvidenceIsDeterministicAndKeepsAuthoritySeparated(t *testing.T) {
	first, err := buildReferenceEvidence()
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildReferenceEvidence()
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := fingerprintJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := fingerprintJSON(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstJSON != secondJSON {
		t.Fatal("reference Evidence is not deterministic")
	}
	if first.Schema != "tei.reference.build-execution-binding/v0" {
		t.Fatalf("unexpected Evidence schema %q", first.Schema)
	}
	if len(first.SharedPluginBindings) != 2 || len(first.SharedPluginBindingFingerprints) != 2 {
		t.Fatalf("shared Plugin binding evidence: %#v", first.SharedPluginBindings)
	}
	if first.SharedPluginBindingFingerprints[0] == first.SharedPluginBindingFingerprints[1] {
		t.Fatal("distinct logical Plugins share one binding fingerprint")
	}
	if first.MultiOutputComparison.Reproducible {
		t.Fatal("divergent multi-output fixture was reported as reproducible")
	}
	if len(first.MultiOutputComparison.MissingFromSecond) == 0 ||
		len(first.MultiOutputComparison.ExtraInSecond) == 0 ||
		len(first.MultiOutputComparison.Changed) == 0 {
		t.Fatalf("multi-output difference Evidence is incomplete: %#v", first.MultiOutputComparison)
	}
	if first.RuntimeEngineBinding.Engine.ID == "" || first.RuntimeEngineBindingFingerprint == "" {
		t.Fatalf("Runtime Engine binding Evidence is incomplete: %#v", first.RuntimeEngineBinding)
	}
	if first.BuildGraphFingerprint == "" {
		t.Fatal("Build graph fingerprint is missing from Reference Evidence")
	}
	if len(first.BuildGraphOrder) != 2 ||
		first.BuildGraphOrder[0] != "stage1" ||
		first.BuildGraphOrder[1] != "stage2" {
		t.Fatalf("bootstrap Build graph order: %#v", first.BuildGraphOrder)
	}
	if len(first.BuildGraph.Seeds) != 1 ||
		first.BuildGraph.Seeds[0].TrustBoundary != "external-stage0-seed" {
		t.Fatalf("stage0 trust boundary is missing from Build graph Evidence: %#v", first.BuildGraph.Seeds)
	}
	if normalizeDigest(first.BuildGraph.Seeds[0].ArtifactDigest) !=
		normalizeDigest(first.BootstrapStage1Plan.Builder.ArtifactDigest) {
		t.Fatal("stage0 seed does not bind the exact stage1 builder")
	}
	if len(first.BuildGraph.Edges) != 1 ||
		first.BuildGraph.Edges[0].FromNode != "stage1" ||
		first.BuildGraph.Edges[0].ToNode != "stage2" ||
		first.BuildGraph.Edges[0].OutputRole != "builder-artifact" {
		t.Fatalf("stage1 -> stage2 Build graph edge is incomplete: %#v", first.BuildGraph.Edges)
	}
	for _, forbidden := range []string{"accepted", "runtime-authorized"} {
		for _, limit := range first.AuthorityLimits {
			if limit == forbidden {
				t.Fatalf("Reference Evidence claims forbidden authority marker %q", limit)
			}
		}
	}
}


func TestExecutionIdentityCanonicalizesValidSHA256HexCase(t *testing.T) {
	lower := referencePluginBuildPlan("tei.reference.plugin", strings.Repeat("a", 64))
	lower.Dependencies = []plugin.BuildDependency{{
		Kind: "sdk", ID: "reference-sdk", Version: "1.0.0", ArtifactDigest: digestC,
	}}
	upper := lower
	upper.Sources = append([]plugin.BuildSource(nil), lower.Sources...)
	upper.Dependencies = append([]plugin.BuildDependency(nil), lower.Dependencies...)
	upper.Sources[0].Digest = uppercaseDigest(lower.Sources[0].Digest)
	upper.Builder.ArtifactDigest = uppercaseDigest(lower.Builder.ArtifactDigest)
	upper.Dependencies[0].ArtifactDigest = uppercaseDigest(lower.Dependencies[0].ArtifactDigest)

	lowerExecution, err := executionPlanFromPluginBuildPlan(lower)
	if err != nil {
		t.Fatal(err)
	}
	upperExecution, err := executionPlanFromPluginBuildPlan(upper)
	if err != nil {
		t.Fatal(err)
	}
	lowerFP, err := buildExecutionPlanFingerprint(lowerExecution)
	if err != nil {
		t.Fatal(err)
	}
	upperFP, err := buildExecutionPlanFingerprint(upperExecution)
	if err != nil {
		t.Fatal(err)
	}
	if lowerFP != upperFP {
		t.Fatal("valid SHA-256 hex letter case changed BuildExecutionPlan identity")
	}
}

func TestArtifactSetIdentityCanonicalizesValidSHA256HexCase(t *testing.T) {
	lower := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{{
			Role: "primary", Format: "application/octet-stream", Digest: digestC,
		}},
	}
	upper := lower
	upper.Members = append([]ArtifactMember(nil), lower.Members...)
	upper.Members[0].Digest = uppercaseDigest(lower.Members[0].Digest)

	lowerFP, err := artifactSetFingerprint(lower)
	if err != nil {
		t.Fatal(err)
	}
	upperFP, err := artifactSetFingerprint(upper)
	if err != nil {
		t.Fatal(err)
	}
	if lowerFP != upperFP {
		t.Fatal("valid SHA-256 hex letter case changed Artifact Set identity")
	}
	comparison, err := compareArtifactSets(lower, upper)
	if err != nil {
		t.Fatal(err)
	}
	if !comparison.Reproducible {
		t.Fatalf("hex-case-only artifact difference was treated as content drift: %#v", comparison)
	}
}

func TestSubjectBindingFingerprintsCanonicalizeValidHexCase(t *testing.T) {
	plan := referencePluginBuildPlan("tei.reference.plugin", strings.Repeat("a", 64))
	execution, err := executionPlanFromPluginBuildPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	executionFP, err := buildExecutionPlanFingerprint(execution)
	if err != nil {
		t.Fatal(err)
	}
	set := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{{
			Role: "primary", Format: "application/octet-stream", Digest: digestC,
		}},
	}
	setFP, err := artifactSetFingerprint(set)
	if err != nil {
		t.Fatal(err)
	}

	lowerPlugin := PluginSubjectBinding{
		FormatVersion: pluginSubjectBindingFormatVersion,
		Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.reference.plugin"), Version: "1.0.0"},
		PluginDefinitionFingerprint: strings.Repeat("a", 64),
		ExecutionPlanFingerprint: executionFP,
		ArtifactSetFingerprint: setFP,
		OutputRoles: []string{"primary"},
	}
	upperPlugin := lowerPlugin
	upperPlugin.PluginDefinitionFingerprint = strings.ToUpper(lowerPlugin.PluginDefinitionFingerprint)
	upperPlugin.ExecutionPlanFingerprint = strings.ToUpper(lowerPlugin.ExecutionPlanFingerprint)
	upperPlugin.ArtifactSetFingerprint = strings.ToUpper(lowerPlugin.ArtifactSetFingerprint)

	lowerPluginFP, err := pluginSubjectBindingFingerprint(lowerPlugin)
	if err != nil {
		t.Fatal(err)
	}
	upperPluginFP, err := pluginSubjectBindingFingerprint(upperPlugin)
	if err != nil {
		t.Fatal(err)
	}
	if lowerPluginFP != upperPluginFP {
		t.Fatal("valid fingerprint hex letter case changed Plugin subject binding identity")
	}
	if err := verifyPluginSubjectBinding(upperPlugin, execution, set); err != nil {
		t.Fatalf("valid uppercase Plugin binding was rejected: %v", err)
	}

	lowerEngine := RuntimeEngineSubjectBinding{
		FormatVersion: runtimeEngineSubjectBindingFormatVersion,
		Engine: pruntime.RuntimeEngineIdentity{
			ID: core.MustCanonicalID("tei.runtime_engine.reference"),
			Version: "1.0.0",
			ContractVersion: "tei.transform-runtime/v1",
			ImplementationDigest: digestC,
			Target: "host/reference",
		},
		ExecutionPlanFingerprint: executionFP,
		ArtifactSetFingerprint: setFP,
		OutputRole: "primary",
	}
	upperEngine := lowerEngine
	upperEngine.Engine.ImplementationDigest = uppercaseDigest(lowerEngine.Engine.ImplementationDigest)
	upperEngine.ExecutionPlanFingerprint = strings.ToUpper(lowerEngine.ExecutionPlanFingerprint)
	upperEngine.ArtifactSetFingerprint = strings.ToUpper(lowerEngine.ArtifactSetFingerprint)

	lowerEngineFP, err := runtimeEngineSubjectBindingFingerprint(lowerEngine)
	if err != nil {
		t.Fatal(err)
	}
	upperEngineFP, err := runtimeEngineSubjectBindingFingerprint(upperEngine)
	if err != nil {
		t.Fatal(err)
	}
	if lowerEngineFP != upperEngineFP {
		t.Fatal("valid SHA-256/fingerprint hex letter case changed Runtime Engine binding identity")
	}
	if err := verifyRuntimeEngineSubjectBinding(upperEngine, execution, set); err != nil {
		t.Fatalf("valid uppercase Runtime Engine binding was rejected: %v", err)
	}
}

func TestCurrentBuildPlanWithUppercaseHexCanBeProjected(t *testing.T) {
	plan := referencePluginBuildPlan("tei.reference.plugin", strings.Repeat("A", 64))
	plan.Sources[0].Digest = uppercaseDigest(plan.Sources[0].Digest)
	plan.Builder.ArtifactDigest = uppercaseDigest(plan.Builder.ArtifactDigest)
	build, err := plugin.LockBuildOutput(plan, uppercaseDigest(digestC))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := projectPluginBuild(plan, build); err != nil {
		t.Fatalf("current valid BuildPlan with uppercase hex could not be projected: %v", err)
	}
}
