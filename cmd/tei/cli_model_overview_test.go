package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"tei/internal/operations"
)

// `model overview` reads the same Workspace load as `model catalog`; its
// sections are the catalog's top-level entries, and its validation counts add
// up to the catalog's verdict once unattributed findings are included
// (RQ-036, design §19.8 G5).
func TestModelOverviewAgreesWithCatalogOnTheReferenceWorkspace(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var catalogOut bytes.Buffer
	if code, err := runModelCommand([]string{"catalog", "-workspace", root}, &catalogOut); err != nil || code != 0 {
		t.Fatalf("catalog: code=%d err=%v body=%s", code, err, catalogOut.String())
	}
	var catalog modelCLIResponse
	if err := json.Unmarshal(catalogOut.Bytes(), &catalog); err != nil || catalog.Result == nil {
		t.Fatalf("catalog decode: %v", err)
	}

	var out bytes.Buffer
	if code, err := runModelCommand([]string{"overview", "-workspace", root}, &out); err != nil || code != 0 {
		t.Fatalf("overview: code=%d err=%v body=%s", code, err, out.String())
	}
	var response modelOverviewCLIResponse
	if err := json.Unmarshal(out.Bytes(), &response); err != nil || !response.OK || response.Result == nil {
		t.Fatalf("overview decode: %v %s", err, out.String())
	}
	got := response.Result
	if response.SchemaVersion != "tei.cli.model-overview/v0" || got.Effect != "read-only" {
		t.Fatalf("envelope: %+v", response)
	}
	if got.WorldFingerprint != catalog.Result.WorldFingerprint ||
		got.SourceIndexFingerprint != catalog.Result.SourceIndexFingerprint ||
		got.ModelRevision != catalog.Result.ModelRevision {
		t.Fatal("overview and catalog disagree on the model version")
	}
	roots := 0
	for _, entry := range catalog.Result.Entries {
		if entry.Parent == nil {
			roots++
		}
	}
	if got.SectionTotal != roots {
		t.Fatalf("sectionTotal=%d, catalog has %d top-level entries", got.SectionTotal, roots)
	}
	errors := got.Totals.ValidationErrors + got.Unattributed.ValidationErrors + got.Unattributed.SourceRootErrors
	warnings := got.Totals.ValidationWarnings + got.Unattributed.ValidationWarnings + got.Unattributed.SourceRootWarnings
	if errors != catalog.Result.Validation.ErrorCount || warnings != catalog.Result.Validation.WarningCount {
		t.Fatalf("overview counts %d errors / %d warnings, catalog verdict %d / %d",
			errors, warnings, catalog.Result.Validation.ErrorCount, catalog.Result.Validation.WarningCount)
	}
}

func TestModelOverviewFailuresAreMachineReadable(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	cases := []struct {
		name string
		args []string
	}{
		{"missing workspace", []string{"overview"}},
		{"empty workspace", []string{"overview", "-workspace="}},
		{"limit zero", []string{"overview", "-workspace", root, "-limit", "0"}},
		{"limit over max", []string{"overview", "-workspace", root, "-limit", "1001"}},
		{"positional", []string{"overview", "-workspace", root, "extra"}},
		{"repeated workspace", []string{"overview", "-workspace", root, "-workspace", root}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			code, err := runModelCommand(tc.args, &out)
			if err != nil || code != 2 {
				t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
			}
			var response modelOverviewCLIResponse
			if err := json.Unmarshal(out.Bytes(), &response); err != nil {
				t.Fatalf("not JSON: %v %s", err, out.String())
			}
			if response.OK || response.Error == nil || response.Error.Code != "INVALID_ARGUMENT" ||
				response.SchemaVersion != "tei.cli.model-overview/v0" {
				t.Fatalf("response=%+v", response)
			}
		})
	}
}

// model locate carries the roll-up for the element it locates.
func TestModelLocateCarriesSubtreeAttention(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var out bytes.Buffer
	if code, err := runModelCommand([]string{"locate", "-workspace", root, "-subject-kind", "semantic", "-subject-id", "std"}, &out); err != nil || code != 0 {
		t.Fatalf("locate: code=%d err=%v body=%s", code, err, out.String())
	}
	var response struct {
		Result *operations.ModelLocation `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil || response.Result == nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Result.Attention == nil {
		t.Fatal("locate omits attention for its subtree")
	}
}
