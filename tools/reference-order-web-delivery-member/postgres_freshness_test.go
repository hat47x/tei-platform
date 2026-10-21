package orderwebmember

// Time in these tests is measured on the monotonic clock of the test process.
// The units age their caches on the same kernel's monotonic clock, so the
// declared bounds and their measurement share one clock domain. The DB clock
// is a wall clock: under WSL2 it was observed to jump forward by about five
// seconds within one test, which made a correct unit look late. The tests
// record the drift between the two clocks (clockDriftMs) instead of measuring
// with the DB clock.
//
// An origin is taken when the response of the write that the bound is about
// (revoke, scope removal, stock adjustment) has arrived, which is after its
// commit. Requests that start at origin+bound or later must see the write.

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------- P-OPS-02

type pollRecord struct {
	Unit    string  `json:"unit"`
	StartMS float64 `json:"startMs"` // request start, ms after the origin
	EndMS   float64 `json:"endMs"`
	Status  int     `json:"status"`
}

// pollWindow polls each unit (one request per unit every 250 ms, below the
// per-member rate) until the monotonic clock has passed origin+until. The
// pacing only keeps the rate; the loop ends on the observed elapsed time.
func pollWindow(h *harness, units []*unitProc, path, principal string, origin time.Time, until time.Duration, onResp func(u *unitProc, rec pollRecord, r resp)) {
	for {
		for _, u := range units {
			start := time.Since(origin)
			// A 429 is answered after authentication, so it counts as accepted.
			r, err := u.do("GET", path, principal, nil, nil)
			if err != nil {
				h.t.Fatal(err)
			}
			onResp(u, pollRecord{Unit: u.name, StartMS: ms(start), EndMS: ms(time.Since(origin)), Status: r.status}, r)
		}
		if time.Since(origin) > until {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

type revocationOutcome struct {
	AcceptedAfterOrigin int          `json:"acceptedAfterOrigin"`
	LastAcceptedStart   float64      `json:"lastAcceptedStartMs"`
	FirstRefusedEnd     float64      `json:"firstRefusedEndMs"`
	AcceptedAfterBound  int          `json:"acceptedAfterBound"`
	Polls               []pollRecord `json:"polls"`
}

type shrinkOutcome struct {
	NewRowSeen         int `json:"newRowSeen"`
	RemovedSeenInBound int `json:"removedAccountRowsSeenWithinBound"`
	RemovedSeenAfter   int `json:"removedAccountRowsSeenAfterBound"`
	Polls              int `json:"polls"`
}

func scenarioPOPS02(t *testing.T, h *harness, n, s *unitProc) map[string]any {
	bound := DefaultConfig(KindMember).AuthzMaxAge
	boundMS := float64(bound.Milliseconds())
	v := h.startUnit("member-drop-notify", KindMember, unitOptions{args: []string{"-drop-notify", "-pool-max", "2"}})
	defer v.stop()
	res := map[string]any{"declaredBoundMs": bound.Milliseconds(), "clockDomain": "monotonic-test-process"}

	// 1. Revoke a member at the staff unit; query it at two member units.
	for _, u := range []*unitProc{n, v} {
		if r := u.must("GET", "/member/orders?$top=1", MemberRevoked, nil, nil); r.status != 200 {
			t.Fatalf("warm %s: %d", u.name, r.status)
		}
	}
	if r := s.must("POST", "/staff/principals/"+MemberRevoked+"/revoke", StaffMain, nil, nil); r.status != 200 {
		t.Fatalf("revoke: %d %s", r.status, r.body)
	}
	origin := time.Now()
	rev := map[string]*revocationOutcome{n.name: {FirstRefusedEnd: -1}, v.name: {FirstRefusedEnd: -1}}
	pollWindow(h, []*unitProc{n, v}, "/member/orders?$top=1", MemberRevoked, origin, bound+500*time.Millisecond,
		func(u *unitProc, rec pollRecord, r resp) {
			x := rev[u.name]
			x.Polls = append(x.Polls, rec)
			if r.status == 401 {
				if x.FirstRefusedEnd < 0 {
					x.FirstRefusedEnd = rec.EndMS
				}
				return
			}
			x.AcceptedAfterOrigin++
			x.LastAcceptedStart = rec.StartMS
			if rec.StartMS >= boundMS {
				x.AcceptedAfterBound++
			}
		})
	for name, x := range rev {
		if x.AcceptedAfterBound != 0 || x.FirstRefusedEnd < 0 {
			t.Errorf("%s: revocation not effective within %v: %+v", name, bound, x)
		}
	}
	if rev[v.name].AcceptedAfterOrigin == 0 {
		t.Errorf("drop-notify unit never served the cached grant after the revocation; the bound was not exercised")
	}
	res["revocation"] = rev

	// 2. Scope reduction and an added row, in both orders of evaluation.
	shrink := func(principal string, removed int64, addFirst bool, units []*unitProc) (map[string]any, map[string]*shrinkOutcome) {
		for _, u := range units {
			r := u.must("GET", "/member/orders?$top=100", principal, nil, nil)
			if r.status != 200 || !strings.Contains(string(r.body), fmt.Sprintf(`"customerId":%d`, removed)) {
				t.Fatalf("warm %s for %s: %d", u.name, principal, r.status)
			}
		}
		var newID int64
		add := func() {
			var out struct {
				OrderID int64 `json:"orderId"`
			}
			s.must("POST", "/staff/orders", StaffMain, map[string]any{"accountId": removed, "itemCode": ItemMain, "quantity": 1, "total": "1234"}, nil).json(t, &out)
			newID = out.OrderID
		}
		remove := func() {
			if r := s.must("POST", "/staff/principals/"+principal+"/scopes/remove", StaffMain, map[string]any{"accountId": removed}, nil); r.status != 200 {
				t.Fatalf("remove scope: %d %s", r.status, r.body)
			}
		}
		if addFirst {
			add()
			remove()
		} else {
			remove()
			add()
		}
		origin := time.Now()
		out := map[string]*shrinkOutcome{}
		for _, u := range units {
			out[u.name] = &shrinkOutcome{}
		}
		pollWindow(h, units, "/member/orders?$top=100", principal, origin, bound+500*time.Millisecond,
			func(u *unitProc, rec pollRecord, r resp) {
				x := out[u.name]
				x.Polls++
				if r.status != 200 {
					t.Errorf("%s: %d", u.name, r.status)
					return
				}
				var l struct {
					Orders []map[string]any `json:"orders"`
				}
				r.json(t, &l)
				removedSeen := false
				for _, o := range l.Orders {
					if int64(o["orderId"].(float64)) == newID {
						x.NewRowSeen++
					}
					if int64(o["customerId"].(float64)) == removed {
						removedSeen = true
					}
				}
				if removedSeen {
					if rec.StartMS >= boundMS {
						x.RemovedSeenAfter++
					} else {
						x.RemovedSeenInBound++
					}
				}
			})
		return map[string]any{"principal": principal, "removedAccount": removed, "addedOrder": newID, "addBeforeShrink": addFirst, "units": out}, out
	}
	verify := func(principal string, x *shrinkOutcome, unit string, wantStaleWindow, wantLeak bool) {
		if x.RemovedSeenAfter != 0 {
			t.Errorf("%s %s: removed account still visible after the bound: %+v", principal, unit, x)
		}
		if wantLeak != (x.NewRowSeen > 0) {
			t.Errorf("%s %s: row added around the reduction seen %d times (expected leak=%v)", principal, unit, x.NewRowSeen, wantLeak)
		}
		if wantStaleWindow && x.RemovedSeenInBound == 0 {
			t.Errorf("%s %s: the stale grant was never used, the case was not exercised", principal, unit)
		}
	}
	a, ao := shrink(MemberCorp, AccountCorp2, false, []*unitProc{n, v})
	b, bo := shrink(MemberCorp2, 7, true, []*unitProc{n, v})
	v.stop()
	neg := h.startUnit("member-unsafe-snapshot", KindMember, unitOptions{args: []string{"-drop-notify", "-unsafe-ignore-authz-snapshot", "-pool-max", "2"}})
	defer neg.stop()
	c, co := shrink(MemberCorp3, 9, false, []*unitProc{neg})
	verify(MemberCorp, ao[n.name], n.name, false, false)
	verify(MemberCorp, ao[v.name], v.name, true, false)
	verify(MemberCorp2, bo[n.name], n.name, false, false)
	verify(MemberCorp2, bo[v.name], v.name, true, false)
	verify(MemberCorp3, co[neg.name], neg.name, true, true)
	res["shrinkThenAdd"] = a
	res["addThenShrink"] = b
	res["negativeControlIgnoreSnapshot"] = c
	res["notifyUnitAuthz"] = n.stats()["authz"]
	res["clockDriftMs"] = h.clockDrift()
	observed(t, "P-OPS-02", res)
	return res
}

func TestPostgresPOPS02RevocationAcrossUnits(t *testing.T) {
	h := newHarness(t)
	s := h.startUnit("staff", KindStaff, unitOptions{})
	m := h.startUnit("member", KindMember, unitOptions{})
	scenarioPOPS02(t, h, m, s)
}

// ---------------------------------------------------------------- P-DB-01

// versionTimeline records, per item, when each version was known to be
// committed (the arrival of the writer's response, which is after the commit).
type versionTimeline struct {
	mu     sync.Mutex
	points map[string][]versionPoint
}

type versionPoint struct {
	version int64
	at      time.Time
}

func (vt *versionTimeline) add(item string, version int64, at time.Time) {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	vt.points[item] = append(vt.points[item], versionPoint{version, at})
}

// required is the highest version known committed at or before t.
func (vt *versionTimeline) required(item string, t time.Time) int64 {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	var v int64
	for _, p := range vt.points[item] {
		if !p.at.After(t) && p.version > v {
			v = p.version
		}
	}
	return v
}

type staleCheck struct {
	Responses      int   `json:"responses"`
	FromCache      int   `json:"fromCache"`
	Violations     int   `json:"violations"`
	MaxLagVersions int64 `json:"maxVersionsBehindRequired"`
	MaxVersionLag  int64 `json:"maxVersionsBehindLatestKnown"`
}

// checkDisplay: a display served for a request that started at start must be
// at least the version that was committed before start-bound.
func checkDisplay(t *testing.T, vt *versionTimeline, sc *staleCheck, item string, start time.Time, r resp, bound time.Duration) StockDisplay {
	var d StockDisplay
	r.json(t, &d)
	sc.Responses++
	if d.Source == "cache" {
		sc.FromCache++
	}
	if need := vt.required(item, start.Add(-bound)); d.Version < need {
		sc.Violations++
		sc.MaxLagVersions = max(sc.MaxLagVersions, need-d.Version)
	}
	sc.MaxVersionLag = max(sc.MaxVersionLag, vt.required(item, start)-d.Version)
	return d
}

func (h *harness) adjust(s *unitProc, vt *versionTimeline, item string, delta int64) {
	var out struct {
		Version int64 `json:"version"`
	}
	r := s.must("POST", "/staff/items/"+item+"/adjust", StaffMain, map[string]any{"delta": delta}, nil)
	if r.status != 200 {
		h.t.Fatalf("adjust: %d %s", r.status, r.body)
	}
	r.json(h.t, &out)
	vt.add(item, out.Version, time.Now())
}

func scenarioPDB01(t *testing.T, h *harness, n, s *unitProc) map[string]any {
	bound := DefaultConfig(KindMember).StockMaxAge
	res := map[string]any{"declaredBoundMs": bound.Milliseconds(), "clockDomain": "monotonic-test-process"}
	vt := &versionTimeline{points: map[string][]versionPoint{}}
	var cur int64
	_ = h.admin.QueryRow(h.ctx, `SELECT version FROM owdm.item WHERE item_code = $1`, ItemMain).Scan(&cur)
	vt.add(ItemMain, cur, time.Now())
	variants := []struct {
		name   string
		args   []string
		unsafe bool
	}{
		{"reorder-drop-notify", []string{"-inject-stock-fill-reorder", "-drop-notify", "-pool-max", "2"}, false},
		{"reorder-notify", []string{"-inject-stock-fill-reorder", "-pool-max", "2"}, false},
		{"reorder-unsafe-stamp-at-set", []string{"-inject-stock-fill-reorder", "-drop-notify", "-unsafe-stamp-at-set", "-pool-max", "2"}, true},
	}
	for _, variant := range variants {
		d := h.startUnit("member-"+variant.name, KindMember, unitOptions{args: variant.args})
		sc := &staleCheck{}
		type r1Result struct {
			r   resp
			err error
		}
		done := make(chan r1Result, 1)
		r1Start := time.Now()
		go func() {
			r, err := d.do("GET", "/member/items/"+ItemMain+"/stock", MemberA, nil, nil)
			done <- r1Result{r, err}
		}()
		// Move the canonical row on until the delayed fill returns; the unit
		// holds the fill until a newer version has been current for longer
		// than the bound (observed by the unit), so one or two adjustments
		// are enough whatever the interleaving.
		adjusts := 0
		var r1 r1Result
	wait:
		for {
			h.adjust(s, vt, ItemMain, -1)
			adjusts++
			deadline := time.Now().Add(bound + time.Second)
			for time.Now().Before(deadline) {
				select {
				case r1 = <-done:
					break wait
				case <-time.After(20 * time.Millisecond):
				}
			}
			if adjusts > 5 {
				t.Fatalf("%s: the delayed fill did not complete", variant.name)
			}
		}
		if r1.err != nil || r1.r.status != 200 {
			t.Fatalf("%s: delayed read: %v %d", variant.name, r1.err, r1.r.status)
		}
		first := checkDisplay(t, vt, sc, ItemMain, r1Start, r1.r, bound)
		r1Done := time.Now()
		for i := 0; time.Since(r1Done) <= bound+300*time.Millisecond; i++ {
			start := time.Now()
			r := d.must("GET", "/member/items/"+ItemMain+"/stock", LoadMemberID(1+i%10), nil, nil)
			if r.status != 200 {
				t.Fatalf("%s: display %d", variant.name, r.status)
			}
			checkDisplay(t, vt, sc, ItemMain, start, r, bound)
			time.Sleep(50 * time.Millisecond)
		}
		st := d.stats()["stockCache"]
		d.stop()
		if variant.unsafe && sc.Violations == 0 {
			t.Errorf("%s: negative control did not exceed the bound; the injection is not effective: %+v", variant.name, sc)
		}
		if !variant.unsafe && sc.Violations != 0 {
			t.Errorf("%s: served a value older than the declared bound: %+v", variant.name, sc)
		}
		res[variant.name] = map[string]any{"display": sc, "adjusts": adjusts, "delayedReadVersion": first.Version,
			"delayedReadSource": first.Source, "cache": st}
	}

	// Decisions read the canonical row: 20 concurrent reorders of one unit of
	// ITEM-HOT (stock 10), each sent with the version shown on the display.
	var disp, disp2 StockDisplay
	_ = h.admin.QueryRow(h.ctx, `SELECT version FROM owdm.item WHERE item_code = $1`, ItemHot).Scan(&cur)
	vt.add(ItemHot, cur, time.Now())
	n.must("GET", "/member/items/"+ItemHot+"/stock", ReorderMemberID(1), nil, nil).json(t, &disp)
	n.must("GET", "/member/items/"+ItemHot+"/stock", ReorderMemberID(2), nil, nil).json(t, &disp2)
	var mu sync.Mutex
	statuses := map[int]int{}
	reuse := map[string]int{}
	decidedOn := map[string]int{}
	var wg sync.WaitGroup
	for i := 1; i <= ReorderMembers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := n.must("POST", fmt.Sprintf("/member/orders/%d/reorder", (9+i)*1000+1), ReorderMemberID(i),
				map[string]any{"displayedStockVersion": disp.Version}, nil)
			var rr ReorderResult
			_ = json.Unmarshal(r.body, &rr)
			if r.status == 201 {
				vt.add(ItemHot, rr.StockVersion, time.Now())
			}
			mu.Lock()
			statuses[r.status]++
			reuse[rr.DisplayReuse]++
			decidedOn[rr.DecidedOn]++
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	var stock, created int64
	_ = h.admin.QueryRow(h.ctx, `SELECT stock FROM owdm.item WHERE item_code = $1`, ItemHot).Scan(&stock)
	_ = h.admin.QueryRow(h.ctx, `SELECT count(*) FROM owdm.orders WHERE item_code = $1 AND order_id >= 10000000`, ItemHot).Scan(&created)
	if statuses[201] != 10 || statuses[409] != 10 || len(statuses) != 2 || stock != 0 || created != 10 || decidedOn["canonical"] != ReorderMembers {
		t.Errorf("reorder decisions: statuses %v stock %d created %d decidedOn %v", statuses, stock, created, decidedOn)
	}
	sc := &staleCheck{}
	start := time.Now()
	after := checkDisplay(t, vt, sc, ItemHot, start, n.must("GET", "/member/items/"+ItemHot+"/stock", ReorderMemberID(3), nil, nil), bound)
	if sc.Violations != 0 {
		t.Errorf("display after reorders exceeds the bound: %+v", sc)
	}
	res["reorder"] = map[string]any{"initialStock": disp.Stock, "displayedVersion": disp.Version, "secondDisplaySource": disp2.Source,
		"statuses": statusKeys(statuses), "displayReuse": reuse, "decidedOn": decidedOn, "finalStock": stock, "createdOrders": created,
		"overAllocated": created - disp.Stock, "displayAfter": map[string]any{"check": sc, "source": after.Source, "stock": after.Stock, "asOf": after.AsOf}}
	res["clockDriftMs"] = h.clockDrift()
	res["retriedAfter429"] = h.retried.Load()
	observed(t, "P-DB-01", res)
	return res
}

func statusKeys(m map[int]int) map[string]int {
	out := map[string]int{}
	for k, v := range m {
		out[strconv.Itoa(k)] = v
	}
	return out
}

func TestPostgresPDB01StockCacheReorderedUpdate(t *testing.T) {
	h := newHarness(t)
	s := h.startUnit("staff", KindStaff, unitOptions{})
	m := h.startUnit("member", KindMember, unitOptions{})
	scenarioPDB01(t, h, m, s)
}
