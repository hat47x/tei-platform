// Package main records the source-level Go dependency closure of one target
// entry point. This is evidence, not an authority for runtime packaging.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type listedModule struct {
	Path string `json:"Path"`
	Main bool   `json:"Main"`
}

type listedPackage struct {
	ImportPath   string                 `json:"ImportPath"`
	Dir          string                 `json:"Dir"`
	Name         string                 `json:"Name"`
	Module       *listedModule          `json:"Module"`
	GoFiles      []string               `json:"GoFiles"`
	CgoFiles     []string               `json:"CgoFiles"`
	EmbedFiles   []string               `json:"EmbedFiles"`
	CFiles       []string               `json:"CFiles"`
	CXXFiles     []string               `json:"CXXFiles"`
	MFiles       []string               `json:"MFiles"`
	HFiles       []string               `json:"HFiles"`
	FFiles       []string               `json:"FFiles"`
	SFiles       []string               `json:"SFiles"`
	SwigFiles    []string               `json:"SwigFiles"`
	SwigCXXFiles []string               `json:"SwigCXXFiles"`
	SysoFiles    []string               `json:"SysoFiles"`
	Imports      []string               `json:"Imports"`
	Error        *struct{ Err string }  `json:"Error"`
	Incomplete   bool                   `json:"Incomplete"`
	DepsErrors   []struct{ Err string } `json:"DepsErrors"`
}

type SourceInput struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type SourcePackage struct {
	ImportPath      string        `json:"importPath"`
	GoFiles         []string      `json:"goFiles,omitempty"`
	CgoFiles        []string      `json:"cgoFiles,omitempty"`
	AuxiliaryInputs []SourceInput `json:"auxiliaryInputs,omitempty"`
	Imports         []string      `json:"imports,omitempty"`
}

// auxiliaryInputs records source and embedded inputs not represented by
// GoFiles/CgoFiles. These Go-list fields are not an exhaustive inventory of
// external compiler, linker, pkg-config, or C-include dependencies.
func (p listedPackage) auxiliaryInputs() []SourceInput {
	var inputs []SourceInput
	for _, group := range []struct {
		kind string
		files []string
	}{
		{"embed", p.EmbedFiles},
		{"c", p.CFiles},
		{"cxx", p.CXXFiles},
		{"objective-c", p.MFiles},
		{"header", p.HFiles},
		{"fortran", p.FFiles},
		{"assembly", p.SFiles},
		{"swig", p.SwigFiles},
		{"swig-cxx", p.SwigCXXFiles},
		{"syso", p.SysoFiles},
	} {
		for _, file := range group.files {
			inputs = append(inputs, SourceInput{Kind: group.kind, Path: file})
		}
	}
	sort.Slice(inputs, func(i, j int) bool {
		if inputs[i].Kind == inputs[j].Kind {
			return inputs[i].Path < inputs[j].Path
		}
		return inputs[i].Kind < inputs[j].Kind
	})
	return inputs
}

type Report struct {
	Entry          string          `json:"entry"`
	SourceRevision string          `json:"sourceRevision"`
	Module         string          `json:"module"`
	GoVersion      string          `json:"goVersion"`
	GOOS           string          `json:"goos"`
	GOARCH         string          `json:"goarch"`
	CGOEnabled     string          `json:"cgoEnabled"`
	GoFlags        string          `json:"goFlags"`
	BuildTags      string          `json:"buildTags,omitempty"`
	Packages       []SourcePackage `json:"packages"`
}

