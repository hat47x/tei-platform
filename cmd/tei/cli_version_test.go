package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

func TestBuildIdentityFromStampedBuildInfo(t *testing.T) {
	info := &debug.BuildInfo{
		GoVersion: "go1.26.9",
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"},
			{Key: "vcs.modified", Value: "true"},
			{Key: "-ldflags", Value: "-s -w"},
			{Key: "GOOS", Value: "linux"},
		},
	}
	identity := buildIdentity(info, true)
	if identity.SchemaVersion != "tei.cli.version/v0" || identity.Effect != "read-only" ||
		identity.Source != "build-info" ||
		identity.VCSRevision != "0123456789abcdef0123456789abcdef01234567" ||
		!identity.VCSModified || identity.GoVersion != "go1.26.9" {
		t.Fatalf("stamped identity lost: %+v", identity)
	}
	if len(identity.BuildSettings) != 1 || identity.BuildSettings[0] != "-ldflags=-s -w" {
		t.Fatalf("only build-affecting settings are kept: %+v", identity.BuildSettings)
	}
}

func TestBuildIdentityWithoutBuildInfoIsExplicitlyUnavailable(t *testing.T) {
	identity := buildIdentity(nil, false)
	if identity.Source != "unavailable" || identity.VCSRevision != "" || identity.VCSModified ||
		identity.GoVersion != runtime.Version() || identity.GOOS != runtime.GOOS || identity.GOARCH != runtime.GOARCH {
		t.Fatalf("missing build info must be reported, not guessed: %+v", identity)
	}
}

func TestVersionCommandRejectsUnknownArguments(t *testing.T) {
	var out bytes.Buffer
	code, err := runVersionCommand([]string{"--verbose"}, &out)
	if err != nil || code != 2 || !strings.Contains(out.String(), "--verbose") {
		t.Fatalf("unknown argument must fail with exit 2: code=%d err=%v out=%q", code, err, out.String())
	}
}

func TestVersionCommandTextAndJSON(t *testing.T) {
	var text bytes.Buffer
	if code, err := runVersionCommand(nil, &text); err != nil || code != 0 {
		t.Fatalf("text version: code=%d err=%v out=%q", code, err, text.String())
	}
	if !strings.HasPrefix(text.String(), "tei tei.cli.version/v0 "+runtime.GOOS+"/"+runtime.GOARCH+" go=") ||
		!strings.Contains(text.String(), "revision=") {
		t.Fatalf("unexpected text identity: %q", text.String())
	}

	var js bytes.Buffer
	if code, err := runVersionCommand([]string{"-json"}, &js); err != nil || code != 0 {
		t.Fatalf("json version: code=%d err=%v out=%q", code, err, js.String())
	}
	var identity cliBuildIdentity
	if err := json.Unmarshal(js.Bytes(), &identity); err != nil {
		t.Fatalf("version -json is not one JSON document: %v: %q", err, js.String())
	}
	if identity.SchemaVersion != "tei.cli.version/v0" || identity.GOOS != runtime.GOOS ||
		identity.GOARCH != runtime.GOARCH || identity.GoVersion != runtime.Version() ||
		(identity.Source != "build-info" && identity.Source != "unavailable") {
		t.Fatalf("unexpected JSON identity: %+v", identity)
	}
}

// The identity must describe the executable, so it is checked through a real
// process build rather than the test binary.
func TestVersionCommandInBuiltProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	run := func(args ...string) (string, int) {
		command := exec.Command(binary, args...)
		command.Env = append(os.Environ(), "GIT_DIR=")
		var stdout bytes.Buffer
		command.Stdout = &stdout
		err := command.Run()
		if exit, ok := err.(*exec.ExitError); ok {
			return stdout.String(), exit.ExitCode()
		} else if err != nil {
			t.Fatalf("run %v: %v", args, err)
		}
		return stdout.String(), 0
	}
	text, code := run("version")
	if code != 0 || !strings.Contains(text, "tei.cli.version/v0") {
		t.Fatalf("version: exit=%d out=%q", code, text)
	}
	js, code := run("version", "-json")
	if code != 0 {
		t.Fatalf("version -json: exit=%d out=%q", code, js)
	}
	var identity cliBuildIdentity
	if err := json.Unmarshal([]byte(js), &identity); err != nil {
		t.Fatalf("version -json: %v: %q", err, js)
	}
	if identity.GoVersion != runtime.Version() || identity.GOOS != runtime.GOOS {
		t.Fatalf("process identity disagrees with the toolchain that built it: %+v", identity)
	}
	if _, code := run("version", "-workspace", "."); code != 2 {
		t.Fatalf("version must reject unrelated flags: exit=%d", code)
	}
}
