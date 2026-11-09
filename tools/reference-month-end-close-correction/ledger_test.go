package monthendclose

import (
	"bytes"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
)

// syntheticItems are ten items with mixed rates and prices that leave
// fractions of a yen.
func syntheticItems() []ItemDef {
	defs := []struct {
		code, name string
		rate       int
		price      string
	}{
		{"IT-01", "米", 8, "98.73"}, {"IT-02", "茶", 8, "33.33"}, {"IT-03", "菓子", 8, "0.99"}, {"IT-04", "牛乳", 8, "12.34"}, {"IT-05", "パン", 8, "45.67"},
		{"IT-06", "洗剤", 10, "101.37"}, {"IT-07", "電池", 10, "7.77"}, {"IT-08", "文具", 10, "19.99"}, {"IT-09", "工具", 10, "123.45"}, {"IT-10", "雑貨", 10, "66.67"},
	}
	var out []ItemDef
	for _, d := range defs {
		out = append(out, ItemDef{ItemCode: d.code, Name: d.name, TaxRate: d.rate, UnitPrice: d.price, ValidFrom: "2026-01-01", RecordedAt: "2026-08-01T09:00:00+09:00"})
	}
	return out
}

// syntheticOrderEvents returns the events of n order lines in order: per
// order, the registration and then its changes and cancellation by version.
// Every sixth order changes its quantity, every fifteenth twice, every
// twenty-fifth is cancelled.
func syntheticOrderEvents(n int) []OrderEvent {
	var out []OrderEvent
	for i := 1; i <= n; i++ {
		no := fmt.Sprintf("SO-L%03d", i)
		day := 1 + i%30
		ship := fmt.Sprintf("2026-09-%02d", day)
		qty := int64(1 + (i*3)%11)
		item := fmt.Sprintf("IT-%02d", 1+(i*7)%10)
		if item == "IT-03" {
			qty *= 13 // many cheap units
		}
		ev := func(kind string, v int, q int64, recv string) OrderEvent {
			return OrderEvent{EventID: fmt.Sprintf("ev-%s-v%d", no, v), Source: SourceMessage, Kind: kind, OrderNo: no, Version: v, BusinessDate: ship, Qty: q, ReceivedAt: recv}
		}
		reg := ev(EventRegistered, 1, qty, fmt.Sprintf("%sT10:%02d:00+09:00", ship, i%60))
		reg.Customer, reg.ItemCode = fmt.Sprintf("C%02d", 1+i%4), item
		out = append(out, reg)
		v := 1
		if i%6 == 0 {
			v++
			out = append(out, ev(EventChanged, v, qty+2, fmt.Sprintf("%sT15:%02d:00+09:00", ship, i%60)))
		}
		if i%15 == 0 {
			v++
			out = append(out, ev(EventChanged, v, qty+5, fmt.Sprintf("%sT16:%02d:00+09:00", ship, i%60)))
		}
		if i%25 == 0 {
			v++
			out = append(out, ev(EventCancelled, v, 0, fmt.Sprintf("%sT17:%02d:00+09:00", ship, i%60)))
		}
	}
	return out
}

func TestFoldOrderIgnoresArrivalOrder(t *testing.T) {
	evs := syntheticOrderEvents(100)
	byOrder := map[string][]OrderEvent{}
	for _, e := range evs {
		if err := e.Validate(); err != nil {
			t.Fatal(err)
		}
		byOrder[e.OrderNo] = append(byOrder[e.OrderNo], e)
	}
	rng := rand.New(rand.NewSource(7))
	for no, list := range byOrder {
		want := FoldOrder(no, list)
		for k := 0; k < 5; k++ {
			p := append([]OrderEvent(nil), list...)
			rng.Shuffle(len(p), func(i, j int) { p[i], p[j] = p[j], p[i] })
			if got := FoldOrder(no, p); got != want {
				t.Fatalf("%s: %+v, want %+v", no, got, want)
			}
		}
		// Without the registration the order waits, it does not drop events.
		if len(list) > 1 {
			if got := FoldOrder(no, list[1:]); got.Status != "pending-base" || got.Version != want.Version {
				t.Fatalf("%s without registration: %+v", no, got)
			}
		}
	}
	if s := FoldOrder("SO-L025", byOrder["SO-L025"]); s.Status != "cancelled" || s.Qty != 0 {
		t.Fatalf("cancelled order %+v", s)
	}
	if s := FoldOrder("SO-L030", byOrder["SO-L030"]); s.Version != 3 || s.Qty != int64(1+(30*3)%11)+5 {
		t.Fatalf("twice changed order %+v", s)
	}
}

