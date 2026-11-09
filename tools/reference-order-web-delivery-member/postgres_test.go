package orderwebmember

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// ---------------------------------------------------------------- P-B2C-01

func scenarioPB2C01(t *testing.T, h *harness, m, s *unitProc) map[string]any {
	res := map[string]any{}
	memberNames := map[string]bool{}
	for _, f := range h.member.Fields {
		memberNames[f.Name] = true
	}
	internal := InternalFieldNames()

	// Own list: only the projection's fields, only the caller's rows.
	var list struct {
		Orders []map[string]any `json:"orders"`
	}
	r := m.must("GET", "/member/orders?$top=100", MemberA, nil, nil)
	if r.status != 200 {
		t.Fatalf("own list: %d %s", r.status, r.body)
	}
	r.json(t, &list)
	extraKeys := 0
	for _, o := range list.Orders {
		for k := range o {
			if !memberNames[k] {
				extraKeys++
			}
		}
		if o["customerId"] != float64(AccountA) {
			t.Errorf("foreign row in own list: %v", o)
		}
	}
	if len(list.Orders) != 20 || extraKeys != 0 {
		t.Errorf("own list: %d orders, %d keys outside the projection", len(list.Orders), extraKeys)
	}
	res["ownListOrders"] = len(list.Orders)
	res["keysOutsideMemberProjection"] = extraKeys

	// Other member's order: 404 with the same body as a missing order.
	foreign := int64(AccountB*1000 + 1)
	rf := m.must("GET", fmt.Sprintf("/member/orders/%d", foreign), MemberA, nil, nil)
	rm := m.must("GET", "/member/orders/99999999", MemberA, nil, nil)
	if rf.status != 404 || rm.status != 404 || string(rf.body) != string(rm.body) {
		t.Errorf("foreign order must look exactly like a missing one: %d %s / %d %s", rf.status, rf.body, rm.status, rm.body)
	}
	rb := m.must("GET", fmt.Sprintf("/member/orders/%d", foreign), MemberB, nil, nil)
	if rb.status != 200 {
		t.Errorf("owner cannot read its order: %d", rb.status)
	}
	rfl := m.must("GET", fmt.Sprintf("/member/orders?$filter=orderId%%20eq%%20%d", foreign), MemberA, nil, nil)
	var fl struct {
		Orders []map[string]any `json:"orders"`
	}
	rfl.json(t, &fl)
	if rfl.status != 200 || len(fl.Orders) != 0 {
		t.Errorf("filter on a foreign order id returned rows: %s", rfl.body)
	}
	res["foreignOrderStatus"] = rf.status
	res["foreignAndMissingBodiesEqual"] = string(rf.body) == string(rm.body)

	// Internal fields cannot be selected, filtered, ordered or counted.
	rejected := 0
	queries := []string{"$select=cost", "$select=orderId,staffMemo", "$filter=creditClass%20eq%20'" + SentinelCredit + "'",
		"$orderby=cost%20desc", "$filter=staffMemo%20ne%20null", "$count=true", "$select=order_id", "$apply=aggregate(cost)"}
	for _, q := range queries {
		rq := m.must("GET", "/member/orders?"+q, MemberA, nil, nil)
		if rq.status == 400 {
			rejected++
		} else {
			t.Errorf("%s: %d %s", q, rq.status, rq.body)
		}
	}
	res["internalQueriesRejected"] = fmt.Sprintf("%d/%d", rejected, len(queries))

	// Response difference against the staff projection of the same order.
	var staffView, memberView map[string]any
	s.must("GET", "/staff/orders/1001", StaffMain, nil, nil).json(t, &staffView)
	m.must("GET", "/member/orders/1001", MemberA, nil, nil).json(t, &memberView)
	var diff []string
	for k := range staffView {
		if _, ok := memberView[k]; !ok {
			diff = append(diff, k)
		}
	}
	slices.Sort(diff)
	want := slices.Clone(internal)
	slices.Sort(want)
	if !slices.Equal(diff, want) {
		t.Errorf("staff minus member fields = %v, want exactly the internal fields %v", diff, want)
	}
	res["staffMinusMemberFields"] = diff

	// Updates outside the accepted list are refused and change nothing.
	own := int64(AccountA*1000 + 1)
	before := h.orderRow(own)
	beforeForeign := h.orderRow(foreign)
	refused := 0
	bodies := []map[string]any{{"status": "confirmed"}, {"total": "1"}, {"cost": 0}, {"creditClass": "A"},
		{"deliveryNote": "x", "status": "confirmed"}, {"version": 9}, {"staffMemo": "x"}, {"customerId": AccountB}}
	for _, b := range bodies {
		rp := m.must("PATCH", fmt.Sprintf("/member/orders/%d", own), MemberA, b, map[string]string{"If-Match": "1"})
		if rp.status == 403 {
			refused++
		} else {
			t.Errorf("PATCH %v: %d %s", b, rp.status, rp.body)
		}
	}
	rp := m.must("PATCH", fmt.Sprintf("/member/orders/%d", foreign), MemberA, map[string]any{"deliveryNote": "x"}, map[string]string{"If-Match": "1"})
	if rp.status != 404 {
		t.Errorf("PATCH foreign: %d", rp.status)
	}
	rp = m.must("PATCH", fmt.Sprintf("/member/orders/%d", own), MemberA, map[string]any{"deliveryNote": "x"}, nil)
	if rp.status != 428 {
		t.Errorf("PATCH without If-Match: %d", rp.status)
	}
	unchanged := fmt.Sprint(h.orderRow(own)) == fmt.Sprint(before) && fmt.Sprint(h.orderRow(foreign)) == fmt.Sprint(beforeForeign)
	if !unchanged {
		t.Errorf("a refused update changed the DB")
	}
	res["forbiddenUpdatesRefused"] = fmt.Sprintf("%d/%d", refused, len(bodies))
	res["dbUnchangedAfterRefusals"] = unchanged

	// The accepted field changes only itself and the version.
	rp = m.must("PATCH", fmt.Sprintf("/member/orders/%d", own), MemberA, map[string]any{"deliveryNote": "置き配"}, map[string]string{"If-Match": "1"})
	after := h.orderRow(own)
	var changed []string
	for k, v := range after {
		if fmt.Sprint(before[k]) != fmt.Sprint(v) {
			changed = append(changed, k)
		}
	}
	slices.Sort(changed)
	if rp.status != 200 || !slices.Equal(changed, []string{"delivery_note", "version"}) {
		t.Errorf("accepted update: %d %s changed %v", rp.status, rp.body, changed)
	}
	if rc := m.must("PATCH", fmt.Sprintf("/member/orders/%d", own), MemberA, map[string]any{"deliveryNote": "y"}, map[string]string{"If-Match": "1"}); rc.status != 409 {
		t.Errorf("stale If-Match: %d", rc.status)
	}
	res["acceptedUpdateChangedColumns"] = changed

	// The DB itself refuses the member role internal columns.
	mc, err := pgxConnect(h, h.roleDSN(MemberRole))
	if err != nil {
		t.Fatal(err)
	}
	defer mc.Close(h.ctx)
	dbRefused := map[string]string{}
	for _, q := range []string{`SELECT cost FROM owdm.orders LIMIT 1`, `SELECT staff_memo FROM owdm.orders LIMIT 1`,
		`UPDATE owdm.orders SET status = 'confirmed' WHERE order_id = 1001`, `UPDATE owdm.orders SET total_amount = 1 WHERE order_id = 1001`} {
		_, err := mc.Exec(h.ctx, q)
		dbRefused[q] = sqlState(err)
		if sqlState(err) != "42501" {
			t.Errorf("member role was not refused by the DB: %s: %v", q, err)
		}
	}
	res["dbRefusedForMemberRole"] = dbRefused
	if h.leaks.Load() != 0 {
		t.Errorf("internal sentinel appeared in %d member responses", h.leaks.Load())
	}
	res["memberBodiesScanned"] = h.scanned.Load()
	res["memberBodiesWithInternalSentinel"] = h.leaks.Load()
	observed(t, "P-B2C-01", res)
	return res
}

