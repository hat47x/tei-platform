package architecturecheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This check pins one narrow claim about .github/workflows: a workflow that
// runs `go test` for a root-module package must also list that package under
// every `paths` filter it triggers on. Otherwise breaking the package does not
// start the job that claims to test it, and the break stays invisible until an
// unrelated change happens to touch a listed path (#5460 evidence:
// plan/evidence/XTEA_LOWCODE_POSTGRES_SUITES_PATHS_2026-11.md).
//
// It does not judge `go test ./...` (a whole-module run), steps whose
// working-directory belongs to a nested module (cmd/tei-studio,
// tools/reference-plugin-registry), or whether the listed tests mean anything.

// workflowPathsAllowed names the workflow/package pairs that the check still
// finds uncovered. Every entry must say why it is not fixed in this change.
// The list is shrink-only: a pair that becomes covered is reported as stale so
// the entry is removed instead of granting the same hole to a renamed package.
var workflowPathsAllowed = map[string]string{
	"rea-external-candidate-contract.yml cmd/tei":                                   "runs ./cmd/tei filtered to TestREACandidatesCLI; paths list cmd/tei/cli_three_element_rea_candidates*.go, the file those tests live in. cmd/tei/** would start the job on every CLI change for two tests whose compile break repository-health already reports",
	"rea-external-candidate-contract.yml internal/operations":                       "runs ./internal/operations filtered to the external-candidate tests; the paths entry internal/operations/external_candidates* names the file that changes. internal/operations/** would start the job on every operations change",
	"rea-external-evidence-intake.yml cmd/tei":                                      "runs ./cmd/tei filtered to TestReaEvidence|TestDiscoveryModes|TestCommandCatalogFlags|TestContractDocument; the paths entry cmd/tei/rea_evidence* names the changed file. cmd/tei/** costs a job on every CLI change",
	"rea-external-evidence-intake.yml internal/operations":                          "runs ./internal/operations with -run TestExternalCandidate; the paths entry internal/operations/external_candidates* names the file that changes",
	"tei-one-revision-capability.yml cmd/tei":                                       "runs ./cmd/tei filtered to TestValidManifestPublication; the paths entry cmd/tei/cli_operations_artifacts*.go names the file that changes. cmd/tei/** is too broad for one test",
	"tei-org-case-mapping-bridge-ci.yml internal/operations":                        "runs ./internal/operations with two -run names while paths name one test file. internal/operations/** would start this job on every operations change",
	"tei-org-headless-native-ci.yml internal/operations":                            "runs ./internal/operations with -run TestOrgHeadless while paths name one test file. internal/operations/** would start this job on every operations change",
	"tei-postgres-unrun-suites-ci.yml cmd/tei":                                      "runs ./cmd/tei with three -run names; paths list the six cmd/tei files those tests live in. cmd/tei/** would start a 30-minute service-container job on every CLI change, so the file list is the intended trade-off and not a forgotten entry",
	"tei-postgres-unrun-suites-ci.yml internal/operations":                          "runs ./internal/operations with the org-axis -run names; paths list the six org-axis test files. internal/operations/** would start a 30-minute service-container job on nearly every PR",
	"tei-postgres-unrun-suites-ci.yml tools/reference-first-web-target":             "runs ./tools/reference-first-web-target with three -run names; paths list the three read tests and the reader source. The package glob would also start the job for unrelated page changes",
	"tei-ra1-browse-preexecutor-tests.yml cmd/tei":                                  "runs ./cmd/tei in the browse preexecutor job, but cmd/tei is not its subject and repository-health covers it",
	"tei-runner-order-submit-postgres-ci.yml tools/reference-first-web-target":      "runs ./tools/reference-first-web-target filtered to TestRunnerOrderSubmit; paths name the three order-submit files. The package glob would start the postgres job for unrelated page changes",
	"tei-runtime-engine-evidence-replay.yml internal/plugin/runtime":                "runs ./internal/plugin/runtime filtered to one replay test; paths name that test file. The package glob would run the 35-minute replay job on unrelated runtime changes",
	"tei-scenario-selection-pin-qualification.yml cmd/tei":                          "runs ./cmd/tei with scenario -run names; paths list cmd/tei/scenario*.go. cmd/tei/** is too broad for those tests",
	"tei-scenario-selection-pin-qualification.yml tools/reference-change-assurance": "runs ./tools/reference-change-assurance with two -run names; paths list the two files that hold them. The package glob would start the job for unrelated reference changes",
	"tei-transform-historical-head-qualification.yml internal/plugin/transform":     "runs ./internal/plugin/transform/... against a fixed historical commit (BROKEN_SHA/FIX_SHA), not the checkout. The package is not this workflow's working-tree target, so its paths only need the workflow file",
	"tei-workspace-source-root-platform.yml internal/workspace":                     "runs ./internal/workspace with a fixed security -run list while paths name the specific source-open files. Widening to internal/workspace/** would run a Windows job on every workspace change",
}

