package okf

import (
	"os"
	"strings"
	"testing"
)

// TestReservedFilesAreNotConcepts covers OKF v0.2 section 3.1: index.md and log.md
// have defined meaning and must not be used for concept documents.
func TestReservedFilesAreNotConcepts(t *testing.T) {
	bundle := readFixture(t)
	for _, concept := range bundle.Concepts {
		base := concept.Path
		if index := strings.LastIndex(base, "/"); index >= 0 {
			base = base[index+1:]
		}
		if base == "index.md" || base == "log.md" {
			t.Fatalf("reserved file %s was treated as a concept", concept.Path)
		}
	}
	if len(bundle.Indexes) != 1 {
		t.Fatalf("index count = %d", len(bundle.Indexes))
	}
}

// TestIndexEntriesAreRead covers section 8's listing shape, including the optional
// description and the relative target.
func TestIndexEntriesAreRead(t *testing.T) {
	bundle := readFixture(t)
	if len(bundle.Indexes) != 1 {
		t.Fatalf("index count = %d", len(bundle.Indexes))
	}
	index := bundle.Indexes[0]
	if !index.IsRoot {
		t.Fatal("the fixture index is the bundle root")
	}
	if len(index.Entries) != 3 {
		t.Fatalf("entries = %d, want 3: %+v", len(index.Entries), index.Entries)
	}
	targets := map[string]bool{}
	for _, entry := range index.Entries {
		targets[entry.Target] = true
		if entry.Resolution != ResolutionDocumentRelative {
			t.Errorf("entry %q resolution = %q, want document-relative", entry.Target, entry.Resolution)
		}
		if len(entry.Resolved) != 1 {
			t.Errorf("entry %q resolved = %v, want one path", entry.Target, entry.Resolved)
		}
	}
	for _, want := range []string{"data/order.md", "functions/submit.md", "business/submit.md"} {
		if !targets[want] {
			t.Errorf("index is missing entry %q; got %v", want, targets)
		}
	}
}

// TestIndexEntryWithMissingTargetIsObserved covers an index whose target does not
// exist: the entry stays readable and the gap is reported.
func TestIndexEntryWithMissingTargetIsObserved(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"index.md": "---\nokf_version: \"0.2\"\n---\n\n# Concepts\n\n" +
			"* [Order data](data/order.md) - a synthetic data concept\n" +
			"* [Missing](data/missing.md) - not written yet\n",
	})
	bundle := readFixture(t, mutated)
	if !bundle.Format.Conformant {
		t.Fatalf("an incomplete index must stay conformant; errors: %+v", bundle.Format.FormatErrors)
	}
	if len(bundle.Indexes) != 1 {
		t.Fatalf("index count = %d", len(bundle.Indexes))
	}
	if !hasCode(bundle.Indexes[0].Diagnostics, CodeIndexEntryTargetMissing) {
		t.Fatalf("index codes = %v, want %s", codes(bundle.Indexes[0].Diagnostics), CodeIndexEntryTargetMissing)
	}
}

// TestIndexEntryOutsideTheBundleIsObserved covers an escaping index target.
func TestIndexEntryOutsideTheBundleIsObserved(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"index.md": "---\nokf_version: \"0.2\"\n---\n\n# Concepts\n\n" +
			"* [Outside](../outside.md) - a target that leaves the bundle\n",
	})
	bundle := readFixture(t, mutated)
	if !bundle.Format.Conformant {
		t.Fatalf("an escaping index target must stay conformant; errors: %+v", bundle.Format.FormatErrors)
	}
	if !hasCode(bundle.Indexes[0].Diagnostics, CodeIndexEntryTargetEscapes) {
		t.Fatalf("index codes = %v, want %s", codes(bundle.Indexes[0].Diagnostics), CodeIndexEntryTargetEscapes)
	}
}

// TestLogEntriesAreRead covers section 9's date-grouped shape. The upstream sample
// bundle carries frontmatter in log.md, so that shape is read and reported rather
// than rejected.
func TestLogEntriesAreRead(t *testing.T) {
	logBody := "---\ntype: Log\ntitle: Bundle history\n---\n\n# Bundle history\n\n" +
		"## 2026-07-01\n\n" +
		"- **Verified** the bundle for OKF v0.2 conformance.\n\n" +
		"## 2026-06-30\n\n" +
		"- **Re-generated** the revenue concept.\n" +
		"- **Deprecated** the legacy definition.\n"
	mutated := mutateFixture(t, map[string]string{"log.md": logBody})
	bundle := readFixture(t, mutated)
	if len(bundle.Logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(bundle.Logs))
	}
	log := bundle.Logs[0]
	if !log.HasFrontmatter {
		t.Fatal("the reader must record the frontmatter block")
	}
	if !hasCode(log.Diagnostics, CodeLogFrontmatterPresent) {
		t.Fatalf("log codes = %v, want %s", codes(log.Diagnostics), CodeLogFrontmatterPresent)
	}
	if len(log.Entries) != 2 {
		t.Fatalf("date groups = %d, want 2: %+v", len(log.Entries), log.Entries)
	}
	if log.Entries[0].Date != "2026-07-01" || len(log.Entries[0].Entries) != 1 {
		t.Fatalf("first group = %+v", log.Entries[0])
	}
	if log.Entries[1].Date != "2026-06-30" || len(log.Entries[1].Entries) != 2 {
		t.Fatalf("second group = %+v", log.Entries[1])
	}
	if findConcept(bundle, "log") != nil {
		t.Fatal("log.md must not appear as a concept")
	}
}

