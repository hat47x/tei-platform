# Function Read FootprintとProvider実アクセスの差を観測する（2026-11-02）

**状態**：設計・独立合成Reference候補。正式TEI Data Operations／Query Planner／Host Policyではない。
**主Owner**：[Data Operations #2622](https://github.com/hat47x/tei-platform/issues/2622)、[Function Data Context #2506](https://github.com/hat47x/tei-platform/issues/2506)。
**参照**：[Everything is Data #2619](https://github.com/hat47x/tei-platform/issues/2619)、[#2620 Role](https://github.com/hat47x/tei-platform/issues/2620)、[#2621 Authoring](https://github.com/hat47x/tei-platform/issues/2621)、[Draft #2624](https://github.com/hat47x/tei-platform/pull/2624)（明示Mapping・合成20試験）、[#2516](https://github.com/hat47x/tei-platform/issues/2516)（Consumer Claim/開示）、[#2515](https://github.com/hat47x/tei-platform/issues/2515)（外部Harness機密）、[#2387](https://github.com/hat47x/tei-platform/issues/2387)（SQL構造）、[#2611](https://github.com/hat47x/tei-platform/issues/2611)（DB Alias）、[#2018](https://github.com/hat47x/tei-platform/issues/2018)（開発効果）。

## 1. 差分：Outputの有限不変性と、実取得範囲は別

#2624のEM-18〜20は、未宣言の秘密Fieldや未選択Relationだけを変更した二世界で自作Pythonの**Resultが不変**、選択したFieldならResultが変わることを合成20件中に確認した。この出力依存性は明示Function Data Contextの設計検証として有用。

しかし、**Output equivalence ≠ Provider read minimality** である。

- **Narrow Provider**：必要なOrder/LineのFieldと合法なJoin補助キーだけを要求する。
- **Overfetch Provider**：同じOutputを返すが、内部で秘密Fieldも読み、返却直前に削除する。
- **Unmapped Relation Provider**：同じOutputを返すが、無関係なPaymentの存在を照会する。

三者のJSON結果が同じでも、Read Trace・許可された実アクセス・MetadataやLogの露出は異なり得る。Schema/Mapping/Roleの「使わない」という宣言だけでは、実Providerにおける過剰取得が無いことを証明できない。一方、実Queryを観測してもDB Storage/Cache・Networkのすべてを観測したとは限らない。

## 2. 既存Ownerの五つの観測面

| 面 | 検証する事実 | 保証しないこと／Owner |
| --- | --- | --- |
| **Declared footprint** | Functionが指定したRoot/Relation Path/Field/Scope/Grain/完全性要求 | 実Queryが発行されたこと。#2506 |
| **Realized request** | Provider Alias、SQL/HTTP要求、Role/実対象、Filter/Projection/Pagination、補助読取りの目的 | そのPlanが実行されたこと。#2622/#2387/#2611 |
| **Executed-access witness** | 送信したQuery/Request、引数、実行物/Session、Provider Trace/Audit、Response | DB内部I/O/Cacheや全副作用を完全に観測した証明ではない。Provider/Host Owner |
| **Result Slice/Claim** | 取得値とID、時点、重複・NULL/Unknown/Partial、実完全性の証拠 | 実アクセスや他Contextの完全性。#2506/#2516 |
| **Consumer disclosure** | CLI/SHO/Tool JSON、Log/Temp、Network、公開対象の値・Metadata/Trace | ローカルread-onlyによる外部送出権限。#2515/#2516/#1178 |

上の行は比較観点であり、別の共通API・Registry・Evidence DBのschemaではない。実装時は既存Provider/Result/Source/HostのReceiptを用いる。

### 補助読取りの条件
Outputに無いFieldが全て違反とは限らない。Join Key、Tenant Policy、Hash/Index、実行に必要な検証Fieldが合法に必要な場合がある。したがって**Output Field／意味・実行依存Field／実Provider許可Field／実観測Read／外部開示Field**を分離する。細粒度ProjectionができないAPIでも、意味Outputの限定とProvider上のOverfetch、外部開示の可否を正確に説明できるようにする。Provider上の過剰取得を許可するかはHost/組織Policyの判断。

## 3. 初期圧力（RW-01〜12）

| ID | 入力・反例 | 期待する判別 |
| --- | --- | --- |
| RW-01 | 明示したLine.amountを狭く取得 | 必要なField/RelationのRead Traceで結果を得る |
| RW-02 | 秘密Fieldも取得し、出力時だけ削除 | JSON同値でもRead Traceの差を検出 |
| RW-03 | 未選択のPayment Relationを読む | JSON同値でも無関係Relation要求を検出 |
| RW-04 | Join/Policy補助キーを追加で読む | 合法な補助依存と無断Overfetchを区別 |
| RW-05 | APIがField選択を提供せず全Resource取得 | 意味出力一致とProvider能力差を併記 |
| RW-06 | Traceが未取得、内部Cache/Prefetch不明 | Outputが正しくてもRead最小性はunknown |
| RW-07 | 全件要求だがPagination未完了 | Declared allとObserved completenessを分離 |
| RW-08 | 値同一だがStatus/Timing/件数が変化 | 内容JSON一致から一般非干渉を証明しない |
| RW-09 | 内部Readは許可、AIへ送出は許可外 | Local ReadとRecipient disclosureを分離 |
| RW-10 | 同一Semantic IDが複数Alias/Replicaへ束縛 | 物理Target/Tenant/Snapshotと取得資格を分離 |
| RW-11 | SQL Pushdownとアプリ側Aggregateが同じ値 | NULL/Bag/Unit/Time/Grain/Completenessの法則保存を別に確認 |
| RW-12 | Code/SHO/CLIが同じFieldを表示 | 表示/公開/実Read/業務受理を別Evidenceとする |

**局所合成の最初の対象はRW-01〜07の狭い部分**。特にOutput同一・Access Trace差の対照を固定し、自作Adapterが生成した期待値をそのままOracleへ流用しない。実DB/OS/Providerを通すRW-08〜12は後段の別Ownerで検証。

## 4. 既存案件との進行順

- **L0 合成Provider Adapter**：Order/Line/Payment、private_note、join補助キーを架空データに持たせ、Read Trace・Output・Scopeを別に採取。Python単体試験はTEI製品のProvider実証ではない。
- **L1 実TEIモデル→Context→Query**：#2624のSource/Mappingと実TEIのData Context/Query Workerを固定版で照合。実SQL/Request/Parameters、実Provider Alias/Role/Session、実Read/Resultの紐づく範囲を測る。Query Explainだけでは実行やHost権限を保証しない。
- **L2 二つの異質Provider**：RDB等でField選択が可能なTargetと、全Resource APIを同じ意味Functionで比較。実物理Readの保証とExternal Disclosureを分け、観測できないProvider内部はunknownに留める。
- **L3 AI/SHO/異質Consumer**：TEI Skill #2605・TAI・SHOは許可済みProjectionを利用し、Raw Provider Traceを当然には受け取らない。二世界の開示と情報推定は#2516、実外部送出・OS側拒否は#2515をOwnerとする。
- **L4 実効価値**：明示Mapping、補助Read説明、Overfetchの検出、適格な代替Provider選定のための定義・修正・AI問合せ工数を#2018へ渡す。大規模な要件圧力は#2493で索引。

### 成功の定義と既存構造の再利用
Semantic Normal Formを生成できたこと、Data Contextの明示Mappingが成立したこと、ProviderがそのDataを読んだこと、Query上のOverfetchがないこと、外部AIへ送ってよいことは別々に観測する。

Role/Source正本は#2620/#2621/#988、Mapping/Result Sliceは#2506、Data Operations/Providerは#2622、情報送出とClaimは#2515/#2516、業務Effectは#1174。新中央Host・万能InfoFlow Checker・独自Provider Resolver・別トランザクションEngineを必要条件にしない。

**資格**：RW試験の合成Referenceは今後の実証案。L1〜L4の実TEI・実DB/API・Host/OS、正式Role/Data Ops API採用、Owner意味受理はこの文書では未確認。