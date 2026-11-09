package ordermigrationdualdb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleOrders() []Order {
	return []Order{
		{OrderNo: "SO-000002", CustomerCode: "C001", OrderedOn: "2026-08-03", Status: "open", Lines: []Line{
			{LineNo: 2, ItemCode: "ITM-B", Qty: 2, UnitPrice: "900000000000000000", Amount: "1800000000000000000"},
			{LineNo: 1, ItemCode: "ITM-A", Qty: 1, UnitPrice: "120", Amount: "120"},
		}},
		{OrderNo: "SO-000001", CustomerCode: "C002", OrderedOn: "2026-09-30", Status: "open", Lines: []Line{
			{LineNo: 1, ItemCode: "ITM-C", Qty: 3, UnitPrice: "999999999999999999", Amount: "999999999999999999"},
		}},
		{OrderNo: "so-legacy-01", CustomerCode: "", OrderedOn: "2026-09-01", Status: "open", Lines: []Line{
			{LineNo: 1, ItemCode: "ITM-A", Qty: 1, UnitPrice: "1", Amount: "1"},
		}},
	}
}

func TestRowHashIgnoresLineOrderAndSeesOneYen(t *testing.T) {
	o := sampleOrders()[0]
	swapped := o
	swapped.Lines = []Line{o.Lines[1], o.Lines[0]}
	if RowHash(o) != RowHash(swapped) {
		t.Fatal("row hash depends on line read order")
	}
	off := o
	off.Lines = append([]Line(nil), o.Lines...)
	off.Lines[1].Amount = "121"
	if RowHash(o) == RowHash(off) {
		t.Fatal("one yen difference not visible in row hash")
	}
	// Field boundaries are length-prefixed: moving a character between
	// adjacent fields must change the hash.
	a := Order{OrderNo: "SO-1", CustomerCode: "AB", OrderedOn: "2026-01-01", Status: "x"}
	b := Order{OrderNo: "SO-1A", CustomerCode: "B", OrderedOn: "2026-01-01", Status: "x"}
	if RowHash(a) == RowHash(b) {
		t.Fatal("field boundary ambiguity")
	}
}

func TestObserveSumsBeyondInt64(t *testing.T) {
	obs, err := Observe(sampleOrders())
	if err != nil {
		t.Fatal(err)
	}
	if obs.OrderCount != 3 || obs.LineCount != 4 {
		t.Fatalf("counts %d/%d", obs.OrderCount, obs.LineCount)
	}
	if obs.AmountSum != "2800000000000000120" {
		t.Fatalf("sum %s", obs.AmountSum)
	}
	if obs.MonthlyAmountSum["2026-08"] != "1800000000000000120" || obs.MonthlyAmountSum["2026-09"] != "1000000000000000000" {
		t.Fatalf("monthly %v", obs.MonthlyAmountSum)
	}
	bad := sampleOrders()
	bad[0].Lines[0].Amount = "12.5"
	if _, err := Observe(bad); err == nil {
		t.Fatal("fractional yen accepted")
	}
	dup := append(sampleOrders(), sampleOrders()[0])
	if _, err := Observe(dup); err == nil {
		t.Fatal("duplicate key accepted")
	}
}

func TestCompareDetectsOneYenOnOneRow(t *testing.T) {
	src, _ := Observe(sampleOrders())
	same, _ := Observe(sampleOrders())
	if d := Compare(DefaultRules(), src, same); len(d) != 0 {
		t.Fatalf("equal sets differ: %v", d)
	}
	changed := sampleOrders()
	changed[1].Lines[0].Amount = "1000000000000000000"
	dst, _ := Observe(changed)
	got := map[string]Difference{}
	for _, d := range Compare(DefaultRules(), src, dst) {
		got[d.Rule] = d
	}
	for _, rule := range []string{RuleAmountSum, RuleMonthlyAmount, RuleRowHash, RuleRowsetDigest} {
		if _, ok := got[rule]; !ok {
			t.Fatalf("rule %s did not detect the one yen difference: %v", rule, got)
		}
	}
	if _, ok := got[RuleOrderCount]; ok {
		t.Fatal("count rule reported a value-only difference")
	}
	if k := got[RuleRowHash].Keys; len(k) != 1 || k[0] != "SO-000001" {
		t.Fatalf("row-hash keys %v", k)
	}
	if d := Compare([]string{"made-up"}, src, src); len(d) != 1 {
		t.Fatal("unknown rule passed silently")
	}
	missing, _ := Observe(sampleOrders()[:2])
	ks := Compare([]string{RuleKeySet}, src, missing)
	if len(ks) != 1 || ks[0].Keys[0] != "missing:so-legacy-01" {
		t.Fatalf("key set %v", ks)
	}
}

