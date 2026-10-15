package derivationdisclosure

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/semantic/derivationref"
)

type fixtureStore struct {
	users map[string]legacyauth.User
}

func (s fixtureStore) FindBySubject(_ context.Context, id string) (legacyauth.User, bool, error) {
	user, ok := s.users[id]
	return user, ok, nil
}

// This is a test-only policy table plugged into the real #1118/#1043
// security.authorize Adapter. It is NOT a second product authority.
type fixtureRule struct {
	allowed map[permissionCheck]bool
	seen    []permissionCheck
	broken  bool
}

func (r *fixtureRule) Evaluate(_ context.Context, u legacyauth.User, action, resource string) (bool, string, error) {
	if r.broken {
		return false, "", errors.New("secret cost=300; internal policy crashed")
	}
	key := permissionCheck{subject: u.SubjectID, action: action, resource: resource}
	r.seen = append(r.seen, key)
	return r.allowed[key], "cost=300 for private tenant", nil
}

func newFixture(t *testing.T, mode Mode, who string, surfaces ...Surface) (TrustedContext, QueryPlan, *fixtureRule, legacyauthorization.Adapter) {
	t.Helper()
	host := TrustedContext{
		SubjectID: who, Tenant: "tenant.t1", Purpose: "sales.order",
		PolicyPartition: "policy-epoch-A",
	}
	if mode == Aggregate {
		host.ProviderSubjectID = "trusted.aggregate.service"
	}
	plan := QueryPlan{
		Definition: derivationref.Definition{
			ID: "sales.line_amount", Pure: true, Operator: "multiply",
			Inputs: []string{"sales.order_line.quantity", "sales.order_line.cost"},
		},
		Target: "sales.order_line.amount", Mode: mode,
		Rows: []AuthorizedRow{{ID: "sales.row_a", Tenant: "tenant.t1"}},
		Surfaces: surfaces,
	}
	if len(surfaces) == 0 {
		plan.Surfaces = []Surface{Select}
		if mode == Aggregate {
			plan.Surfaces = []Surface{Group}
		}
	}
	rule := &fixtureRule{allowed: map[permissionCheck]bool{}}
	adapter := legacyauthorization.Adapter{
		Store: fixtureStore{users: map[string]legacyauth.User{
			"staff.admin": {SubjectID: "staff.admin", OrganizationCode: "sales", RoleCode: "admin"},
			"staff.public": {SubjectID: "staff.public", OrganizationCode: "sales", RoleCode: "public"},
			"trusted.aggregate.service": {SubjectID: "trusted.aggregate.service", OrganizationCode: "system", RoleCode: "provider"},
		}},
		Rule: rule,
	}
	return host, plan, rule, adapter
}

func grantAll(rule *fixtureRule, host TrustedContext, plan QueryPlan) {
	for _, req := range requiredPermissions(host, plan) {
		rule.allowed[req] = true
	}
}

func decimal(v string) derivationref.Input {
	return derivationref.Input{State: derivationref.Present, Decimal: v}
}

func itemAmount(ctx context.Context) (derivationref.Input, error) {
	return derivationref.Multiply(decimal("2"), decimal("300"))
}

func TestD5AuthorizedDirectReadWithRealSecurityCapability(t *testing.T) {
	host, plan, policy, authority := newFixture(t, Detail, "staff.admin")
	grantAll(policy, host, plan)
	value, err := Evaluate(context.Background(), host, plan, authority, itemAmount)
	if err != nil || value.State != derivationref.Present || value.Decimal != "600" {
		t.Fatalf("admin expected amount 600, got %+v, %v", value, err)
	}
	if len(policy.seen) != 2*len(requiredPermissions(host, plan)) {
		t.Fatalf("authorization checks missing before/after evaluation: %d", len(policy.seen))
	}
}

func TestD5CannotDiscloseDerivedValueByHidingPrivateInput(t *testing.T) {
	host, plan, policy, authority := newFixture(t, Detail, "staff.public", Select)
	grantAll(policy, host, plan)
	private := permissionCheck{subject: host.SubjectID, action: "derive.input.read",
		resource: "tenant/tenant.t1/purpose/sales.order/input/sales.order_line.cost"}
	delete(policy.allowed, private)
	evaluated := false
	got, err := Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
		evaluated = true
		return decimal("600"), nil
	})
	if !errors.Is(err, ErrDenied) || evaluated || got != (derivationref.Input{}) {
		t.Fatalf("public disclosed private amount: value=%+v err=%v evaluated=%v", got, err, evaluated)
	}
	if strings.Contains(err.Error(), "cost") || strings.Contains(err.Error(), "300") {
		t.Fatalf("denial revealed hidden data: %v", err)
	}
}

