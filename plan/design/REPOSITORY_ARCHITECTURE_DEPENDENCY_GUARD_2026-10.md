# Repositoryの依存方向を守る最小ガード（2026-10）

- 状態: 実装候補。正式なRepository Health Evidenceは未取得
- 対象: RQ-023（継続開発・保守性）。Plugin拡張性はRQ-031〜033とも関連
- 作業: GitHub #980
- 観測基線: main `f27c45f718e288b2ee99f1e09eab88f946ca77d4`（2026-10-08）
- 正本との関係: 技術構造の正本は `docs/architecture.md`、開発・統合手順の正本は `AGENTS.md` / `CONTRIBUTING.md`。この文書は依存検査の対象と限界を説明するもので、新しいProduct RequirementやPlugin Contractを定義しない。

## 背景

TEIでは業務上の意味、実装、実行権限、観測結果を区別する。標準機能を含めたPlugin化や、Web・Desktop等への展開を進めても、その区別がソースコードの参照関係で崩れてはならない。

一方、現在は多数の作業レーンが進行している。リポジトリを一括分割したり、全Goパッケージを移動したりすると、機能改善とは無関係な差分によって競合やqualificationの再取得を増やしてしまう。まずは既に成立している小さな依存境界を検査し、後続変更が不用意に逆流させないことを優先する。

## 観測と判断

2026-10-08時点のmain treeには、`internal/` 397 files（production Go 172、test Go 225）、`tools/` 368 files、`plan/` 338 filesが存在した。`go.mod`は単一の `module tei` である。これらはディレクトリ規模の観測であって、コード量や文書量だけから構成不良と判定するものではない。

代表的なGo source importを確認したところ、`internal/core`と`internal/subject`は他のTEI内部packageに依存せず、`internal/permission`はcoreだけに依存する。`internal/semantic`配下ではcore / semanticへの依存に収まり、`internal/stdlib`はcore / semanticだけを参照する。

対照的に、`internal/plugin/model.go`はdatabase/migration、deployment、environment、permission等の型を参照している。これを現時点で単純に「違反」と定義すると、既存のCapability表現を壊す。汎用Capability Contractへの移行判断は #951 とDraft PR #963の担当範囲であり、本ガードには先取りしない。

全パッケージの依存グラフは未取得であり、循環やすべての望ましくない依存が解消されたと主張しない。Go自体が検出するimport cycleと、アーキテクチャ上好ましくないがコンパイル可能な依存は別問題である。

## 防ぎたい失敗

- 共通ID等の最下層へPluginやRuntimeの型が流入し、将来のTarget実装まで引きずる。
- Semantic modelがDatabase / Deployment等の実現方式へ依存し、意味の定義だけを独立に読み込めなくなる。
- `internal/`から実行用`cmd/`や検証用`tools/`をimportして、ReferenceやCLIの実装が製品の依存となる。
- 実装変更に伴う依存の逆流が、レビューでは見落とされ、後の分割・self-build段階まで蓄積する。

## 最小の検査契約

`tools/repository-architecture-check/dependency_test.go`はGo標準ライブラリのparserを使い、`internal/`配下のproduction `.go`を再帰的に調べる。`_test.go`の依存は製品依存と区別する。

| Source | 現段階で許容するTEI内部の直接import |
| --- | --- |
| `internal/core/**` | なし |
| `internal/subject/**` | なし |
| `internal/permission/**` | `internal/core` |
| `internal/semantic/**` | `internal/core` と `internal/semantic/**` |
| `internal/stdlib/**` | `internal/core` と `internal/semantic` |

さらに、すべての`internal/**`から、同じGoモジュールの`cmd`、`tools`、`examples`、`reference`（各ディレクトリ直下のpackageを含む）へのimportを禁止する。モジュール名は`go.mod`から取得するため、将来のモジュール名変更によって検査が黙って無効になることを避ける。Go標準ライブラリや外部ライブラリについては、この検査で新しい禁止規則を設けない。

拒否したimportは元のGoファイルのRepository相対パス、import path、理由を報告する。parseまたはfilesystemの失敗も検査成功として扱わない。検査するローカルimportのprefixは現在の`go.mod`の`module`宣言から取得する。将来module pathを変更しても、以前の`tei/`という文字列のままにして依存検査が無効化されることを防ぐ。

