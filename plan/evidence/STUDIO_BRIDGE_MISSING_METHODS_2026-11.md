# Studio の binding 欠落で、ビューと Command 操作が到達できていなかった件（2026-11-11）

## 対象

- 実装: `cmd/tei-studio/main.go`（`StudioBridge`へ7メソッドを追加）
- 試験: `cmd/tei-studio/bridge_parity_test.go`（新規）、`cmd/tei-studio/host_selected_test.go`（操作数の固定を更新）
- 文書: `cmd/tei-studio/README.md`
- 対象版: `45d2122dc`（`main`）
- 環境: Windows、Go `1.26.10`（`cmd/tei-studio`はnested module、`-tags server`）

## 見つけた欠落

`StudioBridge`が Wails service へ公開していたのは**5メソッドだけ**でした。

```text
ListWorkspaces, CatalogWorkspace, ValidateWorkspace, InspectSubject, ObserveXead
```

一方 `assets/client.mjs` は**12の名前**を `main.StudioBridge.*` として呼んでいました。

```text
+ ViewPattern, ViewApplication, ViewPackage, PlanCommand,
  ExecuteCommand, ExecuteCommandWithPeriods, ExecuteCommandWithPolicy
```

Wails は**名前でメソッドを解決**するため、登録の無い名前はコンパイル時に落ちず、**ブラウザーのbinding時に失敗**します。結果として Workbench のビュー表示・Command計画・実行の操作は、押しても binding エラーになる状態でした。

`internal/studiohost` にはこれら7つが**すべて実装済み**で、`assets/main.js` も呼んでいます。**ホストAPIと UI の橋だけが欠けていました**（`cmd/tei-studio`はnested moduleで root の `go test ./...` の対象外、かつ `bridge_parity_test.go` のような突き合わせ試験が無かったため、長く気づかれていませんでした）。

## 変更

1. `StudioBridge`へ7メソッドを追加（各メソッドは `internal/studiohost` へ委譲するだけ。判断は Host 側にあります）。
2. `bridge_parity_test.go` を追加。`assets/client.mjs` の `main.StudioBridge.X` 名と `main.go` のメソッドを**両方向で**突き合わせます（片方だけ足した場合も落ちます）。
3. `host_selected_test.go` の「公開面の数が5」という固定を12へ更新し、**この数が5のままだったこと自体が欠落を許していた**と明記しました。
4. `cmd/tei-studio/README.md` の「5操作」「5つの読取り専用メソッド」という記述を12操作の実態へ更新（内訳と #4295 を明記）。

## 実行

```powershell
cd cmd/tei-studio
go build -mod=readonly -tags server .
go test -mod=readonly -count=1 -tags server .
go test -mod=readonly -count=1 .
node --test test/client.test.mjs
```

## 観測

- `go build -tags server` 成功。
- `go test -tags server` 成功。`go test`（タグ無し）も成功。
- `bridge_parity_test.go` 成功（欠落0・未使用0）。
- `node --test test/client.test.mjs` 成功（3件）。

## 読み取れること

- Workbench が使う12の操作すべてが Go 側で登録されている。
- `assets/client.mjs` の呼出名と登録メソッドの不一致は、試験で検出される。

## 読み取れないこと

- **実際のデスクトップ画面での動作**。Desktop版は OS 固有の WebView 依存が要り、この環境では起動していません。`-tags server` の Go 試験とJS試験までを確認しています。
- Wails の生成binding（`#3` のTypeScript UI）への移行状況。
- 7メソッドを UI から呼んだときの見た目・操作感（画面の設計判断）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。**UI と Host の橋の欠落を、名前の突き合わせ試験で塞いだ**記録です。画面での動作確認は未実施であることを明記します。
