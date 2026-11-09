// This file checks that a directory README mentions every document in the
// directory (#2356).
//
// It only checks that each entry name appears in the README. It does not
// judge how the entry is described, and it does not read document states:
// a document's own header stays the source of its state.
package plancheck

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CheckReadmeMentionsEntries returns one message per entry of dir (other than
// README.md and hidden files) whose name does not appear in dir/README.md.
// A directory entry is matched by its name, with or without a trailing slash.
func CheckReadmeMentionsEntries(dir string) ([]string, error) {
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	text := string(readme)
	var problems []string
	for _, e := range entries {
		name := e.Name()
		if name == "README.md" || strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.Contains(text, name) {
			problems = append(problems, fmt.Sprintf("%s: README.md に載っていない", name))
		}
	}
	sort.Strings(problems)
	return problems, nil
}
