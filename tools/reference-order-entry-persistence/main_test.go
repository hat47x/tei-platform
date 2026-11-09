package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/database/physical"
	postgresprovider "tei/internal/database/providers/postgres"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func referenceWorld(t *testing.T) (ir.World, *databaseTarget, string) {
	t.Helper()
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	stages, target := preflightWorld(loaded.World)
	if !allResolved(stages) || target == nil {
		t.Fatalf("reference world did not resolve: %+v", stages)
	}
	return loaded.World, target, sourceindex.ModelRevision(index)
}

// fakeStoreは、DBを使わずにdispatchの有無を数える。実PostgreSQLの結果ではない。
type fakeStore struct {
	calls int
	err   error
}

func (f *fakeStore) SubmitOrder(context.Context, int64) (int64, error) {
	f.calls++
	return 4242, f.err
}

func TestWorldPreflightResolvesOrderEntryToPostgres(t *testing.T) {
	_, target, _ := referenceWorld(t)
	if target.Instance.Provider != "postgres" || string(target.Definition) != "sales_core" {
		t.Fatalf("unexpected target: %+v", target)
	}
}

func TestDDLIsDerivedFromPersistenceMapping(t *testing.T) {
	world, target, _ := referenceWorld(t)
	ddl, err := renderCreateDDL(world, target)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(ddl, "\n")
	for _, want := range []string{
		`CREATE TABLE "public"."orders"`, `CREATE TABLE "public"."order_lines"`,
		`"order_id"`, `"total_amount"`, `"unit_price"`, `ADD PRIMARY KEY ("order_id")`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("DDL does not contain %s:\n%s", want, joined)
		}
	}
	// seedの列名は、導出したDDLの列名と一致していなければならない。
	for _, seed := range fixtureSeedSQL {
		if !strings.Contains(seed, "public.order") {
			t.Fatalf("seed does not target the fixture tables: %s", seed)
		}
	}
	// seedとadapterが使う列は、導出したDDLに存在しなければならない。
	for _, column := range []string{"order_id", "customer_id", "ordered_at", "status", "total_amount",
		"order_line_id", "line_number", "item_code", "quantity", "unit_price"} {
		if !strings.Contains(joined, `"`+column+`"`) {
			t.Fatalf("seed column %s is absent from the derived DDL", column)
		}
	}
}

func matchingSnapshot(t *testing.T, world ir.World, target *databaseTarget) physical.SchemaSnapshot {
	t.Helper()
	desired, err := desiredSchema(world, target)
	if err != nil {
		t.Fatal(err)
	}
	return desired
}

func introspectReturning(s physical.SchemaSnapshot) introspectFunc {
	return func(context.Context) (physical.SchemaSnapshot, error) { return s, nil }
}

func TestMatchingPhysicalSchemaResolves(t *testing.T) {
	world, target, _ := referenceWorld(t)
	stage := preflightSchema(context.Background(), world, target, introspectReturning(matchingSnapshot(t, world, target)))
	if stage.Status != stageResolved {
		t.Fatalf("identical schema did not resolve: %+v", stage)
	}
}

// S02(物理側): 列が欠けた物理schemaでは、Command前に拒否し、storeを一度も呼ばない。
func TestPhysicalSchemaMismatchIsRefusedBeforeAnyWrite(t *testing.T) {
	world, target, revision := referenceWorld(t)
	drifted := matchingSnapshot(t, world, target)
	tables := append([]physical.Table(nil), drifted.Tables...)
	for i := range tables {
		if tables[i].Name == "orders" {
			columns := append([]physical.Column(nil), tables[i].Columns...)
			tables[i].Columns = columns[:len(columns)-1]
		}
	}
	drifted.Tables = tables

	stages, _ := preflightWorld(world)
	stages = append(stages, checkExpectedSource(revision, revision))
	stages = append(stages, preflightSchema(context.Background(), world, target, introspectReturning(drifted)))
	store := &fakeStore{}
	adapters := &adapterSet{store: store}
	gate := submitIfReady(context.Background(), driver.NewResolver(world), stages, adapters, fixtureOrderID)
	if !gate.Refused || store.calls != 0 || adapters.invoked != 0 {
		t.Fatalf("physical mismatch reached the store: refused=%v calls=%d invoked=%d", gate.Refused, store.calls, adapters.invoked)
	}
	if findStage(stages, "physical-schema").Status == stageResolved {
		t.Fatal("drifted schema resolved")
	}
	if findStage(stages, "source-revision").Status != stageResolved {
		t.Fatal("the Source stage must be independent of the physical schema stage")
	}
}

