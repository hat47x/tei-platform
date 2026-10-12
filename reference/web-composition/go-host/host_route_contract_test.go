package webcomposition

import (
	"context"
	"net/http"
	"testing"
)

// These cases bind the protected Go Host's route construction to the E1
// read-only composition check. They are synthetic policy construction tests,
// not evidence that a production reverse proxy preserves a specific URL.
func TestProtectedHostRejectsPathsNotRepresentableInE1(t *testing.T) {
	deps := Dependencies{
		VerifySession: func(*http.Request) (Principal, error) {
			return Principal{ProviderID: "corporate", Issuer: "https://example.invalid/issuer", Subject: "sub-1", PrincipalID: "principal-1"}, nil
		},
		VerifyCSRF: func(*http.Request, Principal) bool { return true },
		Authorize: func(context.Context, Principal, string) (bool, error) { return true, nil },
		Invoke: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		Audit: func(AuditRecord) {},
	}
	newPolicy := func(path, action string) Policy {
		return Policy{
			Path: path, Method: http.MethodPost, ActionID: action,
			Providers: map[string]string{"corporate": "https://example.invalid/issuer"},
		}
	}
	for _, path := range []string{"/scenario/submit", "/scenario/+@;,$"} {
		if _, err := NewHost(newPolicy(path, "order_entry.submit_order"), deps); err != nil {
			t.Fatalf("canonical POST Action route %q was refused: %v", path, err)
		}
	}
	for _, tc := range []struct {
		name string
		path string
	}{
		{"double quote escaped by net/url", "/scenario/a\"b"},
		{"single quote escaped by net/url", "/scenario/a'b"},
		{"exclamation escaped by net/url", "/scenario/a!b"},
		{"asterisk escaped by net/url", "/scenario/a*b"},
		{"brackets escaped by net/url", "/scenario/a[b]"},
		{"braces escaped by net/url", "/scenario/a{b}"},
		{"angle brackets escaped by net/url", "/scenario/a<b>"},
		{"pipe escaped by net/url", "/scenario/a|b"},
		{"caret escaped by net/url", "/scenario/a^b"},
		{"backtick escaped by net/url", "/scenario/a`b"},
		{"parenthesis escaped by net/url", "/scenario/a(b)"},
		{"space in path", "/scenario/sub mit"},
		{"horizontal tab", "/scenario/sub\tmit"},
		{"newline", "/scenario/sub\nmit"},
		{"NUL byte", "/scenario/sub\x00mit"},
		{"ASCII DEL", "/scenario/sub\x7fmit"},
		{"non ASCII path", "/scenario/受注"},
		{"percent encoded alias", "/scenario/%73ubmit"},
		{"query string", "/scenario/submit?role=admin"},
		{"fragment", "/scenario/submit#fragment"},
		{"duplicate slash", "/scenario//submit"},
		{"dot segment", "/scenario/./submit"},
		{"parent segment", "/scenario/../submit"},
		{"backslash", "/scenario/submit\\admin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewHost(newPolicy(tc.path, "order_entry.submit_order"), deps); err == nil {
				t.Fatalf("unrepresentable protected route %q was accepted", tc.path)
			}
		})
	}
	for _, action := range []string{"", " ", "\t", "\n"} {
		if _, err := NewHost(newPolicy("/scenario/submit", action), deps); err == nil {
			t.Errorf("blank Action ID %q bypassed policy construction", action)
		}
	}
}
