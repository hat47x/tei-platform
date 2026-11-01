package operations

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/workspace"
)

func referenceTraceKey() modelref.Key {
	return modelref.Key{Scope: "reference-observation", Subject: subject.Subject{
		Kind: "semantic-relation", ID: "sales.order.customer",
	}}
}

func referenceWorkspaceRoot() string {
	return filepath.Join("..", "..", "reference", "workspace")
}

// TraceWorkspaceは、Source Indexの構築とWorldの直列化をTraceの投影・整列と並行に行う。
// 並行にしても、3つの指紋が従来の逐次の計算（workspaceProjectionFingerprints）と一致する。
func TestTraceWorkspaceFingerprintsMatchSequentialComputation(t *testing.T) {
	root := referenceWorkspaceRoot()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	// 従来の逐次処理と同じ順序: index.BuildがWorldのプラグイン定義を正規化した後に、
	// Source Indexを作り、指紋を求める。
	derived := index.Build(loaded.World)
	projection := derivedbridge.FromIndex(derived, referenceTraceKey().Scope)
	if _, err := modelref.TraceDirect(projection.Graph, projection.Rules, referenceTraceKey(), modelref.TraceOptions{Mode: "summary"}); err != nil {
		t.Fatal(err)
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	worldFP, sourceFP, err := workspaceProjectionFingerprints(loaded.World, sources)
	if err != nil {
		t.Fatal(err)
	}
	// 期待値を実装の補助関数だけに頼らない。Worldの指紋は、正規化後のWorldのJSONの
	// SHA-256であり、補助関数が中身を壊す変異をここで検出する。
	worldJSON, err := json.Marshal(loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	if want := sourceindex.Fingerprint(worldJSON); worldFP != want {
		t.Fatalf("World fingerprint %s is not the digest of the normalized World JSON %s", worldFP, want)
	}

	for _, options := range []modelref.TraceOptions{
		{Mode: "summary"},
		{Mode: "full", Direction: "both"},
		{Mode: "full", Direction: "outgoing", Limit: 1},
	} {
		got, err := TraceWorkspace(root, referenceTraceKey(), options)
		if err != nil {
			t.Fatalf("%+v: %v", options, err)
		}
		if got.WorldFingerprint != worldFP || got.SourceIndexFingerprint != sourceFP {
			t.Fatalf("%+v: fingerprints differ from the sequential computation\n world %s vs %s\n source %s vs %s",
				options, got.WorldFingerprint, worldFP, got.SourceIndexFingerprint, sourceFP)
		}
		if want := workspaceTraceFingerprint(worldFP, sourceFP, got.Trace.SnapshotFingerprint); got.SnapshotFingerprint != want {
			t.Fatalf("%+v: snapshot fingerprint %s, want %s", options, got.SnapshotFingerprint, want)
		}
	}
}

// index.Buildは、Environmentの解決でプラグイン定義の入れ子のスライスを、Worldの上で
// その場で並べ替える（plugin.NormalizeDefinition）。TraceWorkspaceは、その後のWorldを
// 直列化して指紋にしてきたので、Buildの完了前にWorldを読む処理を並行に始めてはならない。
// 一方、Buildの後に並行して走る段（投影・TraceDirect・Source Indexの構築）は、
// Worldを書き換えない。並行に読むWorldの直列化が、逐次の結果と食い違わない前提である。
func TestTraceWorkspaceStepsAfterIndexBuildDoNotMutateTheLoadedWorld(t *testing.T) {
	root := referenceWorkspaceRoot()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	derived := index.Build(loaded.World)
	before, err := json.Marshal(loaded.World)
	if err != nil {
		t.Fatal(err)
	}

	projection := derivedbridge.FromIndex(derived, referenceTraceKey().Scope)
	if _, err := modelref.TraceDirect(projection.Graph, projection.Rules, referenceTraceKey(), modelref.TraceOptions{Mode: "full", Direction: "both"}); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.BuildSourceIndex(root, loaded); err != nil {
		t.Fatal(err)
	}

	after, err := json.Marshal(loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("a step that runs beside the World serialization changed the loaded World")
	}
}

// 同じWorkspaceへの同時呼出しが、常に同じ結果を返す。-raceで共有状態の書込みを検出する。
func TestTraceWorkspaceConcurrentCallsReturnIdenticalResults(t *testing.T) {
	root := referenceWorkspaceRoot()
	options := modelref.TraceOptions{Mode: "full", Direction: "both"}
	baseline, err := TraceWorkspace(root, referenceTraceKey(), options)
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(baseline)
	if err != nil {
		t.Fatal(err)
	}

	const callers = 12
	var wg sync.WaitGroup
	results := make([][]byte, callers)
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got, err := TraceWorkspace(root, referenceTraceKey(), options)
			if err != nil {
				errs[i] = err
				return
			}
			results[i], errs[i] = json.Marshal(got)
		}(i)
	}
	wg.Wait()
	for i := range results {
		if errs[i] != nil {
			t.Fatalf("caller %d: %v", i, errs[i])
		}
		if string(results[i]) != string(want) {
			t.Fatalf("caller %d returned a different result", i)
		}
	}
}

