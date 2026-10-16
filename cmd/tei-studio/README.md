# TEI Studio PoC A — Wails Host（読取り専用Reference）

Issue [#1824](https://github.com/hat47x/tei-platform/issues/1824)、担当#2。正式なDesktop/Web Target採用やマルチユーザーWeb Hostの実装を示しません。

## 現在の構成

- `internal/studiohost`: Wailsに依存しない薄いHost allowlist。Workspace rootは起動時の`-workspace`でのみ指定し、Rendererの入力では選ばせない。Canonical product operationsを呼び出し、Studio限定のsource observerには許可済みWorkspaceから上限付きで読み取ったバイト列だけを渡します。
- `cmd/tei-studio/main.go`: `StudioBridge`の5つの読取り専用メソッドをWails serviceへ手登録する、PoC Aの通常ビルド入口。
- `cmd/tei-studio/window_{desktop,server}.go`: 同じGo serviceをWeb（`-tags server`）とDesktopで使用。Webは`127.0.0.1`固定です。環境変数`WAILS_SERVER_HOST`が明示された場合、値が`127.0.0.1`以外なら起動を拒否します。
- `cmd/tei-studio/xead_observer.go`: `internal/xeadsource`を使うStudio専用の読取り・投影adapter。nested moduleの外へこのadapter依存を広げず、`internal/operations`から分離します。
- `cmd/tei-studio/assets`: Go側へ埋め込む単純な初期画面。`frontend`の正式なUI/manifest/lock/登録箇所は#3が所有します。これは#3が差し替えるまでの確認用bootstrapです。

## バージョン・起動

Go `1.26.9`、Wails `github.com/wailsapp/wails/v3@v3.0.0-beta.29`（pre-release）。基準mainは`128429482e02d96afe0123ee7da023256094dfca`です。`cmd/tei-studio/go.mod`・`go.sum`は、[#1824の実生成記録](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6167299922)と[Go1.26.9のCI実生成artifact](https://github.com/hat47x/tei-platform/actions/runs/38575429096)の内容と一致したものを採用しています。

```sh
# リポジトリrootから実行する。go versionは1.26.9
cd cmd/tei-studio
go mod download
go mod tidy
git diff --exit-code -- go.mod go.sum  # 依存lockに予期しない変更がないこと

# Linux上のWails Server BuildはCGO無効。read-only loopback限定
CGO_ENABLED=0 go build -mod=readonly -tags server -o tei-studio-server .
./tei-studio-server -workspace ../../reference/workspace -port 43123
# 別ターミナルから http://127.0.0.1:43123/ を表示

# 起動拒否を別途検証（この実行は非ゼロで終了すること）
WAILS_SERVER_HOST=0.0.0.0 ./tei-studio-server -workspace ../../reference/workspace -port 43124
# 期待メッセージ: WAILS_SERVER_HOST must be unset or exactly 127.0.0.1

# Desktop版（OS固有のWebView開発依存が必要）
go build -mod=readonly -o tei-studio .
./tei-studio -workspace ../../reference/workspace
```

Go検査はリポジトリrootで`go test -count=1 ./internal/workspace ./internal/operations ./internal/studiohost`、JS検査は`node --test cmd/tei-studio/test/*.test.mjs`を実行します。`cmd/tei-studio`はnested Go moduleのため、rootのGoテストはrootへ戻してから実行してください。Wails CLIと版を一致させる場合の指定は `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29`。PoCだけをnested Go moduleに隔離し、rootの全体CIでCGO/OS WebView依存を無条件に増やしません（ADR-0033の恒久的なsubproject構成決定ではありません）。最初の表示は生成bindingを使用せず、公式の`Call.ByName`を直接呼ぶbootstrapです。#3のTypeScript UIが整い次第、同じ`StudioBridge`5操作から公式生成bindingへ移します。
StudioのGo検査は`cmd/tei-studio`内で`go test -mod=readonly -count=1 -tags server .`を実行します。

## #3への引継ぎ：bootstrapと正式UIの交換点

- **#2が所有するHost側登録点**：`cmd/tei-studio/main.go`の`application.NewService(&StudioBridge{host: host})`。`//go:embed assets`で埋め込む`cmd/tei-studio/assets/`を`application.AssetFileServerFS(assets)`で配信する。Server/Desktop双方のGo Operationはここで同じものを使用する。
- **交換する資産（#3所有の構成出力）**：`assets/index.html`と`assets/main.js`が仮の一覧/検証/参照UI。#3のfrontend composition rootで生成するHTML/JS/CSSを、統合検査専用snapshotの同配信位置へ配置する。#3が#2の原branchを直接変更する必要はない。正式なbuild入力/配布layoutは#3のroot提出後に合意する。
- **維持するHost-neutral Port**：`assets/client.mjs`の`createReadOnlyClient(callByName)`は5操作を提供するHost adapter用port。`assets/main.js`が`/wails/runtime.js`の`Call.ByName`を注入する。#3のUI部品（Editor/Graph/Command）はWailsに直接依存せず、この5操作の結果をStudio固有Presenter経由で使用する。
- **現在のbootstrapが使用する1行の接続例**：`const studioHost = createReadOnlyClient((name, ...args) => Call.ByName(name, ...args));`。呼出名は`main.StudioBridge.*`、戻りは本README下記の`{ok,value}`/`{ok:false,failure}`。画面の成功表示をValidation合格や編集権限と取り違えない。
- **#1への手登録比較基準**：PoC B builderはこの手登録Hostと#3のUI出力を対照とし、生成物は検証snapshotで接続する。#1/#3からこのbranchの`main.go`を同時編集しない。

## #3/#6/#10への固定でない接続申し送り

Wails JavaScript service名（初期bootstrap）：`main.StudioBridge`。公開メソッドは5つです。`ObserveXead`は許可済みWorkspace内の単一`.xead`ファイルを読み、境界付きprojectionを返します。ファイル内容や内部例外はfailure messageへ含めません。

| method | 引数 | 応答 `value` |
|---|---|---|
| `ListWorkspaces()` | なし | `[{id,label}]`（`workspace-1`, ...） |
| `CatalogWorkspace(workspaceID, namespace)` | 登録済みID、exact namespaceまたは空文字 | 既存`operations.ModelCatalog` |
| `ValidateWorkspace(workspaceID)` | 登録済みID | 既存`operations.WorkspaceValidation`。`ok=true`でもDirect/Diagnosticsに不正があり得る |
| `InspectSubject(workspaceID, kind, subjectID)` | 登録済みID、exact canonical subject | 既存`operations.SubjectInspection` |
| `ObserveXead(workspaceID, relativePath)` | 登録済みID、Workspace相対`.xead` path | 読取り・深さ・要素数の上限を適用したStudio専用観察DTO。正規化・保存・書込みは行わない |

戻りは`{ok:true,value:T}`または`{ok:false,failure:{code,stage?,message}}`。`WORKSPACE_NOT_ALLOWED`、`WORKSPACE_ROOT_CHANGED`、`WORKSPACE_LOAD_FAILED`（stage=load）、`SOURCE_INDEX_FAILED`（stage=source-index）、`INVALID_SUBJECT`、`UNSUPPORTED_SUBJECT_KIND`、`SUBJECT_NOT_FOUND`、`SUBJECT_AMBIGUOUS`、`OPERATION_FAILED`を区別します。`ObserveXead`は`XEAD_SOURCE_READ_FAILED`、`XEAD_SOURCE_TOO_LARGE`、`XEAD_UNSUPPORTED_FORMAT`、`XEAD_PARSE_FAILED`、`XEAD_PROJECTION_FAILED`を返します。JS側は`HOST_TRANSPORT_FAILURE`・`INVALID_HOST_RESPONSE`を加えます。このenvelopeはPoC Hostからの表示用で、CLI形式や新しい標準Contractを定義しません。

## 非目標・未検証

Go1.26.9で生成した依存lockは原本へ反映しました。以前のhead `d070d76457f4140cf61f8c1c1357baf99a5530aa`では[専用CI #38575429096](https://github.com/hat47x/tei-platform/actions/runs/38575429096)のGo read-only・JS・CGO無効Serverビルド・loopback HTTP・公開listen拒否・Windows DesktopバイナリビルドがPASSしています。**原本lock反映後のheadへ過去のCI結果を自動転用しません**。#Workの実ブラウザー操作は過去head `fb52692dd518d930d696464e46962d8382969d7c`で確認されたもので、最新headでは未確認です。Desktop実WebView、Studio統合画面、2ブラウザー/Session隔離、OS配布/署名、CLI/Go/Wails実通信の意味照合、人的受入は未確認です。

Wails Server BuildはEvent broadcastとService singletonを持ちます。本PoCはユーザー別状態・副作用・イベントを持たず、HTTPをloopbackに限定しますが、それだけで通信主体の認証やWorkspace内容の機密保護は保証されません。Webを公開ネットワークへ出したり複数ユーザーへ提供したりしないでください。Workspace内部のシンボリックリンク、ファイル置換、OS境界はここで証明していません。検査は#10、意味比較は#6、PoC B builderは#1、PoC C画面構成は#3が担当します。


## PoC B生成GoサービスをHost内部で利用する固定実装（#1833連携）

通常の `go build` は`host_default.go`から`NewReadOnlyWithXeadObserver`を使い、通常Operation経路とStudio限定のXEAD observerを同じWorkspace allowlist配下へ接続する。`studio_generated` ビルドタグ指定時だけ、`host_generated.go` が #1833の出力した `staticservices.NewCatalogService` / `NewValidateService` をコンパイル時の型検査付きでHostの `ServiceFactory` に注入する。Wailsのサービス登録は `StudioBridge` の5操作のみ。生成サービスをWailsへ登録しない。

`internal/studiohost.ReadOnly` がRendererのWorkspace IDを起動時allowlistに照合し、許可済みrootを確定させてからだけFactoryへ渡す。生成サービスは `Catalog(namespace)` / `Validate()` のみ担当し、List/Inspectは従来どおり。通常経路はFactory=nilで直接Operation呼出し。Factory構築失敗、サービス欠落、Operation失敗は従来のHost failure envelopeへ変換し、非分類の内部エラー/絶対パスは公開しない。

生成器・fixtureの固定版は [#1833 `fe8e37b1c735dc138c36cb2d37db94ee44424b9e`](https://github.com/hat47x/tei-platform/pull/1833)。**生成Goファイルを手書き模倣してcommitしない**。未統合の#1833から該当ディレクトリだけを検証用snapshotに抽出して生成する。

```sh
# 専用snapshotのrepo root（#1832をベースにした検証用checkout）。PRをmergeしない
B=fe8e37b1c735dc138c36cb2d37db94ee44424b9e
git fetch --no-tags --depth=50 https://github.com/hat47x/tei-platform.git refs/pull/1833/head
git cat-file -e "$B^{commit}"  # fixed historical SHA must be locally available
git archive "$B" tools/reference-studio-static-composition | tar -xf -
go run ./tools/reference-studio-static-composition \
  -fixture tools/reference-studio-static-composition/fixture.json \
  -out internal/studiohost/staticservices
go test -count=1 ./internal/studiohost/...
cd cmd/tei-studio
CGO_ENABLED=0 go test -mod=readonly -count=1 -tags 'studio_generated,server' .
CGO_ENABLED=0 go build -mod=readonly -tags 'studio_generated,server' -o tei-studio-generated-server .
./tei-studio-generated-server -workspace ../../reference/workspace -port 43128
```

検証項目：型付きコンストラクタの実Goコンパイルと生成サービスを経由したHostメソッド実行、通常Hostの5操作/回帰、未知Workspace ID拒否、許可済みroot以外をFactoryに渡さないこと、内部例外のメッセージ抑止、警告・診断を含む結果一致、Wails Serverのloopback起動・非loopback環境変数拒否。これらは `studio_generated,server` タグと #1833出力の両方を含む **固定snapshotでの実行結果** と分けて記録する。HTTP health成功だけでWails IPCや実Desktop画面での生成Operation呼出し成功は主張しない。

検証済みの通常Host head/Browser/Windows Desktop binary結果を、この生成構成の実Desktop稼働やWeb認証/Session隔離の合格へ転用しない。生成型追加と既存`Services(root) []any`の一般化については#1との境界に留め、二重Registry/Renderer指定のrootは作らない。
