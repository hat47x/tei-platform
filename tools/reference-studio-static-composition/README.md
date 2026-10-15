# TEI Studio PoC B — 静的合成の最小参照実装


## 最新の固定snapshot実証（2026-10-15）

#Workの実測：[Issue #1824 #6181914721](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6181914721)。下記は**別Draftの所有pathをexact SHAで抽出した隔離snapshot**の実証であり、#1833単独checkoutの実行やmainへの統合、製品受入を意味しない。

| 固定入力 | exact revision |
| --- | --- |
| #1 静的生成器 / Draft #1833 | `fe8e37b1c735dc138c36cb2d37db94ee44424b9e` |
| #2 Host / Draft #1832（`studio_generated,server`） | `4bc437c72e63a3b5d578b1efe2c60be510a9f8a9` |
| Workbench / Draft #1849 | `c55ad2285268533f5f1c90fbc9fe766bae57f009` |
| Editor / Draft #1830 | `a7084fe1711cc48eff56fd20e3dc81c508f520a3` |
| Graph / Draft #1828 | `786d8d82e19e8b08690aceadf9d180374e211448` |
| Command / Draft #1829 | `01f5022240fd86c6ac38c1107310e299a1615b48` |

- **実物の一致**：生成`services.generated.go` SHA-256 `867501c5a98d71337178d8bd7995e2e4eb6d276accd25151a2c18d8cb5adee50`、生成Workbench bundle SHA-256 `1166d9489c76028ba21c9eaf44d25c6f36f98d5845d1f5a0599ea51bdfb1f1ee`。#2の生成Factoryは既存Hostが許可したrootだけを受け取り、追加Registryや生成サービスの直接Wails登録を行わない。
- **PASS（#Work、WSL Linux / Go 1.26.9 / Wails v3 beta.29）**：生成モードのGo `test`・`vet`・Server `build`。IABの実ブラウザーで `127.0.0.1:43130` の生成Wails Serverを操作し、workspace-1 Catalog **72件**／Validate **警告3件**、workspace-2 Catalog **19件**／警告0件（旧診断残留なし）。workspace-1に戻り、Graphの`customerId`選択から`sales.order.customer_id` の実InspectSubject取得と親Orderとの2ノード関係を確認。配信bundleのSHA照合もPASS。
- **PASS（別CI）**：[#38662384542](https://github.com/hat47x/tei-platform/actions/runs/38662384542) は#2 exact `4bc437c7` に対し、Windows Desktopの通常・生成モード**バイナリビルド**を成功。GUI実起動の証拠ではない。
- **NOT RUN / 未実証**：#WorkのWSL Linuxで`-race`は`pkg-config`不在により開始不可。実Desktopのnative WebView起動・終了、Windows Desktop実画面、接続別認証/Session隔離、完全Source Closure／署名／Plugin・Build Lock、全OS資源境界およびPoC全体の正式合格は未確認。

この記録は下記の設計時点の未検証事項を**当該固定snapshotについてのみ更新**する。過去headのFAIL/SKIP、異なるGraph/Editor版、`studio_generated`なしの通常HostでのUI検証結果へ転用しない。すべてのDraftは未mergeのまま維持する。


対応: [Issue #1824](https://github.com/hat47x/tei-platform/issues/1824)、Goの構造的合成 [ADR-0032](../../plan/adr/0032-composition-kernel-replacement.md)、UI構成キット [#997](../../reference/ui-customization/composition/README.md)。

## 対象と非目標

`fixture.json` は **PoC専用の選択fixture** であり、正式なTEI DSL・Composition Profile schema・Plugin Lockではありません。この実装は新しいResolverを作りません。Go側は既存の `internal/composition.Registry.Apply / InspectWiringSet`、UI側は既存の `resolveUICompositionKit` を呼び、信頼されたソース識別子と型付きread-only Operationに対応する**静的構築用のソース**を作ります。

対象は2つのGoサービス `studio.catalog` / `studio.validate` と、2つのUI公開点 `graph` / `commands` です。Goサービスは既存 `operations.CatalogWorkspace` と `operations.ValidateWorkspace` への薄い接続です。UI実装は #9 Graph / #5 Command の提供する `mountReadonlyGraph` / `mountCommandPanel` を参照します。UI部品内の権限判定、第二Operation、第二Plugin Resolver、動的Goロードはありません。

## 作成手順

リポジトリルートから実行します。生成UIの相対importは下記の固定配置を前提にしているため、出力先は専用の `generated/` ディレクトリを指定してください。生成物はPoCのローカル派生物であり、現段階では正本としてcommitしません。実行前に `go.mod` の要求するGo版を満たす必要があります。

```sh
OUT="tools/reference-studio-static-composition/generated"
go test -count=1 ./tools/reference-studio-static-composition
node --test tools/reference-studio-static-composition/compose-ui.test.mjs

go run ./tools/reference-studio-static-composition \
  -fixture tools/reference-studio-static-composition/fixture.json -out "$OUT"
node tools/reference-studio-static-composition/compose-ui.mjs \
  tools/reference-studio-static-composition/fixture.json "$OUT"
ls "$OUT"
```

出力は `services.generated.go` と `ui.generated.mjs`、それぞれの `*.provenance.json` です。来歴は選択fixtureのSHA-256、生成ソースのSHA-256、Goサービス解決順、UI Kitのtraceを保持します。**これらは構築入力の照合値に過ぎず、実際のGo/TS資産digest、Plugin Lock、ビルド成果物署名、許諾、権限、実サービス稼働の証明ではありません。** 完全なビルド資産来歴の接続は後続作業です。

生成Goファイルのpackageは `staticservices` です。#2のHostは、生成ファイルをGoパッケージに配置したうえで `Services(trustedWorkspaceRoot)` の静的構成リストを受け取れます。ただし渡すrootの選択とアクセス制御は必ずHost側で行い、ブラウザーが任意のrootを指定できるAPIとして公開してはいけません。既存の共通Operation呼出しを直接Wails Binderの認可代わりにしません。

生成UIファイルは `studioUIContributions` の **名前付き2要素の配列**です。別のRuntime RegistryやEvent Busではなく、#3が所有するcomposition rootへ渡すための静的import一覧です。現状のimport先 `reference/tei-studio-graph/graph.ts` と `reference/tei-studio-command/command.ts` は並行するDraft PR #1828 / #1829 の資産であり、通常の単独branchでは実装が揃っていません。**上記のexact snapshotでは生成import・bundle・実ブラウザー起動が検証済み**ですが、別revisionや正式なビルド構成への一般的適合までは保証しません。#3と登録入口が決まるまでは手書きのcomposition rootを変更しません。

## 拒否と不成立条件

Go側は未知のサービス、重複選択、互換しないexact version、異なるscope、不完全なfixtureを拒否します。構造的な契約依存の適格性は既存Registryが担当し、選択結果だけで実行権限を得たとは扱いません。UI側は既存Kit Resolverの未知implementation・重複/未知slot・version/framework不一致を利用して拒否します。ファイル列挙順や曖昧なfallbackで順序を決定しません。

この**独立Reference単体では**、Hostの認可・Wailsでの動作・製品用Artifact/Build Lockを証明しません。ただし上記固定snapshotについては#Workが**生成GoのHost内部使用、UI生成import・実bundle、実Wails Serverブラウザー操作**を独立に実測しています。異なるsnapshot、実Desktop WebView、Browser session隔離、任意プロセスロード、完全なArtifact/Build Lock来歴、製品用TEI宣言の一般解決は依然として未実証です。したがってPoC B全体の合格とはしません。

#7の独立黒箱検査は、本Referenceで使うfixtureと生成結果の内容・拒否/来歴を参照できますが、#7固有の検査器・負例fixture自体は#7の担当のままです。

## #1849 固定workbenchとの最小接続（mergeしない検証snapshot）

対象は #Workの [Draft PR #1849](https://github.com/hat47x/tei-platform/pull/1849) **exact head `9a290dc61f585dfbf292e9032e5c5bbbad04351a`** の `reference/tei-studio-workbench/main.ts`（Git blob `2a65a9f70245068bf2455248d98d280b382b01e4`）。#7とこのWork headを共通にする。#2 Host `d520689cfb0a2e2919aa86b0beeba54802b31465`、#4 Editor `cc57052f23b69582fc0d3800ab92b2210a28bb4f`、#9 Graph `36799dba33fe0e19db3f7b56e7af1e18da134a71`、#5 Command `26f47772ed49f5c4a9e1c50026c335783bc5e153` を#Workの基準入力として用いる。Go/Node builderの原本headも実行時に固定し記録する。

`compose-ui.mjs`は従来の `studioUIContributions` を引き続き出力し、同一静的importから **`mountReadonlyGraph`、`mountCommandPanel` の名前付きalias** と型情報 `ui.generated.d.mts` を追加する。新たなResolver/Registryはない。出力ディレクトリは上記 `generated/` に固定する。既存workbenchの2 importだけを次へ差し替える。

```diff
-import { mountReadonlyGraph } from "../tei-studio-graph/graph.ts";
-import { mountCommandPanel } from "../tei-studio-command/command.ts";
+import { mountReadonlyGraph, mountCommandPanel } from "../../tools/reference-studio-static-composition/generated/ui.generated.mjs";
```

この差分は#Workの原本へ直接適用しない。**#1849 exactのファイルを取り出した検証専用checkout**で、以下を実行して `main.generated.ts` を隣に生成する（元の `main.ts` は不変）。アダプターは対象原本Git blobが不一致なら `workbench_revision_mismatch` で拒否し、出力に同名異内容がある場合も上書きしない。Goサービスの生成は別の操作であり、Wails Hostへ生成Goサービスを直接登録しない。

```sh
# リポジトリroot／#1849等の固定ソースをpath別に置いた専用snapshotで実行
OUT=tools/reference-studio-static-composition/generated
go test -count=1 ./tools/reference-studio-static-composition
node --test tools/reference-studio-static-composition/*.test.mjs

go run ./tools/reference-studio-static-composition \
  -fixture tools/reference-studio-static-composition/fixture.json -out "$OUT"
node tools/reference-studio-static-composition/compose-ui.mjs \
  tools/reference-studio-static-composition/fixture.json "$OUT"
node tools/reference-studio-static-composition/connect-workbench.mjs \
  reference/tei-studio-workbench/main.ts

cd reference/tei-studio-workbench
npm ci
npm test
npm run typecheck
# 本体main.tsには書き込まず、接続後main.generated.tsを実bundle化
node --input-type=module -e 'import {build} from "esbuild"; import {resolve} from "node:path"; await build({absWorkingDir:process.cwd(),entryPoints:["main.generated.ts"],outfile:"dist/main.generated.js",bundle:true,platform:"browser",format:"esm",target:"es2022",nodePaths:[resolve("node_modules")]});'
```

`connect-workbench.test.mjs`は差し替え2行以外のHost呼出しが不変であること、重複/欠落importと未固定のworkbench SHAを拒否すること、生成ソース/型定義/来歴のSHAを確認する。実Workbenchの照合には`testdata/poc-b-workbench-snapshot/`の固定試料を明示的に使い、実行時checkoutにある`reference/tei-studio-workbench`や`reference/tei-studio-command`の内容を暗黙に読みません。試料は新pinのexact head `c55ad2285268533f5f1c90fbc9fe766bae57f009`とCommand exact head `01f5022240fd86c6ac38c1107310e299a1615b48`から取得し、各Git blob SHAを既存pinと照合します。Graph/Commandのimport名は型付きaliasへ静的に結び付き、#1849の既存のEditor、Presenter、workspace選択、Commandのinvoke、`createReadOnlyClient` と四操作 `ListWorkspaces/CatalogWorkspace/ValidateWorkspace/InspectSubject` を変更しません。Graph/CommandをHost権限として解釈しません。

**負例と再生成**：Go側の不足サービス・重複・版/スコープ不一致は既存 `composeServices` で拒否し、fixtureサービスの並びを反転しても生成Goソースのバイト列は一致させる。UI側は重複slot/implementation・未知impl・Kit版 `latest`（既存Resolverの `invalid_request`）・PoC固定 `0.1.0` 以外の正しいSemVerも拒否し、Kit slot/changesの並びに依存せずソース/型定義を一致させる。入力自体を変更したとき `fixtureSha256` は異なってよい。実物2回生成のファイルSHAと検証snapshotの結果は#7の検査対象。

**Host受入との違い**：生成Go `Services(root)` の2サービスは `operations` の意味比較用であり、#2の実BridgeのWorkspace許可リスト、非任意path制約、`{ok:true,value}` / `{ok:false,failure}`、Host transport失敗処理を置換しない。実Wails Server/Browser/Desktop確認、完全な依存closure、署名、Build/Plugin Lock、生成Wails bindingの適合は単体・bundle検査と分けてIssueへ記録する。#1849/#1832/#7原本やmainをmerge/editしない。

## #1849 workbench固定版の更新（2026-10-15）

**旧pinを維持**：#1849 exact `9a290dc61f585dfbf292e9032e5c5bbbad04351a`、`main.ts` Git blob `2a65a9f70245068bf2455248d98d280b382b01e4`、Command #1829 `26f47772ed49f5c4a9e1c50026c335783bc5e153`。

**新pinを追加**：#1849 exact `c55ad2285268533f5f1c90fbc9fe766bae57f009`、`main.ts` Git blob `cc2644e5a9167dbc3a0b5368ff270276a2ba1b67`、Command #1829 exact `01f5022240fd86c6ac38c1107310e299a1615b48`。この版は`reference/tei-studio-workbench/responses.ts`を使い、Commandの`beginContext`・`setDiagnosticsForContext`と併せて旧応答を破棄する。これらの関連ファイルを同じ固定snapshotへ取り出し、古いCommandの混入を避けること。Go Hostは従前の4操作・read-only envelope・起動時workspace allowlistを維持する。

`connect-workbench.mjs`は**この2種類の実際のGit blob SHAだけ**を受理し、入力の内容から一致する固定版を自動識別する。 さらに snapshot 内のCommand `command.ts`のGit blob（旧 `4e0d5484f82944d5b7d3b0b8431f3a9a1eaa7081`／新 `6c6ee81288c9857ca2f1c282639ba49f059f5535`）を検証し、新headでは `responses.ts` のGit blob `9fa305e227bee96b66002362121db3878e92327d` も要求する。旧・新のソース混在を `command_revision_mismatch`／`responses_revision_mismatch` で拒否する。`connectWorkbenchSource(source)`に`pin=false`を渡す経路は存在せず、第2引数を与えた呼出し自体を拒否する。実際のCLIは常にSHAを検証し、どちらのpinを使用したかログへ出す。検証用の純粋な`inspectWorkbenchImports(source)`はimportの重複・不足を調べるだけで、生成成果物の接続・保存は行わない。

#7と共有する新固定snapshotは **#1833の本差分後exact head + #1849 `c55ad2285268533f5f1c90fbc9fe766bae57f009` + #1829 `01f5022240fd86c6ac38c1107310e299a1615b48`**。#Workによる旧版`11e80396...`の実bundle・Host検査と、新pin版の結果を混同しない。新pin版で`node --test`、生成moduleの実ファイル2回一致、`main.generated.ts`のstrict typecheckとbundle、Hostの実4操作、Command旧応答排除を独立に確認するまで新組合せのPASSとはしない。


## #2の実Host内部注入との接続（Issue #1824 / #6181598991）

#2は [Draft PR #1832](https://github.com/hat47x/tei-platform/pull/1832) のHost所有パスに型付きOperation注入を実装した。#1の**既存**Go生成器が実際に出力する次の二つのconstructor/methodだけで、#2のtyped interfaceへ適合する。不要な`NewTypedServices`や第二Factory/Registryは設けない。

```go
// package staticservices（PoC Bの実生成結果）
func NewCatalogService(root string) *CatalogService
func (s *CatalogService) Catalog(namespace string) (operations.ModelCatalog, error)
func NewValidateService(root string) *ValidateService
func (s *ValidateService) Validate() (operations.WorkspaceValidation, error)
```

**対象Git blobを保持**：#1 `main.go`は `8d6eca235264f00a44e43484b757bd60fc00c270`、`main_test.go`は `458cea39a4dbad2f3fd0322b2b61f8ecd8ed9756`。これは#2の固定CIで実際に使用した#1833 `fe8e37b1c735dc138c36cb2d37db94ee44424b9e`と同じ生成器・試験コードであり、未証明の追加APIは採用しなかった。

### #2が実装・検証したHost契約

- #2実装head: `9a0c195e38861bf64ad7cdd7100558ea65513d75`。通常 `NewReadOnly(paths)` の場合factory=nilで通常Operationを利用。生成モードは `NewReadOnlyWithFactory(paths, factory)`、`type ServiceFactory func(trustedRoot string) (RootServices,error)` を利用する。
- `ReadOnly.workspace(workspaceID)` で**起動時allowlistを先に検証**。許可されたrootのみ`ServiceFactory`へ渡し、Catalog/Validateを生成typed serviceへ委譲する。未知・パス風のIDは拒否。失敗は従来の`operationFailure(err)`でコード・stage・公開メッセージに変換する。List/Inspectは従来のOperationのまま。
- 通常build tag `!studio_generated` は既定動作、生成mode tag `studio_generated` は `cmd/tei-studio/host_generated.go`で静的生成constructorを使い、**Wailsには引き続き既存 `StudioBridge` 4メソッドだけ登録**。以前検討した`studio_poc_b_generated`とは異なるため誤用しない。
- 生成ソースは検証専用snapshotで `internal/studiohost/staticservices/` へ出力する。#2の`host_generated.go`が `tei/internal/studiohost/staticservices` をimport。元の#1832や#1833ブランチへ出力ファイルをcommitしない。

### 固定snapshotの再現

```sh
# #2 head 9a0c195e + #1 generator exact fe8e37b1 のpathだけ抽出（Draft非merge）
go run ./tools/reference-studio-static-composition \
  -fixture tools/reference-studio-static-composition/fixture.json \
  -out internal/studiohost/staticservices
go test -count=1 ./internal/studiohost/... ./tools/reference-studio-static-composition
go vet ./internal/studiohost/... ./tools/reference-studio-static-composition
cd cmd/tei-studio
CGO_ENABLED=0 go test -mod=readonly -count=1 -tags 'studio_generated,server' .
CGO_ENABLED=0 go build -mod=readonly -tags 'studio_generated,server' -o tei-studio-generated-server .
./tei-studio-generated-server -workspace ../../reference/workspace -port 43128
```

**#2の独立CI証拠**：[run #38661488708](https://github.com/hat47x/tei-platform/actions/runs/38661488708)、Go1.26.9、Wails beta29。生成ファイルSHA-256 `867501c5a98d71337178d8bd7995e2e4eb6d276accd25151a2c18d8cb5adee50`、`TestGeneratedGoServicesRunThroughRealHost` PASS、許可済みroot factory・エラー非漏洩・通常Host回帰PASS、生成Server実build/loopback health・配信smoke PASS、Windows通常Desktop binary build PASS。旧#2のCI失敗は固定PR HEADを移動先からfetchしてsha照合できなかった競合であり、後続CIはpinしたcommitからsource抽出して解消した。

このCIは実生成Go packageの**Host内での使用**を試験しているが、HTTP healthが成功したことからWailsの実ブラウザーIPCで生成serviceへのCatalog/Validate要求を観測したとは推論しない。生成Serverのブラウザー操作、Desktop実WebView、実セッション認可/全秘密非漏洩/完成したSource Closure/Lockは別Gate。旧#1849生成UI版の実ブラウザーPASSとも混ぜない。