通常の `go test -count=1 ./...` に含まれるため、従来の `tools/repository-health-check.sh` の仕様は変更しない。CLIの成否だけでなく、正しいrepository revisionに対する全検査PASSが必要であるという既存Gateも変更しない。

## 検証の範囲

実装候補はGo 1.23.2の隔離した模擬moduleで単体テストを行った。

- 許容した依存でPASS
- `internal/core`から`internal/plugin`へ不正importを注入するとFAIL
- 不正importを戻すとPASS

これは分類器とsource走査の局所検証に限る。**TEIのactual main / PR head全体のGo test、vet、Workspace validationを通過したEvidenceではない。** PRのmerge前には、対象exact headのtrusted full checkoutでRepository Healthの全項目を実行する必要がある。

## 読み取り専用の依存グラフ

`tools/repository-dependency-report/`に、production Goの直接importをJSONへ出力する診断ツールを追加した。特定の依存方向を設計上の違反と判定する検査とは、役割を分ける。

```sh
go run ./tools/repository-dependency-report -root . > /tmp/tei-direct-import-graph.json
```

出力は、Goモジュール名、対象packageとproduction Go file数、重複排除した直接依存先・直接依存元の数、各依存を記述したファイルの一覧を含む。ソースファイル順に依存しない出力とし、timestampや実行者情報を埋め込まない。過去の履歴と比較する際はGit revisionと実行環境のEvidenceを別に添える。

これはbuild tagを解決せず、alternate targetのファイルも含める「source-levelの潜在的な直接importグラフ」である。compileされる依存グラフ、Runtimeで呼び出された機能、Pluginへの実行権限を表すものではない。内部importの行き先にGo sourceが存在しない等の失敗は、完全な出力として公開せず明示的なエラーとする。

Go 1.23.2の隔離した合成fixtureでは、この診断ツールのunit test、go vet、gofmtの正常終了を観測した。**TEIのfull checkoutに対する実測JSONの取得、およびexact PR headのRepository Health PASSはまだ得ていない。** 構成の良否を未取得の集計値で断定しない。

### 代表的な実ソースから見える境界

main `f27c45f`〜`ff6efce`の確認では、次の直接importが観測できる。

- `internal/plugin/model.go`は`internal/database/migration`、`internal/deployment`、`internal/environment`等の型を利用する。Pluginの共通契約が個別実行領域の型に依存する課題は#951で扱う。
- `internal/database/migration/plan.go`、`render.go`、`postgres_apply.go`は`internal/database/providers/postgres`をimportする。PostgreSQL固有のSQL文法自体はADR-0015に従いProvider側へ移っているが、共通migration packageが具体Providerと結合している事実は残る。
- `internal/database/realization/realize.go`はPostgreSQL/OracleのPolicyを組み立てる。これはADR-0013で意図して導入したcomposition境界であり、単なる具体Providerへのimportを一律に欠陥と判定しない。
- `internal/ir/world.go`は複数の業務モデルをまとめる。集約点であることだけを理由に分解しない。
- `internal/syntax/parser.go`はApplication、Database、Deployment、Plugin、Semantic等の型を直接参照する。また`internal/workspace/loader.go`は現行のsource rootをswitchで読み分ける。これは現時点の実装方式を表すが、DSL自体をPlugin化する際のLanguage Plugin bootstrapと読込み契約はまだ別途実証が必要である。Issue #988 / PR #989の担当範囲とし、本レーンからParserやWorkspace Loaderを改変しない。

現時点でPostgreSQL固有のmigration処理を急いで移動すると、partition観測を必須とする破壊的操作のfail-closed条件を壊すおそれがある。第二Providerの実証で、共通PlanがProvider専用型なしに成り立つ必要が明らかになったときに、render/applyのcomposition責務を分割する。その際は現在のfresh observation・SQL再検査・副作用前停止のテストを維持する。

## 段階的な次の判断

