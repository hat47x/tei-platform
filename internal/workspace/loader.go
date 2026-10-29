package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
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
	Imports                     []ResolvedImport `json:"imports,omitempty"`
	World                       ir.World
	PluginImplementations       []implementation.Source
	PluginImplementationSources map[string]SourceLocation
	Sources                     map[string]SourceLocation
	ManagedFiles                []ManagedFile
	SourceContributions         []SourceContribution

	// Private, transient read provenance. Not part of Canonical World or a
	// serialized Source Index; only Load may populate this snapshot.
	sourceSnapshotRoot string
	sourceSnapshot     map[string][]byte
	// Load-time bytes of managed files that no domain loader parsed (for
	// example README.md or unknown-root assets). Kept apart from
	// sourceSnapshot so the parser-bound meaning of that map, and of
	// IndexedLoadedSnapshot, is unchanged. Used only by BuildSourceIndexFromLoad.
	sourceManagedExtra map[string][]byte

	// Only populated during Load. One pinned directory handle protects all
	// domain Source reads from a Workspace root pathname being replaced.
	// Never retain an open OS handle in the returned Result.
	sourceReadRoot *os.Root
}

func Load(root string) (Result, error) {
	return loadWithStack(root, map[string]bool{})
}

func loadWithStack(root string, visiting map[string]bool) (Result, error) {
	// Canonicalize once, then pin the directory before reading the manifest.
	// Otherwise a rename between manifest and domain-source reads could
	// silently redirect the second read to an unrelated Workspace.
	rootReal, err := canonicalWorkspaceRoot(root)
	if err != nil {
		return Result{}, err
	}
	if visiting[rootReal] {
		return Result{}, fmt.Errorf("cyclic Workspace import at %s", root)
	}
	sourceRoot, err := openWorkspaceRoot(rootReal)
	if err != nil {
		return Result{}, err
	}
	defer sourceRoot.Close()
	visiting[rootReal] = true
	defer delete(visiting, rootReal)

	manifestYAML := filepath.Join(root, "tei.workspace.yaml")
	manifestTEI := filepath.Join(root, "tei.workspace.tei")
	var roots []string
	var imports []WorkspaceImport
	if src, err := readPinnedWorkspaceFile(sourceRoot, "tei.workspace.yaml"); err == nil {
		manifest, parseErr := parseWorkspaceManifestYAML(string(src))
		if parseErr != nil {
			return Result{}, fmt.Errorf("%s: %w", manifestYAML, parseErr)
		}
		roots, imports = manifest.Sources, manifest.Imports
	} else if src, legacyErr := readPinnedWorkspaceFile(sourceRoot, "tei.workspace.tei"); legacyErr == nil {
		var parseErr error
		roots, parseErr = syntax.NewParser(string(src)).ParseWorkspace()
		if parseErr != nil {
			return Result{}, fmt.Errorf("%s: %w", manifestTEI, parseErr)
		}
	} else {
		return Result{}, fmt.Errorf("TEI workspace manifest not found: expected %s", manifestYAML)
	}
	if err := validateWorkspaceSourceRoots(roots); err != nil {
		return Result{}, err
	}
	// Inventory and verify the physical boundary before any domain loader can
	// read a Source file or parse a symlink to an outside target.
	managedFiles, err := scanManagedFiles(sourceRoot, roots)
	if err != nil {
		return Result{}, err
	}
	r := Result{
		Sources:                     map[string]SourceLocation{},
		PluginImplementationSources: map[string]SourceLocation{},
		ManagedFiles:                managedFiles,
		sourceSnapshotRoot:          rootReal,
		sourceSnapshot:              make(map[string][]byte),
		sourceReadRoot:              sourceRoot,
	}
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
	if err := loadWorkspaceImports(root, imports, &r, visiting); err != nil {
		return Result{}, err
	}
	Normalize(&r.World)
	// Capture the bytes of managed files that no loader parsed, through the
	// same pinned root, so the Source Index can be built without a later
	// filesystem read. Imported Workspaces are loaded by their own call and
	// are not covered. Nothing here is parsed.
	if err := captureManagedExtra(sourceRoot, &r); err != nil {
		return Result{}, err
	}
	// The loader snapshot stays available, but an OS handle must not escape
	// the Load lifetime. Imports close their own pinned roots independently.
	r.sourceReadRoot = nil
	return r, nil
}

