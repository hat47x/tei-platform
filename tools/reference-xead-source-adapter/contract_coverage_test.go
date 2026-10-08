package main

import (
	"strings"
	"testing"
)

func TestClassificationCoverageSeparatesAccountingFromAcceptance(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2"><Table/><Mystery/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	contract := []byte(`{"sourceProfiles":{"xead":{"rootKind":"System","supportedFormatVersions":["1.2"]}},"constructMappings":{"xead":{"System":{"classification":["source-metadata"]},"Table":{"classification":["canonical-candidate"]}}}}`)
	coverage, err := CheckClassificationCoverage(FormatXead, doc, contract)
	if err != nil {
		t.Fatal(err)
	}
	if coverage.Complete || coverage.UnmappedObservedKindCount != 1 || len(coverage.UnmappedObservedKinds) != 0 {
		t.Fatalf("coverage leaked arbitrary unknown kind or lost unresolved accounting: %+v", coverage)
	}
	if len(coverage.AuthorityLimits) == 0 {
		t.Fatal("classification coverage lost authority limits")
	}
}

func TestClassificationCoveragePassesWhenEveryObservedKindHasMapping(t *testing.T) {
	doc, err := Observe(FormatXeaf, []byte(`<System FormatVersion="1.2"><Function/><Script/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	contract := []byte(`{"sourceProfiles":{"xeaf":{"rootKind":"System","supportedFormatVersions":["1.2"]}},"constructMappings":{"xeaf":{"System":{"classification":["source-metadata"]},"Function":{"classification":["realization"]},"Script":{"classification":["native-preserved"]},"Unused":{"classification":["unresolved"]}}}}`)
	coverage, err := CheckClassificationCoverage(FormatXeaf, doc, contract)
	if err != nil {
		t.Fatal(err)
	}
	if !coverage.Complete {
		t.Fatalf("coverage=%+v", coverage)
	}
	if strings.Join(coverage.UnobservedMappedKinds, ",") != "Unused" {
		t.Fatalf("unobserved=%v", coverage.UnobservedMappedKinds)
	}
}

func TestClassificationCoverageRejectsNamespacedLookalike(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2" xmlns:x="urn:extension"><x:Table/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	contract := []byte(`{"sourceProfiles":{"xead":{"rootKind":"System","supportedFormatVersions":["1.2"]}},"constructMappings":{"xead":{"System":{"classification":["source-metadata"]},"Table":{"classification":["canonical-candidate"]}}}}`)
	coverage, err := CheckClassificationCoverage(FormatXead, doc, contract)
	if err != nil {
		t.Fatal(err)
	}
	if coverage.Complete || coverage.UnknownObservedElements != 1 {
		t.Fatalf("namespaced lookalike was hidden by local-name mapping: %+v", coverage)
	}
}
