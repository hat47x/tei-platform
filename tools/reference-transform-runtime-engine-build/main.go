package main

import (
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

	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
	"tei/tools/internal/gobuilder"
)

const (
	evidenceSchema = "tei.reference.transform-engine-build-evidence/v1"
	enginePackage  = "./tools/reference-transform-runtime-engine"
	engineID       = "tei.runtime.engine.deterministic_transform.reference"
	engineVersion  = "1.0.0"
	engineContract = "tei.runtime-engine/deterministic-transform-v1"
)

type sourceMember = gobuilder.SourceMember

type goBuilderEvidence struct {
	Version         string `json:"version"`
	BinaryDigest    string `json:"binaryDigest"`
	ToolchainDigest string `json:"toolchainDigest"`
}

type buildEvidence struct {
	Schema               string                          `json:"schema"`
	RepositoryRevision   string                          `json:"repositoryRevision"`
	EngineID             string                          `json:"engineId"`
	EngineVersion        string                          `json:"engineVersion"`
	EngineContract       string                          `json:"engineContract"`
	Target               string                          `json:"target"`
	RuntimeEngine        *pruntime.RuntimeEngineIdentity `json:"runtimeEngine,omitempty"`
	Entrypoint           string                          `json:"entrypoint"`
	SourceClosure        []sourceMember                  `json:"sourceClosure"`
	SourceClosureDigest  string                          `json:"sourceClosureDigest"`
	Builder              goBuilderEvidence               `json:"builder"`
	Recipe               []string                        `json:"recipe"`
	BuildSettings        []string                        `json:"buildSettings"`
	HostEnvironment      []string                        `json:"hostEnvironment,omitempty"`
	FirstArtifactDigest  string                          `json:"firstArtifactDigest"`
	SecondArtifactDigest string                          `json:"secondArtifactDigest"`
	Reproducible         bool                            `json:"reproducible"`
	ArtifactDigest       string                          `json:"artifactDigest"`
	EvidenceFingerprint  string                          `json:"evidenceFingerprint"`
}

type goListModule = gobuilder.GoListModule

type goListPackage = gobuilder.GoListPackage

func main() {
	repo := flag.String("repo", ".", "path to a clean TEI repository checkout")
	artifactOut := flag.String("artifact-out", "", "optional path outside the repository for the verified engine artifact")
	goExecutable := flag.String("go", "", "optional exact Go executable path; defaults to the current runtime GOROOT")
	flag.Parse()

	evidence, err := run(*repo, *artifactOut, *goExecutable)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(evidence); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !evidence.Reproducible {
		fmt.Fprintln(os.Stderr, "transform Runtime Engine build is not reproducible")
		os.Exit(1)
	}
}

