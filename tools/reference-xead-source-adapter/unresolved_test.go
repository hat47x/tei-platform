package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUnresolvedProjectionAggregatesSourceMappingAndSemanticUnresolved(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System><Table/><Mystery/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	receipt := PublicReceipt(doc)
	classification, err := BuildClassificationProjection(FormatXead, receipt, classificationContractFixture())
	if err != nil {
		t.Fatal(err)
	}
	projection, err := BuildUnresolvedProjection(FormatXead, receipt, classification)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, item := range projection.Items {
		counts[item.Code+"|"+item.Kind] += item.Count
	}
	if counts["unknown-element|"] != 1 {
		t.Fatalf("source unresolved missing: %+v", projection.Items)
	}
	if counts["classification-unmapped-redacted|"] != 1 {
		t.Fatalf("redacted mapping unresolved missing: %+v", projection.Items)
	}
	if counts["semantic-classification-unresolved|Table"] != 1 {
		t.Fatalf("semantic unresolved missing: %+v", projection.Items)
	}
}

func TestUnresolvedProjectionDoesNotLeakSourceLocatorOrValues(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System ID="private-system"><Mystery Secret="private-value"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	receipt := PublicReceipt(doc)
	classification, err := BuildClassificationProjection(FormatXead, receipt, classificationContractFixture())
	if err != nil {
		t.Fatal(err)
	}
	projection, err := BuildUnresolvedProjection(FormatXead, receipt, classification)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"private-system", "private-value", "/System[1]", "/Mystery[1]"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("unresolved projection leaked source content %q: %s", forbidden, text)
		}
	}
}

func TestUnresolvedProjectionRejectsFormatMismatch(t *testing.T) {
	_, err := BuildUnresolvedProjection(
		FormatXead,
		PublicationReceipt{Format: FormatXead},
		ClassificationProjection{Format: FormatXeaf},
	)
	if err == nil || !strings.Contains(err.Error(), "format mismatch") {
		t.Fatalf("format mismatch accepted: %v", err)
	}
}


func TestMappingUnresolvedDoesNotClaimMappedExtension(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System xmlns:x="urn:ext"><Table/><x:Table/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	receipt := PublicReceipt(doc)
	classification, err := BuildClassificationProjection(FormatXead, receipt, classificationContractFixture())
	if err != nil {
		t.Fatal(err)
	}
	projection, err := BuildUnresolvedProjection(FormatXead, receipt, classification)
	if err != nil {
		t.Fatal(err)
	}
	var mappingItem *UnresolvedSummary
	var semanticItem *UnresolvedSummary
	for i := range projection.Items {
		item := &projection.Items[i]
		if item.Code == "classification-mixed" && item.Kind == "Table" {
			mappingItem = item
		}
		if item.Code == "semantic-classification-unresolved" && item.Kind == "Table" {
			semanticItem = item
		}
	}
	if mappingItem == nil {
		t.Fatalf("mapping unresolved item missing: %+v", projection.Items)
	}
	if mappingItem.Extension != "" {
		t.Fatalf("unknown observation was assigned to mapped extension: %+v", mappingItem)
	}
	if semanticItem == nil || semanticItem.Extension != "reference.xead.relational-semantics" {
		t.Fatalf("mapped semantic candidate lost its extension: %+v", semanticItem)
	}
	if mappingItem.Count != 1 || semanticItem.Count != 1 {
		t.Fatalf("mixed mapped/unresolved counts are wrong: mapping=%+v semantic=%+v", mappingItem, semanticItem)
	}
}
