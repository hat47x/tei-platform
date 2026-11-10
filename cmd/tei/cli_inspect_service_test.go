package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

type cliInspectionServiceFunc func(sourceindex.Subject) (operations.SubjectInspection, error)

func (fn cliInspectionServiceFunc) Inspect(subject sourceindex.Subject) (operations.SubjectInspection, error) {
	return fn(subject)
}

func selectedInspectionFixture(subject sourceindex.Subject) operations.SubjectInspection {
	return operations.SubjectInspection{
		SchemaVersion:          "tei.operations.inspect/v0",
		Effect:                 "read-only",
		Subject:                subject,
		Canonical:              json.RawMessage(`{"id":"sales.order"}`),
		WorldFingerprint:       "world-v1",
		SourceIndexFingerprint: "source-v1",
	}
}

// A temporary directory lacking a TEI manifest cannot be inspected by the
// built-in implementation. A success here is therefore proof the supplied
// typed call site, rather than the built-in fallback, was invoked.
func TestCLIInspectUsesInjectedHostService(t *testing.T) {
	root := t.TempDir()
	want := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	calls := 0
	service := cliInspectionServiceFunc(func(got sourceindex.Subject) (operations.SubjectInspection, error) {
		calls++
		if got != want {
			t.Fatalf("wrong subject: %+v", got)
		}
		return selectedInspectionFixture(want), nil
	})
	var out bytes.Buffer
	code, err := inspectWorkspaceJSONWithService(&out, root, "", want.Kind, want.ID, 0, service)
	if err != nil || code != 0 {
		t.Fatalf("injected service failed: code=%d err=%v out=%s", code, err, out.String())
	}
	if calls != 1 {
		t.Fatalf("injected provider invoked %d times, want 1", calls)
	}
	var envelope cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || envelope.Result == nil || envelope.Error != nil || envelope.Result.Subject != want {
		t.Fatalf("incorrect injected service result: %+v", envelope)
	}
}

// Explicit service selection and its failure must never result in an
// untracked call to the built-in implementation.
func TestCLIInspectMissingServiceFailsClosed(t *testing.T) {
	var out bytes.Buffer
	code, err := inspectWorkspaceJSONWithService(&out, t.TempDir(), "", "semantic-data", "sales.order", 0, nil)
	if err != nil || code != 2 {
		t.Fatalf("missing provider code=%d err=%v", code, err)
	}
	var envelope cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.OK || envelope.Result != nil || envelope.Error == nil || envelope.Error.Code != "INSPECTION_PROVIDER_MISSING" {
		t.Fatalf("missing selected provider was not rejected: %+v", envelope)
	}
}

func TestCLIInspectValidatesBeforeInvokingService(t *testing.T) {
	calls := 0
	service := cliInspectionServiceFunc(func(subject sourceindex.Subject) (operations.SubjectInspection, error) {
		calls++
		return selectedInspectionFixture(subject), nil
	})
	var out bytes.Buffer
	code, err := inspectWorkspaceJSONWithService(&out, t.TempDir(), "-world-conflicts", "semantic-data", "sales.order", 0, service)
	if err != nil || code != 2 || calls != 0 {
		t.Fatalf("invalid CLI input reached service: code=%d err=%v calls=%d", code, err, calls)
	}
}

