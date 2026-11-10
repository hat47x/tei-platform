package workspace

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
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
	// SemanticDigest optionally pins this particular normalized namespace; it
	// is not a plugin artifact digest or a recursive dependency lock.
	SemanticDigest string `json:"semanticDigest,omitempty"`
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
	// Name is the `workspace:` value the author wrote. It is the identity a
	// caller may cite when several Workspaces use the same Canonical IDs. It
	// is not a path, a version, or an acceptance record.
	Name          string
	FormatVersion int
	Sources       []string
	Imports       []WorkspaceImport
}

// parseWorkspaceManifestYAML accepts the existing small workspace YAML subset
// plus explicit single-namespace imports. It deliberately does not pretend to
// be a general YAML parser. All unsupported entries fail with a source line.
//
// One leading UTF-8 BOM is ignored, matching the syntax lexer: Windows editors
// and PowerShell redirection add it, and refusing that file while accepting a
// BOM in a .tei source was an unexplained asymmetry (#3556). The bytes of the
// manifest are still hashed as read, so a BOM changes manifestIdentity, not the
// meaning of the file.
//
// A `#` starts a comment only at the start of a line or after whitespace,
// which is YAML's rule for a plain scalar. `scn#1` is a name containing `#`,
// not a truncated name (#3557).
func parseWorkspaceManifestYAML(src string) (workspaceManifest, error) {
	src = strings.TrimPrefix(src, "\ufeff")
	if strings.Contains(src, "\ufeff") {
		return workspaceManifest{}, fmt.Errorf("a UTF-8 BOM is only accepted once, at the start of the file")
	}
	var out workspaceManifest
	section := ""
	current := -1
	for n, raw := range strings.Split(src, "\n") {
		line := manifestLine(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "workspace:") {
			// The name is informational identity, not a source root or import.
			// A manifest without it keeps loading (historical fixtures), and the
			// CLI reports an empty name rather than inventing one.
			if out.Name != "" {
				return workspaceManifest{}, fmt.Errorf("line %d: duplicate workspace name", n+1)
			}
			name, err := manifestScalar(strings.TrimPrefix(line, "workspace:"))
			if err != nil {
				return workspaceManifest{}, fmt.Errorf("line %d: workspace name: %w", n+1, err)
			}
			out.Name = name
			continue
		}
		if strings.HasPrefix(line, "formatVersion:") {
			if section != "" {
				return workspaceManifest{}, fmt.Errorf("line %d: formatVersion must appear before sources and imports", n+1)
			}
			if out.FormatVersion != 0 {
				return workspaceManifest{}, fmt.Errorf("line %d: duplicate formatVersion", n+1)
			}
			// formatVersion stays unquoted: "1" is refused rather than unquoted
			// to 1, so a quoted version is not silently accepted (#3557).
			value := strings.TrimSpace(strings.TrimPrefix(line, "formatVersion:"))
			version, parseErr := strconv.Atoi(value)
			if parseErr != nil || version < 1 {
				return workspaceManifest{}, fmt.Errorf("line %d: formatVersion must be a positive integer", n+1)
			}
			out.FormatVersion = version
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
			value, err := manifestScalar(strings.TrimPrefix(line, "-"))
			if err != nil {
				return workspaceManifest{}, fmt.Errorf("line %d: source root: %w", n+1, err)
			}
			if value == "" {
				return workspaceManifest{}, fmt.Errorf("line %d: empty source root", n+1)
			}
			out.Sources = append(out.Sources, value)
		case "imports":
			if strings.HasPrefix(line, "- alias:") {
				alias, err := manifestScalar(strings.TrimPrefix(line, "- alias:"))
				if err != nil {
					return workspaceManifest{}, fmt.Errorf("line %d: import alias: %w", n+1, err)
				}
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
				value, err := manifestScalar(strings.TrimPrefix(line, "path:"))
				if err != nil {
					return workspaceManifest{}, fmt.Errorf("line %d: import path: %w", n+1, err)
				}
				out.Imports[current].Path = value
			case strings.HasPrefix(line, "namespace:"):
				if out.Imports[current].Namespace != "" {
					return workspaceManifest{}, fmt.Errorf("line %d: duplicate import namespace", n+1)
				}
				value, err := manifestScalar(strings.TrimPrefix(line, "namespace:"))
				if err != nil {
					return workspaceManifest{}, fmt.Errorf("line %d: import namespace: %w", n+1, err)
				}
				out.Imports[current].Namespace = value
			case strings.HasPrefix(line, "semanticDigest:"):
				if out.Imports[current].SemanticDigest != "" {
					return workspaceManifest{}, fmt.Errorf("line %d: duplicate import semanticDigest", n+1)
				}
				value, err := manifestScalar(strings.TrimPrefix(line, "semanticDigest:"))
				if err != nil {
					return workspaceManifest{}, fmt.Errorf("line %d: import semanticDigest: %w", n+1, err)
				}
				out.Imports[current].SemanticDigest = value
				if !validImportSemanticDigest(out.Imports[current].SemanticDigest) {
					return workspaceManifest{}, fmt.Errorf("line %d: invalid import semanticDigest: expected sha256 followed by 64 lowercase hex digits", n+1)
				}
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
		if imp.SemanticDigest != "" && !validImportSemanticDigest(imp.SemanticDigest) {
			return workspaceManifest{}, fmt.Errorf("import %q: invalid semanticDigest", imp.Alias)
		}
		if imp.Path == "" || imp.Path == "." || path.IsAbs(imp.Path) ||
			strings.ContainsAny(imp.Path, "\\:") || path.Clean(imp.Path) != imp.Path {
			return workspaceManifest{}, fmt.Errorf("import %q: path %q must be a clean relative Workspace directory", imp.Alias, imp.Path)
		}
	}
	return out, nil
}

// manifestLine strips a YAML comment and trims the result. A `#` begins a
// comment only at the start of the line or after whitespace and outside quotes,
// so `scn#1` keeps its `#` (#3557).
func manifestLine(raw string) string {
	inSingle, inDouble := false, false
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if inSingle || inDouble {
				continue
			}
			if i != 0 && raw[i-1] != ' ' && raw[i-1] != '\t' {
				continue
			}
			return strings.TrimSpace(raw[:i])
		}
	}
	return strings.TrimSpace(raw)
}

