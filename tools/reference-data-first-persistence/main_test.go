package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	dbrealization "tei/internal/database/realization"
	"tei/internal/database/schemadiff"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/workspace"
)

// このテストはDBを必要としない純粋処理とmock処理だけを検査する。
// 実PostgreSQLを使う証明はgo runが出力するreceiptであり、このテストの成功では代替しない。

const repoRootFromPackage = "../.."

func loadPrepared(t *testing.T, omit ...string) workspace.Result {
	t.Helper()
	prepared, err := prepareWorkspace(repoRootFromPackage, filepath.Join(t.TempDir(), "ws"), omit...)
	if err != nil {
		t.Fatalf("prepareWorkspace: %v", err)
	}
	if !prepared.RequestTeiUnchanged {
		t.Fatalf("overlayがrequest.teiを変更しています")
	}
	loaded, err := workspace.Load(prepared.Root)
	if err != nil {
		t.Fatalf("workspace.Load: %v", err)
	}
	return loaded
}

func TestOverlayLoadsWithoutValidationErrors(t *testing.T) {
	loaded := loadPrepared(t)
	if n, codes := countValidationErrors(loaded.World); n != 0 {
		t.Fatalf("Validation Errorが%d件あります: %v", n, codes)
	}
}

func TestPreflightWorldResolvesDeclaredBindingsFromEnvironment(t *testing.T) {
	loaded := loadPrepared(t)
	stages, target := preflightWorld(loaded.World)
	if !allResolved(stages) {
		t.Fatalf("世界の段階が解決されていません: %+v", stages)
	}
	if target == nil || target.Definition != "request_store" || target.Instance.Provider != environment.ProviderPostgres {
		t.Fatalf("environmentのbindingから導出したDatabase識別が想定と異なります: %+v", target)
	}
}

func TestPreflightWorldWithoutRealizeRefusesAndDoesNotDispatch(t *testing.T) {
	loaded := loadPrepared(t, realizeFile)
	stages, _ := preflightWorld(loaded.World)
	if got := findStage(stages, "command-pattern-binding"); got.Status != stageUnresolved {
		t.Fatalf("realizeがないのにBindingが解決済みです: %+v", got)
	}

	dispatched := 0
	adapters := &adapterSet{insert: func(context.Context, submission) (int64, error) { dispatched++; return 1, nil }}
	resolver := driver.NewResolver(loaded.World)

	gate := submitIfReady(context.Background(), resolver, append(stages, stageResult{Stage: "physical-schema", Status: stageResolved}), adapters, submission{})
	if !gate.Refused || dispatched != 0 || adapters.dispatched != 0 {
		t.Fatalf("実行ゲートが拒否していません: %+v dispatched=%d", gate, dispatched)
	}
	if _, err := resolver.InvokeCapability(context.Background(), applicationID, capabilityID, adapters, submission{}); err == nil {
		t.Fatal("Driverがrealizeのない呼出しを拒否していません")
	}
	if dispatched != 0 {
		t.Fatalf("Driverの拒否後にadapterが呼ばれました: %d", dispatched)
	}
}

func TestSubmitIfReadyDispatchesOnlyWhenEveryStageIsResolved(t *testing.T) {
	loaded := loadPrepared(t)
	resolver := driver.NewResolver(loaded.World)
	worldStages, _ := preflightWorld(loaded.World)
	resolvedSchema := stageResult{Stage: "physical-schema", Status: stageResolved}

	cases := []struct {
		name         string
		schema       stageResult
		wantDispatch int
	}{
		{"解決済み", resolvedSchema, 1},
		{"未解決", stageResult{Stage: "physical-schema", Status: stageUnresolved, Detail: "create-table"}, 0},
		{"未対応", stageResult{Stage: "physical-schema", Status: stageUnsupported}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotInput submission
			adapters := &adapterSet{insert: func(_ context.Context, s submission) (int64, error) { gotInput = s; return 42, nil }}
			in := submission{CommandID: "cmd-1", RequestID: "req-1", Summary: "s", Status: "received"}
			gate := submitIfReady(context.Background(), resolver, append(append([]stageResult(nil), worldStages...), tc.schema), adapters, in)
			if adapters.dispatched != tc.wantDispatch {
				t.Fatalf("dispatched=%d want=%d", adapters.dispatched, tc.wantDispatch)
			}
			if tc.wantDispatch == 0 {
				if !gate.Refused || gate.Invocation != nil {
					t.Fatalf("拒否されていません: %+v", gate)
				}
				return
			}
			if gate.Refused || gate.Err != nil || gate.Invocation == nil {
				t.Fatalf("実行できていません: %+v", gate)
			}
			if gate.Invocation.Result.Mode != capabilityinvoke.ResultOutput || gate.Invocation.Binding.Binding.Identity() != bindingIdentity {
				t.Fatalf("結果のモードまたはBindingが想定と異なります: %+v", gate.Invocation)
			}
			if gotInput != in {
				t.Fatalf("adapterへ渡った入力が異なります: %+v", gotInput)
			}
		})
	}
}

