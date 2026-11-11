package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"tei/internal/buildintent"
)

// §4.50: the whole chain, on the real subject. A declaration that names ./cmd/tei
// and its module-local closure is resolved into a plan, the plan is built from a
// tree made only of its sources, and the five conditions hold.
//
// This is the observation §4.49 said was missing: there, the isolated build of
// ./cmd/tei was measured but the claim was not re-established on that subject.

type fullChainReport struct {
	OK  bool `json:"ok"`
	L2B *struct {
		Claimed bool     `json:"claimed"`
		Unmet   []string `json:"unmet"`
	} `json:"l2b"`
	PlanReceipt *struct {
		State string `json:"state"`
	} `json:"planReceipt"`
	PlanCrossCheck *struct {
		State string `json:"state"`
	} `json:"planCrossCheck"`
	PlanArtifactBinding *struct {
		State string `json:"state"`
	} `json:"planArtifactBinding"`
	ArtifactReproducibility *struct {
		State string `json:"state"`
	} `json:"artifactReproducibility"`
	DefinitionBinding *struct {
		State string `json:"state"`
	} `json:"definitionBinding"`
	ChildIsolation *struct {
		State       string   `json:"state"`
		MemberCount int      `json:"memberCount"`
		ModuleFiles []string `json:"moduleFiles"`
	} `json:"childIsolation"`
	ChildRun *struct {
		ChildArtifactDigest string `json:"childArtifactDigest"`
	} `json:"childRun"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// writeFullChainDeclaration writes a declaration naming every module-local
// package the build of entry reads, plus the module context. The digests are
// computed by the resolver, exactly as the entry will compute them.
//
// It also returns the member count the isolated tree must have, so the caller
// asserts against the SAME package list this declaration was built from
// instead of a stale literal.
func writeFullChainDeclaration(t *testing.T, repoRoot, entry, dir string) (string, int) {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	listCmd := exec.Command(goExe, "list", "-deps", "-f", "{{if not .Standard}}{{.Dir}}{{end}}", entry)
	listCmd.Dir = repoRoot
	out, err := listCmd.Output()
	if err != nil {
		t.Fatalf("go list %s: %v", entry, err)
	}
	dirs := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		d := strings.TrimSpace(line)
		if d != "" && strings.HasPrefix(d, repoRoot) {
			dirs[d] = true
		}
	}
	ordered := make([]string, 0, len(dirs))
	for d := range dirs {
		ordered = append(ordered, d)
	}
	sort.Strings(ordered)

	resolver := buildintent.DirSourceResolver{Root: repoRoot}
	type source struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
		Ref  string `json:"ref"`
	}
	// Only locators go in the declaration: the resolver derives the digests.
	sources := make([]source, 0, len(ordered)+2)
	for i, d := range ordered {
		rel, err := filepath.Rel(repoRoot, d)
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, source{Kind: "go-package", ID: "pkg" + itoa(i), Ref: filepath.ToSlash(rel)})
	}
	sources = append(sources,
		source{Kind: "go-module", ID: "go.mod", Ref: "go.mod"},
		source{Kind: "go-module", ID: "go.sum", Ref: "go.sum"},
	)
	_ = resolver

	body := map[string]any{
		"formatVersion": 1,
		"subject": map[string]any{
			"kind": "plugin", "id": "tei.command.tei", "version": "0.0.0", "definitionSource": "pkg0",
		},
		"sources":    sources,
		"builder":    map[string]any{"contract": "tei.plugin-build", "compatibleMajor": 1},
		"recipe":     map[string]any{"kind": "go-command", "entrypoint": entry},
		"outputKind": "runtime-artifact",
		"target":     "host/linux/amd64",
		"profile":    "release",
	}
	encoded, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "intent.json")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	// One member per declared source: every go-package plus go.mod and go.sum.
	return path, len(sources)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var parts []string
	for i > 0 {
		parts = append([]string{string(rune('0' + i%10))}, parts...)
		i /= 10
	}
	return strings.Join(parts, "")
}

func TestSelfBuildFullChainOnTheRealSubject(t *testing.T) {
	if os.Getenv("TEI_SELF_BUILD_FULL_CHAIN") == "" {
		// This builds the whole tree twice and takes around four minutes. It is
		// opt-in rather than skipped by -short, because the default of a plain
		// `go test ./...` is what most callers run.
		t.Skip("set TEI_SELF_BUILD_FULL_CHAIN=1 to run the full chain on ./cmd/tei (about 4 minutes)")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cli := buildToolBinary(t, ".", "tei")
	builder := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-go-build-from-plan"), "go-build-from-plan")
	authoring := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-build-catalog-authoring"), "catalog-author")
	resolverTool := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-build-authoring-intent"), "resolver")

	// Go writes its module cache read-only, so t.TempDir's cleanup cannot remove
	// the state directory this build creates. A manual directory with a
	// permission-fixing cleanup keeps the failure from masking the result.
	dir, err := os.MkdirTemp("", "tei-full-chain-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { removeReadOnlyTree(dir) })
	workspace := outerWorkspace(t, dir)
	intentPath, wantMembers := writeFullChainDeclaration(t, repoRoot, "./cmd/tei", dir)

	// The catalog, authored from a capability declaration.
	builderBytes, err := os.ReadFile(builder)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "builder"), builderBytes, 0o755); err != nil {
		t.Fatal(err)
	}
	declaration := `{
  "providers": [
    {"plugin": "tei.builder.go", "version": "0.0.0", "implementation": "go.build-from-plan",
     "contract": "tei.plugin-build", "contractMajor": 1, "artifact": "builder",
     "recipeKinds": ["go-command"], "targets": ["host/linux/amd64"], "profiles": ["release"]}
  ],
  "targets": [{"target": "host/linux/amd64", "profiles": ["release"]}]
}
`
	declarationPath := filepath.Join(dir, "capability.json")
	if err := os.WriteFile(declarationPath, []byte(declaration), 0o600); err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(dir, "catalog.json")
	if out, err := exec.Command(authoring, "-declaration", declarationPath, "-root", dir, "-output", catalogPath).CombinedOutput(); err != nil {
		t.Fatalf("author the catalog: %v\n%s", err, out)
	}
	catalogBytes, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}

	// The plan, derived by the reference resolver, so the child is given exactly
	// the plan the entry will compare against.
	// The entry derives it itself, so the child only needs the declaration path.
	planPath := filepath.Join(dir, "plan.json")
	derive := func() {
		cmd := exec.Command(resolverTool, "-intent", intentPath, "-catalog", catalogPath, "-source-root", repoRoot)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("derive the plan: %v", err)
		}
		var receipt struct {
			Plan json.RawMessage `json:"plan"`
		}
		if err := json.Unmarshal(out, &receipt); err != nil {
			t.Fatalf("decode the receipt: %v", err)
		}
		if err := os.WriteFile(planPath, append(receipt.Plan, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	derive()

	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	// ./cmd/tei pulls external modules; the build may only take them from the
	// supply the entry names, so this test points at the host's download cache.
	cacheOut, cacheErr := exec.Command(goExe, "env", "GOMODCACHE").Output()
	if cacheErr != nil {
		t.Skipf("go env GOMODCACHE unavailable: %v", cacheErr)
	}
	supply := filepath.Join(strings.TrimSpace(string(cacheOut)), "cache", "download")
	if _, statErr := os.Stat(supply); statErr != nil {
		t.Skipf("the module download cache is unavailable: %v", statErr)
	}
	childArg, err := json.Marshal([]string{
		"-plan", planPath,
		"-intent", intentPath,
		"-repo", repoRoot,
		"-go", goExe,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(dir, "artifact")
	var stdout, stderr strings.Builder
	cmd := exec.Command(cli,
		"self-build",
		"-workspace", workspace,
		"-intent", "tei.command.tei",
		"-intent-file", intentPath,
		"-build-catalog", catalogPath,
		"-build-catalog-digest", digestOfBytes(catalogBytes),
		"-build-source-root", repoRoot,
		"-child", builder,
		"-child-arg", string(childArg),
		"-supply", supply,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
	)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run()

	var report fullChainReport
	if err := json.Unmarshal([]byte(stdout.String()), &report); err != nil {
		t.Fatalf("decode the report: %v\n%s", err, stderr.String())
	}
	if !report.OK {
		t.Fatalf("the attempt failed: %+v\n%s", report.Error, stderr.String())
	}
	if report.L2B == nil || !report.L2B.Claimed {
		t.Fatalf("l2b was not claimed: %+v", report.L2B)
	}
	for name, state := range map[string]string{
		"planReceipt":             receiptStateOf(report.PlanReceipt),
		"planCrossCheck":          recordStateOf(report.PlanCrossCheck),
		"planArtifactBinding":     recordStateOf(report.PlanArtifactBinding),
		"artifactReproducibility": recordStateOf(report.ArtifactReproducibility),
	} {
		want := map[string]string{
			"planReceipt":             intentResolutionResolved,
			"planCrossCheck":          planCrossCheckMatched,
			"planArtifactBinding":     planArtifactBound,
			"artifactReproducibility": reproducibilityReproduced,
		}[name]
		if state != want {
			t.Fatalf("%s = %q, want %q", name, state, want)
		}
	}
	if report.DefinitionBinding == nil || report.DefinitionBinding.State != definitionBindingBound {
		t.Fatalf("definitionBinding = %+v", report.DefinitionBinding)
	}
	// The plan named the module context, so isolation is the default and the tree
	// is made only of planned sources.
	if report.ChildIsolation == nil || report.ChildIsolation.State != childIsolationIsolated {
		t.Fatalf("childIsolation = %+v", report.ChildIsolation)
	}
	if len(report.ChildIsolation.ModuleFiles) != 0 {
		t.Fatalf("the tree took module files from the working tree: %v", report.ChildIsolation.ModuleFiles)
	}
	// The member count is DERIVED from the same package list this declaration
	// was written from, plus go.mod and go.sum. It used to be the literal 75,
	// which was true when the build closure had 73 module-local packages; adding
	// one package anywhere in the closure made the test fail even though the
	// chain worked. A literal here is a maintenance trap, not a claim (#3394).
	if report.ChildIsolation.MemberCount != wantMembers {
		t.Fatalf("memberCount = %d, want %d (the declared %d packages plus go.mod and go.sum)",
			report.ChildIsolation.MemberCount, wantMembers, wantMembers-2)
	}
	if report.ChildRun == nil || report.ChildRun.ChildArtifactDigest == "" {
		t.Fatal("the child reported no artifact")
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("no artifact was produced: %v", err)
	}
}

// removeReadOnlyTree deletes a tree whose files may be read-only, ignoring the
// errors that a partially removed tree produces.
func removeReadOnlyTree(root string) {
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !entry.IsDir() {
			_ = os.Chmod(path, 0o644)
		}
		return nil
	})
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			_ = os.Chmod(path, 0o755)
		}
		return nil
	})
	_ = os.RemoveAll(root)
}

func receiptStateOf(r *struct {
	State string `json:"state"`
}) string {
	if r == nil {
		return ""
	}
	return r.State
}

func recordStateOf(r *struct {
	State string `json:"state"`
}) string {
	if r == nil {
		return ""
	}
	return r.State
}
