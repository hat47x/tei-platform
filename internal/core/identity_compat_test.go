package core

import (
	"encoding/json"
	"reflect"
	"testing"

	"tei/core/identity"
)

func TestLegacyCoreTypesAreExactlyCanonicalTypes(t *testing.T) {
	var oldID CanonicalID = identity.MustCanonicalID("sales.order")
	var canonicalID identity.CanonicalID = oldID
	var oldName LocalName = identity.MustLocalName("line_item")
	var canonicalName identity.LocalName = oldName
	var oldKind Kind = identity.KindDatabase
	var canonicalKind identity.Kind = oldKind
	var oldRef Ref = identity.Ref{Kind: identity.KindDatabase, ID: canonicalID}
	var canonicalRef identity.Ref = oldRef
	if reflect.TypeOf(oldID) != reflect.TypeOf(canonicalID) ||
		reflect.TypeOf(oldName) != reflect.TypeOf(canonicalName) ||
		reflect.TypeOf(oldKind) != reflect.TypeOf(canonicalKind) ||
		reflect.TypeOf(oldRef) != reflect.TypeOf(canonicalRef) {
		t.Fatal("legacy core defined new Go types rather than aliases")
	}
	if got := oldRef.String(); got != "database:sales.order" {
		t.Fatalf("legacy Ref formatting changed: %q", got)
	}
	for _, pair := range [][2]Kind{
		{KindSemantic, identity.KindSemantic},
		{KindApplication, identity.KindApplication},
		{KindDatabase, identity.KindDatabase},
		{KindContract, identity.KindContract},
		{KindEnvironment, identity.KindEnvironment},
		{KindCapability, identity.KindCapability},
	} {
		if pair[0] != pair[1] {
			t.Fatalf("legacy Kind constant diverged: %q and %q", pair[0], pair[1])
		}
	}
	a, err := json.Marshal(oldRef)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(canonicalRef)
	if err != nil || string(a) != string(b) {
		t.Fatalf("legacy Ref JSON diverged: %s, %s, %v", a, b, err)
	}
}

func TestLegacyCoreHelpersUseOneCanonicalImplementation(t *testing.T) {
	for _, raw := range []string{" sales.order ", "invalid-name", "", "a.b.c"} {
		old, oldErr := ParseCanonicalID(raw)
		now, nowErr := identity.ParseCanonicalID(raw)
		if old != now || (oldErr == nil) != (nowErr == nil) ||
			(oldErr != nil && oldErr.Error() != nowErr.Error()) {
			t.Fatalf("canonical ID drift for %q: old %q %v, new %q %v", raw, old, oldErr, now, nowErr)
		}
	}
	for _, raw := range []string{" orderItem ", "bad@item", "OrderStatus", "abc-def", ""} {
		old, oldErr := SymbolToSegment(raw)
		now, nowErr := identity.SymbolToSegment(raw)
		if old != now || (oldErr == nil) != (nowErr == nil) ||
			(oldErr != nil && oldErr.Error() != nowErr.Error()) {
			t.Fatalf("symbol drift for %q", raw)
		}
	}
	for _, raw := range []string{"1.2.3", "v1.2.3", "", " 01.03.06 "} {
		if ValidNumericVersion(raw) != identity.ValidNumericVersion(raw) {
			t.Fatalf("version drift for %q", raw)
		}
	}
}
