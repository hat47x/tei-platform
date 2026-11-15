package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestStrictJSONRejectsShadowedClosureAndNestedDependencies(t *testing.T) {
	v := sample()
	v.Edges = []edge{{From: "pdp.banner", To: "pdp.offer-rule"}}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	original := string(data)
	cases := []struct {
		name        string
		needle      string
		replacement string
	}{
		{"duplicate-closure", `"declaredScopeClosed":true`, `"declaredScopeClosed":false,"declaredScopeClosed":true`},
		{"case-shadowed-closure", `"declaredScopeClosed":true`, `"declaredScopeClosed":false,"DeclaredScopeClosed":true`},
		{"unicode-shadowed-closure", `"declaredScopeClosed":true`, `"declaredScopeClosed":false,"declaredſcopeClosed":true`},
		{"unicode-shadowed-scope", `"scope":"pdp-public"`, `"scope":"untrusted","ſcope":"pdp-public"`},
		{"unicode-shadowed-nodes", `"nodes":[`, `"nodes":[],"nodeſ":[`},
		{"duplicate-required", `"required":[`, `"required":[],"required":[`},
		{"duplicate-nested-edge", `"from":"pdp.banner"`, `"from":"untrusted.source","from":"pdp.banner"`},
		{"case-shadowed-nested-node", `"id":"pdp.banner"`, `"id":"untrusted.node","ID":"pdp.banner"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(original, tc.needle) {
				t.Fatalf("missing input marker %s", tc.needle)
			}
			payload := strings.Replace(original, tc.needle, tc.replacement, 1)
			var out bytes.Buffer
			if err := run(strings.NewReader(payload), &out); err == nil {
				t.Fatal("shadowed JSON input was accepted")
			}
			if out.Len() != 0 {
				t.Fatalf("rejected input produced report: %s", out.String())
			}
		})
	}
}

func TestStrictJSONInputCannotHideTrailingBytesPastLimit(t *testing.T) {
	data, err := json.Marshal(sample())
	if err != nil {
		t.Fatal(err)
	}
	payload := string(data) + strings.Repeat(" ", 1<<20) + `{"declaredScopeClosed":true}`
	var out bytes.Buffer
	if err := run(strings.NewReader(payload), &out); err == nil {
		t.Fatal("oversize JSON trailing input was accepted")
	}
	if out.Len() != 0 {
		t.Fatalf("oversize input produced report: %s", out.String())
	}
}

func TestStrictJSONUnicodeSimpleFoldMatchesActualDecoder(t *testing.T) {
	// Go's typed Decoder treats a long-s as a case-insensitive match for
	// ASCII s. Lowercasing the keys alone does not detect the duplicate.
	input := []byte(`{"scope":"original","ſcope":"overwritten"}`)
	var decoded struct {
		Scope string `json:"scope"`
	}
	if err := json.Unmarshal(input, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Scope != "overwritten" {
		t.Fatalf("Unicode shadow fixture did not overwrite the field: %q", decoded.Scope)
	}
	if simpleFoldJSONKey("scope") != simpleFoldJSONKey("ſcope") ||
		simpleFoldJSONKey("declaredScopeClosed") != simpleFoldJSONKey("declaredſcopeClosed") ||
		simpleFoldJSONKey("nodes") != simpleFoldJSONKey("nodeſ") {
		t.Fatal("Unicode simple-fold equivalent keys remained distinct")
	}
	if simpleFoldJSONKey("scope") == simpleFoldJSONKey("scopes") {
		t.Fatal("two independent keys were collapsed")
	}
	if err := inspectJSONKeys(input); err == nil {
		t.Fatal("Unicode case-shadow was accepted by strict decoder")
	}
}
