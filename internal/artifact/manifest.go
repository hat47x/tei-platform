package artifact

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

// ManifestEntry is one artifact's identity inside a set.
type ManifestEntry struct {
	Kind        string `json:"kind"`
	Rule        string `json:"rule"`
	Version     string `json:"version"`
	Fingerprint string `json:"fingerprint"`
	Digest      string `json:"digest"`
	Unresolved  int    `json:"unresolved"`
}

// Manifest is a deterministic set of derived artifacts that share one input
// revision. It answers "what was produced, and is any of it unresolved" without
// carrying the artifact bodies.
type Manifest struct {
	InputRevision string          `json:"inputRevision"`
	Entries       []ManifestEntry `json:"entries"`
	Digest        string          `json:"digest"`
	// Complete is true only when every artifact has no unresolved entry. An
	// unresolved artifact is still listed; it is not a successful set.
	Complete bool `json:"complete"`
}

// NewManifest builds the manifest of a set of artifacts. Artifacts must share
// one input revision: a set that mixes revisions can not be compared or
// regenerated as a unit, so it is refused instead of being listed anyway.
func NewManifest(inputRevision string, artifacts []Artifact) (Manifest, error) {
	if inputRevision == "" {
		return Manifest{}, fmt.Errorf("manifest input revision is required")
	}
	if len(artifacts) == 0 {
		return Manifest{}, fmt.Errorf("manifest requires at least one artifact")
	}
	for _, artifact := range artifacts {
		if artifact.InputRevision != inputRevision {
			return Manifest{}, fmt.Errorf("artifact %s was derived from %s, not %s", artifact.Kind, artifact.InputRevision, inputRevision)
		}
		if err := artifact.Verify(); err != nil {
			return Manifest{}, fmt.Errorf("artifact %s does not verify: %w", artifact.Kind, err)
		}
	}

	entries := make([]ManifestEntry, 0, len(artifacts))
	complete := true
	for _, artifact := range artifacts {
		entries = append(entries, ManifestEntry{
			Kind:        artifact.Kind,
			Rule:        artifact.Rule.ID,
			Version:     artifact.Rule.Version,
			Fingerprint: artifact.Rule.Fingerprint,
			Digest:      artifact.Digest,
			Unresolved:  len(artifact.Unresolved),
		})
		if len(artifact.Unresolved) > 0 {
			complete = false
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		if entries[i].Rule != entries[j].Rule {
			return entries[i].Rule < entries[j].Rule
		}
		return entries[i].Digest < entries[j].Digest
	})

	manifest := Manifest{
		InputRevision: inputRevision,
		Entries:       entries,
		Complete:      complete,
	}
	manifest.Digest = fingerprintManifest(manifest)
	return manifest, nil
}

// Verify recomputes the manifest digest.
func (m Manifest) Verify() error {
	if m.Digest != fingerprintManifest(m) {
		return fmt.Errorf("manifest digest does not match its entries")
	}
	return nil
}

func fingerprintManifest(m Manifest) string {
	var b strings.Builder
	b.WriteString("tei.artifact.manifest/v0\n")
	b.WriteString(m.InputRevision + "\n")
	for _, entry := range m.Entries {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\t%d\n", entry.Kind, entry.Rule, entry.Version, entry.Fingerprint, entry.Digest, entry.Unresolved)
	}
	b.WriteString(fmt.Sprintf("complete=%t\n", m.Complete))
	return sourceindex.Fingerprint([]byte(b.String()))
}
