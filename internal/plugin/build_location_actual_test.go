package plugin_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/plugin"
)

// This exercises the installed Go compiler on a synthetic module, not a TEI
// builder adapter or a proof that other compilers are location independent.
func TestActualGoBuildLocationAndContent(t *testing.T) {
	root := t.TempDir()
	const original = "package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"original\") }\n"
	const changed = "package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"changed\") }\n"
	write := func(dir, source string) {
		t.Helper()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, content := range map[string]string{"go.mod": "module example.invalid/relocation\n\ngo 1.24\n", "main.go": source} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	a, b := filepath.Join(root, "first"), filepath.Join(root, "second")
	write(a, original)
	write(b, original)
	build := func(dir, output string, trim bool) []byte {
		t.Helper()
		args := []string{"build", "-buildvcs=false"}
		if trim {
			args = append(args, "-trimpath")
		}
		args = append(args, "-o", output, ".")
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0", "GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build: %v\n%s", err, out)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	out := func(name string) string { return filepath.Join(root, "output-"+name+ext) }
	first := build(a, out("first"), true)
	repeat := build(a, out("repeat"), true)
	moved := build(b, out("moved"), true)
	if !bytes.Equal(first, repeat) || !bytes.Equal(first, moved) {
		t.Fatal("trimpath builds did not reproduce identical bytes")
	}
	plainA := build(a, out("plain-a"), false)
	plainB := build(b, out("plain-b"), false)
	write(b, changed)
	modified := build(b, out("modified"), true)
	if bytes.Equal(first, modified) {
		t.Fatal("modified source produced identical executable bytes")
	}
	for _, tc := range []struct{ path, want string }{{out("first"), "original\n"}, {out("moved"), "original\n"}, {out("modified"), "changed\n"}} {
		got, err := exec.Command(tc.path).CombinedOutput()
		if err != nil || string(got) != tc.want {
			t.Fatalf("run %s: output=%q err=%v", tc.path, got, err)
		}
	}
	planA := buildPlanFixture()
	planB := buildPlanFixture()
	planA.Sources[0].Digest = plugin.BuildOutputDigest([]byte(original))
	planB.Sources[0].Digest = planA.Sources[0].Digest
	planA.Sources[0].Locator = "first/main.go"
	planB.Sources[0].Locator = "second/main.go"
	fpA, err := plugin.BuildPlanFingerprint(planA)
	if err != nil {
		t.Fatal(err)
	}
	fpB, err := plugin.BuildPlanFingerprint(planB)
	if err != nil {
		t.Fatal(err)
	}
	if fpA == fpB {
		t.Fatal("BuildPlan v1 must retain locator-sensitive fingerprints")
	}
	t.Logf("Go compiler: trimpath same bytes on repeat and move; changed content differs; without trimpath location changes output=%t; BuildPlan v1 locators differ", !bytes.Equal(plainA, plainB))
}
