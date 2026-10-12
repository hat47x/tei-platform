package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

func TestCapabilityDataEffectParsesWithoutInferringRuntimeAuthority(t *testing.T) {
	src := "namespace intake\ndata Request {\n    attribute requestId type std.identifier\n}\ncapability intake.request.create {\n    input request intake.request\n    output intake.request\n    effect create intake.request\n}\n"
	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(ns.Capabilities) != 1 || ns.Capabilities[0].Effect == nil ||
		ns.Capabilities[0].Effect.Kind != semantic.DataEffectCreate ||
		ns.Capabilities[0].Effect.Data != "intake.request" {
		t.Fatalf("effect was not preserved: %+v", ns.Capabilities)
	}
	legacy := "namespace intake\ncapability intake.request.inspect {\n    output std.code\n}\n"
	parsed, err := syntax.NewParser(legacy).ParseSemanticFile()
	if err != nil || len(parsed.Capabilities) != 1 || parsed.Capabilities[0].Effect != nil {
		t.Fatalf("an old capability must remain valid with no effect: %+v, %v", parsed, err)
	}
}

func TestCapabilityDataEffectRejectsDuplicateAndUnknownKind(t *testing.T) {
	cases := []struct {
		name    string
		effects string
		want    string
	}{
		{"unknown operation", "effect upsert intake.request", "unsupported data effect"},
		{"duplicate", "effect create intake.request\n    effect update intake.request", "more than once"},
		{"invalid identity", "effect update Not-Canonical", "invalid canonical"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			src := "namespace intake\ncapability intake.request.save {\n    " + tt.effects + "\n}\n"
			_, err := syntax.NewParser(src).ParseSemanticFile()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}
