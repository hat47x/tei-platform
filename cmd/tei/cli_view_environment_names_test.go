package main

import (
	"os"
	"strings"
	"testing"
)

// The runtime entry of `tei view -environment` must name nothing from a
// particular Workspace. plan/NOW.md, road 1: a definition is enough for any
// header-detail Pattern to take the same path, and an entry that knows the
// reference Workspace is no evidence of that.
func TestViewEnvironmentEntryNamesNoWorkspaceIdentifier(t *testing.T) {
	for _, file := range []string{"cli_view_environment.go"} {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"order_entry", "order_line", "order_id", "sales.", "\"development\"", "sales_core"} {
			if strings.Contains(string(content), name) {
				t.Errorf("%s names %q from the reference Workspace", file, name)
			}
		}
	}
}
