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
)

const (
	evidenceSchema = "tei.reference.transform-engine-build-evidence/v1"
	enginePackage  = "./tools/reference-transform-runtime-engine"
	engineID       = "tei.runtime.engine.deterministic_transform.reference"
	engineVersion  = "1.0.0"
	engineContract = "tei.runtime-engine/deterministic-transform-v1"
)

type sourceMember struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type goBuilderEvidence struct {
	Version         string `json:"version"`
	BinaryDigest    string `json:"binaryDigest"`
	ToolchainDigest string `json:"toolchainDigest"`
}

type buildEvidence struct {
	Schema               string            `json:"schema"`
	RepositoryRevision   string            `json:"repositoryRevision"`
	EngineID             string                          `json:"engineId"`
	EngineVersion        string                          `json:"engineVersion"`
	EngineContract       string                          `json:"engineContract"`
	Target               string                          `json:"target"`
	RuntimeEngine        *pruntime.RuntimeEngineIdentity `json:"runtimeEngine,omitempty"`
	Entrypoint           string            `json:"entrypoint"`
	SourceClosure        []sourceMember    `json:"sourceClosure"`
	SourceClosureDigest  string            `json:"sourceClosureDigest"`
	Builder              goBuilderEvidence `json:"builder"`
	Recipe               []string          `json:"recipe"`
	BuildSettings        []string          `json:"buildSettings"`
	FirstArtifactDigest  string            `json:"firstArtifactDigest"`
	SecondArtifactDigest string            `json:"secondArtifactDigest"`
	Reproducible         bool              `json:"reproducible"`
	ArtifactDigest       string            `json:"artifactDigest"`
	EvidenceFingerprint  string            `json:"evidenceFingerprint"`
}

type goListModule struct {
	Path string
	Dir  string
}

type goListPackage struct {
	Dir        string
	ImportPath string
	Standard   bool
	Module     *goListModule
	GoFiles    []string
	CgoFiles   []string
	CFiles     []string
	CXXFiles   []string
	MFiles     []string
	HFiles     []string
	FFiles     []string
	SFiles     []string
	SwigFiles  []string
	SwigCXXFiles []string
	SysoFiles  []string
	EmbedFiles []string
}

