package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"tei/internal/knowledgecontext"
	"tei/internal/modelref"
	"tei/internal/operations"
)

// TestDiscoveryArgumentValuesMatchImplementation pins #3242: the closed value
// sets a caller reads from `tei commands` must be the sets the command
// validates. Each case runs the command with every advertised value, then with
// one value outside the set, so an advertised value that the code rejects or an
// accepted value that discovery omits both fail here.
func TestDiscoveryArgumentValuesMatchImplementation(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	reference := filepath.Join("..", "..", "reference", "workspace")

	run := func(args ...string) (int, []byte, string) {
		t.Helper()
		command := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("%q: CLI execution: %v", args, err)
			}
			code = exit.ExitCode()
		}
		return code, stdout.Bytes(), stderr.String()
	}

	envelopeCode := func(t *testing.T, body []byte) (bool, string) {
		t.Helper()
		var response struct {
			OK    bool `json:"ok"`
			Error *struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("not one envelope: %v (%s)", err, body)
		}
		if response.Error == nil {
			return response.OK, ""
		}
		return response.OK, response.Error.Code
	}

	// inspect: every advertised kind must be resolved, not rejected as unsupported.
	for _, kind := range operations.InspectionSubjectKinds {
		code, body, stderr := run("inspect", "-workspace", reference,
			"-subject-kind", kind, "-subject-id", "sales.order")
		if stderr != "" {
			t.Fatalf("inspect %s wrote stderr=%q", kind, stderr)
		}
		_, failure := envelopeCode(t, body)
		if failure == "UNSUPPORTED_SUBJECT_KIND" {
			t.Fatalf("inspect advertises %s but rejects it", kind)
		}
		if code != 0 && code != 1 {
			t.Fatalf("inspect %s: exit=%d body=%s", kind, code, body)
		}
	}

	// A value outside the set must be rejected as unsupported, so the set is
	// demonstrably checked and not merely advertised.
	code, body, _ := run("inspect", "-workspace", reference,
		"-subject-kind", "not-a-kind", "-subject-id", "sales.order")
	if _, failure := envelopeCode(t, body); failure != "UNSUPPORTED_SUBJECT_KIND" || code != 2 {
		t.Fatalf("inspect accepted a kind outside the set: exit=%d body=%s", code, body)
	}

	// trace: kinds, direction and mode.
	for _, kind := range operations.TraceSubjectKinds {
		code, body, _ := run("trace", "-workspace", reference, "-scope", "test",
			"-subject-kind", kind, "-subject-id", "sales.order", "-mode", "summary")
		_, failure := envelopeCode(t, body)
		if failure == "UNSUPPORTED_SUBJECT_KIND" || failure == "INVALID_ARGUMENT" {
			t.Fatalf("trace advertises %s but rejects it as an argument: exit=%d body=%s", kind, code, body)
		}
		if code != 0 && code != 1 {
			t.Fatalf("trace %s: exit=%d body=%s", kind, code, body)
		}
	}
	for _, direction := range modelref.TraceDirections {
		code, body, _ := run("trace", "-workspace", reference, "-scope", "test",
			"-subject-kind", "semantic-relation", "-subject-id", "sales.order.customer",
			"-direction", direction, "-mode", "summary")
		if code != 0 {
			t.Fatalf("trace -direction %s: exit=%d body=%s", direction, code, body)
		}
	}
	for _, mode := range modelref.TraceModes {
		code, body, _ := run("trace", "-workspace", reference, "-scope", "test",
			"-subject-kind", "semantic-relation", "-subject-id", "sales.order.customer",
			"-mode", mode)
		if code != 0 {
			t.Fatalf("trace -mode %s: exit=%d body=%s", mode, code, body)
		}
	}
	for _, bad := range [][]string{
		{"trace", "-workspace", reference, "-scope", "test", "-subject-kind", "semantic-data",
			"-subject-id", "sales.order", "-direction", "sideways"},
		{"trace", "-workspace", reference, "-scope", "test", "-subject-kind", "semantic-data",
			"-subject-id", "sales.order", "-mode", "partial"},
	} {
		code, body, _ := run(bad...)
		if _, failure := envelopeCode(t, body); failure != "INVALID_ARGUMENT" || code != 2 {
			t.Fatalf("%q accepted a value outside the set: exit=%d body=%s", bad, code, body)
		}
	}

	// view: every advertised format is accepted for the application view.
	for _, format := range []string{"text", "json"} {
		code, _, _ := run("view", "-workspace", reference, "-application", "order_entry", "-format", format)
		if code != 0 {
			t.Fatalf("view -format %s: exit=%d", format, code)
		}
	}
	for _, format := range []string{"html", "print"} {
		code, _, _ := run("view", "-workspace", reference, "-application", "order_entry",
			"-pattern", "order_entry.order_lines", "-format", format)
		if code != 0 {
			t.Fatalf("view -pattern -format %s: exit=%d", format, code)
		}
	}
	code, body, _ = run("view", "-workspace", reference, "-application", "order_entry", "-format", "yaml")
	if _, failure := envelopeCode(t, body); failure != "INVALID_ARGUMENT" || code != 2 {
		t.Fatalf("view accepted a format outside the set: exit=%d body=%s", code, body)
	}

	// knowledge-context -disclose.
	for _, section := range knowledgecontext.SelectableDisclosureSections {
		code, _, _ := run("knowledge-context", "-workspace", reference,
			"-subject-kind", "semantic-data", "-subject-id", "sales.order", "-disclose", section)
		if code != 0 {
			t.Fatalf("knowledge-context -disclose %s: exit=%d", section, code)
		}
	}
	code, body, _ = run("knowledge-context", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order", "-disclose", "not-a-section")
	if _, failure := envelopeCode(t, body); failure != "KNOWLEDGE_BUILD_FAILED" || code != 2 {
		t.Fatalf("knowledge-context accepted a section outside the set: exit=%d body=%s", code, body)
	}

	// knowledge-data-design-candidate-review -data-design-candidate-kind is
	// checked against the review that resolves it; an unknown kind fails.
	code, body, _ = run("knowledge-data-design-candidate-review", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order_line",
		"-data-design-candidate-kind", "not-a-kind", "-data-design-candidate-fingerprint", "sha256:00")
	if _, failure := envelopeCode(t, body); failure == "" {
		t.Fatalf("candidate review accepted a kind outside the set: exit=%d body=%s", code, body)
	}

	// The catalogue and the implementation name the same sets.
	catalog := commandCatalog()
	sets := map[string]map[string][]string{}
	defaults := map[string]map[string]string{}
	for _, record := range catalog.Commands {
		sets[record.Name] = record.ArgumentValues
		defaults[record.Name] = record.ArgumentDefaults
	}
	for name, want := range map[string]map[string][]string{
		"inspect": {"subject-kind": operations.InspectionSubjectKinds},
		"trace": {
			"subject-kind": operations.TraceSubjectKinds,
			"direction":    modelref.TraceDirections,
			"mode":         modelref.TraceModes,
		},
		"view":              {"format": []string{"text", "json", "html", "print"}},
		"knowledge-context": {"disclose": knowledgecontext.SelectableDisclosureSections},
	} {
		got, ok := sets[name]
		if !ok || len(got) != len(want) {
			t.Fatalf("%s advertises %v, want %v", name, got, want)
		}
		for flag, values := range want {
			if len(got[flag]) != len(values) {
				t.Fatalf("%s -%s advertises %v, want %v", name, flag, got[flag], values)
			}
			for i := range values {
				if got[flag][i] != values[i] {
					t.Fatalf("%s -%s advertises %v, want %v", name, flag, got[flag], values)
				}
			}
		}
	}
	if defaults["trace"]["direction"] != "both" || defaults["trace"]["mode"] != "summary" {
		t.Fatalf("trace defaults=%v", defaults["trace"])
	}
	if defaults["view"]["format"] != "text" {
		t.Fatalf("view defaults=%v", defaults["view"])
	}
}
