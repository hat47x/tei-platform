package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/buildintent"
)

// These tests pin the property that makes the tool worth having: the catalog it
// writes names builder artifacts whose bytes it read. A declaration cannot supply
// a digest, and a path that leaves the root is refused.

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func declarationFor(artifact string) string {
	return `{
  "providers": [
    {
      "plugin": "tei.builder.go",
      "version": "1.2.0",
      "implementation": "go.builder",
      "contract": "tei.plugin-build",
      "contractMajor": 1,
      "artifact": "` + artifact + `",
      "recipeKinds": ["go-package"],
      "targets": ["host/linux/amd64"],
      "profiles": ["release"]
    }
  ],
  "targets": [{"target": "host/linux/amd64", "profiles": ["release"]}]
}`
}

func TestAuthorComputesTheArtifactDigest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "bin", "builder"), "builder-bytes")
	declarationPath := filepath.Join(root, "declaration.json")
	writeFile(t, declarationPath, declarationFor("bin/builder"))
	outputPath := filepath.Join(root, "catalog.json")

	result, err := author(declarationPath, root, outputPath)
	if err != nil {
		t.Fatalf("author: %v", err)
	}

	catalogBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := buildintent.ParseBuildCatalog(catalogBytes)
	if err != nil {
		t.Fatalf("the written catalog does not parse: %v", err)
	}
	if len(catalog.Providers) != 1 {
		t.Fatalf("providers = %d", len(catalog.Providers))
	}
	provider := catalog.Providers[0]
	// The digest is the one over the file, not anything the declaration said.
	want := digestOfTestBytes([]byte("builder-bytes"))
	if provider.ArtifactDigest != want {
		t.Fatalf("artifactDigest = %s, want %s", provider.ArtifactDigest, want)
	}
	if provider.AuthorityScope != buildintent.BuilderAuthorityScope {
		t.Fatalf("authorityScope = %q", provider.AuthorityScope)
	}
	if result.CatalogDigest != digestOfTestBytes(catalogBytes) {
		t.Fatal("the reported catalog digest is not over the bytes written")
	}
	if result.Providers != 1 || result.Targets != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestAuthorRefusesADeclaredDigest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "builder"), "bytes")
	declarationPath := filepath.Join(root, "declaration.json")
	declaration := strings.Replace(declarationFor("builder"), `"artifact": "builder",`,
		`"artifact": "builder", "artifactDigest": "sha256:`+strings.Repeat("a", 64)+`",`, 1)
	writeFile(t, declarationPath, declaration)

	_, err := author(declarationPath, root, filepath.Join(root, "catalog.json"))
	if err == nil {
		t.Fatal("a declared digest was accepted")
	}
	if !strings.Contains(err.Error(), "must not be declared") {
		t.Fatalf("the error does not name the reason: %v", err)
	}
}

func TestAuthorRefusesAPathOutsideTheRoot(t *testing.T) {
	root := t.TempDir()
	declarationPath := filepath.Join(root, "declaration.json")
	writeFile(t, declarationPath, declarationFor("../outside"))

	_, err := author(declarationPath, root, filepath.Join(root, "catalog.json"))
	if err == nil {
		t.Fatal("a path outside the root was accepted")
	}
	if !strings.Contains(err.Error(), "leaves -root") {
		t.Fatalf("the error does not name the reason: %v", err)
	}
}

func TestAuthorRefusesAMissingArtifact(t *testing.T) {
	root := t.TempDir()
	declarationPath := filepath.Join(root, "declaration.json")
	writeFile(t, declarationPath, declarationFor("not-there"))

	_, err := author(declarationPath, root, filepath.Join(root, "catalog.json"))
	if err == nil {
		t.Fatal("a missing artifact was accepted")
	}
	// The wrapped os.Stat error is the portable spelling of "no such file":
	// the text differs by platform ("no such file or directory" on Unix, "the
	// system cannot find the file specified" on Windows), so test the error
	// value rather than the message (#3783).
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the error does not name the reason: %v", err)
	}
	if !strings.Contains(err.Error(), "not-there") {
		t.Fatalf("the error does not name the artifact: %v", err)
	}
}

func TestAuthorRefusesToOverwrite(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "builder"), "bytes")
	declarationPath := filepath.Join(root, "declaration.json")
	writeFile(t, declarationPath, declarationFor("builder"))
	outputPath := filepath.Join(root, "catalog.json")
	writeFile(t, outputPath, "existing")

	_, err := author(declarationPath, root, outputPath)
	if err == nil {
		t.Fatal("an existing output was overwritten")
	}
	if !strings.Contains(err.Error(), "does not overwrite") {
		t.Fatalf("the error does not name the reason: %v", err)
	}
}

func TestRunPrintsTheDigestTheCallerNeeds(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "builder"), "bytes")
	declarationPath := filepath.Join(root, "declaration.json")
	writeFile(t, declarationPath, declarationFor("builder"))
	outputPath := filepath.Join(root, "catalog.json")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"-declaration", declarationPath,
		"-root", root,
		"-output", outputPath,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d: %s", code, stderr.String())
	}
	var result Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v\n%s", err, stdout.String())
	}
	if !strings.HasPrefix(result.CatalogDigest, "sha256:") {
		t.Fatalf("catalogDigest = %q", result.CatalogDigest)
	}
	if result.CatalogPath != outputPath {
		t.Fatalf("catalogPath = %q", result.CatalogPath)
	}
}

func digestOfTestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
