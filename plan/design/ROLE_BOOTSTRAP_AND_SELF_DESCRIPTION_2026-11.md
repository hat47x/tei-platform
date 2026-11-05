# TEI Everything is Data — 最小Bootstrap・Role解釈循環・利用資格の接続計画（2026-11-02）

**状態：横断設計候補／実TEI Core・DSL・Role Resolverへの採用前。** 本計画の設計上の中心は、標準データ構造でRole/Operator/構文Plugin自身を記述可能にしながら、**その自己記述を初めて読むための有限な解釈の根**を明示し、循環と未決、実行資格の区別を機械検査できるようにすること。

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

この観測を基に読み取り専用比較へ `contractCandidates` を追加した。変更したContractの前後の明示参照を8種類だけ、Database・Contractの完全一致で列挙し、Query単位の参照にはQuery名を残す。対象はApplicationのContract要求、Application ReleaseのContract要求、reference PatternのContract Query呼び出し、Database Releaseの提供Contract、Solution ReleaseのContract gate、header-detail Binding、Executor Binding、SQL realizationである。Contract単位の候補であり同一Contract内の別Queryも保守的に表示する。未走査参照・動的依存・互換性・実行結果は未検証と出力へ明示し、既存の受理判定には接続しない。隔離ReferenceでApplication要求・header-detail Binding・Executor Binding・SQL realizationを前後に各1件確認し、`master_data.customer` の版変更では `order_entry.customer_reference` と `repair_intake.customer_reference` の `find_by_id` 呼び出しが前後各1件の候補になることを確認した。別Contract・別Databaseの誤一致と、削除前参照の喪失を合成負例で検査する。

**Release記録との対応**：`sales_core.order` を v4.2 から v5.0 へ上げ、同時に現行Applicationの要求を `^5` へ変えると `validate` は成功する。しかし `releases/applications/order-entry/3.8.0.tei` の `^4` と `sales_core` 4.1.0／4.2.0 の提供4.xは診断されず、Release記録と現在定義のmajor不一致が無言で残る。Application要求を変えずにv5.0へ上げた場合は `DATABASE_CONTRACT_VERSION_UNSATISFIED` で停止する。逆方向として現行定義をv3.0、現行要求を `^3` へ下げると、提供4.xが現行定義より新しいため `DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION` が出る。つまりDatabase Releaseは「記録が現行より先行」だけを警告し、「現行が記録より先行」は警告しない。Application Releaseの要求はContractの存在と所有者だけを検査し、互換majorは現在定義と照合しない。v4.3へのminor更新は成功し、Release記録は比較の `contractCandidates` に前後とも現れる。歴史的Releaseを一律に不正としないため `validate` の受理条件は変えず、この非対称を読み取り専用候補で可視化するに留める。実Deploymentの順序・データ移行・Provider実行は未検証である。

**Solution ReleaseのContract gate**：`order-rollout` の `master_ready` は `master_data.customer ^2` を宣言する。現行の `master_data.customer` をv3.0へ上げ、現行Application（`order_entry` と `repair_intake`）の要求を `^3` へ揃えると、`validate` は成功し `solution-release-graph` も `executable=true` のまま `master_ready` を `order_app` の前提として解決する。gateの `^2` は履歴Application Release 3.8.0の `^2` とだけ照合され、現在定義のv3.0とは比較されない。これはgateが実行時に観測する提供Contractのmajor（`internal/deployment/durable`）とも別の検査である。一方、gateのContractを存在しない名前に変えると、そのgateを必要とするApplication Releaseがあるため `validate` が `SOLUTION_RELEASE_NOT_EXECUTABLE` で停止する。ただし、どのApplication Releaseからも必要とされないgateを追加し、存在しないContractとmajorを宣言しても `validate` は成功し、診断は出ない。`compatibleMajor` が0の場合は `INCOMPLETE_DATABASE_GATE_STEP` で拒否される。gate宣言自体は `contractCandidates` に前後とも現れる。gateの現在定義との照合を必須にするかは受理条件を変えるため、ここでは観測だけを残す。

