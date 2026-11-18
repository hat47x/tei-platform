//go:build !tei_repair_only

package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// The three signature diagnostics of §12 work item 1 must reach the product
// path, not only the syntax package. The Markdown loader is all-or-nothing, so
// each of them stops the whole source; this pins that at the Load boundary and
// that each keeps its own wording, so one problem is not reported as another.
func TestRuleAuthoringSignatureDiagnosticsReachLoad(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		want   string
		ruleID string
	}{
		{
			name: "undeclared input name",
			body: "namespace sales\noperator sales.op: takes low: number\n" +
				"rule sales.item.probe:\n  sales.item.amount is sales.op high: 100\n",
			want:   `declares no input named "high"`,
			ruleID: "sales.item.probe",
		},
		{
			name: "value of the wrong type",
			body: "namespace sales\noperator sales.op: takes low: number\n" +
				"rule sales.item.probe:\n  sales.item.amount is sales.op low: ITEM-A\n",
			want:   "takes a number",
			ruleID: "sales.item.probe",
		},
		{
			name:   "input read over all rows",
			body:   "namespace sales\noperator sales.op:\n  takes start: number of each row\n",
			want:   "no place for an input read over all rows",
			ruleID: "sales.op",
		},
		{
			name:   "refutation statement",
			body:   "namespace sales\noperator sales.op:\n  takes start: number\n  refutes on some rows\n",
			want:   "no place for a refutation statement",
			ruleID: "sales.op",
		},
		{
			name: "undeclared value-absent mapping",
			body: "namespace sales\noperator sales.op: takes end: number\n" +
				"rule sales.item.probe:\n  sales.item.amount is sales.op end: 1 or open\n",
			want:   `no value-absent mapping named "open"`,
			ruleID: "sales.item.probe",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeProbeWorkspaceFile(t, root, "tei.workspace.yaml",
				"workspace: probe\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n")
			writeProbeWorkspaceFile(t, root, "semantic/probe.tei.md",
				"```tei\n"+tc.body+"```\n")

			_, err := workspace.Load(root)
			if err == nil {
				t.Fatalf("%s was accepted at Load", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("the refusal does not say %q: %v", tc.want, err)
			}
			if !strings.Contains(err.Error(), tc.ruleID) {
				t.Fatalf("the refusal does not name the rule %q: %v", tc.ruleID, err)
			}
			// The diagnostic points into the source, not at the file start.
			if !strings.Contains(err.Error(), "probe.tei.md:") {
				t.Fatalf("the refusal carries no source position: %v", err)
			}
		})
	}
}

func writeProbeWorkspaceFile(t *testing.T, root, slashPath, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(slashPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
