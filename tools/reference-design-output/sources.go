package designoutput

import (
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

// addFile appends one derived file and its manifest entry. The digest is over
// the exact bytes written, so a reader can verify the file without trusting the
// manifest.
func addFile(bundle *Bundle, name, kind, schema, structureRevision, path string, content []byte, unresolved []string) {
	bundle.Files = append(bundle.Files, File{Path: path, Content: content})
	artifact := Artifact{
		Name: name, Kind: kind, Schema: schema, Path: path,
		Bytes: len(content), Digest: digestOf(content),
		StructureRevision: structureRevision,
	}
	if len(unresolved) > 0 {
		artifact.Unresolved = append([]string(nil), unresolved...)
		sort.Strings(artifact.Unresolved)
	}
	bundle.Manifest.Artifacts = append(bundle.Manifest.Artifacts, artifact)
}

// withExtension replaces the base name's extension. The base has no extension,
// so this never strips a real name.
func withExtension(base, extension string) string {
	return strings.TrimSuffix(base, "."+extension) + "." + extension
}

// editableSources lists the canonical workspace files a user may edit. Only
// definition and authoring roles are exposed: implementation and derived
// artifacts are not a second source. Duplicate paths are kept once with the
// first fingerprint.
func editableSources(index sourceindex.Index) []EditableSource {
	fingerprints := make(map[string]string, len(index.Artifacts))
	for _, artifact := range index.Artifacts {
		fingerprints[artifact.Path] = artifact.Fingerprint
	}
	seen := map[string]bool{}
	var out []EditableSource
	for _, reference := range index.References {
		if reference.Role != sourceindex.RoleDefinition && reference.Role != sourceindex.RoleAuthoring {
			continue
		}
		path := strings.TrimSpace(reference.ArtifactPath)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, EditableSource{Path: path, Fingerprint: fingerprints[path], Role: string(reference.Role)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
