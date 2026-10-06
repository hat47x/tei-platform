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
	"tei/internal/plugin"
	"tei/internal/workspace"
)

const (
	referencePluginID      = "tei.process.reference"
	referencePluginVersion = "1.0.0"
	referenceWorkspace     = "reference/plugin-self-build/workspace"
	referenceDescriptor    = "reference/plugin-self-build/workspace/plugins/tei-process-reference/1.0.0/plugin.tei"
	referencePackage       = "./tools/reference-process-plugin"
)

type evidence struct {
	Schema                       string                   `json:"schema"`
	RepositoryRevision           string                   `json:"repositoryRevision"`
	RepositoryTreeDigest         string                   `json:"repositoryTreeDigest"`
	GoVersion                    string                   `json:"goVersion"`
	GoBinaryDigest               string                   `json:"goBinaryDigest"`
	GoToolchainDigest            string                   `json:"goToolchainDigest"`
	BuildPlan                    plugin.BuildPlan         `json:"buildPlan"`
	BuildPlanFingerprint         string                   `json:"buildPlanFingerprint"`
	FirstArtifactDigest          string                   `json:"firstArtifactDigest"`
	SecondArtifactDigest         string                   `json:"secondArtifactDigest"`
	Reproducible                 bool                     `json:"reproducible"`
	LockedBuild                  plugin.LockedBuild       `json:"lockedBuild"`
	BuildArtifactLockFingerprint string                   `json:"buildArtifactLockFingerprint"`
	RuntimeDefinitionFingerprint string                   `json:"runtimeDefinitionFingerprint"`
	RuntimePluginLock            plugin.Lock              `json:"runtimePluginLock"`
}

