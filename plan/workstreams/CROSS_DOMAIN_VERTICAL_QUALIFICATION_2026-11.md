# 横断機能を一つの実使用経路で検証する計画（2026-11）

- **状態**：既存計画を接続する実証順序の提案。新たな製品Gate、ABI、意味正本、Runtime Host、Resolver、Registry、Schedulerは定義しない。
- **位置づけ**：各領域の詳細設計は既存文書と担当Issueが正本。この文書は重複実装を減らすために、**同一の使用経路を何がどこまで証明したか**を横断して確認する。
- **基準日**：2026-11-08。本文の検証例は未実行の提案であり、既存の限定ReferenceやCI成功を製品全体の完了へ拡張しない。
- **参照**：[製品計画](../MASTER_PLAN.md)、[Core以外Plugin化の実装計画](../design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)、[操作面の導出](../design/OPERATION_SURFACE_DESIGN_2026-11.md)、[性能・AI最適化](../design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md)、[API失敗とClient反応](../design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)、[増分再利用の実証](INCREMENTAL_REUSE_EVIDENCE_EXECUTION_PROTOCOL_2026-11.md)。

## 1. 統合判断

TEIが共通化するのは画面やDomain固有の工程列ではなく、**意味・対象・型・権限・副作用・証拠・変更時の検査**である。Coreは小さく保ち、その外側のLanguage／Operator／Data Provider／Generator／CLI・Studio Adapter／標準表現等を契約に基づく選択可能なPluginとして扱う。標準Rendererを利用しないHeadlessなアプリケーションも、標準的な利用形態とする。

一つの操作に対して、少なくとも次を**別の事実**として扱う。

1. **意味・契約が記述された**：Source occurrence、Accepted Meaningと未受理Candidate、型／NULL／単位／粒度、OperationとScopeを追える。
2. **実構成で選択された**：Capability候補一覧とEffective Selection／Lock／Plugin実装版／Providerの適合を区別する。
3. **実Buildに含まれた**：実入力閉包、Toolchain、Artifact Revision、リンク後の物理混入・非干渉を検査する。
4. **実Consumerから呼び出された**：通常CLI、Studio、Web等のどの入口かを示し、実際に選択Providerへ到達した証跡を持つ。
5. **結果・副作用を独立に観測した**：Result、外部Read、Commit/Receipt、拒否、Commit Unknownを分離する。
6. **現在のConsumerに適格である**：Source・Plugin・Grant・Disclosure・Provider状態・必要Artifact集合の鮮度を評価する。
7. **独立Expectedで受け入れた**：正常系だけでなく部分取得、誤選択、失効、失敗・再試行を照合する。Ownerによる業務意味受理とは別に記録する。

これは新しい七段階Gateではなく、既存BOM／Source Evidence／Consumer資格を同じ試行に対応付ける**説明用の観点**である。「宣言された」「選択された」「CIで限定PASS」「実Providerで作用した」を一つのDoneへ潰さない。

## 2. 共通の実行・図面契約

標準／案件カスタム／合成後の実効状態を、**型付きOperation Lifecycle Graph**として比較する。Intent → Prepare → Validate → Authorize → Execute → Observe → Resultは説明用の基本系列であり、固定実行順序や一律の同期処理を強制しない。GraphのSlot／Hook／Guard／分岐／サブフロー／Provider交換は構造化したPatchとして表し、型・権限・副作用・受理境界を検証してから生成に使う。図面のレイアウト変更は意味変更ではない。

**Interaction Flow／Business Flow／Adaptive Flowを分ける。** AIチャットのStreaming・Tool Call・Approval・Branch、音声入力の無音・中断・再確認、独自B2C FrontendのJourney／ExperimentはInteraction側の異なる事例であり、Business Effectを自動的に承認しない。標準表現型はFunction内の表現インスタンスに配置し、Data topologyとPresentation topologyのBinding、読み書きの可逆性、情報の欠落を明示する。編集できない投影をEditableと表示しない。

図面からの生成は説明用ダイアグラムのHTML化ではない。最低限、`Source→Typed Graph→Validated Patch→Effective Graph→Target Artifact→実Consumer結果`を追い、Source・Graph・生成器・Targetの版を束縛する。図面の全容性はRequired／Loaded／Unresolved／Hiddenを区別し、未取得部分を勝手に補わない。

## 3. 重複を避ける最小縦断試験