func TestD5EveryDisclosureSurfaceNeedsIndependentAuthority(t *testing.T) {
	for _, surface := range []Surface{Select, Filter, Sort, Count, Group, Page, Metadata, Trace} {
		t.Run(string(surface), func(t *testing.T) {
			mode := Detail
			if surface == Group {
				mode = Aggregate
			}
			host, plan, policy, authority := newFixture(t, mode, "staff.public", surface)
			grantAll(policy, host, plan)
			var checked permissionCheck
			for _, p := range requiredPermissions(host, plan) {
				if p.action == "derive.surface."+string(surface) {
					checked = p
				}
			}
			if checked.action == "" {
				t.Fatal("surface was not included in required grants")
			}
			delete(policy.allowed, checked)
			called := false
			got, err := Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
				called = true
				return decimal("600"), nil
			})
			if !errors.Is(err, ErrDenied) || called || got != (derivationref.Input{}) {
				t.Fatalf("ungranted %s disclosed data: %+v, %v", surface, got, err)
			}
		})
	}
}

func TestD5PublicAggregateUsesTrustedProviderPrivateInputOnly(t *testing.T) {
	host, plan, policy, authority := newFixture(t, Aggregate, "staff.public", Group)
	plan.Rows = append(plan.Rows, AuthorizedRow{ID: "sales.row_b", Tenant: "tenant.t1"})
	grantAll(policy, host, plan)
	// Public caller has no raw cost read grant, but the selected provider
	// has its own explicit internal-read grant in security.authorize.
	for check := range policy.allowed {
		if check.subject == host.SubjectID && check.action == "derive.input.read" {
			t.Fatal("aggregate accidentally granted raw input read to caller")
		}
	}
	result, err := Evaluate(context.Background(), host, plan, authority, func(ctx context.Context) (derivationref.Input, error) {
		total := new(big.Rat)
		for _, in := range [][2]string{{"2", "300"}, {"3", "100"}} {
			item, err := derivationref.Multiply(decimal(in[0]), decimal(in[1]))
			if err != nil {
				return derivationref.Input{}, err
			}
			n, ok := new(big.Rat).SetString(item.Decimal)
			if !ok {
				return derivationref.Input{}, fmt.Errorf("unexpected decimal")
			}
			total.Add(total, n)
		}
		return decimal(total.FloatString(0)), nil
	})
	if err != nil || result.Decimal != "900" {
		t.Fatalf("authorized department-only aggregate: %+v %v", result, err)
	}
	if len(policy.seen) != 2*len(requiredPermissions(host, plan)) {
		t.Fatalf("aggregate skipped a distinct row/provider/disclosure permission: %+v", policy.seen)
	}
	// In the same scope, a detail request remains denied.
	detailHost, detailPlan, _, _ := newFixture(t, Detail, "staff.public", Select)
	detailPlan.Rows = plan.Rows
	if v, err := Evaluate(context.Background(), detailHost, detailPlan, authority, itemAmount); !errors.Is(err, ErrDenied) || v != (derivationref.Input{}) {
		t.Fatalf("aggregate permission widened to detail: %+v %v", v, err)
	}
}

// Input access is NOT authority to execute a provider realization. Both
// checks go through the existing security.authorize adapter, not a D5 rule.
func TestD5AggregateProviderExecuteGrantIsIndependent(t *testing.T) {
	host, plan, policy, authority := newFixture(t, Aggregate, "staff.public", Group)
	grantAll(policy, host, plan)
	execute := permissionCheck{
		subject:  host.ProviderSubjectID,
		action:   "derive.provider.execute",
		resource: "tenant/tenant.t1/purpose/sales.order/rule/sales.line_amount",
	}
	if !policy.allowed[execute] {
		t.Fatal("aggregate plan did not require provider execution authorization")
	}
	delete(policy.allowed, execute)
	called := false
	result, err := Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
		called = true
		return decimal("600"), nil
	})
	if !errors.Is(err, ErrDenied) || called || result != (derivationref.Input{}) {
		t.Fatalf("internal read + caller aggregate permission bypassed provider execution denial: %+v %v", result, err)
	}
	if len(policy.seen) == 0 || policy.seen[len(policy.seen)-1] != execute {
		t.Fatalf("denial must come from existing Authority on the selected provider identity: %+v", policy.seen)
	}
	// The same caller may receive an authorized aggregate without acquiring
	// any private input-read grant for itself.
	policy.seen = nil
	policy.allowed[execute] = true
	result, err = Evaluate(context.Background(), host, plan, authority, itemAmount)
	if err != nil || result.State != derivationref.Present || result.Decimal != "600" {
		t.Fatalf("existing Authority approved provider execution but result was blocked: %+v %v", result, err)
	}
	for _, check := range policy.seen {
		if check.subject == host.SubjectID && check.action == "derive.input.read" {
			t.Fatal("aggregate provider authorization leaked raw input permission to the caller")
		}
	}
	if len(policy.seen) != 2*len(requiredPermissions(host, plan)) {
		t.Fatalf("expected separate pre/post execution authorization checks: %+v", policy.seen)
	}
}

