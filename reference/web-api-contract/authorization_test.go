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
		Mode:       capabilityinvoke.ResultOutcome,
		Outcome:    "allowed",
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
	deny := func(p *authorizationProbe) { p.result.Outcome = "denied" }
	wrongCapability := func(p *authorizationProbe) { p.result.Capability = core.MustCanonicalID("other") }
	wrongMode := func(p *authorizationProbe) { p.result.Mode = "unexpected" }
	wrongSubject := func(p *authorizationProbe) {
		p.result.Payload = legacyauthorization.AuthorizationGrant{
			SubjectID: "other", Action: "browse", Resource: "order_entry.recent_orders",
		}
	}
	wrongAction := func(p *authorizationProbe) {
		p.result.Payload = legacyauthorization.AuthorizationGrant{
			SubjectID: "staff-1", Action: "execute", Resource: "order_entry.recent_orders",
		}
	}
	wrongResource := func(p *authorizationProbe) {
		p.result.Payload = legacyauthorization.AuthorizationGrant{
			SubjectID: "staff-1", Action: "browse", Resource: "other",
		}
	}
	badPayload := func(p *authorizationProbe) { p.result.Payload = "allowed" }
	backendError := func(p *authorizationProbe) { p.err = errors.New("backend unavailable") }
	cases := []struct {
		name   string
		modify func(*authorizationProbe)
		want   error
	}{
		{"denied", deny, ErrRejected},
		{"wrong capability", wrongCapability, ErrAuthorizationContract},
		{"wrong result mode", wrongMode, ErrAuthorizationContract},
		{"wrong subject in grant", wrongSubject, ErrAuthorizationContract},
		{"wrong action in grant", wrongAction, ErrAuthorizationContract},
		{"wrong resource in grant", wrongResource, ErrAuthorizationContract},
		{"wrong payload type", badPayload, ErrAuthorizationContract},
		{"backend error", backendError, ErrAuthorizationUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := allowedProbe()
			tc.modify(p)
			if err := CheckExistingAuthorization(context.Background(), p,
				"staff-1", "browse", "order_entry.recent_orders"); !errors.Is(err, tc.want) {
				t.Fatalf("authorization outcome not classified: got=%v want=%v", err, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		subject  string
		action   string
		resource string
	}{
		{"", "browse", "order_entry.recent_orders"},
		{"staff-1", "", "order_entry.recent_orders"},
		{"staff-1", "browse", ""},
		{"other", "browse", "order_entry.recent_orders"},
	} {
		p := allowedProbe()
		want := ErrRejected
		if tc.subject == "other" {
			want = ErrAuthorizationContract
		}
		if err := CheckExistingAuthorization(context.Background(), p,
			tc.subject, tc.action, tc.resource); !errors.Is(err, want) {
			t.Fatalf("invalid identity/resource not rejected: %+v err=%v", tc, err)
		}
	}
	if err := CheckExistingAuthorization(context.Background(), nil,
		"staff-1", "browse", "order_entry.recent_orders"); !errors.Is(err, ErrAuthorizationUnavailable) {
		t.Fatal("missing authorization provider was not classified as unavailable")
	}
	if err := CheckExistingAuthorization(nil, allowedProbe(),
		"staff-1", "browse", "order_entry.recent_orders"); !errors.Is(err, ErrAuthorizationUnavailable) {
		t.Fatal("missing context was not classified as unavailable")
	}
}
