package failure_test

import (
	"errors"
	"fmt"
	"testing"

	"tei/internal/failure"
)

type sampleFailure struct {
	code    string
	message string
	cause   error
	kind    failure.Kind
	retry   failure.RetryPolicy
	unknown bool
}

func (f *sampleFailure) Error() string                  { return f.code + ": " + f.message }
func (f *sampleFailure) Unwrap() error                  { return f.cause }
func (f *sampleFailure) FailureCode() string            { return f.code }
func (f *sampleFailure) FailureMessage() string         { return f.message }
func (f *sampleFailure) FailureCause() error            { return f.cause }
func (f *sampleFailure) Kind() failure.Kind             { return f.kind }
func (f *sampleFailure) Retryable() failure.RetryPolicy { return f.retry }
func (f *sampleFailure) HasEffectUnknown() bool         { return f.unknown }

func TestAsFailure(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantOK    bool
		wantKind  failure.Kind
		wantRetry failure.RetryPolicy
	}{
		{
			name:      "nil error",
			err:       nil,
			wantOK:    false,
			wantKind:  "",
			wantRetry: "",
		},
		{
			name:      "plain error",
			err:       errors.New("plain"),
			wantOK:    false,
			wantKind:  "",
			wantRetry: "",
		},
		{
			name:      "wrapped failure",
			err:       fmt.Errorf("wrapped: %w", &sampleFailure{code: "X", message: "y", kind: failure.BusinessRejection, retry: failure.NotRetryable, unknown: false}),
			wantOK:    true,
			wantKind:  failure.BusinessRejection,
			wantRetry: failure.NotRetryable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, ok := failure.AsFailure(tt.err)
			if ok != tt.wantOK {
				t.Fatalf("AsFailure() ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if f.Kind() != tt.wantKind {
				t.Fatalf("Kind() = %v, want %v", f.Kind(), tt.wantKind)
			}
			if f.Retryable() != tt.wantRetry {
				t.Fatalf("Retryable() = %v, want %v", f.Retryable(), tt.wantRetry)
			}
		})
	}
}

func TestKindPredicates(t *testing.T) {
	business := &sampleFailure{kind: failure.BusinessRejection}
	technical := &sampleFailure{kind: failure.TechnicalFailure}
	partial := &sampleFailure{kind: failure.PartialSuccess}
	plain := errors.New("plain")

	if !failure.IsBusinessRejection(business) {
		t.Error("IsBusinessRejection(business) = false")
	}
	if failure.IsBusinessRejection(technical) {
		t.Error("IsBusinessRejection(technical) = true")
	}
	if failure.IsBusinessRejection(plain) {
		t.Error("IsBusinessRejection(plain) = true")
	}

	if !failure.IsTechnicalFailure(technical) {
		t.Error("IsTechnicalFailure(technical) = false")
	}
	if failure.IsTechnicalFailure(business) {
		t.Error("IsTechnicalFailure(business) = true")
	}

	if !failure.IsPartialSuccess(partial) {
		t.Error("IsPartialSuccess(partial) = false")
	}
	if failure.IsPartialSuccess(business) {
		t.Error("IsPartialSuccess(business) = true")
	}
}

func TestRetryable(t *testing.T) {
	retryable := &sampleFailure{retry: failure.Retryable}
	not := &sampleFailure{retry: failure.NotRetryable}
	lookup := &sampleFailure{retry: failure.LookupFirst}
	plain := errors.New("plain")

	if !failure.IsRetryable(retryable) {
		t.Error("IsRetryable(retryable) = false")
	}
	if !failure.IsRetryable(lookup) {
		t.Error("IsRetryable(lookup) = false")
	}
	if failure.IsRetryable(not) {
		t.Error("IsRetryable(not) = true")
	}
	if failure.IsRetryable(plain) {
		t.Error("IsRetryable(plain) = true")
	}
}

func TestEffectMayBeUnknown(t *testing.T) {
	unknown := &sampleFailure{unknown: true}
	known := &sampleFailure{unknown: false}
	plain := errors.New("plain")

	if !failure.EffectMayBeUnknown(unknown) {
		t.Error("EffectMayBeUnknown(unknown) = false")
	}
	if failure.EffectMayBeUnknown(known) {
		t.Error("EffectMayBeUnknown(known) = true")
	}
	if failure.EffectMayBeUnknown(plain) {
		t.Error("EffectMayBeUnknown(plain) = true")
	}
}
