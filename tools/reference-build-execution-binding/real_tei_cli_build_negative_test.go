package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/plugin"
)

// (a) 閉包の外のSourceを足す / 閉包内のSourceを欠落させると、
// Plan指紋が変わり、観測した閉包との照合でも拒否される。
func TestRealTEICLIBuildRejectsPlanSourceDriftFromClosure(t *testing.T) {
	goTool := requireRealGo(t)
	repoRoot := realBuildFindRepoRoot(t)
	env := realBuildEnv()
	closure, err := enumerateGoSourceClosure(goTool, repoRoot, repoRoot, "./cmd/tei", env)
	if err != nil {
		t.Fatalf("baseline closure is not enumerable: %v", err)
	}
	digest, err := actualGoDigestFile(goTool)
	if err != nil {
		t.Fatal(err)
	}
	version, err := realBuildToolchainVersion(goTool, env)
	if err != nil {
		t.Fatal(err)
	}
	goEnv, err := realBuildObserveGoEnv(goTool, env)
	if err != nil {
		t.Fatal(err)
	}
	base := newRealBuildPlan("cmd/tei", version, digest, goEnv, closure)
	baseFP, err := buildExecutionPlanFingerprint(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyPlanSourcesMatchClosure(base, closure); err != nil {
		t.Fatalf("baseline plan must match its closure: %v", err)
	}

	withPlan := func(mutate func([]plugin.BuildSource) []plugin.BuildSource) BuildExecutionPlan {
		p := base
		p.Sources = mutate(append([]plugin.BuildSource(nil), base.Sources...))
		return p
	}
	studioMain := filepath.Join(repoRoot, "cmd", "tei-studio", "main.go")
	studioBytes, err := os.ReadFile(studioMain)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]BuildExecutionPlan{
		"outside-closure-file-added": withPlan(func(s []plugin.BuildSource) []plugin.BuildSource {
			return append(s, plugin.BuildSource{
				Kind: "go-source", ID: "cmd/tei-studio/main.go",
				Locator: studioMain, Digest: realBuildDigestBytes(studioBytes),
			})
		}),
		"closure-member-dropped": withPlan(func(s []plugin.BuildSource) []plugin.BuildSource {
			for i := range s {
				if s[i].ID == "cmd/tei/main.go" {
					return append(s[:i], s[i+1:]...)
				}
			}
			t.Fatal("cmd/tei/main.go not found in the plan")
			return s
		}),
		"go-sum-dropped": withPlan(func(s []plugin.BuildSource) []plugin.BuildSource {
			for i := range s {
				if s[i].ID == "go.sum" {
					return append(s[:i], s[i+1:]...)
				}
			}
			t.Fatal("go.sum not found in the plan")
			return s
		}),
		"member-digest-changed": withPlan(func(s []plugin.BuildSource) []plugin.BuildSource {
			s[0].Digest = realBuildDigestBytes([]byte("tampered"))
			return s
		}),
	}
	for name, plan := range cases {
		fp, err := buildExecutionPlanFingerprint(plan)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if fp == baseFP {
			t.Errorf("%s: plan fingerprint did not change", name)
		}
		if err := verifyPlanSourcesMatchClosure(plan, closure); err == nil {
			t.Errorf("%s: plan was accepted against the observed closure", name)
		}
	}
}