func TestD5UntrustedAggregateExecutorCannotReadPrivateInputs(t *testing.T) {
	host, plan, policy, authority := newFixture(t, Aggregate, "staff.public", Group)
	grantAll(policy, host, plan)
	for p := range policy.allowed {
		if p.subject == host.ProviderSubjectID && p.action == "derive.input.internal" {
			delete(policy.allowed, p)
		}
	}
	called := false
	result, err := Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
		called = true
		return decimal("600"), nil
	})
	if !errors.Is(err, ErrDenied) || result != (derivationref.Input{}) || called {
		t.Fatalf("unauthorized provider read succeeded: %+v %v", result, err)
	}
}

func TestD5TenantAndRowRestrictionsApplyBeforeGroupFilterAndCount(t *testing.T) {
	for _, surface := range []Surface{Group, Filter, Count, Page} {
		t.Run(string(surface), func(t *testing.T) {
			mode := Detail
			if surface == Group {
				mode = Aggregate
			}
			host, plan, policy, authority := newFixture(t, mode, "staff.public", surface)
			plan.Rows = append(plan.Rows, AuthorizedRow{ID: "sales.hidden_row", Tenant: "tenant.t2"})
			grantAll(policy, host, plan)
			visited := false
			result, err := Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
				visited = true
				return decimal("600"), nil
			})
			if !errors.Is(err, ErrInvalidPlan) || visited || result != (derivationref.Input{}) ||
				len(policy.seen) != 0 {
				t.Fatalf("%s exposed another tenant via row membership: %+v %v", surface, result, err)
			}
		})
	}
	host, plan, policy, authority := newFixture(t, Aggregate, "staff.public", Group, Count)
	grantAll(policy, host, plan)
	delete(policy.allowed, permissionCheck{subject: host.SubjectID, action: "derive.row.include",
		resource: "tenant/tenant.t1/purpose/sales.order/row/sales.row_a"})
	value, err := Evaluate(context.Background(), host, plan, authority, itemAmount)
	if !errors.Is(err, ErrDenied) || value != (derivationref.Input{}) {
		t.Fatalf("missing row authorization returned group/count: %+v %v", value, err)
	}
}

func TestD5RevocationAfterEvaluationAndCachedOutputRecheck(t *testing.T) {
	host, plan, policy, authority := newFixture(t, Detail, "staff.admin", Select)
	grantAll(policy, host, plan)
	target := permissionCheck{subject: host.SubjectID, action: "derive.surface.select",
		resource: "tenant/tenant.t1/purpose/sales.order/target/sales.order_line.amount"}
	got, err := Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
		delete(policy.allowed, target)
		return decimal("600"), nil
	})
	if !errors.Is(err, ErrDenied) || got != (derivationref.Input{}) {
		t.Fatalf("revoked grant returned late result: %+v %v", got, err)
	}
	grantAll(policy, host, plan)
	cached := decimal("600")
	got, err = Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
		return cached, nil
	})
	if err != nil || got.Decimal != "600" {
		t.Fatalf("valid current cache read denied: %+v %v", got, err)
	}
	delete(policy.allowed, target)
	got, err = Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
		t.Fatal("revoked cached value must not be fetched")
		return cached, nil
	})
	if !errors.Is(err, ErrDenied) || got != (derivationref.Input{}) {
		t.Fatalf("revoked cache exposed old value: %+v %v", got, err)
	}
}

