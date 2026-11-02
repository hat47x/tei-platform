package plancheck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var pressureCaseRow = regexp.MustCompile(`^\| PC-(\d+) \|`)

// Count unescaped Markdown table separators. A type expression containing an
// escaped pipe (for example A\|B) is a single cell, not another column.
func pressureTableColumns(line string) int {
	separators := 0
	for i := 0; i < len(line); i++ {
		if line[i] == '\\' && i+1 < len(line) && line[i+1] == '|' {
			i++
			continue
		}
		if line[i] == '|' {
			separators++
		}
	}
	return separators - 1
}

// Deliberately does not lock the number of cases: parallel research is free
// to append new cases, but not to silently break identities or the table.
func pressureCorpusProblems(content string) []string {
	var problems []string
	expected := 1
	for n, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "| PC-") {
			continue
		}
		match := pressureCaseRow.FindStringSubmatch(line)
		if match == nil {
			problems = append(problems, fmt.Sprintf("line %d: malformed PC identifier", n+1))
			expected++
			continue
		}
		id, err := strconv.Atoi(match[1])
		if err != nil {
			problems = append(problems, fmt.Sprintf("line %d: invalid case id: %v", n+1, err))
			expected++
			continue
		}
		if id != expected {
			problems = append(problems, fmt.Sprintf("line %d: case id PC-%d, expected PC-%d (gap, duplicate or ordering)", n+1, id, expected))
		}
		expected++
		if cells := pressureTableColumns(line); cells != 6 {
			problems = append(problems, fmt.Sprintf("line %d PC-%d: %d cells, expected 6", n+1, id, cells))
		} else {
			// An extra separator can conceal a missing field even when the
			// total column count still looks correct.
			parts := strings.Split(strings.ReplaceAll(line, `\|`, ""), "|")
			for _, part := range parts[1 : len(parts)-1] {
				if strings.TrimSpace(part) == "" {
					problems = append(problems, fmt.Sprintf("line %d PC-%d: empty cell", n+1, id))
					break
				}
			}
		}
	}
	if expected == 1 {
		problems = append(problems, "no PC rows found")
	}
	return problems
}

func TestPressureCorpusHasSequentialIDsAndSixColumns(t *testing.T) {
	path := filepath.Join(repositoryRoot(t), "plan", "design", "DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, problem := range pressureCorpusProblems(string(content)) {
		t.Error(problem)
	}
}

func TestPressureCorpusInspectorCatchesBadRows(t *testing.T) {
	good := "| PC-1 | List<A\\|B> | shape | facet | negative | owner |\n" +
		"| PC-2 | Map | shape | facet | negative | owner |\n"
	if problems := pressureCorpusProblems(good); len(problems) != 0 {
		t.Fatalf("escaped pipe valid row: %v", problems)
	}
	bad := "| PC-1 | A | shape | facet | negative | owner |\n" +
		"| PC-2 | B | facet | negative | owner |\n" +
		"| PC-2 | C | shape | facet | negative | owner |\n"
	if problems := pressureCorpusProblems(bad); len(problems) < 2 {
		t.Fatalf("expected column and duplicate-ID failures, got %v", problems)
	}
}

func TestPressureCorpusInspectorRejectsInvalidIDAndEmptyCell(t *testing.T) {
	bad := "| PC-ONE | A | shape | facet | negative | owner |\n" +
		"| PC-2 | B |  | facet | negative | owner |\n"
	if problems := pressureCorpusProblems(bad); len(problems) != 2 {
		t.Fatalf("expected malformed ID and empty cell failures, got %v", problems)
	}
}

// Numbered sections should remain in order as independent agents append
// design notes; a repeated section number silently breaks cross-references.
func TestStructureProjectionNumberedSections(t *testing.T) {
	path := filepath.Join(repositoryRoot(t), "plan", "design", "STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sections := regexp.MustCompile(`(?m)^## (\d+)\.`)
	expected := 1
	for _, match := range sections.FindAllStringSubmatch(string(data), -1) {
		actual, err := strconv.Atoi(match[1])
		if err != nil {
			t.Fatal(err)
		}
		if actual != expected {
			t.Errorf("numbered section %d appears where section %d was expected", actual, expected)
		}
		expected++
	}
	if expected == 1 {
		t.Fatal("expected numbered design sections")
	}
}

// Case rows and their Markdown header/separator must agree. Otherwise the
// rendered table quietly assigns metadata to the wrong semantic columns.
func TestPressureCorpusSectionHeadersHaveSixColumns(t *testing.T) {
	path := filepath.Join(repositoryRoot(t), "plan", "design", "DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	sections := 0
	for i, line := range lines {
		if len(line) < 5 || !strings.HasPrefix(line, "## ") ||
			line[3] < 'A' || line[3] > 'Z' || line[4] != '.' {
			continue
		}
		sections++
		found := false
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(lines[j], "## ") {
				break
			}
			if !strings.HasPrefix(lines[j], "| ID |") {
				continue
			}
			found = true
			if got := pressureTableColumns(lines[j]); got != 6 {
				t.Errorf("section %s: header has %d columns, want 6", line, got)
			}
			if j+1 >= len(lines) || !strings.HasPrefix(lines[j+1], "| ---") {
				t.Errorf("section %s: missing table separator row", line)
			} else if got := pressureTableColumns(lines[j+1]); got != 6 {
				t.Errorf("section %s: separator has %d columns, want 6", line, got)
			}
			break
		}
		if !found {
			t.Errorf("section %s: missing case table header", line)
		}
	}
	if sections == 0 {
		t.Fatal("no alphabetical pressure corpus sections")
	}
}
