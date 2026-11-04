package languagereference

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/scenariotest"
)

// AI pipeline fixtures are test-only candidate inputs. They do not define a
// production DSL, accepted business meaning, a scenario SSOT, or an oracle.
type aiPipelineFixture struct {
	Contracts map[string]OperatorContract `json:"contracts"`
	Cases     []aiPipelineCase            `json:"cases"`
}

type aiPipelineCase struct {
	ID       string           `json:"id"`
	Source   string           `json:"source"`
	Pins     []string         `json:"pins"`
	Catalog  []string         `json:"catalog"`
	Expected aiPipelineExpect `json:"expected"`
}

type aiPipelineExpect struct {
	Stage                  string `json:"stage"`
	OutputType             string `json:"output_type"`
	ErrorCode              string `json:"error_code"`
	SpanToken              string `json:"span_token"`
	Step                   *int   `json:"step"`
	FirstOperatorByteStart *int   `json:"first_operator_byte_start"`
}

// These independently fixed characterization anchors cannot be deleted or
// replaced by generating another JSON file. New AI-authored cases remain
// candidates until their expectations receive independent review.
var aiPipelineAnchors = map[string]struct {
	source string
	stage  string
	output string
	code   string
}{
	"p0-pure-expand":     {"rows/order |> std.where |> std.expand", "ok", "rows/line", ""},
	"p0-utf8-byte-span":  {"　rows/order |> std.where", "ok", "rows/order", ""},
	"p0-confusable-pin":  {"rows/order |> std.ｗhere", "lower_error", "", "invalid"},
	"p0-effect-rejected": {"rows/order |> std.invoke", "lower_error", "", "effect"},
}

func TestAIPipelineCandidateCases(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "ai_pipeline", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no AI pipeline candidate fixture files: %v", err)
	}
	seen := make(map[string]bool)
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		// Reuse the existing Driver Scenario JSON gate: decoding alone accepts
		// duplicate object keys, so an AI-authored candidate could silently
		// overwrite a source, expected result, or Operator fingerprint.
		if err := scenariotest.RejectDuplicateJSONMembers(data); err != nil {
			t.Fatalf("%s: ambiguous JSON fixture: %v", path, err)
		}
		var suite aiPipelineFixture
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&suite); err != nil {
			t.Fatalf("%s: invalid fixture JSON: %v", path, err)
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			t.Fatalf("%s: unexpected trailing JSON: %v", path, err)
		}
		if len(suite.Cases) == 0 {
			t.Fatalf("%s: empty fixture cannot pass", path)
		}
		for _, tc := range suite.Cases {
			if tc.ID == "" || tc.Source == "" || seen[tc.ID] {
				t.Fatalf("%s: missing/duplicate case ID or empty source: %q", path, tc.ID)
			}
			seen[tc.ID] = true
			if anchor, ok := aiPipelineAnchors[tc.ID]; ok {
				if tc.Source != anchor.source || tc.Expected.Stage != anchor.stage ||
					tc.Expected.OutputType != anchor.output || tc.Expected.ErrorCode != anchor.code {
					t.Fatalf("%s: independently fixed baseline %s was changed", path, tc.ID)
				}
			}
			t.Run(tc.ID, func(t *testing.T) {
				pins := resolveAIContracts(t, suite.Contracts, tc.Pins)
				catalog := resolveAIContracts(t, suite.Contracts, tc.Catalog)
				syntax, err := ParsePipelineSyntax(tc.Source)
				if tc.Expected.Stage == "parse_error" {
					checkAIPipelineFailure(t, tc, err)
					return
				}
				if err != nil {
					t.Fatalf("syntax unexpectedly failed: %v", err)
				}
				for _, token := range append([]PipelineToken{syntax.Input}, syntax.Operators...) {
					if token.Span.Start < 0 || token.Span.End > len(tc.Source) ||
						token.Span.End < token.Span.Start ||
						tc.Source[token.Span.Start:token.Span.End] != token.ID {
						t.Fatalf("source byte span changed: %+v", token)
					}
				}
				if start := tc.Expected.FirstOperatorByteStart; start != nil {
					if len(syntax.Operators) == 0 || syntax.Operators[0].Span.Start != *start {
						t.Fatalf("first operator byte start = %+v, want %d", syntax.Operators, *start)
					}
				}
				steps, err := LowerPinnedPipeline(syntax, pins, catalog)
				if tc.Expected.Stage == "lower_error" {
					checkAIPipelineFailure(t, tc, err)
					return
				}
				if tc.Expected.Stage != "ok" || tc.Expected.OutputType == "" {
					t.Fatalf("unrecognized/incomplete expected stage: %+v", tc.Expected)
				}
				if err != nil {
					t.Fatalf("pinned lowering unexpectedly failed: %v", err)
				}
				if len(steps) != len(syntax.Operators) {
					t.Fatalf("lowering omitted operators: %d of %d", len(steps), len(syntax.Operators))
				}
				calls := make([]PinnedCall, 0, len(steps))
				for i, step := range steps {
					if step.Span != syntax.Operators[i].Span {
						t.Fatalf("lowering lost source span: got %+v, want %+v", step.Span, syntax.Operators[i].Span)
					}
					calls = append(calls, PinnedCall{Contract: step.Contract, InputType: step.InputType})
				}
				got, err := ComposePinned(syntax.Input.ID, calls, catalog)
				if err != nil || got != tc.Expected.OutputType {
					t.Fatalf("composed type %q, error %v; independent expected %q", got, err, tc.Expected.OutputType)
				}
			})
		}
	}
	for id := range aiPipelineAnchors {
		if !seen[id] {
			t.Fatalf("missing mandatory characterization anchor %q", id)
		}
	}
}