1. この狭い依存ガードをfull Repository Healthへ組み込み、既存mainや並行PRに不意のfailを起こさないことを確認する。
2. #951のCapability Contract移行が成立した後、Plugin descriptorとimplementationの参照方向について別の機械検査を検討する。
3. 全packageのimport graph、変更の集中箇所、Referenceからproductionへの移行例を取得し、必要な制約だけ増やす。
4. 外部利用が生じるまでは`module tei`の置換、多module化、リポジトリ分割は保留する。


## 次段階の依存調査（部分観測・2026-10-08）

代表的なproduction sourceの直接importを追加確認した。これは全package graphではなく、変更判断に必要な局所観測である。

- `internal/ir/world.go` は application、database、environment、plugin、query、realization、releaseplan、semantic など複数領域を集約する。Worldが現在の統合表現として妥当か、Plugin-first拡張時に固定フィールド追加を強いるかは、実際の拡張例で判断する。
- `internal/plugin/model.go` と `internal/plugin/build.go` は core、database/migration、deployment、environment、permission を直接importする。Capability契約の移行前に禁止規則を追加しない（#951）。
- `internal/plugin/runtime/adapters.go` は deployment配下の具体的なexecutor、evidence、observationなどをimportする。実装adapterの配置として意図的か、runtimeが特定targetへ依存する構造かは分離して評価する。
- `internal/database/migration/plan.go` は `internal/database/providers/postgres` を直接importする。DB providerの複数実装を増やす前に、migration planの汎用部分とPostgres固有のrender/applyの境界を評価する。即時のpackage移動は行わない。

次の判断は「新しいPlugin/DB Provider/Targetを一つ追加する際、既存の中核packageの変更が何件必要か」を計測し、具体的な拡張阻害を示すこと。全Go import graphはGo toolchainで `go list -json ./...` 等を使い、生成物として扱う。生成したグラフを意味の正本や永続的な手書き台帳にしない。

## 拡張時変更量の検証契約

依存本数だけでなく、新しいPlugin・DB Provider・Targetを追加するときに既存の中核コードを何箇所変更するかを測定する。

1. 登録・構成：descriptor、binding、workspace等の宣言変更。必要な登録は欠陥とみなさない。
2. 中核モデル・ディスパッチ：`internal/plugin/model.go`、`internal/ir/world.go`、共通resolver等の既存enum・switch・フィールドを編集する必要があるか。
3. 実装Adapter：新しい実装専用のpackage、runtime adapter、provider-specific renderer等を追加するだけで済むか。

測定項目は既存中核ファイルの変更数、実装追加ファイル数、既存テスト変更数、依存方向、実行時の権限・Conformance、Go test・Workspace validation結果。中核変更ゼロだけを合格条件としない。

### 全依存グラフの取得方法

trusted full checkoutでrevisionとGo環境（GOOS、GOARCH、build tags）を固定し、`go list -json ./...`のpackage別`Imports`から同一module内の直接依存を集計する。必要なら`go list -deps -json ./...`で推移的依存を調べるが、直接importの本数と混同しない。入次数・出次数・領域横断importを出し、実装追加時の変更量と突き合わせる。

Plugin capabilityの変更は #951 / PR #963、TEI-native sourceは #957 / PR #969に委ねる。Parser/Workspaceの別作業 #988 / PR #989と重複させない。DB Providerは既存のfresh observationと破壊的操作のfail-closed条件を維持する。

このグラフは一時的な分析成果物であり正本ではない。測定前に広域の禁止規則を追加しない。

## ソース確認による拡張障壁の具体化（2026-10-08）

`internal/plugin/model.go` の `CapabilityKind.Valid()` は7種類の文字列をswitchで列挙している。`Capability` 構造体には `Actions`、`Observations`、`Evidence`、`DatabaseProviders`、`MigrationTools`、`Features`、`Permissions` が型付きフィールドとして並ぶ。このため、未知の能力種別を追加する試行では、まずKindの検証とdescriptorの保存・解釈・bindingを確認する必要がある。列挙にないKindが受理されないことはソース上で確認できるが、未知の能力が別の拡張機構で扱えるかは未検証であり「拡張不能」とは断定しない。

