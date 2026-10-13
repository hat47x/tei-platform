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

// ImportedSourceOrigin is a read-only navigation hint into the provider's
// Workspace. ArtifactPath is relative to that provider, never to the consumer.
// Neither a source anchor nor its digest grants acceptance or edit authority.
type ImportedSourceOrigin struct {
	Alias        string                `json:"alias"`
	ProviderPath string                `json:"providerPath"`
	Namespace    string                `json:"namespace"`
	Subject      sourceindex.Subject   `json:"subject"`
	Artifact     sourceindex.Artifact  `json:"artifact"`
	Reference    sourceindex.Reference `json:"reference"`
}

// LocateImportedSource builds navigation evidence on demand. The provider is
// freshly loaded and its normalized semantic digest must match the consumer's
// resolved import. Source files remain outside the consumer Source Index.
// A current provider snapshot is not a guarantee that the provider's physical
// file layout has stayed unchanged since the earlier consumer Load.
func LocateImportedSource(consumerRoot string, loaded Result, alias string, target sourceindex.Subject) (ImportedSourceOrigin, error) {
	var match *ResolvedImport
	for i := range loaded.Imports {
		if loaded.Imports[i].Alias != alias {
			continue
		}
		if match != nil {
			return ImportedSourceOrigin{}, fmt.Errorf("import alias %q is ambiguous", alias)
		}
		match = &loaded.Imports[i]
	}
	if match == nil {
		return ImportedSourceOrigin{}, fmt.Errorf("import alias %q is not resolved", alias)
	}
	if match.Path == "" || match.Path == "." || path.IsAbs(match.Path) ||
		strings.ContainsAny(match.Path, "\\:") || path.Clean(match.Path) != match.Path {
		return ImportedSourceOrigin{}, fmt.Errorf("import %q has invalid provider path %q", alias, match.Path)
	}
	root := filepath.Join(consumerRoot, filepath.FromSlash(match.Path))
	provider, err := Load(root)
	if err != nil {
		return ImportedSourceOrigin{}, fmt.Errorf("import %q provider: %w", alias, err)
	}
	if _, ok := provider.Sources["semantic:"+match.Namespace]; !ok {
		return ImportedSourceOrigin{}, fmt.Errorf("import %q provider no longer owns namespace %q", alias, match.Namespace)
	}
	var selected *semantic.Namespace
	for i := range provider.World.Semantics {
		if string(provider.World.Semantics[i].ID) == match.Namespace {
			selected = &provider.World.Semantics[i]
			break
		}
	}
	if selected == nil {
		return ImportedSourceOrigin{}, fmt.Errorf("import %q namespace %q disappeared", alias, match.Namespace)
	}
	bytes, err := json.Marshal(selected)
	if err != nil {
		return ImportedSourceOrigin{}, fmt.Errorf("import %q fingerprint: %w", alias, err)
	}
	if sourceindex.Fingerprint(bytes) != match.SemanticDigest {
		return ImportedSourceOrigin{}, fmt.Errorf("import %q semantic digest is stale", alias)
	}
	if !importedNamespaceOwnsSubject(*selected, target) {
		return ImportedSourceOrigin{}, fmt.Errorf("import %q subject %s:%s is not part of namespace %q", alias, target.Kind, target.ID, match.Namespace)
	}
	idx, err := BuildSourceIndex(root, provider)
	if err != nil {
		return ImportedSourceOrigin{}, fmt.Errorf("import %q provider source index: %w", alias, err)
	}
	var ref *sourceindex.Reference
	for i := range idx.References {
		if idx.References[i].Subject != target {
			continue
		}
		if ref != nil {
			return ImportedSourceOrigin{}, fmt.Errorf("import %q subject %s:%s has ambiguous source anchors", alias, target.Kind, target.ID)
		}
		ref = &idx.References[i]
	}
	if ref == nil {
		return ImportedSourceOrigin{}, fmt.Errorf("import %q subject %s:%s has no source anchor", alias, target.Kind, target.ID)
	}
	for _, artifact := range idx.Artifacts {
		if artifact.Path == ref.ArtifactPath {
			return ImportedSourceOrigin{
				Alias:        match.Alias,
				ProviderPath: match.Path,
				Namespace:    match.Namespace,
				Subject:      target,
				Artifact:     artifact,
				Reference:    *ref,
			}, nil
		}
	}
	return ImportedSourceOrigin{}, fmt.Errorf("import %q subject %s:%s source artifact is missing", alias, target.Kind, target.ID)
}

// A source locator must never select a similarly named Subject in an
// unimported namespace, even if the provider also owns that declaration.
func importedNamespaceOwnsSubject(ns semantic.Namespace, subject sourceindex.Subject) bool {
	same := func(kind, id string) bool { return subject.Kind == kind && subject.ID == id }
	if same("semantic", string(ns.ID)) {
		return true
	}
	for _, typ := range ns.Types {
		if same("semantic-type", string(typ.ID)) {
			return true
		}
	}
	for _, data := range ns.Data {
		if same("semantic-data", string(data.ID)) {
			return true
		}
		for _, attr := range data.Attributes {
			if same("semantic-attribute", string(attr.ID)) {
				return true
			}
		}
		for _, relation := range data.Relations {
			if same("semantic-relation", string(relation.ID)) {
				return true
			}
		}
		for _, key := range data.CandidateKeys {
			if same("semantic-candidate-key", string(key.ID)) {
				return true
			}
		}
		for _, dep := range data.FunctionalDependencies {
			if same("semantic-functional-dependency", string(dep.ID)) {
				return true
			}
		}
	}
	for _, cap := range ns.Capabilities {
		if same("semantic-capability", string(cap.ID)) {
			return true
		}
	}
	for _, rule := range ns.Validations {
		if same("semantic-validation", string(rule.ID)) {
			return true
		}
	}
	return false
}
