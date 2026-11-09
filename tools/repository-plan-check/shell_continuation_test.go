package plancheck

import (
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// bs is one backslash. The test data below is built from it so that the
// number of backslashes in each case stays visible in the source.
const bs = "\x5c"

// shellFenceLanguages are the code-fence languages whose lines are read as shell
// commands that a reader pastes into a terminal.
var shellFenceLanguages = map[string]bool{"bash": true, "sh": true, "shell": true, "zsh": true}

// doubleBackslashContinuations returns the 1-based line numbers of lines inside
// bash/sh/shell/zsh code fences that end in exactly two backslashes.
//
// In a shell, a backslash followed by a newline continues the command. Two
// backslashes are an escaped backslash, so the next line runs as a separate
// command and a literal backslash is passed as an argument. Rendering hides the
// mistake because Markdown shows the same line either way (#2331).
func doubleBackslashContinuations(text string) []int {
	var lines []int
	fenceMarker := ""
	shell := false
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimLeft(line, " \t")
		if fenceMarker == "" {
			marker := fenceOpening(trimmed)
			if marker == "" {
				continue
			}
			fenceMarker = marker
			info := strings.Fields(strings.TrimLeft(trimmed, marker[:1]))
			shell = len(info) > 0 && shellFenceLanguages[strings.ToLower(info[0])]
			continue
		}
		if strings.HasPrefix(trimmed, fenceMarker) && strings.Trim(trimmed, fenceMarker[:1]) == "" {
			fenceMarker = ""
			shell = false
			continue
		}
		if shell && trailingBackslashes(line) == 2 {
			lines = append(lines, i+1)
		}
	}
	return lines
}

// fenceOpening returns the opening fence (three or more backticks or tildes), or "".
func fenceOpening(trimmed string) string {
	for _, ch := range []string{"`", "~"} {
		n := len(trimmed) - len(strings.TrimLeft(trimmed, ch))
		if n >= 3 {
			// The info string of a backtick fence cannot contain backticks.
			if ch == "`" && strings.Contains(trimmed[n:], "`") {
				return ""
			}
			return strings.Repeat(ch, n)
		}
	}
	return ""
}

func trailingBackslashes(line string) int {
	return len(line) - len(strings.TrimRight(line, bs))
}

func TestDoubleBackslashContinuationDetection(t *testing.T) {
	const fence = "```"
	two := strings.Repeat(bs, 2)
	three := strings.Repeat(bs, 3)
	cases := []struct {
		name string
		text string
		want []int
	}{
		{"bash block with doubled continuation", fence + "bash\ngo run ./cmd/tei validate " + two + "\n  -workspace x\n" + fence + "\n", []int{2}},
		{"sh, shell and zsh are shell blocks", fence + "sh\na " + two + "\n" + fence + "\n" + fence + "shell\nb " + two + "\n" + fence + "\n" + fence + "zsh\nc " + two + "\n" + fence + "\n", []int{2, 5, 8}},
		{"language is case-insensitive and may carry attributes", fence + "Bash title=x\na " + two + "\n" + fence + "\n", []int{2}},
		{"single backslash continuation is correct", fence + "bash\ngo run ./cmd/tei validate " + bs + "\n  -workspace x\n" + fence + "\n", nil},
		{"three trailing backslashes end in a continuation", fence + "bash\na " + three + "\n" + fence + "\n", nil},
		{"two backslashes in the middle of a line are not flagged", fence + "bash\necho " + two + " x\n" + fence + "\n", nil},
		{"CRLF line endings are accepted", fence + "bash\r\na " + two + "\r\nb\r\n" + fence + "\r\n", []int{2}},
		{"other languages are not shell", fence + "go\nx := 1 " + two + "\n" + fence + "\n" + fence + "\ns " + two + "\n" + fence + "\n" + fence + "text\ns " + two + "\n" + fence + "\n", nil},
		{"text outside a fence is not checked", "prose " + two + "\n", nil},
		{"tilde fence is a fence", "~~~bash\na " + two + "\n~~~\n", []int{2}},
		{"closing fence must be as long as the opening one", "````bash\n" + fence + "\na " + two + "\n````\nafter " + two + "\n", []int{3}},
		{"indented fence in a list item", "- item\n\n  " + fence + "bash\n  a " + two + "\n  " + fence + "\n", []int{4}},
		{"a block after the closing fence is judged by its own language", fence + "bash\nok\n" + fence + "\n" + fence + "json\n{\"a\": 1} " + two + "\n" + fence + "\n", nil},
	}
	for _, c := range cases {
		if got := doubleBackslashContinuations(c.text); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// TestShellCodeBlocksHaveNoDoubleBackslashContinuation keeps the paste-and-run
// commands in the documents valid. Link checks do not read code blocks, so a
// broken command line would otherwise pass every existing document check.
func TestShellCodeBlocksHaveNoDoubleBackslashContinuation(t *testing.T) {
	var offenders []string
	forEachMarkdown(t, func(rel, text string) {
		for _, line := range doubleBackslashContinuations(text) {
			offenders = append(offenders, rel+":"+strconv.Itoa(line))
		}
	})
	sort.Strings(offenders)
	for _, o := range offenders {
		t.Errorf("shellのコードブロックの行末が2文字のバックスラッシュです。1文字にしないと行継続にならず、貼り付けて実行できません: %s", o)
	}
}
