package orderwebmember

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config is the configuration of one deployment unit (one OS process).
type Config struct {
	Unit      string `json:"unit"` // member | staff
	DSN       string `json:"-"`
	Listen    string `json:"-"`
	ReadyFile string `json:"-"`
	Workspace string `json:"-"`

	PoolMax     int           `json:"poolMax"`
	AuthzMaxAge time.Duration `json:"authzMaxAge"`
	StockMaxAge time.Duration `json:"stockMaxAge"`
	Limits      LimitConfig   `json:"limits"`
	// MemberQueryBurn is fault injection: every member order read also makes
	// the DB count this many rows of generate_series (a CPU-consuming slow
	// downstream, unlike pg_sleep which only holds a connection).
	MemberQueryBurn int `json:"memberQueryBurn"`

	// Fault injection and negative controls.
	DropNotify                bool `json:"dropNotify"`
	InjectStockFillReorder    bool `json:"injectStockFillReorder"`
	Unprotected               bool `json:"unprotected"`
	UnsafeIgnoreAuthzSnapshot bool `json:"unsafeIgnoreAuthzSnapshot"`
	UnsafeStampAtSet          bool `json:"unsafeStampAtSet"`

	ExpectProjectionDigest string `json:"-"`
}

// DefaultConfig is the protected configuration.
func DefaultConfig(unit string) Config {
	return Config{
		Unit: unit, Listen: "127.0.0.1:0", PoolMax: 4,
		AuthzMaxAge: 2 * time.Second, StockMaxAge: time.Second,
		Limits: LimitConfig{PerPrincipalRate: 5, PerPrincipalBurst: 20, TierRate: 50, TierBurst: 10,
			TierConcurrency: 2, AdmitWait: 50 * time.Millisecond},
	}
}

// Digest identifies the behaviour-relevant configuration (not addresses).
func (c Config) Digest() string {
	raw, _ := json.Marshal(c)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Ready is what a unit writes once it serves.
type Ready struct {
	Unit             string         `json:"unit"`
	Addr             string         `json:"addr"`
	PID              int            `json:"pid"`
	ProjectionDigest string         `json:"projectionDigest"`
	ConfigDigest     string         `json:"configDigest"`
	ReadPlan         ReadPlanSource `json:"readPlan"`
	DBRole           string         `json:"dbRole"`
}

type unit struct {
	cfg             Config
	proj            Projection
	pool            *pgxpool.Pool
	authz           *AuthzCache
	stock           *StockCache
	limiter         *Limiter
	counts          struct{ admitted, shed, served atomic.Int64 }
	reorderInjected atomic.Bool
	ready           Ready
}

// Serve runs one unit until ctx ends.
func Serve(ctx context.Context, cfg Config) error {
	if cfg.Unit != KindMember && cfg.Unit != KindStaff {
		return fmt.Errorf("unknown unit %q", cfg.Unit)
	}
	loaded, err := LoadWorkspace(cfg.Workspace)
	if err != nil {
		return err
	}
	proj, err := DeriveProjection(loaded, cfg.Unit)
	if err != nil {
		return err
	}
	if cfg.ExpectProjectionDigest != "" && cfg.ExpectProjectionDigest != proj.Digest {
		return fmt.Errorf("projection digest %s does not match the expected %s; refusing to start", proj.Digest, cfg.ExpectProjectionDigest)
	}
	pc, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return err
	}
	pc.MaxConns = int32(cfg.PoolMax)
	pc.MinConns = int32(cfg.PoolMax)
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return err
	}
	defer pool.Close()
	u := &unit{cfg: cfg, proj: proj, pool: pool, authz: NewAuthzCache(pool, cfg.AuthzMaxAge),
		stock: NewStockCache(cfg.StockMaxAge), limiter: NewLimiter(cfg.Limits)}
	u.authz.DropNotify = cfg.DropNotify
	u.stock.UnsafeStampAtSet = cfg.UnsafeStampAtSet
	if cfg.InjectStockFillReorder {
		u.stock.FillHook = u.waitForNewerVersion
	}
	var role string
	if err := pool.QueryRow(ctx, `SELECT current_user`).Scan(&role); err != nil {
		return err
	}
	listenCtx, stopListen := context.WithCancel(ctx)
	defer stopListen()
	listening := make(chan error, 1)
	go u.listen(listenCtx, listening)
	if err := <-listening; err != nil {
		return err
	}
	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: u.routes(), ReadHeaderTimeout: 5 * time.Second}
	u.ready = Ready{Unit: cfg.Unit, Addr: ln.Addr().String(), PID: os.Getpid(), ProjectionDigest: proj.Digest,
		ConfigDigest: cfg.Digest(), ReadPlan: proj.Source, DBRole: role}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	if cfg.ReadyFile != "" {
		raw, _ := json.Marshal(u.ready)
		tmp := cfg.ReadyFile + ".tmp"
		if err := os.WriteFile(tmp, raw, 0o644); err != nil {
			return err
		}
		if err := os.Rename(tmp, cfg.ReadyFile); err != nil {
			return err
		}
	}
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// listen holds one dedicated connection for NOTIFY owdm_authz and owdm_stock.
func (u *unit) listen(ctx context.Context, started chan<- error) {
	conn, err := pgx.Connect(ctx, u.cfg.DSN)
	if err != nil {
		started <- err
		return
	}
	defer conn.Close(context.Background())
	for _, ch := range []string{"owdm_authz", "owdm_stock"} {
		if _, err := conn.Exec(ctx, "LISTEN "+ch); err != nil {
			started <- err
			return
		}
	}
	started <- nil
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return
		}
		switch n.Channel {
		case "owdm_authz":
			u.authz.notice(n.Payload)
		case "owdm_stock":
			if u.cfg.DropNotify {
				continue
			}
			if i := strings.LastIndexByte(n.Payload, ':'); i > 0 {
				if v, err := strconv.ParseInt(n.Payload[i+1:], 10, 64); err == nil {
					u.stock.Invalidate(n.Payload[:i], v)
				}
			}
		}
	}
}

