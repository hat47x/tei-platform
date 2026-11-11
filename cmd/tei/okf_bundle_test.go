package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// okfFixturePath is the K1 synthetic bundle. The tests read it in place; a test that
// needs a broken shape copies it into a temporary directory first, because the
// repository's Markdown link check would reject an intentionally broken link in a
// managed file.
const okfFixturePath = "../../testdata/okf-k1-v02"

// decodeOKFEnvelope runs the command and returns the decoded envelope.
func decodeOKFEnvelope(t *testing.T, out bytes.Buffer) (map[string]any, int) {
	t.Helper()
	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not one JSON envelope: %v (%q)", err, out.String())
	}
	return envelope, 0
}

// TestOKFBundleCLIReadsTheFixture covers the first development goal: a real OKF v0.2
// bundle read through the existing CLI, with the concept list, sources, revision,
// unresolved references and format diagnostics available as machine-readable data.
func TestOKFBundleCLIReadsTheFixture(t *testing.T) {
	var out bytes.Buffer
	code, err := okfBundleJSON(&out, okfFixturePath, "", "", "", "", 0, 0, 0, 0)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
	}
	envelope, _ := decodeOKFEnvelope(t, out)

	if got := envelope["schemaVersion"]; got != "tei.cli.okf-bundle/v0" {
		t.Fatalf("schemaVersion = %v", got)
	}
	if got := envelope["ok"]; got != true {
		t.Fatalf("ok = %v", got)
	}
	if got := envelope["mode"]; got != resultModeBundle {
		t.Fatalf("mode = %v", got)
	}

	result := envelope["result"].(map[string]any)
	if got := result["okfVersion"]; got != "0.2" {
		t.Fatalf("okfVersion = %v", got)
	}
	revision := result["bundleRevision"].(map[string]any)
	if revision["algorithm"] != "sha256" {
		t.Fatalf("revision algorithm = %v", revision["algorithm"])
	}
	value, _ := revision["value"].(string)
	if !strings.HasPrefix(value, "sha256:") {
		t.Fatalf("revision value = %q", value)
	}

	concepts := result["concepts"].([]any)
	if len(concepts) != 3 {
		t.Fatalf("concepts = %d, want 3", len(concepts))
	}
	ids := map[string]bool{}
	for _, raw := range concepts {
		concept := raw.(map[string]any)
		ids[concept["conceptId"].(string)] = true
		if concept["artifactDigest"] == "" {
			t.Errorf("concept %v has no digest", concept["conceptId"])
		}
	}
	for _, want := range []string{"business/submit", "data/order", "functions/submit"} {
		if !ids[want] {
			t.Errorf("missing concept %q", want)
		}
	}

	format := result["format"].(map[string]any)
	if format["conformant"] != true {
		t.Fatalf("format.conformant = %v, errors = %v", format["conformant"], format["formatErrors"])
	}
	if got := format["concepts"]; got != float64(3) {
		t.Fatalf("format.concepts = %v", got)
	}
	// The not-evaluated list is part of the answer: a conformant bundle is not an
	// accepted one.
	notEvaluated := result["notEvaluated"].(map[string]any)["items"].([]any)
	if len(notEvaluated) == 0 {
		t.Fatal("notEvaluated.items must be present")
	}
}

// TestOKFBundleCLIConceptMode covers the single-concept projection.
func TestOKFBundleCLIConceptMode(t *testing.T) {
	var out bytes.Buffer
	code, err := okfBundleJSON(&out, okfFixturePath, "data/order", "", "", "", 0, 0, 0, 0)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v body=%s", code, err, out.String())
	}
	envelope, _ := decodeOKFEnvelope(t, out)
	if envelope["mode"] != resultModeConcept {
		t.Fatalf("mode = %v", envelope["mode"])
	}
	result := envelope["result"].(map[string]any)
	if _, present := result["concepts"]; present {
		t.Fatal("mode=concept must not return the whole concept list")
	}
	concept := result["concept"].(map[string]any)
	if concept["conceptId"] != "data/order" {
		t.Fatalf("conceptId = %v", concept["conceptId"])
	}
	sources := concept["sources"].([]any)
	if len(sources) != 1 {
		t.Fatalf("sources = %d", len(sources))
	}
	source := sources[0].(map[string]any)
	if source["resourceKind"] != "scope-descriptor" {
		t.Fatalf("resourceKind = %v", source["resourceKind"])
	}
	if source["resolution"] != "scope-descriptor-not-fetched" {
		t.Fatalf("resolution = %v", source["resolution"])
	}
}

