package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadExternalRegularFileStableAcceptsExternalRegularFileAndSymlink(t *testing.T) {
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
	content := []byte("postgres://user:secret@example.invalid/evidence\n")
	if err := os.WriteFile(secret, content, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readExternalRegularFileStable(secret, repositoryRoot)
	if err != nil {
		t.Fatalf("regular external fileを拒否しました: %v", err)
	}
	if !reflect.DeepEqual(got, content) {
		t.Fatalf("content=%q want=%q", got, content)
	}

	link := filepath.Join(outside, "dsn-link")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlink test unavailable: %v", err)
	}
	gotLink, err := readExternalRegularFileStable(link, repositoryRoot)
	if err != nil {
		t.Fatalf("Repository外fileへ向くsymlinkを拒否しました: %v", err)
	}
	if !reflect.DeepEqual(gotLink, content) {
		t.Fatalf("symlink content=%q want=%q", gotLink, content)
	}
}

func TestReadExternalRegularFileStableRejectsSymlinkIntoRepository(t *testing.T) {
	base := t.TempDir()
	repositoryRoot := filepath.Join(base, "repo")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(repositoryRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}

	inside := filepath.Join(repositoryRoot, "dsn")
	if err := os.WriteFile(inside, []byte("postgres://inside"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(outside, "link-to-inside")
	if err := os.Symlink(inside, link); err != nil {
		t.Skipf("symlink test unavailable: %v", err)
	}
	if _, err := readExternalRegularFileStable(link, repositoryRoot); err == nil {
		t.Fatal("Repository内fileへ向くsymlinkを受理しています")
	}
}

func TestValidateOpenedExternalFileRejectsPathReplacement(t *testing.T) {
	base := t.TempDir()
	repositoryRoot := filepath.Join(base, "repo")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(repositoryRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}

	first := filepath.Join(outside, "first")
	second := filepath.Join(outside, "second")
	if err := os.WriteFile(first, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(outside, "current")
	if err := os.Symlink(first, link); err != nil {
		t.Skipf("symlink test unavailable: %v", err)
	}

	file, err := os.Open(link)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := validateOpenedExternalFile(link, repositoryRoot, openedInfo); err != nil {
		t.Fatalf("置換前pathを拒否しました: %v", err)
	}

	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(second, link); err != nil {
		t.Fatal(err)
	}
	if err := validateOpenedExternalFile(link, repositoryRoot, openedInfo); err == nil {
		t.Fatal("open後に別fileへ差し替わったpathを受理しています")
	}
}