func TestSubmitIfReadyReportsAdapterFailureAsExecutionError(t *testing.T) {
	loaded := loadPrepared(t)
	resolver := driver.NewResolver(loaded.World)
	worldStages, _ := preflightWorld(loaded.World)
	adapters := &adapterSet{insert: func(context.Context, submission) (int64, error) { return 0, errors.New("duplicate key") }}
	gate := submitIfReady(context.Background(), resolver, append(worldStages, stageResult{Stage: "physical-schema", Status: stageResolved}), adapters, submission{})
	if gate.Err == nil || gate.Invocation != nil {
		t.Fatalf("DB障害を実行失敗として返していません: %+v", gate)
	}
}

func TestAdapterSetRejectsUnexpectedBinding(t *testing.T) {
	adapters := &adapterSet{insert: func(context.Context, submission) (int64, error) { return 1, nil }}
	_, err := adapters.ResolveAdapter(context.Background(), realization.CustomGoBinding{Application: "other_app", Capability: capabilityID})
	if err == nil {
		t.Fatal("想定外のBindingにadapterを返しました")
	}
}

func desiredSnapshot(t *testing.T, world ir.World, target *databaseTarget) physical.SchemaSnapshot {
	t.Helper()
	desired, err := dbrealization.RealizeDesired(world.Persistence, string(target.Definition), target.Instance.Provider)
	if err != nil {
		t.Fatalf("RealizeDesired: %v", err)
	}
	return desired
}

func TestPreflightSchemaClassifiesObservedSchema(t *testing.T) {
	loaded := loadPrepared(t)
	_, target := preflightWorld(loaded.World)
	desired := desiredSnapshot(t, loaded.World, target)
	if len(desired.Tables) != 1 || desired.Tables[0].Schema != "public" || desired.Tables[0].Name != "requests" {
		t.Fatalf("導出された物理schemaが想定と異なります: %+v", desired.Tables)
	}

	withoutColumn := desired
	withoutColumn.Tables = append([]physical.Table(nil), desired.Tables...)
	withoutColumn.Tables[0].Columns = withoutColumn.Tables[0].Columns[:2]

	cases := []struct {
		name       string
		introspect introspectFunc
		want       stageStatus
		wantDetail string
	}{
		{"一致", func(context.Context) (physical.SchemaSnapshot, error) { return desired, nil }, stageResolved, ""},
		{"空のDatabase", func(context.Context) (physical.SchemaSnapshot, error) {
			return physical.SchemaSnapshot{Database: desired.Database}, nil
		}, stageUnresolved, "create-table"},
		{"列が欠けている", func(context.Context) (physical.SchemaSnapshot, error) { return withoutColumn, nil }, stageUnresolved, "add-column"},
		{"観測失敗", func(context.Context) (physical.SchemaSnapshot, error) {
			return physical.SchemaSnapshot{}, errors.New("connection refused")
		}, stageUnresolved, "観測できません"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := preflightSchema(context.Background(), loaded.World, target, tc.introspect)
			if got.Status != tc.want {
				t.Fatalf("status=%s want=%s detail=%s", got.Status, tc.want, got.Detail)
			}
			if tc.wantDetail != "" && !strings.Contains(got.Detail, tc.wantDetail) {
				t.Fatalf("detail=%q に%qが含まれません", got.Detail, tc.wantDetail)
			}
		})
	}
}

