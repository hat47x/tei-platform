# TEI Studio PoC B — 静的合成の最小参照実装

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

生成UIファイルは `studioUIContributions` の **名前付き2要素の配列**です。別のRuntime RegistryやEvent Busではなく、#3が所有するcomposition rootへ渡すための静的import一覧です。現状のimport先 `reference/tei-studio-graph/graph.ts` と `reference/tei-studio-command/command.ts` は並行するDraft PR #1828 / #1829 の資産であり、対象コミットと依存版を固定し同じbuild内へ取り込むまでは **UI生成結果のimport・起動を確認できません**。#3と登録入口が決まるまでは手書きのcomposition rootを変更しません。

## 拒否と不成立条件

Go側は未知のサービス、重複選択、互換しないexact version、異なるscope、不完全なfixtureを拒否します。構造的な契約依存の適格性は既存Registryが担当し、選択結果だけで実行権限を得たとは扱いません。UI側は既存Kit Resolverの未知implementation・重複/未知slot・version/framework不一致を利用して拒否します。ファイル列挙順や曖昧なfallbackで順序を決定しません。

現時点でこの独立Referenceが**証明しないもの**は、#2の通常ビルドとの実サービス結果の一致、#3の画面との配線、2 UIモジュールを含む実Bundlerの成功、Wails Web/Desktop実機、Browser session隔離、任意プロセスロード、完全なArtifact/Build Lock来歴、製品用TEI宣言の一般解決です。これらは「未実施・依存待ち」とし、PoC B全体の合格とはしません。

#7の独立黒箱検査は、本Referenceで使うfixtureと生成結果の内容・拒否/来歴を参照できますが、#7固有の検査器・負例fixture自体は#7の担当のままです。
