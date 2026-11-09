package gobuilder

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEnvironmentFixedPolicyAndStableOrder(t *testing.T) {
	expected := []string{
		"CGO_ENABLED=0",
		"GOARCH=amd64",
		"GOENV=off",
		"GOEXPERIMENT=",
		"GOFLAGS=",
		"GOOS=linux",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOTOOLCHAIN=local",
		"GOVCS=*:off",
		"GOWORK=off",
	}
	if got := FixedSettings("linux", "amd64"); !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected common Go build settings: got %#v want %#v", got, expected)
	}
	env, err := Environment([]string{"PATH=/ambient/bin", "GOFLAGS=-mod=mod", "AWS_SECRET_ACCESS_KEY=secret"}, "", "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(env, expected) {
		t.Fatalf("unexpected unisolated-state Go environment: got %#v want %#v", env, expected)
	}
}

func TestEnvironmentIsolatesTwoAttemptsAndIgnoresAmbientInputs(t *testing.T) {
	root := t.TempDir()
	ambient := []string{
		"PATH=/ambient/bin",
		"HOME=/ambient/home",
		"LANG=ja_JP.UTF-8",
		"HTTPS_PROXY=https://proxy.example.invalid",
		"AWS_SECRET_ACCESS_KEY=secret",
		"GOFLAGS=-mod=mod",
	}
	first, err := Environment(ambient, filepath.Join(root, "first"), "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Environment(ambient, filepath.Join(root, "second"), "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range first {
		if strings.HasPrefix(item, "PATH=") || strings.HasPrefix(item, "LANG=") ||
			strings.HasPrefix(item, "HTTPS_PROXY=") || strings.HasPrefix(item, "AWS_SECRET_ACCESS_KEY=") {
			t.Fatalf("ambient Host input leaked into Go environment: %s", item)
		}
	}
	a, b := mapEnv(t, first), mapEnv(t, second)
	if len(a) != len(b) {
		t.Fatalf("inconsistent build settings: first %d second %d", len(a), len(b))
	}
	for _, key := range []string{"GOCACHE", "GOMODCACHE", "GOPATH", "HOME", "USERPROFILE", "TMPDIR", "TMP", "TEMP"} {
		if a[key] == "" || b[key] == "" || a[key] == b[key] {
			t.Fatalf("%s not isolated across build attempts: %q / %q", key, a[key], b[key])
		}
	}
	for _, state := range []string{"first", "second"} {
		for _, dir := range []string{"home", "tmp"} {
			info, err := os.Stat(filepath.Join(root, state, dir))
			if err != nil || !info.IsDir() {
				t.Fatalf("missing isolated %s/%s: %v", state, dir, err)
			}
		}
	}
}

func TestEnvironmentWindowsExplicitHostAllowlist(t *testing.T) {
	base := []string{
		"Path=C:\\ambient",
		"SystemRoot=C:\\Windows",
		"COMSPEC=C:\\Windows\\System32\\cmd.exe",
		"LANG=ja_JP.UTF-8",
	}
	host, err := AllowedHostEnvironment(base, "windows")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"COMSPEC=C:\\Windows\\System32\\cmd.exe", "SYSTEMROOT=C:\\Windows"}
	if !reflect.DeepEqual(host, want) {
		t.Fatalf("Windows allowlist mismatch: got %#v want %#v", host, want)
	}
	env, err := Environment(base, "", "windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range want {
		found := false
		for _, actual := range env {
			if actual == entry {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("allowed Windows input missing: %s", entry)
		}
	}
	if _, err := AllowedHostEnvironment([]string{"SystemRoot=C:\\Windows", "SYSTEMROOT=D:\\Windows"}, "windows"); err == nil {
		t.Fatal("conflicting case-insensitive Host variable must be rejected")
	}
	if unix, err := AllowedHostEnvironment(base, "linux"); err != nil || len(unix) != 0 {
		t.Fatalf("Unix inherited Host input: %#v, %v", unix, err)
	}
}

func TestEnvironmentRejectsUnusableStateDir(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "already-file")
	if err := os.WriteFile(file, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Environment(nil, file, "linux", "amd64"); err == nil {
		t.Fatal("state directory cannot be a file")
	}
}

func mapEnv(t *testing.T, entries []string) map[string]string {
	t.Helper()
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			t.Fatalf("malformed environment variable %q", entry)
		}
		if _, exists := result[key]; exists {
			t.Fatalf("duplicate environment variable %s", key)
		}
		result[key] = value
	}
	return result
}
