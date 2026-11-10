// Command adr-index-check verifies that plan/adr/README.md's index and the
// ADR files under plan/adr/ correspond without gaps: every file appears
// exactly once, every linked file exists, and each entry's stated status
// matches the file's own status line. Deleted ADRs (missing files) must be
// marked 削除 in the index. The index is documentation; this check guards
// its correspondence with the records it indexes, not their content.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type entry struct {
	number string
	link   string // href target, empty when the file is deleted
	status string // parenthesized annotation after the title, if any
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: adr-index-check <repository-root>")
		os.Exit(2)
	}
	if err := run(filepath.Join(os.Args[1], "plan", "adr")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(dir string) error {
	files, err := adrFiles(dir)
	if err != nil {
		return err
	}
	index, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		return err
	}
	indexed, errs := parseIndex(string(index))
	if len(indexed) == 0 {
		errs = append(errs, "no ADR index entries found in README.md")
	}
	fileNumbers := map[string]string{} // 4-digit number -> file base name
	for _, f := range files {
		base := filepath.Base(f)
		fileNumbers[base[:4]] = base
	}
	seen := map[string]bool{}
	for _, e := range indexed {
		if seen[e.number] {
			errs = append(errs, "duplicated index entry: "+e.number)
			continue
		}
		seen[e.number] = true
		if e.link == "" {
			// Deleted ADR: the number stays in the index without a link.
			if _, exists := fileNumbers[e.number]; exists {
				errs = append(errs, "indexed as deleted but the file exists: "+e.number)
			}
			continue
		}
		base, linked := fileNumbers[e.number]
		if !linked {
			errs = append(errs, "index links a missing file: "+e.link)
			continue
		}
		want, ok := fileStatus(filepath.Join(dir, base))
		if !ok {
			errs = append(errs, "no 状態 line in "+e.link)
			continue
		}
		if e.status != "" {
			// Both sides may annotate; the leading status word (before any
			// parenthesis) is the contract. Free-form notes may differ.
			// The index note may also use inverted word order such as
			// 目標方針として採用, so match when either head contains the
			// other's status word.
			fileHead, indexHead := statusHead(want), statusHead(strings.TrimSuffix(strings.TrimPrefix(e.status, "（"), "）"))
			if !statusWordsAgree(fileHead, indexHead) {
				errs = append(errs, fmt.Sprintf("status mismatch for %s: file=%q index=%q", e.number, want, e.status))
			}
		}
		if e.status == "" && want != "採用" {
			errs = append(errs, fmt.Sprintf("index lacks a status note for %s: file status is %q", e.number, want))
		}
	}
	for n := range fileNumbers {
		if !seen[n] {
			errs = append(errs, "ADR file missing from the index: "+n)
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("plan/adr/README.md does not match plan/adr/0*.md:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

func adrFiles(dir string) ([]string, error) {
	ms, err := filepath.Glob(filepath.Join(dir, "0*.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(ms)
	return ms, nil
}

var (
	lineRe   = regexp.MustCompile(`^- (\[ADR-(\d{4})[^\]]*\]\(([^)]+)\)|ADR-(\d{4}):[^\n]*)`)
	statusRe = regexp.MustCompile(`（([^）]*)）$`)
)

func parseIndex(index string) ([]entry, []string) {
	var entries []entry
	var errs []string
	for _, line := range strings.Split(index, "\n") {
		if !strings.HasPrefix(line, "- [ADR-") && !strings.HasPrefix(line, "- ADR-") {
			continue
		}
		m := lineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		e := entry{}
		if m[2] != "" {
			e.number, e.link = m[2], m[3]
		} else {
			e.number = m[4]
		}
		rest := strings.TrimPrefix(line, m[0])
		if sm := statusRe.FindStringSubmatch(rest); sm != nil {
			e.status = sm[0]
		} else if sm := statusRe.FindStringSubmatch(m[0]); sm != nil {
			e.status = sm[0]
		}
		if e.link == "" && !strings.Contains(e.status, "削除") {
			errs = append(errs, "unlinked index entry must state 削除: "+e.number)
		}
		entries = append(entries, e)
	}
	return entries, errs
}

func statusWordsAgree(fileHead, indexHead string) bool {
	for _, w := range []string{"採用", "提案中", "置き換え済み", "却下"} {
		if strings.Contains(fileHead, w) && strings.Contains(indexHead, w) {
			return true
		}
	}
	return fileHead == indexHead
}

func statusHead(status string) string {
	if i := strings.IndexAny(status, "（("); i >= 0 {
		return strings.TrimSpace(status[:i])
	}
	return strings.TrimSpace(status)
}

func fileStatus(path string) (string, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(b), "\n") {
		t := strings.TrimSpace(line)
		for _, prefix := range []string{"- 状態: ", "- **状態**："} {
			if strings.HasPrefix(t, prefix) {
				return strings.TrimSpace(strings.TrimPrefix(t, prefix)), true
			}
		}
	}
	return "", false
}
