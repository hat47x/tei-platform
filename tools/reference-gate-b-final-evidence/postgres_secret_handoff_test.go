package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPostgresDSNFilePathFromEntriesRejectsDirectDSNCaseInsensitively(t *testing.T) {
	const directDSN = "postgres://user:secret@example.invalid/evidence"
	_, err := loadPostgresDSNFilePathFromEntries([]string{
		"PATH=/usr/bin",
		"tei_test_postgres_dsn=" + directDSN,
		"TEI_TEST_POSTGRES_DSN_FILE=/private/dsn",
	})
	if err == nil {
		t.Fatal("大小文字違いのdirect DSN launchを受理しました")
	}
	if strings.Contains(err.Error(), directDSN) || strings.Contains(err.Error(), "user:secret@") {
		t.Fatalf("DSN本文をerrorへ露出しています: %q", err)
	}
}

func TestLoadPostgresDSNFilePathFromEntriesRejectsCaseInsensitiveDuplicateFileKeys(t *testing.T) {
	_, err := loadPostgresDSNFilePathFromEntries([]string{
		"TEI_TEST_POSTGRES_DSN_FILE=/private/first",
		"tei_test_postgres_dsn_file=/private/second",
	})
	if err == nil {
		t.Fatal("大小文字違いのduplicate DSN file keyを受理しました")
	}
}

func TestLoadPostgresDSNFilePathFromEntriesAcceptsSingleFilePath(t *testing.T) {
	const path = "/private/postgres-dsn"
	got, err := loadPostgresDSNFilePathFromEntries([]string{
		"PATH=/usr/bin",
		"TEI_TEST_POSTGRES_DSN_FILE=" + path,
		"OTHER=value",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("path=%q want=%q", got, path)
	}
}

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
