package main

import (
	"sort"
	"testing"

	"tei/internal/operations"
)

// TestSubjectKindEnumsComeFromTheOperations pins #3420: a client that reads
// inputSchema must see the same closed set the operation accepts. Before this,
// tei_inspect and tei_trace declared subjectKind as a bare string, while the CLI
// advertised 12 and 10 values for the same flag, so a client had to guess.
func TestSubjectKindEnumsComeFromTheOperations(t *testing.T) {
	tools := map[string]map[string]any{}
	for _, tool := range toolList() {
		tools[tool.Name] = tool.InputSchema
	}

	cases := []struct {
		tool string
		want []string
	}{
		{"tei_inspect", operations.InspectionSubjectKinds},
		{"tei_trace", operations.TraceSubjectKinds},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			schema, ok := tools[tc.tool]
			if !ok {
				t.Fatalf("%s is not advertised", tc.tool)
			}
			properties, ok := schema["properties"].(map[string]any)
			if !ok {
				t.Fatalf("%s has no properties object", tc.tool)
			}
			kindSchema, ok := properties["subjectKind"].(map[string]any)
			if !ok {
				t.Fatalf("%s does not declare subjectKind", tc.tool)
			}
			raw, ok := kindSchema["enum"]
			if !ok {
				t.Fatalf("%s: subjectKind has no enum; a client cannot tell which kinds exist", tc.tool)
			}
			got, ok := raw.([]string)
			if !ok {
				t.Fatalf("%s: subjectKind enum is %T, want []string", tc.tool, raw)
			}
			if !sameSet(got, tc.want) {
				t.Fatalf("%s: enum=%v, operation accepts %v", tc.tool, got, tc.want)
			}
		})
	}
}

// The enum must not overclaim: every advertised value has to be a kind the
// operation actually resolves, and an unadvertised value must be refused. The
// sets are the operation's own, so this is a check that they were not narrowed
// or widened by the adapter.
func TestSubjectKindEnumsMatchTheAdvertisedKindLists(t *testing.T) {
	// The CLI advertises these same slices; the adapter now reads them directly,
	// so a mismatch here would mean the adapter stopped using the operation.
	if len(operations.InspectionSubjectKinds) != 18 {
		t.Fatalf("inspection kinds changed to %d; update the CLI advertisement and this pin together",
			len(operations.InspectionSubjectKinds))
	}
	if len(operations.TraceSubjectKinds) != 13 {
		t.Fatalf("trace kinds changed to %d; update the CLI advertisement and this pin together",
			len(operations.TraceSubjectKinds))
	}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	left := append([]string(nil), a...)
	right := append([]string(nil), b...)
	sort.Strings(left)
	sort.Strings(right)
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
