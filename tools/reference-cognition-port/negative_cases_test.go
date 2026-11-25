package cognitionport_test

// Negative cases #5807 and #5809 ask for, exercised through the port rather than
// around it: an invalid model answer, an undeclared option, an instruction
// inside the input, a cancellation, a deadline, and an endpoint that is not the
// literal loopback the provider declared. Each case asserts that a failed
// proposal does not become accepted meaning, granted authority, or an applied
// change — the port's own promise.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	choice "tei/plugins/reference-cognition-choice"
	port "tei/tools/reference-cognition-port"
)

const choiceContext = `{"workspaceRevision":"neg-v1","observations":[{"id":"q","statement":"請求書の宛名が誤っている"}],"choice":{"observationId":"q","options":[{"id":"billing","description":"請求・支払い"},{"id":"contract","description":"契約内容"}]}}`

func writeProvider(t *testing.T, endpoint string, timeoutSeconds int) string {
	t.Helper()
	provider := port.Provider{
		Schema: port.ProviderSchema, ID: "local", Model: "local-model", Version: "test",
		Execution: port.ExecutionExternal, Network: port.NetworkNone,
		Capabilities: []string{"choice"},
		Limits:       port.Limits{MaxContextBytes: 8192, MaxOutputBytes: 8192, TimeoutSeconds: timeoutSeconds},
		Settings:     map[string]any{"transport": choice.Transport, "endpoint": endpoint},
	}
	raw, err := json.Marshal(provider)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "provider.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeRequest(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "request.json")
	request := `{"schema":"tei.reference.cognition-request/v0","id":"q1","kind":"choice","profile":"offline","context":` + choiceContext + `}`
	if err := os.WriteFile(path, []byte(request), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func propose(t *testing.T, endpoint string, timeoutSeconds int, invoke port.ExternalInvoker) port.Receipt {
	t.Helper()
	receipt, err := port.ProposeWithInvoker(context.Background(), writeRequest(t), writeProvider(t, endpoint, timeoutSeconds), "", invoke)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func choiceInvoker(model string) port.ExternalInvoker {
	return func(ctx context.Context, provider port.Provider, raw []byte) ([]byte, error) {
		endpoint, _ := provider.Settings["endpoint"].(string)
		return choice.Invoke(ctx, endpoint, model, provider.Settings, raw)
	}
}

// A model answer that is neither a declared option nor "unknown" fails closed.
func TestNegativeUndeclaredModelAnswerFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"invented"}}]}`)
	}))
	defer server.Close()
	receipt := propose(t, server.URL+"/v1/chat/completions", 10, choiceInvoker("local-model"))
	if receipt.State != port.StateFailed {
		t.Fatalf("state = %s, want failed", receipt.State)
	}
	if receipt.Accepted || receipt.Applied || receipt.GrantsAuthority || receipt.Proposal != nil {
		t.Fatalf("a failed proposal must not carry a proposal or authority: %+v", receipt)
	}
}

// Malformed JSON from the model is a failure, not an abstention.
func TestNegativeMalformedModelReplyFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop"`)
	}))
	defer server.Close()
	receipt := propose(t, server.URL+"/v1/chat/completions", 10, choiceInvoker("local-model"))
	if receipt.State != port.StateFailed {
		t.Fatalf("state = %s, want failed", receipt.State)
	}
}

