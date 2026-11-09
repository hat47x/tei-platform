package orderwebmember

import (
	"math"
	"sync"
	"time"
)

// LimitConfig is the member tier quota. Rates are tokens per second.
type LimitConfig struct {
	PerPrincipalRate  float64
	PerPrincipalBurst float64
	TierRate          float64
	TierBurst         float64
	// TierConcurrency bounds member requests that hold a DB connection at the
	// same time. A request that cannot get a slot within AdmitWait is shed.
	TierConcurrency int
	AdmitWait       time.Duration
}

type bucket struct {
	tokens float64
	last   time.Time
}

func (b *bucket) take(now time.Time, rate, burst float64) (bool, time.Duration) {
	if b.last.IsZero() {
		b.tokens, b.last = burst, now
	}
	b.tokens = math.Min(burst, b.tokens+now.Sub(b.last).Seconds()*rate)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) / rate * float64(time.Second))
}

func (b *bucket) refund(burst float64) { b.tokens = math.Min(burst, b.tokens+1) }

// Limiter applies, in order, the tier concurrency slot, the per-principal
// bucket and the tier bucket. It never queues beyond AdmitWait: an overloaded
// member tier answers 429 instead of waiting, so it cannot hold more DB
// connections than its slots.
type Limiter struct {
	cfg   LimitConfig
	now   func() time.Time
	slots chan struct{}
	mu    sync.Mutex
	per   map[string]*bucket
	tier  bucket
}

func NewLimiter(cfg LimitConfig) *Limiter {
	if cfg.TierConcurrency < 1 {
		cfg.TierConcurrency = 1
	}
	return &Limiter{cfg: cfg, now: time.Now, slots: make(chan struct{}, cfg.TierConcurrency), per: map[string]*bucket{}}
}

// Decision is the result of Admit. RetryAfter is in whole seconds (at least
// 1) for the Retry-After header when Admitted is false.
type Decision struct {
	Admitted   bool
	Reason     string // admitted | principal-rate | tier-rate | tier-concurrency
	RetryAfter int
	release    func()
}

// Release returns the concurrency slot of an admitted request.
func (d Decision) Release() {
	if d.release != nil {
		d.release()
	}
}

func retrySeconds(d time.Duration) int {
	s := int(math.Ceil(d.Seconds()))
	if s < 1 {
		s = 1
	}
	return s
}

func (l *Limiter) Admit(principal string) Decision {
	l.mu.Lock()
	now := l.now()
	b := l.per[principal]
	if b == nil {
		b = &bucket{}
		l.per[principal] = b
	}
	if ok, wait := b.take(now, l.cfg.PerPrincipalRate, l.cfg.PerPrincipalBurst); !ok {
		l.mu.Unlock()
		return Decision{Reason: "principal-rate", RetryAfter: retrySeconds(wait)}
	}
	if ok, wait := l.tier.take(now, l.cfg.TierRate, l.cfg.TierBurst); !ok {
		b.refund(l.cfg.PerPrincipalBurst)
		l.mu.Unlock()
		return Decision{Reason: "tier-rate", RetryAfter: retrySeconds(wait)}
	}
	l.mu.Unlock()
	timer := time.NewTimer(l.cfg.AdmitWait)
	defer timer.Stop()
	select {
	case l.slots <- struct{}{}:
		return Decision{Admitted: true, Reason: "admitted", release: func() { <-l.slots }}
	case <-timer.C:
		// A shed request did not use the quota; give the tokens back so the
		// admitted rate is bounded by the slots, not eaten by shed requests.
		l.mu.Lock()
		b.refund(l.cfg.PerPrincipalBurst)
		l.tier.refund(l.cfg.TierBurst)
		l.mu.Unlock()
		return Decision{Reason: "tier-concurrency", RetryAfter: 1}
	}
}
