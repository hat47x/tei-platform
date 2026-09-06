package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
	"tei/internal/validation"
	"tei/internal/workspace"
	"tei/tools/internal/repositoryevidence"
)

const fixtureObject = "public.order_lines"
const evidenceNature = "real-postgres-ephemeral-fixture"

type evidence struct {
	EvidenceNature                        string                        `json:"evidenceNature"`
	RepositoryRevision                    string                        `json:"repositoryRevision"`
	WorkingTreeClean                      bool                          `json:"workingTreeClean"`
	ReproducibleRevision                  bool                          `json:"reproducibleRevision"`
	PostRunRepository                     repositoryevidence.PostRun    `json:"postRunRepository"`
	FixtureObject                         string                        `json:"fixtureObject"`
	PreexistingFixtureProtected           bool                          `json:"preexistingFixtureProtected"`
	FixtureOwnedByTool                    bool                          `json:"fixtureOwnedByTool"`
	FixtureDatabaseSafety                 fixtureDatabaseSafetyEvidence `json:"fixtureDatabaseSafety"`
	FixtureCleanupSafety                  fixtureCleanupSafetyEvidence  `json:"fixtureCleanupSafety"`
	Pattern                               core.CanonicalID              `json:"pattern"`
	Relation                              core.CanonicalID              `json:"relation"`
	Environment                           core.CanonicalID              `json:"environment"`
	DatabaseInstance                      core.CanonicalID              `json:"databaseInstance"`
	ExecutorCapability                    core.CanonicalID              `json:"executorCapability"`
	ContractQueryTarget                   string                        `json:"contractQueryTarget"`
	ParameterNames                        []string                      `json:"parameterNames"`
	ParameterValuesPersisted              bool                          `json:"parameterValuesPersisted"`
	Rows                                  int                           `json:"rows"`
	ZeroRows                              int                           `json:"zeroRows"`
	ExecutionErrorObserved                bool                          `json:"executionErrorObserved"`
	SemanticAttributes                    []core.CanonicalID            `json:"semanticAttributes"`
	RelationStorageColumn                 string                        `json:"relationStorageColumn"`
	SemanticOrderLineParentAttributeAdded bool                          `json:"semanticOrderLineParentAttributeAdded"`
	SemanticMappingApplied                bool                          `json:"semanticMappingApplied"`
}