| 優先 | 単一の試行で確かめること | 必須の負例・独立Expected | 既存担当 |
| --- | --- | --- | --- |
| **P0-A** | 通常配布形態のGo CLIから一OperationをInspect→Resolve→Invokeし、明示選択したPlugin B／Provider Bへ実際に到達する。通常のBuildと実呼出しを同一実装版で照合する。 | Bを選択したのにAを呼ぶ、暗黙fallback、未選択Pluginの実混入、契約だけ合うが実体不在、権限不一致。 | #1251／#951／#1016／#3267 |
| **P0-B** | P0-Aと同じSource・選択・Attemptを使い、CLIのCatalog/Inspect/Trace/Validate、BOM、Artifactを突き合わせる。 | 0件／未読込Source／同名別内容／実消費Source差、選択前の候補と実選択の混同。 | #2595／#2600／#1180／#2514 |
| **P1-A** | 一つのFunctionからQuery→Dialect→実DB Provider→Result→Contextへ進むread-only試行。既存SQLの直接利用も許容し、TEI専用SQLへの強制変換はしない。 | NULL／欠損／0件／partial／permission-denied、JOIN・非存在条件などの否定依存、Provider切替後の実Readの誤認。 | #1126／#2582／#2622 |
| **P1-B** | 既存Capabilityの検証→認可→書込み→外部Observation／Receiptを一Attemptで分離して記録する。 | business reject、technical failure、commit unknown、重複要求、CAS競合、前段成功のみでCommit成功を主張する誤り。 | #1174／#1129／#3274 |
| **P1-C** | 同一Meaningからread-only Model Map、三視点図、CRUD Matrix、標準/カスタム/実効Lifecycleを導出し、変更案のImpact→Accept→再生成を限定一件で照合する。 | 未受理関係の自動採用、同label別identity、多重辺消失、レイアウト変更をMeaning更新扱い、非可逆投影からの無断書戻し。 | #1124／#1125／#1126／#1127／#2497／#3503 |
| **P2-A** | 実Consumerを変えても意味・Operationの契約が同一に扱えるか、完全Headlessの独自UIとStream/Approval型Interactionで圧力試験する。 | 必須TEI DOM・Web Runtime・追加hopの強制、UI操作の成功を業務効果と混同、UI変更に伴う無関係なMeaning再受理。 | #3509／#3503／#1158 |
| **P2-B** | P0/P1で資格が確認できた同じ入力の変更系列をCold／Warm／Incrementalで反復し、提案→Preview→Verify→Accept→Apply→再観測を比較する。 | Plugin候補だけの変化と実Selection変化、A→B→A、誤再利用、不要な全面失効、速度改善したがGrant／Source資格が古い。 | #2600／#3271／#2018 |

**先行順**：P0-Aの「選択B→実B呼出し」が最初の停止条件。P0-Bは同時並行でよい。P1-A/B/Cは独立して進められるが、選択・実入力・実Consumerの証拠が欠けた部分は `unknown` のまま保持する。P2は比較用の異質事例であり、P0を妨げない。

## 4. 同じ試行を結び付ける記録

個別のOwnerが持つ既存ログ／BOM／Trace／Artifactを、次の項目で対応付ける。新たな中央台帳や第二正本は不要。

| 確認の単位 | 記録する最低限の情報 |
| --- | --- |
| 試行・入力 | Attempt/Scenario ID、独立Expected版、World／Source Scope、Source occurrenceとRevision、実Readの完全部分 |
| 解決・Build | Contract/Capability/Plugin/Providerと実選択の版、Build／Toolchain、実入力閉包、Artifact集合のRevision |
| 動作・作用 | 入口Consumer、Operation/Function/Query、実Invoke先、認可Scope、Result、外部Read／Effect／Receiptの観測時点 |
| 現在資格 | Source fresh/stale/unavailable/partial等、Consumer必要member、Grant、Disclosure、適用できる対象範囲と未確認範囲 |
| 判定 | 実測PASS／FAIL／NOT RUN／UNKNOWN、機能・意味・観測・性能の別判定、Ownerの受理状態 |

外部情報が移動して内容が同一の場合にも、計算済みMeaningの再利用可能性とSource位置・引用・現在権限の再資格化を別に扱う。Artifact bytes一致は実Providerの選択やOwnerによる受理を証明しない。現存するResultと、現在そのResultを利用できるかも独立に判定する。

## 5. 既存計画への組込みと採用判断

