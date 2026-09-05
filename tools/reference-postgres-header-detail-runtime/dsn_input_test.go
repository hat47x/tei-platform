package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestLoadPostgresDSNInputAcceptsDirectValueForStandaloneCompatibility(t *testing.T) {
	const direct = "postgres://user:password@example.invalid/evidence"
	got, err := loadPostgresDSNInput(
		func(key string) (string, bool) {
			if key == postgresDSNEnv {
				return direct, true
			}
			return "", false
		},
		func(string) ([]byte, error) {
			t.Fatal("direct DSN入力でsecret fileを読みました")
			return nil, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != direct {
		t.Fatalf("DSN=%q want=%q", got, direct)
	}
}

func TestLoadPostgresDSNInputAcceptsSecretFile(t *testing.T) {
	const path = "/private/postgres-dsn"
	const direct = "postgres://user:password@example.invalid/evidence"
	got, err := loadPostgresDSNInput(
		func(key string) (string, bool) {
			if key == postgresDSNFileEnv {
				return path, true
			}
			return "", false
		},
		func(gotPath string) ([]byte, error) {
			if gotPath != path {
				t.Fatalf("path=%q want=%q", gotPath, path)
			}
			return []byte(direct + "\n"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != direct {
		t.Fatalf("DSN=%q want=%q", got, direct)
	}
}

func TestLoadPostgresDSNInputRejectsAmbiguousDirectAndFileInputs(t *testing.T) {
	readCalled := false
	_, err := loadPostgresDSNInput(
		func(key string) (string, bool) {
			switch key {
			case postgresDSNEnv:
				return "postgres://direct-secret", true
			case postgresDSNFileEnv:
				return "/private/postgres-dsn", true
			default:
				return "", false
			}
		},
		func(string) ([]byte, error) {
			readCalled = true
			return nil, nil
		},
	)
	if err == nil {
		t.Fatal("direct DSNとfile pathの同時指定を受理しました")
	}
	if readCalled {
		t.Fatal("ambiguous入力をrejectする前にsecret fileを読みました")
	}
	if strings.Contains(err.Error(), "postgres://direct-secret") || strings.Contains(err.Error(), "/private/postgres-dsn") {
		t.Fatalf("secret値またはpathをerrorへ露出しています: %q", err)
	}
}

func TestLoadPostgresDSNInputDoesNotExposeFilePathOnReadFailure(t *testing.T) {
	const path = "/private/postgres-dsn"
	_, err := loadPostgresDSNInput(
		func(key string) (string, bool) {
			if key == postgresDSNFileEnv {
				return path, true
			}
			return "", false
		},
		func(string) ([]byte, error) { return nil, errors.New("permission denied: " + path) },
	)
	if err == nil {
		t.Fatal("secret file read failureを受理しました")
	}
	if strings.Contains(err.Error(), path) {
		t.Fatalf("secret file pathをerrorへ露出しています: %q", err)
	}
}

func TestNormalizePostgresDSNFileRejectsUnsafeContent(t *testing.T) {
	const dsn = "postgres://user:password@example.invalid/evidence"
	for _, content := range [][]byte{[]byte(dsn), []byte(dsn + "\n"), []byte(dsn + "\r\n")} {
		got, err := normalizePostgresDSNFile(content)
		if err != nil {
			t.Fatal(err)
		}
		if got != dsn {
			t.Fatalf("DSN=%q want=%q", got, dsn)
		}
	}

	for _, content := range [][]byte{
		nil,
		[]byte("\n"),
		[]byte("postgres://first\npostgres://second"),
		[]byte("postgres://first\r"),
		[]byte("postgres://first\x00tail"),
	} {
		if _, err := normalizePostgresDSNFile(content); err == nil {
			t.Fatalf("不正なsecret file内容を受理しました: %q", content)
		}
	}
}

func TestClearPostgresDSNEnvironmentRemovesBothInputs(t *testing.T) {
	t.Setenv(postgresDSNEnv, "postgres://direct-secret")
	t.Setenv(postgresDSNFileEnv, "/private/postgres-dsn")

	if err := clearPostgresDSNEnvironment(); err != nil {
		t.Fatal(err)
	}
	if _, exists := os.LookupEnv(postgresDSNEnv); exists {
		t.Fatal("direct DSN environmentが残っています")
	}
	if _, exists := os.LookupEnv(postgresDSNFileEnv); exists {
		t.Fatal("DSN file path environmentが残っています")
	}
}
