package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"tei/internal/operations"
)

// `model locate` and `model catalog` read the same Workspace load, so a person
// looking at a diagram and an AI reading the location text can confirm one
// model version by fingerprint (RQ-036, design §19.8 MAP-21).
func TestModelLocateAgreesWithCatalogOnTheReferenceWorkspace(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var catalogOut bytes.Buffer
	if code, err := runModelCommand([]string{"catalog", "-workspace", root}, &catalogOut); err != nil || code != 0 {
		t.Fatalf("catalog: code=%d err=%v body=%s", code, err, catalogOut.String())
	}
	var catalog modelCLIResponse
	if err := json.Unmarshal(catalogOut.Bytes(), &catalog); err != nil || catalog.Result == nil {
		t.Fatalf("catalog decode: %v %s", err, catalogOut.String())
	}
	var target *operations.ModelCatalogEntry
	for i, entry := range catalog.Result.Entries {
		if entry.Subject.Kind == "semantic-relation" && !entry.Prelude && !entry.Ambiguous && entry.Parent != nil {
			target = &catalog.Result.Entries[i]
			break
		}
	}
	if target == nil {
		t.Skip("reference workspace declares no relation; nothing to locate")
	}

	var out bytes.Buffer
	code, err := runModelCommand([]string{"locate", "-workspace", root,
		"-subject-kind", target.Subject.Kind, "-subject-id", target.Subject.ID}, &out)
	if err != nil || code != 0 {
		t.Fatalf("locate: code=%d err=%v body=%s", code, err, out.String())
	}
	var response modelLocateCLIResponse
	if err := json.Unmarshal(out.Bytes(), &response); err != nil || !response.OK || response.Result == nil {
		t.Fatalf("locate decode: %v %s", err, out.String())
	}
	got := response.Result
	if response.SchemaVersion != "tei.cli.model-locate/v0" || got.Effect != "read-only" {
		t.Fatalf("envelope: %+v", response)
	}
	if got.WorldFingerprint != catalog.Result.WorldFingerprint ||
		got.SourceIndexFingerprint != catalog.Result.SourceIndexFingerprint ||
		got.ModelRevision != catalog.Result.ModelRevision {
		t.Fatalf("location and catalog disagree on the model version:\n location=%s %s %s\n catalog=%s %s %s",
			got.WorldFingerprint, got.SourceIndexFingerprint, got.ModelRevision,
			catalog.Result.WorldFingerprint, catalog.Result.SourceIndexFingerprint, catalog.Result.ModelRevision)
	}
	if len(got.Path) == 0 || got.Path[len(got.Path)-1].Subject != *target.Parent {
		t.Fatalf("path does not end at the catalog parent %v: %+v", *target.Parent, got.Path)
	}
	if got.Path[0].Parent != nil {
		t.Fatalf("path must start at a top-level element: %+v", got.Path[0])
	}
	found := false
	for _, link := range got.Outgoing {
		if link.Role == operations.LinkRoleRelationTarget && link.From == target.Subject && link.To.ID == target.Detail {
			found = true
		}
	}
	if !found {
		t.Fatalf("relation target %s missing from outgoing: %+v", target.Detail, got.Outgoing)
	}
}

func TestModelLocateFailuresAreMachineReadable(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	cases := []struct {
		name string
		args []string
		exit int
		code string
	}{
		{"missing subject", []string{"locate", "-workspace", root}, 2, "INVALID_ARGUMENT"},
		{"empty subject id", []string{"locate", "-workspace", root, "-subject-kind", "semantic", "-subject-id="}, 2, "INVALID_ARGUMENT"},
		{"limit zero", []string{"locate", "-workspace", root, "-subject-kind", "semantic", "-subject-id", "std", "-limit", "0"}, 2, "INVALID_ARGUMENT"},
		{"repeated subject id", []string{"locate", "-workspace", root, "-subject-kind", "semantic", "-subject-id", "std", "-subject-id", "std"}, 2, "INVALID_ARGUMENT"},
		{"unknown subject", []string{"locate", "-workspace", root, "-subject-kind", "semantic-data", "-subject-id", "no.such.data"}, 1, operations.LocationCodeSubjectNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			code, err := runModelCommand(tc.args, &out)
			if err != nil || code != tc.exit {
				t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
			}
			var response modelLocateCLIResponse
			if err := json.Unmarshal(out.Bytes(), &response); err != nil {
				t.Fatalf("not JSON: %v %s", err, out.String())
			}
			if response.OK || response.Error == nil || response.Error.Code != tc.code ||
				response.SchemaVersion != "tei.cli.model-locate/v0" {
				t.Fatalf("response=%+v", response)
			}
		})
	}
}
