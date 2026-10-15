package studiohost

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeObserveXeadHostFixture(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestReadOnlyObserveXeadReturnsBoundedProjectionForAllowedWorkspace(t *testing.T) {
	root := t.TempDir()
	writeObserveXeadHostFixture(t, root, "model.xead", `<System DatabasePassword="synthetic-host-secret"><Table ID="T1" Name="Order"><TableField ID="F1" Name="OrderID" DataTypeID="D1"/></Table></System>`)
	host, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}

	got := host.ObserveXead("workspace-1", "model.xead")
	if !got.OK || got.Value == nil || got.Failure != nil {
		t.Fatalf("ObserveXead() = %+v", got)
	}
	if got.Value.Effect != "read-only" || len(got.Value.Observation.Tables) != 1 || got.Value.Observation.Tables[0].Source.ID != "T1" {
		t.Fatalf("unexpected observation: %+v", got.Value)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "synthetic-host-secret") {
		t.Fatal("host response leaked a source attribute value")
	}
}

func TestReadOnlyObserveXeadRejectsUnknownWorkspaceAndUnsafePaths(t *testing.T) {
	root := t.TempDir()
	writeObserveXeadHostFixture(t, root, "model.xead", `<System/>`)
	host, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		id, path, code string
	}{
		{id: "workspace-2", path: "model.xead", code: "WORKSPACE_NOT_ALLOWED"},
		{id: "workspace-1", path: "../synthetic-private.xead", code: "XEAD_SOURCE_READ_FAILED"},
		{id: "workspace-1", path: filepath.Join(root, "model.xead"), code: "XEAD_SOURCE_READ_FAILED"},
	} {
		got := host.ObserveXead(test.id, test.path)
		if got.OK || got.Failure == nil || got.Failure.Code != test.code {
			t.Fatalf("ObserveXead(%q,%q) = %+v; want %s", test.id, test.path, got, test.code)
		}
		if strings.Contains(got.Failure.Message, root) || strings.Contains(got.Failure.Message, "synthetic-") {
			t.Fatalf("failure leaked path or source data: %+v", got.Failure)
		}
	}
}

func TestReadOnlyObserveXeadRejectsDifferentRootIdentity(t *testing.T) {
	root, replacement := rootReplacementPair(t)
	writeObserveXeadHostFixture(t, root, "model.xead", `<System><Table ID="original"/></System>`)
	writeObserveXeadHostFixture(t, replacement, "model.xead", `<System><Table ID="replacement"/></System>`)
	host, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	replaceWorkspaceRoot(t, root, replacement)
	got := host.ObserveXead("workspace-1", "model.xead")
	if got.OK || got.Failure == nil || got.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("different Workspace root identity was not rejected: %+v", got)
	}
	if strings.Contains(got.Failure.Message, root) || strings.Contains(got.Failure.Message, "replacement") {
		t.Fatalf("root-change failure leaked a path or source ID: %+v", got.Failure)
	}
}
