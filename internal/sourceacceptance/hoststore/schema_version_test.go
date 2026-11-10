package hoststore

import "testing"

// The store stamps its file with a schema version. The value is a wire contract:
// a different version in the file is refused (store.go compares it on every
// read), so the literal is asserted here as well as used there (#4032).
func TestStoreSchemaVersionIsFixed(t *testing.T) {
	if schemaVersion != "tei.host-acceptance-store/v0" {
		t.Fatalf("schemaVersion=%q", schemaVersion)
	}
}