func resolveAIContracts(t *testing.T, contracts map[string]OperatorContract, refs []string) []OperatorContract {
	t.Helper()
	result := make([]OperatorContract, 0, len(refs))
	for _, ref := range refs {
		c, ok := contracts[ref]
		if !ok {
			t.Fatalf("unknown OperatorContract reference %q", ref)
		}
		result = append(result, c)
	}
	return result
}

func checkAIPipelineFailure(t *testing.T, tc aiPipelineCase, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s/%s rejection but pipeline succeeded", tc.Expected.Stage, tc.Expected.ErrorCode)
	}
	var diag *PipelineDiagnostic
	if !errors.As(err, &diag) {
		t.Fatalf("lost source-located diagnostic: %v", err)
	}
	if tc.Expected.Step != nil && diag.Step != *tc.Expected.Step {
		t.Fatalf("diagnostic step %d, want %d", diag.Step, *tc.Expected.Step)
	}
	if tc.Expected.SpanToken != "" {
		if diag.Span.Start < 0 || diag.Span.End > len(tc.Source) ||
			diag.Span.End < diag.Span.Start ||
			tc.Source[diag.Span.Start:diag.Span.End] != tc.Expected.SpanToken {
			t.Fatalf("diagnostic bytes %+v do not cover %q", diag.Span, tc.Expected.SpanToken)
		}
	}
	var target error
	switch tc.Expected.ErrorCode {
	case "syntax":
		if tc.Expected.Stage != "parse_error" {
			t.Fatalf("syntax error expected from wrong stage: %+v", tc.Expected)
		}
		return
	case "invalid":
		target = ErrOperatorInvalid
	case "missing":
		target = ErrOperatorMissing
	case "ambiguous":
		target = ErrOperatorAmbiguous
	case "effect":
		target = ErrEffectEscalation
	default:
		t.Fatal(fmt.Errorf("unrecognized error classification %q", tc.Expected.ErrorCode))
	}
	if tc.Expected.Stage != "lower_error" || !errors.Is(err, target) {
		t.Fatalf("expected %s at lower stage, got %v", tc.Expected.ErrorCode, err)
	}
}

func TestAIPipelineRejectsAmbiguousJSONMembers(t *testing.T) {
	// Same key twice is not an independent alternate expectation. The
	// central Scenario scanner must reject it before a normal JSON Decode.
	candidate := []byte(`{"contracts":{},"cases":[{"id":"x","source":"rows/order |> std.where","source":"rows/other |> std.where"}]}`)
	if err := scenariotest.RejectDuplicateJSONMembers(candidate); err == nil {
		t.Fatal("duplicate JSON source member accepted")
	}
}
