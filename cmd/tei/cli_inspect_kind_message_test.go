package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/operations"
)

// `inspect` accepts a closed set of subject kinds that the catalogue publishes.
// An unknown kind is refused with UNSUPPORTED_SUBJECT_KIND, but the message
// used to name only the bad value: a caller learned that it was wrong and
// nothing about what to write instead. `model search` and `trace` already list
// their accepted values (#3242).
//
// The expected list is read from operations.InspectionSubjectKinds, not copied,
// so the advertised set and the message cannot drift apart.
func TestInspectUnsupportedKindNamesTheAcceptedKinds(t *testing.T) {
	binary := buildCLIForInspectKindTest(t)
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	var out bytes.Buffer
	command := exec.Command(binary, "inspect", "-workspace", reference,
		"-subject-kind", "not-a-kind", "-subject-id", "sales.order")
	command.Stdout = &out
	if err := command.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 {
			t.Fatalf("inspect: %v (%q)", err, out.String())
		}
	}

	var body struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatalf("not one JSON document: %v (%q)", err, out.String())
	}
	if body.Error == nil || body.Error.Code != operations.InspectionUnsupportedKind {
		t.Fatalf("error=%+v, want code %s", body.Error, operations.InspectionUnsupportedKind)
	}
	// The value the caller used is named, so the message is about this call.
	if !strings.Contains(body.Error.Message, "not-a-kind") {
		t.Fatalf("message does not name the bad value: %q", body.Error.Message)
	}
	// Every accepted kind is named, so the caller can correct the value
	// without reading the source.
	for _, kind := range operations.InspectionSubjectKinds {
		if !strings.Contains(body.Error.Message, kind) {
			t.Fatalf("message omits accepted kind %q: %q", kind, body.Error.Message)
		}
	}
}

// A supported kind does not get this message at all: the guard is about the
// value, not about refusing inspection.
func TestInspectSupportedKindIsNotRefusedAsUnsupported(t *testing.T) {
	binary := buildCLIForInspectKindTest(t)
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	var out bytes.Buffer
	command := exec.Command(binary, "inspect", "-workspace", reference,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order")
	command.Stdout = &out
	// A supported kind reaches the World and answers; its exit code is 0 when
	// the subject exists.
	if err := command.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 2 &&
			strings.Contains(out.String(), operations.InspectionUnsupportedKind) {
			t.Fatalf("a published kind was refused as unsupported: %s", out.String())
		}
	}
}

func buildCLIForInspectKindTest(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	return binary
}
