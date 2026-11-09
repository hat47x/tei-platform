package main

import (
	"strings"
	"testing"
)

func sourceProfileContractFixture(format SourceFormat) []byte {
	return []byte(`{"sourceProfiles":{"` + string(format) + `":{"rootKind":"System","supportedFormatVersions":["1.2"]}},"constructMappings":{"` + string(format) + `":{"System":{"classification":["source-metadata"]}}}}`)
}

func TestSourceObservationAllowsUnsupportedFormatVersionForForensics(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="9.9"><Table/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Nodes[0].Attributes["FormatVersion"] != "9.9" {
		t.Fatalf("unsupported format version was not preserved in local observation: %+v", doc.Nodes[0])
	}
}

func TestClassificationRejectsUnsupportedFormatVersion(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="1.3"><Table/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = BuildClassificationProjection(FormatXead, doc, sourceProfileContractFixture(FormatXead))
	if err == nil || !strings.Contains(err.Error(), "FormatVersion 1.3 is not supported") {
		t.Fatalf("unsupported source version was accepted: %v", err)
	}
}

func TestClassificationRejectsMissingFormatVersion(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System><Table/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = BuildClassificationProjection(FormatXead, doc, sourceProfileContractFixture(FormatXead))
	if err == nil || !strings.Contains(err.Error(), "has no FormatVersion") {
		t.Fatalf("missing source version was accepted: %v", err)
	}
}

func TestClassificationRejectsWrongRootKind(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<Table FormatVersion="1.2"/>`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = BuildClassificationProjection(FormatXead, doc, sourceProfileContractFixture(FormatXead))
	if err == nil || !strings.Contains(err.Error(), "does not match xead source profile root System") {
		t.Fatalf("wrong root kind was accepted: %v", err)
	}
}

func TestClassificationRejectsNamespacedRootLookalike(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<x:System xmlns:x="urn:extension" FormatVersion="1.2"/>`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = BuildClassificationProjection(FormatXead, doc, sourceProfileContractFixture(FormatXead))
	if err == nil || !strings.Contains(err.Error(), "not a known unqualified source construct") {
		t.Fatalf("namespaced root lookalike was accepted: %v", err)
	}
}

func TestClassificationReportsAcceptedSourceFormatVersion(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2"/>`))
	if err != nil {
		t.Fatal(err)
	}
	projection, err := BuildClassificationProjection(FormatXead, doc, sourceProfileContractFixture(FormatXead))
	if err != nil {
		t.Fatal(err)
	}
	if projection.SourceFormatVersion != "1.2" {
		t.Fatalf("source format version=%q want=1.2", projection.SourceFormatVersion)
	}

	coverage, err := CheckClassificationCoverage(FormatXead, doc, sourceProfileContractFixture(FormatXead))
	if err != nil {
		t.Fatal(err)
	}
	if coverage.SourceFormatVersion != "1.2" {
		t.Fatalf("coverage source format version=%q want=1.2", coverage.SourceFormatVersion)
	}
}

func TestSourceProfileContractRequiresDeclaredProfile(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2"/>`))
	if err != nil {
		t.Fatal(err)
	}
	contract := []byte(`{"constructMappings":{"xead":{"System":{"classification":["source-metadata"]}}}}`)
	_, err = BuildClassificationProjection(FormatXead, doc, contract)
	if err == nil || !strings.Contains(err.Error(), "has no xead source profile") {
		t.Fatalf("contract without source profile was accepted: %v", err)
	}
}
