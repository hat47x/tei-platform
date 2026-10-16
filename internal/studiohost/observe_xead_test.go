package studiohost

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	const source = `<System Secret="synthetic-source-secret"/>`
	if err := os.WriteFile(filepath.Join(root, "model.xead"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name        string
		err         error
		code        string
		failureText string
	}{
		{name: "typed parser error", err: &XeadObservationError{Code: XeadParseFailed}, code: XeadParseFailed, failureText: "X-TEA source observation failed"},
		{name: "typed projection error", err: &XeadObservationError{Code: XeadProjectionFailed}, code: XeadProjectionFailed, failureText: "X-TEA source observation failed"},
		{name: "unknown typed error", err: &XeadObservationError{Code: "private-path-secret"}, code: "OPERATION_FAILED", failureText: "workspace operation failed"},
		{name: "untyped error", err: errors.New("private-path-secret"), code: "OPERATION_FAILED", failureText: "workspace operation failed"},
		{name: "typed nil error", err: typedNilXeadObservationError(), code: "OPERATION_FAILED", failureText: "workspace operation failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			observer := &observerFailure{err: tc.err}
			host, err := NewReadOnlyWithXeadObserver([]string{root}, observer)
			if err != nil {
				t.Fatal(err)
			}
			got := host.ObserveXead("workspace-1", "model.xead")
			if !observer.called || observer.content != source {
				t.Fatalf("observer was not called with allowed source bytes: %+v", observer)
			}
			if got.OK || got.Failure == nil || got.Failure.Code != tc.code || got.Failure.Message != tc.failureText {
				t.Fatalf("failure was not mapped safely: %+v", got)
			}
			if strings.Contains(got.Failure.Message, "private-path-secret") || strings.Contains(got.Failure.Message, "synthetic-source-secret") {
				t.Fatalf("failure leaked private content: %+v", got.Failure)
			}
		})
	}

	host, err := NewReadOnlyWithXeadObserver([]string{root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := host.ObserveXead("workspace-1", "model.xead")
	if got.OK || got.Failure == nil || got.Failure.Code != "OPERATION_FAILED" || got.Failure.Message != "workspace operation failed" {
		t.Fatalf("nil observer failure was not stable: %+v", got)
	}
}

func typedNilXeadObservationError() error {
	var err *XeadObservationError
	return err
}

type observerFailure struct {
	err     error
	called  bool
	content string
}

func (o *observerFailure) Observe(content []byte) (XeadObservation, error) {
	o.called, o.content = true, string(content)
	return XeadObservation{}, o.err
}
