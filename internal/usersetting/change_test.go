package usersetting_test

import (
	"context"
	"testing"

	"tei/internal/usersetting"
)

func changeTable(t *testing.T, withUser bool) usersetting.Table {
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

// TestChangeUpdatesADeclaredKey checks that a declared key is replaced in place and
// the rest of the table is kept.
func TestChangeUpdatesADeclaredKey(t *testing.T) {
	table := changeTable(t, false)
	updated, err := table.Change(usersetting.Change{Key: "page_size", Value: "100"})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 1 || updated[0].Value != "100" {
		t.Fatalf("updated=%+v", updated)
	}
}

// TestChangeRefusesAnUndeclaredKey checks that a misspelt key is refused instead of
// creating a second setting. X-TEA wrote any key the script named, so a typo made a
// setting nobody read.
func TestChangeRefusesAnUndeclaredKey(t *testing.T) {
	table := changeTable(t, false)
	if _, err := table.Change(usersetting.Change{Key: "page_sise", Value: "100"}); err == nil {
		t.Fatal("an undeclared key was created")
	}
}

// TestChangeAddsAUsersRow checks that a user's own row is added beside the shared
// one, so the shared default stays for everyone else.
func TestChangeAddsAUsersRow(t *testing.T) {
	table := changeTable(t, true)
	updated, err := table.Change(usersetting.Change{Key: "page_size", User: "alice", Value: "200"})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 2 {
		t.Fatalf("updated=%+v", updated)
	}
	indexed, err := usersetting.NewSettings(true, updated)
	if err != nil {
		t.Fatal(err)
	}
	alice, err := indexed.Value("page_size", "alice")
	if err != nil || alice != "200" {
		t.Fatalf("alice=%q err=%v", alice, err)
	}
	shared, err := indexed.Value("page_size", "bob")
	if err != nil || shared != "50" {
		t.Fatalf("shared=%q err=%v", shared, err)
	}
}

// TestChangeRefusesAUserOnASharedData checks that a Data without the user role
// refuses a user, because there is no row to hold it.
func TestChangeRefusesAUserOnASharedData(t *testing.T) {
	table := changeTable(t, false)
	if _, err := table.Change(usersetting.Change{Key: "page_size", User: "alice", Value: "200"}); err == nil {
		t.Fatal("a user was accepted by a shared data")
	}
}

// TestChangeRequiresAKey checks that a change without a key is refused.
func TestChangeRequiresAKey(t *testing.T) {
	table := changeTable(t, false)
	if _, err := table.Change(usersetting.Change{Value: "100"}); err == nil {
		t.Fatal("a keyless change was accepted")
	}
}

// TestApplyRecordsTheCheckedList checks that Apply checks and writes in one call, so
// a caller cannot check with one table and write another's list.
func TestApplyRecordsTheCheckedList(t *testing.T) {
	table := changeTable(t, false)
	store := usersetting.NewMemoryStore()
	ctx := context.Background()
	if err := table.Apply(ctx, store, usersetting.Change{Key: "page_size", Value: "100"}); err != nil {
		t.Fatal(err)
	}
	recorded, ok := store.Get("pref.setting")
	if !ok || len(recorded) != 1 || recorded[0].Value != "100" {
		t.Fatalf("recorded=%+v ok=%v", recorded, ok)
	}
	// A refused change writes nothing.
	if err := table.Apply(ctx, store, usersetting.Change{Key: "page_sise", Value: "1"}); err == nil {
		t.Fatal("an undeclared key was applied")
	}
	recorded, _ = store.Get("pref.setting")
	if len(recorded) != 1 || recorded[0].Value != "100" {
		t.Fatalf("the refused change reached the store: %+v", recorded)
	}
}

// TestApplyRefusesAMissingStore checks the configuration fault.
func TestApplyRefusesAMissingStore(t *testing.T) {
	table := changeTable(t, false)
	if err := table.Apply(context.Background(), nil, usersetting.Change{Key: "page_size", Value: "1"}); err == nil {
		t.Fatal("a missing store was accepted")
	}
}

// TestMemoryStoreDistinguishesAbsentFromEmpty checks that a Data nothing was written
// for is different from a list written empty.
func TestMemoryStoreDistinguishesAbsentFromEmpty(t *testing.T) {
	store := usersetting.NewMemoryStore()
	if _, ok := store.Get("pref.setting"); ok {
		t.Fatal("an unwritten data reported a list")
	}
	if err := store.Put(context.Background(), "pref.setting", nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get("pref.setting"); !ok {
		t.Fatal("an empty written list was reported as absent")
	}
}
