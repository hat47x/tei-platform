package access_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

// TestRecordHashCoversEveryFieldOfTheRecord pins the property the hand-written
// hash did not have: a field added to Record is covered without anyone
// remembering to add it to the hash input.
//
// It works by hashing two records that differ only in one field, and passing
// only when the hashes differ. When Record gains a field, this list must grow
// with it; the test fails loudly if the hash stops covering one.
func TestRecordHashCoversEveryFieldOfTheRecord(t *testing.T) {
	base := access.Record{Capability: "sales.close", Allowed: false, Missing: []string{"network:connect db:5432"}}
	variants := map[string]access.Record{
		"capability": {Capability: "sales.reopen", Allowed: false, Missing: base.Missing},
		"allowed":    {Capability: base.Capability, Allowed: true, Missing: base.Missing},
		"missing":    {Capability: base.Capability, Allowed: false, Missing: nil},
	}
	for name, variant := range variants {
		dir := t.TempDir()
		pathA := filepath.Join(dir, "a.jsonl")
		pathB := filepath.Join(dir, "b.jsonl")
		if err := (access.ChainFile{Path: pathA}).Append(base); err != nil {
			t.Fatalf("%s: append a: %v", name, err)
		}
		if err := (access.ChainFile{Path: pathB}).Append(variant); err != nil {
			t.Fatalf("%s: append b: %v", name, err)
		}
		a, err := (access.ChainFile{Path: pathA}).Head()
		if err != nil {
			t.Fatalf("%s: head a: %v", name, err)
		}
		b, err := (access.ChainFile{Path: pathB}).Head()
		if err != nil {
			t.Fatalf("%s: head b: %v", name, err)
		}
		if a == b {
			t.Fatalf("changing %s did not change the hash, so it is not covered", name)
		}
	}
}

// TestChainFileRefusesARecordFromAnotherRule checks the version marker: a record
// written by another rule is reported as a version mismatch, not as tampering.
func TestChainFileRefusesARecordFromAnotherRule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.jsonl")
	content := `{"capability":"sales.close","allowed":false,"chainVersion":"tei.access.chain/v1","prevHash":"sha256:` +
		strings.Repeat("0", 64) + `","hash":"sha256:` + strings.Repeat("a", 64) + `"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	err := (access.ChainFile{Path: path}).Verify()
	if err == nil {
		t.Fatal("a record from another chain rule must not verify")
	}
	if !strings.Contains(err.Error(), "was written under chain") {
		t.Fatalf("the refusal must name the version, not just the hash: %v", err)
	}
}

// TestChainFileWritesTheCurrentVersion checks that the marker is present, so a
// reader does not have to guess which rule wrote the file.
func TestChainFileWritesTheCurrentVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.jsonl")
	guard := access.Guard{
		Granted: []permission.Spec{{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:db"}},
		Journal: access.ChainFile{Path: path},
	}
	if _, err := guard.Check(core.MustCanonicalID("sales.close"), []permission.Spec{{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:other"}}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"chainVersion":"tei.access.chain/v2"`) {
		t.Fatalf("the record must name the chain rule that wrote it: %s", raw)
	}
}