func captureManagedExtra(root *os.Root, r *Result) error {
	r.sourceManagedExtra = make(map[string][]byte)
	for _, managed := range r.ManagedFiles {
		content, err := readPinnedWorkspaceFile(root, managed.Path)
		if err != nil {
			return fmt.Errorf("capture managed source %q at Load: %w", managed.Path, err)
		}
		if previous, ok := r.sourceSnapshot[managed.Path]; ok {
			if !bytes.Equal(previous, content) {
				return fmt.Errorf("source %q changed while loading Workspace", managed.Path)
			}
			continue
		}
		r.sourceManagedExtra[managed.Path] = content
	}
	return nil
}

// parseWorkspaceYAML is retained for existing callers of the source-root-only API.
func parseWorkspaceYAML(src string) ([]string, error) {
	manifest, err := parseWorkspaceManifestYAML(src)
	return manifest.Sources, err
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

// A declared source must be owned by exactly one source root. Duplicate and
// ancestor/descendant roots could otherwise parse one canonical declaration
// more than once or create duplicate Source Index artifacts.
func validateWorkspaceSourceRoots(roots []string) error {
	for i, sourceRoot := range roots {
		if err := validateWorkspaceSourceRoot(sourceRoot); err != nil {
			return fmt.Errorf("workspace source root %q: %w", sourceRoot, err)
		}
		for _, prior := range roots[:i] {
			if sourceRoot == prior || strings.HasPrefix(sourceRoot, prior+"/") ||
				strings.HasPrefix(prior, sourceRoot+"/") {
				return fmt.Errorf("workspace source roots %q and %q overlap", prior, sourceRoot)
			}
		}
	}
	return nil
}

// scanManagedFiles inventories native files without granting them semantic
// authority. Missing optional roots remain forward-compatible; unreadable,
// escaped, broken-linked or non-regular source entries fail instead of being
// silently omitted or introduced as Workspace-managed artifacts.
// scanManagedFiles inventories managed files through the same pinned root
// used by Load's manifest and source reads. Never use filepath.WalkDir or
// EvalSymlinks here: their independent pathname lookups may be redirected
// when the Workspace directory or an ancestor is renamed.
//
// This is not an atomic multi-file snapshot. Entries and contents may change
// in place during Load; the individual rooted source read remains authoritative
// for the actual bytes parsed and captured by the Source Index.
func scanManagedFiles(rootHandle *os.Root, roots []string) ([]ManagedFile, error) {
	var out []ManagedFile
	type seenSource struct {
		path string
		info os.FileInfo
	}
	var seen []seenSource
	for _, sourceRoot := range roots {
		if err := validateWorkspaceSourceRoot(sourceRoot); err != nil {
			return nil, fmt.Errorf("workspace source root %q: %w", sourceRoot, err)
		}
		prefix := ""
		missingOptionalRoot := false
		// Distinguish a genuinely missing optional root from an existing
		// broken or escaping link. Keep source roots as directories; file
		// symlinks within those directories remain valid managed assets.
		for _, component := range strings.Split(sourceRoot, "/") {
			if prefix == "" {
				prefix = component
			} else {
				prefix += "/" + component
			}
			info, err := rootHandle.Lstat(prefix)
			if errors.Is(err, os.ErrNotExist) {
				missingOptionalRoot = true
				break
			}
			if err != nil {
				return nil, fmt.Errorf("stat workspace source root %q at %s: %w", sourceRoot, prefix, err)
			}
			if _, err := rootHandle.Stat(prefix); err != nil {
				return nil, fmt.Errorf("resolve workspace source root %q at %s outside the Workspace or inaccessible: %w", sourceRoot, prefix, err)
			}
			if !info.IsDir() {
				return nil, fmt.Errorf("workspace source root %q must be a directory", sourceRoot)
			}
		}
		if missingOptionalRoot {
			continue
		}
		// fs.WalkDir uses Root.FS to resolve every traversal within the
		// pinned directory handle. The callback re-stats each leaf through
		// that handle; even a swapped symlink cannot redirect it outside.
		err := fs.WalkDir(rootHandle.FS(), sourceRoot, func(logicalPath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return fmt.Errorf("scan managed source %s outside the Workspace or inaccessible: %w", logicalPath, walkErr)
			}
			if d.IsDir() {
				return nil
			}
			info, err := rootHandle.Stat(logicalPath)
			if err != nil {
				return fmt.Errorf("managed source %s resolves outside the Workspace or is inaccessible: %w", logicalPath, err)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("managed source %s is not a regular file", logicalPath)
			}
			// Compare actual file identity, not just the textual path:
			// hard links and in-Workspace symlink aliases must not produce
			// multiple canonical Source Index artifacts for one file.
			for _, previous := range seen {
				if os.SameFile(previous.info, info) {
					return fmt.Errorf("managed sources %q and %q resolve to the same physical file", previous.path, logicalPath)
				}
			}
			seen = append(seen, seenSource{path: logicalPath, info: info})
			out = append(out, ManagedFile{
				Path: logicalPath, SourceRoot: sourceRoot,
				Extension: strings.ToLower(filepath.Ext(d.Name())),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// listPinnedDomainSources enumerates files through the same Root used for
// Source reads, rather than through independently resolved Workspace paths.
// Direct per-domain loader tests may not have a Load root; those calls open a
// temporary rooted handle instead. This inventories names, not an atomic
// snapshot of file contents: the later pinned read remains authoritative.
func listPinnedDomainSources(root string, r *Result, directory string, keep func(string) bool) ([]string, error) {
	rootHandle := r.sourceReadRoot
	if rootHandle == nil {
		var err error
		rootHandle, err = os.OpenRoot(root)
		if err != nil {
			return nil, fmt.Errorf("open Workspace root for %s listing: %w", directory, err)
		}
		defer rootHandle.Close()
	}
	var files []string
	err := fs.WalkDir(rootHandle.FS(), directory, func(logical string, d fs.DirEntry, walkErr error) error {
		if errors.Is(walkErr, os.ErrNotExist) && logical == directory {
			return nil // Optional absent domain root.
		}
		if walkErr != nil {
			return fmt.Errorf("list Workspace source %s: %w", logical, walkErr)
		}
		if d.IsDir() || !keep(d.Name()) {
			return nil
		}
		info, err := rootHandle.Stat(logical)
		if err != nil {
			return fmt.Errorf("resolve Workspace source %s: %w", logical, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Workspace source %s is not a regular file", logical)
		}
		files = append(files, filepath.Join(root, filepath.FromSlash(logical)))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func loadPlugins(root string, r *Result) error {
	files, err := listPinnedDomainSources(root, r, "plugins", func(name string) bool {
		return name == "plugin.tei"
	})
	if err != nil {
		return err
	}

	for _, path := range files {
		src, err := readSourceForLoad(root, r, path)
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
	files, err := listPinnedDomainSources(root, r, "bundles", func(name string) bool {
		return name == "bundle.tei"
	})
	if err != nil {
		return err
	}

	for _, path := range files {
		src, err := readSourceForLoad(root, r, path)
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
	files, err := listPinnedDomainSources(root, r, "semantic", func(name string) bool {
		return strings.HasSuffix(name, ".tei")
	})
	if err != nil {
		return err
	}

	byNamespace := map[core.CanonicalID]int{}
	for _, path := range files {
		src, err := readSourceForLoad(root, r, path)
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
			for _, key := range d.CandidateKeys {
				r.Sources["semantic-candidate-key:"+string(key.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
			}
			for _, fd := range d.FunctionalDependencies {
				r.Sources["semantic-functional-dependency:"+string(fd.ID)] = SourceLocation{File: relPath(root, path), Line: 1, Column: 1}
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
		src, err := readSourceForLoad(root, r, path)
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
			patternSource, err := readSourceForLoad(root, r, patternPath)
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
			qsrc, err := readSourceForLoad(root, r, qpath)
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
				sqlSource, err := readSourceForLoad(root, r, nativePath)
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
			bsrc, err := readSourceForLoad(root, r, bpath)
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
		src, err := readSourceForLoad(root, r, path)
		if err != nil {
			return err
		}
		d, err := syntax.NewParser(string(src)).ParseDatabase()
		if err != nil {
			return sourceErr(path, err)
		}
		contractFiles, _ := globSorted(filepath.Join(filepath.Dir(path), "contracts", "*.tei"))
		for _, cpath := range contractFiles {
			csrc, err := readSourceForLoad(root, r, cpath)
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
			psrc, err := readSourceForLoad(root, r, ppath)
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
		if psrc, err := readSourceForLoad(root, r, providerPath); err == nil {
			cfg, err := syntax.NewParser(string(psrc)).ParseMigrationProvider()
			if err != nil {
				return sourceErr(providerPath, err)
			}
			r.World.MigrationProviders = append(r.World.MigrationProviders, cfg)
			r.Sources["migration-provider:"+string(cfg.Database)] = SourceLocation{File: relPath(root, providerPath), Line: 1, Column: 1}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		migrationFiles, _ := globSorted(filepath.Join(filepath.Dir(path), "migrations", "*", "migration.tei"))
		for _, mpath := range migrationFiles {
			msrc, err := readSourceForLoad(root, r, mpath)
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
		src, err := readSourceForLoad(root, r, path)
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
		src, err := readSourceForLoad(root, r, path)
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
		src, err := readSourceForLoad(root, r, path)
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
		src, err := readSourceForLoad(root, r, path)
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
