package designoutput

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/environment"
)

func TestVerifyMatchesCurrentDerivationWithoutWriting(t *testing.T) {
	for _, code := range []bool{false, true} {
		t.Run(map[bool]string{false: "documents", true: "with code"}[code], func(t *testing.T) {
			root := referenceWorkspace(t)
			dir := filepath.Join(t.TempDir(), "bundle")
			raw := generatorContract(t)
			var err error
			if code {
				_, err = WriteWithGenerator(root, dir, "postgres", raw, "template", "1")
			} else {
				_, err = Write(root, dir, "postgres")
			}
			if err != nil {
				t.Fatal(err)
			}
			before, sourceBefore := readTree(t, dir), readTree(t, root)
			// Verification re-renders in memory, with no external build tool available.
			t.Setenv("PATH", t.TempDir())
			if code {
				_, err = VerifyWithGenerator(root, dir, "postgres", raw, "template", "1")
			} else {
				_, err = Verify(root, dir, "postgres")
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, readTree(t, dir)) || !reflect.DeepEqual(sourceBefore, readTree(t, root)) {
				t.Fatal("verification changed bundle or Workspace")
			}
		})
	}
}

func TestVerifyRejectsModifiedMissingAndExtraFiles(t *testing.T) {
	for _, change := range []string{"document", "same-size document", "source", "candidate", "contract", "index", "missing", "extra", "manifest", "unsafe manifest path", "rewritten digests"} {
		t.Run(change, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "bundle")
			raw := generatorContract(t)
			manifest, err := WriteWithGenerator(referenceWorkspace(t), dir, "postgres", raw, "template", "1")
			if err != nil {
				t.Fatal(err)
			}
			put := func(name string, b []byte) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			switch change {
			case "document":
				put("table-definitions.md", []byte("edited document"))
			case "same-size document":
				b, err := os.ReadFile(filepath.Join(dir, "table-definitions.md"))
				if err != nil || len(b) == 0 {
					t.Fatalf("read document: %v", err)
				}
				b[0] ^= 1
				put("table-definitions.md", b)
			case "source":
				put("generated/main.go", []byte("edited source"))
			case "candidate":
				put("generated/candidate.json", []byte("{}"))
			case "contract":
				put("generated/contract.json", []byte("{}"))
			case "index":
				put("index.html", []byte("edited navigation"))
			case "missing":
				if err := os.Remove(filepath.Join(dir, "function-definitions.md")); err != nil {
					t.Fatal(err)
				}
			case "extra":
				put("unlisted.txt", []byte("unverified addition"))
			case "manifest":
				put("manifest.json", []byte("{}"))
			case "unsafe manifest path":
				manifest.Artifacts[0].Path = "../outside.txt"
				b, err := json.MarshalIndent(manifest, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				put("manifest.json", append(b, '\n'))
			case "rewritten digests":
				// Locally consistent hashes must not replace the caller-selected derivation.
				b, err := os.ReadFile(filepath.Join(dir, "table-definitions.md"))
				if err != nil || len(b) == 0 {
					t.Fatalf("read document: %v", err)
				}
				b[0] ^= 1
				put("table-definitions.md", b)
				for i := range manifest.Artifacts {
					a := &manifest.Artifacts[i]
					b, err := os.ReadFile(filepath.Join(dir, a.Path))
					if err != nil {
						t.Fatal(err)
					}
					a.Digest = digestOf(b)
					a.Bytes = len(b)
				}
				b, err = json.MarshalIndent(manifest, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				put("manifest.json", append(b, '\n'))
			}
			before := readTree(t, dir)
			if _, err := VerifyWithGenerator(referenceWorkspace(t), dir, "postgres", raw, "template", "1"); err == nil {
				t.Fatal("changed bundle accepted")
			}
			if !reflect.DeepEqual(before, readTree(t, dir)) {
				t.Fatal("verification modified the saved bundle")
			}
		})
	}
}

func TestVerifyRejectsChangedInputs(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.CopyFS(root, os.DirFS(referenceWorkspace(t))); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "bundle")
	raw := generatorContract(t)
	if _, err := WriteWithGenerator(root, dir, "postgres", raw, "template", "1"); err != nil {
		t.Fatal(err)
	}
	for _, settings := range []struct {
		name, strategy, version, provider string
		contract                          []byte
	}{
		{"generator version", "template", "2", "postgres", raw},
		{"strategy", "alternative", "1", "postgres", raw},
		{"contract bytes", "template", "1", "postgres", append(append([]byte(nil), raw...), '\n')},
		{"provider", "template", "1", "unknown-provider", raw},
	} {
		t.Run(settings.name, func(t *testing.T) {
			if _, err := VerifyWithGenerator(root, dir, environment.DatabaseProvider(settings.provider), settings.contract, settings.strategy, settings.version); err == nil {
				t.Fatal("different inputs accepted")
			}
		})
	}
	if _, err := Verify(root, dir, "postgres"); err == nil {
		t.Fatal("code bundle accepted without caller-selected contract")
	}
	var source string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if source == "" && !entry.IsDir() && strings.HasSuffix(path, ".tei") {
			source = path
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if source == "" {
		t.Fatal("no Source file found")
	}
	b, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	// A byte-only Source revision change is distinct from a semantic change.
	if err := os.WriteFile(source, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyWithGenerator(root, dir, "postgres", raw, "template", "1"); err == nil {
		t.Fatal("stale Source revision accepted")
	}
}

func TestVerifyRejectsSymlinksAndDoesNotCreateOutput(t *testing.T) {
	parent := t.TempDir()
	missing := filepath.Join(parent, "missing")
	if _, err := Verify(referenceWorkspace(t), missing, "postgres"); err == nil {
		t.Fatal("missing bundle accepted")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("verification created an output: %v", err)
	}
	dir := filepath.Join(parent, "bundle")
	if _, err := Write(referenceWorkspace(t), dir, "postgres"); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(dir, "table-definitions.md"))
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.md")
	if err := os.WriteFile(outside, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "table-definitions.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "table-definitions.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Verify(referenceWorkspace(t), dir, "postgres"); err == nil {
		t.Fatal("external symlink accepted")
	}
	after, err := os.ReadFile(outside)
	if err != nil || !reflect.DeepEqual(original, after) {
		t.Fatal("external file changed")
	}
}