**未使用gateの読み取り専用候補**：未使用かつ不成立のgateを、比較CLIの `unreferencedUnsatisfiableGateCandidates` として前後snapshot別に列挙する。条件は、どのApplication Releaseの要求（Database・Contract・互換major）にも一致せず、現在のContract定義にもDatabase Releaseの提供にも互換majorを満たすものがない、の両方である。どちらか一方だけのgateは出さない。参照されるgateはrelease compilationで検査され、満たせるgateには提供元があるため、この積集合だけを観測する。現行Referenceでは候補は空で、`master_data.nonexistent ^7` の未使用gateを追加すると `validate` 成功のまま1件出て、識別子・Contract・majorが保持される。`sales_core.order_line ^1` の未使用gateは現在定義で満たせるため出ず、削除したgateは変更前snapshotにだけ残る。合成試験では、参照されるgate、現在定義で満たせるgate、Database Releaseで満たせるgateを除外し、Database所有者の異なるgateを含める。実行時のDeployment状態、動的利用者、Provider動作は未検証であり、空の候補は問題なしの証明ではない。

**Application ReleaseのContract要求と現在定義**：履歴Application ReleaseのContract要求は `validate` では存在と所有者だけを検査され、互換majorは現在定義と照合されない。そこで `applicationReleaseContractCandidates` として、現在定義に存在しない要求と、互換majorが現在定義のmajorと異なる要求を前後snapshot別に列挙する。現行Referenceでは `order_entry@3.7.0` の `sales_core.order_legacy ^1` が前後に各1件出る。`sales_core.order` をv5.0へ上げ、現行要求を `^5` に揃えると `validate` は成功し、`order_entry@3.8.0` の `^4` が `currentContractVersion: 5.0`、`reason: compatible-major-mismatch` として変更後だけに出る。v4.3へのminor更新では増えない。minor差を無視するのは互換majorの判定に合わせた限定であり、履歴Releaseの互換性や実行時の挙動を保証しない。受理条件は変更しない。

**Database Releaseの提供Contractと現在定義**：Database Releaseの提供Contractは `validate` で現在定義との存在・所有者・先行方向だけを検査され、記録が現在定義より古い方向は診断されない。そこで `databaseReleaseContractCandidates` として、提供Contractのmajorが現在定義のmajorより低いものを前後snapshot別に列挙する。現在定義に存在しない提供は既存の `DATABASE_RELEASE_CONTRACT_NOT_CURRENT` が扱い、先行方向は `DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION` が扱うため、どちらも候補にしない。minorだけが古い提供も含めない。現行Reference（`sales_core.order` v4.2）では候補は空で、v5.0へ上げると4.1.0と4.2.0が `providedContractVersion: 4.1`／`4.2`、`currentContractVersion: 5.0` 付きで変更後に出る。v3.0へ下げると4.1.0と4.2.0は既存の先行警告だけになり候補は空、v4.3へのminor更新でも空のままである。履歴Releaseの正否、データ移行、実行時のDeployment状態、Providerの動作は未確認であり、空の候補は問題なしの証明ではない。受理条件は変更しない。

**候補ビューの負担比較**：独立したWorkspace複製に代表的な変更を加え、4つの候補ビュー（`contractCandidates`、`unreferencedUnsatisfiableGateCandidates`、`applicationReleaseContractCandidates`、`databaseReleaseContractCandidates`）の件数が意図した境界だけを報告するかを測る。現行Referenceのbaselineはcontract 0・gate 0・appRelease 2（`order_entry@3.7.0` の履歴Contract欠落が前後各1件）・dbRelease 0。無関係な `sales_core.order_line` v1.0→v1.1 と `sales_core.order` v4.2→v4.3 のminor更新ではcontractだけが1件になり、他の3ビューはbaselineのまま。`sales_core.order` v5.0と現行要求 `^5` の同時変更ではdbReleaseが2件（4.1.0／4.2.0）、appReleaseが1件（`order_entry@3.8.0` の `^4`）だけ増え、gateは0件のまま。未使用の `master_data.nonexistent ^7` gateを追加するとgateだけが1件増える。要求を `^4` のまま残したv5.0は `validate` が失敗し、既定の比較は終了1・候補0件、`-include-unvalidated-candidates` でも終了1とエラー診断を維持したままdbRelease 2件とappRelease 1件を出す。つまり、この範囲では候補ビュー間の相互混入は観測されず、未検証状態の候補は明示操作なしに出ない。ただし対象は合成Referenceの5シナリオだけで、実変更履歴・実利用者の負担、Providerの意味適合性、他環境での再現、実行時Deploymentとデータ移行は未検証である。件数が0でも問題なしの証明にはならない。

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