func TestPostgresPB2C01MemberProjectionAndOwnership(t *testing.T) {
	h := newHarness(t)
	s := h.startUnit("staff", KindStaff, unitOptions{})
	m := h.startUnit("member", KindMember, unitOptions{})
	scenarioPB2C01(t, h, m, s)
}

// ---------------------------------------------------------------- through

func TestPostgresThroughStep6(t *testing.T) {
	h := newHarness(t)
	s := h.startUnit("staff", KindStaff, unitOptions{})
	n := h.startUnit("member", KindMember, unitOptions{})
	id := Identity{ReadPlan: h.member.Source, MemberProjection: h.member.Digest, StaffProjection: h.staff.Digest,
		MemberUnitConfig: n.ready.ConfigDigest, StaffUnitConfig: s.ready.ConfigDigest,
		MemberUnitProjection: n.ready.ProjectionDigest, StaffUnitProjection: s.ready.ProjectionDigest}
	if h.member.Source != h.staff.Source {
		t.Fatalf("projections from different read plans: %+v %+v", h.member.Source, h.staff.Source)
	}
	var chain Chain
	step := func(name string, f func() map[string]any) {
		r := f()
		chain.Append(name, id, !t.Failed(), r)
	}
	step("unit-admission", func() map[string]any {
		_, errM := h.tryStartUnit("member-foreign-projection", KindMember, unitOptions{expect: h.staff.Digest})
		_, errS := h.tryStartUnit("staff-foreign-projection", KindStaff, unitOptions{expect: h.member.Digest})
		for _, err := range []error{errM, errS} {
			if err == nil || !strings.Contains(err.Error(), "refusing to start") {
				t.Errorf("a unit started with a projection other than the fixed one: %v", err)
			}
		}
		if n.ready.DBRole != MemberRole || s.ready.DBRole != StaffRole || n.ready.PID == s.ready.PID {
			t.Errorf("units are not separate processes with their own roles: %+v %+v", n.ready, s.ready)
		}
		return map[string]any{"member": n.ready, "staff": s.ready, "foreignMemberRefused": errM != nil, "foreignStaffRefused": errS != nil}
	})
	step("P-B2C-01", func() map[string]any { return scenarioPB2C01(t, h, n, s) })
	step("P-OPS-02", func() map[string]any { return scenarioPOPS02(t, h, n, s) })
	step("P-DB-01", func() map[string]any { return scenarioPDB01(t, h, n, s) })
	step("P-B2C-02", func() map[string]any { return scenarioPB2C02(t, h, s) })
	step("member-response-scan", func() map[string]any {
		if h.leaks.Load() != 0 {
			t.Errorf("internal sentinels in %d member responses", h.leaks.Load())
		}
		return map[string]any{"scanned": h.scanned.Load(), "withInternalSentinel": h.leaks.Load()}
	})
	required := []string{"unit-admission", "P-B2C-01", "P-OPS-02", "P-DB-01", "P-B2C-02", "member-response-scan"}
	if err := chain.Verify(required); err != nil {
		t.Error(err)
	}
	// Replacing one identity in one record breaks the chain.
	broken := Chain{Records: slices.Clone(chain.Records)}
	broken.Records[2].Identity.MemberUnitConfig = "sha256:other"
	if broken.Verify(required) == nil {
		t.Error("a record with another unit identity still verified")
	}
	raw, _ := json.MarshalIndent(chain, "", "  ")
	if dir := os.Getenv(outDirEnv); dir != "" {
		_ = os.WriteFile(filepath.Join(dir, "chain.json"), raw, 0o644)
	}
	observed(t, "chain", map[string]any{"head": chain.Head(), "records": len(chain.Records), "identity": id})
}
