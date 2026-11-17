package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestInvalidCanonicalIDFlagsFailWithoutAPanic pins that a mistyped identifier
// ends the command as an argument failure. The expected text is written here,
// not copied from the product: a caller must be told which flag is wrong and
// what a legal value looks like, and must not receive a goroutine dump.
//
// Before this change these commands called MustCanonicalID on the raw flag, so
// `-database sales-core` (a hyphen is illegal inside a canonical segment)
// printed a panic and a stack trace with exit code 2. A script that inspects
// the exit code could not tell a bad argument from a crash.
func TestInvalidCanonicalIDFlagsFailWithoutAPanic(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	cases := []struct {
		name  string
		args  []string
		flag  string
		value string
	}{
		{
			name:  "desired-schema database",
			args:  []string{"desired-schema", "-workspace", referenceWorkspaceRootForIDs(t), "-database", "sales-core", "-provider", "postgres"},
			flag:  "database",
			value: "sales-core",
		},
		{
			name:  "plan-migration database",
			args:  []string{"plan-migration", "-workspace", referenceWorkspaceRootForIDs(t), "-database", "sales-core", "-provider", "postgres", "-actual", "absent.json", "-version", "v1"},
			flag:  "database",
			value: "sales-core",
		},
		{
			name:  "migration-fingerprint database",
			args:  []string{"migration-fingerprint", "-workspace", referenceWorkspaceRootForIDs(t), "-database", "sales-core", "-version", "v1"},
			flag:  "database",
			value: "sales-core",
		},
		{
			name:  "database-reconcile database",
			args:  []string{"database-reconcile", "-workspace", referenceWorkspaceRootForIDs(t), "-database", "sales-core", "-provider", "postgres", "-actual", "absent.json", "-history", "absent.json"},
			flag:  "database",
			value: "sales-core",
		},
		{
			name:  "database-compatibility application",
			args:  []string{"database-compatibility", "-workspace", referenceWorkspaceRootForIDs(t), "-application", "order-entry"},
			flag:  "application",
			value: "order-entry",
		},
		{
			name:  "application-release-compatibility application",
			args:  []string{"application-release-compatibility", "-workspace", referenceWorkspaceRootForIDs(t), "-application", "order-entry", "-version", "v1"},
			flag:  "application",
			value: "order-entry",
		},
		{
			name:  "solution-release-graph solution",
			args:  []string{"solution-release-graph", "-workspace", referenceWorkspaceRootForIDs(t), "-solution", "my-solution", "-version", "v1"},
			flag:  "solution",
			value: "my-solution",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runCLIProcessWithStreams(t, binary, tc.args...)
			if got.code != 2 {
				t.Fatalf("exit=%d want 2 (an argument failure)", got.code)
			}
			if strings.Contains(got.stderr, "panic:") || strings.Contains(got.stderr, "goroutine ") {
				t.Fatalf("a mistyped identifier produced a stack trace:\n%s", got.stderr)
			}
			if got.stdout != "" {
				t.Fatalf("standard output must stay empty for an argument failure, got %q", got.stdout)
			}
			// The flag name is what the caller has to fix.
			if !strings.Contains(got.stderr, "-"+tc.flag) {
				t.Fatalf("the failure must name the offending flag -%s, got %q", tc.flag, got.stderr)
			}
			// The parser's rule is what makes the value fixable.
			if !strings.Contains(got.stderr, "canonical id") {
				t.Fatalf("the failure must repeat the identifier rule, got %q", got.stderr)
			}
			if strings.Contains(got.stderr, tc.value+"\":") {
				// The value is echoed, but it must not be the only information.
				if !strings.Contains(got.stderr, "segment") {
					t.Fatalf("the failure must explain the segment rule, got %q", got.stderr)
				}
			}
		})
	}
}

// TestValidCanonicalIDFlagsStillAnswer keeps the new check from rejecting the
// documented identifiers. Without this, "reject the invalid value" could be
// satisfied by rejecting every value.
func TestValidCanonicalIDFlagsStillAnswer(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{
			name: "desired-schema",
			args: []string{"desired-schema", "-workspace", referenceWorkspaceRootForIDs(t), "-database", "sales_core", "-provider", "postgres"},
		},
		{
			name: "database-compatibility",
			args: []string{"database-compatibility", "-workspace", referenceWorkspaceRootForIDs(t), "-application", "order_entry"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := runCLIProcessWithStreams(t, binary, tc.args...)
			if got.code != 0 {
				t.Fatalf("%s rejected a declared identifier: exit=%d stderr=%s", tc.name, got.code, got.stderr)
			}
			if strings.TrimSpace(got.stdout) == "" {
				t.Fatalf("%s answered nothing", tc.name)
			}
		})
	}
}

// referenceWorkspaceRootForIDs returns the reference Workspace as an absolute
// path, so the built binary receives the same directory the tests read.
func referenceWorkspaceRootForIDs(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
