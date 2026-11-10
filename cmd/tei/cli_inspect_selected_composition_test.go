package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/inspectoradapter"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

func TestCLIInspectTraversesSelectedAuthoringProvider(t *testing.T) {
	const scope = "authoring-tool"
	contract := composition.Contract{
		ID: "tei.operations.inspect", Version: "0", Shape: "bounded-reference-v0",
	}
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{
		{ID: "inspector-A", Scope: scope, Offers: contract},
		{ID: "inspector-B", Scope: scope, Offers: contract},
	}, nil); err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select("inspector-B", scope)
	if err != nil {
		t.Fatal(err)
	}
	digest := "sha256:" + strings.Repeat("b", 64)
	binding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: selection.ProviderID, ArtifactDigest: digest, Generation: 1,
	}
	calls := 0
	actual, err := inspectoradapter.Bind(registry, selection, binding, contract,
		inspectoradapter.Registration{
			ProviderID: "inspector-B", ArtifactDigest: digest,
			Inspect: func(subject sourceindex.Subject) (operations.SubjectInspection, error) {
				calls++
				return operations.SubjectInspection{
					SchemaVersion:          "tei.operations.inspect/v0",
					Effect:                 "read-only",
					Subject:                subject,
					Canonical:              json.RawMessage(`{"id":"sales.order"}`),
					WorldFingerprint:       "fp-world",
					SourceIndexFingerprint: "fp-index",
				}, nil
			},
		}, func(composition.Binding) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	// Empty Workspace has no TEI manifest. The CLI can only succeed by
	// invoking the instance selected by existing Composition, not built-in.
	root := t.TempDir()
	var out bytes.Buffer
	code, err := inspectWorkspaceJSONWithService(&out, root, "", "semantic-data", "sales.order", 0, actual)
	if err != nil || code != 0 || calls != 1 {
		t.Fatalf("selected-B CLI call: code=%d err=%v calls=%d body=%s", code, err, calls, out.String())
	}
	var report cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.OK || report.Result == nil || report.Result.Subject.ID != "sales.order" {
		t.Fatalf("wrong CLI selected result: %+v", report)
	}
	// A selected implementation cannot substitute for B even with the same
	// response. A refused binding must not execute either provider.
	_, err = inspectoradapter.Bind(registry, selection, binding, contract,
		inspectoradapter.Registration{
			ProviderID: "inspector-A", ArtifactDigest: digest,
			Inspect: func(s sourceindex.Subject) (operations.SubjectInspection, error) {
				calls++
				return operations.SubjectInspection{Subject: s}, nil
			},
		}, func(composition.Binding) error { return nil })
	if err == nil || calls != 1 {
		t.Fatalf("selected B admitted A: err=%v calls=%d", err, calls)
	}
	// Revocation must happen before entry to the selected implementation.
	// The CLI receives a classified failure and does not re-run the builtin.
	registryVersion := selection.Revision
	if _, err := registry.Apply(registryVersion, nil, []string{"inspector-B"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	code, err = inspectWorkspaceJSONWithService(&out, root, "", "semantic-data", "sales.order", 0, actual)
	if err != nil || code == 0 || calls != 1 {
		t.Fatalf("stale selected binding reached provider: code=%d err=%v calls=%d", code, err, calls)
	}
	var failed cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &failed); err != nil {
		t.Fatal(err)
	}
	if failed.OK || failed.Error == nil || failed.Result != nil {
		t.Fatalf("stale Inspector returned success: %+v", failed)
	}
}
