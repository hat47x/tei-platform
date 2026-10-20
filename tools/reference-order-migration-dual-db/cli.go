package ordermigrationdualdb

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

const usage = `usage: tei-order-migration <command> [flags]

commands (DSNs come from TEI_OMDD_LEGACY_DSN, TEI_OMDD_PRIMARY_DSN, TEI_OMDD_INVENTORY_DSN):
  plan      -chunk-size N -out plan.json   step 1: observe the legacy DB and write the plan
  snapshot  -plan plan.json                step 2: copy the legacy rows and bind them to the plan
  migrate   -plan plan.json                step 3: migrate chunks into the primary DB (resumable)
  dispatch  -plan plan.json                step 3: deliver pending reservations to the inventory DB
  run       -plan plan.json                step 3: migrate, then one dispatch pass
  status    -plan plan.json                observe both DBs and classify every migrated order
  reconcile -plan plan.json                compare the staged copy with the primary DB
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
	planPath := fs.String("plan", "", "plan file")
	out := fs.String("out", "", "plan output file (plan command)")
	chunkSize := fs.Int("chunk-size", 20, "orders per chunk (plan command)")
	connectTimeout := fs.Duration("connect-timeout", DefaultDispatchOptions().ConnectTimeout, "inventory connect bound")
	opTimeout := fs.Duration("op-timeout", DefaultDispatchOptions().OpTimeout, "inventory operation bound")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	opt := DispatchOptions{ConnectTimeout: *connectTimeout, OpTimeout: *opTimeout}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	result, err := runCommand(ctx, cmd, *planPath, *out, *chunkSize, opt)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", cmd, err)
		return 1
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if r, ok := result.(ReconcileResult); ok && !r.Passed {
		return 3
	}
	return 0
}

func connectEnv(ctx context.Context, role Role) (*pgx.Conn, error) {
	dsn := os.Getenv(role.DSNEnv())
	if dsn == "" {
		return nil, fmt.Errorf("%s is not set", role.DSNEnv())
	}
	return Connect(ctx, dsn, role, 10*time.Second)
}

func runCommand(ctx context.Context, cmd, planPath, out string, chunkSize int, opt DispatchOptions) (any, error) {
	if cmd == "plan" {
		if out == "" {
			return nil, fmt.Errorf("-out is required")
		}
		legacy, err := connectEnv(ctx, RoleLegacy)
		if err != nil {
			return nil, err
		}
		defer legacy.Close(context.Background())
		plan, err := MakePlan(ctx, legacy, chunkSize)
		if err != nil {
			return nil, err
		}
		if err := WritePlan(out, plan); err != nil {
			return nil, err
		}
		return map[string]any{"planDigest": plan.Digest, "chunks": len(plan.Chunks), "observed": plan.Source.Observed}, nil
	}
	if planPath == "" {
		return nil, fmt.Errorf("-plan is required")
	}
	plan, err := ReadPlan(planPath)
	if err != nil {
		return nil, err
	}
	primary, err := connectEnv(ctx, RolePrimary)
	if err != nil {
		return nil, err
	}
	defer primary.Close(context.Background())
	inventoryDSN := os.Getenv(RoleInventory.DSNEnv())
	switch cmd {
	case "snapshot":
		legacy, err := connectEnv(ctx, RoleLegacy)
		if err != nil {
			return nil, err
		}
		defer legacy.Close(context.Background())
		rec, existing, err := TakeSnapshot(ctx, legacy, primary, plan)
		return map[string]any{"snapshot": rec, "existing": existing}, err
	case "migrate":
		return Migrate(ctx, primary, plan)
	case "dispatch":
		if _, err := CheckBound(ctx, primary, plan); err != nil {
			return nil, err
		}
		return Dispatch(ctx, primary, inventoryDSN, opt)
	case "run":
		return Run(ctx, primary, inventoryDSN, plan, opt)
	case "status":
		if _, err := CheckBound(ctx, primary, plan); err != nil {
			return nil, err
		}
		return Status(ctx, primary, inventoryDSN, plan, opt)
	case "reconcile":
		return Reconcile(ctx, primary, plan)
	}
	return nil, fmt.Errorf("unknown command %q\n%s", cmd, usage)
}
