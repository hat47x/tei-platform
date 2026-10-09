package actionboundary

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
)

type fixedSubject struct {
	id  string
	err error
}

func (s fixedSubject) Subject(context.Context) (string, error) { return s.id, s.err }

type authorizationAdapter struct {
	outcome core.LocalName
	grant   legacyauthorization.AuthorizationGrant
	err     error
}

func (a authorizationAdapter) Invoke(_ context.Context, c core.CanonicalID, raw any) (capabilityinvoke.BusinessResult, error) {
	if a.err != nil {
		return capabilityinvoke.BusinessResult{}, a.err
	}
	in := raw.(legacyauthorization.Input)
	grant := a.grant
	if grant.SubjectID == "" {
		grant = legacyauthorization.AuthorizationGrant{SubjectID: in.SubjectID, Action: in.Action, Resource: in.Resource}
	}
	outcome := a.outcome
	if outcome == "" {
		outcome = "allowed"
	}
	return capabilityinvoke.BusinessResult{Capability: c, Mode: capabilityinvoke.ResultOutcome, Outcome: outcome, Payload: grant}, nil
}
func TestCapabilityAuthorizer(t *testing.T) {
	cap := semantic.Capability{ID: legacyauthorization.AuthorizationCapability, Outcomes: []semantic.Outcome{{Name: "allowed", Payload: "auth.grant"}, {Name: "denied", Payload: "auth.denial"}}}
	mk := func(a capabilityinvoke.Adapter) CapabilityAuthorizer {
		return CapabilityAuthorizer{Subjects: fixedSubject{id: "trusted-subject"}, Capability: cap, Adapter: a}
	}
	cases := []struct {
		name       string
		authorizer CapabilityAuthorizer
		denied     bool
	}{
		{"matching grant", mk(authorizationAdapter{}), false},
		{"denied outcome", mk(authorizationAdapter{outcome: "denied"}), true},
		{"unknown outcome", mk(authorizationAdapter{outcome: "unknown"}), true},
		{"adapter failure", mk(authorizationAdapter{err: errors.New("unavailable")}), true},
		{"wrong subject in grant", mk(authorizationAdapter{grant: legacyauthorization.AuthorizationGrant{SubjectID: "other", Action: "execute", Resource: "sui/doc/move"}}), true},
		{"missing subject", CapabilityAuthorizer{Subjects: fixedSubject{}, Capability: cap, Adapter: authorizationAdapter{}}, true},
		{"subject lookup failure", CapabilityAuthorizer{Subjects: fixedSubject{err: errors.New("no session")}, Capability: cap, Adapter: authorizationAdapter{}}, true},
		{"missing adapter", CapabilityAuthorizer{Subjects: fixedSubject{id: "trusted-subject"}, Capability: cap}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.authorizer.Authorize(context.Background(), "sui", "doc", "move")
			if tc.denied && !errors.Is(err, ErrDenied) {
				t.Fatalf("expected denied: %v", err)
			}
			if !tc.denied && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuthorizationResourceDoesNotConflateIdentitySegments(t *testing.T) {
 first:=authorizationResource("a/b","c","move")
 second:=authorizationResource("a","b/c","move")
 if first==second || first!="a%2Fb/c/move" || second!="a/b%2Fc/move" {
  t.Fatalf("resource collision or unexpected encoding: %q / %q",first,second)
 }
 if authorizationResource("sui","doc","sui.move")!="sui/doc/sui.move" {
  t.Fatal("simple resource identifiers must remain compatible")
 }
 cap:=semantic.Capability{ID:legacyauthorization.AuthorizationCapability,Outcomes:[]semantic.Outcome{{Name:"allowed",Payload:"auth.grant"},{Name:"denied",Payload:"auth.denial"}}}
 a:=CapabilityAuthorizer{
  Subjects:fixedSubject{id:"trusted-subject"},Capability:cap,
  Adapter:authorizationAdapter{grant:legacyauthorization.AuthorizationGrant{
   SubjectID:"trusted-subject",Action:"execute",Resource:first,
  }},
 }
 if err:=a.Authorize(context.Background(),"a","b/c","move");!errors.Is(err,ErrDenied) {
  t.Fatalf("grant for another identity must be denied: %v",err)
 }
 if err:=a.Authorize(context.Background(),"a/b","c","move");err!=nil {
  t.Fatalf("matching encoded resource must be allowed: %v",err)
 }
}
