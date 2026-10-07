package main

import (
	"context"
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
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/plugin/implementation"
	pluginbuiltin "tei/internal/plugin/implementation/builtin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/validation"
	"tei/internal/workspace"
)

const (
	referencePluginID      = "tei.process.reference"
	referencePluginVersion = "1.0.0"
	referenceWorkspace     = "reference/plugin-self-build/workspace"
	referenceDescriptor    = "reference/plugin-self-build/workspace/plugins/tei-process-reference/1.0.0/plugin.tei"
	referencePackage       = "./tools/reference-process-plugin"
	packagePluginID        = "tei.runtime.reference"
	packagePluginVersion   = "1.0.0"
	packageWorkspace       = "reference/workspace"
	packageDescriptor      = "reference/workspace/plugins/tei-runtime-reference/1.0.0/plugin.tei"
	packageBuilderPackage  = "./tools/reference-plugin-package-builder"
	nativePluginID         = "tei.native.runtime.reference"
	nativePluginVersion    = "1.0.0"
	nativeDescriptor       = "reference/plugin-self-build/workspace/plugins/tei-native-runtime-reference/1.0.0/plugin.tei"
	nativeImplementation   = "reference/plugin-self-build/workspace/plugin-implementations/tei-native-runtime-reference/1.0.0/implementation.tei"
	nativeBuilderPackage    = "./tools/reference-native-plugin-builder"
)

