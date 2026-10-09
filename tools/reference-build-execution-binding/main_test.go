package main

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

const (
	digestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	digestC = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	digestD = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

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
