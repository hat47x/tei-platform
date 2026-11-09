package webcomposition

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

// routePairFixture is consumed only by this bounded reference test, not by TEI
// Core, the Go Host, or any application runtime.
type routePairFixture struct {
	E1 struct {
		Route struct {
			ID       string `json:"id"`
			Method   string `json:"method"`
			Path     string `json:"path"`
			ActionID string `json:"actionId"`
		} `json:"route"`
	} `json:"e1"`
	GoHost struct {
		Method    string            `json:"method"`
		Path      string            `json:"path"`
		ActionID  string            `json:"actionId"`
		Providers map[string]string `json:"providers"`
	} `json:"goHost"`
}

// Same names, without actual contract or runtime connection, are not proof
// of Graph identity, terminal coverage, business authorization or effects.
func routePairMatches(f routePairFixture) bool {
	return f.E1.Route.Method == f.GoHost.Method &&
		f.E1.Route.Path == f.GoHost.Path &&
		f.E1.Route.ActionID == f.GoHost.ActionID
}

func TestE1GoHostDeclaredRoutePair(t *testing.T) {
	data, err := os.ReadFile("../e1-go-route-pair.fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	var f routePairFixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	if f.E1.Route.ID != "sales.submit" ||
		f.E1.Route.Method != http.MethodPost ||
		f.E1.Route.Path != "/scenario/submit" ||
		f.E1.Route.ActionID != "order_entry.submit_order" {
		t.Fatalf("reference fixture differs from independent route expected: %+v", f.E1.Route)
	}
	if !routePairMatches(f) {
		t.Fatal("baseline E1 and Host declarations are not identical")
	}

	newHost := func(x routePairFixture) error {
		_, err := NewHost(Policy{
			Method:    x.GoHost.Method,
			Path:      x.GoHost.Path,
			ActionID:  x.GoHost.ActionID,
			Providers: x.GoHost.Providers,
		}, Dependencies{
			VerifySession: func(*http.Request) (Principal, error) {
				return Principal{}, nil
			},
			VerifyCSRF: func(*http.Request, Principal) bool { return false },
			Authorize: func(context.Context, Principal, string) (bool, error) {
				return false, nil
			},
			Invoke: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			Audit:  func(AuditRecord) {},
		})
		return err
	}
	if err := newHost(f); err != nil {
		t.Fatalf("Go Host rejected the paired fixture: %v", err)
	}

	t.Run("HDG-B02-different-valid-Host-route", func(t *testing.T) {
		changed := f
		changed.GoHost.Path = "/scenario/other"
		if routePairMatches(changed) {
			t.Fatal("route mismatch was silently accepted as a pair")
		}
		if err := newHost(changed); err != nil {
			t.Fatalf("different route should be valid to Host alone: %v", err)
		}
	})
	t.Run("HDG-B03-different-valid-Host-Action", func(t *testing.T) {
		changed := f
		changed.GoHost.ActionID = "order_entry.another_action"
		if routePairMatches(changed) {
			t.Fatal("Action mismatch was silently accepted as a pair")
		}
		if err := newHost(changed); err != nil {
			t.Fatalf("different Action should be valid to Host alone: %v", err)
		}
	})
	t.Run("HDG-B04-incompatible-Host-method", func(t *testing.T) {
		changed := f
		changed.GoHost.Method = http.MethodGet
		if routePairMatches(changed) {
			t.Fatal("method mismatch was silently accepted as a pair")
		}
		if err := newHost(changed); err == nil {
			t.Fatal("Host accepted GET Action despite POST-only reference contract")
		}
	})
	t.Run("HDG-B05-route-pair-does-not-qualify-Provider", func(t *testing.T) {
		changed := f
		changed.GoHost.Providers = map[string]string{
			"corporate": "https://different-issuer.example.invalid",
		}
		if !routePairMatches(changed) {
			t.Fatal("Provider change should not alter route identity")
		}
		if err := newHost(changed); err != nil {
			t.Fatalf("Host can compile a different Provider policy locally: %v", err)
		}
		// No comparison to an actual session/Grant or E1 provider resolution
		// was performed; a matched Route alone cannot grant that qualification.
	})
}