// workflowUnresolvedAllowed names the workflows whose `go test` package list
// comes from a shell variable, so the check cannot compare it against paths.
// They are registered rather than skipped silently: a new variable package
// list has to be added here (with a reason) or made literal.
var workflowUnresolvedAllowed = map[string]string{
	"tei-windows-go-test-measurement.yml": "runs `go test @packages` for the four Windows-only packages; listing each of them under paths/`go test ./<pkg>` would turn a measurement job into a per-package gate it was not designed to be (#2319)",
	"tei-windows-go-test-survey.yml":      "runs `go test @packages` where $packages is a PowerShell array covering 12 packages; rewriting it into twelve literal arguments changes how the survey reports, and the survey is not a required gate",
}

// workflowPathTarget is one root-module package a workflow runs `go test` on.
type workflowPathTarget struct {
	pattern string // "internal/x" or "internal/x/..." subtree form
}

// workflowTrigger is one `on:` event and the paths whitelist it uses.
type workflowTrigger struct {
	event string
	paths []string
}

// workflowPathViolation is one package that a filtered workflow tests without
// listing it in that event's paths.
type workflowPathViolation struct {
	workflow string
	event    string
	target   string
}

func (v workflowPathViolation) key() string {
	return v.workflow + " " + v.target
}

func (v workflowPathViolation) String() string {
	return fmt.Sprintf("%s: %s runs go test on ./%s but that package is not in its %s paths; add that directory glob to the event's paths, or register the pair in workflowPathsAllowed with the reason it is deferred", v.workflow, v.event, v.target, v.event)
}

// pathCovers reports whether one `paths` entry makes a change to pkg start the
// workflow. A trailing `**` is a directory glob over its subtree; an entry that
// names a file inside the package covers that file, not the package.
func pathCovers(entry, pkg string) bool {
	entry = strings.Trim(strings.TrimSpace(entry), "'\"")
	entry = strings.TrimPrefix(entry, "./")
	entry = strings.TrimSuffix(entry, "/")
	pkg = strings.TrimSuffix(pkg, "/")
	if entry == pkg || entry == "**" {
		return true
	}
	if strings.HasSuffix(entry, "/**") {
		base := strings.TrimSuffix(entry, "/**")
		return pkg == base || strings.HasPrefix(pkg, base+"/")
	}
	return false
}

// workflowPathCovered reports whether any paths entry covers the target.
func workflowPathCovered(paths []string, target workflowPathTarget) bool {
	base := strings.TrimSuffix(target.pattern, "/...")
	for _, entry := range paths {
		if pathCovers(entry, base) {
			return true
		}
	}
	return false
}

