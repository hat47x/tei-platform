package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

func TestCLIInspectAttributeReadAndErrors(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, tc := range []struct {
		name string
		id   string
		exit int
		code string
	}{
		{"present", "sales.order.order_id", 0, ""},
		{"missing", "sales.order.missing", 1, "SUBJECT_NOT_FOUND"},
		{"other-kind", "sales.order", 1, "SUBJECT_NOT_FOUND"},
		{"inexact", " sales.order.order_id", 2, "INVALID_SUBJECT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			exit, err := inspectWorkspaceJSON(&stdout, root, "", "semantic-attribute", tc.id, 0)
			if err != nil || exit != tc.exit {
				t.Fatalf("inspect exit=%d err=%v stdout=%s", exit, err, stdout.String())
			}
			var response cliInspectionReport
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.SchemaVersion != "tei.cli.inspect/v0" {
				t.Fatalf("changed inspection contract: %+v", response)
			}
			if tc.code != "" {
				if response.OK || response.Result != nil || response.Error == nil || response.Error.Code != tc.code {
					t.Fatalf("invalid error response: %+v", response)
				}
				return
			}
			if !response.OK || response.Error != nil || response.Result == nil {
				t.Fatalf("invalid success response: %+v", response)
			}
			selected := sourceindex.Subject{Kind: "semantic-attribute", ID: tc.id}
			result := response.Result
			if result.Subject != selected || result.Effect != "read-only" || result.SourceStatus != "indexed" || len(result.Sources.References) == 0 {
				t.Fatalf("missing canonical/source projection: %+v", result)
			}
			var attribute semantic.Attribute
			if err := json.Unmarshal(result.Canonical, &attribute); err != nil {
				t.Fatal(err)
			}
			if string(attribute.ID) != tc.id || attribute.Symbol != "orderId" || attribute.TypeRef != "sales.order_id" {
				t.Fatalf("wrong canonical attribute: %+v", attribute)
			}
			for _, ref := range result.Sources.References {
				if ref.Subject != selected {
					t.Fatalf("another subject's Source reference was returned: %+v", ref)
				}
			}
		})
	}
}
