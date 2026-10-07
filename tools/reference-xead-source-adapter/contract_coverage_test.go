package main

import (
	"strings"
	"testing"
)

func TestClassificationCoverageSeparatesAccountingFromAcceptance(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System><Table/><Mystery/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	receipt := PublicReceipt(doc)
	contract := []byte(`{"constructMappings":{"xead":{"System":{"classification":["source-metadata"]},"Table":{"classification":["canonical-candidate"]}}}}`)
	coverage, err := CheckClassificationCoverage(FormatXead, receipt, contract)
	if err != nil {
		t.Fatal(err)
	}
	if coverage.Complete || len(coverage.UnmappedObservedKinds) != 1 || coverage.UnmappedObservedKinds[0] != "Mystery" {
		t.Fatalf("coverage=%+v", coverage)
	}
	if len(coverage.AuthorityLimits) == 0 {
		t.Fatal("classification coverage lost authority limits")
	}
}

func TestClassificationCoveragePassesWhenEveryObservedKindHasMapping(t *testing.T) {
	doc, err := Observe(FormatXeaf, []byte(`<System><Function/><Script/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	contract := []byte(`{"constructMappings":{"xeaf":{"System":{"classification":["source-metadata"]},"Function":{"classification":["realization"]},"Script":{"classification":["native-preserved"]},"Unused":{"classification":["unresolved"]}}}}`)
	coverage, err := CheckClassificationCoverage(FormatXeaf, PublicReceipt(doc), contract)
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
	doc, err := Observe(FormatXead, []byte(`<System xmlns:x="urn:extension"><x:Table/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	contract := []byte(`{"constructMappings":{"xead":{"System":{"classification":["source-metadata"]},"Table":{"classification":["canonical-candidate"]}}}}`)
	coverage, err := CheckClassificationCoverage(FormatXead, PublicReceipt(doc), contract)
	if err != nil {
		t.Fatal(err)
	}
	if coverage.Complete || coverage.UnknownObservedElements != 1 {
		t.Fatalf("namespaced lookalike was hidden by local-name mapping: %+v", coverage)
	}
}
