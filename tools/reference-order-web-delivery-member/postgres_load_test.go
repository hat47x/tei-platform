package orderwebmember

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------- P-B2C-02

// Load parameters. The surge is 48 member workers on 48 accounts; half obey
// Retry-After (one retry, with jitter), half ignore it and resend after 20 ms.
// The staff side confirms pending orders with 2 workers. Conditions are
// interleaved (no surge, protected surge, unprotected surge) in 5 rounds of
// 200 confirms each, so slow drift of the machine affects all three alike.
// Each condition is judged by the median over the rounds of the per-round
// p99: a rare stall of the whole machine (seen under WSL as single 200 ms
// outliers in any condition, including no surge) moves one round, not the
// verdict. The pooled percentiles are recorded as well.
const (
	surgeWorkers   = 48
	ignorePause    = 20 * time.Millisecond
	rounds         = 5
	confirmsBlock  = 200
	confirmWorkers = 2
	// memberQueryBurn makes each member read count 40,000 rows in the DB
	// (about 20 ms of one CPU on the WSL machine): a slow, CPU-consuming
	// downstream.
	memberQueryBurn = 40000
	// Relative targets (see README). The protected p99 of the staff confirm
	// must stay within 3x the no-surge p99 (denominator at least 5 ms). The
	// no-surge p99 itself varied between 11 and 40 ms across runs on WSL, so
	// a tighter ratio would measure the machine, not the bulkhead. The same
	// measurement must fail the target without the protection
	// (unprotected/no-surge > 3x), and the unprotected p99 must be at least
	// 2x the protected p99, so the test is shown to see the interference.
	protectedP99Ratio = 3.0
	baselineFloor     = 5 * time.Millisecond
	controlMinFactor  = 2.0
)

type surgeStats struct {
	ok, limited, other, attempts, logical, badRetryAfter atomic.Int64
	reasons                                              sync.Map
	elapsed                                              time.Duration
	done                                                 chan struct{}
}

func (s *surgeStats) wait() { <-s.done }

func (s *surgeStats) snapshot() map[string]any {
	reasons := map[string]int64{}
	s.reasons.Range(func(k, v any) bool { reasons[k.(string)] = v.(*atomic.Int64).Load(); return true })
	secs := s.elapsed.Seconds()
	return map[string]any{"ok": s.ok.Load(), "limited429": s.limited.Load(), "other": s.other.Load(), "attempts": s.attempts.Load(),
		"logicalRequests": s.logical.Load(), "badRetryAfter": s.badRetryAfter.Load(), "reasons": reasons,
		"seconds": secs, "okPerSecond": float64(s.ok.Load()) / secs, "attemptsPerSecond": float64(s.attempts.Load()) / secs}
}

func runSurge(u *unitProc, stop <-chan struct{}) *surgeStats {
	st := &surgeStats{done: make(chan struct{})}
	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < surgeWorkers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			principal := LoadMemberID(1 + i%LoadMembers)
			obey := i%2 == 0
			for {
				select {
				case <-stop:
					return
				default:
				}
				st.logical.Add(1)
				for attempt := 0; attempt < 2; attempt++ {
					st.attempts.Add(1)
					r, err := u.do("GET", "/member/orders?$top=20", principal, nil, nil)
					if err != nil {
						st.other.Add(1)
						break
					}
					if r.status == 200 {
						st.ok.Add(1)
						break
					}
					if r.status != 429 {
						st.other.Add(1)
						break
					}
					st.limited.Add(1)
					var b struct {
						Reason string `json:"reason"`
					}
					_ = json.Unmarshal(r.body, &b)
					c, _ := st.reasons.LoadOrStore(b.Reason, &atomic.Int64{})
					c.(*atomic.Int64).Add(1)
					ra, err := strconv.Atoi(r.header.Get("Retry-After"))
					if err != nil || ra < 1 {
						st.badRetryAfter.Add(1)
						ra = 1
					}
					if !obey {
						time.Sleep(ignorePause)
						break
					}
					if attempt == 1 {
						break // one retry layer only
					}
					wait := time.Duration(float64(ra) * float64(time.Second) * (1 + rand.Float64()/2))
					select {
					case <-stop:
						return
					case <-time.After(wait):
					}
				}
			}
		}(i)
	}
	go func() {
		wg.Wait()
		st.elapsed = time.Since(start)
		close(st.done)
	}()
	return st
}

