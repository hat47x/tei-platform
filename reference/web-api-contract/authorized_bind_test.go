package webapicontract

import (
	"context"
	"errors"
	"net/url"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/workspace"
)

// These tests use the real legacy authorization adapter and the repository's
// existing legacy users/rules. They do not replace runtime/Provider tests.
func fixtureAuthorization(t *testing.T) legacyauthorization.Adapter {
	t.Helper()
	store, err := legacyauth.LoadFixtureStore(filepath.Join("..", "legacy-auth", "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	rule, err := legacyauthorization.LoadFixtureRule(filepath.Join("..", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	return legacyauthorization.Adapter{Store: store, Rule: rule}
}

func fixtureWorld(t *testing.T) PublishedBrowse {
	t.Helper()
	loaded, err := workspace.Load(filepath.Join("..", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	published, err := ResolvePublishedBrowse(loaded.World,
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.recent_orders"),
		map[core.CanonicalID]FieldPolicy{
			core.MustCanonicalID("sales.order.order_id"): {Select: true, Filter: true},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return published
}

func TestRealLegacyAuthorizationBeforeCanonicalBrowseBind(t *testing.T) {
	auth := fixtureAuthorization(t)
	published := fixtureWorld(t)
	for _, subject := range []string{"subject-order-editor", "subject-order-viewer"} {
		t.Run(subject, func(t *testing.T) {
			trusted := TrustedReadContext{
				PrincipalID: subject, RowScopeID: "trusted-fixture-scope", PolicyRevision: "fixture-r1",
			}
			plan, err := AuthorizeBrowseRead(context.Background(), auth, published,
				url.Values{"$select": {"orderId"}, "$top": {"10"}}, trusted)
			if err != nil {
				t.Fatal(err)
			}
			if plan.PrincipalID != subject || plan.ScopeID != trusted.RowScopeID ||
				plan.Limit != 10 || len(plan.Selected) != 1 || plan.Selected[0] != "orderId" {
				t.Fatalf("authorized plan lost identity or projection: %+v", plan)
			}
			_, err = AuthorizeBrowseRead(context.Background(), auth, published,
				url.Values{"$select": {"total"}}, trusted)
			if !errors.Is(err, ErrRejected) {
				t.Fatalf("operation permission must not grant hidden column: %v", err)
			}
		})
	}
	for _, subject := range []string{
		"subject-unknown", "subject-order-disabled", "subject-order-locked",
	} {
		t.Run(subject, func(t *testing.T) {
			_, err := AuthorizeBrowseRead(context.Background(), auth, published,
				nil, TrustedReadContext{
					PrincipalID: subject, RowScopeID: "trusted-fixture-scope", PolicyRevision: "fixture-r1",
				})
			if !errors.Is(err, ErrRejected) {
				t.Fatalf("unauthorized subject accepted: %v", err)
			}
		})
	}
}

func TestRealLegacyAuthorizationBeforeCanonicalCommandBind(t *testing.T) {
	auth := fixtureAuthorization(t)
	loaded, err := workspace.Load(filepath.Join("..", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	published, err := ResolvePublishedCommand(loaded.World,
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.submit_order"), true)
	if err != nil {
		t.Fatal(err)
	}
	// This CommandSpec only tests authorization/binding. A real Capability
	// input must be validated by the runtime; no write occurs in this test.
	spec := CommandSpec{
		ActionID:            string(published.CapabilityID),
		Allowed:             true,
		RequiresRevision:    true,
		RequiresIdempotency: true,
	}
	req := CommandRequest{
		ActionID: spec.ActionID, ExpectedRevision: "v1", IdempotencyKey: "request-1",
	}
	intent, err := AuthorizeCommandIntent(context.Background(), auth, published,
		req, spec, "subject-order-editor")
	if err != nil {
		t.Fatal(err)
	}
	if intent.PrincipalID != "subject-order-editor" || intent.ActionID != spec.ActionID {
		t.Fatalf("command principal or Capability ID lost: %+v", intent)
	}
	for _, subject := range []string{"subject-order-viewer", "subject-order-unknown", "subject-order-disabled"} {
		_, err := AuthorizeCommandIntent(context.Background(), auth, published, req, spec, subject)
		if !errors.Is(err, ErrRejected) {
			t.Fatalf("unauthorized command principal %q accepted: %v", subject, err)
		}
	}
	forged := published
	forged.PatternID = core.MustCanonicalID("order_entry.recent_orders")
	if _, err := AuthorizeCommandIntent(context.Background(), auth, forged, req, spec,
		"subject-order-editor"); !errors.Is(err, ErrRejected) {
		t.Fatalf("read resource reinterpreted as command: %v", err)
	}
	wrongSpec := spec
	wrongSpec.ActionID = "other.capability"
	if _, err := AuthorizeCommandIntent(context.Background(), auth, published, req,
		wrongSpec, "subject-order-editor"); !errors.Is(err, ErrRejected) {
		t.Fatalf("Capability identity mismatch accepted: %v", err)
	}
}