func run(repo, artifactOut, goExecutable string) (buildEvidence, error) {
	root, err := filepath.Abs(repo)
	if err != nil {
		return buildEvidence{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return buildEvidence{}, fmt.Errorf("resolve repository root: %w", err)
	}
	if err := requireCleanRepository(root); err != nil {
		return buildEvidence{}, err
	}
	revision, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return buildEvidence{}, err
	}

	goPath, err := selectGoExecutable(goExecutable)
	if err != nil {
		return buildEvidence{}, err
	}
	hostEnvironment, err := allowedHostEnvironment(os.Environ())
	if err != nil {
		return buildEvidence{}, err
	}
	goVersion, goRoot, goBinaryDigest, goToolchainDigest, err := identifyGoToolchain(root, goPath)
	if err != nil {
		return buildEvidence{}, err
	}

	tempDir, err := os.MkdirTemp("", "tei-transform-engine-build-*")
	if err != nil {
		return buildEvidence{}, err
	}
	defer os.RemoveAll(tempDir)

	sourceClosure, sourceClosureDigest, err := collectModuleSourceClosure(
		root,
		goPath,
		enginePackage,
		filepath.Join(tempDir, "state-list"),
	)
	if err != nil {
		return buildEvidence{}, err
	}

	firstPath := filepath.Join(tempDir, executableName("engine-first"))
	secondPath := filepath.Join(tempDir, executableName("engine-second"))
	if err := buildGoPackage(root, goPath, enginePackage, firstPath, filepath.Join(tempDir, "state-first")); err != nil {
		return buildEvidence{}, fmt.Errorf("build first transform Runtime Engine: %w", err)
	}
	if err := buildGoPackage(root, goPath, enginePackage, secondPath, filepath.Join(tempDir, "state-second")); err != nil {
		return buildEvidence{}, fmt.Errorf("build second transform Runtime Engine: %w", err)
	}
	comparison, err := gobuilder.CompareArtifactFiles(firstPath, secondPath)
	if err != nil {
		return buildEvidence{}, err
	}
	firstDigest, secondDigest := comparison.FirstDigest, comparison.SecondDigest
	reproducible, artifactDigest := comparison.Reproducible, comparison.CandidateDigest
	target := "host/" + runtime.GOOS + "/" + runtime.GOARCH
	var exactRuntimeEngine *pruntime.RuntimeEngineIdentity
	if reproducible {
		engine, err := exactRuntimeEngineIdentity(artifactDigest, target)
		if err != nil {
			return buildEvidence{}, err
		}
		exactRuntimeEngine = &engine
		if artifactOut != "" {
			if err := copyVerifiedArtifactOutsideRepository(root, firstPath, artifactOut, firstDigest); err != nil {
				return buildEvidence{}, err
			}
		}
	}

	out := buildEvidence{
		Schema:              evidenceSchema,
		RepositoryRevision:  revision,
		EngineID:            engineID,
		EngineVersion:       engineVersion,
		EngineContract:      engineContract,
		Target:              target,
		RuntimeEngine:       exactRuntimeEngine,
		Entrypoint:          enginePackage,
		SourceClosure:       sourceClosure,
		SourceClosureDigest: sourceClosureDigest,
		Builder: goBuilderEvidence{
			Version:         goVersion,
			BinaryDigest:    goBinaryDigest,
			ToolchainDigest: goToolchainDigest,
		},
		Recipe: []string{
			"go",
			"build",
			"-trimpath",
			"-buildvcs=false",
			"-mod=readonly",
			enginePackage,
		},
		BuildSettings:        recordedBuildSettings(),
		HostEnvironment:      hostEnvironment,
		FirstArtifactDigest:  firstDigest,
		SecondArtifactDigest: secondDigest,
		Reproducible:         reproducible,
		ArtifactDigest:       artifactDigest,
	}
	out.EvidenceFingerprint, err = evidenceFingerprint(out)
	if err != nil {
		return buildEvidence{}, err
	}
	_ = goRoot
	return out, nil
}

func selectGoExecutable(explicit string) (string, error) {
	return gobuilder.SelectExecutable(explicit, runtime.GOROOT(), runtime.GOOS)
}

func identifyGoToolchain(repo, goPath string) (version, goRoot, binaryDigest, toolchainDigest string, err error) {
	versionCmd := exec.Command(goPath, "version")
	versionCmd.Dir = repo
	versionEnv, err := buildEnvironment(os.Environ(), "")
	if err != nil {
		return "", "", "", "", err
	}
	versionCmd.Env = versionEnv
	versionRaw, err := versionCmd.Output()
	if err != nil {
		return "", "", "", "", fmt.Errorf("read Go version: %w", err)
	}
	version = strings.TrimSpace(string(versionRaw))

	rootCmd := exec.Command(goPath, "env", "GOROOT")
	rootCmd.Dir = repo
	rootEnv, err := buildEnvironment(os.Environ(), "")
	if err != nil {
		return "", "", "", "", err
	}
	rootCmd.Env = rootEnv
	rootRaw, err := rootCmd.Output()
	if err != nil {
		return "", "", "", "", fmt.Errorf("resolve Go root: %w", err)
	}
	goRoot = strings.TrimSpace(string(rootRaw))
	if resolved, resolveErr := filepath.EvalSymlinks(goRoot); resolveErr == nil {
		goRoot = resolved
	}
	binaryDigest, err = hashFile(goPath)
	if err != nil {
		return "", "", "", "", fmt.Errorf("hash Go binary: %w", err)
	}
	rootDigest, err := hashGoToolchain(goRoot)
	if err != nil {
		return "", "", "", "", fmt.Errorf("hash Go toolchain closure: %w", err)
	}
	toolchainDigest = sha256Digest([]byte(version + "\n" + binaryDigest + "\n" + rootDigest + "\n"))
	return version, goRoot, binaryDigest, toolchainDigest, nil
}

func collectModuleSourceClosure(repo, goPath, packagePath, stateDir string) ([]sourceMember, string, error) {
	return gobuilder.ResolveSourceClosure(repo, goPath, packagePath, stateDir)
}

func packageBuildFiles(pkg goListPackage) []string {
	return gobuilder.PackageBuildFiles(pkg)
}

func exactRuntimeEngineIdentity(artifactDigest, target string) (pruntime.RuntimeEngineIdentity, error) {
	engine := pruntime.RuntimeEngineIdentity{
		ID:                   core.MustCanonicalID(engineID),
		Version:              engineVersion,
		ContractVersion:      engineContract,
		ImplementationDigest: artifactDigest,
		Target:               target,
	}
	if err := engine.Validate(); err != nil {
		return pruntime.RuntimeEngineIdentity{}, fmt.Errorf("exact Runtime Engine identity: %w", err)
	}
	return engine, nil
}

