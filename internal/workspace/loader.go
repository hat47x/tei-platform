package workspace

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/plugin/implementation"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/stdlib"
	"tei/internal/syntax"
)

type SourceLocation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type ManagedFile struct {
	Path       string `json:"path"`
	SourceRoot string `json:"sourceRoot"`
	Extension  string `json:"extension,omitempty"`
}

type SourceContribution struct {
	Subject sourceindex.Subject
	File    string
	Role    sourceindex.Role
}

type Result struct {
	World                       ir.World
	PluginImplementations       []implementation.Source
	PluginImplementationSources map[string]SourceLocation
	Sources                     map[string]SourceLocation
	ManagedFiles                []ManagedFile
	SourceContributions         []SourceContribution
}

func Load(root string) (Result, error) {
	manifestYAML := filepath.Join(root, "tei.workspace.yaml")
	manifestTEI := filepath.Join(root, "tei.workspace.tei")
	var roots []string
	if src, err := os.ReadFile(manifestYAML); err == nil {
		var parseErr error
		roots, parseErr = parseWorkspaceYAML(string(src))
		if parseErr != nil {
			return Result{}, fmt.Errorf("%s: %w", manifestYAML, parseErr)
		}
	} else if src, legacyErr := os.ReadFile(manifestTEI); legacyErr == nil {
		var parseErr error
		roots, parseErr = syntax.NewParser(string(src)).ParseWorkspace()
		if parseErr != nil {
			return Result{}, fmt.Errorf("%s: %w", manifestTEI, parseErr)
		}
	} else {
		return Result{}, fmt.Errorf("TEI workspace manifest not found: expected %s", manifestYAML)
	}
	for _, sourceRoot := range roots {
		if err := validateWorkspaceSourceRoot(sourceRoot); err != nil {
			return Result{}, fmt.Errorf("workspace source root %q: %w", sourceRoot, err)
		}
	}
	// Inventory and verify the physical boundary before any domain loader can
	// read a Source file or parse a symlink to an outside target.
	managedFiles, err := scanManagedFiles(root, roots)
	if err != nil {
		return Result{}, err
	}
	r := Result{Sources: map[string]SourceLocation{}, PluginImplementationSources: map[string]SourceLocation{}, ManagedFiles: managedFiles}
	// TEI standard semantic types are a library prelude, not Micro Kernel kinds.
	r.World.Semantics = append(r.World.Semantics, stdlib.SemanticNamespace())
	for _, sourceRoot := range roots {
		switch sourceRoot {
		case "plugins":
			if err := loadPlugins(root, &r); err != nil {
				return Result{}, err
			}
		case "plugin-implementations":
			if err := loadPluginImplementations(root, &r); err != nil {
				return Result{}, err
			}
		case "implementations":
			return Result{}, fmt.Errorf("workspace source root %q is ambiguous; use plugin-implementations for Plugin implementation Source", sourceRoot)
		case "bundles":
			if err := loadBundles(root, &r); err != nil {
				return Result{}, err
			}
		case "semantic":
			if err := loadSemantics(root, &r); err != nil {
				return Result{}, err
			}
		case "applications":
			if err := loadApplications(root, &r); err != nil {
				return Result{}, err
			}
			if err := loadHeaderDetailContractQueryBindings(root, &r); err != nil {
				return Result{}, err
			}
		case "databases":
			if err := loadDatabases(root, &r); err != nil {
				return Result{}, err
			}
			if err := loadContractQuerySQLRealizations(root, &r); err != nil {
				return Result{}, err
			}
		case "environments":
			if err := loadEnvironments(root, &r); err != nil {
				return Result{}, err
			}
			if err := loadContractQueryExecutorBindings(root, &r); err != nil {
				return Result{}, err
			}
		case "releases":
			if err := loadReleases(root, &r); err != nil {
				return Result{}, err
			}
		default:
			// Other source roots are accepted for forward-compatible workspace layout.
		}
	}
	if err := validatePluginImplementationDefinitions(&r); err != nil {
		return Result{}, err
	}
	Normalize(&r.World)
	return r, nil
}

func parseWorkspaceYAML(src string) ([]string, error) {
	var roots []string
	inSources := false
	for n, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(strings.SplitN(raw, "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "workspace:") {
			continue
		}
		if line == "sources:" {
			inSources = true
			continue
		}
		if inSources && strings.HasPrefix(line, "-") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			if v == "" {
				return nil, fmt.Errorf("line %d: empty source root", n+1)
			}
			roots = append(roots, v)
			continue
		}
		return nil, fmt.Errorf("line %d: unsupported workspace manifest entry %q", n+1, line)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("sources must contain at least one source root")
	}
	return roots, nil
}

