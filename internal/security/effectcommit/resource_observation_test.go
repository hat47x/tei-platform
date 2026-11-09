package effectcommit

import (
	"context"
	"errors"
	"testing"
)

type noReaderProvider struct{}

func (noReaderProvider) Begin(context.Context) (Transaction, error) { return nil, ErrMissingProvider }

type readerProvider struct {
	state ResourceState
	err   error
}

func (readerProvider) Begin(context.Context) (Transaction, error) { return nil, ErrMissingProvider }

func (p readerProvider) ReadResource(context.Context, Scope) (ResourceState, error) {
	return p.state, p.err
}

// TestRunnerReadResourceSeparatesReceiptFromResource checks that a Provider
// without ResourceReader reports the observation as unavailable rather than as
// "not persisted".
func TestRunnerReadResourceSeparatesReceiptFromResource(t *testing.T) {
	_, err := (Runner{Provider: noReaderProvider{}}).ReadResource(context.Background(), Scope{Application: "a", Resource: "r"})
	if !errors.Is(err, ErrResourceObservationUnavailable) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunnerReadResourceUsesTheProviderReader(t *testing.T) {
	got, err := (Runner{Provider: readerProvider{state: ResourceState{Version: 3, Payload: []byte("updated"), Found: true}}}).
		ReadResource(context.Background(), Scope{Application: "a", Resource: "r"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Found || got.Version != 3 || string(got.Payload) != "updated" {
		t.Fatalf("state=%+v", got)
	}
}

func TestRunnerReadResourceWithoutProvider(t *testing.T) {
	_, err := (Runner{}).ReadResource(context.Background(), Scope{})
	if !errors.Is(err, ErrMissingProvider) {
		t.Fatalf("err=%v", err)
	}
}
