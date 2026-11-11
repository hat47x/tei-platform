package workspace

import (
	"fmt"
	"os"
	"path"
	"strings"
)

// KnownWorkspaceSourceRoots names the source roots the Loader interprets.
// A forward-compatible root may be declared for future layouts, but it must be
// named here so a misspelling is not silently accepted forever. Adding a root
// also requires a Loader branch; this list is not a permission to execute.
var KnownWorkspaceSourceRoots = []string{
	"plugins",
	"plugin-implementations",
	"bundles",
	"semantic",
	"applications",
	"databases",
	"environments",
	"releases",
}

func knownWorkspaceSourceRoot(name string) bool {
	for _, known := range KnownWorkspaceSourceRoots {
		if name == known {
			return true
		}
	}
	return false
}

// WorkspaceSourceRootFinding is a manifest source-root problem. It is a
// source-direct finding: no World was built for a misspelled root's files, so
// it cannot carry a Canonical Subject.
type WorkspaceSourceRootFinding struct {
	Root     string `json:"root"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "warning" for a forward-compatible unknown root, "error" for a missing one
	// File is set when the finding is about a managed file whose name the
	// Loader does not interpret, rather than the root itself.
	File string `json:"file,omitempty"`
}

// checkWorkspaceSourceRoots classifies manifest source roots through the same
// pinned root handle Load already opened. A missing root is an error because
// the caller asked for it and nothing was read. An unknown root is a warning:
// the layout is accepted for forward compatibility, but the caller is told the
// Loader does not interpret it. `semantics` (a likely misspelling of
// `semantic`) and other near names therefore become visible instead of a
// downstream cascade.
func checkWorkspaceSourceRoots(rootHandle *os.Root, roots []string) ([]WorkspaceSourceRootFinding, error) {
	var findings []WorkspaceSourceRootFinding
	for _, sourceRoot := range roots {
		if err := validateWorkspaceSourceRoot(sourceRoot); err != nil {
			return nil, fmt.Errorf("workspace source root %q: %w", sourceRoot, err)
		}
		info, err := rootHandle.Stat(sourceRoot)
		if err != nil {
			if os.IsNotExist(err) {
				findings = append(findings, WorkspaceSourceRootFinding{
					Root:     sourceRoot,
					Message:  fmt.Sprintf("declared source root %q does not exist; nothing was read for it", sourceRoot),
					Severity: "error",
				})
				continue
			}
			// A symlink escape or other containment failure is not a finding:
			// fail closed exactly as scanManagedFiles does. Reporting it as a
			// warning would accept the escaped root.
			return nil, fmt.Errorf("inspect workspace source root %q: %w", sourceRoot, err)
		}
		if !info.IsDir() {
			findings = append(findings, WorkspaceSourceRootFinding{
				Root:     sourceRoot,
				Message:  fmt.Sprintf("declared source root %q is not a directory", sourceRoot),
				Severity: "error",
			})
			continue
		}
		if !knownWorkspaceSourceRoot(sourceRoot) {
			// Forward-compatible layout: accepted, but not interpreted.
			findings = append(findings, WorkspaceSourceRootFinding{
				Root:     sourceRoot,
				Message:  fmt.Sprintf("source root %q is not interpreted by this Loader; its files are managed but not read", sourceRoot),
				Severity: "warning",
			})
		}
	}
	return findings, nil
}

// fixedUnitFileNames are the exact file names some Loaders read. A case or
// suffix variant is a near miss: it is managed but not read.
var fixedUnitFileNames = []string{
	"plugin.tei", "bundle.tei", "application.tei", "database.tei",
	"environment.tei", "migration.tei",
}

// CheckWorkspaceUnreadSources reports .tei managed files that no domain loader
// read. They are inventoried, so neither the source root check nor the parser
// ever sees them: a file placed one directory too high validates as `ok` while
// changing nothing (#4128 round 71). Reporting it is a warning, not an error,
// because a workspace may legitimately keep a file the current loaders do not
// interpret yet.
func CheckWorkspaceUnreadSources(managed []ManagedFile) []WorkspaceSourceRootFinding {
	var findings []WorkspaceSourceRootFinding
	for _, file := range managed {
		if file.ReadByLoader || file.Extension != ".tei" {
			continue
		}
		if f := CheckWorkspaceManagedFileNames([]ManagedFile{file}); len(f) > 0 {
			// The name already explains why this file is not read; do not
			// report the same file twice.
			continue
		}
		findings = append(findings, WorkspaceSourceRootFinding{
			Root: file.SourceRoot, File: file.Path, Severity: "warning",
			Message: fmt.Sprintf("managed file %q is a TEI source that no loader reads, so its declarations never reach the model; check its location (for example a persistence mapping belongs in databases/<database>/persistence/)", file.Path),
		})
	}
	return findings
}

// CheckWorkspaceManagedFileNames reports managed files that look like a
// Loader input but whose exact name or extension the Loader does not read.
// This is a warning, not an error: the files stay in the managed inventory.
// It does not prove which files a domain loader parsed.
func CheckWorkspaceManagedFileNames(managed []ManagedFile) []WorkspaceSourceRootFinding {
	var findings []WorkspaceSourceRootFinding
	for _, file := range managed {
		base := path.Base(file.Path)
		lower := strings.ToLower(base)
		for _, exact := range fixedUnitFileNames {
			if lower == exact && base != exact {
				findings = append(findings, WorkspaceSourceRootFinding{
					Root: file.SourceRoot, File: file.Path, Severity: "warning",
					Message: fmt.Sprintf("managed file %q is not read; the Loader requires the exact name %q", file.Path, exact),
				})
			}
		}
		// Semantic and nested pattern/query loaders accept lowercase *.tei.
		if strings.Contains(lower, ".tei") && !strings.HasSuffix(base, ".tei") {
			findings = append(findings, WorkspaceSourceRootFinding{
				Root: file.SourceRoot, File: file.Path, Severity: "warning",
				Message: fmt.Sprintf("managed file %q is not read; the Loader accepts lowercase *.tei files in this root", file.Path),
			})
		}
	}
	return findings
}
