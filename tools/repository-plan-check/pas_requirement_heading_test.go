package plancheck

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The PAS mapping table can contain one row per requirement and still be
// misleading if two detailed sections claim the same RQ identity (as happened
// when a BOM section and a Model Map section both claimed RQ-036).
// This guard checks section identity, not business acceptance or semantics.
var pasRequirementSection = regexp.MustCompile(`^### (RQ-[0-9]{3}) — `)

func pasRequirementSections(text string) (ids []string, repeated []string) {
	seen := map[string]bool{}
	duplicates := map[string]bool{}
	outsideCodeFences(text, func(_ int, line string) {
		m := pasRequirementSection.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			return
		}
		id := m[1]
		ids = append(ids, id)
		if seen[id] {
			duplicates[id] = true
		}
		seen[id] = true
	})
	for id := range duplicates {
		repeated = append(repeated, id)
	}
	sort.Strings(ids)
	sort.Strings(repeated)
	return ids, repeated
}

func TestPASDetailedRequirementSectionsDoNotReuseIDs(t *testing.T) {
	path := filepath.Join(repositoryRoot(t), "plan", "UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ids, repeated := pasRequirementSections(string(data))
	if len(ids) == 0 {
		t.Fatal("no PAS detailed requirement sections were found")
	}
	for _, id := range repeated {
		t.Errorf("same Requirement has more than one PAS detailed section: %s", id)
	}
	defined := map[string]bool{}
	for _, id := range requirementIDs(t) {
		defined[id] = true
	}
	for _, id := range ids {
		if !defined[id] {
			t.Errorf("PAS detail has an undefined Requirement: %s", id)
		}
	}
}

func TestPASDuplicateRequirementHeadingsRegression(t *testing.T) {
	synthetic := strings.Join([]string{
		"### RQ-035 — Workbench",
		"### RQ-036 — Model Map",
		"### RQ-036 — BOM",
		"```markdown",
		"### RQ-036 — fenced example, not a section",
		"```",
		"### RQ-037 — Verified BOM",
	}, "\n")
	_, repeated := pasRequirementSections(synthetic)
	if len(repeated) != 1 || repeated[0] != "RQ-036" {
		t.Fatalf("duplicate section regression failed: %v", repeated)
	}
	_, repeated = pasRequirementSections("### RQ-035 — Workbench\n### RQ-036 — Model Map\n### RQ-037 — BOM\n")
	if len(repeated) != 0 {
		t.Fatalf("distinct requirement headings were rejected: %v", repeated)
	}
}