func main() {
	// DSN本文をGo valueへ取り込んだ後、source probeや後続の不要なsubprocessへ
	// credential候補 / secret file pathを継承しない。file入力なら#308自身のcompile段階にもDSN本文は存在しない。
	dsn, err := loadPostgresDSNFromEnvironment()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := clearPostgresDSNEnvironment(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// 実Provider Evidenceは、後から同じsourceへ戻れるclean revisionに固定する。
	// dirty treeの診断実行をGate B完了Evidenceへ格上げしない。
	repositorySource, err := repositoryevidence.InspectStart()
	if err != nil {
		panic(err)
	}
	if !repositorySource.WorkingTreeClean {
		panic("PostgreSQL Provider Evidenceはclean revisionに固定するため、working treeに未commit差分がある状態ではDatabaseへ接続しません")
	}

	// Databaseへ接続・変更する前に、Evidenceの前提となるCanonical Workspace自体を検証する。
	// 無効なauthoringを前提にfixtureを作成してから失敗する順序を許さない。
	loaded, err := workspace.Load("reference/workspace")
	if err != nil {
		panic(err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity != validation.Error {
			continue
		}
		panic(fmt.Errorf(
			"Reference WorkspaceにValidation ErrorがあるためPostgreSQL Evidenceを実行しません: %s: %s",
			diagnostic.Code,
			diagnostic.Message,
		))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(postgresConnectionError("PostgreSQL接続設定の初期化", err))
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		panic(postgresConnectionError("PostgreSQL接続確認", err))
	}

	// #308は「専用の空PostgreSQL Database」を前提とする。
	// public.order_linesだけが空いている既存業務DBをEvidence fixture先として受け入れない。
	fixtureDatabaseSafety, err := ensureDedicatedEmptyFixtureDatabase(ctx, db)
	if err != nil {
		panic(err)
	}

	// CREATEとDB側ownership marker付与を同一transactionで確定する。
	// Go側booleanだけを根拠に後から同名objectをDROPしない。
	fixtureOwnershipMarker, err := createOwnedFixtureTable(ctx, db)
	if err != nil {
		panic(err)
	}
	fixtureOwned := true
	defer func() {
		if !fixtureOwned {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := removeOwnedFixtureTable(cleanupCtx, db, fixtureOwnershipMarker); err != nil {
			fmt.Fprintf(os.Stderr, "header-detail fixture cleanup warning: %v\n", err)
		}
	}()
	if err := insertFixtureRows(ctx, db); err != nil {
		panic(err)
	}

	resolver := driver.NewResolver(loaded.World)

	contractExecutor := sqlquery.Executor{
		Provider:     environment.ProviderPostgres,
		Realizations: loaded.World.ContractQuerySQLRealizations,
		Runner:       postgresquery.Runner{DB: db},
	}
	registry := driver.ContractQueryExecutorMap{
		{
			Environment: core.MustCanonicalID("development"),
			Instance:    core.MustCanonicalID("sales_core.dev"),
			Capability:  core.MustCanonicalID("database.query.postgres"),
		}: contractExecutor,
	}

	source := mappedOrderID(int64(1001))
	executed, err := resolver.ExecuteHeaderDetailContractQueryInEnvironment(
		ctx,
		core.MustCanonicalID("development"),
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.order_lines"),
		source,
		0,
		registry,
	)
	if err != nil {
		panic(err)
	}
	if len(executed.Execution.Result.Rows) != 2 {
		panic(fmt.Errorf("order_id=1001のOrderLine件数=%d want=2", len(executed.Execution.Result.Rows)))
	}

	wantIDs := []int64{1101, 1102}
	for i, row := range executed.Execution.Result.Rows {
		if got := rawInt64(row.Attributes["sales.order_line.order_line_id"]); got != wantIDs[i] {
			panic(fmt.Errorf("OrderLine[%d].order_line_id=%d want=%d", i, got, wantIDs[i]))
		}
		if _, exists := row.Attributes["sales.order_line.order_id"]; exists {
			panic("物理support column order_idをOrderLineのSemantic attributeとして返しています")
		}
	}

	payload, ok := executed.Execution.Result.Payload.(postgresquery.Result)
	if !ok {
		panic(fmt.Errorf("PostgreSQL Contract Query Executorが想定外のpayloadを返しました: %T", executed.Execution.Result.Payload))
	}
	if len(payload.ParameterNames) != 1 || payload.ParameterNames[0] != "order_id" {
		panic(fmt.Errorf("parameterNames=%#v want=[order_id]", payload.ParameterNames))
	}

	zeroExecuted, err := resolver.ExecuteHeaderDetailContractQueryInEnvironment(
		ctx,
		core.MustCanonicalID("development"),
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.order_lines"),
		mappedOrderID(int64(9999)),
		0,
		registry,
	)
	if err != nil {
		panic(fmt.Errorf("0件のheader-detail取得をExecution Errorへ変換しています: %w", err))
	}
	if len(zeroExecuted.Execution.Result.Rows) != 0 {
		panic(fmt.Errorf("order_id=9999のOrderLine件数=%d want=0", len(zeroExecuted.Execution.Result.Rows)))
	}

	// Execution Errorの実Provider Evidenceを取るため、ここでtool自身が作成したfixtureを削除する。
	// 削除transaction内でACCESS EXCLUSIVE lockを取得し、DB側ownership markerが作成時と一致する場合だけDROPする。
	if err := removeOwnedFixtureTable(ctx, db, fixtureOwnershipMarker); err != nil {
		panic(err)
	}
	fixtureOwned = false
	if _, err := resolver.ExecuteHeaderDetailContractQueryInEnvironment(
		ctx,
		core.MustCanonicalID("development"),
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.order_lines"),
		mappedOrderID(int64(1001)),
		0,
		registry,
	); err == nil {
		panic("物理table欠落によるPostgreSQL実行失敗を正常な空collectionとして扱っています")
	}

	// missing-table Error確認まで終えた後、接続先が開始時と同じ空状態へ戻ったことを明示確認する。
	// tool-owned fixture以外のrelationが残る、または並行変更でrelationが増えたrunは成功Evidenceへ進めない。
	fixtureCleanupSafety, err := verifyFixtureDatabaseRestored(ctx, db)
	if err != nil {
		panic(err)
	}

	semanticAttributes := semanticAttributeIDs(executed.Execution.Result.Rows)
	relationStorageColumn, parentAttributeAdded := inspectOrderLineStorage(loaded)

	out := evidence{
		EvidenceNature:                        evidenceNature,
		RepositoryRevision:                    repositorySource.RepositoryRevision,
		WorkingTreeClean:                      repositorySource.WorkingTreeClean,
		FixtureObject:                         fixtureObject,
		PreexistingFixtureProtected:           true,
		FixtureOwnedByTool:                    true,
		FixtureDatabaseSafety:                 fixtureDatabaseSafety,
		FixtureCleanupSafety:                  fixtureCleanupSafety,
		Pattern:                               core.MustCanonicalID("order_entry.order_lines"),
		Relation:                              core.MustCanonicalID("sales.order.lines"),
		Environment:                           executed.Environment.ID,
		DatabaseInstance:                      executed.Executor.Instance.ID,
		ExecutorCapability:                    executed.Executor.Capability.Capability.ID,
		ContractQueryTarget:                   executed.Execution.Resolution.Binding.Target.Identity(),
		ParameterNames:                        append([]string(nil), payload.ParameterNames...),
		ParameterValuesPersisted:              false,
		Rows:                                  len(executed.Execution.Result.Rows),
		ZeroRows:                              len(zeroExecuted.Execution.Result.Rows),
		ExecutionErrorObserved:                true,
		SemanticAttributes:                    semanticAttributes,
		RelationStorageColumn:                 relationStorageColumn,
		SemanticOrderLineParentAttributeAdded: parentAttributeAdded,
		SemanticMappingApplied:                true,
	}

	postRun, err := repositoryevidence.InspectPost(repositorySource.RepositoryRevision)
	if err != nil {
		panic(err)
	}
	out.PostRunRepository = postRun
	out.ReproducibleRevision = repositoryevidence.IsReproducible(repositorySource, postRun)
	if !postRun.RevisionStable {
		panic("PostgreSQL Evidence実行中にrepository HEADが変化したため、単一revisionのProvider Evidenceとして採用できません")
	}
	if !postRun.WorkingTreeClean {
		panic("PostgreSQL Evidence実行後にworking tree差分が残ったため、記録revisionと実行sourceの同一性を保証できません")
	}
	if !out.ReproducibleRevision {
		panic("PostgreSQL Provider Evidenceを単一のclean revisionへ固定できません")
	}

	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(encoded))
}

func mappedOrderID(orderID int64) queryresult.MappedResult {
	return queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "order_id", Attribute: core.MustCanonicalID("sales.order.order_id")},
		},
		Rows: [][]any{{orderID}},
	}
}