// TestOKFBundleCLIUnknownConceptIsAnArgumentError covers a concept ID that names
// nothing: the command must fail as an argument problem rather than return an empty
// success result.
func TestOKFBundleCLIUnknownConceptIsAnArgumentError(t *testing.T) {
	var out bytes.Buffer
	code, err := okfBundleJSON(&out, okfFixturePath, "no/such/concept", "", "", "", 0, 0, 0, 0)
	if err != nil || code != 2 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	envelope, _ := decodeOKFEnvelope(t, out)
	if envelope["ok"] != false {
		t.Fatalf("ok = %v", envelope["ok"])
	}
	failure, ok := envelope["error"].(map[string]any)
	if !ok || failure["code"] != resultCodeInvalidArgument {
		t.Fatalf("error = %v", envelope["error"])
	}
}

// TestOKFBundleCLIRequiresRoot covers the required flag.
func TestOKFBundleCLIRequiresRoot(t *testing.T) {
	var out bytes.Buffer
	code, err := okfBundleJSON(&out, "", "", "", "", "", 0, 0, 0, 0)
	if err != nil || code != 2 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	envelope, _ := decodeOKFEnvelope(t, out)
	failure := envelope["error"].(map[string]any)
	if failure["code"] != resultCodeInvalidArgument {
		t.Fatalf("error = %v", failure)
	}
}

// TestOKFBundleCLIMissingRootIsAnArgumentError covers a root that does not exist.
func TestOKFBundleCLIMissingRootIsAnArgumentError(t *testing.T) {
	var out bytes.Buffer
	code, err := okfBundleJSON(&out, filepath.Join(t.TempDir(), "absent"), "", "", "", "", 0, 0, 0, 0)
	if err != nil || code != 2 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	envelope, _ := decodeOKFEnvelope(t, out)
	if envelope["error"].(map[string]any)["code"] != resultCodeInvalidArgument {
		t.Fatalf("error = %v", envelope["error"])
	}
}

// TestOKFBundleCLIRejectsWorld covers the mutual exclusion with -world.
func TestOKFBundleCLIRejectsWorld(t *testing.T) {
	var out bytes.Buffer
	code, err := okfBundleJSON(&out, okfFixturePath, "", "", "", "other.json", 0, 0, 0, 0)
	if err != nil || code != 2 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if envelope, _ := decodeOKFEnvelope(t, out); envelope["error"].(map[string]any)["code"] != resultCodeInvalidArgument {
		t.Fatalf("error = %v", envelope["error"])
	}
}

// TestOKFBundleCLIExpectedRevision covers the revision comparison: a matching value
// succeeds and a mismatched one fails without claiming the bundle was read.
func TestOKFBundleCLIExpectedRevision(t *testing.T) {
	var baseline bytes.Buffer
	if code, err := okfBundleJSON(&baseline, okfFixturePath, "", "", "", "", 0, 0, 0, 0); err != nil || code != 0 {
		t.Fatalf("baseline read failed: code=%d err=%v", code, err)
	}
	envelope, _ := decodeOKFEnvelope(t, baseline)
	revision := envelope["result"].(map[string]any)["bundleRevision"].(map[string]any)["value"].(string)

	var matching bytes.Buffer
	if code, err := okfBundleJSON(&matching, okfFixturePath, "", revision, "", "", 0, 0, 0, 0); err != nil || code != 0 {
		t.Fatalf("matching revision read failed: code=%d err=%v", code, err)
	}

	var mismatched bytes.Buffer
	code, err := okfBundleJSON(&mismatched, okfFixturePath, "", "sha256:0000", "", "", 0, 0, 0, 0)
	if err != nil || code != 2 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	failure := decodeFailure(t, mismatched)
	if failure["code"] != resultCodeInvalidArgument {
		t.Fatalf("error = %v", failure)
	}
	if !strings.Contains(failure["message"].(string), "bundle revision is") {
		t.Fatalf("message = %v", failure["message"])
	}
}

