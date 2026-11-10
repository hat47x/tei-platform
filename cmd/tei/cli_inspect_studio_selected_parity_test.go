package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/inspectoradapter"
	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/studiohost"
)

// Both in-process consumers must enter the same exact-selected provider.
// This is not evidence that either production binary selects the provider.
func TestCLIAndStudioUseSameSelectedInspectorImplementation(t *testing.T) {
	const scope = "authoring-tool"
	root := filepath.Join("..", "..", "reference", "workspace")
	contract := composition.Contract{ID: "tei.operations.inspect", Version: "0", Shape: "bounded-reference-v0"}
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
	selected, err := inspectoradapter.Bind(registry, selection, binding, contract,
		inspectoradapter.Registration{
			ProviderID: "inspector-B", ArtifactDigest: digest,
			Inspect: func(subject sourceindex.Subject) (operations.SubjectInspection, error) {
				calls++
				return operations.InspectSubject(root, subject)
			},
		}, func(composition.Binding) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	studio, err := studiohost.NewReadOnlyWithFactory([]string{root}, func(string) (studiohost.RootServices, error) {
		return studiohost.RootServices{Inspection: selected}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	subjects := []sourceindex.Subject{
		{Kind: "semantic-data", ID: "sales.order"},
		{Kind: "semantic-validation", ID: "sales.order_line.line_number_positive"},
		{Kind: "semantic-relation", ID: "sales.order.lines"},
		{Kind: "semantic-capability", ID: "sales.order.submit"},
		{Kind: "application", ID: "order_entry"},
		{Kind: "application-pattern", ID: "order_entry.submit_order"},
	}
	for _, subject := range subjects {
		t.Run(subject.Kind, func(t *testing.T) {
			before := calls
			var out bytes.Buffer
			code, err := inspectWorkspaceJSONWithService(&out, root, "", subject.Kind, subject.ID, 0, selected)
			if err != nil || code != 0 {
				t.Fatalf("selected CLI failed: exit=%d err=%v body=%s", code, err, out.String())
			}
			var report cliInspectionReport
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			got := studio.InspectSubject("workspace-1", subject.Kind, subject.ID)
			if !report.OK || report.Result == nil || report.Error != nil ||
				!got.OK || got.Value == nil || got.Failure != nil ||
				!reflect.DeepEqual(*report.Result, *got.Value) || calls != before+2 {
				t.Fatalf("two consumers did not call selected-B exactly once each: calls=%d cli=%+v studio=%+v",
					calls-before, report, got)
			}
		})
	}
	// A stale Registry epoch cannot silently fall back to the built-in path
	// on either consumer, including when the underlying operation is healthy.
	if _, err := registry.Apply(selection.Revision, nil, []string{"inspector-B"}); err != nil {
		t.Fatal(err)
	}
	before := calls
	var out bytes.Buffer
	first := subjects[0]
	exit, err := inspectWorkspaceJSONWithService(&out, root, "", first.Kind, first.ID, 0, selected)
	if err != nil || exit == 0 {
		t.Fatalf("stale CLI selected service admitted: exit=%d err=%v", exit, err)
	}
	got := studio.InspectSubject("workspace-1", first.Kind, first.ID)
	if got.OK || got.Failure == nil || calls != before {
		t.Fatalf("stale selected provider invoked through Studio: calls=%d result=%+v", calls-before, got)
	}
}
