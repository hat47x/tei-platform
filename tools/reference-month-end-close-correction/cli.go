package monthendclose

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const usage = `usage: tei-month-end-close <command> [flags]

commands (the DSN of the dedicated database tei_mecc comes from TEI_MECC_DSN):
  calendar-load  -file calendar.json            register one calendar version
  calendar-revise -calendar ID -from N -holiday DATE=NAME [-holiday ...] -recorded-at T
                                                register the next calendar version
  payable-load   -file payables.json            register payables (unpaid items with nominal due dates)
  decide         -month YYYY-MM -calendar ID -version N -rule rule.json -evaluated-at T -out decision.json
                                                step 1: close date, cut-off, jobs, due dates; activate; impact
  import         -decision d.json -spec spec.json -file F -received-at T
                                                step 2: import one fixed-length file as one unit
  message        -decision d.json -spec spec.json -file messages.json
                                                step 2: apply simulated message deliveries
  reconcile-shipments -decision d.json          step 2: classify shipments by source and record
  reemit         -spec spec.json -import-id N -out F
                                                step 2: rebuild an imported file from the stored records
  close          -decision d.json -owner ID -ttl D -chunk-size N -as-of T
                                                step 3: run or resume the close job of the month

Refusals (rejected files, refused or fenced close runs, decision mismatches) exit 4.
`

type cliArgs struct {
	file, calendar, rule, out, decision, spec, month, owner string
	evaluatedAt, receivedAt, recordedAt, asOf               string
	version, from, chunkSize                                int
	importID                                                int64
	ttl, chunkPause                                         time.Duration
	holidays                                                multiFlag
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

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
	fs.StringVar(&a.file, "file", "", "input file")
	fs.StringVar(&a.calendar, "calendar", "", "calendar id")
	fs.StringVar(&a.rule, "rule", "", "close rule file")
	fs.StringVar(&a.out, "out", "", "output file")
	fs.StringVar(&a.decision, "decision", "", "decision file from decide")
	fs.StringVar(&a.spec, "spec", "", "file specification")
	fs.StringVar(&a.month, "month", "", "target month YYYY-MM")
	fs.StringVar(&a.owner, "owner", "", "close job owner (server) id")
	fs.StringVar(&a.evaluatedAt, "evaluated-at", "", "evaluation instant (RFC 3339, explicit)")
	fs.StringVar(&a.receivedAt, "received-at", "", "file reception instant (RFC 3339)")
	fs.StringVar(&a.recordedAt, "recorded-at", "", "calendar revision instant (RFC 3339)")
	fs.StringVar(&a.asOf, "as-of", "", "business instant the close job runs at (RFC 3339)")
	fs.IntVar(&a.version, "version", 0, "calendar version")
	fs.IntVar(&a.from, "from", 0, "calendar version to revise")
	fs.IntVar(&a.chunkSize, "chunk-size", 10, "close entries per chunk")
	fs.Int64Var(&a.importID, "import-id", 0, "import unit id")
	fs.DurationVar(&a.ttl, "ttl", 3*time.Second, "close job lease ttl (judged by the database clock)")
	fs.DurationVar(&a.chunkPause, "chunk-pause", 0, "pause after each close chunk (test pacing)")
	fs.Var(&a.holidays, "holiday", "added holiday DATE=NAME (repeatable)")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	result, err := runCommand(ctx, cmd, a)
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", cmd, err)
		if reason, ok := refusal(err); ok {
			enc.Encode(map[string]any{"refused": reason, "error": err.Error(), "partial": result})
			return 4
		}
		return 1
	}
	if err := enc.Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func refusal(err error) (string, bool) {
	if r, ok := IsReject(err); ok {
		return r, true
	}
	for _, e := range []error{ErrDecisionMismatch, ErrAlreadyRunning, ErrAlreadyCompleted, ErrNotYetScheduled, ErrNoImport, ErrFenced, ErrCalendarNotYetKnown, ErrMonthClosing} {
		if errors.Is(err, e) {
			return e.Error(), true
		}
	}
	return "", false
}

func parseTime(name, v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, fmt.Errorf("-%s is required (an explicit instant; the host clock is not used)", name)
	}
	return time.Parse(time.RFC3339, v)
}

