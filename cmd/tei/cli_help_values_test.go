package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"tei/internal/knowledgecontext"
	"tei/internal/modelref"
	"tei/internal/operations"
)

// TestHelpShowsTheSameArgumentValuesAsDiscovery pins the other half of #3242:
// the values a caller needs are now in `tei commands`, and `-h` must not
// restate them by hand. Both come from the same variables, so a change shows
// up in both places.
func TestHelpShowsTheSameArgumentValuesAsDiscovery(t *testing.T) {
	binary := buildCLIForLifecycle(t)

	run := func(args ...string) string {
		t.Helper()
		command := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		if err := command.Run(); err != nil {
			t.Fatalf("%q: %v (stdout=%q stderr=%q)", args, err, stdout.String(), stderr.String())
		}
		if stderr.Len() > 0 {
			t.Fatalf("%q wrote stderr=%q", args, stderr.String())
		}
		return stdout.String()
	}

	for _, tc := range []struct {
		command string
		flag    string
		values  []string
		def     string
	}{
		{"inspect", "subject-kind", operations.InspectionSubjectKinds, ""},
		{"trace", "subject-kind", operations.TraceSubjectKinds, ""},
		{"trace", "direction", modelref.TraceDirections, "both"},
		{"trace", "mode", modelref.TraceModes, "summary"},
		{"view", "format", []string{"text", "json", "html", "print"}, "text"},
		{"knowledge-context", "disclose", knowledgecontext.SelectableDisclosureSections, ""},
	} {
		t.Run(tc.command+" -"+tc.flag, func(t *testing.T) {
			text := run(tc.command, "-h")
			values := strings.Join(tc.values, ", ")
			if !strings.Contains(text, "values: "+values) {
				t.Fatalf("help for %s -%s omits the value set %q:\n%s", tc.command, tc.flag, values, text)
			}
			if tc.def != "" && !strings.Contains(text, "default: "+tc.def) {
				t.Fatalf("help for %s -%s omits default %q:\n%s", tc.command, tc.flag, tc.def, text)
			}
		})
	}

	// A flag without a closed set is still described, and gets no values line:
	// the absence has to be meaningful, not accidental.
	text := run("inspect", "-h")
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "values:") &&
			strings.Contains(text, "-workspace") && !strings.Contains(text, "-subject-kind") {
			t.Fatalf("unexpected values line for a command without sets: %q", line)
		}
	}
	if strings.Contains(text, "values: text") {
		t.Fatalf("inspect -h printed a value set it does not have:\n%s", text)
	}

	// view's two HTML formats are legitimate for a pattern, not for an
	// application alone; help lists the set without promising every
	// combination works.
	viewHelp := run("view", "-h")
	if !strings.Contains(viewHelp, "values: text, json, html, print") {
		t.Fatalf("view -h omits formats:\n%s", viewHelp)
	}
	if !strings.Contains(viewHelp, "default: text") {
		t.Fatalf("view -h omits the default format:\n%s", viewHelp)
	}

}