func TestPreflightSchemaWithoutTargetIsUnresolved(t *testing.T) {
	got, _ := preflightSchema(context.Background(), ir.World{}, nil, func(context.Context) (physical.SchemaSnapshot, error) {
		t.Fatal("Database識別がないのに観測しました")
		return physical.SchemaSnapshot{}, nil
	})
	if got.Status != stageUnresolved {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestUnsupportedProviderIsReportedAsUnsupported(t *testing.T) {
	loaded := loadPrepared(t)
	world := loaded.World
	world.Environments = append([]environment.Environment(nil), world.Environments...)
	for i := range world.Environments {
		world.Environments[i].Instances = append([]environment.DatabaseInstance(nil), world.Environments[i].Instances...)
		for j := range world.Environments[i].Instances {
			world.Environments[i].Instances[j].Provider = environment.ProviderOracle
		}
	}
	stages, target := preflightWorld(world)
	if got := findStage(stages, "environment-database-binding"); got.Status != stageUnsupported {
		t.Fatalf("postgres以外のproviderが未対応になっていません: %+v", got)
	}
	if target != nil {
		t.Fatalf("未対応なのにDatabase識別を返しました: %+v", target)
	}
}

func TestClassifyDiffDistinguishesChangesAndUnresolvedComparisons(t *testing.T) {
	if got := classifyDiff(schemadiff.Diff{}); got.Status != stageResolved {
		t.Fatalf("差分なしが解決済みになりません: %+v", got)
	}
	if got := classifyDiff(schemadiff.Diff{Unresolved: []schemadiff.UnresolvedComparison{{Table: "public.requests", ObjectName: "idx"}}}); got.Status != stageUnresolved {
		t.Fatalf("未確定の比較を無視しています: %+v", got)
	}
	changed := classifyDiff(schemadiff.Diff{Changes: []schemadiff.Change{{Kind: schemadiff.AddColumn, Table: "public.requests", Column: &physical.Column{Name: "summary"}}}})
	if changed.Status != stageUnresolved || !strings.Contains(changed.Detail, "add-column public.requests.summary") {
		t.Fatalf("変更の要約が想定と異なります: %+v", changed)
	}
}

func TestClassifyObservationNeverReportsObservedWhenReadFails(t *testing.T) {
	cases := []struct {
		name      string
		committed bool
		readErr   error
		matches   bool
		want      observationStatus
	}{
		{"一致", true, nil, true, observationObserved},
		{"読取り失敗", true, errors.New("relation does not exist"), false, observationReadFailed},
		{"読取り失敗かつ一致扱い", true, errors.New("closed"), true, observationReadFailed},
		{"不一致", true, nil, false, observationMismatch},
		{"Commandが結果を返していない", false, nil, true, observationCommandNotCommitted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyObservation(tc.committed, tc.readErr, tc.matches); got != tc.want {
				t.Fatalf("got=%s want=%s", got, tc.want)
			}
		})
	}
}

func mappedFor(rows ...[]any) queryresult.MappedResult {
	return queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "summary", Attribute: core.MustCanonicalID("intake.request.summary")},
			{Raw: "request_id", Attribute: core.MustCanonicalID("intake.request.request_id")},
			{Raw: "status", Attribute: core.MustCanonicalID("intake.request.status")},
		},
		Rows: rows,
	}
}