// ensureFixtureObjectAbsentは、Evidence toolが既存のpublic.order_linesを破壊しないためのfail-closed guardである。
// 既存objectが一つでもあれば、内容や用途を推測せず実行を拒否する。
func ensureFixtureObjectAbsent(ctx context.Context, db *sql.DB) error {
	var object sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT to_regclass('public.order_lines')::text`).Scan(&object); err != nil {
		return fmt.Errorf("header-detail fixture objectの既存有無を確認できません: %w", err)
	}
	if object.Valid && strings.TrimSpace(object.String) != "" {
		return fmt.Errorf(
			"安全のためEvidence toolを停止します: %sが既に存在します。既存tableをDROPせず、専用の空PostgreSQL DatabaseをDSN入力へ指定してください",
			object.String,
		)
	}
	return nil
}

func insertFixtureRows(ctx context.Context, db *sql.DB) error {
	rows := []struct {
		orderLineID int64
		orderID     int64
		lineNumber  int64
		itemCode    string
		quantity    string
		unitPrice   string
	}{
		{1101, 1001, 1, "SKU-001", "2.00", "1200.00"},
		{1102, 1001, 2, "SKU-002", "1.00", "800.00"},
		{2101, 2000, 1, "SKU-003", "3.00", "500.00"},
	}
	for _, row := range rows {
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO public.order_lines (order_line_id, order_id, line_number, item_code, quantity, unit_price) VALUES ($1, $2, $3, $4, $5, $6)`,
			row.orderLineID,
			row.orderID,
			row.lineNumber,
			row.itemCode,
			row.quantity,
			row.unitPrice,
		); err != nil {
			return fmt.Errorf("header-detail用fixture rowを追加できません: %w", err)
		}
	}
	return nil
}

func semanticAttributeIDs(rows []driver.ContractQueryRow) []core.CanonicalID {
	seen := map[core.CanonicalID]struct{}{}
	for _, row := range rows {
		for attribute := range row.Attributes {
			seen[attribute] = struct{}{}
		}
	}
	out := make([]core.CanonicalID, 0, len(seen))
	for attribute := range seen {
		out = append(out, attribute)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func inspectOrderLineStorage(loaded workspace.Result) (string, bool) {
	for _, mapping := range loaded.World.Persistence {
		if mapping.ID != "sales_core.order_line" {
			continue
		}
		for _, column := range mapping.Columns {
			if column.Attribute == "sales.order_line.order_id" {
				return "", true
			}
		}
		storage, ok := mapping.RelationStorageByRelation("sales.order.lines")
		if !ok || len(storage.Components) != 1 {
			panic(fmt.Errorf("sales.order.linesのRelation Storageが一意に解決できません: %#v", mapping.RelationStorage))
		}
		return storage.Components[0].Column, false
	}
	panic("sales_core.order_line Persistence Mappingがありません")
}

func rawInt64(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	default:
		panic(fmt.Errorf("raw integer valueの型が想定外です: %T (%v)", value, value))
	}
}
