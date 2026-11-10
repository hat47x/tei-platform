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

- [ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)は**採用済み設計方針**。CLI/MCP/Web/Desktopは型付きApplication APIの別Adapterであり、CLIをHTTP常駐Serverへ変えない。開発用MCPと生成アプリMCPは別Scope。[main #3458](https://github.com/hat47x/tei-platform/pull/3458)のvalidate共有投影に加え、[main #3551](https://github.com/hat47x/tei-platform/pull/3551)はMCP未知引数・型不正の拒否、[main #3563](https://github.com/hat47x/tei-platform/pull/3563)はCatalogのNAMESPACE_NOT_FOUND共通化、[main #3582](https://github.com/hat47x/tei-platform/pull/3582)は4種read-only操作のCLI/MCP応答・失敗・入力検査の限定実測を追加した。**4 Tool参照Adapterの局所一致≠全Operation／実MCP/HTTP/Studio Transportの同値性・現Host Grant・実Receiver完受**。
- [main統合PR #3482](https://github.com/hat47x/tei-platform/pull/3482)の実CLI **13応答とConsumer判別42件**は`semantic-capability:sales.order.submit`に必要なCapability/Application/Source等の**宣言存在と不備の検出**に限定される。具体Function→Query→実Provider→Rowset、未知Ruleの発見、業務Commit、完全Context Closureは別。
- [#167](https://github.com/hat47x/tei-platform/issues/167)では歴史的Referenceとして`order_entry.recent_orders`の実PostgreSQL raw resultと明示列mappingが扱われている。これは**実DBからrawを取得した能力**の既存参照であり、現在の選択FunctionからそのQueryを呼び、同一AttemptでMapper→Function Data Context→CLI/Studioに届けたという新しい証拠ではない。Product `public.orders`と隔離Reference `public.tei_web_scoped_orders_reference`のScope・母集団を混同しない（[#2586](https://github.com/hat47x/tei-platform/issues/2586)）。
- [main統合PR #3479](https://github.com/hat47x/tei-platform/pull/3479)の負例により、**Imported Providerのraw Source/Manifestが変化してもConsumer World/Source Index Fingerprintは不変になり得る**。したがってこの二Fingerprintだけを実Provider Sourceの鮮度や現行資格に使わない。[Draft #3524](https://github.com/hat47x/tei-platform/pull/3524)のsame-Load private witnessは現在**未統合の候補**であり、複数Sourceの原子的Snapshotや全依存閉包を保証しない。
- [main #3381](https://github.com/hat47x/tei-platform/pull/3381)の局所二Consumer seamに続き、[main #3540](https://github.com/hat47x/tei-platform/pull/3540)は通常CLI dispatchとStudio両Hostへ**静的参照Provider**を配線し、silent builtin fallbackを拒否するS2-referenceを成立させた。ただし同じGo関数・Host Binary digestは正式Pluginの実Link/Load/置換・案件Profile・独立Host Grantを証明しない。D1の次の最小差分は**S2-productの実Provider差替えと実Target Conformance**であり、既存Selected Service/Adapterを再実装しない（[#1251](https://github.com/hat47x/tei-platform/issues/1251)）。

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
- **REAのExternal Observation**：[#3394](https://github.com/hat47x/tei-platform/issues/3394)のSource/Evidence→Structural Candidate→**Owner受理**→同一TEI MeaningへのProjectionを区別する。外部逆解析の一致を業務Meaning受理に昇格せず、SHOへの引用・Reader・正式書込み／競合制御・再開はSHO側の別資格とする。REAやSHOのWriterをTEI Coreへ移さない。外部Data Provider/Consumerは既存の型付き境界で接続し、それぞれの保存・配備方式をTEIの標準要件として固定しない。
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

## 7. 2026-11-09 追補：局所成功を製品接続の「不足辺」へ還元する

### 7.1 現mainで成立した範囲を正しく引き継ぐ

前章のD1/R0を置換するのではなく、その**最初に欠けた接続**だけに実装と検証を収束させる。新しいPhase/Gate/Registry/API/Fixture正本を作らない。

| 実装・観測 | 本当に成立したこと | まだ成立していないこと／担当 |
| --- | --- | --- |
| [merged #3381](https://github.com/hat47x/tei-platform/pull/3381)、[#3540](https://github.com/hat47x/tei-platform/pull/3540) | Selected Inspectorの局所二Consumerに加え、静的参照Providerが通常CLI＋Studio両Hostへ配線された（S2-reference） | **S2-product**：案件Profileで正式Pluginを差替え、実Target Link/Load/Conformance・現在Grant/Source/Lockを独立検証。#1251/#951/#1158 |
| [merged #3551](https://github.com/hat47x/tei-platform/pull/3551)、[merged #3563](https://github.com/hat47x/tei-platform/pull/3563)、[merged #3582](https://github.com/hat47x/tei-platform/pull/3582) | MCP不正引数拒否、Namespace不在失敗、4 read-only ToolのCLI/MCP成功・失敗・入力の限定比較 | 全Discovery・Transport・Host権限、外部Harnessへの完全配送、現在のDisclosureは未資格。#3420/#3419/#3262 |
| [merged #3567](https://github.com/hat47x/tei-platform/pull/3567) | Go Generator候補の宣言SourceがBuild対象外なら拒否 | ScriptのCheck→実Interpreter ReadのABA、全Build member/実Tool/実byteの閉包、現在のSource Grant。#2595/#2514 |
| [merged #3579](https://github.com/hat47x/tei-platform/pull/3579) | Function定義書へHook宣言を位相順で追加。生成規則Fingerprintも変化 | 同位相のPlugin合成順、実Hook RunnerとProvider/Effect、旧Artifact再資格化。#3503/#1127/#1803/#2600 |
| [merged #3588](https://github.com/hat47x/tei-platform/pull/3588) | C-2 service carrierを**非試験の参照用入口**から組み立て可能と実証 | 製品通常Consumerから選択したC-2実Call、実Provider/権限/EffectとC-1対照。#1251/#951 |
| [merged #3584](https://github.com/hat47x/tei-platform/pull/3584)、[#3654](https://github.com/hat47x/tei-platform/pull/3654)、[#3661](https://github.com/hat47x/tei-platform/pull/3661)、[#3664](https://github.com/hat47x/tei-platform/pull/3664) | child Plan申告の突合、型付きResolveの移管、recipe/source/dependency/parametersの宣言対Plan照合。別builderでのL2-aは[merged #3587](https://github.com/hat47x/tei-platform/pull/3587) | Plan一致≠実入力/成果物の帰属。Builder要求→解決Identity、Secret/Authorityは未決。L2-bは未成立。**cmd/teiの限定3段L3は既存実測済み**だがG4製品資格ではない。#2831 |

**Discoveryの別問題**：[merged #3590](https://github.com/hat47x/tei-platform/pull/3590)はCLI広告とCLI実受付の整合を調べたが、CLIとMCP tools/listのEffect/Version/Annotation保証はまだ別試験。CLI read-only広告であっても、viewの名前付き出力等のファイル作用が存在し得る。参照MCPのTool registration、CLI実行、Tool reply受領、外部Harness完全配送は異なる合否とする。

### 7.2 同じ対象を束縛しつつ、独立に判定する五つの証拠

D1とR0のAttemptは**別々の試行**でよい。完全性は無制限の全Workspaceではなく、明示した対象・用途・Scopeに対するRequired集合で決まる。別の中心台帳を作らず既存BOM/Trace/Scenarioに情報を結ぶ。

| 軸 | 固定して確認する条件 | 誤合格にしない条件 |
| --- | --- | --- |
| **Source／Authoring** | 管理Root、Reader/Parser、原byte/Span、Imported Provider Manifest・実Load、declared/discovered/loaded/parsed/consumedの集合 | 未読込tei.md、Source Index一致、World FP不変を実Read・freshnessにしない |
| **Meaning／Binding** | Function→Query/Relation/Provider、identity、NULLと欠損、金額・単位・粒度、選択epochとCurrent Scope | 型名一致やQuery ID一致で意味受理・実呼出しを推論しない |
| **Build／Generation** | 実Source/Toolchain/Generator/Lock/Artifact member、実Go選択、選択外混入、Rule版 | manifest Sourceやbuilderの自己申告Plan一致で実BinaryやInput Closureを証明しない |
| **Delivery／Receiver** | CLI/MCP/Studio実入口、mode/schema、Required collection/field、part番号・digest・終了標識、実受領byte | exit0・Tool success・HTTP 200・切断されたJSONを完全配送としない |
| **Authority／Effect** | Host Grant/expected state/対象、実Provider副作用、Receipt、独立read-back、現在の開示資格 | UI confirmed、Graph表示、Build成功、過去のArtifact同一から業務Commit/現在許可を推論しない |

入力を持つHarnessでは、stdoutとstderrの混入、Tool Replyの途中切断、部分ページと未送信ページを別記録する。Requiredの分母が未確定なら、見つかった対象だけで100% Coverageを算出しない。非公開ScopeのID・件数・Raw・Hashの開示もHost Policyに従う。

**Build provenanceの未決**：[Issue #3558](https://github.com/hat47x/tei-platform/issues/3558)で、-buildvcs=falseの実行物がversion -jsonからGit commitを自己証明できないことを実測。外部注入された申告Revision・実Binary digest・checkout SHA・実Processが消費したSourceを別の証拠として扱い、自己申告版をVCS実測と同じ資格へ自動昇格させない。

### 7.3 P0の並列着手とOwnerへのExit

- **P0-S／Source Admission**（#3262/#2621/#2595）：実Loader・Source/Parser/Readerと必要集合を固定する。宣言を読めることと、Functionの実Readができることを分ける。Script/Markdownの受理範囲、Input ABAとSource-only import改版を負例にする。
- **P0-M／read-only Model Map**（#1124/#1125/#1126/#1127）：同じMeaningからData/Function/Business/組織のRole Projection、CRUD/Relation/Pattern、Functionが所有するRepresentation occurrenceとDataへのBindingを導出する。Presentation topologyとData topologyは別。多重辺・未受理関係・編集不能な逆写像を保持する。
- **P0-R／Function実Read**（#2586/#2506/#2622）：Function→Query→実選択Provider→実Rowset→明示Mapping→Data Context→Consumerまで一Attemptで結ぶ。物理Relation/SQL/params/Read Cohort/NULL・partial・0件・非出力条件と、受領したConsumer結果を比較する。
- **P0-P／S2-product検証**（#1251/#951/#1158）：[main #3540](https://github.com/hat47x/tei-platform/pull/3540)の通常CLI・Studio静的参照Hostを固定し、異なる正式Providerを案件Profileで選択する。実Target Artifact/Lock・Load/Invoke・現在Grantを別列にし、B selected→A invoked／黙ったBuiltin fallback／未選択Plugin init・secret漏出を負例にする。S2-referenceの再実装はしない。
- **P1-E／Effectと現在資格**（#1174/#1129/#2600/#2831）：Read成功とWrite/Commitは別。実read-back/Receipt/unknown commitを検証。セルフビルドはL1（外部Go Build）、L2-a（宣言・計画の受渡し）、L2-b（計画と実Builder入力・成果物の帰属）、L3（生成TEI自身→Stage2実行）を別Claimにする。**cmd/teiの限定L3は実測済み**。ただしL2-bおよびG4資格は未成立であり、L2-aのReceiptでL2-bを合格にしない。

この順序は**全領域の逐次Gateではない**。P0-SとP0-M、D1とR0は並列可能。P0-Rを開始するために、全Studio/MCP、全BOM版Matrix、実VUI、Ontology DSL、SHO正式Writer、外部Consumerの配備をHard Dependencyにしない。外部逆解析のLegacy Observed Actual、Owner Accepted Desired、Target Observed Actualは、[#3394](https://github.com/hat47x/tei-platform/issues/3394)でそれぞれ独立Expectedを持つ比較とし、ひとつの「移行成功」に畳まない。

### 7.4 今回の採用範囲と停止線

**採用する計画差分**：存在・宣言の試験をむやみに追加せず、既存の型付きApplication API／Plugin-first／Script-first Modelerに対し、actual entry・Provider・Source・Consumerまでの**最初の未接続辺を塞ぐ**。限定的に成立したGo Inspector・MCP引数拒否・Go Source選択・Function定義書Hook投影は重複実装せず、先行実績として計画に束縛する。

**今回新たに実行していないこと**：D1の正式Pluginへ差し替えるS2-product、R0実Function→実DB→Consumer同一Attempt、実外部Harness全ページ完全配送、完全なSource/Toolchain Closure、製品全体G4（限定L3および限定L2-b Claimとは別資格）、REAからの正式意味受理とSHO正式保存、2×2 BOM全exact tuple、Ownerの業務意味受理。Draftの観測とmainの実績を混同せず、現行ソースとの再照合が必要になれば個別Ownerへ差し戻す。

## 8. 2026-11-09 追加照合：限定L3・Build Plan・契約実値の受入上限

本節は既存の#2831／#3657／#3503への引渡しであり、第二の製品Gate、Owner、Schema正本、Provider、受入Engineは設けない。

### 8.1 セルフビルドを実測に沿って4段階で区別

[既存L3実測](../evidence/TEI_SELF_BUILD_CMD_TEI_L3_2026-11-08.md)では、外部Go seedがstage0を作り、stage0のTEIがstage1を生成し、**stage1のTEIバイナリ自身がstage2を駆動**した。stage1/2の実Artifact byteも一致する。よって限定L3を「未実施」に戻さない。ただし同じchild・供給・Toolchainの一環境の実験であり、Source Closure rejected、Seed Trust未資格、entry自身のL3認定なし、G4 UNQUALIFIEDを併記する。

| 主張 | 確認した範囲 | 別に必要なこと |
| --- | --- | --- |
| **L1 実Go Build** | 実BuildとArtifact digest、stage0外部seedの存在 | 供給/Toolchainの現行資格と異環境比較 |
| **L2-a 宣言→Planの受渡し** | [#3584](https://github.com/hat47x/tei-platform/pull/3584)等の自己申告Plan一致・不一致を区別。[#3654](https://github.com/hat47x/tei-platform/pull/3654)でResolve portを内部へ移管 | 解決child/Builder双方の実読込byte、申告でない独立帰属 |
| **L2-b 計画→実Builder→Artifact** | [#3775](https://github.com/hat47x/tei-platform/pull/3775)で五条件の合成Claim、[#3785](https://github.com/hat47x/tei-platform/pull/3785)の参照Build実試行、[#3795](https://github.com/hat47x/tei-platform/pull/3795)のLinux/Windows二Host確認により**限定L2-b claimed=true**を実測。さらに[#3839](https://github.com/hat47x/tei-platform/pull/3839)は実cmd/tei Go BuildをBuildExecutionPlan/ArtifactSetに束縛。 | Claimの対象は指定参照Catalog/実行物/ホスト。Builder要求と解決Identity、Secret/Authority、Catalogの出典・供給Trust・全Target/G4は別資格。現製品の既定動作で常にClaimが立つわけではない。 |
| **限定L3 実TEI→次のBuild駆動** | stage1生成物＝stage2実Invoker、両段Artifact byteとRecipeの限定一致を記録 | 成立範囲を保存。L2-b・G4の不足と無関係な資格へ拡張しない |

**方式採用を先行させない**：[main #3655](https://github.com/hat47x/tei-platform/pull/3655)/[#3656](https://github.com/hat47x/tei-platform/pull/3656)の調査では、Builder catalogの実製品データソースは未存在。in-process解決を既定へ切り替えるには、fixtureではないCatalogの出所とTrustが必要。現行のchild経由と、既存Resolverのport/fixture分離を維持する。[#3660](https://github.com/hat47x/tei-platform/pull/3660)が実測したCLI Catalog 19件は読取り向け部分集合であり、dispatch全50件と同一ではない。

### 8.2 宣言型と、次段が現実に受け取る値を分ける

[main #3658](https://github.com/hat47x/tei-platform/pull/3658)は参照能力連鎖の合成Binding出力と宣言型の過不足、値のキー受渡しを検出可能にした。これは**既知の不一致を記録した限定試験**であって、宣言と実Bindingが適合したことではない。実HTTPの202受付結果はTaskの受付を表すが、次段が要求するAttempt識別子を返さない。live実行や正式Mappingの採用は未実施であり、[#3657](https://github.com/hat47x/tei-platform/issues/3657)の閉鎖も実連鎖の製品資格とはしない。

D1/R0の型付き縦断では、**Declared Output → actual Binding Output → Next Required Input → Receiver**を別の観測行で照合する。ID・型名・起動順・HTTP受付・JSON構文が一致しても、必須フィールドのPresenceと値の受渡しは独立に検証する。

| 追加負例 | 判別条件 | 担当 |
| --- | --- | --- |
| **XV-09 宣言とBinding出力の不一致** | 必須属性の欠落・未宣言属性の混入を検出。既知許容リストは適合／受理ではない | #1016/#2410/#1180 |
| **XV-10 次段の入力不足** | 合成Providerでは受け渡せても実HTTP受付が次段の識別子を返さない。受付成功と業務完了を分離 | #3658、各Host/Adapter Owner |
| **XV-11 Hook未評価** | [main #3635](https://github.com/hat47x/tei-platform/pull/3635)のStudio参照Hookはbefore系の一部を呼び、入力不在はnotEvaluated、after系はskipped。実Rule Plugin／実DB／Wails製品画面の完了に昇格しない | #3503/#1127/#1824 |
| **XV-12 論理分割の仮定** | CRUDの既知direct CUDと制約に基づくSubsystem候補を、実Effect／Transaction、物理配置、正式Owner受理に昇格しない | #3250、[Draft #3662](https://github.com/hat47x/tei-platform/pull/3662) |

### 8.3 次の引渡し・状態

**#1251/#2586**：既存Inspector二ConsumerとFunction read-firstの各Scenarioへ、実出力の必要属性と次Consumer受領を1行追加し、欠落時は不合格。**#2831**：限定L3の実測を保持しつつ、L2-bの実入力閉包・Builder Identityの意味規則・Authorityを前進させる。**#1124/#3503/#3250**：read-only Model Map、CRUD/Function/RoleとHookの未評価状態を同じSourceへ投影する。論理分割の自動採用は禁止。**#2514/#3262**：Required集合に次段必要属性、Tool Resultの完全受領と現在Disclosureを含める。

**本節は計画反映のみ**。XV-09〜12の新しい製品E2E、L2-b/G4、外部live Provider、Studio/Wails実画面、2×2 BOM、Ownerの正式意味受理はNOT RUN／NOT ACCEPTED。各既存Reference/CIのPASSをその対象外へ一般化しない。

## 9. 2026-11-10 現行main：製品に届いた経路と次の不足辺（旧節の状態表より優先）

この節は受入判定の**基線更新**であり、#1251／#2586／#2831／#1124／#1180などのOwnerを移さない。各領域の詳細な設計・実装記録はリンク先を正本とする。

### 9.1 到達点と、新たな製品出口

| 縦断 | mainに統合された限定成果 | 次の実証だけを受け入れる条件 |
| --- | --- | --- |
| **D1 Plugin-first** | [#3540](https://github.com/hat47x/tei-platform/pull/3540)が通常CLIとStudio Hostに静的参照Providerを配線した。**S2-reference成立**。 | 案件Profileで**別実装B**を選択→実BinaryでLink/Load/InvokeしたBを観測。A選択／B実呼出しやfallback、未選択実装の混入・権限越境を拒否。S2-productは未資格（#1251/#951/#3267）。 |
| **R0 Function→実Read** | 既存実PG raw取得と[#3262](https://github.com/hat47x/tei-platform/issues/3262)の宣言Requiredは別々に進展。 | 一FunctionのInvocation→Query→実DB Provider→raw/明示Mapping→Context→**実Reader受領**を一Attemptに束縛。母集団、NULL/Presence、partial、非出力WHERE/否定依存、Scopeと独立Expectedを比較。**未実証**（#2586）。 |
| **Hook/Operation** | [#3635](https://github.com/hat47x/tei-platform/pull/3635)のStudio参照評価に続き、[#3761](https://github.com/hat47x/tei-platform/pull/3761)は宣言外の結末名を拒否、[#3843](https://github.com/hat47x/tei-platform/pull/3843)は宣言usesのCapabilityをCLI/Studio参照Callerから実呼出しし、呼出し結果を保持。 | 入力のreads→Capability引数対応、**実Rule Plugin**／実Provider・Effect／after-*、Wails製品E2Eは別試験（#1127/#3503/#1180）。「calls.called=true」は業務Commitではない。 |
| **Build/Self-build** | [#3775](https://github.com/hat47x/tei-platform/pull/3775)/[#3785](https://github.com/hat47x/tei-platform/pull/3785)/[#3795](https://github.com/hat47x/tei-platform/pull/3795)で限定L2-bの五条件Claimを実試行・二Hostに固定。[#3760](https://github.com/hat47x/tei-platform/pull/3760)は呼出し元指定Catalogによるentry内Resolve。[#3839](https://github.com/hat47x/tei-platform/pull/3839)は実cmd/teiのGo Build・ArtifactSetと入力閉包を限定接続。先行の**限定L3**も保持。 | 実Builderの全入力Closure、Catalog出典・Trust、Secret/Authority、配布Profileの現行資格が残る。**G4=UNQUALIFIED**。追加再Buildは実Go測定[#3818](https://github.com/hat47x/tei-platform/pull/3818)で約1.93倍の費用。強化既定は[#3844](https://github.com/hat47x/tei-platform/pull/3844)/[#3849](https://github.com/hat47x/tei-platform/pull/3849)の**未統合PR**の範囲を確認し、全試行に強制しない。 |
| **外部Source/CLI/Consumer** | [#2812](https://github.com/hat47x/tei-platform/pull/2812)は選択済みProviderの**同一Loadで実Parserが消費した原文/Manifest**をprivate witnessに固定。[#3829](https://github.com/hat47x/tei-platform/pull/3829)はMCP入力schemaから許容引数を導出、[#3846](https://github.com/hat47x/tei-platform/pull/3846)はtrace offset打切りを明示。 | 公開可否/Source全閉包/実Provider invokeは未保証。CLI Tool SuccessやScenario Verifiedは実DB・受領完了の証拠ではない（[#2830](https://github.com/hat47x/tei-platform/pull/2830)）。#2516/#3262/#3420。 |

### 9.2 P0の採否を決める二つの小さなPacket

新規Harness/中央Ledger/第二Expectedを作らず、既存Scenario・Artifact・Traceの同じAttempt参照を相互に結ぶ。D1・R0は**別Attemptとして独立に完了可能**。

**D1 Product Packet（#1251）**：Workspace/Host Principal/Grant、Application Operation/Subject/Revision、Profile/Contract/Selected Provider ID+version/実Binary digest、Loaded/Invoked Providerと結果、許可範囲、実Consumer受領、B→Aすり替え負例、独立Expected。S2-referenceのGo関数が呼ばれた証拠だけなら**reference PASS／product NOT RUN**と表示。正式Provider差替えの実テストが無い状態でBOMをverifiedとしない。

**R0 Read Packet（#2586）**：FunctionとQueryの**実Binding**、native SQL原byte/展開byte/物理Relation/Dialect/params、actual Provider/Read Cohort、raw Rowsetと明示Mapping、Context grain/Source/partial・NULL/0件、実Reader受領byte/ページ終端、独立ExpectedとDisclosure。Function/Query単独Inspect、実PG Queryだけの成功、途中で切れたTool Replyは**Product R0 PASSに足さない**。

両Packetで必須項目が未観測なら該当軸をunknown/not-runに保ち、実施した負例が不正ならFAILにする。成功率の分母は独立に固定したRequired集合であり、「表示できた要素数」を全体母数と推測しない。

### 9.3 横断の後続：設計ガイド→業務効果→性能と普及

- **図面は全体の案内**：[main #3847](https://github.com/hat47x/tei-platform/pull/3847)のRQ-036/MAP-17〜23を引き継ぎ、概観を保持した現在地→詳細→復帰、人間用図表とAI/CLI/MCP用の**同一Source由来の構造化投影**を比較する。現在地/レイアウトは表示状態、Accepted Meaning/Sourceは一つ。正式Graph Storeや第二Writerを作らない（#1124）。
- **実作用は独立の次段**：[Draft #3854](https://github.com/hat47x/tei-platform/pull/3854)はRunEffect/Compensate/RunPlanを実PostgreSQLで試験し、Succeeded/Observedの分離や補償失敗の保持を限定観測した候補。PR未統合で、Host現在Grant・本番相当Provider・必要Checkの製品保証／G3合格ではない（#1129/#1130/#1174）。
- **最初の性能比較**：[main #3856](https://github.com/hat47x/tei-platform/pull/3856)のNo Cache対Request-local/Memoryを、**R0実Readが成立した同一Function/Provider/Workload/独立Expected**に限定して比較。Cold/Warm、同時Miss、失効競合、Origin障害、Tenant Grant/partial・0件を負例にし、正しさが未検証なら速度を有効改善と数えない（#3271/#2600）。
- **製品価値と汎用性**：[#3796](https://github.com/hat47x/tei-platform/issues/3796)の中小規模案件の標準構成受入、[#3791](https://github.com/hat47x/tei-platform/issues/3791)の物流・GIS等の対照を既存Product Scopeに接続する。最初のD1/R0に分野全域の対応を前提化せず、成立/既存資産接続/新Plugin要/未成立を実測で数える。AIが作る複数Pluginの個別PASSを組合せPASSにしない。
- **外部実行**：[main #2787](https://github.com/hat47x/tei-platform/pull/2787)のEKI OpenAPI固定対照・[#3658](https://github.com/hat47x/tei-platform/pull/3658)のOutput→Next Input不一致を利用。TaskAccepted・Task完了・独立Artifact Read・TEI業務Commit・Owner Meaning Acceptanceは別。EKI/TAI/SHO/KON等のOwnerをTEI Coreへ移さない。

**更新した計画の停止線**：正式Plugin ABIやCatalogの置場、G4の製品受理、RQ-036全図面完成、実Effect/本番性能、Owner意味受理を本書で採用済みにしない。§3〜8の古い「未実装」は歴史的基線であり、現在の合否は本§9とリンク先の**exact revision/実行証拠**を優先する。本追補自身は**文書変更のみ**で、新しいD1 S2-product／R0実Read／G4全体／外部Consumer E2Eを実行していない。