// parseGoList consumes the documented stream of JSON package objects returned
// by go list -deps -json. It retains only compiled Go packages in the main
// module; it neither infers symbol retention nor discards unresolved edges.
func parseGoList(reader io.Reader, entry, tags string) (Report, error) {
	dec := json.NewDecoder(reader)
	seen := make(map[string]listedPackage)
	mainModule := ""
	entryFound := false
	for {
		var p listedPackage
		err := dec.Decode(&p)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Report{}, fmt.Errorf("decode go list: %w", err)
		}
		if p.ImportPath == "" {
			return Report{}, fmt.Errorf("go list has empty package import path")
		}
		if p.Error != nil {
			return Report{}, fmt.Errorf("go list package %s: %s", p.ImportPath, p.Error.Err)
		}
		if p.Incomplete {
			return Report{}, fmt.Errorf("go list package %s is incomplete", p.ImportPath)
		}
		if len(p.DepsErrors) != 0 {
			return Report{}, fmt.Errorf("go list package %s has unresolved dependencies", p.ImportPath)
		}
		if _, ok := seen[p.ImportPath]; ok {
			return Report{}, fmt.Errorf("duplicate go list package %s", p.ImportPath)
		}
		seen[p.ImportPath] = p
		if p.Module != nil && p.Module.Main {
			if mainModule != "" && mainModule != p.Module.Path {
				return Report{}, fmt.Errorf("multiple main modules in go list")
			}
			mainModule = p.Module.Path
		}
		if p.ImportPath == entry {
			if p.Module == nil || !p.Module.Main {
				return Report{}, fmt.Errorf("entry %s is not in the main module", entry)
			}
			entryFound = true
		}
	}
	if !entryFound || mainModule == "" {
		return Report{}, fmt.Errorf("entry package %s or main module not found", entry)
	}
	result := Report{Entry: entry, Module: mainModule, BuildTags: tags, Packages: []SourcePackage{}}
	for _, p := range seen {
		if p.Module == nil || !p.Module.Main || p.Module.Path != mainModule {
			continue
		}
		if len(p.GoFiles)+len(p.CgoFiles) == 0 {
			return Report{}, fmt.Errorf("main-module package %s has no compiled Go source", p.ImportPath)
		}
		files := append([]string(nil), p.GoFiles...)
		cgoFiles := append([]string(nil), p.CgoFiles...)
		imports := append([]string(nil), p.Imports...)
		sort.Strings(files)
		sort.Strings(cgoFiles)
		sort.Strings(imports)
		result.Packages = append(result.Packages, SourcePackage{ImportPath: p.ImportPath, GoFiles: files, CgoFiles: cgoFiles, AuxiliaryInputs: p.auxiliaryInputs(), Imports: imports})
	}
	sort.Slice(result.Packages, func(i, j int) bool { return result.Packages[i].ImportPath < result.Packages[j].ImportPath })
	if len(result.Packages) == 0 {
		return Report{}, fmt.Errorf("no compiled main-module package")
	}
	for _, p := range result.Packages {
		for _, dep := range p.Imports {
			// import "C" is a cgo pseudo-import, not a Go package emitted
			// by go list -deps -json. It is valid only for cgo sources.
			if dep == "C" && len(p.CgoFiles) != 0 {
				continue
			}
			if _, ok := seen[dep]; !ok {
				return Report{}, fmt.Errorf("go list closure missing import %s from %s", dep, p.ImportPath)
			}
		}
	}
	return result, nil
}

// validateTrackedSources checks that every compiled main-module Go/Cgo file
// reported by go list is in the Git index. git status --untracked-files=all
// alone does not expose ignored source files that Go could still compile.
func validateTrackedSources(raw []byte, root string, tracked map[string]struct{}) error {
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("cannot resolve repository root: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	for {
		var p listedPackage
		if err := dec.Decode(&p); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("decode source inventory: %w", err)
		}
		if p.Module == nil || !p.Module.Main {
			continue
		}
		if !filepath.IsAbs(p.Dir) {
			return fmt.Errorf("package %s has no absolute source directory", p.ImportPath)
		}
		relDir, err := filepath.Rel(root, p.Dir)
		if err != nil || relDir == ".." || strings.HasPrefix(relDir, ".."+string(filepath.Separator)) || filepath.IsAbs(relDir) {
			return fmt.Errorf("main-module package %s is outside repository", p.ImportPath)
		}
		inputFiles := append(append([]string(nil), p.GoFiles...), p.CgoFiles...)
		for _, auxiliary := range p.auxiliaryInputs() {
			inputFiles = append(inputFiles, auxiliary.Path)
		}
		for _, file := range inputFiles {
			// Embedded files may contain nested paths, unlike GoFiles. Keep
			// that relative path but reject traversal and noncanonical forms.
			clean := path.Clean(file)
			if file == "" || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") ||
				strings.HasPrefix(file, "/") || strings.Contains(file, "\\") || clean != file {
				return fmt.Errorf("package %s has unsafe source input path %q", p.ImportPath, file)
			}
			rel := filepath.ToSlash(filepath.Join(relDir, file))
			if _, ok := tracked[rel]; !ok {
				return fmt.Errorf("compiled source %q in package %s is not tracked by Git", rel, p.ImportPath)
			}
			// A Git-tracked symlink can point at untracked, ignored, or
			// out-of-tree bytes. Refuse it even when git status is clean.
			full := filepath.Join(root, filepath.FromSlash(rel))
			resolved, err := filepath.EvalSymlinks(full)
			if err != nil {
				return fmt.Errorf("cannot resolve compiled source %q: %w", rel, err)
			}
			// Compare with the canonical root: /var and /private/var can
			// denote the same temporary checkout on some hosts.
			expected := filepath.Join(canonicalRoot, filepath.FromSlash(rel))
			if filepath.Clean(resolved) != filepath.Clean(expected) {
				return fmt.Errorf("compiled source %q traverses a symlink below repository root", rel)
			}
			info, err := os.Stat(full)
			if err != nil || !info.Mode().IsRegular() {
				return fmt.Errorf("compiled source %q is not a regular file", rel)
			}
		}
	}
}

