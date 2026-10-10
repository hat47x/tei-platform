package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// This exercise observes two real static Go entry points and checks that
// source-closure differences correspond to an actual linked symbol. It is a
// synthetic fixture, not a deployable TEI application or an authorization test.
func TestGoBuildStaticEntrySelectionMatchesE0Closure(t *testing.T) {
	for _, binary := range []string{"git", "go"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skipf("%s unavailable: %v", binary, err)
		}
	}
	root := t.TempDir()
	put := func(name, content string) {
		t.Helper()
		dst := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(command string, args ...string) string {
		t.Helper()
		cmd := exec.Command(command, args...)
		cmd.Dir = root
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", command, args, err, output)
		}
		return string(output)
	}
	put("go.mod", "module example.test/tei-closure-fixture\n\ngo 1.23.2\n")
	put("cmd/unified/main.go", "package main\nimport (\n \"fmt\"\n \"example.test/tei-closure-fixture/internal/order\"\n \"example.test/tei-closure-fixture/internal/repair\"\n)\nfunc main() { fmt.Println(order.Run(), repair.Run()) }\n")
	put("cmd/order/main.go", "package main\nimport (\n \"fmt\"\n \"example.test/tei-closure-fixture/internal/order\"\n)\nfunc main() { fmt.Println(order.Run()) }\n")
	put("internal/order/order.go", "package order\n//go:noinline\nfunc Run() string { return \"order\" }\n")
	put("internal/repair/repair.go", "package repair\n//go:noinline\nfunc Run() string { return \"repair\" }\n")
	run("git", "init", "-q")
	run("git", "add", "-A")
	run("git", "-c", "user.name=TEI Reference", "-c", "user.email=tei-reference@example.invalid", "commit", "-qm", "static entry fixture")

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore test cwd: %v", err)
		}
	})
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("CGO_ENABLED", "0")

	unified, err := produceEvidence("./cmd/unified", "")
	if err != nil {
		t.Fatalf("unified E0: %v", err)
	}
	order, err := produceEvidence("./cmd/order", "")
	if err != nil {
		t.Fatalf("order E0: %v", err)
	}
	difference, err := compareReports(unified, order)
	if err != nil {
		t.Fatalf("same-head Go closure comparison: %v", err)
	}
	if unified.SourceRevision != order.SourceRevision ||
		!hasPath(difference.BeforeOnly, "example.test/tei-closure-fixture/internal/repair") ||
		!hasPath(difference.BeforeOnly, "example.test/tei-closure-fixture/cmd/unified") ||
		!hasPath(difference.AfterOnly, "example.test/tei-closure-fixture/cmd/order") ||
		hasPath(difference.AfterOnly, "example.test/tei-closure-fixture/internal/repair") ||
		len(difference.Changed) != 0 {
		t.Fatalf("incorrect static entry source-closure delta: %+v", difference)
	}

	// Keep artifacts outside the clean Git source checkout. No size ratio is
	// asserted: code size depends on Go toolchain, linker, OS, and target.
	outputRoot := t.TempDir()
	unifiedBin := filepath.Join(outputRoot, "unified")
	orderBin := filepath.Join(outputRoot, "order")
	run("go", "build", "-trimpath", "-ldflags=-buildid=", "-o", unifiedBin, "./cmd/unified")
	run("go", "build", "-trimpath", "-ldflags=-buildid=", "-o", orderBin, "./cmd/order")

	symbol := "example.test/tei-closure-fixture/internal/repair.Run"
	unifiedSymbols := run("go", "tool", "nm", unifiedBin)
	orderSymbols := run("go", "tool", "nm", orderBin)
	if !strings.Contains(unifiedSymbols, symbol) || strings.Contains(orderSymbols, symbol) {
		t.Fatalf("linked repair symbol does not match static Go entry selection")
	}
	for _, binary := range []string{unifiedBin, orderBin} {
		info, err := os.Stat(binary)
		if err != nil || info.Size() == 0 {
			t.Fatalf("Go binary absent or empty: %s (%v)", binary, err)
		}
	}
	// Beyond symbol reachability, run each locally built executable and
	// verify that the shared order action behaves identically.
	if unified.GOOS == runtime.GOOS && unified.GOARCH == runtime.GOARCH {
		unifiedResult := strings.TrimSpace(run(unifiedBin))
		orderResult := strings.TrimSpace(run(orderBin))
		if unifiedResult != "order repair" || orderResult != "order" {
			t.Fatalf("unexpected process results: unified=%q order=%q", unifiedResult, orderResult)
		}
	}
	// An E0 package-set difference plus a real linked-symbol difference
	// still does not establish the safety of TEI Provider/Capability removal.
}

func hasPath(paths []string, want string) bool {
	for _, path := range paths {
		if path == want {
			return true
		}
	}
	return false
}
