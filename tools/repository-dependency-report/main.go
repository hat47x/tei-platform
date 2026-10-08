// Command repository-dependency-report emits a deterministic source-level Go
// package dependency inventory. It does not resolve build tags or runtime use.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type pkgInfo struct {
	Path               string `json:"path"`
	SourceFiles        int    `json:"sourceFiles"`
	DirectDependencies int    `json:"directDependencies"`
	DirectDependents   int    `json:"directDependents"`
}

type directEdge struct {
	From        string   `json:"from"`
	To          string   `json:"to"`
	SourceFiles []string `json:"sourceFiles"`
}

type report struct {
	Schema                string       `json:"schema"`
	Module                string       `json:"module"`
	Scope                 string       `json:"scope"`
	ProductionSourceFiles int          `json:"productionSourceFiles"`
	Packages              []pkgInfo    `json:"packages"`
	Dependencies          []directEdge `json:"dependencies"`
}

func modulePath(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	var module string
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "module" {
			continue
		}
		if module != "" {
			return "", fmt.Errorf("duplicate go.mod module directive")
		}
		module = fields[1]
		if strings.HasPrefix(module, "\"") {
			module, err = strconv.Unquote(module)
			if err != nil {
				return "", fmt.Errorf("invalid go.mod module directive: %w", err)
			}
		}
	}
	if module == "" {
		return "", fmt.Errorf("go.mod module directive is missing")
	}
	return module, nil
}

// collect reports direct source imports, including alternate-platform Go files.
// Test files and fixture directories are not part of the production graph.
func collect(root string) (report, error) {
	module, err := modulePath(root)
	if err != nil {
		return report{}, err
	}
	result := report{Schema: "tei.repository.direct-import-graph/v1", Module: module, Scope: "internal production Go source; build tags not resolved", Packages: []pkgInfo{}, Dependencies: []directEdge{}}
	fileCounts := map[string]int{}
	edgeFiles := map[string]map[string]map[string]bool{}
	err = filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != filepath.Join(root, "internal") {
				name := d.Name()
				if name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
					return fs.SkipDir
				}
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_") || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		syntax, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}
		dir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		from := filepath.ToSlash(dir)
		relative = filepath.ToSlash(relative)
		fileCounts[from]++
		result.ProductionSourceFiles++
		for _, spec := range syntax.Imports {
			imported, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return fmt.Errorf("import %s: %w", relative, err)
			}
			if !strings.HasPrefix(imported, module+"/internal/") {
				continue
			}
			to := strings.TrimPrefix(imported, module+"/")
			if edgeFiles[from] == nil {
				edgeFiles[from] = map[string]map[string]bool{}
			}
			if edgeFiles[from][to] == nil {
				edgeFiles[from][to] = map[string]bool{}
			}
			edgeFiles[from][to][relative] = true
		}
		return nil
	})
	if err != nil {
		return report{}, err
	}
	if result.ProductionSourceFiles == 0 {
		return report{}, fmt.Errorf("no production Go files under internal/")
	}
	depCounts, dependentCounts := map[string]int{}, map[string]int{}
	for from, targets := range edgeFiles {
		for to, sourceSet := range targets {
			sources := make([]string, 0, len(sourceSet))
			for s := range sourceSet {
				sources = append(sources, s)
			}
			sort.Strings(sources)
			result.Dependencies = append(result.Dependencies, directEdge{From: from, To: to, SourceFiles: sources})
			depCounts[from]++
			dependentCounts[to]++
		}
	}
	for _, dep := range result.Dependencies {
		if fileCounts[dep.To] == 0 {
			return report{}, fmt.Errorf("missing first-party package %s imported from %s", dep.To, dep.SourceFiles[0])
		}
	}
	sort.Slice(result.Dependencies, func(i, j int) bool {
		if result.Dependencies[i].From != result.Dependencies[j].From {
			return result.Dependencies[i].From < result.Dependencies[j].From
		}
		return result.Dependencies[i].To < result.Dependencies[j].To
	})
	for path, count := range fileCounts {
		result.Packages = append(result.Packages, pkgInfo{Path: path, SourceFiles: count, DirectDependencies: depCounts[path], DirectDependents: dependentCounts[path]})
	}
	sort.Slice(result.Packages, func(i, j int) bool { return result.Packages[i].Path < result.Packages[j].Path })
	return result, nil
}

func main() {
	root := flag.String("root", ".", "repository root containing go.mod and internal/")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	r, err := collect(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
