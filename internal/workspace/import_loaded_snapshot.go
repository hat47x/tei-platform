package workspace

import (
	"bytes"
	"fmt"
	"sort"

	"tei/internal/sourceindex"
)

// importedParsedLoadEvidence is internal-only metadata for the selected
// Namespace and parsed Source bytes captured during the SAME provider Load as
// the consumer's Workspace Import. Manifest is a separate input, never a
// consumer-managed Source Index artifact.
//
// FullInputClosure is always false: provider-wide inputs, negative dependencies,
// multi-file atomicity, current Provider authority and downstream consumers
// remain unqualified. This is not an Acceptance or execution grant.
type importedParsedLoadEvidence struct {
	Alias            string
	ProviderPath     string
	Namespace        string
	SemanticDigest   string
	Manifest         ManifestIdentity
	ParsedSources    []sourceindex.Artifact
	FullInputClosure bool
}

// manifestReadBytes is an opaque pointer in the publicly returned Result.
// Ordinary fmt formatting of Result must not traverse the retained raw bytes.
// This is not authorization against trusted in-process code or reflection.
type manifestReadBytes struct {
	content []byte
}

// importedParsedLoad is private, transient provenance. No raw bytes, Source
// occurrence paths, fingerprints or Manifest details are projected by a new
// public accessor. Importing a Semantic Namespace is not disclosure authority.
// Existing Result.Imports remains a distinct pre-existing public projection.
type importedParsedLoad struct {
	evidence importedParsedLoadEvidence
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
		provider.sourceManifest.Path == "" || provider.sourceManifestBytes == nil || len(provider.sourceManifestBytes.content) == 0 {
		return empty, fmt.Errorf("import %q lacks same-Load parsed Source or manifest evidence", imp.Alias)
	}
	if sourceindex.Fingerprint(provider.sourceManifestBytes.content) != provider.sourceManifest.SHA256 {
		return empty, fmt.Errorf("import %q has inconsistent Load-time manifest identity", imp.Alias)
	}

	files := make(map[string][]byte, len(paths)+1)
	files[provider.sourceManifest.Path] = bytes.Clone(provider.sourceManifestBytes.content)
	evidence := importedParsedLoadEvidence{
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
			return empty, fmt.Errorf("import %q has invalid parsed Source path", imp.Alias)
		}
		raw, ok := provider.sourceSnapshot[path]
		if !ok {
			return empty, fmt.Errorf("import %q parsed Source has no Load-time bytes", imp.Alias)
		}
		if _, duplicate := files[path]; duplicate {
			return empty, fmt.Errorf("import %q has duplicate parsed Source occurrence", imp.Alias)
		}
		artifact, err := sourceindex.NewArtifact(path, raw)
		if err != nil {
			return empty, fmt.Errorf("import %q has invalid parsed Source evidence", imp.Alias)
		}
		files[path] = bytes.Clone(raw)
		evidence.ParsedSources = append(evidence.ParsedSources, artifact)
	}
	return importedParsedLoad{evidence: evidence, files: files}, nil
}

// importedParsedLoadSnapshot retrieves raw bytes for trusted internal
// Workspace operations ONLY. No filesystem read or fresh Provider Load occurs.
// It returns clone-on-read bytes of selected parsed Sources and the Provider
// Manifest. It is intentionally unexported: an explicit Workspace Import is
// not a grant to disclose Provider source text, comments or full Manifest.
//
// A future external Consumer must pass the Provider's existing disclosure and
// authorization boundary before such raw data may be projected. This function
// does not establish full input closure or multi-file atomicity.
func importedParsedLoadSnapshot(loaded Result, alias string) (importedParsedLoadEvidence, func(string) ([]byte, error), error) {
	var empty importedParsedLoadEvidence
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
	if !ok || capture == nil || capture.evidence.Alias != alias ||
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
