package main

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadPostgresDSNSecretRejectsDirectLaunchValue(t *testing.T) {
	const directDSN = "postgres://user:secret@example.invalid/evidence"
	readCalled := false
	_, err := loadPostgresDSNSecret(
		func(key string) (string, bool) {
			switch key {
			case postgresDSNEnv:
				return directDSN, true
			case postgresDSNFileEnv:
				return "/private/dsn", true
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
		t.Fatal("direct DSN launchを受理しました")
	}
	if readCalled {
		t.Fatal("direct DSN launchをrejectする前にsecret fileを読みました")
	}
	if strings.Contains(err.Error(), directDSN) || strings.Contains(err.Error(), "user:secret@") {
		t.Fatalf("DSN本文をerrorへ露出しています: %q", err)
	}
}

func TestLoadPostgresDSNSecretRequiresFileWithoutPersistingPath(t *testing.T) {
	const secretPath = "/private/build/postgres-dsn"
	_, err := loadPostgresDSNSecret(
		func(key string) (string, bool) {
			if key == postgresDSNFileEnv {
				return secretPath, true
			}
			return "", false
		},
		func(string) ([]byte, error) { return nil, errors.New("permission denied") },
	)
	if err == nil {
		t.Fatal("secret file read failureを受理しました")
	}
	if strings.Contains(err.Error(), secretPath) {
		t.Fatalf("secret file pathをerrorへ露出しています: %q", err)
	}
}

func TestNormalizePostgresDSNSecretAcceptsOneLineAndSingleFinalNewline(t *testing.T) {
	const dsn = "postgres://user:password@example.invalid/evidence?sslmode=require"
	for _, content := range [][]byte{
		[]byte(dsn),
		[]byte(dsn + "\n"),
		[]byte(dsn + "\r\n"),
	} {
		got, err := normalizePostgresDSNSecret(content)
		if err != nil {
			t.Fatalf("normalizePostgresDSNSecret() error=%v", err)
		}
		if got != dsn {
			t.Fatalf("DSN=%q want=%q", got, dsn)
		}
	}
}

func TestNormalizePostgresDSNSecretRejectsEmptyMultilineAndNUL(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content []byte
	}{
		{name: "empty", content: nil},
		{name: "newline-only", content: []byte("\n")},
		{name: "crlf-only", content: []byte("\r\n")},
		{name: "multiline", content: []byte("postgres://first\npostgres://second")},
		{name: "extra-newline", content: []byte("postgres://first\n\n")},
		{name: "bare-cr", content: []byte("postgres://first\r")},
		{name: "nul", content: []byte("postgres://first\x00tail")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := normalizePostgresDSNSecret(tc.content); err == nil {
				t.Fatal("不正なsecret file内容を受理しました")
			}
	}
}
