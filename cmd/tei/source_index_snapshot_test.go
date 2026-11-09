package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// The CLI projection and transport-neutral operation must expose identical
// meaning and provenance for one selected subject. This test is not evidence
// of production runtime performance or repository-wide qualification.
func TestSourceIndexInspectionReusesOperationSemantics(t *testing.T) {
	root := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	subject := sourceindex.Subject{Kind: "query", ID: "order_entry.recent_orders"}
	raw, _, err := buildSourceIndexInspection(root, subject.Kind, subject.ID)
	if err != nil {
		t.Fatal(err)
	}
	cli, ok := raw.(sourceIndexSubjectValue)
	if !ok {
		t.Fatalf("unexpected CLI projection: %T", raw)
	}
	typed, err := operations.TraceSubjectSources(root, subject)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cli.SubjectSources, typed) {
		t.Fatalf("CLI projection differs from typed operation:\nCLI=%+v\ntyped=%+v", cli.SubjectSources, typed)
	}
	if cli.Workspace == nil || cli.Workspace.Name == "" {
		t.Fatalf("CLI projection lost the Workspace identity: %+v", cli)
	}
}