func main() {
	repo := flag.String("repo", ".", "path to a clean TEI repository checkout")
	flag.Parse()

	out, err := run(*repo)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(repo string) (evidence, error) {
	root, err := filepath.Abs(repo)
	if err != nil {
		return evidence{}, err
	}
	if err := requireCleanRepository(root); err != nil {
		return evidence{}, err
	}

	revision, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return evidence{}, err
	}
	treeDigest, err := hashTrackedSourceTree(root)
	if err != nil {
		return evidence{}, err
	}

	loaded, err := workspace.Load(filepath.Join(root, filepath.FromSlash(referenceWorkspace)))
	if err != nil {
		return evidence{}, fmt.Errorf("load plugin self-build reference: %w", err)
	}
	definition, err := exactPlugin(loaded.World.Plugins, referencePluginID, referencePluginVersion)
	if err != nil {
		return evidence{}, err
	}
	if definition.Artifact.Digest != "" {
		return evidence{}, fmt.Errorf("reference source definition unexpectedly predeclares artifact %s", definition.Artifact.Digest)
	}
	if len(loaded.World.Environments) != 1 {
		return evidence{}, fmt.Errorf("reference workspace must contain exactly one environment")
	}

	descriptorDigest, err := hashFile(filepath.Join(root, filepath.FromSlash(referenceDescriptor)))
	if err != nil {
		return evidence{}, err
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		return evidence{}, fmt.Errorf("locate Go builder: %w", err)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(goPath); resolveErr == nil {
		goPath = resolved
	}
	goBinaryDigest, err := hashFile(goPath)
	if err != nil {
		return evidence{}, fmt.Errorf("hash Go builder binary: %w", err)
	}
	goVersionCmd := exec.Command(goPath, "version")
	goVersionCmd.Env = buildEnvironment(os.Environ(), "")
	goVersionBytes, err := goVersionCmd.Output()
	if err != nil {
		return evidence{}, fmt.Errorf("read Go builder version: %w", err)
	}
	goVersion := strings.TrimSpace(string(goVersionBytes))
	goRootCmd := exec.Command(goPath, "env", "GOROOT")
	goRootCmd.Env = buildEnvironment(os.Environ(), "")
	goRootBytes, err := goRootCmd.Output()
	if err != nil {
		return evidence{}, fmt.Errorf("resolve Go toolchain root: %w", err)
	}
	goRoot := strings.TrimSpace(string(goRootBytes))
	if resolved, resolveErr := filepath.EvalSymlinks(goRoot); resolveErr == nil {
		goRoot = resolved
	}
	goRootDigest, err := hashGoToolchain(goRoot)
	if err != nil {
		return evidence{}, fmt.Errorf("hash Go toolchain closure: %w", err)
	}
	goToolchainDigest := sha256Digest([]byte(goVersion + "\n" + goBinaryDigest + "\n" + goRootDigest + "\n"))
	if err := requireNoExternalModules(root, goPath); err != nil {
		return evidence{}, err
	}

	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: definition.ID, Version: definition.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(definition),
		Sources: []plugin.BuildSource{
			{
				Kind:    "git-tree",
				ID:      "tei-repository",
				Locator: "git:HEAD",
				Digest:  treeDigest,
			},
			{
				Kind:    "plugin-definition",
				ID:      "tei-process-reference-definition",
				Locator: referenceDescriptor,
				Digest:  descriptorDigest,
			},
		},
		Builder: plugin.BuilderIdentity{
			ID:             core.MustCanonicalID("tei.builder.go.seed"),
			Version:        goVersion,
			ArtifactDigest: goToolchainDigest,
		},
		Target:  "host-process/" + runtime.GOOS + "/" + runtime.GOARCH,
		Profile: "reference",
		Parameters: []plugin.BuildParameter{
			{Name: "buildvcs", Value: "false"},
			{Name: "cgo", Value: "disabled"},
			{Name: "goarch", Value: runtime.GOARCH},
			{Name: "goenv", Value: "off"},
			{Name: "goexperiment", Value: "default"},
			{Name: "goflags", Value: "empty"},
			{Name: "goos", Value: runtime.GOOS},
			{Name: "goproxy", Value: "off"},
			{Name: "gotoolchain", Value: "local"},
			{Name: "gowork", Value: "off"},
			{Name: "mod", Value: "readonly"},
			{Name: "trimpath", Value: "true"},
		},
	}
	planFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return evidence{}, err
	}

	tempDir, err := os.MkdirTemp("", "tei-plugin-self-build-*")
	if err != nil {
		return evidence{}, err
	}
	defer os.RemoveAll(tempDir)

	firstPath := filepath.Join(tempDir, "plugin-first")
	secondPath := filepath.Join(tempDir, "plugin-second")
	if runtime.GOOS == "windows" {
		firstPath += ".exe"
		secondPath += ".exe"
	}
	firstCache := filepath.Join(tempDir, "cache-first")
	secondCache := filepath.Join(tempDir, "cache-second")
	if err := buildReferencePlugin(root, goPath, firstPath, firstCache); err != nil {
		return evidence{}, err
	}
	if err := buildReferencePlugin(root, goPath, secondPath, secondCache); err != nil {
		return evidence{}, err
	}
	firstDigest, err := hashFile(firstPath)
	if err != nil {
		return evidence{}, err
	}
	secondDigest, err := hashFile(secondPath)
	if err != nil {
		return evidence{}, err
	}
	if firstDigest != secondDigest {
		return evidence{}, fmt.Errorf("reference plugin build is not reproducible: %s != %s", firstDigest, secondDigest)
	}

	lockedBuild, err := plugin.LockBuildOutput(plan, firstDigest)
	if err != nil {
		return evidence{}, err
	}
	buildArtifactLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{lockedBuild},
	}
	buildArtifactLockFingerprint, err := plugin.BuildArtifactLockFingerprint(buildArtifactLock)
	if err != nil {
		return evidence{}, err
	}

	realized, err := plugin.RealizeBuiltDefinition(definition, firstDigest)
	if err != nil {
		return evidence{}, err
	}
	runtimeLock, err := plugin.BuildLock(
		plugin.Catalog{Plugins: []plugin.Definition{realized}},
		loaded.World.Environments,
	)
	if err != nil {
		return evidence{}, err
	}
	runtimePlugin, err := exactLockedPlugin(runtimeLock, loaded.World.Environments[0].ID, definition.ID, definition.Version)
	if err != nil {
		return evidence{}, err
	}
	if err := plugin.VerifyRuntimePluginBuild(plan, lockedBuild, realized, runtimePlugin); err != nil {
		return evidence{}, err
	}
	if err := requireCleanRepository(root); err != nil {
		return evidence{}, fmt.Errorf("repository changed during build: %w", err)
	}

	return evidence{
		Schema:                       "tei.reference.plugin-self-build-evidence/v0",
		RepositoryRevision:           revision,
		RepositoryTreeDigest:         treeDigest,
		GoVersion:                    goVersion,
		GoBinaryDigest:               goBinaryDigest,
		GoToolchainDigest:            goToolchainDigest,
		BuildPlan:                    plan,
		BuildPlanFingerprint:         planFingerprint,
		FirstArtifactDigest:          firstDigest,
		SecondArtifactDigest:         secondDigest,
		Reproducible:                 true,
		LockedBuild:                  lockedBuild,
		BuildArtifactLockFingerprint: buildArtifactLockFingerprint,
		RuntimeDefinitionFingerprint: plugin.DefinitionFingerprint(realized),
		RuntimePluginLock:            runtimeLock,
	}, nil
}



