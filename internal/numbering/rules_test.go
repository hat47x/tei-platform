package numbering_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/numbering"
)

func sharedRuleRows() []map[core.CanonicalID]string {
	return []map[core.CanonicalID]string{
		{"seq.rule.rule": "order", "seq.rule.counter": "41", "seq.rule.digits": "5", "seq.rule.prefix": "ORD-"},
		{"seq.rule.rule": "invoice", "seq.rule.counter": "7", "seq.rule.digits": "3", "seq.rule.prefix": "INV-"},
	}
}

// TestResolveRulesBindsTheNamedRule checks that one Data with two rules binds the
// requested rule only, so a wrong rule name cannot issue another rule's number.
func TestResolveRulesBindsTheNamedRule(t *testing.T) {
	rules, err := numbering.ResolveRules(ruleWorld(true), "seq.rule", sharedRuleRows())
	if err != nil {
		t.Fatal(err)
	}
	contract, state, err := rules.BindRule("order")
	if err != nil {
		t.Fatal(err)
	}
	if contract.Digits != 5 || contract.Prefix != "ORD-" || state.Current != 41 {
		t.Fatalf("contract=%+v state=%+v", contract, state)
	}
	invoice, invoiceState, err := rules.BindRule("invoice")
	if err != nil {
		t.Fatal(err)
	}
	if invoice.Digits != 3 || invoice.Prefix != "INV-" || invoiceState.Current != 7 {
		t.Fatalf("contract=%+v state=%+v", invoice, invoiceState)
	}
}

// TestBindRuleRefusesAMissingRule checks that a rule with no row is reported.
func TestBindRuleRefusesAMissingRule(t *testing.T) {
	rules, err := numbering.ResolveRules(ruleWorld(true), "seq.rule", sharedRuleRows())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := rules.BindRule("payment"); err == nil {
		t.Fatal("a rule with no row was accepted")
	}
}

// TestBindRuleRefusesTwoRowsForOneRule checks that one rule cannot hold two
// counters, because then the caller cannot tell which counter to issue from.
func TestBindRuleRefusesTwoRowsForOneRule(t *testing.T) {
	rows := sharedRuleRows()
	rows = append(rows, map[core.CanonicalID]string{"seq.rule.rule": "order", "seq.rule.counter": "99", "seq.rule.digits": "5"})
	rules, err := numbering.ResolveRules(ruleWorld(true), "seq.rule", rows)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := rules.BindRule("order"); err == nil {
		t.Fatal("two rows for one rule were accepted")
	}
}

// TestReserveRuleIssuesFromTheNamedRule checks the joined path: resolve, bind, and
// reserve through one call, so the caller does not hold the contract and the state
// between steps.
func TestReserveRuleIssuesFromTheNamedRule(t *testing.T) {
	rules, err := numbering.ResolveRules(ruleWorld(true), "seq.rule", sharedRuleRows())
	if err != nil {
		t.Fatal(err)
	}
	store := numbering.NewMemoryCounterStore()
	ctx := context.Background()
	if err := store.Save(ctx, "order", 0, numbering.State{Current: 41}); err != nil {
		t.Fatal(err)
	}
	reservation, err := rules.ReserveRule(ctx, store, &memoryUnitOfWork{transaction: &memoryTransaction{}}, "order")
	if err != nil {
		t.Fatal(err)
	}
	if reservation.Number() != "ORD-00042" {
		t.Fatalf("number=%q", reservation.Number())
	}
	if err := reservation.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	state, _, err := store.Load(ctx, "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 42 {
		t.Fatalf("state=%+v", state)
	}
}

// TestReserveRuleWithoutTransactionRollsBackOnRelease checks the non-transactional
// path: a caller that holds its own transaction can still return the counter.
func TestReserveRuleWithoutTransactionRollsBackOnRelease(t *testing.T) {
	rules, err := numbering.ResolveRules(ruleWorld(true), "seq.rule", sharedRuleRows())
	if err != nil {
		t.Fatal(err)
	}
	store := numbering.NewMemoryCounterStore()
	ctx := context.Background()
	if err := store.Save(ctx, "invoice", 0, numbering.State{Current: 7}); err != nil {
		t.Fatal(err)
	}
	number, reservation, err := rules.ReserveRuleWithoutTransaction(ctx, store, "invoice")
	if err != nil {
		t.Fatal(err)
	}
	if number != "INV-008" {
		t.Fatalf("number=%q", number)
	}
	if err := reservation.Release(ctx); err != nil {
		t.Fatal(err)
	}
	state, _, err := store.Load(ctx, "invoice")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 7 {
		t.Fatalf("state=%+v", state)
	}
}

// TestReserveRuleReportsAConcurrentChange checks that the joined path still
// reports a counter that moved between the load and the write.
func TestReserveRuleReportsAConcurrentChange(t *testing.T) {
	rules, err := numbering.ResolveRules(ruleWorld(true), "seq.rule", sharedRuleRows())
	if err != nil {
		t.Fatal(err)
	}
	store := numbering.NewMemoryCounterStore()
	ctx := context.Background()
	if err := store.Save(ctx, "order", 0, numbering.State{Current: 41}); err != nil {
		t.Fatal(err)
	}
	unit := &memoryUnitOfWork{transaction: &memoryTransaction{}}
	first, err := rules.ReserveRule(ctx, store, unit, "order")
	if err != nil {
		t.Fatal(err)
	}
	second, err := rules.ReserveRule(ctx, store, &memoryUnitOfWork{transaction: &memoryTransaction{}}, "order")
	if err != nil {
		t.Fatal(err)
	}
	// The first reservation cannot return its counter: another caller already
	// issued from it. The conflict is reported rather than overwriting the second
	// issuance, which is the guarantee X-TEA lacked (D:Session.java L1995–2002).
	rollbackErr := first.Rollback(ctx)
	if !errors.Is(rollbackErr, numbering.ErrNumberingConflict) {
		t.Fatalf("superseded rollback err=%v", rollbackErr)
	}
	if second.Number() != "ORD-00043" {
		t.Fatalf("second=%q", second.Number())
	}
	state, _, err := store.Load(ctx, "order")
	if err != nil {
		t.Fatal(err)
	}
	if state.Current != 43 {
		t.Fatalf("state=%+v", state)
	}
}
