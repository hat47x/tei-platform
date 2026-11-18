package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// What a header-detail binding states about which rows its Contract Query
// returns is checked by `tei validate` (plan/design/RULE_NORMAL_FORM_AND_
// OPERATOR_SIGNATURE_DECISION_2026-11.md §6). The reference Workspace states
// `covers sales.order.lines`; each case replaces that line in a copy.
func TestValidateChecksWhatABindingStatesAboutItsRows(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := buildCLIForLifecycle(t)
	const anchor = "    covers sales.order.lines\n"
	for _, tc := range []struct {
		name    string
		replace string
		exit    int
		want    string
		absent  string
	}{
		{name: "nothing stated", replace: "", exit: 0, want: "HEADER_DETAIL_BINDING_COVERAGE_UNDECLARED"},
		{name: "the reference Workspace, every line of the order", replace: anchor, exit: 0, absent: "COVERAGE"},
		{name: "only some rows", replace: "    covers some\n", exit: 0, absent: "COVERAGE"},
		{
			// The warning offers `covers %s` as the way to state what the Query
			// returns. That statement is not a proof that every row arrived, so
			// the advice must not read as "add this and the rule is satisfied".
			// A read of a declared binding still answers the rule undecided
			// (RULE_NORMAL_FORM decision §6).
			name:    "the advice does not promise satisfaction",
			replace: "",
			exit:    0,
			want:    "not a proof that every row arrived",
		},
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
			if strings.Count(string(content), anchor) != 1 || strings.Count(string(content), "covers") != 1 {
				t.Fatalf("the reference binding changed:\n%s", content)
			}
			if err := os.WriteFile(path, []byte(strings.Replace(string(content), anchor, tc.replace, 1)), 0o644); err != nil {
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
