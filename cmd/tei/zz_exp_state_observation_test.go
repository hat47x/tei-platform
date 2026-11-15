package main

// EXPERIMENT ONLY (TASK 6). Not for merge as-is.
//
// Runs the L2-b end-to-end attempt several times: with the reference child, and
// with a wrapper that redirects -state to one shared directory and writes the
// passed value back into the child's report. It then compares three judges:
//
//	B0: the entry's own l2b.claimed (child statement + artifact digest)
//	B1: a post-hoc look at the state directory the entry passed
//	B2: the child runs inside a user+mount namespace chosen by the parent side,
//	    where only the selected state and artifact directories are writable
//
// Gated by TEI_EXP_OUT so it never runs in the normal suite.

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const expWrapperSource = `package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

func main() {
	exe, _ := os.Executable()
	raw, _ := os.ReadFile(exe + ".conf")
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	realChild, shared, decoy := lines[0], lines[1], lines[2] == "decoy"
	args := append([]string{}, os.Args[1:]...)
	passed := ""
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-state" {
			passed = args[i+1]
			args[i+1] = shared
		}
	}
	if decoy {
		_ = os.MkdirAll(passed, 0o755)
	}
	cmd := exec.Command(realChild, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	os.Stdout.Write(bytes.ReplaceAll(out.Bytes(), []byte(shared), []byte(passed)))
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			os.Exit(e.ExitCode())
		}
		os.Exit(1)
	}
}
`

// The launcher stands in for the entry's own launching code: the parent side
// chooses it, the child does not. It runs the child in a fresh user+mount
// namespace where only the selected state directory and the artifact's directory
// are writable.
const expLauncherSource = `#!/bin/bash
target=$(sed -n 1p "$0.conf")
log=$(sed -n 2p "$0.conf")
state=""; art=""
args=("$@")
for ((i=0;i<${#args[@]};i++)); do
  case "${args[i]}" in
    -state) state="${args[i+1]}";;
    -artifact-out) art="${args[i+1]}";;
  esac
done
artdir=$(dirname "$art")
mkdir -p "$state" "$artdir"
unshare -rm --propagation private bash -c '
  state="$1"; artdir="$2"; shift 2
  mount --bind "$state" "$state" && mount --bind "$artdir" "$artdir" \
    && mount -o remount,bind,ro / && mount -o remount,bind,ro /mnt/c || exit 97
  exec "$@"' _ "$state" "$artdir" "$target" "$@"
rc=$?
echo "boundary-run state=$state artdir=$artdir rc=$rc" >> "$log"
exit $rc
`

type expCase struct {
	Name            string   `json:"name"`
	Boundary        string   `json:"boundary"`
	ExitCode        int      `json:"entryExitCode"`
	B0Claimed       bool     `json:"l2bClaimed"`
	B0Unmet         []string `json:"l2bUnmet,omitempty"`
	B0Repro         string   `json:"artifactReproducibility"`
	ReproReason     string   `json:"reproducibilityUnperformedReason,omitempty"`
	ErrorCode       string   `json:"errorCode,omitempty"`
	ErrorMessage    string   `json:"errorMessage,omitempty"`
	B1FirstStateDir string   `json:"b1_firstStateDirPostHoc"`
	B1SecondState   string   `json:"b1_secondStateDirPostHoc"`
	SharedDir       string   `json:"sharedDirAfterwards"`
	BoundaryLog     []string `json:"boundaryLog,omitempty"`
}

