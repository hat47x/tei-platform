package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestStage0ReferenceFixtures(t *testing.T) {
	root := repositoryRoot(t)
	workspacePath := filepath.Join(root, "reference", "sui-adoption", "workspace")

	cases := []struct {
		name           string
		fixture        string
		wantValid      bool
		wantDiagnostic string
		wantInstances  int
	}{
		{name: "M1 root", fixture: "M1-root.valid.json", wantValid: true, wantInstances: 1},
		{name: "M2 three level", fixture: "M2-three-level.valid.json", wantValid: true, wantInstances: 3},
		{name: "M3 missing parent", fixture: "M3-missing-parent.invalid.json", wantValid: false, wantDiagnostic: "SUI_HIERARCHY_PARENT_MISSING", wantInstances: 1},
		{name: "M4 cycle", fixture: "M4-cycle.invalid.json", wantValid: false, wantDiagnostic: "SUI_HIERARCHY_CYCLE", wantInstances: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixturePath := filepath.Join(root, "reference", "sui-adoption", tc.fixture)
			got, err := run(workspacePath, fixturePath)
			if err != nil {
				t.Fatalf("run() error = %v", err)
			}

			if got.FixtureValid != tc.wantValid {
				t.Fatalf("FixtureValid = %v, want %v; diagnostics=%v", got.FixtureValid, tc.wantValid, got.Diagnostics)
			}
			if got.InstanceCount != tc.wantInstances {
				t.Fatalf("InstanceCount = %d, want %d", got.InstanceCount, tc.wantInstances)
			}
			if got.SourceRepository != sourceRepository {
				t.Fatalf("SourceRepository = %q, want %q", got.SourceRepository, sourceRepository)
			}
			if got.SourceRevision != sourceRevision {
				t.Fatalf("SourceRevision = %q, want %q", got.SourceRevision, sourceRevision)
			}
			if got.SourceContractBlob != sourceContractBlob ||
				got.SourceHandoffBlob != sourceHandoffBlob ||
				got.SourceContractTestBlob != sourceContractTestBlob ||
				got.SourceHandoffTestBlob != sourceHandoffTestBlob {
				t.Fatalf("source blob provenance mismatch: %+v", got)
			}
			if got.Mapping.DataDefinition != "sui.island" {
				t.Fatalf("DataDefinition = %q", got.Mapping.DataDefinition)
			}
			if got.Mapping.ParentRelation != "sui.island.parent" {
				t.Fatalf("ParentRelation = %q", got.Mapping.ParentRelation)
			}
			if got.Mapping.RelationTarget != "sui.island" {
				t.Fatalf("RelationTarget = %q", got.Mapping.RelationTarget)
			}
			if got.Mapping.RelationCardinal != "one" {
				t.Fatalf("RelationCardinality = %q", got.Mapping.RelationCardinal)
			}
			if got.Mapping.ParentPresenceSemantics != "unresolved" {
				t.Fatalf("ParentPresenceSemantics = %q, want unresolved", got.Mapping.ParentPresenceSemantics)
			}
			if got.Mapping.Lossless {
				t.Fatal("Stage 0 mapping must not claim lossless parent semantics while optionality is unrepresented")
			}
			if got.WriteAttempted {
				t.Fatal("Stage 0 probe must remain read-only")
			}
			if got.CanonicalizedData {
				t.Fatal("Stage 0 probe must not promote SUI instances into TEI Canonical data")
			}
			if tc.wantDiagnostic != "" && !hasDiagnostic(got.Diagnostics, tc.wantDiagnostic) {
				t.Fatalf("diagnostics = %v, want code %s", got.Diagnostics, tc.wantDiagnostic)
			}
		})
	}
}

func TestStage0ConstraintCoverageDoesNotOverclaimTEISemantics(t *testing.T) {
	got := stage0ConstraintCoverage()
	want := map[string]struct {
		status           string
		probeEnforcement string
	}{
		"parent-single-valued":     {status: "shape-only", probeEnforcement: "tei-definition"},
		"parent-presence":          {status: "unresolved", probeEnforcement: "source-contract"},
		"duplicate-island-id":      {status: "shadow-only", probeEnforcement: "reference-local-shadow"},
		"duplicate-child-reference": {status: "shadow-only", probeEnforcement: "reference-local-shadow"},
		"parent-exists":            {status: "shadow-only", probeEnforcement: "reference-local-shadow"},
		"child-exists":             {status: "shadow-only", probeEnforcement: "reference-local-shadow"},
		"parent-child-consistency": {status: "shadow-only", probeEnforcement: "reference-local-shadow"},
		"acyclic-hierarchy":        {status: "shadow-only", probeEnforcement: "reference-local-shadow"},
	}

	if len(got) != len(want) {
		t.Fatalf("constraint coverage count = %d, want %d: %+v", len(got), len(want), got)
	}

	for _, coverage := range got {
		expected, ok := want[coverage.ID]
		if !ok {
			t.Fatalf("unexpected constraint coverage id %q", coverage.ID)
		}
		if coverage.Status != expected.status {
			t.Fatalf("%s status = %q, want %q", coverage.ID, coverage.Status, expected.status)
		}
		if coverage.ProbeEnforcement != expected.probeEnforcement {
			t.Fatalf("%s probeEnforcement = %q, want %q", coverage.ID, coverage.ProbeEnforcement, expected.probeEnforcement)
		}
		if coverage.ID != "parent-single-valued" && coverage.TEIRepresentation != "none" {
			t.Fatalf("%s TEIRepresentation = %q; Stage 0 must not invent native coverage", coverage.ID, coverage.TEIRepresentation)
		}
	}
}

func TestShadowValidatorDoesNotInventIslandIDRequirement(t *testing.T) {
	doc := hierarchyDocument{
		SchemaVersion: "2.0.0",
		Islands: []hierIsland{
			{ID: "", ParentIslandID: nil, ChildIslandIDs: []string{}},
		},
	}

	if diagnostics := validateHierarchy(doc); len(diagnostics) != 0 {
		t.Fatalf("validateHierarchy() diagnostics = %v; source IslandHierarchyContractV1 does not define an ID-required constraint", diagnostics)
	}
}

func TestReadHierarchyFixtureAllowsWhitespaceSchemaVersionLikeSourceContract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.json")
	if err := os.WriteFile(path, []byte(`{
  "schemaVersion": "   ",
  "islands": []
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := readHierarchyFixture(path); err != nil {
		t.Fatalf("readHierarchyFixture() error = %v; source contract rejects only an empty schemaVersion", err)
	}
}

func TestReadHierarchyFixtureRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.json")
	if err := os.WriteFile(path, []byte(`{
  "schemaVersion": "2.0.0",
  "islands": [],
  "unexpected": true
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := readHierarchyFixture(path)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("readHierarchyFixture() error = %v, want unknown field failure", err)
	}
}

func TestReportRemainsSerializable(t *testing.T) {
	root := repositoryRoot(t)
	got, err := run(
		filepath.Join(root, "reference", "sui-adoption", "workspace"),
		filepath.Join(root, "reference", "sui-adoption", "M2-three-level.valid.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(got); err != nil {
		t.Fatalf("report must remain JSON serializable: %v", err)
	}
}

func hasDiagnostic(diagnostics []diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
