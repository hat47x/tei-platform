package ordermigrationdualdb

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseChangeQuotingNullAndOldKey(t *testing.T) {
	ch, err := ParseChange(`table omdd.legacy_order: INSERT: order_no[text]:'A''x' ordered_on[date]:'2026-07-01' amount[numeric]:120 qty[bigint]:3 note[text]:null empty[text]:'' sp[text]:'a b'`)
	if err != nil {
		t.Fatal(err)
	}
	want := []Column{{Name: "order_no", Type: "text", Value: "A'x"}, {Name: "ordered_on", Type: "date", Value: "2026-07-01"},
		{Name: "amount", Type: "numeric", Value: "120"}, {Name: "qty", Type: "bigint", Value: "3"}, {Name: "note", Type: "text", Null: true},
		{Name: "empty", Type: "text", Value: ""}, {Name: "sp", Type: "text", Value: "a b"}}
	if ch.Table != "omdd.legacy_order" || ch.Op != "INSERT" || len(ch.New) != len(want) {
		t.Fatalf("%+v", ch)
	}
	for i, c := range want {
		if ch.New[i] != c {
			t.Fatalf("column %d: %+v want %+v", i, ch.New[i], c)
		}
	}
	del, err := ParseChange(`table omdd.legacy_order_line: DELETE: order_no[text]:'SO-1' line_no[integer]:2`)
	if err != nil || del.Op != "DELETE" || len(del.Old) != 2 || del.New != nil {
		t.Fatalf("%+v %v", del, err)
	}
	if c, ok := del.Get("line_no"); !ok || c.Value != "2" {
		t.Fatalf("delete key %+v", c)
	}
	upd, err := ParseChange(`table omdd.legacy_order: UPDATE: old-key: order_no[text]:'A' new-tuple: order_no[text]:'B' status[text]:'open'`)
	if err != nil || len(upd.Old) != 1 || upd.Old[0].Value != "A" || len(upd.New) != 2 || upd.New[0].Value != "B" {
		t.Fatalf("%+v %v", upd, err)
	}
	tr, err := ParseChange(`table omdd.legacy_order: TRUNCATE: (no-flags)`)
	if err != nil || tr.Op != "TRUNCATE" {
		t.Fatalf("%+v %v", tr, err)
	}
	for _, bad := range []string{
		`table omdd.x: INSERT: a[text]:'unterminated`,
		`table omdd.x: INSERT: a[text]:unchanged-toast-datum`,
		`table omdd.x: MERGE: a[text]:'1'`,
		`not a change`,
	} {
		if _, err := ParseChange(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}

func TestGroupTransactionsUsesCommitPosition(t *testing.T) {
	rows := []DecodedRow{
		{"0/1520348", "BEGIN 732"},
		{"0/1520348", `table omdd.legacy_order: INSERT: order_no[text]:'A'`},
		{"0/15204C8", "COMMIT 732"},
		{"0/15204C8", "BEGIN 733"},
		{"0/1520558", "COMMIT 733"},
		{"0/1520558", "BEGIN 734"},
		{"0/1520600", `table omdd.legacy_order: DELETE: order_no[text]:'A'`},
	}
	txs, err := GroupTransactions(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 2 || txs[0].Xid != 732 || txs[0].CommitLSN.String() != "0/15204C8" || len(txs[0].Changes) != 1 || txs[1].Xid != 733 {
		t.Fatalf("%+v", txs)
	}
	if _, err := GroupTransactions([]DecodedRow{{"0/1", "COMMIT 1"}}); err == nil {
		t.Fatal("COMMIT without BEGIN accepted")
	}
	if _, err := GroupTransactions([]DecodedRow{{"0/1", "message: x"}}); err == nil {
		t.Fatal("unknown row accepted")
	}
}

func TestSnapshotVisibilityIsExact(t *testing.T) {
	s, err := ParsePGSnapshot("740:745:741,743")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[uint64]bool{739: true, 740: true, 741: false, 742: true, 743: false, 744: true, 745: false, 900: false}
	for xid, want := range cases {
		if s.Visible(xid) != want {
			t.Fatalf("xid %d visible=%v", xid, !want)
		}
	}
	empty, _ := ParsePGSnapshot("732:732:")
	if empty.Visible(732) || !empty.Visible(731) {
		t.Fatal("empty xip")
	}
	// Epoch extension: a 32-bit xid near the wrap point belongs to the
	// snapshot's epoch or the previous one.
	wrapped := PGSnapshot{Xmin: 1<<32 + 5, Xmax: 1<<32 + 10}
	if wrapped.Extend(7) != 1<<32+7 || wrapped.Extend(0xfffffff0) != 0xfffffff0 {
		t.Fatalf("extend %d %d", wrapped.Extend(7), wrapped.Extend(0xfffffff0))
	}
	if _, err := ParsePGSnapshot("1:2"); err == nil {
		t.Fatal("malformed snapshot accepted")
	}
	a, _ := ParseLSN("0/15204C8")
	b, _ := ParseLSN("1/0")
	if !(a < b) || b.String() != "1/0" {
		t.Fatalf("lsn order %v %v", a, b)
	}
}

func TestItemCodeMapIsDeclaredAndNeverGuessed(t *testing.T) {
	obs, _ := Observe(sampleOrders())
	base, _ := BuildPlan(obs, 2)
	m := []ItemCodeMapping{{"100002", "1000020002"}, {"100001", "1000010001"}, {"100002", "1000020001"}}
	p, err := WithItemCodeMap(base, m)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Verify(); err != nil || p.Digest == base.Digest {
		t.Fatalf("map must be sealed into the digest: %v", err)
	}
	if !p.Declares(KnownDifferenceItemCode) || base.Declares(KnownDifferenceItemCode) {
		t.Fatal("known difference not declared with the map")
	}
	if got := p.ForwardCodes("100001"); len(got) != 1 || got[0] != "1000010001" {
		t.Fatalf("one-to-one %v", got)
	}
	if got := p.ForwardCodes("100002"); len(got) != 2 {
		t.Fatalf("one-to-many %v", got)
	}
	if got := p.ForwardCodes("ITM-A"); len(got) != 0 {
		t.Fatalf("unmapped %v", got)
	}
	if p.LegacyCode("1000020001") != "100002" || p.LegacyCode("1000020002") != "100002" || p.LegacyCode("ITM-A") != "ITM-A" {
		t.Fatal("reverse conversion")
	}
	if _, err := WithItemCodeMap(base, []ItemCodeMapping{{"100001", "X"}, {"100003", "X"}}); err == nil {
		t.Fatal("many-to-many accepted")
	}
}

func TestBillingMockAppliesAKeyOnce(t *testing.T) {
	m := NewBillingMock()
	srv := httptest.NewServer(m)
	defer srv.Close()
	post := func(key, body string, timeout time.Duration) (*http.Response, error) {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+billingProfilesPath, bytes.NewReader([]byte(body)))
		req.Header.Set("Idempotency-Key", key)
		return (&http.Client{Timeout: timeout}).Do(req)
	}
	body := `{"customerCode":"C001","address":"x"}`
	m.HangAfterApply(1, 2*time.Second)
	if _, err := post("k1", body, 200*time.Millisecond); err == nil {
		t.Fatal("expected timeout")
	}
	resp, err := post("k1", body, time.Second)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("replay %v %v", resp, err)
	}
	resp.Body.Close()
	resp, _ = post("k1", `{"customerCode":"C001","address":"y"}`, time.Second)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("key reuse with another body: %d", resp.StatusCode)
	}
	resp.Body.Close()
	m.HangBeforeApply(1, 2*time.Second)
	if _, err := post("k2", body, 200*time.Millisecond); err == nil {
		t.Fatal("expected timeout")
	}
	q, _ := http.Get(srv.URL + billingRequestsPrefix + "k2")
	if q.StatusCode != http.StatusNotFound {
		t.Fatalf("k2 query %d", q.StatusCode)
	}
	q.Body.Close()
	q, _ = http.Get(srv.URL + billingRequestsPrefix + "k1")
	if q.StatusCode != http.StatusOK {
		t.Fatalf("k1 query %d", q.StatusCode)
	}
	q.Body.Close()
	st := m.Stats()
	if st.Applies != 1 || st.DistinctKeys != 1 || st.MaxPerKey != 1 || st.Replays != 2 {
		t.Fatalf("%+v", st)
	}
}

func TestCLIStep4CommandsRequirePlan(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Main([]string{"send-billing"}, &out, &errb); code == 0 || !strings.Contains(errb.String(), "-plan is required") {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
}
