package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const syntheticXead = `<?xml version="1.0"?>
<System ID="demo" FormatVersion="1">
  <Table ID="demo.order" SortKey="10" CustomAttr="kept-locally">
    <TableField ID="order_no" AttributeType="NATIVE" />
    <TableKey ID="pk" Type="PK" SortKey="1">
      <TableKeyField FieldID="order_no" SortKey="1" />
    </TableKey>
  </Table>
  <Relationship ID="rel1" Table1ID="demo.order" TableKey1ID="pk" Table2ID="demo.customer" TableKey2ID="pk" ExistWhen1="sample prose" />
  <Function ID="fn1" KeyFields="order_no" />
  <Mystery AddedLater="yes">source text that must not be published</Mystery>
</System>`

func TestObserveIsDeterministicAndPreservesUnknownConstructs(t *testing.T) {
	left, err := Observe(FormatXead, []byte(syntheticXead))
	if err != nil {
		t.Fatal(err)
	}
	right, err := Observe(FormatXead, []byte(syntheticXead))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("same bytes produced different observation")
	}
	var mystery *ObservedNode
	for i := range left.Nodes {
		if left.Nodes[i].Kind == "Mystery" {
			mystery = &left.Nodes[i]
			break
		}
	}
	if mystery == nil || mystery.Known {
		t.Fatalf("unknown element was not preserved: %+v", mystery)
	}
	if mystery.Attributes["AddedLater"] != "yes" || !strings.Contains(mystery.Text, "must not be published") {
		t.Fatalf("unknown element content was dropped: %+v", mystery)
	}
}

func TestPublicationReceiptDoesNotContainSourceValuesOrText(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(syntheticXead))
	if err != nil {
		t.Fatal(err)
	}
	receipt := PublicReceipt(doc)
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"demo.order", "order_no", "kept-locally", "sample prose", "source text that must not be published"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("publication receipt leaked source content %q: %s", forbidden, text)
		}
	}
	if receipt.LocatorCount == 0 || receipt.LocatorSetFingerprint == "" {
		t.Fatalf("locator accounting missing: %+v", receipt)
	}
	if receipt.UnknownElements["Mystery"] != 1 {
		t.Fatalf("unknown element accounting=%+v", receipt.UnknownElements)
	}
	if receipt.AttributeNameCounts["Table"]["CustomAttr"] != 1 {
		t.Fatalf("new attribute was silently dropped: %+v", receipt.AttributeNameCounts["Table"])
	}
}

func TestLegacyKeyFactsCarryAuthorityLimits(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(syntheticXead))
	if err != nil {
		t.Fatal(err)
	}
	var kinds = map[string]SourceFact{}
	for _, fact := range doc.Facts {
		kinds[fact.FactType] = fact
	}
	for _, factType := range []string{"legacy-table-key", "legacy-key-field-membership", "legacy-relationship-key-reference", "legacy-function-selection-key"} {
		fact, ok := kinds[factType]
		if !ok {
			t.Fatalf("missing fact %s: %+v", factType, doc.Facts)
		}
		if len(fact.Values) == 0 {
			t.Fatalf("fact %s lost local source values", factType)
		}
		if len(fact.AuthorityLimits) == 0 {
			t.Fatalf("fact %s has no authority limit", factType)
		}
	}
}

func TestElementReorderChangesSnapshotLocatorButNotSourceValuesIntoIdentity(t *testing.T) {
	leftXML := `<System><Table ID="a"/><Table ID="b"/></System>`
	rightXML := `<System><Table ID="b"/><Table ID="a"/></System>`
	left, err := Observe(FormatXead, []byte(leftXML))
	if err != nil {
		t.Fatal(err)
	}
	right, err := Observe(FormatXead, []byte(rightXML))
	if err != nil {
		t.Fatal(err)
	}
	if left.ArtifactSHA256 == right.ArtifactSHA256 {
		t.Fatal("reordered source unexpectedly has same artifact digest")
	}
	if left.Nodes[1].Locator.Path != right.Nodes[1].Locator.Path {
		t.Fatalf("snapshot structural locator should remain positional: %s vs %s", left.Nodes[1].Locator.Path, right.Nodes[1].Locator.Path)
	}
	if left.Nodes[1].Locator.LegacyHints["ID"] == right.Nodes[1].Locator.LegacyHints["ID"] {
		t.Fatal("test did not actually reorder distinct legacy IDs")
	}
}

func TestXeafKnownConstructInventory(t *testing.T) {
	doc, err := Observe(FormatXeaf, []byte(`<System><Function/><Script/><Field/><Column/><Button/><Filter/><Refer/><Detail/><Key/><Menu/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range doc.Nodes {
		if !node.Known {
			t.Fatalf("known Xeaf construct reported unknown: %+v", node)
		}
	}
}
