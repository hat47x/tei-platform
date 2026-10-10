package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFirstModelChangeCLIEndToEnd(t *testing.T) {
	repo := testRepositoryRoot(t)
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/tei")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build tei CLI: %v\n%s", err, out)
	}

	const (
		example = "examples/first-workspace"
		source  = "semantic/intake/request.tei"
	)
	workspaceRoot := t.TempDir()
	for _, relative := range []string{
		"tei.workspace.yaml",
		source,
		"applications/request-log/application.tei",
	} {
		original, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(example), filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(workspaceRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, original, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(expectedExit int, args ...string) string {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Dir = repo
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		exit := 0
		if err != nil {
			var exitError *exec.ExitError
			if e, ok := err.(*exec.ExitError); ok {
				exitError = e
			}
			if exitError == nil {
				t.Fatalf("run tei %v: %v", args, err)
			}
			exit = exitError.ExitCode()
		}
		if exit != expectedExit {
			t.Fatalf("tei %v: exit=%d want=%d stdout=%q stderr=%q", args, exit, expectedExit, stdout.String(), stderr.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("tei %v wrote unexpected stderr: %q", args, stderr.String())
		}
		return stdout.String()
	}
	assertValid := func() {
		t.Helper()
		got := run(0, "validate", "-workspace", workspaceRoot)
		if got != "ok\n" {
			t.Fatalf("unexpected Validation output: %q", got)
		}
	}
	readWorld := func() map[string]json.RawMessage {
		t.Helper()
		raw := run(0, "world", "-workspace", workspaceRoot)
		if strings.Contains(raw, workspaceRoot) || strings.Contains(raw, "semantic/intake/request.tei") {
			t.Fatal("Canonical World must not embed the physical authoring source location")
		}
		var world map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &world); err != nil {
			t.Fatalf("unmarshal world: %v", err)
		}
		return world
	}
	readSourceIndex := func(id string) sourceIndexSubjectView {
		t.Helper()
		raw := run(0, "source-index", "-workspace", workspaceRoot,
			"-subject-kind", "semantic-attribute", "-subject-id", id)
		var result sourceIndexSubjectView
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			t.Fatalf("unmarshal source-index %s: %v", id, err)
		}
		if len(result.References) != 1 || result.References[0].ArtifactPath != source {
			t.Fatalf("source-index for %s lost its definition path: %+v", id, result.References)
		}
		return result
	}

	assertValid()
	before := readWorld()
	originalReference := readSourceIndex("intake.request.status")

	srcPath := filepath.Join(workspaceRoot, filepath.FromSlash(source))
	src, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	const old = "    attribute status type std.code\n"
	const replacement = old + "    attribute priority type std.code\n"
	if strings.Count(string(src), old) != 1 {
		t.Fatal("unexpected first-workspace source; cannot perform the documented one-line edit")
	}
	modified := strings.Replace(string(src), old, replacement, 1)
	if err := os.WriteFile(srcPath, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}

	assertValid()
	after := readWorld()
	if bytes.Equal(before["semantics"], after["semantics"]) {
		t.Fatal("the added business attribute was not reflected in Canonical semantics")
	}
	if !bytes.Equal(before["applications"], after["applications"]) {
		t.Fatal("unchanged application definition was modified by the semantic edit")
	}
	if !bytes.Contains(after["semantics"], []byte("intake.request.priority")) {
		t.Fatal("Canonical semantics do not contain the added business attribute")
	}
	if got := readSourceIndex("intake.request.status"); got.References[0].ArtifactPath != originalReference.References[0].ArtifactPath {
		t.Fatal("existing attribute identity lost its source locator")
	}
	added := readSourceIndex("intake.request.priority")
	if len(added.Artifacts) != 1 || added.Artifacts[0].Fingerprint == "" {
		t.Fatalf("new attribute has no source fingerprint: %+v", added.Artifacts)
	}

	invalid := strings.Replace(modified, "priority type std.code", "priority type std.not_defined", 1)
	if err := os.WriteFile(srcPath, []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	diagnostic := run(1, "validate", "-workspace", workspaceRoot)
	if !strings.Contains(diagnostic, "UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE") || !strings.Contains(diagnostic, source) {
		t.Fatalf("invalid type was not reported with source evidence: %q", diagnostic)
	}

	if err := os.WriteFile(srcPath, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}
	assertValid()
	repaired := readWorld()
	if !bytes.Equal(after["semantics"], repaired["semantics"]) {
		t.Fatal("the corrected model differs from the previously valid semantic meaning")
	}

	// A first-time user can also mistype the used namespace. Validation must
	// link the error to the application source, not just report an unknown ID.
	applicationPath := filepath.Join(workspaceRoot, "applications", "request-log", "application.tei")
	appSource, err := os.ReadFile(applicationPath)
	if err != nil {
		t.Fatal(err)
	}
	const semanticUse = "uses semantic intake"
	if strings.Count(string(appSource), semanticUse) != 1 {
		t.Fatal("unexpected application declaration in first workspace")
	}
	badAppSource := strings.Replace(string(appSource), semanticUse, "uses semantic misspelled", 1)
	if err := os.WriteFile(applicationPath, []byte(badAppSource), 0o644); err != nil {
		t.Fatal(err)
	}
	missingNamespace := run(1, "validate", "-workspace", workspaceRoot)
	if !strings.Contains(missingNamespace, "UNKNOWN_APPLICATION_SEMANTIC_NAMESPACE") ||
		!strings.Contains(missingNamespace, "applications/request-log/application.tei:") {
		t.Fatalf("unknown application namespace did not resolve to its authoring source: %q", missingNamespace)
	}
	if err := os.WriteFile(applicationPath, appSource, 0o644); err != nil {
		t.Fatal(err)
	}
	assertValid()

	// A malformed source is a parse error, not an empty or partially accepted model.
	badSyntax := strings.Replace(modified, "priority type std.code", "priority type }", 1)
	if err := os.WriteFile(srcPath, []byte(badSyntax), 0o644); err != nil {
		t.Fatal(err)
	}
	syntaxDiagnostic := run(1, "validate", "-workspace", workspaceRoot)
	if !strings.Contains(syntaxDiagnostic, "PARSE_ERROR") ||
		!strings.Contains(syntaxDiagnostic, source) {
		t.Fatalf("invalid syntax was not reported with its source: %q", syntaxDiagnostic)
	}
	if err := os.WriteFile(srcPath, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}
	assertValid()

	// A relationship must not be accepted when its target Data does not exist.
	// This is an authoring prerequisite for, not proof of, a working CRUD view.
	if strings.Count(modified, "\n}\n") != 1 {
		t.Fatal("first-workspace data declaration has an unexpected shape")
	}
	badRelation := strings.Replace(
		modified, "\n}\n", "\n    relation parent to intake.unknown_request one\n}\n", 1,
	)
	if err := os.WriteFile(srcPath, []byte(badRelation), 0o644); err != nil {
		t.Fatal(err)
	}
	relationDiagnostic := run(1, "validate", "-workspace", workspaceRoot)
	if !strings.Contains(relationDiagnostic, "UNKNOWN_SEMANTIC_RELATION_TARGET") ||
		!strings.Contains(relationDiagnostic, source+":") {
		t.Fatalf("unknown related Data was not reported with its source: %q", relationDiagnostic)
	}

	correctRelation := strings.Replace(badRelation, "intake.unknown_request", "intake.request", 1)
	if err := os.WriteFile(srcPath, []byte(correctRelation), 0o644); err != nil {
		t.Fatal(err)
	}
	assertValid()
	relationWorld := readWorld()
	if !bytes.Contains(relationWorld["semantics"], []byte("intake.request.parent")) {
		t.Fatal("corrected relation is not present in Canonical semantics")
	}
	// Removing the temporary relation must return to the exact pre-error model.
	if err := os.WriteFile(srcPath, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}
	assertValid()
	if restored := readWorld(); !bytes.Equal(restored["semantics"], after["semantics"]) {
		t.Fatal("removing the experimental relation did not restore canonical meaning")
	}

	// Declared Action types must resolve before the Action can be used for
	// application derivation; validation does not create an executable binding.
	badAction := modified + `
capability intake.request.submit {
    input request intake.unknown_request
    output intake.unknown_result
}
`
	if err := os.WriteFile(srcPath, []byte(badAction), 0o644); err != nil {
		t.Fatal(err)
	}
	actionDiagnostic := run(1, "validate", "-workspace", workspaceRoot)
	for _, code := range []string{
		"UNKNOWN_SEMANTIC_CAPABILITY_INPUT",
		"UNKNOWN_SEMANTIC_CAPABILITY_OUTPUT",
	} {
		if !strings.Contains(actionDiagnostic, code) {
			t.Fatalf("bad Action lacks %s: %q", code, actionDiagnostic)
		}
	}
	if !strings.Contains(actionDiagnostic, source+":") {
		t.Fatalf("bad Action types lost their authoring source: %q", actionDiagnostic)
	}
	repairedAction := strings.ReplaceAll(badAction, "intake.unknown_request", "intake.request")
	repairedAction = strings.ReplaceAll(repairedAction, "intake.unknown_result", "intake.request")
	if err := os.WriteFile(srcPath, []byte(repairedAction), 0o644); err != nil {
		t.Fatal(err)
	}
	assertValid()
	if result := readWorld(); !bytes.Contains(result["semantics"], []byte("intake.request.submit")) {
		t.Fatal("the corrected semantic Action is absent from Canonical meaning")
	}
	if err := os.WriteFile(srcPath, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}
	assertValid()

	// Relation key types must be validated on the standard CLI path, not
	// solely by calling a specialist validator in isolation.
	badKey := strings.Replace(
		modified,
		"    attribute status type std.code\n",
		"    attribute status type std.code\n"+
			"    relation parent to intake.request one\n"+
			"        key intake.request.summary -> intake.request.request_id\n",
		1,
	)
	if err := os.WriteFile(srcPath, []byte(badKey), 0o644); err != nil {
		t.Fatal(err)
	}
	keyDiagnostic := run(1, "validate", "-workspace", workspaceRoot)
	if !strings.Contains(keyDiagnostic, "SEMANTIC_RELATION_KEY_TYPE_MISMATCH") ||
		!strings.Contains(keyDiagnostic, source+":") {
		t.Fatalf("CLI accepted mismatched semantic Relation keys: %q", keyDiagnostic)
	}
	goodKey := strings.Replace(
		badKey, "key intake.request.summary ->", "key intake.request.request_id ->", 1,
	)
	if err := os.WriteFile(srcPath, []byte(goodKey), 0o644); err != nil {
		t.Fatal(err)
	}
	assertValid()
	if err := os.WriteFile(srcPath, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}
	assertValid()
}
