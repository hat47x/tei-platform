package main

import (
	"os"
	"strings"
	"testing"

	"tei/tools/internal/rmtevidence"
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

func TestPublicEvidenceVocabularyMatchesSourceAdapterAndContract(t *testing.T) {
	contractBytes, err := os.ReadFile("../../reference/reference-model-transfer/transcription-contract-v0.json")
	if err != nil {
		t.Fatal(err)
	}

	for _, format := range []SourceFormat{FormatXead, FormatXeaf} {
		mapped, err := parseConstructMappings(format, contractBytes)
		if err != nil {
			t.Fatalf("%s contract: %v", format, err)
		}
		known := knownElements[format]
		publicKinds := rmtevidence.StandardConstructs(string(format))

		if len(mapped) != len(known) || len(publicKinds) != len(known) {
			t.Fatalf(
				"%s vocabulary size drift: contract=%d reader=%d evidence=%d",
				format,
				len(mapped),
				len(known),
				len(publicKinds),
			)
		}

		for kind, mapping := range mapped {
			if _, ok := known[kind]; !ok {
				t.Fatalf("%s contract kind %q is not known by the Source Adapter", format, kind)
			}
			if !rmtevidence.IsStandardConstruct(string(format), kind) {
				t.Fatalf("%s contract kind %q is not public typed-Evidence vocabulary", format, kind)
			}
			for _, classification := range mapping.Classification {
				if !rmtevidence.IsSupportedClassification(classification) {
					t.Fatalf(
						"%s/%s classification %q is not public typed-Evidence vocabulary",
						format,
						kind,
						classification,
					)
				}
			}
			if mapping.Extension != "" && !rmtevidence.IsPublicToken(mapping.Extension) {
				t.Fatalf("%s/%s extension %q is not a public token", format, kind, mapping.Extension)
			}
			for _, limit := range mapping.AuthorityLimits {
				if !rmtevidence.IsPublicToken(limit) {
					t.Fatalf("%s/%s authority limit %q is not a public token", format, kind, limit)
				}
			}
		}

		for kind := range known {
			if _, ok := mapped[kind]; !ok {
				t.Fatalf("%s reader kind %q has no transcription mapping", format, kind)
			}
			if !rmtevidence.IsStandardConstruct(string(format), kind) {
				t.Fatalf("%s reader kind %q is missing from typed-Evidence vocabulary", format, kind)
			}
		}
	}
}
