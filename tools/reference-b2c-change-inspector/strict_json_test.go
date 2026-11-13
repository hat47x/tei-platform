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
