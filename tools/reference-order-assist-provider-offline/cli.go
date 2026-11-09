package orderassistoffline

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FaultExitCode is the exit status of a saga worker stopped by an injected fault.
const FaultExitCode = 3

// Main runs the CLI: the external credit provider process, one saga worker
// run, or the realization build.
func Main(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tei-order-assist-offline credit-provider|saga|realizations [flags]")
		return 2
	}
	var err error
	switch args[0] {
	case "credit-provider":
		err = runCreditProvider(args[1:], stderr)
	case "saga":
		err = runSaga(args[1:], stdout, stderr)
	case "realizations":
		err = runRealizations(args[1:], stdout, stderr)
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runCreditProvider(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("credit-provider", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dsn := fs.String("dsn", "", "DSN of the oapo_credit role")
	listen := fs.String("listen", "127.0.0.1:0", "listen address")
	ready := fs.String("ready-file", "", "file to write the listen address to")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: (&CreditProvider{Pool: pool}).Handler()}
	if *ready != "" {
		raw, _ := json.Marshal(map[string]string{"url": "http://" + ln.Addr().String()})
		tmp := *ready + ".tmp"
		if err := os.WriteFile(tmp, raw, 0o600); err != nil {
			return err
		}
		if err := os.Rename(tmp, *ready); err != nil {
			return err
		}
	}
	go func() { <-ctx.Done(); _ = srv.Close() }()
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func runSaga(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("saga", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dsn := fs.String("dsn", "", "DSN of the oapo_server role")
	creditURL := fs.String("credit-url", "", "credit provider base URL")
	caseID := fs.String("case", "", "case id")
	fault := fs.String("fault", "", "exit-after-reserve-response | exit-before-release-send | exit-after-release-response")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	s := &Saga{Pool: pool, Credit: NewCreditClient(*creditURL)}
	exit := func() {
		fmt.Fprintf(stderr, "injected fault %s: stopping the worker\n", *fault)
		os.Exit(FaultExitCode)
	}
	switch *fault {
	case "":
	case "exit-after-reserve-response":
		s.Hooks.AfterReserveResponse = exit
	case "exit-before-release-send":
		s.Hooks.BeforeReleaseSend = exit
	case "exit-after-release-response":
		s.Hooks.AfterReleaseResponse = exit
	default:
		return fmt.Errorf("unknown fault %q", *fault)
	}
	steps, err := s.Run(ctx, *caseID)
	_ = json.NewEncoder(stdout).Encode(steps)
	return err
}

func runRealizations(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("realizations", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", ".", "repository root")
	out := fs.String("out", "", "build directory (outside the repository)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("-out is required")
	}
	set, err := BuildRealizations(context.Background(), *repo, *out)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(set.ByName)
}