type evidence struct {
	Schema                       string             `json:"schema"`
	RepositoryRevision           string             `json:"repositoryRevision"`
	RepositoryTreeDigest         string             `json:"repositoryTreeDigest"`
	GoVersion                    string             `json:"goVersion"`
	GoBinaryDigest               string             `json:"goBinaryDigest"`
	GoToolchainDigest            string             `json:"goToolchainDigest"`
	PackageBuilderFirstDigest    string             `json:"packageBuilderFirstDigest"`
	PackageBuilderSecondDigest   string             `json:"packageBuilderSecondDigest"`
	PackageBuilderReproducible   bool               `json:"packageBuilderReproducible"`
	PackageBuildPlan             plugin.BuildPlan   `json:"packageBuildPlan"`
	PackageBuildPlanFingerprint  string             `json:"packageBuildPlanFingerprint"`
	PackageFirstArtifactDigest   string             `json:"packageFirstArtifactDigest"`
	PackageSecondArtifactDigest  string             `json:"packageSecondArtifactDigest"`
	PackageReproducible          bool               `json:"packageReproducible"`
	PackageLockedBuild           plugin.LockedBuild `json:"packageLockedBuild"`
	PackageLockFingerprint       string             `json:"packageLockFingerprint"`
	NativeSourceFingerprint      string             `json:"nativeSourceFingerprint"`
	NativeBuilderFirstDigest     string             `json:"nativeBuilderFirstDigest"`
	NativeBuilderSecondDigest    string             `json:"nativeBuilderSecondDigest"`
	NativeBuilderReproducible    bool               `json:"nativeBuilderReproducible"`
	NativeBuildPlan             plugin.BuildPlan   `json:"nativeBuildPlan"`
	NativeBuildPlanFingerprint  string             `json:"nativeBuildPlanFingerprint"`
	NativeFirstArtifactDigest   string             `json:"nativeFirstArtifactDigest"`
	NativeSecondArtifactDigest  string             `json:"nativeSecondArtifactDigest"`
	NativeReproducible          bool               `json:"nativeReproducible"`
	NativeLockedBuild           plugin.LockedBuild `json:"nativeLockedBuild"`
	NativeLockFingerprint       string             `json:"nativeLockFingerprint"`
	NativeDefinitionFingerprint string                      `json:"nativeDefinitionFingerprint"`
	NativeRuntimeEngineSourceDigest string                   `json:"nativeRuntimeEngineSourceDigest"`
	NativeRuntimeEngine          pruntime.RuntimeEngineIdentity `json:"nativeRuntimeEngine"`
	NativeRuntimeEngineFingerprint string                     `json:"nativeRuntimeEngineFingerprint"`
	NativeInvocation            executor.Result             `json:"nativeInvocation"`
	BuildPlan                    plugin.BuildPlan   `json:"buildPlan"`
	BuildPlanFingerprint         string             `json:"buildPlanFingerprint"`
	FirstArtifactDigest          string             `json:"firstArtifactDigest"`
	SecondArtifactDigest         string             `json:"secondArtifactDigest"`
	Reproducible                 bool               `json:"reproducible"`
	LockedBuild                  plugin.LockedBuild `json:"lockedBuild"`
	BuildArtifactLockFingerprint string             `json:"buildArtifactLockFingerprint"`
	RuntimeDefinitionFingerprint string             `json:"runtimeDefinitionFingerprint"`
	RuntimePluginLockFingerprint string             `json:"runtimePluginLockFingerprint"`
	RuntimePluginLock            plugin.Lock        `json:"runtimePluginLock"`
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
	if err := requireValidWorld("plugin self-build reference", loaded.World); err != nil {
		return evidence{}, err
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
	nativeRuntimeEngineSourceDigest, err := hashSourceFiles(root, []string{
		"internal/plugin/implementation/model.go",
		"internal/plugin/implementation/builtin/adapter.go",
		"internal/plugin/runtime/model.go",
		"internal/plugin/runtime/adapters.go",
	})
	if err != nil {
		return evidence{}, fmt.Errorf("hash TEI-native runtime engine source closure: %w", err)
	}
	nativeRuntimeEngine := pruntime.RuntimeEngineIdentity{
		ID:                   core.MustCanonicalID("tei.runtime.engine.static_response.reference"),
		Version:              revision,
		ContractVersion:      "tei.runtime-engine/static-response-v1",
		ImplementationDigest: sha256Digest([]byte(nativeRuntimeEngineSourceDigest + "\n" + goToolchainDigest + "\n")),
		Target:               "host/" + runtime.GOOS + "/" + runtime.GOARCH,
	}
	nativeRuntimeEngineFingerprint, err := pruntime.RuntimeEngineIdentityFingerprint(nativeRuntimeEngine)
	if err != nil {
		return evidence{}, err
	}
	if err := requireNoExternalModules(root, goPath, referencePackage, packageBuilderPackage, nativeBuilderPackage); err != nil {
		return evidence{}, err
	}

	packageLoaded, err := workspace.Load(filepath.Join(root, filepath.FromSlash(packageWorkspace)))
	if err != nil {
		return evidence{}, fmt.Errorf("load standard plugin package reference: %w", err)
	}
	if err := requireValidWorld("standard plugin package reference", packageLoaded.World); err != nil {
		return evidence{}, err
	}
	packageDefinition, err := exactPlugin(packageLoaded.World.Plugins, packagePluginID, packagePluginVersion)
	if err != nil {
		return evidence{}, err
	}
	if packageDefinition.Artifact.Digest != "" {
		return evidence{}, fmt.Errorf("standard package source unexpectedly predeclares artifact %s", packageDefinition.Artifact.Digest)
	}
	packageDescriptorDigest, err := hashFile(filepath.Join(root, filepath.FromSlash(packageDescriptor)))
	if err != nil {
		return evidence{}, err
	}

	packageBuilderDir, err := os.MkdirTemp("", "tei-plugin-package-builder-*")
	if err != nil {
		return evidence{}, err
	}
	defer os.RemoveAll(packageBuilderDir)
	packageBuilderFirstPath := filepath.Join(packageBuilderDir, "builder-first")
	packageBuilderSecondPath := filepath.Join(packageBuilderDir, "builder-second")
	if runtime.GOOS == "windows" {
		packageBuilderFirstPath += ".exe"
		packageBuilderSecondPath += ".exe"
	}
	if err := buildGoPackage(
		root,
		goPath,
		packageBuilderPackage,
		packageBuilderFirstPath,
		filepath.Join(packageBuilderDir, "state-builder-first"),
	); err != nil {
		return evidence{}, fmt.Errorf("build first Plugin package builder: %w", err)
	}
	if err := buildGoPackage(
		root,
		goPath,
		packageBuilderPackage,
		packageBuilderSecondPath,
		filepath.Join(packageBuilderDir, "state-builder-second"),
	); err != nil {
		return evidence{}, fmt.Errorf("build second Plugin package builder: %w", err)
	}
	packageBuilderFirstDigest, err := hashFile(packageBuilderFirstPath)
	if err != nil {
		return evidence{}, err
	}
	packageBuilderSecondDigest, err := hashFile(packageBuilderSecondPath)
	if err != nil {
		return evidence{}, err
	}
	if packageBuilderFirstDigest != packageBuilderSecondDigest {
		return evidence{}, fmt.Errorf(
			"Plugin package builder is not reproducible: %s != %s",
			packageBuilderFirstDigest,
			packageBuilderSecondDigest,
		)
	}

	packagePlan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: packageDefinition.ID, Version: packageDefinition.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(packageDefinition),
		Sources: []plugin.BuildSource{
			{
				Kind:    "git-tree",
				ID:      "tei-repository",
				Locator: "git:HEAD",
				Digest:  treeDigest,
			},
			{
				Kind:    "plugin-definition",
				ID:      "tei-runtime-reference-definition",
				Locator: packageDescriptor,
				Digest:  packageDescriptorDigest,
			},
		},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.package.reference.seed"),
			Version:         revision,
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  packageBuilderFirstDigest,
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "tei-plugin-package",
			Entrypoint: packageDescriptor,
		},
		OutputKind: plugin.BuildOutputPluginPackage,
		Target:     "plugin-package",
		Profile:    "reference",
		Parameters: []plugin.BuildParameter{
			{Name: "encoding", Value: "tei-plugin-package-json-v1"},
		},
	}
	packagePlanFingerprint, err := plugin.BuildPlanFingerprint(packagePlan)
	if err != nil {
		return evidence{}, err
	}
	packagePlanPath := filepath.Join(packageBuilderDir, "package-plan.json")
	packagePlanBytes, err := json.Marshal(packagePlan)
	if err != nil {
		return evidence{}, err
	}
	if err := os.WriteFile(packagePlanPath, packagePlanBytes, 0o644); err != nil {
		return evidence{}, err
	}
	packageFirstPath := filepath.Join(packageBuilderDir, "package-first.json")
	packageSecondPath := filepath.Join(packageBuilderDir, "package-second.json")
	if err := runPluginPackageBuilder(root, packageBuilderFirstPath, packagePlanPath, packageFirstPath); err != nil {
		return evidence{}, fmt.Errorf("first Plugin package materialization: %w", err)
	}
	if err := runPluginPackageBuilder(root, packageBuilderSecondPath, packagePlanPath, packageSecondPath); err != nil {
		return evidence{}, fmt.Errorf("second Plugin package materialization: %w", err)
	}
	packageFirstBytes, err := os.ReadFile(packageFirstPath)
	if err != nil {
		return evidence{}, err
	}
	packageSecondBytes, err := os.ReadFile(packageSecondPath)
	if err != nil {
		return evidence{}, err
	}
	packageFirstArtifactDigest, err := hashFile(packageFirstPath)
	if err != nil {
		return evidence{}, err
	}
	packageSecondArtifactDigest, err := hashFile(packageSecondPath)
	if err != nil {
		return evidence{}, err
	}
	if string(packageFirstBytes) != string(packageSecondBytes) || packageFirstArtifactDigest != packageSecondArtifactDigest {
		return evidence{}, fmt.Errorf(
			"standard plugin package build is not reproducible: %s != %s",
			packageFirstArtifactDigest,
			packageSecondArtifactDigest,
		)
	}
	packageLockedBuild, err := plugin.LockBuildOutput(packagePlan, packageFirstArtifactDigest)
	if err != nil {
		return evidence{}, err
	}
	packageArtifactLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{packageLockedBuild},
	}
	packageLockFingerprint, err := plugin.BuildArtifactLockFingerprint(packageArtifactLock)
	if err != nil {
		return evidence{}, err
	}

	nativeDefinition, err := exactPlugin(loaded.World.Plugins, nativePluginID, nativePluginVersion)
	if err != nil {
		return evidence{}, err
	}
	if nativeDefinition.Artifact.Digest != "" {
		return evidence{}, fmt.Errorf("TEI-native source definition unexpectedly predeclares artifact %s", nativeDefinition.Artifact.Digest)
	}
	nativeSource, err := exactImplementation(loaded.PluginImplementations, nativePluginID, nativePluginVersion)
	if err != nil {
		return evidence{}, err
	}
	nativeSourceFingerprint, err := implementation.Fingerprint(nativeSource)
	if err != nil {
		return evidence{}, err
	}
	nativeDescriptorDigest, err := hashFile(filepath.Join(root, filepath.FromSlash(nativeDescriptor)))
	if err != nil {
		return evidence{}, err
	}
	nativeImplementationDigest, err := hashFile(filepath.Join(root, filepath.FromSlash(nativeImplementation)))
	if err != nil {
		return evidence{}, err
	}

	nativeBuilderDir, err := os.MkdirTemp("", "tei-native-plugin-builder-*")
	if err != nil {
		return evidence{}, err
	}
	defer os.RemoveAll(nativeBuilderDir)
	nativeBuilderFirstPath := filepath.Join(nativeBuilderDir, "builder-first")
	nativeBuilderSecondPath := filepath.Join(nativeBuilderDir, "builder-second")
	if runtime.GOOS == "windows" {
		nativeBuilderFirstPath += ".exe"
		nativeBuilderSecondPath += ".exe"
	}
	if err := buildGoPackage(
		root,
		goPath,
		nativeBuilderPackage,
		nativeBuilderFirstPath,
		filepath.Join(nativeBuilderDir, "state-builder-first"),
	); err != nil {
		return evidence{}, fmt.Errorf("build first native Plugin builder: %w", err)
	}
	if err := buildGoPackage(
		root,
		goPath,
		nativeBuilderPackage,
		nativeBuilderSecondPath,
		filepath.Join(nativeBuilderDir, "state-builder-second"),
	); err != nil {
		return evidence{}, fmt.Errorf("build second native Plugin builder: %w", err)
	}
	nativeBuilderFirstDigest, err := hashFile(nativeBuilderFirstPath)
	if err != nil {
		return evidence{}, err
	}
	nativeBuilderSecondDigest, err := hashFile(nativeBuilderSecondPath)
	if err != nil {
		return evidence{}, err
	}
	if nativeBuilderFirstDigest != nativeBuilderSecondDigest {
		return evidence{}, fmt.Errorf(
			"TEI-native Plugin builder is not reproducible: %s != %s",
			nativeBuilderFirstDigest,
			nativeBuilderSecondDigest,
		)
	}

	nativePlan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      nativeSource.Plugin,
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(nativeDefinition),
		Sources: []plugin.BuildSource{
			{
				Kind:    "git-tree",
				ID:      "tei-repository",
				Locator: "git:HEAD",
				Digest:  treeDigest,
			},
			{
				Kind:    "plugin-definition",
				ID:      "tei-native-runtime-reference-definition",
				Locator: nativeDescriptor,
				Digest:  nativeDescriptorDigest,
			},
			{
				Kind:    "plugin-implementation",
				ID:      "tei-native-runtime-reference-implementation",
				Locator: nativeImplementation,
				Digest:  nativeImplementationDigest,
			},
		},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.native.reference.seed"),
			Version:         revision,
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  nativeBuilderFirstDigest,
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "tei-native-static-response",
			Entrypoint: nativeImplementation,
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "interpreted-builtin/host/" + runtime.GOOS + "/" + runtime.GOARCH,
		Profile:    "reference",
		Parameters: []plugin.BuildParameter{
			{Name: "execution-model", Value: "interpreted-builtin"},
			{Name: "profile", Value: implementation.StaticResponseProfileV1},
		},
	}
	nativePlanFingerprint, err := plugin.BuildPlanFingerprint(nativePlan)
	if err != nil {
		return evidence{}, err
	}

	nativeFirstPath := filepath.Join(nativeBuilderDir, "realization-first.json")
	nativeSecondPath := filepath.Join(nativeBuilderDir, "realization-second.json")
	if err := runNativePluginBuilder(root, nativeBuilderFirstPath, nativeFirstPath); err != nil {
		return evidence{}, fmt.Errorf("first TEI-native realization: %w", err)
	}
	if err := runNativePluginBuilder(root, nativeBuilderSecondPath, nativeSecondPath); err != nil {
		return evidence{}, fmt.Errorf("second TEI-native realization: %w", err)
	}
	nativeFirstBytes, err := os.ReadFile(nativeFirstPath)
	if err != nil {
		return evidence{}, err
	}
	nativeSecondBytes, err := os.ReadFile(nativeSecondPath)
	if err != nil {
		return evidence{}, err
	}
	nativeFirstArtifactDigest, err := hashFile(nativeFirstPath)
	if err != nil {
		return evidence{}, err
	}
	nativeSecondArtifactDigest, err := hashFile(nativeSecondPath)
	if err != nil {
		return evidence{}, err
	}
	if string(nativeFirstBytes) != string(nativeSecondBytes) || nativeFirstArtifactDigest != nativeSecondArtifactDigest {
		return evidence{}, fmt.Errorf(
			"TEI-native builtin realization is not reproducible: %s != %s",
			nativeFirstArtifactDigest,
			nativeSecondArtifactDigest,
		)
	}
	nativeLockedBuild, err := plugin.LockBuildOutput(nativePlan, nativeFirstArtifactDigest)
	if err != nil {
		return evidence{}, err
	}
	nativeArtifactLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{nativeLockedBuild},
	}
	nativeLockFingerprint, err := plugin.BuildArtifactLockFingerprint(nativeArtifactLock)
	if err != nil {
		return evidence{}, err
	}
	nativeRealized, err := plugin.RealizeBuiltDefinition(nativeDefinition, nativeFirstArtifactDigest)
	if err != nil {
		return evidence{}, err
	}
	nativeBuiltin, err := pluginbuiltin.New(nativeFirstBytes, nativeRealized)
	if err != nil {
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
			ID:              core.MustCanonicalID("tei.builder.go.seed"),
			Version:         goVersion,
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  goToolchainDigest,
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "go-package",
			Entrypoint: referencePackage,
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "host-process/" + runtime.GOOS + "/" + runtime.GOARCH,
		Profile:    "reference",
		Parameters: []plugin.BuildParameter{
			{Name: "buildvcs", Value: "false"},
			{Name: "cgo", Value: "disabled"},
			{Name: "goarch", Value: runtime.GOARCH},
			{Name: "goenv", Value: "off"},
			{Name: "goexperiment", Value: "default"},
			{Name: "gocache", Value: "isolated"},
			{Name: "goflags", Value: "empty"},
			{Name: "gomodcache", Value: "isolated-empty"},
			{Name: "goos", Value: runtime.GOOS},
			{Name: "gopath", Value: "isolated-empty"},
			{Name: "goproxy", Value: "off"},
			{Name: "gosumdb", Value: "off"},
			{Name: "gotoolchain", Value: "local"},
			{Name: "govcs", Value: "*:off"},
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
	firstState := filepath.Join(tempDir, "state-first")
	secondState := filepath.Join(tempDir, "state-second")
	if err := buildReferencePlugin(root, goPath, firstPath, firstState); err != nil {
		return evidence{}, err
	}
	if err := buildReferencePlugin(root, goPath, secondPath, secondState); err != nil {
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
	runtimeCatalog := plugin.Catalog{
		Plugins: append([]plugin.Definition(nil), loaded.World.Plugins...),
	}
	replacePluginDefinition(runtimeCatalog.Plugins, realized)
	replacePluginDefinition(runtimeCatalog.Plugins, nativeRealized)

	runtimeLock, err := plugin.BuildLock(runtimeCatalog, loaded.World.Environments)
	if err != nil {
		return evidence{}, err
	}
	runtimeLock, err = plugin.BindRuntimeBuild(runtimeLock, loaded.World.Environments[0].ID, lockedBuild)
	if err != nil {
		return evidence{}, err
	}
	runtimeLock, err = plugin.BindRuntimeBuild(runtimeLock, loaded.World.Environments[0].ID, nativeLockedBuild)
	if err != nil {
		return evidence{}, err
	}
	if runtimeLock.FormatVersion != plugin.BuildTraceLockFormatVersion {
		return evidence{}, fmt.Errorf(
			"runtime plugin lock did not upgrade to build-trace format: got %d",
			runtimeLock.FormatVersion,
		)
	}
	runtimePlugin, err := exactLockedPlugin(runtimeLock, loaded.World.Environments[0].ID, definition.ID, definition.Version)
	if err != nil {
		return evidence{}, err
	}
	if runtimePlugin.BuildPlanFingerprint != lockedBuild.PlanFingerprint {
		return evidence{}, fmt.Errorf("process runtime plugin lock lost build plan trace")
	}
	if err := plugin.VerifyRuntimePluginBuild(plan, lockedBuild, realized, runtimePlugin); err != nil {
		return evidence{}, err
	}
	nativeRuntimePlugin, err := exactLockedPlugin(
		runtimeLock,
		loaded.World.Environments[0].ID,
		nativeDefinition.ID,
		nativeDefinition.Version,
	)
	if err != nil {
		return evidence{}, err
	}
	if nativeRuntimePlugin.BuildPlanFingerprint != nativeLockedBuild.PlanFingerprint {
		return evidence{}, fmt.Errorf("TEI-native runtime plugin lock lost build plan trace")
	}
	if err := plugin.VerifyRuntimePluginBuild(
		nativePlan,
		nativeLockedBuild,
		nativeRealized,
		nativeRuntimePlugin,
	); err != nil {
		return evidence{}, err
	}

	runtimePlan, err := pruntime.BuildPlan(
		runtimeCatalog,
		loaded.World.Environments[0],
		runtimeLock,
	)
	if err != nil {
		return evidence{}, err
	}
	nativeRuntimeEngineLock := pruntime.RuntimeEngineLock{
		FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
		Bindings: []pruntime.RuntimeEngineBinding{{
			Plugin: plugin.Identity{ID: nativeDefinition.ID, Version: nativeDefinition.Version},
			Engine: nativeRuntimeEngine,
		}},
	}
	runtimePlan, err = pruntime.BindRuntimeEngineLock(runtimePlan, nativeRuntimeEngineLock)
	if err != nil {
		return evidence{}, err
	}
	nativeRegistry := pruntime.Registry{
		BuiltinRegistrations: map[string]pruntime.BuiltinRegistration{
			nativePluginID + "@" + nativePluginVersion: {
				Builtin:              nativeBuiltin,
				BuildPlanFingerprint: nativeLockedBuild.PlanFingerprint,
				RealizationDigest:    nativeFirstArtifactDigest,
				RuntimeEngine:        nativeRuntimeEngine,
			},
		},
	}
	nativeAdapter := pruntime.DeploymentExecutor{
		Plan:         runtimePlan,
		Registry:     nativeRegistry,
		CapabilityID: core.MustCanonicalID("runtime.native_reference"),
	}
	nativeInvocation, err := nativeAdapter.Execute(context.Background(), executor.Command{
		ActionID:       "application/start",
		IdempotencyKey: "tei-native-reference",
		Kind:           string(deployment.ActionApplicationStart),
	})
	if err != nil {
		return evidence{}, fmt.Errorf("invoke TEI-native builtin Reference: %w", err)
	}
	if nativeInvocation.AttemptID != "tei-native-reference" ||
		nativeInvocation.Outcome != executor.OutcomeAccepted {
		return evidence{}, fmt.Errorf("unexpected TEI-native builtin Reference result: %#v", nativeInvocation)
	}
	if nativeInvocation.PluginInvocation == nil ||
		nativeInvocation.PluginInvocation.BuildPlanFingerprint != nativeLockedBuild.PlanFingerprint ||
		nativeInvocation.PluginInvocation.RealizationDigest != nativeFirstArtifactDigest ||
		nativeInvocation.PluginInvocation.RuntimeEngineFingerprint != nativeRuntimeEngineFingerprint {
		return evidence{}, fmt.Errorf("TEI-native builtin invocation lost build/runtime engine provenance")
	}
	if err := pruntime.ValidateDeploymentInvocation(
		runtimePlan,
		core.MustCanonicalID("runtime.native_reference"),
		nativeInvocation.PluginInvocation,
	); err != nil {
		return evidence{}, fmt.Errorf("validate TEI-native builtin invocation provenance: %w", err)
	}

	runtimePluginLockFingerprint, err := plugin.EnvironmentLockFingerprint(runtimeLock, loaded.World.Environments[0].ID)
	if err != nil {
		return evidence{}, err
	}
	if err := requireCleanRepository(root); err != nil {
		return evidence{}, fmt.Errorf("repository changed during build: %w", err)
	}

	return evidence{
		Schema:                       "tei.reference.plugin-self-build-evidence/v1",
		RepositoryRevision:           revision,
		RepositoryTreeDigest:         treeDigest,
		GoVersion:                    goVersion,
		GoBinaryDigest:               goBinaryDigest,
		GoToolchainDigest:            goToolchainDigest,
		PackageBuilderFirstDigest:   packageBuilderFirstDigest,
		PackageBuilderSecondDigest:  packageBuilderSecondDigest,
		PackageBuilderReproducible:  true,
		PackageBuildPlan:             packagePlan,
		PackageBuildPlanFingerprint:  packagePlanFingerprint,
		PackageFirstArtifactDigest:   packageFirstArtifactDigest,
		PackageSecondArtifactDigest:  packageSecondArtifactDigest,
		PackageReproducible:          true,
		PackageLockedBuild:           packageLockedBuild,
		PackageLockFingerprint:       packageLockFingerprint,
		NativeSourceFingerprint:      nativeSourceFingerprint,
		NativeBuilderFirstDigest:     nativeBuilderFirstDigest,
		NativeBuilderSecondDigest:    nativeBuilderSecondDigest,
		NativeBuilderReproducible:    true,
		NativeBuildPlan:              nativePlan,
		NativeBuildPlanFingerprint:   nativePlanFingerprint,
		NativeFirstArtifactDigest:    nativeFirstArtifactDigest,
		NativeSecondArtifactDigest:   nativeSecondArtifactDigest,
		NativeReproducible:           true,
		NativeLockedBuild:            nativeLockedBuild,
		NativeLockFingerprint:        nativeLockFingerprint,
		NativeDefinitionFingerprint:       plugin.DefinitionFingerprint(nativeRealized),
		NativeRuntimeEngineSourceDigest: nativeRuntimeEngineSourceDigest,
		NativeRuntimeEngine:             nativeRuntimeEngine,
		NativeRuntimeEngineFingerprint:  nativeRuntimeEngineFingerprint,
		NativeInvocation:                nativeInvocation,
		BuildPlan:                    plan,
		BuildPlanFingerprint:         planFingerprint,
		FirstArtifactDigest:          firstDigest,
		SecondArtifactDigest:         secondDigest,
		Reproducible:                 true,
		LockedBuild:                  lockedBuild,
		BuildArtifactLockFingerprint: buildArtifactLockFingerprint,
		RuntimeDefinitionFingerprint: plugin.DefinitionFingerprint(realized),
		RuntimePluginLockFingerprint: runtimePluginLockFingerprint,
		RuntimePluginLock:            runtimeLock,
	}, nil
}

func requireValidWorld(label string, world ir.World) error {
	diagnostics := validation.Validate(world)
	var failures []string
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != validation.Error {
			continue
		}
		failures = append(failures, diagnostic.Code+": "+diagnostic.Message)
	}
	if len(failures) == 0 {
		return nil
	}
	sort.Strings(failures)
	return fmt.Errorf("%s validation failed: %s", label, strings.Join(failures, "; "))
}



func hashSourceFiles(repo string, paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("source closure must contain at least one file")
	}
	ordered := append([]string(nil), paths...)
	sort.Strings(ordered)
	sum := sha256.New()
	seen := map[string]struct{}{}
	for _, rel := range ordered {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" {
			return "", fmt.Errorf("source closure contains empty path")
		}
		if _, ok := seen[rel]; ok {
			return "", fmt.Errorf("source closure contains duplicate path %s", rel)
		}
		seen[rel] = struct{}{}
		path := filepath.Join(repo, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("source closure symlink is not supported: %s", rel)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("source closure member is not a regular file: %s", rel)
		}
		if _, err := fmt.Fprintf(sum, "%s\x00", rel); err != nil {
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

func requireNoExternalModules(repo, goPath string, packages ...string) error {
	if len(packages) == 0 {
		return fmt.Errorf("at least one Go package is required for dependency closure validation")
	}
	stateDir, err := os.MkdirTemp("", "tei-plugin-self-build-list-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stateDir)

	args := []string{
		"list",
		"-mod=readonly",
		"-deps",
		"-f",
		"{{if .Module}}{{.Module.Path}} {{.Module.Version}}{{end}}",
	}
	args = append(args, packages...)
	cmd := exec.Command(goPath, args...)
	cmd.Dir = repo
	cmd.Env = buildEnvironment(os.Environ(), stateDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("resolve Reference Go dependency closure: %w: %s", err, strings.TrimSpace(string(out)))
	}
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] != "tei" {
			return fmt.Errorf("Reference Go package has unresolved external build dependency %q", line)
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

func buildReferencePlugin(repo, goPath, output, stateDir string) error {
	if err := buildGoPackage(repo, goPath, referencePackage, output, stateDir); err != nil {
		return fmt.Errorf("build reference process plugin: %w", err)
	}
	return nil
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

func runPluginPackageBuilder(repo, builderPath, planPath, output string) error {
	cmd := exec.Command(
		builderPath,
		"-plugin",
		filepath.Join(repo, filepath.FromSlash(packageDescriptor)),
		"-plan",
		planPath,
		"-out",
		output,
	)
	cmd.Dir = repo
	cmd.Env = []string{}
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Plugin package builder: %w: %s", err, strings.TrimSpace(string(combined)))
	}
	return nil
}

func runNativePluginBuilder(repo, builderPath, output string) error {
	cmd := exec.Command(
		builderPath,
		"-implementation",
		filepath.Join(repo, filepath.FromSlash(nativeImplementation)),
		"-plugin",
		filepath.Join(repo, filepath.FromSlash(nativeDescriptor)),
		"-out",
		output,
	)
	cmd.Dir = repo
	cmd.Env = []string{}
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("native Plugin builder: %w: %s", err, strings.TrimSpace(string(combined)))
	}
	return nil
}

func buildEnvironment(base []string, stateDir string) []string {
	out := make([]string, 0, len(base)+12)
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
		"GOSUMDB=off",
		"GOTOOLCHAIN=local",
		"GOVCS=*:off",
		"GOWORK=off",
	)
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

func exactImplementation(sources []implementation.Source, id, version string) (implementation.Source, error) {
	for _, source := range sources {
		if string(source.Plugin.ID) == id && source.Plugin.Version == version {
			return source, nil
		}
	}
	return implementation.Source{}, fmt.Errorf("plugin implementation %s@%s not found", id, version)
}

func replacePluginDefinition(definitions []plugin.Definition, replacement plugin.Definition) {
	for i := range definitions {
		if definitions[i].ID == replacement.ID && definitions[i].Version == replacement.Version {
			definitions[i] = replacement
			return
		}
	}
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
