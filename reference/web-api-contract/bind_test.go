package webapicontract

import (
	"errors"
	"net/url"
	"reflect"
	"testing"
)

func readFields() []Field {
	return []Field{
		{ID: "OrderID", Policy: FieldPolicy{Select: true, Filter: true, Sort: true}},
		{ID: "Amount", Policy: FieldPolicy{Select: true, Filter: true, Sort: true}},
		{ID: "Cost", Policy: FieldPolicy{}},
		{ID: "InternalCode", Policy: FieldPolicy{Filter: true}},
	}
}

func trusted() TrustedReadContext {
	return TrustedReadContext{PrincipalID: "staff-1", RowScopeID: "trusted-scope-01", PolicyRevision: "p42"}
}

func TestReadSubsetHappy(t *testing.T) {
	p, err := BindReadSubset(url.Values{
		"$select": {"OrderID,Amount"}, "$filter": {"Amount gt 100"},
		"$orderby": {"OrderID desc"}, "$top": {"20"},
	}, readFields(), trusted())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Selected, []string{"OrderID", "Amount"}) || p.Filter.FieldID != "Amount" || !p.Order.Descending || p.Limit != 20 || p.PrincipalID != "staff-1" || p.ScopeID != "trusted-scope-01" || p.PolicyRevision != "p42" {
		t.Fatalf("unexpected plan: %#v", p)
	}
}

func TestImplicitSelectNeverIncludesHiddenFields(t *testing.T) {
	p, err := BindReadSubset(url.Values{}, readFields(), trusted())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Selected, []string{"OrderID", "Amount"}) {
		t.Fatalf("leaked projection: %v", p.Selected)
	}
}

func TestFailClosedReadCases(t *testing.T) {
	cases := map[string]url.Values{
		"explicit hidden projection":                 {"$select": {"Cost"}},
		"filter on hidden cost":                      {"$filter": {"Cost gt 100"}},
		"sort on hidden cost":                        {"$orderby": {"Cost"}},
		"read versus filter separated":               {"$select": {"InternalCode"}},
		"filter parser does not evaluate expression": {"$filter": {"Amount gt 1 or Cost gt 10"}},
		"reject function calls":                      {"$filter": {"startswith(Cost,'1')"}},
		"reject nested expansion":                    {"$expand": {"Lines"}},
		"reject count side channels":                 {"$count": {"true"}},
		"reject aggregation side channels":           {"$apply": {"aggregate(Cost with sum)"}},
		"reject unknown parameter":                   {"extra": {"a"}},
		"reject duplicate option":                    {"$select": {"OrderID"}, "select": {"Amount"}},
		"reject duplicate value":                     {"$select": {"Amount", "OrderID"}},
		"reject empty select":                        {"$select": {""}},
		"reject nonexistent field":                   {"$filter": {"Guessed eq 1"}},
		"reject oversized top":                       {"$top": {"1000000"}},
		"reject negative top":                        {"$top": {"-1"}},
		"reject invalid order direction":             {"$orderby": {"Amount sideways"}},
		"reject multi key order":                     {"$orderby": {"Amount desc,Cost"}},
		"reject malformed literal":                   {"$filter": {"Amount gt 1; DROP TABLE Orders"}},
	}
	for label, params := range cases {
		t.Run(label, func(t *testing.T) {
			_, err := BindReadSubset(params, readFields(), trusted())
			if !errors.Is(err, ErrRejected) {
				t.Fatalf("expected rejection, got %v", err)
			}
		})
	}
}

func TestNoTrustedContext(t *testing.T) {
	for _, ctx := range []TrustedReadContext{{}, {PrincipalID: "x", PolicyRevision: "p42"}, {PrincipalID: "x", RowScopeID: "s"}} {
		_, err := BindReadSubset(nil, readFields(), ctx)
		if !errors.Is(err, ErrRejected) {
			t.Fatalf("context must be rejected: %+v", ctx)
		}
	}
}

func TestCommandHappyAndNullClearDistinction(t *testing.T) {
	spec := CommandSpec{ActionID: "sales.order.update", Allowed: true, WritableFields: map[string]bool{"Note": true}, RequiresRevision: true, RequiresIdempotency: true}
	req := CommandRequest{ActionID: spec.ActionID, ExpectedRevision: "v2", IdempotencyKey: "k1", Changes: map[string]FieldChange{"Note": {Kind: "set", Value: nil}}}
	intent, err := BindCommand(req, spec, "staff-1")
	if err != nil {
		t.Fatal(err)
	}
	if intent.Changes["Note"].Kind != "set" {
		t.Fatal("lost set/null meaning")
	}
	req.Changes["Note"] = FieldChange{Kind: "clear"}
	intent, err = BindCommand(req, spec, "staff-1")
	if err != nil || intent.Changes["Note"].Kind != "clear" {
		t.Fatalf("lost clear meaning: %+v, %v", intent, err)
	}
}

func TestCommandRejectsBypass(t *testing.T) {
	spec := CommandSpec{ActionID: "sales.order.update", Allowed: true, WritableFields: map[string]bool{"Note": true}, RequiresRevision: true, RequiresIdempotency: true}
	base := CommandRequest{ActionID: spec.ActionID, ExpectedRevision: "v2", IdempotencyKey: "k1", Changes: map[string]FieldChange{"Note": {Kind: "set", Value: "ok"}}}
	cases := []struct {
		name      string
		req       CommandRequest
		spec      CommandSpec
		principal string
	}{
		{"missing principal", base, spec, ""},
		{"wrong action", CommandRequest{ActionID: "sales.order.delete", ExpectedRevision: "v2", IdempotencyKey: "k1"}, spec, "p"},
		{"missing revision", CommandRequest{ActionID: spec.ActionID, IdempotencyKey: "k1"}, spec, "p"},
		{"missing idempotency", CommandRequest{ActionID: spec.ActionID, ExpectedRevision: "v2"}, spec, "p"},
		{"not authorized action", base, CommandSpec{ActionID: spec.ActionID, Allowed: false}, "p"},
		{"attempt to write cost", CommandRequest{ActionID: spec.ActionID, ExpectedRevision: "v2", IdempotencyKey: "k1", Changes: map[string]FieldChange{"Cost": {Kind: "set", Value: 20}}}, spec, "p"},
		{"unknown write operation", CommandRequest{ActionID: spec.ActionID, ExpectedRevision: "v2", IdempotencyKey: "k1", Changes: map[string]FieldChange{"Note": {Kind: "remove", Value: nil}}}, spec, "p"},
		{"clear has value", CommandRequest{ActionID: spec.ActionID, ExpectedRevision: "v2", IdempotencyKey: "k1", Changes: map[string]FieldChange{"Note": {Kind: "clear", Value: "oops"}}}, spec, "p"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BindCommand(tc.req, tc.spec, tc.principal)
			if !errors.Is(err, ErrRejected) {
				t.Fatalf("expected refusal, got %v", err)
			}
		})
	}
}