// (b) docs/だけの変更では、Source集合もArtifact digestも変わらない。
func TestRealTEICLIBuildIgnoresDocsOnlyChange(t *testing.T) {
	goTool := requireRealGo(t)
	repoRoot := realBuildFindRepoRoot(t)
	env := realBuildEnv()
	closure, err := enumerateGoSourceClosure(goTool, repoRoot, repoRoot, "./cmd/tei", env)
	if err != nil {
		t.Fatalf("baseline closure is not enumerable: %v", err)
	}
	copyRoot := t.TempDir()
	copyClosureTo(t, closure, repoRoot, copyRoot)
	docs := filepath.Join(copyRoot, "docs", "real-build-binding-probe.md")
	spec := realBuildSpec{
		Subject: "cmd/tei(docs-probe)", GoTool: goTool, RepoRoot: copyRoot, ModuleDir: copyRoot,
		Package: "./cmd/tei", Entrypoint: "cmd/tei",
	}
	write := func(body string) {
		if err := os.MkdirAll(filepath.Dir(docs), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(docs, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("first docs text\n")
	spec.OutDir = t.TempDir()
	before := bindRealGoBuild(spec)
	write("second, different docs text\n")
	if err := os.WriteFile(filepath.Join(copyRoot, "docs", "another.md"), []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	spec.OutDir = t.TempDir()
	after := bindRealGoBuild(spec)
	for _, o := range []realBuildOutcome{before, after} {
		if o.Report.Status != realBuildStatusBound {
			t.Fatalf("docs probe build was not bound: %+v", o.Report)
		}
		for _, f := range o.Closure.Files {
			if strings.HasPrefix(f.Rel, "docs/") {
				t.Fatalf("docs file entered the source closure: %s", f.Rel)
			}
		}
	}
	if before.Report.PlanFingerprint != after.Report.PlanFingerprint ||
		before.Report.ArtifactDigest != after.Report.ArtifactDigest ||
		before.Report.SourceClosureCount != after.Report.SourceClosureCount {
		t.Fatalf("docs-only change moved the Source set or Artifact: %+v vs %+v", before.Report, after.Report)
	}
}

// (c) 同じ手順をnested module(cmd/tei-studio)へ適用する。go.sumを欠いた
// 一時コピーではgo buildが失敗し、成功と偽らずBlocked/未確定にする。
// rootが成功したことはnested moduleの成功の根拠にならない。
func TestRealTEICLIBuildNestedModuleWithoutGoSumIsBlocked(t *testing.T) {
	goTool := requireRealGo(t)
	repoRoot := realBuildFindRepoRoot(t)
	env := realBuildEnv()
	studioDir := filepath.Join(repoRoot, "cmd", "tei-studio")
	closure, err := enumerateGoSourceClosure(goTool, repoRoot, studioDir, ".", env)
	if err != nil {
		realBuildSkipOrFail(t, "nested module closure is not enumerable here: %v", err)
	}
	for _, must := range []string{"cmd/tei-studio/go.mod", "cmd/tei-studio/go.sum", "go.mod"} {
		if _, ok := closure.byRel()[must]; !ok {
			t.Fatalf("nested closure lacks %s", must)
		}
	}

	copyRoot := t.TempDir()
	copyClosureTo(t, closure, repoRoot, copyRoot)
	copyStudio := filepath.Join(copyRoot, "cmd", "tei-studio")
	if err := os.Remove(filepath.Join(copyStudio, "go.sum")); err != nil {
		t.Fatal(err)
	}

	// go buildの失敗そのものを直接観測する。
	probe := exec.Command(goTool, "build", "-trimpath", "-buildvcs=false", "-ldflags=-buildid=",
		"-o", filepath.Join(t.TempDir(), "studio-probe"+exeSuffix()), ".")
	probe.Dir, probe.Env = copyStudio, env
	out, err := probe.CombinedOutput()
	if err == nil {
		t.Fatalf("go build unexpectedly succeeded without go.sum: %s", out)
	}
	if !strings.Contains(string(out), "go.sum") {
		t.Fatalf("go build failed, but not because of go.sum: %s", out)
	}

	outcome := bindRealGoBuild(realBuildSpec{
		Subject: "cmd/tei-studio(no-go.sum)", GoTool: goTool, RepoRoot: copyRoot,
		ModuleDir: copyStudio, Package: ".", Entrypoint: "cmd/tei-studio", OutDir: t.TempDir(),
	})
	r := outcome.Report
	if r.Status != realBuildStatusBlocked && r.Status != realBuildStatusUnconfirmed {
		t.Fatalf("missing go.sum was not reported as blocked/unconfirmed: %+v", r)
	}
	if r.Reproducible || r.ArtifactDigest != "" || r.ResultFingerprint != "" || r.Reason == "" ||
		len(outcome.Sets) != 0 {
		t.Fatalf("a failed nested build carries success evidence: %+v", r)
	}
	if r.NotClaimedStatement != realBuildNotClaimedStatement {
		t.Fatal("blocked report lost the not-claimed statement")
	}
}

// (d) 閉包を列挙できない場合は成功にせず、未確定を返す。
func TestRealTEICLIBuildUnenumerableClosureIsUnconfirmed(t *testing.T) {
	repoRoot := realBuildFindRepoRoot(t)
	check := func(name string, spec realBuildSpec) {
		t.Helper()
		o := bindRealGoBuild(spec)
		r := o.Report
		if r.Status != realBuildStatusUnconfirmed {
			t.Fatalf("%s: expected unconfirmed, got %+v", name, r)
		}
		if r.Reproducible || r.ArtifactDigest != "" || r.PlanFingerprint != "" ||
			r.ResultFingerprint != "" || len(o.Sets) != 0 || r.Reason == "" {
			t.Fatalf("%s: unconfirmed report carries success evidence: %+v", name, r)
		}
	}
	// goが実行できない(Go不要の確認)。
	check("missing go executable", realBuildSpec{
		Subject: "cmd/tei", GoTool: filepath.Join(t.TempDir(), "no-such-go"+exeSuffix()),
		RepoRoot: repoRoot, ModuleDir: repoRoot, Package: "./cmd/tei",
		Entrypoint: "cmd/tei", OutDir: t.TempDir(),
	})
	// go listが失敗する(存在しないpackage)。
	goTool := requireRealGo(t)
	check("go list failure", realBuildSpec{
		Subject: "cmd/does-not-exist", GoTool: goTool, RepoRoot: repoRoot, ModuleDir: repoRoot,
		Package: "./cmd/does-not-exist", Entrypoint: "cmd/does-not-exist", OutDir: t.TempDir(),
	})
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
