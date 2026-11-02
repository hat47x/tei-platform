package artifact

import (
	"strings"
	"testing"
)

// These are CHARACTERIZATION tests of PR #2587's generic ArtifactSet contract.
// They intentionally do not declare the observed omissions acceptable behavior.
// Once the owning issues adopt a consumer-scoped qualification policy, replace
// the corresponding expectations with the positive acceptance tests.
//
// AS-01/02: https://github.com/hat47x/tei-platform/pull/2630
// AC-01..04: https://github.com/hat47x/tei-platform/issues/2482
func TestCharacterizeAS01SameRevisionDifferentApplications(t *testing.T) {
	a, err := RenderFunctionDefinition(functionDefinitionInput())
	if err != nil {
		t.Fatal(err)
	}
	otherInput := functionDefinitionInput()
	otherInput.Application.ID = "other_application"
	b, err := RenderFunctionDefinition(otherInput)
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest == b.Digest {
		t.Fatal("the independent application contents must be distinguishable")
	}
	if len(a.Unresolved) != 0 || len(b.Unresolved) != 0 {
		t.Fatalf("the fixture must contain individually complete artifacts: a=%+v b=%+v", a.Unresolved, b.Unresolved)
	}
	manifest, err := NewManifest(a.InputRevision, []Artifact{a, b})
	if err != nil {
		t.Fatal(err)
	}
	set := WorkspaceSet{InputRevision: a.InputRevision, Artifacts: []Artifact{a, b}, Manifest: manifest}
	if !manifest.Complete {
		t.Fatal("current generic Complete reports only zero unresolved")
	}
	if findings := CheckSet(set); len(findings) != 0 {
		t.Fatalf("current generic CheckSet does not qualify application scope: %+v", findings)
	}
	// An application-specific reader still has to determine whether both
	// function definitions belong to the intended application.
	if !strings.Contains(a.Content, "application order_entry") || !strings.Contains(b.Content, "application other_application") {
		t.Fatalf("the two applications are not represented as intended: %q / %q", a.Content, b.Content)
	}
}

func TestCharacterizeAS02SubsetCanHaveACompleteManifest(t *testing.T) {
	full := manifestArtifacts(t, 0)
	if len(full) != 2 {
		t.Fatalf("the independent fixture expects two members: %d", len(full))
	}
	subset := []Artifact{full[0]}
	manifest, err := NewManifest(full[0].InputRevision, subset)
	if err != nil {
		t.Fatal(err)
	}
	set := WorkspaceSet{InputRevision: full[0].InputRevision, Artifacts: subset, Manifest: manifest}
	if !manifest.Complete {
		t.Fatal("generic Complete is based on the supplied subset, not an external expected-member inventory")
	}
	if findings := CheckSet(set); len(findings) != 0 {
		t.Fatalf("generic CheckSet does not know the omitted second expected member: %+v", findings)
	}
	// A separate expected-members oracle, not the newly built manifest, must
	// report that the consumer's original two-member requirement was unmet.
	if len(manifest.Entries) != 1 {
		t.Fatalf("expected a valid singleton manifest: %+v", manifest.Entries)
	}
}

func TestCharacterizeAC01MalformedSurfaceCanPassGenericCheckSet(t *testing.T) {
	const revision = "sha256:characterization"
	content := "{\"schemaVersion\":\"tei.operations.operation-surface/v0\","
	if _, err := decodeOperationSurface(content); err == nil {
		t.Fatal("the negative surface fixture must be malformed JSON")
	}
	a, err := New(KindOperationSurface, Rule{
		ID: OperationSurfaceRuleID, Version: OperationSurfaceRuleVersion, Fingerprint: "sha256:fixture-rule",
	}, OperationSurfaceGenerator, revision, content, nil)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManifest(revision, []Artifact{a})
	if err != nil {
		t.Fatal(err)
	}
	set := WorkspaceSet{InputRevision: revision, Artifacts: []Artifact{a}, Manifest: m}
	if findings := CheckSet(set); len(findings) != 0 {
		t.Fatalf("characterization: malformed JSON is not yet reported by CheckSet: %+v", findings)
	}
}

func TestCharacterizeAC02PostManifestArtifactMutation(t *testing.T) {
	artifacts := manifestArtifacts(t, 0)
	m, err := NewManifest(artifacts[0].InputRevision, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	// Modify a content byte after NewManifest completed, but leave each
	// Artifact.Digest and the manifest as previously reported.
	artifacts[0].Content += "\n"
	if err := artifacts[0].Verify(); err == nil {
		t.Fatal("Artifact.Verify must independently detect the changed bytes")
	}
	set := WorkspaceSet{InputRevision: m.InputRevision, Artifacts: artifacts, Manifest: m}
	if findings := CheckSet(set); len(findings) != 0 {
		t.Fatalf("characterization: CheckSet currently does not call Artifact.Verify: %+v", findings)
	}
}

func TestCharacterizeAC03GapReasonMismatchWithValidDigests(t *testing.T) {
	set := negativeFixtureSet(t)
	found := false
	for i, a := range set.Artifacts {
		if a.Kind != string(KindOperationSurface) {
			continue
		}
		if len(a.Unresolved) == 0 {
			t.Fatal("the reference negative fixture must have surface gaps")
		}
		u := append([]Unresolved(nil), a.Unresolved...)
		u[0].Reason = "different-reason-from-surface"
		changed, err := New(KindOperationSurface, a.Rule, a.Generator, a.InputRevision, a.Content, u)
		if err != nil {
			t.Fatal(err)
		}
		set.Artifacts[i] = changed
		found = true
	}
	if !found {
		t.Fatal("operation surface missing from the negative fixture")
	}
	m, err := NewManifest(set.InputRevision, set.Artifacts)
	if err != nil {
		t.Fatal(err)
	}
	set.Manifest = m
	if findings := CheckSet(set); len(findings) != 0 {
		t.Fatalf("characterization: matching gap subjects can hide differing reasons: %+v", findings)
	}
}

func TestCharacterizeAC04SchemaFieldCanDisagreeWithInlineVersion(t *testing.T) {
	const revision = "sha256:characterization"
	content := "{\"schemaVersion\":\"different/v0\",\"note\":\"tei.operations.operation-surface/v0\",\"operations\":[],\"gaps\":[]}\n"
	p, err := decodeOperationSurface(content)
	if err != nil {
		t.Fatal(err)
	}
	if p.SchemaVersion == "tei.operations.operation-surface/v0" {
		t.Fatal("the actual schema field must differ from the version mentioned in note")
	}
	a, err := New(KindOperationSurface, Rule{
		ID: OperationSurfaceRuleID, Version: OperationSurfaceRuleVersion, Fingerprint: "sha256:fixture-rule",
	}, OperationSurfaceGenerator, revision, content, nil)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManifest(revision, []Artifact{a})
	if err != nil {
		t.Fatal(err)
	}
	set := WorkspaceSet{InputRevision: revision, Artifacts: []Artifact{a}, Manifest: m}
	if findings := CheckSet(set); len(findings) != 0 {
		t.Fatalf("characterization: generic substring search does not validate schemaVersion field: %+v", findings)
	}
}