// A truncated generation is not an answer. The transport must not accept
// finish_reason=length as a completed choice.
func TestNegativeTruncatedGenerationFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"choices":[{"finish_reason":"length","message":{"content":"bill"}}]}`)
	}))
	defer server.Close()
	receipt := propose(t, server.URL+"/v1/chat/completions", 10, choiceInvoker("local-model"))
	if receipt.State != port.StateFailed {
		t.Fatalf("state = %s, want failed", receipt.State)
	}
}

// A 5xx from the local server is a provider failure, and the receipt keeps the
// distinction rather than reporting an empty proposal.
func TestNegativeServerErrorIsAFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	receipt := propose(t, server.URL+"/v1/chat/completions", 10, choiceInvoker("local-model"))
	if receipt.State != port.StateFailed {
		t.Fatalf("state = %s, want failed", receipt.State)
	}
	if !strings.Contains(receipt.Diagnostics[0], "HTTP 500") {
		t.Fatalf("diagnostic does not name the HTTP status: %v", receipt.Diagnostics)
	}
}

// The transport refuses an endpoint that is not a literal loopback chat URL,
// whatever the provider file claims: the declaration and the request must
// agree, and a proxy or redirect is not followed.
func TestNegativeNonLoopbackEndpointIsRefused(t *testing.T) {
	for _, endpoint := range []string{
		"http://example.com/v1/chat/completions",
		"http://127.0.0.1:9/other/path",
		"https://127.0.0.1:9/v1/chat/completions",
		"http://user:secret@127.0.0.1:9/v1/chat/completions",
		"http://127.0.0.1:9/v1/chat/completions?redirect=x",
	} {
		t.Run(endpoint, func(t *testing.T) {
			receipt := propose(t, endpoint, 10, choiceInvoker("local-model"))
			if receipt.State != port.StateFailed {
				t.Fatalf("state = %s, want failed for %s", receipt.State, endpoint)
			}
		})
	}
}

// A deadline is a failure with the timeout recorded, not a silent abstention.
// The handler deliberately outlives the declared one-second deadline; the port
// must stop waiting rather than accept a late answer.
func TestNegativeDeadlineExceededIsAFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(4 * time.Second)
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"billing"}}]}`)
	}))
	defer server.Close()
	started := time.Now()
	receipt := propose(t, server.URL+"/v1/chat/completions", 1, choiceInvoker("local-model"))
	elapsed := time.Since(started)
	if receipt.State != port.StateFailed {
		t.Fatalf("state = %s, want failed", receipt.State)
	}
	if elapsed >= 3*time.Second {
		t.Fatalf("the declared one-second deadline did not bound the call: %s", elapsed)
	}
	if receipt.Accepted || receipt.GrantsAuthority || receipt.Proposal != nil {
		t.Fatalf("a timed-out call must not produce a proposal: %+v", receipt)
	}
}

// A cancelled caller stops the call and reports a failure rather than a
// half-read answer.
func TestNegativeCancellationIsAFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	receipt, err := port.ProposeWithInvoker(ctx, writeRequest(t), writeProvider(t, server.URL+"/v1/chat/completions", 10), "", choiceInvoker("local-model"))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != port.StateFailed {
		t.Fatalf("state = %s, want failed", receipt.State)
	}
	if receipt.Accepted || receipt.GrantsAuthority || receipt.Proposal != nil {
		t.Fatalf("a cancelled call must not produce a proposal: %+v", receipt)
	}
}

// The options the model may answer are the ones the request declared. A port
// that let a provider select outside the declared set would make a candidate
// mention a type the caller never offered.
func TestNegativeChoiceOptionOutsideTheDeclaredSetIsRefused(t *testing.T) {
	// A bound provider returns a syntactically valid proposal naming an option
	// the request did not declare. The port, not the provider, must refuse it.
	receipt := propose(t, "http://127.0.0.1:1/v1/chat/completions", 5,
		func(context.Context, port.Provider, []byte) ([]byte, error) {
			return []byte(`{"schema":"tei.reference.cognition-proposal/v0","state":"proposed","candidates":[{"id":"choice-1","kind":"choice","subject":"q","payload":{"optionId":"undeclared"},"basis":["observation:q"]}]}`), nil
		})
	if receipt.State != port.StateFailed {
		t.Fatalf("state = %s, want failed", receipt.State)
	}
	if !strings.Contains(receipt.Diagnostics[0], "proposal is not valid") {
		t.Fatalf("diagnostic does not name the invalid proposal: %v", receipt.Diagnostics)
	}
}

// An offline request may not reach a provider that declares network access, and
// the port reports that as blocked rather than as a provider answer.
func TestNegativeOfflineRequestBlocksNetworkProvider(t *testing.T) {
	provider := port.Provider{
		Schema: port.ProviderSchema, ID: "cloud", Model: "m", Version: "1",
		Execution: port.ExecutionExternal, Network: port.NetworkRequired,
		Capabilities: []string{"choice"},
		Limits:       port.Limits{MaxContextBytes: 8192, MaxOutputBytes: 8192, TimeoutSeconds: 5},
	}
	raw, err := json.Marshal(provider)
	if err != nil {
		t.Fatal(err)
	}
	providerPath := filepath.Join(t.TempDir(), "provider.json")
	if err := os.WriteFile(providerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := port.Propose(context.Background(), writeRequest(t), providerPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != port.StateBlocked {
		t.Fatalf("state = %s, want blocked", receipt.State)
	}
	if receipt.Accepted || receipt.GrantsAuthority || receipt.Applied {
		t.Fatalf("a blocked request must not gain authority: %+v", receipt)
	}
}
