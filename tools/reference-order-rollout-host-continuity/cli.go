package orderrollouthost

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Main runs the CLI. Subcommands:
//
//	start  print the output of step 1 (read-only)
//	host   run one supervising host process (-role a invokes, -role b takes over)
//	atlas  the stand-in migration tool executable (reads a plugin request on stdin)
//
// When the executable is started under the name "atlas", it is the tool.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tei-order-rollout-host start|host|atlas [flags]")
		return 2
	}
	switch args[0] {
	case "atlas":
		return runAtlas(stdin, stdout, stderr)
	case "start":
		fs := flag.NewFlagSet("start", flag.ContinueOnError)
		repo := fs.String("repo", ".", "repository root")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		s, err := LoadStart(DefaultPaths(*repo))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return encode(stdout, s)
	case "host":
		return runHostCLI(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "unknown subcommand %q\n", args[0])
	return 2
}

func encode(w io.Writer, v any) int {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return 1
	}
	return 0
}

func runHostCLI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("host", flag.ContinueOnError)
	var cfg HostConfig
	repo := fs.String("repo", ".", "repository root (fixed inputs)")
	ws := fs.String("workspace", "", "workspace to build the environment plan from (default: the fixed reference workspace)")
	out := fs.String("out", "", "report file")
	fs.StringVar(&cfg.Role, "role", "a", "a (invoke) or b (take over)")
	fs.StringVar(&cfg.Owner, "owner", "", "supervisor owner id")
	fs.StringVar(&cfg.BinDir, "bin-dir", "", "absolute directory holding the atlas executable (host A)")
	fs.DurationVar(&cfg.ClockOffset, "clock-offset", 0, "skew of this host's clock (recorded only)")
	fs.DurationVar(&cfg.OwnershipTTL, "ownership-ttl", 3*time.Second, "supervisor ownership ttl")
	fs.DurationVar(&cfg.LeaseSlack, "lease-slack", 400*time.Millisecond, "resource lease ttl beyond the plan's renew-before")
	fs.IntVar(&cfg.ApplyHoldMS, "apply-hold-ms", 0, "tool: keep the expand transaction open this long")
	fs.BoolVar(&cfg.FailRenew, "fail-renew", false, "inject lease renewal failure")
	fs.BoolVar(&cfg.FailHeartbeat, "fail-heartbeat", false, "inject ownership heartbeat failure")
	fs.BoolVar(&cfg.ProbeStale, "probe-stale", false, "host A: retry every authority-bearing operation after the call")
	fs.DurationVar(&cfg.ClaimPoll, "claim-poll", 100*time.Millisecond, "host B: claim poll interval")
	fs.DurationVar(&cfg.ClaimTimeout, "claim-timeout", 2*time.Minute, "host B: give up the takeover after this")
	fs.DurationVar(&cfg.ObserveWindow, "observe-window", time.Second, "window in which a running execution must show progress")
	fs.StringVar(&cfg.ReleaseBarrier, "release-barrier", "", "host B: wait for this barrier before returning the lease")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg.Paths = DefaultPaths(*repo)
	if *ws != "" {
		cfg.Paths.Workspace = *ws
	}
	cfg.ChildArgs = os.Getenv(ChildArgsEnv) != ""
	if cfg.Owner == "" || *out == "" {
		fmt.Fprintln(stderr, "-owner and -out are required")
		return 2
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()
	sec := SecretsFromEnv()
	var report any
	var err error
	switch cfg.Role {
	case "a":
		report, err = RunHostA(ctx, cfg, sec)
	case "b":
		report, err = RunHostB(ctx, cfg, sec)
	default:
		fmt.Fprintln(stderr, "-role must be a or b")
		return 2
	}
	if werr := writeJSON(*out, map[string]any{"config": cfg, "report": report, "error": errString(err)}); werr != nil {
		fmt.Fprintln(stderr, "write report failed")
		return 1
	}
	fmt.Fprintf(stdout, "host %s (%s) finished; report written\n", cfg.Role, cfg.Owner)
	if err != nil {
		fmt.Fprintln(stderr, "host error:", err)
		return 1
	}
	return 0
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
