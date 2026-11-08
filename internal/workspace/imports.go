package workspace

import (
	"encoding/hex"
	"encoding/json"
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

// manifestLine returns the content of one manifest line with the comment and
// the surrounding space removed. A `#` starts a comment only outside a quoted
// value and only at the start of the line or after whitespace, so `scn#1` keeps
// its hash instead of being silently read as `scn` (#3557).
func manifestLine(raw string, number int) (string, error) {
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
			if i == 0 || raw[i-1] == ' ' || raw[i-1] == '\t' {
				raw = raw[:i]
				i = len(raw)
			}
		}
	}
	if inSingle || inDouble {
		return "", fmt.Errorf("line %d: unterminated quoted value", number)
	}
	return strings.TrimSpace(raw), nil
}

// manifestScalar unwraps one manifest value. A value may be quoted with ' or "
// when it contains a `#`, a space at either end, or a character the subset
// otherwise reads structurally; the quotes are not part of the value. Text
// after the closing quote is rejected instead of silently dropped (#3557).
func manifestScalar(value string, number int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	quote := value[0]
	if quote != '\'' && quote != '"' {
		return value, nil
	}
	end := strings.IndexByte(value[1:], quote)
	if end < 0 {
		return "", fmt.Errorf("line %d: unterminated quoted value", number)
	}
	rest := strings.TrimSpace(value[end+2:])
	if rest != "" {
		return "", fmt.Errorf("line %d: unexpected text after quoted value: %q", number, rest)
	}
	return value[1 : end+1], nil
}

// parseWorkspaceManifestYAML accepts the existing small workspace YAML subset
// plus explicit single-namespace imports. It deliberately does not pretend to
// be a general YAML parser. All unsupported entries fail with a source line.
//
// One leading UTF-8 BOM is ignored, matching the .tei lexer: an editor or a
// Windows PowerShell redirect that saves "UTF-8" can prefix the BOM, and the
// manifest is a hand-edited file. The manifest digest stays computed over the
// raw bytes the caller read, so the BOM is still part of the identity that
// `manifest.sha256` reports (issue #3556).
func parseWorkspaceManifestYAML(src string) (workspaceManifest, error) {
	var out workspaceManifest
	section := ""
	current := -1
	// A BOM anywhere but the first byte stays an unsupported entry: it is not
	// a prefix, and treating it as whitespace would hide a corrupted file.
	src = strings.TrimPrefix(src, "\ufeff")
	for n, raw := range strings.Split(src, "\n") {
		line, err := manifestLine(raw, n+1)
		if err != nil {
			return workspaceManifest{}, err
		}
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
			name, err := manifestScalar(strings.TrimPrefix(line, "workspace:"), n+1)
			if err != nil {
				return workspaceManifest{}, err
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
			value, err := manifestScalar(strings.TrimPrefix(line, "formatVersion:"), n+1)
			if err != nil {
				return workspaceManifest{}, err
			}
			version, err := strconv.Atoi(value)
			if err != nil || version < 1 {
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
			value, err := manifestScalar(strings.TrimPrefix(line, "-"), n+1)
			if err != nil {
				return workspaceManifest{}, err
			}
			if value == "" {
				return workspaceManifest{}, fmt.Errorf("line %d: empty source root", n+1)
			}
			out.Sources = append(out.Sources, value)
		case "imports":
			if strings.HasPrefix(line, "- alias:") {
				alias, err := manifestScalar(strings.TrimPrefix(line, "- alias:"), n+1)
				if err != nil {
					return workspaceManifest{}, err
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
				value, err := manifestScalar(strings.TrimPrefix(line, "path:"), n+1)
				if err != nil {
					return workspaceManifest{}, err
				}
				out.Imports[current].Path = value
			case strings.HasPrefix(line, "namespace:"):
				if out.Imports[current].Namespace != "" {
					return workspaceManifest{}, fmt.Errorf("line %d: duplicate import namespace", n+1)
				}
				value, err := manifestScalar(strings.TrimPrefix(line, "namespace:"), n+1)
				if err != nil {
					return workspaceManifest{}, err
				}
				out.Imports[current].Namespace = value
			case strings.HasPrefix(line, "semanticDigest:"):
				if out.Imports[current].SemanticDigest != "" {
					return workspaceManifest{}, fmt.Errorf("line %d: duplicate import semanticDigest", n+1)
				}
				value, err := manifestScalar(strings.TrimPrefix(line, "semanticDigest:"), n+1)
				if err != nil {
					return workspaceManifest{}, err
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
		actualDigest := sourceindex.Fingerprint(encoded)
		if imp.SemanticDigest != "" && imp.SemanticDigest != actualDigest {
			return fmt.Errorf("import %q: semantic digest mismatch for namespace %q (expected %s, got %s); review the provider changes and their impact before updating the pin", imp.Alias, imp.Namespace, imp.SemanticDigest, actualDigest)
		}
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
