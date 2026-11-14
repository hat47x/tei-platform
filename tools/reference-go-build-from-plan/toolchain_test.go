package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeGoRoot lays out the four places the closure digest covers. Nothing here is
// executed: the test is about what gets hashed.
func fakeGoRoot(t *testing.T, version string) string {
	t.Helper()
	root := t.TempDir()
	name := "go"
	if runtime.GOOS == "windows" {
		name = "go.exe"
	}
	for path, body := range map[string]string{
		"VERSION":                  version + "\n",
		filepath.Join("bin", name): "go command bytes of " + version,
		filepath.Join("pkg", "tool", runtime.GOOS+"_"+runtime.GOARCH, "compile"): "compiler bytes of " + version,
		filepath.Join("src", "runtime", "x.go"):                                  "package runtime\n",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(root, "bin", name)
}

func TestObserveToolchainTellsTwoToolchainsApart(t *testing.T) {
	first := observeToolchain(fakeGoRoot(t, "go1.0.1"))
	second := observeToolchain(fakeGoRoot(t, "go1.0.2"))
	for _, record := range []*toolchainReport{first, second} {
		if record.NotObserved != "" || record.GoExecutableDigest == "" || record.ClosureDigest == "" || record.GoRoot == "" {
			t.Fatalf("a complete toolchain was not fully observed: %+v", record)
		}
		if !strings.Contains(record.Boundary, "the plan does not name a toolchain") {
			t.Fatalf("the boundary does not say the toolchain is outside the plan: %s", record.Boundary)
		}
	}
	if first.GoExecutableDigest == second.GoExecutableDigest || first.ClosureDigest == second.ClosureDigest {
		t.Fatalf("two different toolchains were reported with the same digest: %+v / %+v", first, second)
	}
}

// The compiler is not the go command. A toolchain whose compiler changed while
// bin/go stayed the same must not look unchanged.
func TestObserveToolchainSeesAChangedCompilerBehindTheSameCommand(t *testing.T) {
	goExe := fakeGoRoot(t, "go1.0.1")
	before := observeToolchain(goExe)
	compiler := filepath.Join(filepath.Dir(filepath.Dir(goExe)), "pkg", "tool", runtime.GOOS+"_"+runtime.GOARCH, "compile")
	if err := os.WriteFile(compiler, []byte("a different compiler"), 0o755); err != nil {
		t.Fatal(err)
	}
	after := observeToolchain(goExe)
	if before.GoExecutableDigest != after.GoExecutableDigest {
		t.Fatal("the go command was not touched, but its digest changed")
	}
	if before.ClosureDigest == after.ClosureDigest {
		t.Fatal("the compiler changed and the closure digest did not")
	}
}

func TestObserveToolchainSaysWhatItCouldNotRead(t *testing.T) {
	dir := t.TempDir()
	lone := filepath.Join(dir, "go")
	if err := os.WriteFile(lone, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	record := observeToolchain(lone)
	if record.GoExecutableDigest == "" || record.ClosureDigest != "" || !strings.Contains(record.NotObserved, "only the executable was hashed") {
		t.Fatalf("an executable outside a Go distribution was not reported as partly observed: %+v", record)
	}

	missing := observeToolchain(filepath.Join(dir, "absent"))
	if missing.GoExecutableDigest != "" || missing.NotObserved == "" {
		t.Fatalf("a missing executable was not reported as not observed: %+v", missing)
	}
}
