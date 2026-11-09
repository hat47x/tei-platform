package orderwebmember

import (
	"context"
	"errors"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	webapicontract "tei/reference/web-api-contract"
)

func workspaceRoot() string { return filepath.Join("..", "..", "reference", "workspace") }

func projections(t *testing.T) (Projection, Projection) {
	t.Helper()
	loaded, err := LoadWorkspace(workspaceRoot())
	if err != nil {
		t.Fatal(err)
	}
	m, err := DeriveProjection(loaded, KindMember)
	if err != nil {
		t.Fatal(err)
	}
	s, err := DeriveProjection(loaded, KindStaff)
	if err != nil {
		t.Fatal(err)
	}
	return m, s
}

func names(p Projection) []string {
	var out []string
	for _, f := range p.Fields {
		out = append(out, f.Name)
	}
	return out
}

func TestProjectionsFromOneReadPlan(t *testing.T) {
	m, s := projections(t)
	if m.Source != s.Source || m.Source.QueryID != "order_entry.recent_orders" || m.Source.DataID != "sales.order" {
		t.Fatalf("projections must come from the same step-1 read plan: %+v %+v", m.Source, s.Source)
	}
	for _, internal := range InternalFieldNames() {
		if m.Has(internal) {
			t.Fatalf("member projection contains internal field %s", internal)
		}
		if !s.Has(internal) {
			t.Fatalf("staff projection lacks %s", internal)
		}
	}
	if !slices.Equal(m.Writable, []string{"deliveryNote"}) {
		t.Fatalf("member accepted fields: %v", m.Writable)
	}
	canon := 0
	for _, f := range m.Fields {
		if f.Origin == "step1-read-plan" {
			canon++
			if f.AttributeID == "" {
				t.Fatalf("canonical field without attribute: %+v", f)
			}
		}
	}
	if canon != 5 {
		t.Fatalf("member projection should carry the 5 read-plan fields: %v", names(m))
	}
	if !m.VerifyDigest() || !s.VerifyDigest() || m.Digest == s.Digest {
		t.Fatal("digests")
	}
	m2, _ := projections(t)
	if m2.Digest != m.Digest {
		t.Fatal("derivation is not deterministic")
	}
	tampered := m
	tampered.Fields = append(slices.Clone(m.Fields), ProjectedField{Name: "cost", Column: "cost", Policy: webapicontract.FieldPolicy{Select: true}})
	if tampered.VerifyDigest() {
		t.Fatal("a projection widened after derivation must not verify")
	}
}

