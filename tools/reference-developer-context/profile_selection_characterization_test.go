package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// profileFixture creates an isolated local package; no private repository,
// network dependency, or platform-specific test binary is needed.
func profileFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files["go.mod"] = "module example.com/tei-profile-probe\n\ngo 1.20\n"
	for name, contents := range files {
		full := filepath.Join(root, name)
		if err := os.WriteFile(full, []byte(contents), 0600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return root
}

// selectedTestFiles reports Go's profile selection; it does not run the
// selected tests. This intentionally uses Go's own build constraints rather
// than adding a second build-tag parser to the developer context navigator.
func selectedTestFiles(t *testing.T, root, goos string, tags ...string) []string {
	t.Helper()
	args := []string{"list", "-json"}
	if len(tags) != 0 {
		args = append(args, "-tags="+strings.Join(tags, ","))
	}
	args = append(args, ".")
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOWORK=off", "GOTOOLCHAIN=local",
		"GOOS="+goos, "GOARCH=amd64", "CGO_ENABLED=0", "GOFLAGS=")
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %v under GOOS=%s: %v\n%s", args, goos, err, raw)
	}
	var pkg struct {
		TestGoFiles []string
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatalf("go list invalid JSON: %v\n%s", err, raw)
	}
	sort.Strings(pkg.TestGoFiles)
	return pkg.TestGoFiles
}

func discoveryCandidatePaths(report Report) []string {
	out := make([]string, 0, len(report.Candidates))
	for _, c := range report.Candidates {
		out = append(out, c.Path)
	}
	sort.Strings(out)
	return out
}

// CC-12: lexical discovery is not Go's selected test set. A cross-compiled
// profile can be inspected with go list even when its tests cannot run here.
func TestDiscoveryProfileCC12GOOSSelection(t *testing.T) {
	root := profileFixture(t, map[string]string{
		"qualification.go":              "package fixture\nfunc Evaluate() {}\n",
		"qualification_linux_test.go":   "package fixture\n",
		"qualification_windows_test.go": "package fixture\n",
	})
	r, err := Discover(root, "qualification.go")
	if err != nil {
		t.Fatal(err)
	}
	wantDiscovered := []string{"qualification_linux_test.go", "qualification_windows_test.go"}
	if got := discoveryCandidatePaths(r); !reflect.DeepEqual(got, wantDiscovered) {
		t.Fatalf("lexical candidates got %v, want %v", got, wantDiscovered)
	}
	cases := []struct {
		goos string
		want []string
	}{
		{"linux", []string{"qualification_linux_test.go"}},
		{"windows", []string{"qualification_windows_test.go"}},
	}
	for _, tc := range cases {
		t.Run(tc.goos, func(t *testing.T) {
			if got := selectedTestFiles(t, root, tc.goos); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Go-selected files got %v, want %v", got, tc.want)
			}
		})
	}
	if !r.ReadOnly || r.Accepted || r.GrantsAuthority {
		t.Fatalf("discovery must not confer meaning/authority: %+v", r)
	}
}

// CC-13: a build tag is another independent selection axis. Discover
// currently reports lexical candidates without qualifying them for tags.
func TestDiscoveryProfileCC13BuildTagSelection(t *testing.T) {
	root := profileFixture(t, map[string]string{
		"qualification.go":           "package fixture\nfunc Evaluate() {}\n",
		"qualification_base_test.go": "package fixture\n",
		"qualification_opt_test.go":  "//go:build optcheck\n\npackage fixture\n",
	})
	r, err := Discover(root, "qualification.go")
	if err != nil {
		t.Fatal(err)
	}
	wantDiscovered := []string{"qualification_base_test.go", "qualification_opt_test.go"}
	if got := discoveryCandidatePaths(r); !reflect.DeepEqual(got, wantDiscovered) {
		t.Fatalf("lexical candidates got %v, want %v", got, wantDiscovered)
	}
	if got := selectedTestFiles(t, root, "linux"); !reflect.DeepEqual(got, []string{"qualification_base_test.go"}) {
		t.Errorf("without optcheck: got %v", got)
	}
	if got := selectedTestFiles(t, root, "linux", "optcheck"); !reflect.DeepEqual(got, wantDiscovered) {
		t.Errorf("with optcheck: got %v, want %v", got, wantDiscovered)
	}
}
