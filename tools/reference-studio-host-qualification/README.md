# TEI Studio PoC A：Host境界の独立検査（#1824／開発#6）

状態：**検査基線の追加段階**。ここにあるGoの自動試験は、既存の読み取り専用操作が複数の独立した作業領域を読む場合の回帰試験です。Webの認証・権限・イベント隔離、Desktopのリソース制限、Wails実WebView、通常ブラウザーの動作を実証するものではありません。

PoC Aの要求・受入条件とWeb/Desktop分離の判断基準は[協調Issue #1824](https://github.com/hat47x/tei-platform/issues/1824)にあります。ここへ新しい製品要求、意味の正本、Host権限機構、受理経路、Gateを定義しません。既存の`internal/operations`（#1158）、`internal/desktophost`（#1529）、Host保護（#1043）、UI構成（#997）の責務を維持します。

## 先行できるGo操作の基線

`readonly_baseline_test.go` は本番Hostを模擬しません。次の狭い事実だけを検査します。

- 2つの独立した一時Workspaceを交互・並行に `operations.CatalogWorkspace` へ渡し、互いの意味名前空間が出力に混ざらないこと。
- 一方のWorkspaceで構文エラーを発生させ、`operations.ValidateWorkspace` が直接の構文診断と正常Workspaceの結果を混同しないこと。
- 読取り専用の投影を、受理権限や書込み権限と取り違えないこと（テストが書込みAPIを呼ばないという範囲）。

基線試験の実行例（**実行結果を伴わない手順**）：

```sh
go version
go test -count=1 ./tools/reference-studio-host-qualification/...
go test -race -count=1 ./tools/reference-studio-host-qualification/...
go test -count=1 ./internal/operations/...
go run ./cmd/tei validate -workspace reference/workspace
```

Goの要求版はルートの`go.mod`で確認します。Go試験の成功は「共有意味操作のread-only基線」に限られ、Hostの2クライアント分離の証明ではありません。

## PoC A Host実装と結合後に実測する行列

以下の行は現時点ですべて **未実施** です。Host担当が最小Operation／UI、起動手順、HTTPまたはWails Bindingの入出力を公開した後、同一版を固定してWeb・Desktop・CLIで比較します。空欄をPASSへ読み替えません。

| ID | 手順・入力 | 合格条件 | 現時点 |
| --- | --- | --- | --- |
| A-01 | 同じ正常WorkspaceをCLI、Go Operation、Web、Desktopから検証 | 診断・識別子・失敗分類が意味上一致。表示差分は別に記録 | 未実施 |
| A-02 | 不正な構文／存在しないWorkspace／Source Index失敗を各Hostへ入力 | 構文診断とload/source-index失敗を区別し、Host起動失敗を業務拒否に計上しない | 未実施 |
| A-03 | Webの独立クライアントA/Bで異なるWorkspaceを開き交互操作 | 選択、パス、結果、診断が混ざらない。認証済み利用者との区別を記録 | 未実施 |
| A-04 | Aで検証後にBへイベントを配信、切断・再接続・セッション失効を実施 | 機微payloadの全体配信や再接続後の越境参照が起きない。グローバルイベントに機微情報を含めない | 未実施 |
| A-05 | Desktopでroot外／親ディレクトリ／symlink／作業中のパス差替えを試行 | Hostで拒否。入力パスやfile dialogの値だけで権限を与えない | 未実施 |
| A-06 | Desktopの2ウィンドウ／WebView再起動／Source外部変更 | Draftや選択状態が無断で別窓・別Workspaceへ移らず、古いrevisionを受理扱いしない | 未実施 |
| A-07 | 同じGo操作とTypeScript UIでWeb・Desktopの実画面を操作 | 実ブラウザー・実WebViewの両方で操作を観測し、対象版と画面証跡を残す | 未実施 |

初期の許容範囲はloopback・単一利用者・read-onlyです。それでも二つのクライアントでの取り違えを検査します。認証、CSRF、HTTP/WebSocketセッション認可、購読隔離が未実装の段階で公開ネットワークに対応したとは主張しません。Wailsの汎用イベントbroadcastには機微情報を載せません。

## 証拠の記録条件

判定ごとに **対象コミットSHA、ブランチ、Go／Node／Wailsの実際の版、OS、WebView、ブラウザー、Workspace入力版、起動・試験コマンド、期待と実測の結果、失敗時のHost拒否と業務診断の区別** を記録します。結果は `pass`／`fail`／`blocked`／`not-run` を使い分けます。テストログに秘密情報や実利用者のパス・内容を残しません。

- Go単体試験と静的解析
- 実Goサービス呼出し
- 実ブラウザー操作
- Desktop実WebView操作
- 人による価値確認

以上を独立して記録します。机上確認、合成Workspace、モックでの成功を、実Hostの動作成功へ昇格しません。セッションとファイル選択の境界をHostに安全に持たせられない場合、Aは未合格として#1824へ報告し、Server Buildの代替を含めて再設計します。

## 担当境界

本ディレクトリはPoC Aの**外側からの検査**のみを担当します。Wails起動、Web/Desktop adapter、TypeScript UI、TEI宣言合成、Editor/Graph、Authn/Authz、Sourceへの反映は実装しません。PoC B（#1）、C、#1158、#1529、#997の担当が決まっている部分を再実装せず、実Hostとの結合が可能になった時点でこの行列を自動・手動の試験へ具体化します。
