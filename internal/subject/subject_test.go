package subject

import (
	"encoding/json"
	"reflect"
	"testing"

	coresubject "tei/core/subject"
)

func TestLegacySubjectIsCanonicalCoreType(t *testing.T) {
	// This assignment requires a true alias: a separately defined type would
	// require conversion, permitting two sources of semantics to drift.
	var old Subject = coresubject.Subject{Kind: "entity", ID: "sales.invoice"}
	var canonical coresubject.Subject = old
	if old.IsZero() || old != canonical ||
		reflect.TypeOf(old) != reflect.TypeOf(canonical) {
		t.Fatal("legacy Subject no longer shares canonical Core identity")
	}
	oldJSON, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	canonicalJSON, err := json.Marshal(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if string(oldJSON) != string(canonicalJSON) {
		t.Fatalf("legacy JSON drifted: %s != %s", oldJSON, canonicalJSON)
	}
	if !(Subject{}).IsZero() {
		t.Fatal("legacy zero check disagrees with Core")
	}
}