// parseWorkflowTriggers extracts the `on:` events and their paths whitelists.
// The files are read as text: the check needs only this small, stable subset,
// and a YAML dependency would add a third-party module for the whole
// repository. Comments are skipped, the inline `[a, b]` form is accepted, and
// the `&anchor` / `*alias` pair used by one workflow is resolved.
func parseWorkflowTriggers(text string) ([]workflowTrigger, error) {
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if line == "on:" {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("no on: block")
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, " ") {
			end = i
			break
		}
	}
	eventAt := map[int]string{}
	for i := start + 1; i < end; i++ {
		line := lines[i]
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   ") {
			name := strings.TrimSuffix(strings.TrimSpace(line), ":")
			if name != "" && !strings.HasPrefix(name, "#") {
				eventAt[i] = name
			}
		}
	}
	anchors := map[string][]string{}
	var out []workflowTrigger
	for i := start + 1; i < end; i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || !strings.HasPrefix(trimmed, "paths:") {
			continue
		}
		event := ""
		for at := i; at > start; at-- {
			if name, ok := eventAt[at]; ok {
				event = name
				break
			}
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, "paths:"))
		anchorName := ""
		if strings.HasPrefix(value, "&") {
			anchorName = strings.TrimPrefix(value, "&")
			value = ""
		}
		if strings.HasPrefix(value, "*") {
			name := strings.TrimPrefix(value, "*")
			items, ok := anchors[name]
			if !ok {
				return nil, fmt.Errorf("unresolved paths alias %q", name)
			}
			out = append(out, workflowTrigger{event: event, paths: items})
			continue
		}
		if value != "" {
			// Inline flow list: paths: [a, b]
			value = strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
			var items []string
			for _, item := range strings.Split(value, ",") {
				if item = strings.TrimSpace(item); item != "" {
					items = append(items, item)
				}
			}
			out = append(out, workflowTrigger{event: event, paths: items})
			continue
		}
		pathIndent := len(line) - len(strings.TrimLeft(line, " "))
		var items []string
		j := i + 1
		for ; j < end; j++ {
			next := lines[j]
			trimmedNext := strings.TrimSpace(next)
			if trimmedNext == "" || strings.HasPrefix(trimmedNext, "#") {
				continue
			}
			nextIndent := len(next) - len(strings.TrimLeft(next, " "))
			if nextIndent <= pathIndent && !strings.HasPrefix(trimmedNext, "-") {
				break
			}
			if !strings.HasPrefix(trimmedNext, "-") {
				break
			}
			item := strings.TrimSpace(strings.TrimPrefix(trimmedNext, "-"))
			if comment := strings.Index(item, " #"); comment >= 0 {
				item = strings.TrimSpace(item[:comment])
			}
			items = append(items, item)
		}
		if anchorName != "" {
			anchors[anchorName] = items
		}
		out = append(out, workflowTrigger{event: event, paths: items})
		i = j - 1
	}
	return out, nil
}

// logicalLines joins a shell command split across physical lines by a trailing
// backslash, so `go test -count=1 \` + `./pkg` is seen as one command.
func logicalLines(text string) []string {
	var out []string
	current := ""
	for _, line := range strings.Split(text, "\n") {
		trimmedRight := strings.TrimRight(line, " \t")
		if strings.HasSuffix(trimmedRight, "\\") {
			current += strings.TrimSuffix(trimmedRight, "\\") + " "
			continue
		}
		out = append(out, current+line)
		current = ""
	}
	if current != "" {
		out = append(out, current)
	}
	return out
}

// parseGoTestTargets extracts root-module `go test ./<pkg>` package patterns.
// Only explicit package paths are targets: `go test ./...` and `.` are skipped
// because a whole-module run is not a package claim, and steps under another
// working directory belong to a nested module. A package list that comes from a
// shell variable is returned in unresolved rather than guessed.
func parseGoTestTargets(text string) (targets []workflowPathTarget, unresolved []string) {
	workingDir := ""
	seen := map[string]bool{}
	for i, line := range logicalLines(text) {
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- name:") && indent <= 8 {
			workingDir = ""
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "working-directory:") {
			workingDir = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "working-directory:")), "'\"")
			continue
		}
		at := strings.Index(trimmed, "go test")
		if at < 0 || strings.HasPrefix(trimmed[:at], "#") {
			continue
		}
		if workingDir != "" && workingDir != "." {
			continue
		}
		args := stripQuoted(trimmed[at+len("go test"):])
		for _, token := range strings.Fields(args) {
			token = strings.Trim(token, "()[]{},")
			token = strings.TrimSuffix(token, "\\")
			if token == "." || token == "./..." || token == "..." || token == "./." {
				continue
			}
			if strings.HasPrefix(token, "$") || strings.HasPrefix(token, "@") {
				unresolved = append(unresolved, fmt.Sprintf("line %d: package list is a shell variable (%s)", i+1, token))
				continue
			}
			if !strings.HasPrefix(token, "./") {
				continue
			}
			pattern := strings.TrimSuffix(strings.TrimPrefix(token, "./"), "/")
			if pattern == "" || pattern == "..." {
				continue
			}
			if !seen[pattern] {
				seen[pattern] = true
				targets = append(targets, workflowPathTarget{pattern: pattern})
			}
		}
	}
	return targets, unresolved
}