func confirmBlock(s *unitProc, ids []int64) ([]time.Duration, int) {
	var mu sync.Mutex
	var lat []time.Duration
	failures := 0
	next := atomic.Int64{}
	var wg sync.WaitGroup
	for w := 0; w < confirmWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := next.Add(1) - 1
				if int(i) >= len(ids) {
					return
				}
				start := time.Now()
				r, err := s.do("POST", fmt.Sprintf("/staff/orders/%d/confirm", ids[i]), StaffMain, map[string]any{"expectedVersion": 1}, nil)
				d := time.Since(start)
				mu.Lock()
				if err != nil || r.status != 200 {
					failures++
				} else {
					lat = append(lat, d)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return lat, failures
}

type connSample struct {
	MaxByRole       map[string]int `json:"maxConnectionsByRole"`
	MaxActiveByRole map[string]int `json:"maxActiveByRole"`
	// MaxSlowByRole counts active member order reads (the injected slow
	// downstream); they are the requests the concurrency slots bound.
	MaxSlowByRole map[string]int `json:"maxActiveSlowReadsByRole"`
	Samples       int            `json:"samples"`
}

func (cs *connSample) sample(h *harness, stop <-chan struct{}) func() {
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := pgxConnect(h, h.adminDSN)
		if err != nil {
			return
		}
		defer conn.Close(h.ctx)
		for {
			rows, err := conn.Query(h.ctx, `SELECT usename, count(*), count(*) FILTER (WHERE state = 'active'),
				count(*) FILTER (WHERE state = 'active' AND query LIKE '%generate_series%' AND query NOT LIKE '%pg_stat_activity%')
				FROM pg_stat_activity WHERE datname = current_database() AND usename IS NOT NULL AND pid <> pg_backend_pid() GROUP BY usename`)
			if err == nil {
				for rows.Next() {
					var u string
					var all, active, slow int
					if rows.Scan(&u, &all, &active, &slow) == nil {
						cs.MaxSlowByRole[u] = max(cs.MaxSlowByRole[u], slow)
						cs.MaxByRole[u] = max(cs.MaxByRole[u], all)
						cs.MaxActiveByRole[u] = max(cs.MaxActiveByRole[u], active)
					}
				}
				rows.Close()
				cs.Samples++
			}
			select {
			case <-stop:
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	}()
	return func() { <-done }
}

func latencySummary(lat []time.Duration) map[string]float64 {
	return map[string]float64{"n": float64(len(lat)), "p50Ms": ms(percentile(lat, 0.5)), "p95Ms": ms(percentile(lat, 0.95)),
		"p99Ms": ms(percentile(lat, 0.99)), "maxMs": ms(percentile(lat, 1))}
}

type loadCondition struct {
	lat      []time.Duration
	roundP99 []float64
	fail     int
	surges   []map[string]any
	conns    *connSample
	idleWait []float64
}

func scenarioPB2C02(t *testing.T, h *harness, s *unitProc) map[string]any {
	cfg := DefaultConfig(KindMember).Limits
	res := map[string]any{"surgeWorkers": surgeWorkers, "ignorePauseMs": ms(ignorePause), "rounds": rounds, "confirmsPerBlock": confirmsBlock,
		"confirmWorkers": confirmWorkers, "memberQueryBurn": memberQueryBurn, "protectedP99Ratio": protectedP99Ratio,
		"baselineFloorMs": ms(baselineFloor), "controlMinFactor": controlMinFactor, "memberLimits": cfg}
	burn := []string{"-member-query-burn", strconv.Itoa(memberQueryBurn)}
	p := h.startUnit("member-protected", KindMember, unitOptions{args: burn})
	defer p.stop()
	// Negative control: no quota, a pool of 48, and the admin role (no
	// role connection limit): the bulkhead is absent.
	u := h.startUnit("member-unprotected", KindMember, unitOptions{dsn: h.adminDSN,
		args: append([]string{"-unprotected", "-pool-max", "48"}, burn...)})
	defer u.stop()
	next := int64(9000001)
	ids := func() []int64 {
		out := make([]int64, confirmsBlock)
		for i := range out {
			out[i] = next
			next++
		}
		return out
	}
	conds := map[string]*loadCondition{"noSurge": {conns: &connSample{MaxByRole: map[string]int{}, MaxActiveByRole: map[string]int{}, MaxSlowByRole: map[string]int{}}},
		"protected":   {conns: &connSample{MaxByRole: map[string]int{}, MaxActiveByRole: map[string]int{}, MaxSlowByRole: map[string]int{}}},
		"unprotected": {conns: &connSample{MaxByRole: map[string]int{}, MaxActiveByRole: map[string]int{}, MaxSlowByRole: map[string]int{}}}}
	confirmBlock(s, ids()[:20]) // warm the staff path
	block := func(name string, target *unitProc) {
		c := conds[name]
		var st *surgeStats
		stop := make(chan struct{})
		if target != nil {
			st = runSurge(target, stop)
			// The surge is running once the unit answers it (observed).
			for st.limited.Load()+st.ok.Load() < 200 {
				time.Sleep(10 * time.Millisecond)
			}
		}
		connStop := make(chan struct{})
		connWait := c.conns.sample(h, connStop)
		lat, fail := confirmBlock(s, ids())
		close(connStop)
		close(stop)
		connWait()
		if st != nil {
			st.wait()
			c.surges = append(c.surges, st.snapshot())
		}
		c.lat = append(c.lat, lat...)
		c.roundP99 = append(c.roundP99, ms(percentile(lat, 0.99)))
		c.fail += fail
		// Queries of a stopped surge keep running in the DB; wait until the
		// DB has no other active client query before the next block.
		c.idleWait = append(c.idleWait, h.waitDBIdle())
	}
	for r := 0; r < rounds; r++ {
		block("noSurge", nil)
		block("protected", p)
		block("unprotected", u)
	}
	base := median(conds["noSurge"].roundP99)
	prot := median(conds["protected"].roundP99)
	unprot := median(conds["unprotected"].roundP99)
	ref := max(base, ms(baselineFloor))
	protRatio := prot / ref
	unprotRatio := unprot / ref
	controlFactor := unprot / prot
	for name, c := range conds {
		res[name] = map[string]any{"staffPooled": latencySummary(c.lat), "staffRoundP99Ms": c.roundP99, "staffMedianRoundP99Ms": median(c.roundP99),
			"staffFailures": c.fail, "member": c.surges, "connections": c.conns, "idleWaitMs": c.idleWait}
	}
	res["referenceP99Ms"] = ref
	res["protectedP99RatioObserved"] = protRatio
	res["unprotectedP99RatioObserved"] = unprotRatio
	res["unprotectedOverProtected"] = controlFactor
	for name, c := range conds {
		if c.fail != 0 {
			t.Errorf("%s: %d staff confirms failed", name, c.fail)
		}
	}
	if protRatio > protectedP99Ratio {
		t.Errorf("protected staff p99 %.1fms is %.2fx the no-surge p99 %.1fms (target %.1fx)", prot, protRatio, ref, protectedP99Ratio)
	}
	if unprotRatio <= protectedP99Ratio || controlFactor < controlMinFactor {
		t.Errorf("negative control p99 %.1fms (%.2fx no-surge, %.2fx protected) does not show the interference", unprot, unprotRatio, controlFactor)
	}
	var ok, limited int64
	for _, sm := range conds["protected"].surges {
		ok += sm["ok"].(int64)
		limited += sm["limited429"].(int64)
		if sm["other"].(int64) != 0 || sm["badRetryAfter"].(int64) != 0 {
			t.Errorf("member side answered other than 200/429 with Retry-After: %v", sm)
		}
		if rate := sm["okPerSecond"].(float64); rate > cfg.TierRate+cfg.TierBurst/sm["seconds"].(float64)+1 {
			t.Errorf("admitted member rate %.1f/s exceeds the tier rate %.0f/s", rate, cfg.TierRate)
		}
	}
	if ok == 0 || limited == 0 {
		t.Errorf("member side did not degrade gradually (ok %d, 429 %d)", ok, limited)
	}
	pc := conds["protected"].conns
	if pc.MaxByRole[MemberRole] > h.limits.MemberConnections || pc.MaxSlowByRole[MemberRole] > cfg.TierConcurrency || pc.Samples == 0 || pc.MaxByRole[StaffRole] == 0 {
		t.Errorf("member role connections exceeded the bulkhead or were not observed: %+v", pc)
	}
	var limit int
	_ = h.admin.QueryRow(h.ctx, `SELECT rolconnlimit FROM pg_roles WHERE rolname = $1`, MemberRole).Scan(&limit)
	res["memberRoleConnectionLimit"] = limit
	res["protectedUnitStats"] = p.stats()
	observed(t, "P-B2C-02", res)
	return res
}

func TestPostgresPB2C02SurgeIsolation(t *testing.T) {
	h := newHarness(t)
	s := h.startUnit("staff", KindStaff, unitOptions{})
	scenarioPB2C02(t, h, s)
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	if len(s)%2 == 1 {
		return s[len(s)/2]
	}
	return (s[len(s)/2-1] + s[len(s)/2]) / 2
}
