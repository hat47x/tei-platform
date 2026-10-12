package workspace

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

// WorkspaceImport describes an explicit dependency on one semantic namespace
// owned by another TEI Workspace. It is not a managed source root.
type WorkspaceImport struct {
	Alias     string `json:"alias"`
	Path      string `json:"path"`
	Namespace string `json:"namespace"`
}

// ResolvedImport records a read-only dependency. SemanticDigest fingerprints
// the normalized semantic definition, not the original source bytes or an
// executable artifact. Source provenance remains owned by the provider.
type ResolvedImport struct {
	Alias          string `json:"alias"`
	Path           string `json:"path"`
	Namespace      string `json:"namespace"`
	SemanticDigest string `json:"semanticDigest"`
}

type workspaceManifest struct {
	Sources []string
	Imports []WorkspaceImport
}

// parseWorkspaceManifestYAML accepts the existing small workspace YAML subset
// plus explicit single-namespace imports. It deliberately does not pretend to
// be a general YAML parser. All unsupported entries fail with a source line.
func parseWorkspaceManifestYAML(src string) (workspaceManifest, error) {
	var out workspaceManifest
	section := ""
	current := -1
	for n, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(strings.SplitN(raw, "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "workspace:") {
			continue
		}
		switch line {
		case "sources:":
			section = "sources"
			continue
		case "imports:":
			section = "imports"
			continue
		}
		switch section {
		case "sources":
			if !strings.HasPrefix(line, "-") {
				return workspaceManifest{}, fmt.Errorf("line %d: unsupported workspace manifest entry %q", n+1, line)
			}
			value := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			if value == "" {
				return workspaceManifest{}, fmt.Errorf("line %d: empty source root", n+1)
			}
			out.Sources = append(out.Sources, value)
		case "imports":
			if strings.HasPrefix(line, "- alias:") {
				alias := strings.TrimSpace(strings.TrimPrefix(line, "- alias:"))
				out.Imports = append(out.Imports, WorkspaceImport{Alias: alias})
				current = len(out.Imports) - 1
				continue
			}
			if current < 0 {
				return workspaceManifest{}, fmt.Errorf("line %d: import must begin with '- alias:'", n+1)
			}
			switch {
			case strings.HasPrefix(line, "path:"):
				if out.Imports[current].Path != "" {
					return workspaceManifest{}, fmt.Errorf("line %d: duplicate import path", n+1)
				}
				out.Imports[current].Path = strings.TrimSpace(strings.TrimPrefix(line, "path:"))
			case strings.HasPrefix(line, "namespace:"):
				if out.Imports[current].Namespace != "" {
					return workspaceManifest{}, fmt.Errorf("line %d: duplicate import namespace", n+1)
				}
				out.Imports[current].Namespace = strings.TrimSpace(strings.TrimPrefix(line, "namespace:"))
			default:
				return workspaceManifest{}, fmt.Errorf("line %d: unsupported import entry %q", n+1, line)
			}
		default:
			return workspaceManifest{}, fmt.Errorf("line %d: unsupported workspace manifest entry %q", n+1, line)
		}
	}
	if len(out.Sources) == 0 {
		return workspaceManifest{}, fmt.Errorf("sources must contain at least one source root")
	}
	aliases := make(map[string]bool, len(out.Imports))
	namespaces := make(map[string]bool, len(out.Imports))
	for _, imp := range out.Imports {
		if imp.Alias == "" || strings.ContainsAny(imp.Alias, " /\\:\t\r\n") {
			return workspaceManifest{}, fmt.Errorf("invalid import alias %q", imp.Alias)
		}
		if aliases[imp.Alias] {
			return workspaceManifest{}, fmt.Errorf("duplicate import alias %q", imp.Alias)
		}
		aliases[imp.Alias] = true
		if imp.Namespace == "" || strings.TrimSpace(imp.Namespace) != imp.Namespace ||
			strings.ContainsAny(imp.Namespace, " /\\:\t\r\n") {
			return workspaceManifest{}, fmt.Errorf("invalid import namespace %q", imp.Namespace)
		}
		if namespaces[imp.Namespace] {
			return workspaceManifest{}, fmt.Errorf("duplicate imported namespace %q", imp.Namespace)
		}
		namespaces[imp.Namespace] = true
		if imp.Path == "" || imp.Path == "." || path.IsAbs(imp.Path) ||
			strings.ContainsAny(imp.Path, "\\:") || path.Clean(imp.Path) != imp.Path {
			return workspaceManifest{}, fmt.Errorf("import %q: path %q must be a clean relative Workspace directory", imp.Alias, imp.Path)
		}
	}
	return out, nil
}

// loadWorkspaceImports uses the provider's own Workspace loader, preserving its
// managed-file boundary. It imports only explicitly named, locally declared
// namespaces; transitive dependencies are not silently re-exported.
func loadWorkspaceImports(root string, imports []WorkspaceImport, result *Result, visiting map[string]bool) error {
	owned := make(map[string]bool, len(result.World.Semantics))
	for _, ns := range result.World.Semantics {
		owned[string(ns.ID)] = true
	}
	for _, imp := range imports {
		providerPath := filepath.Join(root, filepath.FromSlash(imp.Path))
		provider, err := loadWithStack(providerPath, visiting)
		if err != nil {
			return fmt.Errorf("import %q from %q: %w", imp.Alias, imp.Path, err)
		}
		// The provider's source map holds only its own declarations. Namespaces
		// it imported from other providers cannot be re-exported implicitly.
		sourceKey := "semantic:" + imp.Namespace
		if _, ok := provider.Sources[sourceKey]; !ok {
			return fmt.Errorf("import %q: namespace %q is absent or not owned by provider %q", imp.Alias, imp.Namespace, imp.Path)
		}
		var selected *semantic.Namespace
		for i := range provider.World.Semantics {
			if string(provider.World.Semantics[i].ID) == imp.Namespace {
				selected = &provider.World.Semantics[i]
				break
			}
		}
		if selected == nil {
			return fmt.Errorf("import %q: namespace %q not found in provider model", imp.Alias, imp.Namespace)
		}
		if owned[imp.Namespace] {
			return fmt.Errorf("import %q: namespace %q conflicts with an existing namespace", imp.Alias, imp.Namespace)
		}
		encoded, err := json.Marshal(selected)
		if err != nil {
			return fmt.Errorf("import %q: fingerprint namespace: %w", imp.Alias, err)
		}
		owned[imp.Namespace] = true
		result.World.Semantics = append(result.World.Semantics, *selected)
		result.Imports = append(result.Imports, ResolvedImport{
			Alias:          imp.Alias,
			Path:           imp.Path,
			Namespace:      imp.Namespace,
			SemanticDigest: sourceindex.Fingerprint(encoded),
		})
	}
	return nil
}
