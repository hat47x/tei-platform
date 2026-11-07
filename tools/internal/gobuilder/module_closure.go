package gobuilder

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// ModuleClosureClassification is the preflight result for one Go target.
// The existing ResolveSourceClosure rejects every external module root. This
// preflight runs first so that "the target needs an external module that was
// not supplied offline" is reported as its own fact instead of collapsing into
// a generic closure failure. It grants no build, artifact or acceptance
// authority.
const (
	ModuleClosureSupported    = "supported"
	ModuleClosureExternal     = "external-unresolved"
	ModuleClosureMissing      = "missing-module"
	ModuleClosureUnknown      = "unknown"
	moduleClosureSchemaV0     = "tei.reference.go-module-closure-preflight/v0"
	moduleClosureEvidenceNote = "offline go list -e observation; not a build attempt"
)

// ModuleRoot records one module identity selected by the compiler resolver.
// Dir is the physical directory the resolver reported; it is machine
// specific and must not be treated as a portable content identity.
type ModuleRoot struct {
	Path             string `json:"path"`
	Version          string `json:"version,omitempty"`
	Dir              string `json:"dir,omitempty"`
	Resolved         bool   `json:"resolved"`
	SelectedPackages int    `json:"selectedPackages"`
}

// UnresolvedPackage is one non-standard package the resolver reported as
// failing, with the exact text the resolver supplied. Classification of this
// text is deliberately conservative.
type UnresolvedPackage struct {
	ImportPath  string   `json:"importPath"`
	Position    string   `json:"position,omitempty"`
	Error       string   `json:"error"`
	ImportStack []string `json:"importStack,omitempty"`
}

// FileDigest records one module-definition input that is read but not selected
// as compiler source.
type FileDigest struct {
	Path    string `json:"path"`
	Present bool   `json:"present"`
	Digest  string `json:"digest,omitempty"`
}

// ModuleClosurePreflight is the read-only evidence of what the Go resolver
// observed for one package path. Classification is one of the four constants
// above; any observation the resolver did not make stays `unknown`.
type ModuleClosurePreflight struct {
	SchemaVersion          string              `json:"schemaVersion"`
	Nature                 string              `json:"nature"`
	Package                string              `json:"package"`
	Classification         string              `json:"classification"`
	RootModuleDir          string              `json:"rootModuleDir"`
	RootModulePath         string              `json:"rootModulePath,omitempty"`
	GoOS                   string              `json:"goos"`
	GoArch                 string              `json:"goarch"`
	GoExecutableDigest     string              `json:"goExecutableDigest"`
	ResolvedPackages       int                 `json:"resolvedPackages"`
	ModuleRoots            []ModuleRoot        `json:"moduleRoots"`
	Unresolved             []UnresolvedPackage `json:"unresolvedPackages"`
	ModuleFiles            []FileDigest        `json:"moduleFiles"`
	CommandExitCode        int                 `json:"commandExitCode"`
	CommandError           string              `json:"commandError,omitempty"`
	SourceClosureReachable bool                `json:"sourceClosureReachable"`
	Summary                string              `json:"summary"`
}

// IsRootModuleRoot reports whether the preflight identified the target's own
// module root among the selected roots. It is a read of the observation, not a
// qualification of the target.
func (p ModuleClosurePreflight) IsRootModuleRoot() bool {
	return p.RootModulePath != "" && p.RootModuleDir != ""
}

// ResolveModuleClosurePreflight runs one offline `go list -e` and classifies
// the selected module roots of the target. Separate isolated state directories
// are required for independent attempts. The command inherits the same
// deny-by-default environment as BuildPackage.
func ResolveModuleClosurePreflight(repo, goPath, packagePath, stateDir string) (ModuleClosurePreflight, error) {
	absolute, err := filepath.Abs(repo)
	if err != nil {
		return ModuleClosurePreflight{}, err
	}
	root, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		root = filepath.Clean(absolute)
	}
	goDigest, err := digestSourceFile(goPath)
	if err != nil {
		return ModuleClosurePreflight{}, fmt.Errorf("digest Go executable: %w", err)
	}

	cmd := exec.Command(goPath, "list", "-deps", "-json", "-e", "-mod=readonly", packagePath)
	cmd.Dir = root
	env, err := Environment(os.Environ(), stateDir, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return ModuleClosurePreflight{}, fmt.Errorf("prepare Go module preflight environment: %w", err)
	}
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	exitCode := 0
	commandError := ""
	if runErr != nil {
		commandError = strings.TrimSpace(stderr.String())
		if commandError == "" {
			commandError = runErr.Error()
		}
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	preflight, err := classifyModuleClosure(root, packagePath, goPath, goDigest, stdout.Bytes())
	if err != nil {
		return ModuleClosurePreflight{}, err
	}
	preflight.CommandExitCode = exitCode
	preflight.CommandError = commandError
	// A resolver that exited unsuccessfully may have emitted partial output.
	// Keep the observation but never let it claim a reachable closure.
	if runErr != nil && preflight.Classification != ModuleClosureUnknown {
		preflight.Classification = ModuleClosureUnknown
		preflight.Summary = "go list exited unsuccessfully; partial output cannot be qualified"
	}
	preflight.SourceClosureReachable = preflight.Classification == ModuleClosureSupported && runErr == nil

	moduleFiles, err := digestModuleFiles(root)
	if err != nil {
		return ModuleClosurePreflight{}, err
	}
	preflight.ModuleFiles = moduleFiles
	return preflight, nil
}

