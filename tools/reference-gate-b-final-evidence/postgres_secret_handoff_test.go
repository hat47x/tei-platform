package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreparePostgresDSNHandoffCreatesExternalOwnerOnlyFileAndCleansUp(t *testing.T) {
	base := t.TempDir()
	repositoryRoot := filepath.Join(base, "repo")
	if err := os.Mkdir(repositoryRoot, 0o700); err != nil {
		t.Fatal(err)
	}

	const dsn = "postgres://user:secret@example.invalid/evidence"
	path, cleanup, err := preparePostgresDSNHandoff(dsn, repositoryRoot)
	if err != nil {
		t.Fatalf("preparePostgresDSNHandoff() error=%v", err)
	}
	if cleanup == nil {
		t.Fatal("cleanup=nil")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if pathWithinRepository(repositoryRoot, absPath) {
		cleanup()
		t.Fatalf("handoff pathがRepository内です: %q", absPath)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	if string(content) != dsn {
		cleanup()
		t.Fatalf("handoff content=%q", content)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	if !fileInfo.Mode().IsRegular() {
		cleanup()
		t.Fatalf("handoff file mode=%v", fileInfo.Mode())
	}
	if got := fileInfo.Mode().Perm(); got&0o077 != 0 {
		cleanup()
		t.Fatalf("handoff fileがgroup/otherへ権限を持っています: %o", got)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got&0o077 != 0 {
		cleanup()
		t.Fatalf("handoff directoryがgroup/otherへ権限を持っています: %o", got)
	}

	dir := filepath.Dir(path)
	cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("cleanup後もhandoff directoryが残っています: err=%v", err)
	}
}

func TestPreparePostgresDSNHandoffRejectsEmptySecret(t *testing.T) {
	if _, _, err := preparePostgresDSNHandoff("", t.TempDir()); err == nil {
		t.Fatal("空DSNをhandoffへ受理しました")
	}
}
