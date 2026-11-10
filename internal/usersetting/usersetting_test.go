package usersetting_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/usersetting"
)

func settingWorld(userRole bool) ir.World {
	attributes := []semantic.Attribute{
		{ID: "pref.setting.key", TypeRef: "std.identifier", Role: usersetting.RoleKey},
		{ID: "pref.setting.value", TypeRef: "std.text", Role: usersetting.RoleValue},
	}
	if userRole {
		attributes = append(attributes, semantic.Attribute{ID: "pref.setting.user", TypeRef: "std.identifier", Role: usersetting.RoleUser})
	}
	return ir.World{Semantics: []semantic.Namespace{{
		ID:   "pref",
		Data: []semantic.Data{{ID: "pref.setting", Attributes: attributes}},
	}}}
}

// TestResolveRequiresKeyAndValue checks that a setting Data without the key or the
// value is refused with the role name, so the fix is the declaration rather than a
// guess at a column.
func TestResolveRequiresKeyAndValue(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "pref",
		Data: []semantic.Data{{ID: "pref.setting", Attributes: []semantic.Attribute{
			{ID: "pref.setting.key", TypeRef: "std.identifier", Role: usersetting.RoleKey},
		}}},
	}}}
	_, err := usersetting.Resolve(world, "pref.setting")
	if err == nil || !strings.Contains(err.Error(), usersetting.RoleValue) {
		t.Fatalf("err=%v", err)
	}
	if _, err := usersetting.Resolve(world, "pref.missing"); err == nil {
		t.Fatal("an undeclared data was accepted")
	}
}

// TestBindRequiresAKey checks that a row without a key is reported with its
// position: a keyless row cannot be read by anyone.
func TestBindRequiresAKey(t *testing.T) {
	declaration, err := usersetting.Resolve(settingWorld(false), "pref.setting")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := declaration.Bind([]map[core.CanonicalID]string{
		{"pref.setting.key": "", "pref.setting.value": "x"},
	}); err == nil {
		t.Fatal("a keyless row was accepted")
	}
}

// TestValuePrefersTheUsersRow checks that a user's row wins over the shared one,
// which is what setUserValueOf did for a logged-in user (D:Session.java L2112).
func TestValuePrefersTheUsersRow(t *testing.T) {
	declaration, err := usersetting.Resolve(settingWorld(true), "pref.setting")
	if err != nil {
		t.Fatal(err)
	}
	bound, err := declaration.Bind([]map[core.CanonicalID]string{
		{"pref.setting.key": "page_size", "pref.setting.value": "50", "pref.setting.user": ""},
		{"pref.setting.key": "page_size", "pref.setting.value": "200", "pref.setting.user": "alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	settings, err := usersetting.NewSettings(true, bound)
	if err != nil {
		t.Fatal(err)
	}
	value, err := settings.Value("page_size", "alice")
	if err != nil || value != "200" {
		t.Fatalf("value=%q err=%v", value, err)
	}
	// A user without a row of their own reads the shared default.
	shared, err := settings.Value("page_size", "bob")
	if err != nil || shared != "50" {
		t.Fatalf("value=%q err=%v", shared, err)
	}
}

// TestValueReportsAnUndeclaredKey checks that a key nobody declares is reported
// rather than read as empty. X-TEA could not tell a missing key from an unset
// value, so a typo quietly emptied a screen's setting.
func TestValueReportsAnUndeclaredKey(t *testing.T) {
	declaration, err := usersetting.Resolve(settingWorld(false), "pref.setting")
	if err != nil {
		t.Fatal(err)
	}
	bound, err := declaration.Bind([]map[core.CanonicalID]string{
		{"pref.setting.key": "page_size", "pref.setting.value": "50"},
	})
	if err != nil {
		t.Fatal(err)
	}
	settings, err := usersetting.NewSettings(false, bound)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := settings.Value("page_sise", "alice"); err == nil {
		t.Fatal("a misspelt key was read as empty")
	}
}

// TestValueReportsAKeyDeclaredForAnotherUser checks that a key declared only for
// another user is reported, not silently emptied.
func TestValueReportsAKeyDeclaredForAnotherUser(t *testing.T) {
	declaration, err := usersetting.Resolve(settingWorld(true), "pref.setting")
	if err != nil {
		t.Fatal(err)
	}
	bound, err := declaration.Bind([]map[core.CanonicalID]string{
		{"pref.setting.key": "theme", "pref.setting.value": "dark", "pref.setting.user": "alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	settings, err := usersetting.NewSettings(true, bound)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := settings.Value("theme", "bob"); err == nil {
		t.Fatal("another user's key was read as empty")
	}
}

// TestNewSettingsRefusesADuplicateKey checks that two rows for one key and user are
// refused, so the value does not depend on row order.
func TestNewSettingsRefusesADuplicateKey(t *testing.T) {
	if _, err := usersetting.NewSettings(false, []usersetting.Setting{
		{Key: "page_size", Value: "50"},
		{Key: "page_size", Value: "100"},
	}); err == nil {
		t.Fatal("a duplicate key was accepted")
	}
	if _, err := usersetting.NewSettings(true, []usersetting.Setting{
		{Key: "theme", Value: "dark", User: "alice"},
		{Key: "theme", Value: "light", User: "alice"},
	}); err == nil {
		t.Fatal("a duplicate key for one user was accepted")
	}
}

// TestKeysAreStable checks that the key list is sorted, so two runs can be compared.
func TestKeysAreStable(t *testing.T) {
	settings, err := usersetting.NewSettings(true, []usersetting.Setting{
		{Key: "zeta", Value: "1"},
		{Key: "alpha", Value: "2", User: "alice"},
		{Key: "beta", Value: "3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	keys := settings.Keys()
	want := []string{"alpha", "beta", "zeta"}
	if len(keys) != len(want) {
		t.Fatalf("keys=%v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys=%v", keys)
		}
	}
}