// classifyModuleClosure is the pure part of the preflight. It never runs a
// process and therefore can be exercised with recorded resolver output.
func classifyModuleClosure(root, packagePath, goPath, goDigest string, raw []byte) (ModuleClosurePreflight, error) {
	preflight := ModuleClosurePreflight{
		SchemaVersion:      moduleClosureSchemaV0,
		Nature:             moduleClosureEvidenceNote,
		Package:            packagePath,
		RootModuleDir:      root,
		GoOS:               runtime.GOOS,
		GoArch:             runtime.GOARCH,
		GoExecutableDigest: goDigest,
	}
	roots := map[string]*ModuleRoot{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	for {
		var pkg GoListPackage
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return ModuleClosurePreflight{}, fmt.Errorf("decode Go module preflight: %w", err)
		}
		if pkg.Standard {
			continue
		}
		if pkg.Error != nil {
			preflight.Unresolved = append(preflight.Unresolved, UnresolvedPackage{
				ImportPath:  pkg.ImportPath,
				Position:    pkg.Error.Pos,
				Error:       pkg.Error.Err,
				ImportStack: append([]string(nil), pkg.Error.ImportStack...),
			})
			continue
		}
		resolvedModuleDir := ""
		if pkg.Module != nil && pkg.Module.Dir != "" {
			resolvedModuleDir = pkg.Module.Dir
			if evaluated, err := filepath.EvalSymlinks(resolvedModuleDir); err == nil {
				resolvedModuleDir = evaluated
			} else {
				resolvedModuleDir = filepath.Clean(resolvedModuleDir)
			}
		}
		key := pkg.ImportPath
		if pkg.Module != nil && pkg.Module.Path != "" {
			key = pkg.Module.Path + "@" + pkg.Module.Version
		}
		entry, ok := roots[key]
		if !ok {
			entry = &ModuleRoot{Dir: resolvedModuleDir}
			if pkg.Module != nil {
				entry.Path = pkg.Module.Path
				entry.Version = pkg.Module.Version
			}
			roots[key] = entry
		}
		entry.Resolved = true
		entry.SelectedPackages++
		preflight.ResolvedPackages++
	}

	for _, entry := range roots {
		preflight.ModuleRoots = append(preflight.ModuleRoots, *entry)
		if entry.Path != "" && entry.Dir == root && preflight.RootModulePath == "" {
			preflight.RootModulePath = entry.Path
		}
	}
	sort.Slice(preflight.ModuleRoots, func(i, j int) bool {
		return preflight.ModuleRoots[i].Path < preflight.ModuleRoots[j].Path
	})
	sort.Slice(preflight.Unresolved, func(i, j int) bool {
		return preflight.Unresolved[i].ImportPath < preflight.Unresolved[j].ImportPath
	})

	preflight.Classification, preflight.Summary = classifyModuleRoots(root, preflight.ModuleRoots, preflight.Unresolved)
	return preflight, nil
}

// classifyModuleRoots applies a fixed precedence: unknown is retained above
// every other result because an unrecognized failure is not evidence that the
// target is merely short of module content. Missing module content ranks next,
// then a selected external module root, then a supported target module.
func classifyModuleRoots(root string, roots []ModuleRoot, unresolved []UnresolvedPackage) (string, string) {
	for _, pkg := range unresolved {
		if !isModuleSupplyFailure(pkg.Error) {
			return ModuleClosureUnknown, "unrecognized resolver failure is not classified as a module supply gap: " + pkg.ImportPath
		}
	}
	if len(unresolved) > 0 {
		return ModuleClosureMissing, fmt.Sprintf("%d selected package(s) could not be resolved offline", len(unresolved))
	}
	// An empty resolver stream (or one containing only standard-library
	// packages) does not establish that this target selected any package
	// from its own module. Never qualify that absence as supported.
	if len(roots) == 0 {
		return ModuleClosureUnknown, "resolver observed no non-standard package or target module root"
	}
	var external []string
	for _, entry := range roots {
		if entry.Path == "" {
			return ModuleClosureUnknown, "selected package has no module identity"
		}
		if entry.Dir != root {
			external = append(external, entry.Path)
			continue
		}
	}
	if len(external) > 0 {
		sort.Strings(external)
		return ModuleClosureExternal, "selected external module root(s) outside the target module: " + strings.Join(external, ", ")
	}
	return ModuleClosureSupported, "all selected packages belong to the target module and resolved offline"
}

// isModuleSupplyFailure recognizes the resolver messages that mean the exact
// module content was not available from the supplied module set. Anything not
// listed here is left `unknown` rather than guessed.
func isModuleSupplyFailure(errText string) bool {
	lower := strings.ToLower(errText)
	for _, marker := range []string{
		"module lookup disabled",
		"missing go.sum entry",
		"no required module provides",
		"cannot find module providing",
		"module cache",
		"without a go.sum entry",
		"inconsistent vendoring",
		"updates to go.mod needed",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// digestModuleFiles records the module definitions the resolver may have read.
// go.sum is absent for repositories without external modules and is recorded
// as absent rather than as an empty file.
func digestModuleFiles(root string) ([]FileDigest, error) {
	out := make([]FileDigest, 0, 2)
	for _, name := range []string{"go.mod", "go.sum"} {
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				out = append(out, FileDigest{Path: name, Present: false})
				continue
			}
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file", name)
		}
		digest, err := digestModuleFile(path)
		if err != nil {
			return nil, err
		}
		out = append(out, FileDigest{Path: name, Present: true, Digest: digest})
	}
	return out, nil
}

func digestModuleFile(path string) (string, error) {
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