func TestEventDigestIgnoresIdentityAndReception(t *testing.T) {
	e := syntheticOrderEvents(1)[0]
	o := e
	o.EventID, o.Source, o.ReceivedAt = "other", SourceFile, "2026-10-02T18:00:00+09:00"
	if e.ContentDigest() != o.ContentDigest() {
		t.Fatal("same content, different digest")
	}
	o.Qty++
	if e.ContentDigest() == o.ContentDigest() {
		t.Fatal("different content, same digest")
	}
}

func TestSen(t *testing.T) {
	for in, want := range map[string]int64{"0.99": 99, "98.73": 9873, "1": 100, "1.5": 150, "-33.33": -3333, "0": 0} {
		got, err := ParseSen(in)
		if err != nil || got != want {
			t.Fatalf("%s: %d %v", in, got, err)
		}
	}
	for _, bad := range []string{"1.234", "", "x", "1.-2", "+1"} {
		if _, err := ParseSen(bad); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
	if FormatSen(-5) != "-0.05" || FormatSen(123456) != "1234.56" {
		t.Fatal(FormatSen(-5), FormatSen(123456))
	}
}

// The invoice is rounded once per rate: three lines of 33.33 at 8% are
// 79.992 yen of tax in sum, 7 yen, while rounding per line gives 2+2+2.
// Truncation toward zero makes the red invoice's tax the negation; floor
// would make it -8.
func TestGoTaxInvoiceRoundingAndAllocation(t *testing.T) {
	lines := []TaxLine{{1, "C01", "2026-09", 8, 3333}, {2, "C01", "2026-09", 8, 3333}, {3, "C01", "2026-09", 8, 3333},
		{4, "C02", "2026-10", 8, -3333}, {5, "C02", "2026-10", 8, -3333}, {6, "C02", "2026-10", 8, -3333},
		{7, "C01", "2026-09", 10, 10137}}
	r := GoTax(lines)
	want := []InvoiceTax{{"C01", "2026-09", 8, 3, 9999, 7}, {"C01", "2026-09", 10, 1, 10137, 10}, {"C02", "2026-10", 8, 3, -9999, -7}}
	if fmt.Sprint(r.Invoices) != fmt.Sprint(want) {
		t.Fatalf("invoices %+v", r.Invoices)
	}
	// 7*3333/9999 = 2 each, the remainder 1 goes to the smallest slip id
	// among the equal largest amounts.
	if r.Allocation[1] != 3 || r.Allocation[2] != 2 || r.Allocation[3] != 2 || r.Allocation[4] != -3 || r.Allocation[7] != 10 {
		t.Fatalf("allocation %v", r.Allocation)
	}
	var perLine int64
	for _, l := range lines[:3] {
		perLine += l.AmountSen * 8 / 10000
	}
	if perLine != 6 {
		t.Fatalf("per-line rounding %d", perLine)
	}
}

func TestRenderReportIsDeterministic(t *testing.T) {
	h := ReportHeader{Month: "2026-09", Basis: BasisClose, DecisionDigest: "sha256:x", CalendarVersion: 2, CloseDate: "2026-09-30", CloseToken: 2, CutAt: "2026-10-20T00:00:00.123456Z", TaxRule: DefaultTaxRule.ID}
	lines := []ReportLine{{SlipID: 1, Kind: SlipOriginal, OrderNo: "SO-L001", Customer: "C01", ItemCode: "IT-06", Rate: 10, ShipDate: "2026-09-02", PostingMonth: "2026-09", Qty: 3, UnitPriceSen: 10137, AmountSen: 30411}}
	inv := GoTax([]TaxLine{{1, "C01", "2026-09", 10, 30411}}).Invoices
	a, b := RenderReport(h, lines, inv), RenderReport(h, lines, inv)
	if !bytes.Equal(a, b) || !bytes.Contains(a, []byte("line,1,original,SO-L001,C01,IT-06,10,2026-09-02,2026-09,3,101.37,304.11,0,0\n")) ||
		!bytes.Contains(a, []byte("invoice,C01,2026-09,10,1,304.11,30\n")) || !bytes.HasSuffix(a, []byte("total,1,304.11,30\n")) {
		t.Fatalf("report:\n%s", a)
	}
}

func TestPolicyAndTaxRuleFiles(t *testing.T) {
	p, err := ReadLatePolicy(filepath.Join("testdata", "late-arrival-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if p.OnClosedPeriod[EventCancelled] != PolicyHold || p.OnClosedPeriod[EventChanged] != PolicyAdjust {
		t.Fatalf("policy %+v", p)
	}
	bad := LatePolicy{ID: "x", OnClosedPeriod: map[string]string{EventRegistered: PolicyAdjust, EventChanged: "reopen", EventCancelled: PolicyHold, PolicyKeyPrice: PolicyAdjust}}
	if bad.Validate() == nil {
		t.Fatal("reopen accepted")
	}
	if _, err := ReadTaxRule(filepath.Join("testdata", "tax-rule.json")); err != nil {
		t.Fatal(err)
	}
}
