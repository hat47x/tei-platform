package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A stub SHO document command. It prints one receipt and records the arguments
// it was given, so a test can assert the Studio passes the exact paths through
// without inventing a real checkout.
func writeStubShoDocument(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sho-doc.cjs")
	script := "process.stdout.write(" + jsonString(body) + ");\n"
	if err := os.WriteFile(path, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func jsonString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func studioShoDocumentStub(t *testing.T, commandPath string) studioShoDocument {
	return studioShoDocument{
		CommandPath:     commandPath,
		Timeout:         30 * time.Second,
		MaxReceiptBytes: 1 << 20,
	}
}

// TestStudioShoDocumentRelaysTheReceipt pins the happy path: the child's own
// result object reaches the caller unmodified, and the response names its own
// schema and authority limits.
func TestStudioShoDocumentRelaysTheReceipt(t *testing.T) {
	commandPath := writeStubShoDocument(t, `{"kind":"sho-draft-resume","status":"read","path":"/tmp/doc.md"}`)
	got := studioShoDocumentStub(t, commandPath).Run(StudioShoDocumentRequest{
		Operation: "resume", DocumentPath: filepath.Join(t.TempDir(), "doc.md"),
	})
	if !got.OK || got.Failure != nil {
		t.Fatalf("response = %+v failure=%+v", got, got.Failure)
	}
	if got.SchemaVersion != "tei.studio.sho-document/v0" {
		t.Fatalf("schemaVersion = %q", got.SchemaVersion)
	}
	var result struct {
		Kind   string `json:"kind"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(got.Result, &result); err != nil {
		t.Fatalf("result is not the child's object: %v", err)
	}
	if result.Kind != "sho-draft-resume" || result.Status != "read" {
		t.Fatalf("result = %+v", result)
	}
	if len(got.AuthorityLimits) == 0 {
		t.Fatal("authority limits missing")
	}
	for _, limit := range got.AuthorityLimits {
		if limit == "studio-save-is-not-tei-model-ingestion" {
			return
		}
	}
	t.Fatalf("authorityLimits = %v", got.AuthorityLimits)
}

// TestStudioShoDocumentOpenPassesBothPaths pins that `open` forwards the draft
// and the document, because a dropped argument would open the wrong document.
func TestStudioShoDocumentOpenPassesBothPaths(t *testing.T) {
	dir := t.TempDir()
	argvPath := filepath.Join(dir, "argv.json")
	commandPath := filepath.Join(dir, "sho-doc.cjs")
	script := "require('node:fs').writeFileSync(" + jsonString(argvPath) +
		", JSON.stringify(process.argv.slice(2)));\n" +
		"process.stdout.write(" + jsonString(`{"kind":"sho-draft-open"}`) + ");\n"
	if err := os.WriteFile(commandPath, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	draft := filepath.Join(dir, "draft.json")
	document := filepath.Join(dir, "doc.md")
	got := studioShoDocumentStub(t, commandPath).Run(StudioShoDocumentRequest{
		Operation: "open", DraftPath: draft, DocumentPath: document,
	})
	if !got.OK {
		t.Fatalf("response = %+v", got)
	}
	raw, err := os.ReadFile(argvPath)
	if err != nil {
		t.Fatal(err)
	}
	var argv []string
	if err := json.Unmarshal(raw, &argv); err != nil {
		t.Fatal(err)
	}
	if len(argv) != 3 || argv[0] != "open" || argv[1] != draft || argv[2] != document {
		t.Fatalf("argv = %v", argv)
	}
}

// TestStudioShoDocumentRefusesUnconfiguredAndRelativeCommands pins that the
// command is configuration, not discovery: an empty or relative path is refused
// instead of being resolved against the Studio's own directory.
func TestStudioShoDocumentRefusesUnconfiguredAndRelativeCommands(t *testing.T) {
	for name, component := range map[string]studioShoDocument{
		"unconfigured": {},
		"relative":     {CommandPath: "bin/sho-doc.cjs"},
	} {
		t.Run(name, func(t *testing.T) {
			got := component.Run(StudioShoDocumentRequest{
				Operation: "resume", DocumentPath: filepath.Join(t.TempDir(), "doc.md"),
			})
			if got.OK || got.Failure == nil {
				t.Fatalf("response = %+v", got)
			}
			if got.Failure.Code != ShoNotConfigured && got.Failure.Code != ShoInvalidArgument {
				t.Fatalf("code = %q", got.Failure.Code)
			}
		})
	}
}

// TestStudioShoDocumentRefusesSaveAndRelativePaths pins that the surface is
// read-only by construction: save is not offered, and a relative document path
// would resolve against an unpredictable working directory.
func TestStudioShoDocumentRefusesSaveAndRelativePaths(t *testing.T) {
	commandPath := writeStubShoDocument(t, `{"kind":"never"}`)
	component := studioShoDocumentStub(t, commandPath)
	for name, request := range map[string]StudioShoDocumentRequest{
		"save":              {Operation: "save", DocumentPath: filepath.Join(t.TempDir(), "doc.md")},
		"unknown operation": {Operation: "delete", DocumentPath: filepath.Join(t.TempDir(), "doc.md")},
		"relative document": {Operation: "resume", DocumentPath: "doc.md"},
		"relative draft":    {Operation: "open", DraftPath: "draft.json", DocumentPath: filepath.Join(t.TempDir(), "doc.md")},
	} {
		t.Run(name, func(t *testing.T) {
			got := component.Run(request)
			if got.OK || got.Failure == nil || got.Failure.Code != ShoInvalidArgument {
				t.Fatalf("response = %+v", got)
			}
		})
	}
}

// TestStudioShoDocumentReportsStableFailures pins the observable failure kinds:
// a non-zero exit and an unreadable receipt are two different answers, not one
// "no document".
func TestStudioShoDocumentReportsStableFailures(t *testing.T) {
	dir := t.TempDir()
	document := filepath.Join(dir, "doc.md")
	nonZero := filepath.Join(dir, "non-zero.cjs")
	if err := os.WriteFile(nonZero, []byte("process.exit(3);\n"), 0600); err != nil {
		t.Fatal(err)
	}
	unreadable := writeStubShoDocument(t, "not json at all")
	for name, tc := range map[string]struct {
		commandPath string
		wantCode    string
	}{
		"non-zero exit": {commandPath: nonZero, wantCode: ShoNonZeroExit},
		"unreadable":    {commandPath: unreadable, wantCode: ShoReceiptUnreadable},
	} {
		t.Run(name, func(t *testing.T) {
			got := studioShoDocumentStub(t, tc.commandPath).Run(
				StudioShoDocumentRequest{Operation: "resume", DocumentPath: document})
			if got.OK || got.Failure == nil || got.Failure.Code != tc.wantCode {
				t.Fatalf("response = %+v failure=%+v", got, got.Failure)
			}
			// A failure never carries a document state.
			if got.Result != nil {
				t.Fatalf("failure carried a result: %s", got.Result)
			}
		})
	}
}

// TestStudioShoDocumentRelaysTheWholeReceipt pins that the Studio does not
// unwrap or interpret the receipt: the child's object is the answer, so a
// receipt shape the Studio does not know still reaches the caller intact.
func TestStudioShoDocumentRelaysTheWholeReceipt(t *testing.T) {
	commandPath := writeStubShoDocument(t,
		`{"kind":"sho-draft-open","status":"created","claims":{"teiModelIngested":false}}`)
	got := studioShoDocumentStub(t, commandPath).Run(StudioShoDocumentRequest{
		Operation: "open", DraftPath: filepath.Join(t.TempDir(), "draft.json"),
		DocumentPath: filepath.Join(t.TempDir(), "doc.md"),
	})
	if !got.OK {
		t.Fatalf("response = %+v failure=%+v", got, got.Failure)
	}
	var receipt struct {
		Kind   string `json:"kind"`
		Status string `json:"status"`
		Claims struct {
			TeiModelIngested bool `json:"teiModelIngested"`
		} `json:"claims"`
	}
	if err := json.Unmarshal(got.Result, &receipt); err != nil {
		t.Fatalf("result is not the child's object: %v", err)
	}
	if receipt.Kind != "sho-draft-open" || receipt.Status != "created" {
		t.Fatalf("receipt = %+v", receipt)
	}
	// The claim ceiling survives the relay: the Studio does not promote it.
	if receipt.Claims.TeiModelIngested {
		t.Fatal("a save receipt claimed model ingestion")
	}
}

// TestStudioShoDocumentTimesOut pins that a hung child is killed and reported as
// a timeout, never read as an empty document.
func TestStudioShoDocumentTimesOut(t *testing.T) {
	dir := t.TempDir()
	commandPath := filepath.Join(dir, "hang.cjs")
	if err := os.WriteFile(commandPath, []byte("setTimeout(function(){}, 60000);\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got := studioShoDocument{
		CommandPath: commandPath,
		Timeout:     300 * time.Millisecond,
	}.Run(StudioShoDocumentRequest{Operation: "resume", DocumentPath: filepath.Join(dir, "doc.md")})
	if got.OK || got.Failure == nil || got.Failure.Code != ShoTimeout {
		t.Fatalf("response = %+v", got)
	}
}

// TestStudioShoDocumentFromEnvironmentIsConfigurationNotDiscovery pins that an
// unset variable answers `sho-not-configured` rather than searching the
// filesystem for something that might be the right CLI.
func TestStudioShoDocumentFromEnvironmentIsConfigurationNotDiscovery(t *testing.T) {
	original, had := os.LookupEnv("SHO_DOC_COMMAND")
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("SHO_DOC_COMMAND", original)
			return
		}
		_ = os.Unsetenv("SHO_DOC_COMMAND")
	})
	_ = os.Unsetenv("SHO_DOC_COMMAND")
	component := studioShoDocumentFromEnvironment()
	if component.CommandPath != "" {
		t.Fatalf("commandPath = %q", component.CommandPath)
	}
	got := component.Run(StudioShoDocumentRequest{
		Operation: "resume", DocumentPath: filepath.Join(t.TempDir(), "doc.md"),
	})
	if got.OK || got.Failure == nil || got.Failure.Code != ShoNotConfigured {
		t.Fatalf("response = %+v", got)
	}
}

// TestStudioShoDocumentResponseDoesNotLeakPaths pins that the relayed answer
// carries the child's result and the failure message only: no command path, no
// working directory, no environment.
func TestStudioShoDocumentResponseDoesNotLeakPaths(t *testing.T) {
	dir := t.TempDir()
	commandPath := writeStubShoDocument(t, `{"kind":"sho-draft-resume"}`)
	got := studioShoDocumentStub(t, commandPath).Run(StudioShoDocumentRequest{
		Operation: "resume", DocumentPath: filepath.Join(dir, "doc.md"),
	})
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), commandPath) {
		t.Fatalf("response leaked the command path: %s", encoded)
	}
	if strings.Contains(string(encoded), dir) {
		t.Fatalf("response leaked the working directory: %s", encoded)
	}
}
