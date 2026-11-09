// reference-order-entry-persistenceは、order_entryの注文を実PostgreSQLへ書き込み、
// 別のprocessがそれを読み戻すことを確かめる参照ツールである(#1127)。
//
// 一つの試行を四つのprocessに分ける: prepare(fixture作成)、write(Command)、
// read(別processの観測)、cleanup。receiptはprocessごとに出力し、join modeが
// 二つのreceiptを結合できるかを判定する。
//
// このツールが主張しないこと: Sourceは合成のままで人的Acceptanceではない。Host許可も合成。
// 通常のWeb経路は通らない。DDLはPersistence Mappingから導出するが、CRUDは生成していない。
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"tei/internal/database/physical"
	postgresprovider "tei/internal/database/providers/postgres"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
	"tei/tools/internal/repositoryevidence"
)

const maxReceiptBytes = 1 << 20

type options struct {
	mode           string
	workspaceRoot  string
	orderID        int64
	commandID      string
	prepareReceipt string
	writeReceipt   string
	readReceipt    string
}

func main() {
	out, err := run(os.Args[1:], os.Stdout)
	if out != nil {
		encoded, encodeErr := json.MarshalIndent(out, "", "  ")
		if encodeErr != nil {
			fmt.Fprintln(os.Stderr, encodeErr)
			os.Exit(1)
		}
		fmt.Println(string(encoded))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) (*receipt, error) {
	flags := flag.NewFlagSet("reference-order-entry-persistence", flag.ContinueOnError)
	var opts options
	flags.StringVar(&opts.mode, "mode", "", "prepare | write | read | cleanup | join")
	flags.StringVar(&opts.workspaceRoot, "workspace", "reference/workspace", "TEI workspace directory")
	flags.Int64Var(&opts.orderID, "order-id", fixtureOrderID, "対象の注文key")
	flags.StringVar(&opts.commandID, "command-id", "", "Command ID(省略時は生成)")
	flags.StringVar(&opts.prepareReceipt, "prepare-receipt", "", "prepare modeのreceipt(write・cleanupで必要)")
	flags.StringVar(&opts.writeReceipt, "write-receipt", "", "write modeのreceipt(read・joinで必要)")
	flags.StringVar(&opts.readReceipt, "read-receipt", "", "read modeのreceipt(joinで必要)")
	if err := flags.Parse(args); err != nil {
		return nil, err
	}
	if flags.NArg() != 0 {
		return nil, fmt.Errorf("想定外の位置引数があります")
	}
	switch opts.mode {
	case "join":
		return nil, runJoin(opts, stdout)
	case "prepare", "write", "read", "cleanup":
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		return runDatabaseMode(ctx, opts)
	default:
		return nil, fmt.Errorf("-modeはprepare・write・read・cleanup・joinのいずれかです")
	}
}

// ---------------------------------------------------------------------------
// 共通
// ---------------------------------------------------------------------------

type environmentInputs struct {
	repository repositoryState
	loaded     workspace.Result
	world      ir.World
	revision   string
}

// gatherInputsは、DBへ接続する前に、clean revisionのリポジトリとSourceを確定する。
// dirty treeの診断実行は証拠にしない。
func gatherInputs(opts options) (environmentInputs, error) {
	source, err := repositoryevidence.InspectStart()
	if err != nil {
		return environmentInputs{}, err
	}
	if !source.WorkingTreeClean {
		return environmentInputs{}, fmt.Errorf("working treeに未commit差分があるため、DBへ接続しません。commitしてから実行してください")
	}
	loaded, err := workspace.Load(opts.workspaceRoot)
	if err != nil {
		return environmentInputs{}, fmt.Errorf("workspaceを読み込めません: %w", err)
	}
	index, err := workspace.BuildSourceIndex(opts.workspaceRoot, loaded)
	if err != nil {
		return environmentInputs{}, fmt.Errorf("Source Indexを作れません: %w", err)
	}
	return environmentInputs{
		repository: repositoryState{Revision: source.RepositoryRevision, Clean: source.WorkingTreeClean},
		loaded:     loaded,
		world:      loaded.World,
		revision:   sourceindex.ModelRevision(index),
	}, nil
}

func baseReceipt(mode string, in environmentInputs) *receipt {
	return &receipt{
		Schema:              receiptSchema,
		Mode:                mode,
		EvidenceNature:      evidenceNature,
		Application:         string(applicationID),
		Repository:          in.repository,
		SourceModelRevision: in.revision,
		Scope:               fixedScope(),
		Process:             processInfo{ProcessID: os.Getpid()},
		NotExecuted:         notExecuted(),
	}
}

func finishRepository(r *receipt) {
	if post, err := repositoryevidence.InspectPost(r.Repository.Revision); err == nil {
		r.Repository.PostRun = post
	}
}

func readReceiptFile(path string) (receipt, error) {
	var r receipt
	if path == "" {
		return r, fmt.Errorf("receiptのpathが指定されていません")
	}
	file, err := os.Open(path)
	if err != nil {
		return r, fmt.Errorf("receiptを開けません")
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxReceiptBytes+1))
	if err != nil || len(raw) > maxReceiptBytes {
		return r, fmt.Errorf("receiptを読めません")
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return r, fmt.Errorf("receiptがJSONではありません")
	}
	return r, nil
}