// Source roots are portable Workspace-relative directory names. In particular,
// OS-specific separators and drive prefixes must not change which files are
// managed when the same Workspace is opened on another Target.
func validateWorkspaceSourceRoot(sourceRoot string) error {
	if sourceRoot == "" || sourceRoot == "." || !filepath.IsLocal(sourceRoot) ||
		strings.ContainsAny(sourceRoot, "\\\\:") || path.Clean(sourceRoot) != sourceRoot {
		return fmt.Errorf("must be a clean Workspace-relative path")
	}
	return nil
}

// scanManagedFiles inventories native files without granting them semantic
// authority. Missing optional roots remain forward-compatible; unreadable,
// escaped, broken-linked or non-regular source entries fail instead of being
// silently omitted or introduced as Workspace-managed artifacts.
func scanManagedFiles(root string, roots []string) ([]ManagedFile, error) {
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Workspace root: %w", err)
	}
	var out []ManagedFile
	for _, sourceRoot := range roots {
		if err := validateWorkspaceSourceRoot(sourceRoot); err != nil {
			return nil, fmt.Errorf("workspace source root %q: %w", sourceRoot, err)
		}
		base := filepath.Join(root, sourceRoot)
		err := filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) && path == base {
					return nil // an optional source root has not been materialized
				}
				return fmt.Errorf("scan managed source %s: %w", path, walkErr)
			}
			if d.IsDir() {
				return nil
			}
			realPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("resolve managed source %s: %w", path, err)
			}
			rel, err := filepath.Rel(rootReal, realPath)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				return fmt.Errorf("managed source %s resolves outside Workspace", path)
			}
			info, err := os.Stat(realPath)
			if err != nil {
				return fmt.Errorf("stat managed source %s: %w", path, err)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("managed source %s is not a regular file", path)
			}
			out = append(out, ManagedFile{Path: relPath(root, path), SourceRoot: sourceRoot, Extension: strings.ToLower(filepath.Ext(d.Name()))})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func loadPlugins(root string, r *Result) error {
	base := filepath.Join(root, "plugins")
	var files []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || d.Name() != "plugin.tei" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	sort.Strings(files)
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		d, err := syntax.NewParser(string(src)).ParsePlugin()
		if err != nil {
			return sourceErr(path, err)
		}
		r.World.Plugins = append(r.World.Plugins, d)
		r.Sources["plugin:"+string(d.ID)+"@"+d.Version] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
	}
	return nil
}

func loadBundles(root string, r *Result) error {
	base := filepath.Join(root, "bundles")
	var files []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || d.Name() != "bundle.tei" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	sort.Strings(files)
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		b, err := syntax.NewParser(string(src)).ParseBundle()
		if err != nil {
			return sourceErr(path, err)
		}
		r.World.Bundles = append(r.World.Bundles, b)
		r.Sources["bundle:"+string(b.ID)+"@"+b.Version] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
	}
	return nil
}