func TestD5FailClosedOnAuthorityFailureAndEvaluatorSecrets(t *testing.T) {
	host, plan, policy, authority := newFixture(t, Detail, "staff.admin")
	grantAll(policy, host, plan)
	policy.broken = true
	evaluated := false
	got, err := Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
		evaluated = true
		return decimal("600"), nil
	})
	if !errors.Is(err, ErrAuthorityFailure) || evaluated || got != (derivationref.Input{}) ||
		strings.Contains(err.Error(), "cost=300") {
		t.Fatalf("authority error leaked backend detail: %+v %v", got, err)
	}
	policy.broken = false
	got, err = Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
		return derivationref.Input{}, fmt.Errorf("hidden cost=300, tenant=tenant.t1")
	})
	if !errors.Is(err, ErrEvaluationFailure) || got != (derivationref.Input{}) ||
		strings.Contains(err.Error(), "cost") || strings.Contains(err.Error(), "tenant") {
		t.Fatalf("provider error leaked secret: %+v %v", got, err)
	}
	for _, state := range []derivationref.InputState{derivationref.Missing, derivationref.AbsentRelation, derivationref.MultipleRelations} {
		got, err = Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
			return derivationref.Input{State: state}, nil
		})
		if !errors.Is(err, ErrEvaluationFailure) || got != (derivationref.Input{}) {
			t.Fatalf("%s converted to disclosed value: %+v %v", state, got, err)
		}
	}
}

type forgedAuthority struct{}

func (forgedAuthority) Invoke(_ context.Context, _ core.CanonicalID, raw any) (capabilityinvoke.BusinessResult, error) {
	r := raw.(legacyauthorization.Input)
	return capabilityinvoke.BusinessResult{
		Capability: legacyauthorization.AuthorizationCapability,
		Mode: capabilityinvoke.ResultOutcome, Outcome: "allowed",
		Payload: legacyauthorization.AuthorizationGrant{
			SubjectID: "other.principal", Action: r.Action, Resource: r.Resource,
		},
	}, nil
}

func TestD5ForgedOrMismatchedGrantIsNotAnApproval(t *testing.T) {
	host, plan, _, _ := newFixture(t, Detail, "staff.admin")
	got, err := Evaluate(context.Background(), host, plan, forgedAuthority{}, itemAmount)
	if !errors.Is(err, ErrAuthorityFailure) || got != (derivationref.Input{}) {
		t.Fatalf("inexact grant accepted: %+v %v", got, err)
	}
}

func TestD5InvalidPlansFailBeforeAuthorityAndEvaluation(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*TrustedContext, *QueryPlan)
	}{
		{"missing-purpose", func(h *TrustedContext, _ *QueryPlan) { h.Purpose = "" }},
		{"missing-partition", func(h *TrustedContext, _ *QueryPlan) { h.PolicyPartition = "" }},
		{"missing-target", func(_ *TrustedContext, p *QueryPlan) { p.Target = "" }},
		{"wrong-canonical-rule", func(_ *TrustedContext, p *QueryPlan) { p.Definition.ID = "bad-rule" }},
		{"wrong-canonical-input", func(_ *TrustedContext, p *QueryPlan) { p.Definition.Inputs[0] = "invalid-input" }},
		{"impure", func(_ *TrustedContext, p *QueryPlan) { p.Definition.Pure = false }},
		{"unknown-surface", func(_ *TrustedContext, p *QueryPlan) { p.Surfaces = []Surface{"secret-scan"} }},
		{"no-surfaces", func(_ *TrustedContext, p *QueryPlan) { p.Surfaces = []Surface{} }},
		{"empty-rows", func(_ *TrustedContext, p *QueryPlan) { p.Rows = nil }},
		{"duplicate-rows", func(_ *TrustedContext, p *QueryPlan) { p.Rows = append(p.Rows, p.Rows[0]) }},
		{"aggregate-no-provider", func(h *TrustedContext, p *QueryPlan) { p.Mode = Aggregate; p.Surfaces = []Surface{Group} }},
		{"aggregate-disguised-select", func(h *TrustedContext, p *QueryPlan) {
			p.Mode = Aggregate; h.ProviderSubjectID = "trusted.aggregate.service"
		}},
		{"detail-internal-provider", func(h *TrustedContext, _ *QueryPlan) { h.ProviderSubjectID = "trusted.aggregate.service" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host, plan, policy, authority := newFixture(t, Detail, "staff.admin")
			grantAll(policy, host, plan)
			tc.edit(&host, &plan)
			called := false
			got, err := Evaluate(context.Background(), host, plan, authority, func(context.Context) (derivationref.Input, error) {
				called = true
				return decimal("600"), nil
			})
			if !errors.Is(err, ErrInvalidPlan) || called || got != (derivationref.Input{}) || len(policy.seen) != 0 {
				t.Fatalf("%s was accepted: %+v %v", tc.name, got, err)
			}
		})
	}
}
