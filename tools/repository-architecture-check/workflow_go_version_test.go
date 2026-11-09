package architecturecheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var quotedGoVersion = regexp.MustCompile(`(?m)^\s*go-version:\s*['"]?[0-9]`)

// TestWorkflowsFollowGoModVersion pins the single source of the Go version.
// A workflow that writes a literal go-version can drift from go.mod and fail
// before its test bodies run (#2948). go-version-file: go.mod is the norm; the
// only exception is a workflow that documents why it must pin a different
// toolchain separately.
func TestWorkflowsFollowGoModVersion(t *testing.T) {
	root := filepath.Join("..", "..")
	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(workflows) == 0 {
		t.Fatal("no workflow files found")
	}
	for _, workflow := range workflows {
		raw, err := os.ReadFile(workflow)
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		if !quotedGoVersion.MatchString(text) {
			continue
		}
		name := filepath.Base(workflow)
		if !strings.Contains(text, "# allow-literal-go-version:") {
			// Every literal pin must explain itself. The Studio fixed Host
			// workflow was the one drift found in #2948.
			t.Errorf("%s pins a literal go-version; use go-version-file: go.mod or add an allow-literal-go-version note", name)
		}
	}
}

// TestQuotedGoVersionPatternDetectsLiteralPins is the negative control: the
// guard must notice a literal pin, otherwise it passes vacuously after a
// workflow drifts.
func TestQuotedGoVersionPatternDetectsLiteralPins(t *testing.T) {
	for _, line := range []string{
		"          go-version: '1.26.9'",
		"          go-version: \"1.26.9\"",
		"          go-version: 1.26.9",
	} {
		if !quotedGoVersion.MatchString(line) {
			t.Errorf("literal pin not detected: %q", line)
		}
	}
	for _, line := range []string{
		"          go-version-file: go.mod",
		"          # go-version: '1.26.9' in a comment is still a literal note",
	} {
		if quotedGoVersion.MatchString(line) && line == "          go-version-file: go.mod" {
			t.Errorf("go-version-file was treated as a literal pin: %q", line)
		}
	}
}
