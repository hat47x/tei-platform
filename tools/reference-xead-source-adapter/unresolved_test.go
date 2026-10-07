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
	if counts["classification-unmapped|Mystery"] != 1 {
		t.Fatalf("mapping unresolved missing: %+v", projection.Items)
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
