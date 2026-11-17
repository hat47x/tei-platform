package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeModule creates a synthetic nested module. The fixtures are deliberately
// tiny and contain nothing about TEI, so each test shows the detection rather
// than restating a product decision.
func writeModule(t *testing.T, root, rel, goMod, source string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatal(err)
	}
	if source != "" {
		if err := os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func writeScope(t *testing.T, root, body string) string {
	t.Helper()
	path := filepath.Join(root, "scope.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const passingSource = `package sample
import "testing"
func TestPasses(t *testing.T) {}
`

const failingSource = `package sample
import "testing"
func TestFails(t *testing.T) { t.Fatal("deliberate failure") }
`

// requireGo skips when the toolchain is absent rather than reporting success. A
// skip is recorded as not verified, not as passing.
func requireGo(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("(SKIP is not a success) go is unavailable: %v", err)
	}
}

// The parent module passing says nothing about a nested one. Both must be
// verified, and the child is named in the output.
func TestParentAndChildModuleBothSucceed(t *testing.T) {
	requireGo(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/parent\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeModule(t, root, "child", "module example.invalid/child\n\ngo 1.23\n", passingSource)
	scopePath := writeScope(t, root, `{"modules":[{"path":"child","verify":true,"reason":"this fixture module is the subject of the test"}]}`)

	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, false, &stdout, &stderr); code != 0 {
		t.Fatalf("both modules pass but the check failed: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok    child [default] go test") {
		t.Fatalf("the child module was not verified: %q", stdout.String())
	}
}

// The repository root passing says nothing about a failing child. Without this
// the child's failure would leave the root check green and unnoticed.
func TestChildFailureIsDetectedWhileParentSucceeds(t *testing.T) {
	requireGo(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/parent\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeModule(t, root, "child", "module example.invalid/child\n\ngo 1.23\n", failingSource)
	scopePath := writeScope(t, root, `{"modules":[{"path":"child","verify":true,"reason":"this fixture module is the subject of the test"}]}`)

	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, false, &stdout, &stderr); code == 0 {
		t.Fatalf("a failing child module was accepted: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "child [default] go test") {
		t.Fatalf("the failure did not name the child module: %q", stderr.String())
	}
}

// A newly added nested module must be declared. This is the silent-drop case:
// before the declaration comparison the module simply never ran.
func TestNewNestedModuleCannotLeaveScopeSilently(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "plugins/old", "module example.invalid/old\n\ngo 1.23\n", passingSource)
	writeModule(t, root, "plugins/new-plugin", "module example.invalid/new-plugin\n\ngo 1.23\n", passingSource)
	scopePath := writeScope(t, root, `{"modules":[{"path":"plugins/old","verify":true,"reason":"this fixture module is the subject of the test"}]}`)

	// Declaration-only mode keeps this independent of a toolchain; the claim here
	// is the layout comparison.
	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, true, &stdout, &stderr); code == 0 {
		t.Fatalf("an undeclared nested module was accepted: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "plugins/new-plugin") {
		t.Fatalf("the undeclared module was not named: %q", stderr.String())
	}
}

// Scanner reach: the enumeration itself must find the module, otherwise the
// declaration check passes because nothing is compared.
func TestEnumerationFindsNestedModulesAndSkipsIgnoredTrees(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "plugins/a", "module example.invalid/a\n\ngo 1.23\n", "")
	writeModule(t, root, "cmd/b", "module example.invalid/b\n\ngo 1.23\n", "")
	writeModule(t, root, "vendor/c", "module example.invalid/c\n\ngo 1.23\n", "")
	writeModule(t, root, "node_modules/d", "module example.invalid/d\n\ngo 1.23\n", "")
	writeModule(t, root, "testdata/e", "module example.invalid/e\n\ngo 1.23\n", "")
	writeModule(t, root, ".hidden/f", "module example.invalid/f\n\ngo 1.23\n", "")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/root\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := enumerateModules(root)
	if err != nil {
		t.Fatal(err)
	}
	want := "cmd/b,plugins/a"
	if strings.Join(got, ",") != want {
		t.Fatalf("nested modules = %v, want %s", got, want)
	}
}

// An explicitly excluded module is allowed, but only with a reason, and the
// reason is printed so the exclusion is readable instead of implied. Another
// module stays verified, so the exclusion is a real choice rather than a
// wholesale opt-out.
func TestExcludedModuleRequiresAndReportsAReason(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "experimental", "module example.invalid/experimental\n\ngo 1.23\n", passingSource)
	writeModule(t, root, "core-pkg", "module example.invalid/core-pkg\n\ngo 1.23\n", passingSource)
	const verified = `{"path":"core-pkg","verify":true,"reason":"the fixture module that stays verified"}`

	unexplained := writeScope(t, root, `{"modules":[{"path":"experimental","verify":false},`+verified+`]}`)
	var stdout, stderr bytes.Buffer
	if code := run(root, unexplained, true, &stdout, &stderr); code == 0 {
		t.Fatalf("an exclusion without a reason was accepted: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "gives no usable reason") {
		t.Fatalf("the missing reason was not reported: %q", stderr.String())
	}

	explained := writeScope(t, root,
		`{"modules":[{"path":"experimental","verify":false,"reason":"needs a live database this runner does not provide"},`+verified+`]}`)
	stdout.Reset()
	stderr.Reset()
	if code := run(root, explained, true, &stdout, &stderr); code != 0 {
		t.Fatalf("an explained exclusion was rejected: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "excluded experimental: needs a live database") {
		t.Fatalf("the exclusion reason was not printed: %q", stdout.String())
	}
}

// Marking every module unverified would leave the check reporting success while
// running nothing, which is the failure this tool exists to prevent.
func TestMarkingEveryModuleUnverifiedIsRejected(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "a", "module example.invalid/a\n\ngo 1.23\n", passingSource)
	writeModule(t, root, "b", "module example.invalid/b\n\ngo 1.23\n", passingSource)
	scopePath := writeScope(t, root,
		`{"modules":[
		  {"path":"a","verify":false,"reason":"a placeholder that satisfies the length rule"},
		  {"path":"b","verify":false,"reason":"a placeholder that satisfies the length rule"}]}`)

	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, true, &stdout, &stderr); code == 0 {
		t.Fatalf("a scope that verifies nothing was accepted: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "no nested module is verified") {
		t.Fatalf("the empty verification set was not reported: %q", stderr.String())
	}
}

// A placeholder reason does not explain anything, so it is rejected. Otherwise a
// one-character exclusion reason would read as an explicit decision.
func TestPlaceholderReasonIsRejected(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "a", "module example.invalid/a\n\ngo 1.23\n", passingSource)
	writeModule(t, root, "b", "module example.invalid/b\n\ngo 1.23\n", passingSource)
	scopePath := writeScope(t, root,
		`{"modules":[
		  {"path":"a","verify":true,"reason":"x"},
		  {"path":"b","verify":true,"reason":"a real reason for verifying this module"}]}`)

	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, true, &stdout, &stderr); code == 0 {
		t.Fatalf("a placeholder reason was accepted: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "no usable reason") {
		t.Fatalf("the placeholder reason was not reported: %q", stderr.String())
	}
}

// A declared module that no longer exists keeps a claim about a layout that is
// gone, so it is reported rather than ignored.
func TestDeclaredModuleThatNoLongerExistsIsReported(t *testing.T) {
	root := t.TempDir()
	scopePath := writeScope(t, root, `{"modules":[{"path":"gone","verify":true,"reason":"this fixture module is the subject of the test"}]}`)
	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, true, &stdout, &stderr); code == 0 {
		t.Fatalf("a stale declaration was accepted: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "is not a nested Go module") {
		t.Fatalf("the stale declaration was not reported: %q", stderr.String())
	}
}

// dependsOnModule reports whether a go.mod body reaches the given module, in the
// inline and the parenthesised block form of require and replace.
func dependsOnModule(body []byte, module string) bool {
	inBlock := false
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		if trimmed == "" {
			continue
		}
		if inBlock {
			if trimmed == ")" {
				inBlock = false
				continue
			}
			if fields := strings.Fields(trimmed); len(fields) > 0 && fields[0] == module {
				return true
			}
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		if fields[0] != "require" && fields[0] != "replace" {
			continue
		}
		if fields[1] == "(" {
			inBlock = true
			continue
		}
		if fields[1] == module {
			return true
		}
	}
	return false
}

// A plugin module that declares no dependency on the product module is still
// verified. The assertion is on the real repository: the module is declared for
// verification, and its go.mod does not reach the product module. That is the
// property, not a restatement of a file the test itself wrote.
func TestPluginModuleIsVerifiedWithoutBeingLinkedToTheProduct(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	declared, err := loadScope(filepath.Join(root, filepath.FromSlash(defaultScopePath)))
	if err != nil {
		t.Fatal(err)
	}
	const pluginModule = "tools/reference-plugin-registry"
	found := false
	for _, e := range declared.Modules {
		if e.Path != pluginModule {
			continue
		}
		found = true
		if !e.Verify {
			t.Fatalf("%s is not verified: a plugin must be checkable without being linked into the product", pluginModule)
		}
	}
	if !found {
		t.Fatalf("%s is not declared; the unlinked-plugin case is no longer covered", pluginModule)
	}

	productModule, err := readModulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(pluginModule), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if dependsOnModule(raw, productModule) {
		t.Fatalf("%s now depends on %s, so verifying it no longer shows a plugin is checkable unlinked",
			pluginModule, productModule)
	}
}

// readModulePath mirrors the repository's other checks: the product module path
// is read from go.mod so a rename does not silently weaken the assertion above.
func readModulePath(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", os.ErrNotExist
}

// The health script is the only place CI reaches this check on a pull request.
// Deleting the step would leave every test here passing while the check silently
// stopped running, so the wiring itself is pinned -- including that --fast keeps
// the declaration comparison instead of skipping the step entirely.
func TestHealthScriptRunsTheModuleScopeCheck(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "tools", "repository-health-check.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(raw)
	for _, want := range []string{
		`[[ "$fast" == true ]]`,
		"nested Go modules are declared\" \\\n    go run ./tools/repository-module-scope -check-only",
		"nested Go modules are declared and verified\" \\\n    go run ./tools/repository-module-scope",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("repository-health-check.sh no longer runs the module scope check as expected; missing %q", want)
		}
	}
}

// A reason counts characters, not bytes, so a short non-ASCII placeholder does
// not satisfy the rule that a reason must explain something.
func TestReasonLengthCountsCharactersNotBytes(t *testing.T) {
	if usableReason("x") {
		t.Fatal("a one-character reason was accepted")
	}
	if usableReason("検証しない") {
		t.Fatal("a seven-character Japanese placeholder was accepted as a reason")
	}
	if !usableReason("この module は実DBを要するため、この runner では検証しない") {
		t.Fatal("a real Japanese reason was rejected")
	}
}

// A declared dependency on the product module must be noticed in both the inline
// and the parenthesised block form, otherwise the unlinked-plugin claim can rot
// silently.
func TestProductModuleDependencyIsDetectedInBothForms(t *testing.T) {
	const product = "example.invalid/product"
	for _, tc := range []struct {
		name, body string
		linked     bool
	}{
		{"inline require", "module p\n\nrequire " + product + " v0.0.0\n", true},
		{"block require", "module p\n\nrequire (\n\t" + product + " v0.0.0\n)\n", true},
		{"inline replace", "module p\n\nreplace " + product + " => ../..\n", true},
		{"block replace", "module p\n\nreplace (\n\t" + product + " => ../..\n)\n", true},
		{"unrelated block", "module p\n\nrequire (\n\texample.invalid/other v1.0.0\n)\n", false},
		{"no directives", "module p\n\ngo 1.23\n", false},
	} {
		if got := dependsOnModule([]byte(tc.body), product); got != tc.linked {
			t.Errorf("%s: dependsOnModule = %v, want %v", tc.name, got, tc.linked)
		}
	}
}

// At least one module in the real scope must be verified. Otherwise an edit could
// mark every module unverified and the check would still report success.
func TestRepositoryScopeVerifiesAtLeastOneModule(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	declared, err := loadScope(filepath.Join(root, filepath.FromSlash(defaultScopePath)))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range declared.Modules {
		if e.Verify {
			return
		}
	}
	t.Fatal("every module in the repository scope is unverified; the check would report success while running nothing")
}

// A declared known failure that starts passing is stale, so the check fails.
// Otherwise the exclusion would outlive its cause and hide a new problem.
func TestKnownFailureThatNowPassesIsReported(t *testing.T) {
	requireGo(t)
	root := t.TempDir()
	writeModule(t, root, "child", "module example.invalid/child\n\ngo 1.23\n", passingSource)
	scopePath := writeScope(t, root,
		`{"modules":[{"path":"child","verify":true,"reason":"this fixture module is the subject of the test",
		  "knownFailures":[{"test":"TestPasses","reason":"the fixture is expected to keep failing here"}]}]}`)

	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, false, &stdout, &stderr); code == 0 {
		t.Fatalf("a stale known failure was accepted: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "did not fail") {
		t.Fatalf("the stale known failure was not reported: %q", stderr.String())
	}
}

// The declared known failure is skipped and printed while the rest of the
// module still verifies.
func TestKnownFailureIsSkippedAndPrinted(t *testing.T) {
	requireGo(t)
	root := t.TempDir()
	writeModule(t, root, "child", "module example.invalid/child\n\ngo 1.23\n", failingSource)
	scopePath := writeScope(t, root,
		`{"modules":[{"path":"child","verify":true,"reason":"this fixture module is the subject of the test",
		  "knownFailures":[{"test":"TestFails","reason":"it fails on the base for an unrelated cause"}]}]}`)

	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, false, &stdout, &stderr); code != 0 {
		t.Fatalf("a declared known failure blocked the module: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "known-failure child [default] TestFails") {
		t.Fatalf("the known failure was not printed: %q", stdout.String())
	}
}

// A skip is not a failure. If a declared known failure stops running at all, the
// entry describes something that no longer happens and must be removed.
func TestKnownFailureThatIsSkippedIsNotAFailure(t *testing.T) {
	requireGo(t)
	root := t.TempDir()
	skipping := `package sample
import "testing"
func TestSkips(t *testing.T) { t.Skip("not applicable here") }
`
	writeModule(t, root, "child", "module example.invalid/child\n\ngo 1.23\n", skipping)
	scopePath := writeScope(t, root,
		`{"modules":[{"path":"child","verify":true,"reason":"this fixture module is the subject of the test",
		  "knownFailures":[{"test":"TestSkips","reason":"the fixture is expected to keep failing here"}]}]}`)

	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, false, &stdout, &stderr); code == 0 {
		t.Fatalf("a skipped test satisfied a declared known failure: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "did not fail") {
		t.Fatalf("the skip was not reported: %q", stderr.String())
	}
}

// Every entry states why it is verified, so a declaration cannot be added
// without a reason even when nothing is excluded.
func TestVerifiedModuleWithoutAReasonIsReported(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "child", "module example.invalid/child\n\ngo 1.23\n", passingSource)
	scopePath := writeScope(t, root, `{"modules":[{"path":"child","verify":true}]}`)
	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, true, &stdout, &stderr); code == 0 {
		t.Fatalf("a declaration without a reason was accepted: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "gives no usable reason") {
		t.Fatalf("the missing reason was not reported: %q", stderr.String())
	}
}

func TestDeclarationComparisonRejectsDuplicates(t *testing.T) {
	problems := checkDeclarations([]string{"a"}, []entry{
		{Path: "a", Verify: true, Reason: "a reason long enough to pass the rule"},
		{Path: "a", Verify: true, Reason: "a reason long enough to pass the rule"},
	})
	if !strings.Contains(strings.Join(problems, "\n"), "declared twice") {
		t.Fatalf("a duplicate declaration was not reported: %v", problems)
	}
}

// A file guarded by the opposite build constraint is a separate build unit. One
// tag set would verify only one side, so every declared variant must be checked.
// This fixture puts a failing test behind each constraint, so a variant that is
// skipped leaves its side's failure unreported.
func TestBothBuildVariantsAreVerified(t *testing.T) {
	requireGo(t)
	root := t.TempDir()
	dir := filepath.Join(root, "child")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.invalid/child\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The neutral test passes in both configurations, so a variant that runs
	// nothing at all is not what makes this fail.
	if err := os.WriteFile(filepath.Join(dir, "neutral_test.go"),
		[]byte("package sample\nimport \"testing\"\nfunc TestNeutral(t *testing.T) {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "server_test.go"),
		[]byte("//go:build server\n\npackage sample\nimport \"testing\"\nfunc TestServerFails(t *testing.T) { t.Fatal(\"server variant broken\") }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "desktop_test.go"),
		[]byte("//go:build !server\n\npackage sample\nimport \"testing\"\nfunc TestDesktopFails(t *testing.T) { t.Fatal(\"desktop variant broken\") }\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Only the desktop variant is declared. The server-only failure must stay
	// invisible here, which is exactly the defect this fixture exposes.
	onlyDesktop := writeScope(t, root,
		`{"modules":[{"path":"child","verify":true,"reason":"a reason long enough to pass the rule",
		  "variants":[{"name":"desktop"}]}]}`)
	var stdout, stderr bytes.Buffer
	if code := run(root, onlyDesktop, false, &stdout, &stderr); code == 0 {
		t.Fatalf("an undeclared variant escaped verification: %s%s", stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), "TestServerFails") {
		t.Fatalf("the undeclared variant was reported, so this fixture does not show the gap: %q", stderr.String())
	}

	// Declaring both variants surfaces each side's failure.
	both := writeScope(t, root,
		`{"modules":[{"path":"child","verify":true,"reason":"a reason long enough to pass the rule",
		  "variants":[{"name":"server","tags":"server"},{"name":"desktop"}]}]}`)
	stdout.Reset()
	stderr.Reset()
	if code := run(root, both, false, &stdout, &stderr); code == 0 {
		t.Fatalf("failing variants were accepted: %s%s", stdout.String(), stderr.String())
	}
	for _, want := range []string{"TestServerFails", "TestDesktopFails"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("%s was not reported when both variants are declared: %q", want, stderr.String())
		}
	}
	// Each result names the configuration it belongs to, so a failure can be
	// traced to one side of the constraint rather than to the module as a whole.
	for _, want := range []string{"child [server] go vet", "child [desktop] go vet"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("%q is missing from the result: %q", want, stdout.String())
		}
	}
	for _, want := range []string{"child [server] go test failed", "child [desktop] go test failed"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("%q is missing from the failure: %q", want, stderr.String())
		}
	}
}

// A variant without a name, or a repeated name, hides which configuration was
// actually checked.
func TestVariantNamesAreRequiredAndUnique(t *testing.T) {
	for _, tc := range []struct {
		name     string
		variants []variant
		want     string
	}{
		{"unnamed", []variant{{Tags: "server"}}, "without a name"},
		{"duplicate", []variant{{Name: "a"}, {Name: "a"}}, "twice"},
	} {
		problems := checkDeclarations([]string{"m"}, []entry{{
			Path:     "m",
			Verify:   true,
			Reason:   "a reason long enough to pass the rule",
			Variants: tc.variants,
		}})
		if !strings.Contains(strings.Join(problems, "\n"), tc.want) {
			t.Errorf("%s: not reported: %v", tc.name, problems)
		}
	}
}

// An entry with no variants is verified once without build tags, so the common
// case is not made unusable by the variant requirement.
func TestEntryWithoutVariantsUsesOneUnconstrainedConfiguration(t *testing.T) {
	vs := entry{Path: "m"}.variants()
	if len(vs) != 1 || vs[0].Tags != "" || vs[0].Name != defaultVariantName {
		t.Fatalf("default variants = %+v, want one unconstrained %q", vs, defaultVariantName)
	}
}

// The repository's own declaration must cover both sides of the studio build
// constraint; verifying one leaves the other host file untested.
func TestRepositoryScopeVerifiesBothStudioVariants(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	declared, err := loadScope(filepath.Join(root, filepath.FromSlash(defaultScopePath)))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range declared.Modules {
		if e.Path != "cmd/tei-studio" {
			continue
		}
		tagged, untagged := false, false
		for _, v := range e.variants() {
			if strings.TrimSpace(v.Tags) == "" {
				untagged = true
				// The desktop host's UI toolkit only builds on Windows. An
				// untagged variant that targets the runner's own platform
				// cannot build it at all, so the server side would be the only
				// side verified.
				if strings.TrimSpace(v.OS) == "" {
					t.Fatalf("the untagged studio variant targets the runner's own platform, where the desktop host cannot build (%+v)", v)
				}
			} else {
				tagged = true
			}
		}
		if !tagged || !untagged {
			t.Fatalf("cmd/tei-studio declares one build configuration; window_desktop.go and window_server.go are guarded by opposite constraints (%+v)", e.Variants)
		}
		return
	}
	t.Fatal("cmd/tei-studio is not declared")
}

// A variant for another platform can be type-checked but not executed. It must
// be recorded as not run rather than reported as a pass, because a test binary
// built for another OS does not run here.
func TestOtherPlatformTestsAreNotReportedAsRun(t *testing.T) {
	foreign := "linux"
	if runtime.GOOS == "linux" {
		foreign = "windows"
	}
	if testsRunnable(variant{Name: "foreign", OS: foreign}) {
		t.Fatalf("a %s test binary was reported as runnable on %s", foreign, runtime.GOOS)
	}
	if !testsRunnable(variant{Name: "native"}) {
		t.Fatal("a variant without a target platform was treated as unrunnable")
	}
	if !testsRunnable(variant{Name: "same", OS: runtime.GOOS}) {
		t.Fatalf("a variant targeting the host %s was treated as unrunnable", runtime.GOOS)
	}
}

// Two variants with one tag set verify the same build twice while the second
// name claims coverage it never asked for. This is the silent drop the file
// exists to prevent, so it is an error rather than a harmless duplicate.
func TestDuplicateBuildTagsAcrossVariantsAreRejected(t *testing.T) {
	problems := checkDeclarations([]string{"m"}, []entry{{
		Path:   "m",
		Verify: true,
		Reason: "a reason long enough to pass the rule",
		Variants: []variant{
			{Name: "server", Tags: "server"},
			{Name: "desktop", Tags: "server"},
		},
	}})
	joined := strings.Join(problems, "\n")
	if !strings.Contains(joined, "same build tags") {
		t.Fatalf("a duplicated tag set was accepted: %v", problems)
	}
}

// The same tag set spelled differently must still be recognised as a duplicate.
func TestEquivalentTagSpellingsCountAsDuplicates(t *testing.T) {
	if normalizedTags("server,studio_generated") != normalizedTags("studio_generated server") {
		t.Fatal("two spellings of one tag set compared unequal")
	}
	problems := checkDeclarations([]string{"m"}, []entry{{
		Path:   "m",
		Verify: true,
		Reason: "a reason long enough to pass the rule",
		Variants: []variant{
			{Name: "a", Tags: "server,studio_generated"},
			{Name: "b", Tags: "studio_generated server"},
		},
	}})
	if !strings.Contains(strings.Join(problems, "\n"), "same build tags") {
		t.Fatalf("equivalent tag spellings were accepted: %v", problems)
	}
}

// A misspelled key used to be dropped silently by encoding/json, which downgraded
// the entry to one untagged configuration and switched off the coverage the file
// declares. The declaration must be refused instead.
func TestUnknownKeysInDeclarationAreRejected(t *testing.T) {
	for _, body := range []string{
		// A leftover top-level tags field from before variants existed.
		`{"modules":[{"path":"m","verify":true,"reason":"a reason long enough to pass the rule","tags":"server"}]}`,
		// A misspelled "variants".
		`{"modules":[{"path":"m","verify":true,"reason":"a reason long enough to pass the rule","variantz":[{"name":"server"}]}]}`,
		// A misspelled key inside a variant.
		`{"modules":[{"path":"m","verify":true,"reason":"a reason long enough to pass the rule","variants":[{"name":"server","tag":"server"}]}]}`,
	} {
		root := t.TempDir()
		scopePath := writeScope(t, root, body)
		var stdout, stderr bytes.Buffer
		if code := run(root, scopePath, true, &stdout, &stderr); code == 0 {
			t.Errorf("an unknown key was accepted: %s%s", stdout.String(), stderr.String())
		}
	}
}

// A known failure behind one build constraint does not exist in the others, so
// an entry with variants must say which one it belongs to.
func TestKnownFailureRequiresAVariantWhenVariantsAreDeclared(t *testing.T) {
	problems := checkDeclarations([]string{"m"}, []entry{{
		Path:          "m",
		Verify:        true,
		Reason:        "a reason long enough to pass the rule",
		Variants:      []variant{{Name: "server"}, {Name: "desktop"}},
		KnownFailures: []knownFailure{{Test: "TestX", Reason: "a reason long enough to pass the rule"}},
	}})
	if !strings.Contains(strings.Join(problems, "\n"), "without a variant") {
		t.Fatalf("an unscoped known failure was accepted: %v", problems)
	}
}

// A known failure that names a variant that does not exist would never be
// checked, so it is reported.
func TestKnownFailureForUnknownVariantIsRejected(t *testing.T) {
	problems := checkDeclarations([]string{"m"}, []entry{{
		Path:          "m",
		Verify:        true,
		Reason:        "a reason long enough to pass the rule",
		Variants:      []variant{{Name: "server"}},
		KnownFailures: []knownFailure{{Test: "TestX", Variant: "desktop", Reason: "a reason long enough to pass the rule"}},
	}})
	if !strings.Contains(strings.Join(problems, "\n"), "unknown variant") {
		t.Fatalf("a known failure for an unknown variant was accepted: %v", problems)
	}
}

// A known failure scoped to one variant must not be confirmed under the other,
// where the test does not exist. This is the case that made the feature unusable
// once variants were declared.
func TestScopedKnownFailureIsNotConfirmedUnderAnotherVariant(t *testing.T) {
	requireGo(t)
	root := t.TempDir()
	dir := filepath.Join(root, "child")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.invalid/child\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The failure exists only under the server constraint.
	if err := os.WriteFile(filepath.Join(dir, "server_test.go"),
		[]byte("//go:build server\n\npackage sample\nimport \"testing\"\nfunc TestServerOnlyFails(t *testing.T) { t.Fatal(\"broken under server\") }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "neutral_test.go"),
		[]byte("package sample\nimport \"testing\"\nfunc TestNeutral(t *testing.T) {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	scopePath := writeScope(t, root,
		`{"modules":[{"path":"child","verify":true,"reason":"a reason long enough to pass the rule",
		  "variants":[{"name":"server","tags":"server"},{"name":"desktop"}],
		  "knownFailures":[{"test":"TestServerOnlyFails","variant":"server","reason":"it fails under the server constraint only"}]}]}`)

	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, false, &stdout, &stderr); code != 0 {
		t.Fatalf("a variant-scoped known failure blocked the module: %s%s", stdout.String(), stderr.String())
	}
	// It is confirmed under the variant that owns it, and nowhere else.
	if !strings.Contains(stdout.String(), "known-failure child [server] TestServerOnlyFails") {
		t.Errorf("the scoped known failure was not reported for its variant: %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "known-failure child [desktop] TestServerOnlyFails") {
		t.Errorf("the known failure was reported under a variant that does not contain it: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "ok    child [desktop] go test") {
		t.Errorf("the desktop variant did not pass: %q", stdout.String())
	}
}

// A repository with no nested module has nothing to verify, and that is stated
// rather than passing with no output that could hide a skipped run.
func TestRepositoryWithoutNestedModulesIsStated(t *testing.T) {
	root := t.TempDir()
	scopePath := writeScope(t, root, `{"modules":[]}`)
	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, true, &stdout, &stderr); code != 0 {
		t.Fatalf("an empty layout was rejected: %s%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "no nested Go module") {
		t.Fatalf("the empty layout was not stated: %q", stdout.String())
	}
}

// A scope that declares modules while the repository has none is a stale claim,
// so it is still an error.
func TestScopeDeclaringModulesThatDoNotExistIsRejected(t *testing.T) {
	root := t.TempDir()
	scopePath := writeScope(t, root,
		`{"modules":[{"path":"gone","verify":true,"reason":"a reason long enough to pass the rule"}]}`)
	var stdout, stderr bytes.Buffer
	if code := run(root, scopePath, true, &stdout, &stderr); code == 0 {
		t.Fatalf("a stale declaration was accepted: %s%s", stdout.String(), stderr.String())
	}
}

// A scope file saved with a byte order mark is still a valid declaration; the
// mark is an encoding detail, not a corrupted file.
func TestScopeWithByteOrderMarkIsReadable(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "child", "module example.invalid/child\n\ngo 1.23\n", passingSource)
	path := filepath.Join(root, "scope.json")
	body := "\xef\xbb\xbf" + `{"modules":[{"path":"child","verify":true,"reason":"a reason long enough to pass the rule"}]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run(root, path, true, &stdout, &stderr); code != 0 {
		t.Fatalf("a scope with a byte order mark was rejected: %s%s", stdout.String(), stderr.String())
	}
}

// The repository's own scope must match its layout. This is the case the whole
// check exists for: a nested module added without a declaration fails here even
// when nothing else in the repository changed.
func TestRepositoryScopeMatchesLayout(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	nested, err := enumerateModules(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(nested) == 0 {
		t.Fatal("no nested module was found; the enumeration or the layout changed")
	}
	declared, err := loadScope(filepath.Join(root, filepath.FromSlash(defaultScopePath)))
	if err != nil {
		t.Fatal(err)
	}
	if problems := checkDeclarations(nested, declared.Modules); len(problems) != 0 {
		t.Fatalf("module scope does not match the repository layout:\n%s", strings.Join(problems, "\n"))
	}
}
