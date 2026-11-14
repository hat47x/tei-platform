package syntax_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/syntax"
)

// capabilityAuthoringProbeAdapter deliberately records entry into the real
// capabilityinvoke.Adapter boundary. A rejected declaration must not call it.
type capabilityAuthoringProbeAdapter func(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error)

func (f capabilityAuthoringProbeAdapter) Invoke(ctx context.Context, id core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
	return f(ctx, id, input)
}

func capabilityFromAuthoredSource(t *testing.T, body string) semantic.Capability {
	t.Helper()
	source := "namespace sales\ncapability sales.contract_probe {\n" + body + "}\n"
	ns, err := syntax.NewParser(source).ParseSemanticFile()
	if err != nil {
		t.Fatalf("the authored Capability did not parse: %v\n%s", err, source)
	}
	if len(ns.Capabilities) != 1 {
		t.Fatalf("parsed %d Capabilities, want exactly one", len(ns.Capabilities))
	}
	return ns.Capabilities[0]
}

// A parser success is not a World validation or permission to invoke.
// Unlike tests that construct only a malformed semantic.Capability in Go,
// these tests start from source accepted by the existing TEI parser, then
// require the *existing* pre-dispatch boundary to refuse it without effects.
// Each broken source is paired with a one-factor repair and an actual call.
func TestAuthoredCapabilityStructuralContradictionsRefusedBeforeAdapter(t *testing.T) {
	cases := []struct {
		name     string
		broken   string
		repaired string
		want     string
	}{
		{
			name: "ACD-02 duplicate input declaration",
			broken: "  input token std.text\n  input token std.text\n  output std.text\n",
			repaired: "  input token std.text\n  output std.text\n",
			want: "duplicate input",
		},
		{
			name: "ACD-03 output mixed with outcomes",
			broken: "  output std.text\n  outcome ok std.text\n",
			repaired: "  output std.text\n",
			want: "both a single output and named outcomes",
		},
		{
			name: "ACD-04 duplicate outcome declaration",
			broken: "  outcome ok std.text\n  outcome ok std.text\n",
			repaired: "  outcome ok std.text\n  outcome denied std.text\n",
			want: "duplicate outcome",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			broken := capabilityFromAuthoredSource(t, tc.broken)
			repaired := capabilityFromAuthoredSource(t, tc.repaired)
			if broken.ID != repaired.ID || broken.ID != "sales.contract_probe" {
				t.Fatalf("repair changed Capability identity: broken=%s repaired=%s", broken.ID, repaired.ID)
			}

			calls := 0
			adapter := capabilityAuthoringProbeAdapter(func(_ context.Context, id core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
				calls++
				result := capabilityinvoke.BusinessResult{
					Capability: id,
					Mode:       capabilityinvoke.ResultOutput,
					Payload:    "accepted",
				}
				if repaired.Output == "" {
					result.Mode = capabilityinvoke.ResultOutcome
					result.Outcome = "ok"
				}
				return result, nil
			})
			result, err := capabilityinvoke.Invoke(context.Background(), broken, adapter, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("malformed source reached no expected structural refusal: result=%#v err=%v", result, err)
			}
			if calls != 0 {
				t.Fatalf("malformed source dispatched Adapter %d times", calls)
			}
			if result.Capability != "" || result.Mode != "" || result.Outcome != "" || result.Payload != nil {
				t.Fatalf("refused declaration produced a BusinessResult: %#v", result)
			}

			result, err = capabilityinvoke.Invoke(context.Background(), repaired, adapter, nil)
			if err != nil {
				t.Fatalf("single-factor repair was rejected: %v", err)
			}
			if calls != 1 || result.Capability != repaired.ID {
				t.Fatalf("repair must reach Adapter once: calls=%d result=%#v", calls, result)
			}
		})
	}
}

// A syntactically unsupported effect must fail at its original source token.
// No Canonical Capability exists to pass to the invocation boundary here.
func TestAuthoredCapabilityUnknownEffectRetainsParserPosition(t *testing.T) {
	const source = "namespace sales\ncapability sales.contract_probe {\n  effect unsupported-effect sales.order\n  output std.text\n}\n"
	_, err := syntax.NewParser(source).ParseSemanticFile()
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("unknown effect should fail with a source-positioned ParseError, got %v", err)
	}
	if parseErr.Pos.Line != 3 || parseErr.Pos.Column <= 0 || !strings.Contains(parseErr.Msg, "unsupported data effect kind") {
		t.Fatalf("unexpected source-positioned rejection: %#v", parseErr)
	}
}
