# プラグイン構成研究の追試と責務の整理

- 確認日: 2026-10-09
- 状態: 評価待ち。既存Referenceの追試とProductionへ引き継ぐ範囲の提案。
- 関連研究: [DeepSeek HarnessからTEIへ取り入れる設計](PLUGIN_COMPOSITION_HARNESS_RESEARCH_2026-10.md)。外部資料とTEI要求の対応は同書を参照する。
- 正本との関係: 要求、採用済みADR、実装状況を変更しない。本書の試験結果だけではPRの統合やProductionへの導入を承認しない。
- 対象要求: RQ-031 / RQ-032の契約による拡張、RQ-033の構成説明、RQ-003の正本重複の回避。RQ-019 / RQ-020の実行権限と外部副作用の扱いは既存Runtimeの責務として保持する。

## 追試の目的と対象

TEIでは、Harnessと共通する考え方を既存の研究で検討している。別のPlugin Frameworkを作る前に、その研究がどこまで成立しているかを確かめた。Capability Contractについては既存Referenceの試験を再実行し、構成説明については実ソースを観測したうえで、異なる種類の変更を比較した。

| 対象 | 固定したcommit | 用途 |
| --- | --- | --- |
| TEI `main` | `fbcf4b8fc02e205ec624cd68a39b6c642c4d34b9` | 変更比較の基線 |
| PR #963 | `b2172847990bf9bea71971a2709fa02408e81543` | Capability ContractのReferenceと責務分類 |
| PR #1020 | `ccc3a8c70209969bd4ff07375189ad0a365d770d` | 構成観測器と既存ソースの棚卸し |

確認時点では、`main`にあって#963にないcommitが680、#963だけにあるcommitが3だった。#1020ではそれぞれ10と6である。この数はGit履歴の差を示し、変更規模や品質を評価するものではない。各PRの固定したcommitで試験が成功しても、現在の`main`へ統合した結果が成功するとは限らない。

Windows上のローカルRepositoryからWSL Ubuntuへcloneし、上記commitをdetached HEADでcheckoutした。Goの試験は既存の`golang:1.25-bookworm`コンテナで、uid / gid 1000として実行した。rootではファイル権限に関する試験の条件が変わるためである。二つのPRの元commitと、後述する修正後のcommitでは、実行前後のGit状態がcleanであることを確認した。比較用のコピーには意図した差分だけを加えた。

今回の試験は探索段階の基線確認である。#963のE14が要求する、責務分類の固定後に行う統合用のqualificationとは区別する。

## Capability Contractから残す最小要素

PR #963の[ISSUE-000080](https://github.com/hat47x/tei-platform/blob/b2172847990bf9bea71971a2709fa02408e81543/plan/issue/000080-extensible-plugin-capability-contract.md)には、E14としてReferenceの機能追加を凍結し、責務の整理・圧縮・Productionへの最小要素の抽出へ進む条件がある。今回もその条件に従う。以下の型・関数名は当該headの候補実装を特定するためのもので、そのまま公開APIへ昇格させる提案ではない。

