package orderdatarealization

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	// DSNEnv names the disposable PostgreSQL database.
	DSNEnv = "TEI_ODR_DSN"
	// ConfirmEnv must equal ConfirmValue to confirm the database is disposable.
	ConfirmEnv   = "TEI_ODR_CONFIRM_ISOLATED"
	ConfirmValue = "tei_odr"
	// OutDirEnv receives the chain record from the test.
	OutDirEnv = "TEI_ODR_OUT_DIR"
)

const usage = `usage: tei-order-data-realization <command> [flags]

commands:
  design -out DIR      steps 1-4 only (no database); writes DIR/chain-record.json
  run    -out DIR      steps 1-10 against the disposable PostgreSQL database in
                       TEI_ODR_DSN (TEI_ODR_CONFIRM_ISOLATED=tei_odr is required)

flags:
  -workspace DIR       workspace to copy (default reference/workspace); it is only read
  -out DIR             directory for the chain record and the working copies
`

// Main is the CLI entry point; it returns the process exit code.
func Main(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	cmd := args[0]
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	ws := fs.String("workspace", "reference/workspace", "workspace to copy")
	out := fs.String("out", "", "output directory")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *out == "" || (cmd != "design" && cmd != "run") {
		fmt.Fprint(stderr, usage)
		return 2
	}
	root, err := FindRepoRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	src, err := filepath.Abs(*ws)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	c, err := NewChain(root, src, filepath.Join(*out, "work"))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	switch cmd {
	case "design":
		err = c.RunDesign(ctx)
		c.Rec.Passed = err == nil
	case "run":
		dsn := os.Getenv(DSNEnv)
		if dsn == "" || os.Getenv(ConfirmEnv) != ConfirmValue {
			fmt.Fprintf(stderr, "run needs %s and %s=%s\n", DSNEnv, ConfirmEnv, ConfirmValue)
			return 2
		}
		err = c.Run(ctx, dsn)
	}
	raw, _ := json.MarshalIndent(c.Rec, "", "  ")
	if werr := os.WriteFile(filepath.Join(*out, "chain-record.json"), append(raw, '\n'), 0o644); werr != nil {
		fmt.Fprintln(stderr, werr)
		return 1
	}
	fmt.Fprintf(stdout, "{\"passed\": %v, \"record\": %q}\n", c.Rec.Passed, filepath.Join(*out, "chain-record.json"))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !c.Rec.Passed {
		return 1
	}
	return 0
}