func TestMemberReadAndUpdateAllowlists(t *testing.T) {
	m, _ := projections(t)
	ctx := webapicontract.TrustedReadContext{PrincipalID: "M-A", RowScopeID: "accounts@seq1", PolicyRevision: "1"}
	plan, err := m.BindRead(url.Values{}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range plan.Selected {
		if slices.Contains(InternalFieldNames(), n) {
			t.Fatalf("default projection selected %s", n)
		}
	}
	for _, q := range []url.Values{
		{"$select": {"cost"}}, {"$select": {"orderId,staffMemo"}}, {"$filter": {"creditClass eq 'X'"}},
		{"$orderby": {"cost desc"}}, {"$filter": {"total gt 1"}}, {"$count": {"true"}}, {"$select": {"order_id"}},
	} {
		if _, err := m.BindRead(q, ctx); !errors.Is(err, webapicontract.ErrRejected) {
			t.Fatalf("%v accepted", q)
		}
	}
	for _, ch := range []map[string]any{{"status": "confirmed"}, {"total": 1}, {"cost": 0}, {"deliveryNote": "x", "status": "confirmed"}, {"version": 9}} {
		if _, err := m.BindUpdate("member.update-order", "1", ch, "M-A"); err == nil {
			t.Fatalf("%v accepted", ch)
		}
	}
	if _, err := m.BindUpdate("member.update-order", "1", map[string]any{"deliveryNote": "置き配"}, "M-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.BindUpdate("member.update-order", "", map[string]any{"deliveryNote": "x"}, "M-A"); err == nil {
		t.Fatal("update without revision accepted")
	}
}

func TestMemberQueryAppliesScopeAndSnapshotFirst(t *testing.T) {
	m, _ := projections(t)
	ctx := webapicontract.TrustedReadContext{PrincipalID: "M-A", RowScopeID: "s", PolicyRevision: "1"}
	plan, err := m.BindRead(url.Values{"$filter": {"orderId eq 1001"}, "$orderby": {"orderedAt desc"}, "$top": {"5"}}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	q, args, err := memberQuery(m, plan, Grant{Accounts: []int64{1}, Seq: 7}, nil, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "o.account_id = ANY($1) AND o.created_seq <= $2 AND o.order_id = $3::bigint") || len(args) != 4 {
		t.Fatalf("scope must precede the client filter: %s %v", q, args)
	}
	for _, internal := range []string{"cost", "credit_class", "staff_memo", "*"} {
		if strings.Contains(strings.SplitN(q, " FROM ", 2)[0], internal) {
			t.Fatalf("select list contains %s: %s", internal, q)
		}
	}
	q2, _, _ := memberQuery(m, plan, Grant{Accounts: []int64{1}, Seq: 7}, nil, true, 0)
	if strings.Contains(q2, "created_seq") {
		t.Fatal("negative control should drop the snapshot predicate")
	}
}

func TestLimiterShedsWithRetryAfter(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewLimiter(LimitConfig{PerPrincipalRate: 2, PerPrincipalBurst: 2, TierRate: 100, TierBurst: 100, TierConcurrency: 1, AdmitWait: time.Millisecond})
	l.now = func() time.Time { return now }
	d1 := l.Admit("a")
	if !d1.Admitted {
		t.Fatal("first")
	}
	if d := l.Admit("b"); d.Admitted || d.Reason != "tier-concurrency" || d.RetryAfter != 1 {
		t.Fatalf("slot held: %+v", d)
	}
	d1.Release()
	l.Admit("a").Release()
	d := l.Admit("a")
	if d.Admitted || d.Reason != "principal-rate" || d.RetryAfter < 1 {
		t.Fatalf("bucket: %+v", d)
	}
	now = now.Add(600 * time.Millisecond)
	if d := l.Admit("a"); !d.Admitted {
		t.Fatalf("refill: %+v", d)
	} else {
		d.Release()
	}
}

func TestStockCacheDelayedFill(t *testing.T) {
	clock := time.Unix(100, 0)
	for _, unsafe := range []bool{false, true} {
		c := NewStockCache(time.Second)
		c.now = func() time.Time { return clock }
		c.UnsafeStampAtSet = unsafe
		version := int64(1)
		c.FillHook = func(ctx context.Context, v StockValue) {
			// The writer commits version 2 and its invalidation arrives while
			// the fill of version 1 is delayed by 1.5s.
			version = 2
			c.Invalidate("I", 2)
			clock = clock.Add(1500 * time.Millisecond)
		}
		load := func(context.Context) (StockValue, error) { return StockValue{ItemCode: "I", Version: version}, nil }
		v, hit, _ := c.Get(context.Background(), "I", load)
		if v.Version != 1 || hit {
			t.Fatalf("first read: %+v", v)
		}
		c.FillHook = nil
		v, hit, _ = c.Get(context.Background(), "I", load)
		if unsafe && (!hit || v.Version != 1) {
			t.Fatalf("negative control should serve the stale value from the cache: %+v %v", v, hit)
		}
		if !unsafe && (hit || v.Version != 2) {
			t.Fatalf("delayed old fill must not be served: %+v %v", v, hit)
		}
	}
}

func TestChainBreaksWhenAnIdentityChanges(t *testing.T) {
	id := Identity{ReadPlan: ReadPlanSource{QueryID: "q", DataID: "d"}, MemberProjection: "m", StaffProjection: "s",
		MemberUnitConfig: "mc", StaffUnitConfig: "sc", MemberUnitProjection: "m", StaffUnitProjection: "s"}
	var c Chain
	c.Append("a", id, true, map[string]any{"x": 1})
	c.Append("b", id, true, nil)
	if err := c.Verify([]string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if err := c.Verify([]string{"a", "b", "c"}); !errors.Is(err, ErrChain) {
		t.Fatal("missing step accepted")
	}
	for _, mutate := range []func(*Chain){
		func(c *Chain) { c.Records[1].Identity.StaffUnitConfig = "other" },
		func(c *Chain) { c.Records[0].Result["x"] = 2.0 },
		func(c *Chain) { c.Records[1].Prev = "" },
		func(c *Chain) { c.Records[0].Passed = false },
	} {
		cc := Chain{Records: slices.Clone(c.Records)}
		cc.Records[0].Result = map[string]any{"x": 1.0}
		mutate(&cc)
		if err := cc.Verify([]string{"a", "b"}); !errors.Is(err, ErrChain) {
			t.Fatalf("mutated chain verified")
		}
	}
	bad := id
	bad.MemberUnitProjection = "other"
	var c2 Chain
	c2.Append("a", bad, true, nil)
	if err := c2.Verify([]string{"a"}); !errors.Is(err, ErrChain) {
		t.Fatal("a unit reporting another projection must break the chain")
	}
}
