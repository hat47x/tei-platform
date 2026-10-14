# TEI Studio PoC A — Wails Host（読取り専用Reference）

Issue [#1824](https://github.com/hat47x/tei-platform/issues/1824)、担当#2。正式なDesktop/Web Target採用やマルチユーザーWeb Hostの実装を示しません。

## 現在の構成

- `internal/studiohost`: Wailsに依存しない薄いHost allowlist。Workspace rootは起動時の`-workspace`でのみ指定し、Rendererの入力では選ばせない。`internal/operations`の`ValidateWorkspace`、`CatalogWorkspace`、`InspectSubject`を呼び出すだけです。
- `cmd/tei-studio/main.go`: `StudioBridge`の4つの読取り専用メソッドをWails serviceへ手登録する、PoC Aの通常ビルド入口。
- `cmd/tei-studio/window_{desktop,server}.go`: 同じGo serviceをWeb（`-tags server`）とDesktopで使用。Webは`127.0.0.1`固定です。
- `cmd/tei-studio/assets`: Go側へ埋め込む単純な初期画面。`frontend`の正式なUI/manifest/lock/登録箇所は#3が所有します。これは#3が差し替えるまでの確認用bootstrapです。

## バージョン・起動

指定版: Go `1.26.9`（ルート`go.mod`）、Wails `github.com/wailsapp/wails/v3@v3.0.0-beta.29`（pre-release）、対象main `128429482e02d96afe0123ee7da023256094dfca`。実OS・Node・WebView版は実測後に記録します。

```sh
# 依存取得可能な Go 1.26.9 開発環境で依存チェックサムを生成し、go.sum差分をレビュー
cd cmd/tei-studio
go mod tidy

# Desktop（OSのWebView開発依存が必要、repository rootからの相対Workspace）
go build -o tei-studio .
./tei-studio -workspace ../../reference/workspace

# Browser：loopbackの単一利用者参照検証だけ（GUIなし）
go build -tags server -o tei-studio-server .
./tei-studio-server -workspace ../../reference/workspace -port 43123
# http://127.0.0.1:43123/
```

`go test ./internal/studiohost` と `node --test cmd/tei-studio/test/client.test.mjs` は小さな自動検査です。Wails CLIと版を一致させる場合の指定は `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29`。PoCだけをnested Go moduleに隔離し、rootの全体CIでCGO/OS WebView依存を無条件に増やしません（ADR-0033の恒久的なsubproject構成決定ではありません）。最初の表示は生成bindingを使用せず、公式の`Call.ByName`を直接呼ぶbootstrapです。#3のTypeScript UIが整い次第、同じ`StudioBridge`4操作から公式生成bindingへ移します。

## #3/#6/#10への固定でない接続申し送り

Wails JavaScript service名（初期bootstrap）：`main.StudioBridge`。公開メソッド:

| method | 引数 | 応答 `value` |
|---|---|---|
| `ListWorkspaces()` | なし | `[{id,label}]`（`workspace-1`, ...） |
| `CatalogWorkspace(workspaceID, namespace)` | 登録済みID、exact namespaceまたは空文字 | 既存`operations.ModelCatalog` |
| `ValidateWorkspace(workspaceID)` | 登録済みID | 既存`operations.WorkspaceValidation`。`ok=true`でもDirect/Diagnosticsに不正があり得る |
| `InspectSubject(workspaceID, kind, subjectID)` | 登録済みID、exact canonical subject | 既存`operations.SubjectInspection` |

戻りは`{ok:true,value:T}`または`{ok:false,failure:{code,stage?,message}}`。`WORKSPACE_NOT_ALLOWED`、`WORKSPACE_LOAD_FAILED`（stage=load）、`SOURCE_INDEX_FAILED`（stage=source-index）、`INVALID_SUBJECT`、`UNSUPPORTED_SUBJECT_KIND`、`SUBJECT_NOT_FOUND`、`SUBJECT_AMBIGUOUS`、`OPERATION_FAILED`を区別。JS側は`HOST_TRANSPORT_FAILURE`・`INVALID_HOST_RESPONSE`を加えます。このenvelopeはPoC Hostからの表示用で、CLI形式や新しい標準Contractを定義しません。

## 非目標・未検証

`go.sum`のWails依存check-sumは外部module取得不可の作業環境ではまだ生成できません。真正な依存解決とGo/Wailsのbuild・実Web/Desktop WebView、2 browser隔離/イベント、OS別WebView/パッケージング・署名、Go/Host/CLI parityは**未確認**です。環境構築せずに合格扱いしないでください。

Wails Server BuildはEvent broadcastとService singletonを持ちます。本PoCはユーザー別状態・副作用・イベントを持たず、HTTPをloopbackに限定しますが、それだけで通信主体の認証やWorkspace内容の機密保護は保証されません。Webを公開ネットワークへ出したり複数ユーザーへ提供したりしないでください。Workspace内部のシンボリックリンク、ファイル置換、OS境界はここで証明していません。検査は#10、意味比較は#6、PoC B builderは#1、PoC C画面構成は#3が担当します。
