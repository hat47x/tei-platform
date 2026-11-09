package plancheck

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// goModules are the go.mod files whose requirements THIRD_PARTY_NOTICES.md must account for.
var goModules = []string{"go.mod", filepath.Join("cmd", "tei-studio", "go.mod")}

// knownUnlistedModules are requirements that THIRD_PARTY_NOTICES.md does not list yet.
//
// This is a record of a gap reported in an issue, not a decision. Whether these
// modules belong in the notices (product binary or reference tool only), and under
// which license terms, is a human decision (AGENTS.md A4). Remove an entry in the
// same change that lists the module in THIRD_PARTY_NOTICES.md or records why it is
// out of scope there. Never add an entry to silence the check for a new dependency.
//
// Empty since #2330: every module of both go.mod files is now listed in
// THIRD_PARTY_NOTICES.md, with the license facts read from the exact versions.
var knownUnlistedModules = map[string]bool{}

type moduleRequirement struct {
	path, version string
}

var requireLine = regexp.MustCompile(`^\s*([A-Za-z0-9][^\s]*)\s+(v[0-9][^\s]*)(?:\s+//.*)?$`)

// goModRequirements returns the modules named in the require directives of a go.mod
// text, both the block and the single-line form. The module itself is not included.
func goModRequirements(text string) []moduleRequirement {
	var result []moduleRequirement
	inBlock := false
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		switch {
		case inBlock && trimmed == ")":
			inBlock = false
		case inBlock:
			if m := requireLine.FindStringSubmatch(line); m != nil {
				result = append(result, moduleRequirement{m[1], m[2]})
			}
		case trimmed == "require (":
			inBlock = true
		case strings.HasPrefix(trimmed, "require "):
			if m := requireLine.FindStringSubmatch(strings.TrimPrefix(trimmed, "require ")); m != nil {
				result = append(result, moduleRequirement{m[1], m[2]})
			}
		}
	}
	return result
}

// unaccountedModules returns the requirements that no line of notices names together
// with the resolved version, and that are not in known. A requirement that is
// on a line of the notices with a different version is reported too, so that a
// version bump cannot leave a stale license record. The second result lists
// entries of known that no longer apply.
func unaccountedModules(requirements []moduleRequirement, notices string, known map[string]bool) (missing, staleKnown []string) {
	lines := strings.Split(notices, "\n")
	required := map[string]bool{}
	for _, r := range requirements {
		if r.path == "tei" {
			continue
		}
		required[r.path] = true
		listed := false
		for _, line := range lines {
			if strings.Contains(line, r.path) && strings.Contains(line, r.version) {
				listed = true
				break
			}
		}
		if listed {
			if known[r.path] {
				staleKnown = append(staleKnown, r.path+"は既にTHIRD_PARTY_NOTICES.mdへ載っています")
			}
			continue
		}
		if !known[r.path] {
			missing = append(missing, r.path+" "+r.version)
		}
	}
	for path := range known {
		if !required[path] {
			staleKnown = append(staleKnown, path+"はどのgo.modにもありません")
		}
	}
	sort.Strings(missing)
	sort.Strings(staleKnown)
	return missing, staleKnown
}

func TestUnaccountedModuleDetection(t *testing.T) {
	gomod := "module tei\n\ngo 1.26.9\n\nrequire (\n\tgithub.com/a/direct v1.2.3\n\tgithub.com/b/indirect v0.1.0 // indirect\n)\n\nrequire github.com/c/single v2.0.0+incompatible\n\nreplace tei => ../..\n"
	reqs := goModRequirements(gomod)
	if len(reqs) != 3 || reqs[0] != (moduleRequirement{"github.com/a/direct", "v1.2.3"}) || reqs[2].path != "github.com/c/single" {
		t.Fatalf("require directiveの読取り結果が想定と異なります: %v", reqs)
	}

	all := "### github.com/a/direct v1.2.3\n| `github.com/b/indirect` | v0.1.0 | MIT |\n- github.com/c/single v2.0.0+incompatible\n"
	if missing, stale := unaccountedModules(reqs, all, nil); len(missing) != 0 || len(stale) != 0 {
		t.Errorf("全て載っているのに報告されました: missing=%v stale=%v", missing, stale)
	}

	// A new dependency that the notices omit must fail.
	if missing, _ := unaccountedModules(reqs, "### github.com/a/direct v1.2.3\n", nil); len(missing) != 2 {
		t.Errorf("載っていないmoduleを検出できません: %v", missing)
	}

	// A version bump that leaves the old version in the notices must fail.
	if missing, _ := unaccountedModules(reqs, "github.com/a/direct v1.2.2\n`github.com/b/indirect` v0.1.0\ngithub.com/c/single v2.0.0+incompatible\n", nil); len(missing) != 1 || !strings.HasPrefix(missing[0], "github.com/a/direct") {
		t.Errorf("版の食い違いを検出できません: %v", missing)
	}

	// Known gaps are tolerated, and an entry that is no longer needed must fail.
	known := map[string]bool{"github.com/c/single": true, "github.com/gone/module": true}
	missing, stale := unaccountedModules(reqs, "github.com/a/direct v1.2.3\ngithub.com/b/indirect v0.1.0\n", known)
	if len(missing) != 0 || len(stale) != 1 || !strings.HasPrefix(stale[0], "github.com/gone/module") {
		t.Errorf("既知の欠落の扱いが想定と異なります: missing=%v stale=%v", missing, stale)
	}
	_, stale = unaccountedModules(reqs, all, map[string]bool{"github.com/c/single": true})
	if len(stale) != 1 {
		t.Errorf("載せた後に残った既知の欠落を検出できません: %v", stale)
	}
}

// TestThirdPartyNoticesCoverGoModRequirements compares the go.mod requirements of the
// root module and cmd/tei-studio with THIRD_PARTY_NOTICES.md (#2330). A new dependency,
// or a version bump, that the notices do not follow fails here.
func TestThirdPartyNoticesCoverGoModRequirements(t *testing.T) {
	root := repositoryRoot(t)
	noticesBytes, err := os.ReadFile(filepath.Join(root, "THIRD_PARTY_NOTICES.md"))
	if err != nil {
		t.Fatal(err)
	}
	var requirements []moduleRequirement
	for _, name := range goModules {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		reqs := goModRequirements(string(content))
		if len(reqs) == 0 {
			t.Fatalf("%sのrequireを1件も読み取れません", filepath.ToSlash(name))
		}
		requirements = append(requirements, reqs...)
	}
	missing, stale := unaccountedModules(requirements, string(noticesBytes), knownUnlistedModules)
	for _, m := range missing {
		t.Errorf("THIRD_PARTY_NOTICES.mdに載っていないgo.modの依存です（該当版を同じ行に併記します）: %s", m)
	}
	for _, s := range stale {
		t.Errorf("knownUnlistedModulesの整理が必要です: %s", s)
	}
}
