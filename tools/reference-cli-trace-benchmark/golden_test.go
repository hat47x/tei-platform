package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/modelref"
	"tei/internal/operations"
	"tei/internal/workspace"
)

// golden.json の期待値は、#1177で処理を変更する前の実装（コミットbc4b673e、
// 基点65f83eee）で生成した出力のSHA-256である。検査するのは成功時の出力だけで、
// summary・full・full -limit・Source Index・ページ継続が処理変更後も
// バイト単位で変わらないことを示す。構文エラーの失敗文言と「先頭のanchorを採る」
// 規則は、internal/workspace/source_index_anchor_table_test.goで検査する。
// 値を更新するときは、変更前の実装が出力した根拠をPRに残すこと。
const goldenFile = "testdata/golden.json"

func digest(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func computeGolden(t *testing.T) map[string]string {
	t.Helper()
	got := map[string]string{}
	for _, size := range Sizes[:2] { // small, medium。largeは変更前の実装で約2分かかるため手動計測に限る
		root := filepath.Join(t.TempDir(), size.Name)
		if err := Generate(root, size); err != nil {
			t.Fatal(err)
		}
		loaded, err := workspace.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		sources, err := workspace.BuildSourceIndex(root, loaded)
		if err != nil {
			t.Fatal(err)
		}
		got[size.Name+"/sourceIndex"] = digest(sources)
		for _, op := range ops {
			key, options, err := opArgs(op)
			if err != nil {
				t.Fatal(err)
			}
			result, err := operations.TraceWorkspace(root, key, options)
			if err != nil {
				t.Fatalf("%s/%s: %v", size.Name, op, err)
			}
			got[size.Name+"/"+op] = digest(result)
		}
		// ページ継続: 先頭1件の後、返ったfingerprintで次ページを取り、全ページを連結して
		// 一括fullと同じ行になること。
		key, _, _ := opArgs("full")
		whole, err := operations.TraceWorkspace(root, key, modelref.TraceOptions{Mode: "full"})
		if err != nil {
			t.Fatal(err)
		}
		var paged []modelref.TraceRow
		offset := 0
		for {
			page, err := operations.TraceWorkspace(root, key, modelref.TraceOptions{
				Mode: "full", Offset: offset, Limit: 700, ExpectedFingerprint: whole.SnapshotFingerprint,
			})
			if err != nil {
				t.Fatal(err)
			}
			paged = append(paged, page.Trace.Rows...)
			if page.Trace.NextOffset == nil {
				break
			}
			offset = *page.Trace.NextOffset
		}
		if digest(paged) != digest(whole.Trace.Rows) {
			t.Fatalf("%s: ページを連結した行が一括fullと一致しません", size.Name)
		}
		got[size.Name+"/pagedRows"] = digest(paged)
	}
	return got
}

func TestGoldenOutputUnchangedFromBaseline(t *testing.T) {
	got := computeGolden(t)
	if os.Getenv("TEI_WRITE_GOLDEN") == "1" {
		raw, _ := json.MarshalIndent(got, "", "  ")
		if err := os.WriteFile(goldenFile, append(raw, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]string
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != len(got) {
		t.Fatalf("golden件数が違います: want=%d got=%d", len(want), len(got))
	}
	for name, hash := range want {
		if got[name] != hash {
			t.Errorf("%s: 変更前の出力と一致しません\n want %s\n  got %s", name, hash, got[name])
		}
	}
}

// 古い継続token、およびWorkspace更新後の継続tokenが、STALE_SNAPSHOTで拒否されること。
func TestFailureBehaviourUnchanged(t *testing.T) {
	root := filepath.Join(t.TempDir(), "w")
	if err := Generate(root, Sizes[0]); err != nil {
		t.Fatal(err)
	}
	key, _, _ := opArgs("full")
	var typed *modelref.TraceError
	_, err := operations.TraceWorkspace(root, key, modelref.TraceOptions{Mode: "full", ExpectedFingerprint: "sha256:stale"})
	if !errors.As(err, &typed) || typed.Code != "STALE_SNAPSHOT" {
		t.Fatalf("古い継続tokenが拒否されません: %v", err)
	}
	// Source側が変わるとfingerprintが変わり、前ページのtokenはfail-closedで拒否される。
	first, err := operations.TraceWorkspace(root, key, modelref.TraceOptions{Mode: "full", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(root, "semantic", "f0000", "model.tei")
	body, _ := os.ReadFile(extra)
	if err := os.WriteFile(extra, append(body, []byte("\ndata Extra {\n    attribute a0000 type hub.hub_id\n}\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = operations.TraceWorkspace(root, key, modelref.TraceOptions{
		Mode: "full", Offset: 1, Limit: 1, ExpectedFingerprint: first.SnapshotFingerprint,
	})
	if !errors.As(err, &typed) || typed.Code != "STALE_SNAPSHOT" {
		t.Fatalf("Workspace更新後の継続tokenが拒否されません: %v", err)
	}
}
