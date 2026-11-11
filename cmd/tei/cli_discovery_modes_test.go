package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// discoveryRun runs the built CLI and returns its stdout, failing the test on
// stderr or a non-envelope body so a mode observation cannot come from a
// diagnostic path.
func discoveryRun(t *testing.T, binary string, args ...string) []byte {
	t.Helper()
	command := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("%q: CLI execution: %v", args, err)
		}
	}
	if stderr.Len() > 0 {
		t.Fatalf("%q: enveloped command wrote stderr: %q", args, stderr.String())
	}
	var response struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("%q: stdout is not one JSON envelope: %v (%q)", args, err, stdout.String())
	}
	return stdout.Bytes()
}

// discoverySchemaVersion returns the top-level schemaVersion of an envelope.
func discoverySchemaVersion(t *testing.T, body []byte) string {
	t.Helper()
	var response struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return response.SchemaVersion
}

// advertisedSchemaVersion returns the schemaVersion the catalog advertises for a
// command, or "" when the command advertises none.
func advertisedSchemaVersion(t *testing.T, name string) string {
	t.Helper()
	for _, record := range commandCatalog().Commands {
		if record.Name == name {
			return record.SchemaVersion
		}
	}
	t.Fatalf("%s is not advertised by tei commands", name)
	return ""
}

// discoveryMode returns the top-level `mode` of an envelope, and whether the
// field was present at all.
func discoveryMode(t *testing.T, body []byte) (string, bool) {
	t.Helper()
	var response map[string]json.RawMessage
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	raw, ok := response["mode"]
	if !ok {
		return "", false
	}
	var mode string
	if err := json.Unmarshal(raw, &mode); err != nil {
		t.Fatalf("decode mode: %v", err)
	}
	return mode, true
}

