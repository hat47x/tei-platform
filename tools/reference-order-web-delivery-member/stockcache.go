package orderwebmember

import (
	"context"
	"sync"
	"time"
)

// StockValue is one read of the canonical item row. AsOf is the DB clock of
// the statement that read it; Version is the item version of that read.
type StockValue struct {
	ItemCode string    `json:"itemCode"`
	Stock    int64     `json:"stock"`
	Version  int64     `json:"version"`
	AsOf     time.Time `json:"asOf"`
}

type stockEntry struct {
	value StockValue
	// readStart is the local monotonic time taken before the canonical read
	// was issued. Age is measured from it, never from the time the entry was
	// stored, so a delayed fill cannot make an old value look young.
	readStart time.Time
}

// StockCache is the display cache of the member unit. It is never used to
// decide anything: reorder decisions read the canonical row.
//
// Declared freshness: a value served from the cache was read from the
// canonical row no longer than MaxAge before it is served. Invalidations
// (version high-water marks) delivered by NOTIFY also stop an older value
// from being stored after a newer version is known (the delayed-fill race).
type StockCache struct {
	MaxAge time.Duration
	// UnsafeStampAtSet is the negative control: the age is measured from the
	// time the entry is stored, and invalidation marks are ignored.
	UnsafeStampAtSet bool
	// FillHook runs between the canonical read and the store (fault
	// injection for the reordered update).
	FillHook func(ctx context.Context, v StockValue)
	now      func() time.Time

	mu      sync.Mutex
	entries map[string]stockEntry
	marks   map[string]int64
	stats   CacheStats
}

// CacheStats counts what the cache did.
type CacheStats struct {
	Hits             int64 `json:"hits"`
	Loads            int64 `json:"loads"`
	StoreRefusedOld  int64 `json:"storeRefusedOlderThanInvalidation"`
	StoreRefusedAged int64 `json:"storeRefusedAlreadyOlderThanMaxAge"`
	Invalidations    int64 `json:"invalidations"`
}

func NewStockCache(maxAge time.Duration) *StockCache {
	return &StockCache{MaxAge: maxAge, now: time.Now, entries: map[string]stockEntry{}, marks: map[string]int64{}}
}

// Get returns the value and whether it came from the cache.
func (c *StockCache) Get(ctx context.Context, item string, load func(context.Context) (StockValue, error)) (StockValue, bool, error) {
	c.mu.Lock()
	if e, ok := c.entries[item]; ok {
		fresh := c.now().Sub(e.readStart) <= c.MaxAge
		if fresh && (c.UnsafeStampAtSet || e.value.Version >= c.marks[item]) {
			c.stats.Hits++
			c.mu.Unlock()
			return e.value, true, nil
		}
		delete(c.entries, item)
	}
	c.stats.Loads++
	c.mu.Unlock()
	readStart := c.now()
	v, err := load(ctx)
	if err != nil {
		return StockValue{}, false, err
	}
	if c.FillHook != nil {
		c.FillHook(ctx, v)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.UnsafeStampAtSet {
		c.entries[item] = stockEntry{value: v, readStart: c.now()}
		return v, false, nil
	}
	switch {
	case v.Version < c.marks[item]:
		c.stats.StoreRefusedOld++
	case c.now().Sub(readStart) > c.MaxAge:
		c.stats.StoreRefusedAged++
	default:
		if cur, ok := c.entries[item]; !ok || cur.value.Version <= v.Version {
			c.entries[item] = stockEntry{value: v, readStart: readStart}
		}
	}
	return v, false, nil
}

// Invalidate records that version is now current for item.
func (c *StockCache) Invalidate(item string, version int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stats.Invalidations++
	if version > c.marks[item] {
		c.marks[item] = version
	}
	if e, ok := c.entries[item]; ok && e.value.Version < version {
		delete(c.entries, item)
	}
}

// Forget drops an entry (after this unit's own write).
func (c *StockCache) Forget(item string) {
	c.mu.Lock()
	delete(c.entries, item)
	c.mu.Unlock()
}

func (c *StockCache) Stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stats
}
