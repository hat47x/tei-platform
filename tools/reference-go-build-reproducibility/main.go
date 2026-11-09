// reference-go-build-reproducibility は、同じ対象を隔離した状態で二回実際に
// Buildし、成果物の byte digest と、試行の同一性を別々に記録します
// （GitHub #2831 SB-C の前提確認）。
//
// この参照はBuildを実行しますが、artifactの受理・署名・権限・配備は行いません。
// 「二回とも成功し digest が一致した」ことは再現性の観測であり、Builderの
// 信頼でも、Runtime Authority でもありません。
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"tei/tools/internal/gobuilder"
)

const reproducibilitySchema = "tei.reference.go-build-reproducibility/v0"

const claimBoundary = "two isolated build attempts on one machine and one toolchain; digest agreement is an observation of reproducibility, not Builder trust, acceptance, or runtime authority"

// Attempt is one isolated build. AttemptID is generated per invocation and
// never derived from the artifact digest, so equal bytes cannot stand in for
// a second run.
type Attempt struct {
	AttemptID          string   `json:"attemptId"`
	StateDir           string   `json:"stateDir"`
	ExitCode           int      `json:"exitCode"`
	Error              string   `json:"error,omitempty"`
	Succeeded          bool     `json:"succeeded"`
	ArtifactDigest     string   `json:"artifactDigest,omitempty"`
	GoExecutableDigest string   `json:"goExecutableDigest"`
	BuildCommand       []string `json:"buildCommand"`
}

