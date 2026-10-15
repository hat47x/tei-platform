package hostqualification

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// subprocessResult is the real tei CLI transport envelope, not a substituted
// call to a CLI helper. Shared Go operations are called separately in tests.
type subprocessResult struct {
	SchemaVersion string          `json:"schemaVersion"`
	OK            bool            `json:"ok"`
	Result        json.RawMessage `json:"result"`
	Diagnostics   []struct {
		Diagnostic struct {
			Code     string          `json:"code"`
			Severity string          `json:"severity"`
			Subject  json.RawMessage `json:"subject"`
		} `json:"diagnostic"`
		Sources []struct {
			ArtifactPath string            `json:"artifactPath"`
			Range        sourceindex.Range `json:"range"`
		} `json:"sources"`
	} `json:"diagnostics"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func testCLIBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	command := exec.Command("go", "build", "-o", binary, "../../cmd/tei")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build real CLI: %v\n%s", err, output)
	}
	return binary
}

func observeCLI(t *testing.T, binary string, expectedExit int, args ...string) subprocessResult {
	t.Helper()
	command := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	exit := 0
	if err != nil {
		var failure *exec.ExitError
		if !errors.As(err, &failure) {
			t.Fatalf("run CLI %q: %v", args, err)
		}
		exit = failure.ExitCode()
	}
	if exit != expectedExit || stderr.Len() != 0 {
		t.Fatalf("CLI %q exited %d, want %d; stdout=%q stderr=%q",
			args, exit, expectedExit, stdout.String(), stderr.String())
	}
	var response subprocessResult
	decoder := json.NewDecoder(&stdout)
	if err := decoder.Decode(&response); err != nil {
		t.Fatalf("CLI %q did not return JSON: %v", args, err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatalf("CLI %q returned an extra or malformed trailing JSON document: %v", args, err)
	}
	return response
}

func normalizedJSON(t *testing.T, value any) any {
	t.Helper()
	blob, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var normalized any
	if err := json.Unmarshal(blob, &normalized); err != nil {
		t.Fatal(err)
	}
	return normalized
}

func normalizedRawJSON(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		t.Fatalf("decode CLI result: %v", err)
	}
	return normalized
}

// findingSignature compares semantics and source coordinates, not free-form
// Host/CLI explanatory text or the ordering of diagnostics.
func findingSignature(code, severity string, subject json.RawMessage, sources []string) string {
	sort.Strings(sources)
	if len(subject) == 0 || bytes.Equal(subject, []byte("null")) {
		subject = []byte("null")
	}
	return fmt.Sprintf("%s|%s|%s|%v", code, severity, subject, sources)
}

func cliFindings(report subprocessResult) []string {
	out := make([]string, 0, len(report.Diagnostics))
	for _, item := range report.Diagnostics {
		sources := make([]string, 0, len(item.Sources))
		for _, source := range item.Sources {
			sources = append(sources, fmt.Sprintf("%s:%d:%d",
				source.ArtifactPath, source.Range.Start.Line, source.Range.Start.Column))
		}
		out = append(out, findingSignature(
			item.Diagnostic.Code, item.Diagnostic.Severity, item.Diagnostic.Subject, sources))
	}
	sort.Strings(out)
	return out
}

func typedFindings(t *testing.T, result operations.WorkspaceValidation) []string {
	t.Helper()
	// Absence of diagnostics is one semantic state: a non-nil empty slice.
	// cliFindings already returns a non-nil empty slice for the same state.
	out := make([]string, 0, len(result.Diagnostics)+1)
	if result.Direct != nil {
		d := result.Direct
		out = append(out, findingSignature(d.Code, string(d.Severity), nil, []string{
			fmt.Sprintf("%s:%d:%d", d.Source.ArtifactPath,
				d.Source.Range.Start.Line, d.Source.Range.Start.Column),
		}))
	} else {
		for _, projected := range result.Diagnostics {
			d := projected.Diagnostic
			var subject json.RawMessage
			if d.Subject != nil {
				encoded, err := json.Marshal(d.Subject)
				if err != nil {
					t.Fatal(err)
				}
				subject = encoded
			}
			sources := make([]string, 0, len(projected.Sources))
			for _, source := range projected.Sources {
				sources = append(sources, fmt.Sprintf("%s:%d:%d",
					source.ArtifactPath, source.Range.Start.Line, source.Range.Start.Column))
			}
			out = append(out, findingSignature(d.Code, string(d.Severity), subject, sources))
		}
	}
	sort.Strings(out)
	return out
}

// Ensure a nil Go diagnostics field and an empty CLI diagnostics array have
// identical meaning without weakening comparisons for actual diagnostics.
func TestEmptyDiagnosticSignatureParity(t *testing.T) {
	cli := cliFindings(subprocessResult{})
	typed := typedFindings(t, operations.WorkspaceValidation{})
	if !reflect.DeepEqual(cli, typed) {
		t.Fatalf("empty findings differ: CLI=%#v Go=%#v", cli, typed)
	}
}

// missingSourceIndexInputFixture mirrors the missing-target managed source in
// workspace.TestSourceIndexRejectsBrokenSymlink. The manifest makes the source
// root active, so the full Load pipeline gets the same missing input.
func missingSourceIndexInputFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"),
		[]byte("workspace: missing_source_input\nsources:\n  - custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "custom"), 0o755); err != nil {
		t.Fatal(err)
	}
	relative := "custom/missing.tei"
	if err := os.Symlink("not-present.tei", filepath.Join(root, "custom", "missing.tei")); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("broken-link fixture requires symlink support on Windows: %v", err)
		}
		t.Fatalf("Linux broken-link fixture must not silently skip: %v", err)
	}
	return root, relative
}

// A dangling input fails the source-root inventory in Load, BEFORE
// BuildSourceIndex can produce source locations or diagnostic findings.
// The shared Go operation and real CLI must agree on this first stage.
func TestMissingSourceIndexInputStopsAtWorkspaceLoad(t *testing.T) {
	root, missingPath := missingSourceIndexInputFixture(t)
	_, loadErr := workspace.Load(root)
	if loadErr == nil || !strings.Contains(filepath.ToSlash(loadErr.Error()), missingPath) {
		t.Fatalf("missing source was not reported at Load with its path: %v", loadErr)
	}
	_, opErr := operations.ValidateWorkspace(root)
	var stage *operations.ValidationFailure
	if !errors.As(opErr, &stage) || stage.Stage != operations.StageLoad {
		t.Fatalf("missing input must fail before Source Index: %v", opErr)
	}
	cli := observeCLI(t, testCLIBinary(t), 2, "validate", "-workspace", root, "-json")
	if cli.SchemaVersion != "tei.cli.validate/v0" || cli.OK || cli.Error == nil ||
		cli.Error.Code != "WORKSPACE_LOAD_FAILED" || len(cli.Diagnostics) != 0 ||
		!strings.Contains(filepath.ToSlash(cli.Error.Message), missingPath) {
		t.Fatalf("CLI must classify the missing input at Load, with no source-index diagnostics: %+v", cli)
	}
}

func TestCLIProcessVersusReadOnlyOperations(t *testing.T) {
	binary := testCLIBinary(t)
	reference := filepath.Join("..", "..", "reference", "workspace")
	normal := fixtureWorkspace(t, "normal", "namespace normal\n")
	syntax := fixtureWorkspace(t, "syntax", "namespace syntax\n\ndata Broken {\n    attribute broken type }\n}\n")
	missing := filepath.Join(t.TempDir(), "missing")
	semantic := t.TempDir()
	if err := os.WriteFile(filepath.Join(semantic, "tei.workspace.yaml"),
		[]byte("workspace: parity_semantic\nsources:\n  - applications\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(semantic, "applications", "order-entry", "application.tei")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("application order_entry {\n    database sales {\n        target sales_core\n        access invalid\n    }\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name      string
		root      string
		exit      int
		errorCode string
	}{
		{name: "valid", root: normal, exit: 0},
		{name: "syntax-diagnostic", root: syntax, exit: 1},
		{name: "semantic-diagnostic", root: semantic, exit: 1},
		{name: "missing-workspace", root: missing, exit: 2, errorCode: "WORKSPACE_LOAD_FAILED"},
	} {
		t.Run("validate-"+tc.name, func(t *testing.T) {
			cli := observeCLI(t, binary, tc.exit, "validate", "-workspace", tc.root, "-json")
			typed, opErr := operations.ValidateWorkspace(tc.root)
			if tc.errorCode != "" {
				var failure *operations.ValidationFailure
				if !errors.As(opErr, &failure) || failure.Stage != operations.StageLoad ||
					cli.OK || cli.Error == nil || cli.Error.Code != tc.errorCode ||
					len(cli.Diagnostics) != 0 {
					t.Fatalf("CLI load failure vs Go typed failure: cli=%+v go=%v", cli, opErr)
				}
				return
			}
			if opErr != nil || cli.Error != nil || cli.OK == typed.HasErrors() {
				t.Fatalf("CLI status differs from Go operation: cli=%+v go=%+v err=%v", cli, typed, opErr)
			}
			want := typedFindings(t, typed)
			got := cliFindings(cli)
			if tc.exit == 1 && (len(got) == 0 || len(want) == 0) {
				t.Fatalf("diagnostic fixture %q produced no findings: cli=%v go=%v", tc.name, got, want)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("CLI and Go findings differ: cli=%v go=%v", got, want)
			}
		})
	}

	t.Run("catalog-success", func(t *testing.T) {
		cli := observeCLI(t, binary, 0, "model", "catalog", "-workspace", reference, "-namespace", "sales")
		typed, err := operations.CatalogWorkspace(reference, "sales")
		if err != nil {
			t.Fatal(err)
		}
		if !cli.OK || cli.Error != nil ||
			!reflect.DeepEqual(normalizedRawJSON(t, cli.Result), normalizedJSON(t, typed)) {
			t.Fatalf("catalog CLI and Go projection mismatch: cli=%+v go=%+v", cli, typed)
		}
	})
	t.Run("catalog-load-failure", func(t *testing.T) {
		cli := observeCLI(t, binary, 2, "model", "catalog", "-workspace", missing)
		_, err := operations.CatalogWorkspace(missing, "")
		var stage *operations.ValidationFailure
		if !errors.As(err, &stage) || stage.Stage != operations.StageLoad ||
			cli.OK || cli.Error == nil || cli.Error.Code != "WORKSPACE_LOAD_FAILED" {
			t.Fatalf("catalog load mismatch: cli=%+v go=%v", cli, err)
		}
	})
	for _, tc := range []struct {
		name      string
		subject   sourceindex.Subject
		exit      int
		errorCode string
	}{
		{name: "subject-success", subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}},
		{name: "subject-missing", subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.no_such_data"}, exit: 1, errorCode: operations.InspectionNotFound},
		{name: "subject-unsupported", subject: sourceindex.Subject{Kind: "unknown-kind", ID: "sales.order"}, exit: 2, errorCode: operations.InspectionUnsupportedKind},
		{name: "subject-invalid", subject: sourceindex.Subject{Kind: "semantic-data", ID: " sales.order"}, exit: 2, errorCode: operations.InspectionInvalidSubject},
	} {
		t.Run("inspect-"+tc.name, func(t *testing.T) {
			cli := observeCLI(t, binary, tc.exit,
				"inspect", "-workspace", reference, "-subject-kind", tc.subject.Kind, "-subject-id", tc.subject.ID)
			typed, opErr := operations.InspectSubject(reference, tc.subject)
			if tc.errorCode != "" {
				var failure *operations.InspectionError
				if !errors.As(opErr, &failure) || failure.Code != tc.errorCode ||
					cli.OK || cli.Error == nil || cli.Error.Code != tc.errorCode {
					t.Fatalf("inspect rejection mismatch: cli=%+v go=%v", cli, opErr)
				}
				return
			}
			if opErr != nil || !cli.OK || cli.Error != nil ||
				!reflect.DeepEqual(normalizedRawJSON(t, cli.Result), normalizedJSON(t, typed)) {
				t.Fatalf("inspect CLI and Go mismatch: cli=%+v go=%+v err=%v", cli, typed, opErr)
			}
		})
	}
}
