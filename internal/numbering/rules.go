package numbering

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
)

// Rules describe one numbering Data with several rules: the identity role, the
// contract roles, and the rows. It joins what a caller otherwise combines by
// hand: pick the rows of one rule, bind the contract and the state, and reserve.
//
// X-TEA read one row by IDNUMBER out of a table that held every rule
// (D:Session.java L1985–1994). The row's values and the rule's identity came from
// the same place, so a wrong IDNUMBER read another rule's counter. Here the rule
// is filtered first, and one Data is one contract per rule.
type Rules struct {
	Declaration Declaration
	Binding     RuleBinding
	Rows        []map[core.CanonicalID]string
}

// ResolveRules reads a numbering Data and its rows, so a caller does not have to
// resolve the declaration and the identity role separately.
func ResolveRules(world ir.World, dataID core.CanonicalID, rows []map[core.CanonicalID]string) (Rules, error) {
	declaration, err := Resolve(world, dataID)
	if err != nil {
		return Rules{}, err
	}
	binding, err := declaration.BindRule(world)
	if err != nil {
		return Rules{}, err
	}
	return Rules{Declaration: declaration, Binding: binding, Rows: rows}, nil
}

// BindRule binds the rows of one rule into a contract and the counter it holds.
//
// The rows of another rule are not consulted, and a rule with no row is reported,
// so a misspelt rule name does not silently issue from another rule.
func (r Rules) BindRule(ruleID string) (Contract, State, error) {
	rows, err := r.Binding.RowsFor(ruleID, r.Rows)
	if err != nil {
		return Contract{}, State{}, err
	}
	if len(rows) != 1 {
		// One rule is one counter: two rows for one rule mean the caller cannot
		// tell which counter to issue from, so the declaration is refused.
		return Contract{}, State{}, fmt.Errorf("%w: rule %s has %d rows, want 1", ErrNumbering, ruleID, len(rows))
	}
	return r.Declaration.Bind(rows[0])
}

// ReserveRule issues the next number of one rule inside the caller's transaction.
// It is BindRule followed by ReserveInTransaction, so a caller does not hold the
// contract and the state between two steps.
func (r Rules) ReserveRule(ctx context.Context, store CounterStore, unitOfWork UnitOfWork, ruleID string) (*IssuedReservation, error) {
	contract, _, err := r.BindRule(ruleID)
	if err != nil {
		return nil, err
	}
	// The store is the counter's home, so the state read from the row is not
	// passed to Reserve: Reserve loads the current counter itself, which is what
	// makes a concurrent change visible.
	return contract.ReserveInTransaction(ctx, store, unitOfWork, ruleID)
}

// ReserveRuleWithoutTransaction issues the next number of one rule without a
// transaction. It exists for a caller that already holds one and cannot share it
// with the numbering package; the caller then owns committing both.
func (r Rules) ReserveRuleWithoutTransaction(ctx context.Context, store CounterStore, ruleID string) (string, *Reservation, error) {
	contract, _, err := r.BindRule(ruleID)
	if err != nil {
		return "", nil, err
	}
	return contract.Reserve(ctx, store, ruleID)
}