// Reproducibility is the evidence record for the two attempts.
type Reproducibility struct {
	SchemaVersion    string    `json:"schemaVersion"`
	Nature           string    `json:"nature"`
	Package          string    `json:"package"`
	GoOS             string    `json:"goos"`
	GoArch           string    `json:"goarch"`
	GoVersion        string    `json:"goVersion"`
	SupplyRoot       string    `json:"supplyRoot"`
	Attempts         []Attempt `json:"attempts"`
	BothSucceeded    bool      `json:"bothAttemptsSucceeded"`
	DigestsEqual     bool      `json:"artifactDigestsEqual"`
	DistinctAttempts bool      `json:"attemptIdentitiesDistinct"`
	Reproducible     bool      `json:"reproducible"`
	FailureCause     string    `json:"failureCause,omitempty"`
	Boundary         string    `json:"claimBoundary"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reference-go-build-reproducibility", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", ".", "対象リポジトリのcheckout")
	pkg := fs.String("package", "", "Buildするpackage（必須）")
	supply := fs.String("supply", "", "GOPROXY=fileで使う供給ディレクトリ（必須）")
	goExecutable := fs.String("go", "", "exact Go実行ファイル（省略時はPATHのgo）")
	stateRoot := fs.String("state-root", "", "二つの試行用state親ディレクトリ（省略時は一時ディレクトリ）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *pkg == "" || *supply == "" {
		fmt.Fprintln(stderr, "-package and -supply are required")
		return 2
	}
	root := *stateRoot
	cleanup := func() {}
	if root == "" {
		dir, err := os.MkdirTemp("", "tei-repro-*")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		root = dir
		cleanup = func() { removeWritable(dir) }
	}
	defer cleanup()

	evidence, err := Run(*repo, *pkg, *supply, *goExecutable, root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(evidence); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// Run performs two independent builds with distinct isolated state and
// distinct attempt identities, then compares only the artifact bytes.
func Run(repo, packagePath, supply, goArg, stateRoot string) (Reproducibility, error) {
	exactGo, err := resolveGo(goArg)
	if err != nil {
		return Reproducibility{}, err
	}
	goDigest, err := fileDigest(exactGo)
	if err != nil {
		return Reproducibility{}, err
	}
	goVersion, err := commandOutput(exactGo, "version")
	if err != nil {
		return Reproducibility{}, err
	}
	exactSupply, err := filepath.EvalSymlinks(mustAbs(supply))
	if err != nil {
		return Reproducibility{}, fmt.Errorf("supply root is not readable: %w", err)
	}
	evidence := Reproducibility{
		SchemaVersion: reproducibilitySchema,
		Nature:        "two isolated real builds on one machine; agreement is not trust or acceptance",
		Package:       packagePath,
		GoOS:          runtime.GOOS,
		GoArch:        runtime.GOARCH,
		GoVersion:     goVersion,
		SupplyRoot:    exactSupply,
		Boundary:      claimBoundary,
	}
	for ordinal := 1; ordinal <= 2; ordinal++ {
		state := filepath.Join(stateRoot, fmt.Sprintf("attempt-%d", ordinal))
		output := filepath.Join(state, "artifact")
		attemptID, err := randomID()
		if err != nil {
			return Reproducibility{}, err
		}
		attempt := Attempt{
			AttemptID:          attemptID,
			StateDir:           state,
			GoExecutableDigest: goDigest,
			BuildCommand: []string{
				exactGo, "build", "-trimpath", "-buildvcs=false", "-mod=readonly", "-o", "<artifact>", packagePath,
			},
		}
		exitCode, buildError, err := buildOnce(repo, exactGo, packagePath, supply, output, state)
		if err != nil {
			return Reproducibility{}, err
		}
		attempt.ExitCode = exitCode
		attempt.Error = buildError
		if exitCode == 0 {
			attempt.Succeeded = true
			digest, err := fileDigest(output)
			if err != nil {
				return Reproducibility{}, err
			}
			attempt.ArtifactDigest = digest
		}
		evidence.Attempts = append(evidence.Attempts, attempt)
	}
	if len(evidence.Attempts) != 2 {
		return Reproducibility{}, fmt.Errorf("expected exactly two attempts, produced %d", len(evidence.Attempts))
	}
	first, second := evidence.Attempts[0], evidence.Attempts[1]
	evidence.BothSucceeded = first.Succeeded && second.Succeeded
	evidence.DistinctAttempts = first.AttemptID != second.AttemptID
	switch {
	case !evidence.BothSucceeded:
		evidence.FailureCause = "at least one attempt did not produce an artifact"
	case !evidence.DistinctAttempts:
		evidence.FailureCause = "attempt identities were not distinct, so two runs cannot be claimed"
	case first.ArtifactDigest != second.ArtifactDigest:
		evidence.FailureCause = "artifact bytes differed between the two attempts"
	default:
		evidence.DigestsEqual = true
	}
	evidence.Reproducible = evidence.BothSucceeded && evidence.DistinctAttempts && evidence.DigestsEqual
	return evidence, nil
}

func buildOnce(repo, goPath, packagePath, supply, output, stateDir string) (int, string, error) {
	env, err := supplyEnvironment(os.Environ(), stateDir, supply)
	if err != nil {
		return 0, "", err
	}
	cmd := exec.Command(
		goPath,
		"build",
		"-trimpath",
		"-buildvcs=false",
		"-mod=readonly",
		"-o",
		output,
		packagePath,
	)
	cmd.Dir = repo
	cmd.Env = env
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	runErr := cmd.Run()
	if runErr == nil {
		return 0, "", nil
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		return exitErr.ExitCode(), strings.TrimSpace(combined.String()), nil
	}
	return -1, runErr.Error(), nil
}

// supplyEnvironment reuses the adapter's deny-by-default construction and points
// GOPROXY at the named supply. Host module cache is never inherited.
func supplyEnvironment(base []string, stateDir, supply string) ([]string, error) {
	host, err := gobuilder.AllowedHostEnvironment(base, runtime.GOOS)
	if err != nil {
		return nil, err
	}
	out := append([]string(nil), host...)
	for _, setting := range gobuilder.FixedSettings(runtime.GOOS, runtime.GOARCH) {
		key, _, _ := strings.Cut(setting, "=")
		if key == "GOPROXY" || key == "GOFLAGS" {
			continue
		}
		out = append(out, setting)
	}
	out = append(out, "GOPROXY="+fileURL(supply), "GOFLAGS=")
	if stateDir != "" {
		if err := os.MkdirAll(stateDir, 0o755); err != nil {
			return nil, fmt.Errorf("create isolated build state: %w", err)
		}
		for _, dir := range []string{filepath.Join(stateDir, "home"), filepath.Join(stateDir, "tmp")} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create isolated build directory: %w", err)
			}
		}
		out = append(out,
			"GOCACHE="+filepath.Join(stateDir, "cache"),
			"GOMODCACHE="+filepath.Join(stateDir, "modcache"),
			"GOPATH="+filepath.Join(stateDir, "gopath"),
			"HOME="+filepath.Join(stateDir, "home"),
			"USERPROFILE="+filepath.Join(stateDir, "home"),
			"TMPDIR="+filepath.Join(stateDir, "tmp"),
			"TMP="+filepath.Join(stateDir, "tmp"),
			"TEMP="+filepath.Join(stateDir, "tmp"),
		)
	}
	sort.Strings(out)
	return out, nil
}

func fileURL(path string) string {
	clean := filepath.ToSlash(filepath.Clean(path))
	if runtime.GOOS == "windows" {
		return "file:///" + strings.TrimPrefix(clean, "/")
	}
	return "file://" + clean
}

func resolveGo(argument string) (string, error) {
	trimmed := strings.TrimSpace(argument)
	if trimmed == "" {
		trimmed = "go"
	}
	if !strings.Contains(trimmed, "/") && !strings.Contains(trimmed, string(filepath.Separator)) {
		path, err := exec.LookPath(trimmed)
		if err != nil {
			return "", fmt.Errorf("resolve Go executable: %w", err)
		}
		trimmed = path
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve Go executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("Go executable is not a regular file: %s", absolute)
	}
	return absolute, nil
}

func commandOutput(path string, args ...string) (string, error) {
	output, err := exec.Command(path, args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func fileDigest(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func mustAbs(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absolute
}

// randomID names an attempt. It is random on purpose: an attempt identity must
// not be computable from the artifact bytes, so equal bytes can never stand in
// for a second run.
func randomID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("create attempt identity: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func removeWritable(root string) {
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			_ = os.Chmod(path, 0o700)
		} else {
			_ = os.Chmod(path, 0o600)
		}
		return nil
	})
	_ = os.RemoveAll(root)
}
