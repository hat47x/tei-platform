package studiohost

import (
	"os"
	"path/filepath"
	"testing"
)

type observingFake struct {
	called  bool
	content string
}

func (o *observingFake) Observe(content []byte) (XeadObservation, error) {
	o.called, o.content = true, string(content)
	return XeadObservation{Effect: "read-only"}, nil
}

func TestObserveXeadRunsOnlyAfterAllowlistAndRootIdentityChecks(t *testing.T) {
	root, replacement := rootReplacementPair(t)
	if err := os.WriteFile(filepath.Join(root, "model.xead"), []byte(`<System/>`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(replacement, "model.xead"), []byte(`<System ID="replacement"/>`), 0600); err != nil {
		t.Fatal(err)
	}
	observer := &observingFake{}
	host, err := NewReadOnlyWithXeadObserver([]string{root}, observer)
	if err != nil {
		t.Fatal(err)
	}
	denied := host.ObserveXead("workspace-2", "model.xead")
	if denied.OK || denied.Failure == nil || denied.Failure.Code != "WORKSPACE_NOT_ALLOWED" || observer.called {
		t.Fatalf("allowlist bypass: response=%+v observer=%+v", denied, observer)
	}
	ok := host.ObserveXead("workspace-1", "model.xead")
	if !ok.OK || !observer.called || observer.content != `<System/>` {
		t.Fatalf("observer did not receive source bytes only: response=%+v observer=%+v", ok, observer)
	}

	replaceWorkspaceRoot(t, root, replacement)
	observer.called = false
	changed := host.ObserveXead("workspace-1", "model.xead")
	if changed.OK || changed.Failure == nil || changed.Failure.Code != "WORKSPACE_ROOT_CHANGED" || observer.called {
		t.Fatalf("changed root reached observer: response=%+v observer=%+v", changed, observer)
	}
}

func TestObserveXeadFailureDoesNotExposeObserverError(t *testing.T) {
	root := t.TempDir()
	observer := observerFailure{}
	host, err := NewReadOnlyWithXeadObserver([]string{root}, observer)
	if err != nil {
		t.Fatal(err)
	}
	got := host.ObserveXead("workspace-1", filepath.Join(root, "private.xead"))
	if got.OK || got.Failure == nil || got.Failure.Code != "XEAD_SOURCE_READ_FAILED" || got.Failure.Message != "X-TEA source observation failed" {
		t.Fatalf("unsafe failure: %+v", got)
	}
}

type observerFailure struct{}

func (observerFailure) Observe([]byte) (XeadObservation, error) {
	return XeadObservation{}, &XeadObservationError{Code: "XEAD_SOURCE_READ_FAILED"}
}
