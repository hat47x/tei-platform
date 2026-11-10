package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CC-15 is a deliberately staged, harmless path-check/open counterexample.
// This test is NOT an exploitation of readInside or of Discover: it performs
// the mutation between the operations manually, because readInside exposes
// no injection hook for scheduling a concurrent filesystem change.
func TestActualReadCC15PathCheckIsNotFileOpenIdentity(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	const insideBody = "package fixture\n// an in-root public fixture\n"
	const canary = "PUBLIC_SYNTHETIC_OUTSIDE_CANARY_CC15"
	const name = "qualification.go"

	full := filepath.Join(root, name)
	outsidePath := filepath.Join(outside, "public-canary.txt")
	if err := os.WriteFile(full, []byte(insideBody), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outsidePath, []byte(canary), 0600); err != nil {
		t.Fatal(err)
	}

	// These are the initial check steps from readInside: a successful
	// resolved-path check followed by a regular-file Lstat at that point.
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		t.Fatal(err)
	}
	// Compare a resolved child against a resolved root. Comparing it against the
	// unresolved root reports every child as outside whenever the root is
	// reached through a link or a Windows short name (#3783).
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("the initial regular file was not inside root: rel=%q err=%v", rel, err)
	}
	before, err := os.Lstat(full)
	if err != nil || !before.Mode().IsRegular() || before.Size() > maxFileBytes {
		t.Fatalf("initial Lstat did not match: info=%v err=%v", before, err)
	}

	// A controlled path replacement after validation, before the final
	// os.ReadFile. The outside file is an explicit public test-only canary.
	if err := os.Rename(full, filepath.Join(root, "qualification-old.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, full); err != nil {
		t.Skipf("creating symlinks is unavailable in this test environment: %v", err)
	}

	// This last operation is exactly the kind of unrestricted path-based
	// open used by readInside after its earlier validations. It follows the
	// new symlink and reads the harmless outside canary.
	raw, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != canary {
		t.Fatalf("staged path replacement did not change opened bytes: %q", raw)
	}

	// In contrast, calling the actual readInside *after* the replacement
	// revalidates the symlink and refuses the path. The test intentionally
	// does not claim that an actual concurrent readInside call escaped.
	if _, err := readInside(root, name); err == nil {
		t.Fatal("readInside must reject a preexisting out-of-root symlink")
	}

	// A root-relative open is a candidate mitigation for this particular
	// traversal class. It does not establish a joint multi-file snapshot,
	// guard against all mount/device/platform effects, or resolve Owner
	// authorization and consumer disclosure.
	r, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	f, err := r.Open(name)
	if err == nil {
		f.Close()
		t.Fatal("os.Root.Open accepted a symlink outside the test root")
	}
}

func TestActualReadCC15StableInRootFileRemainsReadable(t *testing.T) {
	root := t.TempDir()
	const name = "qualification.go"
	const body = "package fixture\nfunc Evaluate() int { return 1 }\n"
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readInside(root, name)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("stable in-root source contents changed: %q", got)
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	f, err := r.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != body {
		t.Fatalf("root-relative open read %q instead of fixture", b)
	}
}
