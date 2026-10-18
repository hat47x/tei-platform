# TEI Studio PoC A — Wails Host（読取り専用Reference）

Issue [#1824](https://github.com/hat47x/tei-platform/issues/1824)、担当#2。正式なDesktop/Web Target採用やマルチユーザーWeb Hostの実装を示しません。

## 現在の構成

- `internal/studiohost`: Wailsに依存しない薄いHost allowlist。Workspace rootは起動時の`-workspace`でのみ指定し、Rendererの入力では選ばせない。Canonical product operationsを呼び出し、Studio限定のsource observerには許可済みWorkspaceから上限付きで読み取ったバイト列だけを渡します。
- `cmd/tei-studio/main.go`: `StudioBridge`の5つの読取り専用メソッドをWails serviceへ手登録する、PoC Aの通常ビルド入口。
- `cmd/tei-studio/window_{desktop,server}.go`: 同じGo serviceをWeb（`-tags server`）とDesktopで使用。Webは`127.0.0.1`固定です。環境変数`WAILS_SERVER_HOST`が明示された場合、値が`127.0.0.1`以外なら起動を拒否します。
- `cmd/tei-studio/xead_observer.go`: `internal/xeadsource`を使うStudio専用の読取り・投影adapter。nested moduleの外へこのadapter依存を広げず、`internal/operations`から分離します。
- `cmd/tei-studio/assets`: Go側へ埋め込むbootstrapとWorkbench画面。Desktopは`/workbench/`を初期表示し、Serverの既定ページは従来どおりbootstrapです。Workbenchは既存の読取り専用5操作を使うStudioの開発・作成Host画面であり、生成Desktop Application Targetや正式な業務アプリではありません。

### Server HTTP呼出しの境界

Serverビルドでは、<code>/wails/runtime</code>へのHTTP RPC要求について、OriginをHTTPループバックURL、Hostを127.0.0.1と起動portから作る固定値にそれぞれ照合します。HTTPの標準port 80はブラウザーの直列化で省略されるため、期待値にも含めません。別portでは指定portを必須とします。<code>WAILS_SERVER_PORT</code>が定義されていれば空値や同じ値でも起動を拒否し、portは<code>-port</code>で指定します。初期ページや静的資産はOriginなしで取得できます。

このOrigin/Host確認は<code>/wails/runtime</code>のHTTP RPC呼出しだけに適用されます。<code>/health</code>、<code>/wails/custom.js</code>、<code>/wails/events</code>、<code>/wails/stream/ws</code>はWails側の別経路で、このMiddlewareの対象外です。現在のStudioはイベント購読・業務イベントの発行・独自stream handler登録を使用していません。Origin/Host確認は接続者認証、業務操作の認可、利用者間の分離を提供しません。

Wails `v3.0.0-beta.29`はWebSocketの許可設定を`WebSocketOriginPatterns`から作り、実装コメントでは使用するcoder/websocketが要求Hostと同じOriginを既定で許可すると説明しています（[固定版の実装](https://github.com/wailsapp/wails/blob/v3.0.0-beta.29/v3/pkg/application/websocket_server.go)）。これは本アプリの`127.0.0.1`固定照合とは別です。そのため、loopbackへ解決されるDNS別名をOriginとHostの両方に使えばhandshakeが通る可能性がありますが、これは実装からの推測であり、StudioでDNS別名やDNS rebindingを使った試験はしていません。現時点でStudioの業務イベント漏えいや操作実行が起きることは確認されていません。ブラウザースモークもWebSocket経路を確認しません。

AssetOptions.Middlewareへ呼出し計数用handlerを接続し、後段からStudioBridge.ListWorkspacesを直接実行する単体試験もあります。専用CIでは別に、Server実listenerを通るWails RPC、初期画面の資産配信、実ブラウザーからの`ListWorkspaces`呼出しを検証します。

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

Server版のブラウザースモークは、埋込みのbootstrapとWorkbenchをChromiumで開きます。Workbenchでは`/wails/runtime.js`経由のCatalog→Inspect→Validate、直接の子モデル選択、Host応答と画面のcanonical投影・診断一覧の一致を確認します。これはServer版のHTTPブラウザー経路です。Windows専用Desktopスモークは実Wails WebView2でWorkbenchを開き、埋込み資産の読込み、既存Host BridgeのCatalog→Inspect→Validate、読み取り専用のcanonical投影と診断一覧の一致に加えて、最後のモデル一覧項目からTabでEditorへ移動し、Home/Shift+Endで選択した後に文字・Backspace・Deleteを送っても表示本文、scratch Workspace、固定XEAD fixtureが変わらないことを確認します。既知の読取りHost RPCでWebView2 Resource Timingを校正し、キーボード操作中に観測可能な`/wails/runtime`要求が増えないことを確認します。各キーの信頼済みkeydownと対象DOM、Editor focusとDOM選択、WebView2/Edge WebDriver版、各操作後の本文hash、Workspace・fixture hash、WebDriver screenshotをartifactへ記録し、WebView2を正常終了します。これはWebView2のDOM focus/selectionとResource Timingを確認する自動試験で、支援技術や人による操作・受入を確認しません。両スモークは固定Workspaceでの自動確認であり、認証・業務認可・Principal/Workspace分離、共有Web Host、OS配布・署名、人的受入は確認しません。Origin/Host拒否は従来どおり別のcurl検査で扱います。

別のGo単体試験では、同じ参照Workspaceに対するCLIの`model catalog`結果とStudio `ReadOnly.CatalogWorkspace`結果を、`namespace=sales`と全namespaceの両方で比較します。CLIのJSON envelopeを除いた構造化結果を比較し、項目順と権限制約項目の順序は正規化しますが、同じSubjectの重複項目は保持して比較します。`sales.order`のindexed状態、`sales.order.customer`の親・詳細・indexed状態・非曖昧性、namespace指定時のapplication項目除外、全namespace指定時のapplication項目、Workspace読込失敗の分類と部分結果がないことを確認します。重複IDを持つ小さなWorkspaceでも、両経路が2件を保持し、どちらも曖昧として示すことを確認します。ModelCatalogには診断配列がないため、診断内容は比較対象に含みません。このGo単体試験だけでは、共通operationの独立した正しさ、CLI実行ファイルとWails HTTP/Bridge経路、認証・権限同等性は証明しません。

別のPoC A CI stepでは、root moduleから作成した実CLI実行ファイルと、実Listener上のWails `main.StudioBridge.CatalogWorkspace` HTTP RPCを同じ参照Workspaceで呼び、CLI stdoutの`result`とBridge responseの`value`全体を比較します。項目順を正規化して重複項目は保持し、`namespace=sales`、全namespace、空Workspaceのload failureを確認します。失敗時はCLI exit 2 / `WORKSPACE_LOAD_FAILED`と、HTTP 200 response内の`ok=false`、同一failure code・`stage=load`、値がないことを照合します。両経路が同じ`operations.CatalogWorkspace`を使うため、共通operation自体の独立した正しさは証明しません。loopbackでの自動HTTP試験であり、Catalog画面のブラウザー操作、認証・業務認可・Principal/Workspace分離、Desktop WebView、人的受入は未確認です。

Go1.26.9で生成した依存lockは原本へ反映しました。以前のhead `d070d76457f4140cf61f8c1c1357baf99a5530aa`では[専用CI #38575429096](https://github.com/hat47x/tei-platform/actions/runs/38575429096)のGo read-only・JS・CGO無効Serverビルド・loopback HTTP・公開listen拒否・Windows DesktopバイナリビルドがPASSしています。**過去のCI結果を後続headへ自動転用しません**。#Workの実ブラウザー操作は過去head `fb52692dd518d930d696464e46962d8382969d7c`で行われました。今回のブラウザースモークも対象headのCI結果で判定します。Desktop実WebView、Studio統合画面、2ブラウザー/Session隔離、OS配布/署名、人的受入は未確認です。CLI実行ファイルとWails Server HTTP RPCのCatalog比較は上記の別stepで扱いますが、Desktop実WebViewの証拠にはなりません。

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
