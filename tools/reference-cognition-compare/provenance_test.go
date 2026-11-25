package main

// The provenance record is what makes a rate report traceable: which model
// artifact, which template, and which request bytes produced which numbers.
// The record's value is in what it refuses to write: a digest is only present
// when the bytes were read, and a fact that could not be established is listed
// in Unrecorded rather than left blank, because a blank field reads as "not
// applicable" while the truth is "not observed".

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func runBaseline(t *testing.T) ProviderResult {
	t.Helper()
	set, err := readCaseSet(filepath.Join("..", "reference-cognition-port", "testdata", "cases", "choice-baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	return runSet(set, lexicalProvider())
}

func readCaseSet(path string) (CaseSet, error) {
	var set CaseSet
	raw, err := os.ReadFile(path)
	if err != nil {
		return set, err
	}
	decoder := json.NewDecoder(bytes.NewReader(stripBOM(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&set); err != nil {
		return set, err
	}
	return set, nil
}

func TestPerCaseFingerprintsAreRecorded(t *testing.T) {
	result := runBaseline(t)
	if len(result.Outcomes) == 0 {
		t.Fatal("no cases ran")
	}
	for _, outcome := range result.Outcomes {
		// Every case sends a context, so every case has a request digest. A
		// case that reached the provider also has a reply digest; the baseline
		// always answers, so its cases have both.
		if outcome.RequestDigest == "" {
			t.Fatalf("case %s has no request digest", outcome.ID)
		}
		if outcome.ReplyDigest == "" {
			t.Fatalf("case %s has no reply digest", outcome.ID)
		}
	}
	// Two different cases must not fingerprint identically, or the digests
	// carry no information.
	first, second := result.Outcomes[0], result.Outcomes[1]
	if first.RequestDigest == second.RequestDigest {
		t.Fatal("different cases produced the same request digest")
	}
}

func TestProvenanceRecordsTheHarnessAndCaseSet(t *testing.T) {
	record := newProvenance("sha256:"+repeat("a", 64), "cases.json")
	if record.Schema != ProvenanceSchema {
		t.Fatalf("schema = %q", record.Schema)
	}
	if record.Harness == "" || record.CaseSetDigest == "" {
		t.Fatalf("harness or case set digest is missing: %+v", record)
	}
	path := filepath.Join(t.TempDir(), "provenance.json")
	if err := writeProvenance(path, record); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var readBack Provenance
	if err := json.Unmarshal(raw, &readBack); err != nil {
		t.Fatal(err)
	}
	if readBack.CaseSetDigest != record.CaseSetDigest {
		t.Fatal("the written record did not round-trip")
	}
}

// An artifact that cannot be read is reported as unrecorded, never as a zero
// digest: two different models would then fingerprint identically, and the
// record would claim a provenance it does not have.
func TestArtifactDigestRefusesToInventOne(t *testing.T) {
	digest, size, err := artifactDigest(filepath.Join(t.TempDir(), "absent.gguf"))
	if err == nil {
		t.Fatal("an absent artifact must be an error")
	}
	if digest != "" || size != 0 {
		t.Fatalf("a failed read produced %q / %d", digest, size)
	}

	// A readable artifact produces a real digest, and the same bytes always
	// produce the same one.
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.bin")
	if err := os.WriteFile(path, []byte("model bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, firstSize, err := artifactDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	second, secondSize, err := artifactDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || firstSize != secondSize || firstSize != 11 {
		t.Fatalf("digest or size is not stable: %q/%d vs %q/%d", first, firstSize, second, secondSize)
	}
}

// The per-case record copies the fingerprints in run order, so a provenance
// file lines up with the report it accompanies.
func TestCaseProvenanceCopiesTheFingerprints(t *testing.T) {
	result := runBaseline(t)
	entry := caseProvenance(result)
	if len(entry.Cases) != len(result.Outcomes) {
		t.Fatalf("case count = %d, want %d", len(entry.Cases), len(result.Outcomes))
	}
	for index, outcome := range result.Outcomes {
		if entry.Cases[index].Case != outcome.ID {
			t.Fatalf("case %d recorded %q, want %q", index, entry.Cases[index].Case, outcome.ID)
		}
		if entry.Cases[index].RequestDigest != outcome.RequestDigest {
			t.Fatalf("case %s recorded a different request digest", outcome.ID)
		}
		if entry.Cases[index].Millis != outcome.Millis {
			t.Fatalf("case %s recorded a different duration", outcome.ID)
		}
	}
}

// A template that a server does not disclose is not an empty template: an
// empty digest would look like a real fingerprint of nothing.
func TestTemplateDigestIsEmptyOnlyWhenThereIsNoTemplate(t *testing.T) {
	if templateDigest("") != "" {
		t.Fatal("an absent template must not produce a digest")
	}
	if got := templateDigest("{% for m in messages %}"); got == "" {
		t.Fatal("a real template must produce a digest")
	}
}

func repeat(value string, count int) string {
	out := ""
	for i := 0; i < count; i++ {
		out += value
	}
	return out
}
