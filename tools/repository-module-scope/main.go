// Command repository-module-scope keeps every nested Go module inside the
// repository's verification.
//
// `go vet ./...` and `go test ./...` started at the repository root stop at a
// nested go.mod. A module placed under plugins/ or cmd/ therefore leaves the
// root checks green without running its own tests, so it can leave the
// repository's verification with no error at all (#4612). This command
// enumerates nested modules, requires each one to be declared in scope.json,
// and verifies the declared ones.
//
// This is a source-level comparison plus an actual toolchain run. It does not
// show that a module's tests are meaningful, that its results are accepted as
// product success, or that CI lists the module in its path filters.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const defaultScopePath = "tools/repository-module-scope/scope.json"

// ignoredDirs are trees Go does not build as part of this repository: installed
// dependencies, vendored copies, and fixtures. Every dot-prefixed directory is
// skipped as well.
var ignoredDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"testdata":     true,
}

type scope struct {
	Modules []entry `json:"modules"`
}

// entry is one nested module's verification mode.
//
// Verify=false is an exclusion and needs a Reason, which is printed so the
// exclusion is visible rather than implied by omission.
type entry struct {
	Path          string            `json:"path"`
	Verify        bool              `json:"verify"`
	Reason        string            `json:"reason"`
	Tags          string            `json:"tags"`
	Env           map[string]string `json:"env"`
	KnownFailures []knownFailure    `json:"knownFailures"`
}

// knownFailure is a test that fails on the base for a reason this change does
// not fix. It is skipped and printed, and it must keep failing.
type knownFailure struct {
	Test   string `json:"test"`
	Reason string `json:"reason"`
}

// enumerateModules returns the repository-relative directory of every go.mod
// below root, sorted. The root module is not returned: the repository's own
// checks already cover it.
func enumerateModules(root string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || ignoredDirs[name] {
				return fs.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		rel, relErr := filepath.Rel(root, filepath.Dir(path))
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		found = append(found, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(found)
	return found, nil
}

func loadScope(path string) (scope, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return scope{}, fmt.Errorf("read module scope: %w", err)
	}
	// An editor may save this file with a byte order mark. Rejecting it as
	// invalid JSON would read like a corrupt declaration rather than an encoding
	// detail, so the mark is dropped first.
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	var parsed scope
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return scope{}, fmt.Errorf("%s: %w", path, err)
	}
	return parsed, nil
}

// minReasonLength rejects a placeholder such as "x". A reason has to say
// something, otherwise an exclusion is only nominally explained. It counts
// characters, not bytes, so a short Japanese placeholder is rejected too.
const minReasonLength = 20

// usableReason reports whether a reason is long enough to explain anything.
func usableReason(reason string) bool {
	return utf8.RuneCountInString(strings.TrimSpace(reason)) >= minReasonLength
}

// checkDeclarations compares the modules that exist with the modules that are
// declared. Both directions are errors: an undeclared module escapes
// verification silently, and a declared module that no longer exists keeps a
// reason for something other than the current layout.
func checkDeclarations(nested []string, declared []entry) []string {
	var problems []string
	present := map[string]bool{}
	verified := 0
	for _, e := range declared {
		if present[e.Path] {
			problems = append(problems, e.Path+" is declared twice in the module scope")
		}
		present[e.Path] = true
		if e.Verify {
			verified++
		}
		if !usableReason(e.Reason) {
			problems = append(problems, fmt.Sprintf(
				"%s gives no usable reason for its verification mode (at least %d characters): %q",
				e.Path, minReasonLength, e.Reason))
		}
		for _, kf := range e.KnownFailures {
			if strings.TrimSpace(kf.Test) == "" || !usableReason(kf.Reason) {
				problems = append(problems, fmt.Sprintf(
					"%s declares a known failure without a test name and a usable reason (at least %d characters)",
					e.Path, minReasonLength))
			}
		}
	}
	paths := map[string]bool{}
	for _, p := range nested {
		paths[p] = true
		if !present[p] {
			problems = append(problems,
				p+" is a nested Go module that the module scope does not declare, so its tests would never run")
		}
	}
	for _, e := range declared {
		if !paths[e.Path] {
			problems = append(problems, e.Path+" is declared in the module scope but is not a nested Go module")
		}
	}
	// Without this an edit could mark every module unverified and the whole
	// check would still report success.
	if len(nested) > 0 && verified == 0 {
		problems = append(problems,
			"no nested module is verified; at least one must be, otherwise the module scope only records exclusions")
	}
	sort.Strings(problems)
	return problems
}

// moduleEnv fixes the Go environment the same way the Gate runners do and adds
// the entry's own overrides.
func moduleEnv(overrides map[string]string) []string {
	merged := map[string]string{}
	for _, kv := range os.Environ() {
		if name, value, ok := strings.Cut(kv, "="); ok {
			merged[name] = value
		}
	}
	merged["GOENV"] = "off"
	merged["GOWORK"] = "off"
	merged["GOFLAGS"] = ""
	for name, value := range overrides {
		merged[name] = value
	}
	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	sort.Strings(names)
	env := make([]string, 0, len(names))
	for _, name := range names {
		env = append(env, name+"="+merged[name])
	}
	return env
}