// stripQuoted removes single- and double-quoted spans so a -run pattern cannot
// be mistaken for a package argument.
func stripQuoted(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\'' || c == '"' {
			i++
			for i < len(s) && s[i] != c {
				i++
			}
			b.WriteByte(' ')
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// checkWorkflowPaths reports workflow/package pairs where a filtered trigger
// does not cover a package the workflow tests, plus the workflows whose package
// list could not be read at all. The second list must not be dropped: an
// unreadable list is an unchecked workflow, not a clean one.
func checkWorkflowPaths(root string) (violations []workflowPathViolation, unresolved []string, err error) {
	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		return nil, nil, err
	}
	if len(workflows) == 0 {
		return nil, nil, fmt.Errorf("no workflow files found under %s", root)
	}
	for _, workflow := range workflows {
		raw, err := os.ReadFile(workflow)
		if err != nil {
			return nil, nil, err
		}
		text := string(raw)
		triggers, err := parseWorkflowTriggers(text)
		if err != nil {
			// A workflow whose on: block this parser cannot read is skipped,
			// not assumed to have no filter.
			continue
		}
		if len(triggers) == 0 {
			continue
		}
		targets, unknown := parseGoTestTargets(text)
		name := filepath.Base(workflow)
		if len(unknown) > 0 {
			unresolved = append(unresolved, name)
		}
		if len(targets) == 0 {
			continue
		}
		for _, target := range targets {
			for _, trigger := range triggers {
				if workflowPathCovered(trigger.paths, target) {
					continue
				}
				violations = append(violations, workflowPathViolation{
					workflow: name,
					event:    trigger.event,
					target:   strings.TrimSuffix(target.pattern, "/..."),
				})
			}
		}
	}
	return violations, unresolved, nil
}

func TestWorkflowsListEveryPackageTheyTest(t *testing.T) {
	root := filepath.Join("..", "..")
	violations, unresolved, err := checkWorkflowPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) == 0 && len(unresolved) == 0 {
		t.Fatal("no workflow/package pairs inspected; the check would pass vacuously")
	}
	covered := map[string]bool{}
	for _, v := range violations {
		covered[v.key()] = true
		if _, ok := workflowPathsAllowed[v.key()]; ok {
			continue
		}
		t.Errorf("%v", v)
	}
	for key, reason := range workflowPathsAllowed {
		if strings.TrimSpace(reason) == "" {
			t.Errorf("%s is allowed without a reason", key)
		}
		if !covered[key] {
			t.Errorf("%s is no longer uncovered; remove it from workflowPathsAllowed", key)
		}
	}
	unreadable := map[string]bool{}
	for _, name := range unresolved {
		unreadable[name] = true
		if _, ok := workflowUnresolvedAllowed[name]; ok {
			continue
		}
		t.Errorf("%s: the go test package list is a shell variable, so no package can be checked against its paths; use literal package arguments or register the workflow in workflowUnresolvedAllowed with the reason it is deferred", name)
	}
	for name, reason := range workflowUnresolvedAllowed {
		if strings.TrimSpace(reason) == "" {
			t.Errorf("%s is exempted without a reason", name)
		}
		if !unreadable[name] {
			t.Errorf("%s no longer has a variable package list; remove it from workflowUnresolvedAllowed", name)
		}
	}
}

// TestWorkflowPathsAllowlistDoesNotGrow pins the size of the deferred list, so
// a new hole cannot be appended quietly: either a listed pair is fixed and
// removed, or this number is raised in the same change with a reason
// (tools/repository-plan-check pins its diagnostic allowlist the same way).
func TestWorkflowPathsAllowlistDoesNotGrow(t *testing.T) {
	const max = 17
	if len(workflowPathsAllowed) > max {
		t.Fatalf("the deferred list has %d entries (max %d); fix one and remove it, or raise the bound with a reason in the same change", len(workflowPathsAllowed), max)
	}
	if len(workflowUnresolvedAllowed) > 2 {
		t.Fatalf("the unresolved-exemption list has %d entries (max 2); a variable package list must become literal or be registered by replacing one", len(workflowUnresolvedAllowed))
	}
}

