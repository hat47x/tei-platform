package plancheck

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// skipVanishedPath decides what a walk over the repository should do when the
// entry it was handed can no longer be read.
//
// Several tests create a temporary directory under the repository root and
// remove it again (internal/semantic/validationgen and internal/operations do,
// for build fixtures). When the removal lands between a directory being listed
// and being visited, WalkDir reports fs.ErrNotExist for a path that has nothing
// to do with the property being checked, and the check fails for reasons that
// depend on test scheduling. Skipping only that case keeps the run deterministic.
//
// Every other error is returned to the caller. A permission failure or an I/O
// error could hide a file the check exists to inspect, and swallowing those
// would make the check quietly weaker rather than more stable.
func skipVanishedPath(err error) error {
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// The error WalkDir produces for a directory removed during the walk is the one
// case that must be skipped. This uses the real error rather than a sentinel, so
// the test would notice if the detection stopped matching what the filesystem
// actually returns.
func TestAVanishedPathIsSkipped(t *testing.T) {
	root := t.TempDir()
	doomed := filepath.Join(root, "doomed")
	if err := os.Mkdir(doomed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(doomed); err != nil {
		t.Fatal(err)
	}
	// Reading the vanished directory produces the same error WalkDir hands a
	// callback when the entry disappeared after being listed.
	_, err := os.ReadDir(doomed)
	if err == nil {
		t.Fatal("reading a removed directory must fail")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the removed directory did not report ErrNotExist: %v", err)
	}
	if got := skipVanishedPath(err); got != nil {
		t.Fatalf("a vanished path must be skipped, got %v", got)
	}
}

// Every other error must reach the caller. A check that swallowed a permission
// error would silently skip files it exists to inspect.
func TestEveryOtherErrorIsReturned(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: "permission", err: fs.ErrPermission},
		{name: "not a directory", err: errors.New("not a directory")},
		{name: "i/o", err: errors.New("input/output error")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := skipVanishedPath(tc.err); got == nil {
				t.Fatalf("%v must be returned, not skipped", tc.err)
			} else if !errors.Is(got, tc.err) {
				t.Fatalf("the returned error lost the cause: %v", got)
			}
		})
	}
}

// A nil error is "no problem", so the helper must not turn it into one.
func TestNoErrorStaysNoError(t *testing.T) {
	if got := skipVanishedPath(nil); got != nil {
		t.Fatalf("nil must stay nil, got %v", got)
	}
}