// S02(Source側): Source revisionが違えば、物理schemaが一致していても拒否する。理由は別。
func TestSourceRevisionMismatchIsRefusedForItsOwnReason(t *testing.T) {
	world, target, revision := referenceWorld(t)
	stages, _ := preflightWorld(world)
	stages = append(stages, checkExpectedSource("sha256:"+strings.Repeat("0", 64), revision))
	stages = append(stages, preflightSchema(context.Background(), world, target, introspectReturning(matchingSnapshot(t, world, target))))
	store := &fakeStore{}
	adapters := &adapterSet{store: store}
	gate := submitIfReady(context.Background(), driver.NewResolver(world), stages, adapters, fixtureOrderID)
	if !gate.Refused || store.calls != 0 {
		t.Fatalf("Source mismatch reached the store: refused=%v calls=%d", gate.Refused, store.calls)
	}
	if got := findStage(stages, "source-revision"); got.Detail != "SOURCE_REVISION_MISMATCH" {
		t.Fatalf("wrong reason: %+v", got)
	}
	if findStage(stages, "physical-schema").Status != stageResolved {
		t.Fatal("schema stage should be independently resolved")
	}
	if got := checkExpectedSource("", revision); got.Status == stageResolved {
		t.Fatal("a missing expectation must not resolve")
	}
}

func resolvedStages(t *testing.T) (ir.World, []stageResult) {
	t.Helper()
	world, target, revision := referenceWorld(t)
	stages, _ := preflightWorld(world)
	stages = append(stages, checkExpectedSource(revision, revision))
	stages = append(stages, preflightSchema(context.Background(), world, target, introspectReturning(matchingSnapshot(t, world, target))))
	if !allResolved(stages) {
		t.Fatalf("fixture stages unresolved: %+v", stages)
	}
	return world, stages
}

func TestPositivePathWritesExactlyOnceThroughTheDriver(t *testing.T) {
	world, stages := resolvedStages(t)
	store := &fakeStore{}
	adapters := &adapterSet{store: store, grant: func(context.Context) error { return nil }}
	gate := submitIfReady(context.Background(), driver.NewResolver(world), stages, adapters, fixtureOrderID)
	if gate.Refused || gate.Err != nil || gate.Invocation == nil {
		t.Fatalf("gate: %+v", gate)
	}
	if store.calls != 1 || adapters.writeAttempts != 1 || adapters.backendPID != 4242 {
		t.Fatalf("calls=%d attempts=%d pid=%d", store.calls, adapters.writeAttempts, adapters.backendPID)
	}
}

// S05: 書込み直前に現在の許可が失われていれば、storeを呼ばない(dispatch 0)。
func TestRevokedEffectGrantDispatchesNoWrite(t *testing.T) {
	world, stages := resolvedStages(t)
	store := &fakeStore{}
	adapters := &adapterSet{store: store, grant: func(context.Context) error { return errors.New("revoked: secret-policy-detail") }}
	gate := submitIfReady(context.Background(), driver.NewResolver(world), stages, adapters, fixtureOrderID)
	if gate.Err == nil || gate.Invocation != nil {
		t.Fatalf("a revoked grant must be an execution error, not a result: %+v", gate)
	}
	if store.calls != 0 || adapters.writeAttempts != 0 {
		t.Fatalf("the store was written without a current grant: calls=%d", store.calls)
	}
	if strings.Contains(gate.Err.Error(), "secret-policy-detail") {
		t.Fatalf("grant detail leaked: %v", gate.Err)
	}
}

