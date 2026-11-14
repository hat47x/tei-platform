# TEI Everything is Data — 最小Bootstrap・Role解釈循環・利用資格の接続計画（2026-11-02）

**状態：横断設計候補／実TEI Core・DSL・Role Resolverへの採用前。** 本計画の設計上の中心は、標準データ構造でRole/Operator/構文Plugin自身を記述可能にしながら、**その自己記述を初めて読むための有限な解釈の根**を明示し、循環と未決、実行資格の区別を機械検査できるようにすること。

**現時点の読み方（2026-11-14）**：第1〜39節は設計判断と過去の試行を記録したものであり、すべてが製品に実装されたわけではない。JSON Schemaなどの外部定義を図解・診断へつなぐ最初の実装は[第41節](#41-2026-11-14-外部の宣言を読みやすく投影し安全に編集するための実装順)に、業務の意味から説明・図・Mock検証・実行へ展開する際の境界は[第42節](#42-2026-11-14-業務の意味から説明図解実行を展開する自己記述dslの最小実装境界)に整理した。さらに、[第43節](#43-2026-11-14-意味を保つ対応付けを最初の小さな実装と受入へ収束する)ではGraph／TreeとJSON Schemaの対応付けを、一つの試験へ落とす方法と担当境界を定めた。[第44節](#44-2026-11-14-cliを起点とする意味あるコード生成とpluginの自己構築)では、図表・Ruleから実際の処理を生成し、Plugin自身の自己構築と定義の簡素化を測る最初の検収を整理した。[第44.7〜44.8節](#447-生成pluginの機能と説明を同じ定義から辿る)では、Plugin文書を実生成したGraph/Ruleから発見・検証する最小実証を扱う。最優先は引き続き**TEI Studio Desktop自身によるStage2構築**であり、外部Schema対応や構文自己記述を新たな前提条件にはしない。

- **横断Owner**：[Everything is Data #2619](https://github.com/hat47x/tei-platform/issues/2619)、[標準構造・意味Role #2620](https://github.com/hat47x/tei-platform/issues/2620)。
- **実装Owner**：[DSL Language/Operator Plugin #988](https://github.com/hat47x/tei-platform/issues/988)、[Plugin/Service Contract #951](https://github.com/hat47x/tei-platform/issues/951)、既存Semantic Extension Lock／Source/Accepted Meaning。独立のParser Registry・Extension Resolver・実行Hostを新設しない。
- **Consumer**：[Markdown/Source #2621](https://github.com/hat47x/tei-platform/issues/2621)、[Facet/Law #2544](https://github.com/hat47x/tei-platform/issues/2544)、[Product Skill #2605](https://github.com/hat47x/tei-platform/issues/2605)、[将来Skill #2614](https://github.com/hat47x/tei-platform/issues/2614)、[構造保存 #2497](https://github.com/hat47x/tei-platform/issues/2497)、[Studio #1124](https://github.com/hat47x/tei-platform/issues/1124)、[Context #2516](https://github.com/hat47x/tei-platform/issues/2516)。
- **系列の先行計画**：[Draft PR #2610](https://github.com/hat47x/tei-platform/pull/2610)／[#2619 EI-01〜08](https://github.com/hat47x/tei-platform/issues/2619)。本書は**EI-01の上流にある役割記述自体のBootstrapの実証差分**に限る。別の統合受入Gateではない。

## 1. 自己記述可能性と自己実行可能性を分離する

「Everything is Data」は、Role Definition・Rule・Function・Syntax Pluginの説明を同じデータ構造で記述できることを目指す。ただし任意の記述を、それ自身の「これを信頼せよ」指示によって読み込み／実行することはしない。

```text
Layer 0   既存Workspace/Extension Lockに基づく入力・Pinの確認
          + 有限な基本値・Record/Sequence/Refの読取り（既存基盤）
                    ↓
Layer 1   Pinで選ばれたSyntax/Role/OperatorのProvider実装を
          既存Resolverで特定し、要求Shape/依存/適用条件をデータとしてInspect
                    ↓
Layer 2   Source occurrence → desugarされたplain standard data
          → 明示Role binding → typed meaning/validation/qualification
                    ↓
Layer 3   CUI／Studio／Skill／SHO／Targetへの用途別Projection
          + 外部HostのAuthority／実Effectは独立した実行時検証
```

**Layer 0は自己記述データを成立させるための有限な実装上の出発点**。現行の既存Workspace reader、標準基本型、Extension Lock、既存Plugin解決への最小契約として検証し、新しいUniversal Meta-Engineを構築する意味ではない。実際の初期Language/Semantic Setを含む正確な既存経路と、PoC用記号を対応付けてから採用を決める。文字列`role: trusted`や`effect: pure`は自己申告であり、Trusted Provider/Host Grantを生成しない。

## 2. 循環を二種類に分ける

### 2.1 解釈に先立って確定が必要な依存（eager）

例：Role-Aの`required shape`を読むためにRole-Bの**評価完了**が必要であり、同時にBの評価にはAの完了が必要な場合。rootから到達したeager閉路は解釈の根が得られない。

この場合は、既存Plugin ID/versionと依存経路を使って循環を診断し、ロード順・ファイル順・AI推測で一方を選ばない。固定点計算を採用する場合は、その意味・収束条件・失敗時の振る舞いを**該当Operator Owner**の明示契約として別途評価する。初回Referenceではeager閉路を不成立とする。

### 2.2 型・関係の遅延参照（deferred）

例：`Node.children: Sequence<Ref<Node>>`、Tree/Graphの循環辺や、互いを参照する二つのData Model。これは宣言同士の参照であり、直ちにそれらの**値を再帰評価して確定する要求**ではない。

既存Typed Ref/Stable IDへ解決できる遅延参照は、**解釈循環とは別に**成立し得る。データ値を辿る際のtermination、Visited Set/深さ制限、結果の部分取得/未知は利用するOperation・Consumerが定める。「すべての循環参照を拒否する」設計を採用しない。

### 2.3 相互依存の表示

一つのRoleに複数Roleを適用できること、同じ名称で異なるProviderが候補になること、演算`combine`を持つ二つのMonoidを使い分けることを別に扱う。同じ名前・同じCarrierでの自動選択は避け、Use-siteが要求する**Role + Operator ID/version + binding + scope + evidence**で比較する（数学的公理の正本は#2544）。

## 3. 構文糖衣の同一性をBootstrapの証拠へ還元する

- Markdown/DOT/Goa風/MukuroL風/Chain/JSONは各Syntax Pluginが**plain data候補**に変換。Source文書そのものと、変換結果、受理済みMeaningを区別（#2623/#2625）。
- 引用文中の`role`や`op`は**Source occurrence中の値**。Project Profileと明示Role bindingがない限り、実行定義のCandidateとならない（#2621）。
- 同じData byteでもRole/Operator/Syntax PluginのPinが変われば**意味資格の再評価候補**になる。Source byteが変わらないこと、Role名が同じこと、独立Fixtureで同じ値が出たことは、旧Meaning Acceptanceや現在のHost Grantを引き継ぐ根拠にならない。
- 構文／RoleからSkillやTool説明を生成できる場合も、そのSkillが指す**実CLI/API/MCP入口・版**を別に検査（#2605）。未導入Providerの`effect: read-only`宣言で、秘密Sourceの外部開示や自動実行資格を付与しない。

## 4. 独立Reference：RB-01〜RB-24（有界合成の実行済み、実TEI試験はNOT RUN）

初期は`plain data候補 + pinned descriptor候補 + eager/deferred依存`を有限な入力として**read-onlyで**比較する。独立Fixtureの出力Schemaを正式なCore Meta Schemaへ昇格させない。

| Case | 入力 | 期待する判別 |
| --- | --- | --- |
| **RB-01** 最小起動 | Role providerなしでRecord/Sequence/Refを読む | データとしてinspect可能、`Function`のMeaning qualifiedとはしない |
| **RB-02** 決定性 | 同じPin/Source/依存を別列挙順で読む | 同じ候補／診断、file load orderによる採用差なし |
| **RB-03** 未解決Provider | 指定Role Provider ID/versionが存在しない | missing/unsupportedを返し自動導入・自動採用しない |
| **RB-04** 同名異ID | Role名同一でProvider ID・版が異なる | explicit binding不足ならambiguous、first-match採用しない |
| **RB-05** 解釈eager-cycle | A→B、B→Aが双方評価完了を必要とする | 読み込み完了を捏造せず閉路と依存経路を報告 |
| **RB-06** 合法なrecursive Ref | Node.children→Ref<Node>を遅延解決 | eager評価閉路扱いせず、利用時の走査契約を分離 |
| **RB-07** 複数Role | 同一データに独立に適合するModel Roleと説明Role | 各Roleと根拠を保持、片方の採用から他方を実行しない |
| **RB-08** 引用と未信頼文書 | quote中の`trusted-script`／`run-as-admin` | Role実装の信頼／Tool命令／Host権限を生成しない |
| **RB-09** Plugin改版 | Source一致、Role/Operator v1→v2 | 解釈上の資格を失効・再照合対象にし、旧受理を復活させない |
| **RB-10** PureとEffect | `pure`宣言が外部writeを含むProvider | 宣言からpure保証せず、Provider契約と実Targetを検証対象へ |
| **RB-11** Source occurrence | 同じ構造の定義表・Expected・引用、場所は別 | 意味同値候補と編集元Occurrence/Source locatorを分離 |
| **RB-12** Skill/Harness | Role Descriptorから未知コマンドを生成 | 名称解決だけでToolを実在・実行許可・配布済みと認定しない |

### 4.1 依存先もPinする追加負例と現在の実測

最初の12ケースに、初期bootstrapの**根・未受理Role・依存先ピンの扱い**を追加した。合成JS Referenceの識別子や`source.active`は外部で既に分類済みとした**Fixtureの入力**であり、TEIがそのフィールドを見てSourceの信頼性や役割採用を判断する仕様ではない。`requested=[]`でもDataはread-only inspectでき、Roleが解決されていないことを示す。

| Case | 判別すること |
| --- | --- |
| **RB-13** 指定Roleへのpinがない | そのRoleを採用せず`missing-or-stale-pin` |
| **RB-14** eager依存とdeferred参照が混在 | eager依存先は評価し、遅延Refの循環のみで不成立としない |
| **RB-15** Role required shapeを満たさない | 本文値の書換えで補わず未解決 |
| **RB-16** 参照Roleのみpinされ、eager依存先が未pin | **依存先まで版照合が必要**。暗黙に評価可能とみなさない |
| **RB-17** eager依存先も正しくpin | 当該限定shapeではRole候補として検査可能 |
| **RB-18** 同じProvider IDの重複 | 同名・同ID重複は不成立、先勝ち/後勝ちをしない |
| **RB-19** 存在しないeager依存先 | rootが有効でも依存先がなければ未解決 |

**実行済みの限定証拠**：`plan/fixtures/role-bootstrap/check_bootstrap.js`の独立合成Referenceを、コード実行環境のV8 JavaScript isolateで当初**19/19 PASS**、Contract/実装identity比較を加えて**24/24 PASS**。GitHubへ保存し、保存版を再取得して再実行・SHA照合する。**実Nodeテスト、GitHub CI、TEI Language Plugin/Operator/Extension Lockの実結合はこの試行では未実行**。そもそもFixtureの`registry`は、実TEIのPlugin resolverを模擬している小さな比較器であり、新Resolverのプロダクション実装を提案しない。

この有界Referenceが実証できるのは、指定したデータ・ピン・eager/deferred有界例を区別する**合成アルゴリズムの判別**だけ。Roleの意味付与、可逆Source操作、循環した実行型の一般的許容性、正式Trusted Provider契約、Runtime/Hostの権限、Owner意味受理は証明していない。

### 4.2 Exact implementation identityへの強化（RB-20〜24）

**適用範囲の注意（2026-11-03）**：この節で比較するPinは、選択したRole実装とその起動時依存について、特定の実装を再現・照合するための候補である。業務定義から別の定義を参照するたびに、参照先の内容ハッシュをSourceへ記述する要件ではない。通常の定義間の関係は、既存の識別子・スコープ・型付き参照で扱う。実装の選択や版固定が必要な場合も、定義ごとに手入力させるのではなく、既存のWorkspace／Extension Lock等に集約できるかを先に確かめる。

本Referenceは、再現を要求する利用場面と、編集中の参照解決・表示を対照していない。両者を同じ必須条件にするかは未決であり、RB-24の「version-only Pin拒否」はこの合成試験の期待に限る。実際に同版の異なる実装が意味を変える例、既存Lockによる固定、差替え・更新・編集の負担を比較し、追加のFingerprintが必要な境界と不要な境界を決める。成果物の完全性確認や実行時の権限検査を省略するという意味ではない。

**既存APIの限定対照**：`internal/plugin/logical_reference_lock_test.go` は、`modelref.Ref` を変更せず、環境のPlugin Lockを旧版と新版に切り替え、それぞれ指定した版を解決できることを検査する。同じ版のPlugin定義を変更すると旧Lockでは拒否されることも確認する。`modelref.Ref` と `plugin.Lock` を一つの試験から別々に呼んだ結果であり、Role解釈とLockが製品として統合されたこと、builtin実装のバイト列の照合、実際の編集負担や安全性を示すものではない。

**実CLIによる限定対照**：`cmd/tei/reference_lock_value_cli_test.go` は一時Workspaceに同じ型を3回参照する小さなモデルと、builtin Plugin・環境を作り、実CLIをビルドして起動する。型の基底を変更しても3参照を書き換えずに検証でき、モデルのWorld表示は変化した。別の属性を追加した場合もPlugin Lockは変わらず、Sourceの移動も検証できた。存在しない型への参照は診断された。Plugin定義の変更は旧Lockで拒否され、Lockを一つ作り直すと解決でき、モデルの参照は変わらなかった。

比較案として、各参照に型宣言の**生バイト列のハッシュを直接保存する**方式を仮定すると、この型変更で3箇所のPinが古くなる。実際に算出した変更前後のSHA-256が異なることは試験するが、その方式の実装、更新の自動化、入力回数・時間や実作業の負担は測っていない。正規化した意味の指紋や間接参照では結果が異なり得る。実CLIが確認するのは特定の合成WorkspaceとPlugin定義の範囲であり、異なる実装バイナリの照合、正式Role解釈、実Provider実行、データ移行の妥当性は確認していない。型変更後に参照が解決することは、変更が業務上安全であるという意味ではない。

### 4.2.1 価値と照らした境界ごとの暫定整理

次は[製品価値](../../product/value/TEI_PRODUCT_VALUE.md)と[RQ-008](../../requirements/TEI_PRODUCT_REQUIREMENTS.md#rq-008-capability単位で標準実装と固有実装を置き換えられること)を基準とした限定整理であり、要求の改定や安全条件の撤回ではない。

| 場面 | 現在の確認と価値 | 見直す範囲 |
| --- | --- | --- |
| 定義の編集・参照 | 上記CLI試験では、型の参照は安定IDで解決し、3参照への内容ハッシュの手入力を要しなかった | 通常の関係を読むだけの操作に、参照先の固定バイト列を必須とする根拠は未確認。曖昧・欠落・型不一致の診断は保つ |
| 外部Workspaceの意味Import | `internal/workspace/import_digest_pin_test.go` ではnamespace単位の `semanticDigest` は任意。固定時は意味変更を拒否し、Source移動・空白・コメント・正規化される属性順の変更や別namespace追加は許す | 個別参照への生byte Pinとは異なる既存の選択肢。固定なしでは提供元の変更を受け入れるため、変更影響の確認や受理を省略できない。Digestは推移依存を固定せず、不一致は読込み時の失敗なので後述の候補診断でも回避しない |
| Pluginの選択 | `plugin.Lock` は環境ごとに選んだPluginの版・定義Fingerprintを保持する。上記試験で定義変更を旧Lockが拒否した | 同一試験で無関係な業務型の未解決が `plugin-lock` と `plugin-resolve` の両方を停止した。既存のGo resolverは同じWorkspaceとLockからPlugin定義を解決できた。read-only診断の継続を検討するが、検証済み構成・実行許可として返してはならない |
| 意味に影響するEvaluator | RQ-032はAccepted Semantic SetをRuntime環境のPlugin選択と区別する。`tools/reference-semantic-extension-catalog` のReference Planは対象SetのEvaluator bindingを固定し、未解決Setや不一致・Conformance欠落を拒否する | この条件はAccepted意味に影響する評価の候補に対するものであり、全Pluginや日常の参照に意味依存を強制する根拠ではない。Planが要求する全bindingを局所化できるかは依存閉包と受理条件の実証が必要なため保留。Reference内の入力改変はEvidenceを出さず拒否するが、実Runtime Pluginの分離や信頼性を証明しない |
| Build・配布 | RQ-008は外部Pluginの配布物の内容Digest確認を要求する。`internal/plugin/build.go` のBuildPlanは選択したPlugin、Source、Builder、Target等を固定し、成果物Lockとは別である | 通常の定義編集や定義間RefにBuildPlanを必須とする根拠ではない。Plan v1はSourceのLocator変更も別Fingerprintとする。下記の実コンパイラ比較では位置の影響はBuild条件で異なるため、固定の一律緩和は保留し、成果物の同一性確認も削らない |
| 変更影響 | RQ-008はConsumerと再検証範囲の追跡を求め、依存の欠落を影響なしと扱わない。`internal/modelref` は渡されたGraphと変更を検査し、候補を返す | Graphの構造が正しく影響候補が空でも、動的参照や未収集のConsumerがない証明にはならない。両Review APIの `authorityLimits` に網羅性未確認を明記し、追加収集や再検証の要否は用途別に判断する |
| 定義変更の差分 | Reference用 `tools/internal/changeanalysis` はApplicationの意味名前空間の利用宣言を比較する。既存の利用判定は名前空間IDの所属を見ている | 宣言順だけでApplication自体をchangedとする過剰な候補を避けるため、比較用コピーの `SemanticUses` のみ整列する任意の候補APIを試作する。重複は消さず、実行時に宣言順を保つValidationは整列しない。既存の厳密な `DiffApplications` はShared Acceptanceにも使われるため変更しない。候補APIは受理用ではなく、Source順・Targetへの影響と依存の網羅性が未評価であることを結果に含める。既存の受理経路には接続しない |
| 実行・外部作用 | 現在の権限・対象・実際の結果は、定義参照の解決や過去のLock一致とは別 | 権限確認、誤対象の防止、必要な結果照合を読み取りの利便性と引き換えに外さない |

Runtime側の `Registry.InvokeWithProvenance` は選択したCapability、Plugin定義・artifact、権限・必要なResource leaseを扱うが、Semantic Extension Referenceの `SemanticValidationPlan` やShared Acceptanceを入力として評価しない。両者を連結済みとみなしてはならない。Referenceの実process PluginもDeploymentExecutorの応答を返すだけで、業務意味の受理や実際の外部状態の反映を証明しない。Runtimeのin-process実装へ渡すPermission一覧はHost側のProvenanceと別コピーにし、実装が一覧を書き換えてもHostの記録とlease解放が保たれることを試験する。これはprocess分離、悪意あるin-processコードのSandbox、全Provenance要素の完全性、正式な意味評価との接続を証明しない。

独立した実processの受注見積もりProviderはReferenceの業務契約で実計算するが、候補の意味Model・Shapeと正式に適合した証明や、その独立業務操作の受理は別である（`reference/candidates/order-quote/README.md`）。汎用Transportへ業務JSONの形やPrincipalを一律に必須化せず、外側の契約・Providerメタデータの曖昧な重複や別名だけを拒否する。実Provider応答の改変負例はprocess起動後に検出され、HTTP要求の同種の異常はProvider呼出し前に拒否される。Transport成功、結果Shape、業務計算の正しさ、実行権限と意味受理は別々に確認する。

既存の受理済み受注意図の保護された要求内で、実見積もりprocessを副作用のないcoeffectとして呼ぶ試験では、数量・単価などの入力値を変更しても同じHost選択済み契約・Providerを使い、小計が1550から1800へ変化した。変更後も現在のHost権限確認を実施し、閲覧者は両入力ともprocess起動前に拒否された（`tools/reference-capability-transport/authorized_quote_preflight_test.go`）。これは入力値の変更に限る観測であり、意味定義・Shape・認可規則の変更後に再検証を省略できる証拠ではない。独立見積もり操作の意味受理や本番権限も示さない。

**環境単位のLockに関する追加候補**：試作前のCLIの `plugin-lock` は全Environmentを対象とする一方、既存 `plugin.BuildLock` は指定されたEnvironmentの一覧を受け取る。`internal/plugin/lock_scope_value_test.go` の合成例では、productionのPlugin欠落により全環境のLock作成が失敗する一方、developmentだけのLockは作成・解決できた。そのLockでproductionを解決すること、固定したdevelopmentのPlugin定義を改変して解決することは拒否された。利用する環境だけを明示してLockを作るCLIの可能性はあるが、CLIは先にWorkspace全体を検証するため、無関係なエラーによる停止はこのAPI試験だけでは解消しない。

**CLI試作**：Draftの `plugin-lock -environment development` は明示された環境だけのLockを返し、envelopeの `mode: environment` とLock内のEnvironment一覧で範囲を示す。省略時の全環境Lockは変更しない。現行Referenceを使う実CLI試験では、development部分は全環境Lockと一致し、このLockをproductionの解決・Runtime Planに使うと拒否された。明示的な空ID・不正なIDと存在しない環境も拒否する。別環境のPlugin欠落や業務型エラーを含むWorkspaceでは、明示した環境が正常でも全体検証で停止するため、未検証Workspaceから正式Lockを生成する迂回路ではない。実運用の利便性・外部利用者の部分Lockへの対応は未確認で、採用判断は行わない。

**Runtime接続の限定検証**：`cmd/tei/plugin_lock_scope_runtime_test.go` は業務Semantic Dataを持たないPlugin self-build Reference（標準型 `std` は存在）からnative static-response実装をmaterializeし、その実現Digestを結んだ一時Worldに対し、実CLIで環境単位Lockを作成する。既存Runtime PlanとBuiltin adapterを通じて実装の応答を得られ、別環境、定義Fingerprintの不一致、実現byteのDigest不一致は拒否された。Referenceのin-memory検証器はDigestだけを照合し、署名・出所・Host権限を検証しない。またこれはstatic responseを返す実装であり、業務意味の評価やAcceptanceの成立を示さない。別の実CLI試験では、Referenceのdevelopmentに対し全環境Lockと部分Lockから同一のRuntime Planが生成された。

`plugin-lock` と `plugin-resolve` は現在 `cmd/tei/cli_result.go` でWorkspace全体を検証してから実行する。最小の改善候補は、既存コマンドの成功条件を変えることではなく、必要なPlugin・環境の選択結果と、Workspace全体が未検証である旨・元の診断を別に返す**任意のread-only診断**を比較すること。その結果からLockの正式利用やRuntime起動を許可しない。依存する意味定義を必要とするPlugin、未解決の環境、改変されたPlugin定義を誤って有効としないことを実証できなければ、候補を採用しない。専用CLIや新たな正本が必要かも未決である。

**診断の試作**：Draftでは `plugin-resolve -diagnostic-candidates` を明示した場合だけ、全体の診断と、既存Resolverが返した候補を別に表示する。常に `ok: false`、終了1、`workspaceValidated: false` とし、候補は意味受理・実行許可・Runtime Planではない。環境不明、Plugin欠落、Lock不一致では候補を返さない。Workspaceの読み込み自体に失敗した場合は診断を続行できない。未検証の意味がPluginに必要かをこの試作は判定しないため、候補があっても使用可能性は不明である。通常の `plugin-resolve` と `plugin-runtime-plan` の検証・終了条件は変更しない。実Providerや自動化利用者への影響は未確認であり、採用判断は行わない。

追加の負例では、application-start Executorを `sales` の意味を使うApplicationへ結び付け、Application側の型参照を壊した。実CLIは通常のPlugin解決とRuntime Planを拒否し、診断モードだけが候補と元の意味エラーを未検証として返す。これは対象ApplicationとExecutorの接続を使った合成例であり、Plugin実装がその型を実際に消費した証明ではない。現行 `plugin.Definition` に業務意味依存を直接宣言する欄はなく、正式Role／Language Pluginの意味依存を検証できたとは扱わない。

**読み取り専用の比較経路**：Reference toolの `compare-application-membership -before DIR -after DIR` は、二つのWorkspaceから厳密なApplication差分と利用集合の候補差分を別々に表示する。片方でもValidation errorがあれば既定では元診断を返し候補を表示しない。明示的な `-include-unvalidated-candidates` では、差分が計算可能な場合に限って元診断と `workspaceValidationPassed: false` を添え、終了1のまま候補を表示する。SourceやDecisionは書かず、既存のPreview・Review・Acceptanceの計算にも接続しない。候補はTargetの表示順や依存網羅性の検証ではなく、異なる時点の読込み間の原子性も保証しない。実利用者がこの比較で負担を減らせるかは未測定。

**編集から選択までの対照**：`tools/reference-shared-acceptance/definition_plugin_workflow_test.go` は一時Workspaceと実CLIで、Applicationの名前空間利用順と業務Data属性を2ファイルで変更した。2件の型参照と1件のPlugin Lockは変更せず、通常の検証とPlugin解決が成功した。Application利用集合の候補は空でも業務属性は追加されているため、比較コマンドに既存のSemantic subject差分を別欄で表示し、候補側にもSemanticなどが対象外と明記する。未定義型に変えた負例では通常Plugin解決と比較が停止し、明示的なPlugin診断だけが元診断・未検証候補を非成功で返す。この件数は合成例の変更ファイル・参照・Lockの観測であり、実利用者の入力時間や全影響を測ったものではない。

**定義Sourceの置き場所と注記**：現行Referenceの複製で `semantic/sales/order.tei` と `plugins/tei-health-reference/1.0.0/plugin.tei` をサブディレクトリへ移し、移動後のPlugin定義へ注記を1行追記した。`validate`、移動前に作ったLockによる `plugin-resolve`、再生成した `plugin-lock` はすべて通り、Lockの出力は移動前とバイト単位で同一だった。逆に移動した意味定義を削除すると検証は停止する。定義の同一性が置き場所や注記ではなく内容と宣言に基づくことの限定例であり、他のToolやEditorが同じ手順で編集できること、別Workspaceでの再現、実利用者の編集時間は示さない。

**読み込み時のBOM**：WindowsのEditorが付け得るUTF-8 BOMは、読み込み境界で削ると `sourceSnapshot`、Source Index、バイト位置で編集する経路の基準が3バイトずれる。そこで字句解析だけが先頭の1個を無視し、`Offset` は元のbyte列に保つ。Referenceの複製で `semantic/sales/order.tei` の先頭へBOMを付けると `validate`、`plugin-resolve`、`plugin-lock` が通り、Lockは付与前と同一、`compare-application-membership` のWorld集合差分は空だった。末尾へNULを加えると `validate` は停止し、途中のBOMと2個目のBOMは字句解析で拒否する。同じ複製でapplication・database・contract・environment・plugin・release・header-detail Binding・Executor Binding・SQL realization・bundle・migration・persistenceの代表ファイルへ個別にBOMを付けても、`validate`、Lock不変、World差分なしが成立し、BOMを除いた末尾NULは各経路で拒否された。現行Referenceのこれら12経路では挙動の差は見られなかった。実Providerの実行、他ToolがBOMを保存する場合の編集往復、実利用者の負担は示さない。

**既存Reference Workspaceの複製での対照**：`tools/reference-shared-acceptance/reference_workspace_value_workflow_test.go` は現在の `reference/workspace` を二つの一時Workspaceへ複製し、片方の `sales.order` に文字列属性を追加する。1ファイルの編集で検証が通り、development環境の選択済みPlugin定義8件は元のLockで解決でき、Lockの再生成結果も同一だった。実CLIのSemantic差分は追加属性1件、Application利用差分は0件で、既存の影響分析は所有Dataへの1辺とDriver／Target未評価の2項目を返した。新属性の型を未定義に変えると `plugin-lock`、通常の `plugin-resolve`、`plugin-runtime-plan` と比較コマンドは停止し、明示的なPlugin診断は元のエラーと未検証候補を返した。Workspace全体のValidationは正式なPlan・実行の境界として維持し、編集途中の情報取得を止める範囲のみ候補として検討する。対象Pluginは選択しただけで実行しておらず、入力回数・時間、実利用者、影響の網羅性は測定していない。固定されたGate B/C版ではなく複製時点のReferenceであり、元Sourceは変更しない。

同じ複製の別ケースでは、`order_entry` の意味名前空間の利用宣言を1ファイル内で並べ替えた。既存LockでPluginを解決でき、厳密なApplication差分は1件、利用集合の候補は0件だった。Validationの宣言順も変えると候補は1件となり、順序の意味を消さない。存在しない名前空間に変えた負例では通常比較とPlugin解決が停止し、明示的な診断では元診断と未検証状態を残して候補を確認できた。これはSourceの表示順やTargetが不変という証拠ではなく、既存の受理条件も変更しない。

型定義の変更では、Referenceの `sales.order_id` の基底を一時コピー上で変更した。実CLIの検証と旧Plugin Lockによる選択は成功し、参照するAttributeとDatabase Contract QueryのSourceは書き換えなかった。一方、従来のReference影響分析には型変更の分岐がなく、両者の明示参照を候補に出していなかった。型別名・Attribute・Contract Query入力のexact ID参照を候補として追加し、変更・削除時は変更前の参照も区別する。推移依存や保存データ・移行・互換性は未確認と明示し、読み取り専用比較にも候補と未解決事項を表示する。未定義の基底型は通常の検証・Plugin解決で拒否される。型変更が業務上安全であること、実Providerの動作、影響の網羅性は示していない。

Database Contractの別ケースでは、現行Referenceの隔離コピーで `sales_core.order_line` の版を1.0から1.1に変更した。1ファイルの変更後、検証と旧Plugin Lockによるdevelopment選択は成功したが、Semantic・Applicationの候補差分は空となる。これを全体差分なしと誤認しないよう、既存のWorld collection単位比較を読み取り専用比較に追加し `databases` の変更を表示する。これは詳細なContract差分や互換性を判定しない。対照としてQuery入力型を変えると、既存header-detail Bindingの型不一致で検証が停止し、未定義型も拒否された。無効Workspaceでの既定停止と、明示的な未検証比較の終了1・診断保持は維持する。版変更の業務互換性、影響の網羅性や実Providerの動作は未確認。

このContractの限定的な利用先照合では、読み込んだ一時WorkspaceにApplicationのContract利用要求、header-detail Query Binding、development Executor Binding、PostgreSQL SQL realizationが各1件あった。Query単位の3件はDatabase／Contract／Queryの完全一致で照合し、別Contractや別Queryを除外した。版変更時、それら4つのSourceは変更されていない。Query入力名を変更した負例では、既存検証がheader-detail入力とSQL realizationの不一致をともに診断した。これは試験内の明示参照の個別照合であり、比較CLIがContract利用先を網羅的に報告する機能、実SQLやProviderの実行結果、全Release・動的依存の確認ではない。

この観測を基に読み取り専用比較へ `contractCandidates` を追加した。変更したContractの前後の明示参照を8種類だけ、Database・Contractの完全一致で列挙し、Query単位の参照にはQuery名を残す。対象はApplicationのContract要求、Application ReleaseのContract要求、reference PatternのContract Query呼び出し、Database Releaseの提供Contract、Solution ReleaseのContract gate、header-detail Binding、Executor Binding、SQL realizationである。Contract単位の候補であり同一Contract内の別Queryも保守的に表示する。未走査参照・動的依存・互換性・実行結果は未検証と出力へ明示し、既存の受理判定には接続しない。隔離ReferenceでApplication要求・header-detail Binding・Executor Binding・SQL realizationを前後に各1件確認し、`master_data.customer` の版変更では `order_entry.customer_reference` と `repair_intake.customer_reference` の `find_by_id` 呼び出しが前後各1件の候補になることを確認した。別Contract・別Databaseの誤一致と、削除前参照の喪失を合成負例で検査する。ApplicationのContract要求とApplication ReleaseのContract要求は、現在定義と不変のReleaseスナップショットという別の宣言であり、同じ要求を両方が持っていても統合しない。現在定義だけが要求する場合と履歴Releaseだけが要求する場合を独立した正負試験で固定した。Application Release側のContract要求はApplication・Release major系統・datasource aliasごとに、そのContractを最後に宣言したReleaseだけを表示する。削除されたContractでも各系統で最後に宣言したReleaseは残すため削除検出の情報を失わず、同一系統のpatch/minor履歴だけを畳む。数値のmajor.minor.patchでないversionは順序付けできないため表示しない。同じmajorの並行系統は最新宣言1件に集約される境界を、比較CLIの `authorityLimits` に `contract-candidates-application-release-latest-declaration-per-major-line-no-explicit-line-identity` として示す。Database Releaseの提供ContractもDatabase・Release major系統ごとに、そのContractを最後に提供したReleaseだけへ畳む。同じ系統のpatch/minor履歴だけを畳み、最後の提供が履歴側にしかなくても残すため、削除検出の系統被覆は失わない。同じmajorの並行系統が最新提供1件に集約される境界は、`authorityLimits` の `contract-candidates-database-release-latest-provider-per-major-line-no-explicit-line-identity` として示す。同一snapshotで種別・識別子・Queryが完全に同じ候補行は、区別できる情報がないため1件に畳む。

**Release記録との対応**：`sales_core.order` を v4.2 から v5.0 へ上げ、同時に現行Applicationの要求を `^5` へ変えると `validate` は成功する。しかし `releases/applications/order-entry/3.8.0.tei` の `^4` と `sales_core` 4.1.0／4.2.0 の提供4.xは診断されず、Release記録と現在定義のmajor不一致が無言で残る。Application要求を変えずにv5.0へ上げた場合は `DATABASE_CONTRACT_VERSION_UNSATISFIED` で停止する。逆方向として現行定義をv3.0、現行要求を `^3` へ下げると、提供4.xが現行定義より新しいため `DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION` が出る。つまりDatabase Releaseは「記録が現行より先行」だけを警告し、「現行が記録より先行」は警告しない。Application Releaseの要求はContractの存在と所有者だけを検査し、互換majorは現在定義と照合しない。v4.3へのminor更新は成功し、Release記録は比較の `contractCandidates` に前後とも現れる。歴史的Releaseを一律に不正としないため `validate` の受理条件は変えず、この非対称を読み取り専用候補で可視化するに留める。実Deploymentの順序・データ移行・Provider実行は未検証である。

**Solution ReleaseのContract gate**：`order-rollout` の `master_ready` は `master_data.customer ^2` を宣言する。現行の `master_data.customer` をv3.0へ上げ、現行Application（`order_entry` と `repair_intake`）の要求を `^3` へ揃えると、`validate` は成功し `solution-release-graph` も `executable=true` のまま `master_ready` を `order_app` の前提として解決する。gateの `^2` は履歴Application Release 3.8.0の `^2` とだけ照合され、現在定義のv3.0とは比較されない。これはgateが実行時に観測する提供Contractのmajor（`internal/deployment/durable`）とも別の検査である。一方、gateのContractを存在しない名前に変えると、そのgateを必要とするApplication Releaseがあるため `validate` が `SOLUTION_RELEASE_NOT_EXECUTABLE` で停止する。ただし、どのApplication Releaseからも必要とされないgateを追加し、存在しないContractとmajorを宣言しても `validate` は成功し、診断は出ない。`compatibleMajor` が0の場合は `INCOMPLETE_DATABASE_GATE_STEP` で拒否される。gate宣言自体は `contractCandidates` に前後とも現れる。候補 `unreferencedUnsatisfiableGateCandidates` の参照・充足判定は、このrelease compilationの前提解決と同じ互換majorの完全一致を使い、候補側だけで条件を緩めない。gateの現在定義との照合を必須にするかは受理条件を変えるため、ここでは観測だけを残す。

**未使用gateの読み取り専用候補**：未使用かつ不成立のgateを、比較CLIの `unreferencedUnsatisfiableGateCandidates` として前後snapshot別に列挙する。条件は、どのApplication Releaseの要求（Database・Contract・互換major）にも一致せず、現在のContract定義にもDatabase Releaseの提供にも互換majorを満たすものがない、の両方である。どちらか一方だけのgateは出さない。参照されるgateはrelease compilationで検査され、満たせるgateには提供元があるため、この積集合だけを観測する。現行Referenceでは候補は空で、`master_data.nonexistent ^7` の未使用gateを追加すると `validate` 成功のまま1件出て、識別子・Contract・majorが保持される。`sales_core.order_line ^1` の未使用gateは現在定義で満たせるため出ず、削除したgateは変更前snapshotにだけ残る。合成試験では、参照されるgate、現在定義で満たせるgate、Database Releaseで満たせるgateを除外し、Database所有者の異なるgateを含める。実行時のDeployment状態、動的利用者、Provider動作は未検証であり、空の候補は問題なしの証明ではない。

**Application ReleaseのContract要求と現在定義**：履歴Application ReleaseのContract要求は `validate` では存在と所有者だけを検査され、互換majorは現在定義と照合されない。そこで `applicationReleaseContractCandidates` として、互換majorが現在定義のmajorと異なる要求だけを前後snapshot別に列挙する。現在定義に存在しない要求は既存の `APPLICATION_RELEASE_CONTRACT_NOT_CURRENT` が扱うため、候補では繰り返さない。ApplicationごとのRelease major系統・datasource alias・Contractごとに、その要求を最後に宣言したReleaseだけを観測し、系統内のpatch/minor Releaseが同じ要求を繰り返しても同じ状態を重複報告しない。より新しいReleaseが同じ要求を宣言すれば置き換え、新しいReleaseで要求が落ちても最後に宣言したReleaseからは報告する。数値のmajor.minor.patchでないversionは順序付けできないため観測しない。同じApplication・同じmajorで明示的な系統識別子がない並行系統では、同じContractの要求は最新宣言1件に集約される（過小報告）。この制約は比較CLIの `authorityLimits` に `application-release-candidates-latest-declaration-per-major-line-no-explicit-line-identity` として出る。現行Reference（`order_entry@3.7.0` の `sales_core.order_legacy ^1`）では候補は空で、既存警告だけが出る。`sales_core.order` をv5.0へ上げ、現行要求を `^5` に揃えると `validate` は成功し、`order_entry@3.8.0` の `^4` が `currentContractVersion: 5.0` として変更後だけに出る。v4.3へのminor更新では増えない。minor差を無視するのは互換majorの判定に合わせた限定であり、履歴Releaseの互換性や実行時の挙動を保証しない。受理条件は変更しない。

**Database Releaseの提供Contractと現在定義**：Database Releaseの提供Contractは `validate` で現在定義との存在・所有者・先行方向だけを検査され、記録が現在定義より古い方向は診断されない。そこで `databaseReleaseContractCandidates` として、各Databaseの各major系統で最新のReleaseが提供するContractのmajorが現在定義のmajorより低いものを前後snapshot別に列挙する。同一系統の履歴Releaseをすべて繰り返すと同じ古い状態を重複して報告するため、系統ごとの最新Releaseだけを観測する。複数major系統の並行保守を隠さない。現在定義に存在しない提供は既存の `DATABASE_RELEASE_CONTRACT_NOT_CURRENT` が扱い、先行方向は `DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION` が扱うため、どちらも候補にしない。minorだけが古い提供も含めない。現行Reference（`sales_core.order` v4.2）では候補は空で、v5.0へ上げると最新の4.2.0だけが `providedContractVersion: 4.2`、`currentContractVersion: 5.0` 付きで変更後に出る。v3.0へ下げると4.1.0と4.2.0は既存の先行警告だけになり候補は空、v4.3へのminor更新でも空のままである。履歴Releaseの正否、データ移行、実行時のDeployment状態、Providerの動作は未確認であり、空の候補は問題なしの証明ではない。受理条件は変更しない。`contractCandidates` の `database-release-contract-provided` は変更Contractを提供するReleaseを影響範囲として列挙する。両者は同じReleaseを含み得るが、前者は変更影響、後者は系統の古さという別の問いであり、重複として統合しない。

**候補ビューの負担比較**：独立したWorkspace複製に代表的な変更を加え、4つの候補ビュー（`contractCandidates`、`unreferencedUnsatisfiableGateCandidates`、`applicationReleaseContractCandidates`、`databaseReleaseContractCandidates`）の件数が意図した境界だけを報告するかを測る。現行Referenceのbaselineはcontract 0・gate 0・appRelease 0・dbRelease 0。無関係な `sales_core.order_line` v1.0→v1.1 と `sales_core.order` v4.2→v4.3 のminor更新ではcontractだけが1件になり、他の3ビューはbaselineのまま。`sales_core.order` v5.0と現行要求 `^5` の同時変更ではdbReleaseが1件（最新の4.2.0）、appReleaseが1件（`order_entry@3.8.0` の `^4`）だけ増え、gateは0件のまま。未使用の `master_data.nonexistent ^7` gateを追加するとgateだけが1件増える。要求を `^4` のまま残したv5.0は `validate` が失敗し、既定の比較は終了1・候補0件、`-include-unvalidated-candidates` でも終了1とエラー診断を維持したままdbRelease 1件とappRelease 1件を出す。つまり、この範囲では候補ビュー間の相互混入は観測されず、未検証状態の候補は明示操作なしに出ない。ただし対象は合成Referenceの5シナリオだけで、実変更履歴・実利用者の負担、Providerの意味適合性、他環境での再現、実行時Deploymentとデータ移行は未検証である。件数が0でも問題なしの証明にはならない。

**同一major内の複数系統**：Databaseが異なれば同じmajorでも別系統として扱い、合成Workspaceの `line_a` 4.2.0と `line_b` 4.8.0をそれぞれ1件だけ報告し、履歴の4.1.0／4.7.0や別DatabaseのContractを取り違えないことを実CLIで確認した。一方、同じDatabase・同じmajorで明示的な系統識別子がない場合、4.3.0と4.9.0はmajor単位で1件に集約され、4.9.0だけが報告される。これは過剰報告ではなく過小報告であり、今回は Database Release に系統識別子を追加しない。別 Database が系統の境界であり、同一 Database・同一 major の複数系統は未対応境界として受理する。実利用で分離できないケースが確認された場合に再検討する。この制約は比較CLIの `authorityLimits` に `database-release-candidates-latest-per-major-line-no-explicit-line-identity` として出る。合成Workspaceと実CLIの範囲であり、実Provider・本番の系統運用は未検証である。

**Build位置の限定対照**：`internal/plugin/build_location_actual_test.go` は実Goコンパイラで同一内容の合成モジュールを二つの異なるディレクトリからBuildする。Go 1.27.3／Linuxでは `-trimpath` 有効時、同一位置の再Buildと移動後の出力バイトが一致し、Source内容を変えた出力と実行結果は変化した。`-trimpath` なしでは同じ内容でも位置によってバイトが異なった。同じSource DigestでLocatorのみ異なるBuildPlan v1のFingerprintは両条件と独立に異なる。Go単体の比較であり、実TEI Builder AdapterがPlanを実行した証拠、他Target・Compilerでの再現性、移動後のBuild省略の安全性を示さない。

既存の `tools/reference-build-execution-binding` はLocatorを取得ヒントとして除く**別のReference限定**の実行Plan投影を持ち、BuildPlan v1や既存LockのFingerprintを変更しない。条件付きの再利用を検討するなら、Builderが位置の非依存性を保証できる範囲と実際の入力・出力を確認し、既存の候補を対照する。今回の実験だけでv1固定範囲やLock照合を変更しない。

**実AdapterとReference Providerの追加対照**：`tools/internal/gobuilder/provider_location_test.go` は実 `BuildPackage` と `tools/reference-process-plugin` のSource・Go依存を使用し、一時Workspaceの別々のBuild stateで4回Buildする。Go 1.27.3／Linuxでは、同じ相対パスのままWorkspace全体を移すと出力は一致した。一方、Plugin Sourceの内容を保って `main.go` から `relocated.go` に改名すると出力が変わり、Source内容の変更でも出力が変わった。これらは `-trimpath` を常に指定するAdapterでの観測であり、Workspace rootの移動とSourceファイル名の変更を同じ「位置変更」と扱えない。実GoコンパイラとReference ProviderのBuildは実行したが、完全なGit checkout・self-build runner・正式なProvenance／Authority検証ではなく、他のGo版・OS・Providerへの一般化もできない。既存のBuildPlan v1と出力Digest照合は維持する。

[#2620 RE-01〜10](https://github.com/hat47x/tei-platform/issues/2620)と[TAI #271 QH-01〜18](https://github.com/hat47x/tai-pleroma/issues/271)の横断研究から、**同じRole ID/versionでも実装内容が変わり得る**という受入条件を具体化した。前段ReferenceはRoot/Eager依存先について**版文字列だけを比較**していたため、同じ版の実装差を見逃すという、あくまで**このFixture自身の不足**があった。正式TEI製品に当該欠陥があると断定するものではない。

そこで、各Role DescriptorとPinを、`version + contract fingerprint + actual implementation fingerprint`という**比較用の3軸**に拡張。Rootと推移するEager依存先の双方でexact照合を要求する。Fixtureのfingerprint文字列は**テストが与えたラベル**であり、現物バイトをSHA-256計算した真正なLock証拠ではない。実TEIへの接続時には、既存Extension Lock／Language Selectionが検証する**実Provider物・Pin対象・Implementation fingerprint**を利用し、自己申告Stringを信用しない。

| Case | 比較する誤成功・性質 |
| --- | --- |
| **RB-20** | Role ID/version不変、Root実装fingerprintだけ変更→旧Pinを拒否 |
| **RB-21** | Rootのexact pinは不変、Eager依存先の同版実装だけ変更→旧依存Pinを拒否 |
| **RB-22** | 変更後の依存先実装に**明示的にPinを更新**→shape/循環条件が揃うならRole候補として再資格化可能。旧Meaning受理の自動継承はしない |
| **RB-23** | 無関係なRole Provider追加→既存exact-selected Roleの合成出力は変化せず（公開候補発見自体の比較は別#2620 RE） |
| **RB-24** | version文字列だけのPin→実装内容未照合として不成立 |

**限定実行記録**：GitHub PR #2638ブランチで変更した`check_bootstrap.js`をV8で**24/24 PASS**。実際のTEI Provider/Plugin byte digestを計算したものではなく、正式Conformanceも未取得。これは新Resolverの提案ではなく、既存Versioned Selectionの**抜けを検出する受入視点**である。

### 4.3 Roleの変更影響と動的Read依存を一つの「全局依存Graph」にしない

[#2582 DC-01〜12](https://github.com/hat47x/tei-platform/issues/2582)では、同じAmount・同じ出力Fieldでも`Order.customerRef`の再束縛で必要なAccount.limitが変わる例、既存Row IDが0件の`NOT EXISTS`や`Hold`の追加で真理値が撤回される例を扱っている。これは**Runtime Read/Claim資格**であり、本ReferenceのEager Role Provider bootstrapとは異なる。共通概念として言えるのは、「**依存閉包を列挙できなければ不変と断言しない**」という条件のみ。

| 変化 | 確認する依存 | Owner／既存成果 |
| --- | --- | --- |
| Role/Operator自身の同版実装差 | Root＋Eager依存Providerの実byte/Lock、解釈済Meaning、関連Artifact | #2620／#988／本RB-20〜24 |
| FieldのValue不変だがRelation Bindingだけ変更 | Predicate、Relation occurrence、選び直した実入力、Read cohort／completeness | #2582 DC-02、#2506 |
| 0件の否定結果へ新しいMatching rowが加入 | 結果中のRow IDではなく**未出現行を含むPredicate/Scope**と変化観測 | #2582 DC-04〜07、#2387 |
| Provider route／Query semantics変更 | Read witness、NULL/Bag/Sort/Limit、実RDB/API、Consumer用途 | #2622／#2611／#2627／#2628 |
| Consumer開示・Host Grantだけ変更 | 現在の公開・実行資格。歴史的なSource/Role/Receiptは変更しない | #2516／TAI #271／#1174 |

ここで**Identity/Meaning/Build/Runtime Claim/Consumer資格を単一のフラグやglobal revisionで表さない**。同じ見かけの数値でも、歴史的「その期間に変化なし」のClaimと現在値のClaimは異なる（0→1→0のABA）。この表は新しいMeta Schemaや実行時Dependency Engineではなく、各Ownerの試験結果を照合するためのP1受渡し票である。

### 4.4 「実Goに到達した部分」を踏まえた次の一体試行

現行mainは初期基準`9c9ef4d...`から進み、[#2587](https://github.com/hat47x/tei-platform/pull/2587)は**mainへ統合済み**。従って新たなP1統合試験は旧mainを固定したまま現在の既存CLI能力が無いと仮定しない。開始時にmain exact commit・選択された実Executable/argv/Workspace/Source bytes・Extension Lock・Skill bytesを固定し直す。

- **実Language/Operator Reference**：[Draft #2639](https://github.com/hat47x/tei-platform/pull/2639)は**既存Goの**`ParseSelected`/`ParsePipelineSyntax`/`LowerPinnedPipeline`を使い、別表層から同じ限定candidateを得る7件のテストを追加している。ただしSource/Role accepted meaningやFunction Contextの実結合ではなく、**そのPRのexact headのCI結果を別に確認**する。本RB Fixtureの`pinFor`はそのGo Selectionを代替しない。
- **実CLI→Consumer Witness**：[Draft #2626](https://github.com/hat47x/tei-platform/pull/2626)には実Go CLIで公開Reference Workspaceを読んでSource relocation時の意味指紋とSource Index/Trace/引用先が異なることを確認した既報がある。公開Referenceの実Readの根拠と、別SourceのRole解釈・実Studio/SHO Clientの実Loaded bytesは**異なる試行**。これをEI-01〜08の一体PASSに合算しない。
- **最初のB1の判別対象**：実Go Selectionが`ImplementationFingerprint`を持つ場合、(A)固定Lockと実内容byte一致、(B)Rootだけ一致／Eager依存Bは同版だが実体不一致、(C)無関係Providerを追加、(D)未実装RoleのScopeに入ることを区別。実GoがRoleのEager依存という概念を現時点で表現しないなら**unsupported/gap**として報告し、新Role Resolverを先行実装しない。
- **P1継続条件**：Schema/Profileの選択／独立Expected／必要なRead/Role/Source参照が固定できた1モデルだけを、headless →二Consumer→Providerへ段階展開。OwnerのP0-a実DB・P0-b Owner-write・S0/J5/S3と既存Gateは従来どおり優先。

## 5. 最短の既存TEIへの接続

**B0**：本Referenceで、必要情報が基本型/Ref/Pin/Provider descriptor/Role候補のどれに相当するかを資料と現行Sourceから明示する。Eager/Deferred依存のモデル化を決定前のpressureとして残す。

**B1**：#988のLanguage/Semantic Extension Lockと既存read-only inspect/validationへ、実在するPin／Provider ID／Role/Operator依存を渡し、RB-02/03/04/05の判別ができるかを確認する。新Provider ResolverやCore演算の再実装はしない。

**B2**：#2625のSyntax正規化fixtureと#2624のFunction別Mappingを**同じ受理Source・同じ意味ID／同じPlugin版**へ繋ぐには別の契約が必要。#2625は`sales.confirmed_total`、#2624は`Order→Line→Note`と現時点で異なる合成Subjectを扱う。これらの局所PASSを単一縦断PASSと呼ばない。[#2586 HJ](https://github.com/hat47x/tei-platform/issues/2586)／[#2619 EI](https://github.com/hat47x/tei-platform/issues/2619)の同一Fixtureに各Ownerが実対応のある受渡しを提供して初めて意味・Provider/Consumer接合を判定する。

**B3**：#2617のScript-first標準UI、#2605のProduct Skill、#2621のMarkdown/SHO、#2622の多Provider、#2516のConsumer Contextへ同じ確定できたReferenceを供給し、`parsed/role candidate/meaning qualified/tool invoked/actual effect/disclosure`の主張を分ける。

### B4 — 変更後の選択的再資格化（歴史・現在・将来を分ける）

複数AI・複数Consumerが同じ役割記述から多種類の結果を導出する場合、以前に観測した「当時の成功」という**歴史的事実**と、いまその結果を再利用してよいかという**現在の資格**を区別する。上記RB-09（Source同byte／Role Plugin版のみ変更）はその最小反例である。

単一のRepo全体Revision Counterや新しい共通Evidence DBを作る代わりに、**Consumer Claimの用途に必要な、既存Source/Plugin/Role/Operator/Function Mapping/Provider/Artifact/Hostの依存参照**から再確認対象を絞る案を比較する。必要な依存が欠ける場合は「影響しない」と断定せず、当該Claimを未確定にする。**見えない依存（where条件／Sort／Rule／Scope／開示条件等）を出力Fieldだけから除外しない**。

| 変更の例 | 既存Ownerで選択的に見直す事項 | 再利用上の注意 |
| --- | --- | --- |
| Markdown改行・Graphviz layout | Source occurrence／Authoring fidelity。意味解釈が同一と独立確認できる場合に限りMeaningを保持 | Source byteの変化を全Artifact意味変更と自動同一視しない |
| Syntax Pluginのv1→v2 | 当該Source occurrenceの正規化結果、Role Binding、下流Meaning | Source同byteでも旧accepted meaningが新版へ自動移行するとは推測しない |
| Role/Operator依存Bだけ改版 | Eager依存を持つ利用箇所のRole資格、利用する演算の型・Law/Effect、関連Fixture・Generated artifact | 影響していないClaimは明示根拠があれば保持可能、依存不明なら保留 |
| Function Mapping／Rule／Provider path変更 | 宣言したRead footprintと実Provider Read Trace、Operation入力/Storage、Consumerの必要Artifact集合 | Operation ID/Basis不変でもInputs/Storageが変わる反例 #2632、生成一式の相対充足 #2630 |
| Host Grant・Disclosure資格変更 | 現在のPrincipal/目的/開示先/時点で実利用前に再照合 | 歴史的なRole/Resultは残せても現在の実行や送出は別資格 |

参照：#2505版間意味対応、#2600増分ビルド、#2630 Consumer別ArtifactSet、#2516用途相対Claim、#1174実Grant。**再資格化は再生成と同義ではない**。同じArtifactを再利用できても、新しいConsumerが要求する範囲・現在の開示資格は再確認が必要になり得る。逆にMaterialized output bytesが同じでも、Rule/Provider/Targetの依存が変われば、元の十分性が維持された保証にはならない。

**選択的再確認ケース（提案／未実行）**：RI-01 layout-only変更、RI-02 Eager Role-B改版で一方のFunctionだけ影響、RI-03 意味に影響するWHERE/Sort依存が出力列に現れない、RI-04 過去Source/Artifact同一のまま現Host Disclosureが撤回。まず既存#2600/#2516/#2630の試験と重複のない入力差を比較し、必要なら#2586／#2619の同一Runへ追加する。適用根拠が取れないものは未確定として報告する。

**効果測定**：新Core特別型数／Roleの重複宣言数／未解決Roleの診断品質／Plugin差替えでの予期しないMeaning変化／標準UI専用コード増分／外部Skillの架空Tool呼出／同一入力での不同結果。単なる`PASS`件数、Role定義数、ソース文字数を製品品質とみなさない。

**責務と優先順位**：#2620がRole自己記述・Bootstrap判断、#988がLanguage/Operator/Pinの実装、#2544がFacetの法則、#2621がSource/引用/テスト、#2586が同一実Fixtureの縦断、#1119が既存P0/G0〜G4の優先を管理。P0-a実PG Receipt/Lookup、P0-b二Function Owner-write、S0 H2/J5とS3意味受理は維持。二Harness配布・OS/Provider/GUI実証はOwner別。

**証拠の状態**：この計画および対応する独立合成Referenceは、TEI実Parser/Role Provider/Studio/Host/DB/Owner受理の成立を意味しない。正式Grammar/初期bootstrap固定仕様と`std.*` Role名は、本実証で既存契約の不足を観測した後にOwner判断する。


<!-- role-self-interpretation-closure-20261105 -->
## 6. 2026-11-05 横断統合：自己解釈の必要閉包・観測同値・独立Oracle（設計／NOT RUN）

この節は #2620 の SDX-01〜08（2026-11-04追補）を**置換しない**。関連レーン #2823（未読込Workspace Source）、#2505（版間意味対応）、#2514（独立Oracle）、#2586（Query/Contextの同一縦断）、#2600（選択的再利用）、#2831（TEI全体セルフビルド）から、自己記述**固有**の不足に絞って追加する。新しいCore Meta Schema、Interpreter/Resolver、Policy Engine、中央Evidence DB、Runtime Host、または横断Gateの新設はしない。

### 6.1 自己記述を「宣言」「解釈」「適合」「利用」に分ける

1. **Descriptor as Data**：既存の基本値・Record/Sequence/Refで、Role/Operatorの要求Shape、識別子、適用条件、診断、依存（eager / deferred）、実装へのBindingを記述する。Role名や `effect: pure` は説明値であり自己権限ではない。
2. **Interpretable**：既存Language Plugin/Extension Lockで *実Content/Contract/Implementation/推移eager依存* が選択・検証され、限定的なtyped Semantic Candidateへ解釈できる。構文糖衣の同形出力はAccepted Meaningと異なる。
3. **Meaning Qualified / Accepted**：型、Relation/Attribute、Input/Output、演算の定義域、NULL/missing/unfetched、単位・精度・時点・grain、Queryの選択/否定依存を用途に照らして判定する。未定義の条件を勝手に埋めない。Ownerによる意味受理は独立の状態。
4. **Usable / Realized**：必要なTarget能力、Build Artifact/Consumer必要member、実読み込み、現在のHost Grant/Disclosure/Effectを別検査する。Role解釈は実行資格を与えない。

**有限Bootstrap**はこれらを初めて読む既存実装上の根であり、全Parser/Interpreterの自己生成までは初期合格条件にしない。自己記述の「自己適用」が通っても、seedの真正性・外部Host・実DB・全Targetの保証にはならない。

### 6.2 Claim相対の必要閉包（read-only受渡し票）

各Consumer/Function/Build/Role Claimごとに、既存Ownerの記録から**必要条件と観測された証拠を突合する読取り専用ビュー**を組み立てる。これは新しい正本・保存用Schema・全域グラフではない。最低限、次を別列で残す。

| 閉包 | required / resolved / observed の比較 | 不足したときの扱い |
| --- | --- | --- |
| **Source** | Manifestが要求するroot、宣言・列挙されたfile、実際にparseしたSource bytes/span、参照先Revision、actual loaded/consumed inputs | 未読込・綴り誤り・対象外suffixを「空の正常Source」としない。必要なInputが不明ならClaim=unknown |
| **Interpretation** | Language・Role・Operator ID/版・contract/implementation fingerprint、**推移eager依存**、Required Shape、diagnostics | 未Pin・同名Provider競合・同版実体差・eager-cycleはfail-closed。deferred Ref循環は別評価 |
| **Meaning** | Type/Attribute/Relation/Rule、引数、NULL/単位/精度、作用順序、Predicate/WHERE/ORDER/JOIN/NOT EXISTS、grain/scope/completeness/time、独立Expected | Data ID・出力型・出力値だけでは同値未証明。hidden/negative依存の未証明はunknown |
| **Realization / Consumer** | Target/Builder/toolchain、declared graphとactual inputs、ArtifactSet全必要member、Readerの実受領、現在Principal/目的/開示/Grant | Build cache、歴史的Evidence、現在表示・実行資格を混同しない |

判定は対象Claimに必要な閉包に限定する。verifiedな要素は再利用可能でも、未記録・unavailable・partialを「依存なし」と扱わない。各層の資格は独立して `verified / failed / unknown / not-run` 等の既存表示に接続し、新規の一つのPASSフラグへ圧縮しない。内部の秘匿Input/Reasonを無条件にConsumerへ開示しない。

### 6.3 同一Subjectでの最小縦断試行

既存Go Reference PR #2639 の `sales.order` / `sales.order.amount` を**候補**とし、開始時に最新main、実Workspace、Source revision、実Language/Operator Provider/LockとCLI能力を再確認する。独立Expectedは候補Role/Operator自身から生成しない。Source A（正規TEI）と別表層 B（初期はtest-only）を比較する場合、Authoring文面/Source spanは異なるが、評価対象のSemantic Subject・必要なMeaning属性・Role/Operator Pinを同じ比較票に固定する。

| ID | 反証入力 | 必須判別／既存Owner |
| --- | --- | --- |
| **SDX-09** | Workspace manifestの必要rootが存在しない、綴りが異なる、管理一覧のfileがParser未読込 | `validate ok` や空集合を完全なSource closureと誤認しない。#2823/#2600 |
| **SDX-10** | Source移動、同内容に戻るA→B→A、参照先改版 | occurrence/locator/RevisionとMeaning/Acceptanceを分離。#2505/#2621 |
| **SDX-11** | 同Data ID・同出力型のままAttribute型/Relation/WHERE/ORDERの非出力依存だけ変更 | Binder結果一致から意味同値・Build skipを推論しない。#2639 EG-09/#2586/#2600 |
| **SDX-12** | 同CarrierにModel/Test等の複数Role、引用中Role文字列、同名Provider追加 | 明示Use-site binding・候補発見と固定選択・開示を区別。#2620/#988/#2621 |
| **SDX-13** | 誤った演算をExpected生成にも使い、同じ誤りを共有する | Self-consistent PASSを独立Oracle PASSと認定しない。#2514 TAO-01〜04 |
| **SDX-14** | Role/Operator v1同文字列で実装byte変更、推移依存だけ変更、無関係Plugin追加 | 必要依存の資格変更と固定解釈の安定性を分離。#2638 RB-20〜24/#2620 RE-01〜10 |
| **SDX-15** | Stage1/Stage2が一致してもseedが未検証、または未宣言Build input/Consumer必要memberが欠落 | 収束≠seed trust、Artifact一致≠Consumer利用資格。#2831/#2600 |

比較する同値性は **構造 → typed meaning → 用途別観測（値・診断・失敗・read scope・Effect）→ 実Artifact** の順に限定して主張する。構文の違う二つの式で同じ値が出ても、NULL/丸め・Predicate・選択粒度・副作用・failureが異なる場合は「意味同値」とはしない。あらゆるContextでの完全同値を有限テストが証明したとも主張しない。

### 6.4 隣接Ownerへの引渡しと先行順

- **E0（#2620/#988）**：実Go Language Selection、Role/Operator exact Pinと起動依存の実情報をInspect。既存Role Eager依存をProduct側が表せない場合は `unsupported/gap` と明記し、第二Resolverを作らない。SDX-09/12/14の負例を先に固定。
- **E1（#2620/#2514/#2621）**：同じSemantic Subjectで正規TEIと第二表層のCandidate、Source Span、独立Oracle、必要属性を比較。引用を勝手にRole化せず、文書上の原文を保持。SDX-10/11/13を検証。
- **E2（#2586/#2600/#2516）**：Meaning・非出力依存・Source実読込・Consumer用途・現在Grantを、既存Read/Build/Claim Evidenceへ接続。正しい局所再資格化とunsafe skipの双方を見分ける。
- **E3（#2831）**：同じaccepted/locked Source/Role/Operatorの版を既存BuildIntent→Resolved Plan/actual runner入力/ArtifactSetへ結ぶ。stage0 seed→stage1/2照合と未宣言工程を検査し、自己解釈の成功をTEI全体セルフビルド成功と混同しない。
- **可読性比較（DSL Owner）**：Go ASTへの寄せや専用構文の採用を先行決定せず、同じ課題・独立Expected・未取得/NULL/曖昧負例でMarkdown表/簡潔な式/候補DSLを生成AIによる同条件評価に掛ける。日常DSLへ `struct`/`func`/`package`/`import` を強制しない。

**受入記録の最小項目**：exact repository HEAD / tested commit、実Executable/CLI entry、実Source・引用元のbyte/Revision、Subject/Role/Operator/Impl identities、必要閉包 vs 観測閉包、独立Expectedの出典と相関警告、実Consumer/Provider/Target Scope、PASS/FAIL/UNKNOWN/NOT RUN、Owner受理と現在Authorityを別欄に記録。

**責務・優先順位**：本節は #2620 が受け持つ自己記述の接続仕様・反例だけを所有する。#2514がOracle、#2823がWorkspace診断、#2586が実Function縦断、#2600が増分資格、#2831が全体Buildの実証を担当。既存 #1119 P0-a/P0-b/G0〜G4 と #1127等の業務vertical sliceを新たなHard dependencyで塞がない。

**現状**：設計追補のみ。SDX-09〜15の実TEI試験・新CI・実Provider/DB/Studio/SHO/Host／Owner意味受理・全体セルフビルドは **NOT RUN / NOT PROVEN**。PR #2638のbounded synthetic PASSおよびPR #2639の限定Go Reference結果は本節の一体縦断証拠ではない。


<!-- tei-role-self-revision-transition-20261105 -->
## 7. 2026-11-05 自己記述定義の改版・移行・自己更新の受入契約（計画／NOT RUN）

### 7.1 なぜ必要か
#2620/本書§6の「実際に解釈された必要閉包」に加え、**その解釈器自身が読むRole/Operator/言語定義が改版されたときの遷移**を区別する。形式の版と変換の一元化は既存[#1920](https://github.com/hat47x/tei-platform/issues/1920)、Source版間対応は[#2505](https://github.com/hat47x/tei-platform/issues/2505)、Ownerの意味受理は#267、レビュー/撤回の証拠は[#958](https://github.com/hat47x/tei-platform/issues/958) / [Draft #2829](https://github.com/hat47x/tei-platform/pull/2829)、Authoring Source往復は[#1124](https://github.com/hat47x/tei-platform/issues/1124)が担当する。本節はこれらの**自己記述特有の接続条件**を検査する。#1920がClosedであってもRole自己移行が製品上で完了したという証明ではない。

### 7.2 同じ「version」を一つに潰さない
以下を別のidentity/Claimとして扱う。新たな包括Revision Counter・永続的な第二Meta Schemaは追加しない。

| 軸 | 変更の例 | 主張できる範囲 |
| --- | --- | --- |
| **Definition Format Version** | DSL/Markdown/Role記述の形式・構文・属性表現の変更 | Parserが読み取れるか／明示的な形式変換が必要か。意味の受理を含まない（#1920） |
| **Role/Operator Contract Revision** | Required Shape、解釈条件、NULL・単位・grain・Effectの変更 | 同じデータから導かれるtyped Candidate/意味制約の差（#2620/#988） |
| **Exact Interpreter Implementation** | ID/versionが同じまま実Providerのbytes/依存が変更 | 旧Pinと実内容の資格差。実装の新旧比較は既存Extension Lock/実内容証拠（PR #2638 RB-20〜24） |
| **Source Occurrence / Reference Revision** | 文書移動、行再配置、Reference更新、A→B→A | locator/出典の現在性とSource Observationの差。Meaning IDと同一にしない |
| **Accepted Meaning / Review Binding** | Ownerが改版を受理／撤回、証拠が再束縛 | 候補の意味同値と現在のOwner受理は別。旧reviewを自動復活させない |
| **Build / Consumer / Host Qualification** | Toolchain、ArtifactSet、Reader、Grant変更 | 再構築・実受領・実行・開示資格は別。Stage 1/2が一致してもseed trustは未証明 |

「新形式でparseできる」「shapeが適合する」「意味が同等」「受理済み」「現在使える」の5段階を**独立に**検査する。たとえば新Roleが旧型を広く受け付けること（structural backward readability）から、旧FunctionのNULL/丸め/副作用の意味互換性を導かない。

### 7.3 既存機構による自己更新の段階プロトコル

新しいMigration Engineや自己変更可能な本番Interpreterは置かない。既存Workspace/CLI/Language Plugin・Read-only Preview・Semantic Acceptance・Build/Hostの責務へ順番に受け渡す**論理手順**とする。

1. **Freeze / Capture**：現行受理済み定義A、実Source bytes/参照revision、形式版、Role/Operator/Language Contractと実Implementation Pinの推移閉包、当時のMeaning/Artifact/Review Evidenceを固定。必要Sourceが未読込・部分取得なら「完全な比較」へ進まない（#2822/#2823/#2812）。
2. **Prepare B（隔離候補）**：Aを保全したまま、既存形式変換入口（#1920の担当範囲）からBを**候補として**生成。unknown future version・変換不能な属性・未指定NULL/既定値・欠けたSourceは診断し、形式版だけ上げない。編集可能な正本Sourceの無断上書きはしない。
3. **Dual Inspect / Compare**：AはAのexact-pinned Reader/Role/Operatorで、BはBのexact-pinned Reader/Role/Operatorで解釈。必要な同一Semantic SubjectについてShape、型、属性、Relation、Operator引数、NULL/missing/unfetched、金額精度、単位、grain/time、hidden predicate/scope、診断、Effects、依存閉包を比較。比較器自身がA/Bの未受理意味からExpectedを生成しない（#2514）。
4. **Disposition**：`equal-for-claimed-scope / changed / ambiguous / unsupported / unknown` の**判断ラベル候補**を既存Read-only差分／Acceptanceへ渡す。これらは新しいCore列挙型を採用する提案ではない。意味が同等でも、Owner受理・現在Host Grantを自動付与しない。未知や意味喪失があればPreview/保留。
5. **Explicit Cutover**：既存Authoritative Source・Revision一致を再確認し、Ownerが受理した範囲でのみ正本変更・新Lock/Builder・対象Artifactの再資格化へ。Consumerは必要memberを実読込・現在開示資格を再評価。Aの既存実行を黙ってBへ差替えない。
6. **Revert / ABA**：B→AのSource内容復元でも、間の変更・撤回・Provider Selection・Review/Grant・Referenceの履歴が消えたことにはしない。旧Artifactのbytes一致と、現在Meaning/Consumer資格・連続性は別。旧定義の再利用は必要閉包を再資格化して判断。

**停止線**：全段階で元Aの原文と歴史的Evidenceを保全。解釈ができなくなった場合はPlainDataと診断を可能な範囲で維持し、実行・受理・開示を暗黙に継続しない。Edit Preview・Semantic Acceptance・Host Effectは独立操作。Sourceを未収集／Referenceがpartialなら、その区間における「変更なし」は主張しない。

### 7.4 具体的負例：SDX-16〜23

| ID | 変更・圧力 | 正しい判別／既存Owner |
| --- | --- | --- |
| **SDX-16** | 形式版だけv1→v2に書換え、未変換の旧フィールドを放置 | 形式と内容が不整合。成功扱いせず旧Sourceを保全。#1920 |
| **SDX-17** | 旧版の欠落フィールドの暗黙既定値と、新版のNULL/未取得の区別 | 属性が「ない」「NULL」「空」「未取得」を黙って一本化しない。#1920/#2620 |
| **SDX-18** | Markdown表の並べ替え／改行やSource移動だけ変更 | Source位置は変わるが意味差分は独立判定。原文・引用先と現在性も別。#2621/#1124/#2505 |
| **SDX-19** | `sum`/roundのShapeと出力型は同じ、丸め・NULL・grain・失敗動作だけ変更 | structural compatibleでも意味同値は不成立または未証明。独立Expectedを使う。#2514/#988 |
| **SDX-20** | v1とv2のRole Contractが相互参照し、解釈確定に相手の完了が必要 | 即時eager-cycleを拒否。deferred Graph Refの合法な循環と区別。#2620 |
| **SDX-21** | 古いInterpreterが知らない将来形式版／変換時に一属性を黙って削除 | unsupported/情報欠落を診断し、変換成功や自動採用を捏造しない。#1920 |
| **SDX-22** | A→B→AでSource/Roleの内容digest再一致、途中でReview撤回・Evidence再束縛 | 歴史的Reviewを保持。復元digestだけで旧受理・継続性・現在利用資格を復活させない。#958/#2829/#2505 |
| **SDX-23** | BでのMigration Previewは成功、しかし正本Source競合、実Builder依存/Artifact一式/ReaderまたはGrantは古い | PreviewとCutover、実Artifact/Reader/Host資格を独立判定。#1124/#2600/#2831/#1174 |

**初回の最小受入**：同じ公開TEI Workspace/同じSemantic SubjectのA/Bについて、SDX-16/17/19/21の四つを「旧Source・独立Expected・実Format Reader・正負診断」に束縛。次にSDX-18/22のSource/Reviewの再資格化、SDX-20/23の解釈循環とBuild/Consumerを追加する。最初から実DB・Studio GUI・すべてのTargetをHard dependencyにしない。

### 7.5 成立を主張する条件と今後の作業

- **意味互換性の範囲**は、対象Input集合、NULL/未取得、粒度/時点/単位、Rule/Operatorの実装版、失敗・副作用、対象ConsumerのClaimに束縛する。有限CaseのPASSは全入力域・全Hostの等価性の数学的証明ではない。
- **移行のExpectedは独立**：#2514のIndependent Oracle/known mutantを使用し、Migrationが書き出したExpectedをそのまま自己採点することを拒否。特にhalf-even/half-upなど同形で結果の異なる負例を入れる。
- **編集Sourceを正本にする**：#1124のScript-firstと連携し、Studio/SHOはSource span/Revision・可逆Patch・差分Preview・確定受理を別に扱う。新RoleのGeneric UIを自動発見しても書き戻しが可逆と実証されなければread-onlyとする。
- **Go単一バイナリは標準実行形態**。DSL表面は可読性を最優先とし、Go ASTとの親和性を採否の第一条件にしない。別TargetのJava/.NET/Wasm等は既存Lowering/Builderへ委譲し、自己更新の一つのPoCに全Targetを要求しない。
- **引継ぎ**：#2620はSDX-16〜23と自己更新時の必要閉包、#1920は既存形式版変換のOwner（Closedを新実装の証拠にしない）、#2505は版間対応、#958はレビューとwithdraw、#1124/#2621はSource編集、#2514はOracle、#2600は選択的再資格化、#2831はセルフビルドの実証。**第二のRegistry/Resolver/Host/Migration Engineを作らない。**

**実行・Evidence状態**：本§7は設計追補のみ。SDX-16〜23を実TEI CLI/実Provider/Studio/SHO/Build/Ownerで実行していない。既存PR #2638の有限合成Reference PASSや#2639のGo Reference CI成功は、新版Role自己移行・意味同値・現行実行資格の合格証明ではない。PlanのGitHub保存はOwnerによる正式意味受理ではない。


<!-- role-self-description-test-packet-20261105 -->
## 8. 2026-11-05 自己記述の最初の「実接続」試験票：AI生成拡張とClaim上限（計画／未実行）

**今回の優先順位変更**：§6のSDX-09〜15と§7のSDX-16〜23は必要な反例を十分含む。さらに類似テストIDだけを増やさず、**一つの実TEI Workspace/Semantic Subject/Source revisionから既存Go Product/Referenceに到達する実観測**を先に得る。AIがCaseを増やす自由は維持しつつ、Acceptedテスト正本や独立ExpectedをCandidate生成で書換えない（[#1180](https://github.com/hat47x/tei-platform/issues/1180)、[#2514](https://github.com/hat47x/tei-platform/issues/2514)、[#2813](https://github.com/hat47x/tei-platform/issues/2813)）。第二Scenario Runner/JSON DSL/Role Resolver/Permission Engineは作らない。

### 8.1 論理的不変条件：Required / Observed / Undischarged / Claim ceiling

これは**既存OwnerのEvidenceを読み取るための受入票**であって、新しいCore Schema・全域Proof Engine・常時実行されるContract Evaluatorではない。

- **Required(C)**：Claim C（例：このSourceはこのRoleで型付き意味として検証された）を主張するために必要な項目。対象Source root/実内容・選択Language/Role/OperatorのPin＋推移eager依存・Required Shape・属性/Relation/Rule/NULL/grain・期待の独立性・用途/Consumerを列挙する。**要求集合の定義自体**が未確定なら、空集合とみなさず `unknown-requirements` とする。
- **Observed(C)**：同じSubject、Revision、Scope、Tool/Interpreter exact identity、実実行/実読込に束縛できたWitnessのみ。Manifestで「指定」した、管理一覧で「列挙」した、Roleが「純粋」と自己申告した、GitHub CIの別HEADでPASSした、という事実を実観測へ代入しない。
- **Undischarged(C)**：RequiredのうちObservedに**十分に**対応していない項目。missing / partial / unavailable / stale / permission-denied / unsupported / not-run を区別する。必要なWitnessはあるが対象・時点・解釈版が違う場合も未充足。非公開の内訳はConsumerへそのまま出さない。
- **Claim ceiling**：満たした証拠で主張できる最上位の範囲。例：`data-inspectable` → `syntax-candidate` → `role-shape-candidate` → `typed-candidate` → `independently-tested` → `owner-accepted` → `consumer-qualified`。これらは**説明用の段階**であって新しいグローバル状態enumではない。上下の段階は別Ownerの証拠を要するため、自動昇格しない。実Effect/Commitはさらに独立観測。

**要件上の空集合と未検査は異なる**。`Required=[]` は「該当するClaimで義務がないと独立に確定できた」場合だけ許す。必要Sourceを0件しか読んでいない、Worker/Pluginが無かった、部分取得で0件となった等を完全性の証明にしない（#2823、#2586）。逆に、不要と証明された契約まで全世界で再検証しない。

### 8.2 実コードを再利用する最短経路

| 実入口・証拠 | 現在確認できる実機能 | 未成立の自己記述Claim |
| --- | --- | --- |
| `reference/dsl-language-plugin/selection.go` の `ParseSelected` | 明示Language ID/契約版・contract/implementation fingerprintの**すでに固定された**選択 | SourceのRole解釈、正式Accepted Lock/Owner受理、実Parser byte真正性までのE2E |
| 同 `ParsePipelineSyntax` → `LowerPinnedPipeline` / `ResolveOperator` | Source byte span付きPipeline syntax、固定Operatorの型連鎖とpure-only Effect拒否 | where predicate/引数、NULL/grain、Role Required Shape、一般的な演算法則の同値 |
| `plan/fixtures/role-bootstrap/check_bootstrap.js`（PR #2638 main統合） | pinned Role候補・eager/deferred循環・同版実装差の**合成24件** | 製品Role Resolver・実Extension Lock・実Workspaceでの自己解釈 |
| 実 `tei validate -json -workspace <workspace>` および `tei workspace-files` | Workspace操作と別々の管理一覧／検査結果を取得する入口 | `ok`=必要Source全件が実Parsed/Consumedとの保証。#2823の未読込負例を先に検査 |
| 既存Scenario `verify/generate/run/check`、AI JSON候補 (#1180/#2813) | Scenario/DSL候補を増やす既存足場。公開版・実Consumerは別 | 追加件数=独立Oracle/Mutation判別/Accepted Scenarioとしての資格、ではない |

**同一Subjectへ収束**：既存`reference/workspace`の`sales.order`／`sales.order.amount`を初期候補にするが、実Runtimeで読込まれたことをまず確認する。#2639 EG-08/09のtest-only JSONと、実Semantic Namespaceの対象・Attribute Type/Source spanを実Workspaceへ結ぶ。Role Definitionが標準データとして表現されていても、現Goの実Interpreterと受理経路に接合できない場合は `unsupported/gap` を残す。**合成RBケースと別SubjectのEGケースを加算して自己記述完了と報告しない**。

### 8.3 AIが増やせるテストデータと、AIに書換えさせない期待値

既存のScenario/Fixture正本の下に**候補Case**を追加する。特定JSON形式を新しい製品標準にはしない。Caseの論理的な最小列は次の通り。

| 項目 | 例／要求 |
| --- | --- |
| `case_id` / `candidate_revision` | 重複ID、旧Case混入、新旧世代混在を拒否。個々のCaseと採用されたSuiteの版を分離 |
| `source_ref` / `subject` / `role_use_site` | 実Source root/occurrence、固定Semantic ID、明示Role/Operator Selection。引用や説明表を実定義へ暗黙昇格しない |
| `change` / `required_obligations` | 変更した属性・形式版・実装/依存・元Sourceと対象の必要閉包。対象外変化を失効と自動判定しない |
| `expected` / `oracle_provenance` | 検証対象と独立に作成したExpected/既知Mutant、判別Scope、未定義事項。生成AIによる候補Expectedは未受理 |
| `actual_binding` / `observed` | exact executable/argv、Source bytes/Workspace、Provider/Interpreter fingerprints、actual consumed/read、診断・失敗・結果 |
| `undischarged` / `claim_scope` / `acceptance` | 不足義務、主張できる範囲、Owner Meaning Acceptance、現在利用/開示資格を別記録 |

**初期の独立対照（別Origin）**：
- A：`amount` の型は`decimal`、Role/OperatorのPinも同じだが、説明/テスト/引用のSource位置だけを変更。Source locator/文書の資格は更新、必要な型付きMeaningを無条件に全面失効しない（SDX-10/18）。
- B：同じData ID/出力型/Result表示でも`amount`のTypeRef、丸めhalf-even→half-up、`WHERE amount > 0`→`WHERE amount >= 0`のいずれかを変更。丸めの独立Expectedを含め、0境界・1.005・NULL等の既知Mutantを判別し、内容/出力型一致だけで同値にしない（EG-09/SDX-11/19）。`std.where`の実Go ReferenceはPredicate意味を十分表現しないため、ここは現状のgapとして記録。
- C：必要Rootが未読込、Required Shapeが無い、Root/Eager依存のPinが古い、SourceがA→B→A、またはOwner Reviewがwithdrawされた。元の値・版文字列・hashに戻っても旧受理/Source連続性/現在利用を自動復活させない（SDX-09/14/22）。
- D：同じ受理済み計算についてDB別Native SQL実現やGo/Java等Targetを差し替える。**意味上のQuery/Function Identity**と**Provider Dialect・実Artifact/Host資格**を独立比較し、同じ結果型やSQL名だけでProvider間意味同値を主張しない。DBプラグイン・2waySQL固有機構の実装は各Ownerの既存Realizationを利用する（#2619/#2622/#1121）。初回自己記述P0の必須条件にはしない。

**独立採点**：Caseを何件生成したか、実際に適用・実行したか、既知Mutantを何件検出したか、要求したObligationの分母の何割がEvidenceで満たされたか、Unknown/False PASS/不要な全面失効を別計測。生成Candidateの自己整合PASSは正式受理件数へ含めない。Case/Fixture/Expectedの同じ公開世代が実Consumerへ届いたことも#1180/#2604/#2516の別Ownerで確認する。

### 8.4 このPRでの最短着手・終了条件

1. **P0a 現行実入口Inventory**：main exact SHAとPR #2639のHEAD/状態を再取得し、#2638の合成JSと実Go Language/Operator Referenceを**別対象・別結果**で実行記録。実Product CLIが読み込んだSource集合と `workspace-files` / requiredを比較し、#2823の未読込/空成功を明示。
2. **P0b 一つの実Semantic Subject**：`sales.order`を実Workspace/Namespaceの同一revisionで固定し、Role Descriptorを通常データとしてread-only inspect → 明示Selection/Role binding/Required Shape → 型付きMeaning候補まで既存機構でどこまで到達できるか観測。**Role Contractやeager依存が実TEIに無ければ具体的gap**として止める。
3. **P0c 独立ExpectedとAI Case対照**：#2514/#1180の既存受理済み/独立Caseを保持し、AIが新しい変更負例を追加する。構造同値・意味同値・Source版・現在資格を別採点。Parser/Operator合成PASSをOwner受理へ昇格しない。
4. **P1 残余接続**：Productの実Query/Context、Studio/SHOのread-only Projection、#2831 Build graph、Target差／Provider差へOwner別に受け渡す。Go単一バイナリが標準実行で、Role動的解釈を各Requestの必須hot pathにしない。実行時変化のあるGrant/DB/Effectは#1174の現行Fenceで再照合。

**完了と言える最小Claim**：一つの実TEI Subjectについて、実Source・selected Language/Role/Operator・必要なSource閉包・typed Candidate・独立Expected/負例の対応が**同一revision・同一用途**で確認された範囲だけ。Role実装が存在しない場合は正確な`gap`までを成果とし、Product自己記述の完成とは呼ばない。AI Case自動採用、実Target/DB/Host、Owner意味受理や全体セルフビルドは別資格。

**状態**：この§8の追補だけでは新しいCase/Go/JS/CLI/Provider/E2Eは**NOT RUN**。#2813や#2639の他HEADのCI合格、#2638の24 synthetic PASSをこのP0a〜P0cの実結果へ流用しない。本件の設計は新しい正式DSL構文／Meta Schema／Accepted Meaningを採用するものではない。


<!-- self-description-real-go-fit-20261105 -->
### 8.5 実Goの具体的な既存構造とのFit/Gap（2026-11-05 Source-static）

前節の「抽象的なRole Descriptorを全く持たない」等の一般化を避けるため、**mainの実コードを読んで確認できる限定能力**を基準にする。取得した実コードは `internal/semantic/model.go`、`internal/workspace/loader.go`、`reference/dsl-language-plugin/{selection,pipeline_syntax,pinned_lowering,operator}.go`、`cmd/tei/main.go`。本表は静的読み取りであり実Go/CLI/Targetを動かした証拠ではない。

| 実コード／構造 | 正しく再利用すべき既存能力 | 自己記述に残る不足・反例 |
| --- | --- | --- |
| `semantic.Namespace.FormatVersion` | 0（Semanticファイルにheaderなし）の場合は**Workspace Manifestの形式版を継承**。明示header時はManifest版と整合する契約。旧#1920の「各readerが勝手に無版defaultを仮定しない」と両立させる | Semantic header不在だけを一律invalidにはしない。Manifestの存在・実Load・版整合を同一Attemptで確認。形式版≠Role意味契約版 |
| `semantic.Attribute.Role string` | 個別属性にRole値を保持可能。既存Data/Attribute/Relationの構造を再利用 | 文字列Roleがあることと、汎用Role DefinitionのRequired Shape、実Interpreter Binding、eager依存の解決、複数Role競合、Owner受理は**別**。第二Role Resolverを勝手に要求しない |
| `semantic.Attribute.Value *string` | 宣言属性の **Absent / Empty / Value** を形式上区別（nil / pointer to empty / pointer to nonempty） | SQLのNULL、未取得、非開示、Runtime Unknownとは別。3状態の既存フィールドを4〜5状態へ無条件拡張せず、各意味Ownerの適用条件で比較 |
| `workspace.Result.sourceSnapshot`・`semanticSourceFiles`・`sourceManagedExtra` | 同一Load内の実Parsed Sourceと、管理されるが未ParsedなSourceをprivateに区別するための構成がある | `workspace-files`列挙やprivate snapshot内容だけでFull Input Closure・外部Consumer開示を主張しない。必須root/未Parsed/実Build消費の違いは#2823/#2812/#2595へ委譲 |
| `workspace.Result.ManifestIdentity()` | 実LoadしたManifestのPath・SHA256を読取る既存入口 | Manifestは既存Source Indexに混入しない。Manifest bytes/要求root/actual parse/Provider closureの充足を別立てで確認し、Source Index不変で必要入力も不変と推定しない |
| `cmd/tei/main.go` | `scenario`、`validate`、`workspace-files`、`source-index`、`plugin-lock`、`inspect`等の既存CLI入口 | これらの存在は`tei role-interpret`等の架空コマンドを要求する根拠ではない。既存CLI/Go APIから読取可能な範囲を先に観測 |

**修正した受入の焦点**：最初のP0bでは「Roleとして利用可能な属性値が既存Semanticに存在するか」だけではなく、(a) どのAuthoring Sourceと版で読まれ、(b) どの既存Syntax/Role/Operator実装と固定Pinにより、(c) どのRequired Shape・Attribute Type/Ruleを経て、(d) どのtyped Candidateとして得られたのか、という実接合を確認する。Roleが文字列だけで、Required Shapeや適用実装への橋渡しが無いなら、その**正確な位置のgap**をProduct採用のOwnerへ戻す。

**変換互換の注意**：`formatVersion` header省略→Manifest継承という既存の明示的な意味を、旧設計メモの「版が無い=必ず拒否」へ後退させない。逆にManifest itselfが未読込/不正/未知の版なら、ヘッダ省略だけで無条件に採用しない。未知属性の黙った脱落、版番号だけのアップグレード、A→B→Aからの旧Review復活は別の負例として維持する。

**証拠範囲**：main Source-staticの構造・コメントによるFit/Gap確認のみ。実WorkspaceでのRole意味解釈、全形式移行、Owner受理、実Content/Implementation fingerprint真正性、実Provider/Consumer/Hostは**NOT RUN / NOT PROVEN**。


<!-- role-self-description-real-go-source-index-test-20261105 -->
## 9. 2026-11-05 実Go Product Loader→Source Index→Language Reference最初の接続（テスト追加・CI結果は別）

[Draft PR #2860](https://github.com/hat47x/tei-platform/pull/2860)に、`reference/dsl-language-plugin/self_description_workspace_boundary_test.go` を追加。**既存Go実装のみを使用**し、Role Resolverや新しい意味Canon/Parser/CLIは作らない。§8のP0a/P0bを一歩進めた実コードテストであり、Role自己解釈のP0b完成ではない。

### 9.1 実コードで固定したもの

- 公開`reference/workspace`を既存`workspace.Load`で読む。`ManifestIdentity().SHA256`が存在し、`Sources["semantic-data:sales.order"]` が `semantic/sales/order.tei` を指すことを**独立Expected**で検査。
- `workspace.BuildSourceIndexFromLoad`で、**同一Load時に保持されたbyte snapshot**から `sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}` の参照を確認。これは単一Workspace全体のatomic snapshot、ImportsのFull Input Closure、外部Consumerの開示・実Readを証明しない。
- 同じ公開FixtureのSemantic原文を**別途再Readしたテスト入力**として `syntax.NewParser(...).ParseSemanticFile()` を既存 `ParseSelected` に接続。test-onlyの固定Language ID/Contract/Implementation文字列は**実Parser byteの真正なFingerprint/Accepted Lockではない**。実LoadしたNamespace/Attributeの ID/型と照合するが、別途Readしたbyte列をLoaderのprivate snapshotと同一と**無条件には主張しない**。
- `sales.order.total` の独立基準 `TypeRef=std.money` と元の `Role=""` を検査。Source中の`role accounting_amount`追加でAttribute.Role文字列が現れることを試す。ただしRole Required Shape/解釈実装/Owner受理/Effect/Host権限は解決しない。
- 同じAttribute IDを保ったまま`std.money`→`std.text`の変更を解析し、ID一致が型の一致を意味しないことを検査。丸め・NULL・金額の業務意味が等価/非等価であるかは別Oracle/正式Operator Contractが必要。
- テスト用Semantic `formatVersion 2`を実Go Parserでは受け取り、`formatVersion: 1`のManifestと組み合わせた**実`workspace.Load`**で版の不一致を拒否する負例を追加。header省略（FormatVersion=0）のManifest v1継承は実存契約として維持。v2 WorkspaceではSemantic Header必須という既存実Loader条件と矛盾させない。

### 9.2 合格Claimと次のGap

| 既存実コード経路 | 今回テストが検証する主張 | 今回は未証明 |
| --- | --- | --- |
| Workspace Loader + strict Source Index | `sales.order` の実Loadと同一Load時Source locator | Required rootの全Coverage、Imports・Providerの完全Closure、Atomic snapshot |
| Real Semantic Parser via test-only exact Selection | `sales.order.total`のID/TypeRef/Roleと限定的なSource変更差 | 正式Language Plugin Selection/ImplementationFingerprint真正性、Role定義の自己解釈・Owner Acceptance |
| Parser-v2 candidate vs Manifest-v1 Loader | ReadabilityとWorkspace形式版の適格性は別 | 任意のv1→v2完全Migration、将来版・全互換性、実Cutover |
| 独立Sourceのtype-mutant | ID同一≠TypeRef同一を検査 | 全意味同値、属性依存の完全取得、丸め/NULL/粒度/Query Predicateの法則 |

**次段P0b**：同じ`sales.order`を保持し、正式に選択されたLanguage/Role/OperatorとRole Required Shape（実契約として存在するもののみ）を同一Source revision・必要閉包へ接続。未実装なら `unsupported/gap` を場所付きで提示する。#2639の別BranchにあるEG-08/09と異なる実Product Loader経路を提供するが、**両PRの局所PASSを一つのRole自己解釈E2Eに合算しない**。Ownerは #2620/#988、独立Oracleは #2514、Missing Sourceは #2823、Consumerは #2516。

**実行状況**：GitHubへGoテストSourceを追加・再取得したことだけ確認。GitHub Actions等の**exact HEADテストが完了するまではPASS/FAIL未判定**。このテストの成功でも本番Go Single Binary、実DB/Host、正式Role自己解釈/Owner受理を証明しない。


<!-- role-self-description-independent-obligation-basis-20261106 -->
## 10. 2026-11-06 自己記述の非自己証明契約：義務の出所・実読込・実Interpreterを分ける（限定実証＋未実証）

### 10.1 これまで不足していたもの

§8の `Required(C) / Observed(C) / Undischarged(C)` は、`Required(C)` **自身を誰が決めたのか**を区別しなければ循環する。AIが作ったRole Descriptor、未受理なPlugin Manifest、テスト対象DSLが「検査項目0件」と申告しただけで、検査が不要と認定してはならない。反対に、受理済み契約が本当に当該Claimへ適用されない場合は、無関係な検査を強制しない。

本節は既存Evidence／Qualification／Contractの**read-only照合規則**であり、新Core型、Global Proof Registry、Runtime Policy Engine、第二Resolverを追加しない。必要義務は次の発生源と効力を別列で保存する。

| 領域 | Requiredの出所（自己申告に任せない） | Observedの取得・Claim上限 |
| --- | --- | --- |
| Workspace/Source | 実LoadされたManifestと、対象Domainの既存Source Readerの許可・要求root、明示Source/Import依存 | `workspace.Load`、`ManifestIdentity`、`BuildSourceIndexFromLoad`、parsed/managed/unparsedの差。未知root／suffix未読込は#2823、Import未束縛はstrict Source Indexで停止。manifestの列挙が完全性保証になるわけではない |
| Language/Role/Operator | **選択・受理された**既存Contract／適用use-siteとその推移eager依存。未受理なRole DescriptorはCandidateに留める | `ParseSelected` のexact Selection比較、実ParserのCandidate・診断・Source位置。Referenceのfingerprint文字列が実Plugin bytesの検証を代行しない |
| Semantic/Context | Ownerによって適用範囲が定められた意味契約、independent Oracle、型・NULL・Relation・Query Predicate/Grain/Scopeの既存要求 | `Attribute.Role`／`TypeRef`等の実Parsed Dataを観測できても、業務意味や網羅的意味同値を自動証明しない。#2514/#2586 |
| Realization/Build | 必要Target／Application selection／actual compile & link inputs、ArtifactSet必要member、実Consumer契約 | Catalog/Bundle/Lock/Build/Artifact/Readerの各実証を#1251/#1089/#2831/#2516で照合。未選択と実Link除外は別 |
| Effect/Current Authority | 外部Hostの現在GrantとProvider Operation/Effect Preconditions、Business Operation/Receipt/独立Readの契約 | Source/Meaningが受理済みでも実行Grantを新規生成しない。#1129/#1130/#1174/#1163 |

**定式化（新APIではない）**：対象Claim `C=(Subject, RoleUseSite, Operation, Consumer, Scope, Revision/Epoch)` に対して、独立に根拠を持つ必要条件 `R(C)` を定め、観測 `O(C)` と結びつける。 `R(C)` が未確定なら `unknown`。確定した必要条件が欠落・失効・不整合ならその条件を `unmet` とする。観測された条件が局所的にすべて満たされても、別Ownerの受理・Build・Runtime Grantへ自動昇格しない。`R(C)=[]` は「当該Ownerの有効な要求集合が空である」**積極的な根拠**がある時だけ認める。Candidate自身が空集合を出しただけでは不可。

### 10.2 実Go Referenceの接続改善と厳密な限界

`reference/dsl-language-plugin/self_description_workspace_boundary_test.go` の既存 `sales.order` 試験へ、別途 `os.ReadFile` した `semantic/sales/order.tei` の `sourceindex.Fingerprint(raw)` と、`workspace.BuildSourceIndexFromLoad` 由来の同一PathのArtifact fingerprintとの**明示一致検査**を追加した。対象Artifactが欠けた場合も失敗にする。これは後続 `ParseSelected` に渡した**そのraw byte**が、先行LoadのSource Indexに記録された対象file byteと一致するという**当該観測時点の有界Witness**である。

この一致が示さないこと：Workspace全体のatomic snapshot、Manifestの全rootをparseしたこと、Import closure、選択されたLanguage/Role/Operatorの真の実装bytes、Ownerの意味受理、Host Grant、実DB Effect。Source Indexは原Workspace manifestをArtifactに含まないため、`ManifestIdentity`を別照合し続ける。Loadに存在しないrootや読込対象外suffixを診断しない事例は#2823に委譲し、Roleテストだけで解消したとしない。

さらに、Reference `selection.go` の `Selection` は非空のContract/Implementation fingerprint**文字列が一致するか**を検証するが、実Parser実装バイナリをハッシュするAPIではない。`test-only-semantic-parser` を正式なPlugin Integrity Proofに変換しない。

### 10.3 次の受入反証 OBL-01〜07（固定するが新Runnerは作らない）

| ID | 反証入力 | Independent Expected・停止先 |
| --- | --- | --- |
| **OBL-01 自己空集合** | Candidate Role DescriptorがRequired Shape/依存を`[]`と自己申告 | 受理済み契約で要求集合が確定しない限りqualifiedにせず`unknown`。#2620/#988 |
| **OBL-02 Source未列挙** | Manifestに存在しないroot／誤綴り、またはmanaged fileがDomain未解析でも`validate ok` | 対象ClaimへのSource coverage未成立。#2823/#1163。空集合自体をPASSとしない |
| **OBL-03 実装文字列詐称** | `Selection`の同一見かけのID/版/fingerprint文字列で実Parser bytesを交換 | Referenceのexact-match PASSを実Impl真正性へ繰り上げず、実依存byte/LockのOwnerで不一致判定。#988/#2600 |
| **OBL-04 独立Readずれ** | Load/Source Index後にfile内容を書き換え、別途ReadしParserへ渡す | `sourceindex.Fingerprint`差を検出し、同一Source Claimを失効。今回の追加は一致検査のみ。改変Fixture・multi-file atomicityは未実行 |
| **OBL-05 依存の自己削除** | Role更新で必須なeager依存をDescriptorから消し、旧Roleと同じ出力を得る | 未受理Descriptorを義務正本としない。旧契約・新契約の適用／失効を別判定し、出力同一で義務を減らさない。#2620/#2600 |
| **OBL-06 自己Oracle** | Interpreter/GeneratorとExpected作成が同じ誤った演算を共有 | 独立Oracleが無ければ`independently-tested`不成立。既知Mutantを#2514/#1180へ引渡し |
| **OBL-07 当時の成功** | 過去のAccepted Meaning／Build Artifactは有効だが、現在のPrincipal Grantが失効 | 過去のMeaning/Artifact証拠は保持、現在の実行・開示は拒否。#2516/#1163/#1174 |

これらは§6 SDX-09〜15、§7 SDX-16〜23、Issue #2620 RBX-01〜07の**追加のテスト正本ではなく、合否判定を誤らないためのProtocol**。実際の独立Expected、Case登録、Test executionは既存#2514/#1180/#1163へ。特に、`R(C)`を評価する独立Rootが製品に存在しない場合は`unsupported/gap`と報告し、そのためだけに汎用Meta Interpreterを作らない。

### 10.4 最短実施順とExit Gate

1. **E0（本PR #2860の限度）**： `sales.order.total` のGo Testのfresh-read fingerprint突合を実行し、exact HEADのFocused test・Repo Healthの結果を保存。Source位置、Manifest SHA、Source Index Artifact fingerprint、Parser candidate Type/Roleを同じテスト範囲に固定。
2. **E1（#2620/#988/#2823）**： accepted Role Required Shape/Selectionとその要求義務の**独立した出所**が実既存APIにあるかを棚卸し。無い場合、欠けた契約、受渡し位置、診断を一つだけ明示。後付けtest-only Role定義を製品正本としない。
3. **E2（#2514/#2586/#1163）**：同一Source/Subject/Revisionと独立Expected/既知Mutantを突合し、Source・Meaning・Consumer毎のClaim ceilingを別に記録。Provider/Host資格を先回りして認定しない。
4. **E3（#1130/#1251/#2831）**：Userの実Application/Targetに選択されたものと実compiler/link消費、Receipt/別ReadをOwner別に確認。全体セルフビルドは#2831の既存Build graphへ。

**実施済み（この§10編集時点）**：Roleの論理義務のOwner境界を設計し、GoテストSourceへ対象file byte照合コードをコミット。**未実施／未確証**：OBL-01〜07の完全試験、実Role Provider自己解釈、実Plugin byte真正性、全Manifest root coverage、Host/DB/Consumer E2E、Owner Meaning Acceptance。該当コミットのCIの結果は別に確認し、前HEAD `201cafab` のCI成功を新HEADへ流用しない。


<!-- self-description-source-cohort-regression-20261106 -->
## 11. 2026-11-06 同一Source読込の検査契約と段階的な利用資格

§10の「Requiredの独立した出所」に加えて、RequiredとObservedが**同じ実Source・版・試行**に属することを確認する。Source path・Semantic ID・Role文字列の一致だけで同一Revisionとはしない。この節は既存#1163の一改訂Packetと#2514独立Oracleへ渡す照合条件であり、新Core型/Registry/Runnerを作らない。

### 最小の接合表

| 境界 | 実際に照合するもの | 未証明のまま残すもの |
| --- | --- | --- |
| Source | `workspace.Load`、ManifestIdentity、Load時Parsed bytes、Source Index Artifact fingerprint/範囲、後続独立Readのbyte fingerprint | 全root/Importの読込と複数fileの原子的Snapshot |
| Interpreter | 固定Language/Role/Operator Selection、実Input、Source位置、Required Shapeの独立出所 | test-only指紋文字列と実Parser binary真正性の同一視 |
| Meaning | Semantic Subject/Attribute/Relation/型、NULL・Scope・grain、独立Expected・負例 | Candidate自己整合とOwner意味受理の同一視 |
| Consumer / Effect | 実Build入力・Reader受領、現在のHost Grant、Operation/Receipt/独立Read | 過去のGreen CI、RunPlan終了から実Commitや業務完了への昇格 |

### Go負例の追加（本Draft）

`TestSelfDescriptionRejectsStaleDirectReadAsLoadTimeSource`を追加した。Temporary Workspaceの`semantic/probe.tei`をLoadし、strict Source Indexを生成した**後**、同じ相対Path・同じSemantic IDを保持して`std.text`を`std.money`に書き換える。独立再Readのfingerprintが初回Indexと異なり、古い`Result`から再生成したIndexは旧byteを保持し、後から解析したCandidateのTypeRefだけが変わることを確認する。実時刻競争を使わない順次的な負例である。

このテストで主張できるのは、**当該LoadのSource Indexと後読込Sourceが同版かを検出可能**という局所性質である。製品Runtimeが自動で差を拒否する保証、全Workspace閉包、Owner受理、実Plugin真正性、Provider/Host資格を示さない。

### 他チャットとの責務接続と停止線

- #2823：Manifestの未知rootや対象外suffixが黙って無視される問題はWorkspace側で解決。Role Testが正常でも全Required Coverageとは呼ばない。
- #988/#2620：Role/OperatorのRequired Shapeが正式な受理済みContractに到達できなければ、Gapを確定する。test-only `Selection`に実装真正性を付与しない。
- #1251/#2885：通常の純粋演算やOwner規則はPlugin化を前提にしない。選択Contract、配布Bundle、実コンパイル/リンク入力、実行資格を分離する。
- #2586/#2622/#2862：Semantic QueryとDB別SQL、実Provider Read・NULL/Scope/完全性を別資格で扱う。
- #2514/#1180/#1163：独立OracleとMutant、必要条件の適用範囲と実Consumer/Attemptを同一版で結ぶ。局所PASSを加算して新しい横断合格を主張しない。
- #1130/#1174：read-only Role適用からEffect・Dispatch・Current Grantを生成しない。Commitが不明ならReceipt/独立Readの境界へ渡す。

**実証順**：E0 今回の同file読込一致・時間差不一致のGoテスト→E1 Role Required Shapeの実API調査とGap→E2 独立Expected・Consumerへの一改訂接続→E3 実Provider/Target/Host。GoコードとDocsの保存は完了としても、新HEADでのCIを確認するまでテストPASSとは記載しない。正式Role自己解釈、Owner意味受理、実E2E、セルフビルドは未実証。


<!-- role-self-description-authoring-roundtrip-safety-20261106 -->
## 12. 2026-11-06 横断統合：自己記述RoleのAuthoring投影と安全な編集逆写像（設計・NOT RUN）

### 12.1 解釈可能と編集可能を分離する

§9〜11のSource／Role／Interpreterの同一改訂照合は、読取の適格性を高めるが、**Role as DataをStudio/SHO/Markdown表で表示できることは、元のDSL Sourceへ安全に書き戻せることを意味しない**。#2621（Authoring）、#2497（構造保存写像と編集逆写像）、#1124（Studio/Modeler）、#1803（SHO）、#2505（外部参照）への本Issue固有の接点を整理する。

日常DSLの可読性を最優先し、Go ASTを理由にGo固有構文を露出させない。Markdown/表/式/Graphviz/Chainなどユーザーが選択した糖衣表現は、元の原文として保持する。標準データCarrierやtyped Semantic Candidateは**Sourceの第二正本ではない**。構文からMeaningを再構成できても、意味から原Authoringのコメント・並び・記法を一意に再生成できるとは限らない。引用の表やコードフェンスは、形状が一致しても明示Use-site RoleなしにFunction/Rule/Test/Expectedへ昇格させない。

### 12.2 Read → Preview → Patch → Independent Re-read

この節のS/R/Vは概念上の比較票であり新API/新Core Schemaではない。SはAuthoring bytes・occurrence・Source Revision・Workspace/Manifest identity、RはRole Use-site・accepted contract・exact interpreter selection、VはPurpose相対のread-only投影。

1. **Read**：実LoadしたSourceのfingerprint、Role/Operatorの選択、Source span/locator、typed candidateと投影の情報損失（Identity、順序、重複、NULL、grain、隠れた依存、コメント、非可視列）を記録。既存semantic.Attribute.Role文字列の保存と正式Role Interpretationは分離。
2. **Preview**：View上の編集は**変更案**でありSourceの変更やOwner意味受理ではない。Stable occurrence、編集対象の一意なSource対応、適用可能な逆写像、保持すべき非対象情報、当該Editorの現在の変更資格がすべて判明する場合だけ局所Patch候補を作る。Role Descriptorが自身にeditable=trueを書いてもCurrent Grantにはならない。
3. **Apply**：Preview時に見たSource bytes/Revision、Manifest、必要なRole/Transformer版と実際の保存直前の状態を比較。drift、relocation、ABA、Role競合、権限失効、不完全な参照の場合、旧Previewから自動保存せず再解決/差分へ。EditorのSource変更資格とProvider/Host Action権限は別。
4. **Independent Re-read**：保存後の**実Source bytes**を既存Loader/Parser/Role/Source Indexで再読込し、対象属性と未変更領域を独立Expectedへ照合。Save OK、再Parse OK、Owner Accepted、Runtime Effectを別のClaimとして扱う。実Provider書込みはこの試行には含めない。

**条件付き編集法則（#2497へ委譲）**

- No-op preservation：変更がなければSourceへ書かず、元のSource bytes、糖衣表現、コメント、引用を維持。Canonical pretty-printerで暗黙書換えしない。
- Scoped PutGet：編集対象と対象Scopeの逆写像が明示されている場合のみ保存し、再読込したViewが変更予定部分に一致することを確認。同じSourceを使う他のRole/Comment/Ruleを黙って変えない。
- Loss-aware stop：集約、Graph→Tree、複数Source合成、同名ノード/多重Edge、partial read、hidden dependency、非可逆な型変換で元対象が一意に定まらない場合はread-only。必要なら既存#2500の型付きAction Intent候補へ委譲し、直接Entity/DB/Historyへ書き戻さない。
- Change-aware requalification：Authoring見た目・Source occurrence・Role binding・typed meaning・Oracle/Expected・Realization・現在の開示/編集資格を別に比較し、意味が不変でも引用/Editor資格を必要に応じ更新。不要な全体Build失効も無条件に行わない（#2600）。

### 12.3 既存Subjectを共用する判別条件（AP-01〜06：全件NOT RUN）

以下は第二Fixture正本／第二Runnerの追加ではなく、#2621/#2497と既存#1180/#2514の受入へ渡す**同一Caseの比較軸**である。まずPR #2860の公開Workspace sales.order／sales.order.total、同じSource Revisionを固定する。Markdown・Graph側は現状**候補表現**で、Product Parserの正式機能とはみなさない。

| ID | 対照条件 | 独立Expected／停止条件・担当 |
| --- | --- | --- |
| **AP-01 Roleが曖昧な表** | 同じ列を持つ説明表・Function表・Test Expected表 | 表ShapeだけでRoleを決めない。Explicit Use-siteが無ければambiguous/read-only。#2620/#2621/#2514 |
| **AP-02 原文保持** | Markdown装飾/行順/コメント・Graphvizレイアウトの非意味差分、No-op | No-opでは原byte不変。変更が必要な場合も糖衣を勝手にCanonical化しない。Source locator更新とMeaning変更は別。#2621/#2497/#2505 |
| **AP-03 型とRoleの部分編集** | sales.order.totalのRole注記だけ変更、別ケースでTypeRef変更 | 同ID/同Attributeの型・Role変化を別診断。Source spanが一意でなければApply不可。Re-readで局所変更と他領域保持を検証。#988/#1124 |
| **AP-04 一Sourceの複数Role** | モデル説明／Scenario／Oracleを同じCarrierから投影して一Roleだけ編集 | Shared Sourceの他Claimへの影響を列挙。未受理Expectedを書換えない。衝突が残ればPreview止まり。#2620/#2621/#2514/#2600 |
| **AP-05 Lossy view** | 3層→2層サマリー、Graph→Tree、JOIN/aggregate、partial Query | Entity/Relation/EdgeのIdentity・重複・順序・完全性が戻らなければ直接Save不可。明示Action Intentのみ候補。#2497/#2500/#2506/#2622 |
| **AP-06 Preview後Source drift** | 別EditorでTypeRef更新、外部Reference版差、A→B→A、編集Grant撤回 | 旧Preview/旧RevisionでPatchしない。再解決・再Preview・Owner Review/Current Grantを独立判定。#2505/#958/#1124/#2516 |

### 12.4 最短の実証・Exit

E0：既存#2860のactual Workspace/Source Index／Go Parserを同一Sourceで維持し、RoleとTypeRefの位置をread-onlyで観測。E1：元AuthoringのNo-op維持と、可逆／非可逆の差を同じ独立Expectedで比較。E2：**既存の安全なSource Patch機構が確認できた対象のみ**Preview→適用→独立再Readを一往復させる。不在なら具体的Gap/unsupportedを残し、新Editor/Role Resolver/自動Canonizerを実装しない。E3：Source改版後のMeaning/SHO/Studio/Runtime Consumerの資格を#2497/#2516/#2600へ引き渡す。

**証拠上限**：本節は設計・受入票追補のみ。AP-01〜06、正式Markdown Role Parser、Studio/SHO編集、Product Source Patch、Owner Meaning Acceptance、実DB/Host Action、全セルフビルドは**NOT RUN / NOT PROVEN**。#1119のP0/Gateと#2831のBuild責務を変更しない。


<!-- tei-role-candidate-collision-loss-20261106 -->
## 13. 2026-11-06 Role記述の衝突と候補化による情報落ち（実Go Parser特性＋計画）

### 13.1 新しく確認した具体的なGap

`internal/syntax/parser.go` の `parseSemanticData` では、AttributeのOptionを順番に読み、`role`に遭遇するたびローカル変数 `role` を更新し、最終的に `semantic.Attribute.Role string` へ格納する。従って同一Attributeに `role accounting_amount role audit_amount` を書いたとき、現Parserの**Candidate**は最後の `audit_amount` だけを持ち、先行宣言数／Source位置／衝突診断は `Attribute.Role` に保持されない。

これは特定の現行Go ParserとCandidateの性質であり、正式Role解釈が成立した証拠でも、「最後に書いたRoleを正とする」という採用済み意味規則でもない。将来的に多Roleをサポートする場合も、複数Use-site Roleの明示的なBindingと競合判定が必要であり、Source上の重複を黙って単一Roleへ正規化しない。

反対に `description "role privileged_amount"` は字面に `role` があっても通常の文字列であり、Semantic Roleを発生させない。Markdown引用・SHO説明・AI入力にも同じ非昇格原則を適用する。ただし本GoテストはMarkdown Parserの実証ではない。

### 13.2 実Go限定の対照

`reference/dsl-language-plugin/self_description_workspace_boundary_test.go` に `TestSelfDescriptionRoleLiteralAndDuplicateAreNotAcceptedMeaning` を追加。

1. 引用・説明文字列中の `role privileged_amount` を読み、`Attribute.Description` と `Attribute.Role=""` を独立Expectedで照合。
2. `role audit_amount` 単独と `role accounting_amount role audit_amount` の**異なるSource bytes**を固定し、Source fingerprintが異なることを観測する。
3. 現Parserが重複を受け付けた場合、双方のCandidateが同じID/TypeRef/Roleへ投影され、重複宣言の存在をCandidateだけでは復元できないことを観測する。将来Parserが重複を拒否した場合も、この負例テストはその安全な方向を妨げない。
4. 観測したGo Parserの局所性質を、正式Source-Aware DiagnosticやOwner受理へ拡張して主張しない。

### 13.3 Editor/意味解釈での停止条件

| 状態 | Read/Inspect | Meaning/編集資格 |
| --- | --- | --- |
| 単一の明示Roleと確定したSource occurrence | 既存GoのSemantic Candidateとして観測可 | Required Shape/Interpreterが未接続なら未受理。Source Writer資格も別 |
| 説明・引用中のRole風文字列 | 説明として表示可 | 自動Role適用／Provider/Host操作禁止 |
| 重複Role記述が原Sourceに存在 | 適切なSource位置付き警告/診断が必要（現Go単独未実証） | `Attribute.Role`だけで「唯一のRole」を確定しない。重複の解決とPreview前の診断までread-only |
| 同じRole Candidateだが原Source bytesが異なる | 原文・Source fingerprint・revision差を提示 | No-op/書戻し/Review/Build再利用をCandidateの同値から自動推定しない |

**Ownerへの引渡し**：#988は既存Language/Operator/Syntaxの衝突診断・Role適用契約、#2620はEverything is Dataと自己解釈の合格条件、#2621/#2497/#1124はAuthoringの可逆性・編集Preview、#2514は独立Expected/負例、#2600はSource差とMeaning/Build資格の選択的失効を扱う。第二Role Resolver、別AST正本、Markdown-to-Core強制変換は作らない。

**Evidence/停止線**：本節は現Go Source静的照合と追加された限定Go Testに基づく。重複Roleの正式診断、Role Required Shape/契約版、実Interpreter/Owner Meaning Acceptance、Studio/SHOの編集保存、Go Target/DB/Hostは未実証。このテストのCIは**追加後のexact HEAD**で別途確認する。現Parserの上書き挙動を正規仕様・合格とは認定しない。


<!-- role-self-description-draft-promotion-separation-20261106 -->
## 14. 2026-11-06 実Source編集のFit/Gap：下書き保存と受理済みモデルへの昇格を二経路に分ける

### 14.1 他レーンの実コードが示す既存能力と不足

§12は編集逆写像の一般的な条件だった。本節では現mainの実コードを静的照合して、**新Writerを作る前に既存の実入口を活用し、Roleに固有の欠落を小さくする**。最初の用途は公開Workspaceのsales.order.total。SourceのRole文字列とSource IndexのRoleDefinitionを混同しない。

| 実入口（実main） | 既に出来ること | この自己記述Claimに不足すること |
| --- | --- | --- |
| internal/syntax/semantic_anchor.go の ParseSemanticFileAnchored / semanticDataAnchors | semantic-attribute（属性**識別子**token）とsemantic-attribute-type（**型参照**token）の独立Source anchor。byte offsetを持つ | Role optionそのものの一意なtoken spanと重複Role診断。型や属性名のanchorをRoleのPatch位置に転用しない。既存Parserは重複Roleを最後の文字列へ上書きする（§13） |
| internal/workspace/source_index.go の BuildSourceIndexFromLoad | Load時byteによるSource Artifact digest、意味Subject→定義identifier range。strict indexはfilesystemへfallbackしない | 定義anchor≠属性option writable range。Index/Source locatorはOwner editing authorityでない。multi-file atomicity、imports closureは別Owner |
| tools/reference-shared-acceptance の set-attribute-type・Preview/Review/Accept | 既存型参照**だけ**の局所置換、baseModelRevision、overlay→reload→validation→semantic diff→review fingerprint→accept→source再読取、Scratch Source受理の既存例 | 一般的Role option編集op（set-role）／Role Required Shape／完全な実Human Acceptanceはない。原文全体のMarkdown CST/Writerではない |
| reference/studio-authoring/bridge.mjs と editor.mjs | ERのadd-relation、表のadd-validation等をCandidateへ変換、表示フォームはread-only proposal。旧expectedModelRevisionはreject | Role option局所編集は当該Candidateメニュー外。Editorが元ファイルを書き込むと誤表示しない |
| reference/tei-studio-editor/editor.ts と codemirror.ts | byte/UTF-16位置変換とSourceのread-only表示、CodeMirrorのreadOnly | 直接本文入力・保存は公開していない。WebView上で表示できることはProduct Authoring Writerの実証でない |
| Source Acceptance lock とDecision ledger | 協調するAcceptance同士のLock、受理結果のDecision記録／再照合 | 外部Editor/他processがLockを無視すれば排他できない。さらにAcceptの**Source更新後**にDecision recordの書込み失敗が起きた場合、Sourceを戻さず警告とエラーを返す設計である。「CLI失敗=Source不変」とみなさない |

上表はSource静的確認であり、ここで実WorkspaceのRole option edit／実Studio保存／外部Editor競合を試験済みとしない。既存型編集は**比較基準**であり、Role定義への自動汎用化を要求しない。

### 14.2 二つの保存操作を混同しない

**A. Authoring Draft Save（原文保存）**：編集者が原文を一時保存・再開する。Syntax Error、未定義Role、Plugin不在、途中のMarkdownを**原文のまま保持してよい**。説明用MarkdownやSHO下書きの保存にまでAccepted Meaningを要求しない。ただし保存する先の現在のWriter権限／実byte／競合・Source位置は確認し、他人の版を上書きしない。Draft保存成功は**正式Workspace SourceへPromoteされたことを意味しない**。不正なDraftからModel/Runtime/ConsumerへAccepted表示を作らない。Draft保存領域や実File I/Oの設計は#1124/#1803/#2621へ委譲し、この文書では新しい共通DraftStoreを規定しない。

**B. Semantic/Canonical Promote（意味を持つSource変更の受理）**：最新の受理済みSource R0と編集Intent、対象Subject/use-site、Source occurrence、**変更対象Role optionの正確なtoken span**、Role/Operator pin、独立Owner-required obligationsとCurrent Write資格を照合。既存Shared AcceptanceのCandidate→Preview→Review→明示Acceptが受け付ける**既存operationだけ**を通し、再読取後のtyped meaningと意図外byte変化を検証。未対応のRole option編集ではCandidate/read-only Diff止まりとし、型編集Operationに偽装して通さない。Plugin未選択・Role競合・未知Require Shapeなら新しいAccepted Meaningへ昇格しない。

両経路の**共通条件**はNo-opでSource書込みしない、Raw Sourceの糖衣・コメント・BOM/CRLF等を勝手に再整形しない、他Source・Owner資格を捏造しないこと。Raw Draftを直せる自由度とCanonical Promoteの厳しさは矛盾しない。形式的には Draft Saved ≠ Parsed Candidate ≠ Preview Acceptable ≠ Actual Source Changed ≠ Accepted Meaning ≠ Consumer Current ≠ Provider Effect。

### 14.3 非同期UI・外部Writer・記録失敗の扱い

- **UI後着Fence**（#1124/Desktopと接続）：編集/保存/Review/診断のResponseは、Source Revision、選択Subject＋Occurrence、Role/Interpreter版、Consumer Scope/可視性、Editor Session/request generationが一致する**現在の表示用途**にのみ適用。旧R0の成功通知・エラー・ボタン状態をR1の画面へ復活させない。既存表示の失効と新規描画拒否は別assert。Policy失効をConsumerが観測できない期間の即時撤回は保証しない。
- **競合**：accept lockは**協調的**。別Editorが直接Fileを上書きする場合を想定し、構造化Patchは実Source byte/ModelRevisionを直前に再検査し、保存後も別Readで照合する。全OS・全Processでのatomic CAS保証とみなさない。Source A→B→Aなら旧Review/Editor資格が復活したと推論しない。
- **Commitの表示**：実Shared AcceptanceではSourceの変更後にDecision ledgerの記録失敗が起き得る。終了コードだけで「保存失敗・変更なし」と報告せず、actual Source R1／Decision path有無／Warning／Review Binding／再読取を別表示し、安易な二重Acceptや勝手なrollbackを禁止。これはDB ProviderのCommit/Receiptと**異なるSource Authoringの局所事件**。
- **Editorの診断**：Syntax ErrorのあるDraftは復旧可能な原文として保存してよいが、そのまま実行・正式定義への受理を許すわけではない。未知Pluginやunresolved RoleはDraftの可視性とMeaning適格性を別表示。秘匿内容をDiagnostic/AI payloadへ漏らさない（#2516）。

### 14.4 判別に必要な最小実証

**E0 実Go Source anchor Characterization**：Draft #2860の TestSelfDescriptionAttributeAnchorsDoNotSubstituteForRoleOptionSpan に、実 ParseSemanticFileAnchored の属性名anchor=total／型anchor=std.money とRole値accounting_amountの**別位置**を比較する試験を追加。旧AnchorをRole Patch位置へ流用できないことを限定的に確認する（commit 6feb4b10）。将来専用Role anchorが追加された場合に拒否するものではない。Current Source Indexとreal Interpreter/Writerの編集閉包は別。

**E1 既存 Writerの再利用可否**（#1124/#1935/#2621）：Scratchに限り、set-attribute-typeの既存トークン編集・旧版拒否・再読取をbaselineとして同じRole利用位置の不足表に照合。No-opとRaw Draft Saveを別計測。Role writerが未実装ならRead-only / unsupportedのまま。

**E2 差分と停止線**（#988/#2514/#2497）：同一Role二重宣言／quoted role／同名別occurrence／Unicode＋CRLF／Source再配置／Role pin変更／独立Expected未成立を反証。Parser Candidateの単一Role文字列をSourceの唯一宣言と認めず、Source上で診断できない場合は意味の昇格を停止。

**E3 Product UI/Consumer**（#1124/#2516/#958/#2600）：旧Revision成功レスポンス、既存表示、同一内容でPolicy失効、原文保存のみ、Source changedだがDecision ledger記録失敗、を**別状態**として示す。Source Acceptが成功してもDB/Host EffectやOwnerの最終意味判断を証明しない。

**証拠上限と進行判断**：本§14は現mainの静的Fit/GapとテストSource追加を記録したもの。テストのexact HEAD CI、Raw Draft Product Writer、Role option Writer、実Markdown/SHO、実WebView、外部Writerとの原子競合、Owner Meaning Acceptanceは別に確認する。新Parser/Source Writer/Second Acceptance/Host/横断GateをこのDraftに実装しない。#1119のP0と#2831セルフビルドは変更しない。


<!-- tei-role-self-description-source-option-anchor-collision-20261106 -->
## 15. 2026-11-06 SyntaxでのRole重複停止と読取り専用Option Anchor（横断コード接合、Draft／CI別確認）

### 15.1 既存Core/Source Indexを変えずに実現できる最初の縦断

[Draft PR #2898](https://github.com/hat47x/tei-platform/pull/2898)は、#988の現main Parserで見つかった「同一属性に`role A role B`を書くと最後の値で上書きされる」欠落を、**二度目のrole keywordでSource位置付きSyntax Error**として拒否する小さな実装にした。これに加え既存`syntax.ParseSemanticFileAnchored`に`semantic-attribute-role`という**Role値のtokenだけ**を指すSource-side Anchorを追加した（前者は文法拒否、後者はSource navigationであり受理権限ではない）。属性名`semantic-attribute`、型参照`semantic-attribute-type`との区別は§14の既存Go Characterizationと整合する。

2026-11-06現在の実変更範囲は`internal/syntax/parser.go`、`semantic_anchor.go`、既存＋新規Syntaxテスト、Workspaceの重複Role診断接続テスト。SourceIndexの`sourceindex.RoleDefinition`は「そのSource Referenceが定義位置である」という種類で、`semantic.Attribute.Role`や`syntax`の`semantic-attribute-role`とは**別の概念**。`syntax.SemanticAnchored.Anchors`からRole tokenが得られても、その位置が自動的に`workspace.BuildSourceIndexFromLoad`の正規定義Reference、Writerの対象、Ownerの権限へ昇格するわけではない。既存`semantic.Attribute`の型、Generic Source Index、Source Writer、Role Interpreter、Plugin Contractを増殖させない。

### 15.2 異なるレーンから取り込む原則と、取り込まない実装

- **#2891 Scenario JSONの重複キー、#2900 DB Releaseの重複候補**：どちらも「入力を最後の値で上書きして成功と見せない」というFail-closedの反例。だが、JSON Object member／DB状態履歴／Attribute Optionは粒度・順序・多重性契約が違う。共通の万能重複Validatorは設けず、**重複の許否は各宣言の契約が決める**。Roleでは1属性につき同Option複数指定を拒否する。属性が違えばRoleが同じでも独立に記述可能。将来の複数Role use-siteは別の明示構文とOwner受理が必要。
- **#2497/高度組織モデリング**：Roleの宣言個数、Roleと主体／Procedureの結びつき、Source occurrence、Role解釈後の依存を分離。Graph/テーブルの同形表示、ID一致、同じ値でも多重性・適用時点・Source参照資格を消してはならない。書戻しが多対一ならread-onlyまたは明示Action。
- **#2823 Required Root/Source閉包**：現在Readerがたまたま解析した`role`が一意であることは**Workspace内のすべてのrequired Sourceが読み込まれた証明でない**。取込漏れ、外部Import、同Subject別Source、Plugin/Selection違いは別の不合格条件。
- **#2514/#2621/SHO**：説明用文書の`role`文字列、テストのExpected、実Role宣言をContext・引用位置で区分する。Parserが成功してもMarkdown/SHOの表を実FunctionやOracleとして自動採用しない。元Sourceの原byteと糖衣を保存する。

### 15.3 Requirementから観測までの四集合（新Core型ではなく受入票の列）

| 集合 | 記録する内容 | 禁止する推論 |
| --- | --- | --- |
| **Required**（外部受入要求） | Owner受理済みRole必要形・各Consumer/Claimに要求されるSource/Dependency、実Build/Plugin資格 | 候補Role自身が`required=[]`を宣言したから空でよいとはしない |
| **Declared**（Source上の申告） | 定義とRole Optionの原位置/byte/個数、欠落・重複、契約版 | 一個にParseされたからSourceに一個しか無いとはしない |
| **Resolved/Consumed**（実読取・適用） | Load時byte/Manifest/Parser/Plugin選択と利用されたDependency、見落としたroot、解釈失敗 | Source上で宣言されたことからReaderが取得した・解釈できたとしない |
| **Observed/Disclosed**（Consumerと出力） | 元Source/モデル版・実Output/Diagnostic・現在Policy/Scope・可視性・実Attempt | 表示成功や診断不存在をSource全体の成功、秘密情報の不在、Host/Provider実Effectとしない |

四集合は一致すればよい単純なHashではない。「必要なものが本当に閉包として読まれたか」をClaim相対に比較するための列である。Source byte、モデルRevision、Artifact Revision、Role/Operator Pin、Review/Consumer Disclosureの失効をそれぞれ扱い、同一digestへのA→B→Aで旧Review/Grantが自動復活したとは判定しない。

### 15.4 次に実証する受入ケースとOwner（既存Fixtureへ合流）

| ケース | 最小入力と独立Expected | 現状／担当 |
| --- | --- | --- |
| RO-01 | 同Attribute `role first role second`。**第二keyword**のbyte位置にParser Error、Source→Worldは不合格 | #2898でSyntax/WorkspaceテストSource追加、CI未確認／#988 |
| RO-02 | 同じRole値の二重指定、コメント越し、quoted description越し。上書き許容はしない | #2898 Syntax負例、CI未確認／#988 |
| RO-03 | UTF-8日本語、BOM/CRLF、quoted説明`role`、Roleなし属性が混在。Role**値**だけの実byte span、無関係textで架空Anchorなし | #2898 Anchored Syntax負例/正例、CI未確認／#988 |
| RO-04 | 有効SourceをLoad→独立Readする。Role use-site token位置とArtifact/Manifest現在性、Scope、Required Root閉包を比較 | SourceIndex / #2860の局所証拠のみ。**同一Workspace全root完備はNOT PROVEN**／#2823 |
| RO-05 | Roleの型・Required ShapeとImplementation Selection、Role用途・同形Markdown表を比較。Role不明または競合で正式Meaning拒否 | **NOT RUN**／#2620/#988/#2514 |
| RO-06 | Source A→B→A／Role Selection変更／外部Editor競合。古いEditor PreviewとReviewを再使用しない | **NOT RUN**／#2505/#2621/#1124 |
| RO-07 | ロスのある表・Graph ProjectionからRole値を変更。対象Occurrenceが一意でなければ書戻し拒否 | **NOT RUN**／#2497/#2621/#1124 |
| RO-08 | 正常Syntax診断がConsumerに届いても権限変更でDisclose不可、または旧成功Response後着。公開面がCurrent資格で判定される | **NOT RUN**／#1124/#2516/#958 |

**順序**：RO-01〜03のexact HEAD Syntax/Workspace focused + repository-health-checkをまず確認。その合格だけでRO-04〜08を合算しない。次に#2823の独立Required Rootとの比較→正式Role Required Shape→#2621/#2497への局所Source Patch提案→Consumer/Hostの実験へ進む。新Role Writer/Parser分岐/万能Registry/第二Oracle/追加Cross-cut Gate/TEI専用Hostは不要。mainはReviewerの検収まで変更しない。


<!-- tei-role-observed-source-disclosure-handoff-20261106 -->
## 16. 2026-11-06 観測不能なRoleを「存在しない」にしない：実Loader／公開Sliceとの境界（横断実装Fit/Gap）

### 16.1 進んだ二つの隣接実装と、依然つながっていない部分

- **[Draft #2935](https://github.com/hat47x/tei-platform/pull/2935) / #2823**：製品`workspace-files`に管理ファイルごとの`readByLoader`を追加する実装。**当該LoadでLoaderが読んだ**という事実を、Source inventoryから分離できる。HEAD `bd060d36`のRepository Healthほか既知のCIはcompleted/success。ただし`readByLoader=true`はSemantic parse成功・Roleを抽出した事実・Manifest上全required rootsの網羅・Plugin実消費を意味しない。Import先は個々のWorkspaceで別解決。未管理Source、未知Root、選択外Sourceもある。
- **[Draft #2972](https://github.com/hat47x/tei-platform/pull/2972) / TEA T06/T07**：**事前公開資格が与えられたSliceだけ**から三視点へread-onlyに投影。HEAD `61bdee4f`のRepository Healthはsuccess。Source path、未許可Edge/metadata、private countからのgeometry漏洩等を限定テスト。だが現在のPolicy/Grantを判断してPrivate WorldからSliceを生成する実処理は**未実装／未実証**。Sliceの`accepted`という文字列はOwner Meaning Acceptanceの証拠にならない。
- **[Draft #2898](https://github.com/hat47x/tei-platform/pull/2898) / #988**：Semantic Attribute上の重複Role Syntax拒否、Role値Token Anchor。HEAD `e77b705e`では`semanticAttributeOptionAnchors`に`return anchors`が2箇所ありGoの2戻り値関数としてコンパイル不能であることをCI job logで確認した。コミット[`1b49dcc4`](https://github.com/hat47x/tei-platform/commit/1b49dcc44771a4fc457113af27086c14ee972443)で両方`return anchors, nil`へ修正。最新HEADのCIは別に判定。**Role値のSource-side token anchorはWorkspace Source IndexのReferenceへ自動昇格しない**。
- **[Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)**：HEAD `0bf9fac2`の健全性・脆弱性CIがともにsuccess。ここで実証したのは一部SourceをLoad→Index→Go Parseする限定例であり、Role Required Shape/全root/実Owner意味受理は証明しない。

### 16.2 Claimの正規化（新しい中央Schemaではなく既存Evidence票の列）

| 層 | そのレーンが言えること | **言えないこと** |
| --- | --- | --- |
| **Inventoried** | Workspace管理候補の列挙／見つかったPath | 見つからないRoleが全世界に存在しない |
| **Read-by-Loader** | 選択されたLoadでそのFileのbytesを実取得（#2935） | 内容をRoleとしてParseした、必要rootを全部読んだ |
| **Parsed/Anchored** | `ParseSemanticFileAnchored`による局所Role候補と元byte span（#2898） | Plugin/Operator/Required Shapeが有効、OwnerがMeaning受理 |
| **Qualified for claim** | 外部Required Root／実Consumerが必要とするRole/Dependencyと現在Source/Selectionの合致 | Runner/DB/Host Effectを行った |
| **Disclosable now** | 現在のPublic SliceとPolicyで許された局所出力、必要なら非公開に保留（#2972） | Hidden Role/未許可情報の不在、raw Source locatorの公開、将来の再利用資格 |

**グローバル不在の主張は既定で行わない**。必要なSource集合の閉包が未証明、`readByLoader=false`、解析不能、未選択Plugin、Import未束縛、Scope不明、Private Worldを使ったため公開できない、のいずれも表示では`not-observed / withheld / unresolved`を理由付きで区別する。**秘密Roleが存在するかを示す「denied vs nonexistent」差、private count、Error文字列、Source位置・依存辺、図の形・件数も公開資格の対象**。拒否理由を表示する権限自体がない場合、外部応答は同じ公開形へ畳み、監査側だけ詳細を保持する（実二世界I/O確認前に完全非干渉を主張しない）。

**誤った合成の反例**：`readByLoader=true` + `Role anchorあり` + `Public Slice生成成功` でも、必須Plugin Sourceが実未取得・Owner Required Shape未受理・現在Grant未照合なら、Roleは**自己記述の完了・Studioでの編集可能・実行可能**とは言えない。公開Sliceは秘密Sourceの欠落を「Roleゼロ」の根拠に使わない。

### 16.3 既存Ownerへの最小ケース（独立Expected／観測面／上限）

| ID | 差分を一つに絞った反例 | 期待する停止線とOwner |
| --- | --- | --- |
| **RV-01** | managed一覧にRoleを含むSourceがあるが`readByLoader=false` | `Role absent`禁止。Source未読込をInventoryの空Roleと区別。#2823/#2620 |
| **RV-02** | `readByLoader=true`だが非Semantic asset／明示Plugin未選択、Role候補が空 | Role parse/Required Shape充足を名乗らない。#988/#2823/#2620 |
| **RV-03** | Role Sourceは一意・syntax PASS、required Root外に参照またはImportがある | Source閉包・Meaning受理は`unqualified`。自己申告`required=[]`では閉じない。#2620/#2514 |
| **RV-04** | W0/W1の許可公開Sliceは同一、秘密Role数とSource pathは異なる | **許可された投影**の内容・Diagnostic・形が同じ（#2972の限定範囲）。実Policyでの切り出しとタイミングは別未実証。#2516/#1124 |
| **RV-05** | R0のRole anchorを見た後、Raw Source改版／Role Plugin選択変更／Grant撤回 | 古いNavigation/Review/Consumer資格を復権させない。Current Sourceと現在Policyは別に照合。#2621/#2505/#2600 |

**採用する最小順**：まず#2898のexact HEADでSyntax/Workspace testとGo build成功を確認→#2935の`readByLoader`を独立Snapshotに束縛してRO-04/RV-01の対照へ使用→#2620 Required/Declared/Consumedの独立閉包を照合→#2972の事前Qualified Public SliceをConsumer contractの**入力前提**として利用し、#2516/#1124で実Policy/Hostに対して独立二世界を確認。既存#2514 Oracleと#1163 Attempt/Receiptを使い、新Role/Source Registry、中央Disclosure Engine、書込み権限、Hard Gateを追加しない。

**今回の証拠上限**：前記は実PRのSource/CIを照合した横断Fit/Gapであり、Role Required Closure、製品二世界Policy、Public Slice生成者の安全性、Studio/SHO実編集・保存、Role実Target、Owner Acceptance、Provider Effectは**NOT PROVEN**。#1119の既存P0/GateとGo単一バイナリ方針を変更しない。

---

<!-- tei-role-cross-lane-provider-consumption-20261108 -->
## 17. 2026-11-08 横断統合：自己記述をAuthoring Providerの実消費・独立Requiredへ接合（設計・NOT RUN）

**結論**：SD-1 記述可能、SD-2 契約記述可能、SD-3 選択された実Interpreterが消費、SD-4 編集・再解釈可能を別Claimとする。SD-5 TEI全体self-buildは #2831 の専管。標準データ構造で記述したRoleやTraitが、実Interpreter/Providerによって評価されたこと、実装義務が履行されたこと、現在の開示・実行資格があることは別。新Core Meta Schema/Interpreter、第二Resolver/Registry、Writer/Runner、TEI Host、中央Evidence DBを設けない。#1119既存Gateも変更しない。

### 17.1 実装観測から修正した最初のFixture

- **元Subjectは公開Referenceの semantic-data:sales.order**。mainの reference/workspace/semantic/sales/order.tei にある Order.total は std.money 型で、元SourceにはRole Optionが無い。従って現在の実WorkspaceにRoleが適用済み、正式Role Providerが存在する、という前提で試験を組まない。
- Draft #2860 の既存Go Testはworkspace.Load、Load時Source Index、独立Read byte比較、既存Go Syntax Parser、Role候補の局所解析を検証する。しかしそこで使うLanguage Selection identityとFingerprintは test-only、Required Shapeの正式解釈・実Provider実消費・Owner受理は未実証。元SourceのSHA/Versionと試験用派生Sourceを別々に束縛する。
- 隔離した派生Fixtureだけに、Order.totalのRole単一Optionを追加する。Role文字列は試験値であり登録済みContractとは認めない。元Fixture・派生FixtureのSubject IDが一致しても、Source Revision/意味の受理/Build資格が一致するとは推測しない。
- mainの既存 Design Context Probe（#3284/#3296）は実CLI read-only 9応答・Source/Index等を比較し、partial / exploration-only / qualifiedTestGeneration=false として正しく資格を保留している。独立Required Function/Business/Rule/Role/Operator、外部Harness完全受領、C1/NFR、Owner意味受理は未資格。既存試験Consumerを第二正本や正式Product Context Queryへ昇格させない。
- main統合 #3300 の実Source監査では、Authoring Inspector CLI/Studioの直接呼出とComposition Provider経路がまだ結線されていない。現存の service_carrier は application-runtime 専用でありAuthoring Role Interpreterとして使用したとはいえない。Role自己記述とPlugin-first D1は同じResolver/Contract原則を用いるが、実Role Providerの有無は独立に診断する。
- Source AcceptanceのLoad時Source IndexとManifest digestは別欄であり、Host current witnessがManifestのRequired Root/Import意図まで受理した証明ではない。Root変更でSource Index/Worldが同じ場合も、独立Required閉包の再検査を省略しない（#988/#3028/#3262）。

### 17.2 新しい型ではなく、既存標準構造上の五つの関係として扱う

| 関係 | 表すこと | 禁止する誤昇格 |
| --- | --- | --- |
| Descriptor | Role/Operator/Trait/PluginのID、契約版、Required Shape、依存、診断、宣言Effect | trusted/pureの自己申告から認可・無副作用を証明 |
| Use-site Binding | Source occurrenceとSubject、明示Role/Operator、実Provider Pin、Target/Scope | 同名Role・Carrier・実装ラベルの一致から選択成功を推論 |
| Independent Obligation | 用途別のRequired Source/Dependency/Scenario/Expected、外部に固定された根拠 | 候補のrequired空配列で閉包PASS、AI生成CandidateからExpectedを自己追認 |
| Fulfillment | Traitが要求したgenerated/provider/manual-required/externalの実装、actual Build/Load/Invoke、Conformance | Trait名・ヒント・署名・Result一致から実実装履行を推論 |
| Qualification | Source Read、Parsed、Meaning Accepted、Build、現在Host/Consumerの用途相対Claim | Syntax/検査/過去Receiptから現在権限や公開資格を自動付与 |

Roleは意味適用、Atomic Traitは実装義務、Data TopologyはIdentity/Cardinality/Graph、Interpretation Topologyはeager/deferred依存、Representation TopologyはFunction内の表現型InstanceとData Contextとの明示Mapping。これらは一つのTreeへ圧縮しない。将来の複数Role適用は対象Subjectに対する別Use-site Application Relation候補として比較する。現行Attributeの単一Role Optionや重複Optionを複数Role正式構文として使わず、同属性の二重宣言はDraft #2898のSyntax拒否を優先する。

標準表・Tree・Graph・文章・SHOと、SUI的な同一Itemの複数Canvas所属は、対象意味IDを共有する異なるProjection/配置Relationとして扱う。表示上の省略、Graph→Treeの非可逆変換、Itemのラフな草稿化から元のSource/Relationを復元したと主張しない（#2497）。これはP1以降の圧力試験とし、自己記述P0のHard Dependencyにしない。

### 17.3 最短の実行パケット：同一Subject・独立Required・一因子差

1. **P0-A 元Source固定**：実main/CLI binary/argv、Workspace Manifest raw digest、Load Attempt、Source Index Artifact、元Source bytes、Subject/Scope、CLI Reply bytes/end、独立Required集合を記録。Managed/ReadByLoader/Parsed/Anchored/Selected/ActuallyConsumed/Disclosableを個別列とし、未読込を存在しないと呼ばない。
2. **P0-B Role構文と出典**：元SourceにはRole無し、派生SourceにRoleを一つ付与。BOM/CRLF/日本語コメント/引用中role/二重Optionの正常・負例を#2898既存Go Testと照合。read-only Source AnchorをWritable Patch/受理済Meaningに昇格しない。Draft/CIはexact HEADで確認。
3. **P0-C 解釈入口のFit/Gap**：#988 Language/Operatorのexact Pinと、#951/#1016のContract/Provider実選択を確認。Authoring Scopeの正式Role Providerが存在しなければ unsupported と明記し、合成Selectionやapplication-runtime Carrierで「実Interpreter成功」を装わない。Provider A選択・Provider B実呼出は値が同じでもFAIL。
4. **P0-D 独立閉包照合**：#3262の既存read-only CLI Consumerを利用し、Required/Declared/Loaded/Parsed/Selected/Consumed/Observedの範囲付き比較を実施。現Productに実Role Interpreterが無ければGap確定まで。Fixtureテスト成功、Catalog非空、validate OK、Build artifact存在を意味全件PASSへ合算しない。
5. **P1-E 改版と編集**：Role Option一箇所・Plugin同版実bytes・Source A→B→A・Manual Trait未実装を一因子ずつ変える。Raw Draft保存/Preview/一意Patch/Independent Re-read（#2621/#2497）、Owner再受理要否、Build再計算・Evidence再資格化、現Host開示を分離。正式 *.tei.md Reader/段階移行は#3030のOwnerであり、同形式のDraft保存成功はTEIでの意味Source登録ではない。

### 17.4 追加判別ケース ESD-11〜19（提案、全件NOT RUN）

| ID | 単独変更・反証対象 | 独立Expected・停止線 |
| --- | --- | --- |
| ESD-11 | D1でProvider Aを選択、実CLI/StudioはBを呼出し、出力は同じ | Result一致でも不合格。Contract/Scope/Epoch/実Call identityを比較（#1251 AS-02） |
| ESD-12 | Authoring Role用途へapplication-runtime専用Carrierを流用 | 必要な契約/Scope無しなら unsupported。共通Go関数＝同一Provider実利用ではない |
| ESD-13 | Manifest Rootだけ変更、Source IndexとWorld byteは同じ | 独立Required Root充足はunknown。Manifest変更を無関係と判定しない |
| ESD-14 | 管理Rootの必須Sourceに *.tei.md を追加、正式Language Reader未選択 | Managed/ReadとParsed/Meaningを分離。旧*.tei/未管理Markdownは対照として保持 |
| ESD-15 | 同Attribute role二重指定・引用中role・BOM/CRLF | 重複は位置付き拒否、引用は非宣言、単独値Anchorは実byte位置に一致 |
| ESD-16 | Trait manual-required 未実装、hintのみ存在 | unfulfilled。Hint、Role候補、実装済みを混同しない |
| ESD-17 | Node→Ref<Node>のdeferred循環、A↔Bのeager解釈循環 | 前者保持可能、後者拒否。固定点解釈を暗黙で導入しない |
| ESD-18 | Secret Requiredだけ異なる二世界、公開入力は同じ | 公開Response/診断/件数/図形から秘密の有無を漏らさず、監査側では未充足を残す |
| ESD-19 | Role解釈・Artifact bytesは一致するが、stage2実Processがstage1を未使用 | 自己ビルドL3は不合格（#2831）。ビルド再実行と再入の別Claim |

C0/C1は各Obligation/実到達分岐を分母にする。Candidateを評価する同じRole/AIからExpectedを作らず、#2514/#1180の独立ExpectedとMutantを使う。外部Harness Reply切断、Runtime/Host Grant未観測の場合は qualified としない。

### 17.5 Ownerと順序：第二の縦断Gateを作らない

| 順 | Owner | 成果・Exit |
| --- | --- | --- |
| P0-A | #2620/#2823/#3262 | 公開sales.orderのSource/Manifest/Load/Indexと独立Requiredの範囲付きGap |
| P0-B | #988/#2620、Draft #2898/#3028 | Role Option、Source Anchor、K0 Seed/Language/Operator exact Pinの局所反証 |
| P0-C | #1251/#951/#1016/#1158、main #3300 | Authoring Provider選択と実call-site（Inspector D1）をまず確認。Role処理の有無は別 |
| P1-A | #2620/#3278/#2514 | 最小Role Interpreter実消費と一つのAtomic Trait履行を独立Expectedで検証 |
| P1-B | #2621/#2497/#1124、#3030 | Markdown草稿保存・Preview・局所逆Patch・再Read・再受理要否を別結果 |
| P2 | #2516/#1124/#2831 | 同じMeaningをStudio/SHO/実Targetに投影し、Consumer完全受領とBuild L1/L2/L3を別実証 |

**今回のEvidence**：他チャットの成熟設計と既存GitHub main/公開Draft Sourceの静的照合、この計画追補のみ。新Go/Node Test、実Role Provider、実CLI/Studio共通呼出、Authoring Service Carrier実現、正式Markdown Reader、Owner Meaning Acceptance、External Host Effect、全体Self-buildは NOT RUN/NOT PROVEN。必要情報が欠けた場合に unsupported/unknown/partial と答えられることを、無理に実装成功と扱うより先に受け入れる。

---

<!-- tei-role-application-occurrence-obligations-20261108 -->
## 18. 2026-11-08：Raw Item・複数Role・表現Occurrence・実行Variantを自己記述の独立関係として検証する

**状態：設計・受入案。ESD-20〜29はすべてNOT RUN。** §17のESD-11〜19を置換せず、他チャットのSUI Item/Canvas、標準表現型、TEI三視点モデル、テストT1、CLI EffectとDriftの実観測を受けて追加する。これは新しいCore Itemクラス、固定Role enum、Canvas Engine、Source Writer、Test SSOT、認可Host、汎用Meta Interpreterの採用ではない。

### 18.1 Everything is Dataは「未整理でも保存できる」が「自動的に意味受理・実行できる」ではない

未整理の自由文章・未読ファイル・曖昧な断片を、Raw Source/Item相当の標準構造として保持できる方針を検討する。分類不可・Plugin未選択・情報不足を保存失敗の理由にせず、保存・読取り・Role候補化・正式な意味受理を別の状態にする。Raw ItemをProductの新しい永続Core型として先に決定しない（SUI側のItemモデルは他領域に有用な圧力事例でありTEI Kernelへそのまま移植しない）。

- **Task / Action / Eventは固定の排他Item型ではなく、明示的に適用する意味Roleの候補**。Task＝予定/実施対象、Action＝操作意図と入力/対象、Event＝時点や根拠を持つ出来事の記録、という相違を独立Contractとして検証する。Taskがあることから実行可能Actionを自動生成しない。Event観測と未確定Effect/Receiptも混同しない。
- **Role DefinitionとRole Applicationを分離**。前者は適用規則/Required Shape/Contract、後者はsubject/use-site/scope/Role version/Source occurrenceへのBinding。Role候補ラベルが存在してもProvider実装・Owner受理・Operation Offer・Host Grantを付与しない。
- **Atomic TraitはRoleと別の履行義務**。Role適用の後で必要なTraitが明示された場合のみ既存#3278/#951でFulfillmentを検証する。Hintや読みやすい自然言語説明から暗黙にrequired履行へ昇格しない。
- **専門家の認知順を守る**。日常のDSLにGoのstruct/func/importやPlugin内部Fingerprintを露出する必要はない。Markdown表/短い式はAuthoring Projectionであり、Source locator・独立Meaning/Provider Pinは既存検査層が保持する。Go AST/コード生成の都合を主構文決定基準にしない。

### 18.2 四つのIdentityと三つのTopologyを横断する（新しい正式Schemaではなく検収票の列）

| 関係 | 必須の区別 | 同一性の誤認を拒否する例 |
| --- | --- | --- |
| Subject Identity | 同じItem/Entity/Model subjectを指す安定Identity | 文字列が同じ別SubjectをAIが黙って統合 |
| Source Occurrence | 原文Document Revision・byte位置・Container/Anchor | CRLF/BOMや文書移動後に旧byte spanをCurrentと誤認 |
| Role Application | 同一Subjectへの用途別Role/Contract/Scope/適用Source | 単一属性への重複role Optionを正式な複数Role Bindingとして後勝ち採用 |
| View Occurrence | Canvas/Tree/Diagram/SHOに置いた表示のID、配置・表示範囲・Disclosure | 一つのItemを二Canvasへ置くとSubject/Actionが二重生成される |

Data Topology（Entity/Relation/Graphのidentity/cardinality/時点）、Interpretation Topology（Role/Operator/Pluginのeager/deferred依存）、Representation Topology（Function所有の表現InstanceとData Contextの明示Mapping）は独立。Graph→Tree、同Subjectを二Canvasへ配置、同CardのTitleだけ表示、SHO Markdownで省略する場合は、落ちた辺・隠れたSource・未取得部分を逆投影で補間しない（#2497、Draft #3303/#3263）。一方、純粋なレイアウト変更を意味・Rule・Target再生成へ機械的に直結させない。Sourceを編集するには元Occurrenceの一意性・Current Revision・対象Binding・逆写像可能性を別に満たす。

### 18.3 Self-descriptionの等価性は一種類ではない

| 比較 | 主張できる同一性 | 主張できない同一性 |
| --- | --- | --- |
| Source raw bytes一致 | 固定Encoding/line endingの原文一致 | PluginやOwner Meaningの同一性 |
| Parsed/typed構造一致 | 既存選択済LanguageとType契約の下での候補一致 | 正式受理済みMeaning/Use-siteの一致 |
| Accepted Meaning一致 | 指定Owner/Scope/Revisionの意味受理 | Source occurrence/実Target/現在Grantの一致 |
| 実Provider結果一致 | 当該入力とAttemptのResult一致 | 選択ID/実呼出Provider/Effect履行の一致 |
| 公開Projection一致 | 当該公開Consumerへ見える同一出力 | 見えないSource/Relation/全Requiredの不在 |

必要に応じて内容ハッシュとSource Reference位置を別追跡する。A→B→Aでも過去Reviewの現在資格を自動復活させない。§17/Role Descriptor・Traitの改版条件を再利用し、中央Global Revision Counterを増やさない。

### 18.4 直近main/PRから取り込む三つの実装上の教訓

1. **Read Modeは契約の一部**：#3262の実CLI9応答を使うConsumerで、source-index全体モードとsubjectモードなど異なるOutput shapeを取り違えると、Required Ruleが実在しても誤ってMissing判定する。Draft #3324（1 Rule/subject照合）とDraft #3326（3 Rule/full-index照合）は異なる提案で、双方の個別CIが成功しても一体Context Readerの実現・同時merge適格ではない。Role/Descriptorの『無い』もMode、Scope、Pages/End、Read-by-Loader、Parsed/Consumed/Disclosureの一致で判定する。
2. **CLIのEffectはCommand名ではなくInvocation Variantに依存**：#3182/#3302のview -executeは、read-only Discoveryに見えるCommandのOptionでローカル書込みを起こせる圧力例。#3302はclosed/unmerged。自己記述されたCapabilityのread-only表明を、すべてのFlag/操作へ普遍化しない。実オプション・選択Offer/Provider・Actual Effectを別に評価し、Toolを生成AIが自動実行してよいかはHost現在資格で判定する。
3. **Operationが『current』なのは入力比較が成立した限定範囲**：main統合 #3325 のread-only Drift ReferenceはWorld/ModelRev一致=current、差異=stale、必要値を指定しない場合=uncheckedを区別する。basis文字列一致だけで属性変更を検出しない。過去のRole/Operation Previewを後続のBuild/Dispatch/Saveの現在資格と扱わない。

加えてmain統合 #3304は、既知の行番号0 Ruleを既存Pinned実CLIで判別するT1-Aの有限実証。未知のRuleをAIが発見するT1-B、Roleの必要閉包、C1/NFR、Owner受理とは別。Source→Role→Expectedを同じ生成AIの自己申告だけで正当化せず、#2514の独立Expected・Mutantを検証基準にする。#2497で確認したSHO消費のWindows CRLF変化も、同じ表示と原Source byte/Anchorが異なる実反例として使用するが、TEIの正式Markdown Writer/Readerが成立したと主張しない。

### 18.5 追加の独立判別 ESD-20〜29（すべて提案・NOT RUN）

| Case | 一軸だけ変える | Expected / Owner |
| --- | --- | --- |
| ESD-20 | 未分類Raw Itemを読み込み、明示Role/Contract無し | Source保存・読取りは可、Role Candidate/正式Meaning/実Actionは自動生成しない。#2620/#2621 |
| ESD-21 | 一SubjectへTaskとEventの役割候補を追加 | RoleごとのRequired Shape/時点/根拠を区別。Task＝実Action実行資格、Event＝Receiptと推論しない。#2620/#2507 |
| ESD-22 | 同じSubjectを二Canvasへ出し、一方の配置だけ変更 | Subject/Meaning/Task/Actionは1つ、View Occurrenceは2つ。レイアウト差でMeaning再受理を要求しない。#2497/三視点Owner |
| ESD-23 | 同文面の別Subjectを並べ、同一Subjectの二Source occurrenceとも比較 | 文字列一致でSubjectを統合しない。複数OccurrenceはIdentityだけでなく出典/改版をそれぞれ保持。#2620/#2497 |
| ESD-24 | 多重辺Graph→単純Tree/Markdownへ非単射投影、その表示から辺を編集 | Lossがある限り元の欠落辺を削除・作出しない。編集逆写像を拒否/要Review。#2497/#1124 |
| ESD-25 | Source Indexのsubject/index Reader Modeを取り違える | 真に存在するRequired RuleのFalse Missingを拒否。Mode別shape、Scope、Revision、Reader実Consumedを記録。#3262/#2514 |
| ESD-26 | read-only viewのFlagへ-executeを付与する | read-only資格から外れ、書込み可能Variant/Host Grantは再評価。Command名やEffect自己申告だけで許可しない。#988/#951/#1174 |
| ESD-27 | OperationをWorld/ModelRev指定なしで照会、続いて属性だけ変更 | uncheckedとcurrent/staleを区別。basis同一でも現在利用可と誤判定しない。#2507/#3325 |
| ESD-28 | 同じMarkdown見た目のSourceをLF→CRLF、BOM変更で保存 | raw byte/Source spanが変わる。適切な再Parseなしに旧AnchorをCurrent扱いしない。同一Meaningは独立判定。#2621/#2497 |
| ESD-29 | AIが既知不正Rule Caseを再利用する／独立Requiredの未知Ruleだけ欠落させる | 既知T1-A成功からT1-B/C1完全へ昇格せず、必要Rule欠落を検出可能なSource Readerが無い場合はpartial/unknown。#3262/#2514 |

**検証基準の作成・管理**：この表は新しいCase/Scenario SSOTではない。#1180 Driver Scenarioに適用対象と独立Expectedを固定し、#2514に正常対照・mutant・Claim ceilingを記録する。Role Descriptor自身から生んだExpectedは独立基準とは呼ばない。#3304やDraft #3324/#3326、#3325、#2497の別Runを一つの統合PASSに足さない。

### 18.6 実行に向けた最短順序と採用条件

- **P0（現在の不足を狭く確認）**：§17のsales.order元Source/派生Role Sourceを使い、ESD-25（source-index Mode/必要Rule）は既存実CLI、ESD-27（Current/Unchecked）はmain統合のread-only Drift Reference（実Product CLIの既定契約とは別）、ESD-20（無Role Sourceの正例）は公開Raw Sourceの独立Readでそれぞれ先に調べる。現実のRole Interpreterが無ければunknown/unsupportedを正しく返すところまでを受入とし、新Meta Parserや万能Item Handlerを作らない。
- **P1（Role合成の成立を反証）**：ESD-21/23および既存ESD-15/16のRole Application/Atomic Traitを同一Subject/Scopeで検証。現Attribute.Roleの単一値から複数Roleの自動選択を推論せず、明示Role Application Relationが必要かを既存Semantic/Graph/Contract上で比較してから採否を判断。
- **P1（複数表現・編集）**：ESD-22/24/28は#2497/#2621/#1124と三視点Ownerに委ねる。Functionに配置されたRepresentation Instance/Bindingと、同Itemの複数View Occurrenceを区別。UI上の同一表示・図の一致から意味・出典の完全性を主張しない。
- **P2（実操作とテスト生成）**：ESD-26/29を#951/#1174/#3262/#2514の既存実Provider/Scenarioへ接続。CLI/MCP/Studioは同じread-only Design Context/Contractを投影するが、現時点で未実装のToolをあると広告しない。外部Harnessのpage/end/Reply digestの完全受領を別に観測。

**今回のClaim ceiling**：他チャットの設計統合・既存GitHub Issue/PR/Codeの静的照合に基づく計画差分。ESD-20〜29、正式Role多重適用Syntax/Interpreter、Raw ItemのTEI製品実装、SUI/TEI共通UI、MCP/Studio、Role/Builder E2E、実Host Effect、全体G4 Self-buildは未実施・未受理。Role解釈のP0不足は#2620に残し、SUIや表現Projection/三視点の独立Ownerを上書きしない。

---

<!-- tei-role-raw-authoring-and-boundary-consumed-20261108 -->
## 19. 2026-11-08 実Source読取の最初の結節：Raw Document Inventory、11 CLI Reply、用途相対Claim（最小コード試行）

**状態（この節の変更時点）**：実main Source/API・並行PRを再照合し、既存Draft #2860のGo ReferenceにRaw Markdownの限定負例を1件追加した。GitHubへコード保存した事実と、CI/Go実行結果は区別する。ここでは新Product Markdown ReaderやRole Interpreterを実装したと主張しない。

### 19.1 第18節の計画に対する、実mainからの重要な状態更新

- **Source Index Readerモード問題の限定解消**：[merged #3354](https://github.com/hat47x/tei-platform/pull/3354) は重複Draft #3324/#3326をともにclosed/unmergedとし、mainの既存 tools/reference-issue-test-ability/context_gate.py **一Consumer**へ統合した。実read-only CLIは従来9応答に全indexモードのsource-indexとValidation Inspectを加え**11応答**。固定独立3 Rule、1 Focus Rule Canonical、Related Relation、Source ArtifactとInspect↔Indexを照合。exact-head [専用CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962)で**32個の限定合成Control**と既存6 Scenario Step＋既知T1-Aの局所試験が成功。mainへmerge済みだが、main push側CI/実外部H3は別判定。これらは3 Rule scopeのbounded presenceであり、全Function/Business/Operator/Role/Manifest required closure、同時刻Snapshot、未知AI T1-B、Owner受理ではない。引き続き partial / exploration-only / qualifiedTestGeneration=false。
- **Raw Markdownの具体的断絶**：[Issue #3346](https://github.com/hat47x/tei-platform/issues/3346) の実観測では docs/plan rootをmanifestで管理しても、通常*.mdは managedFilesに列挙されるだけで readByLoader=false、Semantic catalog/inspect/traceには到達しない。現 internal/workspace/loader.go の ReadByLoader契約と、internal/workspace/source_index.go の managed Artifact収集を別々に確認した。**原文byteがSource Indexに記録されても、意味解釈／Role Required Shapeの解決ではない**。一般Markdown Raw、TEI Authoring *.tei.md（Draft #3031未選択Reader拒否）、旧*.tei、SQL/外部形式の受理経路を混ぜない。
- **比較するFingerprintの意味範囲**：[Issue #3345](https://github.com/hat47x/tei-platform/issues/3345)では viewのApplication利用Namespaceだけの semanticFingerprint と、CatalogのWorld Fingerprint/Source Index Fingerprintは同じ対象ではない。Raw Sourceのbyte hash、Loader Load-time Index、Source Root Manifest、Namespace-relative Semantic、World、Role Providerのexact implementation、Function Build Inputsを『どれもFingerprint』として同一比較しない。
- **実CLI Replyの形とEffect**：[Issue #3347](https://github.com/hat47x/tei-platform/issues/3347)のdiagnostics wrapper、requirementsの裸配列、Catalog中std builtin混在は『見つからない』の偽診断の原因になり得る。既存受信側はCommand別の実Schema/Scopeを尊重する。[merged #3333](https://github.com/hat47x/tei-platform/pull/3333)はviewのローカル書込みFlagをwriteFlagsとして機械可読化して実File Writeを観測したが、全Variantのread-onlyやHost認可まで保証していない。Tool Producerのread-onlyとSession/Log/Egress Sinkの無書込み・無開示は別Owner（#2515）。
- **セルフビルドの物理再入Reference**：[PR #3360](https://github.com/hat47x/tei-platform/pull/3360)は別Go driverのstage1 Artifactを実Invokerとしてstage2を実行した有限実測を報告。ただし対象は cmd/tei でなく外部Seedの自己再構築可能driverで、PR exact-head CIも区別する。Role自己記述のSD-3/4をL3 Self-build合格へ加算しない（#2831）。

### 19.2 Rawデータが自己記述へ昇格する際の二レーン

| レーン | 入場条件 | 主張できる上限 | 最初の欠落Edge |
| --- | --- | --- | --- |
| **D：Raw Authoring/Knowledge** | File/Markdown/OKF Itemのraw bytesとSource Path/Revisionが読める | 履歴付きRaw保持、Reference、用途次第で全文閲覧。ただしReader未選択なら意味候補なし | 管理外文書を含むReader/受信Scope、情報開示、原文の完全配送。Issue #3346/#2621 |
| **S：Typed Semantic Authoring** | Workspaceが選択したGrammarと実ParserでParse、Attribute/Role Anchorを確定 | Parsed/Anchored/Type Candidate、独立Requiredがあれば範囲付きMeaning候補 | 正式 *.tei.md Reader、Role Provider exact Selection/Actual Invoke、Owner Accepted Shape。#988/#2620/#3031 |

**昇格は明示的なイベント**：D→Sの正規化・分類候補をAIが提案しても、Source/Version/Selector/Scope/Required Shape/独立Expected/Owner Reviewを欠いてMeaningにはならない。Dに置いた『Task』『Event』『role trusted_amount』の単語を自動的にTask実行、Event Receipt、Role宣言、Trait履行へ昇格させない。D/Sをまたぐ同Subject/複数CanvasのView OccurrenceやSHO抽出は別Projectionであり、逆Write/Source lossを独立に検証。

既存のread-only CLIで 'docs managed but uninterpreted' と表示されることは望ましい正例であって、未対応を不可視化することよりも安全。だが今後Dの中身を読む対象にもする場合は、単に Source Index のArtifactだけを公開するのではなく、現在のConsumer GrantとSource/Document用途に応じた出力範囲を検証する。秘密の有無・件数・Source locatorをUnauthorized Consumerへ漏らさない。

### 19.3 今回のGo Reference更新と独立受入の上限

**追加した限定Product API test**：[Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)の reference/dsl-language-plugin/self_description_workspace_boundary_test.go に TestSelfDescriptionManagedRawDocumentDoesNotBecomeSemanticRole を追加（[commit abf901d0](https://github.com/hat47x/tei-platform/commit/abf901d02a77bc9f84459165c753e903a71e0252)）。

検査対象は実 workspace.Load と workspace.BuildSourceIndexFromLoad。独立に固定した二Source（semantic/probe.tei のProbe Item、docs/design.md のRole様自然文）に対し、(a)双方がManaged Inventoryへ入る、(b)SemanticはReadByLoader=trueだがRaw docはfalse、(c)Raw docのLoad-time Source Index Artifact fingerprintは実raw bytesを保存、(d)Raw docはSemantic Definition Referenceを持たない、(e)Semantic SubjectのSource Index参照は生きる、(f)後からdoc bytesを書換えても旧Load IndexのAsset fingerprintは変化しない、を判別する。

これはRaw資料を**現在の既存Loaderが誤って意味へ昇格させていないか**を検査する Product APIへの局所テストである。Raw bytesを実際にReaderが意味解釈した、Role PluginがInvokeされた、非公開Documentが守られた、正式 *tei.md Readerが成立したことの証拠ではない。Scopeはtemp Workspace内、外部Host/Studioなし。新テストのGo/CIは**保存時NOT RUN**であり、CIが完了してからpass/failを記載する。

### 19.4 次のP0は「必要RoleとDocumentが見えない理由」の判別を具体化する

| 順序 | 単位／Owner | 受入・停止条件 |
| --- | --- | --- |
| **P0-D0** | 今回Go限定Test + #3346/#2823 | Rawのmanaged/unread/byte-indexedを区別。Go Testのexact-head CIを取得し、失敗時はAPI実挙動を優先して設計を修正 |
| **P0-S0** | main #3354の単一Context Consumer + #3262 | 11 replyの**対象Modeと比較可能なFP Scope**、独立Required 3 Ruleの認定範囲を固定。Partialから full へ勝手に上げない |
| **P0-S1** | #988/#2620/Role Provider／既存 #951 | 公開sales.order元（Role無）と明示Role派生を区別。Provider実選択/実呼出が無ければunsupportedで停止 |
| **P1-D/S** | #2621/#2497/#3346、三視点 #1124 | Raw保管→選択済Parser→Source Anchor→Preview→一意Patch→再Readを順に接続。形式/Roleの別名一致で書戻し不可 |
| **P1-F** | #3262/#2514/#1251/#3278 | 一Functionの必要Data/Relation/Rule/Role/Trait/実消費を独立Requiredで狭く検査。未知必要物があればAIテスト生成資格を保留 |
| **P2** | #2509/#2515/#1174/#2831 | 実H2/H3 Replyのend digest・Audience/Sink・Runtime Effects・生成TEI自身の再入を各Ownerの別Runで資格化 |

**ESD-20のProduct API側局所対照を一つ増やした**。ESD-25は#3354で単一Context Consumerへ既に採用されたため、別Context Reader/独立Requiredテーブルを自己記述側に作らない。追加Case番号/第二Test SSOT/新Core型を増やさず、§17/18のESDに実Evidenceの上限を付ける。#1119既存Gateを変更しない。

---

<!-- tei-role-independent-closure-and-real-invoke-20261108 -->
## 20. 2026-11-08：自己記述Roleの最初の成立条件を「必要閉包×選択階層×実消費」に限定する（横断P0接続）

**状態：計画への採用候補・新ESD-30〜37はNOT RUN。** §19の実Go Raw文書試験と main統合 #3354（Design Context）に、続いてmainへ統合された #3337/#3367/#3368（Capability Selection、D1受入、Atomic Trait独立Required）を接合する。正式Role Interpreter、Self-describing Grammar/ABI、新Production Tool、Host認可、Agent用万能Source Readerを実装したという主張ではない。

### 20.1 自己記述の根拠は自己申告で閉じない

Role定義をTEIの標準データで記述できても、そのRole自身が「Required Sourceは全部ある」「自分の解釈器は正しい」「Traitを満たす」「実行してよい」と宣言しても採用根拠にならない。Bootstrapは既存 #988 のK0 Seed→K1 Workspace/Manifest Source→K2 exact Language Selection→K3 Parse Candidate→K4 Operator/Role Binding→K5独立Required/Owner意味受理→K6物理実消費を維持。実Reader/Interpreter未対応の場合は unsupported と表示し、データとしてのRaw保持を継続する。

**最小の受入票（新Core Schemaではない）**：一つのSubject/Function/用途につき、外部固定の RequiredSet、Manifest/Source Inventory、Loader ReadSet、ParseSet/Anchor、Selected Language/Role/Operator、Resolved exact Provider、Actual Call/ConsumedSet、Owner Accepted Meaning、現在Consumer/Host資格を別欄にする。

| 判定 | 肯定に必要な根拠 | 停止する誤認 |
| --- | --- | --- |
| Sourceに存在 | 指定Scope/Source版の実Inventory/Source byte | 世界全体のRole不在を、限定Workspaceの未発見から推論 |
| Required閉包 | **用途別に独立固定した集合** R と、実Consumer観測済みの対応付き集合 C の差。RのUniverse/Scopeも明示 | 候補定義内 required=[] や Plugin提供一覧を必要件数の分母に使う |
| Interpreter選択 | Language/Role Contract + Capability Kind/ID + Scope + exact Provider Pin | Bundle/PluginがselectedだからRole Providerもselectedだと推論 |
| 実消費 | 対象Build/RuntimeにProviderが含まれ、**選択した実装が実call-siteで起動された**証拠 | A/Bが同じ値を返したことをSelection一致・Executionとする |
| Role Meaning | 独立Expectedで必要Shape/Null/関係/単位/時点を評価しOwner受理範囲を固定 | Parser candidateやRole自身の宣言をAccepted Meaningに昇格 |
| 現在利用資格 | 受信Audience・目的/Scope・Rev・現在Host/Disclosure、Replyの完全受領 | 旧Receipt/Source Index hash/Role純粋性をGrantに流用 |

必要条件の集合差 R − C が空でも、RのUniverse自体が閉じていない、実Readがpartial、依存元Pluginがunknown、Reader modeが異なる、秘匿で開示不能なら full と言わない。逆に**独立R外の任意のCandidateが未選択であること**は正常な場合があるため不合格へ過剰昇格しない。これは #3354 の3 Rule/32局所Controlの正負対照から、自己記述に取り込む最小原則である。

### 20.2 Pluginの『ある／選ばれた／呼ばれた』を4段階から5段階へ分ける

main統合 [#3337](https://github.com/hat47x/tei-platform/pull/3337) の実read-only Capability Coverageは、Plugin Package選択と個別 Capability Kind+ID 選択を別観測にした。DB技術名とCapability IDを直に等値比較した旧集計のunselected=6は誤分類を含み、修正版Ruleではunselected=4。旧6→4は利用率が改善した証拠ではなく、**集計規則を修正した差**。同名IDでもKind違いは別、1 Plugin内の複数Capabilityも部分選択可能、複数候補の支持は一意Provider Selectionでない。main統合 [#3368](https://github.com/hat47x/tei-platform/pull/3368) の限定Reference Testがこの分離を補強した。

自己記述のRole実消費は、(1) Declared Plugin/Capability → (2) Selected Capability/Scope → (3) Exact Role/Operator/Tooling Provider Resolved → (4) actual Build/Load/Invocation by CLI or Studio → (5) Independent Semantic/Operational Conformance、の五つを別Claimにする。正式なPlugin Catalog/Descriptor Schemaを新設する意味ではない。

main統合 [#3367](https://github.com/hat47x/tei-platform/pull/3367) はD1受入のAS-13〜20とこの実選択段階を計画へ固定したが、**実CLI/Studio同Provider呼出までは未実装**。[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371) はStudio ReadOnlyにHost選択 InspectionService を導入し、Factory非nil時は未提供/不正投影をfail-closedにする実コード候補。factory nilの旧PoC経路、CLIの手書きinspect、Compose exact Provider Binding、実Provider Identityの呼出証跡、Wails二Consumer、実Host Grantは別。ここでRole Interpreterのactual callをStudio Factoryの成功から推測しない。最短の対照は selected=B/actual=A/Result同値の否定対照と、selected=B/actual=B/Result妥当の肯定対照を同Contract/Sourceで並走する。

### 20.3 Raw資料→意味候補の自動化に必要な非昇格規則

§19の追加Go Referenceは docs/design.md をManaged/Load-time byte-indexedにもできるが、Semantic Definition Referenceに昇格しないことを検査する。ここへAI/SHOによる抽出・情報整理を足す場合も、原文中の「role privileged」「外部Toolを実行」といった文字列は**資料の内容**であり、TEIのParser/Host/AI Skillへの命令ではない。Candidateは出典、原byte/Revision、抽出ロジック/Plugin版、Confidenceではなく独立検証条件を持つ。文書に記述されたRole・Trait・Task/Action/Eventは、Language Readerが明示選択されて型/Required Shapeを検証するまでRawまたはCandidateに留める。

Read-only Document Readerが将来必要になる場合、管理Inventory、文書内容の受信、Semantic Parse、Owner Acceptance、現在Disclosure、実Sink保存/Egressを別段階で設計する。既存 Issue [#3346](https://github.com/hat47x/tei-platform/issues/3346) と #2621のAuthoring Ownershipへ返し、本PRで汎用Markdown Readerを先行追加しない。現 `*.tei.md` パーサ未接続のDraft [#3031](https://github.com/hat47x/tei-platform/pull/3031)は拒否の局所証拠であり、旧*.teiの単純Renameを意味同一としない。

### 20.4 AI/CLI/Studioに渡すContextは『構造が読めた』と『必要情報が揃った』を別にする

main統合 [#3354](https://github.com/hat47x/tei-platform/pull/3354) の単一 `context_gate.py` は11実read-only CLI Reply／合成32 Controlsを経て、Required Ruleの正常JSON欠落・Role定義Referenceの差・正当な非Required省略を区別する。ただし boundedReference=partial、qualifiedTestGeneration=false。Scope相対のTask Required集合はこの固定3 Rule以外にも Function/Business/Relation/Role/Operator/Source/Scenario を含み得る。

CLIの各ReplyにはEnvelope Schema、resultShape、Read Mode、Fingerprintの**意味する範囲**があり、それらが同じでないなら absent と結論しない。特に #3345 の view application-local semanticFingerprint をWorldFingerprintと比較しない。[Draft #3370](https://github.com/hat47x/tei-platform/pull/3370) は requirements.result の**実配列**をresultShape=arrayとして広告する候補。未宣言結果をObjectと推測しない。#3347のdiagnostics wrapperやstd builtin混在を理解できない場合は malformed/unknown とRequired Missingを分離。

複数Consumerへ自己記述を返す際は、Read Sliceが許可済みであること、Outputが途中切断されていないこと、External Tool ReplyとSession Sinkに原文/秘密Pathを漏らさないことを別検証する。Row/Role/Secretが非開示でも、未許可Consumerへの『秘密Roleが存在するか』を答える必要はない。Scopeが不明ならGlobal Absentを主張せず、監査側のみ内部不足理由を保持する。

### 20.5 新しい一軸判別 ESD-30〜37（全件提案・NOT RUN）

| ID | 反例 | 独立Expected／既存Owner |
| --- | --- | --- |
| ESD-30 | Plugin Pは選択済、P内Role Capability Rは未選択 | RのSelectionは未充足。Plugin SelectedをRole Bindingに流用しない。#1251/#3337 |
| ESD-31 | 同一IDのCapabilityが別Kind、または1 Pluginの2能力から一方のみ選択 | Kind+ID/Scopeで独立判定。Capability総数をRequired Role分母に使わない。#3278/#3368 |
| ESD-32 | DB技術名とCapability IDの比較規則が変わり、候補選択数だけ差 | rule fingerprintが異なる旧6/新4を単一時点の改善としない。#1251/#3278 |
| ESD-33 | Role Provider A選択、CLIはA、StudioはHost Factory経由Bを実呼出、Result一致 | 同一Provider実利用を拒否。双方の実call-site/Build Pinで個別検証。#3371/#951 |
| ESD-34 | Factory非nilだがInspection未提供。既存built-inなら成功するSubject | 明示未結線ならfail closed。PoC fallbackの無条件使用を拒否。#3371/#1824 |
| ESD-35 | Raw MarkdownにRole/Host操作を命令形で書くが実Parser未選択 | Raw byte保持のみ。役割宣言・Tool Invocation・Meaningへ昇格させない。#3346/#2620 |
| ESD-36 | `requirements`正常成功でresult=[]、別Readにはrequired Ruleが存在 | JSON Shapeはvalid array。『Requirement情報なし』と『必要Rule不在』は独立、未知ならpartial。#3347/#3370/#3262 |
| ESD-37 | 提示FingerprintがWorldとview Namespaceなど比較Scope不一致 | version mismatched/uncheckedとRequired Absentを区別。Host/Grant・BOMへ伝播させない。#3345/#2514 |

### 20.6 最小の次Runと完了判定（旧P0/Gateを増やさない）

**Run S0（Role原文）**：前回追加した `TestSelfDescriptionManagedRawDocumentDoesNotBecomeSemanticRole` のexact PR HEAD/CIを最初に判定。既存Raw/Semantic 2 Source・Manifest digest・Loaded/Artifact/Reference・mutated Raw Revisionを保持。期待はRaw未Parsed／Semantic Parsedの対照。TestがGreenでも直接Role意味受理へ昇格しない。

**Run S1（必要集合）**：main #3354単一Consumerの実11 Reply/独立3 Rule対照を再利用し、ESD-36/37のReader Mode・Shape・Scope差をSchema別に確認。Source側 `Role` が実現されていないことを『未発見』ではなく `unsupported/no-provider-evidence` と表示できるかFit/Gapに残す。新CaseのExpectedは #2514、Scenarioは #1180。

**Run S2（Provider actual）**：#1251/#951のInspector D1選択、#3371のStudio FactoryおよびCLIの実入口から同一ContractのProvider A/Bを選択し、実Call/Resultを対照する。実Plugin Role解釈とは別、最初は既存read-only inspectだけで P0 技術接合を確認。Provider selectionの実証後に必要Role Contract一つだけを載せ、Language Kernel/Parser/Role専用Registryを新設しない。

**Run S3（意味と実行）**：S2の実選択証拠と、Role必要Shape・原子的Trait Fulfillment・独立Expectedを接合。正規化・Owner Acceptance、現在Grant、Tool全受信、Target Build（Go単一binary）を各既存OwnerのClaimで審査する。[main統合 #3360](https://github.com/hat47x/tei-platform/pull/3360) のdriver stage1→stage2再入は本物の限定物理Referenceだが**対象はcmd/teiではない**。TEI全体のG4 L3資格をここへ流用しない（#2831）。

**停止線**：製品Ready判定・全Required Closure・AI T1-B生成・実Domain Conformanceを、Reference CIのPASS数やDescription/Role名から導かない。main再Base/Go Build/CIは最新exact HEADで判断する。既存G0〜G4 Gateを保持し、D1/Raw/Role/Trait/BOM/Self-buildを全て最初のP0 Hard Dependencyにしない。

### 20.7 CI実測への補正：Go試験PASSとFormatting失敗を分ける

<!-- tei-role-ci-gofmt-fix-20261108 -->

第19節で追加したRaw Markdownテストを含む exact HEAD e14aab213f6057f17744a4881e32e4a6ed7eac33 の[Repository Health #42394860575](https://github.com/hat47x/tei-platform/actions/runs/42394860575)は**completed/failure**。実Jobログでは `go vet ./...`、`go test -count=1 -timeout 60m ./...`、`tei validate -workspace reference/workspace`がPASSし、`reference/dsl-language-plugin`パッケージの試験もPASS。一方 `gofmt -l . is empty` はFAIL、対象は追加した `reference/dsl-language-plugin/self_description_workspace_boundary_test.go`。この段階の総合結果をPASSと報告しない。

失敗理由をGo Test failure / Doc Plan failure / Formatting failureに分類したうえで、追加関数直前の不要な空行1行だけを[commit 1668d617](https://github.com/hat47x/tei-platform/commit/1668d617a7b47202feff345f742a286713972637)で除去した。既存FixtureとExpected/Assertionは変更なし。**修正後のPR HEADでCI completed/successを取得するまではRepository Health未資格**。過去のGo Test PASSは元HEAD限定、修正後の現HEADへ無条件に転用しない。正式Meaning/実Role Provider/Runtime Authorityへも昇格しない。

---

<!-- tei-role-actual-rule-and-decision-evidence-20261108 -->
## 21. 2026-11-08：実Roleを使う業務規則・Ontology判断根拠・外部REA候補の受理境界（実Go差分とO0計画）

**今回採用する範囲**：既存mainの`currency-rate-fallback`という具体的なRole→行値→業務規則の実実装を、SD-2/3への限定的な圧力事例として使用する。§17〜20のSource/Independent Required/actual Provider資格を上書きしない。[Ontology構想 #3475](https://github.com/hat47x/tei-platform/issues/3475)のO0静的Fit/Gapだけを自己記述へ接続し、実Digital Twin/Sim/SDK/SEI Hostは現在のP0実装に追加しない。REAの外部原資料→観測Evidence→TEI Candidateも別経路として保持し、役割・業務同義・現在Grantを自動受理しない。

### 21.1 mainに存在する最初の具体的な『Role宣言＋Data値でRuleを決める』Reference

[main統合 #3410](https://github.com/hat47x/tei-platform/pull/3410) の `internal/currencyrate/fallback_declaration.go` は `RoleFallback = "currency-rate-fallback"` を導入し、既存 `ir.World.Semantics`内の `Attribute.Role` を `ResolveFallback` で参照する。行値を`FallbackRows`で読むことにより`none`/`annual`という規則候補を選び、知らない文字列・空文字・複数行の規則不一致を拒否する。`internal/validation`にRoleを追加し、5 Go Testの対象を固定。X-TEAの旧動的ランタイムに埋もれた月次0比較を、少なくとも**モデル宣言と行値の組合せで観測できる**有限なProduct実装である。

ただしこれは**ドメイン実装が固定Role名と値を理解する**実績であり、[Draft #2860](https://github.com/hat47x/tei-platform/pull/2860) のtest-only `ParseSelected`が選択した汎用Role Interpreterや、Plugin-first D1のactual Providerを通じた自己記述実行ではない。`RoleFallback`はコードに定数として存在し、汎用Descriptor定義から意味契約を生成した事実はない。この実装をSD-1/SD-2の実用的な入口とし、SD-3『自分で定義したRoleを選択済み実Interpreterが解釈』の受入上限を越えない。

**既存実コードから判別できる見落とし候補（まだ不具合認定しない）**：`FallbackRows`は `[]map[CanonicalID]string` を受けるが、引数に実Read/Completeness/Scope/Source/AsOfがない。Roleが指定されたDataでも`len(rows)==0`では現実装が`FallbackNone`を返す。`row[d.Attribute]`は未存在Keyと明示空値をGoの同じ空文字として読む。現在の5試験では、全行取得済みの本当の0件と部分取得/非開示での0件、未存在とNULL相当と明示空、同じ値だが別Scope/Providerの比較、旧Rate Revision改訂の再資格化を実証していない。**行0件⇒フォールバック無しを一般的な真の結論とするには、用途別独立Read/Completeness証拠が必要**。変更要否は業務Owner/Rate機能の別判断であり、ここで失敗コード・新Loader・必須APIを直接変更しない。

### 21.2 世界の一部分を決定へ渡すときの自己記述Roleは、完全性の説明も必要

[Ontology DSL・Decision Twin #3475](https://github.com/hat47x/tei-platform/issues/3475)がO0で求めるのは、判断に必要なSubject/Relation/State/Rule/Sourceの**用途相対の切出し（Decision Context Slice）**。必要集合はモデル候補/AI自身が`required=[]`と宣言して確定させず、#3262の独立Required・#2514の独立Expectedに束縛する。取得できた集合・取得できなかった必要集合・取得不能の理由・粒度・観測時刻/有効期間・改訂・現在Disclosureを分ける。

同じSource/Modelでも Observation、Inferred、Desired、Committed、Predictedは異なる意味Roleであり、表示・外部SDKの同型値を使うことがあっても無音で同一状態に変換しない。Caseとして『在庫18と見える古い記録、現物10、確保6、未確定入荷8』のような、見かけの同一値に対し時点・権限・未確定性が異なる外部資料を使えるが、具体的な業務Expectedは#3475/SEI #677が所有し、#2620で第二Decision Engine/シミュレーション/主体性を生やさない。

**自己記述のDecision/Action役割もCandidateに留める**。現場の即応・局所手続きでさえ、Role=Action宣言は現在のHost実行資格を与えない。SEI側へ渡すときは `Inquiry`（判断要求）、`Delivery/Reply`（通信）、`Decision`（主体の判断）、`Operation`（実行意図）、`Receipt`（外部Effect証跡）、`Outcome`（観測）のIdentity/Attemptを別に管理する。TEIにSEIの目的・記憶・承認を置き換える万能Hostは作らない。現状は**O0の設計比較・read-only ContextのFit/Gapのみ**。実Ontology DSL・SDKは後続O1、実Twin/Simは#3475で定めたセルフビルドとSUI基本機能完成後のGateを維持する。

### 21.3 REA由来SourceとSHO/SUI文書では、同じRole文字列でも意味の根拠が違う

リバースエンジニアリングで発見された旧SQL・実テーブル・転記文書・DB観測は、最初は外部Evidence/Structural Candidateである。TEIの型/Relation/Roleと同名でも、原版、取得手法、Source位置、Identity/Temporal Grain、暗黙条件、NULL/部分集合、Ownerが許した受理範囲を保存しない限り正式Semanticや更新対象へ昇格させない。REAの正本/Adapterは既存#950/#3394系、Authoring・`*.tei.md` Readerは#2621/#988/#2823、SHOのSource backlink/ProvenanceはSHO固有Ownerへ委譲する。

通常MarkdownからRoleのような文字列を読めても、Loaderの `ManagedFiles.ReadByLoader=false` なら意味として未解釈である。[Issue #3346](https://github.com/hat47x/tei-platform/issues/3346) は**closed**だが、閉鎖を `*.tei.md` 正式Loader/Parserが成立した証明としない。[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031) は未対応Sourceを拒否する限定Guard、正式Readerは未接続。同一Item/SubjectのCanvas表示Occurrenceと原文/Meaning受理は別、表示ProjectionのLossから逆編集先を推測しない（#2497）。OntologyのSDK/SHO文書を生成できても、Source相対のMeaningと現在Host資格は再検証する。

### 21.4 増分再生成と自己ビルド：ついにcmd/teiの実Stage2再入、ただしG4とは別

前§20時点では driver 自身の限定再入だけを参照したが、[main統合 #3433](https://github.com/hat47x/tei-platform/pull/3433)で `tei self-build`の薄いchild起動入口、[main統合 #3440](https://github.com/hat47x/tei-platform/pull/3440)で**cmd/tei自身のstage0→stage1→stage2物理再入実測**に進んだ。stage1 Artifactの実digestがstage2 Invoker digestと一致し、同一child/Artifact比較が成立する限定証拠が記録されている。さらに[main統合 #3463](https://github.com/hat47x/tei-platform/pull/3463)はWorkspace/State/Outputを明示した呼出外形とFail-closedを追加した。

ただし #3463 の `-intent` はまだ**記録のみで実Build Intent解釈をしない**。`qualified=false`を維持し、実Source Closure既定資格はrejected、Seed Trust/供給Trust/Owner受理は別。従って **SB-C1物理L3の限定的成功≠TEI標準ModelのBuild Intentを自己解釈して全体を再構築したG4成功**。Role/DSL自体の自己解釈SD-3も、物理再入によって成立したとはしない。SDK/Role DefinitionからGenerated Artifactへの変更影響はSource/Model/Toolchain/Provider/Target別に必要な範囲のみ失効させる。新Build Host/Registryを追加しない。

### 21.5 追加する限定反例 ESD-38〜44（独立Scenarioへの委譲、現時点NOT RUN）

| Case | 一軸を変える観測 | 必要な判別／Owner |
| --- | --- | --- |
| ESD-38 | RoleなしRate Data vs `currency-rate-fallback`付与、双方2行完全取得 | 宣言無いときのみ明示規則なし。Roleの有無と行値の妥当性は別。実mainの#3410局所Referenceを使用。#2620/#2514 |
| ESD-39 | Role宣言済・0行完全取得 vs 部分取得/権限で見えない0行 | `FallbackRows`の現戻値が同じでも、用途相対の確定規則は未保証。独立Read Completenessがない場合はunknown。Rate Owner/#2586 |
| ESD-40 | 指定Role行Keyを欠落、明示空、NULL相当とする | Go map lookupの空文字同値を意味上の同値としない。Missing/NULL/Explicit Emptyの受入をRate/Standard Data Ownerに確認。#2495/#2514 |
| ESD-41 | 同じ正しいRule文字列だが、Role/Plugin/World/AsOfの一つだけ変更 | 旧Conformance/SDK/生成物の現在資格を無条件再利用しない。無関係なSourceは過剰失効しない。#2600/#951 |
| ESD-42 | REAから同名Data/Roleを採取し、旧SQLにのみ更新条件を発見 | 構造候補を保持し、独立ExpectedとOwner受理までTEI Action/Source Patch/実Commitを認めない。#950/#267 |
| ESD-43 | Role付きSubjectを判断Contextへ渡すが、必要な負の制約Ruleだけ未Loaded | Rule不在ではなく判断Context partial/unknown。SEIへ存在断定/Decision/Grantを自動伝播しない。#3262/#3475 |
| ESD-44 | cmd/teiのStage2再入は成功、Build Intentは未解決／sourceClosure rejected | SB-C1物理再入の限定成功とG4未資格を併記。Role SD-3・Target実行認可を付与しない。#2831 |

### 21.6 新規実Go ReferenceとCIの再資格化

**追加テスト（保存済・実CI判定前）**：`reference/dsl-language-plugin/self_description_workspace_boundary_test.go`に `TestSelfDescriptionRawDocumentRevisionIsNotSemanticRoleRevision` を追加。同じ`semantic/probe.tei`＋Workspace Manifestを固定し、未解釈 `docs/design.md` のRole風文字列だけA→Bへ改訂して**別々の実 `workspace.Load`→`BuildSourceIndexFromLoad`**を実行する。独立Expectedは、原文Artifact fingerprintは改訂で変化、Manifest digest・Parsed World・Semantic Source/Referenceは変化なし、Raw DocumentにはSemantic Definition Referenceが発生しないこと。これはWorld/Source Fingerprintの対象範囲を区別する局所回帰であり、Raw文書の開示許可、正式Role Parser、決定能力、非公開二世界Noninterferenceの証明ではない。

**CI真偽とベース差異**：前§20のコミット `e0a60786ffe84bc68c4e27bfa59c37a1944cb241` の[Health #42396595545](https://github.com/hat47x/tei-platform/actions/runs/42396595545)はcompleted/**failure**。実ログで Go test/vet/tei validateはPASS、gofmt FAILが指すファイルは `tools/reference-plugin-capability-coverage/coverage_test.go` で、当PRのGo新Testではなく並行mainから仮統合に含まれたもの。依存の脆弱性検査はsuccess。新テスト追加の現在HEADについてGo/Healthの合格を未確認のままにし、他レーンのFormatting失敗を自分のRole Testの意味不一致と誤報しない。既存mainの整形修正は該当Ownerが対応し、本Draftでは無関係ファイルを巻き取らない。

**実装上の次手**：まず既存CLI/Studio [Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)/[#3371](https://github.com/hat47x/tei-platform/pull/3371)の型付きInspector注入点を同一Contractへ整合させ、#951のexact Selected Provider→実Call siteをTraceする。Role/Rate値の一般化は選択済Interpreterを介した実受入が成立した後の比較とし、現在のmain #3410を過大一般化しない。OwnerとTargetを変更しない第二Resolver/Host/Meaning Writer/BOM Engineは作らない。今回のOntological/REA/SEI横断はO0設計・有限Fixtureだけであり、将来のO2/O3/O4を現P0の直列Hard Gateにしない。

---

<!-- tei-role-domain-source-row-triangle-20261108 -->
## 22. 2026-11-08：Role定義・行観測・原資料改訂の三つの時計を実Workspaceで分離

### 22.1 今回の実装Sliceと受入の位置

main[#3410](https://github.com/hat47x/tei-platform/pull/3410)の為替フォールバックは `Attribute.Role="currency-rate-fallback"` と行値で業務規則を選ぶ最初の具体的な実用例である。前§21ではDomain `ir.World` をGoで合成した既存ReferenceとRaw文書改訂のProduct API試験を区別していた。今回は既存[Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)の `reference/dsl-language-plugin/self_description_workspace_boundary_test.go` に **`TestSelfDescriptionWorkspaceRoleFeedsExistingCurrencyFallback`** を追加。実`.tei`ファイル→実`workspace.Load`→同一LoadからSource Index→`currencyrate.ResolveFallback`→`FallbackRows`の有限接続をコードに固定した（追加元[commit 84e77bc4](https://github.com/hat47x/tei-platform/commit/84e77bc4857d948c49d2becc3651bc7d75c5f32b)、別Data値対照[commit a55f973a](https://github.com/hat47x/tei-platform/commit/a55f973acd7905ff83f6b11c297e45521c6eb848)）。**Go Test実行/CIはこのコード追加時点NOT VERIFIED**。本試験は`-workspace`を通したParsed Worldを業務専用Go処理へ渡すもので、実DB/Providerの行取得・汎用Language Pluginのexact選択・正式Role Required Shape/Owner受理ではない。

同じ実行で以下の非同値を識別する：(i) `.tei`にRoleあり＋2行`annual`→`annual`、(ii)同じRole Sourceで2行`none`→`none`、(iii)未知行値／同一Data内の規則混在→拒否、(iv)同じ行値でもSourceからRole注釈除去→`none`。さらにRoleがあるのにRows未取得(`nil`)の場合も現Domain APIは`none`となるという**観測値**を固定するが、Read Completenessなしにその出力を『実世界で代替規則が不要』というAccepted Meaningへ昇格しない。

### 22.2 三つのRevisionと資格の失効を混同しない

| 変更内容（固定対照） | 原Source/Index revision | Parsed World/Role meaning candidate | 実Rowset/Domain result | 独立して確認する資格 |
| --- | --- | --- | --- | --- |
| **Raw `.md` 文面A→Bのみ** (§21の実Test) | Raw Artifact Fingerprintは変化 | Semantic World/Referenceは同一 | 未観測・影響不明 | Context PacketのSource Index Pinはstale候補。意味計算の再利用候補と別 |
| **`.tei` のRole付与を外す** (§22の実Test) | Source Artifactは改版 | 同Data IDでもRole意味が異なる | 同じ行で`annual`→`none` | Domain規則の再計算が必要。Role解釈に必要なVersion/Ownerを再検査 |
| **Role Source固定・行値`annual`→`none`** (§22の実Test) | Source/Role定義は固定 | Parsed Worldは同一 | Domain resultが変化 | 実Read completeness・Provider/Scope/AsOfが無ければ現在の業務結論は未資格 |
| **選択Provider/Rule/Toolchain改版**（既存#1251/#2600の設計負例） | Sourceが同じ場合もある | Role Descriptorが同じ場合もある | 解釈/結果の同義は未保証 | selected→actual call、Conformance、Build/Owner Evidenceの再資格化が別途必要 |

Source Fingerprint一致だけでは現在のResult同値を保証しない。また**Semantic Worldが同一でもSource Indexが改版されたら、旧Context Packetを現在の出典証拠として使い回せない**。main統合[PR #3477](https://github.com/hat47x/tei-platform/pull/3477)が実CLI Context→ScenarioでWorld同一・SourceIndex変更（LF追記）の旧Packet拒否を検証済み。逆にRaw文書の文面変更だけで**依存していない全計算を必ず再実行する**必要もない。『意味上の再計算候補』と『現在のSource/Packet資格』を別々の依存判定として扱う。#2600が再生成、#1163がEvidence Packet、#3262がDesign Contextの既存正本。

### 22.3 Context Required拡張はmainで進行：申告の存在はFunction Invokeではない

本節執筆時の[main統合 #3482](https://github.com/hat47x/tei-platform/pull/3482)は、固定 `sales.order.submit` Capability/`order_entry` Application/Source・3 Rule・DB申告を既存**単一Context Consumer**に追加した。対象PR exact HEADの[専用CI #42462672188](https://github.com/hat47x/tei-platform/actions/runs/42462672188)およびmerge後mainの[push専用CI #42462861657](https://github.com/hat47x/tei-platform/actions/runs/42462861657)がcompleted/success：**13実read-only CLI Reply／42限定的なConsumer Controls**。同World/SourceIndexのContext Packet freshnessと6 Pinned Scenario＋既知T1-Aの局所検証を保持する。C1/NFR/外部H3/未知T1-Bは対象外であり、全Repository HealthやOwner受理と同義ではない。

最重要の非主張は、`Capability`/`Application`/`uses semantic sales`等が揃っても**Function Invocation Binding、Query/Provider actual Read、Host Grant/Effectが `not-observed`**ということ。`sales.order`の宣言閉包と今回の`fx.rate`専用Role試験は**異なるSubject/用途/Attempt**であり足し合わせて『一Function→Role Interpreter→Provider実行』の成功としない。Role専用の第二Context Readerを増設せず、#3262のRequiredSet・#951のactual Provider/ABI・#2586実Function Read・#2514独立Expectedにそれぞれ引き渡す。

### 22.4 収束させる最短受入、二重正本と過剰計画を排除

**S0（今回実装・CI判定待ち）**：一つの`fx.rate`実SourceのRoleあり/なしと、同じSourceの異なる行値を固定独立Expectedで判別する。これがGreenでも『Domain-specific `ResolveFallback`実行』まででSD-3汎用Interpreter合格ではない。部分/非開示Readゼロと`FallbackRows(nil)`の差は#2586/Rate Ownerで将来のRowset completeness入力を検討する。Rate APIの現在動作をここで黙って変更しない。

**S1（main基線を再利用）**：#3482の13実CLI・42限定Controlに同一用途の必要Role/Value/Operator Source Closureが加えられる場合だけ、Source/World/Context PacketのRevisionを比較する。新しいRole受入は元`fx.rate`とは別fixtureであり、旧 `sales.order`へのRole付与を勝手に行わない。

**S2（先行Draftとの接合）**：[CLI Service seam #3381](https://github.com/hat47x/tei-platform/pull/3381)と[Studio Host seam #3371](https://github.com/hat47x/tei-platform/pull/3371)の両DraftにあるInspector注入口を、既存#951のexact Provider selectionへ一Consumerずつ実配線する。B選択→A actual callをResult同値でも拒否、Plugin構造の説明だけで成功認定しない。Role Interpreterへの接続はread-only Inspectorのactual-providerが成立した後。

**後続**：`*.tei.md`の意味Source読取は#2621（[closed #3346](https://github.com/hat47x/tei-platform/issues/3346)は未読込が明示されるProduct挙動の限定合格、正式Parserではない）、REA/SHO由来EvidenceのCandidate採用は#3394/#267、SEI判断は[Ontology #3475](https://github.com/hat47x/tei-platform/issues/3475)のO0設計比較に留める。セルフビルドの物理L3とBuild Intent解釈/G4は#2831で別審査。現行Critical Path G0〜G4を変更せず、Raw資料の存在やRole文字列からRuntime権限・SEI Decision・DB更新を生成しない。

**追加独立反例（Scenario正本への提案・この節で実行しない）**：ESD-45 同Role Source・異Row値、ESD-46 同Row値・Role除去、ESD-47 Source-only改訂でWorld同一の旧Packet不正流用、ESD-48 Capability/Application全宣言有りでも実Invoke欠落、ESD-49 Rate規則Rows0件のcomplete/partial/非開示区別。新Runner/Registry/文法/Host、第二Evidence SSOTは作らない。

---

<!-- role-self-description-prelude-import-consumer-20261112 -->
## 23. 2026-11-12：標準型Preludeの自己記述と、Import/実Provider/Consumerの根拠連鎖を段階接合

**本節の位置**：#2620が意味Roleと自己記述の正本、#988が有限Language/Bootstrap、#4350/#4351〜4353が標準Facet/Mapping/表現型の段階実装、#3262が用途相対Required Context、#1251/#951がPlugin選択・実呼出、#2586が実Function Read、#2831が自己Build、#2514が独立Expectedを所有する。本節はその接合と停止条件の最小差分。既存G0〜G4、Core最小/Plugin-first、Go単一バイナリ、業務専門家可読なDSLを変更しない。新Type Registry/Role Resolver/Host/全体Gateを追加しない。

### 23.1 `std.*`の『意味があるのにSourceがない』を初回Bootstrap圧力にする

現main `internal/stdlib/semantic.go` の `SemanticNamespace()`は`std.binary`〜`std.time`の12 primitiveをGoで作るM0 in-process Preludeである。実CLIの観測では`model catalog`に`prelude:true`と`sourceStatus:not-indexed`が併存し、`inspect std.text`にはCanonical型があるがArtifact/Referenceがない（#2620のCLI実測）。これは意図的な現状であって、`sourceStatus:not-indexed`を『型が存在しない』とするのも、『Sourceが揃った』とするのも誤り。型名を`.tei.md`へ機械的に移しただけでは実Loader受理・Primitiveの意味・現Binaryの型実装を証明しない。

**有限Bootstrap候補**：K0=Parser/型識別の最小固定Seed・バイナリ中の標準Preludeを明記、K1=同じ12型を標準データ/選択済みLanguage ReaderでSourceとして記述できる候補、K2=既存Preludeと別途固定ExpectedによりID・型・NULL/欠損/数量精度・Source span/Revision/依存閉包を対照、K3=Sourceを正式に解釈した版のみStandard Library実体へ切替。K0をK1の自己申告だけで消さず、eagerな自己参照を拒否しdeferredな型参照とは区別。名称・型数・見かけのCanonical一致だけでK3に進めない。正式Package/Plugin分離方法は#988/#951の受入事項とし新方式を先決めしない。

**Source由来の二値化を避ける**：`builtin/prelude (compiled-in)`、`indexed/loaded (Source+Reader+Revision)`、`candidate (Raw/未受理)`、`unknown/unavailable`は観測上異なる。Preludeに架空のArtifact path/Source spanを付けない。Build Artifactのdigestはコード込みPreludeを固定する助けにはなるが、可読なDSL Source/Owner Accepted Meaningの代用品にはならない。

### 23.2 Imported ProviderのVersionはConsumer World/Indexとは別の根拠

main統合[#3477](https://github.com/hat47x/tei-platform/pull/3477)は同WorkspaceのWorld不変・Source Index改訂時に旧Context Packetを拒否する有限実試験。main統合[#3479](https://github.com/hat47x/tei-platform/pull/3479)の局所Go反例は、**Import先Providerの原文やManifestが変更されてもConsumer World/Indexが変わらない場合がある**ことを示す。従って用途Requiredの閉包が外部Providerに達する場合、Consumer Manifest/World/Indexだけでcompleteとしない。Provider Manifest Identity、実Source occurrence/元byte、Import binding、実Reader消費と同AttemptのSource/Scopeを別Claimとして要求する。

Provider Source/Manifestがunchangedでも、選択Language/Role Operator/Plugin Build/Host Grantが変われば実解釈・受入資格は失効し得る。一方、Required外のProviderの変更で無関係なFunction全体を過剰失効させない。Fingerprint同値はA→B→Aの履歴や同一原子Snapshotも保証しない。Provider Rawは観測できてもRoleが未解析なら`not-loaded/unsupported`、ScopeやSecretで開示不可なら内部不足理由を許可外Audienceへ漏らさない。独立Required Rの定義をCandidate自身の`required=[]`へ委ねない。

### 23.3 Shared Application APIと実Provider呼出を一つの試験で混ぜない

採用済み[ADR-0048](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0048-shared-application-api-and-transport-adapters.md)の`catalog/inspect/trace/validate`は共通Application Operationが意味判定を所有し、CLI/MCP/Web/Studioは輸送・表現を分離する。main [#3403](https://github.com/hat47x/tei-platform/pull/3403)のMCP参照Adapterとmain [#3540](https://github.com/hat47x/tei-platform/pull/3540)の通常CLI/Studio Inspector呼出はそれぞれ有限実装。だが`ok:true`/同Canonical応答から、Input Schema/validate診断の完全同値、外部Tool Reply終端受領、Current Disclosure、正式Profile選択Provider Bの実Binary・正しいSemantic Sourceの消費を一括合格としない。

最新#1251のComposition監査ではHost側がID→Entry→Manifestを照合する局所接合がある一方、自己整合した偽Entryや同一Manifest・異Semantic Sourceは見逃し得る。これを新Resolver/Registryで隠さず、既存#951選択→実Artifact/Entry→**同じLoadの必要Source**→実Call→Receiver取得のWitnessで検証する。GUI/AI向け固定Selectionは`meaningSubjectId/occurrence/revision/scope/source reference/view/allowed operation`をPinし、画面上の一時的選択解除を対象変更と扱わない（#1124/#3262）。Source/Grant失効時は編集・実Operationを再確認する。

### 23.4 実Functionと自己Buildの優先順位は変えない

初回の製品縦断は#2586の`order_entry.recent_orders`で **Function→Query/SQL→選択済み実Provider→実Rowset→Mapper→Data Context→Consumer** を一Source/Scope/Attempt・手書きExpectedで測る。`sales.order.submit`の13実CLI/42 bounded Context Controlsは別Functionの局所実績であり、この実Read資格へ転記しない。UI/SEI/SHO/KON/Ontology/Twinや表現型全対応を前提とせず、標準型Prelude/Roleの検証はこのP0-Rと並行する独立レーンとする。

main [#3760](https://github.com/hat47x/tei-platform/pull/3760)では明示Build Catalog+digest+source-rootから`tei self-build` entryがin-process Planを導出し、参照Planとのfingerprint一致を記録した。しかし`qualified=false`/G4 UNQUALIFIEDが正である。後続のL2-b試験では実Disk Artifact byteの再ハッシュで虚偽申告は拒否できたが、Proxy背後のBuilder同一性、childが実際に使用したstate、Source trustまで確認できるとは限らない。Self-build ArtifactがRole意味や`std.*`のSource受理を自動承認しない。

### 23.5 追加の独立反例 ESD-50〜57（設計候補、今回NOT RUN）

| Case | 独立Expected・拒否条件 | Owner |
| --- | --- | --- |
| ESD-50 | `std.text`は`prelude:true/not-indexed`。存在自体は真、Source全件完備は未成立。ユーザーDataの`indexed`と分離 | #2620/#988 |
| ESD-51 | 同名`std.decimal`をRaw/未選択Sourceで再定義。名前一致でPreludeを上書きしない・循環Seed受理しない | #988/#2514 |
| ESD-52 | Standard Source版の型/単位を変えたがGo固定Preludeが同じ。差分を未解釈/不一致として診断し、IR一致だけでMeaning採用しない | #2620/#4351 |
| ESD-53 | Import Providerの原文A→B、Consumer World/Source Index不変。依存Requiredの現在Source充足はunknown/stale | #3479/#3262 |
| ESD-54 | Provider Manifestのみ変更、Consumer Revision不変。Provider側Manifest Revを別Pinし、無関係Providerは過剰失効しない | #3479/#2600 |
| ESD-55 | 選択Provider BのManifestは正しいが実Entry A、または同Manifest異Semantic Source。Result同値でも選択/Source Consumption未証明 | #951/#1251 |
| ESD-56 | CLI/MCPのCanonical同値、validateの意味診断/Tool受領に差。Operation成功と診断・配送資格を分ける | #3419/#3420/#3262 |
| ESD-57 | Role正しくParse、Rowset 0件/partial/非開示、またはGUI選択が旧Revision。結果/表示成功を完全Read・操作資格へ昇格させない | #2586/#1124/#2514 |

**最小Exit**：E0 Preludeの実CLI現状と固定12型を再現、K0/K1/K2の失効・衝突を手書きExpectedで評価（新Proto APIは不要）。E1 #3479外部Provider Source/Manifest版の差を用途Requiredへ結び、必要なProviderだけunknownに落とす。E2 #2586一Function実Readと#951正しいProvider Entry/Sourceを**同一Attempt**で照合。E3 ADR-0048共通APIのCLI/MCP受領結果を検証。全ケースが局所PASSしてもOwner Meaning/正式Provider S2-product/全Source原子性/盲検T1-B/外部H2/H3/実DB Commit/G4は別資格。既存#1119 G0〜G4と#1180/#2514 Scenario正本は不変。

---

<!-- tei-role-one-function-actual-use-and-observer-trust-20261112 -->
## 24. 2026-11-12：自己記述の合格を「同名の定義」から「一用途で実消費した証拠」へ進める

**今回の増分判断**：§23は標準Prelude、Imported Provider鮮度、共通APIの**候補と必要条件**を整理した。本節は、11/12の他レーンで確認された実Sourceと敵対的なBuild観測を使い、**Source／Meaning／Realizationの同値性が一致しない場合に、どこで止めるか**を一Functionの最短受入に固定する。新たなCore型・独自Host・第二Resolver・Reader・Test Runnerは作らない。実際の採否は#2620と関連Ownerに委ねる。

### 24.1 三種類の同値性を独立に判定する

| 同値性 | 一致に必要なもの | 一致しても不足するもの |
| --- | --- | --- |
| **名称・構造** | Workspace/Subject/Role/Query ID、型、宣言Shape、Version | Sourceの参照先、物理SQL、適用時点、業務意味 |
| **意味** | 独立RequiredとExpected、単位・NULL/欠損・粒度・関係・時間・Scope、Ownerが受理した適用範囲 | 実選択Provider、実DB Read、実Host権限 |
| **実消費・観測** | exact Source occurrence／Manifest／Dependency closure、選択Plugin/Provider/Artifact／実Entry／Invocation Attempt、実RowsetとConsumer受領 | 未受理のMeaning、現在Grant・最終Effect／Outcome |

これらのいずれか一つがUnknownなら**その列のみUnknown**とする。Source bytesが同じ、Semantic Worldが同じ、Build成果物のbytesが同じ、CLIとMCPの返却JSONが同じ、という観測はそれぞれ別Claimであり、横断の同一性は自動的に成立しない。存在を否定する場合、当該Universe/Scopeが**閉じている証拠**が必要。未ロード・未選択・権限不足・途中切断・欠けたRequiredの条件では「不存在」を返さない。

### 24.2 同名Queryの実際の食い違いを、自己記述Roleの最初の異種Domain試験とする

現mainでは二つのWorkspaceが**同じ** `order_entry.recent_orders` を別Sourceとして定義する。[main #4258](https://github.com/hat47x/tei-platform/pull/4258)/[#4344](https://github.com/hat47x/tei-platform/pull/4344)により、Reference側は `public.orders` / `:from` / 5列、X-TEA側は `sales."order"` / `:status` / 2列とSource-staticに区別される。Reference SQLが`:from`を要求するのにQuery宣言に `input from` がないことも、Runtime Input Bindingとしては未解決。同じQuery ID・結果ID・似た業務説明から **同一Selection／同一SQL／意味同等**とは言えない。

[旧Draft #4076](https://github.com/hat47x/tei-platform/pull/4076)はclosed/unmerged。#2586の11/12 current-main監査では旧Draftの `ExecuteBrowseQueryInEnvironment` と `internal/functioncontext` の該当製品パスを確認できず、旧Branchの実PG試験PASSを現在mainのProduct R0に加算できない。[Draft #4482](https://github.com/hat47x/tei-platform/pull/4482)／[Draft #4484](https://github.com/hat47x/tei-platform/pull/4484) は最初の未接続辺を一Functionで直す入場計画。自己記述側は、この既存P0-Rを置換せず、同一IDでも `Workspace Root → exact Query Source → SQL asset → declared Input → selected DB Alias/Instance → actual Rowset → Function Context → Consumer` を判別するRead-only契約だけを追加する。正例の全Rowset・0件・partial/NULL・実Provider B/A取り違えは#2586/#2514の独立Expectedが所有。

### 24.3 Inspectorの実装済み範囲と、証明していない実装同一性

main統合[#3381](https://github.com/hat47x/tei-platform/pull/3381)/[#3371](https://github.com/hat47x/tei-platform/pull/3371)/[#3540](https://github.com/hat47x/tei-platform/pull/3540)により、CLIとStudioへHost-owned read-only Inspectorの接合点は存在する。ただし現行の局所Go Call・Host Inventory・実関数Entry記録だけでは、**正式Profile/Contractが選んだ外部Providerの独立Build/Link、同一Load Source、現在Grant、二Consumer実製品資格**まで立証されていない。#1251の[main #4258](https://github.com/hat47x/tei-platform/pull/4258)関連試験で、B選択→B Entryという局所正例と、Inventoryの一貫した虚偽／同Manifest異Semantic Sourceという残留負例は別扱い。結果JSONやProvider IDが一致しても、実ConsumeされたSourceを再確認する。

Roleを第一級のDataとして表現できることは、Provider ConformanceやOwner Meaningを**自己承認することではない**。`std.*`固定PreludeをSourceへ移すK0〜K3も同様に、現Go実装の12型と可読な候補定義の型名一致をIndependent Expectedにしない。必要な場合は異型/NULL/精度・依存循環の負例を挟み、既存#988 Bootstrap/Language Selectionと#4350 Facet・Mapping工程を使用する。

### 24.4 『選ばれた対象』と『実際に使った状態』を検査の主語にする

Studio/AI対話では、ユーザーが固定した `meaningSubjectId / occurrence / source reference / revision / scope / selected view / allowed operation` を**一つの作業Contextとして保持**する。図面切替、Canvas再配置、別Sourceに同名Subjectがある、後から届いたReply、別Provider版への解決変更だけでは、AIの編集対象を無音切替しない。Source/Revision/Policyが失効したら対象の再束縛・Previewと独立再Readを要求する。UI表示上のSource pathや隠しRoleの存在を開示権限の代わりにしない。Ownerは#3262/#1124/#2516/#267。

この原則はBuildにも適用する。[Issue #4486](https://github.com/hat47x/tei-platform/issues/4486)の11/12実測では、self-build childが渡されたstate Pathを**申告だけ正しく書き戻し実際には共有**していても、現行申告比較は `l2b.claimed=true` に到達し得る。空ディレクトリを作るだけで事後確認もすり抜ける。一方、Linux user/mount namespaceで書込を境界制約した限定実験では負例を停止できた。ただし実験コードは未コミット、他OS/本番Entrypoint/Source Closure/G4は未検証。**自己申告・後検査・Host強制**を Evidence basis 別に表示し、同じ`claimed`を独立観測と呼ばない。OS境界をRole自己解釈や一Function P0の新Hard Gateにせず、#2831/#4486に委譲する。

### 24.5 一軸の独立反例 ESD-58〜65（#1180/#2514に委譲・NOT RUN）

| ID | 独立に変える要因 | 必要なExpected／Owner |
| --- | --- | --- |
| ESD-58 | 同じQuery ID、別Workspaceの `public.orders` と `sales."order"` | 同名の宣言は別Source/SQL/Parameter・別Expected。Source-static対照を#2586へ引渡し |
| ESD-59 | Reference SQLの`:from`だけが必須、QueryにInput宣言なし | Sourceとして読めてもRuntime Input Closureは未成立。暗黙Bindを拒否。#2586/#4359 |
| ESD-60 | SourceもResultも同型だが、Selected=B/実Entry=A | 実Profile→Artifact/Entry/Source/Invocationを比較し、値一致でもFAIL。#1251/#951 |
| ESD-61 | InspectorのHost Inventoryは一貫するが同Manifest・異Semantic Source | Hostの自己報告ではSource実消費を証明できない。独立Source witnessを要求。#1251/#3479 |
| ESD-62 | 同名・同位置の別SubjectへCanvas遷移後、旧AI Replyが届く | PinされたIdentity/Revision/Occurrenceが一致しない編集Intentは再束縛／拒否。#3262/#1124 |
| ESD-63 | Requirementの対象集合がpartialで未読込の唯一Ruleだけ不足 | `required=[]`・0件・正常JSONを全件不在にしない。独立Required／Scopeを明示。#3262/#2514 |
| ESD-64 | self-build childが二回の共有stateを別Pathと自己申告し成果物は一致 | 申告ベースの成功とHost強制の隔離を分け、G4に昇格しない。#4486/#2831 |
| ESD-65 | 同じ12 Prelude型名・同じCanonical投影だがNULL/Precision/Role依存の意味差 | 文字列・構造同値を受理済みMeaningとしない。K0〜K3独立Expectedで段階停止。#988/#2620 |

### 24.6 最短の実行順と終了条件

**E0（自己記述側）**：現main `std.*` Preludeの固定12型とSourceなし状態を実CLIで取得する既存§23のExitを再利用し、Ownerが受け入れる意味Expectedを列挙。新Parserへの移設は最初のPoCでは求めない。**E1（P0-R、横断最優先）**：#2586が `recent_orders` のRoot/SQL/Parameter差をSource-staticで固定し、1件の実PG Provider Read→Rowset→Function Context→通常CLIへ一Attemptで到達できる最初の不足辺を修正。**E2（S2-product）**：#1251/#951が正式Provider Selection・実Artifact/同一Load Source・通常CLI/Studio Call-site一致を反証可能にする。**E3（Consumer/Build）**：ADR-0048 CLI/MCPのMeaning診断/完全受領と、#2831のBuilder/State観測基盤を個別に確認し、同一Attemptの十分なWitnessのみ#1163へ渡す。

**停止条件**：前段の局所PASSを相互に足し合わせて全Function/全Required/Owner受理/G4とは呼ばない。実Function Read、OS強制隔離、全Prelude Source化、未知Rule T1-B、正式Meaning/Authorityを、この文書追補の時点で完了扱いしない。UI固定Selectionと共通APIは実Provider未結線の代替ではない。過去のPR・Issueの実測と、今回のESD-58〜65の未実行提案を区別する。

## 24. 2026-11-13 横断統合：自己記述の製品価値・適格性・第二の意味正本を作らない実証順

本節は、#2620 の既存SDX/ESD、#4536 の最小Role実装、#4359 のDSL表層比較、#4350 の標準データ様式、#1124 の統合Modeler、#2514 の独立検証基準、#1251 のPlugin実Provider、#2831 のTEI全体セルフビルドを**一つの製品像と実装順へ照合する追補**である。上記のIssueの正本・受入権限を移さない。本節の追加は設計であり、新Product E2E、正式Role Definition/Interpreter、Owner意味受理は行っていない。

### 24.1 到達像と、同じ「自己」を混同しないこと

TEIの自己記述は「開発者がTEIの言語・型・Role・Operator・Generatorの意味と依存関係を標準構造で読め、変更候補を検証・再生成できること」を最優先価値とする。利用者のためには、簡潔な業務記述・三視点の相互参照・逆引き・影響分析・保存後再検査につながることが重要である。Go ASTや現行Semanticの制約は実現性・互換性を調べる対象であって、目指す仕様そのものの上限ではない。

| 到達段階 | 何ができれば成立するか | それだけでは成立しないもの |
| --- | --- | --- |
| D0 定義を読める | Role/Function/Test/Generatorなどの記述を通常データとして読め、Source occurrenceと安定Subjectを追跡できる | 意味適合・実行資格 |
| D1 意味を解釈できる | 明示選択したInterpreter/OperatorとRequired Shapeで型付き候補・診断を得る | 別Targetの動作同値・Owner意味受理 |
| D2 表現を往復できる | 表・式・図・DSLなどから必要なIdentity/Relation/NULL/grain等を損失表示付きで投影・必要なら逆写像する | 自動Writable・意図しないSource書換え |
| D3 安全に改版できる | 固定された旧意味・Source/版と候補新意味を独立Expected・差分・必要再資格化で比較できる | 変更の自動承認・現在Host Grant |
| D4 自分自身を構築できる | #2831のTEI Build Intent→実Builder→実Artifact→Stage2再入を同一実際の経路で観測できる | 定義自己解釈(D1)の証拠からセルフビルド合格を推論すること |

D0〜D4は機能の大まかな到達像であり、すべてを同時に要求する共通Gateではない。Product用途ごとに必要な段階と観測を明示し、最初のRead-only実証をGUIやセルフビルドの完成待ちにしない。

### 24.2 最小Bootstrapと自己記述可能領域の境界

**固定の最小根**は既存Workspace/Sourceの読取り、基本値・Record/Sequence/Refの構造識別、版と実装の既存選択、必須入力の検査を行う。標準データで記述される領域は「型とFacet」「Role Required Shapeと適用条件」「Operator/Plugin契約」「Function/Rule/Test/Projection/Build定義」。これらは既存の意味OwnerやResolver/Builderへ明示bindingして初めて意味候補となる。記述の自己申告から新しい実行能力を生み出さない。

「Roleを標準Recordで記述できること」と「そのRecordを実Role契約として評価できること」は異なる。契約が存在しない箇所は unsupported と記録し、その不足を満たす最小拡張を #988/#951/#2620 へ渡す。万能なMeta Interpreter、第二Role Registry、別のSource Writer、新Hostを前提にしない。

自己参照のGraph/Refは遅延参照として残せるが、解釈を実行するのに未解釈の自分自身を必要とする eager閉路は拒否する。改版候補Bは**現在資格を持つ旧Interpreter A**で隔離して検査し、Source/Contract/Implementationのexact Pin、独立Expected、Ownerの通常開発相当のレビューを経たうえで切り替える。A→B→Aとsource byte一致だけで過去の受理・Current Grantを復活させない。

### 24.3 意味と表現の境界：同一Shapeでも同じ意味とは限らない

標準構造（Value/Record/Sequence/Relation/Graph/Ref/Expression）と、その構造へ適用する意味Role、型/Facet、Operator、表示表現は別の関心事とする。「同じRole」という語でも、Semantic Role、業務Actor/手続き上の担当、実装Pluginの選択、Runtime Principal/Grantは相互に変換しない。

三要素分析法のデータモデル・機能モデル・業務モデルは原典の用語・図法・相互参照を尊重する。同じGraph等で格納できても、三種のRoleを並べるだけで原典再現と呼ばない。正本の意味IDと、Source occurrence、図の表示ID、Relationのoccurrence、CRUDセル、時間条件、Identityを区別する。三視点の逆引き・参照数・未使用項目は同じ依存グラフの用途別投影とし、専用Editor正本を複製しない。

Projection PとClaim Qについて、異なる原状態A/Bが P(A)=P(B) だが Q(A)≠Q(B) となる場合、PだけではQを判断できない。Graphの多重辺、SQL由来の重複、NULL/未取得/非開示/完全0件、部分取得の否定依存、数量の単位・丸め位相、履歴時点などを**当該Claimに必要な範囲で**損失として明示する。表示が等しいことからMeaning/編集/実行の適格性を推定しない。#2495/#2497/#2500/#2544のOwnerで扱う。

### 24.4 AI・Studio・OKFとの接合

Studio/CLI/外部AI Harnessは、カーソル上の一時選択とは別に、**固定Subject（Workspace/Meaning ID/Source occurrence/Revision/用途）**を明示的に保持する。Source改版・A→B→A・遅延Reply・Current Disclosure変更時に、選択意図は維持しても現在の適格性は再評価する。図の描画edge IDを意味Relationとして自動採用しない。画面上のPreviewやAI提案からSourceへ反映する場合は、既存Typed Operation・Preview/Diff・Conditional Save・保存後独立Readを再利用する。AI Harnessは等価なAdapter候補であり、特定製品をTEIの意味正本にしない。

OKF等の文書管理、SHO文書、Git/NAS/S3等の外部Evidence保管は出典の追跡とレビューを支えるが、Readerが解釈できた事実からMeaning受理・正しさ・実行資格を保証しない。AI生成による誤った設計の混入を検出しやすくすることを重視し、承認単位はファイルまたは意味のある要素・Role単位、人的判断は通常開発相当とする。AI生成Test/Expectedの自己追認は#2514の独立期待値で防ぐ。

### 24.5 Claim相対の最小観測票（第二Schema/証拠ストアではない）

| 列群 | 最初に収集する実際の情報 | 不明のときの扱い |
| --- | --- | --- |
| 対象・出典 | Workspace / Subject identity / Source occurrence・span・raw digest・Revision・Manifest raw | 対象を仮の同名Subjectへ付け替えない |
| Requiredと実読 | 外部根拠付きRequired Source、Declared Source、Loader実読、実Parser/Interpreter/ConsumerのConsumed | 自己申告required=[]を完全性の根拠にしない。未読込はunknown |
| 解釈 | Source表層、Role use-site、Shape、Semantic/Facet、Language/Role/Operator実選択・推移pin、型付きCandidate/診断 | Role文字列/Type ID一致だけで汎用解釈や意味同値を宣言しない |
| 利用条件 | Target/Artifact/実Builder・実Provider/実Consumer、現在Scope/Grant/Disclosure、Effects/Receipt | read-only/preview、Build success、過去の受理を実行許可にしない |
| 検証 | 独立Expected revision、既知Mutant、Actual、負例、Coverage分母、Claim上限、同一Attempt識別 | skippedをPASSにしない。別Attempt・別Subjectの緑を加算しない |

内部の照合は用途ごとに Required(C) / Observed(C) / Undischarged(C) を分け、Claimに必要なSource/実装/Consumerのどこが最初に未証明かを特定する。Source/World/Artifactの指紋が同じでもManifest必須Root、外部原文、実Provider消費、現在権限が同じとは限らない。観測不足なら unknown、契約欠落なら unsupported、反証に成功した場合は FAIL/insufficient を独立に記録する。

### 24.6 実装計画：Roleの局所成功から汎用化へ進む条件

**P0a（#4536：実Go Fit/Gap）**：mainの既存fx.rate、Role重複拒否、Source Index、既存Language Plugin Selection/Workspace Loaderから一つのSubjectの読取経路を再現し、上の観測票を記載。既存テストを再利用し、全Managed Sourceを読んだと推測しない。現行Goの限界は理想仕様の否定ではなく最小Gapとして扱う。

**P0b（#4536：独立Expected）**：Role有無と同Role異行値の対照に、必須Source未読込、同Role/型でも異なるNULL・partial・完全0件の一因子反例を加える。ただし現行Rate APIがそれらを区別できない場合は黙って既定値を修正せず未対応と記録。実Source/同Revision/同Attemptの範囲を固定し、#1180/#2514の既存Scenario/Expectedを使用。

**P0c（#4536：汎用化の最低条件）**：標準Record/RefのRole Descriptorを、専用データ項目名をハードコードせず**既存の解釈/選択契約へ明示bindingできるか**を調査。単一ドメイン専用関数の成功を汎用Interpreter成功にしない。必要なRequired Shapeやimplementation pinが欠ける場合は、不足フィールドと実消費点を示して#988/#951/#2620へ返す。

**P1（異質な第二Subject）**：fx.rateだけに最適化された構造を避けるため、既存Validation（例：sales.order_line.quantity_positive）等、意味・型・必要Sourceの異なる**一つのSubject**に同じ解釈手順を適用する。元のRoleと無関係なSubjectを同一Attemptと称しない。#4359の新rule糖衣はそのParser/anchor/既存Validationへの変換が独立に成立した後で比較し、旧構文を無断移行しない。

**P2（異種Consumerの限定往復）**：同じ受理済みMeaning/RevisionをCLI JSONとStudio/SHOの二種の表現へ投影し、元Source位置・参照・必要なIdentity・診断・Claim相対の情報損失を確認。編集可能性とHost実行権限は別の検証とする。Product Inspector/実Providerは#1251、実DB Readは#2586、Modelerは#1124、全体セルフビルドは#2831がOwner。

**性能の検収**：P0の既存Read-only baselineと比べて、余分なSource scan、再Lex、未選択Pluginのload、依存全走査、増分再検証範囲、CLI応答時間・メモリを計測する。最初から数値目標を仮定しない。Source位置/影響追跡に必要な情報を減らして高速化しない。IDE/Studioに新たな解決エンジンを持たせず、既存Go Application ServiceとCLIを共用する。

**完了の停止線**：#4536の実Go接合は局所P0の完了、#4359は表層と意味変換、#1251/#2586は実Provider/Read、#2831は実TEI再入をそれぞれ独立審査する。旧#1119 G0〜G4に新しい全件必須Gateを追加しない。今回のESD/SD-S0の拡張Caseの新規実装・実行、Product Role Descriptor Interpreter、Studio/SHO、Current Grant、実Target/セルフビルド資格は **NOT RUN / NOT PROVEN**。
## 25. 2026-11-13 横断深化：自己記述と生成AIによる生成器生成を接続する

本節は§24のD0〜D4を置き換えず、標準データに記述した意味から**生成器を選択・合成・必要なら生成AIで拡張する**際の最小契約を補う。実装責務は [#4544](https://github.com/hat47x/tei-platform/issues/4544)（G1/G3）、[#1063](https://github.com/hat47x/tei-platform/issues/1063)（生成時DSL）、[#2016](https://github.com/hat47x/tei-platform/issues/2016)（既存生成器選抜Reference）、[#2595](https://github.com/hat47x/tei-platform/issues/2595)（検査／実消費Source）、[#951](https://github.com/hat47x/tei-platform/issues/951)（Plugin実選択）、[#2514](https://github.com/hat47x/tei-platform/issues/2514)（独立検証）に維持する。§24のRole自己解釈 #4536 と平行に進め、双方の完了を互いのP0のHard Dependencyにしない。

### 25.1 既存実装を先に使い、未接続の意味だけを検証する

2026-11-13時点のmain `tools/reference-generator-selection/README.md`で確認できる範囲は、受理済み`line-amount`契約／固定fixture、複数Generator candidateの比較、Go/Python/JavaScript/Java/WASM/WASI候補のテンプレート生成、外部AI候補のSource/Prompt/Context記録、Registry/失効、宣言Sourceを検査後に一時snapshotへ渡す実装である。これらを再び新規Generator Engineとして設計しない。**宣言Sourceのsnapshot読取は、Buildに含まれる全file・import/require・toolchain・network等の必要閉包や実OS権限を証明しない**。#2595/#1370へ実行帰属を渡す。

生成器生成の製品価値は、生成AIが毎回すべての実装を作り直すことではない。標準／ドメイン／案件別Pluginの既存部品を第一候補とし、**再利用→TEI-native Transform合成→不足部の固有Plugin→通常の独自コード**の順に比較する。これは唯一の適格な方式を固定する優先順位ではなく、費用・柔軟性・Target制約・意味の保持で候補を比較する順序である。ローコード採用と通常のコーディングを排他的にしない。

### 25.2 四つの正しさは別のClaim

| 項目 | 固定するもの | それだけでは証明できないこと |
| --- | --- | --- |
| 意味契約 | Accepted Subject/Role/Type/Facet/Operation、NULL/未取得、Precision/Unit、Effect/Scopeと独立根拠 | 実Generatorの適合・Target利用資格 |
| Generation Intent | Target Profile、既存Operator/Generatorの選択、必要な合成順序、解消できないGap | 選んだ実装がその通り実行されたこと |
| Generator Candidate | Template／手書き／AI生成の由来、実Source/依存/Implementation exact Pinと契約 | Generated Artifact全入力・挙動の意味同値 |
| Generated Artifact/Consumer | 実Builderが消費したSource/Toolchain、Build Artifact、実Consumer Read・Invoke/Effect | 入力契約全体の独立正しさ・現在の認可 |

**Generator DefinitionやRole Descriptorを標準Record/Relation/Refで表現できる範囲**を最初に調べる。正式な新Core型・公開Grammarを必要と断定しない。Descriptorが申告する`pure`/`safe`/`required=[]`、AIの説明文、`build ok`は実挙動と必要Source閉包の証拠にならない。明示Pinと既存Provider/Build/Hostの実観測を用いる。

### 25.3 実行順：G1 → G3 → G2

1. **G1：生成規約（最初の縦断）**。一つの受理済み`line-amount`契約とTarget Profileに対して、標準Generatorで足りる場合、TEI-native Transform合成で足りる場合、固有Plugin/通常コードが必要な場合を分類する。分類は候補の提案であり、適合・権限の確定ではない。生成AIは不明なNULL・丸め・履歴・単位を推測して埋めない。
2. **G3：意味的Conformance（G1の直後）**。既存#2016の固定Fixtureと#2514の独立Expectedを使う。正常入力、0/missing/null/未取得、丸め位相・境界値、集合の重複・順序・粒度、Targetが表現できない契約を区別する。対象契約に存在しない状態を架空に受理させず、`unsupported`／`unknown`と表現可能範囲を報告する。結果一致だけではテストの判別力を示せないため、NULL→0やhalf-even→half-upなど既知Mutantを落とす。
3. **G2：Generator/Plugin合成（必要性判明後）**。二つのTransform/Operatorの結合について、入力出力型、`A→B`と`B→A`の順序、部分失敗、実選択されたProviderと実Call、Effect/Target制約、同名同版の実装差を対照する。合成可能であることから交換可能・副作用なし・すべてのTargetで同値という結論を導かない。汎用の合成代数や新Selectorを先行開発しない。

### 25.4 独立反証と実装境界

最小ケースは #4544 の GG-01〜10を参照。**第一受入の範囲はGG-01/02/03/06**と、現行Meaning/Target契約が持たない情報のFit/Gapに限定する。GG-04/05/07/08/09/10は既存Ownerの後続試験へ渡し、全件を最初の生成器PoCの合格要件にしない。

特に次の二つの失敗を重視する。

- **同じテストを生成器と生成AIが共通の誤理解から作る**：FixtureやExpectedを候補自身に生成・修正させて合格を評価しない。正負例と独立Expected、既知Mutantの判別力を#1180/#2514から固定する。過剰な大量の生成テストより、異なる誤りを発見できる少数の検査を優先する。
- **自動生成と手書きコードの境界を壊す**：生成対象file・拡張hook/Plugin・人が保守するSourceを明示してDiffと所有範囲を確認する。候補Sourceの再生成は既存のAccepted Meaningや人が編集したコードを無条件に書き換えない。再生成時の衝突はPreview/停止にし、Source正本と派生Artifactを分離する。

### 25.5 開発体制、成果物と停止線

高度なデータ／業務モデリングは開発者が主導し、必要に応じてドメイン専門家と協業する。基盤担当者は標準PluginとGenerator契約を整備し、生成AIは標準部品の選択・接続、差分提案、検証Caseの増補、狭い固有実装を支援する。**人の意味受理は通常開発相当**とし、毎回別の形式的な承認機構を追加しない。Source文書はOKF/SHO等の出典・変更追跡で支援できるが、文書を読めた事実をMeaning Acceptanceへ昇格させない。

具体的な引渡しは、#4544（G1/G3候補選抜）、#2595（実Build input closure）、#1121（Target-neutral Lowering）、#943/#1370（Artifact/実行同一性）、#2600（増分再資格化）、#2831（TEI全体セルフビルド）。最初の実証はCLI/headlessを先行し、Studio/SHO/Web/EKIなどの完成を必須にしない。TEIの通常Go単一バイナリ配布と、必要に応じた外部Java/.NET/WASM Targetの方針を維持する。

**Evidence：本§25は設計書への追補のみ。** 現行mainの参照生成器READMEと#2016/#2595/#1063等を静的に照合した。GG-01〜10の新規Go/Target/AI候補試験、Descriptor→Generatorの製品縦断、再生成の実Artifact、正式Owner意味受理は **NOT RUN / NOT PROVEN**。既存#1119のG0〜G4を変更しない。

## 26. 2026-11-13 製品縦断への収束：自己記述・生成器・Function起動・セルフビルドを結ぶ

§24のD0〜D4、§25のG1→G3→G2は、**成立を区別するための段階**であり、それぞれに別の製品エンジンを設ける意味ではない。Studio・Function起動・データ様式・AI Harness・Self-Buildの議論を踏まえると、最上位の製品目標は**TEI Studio Desktopを構築し、その実成果物をTEI自身の定義・実行経路で再構築できること**である。実Functionの起動は製品機能を先に検証する独立レーンであり、Desktopセルフビルドそのものではない。GUIが完成するまで製品CLI/共通APIの実証を止めない。

本節は既存の[#4495](https://github.com/hat47x/tei-platform/issues/4495)（Function→Entry→実Invoke）、[#4536](https://github.com/hat47x/tei-platform/issues/4536)（Role読取）、[#4544](https://github.com/hat47x/tei-platform/issues/4544)（Generator構成・Conformance）、[#2831](https://github.com/hat47x/tei-platform/issues/2831)（全体Self-Build）を**別Ownerのまま接続**する。新Coordinator、第二Resolver/Registry/Host、新しいG0〜G4 Gateは作らない。

### 26.1 実験対象は異なる。同名・同値を同一契約とみなさない

| 対象 | 既存の成立範囲 | まだ主張できないこと |
| --- | --- | --- |
| fx.rate（#4536） | 実WorkspaceのRole表層、Source Index、ドメイン専用Fallback解釈 | 汎用Role InterpreterやGeneratorへの接続 |
| line-amount（#2016/#4544） | 固定Contract・Fixture、多言語Generator候補と選抜Reference | 製品Function Entryと同一Meaning/版であること |
| #4495 pure Function（例：数量2×単価300＝600） | 共通APIと製品CLI二入口からの実Provider Invokeを目指す最小計画 | 同じ結果のline-amountを意味同一と判定すること、self-build |

**最初の製品結合対象は#4495のAccepted Function/Entry一件**とする。#4544のGenerator CandidateをそのFunctionの同じMeaning/Contractに束縛できるかは、Identity/入力・出力型/NULL・丸め/Target Profile/Source revisionを比較してから決める。異なる参照Fixtureの成功を同一Attemptの合格に加算しない。束縛不能なら明示unsupported/Gapで止め、Generatorを恣意的に作り替えない。

### 26.2 最短の実証順と並走の境界

| 順序・Owner | 最小検証と成立した場合の上限 |
| --- | --- |
| **P0-X #4495（Function実行レーンの最優先）** | pure/read-only Function一件を同じEntryから直接Typed Application APIと実tei CLIで起動。Source/Meaning/Entry/selected Provider/loaded implementation/actual Invoke/Resultを一改訂で追跡。結果一致でもB選択A実行は拒否。Product Function起動の局所資格まで |
| **P0-S #4536（並行）** | fx.rateの独立Required Sourceと実Reader/Role解釈を接合。汎用GeneratorやGUIを待たずにGapを特定 |
| **P0-G #4544（並行）** | 既存line-amount Contractで標準Generator選択・意味的Conformance・既知Mutantを検査。参照Generator合格をP0-Xに足さない |
| **P1-J #4495×#4544** | 同じAccepted Function/ContractをInputにできる場合のみ、Generator構成→生成したEntry/Provider→製品CLI実Invokeを一Attemptで比較。Contract Fit不成立ならjoinも不成立 |
| **P1-B #2831（並行）** | TEI Build Intent→実Builder入力→実Artifact、さらにStage1実プロセス自身によるStage2 Build呼出しを分離実証。二回Go buildしただけでは再入成功にならない |
| **P2-UI #1124/#3422** | 同一Subjectの固定選択をStudio/図表へ投影し、CLI・Typed APIと出典/失効/結果を比較。GUI完成はP0-Xの必須条件ではない |

#4198の型付きLifecycle Graphは不変Obligation＋可変Hookの候補だが、Graph Compiler完成を初回pure Function起動の前提にしない。最初は既存APIで型/選択/実Invoke/現在資格/Resultを検査する。業務Effect・DB更新の実証は#1129/#1174等へ委譲し、合成pure成功を更新権限へ拡大しない。TEI Studioは一つの意味正本と図表/DSL/SQL/テストから実システムにつながることを優先する。

### 26.3 既存Scenario正本へ渡す独立反例（仮ID／未実装）

| ID | 一因子変更 | 期待する停止線 |
| --- | --- | --- |
| SJ-01 | Product FunctionとGenerator候補の出力値を同じに保ち、Meaning/NULL/丸め/Contract revisionだけ変更 | 同一契約としてjoinせず、unsupported/不一致を表示 |
| SJ-02 | Selected Generator/Provider B、実Loaded/Invoked A（結果は同値） | 実Identity不一致でFAIL |
| SJ-03 | 自称Required空、しかしManifest rootまたは実Consumed依存が欠落 | 完全性はunknown、空の検査成功としない |
| SJ-04 | Meaning不変でGeneratorだけ変更／未選択Pluginだけ変更／レイアウトだけ変更 | 必要な部分のみ再資格化。旧Artifactの不当再利用と全域無駄ビルドの両方を検出 |
| SJ-05 | Stage1/Stage2 digestは同じだが、外部Launcherが二度Build | Stage1自身の実Process／Stage2実Invokeが無ければSelf-Build L3不成立 |
| SJ-06 | Studio/AIが固定した対象のWorkspace/Source/Grant改版後に旧Replyが到着 | 旧対象への誤再束縛・旧Grantの現在利用を拒否 |
| SJ-07 | 同一表示Graph/CRUDに異なるRelation多重度・Source occurrenceが潜む | 投影の一致から元Meaning/逆編集/実行資格を推論しない |

SJ-01/02はP1-Jで優先、SJ-03/04は#2600/#2595、SJ-05は#2831、SJ-06/07は#1124/#2497/#2516へ引き渡す。独立Expectedの正本は#1180/#2514に置き、実装候補自身や生成AIによる正解の自己作成を避ける。

### 26.4 必要な証拠はClaim相対の最小列でつなぐ

**Source/Meaning**＝Workspace/Subject/Source occurrence・raw byte/revision、Manifest/独立Required集合、Role/Operator/Facet、Accepted Meaning・Contract revision。

**Composition/Build/Runtime**＝Target、Selected Generator/Provider/Pluginとactual bytes、build-onlyとruntime dependency、実Builder消費入力、Artifact/Process image、Entry/Operation/Attempt/実Invoke。

**独立検証/Consumer**＝独立Expected・Mutant・Actual Result/Failure、現在Host Grant、開示に適格なConsumer、Claim上限、未確認/未対応/未実施。CLIとAPIは外側のJSON bytesではなく、型付きOutcomeとFailure/出典の意味を比較する。

**性能・失効**＝Role/Generator/Plugin改版から実際に再解析・再生成した範囲、所要時間・メモリ。非選択Pluginやlayout-only変更によって無関係な意味を無効化せず、必要なSource/Build/Host改版は見落とさない。

これは既存Source Index/Scenario/Receiptの**照合票**であり、新しい公開Schema・Proof Storeではない。欠けた入力を既定値補完せずunknown/unsupported/not-run/failedで区別する。ドメイン専門家による業務意味の確認は通常開発相当とし、生成AIの誤解を独立Expected・出典管理（SHO/OKF等）・レビューで検出する。

### 26.5 今回の状態

#4495/#4536/#4544/#2831と既存設計を静的に照合した設計追補のみ。SJ-01〜07の新Go/CLI/Studio/Builder実行、同じMeaningのRole→Generator→Function→製品Invoke結合、TEI Stage1実プロセスからのStage2再入、Owner意味受理は **NOT RUN / NOT PROVEN**。既存#1119のG0〜G4は変更しない。
## 27. 2026-11-13 Desktop自己構築を最優先にする実装計画と最小自己記述

### 27.1 製品の目的と、各レーンの優先度を補正する

**製品全体の第一目標は、TEI Studio Desktopの実構築と、TEI自身によるDesktop再構築である。** §26のFunction起動 #4495 は、稼働システムとの接合を確かめるFunctionレーン内の先行実証である。Role自己解釈 #4536、Generator選抜 #4544、DesktopのWails利用 #1824、全体Self-Build #2831を一つの新しい完了Gateにはせず、既存実装を接続しながら並行して進める。X-TEA三要素/Modeler/Data ScriptのDesktop Plugin移管は第二の製品柱だが、最初のSelf-BuildのHard Dependencyにはしない。

2026-11-13のmainでは cmd/tei-studio/go.mod は独立Go module（go 1.26.11、Wails v3.0.0-beta.29、root TEIへのreplace）。cmd/tei-studio/main.go の StudioBridge は既存 studiohost.ReadOnly のList/Catalog/Validate/Inspect等へ委譲する。**BridgeやrootのGoテストの存在を、実Desktopバイナリ、WebView、モデル駆動生成の合格と読み替えない。** 実OS向けTarget、選択済みnested moduleのgo.mod/go.sum、Wails/Node/Go依存と実成果物を別に確認する（#1251/#1824）。

### 27.2 自己記述・生成・実Desktopの必要閉包を分ける

| 閉包 | 必要な情報と確認 | 成立しない主張 |
| --- | --- | --- |
| 定義 | 実WorkspaceのSource raw/occurrence、独立Required Root、Meaning/Role/Operator/Generator Contract、Language/Implementation pin | データとして読めた＝受理済みMeaning・全Required完備、ではない |
| Build | 実Builderが消費したSourceと生成file、nested module・Toolchain、外部Seed、Stage1実image、Stage2実call、Artifact digest | Build成功・go.sumの存在だけで全依存や再入を証明しない |
| Consumer | Desktop/Wailsの実Process・WebView、共通API/Provider・Result・Current Grant、OS/Target依存 | CLIやReferenceのPASSは実Desktop製品完成ではない |

Build Graph外の未宣言状態が**存在しない**という主張は、事後列挙やchild自己申告で証明できない。#4486の実測に従い、観測境界Wを限定し、必要な資源についてOS側の強制と独立した観測を組み合わせる。Linuxの隔離結果をWindows Desktopの完全保証へ転用しない。強制できない環境では当該完全性をunknown/unperformedとする。現行tei self-build outerは実Childの参照入口であって、qualified=false等の未受入境界を消さない。

### 27.3 Desktop構築の最短4段階

| 段階・Owner | 最小実装・受入 | 成立Claimの上限 |
| --- | --- | --- |
| DS0 現行Desktop Seed（#1824） | 実Target OSでWails AppをBuild・起動し、Process/WebViewとCatalog/Inspectの実一往復を確認。Nested moduleを含む | 通常Desktopが動作した。TEIからの生成を意味しない |
| DS1 生成部品の一件接合（#1824/#4544） | 同じMeaning/Revisionのread-only View/Typed Operation一件を既存Generator/Transformから生成し、実Wails asset/Bridgeへ組み込む。生成物と人が所有するSourceを区別 | 一つのTEI駆動部品が実Desktopへ届く。Desktop全体生成ではない |
| DS2 TEI Build Intent（#2831/#943） | TEI標準Build定義から既存Builderに実入力を渡し、Studio nested Go依存・Toolchain・Target・生成物digestを束縛 | 範囲付きL2。TEI生成image自身による再入は未証明 |
| DS3 Stage1自身の再入（#2831） | Stage1で生成したTEIの実Process image/argvがStage2 Build Intentを解決し実Builderを呼び、Stage2 Desktop Artifactに到達 | 実験範囲のL3。外部Launcherによる二度Buildは不可 |

DS0とDS2は早期に並行し、DS1では最小のread-only部品だけを選ぶ。DS3で不足するのがTEIの実入口なら、既存Build AdapterやTyped Operation/CLIに薄く接続する。新しいHost、万能Meta Interpreter、全面DSL移行、新Build Engine/Source Writerは先行開発しない。Studio GUI/WebViewの検収はHeadless結果とは別に実施する。

### 27.4 直近mainの進捗を接合する

**実Read #2586**：[main統合PR #4539](https://github.com/hat47x/tei-platform/pull/4539)は、order_entry.recent_ordersのPostgreSQL 16.15実Read（Environment→SQL→Rowset→Function Data Context）と別プロセス tools/reference-browse-read のJSON受領を実証した。**通常tei CLI、Studio、Harnessの受領は未実行**。R0b通常CLI→R0c同じFunction/RowsetをStudioへ接続し、必要なProvider/Source/Scope/NULL・0件の情報を保持する。R0bはDS0/DS2と並走してよい。RLS/全件・不存在や現在性を推論しない。

**三要素分析法 #1124**：[main統合PR #4534](https://github.com/hat47x/tei-platform/pull/4534)はCRUD矛盾の全件報告と、構造OKとレビュー資格の分離を改善した。Data/Function/Businessの原典意味・CRUD/参照・図面をStudioへ投影する際、この既存Go構造検査を再利用する。構造検査PASSや描画IDからMeaning受理・編集権限を自動生成しない。

**AI/SHO/OKF**：#4515/#4503の固定Pinは操作対象の意図を維持するものであり、現在のRead/Share/Write/Run権限の永久保存ではない。Source/Workspace/Grant改版後も対象を勝手に付け替えず、現行資格とDisclosureを再確認する。許可されたContextだけHarnessへ共有し、SHO等の保存済み文書は出典と再読取で確認する。外部Harness/CloudがなくてもLocal Desktop利用を可能にする。

### 27.5 最初の反証と実測列（既存Scenario/Expectedへの引渡し）

| ID | 変更・故障 | 期待 |
| --- | --- | --- |
| DS-N01 | root Go Build成功、Studio nested moduleの依存不足 | Desktop合格とはしない。実Target dependencyを報告 |
| DS-N02 | CLI Catalog PASS、実Wails ProcessまたはWebView未起動 | DS0未成立。GUI未確認をPASS扱いしない |
| DS-N03 | 選択したGenerator Bと実Buildに使われた旧実装Aが同じ表示結果 | Input/Selected Pin不一致でFAIL |
| DS-N04 | Stage1/Stage2 digest一致、実際は外部Launcherが二度Go Build | Stage1実ProcessがStage2を起動しなければDS3不成立 |
| DS-N05 | state非共有をchild申告/事後一覧だけで評価 | #4486のOS強制境界がなければ隔離のClaimはunknown |
| DS-N06 | 実PG Reference Read成功、通常tei CLI/Studioには未接続 | Reference、CLI、StudioのClaimを独立に評価 |
| DS-N07 | CRUD構造OKでもBusiness意味・Function責務が未確認 | Accepted MeaningやSource書戻しを認定しない |
| DS-N08 | PinしたSubject Aの旧ReplyがSource/Grant変更後に到着 | Pin意図を保持し、旧Replyから現在利用資格を復活させない |

DS-N01〜04をDS0〜3の初回優先反例とする。観測列は Exact HEAD、Target/OS、Manifest・Source closure、実Build input、nested dependency、Stage1/Stage2 Process、Selected/Actual Builder、Artifact digest、Desktop実Consumer、独立ExpectedとClaim ceiling。第二のFixture/Receipt/Proof Registryは作らず#1180/#2514/#943/#2831の正本を参照する。

**状態**：今回の追補はmainのWails Studio module/Bridge/self-build outerとmain統合済み#4539/#4534、既存#4486・#1824・#2831および関連チャットとの静的照合に基づく設計更新のみ。DS0〜3の実Build、Stage2再入、OS境界、通常tei CLI→Studio実PG受領、X-TEA Desktop Plugin E2Eは新たに実行していない（NOT RUN / NOT PROVEN）。#1119 G0〜G4を変更しない。
## 28. 2026-11-13 Generated Desktopを最初の接合対象とする：既存CIの断線を最小化

### 28.1 確認できる二つの実試験と、両者の間に残る具体的なEdge

現在mainの `.github/workflows/tei-studio-wails-poc-a-ci.yml` の `generated-current-workbench-browser` は、PoC Bのfixture/Go staticservices/UIを生成→`connect-workbench.mjs`で現行Workbenchを固定Pin照合→`main.generated.ts`をesbuildでDesktop埋込先の `cmd/tei-studio/assets/workbench/main.js`へ出力→`studio_generated,server` tag付きGo HostをBuild→loopbackのChromiumでGraph/Command操作を検証する。Sourceと生成bundle等のdigestも記録する。**これは生成Workbenchを含むserver-modeの実Browser/Host Sliceであり、Windows Wails Desktop WebView2の実行ではない。**

別の `.github/workflows/tei-studio-wails-desktop-webview-smoke.yml` はWindows 2022/Go 1.26.11で通常の `cmd/tei-studio` Wails DesktopバイナリをBuildし、`test/desktop_webview_smoke.py`を使ってWebView2を実操作する。現行JobのBuildは `go build -mod=readonly -o ... .` で、**生成候補専用の `studio_generated` tagと直前のPoC B/Workbench生成を含まない**。したがってこの二つのGreenを足して『生成したWorkbenchを実Desktopで使った』と主張しない。現行CIの完全な今次実行結果は別途exact HEADで確認し、Workflowコードの存在だけでGreenとしない。

なお、`cmd/tei-studio/main.go`は `//go:embed assets` によって生成済みassetsを実バイナリへ収録する形式である。生成assetが存在すること、Buildがどのasset byteを取り込んだか、WebView2が実際にそのassetを表示したことは**三つの別観測**とする。`reference/tei-studio-workbench/build.mjs`にはesbuildの依存metafileとTHIRD_PARTY_NOTICES出力があるが、現行generated-current-workbenchジョブはesbuild直呼出しのため、配布時に必要な依存・ライセンス情報の保持範囲を別途照合する。

### 28.2 最初の縦断試験を既存Jobの差分として実装する（#1824 Owner）

| Step | 実変更・観測 | 明示的な失敗条件 |
| --- | --- | --- |
| **GD-0 現行主線を保存** | ordinary Windows WebView2実行とgenerated server Browserのそれぞれのexact HEAD、build tag、Go/Node/Wails版、実artifact/画面結果を保存 | 片方の成功を他方に転用しない |
| **GD-1 原Sourceを固定して生成** | 既存`reference-studio-static-composition`→`compose-ui.mjs`→`connect-workbench.mjs`を**同一checkout/固定fixture**で実施。現行`main.ts`/Command/Responses/GraphのPin、生成Source/JS/Go blobとdigestを採取 | Unknown Workbench revision、依存不一致、改変済みderived output、pin bypassを拒否 |
| **GD-2 生成assetを実Desktopへ** | GD-1で生成した `assets/workbench/main.js` と `services.generated.go` を保持し、Windows Wailsで **`studio_generated`（serverは付けない）** を付けて実exeをBuild。実Build image/Go module sum/asset digest/Targetを結び付ける | Windows/nested dependency失敗、Selected bundle Bなのに実exeがAを収録、通常Desktopを誤流用 |
| **GD-3 実WebView2/IPC受領** | 同じexeを既存`desktop_webview_smoke.py`の入口で起動し、固定WorkspaceのGraph/Command等の**生成版だと識別できる表示・IPC呼出し・結果**を採取。read-only意味は既存`studiohost`へ委譲する | 画面だけ表示、偽Host、方法未登録、未知Failure、旧Process/旧assetへの接続、未実施をPASSにしない |
| **GD-4 Self-Buildへ引渡し** | GD-2の実入力閉包を #2831 Build Intent/actual Builderへ接続し、Stage1 TEI image→Stage2生成Desktop exeの同一Attemptを追う | 外部スクリプトによる二度Buildや同digestだけではL3にならない |

GD-1〜3は**既存PoC BとWindows smokeの組合せに限定した小PR**として #1824 が所有する。専用Runtime/Resolver/Host/CI Runnerを新設しない。Source PinやTargetに不整合がある場合は理由を記録し、無制限に新しいPinを受理させてGreenにしない。既存Jobが同じ範囲を既に実行していた場合は重複実装せず、そのexact HEAD/Attemptを採用して不足だけ補う。

### 28.3 自己記述すべき最小のBuild関係（新DSL/Schemaではない）

最初の自己記述は、LanguageやGeneratorの一般再帰定義ではなく、**特定のDesktop部品一件を構築する際の依存と選択の説明**とする。既存のBuild Intent/Target-neutral Planと、Source Index/Selection/Artifact Lockに、次の情報が届くかをread-onlyに調べる。

| 既存の何を参照するか | 最低限追跡する識別子・関係 |
| --- | --- |
| Source / Definition | Workspace・Meaning/Subject・原WorkbenchのSource occurrenceとbyte/Revision、固定PinとRequired閉包 |
| Plugin/Generator Selection | PoC B Generatorと実選択Implementation、入力fixture・Go/TSの契約と出力、未選択Pluginの非干渉 |
| Transform / Derived Source | fixture→staticservicesとui.generated→main.generated.ts→esbuild JS、各入力/出力digestと実toolchain |
| Target Build | Desktop target OS、nested Go module/go.sum、build tag studio_generated、実embed対象、actual builder実入力・exe image |
| Consumer | 実Wails process/WebView2、呼出したStudioBridge method/typed API、固定Subjectと返答。read-onlyとCurrent Grantを別資格 |

**接合点が無ければ新型を発明するのではなく、最初に不足した既存契約の入力・出力欄を明記する。** 例えば生成Sourceを実Builderの入力として外部から観測できないなら、Build Intentだけを正しい意味で書けたとしてもDS2に昇格しない。Derived UIのSource code/byteが同じでも、別Toolchain/asset packaging/実Host imageなら同じ成果物とは認定しない。

### 28.4 最初の反例、他レーンの進捗、性能

| Case | 変更 | 独立Expected / Claim上限 |
| --- | --- | --- |
| GD-N01 | Generated server browser成功、ordinary Desktop smoke成功、生成Desktop試験は未実行 | `generated-desktop`はNOT RUN。PASS加算禁止 |
| GD-N02 | main.tsだけ更新・Command/Responses/Graph Pinが古い／生成bundleだけ古い | Pin/依存改版不一致を拒否。旧assetを実Desktopへ流さない |
| GD-N03 | Go Serviceは生成版、JS assetは通常版（または逆） | 実Binary/asset/IPCの同一Selection/Revisionを証明できずFAIL |
| GD-N04 | Linux server tag成功、Windows Desktop targetでnested dependency不備 | Windows Desktopの成功に昇格しない |
| GD-N05 | Build graph宣言と実Build入力が異なる／build state非分離 | #4486の強制境界Wで観測できた範囲以外はunknown、SelfBuild L3不成立 |
| GD-N06 | 生成したUI上のGraphだけ一致、意味Relation・Pin・Selected Providerが異なる | 表示同一からMeaning/実作用を推論しない |

直近mainでは[#4550](https://github.com/hat47x/tei-platform/pull/4550)のRequired Consumerの間接依存探索、[#4552](https://github.com/hat47x/tei-platform/pull/4552)の実PG→Function Context計画反映が統合済み。前者はCaller申告Graph内の閉包に限られ、全Source universe証明ではない。後者は既存[#4539](https://github.com/hat47x/tei-platform/pull/4539)の実PG Reference成功を更新した計画で、製品`tei` CLI／Wails IPC実受領ではない。**GUIをBuildしてもDB/Relationの業務意味を勝手に自動受理しない。**

X-TEAは参照元の歴史的名称として必要な範囲で使い、新規実装のPlugin/Capability命名はTEI体系へ揃える。三要素分析法（データモデル／機能モデル／業務モデル）はRole名三つに還元せず、Source位置・CRUD・参照多重度と原典の意味を保持。これらの本格Plugin移管、REA/SHO/AI連携、実PG/Wails双ConsumerはGD-1〜3のHard Dependencyではなく、同じSource/Typed Operation/Plugin契約へ段階的に追加する。

性能検収では、generated server版とDesktop版のBuild時間・JS生成時間・不要Pluginの組込・manifest依存・差分変更時の再生成範囲を**実測後に比較**する。未計測の速度改善、全OS再現性、GUIの正式品質を約束しない。正式DSL字句文法／Unicode識別子／新Target ABIもここで採用しない。

**今回の確認範囲：GitHub mainの既存2 Workflows、Workbench pin/adaptor、Studio Go embed/Bridge、Build Intent outer、他Issue/PRの静的読取と設計追補。** GD-1〜4の新規実Windows Wails Build／WebView2/生成Desktop/Stage2実行、Go/Node CI実行、Ownerの意味受理は **NOT RUN / NOT PROVEN**。#1119 G0〜G4変更なし。
## 29. 2026-11-13 最初の実装PR：生成Workbenchを実Windows Wails Desktopで使用する

### 29.1 §28の設計から実装への接続

第27節GD-1〜3を、[#4579 Draft](https://github.com/hat47x/tei-platform/pull/4579)（`ci/generated-wails-desktop-webview2-20261113`）で具体化した。**従来のordinary Windows WebView2 Jobは残し**、同じWorkflowに独立の`generated-desktop-webview2-smoke` Jobを追加している。このJobは#1824に既存のPoC B `reference-studio-static-composition`／`compose-ui.mjs`／`connect-workbench.mjs`と、既存`desktop_webview_smoke.py`を呼ぶため、新しい生成器・Desktop Host・WebDriver/Runnerを実装しない。

GD-1はexact PR HEADのSource/Command/Responses/Graph Pinと同一fixtureからGenerated Go `services.generated.go`／UI／Workbench TypeScript／esbuild JavaScriptの実byteを記録する。GD-2はWindowsの独立Studio Go module/Toolchainを使用し、**`studio_generated`だけを指定して（`server`は付けず）**native Wails Desktop exeをBuildする。GD-3は同一exeを既存WebView2 smokeで実起動し、Graph/Command、`StudioBridge.CatalogWorkspace`、`InspectSubject`、`ValidateWorkspace`等の結果を実画面で比較する。

さらに、生成時 `cmd/tei-studio/assets/workbench/main.js` のSHA-256をJob環境へ渡し、実WebView2がsame-origin `/workbench/main.js` から再取得したbyteのSHA-256と一致しない場合はFAILとする。この検査は生成版Jobでのみ有効化し、通常版Desktop smokeの合格条件を変えない。**Source byte→生成asset→WebView2配信byte**を同一Jobで照合するが、実生成Goサービスの全命令・全Plugin呼出、バイナリ内Embed自体の独立検査、完全Builder input closureや配布依存の認定までは主張しない。

### 29.2 独立する次の成果と並行レーン

| Owner/PR | 既に進んだ範囲と今回の停止線 |
| --- | --- |
| #1824／#4579 | 生成Workbenchのnative Windows WebView2・実Bridgeを検証する初回CI。Workflow committed≠CI Green。負例はsource pin不一致、生成bundleと実配信bundleの不一致、実Wails/IPC失敗 |
| #2831 | #4579で観測する生成Source／nested module／Target／実exeを、既存Build IntentとStage1実プロセスのStage2再入へ接続。#4579自体はL3の証拠ではない |
| #1124／[Draft #4570](https://github.com/hat47x/tei-platform/pull/4570) | 三要素分析法のread-only PluginをStudioBridgeから到達させる別レーン。現時点ではserver UI試験まで、Windows native Desktopは別資格。#4579の生成asset CIと混ぜてOwnerの編集領域を競合させない |
| #1121／[Draft #4559](https://github.com/hat47x/tei-platform/pull/4559) | 実Query `recent_orders` のSource束縛Function生成Referenceは、入力parameterの正式宣言・SQL Source Index等が未完了。生成UIのGreenを実Function生成やDBの完結に昇格しない |
| #4536／#4544 | Role自己記述／Generator契約のProduct汎化は独立研究。PoC B固定Fixtureでの生成Desktop成功により万能Interpreter完成と主張しない |

### 29.3 CIの最初の資格と次の判定

新しいJobを含む[#4579](https://github.com/hat47x/tei-platform/pull/4579) HEAD `5b9fb2a5300ac0b2b33476ade753596693c3407e`について、2026-11-13のPR Open後に`TEI Studio Wails Desktop WebView smoke`などのGitHub Actionsがqueued/in_progressであることを確認した。**本段階ではCI successは未確認**。正確なWorkflow run／Job step／exact HEAD／生成bundle digest／実WebView2 snapshot／失敗時のlogを取得して検収する。`server`版Green、ordinary Desktop Green、生成版native Desktop Greenは別Claimとする。直接確認できないものはNOT VERIFIEDと表示し、推測したPASSにしない。

**今回の追加は設計§29と実CI Draft PRの新設であり、セルフビルドStage1→Stage2、X-TEA native Desktop実受入、実DBのStudio受領はまだ完了していない。** 既存#1119 G0〜G4／#1180独立Expectedを変更せず、Source/Build/Consumerを連鎖させる最初の狭い実証を優先する。
### 29.4 2026-11-13 実Windows Generated DesktopのCI観測（PR #4579 exact HEAD）

[TEI Studio Wails Desktop WebView smoke Run #43361607881](https://github.com/hat47x/tei-platform/actions/runs/43361607881) において、[Draft PR #4579](https://github.com/hat47x/tei-platform/pull/4579) HEAD `5b9fb2a5300ac0b2b33476ade753596693c3407e` の**新規 generated-desktop-webview2-smoke Jobは completed/success**。同Runの ordinary `desktop-webview2-smoke` Jobも completed/success。重要なのは、二つのGreenを加算したのではなく、**新Job一件の中で生成Source→native Windows Wails exe→実WebView2を接続して合格した**ことである。

実Jobログで確認した限定証拠：Windows上の固定PoC B／Workbench生成・Pin／Go/TypeScriptの型・テスト・esbuild、`studio_generated`（`server`なし）Desktop exe Build、実WebView2テストがsuccess。`generated-workbench-asset-verified` は `/workbench/main.js` **681,264 bytes、SHA-256 `351bedd94693e80d08baca611625368c05ff59450f27c79c5a6ebc0c6e247d63`**を報告し、生成時SHAと実WebView2から再取得したbyteの一致を確認した。`workbench-catalog-inspect-validate-verified` はCatalog **72件**、初期`sales.order`→`sales.order.order_id` Inspect、検証表示**エラー0・警告3**、実画面パス`/workbench/`を記録。参照Editorはread-only、実WebView2キーボード検査も成功。これらは**このHEAD／固定Workspace／このRunnerの有限な正例**である。

**今回更新できるClaim**：§28のGD-1〜3、§27のDS1について「固定PoC B Generated WorkbenchをWindows Wails実Desktopへ収録し、実WebView2/IPC/Hostから利用できた」ことの限定実証。DS0 ordinary Desktopも同一Runの別Jobで成功。`#4559`のFunction生成や`#4570`の三要素Plugin、DB Provider、Role Interpreter一般化の資格は付随しない。**DS2/GD-4 Build IntentからのTEI駆動BuilderとDS3/L3 Stage1自身によるStage2再入は依然NOT RUN/NOT PROVEN**。Buildに含まれる全依存、OS境界W、全Platform、現在Host Grant・業務Ownerの意味受理も証明しない。

**次の実装引渡し**：#2831は今回の生成Source/Toolchain/Build Target/実Exe digestを`BuildIntent→actual Builder input→Stage1 Process→Stage2 Desktop Artifact`に接続し、Runnerが外部から2回buildする偽L3と区別する。#1824はこの実WebView2 smokeを退行基線として維持し、#4570の三視点Pluginは同じConsumer契約へ接続する際に別途実検収する。新規Registry、Proof Store、E2E Runnerを作らない。

## 30. 2026-11-13 横断更新：部品再入と生成Desktopを実Build Intentへ接合する

### 30.1 前節で未成立だった二つの辺が別々のDraftで前進した

**前提は変更せず、観測された到達点のみ更新する。** #2831の最終目標はTEI自身によるStudio Desktop全体の再構築であり、#4495のpure Function起動やGenerator自己記述は別の重要レーンである。

| Subject・PR | 実証済みの狭いClaim | 未成立・統合時の停止線 |
| --- | --- | --- |
| [#4579](https://github.com/hat47x/tei-platform/pull/4579) HEAD 5b9fb2a（2026-11-13） | [Windows Actions #43361607881](https://github.com/hat47x/tei-platform/actions/runs/43361607881) completed/success。同一JobのPoC B固定Source→generated Workbench＋Go services→native Wails studio_generated（serverなし）→WebView2 real IPC。生成JSと実WebView2配信JSのSHA-256一致（681,264 bytes、351bedd94693e80d08baca611625368c05ff59450f27c79c5a6ebc0c6e247d63） | TEI Build Intentに登録されたBuildではない。Stage1自身のStage2 Desktop build・全Source closure・全OS・実業務Meaningの受理ではない |
| [#4572](https://github.com/hat47x/tei-platform/pull/4572) HEAD be403a6（2026-11-13） | Inspector reader一つのContract＋deterministic-transform Source→生成TS→esbuild/Go embed→実Windows WebView2使用を局所実測。Linux/WSL2の stage_reentry.py では**Stage1のTEI実ProcessがStage2でGo生成器と部品を再構築**し reentryObserved=true。CI #43342784405/#43342784377/#43342784325 はexact HEADでsuccess | **Stage2にWails Desktopは含まれない。** Node/esbuildとWailsの工程はBuild Graph外。Stage2のGo childはStage0 seed由来。local実測とCI成功の対象試験も別。L2-b/G4/総合Desktop再入を主張しない |
| [#4559](https://github.com/hat47x/tei-platform/pull/4559) HEAD c5fcc1d（2026-11-13） | order_entry.recent_orders Sourceからread-only Go CLIを生成し、実PostgreSQLと既存参照Readerを比較する限定実測 | input from未宣言、SQL Source Index外、Host/Environmentと全Consumerの同一性は未資格。Studio再入の合格に足さない |
| [#4570](https://github.com/hat47x/tei-platform/pull/4570) HEAD 4d89ad0（2026-11-13） | X-TEA**三要素分析法のモデル** Data/Function/Business・CRUDのread-only Bridgeを合成Workspace/server UIから到達 | 実Native WebView2と実X-TEA原典Sourceは未確認。業務意味受理とTEI独自拡張を区別 |

いずれもDraft／未mergeである。#4579の成功した固定PoC Bと、#4572のInspector部品は異なるSubject・生成方式・コミット・実行Attemptなので、各成功を単に足してDS3合格としてはならない。

### 30.2 次に作る最小の実装差分は「Builderを管理する一本の縦断」である

**P0接続**：#2831の既存 self-build outer／Build Catalog／Build Intentに、#4572のdeterministic part transform、#4579のWorkbench/UI＋Go service composition、TypeScript→esbuild→実embed、独立cmd/tei-studio moduleでのnative Windows Wails executable Buildを**既存Plan/Builder Adapterの範囲で**表現できるかFit/Gapを先に実コードで確認する。新DSL/汎用Stage Engineを起票の前提にしない。

対象の一本は、固定されたMeaning/Source/生成器を持つread-only Inspector操作、固定PoC B fixture、同じ生成Desktop Targetを選ぶ。Source Contractと手書き比較Oracleを共有させず、両Draftの成果を実際に統合した**単一のexact HEAD**で検証する。PoC B由来の生成Go ServiceとInspector readerは別由来としてPinを保持し、どちらかを暗黙に書き換えない。

最初の実装単位：
1. **B0 枝の統合基線と反例を固定**：#4572/#4579の合流で共通cmd/tei-studio/test/desktop_webview_smoke.pyの*両方の判別条件*を保持。#4572差分のPython __pycache__/*.pyc二件は追跡不要な生成物として除外・再混入防止。両者のGo/Node/Python/WinToolchainやbundle digestが異なるため、過去別HEADのdigestを統合HEADのExpectedとして流用しない。
2. **B1 定義→Transform→Bundle→Target Buildを計画へ接合**：実Source raw/Generator Source/Contract Revision、各中間生成物、esbuild/Go embed、cmd/tei-studio独立go.mod/go.sum、Wails build tags（studio_generated、serverなし）、Toolchain・OSの入力集合を列挙。既存Build Catalog表現不能のNodeは明示unsupportedにして追加の最小Adapterを別途評価する。
3. **B2 Stage1実Process→Stage2 Desktop**：Stage1のTEI image digest/argv/execution Attempt、実Builderへの入力・Target・Artifact digestを保ち、**Stage1自身がStage2 Desktop Buildを起動した**ことを観測。Stage0/外部Python/Workflowが二度Wails buildした場合は拒否。OS跨ぎで実ビルド不能ならremote実行を暗黙採用せず、対応Target/実Gatewayの契約と認可・Outputを別検証する。
4. **B3 Stage2実Desktop Consumer**：Stage2のexeをそのexact HEADの実Windows WebView2で起動。生成partが使われた証拠、実取得bundleのbyte/digest、Catalog/Inspect/Graph/Hostの実応答を再取得し、Stage1とStage2それぞれのConsumer資格を比較。数値・digestの一部分一致を全体再現性とはしない。

### 30.3 独立した拒否条件、証拠と優先順位

| ID | 故障・一因子変更 | 期待する判定 |
| --- | --- | --- |
| BD-01 | #4572/#4579を別HEADのまま二つのGreenで合算 | Product結合未成立。単一統合HEADで実行が必要 |
| BD-02 | 合流時にWebView2 PythonからInspector使用検査またはgenerated JS digest検査を落とす | 少なくとも片方の既知負例を検出しないため不合格 |
| BD-03 | Catalogで選択したTransform Bに対してBuilderがAを実行、または旧bundleをembed | 実Consumed Source/Builder/asset identity不一致でFail。画面が同じでも拒否 |
| BD-04 | Build IntentにNode/esbuild/Wails工程が無いのに全体Builtと主張 | Build-closure不成立。未表現工程をunknown/unsupportedとする |
| BD-05 | Linux/Windows Toolchain、nested module sum、Wails tagsやGo/Node版を無断変更 | Target/Profile違いの別Attempt。過去digestの流用禁止 |
| BD-06 | Stage2 DesktopをStage0/Runnerが起動し、Stage1は部品だけ生成 | 自己再入の部分成立とDesktop全体DS3の不成立を区別 |
| BD-07 | Stage2 exe成功・WebView/IPCが未起動、または旧契約のInspector dataset | DS3のBuild Attemptと実Consumer資格を別に判定 |
| BD-08 | Build stateの宣言外読み書きをchild自己申告のみで否定 | #4486の観測境界W内にOS側の強制がない限り当該非干渉を認定しない |

実証票は Source/Contract Revision、Generation Intent・selected Generator、actual Builder bytes、Build Target OS/Toolchain、nested module、Intermediate artifact hash、Stage1 process/Stage2 invocation、Stage2 native exe、実WebView2 readback/IPC、独立Expected・negative test・Scopeを**同一exact HEAD/Attemptの参照**として並べる。既存Receipt・#1180/#2514を再利用し、第二のRegistry・Host・Provenance Storeは作らない。

**次に解く順序**：#2831 B0/B1を実装着手の主線に置く。#1824は既存Windows generated-desktop jobを継続基線とし、#4572/#4579の合流後も反例を削らない。#1124の三要素分析法のモデルPluginと#2586/#4559の実Function・DB受領は並走させるが、Desktop Self-Buildの新Hard Gateにしない。#4544 G1/G3/G2で生成器自体をAIに作らせる研究をB0〜B3の先行条件にしない。

**この§30の状態**：PR #4579 CI success、#4572 exact HEAD CI successおよびbranchの局所reentry実測、#4559/#4570のDraft試験報告を静的照合した計画更新。**統合HEADのBuild Intent→Stage1 TEI→Stage2 native Windows Desktop→実WebView2というB0〜B3の新試験はNOT RUN**。Owner受理・全Platform G4も未証明。
## 31. 2026-11-13 同一HEAD生成Desktop検収：B0実証とB1の実入力断線

### 31.1 初めて成立した限定的な同一HEAD実証

[統合検証用Draft PR #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `b26e87781e00b19bf142b22a35cfc62a857e88a3`を、#4572のInspector契約Source/Transform/Generated readerをSeedとして、#4579の独立Windows生成Desktop Jobとsame-origin JS digest検査を接続して作成した。元PRの各HEADやmainを書き換えず、重複mergeを防ぐため**一時的な統合検証PR**とする。#4583のB0がOwner。

[Windows Actions #43368714294](https://github.com/hat47x/tei-platform/actions/runs/43368714294)では、生成版 `generated-desktop-webview2-smoke` **completed/success**、通常版 `desktop-webview2-smoke` **completed/success**。生成版一JobでPoC B Go/TypeScript生成→esbuild→Windows native Wails studio_generated（serverなし）→実WebView2/StudioBridge検査へ到達。統合Python smokeは、契約から生成したInspector readerの実WebView digest/受理回数（初回/Graph後）と、実配信JSのbyte hashを**同時に失敗条件**とする。後続ログに`generated-workbench-asset-verified`が出たので、その前のInspector判定もこのJobで通過した。

実WebView2配信bundleは **682,327 bytes／SHA-256 `755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`**。#4579単独の681,264 bytes／SHA-256 `351b...`とは**別HEAD・別生成Sourceに対応する別Artifact**であり旧digestを持ち越さない。Catalog 72件、`sales.order`からGraph `sales.order.order_id`、エラー0・警告3などをログで確認。Embedded Bundle、Incremental、Independent Consumer、依存脆弱性の同HEAD CIもsuccess（全CIの終了一覧は別確認）。

統合に際して、#4572に混入していた追跡済みPython bytecode（`.pyc`）2件を**統合ブランチから除去**、`.gitignore`に__pycache__/と*.py[cod]を追加。これを正式#4572が自動的に修正済みとは扱わない。合流時は元ordinary Desktop、Generated Desktop、Inspector使用、実WebView2配信hashの四条件を一つも削除しない。

**成立Claim**＝『この固定Source/Windows Runner/Toolchain/PoC B/Contractに対し、生成Inspectorと生成Desktopを一つのexact HEADで実WebView2が消費した』。**未成立**＝#4583 B1 BuildIntentからの全Node実Builder実行、B2 Stage1生成TEI自身がStage2 DesktopをBuild、B3そのStage2 exe実WebView2、Host OS境界Wでのstate非干渉、全OS配布/Plugin/Owner受理/G4。B0の局所成功からDS2/DS3を昇格させない。

### 31.2 次のB1を構成する実Nodeと、既存APIで足りない境界

main `cmd/tei/cli_self_build_inprocess.go` は実Build Catalog raw/digest/rootを読み、`buildintent.ParseBuildCatalog`と既存解決経路を呼ぶ。一方#4572 `tools/reference-studio-part-generator/main.go` は**単一TypeScript部品のchild**を実現するだけで、`stage_reentry.py`の説明にもNode/esbuildとWails BuildはGraph外と明記されている。#4579のWindows Jobは実生成・Bundle・Wailsを行うが、**GitHub Actions Runnerが順序を管理する**。これは新Builderを作るべき根拠ではなく、まず既存BuildIntent/Build CatalogのFit/Gapを実消費工程で照合するべき根拠である。

| Build Node候補 | 必要な実Producer/Consumer・失効入力 | 現在の境界 |
| --- | --- | --- |
| N0 Inspector契約→TS | 契約raw、Transform Source/Engine、生成TSのexact bytes | #4572 childで実証。Catalog/Intent全体のNodeは未接続 |
| N1 PoC B生成Go/TS | 固定fixture、Composition Generator/Source、生成services/entry TS | #4579 Runner上で実行。TEI BuildIntentの実Nodeではない |
| N2 Bundle | N0/N1のTS＋既存Workbench/Command/Graph、npm lock・Node/esbuild、生成JS/assetのbyte | 実esbuildはWindows Jobで呼ぶが、TEI Graph外 |
| N3 Desktop Binary | N2 JSの実embed、Studio nested go.mod/go.sum、Go/Wails/Target/tags、実exe digest | Windows Jobでnative build、TEI Graph外 |
| Consumer検証 | N3実Process/WebView2、N2 served JS、Inspector使用、IPC | #4587 B0は実検証済み。ただしStage2生成物ではない |

**実装順**：B1a実Job/NodeのRead-only inventoryとBuildCatalogに表現可能なinput/output/orderingの表作成 → B1b最初に未表現のNode/Adapterだけ既存Build契約へ加える → B1c TEI自身のactual resolved plan/consumed byteとCI/OS側actualを比較 → B2 Stage1自身の実ProcessがN0〜N3を経由したStage2実exeを構築 → B3新Stage2 exeをWindows WebView2が受領。WindowsでStage1再入に直接到達できない場合は`unsupported`とし、Linuxで部品だけ再入できた過去実績を流用しない。

**次の独立負例**：生成契約のみ変更してN2の旧assetを流用／npm lockまたはnested moduleだけ変更してN3を再利用／N1 Go servicesの旧版とN0新Inspectorを混在／CI RunnerがN3を直接BuildしてStage1実Processは非参加／二回Buildのstateが宣言外共有、を一因子ごとに分離する。Expectedの正本は#1180/#2514、Build Ownerは#2831、Desktop実Consumer Ownerは#1824、B0〜B3接合は#4583。新DSL・Host・Meta Interpreter・第二CI Runnerなし。

**本節の状態**：#4587 exact HEADのWindows native Job二本成功は実証済み。B1 Build Catalog/Intent実合成、B2 Stage1→Stage2 Desktop、B3 Stage2実WebView2、正式Meaning/Owner受理は **NOT RUN / NOT PROVEN**。PR #4541は設計Draft、PR #4587もDraft／main未merge。

<!-- tei-self-description-language-contract-20261114 -->
## 32. 2026-11-14 横断統合：DSL文法・意味・Studio・実Buildの自己記述を一つの受渡し計画へ（設計のみ）

### 32.1 定義データが自己解釈・自己実行を保証しない

対象Ownerは #2620/#4536（Role自己記述）、#988/#1018/#4359（Language・構文）、#4544（Generator）、#2497（写像損失）、#1124/#1824（Studio）、#2831/#4583（自己構築）、#1180/#2514（独立Expected）。新Core Schema、Registry、Meta Interpreter、Host、Gate、Scenarioの第二正本は追加しない。第23節のD0〜D4もそのまま維持する。

自己記述の最小対象を既存のRecord/Relation/Ref/Rule/Expressionで表現できるか、次の4分類でFit/Gap評価する。**分類名は設計上の観測項目であり、新たな正式API名ではない**。

| 対象 | 宣言で表す内容 | 宣言以外に必要な成立証拠 |
| --- | --- | --- |
| Source/Surface | Language ID/版/root、token・quote・comment・identifier、元UTF-8 byte位置、糖衣表現 | 実選択されたParser/Source Reader。Grammarをデータとして読めるだけではParse成功ではない |
| Semantic Binding | syntax occurrence→既存Semantic/Role/Rule/Operatorへの対応、型・Facet・NULL・Scope・Required、正確なSelection・推移pin | 実Validator/Interpreter、独立Required Closure、独立Expected、Owner意味受理。候補自身の`required=[]`や`pure`は資格ではない |
| Projection/Editing | CLI/Studio/SHO/SUIへの形、Identity・多重性・順序・Source span、損失・可逆性 | 実Consumer read、固定Subject、revision、書戻し対象一意性、Preview/Review/再読取。read-onlyと編集可能を区別 |
| Generator/Conformance | Target、Generation Intent、選択済みGenerator、実Builder、生成Artifact、独立Known Mutant | 実選択と実Process・実消費byte・ArtifactSet・実Consumerの対応。Generator自称合格はBuild/Host資格ではない |

**自己記述の停止線**：`Descriptorを記述できる`≠`そのDescriptorに従う実Parserが選択・起動された`≠`正しいMeaning`≠`Owner accepted`≠`実Build/実Hostで利用できた`。K0〜K6の有限Seedを保ち、新版候補を旧版固定Interpreterで解釈できる場合だけ差分評価する。旧版にない構文は`unsupported`として新Parser実装・独立比較へ渡し、自己申告によるLanguage更新・fallbackを認めない。

### 32.2 一本の製品経路にするが、独立レーンを強制直列化しない

| 順 | 最小接続・実証 | Scope・Owner |
| --- | --- | --- |
| **L0** | 実`fx.rate`の同一Source・Subject・Revisionで、Roleあり/なしと行値、実Loader/Parsed/Anchor/Required/Selected/Domain Readerを突合 | #4536 P0a/b/c。`currencyrate`のドメイン固有成功は汎用Role Interpreter合格ではない |
| **L1** | 異質な既存`semantic.Validation`（例`sales.order_line.quantity_positive`）を第二Subjectとして同じ必要閉包/意味観測票へ | #4536 P1-A/#988。Go専用コード二つを並べただけでは汎用化と呼ばない |
| **L2** | 従来`validation ... positive`と提案`rule ...: ... is positive`、Markdown表を既存SemanticへLower。Source byte/spanは各表現で保持、意味同値は独立Expectedで比較 | #4359 S0→S1→S2→S3/#2621。提案`rule`は未採用、`*.tei.md`を黙ってparseしない。専門家用DSLへGo `struct/func/import`を露出しない |
| **L3** | 同一MeaningをCUI/Studio Inspector/SHOへ投影、fixed subject/Revision・構造損失・Current Disclosureを検査。no-op保存はbyte不変 | #2497/#1124/#2621。read-only Inspectorの成功はWritable Source span・意味受理ではない |
| **L4** | Accepted Meaningから実選択Generator→generated Inspector→BuildIntent/実Builder→Windows Wails/WebView2 Consumerまでの一Attemptを照合 | #4544/#2831/#4583/#1824。Role/Grammar自己記述の完了をDesktop B1/B2/B3の新Hard Gateにしない |

**今の実装優先順位**：#4583のB1-B→B2→B3（実Studio Desktop Self-build）を第一実装線に置き、#4536 S0の実接合と#4359のS0構文比較を並走。標準Meta Grammar/汎用Generator AI設計は、この二経路の不足契約が具体化してから。#1119 P0/G0〜G4のGateは変更しない。

### 32.3 2026-11-14 時点の現物・Claim ceiling

- #2638/#2639/#2898/#2860/#3028はmainにmerge済み。ただし限定synthetic/Reference/Go試験を一体のRole自己解釈合格へ合算しない。
- #4587の固定Source・同一exact HEAD Windows CIでは生成Inspectorと生成Workbenchの実WebView2消費（B0）が限定実証された（第30節）。
- #4583の2026-11-14記録では、実TEI `self-build -child`からN0 Generatorを起動し、実disk再読取したTSと実Child報告のdigestを比較した限定PASSがある。またN2 esbuildで選択されたrepo入力24件、N3 Goの選択303件、native WebView2まで一Job内で観測された。しかし`qualifiedBuildIntent=false`、全外部依存閉包/OS強制はunknownであり、N1〜N3はCI Workflowが実Invokerである。
- #4623の旧HEADはCaller指定Child digestの事前pinに関する8/8 CI成功。新HEAD `-bind-resolved-builder`の変更と最新CI結果、non-Plugin Desktop PlanとStage1→Stage2実再入は**別確認**。古いHEADのGreenを新実装・Windows・L3へ移さない。
- 現行名は`tei.workspace.yaml`、`.tei`、`std.*`等を維持。#4371のLanguage宣言Schemaや#4359のDSL新構文は未採用。

### 32.4 独立反例（仮ID。実装・追加実行はすべてNOT RUN）

| ID | 一因子だけ変更 | 独立Expected／停止線 |
| --- | --- | --- |
| SD-L01 | 表層別で値が等しく、NULL/Operator pinのみ異なる | 意味同値と断定せず、Evaluation契約の差を報告 |
| SD-L02 | Grammar descriptor同じで実Parser A→B／選択B・実呼出A | 宣言≠選択≠実消費。Identity/推移pin違反はFAIL |
| SD-L03 | 純粋なRule糖衣に副作用を付与 | 構文から実行権限へ昇格させず拒否 |
| SD-L04 | BOM/CRLF/quoted Role/Role重複 | 正確なSource byte anchor。同属性重複は第二Role位置で拒否、引用は非宣言 |
| SD-L05 | 同Label二Node・多重有向辺をTree/Graph表示 | 視覚的辺≠意味上のEdge ID。非可逆投影の自動書戻し不可 |
| SD-L06 | Required Root一件未読込、候補は`required=[]` | 独立Required閉包不成立。Role absentの判定不可 |
| SD-L07 | Source A→B→AまたはGenerator/Toolchainのみ更新 | Source/Meaning/World/Artifact/Reviewを別Revision/再資格化 |
| SD-L08 | Draft保存成功だがRole値Writable span・base revision不明 | 原文保全≠Accepted Meaning変更。read-only/Reviewへ停止 |
| SD-L09 | 新Inspector契約に旧bundleをembed | 実WebView2 readback byte/生成Artifact/Contract不一致で拒否 |
| SD-L10 | CIが二回BuildしStage1実ProcessがDesktop Builderを呼ばない、またはnon-Plugin DesktopをPluginと偽装 | B2/L3は不成立。N0限定成功はそのまま保持 |
| SD-L11 | Semantic Roleと業務ActorとHost Principalが同名 | 互いの業務担当・実Grantの根拠にならない |
| SD-L12 | 完全な0件とpartial/hidden/unavailableの0件 | 不存在/完全性を推論せずunknown/unsupported |

Expected/Mutantは#1180/#2514の**既存正本へ差分として**渡す。`structural/semantic/consumer/builder`の各Claimの独立観測列を既存Evidenceに結び、実Provider Read/OSアクセス/Current Grantの未観測部分はunknownとして残す。

### 32.5 Studio実装方式・将来SaaSへの制約

Go単一バイナリ＋Wails/WebViewをDesktop実装基準とする。ReactはDSL自己記述から必然的に決まらない。標準Record/Relation/Tree/Graphの意味Projectionと実Rendererを分離し、Web Components/Lit等Web標準寄りの構成と比較できるようにする（フレームワーク採用・全面移行は未決）。生成AIのコード生成効率より、専門家の認知順・低ノイズな日常DSLを優先する。

SaaS将来対応では複数Git branch・DB alias・Workspace/session・Build attemptを区別し、現在のHost/Providerの認証認可、資源所有・隔離をRuntime/Execution境界に置く。Source Meaning/Build Intent/Rendererに固定のローカルPath・現在のPrincipalを意味正本として埋め込まない。現Desktop一Workspaceの成功を複数ユーザーSaaSの隔離成功へ昇格しない。KON/SUI/SHO連携も新たなTEI固有Hostや第二Policy Engineを前提にしない。

### 32.6 引継ぎ・判定

P0-A #4536は実`fx.rate`の最初の未成立EdgeだけをGo/Workspace現物で判別する。P0-B #2831/#4583は同一Build IntentのN0〜N3 actual Builder/actual Consumer接合へ進める。P1 #4359/#988は旧Validationを使う低ノイズ構文のS0比較（Ownerの構文採用を待つ）、#4544は既存`line-amount`とG1→G3→G2を並走。P2 #2497/#1124/#2621は損失を表明できる投影とRev付きSource Patch候補を後続実証する。新しい横断Issue・Validator/Resolver/Hostは起こさない。

**達成資格を個別報告**：記述Dataとしてinspect可／実Parser・Role解釈／独立意味Expected／Source/Consumer往復／Generator実選択とBuilder実消費／Stage1→Stage2 Studio artifactとWindows実受領。各段階の同一Source/HEAD/Attemptを結ばない限り上位資格へ昇格しない。ここに記した新SD-L試験、正式Grammar/Role ABI、Product一体Run、Owner意味受理、B2/B3は**NOT RUN / NOT ADOPTED**。

<!-- tei-self-description-grammar-transport-20261114 -->
## 33. 2026-11-14 言語自己記述の有限核とStudio実行配置の分離（横断設計・未実装）

### 33.1 追加する設計原則：言語意味はTransport・プロセス配置から独立

他チャットのStudio DesktopとServerの分離、ローカル従属プロセス、将来SaaS、AI Harness、専門家可読DSL、増分Buildを、#2620/#4536/#988/#1018/#3421/#1824/#4583の既存Ownerへ接続する。既存[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)の依存方向を維持し、新しいTEI Hostや第二Resolver/Registryを作らない。

| 境界 | 固定する意味上の契約 | 配置・実現によって変わるもの |
| --- | --- | --- |
| Authoring | Source byte/occurrence/Revision、Language、Role、Token/span、独立Required | Editor、Markdown/Graphviz/表、File/Remote Source reader |
| Semantic | Canonical Subject、型/Facet/Operator、Meaning、診断、共通Operation | CLI/IPC/HTTP/MCPの呼出し形態 |
| Realization | PinされたGenerator/Provider/Builder、Target、Source closure、Artifact | 実BuilderのProcess、Toolchain/OS、local/remote環境 |
| Authority | Host確定Workspace/Principal/Scope、現在Grant、Disclosure/Effect | 本地Desktop、従属Local Server、共有Web/SaaSの隔離と認証 |

Language/Role InterpreterやApplication APIはWails window/HTTP route/MCP tool名に依存しない。DesktopのGo単一バイナリ配布は既定のまま。UIを薄くしてローカル従属ServerへBuild/Git/DBを置く案は**候補**であり、正式採用していない。同じ型付きRequest/Result/Failureをin-process・Wails IPC・Windows named pipe/Unix domain socket・loopback HTTP・remote HTTPで比較し、HTTPより速い等の未測定の保証は置かない。Transport失敗≠Operation失敗≠Owner受理≠Effect不確定。

### 33.2 自己記述Grammarの最小Kernel：説明データと実Parserの切断

現mainの`reference/dsl-language-plugin/selection.go`は`Selection{LanguageID, ContractVersion, ContractFingerprint, ImplementationID, ImplementationFingerprint}`という**既受理exact pinを要求するReference**。`ParseSelected`が返すのはCandidateであって意味受理ではない。`internal/workspace/loader.go`はManifest bytesのIdentity、`ManagedFile.ReadByLoader`、Load時Source snapshotを持つが、現Productのdomain rootは固定dispatchされる。**データ記述されたGrammarを、実製品が選択したParserで読む経路は未確認**であり、#988/#1018のFit/Gapとする。

有限核の最小責務（新ABI/実装Packageの宣言ではない）：

1. **K0 Seed**：固定された旧Parser/Reader・基本Value/Record/Sequence/Ref・source bytes。信頼はDescriptorから取得しない。
2. **K1 Select**：Workspace/Language/Operatorの契約版と実Implementation/推移依存をexact pin。不在・曖昧・変更時は拒否。
3. **K2 Parse**：実Parserを呼び、Source byte anchor/診断/構文候補を返す。Grammarにはtoken/quote/comment、choice/sequence/repeat、Ref、priority/associativity、Unicode識別子Policy等を**データとして表現する可能性**があるが、実Parser未対応なら`unsupported`。曖昧・無限/無進行再帰・資源上限を黙って解決しない。
4. **K3 Interpret**：候補を既存Semantic/Role/Operatorへ結ぶ。意味解釈のEager dependency cycleを拒否、deferred Graph/Tree Refの循環は別扱い。NULL・missing・unavailable・partial・complete emptyを区別。
5. **K4 Qualify**：独立Required/Expected、実Selected/Consumed、Owner通常Reviewを照合。`required=[]`や`trusted`/`pure`の自己申告を資格にしない。
6. **K5 Realize/Observe**：既存#1063/#4544 Generator・#2595実Build入力・#2831実Artifact/Consumer・外部Hostへ渡す。Grammar自己記述≠自己生成≠自己Build。

改版では旧Interpreterが解釈可能なN+1候補だけを旧版固定で評価し、独立Expected・Review・Lock更新を経て新Parserへ切替。旧版が読めない新構文は別実装と独立比較を要求。A→B→Aで旧受理を自動復活させない。Load時Source読取は、Importや外部Providerを含む完全なatomic snapshotを意味しない。

### 33.3 Claimを同一試行で追跡し、増分の過不足を判定する

| 観測 | 最小Evidence列 | 禁止する昇格 |
| --- | --- | --- |
| Required/Declared | 独立Required Root、manifest byte ID、Source occurrence、Role/Grammar 宣言 | managed=parsed、自己申告 required=[]=完全集合 |
| Selected/Actual | Language/Operator契約・実Parser exact pin、推移Lock、実Parser入力・byte anchor | インストール済み=選択済み、選択済み=実消費 |
| Typed/Accepted | Semantic ID/Type/Facet/NULL、独立Expected/known mutant、Owner Acceptance、Revision | 同じ値/同じRole名=意味同値、CI Green=Owner受理 |
| Build/Consumer | Current Disclosure、実Builder image/argv/Target、actual input/Artifact、実WebView2/Host | Build Intent存在=実Build、2回Build=Stage1再入、旧Grant=現在Grant |

#2600のIncrementalではSource移動・内容変更・A→B→A、Operator/Parser/Generator pin更新、beforeから削除されたDependency edge、未選択Plugin、非出力Predicateを一因子ずつ比較する。Meaning再評価、Artifact再生成、Evidence再受入、Current Grant再検査を同一Cache Hitで代用しない。false reuseと不要な過剰失効を別に測る。

### 33.4 Studio・AI Harness・安全なWeb公開の受渡し

- Studio/CLI/SHO/SUI/AIには同じ受理済みMeaningから、損失を明示したread-only投影を渡す。Source編集は#2497/#2621/#1124の対象一意性・Writable span・base revision・Preview/Review/独立Reloadを満たす範囲のみ。Graph多重辺のUI表示から元Edge IDを推定しない。
- AI HarnessのSkill/Tool候補は`選択済み契約→実Operation API`に基づき生成する。使用可能なToolの提示はCurrent Grantや実呼出し資格ではなく、非公開Source/Expectedを勝手にLLMへ開示しない。個々のHarnessへ第二のSemantic Interpreterを実装しない。
- 現`cmd/tei-studio/main.go`の`StudioBridge`には`ExecuteCommand*`やSHO操作も登録され、local serverは127.0.0.1のPoCに限定。#4627のexported-method inventoryは公開面の差分検査であって認証・認可・開示の実保証ではない。#3421/#1043/#1824のHostに適切な実Policy/Resource隔離を委譲し、現Bridgeをそのまま共有Webへ公開しない。
- 開発サーバーのBuild/Git/DB/AI処理ではworkspace/branch/db alias/session/attempt/Principalの結合を**実Host/Execution**に置く。DSLの意味へOS raw Pathや認証情報を埋め込まない。Remoteから切断した際にlocal Builder/Providerへ黙ってfallbackしない。

### 33.5 追加の反証Case（既存#1180/#2514へ委譲、全件NOT RUN）

| ID | 一因子の差 | 独立に判別すること |
| --- | --- | --- |
| SD-T01 | 同Pin Source、Parser Providerロード順を反転 | pin一致実装だけを選択、重複は曖昧拒否 |
| SD-T02 | Grammar descriptorを定義、新`rule`未実装 | `unsupported`。黙ったGo/旧Parser fallbackなし |
| SD-T03 | 意味Eager依存A→B→Aとdeferred Graph Ref A↔B | 前者拒否、後者Refとして保持 |
| SD-T04 | Parser/Grammar改版、Source byte同じ／A→B→A | Old Acceptance/Review/Artifactの現在資格を自動復活しない |
| SD-T05 | 旧DSLと糖衣構文の値だけ一致、NULL/短絡Read footprint相違 | Value同値≠Evaluation同値。実Provider I/O未観測はunknown |
| SD-T06 | Direct Go/CLI/IPC/loopback HTTPを同Source・Revision・Host方針で比較 | 共通Failure/診断/Meaning同値を比較。未実装TransportはNOT RUN |
| SD-T07 | IPC/HTTP失敗、Provider側のCommit成否不明 | Timeout=RollbackではなくEffect unknown、無条件再送拒否 |
| SD-T08 | 15 exported Wails Bridge methodsをWeb公開 | Method inventory合格≠公開/認可/情報開示の資格 |
| SD-T09 | 未選択PluginやPrivate RoleをAI Skill候補に含める | Discovery・Current Disclosure・Grantを分離 |
| SD-T10 | Source以外のGrammar pinまたは削除依存辺だけ変更 | 正しい局所再評価と過剰失効を独立Expectedで測る |
| SD-T11 | Local server死活/Workspace ID/DB aliasが変化し旧UI返信が後着 | 新Sessionへ旧結果を表示せず、Resource ownershipを検査 |
| SD-T12 | Stage1はN0部品生成のみ、CI RunnerがN2/N3を起動 | #4583のN0限定成功は保持、B2/G4の自己再入は不成立 |

### 33.6 反映した優先順位と停止線

**P0-A** #4536の`fx.rate`同一Source・Revision実観測と、#988/#1018におけるReference Language Selection→製品Workspace固定LoaderのGap。新Meta Schema/Parser ABIは先行採用しない。

**P0-B** #2831/#4583のnon-Plugin Desktop BuildIntent/Builder→N0/N2/N3実工程→Stage1自身がStage2を構築→Windows WebView2受領。#4623/#4624の別PR/HEADの局所試験を一体PASSとみなさない。

**P1** #4359/#2621で旧Validationと低ノイズな構文・Markdown/GraphvizのSource-faithfulness/専門家読解/独立Expected、#3421/#1824でDirect/IPC/local HTTPの共通操作・p50/p95・プロセス死活・Web公開面を別検証。数値性能目標は測定後に決定。

**P2** #4544/#2600の生成器自己記述と増分再生成、#1124/SHO/SUI/AI Harnessへの複数Consumer投影・限定編集。いずれもP0 Desktop Self-buildの新Hard Dependencyにしない。

**状態**：本節は現Sourceと他チャット議論の整合を記述したDraft設計更新。新Product Grammar/Language Adapter、ローカル従属Server、IPC比較、実Windows/Go検証、Stage2、正式Owner Meaning AcceptanceはNOT RUN/NOT ADOPTED。#1119既存G0〜G4や独立Expected正本を変更しない。

<!-- tei-self-description-obligations-20261114 -->
## 34. 2026-11-14 DSL自己記述の次の接合：評価意味・Trait義務・検証義務・製品入口（設計・NOT RUN）

### 34.1 Grammar自己記述を「実行意味の自己承認」と混同しない

この節は、第31〜32節の固定Language/Role/実Parser/Host分離に、他チャットで進んだ[#4599](https://github.com/hat47x/tei-platform/pull/4599)（構文・振る舞い・Model of Computation・Source逆写像）、[#3278](https://github.com/hat47x/tei-platform/issues/3278)/[#3279](https://github.com/hat47x/tei-platform/issues/3279)（原子的Trait/実装義務）、[#4361](https://github.com/hat47x/tei-platform/issues/4361)（独立Expected）、[#2514](https://github.com/hat47x/tei-platform/issues/2514)（Claim別Conformance）、[#4515](https://github.com/hat47x/tei-platform/issues/4515)（Studio Subject Pin）を接続する。**新しいCore Meta Schema／Trait Resolver／Oracle Generator／State Engine／Build Runnerは設けない。**

| 自己記述で参照する対象 | 既存構造へ接続する内容 | 表現できたことから主張してはならないこと |
| --- | --- | --- |
| Source/Grammar | Language exact selection、syntax occurrence、quoted/UTF-8/span、糖衣表現と元byte、Role適用 | Sourceを描画できた＝実Parserが解釈した、またはMeaning受理された |
| Structural Meaning | Subject、Type/Facet、Identity、Relation多重性、Data Context、Rule/Operator契約 | Shape同一＝意味同値、非単射Projectionの安全な逆編集 |
| **Evaluation/Time** | 純粋計算と評価順序/短絡、NULL/欠損/未取得、State transition、Event order、時点・期間、deferred/eager依存 | 宣言したRuleだけで実実行順・取得の不存在・外部Effectを保証した |
| **Trait Obligation** | Atomic Trait Definition → Requirement/Binding → method（provided/generated/manual-required/external）→ Fulfillment → Conformance | Traitラベル＝機能提供済み、Provider自己申告＝Target合格、Conformance＝Host Grant |
| Test/Proof Obligation | Claim/Subject/Requirement/Source・版/独立Expectedの参照/実RunnerとObservation | 定義から自動生成したExpected＝独立Oracle、予定したテスト＝実行済み証拠 |
| Realization/Consumer | 選択済みPlugin/Generator/Builder、Target Profile、実消費Input/ArtifactSet、CLI/Studio/SHO | 実装候補・Artifact bytes一致＝現在実行/開示資格やセルフビルドG4 |

ここでの**Evaluation/Time**は領域ごとに必要なModel of Computationを区別するための*記述上の分類*であり、TEI Coreに共通Scheduler/State Machineを設置する提案ではない。業務Function、イベント駆動UI、ゲーム・ロボティクス等の異質Domainでも、構文面の糖衣ではなく**選択した評価契約**が同値性を規定する。

### 34.2 「必要なテストを導出する」ことと「正解値を作る」ことを分離

自己記述されたType/Role/Operator/Trait/Targetから、検査が必要な項目を列挙することは有用。ただし、期待する結果まで実装候補から自動生成すると同じ誤りを共有してしまう。既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)のScenario/独立Expectedを唯一の正本とする。

**最小検収票（新共有Schemaではない）**：次の項目を既存記録への参照として一行ずつ結ぶ。

- **Target**：固定Workspace/Subject/Role/Source occurrence/Meaning Revision（同一IDでも別Workspace・旧Sourceなら別対象）
- **Obligation**：必要Source/Scope/Completeness、型・Facet・Evaluation/Time・Trait要求・未選択Provider、対象Target/Consumer
- **Independent Expected**：事前に定めた基準と根拠、正負・既知Mutant、必要Caseの集合、未実装/unknownの扱い
- **Execution**：実Parser/Interpreter/Provider/Generator/Builder/Consumer、exact Implementation/Toolchain/Attempt/actual consumed bytes
- **Assessment**：各Caseのtested/untested/unsupported/failed/stale、Claimへの充足、Observed Result、Owner Meaning Acceptance/Current Authorityの別判定

Rule宣言から`quantity > 0`の正常系・ゼロ・負・NULL等の**検査論点**を列挙できても、採用すべき丸め・NULL・時刻・空集合意味を自動決定してはならない。未決事項は`unresolved`としてOwnerの既存レビューへ送る。手動で定められた独立Expectedと既知Mutantで判別力が成立した部分だけ資格の根拠とする。

### 34.3 直近Product入口は局所Parser候補を「完全World」にしない

[#4405](https://github.com/hat47x/tei-platform/pull/4405)には、低ノイズ`rule`の限定Parser/Lowering、既存`semantic.Validation`とEvaluatorによるGoテスト、Source Anchorと局所Formatterの実装候補がある。**PRは未merge**、`workspace.Load`の正式Language選択と通常Source保存/Studio編集は未接続。[#4607](https://github.com/hat47x/tei-platform/pull/4607)は#4405依存の積上げDraftで、独立Expected目録/Case数の差を検査する方向。旧HEADのGo PASSから積上げHEADの合格を推定しない。

**最初の製品受入は一つの既存Validation Ruleに限定する**：

1. **S0 Coverage**：[#4361](https://github.com/hat47x/tei-platform/issues/4361)のSF-A〜D／EX-01〜09等の独立Expectedと、実Go TestがどのCaseを実際に読んだかの台帳を突合する。未消費のExpectedを`PASS`と集計しない。SF-DはRule identity未宣言なので、`check ... with std.op.positive@1`からRule IDを暗黙生成せず`NOT ESTABLISHED`。
2. **S1 Product Boundary**：新`rule`のParseが構文候補を部分生成しても、Workspaceに重複ID/欠落Source/未知pin/部分Loweringがあれば**完全なWorldとして受理・公開しない**。診断・元SourceはDraftとして残してよい。旧新ファイル間の重複IDは順序で選択しない。#4359/#988/#1018の所有境界。
3. **S2 Evaluation**：量が正かのような純粋Validationを固定Evaluatorで評価し、`satisfied/not-satisfied/error/unknown`、完全空集合/未取得、短絡・潜在Requiredと実Read Footprintを分離。実Provider Readがない試験を実取得/非取得の証拠としない。#988 EQ0〜EQ3を再利用。
4. **S3 Source Fidelity**：正常な局所Formatter候補があっても通常Save Writerではない。base Revision/選択Occurrence/元Syntax/非対象byte/Role anchorを照合してPreview Diff→明示Apply→独立Reloadする。no-opはzero-write。曖昧・stale・非可逆Projectionはread-only/Review。
5. **S4 Independent Consumers**：同じAccepted RuleをCLI InspectとStudio Inspector/SHOから参照し、`Subject Pin`/Current Disclosure/遅着拒否、実Reader/Consumerの差を記録。Studio側表示だけではOwner Meaning変更の資格を得ない。

S0〜S4は新たなCritical-Path Gateではなく、#4359/#4361/#4515/#2514の既存工程の**実装可能な最初の範囲**。#4536の`fx.rate` Role S0と#4583のDesktop B1-B→B2→B3は引き続き並走する。

### 34.4 Trait・時間規則とCompiler/Generatorの接続条件

Trait Definitionからの**実装義務は拘束力を持つ**。例えば`manual-required`のmemberが欠ければ`unfulfilled`、対応Providerがなく読めていなければ`unknown/not-loaded`、宣言された実装があっても未検証なら`provided-but-unverified`、Targetの独立試験が成立して初めてそのScopeの`conformance-observed`。開発時FulfillmentはRuntime Host Grant/業務受理と別。

複数Traitの合成順は、要求集合の合成として順序非依存を基本とするが、**副作用のあるOperator/Hookの実行順を勝手に可換とみなさない**。Trait A+Bを同一Targetへ実現するとき、要求するFunction/Plugin/Implementation/ABI/Target/Effectの互換性を別に確認し、競合は#3278/#3279/#951へ返す。

時間・イベント系のDSLを自己記述する場合も、正しさは`same terminal value`のみではない。State変更の因果、Event順序、重複と再送、異周期のCheckpoint、NULL・decimal roundingの位相、HostでCommitした後のReceipt/Unknownを必要に応じてClaimへ含める。モデルごとのModel of Computationを明示し、純粋なFunctionやValidatorに共通のEvent Runtimeを強制しない。実Provider/Host/Targetの効果資格は#1129/#1174/#1121、Generator選抜は#4544へ委譲。

### 34.5 追加する反証と期待Claim（提案・今回新規NOT RUN）

| Case | 一つの変更・故障 | 合格の判別条件 |
| --- | --- | --- |
| SD-O01 | Ruleの表層を変え、結果/IDは同じだがOperator/NULL policyを変更 | 構文同値と評価意味同値は別。IDや値一致だけではPASS不可 |
| SD-O02 | S0 JSONにExpected Case追加、Go Testはまだ未消費 | Required Test件数/Consumed件数を分けて未検証を明示 |
| SD-O03 | SF-D表層にRule identityを宣言しない | 名前・行順・位置から補完せずMeaning ID同値を未確立とする |
| SD-O04 | 新`rule`候補が一部有効でも、同Workspaceに旧構文同IDと不正Ruleが混在 | Diagnostic保持可、World完全受理/Build/Consumer公開は禁止 |
| SD-O05 | 同じResultでも短絡が異なりsecret/別Providerを先読みする | Valueのみ一致、Evaluation/Read/Disclosure一致を主張しない |
| SD-O06 | Trait`manual-required`が宣言され実装1 member欠落、またはProvider未ロード | unfulfilled/unknown。注記に縮退させず誤PASSを拒否 |
| SD-O07 | Traitの実装は`generated`、Generator実装だけ変更・旧Artifact再利用 | Semantic同一でもArtifact/Evidence再資格を別に判定 |
| SD-O08 | 同じ状態最終値、Event順/重複/Checkpoint/時点が異なる | 選択したModel of Computationに応じた独立Expectedが必要 |
| SD-O09 | 原Sourceの同じ`role` optionがWriter未対応、UIだけ編集可能 | UI Draft/PreviewとMeaning変更を区別し、zero-writeで停止 |
| SD-O10 | 一つのSource RevisionをStudio/AIで非同期表示中にScope/Grantのみ失効 | Pin intentは保持、旧表示/AI disclosureの現在資格は別判定 |
| SD-O11 | 選択されたGenerator Bに対して実BuilderはA/古いBundleを消費 | Typed MeaningやテストPASSとは別にBuild/Consumerの不一致をFAIL |
| SD-O12 | テスト義務をAIが自動列挙し、Expectedも同じ候補から作る | 検証義務の提案は許可、独立Oracleの資格は不成立 |

**これらは既存のEX/DTQ/DSL-I/TR/TC/BD/PINケースの対応表であり、新しいFixtureの第二正本ではない。** 重複するCaseは新しいテストを書かず既存#1180/#2514へ参照付けする。実装時の完了条件は、反例が本当に実Reader/Executorのどの境界まで到達したかを毎Caseで明示すること。

### 34.6 既存計画へ組み込む順序とPR境界

| 優先 | 最小差分の単位 | 受入／担当 |
| --- | --- | --- |
| **P0（DSL局所）** | #4361のExpected／Case目録と実Go消費を同一固定HEADで照合。SF-D未ID・NULL/empty/missing/unknown・重複を検査 | #4359/#4361/#2514、既存#4405/#4607を再利用。新Parser/第二Expectedを作らない |
| **P0（Role）** | #4536の`fx.rate`同一Source/Required/Role/Domain Reader S0 | #2620/#988。新汎用Role InterpreterはGap確認まで作らない |
| **P0（Studio Build）** | #4583 B1-B→B2→B3。non-Plugin Desktop実Builder/Stage2実Process/実Windows Consumerを同一Attemptで接合 | #2831/#1824。DSL新構文/新Traitを先行Hard Gateにしない |
| **P1** | 新Rule Product入口のfail-closed、Source局所Patch/no-op、Studio固定対象/後着を一Ruleで検証 | #4359/#4366/#1124/#4515/#2497、実適用・Meaning受理を別判定 |
| **P1（Trait）** | #3279原子Traitの`manual-required`/`generated`で実装義務・Target独立Conformanceを一Provider/一Consumerへ | #951/#3278/#3279、既存#1180/#2514。新Trait Registryなし |
| **P2** | 状態遷移/イベント系の異質Domain、生成AI/Test Obligation自動提案、複数Target/Provider・増分影響比較 | #4599/#4193/#4544/#2600。新Core Scheduler/Host/Validatorなし |

**合格の上限**：構造を読めた・構文候補を得た・意味が評価できた・Traitが履行された・独立Expectedを実消費した・OwnerがMeaningを受理した・生成物が実Consumerで使用された、をそれぞれ別Claimとして保持する。`#4405の局所GoテストPASS`を`Workspace新DSL正式採用`に、`#3279の計画`を`Trait Runtime製品化`に、`#4583 B0/N0`を`Stage2 Self-build完了`に読み替えない。

**状態**：#4541の既存設計への追補・関連Issueの引継ぎのみ。新Go/CLI/Node/Windows/DB実走、Trait Fulfillment実装、Source Writer、Product新Rule Workspace、Owner意味受理、Stage2/G4は今回NOT RUN / NOT ADOPTED。

<!-- tei-self-description-change-qualification-20261114 -->
## 35. 2026-11-14 自己記述DSLの版互換・安全な編集・DBとStudioの時点資格を接続（計画／NOT RUN）

### 35.1 今回の差分：構文が読めることと、変更できることの間の「時点」を明示

第31〜33節の有限Bootstrap／Role／Trait／独立Expectedに対し、Studio/Web・AI Harness、SHO/SUI文書、DB Workbench、Native SQL、データ/スキーマ移行、Desktopセルフビルドから受け取る**変更時の再資格化**を接合する。既存[#4367](https://github.com/hat47x/tei-platform/issues/4367)形式互換、[#4366](https://github.com/hat47x/tei-platform/issues/4366)zero-write、[#2497](https://github.com/hat47x/tei-platform/issues/2497)Projection逆写像、[#4515](https://github.com/hat47x/tei-platform/issues/4515)Subject Pin、[#4496](https://github.com/hat47x/tei-platform/issues/4496)Data Workbench、[#2514](https://github.com/hat47x/tei-platform/issues/2514)Expected、[#1163](https://github.com/hat47x/tei-platform/issues/1163)Evidenceに引き渡す。**新しいSource Writer／Transaction Coordinator／Policy Engine／Migration Engineを作らない**。

- **形式可読**：選択済みParserが旧/新表層を読めること。
- **意味互換**：同じSubject/Role/Operator/型/NULL・空集合/評価時点/Scopeのもとで、独立Expectedから意味の保持が判定できること。
- **変更可能**：一意なWritable Source occurrenceまたはDBの実行可能な対象Key+Versionを持ち、競合・損失・権限を実操作に束縛できること。
- **現在利用可**：対象・Parser/Provider/Generator/Hostの現在の資格が実Consumer時点で成立すること。

これらは直線的な単一`PASS`ではない。同じ意味を読めても元Sourceへの安全な逆写像が不明ならread-only。局所編集成功でも出力Artifactの現行資格、現在の開示資格やCommit結果を自動承認しない。

### 35.2 形式版と意味契約の直交性（版選定はOwner判断を維持）

実`internal/formatversion/formatversion.go`は、Definition formatを中央表現`Document/Element/Attribute`へDecode/Encodeする既存境界で、属性の`Absent/Empty/Value`を区別し、同一版変換identity、要素単位の損失診断、未知版の全体拒否を要求する。これらの**Source形式上の三値**は、SQLの`NULL`／未取得／非開示と同じ分類ではない。異なる意味領域を同じenumへ安易に統合しない。

[#4367](https://github.com/hat47x/tei-platform/issues/4367)の候補C1（Workspace formatVersionを明示更新）/C2（Language Contract pinで新構文を選択）は**未決定**。C3（版未宣言の自動併存）は黙った再解釈のため推奨されない。**本節ではC1/C2を選ばない**。どちらでも必要な互換確認：

| 軸 | 固定・比較内容 | 禁止する推論 |
| --- | --- | --- |
| 形式版 | Workspace formatVersion、内包Semantic header、Codec、旧new構文の受理条件 | Format一致＝同じLanguage実装／新`rule`を旧版で自動解釈 |
| Language pin | Parser契約・実Implementation・選択経路・Toolchainと選択時Source | 先頭で見つかったPlugin／latest Parserが適格 |
| 意味版 | Semantic Role/Operator/Facet/Requirements、NULL/decimal/grain/Time、独立Expected | 同じSource byte/Subject ID/Value＝Meaning同値 |
| Projection版 | Markdown/Graph/Tree/SHO/StudioのReader、lossとinvertibility、原文EOL/BOM | 見た目の一致＝Raw byte一致、非単射View＝更新対象一意 |
| Realization | Target/Generator/Builder、実Artifact/実Input、Host/Provider ABI | 以前のArtifact PASS＝現在のRunner/Grant資格 |

現mainでは`workspace.Load`のmanifest/実読取byteが区別され、`*.tei.md`管理rootの正式Parserは未束縛なら拒否される。SHOでMarkdown草稿を保存できることと、Product意味Sourceとして`*.tei.md`を採用できることは別。旧版のSource→新Codecの変換は、原文損失/Element単位の診断と**明示操作**なしに自動上書きしない。異なる版のSemantic Parse成功を「自動Migrated」と呼ばない。

### 35.3 変更操作は五つの観測点で検収（新State Machine/APIではない）

Studio／AIがSourceまたはDB Rowの編集候補を作るとき、次の**同一Operation Attemptに対する監査・検収ポイント**を既存Operation/SourceIndex/Host/Effect Receiptに参照付けする。これは共通Application APIに巨大な共通Envelopeや新Coordinatorを追加する仕様ではない。

| 時点 | Source変更の場合 | DB/Provider作用の場合 | 観測しない限り昇格できない資格 |
| --- | --- | --- | --- |
| **T0 Preview** | 元Source byte/occurrence/selected Subject/base revision、差分・非対象byte、原糖衣保持 | Native SQL/Rowset/Selection、実Row identity・row-version・Pending Changes、Query/Read Scope | 表示だけで「編集対象が一意」としない |
| **T1 Resolve** | 現Workspace/Manifest/Language pin・対象Occurrence/Source span・Read/Requiredを再解決 | 実Provider/DB alias/Instance、Query/dialect/Bind/Selected Key+Version、Grantの出所 | CandidateのProvider名・UI label・SQL row番号をIdentityにしない |
| **T2 Admit** | Apply直前にbase byte/RevisionのCAS、Meaning/Rule/独立Expected、対象範囲とWritable判定 | Effect直前のCurrent Grant/Scope/Row version/Schema drift/Operation id/再送条件 | T0時点の成功・旧GrantでApplyを許可しない |
| **T3 Commit** | 実際に書いたSource byte/identity・Conflict/no-op zero-write・書込み後Revision、途中失敗 | Providerの実Commit/拒否/部分効果/operation identity、Receiptと不確定状態 | API応答`OK`や通信成功＝持続確定とはしない |
| **T4 Observe** | 独立再読取→Parse/Source Index/Meaningと差分の照合、Studio/SHO/SUIの受領 | 独立Row Read/履歴Provenance/Consumer receipt、実公開Policy/見える範囲 | 旧キャッシュやClient表示だけで実保存・更新完了としない |

Source editとDB Updateの**物理Atomicity/rollback保証はProvider/FSの実能力次第**であり、五時点に区切るだけでは保証されない。部分失敗/response lostなら`effect unknown`を保持し、同じOperation Identityで既存の適格Receipt Lookup/独立Readを優先。無条件Retryを認めない。Source/DB両方を変える操作のall-or-nothingは別の製品要件で、共通のTransaction Managerをここに導入しない。

### 35.4 Source/Subject/RowとAI文脈のIdentityを混同しない

- **Source**：Stable Semantic ID、実Source occurrence（file/span/raw byte）、Workspace manifest Identity、Source Revision、読み込まれたRoot/Plugin/Import closureは別。`source-index`のTrace参照0件やDisplay labelから不存在・対象一意を主張しない。選択Intentは`#4515`のSubject Pinに保持できても**Current Grant・Source/Meaningの現在性**は自動復元しない。
- **Row/DB**：業務membership identity、業務Key、実Physical PK、record-version、query Selection、DB Instance/Aliasを混同しない。実Read後のJOIN/aggregate/virtual Key/partial Rowsetは表示できても安全な逆Update対象とは限らず、User/AIがSQL表示セルを指しただけでDML適用しない。Source意味モデルとDB schema/historyを別記録する。
- **AI/SHO/SUI**：Editor selection/SHO span/SUI Card/Graph描画edgeは**参照意図**であって自動的なSource writable identityではない。Graph同ラベル多重辺、説明文の同文字列別Occurrence、SHO/Markdown OS間CRLF変換を情報損失として検査。Candidateが元Source/Roleを一意に逆写像できなければPreviewとread-onlyで停止。許可を失った場合は既描画DOM/ARIA・AI payload/Tool ContextもCurrent Disclosureと別に再検査する。
- **時間的有効性**：`reference/tei-studio-workbench/responses.ts`の`responseScope`は**UI返信順序**を管理するのみであり、Source/Grant/Policyの変更検知・古いDOM消去・Host authorizationではない。T0〜T4のSource/Grantの更新観測がない場合、即時失効を保証せず`unknown`とする。観測された失効後に旧Replyを新ContentとしてPublishすることは拒否する。

### 35.5 Native SQL・型付きQueryの限定成果と次の接合

[#4496](https://github.com/hat47x/tei-platform/issues/4496)のNative SQL→選択→保留差分→単一Row CAS→実再読取を既存Ownerとして維持。Native SQLはDB方言・NULL・JOIN多重度・order・Provider Read Scopeを含み、TEI標準Query候補への変換を**自動意味同値の保証**としない。複雑集計をNativeに残す選択を許容し、DB clientとしてのStudioの実用性を阻害しない。

現[#4631](https://github.com/hat47x/tei-platform/pull/4631)の積上げDraftは、宣言済みBrowse Queryの`std.datetime`に限る厳格な型検査を既存共通Executor到達前で実行し、欠損/型違い/NULL等を拒否する限定RA-1。PR記載の同一HEAD専用Go Workflow成功は**型付き入力前置検査**の資格であり、隔離PostgreSQLからのactual Row Read、Provider/Native二Consumer、実DB履歴・更新資格、Owner意味受理の成功とは別。今後のS0実Read→S1固定Selection→S2 Pending Delta→S3単一Row CAS/Receipt→S4 Studio/CLI/Harnessの受領は#4496/#2586/#2500/#1174へ渡し、DSL Meta Parser完了をHard Gateにしない。

Migration/Historyでは、**履歴がSource/旧DBに存在することと新Targetに再現したこと**、業務IDとrecord-version、schemaの過去状態/現状態、old Reader/Writer互換を分離。欠けた過去を便宜上生成しない。`History unknown`や`DDL drift`はApply直前に停止。移行は実隔離DBの独立Readerでold/new as-ofを照合できた範囲のみ限定PASSとする。

### 35.6 独立反例（SD-C01〜12：既存OwnerのCaseへ参照対応、新テスト未実行）

| Case | 変更・故障の一因子 | 独立Expected／停止線 |
| --- | --- | --- |
| SD-C01 | 同Source/VersionだがLanguage pin/実Parser A→B、またはOperator ABI変更 | 形式Reader PASSと意味同値・実Parser選択の資格を分離 |
| SD-C02 | 旧Validationと新Rule同ID、どちらか一方だけ旧Version | 重複・Unsupportedを拒否、先勝ち/後勝ち・自動改版なし |
| SD-C03 | Format Codecが`Absent`と`Empty`を潰す、または`NULL`へ無根拠対応 | Element損失診断。Source presenceとDB NULLは別 |
| SD-C04 | 空のno-op Saveや別OriginのLF→CRLF/BOM変換 | byte不変・zero-write、明示Patch以外のformatter side effectを拒否 |
| SD-C05 | Source A→B→Aで同span/IDへ遅着AI Patch | base version/byte/manifest/Pinを照合。旧Intentを暗黙再資格化しない |
| SD-C06 | Graph/SHO/KJ Viewが異なるSource occurrenceを同Labelに潰す | non-injective逆写像の自動Applyはunsupported/read-only |
| SD-C07 | Query Resultの行番号は同じだが実DB alias/PK/row versionが変化 | Preview→Admit時にRow CAS/Resourceを再解決、別対象へretarget禁止 |
| SD-C08 | JOIN/aggregate/partial/hidden Rowsetから一つのセル編集を要求 | 閲覧可と更新可能を分離、完全集合や実Key未確認ではDMLなし |
| SD-C09 | Snapshot一致でもGrant/Schema/Provider ImplementationがT2直前に失効 | Preview PASSからApply許可を推定せず拒否/unknown |
| SD-C10 | Commit後HTTP/IPC Reply紛失、Rowは実更新済みか不明 | 無条件再送せずOperation idとReceipt Lookup/独立Readで再確認 |
| SD-C11 | Source/Meaningは一致、旧履歴Sourceなしで新DB Historyを補填しようとする | 「過去不明」を維持。業務IDと履歴版を分けてas-of確認 |
| SD-C12 | Stage1生成成功後にLanguage/Generator/Builder pinか実N2 dependency変更 | Source/Meaning資格とArtifact/実Consumer再Build資格を独立失効。B2/G4偽PASS拒否 |

既存#4366 F、#4367 C、#2497 MP、#4515 PIN、#4496 WCE、#2514 Q、#2600 DI、#4583 BDを対応させる。新Test Runner・Case SSOT・全体Permission Engineは作らない。未実行/実Consumer未到達のケースは`NOT RUN`/`unknown`。

### 35.7 計画に組み込む優先順・実装判断

| 優先 | 成熟した受入条件の引渡し | Owner |
| --- | --- | --- |
| **P0主線** | non-Plugin Desktop Build B1-B/C→Stage1実Builder→Stage2→Windows実WebView2受領。実Source/Builder/Artifact/Headが一つのAttemptでつながるか | #2831/#4583、既存#1119 Gate。不必要なDSL移行を新Hard Gateにしない |
| **P0並走** | 旧Rule/新RuleのS0独立Expected・Case消費、旧/新FormatとLanguage pinの**版未採用の互換性票**、Sourceの無変更Save zero-writeと失敗原子性 | #4361/#4366/#4367/#4359/#2514。既存#4405/#4607を再利用 |
| **P0並走** | `fx.rate` Role S0でSource/Role/必要Reader/実消費の同一Subject束縛 | #4536/#2620/#988 |
| **P1** | Read-only Subject PinとStudio/SHO/AI/Graph投影のSource/Grant失効、T0〜T4の条件付きPatch/独立再Parse | #4515/#1124/#2497/#2621/#2516。安全な単一Occurrenceが先 |
| **P1** | Typed Query`std.datetime`の既存限定検査を、実Provider Row Read→Key/Version Pin→一行CAS/Receipt/独立Readへ接続 | #4631/#4496/#2586/#2500/#1174。実DB入場は別 |
| **P2** | DSL/Grammar改版×Target Generator×History migration×Incrementalの正しい再資格化・元Reader/Writer互換、SaaS multi-Workspace/DB隔離 | #2600/#4544/#2497/#2892/#4125。Source/DB履歴を捏造しない |

**今回の事実／資格**：Sourceの現Go `formatversion`中央変換・`workspace.Load`実Reader・`responses.ts`表示順序、既存#4366/#4367/#4496の計画、#4631の限定Go実装報告を照合して本Draftを更新する。現Productの新`rule`/`*.tei.md`全体導入、Source Writer/Studio条件付き実Apply、Native SQL→実DB Write/History Migration、Stage2 Desktop、Current Host Grantの新E2Eは本節では**NOT RUN/NOT ADOPTED**。現行契約を超えるMigration/Schema/Execution権限はOwner判断まで付与しない。

<!-- tei-self-description-single-attempt-20261114 -->
## 36. 2026-11-14 自己記述からStudio実受領までの最小実証票と実行配置の独立（計画と限定CI資格）

### 36.1 今回追加する判断：自己記述に必要なのは「定義の再帰」ではなく「選択・実消費の証拠連鎖」

本節は第31〜34節の抽象境界を、[#4536](https://github.com/hat47x/tei-platform/issues/4536) Role S0、[#4359](https://github.com/hat47x/tei-platform/issues/4359)/[#4405](https://github.com/hat47x/tei-platform/pull/4405) DSL第一規則、[#4607](https://github.com/hat47x/tei-platform/pull/4607)/[#4608](https://github.com/hat47x/tei-platform/pull/4608) 独立Expected/実Read Demand、[#4634](https://github.com/hat47x/tei-platform/pull/4634) Build選択の統合PR、[#4583](https://github.com/hat47x/tei-platform/issues/4583) Desktop Self-build、[#4635](https://github.com/hat47x/tei-platform/issues/4635) Studio配置4候補に基づく**実装可能な検収票**へ落とす。

目標は「Grammar/Role/Generatorの定義体が標準データとして保存され、固定した選択を通じて実Interpreter/Generator/Builderに実際に利用されたことを、独立Expectedと同一Source/Attemptで確かめる」。記述できたこと、自分で自分を解釈したこと、自分が自分をビルドしたことを同一の`self-describing`合格にしない。**新しいMeta Parser／統一Receipt Engine／第二Meaning Storeは導入しない。**

### 36.2 具体的な縦断ケース：独立した二つのSemantic対象と一つのStudio Consumerをつなぐ

| Slice | 固定する実対象・入口 | 最初の観測／停止線 | 使用する既存Owner |
| --- | --- | --- | --- |
| **R0 Role** | 実Workspaceの`fx.rate`と選択されたRole宣言 | 実Source byte/manifest/root→Parsed/Anchored→Required/Consumed→Domain Readerを同一Snapshotで照合。候補が自己申告する`required=[]`は証拠にしない | #4536/#2620。旧Role Referenceは再実装しない |
| **R1 Validation** | 既存`sales.order_line.quantity_positive`の旧`validation`表層と新`rule`候補（別版・別Fixtureで検証） | Typed `semantic.Validation`、NULL/欠損・評価結果、Scope、Source位置、Role/Operator契約、暗黙Rule ID禁止を比較。新構文未選択ならProduct Loadでは拒否 | #4359/#4361/#988/#4405 |
| **R2 Evaluation Read** | `validationeval.Source.Number(attribute)`と`RelationKeys(relation,attribute)`を固定Probeへ | 同結果値でも**要求した対象・順序・回数**を確認。Probe呼出し≠実DB行/物理Provider read／short-circuit完全証明 | #4608/#2586/#2514 |
| **R3 Studio Read** | 既存`InspectSubject`、生成Inspector TS、Workbench/Wails WebView2 | Subject/Workspace/Source Revision・実Contract digest・実JS受領・Current Disclosure・新旧返信を別観測。Client描画ID/Subject PinをMeaning IDの証拠にしない | #4572/#4579/#4587/#4515/#1824 |
| **R4 Build Selection** | BuildIntentのSource/Builder/Deps宣言→Resolved Builder→実子process image | 契約/選択/起動前digestが同一checkoutで検査された範囲のみ合格。全Targetとread-time input closureは別 | #4623/#4624/#4634/#4583 |
| **R5 Product Self-build** | 実Non-Plugin StudioのN0 Inspector TS→N1 Go/TS→N2 esbuild→N3 Go/Wails→実Windows WebView2 | 各工程の`actual consumed`と生成Stage1 imageによるStage2起動を同一Attemptに固定し、Stage2 exe自体が受領されて初めてB2/B3。CI Runnerによる別build成功は不可 | #2831/#4583/#2514 |

**接合の正しい順番**：R0とR1/R2を独立にS0/P0で進め、R3は現Desktopの同一HEAD限定成功を保持。R4を既存Build選択に接続してR5の不足を埋める。異なるFixtureのR0とR1や、別PRのR2とR3とR4を**一つの自動受理済み製品Trialと称しない**。共通検収票は既存Source Index/Build Receipt/Scenarioを参照し、観測できない欄を`unknown`とする。

### 36.3 統合PRの最新CIを正確に反映し、次の断線を狭くする

2026-11-14現時点に以下を直接確認した。詳細な固定CI IDはそのPR・既存Ownerの証拠を参照する。

1. **[#4623](https://github.com/hat47x/tei-platform/pull/4623) exact HEAD `f1e219a25976946bcc8a19ef57f5490333ec8a84`**：GitHub ActionsのPR Workflowが**8/8 completed/success**。既存`tei self-build`が明示`-bind-resolved-builder`から`Plan.Builder.ArtifactDigest`と実Child SHAを起動前に照合し、Selected B/Actual Aを副作用起動前に拒否する**Plugin主体の限定結線**に対するCI。
2. **[#4634](https://github.com/hat47x/tei-platform/pull/4634) exact HEAD `653d0d9a1d0125631aec4379550d2bce20364207`**：このTurnで`fetch_commit_workflow_runs`から**8/8 PR Workflow completed/success**を直接確認。Self-build workflow `43548771542`、Repository Health `43548771555`もsuccess。旧PR本文の`統合HEADのCI結果未確認`は照合時点の履歴であり、現在は**同一HEADの限定Go/CI検証資格が成立**。#4624の`BuildIntent宣言→Resolver応答の拒否`と#4623の`Resolved Builder→実Child起動前Pin`の交差影響を同Checkoutで試験した範囲に限定。
3. **[#4608](https://github.com/hat47x/tei-platform/pull/4608) HEAD `6a49b12069c8ca18af76ecc7ad93b9c9a6547811`**：`Number`/`RelationKeys`のRead Demandを観測する一テストファイルの積上げDraft。取得できたWorkflow `43410961578`はIssue-to-Design dogfood 1件のみsuccess。これを新Go Probeテストの全CI PASS、Sibling #4607と同一HEAD、実Provider read合格へ昇格しない。
4. **[#4587](https://github.com/hat47x/tei-platform/pull/4587)**：生成InspectorとWindows WebView2を同一HEADで接続した既存H1/B0/N0の限定CI成功を参照する。[#4634]のBuildIntent→Child Pin成功を合算して**Studio B2/B3**を成立させない。

**次の一辺（B1-B/C）**：正式Non-Plugin Desktop SubjectのSource/Builderを、現在のPlugin主体`BuildIntent`で虚偽のPluginへ化けさせず、既存`Build Execution/ArtifactSet`の単位で**N0→N2の実TS消費、N2→N3のJS embed入力、N3実Go/Wails toolchain**を表現して観測する。選択時digest・OS loader read時byte・Transitive deps/Node module/Go embed/OS tagを同一の検査で証明したことにしない。#4634の限定PASSを保持したまま#4583が実Consumerまで結線する。

### 36.4 Studio配置4案でも同じ自己記述Meaningを使用する

[#4635](https://github.com/hat47x/tei-platform/issues/4635)で明示された四案を使い、**DSL Semantic/Operation契約の変更を必要としないか**を配置比較の第一条件とする。

| 配置 | 配布とプロセス境界 | 自己記述側の受入で変化してはいけない内容 |
| --- | --- | --- |
| **A Native** | Wails UIとGo Serviceが同一Process | Language/Role/Operator固定pin、Workspace/Subject/Result/Failure/Source診断 |
| **B0 Self-spawn** | 同じGUI実行fileからWorker子Processを起動＋local IPC。**一ファイル配布と二Go Runtime/二memory spaceは両立しうる** | 選択Language/実Readerの意味、対象・Resultを維持。Process image/argv/peer/parent-child/停止・更新は新たな実行証拠 |
| **B1 Wails Server HTTP/WS** | 別`-tags server` Wails Server artifactの子Processを起動 | 同じOperation型と失敗分類。Server Host/URL/WSとWeb公開面の権限/秘密開示は別 |
| **B2 Wails Server Pipe/UDS** | 別Server artifact＋Windows Named Pipe／Unix Domain Socket（OS依存） | 同じOperation型。JSがpipeに直接触れる前提なし。Native Bridge→薄いGo IPC Client→Child Serviceで接続 |

A/B0/B1/B2は**配置方式の比較用識別子**であり、#4583の**DesktopセルフビルドB0/B1/B2/B3**とは無関係。同じ記号を進捗欄に混在させない。正式採用は未決で、現実のNative Desktop/Go単一バイナリ配布を基準にする。

同Source・同Operation・同Host方針のread-only`ListWorkspaces/ValidateWorkspace/InspectSubject`について、Semantic結果・Source/Workspace Identity/diagnosticsの同値を比較する。数百byte RPC、数十KB一覧、MB級Graph、長時間Build/進捗を別系列として、p50/p95/p99、起動・初回描画・合計メモリ・CPU・serialize・子Process障害/再接続を**実測**しない限り速度優位を宣言しない。B0のself-spawnは**Wails Server Buildを自動利用したことにならない**。

安全性：子Process peer本人性/endpoint専有/ACL、version handshake、古いReply、Parent crash/orphan、同exe更新競合、Build/DB/AI Effect unknown、未保存Editor Buffer/Pin、Wails Server 15 exported methodsと実Web公開権限の分離。Current Grant再確認前に新Sessionへ旧結果を流さない。remote通信断でlocalへ無断fallback／作用不確定のblind retryをしない。多ユーザー公開・SaaS isolationは#3421/#1043/#4635が別Owner。

### 36.5 優先作業と独立負例（旧Caseを再利用、新Test SSOTなし）

| 優先 | 薄い実装・実証 | 必ず分離する境界 |
| --- | --- | --- |
| **P0 Role** | #4536 S0a/b/c `fx.rate`の実Workspace/Role宣言/Required closure/Domain Reader/独立Expectedを1 Subjectで接続。次に旧`semantic.Validation`を第二Subjectとして比較 | Sourceに見えるRole＝実Reader消費／意味受理ではない。新Ruleを依存条件にしない |
| **P0 DSL** | #4405の`rule`候補、#4607 Expected JSON消費、#4608 Read Demandを**ひとつの積上げHEAD**で試験し、SF-D IDなし/旧新同ID/NULL/partial/fail-closedを検査 | sibling Draft別HEADのCI成功を足さない。Probe要求≠実DB read、完全なSource closureでもない |
| **P0 Build** | #4634の限定統合CI成功を基点に、#4583正式Non-Plugin BuildExecution→Stage1→Stage2→Windows WebView2を接続 | selected/actual child/actual N0 N2 N3/actual consumerを一Attemptで結ぶ。外部CI二回Buildを再帰成功としない |
| **P1 Studio** | #4635 A/B0/B1/B2を同一Fixture/Operationで順に比較し、Native BridgeとHost Disclosureを維持 | 配置比較をP0 Self-buildのHard Gateにしない。HTTP vs Named Pipeは測定後判断 |
| **P2 役割/Generator** | #4544の実Generator選択と自己記述可能なContract/Intentの局所実証、#2600局所再利用、Studio/AI/SHO/SUI限定Projection | 履歴Evidence/Artifact Byte/Owner Meaning/Current Grantを自動昇格しない |

独立反証の最小再利用群：#4536の`SD-S0-01/02/05/06`、#4361の`EX`/`SF`、#4608の誤Number/Relation要求、#4583`BD-03〜06`、#4635のChild crash/peer/version/ACL、#4515の異Workspace/遅着`PIN`、#2514のEvidential monotonicity、#2600のdeleted edge/ABA。1つのCaseに対する`declared / selected / observed demanded / actual consumed / verified / current qualified / consumer received`を一枚で比較し、観測外は`unknown`。新Test Runner/Registry/Authorityは不要。

**状態**：本`35は既存Source・Issue・PR・固定Commitに基づく計画反映。#4634の8/8 successはGitHub Actionsにある**既存CIの直接確認**であり、今回新しいGo/Windowsテストを実行したのではない。#4608のGo Probe実行、#4634の正式main merge、Non-Plugin Studio Builder全工程、Windows Stage2、A/B0/B1/B2通信性能・障害試験、正式Grammar/Role/Owner受理は**NOT RUN / NOT ADOPTED**。

<!-- tei-self-description-product-witness-20261114 -->
## 37. 2026-11-14 自己記述の製品受入：宣言・選抜・実行・実読取・Consumer受領を分ける（横断統合）

### 37.1 追加の判断：自己記述の完成条件は「再帰的に書けること」ではなく「実消費の証拠連鎖」

第31〜35節の境界に、最新の[#4641](https://github.com/hat47x/tei-platform/pull/4641) Generator選抜→実Builder/Artifact/Consumer、[#4634](https://github.com/hat47x/tei-platform/pull/4634) BuildIntent統合、[#4587](https://github.com/hat47x/tei-platform/pull/4587) Desktop H0/H1、[#4637](https://github.com/hat47x/tei-platform/pull/4637) Local REA Evidence実byte読取、[#4638](https://github.com/hat47x/tei-platform/pull/4638) 隔離PGの実Browse Read、[#4636](https://github.com/hat47x/tei-platform/pull/4636) Script Launch事前宣言拒否を接合する。

いずれも「同じデータ構造を自己記述できる／同じ最終Valueを得た」ことをもって実Implementation/Providerの使用資格を自動承認しない。既存[#1163](https://github.com/hat47x/tei-platform/issues/1163) Evidence Packet、[#2514](https://github.com/hat47x/tei-platform/issues/2514) Independent Expected、[#1180](https://github.com/hat47x/tei-platform/issues/1180) Scenario、[#2600](https://github.com/hat47x/tei-platform/issues/2600) Incrementalを参照して、**同一対象・同一Attemptで読める最小の検収票**を定義する。新たな共通Core Schema、Owner承認Engine、Receipt/Proof Store、第二Registry、万能Source Readerは作らない。

| Claim（既存証拠への参照） | 何を実際に記録するか | 代表的な誤昇格 |
| --- | --- | --- |
| **D Declarative** | 原byte、実Source/Manifest/Subject occurrence、選択Language/Role/Generator/Build Contract、独立Required | 宣言だけ存在＝実装・選択・意味受理 |
| **S Selected** | exact Provider/Implementation/Builder/Artifact pin、Resolver応答と宣言の整合、Target/Stage/Role | 選択B＝実Aを暗黙許可／未選択Pluginを代用 |
| **I Invoked** | 実Parser/Reader/Adapter/Generator/Child processのInvoke対象、Attempt/OS/Toolchain/args | 予定Recipe＝実Child/Go Providerを起動した |
| **R Read/Produced** | 実入力Byte/Path/Role、実Query bind・Provider instance、実Graph member・Artifact digest、処理時点/Scope | 同Digest＝同Provenance、Providerの呼出要求＝物理Read |
| **V Verified** | 実Runnerと独立Expected/Negative Mutant、source/consumer/targetごとの判定、Unknown/Skipped | Test caseの宣言＝実行成功、別HEAD CIの合算＝同Trial合格 |
| **C Consumed** | 実CLI/Studio/WebView2/SHO/外部Hostの識別と受領内容/Version/表示またはEffect receipt | Artifactが存在＝Consumer利用、画面描画＝Owner受理／現在Grant |

D/S/I/R/V/Cは**観測の分類軸**であって新しいパイプラインAPIや直列Gateではない。Rが独立に観測できない場合はunknownのまま。Cでの開示・Current Grant・外部Effectの受理はHost/Ownerが別途判定し、D/Vの成功では補えない。Claimごとに実Subject・Source Revision・Scope・Attemptを固定し、一つの共通「完全証明」へまとめない。

### 37.2 同一モデルの論理Graphと物理Build/Read Graphを混同しない

[#4641](https://github.com/hat47x/tei-platform/pull/4641)はGenerator選抜→実Builder→成果物実消費を段階別の資格として整理し、最適化で論理Graphと物理Graphの一対一対応が崩れる場合の反証を既存GCX-01〜08に委譲している。本節の実装上の結論は：

- **論理側**：Subject/Role/Function/Operator/Generator Contract/Required Sourceと、Target要求・Output Roleが同じ意味で何を作るかを規定。Sourceの糖衣/文書投影/三要素分析法の図が異なってもMeaning ownerを増やさない。
- **物理側**：一つの実Builderが複数Outputを出す、一Outputを複数段・最適化・Embed・Link経由で作る、あるLogical memberが消滅する等は、**実行されたTransform/Stage/Manifest**とSource→Output→Consumerの対応で検証。単一の生成物ファイル名とLogical IDの1対1対応を仮定しない。
- **不明な辺**：`BuildInput closure`、Node esbuild selected Source、Go embed/member、OS build tags、実Provider Reader scope、Current Host Grantは互いを代替しない。1つの実Read traceだけで将来短絡条件やNOT EXISTS、Source追加を含むPotential/Required依存の完全性を証明しない。
- **実Targetが作れない時の停止線**：現行`BuildIntent`のPlugin主体Subject制約を回避するため、Studioを偽のPlugin Subjectとして登録しない。既存[#1080](https://github.com/hat47x/tei-platform/issues/1080)/[#1097](https://github.com/hat47x/tei-platform/issues/1097)のBuildExecution/ArtifactSetを経由し、非Plugin Targetの最初の未表現辺から埋める。形式ばかり新設せず既存Contractに薄いTarget Adapterで接続する。

### 37.3 最新Actionsからの実証評価と現在の不足（対象HEADを厳守）

GitHubの該当HEADとWorkflow/Job結果を直接読み、次を確認した。

| 対象・exact HEAD | 直接観測できた事実 | 今回の主張の上限 |
| --- | --- | --- |
| [#4634](https://github.com/hat47x/tei-platform/pull/4634) `7d6a1238e6efbd83fb2bd4cfda9667c181567920` | **8/8 GitHub Workflow completed/success**。Self-build `43554149464`・Health `43554149442`等。前HEAD `653d0d9`の8/8に加え**新Docs HEAD自体**を確認 | Plugin主体のBuildIntent宣言/Resolverと実Child起動前pinの限定統合検証。Studio正式N0～N3/Stage1→Stage2は未実証 |
| [#4587](https://github.com/hat47x/tei-platform/pull/4587) `357345d73b3ab74d67ec8bda675fcd63768e9c48` | **12 Workflow中11 success・1 failure**。Windows WebView2 `43554075677`のJob `generated-desktop-webview2-smoke`はsuccess、`desktop-webview2-smoke`はfailure | 通常Desktop失敗はJob logで`powershell.exe -NoProfile -NonInteractive -Command '[System.Environment]::OSVersion.VersionString'`が**15秒Timeout**（`SMOKE_RESULT=FAIL`）。生成版Job successを旧版＋新H0全面PASSへ昇格させず、通常版故障の再現・切分を残す。正式Stage1→Stage2でもない |
| [#4637](https://github.com/hat47x/tei-platform/pull/4637) `170f9be5907cea4af981582ab4c11c5dafb91e87` | **12/12 Workflow completed/success**。Host allowlistのSelectorと上限4MiB付きLocal REA evidence byte readのGo境界 | Wails Bridge/MCP/Browser/AI Agentへの公開・Current Principal/Grant/Source現在性・開示先/Consumer receiptは未検証 |
| [#4638](https://github.com/hat47x/tei-platform/pull/4638) `0ce3a52f070314741f72462982ff940a1f791a28` | 実使い捨てPostgreSQL専用Workflow `43553849861` **1/1 success**。PR本文は7ケース全PASS・Skip0と報告 | `tools/reference-browse-read`の隔離PG実Readのみ。通常Product `tei` CLI、Standard/Native二Consumer、Host Grant、Owner Meaning受理は未成立 |
| [#4636](https://github.com/hat47x/tei-platform/pull/4636) `da807abc65d560134e9a6aa4c4f7b8d92d3b2fef` | mainへ**merged**。宣言Inputs重複/不正Effect/nil ContextのGo Adapter前拒否に限定 | 宣言の構造上の不正を拒否する能力。公開Entry/意味的pure/Host選択Provider実Call/Result Consumer/Effectは別 |
| [#4641](https://github.com/hat47x/tei-platform/pull/4641) `1abbc17635cc75d9db0e209684d3b91d427d2b38` | Generator契約文書の`8`、GCX-01〜08、論理/物理Graph接合を追加したdocs Draft。照合時点の2 Workflowは`1 success・1 in_progress` | Go/生成・実Builder/Consumerの新試験は未実施。CI未完了を全件PASSとしない |

[#4587](https://github.com/hat47x/tei-platform/pull/4587)のWindowsログで**生成Desktop Jobがsuccess**だったことと**通常Desktop JobがOS情報取得Timeout**でfailureだったことは、同一Workflow内でも別の観測事実。単なる環境推定からWorkflow全体をPASSへ書き換えない。発生したTimeoutを新DSLの意味不整合やGenerator不具合の証拠とすることもできない。

### 37.4 最初に書くべき「製品縦断」試験の採点粒度

従来`35のR0〜R5を実行単位に切り直す。新Test SSOTを立てず、各Ownerへ次の**一連の小差分**だけ渡す。

**Slice A — DSL / Role / Evaluation**（#4536/#4359/#4405/#4607/#4608）
- `fx.rate`の一Source・一SubjectでD（Role宣言・Option anchor）→S（選択Interpreter）→I/R（Domain ReaderとSource Required実読取）→V（独立Expected）を、既存WorkspaceとGo read-onlyで照合。まず有/無Role、同Role別Source、未読込rootの一因子対照。
- 別Fixtureの`quantity_positive`で既存Validationと`rule`候補のTyped Value/ID/Scopeと実`validationeval.Source.Number/RelationKeys`の要求を比較する。#4608のProbeは**requested**であって物理DB実Readではない。旧/新Ruleの製品採否をここで先取りしない。

**Slice B — Generator / Desktop Self-build**（#4544/#4641/#4583/#4634/#4587）
- **次の一辺はN0→N2→N3**。生成`Inspector TS`のoutput role/Producer/Selected esbuild input raw byte/選択member、生成JS→実Go embed/Toolchain/OS tagsを個別に固定し、B（別Builder/Path/Role）や古いbyteを受け取らないことを試験する。
- 次に非Plugin Desktop N1/N2/N3を`BuildExecution/ArtifactSet`の**実Invoker**に接続し、Plugin主体の`BuildIntent`資格と区別。選択Plan/実Child pinは統合#4634の限定実績を再利用するが、Non-Pluginが現行Resolveを使えないときは`unsupported`として第一Gapを記録。
- 同一Attemptで**実Stage1 TEI imageがStage2 Windows Desktopを生成**し、Stage2の実exe/Source・BundleをWindows WebView2で消費するところまで到達した場合だけB2/B3を申請。#4587生成版success＋#4634 builder successを合算しない。

**Slice C — Host Evidence / Provider and AI**（#4503/#4637/#4638/#2586/#4495/#3422）
- #4637の固定Allowlistから実byteをReaderが返すことは`R`の限定証拠。Source/Revision/Principal/Workspace/Grant/Audience/ReceiverとT0〜T4を再確認できないため、Browser・AI Harness・SHOへ当該Readerの公開入口を増やさない。Skillの能力候補はOperator/API契約から生成してよいが**現在の送出許可**はHostで決める。
- #4638の実PostgreSQL限定成功はReference Readerとして保持。**Product `tei` CLI/Function Data Context/Native＋標準二Consumer**から当該Providerが実行されたことはまだ独立に観測が必要。`readDemand`・`realDBRead`・`disclosed Slice`・`endConsumerReceived`を別列にする。
- #4636 mainの入力/Effect宣言事前拒否はScript Launchの静的停止線であり、未宣言Effect＝pure、静的PASS＝Provider実起動・Commitを意味しない。Hostが選んだProviderと実Called Implementation/Receiptを次段で照合する。

### 37.5 統一する反証ではなく、「同じ間違いを各Ownerで見抜く」検収

| 対照ID（既存Case参照） | 異常の一因子 | 失敗すべき誤判定 |
| --- | --- | --- |
| **WIT-A** #4536 SD-S0／#4608 Read Demand | 同値を返す別Attribute/Relation／Role欠落 | Value一致で必要Readerを誤認しない |
| **WIT-B** #4641 GCX／#4634 child pin | Builder Contract B選択、実Child A、またはOutput role/Pathのみ変化 | Caller申告と同File Hashのみで対象選抜を保証しない |
| **WIT-C** #4587 H0／#4583 BD | N0 byteは同じだがN2 select metadata・N3 Embedで別versionを消費 | Stage1/2/Windows全成功を偽装しない |
| **WIT-D** #4637／#4503／#2516 | Host Local Reader PASS、非許可Principal/Workspace/外部Audienceへ送出 | Reader成功を外部開示許可にしない |
| **WIT-E** #4638／#2586 WCE | 実PGのReference Reader PASS、Product Entryは未接続 | 隔離DB Read成功をProduct実Consumer成功にしない |
| **WIT-F** #4636／#4495／#1174 | Effect宣言空、Adapter応答欠損／ACK loss | pure/Commit/Retry資格を自動生成しない |
| **WIT-G** #4635／#3421 | Native→self-spawn/Server移行、古いreply・別Peer・元Host Grant消失 | Transport互換からCurrent AuthZ/Disclosure合格を作らない |
| **WIT-H** #2600 DI／#2595 | A→B→A/非選択Source追加/依存辺削除と実Build再利用 | 同一byte/ResultだけでEvidenceを復活させない |

WIT-A〜Hは既存各Caseの**横断索引**であり、第二のFixture/テストRunnerではない。OwnerはそのCaseの実Scopeとfixed HEAD/Attempt/Build/Consumerを記録する。よく似たSource・別Provider・別PRの結果を結合して新しいPASSを作らない。

### 37.6 Planへの反映と統合判断

**P0最優先**：#4583/#2831で非Plugin Studioの`N0→N2→N3`実Source/Builder/Artifact/Consumer連結を行い、Windows正常/生成Desktop別Jobの合否を保持しつつStage1→Stage2へ進める。#4634統合HEADの実CI成功は再利用するが、正式main merge判断・新ビルド方式採用はOwnerと既存Gateに従う。

**P0並走**：#4536 Role S0（`fx.rate`）と#4359/#4405/#4607/#4608 DSL S0/S1は、限定した実Source・Reader要求・独立Expectedの差を先に検出。正式Grammar拡大、`*.tei.md`製品Reader、AIにSource書戻し権限を付与することはここではしない。

**P1**：#4641 Generator選択/実Builder/物理Graph Witness、#4637 REA Host実Read→適格Disclosure→別Consumer、#4638 reference PG→実Product Operation/標準+Native二Consumer、#4635 Studio配置比較を、対象ごとの実証に限定する。現行のGo単一実行ファイルとNative Studioを標準基準とし、IPC/別Wails Server/SaaSを自己構築の前提Gateにしない。

**実装責務**：この設計文書は既存Ownerへの依存条件を明文化するだけ。Registry/Meta Interpreter/Host/Policy/Oracle/Scenario Engineの新設、Plugin偽装、Ownerレビューの新階層を作らない。

**状態**：第36節はGitHub固定PR/実Actions/一件のWindows失敗Jobログに基づく**docs-only計画追補**。今回新たなGo・Node・Windows・PostgreSQL実行や実Provider統合テストは行っていない。PR #4587の現HEAD 11/12と#4634の現HEAD 8/8、#4637 12/12、#4638専用1/1を**別の証拠範囲**として保持。正式Source Role/Grammar Owner意味受理、MainでのStudio自己構築、外部Host現在開示・効果資格は未完了。

<!-- tei-self-description-current-frozen-20261114 -->
## 38. 2026-11-14 Current／Frozenの版別自己記述とReader実行主体の資格を分離する（横断設計・限定CI観測）

### 38.1 問題設定：同じQuery IDを現在のSourceと歴史Snapshotの双方で使うとき

第31〜36節の「Source byte／Meaning／Selected Implementation／実Reader／実Consumerの分離」に、[#4626](https://github.com/hat47x/tei-platform/pull/4626) RA-0の実際の赤緑を接続する。実`reference/workspace/applications/order-entry/queries/recent_orders.tei`で`input from std.datetime`を追加し、同じ`order_entry.recent_orders`でも**現在のSourceと旧固定Sourceでは意味契約とReader能力が異なる**ことがCIで顕在化した。

- **Current World**：現在選ばれたWorkspace/Manifest/Source/Parser/Operator/Query/Owner資格を、現在のGoldensと独立Expectedで検査する。旧Goldenを現在のQueryへ当てない。`input from std.datetime`がSourceに存在し`:from`が明示宣言済みであることを、実`workspace.Load`・Query shapeと照合する。
- **Frozen Historical World**：歴史受入に必要な当時のRaw Source byte・旧Manifest・旧Parser/Runner/期待値を、**固定commit/Raw SHAに束縛した隔離Workspace**でのみ扱う。現在のSourceを破壊・書戻ししない。旧Runnerが新Grammarを読めないことを、新Grammarの製品不適格と短絡しない。
- **Requalification**：いずれかのWorldで`PASS`しても、他方のSource受理、現Provider実呼出、現在Owner Meaning、Current Host Grant、生成Target/Artifact/Runtime、Stage2に自動昇格させない。Fingerprint一致はA→B→Aの途中変化や元Attemptの実読取・Grant/Generator版を再生しない。

**実観測（#4626）**：初期のSource変更で歴史R0 Revisionと新Query Source Revisionが不一致になり、Old Runnerが新`input`文法を読めず停止。後続HEAD `94876821f784d5fa9f71b15a05ac4156221b2bc8`ではCurrent `reference/world.json`・Current Query/FP期待値と、固定旧commit `e888bb8f…`／旧Query raw SHA`a7c1c4f6…`を使うFrozen Worktreeを分離する変更を追加。歴史Scratchに変更するパスは旧Manifest 2件＋旧Query Source 1件の計3件と固定。**これは現在のProduct Meaning Acceptanceではなく、Current/Frozenを混ぜないための実装修正**。

### 38.2 Source解釈・操作・Receiptの「三つの時点」を明記（新時計Engineは作らない）

| 時点の種類 | 束縛する対象 | 資格と停止線 |
| --- | --- | --- |
| **Definition Epoch** | World/Model/Source Revision、選択Format/Language/Parser/Role/Query、Required Root/Import、独立Expectedの版 | 同じID・Sourceに見える`from`・型付きValueの一致だけではSemantic同値／選択Reader適格性にならない |
| **Execution Attempt** | 実Reader/Builder/Provider/Host Process identity、Source実読取Byte・Input Scope・actual selected deps、Output/Receipt、独立試験 | Fixed historical byteへ後から戻しても、以前のAttemptがそのbyteを消費した証明にはならない。別HEAD/別Attemptの成功を合算しない |
| **Current Consumer/Grant Epoch** | 実Consumer、Principal/Workspace/Worktree、Audience/Disclosure、現在Source・Permission、Response/Effect Receipt | Read成功・Test子ProcessのReceived＝製品Studio/SHO/AI Recipientへ現時点で開示許可されていること、ではない |

これは第34節T0〜T4の**操作中の確認時点**と、第36節D/S/I/R/V/Cの**観測分類**に追加する**履歴と現在の比較キー**であって、新Core型・Clock/Version Registry・Approval Workflowではない。文書／Scenario／既存Evidence Packetが保持する値だけを参照し、観測されないEpochは`unknown`で停止する。

### 38.3 本当に成熟した現HEADの成果／未確定結果を区別

| Subject／PRの実HEAD | 検査した事実 | 合格資格の上限 |
| --- | --- | --- |
| **RA-0 [#4626](https://github.com/hat47x/tei-platform/pull/4626)** `94876821f784d5fa9f71b15a05ac4156221b2bc8` | 18 Workflowの取得時点で**16 success／1 failure／1 in_progress**。失敗はBrowser→JSON API専用Binaryの`tei/internal/database/sqlgen` 2 symbol依存検出（[#1182](https://github.com/hat47x/tei-platform/issues/1182)）。Current/Frozen分離修正後のHealth/Self-build結果も対象HEADで追跡 | RA-0新Query Sourceの試験と歴史Runner再現の限定修正。実APIリンク依存失敗は未解消。RA-1/#4631やRA-2/#4638の旧base成功は新RA-0同一checkout PASSではない |
| **REA Local Reader [#4637](https://github.com/hat47x/tei-platform/pull/4637)** `f9e355d5ccdc96bf4a25ff939fca444a18de1942` | 新Test子OS Processを追加したHEADは取得時点で**12 Workflow中11 success／1 in_progress**。旧HEAD `170f9be5`の12/12 successを引き継がない | Host管理Selector／Workspace-relative Path、最大4MiB Raw byteのLocal Reader、Childが固定Binding→実Read→型付き投影→親へReceiptを返す**テスト専用Process**。製品Studio外部Consumer、Disclosure権限、Current Principal認可は未実証 |
| **Builder統合 [#4634](https://github.com/hat47x/tei-platform/pull/4634)** `7d6a1238e6efbd83fb2bd4cfda9667c181567920` | **8/8 PR Workflow completed/success** | Plugin主体BuildIntent宣言・Resolver・実Child起動前Pinの限定成功。非Plugin N0/N2/N3、実Stage1→Stage2 Windows Wailsは別 |
| **Generator設計 [#4641](https://github.com/hat47x/tei-platform/pull/4641)** `1abbc17635cc75d9db0e209684d3b91d427d2b38` | **2/2 Workflow completed/success**、設計MarkdownのみのDraft | 生成器選抜→Builder→Artifact→Consumerと論理/物理Graphの受入設計。Go実装／正式Target資格は未証明 |
| **Script Launch/Evidence計画 [#4642](https://github.com/hat47x/tei-platform/pull/4642)** `74a60495dfb5fabe6c5b9a700772fb051d1cf705` | **mainへmergedのdocs-only PR**。RA-2 PG、Windows生成/通常Job、署名付きRelease実HTTP検査の別資格を既存Script Launch文書へ反映 | docsのmain採用は新Product Function・実PGの通常CLI二Consumer・Stage2実構築・権限受理と同義ではない |

最新情報を古い`36`の既存HEAD実績で上書きしない。`#4637`旧12/12成功と**新11成功＋1未完了**、`#4626`18中1実失敗、`#4634`現8/8、`#4641`現2/2は**異なるSource/実行資格**として同一段落内でも識別する。時間経過でCIが変わるため、検証者は再度exact HEAD/Run/Attemptを確認する。

### 38.4 言語・Role・AI Skillを履歴の再生権限で混同しない

第35節の`fx.rate` Role S0と既存`semantic.Validation`の二Subjectに、Current/Frozen対応を**必要な場合にだけ**追加する。`*.tei.md` Readerや汎用Meta Interpreterの採用、全Roleに歴史Runnerを要求することはP0の前提ではない。

- **Role/Grammar**：現在Sourceの`Option anchor／Required closure`を実Readerへ束縛する。歴史Writer/Parserで当時Sourceを再解析する場合は旧選択Language pin・旧Script/Operator・raw byteを別に保持。Sourceを旧Formatに戻しただけで現在のSource indexやOwner受理を返さない。
- **Generator/Builder**：ToolchainやBuild Catalogが新しい現在版へ進んだ場合、歴史Artifactを再利用できることと現在Stage2を生成できることを分離。#4641 GCXと#4583 BDを流用し、選択Builderと実Go/Node/esbuild/OS Loaderの読取閉包を段階別に確認する。
- **Agent Skill／Harness**：他チャットの横断成果を**Instruction/Package（何ができると記述したか）／Subject/Information（どのSource/Evidenceへアクセスできるか）／Capability/Authority（当該Operationを今のHostが許可したか）**の三閉包として参照する。Skill/AIの説明は契約候補・操作候補であり、旧Sourceにアクセスできること、現在のSubject全体が利用可能であること、外部API実Call・開示・実Effectが許可済みであることを自認させない。既存[#4503](https://github.com/hat47x/tei-platform/issues/4503)、[#2516](https://github.com/hat47x/tei-platform/issues/2516)、[#3422](https://github.com/hat47x/tei-platform/issues/3422)の権限・送出・受領を再利用し、独自Agent/Skill Runtime、秘密情報のGitHub/外部送信を新設しない。

**REA Processの具体例**：#4637では選択Pathを呼出ユーザーから受けず、Hostだけが固定Selector→相対Path/Raw DigestをBindし、呼出にPath/ReadAtを渡さない。別OSテスト子Processに限定したReceiptでも「実Readerが子で動いた」証拠にはなるが、Product StudioBridge/Web/AI RecipientへのDisclosure・独立Delivery・Current Grantの証拠ではない。現在のAPIへ安易に公開しない。

### 38.5 最小反証セット（既存SSOTの組み合わせ、第二Fixtureなし）

| CW-ID | 比較する一因子差 | Independent Expectedと停止点 | 引渡し |
| --- | --- | --- | --- |
| **CW-01** | 同Query ID、旧Sourceの未宣言`:from` vs現在`input from std.datetime` | Current Readerは現Query shape、Frozen Runnerは旧Raw/旧Manifestのみで再現。旧解析成功→現Meaning受理を推定しない | #4626/#4536/#4359 |
| **CW-02** | Frozen旧Queryのraw byteは同値だがParser/Runner選択が変化 | 読み手のPin不一致は要再資格。固定byteだけで互換性合格としない | #988/#4367/#2514 |
| **CW-03** | Current Sourceは改版済みだが旧Golden/Source FPのみ流用 | 期待値を現在のSourceに強引に合わせるのではなく、Current GoldenとFrozen独立Expectedを別で検証 | #4626/#2600/#1180 |
| **CW-04** | 別の未利用QueryをCurrent Worldへ追加／削除 | 過剰な全体失効と不足したRequired/Negative依存失効を別計測 | #3262/#2600 |
| **CW-05** | Browser API Binaryに想定外`sqlgen` Symbolが混入 | Linked Go実依存監査は失敗を保持。型付きQuery Source変更が原因と確証する前にallowlist緩和／CI SKIPしない | #1182/#4626 |
| **CW-06** | Host Raw EvidenceのA→B→A変更、同Selector/最終Digestに復帰 | Old Read AttemptのReceiptを現在へ復活させず、Root/Revision/Reader Attemptの独立観測が必要 | #4637/#4503/#2514 |
| **CW-07** | REA Child Processに正しいReceiptがあるがCurrent Principal/Recipientは別 | テストProcessの成功はProduct Client受領/Disclosureではない。開示不明なら外部送出0 | #4637/#2516/#1824 |
| **CW-08** | Build selected/actual Childが正しくても、N2実memberかStage1→Stage2が未観測 | #4634の8/8を#4587 Windows成功と合算してSelf-build B2/B3 PASSにしない | #4583/#4641/#2514 |

### 38.6 計画へ落とす実装順序：現行・歴史・実Reader・Consumerを別入口で狭く検証

**P0-A Current/Frozen Reference（#4626/#1182/#2514）**：既存`recent_orders`を一対のSource例として、現行Query Source/Goldensと歴史旧commit/旧Manifest/旧Runnerを実byteで分離。現在のWorkspace/Referenceの手直しを旧Runnerだけのために行わない。まず現HEAD #4626のHealth/Self-build完了結果と`sqlgen`失敗を固定し、真のリンク元と変更影響を特定。対応する最新RAFT検証を独立Expectedへ戻す。P0新Meta version Engineは不要。

**P0-B DSL/Role（#4536/#4405/#4607/#4608）**：`fx.rate`一実SourceでRole/Required/Reader要求、次に既存Validationの第二Subjectを、現在選択されたParser/OperatorとSource Indexへ接続。Rule新構文/Markdown/新Converterの製品採用は独立。歴史Queryの失敗をDSL汎用Parserの再実装要件としない。

**P0-C Desktop（#4583/#2831/#4634/#4587）**：Plugin主体の実Builder Pin統合8/8を利用して、非PluginのN0 TS→N2 esbuild→N3 Wails Embedを`BuildExecution/ArtifactSet`の実Invokeへ接続。Stage1がStage2を構築し、当該Stage2 Windows WebView2で実入力/出力を検証するまでSelf-build L3/G4は未達。既存の通常Desktop PowerShell OSVersion Timeoutと生成Desktop Job成功は別Caseのまま。

**P1-D Evidence Reader（#4637/#4503/#2516）**：Host-owned Selectorによる実byte ReadとTest子Process証拠を、実Product Studio/CLI/SHOの適格Consumerへ渡す前にWorkspace/Principal/Grant/DisclosureとReceiptを照合する。Current/T0〜T4を必ず別確認。Child起動成功を外部Recipient許可にしない。

**P1-E Product Read（#4626→#4631→#4638→#2586）**：RA-0 typed Query Source、RA-1 Executor前Input Guard、RA-2 disposable PG16、RA-3通常Product CLI→Function Context→Standard/Native二Consumerを、各PRが現行Source基線へ積上がるたびに**同一checkout/同一Attempt**で再確認。実Provider Selector/actual Instance、Native SQL方言/NULL/JOIN/Scope/Grantと、CUI/Studio受領は別資格。

**状態**：この節は、他チャット・現行Issue/PR・GitHub Actionsの直接取得に基づく設計Draft更新。新Go/Windows/PG/子Process試験を本節で実施したわけではなく、Source形式、Semantic Role、History復元、AI Skill実行権限、Studio Public Web、Stage2 Self-buildの**正式採用／Owner意味受理は未実施**。主計画は#4583 Desktop P0を維持し、History Runner修正とHost Reader/Skill正規化を新Hard Gateにしない。

<!-- tei-self-description-contribution-coverage-toolchain-20261114 -->
## 39. 2026-11-14 自己記述の実装境界を四軸に収束：意味同一性・段階別Contribution・読取被覆・Toolchain実体（実装引継ぎ）

### 39.1 今回の追加は「さらに一般化すること」ではなく、既存の意味を実用の四軸で取り違えないこと

第31〜37節で列挙したRole/Grammar/意味/Build/Source/Evidenceの軸は維持する。今回の他チャットで実装・検収が進んだ[#4646](https://github.com/hat47x/tei-platform/pull/4646)（段階別Plugin Contribution）、[#4647](https://github.com/hat47x/tei-platform/pull/4647)（Function Data Context実Read Scope）、[#4648](https://github.com/hat47x/tei-platform/pull/4648)（Builder Toolchainの実体観測）、[#4644](https://github.com/hat47x/tei-platform/pull/4644)（修復済RA-0/RA-1/RA-2の一HEAD試験）、[#4559](https://github.com/hat47x/tei-platform/pull/4559)（read-only Function生成）を踏まえ、**DSL自己記述の最初の製品出口**を次の四軸で限定する。

| 軸 | 不変・独立に比較する内容 | 誤った短絡と停止条件 | 実装／判断Owner |
| --- | --- | --- | --- |
| **M：Meaning Identity** | Subject kind/ID、Meaning version/fingerprint、Role/Operator/Type/Facet/Expected、Current/Frozen Sourceの区別 | 同ID／同Result／同byte＝Owner意味受理・旧新互換、とはしない | #2620/#4536/#988/#2514 |
| **P：Phase Contribution** | Editor／Build／Runtime／Testごとに必要なContract/Scope/Target/Provider/Artifactの適格な提供。必要な段階が欠ければ全体適用不可 | 同MeaningだからRuntime用PackageをEditorへ自動流用、候補一覧順でPluginを選抜、とはしない | #951/#4646/#3279/#1251 |
| **R：Read Coverage** | Required Scope、実Provider/Instance、宣言Query/Bind、実読取と観測可能なRLS/Limit/Join/View、partial/unestablished/unknown、Result列由来 | 実結果0行＝完全に不存在、メタ情報で見えた範囲に制限なし＝complete、とはしない | #2586/#4647/#4644/#4496 |
| **T：Toolchain/Target** | Selected Builder、Toolchain実体/版/Path byte、Go/Node/Target OS/タグ、実Build入力・Artifact byte・Stage2 Consumer | 同Planで生成Byte同一／Builder申告matched＝OS exec/trust/G4、とはしない | #2831/#4648/#4583/#2595 |

M/P/R/Tは**既存のSource/Contract/BuildExecution/Evidenceへの検収の観点**であり、共通Coreに`MP RT`なる新Schema、万能Trait Resolver、第二Version Registryを追加しない。正規化やキャッシュは真の要求を満たす最小の層に留める。

### 39.2 Phase Contribution：Meaningの同一性と段階固有の実装要求は独立

[#4646](https://github.com/hat47x/tei-platform/pull/4646) exact HEAD `77463edb17642d6400202d0229f33ec69db23b90`は既存[#951](https://github.com/hat47x/tei-platform/issues/951)のContract Resolverを使い、**一つのMeaning ID/version/fingerprintとEditor／Build／Runtime／Test別Contract**を対応付けるread-only Referenceを追加した。四段階、Runtime Packageだけ版が異なる例、Target/Scope/Provider/Artifact等の不一致12種類、曖昧候補と明示wiring、Editor-only・Runtime欠落、候補列挙順を比較する。**実コードのロード、正式Target同値、権限/Provider実Invoke、永続Lock、Artifact bytesは検証していない**。

自己記述の判断：同一Meaningを扱うEditor用PackageとRuntime用Packageは、**一つの実装配布物を常に共有する必要はない**。一方、必要な段階のContractが揃わなければ「全段階で適格」とはしない。Coreは意味IDと契約解決を保ち、Target/Stage別のRequired Contributionは選択されたProfileの責務とする。Phase別のContractの不整合に対して**部分的な全体成功Plan**を返さない。Atomic Traitの`required/provided/manual-required/fulfillment`と現在Host Grantは別軸。

次の局所検証は、`fx.rate`（Role）と既存`Validation`を同じMeaning/Subjectで別段階へ**無理に配布したことにしない**。まず一つのread-only Function/Rule Subjectを選び、既存Contract解決による`Editor read／Build projection／Runtime evaluation／Test Expected`の四段階に必要なContributionを列挙し、必要段階欠落・別Target/Scope・別Provider・順序並べ替えを反例にする。#4646のReference成功をProduct Studio/Target採用へ昇格しない。

### 39.3 Read Coverage：Native SQLから完全性を捏造しないことを製品契約に含める

[#4647](https://github.com/hat47x/tei-platform/pull/4647) exact HEAD `a962a3f6b13c58d93ad740998e5d94728c53e88d`は既存`internal/functioncontext`へ、PostgreSQL Readの観測可能な制約を**保守的に`partial`へ落とす**限定実装を追加。RLS、LIMIT、宣言と違うRelation、View/partition/foreign table/function scanを観測できた場合に非完全性を返す。SQLの述語・STABLE関数などは不可視になり得るため、`complete`や`EmptyJustified=true`を出さず、`unestablished/open`として保持する。独立対照では表3行・関数適用後2行となる隠れた述語を再現し、誤った完全性への昇格を拒否。7 Mutantが検出されるという同PRの試験報告あり。

**受入の変化**：DSL/Roleの`Required Source`は、Function Data Contextの実読取結果だけから逆算して確定させない。`requested`／`observed restrictions`／`returned rows`／`disclosed slice`／`independently complete`を分離し、未知は`unestablished`に残す。NULLの`stored`起源や`entity`粒度、列単位の出所・OID/attnumは同PRで**W4未実装**。SQL`sql.Tx`では列出所を得られず、必要ならProvider固有のpgx接続境界で局所実証を行う。W4を全Source型・全DB ProviderのHard Gateにしない。

現HEADのActions **7/7 completed/success**を確認したが、実製品CLI／外部Host/Grant／構造化Queryでの完全被覆は別。実PG Reference Reader[#4644](https://github.com/hat47x/tei-platform/pull/4644)の7ケースや`#4608`の要求Probeを#4647の完全性証拠へ無条件に合算しない。

### 39.4 Toolchain：同じBuild Planから異なるArtifact bytesが生まれる実例を正本へ反映

[#4648](https://github.com/hat47x/tei-platform/pull/4648) exact HEAD `64fb0f9abdf1a8cf947a844a973eb6d48685623f`で、**同一Plan／Go1.26.11とGo1.27.0**の2条件は双方`l2b.claimed=true`・`toolchainObservation.state=matched`を返したが、**生成ArtifactのSHA-256は別**（PR実測は`b3b54142…`／`fc29a6d0…`）。現行Build PlanはGo Toolchainを正式に指名しておらず、今回の追加は実体観測・報告のみ。GitHub Actionsは同HEADで**8/8 completed/success**。ただしSeed Trust／OS Loader実行したPath Byte／Target別完全閉包／Stage2再入／G4は実証していない。

**Ownerに残す選択**：
- Toolchain pinをBuild Planの入力にするか、選択Builder契約／Build Executionの検証・Qualification専用に置くか。
- `go version`文字列だけか、実`go` executable digest・GOROOT distribution・依存の再現単位まで固定するか。実`go` Path byteの読取一致だけで、OS execやSeed Trustを証明したことにはしない。
- Cacheにおける`World Fingerprint`・`Model Revision`・`Artifact Revision`と、`Selected Toolchain`の版差をどこで失効させるか。**Artifact再利用／Build省略／Evidence再利用を別判定**し、同Plan異Artifactを誤ってCASに上書きしない。

この判断をする前に`#4648`の`matched`を「Goの意味や最適化が同じ」「別版コンパイラの成果物が同byte」とは呼ばない。元PlanがToolchain非指定である以上、両版成功は現行仕様の説明であり、仕様をPin型へ改定した後の試験結果ではない。性能（Toolchain Source全部のdigest再読取コスト）と出荷配布のGo単一バイナリ標準も独立評価する。

### 39.5 一度だけ行うべきQuery受入の積上げ：RA-0/1/2 Reference成功とProduct生成のずれ

[#4644](https://github.com/hat47x/tei-platform/pull/4644) exact HEAD `b5c23044ebaf8671d73a140da92fc9c4d74edff4`では、修正済み[#4626](https://github.com/hat47x/tei-platform/pull/4626) RA-0（`input from std.datetime`／現行SourceとFrozen旧Runnerを分離）＋[#4631](https://github.com/hat47x/tei-platform/pull/4631) RA-1（Executor前Typed Input拒否）＋[#4638](https://github.com/hat47x/tei-platform/pull/4638) RA-2（隔離PG16親＋7 subtest非SKIP）が、**同一積上げHEADの専用Workflow 3/3 completed/success**。これはReferenceと限定Goの実証であり、RA-0単独HEAD `9eecf2b9`の**18中17 success・1 failure**（API Binary`sqlgen`依存閉包検査[#1182](https://github.com/hat47x/tei-platform/issues/1182)）を消さない。RA-0のHistory/Health成功もCurrent Meaning Owner受理にはならない。

さらに[#4559](https://github.com/hat47x/tei-platform/pull/4559)は`order_entry.recent_orders`をSource→read-only Go Function Binaryへ生成し、使い捨てPGとの限定比較を行っているが、**そのPRが基にするSourceでは`input from`が未宣言**であり、SQL`:from`を暫定Source-of-Truthにしている旨を記録している。`#4644`は**明示宣言済み`from`を含む別Source改訂**。同じFunction ID/SQLのように見えても、**両ブランチの生成器・Query型・Source Revisionをそのまま一つのAccepted Functionと呼ばない**。

**次のProduct Exit（#2586/#1121）**：現Source/Format/Typed Queryを先に一つへ揃える→`actual selected Query/Provider/DB Instance + typed :from`を固定→Sourceから生成したread-only Functionの実Builder/Artifact/Entryを選択→通常`tei` CLI/共通Application APIから呼び出す→Provider実Row ReadとFunction Data ContextのScope（partial/unestablished）、独立Expected、Standard/Native異質二Consumerの受領を別々に証拠化。`#4559`の古い`undeclared` fallbackを正式Productへ流用せず、Ownerによる意味/公開Entry/Host Grantを独立判定。作り直すのは必要な接合点だけで、DSL/Query Engine/Providerを複製しない。

### 39.6 「何を次に実装するか」をOwnerごとに一枚へ収束

| 優先/既存Owner | 最小の具体的な次タスク | 独立Expected／停止条件 |
| --- | --- | --- |
| **P0 #2831/#4583** | Non-Plugin DesktopのN0/N1/N2/N3を実BuildExecution/ArtifactSetで結線。Go/Node/esbuild/Embed/Toolchainの実消費と実Stage1→Stage2→Windows WebView2を一Attemptで取得 | #4634 Builder選抜8/8、#4648 Toolchain実体8/8は別証拠。#4587旧HEADの通常Windows失敗は履歴として残し、新HEAD `9b58533d`の全12 Workflow successをB0限定の実績として区別する。正式Stage1→Stage2はなお未検証 |
| **P0 #4536/#4359/#988** | `fx.rate` Role S0に既存Validation第二Subjectを追加し、Schema/Parser/Reader要求/独立Expectedの「実適用可能なContribution」を#4646 Referenceと照合 | Required/readDemandの一致は物理Provider read・Full Meaning/Grant合格でない。新`rule`/`*.tei.md`の正式採用を先に要求しない |
| **P1 #951/#3279/#4646** | Meaning ID/Version/FPとEditor/Build/Runtime/Testの必要Contributionを小Product Subjectで受理前検査。Runtime-only Package版違いを含む12反例 | 未要求Stageの欠落で全体失敗にしない一方、要求Stage不足はFail closed。Layout/Package版＝Meaning版としない |
| **P1 #2586/#4647/#4496** | Function Contextの`partial/unestablished`を実ReadからConsumerへ運び、W4列出所を必要な1 Providerで確認する。RA-2P/RA-3 Product入口を別PRへ | 0行/SQL Plan/RLS監査だけで`complete/EmptyJustified`、stored NULL、行粒度Entityを認定しない |
| **P1 #1121/#4559/#4644** | `recent_orders`の新型付きSourceを生成read-only Functionへ再接合し、実Provider/二Consumer/独立Expectedを照合 | 旧`#4559`の未宣言`:from` fallback・別Source Revisionを現在受理Sourceと混ぜない |
| **P2 #2600/#2595/#2831** | 同Plan異Toolchain、Toolchain更新A→B→A、選択Builder実体差、別Artifact生成をFull/Incremental/Stage2の試験へ反映 | Toolchain観測`matched`や同Source ByteをBuild省略・Evidence復活・Seed Trust/G4へ昇格させない |

**次に新しい設計節を増やす前の受入ライン**：上記P0のいずれか一つについて、既存Ownerの実Source/実コード/独立Expectedを同一HEAD・同一Attemptで試験し、`declared / selected / actual invoked / actual read-or-produced / verified / consumer received`のうち本当に観測された欄だけをPASSにする。Source Writer、Role Interpreter、Builder、Provider/Host/Registryを新設して迂回しない。独立Expectedの第二正本、新Hard Gateを作らない。

**状態**：本節は既存PRの変更内容と該当exact HEADのActions結果を照合した**docs-only統合・実装引継ぎ**。#4646 3/3、#4647 7/7、#4648 8/8、#4644 3/3の各限定CI成功を同一製品E2Eに合算しない。正式Studio Stage1→Stage2、Source新Rule/Markdown正式Reader、W4実Consumer、RA-2P/RA-3 Product CLI二Consumer、Owner意味受理、新Go/PG/Windows実行は本更新で**NOT RUN / NOT ACCEPTED**。

<!-- tei-self-description-four-layer-compiler-diagnostic-20261114 -->
## 40. 2026-11-14 自己記述DSLの第一製品診断：四層の拒否位置と独立Expectedを実装する（設計・実装引渡し）

### 40.1 P0の次の小実装：同一の不正Capabilityを「読めた／意味を決めた／起動を止めた／公開可能」に混同しない

第31〜38節の自己記述契約に、[#4652](https://github.com/hat47x/tei-platform/pull/4652)のDSL診断／Capability実行前拒否の四層、[#4653](https://github.com/hat47x/tei-platform/pull/4653)の実Consumerまでの横断検収を接続する。対象は、**既存Grammarで表現できる一つのCapability宣言**。新`rule`構文、`*.tei.md` Reader、第二のParser/Validatorを先に正式採用しない。

現行mainの`internal/semantic/capabilityinvoke/invocation.go`には`validateInvocationDeclaration`があり、`validateResultContract`の結果整合、およびInput名・型参照・Effect宣言の構造を、`adapter.Invoke`へ進む前に検査する。この実装には、実Sourceのbyte span、全Workspace参照解決、現在Host Grant、公開Entry、Provider真正性、値／NULL／業務意味受理は含まれない。この**できることとできないこと**をStudio/CLI/AIへ同じ診断として投影する。

| 層 | 正本・実行入口 | 受入条件 | 意図的に断定しないこと |
| --- | --- | --- | --- |
| **L1 Authoring Syntax** | 現Source byte→既存Language Plugin/Parser→既存Anchor/diagnostic | Sourceに実在する異常の位置を示す。byte span・Occurrence・Workspace Revisionを保持。未読込Source、旧Frozen Source、`*.tei.md`未結線はfail-closed | Syntax Parse成功＝Semantic受理。SourceのないRuntime errorへ偽のspanを合成 |
| **L2 Workspace Semantic** | 実`workspace.Load`/`validation.Validate`、Canonical identity/TypeRef/Relation/Source Index | ID重複、未解決TypeRef、異なるScopeや正本の不足を、当時のWorkspace実入力と照合。局所の無関係Subjectは閲覧継続 | 全Workspace valid＝実Entry公開、Required完全取得、Current Grant |
| **L3 Pre-dispatch** | main`capabilityinvoke.Invoke`の宣言静的検査→実Adapter | Input名重複、Input名／型欠落、Output/Outcome両立不備、Effect形式不正、nil contextを**実Adapter call=0**で停止。すでに検出する既存Validatorがある場合は重複実装しない | 宣言`Effect=nil`を業務pureと認定。実Adapter呼出結果やProvider／DBの意味適合 |
| **L4 Product Admission** | 公開Entry/Host選択Provider/現在Principal・Grant・Target・Source Revision | 通常製品入口から選択された実Providerへ呼び出す前に、現在のHost/Source/Artifact/Scopeを再評価。未公開／失効／選択B→実Aならcall0 | L1〜L3 PASSが公開許可やEffect/Commit Receipt。通常CLIや外部Consumerが実受領したという推論 |

**責務の接合**：L1/L2は[#4359](https://github.com/hat47x/tei-platform/issues/4359)/[#4361](https://github.com/hat47x/tei-platform/issues/4361)、L3はmain統合済み[#4629](https://github.com/hat47x/tei-platform/pull/4629)/[#4636](https://github.com/hat47x/tei-platform/pull/4636)と[#4495](https://github.com/hat47x/tei-platform/issues/4495)、L4は[#3422](https://github.com/hat47x/tei-platform/issues/3422)/#4495/#1174が所有。四層の結果を取りまとめるための独自判定器は作らない。Source診断とGo Runtime errorは対象/Source Revision/Scopeが合うときだけ連結し、後者にない位置を創作しない。

### 40.2 既存の反証を「誤りの起きる場所」で分割する

新しい独立Expectedの第二正本を作らず、[#4652](https://github.com/hat47x/tei-platform/pull/4652)のACD-01〜08、#2514の較正、#1180のScenarioへ参照を追加する。実装最初のケースは、既存Sourceで到達できる**ACD-02／ACD-03**を優先する。

| 組み合わせ | 意図した一因子差 | 必ず確認すること |
| --- | --- | --- |
| **ACD-02/03** | 宣言内Input名重複、ResultのOutput/Outcome不正併記 | L1で実spanを保持できる範囲とL2の既存検出Fit/Gapを実測。L3では宣言不備→`adapter.Invoke` call0を実試験し、他層のPASSを代用しない |
| **ACD-05** | 形式は正しいがTypeRef参照先がない | L1 Parse成功は許すがL2で要素と解決対象の不一致を検出。L3の単純な`type != ""`のみで意味的TypeRef存在を捏造しない |
| **ACD-06** | Effect欄が省略された | 静的に不正でないことと、業務pure/Runtime無作用/Host side effect未発生を分ける。業務pureの推定なし |
| **ACD-07/08** | SourceにないCanonical ID、改版前後で同IDの別内容 | L1は存在しないSource spanを捏造しない。Current/Frozen/Revisionsを固定し、旧診断や旧Pinned Subjectが現在の別Meaningへ自動移行しない |
| **Owner／Host対照** | 上記の構造的な宣言は正常だが、Entry未公開または現在Grant失効 | L1/L2/L3がPASSでもL4は拒否し、Providerの実呼出しは0。`pure`と認可を混同しない |

**実証可能な最小PR**：同じSource byte/Subjectについて`ParseSelected/Workspace Validate/Capability Pre-dispatch`が既存APIでどう扱われるかを、既存Go testに局所追加して比較する。既存L1/L2診断がない場合はその空欄を`unsupported/unbound`と記録し、Runtime文字列へ架空のspan/diagnostic codeを付け足さない。新規Grammar採用やSource Writerを先行条件にしない。

### 40.3 実RunnerでExpectedが誤っていた事例を使い、独立検証を強くする

[#4637](https://github.com/hat47x/tei-platform/pull/4637)のHost-owned REA実Readerでは、**同じProducer Evidence IDを二つのRootで扱う試験**が追加された。初回Expectedは「Root Bの改変Rawに合わせてBinding digestを変更すれば正常Candidateを返せる」だったが、実Go testは`OPERATION_FAILED`を返した。**Raw digestが合っても、正規Producer deliveryが証明されない**ためであり、Provider/Reader側の制約を緩めてExpectedへ合わせるのは誤り。修正版ではA/Bの両Rootを未改変同一Rawで正常化し、その後の改変Bは負例へ分離。BのDigestを更新しても改変後のProducer deliveryを受理しない。

この事例をDSL自己記述にも適用する：**テストから独立したExpected**は「固定JSONが存在する」ことと同義ではない。前提に反するExpectedは修正し、その一因子差と観測を履歴として残す。Source/byte一致、Producer/Parser申告、Role名、Candidate generated resultを**それ自身の正解**として自己採点しない。READ→Project→Deliverの各境界に独立根拠がなければUnknown/Unacceptedを維持。L1のByte spanとL3のAdapter call0も、それぞれの独立対照を持つ。

新REA実Readerの**現PR HEAD`9b8f5ca9b50d34040cfa0682f026f4fc29090f01`**では、Actions取得時点で**12 Workflowすべてcompleted/success（旧・中間判定11件成功／Health実行中は履歴として区別）**。別Process Readerや二Rootテストの限定成功も、Studio/Wails/MCP/AI Recipientへの正式配送・Current Principal/Grant/Disclosureを証明しない。#4503/#2516への既存引渡しを維持する。

### 40.4 最新Studio CIの回復を正確に反映し、P0を前へ進める

[#4587](https://github.com/hat47x/tei-platform/pull/4587)の旧HEAD`357345d7`では、Windows通常Desktop JobでOS情報取得のPowerShellが15秒Timeoutとなったが、生成Desktop Jobは成功していた。新HEAD`9b58533df8764a74c578d478d8c6ee92a382642e`はOS版記録だけをPython`sys.getwindowsversion()`へ変更して外部PowerShell実行への依存を除いた。対象[Windows Workflow #43572235468](https://github.com/hat47x/tei-platform/actions/runs/43572235468)の**generated-desktop-webview2-smokeとdesktop-webview2-smokeの両Jobがcompleted/success**であることを直接確認した。新HEAD全12 Workflowは、その後**12/12 completed/success**まで直接再確認できた。途中の9件成功・3件実行中は履歴上の中間値であり、最新の判定ではない。

この局所修正でWindows**B0/H0/H1のCI再検証範囲**は改善したが、Non-Plugin`BuildExecution/ArtifactSet`で`N0→N2→N3`を実起動し、**Stage1 TEI image自身がStage2 Studio Windows exeをBuild→そのStage2 exeでWebView2を受領**するP0/B2/B3は別。テスト前のOS診断プロセスが不要な単なるPreflightにならないことは製品品質上の改善だが、実Builder/Artifact/Toolchainの完全閉包に読み替えない。

### 40.5 次の着手・担当・計画への組込み

| 優先／Owner | 最小の着手単位 | 合格と不合格の境界 |
| --- | --- | --- |
| **P0 #4359/#4361＋#4495** | 旧Capability SourceのACD-02/03についてL1/2/3の既存Go入口を実比較。現mainの`validateInvocationDeclaration`とSource/Workspace Validatorを再利用 | Runtime`adapter.Invoke`を0で停止するが、Sourceにないspanを作らない。L4公開・Grant/実Providerを別試験 |
| **P0 #2831/#4583** | #4587新Windows両Job successを基線に、選択されたNon-Plugin Build Intentの`N0→N2→N3`と実Stage1→Stage2を結線 | 同HEADの実Stage2 Windows WebView2到達だけをB2/B3申請。新Windows両Job successをStage2自己構築とは呼ばない |
| **P1 #2514/#1180** | ACD-01〜08を既存Expected/Scenarioへ接続し、改版・Source不在・TypeRef未解決・Effect未宣言の一因子反証を保持 | Invalid Expectedに合わせてReader/Interpreterを弱めない。旧Source/未取得をCurrent Positiveへ昇格させない |
| **P1 #4503/#2516** | REA二Root・子Processの実ReadからCurrent Principal/Workspace/Recipient別Disclosureまで段階検証 | Test Child ReceiptとProducer Delivery/Current Grant/外部Consumer受領を別に評価し、許可前のWeb/MCP公開は行わない |
| **P1 #4653/#2620** | 新Consumer横断計画を**正本追加ではなく既存Ownerへ戻す参照索引**として利用し、Definition/Binding/Authority/Execution/Acquisition/Delivery/Acceptanceの各既存Receipt位置を対応付ける | 専用Universal Validator/新Proof Engineを作らない。どこが未観測か分かるようにする |

**本節で成熟した判断**：DSLの自己記述を「定義できた」「文法的に正しい」「実行宣言を静的に検査できた」「実Providerを呼べた」「権限を得た」に分け、それぞれ既存の実コードと独立Expectedを使う。新しく実装する必要があるのは**同一Subjectと同一RevisionでのL1/L2/L3比較という小さな接続試験**であり、新Meta Schemaや「すべてを知る」診断器ではない。

**状態**：`39はGitHub現mainの実コード、Draft #4652/#4653、既存Actions・REA反例に基づく**Docs-only追加**。今回新しいGo/Windows/E2E・Parser/Studio/Child testを実行したものではない。#4587新HEAD両Windows Job成功も全Workflow/正式Stage2/Owner Meaning受理を証明しない。

<!-- tei-self-description-schema-view-20261114 -->
## 41. 2026-11-14 外部の宣言を読みやすく投影し、安全に編集するための実装順

### 41.1 まず実現するのは、構文の自由化よりも「理解できる表示」

第38・39節では、意味の同一性、段階ごとの実装、実読取範囲、Toolchain、構文診断と実行直前の拒否を切り分けた。次に必要なのは、それらの情報をドメイン専門家が**普段は簡潔に読み、必要なときに根拠へ戻れる**形にすることだ。

[協業UX Draft #4605 第32節](https://github.com/hat47x/tei-platform/pull/4605)と、[宣言的構文 #4359](https://github.com/hat47x/tei-platform/issues/4359)、[Language Plugin #1018](https://github.com/hat47x/tei-platform/issues/1018)、[投影と逆写像 #2497](https://github.com/hat47x/tei-platform/issues/2497)を共通の実装線に置く。最初の対象は**公開した固定JSON Schema一つ**とし、独自DSLの文法を何でも定義できる仕組みを先に作らない。

利用者が最初に見るものは、業務上の名前、属性、参照先、必須条件、制約の概要でよい。`$defs`／`$ref`の解決経路、原文位置、型・制約・Operatorの細部、版と実装の選択、未解決の理由は、Inspector・ツールチップ・ジャンプ・診断から必要に応じて確認できるようにする。この「情報を減らす」操作は**画面上の折り畳み**であり、元のSourceや意味情報を削除することではない。

### 41.2 意味の特性は、構文と表示形式の中間に置く

新しい大きな共通Schemaではなく、TEIが持つ標準データ構造・Role・Facet・型付き参照の上に、必要な**意味特性**を明示的に束縛する。

| 観点 | 区別して保存・検査するもの | 誤った実装 |
| --- | --- | --- |
| **識別** | Source occurrence、Qualified Subject、意味ID、Graph node/edge ID、表示ラベル | 同名属性や画面上の座標を同じ実体とみなす |
| **関係** | 参照方向、多重性、所有関係、関係自体の属性、適用条件 | Graphの見た目だけで意味上の関係を決める |
| **順序・集合** | 意味上の順序、表示の並び、重複許容、単位・精度、NULL／欠落／未知 | 並べ替えや表示上の省略で意味まで変更する |
| **制約** | 型、必須性、選択肢、条件付き制約、適用Scope、制約を検査した実Operator | 外部Schemaを構文解析しただけで制約が成立したと認める |
| **投影** | 元の要素・参照先、変換の損失、曖昧性、逆写像の可否、Source版 | 図に出せなかった情報を無かったことにする |

JSON Schemaでは特に、`$ref`、`$defs`、`oneOf`／`anyOf`、配列、追加Keyword、未知の拡張を、**読み取れたもの／未解決のもの／TEIの標準構造に写せないもの**として分ける。`oneOf`は「複数の候補を自由に選べる」という表示へ単純化しない。外部参照の所在や権限が不明なら、勝手に取りに行ったり意味を補ったりしない。

ここでいう意味特性は、外部仕様をTEIの業務モデルへ無断変換する許可ではない。構文木から得た候補と、検証・受理済みの意味は引き続き区別する。Graph・Tree・Table・属性カードは**同じ対象についての別の見方**であり、それぞれ第二のSource正本にならない。

### 41.3 読取・説明・入力支援・編集・実行を別の達成段階にする

最初から双方向編集や新言語の実行を一括で実装しない。既存Workbench、Reader、Validator、Source Writer、Operationと権限の境界を利用する。

| 段階 | まず確認すること | 許されない近道 |
| --- | --- | --- |
| **V0 原文を読む** | 選択したReader/版、Raw byte、対象のOccurrenceと原文位置が実在すること | Parser成功を意味受理とする |
| **V1 分かりやすく説明する** | Tree・属性表・制約一覧・Inspectorが同じ対象へ戻ること。非対応や未取得を区別できること | 見えない`$ref`や制約を無音で捨てる |
| **V2 入力を支援する** | 既存L1構文診断とL2 Workspace検証の根拠が原文位置へ戻ること | Runtime側に存在しないSource位置を作る |
| **V3 一項目だけ編集する** | 逆写像が一意な項目に限り、対象ID/元Revision/原文差分を固定してPreview→条件付きWriter→独立再Read | 非可逆Graphや古いAI案から自動保存する |
| **V4 実際に使用する** | 選択された実装、実Consumer、現在Grant、Provider/Effect Receiptを用途別に検証 | 表示・診断・保存成功から実行許可を推定する |

V0～V2の閲覧は**最初の小実証**。V3は#2497/#1124のSource Writer契約が既に対応できるときだけ一項目で試す。V4は#39のL3/L4および既存Host/Provider側の資格であり、JSON Schema表示実装からは自動で開かない。原文のコメント、改行、キー順、未知Keywordなどは、特定のWriterが安全に保持できると確認できるまで、原文を自動整形しない。

人間とAIには同じ「対象・元版・差分・検証結果」を見せる。AIの提案は候補であって承認済みの変更ではない。Editorの未保存内容と保存済Source、表示中の対象とPinned対象を区別し、背景の再検査によって選択や未保存編集を壊さない。最初の画面結線は[#4605](https://github.com/hat47x/tei-platform/pull/4605)の既存**UX-PIN／UX-DIAG／UX-READ**へ割り当て、新たな第四のUX実行票を作らない。

### 41.4 構文自体の自己記述は、有限なBootstrapから段階的に行う

TEI DSLの構文定義を通常のデータとして保持し、図や入力支援もそこから生成する構想は維持する。ただし、初めてその定義を読むParserまで自己記述へ置き換えると起動根拠が循環する。

初期段階では**既存の有限Seed Readerと、明示選択されたLanguage Plugin／Parser実装**を使う。採用候補の文法記述を固定Parserで読み、構文候補→標準データ構造→意味の型・Role・Operatorへ落とし、旧版Readerと独立Expectedで検査する。記述されたGrammarが自身を承認したことにはしない。ANTLR／Tree-sitter／PEGなどは、Source位置の精度、エラー回復、決定性、版Pin、対象Target、編集との相性を比較してから選択する。

「自由な構文」を最終目標とする一方、**最初の製品成果は、外部Schemaを安全に読むこと**とする。業務DSLに`struct`／`func`やGo型引数を表出させない。入力順は業務専門家の認知順を優先し、Markdown/Graphviz/関数チェーンは、標準構造の糖衣表現として元形式を保つ。

### 41.5 独立Expectedに渡す、最初の八つの反例

下表は[#4605 第32節](https://github.com/hat47x/tei-platform/pull/4605)のSX-01〜08を、自己記述の既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ**重複させず参照するための実装順**である。新Test SSOTやRenderer Registryは作らない。

| 受入対象 | 最初の一因子差 | 正しく停止・表示すべきこと |
| --- | --- | --- |
| SX-01／V0-V1 | `$ref`／`oneOf`／未知Keyword | 読めた範囲だけを説明し、未知要素・損失を明示する |
| SX-02／V1 | 同名属性が別Occurrenceに存在する | 表示名だけでは編集対象を確定しない |
| SX-03／V2 | Parserは通ったが参照先Role・Type・Operatorが未解決 | 構文成功と意味検証の未達を別に診断する |
| SX-04／V3 | 元SourceがR0→R1→R0、あるいはDirty Bufferと競合 | 古い位置情報から自動Patchせず再確認する |
| SX-05／V1 | Source一部が未取得・非開示 | その存在・件数を不適切に漏らさず、取得範囲を不明として扱う |
| SX-06／V0 | 同じLanguage名で別Parser版・未選択の自己記述 | 明示Pinなしに実装を選ばず、起動循環を拒否する |
| SX-07／V1 | 同じSourceをCLIとStudioで別表示 | 対象ID、制約、既知の損失を照合し、画面の成功をRuntime成功へ流用しない |
| SX-08／V2 | Background診断・後着AI応答・Focus移動 | 現行選択とPin・未保存編集を変更しない |

最初の実装Caseは**固定JSON Schema一件のV0/V1と、Source位置に戻るV2の診断一件**までとする。逆編集V3と任意文法V4は後続。数学・統計・WBS/PERT・科学Toolkit（[Draft #4663](https://github.com/hat47x/tei-platform/pull/4663)）にも投影の仕組みは再利用し得るが、数値計算の正しさや単位・精度・時系列等の適合は各専門Operatorの独立試験が必要であり、Rendererの合格を領域計算の受理にしない。

### 41.6 最新の実装状況を踏まえた、次の変更単位

**第一優先：Studio Desktopの自己構築（#4583/#2831）**。[#4587](https://github.com/hat47x/tei-platform/pull/4587) 現HEAD `9b58533df8764a74c578d478d8c6ee92a382642e`は**12/12 GitHub Workflow completed/success**。通常・生成のWindows WebView2両Jobも同じHEADで成功した。ただしこれは**生成された部品・Desktop B0の実利用**まで。Non-Plugin BuildExecutionでN0→N2→N3を実駆動したことや、Stage1がStage2 Windows Studioを生成し、そのStage2を起動したB2/B3は未実証である。#4634のBuilder pin成功や#4648のToolchain観測を合算しない。

**並行する最小診断：CapabilityのSourceから実拒否まで（#4359/#4495）**。[#4654](https://github.com/hat47x/tei-platform/pull/4654)現HEAD `a7a9aec8a2450be1f3fa9bd2c4aa19826c01fa98`は**7/8 Workflow success・Repository Health failure**。実ログでは`internal/syntax/capability_authoring_preflight_boundary_test.go`の`gofmt`差分が失敗要因で、Goの全テスト自体はPASS。[#4655](https://github.com/hat47x/tei-platform/pull/4655)現HEAD `3261d172a573ba95e56236291b5a572fd7dbc9e4`も**7/8 success・Health failure**。実ログでは`internal/validation/capability_input_parity_test.go`の`gofmt`差分に加え、mainへ統合された設計文書`CONSUMER_QUALIFIED_CROSS_LANE_ACCEPTANCE_2026-11-14.md`が`plan/design/README.md`索引に載っていないため、`TestPlanReadmesMentionEveryEntry`が失敗。これは**Source→Parser→Workspace Validator→Adapter前拒否**の論理が実装不能という証拠ではなく、まず整形・索引を正しいブランチ上で修正して同一HEADのCIを再実行するべき状態である。成功した一部テストをPR全体PASSとは記さない。

**既存REA試験の更新**。[#4637](https://github.com/hat47x/tei-platform/pull/4637)現HEAD `9b8f5ca9b50d34040cfa0682f026f4fc29090f01`は**12/12 Workflow success**。Hostが固定したSelectorからbyteを読んだ実績と、許可されたStudio/SHO/AI Consumerへの現在時点の開示は別である。Raw Digestを変更してもProducerによる元の配送が証明されない場合は受理しない独立Expectedを保つ。

**実装の切り分け**：P0は①#4583 Stage1→Stage2の実Build、②既存CapabilityのL1/L2/L3診断の復旧と一結線。P1は③JSON Schema V0/V1/V2の限定read-only UX、④実Provider/独立Consumer、⑤一意に戻せる編集V3。P2は⑥Language Grammarの自己記述、⑦数理・科学Toolkit等への広げ方。外部JSON Schema対応、AI協業、独自Syntax PluginをStudioセルフビルドの新Hard Gateにしない。

**状態**：本節は現行GitHubのSource/Issue/PR、実CI、協業UX Draftの設計を反映した**計画更新**。JSON Schema製品Reader/図解、逆編集、Grammarコンパイル、Go/Node/Windowsの新規E2E、正式Owner意味受理、Stage2再構築は本節の作業では**未実施**。

<!-- tei-self-description-execution-projection-20261114 -->
## 42. 2026-11-14 業務の意味から説明・図解・実行を展開する：自己記述DSLの最小実装境界

### 42.1 今回の結論：実行方法ではなく「何が起きるか」を表す

第40節では、外部SchemaやDSLの構文を、原文に戻れるTree・属性表・図・診断へ投影する方針を定めた。次に、業務の手順や実行の流れも同じ方式で扱えるようにする。

2026-11-14に**mainへ統合された設計**である[実行モデルの図解・Target移植 #4665](https://github.com/hat47x/tei-platform/pull/4665)と[Process/Trigger/Operation意味整理 #4666](https://github.com/hat47x/tei-platform/pull/4666)は、TEI独自のEvent LoopやHostを必要としない、実行環境に依存しない意味の分離を提案している。ここではその成果を**DSLの自己記述・図解・文章化の適用条件**に絞って取り込む。mainへの文書統合を、正式DSL文法・共通Runtime・Target間の動作同値の受理とはしない。

区別する最小の意味は次のとおり。

| 概念 | 利用者に説明したいこと | 別物として保持するもの |
| --- | --- | --- |
| **Lifecycle** | アプリケーションやHostは、いつ準備・待機・停止・復旧するか | 一件のInvocation、起動要求を発した利用者 |
| **Trigger** | 何が処理開始のきっかけになるか | Eventの宣言と実際の到来、公開Entryと呼出許可 |
| **Entry・Invocation** | どの操作へ、どの対象・入力・版・期限で依頼するか | 対象発見、Hostの現在Grant、実行Attempt |
| **Operation・State** | 業務上どの条件を検査し、どの値・状態候補を求めるか | 実Providerの選抜、実入力取得、DB Commit |
| **Effect・Outcome** | 何が保存・通知・更新され、業務上どう判定されたか | 要求された作用、作用の確認Receipt、不確実な作用 |
| **Response・Observation** | 呼出元や利用者に何が返り、何を独立に確認できたか | HTTP/CLIの返答、永続化、配送、Recipientの実受領 |

例えば「受注を確定する」という業務操作を、HTTP要求・画面操作・CLIのいずれからでも起動できる。異なるのはTrigger、Host、公開Entry、応答方式であって、共有すべき業務意味そのものではない。ただし二つの入口が同じFunction IDを示しただけで、**実際に同じProvider・権限・Input/Effectを使った**とは認めない。

### 42.2 説明文・図解・テストは同じ意味に戻れるようにする

設計書を生成する際は、Data・Function・Businessの業務記述から、読者に自然な日本語として伝えられる文章を作る。説明のためだけに実行順、業務上の条件、通知の時点を補わない。元の記述に不足があれば、文章へ仮定を紛れ込ませず、**確認すべき条件**として明示する。

| 投影先 | 主な問い | 必要なSource参照と検査 |
| --- | --- | --- |
| **業務手順の説明** | 誰が何を、どの条件で行うのか | Accepted MeaningのSubject／版、条件／例外、当該文の出典Occurrence |
| **処理フロー・状態図** | どこで分岐し、どの状態へ移るのか | Typed GraphのNode/Edge identity、方向、多重度、Guard、未知の依存、図上での損失 |
| **イベント・相互作用図** | どの契機で呼び出され、誰に返るのか | Triggerの発生、Entry、Invocation/Attempt、Responseの受領者を別Identityで保持 |
| **Data・Effect図** | 何を読み、どの保存・通知を要求するのか | Required/実Read、Effect要求/Commit/Receipt、Access ScopeとDisclosure |
| **実行Trace・検査票** | 何が実際に起き、何がまだ不明か | Source・Selected実装・Attempt・Result・独立Expected・Consumer受領 |

見た目だけの差異（Graph配置、折畳み、ラベル、表示順序）を、業務意味の改版や再Buildの理由にしない。説明文・図・Markdown・Graphvizを編集する場合も、**元のSource位置と意味対象が一意に解決できる部分だけ**既存Writerへ変更候補を渡す。生成した日本語説明が自然でも、Sourceに存在しない条件を逆書込みしてはならない。実行Traceを設計上の正常フローと混同しない。

この観点は[協業UX Draft #4605](https://github.com/hat47x/tei-platform/pull/4605)のUX-PIN/UX-DIAG/UX-READと共通である。閲覧中のBを保ったまま固定対象Aを裏で再確認する場合、BのFocus・Graph・Dirty Bufferは変えない。AIが説明や修正候補を返す場合も、限定したSource／Meaning／Scopeを明示し、古い応答や未保存編集を自動適用しない。

### 42.3 TODO・Mockで通した設計と、実Providerが動いた製品を分ける

外部処理の正式実装がまだない時点でも、業務手順・Input/Output・例外・許されるEffectの契約はTEIに記述できるようにする。設計の初期段階では、その契約に基づく**TODO／Mock**と独立したテストパターンを使って全体をシミュレーションすることは有効である。

ただし、Mockの応答は**実DB保存・通知・外部API・権限の確認結果ではない**。また、Mockが期待値を自分自身から作ると、誤った意味・順序・境界を検出できない。受注確定例の最小受入を次のように分ける。

| 段階 | 最初に確かめる内容 | 合格としない内容 |
| --- | --- | --- |
| **S0 Meaning／設計** | 受注確定の条件、入力型、業務結果、必要な外部機能と不変条件を既存DSL/Roleで記述し、文・図・テスト対象へ参照できる | 実DB更新、通知送信、業務上のOwner受理 |
| **S1 Mock Simulation** | 固定入力・独立Expectedで、正常・Guard拒否・Provider未接続・Timeoutを判別 | Mock正常応答から実Effectの成功を主張する |
| **S2 実Entry／実Provider** | 公開済みの通常製品入口、Host選択Providerと実Called Go実装、現在のGrant、実Inputの読取・型を同一Attemptで照合 | 選択の申告やreadDemandだけで物理Read／許可を証明する |
| **S3 実Effect／受領** | Commit/外部通知の実Receipt、再読取、Recipientへの配送と未確認の作用を区別 | ACK喪失・部分応答を自動成功、または安全な再実行可能とする |
| **S4 異質Target** | 同じ受理済み業務意味をCLI／Desktop／Webなどで比較し、Target Binding/Hostの差を明示 | 同じ図や戻り値だけで並行性・実行順・権限・Target同値を認定する |

これらのS0〜S4は**この節での説明用の検収順**であり、既存Self-buildのS0〜S5やStudio B0/B1/B2/B3の新しい名称ではない。正式な実行・権限・副作用の判断は[#4495](https://github.com/hat47x/tei-platform/issues/4495)、[#1158](https://github.com/hat47x/tei-platform/issues/1158)、[#1129](https://github.com/hat47x/tei-platform/issues/1129)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、独立Expectedは[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)に委ねる。新しいWorkflow Runner/Approval Engine/業務Hostは設けない。

### 42.4 最初の実証は「一つの受注操作」を三つの見方で比べる

最初から汎用Lifecycle Graph CompilerやSchedulerを作らない。既存の一Function／一Trigger／一つのread-only説明を題材に、次だけを実施する。

1. **意味の読み取り**：実Sourceに存在する業務操作・型付きEntry・Guard・必要なData/Effectを、既存Reader/Role/型付きGraphへ読み、対象ID・Source位置・不明点を保持する。SourceにないTriggerやEffectは作らない。
2. **図と文章の照合**：処理説明・Event/Call図・Data/Effect表の各表示が同じ意味Subjectへ戻ることを検査する。GraphのLayoutだけ変えても意味Revision・実Build入力を変更しない。複数Edgeが同じ両端を持っても、Edge identityと業務上の関係を落とさない。
3. **契約に基づく限定Mock**：未接続のDB／通知はTODOに残し、固定された独立ExpectedとMockで、Guard拒否、Effect未確認、Response喪失、重複Triggerを判別する。**この実験は設計の検証であり、製品の実Effect/実Consumer資格ではない**。
4. **製品入口との接続判定**：既存[#4495](https://github.com/hat47x/tei-platform/issues/4495)/[#2586](https://github.com/hat47x/tei-platform/issues/2586)が実Entry・Provider・Reader/Receiptを取得した段階で、Mockを一つずつ実実装へ置き換えて再試験する。実装のないまま図と文章の見た目だけを完成扱いしない。

代表的な反証は、[PX-01〜12](https://github.com/hat47x/tei-platform/blob/main/plan/design/PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)および[Lifecycle設計](https://github.com/hat47x/tei-platform/blob/main/plan/design/PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md)をそのまま索引とする。優先するのは**Event到来≠実行許可**、**同じ結果≠同じRead/Effect**、**Commit前通知／ACK喪失**、**Source改版後の古い図／古いAI応答**、**図Layout変更≠意味改版**の五対照。新たなCase番号や第二のFixture正本は作らない。

### 42.5 P0を変えずに計画へ反映する

**P0-A Studio Desktop**：[#4583](https://github.com/hat47x/tei-platform/issues/4583)/#2831のnon-Plugin BuildExecution→実N0/N2/N3→実Stage1がStage2 Windows Desktopを生成→当該Stage2を実WebView2で使用する経路を最優先する。[#4587](https://github.com/hat47x/tei-platform/pull/4587)現HEAD `9b58533df8764a74c578d478d8c6ee92a382642e`の12/12 Workflow成功は**B0の生成/通常Windows Consumer**の資格であり、Stage1→Stage2の完成ではない。

**P0-B 既存DSL診断**：[#4654](https://github.com/hat47x/tei-platform/pull/4654)/[#4655](https://github.com/hat47x/tei-platform/pull/4655)は現HEADとも8 Workflow中7 success・Health 1 failure。実ログで確認した`gofmt`不一致、#4655の設計索引欠落を各Ownerが修正し、同一新HEADの正式Healthを再実行する。その後、Source位置に戻るSyntax/Workspace診断とRuntime Adapter call0を小さく結ぶ。外部Schemaや今回の図解の実装を、その修復に先行させない。

**P0-C 通常Entry／実Provider**：[#4495](https://github.com/hat47x/tei-platform/issues/4495)/[#2586](https://github.com/hat47x/tei-platform/issues/2586)の通常`tei` CLI→選択した実Go Provider/実Read→Function Context→独立Consumerを並行P0として継続する。モック、Reference PG、別HEADの個別成功を一つの製品E2Eに足さない。

**P1 表現と検証**：第40節V0〜V2の固定JSON Schema閲覧・原文位置へのジャンプと、本節の一Function/Triggerのread-only説明・図・Mock比較を、同じSource-backed UI技術として利用する。実装基準は**まずデータを正しく読めること、次に意味と表現の差を示せること**。JSON Schema/科学Toolkit/汎用文法対応をDesktop Stage2のHard Gateにしない。

**P2 高度な自己記述と移植**：Grammar自体の自己記述、異言語Target、状態機械の統一、数理・統計・WBS等専門Toolkitは、既存Role/Operator/Providerと受理済み意味の実比較で段階的に採否を決める。数式/単位・精度、Event順序、Effectの独立Expectedを各Ownerが所有し、図解の成功を科学計算やTarget実行の正しさへ昇格させない。

<!-- tei-self-description-generated-readonly-effect-20261114 -->
### 42.6 「読むだけ」の定義を、生成アプリケーションの実行前後でも守る

**統合後の状態（2026-11-14）**：以下は旧mainのSourceで見つかった反証の記録であり、現在の実装状態ではない。[統合PR #4683](https://github.com/hat47x/tei-platform/pull/4683)の取り込み後、現mainの`tools/reference-application-binary/templates/main.go.tmpl`はDB不要の`validate`を分離し、通常`query/command`の暗黙`applyMigrations`を除去している。実行バイナリの無作用・全Providerでの安全性は、修正Sourceの読取ではなく[#4657](https://github.com/hat47x/tei-platform/issues/4657)の対象別実試験で判断する。


自己記述DSLが「データを読む」「検証する」と説明したOperationについて、**生成された実行ファイルが予期しないデータ変更を起こさないこと**は、構文や意味の正しさとは別に確認する必要がある。

この問題は想定上の危険ではない。2026-11-14のmainで実Sourceを確認すると、`tools/reference-application-binary/templates/main.go.tmpl`では`query`、`command`、`validate`が共通で`sql.Open("sqlite", cfg.DSN)`→`Ping`→**`applyMigrations`**を経てから個々の処理へ進む。`migrations.go.tmpl`は生成されたMigrationの`SchemaSQL`を`db.ExecContext`で順次実行する。現在の実装は、問合せや検証の入口からもDDL等へ到達し得る構造である。

一方、`generator_test.go`の既存二試験は`public.`を含むPostgreSQL方言のMigrationに対し`Skip`する。`internal/appgen/model.go`の`ctx.Databases = world.Databases`／`ctx.Migrations = world.Migrations`は対象ApplicationのRequired DBへ絞っていない。`config.go.tmpl`の`TEI_ENVIRONMENT`は値として保持するが接続は`TEI_DSN`による一つのSQLite DBである。これは**Reference生成器の現状**であり、すべてのTEI製品Targetで同じ挙動が確認されたという意味ではない。詳細なSourceと修正対象は既存[生成Binary RB0 #4657](https://github.com/hat47x/tei-platform/issues/4657)／[Migration #4234](https://github.com/hat47x/tei-platform/issues/4234)が所有する。

この実例から、意味と生成物の契約を次のように接続する。

| 確認する境界 | 必要な区別 | 最初の到達条件 |
| --- | --- | --- |
| **DSL・意味** | Query/Validationの業務上の目的、宣言Effect、許されたRead Scope | SourceにEffectが書かれていないことだけで実行が無作用だとは認定しない |
| **Generator・Target** | 選択Application/Environment/Providerと、生成Entry/依存Migration/実Target方言 | 読取・検証の通常入口から暗黙Applyへの経路を除き、PostgreSQL SQLをSQLiteへ無条件に渡さない |
| **実行前の入場** | 必要なSchema/Historyの読取り、現在Grant、未初期化／不一致／未確認 | 適合しなければ`not-initialized`／`unknown`／`unsupported`で停止し、自動修復しない |
| **実行中の効果** | Query実Provider、接続モード、DBが実際に行ったDDL/DML/更新と結果 | 「Migrationを呼ばなかった」は一つの必要条件。SQL内関数等の作用まで否定するには別の制約／実観測が必要 |
| **成果物と利用先** | 実Generated Binary、Toolchain、Source Revision、Consumer Attemptと独立再読取 | Generator成功や`exit 0`だけで、無変更・現在資格・Consumer受領としない |

**最初のI0修正**は、通常の`query/command/validate`から暗黙のMigration Applyを取り除き、既存SQLiteを壊さず、未存在ファイルを必要以上に作らないことである。`validate`がDBを必要としない場合は接続そのものを省く。必要なDBをまだ初期化していなければ、暗黙に初期化せず理由を返す。明示的なSchema管理Operationの適合、Version/History、方言・複数DB選択、GrantとRollback/Receiptは#4234の後続で検収する。**「読取処理である」という宣言から、任意SQLの副作用が完全にないと推論することはしない。**

既存[RBM反証 #4657](https://github.com/hat47x/tei-platform/issues/4657)を、Source定義→Generator→実Binaryという一つの検証線に使う。

- **未存在SQLiteの通常Query**：別Processで生成Binaryを起動し、実DBファイル作成0、Migration Apply 0、DDL 0で`not-initialized`等の失敗を受領する。`Ping`やConnection初期化による副作用も含めて確認する。
- **既存SQLiteのQuery／DB不要Validation**：二度呼んでもMigration由来DDL/DML 0、Schema/行/Historyの実Read差分なし。接続が必要ないValidationではDB接続0を確認する。
- **Provider・Dialect・対象DBの違い**：`sql postgres`をSQLiteへ渡さず、無関係な別ApplicationのMigrationを選択しない。必要Target/Scopeが確定できなければ`unknown/unsupported`とする。
- **テストの実効性**：`public.`による`Skip`をMigration合格として数えない。別のSQLite適合Fixtureで生成した実バイナリを**非SKIPで起動**し、前後のSchema・データを独立に比較する。予期せぬ変更を検出するため、生成器自身が出したResultを正解の唯一の根拠にしない。

**Source／Artifactの現在資格も混同しない。** 現`model.go`の`sourceRevision`はRoot直下のエントリ名から作る簡易的な値で、ファイル内容の変化を証明するFingerprintではない。また`internal/appgen/generator.go`は必要な`BuildArtifactDigest`がない場合に`sha256:PLACEHOLDER`を入れる。この値を実生成Binaryの真正なdigestと呼ばない。実Artifact DigestはBuild完了後の外部Receipt/Manifestへ記録し、World/Model/Artifactの再利用・失効は[#2600](https://github.com/hat47x/tei-platform/issues/2600)の判断に戻す。自分のバイナリに自分の最終Hashを埋めることを自己記述の必須条件にはしない。

### 42.7 処理の抽出・移植でも、実際の契約を保てる範囲だけ変換する

同日mainへ統合された[実行モデルの契約保存変換 #4673](https://github.com/hat47x/tei-platform/pull/4673)は、第41節の図解を**実際のリファクタリングへ適用する際の境界**を具体化した。図の一部分をFunctionとして切り出すとき、見た目のまとまりだけでは独立した業務処理にならない。

まず変換の性質を分ける。

| 変換 | 最初に確認すること | 必ず残す条件 |
| --- | --- | --- |
| **表示だけを変える** | Tree/Graph/Layout/説明文の表示差分 | Source/意味Revisionと実Buildを不要に変更しない |
| **糖衣構文を変える** | 変更前後のSource位置、解釈Parser・Role・Required | 同じ構造候補でも原文同一・Owner意味受理と同一視しない |
| **処理単位を切り出す** | 境界を横切る型付き値、Read依存、例外、評価順、時点、状態所有 | Context/Grant/Transaction/Effectの必要部分が欠けたら実変換しない |
| **別Targetへ移す** | Targetが要求順序・精度・NULL・Effect/復旧を満たせるか | 実CallとConsumer受領を別々に確認し、同値Resultのみで合格にしない |
| **意図的に仕様を変える** | 変更理由・新Expected・Ownerの判断 | 従来の意味保存変換と偽らず、意味の受理からやり直す |

具体例は、受注確定の処理図から「受注情報を読む→検証する」部分を抽出してread-only Functionへ移す場合である。抽出前後で同じ受注データが返っても、**生成Binaryが先にMigration DDLを発行するなら実効果は同じではない**。Read Scope、必要なEntry/Grant、DB Connection/Transaction、実作用、Response/Receiptを比較する。Codeの一部やDiagramが移動しただけではBuild/Provider/Effectの旧資格を引き継がない。

変換後の再検証も一律ではない。Layoutのみなら表示とSource-back navigation、型・意味を変えたら関係するMeaning/Operator、TargetやBuilderを変えたら実ArtifactとConsumer、Execution境界を変えたら実Provider/Effect/Receiptをそれぞれ確認する。ただしSourceや実ProviderのA→B→Aを、旧Attemptの証拠の復活としない。対象と依存の把握が`partial`なら、その範囲でのみ再利用候補を示す。

初期の実装試験は**同じpure Functionを小さく切り出す一例**と、**戻り値が同じでも前段MigrationやProvider Readが異なる負例一例**に絞る。既存[#4673](https://github.com/hat47x/tei-platform/pull/4673)のPX-23〜29を[#4198](https://github.com/hat47x/tei-platform/issues/4198)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ参照し、新しいGraph Compiler、Migration Engine、第二のExpected正本は作らない。

**計画への割当**：生成Binaryの通常起動の安全性は#4657の**I0**として先に修正・実検証し、明示Migration管理は#4234に残す。DSLのSource→実行前拒否と通常Product Providerの検収は#4359/#4495/#2586、再Build・Artifactは#2831/#2600、Graph変換と表示は#4198/#2497を維持する。全体で最優先の**TEI Studio非Plugin Stage1→Stage2 Windows Desktopセルフビルド**は変えず、新しいフローチャート／汎用文法が完成するまでP0を止めない。

**状態**：第41.6・41.7節はmainの実Source、#4657の既存計画、main統合済み#4673を照合した**文書による引継ぎ**である。現行Referenceの暗黙Migration Callはコード上確認したが、今回そのGoコードを修正したり、実Generated Binary／SQLite／PostgreSQL／Windows／E2Eを新たに実行したわけではない。I0のPASS、全SQL read-only、安全なMigration適用、Owner意味受理、正式な製品Entry/Consumerは**未確認／未受理**。

**状態**：本節はmainに統合済みの[#4665](https://github.com/hat47x/tei-platform/pull/4665)/[#4666](https://github.com/hat47x/tei-platform/pull/4666)、協業UX Draft、既存DSL/Buildの実CIを踏まえた**docs-only計画追補**。新規Go/Node/Windows/DB/Runtime/Mock E2E、Graph Compiler、公式Grammarの採用、Owner意味受理、Stage2 Desktopセルフビルド実行は**NOT RUN / NOT ACCEPTED**。運用上の正本は既存`MASTER_PLAN`と各Owner Issueであり、この章は横断参照と最小実装順を示す。

<!-- tei-self-description-projection-qualification-slice-20261114 -->
## 43. 2026-11-14 意味を保つ対応付けを、最初の小さな実装と受入へ収束する

### 43.1 表示の便利さと、意味の保存は別に判断する

前節までに、外部Schemaからの構造表示、原文位置へのジャンプ、業務プロセスの説明・図解と実行の資格を整理した。この構想を実装する際に残る核心は、**一つの構造を別の形へ見せたとき、どの意味が保持され、どの情報が落ちたか**を利用者と開発者が判断できることにある。

[標準構造の意味的対応付け Draft #4667](https://github.com/hat47x/tei-platform/pull/4667)は、Identity／Relation／Collection／Presence／Quantity／Time／Scope／Authorityを、すべての対象に押し付ける新しい型階層ではなく、**必要な操作の前提条件**として比較する方式を提案している。ここでは[#2495](https://github.com/hat47x/tei-platform/issues/2495)の標準構造・Facet、[#2544](https://github.com/hat47x/tei-platform/issues/2544)のLaw、[#2497](https://github.com/hat47x/tei-platform/issues/2497)のProjection、[#2620](https://github.com/hat47x/tei-platform/issues/2620)の意味Roleをそのまま使い、実装に必要な対応だけを確かめる。

次の三つは同じ「一致」ではない。

| 比較するもの | 確認できること | そこから言えないこと |
| --- | --- | --- |
| **原文の一致** | Raw byte、Source位置・版、選択Readerを特定できる | 同じ業務意味である、現在受理済みである |
| **指定した観測についての一致** | 指定したNode/Edge/属性/順序などを二つのViewで同じ対象として追跡できる | 省略した属性・制約を復元できる、すべての編集が可逆である |
| **意味・実行の適合** | Ownerに受理されたMeaningと、対象Operator・Target・実Consumerについて個別に検証できる | ほかのTargetや時点でも自動的に有効である |

「同じDataからTreeもGraphも作れる」という主張は、**各Viewで何を観測できるか**と切り離せない。表現間の変換は、埋め込み・部分射影・集約・近似・意味解釈を区別する。例えばTreeにNode/Edgeの元IDを付けて表示できても、共有Node・複数親・循環や多重Edgeを編集結果から元どおり復元できるとは限らない。画面上の「編集できそうな形」を、Writerが対応できる証拠にしない。

### 43.2 最初の実装は、特定のGraphとSchemaで二段階に分ける

**第一段階：既存のGraph／Tree読取**。同じラベルを持つ二Node、同じSource/Targetを結ぶ二つの有向Edge、Edge固有の属性、別の親からの参照を含む、**一つの小さな固定Graph**を使う。既存Reader/構造・IDで元データを確認し、Tree/Propertyへread-only投影する。各表示に元Node/Edge identityと取得Scopeを結び、表示しきれない辺・属性・共有関係を明示する。Treeで描けたことは「Graphの全情報を保って編集できる」ことの証拠ではない。循環・孤立Nodeはこの同じ固定題材の限定変種で比較する。

**第二段階：固定JSON Schemaの説明**。第40節V0〜V2と[#4667](https://github.com/hat47x/tei-platform/pull/4667)の構造特性を、**明示した一つのDialect**へ限定して使う。`$schema`・`$id`・`$defs`／`$ref`、`oneOf`と`anyOf`の相違、条件付き制約、`default`などの注釈、未知Keywordを構文・制約・表示注釈として区別する。`$dynamicRef`などの評価環境依存機能を既存Readerが扱えない場合は、実装済みと装わず`unsupported`または未評価として扱う。

JSON Schemaは**JSONの値が満たす条件を記述する言語**である。Schemaの`properties`をRecordの全項目が実在すると読み替えたり、`default`を自動保存すべき入力値とみなしたりしない。`oneOf`は指定された候補のうち**一つだけ**が成立する制約で、`anyOf`と同じ「選べる候補一覧」にはできない。`unevaluatedProperties`等の適否が不明なら、構造だけを読んだ状態に留める。

まずHeadlessのSource位置・構造・診断を照合し、その後必要な部分だけをStudioとCLIの実Consumerへ表示する。存在しないSchema Readerを作ったことにして、GUI実装を先に進めない。新Parser/Grammar IR/Renderer Registryの必要性は、既存[#988](https://github.com/hat47x/tei-platform/issues/988)／#1018のReader/Language Pluginで対応できない具体的な差が見つかった場合に限って検討する。

### 43.3 逆編集の適否を、表示結果ではなく変更対象で決める

Studioで一項目の変更を提案する際は、次を別々に扱う。

- **現在の対象**：Qualified Subject、元のSource occurrence／Revision、どのVersion・Language実装で読んだか。
- **投影の条件**：元のNode／Edge／属性／Schema Keywordと表示先の対応、落ちた情報、共有参照、取得範囲。
- **変更の条件**：当該操作で対象が一意か、既存Writerがその変化を表せるか、未保存Bufferや後着AI候補と競合しないか。
- **保存後の確認**：Preview→差分→元Revision確認→既存Writer→独立再Read。意味受理、DB更新、Host Grantや実Target利用は別資格。

初期実装で確認するのは**read-onlyの対象ジャンプと、一意な項目だけの局所Patch候補**まで。GraphをTreeへ射影して多重辺・共有参照が落ちた場合、共有`$ref`の先を複数画面が使う場合、SourceがR0→R1→R0と戻った場合は、自動保存せず対象選択と現在版の再確認に戻る。Designerが「原文に戻れる」と表示しても、未対応のSource Writerがあれば保存操作は`unsupported`のままにする。

これは[協業UX Draft #4605](https://github.com/hat47x/tei-platform/pull/4605)のUX-PIN／UX-DIAG／UX-READへ吸収し、新しい第四のUXフローや承認機構を作るものではない。自然文・Graph・Property・原文のどこから操作しても、同じ対象に戻れることを重視する。

### 43.4 一つの検証票で区別する五つの状態

[#4667](https://github.com/hat47x/tei-platform/pull/4667)のSM-01〜14、[#4605](https://github.com/hat47x/tei-platform/pull/4605)のSX-01〜08と既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)を、**新Fixtureの第二正本を作らず**次の限定試験として結び付ける。

| 証明したい状態 | 固定入力と一因子差 | 独立Expectedで確かめること |
| --- | --- | --- |
| **Read** | 同名Node／並行有向Edge／別親を持つGraph | Readerが二Node・二Edge・各IDと関係属性を区別し、未取得は空と扱わない |
| **Project** | 同じGraphからProperty/Treeへ変換 | 戻れる位置と消失情報を明示し、read-only表示は許容する |
| **Diagnose** | Schemaの`oneOf`／`anyOf`、未知Keyword、未解決`$ref` | 取り込んだ構文・適用可能な制約・未評価を区別し、存在しないSource位置を作らない |
| **Propose** | 一意な単一Keyword対、共有`$ref`先・古いRevision | 前者だけ条件付きPatch候補、後者は曖昧／staleとして自動書込0 |
| **Consume** | 同じSubjectをCLIとStudioへ投影 | 各実Consumerが受領したことを独立に確認。見た目一致だけでMeaning・Grant・Writer・Buildを受理しない |

記録する最小限の情報は`Subject/Source Revision、選択Reader、Projection/Target、Scope、preserved/omitted、独立Expectedの根拠、実Consumer/Attempt`とする。既存EvidenceとScenarioの**適用先を明確にする記録欄**であって、新しいCore構造、Version Registry、万能Projection Engineは不要である。どの項目を実際に観測できたかを示し、未観測欄は`unknown`とする。

### 43.5 最新の進捗と、実装の優先順位

- **設計上の進展**：[構造対応付け #4667](https://github.com/hat47x/tei-platform/pull/4667) HEAD `d5039dd048fb552f4462b6ebc12d1709d22d0e21`はDraft/未mergeで、確認時のGitHub Actionsは**1 success／1 in_progress**。SM-01〜14とS0〜S5は受入計画であって、実試験の合格ではない。[実行モデル追補 #4668](https://github.com/hat47x/tei-platform/pull/4668)はmainへ統合されたdocs-only設計であり、Target移植の実機検証を示さない。
- **近接する設計**：[数理・計画Toolkit #4670](https://github.com/hat47x/tei-platform/pull/4670)は、移動平均→WBS/CPM→Solver/DOEの独立Expectedの順と、Schema→Source位置→限定編集のStudio実証を提案するDraft。これらを**Rendererができたから数値的にも正しい**と扱わない。[Plugin Role/Stage #4671](https://github.com/hat47x/tei-platform/pull/4671)は同一Meaningに対する段階別Contribution/Source往復を整理した新Draftで、確認時**1 success／1 in_progress**。Editorで読めることとRuntime/Stage2へ必要なContributionが提供されたことを分離する。
- **P0は変更しない**：[#4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `9b58533df8764a74c578d478d8c6ee92a382642e`の12/12 Workflow成功は生成・通常DesktopのB0実績に限定する。正式な非Plugin Build Intent→Stage1がStage2を生成→そのStage2がWindows WebView2で実行される検証は[#4583](https://github.com/hat47x/tei-platform/issues/4583)/#2831の最優先。通常製品CLI→選択した実Provider→独立Consumerも[#4495](https://github.com/hat47x/tei-platform/issues/4495)/#2586の並行P0とする。Capability Sourceの検査[#4654](https://github.com/hat47x/tei-platform/pull/4654)/[#4655](https://github.com/hat47x/tei-platform/pull/4655)は各現HEAD**7/8 CI success、Health failure**（Go整形、#4655は設計索引も不足）を修正後に再検収する。
- **P1の最短実装**：#2497/#2514による**一GraphのRead→二View→loss診断**を先に通し、次に#988/#2620/#1124で固定DialectのJSON Schema V0〜V2を試す。逆編集・宣言的View・Grammar自身の自己定義は、必要なSource同一性とWriter可否が実証されてから個別に判断する。

**実装判断の基準**は、同じ表示結果を作れたことではない。**意味を保持する条件を説明でき、失われた情報を隠さず、戻せない編集を停止できること**である。これを一つの小さな試験で確認し、足りない契約だけを既存Ownerが補う。

**状態**：本節は直近の他チャット・Issue・GitHub PR/Actionsの読取りによる**docs-only計画の追補**。SM/SXの新しい実試験、Graph/Schema ReaderやGUIの追加、Go/Windows/Provider/DBの実行、Owner意味受理、Stage2の新セルフビルドは行っていない。正式なMeaning Schema/Grammar IR/Plugin ABIの採用をここで宣言しない。

<!-- tei-self-description-cli-native-self-build-20261114 -->
## 44. 2026-11-14 CLIを起点とする意味あるコード生成とPluginの自己構築

### 44.1 残っていた課題：「TEIで組み立てた」と「TEIから実装を生み出した」は違う

自己記述DSLの到達点は、`*.tei.md`がGoの既存関数へ処理を委ねることだけではない。業務・構造の**意味を持つ図、表、Rule、条件**をTEI側で解釈し、実際の検証・計算・選択処理を生成できる領域を広げる。そのための最初の実証は、完成したStudio GUIを必要としない。通常のTEI CLIを起点とし、必要なら既存CLI/APIを薄く公開するMCPからも同じ操作を扱う。

この判断は[Plugin自己生成の計画 Draft #4675](https://github.com/hat47x/tei-platform/pull/4675)、[CLI-firstの製品完成計画 Draft #4676](https://github.com/hat47x/tei-platform/pull/4676)に具体化された。また、[公開図解の継承 #4677](https://github.com/hat47x/tei-platform/pull/4677)と[PluginのRole/Stage適合 #4671](https://github.com/hat47x/tei-platform/pull/4671)は**mainへ設計文書として統合済み**である。

ここで定義する初期の成果は、**TEI定義が本当に実装ロジックを決定し、その生成物を選択されたPluginが実行し、その結果を独立したConsumerが受け取れること**とする。固定JSONを吐く、未解釈のGoを呼ぶ、Goを二度ビルドする、Stage1と名付けた外部スクリプトを走らせる、といった振る舞いだけでは成立しない。

### 44.2 同じ小さなGraphを、表示・生成・Plugin・セルフビルドで使い通す

第42節の固定Graphを、この検証の入力として再利用する。識別子`n-1`、`n-2`を持つ二Nodeは、表示名をどちらも`item`とする。`e-1`、`e-2`は異なるEdge IDを持ち、初期状態ではどちらも`n-1→n-2`を向く。各Edgeの意味属性と向きを、配置座標とは別に保持する。

**生成する最初の処理**は「指定Nodeから出るEdge IDの列とその件数を求める」ことに絞る。入力のStable IDを参照し、同名の表示ラベルではNodeを同定しない。出辺の順序は、Sourceで明示されない限り業務意味の一部にしない。独立Expectedは出辺の集合・件数を固定する。

| 一因子差 | 期待する実動作 | 合格とはしない近道 |
| --- | --- | --- |
| 初期Graph | `n-1`の出辺は`e-1,e-2`の2本、`n-2`は0本 | 固定値`2`を返すだけ、同名Nodeを併合 |
| `e-2`だけ逆方向へ変更 | `n-1`の出辺は`e-1`の1本、`n-2`は`e-2`の1本 | 既存Go側の表だけ手修正して結果を合わせる |
| `e-2`を削除 | `n-1`の出辺は`e-1`の1本 | 表示から消しただけで生成結果を変更しない |
| 図の座標のみ変更 | Generated Rule/実ロジック・意味結果は不変 | Layout差分をMeaning/Codeの変更と扱う |
| Edge IDを重複させる | 診断の上で採用・生成を拒否 | 最初のEdgeで上書き／後勝ち採用 |

最初の生成内容は、単なるGraph Viewerや定数テーブルではなく、**GraphのID・方向・重複制約を評価するコード**とする。この限定例なら、Source変更と生成された処理の因果関係を追いやすい。`Graphviz`の配置図やMarkdown表が使われても、それ自体を意味正本として自動昇格させず、既存Source ReaderとRole Bindingで解釈対象・位置・版を確定する。現在の`*.tei.md`に正規のLoader/Readerが存在しない場合は`unsupported`と報告し、暫定変換やテスト用Bridgeを正式な読込み成功とは扱わない。

### 44.3 合格を五つに分け、再帰セルフビルドを最後まで追う

| 到達段階 | 検証する実際の境界 | 不合格・未実証となる例 |
| --- | --- | --- |
| **R0 Source解釈** | `*.tei.md`の実byte、図表のOccurrence、明示Role、Required Sourceと選択Language/Parserを実Readerで結ぶ | 説明上のMarkdownを無検証の手書きGoへ委譲 |
| **R1 意味のあるコード生成** | Rule/Graphの意味からEdge判定処理を生成し、変更で生成処理が変わる | コードに意味上の条件が現れないWrapper-only |
| **R2 実Plugin呼出し** | 生成コードを既存Builder/ArtifactSetから実Binaryに組み込み、選択Providerを通常CLIで実呼出し・受領 | Build successだけで実Invokeを省略、選択Provider Bに対し実Aを呼ぶ |
| **R3 再生成と次段の実利用** | Stage1の実TEI Processが同じ定義からStage2 TEI/Pluginを構築し、Stage2自身が生成Pluginを実使用 | 外部Go Buildを二回しただけ、Stage2は古いArtifact |
| **R4 第二用途・Desktopへ展開** | 同じ意味をCLI以外のread-only Inspectorや、最終的なWindows Wails/WebView2でも受領 | Testだけ通し、実Studio/第二Consumerを未起動 |

R0〜R4はこの節での**検収の見通し**であり、[製品計画 #4676](https://github.com/hat47x/tei-platform/pull/4676)のSB-0〜4、[Plugin計画 #4675](https://github.com/hat47x/tei-platform/pull/4675)のR0〜R5、StudioのB0〜B3を改名・統合する正式工程表ではない。実装Ownerは各PRの既存工程を維持する。**最初に実行するのはR0〜R2**で、R3を通らない段階は「生成Pluginの局所的な成立」と明記する。既存BuildIntent・Builder・ArtifactSet・Source/Dependency/Generatorの実装を再利用し、別のBuild Engineを設けない。

生成したGoは原則として再生成可能な成果物であり、手書きによる後付け修理を成功の証拠にしない。拡張点が必要な場合は[#2090](https://github.com/hat47x/tei-platform/issues/2090)のSource保全と契約上の境界へ戻す。

### 44.4 Plugin自己構築の進歩を測る方法

「TEI化率」を単一の見栄えのよい数字にしない。**比較対象のFunction、Rule、条件分岐、Plugin Roleと実装義務の集合を先に固定**し、次の四観点を独立に測る。

| 観点 | 測るもの | 評価上の注意 |
| --- | --- | --- |
| **意味処理の生成範囲** | 固定した実装義務のうち、TEI Sourceから実ロジックを生成したもの／Goへ委譲したもの／手書きGoのみ／未実装を分類 | Goへの委譲ラッパーをTEI生成による実装完了と数えない |
| **Buildの自己制御** | 実Reader、Generator、Builder、Artifact/Member、Stage1→Stage2のどの辺をTEI自身が選択・起動・照合したか | 外部Launcherや二度のGo Buildを再帰成功としない |
| **Plugin自身の再生成** | Pluginの処理・Validator・Generatorのうち、次の版のTEI定義から再生成でき、実選択・呼出しされた範囲 | 実Pluginを呼ばず、成果物のファイル数だけ数えない |
| **定義・保守の簡素化** | 同じIndependent Expectedを満たす前後で、重複したRule/設定、補助Go、手修正箇所、改版に必要な変更箇所と確認負担 | DSL行数だけを減らし、隠れたGoや外部設定へ複雑さを移したら改善としない |

最初の測定では比率の目標値を先に決めず、**対象集合と観測値を提示する**。Graphの向き変更時に手書きGoの修正が残るなら、その残りを改善候補として示す。反対にDSLの宣言が長くても、二つの用途で同一の意味を安全に再利用でき、既存のGo分岐を減らせたなら価値を評価する。成果物byte・生成Go行数・CIのGreen件数は、上記の分子・分母の代用にならない。

### 44.5 上位Pluginから公開された図解を、契約どおりに再利用する

[main統合済み#4677](https://github.com/hat47x/tei-platform/pull/4677)では、依存する上位Pluginから、型付きGraph、公開Diagram、Relation、Guard、Trait/Obligation、検証参照を**契約で許された範囲に限り継承的に活用**する方針が整理された。これはクラス継承ではなく、既存Plugin依存・Contract参照・Projection合成による利用である。

例えば共通Lifecycleを公開するPlugin A、Aを使う業務Plugin B、Bの図を使うUI Plugin Cの三段階で、CがAの非公開情報まで見られるわけではない。A→BとB→Cの双方について、公開範囲と**再公開の許可**を明示的に照合する。依存関係が成立しても、現在のSource閲覧・書戻し・生成・実行・外部AIへの送出が自動で許可されることはない。

最初のread-only比較は、**上位Plugin Aが公開した一Graph、Bがそこへ局所差分を重ねたView、CLIが表示した型付き表**の三つで行う。元Node/Edge/Role/ID/版、公開Scope、失われた情報、追加Overlayの所有者が一致することを確認する。同名Nodeや多重辺をラベルで統合しない。親の必須Guardを子が図上で消しても、実ExecutionのObligationは消えない。親の意味を修正する場合は親OwnerのSource Change Proposalへ返し、子は自身のOverlayだけを変更する。

**受理を分ける**：公開図面の受領、派生Viewの合成、契約に基づく再公開、独立した実Provider呼出し、Host Grant、生成Targetでの利用は別々の結果とする。公開Facetの版が変わった場合も、実際に選択・消費した依存先だけ再評価し、古いViewやAI要約を現在のDisclosure資格として再利用しない。反例は既存[#4677のPX-38〜47](https://github.com/hat47x/tei-platform/pull/4677)を[#951](https://github.com/hat47x/tei-platform/issues/951)/[#2497](https://github.com/hat47x/tei-platform/issues/2497)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ紐付け、新しい公開Facet Registryを作らない。

### 44.6 次に着手する実装と、現時点の到達上限

**統合後の状態（2026-11-14）**：この節でDraftと記した[#4541](https://github.com/hat47x/tei-platform/pull/4541)と[#4676](https://github.com/hat47x/tei-platform/pull/4676)は、[#4683](https://github.com/hat47x/tei-platform/pull/4683)経由でmain統合済み。[#4675](https://github.com/hat47x/tei-platform/pull/4675)は取得時点では設計Draft。#4683/#4684の対象PR HEADはそれぞれ14/14 Workflow successだが、CLIでのTEI-native Plugin生成やStage2 Desktop完成の証明ではない。以降の歴史的CI記録を現状判定へ流用しない。


1. **最初の実装PR**：#2621/#4359/#2620が、上記の固定Graphを`*.tei.md`から**正規のReaderで読めるか**を実Sourceで検査する。読めないなら具体的な不足箇所だけをLanguage Plugin #988/#1018へ返し、代替JSON等を正式TEIの成功としない。読める範囲だけをSource/Role/Ruleとして固定する。
2. **次の実装PR**：#4544/#951/#1121/#1251の既存Generator/Builder/Providerで出辺の検証・選択処理を実生成し、通常CLIの実呼出し・独立Expectedへつなぐ。最初はEdge反転、削除、Layout-only、重複IDの正負を一つのFixtureで判別する。
3. **再帰への接続**：#2831/#4583のStage1実ProcessからStage2実Artifactを構築し、同じPluginをStage2でも実使用する。非Plugin Studio BuildとPluginの生成比率は**別のClaim**。CLIによる局所実証を早めても、正式Windows Desktop Stage1→Stage2を置換しない。
4. **第二用途と簡素化の評価**：CLIの結果をread-only Inspector/Studioに持ち込み、Plugin A/Bの公開View合成と局所Overlayを検証する。同一Expectedを保ったまま、重複Rule・手書き分岐・補助Goの変化を実測する。

**取得時のGitHub状態**：#4675 HEAD `5fe713138f7d9a5d31001261dd82346594408ebf`は**docs-only Draft、2/2 Workflow success**。#4676 HEAD `b27c493c2d101c6f8b29ce7aa342b562d99a2459`も**docs-only Draft、2/2 success**。#4671と#4677は**mainへ統合済みの設計文書**であり、生成Plugin/公開Facet継承の実装成立ではない。[#4587](https://github.com/hat47x/tei-platform/pull/4587)のB0 Windows CI 12/12 successもStage2自己構築の証拠ではない。

**全体のP0は変更しない。** Studio DesktopをTEI自身がStage1→Stage2まで構築し、生成されたWindows Wails/WebView2で使用する製品出口と、通常CLIからの実Entry→実Provider→独立Consumerの検収を維持する。一方、CLIで先に試せるGraph/RuleからのPlugin自己生成は、その製品目標を早める**並行する小さな実装**とする。新Host/Parser Registry/Runner/万能Graph Compiler/意味正本は設けない。

**状態**：本節は他チャットの成果と実GitHubのPR・main設計・Actionsを照合した計画更新である。`*.tei.md`の正式Graph Reader、Graph Ruleのコード生成、Plugin実行、Stage1→Stage2 CLI/Windows、Plugin自己生成比率・定義削減の実測、公開Facet合成の実テスト、Owner意味受理は今回**NOT RUN / NOT ACCEPTED**。設計の採用候補と、完成した製品機能を混同しない。<!-- tei-self-description-docs-generated-graph-20261114 -->
### 44.7 生成Pluginの機能と説明を、同じ定義から辿る

第44.2節の同名二Node・有向多重Edgeを用いる**出辺ID集合／件数の生成Plugin**を、Plugin文書の最初の共通実証にも使う。専用の別Fixtureや新しいPlugin文書正本を増やすのではなく、**一つの業務上の意味から生成した実装と、その使い方を調べる経路**が一致していることを検証する。

[Plugin文書契約 #4679](https://github.com/hat47x/tei-platform/issues/4679)と[既存Plugin設計の`18](https://github.com/hat47x/tei-platform/blob/main/plan/design/PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)を受入Ownerとする。Plugin Aは固定した公開Graph/Ruleと短い説明を提供し、Plugin BはAの公開Contractに依存して出辺処理を生成する、という二Pluginで始める。

最初に利用者へ示す説明は、Plugin ID・用途、Contract/Version、公開Input/Output、選択Target/必要条件、最短のread-only例、直接依存、制限、マニュアル索引とする。契約から確定できる署名・版・公開モデル・選択Providerは**実ContractとSourceから導出**する。説明文・使用方法・設計意図・故障対応は**Plugin自身が所有するMarkdownマニュアル**を使い、必要なときだけ辿る。説明を複製して別のCapability正本としない。

対象の固定には、`Plugin/Contract ID・解決済みVersion・Dependency Lock・Source occurrence/Revision・Doc ID/Page Ref・公開Scope・Consumer/Attempt`を用いる。これは既存のContract、Source、Projection、Disclosureの参照を照合するための欄であり、新しいDocs Registryや万能Schemaではない。文書が読めることは、Readerが業務DSLを理解したこと、実Pluginが組み込まれたこと、現在のGrantや再公開許可があることを保証しない。

### 44.8 初回試験で「コード・文書・上位参照」のずれを検出する

| 一つだけ変える条件 | 生成処理と文書の期待結果 | 停止・未受理にすること |
| --- | --- | --- |
| 初期Graph／A v1→Bの固定依存 | TEI定義から出辺`e-1,e-2`を評価する実Pluginが選択され、Bの概要からA v1の公開Graph・マニュアルへ辿れる | 固定JSON出力・Go Wrapperのみ、A別版の説明を現行扱い |
| `e-2`だけ逆向き | 生成ロジック／CLI受領が各Nodeの出辺1本へ変わり、利用例・説明の適用版を再照合する | Source変更前の文書・Plugin Artifact・旧Expectedをそのまま採用 |
| 配置・説明文だけ変更 | 表示と文書の版差を記録し、意味上の出辺Rule・実Buildへの不必要な失効を避ける | Layout変更を意味変更と誤認し、無関係な実装を再Build |
| Aの公開範囲を縮める | Bは新しい公開Scopeで参照し直し、必要時はpartial/deniedと表示する | 古い文書CacheやAI要約から非公開Node・件数・RefをCへ再公開 |
| 上位依存を増やす | 直接/推移参照とStable ID・Versionを保持し、複数親、循環、同名文書を診断 | 登録順winner、循環の無限探索、アクセス不能を「依存なし」と確定 |
| 未実装のDocs CLI操作をSkillが提示 | 実`tei commands`/現行Inventoryで使用可能性を再発見する | 架空の`plugin docs`コマンドを実在する例として実行・配布 |

最初の作業順は**同一実CLIバイナリによる現行コマンド/Source Readerの確認→A/Bの公開Facetと一ページのread-only参照→同じGraph定義から出辺ロジックを生成しBの実Callを受領**とする。`*.tei.md`の正規Readerが未実装なら、その範囲は`unsupported`とし、手変換や代替JSONを正式なTEI-native生成合格にしない。

外部Harness向けの[既存Product Skill](https://github.com/hat47x/tei-platform/blob/main/skills/yi-tei-modeling/SKILL.md)は比較Baselineとして変更しない。新版は短い入口に留め、詳細は`references/`とPlugin所有の文書へ遅延参照する。Tool-onlyとSkill＋詳細参照を、同じ実Binary・Task・Workspace/Scopeで比べ、実Tool argv・Reply受領・文書版・誤呼出し・Context費用を別測定する。Skillへ全Plugin資料を常時同梱しない。

**引渡し先**：Documentation Surfaceと依存参照は[#4679](https://github.com/hat47x/tei-platform/issues/4679)、Source/Roleは#2621/#988/#2620、Pluginの選択・再公開は#951/#1016、Graph逆写像は#2497、実生成は[#4544](https://github.com/hat47x/tei-platform/issues/4544)/#1121、CLI実呼出しは#1251、Disclosureは#2516、Skill比較は#2605/#2509、独立Expectedは#1180/#2514。最終製品のStage1→Stage2 Windows Desktopは#4583/#2831を最優先とし、この小実証を新しいHard Gateにしない。

**現時点の受理上限**：mainの#4683/#4684の各14/14 GitHub Workflow成功とReference生成Binaryの通常Migration呼出し修正は実装進捗として記録する。しかし、A/B文書階層の実CLI取得、Graph由来の生成Plugin実Call、短いSkillの改善、Studio文書ナビゲーション、Stage2 Desktopはこの追補による新実証ではない。**本節は設計・受入手順の更新であり、新しいGo/Windows/CLI/AI実行試験やOwnerの正式意味受理を実施したものではない。**
