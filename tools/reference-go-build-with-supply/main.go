// reference-go-build-with-supply は、明示した供給だけを使う隔離環境でGo対象を
// 実Buildし、成果物digestと、既存Source閉包が同じ対象をどう扱うかを別々に
// 記録します（GitHub #2831 SB-B事前確認 / X1）。
//
// この参照はBuildを実行しますが、artifactの受理・署名・権限・配備は行いません。
// 出力されるartifact digestは、その試行で作られたバイト列の同一性だけを示します。
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"tei/tools/internal/gobuilder"
)

const buildSchema = "tei.reference.go-build-with-supply/v0"

const claimBoundary = "one isolated build attempt from the named supply; no closure qualification, no acceptance, no build or runtime authority"

const (
	SourceClosureQualified = "qualified"
	SourceClosureRejected  = "rejected"
)

// BuildEvidence is one attempt.
type BuildEvidence struct {
	SchemaVersion      string   `json:"schemaVersion"`
	Nature             string   `json:"nature"`
	Package            string   `json:"package"`
	GoOS               string   `json:"goos"`
	GoArch             string   `json:"goarch"`
	GoExecutableDigest string   `json:"goExecutableDigest"`
	SupplyRoot         string   `json:"supplyRoot"`
	BuildCommand       []string `json:"buildCommand"`
	BuildSucceeded     bool     `json:"buildSucceeded"`
	BuildExitCode      int      `json:"buildExitCode"`
	BuildError         string   `json:"buildError,omitempty"`
	ToolchainPopulated []string `json:"moduleCacheEntries,omitempty"`
	ArtifactDigest     string   `json:"artifactDigest,omitempty"`
	// StateDir is the isolated build state this process actually used. It is a
	// statement about this run, not a reading of a plan: a caller that asked for a
	// separate state can check that it got one.
	StateDir               string `json:"stateDir,omitempty"`
	SourceClosureState     string `json:"sourceClosureState"`
	SourceClosureError     string `json:"sourceClosureError,omitempty"`
	ArtifactWithoutClosure bool   `json:"artifactProducedDespiteUnqualifiedClosure"`
	Boundary               string `json:"claimBoundary"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reference-go-build-with-supply", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", ".", "対象リポジトリのcheckout")
	pkg := fs.String("package", "", "Buildするpackage（必須）")
	supply := fs.String("supply", "", "GOPROXY=fileで使う供給ディレクトリ（必須）")
	artifactOut := fs.String("artifact-out", "", "成果物を置くリポジトリ外のパス（省略時は一時ファイル）")
	goExecutable := fs.String("go", "", "exact Go実行ファイル（省略時はPATHのgo）")
	stateDir := fs.String("state", "", "分離したBuild state（省略時は一時ディレクトリ）")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *pkg == "" || *supply == "" {
		fmt.Fprintln(stderr, "-package and -supply are required")
		return 2
	}
	state := *stateDir
	cleanup := func() {}
	if state == "" {
		dir, err := os.MkdirTemp("", "tei-go-build-supply-*")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		state = dir
		cleanup = func() { os.RemoveAll(dir) }
	}
	defer cleanup()

	evidence, err := BuildWithSupply(*repo, *goExecutable, *pkg, *supply, *artifactOut, state)
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

// BuildWithSupply builds the target with the existing adapter against an
// isolated module cache whose only proxy is the named supply, then separately
// asks the same adapter's source-closure resolver about the same target.
//
// The two answers are reported apart on purpose: a produced artifact and a
// qualified source closure are different facts, and the X1 negative is exactly
// the case where one holds and the other does not.
func BuildWithSupply(repo, goArgument, packagePath, supply, artifactOut, stateDir string) (BuildEvidence, error) {
	root, err := filepath.Abs(repo)
	if err != nil {
		return BuildEvidence{}, err
	}
	if evaluated, err := filepath.EvalSymlinks(root); err == nil {
		root = evaluated
	}
	exactSupply, err := filepath.EvalSymlinks(mustAbs(supply))
	if err != nil {
		return BuildEvidence{}, fmt.Errorf("supply root is not readable: %w", err)
	}
	exactGo, err := resolveGo(goArgument)
	if err != nil {
		return BuildEvidence{}, err
	}
	goDigest, err := fileDigest(exactGo)
	if err != nil {
		return BuildEvidence{}, err
	}
	evidence := BuildEvidence{
		StateDir:           stateDir,
		SchemaVersion:      buildSchema,
		Nature:             "one isolated build attempt from a named supply; not an acceptance",
		Package:            packagePath,
		GoOS:               runtime.GOOS,
		GoArch:             runtime.GOARCH,
		GoExecutableDigest: goDigest,
		SupplyRoot:         exactSupply,
		Boundary:           claimBoundary,
		BuildCommand: []string{
			exactGo, "build", "-trimpath", "-buildvcs=false", "-mod=readonly", "-o", "<artifact>", packagePath,
			"(deny-by-default environment; GOPROXY=file://<supply>; isolated GOMODCACHE)",
		},
	}

	artifact := artifactOut
	if artifact == "" {
		artifact = filepath.Join(stateDir, "artifact")
	}
	exitCode, buildError, err := buildFromSupply(root, exactGo, packagePath, supply, artifact, stateDir)
	if err != nil {
		return BuildEvidence{}, err
	}
	evidence.BuildExitCode = exitCode
	evidence.BuildError = buildError
	if exitCode == 0 {
		evidence.BuildSucceeded = true
		digest, err := fileDigest(artifact)
		if err != nil {
			return BuildEvidence{}, fmt.Errorf("digest produced artifact: %w", err)
		}
		evidence.ArtifactDigest = digest
	}
	evidence.ToolchainPopulated = moduleCacheModules(filepath.Join(stateDir, "modcache"))

	// Same target, through the same adapter's source-closure resolver.
	_, _, closureErr := gobuilder.ResolveSourceClosure(root, exactGo, packagePath, filepath.Join(stateDir, "closure-state"))
	if closureErr == nil {
		evidence.SourceClosureState = SourceClosureQualified
	} else {
		evidence.SourceClosureState = SourceClosureRejected
		evidence.SourceClosureError = closureErr.Error()
	}
	evidence.ArtifactWithoutClosure = evidence.BuildSucceeded && evidence.SourceClosureState != SourceClosureQualified
	return evidence, nil
}

func mustAbs(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absolute
}

// buildFromSupply runs the adapter build with the supply as the only proxy.
// BuildPackage's own environment already isolates every cache and cache-busting
// setting; only GOPROXY differs, so the supply is applied by delegating to the
// same invocation with a supply-derived state directory contract.
func buildFromSupply(root, goPath, packagePath, supply, artifact, stateDir string) (int, string, error) {
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
		artifact,
		packagePath,
	)
	cmd.Dir = root
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

// moduleCacheModules lists the module directories that actually appeared in the
// isolated cache, which shows which supply entries the build consumed.
func moduleCacheModules(cache string) []string {
	info, err := os.Stat(cache)
	if err != nil || !info.IsDir() {
		return nil
	}
	var found []string
	_ = filepath.WalkDir(cache, func(path string, entry os.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if strings.HasSuffix(name, "@v") || strings.Contains(name, "@") {
			if relative, err := filepath.Rel(cache, path); err == nil {
				found = append(found, filepath.ToSlash(relative))
			}
			return filepath.SkipDir
		}
		return nil
	})
	return found
}
