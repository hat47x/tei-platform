package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const goodFixture = "{\"fixtureVersion\":\"tei-studio-poc-b-reference-v0\",\"scope\":\"studio.local\",\"services\":[{\"id\":\"studio.validate\",\"version\":\"0.1.0\"},{\"id\":\"studio.catalog\",\"version\":\"0.1.0\"}],\"uiKit\":{\"schemaVersion\":\"tei-ui-composition-reference-v0\"}}"

func TestComposeTwoServicesStableAndStatic(t *testing.T) {
	a, evidence, err := composeServices([]byte(goodFixture))
	if err != nil {
		t.Fatal(err)
	}
	b, repeated, err := composeServices([]byte(goodFixture))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) || evidence.GeneratedSHA256 != repeated.GeneratedSHA256 {
		t.Fatal("generation is not deterministic")
	}
	if len(evidence.ServiceOrder) != 2 ||
		evidence.ServiceOrder[0] != "studio.catalog" ||
		evidence.ServiceOrder[1] != "studio.validate" {
		t.Fatalf("unexpected existing-registry order: %v", evidence.ServiceOrder)
	}
	for _, symbol := range []string{"NewCatalogService", "NewValidateService", "CatalogWorkspace", "ValidateWorkspace", "func Services(root string) []any"} {
		if !bytes.Contains(a, []byte(symbol)) {
			t.Fatalf("missing compiled-in symbol %s", symbol)
		}
	}
	if bytes.Contains(a, []byte("os/exec")) || bytes.Contains(a, []byte("plugin.Open")) {
		t.Fatal("dynamic plugin loading introduced")
	}
}

func TestComposeRejectsUnknownDuplicateAndWrongVersion(t *testing.T) {
	cases := []struct{ name, before, after, reason string }{
		{"unknown", "\"studio.catalog\"", "\"unknown.module\"", "unknown service"},
		{"duplicate", "\"studio.validate\"", "\"studio.catalog\"", "duplicate service"},
		{"version", "\"0.1.0\"", "\"9.0.0\"", "incompatible version"},
		{"scope", "\"studio.local\"", "\"other.tenant\"", "unsupported or incomplete"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := strings.Replace(goodFixture, tc.before, tc.after, 1)
			if _, _, err := composeServices([]byte(input)); err == nil || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("expected %q, got %v", tc.reason, err)
			}
		})
	}
}

func TestComposeRejectsMalformedAndIncomplete(t *testing.T) {
	for _, raw := range []string{"{", "{}", strings.Replace(goodFixture, "\"uiKit\":{\"schemaVersion\":\"tei-ui-composition-reference-v0\"}", "\"uiKit\":null", 1)} {
		if _, _, err := composeServices([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid input %q", raw)
		}
	}
}

func TestProvenanceSerialization(t *testing.T) {
	_, evidence, err := composeServices([]byte(goodFixture))
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("fixtureSha256")) ||
		!bytes.Contains(b, []byte("generatedSha256")) ||
		!bytes.Contains(b, []byte("authorityLimitations")) {
		t.Fatalf("incomplete provenance: %s", b)
	}
}

func TestServiceSelectionOrderDoesNotChangeGeneratedSource(t *testing.T) {
	original, originalEvidence, err := composeServices([]byte(goodFixture))
	if err != nil {
		t.Fatal(err)
	}
	var p profile
	if err := json.Unmarshal([]byte(goodFixture), &p); err != nil {
		t.Fatal(err)
	}
	p.Services[0], p.Services[1] = p.Services[1], p.Services[0]
	reordered, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	actual, evidence, err := composeServices(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, actual) || evidence.GeneratedSHA256 != originalEvidence.GeneratedSHA256 {
		t.Fatal("generated Go changes with fixture service selection order")
	}
	if evidence.FixtureSHA256 == originalEvidence.FixtureSHA256 {
		t.Fatal("input fingerprint must distinguish reordered raw fixture")
	}
}

func TestMissingServiceCannotProduceIncompleteStaticHost(t *testing.T) {
	var p profile
	if err := json.Unmarshal([]byte(goodFixture), &p); err != nil {
		t.Fatal(err)
	}
	p.Services = p.Services[:1]
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = composeServices(raw)
	if err == nil || !strings.Contains(err.Error(), "unsupported or incomplete") {
		t.Fatalf("expected missing-service rejection, got %v", err)
	}
}
