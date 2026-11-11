package main

import (
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

func matrixPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repositoryRoot(t), "reference", "sui-adoption", "scope-isolation-matrix.v1.json")
}

// The shipped matrix must satisfy its own rules: all 14 requirement paths are
// present, every covered entry names a test, and every not-applicable /
// uncovered entry carries a reason (and an owner for uncovered).
func TestShippedMatrixIsComplete(t *testing.T) {
	got, err := run(matrixPath(t), "")
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !got.RequirementOK {
		t.Fatalf("requirement is not complete: %+v", got.Findings)
	}
	if len(got.Findings) != 0 {
		t.Fatalf("findings = %+v", got.Findings)
	}
	if got.PathsCount != 14 {
		t.Fatalf("pathCount = %d, want 14", got.PathsCount)
	}
	if got.TestRefs == 0 {
		t.Fatal("no test references were collected")
	}
	// The one known gap must stay visible instead of being dropped silently.
	if got.Uncovered != 1 {
		t.Fatalf("uncovered = %d, want 1 (cache is a known gap)", got.Uncovered)
	}
}

// A missing requirement path must fail: silence about a path is not the same as
// "the feature does not exist".
func TestMissingRequirementPathFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "matrix.json")
	// No `cache` entry.
	writeTestMatrix(t, path, strings.Replace(
		`{"schema":"tei.conformance.scope-isolation-matrix/v1","source":{"requirement":"r","requirementPath":"p","requirementText":"t","authority":"a","languageIndependent":true},"paths":[`+
			strings.Join([]string{
				entry("get"), entry("put"), entry("list"), entry("search"), entry("count"),
				entry("export"), entry("share"), entry("import"), entry("mcp"), entry("webhook"),
				entry("job"), entry("audit"), entry("agent"),
			}, ",")+`]}`, "", "", 1))
	got, err := run(path, "")
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if len(got.Findings) == 0 {
		t.Fatal("a missing path was accepted")
	}
	if !strings.Contains(got.Findings[0].Path, "cache") && !strings.Contains(got.Findings[0].Detail, "missing") {
		t.Fatalf("unexpected finding: %+v", got.Findings)
	}
}

// A rename on the SUI side must fail verification rather than leave the matrix
// silently stale.
func TestRenamedTestRefFailsVerification(t *testing.T) {
	dir := t.TempDir()
	sui := filepath.Join(dir, "sui")
	backend := filepath.Join(sui, "03_Implement", "backend", "tests")
	if err := os.MkdirAll(backend, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(backend, "test_matrix_fixture.py"),
		[]byte("def test_present() -> None:\n    pass\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "matrix.json")
	writeTestMatrix(t, path, `{"schema":"tei.conformance.scope-isolation-matrix/v1",`+
		`"source":{"requirement":"r","requirementPath":"p","requirementText":"t","authority":"a","languageIndependent":true},`+
		`"paths":[{"id":"get","route":"GET","status":"covered","tests":[{"repo":"sui-sensemaking",`+
		`"file":"03_Implement/backend/tests/test_matrix_fixture.py","name":"test_renamed_away"}]}]}`)
	got, err := run(path, sui)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if len(got.Findings) == 0 {
		t.Fatal("a renamed test was accepted; the matrix would go stale silently")
	}
}

// A test that is present must verify, so the check is not vacuous.
func TestPresentTestRefVerifies(t *testing.T) {
	dir := t.TempDir()
	sui := filepath.Join(dir, "sui")
	backend := filepath.Join(sui, "03_Implement", "backend", "tests")
	if err := os.MkdirAll(backend, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(backend, "test_matrix_fixture.py"),
		[]byte("def test_present() -> None:\n    pass\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "matrix.json")
	writeTestMatrix(t, path, `{"schema":"tei.conformance.scope-isolation-matrix/v1",`+
		`"source":{"requirement":"r","requirementPath":"p","requirementText":"t","authority":"a","languageIndependent":true},`+
		`"paths":[{"id":"get","route":"GET","status":"covered","tests":[{"repo":"sui-sensemaking",`+
		`"file":"03_Implement/backend/tests/test_matrix_fixture.py","name":"test_present"}]}]}`)
	got, err := run(path, sui)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !got.VerifiedTests {
		t.Fatal("verification did not run")
	}
	if got.VerifiedRefs != 1 {
		t.Fatalf("verifiedTestRefs = %d, want 1", got.VerifiedRefs)
	}
}

// not-applicable must not pretend to have evidence.
func TestNotApplicableMustNotCarryTests(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "matrix.json")
	writeTestMatrix(t, path, `{"schema":"tei.conformance.scope-isolation-matrix/v1",`+
		`"source":{"requirement":"r","requirementPath":"p","requirementText":"t","authority":"a","languageIndependent":true},`+
		`"paths":[{"id":"search","status":"not-applicable","reason":"no route",`+
		`"tests":[{"repo":"sui-sensemaking","file":"x.py","name":"y"}]}]}`)
	got, err := run(path, "")
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if len(got.Findings) == 0 {
		t.Fatal("not-applicable with test references was accepted")
	}
}

// uncovered must name an owner, otherwise it is just a forgotten path.
func TestUncoveredNeedsAnOwner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "matrix.json")
	writeTestMatrix(t, path, `{"schema":"tei.conformance.scope-isolation-matrix/v1",`+
		`"source":{"requirement":"r","requirementPath":"p","requirementText":"t","authority":"a","languageIndependent":true},`+
		`"paths":[{"id":"cache","status":"uncovered","reason":"no test"}]}`)
	got, err := run(path, "")
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if len(got.Findings) == 0 {
		t.Fatal("uncovered without an owner was accepted")
	}
}

// The matrix language-independence flag must be true: this file is meant to be
// readable by a non-Go consumer too.
func TestMatrixMustDeclareLanguageIndependence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "matrix.json")
	writeTestMatrix(t, path, `{"schema":"tei.conformance.scope-isolation-matrix/v1",`+
		`"source":{"requirement":"r","requirementPath":"p","requirementText":"t","authority":"a","languageIndependent":false},`+
		`"paths":[{"id":"get","status":"uncovered","reason":"r","owner":"o"}]}`)
	got, err := run(path, "")
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if len(got.Findings) == 0 {
		t.Fatal("languageIndependent=false was accepted")
	}
}

func writeTestMatrix(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// entry builds a covered path entry with one test reference.
func entry(id string) string {
	return `{"id":"` + id + `","status":"covered","tests":[{"repo":"sui-sensemaking",` +
		`"file":"03_Implement/backend/tests/test_docs_tenant_isolation_surfaces.py",` +
		`"name":"test_list_returns_only_the_active_tenants_documents"}]}`
}