func TestRowMatchesSubmissionUsesSemanticAttributesNotColumnOrder(t *testing.T) {
	s := submission{RequestID: "req-1", Summary: "hello", Status: "received"}
	if ok, err := rowMatchesSubmission(mappedFor([]any{"hello", []byte("req-1"), "received"}), s); err != nil || !ok {
		t.Fatalf("一致するはずです: ok=%v err=%v", ok, err)
	}
	if ok, _ := rowMatchesSubmission(mappedFor([]any{"hello", "req-1", "other"}), s); ok {
		t.Fatal("値が異なるのに一致扱いです")
	}
	if ok, _ := rowMatchesSubmission(mappedFor(), s); ok {
		t.Fatal("0行なのに一致扱いです")
	}
	if ok, _ := rowMatchesSubmission(mappedFor([]any{"hello", "req-1", "received"}, []any{"hello", "req-1", "received"}), s); ok {
		t.Fatal("2行なのに一致扱いです")
	}
	if _, err := rowMatchesSubmission(mappedFor([]any{"hello", int64(1), "received"}), s); err == nil {
		t.Fatal("文字列でない値を黙って受理しました")
	}
}

func TestLoadPostgresDSNInputRequiresExactlyOneSource(t *testing.T) {
	env := func(values map[string]string) dsnLookupEnvFunc {
		return func(key string) (string, bool) { v, ok := values[key]; return v, ok }
	}
	read := func(path string) ([]byte, error) {
		if path == "dsn.txt" {
			return []byte("postgres://example/db\n"), nil
		}
		return nil, os.ErrNotExist
	}

	if _, err := loadPostgresDSNInput(env(nil), read); err == nil {
		t.Fatal("どちらも未設定なのに受理しました")
	}
	if _, err := loadPostgresDSNInput(env(map[string]string{postgresDSNEnv: "a", postgresDSNFileEnv: "dsn.txt"}), read); err == nil {
		t.Fatal("両方が設定されているのに受理しました")
	}
	if got, err := loadPostgresDSNInput(env(map[string]string{postgresDSNEnv: "postgres://x"}), read); err != nil || got != "postgres://x" {
		t.Fatalf("直接入力: got=%q err=%v", got, err)
	}
	if got, err := loadPostgresDSNInput(env(map[string]string{postgresDSNFileEnv: "dsn.txt"}), read); err != nil || got != "postgres://example/db" {
		t.Fatalf("file入力: got=%q err=%v", got, err)
	}
	if _, err := loadPostgresDSNInput(env(map[string]string{postgresDSNFileEnv: "missing.txt"}), read); err == nil || strings.Contains(err.Error(), "missing.txt") {
		t.Fatalf("読取り失敗でpathを漏らしているか、受理しています: %v", err)
	}
	if _, err := normalizePostgresDSNFile([]byte("a\nb\n")); err == nil {
		t.Fatal("複数行を受理しました")
	}
}

func TestValidateDedicatedFixtureDatabase(t *testing.T) {
	cases := []struct {
		name          string
		database      string
		relations     int
		eventTriggers int
		wantErr       bool
	}{
		{"専用の空Database", "tei_e2a1", 0, 0, false},
		{"既定のpostgres", "postgres", 0, 0, true},
		{"template1", "template1", 0, 0, true},
		{"空の識別", " ", 0, 0, true},
		{"既存relationあり", "tei_e2a1", 1, 0, true},
		{"Event Triggerあり", "tei_e2a1", 0, 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDedicatedFixtureDatabase(tc.database, tc.relations, tc.eventTriggers)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
	if err := validateFixtureDatabaseRestored(0, 0); err != nil {
		t.Fatalf("復元済みを拒否しました: %v", err)
	}
	if validateFixtureDatabaseRestored(1, 0) == nil || validateFixtureDatabaseRestored(0, 1) == nil {
		t.Fatal("残存を見逃しました")
	}
}

func TestFixtureOwnershipMarker(t *testing.T) {
	marker, err := newFixtureOwnershipMarker()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(marker, "tei:reference-data-first-persistence:") {
		t.Fatalf("このツール専用の接頭辞ではありません: %s", marker)
	}
	if err := validateFixtureOwnershipMarker(marker); err != nil {
		t.Fatal(err)
	}
	if validateFixtureOwnershipMarker("tei:reference-postgres-header-detail-runtime:"+strings.Repeat("0", 32)) == nil {
		t.Fatal("別ツールの接頭辞を受理しました")
	}
	if validateFixtureOwnershipMarker(fixtureOwnershipPrefix+"zz") == nil {
		t.Fatal("不正な長さを受理しました")
	}
}
