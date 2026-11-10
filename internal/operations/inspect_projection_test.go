package operations

import (
	"encoding/json"
	"testing"

	"tei/internal/sourceindex"
)

func TestValidateInspectionProjection(t *testing.T) {
	requested := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	fixture := func() SubjectInspection {
		return SubjectInspection{
			SchemaVersion:          "tei.operations.inspect/v0",
			Effect:                 "read-only",
			Subject:                requested,
			Canonical:              json.RawMessage(`{"id":"sales.order"}`),
			WorldFingerprint:       "world-fp",
			SourceIndexFingerprint: "source-index-fp",
		}
	}
	if err := ValidateInspectionProjection(requested, fixture()); err != nil {
		t.Fatalf("valid projection rejected: %v", err)
	}
	for _, tc := range []struct {
		name   string
		change func(*SubjectInspection)
	}{
		{"subject", func(v *SubjectInspection) { v.Subject.ID = "other" }},
		{"schema", func(v *SubjectInspection) { v.SchemaVersion = "other" }},
		{"effect", func(v *SubjectInspection) { v.Effect = "write" }},
		{"canonical-absent", func(v *SubjectInspection) { v.Canonical = nil }},
		{"canonical-invalid", func(v *SubjectInspection) { v.Canonical = json.RawMessage(`{"id":`) }},
		{"other-canonical-id", func(v *SubjectInspection) { v.Canonical = json.RawMessage(`{"id":"sales.other"}`) }},
		{"missing-canonical-id", func(v *SubjectInspection) { v.Canonical = json.RawMessage(`{"other":"value"}`) }},
		{"null-canonical", func(v *SubjectInspection) { v.Canonical = json.RawMessage(`null`) }},
		{"array-canonical", func(v *SubjectInspection) { v.Canonical = json.RawMessage(`[{"id":"sales.order"}]`) }},
		{"uppercase-id-field", func(v *SubjectInspection) { v.Canonical = json.RawMessage(`{"ID":"sales.order"}`) }},
		{"case-alias-after-id", func(v *SubjectInspection) { v.Canonical = json.RawMessage(`{"id":"sales.order","ID":"sales.other"}`) }},
		{"duplicate-id", func(v *SubjectInspection) { v.Canonical = json.RawMessage(`{"id":"sales.order","id":"sales.order"}`) }},
		{"duplicate-id-mismatch", func(v *SubjectInspection) { v.Canonical = json.RawMessage(`{"id":"sales.order","id":"sales.other"}`) }},
		{"nested-id-only", func(v *SubjectInspection) { v.Canonical = json.RawMessage(`{"nested":{"id":"sales.order"}}`) }},
		{"world-fingerprint", func(v *SubjectInspection) { v.WorldFingerprint = "" }},
		{"source-index-fingerprint", func(v *SubjectInspection) { v.SourceIndexFingerprint = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := fixture()
			tc.change(&result)
			if err := ValidateInspectionProjection(requested, result); err == nil {
				t.Fatal("invalid Inspector projection accepted")
			}
		})
	}
}