// waitForNewerVersion is the injected reorder: the first fill of the display
// cache waits until the canonical row has a newer version, and then for longer
// than the declared bound on this process's monotonic clock. The store
// therefore happens after the DB update, after its invalidation, and late
// enough that serving the old value would exceed the bound.
func (u *unit) waitForNewerVersion(ctx context.Context, v StockValue) {
	if !u.reorderInjected.CompareAndSwap(false, true) {
		return
	}
	var seen time.Time
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if seen.IsZero() {
			var cur int64
			if err := u.pool.QueryRow(ctx, `SELECT version FROM owdm.item WHERE item_code = $1`, v.ItemCode).Scan(&cur); err != nil {
				return
			}
			if cur > v.Version {
				seen = time.Now()
			}
		} else if time.Since(seen) > u.cfg.StockMaxAge {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (u *unit) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, u.ready) })
	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"authz": u.authz.Stats(), "stockCache": u.stock.Stats(),
			"admitted": u.counts.admitted.Load(), "shed": u.counts.shed.Load(), "pool": map[string]int32{
				"max": u.pool.Stat().MaxConns(), "total": u.pool.Stat().TotalConns()}})
	})
	if u.cfg.Unit == KindMember {
		mux.HandleFunc("GET /member/orders", u.member(u.listOrders))
		mux.HandleFunc("GET /member/orders/{id}", u.member(u.getOrder))
		mux.HandleFunc("PATCH /member/orders/{id}", u.member(u.patchOrder))
		mux.HandleFunc("POST /member/orders/{id}/reorder", u.member(u.reorder))
		mux.HandleFunc("GET /member/items/{code}/stock", u.member(u.stockDisplay))
	} else {
		mux.HandleFunc("GET /staff/orders/{id}", u.staff(u.staffGetOrder))
		mux.HandleFunc("POST /staff/orders/{id}/confirm", u.staff(u.confirm))
		mux.HandleFunc("POST /staff/orders", u.staff(u.createOrder))
		mux.HandleFunc("POST /staff/principals/{id}/revoke", u.staff(u.revoke))
		mux.HandleFunc("POST /staff/principals/{id}/scopes/remove", u.staff(u.removeScope))
		mux.HandleFunc("POST /staff/items/{code}/adjust", u.staff(u.adjustStock))
	}
	return mux
}

type handler func(w http.ResponseWriter, r *http.Request, g Grant)

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if t, ok := strings.CutPrefix(h, "Bearer "); ok {
		return t
	}
	return ""
}

func (u *unit) authenticate(w http.ResponseWriter, r *http.Request, kind string) (Grant, bool) {
	g, err := u.authz.Resolve(r.Context(), bearer(r))
	switch {
	case errors.Is(err, ErrUnauthenticated), errors.Is(err, ErrRevoked):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
		return g, false
	case err != nil:
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return g, false
	case g.Kind != kind:
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return g, false
	}
	w.Header().Set("X-Owdm-Authz-Version", strconv.FormatInt(g.Version, 10))
	w.Header().Set("X-Owdm-Authz-Seq", strconv.FormatInt(g.Seq, 10))
	return g, true
}

func (u *unit) member(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Owdm-Unit", u.cfg.Unit)
		g, ok := u.authenticate(w, r, KindMember)
		if !ok {
			return
		}
		if !u.cfg.Unprotected {
			d := u.limiter.Admit(g.PrincipalID)
			if !d.Admitted {
				u.counts.shed.Add(1)
				w.Header().Set("Retry-After", strconv.Itoa(d.RetryAfter))
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate-limited", "reason": d.Reason})
				return
			}
			defer d.Release()
		}
		u.counts.admitted.Add(1)
		h(w, r, g)
	}
}

func (u *unit) staff(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Owdm-Unit", u.cfg.Unit)
		g, ok := u.authenticate(w, r, KindStaff)
		if !ok {
			return
		}
		h(w, r, g)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readBody(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 64<<10))
	dec.UseNumber()
	return dec.Decode(v)
}
