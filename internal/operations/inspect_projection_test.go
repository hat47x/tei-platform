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
