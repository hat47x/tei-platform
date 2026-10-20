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
  snapshot  -plan plan.json                step 2: create the change feed slot, copy the legacy rows, bind them to the plan
  migrate   -plan plan.json                step 3: migrate chunks into the primary DB (resumable)
  dispatch  -plan plan.json                step 3: deliver pending reservations to the inventory DB
  run       -plan plan.json                step 3: migrate, then one dispatch pass
  status    -plan plan.json                observe both DBs and classify every migrated order
  reconcile -plan plan.json                step 3: compare the staged copy with the primary DB
  apply-changes -plan plan.json            step 4: apply the legacy change feed to the primary DB
  switch-owner  -plan plan.json -field F -owner legacy|primary
                                           step 4: move the master of one customer field
  expand-name   -plan plan.json            step 4: expand and migrate phases of the name split
  contract-name -plan plan.json            step 4: check the contract phase (refused while old writers remain)
  send-billing  -plan plan.json -billing-url URL
                                           step 4: send pending customer changes to the billing system
  reconcile-billing -plan plan.json -billing-url URL
                                           step 4: settle unknown billing results by querying the billing system
  reverse-sync  -plan plan.json            steps 4/6: deliver primary changes to the legacy DB
  gate      -plan plan.json                step 5: reconcile and, if nothing blocks, switch to the primary
  rollback  -plan plan.json                step 6: rollback drill back to the legacy DB
`

type cliArgs struct {
	planPath, out, field, owner string
	chunkSize                   int
	opt                         DispatchOptions
	billing                     BillingOptions
}

// Main is the CLI entry point; it returns the process exit code.
func Main(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	cmd := args[0]
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	var a cliArgs
	fs.StringVar(&a.planPath, "plan", "", "plan file")
	fs.StringVar(&a.out, "out", "", "plan output file (plan command)")
	fs.IntVar(&a.chunkSize, "chunk-size", 20, "orders per chunk (plan command)")
	fs.StringVar(&a.field, "field", "", "customer field (switch-owner)")
	fs.StringVar(&a.owner, "owner", "", "new owner: legacy or primary (switch-owner)")
	fs.StringVar(&a.billing.URL, "billing-url", "", "billing system base URL")
	fs.DurationVar(&a.billing.Timeout, "billing-timeout", 2*time.Second, "billing call bound")
	connectTimeout := fs.Duration("connect-timeout", DefaultDispatchOptions().ConnectTimeout, "inventory connect bound")
	opTimeout := fs.Duration("op-timeout", DefaultDispatchOptions().OpTimeout, "inventory operation bound")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	a.opt = DispatchOptions{ConnectTimeout: *connectTimeout, OpTimeout: *opTimeout}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	result, err := runCommand(ctx, cmd, a)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", cmd, err)
		if result != nil {
			json.NewEncoder(stdout).Encode(result)
		}
		return 1
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	switch r := result.(type) {
	case ReconcileResult:
		if !r.Passed {
			return 3
		}
	case GateResult:
		if !r.Passed {
			return 3
		}
	case RollbackResult:
		if !r.Passed {
			return 3
		}
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

func runCommand(ctx context.Context, cmd string, a cliArgs) (any, error) {
	if cmd == "plan" {
		if a.out == "" {
			return nil, fmt.Errorf("-out is required")
		}
		legacy, err := connectEnv(ctx, RoleLegacy)
		if err != nil {
			return nil, err
		}
		defer legacy.Close(context.Background())
		plan, err := MakePlan(ctx, legacy, a.chunkSize)
		if err != nil {
			return nil, err
		}
		if err := WritePlan(a.out, plan); err != nil {
			return nil, err
		}
		return map[string]any{"planDigest": plan.Digest, "chunks": len(plan.Chunks), "observed": plan.Source.Observed, "knownDifferences": plan.KnownDifferences}, nil
	}
	if a.planPath == "" {
		return nil, fmt.Errorf("-plan is required")
	}
	plan, err := ReadPlan(a.planPath)
	if err != nil {
		return nil, err
	}
	primary, err := connectEnv(ctx, RolePrimary)
	if err != nil {
		return nil, err
	}
	defer primary.Close(context.Background())
	inventoryDSN := os.Getenv(RoleInventory.DSNEnv())
	withLegacy := func(fn func(legacy *pgx.Conn) (any, error)) (any, error) {
		legacy, err := connectEnv(ctx, RoleLegacy)
		if err != nil {
			return nil, err
		}
		defer legacy.Close(context.Background())
		return fn(legacy)
	}
	bound := func() error {
		_, err := CheckBound(ctx, primary, plan)
		return err
	}
	switch cmd {
	case "snapshot":
		return withLegacy(func(legacy *pgx.Conn) (any, error) {
			rec, existing, err := TakeSnapshot(ctx, legacy, primary, plan)
			return map[string]any{"snapshot": rec, "existing": existing}, err
		})
	case "migrate":
		return Migrate(ctx, primary, plan)
	case "dispatch":
		if err := bound(); err != nil {
			return nil, err
		}
		return Dispatch(ctx, primary, inventoryDSN, a.opt)
	case "run":
		return Run(ctx, primary, inventoryDSN, plan, a.opt)
	case "status":
		if err := bound(); err != nil {
			return nil, err
		}
		return Status(ctx, primary, inventoryDSN, plan, a.opt)
	case "reconcile":
		return Reconcile(ctx, primary, plan)
	case "apply-changes":
		return withLegacy(func(legacy *pgx.Conn) (any, error) { return ApplyChanges(ctx, legacy, primary, plan) })
	case "switch-owner":
		if err := SwitchFieldOwner(ctx, primary, plan, a.field, a.owner, "cli:switch-owner"); err != nil {
			return nil, err
		}
		return map[string]string{"field": a.field, "owner": a.owner}, nil
	case "expand-name":
		return ExpandNameSplit(ctx, primary, plan)
	case "contract-name":
		return CheckContractNameSplit(ctx, primary, plan)
	case "send-billing", "reconcile-billing":
		if a.billing.URL == "" {
			return nil, errBillingURL
		}
		if err := bound(); err != nil {
			return nil, err
		}
		if cmd == "send-billing" {
			return SendBilling(ctx, primary, a.billing)
		}
		return ReconcileBilling(ctx, primary, a.billing)
	case "reverse-sync":
		return withLegacy(func(legacy *pgx.Conn) (any, error) { return ReverseSync(ctx, legacy, primary, plan) })
	case "gate":
		return withLegacy(func(legacy *pgx.Conn) (any, error) { return Gate(ctx, legacy, primary, inventoryDSN, plan, a.opt) })
	case "rollback":
		return withLegacy(func(legacy *pgx.Conn) (any, error) { return RollbackDrill(ctx, legacy, primary, plan) })
	}
	return nil, fmt.Errorf("unknown command %q\n%s", cmd, usage)
}
