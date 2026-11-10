package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

// 実TEI CLI(cmd/tei)を、既存のBuildExecutionPlan / ArtifactSet /
// BuildExecutionResult / compareArtifactSets へ限定的に束縛する試験。
// 新しいPlan型・DSL・Plugin IDは作らない。ここで得られるのは
// 「同一入力・同一toolchainでの2回のbuildが同じbyteになった」という観測であり、
// Stage収束・seed信頼・Host署名・Owner受理は主張しない。

const (
	realBuildStatusBound           = "bound-reproducible-observed"
	realBuildStatusBlocked         = "blocked"
	realBuildStatusUnconfirmed     = "unconfirmed"
	realBuildStatusNotReproducible = "not-reproducible"
	realBuildNotClaimedStatement   = "Stage収束とseed信頼は未主張"
)

// 閉包が確認できていない間は成功にしない。goが無い環境ではSkipする
// (Skipは成功の根拠にならない)。
func requireRealGo(t *testing.T) string {
	t.Helper()
	goTool, err := execLookPathGo()
	if err != nil {
		realBuildSkipOrFail(t, "Go executable is unavailable: %v", err)
	}
	return goTool
}

// TEI_REQUIRE_REAL_BUILD=1 のとき、Skipを失敗に倒す(Skipの見逃し防止)。
func realBuildSkipOrFail(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv("TEI_REQUIRE_REAL_BUILD") == "1" {
		t.Fatalf("TEI_REQUIRE_REAL_BUILD=1 but the real build cannot run: "+format, args...)
	}
	t.Skipf("(SKIP is not a success) "+format, args...)
}

func execLookPathGo() (string, error) { return exec.LookPath("go") }

type closureFile struct {
	Rel    string // repository root relative, slash separated
	Kind   string
	Digest string
	Abs    string
}

type sourceClosure struct {
	Files           []closureFile
	ExternalModules []string // path@version, pinned only through go.sum
}

func (c sourceClosure) byRel() map[string]closureFile {
	out := make(map[string]closureFile, len(c.Files))
	for _, f := range c.Files {
		out[f.Rel] = f
	}
	return out
}

type goListModule struct {
	Path    string
	Version string
	Main    bool
	Dir     string
	GoMod   string
	Replace *goListModule
}

type goListError struct{ Err string }

type goListPackage struct {
	ImportPath   string
	Dir          string
	Standard     bool
	Incomplete   bool
	Module       *goListModule
	Error        *goListError
	DepsErrors   []*goListError
	GoFiles      []string
	CgoFiles     []string
	CFiles       []string
	CXXFiles     []string
	MFiles       []string
	HFiles       []string
	FFiles       []string
	SFiles       []string
	SwigFiles    []string
	SwigCXXFiles []string
	SysoFiles    []string
	EmbedFiles   []string
}

// buildへ渡す環境は許可リストで作る。os.Environ()をそのまま通さない。
// 出力に影響しうるGo設定(GOEXPERIMENT、GOAMD64等のGOARCH別設定、GOENV、GODEBUG、
// GOFIPS140、GOCACHEPROG等)は呼び出し側の値を通さず、空または固定値にする。
var realBuildPassThroughEnv = map[string]bool{
	"PATH": true, "PATHEXT": true, "SYSTEMROOT": true, "SYSTEMDRIVE": true, "WINDIR": true,
	"COMSPEC": true, "TEMP": true, "TMP": true, "TMPDIR": true, "HOME": true,
	"USERPROFILE": true, "LOCALAPPDATA": true, "APPDATA": true, "XDG_CACHE_HOME": true,
	"GOROOT": true, "GOPATH": true, "GOCACHE": true, "GOMODCACHE": true, "GOTMPDIR": true,
	"GOPROXY": true, "GOSUMDB": true, "GONOSUMDB": true, "GOPRIVATE": true, "GONOPROXY": true,
	"GOINSECURE": true, "GOVCS": true,
}

