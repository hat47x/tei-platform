package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/xeadsource"
)

func capabilityReferenceRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve capability Reference source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func sourceAdapterTranscriptionContract() []byte {
	return []byte(`{
  "sourceProfiles": {
    "xead": {
      "rootKind": "System",
      "supportedFormatVersions": ["1.2"]
    },
    "xeaf": {
      "rootKind": "System",
      "supportedFormatVersions": ["1.2"]
    }
  },
  "constructMappings": {
    "xead": {
      "System": {
        "classification": ["source-metadata"],
        "extension": "reference.source"
      },
      "Table": {
        "classification": ["data"],
        "extension": "reference.data",
        "authorityLimits": ["source-table-is-not-canonical-data-model"]
      }
    },
    "xeaf": {
      "System": {
        "classification": ["source-metadata"],
        "extension": "reference.source"
      },
      "Function": {
        "classification": ["interaction"],
        "extension": "reference.interaction",
        "authorityLimits": ["source-function-is-not-canonical-interaction"]
      }
    }
  }
}`)
}

func sourceAdapterRegistryFixture(t *testing.T) (
	Contract,
	Provider,
	*SourceAdapterRegistry,
) {
	t.Helper()
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		referenceXeadReaderImplementationID,
	)
	implementation, err := ReferenceXeadSourceAdapterImplementation(provider)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewSourceAdapterRegistry([]SourceAdapterImplementation{
		implementation,
	})
	if err != nil {
		t.Fatal(err)
	}
	return contract, provider, registry
}

func sourceAdapterResolveInput(
	contract Contract,
	provider Provider,
) ResolveInput {
	return ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence: []ConformanceEvidence{
			evidenceFor(provider, contract, "source-adapter-conformance"),
		},
	}
}

func findSourceNode(root xeadsource.Node, name string) (xeadsource.Node, bool) {
	if root.Name == name {
		return root, true
	}
	for _, child := range root.Children {
		if found, ok := findSourceNode(child, name); ok {
			return found, true
		}
	}
	return xeadsource.Node{}, false
}

func hasCandidate(
	values []xeadsource.Candidate,
	name string,
	unresolved bool,
) bool {
	for _, value := range values {
		if value.SourceName == name && value.Unresolved == unresolved {
			return true
		}
	}
	return false
}

