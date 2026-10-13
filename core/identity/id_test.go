package identity

import (
	"testing"
)

func TestCanonicalIDsPreserveM0IdentityRules(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  CanonicalID
	}{
		{"sales.order", "sales.order"},
		{" sales_order.line_1 ", "sales_order.line_1"},
		{"std.deployment.executor", "std.deployment.executor"},
	} {
		got, err := ParseCanonicalID(tc.input)
		if err != nil || got != tc.want {
			t.Fatalf("ParseCanonicalID(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
		}
	}
	for _, bad := range []string{"", "a..b", "A.B", "a-b", "a/b", "1start", "a."} {
		if _, err := ParseCanonicalID(bad); err == nil {
			t.Fatalf("unapproved M0 canonical identity accepted: %q", bad)
		}
	}
	if got := MustCanonicalID("sales.order"); got != "sales.order" {
		t.Fatalf("MustCanonicalID drifted: %s", got)
	}
}

func TestLocalNamesRefSymbolsAndNumericVersion(t *testing.T) {
	if got, err := ParseLocalName(" line_2 "); err != nil || got != "line_2" {
		t.Fatalf("local name changed: %s, %v", got, err)
	}
	for _, bad := range []string{"line.part", "Line", "", "line-part"} {
		if _, err := ParseLocalName(bad); err == nil {
			t.Fatalf("bad local name accepted: %q", bad)
		}
	}
	if got := MustLocalName("account"); got != "account" {
		t.Fatal("MustLocalName changed")
	}
	reference := Ref{Kind: KindDatabase, ID: MustCanonicalID("sales.order")}
	if reference.String() != "database:sales.order" {
		t.Fatalf("reference string changed: %q", reference.String())
	}
	if got := MustSymbolSegment("OrderStatus"); got != "order_status" {
		t.Fatalf("symbol conversion changed: %q", got)
	}
	if got, err := SymbolToSegment("customer-id"); err != nil || got != "customer_id" {
		t.Fatalf("hyphenated symbol conversion changed: %q %v", got, err)
	}
	for _, bad := range []string{"", "値"} {
		if _, err := SymbolToSegment(bad); err == nil {
			t.Fatalf("bad symbol accepted: %q", bad)
		}
	}
	for _, v := range []string{"1.2.3", " 0.00.9 ", "12.2.30"} {
		if !ValidNumericVersion(v) {
			t.Fatalf("valid M0 version rejected: %q", v)
		}
	}
	for _, v := range []string{"1.2", "v1.2.3", "1.2.3-rc1", "1.x.3", "", "1.2.3.4"} {
		if ValidNumericVersion(v) {
			t.Fatalf("invalid version accepted: %q", v)
		}
	}
}

func TestMustHelpersRejectInvalidInputs(t *testing.T) {
	for _, invoke := range []func(){
		func() { MustCanonicalID("bad-name") },
		func() { MustLocalName("bad.name") },
		func() { MustSymbolSegment("値") },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("Must helper unexpectedly accepted an invalid value")
				}
			}()
			invoke()
		}()
	}
}
