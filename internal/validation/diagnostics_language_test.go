package validation

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// japaneseDiagnostic matches a message written in Japanese, which is how
// diagnostics reached the user through `tei validate` before #4174.
var japaneseDiagnostic = regexp.MustCompile(`[\x{3040}-\x{30ff}\x{4e00}-\x{9fff}]`)

// TestDiagnosticsReachTheUserInEnglish pins #4174. The CLI's diagnostics were
// mixed English and Japanese inside the same code class, so a caller could not
// classify or summarise them mechanically, and a collaborator reading English
// could not read the Japanese ones.
//
// The check is on non-comment source lines only: comments are the
// developer-facing explanation (Japanese, per the repository documentation
// rules) and a diagnostic is the string a user reads.
func TestDiagnosticsReachTheUserInEnglish(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	offenders := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		body := strings.TrimPrefix(string(raw), "\xef\xbb\xbf")
		for i, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if japaneseDiagnostic.MatchString(line) {
				offenders = append(offenders,
					name+":"+strconv.Itoa(i+1)+" "+strings.TrimSpace(line))
			}
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("these lines still write a diagnostic in Japanese:\n  %s",
			strings.Join(offenders, "\n  "))
	}
}
