package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPostgresDSNFromEnvironmentOutsideRepositoryAfterSourcePreflightAcceptsExternalFile(t *testing.T) {
	base := t.TempDir()
	repositoryRoot := filepath.Join(base, "repo")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(repositoryRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "dsn")
	if err := os.WriteFile(secret, []byte("postgres://user:secret@example.invalid/evidence\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv(postgresDSNFileEnv, secret)
	unsetEnvForTest(t, postgresDSNEnv)
	got, err := loadPostgresDSNFromEnvironmentOutsideRepositoryAfterSourcePreflight(repositoryRoot)
	if err != nil {
		t.Fatalf("loadPostgresDSNFromEnvironmentOutsideRepositoryAfterSourcePreflight() error=%v", err)
	}
	if got != "postgres://user:secret@example.invalid/evidence" {
		t.Fatalf("DSN=%q", got)
	}
}

func TestLoadPostgresDSNFromEnvironmentOutsideRepositoryAfterSourcePreflightRejectsInsideFileWithoutLeakingPath(t *testing.T) {
	base := t.TempDir()
	repositoryRoot := filepath.Join(base, "repo")
	if err := os.Mkdir(repositoryRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(repositoryRoot, "very-secret-dsn")
	if err := os.WriteFile(secret, []byte("postgres://user:secret@example.invalid/evidence"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv(postgresDSNFileEnv, secret)
	unsetEnvForTest(t, postgresDSNEnv)
	_, err := loadPostgresDSNFromEnvironmentOutsideRepositoryAfterSourcePreflight(repositoryRoot)
	if err == nil {
		t.Fatal("Repository内secret fileを受理しています")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "very-secret-dsn") {
		t.Fatalf("errorへlocal secret pathを漏らしています: %v", err)
	}
}

func unsetEnvForTest(t *testing.T, name string) {
	t.Helper()
	old, existed := os.LookupEnv(name)
	if err := os.Unsetenv(name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(name, old)
			return
		}
		_ = os.Unsetenv(name)
	})
}
