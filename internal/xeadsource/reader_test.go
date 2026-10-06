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
	if got.SHA256 != "da131513de53bd2ab2ae2ef7e2e4f66876bf0b79c297cf0ba277b4b23a602560" {
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

func TestParseRecognizesCompositeTableKeyWithoutInventingBusinessIdentity(t *testing.T) {
	source := []byte(`<System><Table ID="T1"><TableField ID="F1"/><TableField ID="F2"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="F1" SortKey="1"/><TableKeyField FieldID="F2" SortKey="2"/></TableKey></Table></System>`)
	got, err := Parse(FormatXead, source)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.UnknownNames) != 0 {
		t.Fatalf("known key constructs were reported as unknown: %#v", got.UnknownNames)
	}
	var keyCount, keyFieldCount int
	for _, entry := range got.Inventory {
		switch entry.Name {
		case "TableKey":
			keyCount = entry.Count
		case "TableKeyField":
			keyFieldCount = entry.Count
		}
	}
	if keyCount != 1 || keyFieldCount != 2 {
		t.Fatalf("key inventory drifted: key=%d fields=%d inventory=%#v", keyCount, keyFieldCount, got.Inventory)
	}
	table := got.Root.Children[0]
	key := table.Children[2]
	if key.Name != "TableKey" || !key.Known {
		t.Fatalf("table key was not preserved as a known source construct: %#v", key)
	}
	if !reflect.DeepEqual(key.Attributes, []Attribute{{Name: "ID", Value: "K1"}, {Name: "Type", Value: "PK"}}) {
		t.Fatalf("table key attributes drifted: %#v", key.Attributes)
	}
	if len(key.Children) != 2 ||
		key.Children[0].Name != "TableKeyField" ||
		key.Children[1].Name != "TableKeyField" ||
		!key.Children[0].Known ||
		!key.Children[1].Known {
		t.Fatalf("composite key field membership was not preserved: %#v", key.Children)
	}
}

func TestKnownConstructsCoverObservedReferenceInventory(t *testing.T) {
	xead, err := Parse(FormatXead, []byte(`<System><Subsystem/><Department/><DataType/><Table><TableField/><TableKey><TableKeyField/></TableKey></Table><Relationship/><FunctionType/><Function><FunctionUsedByThis/><IOTable/><IOPanel/><IOSpool/></Function><Role/><Task><TaskAction><TaskFunctionIO/></TaskAction></Task><SubjectArea/><DataflowNode/><DataflowLine/><Instance/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(xead.UnknownNames) != 0 {
		t.Fatalf("observed Xead constructs remain unknown: %#v", xead.UnknownNames)
	}

	xeaf, err := Parse(FormatXeaf, []byte(`<System><Subsystem/><Table><Key/><Script/></Table><Function><Field/><Column/><Button/><Filter/><Refer/><Detail/></Function><Menu/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(xeaf.UnknownNames) != 0 {
		t.Fatalf("observed Xeaf constructs remain unknown: %#v", xeaf.UnknownNames)
	}
}

func TestRejectsUnsupportedFormat(t *testing.T) {
	if _, err := Parse(Format("other"), []byte("<System/>")); err == nil {
		t.Fatal("expected unsupported format error")
	}
}
