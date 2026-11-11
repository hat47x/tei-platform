# Studio の UI から `after-*` 経路へ到達する配線の確認（2026-11-11）

## 対象

- 実装: `cmd/tei-studio/main.go`（`StudioBridge.ExecuteCommandWithReferenceStore`）、`cmd/tei-studio/assets/client.mjs`（port）、`cmd/tei-studio/assets/index.html`（チェックボックス）、`cmd/tei-studio/assets/main.js`（実行ボタン）
- 試験: `cmd/tei-studio/bridge_parity_test.go`、`cmd/tei-studio/host_selected_test.go`、`cmd/tei-studio/host_generated_test.go`、`cmd/tei-studio/test/client.test.mjs`、`cmd/tei-studio/test/main-async-scope.test.mjs`
- 対象版: `1a8b32443`（`main`）
- 環境: Windows、Go `1.26.10`（`cmd/tei-studio`はnested module、`-tags server`）

## 前段で残っていたこと

`internal/studiohost` には `ExecuteCommandWithReferenceStore` があり、`cmd/tei-studio` の `StudioBridge` にも12操作が登録されました（[binding 欠落の修正記録](STUDIO_BRIDGE_MISSING_METHODS_2026-11.md)）。しかし**その入口を UI から呼ぶ配線が無く**、Workbench から `after-*` へ到達できませんでした。

## 実装したこと

1. `StudioBridge.ExecuteCommandWithReferenceStore` を登録（`internal/studiohost` へ委譲）。
2. `assets/client.mjs` の port に `executeCommandWithReferenceStore` を追加。
3. `assets/index.html` に**任意指定のチェックボックス**「変更後のフックまで実行（メモリ上の参照Storeへ適用。永続しない）」を追加。
4. `assets/main.js` の参照実行ボタンが、チェック時は `executeCommandWithReferenceStore`、未チェック時は従来どおり `executeCommandWithPolicy` を呼ぶ。

**既定は変えていません。** チェックしなければ従来どおり `after-*` は理由つきで skip されます。

## 試験（表面の固定を更新）

- `host_selected_test.go` の公開面の数を12→13へ（追加した1操作）。
- `host_generated_test.go` の列挙に `ExecuteCommandWithReferenceStore` を追加（**13操作の名前を列挙**）。
- `bridge_parity_test.go` は `client.mjs` の13名と橋の13メソッドが一致することを確認（**成功**）。

## 実行

```powershell
cd cmd/tei-studio
go mod download
go build -mod=readonly -tags server .
go test -mod=readonly -count=1 -tags server .
go test -mod=readonly -count=1 .
node --test test/client.test.mjs
node --test test/main-async-scope.test.mjs
```

## 観測

- `go build -tags server` 成功、`go test -tags server` 成功、`go test`（タグ無し）成功。
- `node --test test/client.test.mjs` 成功（3件）、`test/main-async-scope.test.mjs` 成功（5件）。
- `-tags studio_generated,server` は**この環境では setup に失敗**する（生成サービス `internal/studiohost/staticservices` が手元に無く、CIで生成される）。**未実行として記録**し、成功として扱いません。

## 読み取れること

- Workbench の参照実行から、チェック時に `ExecuteCommandWithReferenceStore` を呼べる。
- 未チェック時は従来の入口のままで、既定の意味は変わらない。
- 橋と port の名前は13で一致する。

## 読み取れないこと

- **実際のデスクトップ画面での操作**（OS固有のWebView依存が必要。この環境では起動していない）。
- 生成タグ付き試験（CIで生成物が出た場合の結果）。
- `after-*` 拒否時の `effectReverted` が画面でどう見えるか（応答JSONに含まれるが、表示の設計は未検討）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。**UI から `after-*` 経路へ到達する配線を、任意指定として繋いだ**記録です。画面での操作確認と生成タグ付き試験は未実施であることを明記します。