func newCommandID() (string, error) {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("Command idを生成できません")
	}
	return "cmd-" + hex.EncodeToString(random[:]), nil
}

func requireOwnReceipt(r receipt, mode string) error {
	if r.Schema != receiptSchema || r.Mode != mode || r.Application != string(applicationID) {
		return fmt.Errorf("%s receiptではありません(schema・mode・applicationが不一致)", mode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// join
// ---------------------------------------------------------------------------

func runJoin(opts options, stdout io.Writer) error {
	write, err := readReceiptFile(opts.writeReceipt)
	if err != nil {
		return err
	}
	read, err := readReceiptFile(opts.readReceipt)
	if err != nil {
		return err
	}
	problems := verifyJoin(write, read)
	fmt.Fprintln(stdout, joinSummary(problems))
	if len(problems) != 0 {
		return errors.New("receiptを結合できません")
	}
	return nil
}

// ---------------------------------------------------------------------------
// DBを使うmode
// ---------------------------------------------------------------------------

func runDatabaseMode(ctx context.Context, opts options) (*receipt, error) {
	dsn, err := loadPostgresDSNInput(os.LookupEnv, os.ReadFile)
	if err != nil {
		return nil, err
	}
	if err := clearPostgresDSNEnvironment(); err != nil {
		return nil, err
	}
	in, err := gatherInputs(opts)
	if err != nil {
		return nil, err
	}
	db, err := openDB(dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("PostgreSQL接続確認に失敗しました(接続情報は表示しません)")
	}
	var (
		out *receipt
	)
	switch opts.mode {
	case "prepare":
		out, err = runPrepare(ctx, in, db)
	case "write":
		out, err = runWrite(ctx, opts, in, db)
	case "read":
		out, err = runRead(ctx, opts, in, db)
	case "cleanup":
		out, err = runCleanup(ctx, opts, in, db)
	}
	if out != nil {
		finishRepository(out)
	}
	return out, err
}

func runPrepare(ctx context.Context, in environmentInputs, db *sql.DB) (*receipt, error) {
	out := baseReceipt("prepare", in)
	stages, target := preflightWorld(in.world)
	out.Preflight = stages
	if !allResolved(stages) || target == nil {
		return out, fmt.Errorf("DBへ書き込む前の世界の解決が完了しません")
	}
	ddl, err := renderCreateDDL(in.world, target)
	if err != nil {
		return out, err
	}
	safety, err := ensureDedicatedEmptyDatabase(ctx, db)
	out.Safety = &safety
	if err != nil {
		return out, err
	}
	marker, err := createFixture(ctx, db, ddl)
	if err != nil {
		return out, err
	}
	keepFixture := false
	defer func() {
		if keepFixture {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if cerr := removeFixture(cleanupCtx, db, marker); cerr != nil {
			fmt.Fprintf(os.Stderr, "fixture cleanup warning: %v\n", cerr)
		}
	}()
	introspect := func(c context.Context) (physical.SchemaSnapshot, error) {
		return postgresprovider.Provider{Catalog: postgresprovider.SQLCatalogReader{DB: db}}.Introspect(c, target.Instance)
	}
	schemaStage := preflightSchema(ctx, in.world, target, introspect)
	out.Preflight = append(out.Preflight, schemaStage)
	matches := schemaStage.Status == stageResolved
	out.Prepare = &prepareEvidence{
		DDLStatements:        len(ddl),
		OwnershipMarker:      marker,
		SeedOrderID:          fixtureOrderID,
		SchemaMatchesDesired: matches,
	}
	if !matches {
		return out, fmt.Errorf("DDL適用後も物理schemaが望ましい形に一致しません: %s", schemaStage.Detail)
	}
	keepFixture = true
	return out, nil
}

func runWrite(ctx context.Context, opts options, in environmentInputs, db *sql.DB) (*receipt, error) {
	out := baseReceipt("write", in)
	prepared, err := readReceiptFile(opts.prepareReceipt)
	if err != nil {
		return out, err
	}
	if err := requireOwnReceipt(prepared, "prepare"); err != nil || prepared.Prepare == nil {
		return out, fmt.Errorf("prepare receiptが不正です")
	}
	// 接続先がprepareで作ったfixtureであることを、DB側のmarkerで確認する。
	observed, err := observeOwnershipMarker(ctx, db)
	if err != nil {
		return out, err
	}
	if observed != prepared.Prepare.OwnershipMarker {
		return out, fmt.Errorf("安全のため停止します: DB側ownership markerがprepare receiptと一致しません")
	}
	stages, target := preflightWorld(in.world)
	// Source revisionの不一致は、物理schemaの不一致とは別の段階として拒否する(S02)。
	stages = append(stages, checkExpectedSource(prepared.SourceModelRevision, in.revision))
	introspect := func(c context.Context) (physical.SchemaSnapshot, error) {
		if target == nil {
			return physical.SchemaSnapshot{}, errors.New("target unresolved")
		}
		return postgresprovider.Provider{Catalog: postgresprovider.SQLCatalogReader{DB: db}}.Introspect(c, target.Instance)
	}
	stages = append(stages, preflightSchema(ctx, in.world, target, introspect))
	out.Preflight = stages

	commandID := opts.commandID
	if commandID == "" {
		commandID, err = newCommandID()
		if err != nil {
			return out, err
		}
	}
	// 許可は合成: 常にcurrent。実Hostの現在の効果許可ではない(receiptのscopeに明記)。
	adapters := &adapterSet{store: pgStore{db: db}, grant: func(context.Context) error { return nil }}
	resolver := driver.NewResolver(in.world)
	gate := submitIfReady(ctx, resolver, stages, adapters, opts.orderID)

	evidence := &writeEvidence{
		CommandID:         commandID,
		OrderID:           opts.orderID,
		BindingIdentity:   bindingIdentity,
		GateRefused:       gate.Refused,
		CommandDispatched: adapters.invoked > 0,
		WriteAttempts:     adapters.writeAttempts,
	}
	out.Write = evidence
	out.Process.BackendPID = adapters.backendPID
	switch {
	case gate.Refused:
		evidence.Failure = "gate-refused"
		return out, fmt.Errorf("実行前の段階が未解決のため、Commandを実行しませんでした: %v", gate.Reasons)
	case gate.Err != nil:
		evidence.Failure = "command-failed"
		return out, fmt.Errorf("Commandが失敗しました: %w", gate.Err)
	}
	evidence.ResultMode = string(gate.Invocation.Result.Mode)
	return out, nil
}

func runRead(ctx context.Context, opts options, in environmentInputs, db *sql.DB) (*receipt, error) {
	out := baseReceipt("read", in)
	written, err := readReceiptFile(opts.writeReceipt)
	if err != nil {
		return out, err
	}
	if err := requireOwnReceipt(written, "write"); err != nil || written.Write == nil {
		return out, fmt.Errorf("write receiptが不正です")
	}
	if _, err := observeOwnershipMarker(ctx, db); err != nil {
		return out, err
	}
	commandPresent := !written.Write.GateRefused && written.Write.CommandDispatched && written.Write.Failure == ""
	observed, readErr := readOrder(ctx, db, written.Write.OrderID)
	sameSource := in.revision == written.SourceModelRevision
	status := classifyObservation(commandPresent, readErr, sameSource, readErr == nil && rowMatchesSubmission(observed))

	evidence := &observationEvidence{
		ReadIdentity:          "order_entry.sales.orders@separate-process-read",
		Status:                string(status),
		CommandID:             written.Write.CommandID,
		OrderID:               written.Write.OrderID,
		WriteProcessID:        written.Process.ProcessID,
		WriteBackendPID:       written.Process.BackendPID,
		SourceRevisionMatches: sameSource,
	}
	if readErr != nil {
		evidence.ReadFailure = "read-failed"
	} else {
		evidence.RowStatus = observed.Status
		evidence.TotalAmount = observed.TotalAmount
		evidence.LineCount = observed.LineCount
		evidence.ReadBackendPID = observed.BackendPID
		out.Process.BackendPID = observed.BackendPID
	}
	out.Observation = evidence
	if status != observationObserved {
		return out, fmt.Errorf("観測状態が%sです(observedではありません)", status)
	}
	return out, nil
}

func runCleanup(ctx context.Context, opts options, in environmentInputs, db *sql.DB) (*receipt, error) {
	out := baseReceipt("cleanup", in)
	prepared, err := readReceiptFile(opts.prepareReceipt)
	if err != nil {
		return out, err
	}
	if err := requireOwnReceipt(prepared, "prepare"); err != nil || prepared.Prepare == nil {
		return out, fmt.Errorf("prepare receiptが不正です")
	}
	if err := removeFixture(ctx, db, prepared.Prepare.OwnershipMarker); err != nil {
		return out, err
	}
	cleanup, err := verifyDatabaseRestored(ctx, db)
	out.Cleanup = &cleanup
	return out, err
}
