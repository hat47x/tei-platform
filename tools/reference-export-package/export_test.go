package exportpackage

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func newRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("alpha"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.txt"), []byte("beta"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestExportAndVerify(t *testing.T) {
	root := newRoot(t)
	out := filepath.Join(t.TempDir(), "package.zip")
	manifest, err := Export(root, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Files) != 2 || manifest.Digest == "" {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	verified, err := Verify(out)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Digest != manifest.Digest || len(verified.Files) != 2 {
		t.Fatalf("verified manifest differs: %+v", verified)
	}
}

func TestTamperedPackageIsDetected(t *testing.T) {
	root := newRoot(t)
	out := filepath.Join(t.TempDir(), "package.zip")
	if _, err := Export(root, out); err != nil {
		t.Fatal(err)
	}
	tampered := filepath.Join(t.TempDir(), "tampered.zip")
	rewriteZip(t, out, tampered, "files/a.txt", []byte("changed"))
	if _, err := Verify(tampered); err == nil {
		t.Fatal("tampered package was verified")
	}
}

func rewriteZip(t *testing.T, source, destination, name string, content []byte) {
	t.Helper()
	reader, err := zip.OpenReader(source)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	file, err := os.Create(destination)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range reader.File {
		var data []byte
		if entry.Name == name {
			data = content
		} else {
			source, err := entry.Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err = io.ReadAll(source)
			source.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
		target, err := writer.Create(entry.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := target.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSymlinkInRootIsRejected(t *testing.T) {
	root := newRoot(t)
	if err := os.Symlink(filepath.Join(root, "a.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Export(root, filepath.Join(t.TempDir(), "package.zip")); err == nil {
		t.Fatal("package with a symlink was exported")
	}
}

func TestOutputInsideRootIsRejected(t *testing.T) {
	root := newRoot(t)
	if _, err := Export(root, filepath.Join(root, "package.zip")); err == nil {
		t.Fatal("output inside the package root was accepted")
	}
}
