package modelref

import (
	"errors"
	"testing"

	"tei/internal/subject"
)

func traceRequestTestKey() Key {
	return Key{Scope: "s", Subject: subject.Subject{Kind: "k", ID: "i"}}
}

func traceRequestTestGraph() Graph {
	return Graph{Schema: "tei.modelref.v0", Nodes: []Node{{Key: traceRequestTestKey()}}, Edges: []Edge{}}
}

// ValidateTraceRequestは、TraceDirectが主体と引数だけから返すエラーと同じものを返す。
// TraceDirectが要求の誤りとして拒否しない要求では、nilを返す。
func TestValidateTraceRequestMatchesTraceDirect(t *testing.T) {
	cases := []struct {
		name    string
		subject Key
		options TraceOptions
	}{
		{"defaults", traceRequestTestKey(), TraceOptions{}},
		{"explicit summary", traceRequestTestKey(), TraceOptions{Mode: "summary", Direction: "outgoing"}},
		{"explicit full with page", traceRequestTestKey(), TraceOptions{Mode: "full", Direction: "incoming", Offset: 1, Limit: 2}},
		{"empty scope", Key{Subject: subject.Subject{Kind: "k", ID: "i"}}, TraceOptions{}},
		{"empty kind", Key{Scope: "s", Subject: subject.Subject{ID: "i"}}, TraceOptions{}},
		{"empty id", Key{Scope: "s", Subject: subject.Subject{Kind: "k"}}, TraceOptions{}},
		{"bad direction", traceRequestTestKey(), TraceOptions{Direction: "sideways"}},
		{"bad mode", traceRequestTestKey(), TraceOptions{Mode: "bogus"}},
		{"negative offset", traceRequestTestKey(), TraceOptions{Mode: "full", Offset: -1}},
		{"negative limit", traceRequestTestKey(), TraceOptions{Mode: "full", Limit: -1}},
		{"summary with offset", traceRequestTestKey(), TraceOptions{Mode: "summary", Offset: 1}},
		{"summary with limit", traceRequestTestKey(), TraceOptions{Mode: "summary", Limit: 1}},
		{"bad direction and bad mode: direction is checked first", traceRequestTestKey(), TraceOptions{Direction: "x", Mode: "y"}},
		{"expected fingerprint is not a request error", traceRequestTestKey(), TraceOptions{ExpectedFingerprint: "sha256:stale"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			validateErr := ValidateTraceRequest(tc.subject, tc.options)
			_, traceErr := TraceDirect(traceRequestTestGraph(), nil, tc.subject, tc.options)

			var validateCode, traceCode string
			var te *TraceError
			if errors.As(validateErr, &te) {
				validateCode = te.Code
			}
			if validateErr != nil && validateCode == "" {
				t.Fatalf("ValidateTraceRequest returned a non-TraceError: %v", validateErr)
			}
			if errors.As(traceErr, &te) {
				traceCode = te.Code
			}
			isRequestCode := traceCode == "INVALID_SUBJECT" || traceCode == "INVALID_ARGUMENT"
			if validateErr == nil && isRequestCode {
				t.Fatalf("TraceDirect rejects the request (%s) but ValidateTraceRequest accepted it", traceCode)
			}
			if validateErr != nil && validateCode != traceCode {
				t.Fatalf("ValidateTraceRequest says %s, TraceDirect says %s", validateCode, traceCode)
			}
			if validateErr != nil && validateErr.Error() != traceErr.Error() {
				t.Fatalf("messages differ: %q vs %q", validateErr, traceErr)
			}
		})
	}
}

// グラフに依存するエラーは、要求だけの検証では報告しない。
func TestValidateTraceRequestIgnoresGraphDependentErrors(t *testing.T) {
	unsupported := Graph{Schema: "other"}
	if _, err := TraceDirect(unsupported, nil, traceRequestTestKey(), TraceOptions{}); err == nil {
		t.Fatal("precondition: the unsupported schema must be rejected by TraceDirect")
	}
	if err := ValidateTraceRequest(traceRequestTestKey(), TraceOptions{}); err != nil {
		t.Fatalf("a valid request was rejected: %v", err)
	}
	missing := Key{Scope: "s", Subject: subject.Subject{Kind: "k", ID: "absent"}}
	if _, err := TraceDirect(traceRequestTestGraph(), nil, missing, TraceOptions{}); err == nil {
		t.Fatal("precondition: an absent subject must be rejected by TraceDirect")
	}
	if err := ValidateTraceRequest(missing, TraceOptions{}); err != nil {
		t.Fatalf("a graph-dependent error was reported by the request check: %v", err)
	}
}

// 複数の誤りが重なる要求では、どの誤りが先に報告されるか（検査の順序）を固定する。
// ValidateTraceRequestとTraceDirectは同じ検査を共有するので、二者の比較では順序の変化を検出できない。
func TestValidateTraceRequestKeepsTheOrderOfChecks(t *testing.T) {
	for _, tc := range []struct {
		name    string
		subject Key
		options TraceOptions
		code    string
		message string
	}{
		{"subject before direction", Key{}, TraceOptions{Direction: "x"}, "INVALID_SUBJECT", "scope, kind, and id are required"},
		{"direction before mode", traceRequestTestKey(), TraceOptions{Direction: "x", Mode: "y"}, "INVALID_ARGUMENT", "direction must be outgoing, incoming, or both"},
		{"mode before offset", traceRequestTestKey(), TraceOptions{Mode: "y", Offset: -1}, "INVALID_ARGUMENT", "mode must be summary or full"},
		{"offset and limit last", traceRequestTestKey(), TraceOptions{Mode: "summary", Limit: 3}, "INVALID_ARGUMENT", "offset/limit require nonnegative values and full mode"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var te *TraceError
			err := ValidateTraceRequest(tc.subject, tc.options)
			if !errors.As(err, &te) || te.Code != tc.code || te.Message != tc.message {
				t.Fatalf("got %v, want %s: %s", err, tc.code, tc.message)
			}
		})
	}
}
