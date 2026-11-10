package persistence

import "testing"

// The value lists and Valid must stay one answer: the diagnostics quote the
// list, so a list that omits an accepted value would tell an author that their
// working declaration is wrong (#2094 round 54).
func TestStorageTypesAndValidAgree(t *testing.T) {
	declared := []StorageType{StorageInt64, StorageVarchar, StorageText, StorageDateTimeTZ, StorageDecimal, StorageJSON}
	if len(declared) != len(StorageTypes) {
		t.Fatalf("StorageTypes has %d entries but %d are declared; update both together", len(StorageTypes), len(declared))
	}
	for _, v := range StorageTypes {
		if !v.Valid() {
			t.Errorf("StorageTypes lists %q but Valid() rejects it", v)
		}
	}
	seen := map[StorageType]bool{}
	for _, v := range StorageTypes {
		seen[v] = true
	}
	for _, v := range declared {
		if !seen[v] {
			t.Errorf("declared storage type %q is missing from StorageTypes", v)
		}
	}
	for _, bad := range []StorageType{"", "INT64", "int64 ", "numberish", "datetime-tz "} {
		if bad.Valid() {
			t.Errorf("Valid() accepted %q", bad)
		}
	}
}

func TestNullabilitiesAndValidAgree(t *testing.T) {
	declared := []Nullability{Required, Optional}
	if len(declared) != len(Nullabilities) {
		t.Fatalf("Nullabilities has %d entries but %d are declared; update both together", len(Nullabilities), len(declared))
	}
	for _, v := range Nullabilities {
		if !v.Valid() {
			t.Errorf("Nullabilities lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []Nullability{"", "Required", "maybe", "required "} {
		if bad.Valid() {
			t.Errorf("Valid() accepted %q", bad)
		}
	}
}
