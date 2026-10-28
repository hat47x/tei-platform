package pluginbuildsupply

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// The through tests need a clean checkout of the committed branch and the
// exact Go executable; tools/verify-plugin-build-supply.py provides both.
const (
	envRepo   = "TEI_PBS_REPO"
	envGo     = "TEI_PBS_GO"
	envOutDir = "TEI_PBS_OUT_DIR"
)

func TestMain(m *testing.M) {
	if os.Getenv(EnvRole) == RoleBuilder {
		os.Exit(BuilderMain(os.Args[1:], os.Stdout, os.Stderr))
	}
	code := m.Run()
	if shared.work != "" {
		os.RemoveAll(shared.work)
	}
	os.Exit(code)
}

func observed(t *testing.T, name string, v any) {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"test": t.Name(), "name": name, "value": v})
	t.Logf("observed %s", raw)
	if dir := os.Getenv(envOutDir); dir != "" {
		f, err := os.OpenFile(filepath.Join(dir, "observed.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			f.Write(append(raw, '\n'))
			f.Close()
		}
	}
}

type environment struct {
	repo, goPath, work string
	tools              map[string]string
}

func requireEnvironment(t *testing.T) environment {
	t.Helper()
	repo, goPath := os.Getenv(envRepo), os.Getenv(envGo)
	if repo == "" || goPath == "" {
		t.Skipf("%s and %s are required; run tools/verify-plugin-build-supply.py", envRepo, envGo)
	}
	if runtime.GOOS != "linux" {
		t.Fatalf("the through tests run on Linux (process groups, stdio plugin launch); got %s", runtime.GOOS)
	}
	shared.envOnce.Do(func() {
		shared.env, shared.envErr = prepareEnvironment(repo, goPath)
	})
	if shared.envErr != nil {
		t.Fatal(shared.envErr)
	}
	return shared.env
}

// prepareEnvironment builds the existing CLIs once, unchanged, from the
// clean checkout. No module or toolchain is fetched.
func prepareEnvironment(repo, goPath string) (environment, error) {
	work, err := os.MkdirTemp("", "tei-pbs-work-")
	if err != nil {
		return environment{}, err
	}
	shared.work = work
	env := environment{repo: repo, goPath: goPath, work: work, tools: map[string]string{}}
	for name, pkg := range map[string]string{
		"self-build":          "./tools/reference-plugin-self-build",
		"authoring-intent":    "./tools/reference-build-authoring-intent",
		"contract-resolution": "./tools/reference-builder-contract-resolution",
		"attempt-evidence":    "./tools/reference-build-attempt-evidence",
		"artifact-store":      "./tools/reference-build-artifact-store",
	} {
		out := filepath.Join(work, "tools", name)
		cmd := exec.Command(goPath, "build", "-o", out, pkg)
		cmd.Dir = repo
		cmd.Env = append(withoutKeys(os.Environ(), "GOFLAGS", "GOPROXY", "GOTOOLCHAIN", "GOWORK"),
			"GOFLAGS=-mod=readonly", "GOPROXY=off", "GOTOOLCHAIN=local", "GOWORK=off")
		if raw, err := cmd.CombinedOutput(); err != nil {
			return environment{}, fmt.Errorf("build %s: %v: %s", pkg, err, raw)
		}
		env.tools[name] = out
	}
	return env, nil
}

func withoutKeys(env []string, keys ...string) []string {
	out := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		drop := false
		for _, k := range keys {
			if strings.EqualFold(key, k) {
				drop = true
			}
		}
		if !drop {
			out = append(out, entry)
		}
	}
	return out
}

func runTool(t *testing.T, path string, args ...string) (int, string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return code, stdout.String(), stderr.String()
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// copyRepository makes an independent repository from the committed tree,
// applies edit and, when commit is true, commits the result. git archive and
// a local commit need no network.
func copyRepository(t *testing.T, env environment, name string, edit func(dir string), commit bool) string {
	t.Helper()
	dir := filepath.Join(env.work, "copy-"+name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := exec.Command("sh", "-c", `git -C "$1" archive HEAD | tar -x -C "$2"`, "sh", env.repo, dir)
	if raw, err := archive.CombinedOutput(); err != nil {
		t.Fatalf("copy repository: %v: %s", err, raw)
	}
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=tei-pbs", "-c", "user.email=tei-pbs@invalid", "-c", "commit.gpgsign=false"}, args...)...)
		if raw, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, raw)
		}
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-q", "-m", "copy")
	if edit != nil {
		edit(dir)
	}
	if commit {
		git("add", "-A")
		git("commit", "-q", "-m", "edit")
	}
	return dir
}

var shared struct {
	envOnce sync.Once
	env     environment
	envErr  error
	work    string

	throughOnce sync.Once
	through     *throughRun
	throughErr  error
}