func runCommand(ctx context.Context, cmd string, a cliArgs) (any, error) {
	dsn := os.Getenv(DSNEnv)
	if dsn == "" {
		return nil, fmt.Errorf("%s is not set", DSNEnv)
	}
	conn, err := Connect(ctx, dsn, 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close(context.Background())
	decision := func() (Decision, error) {
		if a.decision == "" {
			return Decision{}, fmt.Errorf("-decision is required")
		}
		return ReadDecision(a.decision)
	}
	switch cmd {
	case "calendar-load":
		c, err := ReadCalendar(a.file)
		if err != nil {
			return nil, err
		}
		created, err := LoadCalendar(ctx, conn, c)
		return map[string]any{"calendar": c.ID, "version": c.Version, "digest": c.Digest, "created": created}, err
	case "calendar-revise":
		var add []Holiday
		for _, h := range a.holidays {
			date, name, _ := strings.Cut(h, "=")
			add = append(add, Holiday{Date: date, Name: name})
		}
		c, err := ReviseCalendar(ctx, conn, a.calendar, a.from, add, a.recordedAt)
		if err != nil {
			return nil, err
		}
		return map[string]any{"calendar": c.ID, "version": c.Version, "revises": c.Revises, "digest": c.Digest, "recordedAt": c.RecordedAt}, nil
	case "payable-load":
		var ps []Payable
		if err := readJSON(a.file, &ps); err != nil {
			return nil, err
		}
		for _, p := range ps {
			if _, err := conn.Exec(ctx, `INSERT INTO mecc.payable VALUES ($1,$2::text::date) ON CONFLICT (payable_id) DO UPDATE SET nominal_due=EXCLUDED.nominal_due`, p.ID, p.NominalDue); err != nil {
				return nil, err
			}
		}
		return map[string]any{"payables": len(ps)}, nil
	case "decide":
		rule, err := ReadRule(a.rule)
		if err != nil {
			return nil, err
		}
		at, err := parseTime("evaluated-at", a.evaluatedAt)
		if err != nil {
			return nil, err
		}
		if a.out == "" {
			return nil, fmt.Errorf("-out is required")
		}
		res, err := Decide(ctx, conn, a.month, a.calendar, a.version, rule, at)
		if err != nil {
			return nil, err
		}
		return res, WriteDecision(a.out, res.Decision)
	case "import":
		d, err := decision()
		if err != nil {
			return nil, err
		}
		spec, err := ReadFileSpec(a.spec)
		if err != nil {
			return nil, err
		}
		b, err := os.ReadFile(a.file)
		if err != nil {
			return nil, err
		}
		at, err := parseTime("received-at", a.receivedAt)
		if err != nil {
			return nil, err
		}
		return ImportFile(ctx, conn, d, spec, b, at)
	case "message":
		d, err := decision()
		if err != nil {
			return nil, err
		}
		spec, err := ReadFileSpec(a.spec)
		if err != nil {
			return nil, err
		}
		var msgs []Message
		if err := readJSON(a.file, &msgs); err != nil {
			return nil, err
		}
		outcomes := map[string]int{}
		for _, m := range msgs {
			o, err := ReceiveMessage(ctx, conn, d, spec, m)
			if err != nil {
				return outcomes, err
			}
			outcomes[o]++
		}
		return outcomes, nil
	case "reconcile-shipments":
		d, err := decision()
		if err != nil {
			return nil, err
		}
		return ReconcileShipments(ctx, conn, d)
	case "reemit":
		spec, err := ReadFileSpec(a.spec)
		if err != nil {
			return nil, err
		}
		b, err := Reemit(ctx, conn, spec, a.importID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"bytes": len(b), "sha256": FileSHA256(b)}, os.WriteFile(a.out, b, 0o644)
	case "close":
		d, err := decision()
		if err != nil {
			return nil, err
		}
		at, err := parseTime("as-of", a.asOf)
		if err != nil {
			return nil, err
		}
		return RunClose(ctx, conn, d, CloseOptions{OwnerID: a.owner, TTL: a.ttl, ChunkSize: a.chunkSize, AsOf: at, HostNow: HostClock(), ChunkPause: a.chunkPause})
	}
	return nil, fmt.Errorf("unknown command %q\n%s", cmd, usage)
}
