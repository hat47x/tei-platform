package exportpackage

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func categoryRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range Categories {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "payload.txt"), []byte(dir+" payload"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func packageEntries(t *testing.T, path string) []string {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var names []string
	for _, entry := range reader.File {
		names = append(names, entry.Name)
	}
	return names
}

func TestExportCarriesEveryCategoryByDefault(t *testing.T) {
	root := categoryRoot(t)
	out := filepath.Join(t.TempDir(), "package.zip")
	manifest, err := ExportWithPolicy(root, out, Policy{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manifest.IncludedCategories, Categories) {
		t.Fatalf("included = %v", manifest.IncludedCategories)
	}
	if len(manifest.ExcludedCategories) != 0 || len(manifest.EmptyCategories) != 0 {
		t.Fatalf("manifest = %+v", manifest)
	}
	if len(manifest.Files) != 4 {
		t.Fatalf("files = %+v", manifest.Files)
	}
	for _, file := range manifest.Files {
		if file.Category == "" {
			t.Errorf("file %q has no category", file.Path)
		}
	}
	verified, err := Verify(out)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(verified.IncludedCategories, Categories) {
		t.Fatalf("verified included = %v", verified.IncludedCategories)
	}
}

func TestExcludedCategoriesAreRecordedAndAbsent(t *testing.T) {
	root := categoryRoot(t)
	out := filepath.Join(t.TempDir(), "package.zip")
	manifest, err := ExportWithPolicy(root, out, Policy{Exclude: []string{CategoryAuthority, CategoryEvidence}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manifest.ExcludedCategories, []string{CategoryAuthority, CategoryEvidence}) {
		t.Fatalf("excluded = %v", manifest.ExcludedCategories)
	}
	if !reflect.DeepEqual(manifest.IncludedCategories, []string{CategoryMeaning, CategoryData}) {
		t.Fatalf("included = %v", manifest.IncludedCategories)
	}
	for _, file := range manifest.Files {
		if file.Category == CategoryAuthority || file.Category == CategoryEvidence {
			t.Fatalf("excluded category leaked into the package: %+v", file)
		}
	}
	for _, name := range packageEntries(t, out) {
		if strings.Contains(name, "authority/") || strings.Contains(name, "evidence/") {
			t.Fatalf("excluded file is in the zip: %s", name)
		}
	}
}

func TestIncludedCategoryWithoutFilesIsExplicit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, CategoryMeaning), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, CategoryMeaning, "m.txt"), []byte("m"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "package.zip")
	manifest, err := ExportWithPolicy(root, out, Policy{Include: []string{CategoryMeaning, CategoryData}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manifest.EmptyCategories, []string{CategoryData}) {
		t.Fatalf("empty = %v", manifest.EmptyCategories)
	}
}

func TestUnclassifiedFileIsRejected(t *testing.T) {
	root := categoryRoot(t)
	if err := os.WriteFile(filepath.Join(root, "stray.txt"), []byte("stray"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "package.zip")
	if _, err := ExportWithPolicy(root, out, Policy{}); err == nil {
		t.Fatal("an unclassified file was exported")
	}
}

func TestUnknownPolicyCategoryIsRejected(t *testing.T) {
	root := categoryRoot(t)
	out := filepath.Join(t.TempDir(), "package.zip")
	if _, err := ExportWithPolicy(root, out, Policy{Include: []string{"secrets"}}); err == nil {
		t.Fatal("an unknown category was accepted")
	}
}

func TestPolicyExportIsDeterministic(t *testing.T) {
	root := categoryRoot(t)
	dir := t.TempDir()
	first, err := ExportWithPolicy(root, filepath.Join(dir, "a.zip"), Policy{Include: []string{CategoryMeaning, CategoryData}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ExportWithPolicy(root, filepath.Join(dir, "b.zip"), Policy{Include: []string{CategoryMeaning, CategoryData}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("not deterministic: %+v != %+v", first, second)
	}
}

func TestPlainExportKeepsWorking(t *testing.T) {
	root := newRoot(t)
	out := filepath.Join(t.TempDir(), "package.zip")
	manifest, err := Export(root, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Files) != 2 {
		t.Fatalf("files = %+v", manifest.Files)
	}
	reader, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	found := false
	for _, entry := range reader.File {
		if entry.Name == "manifest.json" {
			stream, err := entry.Open()
			if err != nil {
				t.Fatal(err)
			}
			raw, err := io.ReadAll(stream)
			stream.Close()
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "includedCategories") {
				t.Fatal("plain Export invented category metadata")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("manifest.json not found")
	}
}
