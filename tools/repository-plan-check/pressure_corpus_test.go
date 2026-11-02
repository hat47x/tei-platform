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
		match := pressureCaseRow.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		id, err := strconv.Atoi(match[1])
		if err != nil {
			problems = append(problems, fmt.Sprintf("line %d: invalid case id: %v", n+1, err))
			continue
		}
		if id != expected {
			problems = append(problems, fmt.Sprintf("line %d: case id PC-%d, expected PC-%d (gap, duplicate or ordering)", n+1, id, expected))
		}
		expected = id + 1
		if cells := pressureTableColumns(line); cells != 6 {
			problems = append(problems, fmt.Sprintf("line %d PC-%d: %d cells, expected 6", n+1, id, cells))
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