// manifestScalar reads one value of the manifest subset. A value wrapped in
// matching quotes is returned without them; otherwise the text is returned as
// written (already trimmed). A missing closing quote, an unescaped quote inside
// the value, or trailing text after the closing quote is refused with the
// reason, so a malformed line is named rather than reinterpreted (#3557).
// Escape sequences are not interpreted.
func manifestScalar(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if trimmed[0] != '"' && trimmed[0] != '\'' {
		return trimmed, nil
	}
	quote := trimmed[:1]
	closing := strings.Index(trimmed[1:], quote)
	if closing < 0 {
		return "", fmt.Errorf("quoted value is missing its closing %s", quote)
	}
	inner := trimmed[1 : 1+closing]
	if strings.Contains(inner, quote) {
		return "", fmt.Errorf("quoted value contains an unescaped %s", quote)
	}
	if rest := strings.TrimSpace(trimmed[1+closing+1:]); rest != "" {
		return "", fmt.Errorf("quoted value is followed by %q", rest)
	}
	return inner, nil
}

// validImportSemanticDigest checks the existing semantic fingerprint representation.
// The pin is an explicit, fail-closed assertion rather than a new lockfile.
func validImportSemanticDigest(value string) bool {
	const prefix = "sha256:"
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, prefix))
	return err == nil
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
		provider, err := loadWithStackHooks(providerPath, visiting, loadHooks{collectSyntax: result.syntaxErrors != nil})
		if err != nil {
			var syntaxErrors *SourceErrors
			if result.syntaxErrors != nil && errors.As(err, &syntaxErrors) {
				result.syntaxErrors.Errors = append(result.syntaxErrors.Errors, syntaxErrors.Errors...)
				continue
			}
			return fmt.Errorf("import %q from %q: %w", imp.Alias, imp.Path, err)
		}
		// Do not resolve ownership or digest against an incomplete World.
		if result.syntaxErrors != nil && len(result.syntaxErrors.Errors) > 0 {
			continue
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
		actualDigest := sourceindex.Fingerprint(encoded)
		if imp.SemanticDigest != "" && imp.SemanticDigest != actualDigest {
			return fmt.Errorf("import %q: semantic digest mismatch for namespace %q (expected %s, got %s); review the provider changes and their impact before updating the pin", imp.Alias, imp.Namespace, imp.SemanticDigest, actualDigest)
		}
		// Record the selected Namespace's actual Parser-consumed bytes and
		// Provider Manifest from the same nested Load. This is deliberately
		// private, selected-scope and does not qualify full input closure.
		capture, err := captureImportedParsedLoad(imp, actualDigest, provider)
		if err != nil {
			return fmt.Errorf("import %q: %w", imp.Alias, err)
		}
		if result.importedParsedLoads == nil {
			result.importedParsedLoads = make(map[string]*importedParsedLoad)
		}
		result.importedParsedLoads[imp.Alias] = &capture
		owned[imp.Namespace] = true
		result.World.Semantics = append(result.World.Semantics, *selected)
		result.Imports = append(result.Imports, ResolvedImport{
			Alias:          imp.Alias,
			Path:           imp.Path,
			Namespace:      imp.Namespace,
			SemanticDigest: actualDigest,
		})
	}
	return nil
}
