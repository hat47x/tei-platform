package plancheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
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

var markdownLink = regexp.MustCompile(`\]\(([^)#\s]+)(?:#[^)]*)?\)`)

func TestMarkdownRelativeLinksResolve(t *testing.T) {
	root := repositoryRoot(t)
	var broken []string
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
		inFence := false
		for _, line := range strings.Split(string(content), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inFence = !inFence
				continue
			}
			if inFence {
				continue
			}
			for _, m := range markdownLink.FindAllStringSubmatch(line, -1) {
				target := m[1]
				if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "mailto:") {
					continue
				}
				resolved := filepath.Join(filepath.Dir(path), filepath.FromSlash(target))
				if _, err := os.Stat(resolved); err != nil {
					rel, _ := filepath.Rel(root, path)
					broken = append(broken, filepath.ToSlash(rel)+" -> "+target)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(broken)
	for _, b := range broken {
		t.Errorf("Markdownの相対リンク先が存在しません: %s", b)
	}
}

// TestMarkdownTitleAppearsOnce catches a document that was accidentally pasted into
// itself (for example by a string-replace whose replacement text contained `$`
// patterns): the title line of a document must not reappear later in the same file.
func TestMarkdownTitleAppearsOnce(t *testing.T) {
	root := repositoryRoot(t)
	var repeated []string
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
		text := string(content)
		title, _, _ := strings.Cut(text, "\n")
		if strings.HasPrefix(title, "# ") && strings.Count(text, title) > 1 {
			rel, _ := filepath.Rel(root, path)
			repeated = append(repeated, filepath.ToSlash(rel)+": "+title)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(repeated)
	for _, r := range repeated {
		t.Errorf("文書のtitle行が同じfile内で繰り返されています（自己複製の疑い）: %s", r)
	}
}
