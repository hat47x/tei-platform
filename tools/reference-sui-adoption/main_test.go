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