// Loadは成功するが、Source Indexの構築が失敗するWorkspaceを作る。
// 永続化の定義に同じ名前のindexを二つ宣言すると、BuildSourceIndexが
// 「persistence-index source identityは一意」の検査で失敗する。
func workspaceWithFailingSourceIndex(t *testing.T) string {
	t.Helper()
	source := referenceWorkspaceRoot()
	root := t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	persistence := filepath.Join(root, "databases", "sales-core", "persistence", "order.tei")
	content, err := os.ReadFile(persistence)
	if err != nil {
		t.Fatal(err)
	}
	const declaration = "    index orders_customer_idx on sales.order.customer_id\n"
	// Windows checkouts may convert LF to CRLF; the declaration is matched on LF text.
	content = []byte(strings.ReplaceAll(string(content), "\r\n", "\n"))
	if !strings.Contains(string(content), declaration) {
		t.Fatalf("fixture changed: the index declaration is gone from %s", persistence)
	}
	broken := strings.Replace(string(content), declaration, declaration+declaration, 1)
	if err := os.WriteFile(persistence, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestWorkspaceWithFailingSourceIndexLoadsButFailsAtSourceIndex(t *testing.T) {
	root := workspaceWithFailingSourceIndex(t)
	if _, err := workspace.Load(root); err != nil {
		t.Fatalf("the fixture must still load: %v", err)
	}
	_, err := TraceWorkspace(root, referenceTraceKey(), modelref.TraceOptions{Mode: "summary"})
	var failure *ValidationFailure
	if !errors.As(err, &failure) || failure.Stage != StageSourceIndex {
		t.Fatalf("expected a Source Index validation failure, got %v", err)
	}
	if !strings.Contains(err.Error(), "duplicate index name") {
		t.Fatalf("unexpected Source Index failure: %v", err)
	}
}

// Traceの失敗は、並行に走らせたSource Index側の失敗より優先して、従来どおりそのまま返る。
func TestTraceWorkspaceReportsTraceErrorBeforeSourceIndexFailure(t *testing.T) {
	root := workspaceWithFailingSourceIndex(t)

	// 引数の誤りは、何も構築する前に返る。
	_, err := TraceWorkspace(root, referenceTraceKey(), modelref.TraceOptions{Mode: "bogus"})
	var traceErr *modelref.TraceError
	if !errors.As(err, &traceErr) || traceErr.Code != "INVALID_ARGUMENT" {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}

	// グラフを見て初めて分かる失敗（存在しない主体）は、Source Index側の失敗より優先する。
	// 検査対象外の種別なので、inspectCanonicalでは弾かれず、TraceDirectが返す。
	missing := modelref.Key{Scope: "reference-observation", Subject: subject.Subject{Kind: "no-such-kind", ID: "x"}}
	_, err = TraceWorkspace(root, missing, modelref.TraceOptions{Mode: "summary"})
	var failure *ValidationFailure
	if errors.As(err, &failure) {
		t.Fatalf("the Source Index failure replaced the trace error: %v", err)
	}
	if !errors.As(err, &traceErr) || traceErr.Code != "SUBJECT_NOT_FOUND" {
		t.Fatalf("expected SUBJECT_NOT_FOUND from the trace, got %v", err)
	}
}

// Traceが失敗して戻るときも、並行に走らせた処理は合流済みで、goroutineが残らない。
func TestTraceWorkspaceLeavesNoGoroutinesWhenTheTraceFails(t *testing.T) {
	missing := modelref.Key{Scope: "reference-observation", Subject: subject.Subject{Kind: "no-such-kind", ID: "x"}}
	for i := 0; i < 5; i++ {
		before := runtime.NumGoroutine()
		_, err := TraceWorkspace(referenceWorkspaceRoot(), missing, modelref.TraceOptions{Mode: "summary"})
		var traceErr *modelref.TraceError
		if !errors.As(err, &traceErr) || traceErr.Code != "SUBJECT_NOT_FOUND" {
			t.Fatalf("expected SUBJECT_NOT_FOUND, got %v", err)
		}
		// Runtime background goroutines can appear immediately after a call
		// returns. Wait briefly for the count to settle before treating a
		// transient increase as a leaked goroutine.
		deadline := time.Now().Add(2 * time.Second)
		for {
			after := runtime.NumGoroutine()
			if after <= before {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("run %d: %d goroutines before the call, %d after; the call returned before its goroutines finished", i, before, after)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}
