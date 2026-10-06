package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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
	GoBuilderDigest              string                   `json:"goBuilderDigest"`
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
	treeBytes, err := gitOutputBytes(root, "ls-tree", "-r", "--full-tree", "HEAD")
	if err != nil {
		return evidence{}, err
	}
	treeDigest := sha256Digest(treeBytes)

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
	goDigest, err := hashFile(goPath)
	if err != nil {
		return evidence{}, fmt.Errorf("hash Go builder: %w", err)
	}
	goVersionBytes, err := exec.Command(goPath, "version").Output()
	if err != nil {
		return evidence{}, fmt.Errorf("read Go builder version: %w", err)
	}
	goVersion := strings.TrimSpace(string(goVersionBytes))

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
			ArtifactDigest: goDigest,
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
	if err := buildReferencePlugin(root, goPath, firstPath); err != nil {
		return evidence{}, err
	}
	if err := buildReferencePlugin(root, goPath, secondPath); err != nil {
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
		GoBuilderDigest:              goDigest,
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

func buildReferencePlugin(repo, goPath, output string) error {
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
	cmd.Env = buildEnvironment(os.Environ())
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build reference process plugin: %w: %s", err, strings.TrimSpace(string(combined)))
	}
	return nil
}

func buildEnvironment(base []string) []string {
	remove := map[string]struct{}{
		"CGO_ENABLED": {},
		"GOAMD64":     {},
		"GOARCH":      {},
		"GOARM":       {},
		"GOARM64":     {},
		"GOENV":       {},
		"GOEXPERIMENT": {},
		"GOFLAGS":     {},
		"GOMIPS":      {},
		"GOMIPS64":    {},
		"GOOS":        {},
		"GOPPC64":     {},
		"GORISCV64":   {},
		"GOROOT":      {},
		"GOPROXY":     {},
		"GOTOOLCHAIN": {},
		"GOWORK":      {},
	}
	out := make([]string, 0, len(base)+8)
	for _, entry := range base {
		key := entry
		if i := strings.IndexByte(entry, '='); i >= 0 {
			key = entry[:i]
		}
		if _, drop := remove[key]; !drop {
			out = append(out, entry)
		}
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
	if _, err := sum.ReadFrom(f); err != nil {
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