// TestPathCovers is the unit control for the glob rule. A file entry inside a
// package must not count as coverage: that is the exact shape the postgres job
// had before #5449 (read_scenario_page_postgres_test.go listed, the package
// still breaking).
func TestPathCovers(t *testing.T) {
	for _, tc := range []struct {
		entry, pkg string
		want       bool
	}{
		{"internal/x/**", "internal/x", true},
		{"internal/x/**", "internal/x/y", true},
		{"internal/x/**", "internal/xylophone", false},
		{"internal/x", "internal/x", true},
		{"'internal/x/**'", "internal/x", true},
		{"./internal/x/**", "internal/x", true},
		{"internal/x", "internal/x/y", false},
		{"internal/x/y.go", "internal/x", false},
		{"internal/**", "internal/x", true},
		{"**", "internal/x", true},
		{"**", ".", true},
		{"cmd/tei/**", "cmd/tei", true},
		{"cmd/tei/**", "cmd/tei-studio", false},
	} {
		if got := pathCovers(tc.entry, tc.pkg); got != tc.want {
			t.Errorf("pathCovers(%q, %q) = %v, want %v", tc.entry, tc.pkg, got, tc.want)
		}
	}
}

// TestWorkflowPathsCheckFailsClosed builds a workflow whose paths do not cover
// the package it tests and requires the checker to report it. Without this the
// guard could pass vacuously after a parser change.
func TestWorkflowPathsCheckFailsClosed(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	workflow := `name: probe
on:
  pull_request:
    paths:
      - 'cmd/tei/cli_view.go'
      - '.github/workflows/probe.yml'
jobs:
  probe:
    steps:
      - name: test
        run: |
          go test -count=1 -run '^TestX$' \
            ./internal/driver ./cmd/tei
`
	if err := os.WriteFile(filepath.Join(dir, "probe.yml"), []byte(workflow), 0o600); err != nil {
		t.Fatal(err)
	}
	violations, unresolved, err := checkWorkflowPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, v := range violations {
		keys[v.key()] = true
	}
	if len(unresolved) != 0 {
		t.Fatalf("unexpected unresolved package lists: %v", unresolved)
	}
	for _, want := range []string{"probe.yml cmd/tei", "probe.yml internal/driver"} {
		if !keys[want] {
			t.Errorf("coverage gap not reported: %s (got %v)", want, keys)
		}
	}
	// A paths entry that covers the package must clear the pair.
	covered := `name: probe
on:
  pull_request:
    paths:
      - 'internal/driver/**'
      - 'cmd/tei/**'
jobs:
  probe:
    steps:
      - name: test
        run: go test ./cmd/tei ./internal/driver
`
	if err := os.WriteFile(filepath.Join(dir, "probe.yml"), []byte(covered), 0o600); err != nil {
		t.Fatal(err)
	}
	violations, _, err = checkWorkflowPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("covered package still reported: %v", violations)
	}
	// A `working-directory` step belongs to a nested module and must be out of
	// scope, otherwise the check would demand the outer paths cover it.
	nested := `name: probe
on:
  pull_request:
    paths:
      - 'cmd/tei-studio/**'
jobs:
  probe:
    steps:
      - name: nested
        working-directory: cmd/tei-studio
        run: go test .
`
	if err := os.WriteFile(filepath.Join(dir, "probe.yml"), []byte(nested), 0o600); err != nil {
		t.Fatal(err)
	}
	violations, _, err = checkWorkflowPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("nested-module step should be out of scope: %v", violations)
	}
	// A shell-variable package list must surface as unresolved, not as clean.
	variable := `name: probe
on:
  pull_request:
    paths:
      - 'internal/driver/**'
jobs:
  probe:
    steps:
      - name: variable
        run: go test @packages
`
	if err := os.WriteFile(filepath.Join(dir, "probe.yml"), []byte(variable), 0o600); err != nil {
		t.Fatal(err)
	}
	_, unresolved, err = checkWorkflowPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(unresolved) != 1 || unresolved[0] != "probe.yml" {
		t.Fatalf("variable package list not reported as unresolved: %v", unresolved)
	}
}
