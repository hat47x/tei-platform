package numbering_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/numbering"
	"tei/internal/semantic"
)

func ruleWorld(role bool) ir.World {
	attributes := []semantic.Attribute{
		{ID: "seq.rule.counter", TypeRef: "std.integer", Role: numbering.RoleCounter},
		{ID: "seq.rule.digits", TypeRef: "std.integer", Role: numbering.RoleDigits},
		{ID: "seq.rule.prefix", TypeRef: "std.text", Role: numbering.RolePrefix},
	}
	if role {
		attributes = append(attributes, semantic.Attribute{ID: "seq.rule.rule", TypeRef: "std.identifier", Role: numbering.RoleRule})
	}
	return ir.World{Semantics: []semantic.Namespace{{
		ID:   "seq",
		Data: []semantic.Data{{ID: "seq.rule", Attributes: attributes}},
	}}}
}

func ruleRows(shared bool) []map[core.CanonicalID]string {
	rows := []map[core.CanonicalID]string{
		{"seq.rule.counter": "41", "seq.rule.digits": "5", "seq.rule.prefix": "ORD-"},
		{"seq.rule.counter": "7", "seq.rule.digits": "3", "seq.rule.prefix": "INV-"},
	}
	if shared {
		rows[0]["seq.rule.rule"] = "order"
		rows[1]["seq.rule.rule"] = "invoice"
	}
	return rows
}

// TestBindRuleReadsTheOptionalRole checks that the rule identity role is optional
// and reported.
func TestBindRuleReadsTheOptionalRole(t *testing.T) {
	declaration, err := numbering.Resolve(ruleWorld(true), "seq.rule")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindRule(ruleWorld(true))
	if err != nil {
		t.Fatal(err)
	}
	if binding.Attribute != "seq.rule.rule" {
		t.Fatalf("binding=%+v", binding)
	}
	absent, err := declaration.BindRule(ruleWorld(false))
	if err != nil {
		t.Fatal(err)
	}
	if absent.Attribute != "" {
		t.Fatalf("binding=%+v", absent)
	}
}

// TestRowsForSeparatesRulesInOneData checks that two rules in one Data do not read
// each other's rows. X-TEA kept every rule in one table and read the row by
// IDNUMBER (D:Session.java L1985–1994).
func TestRowsForSeparatesRulesInOneData(t *testing.T) {
	declaration, err := numbering.Resolve(ruleWorld(true), "seq.rule")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindRule(ruleWorld(true))
	if err != nil {
		t.Fatal(err)
	}
	order, err := binding.RowsFor("order", ruleRows(true))
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 1 || order[0]["seq.rule.prefix"] != "ORD-" {
		t.Fatalf("order=%+v", order)
	}
	invoice, err := binding.RowsFor("invoice", ruleRows(true))
	if err != nil {
		t.Fatal(err)
	}
	if len(invoice) != 1 || invoice[0]["seq.rule.prefix"] != "INV-" {
		t.Fatalf("invoice=%+v", invoice)
	}
}

// TestRowsForRefusesAnUnknownOrEmptyRule checks that a missing rule name, a rule
// with no row, and a row without a rule are reported rather than guessed.
func TestRowsForRefusesAnUnknownOrEmptyRule(t *testing.T) {
	declaration, err := numbering.Resolve(ruleWorld(true), "seq.rule")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindRule(ruleWorld(true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := binding.RowsFor("", ruleRows(true)); err == nil {
		t.Fatal("an unnamed rule was accepted")
	}
	if _, err := binding.RowsFor("payment", ruleRows(true)); err == nil {
		t.Fatal("a rule with no row was accepted")
	}
	rows := ruleRows(true)
	rows[1]["seq.rule.rule"] = ""
	if _, err := binding.RowsFor("order", rows); err == nil {
		t.Fatal("a row without a rule was accepted")
	}
}

// TestRowsForKeepsASingleRuleDataWhole checks that a Data without the identity role
// returns every row, so the single-rule case keeps working.
func TestRowsForKeepsASingleRuleDataWhole(t *testing.T) {
	declaration, err := numbering.Resolve(ruleWorld(false), "seq.rule")
	if err != nil {
		t.Fatal(err)
	}
	binding, err := declaration.BindRule(ruleWorld(false))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := binding.RowsFor("order", ruleRows(false))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%+v", rows)
	}
	if _, ok := binding.RuleFor(rows[0]); ok {
		t.Fatal("a single-rule data reported a rule name")
	}
}
