package studiohost

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

type inspectionServiceFunc func(sourceindex.Subject) (operations.SubjectInspection, error)

func (f inspectionServiceFunc) Inspect(subject sourceindex.Subject) (operations.SubjectInspection, error) {
	return f(subject)
}

func inspectedReference(subject sourceindex.Subject) operations.SubjectInspection {
	return operations.SubjectInspection{
		SchemaVersion:          "tei.operations.inspect/v0",
		Effect:                 "read-only",
		Subject:                subject,
		Canonical:              json.RawMessage(`{"id":"sales.order"}`),
		WorldFingerprint:       "world-v1",
		SourceIndexFingerprint: "source-v1",
	}
}

// An injected service must be the actual Inspector call site, not a factory
// that Studio discovers while it silently invokes the built-in operation.
func TestInspectionUsesHostSelectedService(t *testing.T) {
	root := t.TempDir() // No TEI manifest: the built-in Inspector cannot load it.
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	want := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	factoryCalls, invokeCalls := 0, 0
	h, err := NewReadOnlyWithFactory([]string{root}, func(trustedRoot string) (RootServices, error) {
		factoryCalls++
		if trustedRoot != resolvedRoot {
			t.Fatalf("factory received a different root: %q != %q", trustedRoot, resolvedRoot)
		}
		return RootServices{
			Inspection: inspectionServiceFunc(func(subject sourceindex.Subject) (operations.SubjectInspection, error) {
				invokeCalls++
				if subject != want {
					t.Fatalf("wrong Inspector subject: %+v", subject)
				}
				return inspectedReference(subject), nil
			}),
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got := h.InspectSubject("workspace-1", want.Kind, want.ID)
	if !got.OK || got.Value == nil || got.Value.Subject != want || got.Failure != nil {
		t.Fatalf("host did not return selected Inspector result: %+v", got)
	}
	if factoryCalls != 1 || invokeCalls != 1 {
		t.Fatalf("factory=%d invoked=%d, want exactly once each", factoryCalls, invokeCalls)
	}
}

// A non-nil factory is a Host-owned selection path. If its Inspector is
// absent, invoking the normal built-in Operation would hide the failed bind.
func TestInspectionFactoryMissingServiceFailsClosed(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	var subject sourceindex.Subject
	for _, entry := range catalog.Entries {
		if _, err := operations.InspectSubject(root, entry.Subject); err == nil {
			subject = entry.Subject
			break
		}
	}
	if subject == (sourceindex.Subject{}) {
		t.Fatal("fixture has no inspectable Subject for proving the forbidden fallback")
	}
	calls := 0
	h, err := NewReadOnlyWithFactory([]string{root}, func(string) (RootServices, error) {
		calls++
		return RootServices{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got := h.InspectSubject("workspace-1", subject.Kind, subject.ID)
	if got.OK || got.Value != nil || got.Failure == nil || got.Failure.Code != "OPERATION_FAILED" || calls != 1 {
		t.Fatalf("missing selected service was not rejected: calls=%d response=%+v", calls, got)
	}
}

// A provider that responds with another Subject or with a non-read-only
// projection must not be treated as a successful Inspector response.
func TestInspectionFactoryRejectsInconsistentProjection(t *testing.T) {
	want := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	for _, test := range []struct {
		name   string
		change func(*operations.SubjectInspection)
	}{
		{"wrong-subject", func(v *operations.SubjectInspection) {
			v.Subject.ID = "sales.other"
		}},
		{"wrong-effect", func(v *operations.SubjectInspection) {
			v.Effect = "write"
		}},
		{"missing-source-fingerprint", func(v *operations.SubjectInspection) {
			v.SourceIndexFingerprint = ""
		}},
		{"missing-canonical", func(v *operations.SubjectInspection) {
			v.Canonical = nil
		}},
		{"invalid-canonical-json", func(v *operations.SubjectInspection) {
			v.Canonical = json.RawMessage(`{"id":`)
		}},
		{"wrong-canonical-id", func(v *operations.SubjectInspection) {
			v.Canonical = json.RawMessage(`{"id":"sales.other"}`)
		}},
		{"missing-canonical-id", func(v *operations.SubjectInspection) {
			v.Canonical = json.RawMessage(`{"other":"value"}`)
		}},
		{"unsupported-schema", func(v *operations.SubjectInspection) {
			v.SchemaVersion = "custom"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			h, err := NewReadOnlyWithFactory([]string{root}, func(string) (RootServices, error) {
				return RootServices{Inspection: inspectionServiceFunc(func(sourceindex.Subject) (operations.SubjectInspection, error) {
					v := inspectedReference(want)
					test.change(&v)
					return v, nil
				})}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			got := h.InspectSubject("workspace-1", want.Kind, want.ID)
			if got.OK || got.Value != nil || got.Failure == nil || got.Failure.Code != "OPERATION_FAILED" {
				t.Fatalf("accepted inconsistent read-only projection: %+v", got)
			}
		})
	}
}

func TestInspectionChecksWorkspaceBeforeCallingFactory(t *testing.T) {
	root, replacement := rootReplacementPair(t)
	calls := 0
	h, err := NewReadOnlyWithFactory([]string{root}, func(string) (RootServices, error) {
		calls++
		return RootServices{Inspection: inspectionServiceFunc(func(subject sourceindex.Subject) (operations.SubjectInspection, error) {
			return inspectedReference(subject), nil
		})}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	denied := h.InspectSubject("workspace-2", "semantic-data", "sales.order")
	if denied.OK || denied.Failure == nil || denied.Failure.Code != "WORKSPACE_NOT_ALLOWED" {
		t.Fatalf("out-of-allowlist Inspect succeeded: %+v", denied)
	}
	replaceWorkspaceRoot(t, root, replacement)
	changed := h.InspectSubject("workspace-1", "semantic-data", "sales.order")
	if changed.OK || changed.Failure == nil || changed.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("changed root Inspect succeeded: %+v", changed)
	}
	if calls != 0 {
		t.Fatalf("untrusted workspace reached factory %d times", calls)
	}
}

func TestInspectionDoesNotExposeHostFactoryErrors(t *testing.T) {
	const privateText = "private-provider-path-and-token"
	h, err := NewReadOnlyWithFactory([]string{t.TempDir()}, func(string) (RootServices, error) {
		return RootServices{}, errors.New(privateText)
	})
	if err != nil {
		t.Fatal(err)
	}
	got := h.InspectSubject("workspace-1", "semantic-data", "sales.order")
	if got.OK || got.Failure == nil || got.Failure.Code != "OPERATION_FAILED" ||
		strings.Contains(got.Failure.Message, privateText) {
		t.Fatalf("returned private factory error or success: %+v", got)
	}
}

// TestHostSelectedInspectorTypedErrorsAreSanitized shows why typed errors
// from a selected Provider are less trustworthy than the built-in Inspector:
// the Provider controls both Message and Code. The Studio Host must expose
// neither internal paths/tokens nor an arbitrary error classification.
func TestHostSelectedInspectorTypedErrorsAreSanitized(t *testing.T) {
	const secret = "private-provider-path-and-token"
	tests := []struct {
		name string
		code string
		want string
	}{
		{"not-found", operations.InspectionNotFound, operations.InspectionNotFound},
		{"ambiguous", operations.InspectionAmbiguous, operations.InspectionAmbiguous},
		{"unsupported-kind", operations.InspectionUnsupportedKind, operations.InspectionUnsupportedKind},
		{"invalid-subject", operations.InspectionInvalidSubject, operations.InspectionInvalidSubject},
		{"forged-private-code", "PRIVATE_PATH_" + secret, "OPERATION_FAILED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			h, err := NewReadOnlyWithFactory([]string{root}, func(string) (RootServices, error) {
				return RootServices{Inspection: inspectionServiceFunc(func(sourceindex.Subject) (operations.SubjectInspection, error) {
					return operations.SubjectInspection{}, &operations.InspectionError{
						Code: tt.code, Message: secret,
					}
				})}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			got := h.InspectSubject("workspace-1", "semantic-data", "sales.order")
			if got.OK || got.Value != nil || got.Failure == nil ||
				got.Failure.Code != tt.want ||
				strings.Contains(got.Failure.Message, secret) ||
				strings.Contains(got.Failure.Code, secret) {
				t.Fatalf("selected Inspector leaked typed Provider details: %+v", got)
			}
		})
	}
}
