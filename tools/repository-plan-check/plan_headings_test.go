package plancheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// planSectionHeadings returns the `## N.` headings of a document in order.
// Fenced code blocks are skipped: an evidence record quotes the broken shape it
// reports, and that quotation is not a heading (#3919).
func planSectionHeadings(body string) []string {
	var out []string
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			out = append(out, line)
		}
	}
	return out
}

// TestPlanDocumentsHaveNoBackToBackSectionHeadings catches a merge artefact:
// a heading left behind without its body, immediately followed by the heading
// that replaced it. The dogfooding plan carried
//
//	## 11. 第5ラウンド…一致と入力検証
//	## 11. 第5ラウンド…一致と厳密入力
//
// so §11 named two sections and the first read as a section whose body had been
// lost (#3919). TestMarkdownTitleAppearsOnce already guards the `# ` title; the
// `## ` level had no equivalent.
func TestPlanDocumentsHaveNoBackToBackSectionHeadings(t *testing.T) {
	var failures []string
	err := filepath.Walk(filepath.Join("..", "..", "plan"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if !dogfoodingDocument(filepath.ToSlash(path)) {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Adjacency in the raw lines matters: two headings separated by body
		// text are two normal sections, while two headings with nothing between
		// them mean the first has lost its body.
		lines := strings.Split(string(raw), "\n")
		inFence := false
		previousHeading := -1
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inFence = !inFence
				previousHeading = -1
				continue
			}
			if inFence {
				continue
			}
			if !strings.HasPrefix(line, "## ") {
				if strings.TrimSpace(line) != "" {
					previousHeading = -1
				}
				continue
			}
			if previousHeading == i-1 {
				failures = append(failures, path+":"+strconv.Itoa(i)+": "+lines[previousHeading]+" | "+line)
			}
			previousHeading = i
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(failures) != 0 {
		t.Fatalf("%d back-to-back section heading(s); the first has no body:\n  %s",
			len(failures), strings.Join(failures, "\n  "))
	}
}

// TestPlanDocumentsHaveNoDuplicateSectionNumbers catches two sections claiming
// the same number, which makes a reference like "§11" ambiguous (#3919).
func TestPlanDocumentsHaveNoDuplicateSectionNumbers(t *testing.T) {
	number := regexp.MustCompile(`^## ([0-9]+)\.`)
	var failures []string
	err := filepath.Walk(filepath.Join("..", "..", "plan"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if !dogfoodingDocument(filepath.ToSlash(path)) {
			return nil
		}
		slash := filepath.ToSlash(path)
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		seen := map[string]string{}
		for _, heading := range planSectionHeadings(string(raw)) {
			match := number.FindStringSubmatch(heading)
			if match == nil {
				continue
			}
			if previous, ok := seen[match[1]]; ok {
				failures = append(failures, slash+": §"+match[1]+" twice: "+previous+" | "+heading)
				continue
			}
			seen[match[1]] = heading
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(failures) != 0 {
		t.Fatalf("%d duplicated section number(s):\n  %s", len(failures), strings.Join(failures, "\n  "))
	}
}

// dogfoodingDocument limits these checks to the documents this work owns: the
// dogfooding plan and its evidence records. A stray heading in an unrelated
// design document is that document's owner's to fix, and failing here would
// block their PR (#3919).
func dogfoodingDocument(slash string) bool {
	base := filepath.Base(slash)
	if base == "TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11.md" {
		return true
	}
	return strings.Contains(slash, "/evidence/TEI_DOGFOODING_")
}