func main() {
	repo := flag.String("repo", ".", "path to a clean TEI repository checkout")
	artifactOut := flag.String("artifact-out", "", "optional path outside the repository for the verified engine artifact")
	flag.Parse()

	evidence, err := run(*repo, *artifactOut)
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

func run(repo, artifactOut string) (buildEvidence, error) {
	root, err := filepath.Abs(repo)
	if err != nil {
		return buildEvidence{}, err
	}
	if resolved, resolveErr := filepath.EvalSymlinks(root); resolveErr == nil {
		root = resolved
	}
	if err := requireCleanRepository(root); err != nil {
		return buildEvidence{}, err
	}
	revision, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return buildEvidence{}, err
	}

	goPath, err := exec.LookPath("go")
	if err != nil {
		return buildEvidence{}, fmt.Errorf("locate Go builder: %w", err)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(goPath); resolveErr == nil {
		goPath = resolved
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
	firstDigest, err := hashFile(firstPath)
	if err != nil {
		return buildEvidence{}, err
	}
	secondDigest, err := hashFile(secondPath)
	if err != nil {
		return buildEvidence{}, err
	}
	reproducible, artifactDigest := resolveReproducibility(firstDigest, secondDigest)
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
		BuildSettings:        fixedGoBuildSettings(),
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

func identifyGoToolchain(repo, goPath string) (version, goRoot, binaryDigest, toolchainDigest string, err error) {
	versionCmd := exec.Command(goPath, "version")
	versionCmd.Dir = repo
	versionCmd.Env = buildEnvironment(os.Environ(), "")
	versionRaw, err := versionCmd.Output()
	if err != nil {
		return "", "", "", "", fmt.Errorf("read Go version: %w", err)
	}
	version = strings.TrimSpace(string(versionRaw))

	rootCmd := exec.Command(goPath, "env", "GOROOT")
	rootCmd.Dir = repo
	rootCmd.Env = buildEnvironment(os.Environ(), "")
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
	cmd := exec.Command(
		goPath,
		"list",
		"-deps",
		"-json",
		"-mod=readonly",
		packagePath,
	)
	cmd.Dir = repo
	cmd.Env = buildEnvironment(os.Environ(), stateDir)
	raw, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, "", fmt.Errorf("resolve Go source closure: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, "", fmt.Errorf("resolve Go source closure: %w", err)
	}

	rootClean, err := filepath.EvalSymlinks(repo)
	if err != nil {
		rootClean = filepath.Clean(repo)
	}
	tracked, err := gitTrackedPaths(repo)
	if err != nil {
		return nil, "", err
	}
	paths := map[string]struct{}{"go.mod": {}}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	for {
		var pkg goListPackage
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, "", fmt.Errorf("decode Go dependency closure: %w", err)
		}
		if pkg.Standard {
			continue
		}
		if pkg.Module == nil {
			return nil, "", fmt.Errorf("Go dependency %s has no module identity", pkg.ImportPath)
		}
		moduleDir, err := filepath.EvalSymlinks(pkg.Module.Dir)
		if err != nil {
			moduleDir = filepath.Clean(pkg.Module.Dir)
		}
		if moduleDir != rootClean {
			return nil, "", fmt.Errorf(
				"transform Runtime Engine has external module dependency %s from %s",
				pkg.Module.Path,
				pkg.Module.Dir,
			)
		}
		for _, name := range packageBuildFiles(pkg) {
			full := filepath.Join(pkg.Dir, name)
			rel, err := filepath.Rel(repo, full)
			if err != nil {
				return nil, "", err
			}
			rel = filepath.ToSlash(rel)
			if rel == ".." || strings.HasPrefix(rel, "../") {
				return nil, "", fmt.Errorf("Go build source escapes repository: %s", full)
			}
			paths[rel] = struct{}{}
		}
	}

	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	members := make([]sourceMember, 0, len(ordered))
	for _, rel := range ordered {
		if _, ok := tracked[rel]; !ok {
			return nil, "", fmt.Errorf("Go source closure member is not Git-tracked: %s", rel)
		}
		info, err := os.Lstat(filepath.Join(repo, filepath.FromSlash(rel)))
		if err != nil {
			return nil, "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, "", fmt.Errorf("Go source closure member is not a regular file: %s", rel)
		}
		digest, err := hashFile(filepath.Join(repo, filepath.FromSlash(rel)))
		if err != nil {
			return nil, "", err
		}
		members = append(members, sourceMember{Path: rel, Digest: digest})
	}
	return members, sourceClosureFingerprint(members), nil
}

func packageBuildFiles(pkg goListPackage) []string {
	var out []string
	for _, files := range [][]string{
		pkg.GoFiles,
		pkg.CgoFiles,
		pkg.CFiles,
		pkg.CXXFiles,
		pkg.MFiles,
		pkg.HFiles,
		pkg.FFiles,
		pkg.SFiles,
		pkg.SwigFiles,
		pkg.SwigCXXFiles,
		pkg.SysoFiles,
		pkg.EmbedFiles,
	} {
		out = append(out, files...)
	}
	sort.Strings(out)
	return out
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
	if firstDigest != secondDigest {
		return false, ""
	}
	return true, firstDigest
}

func sourceClosureFingerprint(members []sourceMember) string {
	ordered := append([]sourceMember(nil), members...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	sum := sha256.New()
	for _, member := range ordered {
		fmt.Fprintf(sum, "%s\x00%s\n", member.Path, member.Digest)
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil))
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
	cmd.Env = buildEnvironment(os.Environ(), stateDir)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build %s: %w: %s", packagePath, err, strings.TrimSpace(string(combined)))
	}
	return nil
}

func fixedGoBuildSettings() []string {
	settings := []string{
		"CGO_ENABLED=0",
		"GOARCH=" + runtime.GOARCH,
		"GOENV=off",
		"GOEXPERIMENT=",
		"GOFLAGS=",
		"GOOS=" + runtime.GOOS,
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOTOOLCHAIN=local",
		"GOVCS=*:off",
		"GOWORK=off",
	}
	sort.Strings(settings)
	return settings
}

func buildEnvironment(base []string, stateDir string) []string {
	out := make([]string, 0, len(base)+14)
	for _, entry := range base {
		key := entry
		if i := strings.IndexByte(entry, '='); i >= 0 {
			key = entry[:i]
		}
		if strings.HasPrefix(key, "GO") || key == "CGO_ENABLED" {
			continue
		}
		out = append(out, entry)
	}
	out = append(out, fixedGoBuildSettings()...)
	if stateDir != "" {
		out = append(out,
			"GOCACHE="+filepath.Join(stateDir, "cache"),
			"GOMODCACHE="+filepath.Join(stateDir, "modcache"),
			"GOPATH="+filepath.Join(stateDir, "gopath"),
		)
	}
	sort.Strings(out)
	return out
}

func hashGoToolchain(goRoot string) (string, error) {
	if strings.TrimSpace(goRoot) == "" {
		return "", fmt.Errorf("Go toolchain root is empty")
	}
	paths := []string{
		filepath.Join(goRoot, "VERSION"),
		filepath.Join(goRoot, "bin", executableName("go")),
		filepath.Join(goRoot, "pkg", "tool", runtime.GOOS+"_"+runtime.GOARCH),
		filepath.Join(goRoot, "src"),
	}
	sum := sha256.New()
	for _, root := range paths {
		info, err := os.Lstat(root)
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			if err := hashToolchainEntry(sum, goRoot, root, info); err != nil {
				return "", err
			}
			continue
		}
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			return hashToolchainEntry(sum, goRoot, path, info)
		}); err != nil {
			return "", err
		}
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}

func hashToolchainEntry(sum io.Writer, goRoot, path string, info os.FileInfo) error {
	rel, err := filepath.Rel(goRoot, path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if _, err := fmt.Fprintf(sum, "%s\x00%o\x00", rel, info.Mode()); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("Go toolchain symlink is not supported in this Reference: %s", rel)
	}
	if !info.Mode().IsRegular() {
		_, err := fmt.Fprint(sum, "special\n")
		return err
	}
	if _, err := fmt.Fprint(sum, "file\x00"); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(sum, file); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	_, err = fmt.Fprint(sum, "\n")
	return err
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
	if resolved, resolveErr := filepath.EvalSymlinks(repoAbs); resolveErr == nil {
		repoAbs = resolved
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
	if err := os.Rename(tempPath, destinationResolved); err != nil {
		return fmt.Errorf("publish verified Runtime Engine artifact: %w", err)
	}
	return nil
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func gitTrackedPaths(repo string) (map[string]struct{}, error) {
	all := []string{"-C", repo, "ls-files", "-z"}
	cmd := exec.Command("git", all...)
	raw, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	out := map[string]struct{}{}
	for _, path := range strings.Split(string(raw), "\x00") {
		if path == "" {
			continue
		}
		out[filepath.ToSlash(path)] = struct{}{}
	}
	return out, nil
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