func loadSemantics(root string, r *Result) error {
	base := filepath.Join(root, "semantic")
	var files []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".tei") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	sort.Strings(files)
	byNamespace := map[core.CanonicalID]int{}
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		ns, err := syntax.NewParser(string(src)).ParseSemanticFile()
		if err != nil {
			return sourceErr(path, err)
		}
		idx, ok := byNamespace[ns.ID]
		if !ok {
			idx = len(r.World.Semantics)
			byNamespace[ns.ID] = idx
			r.World.Semantics = append(r.World.Semantics, semantic.Namespace{ID: ns.ID})
			r.Sources["semantic:"+string(ns.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
		}
		r.World.Semantics[idx].Types = append(r.World.Semantics[idx].Types, ns.Types...)
		r.World.Semantics[idx].Data = append(r.World.Semantics[idx].Data, ns.Data...)
		r.World.Semantics[idx].Capabilities = append(r.World.Semantics[idx].Capabilities, ns.Capabilities...)
		r.World.Semantics[idx].Validations = append(r.World.Semantics[idx].Validations, ns.Validations...)
		for _, typ := range ns.Types {
			r.Sources["semantic-type:"+string(typ.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
		}
		for _, d := range ns.Data {
			r.Sources["semantic-data:"+string(d.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
			for _, a := range d.Attributes {
				r.Sources["semantic-attribute:"+string(a.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
			}
			for _, relation := range d.Relations {
				r.Sources["semantic-relation:"+string(relation.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
			}
		}
		for _, c := range ns.Capabilities {
			r.Sources["semantic-capability:"+string(c.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
		}
		for _, validation := range ns.Validations {
			r.Sources["semantic-validation:"+string(validation.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
		}
	}
	return nil
}

func loadApplications(root string, r *Result) error {
	manifests, err := globSorted(filepath.Join(root, "applications", "*", "application.tei"))
	if err != nil {
		return err
	}
	for _, path := range manifests {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		a, err := syntax.NewParser(string(src)).ParseApplication()
		if err != nil {
			return sourceErr(path, err)
		}
		r.Sources["application:"+string(a.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
		unitDir := filepath.Dir(path)
		patternFiles, _ := globSorted(filepath.Join(unitDir, "patterns", "*.tei"))
		for _, patternPath := range patternFiles {
			patternSource, err := os.ReadFile(patternPath)
			if err != nil {
				return err
			}
			pattern, err := syntax.NewParser(string(patternSource)).ParseApplicationPattern()
			if err != nil {
				return sourceErr(patternPath, err)
			}
			a.Patterns = append(a.Patterns, pattern)
			r.Sources["application-pattern:"+string(pattern.ID)] = SourceLocation{File: relPath(root, patternPath), Line: 1, Column: 1}
		}
		r.World.Applications = append(r.World.Applications, a)
		qfiles, _ := globSorted(filepath.Join(unitDir, "queries", "*.tei"))
		for _, qpath := range qfiles {
			qsrc, err := os.ReadFile(qpath)
			if err != nil {
				return err
			}
			authoring, err := syntax.NewParser(string(qsrc)).ParseQueryAuthoring(a.ID)
			if err != nil {
				return sourceErr(qpath, err)
			}
			q := authoring.Value
			if authoring.NativeSQLPath != "" {
				nativePath, err := resolveNativeSQLFile(root, qpath, authoring.NativeSQLPath)
				if err != nil {
					return fmt.Errorf("%s: SQL file %q: %w", qpath, authoring.NativeSQLPath, err)
				}
				sqlSource, err := os.ReadFile(nativePath)
				if err != nil {
					return fmt.Errorf("%s: read SQL file %q: %w", qpath, authoring.NativeSQLPath, err)
				}
				q.Source = strings.TrimSpace(string(sqlSource))
				r.SourceContributions = append(r.SourceContributions, SourceContribution{
					Subject: sourceindex.Subject{Kind: "query", ID: string(q.ID)},
					File:    relPath(root, nativePath),
					Role:    sourceindex.RoleAuthoring,
				})
			}
			r.World.Queries = append(r.World.Queries, q)
			r.Sources["query:"+string(q.ID)] = SourceLocation{File: relPath(root, qpath), Line: 1, Column: 1}
		}
		bfiles, _ := globSorted(filepath.Join(unitDir, "realizations", "*.tei"))
		for _, bpath := range bfiles {
			bsrc, err := os.ReadFile(bpath)
			if err != nil {
				return err
			}
			b, err := syntax.NewParser(string(bsrc)).ParseRealization()
			if err != nil {
				return sourceErr(bpath, err)
			}
			b.Application = a.ID
			r.World.Bindings = append(r.World.Bindings, b)
			r.Sources["binding:"+b.Identity()] = SourceLocation{File: relPath(root, bpath), Line: 1, Column: 1}
		}
	}
	return nil
}

func loadDatabases(root string, r *Result) error {
	manifests, err := globSorted(filepath.Join(root, "databases", "*", "database.tei"))
	if err != nil {
		return err
	}
	for _, path := range manifests {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		d, err := syntax.NewParser(string(src)).ParseDatabase()
		if err != nil {
			return sourceErr(path, err)
		}
		contractFiles, _ := globSorted(filepath.Join(filepath.Dir(path), "contracts", "*.tei"))
		for _, cpath := range contractFiles {
			csrc, err := os.ReadFile(cpath)
			if err != nil {
				return err
			}
			c, err := syntax.NewParser(string(csrc)).ParseContract()
			if err != nil {
				return sourceErr(cpath, err)
			}
			d.Contracts = append(d.Contracts, c)
			contractSource := SourceLocation{File: relPath(root, cpath), Line: 1, Column: 1}
			r.Sources["database-contract:"+string(c.ID)] = contractSource
			for _, query := range c.Queries {
				r.Sources["database-contract-query:"+string(c.ID)+"/"+string(query.Name)] = contractSource
			}
		}
		persistenceFiles, _ := globSorted(filepath.Join(filepath.Dir(path), "persistence", "*.tei"))
		for _, ppath := range persistenceFiles {
			psrc, err := os.ReadFile(ppath)
			if err != nil {
				return err
			}
			m, err := syntax.NewParser(string(psrc)).ParsePersistence()
			if err != nil {
				return sourceErr(ppath, err)
			}
			r.World.Persistence = append(r.World.Persistence, m)
			r.Sources["persistence:"+string(m.ID)] = SourceLocation{File: relPath(root, ppath), Line: 1, Column: 1}
		}
		providerPath := filepath.Join(filepath.Dir(path), "migration-provider.tei")
		if psrc, err := os.ReadFile(providerPath); err == nil {
			cfg, err := syntax.NewParser(string(psrc)).ParseMigrationProvider()
			if err != nil {
				return sourceErr(providerPath, err)
			}
			r.World.MigrationProviders = append(r.World.MigrationProviders, cfg)
			r.Sources["migration-provider:"+string(cfg.Database)] = SourceLocation{File: relPath(root, providerPath), Line: 1, Column: 1}
		} else if !os.IsNotExist(err) {
			return err
		}
		migrationFiles, _ := globSorted(filepath.Join(filepath.Dir(path), "migrations", "*", "migration.tei"))
		for _, mpath := range migrationFiles {
			msrc, err := os.ReadFile(mpath)
			if err != nil {
				return err
			}
			m, err := syntax.NewParser(string(msrc)).ParseMigration()
			if err != nil {
				return sourceErr(mpath, err)
			}
			r.World.Migrations = append(r.World.Migrations, m)
			r.Sources["migration:"+string(m.Database)+"/"+m.ID] = SourceLocation{File: relPath(root, mpath), Line: 1, Column: 1}
		}
		r.World.Databases = append(r.World.Databases, d)
		r.Sources["database:"+string(d.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
	}
	return nil
}

func loadReleases(root string, r *Result) error {
	appFiles, err := globSorted(filepath.Join(root, "releases", "applications", "*", "*.tei"))
	if err != nil {
		return err
	}
	for _, path := range appFiles {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := syntax.NewParser(string(src)).ParseApplicationRelease()
		if err != nil {
			return sourceErr(path, err)
		}
		r.World.ApplicationReleases = append(r.World.ApplicationReleases, rel)
		r.Sources["application-release:"+string(rel.Application)+"/"+rel.Version] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
	}

	dbFiles, err := globSorted(filepath.Join(root, "releases", "databases", "*", "*.tei"))
	if err != nil {
		return err
	}
	for _, path := range dbFiles {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := syntax.NewParser(string(src)).ParseDatabaseRelease()
		if err != nil {
			return sourceErr(path, err)
		}
		r.World.DatabaseReleases = append(r.World.DatabaseReleases, rel)
		r.Sources["database-release:"+string(rel.Database)+"/"+rel.Version] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
	}

	solutionFiles, err := globSorted(filepath.Join(root, "releases", "solutions", "*", "*.tei"))
	if err != nil {
		return err
	}
	for _, path := range solutionFiles {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := syntax.NewParser(string(src)).ParseSolutionRelease()
		if err != nil {
			return sourceErr(path, err)
		}
		r.World.SolutionReleases = append(r.World.SolutionReleases, rel)
		r.Sources["solution-release:"+string(rel.ID)+"/"+rel.Version] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
	}
	return nil
}

func loadEnvironments(root string, r *Result) error {
	manifests, err := globSorted(filepath.Join(root, "environments", "*", "environment.tei"))
	if err != nil {
		return err
	}
	for _, path := range manifests {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		e, err := syntax.NewParser(string(src)).ParseEnvironment()
		if err != nil {
			return sourceErr(path, err)
		}
		r.World.Environments = append(r.World.Environments, e)
		r.Sources["environment:"+string(e.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
	}
	return nil
}

func resolveNativeSQLFile(root, declarationPath, rawRef string) (string, error) {
	ref := strings.TrimSpace(rawRef)
	if ref == "" {
		return "", fmt.Errorf("path is empty")
	}
	ref = strings.ReplaceAll(ref, "\\", "/")
	refPath := filepath.FromSlash(ref)
	if filepath.IsAbs(refPath) || filepath.VolumeName(refPath) != "" {
		return "", fmt.Errorf("path must be relative to the Query declaration")
	}
	if strings.ToLower(filepath.Ext(refPath)) != ".sql" {
		return "", fmt.Errorf("path must refer to a .sql file")
	}

	candidate := filepath.Clean(filepath.Join(filepath.Dir(declarationPath), refPath))
	logicalRel, err := filepath.Rel(root, candidate)
	if err != nil || logicalRel == ".." || strings.HasPrefix(logicalRel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes the Workspace")
	}

	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve Workspace root: %w", err)
	}
	candidateReal, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", err
	}
	realRel, err := filepath.Rel(rootReal, candidateReal)
	if err != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path resolves outside the Workspace")
	}

	info, err := os.Stat(candidateReal)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("path does not refer to a regular file")
	}
	return candidate, nil
}

func globSorted(pattern string) ([]string, error) {
	xs, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(xs)
	return xs, nil
}
func relPath(root, path string) string {
	x, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(x)
}
func sourceErr(path string, err error) error { return wrapSourceError(path, err) }

func Normalize(w *ir.World) {
	for i := range w.Plugins {
		w.Plugins[i] = plugin.NormalizeDefinition(w.Plugins[i])
	}
	sort.Slice(w.Plugins, func(i, j int) bool {
		if w.Plugins[i].ID == w.Plugins[j].ID {
			return w.Plugins[i].Version < w.Plugins[j].Version
		}
		return w.Plugins[i].ID < w.Plugins[j].ID
	})
	for i := range w.Bundles {
		sort.Slice(w.Bundles[i].Plugins, func(a, b int) bool { return w.Bundles[i].Plugins[a].Plugin < w.Bundles[i].Plugins[b].Plugin })
	}
	sort.Slice(w.Bundles, func(i, j int) bool {
		if w.Bundles[i].ID == w.Bundles[j].ID {
			return w.Bundles[i].Version < w.Bundles[j].Version
		}
		return w.Bundles[i].ID < w.Bundles[j].ID
	})
	sort.Slice(w.Semantics, func(i, j int) bool { return w.Semantics[i].ID < w.Semantics[j].ID })
	for i := range w.Semantics {
		sort.Slice(w.Semantics[i].Types, func(a, b int) bool { return w.Semantics[i].Types[a].ID < w.Semantics[i].Types[b].ID })
		sort.Slice(w.Semantics[i].Data, func(a, b int) bool { return w.Semantics[i].Data[a].ID < w.Semantics[i].Data[b].ID })
		for j := range w.Semantics[i].Data {
			sort.Slice(w.Semantics[i].Data[j].Attributes, func(a, b int) bool {
				return w.Semantics[i].Data[j].Attributes[a].ID < w.Semantics[i].Data[j].Attributes[b].ID
			})
			sort.Slice(w.Semantics[i].Data[j].Relations, func(a, b int) bool {
				return w.Semantics[i].Data[j].Relations[a].ID < w.Semantics[i].Data[j].Relations[b].ID
			})
		}
		for j := range w.Semantics[i].Capabilities {
			sort.Slice(w.Semantics[i].Capabilities[j].Outcomes, func(a, b int) bool {
				return w.Semantics[i].Capabilities[j].Outcomes[a].Name < w.Semantics[i].Capabilities[j].Outcomes[b].Name
			})
		}
		sort.Slice(w.Semantics[i].Capabilities, func(a, b int) bool { return w.Semantics[i].Capabilities[a].ID < w.Semantics[i].Capabilities[b].ID })
		sort.Slice(w.Semantics[i].Validations, func(a, b int) bool { return w.Semantics[i].Validations[a].ID < w.Semantics[i].Validations[b].ID })
	}
	sort.Slice(w.Applications, func(i, j int) bool { return w.Applications[i].ID < w.Applications[j].ID })
	sort.Slice(w.Databases, func(i, j int) bool { return w.Databases[i].ID < w.Databases[j].ID })
	sort.Slice(w.Persistence, func(i, j int) bool { return w.Persistence[i].ID < w.Persistence[j].ID })
	sort.Slice(w.MigrationProviders, func(i, j int) bool { return w.MigrationProviders[i].Database < w.MigrationProviders[j].Database })
	sort.Slice(w.ApplicationReleases, func(i, j int) bool {
		if w.ApplicationReleases[i].Application == w.ApplicationReleases[j].Application {
			return w.ApplicationReleases[i].Version < w.ApplicationReleases[j].Version
		}
		return w.ApplicationReleases[i].Application < w.ApplicationReleases[j].Application
	})
	sort.Slice(w.SolutionReleases, func(i, j int) bool {
		if w.SolutionReleases[i].ID == w.SolutionReleases[j].ID {
			return w.SolutionReleases[i].Version < w.SolutionReleases[j].Version
		}
		return w.SolutionReleases[i].ID < w.SolutionReleases[j].ID
	})
	sort.Slice(w.DatabaseReleases, func(i, j int) bool {
		if w.DatabaseReleases[i].Database == w.DatabaseReleases[j].Database {
			return w.DatabaseReleases[i].Version < w.DatabaseReleases[j].Version
		}
		return w.DatabaseReleases[i].Database < w.DatabaseReleases[j].Database
	})
	sort.Slice(w.Migrations, func(i, j int) bool {
		if w.Migrations[i].Database == w.Migrations[j].Database {
			return w.Migrations[i].ID < w.Migrations[j].ID
		}
		return w.Migrations[i].Database < w.Migrations[j].Database
	})
	for i := range w.Databases {
		sort.Slice(w.Databases[i].Contracts, func(a, b int) bool { return w.Databases[i].Contracts[a].ID < w.Databases[i].Contracts[b].ID })
	}
	sort.Slice(w.Environments, func(i, j int) bool { return w.Environments[i].ID < w.Environments[j].ID })
	sort.Slice(w.Queries, func(i, j int) bool { return w.Queries[i].ID < w.Queries[j].ID })
	sort.Slice(w.Bindings, func(i, j int) bool { return w.Bindings[i].Capability < w.Bindings[j].Capability })
	sort.Slice(w.HeaderDetailContractQueryBindings, func(i, j int) bool {
		return w.HeaderDetailContractQueryBindings[i].Identity() < w.HeaderDetailContractQueryBindings[j].Identity()
	})
	sort.Slice(w.ContractQueryExecutorBindings, func(i, j int) bool {
		return w.ContractQueryExecutorBindings[i].Identity() < w.ContractQueryExecutorBindings[j].Identity()
	})
	sort.Slice(w.ContractQuerySQLRealizations, func(i, j int) bool {
		return w.ContractQuerySQLRealizations[i].Identity() < w.ContractQuerySQLRealizations[j].Identity()
	})
	for i := range w.Applications {
		sort.Slice(w.Applications[i].Databases, func(a, b int) bool {
			return w.Applications[i].Databases[a].Alias < w.Applications[i].Databases[b].Alias
		})
		sort.Slice(w.Applications[i].Patterns, func(a, b int) bool {
			return w.Applications[i].Patterns[a].ID < w.Applications[i].Patterns[b].ID
		})
	}
	for i := range w.Environments {
		sort.Slice(w.Environments[i].Plugins, func(a, b int) bool { return w.Environments[i].Plugins[a].ID < w.Environments[i].Plugins[b].ID })
		sort.Slice(w.Environments[i].Bundles, func(a, b int) bool { return w.Environments[i].Bundles[a].ID < w.Environments[i].Bundles[b].ID })
		sort.Slice(w.Environments[i].Features, func(a, b int) bool { return w.Environments[i].Features[a] < w.Environments[i].Features[b] })
		sort.Slice(w.Environments[i].Instances, func(a, b int) bool { return w.Environments[i].Instances[a].ID < w.Environments[i].Instances[b].ID })
		sort.Slice(w.Environments[i].Bindings, func(a, b int) bool {
			x, y := w.Environments[i].Bindings[a], w.Environments[i].Bindings[b]
			if x.Application == y.Application {
				return x.Alias < y.Alias
			}
			return x.Application < y.Application
		})
	}
}

// ParseRef is kept here for later loader-level qualified references.
func ParseRef(kind core.Kind, id string) core.Ref {
	return core.Ref{Kind: kind, ID: core.MustCanonicalID(strings.TrimSpace(id))}
}