func expRunCase(t *testing.T, name, outDir, cli, realChild, resolver, wrapper string, boundary bool) expCase {
	root, intentPath, sourceRoot, _ := setupL2BFixture(t)
	workspace := outerWorkspace(t, root)
	shared := filepath.Join(t.TempDir(), "SHARED-state") // outside root on purpose

	child := realChild
	if wrapper != "" {
		raw, err := os.ReadFile(wrapper)
		if err != nil {
			t.Fatal(err)
		}
		child = filepath.Join(root, "wrapper-child")
		if err := os.WriteFile(child, raw, 0o755); err != nil {
			t.Fatal(err)
		}
		mode := "plain"
		if strings.Contains(name, "decoy") {
			mode = "decoy"
		}
		if err := os.WriteFile(child+".conf", []byte(realChild+"\n"+shared+"\n"+mode+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	boundaryLog := filepath.Join(outDir, name+".boundary.log")
	if boundary {
		launcher := filepath.Join(root, "launcher")
		if err := os.WriteFile(launcher, []byte(expLauncherSource), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(launcher+".conf", []byte(child+"\n"+boundaryLog+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		child = launcher
	}
	catalogPath, catalogDigest := authorCatalog(t, root, sourceRoot, child)

	planPath := filepath.Join(root, "plan.json")
	resolveOut, err := exec.Command(resolver, "-intent", intentPath, "-catalog", catalogPath, "-source-root", sourceRoot).Output()
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	var receipt struct {
		Plan json.RawMessage `json:"plan"`
	}
	if err := json.Unmarshal(resolveOut, &receipt); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, append(receipt.Plan, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	childArg, _ := json.Marshal([]string{"-plan", planPath, "-plugin", filepath.Join(sourceRoot, "plugin.tei"), "-intent", intentPath})

	firstState := filepath.Join(root, "state")
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(cli, "self-build",
		"-workspace", workspace,
		"-intent", "tei.plugin.l2b_e2e",
		"-intent-file", intentPath,
		"-build-catalog", catalogPath,
		"-build-catalog-digest", catalogDigest,
		"-build-source-root", sourceRoot,
		"-child", child,
		"-child-arg", string(childArg),
		"-state", firstState,
		"-output", filepath.Join(root, "package.json"),
	)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	_ = os.WriteFile(filepath.Join(outDir, name+".report.json"), stdout.Bytes(), 0o600)
	_ = os.WriteFile(filepath.Join(outDir, name+".stderr.txt"), stderr.Bytes(), 0o600)

	result := expCase{Name: name, Boundary: "none"}
	if boundary {
		result.Boundary = "user+mount namespace; writable: selected state, artifact directory"
	}
	if exit, ok := runErr.(*exec.ExitError); ok {
		result.ExitCode = exit.ExitCode()
	}
	var report l2bEndToEndReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("%s: decode report: %v\n%s\n%s", name, err, stdout.String(), stderr.String())
	}
	if report.L2B != nil {
		result.B0Claimed = report.L2B.Claimed
		result.B0Unmet = report.L2B.Unmet
	}
	if report.ArtifactReproducibility != nil {
		result.B0Repro = report.ArtifactReproducibility.State
		result.ReproReason = report.ArtifactReproducibility.UnperformedReason
	}
	if report.Error != nil {
		result.ErrorCode, result.ErrorMessage = report.Error.Code, report.Error.Message
	}
	result.B1FirstStateDir = "absent"
	if info, err := os.Stat(firstState); err == nil && info.IsDir() {
		result.B1FirstStateDir = "exists"
	}
	// The entry removes the second state directory before it returns, so nothing
	// outside the entry can look at it afterwards.
	result.B1SecondState = "NOT RUN (entry deletes it before returning)"
	result.SharedDir = "absent"
	if _, err := os.Stat(shared); err == nil {
		result.SharedDir = "exists"
	}
	if raw, err := os.ReadFile(boundaryLog); err == nil {
		result.BoundaryLog = strings.Split(strings.TrimSpace(string(raw)), "\n")
	}
	return result
}

func TestExpStateObservation(t *testing.T) {
	outDir := os.Getenv("TEI_EXP_OUT")
	if outDir == "" {
		t.Skip("experiment; set TEI_EXP_OUT")
	}
	if err := exec.Command("unshare", "-rm", "true").Run(); err != nil {
		t.Skipf("unprivileged user+mount namespaces unavailable: boundary NOT RUN (%v)", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cli := buildToolBinary(t, ".", "tei")
	realChild := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-plugin-package-build-child"), "build-child")
	resolver := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-build-authoring-intent"), "resolver")

	wrapperDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(wrapperDir, "main.go"), []byte(expWrapperSource), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wrapperDir, "go.mod"), []byte("module expwrapper\n\ngo 1.21\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(wrapperDir, "wrapper")
	build := exec.Command("go", "build", "-o", wrapper, ".")
	build.Dir = wrapperDir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build wrapper: %v\n%s", err, out)
	}

	cases := []expCase{
		expRunCase(t, "P0-reference-child", outDir, cli, realChild, resolver, "", false),
		expRunCase(t, "N0-shared-state", outDir, cli, realChild, resolver, wrapper, false),
		expRunCase(t, "N0-shared-state-decoy", outDir, cli, realChild, resolver, wrapper, false),
		expRunCase(t, "P2-reference-child-in-boundary", outDir, cli, realChild, resolver, "", true),
		expRunCase(t, "N2-shared-state-in-boundary", outDir, cli, realChild, resolver, wrapper, true),
		expRunCase(t, "N2-shared-state-decoy-in-boundary", outDir, cli, realChild, resolver, wrapper, true),
	}
	summary, _ := json.MarshalIndent(cases, "", "  ")
	_ = os.WriteFile(filepath.Join(outDir, "summary.json"), summary, 0o600)
	t.Logf("\n%s", summary)
}