func hashTrackedSourceTree(repo string) (string, error) {
	listed, err := gitOutputBytes(repo, "ls-files", "-z")
	if err != nil {
		return "", err
	}
	parts := strings.Split(string(listed), "\x00")
	paths := make([]string, 0, len(parts))
	for _, rel := range parts {
		if rel != "" {
			paths = append(paths, rel)
		}
	}
	sort.Strings(paths)

	sum := sha256.New()
	for _, rel := range paths {
		path := filepath.Join(repo, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(sum, "%s\x00", filepath.ToSlash(rel)); err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("tracked source symlink is not supported in the first self-build reference: %s", rel)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("tracked source %s is not a regular file or symlink", rel)
		}
		if _, err := fmt.Fprint(sum, "file\x00"); err != nil {
			return "", err
		}
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(sum, f); err != nil {
			f.Close()
			return "", err
		}
		if err := f.Close(); err != nil {
			return "", err
		}
		if _, err := fmt.Fprint(sum, "\n"); err != nil {
			return "", err
		}
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}

func requireNoExternalModules(repo, goPath string) error {
	cmd := exec.Command(
		goPath,
		"list",
		"-deps",
		"-f",
		"{{if .Module}}{{.Module.Path}} {{.Module.Version}}{{end}}",
		referencePackage,
	)
	cmd.Dir = repo
	cmd.Env = buildEnvironment(os.Environ(), "")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("resolve reference plugin dependency closure: %w: %s", err, strings.TrimSpace(string(out)))
	}
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] != "tei" {
			return fmt.Errorf("reference plugin has unresolved external build dependency %q", line)
		}
	}
	return nil
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

func hashToolchainEntry(sum interface {
	Write([]byte) (int, error)
}, goRoot, path string, info os.FileInfo) error {
	rel, err := filepath.Rel(goRoot, path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if _, err := fmt.Fprintf(sum, "%s\x00%o\x00", rel, info.Mode()); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("Go toolchain symlink is not supported in the first self-build reference: %s", rel)
	}
	if !info.Mode().IsRegular() {
		_, err := fmt.Fprint(sum, "special\n")
		return err
	}
	if _, err := fmt.Fprint(sum, "file\x00"); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(sum, f); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	_, err = fmt.Fprint(sum, "\n")
	return err
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func buildReferencePlugin(repo, goPath, output, cacheDir string) error {
	cmd := exec.Command(
		goPath,
		"build",
		"-trimpath",
		"-buildvcs=false",
		"-mod=readonly",
		"-o",
		output,
		referencePackage,
	)
	cmd.Dir = repo
	cmd.Env = buildEnvironment(os.Environ(), cacheDir)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build reference process plugin: %w: %s", err, strings.TrimSpace(string(combined)))
	}
	return nil
}

func buildEnvironment(base []string, cacheDir string) []string {
	out := make([]string, 0, len(base)+10)
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
	out = append(out,
		"CGO_ENABLED=0",
		"GOARCH="+runtime.GOARCH,
		"GOENV=off",
		"GOEXPERIMENT=",
		"GOFLAGS=",
		"GOOS="+runtime.GOOS,
		"GOPROXY=off",
		"GOTOOLCHAIN=local",
		"GOWORK=off",
	)
	if cacheDir != "" {
		out = append(out, "GOCACHE="+cacheDir)
	}
	sort.Strings(out)
	return out
}

func requireCleanRepository(repo string) error {
	status, err := gitOutput(repo, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("repository must be clean before self-build; git status is not empty")
	}
	return nil
}

func exactPlugin(definitions []plugin.Definition, id, version string) (plugin.Definition, error) {
	for _, definition := range definitions {
		if string(definition.ID) == id && definition.Version == version {
			return definition, nil
		}
	}
	return plugin.Definition{}, fmt.Errorf("plugin %s@%s not found", id, version)
}

func exactLockedPlugin(lock plugin.Lock, environmentID core.CanonicalID, id core.CanonicalID, version string) (plugin.LockedPlugin, error) {
	for _, environment := range lock.Environments {
		if environment.Environment != environmentID {
			continue
		}
		for _, locked := range environment.Plugins {
			if locked.ID == id && locked.Version == version {
				return locked, nil
			}
		}
		return plugin.LockedPlugin{}, fmt.Errorf("runtime lock has no plugin %s@%s", id, version)
	}
	return plugin.LockedPlugin{}, fmt.Errorf("runtime lock has no environment %s", environmentID)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}

func sha256Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func gitOutput(repo string, args ...string) (string, error) {
	out, err := gitOutputBytes(repo, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitOutputBytes(repo string, args ...string) ([]byte, error) {
	all := append([]string{"-C", repo}, args...)
	cmd := exec.Command("git", all...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
}
