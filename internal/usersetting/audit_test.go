package usersetting_test

import (
	"context"
	"testing"
	"time"

	"tei/internal/usersetting"
)

func auditTable(t *testing.T, withUser bool) usersetting.Table {
	t.Helper()
	declaration, err := usersetting.Resolve(settingWorld(withUser), "pref.setting")
	if err != nil {
		t.Fatal(err)
	}
	table, err := usersetting.NewTable(declaration, []usersetting.Setting{
		{Key: "page_size", Value: "50"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return table
}

// TestApplyAuditedRecordsWhoChangedWhat checks that an accepted change leaves one
// entry naming the setting, the actor, and the previous value. X-TEA wrote settings
// with no record of who changed them (D:Session.java L2112).
func TestApplyAuditedRecordsWhoChangedWhat(t *testing.T) {
	table := auditTable(t, false)
	store := usersetting.NewAuditedStore(nil)
	fixed := time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)
	store.SetClock(func() time.Time { return fixed })

	if err := table.ApplyAudited(context.Background(), store, usersetting.Change{Key: "page_size", Value: "100"}, "alice"); err != nil {
		t.Fatal(err)
	}
	entries := store.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries=%+v", entries)
	}
	entry := entries[0]
	if entry.Actor != "alice" || entry.Key != "page_size" || entry.After != "100" {
		t.Fatalf("entry=%+v", entry)
	}
	if !entry.Had || entry.Before != "50" {
		t.Fatalf("the previous value was not recorded: %+v", entry)
	}
	if !entry.At.Equal(fixed) {
		t.Fatalf("the time is not from the clock: %v", entry.At)
	}
}

// TestApplyAuditedDistinguishesAFirstWrite checks that a setting that did not exist
// has Had=false, so "first write" and "changed from empty" stay different facts.
func TestApplyAuditedDistinguishesAFirstWrite(t *testing.T) {
	declaration, err := usersetting.Resolve(settingWorld(true), "pref.setting")
	if err != nil {
		t.Fatal(err)
	}
	table, err := usersetting.NewTable(declaration, nil)
	if err != nil {
		t.Fatal(err)
	}
	store := usersetting.NewAuditedStore(nil)
	// A table with no rows has no declared key, so the change is refused; the audit
	// path must not record a refused change.
	if err := table.ApplyAudited(context.Background(), store, usersetting.Change{Key: "page_size", User: "alice", Value: "200"}, "alice"); err == nil {
		t.Fatal("a change to an undeclared key was accepted")
	}
	if len(store.Entries()) != 0 {
		t.Fatal("a refused change was recorded")
	}
}

// TestApplyAuditedRefusesAnUnattributedChange checks that a change without an actor
// is refused rather than recorded with an empty actor: an unattributed change is not
// an audit record.
func TestApplyAuditedRefusesAnUnattributedChange(t *testing.T) {
	table := auditTable(t, false)
	store := usersetting.NewAuditedStore(nil)
	if err := table.ApplyAudited(context.Background(), store, usersetting.Change{Key: "page_size", Value: "100"}, ""); err == nil {
		t.Fatal("an unattributed change was accepted")
	}
	if len(store.Entries()) != 0 {
		t.Fatal("an unattributed change was recorded")
	}
	// The store is unchanged too, so the refusal is complete.
	if value, err := table.Settings.Value("page_size", ""); err != nil || value != "50" {
		t.Fatalf("value=%q err=%v", value, err)
	}
}

// TestApplyAuditedKeepsTheLogAcrossChanges checks that the log grows and keeps the
// order, so a reader can see the sequence rather than only the last value.
func TestApplyAuditedKeepsTheLogAcrossChanges(t *testing.T) {
	table := auditTable(t, false)
	store := usersetting.NewAuditedStore(nil)
	ctx := context.Background()
	if err := table.ApplyAudited(ctx, store, usersetting.Change{Key: "page_size", Value: "100"}, "alice"); err != nil {
		t.Fatal(err)
	}
	// The second change is checked against the table as it was at first, which is
	// what a caller holds; the audit records what that table had.
	second := table
	second.Values = []usersetting.Setting{{Key: "page_size", Value: "100"}}
	if err := second.ApplyAudited(ctx, store, usersetting.Change{Key: "page_size", Value: "200"}, "bob"); err != nil {
		t.Fatal(err)
	}
	entries := store.Entries()
	if len(entries) != 2 {
		t.Fatalf("entries=%+v", entries)
	}
	if entries[0].Actor != "alice" || entries[1].Actor != "bob" {
		t.Fatalf("order lost: %+v", entries)
	}
	if entries[1].Before != "100" || entries[1].After != "200" {
		t.Fatalf("second entry=%+v", entries[1])
	}
}
