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
| Build・配布 | RQ-008は外部Pluginの配布物の内容Digest確認を要求する。`internal/plugin/build.go` のBuildPlanは選択したPlugin、Source、Builder、Target等を固定し、成果物Lockとは別である | 通常の定義編集や定義間RefにBuildPlanを必須とする根拠ではない。Plan v1はSourceのLocator変更も別Fingerprintとする。下記の実コンパイラ比較では位置の影響はBuild条件で異なるため、固定の一律緩和は保留し、成果物の同一性確認も削らない |
| 変更影響 | RQ-008はConsumerと再検証範囲の追跡を求め、依存の欠落を影響なしと扱わない。`internal/modelref` は渡されたGraphと変更を検査し、候補を返す | Graphの構造が正しく影響候補が空でも、動的参照や未収集のConsumerがない証明にはならない。両Review APIの `authorityLimits` に網羅性未確認を明記し、追加収集や再検証の要否は用途別に判断する |
| 定義変更の差分 | Reference用 `tools/internal/changeanalysis` はApplicationの意味名前空間の利用宣言を比較する。既存の利用判定は名前空間IDの所属を見ている | 宣言順だけでApplication自体をchangedとする過剰な候補を避けるため、比較用コピーの `SemanticUses` のみ整列する任意の候補APIを試作する。重複は消さず、実行時に宣言順を保つValidationは整列しない。既存の厳密な `DiffApplications` はShared Acceptanceにも使われるため変更しない。候補APIは受理用ではなく、Source順・Targetへの影響と依存の網羅性が未評価であることを結果に含める。既存の受理経路には接続しない |
| 実行・外部作用 | 現在の権限・対象・実際の結果は、定義参照の解決や過去のLock一致とは別 | 権限確認、誤対象の防止、必要な結果照合を読み取りの利便性と引き換えに外さない |

`plugin-lock` と `plugin-resolve` は現在 `cmd/tei/cli_result.go` でWorkspace全体を検証してから実行する。最小の改善候補は、既存コマンドの成功条件を変えることではなく、必要なPlugin・環境の選択結果と、Workspace全体が未検証である旨・元の診断を別に返す**任意のread-only診断**を比較すること。その結果からLockの正式利用やRuntime起動を許可しない。依存する意味定義を必要とするPlugin、未解決の環境、改変されたPlugin定義を誤って有効としないことを実証できなければ、候補を採用しない。専用CLIや新たな正本が必要かも未決である。

**診断の試作**：Draftでは `plugin-resolve -diagnostic-candidates` を明示した場合だけ、全体の診断と、既存Resolverが返した候補を別に表示する。常に `ok: false`、終了1、`workspaceValidated: false` とし、候補は意味受理・実行許可・Runtime Planではない。環境不明、Plugin欠落、Lock不一致では候補を返さない。Workspaceの読み込み自体に失敗した場合は診断を続行できない。未検証の意味がPluginに必要かをこの試作は判定しないため、候補があっても使用可能性は不明である。通常の `plugin-resolve` と `plugin-runtime-plan` の検証・終了条件は変更しない。実Providerや自動化利用者への影響は未確認であり、採用判断は行わない。

追加の負例では、application-start Executorを `sales` の意味を使うApplicationへ結び付け、Application側の型参照を壊した。実CLIは通常のPlugin解決とRuntime Planを拒否し、診断モードだけが候補と元の意味エラーを未検証として返す。これは対象ApplicationとExecutorの接続を使った合成例であり、Plugin実装がその型を実際に消費した証明ではない。現行 `plugin.Definition` に業務意味依存を直接宣言する欄はなく、正式Role／Language Pluginの意味依存を検証できたとは扱わない。

**読み取り専用の比較経路**：Reference toolの `compare-application-membership -before DIR -after DIR` は、二つのWorkspaceから厳密なApplication差分と利用集合の候補差分を別々に表示する。片方でもValidation errorがあれば元診断を返し、候補を表示しない。SourceやDecisionは書かず、既存のPreview・Review・Acceptanceの計算にも接続しない。実CLIの一時Workspaceで、名前空間利用順の入替え、利用の削除、未定義名前空間を比較した。候補はTargetの表示順や依存網羅性の検証ではなく、異なる時点の読込み間の原子性も保証しない。実利用者がこの比較で負担を減らせるかは未測定。

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
