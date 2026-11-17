package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"tei/internal/businessprocedure"
	"testing"
)

func businessProcedureFixturePath() string {
	return filepath.Join("..", "..", "reference", "business-system-design",
		"order-acceptance-business-procedure.json")
}

func runBusinessProcedure(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var out bytes.Buffer
	code, err := runBusinessProcedureCommand(args, &out)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return code, out.String()
}

// TestBusinessProcedureSchemaVersionIsAsserted pins the envelope version to the
// catalogue so a caller reading `tei commands` knows what it will receive.
func TestBusinessProcedureSchemaVersionIsAsserted(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	code, body := runBusinessProcedure(t,
		"-workspace", root, "-procedure", businessProcedureFixturePath())
	if code != 0 {
		t.Fatalf("code=%d body=%s", code, body)
	}
	assertSchemaVersionLiteral(t, []byte(body), "tei.cli.business-procedure/v0")
	assertAdvertisedVersion(t, "business-procedure", decodeSchemaVersion(t, body))
}

func decodeSchemaVersion(t *testing.T, body string) string {
	t.Helper()
	var envelope struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("not JSON: %v: %s", err, body)
	}
	return envelope.SchemaVersion
}

// TestBusinessProcedureProjectsTheExample runs the public example through the
// real Workspace and checks the five properties the issue asks to verify.
func TestBusinessProcedureProjectsTheExample(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	code, body := runBusinessProcedure(t,
		"-workspace", root, "-procedure", businessProcedureFixturePath())
	if code != 0 {
		t.Fatalf("code=%d body=%s", code, body)
	}

	var envelope struct {
		SchemaVersion string `json:"schemaVersion"`
		OK            bool   `json:"ok"`
		OKScope       string `json:"okScope"`
		Result        struct {
			Effect  string `json:"effect"`
			Trigger struct {
				Invocation bool `json:"invocation"`
			} `json:"trigger"`
			Actor struct {
				BusinessActorLabel    string `json:"businessActorLabel"`
				AuthorizationRoleCode string `json:"authorizationRoleCode"`
			} `json:"actor"`
			Steps []struct {
				ID                string `json:"id"`
				Realization       string `json:"realization"`
				Manual            bool   `json:"manual"`
				AutoExecuted      bool   `json:"autoExecuted"`
				NoFunction        bool   `json:"noFunction"`
				ManualInstruction string `json:"manualInstruction"`
				Function          *struct {
					ID       string `json:"id"`
					Subject  string `json:"subject"`
					Artifact string `json:"artifact"`
					Line     int    `json:"line"`
				} `json:"function"`
			} `json:"steps"`
			Branches []struct {
				ID      string `json:"id"`
				Outcome string `json:"outcome"`
			} `json:"branches"`
			ManualSteps []string   `json:"manualSteps"`
			Diagnostics []struct{} `json:"diagnostics"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("decode: %v: %s", err, body)
	}
	if !envelope.OK || envelope.OKScope != "projection-produced-only" {
		t.Fatalf("ok=%v okScope=%q; the projection must not claim more", envelope.OK, envelope.OKScope)
	}
	if envelope.Result.Effect != "read-only" {
		t.Fatalf("effect=%q", envelope.Result.Effect)
	}
	if envelope.Result.Trigger.Invocation {
		t.Fatal("the trigger was turned into an invocation")
	}
	if envelope.Result.Actor.BusinessActorLabel == envelope.Result.Actor.AuthorizationRoleCode {
		t.Fatal("the business actor equals the authorization role code")
	}

	// 1. the manual phone confirmation did not disappear.
	if len(envelope.Result.ManualSteps) != 3 {
		t.Fatalf("manualSteps=%v, want three", envelope.Result.ManualSteps)
	}
	var phoneConfirmed bool
	for _, step := range envelope.Result.Steps {
		if step.Manual {
			if step.AutoExecuted {
				t.Fatalf("manual step %s was auto-executed", step.ID)
			}
			if step.ManualInstruction == "" {
				t.Fatalf("manual step %s lost its instruction", step.ID)
			}
		}
		if step.ID == "BP-A02" {
			phoneConfirmed = step.Manual && step.NoFunction && step.Function == nil
		}
		// 2. every resolved Function reaches one target and its Source.
		if step.Function != nil {
			if step.Function.Subject == "" || step.Function.Artifact == "" || step.Function.Line < 1 {
				t.Fatalf("step %s resolved without a target and source: %+v", step.ID, step.Function)
			}
		}
	}
	if !phoneConfirmed {
		t.Fatal("the phone confirmation is missing or was given a Function")
	}

	// 3. approval and rejection are separate branches.
	if len(envelope.Result.Branches) != 2 {
		t.Fatalf("branches=%+v, want two", envelope.Result.Branches)
	}
	if envelope.Result.Branches[0].ID == envelope.Result.Branches[1].ID {
		t.Fatal("the two branches share one identity")
	}
	outcomes := map[string]bool{}
	for _, branch := range envelope.Result.Branches {
		outcomes[branch.Outcome] = true
	}
	if !outcomes["approval"] || !outcomes["rejection"] {
		t.Fatalf("outcomes=%v, want both approval and rejection", outcomes)
	}

	// 4. the example projects without a finding, and the field is a list.
	if envelope.Result.Diagnostics == nil {
		t.Fatal("diagnostics is null; a reader cannot tell 'none' from 'absent'")
	}
	if len(envelope.Result.Diagnostics) != 0 {
		t.Fatalf("the example produced findings: %s", body)
	}
}

// The command must refuse input it cannot honour rather than project something
// it guessed at.
func TestBusinessProcedureRejectsIncompleteInput(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")

	for _, tc := range []struct {
		name string
		args []string
		code string
	}{
		{name: "no procedure", args: []string{"-workspace", root}, code: "INVALID_ARGUMENT"},
		{name: "no workspace", args: []string{"-procedure", businessProcedureFixturePath()}, code: "INVALID_ARGUMENT"},
		{name: "positional", args: []string{"-workspace", root, "-procedure", businessProcedureFixturePath(), "extra"}, code: "INVALID_ARGUMENT"},
		{name: "missing file", args: []string{"-workspace", root, "-procedure", filepath.Join(t.TempDir(), "missing.json")}, code: "PROCEDURE_LOAD_FAILED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, body := runBusinessProcedure(t, tc.args...)
			if code == 0 {
				t.Fatalf("exit 0 for %v: %s", tc.args, body)
			}
			var envelope struct {
				OK    bool `json:"ok"`
				Error *struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(body), &envelope); err != nil {
				t.Fatalf("not JSON: %v: %s", err, body)
			}
			if envelope.OK || envelope.Error == nil || envelope.Error.Code != tc.code {
				t.Fatalf("error=%+v, want %s: %s", envelope.Error, tc.code, body)
			}
		})
	}
}

// A document that declares another schema or another artifact role is refused
// instead of read as if it were this one. A misspelled member is refused too:
// dropping it would let `manualInsturction` read as a manual step with no
// instruction, which is the one thing the projection must not do.
func TestBusinessProcedureRejectsUnexpectedDocument(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	dir := t.TempDir()

	for _, tc := range []struct {
		name string
		body string
		code string
	}{
		{
			name: "schema",
			body: `{"schemaVersion":"business-procedure-model-reference-v1","artifactRole":"reference-authoring-fixture"}`,
			code: "PROCEDURE_SCHEMA_UNSUPPORTED",
		},
		{
			name: "role",
			body: `{"schemaVersion":"business-procedure-model-reference-v0","artifactRole":"canonical"}`,
			code: "PROCEDURE_ARTIFACT_ROLE_UNSUPPORTED",
		},
		{
			name: "unknown member",
			body: `{"schemaVersion":"business-procedure-model-reference-v0","artifactRole":"reference-authoring-fixture",` +
				`"procedure":{"id":"p"},"steps":[{"id":"BP-X1","action":"a","realization":"manual","manualInsturction":"typo"}]}`,
			code: "PROCEDURE_LOAD_FAILED",
		},
		{
			name: "second document",
			body: `{"schemaVersion":"business-procedure-model-reference-v0","artifactRole":"reference-authoring-fixture"}` +
				`{"schemaVersion":"business-procedure-model-reference-v0","artifactRole":"reference-authoring-fixture"}`,
			code: "PROCEDURE_LOAD_FAILED",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".json")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			code, out := runBusinessProcedure(t, "-workspace", root, "-procedure", path)
			if code == 0 {
				t.Fatalf("exit 0: %s", out)
			}
			if !strings.Contains(out, tc.code) {
				t.Fatalf("body=%s, want %s", out, tc.code)
			}
		})
	}
}

// An identity the Workspace does not declare is reported, not filled in.
func TestBusinessProcedureReportsUnresolvedReference(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	raw, err := os.ReadFile(businessProcedureFixturePath())
	if err != nil {
		t.Fatal(err)
	}
	// Point the submit step at an identity that does not exist.
	mutated := strings.Replace(string(raw),
		`"id": "order_entry.submit_order"`,
		`"id": "order_entry.submit_order_missing"`, 1)
	dir := t.TempDir()
	path := filepath.Join(dir, "procedure.json")
	if err := os.WriteFile(path, []byte(mutated), 0o600); err != nil {
		t.Fatal(err)
	}

	code, body := runBusinessProcedure(t, "-workspace", root, "-procedure", path)
	if code != 0 {
		t.Fatalf("the projection itself failed: %s", body)
	}
	if !strings.Contains(body, "BUSINESS_REFERENCE_UNRESOLVED") {
		t.Fatalf("the unresolved reference was not reported: %s", body)
	}
	// The raw reference must survive in the projection.
	if !strings.Contains(body, "order_entry.submit_order_missing") {
		t.Fatalf("the unresolved reference was dropped from the projection: %s", body)
	}
}

// Help must describe the command's own grammar and start the documented way.
func TestBusinessProcedureHelp(t *testing.T) {
	if !isAcceptedCommand("business-procedure") {
		t.Fatal("business-procedure is not an accepted command")
	}
	var fsHelp bytes.Buffer
	code, err := runBusinessProcedureCommand([]string{"-h"}, &fsHelp)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	text := fsHelp.String()
	if !strings.HasPrefix(text, "usage: tei business-procedure") {
		t.Fatalf("help does not start with the usage line: %q", text)
	}
	for _, flagName := range []string{"-workspace", "-procedure"} {
		if !strings.Contains(text, flagName) {
			t.Fatalf("help omits %s: %s", flagName, text)
		}
	}
}

// This command parses its own FlagSet, so the shared repeatable-flag pass does
// not run for it. Without the guard a repeated single-value flag silently keeps
// the last value, which is the mistake the other self-parsed commands refuse.
func TestBusinessProcedureRefusesARepeatedFlag(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	code, body := runBusinessProcedure(t,
		"-workspace", root,
		"-procedure", businessProcedureFixturePath(),
		"-procedure", businessProcedureFixturePath())
	if code == 0 {
		t.Fatalf("exit 0 for a repeated -procedure: %s", body)
	}
	if !strings.Contains(body, "INVALID_ARGUMENT") || !strings.Contains(body, "procedure") {
		t.Fatalf("a repeated flag was not refused as an argument error: %s", body)
	}
}

// The envelope names the Workspace whose Source Index resolved the references,
// so a reader can tell which Workspace a projection came from.
func TestBusinessProcedureReportsTheWorkspaceIdentity(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	code, body := runBusinessProcedure(t,
		"-workspace", root, "-procedure", businessProcedureFixturePath())
	if code != 0 {
		t.Fatalf("code=%d body=%s", code, body)
	}
	var envelope struct {
		Workspace *struct {
			Name     string `json:"name"`
			Manifest struct {
				Path string `json:"path"`
			} `json:"manifest"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("decode: %v: %s", err, body)
	}
	if envelope.Workspace == nil {
		t.Fatal("the envelope does not name the Workspace")
	}
	if envelope.Workspace.Name != "tei_reference" {
		t.Fatalf("workspace name=%q, want tei_reference", envelope.Workspace.Name)
	}
	if envelope.Workspace.Manifest.Path != "tei.workspace.yaml" {
		t.Fatalf("manifest path=%q", envelope.Workspace.Manifest.Path)
	}
}

// The contract row promises the inner result schema. Asserting it here keeps the
// advertised inner version from drifting away from what the package writes.
func TestBusinessProcedureResultSchemaVersionIsAsserted(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	code, body := runBusinessProcedure(t,
		"-workspace", root, "-procedure", businessProcedureFixturePath())
	if code != 0 {
		t.Fatalf("code=%d body=%s", code, body)
	}
	var envelope struct {
		Result struct {
			SchemaVersion string `json:"schemaVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("decode: %v: %s", err, body)
	}
	if envelope.Result.SchemaVersion != businessprocedure.ProjectionSchemaVersion {
		t.Fatalf("inner schemaVersion=%q, want %q",
			envelope.Result.SchemaVersion, businessprocedure.ProjectionSchemaVersion)
	}
}
