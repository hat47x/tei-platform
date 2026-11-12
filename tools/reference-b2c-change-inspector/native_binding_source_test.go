package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// b0NativeSourcePin is a test-local declared binding. It is not an API,
// inferred native dependency, accepted meaning, current grant or write ticket.
type b0NativeSourcePin struct {
	ConsumerID        string
	WorkspaceName     string
	WorkspaceRoot     string
	Subject           sourceindex.Subject
	SourcePath        string
	SourceFingerprint string
}

// b0CheckNativePin consumes one actual Load and its strict Source Index.
// It does not scan the browser, filesystem or any Provider independently.
func b0CheckNativePin(root string, loaded workspace.Result, pin b0NativeSourcePin) (operations.SubjectInspection, error) {
	if pin.ConsumerID == "" || pin.WorkspaceName == "" || pin.WorkspaceRoot == "" || pin.SourcePath == "" || pin.SourceFingerprint == "" {
		return operations.SubjectInspection{}, errors.New("explicit native binding is incomplete")
	}
	if filepath.Clean(root) != filepath.Clean(pin.WorkspaceRoot) {
		return operations.SubjectInspection{}, errors.New("native binding targets another Workspace root")
	}
	// The strict index check binds this Load to its original real root.
	// Descriptive Workspace identity and byte hashes alone are not globally unique.
	idx, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		return operations.SubjectInspection{}, err
	}
	inspection, err := operations.InspectionFromLoad(loaded, idx, pin.Subject)
	if err != nil {
		return operations.SubjectInspection{}, err
	}
	if inspection.Workspace == nil || inspection.Workspace.Name != pin.WorkspaceName {
		return operations.SubjectInspection{}, fmt.Errorf("native binding workspace does not match loaded Workspace")
	}
	if inspection.SourceStatus != "indexed" || len(inspection.Sources.References) != 1 {
		return operations.SubjectInspection{}, fmt.Errorf("native binding has no unique Source occurrence")
	}
	ref := inspection.Sources.References[0]
	if ref.Subject != pin.Subject || ref.ArtifactPath != pin.SourcePath {
		return operations.SubjectInspection{}, fmt.Errorf("native binding Source occurrence differs")
	}
	if ref.Range.Start.Offset >= ref.Range.End.Offset {
		return operations.SubjectInspection{}, fmt.Errorf("native binding requires a nonempty Source anchor")
	}
	matches := 0
	for _, a := range inspection.Sources.Artifacts {
		if a.Path == pin.SourcePath {
			matches++
			if a.Fingerprint != pin.SourceFingerprint {
				return operations.SubjectInspection{}, fmt.Errorf("native binding Source bytes are stale")
			}
		}
	}
	if matches != 1 {
		return operations.SubjectInspection{}, fmt.Errorf("native binding requires exactly one matching Source artifact")
	}
	return inspection, nil
}

