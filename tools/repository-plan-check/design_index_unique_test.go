package plancheck

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// An index entry may have a different description in a concurrent branch, but
// the destination document still has one canonical row in plan/design/README.md.
var designIndexDocumentRow = regexp.MustCompile("^\\| \\[[^]]+\\]\\(([^)]+)\\) \\|")

func duplicateDesignIndexPaths(contents string) ([]string, int, int) {
	seen := make(map[string]int)
	inIndex := false
	headingCount := 0
	entryCount := 0
	for _, raw := range strings.Split(contents, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "## 一覧" {
			headingCount++
			inIndex = true
			continue
		}
		if inIndex && strings.HasPrefix(line, "## ") {
			inIndex = false
		}
		if !inIndex {
			continue
		}
		m := designIndexDocumentRow.FindStringSubmatch(line)
		if len(m) != 2 {
			continue
		}
		seen[m[1]]++
		entryCount++
	}
	var duplicates []string
	for path, count := range seen {
		if count > 1 {
			duplicates = append(duplicates, path)
		}
	}
	sort.Strings(duplicates)
	return duplicates, headingCount, entryCount
}

func TestDesignReadmeDocumentIndexDoesNotDuplicatePaths(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "plan", "design", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	duplicates, headings, entries := duplicateDesignIndexPaths(string(contents))
	if headings != 1 || entries == 0 {
		t.Fatalf("design index has headings=%d entries=%d; expected one nonempty index", headings, entries)
	}
	if len(duplicates) != 0 {
		t.Fatalf("design index has repeated document paths after branch integration: %v", duplicates)
	}
}

func TestDesignReadmeDuplicateDetectionHandlesDescriptionAndLineEndings(t *testing.T) {
	contents := "## 一覧\n| 文書 | 対象 |\n| --- | --- |\n" +
		"| [A](A.md) | original |\n" +
		"| [B](B.md) | unique |\n" +
		"| [A](A.md) | updated description |\r\n" +
		"\n## Other\n| [A](A.md) | not an index row |\n"
	duplicates, headings, entries := duplicateDesignIndexPaths(contents)
	if headings != 1 || entries != 3 || len(duplicates) != 1 || duplicates[0] != "A.md" {
		t.Fatalf("failed to detect index duplicate: headings=%d entries=%d duplicates=%v", headings, entries, duplicates)
	}
}
