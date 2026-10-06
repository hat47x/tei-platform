package xeadsource

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParsePreservesUnknownConstructsAndBindsExactBytes(t *testing.T) {
	source := []byte(`<System Version="1"><Table ID="T1"><TableField ID="F1"/><FutureThing z="2" a="1"> raw </FutureThing></Table></System>`)
	got, err := Parse(FormatXead, source)
	if err != nil { t.Fatal(err) }
	if got.SHA256 != "d7199339eced69fa7c01d12ec244225ad67c6080ed7e31f20a21ff260ecfb253" {
		t.Fatalf("unexpected digest: %s", got.SHA256)
	}
	if !reflect.DeepEqual(got.UnknownNames, []string{"FutureThing"}) {
		t.Fatalf("unknown constructs were not preserved: %#v", got.UnknownNames)
	}
	future := got.Root.Children[0].Children[1]
	if future.Known || future.Text != "raw" || future.Locator != "/System[1]/Table[1]/FutureThing[2]" {
		t.Fatalf("unexpected future node: %#v", future)
	}
	if !reflect.DeepEqual(future.Attributes, []Attribute{{Name:"a", Value:"1"}, {Name:"z", Value:"2"}}) {
		t.Fatalf("attributes not preserved deterministically: %#v", future.Attributes)
	}
}

func TestParseIsDeterministic(t *testing.T) {
	source := []byte(`<System><Function ID="F1"/><Unknown b="2" a="1"/></System>`)
	a, err := Parse(FormatXeaf, source)
	if err != nil { t.Fatal(err) }
	b, err := Parse(FormatXeaf, source)
	if err != nil { t.Fatal(err) }
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	if string(aj) != string(bj) {
		t.Fatalf("same bytes produced different output\n%s\n%s", aj, bj)
	}
}

func TestElementReorderChangesSnapshotLocatorWithoutInventingIdentity(t *testing.T) {
	a, err := Parse(FormatXead, []byte(`<System><Table ID="A"/><Table ID="B"/></System>`))
	if err != nil { t.Fatal(err) }
	b, err := Parse(FormatXead, []byte(`<System><Table ID="B"/><Table ID="A"/></System>`))
	if err != nil { t.Fatal(err) }
	if a.SHA256 == b.SHA256 {
		t.Fatal("different exact snapshots must have different digests")
	}
	if a.Root.Children[0].Locator != b.Root.Children[0].Locator {
		t.Fatal("locator shape should remain snapshot-local and structural")
	}
	if a.Root.Children[0].Attributes[0].Value == b.Root.Children[0].Attributes[0].Value {
		t.Fatal("test fixture did not reorder legacy IDs")
	}
}

func TestFormatKnownSetsDoNotConflateXeadAndXeafFunctionSurfaces(t *testing.T) {
	xead, err := Parse(FormatXead, []byte(`<System><Task/><Script/></System>`))
	if err != nil { t.Fatal(err) }
	xeaf, err := Parse(FormatXeaf, []byte(`<System><Task/><Script/></System>`))
	if err != nil { t.Fatal(err) }
	if !reflect.DeepEqual(xead.UnknownNames, []string{"Script"}) {
		t.Fatalf("xead unknowns: %#v", xead.UnknownNames)
	}
	if !reflect.DeepEqual(xeaf.UnknownNames, []string{"Task"}) {
		t.Fatalf("xeaf unknowns: %#v", xeaf.UnknownNames)
	}
}

func TestRejectsUnsupportedFormat(t *testing.T) {
	if _, err := Parse(Format("other"), []byte("<System/>")); err == nil {
		t.Fatal("expected unsupported format error")
	}
}
