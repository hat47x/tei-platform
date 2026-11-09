//go:build unix

package ordermigrationdualdb

import (
	"errors"
	"testing"
	"time"
)

// P-MIG-03: the contract phase is decided from observed writes, not from a
// client's self-declaration. An old-format write seen in the declared window
// blocks; only a window with new-version writes and no old-version write
// allows. A window with no observation at all is not evidence and is refused.
func TestPostgresPMIG03ContractUsesObservedWrites(t *testing.T) {
	h := newHarness(t, ampleStock())
	plan, planPath := h.migratedPlan(30)
	h.applyChanges(plan)
	if _, err := ExpandNameSplit(h.ctx, h.primary, plan); err != nil {
		t.Fatal(err)
	}
	if err := SwitchFieldOwner(h.ctx, h.primary, plan, FieldName, "primary", "test:observe-writes"); err != nil {
		t.Fatal(err)
	}

	var beforeOld time.Time
	if err := h.primary.QueryRow(h.ctx, `SELECT clock_timestamp()`).Scan(&beforeOld); err != nil {
		t.Fatal(err)
	}
	// An old-format writer writes full_name only.
	h.exec(h.primary, `UPDATE omdd.customer SET full_name='観測 太郎' WHERE customer_code='C006'`)
	if n := h.count(h.primary, `SELECT count(*) FROM omdd.schema_write WHERE change_id=$1 AND version=$2 AND entity_key='C006' AND observed_at >= $3`,
		NameSplitChange, SchemaVersionV1, beforeOld); n != 1 {
		t.Fatalf("old-version write observations after the boundary = %d, want 1", n)
	}

	refused, err := CheckContractNameSplit(h.ctx, h.primary, plan, ContractOptions{Since: beforeOld, Now: beforeOld.Add(time.Hour)})
	if !errors.Is(err, ErrContractRefused) || refused.Allowed {
		t.Fatalf("contract with an observed old write: %+v %v", refused, err)
	}
	if refused.ObservedOld == 0 || refused.LastOldAt.IsZero() {
		t.Fatalf("old write was not counted: %+v", refused)
	}
	if !contains(refused.Blockers, "observed-old-version-write:C006") {
		t.Fatalf("blockers do not name the observed old write: %v", refused.Blockers)
	}

	// Retire every declared old-version writer and start a new window.
	h.exec(h.primary, `UPDATE omdd.schema_client SET active=false WHERE schema_version=$1`, SchemaVersionV1)
	var afterOld time.Time
	if err := h.primary.QueryRow(h.ctx, `SELECT clock_timestamp()`).Scan(&afterOld); err != nil {
		t.Fatal(err)
	}
	str := func(s string) *string { return &s }
	if err := UpdateCustomer(h.ctx, h.primary, "C006", CustomerChange{FamilyName: str("観測"), GivenName: str("次郎")}); err != nil {
		t.Fatal(err)
	}

	allowed, err := CheckContractNameSplit(h.ctx, h.primary, plan, ContractOptions{Since: afterOld, Now: afterOld.Add(time.Hour)})
	if err != nil || !allowed.Allowed {
		t.Fatalf("contract with only a new write observed: %+v %v", allowed, err)
	}
	if allowed.ObservedOld != 0 || allowed.ObservedNew == 0 {
		t.Fatalf("observation counts are wrong: %+v", allowed)
	}
	// The same window through the CLI: the flag is parsed and the decision
	// matches the function.
	if r := h.child("", "contract-name", "-plan", planPath, "-since", afterOld.UTC().Format(time.RFC3339Nano)); r.code != 0 {
		t.Fatalf("contract-name CLI with an observed window: exit %d %s", r.code, r.stderr)
	}

	// The declarations are all retired, but a window with no observation is
	// not evidence that the old version is gone.
	var later time.Time
	if err := h.primary.QueryRow(h.ctx, `SELECT clock_timestamp()`).Scan(&later); err != nil {
		t.Fatal(err)
	}
	unobserved, err := CheckContractNameSplit(h.ctx, h.primary, plan, ContractOptions{Since: later, Now: later.Add(time.Hour)})
	if !errors.Is(err, ErrContractRefused) || unobserved.Allowed {
		t.Fatalf("contract without observations: %+v %v", unobserved, err)
	}
	if !contains(unobserved.Blockers, "observed-new-version-writes:0") {
		t.Fatalf("missing observation was not a blocker: %v", unobserved.Blockers)
	}

	// An undeclared or empty window is refused too.
	missing, err := CheckContractNameSplit(h.ctx, h.primary, plan, ContractOptions{})
	if !errors.Is(err, ErrContractRefused) || !contains(missing.Blockers, "observation-window:not-declared") {
		t.Fatalf("contract without a window: %+v %v", missing, err)
	}
	empty, err := CheckContractNameSplit(h.ctx, h.primary, plan, ContractOptions{Since: later, Now: later})
	if !errors.Is(err, ErrContractRefused) || !contains(empty.Blockers, "observation-window:empty") {
		t.Fatalf("contract with an empty window: %+v %v", empty, err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
