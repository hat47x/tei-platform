package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/plugin"
)

func realBuildEnvMap(env []string) map[string][]string {
	out := map[string][]string{}
	for _, kv := range env {
		parts := strings.SplitN(kv, "=", 2)
		out[strings.ToUpper(parts[0])] = append(out[strings.ToUpper(parts[0])], parts[1])
	}
	return out
}

// buildへ渡す環境は許可リストで、呼び出し側の出力影響設定を通さない。
func TestRealTEICLIBuildEnvIsAllowlisted(t *testing.T) {
	polluted := map[string]string{
		"GOAMD64": "v3", "GOARM64": "v8.1", "GOEXPERIMENT": "boringcrypto", "GODEBUG": "x=1",
		"GOENV": filepath.Join(t.TempDir(), "env"), "GOFIPS140": "latest", "GOFLAGS": "-race",
		"GOCACHEPROG": "evil", "GO386": "softfloat", "TEI_UNRELATED_SECRET": "leak",
	}
	for k, v := range polluted {
		t.Setenv(k, v)
	}
	got := realBuildEnvMap(realBuildEnv())
	for k, want := range realBuildPinnedEnv() {
		vals := got[k]
		if len(vals) != 1 || vals[0] != want {
			t.Errorf("%s = %v, want exactly %q", k, vals, want)
		}
	}
	if _, leaked := got["TEI_UNRELATED_SECRET"]; leaked {
		t.Error("an unrelated environment variable was passed to the build")
	}
	if len(got["PATH"]) != 1 {
		t.Error("PATH must be passed through")
	}
}

// 実環境に足したGOAMD64等はbuildへ漏れない。漏れるならPlan指紋が変わる。
func TestRealTEICLIBuildGoEnvInputsAreBoundToPlan(t *testing.T) {
	goTool := requireRealGo(t)
	repoRoot := realBuildFindRepoRoot(t)
	spec := func() realBuildSpec {
		return realBuildSpec{
			Subject: "cmd/tei", GoTool: goTool, RepoRoot: repoRoot, ModuleDir: repoRoot,
			Package: "./cmd/tei", Entrypoint: "cmd/tei", OutDir: t.TempDir(),
		}
	}
	base := bindRealGoBuild(spec())
	if base.Report.Status != realBuildStatusBound {
		t.Fatalf("baseline not bound: %+v", base.Report)
	}
	t.Setenv("GOAMD64", "v3")
	t.Setenv("GOARM64", "v8.1")
	t.Setenv("GOEXPERIMENT", "nocoverageredesign")
	t.Setenv("GODEBUG", "madvdontneed=1")
	polluted := bindRealGoBuild(spec())
	if polluted.Report.Status != realBuildStatusBound {
		t.Fatalf("polluted-env build not bound: %+v", polluted.Report)
	}
	if polluted.Report.PlanFingerprint != base.Report.PlanFingerprint ||
		polluted.Report.ArtifactDigest != base.Report.ArtifactDigest {
		t.Fatalf("caller GOAMD64/GOEXPERIMENT leaked into the build: %+v vs %+v", polluted.Report, base.Report)
	}
	// Planはgo envの観測値を持つ。値が変わればPlan指紋も変わる。
	params := map[string]string{}
	for _, p := range base.Plan.Parameters {
		params[p.Name] = p.Value
	}
	for _, k := range []string{"goenv.GOAMD64", "goenv.GOEXPERIMENT", "goenv.GODEBUG", "goenv.GOFIPS140", "goenv.GOENV"} {
		if _, ok := params[k]; !ok {
			t.Fatalf("plan lacks %s", k)
		}
	}
	changed := base.Plan
	changed.Parameters = append([]plugin.BuildParameter(nil), base.Plan.Parameters...)
	for i := range changed.Parameters {
		if changed.Parameters[i].Name == "goenv.GOAMD64" {
			changed.Parameters[i].Value = "v3"
		}
	}
	fp, err := buildExecutionPlanFingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if fp == base.Report.PlanFingerprint {
		t.Fatal("a changed go env value did not change the plan fingerprint")
	}
}

