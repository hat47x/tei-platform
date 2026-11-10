package modelref_test

import (
	"testing"

	"tei/internal/modelref"
)

// The impact review carries the schema version a reader branches on, so the
// identifier is asserted here rather than only written by the builder (#4032).
func TestImpactReviewReportsItsSchemaVersion(t *testing.T) {
	before, after := fixture(), fixture()
	review := modelref.ReviewImpactDiff(before, after, rules())
	if review.SchemaVersion != "tei.modelref.impact-review/v0" {
		t.Fatalf("schemaVersion=%q", review.SchemaVersion)
	}
}
