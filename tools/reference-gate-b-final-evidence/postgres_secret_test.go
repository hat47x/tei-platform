package main

import (
	"strings"
	"testing"
)

func TestLoadPostgresDSNFilePathRejectsDirectLaunchValue(t *testing.T) {
	const directDSN = "postgres://user:secret@example.invalid/evidence"
	_, err := loadPostgresDSNFilePath(func(key string) (string, bool) {
		switch key {
		case postgresDSNEnv:
			return directDSN, true
		case postgresDSNFileEnv:
			return "/private/dsn", true
		default:
			return "", false
		}
	})
	if err == nil {
		t.Fatal("direct DSN launchを受理しました")
	}
	if strings.Contains(err.Error(), directDSN) || strings.Contains(err.Error(), "user:secret@") {
		t.Fatalf("DSN本文をerrorへ露出しています: %q", err)
	}
}

func TestLoadPostgresDSNFilePathRequiresNonEmptyPath(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		set   bool
	}{
		{name: "unset", set: false},
		{name: "empty", value: "", set: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := loadPostgresDSNFilePath(func(key string) (string, bool) {
				if key == postgresDSNFileEnv {
					return tc.value, tc.set
				}
				return "", false
			}); err == nil {
				t.Fatal("DSN file pathなしの起動を受理しました")
			}
		})
	}
}

func TestLoadPostgresDSNFilePathReturnsPathWithoutReadingSecret(t *testing.T) {
	const path = "/private/build/postgres-dsn"
	got, err := loadPostgresDSNFilePath(func(key string) (string, bool) {
		if key == postgresDSNFileEnv {
			return path, true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("path=%q want=%q", got, path)
	}
}