func realBuildPinnedEnv() map[string]string {
	return map[string]string{
		"GOFLAGS": "", "GOWORK": "off", "GOTOOLCHAIN": "local", "CGO_ENABLED": "0",
		"GOOS": runtime.GOOS, "GOARCH": runtime.GOARCH, "GO111MODULE": "on", "GOENV": "off",
		"GOEXPERIMENT": "", "GODEBUG": "", "GOFIPS140": "",
		"GOAMD64": "", "GOARM": "", "GOARM64": "", "GO386": "", "GOMIPS": "", "GOMIPS64": "",
		"GOPPC64": "", "GORISCV64": "", "GOWASM": "", "GOCACHEPROG": "",
	}
}

func realBuildEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(kv, "=", 2)[0])
		if realBuildPassThroughEnv[key] {
			env = append(env, kv)
		}
	}
	for k, v := range realBuildPinnedEnv() {
		env = append(env, k+"="+v)
	}
	sort.Strings(env)
	return env
}

// build引数は1箇所で定義し、Planのflags parameterもここから作る。
var realBuildFlags = []string{"-trimpath", "-buildvcs=false", "-ldflags=-buildid="}

// 出力に影響しうるgo envのキー。観測値をPlanのParametersへ入れる(buildと同じ環境で取得)。
var realBuildGoEnvKeys = []string{
	"CGO_ENABLED", "GO111MODULE", "GO386", "GOAMD64", "GOARCH", "GOARM", "GOARM64", "GODEBUG",
	"GOENV", "GOEXPERIMENT", "GOFIPS140", "GOFLAGS", "GOMIPS", "GOMIPS64", "GOOS", "GOPPC64",
	"GORISCV64", "GOTOOLCHAIN", "GOWASM", "GOWORK",
}

func realBuildObserveGoEnv(goTool string, env []string) ([]plugin.BuildParameter, error) {
	cmd := exec.Command(goTool, append([]string{"env", "-json"}, realBuildGoEnvKeys...)...)
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var observed map[string]string
	if err := json.Unmarshal(out, &observed); err != nil {
		return nil, err
	}
	params := make([]plugin.BuildParameter, 0, len(realBuildGoEnvKeys))
	for _, k := range realBuildGoEnvKeys {
		v, ok := observed[k]
		if !ok {
			v = "<not-reported>"
		}
		params = append(params, plugin.BuildParameter{Name: "goenv." + k, Value: v})
	}
	return params, nil
}

func realBuildDigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func realBuildFindRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.HasPrefix(strings.TrimSpace(string(data)), "module tei\n") ||
			err == nil && strings.TrimSpace(strings.SplitN(string(data), "\n", 2)[0]) == "module tei" {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root (module tei) was not found")
		}
		dir = parent
	}
}

func realBuildCanon(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	return filepath.Clean(path)
}