func b0SourceDigest(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// TestB0OneNativeFieldBoundToActualManagedSource exercises one *declared*
// native field -> real Canonical Subject + source occurrence + byte pin.
// Two separate Loads may be compared, never misrepresented as atomic.
func TestB0OneNativeFieldBoundToActualManagedSource(t *testing.T) {
	root := t.TempDir()
	manifest := "workspace: native_pdp_source_case\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	b0Write(t, root, b0OrderA)
	read := func() (workspace.Result, sourceindex.Index) {
		t.Helper()
		loaded, err := workspace.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		idx, err := workspace.BuildSourceIndexFromLoad(root, loaded)
		if err != nil {
			t.Fatal(err)
		}
		return loaded, idx
	}
	loadedA, indexA := read()
	pin := b0NativeSourcePin{
		ConsumerID:        "native-pdp.order-id",
		WorkspaceName:     "native_pdp_source_case",
		WorkspaceRoot:     root,
		Subject:           sourceindex.Subject{Kind: "semantic-attribute", ID: "sales.order.order_id"},
		SourcePath:        b0SourcePath,
		SourceFingerprint: b0SourceDigest([]byte(b0OrderA)),
	}
	inspectionA, err := b0CheckNativePin(root, loadedA, pin)
	if err != nil {
		t.Fatalf("declared binding did not resolve one real Source: %v", err)
	}
	ref := inspectionA.Sources.References[0]
	if ref.Subject != pin.Subject || ref.ArtifactPath != b0SourcePath ||
		len(inspectionA.Canonical) == 0 || len(inspectionA.AuthorityLimits) == 0 {
		t.Fatalf("binding failed independent identity/authority checks: %+v", inspectionA)
	}

	// A native field is declared, not parsed from DOM or inferred from SQL.
	// New Source bytes require explicit rebinding; same subject ID is not enough.
	b0Write(t, root, b0OrderB)
	loadedB, indexB := read()
	if _, err := b0CheckNativePin(root, loadedB, pin); err == nil {
		t.Fatal("old byte-pinned native binding survived Source drift")
	}
	rebinding := pin
	rebinding.SourceFingerprint = b0SourceDigest([]byte(b0OrderB))
	if _, err := b0CheckNativePin(root, loadedB, rebinding); err != nil {
		t.Fatalf("explicit rebinding to one real Source failed: %v", err)
	}

	sourceNode := "semantic-attribute:sales.order.order_id"
	// This declaration is a hypothesis supplied by the caller, NOT a verified
	// native dependency graph or evidence of an actual Consumer invocation.
	declared := input{
		Schema:              inputSchema,
		Scope:               pin.ConsumerID,
		BaseRevision:        sourceindex.ModelRevision(indexA),
		CandidateRevision:   sourceindex.ModelRevision(indexB),
		DeclaredScopeClosed: false,
		Nodes:               []node{{ID: sourceNode, Layer: "business"}, {ID: pin.ConsumerID, Layer: "presentation"}},
		Edges:               []edge{{From: sourceNode, To: pin.ConsumerID}},
		Changed:             []string{sourceNode},
		Required:            []string{sourceNode, pin.ConsumerID},
		Unobserved:          []string{"native-pdp.dynamic-import-not-inspected"},
	}
	report, err := inspect(declared)
	if err != nil {
		t.Fatal(err)
	}
	if report.Scope != "native-pdp.order-id" || report.Coverage != "partial-or-unverified" ||
		report.Disposition != "known-interaction-business-or-release-impact" ||
		!reflect.DeepEqual(report.Lanes, []string{"C1", "C3"}) ||
		!reflect.DeepEqual(report.Impacted, []string{"native-pdp.order-id", sourceNode}) ||
		!reflect.DeepEqual(report.Unresolved, []string{"native-pdp.dynamic-import-not-inspected"}) ||
		len(report.AuthorityLimits) == 0 {
		t.Fatalf("bounded native binding impact does not match fixed expected: %+v", report)
	}
}

func TestB0NativeBindingRefusesWrongWorkspaceSubjectAndOccurrence(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace: native_binding_negative\nformatVersion: 1\nsources:\n  - semantic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b0Write(t, root, b0OrderA)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	base := b0NativeSourcePin{
		ConsumerID:        "native-pdp.order-id",
		WorkspaceName:     "native_binding_negative",
		WorkspaceRoot:     root,
		Subject:           sourceindex.Subject{Kind: "semantic-attribute", ID: "sales.order.order_id"},
		SourcePath:        b0SourcePath,
		SourceFingerprint: b0SourceDigest([]byte(b0OrderA)),
	}
	if _, err := b0CheckNativePin(root, loaded, base); err != nil {
		t.Fatalf("positive control failed: %v", err)
	}
	// Identical manifest, semantic Source and bytes in another root do not
	// establish that a declared Native binding targets that other Workspace.
	otherRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(otherRoot, "tei.workspace.yaml"), []byte("workspace: native_binding_negative\nformatVersion: 1\nsources:\n  - semantic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b0Write(t, otherRoot, b0OrderA)
	other, err := workspace.Load(otherRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b0CheckNativePin(otherRoot, other, base); err == nil {
		t.Fatal("byte-identical separate Workspace root silently reused old Native binding")
	}
	cases := []struct {
		name   string
		mutate func(*b0NativeSourcePin)
	}{
		{"wrong-workspace", func(p *b0NativeSourcePin) { p.WorkspaceName = "other_workspace" }},
		{"wrong-workspace-root", func(p *b0NativeSourcePin) { p.WorkspaceRoot = t.TempDir() }},
		{"wrong-source-path", func(p *b0NativeSourcePin) { p.SourcePath = "semantic/other/order.tei" }},
		{"stale-bytes", func(p *b0NativeSourcePin) { p.SourceFingerprint = b0SourceDigest([]byte("other")) }},
		{"missing-attribute", func(p *b0NativeSourcePin) { p.Subject.ID = "sales.order.missing" }},
		{"wrong-kind", func(p *b0NativeSourcePin) { p.Subject.Kind = "semantic-data" }},
		{"missing-consumer", func(p *b0NativeSourcePin) { p.ConsumerID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.mutate(&p)
			_, err := b0CheckNativePin(root, loaded, p)
			if err == nil {
				t.Fatalf("unqualified native binding passed: %+v", p)
			}
			if strings.TrimSpace(err.Error()) == "" {
				t.Fatal("missing rejection diagnostic")
			}
		})
	}
}