func hasSourceAdapterLimit(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestCapabilityResolutionInvokesRealSourceReaderForXeadAndXeaf(t *testing.T) {
	contract, provider, registry := sourceAdapterRegistryFixture(t)
	tests := []struct {
		name        string
		format      xeadsource.Format
		source      string
		unknownName string
		unknownAttr string
		unknownText string
	}{
		{
			name:        "xead",
			format:      xeadsource.FormatXead,
			source:      `<System FormatVersion="1.2"><Table ID="T1"><VendorThing Secret="kept">source text</VendorThing></Table></System>`,
			unknownName: "VendorThing",
			unknownAttr: "kept",
			unknownText: "source text",
		},
		{
			name:        "xeaf",
			format:      xeadsource.FormatXeaf,
			source:      `<System FormatVersion="1.2"><Function ID="F1"><VendorAction Secret="kept">runtime text</VendorAction></Function></System>`,
			unknownName: "VendorAction",
			unknownAttr: "kept",
			unknownText: "runtime text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execution, err := registry.ResolveAndInvoke(
				sourceAdapterResolveInput(contract, provider),
				SourceAdapterRequest{
					Format:                tt.format,
					SourceBytes:           []byte(tt.source),
					TranscriptionContract: sourceAdapterTranscriptionContract(),
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			if execution.Resolution.Contract.ID != sourceAdapterCapabilityContractID ||
				execution.Resolution.Provider != provider {
				t.Fatalf("resolution=%+v", execution.Resolution)
			}
			if !strings.HasPrefix(execution.SourceArtifactDigest, "sha256:") ||
				execution.SourceArtifactDigest != "sha256:"+execution.Inspection.Document.SHA256 {
				t.Fatalf(
					"source digest=%q document=%q",
					execution.SourceArtifactDigest,
					execution.Inspection.Document.SHA256,
				)
			}
			if execution.Inspection.Document.Format != tt.format {
				t.Fatalf("format=%q want=%q", execution.Inspection.Document.Format, tt.format)
			}
			if len(execution.Inspection.Document.UnknownNames) != 1 ||
				execution.Inspection.Document.UnknownNames[0] != tt.unknownName {
				t.Fatalf("unknownNames=%+v", execution.Inspection.Document.UnknownNames)
			}
			node, ok := findSourceNode(execution.Inspection.Document.Root, tt.unknownName)
			if !ok {
				t.Fatalf("unknown node %q was dropped", tt.unknownName)
			}
			if node.Known {
				t.Fatalf("unknown node was promoted to known: %+v", node)
			}
			var secret string
			for _, attribute := range node.Attributes {
				if attribute.Name == "Secret" {
					secret = attribute.Value
				}
			}
			if secret != tt.unknownAttr || node.Text != tt.unknownText {
				t.Fatalf("unknown source detail was lost: node=%+v", node)
			}
			if !hasCandidate(
				execution.Inspection.Candidates,
				tt.unknownName,
				true,
			) {
				t.Fatalf(
					"unknown source candidate was not unresolved: %+v",
					execution.Inspection.Candidates,
				)
			}
			for _, limit := range []string{
				"source-inspection-is-not-canonical-authority",
				"capability-resolution-is-not-canonical-acceptance",
				"authoring-tool-capability-is-not-application-runtime-authority",
				"source-adapter-execution-is-read-only-observation",
				"source-adapter-inspection-does-not-replace-rmt-publication-receipt",
				"source-adapter-inspection-is-local-private-evidence",
			} {
				if !hasSourceAdapterLimit(execution.AuthorityLimits, limit) {
					t.Fatalf("authority limit %q missing: %+v", limit, execution.AuthorityLimits)
				}
			}
		})
	}
}

func TestSourceAdapterConsumerBindsResultToExactSourceBytes(t *testing.T) {
	contract, provider, registry := sourceAdapterRegistryFixture(t)
	request := SourceAdapterRequest{
		Format:                xeadsource.FormatXead,
		SourceBytes:           []byte(`<System FormatVersion="1.2"><Table ID="T1"/></System>`),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	}
	left, err := registry.ResolveAndInvoke(
		sourceAdapterResolveInput(contract, provider),
		request,
	)
	if err != nil {
		t.Fatal(err)
	}

	request.SourceBytes = append(append([]byte(nil), request.SourceBytes...), ' ')
	right, err := registry.ResolveAndInvoke(
		sourceAdapterResolveInput(contract, provider),
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if left.SourceArtifactDigest == right.SourceArtifactDigest {
		t.Fatal("different exact source bytes produced the same source artifact identity")
	}
	if left.Resolution.ResolutionFingerprint != right.Resolution.ResolutionFingerprint {
		t.Fatal("source byte change unexpectedly changed capability resolution identity")
	}
}

func TestSourceAdapterConsumerRequiresCurrentConformanceResolution(t *testing.T) {
	contract, provider, registry := sourceAdapterRegistryFixture(t)
	input := sourceAdapterResolveInput(contract, provider)
	input.Evidence = nil

	_, err := registry.ResolveAndInvoke(input, SourceAdapterRequest{
		Format:                xeadsource.FormatXead,
		SourceBytes:           []byte(`<System FormatVersion="1.2"/>`),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	})
	if err == nil || !strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("Source Adapter executed without Conformance Evidence: %v", err)
	}
}

func TestSourceAdapterConsumerRejectsRuntimeScope(t *testing.T) {
	contract, provider, registry := sourceAdapterRegistryFixture(t)
	input := sourceAdapterResolveInput(contract, provider)
	input.Scope = ScopeApplicationRuntime

	_, err := registry.ResolveAndInvoke(input, SourceAdapterRequest{
		Format:                xeadsource.FormatXead,
		SourceBytes:           []byte(`<System FormatVersion="1.2"/>`),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	})
	if err == nil || !strings.Contains(err.Error(), "authoring-tool") {
		t.Fatalf("Source Adapter executed with Runtime Authority: %v", err)
	}
}

func TestSourceAdapterConsumerRejectsResolvedButUninstalledArtifact(t *testing.T) {
	contract, provider, registry := sourceAdapterRegistryFixture(t)
	other := provider
	other.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)

	_, err := registry.ResolveAndInvoke(
		sourceAdapterResolveInput(contract, other),
		SourceAdapterRequest{
			Format:                xeadsource.FormatXead,
			SourceBytes:           []byte(`<System FormatVersion="1.2"/>`),
			TranscriptionContract: sourceAdapterTranscriptionContract(),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("uninstalled resolved artifact was executed: %v", err)
	}
}

func TestReferenceSourceAdapterImplementationRejectsDifferentImplementationIdentity(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		"other.reader",
	)
	if _, err := ReferenceXeadSourceAdapterImplementation(provider); err == nil {
		t.Fatal("different implementation identity was registered as Reference Xead reader")
	}
}

