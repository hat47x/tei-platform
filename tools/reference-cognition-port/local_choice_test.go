package cognitionport_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	choice "tei/plugins/reference-cognition-choice"
	port "tei/tools/reference-cognition-port"
)

func TestLocalChoiceComposition(t *testing.T) {
	for _, tc := range []struct {
		answer string
		want   string
	}{
		{"billing", port.StateProposed},
		{"unknown", port.StateUnknown},
		{"invented", port.StateFailed},
		{"billing and contract", port.StateFailed},
	} {
		t.Run(tc.answer, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" || r.Method != "POST" {
					t.Error("unexpected API request")
					w.WriteHeader(400)
					return
				}
				var req struct {
					Model     string `json:"model"`
					MaxTokens int    `json:"max_tokens"`
					Messages  []struct {
						Content string `json:"content"`
					} `json:"messages"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Model != "local-model" || req.MaxTokens != 128 || len(req.Messages) != 2 {
					t.Error("invalid model request")
					w.WriteHeader(400)
					return
				}
				fmt.Fprintf(w, `{"choices":[{"finish_reason":"stop","message":{"content":%q}}]}`, tc.answer)
			}))
			defer server.Close()
			dir := t.TempDir()
			requestPath := filepath.Join(dir, "request.json")
			providerPath := filepath.Join(dir, "provider.json")
			request := `{"schema":"tei.reference.cognition-request/v0","id":"inquiry-1","kind":"choice","profile":"offline","context":{"workspaceRevision":"fixture-v1","observations":[{"id":"inquiry","statement":"先月の請求額が違う"}],"choice":{"observationId":"inquiry","options":[{"id":"billing","description":"請求・支払い"},{"id":"contract","description":"契約の変更"}]}}}`
			provider := port.Provider{Schema: port.ProviderSchema, ID: "local", Model: "local-model", Version: "test", Execution: port.ExecutionExternal, Network: port.NetworkNone, Capabilities: []string{"choice"}, Limits: port.Limits{MaxContextBytes: 8192, MaxOutputBytes: 8192, TimeoutSeconds: 5}, Settings: map[string]any{"endpoint": server.URL + "/v1/chat/completions", "transport": choice.Transport}}
			raw, err := json.Marshal(provider)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(requestPath, []byte(request), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(providerPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			receipt, err := port.ProposeWithInvoker(context.Background(), requestPath, providerPath, "", func(ctx context.Context, p port.Provider, raw []byte) ([]byte, error) {
				return choice.Invoke(ctx, p.Settings["endpoint"].(string), p.Model, raw)
			})
			if err != nil {
				t.Fatal(err)
			}
			if receipt.State != tc.want || receipt.Accepted || receipt.Applied || receipt.GrantsAuthority {
				t.Fatalf("unexpected receipt: %+v", receipt)
			}
			if tc.want == port.StateProposed && (receipt.Proposal == nil || receipt.Proposal.Candidates[0].Payload["optionId"] != "billing") {
				t.Fatal("unexpected choice")
			}
		})
	}
}

func TestChoicePortRejectsForeignOptionEvenFromBoundProvider(t *testing.T) {
	dir := t.TempDir()
	req := filepath.Join(dir, "request.json")
	prov := filepath.Join(dir, "provider.json")
	if err := os.WriteFile(req, []byte(`{"schema":"tei.reference.cognition-request/v0","id":"r","kind":"choice","profile":"offline","context":{"workspaceRevision":"r1","observations":[{"id":"x","statement":"example"}],"choice":{"observationId":"x","options":[{"id":"a","description":"A"},{"id":"b","description":"B"}]}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prov, []byte(`{"schema":"tei.reference.cognition-provider/v0","id":"p","model":"m","version":"v","execution":"external","network":"none","capabilities":["choice"],"limits":{"maxContextBytes":8192,"maxOutputBytes":8192,"timeoutSeconds":5}}`), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := port.ProposeWithInvoker(context.Background(), req, prov, "", func(context.Context, port.Provider, []byte) ([]byte, error) {
		return []byte(`{"schema":"tei.reference.cognition-proposal/v0","state":"proposed","candidates":[{"id":"c","kind":"choice","subject":"x","payload":{"optionId":"foreign"},"basis":["observation:x"]}]}`), nil
	})
	if err != nil || result.State != port.StateFailed {
		t.Fatalf("expected rejection: %+v %v", result, err)
	}
}