候補実装は、同じcommitの[ResolverとLock](https://github.com/hat47x/tei-platform/blob/b2172847990bf9bea71971a2709fa02408e81543/tools/reference-plugin-capability-contract/main.go)、[legacy宣言からの導出](https://github.com/hat47x/tei-platform/blob/b2172847990bf9bea71971a2709fa02408e81543/tools/reference-plugin-capability-contract/legacy_projection.go)、[既存Executor選択との照合](https://github.com/hat47x/tei-platform/blob/b2172847990bf9bea71971a2709fa02408e81543/tools/reference-plugin-capability-contract/runtime_selection.go)、[Production Runtimeへの接続](https://github.com/hat47x/tei-platform/blob/b2172847990bf9bea71971a2709fa02408e81543/tools/reference-plugin-capability-contract/production_execution_bridge.go)を確認した。

| 区分 | 候補となる型・関数 | Productionへの引き継ぎ方 |
| --- | --- | --- |
| #951の中心 | `Contract`、`Requirement`、`AuthorityScope`、`Provider`、`ConformanceEvidence`、`Wiring`、`ResolveInput`、`Resolution` | 契約の意味、実装のidentity、適合根拠、明示選択を保つ。Providerを増やすたびにCoreのCapability kindを増やす方式から移行する |
| #951の中心 | `resolveCapability`、`CapabilityResolutionLock`、`buildCapabilityResolutionLock`、`verifyCapabilityResolutionLockReplay`、`verifyCapabilityResolutionLockEvidence` | 新規の解決、保存したexactな選択の再現、適合根拠の現在性を分ける。既存Plugin Lockの権限を置き換えない |
| #951の中心 | `resolveCapabilityWithAcceptedContractAuthority`、`verifyCapabilityResolutionLockAcceptedContractAuthority`、`verifyCapabilityResolutionAcceptedConformance`と、それぞれのbinding型 | 契約を定義できる主体と、適合を証明できる主体を検査する。ReferenceのGrantや文字列fingerprintを実署名・信頼根の代用品にはしない |
| 移行時の接続 | `LegacyCapabilityMapping`、`projectLegacyCapabilityProvider` | 既存のtypedなCapability宣言から一方向に導出する。新旧の宣言を別々の正本にしない |
| 既存Runtimeとの接続 | `RuntimeCapabilityBinding`、`RuntimeCapabilitySelectionBinding`、`ProductionExecutionBridge`と、その照合関数 | 既存のPlugin Lock、Contract Query Executor選択、`EnvironmentPlan`へ照合する責務だけを残す。Provider選択・認可・起動を二重に実装しない |
| 担当へ返す研究用実装 | `ProviderBuildRealization`、Target packaging関係 | Source → Build → ArtifactはISSUE-000079、Target適合はISSUE-000026 / RQ-024へ返す。#951の常設Build基盤にしない |
| 担当へ返す研究用実装 | Execution Profile / Grant、Resource、Lease、Sandbox、process実行のfixture | 既存Plugin Runtimeの担当へ返す。一般契約の研究packageを第二のRuntimeへ育てない |
| Consumer側の責務 | Xead / XeafのSource Adapter、業務結果・failure・unknown・retry | Source Adapterは#950、業務処理はISSUE-000009 / RQ-018〜020へ返す。Resolverが業務成功を判定しない |

### 圧縮の前に守る条件

`runtime_binding.go`、`runtime_selection.go`、`production_execution_bridge.go`は、同じ選択を別々の段階で照合している。照合の重複だけを理由に削ると、保存済みの選択を現在の権限として扱う危険がある。まず「何のidentityを照合するか」「どの時点のAuthorityを確認するか」「誰が実行するか」を揃えてから、重なる処理をまとめる。

`normalizedLegacyCapabilityFingerprint`は、既存の`plugin.NormalizeDefinition`を使う前にCapabilityを複製している。呼び出し元の宣言を変更しないための処理なので、単にJSON往復を削ることは勧めない。一方、Referenceの`fingerprint(any)`はJSONのstruct field順に依存する方式を自ら宣言している。既存Lockのcanonicalizationとの違いを残したまま、共通の公開fingerprint関数へ統合してはいけない。

各試験で繰り返すfixtureの生成処理は、まとめられる可能性がある。ただし、Contract、Artifact、Target、Evidence、現在のAuthorityのどれを壊した試験かが読み取れる形を保つ。本追試では実装を圧縮していない。削減量や保守工数の改善も測っていない。

### Production化前に残る停止条件

E13で指摘されているとおり、`Requirement.CompatibleMajor`は数値上の版の範囲しか定めない。`compatibleContracts`は同じmajorの契約を候補にし、高い版から探索する。正当なpublisherが意味を壊すminor版を公開した場合、publisherの権限確認だけでは既存Consumerとの互換性を証明できない。

新規の解決や更新判断をProductionへ持ち込む前に、互換性を示す検証可能な根拠、またはConsumerが受理した基準契約のfingerprint等が必要である。Lockから選択を再現する場合は、引き続き固定した契約を使う。この不足を、試験成功や同じmajor番号で補わない。

## 構成説明で比較する変更

PR #1020の[E1観測器](https://github.com/hat47x/tei-platform/blob/ccc3a8c70209969bd4ff07375189ad0a365d770d/reference/application-composition/e1-observe.mjs)を、そのPR自身の実ソースと`main`基線へ実行した。E1は14か所の文字列とファイル内容のSHA-256を記録する。parser、Providerの適合検査、認可、Runtimeの稼働確認ではない。

次に`main`基線から独立したコピーを作り、以下の変更を加えた。UIとDB接続の差を観察するための実験であり、新しい案件構成schemaやResolverは追加していない。UIの確認には、Reactの要素生成を模した実装（fake React）を使った。

| コピー | 加えた変更 | 既存機構で確かめること |
| --- | --- | --- |
| 基線 | 変更なし | E1の観測と既存Workspace検証の基準 |
| UI | `project-extension.mjs`の見出しを`Details`から`Order details`へ変更 | UIファイルの差だけを検出する。既存UI試験とfake Reactの要素構造で変更箇所を確認する |
| DB接続・同じ定義 | developmentに`definition sales_core` / `provider postgres`の`sales_core.preview` Instanceを追加し、`order_entry.sales`の接続先を変更 | 既存Environment宣言で接続変更を表し、Workspace検証で静的整合を確認する |
| DB接続・異なる定義 | `order_entry.sales`を`master_data.dev`へ接続 | 不適切なDB定義への接続を既存Workspace検証が拒否するか確認する |

DB接続・同じ定義の操作は、一つの接続変更に対してInstance追加とBinding変更の二か所を編集する。物理DB、credential、接続endpointを用意した実験ではない。同じProviderを使う例なので、PostgreSQLから別製品への交換も証明しない。

## 実行結果

[検証記録](../../reference/plugin-composition-research/2026-10-09/validation.json)に、対象commit、実行環境、コマンド、終了コード、観測値、意図した差分を保存した。Goは`go1.25.14 linux/amd64`、Node.jsは`v20.20.2`。コンテナimage IDも記録している。

| 対象 | 結果 | 証明した範囲 |
| --- | --- | --- |
| #963の限定試験 | サブテストを除く103試験が成功。失敗0 | 当該ReferenceのResolver、Lock、Authority照合、接続の試験。合成fixtureを含む |
| #963の全体健全性検査 | 4項目すべて成功 | 固定headでのgofmt、vet、Go全試験、Reference Workspace検証 |
| #1020のE1試験 | 4試験が成功 | 合成fixtureでの文字列観測、欠落・変更、経路逸脱の扱い |
| #1020と`main`の実ソース観測 | それぞれ14/14か所を観測 | 指定した文字列の存在とファイル内容の識別 |
| UI変更 | 既存UI試験9件と、変更した見出しの要素構造の確認が成功。Workspace検証も成功 | fake Reactによる構造と静的Workspace整合。実ReactDOMやブラウザー表示は未検証 |
| 同じDB定義への接続変更 | Workspace検証が成功 | 同じ`sales_core`定義を持つ別Instanceへの静的接続。実DB接続は未検証 |
| 異なるDB定義への誤接続 | Workspace検証が拒否 | `DATABASE_BINDING_TARGET_MISMATCH`と`SQL_DIALECT_TARGET_MISMATCH`を検出 |

基線、UI変更、同じDB定義への接続変更のWorkspace検証には、過去のReleaseが現在は存在しないContractを参照するという共通の警告が3件ある。終了コード0と末尾の`ok`を確認した結果であり、警告ゼロという意味ではない。Goの最小対応版での試験や、全試験で外部DBへ実接続したことも主張しない。

追試する場合は、表のcommitをそれぞれ別のtrusted cloneへcheckoutし、検証記録の`commands`にあるコマンドを実行する。全体健全性検査はRepository rootで、`--fast`を付けずに行う。変更比較には`main`基線のコピーを使い、検証記録の`e2.variants`にある各`intentional_patch`を別々に適用する。観測器は#1020の固定版を使い、引数として各コピーのrootを渡す。DB誤接続の検証終了コード1と、二つのDB変更の観測終了コード1は、期待した結果である。

### #1020で見つけた不備と修正

元のheadで全体健全性検査を実行すると、`TestMarkdownHasNoEscapedBackticks`が失敗した。`PROJECT_PLUGIN_COMPOSITION_DESIGN.md`と`requirements-prompt.md`に、インラインコード用バッククォートの不要なエスケープがあった。

既存の`design/application-composition-profile`ブランチで2文書の8行を修正し、ローカルcommit `7ccbbb8e53d9f52abba385d04925e66c7758c65a`へ記録した。日本語本文を通読し、意味や識別子を変えずに表記を直した。このcommitのtrusted cloneで全体健全性検査4項目がすべて成功し、実行前後もcleanだった。GoとJavaScriptのコードは変更していない。修正はpushしておらず、公開PRのheadは元のままである。

### 変更比較から分かったこと

UIの変更では、E1の14項目はすべて「観測できた」のままだったが、`order.project-ui-extension`のファイル内容のSHA-256だけが変わった。Git差分もUI拡張1ファイルに限られ、意味のSource、Application、Plugin Lockは編集していない。ただし、Sourceが変わっていないことだけでは、実UIの動作や業務結果が同じだとまでは言えない。

二つのDB接続変更では、E1はいずれも`order.dev-binding`を`marker-missing`として未解決にした。観測器が探す元の接続文字列がなくなったためであり、妥当な変更と誤接続を区別した結果ではない。その区別は既存のWorkspace検証が行った。

DB変更では`order.provider-bundle`のSHA-256も変わったが、Bundleの指定自体は変えていない。Bindingと同じEnvironmentファイル全体からSHA-256を計算するためである。ファイル内容の差を、そのまま契約・依存・権限の意味上の差とみなしてはいけない。

以上から、構成説明の入口には既存Sourceの参照と内容識別を再利用できるが、診断はSourceの所有者である既存検証器へ戻す必要がある。E1を万能の構成Resolverへ拡張する根拠にはならない。

| 説明したい情報 | 今回使えた既存の場所 | 今回の不足 |
| --- | --- | --- |
| UI差分の所在 | Project-owned UI拡張とGit差分 | UI Kitの正式な採用記録、親版との互換性、実ブラウザーでの挙動 |
| ApplicationとDBの接続 | Application宣言、EnvironmentのInstance / Binding、Workspace検証 | 物理接続の成功、実Providerの適合、運用時の状態 |
| Providerの選択とidentity | 既存Plugin Lockと#951の契約研究 | Capability ContractをProductionへ組み込む作業、E13の意味互換性 |
| 実行可否 | 既存RuntimeのPermission / Resource / Lease境界 | 今回の静的変更比較では確認していない |

この小さな比較では、新しい構成manifestが必要だとは立証できなかった。正式なUI採用・互換性の情報が必要になる可能性は残るが、既存の宣言へ参照を加えれば足りるかは未確認である。SUIについて実行したのはTEI側のStage 0にある4観測だけで、SUIの実Canvasを第二Consumerとして統合した実験ではない。

## 引き継ぎと適用判断

Harnessからまず取り入れるのは、契約・実装・利用側を分ける考え方と、選択した構成を説明できる仕組みである。[公式Architecture](https://github.com/deepseek-ai/deepseek-harness/blob/5badb15009ae1756c3afe0ae0cef1faafc290ccc/docs/architecture.md)と[app-boot](https://github.com/deepseek-ai/deepseek-harness/blob/5badb15009ae1756c3afe0ae0cef1faafc290ccc/packages/boot/app-boot/README.md)を同じcommitで再確認した。TEIでは既存のSource、Environment、Lock、Runtime Authorityへ接続する。共通Contextや新しいPlugin runtimeを先に必須化する根拠は、今回の検証対象にはない。

#951への引き継ぎ案は、上記の最小要素と周辺責務の分類をレビューし、E13の互換性条件を満たしてからProductionへの移行範囲を決めることである。#1016では、既存Sourceへの参照と各検証器の診断を組み合わせた構成説明を先に評価する。実行中の登録交換、外部副作用のunknownからの復旧、OSの隔離、実React Canvas、利用者の変更負担や性能は別の実証が必要である。GitHub上のIssueやPRへの投稿・更新は行っていない。