func TestPlanChunksCoverEveryKeyInByteOrder(t *testing.T) {
	obs, _ := Observe(sampleOrders())
	plan, err := BuildPlan(obs, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Verify(); err != nil {
		t.Fatal(err)
	}
	if len(plan.Chunks) != 2 || plan.Chunks[0].FromKey != "" || plan.Chunks[1].FromKey != "so-legacy-01" || plan.Chunks[1].ToKeyExclusive != "" {
		t.Fatalf("chunks %+v", plan.Chunks)
	}
	if plan.ChunkFor("SO-000001").ID != "c0001" || plan.ChunkFor("so-legacy-01").ID != "c0002" || plan.ChunkFor("zz-new").ID != "c0002" || plan.ChunkFor("A").ID != "c0001" {
		t.Fatal("ChunkFor does not follow byte order")
	}
	again, _ := BuildPlan(obs, 2)
	if again.Digest != plan.Digest {
		t.Fatal("plan digest is not deterministic")
	}
	other, _ := BuildPlan(obs, 1)
	if other.Digest == plan.Digest {
		t.Fatal("different chunking produced the same digest")
	}
}

func TestReadPlanRefusesTamperAndUnknownFields(t *testing.T) {
	obs, _ := Observe(sampleOrders())
	plan, _ := BuildPlan(obs, 2)
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.json")
	if err := WritePlan(path, plan); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPlan(path); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	tampered := strings.Replace(string(raw), `"amountSum": "2800000000000000120"`, `"amountSum": "2800000000000000121"`, 1)
	if tampered == string(raw) {
		t.Fatal("fixture did not contain the amount")
	}
	os.WriteFile(path, []byte(tampered), 0o644)
	if _, err := ReadPlan(path); err == nil || !strings.Contains(err.Error(), "plan digest mismatch") {
		t.Fatalf("tampered plan accepted: %v", err)
	}
	var m map[string]any
	json.Unmarshal(raw, &m)
	m["extra"] = true
	extra, _ := json.Marshal(m)
	os.WriteFile(path, extra, 0o644)
	if _, err := ReadPlan(path); err == nil {
		t.Fatal("unknown field accepted")
	}
	gap := plan
	gap.Chunks = append([]Chunk(nil), plan.Chunks...)
	gap.Chunks[1].FromKey = "so-legacy-02"
	gap, _ = Seal(gap)
	if err := gap.Verify(); err == nil {
		t.Fatal("non-contiguous chunks accepted")
	}
}

func TestIdentifiersAreDeterministic(t *testing.T) {
	d := "sha256:0123456789abcdef0123456789abcdef"
	if EffectID(d, "SO-1") != "migrate-allocate:0123456789abcdef:SO-1#1" {
		t.Fatal(EffectID(d, "SO-1"))
	}
	if EffectID(d, "SO-1") == EffectID("sha256:ffffffffffffffff0000", "SO-1") {
		t.Fatal("effect id does not depend on the plan")
	}
}

func TestStatusSettledNeedsObservationOfBothDatabases(t *testing.T) {
	s := StatusReport{Orders: 2, Classes: map[string]int{ClassBothCommitted: 1, ClassCompensated: 1}}
	if s.Settled() {
		t.Fatal("unobserved inventory reported settled")
	}
	s.InventoryObserved = true
	if !s.Settled() {
		t.Fatal("settled state not recognised")
	}
	s.Classes = map[string]int{ClassBothCommitted: 1, ClassPrimaryOnlyAwaiting: 1}
	if s.Settled() {
		t.Fatal("undetermined order folded into settled")
	}
}