func TestStoreFailureIsAnExecutionErrorNotABusinessResult(t *testing.T) {
	world, stages := resolvedStages(t)
	store := &fakeStore{err: errors.New("connection reset")}
	adapters := &adapterSet{store: store}
	gate := submitIfReady(context.Background(), driver.NewResolver(world), stages, adapters, fixtureOrderID)
	if gate.Err == nil || gate.Invocation != nil {
		t.Fatalf("store failure became a result: %+v", gate)
	}
}

// N11: Commandが成功しても、別の読取りが失敗・不一致・別Sourceなら、observedにしない。
func TestObservationIsNeverObservedWithoutAMatchingIndependentRead(t *testing.T) {
	good := orderObservation{Status: "submitted", TotalAmount: fixtureTotalAmount, LineCount: fixtureLineCount}
	cases := []struct {
		name          string
		commandResult bool
		readErr       error
		sameSource    bool
		row           orderObservation
		want          observationStatus
	}{
		{"observed", true, nil, true, good, observationObserved},
		{"command not committed", false, nil, true, good, observationCommandNotCommitted},
		{"read failed", true, errors.New("closed pool"), true, good, observationReadFailed},
		{"row still pending", true, nil, true, orderObservation{Status: "pending", TotalAmount: fixtureTotalAmount, LineCount: fixtureLineCount}, observationMismatch},
		{"lines missing", true, nil, true, orderObservation{Status: "submitted", TotalAmount: fixtureTotalAmount, LineCount: 1}, observationMismatch},
		{"other source", true, nil, false, good, observationSourceRevisionMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyObservation(tc.commandResult, tc.readErr, tc.sameSource, tc.readErr == nil && rowMatchesSubmission(tc.row))
			if got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}

func goodReceipts() (receipt, receipt) {
	write := receipt{
		Schema: receiptSchema, Mode: "write", EvidenceNature: evidenceNature, Application: "order_entry",
		Repository:          repositoryState{Revision: "abc", Clean: true},
		SourceModelRevision: "sha256:src", Scope: fixedScope(), Process: processInfo{ProcessID: 100, BackendPID: 11},
		Write: &writeEvidence{CommandID: "cmd-1", OrderID: fixtureOrderID, CommandDispatched: true, WriteAttempts: 1},
	}
	read := receipt{
		Schema: receiptSchema, Mode: "read", EvidenceNature: evidenceNature, Application: "order_entry",
		Repository:          repositoryState{Revision: "abc", Clean: true},
		SourceModelRevision: "sha256:src", Scope: fixedScope(), Process: processInfo{ProcessID: 200, BackendPID: 22},
		Observation: &observationEvidence{CommandID: "cmd-1", OrderID: fixtureOrderID, Status: string(observationObserved)},
	}
	return write, read
}

func TestJoinAcceptsOnlyOneAttemptAcrossSeparateProcesses(t *testing.T) {
	write, read := goodReceipts()
	if problems := verifyJoin(write, read); len(problems) != 0 {
		t.Fatalf("a consistent pair must join: %v", problems)
	}
	mutations := map[string]func(*receipt, *receipt){
		"same process":       func(w, r *receipt) { r.Process.ProcessID = w.Process.ProcessID },
		"same DB connection": func(w, r *receipt) { r.Process.BackendPID = w.Process.BackendPID },
		"other command":      func(w, r *receipt) { r.Observation.CommandID = "cmd-2" },
		"other order":        func(w, r *receipt) { r.Observation.OrderID = 1002 },
		"other source":       func(w, r *receipt) { r.SourceModelRevision = "sha256:other" },
		"other repository":   func(w, r *receipt) { r.Repository.Revision = "def" },
		"dirty repository":   func(w, r *receipt) { w.Repository.Clean = false },
		"not observed":       func(w, r *receipt) { r.Observation.Status = string(observationMismatch) },
		"command not run":    func(w, r *receipt) { w.Write.CommandDispatched = false },
		"gate refused":       func(w, r *receipt) { w.Write.GateRefused = true },
		"hand-written":       func(w, r *receipt) { w.Scope.HandWrittenOverlay = true },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			w, r := goodReceipts()
			mutate(&w, &r)
			if problems := verifyJoin(w, r); len(problems) == 0 {
				t.Fatal("joined an inconsistent pair")
			}
		})
	}
}