func realBuildRel(root, path string) (string, bool) {
	rel, err := filepath.Rel(realBuildCanon(root), realBuildCanon(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func realBuildGoSumHas(goSum []byte, path, version string) bool {
	gomod, h1 := false, false
	for _, line := range strings.Split(string(goSum), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 || f[0] != path {
			continue
		}
		if f[1] == version+"/go.mod" {
			gomod = true
		}
		if f[1] == version {
			h1 = true
		}
	}
	return gomod && h1
}

// go list -deps -json でモジュール内のbuild入力を列挙する。
// 列挙できない・解決が曖昧・go.sumが無い場合はerror(=未確定)を返す。
func enumerateGoSourceClosure(goTool, repoRoot, moduleDir, pkg string, env []string) (sourceClosure, error) {
	var closure sourceClosure
	cmd := exec.Command(goTool, "list", "-deps", "-json", pkg)
	cmd.Dir = moduleDir
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return closure, fmt.Errorf("go list -deps failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	goSum, err := os.ReadFile(filepath.Join(moduleDir, "go.sum"))
	if err != nil {
		return closure, fmt.Errorf("main module go.sum is required: %w", err)
	}
	files := map[string]closureFile{}
	add := func(abs, kind string) error {
		rel, ok := realBuildRel(repoRoot, abs)
		if !ok {
			return fmt.Errorf("closure file %s is outside the repository", abs)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		if _, dup := files[rel]; !dup {
			files[rel] = closureFile{Rel: rel, Kind: kind, Digest: realBuildDigestBytes(data), Abs: abs}
		}
		return nil
	}
	externals := map[string]struct{}{}
	modDirs := map[string]struct{}{}
	dec := json.NewDecoder(&stdout)
	packages := 0
	for dec.More() {
		var p goListPackage
		if err := dec.Decode(&p); err != nil {
			return closure, fmt.Errorf("go list output is not decodable: %w", err)
		}
		if p.Error != nil || len(p.DepsErrors) > 0 || p.Incomplete {
			return closure, fmt.Errorf("package %s is incomplete or has errors", p.ImportPath)
		}
		if p.Standard {
			continue
		}
		packages++
		if p.Module == nil {
			return closure, fmt.Errorf("package %s has no module (unresolved input)", p.ImportPath)
		}
		if rel, inRepo := realBuildRel(repoRoot, p.Dir); inRepo {
			_ = rel
			groups := []struct {
				kind  string
				names [][]string
			}{
				{"go-source", [][]string{p.GoFiles, p.CgoFiles}},
				{"native-source", [][]string{p.CFiles, p.CXXFiles, p.MFiles, p.HFiles, p.FFiles, p.SFiles, p.SwigFiles, p.SwigCXXFiles, p.SysoFiles}},
				{"embed-file", [][]string{p.EmbedFiles}},
			}
			for _, g := range groups {
				for _, names := range g.names {
					for _, name := range names {
						if err := add(filepath.Join(p.Dir, filepath.FromSlash(name)), g.kind); err != nil {
							return closure, err
						}
					}
				}
			}
			if p.Module.Dir == "" {
				return closure, fmt.Errorf("package %s module directory is unknown", p.ImportPath)
			}
			modDirs[p.Module.Dir] = struct{}{}
			continue
		}
		mod := p.Module
		if mod.Replace != nil {
			if mod.Replace.Version == "" {
				return closure, fmt.Errorf("module %s replaced by a local directory outside the repository", mod.Path)
			}
			mod = mod.Replace
		}
		if mod.Version == "" || !realBuildGoSumHas(goSum, mod.Path, mod.Version) {
			return closure, fmt.Errorf("external module %s@%s is not pinned by go.sum", mod.Path, mod.Version)
		}
		externals[mod.Path+"@"+mod.Version] = struct{}{}
	}
	if packages == 0 {
		return closure, fmt.Errorf("go list returned no non-standard package")
	}
	modDirs[moduleDir] = struct{}{}
	for dir := range modDirs {
		if err := add(filepath.Join(dir, "go.mod"), "go-mod"); err != nil {
			return closure, err
		}
		sum := filepath.Join(dir, "go.sum")
		if _, err := os.Stat(sum); err == nil {
			if err := add(sum, "go-sum"); err != nil {
				return closure, err
			}
		} else if dir == moduleDir {
			return closure, fmt.Errorf("main module go.sum is required")
		}
	}
	for _, f := range files {
		closure.Files = append(closure.Files, f)
	}
	sort.Slice(closure.Files, func(i, j int) bool { return closure.Files[i].Rel < closure.Files[j].Rel })
	for e := range externals {
		closure.ExternalModules = append(closure.ExternalModules, e)
	}
	sort.Strings(closure.ExternalModules)
	return closure, nil
}

func realBuildToolchainVersion(goTool string, env []string) (string, error) {
	cmd := exec.Command(goTool, "version")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// 既存のBuildExecutionPlanへ閉包を並べる。DSLは作らない。
func newRealBuildPlan(entrypoint, goVersion, toolchainDigest string, goEnv []plugin.BuildParameter, closure sourceClosure) BuildExecutionPlan {
	sources := make([]plugin.BuildSource, 0, len(closure.Files))
	for _, f := range closure.Files {
		sources = append(sources, plugin.BuildSource{Kind: f.Kind, ID: f.Rel, Locator: f.Abs, Digest: f.Digest})
	}
	params := append([]plugin.BuildParameter(nil), goEnv...)
	params = append(params,
		plugin.BuildParameter{Name: "flags", Value: strings.Join(realBuildFlags, " ")},
		plugin.BuildParameter{Name: "go-version", Value: goVersion},
		plugin.BuildParameter{Name: "go-executable-digest", Value: toolchainDigest},
	)
	return BuildExecutionPlan{
		FormatVersion: buildExecutionPlanFormatVersion,
		Sources:       sources,
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.go_real_tei_cli_reference"),
			Version:         goVersion,
			ContractVersion: "tei.reference.go-build/v1",
			ArtifactDigest:  toolchainDigest,
		},
		Recipe:         plugin.BuildRecipe{Kind: "go-build", Entrypoint: entrypoint},
		OutputContract: "go-executable",
		Target:         runtime.GOOS + "/" + runtime.GOARCH,
		Parameters:     params,
	}
}

// Planに並んだSourceが、観測した閉包と1件ずつ一致することを確かめる。
func verifyPlanSourcesMatchClosure(plan BuildExecutionPlan, closure sourceClosure) error {
	want := closure.byRel()
	got := map[string]string{}
	for _, s := range plan.Sources {
		got[s.ID] = s.Digest
	}
	var problems []string
	for id, f := range want {
		d, ok := got[id]
		switch {
		case !ok:
			problems = append(problems, "missing:"+id)
		case d != f.Digest:
			problems = append(problems, "digest-mismatch:"+id)
		}
	}
	for id := range got {
		if _, ok := want[id]; !ok {
			problems = append(problems, "outside-closure:"+id)
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("plan sources differ from the observed closure: %s", strings.Join(problems, ","))
	}
	return nil
}

type realBuildReport struct {
	Subject             string   `json:"subject"`
	Status              string   `json:"status"`
	Reason              string   `json:"reason,omitempty"`
	SourceClosureCount  int      `json:"sourceClosureCount"`
	ExternalModuleCount int      `json:"externalModuleCount"`
	ToolchainDigest     string   `json:"toolchainDigest,omitempty"`
	ToolchainVersion    string   `json:"toolchainVersion,omitempty"`
	Target              string   `json:"target"`
	PlanFingerprint     string   `json:"planFingerprint,omitempty"`
	ArtifactDigest      string   `json:"artifactDigest,omitempty"`
	ResultFingerprint   string   `json:"resultFingerprint,omitempty"`
	Reproducible        bool     `json:"reproducible"`
	NotClaimed          []string `json:"notClaimed"`
	NotClaimedStatement string   `json:"notClaimedStatement"`
}

type realBuildSpec struct {
	Subject    string // evidence label, e.g. cmd/tei
	GoTool     string
	RepoRoot   string
	ModuleDir  string
	Package    string // go package pattern relative to ModuleDir
	Entrypoint string // repository relative recipe entrypoint
	OutDir     string
	// Exec runs one build command. nil means exec; tests wrap it to observe invocations.
	Exec func(cmd *exec.Cmd) ([]byte, error)
}

type realBuildOutcome struct {
	Report  realBuildReport
	Plan    BuildExecutionPlan
	Closure sourceClosure
	Sets    []ArtifactSet
}

func newRealBuildReport(subject string) realBuildReport {
	return realBuildReport{
		Subject: subject, Target: runtime.GOOS + "/" + runtime.GOARCH,
		NotClaimed: []string{
			"stage-convergence", "seed-trust", "host-signature", "owner-acceptance",
			"stage1-stage2-comparison",
		},
		NotClaimedStatement: realBuildNotClaimedStatement,
	}
}

// 成功は「閉包を列挙でき、2回buildでき、byteが一致した」場合だけ。
// 列挙できない=unconfirmed、go build失敗=blocked。どちらも成功にしない。
func bindRealGoBuild(spec realBuildSpec) realBuildOutcome {
	out := realBuildOutcome{Report: newRealBuildReport(spec.Subject)}
	fail := func(status, format string, args ...any) realBuildOutcome {
		out.Report.Status = status
		out.Report.Reason = fmt.Sprintf(format, args...)
		return out
	}
	env := realBuildEnv()
	toolchainDigest, err := actualGoDigestFile(spec.GoTool)
	if err != nil {
		return fail(realBuildStatusUnconfirmed, "toolchain digest unavailable: %v", err)
	}
	goVersion, err := realBuildToolchainVersion(spec.GoTool, env)
	if err != nil {
		return fail(realBuildStatusUnconfirmed, "go version unavailable: %v", err)
	}
	out.Report.ToolchainDigest, out.Report.ToolchainVersion = toolchainDigest, goVersion
	closure, err := enumerateGoSourceClosure(spec.GoTool, spec.RepoRoot, spec.ModuleDir, spec.Package, env)
	if err != nil {
		return fail(realBuildStatusUnconfirmed, "source closure not enumerable: %v", err)
	}
	out.Closure = closure
	out.Report.SourceClosureCount = len(closure.Files)
	out.Report.ExternalModuleCount = len(closure.ExternalModules)
	goEnv, err := realBuildObserveGoEnv(spec.GoTool, env)
	if err != nil {
		return fail(realBuildStatusUnconfirmed, "go env unavailable: %v", err)
	}
	out.Plan = newRealBuildPlan(spec.Entrypoint, goVersion, toolchainDigest, goEnv, closure)
	planFP, err := buildExecutionPlanFingerprint(out.Plan)
	if err != nil {
		return fail(realBuildStatusUnconfirmed, "plan is invalid: %v", err)
	}
	out.Report.PlanFingerprint = planFP
	if err := verifyPlanSourcesMatchClosure(out.Plan, closure); err != nil {
		return fail(realBuildStatusUnconfirmed, "%v", err)
	}
	for _, id := range []string{"first", "second"} {
		output := filepath.Join(spec.OutDir, id, "tei-real")
		if runtime.GOOS == "windows" {
			output += ".exe"
		}
		if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
			return fail(realBuildStatusUnconfirmed, "%v", err)
		}
		args := append([]string{"build"}, realBuildFlags...)
		args = append(args, "-o", output, spec.Package)
		cmd := exec.Command(spec.GoTool, args...)
		cmd.Dir, cmd.Env = spec.ModuleDir, env
		run := spec.Exec
		if run == nil {
			run = func(c *exec.Cmd) ([]byte, error) { return c.CombinedOutput() }
		}
		if combined, err := run(cmd); err != nil {
			return fail(realBuildStatusBlocked, "%s go build failed: %v: %s", id, err, strings.TrimSpace(string(combined)))
		}
		digest, err := actualGoDigestFile(output)
		if err != nil {
			return fail(realBuildStatusUnconfirmed, "%v", err)
		}
		out.Sets = append(out.Sets, ArtifactSet{
			FormatVersion: artifactSetFormatVersion,
			Members: []ArtifactMember{{
				Role: "executable", Format: "application/octet-stream", Digest: digest, Locator: output,
			}},
		})
	}
	comparison, err := compareArtifactSets(out.Sets[0], out.Sets[1])
	if err != nil {
		return fail(realBuildStatusUnconfirmed, "%v", err)
	}
	out.Report.ArtifactDigest = out.Sets[0].Members[0].Digest
	if !comparison.Reproducible {
		return fail(realBuildStatusNotReproducible, "two builds differ: %+v", comparison)
	}
	resultFP, err := buildExecutionResultFingerprint(BuildExecutionResult{
		FormatVersion:            buildExecutionResultFormatVersion,
		ExecutionPlanFingerprint: planFP, ArtifactSetFingerprint: comparison.FirstFingerprint,
	})
	if err != nil {
		return fail(realBuildStatusUnconfirmed, "%v", err)
	}
	out.Report.ResultFingerprint = resultFP
	out.Report.Reproducible = true
	out.Report.Status = realBuildStatusBound
	return out
}

// 閉包のファイルだけを別ディレクトリへ複製する(閉包の十分性確認・負例用)。
func copyClosureTo(t *testing.T, closure sourceClosure, repoRoot, dst string) {
	t.Helper()
	for _, f := range closure.Files {
		data, err := os.ReadFile(f.Abs)
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(dst, filepath.FromSlash(f.Rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRealTEICLIBuildBindsToReferenceResult(t *testing.T) {
	goTool := requireRealGo(t)
	repoRoot := realBuildFindRepoRoot(t)
	outcome := bindRealGoBuild(realBuildSpec{
		Subject: "cmd/tei", GoTool: goTool, RepoRoot: repoRoot, ModuleDir: repoRoot,
		Package: "./cmd/tei", Entrypoint: "cmd/tei", OutDir: t.TempDir(),
	})
	report := outcome.Report
	if report.Status != realBuildStatusBound {
		t.Fatalf("real cmd/tei build was not bound: %+v", report)
	}
	if !report.Reproducible || report.SourceClosureCount < 3 ||
		report.ToolchainDigest == "" || report.ArtifactDigest == "" ||
		report.PlanFingerprint == "" || report.ResultFingerprint == "" {
		t.Fatalf("incomplete evidence: %+v", report)
	}

	files := outcome.Closure.byRel()
	for _, must := range []string{"go.mod", "go.sum", "cmd/tei/main.go"} {
		if _, ok := files[must]; !ok {
			t.Fatalf("closure lacks %s", must)
		}
	}
	for rel := range files {
		if strings.HasSuffix(rel, "_test.go") || strings.HasPrefix(rel, "docs/") ||
			strings.HasPrefix(rel, "cmd/tei-studio/") {
			t.Fatalf("closure contains an unrelated input: %s", rel)
		}
	}
	// 独立確認: cmd/tei直下の非test Goファイルは必ず閉包に入る。
	entries, err := os.ReadDir(filepath.Join(repoRoot, "cmd", "tei"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			if _, ok := files["cmd/tei/"+name]; !ok {
				t.Fatalf("closure lacks cmd/tei/%s", name)
			}
		}
	}

	// Planは同じ入力から同じ指紋になり、閉包とも1件ずつ一致する。
	again, err := buildExecutionPlanFingerprint(outcome.Plan)
	if err != nil || again != report.PlanFingerprint {
		t.Fatalf("plan fingerprint is unstable: %q %q %v", again, report.PlanFingerprint, err)
	}
	if err := verifyPlanSourcesMatchClosure(outcome.Plan, outcome.Closure); err != nil {
		t.Fatal(err)
	}
	pinned, err := pinReproducibleArtifactSet(outcome.Sets[0], outcome.Sets[1])
	if err != nil {
		t.Fatalf("reproducible artifacts were not pinned: %v", err)
	}
	setFP, err := artifactSetFingerprint(outcome.Sets[0])
	if err != nil || pinned != setFP {
		t.Fatalf("pinned identity differs: %q %q %v", pinned, setFP, err)
	}
	if outcome.Sets[0].Members[0].Locator == outcome.Sets[1].Members[0].Locator {
		t.Fatal("the two builds were not written to distinct directories")
	}

	// 閉包の十分性: 閉包のファイルだけの複製でbuildでき、同じbyteになる。
	// 閉包から1件欠けるとここでgo buildが失敗する。
	copyRoot := t.TempDir()
	copyClosureTo(t, outcome.Closure, repoRoot, copyRoot)
	copyOutcome := bindRealGoBuild(realBuildSpec{
		Subject: "cmd/tei(closure-copy)", GoTool: goTool, RepoRoot: copyRoot, ModuleDir: copyRoot,
		Package: "./cmd/tei", Entrypoint: "cmd/tei", OutDir: t.TempDir(),
	})
	if copyOutcome.Report.Status != realBuildStatusBound {
		t.Fatalf("closure-only copy did not build: %+v", copyOutcome.Report)
	}
	if copyOutcome.Report.ArtifactDigest != report.ArtifactDigest {
		t.Fatalf("closure-only copy yielded a different artifact: %s vs %s",
			copyOutcome.Report.ArtifactDigest, report.ArtifactDigest)
	}
	if copyOutcome.Report.SourceClosureCount != report.SourceClosureCount {
		t.Fatalf("closure size differs in the copy: %d vs %d",
			copyOutcome.Report.SourceClosureCount, report.SourceClosureCount)
	}

	// 構造化Evidence。明記すべき限定を含み、往復で同一になる。
	serialized, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "real-tei-cli-build.json")
	if err := os.WriteFile(path, serialized, 0o600); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var replay realBuildReport
	if err := json.Unmarshal(saved, &replay); err != nil {
		t.Fatal(err)
	}
	if replay.NotClaimedStatement != realBuildNotClaimedStatement ||
		!strings.Contains(string(saved), "Stage収束とseed信頼は未主張") ||
		replay.ArtifactDigest != report.ArtifactDigest ||
		replay.SourceClosureCount != report.SourceClosureCount {
		t.Fatalf("evidence json lost required fields: %s", saved)
	}
	t.Logf("real cmd/tei binding evidence:\n%s", saved)
}
