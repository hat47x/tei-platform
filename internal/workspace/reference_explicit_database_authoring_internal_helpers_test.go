package workspace

import (
	"strings"
	"testing"
)

// requireExplicitDatabaseAuthoring is the internal-package counterpart of the helper
// in reference_explicit_database_authoring_helpers_test.go.
func requireExplicitDatabaseAuthoring(t *testing.T, result Result, sourceKey, subject string) {
	t.Helper()
	location, ok := result.Sources[sourceKey]
	if !ok || !strings.HasPrefix(location.File, "databases/") {
		t.Fatalf("%sが明示的なDatabase authoring sourceなしに存在します: key=%s location=%#v", subject, sourceKey, location)
	}
}