func trackedGitFiles() (map[string]struct{}, error) {
	output, err := exec.Command("git", "ls-files", "--cached", "-z").Output()
	if err != nil {
		return nil, fmt.Errorf("git tracked file inventory unavailable: %w", err)
	}
	files := make(map[string]struct{})
	for _, file := range strings.Split(string(output), "\x00") {
		if file != "" {
			files[file] = struct{}{}
		}
	}
	return files, nil
}

func validEntry(entry string) bool {
	if !strings.HasPrefix(entry, "./") || strings.ContainsAny(entry, "*\\ \t\r\n") {
		return false
	}
	for _, segment := range strings.Split(strings.TrimPrefix(entry, "./"), "/") {
		if segment == "" || segment == "." || segment == ".." || segment == "..." {
			return false
		}
	}
	return path.Clean(strings.TrimPrefix(entry, "./")) == strings.TrimPrefix(entry, "./")
}

func repositoryIdentity() (string, error) {
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("git repository root is unavailable: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if filepath.Clean(strings.TrimSpace(string(top))) != filepath.Clean(cwd) {
		return "", fmt.Errorf("run from repository root")
	}
	status, err := exec.Command("git", "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil {
		return "", fmt.Errorf("git status failed: %w", err)
	}
	if len(status) != 0 {
		return "", fmt.Errorf("repository has uncommitted or untracked changes")
	}
	rev, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git revision unavailable: %w", err)
	}
	return strings.TrimSpace(string(rev)), nil
}

func produceEvidence(entry, tags string) (Report, error) {
	if !validEntry(entry) {
		return Report{}, fmt.Errorf("entry must be one literal repository-relative Go package (./path/to/entry)")
	}
	revision, err := repositoryIdentity()
	if err != nil {
		return Report{}, err
	}
	args := []string{"list", "-deps", "-json"}
	if tags != "" {
		args = append(args, "-tags="+tags)
	}
	args = append(args, entry)
	goList := exec.Command("go", args...)
	// GOFLAGS/GOWORK changes alter the package graph; use an isolated module
	// build profile instead of inheriting user workspace and hidden flags.
	goList.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
	var stderr bytes.Buffer
	goList.Stderr = &stderr
	output, err := goList.Output()
	if err != nil {
		return Report{}, fmt.Errorf("go list failed: %w: %s", err, stderr.String())
	}
	// Resolve the entry import path from a single-package go list invocation.
	entryArgs := []string{"list", "-f", "{{.ImportPath}}"}
	if tags != "" {
		entryArgs = append(entryArgs, "-tags="+tags)
	}
	entryCmd := exec.Command("go", append(entryArgs, entry)...)
	entryCmd.Env = goList.Env
	entryOut, err := entryCmd.Output()
	if err != nil {
		return Report{}, fmt.Errorf("resolve entry package: %w", err)
	}
	report, err := parseGoList(bytes.NewReader(output), strings.TrimSpace(string(entryOut)), tags)
	if err != nil {
		return Report{}, err
	}
	tracked, err := trackedGitFiles()
	if err != nil {
		return Report{}, err
	}
	root, err := os.Getwd()
	if err != nil {
		return Report{}, err
	}
	if err := validateTrackedSources(output, root, tracked); err != nil {
		return Report{}, err
	}
	envCmd := exec.Command("go", "env", "-json", "GOVERSION", "GOOS", "GOARCH", "CGO_ENABLED", "GOFLAGS")
	envCmd.Env = goList.Env
	envOut, err := envCmd.Output()
	if err != nil {
		return Report{}, fmt.Errorf("go env failed: %w", err)
	}
	var env struct{ GOVERSION, GOOS, GOARCH, CGO_ENABLED, GOFLAGS string }
	if err := json.Unmarshal(envOut, &env); err != nil {
		return Report{}, fmt.Errorf("decode go env: %w", err)
	}
	report.GoVersion, report.GOOS, report.GOARCH, report.CGOEnabled, report.GoFlags = env.GOVERSION, env.GOOS, env.GOARCH, env.CGO_ENABLED, env.GOFLAGS
	revisionAfter, err := repositoryIdentity()
	if err != nil {
		return Report{}, fmt.Errorf("repository changed during observation: %w", err)
	}
	if revisionAfter != revision {
		return Report{}, fmt.Errorf("revision changed during observation")
	}
	report.SourceRevision = revision
	return report, nil
}