func TestCLIInspectRejectsInconsistentServiceResult(t *testing.T) {
	want := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	for _, test := range []struct {
		name   string
		change func(*operations.SubjectInspection)
	}{
		{"wrong-subject", func(v *operations.SubjectInspection) { v.Subject.ID = "sales.other" }},
		{"write-effect", func(v *operations.SubjectInspection) { v.Effect = "write" }},
		{"wrong-schema", func(v *operations.SubjectInspection) { v.SchemaVersion = "custom" }},
		{"no-world-fingerprint", func(v *operations.SubjectInspection) { v.WorldFingerprint = "" }},
		{"no-source-fingerprint", func(v *operations.SubjectInspection) { v.SourceIndexFingerprint = "" }},
		{"no-canonical", func(v *operations.SubjectInspection) { v.Canonical = nil }},
		{"malformed-canonical", func(v *operations.SubjectInspection) { v.Canonical = json.RawMessage(`{"id":`) }},
		{"wrong-canonical-id", func(v *operations.SubjectInspection) { v.Canonical = json.RawMessage(`{"id":"sales.other"}`) }},
		{"no-canonical-id", func(v *operations.SubjectInspection) { v.Canonical = json.RawMessage(`{"other":"value"}`) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service := cliInspectionServiceFunc(func(sourceindex.Subject) (operations.SubjectInspection, error) {
				calls++
				v := selectedInspectionFixture(want)
				test.change(&v)
				return v, nil
			})
			var out bytes.Buffer
			code, err := inspectWorkspaceJSONWithService(&out, t.TempDir(), "", want.Kind, want.ID, 0, service)
			if err != nil || code != 2 || calls != 1 {
				t.Fatalf("invalid projection accepted: code=%d err=%v calls=%d", code, err, calls)
			}
			var envelope cliInspectionReport
			if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.OK || envelope.Result != nil || envelope.Error == nil || envelope.Error.Code != "INSPECTION_PROVIDER_INVALID" {
				t.Fatalf("invalid selected result accepted: %+v", envelope)
			}
		})
	}
}

func TestCLIInspectDoesNotExposeUnknownProviderErrors(t *testing.T) {
	const privateData = "private-provider-path-and-token"
	service := cliInspectionServiceFunc(func(sourceindex.Subject) (operations.SubjectInspection, error) {
		return operations.SubjectInspection{}, errors.New(privateData)
	})
	var out bytes.Buffer
	code, err := inspectWorkspaceJSONWithService(&out, t.TempDir(), "", "semantic-data", "sales.order", 0, service)
	if err != nil || code != 2 || strings.Contains(out.String(), privateData) {
		t.Fatalf("private provider error leaked: code=%d err=%v body=%s", code, err, out.String())
	}
	var envelope cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error == nil || envelope.Error.Code != "INSPECTION_FAILED" || envelope.OK {
		t.Fatalf("provider error was not returned: %+v", envelope)
	}
}

func TestCLIInspectTypedProviderNotFoundPreservesExitAndErrorCode(t *testing.T) {
	service := cliInspectionServiceFunc(func(sourceindex.Subject) (operations.SubjectInspection, error) {
		return operations.SubjectInspection{}, &operations.InspectionError{Code: operations.InspectionNotFound, Message: "not found"}
	})
	var out bytes.Buffer
	code, err := inspectWorkspaceJSONWithService(&out, t.TempDir(), "", "semantic-data", "sales.order", 0, service)
	if err != nil || code != 1 {
		t.Fatalf("not found exit=%d err=%v", code, err)
	}
	var envelope cliInspectionReport
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != operations.InspectionNotFound {
		t.Fatalf("typed error misclassified: %+v", envelope)
	}
}

func TestCLIInspectMasksForgedProviderFailureCodeAndMessage(t *testing.T) {
	const secret = "private-provider-token"
	for _, tc := range []struct {
		name     string
		code     string
		wantCode string
		wantExit int
	}{
		{name: "known-error-code", code: operations.InspectionNotFound, wantCode: operations.InspectionNotFound, wantExit: 1},
		{name: "forged-code", code: "LEAK_" + secret, wantCode: "INSPECTION_FAILED", wantExit: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := cliInspectionServiceFunc(func(sourceindex.Subject) (operations.SubjectInspection, error) {
				return operations.SubjectInspection{}, &operations.InspectionError{Code: tc.code, Message: secret}
			})
			var out bytes.Buffer
			exit, err := inspectWorkspaceJSONWithService(&out, t.TempDir(), "", "semantic-data", "sales.order", 0, service)
			if err != nil || exit != tc.wantExit || strings.Contains(out.String(), secret) {
				t.Fatalf("forged classification leaked: exit=%d err=%v out=%s", exit, err, out.String())
			}
			var response cliInspectionReport
			if err := json.Unmarshal(out.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.OK || response.Error == nil || response.Error.Code != tc.wantCode {
				t.Fatalf("unexpected error classification: %+v", response)
			}
		})
	}
}
