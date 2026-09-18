package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditAcceptsSameRunArtifacts(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	result, err := audit(finalPath, dir)
	if err != nil {
		t.Fatalf("audit() error=%v", err)
	}
	if !result.ArtifactAuditPassed || result.RepositoryRevision != "R" || len(result.Lanes) != 3 {
		t.Fatalf("result=%#v", result)
	}
}

func TestAuditRejectsModifiedArtifactBytes(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	path := filepath.Join(dir, laneFiles["generated-artifact-reproducibility"])
	if err := os.WriteFile(path, []byte(`{"tampered":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := audit(finalPath, dir); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("err=%v, want SHA-256 failure", err)
	}
}

func TestAuditRejectsMixedRevision(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	path := filepath.Join(dir, laneFiles["deterministic-repository"])
	obj := repositoryChild("R2", false)
	content := mustJSON(t, obj)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	final := readObject(t, finalPath)
	lanes := final["lanes"].([]any)
	for _, raw := range lanes {
		lane := raw.(map[string]any)
		if lane["name"] == "deterministic-repository" {
			lane["evidenceArtifactFingerprint"] = fingerprint(content)
		}
	}
	writeJSON(t, finalPath, final)
	if _, err := audit(finalPath, dir); err == nil || !strings.Contains(err.Error(), "repositoryRevision") {
		t.Fatalf("err=%v, want revision failure", err)
	}
}

func TestAuditRejectsPostgresCleanupFailure(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	path := filepath.Join(dir, laneFiles["real-postgres-ephemeral-fixture"])
	obj := postgresChild("R")
	obj["fixtureCleanupSafety"].(map[string]any)["verified"] = false
	content := mustJSON(t, obj)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	final := readObject(t, finalPath)
	for _, raw := range final["lanes"].([]any) {
		lane := raw.(map[string]any)
		if lane["name"] == "real-postgres-ephemeral-fixture" {
			lane["evidenceArtifactFingerprint"] = fingerprint(content)
		}
	}
	writeJSON(t, finalPath, final)
	if _, err := audit(finalPath, dir); err == nil || !strings.Contains(err.Error(), "verified=false") {
		t.Fatalf("err=%v, want cleanup failure", err)
	}
}

func TestAuditRejectsSymlinkArtifact(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	path := filepath.Join(dir, laneFiles["deterministic-repository"])
	target := filepath.Join(dir, "replacement.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := audit(finalPath, dir); err == nil || !strings.Contains(err.Error(), "non-symlink") {
		t.Fatalf("err=%v, want symlink failure", err)
	}
}

func TestAuditRejectsSensitiveEvidenceField(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	path := filepath.Join(dir, laneFiles["real-postgres-ephemeral-fixture"])
	obj := postgresChild("R")
	obj["databaseName"] = "secret-db-name"
	content := mustJSON(t, obj)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	final := readObject(t, finalPath)
	for _, raw := range final["lanes"].([]any) {
		lane := raw.(map[string]any)
		if lane["name"] == "real-postgres-ephemeral-fixture" {
			lane["evidenceArtifactFingerprint"] = fingerprint(content)
		}
	}
	writeJSON(t, finalPath, final)
	if _, err := audit(finalPath, dir); err == nil || !strings.Contains(err.Error(), "databaseName") {
		t.Fatalf("err=%v, want sensitive field failure", err)
	}
}

func TestAuditRejectsUnexpectedRepositoryCheck(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	path := filepath.Join(dir, laneFiles["deterministic-repository"])
	obj := repositoryChild("R", false)
	obj["checks"] = append(obj["checks"].([]any), map[string]any{"name": "unexpected-check", "passed": true})
	content := mustJSON(t, obj)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	updateFinalLaneFingerprint(t, finalPath, "deterministic-repository", content)
	if _, err := audit(finalPath, dir); err == nil || !strings.Contains(err.Error(), "checks=4 want=3") {
		t.Fatalf("err=%v, want exact check-set failure", err)
	}
}

func TestAuditRejectsRepositoryWorkspaceMismatch(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	path := filepath.Join(dir, laneFiles["deterministic-repository"])
	obj := repositoryChild("R", false)
	obj["workspace"] = "reference/other-workspace"
	content := mustJSON(t, obj)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	updateFinalLaneFingerprint(t, finalPath, "deterministic-repository", content)
	if _, err := audit(finalPath, dir); err == nil || !strings.Contains(err.Error(), "workspace") {
		t.Fatalf("err=%v, want workspace identity failure", err)
	}
}

func TestAuditRejectsRepositoryCommandMismatch(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	path := filepath.Join(dir, laneFiles["deterministic-repository"])
	obj := repositoryChild("R", false)
	checks := obj["checks"].([]any)
	checks[0].(map[string]any)["command"] = []any{"go", "test", "./other"}
	content := mustJSON(t, obj)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	updateFinalLaneFingerprint(t, finalPath, "deterministic-repository", content)
	if _, err := audit(finalPath, dir); err == nil || !strings.Contains(err.Error(), "command") {
		t.Fatalf("err=%v, want command binding failure", err)
	}
}

func TestAuditRejectsGeneratedCommandMismatch(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	path := filepath.Join(dir, laneFiles["generated-artifact-reproducibility"])
	obj := generatedChild("R")
	checks := obj["checks"].([]any)
	checks[1].(map[string]any)["command"] = []any{"go", "run", "./tools/other"}
	content := mustJSON(t, obj)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	updateFinalLaneFingerprint(t, finalPath, "generated-artifact-reproducibility", content)
	if _, err := audit(finalPath, dir); err == nil || !strings.Contains(err.Error(), "command") {
		t.Fatalf("err=%v, want generated command binding failure", err)
	}
}

func TestAuditRejectsPostgresIdentityMismatch(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	path := filepath.Join(dir, laneFiles["real-postgres-ephemeral-fixture"])
	obj := postgresChild("R")
	obj["contractQueryTarget"] = "other_query"
	content := mustJSON(t, obj)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	updateFinalLaneFingerprint(t, finalPath, "real-postgres-ephemeral-fixture", content)
	if _, err := audit(finalPath, dir); err == nil || !strings.Contains(err.Error(), "contractQueryTarget") {
		t.Fatalf("err=%v, want PostgreSQL identity failure", err)
	}
}

func TestAuditRejectsPostgresSemanticAttributeMismatch(t *testing.T) {
	finalPath, dir := writeFixture(t, "R")
	path := filepath.Join(dir, laneFiles["real-postgres-ephemeral-fixture"])
	obj := postgresChild("R")
	obj["semanticAttributes"] = []any{"sales.order_line.order_line_id"}
	content := mustJSON(t, obj)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	updateFinalLaneFingerprint(t, finalPath, "real-postgres-ephemeral-fixture", content)
	if _, err := audit(finalPath, dir); err == nil || !strings.Contains(err.Error(), "semanticAttributes") {
		t.Fatalf("err=%v, want Semantic attribute failure", err)
	}
}

func updateFinalLaneFingerprint(t *testing.T, finalPath, laneName string, content []byte) {
	t.Helper()
	final := readObject(t, finalPath)
	for _, raw := range final["lanes"].([]any) {
		lane := raw.(map[string]any)
		if lane["name"] == laneName {
			lane["evidenceArtifactFingerprint"] = fingerprint(content)
			writeJSON(t, finalPath, final)
			return
		}
	}
	t.Fatalf("lane=%s not found", laneName)
}

func writeFixture(t *testing.T, revision string) (string, string) {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "children")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	children := map[string]map[string]any{
		"deterministic-repository":           repositoryChild(revision, false),
		"generated-artifact-reproducibility": generatedChild(revision),
		"real-postgres-ephemeral-fixture":    postgresChild(revision),
	}
	lanes := make([]any, 0, 3)
	for _, name := range []string{"deterministic-repository", "generated-artifact-reproducibility", "real-postgres-ephemeral-fixture"} {
		content := mustJSON(t, children[name])
		filename := laneFiles[name]
		if err := os.WriteFile(filepath.Join(dir, filename), content, 0o600); err != nil {
			t.Fatal(err)
		}
		lanes = append(lanes, map[string]any{
			"name": name, "expectedNature": name, "observedNature": name,
			"repositoryRevision": revision, "reproducibleRevision": true,
			"artifactFile": filename, "evidenceArtifactFingerprint": fingerprint(content), "passed": true,
		})
	}
	final := map[string]any{
		"evidenceNature":     "gate-b-three-lane-final-convergence",
		"repositoryRevision": revision, "workingTreeClean": true, "reproducibleRevision": true,
		"postRunRepository": postRun(revision), "goVersion": "go version go1.23.2 linux/amd64",
		"goEnvironment":  map[string]any{"externalWorkspaceActive": false, "goFlagsEmpty": true, "goEnvDisabled": true},
		"broadRequested": false, "allLanesSameRevision": true, "childArtifactsRetained": true,
		"lanes": lanes,
	}
	finalPath := filepath.Join(base, "final.json")
	writeJSON(t, finalPath, final)
	return finalPath, dir
}

func postRun(revision string) map[string]any {
	return map[string]any{"checked": true, "repositoryRevision": revision, "revisionStable": true, "workingTreeClean": true}
}

func repositoryChild(revision string, broad bool) map[string]any {
	checks := []any{
		map[string]any{
			"name":    "final-ui-less-scenario",
			"command": []any{"go", "test", "./internal/driver", "-run", "^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$", "-count=1"},
			"passed":  true,
		},
		map[string]any{
			"name":    "external-evidence-intake",
			"command": []any{"go", "test", "./internal/database/externalqueryevidence", "./tools/reference-external-query-evidence", "-count=1"},
			"passed":  true,
		},
		map[string]any{
			"name":    "repository-evidence-source-boundary",
			"command": []any{"go", "test", "./tools/internal/repositoryevidence", "-count=1"},
			"passed":  true,
		},
	}
	if broad {
		checks = append(checks, map[string]any{
			"name":    "repository-wide-go-test",
			"command": []any{"go", "test", "./..."},
			"passed":  true,
		})
	}
	return map[string]any{
		"evidenceNature": "deterministic-repository", "repositoryRevision": revision,
		"workingTreeClean": true, "reproducibleRevision": true, "postRunRepository": postRun(revision),
		"workspace": "reference/workspace", "workspaceValidation": map[string]any{"passed": true, "errors": []any{}},
		"checks": checks, "broadRequested": broad,
	}
}

func generatedChild(revision string) map[string]any {
	return map[string]any{
		"evidenceNature": "generated-artifact-reproducibility", "repositoryRevision": revision,
		"workingTreeClean": true, "reproducibleRevision": true, "postRunRepository": postRun(revision),
		"checks": []any{
			map[string]any{
				"name":    "world-plugin-lock-golden",
				"command": []any{"go", "run", "./tools/reference-golden"},
				"passed":  true,
			},
			map[string]any{
				"name":    "durable-run-golden",
				"command": []any{"go", "run", "./tools/reference-durable-run-golden"},
				"passed":  true,
			},
		},
	}
}

func postgresChild(revision string) map[string]any {
	return map[string]any{
		"evidenceNature": "real-postgres-ephemeral-fixture", "repositoryRevision": revision,
		"workingTreeClean": true, "reproducibleRevision": true, "postRunRepository": postRun(revision),
		"fixtureObject": "public.order_lines", "preexistingFixtureProtected": true, "fixtureOwnedByTool": true,
		"fixtureDatabaseSafety": map[string]any{
			"guardPassed": true, "databaseIdentityVerified": true, "databaseIdentityPersisted": false,
			"nonDefaultDatabaseVerified": true, "userRelationsBeforeFixture": 0, "eventTriggersBeforeFixture": 0,
			"fixtureObjectAbsentBeforeCreate": true,
		},
		"fixtureCleanupSafety": map[string]any{
			"verified": true, "userRelationsAfterCleanup": 0, "eventTriggersAfterCleanup": 0,
			"fixtureObjectAbsentAfterCleanup": true,
		},
		"pattern": "order_entry.order_lines", "relation": "sales.order.lines", "environment": "development",
		"databaseInstance": "sales_core.dev", "executorCapability": "database.query.postgres", "contractQueryTarget": "sales_core/sales_core.order_line/find_by_order",
		"parameterNames": []any{"order_id"}, "parameterValuesPersisted": false, "rows": 2, "zeroRows": 0,
		"executionErrorObserved": true,
		"semanticAttributes": []any{
			"sales.order_line.item_code",
			"sales.order_line.line_number",
			"sales.order_line.order_line_id",
			"sales.order_line.quantity",
			"sales.order_line.unit_price",
		},
		"relationStorageColumn": "order_id", "semanticOrderLineParentAttributeAdded": false, "semanticMappingApplied": true,
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(content, '\n')
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.WriteFile(path, mustJSON(t, value), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readObject(t *testing.T, path string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(content, &obj); err != nil {
		t.Fatal(err)
	}
	return obj
}