func TestSourceAdapterRegistryRejectsDuplicatePhysicalImplementation(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		referenceXeadReaderImplementationID,
	)
	implementation, err := ReferenceXeadSourceAdapterImplementation(provider)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewSourceAdapterRegistry([]SourceAdapterImplementation{
		implementation,
		implementation,
	}); err == nil || !strings.Contains(err.Error(), "registered more than once") {
		t.Fatalf("duplicate physical implementation was accepted: %v", err)
	}
}


func TestSourceAdapterConsumerRejectsUnsupportedSourceProfile(t *testing.T) {
	contract, provider, registry := sourceAdapterRegistryFixture(t)
	_, err := registry.ResolveAndInvoke(
		sourceAdapterResolveInput(contract, provider),
		SourceAdapterRequest{
			Format:                xeadsource.FormatXead,
			SourceBytes:           []byte(`<System FormatVersion="9.9"><Table/></System>`),
			TranscriptionContract: sourceAdapterTranscriptionContract(),
		},
	)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unsupported source profile reached semantic inspection: %v", err)
	}
}


func TestCapabilitySourceAdapterUsesCurrentReferenceTranscriptionContract(t *testing.T) {
	contract, provider, registry := sourceAdapterRegistryFixture(t)
	root := capabilityReferenceRepositoryRoot(t)
	contractBytes, err := os.ReadFile(filepath.Join(
		root,
		"reference",
		"reference-model-transfer",
		"transcription-contract-v0.json",
	))
	if err != nil {
		t.Fatal(err)
	}

	execution, err := registry.ResolveAndInvoke(
		sourceAdapterResolveInput(contract, provider),
		SourceAdapterRequest{
			Format: xeadsource.FormatXead,
			SourceBytes: []byte(
				`<System FormatVersion="1.2"><Table ID="T1"><TableField ID="F1"/></Table></System>`,
			),
			TranscriptionContract: contractBytes,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if execution.Inspection.TranscriptionContractFingerprint == "" ||
		execution.Inspection.ClassificationPolicyFingerprint == "" {
		t.Fatalf("current Reference contract provenance was not preserved: %+v", execution.Inspection)
	}

	var table *xeadsource.Candidate
	for i := range execution.Inspection.Candidates {
		if execution.Inspection.Candidates[i].SourceName == "Table" {
			table = &execution.Inspection.Candidates[i]
			break
		}
	}
	if table == nil {
		t.Fatal("Table candidate is missing")
	}
	if table.Unresolved ||
		table.Extension != "reference.xead.relational-semantics" {
		t.Fatalf("current Reference Table classification was not applied: %+v", *table)
	}
}
