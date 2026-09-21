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

// requirementIDs returns the RQ ids defined as sections of the Product Requirements document.
func requirementIDs(t *testing.T) []string {
	t.Helper()
	return idsMatching(t, filepath.Join("requirements", "TEI_PRODUCT_REQUIREMENTS.md"), `(?m)^##[ \t]+(RQ-[0-9]{3})\b`, "")
}

// idsMatching collects the capture group of every match of pattern in the file. When section is not
// empty, only the text from that heading up to the next heading of the same level is searched, so a
// table is read as itself and not as every RQ mention in the document.
func idsMatching(t *testing.T, relPath, pattern, section string) []string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), relPath))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if section != "" {
		start := strings.Index(text, section)
		if start < 0 {
			t.Fatalf("%sに見出し%qがありません。RQ coverageを追えないのでfail closedします", relPath, section)
		}
		text = text[start+len(section):]
		if end := strings.Index(text, "\n## "); end >= 0 {
			text = text[:end]
		}
	}
	var ids []string
	for _, m := range regexp.MustCompile(pattern).FindAllStringSubmatch(text, -1) {
		ids = append(ids, m[1])
	}
	return ids
}

func assertCoversEveryRequirement(t *testing.T, defined []string, relPath string, covered []string) {
	t.Helper()
	seen := map[string]int{}
	for _, id := range covered {
		seen[id]++
	}
	definedSet := map[string]bool{}
	for _, id := range defined {
		definedSet[id] = true
		switch seen[id] {
		case 0:
			t.Errorf("%sが%sに載っていません。Requirementを追加したらEvidence mapも同じ変更で更新します", id, relPath)
		case 1:
		default:
			t.Errorf("%sが%sに%d回載っています", id, relPath, seen[id])
		}
	}
	var unknown []string
	for id := range seen {
		if !definedSet[id] {
			unknown = append(unknown, id)
		}
	}
	sort.Strings(unknown)
	for _, id := range unknown {
		t.Errorf("%sは%sに載っていますが、requirements/TEI_PRODUCT_REQUIREMENTS.mdに定義がありません", id, relPath)
	}
}

// TestEveryRequirementIsMappedToAnAcceptanceScenario keeps the Requirement set and its verification
// map from drifting apart. RQ-027-030 were added by the 2026-09-18 product redefinition and were
// missing from both maps until 2026-09-21, so neither had a PAS nor an Evidence tier.
func TestEveryRequirementIsMappedToAnAcceptanceScenario(t *testing.T) {
	defined := requirementIDs(t)
	if len(defined) == 0 {
		t.Fatal("Product Requirementを1件も読み取れません")
	}
	row := `(?m)^\|[ \t]*(RQ-[0-9]{3})[ \t]*\|`
	assertCoversEveryRequirement(t, defined,
		"plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md",
		idsMatching(t, filepath.Join("plan", "UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md"), row, "## 11. Requirement → Primary PAS mapping"))
	assertCoversEveryRequirement(t, defined,
		"plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md",
		idsMatching(t, filepath.Join("plan", "PRODUCT_REQUIREMENT_GAP_AUDIT.md"), row, "## 4. Requirement別監査"))
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