// TestLogNonISODateIsObserved covers section 9's date requirement. A free-form
// heading ends the current group rather than being read as a date.
func TestLogNonISODateIsObserved(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"log.md": "# History\n\n## July 1, 2026\n\n- something happened\n\n## 2026-06-30\n\n- a real date group\n",
	})
	bundle := readFixture(t, mutated)
	if len(bundle.Logs) != 1 {
		t.Fatalf("log count = %d", len(bundle.Logs))
	}
	log := bundle.Logs[0]
	if !hasCode(log.Diagnostics, CodeLogDateNotISO) {
		t.Fatalf("log codes = %v, want %s", codes(log.Diagnostics), CodeLogDateNotISO)
	}
	if len(log.Entries) != 1 || log.Entries[0].Date != "2026-06-30" {
		t.Fatalf("entries = %+v, want only the ISO-dated group", log.Entries)
	}
}

// nonASCIIConceptName is a two-character Japanese filename stem, written as escapes
// so this file stays pure ASCII.
const nonASCIIConceptName = "\u6ce8\u6587"

// TestNonASCIIConceptIDsAreIdentifiers covers a bundle whose paths are not ASCII.
// The upstream reference's path helper rejects these; this reader is a format
// reader, so it keeps the path as the identifier rather than racing the reference.
func TestNonASCIIConceptIDsAreIdentifiers(t *testing.T) {
	mutated := mutateFixture(t, map[string]string{
		"knowledge/" + nonASCIIConceptName + ".md": "---\ntype: Data Entity\ntitle: " + nonASCIIConceptName + "\n---\n\nbody\n",
	})
	bundle := readFixture(t, mutated)
	concept := findConcept(bundle, "knowledge/"+nonASCIIConceptName)
	if concept == nil {
		t.Fatalf("missing the non-ASCII concept; got %v", conceptIDs(bundle))
	}
	if !bundle.Format.Conformant {
		t.Fatalf("a non-ASCII path is not a format violation; errors: %+v", bundle.Format.FormatErrors)
	}
}

// TestRevisionIsStableAcrossReadOrder covers determinism: the revision depends on
// the path/digest set, not on directory iteration order.
func TestRevisionIsStableAcrossReadOrder(t *testing.T) {
	first := readFixture(t)
	second := readFixture(t)
	if first.Revision.Value != second.Revision.Value {
		t.Fatalf("the same bytes produced two revisions: %s / %s", first.Revision.Value, second.Revision.Value)
	}
	if !strings.HasPrefix(first.Revision.Value, "sha256:") {
		t.Fatalf("revision = %q, want a sha256 prefix", first.Revision.Value)
	}
	if first.Revision.Algorithm != "sha256" {
		t.Fatalf("algorithm = %q", first.Revision.Algorithm)
	}
}

// TestRevisionCoversEveryFile covers membership sensitivity: adding a file changes
// the revision even when no existing byte changed.
func TestRevisionCoversEveryFile(t *testing.T) {
	baseline := readFixture(t)
	mutated := mutateFixture(t, map[string]string{
		"knowledge/extra.md": "---\ntype: Reference\ntitle: extra\n---\n\nbody\n",
	})
	withExtra := readFixture(t, mutated)
	if baseline.Revision.Value == withExtra.Revision.Value {
		t.Fatal("a new concept file must change the bundle revision")
	}
	if len(withExtra.Concepts) != len(baseline.Concepts)+1 {
		t.Fatalf("concept count = %d, want %d", len(withExtra.Concepts), len(baseline.Concepts)+1)
	}
}

// TestCaseCollisionIsTwoDistinctConcepts covers a Windows-hostile pair. The reader
// reports both files rather than silently collapsing them, because the digest set
// is case-sensitive even where the filesystem is not. On a case-insensitive host
// the two paths cannot coexist, so the probe is skipped there rather than claimed
// as a pass.
func TestCaseCollisionIsTwoDistinctConcepts(t *testing.T) {
	probe := t.TempDir()
	writeFile(t, probe+"/lower.md", "a")
	if _, err := os.Stat(probe + "/LOWER.md"); err == nil {
		t.Skip("this platform treats paths case-insensitively; the collision cannot be constructed")
	}
	mutated := mutateFixture(t, map[string]string{
		"Data/order.md": "---\ntype: Data Entity\ntitle: Uppercase order\n---\n\nbody\n",
	})
	bundle := readFixture(t, mutated)
	upper := findConcept(bundle, "Data/order")
	lower := findConcept(bundle, "data/order")
	if upper == nil || lower == nil {
		t.Fatalf("both paths must be listed; got %v", conceptIDs(bundle))
	}
	if upper.ConceptID == lower.ConceptID {
		t.Fatal("the concept IDs must stay distinct")
	}
}

// TestBaselineHasNoDuplicateConceptIDs is a structural guard: two distinct paths
// that reduce to one concept ID would make the map ambiguous.
func TestBaselineHasNoDuplicateConceptIDs(t *testing.T) {
	bundle := readFixture(t)
	ids := map[string]bool{}
	for _, concept := range bundle.Concepts {
		if ids[concept.ConceptID] {
			t.Fatalf("concept ID %q is duplicated in the baseline", concept.ConceptID)
		}
		ids[concept.ConceptID] = true
	}
}