func independentFileDigest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Toolchain digestは別経路で計算した値と一致し、goの1byte変更でPlan指紋が変わる。
// 2回目のbuildは実際に走り、Planのflagsは実build引数と一致する。
func TestRealTEICLIBuildToolchainAndTwoBuildsAreIndependentlyChecked(t *testing.T) {
	goTool := requireRealGo(t)
	repoRoot := realBuildFindRepoRoot(t)
	type call struct {
		args   []string
		output string
		exists bool
	}
	var calls []call
	spec := realBuildSpec{
		Subject: "cmd/tei", GoTool: goTool, RepoRoot: repoRoot, ModuleDir: repoRoot,
		Package: "./cmd/tei", Entrypoint: "cmd/tei", OutDir: t.TempDir(),
		Exec: func(cmd *exec.Cmd) ([]byte, error) {
			out, err := cmd.CombinedOutput()
			c := call{args: append([]string(nil), cmd.Args[1:]...)}
			for i, a := range c.args {
				if a == "-o" && i+1 < len(c.args) {
					c.output = c.args[i+1]
				}
			}
			_, statErr := os.Stat(c.output)
			c.exists = statErr == nil
			calls = append(calls, c)
			return out, err
		},
	}
	o := bindRealGoBuild(spec)
	if o.Report.Status != realBuildStatusBound {
		t.Fatalf("not bound: %+v", o.Report)
	}

	// 2回のbuildが実際に走り、別々の出力を作った。
	if len(calls) != 2 {
		t.Fatalf("expected 2 go build invocations, got %d", len(calls))
	}
	if calls[0].output == "" || calls[0].output == calls[1].output || !calls[0].exists || !calls[1].exists {
		t.Fatalf("builds did not produce distinct outputs: %+v", calls)
	}
	if o.Sets[0].Members[0].Locator != calls[0].output || o.Sets[1].Members[0].Locator != calls[1].output {
		t.Fatal("artifact members do not point at the invoked build outputs")
	}

	// Planのflagsは実build引数と一致し、必要なflagが実際に渡っている(独立の期待値)。
	wantFlags := []string{"-trimpath", "-buildvcs=false", "-ldflags=-buildid="}
	planFlags := ""
	for _, p := range o.Plan.Parameters {
		if p.Name == "flags" {
			planFlags = p.Value
		}
	}
	for _, c := range calls {
		if c.args[0] != "build" {
			t.Fatalf("unexpected command: %v", c.args)
		}
		oIdx := -1
		for i, a := range c.args {
			if a == "-o" {
				oIdx = i
			}
		}
		if oIdx < 0 || strings.Join(c.args[1:oIdx], " ") != planFlags {
			t.Fatalf("plan flags %q differ from build args %v", planFlags, c.args)
		}
	}
	if planFlags != strings.Join(wantFlags, " ") {
		t.Fatalf("flags = %q, want %q", planFlags, strings.Join(wantFlags, " "))
	}

	// Toolchain digestはgo実行ファイルを別経路で読んだ値と一致する。
	want := independentFileDigest(t, goTool)
	if o.Report.ToolchainDigest != want || o.Plan.Builder.ArtifactDigest != want {
		t.Fatalf("toolchain digest %q / builder %q differ from independent %q",
			o.Report.ToolchainDigest, o.Plan.Builder.ArtifactDigest, want)
	}
	params := map[string]string{}
	for _, p := range o.Plan.Parameters {
		params[p.Name] = p.Value
	}
	if params["go-executable-digest"] != want {
		t.Fatal("plan parameter go-executable-digest is not the go executable digest")
	}

	// goを1byte変えた複製はdigestとPlan指紋を変える。
	data, err := os.ReadFile(goTool)
	if err != nil {
		t.Fatal(err)
	}
	tampered := filepath.Join(t.TempDir(), "go-tampered"+exeSuffix())
	if err := os.WriteFile(tampered, append(data, 0), 0o700); err != nil {
		t.Fatal(err)
	}
	tamperedDigest, err := actualGoDigestFile(tampered)
	if err != nil || tamperedDigest == want {
		t.Fatalf("tampered go has the same digest: %q %v", tamperedDigest, err)
	}
	goEnv, err := realBuildObserveGoEnv(goTool, realBuildEnv())
	if err != nil {
		t.Fatal(err)
	}
	tamperedPlan := newRealBuildPlan("cmd/tei", o.Report.ToolchainVersion, tamperedDigest, goEnv, o.Closure)
	tamperedFP, err := buildExecutionPlanFingerprint(tamperedPlan)
	if err != nil || tamperedFP == o.Report.PlanFingerprint {
		t.Fatalf("tampered toolchain did not change the plan fingerprint: %v", err)
	}
}
