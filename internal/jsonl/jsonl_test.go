package jsonl_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/jsonl"
)

type record struct {
	Name    string `json:"name"`
	Allowed bool   `json:"allowed"`
}

func TestAppendAndReadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.jsonl")
	for _, name := range []string{"first", "second"} {
		if err := jsonl.Append(path, "test journal", record{Name: name, Allowed: name == "second"}); err != nil {
			t.Fatalf("append %s: %v", name, err)
		}
	}
	records, err := jsonl.Read(path, "test journal", jsonl.DecodeJSON[record])
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(records) != 2 || records[0].Name != "first" || !records[1].Allowed {
		t.Fatalf("unexpected records: %+v", records)
	}
}

// TestTheFileIsOwnerOnly pins the rule the four copies each had to remember: a
// record other users can rewrite is a record that can be made to say something
// else.
func TestTheFileIsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not observable on Windows; the Linux CI run checks it")
	}
	path := filepath.Join(t.TempDir(), "records.jsonl")
	if err := jsonl.Append(path, "test journal", record{Name: "x"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("the file is readable or writable by others: %v", info.Mode().Perm())
	}
}

func TestAMissingFileIsEmpty(t *testing.T) {
	records, err := jsonl.Read(filepath.Join(t.TempDir(), "none.jsonl"), "test journal", jsonl.DecodeJSON[record])
	if err != nil || records != nil {
		t.Fatalf("a missing file must read as empty, not as an error: %+v err=%v", records, err)
	}
}

// TestADamagedLineIsAnErrorAndKeepsTheEarlierRecords pins the choice: corruption
// is not "nothing after this happened". The earlier records are still returned,
// so a caller can say how far the file is trustworthy.
func TestADamagedLineIsAnErrorAndKeepsTheEarlierRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.jsonl")
	if err := jsonl.Append(path, "test journal", record{Name: "kept"}); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("not-json\n"); err != nil {
		t.Fatal(err)
	}
	file.Close()

	records, err := jsonl.Read(path, "test journal", jsonl.DecodeJSON[record])
	if err == nil {
		t.Fatal("a damaged line must be reported")
	}
	if len(records) != 1 || records[0].Name != "kept" {
		t.Fatalf("the records before the damage must be returned: %+v", records)
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("the error must name the line, so the damage can be found: %v", err)
	}
}

func TestABlankLineIsSkipped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.jsonl")
	if err := jsonl.Append(path, "test journal", record{Name: "a"}); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("\n\n"); err != nil {
		t.Fatal(err)
	}
	file.Close()
	if err := jsonl.Append(path, "test journal", record{Name: "b"}); err != nil {
		t.Fatal(err)
	}
	records, err := jsonl.Read(path, "test journal", jsonl.DecodeJSON[record])
	if err != nil || len(records) != 2 {
		t.Fatalf("blank lines are not records: %+v err=%v", records, err)
	}
}

// TestTheLabelReachesTheError keeps the callers distinguishable: four journals
// share this code, and a message that does not say which one failed sends the
// reader to the wrong file.
func TestTheLabelReachesTheError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.jsonl")
	if err := os.WriteFile(path, []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := jsonl.Read(path, "effect journal", jsonl.DecodeJSON[record])
	if err == nil || !strings.Contains(err.Error(), "effect journal") {
		t.Fatalf("the error must name the journal: %v", err)
	}
}