// 手書きoverlay(別Application request_log)のE2a-1 receiptは、この証拠と結合しない。
func TestJoinRefusesTheHandWrittenOverlayReceipt(t *testing.T) {
	overlay := []byte(`{
  "evidenceNature": "real-postgres-ephemeral-fixture",
  "repositoryRevision": "abc", "workingTreeClean": true,
  "scope": {"productE2": false, "handWrittenOverlay": true, "generatedRoundTrip": false},
  "positive": {"commandId": "cmd-1", "observationStatus": "observed", "databaseProvider": "postgres"}
}`)
	var handWritten receipt
	if err := json.Unmarshal(overlay, &handWritten); err != nil {
		t.Fatal(err)
	}
	write, read := goodReceipts()
	for name, pair := range map[string][2]receipt{"as write": {handWritten, read}, "as read": {write, handWritten}} {
		problems := verifyJoin(pair[0], pair[1])
		if len(problems) == 0 {
			t.Fatalf("%s: the hand-written overlay receipt joined", name)
		}
		joined := strings.Join(problems, "|")
		if !strings.Contains(joined, "schema") || !strings.Contains(joined, "application") {
			t.Fatalf("%s: refusal does not name the schema/application mismatch: %v", name, problems)
		}
	}
}

func TestDedicatedDatabaseAndOwnershipGuards(t *testing.T) {
	for _, name := range []string{"", "postgres", "template0", "template1"} {
		if validateDedicatedFixtureDatabase(name, 0, 0) == nil {
			t.Fatalf("database %q accepted", name)
		}
	}
	if validateDedicatedFixtureDatabase("tei_fixture", 1, 0) == nil || validateDedicatedFixtureDatabase("tei_fixture", 0, 1) == nil {
		t.Fatal("a non-empty database was accepted")
	}
	if err := validateDedicatedFixtureDatabase("tei_fixture", 0, 0); err != nil {
		t.Fatal(err)
	}
	marker, err := newOwnershipMarker()
	if err != nil || validateOwnershipMarker(marker) != nil {
		t.Fatalf("generated marker invalid: %v", err)
	}
	for _, bad := range []string{"", "x", ownershipPrefix, ownershipPrefix + strings.Repeat("A", 32), ownershipPrefix + strings.Repeat("g", 32),
		ownershipPrefix + strings.Repeat("a", 31), ownershipPrefix + strings.Repeat("a", 32) + "'; DROP TABLE x;--",
		"tei:reference-data-first-persistence:" + strings.Repeat("a", 32)} {
		if validateOwnershipMarker(bad) == nil {
			t.Fatalf("marker %q accepted", bad)
		}
	}
}

func TestDSNInputIsExclusiveAndRedacted(t *testing.T) {
	lookup := func(values map[string]string) dsnLookupEnvFunc {
		return func(k string) (string, bool) { v, ok := values[k]; return v, ok }
	}
	read := func(string) ([]byte, error) { return []byte("postgres://user:pw@host/db\n"), nil }
	if _, err := loadPostgresDSNInput(lookup(map[string]string{postgresDSNEnv: "a", postgresDSNFileEnv: "b"}), read); err == nil {
		t.Fatal("both inputs accepted")
	}
	if _, err := loadPostgresDSNInput(lookup(nil), read); err == nil {
		t.Fatal("no input accepted")
	}
	got, err := loadPostgresDSNInput(lookup(map[string]string{postgresDSNFileEnv: "f"}), read)
	if err != nil || got != "postgres://user:pw@host/db" {
		t.Fatalf("file input: %q %v", got, err)
	}
	failing := func(string) ([]byte, error) { return nil, errors.New("open /secret/path: denied") }
	if _, err := loadPostgresDSNInput(lookup(map[string]string{postgresDSNFileEnv: "f"}), failing); err == nil || strings.Contains(err.Error(), "/secret/path") {
		t.Fatalf("path leaked: %v", err)
	}
}

