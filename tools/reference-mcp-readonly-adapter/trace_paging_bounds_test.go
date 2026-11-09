package main

import (
	"encoding/json"
	"testing"
)

// TestTracePagingBoundsAreDeclared pins #3420: trace accepts offset and limit of
// 0 (meaning "not set") and refuses negative values, but the schema said only
// "integer". A client could not tell which values the operation refuses, while
// `model locate` declared its own range from 1 to LocationMaxLimit.
func TestTracePagingBoundsAreDeclared(t *testing.T) {
	var traceSchema map[string]any
	for _, tool := range toolList() {
		if tool.Name == "tei_trace" {
			traceSchema = tool.InputSchema
		}
	}
	if traceSchema == nil {
		t.Fatal("tei_trace is not advertised")
	}
	properties, ok := traceSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("tei_trace has no properties")
	}
	for _, name := range []string{"offset", "limit"} {
		raw, ok := properties[name]
		if !ok {
			t.Fatalf("tei_trace does not declare %s", name)
		}
		schema, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("%s schema is %T", name, raw)
		}
		minimum, present := schema["minimum"]
		if !present {
			t.Fatalf("%s has no minimum; a client cannot tell that a negative value is refused", name)
		}
		if minimum != 0 {
			t.Fatalf("%s minimum=%v, want 0 (0 means \"not set\" and is accepted)", name, minimum)
		}
	}
}

// The two limits are different on purpose and both must say so: trace treats 0
// as "not set", model locate starts at 1 and caps at LocationMaxLimit.
func TestTheTwoLimitsDifferInTheirDeclarations(t *testing.T) {
	schemas := map[string]map[string]any{}
	for _, tool := range toolList() {
		properties, _ := tool.InputSchema["properties"].(map[string]any)
		if properties == nil {
			continue
		}
		if raw, ok := properties["limit"].(map[string]any); ok {
			schemas[tool.Name] = raw
		}
	}
	traceLimit, ok := schemas["tei_trace"]
	if !ok {
		t.Fatal("tei_trace does not declare limit")
	}
	locateLimit, ok := schemas["tei_model_locate"]
	if !ok {
		t.Fatal("tei_model_locate does not declare limit")
	}
	if traceLimit["minimum"] != 0 {
		t.Fatalf("tei_trace limit minimum=%v, want 0", traceLimit["minimum"])
	}
	if locateLimit["minimum"] != 1 {
		t.Fatalf("tei_model_locate limit minimum=%v, want 1", locateLimit["minimum"])
	}
	if _, present := locateLimit["maximum"]; !present {
		t.Fatal("tei_model_locate limit has no maximum")
	}
}

// A declared minimum is a promise: the operation must accept the minimum and
// refuse the value just below it. This is checked against the real dispatch so
// the schema cannot drift from behaviour.
func TestDeclaredTraceMinimumIsWhatTheOperationDoes(t *testing.T) {
	root := referenceWorkspace(t)
	cases := []struct {
		name    string
		args    map[string]any
		wantErr bool
	}{
		{"limit at minimum", map[string]any{"limit": 0}, false},
		{"limit below minimum", map[string]any{"limit": -1}, true},
		{"offset at minimum", map[string]any{"offset": 0}, false},
		{"offset below minimum", map[string]any{"offset": -1}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			arguments := map[string]any{
				"scope": "s", "subjectKind": "semantic-data",
				"subjectId": "sales.order", "mode": "full",
			}
			for key, value := range tc.args {
				arguments[key] = value
			}
			raw, err := json.Marshal(map[string]any{
				"jsonrpc": "2.0", "id": 1, "method": "tools/call",
				"params": map[string]any{"name": "tei_trace", "arguments": arguments},
			})
			if err != nil {
				t.Fatal(err)
			}
			request, decodeErr := decodeRequest(string(raw))
			if decodeErr != nil {
				t.Fatalf("decode: %+v", decodeErr)
			}
			response := handle(request, root)
			// A negative value reaches the operation, which refuses it as a
			// semantic failure in the document (isError), not as a protocol
			// error. The declared minimum must agree with that boundary.
			refused := response.Error != nil
			if result, ok := response.Result.(toolResult); ok && result.IsError {
				refused = true
			}
			if refused != tc.wantErr {
				t.Fatalf("refused=%v, want %v (%+v)", refused, tc.wantErr, response)
			}
		})
	}
}
