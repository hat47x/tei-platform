package workspace

import (
	"bytes"
	"fmt"
	"sort"

	"tei/internal/sourceindex"
)

// ImportedParsedLoadEvidence describes the selected provider Namespace and the
// exact parsed Source bytes captured during the SAME nested provider Load as
// the consumer's Workspace Import. Manifest is a separate input, never a
// consumer-managed Source Index artifact.
//
// FullInputClosure is always false: provider-wide inputs, negative dependencies,
// multi-file atomicity, current Provider authority and downstream consumers
// remain unqualified. This is not an Acceptance or execution grant.
type ImportedParsedLoadEvidence struct {
	Alias            string
	ProviderPath     string
	Namespace        string
	SemanticDigest   string
	Manifest         ManifestIdentity
	ParsedSources    []sourceindex.Artifact
	FullInputClosure bool
}

// importedParsedLoad is private, transient provenance. It is never exported by
// serialization of Result; a caller must explicitly request the read-only
// projection and must separately authorize disclosure of any raw Source.
type importedParsedLoad struct {
	evidence ImportedParsedLoadEvidence
	files    map[string][]byte
}

// captureImportedParsedLoad is called only on the Result returned by the
// provider's nested Load, before those bytes are discarded by Import selection.
// It does not reopen a physical file or extend the source scope to unrelated
// provider Namespaces or transitive imports.
func captureImportedParsedLoad(imp WorkspaceImport, digest string, provider Result) (importedParsedLoad, error) {
	var empty importedParsedLoad
	paths := provider.semanticFilesByNamespace[imp.Namespace]
	if len(paths) == 0 || provider.sourceSnapshot == nil ||
		provider.sourceManifest.Path == "" || len(provider.sourceManifestBytes) == 0 {
		return empty, fmt.Errorf("import %q lacks same-Load parsed Source or manifest evidence", imp.Alias)
	}
	if sourceindex.Fingerprint(provider.sourceManifestBytes) != provider.sourceManifest.SHA256 {
		return empty, fmt.Errorf("import %q has inconsistent Load-time manifest identity", imp.Alias)
	}

	files := make(map[string][]byte, len(paths)+1)
	files[provider.sourceManifest.Path] = bytes.Clone(provider.sourceManifestBytes)
	evidence := ImportedParsedLoadEvidence{
		Alias:          imp.Alias,
		ProviderPath:   imp.Path,
		Namespace:      imp.Namespace,
		SemanticDigest: digest,
		Manifest:       provider.sourceManifest,
	}
	sortedPaths := append([]string(nil), paths...)
	sort.Strings(sortedPaths)
	for _, path := range sortedPaths {
		normalized, err := sourceindex.NormalizeWorkspacePath(path)
		if err != nil || normalized != path || path == evidence.Manifest.Path {
			return empty, fmt.Errorf("import %q has invalid parsed Source path %q", imp.Alias, path)
		}
		raw, ok := provider.sourceSnapshot[path]
		if !ok {
			return empty, fmt.Errorf("import %q parsed Source %q has no Load-time bytes", imp.Alias, path)
		}
		if _, duplicate := files[path]; duplicate {
			return empty, fmt.Errorf("import %q has duplicate parsed Source %q", imp.Alias, path)
		}
		artifact, err := sourceindex.NewArtifact(path, raw)
		if err != nil {
			return empty, fmt.Errorf("import %q parsed Source %q: %w", imp.Alias, path, err)
		}
		files[path] = bytes.Clone(raw)
		evidence.ParsedSources = append(evidence.ParsedSources, artifact)
	}
	return importedParsedLoad{evidence: evidence, files: files}, nil
}

// ImportedParsedLoadSnapshot retrieves a read-only in-memory view of parsed
// Source and manifest bytes captured during the selected external Import's
// initial provider Load. No filesystem read or LocateImportedSource re-Load
// occurs here. The returned reader makes independent copies on every call.
//
// Only the selected Namespace's parsed Source files and the provider Manifest
// are readable; unrelated provider Sources and transitive imported files are
// excluded. Raw bytes require an explicit read and must not be sent to logs,
// AI or other consumers without their own disclosure permission.
//
// This does not satisfy BuildSourceIndexFromLoad for imported Workspaces, and
// does not claim an atomic multi-file snapshot or complete input closure.
func ImportedParsedLoadSnapshot(loaded Result, alias string) (ImportedParsedLoadEvidence, func(string) ([]byte, error), error) {
	var empty ImportedParsedLoadEvidence
	var selected *ResolvedImport
	for i := range loaded.Imports {
		if loaded.Imports[i].Alias != alias {
			continue
		}
		if selected != nil {
			return empty, nil, fmt.Errorf("import alias %q is ambiguous", alias)
		}
		selected = &loaded.Imports[i]
	}
	if selected == nil {
		return empty, nil, fmt.Errorf("import alias %q is not resolved", alias)
	}
	capture, ok := loaded.importedParsedLoads[alias]
	if !ok || capture.evidence.Alias != alias ||
		capture.evidence.ProviderPath != selected.Path ||
		capture.evidence.Namespace != selected.Namespace ||
		capture.evidence.SemanticDigest != selected.SemanticDigest {
		return empty, nil, fmt.Errorf("import %q lacks matching same-Load parsed evidence", alias)
	}
	snapshot := make(map[string][]byte, len(capture.files))
	for path, raw := range capture.files {
		snapshot[path] = bytes.Clone(raw)
	}
	for _, artifact := range capture.evidence.ParsedSources {
		raw, exists := snapshot[artifact.Path]
		if !exists || sourceindex.Fingerprint(raw) != artifact.Fingerprint {
			return empty, nil, fmt.Errorf("import %q has inconsistent parsed Source evidence", alias)
		}
	}
	if raw, exists := snapshot[capture.evidence.Manifest.Path]; !exists ||
		sourceindex.Fingerprint(raw) != capture.evidence.Manifest.SHA256 {
		return empty, nil, fmt.Errorf("import %q has inconsistent manifest evidence", alias)
	}
	read := func(path string) ([]byte, error) {
		normalized, err := sourceindex.NormalizeWorkspacePath(path)
		if err != nil || normalized != path {
			return nil, fmt.Errorf("import %q requires a canonical source path", alias)
		}
		raw, exists := snapshot[path]
		if !exists {
			return nil, fmt.Errorf("import %q source %q is outside its captured read scope", alias, path)
		}
		return bytes.Clone(raw), nil
	}
	out := capture.evidence
	out.ParsedSources = append([]sourceindex.Artifact(nil), capture.evidence.ParsedSources...)
	return out, read, nil
}