func decodeFailure(t *testing.T, out bytes.Buffer) map[string]any {
	t.Helper()
	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	failure, ok := envelope["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error in %s", out.String())
	}
	return failure
}

// TestOKFBundleCLIWorkspaceIdentityIsOptional covers -workspace: the identity is
// attached only when the bundle root is inside the workspace.
func TestOKFBundleCLIWorkspaceIdentityIsOptional(t *testing.T) {
	workspaceRoot := filepath.Join("..", "..", "reference", "workspace")
	if _, err := os.Stat(workspaceRoot); err != nil {
		t.Skip("reference workspace is unavailable")
	}
	// A bundle root outside the workspace is refused rather than silently
	// mislabelled.
	var outside bytes.Buffer
	code, err := okfBundleJSON(&outside, okfFixturePath, "", "", workspaceRoot, "", 0, 0, 0, 0)
	if err != nil || code != 2 {
		t.Fatalf("code=%d err=%v body=%s", code, err, outside.String())
	}
	failure := decodeFailure(t, outside)
	if !strings.Contains(failure["message"].(string), "does not contain -root") {
		t.Fatalf("message = %v", failure["message"])
	}
}

// TestOKFBundleCLILimitsAreOptional covers the limit flags: an explicit small limit
// turns an otherwise conformant bundle into a reported non-conformant read.
func TestOKFBundleCLILimitsAreOptional(t *testing.T) {
	var out bytes.Buffer
	code, err := okfBundleJSON(&out, okfFixturePath, "", "", "", "", 64, 0, 0, 0)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	envelope, _ := decodeOKFEnvelope(t, out)
	result := envelope["result"].(map[string]any)
	format := result["format"].(map[string]any)
	if format["conformant"] != false {
		t.Fatal("a refused file read must not be reported as conformant")
	}
	errors := format["formatErrors"].([]any)
	found := false
	for _, raw := range errors {
		if raw.(map[string]any)["code"] == "FILE_TOO_LARGE" {
			found = true
		}
	}
	if !found {
		t.Fatalf("formatErrors = %v, want FILE_TOO_LARGE", errors)
	}
}

// TestOKFBundleIsAdvertised covers discovery: the command must appear with the
// schemaVersion and the two modes it can return, so a caller can branch without
// reading prose.
func TestOKFBundleIsAdvertised(t *testing.T) {
	record := catalogRecord(t, "okf-bundle")
	if record.Effect != "read-only" {
		t.Fatalf("effect = %q, want read-only", record.Effect)
	}
	if record.SchemaVersion != "tei.cli.okf-bundle/v0" {
		t.Fatalf("schemaVersion = %q", record.SchemaVersion)
	}
	if record.JSONContract != "enveloped" {
		t.Fatalf("jsonContract = %q", record.JSONContract)
	}
	if !containsMode(record.Modes, resultModeBundle) || !containsMode(record.Modes, resultModeConcept) {
		t.Fatalf("modes = %v, want bundle and concept", record.Modes)
	}
	allowed, ok := selectedReadOnlyFlagAllowlist["okf-bundle"]
	if !ok {
		t.Fatal("okf-bundle has no accepted-flag list")
	}
	for _, flag := range []string{"root", "concept", "workspace", "expected-bundle-revision", "max-file-bytes", "max-bundle-bytes", "max-concepts"} {
		if !hasFlag(allowed, flag) {
			t.Errorf("accepted flags %v lack %q", allowed, flag)
		}
	}
}

func hasFlag(flags []string, want string) bool {
	for _, flag := range flags {
		if flag == want {
			return true
		}
	}
	return false
}

// TestOKFBundleIsReadOnlyByConstruction covers the effect claim: the command writes
// nothing. Its only write path is the envelope to stdout.
func TestOKFBundleIsReadOnlyByConstruction(t *testing.T) {
	before, err := os.ReadDir(okfFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code, err := okfBundleJSON(&out, okfFixturePath, "", "", "", "", 0, 0, 0, 0); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	after, err := os.ReadDir(okfFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatal("the read changed the bundle directory")
	}
	record := catalogRecord(t, "okf-bundle")
	if len(record.WriteFlags) != 0 {
		t.Fatalf("writeFlags = %v, want none", record.WriteFlags)
	}
}
