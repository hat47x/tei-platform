package main

import (
	"encoding/json"
	"fmt"
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
	for _, forbidden := range []string{"demo.order", "order_no", "kept-locally", "sample prose", "source text that must not be published", "Mystery", "AddedLater", "CustomAttr"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("publication receipt leaked source content %q: %s", forbidden, text)
		}
	}
	if receipt.LocatorCount == 0 || receipt.LocatorSetFingerprint == "" {
		t.Fatalf("locator accounting missing: %+v", receipt)
	}
	if receipt.UnknownElementCount != 1 || receipt.UnknownAttributeCount != 1 {
		t.Fatalf("unknown aggregate accounting elements=%d attributes=%d", receipt.UnknownElementCount, receipt.UnknownAttributeCount)
	}
	var customAttrPreserved bool
	for _, node := range doc.Nodes {
		if node.Kind == "Table" && node.Attributes["CustomAttr"] == "kept-locally" {
			customAttrPreserved = true
			break
		}
	}
	if !customAttrPreserved {
		t.Fatalf("new attribute was silently dropped from local observation: %+v", doc.Nodes)
	}
	if receipt.AttributeCounts["Table"] == 0 {
		t.Fatalf("public attribute occurrence accounting missing: %+v", receipt.AttributeCounts)
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
	for _, factType := range []string{"legacy-table-key", "legacy-key-field-membership", "legacy-relationship-key-reference"} {
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
	doc, err := Observe(FormatXeaf, []byte(`<System><Function KeyFields="row_id"/><Script/><Field/><Column/><Button/><Filter/><Refer Fields="a" ToKeyFields="b" WithKeyFields="c" ToTable="t"/><Detail KeyFields="line_id" HeaderKeyFields="row_id"/><Key Fields="row_id" Type="PK"/><Menu/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range doc.Nodes {
		if !node.Known {
			t.Fatalf("known Xeaf construct reported unknown: %+v", node)
		}
	}
	factTypes := map[string]bool{}
	for _, fact := range doc.Facts {
		factTypes[fact.FactType] = true
	}
	for _, want := range []string{"legacy-function-selection-key", "legacy-detail-key-binding", "runtime-table-key", "runtime-reference-key-binding"} {
		if !factTypes[want] {
			t.Fatalf("missing Xeaf source fact %s: %+v", want, doc.Facts)
		}
	}
}

func TestXMLMarkupIsAccountedWithoutPublishingItsContent(t *testing.T) {
	xmlInput := []byte(`<?xml version="1.0"?><System><!--private comment--><?tool secret?><Table/></System>`)
	doc, err := Observe(FormatXead, xmlInput)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Markup) != 3 {
		t.Fatalf("markup=%+v", doc.Markup)
	}
	receipt := PublicReceipt(doc)
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Contains(text, "private comment") || strings.Contains(text, "secret") {
		t.Fatalf("markup content leaked into publication receipt: %s", text)
	}
	if receipt.MarkupCounts["processing-instruction"] != 2 || receipt.MarkupCounts["comment"] != 1 {
		t.Fatalf("markup counts=%+v", receipt.MarkupCounts)
	}
}

func TestNamespacedLookalikeDoesNotBecomeKnownXeadConstruct(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System xmlns:x="urn:extension"><x:Table ID="not-xead"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Nodes) != 2 || doc.Nodes[1].Known || doc.Nodes[1].Namespace != "urn:extension" {
		t.Fatalf("namespaced lookalike was treated as known: %+v", doc.Nodes)
	}
	if len(doc.Unresolved) != 1 || doc.Unresolved[0].Code != "namespaced-element" {
		t.Fatalf("unresolved=%+v", doc.Unresolved)
	}
}

func TestNamespacedLookalikeDoesNotProduceKnownSourceFacts(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(`<System xmlns:x="urn:extension"><x:TableKey ID="private" Type="PK"/><x:Relationship Table1ID="private"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Facts) != 0 {
		t.Fatalf("namespaced lookalike produced standard Xead facts: %+v", doc.Facts)
	}
	receipt := PublicReceipt(doc)
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private", "TableKey", "Relationship"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("public receipt leaked or promoted namespaced source construct %q: %s", forbidden, encoded)
		}
	}
	if receipt.UnknownElementCount != 2 {
		t.Fatalf("unknown element count=%d want=2", receipt.UnknownElementCount)
	}
}

func TestPublicLocatorDigestDoesNotHashSourceDerivedPath(t *testing.T) {
	doc, err := Observe(FormatXead, []byte(syntheticXead))
	if err != nil {
		t.Fatal(err)
	}
	receipt := PublicReceipt(doc)
	if len(receipt.Unresolved) != 1 {
		t.Fatalf("unresolved=%+v", receipt.Unresolved)
	}
	unknownOrdinal := 0
	unknownPath := ""
	for i, node := range doc.Nodes {
		if !node.Known {
			unknownOrdinal = i + 1
			unknownPath = node.Locator.Path
			break
		}
	}
	if unknownOrdinal == 0 {
		t.Fatal("fixture has no unknown node")
	}
	expected := digestString(fmt.Sprintf("%s|node-ordinal:%d", doc.ArtifactSHA256, unknownOrdinal))
	if receipt.Unresolved[0].LocatorDigest != expected {
		t.Fatalf("public locator digest=%q want ordinal digest=%q", receipt.Unresolved[0].LocatorDigest, expected)
	}
	legacyPathDigest := digestString(doc.ArtifactSHA256 + "|" + unknownPath)
	if receipt.Unresolved[0].LocatorDigest == legacyPathDigest {
		t.Fatal("public locator digest still hashes source-derived structural path")
	}
}

func TestObserveRejectsDuplicateExpandedAttribute(t *testing.T) {
	_, err := Observe(FormatXead, []byte(`<System FormatVersion="1.2" FormatVersion="9.9"/>`))
	if err == nil || !strings.Contains(err.Error(), "duplicate attribute FormatVersion") {
		t.Fatalf("duplicate unqualified attribute was silently normalized: %v", err)
	}

	_, err = Observe(FormatXead, []byte(`<System xmlns:x="urn:private" x:A="1" x:A="2"/>`))
	if err == nil || !strings.Contains(err.Error(), "duplicate attribute {urn:private}A") {
		t.Fatalf("duplicate namespaced attribute was silently normalized: %v", err)
	}
}