- **#1119の主線を変更しない**。横断試験は既存の三視点ローコード一往復に重ねるが、Gate A/B/Cの過去のReady判定を変更せず、新たな必須Gateも加えない。
- **Plugin方式**は#951／#1016／#1251／#3267の既存Core／Capability／Composition／BOMへ返す。静的合成・動的ロード・外部Processの選択は、Go単一バイナリと外部Targetの要求に照らして実測後に決める。全部同じロード方式にはしない。
- **図面／操作／Studio**は#1124〜#1127／#3503の既存Design・Target Consumerに返す。DiagramとRuntime Graphは必ず構造上の同一性と変更差分を検査し、画面を新たな業務意味の正本にしない。
- **Query／Cache／最適化**は既存DB/Source/Provider・BuildのOwnerへ返す。Optimizing Pluginは結果の正しさを先に証明し、その後にp50/p95、再Read数、Cold/Warm差、実CPU/待ち時間を比較する。
- **B2C／チャット等の異質Consumer**は標準表現型の強制対象ではない。共通契約を使えることと、専用Frontend・外部Providerを自由に選べることを共に守る。

**改定条件**：P0-Aで実Providerへ到達しない、P1で独立Expectedが失われる、P1-CでGraphから生成できず別の意味定義が必要になる、P2で独自UIに不要なRuntime負荷が生じる、またはReuseで誤資格付与が見つかる場合は、該当Ownerの契約・要求・選択方式へ戻す。証拠が揃わない領域を新Componentの導入で覆い隠さない。

**この更新の限界**：ここで列挙したP0〜P2の同一試行、実DB Effect、図面駆動Target生成、異質Consumer完全受領、定量的な高速化、Ownerの意味受理は**今回未実行**。既存文書の限定試験成功と、今後成立させるべき製品受入を混同しない。

## 6. 2026-11-09 追補：共通Application APIと単一Function実Readを二本の独立した縦断試験に収束

### 6.1 現行mainで先に使う事実と、証明していないこと

