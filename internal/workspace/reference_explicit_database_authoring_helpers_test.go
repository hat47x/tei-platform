package workspace_test

import (
	"strings"
	"testing"

	"tei/internal/workspace"
)

// requireExplicitDatabaseAuthoring checks that a physical artefact in the World
// (Persistence Mapping, Database Contract, SQL realization) has its own explicit
// authoring source under databases/. Semantic / Pattern authoring alone must not
// create it, but the Reference Workspace now authors these artefacts explicitly.
func requireExplicitDatabaseAuthoring(t *testing.T, result workspace.Result, sourceKey, subject string) {
	t.Helper()
	location, ok := result.Sources[sourceKey]
	if !ok || !strings.HasPrefix(location.File, "databases/") {
		t.Fatalf("%sが明示的なDatabase authoring sourceなしに存在します: key=%s location=%#v", subject, sourceKey, location)
	}
}