`internal/ir/world.go` の `World` は `Plugins`、`Bundles`、`Semantics`、`Applications`、`Databases`、`Migrations`、`MigrationProviders`、各種binding等を個別の型付きsliceとして保持する。これは明示的な統合モデルとして利点がある一方、新しいDSL全体や独立したモデル種別を追加する際にWorldのフィールド変更が必要になるかを試すべき箇所である。未知フィールドの受理・保持・意味付けはParser/Workspace側の検証と切り離して判断しない。

`internal/database/migration/plan.go` は共通 `planWithRenderer` にrender関数を注入しており、計画組立処理には既に分離の余地がある。一方 `PlanPostgresWithObservation` はPostgres固有の `SchemaObservation` を要求し、破壊的変更の前提確認を伴う。依存の見た目だけを整える目的でこの安全条件を汎用化・削除しない。

**次の具体的な受け入れ実験**： (A) 既存Kindに含まれない架空Capabilityをdescriptorとして宣言し、parse→validate→lock→resolveの各段階でどこまで保持できるか記録する。(B) 既存WorldフィールドにないモデルをDSL pluginで追加する想定で、parser→IR→validation→serializationを通過するために必要な既存ファイル変更を列挙する。(C) 第2のDB Providerで非破壊migrationを計画する際、共通Planを変更せずrenderer追加だけで済むか確認し、破壊的操作の安全条件は別途試験する。いずれも現時点では実験計画であり、成功を主張しない。

## Capability解決経路の追加確認（2026-10-08）

`internal/plugin/resolve.go` の `Catalog.ResolveEnvironment` はPlugin依存を再帰的に解決し、各Definitionの `Provides` を `ResolvedCapability` に転写している。この箇所ではCapabilityKindのswitchによる種類別分岐を行わず、`ResolvedSet.Capability(kind,id)` もKindとIDの一致で探索する。したがって、未知Kindの追加がresolver自体のコード変更を必ず要するという仮説は支持されない。

実際の境界は `CapabilityKind.Valid()` の既知Kind検証、Definitionの検証呼出元、そしてCapabilityごとに型付きの `Actions` 等の属性を参照するruntime bindingにある。次の試験では「未知Kindを含むDefinitionのValidationはどう失敗するか」と「既存Kindで新しいIDを指定した場合はCore無変更で解決できるか」を分ける。Kind追加とID追加を混同しない。

なおこの観測は静的なソース確認であり、実際のworkspace読込・validation・実行のE2E成功を証明しない。関連するPlugin Capability契約レーン（#951 / PR #963）へ検証論点を引き継ぎ、本依存ガードPRに重複したPlugin実装を追加しない。

## Lock・Package経路の確認と試験優先度

`internal/plugin/resolve_test.go` は既存のdeployment-executor種別でCapabilityの解決を検証している。新しいIDだけを追加する独立した回帰試験や未知KindのE2E試験は、このファイル内には確認できない。
`internal/plugin/lock.go` はLock format versionを検査する。`internal/plugin/lock_test.go` には定義の順序を入れ替えた際のfingerprint不変性と旧Lock format互換性の試験がある。Capability種別の拡張を検討する際には、単なるJSONの往復だけでなく、Lockのfingerprintと旧形式の読込互換性を守る必要がある。
`internal/plugin/package.go` の `MaterializePluginPackage` はDefinitionとBuildPlanのfingerprint一致を要求し、正規化したDefinitionをmanifestに格納する。未知Kindがこの経路を通る場合は、定義の検証とfingerprintの意味を同時に検査する。

**検証の優先順**： (1) 既存Kind・新IDでResolveEnvironment→Capability取得、(2) 未知KindのDefinitionの検証箇所特定、(3) Lock/Packageの正規化・fingerprint・旧形式互換、(4) 実行権限とAdapter dispatch。各段階の結果は成功・失敗とその位置を記録し、全経路を通す前に「任意Capability対応済」と宣言しない。

## 非目標

本ガードはRuntime Permission、Plugin Authority、Provider Conformance、外部実装のsource identityを検査しない。また、Plugin packageとphysical process / Wasm artifactが一対一であることを要求しない。Source importがきれいでもRuntimeの振る舞いまで安全・適合済みとはいえない。
