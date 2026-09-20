package plancheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source pathを取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func numberedFiles(t *testing.T, dir string, digits int) map[string][]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`^([0-9]{` + itoa(digits) + `})-.+\.md$`)
	byNumber := map[string][]string{}
	for _, entry := range entries {
		if m := pattern.FindStringSubmatch(entry.Name()); m != nil {
			byNumber[m[1]] = append(byNumber[m[1]], entry.Name())
		}
	}
	return byNumber
}

func itoa(n int) string {
	return string(rune('0' + n))
}

func assertUnique(t *testing.T, kind string, byNumber map[string][]string) {
	t.Helper()
	var numbers []string
	for number, files := range byNumber {
		if len(files) > 1 {
			numbers = append(numbers, number)
		}
	}
	sort.Strings(numbers)
	for _, number := range numbers {
		t.Errorf("%s番号%sが複数のfileで使われています: %v", kind, number, byNumber[number])
	}
}

func TestIssueNumbersAreUnique(t *testing.T) {
	assertUnique(t, "Issue", numberedFiles(t, filepath.Join(repositoryRoot(t), "plan", "issue"), 6))
}

func TestADRNumbersAreUnique(t *testing.T) {
	assertUnique(t, "ADR", numberedFiles(t, filepath.Join(repositoryRoot(t), "plan", "adr"), 4))
}

func TestIssueStatesUseTheSixAllowedValues(t *testing.T) {
	root := filepath.Join(repositoryRoot(t), "plan", "issue")
	allowed := map[string]bool{
		"Proposed": true, "Ready": true, "In Progress": true,
		"Blocked": true, "Done": true, "Superseded": true,
	}
	statusLine := regexp.MustCompile(`(?m)^- 状態:[ \t]*(.+)$`)
	separator := regexp.MustCompile(`\s+[—–-]\s+|\s*/\s*|[（(]`)
	for _, files := range numberedFiles(t, root, 6) {
		for _, name := range files {
			content, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			m := statusLine.FindStringSubmatch(string(content))
			if m == nil {
				t.Errorf("%sに`- 状態:`行がありません", name)
				continue
			}
			value := strings.TrimSpace(m[1])
			head := strings.TrimSpace(separator.Split(value, 2)[0])
			if !allowed[head] {
				t.Errorf("%sの状態%qが、AGENTS.mdの6値（Proposed / Ready / In Progress / Blocked / Done / Superseded）の外です", name, value)
			}
		}
	}
}

func forEachMarkdown(t *testing.T, visit func(rel, text string)) {
	t.Helper()
	root := repositoryRoot(t)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		visit(filepath.ToSlash(rel), string(content))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func outsideCodeFences(text string, visit func(lineNumber int, line string)) {
	inFence := false
	for i, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			visit(i+1, line)
		}
	}
}

var markdownLink = regexp.MustCompile(`\]\(([^)#\s]+)(?:#[^)]*)?\)`)

func TestMarkdownRelativeLinksResolve(t *testing.T) {
	root := repositoryRoot(t)
	var broken []string
	forEachMarkdown(t, func(rel, text string) {
		dir := filepath.Dir(filepath.Join(root, filepath.FromSlash(rel)))
		outsideCodeFences(text, func(_ int, line string) {
			for _, m := range markdownLink.FindAllStringSubmatch(line, -1) {
				target := m[1]
				if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "mailto:") {
					continue
				}
				if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(target))); err != nil {
					broken = append(broken, rel+" -> "+target)
				}
			}
		})
	})
	sort.Strings(broken)
	for _, b := range broken {
		t.Errorf("Markdownの相対リンク先が存在しません: %s", b)
	}
}

// A document pasted into itself repeats its own title line.
func TestMarkdownTitleAppearsOnce(t *testing.T) {
	var repeated []string
	forEachMarkdown(t, func(rel, text string) {
		title, _, _ := strings.Cut(text, "\n")
		title = strings.TrimRight(title, "\r")
		if !strings.HasPrefix(title, "# ") {
			return
		}
		count := 0
		outsideCodeFences(text, func(_ int, line string) {
			if strings.TrimRight(line, "\r") == title {
				count++
			}
		})
		if count > 1 {
			repeated = append(repeated, rel+": "+title)
		}
	})
	sort.Strings(repeated)
	for _, r := range repeated {
		t.Errorf("文書のtitle行が同じfile内で繰り返されています（自己複製の疑い）: %s", r)
	}
}

// A heredoc that escapes every code-span backtick with a backslash breaks the code spans.
func TestMarkdownHasNoEscapedBackticks(t *testing.T) {
	var offenders []string
	forEachMarkdown(t, func(rel, text string) {
		reported := false
		outsideCodeFences(text, func(lineNumber int, line string) {
			if !reported && strings.Contains(line, "\\`") {
				reported = true
				offenders = append(offenders, rel+":"+strconv.Itoa(lineNumber))
			}
		})
	})
	sort.Strings(offenders)
	for _, o := range offenders {
		t.Errorf("バッククォートがバックスラッシュでescapeされています（heredocのescape事故の疑い）: %s", o)
	}
}