// TestDiscoveryAdvertisesEveryModeAResponseCanCarry pins the `modes` field of
// the discovery catalogue against real responses. `mode` changes the shape of
// a result, so a caller that branches on it needs the closed set before it
// runs anything: a newly added mode would otherwise only surface as a runtime
// misparse. The cases below exercise the invocations that select a mode.
func TestDiscoveryAdvertisesEveryModeAResponseCanCarry(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	reference := filepath.Join("..", "..", "reference", "workspace")
	dir := t.TempDir()

	// knowledge-review-handoff and -disclose need real files, and the
	// candidate review needs a fingerprint from the same subject's context.
	requirementsPath := filepath.Join(dir, "requirements.json")
	if err := os.WriteFile(requirementsPath, []byte(`{"expectedKind":"semantic-data"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	handoffBody := discoveryRun(t, binary, "knowledge-review-handoff", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order",
		"-qualification-requirements", requirementsPath)
	var handoffEnvelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(handoffBody, &handoffEnvelope); err != nil || len(handoffEnvelope.Result) == 0 {
		t.Fatalf("knowledge-review-handoff has no result: %v (%s)", err, handoffBody)
	}
	handoffPath := filepath.Join(dir, "handoff.json")
	if err := os.WriteFile(handoffPath, handoffEnvelope.Result, 0o644); err != nil {
		t.Fatal(err)
	}

	// rea-evidence reads saved files rather than a Workspace, so its inputs
	// are written here: one saved analysis output, and one saved view
	// delivery in the framing of the declared version.
	reaEvidencePath := filepath.Join(dir, "rea-analysis.json")
	if err := os.WriteFile(reaEvidencePath, []byte(syntheticEvidence), 0o600); err != nil {
		t.Fatal(err)
	}
	reaViewPath := filepath.Join(dir, "rea-view.jsonl")
	if err := os.WriteFile(reaViewPath, []byte(syntheticView), 0o600); err != nil {
		t.Fatal(err)
	}

	lineContext := discoveryRun(t, binary, "knowledge-context", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order_line")
	var lineShape struct {
		Result struct {
			DataDesign *struct {
				FunctionalDependencyCandidates []struct {
					Fingerprint string `json:"fingerprint"`
				} `json:"functionalDependencyCandidates"`
			} `json:"dataDesign"`
		} `json:"result"`
	}
	if err := json.Unmarshal(lineContext, &lineShape); err != nil {
		t.Fatalf("decode knowledge-context: %v", err)
	}
	if lineShape.Result.DataDesign == nil || len(lineShape.Result.DataDesign.FunctionalDependencyCandidates) == 0 {
		t.Fatal("reference workspace lost its FD candidate for the discovery mode test")
	}
	candidateFingerprint := lineShape.Result.DataDesign.FunctionalDependencyCandidates[0].Fingerprint

	cases := []struct {
		args []string
		// want is the mode the response must carry. The second result reports
		// whether the field must be absent instead.
		want       string
		mustAbsent bool
	}{
		{args: []string{"source-index", "-workspace", reference}, want: resultModeIndex},
		{args: []string{"source-index", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order"}, want: resultModeSubject},
		{args: []string{"knowledge-context", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order"}, want: resultModeContext},
		{args: []string{"knowledge-context", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order", "-disclose", "source"}, want: resultModeDisclosure},
		{args: []string{"knowledge-qualify", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order", "-qualification-requirements", requirementsPath}, want: resultModeQualification},
		{args: []string{"knowledge-review-plan", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order"}, want: resultModePlan},
		{args: []string{"knowledge-review-handoff", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order", "-qualification-requirements", requirementsPath}, want: resultModeHandoff},
		{args: []string{"knowledge-review-disclose", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order", "-review-handoff", handoffPath}, want: resultModeDisclosure},
		{args: []string{"knowledge-data-design-candidate-review", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order_line", "-data-design-candidate-kind", "functional-dependency", "-data-design-candidate-fingerprint", candidateFingerprint}, want: resultModeReview},
		{args: []string{"plugin-lock", "-workspace", reference}, mustAbsent: true},
		{args: []string{"plugin-lock", "-workspace", reference, "-environment", "development"}, want: "environment"},
		{args: []string{"plugin-resolve", "-workspace", reference, "-environment", "development"}, mustAbsent: true},
		{args: []string{"plugin-resolve", "-workspace", reference, "-environment", "development", "-diagnostic-candidates"}, want: "unvalidated-candidates"},
		{args: []string{"requirements", "-workspace", reference}, mustAbsent: true},
		// -tool-path names a file that exists, so the receipt is a whole one
		// rather than one from an environment without the tool.
		{args: []string{"rea-evidence", "-analysis-output", reaEvidencePath, "-tool-path", reaEvidencePath}, want: "evidence"},
		{args: []string{"rea-evidence", "-analysis-output", reaEvidencePath, "-request-out", filepath.Join(dir, "rea-request.json"), "-provider-version", "6.8.0"}, want: "request"},
		{args: []string{"rea-evidence", "-analysis-output", reaEvidencePath, "-view", reaViewPath, "-provider-version", "6.8.0"}, want: "view"},
		{args: []string{"rea-evidence", "-analysis-output", reaEvidencePath, "-view", reaViewPath, "-request-out", filepath.Join(dir, "rea-request-view.json"), "-provider-version", "6.8.0"}, want: "request-view"},
	}

	advertised := map[string][]string{}
	for _, record := range commandCatalog().Commands {
		advertised[record.Name] = record.Modes
	}

	for _, tc := range cases {
		body := discoveryRun(t, binary, tc.args...)
		// The response must carry the schemaVersion the catalog advertises, so
		// advancing a command's schema without changing the response (or the
		// reverse) fails here rather than only in the discovery comparison that
		// covers a subset of commands (#4032).
		if want := advertisedSchemaVersion(t, tc.args[0]); want != "" {
			if got := discoverySchemaVersion(t, body); got != want {
				t.Fatalf("%q: schemaVersion=%q want %q", tc.args, got, want)
			}
		}
		mode, present := discoveryMode(t, body)
		if tc.mustAbsent {
			if present {
				t.Fatalf("%q: mode=%q, but this call has one output shape", tc.args, mode)
			}
			if len(advertised[tc.args[0]]) > 0 && !containsMode(advertised[tc.args[0]], "") {
				t.Fatalf("%s advertises modes %v but this call returns none", tc.args[0], advertised[tc.args[0]])
			}
			continue
		}
		if !present || mode != tc.want {
			t.Fatalf("%q: mode=%q present=%t want %q", tc.args, mode, present, tc.want)
		}
		if !containsMode(advertised[tc.args[0]], tc.want) {
			t.Fatalf("%s returned mode %q, which discovery does not advertise (%v)", tc.args[0], tc.want, advertised[tc.args[0]])
		}
	}
}

// TestDiscoveryModesAreAClosedAdvertisedSet rejects a response mode the
// catalogue does not list for any command, including an unknown mode from the
// discovery record itself: the list must stay small and named.
func TestDiscoveryModesAreAClosedAdvertisedSet(t *testing.T) {
	known := map[string]bool{
		resultModeIndex: true, resultModeSubject: true, resultModeContext: true,
		resultModeDisclosure: true, resultModeQualification: true, resultModePlan: true,
		resultModeHandoff: true, resultModeReview: true, "environment": true,
		"unvalidated-candidates": true, "": true,
		// rea-evidence names which optional inputs the response carries; the
		// shape really does change with them (a receipt alone, plus a request
		// check, plus a view inspection).
		"evidence": true, "request": true, "view": true, "request-view": true,
	}
	for _, record := range commandCatalog().Commands {
		for _, mode := range record.Modes {
			if !known[mode] {
				t.Errorf("%s advertises an unnamed mode %q", record.Name, mode)
			}
		}
	}
}

func containsMode(modes []string, want string) bool {
	for _, mode := range modes {
		if mode == want {
			return true
		}
	}
	return false
}