- [ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)は**採用済み設計方針**。CLI/MCP/Web/Desktopは型付きApplication APIの別Adapterであり、CLIをHTTP常駐Serverへ変えない。開発用MCPと生成アプリMCPは別Scope。現状の実装適合は[main統合PR #3458](https://github.com/hat47x/tei-platform/pull/3458)の`validate`正常・Load失敗・重複宣言の限定投影比較などであり、全操作／全Transportの同等性・実Host Grantは未証明。
- [main統合PR #3482](https://github.com/hat47x/tei-platform/pull/3482)の実CLI **13応答とConsumer判別42件**は`semantic-capability:sales.order.submit`に必要なCapability/Application/Source等の**宣言存在と不備の検出**に限定される。具体Function→Query→実Provider→Rowset、未知Ruleの発見、業務Commit、完全Context Closureは別。
- [#167](https://github.com/hat47x/tei-platform/issues/167)では歴史的Referenceとして`order_entry.recent_orders`の実PostgreSQL raw resultと明示列mappingが扱われている。これは**実DBからrawを取得した能力**の既存参照であり、現在の選択FunctionからそのQueryを呼び、同一AttemptでMapper→Function Data Context→CLI/Studioに届けたという新しい証拠ではない。Product `public.orders`と隔離Reference `public.tei_web_scoped_orders_reference`のScope・母集団を混同しない（[#2586](https://github.com/hat47x/tei-platform/issues/2586)）。
- [main統合PR #3479](https://github.com/hat47x/tei-platform/pull/3479)の負例により、**Imported Providerのraw Source/Manifestが変化してもConsumer World/Source Index Fingerprintは不変になり得る**。したがってこの二Fingerprintだけを実Provider Sourceの鮮度や現行資格に使わない。[Draft #3524](https://github.com/hat47x/tei-platform/pull/3524)のsame-Load private witnessは現在**未統合の候補**であり、複数Sourceの原子的Snapshotや全依存閉包を保証しない。
- [#1251](https://github.com/hat47x/tei-platform/issues/1251)のCLI／Studio InspectorはBuiltin直結の通常入口とService注入候補を区別している。**同じ選択済みProviderをproductionの双方から実呼出しするD1**は未資格。Service test、Contract検査、同じ出力値だけでは`selected B → invoked B`を証明しない。

### 6.2 直近の二本の縦断。共通証拠は使うがAttemptを偽って連結しない

| 縦断 | 最小実験と観測する辺 | 最初の合格条件 | 停止と担当 |
| --- | --- | --- | --- |
| **D1 Inspector Plugin-first** | Hostが確定した`authoring-tool` Provider B → 同じ型付きInspection Operation → **通常CLIとStudio Hostそれぞれの実Entry**。Consumerごとにexact binary/Contract/Lock/実Provider callを固定。 | B選択時にBだけが呼び出され、Aの同値ResultであってもA実呼出しを拒否。CLIとStudioの外側JSON byteではなく、同じ対象・同じScopeのtyped結果を比較。 | [#1251](https://github.com/hat47x/tei-platform/issues/1251)、選択の[#951](https://github.com/hat47x/tei-platform/issues/951)/[#1016](https://github.com/hat47x/tei-platform/issues/1016)、Adapterの[#1158](https://github.com/hat47x/tei-platform/issues/1158)。D1のInspectorが実Provider Queryまで接続したと誤称しない。 |
| **R0 Function read-first** | *一つの対象Function*の実Invocation Binding → Query `order_entry.recent_orders`（適用する場合）→ Source/native SQL/Dialect/params/physical relation → **実選択Provider呼出し** → raw Rowset → 明示Result Mapping → Function Data Context →受領したread-only Consumer。 | 同一Attemptの実Rowset、対象母集団、列Presence・格納NULL、Mapperとsemantic ID、Context grain/scope/epoch/completeness、Consumer受領を独立Expectedと照合。別々に成功したFunction inspectとQuery inspectは不合格。 | [#2586](https://github.com/hat47x/tei-platform/issues/2586)統合Owner、[#2506](https://github.com/hat47x/tei-platform/issues/2506) Context、[#2622](https://github.com/hat47x/tei-platform/issues/2622) Provider、[#3262](https://github.com/hat47x/tei-platform/issues/3262) Required Read。 |

**D1とR0は最初から「同一Operationの同一Attempt」である必要はない**。対象と用途が違う試験のIDを合算せず、それぞれの`Scenario→Attempt→actual selected/invoked/source/received`を閉じる。最初に共有すべきなのは既存Contractの識別方式・独立Expected・Current Scopeの判定であり、巨大な統合Runnerではない。R0を最初に通すために、D1の全Studio移管・VUI・Ontology DSL・SQLGlot実Worker・SHO正式Writer・Verified BOM全行を必須条件にしない。

### 6.3 R0を判別力のある固定試験にする（新しい独立Fixture正本は作らない）

[#2586](https://github.com/hat47x/tei-platform/issues/2586)の既存HJ / B-Read+系列へ、次の観測を連携する。各Caseに独立Expected・対象Source/Query/Provider/Consumer版・実Runner・Receiptまたはread witness・秘密開示範囲を明示する。

| 反例 | 誤ってPASSにしがちなこと | 必要な判定 |
| --- | --- | --- |
| **XV-01 結合切断** | FunctionとQueryの個別InspectがPASS | Invocation Binding実在／型適合／実Readを別途確認、無ければ`not-bound` |
| **XV-02 Queryのすり替わり** | Query ID・列・行数一致、物理relation/params/Scopeが別 | 原SQLと展開後SQL、Source occurrence、Datasource、対象母集団、Provider実消費で区別 |
| **XV-03 Provider不一致** | B選択、A実呼出、結果が偶然同値 | `selected != invoked`でFAIL、Provider identityと実Code/Build Memberを照合 |
| **XV-04 値と列の不在** | DB NULL、`NULL AS status`、列未選択、Column missingを一つのnullにする | Field Presence、SQL projection、物理NULL、Mapping Gapを分離 |
| **XV-05 0件と部分集合** | RLS/権限/timeout/Limitで0件・部分取得→業務全件不存在と解釈 | Cohort、Read Scope、Complete/Partial/Unknown、否定依存のRequired集合を明示 |
| **XV-06 Source-only drift/ABA** | 外部raw変更時にConsumer FP不変、A→B→A復帰で旧受理を復活 | imported raw/Manifestの同一Load証拠とCurrent Source/Grantを再資格化。観測不能ならunknown |
| **XV-07 Tool受領切断** | CLI exit0／MCP Tool起動成功だけで完全配送とする | 部分番号、length、end marker、Reader受信byte/範囲、公開資格。途中切断をcompleteにしない |
| **XV-08 read-onlyからの誤作用** | 表示可能な集計値・Relation edgeから更新Actionを勝手に発行 | Intent候補とAuthoritative Action/Host Grant、単一Entity・Relation変更・業務Effectを分離 |

**成果物**：対象を束縛したread-only比較票と最小Failure Trace。P0は`XV-01/02`の現行Source静的照合、P1は実`B-Read+`の一試行、P2は`XV-03〜08`から代表負例、P3で別Consumer。静的照合のみの段階で「実DB read成立」としない。

### 6.4 派生図面・REA・BOMへの受け渡し（主線を増やさない）

- **三視点・組織・Lifecycle**：既存[#1124](https://github.com/hat47x/tei-platform/issues/1124)/#1125/#1126/#1127、[RQ-036](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)、[#3503](https://github.com/hat47x/tei-platform/issues/3503)のSource-first Role Projectionで、Data/Function/Business/組織の主体・権限・有効期間・State・Actionを**同一stable identityへの別Role**としてたどる。Script/DSL/Markdownを正本とし、CRUD表/Model Map/Graphのレイアウトを第二Meaning正本にしない。先にread-only Model Map→Gap→Sourceへnavigateを検証し、Graph Patch ABI・同一Sourceへのtyped edit/Save・Generator Targetは後続の固有Ownerで扱う。
- **REAのExternal Observation**：[#3394](https://github.com/hat47x/tei-platform/issues/3394)のSource/Evidence→Structural Candidate→**Owner受理**→同一TEI MeaningへのProjectionを区別する。外部逆解析の一致を業務Meaning受理に昇格せず、SHOへの引用・Reader・正式書込み／競合制御・再開はSHO側の別資格とする。REAやSHOのWriterをTEI Coreへ移さない。KON等は任意の外部Data Provider/Consumerであり、別製品の保存・配備方式を本書で規定しない。
- **Verified BOM/Atomic Trait**：[#3267](https://github.com/hat47x/tei-platform/issues/3267)/[#3279](https://github.com/hat47x/tei-platform/issues/3279)で`required Trait→actual fulfillment→exact Lock→Build→Invocation→独立Expected→Consumer`の各exact tupleを記録。Provider B1/B2×Consumer CLI/Studioの**4組すべて**が同じ資格を得ない限り「2×2 Verified」と称さない。SBOM、候補探索、Pairwise実行計画、過去のArtifact成功を互換性PASSと混同しない。BOM要求へ独立した`RQ-036`を割り当て直さない（同番号は既存Model Map要求）。要求番号の再採番はOwner側で扱う。
- **Everything is Data／DSL・Query**：[#2619](https://github.com/hat47x/tei-platform/issues/2619)/#2620/#988の標準データ・Meaning・Syntax Sugar/Source Fidelityを維持。Native SQLの利用も許容し、Queryの解析候補からNULL/集合完全性/金額精度・Domain意味を推測採用しない。日常DSLにGoの`struct/func/package/import`を露出する必要はない。Go単一バイナリを標準候補としつつ、外部Host TargetやPlugin単位の実装方式は実測で選ぶ。

### 6.5 計画更新後のOwner別完了票と保留線

| Owner | この横断計画から渡す一つの成果物 | PASSするまで保留すること |
| --- | --- | --- |
| #1251 / #951 / #1016 | D1同一Contractで**通常CLIとStudioの実選択→実Call**の2 Consumer 証拠、A/Bすり替え負例 | Plugin-first全体完成、全API/Transport parity |
| #2586 / #2506 / #2622 | R0 Function→Query→実Rowset→Mapping→Context→Read Consumerの**同一Attempt**と独立Expected | Context/Queryの局所PASSから実Function read成功を推論 |
| #3262 / #1158 / #3420 | required/loaded/bound/returnedと、参照CLI/MCPの受領完全性・開示・診断分類 | 参照MCPのTool登録／`validate`限定一致から外部Harness/全操作同値を推論 |
| #1124 / #3503 / #2497 | read-only Map、三視点Role・多重Edge・Source位置、標準/実効GraphのLoss/Gap | 永続Graph ABI・全図編集・実Target生成 |
| #3394 / #1126 / #2516 | 外部観測・構造候補・Owner受理・用途別Consumerの各Revision | 外部解析結果を意味正本へ自動採用 |
| #3267 / #3279 / #1180 / #2514 | exact tuple別Conformanceと未実施／不適合を含む4行Matrix | BOM資格、業務Effect、Host現在権限 |
| #2600 / #3271 / #2018 | D1/R0の実入力閉包と誤再利用を先に抑えたCold/Warm計測 | 実証のない高速化率や普遍的Cache仕様 |

**改訂時の停止線**：新しい標準要求番号、正式Plugin ABI、Lifecycle Graph永続形式、Ontology DSL/デジタルツインの実行Host、独自Harnessの導入は、この横断PRの決定事項にしない。異種Consumerへの配送能力、Contextの全容、外部Effect、Owner採用は個別証拠で判断する。**本節のXV-01〜08・D1/R0の新しい同一Attempt E2E・2×2 BOM・実Studio二Consumer・外部REA→SHOの正式保存はNOT RUN／NOT ACCEPTED。**
