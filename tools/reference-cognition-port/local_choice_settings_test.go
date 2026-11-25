package cognitionport_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	choice "tei/plugins/reference-cognition-choice"
)

func TestLocalChoiceCompletionBoundIsDeclaredNotClamped(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings map[string]any
		wantErr  bool
	}{
		{"absent keeps the default", map[string]any{"transport": choice.Transport}, false},
		{"declared cap is sent", map[string]any{"transport": choice.Transport, choice.SettingMaxTokens: float64(1024)}, false},
		{"zero is refused", map[string]any{"transport": choice.Transport, choice.SettingMaxTokens: float64(0)}, true},
		{"fraction is refused", map[string]any{"transport": choice.Transport, choice.SettingMaxTokens: 1.5}, true},
		{"over the transport maximum is refused", map[string]any{"transport": choice.Transport, choice.SettingMaxTokens: float64(choice.MaxMaxTokens + 1)}, true},
		{"wrong type is refused", map[string]any{"transport": choice.Transport, choice.SettingMaxTokens: "1024"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotMax int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					MaxTokens int `json:"max_tokens"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
				}
				gotMax = req.MaxTokens
				fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"unknown"}}]}`)
			}))
			defer server.Close()
			settings := map[string]any{}
			for k, v := range tc.settings {
				settings[k] = v
			}
			settings["endpoint"] = server.URL + "/v1/chat/completions"
			raw := []byte(`{"observations":[{"id":"x","statement":"example"}],"choice":{"observationId":"x","options":[{"id":"a","description":"A"},{"id":"b","description":"B"}]}}`)
			_, err := choice.Invoke(context.Background(), settings["endpoint"].(string), "local-model", settings, raw)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected the completion bound to be refused")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := choice.DefaultMaxTokens
			if v, ok := tc.settings[choice.SettingMaxTokens].(float64); ok {
				want = int(v)
			}
			if gotMax != want {
				t.Fatalf("max_tokens = %d, want %d", gotMax, want)
			}
		})
	}
}
