package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReferenceGoBuildEnvironmentRejectsAmbientHostInputs(t *testing.T) {
	firstDir := filepath.Join(t.TempDir(), "first")
	secondDir := filepath.Join(t.TempDir(), "second")
	host := []string{
		"PATH=/ambient/bin",
		"HOME=/ambient/home",
		"LANG=ja_JP.UTF-8",
		"HTTPS_PROXY=https://proxy.example.invalid",
		"AWS_SECRET_ACCESS_KEY=sample-secret",
		"GOFLAGS=-mod=mod",
		"GOPROXY=https://proxy.example.invalid",
	}
	first, err := buildEnvironment(host, firstDir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildEnvironment(host, secondDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != len(second) {
		t.Fatalf("build environment changed its keys between isolated runs: %d != %d", len(first), len(second))
	}
	asMap := func(entries []string) map[string]string {
		t.Helper()
		out := make(map[string]string)
		for _, entry := range entries {
			key, value, ok := strings.Cut(entry, "=")
			if !ok {
				t.Fatalf("malformed build environment entry: %q", entry)
			}
			if _, duplicate := out[key]; duplicate {
				t.Fatalf("duplicate build environment key: %q", key)
			}
			out[key] = value
		}
		return out
	}
	a, b := asMap(first), asMap(second)
	for _, key := range []string{"PATH", "LANG", "HTTPS_PROXY", "AWS_SECRET_ACCESS_KEY"} {
		if _, leaked := a[key]; leaked {
			t.Fatalf("ambient Host input %q leaked into build", key)
		}
	}
	for key, want := range map[string]string{
		"CGO_ENABLED": "0", "GOENV": "off", "GOEXPERIMENT": "",
		"GOFLAGS": "", "GOPROXY": "off", "GOSUMDB": "off",
		"GOTOOLCHAIN": "local", "GOVCS": "*:off", "GOWORK": "off",
	} {
		if a[key] != want {
			t.Fatalf("Go setting %q = %q, want %q", key, a[key], want)
		}
	}
	isolated := map[string]string{
		"GOCACHE": "cache", "GOMODCACHE": "modcache", "GOPATH": "gopath",
		"HOME": "home", "USERPROFILE": "home", "TMPDIR": "tmp",
		"TMP": "tmp", "TEMP": "tmp",
	}
	for key, suffix := range isolated {
		if a[key] != filepath.Join(firstDir, suffix) || b[key] != filepath.Join(secondDir, suffix) {
			t.Fatalf("%s not isolated across builds: first=%q second=%q", key, a[key], b[key])
		}
		if a[key] == b[key] {
			t.Fatalf("%s reused the same directory for independent builds", key)
		}
	}
	for _, dir := range []string{"home", "tmp"} {
		info, err := os.Stat(filepath.Join(firstDir, dir))
		if err != nil || !info.IsDir() {
			t.Fatalf("missing isolated state directory %q: %v", dir, err)
		}
	}
}

func TestReferenceGoBuildHostEnvironmentHasExplicitWindowsAllowlist(t *testing.T) {
	values, err := allowedHostEnvironmentForOS([]string{
		"Path=C:\\ambient",
		"SystemRoot=C:\\Windows",
		"COMSPEC=C:\\Windows\\System32\\cmd.exe",
		"LANG=ja_JP.UTF-8",
		"NOT_A_BUILD_INPUT=secret",
	}, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0] != "COMSPEC=C:\\Windows\\System32\\cmd.exe" || values[1] != "SYSTEMROOT=C:\\Windows" {
		t.Fatalf("unexpected Windows Host allowlist: %#v", values)
	}
	if _, err := allowedHostEnvironmentForOS([]string{
		"SystemRoot=C:\\Windows",
		"SYSTEMROOT=D:\\Windows",
	}, "windows"); err == nil {
		t.Fatal("conflicting case-insensitive Windows Host entries were accepted")
	}
	linux, err := allowedHostEnvironmentForOS([]string{"PATH=/ambient/bin", "HOME=/ambient", "AWS_TOKEN=secret"}, "linux")
	if err != nil || len(linux) != 0 {
		t.Fatalf("Unix build inherited Host environment: %#v, %v", linux, err)
	}
}
