package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func classificationContractFixture() []byte {
	value := map[string]any{
		"constructMappings": map[string]any{
			"xead": map[string]any{
				"System": map[string]any{
					"classification": []string{"source-metadata"},
					"extension":      "reference.compat.xead.source",
					"rules":          []string{"system metadata stays source-scoped"},
				},
				"Table": map[string]any{
					"classification":  []string{"canonical-candidate", "realization", "unresolved"},
					"extension":       "reference.xead.relational-semantics",
					"authorityLimits": []string{"table-is-not-automatically-domain-data"},
				},
			},
		},
	}
	out, _ := json.Marshal(value)
	return out
}

func TestLocalClassificationKeepsLocatorAndMappingSeparateFromSemanticUnresolved(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System><Table ID="local"/><Mystery/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := ClassifyObservedDocument(FormatXead, doc, classificationContractFixture())
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 3 {
		t.Fatalf("candidate count=%d want=3", len(candidates))
	}
	if candidates[1].Locator == "" || candidates[1].Extension != "reference.xead.relational-semantics" {
		t.Fatalf("mapped candidate=%+v", candidates[1])
	}
	if candidates[1].MappingStatus != "mapped" || !containsString(candidates[1].Classification, "unresolved") {
		t.Fatalf("mapping status and semantic classification were conflated: %+v", candidates[1])
	}
	if candidates[2].MappingStatus != "unmapped" || candidates[2].Classification[0] != "unresolved" {
		t.Fatalf("unknown source construct was not kept unmapped/unresolved: %+v", candidates[2])
	}
}

func TestPublishedClassificationDoesNotContainSourceLocatorOrValues(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System ID="private-system"><Table ID="private-table"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	receipt := PublicReceipt(doc)
	projection, err := BuildClassificationProjection(FormatXead, receipt, classificationContractFixture())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"private-system", "private-table", "/System[1]", "/Table[1]"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("public classification leaked source value or locator %q: %s", forbidden, text)
		}
	}
	if len(projection.Kinds) != 2 {
		t.Fatalf("published kind count=%d want=2", len(projection.Kinds))
	}
}

func TestPublishedClassificationKeepsUnmappedKindUnresolved(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System><Mystery/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	projection, err := BuildClassificationProjection(FormatXead, PublicReceipt(doc), classificationContractFixture())
	if err != nil {
		t.Fatal(err)
	}
	var mystery *PublishedClassification
	for i := range projection.Kinds {
		if projection.Kinds[i].Kind == "Mystery" {
			mystery = &projection.Kinds[i]
			break
		}
	}
	if mystery == nil || mystery.MappingStatus != "unmapped" || mystery.UnresolvedObservedCount != 1 {
		t.Fatalf("unmapped kind not preserved: %+v", projection.Kinds)
	}
}

func TestNamespacedLookalikeDoesNotHideBehindMappedLocalName(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System xmlns:x="urn:ext"><Table/><x:Table/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	projection, err := BuildClassificationProjection(FormatXead, PublicReceipt(doc), classificationContractFixture())
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range projection.Kinds {
		if kind.Kind == "Table" {
			if kind.MappingStatus != "mixed" || kind.UnresolvedObservedCount != 1 || kind.ObservedCount != 2 {
				t.Fatalf("namespaced lookalike was hidden by mapped local name: %+v", kind)
			}
			return
		}
	}
	t.Fatal("Table classification missing")
}

func TestClassificationRejectsMappingWithoutClassification(t *testing.T) {
	contract := []byte(`{"constructMappings":{"xead":{"System":{"extension":"reference.compat.xead.source"}}}}`)
	doc, err := Observe(FormatXead, []byte(`<System/>`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildClassificationProjection(FormatXead, PublicReceipt(doc), contract); err == nil || !strings.Contains(err.Error(), "has no classification") {
		t.Fatalf("invalid mapping accepted: %v", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