func resolveReproducibility(firstDigest, secondDigest string) (bool, string) {
	return gobuilder.ResolveReproducibility(firstDigest, secondDigest)
}

func sourceClosureFingerprint(members []sourceMember) string {
	return gobuilder.SourceClosureDigest(members)
}

func evidenceFingerprint(e buildEvidence) (string, error) {
	e.EvidenceFingerprint = ""
	raw, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func buildGoPackage(repo, goPath, packagePath, output, stateDir string) error {
	return gobuilder.BuildPackage(repo, goPath, packagePath, output, stateDir)
}

func fixedGoBuildSettings() []string {
	return gobuilder.FixedSettings(runtime.GOOS, runtime.GOARCH)
}

func recordedBuildSettings() []string {
	settings := append([]string(nil), fixedGoBuildSettings()...)
	settings = append(settings,
		"STATE_ISOLATION=GOCACHE,GOMODCACHE,GOPATH,HOME,USERPROFILE,TMPDIR,TMP,TEMP",
		"HOST_ENVIRONMENT=deny-by-default",
	)
	sort.Strings(settings)
	return settings
}

func allowedHostEnvironment(base []string) ([]string, error) {
	return allowedHostEnvironmentForOS(base, runtime.GOOS)
}

func allowedHostEnvironmentForOS(base []string, goos string) ([]string, error) {
	return gobuilder.AllowedHostEnvironment(base, goos)
}

func buildEnvironment(base []string, stateDir string) ([]string, error) {
	return gobuilder.Environment(base, stateDir, runtime.GOOS, runtime.GOARCH)
}

func hashGoToolchain(goRoot string) (string, error) {
	return gobuilder.HashToolchain(goRoot, runtime.GOOS, runtime.GOARCH)
}

func copyVerifiedArtifactOutsideRepository(repo, source, destination, expectedDigest string) error {
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	parentResolved, err := filepath.EvalSymlinks(filepath.Dir(destinationAbs))
	if err != nil {
		return fmt.Errorf("resolve artifact-out parent: %w", err)
	}
	destinationResolved := filepath.Join(parentResolved, filepath.Base(destinationAbs))

	repoAbs, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	repoAbs, err = filepath.EvalSymlinks(repoAbs)
	if err != nil {
		return fmt.Errorf("resolve repository root for artifact export: %w", err)
	}
	rel, err := filepath.Rel(repoAbs, destinationResolved)
	if err != nil {
		return err
	}
	parentPrefix := ".." + string(os.PathSeparator)
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, parentPrefix)) {
		return fmt.Errorf("artifact-out must be outside the repository")
	}
	if _, err := os.Lstat(destinationResolved); err == nil {
		return fmt.Errorf("artifact-out already exists")
	} else if !os.IsNotExist(err) {
		return err
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	tempFile, err := os.CreateTemp(parentResolved, ".tei-transform-engine-*")
	if err != nil {
		return fmt.Errorf("create verified Runtime Engine artifact temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if err := tempFile.Chmod(0o755); err != nil {
		tempFile.Close()
		return err
	}
	if _, err := io.Copy(tempFile, sourceFile); err != nil {
		tempFile.Close()
		return fmt.Errorf("copy verified Runtime Engine artifact: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	digest, err := hashFile(tempPath)
	if err != nil {
		return err
	}
	if digest != expectedDigest {
		return fmt.Errorf("copied Runtime Engine artifact digest changed: got %s want %s", digest, expectedDigest)
	}
	if err := publishVerifiedArtifactNoClobber(tempPath, destinationResolved); err != nil {
		return err
	}
	return nil
}

func publishVerifiedArtifactNoClobber(tempPath, destination string) error {
	// tempPath is created in destination's parent directory. A hard-link
	// publish is atomic with respect to destination creation and fails when the
	// destination already exists, unlike rename on platforms where rename may
	// replace an existing file.
	if err := os.Link(tempPath, destination); err != nil {
		return fmt.Errorf("publish verified Runtime Engine artifact without overwrite: %w", err)
	}
	return nil
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func requireCleanRepository(repo string) error {
	status, err := gitOutput(repo, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("repository must be clean before transform Runtime Engine build; git status is not empty")
	}
	return nil
}

func gitOutput(repo string, args ...string) (string, error) {
	all := append([]string{"-C", repo}, args...)
	cmd := exec.Command("git", all...)
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}

func sha256Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
