package targetprofile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLocalUpdateLockRejectsAllWriters(t *testing.T) {
	_, _, _, candidates := selectionPaths(t)
	root := t.TempDir()
	current, retired := []byte("current"), []byte("retired")
	for name, data := range map[string][]byte{
		"artifact": current, "current": []byte("a\n"), "previous": []byte("b\n"),
		"slots/a/artifact": current, "slots/b/artifact": retired,
		"history/" + strings.TrimPrefix(digestBytes(retired), "sha256:"): retired,
	} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Reserve the on-disk lock independently of the implementation helper.
	lock := filepath.Join(root, ".update-lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		t.Fatal(err)
	}
	snapshot := func() map[string]string {
		t.Helper()
		files := map[string]string{}
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() {
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				files[path] = string(b)
			} else {
				files[path] = "<directory>"
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return files
	}
	before := snapshot()
	check := func(passed bool, reason string, err error) {
		t.Helper()
		if err != nil || passed || !strings.Contains(reason, "local update lock unavailable") {
			t.Fatalf("writer did not reject held lock: passed=%v reason=%q err=%v", passed, reason, err)
		}
		if !reflect.DeepEqual(before, snapshot()) {
			t.Fatal("blocked writer changed the root")
		}
	}
	base := filepath.Join(repoRoot(t), "tools/reference-target-profile/testdata/desktop-replace.json")
	for _, mode := range []string{UpdateReplace, UpdateInPlace, UpdateRolling} {
		profile, err := Load(base)
		if err != nil {
			t.Fatal(err)
		}
		profile.Update = mode
		b, err := json.Marshal(profile)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "profile.json")
		if err := os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
		r, err := Deploy(context.Background(), DeployOptions{ProfilePath: path, CandidatesDir: candidates, RootDir: root})
		check(r.Passed, r.Reason, err)
	}
	r, err := Rollback(root)
	check(r.Passed, r.Reason, err)
	r, err = RollbackTo(root, digestBytes(retired))
	check(r.Passed, r.Reason, err)
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	r, err = RollbackTo(root, digestBytes(retired))
	if err != nil || !r.Passed {
		t.Fatalf("released root stayed blocked: %+v %v", r, err)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatalf("successful rollback left lock: %v", err)
	}
}

func TestLocalUpdateLockHasOneConcurrentOwner(t *testing.T) {
	root := t.TempDir()
	type result struct {
		release func()
		err     error
	}
	results := make(chan result, 16)
	start := make(chan struct{})
	for i := 0; i < cap(results); i++ {
		go func() { <-start; release, err := lockLocalUpdate(root); results <- result{release, err} }()
	}
	close(start)
	owners := 0
	for i := 0; i < cap(results); i++ {
		r := <-results
		if r.err == nil {
			owners++
			defer r.release()
		}
	}
	if owners != 1 {
		t.Fatalf("got %d concurrent owners", owners)
	}
	release, err := lockLocalUpdate(t.TempDir())
	if err != nil {
		t.Fatalf("independent root blocked: %v", err)
	}
	release()
}

func TestLocalUpdateLockReleasedOnFailure(t *testing.T) {
	_, _, _, candidates := selectionPaths(t)
	root := t.TempDir()
	check := func() {
		t.Helper()
		if _, err := os.Stat(filepath.Join(root, ".update-lock")); !os.IsNotExist(err) {
			t.Fatalf("failed operation left lock: %v", err)
		}
	}
	if r, _ := Rollback(root); r.Passed {
		t.Fatal("empty rolling rollback passed")
	}
	check()
	if r, _ := RollbackTo(root, digestBytes([]byte("missing"))); r.Passed {
		t.Fatal("missing history rollback passed")
	}
	check()
	t.Setenv("PATH", t.TempDir())
	r, err := Deploy(context.Background(), DeployOptions{
		ProfilePath:   filepath.Join(repoRoot(t), "tools/reference-target-profile/testdata/desktop-replace.json"),
		CandidatesDir: candidates, RootDir: root,
	})
	if err != nil || r.Passed || !strings.Contains(r.Reason, "candidate build failed") {
		t.Fatalf("expected build failure: %+v %v", r, err)
	}
	check()
}
