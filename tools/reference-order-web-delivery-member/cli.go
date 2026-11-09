package orderwebmember

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os/signal"
	"syscall"
)

// Main is the command line of the two units.
//
//	serve -unit member|staff -dsn DSN -workspace DIR [-listen ADDR] [-ready-file FILE] ...
//	projection -unit member|staff -workspace DIR
func Main(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: serve|projection [flags]")
		return 2
	}
	switch args[0] {
	case "serve":
		return serveCmd(args[1:], stderr)
	case "projection":
		fs := flag.NewFlagSet("projection", flag.ContinueOnError)
		fs.SetOutput(stderr)
		kind := fs.String("unit", KindMember, "member or staff")
		ws := fs.String("workspace", "reference/workspace", "workspace that holds the step-1 read plan")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		loaded, err := LoadWorkspace(*ws)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		p, err := DeriveProjection(loaded, *kind)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(p)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func serveCmd(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	unitName := fs.String("unit", "", "member or staff")
	cfg := DefaultConfig("")
	fs.StringVar(&cfg.DSN, "dsn", "", "PostgreSQL DSN of the unit's own role")
	fs.StringVar(&cfg.Listen, "listen", cfg.Listen, "listen address")
	fs.StringVar(&cfg.ReadyFile, "ready-file", "", "file written with the address once serving")
	fs.StringVar(&cfg.Workspace, "workspace", "reference/workspace", "workspace that holds the step-1 read plan")
	fs.IntVar(&cfg.PoolMax, "pool-max", cfg.PoolMax, "DB pool size")
	fs.DurationVar(&cfg.AuthzMaxAge, "authz-max-age", cfg.AuthzMaxAge, "declared bound for revocation and scope reduction")
	fs.DurationVar(&cfg.StockMaxAge, "stock-max-age", cfg.StockMaxAge, "declared staleness bound of the stock display")
	fs.Float64Var(&cfg.Limits.PerPrincipalRate, "member-rate", cfg.Limits.PerPrincipalRate, "requests per second per member")
	fs.Float64Var(&cfg.Limits.PerPrincipalBurst, "member-burst", cfg.Limits.PerPrincipalBurst, "burst per member")
	fs.Float64Var(&cfg.Limits.TierRate, "tier-rate", cfg.Limits.TierRate, "requests per second of the member tier")
	fs.Float64Var(&cfg.Limits.TierBurst, "tier-burst", cfg.Limits.TierBurst, "burst of the member tier")
	fs.IntVar(&cfg.Limits.TierConcurrency, "tier-concurrency", cfg.Limits.TierConcurrency, "member requests holding a DB connection at once")
	fs.DurationVar(&cfg.Limits.AdmitWait, "admit-wait", cfg.Limits.AdmitWait, "longest wait for a member slot before 429")
	fs.IntVar(&cfg.MemberQueryBurn, "member-query-burn", 0, "fault injection: rows of generate_series counted per member order read")
	fs.BoolVar(&cfg.DropNotify, "drop-notify", false, "fault injection: ignore NOTIFY invalidations")
	fs.BoolVar(&cfg.InjectStockFillReorder, "inject-stock-fill-reorder", false, "fault injection: delay the first cache fill until the canonical item moved on")
	fs.BoolVar(&cfg.Unprotected, "unprotected", false, "negative control: no member quota or rate limit")
	fs.BoolVar(&cfg.UnsafeIgnoreAuthzSnapshot, "unsafe-ignore-authz-snapshot", false, "negative control: do not hide rows created after the grant was read")
	fs.BoolVar(&cfg.UnsafeStampAtSet, "unsafe-stamp-at-set", false, "negative control: age cache entries from the store, ignore invalidations")
	fs.StringVar(&cfg.ExpectProjectionDigest, "expect-projection-digest", "", "refuse to start unless the derived projection has this digest")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg.Unit = *unitName
	if cfg.DSN == "" {
		fmt.Fprintln(stderr, "-dsn is required")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := Serve(ctx, cfg); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
