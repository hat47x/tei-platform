package derivationdisclosure

import (
	"context"
	"errors"
	"testing"

	"tei/internal/semantic/derivationref"
)

// TestD5SameRuleDifferentPrincipalChangesDisclosure pins acceptance item 5:
// the same Rule ID discloses for one principal and not for another. The only
// difference between the two runs is the principal and its granted checks.
func TestD5SameRuleDifferentPrincipalChangesDisclosure(t *testing.T) {
	adminHost, adminPlan, adminPolicy, adminAuthority := newFixture(t, Detail, "staff.admin")
	grantAll(adminPolicy, adminHost, adminPlan)
	adminValue, err := Evaluate(context.Background(), adminHost, adminPlan, adminAuthority, itemAmount)
	if err != nil || adminValue.Decimal != "600" {
		t.Fatalf("admin: value=%+v err=%v", adminValue, err)
	}

	publicHost, publicPlan, publicPolicy, publicAuthority := newFixture(t, Detail, "staff.public")
	grantAll(publicPolicy, publicHost, publicPlan)
	var disclose permissionCheck
	for _, check := range requiredPermissions(publicHost, publicPlan) {
		if check.action == "derive.detail.disclose" {
			disclose = check
		}
	}
	if disclose.action == "" {
		t.Fatal("detail disclosure was not required")
	}
	delete(publicPolicy.allowed, disclose)

	called := false
	got, err := Evaluate(context.Background(), publicHost, publicPlan, publicAuthority, func(context.Context) (derivationref.Input, error) {
		called = true
		return decimal("600"), nil
	})
	if !errors.Is(err, ErrDenied) || called || got != (derivationref.Input{}) {
		t.Fatalf("public: value=%+v err=%v called=%v", got, err, called)
	}
	if adminPlan.Definition.ID != publicPlan.Definition.ID {
		t.Fatalf("the two runs must use the same rule id: %q != %q", adminPlan.Definition.ID, publicPlan.Definition.ID)
	}
}

// TestD5SameRuleDifferentTenantChangesDisclosure pins the tenant half of
// acceptance item 5. The rule and principal stay the same; only the tenant and
// the granted resources differ.
func TestD5SameRuleDifferentTenantChangesDisclosure(t *testing.T) {
	t1Host, t1Plan, t1Policy, t1Authority := newFixture(t, Detail, "staff.admin")
	grantAll(t1Policy, t1Host, t1Plan)
	value, err := Evaluate(context.Background(), t1Host, t1Plan, t1Authority, itemAmount)
	if err != nil || value.Decimal != "600" {
		t.Fatalf("tenant t1: value=%+v err=%v", value, err)
	}

	t2Host, t2Plan, t2Policy, t2Authority := newFixture(t, Detail, "staff.admin")
	t2Host.Tenant = "tenant.t2"
	t2Plan.Rows = []AuthorizedRow{{ID: "sales.row_b", Tenant: "tenant.t2"}}
	// No grants for tenant t2.
	got, err := Evaluate(context.Background(), t2Host, t2Plan, t2Authority, itemAmount)
	if !errors.Is(err, ErrDenied) || got != (derivationref.Input{}) {
		t.Fatalf("tenant t2: value=%+v err=%v", got, err)
	}
	if t1Plan.Definition.ID != t2Plan.Definition.ID {
		t.Fatalf("the two runs must use the same rule id: %q != %q", t1Plan.Definition.ID, t2Plan.Definition.ID)
	}
	for check := range t2Policy.allowed {
		if check.subject != "staff.admin" {
			t.Fatalf("unexpected grant: %+v", check)
		}
	}
}
