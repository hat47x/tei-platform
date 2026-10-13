package webapicontract

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

type authorizationProbe struct {
	calls  int
	input  legacyauthorization.Input
	result capabilityinvoke.BusinessResult
	err    error
}

func (p *authorizationProbe) Invoke(_ context.Context, capability core.CanonicalID, raw any) (capabilityinvoke.BusinessResult, error) {
	p.calls++
	if capability != legacyauthorization.AuthorizationCapability {
		return capabilityinvoke.BusinessResult{}, errors.New("wrong capability")
	}
	p.input, _ = raw.(legacyauthorization.Input)
	return p.result, p.err
}

func allowedProbe() *authorizationProbe {
	return &authorizationProbe{result: capabilityinvoke.BusinessResult{
		Capability: legacyauthorization.AuthorizationCapability,
		Mode: capabilityinvoke.ResultOutcome,
		Outcome: "allowed",
		Payload: legacyauthorization.AuthorizationGrant{
			SubjectID: "staff-1", Action: "browse", Resource: "order_entry.recent_orders",
		},
	}}
}

func TestExistingAuthorizationUsesTrustedSubjectAndExactOperation(t *testing.T) {
	p := allowedProbe()
	err := CheckExistingAuthorization(context.Background(), p,
		"staff-1", "browse", "order_entry.recent_orders")
	if err != nil {
		t.Fatal(err)
	}
	if p.calls != 1 || p.input.SubjectID != "staff-1" ||
		p.input.Action != "browse" || p.input.Resource != "order_entry.recent_orders" {
		t.Fatalf("incorrect security.authorize invocation: %+v", p)
	}
}

func TestExistingAuthorizationFailsClosed(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*authorizationProbe)
		subject  string
		action   string
		resource string
	}{
		{"denied", func(p *authorizationProbe) { p.result.Outcome = "denied" }, "staff-1", "browse", "order_entry.recent_orders"},
		{"wrong capability", func(p *authorizationProbe) { p.result.Capability = core.MustCanonicalID("other") }, "staff-1", "browse", "order_entry.recent_orders"},
		{"wrong result mode", func(p *authorizationProbe) { p.result.Mode = "unexpected" }, "staff-1", "browse", "order_entry.recent_orders"},
		{"wrong subject in grant", func(p *authorizationProbe) {
			p.result.Payload = legacyauthorization.AuthorizationGrant{
				SubjectID: "other", Action: "browse", Resource: "order_entry.recent_orders",
			}
		}, "staff-1", "browse", "order_entry.recent_orders"},
		{"wrong action in grant", func(p *authorizationProbe) {
			p.result.Payload = legacyauthorization.AuthorizationGrant{
				SubjectID: "staff-1", Action: "write", Resource: "order_entry.recent_orders",
			}
		}, "staff-1", "browse", "order_entry.recent_orders"},
		{"wrong resource in grant", func(p *authorizationProbe) {
			p.result.Payload = legacyauthorization.AuthorizationGrant{
				SubjectID: "staff-1", Action: "browse", Resource: "other",
			}
		}, "staff-1", "browse", "order_entry.recent_orders"},
		{"wrong payload type", func(p *authorizationProbe) { p.result.Payload = "allowed" }, "staff-1", "browse", "order_entry.recent_orders"},
		{"backend error", func(p *authorizationProbe) { p.err = errors.New("backend unavailable") }, "staff-1", "browse", "order_entry.recent_orders"},
		{"missing subject", nil, "", "browse", "order_entry.recent_orders"},
		{"missing action", nil, "staff-1", "", "order_entry.recent_orders"},
		{"missing resource", nil, "staff-1", "browse", ""},
		{"cross-user request", nil, "other", "browse", "order_entry.recent_orders"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := allowedProbe()
			if tc.mutate != nil {
				tc.mutate(p)
			}
			if err := CheckExistingAuthorization(context.Background(), p,
				tc.subject, tc.action, tc.resource); !errors.Is(err, ErrRejected) {
				t.Fatalf("unexpected authorization success: %v", err)
			}
		})
	}
	if err := CheckExistingAuthorization(context.Background(), nil,
		"staff-1", "browse", "order_entry.recent_orders"); !errors.Is(err, ErrRejected) {
		t.Fatal("missing authorization provider accepted")
	}
}