func TestRunRefusesUnknownModeAndDirtyOrMissingInputs(t *testing.T) {
	if _, err := run([]string{"-mode", "nope"}, os.Stdout); err == nil {
		t.Fatal("unknown mode accepted")
	}
	if _, err := run([]string{"-mode", "write", "extra"}, os.Stdout); err == nil {
		t.Fatal("positional argument accepted")
	}
	t.Setenv(postgresDSNEnv, "")
	t.Setenv(postgresDSNFileEnv, "")
	os.Unsetenv(postgresDSNEnv)
	os.Unsetenv(postgresDSNFileEnv)
	if _, err := run([]string{"-mode", "prepare"}, os.Stdout); err == nil {
		t.Fatal("prepare ran without a DSN")
	}
}

// ---------------------------------------------------------------------------
// 実PostgreSQL(DSNがあるときだけ)。ローカルでDSNが無ければ実行しない(NOT_RUN)。
// CIでは専用の空DBに対して実行する。
// ---------------------------------------------------------------------------

func TestRealPostgresRoundTrip(t *testing.T) {
	dsn, set := os.LookupEnv(postgresDSNEnv)
	if !set || dsn == "" {
		t.Skip("NOT_RUN: TEI_TEST_POSTGRES_DSN is not set; real PostgreSQL evidence was not acquired by this run")
	}
	ctx := context.Background()
	world, target, _ := referenceWorld(t)
	writer, err := openDB(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	reader, err := openDB(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if _, err := ensureDedicatedEmptyDatabase(ctx, writer); err != nil {
		t.Fatal(err)
	}
	ddl, err := renderCreateDDL(world, target)
	if err != nil {
		t.Fatal(err)
	}
	marker, err := createFixture(ctx, writer, ddl)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := removeFixture(ctx, writer, marker); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}()
	introspect := func(c context.Context) (physical.SchemaSnapshot, error) {
		return postgresprovider.Provider{Catalog: postgresprovider.SQLCatalogReader{DB: writer}}.Introspect(c, target.Instance)
	}
	if stage := preflightSchema(ctx, world, target, introspect); stage.Status != stageResolved {
		t.Fatalf("DDL derived from the Persistence Mapping does not match the observed schema: %+v", stage)
	}

	// 許可が失われている間は、行が変わらない(S05)。
	denied := &adapterSet{store: pgStore{db: writer}, grant: func(context.Context) error { return errors.New("revoked") }}
	stages, _ := preflightWorld(world)
	stages = append(stages, checkExpectedSource("x", "x"), preflightSchema(ctx, world, target, introspect))
	gate := submitIfReady(ctx, driver.NewResolver(world), stages, denied, fixtureOrderID)
	if gate.Err == nil || denied.writeAttempts != 0 {
		t.Fatalf("revoked grant wrote: %+v attempts=%d", gate, denied.writeAttempts)
	}
	before, err := readOrder(ctx, reader, fixtureOrderID)
	if err != nil || before.Status != "pending" {
		t.Fatalf("row changed without a grant: %+v %v", before, err)
	}

	adapters := &adapterSet{store: pgStore{db: writer}, grant: func(context.Context) error { return nil }}
	gate = submitIfReady(ctx, driver.NewResolver(world), stages, adapters, fixtureOrderID)
	if gate.Refused || gate.Err != nil {
		t.Fatalf("command failed: %+v", gate)
	}
	observed, err := readOrder(ctx, reader, fixtureOrderID)
	if err != nil || !rowMatchesSubmission(observed) {
		t.Fatalf("independent read does not show the submitted order: %+v %v", observed, err)
	}
	if observed.BackendPID == adapters.backendPID {
		t.Fatal("the read used the command's DB connection")
	}
	// 二回目のsubmitは、pendingの行が無いので失敗し、成功扱いにしない。
	again := &adapterSet{store: pgStore{db: writer}}
	if g := submitIfReady(ctx, driver.NewResolver(world), stages, again, fixtureOrderID); g.Err == nil {
		t.Fatal("a second submit of the same order reported success")
	}
}