func goExecutable() (string, error) {
	if path, err := exec.LookPath("go"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("go is not on PATH")
}

// goRun runs one go subcommand in the module directory and reports its combined
// output and whether the command failed. Callers decide what a failure means,
// because a non-zero exit is the expected result for a declared known failure.
func goRun(goBin, dir string, env []string, args ...string) (string, bool) {
	cmd := exec.Command(goBin, args...)
	cmd.Dir, cmd.Env = dir, env
	out, err := cmd.CombinedOutput()
	return string(out), err != nil
}

func tagArgs(e entry) []string {
	if strings.TrimSpace(e.Tags) == "" {
		return nil
	}
	return []string{"-tags", e.Tags}
}

func skipPattern(e entry) string {
	names := make([]string, 0, len(e.KnownFailures))
	for _, kf := range e.KnownFailures {
		names = append(names, kf.Test)
	}
	return "^(" + strings.Join(names, "|") + ")$"
}

// confirmKnownFailures shows each declared known failure still fails. A skip is
// not a failure: `go test` exits 0 for a skipped test, and a test removed by a
// build tag reports neither. The `--- FAIL:` line is therefore required per
// named test, so the entry cannot be satisfied by the test's absence.
func confirmKnownFailures(goBin, dir string, e entry, env []string, stdout io.Writer) []string {
	var problems []string
	for _, kf := range e.KnownFailures {
		args := append([]string{"test", "-count=1", "-v"}, tagArgs(e)...)
		args = append(args, "-run", "^"+kf.Test+"$", "./...")
		out, failed := goRun(goBin, dir, env, args...)
		switch {
		case failed && strings.Contains(out, "--- FAIL: "+kf.Test):
			fmt.Fprintf(stdout, "known-failure %s %s: %s\n", e.Path, kf.Test, kf.Reason)
		case !failed:
			problems = append(problems, fmt.Sprintf(
				"%s %s did not fail: a test that passes, skips or no longer exists is not the declared known failure; remove the entry",
				e.Path, kf.Test))
		default:
			// A build failure or a TestMain exit also produces a non-zero status
			// without a --- FAIL: line for this test, so the message names both
			// possibilities instead of asserting the declared cause changed.
			problems = append(problems, fmt.Sprintf(
				"%s %s did not report a failure of its own; the package may fail to build, or the test may exit another way:\n%s",
				e.Path, kf.Test, out))
		}
	}
	return problems
}

// verifyModule runs vet and the module's tests, then confirms each declared
// known failure still fails. It returns one message per problem.
func verifyModule(root string, e entry, stdout io.Writer) []string {
	dir := filepath.Join(root, filepath.FromSlash(e.Path))
	goBin, err := goExecutable()
	if err != nil {
		return []string{err.Error()}
	}
	env := moduleEnv(e.Env)
	var problems []string

	vetArgs := append([]string{"vet"}, tagArgs(e)...)
	vetArgs = append(vetArgs, "./...")
	if out, failed := goRun(goBin, dir, env, vetArgs...); failed {
		problems = append(problems, fmt.Sprintf("%s go vet failed:\n%s", e.Path, out))
	} else {
		fmt.Fprintf(stdout, "ok    %s go vet\n", e.Path)
	}

	testArgs := append([]string{"test", "-count=1"}, tagArgs(e)...)
	if len(e.KnownFailures) > 0 {
		testArgs = append(testArgs, "-skip", skipPattern(e))
	}
	testArgs = append(testArgs, "./...")
	if out, failed := goRun(goBin, dir, env, testArgs...); failed {
		problems = append(problems, fmt.Sprintf("%s go test failed:\n%s", e.Path, out))
	} else {
		fmt.Fprintf(stdout, "ok    %s go test\n", e.Path)
	}

	return append(problems, confirmKnownFailures(goBin, dir, e, env, stdout)...)
}

func run(root, scopePath string, checkOnly bool, stdout, stderr io.Writer) int {
	nested, err := enumerateModules(root)
	if err != nil {
		fmt.Fprintf(stderr, "FAIL %v\n", err)
		return 1
	}
	declared, err := loadScope(scopePath)
	if err != nil {
		fmt.Fprintf(stderr, "FAIL %v\n", err)
		return 1
	}
	problems := checkDeclarations(nested, declared.Modules)
	fmt.Fprintf(stdout, "nested-modules %d\n", len(nested))
	if len(nested) == 0 {
		// A repository with no nested module has nothing to verify; that is a
		// real state, not a failure, and it is stated so an empty output does
		// not look like a run that skipped its work.
		fmt.Fprintln(stdout, "no nested Go module in this repository; nothing to verify")
	}
	for _, p := range nested {
		fmt.Fprintf(stdout, "module %s\n", p)
	}
	for _, e := range declared.Modules {
		if e.Verify {
			continue
		}
		fmt.Fprintf(stdout, "excluded %s: %s\n", e.Path, e.Reason)
	}
	if len(problems) == 0 && !checkOnly {
		for _, e := range declared.Modules {
			if !e.Verify {
				continue
			}
			problems = append(problems, verifyModule(root, e, stdout)...)
		}
	}
	for _, p := range problems {
		fmt.Fprintf(stderr, "FAIL %s\n", p)
	}
	return len(problems)
}

func main() {
	root := flag.String("root", ".", "repository root")
	scopePath := flag.String("scope", defaultScopePath, "nested module scope declaration")
	checkOnly := flag.Bool("check-only", false, "compare declarations with the layout without running the modules")
	flag.Parse()
	if code := run(*root, *scopePath, *checkOnly, os.Stdout, os.Stderr); code != 0 {
		os.Exit(1)
	}
}
