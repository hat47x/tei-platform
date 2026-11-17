package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// What a header-detail binding states about which rows its Contract Query
// returns is checked by `tei validate` (plan/design/RULE_NORMAL_FORM_AND_
// OPERATOR_SIGNATURE_DECISION_2026-11.md §6). The reference Workspace does
// not state it yet: its accepted state changes only through the shared
// acceptance path, so the statement is added to a copy here.
func TestValidateChecksWhatABindingStatesAboutItsRows(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := buildCLIForLifecycle(t)
	const anchor = "    input sales.order.order_id to order_id\n"
	for _, tc := range []struct {
		name    string
		replace string
		exit    int
		want    string
		absent  string
	}{
		{name: "the reference Workspace, nothing stated", replace: "", exit: 0, want: "HEADER_DETAIL_BINDING_COVERAGE_UNDECLARED"},
		{name: "every line of the order", replace: "    covers sales.order.lines\n", exit: 0, absent: "COVERAGE"},
		{name: "only some rows", replace: "    covers some\n", exit: 0, absent: "COVERAGE"},
		{name: "another relation", replace: "    covers sales.order.customer\n", exit: 1, want: "HEADER_DETAIL_BINDING_COVERS_OTHER_RELATION"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
			path := filepath.Join(root, "applications", "order-entry", "realizations", "header-detail", "order-lines.tei")
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(content), anchor) != 1 || strings.Contains(string(content), "covers") {
				t.Fatalf("the reference binding changed:\n%s", content)
			}
			if err := os.WriteFile(path, []byte(strings.Replace(string(content), anchor, anchor+tc.replace, 1)), 0o644); err != nil {
				t.Fatal(err)
			}
			got := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
			output := got.stdout + got.stderr
			if got.code != tc.exit || (tc.want != "" && !strings.Contains(output, tc.want)) || (tc.absent != "" && strings.Contains(output, tc.absent)) {
				t.Fatalf("code=%d output=%s", got.code, output)
			}
		})
	}
}
