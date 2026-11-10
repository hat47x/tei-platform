package operations

import (
	"testing"

	"tei/internal/sourceindex"
)

// A located Subject is returned with the schema version a reader branches on, so
// the identifier is asserted here rather than only written by the builder
// (#4032).
func TestLocateReportsItsSchemaVersion(t *testing.T) {
	got := locate(t, locationWorld(), sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.customer"}, 50)
	if got.SchemaVersion != "tei.operations.model-location/v0" {
		t.Fatalf("schemaVersion=%q", got.SchemaVersion)
	}
}
