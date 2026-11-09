package orderassistoffline

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// Tests in this file need neither a database nor Node.js.

const sampleDoc = `{"quoteId":"Q-1","customerCode":"C-100","currency":"JPY","discountRateBp":500,"futureTop":{"x":1},
"lines":[{"sku":"PEN","quantity":3,"unitPriceMinor":250,"priceListVersion":1},
{"sku":"BOOK","quantity":2,"unitPriceMinor":400,"priceListVersion":1,"packaging":"box","futureLine":[1,2]}]}`

func TestDocumentKeepsUnknownMembersThroughEdits(t *testing.T) {
	d, err := ParseDoc([]byte(sampleDoc))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Set("lines/BOOK/quantity", rawInt(6)); err != nil {
		t.Fatal(err)
	}
	again, err := ParseDoc(d.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{"futureTop": `{"x":1}`, "discountRateBp": "500", "lines/BOOK/packaging": `"box"`,
		"lines/BOOK/futureLine": "[1,2]", "lines/BOOK/quantity": "6", "lines/PEN/quantity": "3"} {
		got, ok := again.Get(path)
		if !ok || !EqualJSON(got, json.RawMessage(want)) {
			t.Errorf("%s = %s, want %s", path, got, want)
		}
	}
	naive := naiveDoc(again, 1)
	for _, path := range []string{"futureTop", "discountRateBp", "lines/BOOK/packaging", "lines/BOOK/futureLine"} {
		if _, ok := naive.Get(path); ok {
			t.Errorf("naive generation-1 decode kept %s", path)
		}
	}
	if _, ok := naiveDoc(again, 2).Get("lines/BOOK/packaging"); !ok {
		t.Error("generation-2 decode dropped a generation-2 member")
	}
	if err := d.Set("lines/BOOK/sku", rawString("X")); err == nil {
		t.Error("sku must not be settable")
	}
	if _, err := ParseDoc([]byte(`{"lines":[{"sku":"A"},{"sku":"A"}]}`)); !errors.Is(err, ErrDocument) {
		t.Errorf("duplicate SKU accepted: %v", err)
	}
}

func TestFieldGenerations(t *testing.T) {
	cases := map[string][2]bool{ // path -> known to gen1, gen2
		"discountRateBp":             {false, true},
		"customerCode":               {true, true},
		"lines/PEN/quantity":         {true, true},
		"lines/PEN/priceListVersion": {true, true},
		"lines/PEN/packaging":        {false, true},
		"lines/PEN/futureLine":       {false, false},
		"lines/PEN":                  {false, false},
	}
	for path, want := range cases {
		if got := [2]bool{KnownToGeneration(path, 1), KnownToGeneration(path, 2)}; got != want {
			t.Errorf("%s: got %v want %v", path, got, want)
		}
	}
}

func TestQuoteRequestAndDigest(t *testing.T) {
	d, _ := ParseDoc([]byte(sampleDoc))
	g1, _ := d.QuoteRequest(1)
	g2, _ := d.QuoteRequest(2)
	if strings.Contains(string(g1), "discountRateBp") || !strings.Contains(string(g2), `"discountRateBp":500`) {
		t.Fatalf("requests: %s / %s", g1, g2)
	}
	j := Judgment{RuleVersion: Gen2.RuleVersion, Result: QuoteResult{SubtotalMinor: 1550, NetMinor: 1473}}
	base := QuoteDigest(d, j, 1)
	other := d.Clone()
	_ = other.Set("lines/BOOK/packaging", rawString("bag"))
	if QuoteDigest(other, j, 1) == base {
		t.Error("digest ignores a generation-2 member")
	}
	if QuoteDigest(d, j, 2) == base {
		t.Error("digest ignores the price list version")
	}
	j2 := j
	j2.Result.NetMinor = 1474
	if QuoteDigest(d, j2, 1) == base {
		t.Error("digest ignores the decided result")
	}
	if QuoteDigest(d.Clone(), j, 1) != base {
		t.Error("digest is not stable for an equal document")
	}
}

func TestPoliciesAndMigration(t *testing.T) {
	v1, err := LoadPolicy("testdata/approval-policy-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	v2, err := LoadPolicy("testdata/approval-policy-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	m, err := LoadMigration("testdata/policy-migration-v1-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Validate(v1, v2); err != nil {
		t.Fatal(err)
	}
	if _, mapped := m.StageMap["S2-sales-head"]; mapped {
		t.Fatal("the migration fixture must leave S2-sales-head unmapped")
	}
	if policyDigest(v1) == policyDigest(v2) {
		t.Fatal("policy digests collide")
	}
	bad := m
	bad.StageMap = map[string]string{"S1-manager": "T9-unknown"}
	if err := bad.Validate(v1, v2); err == nil {
		t.Error("migration to an unknown stage accepted")
	}
	forward := v1
	forward.Stages = append([]Stage(nil), v1.Stages...)
	forward.Stages[0].SendbackTo = "S3-director"
	if err := forward.Validate(); err == nil {
		t.Error("sendback to a later stage accepted")
	}
	undeclared := v1
	undeclared.SendbackInvalidates = ""
	if err := undeclared.Validate(); err == nil {
		t.Error("undeclared sendback scope accepted")
	}
}

func TestChainVerify(t *testing.T) {
	id := Identity{QuoteID: "Q", RuleVersion: "r", TerminalAGeneration2: "a", TerminalBGeneration1: "b",
		ServerRealization: "s", BatchRealization: "w", WASIRunner: "n", PolicyDigest: "p"}
	var c Chain
	c.Append("offline", id, "", true, map[string]any{"n": 1})
	c.Append("revalidated", id, "sha256:d3", true, nil)
	c.Append("approved", id, "sha256:d3", true, nil)
	if err := c.Verify([]string{"offline", "approved"}, "revalidated"); err != nil {
		t.Fatal(err)
	}
	tampered := Chain{Records: append([]Record(nil), c.Records...)}
	tampered.Records[1].Result = map[string]any{"n": 2}
	if err := tampered.Verify(nil, "revalidated"); !errors.Is(err, ErrChain) {
		t.Errorf("tampered record accepted: %v", err)
	}
	var drift Chain
	drift.Append("revalidated", id, "sha256:d3", true, nil)
	drift.Append("credit", id, "sha256:d2", true, nil)
	if err := drift.Verify(nil, "revalidated"); !errors.Is(err, ErrChain) {
		t.Errorf("a different quote digest after the carry step was accepted: %v", err)
	}
	other := id
	other.PolicyDigest = "q"
	var mixed Chain
	mixed.Append("revalidated", id, "sha256:d3", true, nil)
	mixed.Append("credit", other, "sha256:d3", true, nil)
	if err := mixed.Verify(nil, "revalidated"); !errors.Is(err, ErrChain) {
		t.Errorf("identity change accepted: %v", err)
	}
}

func TestCreditKeyBindsCaseAndDigest(t *testing.T) {
	a := CreditKey("CASE-1", "sha256:"+strings.Repeat("a", 64))
	b := CreditKey("CASE-1", "sha256:"+strings.Repeat("b", 64))
	if a == b || a != CreditKey("CASE-1", "sha256:"+strings.Repeat("a", 64)) || strings.ContainsAny(a, "/?#") {
		t.Fatalf("keys: %s %s", a, b)
	}
}
