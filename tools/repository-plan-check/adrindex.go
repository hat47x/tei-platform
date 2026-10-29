// This file checks that plan/adr/README.md lists exactly the ADR files
// in plan/adr, with matching status (#2343).
//
// It reads files only. It does not decide which ADR is adopted; it reports
// where the index and the files disagree.
package plancheck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	fileNamePattern = regexp.MustCompile(`^(\d{4})-.+\.md$`)
	// - [ADR-0001: title](0001-file.md)（status note）
	linkedLine = regexp.MustCompile(`^- \[ADR-(\d{4})[:：][^\]]*\]\(([^)]+)\)(.*)$`)
	// - ADR-0002: title (the file was removed before publication)
	unlinkedLine = regexp.MustCompile(`^- ADR-(\d{4})[:：]`)
	statusLine   = regexp.MustCompile(`^- (?:\*\*)?状態(?:\*\*)?[:：]\s*(?:\*\*)?\s*(採用|提案中|置き換え済み|却下)`)
	noteStatus   = regexp.MustCompile(`^[（(](採用|提案中|置き換え済み|却下)`)
)

// Entry is one ADR line in the index.
type Entry struct {
	Number string
	Target string // empty when the index line has no link
	Status string // 採用 when the line carries no status note
}

// ParseIndex reads the 判断履歴 lines of the index.
func ParseIndex(readme string) []Entry {
	var entries []Entry
	for _, line := range strings.Split(readme, "\n") {
		line = strings.TrimRight(line, "\r")
		if m := linkedLine.FindStringSubmatch(line); m != nil {
			status := "採用"
			if n := noteStatus.FindStringSubmatch(strings.TrimSpace(m[3])); n != nil {
				status = n[1]
			}
			entries = append(entries, Entry{Number: m[1], Target: m[2], Status: status})
		} else if m := unlinkedLine.FindStringSubmatch(line); m != nil {
			entries = append(entries, Entry{Number: m[1]})
		}
	}
	return entries
}

// FileStatus returns the status the ADR file declares in its header lines.
func FileStatus(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	if len(lines) > 30 {
		lines = lines[:30]
	}
	for _, line := range lines {
		if m := statusLine.FindStringSubmatch(strings.TrimRight(line, "\r")); m != nil {
			return m[1], true
		}
	}
	return "", false
}

// CheckADRIndex compares dir/README.md with the ADR files in dir and returns one
// message per disagreement. An empty result means they agree.
func CheckADRIndex(dir string) ([]string, error) {
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		return nil, err
	}
	names, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	filesByNumber := map[string][]string{}
	for _, n := range names {
		if m := fileNamePattern.FindStringSubmatch(n.Name()); m != nil && !n.IsDir() {
			filesByNumber[m[1]] = append(filesByNumber[m[1]], n.Name())
		}
	}

	var problems []string
	numbers := make([]string, 0, len(filesByNumber))
	for number, files := range filesByNumber {
		numbers = append(numbers, number)
		if len(files) > 1 {
			sort.Strings(files)
			problems = append(problems, fmt.Sprintf("ADR-%s: 同じ番号のファイルが%d件ある: %s", number, len(files), strings.Join(files, ", ")))
		}
	}
	sort.Strings(numbers)

	seen := map[string]bool{}
	for _, e := range ParseIndex(string(readme)) {
		if seen[e.Number] {
			problems = append(problems, fmt.Sprintf("ADR-%s: 索引に複数回載っている", e.Number))
		}
		seen[e.Number] = true
		files := filesByNumber[e.Number]
		if e.Target == "" {
			if len(files) > 0 {
				problems = append(problems, fmt.Sprintf("ADR-%s: ファイル %s があるのに、索引にリンクがない", e.Number, files[0]))
			}
			continue // 公開前に削除したADRは、番号だけ残してよい
		}
		if _, err := os.Stat(filepath.Join(dir, e.Target)); err != nil {
			problems = append(problems, fmt.Sprintf("ADR-%s: 索引のリンク先 %s がない", e.Number, e.Target))
			continue
		}
		if !strings.HasPrefix(e.Target, e.Number+"-") {
			problems = append(problems, fmt.Sprintf("ADR-%s: 索引のリンク先 %s の番号が一致しない", e.Number, e.Target))
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Target))
		if err != nil {
			return nil, err
		}
		status, ok := FileStatus(string(body))
		if !ok {
			problems = append(problems, fmt.Sprintf("ADR-%s: %s の冒頭に状態の行を読み取れない", e.Number, e.Target))
		} else if status != e.Status {
			problems = append(problems, fmt.Sprintf("ADR-%s: 状態が一致しない（索引: %s、ファイル: %s）", e.Number, e.Status, status))
		}
	}
	for _, number := range numbers {
		if !seen[number] {
			problems = append(problems, fmt.Sprintf("ADR-%s: ファイル %s が索引に載っていない", number, filesByNumber[number][0]))
		}
	}
	sort.Strings(problems)
	return problems, nil
}
