package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizedEnvironmentIsInheritedAndDisablesGoEnvFile(t *testing.T) {
	dir := t.TempDir()
	goEnvFile := filepath.Join(dir, "goenv")
	if err := os.WriteFile(goEnvFile, []byte("GOFLAGS=-tags=should_not_leak\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOENV", goEnvFile)
	t.Setenv("GOWORK", "")
	t.Setenv("GOFLAGS", "")

	if _, err := inspectGoEnvironmentWith(fakeGoEnvironmentOutput("", ""), os.Setenv); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		key  string
		want string
	}{
		{key: "GOWORK", want: "off"},
		{key: "GOFLAGS", want: ""},
		{key: "GOENV", want: "off"},
	} {
		if got := os.Getenv(tc.key); got != tc.want {
			t.Fatalf("process %s=%q want=%q", tc.key, got, tc.want)
		}
	}

	out, err := exec.Command("go", "env", "GOFLAGS").CombinedOutput()
	if err != nil {
		t.Fatalf("go env GOFLAGS: %v: %s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "" {
		t.Fatalf("child effective GOFLAGS=%q want empty", got)
	}

	out, err = exec.Command("go", "env", "GOWORK").CombinedOutput()
	if err != nil {
		t.Fatalf("go env GOWORK: %v: %s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "off" {
		t.Fatalf("child effective GOWORK=%q want off", got)
	}
}
