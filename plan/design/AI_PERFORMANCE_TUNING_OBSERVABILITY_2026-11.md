# 生成AIによる性能チューニングのための設計ヒント・統計観測・検証（2026-11）

- **状態**：横断設計・実証計画候補。正式Schema、CLI/API、実装、実Target性能改善、AI自動適用、Owner意味受理は **NOT RUN / NOT ACCEPTED**。本書のサンプル名・フィールドは説明用であり、現行TEIの公開契約ではない。
- **目的**：TEIのモデル（Data / Function / Business、Query、Data Context、Source、Plugin、Lifecycle）と、実際の実行統計を同一対象・版・範囲で接合し、生成AIが**根拠付きの最適化候補**を発見・比較・検証できるようにする。対象はアプリケーション、TEI自身のビルド、DB、API、UI、配信、Plugin合成を含む。
- **基本判断**：計測、設計ヒント、統計、推論、最適化提案、実装、検証結果は異なる。AIの予測・計画・Cache hitから性能改善や業務意味の受理を主張しない。標準は最適化の上限ではなく、**案件固有の実装へ安全に逃げられる足場**とする。
- **非目標**：第二のOptimizer / Query Planner / CAS / Cache Host / Resolver / Registry / 認可Engine / 監視SaaSの新設、全アプリへの常駐Agent強制、あらゆるTargetへの常時計測、実行統計の業務正本化、本番DBの無許可チューニング、既存製品Gateの変更。

## 1. 既存契約との責務分担

| 既存Owner | 担当する正本・境界 | 本書の追加分 |
| --- | --- | --- |
| [#3271](https://github.com/hat47x/tei-platform/issues/3271)、[#1641](https://github.com/hat47x/tei-platform/issues/1641) | DerivedData、Cache/Snapshot、保存・再利用資格、業務確定値の分離 | 最適化候補の収集・適用判断へCache統計と更新競合のEvidenceを提供 |
| [#2582](https://github.com/hat47x/tei-platform/issues/2582)、[#2506](https://github.com/hat47x/tei-platform/issues/2506) | Contextの依存、粒度、結果Scope、再取得・差分適用 | 依存ヒントと実際の再取得回数・再計算費用を結び付ける |
| [#2387](https://github.com/hat47x/tei-platform/issues/2387)、[#2622](https://github.com/hat47x/tei-platform/issues/2622)、[#3270](https://github.com/hat47x/tei-platform/issues/3270) | Queryの意味・Lineage、Data Provider、DB Workbench/Schema CRUD | SQL実行計画・実統計・Index候補をSource/Queryへ逆引き |
| [#951](https://github.com/hat47x/tei-platform/issues/951)、[#1016](https://github.com/hat47x/tei-platform/issues/1016)、[#3267](https://github.com/hat47x/tei-platform/issues/3267) | Contract、Provider Resolver、Pin、Qualified BOM | 観測と最適化Pluginの実選択・実呼出し・版・Target束縛 |
| [#3278](https://github.com/hat47x/tei-platform/issues/3278)、[#3279](https://github.com/hat47x/tei-platform/issues/3279) | 原子的Traitの実装義務・合成・Fulfillment | 保証が必要な最適化のみTrait要求化し、Advisory Hintと分離 |
| [#3262](https://github.com/hat47x/tei-platform/issues/3262)、[#2516](https://github.com/hat47x/tei-platform/issues/2516) | AI/CLI/MCP向けDesign Contextの完全性・開示 | 用途相対の「AI Tuning Packet」read-only投影 |
| [#1180](https://github.com/hat47x/tei-platform/issues/1180)、[#2514](https://github.com/hat47x/tei-platform/issues/2514)、[#2600](https://github.com/hat47x/tei-platform/issues/2600) | 独立Expected、Scenario、Build再利用・費用実測 | 性能比較・統計収集費用・最適化前後の因果Evidence |
| [#3503](https://github.com/hat47x/tei-platform/issues/3503)、[#3509](https://github.com/hat47x/tei-platform/issues/3509) | Interaction/Operation/Generator Lifecycle、B2Cでの脱出経路 | Hookへの計測・戦略のBinding、Browser/Edgeを含む任意最適化 |
| [#1119](https://github.com/hat47x/tei-platform/issues/1119) | 横断製品Gate | 既存Gateに新Hard Gateを追加しない |

設計上のRole、Trait、Provider Capability、Source Observation、Acceptance、実行Authorityを混同しない。原子Traitの名前を増やすこと自体が機能の増加ではない。

## 2. 「AIがチューニングできる設計」の最小材料

**先に機械で取得できる情報を整え、生成AI向けの専用DSLを先行設置しない。** Design Context Queryの用途別投影として次の六群を結び付ける。

| 群 | 代表的な材料 | 出所と限界 |
| --- | --- | --- |
| 設計意図 | Function/Operation、Queryの意味、Semantic Entity/Relation、結果粒度、整合性・鮮度・許容遅延、性能目標と優先度 | Sourceと受理範囲に束縛。未記載のSLOをAIが確定しない |
| 構造ヒント | 主キー/外部キー、Relation多重度、索引・制約、Derivedの依存グラフ、集約軸、選択/ページング、Execution Hook | DDL観測≠業務意味。多重度・件数を静的に決めつけない |
| 実装の選択肢 | Query/Native SQL、Provider Capability、Cache Instance、生成Target、Batch/Stream/局所更新の実現能力 | Discoverable/Selected/Built/Invoked/Qualifiedを区別 |
| 観測されたWorkload | リクエスト量、データ量/サイズ分布、検索条件の選択性、同時実行、Read/Write比率、Hot/Cold、偏り、変更・失効頻度 | 観測窓・標本・環境・Principalの開示範囲に限定 |
| 実際の費用 | End-to-end latency、p50/p95/p99、CPU/Alloc/GC、DB呼出回数・行数・I/O、ネットワーク、再取得・Cache、観測オーバーヘッド | 計測地点・欠測・計測負荷を保持。DB cost値を時間と混同しない |
| 制約と反証 | 認可・Tenant、現在Grant、NULL/partial/unknown、非単調集計、Effect、整合性、Target、独立Expected、Rollback | 性能向上が業務結果・開示資格の変更を許さない |

**設計ヒントは発見対象の索引**であり、実行方式への命令ではない。たとえば「参照が多い」「数百万行になり得る」「時点更新が頻繁」は提案の理由になるが、観測条件がない限りキャッシュやIndexの自動採用理由にはしない。

この六群の各要素は、最低限 **declared / observed / inferred / proposed / verified / accepted** の出所・資格を区別する。すべてにSource/Revision、Scope/粒度、Time window、Producer/方法、Completeness、Disclosureを付すべきかを既存標準データRoleで検証する。新しい全域Enum/第二台帳を作らない。

### 2.1 AIが見つけるべき典型的なシグナル

| シグナル | 根拠となる組合せ | 候補例 | 必須反例 |
| --- | --- | --- | --- |
| N+1取得 | FunctionのRelation traversal＋同型SQL実行回数＋実Rowset | Batch、Join、Request-local Cache | 観測が部分Trace／各取得の権限・時点が異なる |
| 過剰再計算 | Derived依存グラフ＋変更対象＋実CPU計測 | 局所再評価 | NOT EXISTS、Top-N、LEFT JOINの結果撤回 |
| Index不足の可能性 | 条件/並び順＋実Query Plan＋データ分布＋統計鮮度 | Index・Query改善 | 統計が古い／書込み負荷・容量が増す |
| 推定件数の乖離 | 推定行数と実行行数＋JOIN/Predicate＋DB統計取得時点 | 統計更新・Query見直し | 統計粒度とパラメータの偏り・複数列相関 |
| Cacheの逆効果 | Hit/Miss＋再生成時間＋メモリ/転送/失効費 | Policy変更、Cache廃止 | Hit率だけ改善しp99が悪化 |
| コールド時の集中 | Miss同時数＋Load latency＋Backend queue | Request Coalescing、負荷制限 | 同一性・Tenant/Grant違いの要求を束ねる |
| 分散・配信費 | Edge/Network/Origin実測＋鮮度要件 | CDN、Streaming | Edge purgeのACKを即時の全地点失効と誤認 |
| Trace空白 | サンプル率・計測対象Coverage＋欠測窓 | 追加の限定計測 | 欠測から「処理無し」を推論 |

## 3. 統計収集：既存Observabilityへ接続するPlugin群

一律の常時詳細収集を避ける。**低負荷の常時集計 → 必要時の詳細Trace → 対象を絞ったProfiling → 独立Benchmark** の段階を標準構成の候補とする。

| 統計の粒度 | 内容 | 保存・観測の原則 |
| --- | --- | --- |
| Request / Operation | Duration分布、throughput、失敗、queue待ち、同時数、End-to-end Trace | Trace/Span IDとTEI Function/OperationのSource Refを対応。Metric labelは低Cardinalityに限定 |
| Query / Provider Call | Query fingerprint/summary、実Provider Instance、呼出回数、行数・bytes、latency、例外、Result scope | SQL本文・パラメータ・URI/機密情報は既定で集めない。正規化や秘匿を優先 |
| Cache / Derived | hit/miss、eligible/rejected、eviction、invalidation原因、再生成費、Key scope | Key文字列・Tenant識別・非開示件数をMetric labelやAI入力に出さない |
| Runtime / Host | CPU、Alloc、GC、Mem、Pool待ち、Scheduler、Network、UI描画 | Go pprof/trace、Browser Performance、外部APMへのAdapter。連続Profilerは強制しない |
| DB / Storage | pg_stat_statements等の累積統計、Query Plan、estimate/actual rows、統計時点、I/O | 実Sourceの能力・権限に従う。実行計画取得は原則Read-only、EXPLAIN ANALYZEは実クエリを実行する点に注意 |
| Build / Toolchain | Parser/Resolve/Generate/Buildの各段時間、Cache再利用、Artifact、Evidence費用 | #2600の実消費入力とArtifact/Evidenceの区別を保持 |

**計測の費用も測る**。Sampling率、欠測、観測に伴うCPU/Memory/Network増、Storage/egress、Retention、Exporter停止時の影響を観測する。計測停止で業務処理が止まらない設計を優先する。重大な一部の監査/Evidenceは監視Telemetryとは別の既存契約で保持する。

### 3.1 数字を解釈する前に残す情報

1. 集計Window（開始/終了）、Time source、Reset/Restart境界、Temporal granularity、測定単位。
2. Environment/Target、Build artifact、Source/Contract/Plugin/Generator Lock、Schema/Database physical revision。
3. Sample count、Sampling/Drop率、観測可能な範囲、Complete/Partial/Unknown、遅延到着・重複排除。
4. 分布（Histogramやraw sampleの扱い）、p95/p99の母数、Warm/Cold、同時負荷、パラメータ・データの偏り。
5. Providerの実識別、Query fingerprintのアルゴリズム/適用範囲。DB側queryidはTEIの永続Canonical IDと同一視しない。
6. 実行Planの推定値と実測値の区別、統計取得・ANALYZEの時点、Planner/DB版。DBの推定costをミリ秒へ変換しない。

OpenTelemetryのSpan/Metric/Exemplarに接続するが、TEIが全Metricを独自保管する必要はない。メトリクス集計には低Cardinalityラベル、詳細なQuery/Traceはアクセス制御された別経路を用いる。必要なTraceをSamplingで失った場合、その原因分析は不確実なままとする。AIへのエクスポートは#2516の用途相対開示を経由する。

### 3.2 計測ヒントを生成物へ出す際の要件

- Source Refと意味に対応付いた **低負荷な計測地点** をGenerator/PluginのHookへ配置できる。Bindingを無効化・変更してもFunction結果の意味を壊さない。
- Instrumentationは静的結合/生成Adapter/外部Hostを許容し、TEI共通Hostの常時介在を前提にしない。
- 計測の有効化・無効化、サンプリング、Export先、RetentionはTarget/Environment別。収集の有無を「業務機能未実行」と誤判定しない。
- 高Cardinality属性、個人情報、秘密参照、実SQLパラメータ、生データ、Authorizationは既定収集しない。必要なら明示的な目的・権限・短期保持を要求する。
- 実際のObserver Providerと呼出結果を既存のSelected/Invoked/Evidenceで照合し、単なる設定表示を「計測済み」としない。

## 4. AI Tuning Packet：既存Design Contextの用途別read-only投影

「AIに詳しい統計を送る」ではなく、対象Function/Query/Operationに必要な**設計ヒント＋観測値＋未観測＋制約**を圧縮した投影を利用する。名称は仮称。既存#3262 Design Context/Source Index/Typed Refと#2516 Disclosureを再利用し、新Canonical Schemaや新Registryは作らない。

~~~text
TuningPacket (説明用。現行APIではない)
  target: Function/Query/Operation typed reference + revision
  objective: latency/throughput/cost + SLO/priority + accepted source
  constraints: correctness/freshness/consistency/tenant/disclosure/effect
  model: dependency, relation, query plan, context scope, grain
  realization: resolved Provider/Plugin/Target/Build/DB/schema revision
  workload: windows, histogram, distribution, sampled count, completeness
  observations: selected spans/metrics/profiles, estimate-vs-actual
  candidates: optimization proposal(s) + expected mechanism + risk
  tests: independent Expected, negatives, cold/warm/load variants
  result: NOT RUN | measured with scoped evidence | rejected/unknown
~~~

次の六段階を明示し、結論を一足飛びに進めない。

~~~text
Design intent + Source ─┐
Actual Call/Trace/Stats ├─> Evidence join ─> Candidate ─> Preview
Plugin/Target/Grant ────┘                               │
                                 Independent correctness & benchmark
                                                       │
                              Owner decision / authorized application
                                                       │
                                   post-change observation / rollback
~~~

AIの出力は**提案**にとどめる。予測効果（estimated）と実測改善（observed）、Test Passと業務受理、実行権限と操作完了を区別する。ActionがDB DDL、運用設定変更、Provider更新、外部副作用を含む場合は既存のPlan/Grant/Effect/Receipt/独立再観測を使用する。

### 4.1 チューニング提案の最低記載事項

- **What/Where**：対象のTyped Ref、Source位置、適用Target/Environment、影響する変更箇所。
- **Why**：どの統計・構造ヒントから候補になったか。未取得の情報・別解・推論上の弱点。
- **How**：利用する既存Provider/Contract、Hook、生成方式、Resourceへの追加負担、失効・競合時の動作。
- **Expected**：実際の正しさの独立Expected、対象のSLO、想定改善範囲と測定条件。
- **Risk**：性能退行、計測費、データ漏洩、Cache stale、DB write負荷、Rollbackの可否。
- **Decision**：Proposal→Preview→Verified→Applied/Rejectedを根拠とともに追えること。ただし新中央Approvalを作らない。

## 5. 最適化候補の探索順と安全な最適化

探索順の基本は (a) 不要な実行・N+1・重複/過大取得を除去、(b) Query/Batch/演算/並列の実行方式を改善、(c) 再計算・Cache・Materializationで再利用する、の三段。固定の最適順ではなく、実測と制約で変更する。

- Cache Strategy：L1/L2/L3、admission/eviction、request-local、TTL、stale-while-revalidate、singleflight相当をProvider能力として比較する。**Cache key一致と再利用資格・現在開示は別**。Write-behindは確定保存値をCacheとして扱う理由にしない。
- Query Strategy：SQL Pushdown/Native SQL、Index、Statistics、Plan shapeを候補とする。NULL/粒度/順序/時点/部分集合/Collation/金額精度を保ち、実DB側の更新効果は別検証。SQL-authoredのSource of Truthを書き換える場合は明示的なSource変更として扱う。
- Runtime Strategy：Batch、Streaming、Pooling、Parallelism、Backpressure、UI Virtualization、SSR/CDNなどは媒体・Target固有。標準RendererやTEI Runtimeを通さないヘッドレスな構成も合法とする。
- Dynamic strategy：状況に応じたCache比率やPrefetch切替は将来候補。統計遅延/フィードバック振動/負荷急変の反例が揃うまで常時自律変更を行わない。

「高速だが誤答」を許さないことと、「最速を目指して運用障害を招かない」ことを独立に検証する。CPU/Memory/DB/Network/Storage間の移転費、tail latency、同時負荷を比較に含める。

## 6. 検証とBenchmarkの成立条件

- 同じ業務Source/Meaning/入力Dataset/Independent Expectedで、**baselineとcandidate**を実Targetで比較。元実装由来のExpectedを使わない。
- 同一の負荷分布、データ量、Read/Write比、同時数、Warm/Cold、Cache invalidation、制御された試行順と反復数を保持する。時刻差・環境差・他負荷は交絡因子として表示する。
- p50/p95/p99を比較するときは標本数・Histogram誤差・信頼可能性を明記。平均だけの改善でtail regressionを隠さない。
- Profiling・Sampling・Exporterを有効化した場合と無効化した場合の計測費を評価し、性能測定のための計測が結果を歪める場合を記録する。
- 結果の値だけでなく、Entity identity、NULL/missing/unknown、取得Scope、時点/順序、Effects/Grant、対象Provider、Source/Toolchain/Build/Schema versionの一致を検証。
- Rollbackは「Code/Strategyの巻戻し」と「Schema/データ副作用の復元」を同一視しない。適用後は独立Read/Trace/Metricを再取得する。
- 将来のAI自動探索は候補の反復生成までから始める。自動適用は安全な変更範囲・権限・観測の実証後に限定する。

### 6.1 最小の独立反例セット（すべてNOT RUN）

| ID | 圧力ケース | 必須Expected |
| --- | --- | --- |
| AT-01 | 実Function→Query→DB Provider Read＋Function Trace | 正しい対象Ref/Source/実選択Provider/実呼出Provider/Artifact/Scopeが一Attemptに結び付く |
| AT-02 | 同一SQL形状だがパラメータ・Tenant・取得時点が異なる | Query summaryの集約は可、結果/権限/Cache Keyを誤共有しない |
| AT-03 | N+1を合成Fixtureで観測しBatch候補を提案 | Query回数と結果集合・粒度を正しく比較。未計測区間を無処理としない |
| AT-04 | DB計画で推定1行→実10万行、統計採取時点不一致 | Optimizerの推定と実測を分離し、Index作成を自動確定しない |
| AT-05 | D1のみのDerivedとD1×D2の非単調集計を一部更新 | D1-onlyは根拠あれば維持、JOIN/NOT EXISTS/Top-Nは失効・retract/再取得 |
| AT-06 | Cache hit、Tenant/Grant失効、同byte・別Principal | 現在Disclosure拒否。存在情報やKeyのTelemetry漏洩も拒否 |
| AT-07 | Miss競合・A旧値Read→B更新/失効→A古い書込 | 古い世代の復活防止、根拠不足なら再取得/unknown |
| AT-08 | Trace欠損、Sampling率変更、低件数のp99 | 欠測/推定不確実性を明示。未観測の性能改善を主張しない |
| AT-09 | Source A→B→A、Provider/Toolchain/Schema版変更 | 古いBenchmark/Evidence/Grantを自動復活させない |
| AT-10 | Selected Provider B、実呼出しProvider A／Observer未呼出し | 見かけの値一致・設定表示で合格にせず、実選択と実Callの不一致を拒否 |
| AT-11 | AI提案のIndex/DDL、EXPLAIN ANALYZEによる更新系SQL | read-only分析と実行可能な変更を分離し、本番で無許可実行しない |
| AT-12 | Hit率向上だがp99/CPU/Storage費・書込遅延が退行 | 総合SLO/Cost Guardrailで不採用またはRollback |
| AT-13 | 計測有効/無効の処理費とTelemetry export障害 | Observable overheadを実測し、業務操作を不要に停止させない |
| AT-14 | AIが不完全なSourceから「不要索引」「削除可能」等を推論 | Candidateのまま、Owner受理・Host権限・独立観測に昇格させない |

## 7. 段階的な開発計画と停止条件

| 段階 | 実施する最小内容 | 完了を示す証拠 |
| --- | --- | --- |
| **P0-A：材料調査** | 既存CLI/Design Context/Query/Provider/Metric・pprof等を棚卸し。Targetごとの既存収集可能範囲と欠落を可視化 | 既存Source/Actual Call/受理StatusとのFit/Gap、重複機構を作らない根拠 |
| **P0-B：read-only観測縦断** | まず order_entry.recent_orders 等、実際に呼び出したFunction→Query/SQL→DB Provider→Rowsetを観測。小さなTEI Source/Function参照だけをOTel風Traceへ結合 | 誤Provider・部分取得・欠測、Query summary、latency/rowsの正負例と独立Expected |
| **P0-C：AI Tuning Packet** | #3262 Design Contextの用途別read-only投影に、限定統計・欠測・鮮度・Disclosureを接続。CLIを先行しStudio/MCPは同じ契約を再利用 | 同じPacketの根拠列と開示境界、未観測はunknown、AI候補≠受理 |
| **P1：可換の最適化実証** | Cacheなし/Memory/Localの切替、N+1→Batch、SQL Pushdownの少なくとも異質な2方式を選ぶ。cold/warm・高負荷・更新競合の試験 | byte/Meaning/Effect/結果一致、p50/p95/p99・費用、逆効果と失効を計測 |
| **P2：専門Observer/Analyzer Plugin** | 実DBの統計/Query Plan、Go profile、Browser/Edge等のうち需要があるものを段階追加。Provider能力と権限別実装を検証 | 実選択Provider/実Call、Scoped Coverage、観測オーバーヘッド、BOM/TargetごとのEvidence |
| **P3：AI提案→検証→適用** | AIの候補生成、構成差分、Scenario/Benchmark、監視・Rollbackを接続。低リスクの単一候補から始める | baseline/candidateの独立比較、変更実行のGrant/Receipt、適用後の独立観測 |

**優先順位**：現在の#1119 G0〜G4、実Providerの一往復、自己構築を置換しない。P0のObserver/Packetはその経路に重ねる。既存製品には計測がない状態でも構成できる余地を残す。

**縮小・撤回条件**：新Packetを作らないと既存Design Contextが拡張できない場合は、既存契約の部分投影へ縮小する。計測費が改善効果を上回る、型/Scope/Grant一致を検証できない、選択Providerを実Callで確認できない、第二Resolver/Hostが必要になる、独立Expectedが不足する場合、当該最適化を未受理のまま留保する。

## 8. 外部技術の参照先（調査資料であり製品実装の採用決定ではない）

- OpenTelemetry: [DB Span Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/db/database-spans/)、[DB Metrics](https://opentelemetry.io/docs/specs/semconv/db/database-metrics/)、[Metrics Data Model](https://opentelemetry.io/docs/specs/otel/metrics/data-model/)。低Cardinality summary、Histogram/Window、観測のSamplingとLimitsを参考にする。
- PostgreSQL 18: [pg_stat_statements](https://www.postgresql.org/docs/18/pgstatstatements.html)、[EXPLAIN](https://www.postgresql.org/docs/18/using-explain.html)、[Planner Statistics](https://www.postgresql.org/docs/18/planner-stats-details.html)。queryidの安定性は限定的。推定行数と実測行数を区別し、EXPLAIN ANALYZEは実際に文を実行する。
- Go: [Diagnostics](https://go.dev/doc/diagnostics)。pprof等は実Targetで費用を測って利用する。
- Raychaudhury et al., [Coresets for Robust Query Optimization](https://doi.org/10.1145/3801896) (2026)。不確かなCardinality推定の下で頑健な実行計画を選ぶ研究例。特定アルゴリズムをTEI標準に直結させない。

## 9. 第2段：Hint・Observationの結合に「実行資格」と「統計上の根拠」を持たせる（2026-11-08）

[#3514](https://github.com/hat47x/tei-platform/pull/3514)で成立した**設計計画**を、試験可能な最小判別条件に下ろす。性能に関する「Hint」は、設計者の意図、Sourceに実際にある構造、生成AIの仮説、運用統計を混ぜた単一の指示文ではない。既存のTyped Reference/Role/Claim/Observation/Assessmentを利用し、Design Hintは**検索・分析の手掛かり**、Performance Observationは**測定事実**、Optimization Proposalは**提案**、Candidate Resultは**独立テストの結果**として切り分ける。新しいPerformance意味正本や万能Optimizerは設けない。

### 9.1 AIが自力で抽出できる「設計上のヒント」

| Sourceからの手掛かり | 何を推定できるか | 機械的に確定してはいけないこと |
| --- | --- | --- |
| Entity/Relation多重度、FK/Index、必須・NULL、取得粒度 | N+1、JOIN増幅、Index候補の探索範囲 | DB上の実Cardinality、コスト、業務上の多重度をDDLだけで確定 |
| Query Predicate/Sort/Limit、選択されるPhysical Provider | Pushdown/Batch/ページングの候補 | SELECTの見た目だけで実SQL・実Rowsetの等価性を確定 |
| Derived Ruleと正/負依存、non-monotonic演算（NOT EXISTS/LEFT JOIN/Top-N等） | 再計算できる候補、差分適用の危険範囲 | 部分集合での不存在、insert-onlyでの安全な増分更新 |
| Function/Contextの選択epoch、表示パターン、Page/Stream設定 | 同一画面の冗長取得、初期表示遅延の調査対象 | 異なるContext/Principal/epochのCache再利用 |
| Operation/Effect/Receipt/Host Grant | 更新に伴うCache失効の契機候補 | ACK・Command成功・古いReceiptをDB更新確定や現在認可に昇格 |
| Plugin/Target Lock、Hook/Generator/Binding | 観測・最適化を配置できる場所、Provider能力 | 宣言/選択から実Loader/Invocationや計測動作を確定 |
| NFR/SLO/業務優先度、保持/鮮度制約 | 目的関数とGuardrailの候補 | 未記載の速度閾値、業務価値、更新回数をAIが正本化 |

**隠れた依存と負の依存も含める**。SQL本文不変でも新しいDialect候補が追加されると選択結果が変わる（#2586 QX-01）。Schema/Plugin/Provider/Grant/Source Scopeの変化があれば旧Hint/統計/Benchmarkの適用資格を再判定する。A→B→Aでも旧Measurementを現在のEvidenceとして自動復活させない。**Source byte、World/Model Revision、Build Artifact、Provider/DB revision、Observation Window、Consumer資格**を用途別に照合する。

### 9.2 計測地点と低負荷統計の段階的構成

1. **常時・低費用**：Operation/Provider/Query summaryごとのcall count、latency histogram、error、cache admission/hit/miss/eviction、row countの許可された集約、Pool queue。集計キーにRaw SQL、Trace ID、Tenant/Principal、PK、パラメータを含めない。
2. **条件付き詳細Trace**：特定のFunction/Query/障害Windowを対象に、実Callの時刻・処理段・選択/実呼出Provider・Source/Scope・Span連鎖を限定収集。Sampling/drop/遅延到着/境界を明示し、未観測から無処理を主張しない。
3. **期間限定Profile/Plan**：CPU/Allocation/GC、DB EXPLAIN/実行計画、Index・統計鮮度。Profilerのオーバーヘッドを独立測定し、計測失敗を業務失敗に変換しない。
4. **隔離Benchmark**：Baseline/Candidateを同一Source・必要な実Dataset・Workload/Target条件・Independent Expectedで対照。適用後は本番の独立観測を得る。実行統計を収集しただけでは改善を認定しない。

OpenTelemetryのstable DB semantic conventionsでは低Cardinalityな `db.query.summary` を使える一方、Query本文・パラメータや一部の属性にはOpt-in/容量・秘匿の考慮が必要。生成Targetへ常時「全SQLパラメータをTraceへ書く」実装を強制しない。個別DBで提供される累積統計のQuery IDはTEIの永続Semantic IDではなく、Schema変更・DB版/再起動・統計resetの前後を同一測定とみなさない。

**長期保持の基線**は低CardinalityなHistogram/CountとSource/Target/Window概要、**短期・限定保管**は高粒度Trace/Plan/Profileとする。保管場所/期間、収集許可、目的、region、事後削除は既存のSource/Provider/Disclosureへ接続。OTelを未導入でも計測無しで動くアプリを正式な生成形態として残す。

### 9.3 「前後比較できる」ことを先に設計する

単一計測から「改善率」「p99」「N+1」「実行計画が悪い」を確定しない。基準と候補で、(a)固定する条件、(b)観測Window/標本数、(c)計測オーバーヘッド、(d)CIや他負荷による交絡、(e)DBの統計鮮度、(f)実Provider/Target/Build、(g)意味・開示の同等性、を記録する。ヒストグラム境界・sampling・単位が違うp99を単純差分しない。統計が不完全なときは探索候補の優先順位付けに利用してよいが、SLO合格・性能向上の証拠へ昇格しない。

既存 [CLI trace benchmark](../../tools/reference-cli-trace-benchmark/README.md) は、実際の段別ProfileとSource Indexの改善、出力同一性、同一環境での比較、競合検出を扱った実例である。ここで示された過去の計測値を、**生成アプリのDB Readや全TEIの現在性能の証拠へ転用しない**。同Referenceの「支配費用を観測→候補を絞る→変更→出力同一性・メモリ/時間を再測定」という手順のみを再利用する。

## 10. P0の次に進むための限定Referenceと受入境界

[reference-ai-tuning-observation](../../tools/reference-ai-tuning-observation/README.md) を**試験用Consumer**として追加する。設計SourceやDBを直接解釈する製品実装ではなく、外部から渡された合成Design/Call/Statistics/Delivery/Disclosureの「矛盾・欠落・取り違え」の判別能力だけを確認する。

- 調査用の繰り返しReadの兆候は `investigate-repeated-reads` であり、N+1の確定やBatch化の受理ではない。
- Source Revision、World Generation、対象のTyped Reference、Query、Provider選択/実呼出、Scope、Grant Epoch、統計Window、分割Replyの連番/end markerを一つの観測記録で照合する。
- 元のJSON申告に一致するだけなら `input-consistent` としか言わない。**実Providerを呼んだこと、Reply digestが内容に結び付くこと、ユーザーの開示資格が実在することは未証明**。実際のRead接続（#2586）と独立Evidence（#1180/#2514）で後から資格を上げる。
- 試験用Consumerが元データ・SQL・Tenant/Principal・Source pathをそのまま出力しないよう、Disclosure拒否は固定の `withheld` とする。これも実本番Authとは別。
- 既存の観測統計を材料に提案する場合でも、発見した候補を Source/Contract の変更として扱い、変更後の independent correctness + measured performance を両方通す。

### 次の実製品縦断（この追加では未実行）

**対象は一つのFunctionと一つの実Query/Provider Readに限定**し、次の順に証拠を積む。

| 順 | 担当Ownerと成果 | 非充足として残す境界 |
| --- | --- | --- |
| T0 | #3262で必要Source/Function/Rule/Relationと未取得・Partialを一つのRead Contextとして取得 | Design Contextの完全Closure、Tool Reply end-to-end |
| T1 | #2586/#2622で同一AttemptのQuery Binding→実選択Provider→実Call→Rowset/Grain/Scopeを証明 | Preview/Mock/静的SQLを実Callにしない |
| T2 | #3271/#2582で当該ReadのCache無し/Memory/Localと再取得・失効の正負例 | 再利用資格と開示を分離、見かけのHitだけでPASSにしない |
| T3 | #3262/#2516でSource/実Call/Window/欠測/Grantをread-only AI向けPacketへ投影し、Tool受領を照合 | JSON parse/exit0/1ページだけを完全受領にしない |
| T4 | #1180/#2514/#2600で候補前後のMeaning/Effect/性能/計測費を独立に比較 | 未実測のAI改善予想や合成結果を本番改善へ昇格しない |

Targetはまず既存Go・SQLite/PostgreSQL等の**実在する接続経路**から選び、未知の新コマンドや第二Readerを設計正本にしない。各段階は既存のGateの上に載せるが、**追加Hard Gateではない**。P0の観測・検査だけで時間を使い切らず、最終的に有益な最適化が生まれるか（追加行数/負荷/運用費を含め）を実測する。

**状態**：本追補・Reference追加で実DB/DB統計/実TEI Function/外部Tool/AI/改善結果が接続されたとは主張しない。合成Referenceの実行資格と、実Target実証の資格を独立に報告する。

## 11. 横断統合：性能要求から実装とEvidenceまでの最小契約（2026-11-08追補）

本追補は#3271 Cache/Snapshot、#2582 Data Context、#3278 Trait、#3503 Lifecycle、#3509 B2C、#2600 Build再利用を、既存AI Tuning Packet設計と接続する。**各分野の正本を作り直さず、組込み標準と案件固有実装を同一条件で比較可能にする**ことが目的。以下の名称・Schema/API・Plugin Roleは設計候補であり、採用済みの公開契約ではない。

### 11.1 最適化の判定単位



最適化の単位は Subject × Workload × Requirement/Guardrail × Realization × Evidence とする。

| 軸 | 記録したい内容 | 失敗させる誤認 |
| --- | --- | --- |
| Subject | Function/Operation/Query/Derived ClaimのTyped Ref、Source位置、Meaningと受理範囲 | Query文字列一致を業務的同一性とする |
| Workload | Target/Host/環境、Window、データ量と分布、Read/Write比、同時数、操作Mix、Cold/Warm、Variant、Sampling/欠測 | 観測した一部を全利用者・全時点とみなす |
| Requirement | SLO、p95/p99、throughput、CPU/Memory/DB/Network/Storage費、鮮度、整合性、可用性、現在Authority/Disclosure | 未設定の性能目標をAIが補完して合格扱い |
| Realization | selected/built/invoked/qualifiedの各Provider/Plugin/Target・exact Lock、実DB Plan、Cache Policy | 構成にあるProviderが実際に呼ばれたと推測 |
| Evidence | 独立Expected、結果粒度、実Benchmark、統計母数と誤差、計測負担、独立再観測、Receipt、現在の資格 | Hit率・DB推定Cost・AI予測を実改善と呼ぶ |

判定は**意味・現在の使用資格・正しさを保ち、対象Workloadの事前に定めた制約を満たす**こととする。速度の改善とp99/書込み負荷/費用の退行は別列で示す。品質に新しい正本を作らず、#565のQuality Envelopeと既存Scenario/Evidenceに用途別投影する。

### 11.2 最適化の候補探索と撤回可能性

発見の起点は (1)不要な実行・N+1・過大取得を減らす、(2)SQL/Index/Batch/Stream/並列/Renderingを改善する、(3)Derived/Cache/Snapshotで再利用する、の三種類。ただし適用順と採否は実測と制約で決める。**Cacheなし、TEI共通Runtime非介在、標準UIなしの完全Headlessも正式な比較候補**とする。

| 対象 | 主な候補 | 異質なリスク |
| --- | --- | --- |
| 取得・演算 | Query Pushdown、N+1除去、Batch、Provider native SQL | NULL、粒度、非単調差分、DB書込負荷 |
| Function / Context | 依存Claimごとの局所維持・再計算、再取得Scope | 部分取得、NOT EXISTS、Top-N、JOIN撤回 |
| Materialization | request-local memo、Memory/Local、分散Cache、Snapshot、Edge配信 | 世代・失効・Stampede、Tenant、Grant、Storage/egress |
| Target / UI / Delivery | Pool、Backpressure、Stream、仮想化、SSR/CDN | p99、追加Hop、初期描画・データの開示 |

分散Cache、独立常駐Agent、自動Adaptive制御はP0の前提にしない。最適化のためだけに新たなCache Host、第二Query Planner、第二Resolver、第二認可基盤を設けない。

## 12. Typed Lifecycle Graphを最適化の入力にする

#3503のOperation/Interaction/Generation Lifecycleを区別したうえで、Read/Derivedの標準フロー、案件差分、合成後の実効Flowを同じTyped Graph/Roleから導出する。**説明図だけで終えず、Generator/検証入力としてSource/Lock/Target/実呼出しを照合する**。#1124/#2497の図表・投影意味と統合し、独自の二重Flow正本は作らない。

~~~text
Function / Operation Read
 → Context・現在のAuthority/Disclosureを検査
 → Strategyの適用可能性を判定
 → Cache reuse判定 または Source/Query Providerへ
 → 必要時のDerived再計算・Materialization
 → Result Scope/Completeness/Revision/Freshnessを検査
 → Consumerに応答（必要なCurrent Disclosureを再検証）
~~~

上は論理的説明であり、全Targetの固定Runtime順序ではない。実装は分岐・省略・合成・静的生成・Target nativeを認める。固定するのは**業務意味、型、必要なSource/Dependency/時点、現在Grant、必要なEffect/Receipt、独立Expected**という保証である。権限チェックの具体位置はHost/Target契約に従い、論理図の1ノードを実行するだけで保証したと主張しない。

| Graphで示す分類 | 変更可能性 | 確認すること |
| --- | --- | --- |
| 不変の保証境界 | Pluginでは迂回不可 | 意味、現在Grant/Disclosure、業務確定値、unknown Effect |
| Typed Extension Binding | 交換可能 | Provider/Adapter/Targetの実選択・実呼出し、生成結果の同等性 |
| Performance Policy | 制約付き調整 | TTL/容量/Prefetch/Samplingが保証を弱めないか |
| Observation / Hint | 任意、除去可能 | Instrumentationの無効化で結果を変えず、余計なHopを残さない |

CLI/Studio/SHOはModel全体、個別ノード、固定/Custom/実効フロー、Typed Ref、Source、Plugin Lock、未解決箇所を同じRead-only情報から案内する。全体集合が不明ならcompleteを主張しない。Graph変更がGeneratorに反映され、Target動作に到達したことを独立実証する。

## 13. Pluginの実装方式と安全なAdaptive Strategy

### 13.1 三段階の選定


1. 標準Function/Rule/Target-nativeで要件を満たせるかを先に検討する。
2. 不足する場合だけ既存#951 Contract/Provider/CompositionへObserver、Analyzer、Proposer、Strategy、Provider、Verifier等の**役割Binding**を追加する。これらをCore固定enumや独立Resolverへ昇格させない。
3. Go単一バイナリに適した静的合成・コード生成を既定候補にし、既存APM/DB/CacheへのAdapterを許す。独立プロセス・動的ロードは更新・隔離・実測利得が必要な案件に限る。

#3278原子的Traitは、Requiredなら実装義務とConformanceを要求し、Advisoryはヒントとしてのみ扱う。Cacheable等の名称だけでは保証にならない。Capability自己宣言、実現、実Target Evidence、現在の実行権限を区別する。Plugin Bundle配布と選択した構成、実Build依存の閉包を分離し、未選択Pluginが結果・ビルド・Cache Keyへ混入しないことを確認する。

### 13.2 安全な最適化の競合とFallback

- **同時Missの合流**：Request Coalescingは純粋なReadでQuery/parameter、Source revision、Context epoch、鮮度条件、Tenant/Disclosure scopeが両立する単位に限定。異なる現在権限や不確定な結果を一つのFutureへ束ねない。書込みの重複防止はCacheでなく既存Operation ID/Effect契約が所有する。
- **旧値復活**：Aが旧版を読みBが更新・失効後、Aが旧版を書き戻す競合では、Sourceの世代・条件付き書込み等が有効な場合にだけ防止保証を主張。Provider能力が未知なら強い鮮度を表明せず、条件に応じ再取得・利用不可・unknownへ戻す。
- **キーと資格の分離**：Key hitでもInput Closure、Rule/Operator/Toolchain/Schema、Snapshot Scope/Completeness、現在Grantを再照合する。Cache evictionは業務確定保存値を上書き・削除する理由にならない。
- **Query意味保存**：Native SQL/Pushdown/Batchの同等性はNULL/missing/非開示、集合粒度、時点、Collation、精度、ORDER、部分取得と非単調なRetractまで検証する。
- **Adaptive**：P0/P1は自由な本番自動書換えを行わず、事前に適合性を検証したStrategyだけを切替候補とする。後続の適応制御ではヒステリシス、最小保持時間、切替数上限、欠測/Exporter障害時の安全なFallback、変更履歴とRollbackを検査する。採用前の提案はSource・業務意味・権限を変更しない。

## 14. 検証に追加する反例（AT-15〜24、すべてNOT RUN）

既存AT-01〜14、#3271 CS-01〜10、#2582 RC-01〜09および#1180/#2514の独立Expectedを再利用する。新しいScenario正本を作らない。

| ID | 操作・故障注入 | 独立Expected |
| --- | --- | --- |
| AT-15 | Query形状・値は同じだがContext epoch、Tenant、Principal、B2C Variantが異なる | Metric集約の可否とResult/Coalescing/Cache共有の資格を分離。不明な共有は拒否 |
| AT-16 | Cache Miss集中と同時更新、旧値読取り後の失効、CAS拒否、再起動 | Backend負荷と旧値復活の両方を確認。未保証はunknown |
| AT-17 | Memory/Localの交換、Selected Provider Bで実Invoked A、未選択Plugin混入 | exact Lock/Build/実Call・Artifact/Consumerを一Attemptで区別 |
| AT-18 | 同じHash/TTLでもRule/Schema/Source revisionとOwner資格が変化 | 旧結果の再利用と現在資格・必要再検証を分離 |
| AT-19 | TEI QueryとNative SQL/BatchでJOIN・NULL・NOT EXISTS・部分取得が混在 | Entity/grain・Retract・空集合/未取得が一致しない場合、最適化棄却 |
| AT-20 | 少数標本・偏ったSampling・p99比較・Trace欠落 | 母数/誤差/代表性/欠測を示し改善確定を拒否 |
| AT-21 | p50とHit率改善、一方でp99/DB write latency/Memory/egress退行 | 変更前に指定したGuardrailで棄却またはRollback可能 |
| AT-22 | Lifecycle Hook・Generator・Targetの変更と手動拡張の再生成 | 固定/差分/実効Graph、実Call、必要保証とTarget別Evidenceを検証 |
| AT-23 | APM/Profiler停止、Exporter障害、Clientは完全Headless | 本体の意味を維持し、未観測を0ms・未実行と解釈しない |
| AT-24 | Adaptive Strategyが負荷変動で頻繁に反転する | 切替抑制/Fallback/履歴・現在資格を確認し、未検証Strategyに切替えない |

比較票の論理列は Subject/Revision、actual selected+invoked Provider/Target/Artifact、Workload fingerprint、Source/Schema/Rule、Independent Expected、Measured distribution/overhead、Correctness、Guardrail、Grant、Verdict、Unknownとする。固定の公開JSON Schemaと誤認しない。

## 15. 既存T0〜T4へ接続する実装順・所有先

上記の横断検討によって新しいT0〜T4や別の最適化工程を作らない。現行§10の「次の実製品縦断」をそのまま利用し、**到達証拠に追加する検証項目**だけを以下のように重ねる。

| 既存段階（§10） | この追補で追加する検証 | 既存Owner |
| --- | --- | --- |
| T0 Design Context | Subject × Workload × Guardrailの出典、Source/未取得/未知の依存、Typed Graphの全体とCoverage | #3262/#3503/#1124 |
| T1 実Query/Provider Read | 実selected/built/invokedの一致、Query planと結果粒度、Observer有無と計測費 | #2586/#2387/#2622/#3270/#951 |
| T2 Cacheなし/Memory/Local | Stampede、generation fence、Result/CoalescingのTenant/epoch/Grant、Cache/Query二方式の同等性 | #3271/#2582/#3278 |
| T3 AI Tuning Packet | 観測Window、p99母数とSampling、開示限定、Plugin/Graph差分の根拠、提案と実証の区別 | #3262/#2516/#3503 |
| T4 独立比較 | Correctness先行、同一Workloadのcold/warmと同時負荷、p95/p99・write/egress・計測負担、Rollback | #1180/#2514/#2600/#565 |

Go単一バイナリにおける実証後、Web/Desktop/完全Headless/Java/.NET等の**異質な実Consumerで追加検証**する。ただし別Targetの実行証拠は転用しない。B2Cの高トラフィック・Edge/CDN、Adaptive Strategyは需要と費用対効果が判明してから扱う（#3509）。実装先は既存#951 Resolver、#3271 Cache/Snapshot、#2582 Context、#3278 Trait、#3503 Lifecycle/Generator、#2600 Build再利用へ戻し、新Core/Optimizer/Cache Host/認可Engine/Hard Gateを作らない。

**撤回条件**：標準Rule/Target-nativeで十分、または構成選択が実Callへ到達しない、意味・Source/Scope/Authorityが保存できない、独立Expectedがない、追加観測/Plugin負荷が利得を超える場合は、当該抽象・自動選択を縮小/撤回する。実装の証拠がないCandidateをOwner受理や本番適用へ昇格しない。

**現況（2026-11-08）**：追加AT-15〜24と上記新規統合試験は設計候補。§10の限定Referenceがあっても実DB Read/実OTel/生成Target性能改善・AI実適用・Owner意味受理は **NOT RUN / NOT ACCEPTED**。既存のGate B/Cや限定Reference PASSを新性能保証に読み替えない。

## 16. 実ReadとConsumer受領の資格を分ける：性能計測の最小縦断（2026-11-08追加）

### 16.1 並行領域で成立した証拠と、まだ成立していない証拠

この節は現行§10のT0〜T4を置換するものではない。別チャット・別PRからの成果を**同じ到達度として足し算しない**ための接続票である。

| 根拠・状態（2026-11-08） | 使える材料 | 成立しない主張 |
| --- | --- | --- |
| merged PR #3520：AI性能観測の19件の合成対照 | Design/Call/Statistics/Delivery/Disclosureの矛盾・欠測を試験用Consumerで判別 | 実DB Providerの呼出し、生成アプリの実計測、実AI改善、外部Tool完全配送 |
| merged PR #3523：TraitとPerformance Observationの分離 | Atomic Trait/Measurement Contract/Optimization Hint/Profileの役割分離 | Traitを宣言すれば実装・Target性能・Owner受理が成立 |
| merged PR #3525：Plugin実呼出しとLifecycle/観測の接続 | 選択と実呼出し、型付きHook、Current Grantの区別 | 対象FunctionのCache/Query最適化が実際に適用済み |
| merged PR #3513：一Function実Readの計画 | order_entry.recent_ordersを入口にQuery→Provider→Rowset→Contextを一Attemptで照合する受入手順 | 実PostgreSQL Read、正しい結果やSource帰属が完結 |
| Draft PR #3524：同一Loadの外部Source/Manifest証拠 | 選択NamespaceのSource原文・Manifestを同一Loadから保持する候補 | 現mainへ導入済み／全Input Closure／Provider全体Snapshotが成立 |
| merged PR #3527：AI観測Reference CI整備 | 索引とWorkflowの固定SHA・Timeoutを修正しmainへ反映 | この変更だけで全main CIや実DB／実性能の合格にはならない |
| Draft PR #3515：Harness/Adaptive Flow横断検討 | Tool配送・Source完全性・実権限を分離する検討材料 | 外部Harnessから本番AI最適化が可能 |

※本表のPR状態は2026-11-08の確認時点。並行作業の後続状態は作業前に再取得する。特定版の限定合成試験PASSを他Targetや現在のmain全体PASSへ転記しない。

性能最適化の実証でまず必要なのは、**実際に処理されたものと、AI/Observerが受け取ったものの因果連鎖**である。観測用JSONの自己整合や実CLI exit=0だけでは、この連鎖は証明できない。

### 16.2 対象一件の「発生 → 処理 → 観測 → 受領」を検証する

初期対象は既存order_entry.recent_ordersのReadのみ。既存Source/Function、Query Binding、DB Provider、Result Sliceを利用し、試験専用の第二Readerや第二実行Hostを作らない。

~~~text
Source / Meaning (version + required closure)
  → Function Data Context / Query Binding (parameter, predicate, epoch)
  → Selected Provider (exact Contract/Instance/Lock)
  → Actual Provider Invocation (attempt, actual SQL or native query)
  → Actual Rowset / Mapping (grain, scope, completeness, source time)
  → Consumer Result (actual projection, current disclosure)
  ├─→ Runtime Trace / Aggregate Observation (sampling, gaps, overhead)
  └─→ CLI / Tool Reply (segments, end marker, delivered bytes)
        → AI Tuning Packet (read-only, purpose-limited)
        → Candidate, never automatic Acceptance
~~~

一Attemptの既存ID/Traceの束縛により、以下を**別の資格列**とする。新しい中央資格判定器や公開Schemaではない。

| 証拠列 | 必要な独立観測 | 合格しても証明しないこと |
| --- | --- | --- |
| Source observed | 取得された実byte/Manifest/Revisionと読取位置、未取得Source | 全依存Closure、Owner Meaning |
| Provider selected | 実解決されたContract/Instance/Target exact Lock | 実際のProvider Call |
| Provider invoked | 実Driver/Adapter/SQLの呼出しと入出力Attempt | DB側Commit、Rowsetの完全性 |
| Result delivered | Mapping後のResult/Scope/部分性、ConsumerまでのReply全件とend marker | 現在の外部LLM開示許可、業務上の確定 |
| Measurement observed | 実Span/Metric/Window、集計の標本・欠測・計測費用 | p99改善、因果性、SLO達成 |
| Candidate qualified | 同じ入力範囲の独立ExpectedとBaseline/Candidateの比較 | Owner Meaning、外部配備、現在Grant |

Current Grant/Disclosureはデータの所在や以前の資格と分離し、Consumer直前・実行時に既存Host/Providerで判定する。許可外のQuery文字列、Parameter、Rowset、Tenant/Principal、Source pathや件数、秘密のdigestをTrace・Tool・AI入力に出さない。実際の外部Consumer配送が確認できなければ、内部CLIでの検査成功は外部AI提案の資格にならない。

## 17. Cacheの整合性・可用性・性能の同時設計

### 17.1 「何を保持したか」を区別し、保存先を自動的に業務正本にしない

| データの性格 | 代表例 | 再利用・失効・保存上の最小区別 |
| --- | --- | --- |
| 入力内容・生成Artifact | TEIの解析結果、Source指紋、生成ファイル | exact byte/実消費Closure、生成器・Toolchain版、ArtifactSet。再利用とEvidence再資格は別（#2600） |
| ReadのSnapshot/Result Slice | 検索行、ページング済結果、外部API Snapshot | Predicate、粒度、取得範囲、observedAt/effectiveAt、complete/partial、Source版、現在のGrant |
| 再生成できるDerivedData | 集計、在庫の参考値、表示用Projection | Ruleと正/負依存、変更時のretract/recompute、Materializationの破棄可能性 |
| 業務上確定した事実 | 確定受注額、決済・承認の記録 | 業務Actionと権威あるProviderによる確定。Cache TTL/eviction/AI再計算では書き換えない |

この分類はCore固定の巨大種別を提案するものではなく、#1641/#3271の既存意味契約へのRole投影である。KON Dataspaceは必要に応じた永続データ・版管理先の選択肢であり、**TEIの二つ目の意味正本や必須Web Host/常駐Cacheではない**。

### 17.2 鮮度と障害時動作を機械的に判別できるようにする

強い現在値保証、版付きSnapshot読取、明示的な古さ上限付き参考表示、完全に再生成可能なImmutable Artifactは**互換な性能Profileではない**。同じTTLで扱わず、用途ごとの有効な鮮度条件・Sourceの更新観測能力・失効経路・利用者への表示資格を確認する。

| 不確実性・障害 | 安全な既定動作（用途依存） | 禁止する短絡 |
| --- | --- | --- |
| Cache HitだがSource revision/negative dependency不明 | Revalidate、再取得、またはunknown/利用不可 | 値一致・TTL残存を現在値の証拠にする |
| Cache-aside旧Read→新Write＋invalidate→旧populate | Source世代/条件付Putが成立すれば拒否。なければ強い鮮度を主張しない | invalidate ACKで競合が消えたとする |
| 複数ProviderでSnapshot時点不一致 | ソースごとの時点を出し、必要な横断一貫性がない結果を保留 | 単一全域Snapshotと偽る |
| Partial/0件/permission-denied | 完全性と開示範囲に応じたClaimだけ利用 | 不存在・合計0・権限なしの件数を推論 |
| Cache/Observerが停止 | 安全なRead Fallbackを持つ場合のみ継続。負荷上限と現在権限を維持 | 無制限のOrigin再取得、Effect再送、未観測0ms |
| Edge/CDN配信・A/B Variant切替 | 配信Variant/Scopeと最終Server権威・Cache Key/Invalidationを別々に検査 | Purge ACKを全Edge失効、表示価格を確定価格とする |

**Failure budgetの分離**：キャッシュ障害からのFallbackは無償ではない。Originの同時Read、Pool待機、Rate Limit、タイムアウト、Backpressure、Queue上限を含め、障害中も許容できる負荷を計測する。Stale-while-revalidateや古い値の提供は、目的と古さ上限が明示されたReadに限る。価格・在庫・決済・外部Effectでは現在の権威あるProviderによる最終検証を省略しない。

### 17.3 最適化の設定粒度と構成衝突

容量、Admit/Evict、TTL、Batchサイズ、並列数、Pool、Prefetch、計測率などは**最適化Policy**であり、業務意味や現在Grantではない。必要なScopeをFunction/Query/Provider Instance/Target/Environmentの中から限定し、設定値のSource・版・選択理由・実Targetへの反映を追跡する。異なるScopeで設定が衝突した場合は、既存#951/#1016の選択・構成結果から説明できない限り暗黙の後勝ちとしない。

Profileは初期値や原子Trait要求集合を展開する便利な束であり、最適化の自動採用・性能保証にはしない。ホット変更する動的Policyも、変更が実際にTargetへ反映されたことと、影響したWindow・Rollback境界を別々に確認する。安全なStrategyの自動選択は事前適格な実装間の限定切替から始め、将来の自由な自動再生成/DDL/本番デプロイと分離する。

## 18. 「速くなった」を判別するBenchmark契約と次の最小実証

### 18.1 Benchmarkの比較条件

最初の同等性と改善の比較対象は、同じFunctionと実Dataset/Workload、独立Expected、実Target、同じConsumer公開Scopeの **Baseline と Candidate** とする。両者に差があれば「どこを同等と扱ったか」を列挙する。

- 目的関数・許容下限：p95/p99、throughput、CPU/Memory、DB Read/Write、Network/egress、運用費、測定費と鮮度・整合性を**比較前**に束縛。Hit率だけを最適化しない。
- 試験入力：Read/Write混合、hot/coldキー、データ分布・偏り、結果サイズ、同時実行数、Batch、ページ境界、長時間稼働/再起動、Source/Schema/Provider exact revisionを保存。
- 実行：Warmup/steady state/失効直後/Origin停止を区別し、試行順によるキャッシュ温度・DB統計変化の影響を抑える。反復・実行順・他負荷・マシン資源を記録し、観測しなかった条件を埋めない。
- 統計：Histogramの単位・境界と標本数、Drop/Sampling、p99の低標本不確実性、計測On/Offによる遅延を保持する。平均やHit率が改善してもp99悪化なら条件次第で棄却。
- 結果：正しさ（Entity Identity、NULL/欠損、Scope、Rule版、Order、Effect）と性能を別判定。実験VariantによるCVRや売上への影響は配信速度と同一Claimにしない（#3509）。
- 適用：Candidate→Preview→独立Expected/Benchmark→変更承認/既存Host Grant→実適用→独立再観測。Index/DDLや外部副作用は既存#3270/#1174のPlan/Receiptに委譲。RollbackでSchemaや業務データが自動復旧すると主張しない。

### 18.2 AT-25〜32：横断的な追加反例（Scenario正本は#1180／全件NOT RUN）

| ID | 故障注入・差分 | 独立Expected・判別 |
| --- | --- | --- |
| AT-25 | 合成Observerは整合するが、実FunctionはProviderを呼ばない／別Providerを呼ぶ | 合成input-consistentとactual invokedを別判定。T1実Callがない限り性能改善資格なし |
| AT-26 | CLI stdoutは全量でもHarness Tool Replyが途中切断/Part欠落/終端欠落 | 内部Read合格・外部Consumer受領不合格を分離。機密内容や件数は漏らさない |
| AT-27 | Provider Manifest/選択規則だけ変更、元Query Source byteは同一 | 実選択Closureを再評価、未知負依存を未変更扱いしない。A→B→Aで旧Evidence/Grantを復活しない |
| AT-28 | 同時MissとCache/Origin停止、Originへの一斉Fallback | CoalescingとBackpressure/上限、失効・タイムアウト・再試行増幅を別々に計測。Effect再送なし |
| AT-29 | 完全性不明の0件ResultをNegative Cache、部分SnapshotでNOT EXISTS | 不存在の無断確定を拒否し、対象のQuery/Contextだけ未確定化 |
| AT-30 | Profile/Policyを変更したが生成Targetに反映されない、異ScopeのPolicyが衝突 | selected/compiled/invokedを実比較。宣言だけの「最適化済み」を拒否 |
| AT-31 | 同じBenchmark入力でもWarmup/サンプル不足/Histogram差/計測負荷が変化 | 有効な条件一致・標本/誤差がなければ改善は未確定、次回の追加測定を示す |
| AT-32 | B2C SSR/Edgeの旧Variantと価格・在庫更新、Cache hit・画面表示は正常 | 配信Variant、Current Disclosure、Server最終確定を区別。表示値から受注/決済成功を導かない |

### 18.3 既存T0〜T4への割付と停止条件

| 既存段階 | 最短の次作業 | 追加CaseとOwner |
| --- | --- | --- |
| T0 | Source/Meaning/Query/Provider/Perf Requirementの必要Closureを一Functionで照合。外部Provider ManifestとBOM選択規則の未確認依存を可視化 | AT-27/30。#3262/#951/#1016、Source実消費の#2600 |
| T1 | order_entry.recent_ordersの実Readを同じAttemptでSource→Binding→選択→実SQL/Adapter→Rowset/Context→計測→Consumerまで照合。実Provider到達を先に証明 | AT-25/26。#2586/#2622/#3270、送出/開示#2516 |
| T2 | Cacheなし→Memory/Localのうち一つに限定し、同時Miss・Source更新・0件/部分集合・Origin障害を比較 | AT-28/29。#3271/#2582/#3278 |
| T3 | 既存Design ContextとAI Tuning Packetに観測事実・欠測・適用資格を載せる。AIの候補が実Targetへ入ったと誤認しない | AT-26/30。#3262/#2516/#3503 |
| T4 | 同じ独立Expected/WorkloadのBaseline/Candidateを実比較し、p95/p99・資源・観測費をMeasuredとUnknownに分ける | AT-31/32。#1180/#2514/#565/#3509 |

**実装順の強制ではないが、主張の依存順は厳守する**。T0のSource/Contractが未知でも限定Traceの調査は可能。しかし実Call・Consumer・Independent Expectedを欠く状態で「AIが正しく自動最適化した」「本番SLOを満たした」と結論しない。Cache/Observerが不要な案件は当該PluginなしのTargetを選択可能とし、同じ保証を達成できる標準実装・Native SQL・案件固有コードを正規の選択肢に残す。

**現在の資格（2026-11-08）**：本節は設計統合と独立反例の追加であり、AT-25〜32、実order_entry DB Provider Read、外部AI Consumer完全受領、Cacheの実同時負荷、実性能Benchmark、運用Adaptive適用、正式Owner意味受理は **NOT RUN / NOT ACCEPTED**。PR #3520の19合成対照・限定Referenceと異なるClaimである。


## 19. 2026-11-09更新：実結合への最小受渡しと性能観測の証拠上限

本節は既存T0〜T4、#2586のM1b→R0、#3271のCache/Snapshot、#1180/#2514の独立Expectedを**置換しない**。2026-11-09の実装状況と比較し、最小の一Function実読取へつながる追補だけを示す。性能機能の追加宣言より、宣言が実行へ届いたことの実証を優先する。

### 19.1 新しい実装事実と、まだ証明されていないこと

| 2026-11-09に確認した既存成果 | 本設計で再利用する点 | なお未確認の境界 |
| --- | --- | --- |
| #2586の最新M1b→R0受渡し | FunctionのQuery/Relation binding→実SQL/parameter→Selected/Invoked Provider→Rowset/Context→Consumerを一Attemptで対照し、B-Disconnected/B-OtherSQL/B-ProviderMismatchを区別 | order_entry.recent_ordersの製品Functionで全辺が実結合した証拠 |
| merged PR #3689（internal/hook.RunReachable）および #3761（ValidateOutcome） | Reachable/Executed/Warned/Skipped/NotEvaluated/Refusedと宣言外Outcome拒否を、実装済みのHook境界から観測できる | after-* phase、DB commit、全reachable path、Cache失効の実完了 |
| merged PR #3757（宣言されたBindingの実行到達調査） | 宣言の有無と実解決・非試験呼出しの有無を区別する既存実装箇所を再利用 | 汎用Optimization StrategyがすべてのFunction/Targetから実Invoke可能という保証 |
| merged PR #3717（PostgreSQLの4系統にDSNとserviceを渡すCIを追加） | DBを使う一部の試験が設定欠落でskipされる問題への対処 | order_entry.recent_ordersのR0、各CI runの成功、最適化Benchmark |
| merged PR #3760（self-build entryのin-process catalog導出） | Plan fingerprintを実処理入力と結び付ける方向、Toolchain/Artifactの測定可能性 | L2-b/G4 qualified self-build、業務Read Cacheの正しさ |
| main #3479（外部Provider原文/Manifest変更）、Draft #3524（same-Load witness） | Consumer World fingerprint不変でも外部Sourceのrevision/manifestを別に検査する必要 | Draftの現main採用・全Input Closure、複数Sourceの原子的同時点 |

上記は同じ一つの実Targetに対する一括PASSではない。PRがmergedでも、そのPRの試験対象と現在のConsumer、Host、Provider、Sourceの資格範囲を保持する。ソース一致・宣言の存在・計測可能性から、実測性能改善やOwner意味受理を導かない。

### 19.2 T1で要求する最小のread-only性能Witness（既存契約のJoin）

例示列は固定の新Schemaではなく、既存Source/Function/Query/Provider/Observation/Receiptを同一Attemptで結ぶ照合票。

| 結合する位置 | 取得するWitness | 明示する非推論 |
| --- | --- | --- |
| Source→Function | exact Workspace/Function ID、Source occurrenceとLoader消費byte、World/Context revision、必要なQuery/Relation | Source Indexやcatalogに載っただけでは実消費・Meaning受理ではない |
| Function→Query/Relation | Query semantic ID、relation path、実Context epoch、bindingが選んだSQL Source/physical relation | FunctionとQueryを別々にinspectできてもM1b成立ではない |
| Query→Provider | 選択DB alias/Instance/Lockと、実呼出しExecutor、実Statement/Parameter/SQL digest、実Read開始・終了 | 実Query plan/名前や合成Connectorの戻り値がProduct実DB Readを証明しない |
| Provider→Result/Context | 実Rowsetのcolumn presence/NULL、grain、row scope、時点、partial/complete/unknown、独立Expectedの観測対象 | 0件、NULL仮列、RLS部分集合を不存在・全件・業務成功にしない |
| Result→Consumer/Observer | Consumerの実受領とpart/end、Observerの実Call、Sample数/Drop/Window、計測On/Off費用、現在のDisclosure | Collector/JSON入力一致≠Consumer全量受領、未観測≠0ms、Cache hit≠Grant |
| Candidate→Measure | same Dataset/Workload/TargetのBaselineと独立Expected、p50/p95/p99、DB/CPU/Network/計測費、更新競合 | Native SQL/Cacheによる短縮を結果・権限の同等性抜きで採用しない |

**一Attemptの対応が確立しなければ、AI Tuning Packetへ観測候補として投影しても実測性能の資格へ昇格させない**。Referenceと本番のSQL物理表が違う場合はQuery ID/画面結果が同じでも別の母集団として扱う（#2586 B-OtherSQL）。Current Disclosureは送信・実Consumerに相対的な既存Host権限で判定する。

### 19.3 キャッシュとLifecycle Hookを混同しない

HookでReached/Executedが報告されても、それは該当段階のHook処理の局所観測であり、Sourceの更新、CacheのInvalidation、外部ProviderのRead、確定後Effectの完了を示さない。#3689のReferenceReachはafter-*へ届かず、NotEvaluatedとSkippedも異なる。#3761のOutcome語彙検査は宣言外の返却を拒否するが、Cache鮮度や全Pathの動作を保証しない。

Cacheへの再利用・更新可否は既存#3271のSource Observation/Derivation/Materialization/Reuse Eligibility/Current Disclosure/Business Acceptanceと、#2582の結果Claim・正負依存で判定する。機械的なTTL、Hook成功、DB ACKだけで完全Snapshotや現在値を宣言しない。実測できない外部Source/Manifest変更、NOT EXISTS・0件/partial、Tenant/Context epoch、Miss競合時は必要なClaimをunknownに戻し、要求された鮮度を満たせなければRead fallbackまたは最適化不適格を選ぶ。無制限FallbackによるOrigin過負荷はAT-28を使って別測定する。

### 19.4 次に実行する一連の比較を最小化する

- **T0/M1b**：#2586のB-DisconnectedとB-OtherSQLを、現在の実Sourceと既存Reader/Bindingの表だけで先に固定する。物理 SQL source、DB alias、Query/Relation、Contextのどの辺が未結合かを列挙。未確定の結合を補うために新Resolver/Query DSL/Providerを作らない。
- **T1/R0**：既存の実Function readで正例B-Read+を一Attemptだけ成立させ、B-ProviderMismatch、Column Presence/NULL、部分取得を独立負例として加える。実PostgreSQLが未到達なら、instrumented Connectorの限定実証で止まり、Product実DB Readと書かない。
- **T2**：T1で成立した同じQuery/Context/Read Cohortについて、まずNo Cache対Request-localまたはMemoryの一方式を比較。必要な場合だけLocalへ進み、AT-16/28/29を同じIndependent Expectedに接続。Cache以外のN+1/Pushdown改善も比較候補に残す。
- **T3/T4**：実Readの観測範囲だけを#3262のread-only Contextへ投影し、完全受領・目的限定の開示を検証。実測BenchmarkはCorrectnessとp95/p99・費用を二枚に分け、採用/撤回/rollback判断を記録。CLIだけの正例をStudioや外部AIの証拠へ流用しない。

**今回の更新状態**：既存実装の資格の再基準化と計画文書の整合修復であり、この変更による実Function→実DB Read、Memory Cacheの性能比較、外部AI受領、Adaptive Tuning、正式Owner意味受理は **NOT RUN / NOT ACCEPTED**。#1119のG0〜G4や別の受理Gateは変更しない。


## 20. キャッシュ採用の経済性と更新境界：実Read後に判定する最小Policy（2026-11-09追補）

§19のM1b→R0を最優先とし、実Read未成立のまま新たな最適化Runtime・固定Schema・計測サービスを設計しない。§17のCache正しさと§18のBenchmarkを、**「使えるCache」から「使う価値のあるCache」への判断**に接続する。下記の指標・計算は候補比較の分析票であり、TEIの正式DSL・SLO・公開APIではない。

### 20.1 速度・正しさ・費用を同時に評価する

単純な平均読取時間の近似は、Hit率を h、Hit時の全費用を H、Miss時の全費用を M として E[T] ≈ hH+(1-h)M と置ける。ただし H はlookup/serialization/authorization/revalidation、M はlookup失敗＋origin read＋materialization/putを含む。**同時実行、queue、tail latency、無効化・再取得、write amplification、外部egress、観測費用を含まない近似**であり、これだけでp95/p99やSLO達成を宣言しない。

採用判断には少なくとも次の比較を残す。

| 比較軸 | 最小の測定/見積り | 拒否・撤回条件 |
| --- | --- | --- |
| 正しさ・利用資格 | 同一Function/Query/Result Scope、独立Expected、Source/Rule/Provider revision、現在Grant | 不一致、未知依存をFreshと誤認、部分集合から不存在を断定 |
| 利用者体感 | p50/p95/p99、first-result、cold/warm、concurrency、timeout率 | 平均改善でも事前Guardrailのtail退行 |
| Origin負荷 | Query回数/行数/CPU、Pool wait、DB write latency、miss時のburst | Origin障害やCache停止で許容負荷を超過 |
| Cacheの運用負担 | memory/size、eviction、write/refresh、storage/egress、Provider運用、観測overhead | Cacheなし・Request-localより費用対効果が悪い |
| 変更と復旧 | invalidation遅延、競合、Provider再起動、rollback可能性 | 条件付Put/世代判定がないのに強い鮮度を要求 |

**Cacheなしを正式な対照群として残す**。最初の実証はNo CacheとRequest-local/Memoryの一方式だけで、同じRead Cohortを比較する。複数Cache Backend、Adaptive TTL、Redis/S3/CDNは最初の実Readと正しさの成立条件ではない。Cache以外のBatch/Query Pushdown/Index改善が優れていればそちらを採る。

### 20.2 更新の「通知」と「確定」を区別する

Business ActionやDB更新を契機にCacheを失効させる場合、**Hook到達、Command ACK、DB commit、Cache invalidation、他Consumerの観測**は別の事実である。#1129/#1174のOperation/Effect、#3271のReuse Eligibility、#2582のClaim依存に接続し、以下の境界を確認する。

1. 更新前・未確定・rollback・Effect unknownでは「DBの新しい確定値が存在する」と推測しない。先行失効による余分なMissは許容可能でも、確定していない値を新しいSnapshotとして公開しない。
2. Commit成功が独立に観測できても、非同期失効通知の配送・全Cacheへの適用・Edge purge完了を自動的に証明しない。必要な鮮度を満たすReadでは、Source revision/世代の再検証や安全な直接Readを使う。
3. Outbox/CDC/Event連携は**選択可能なProvider手段**とし、同期の業務不変条件を非同期通知だけへ移さない。CDC gap、遅延、重複、逆順、再起動時は受信済みイベントの有無をSource全体の完全性とみなさない。
4. Eventからの再計算・再公開はResult Claimの正/負依存、Context epoch、Source Scope、Ownerの現在Grantを再評価する。NOT EXISTS/Top-N/LEFT JOIN/集計はinsert-only更新で済ませない。
5. Cache invalidationは業務Actionの冪等化・Effect unknown回復の代替ではない。再試行の判断は既存Operation ID/Receiptの所有領域へ戻す。

**例外を明示**：強い一貫性を保証するProviderの条件付Read/Writeやトランザクション内更新が独立に証明された用途では、上記の保守的Fallbackを縮小できる。ただし全Backendにその能力を仮定しない。

### 20.3 観測の粒度と安全なコスト

TEIのTyped Query/Function IDと外部ObservabilityのQuery Summaryは役割が違う。低Cardinalityの集計キーを優先し、実SQL原文、パラメータ、個人・Tenant・Principal IDをMetric labelに使わない。OpenTelemetryのDB Semantic Conventionsでは低Cardinalityの `db.query.summary` が推奨され、`db.query.text` と `db.query.parameter.<key>` には収集・秘匿・容量上の注意がある（外部参照： https://opentelemetry.io/docs/specs/semconv/db/database-spans/ ）。必要な詳細Traceは権限・期間・Samplingを限定し、観測OFFの通常Targetを正式な比較対象に残す。

計測点は既存Provider/Host/TargetのObserver Bindingで足りるかを先に確認する。外部APMやOTel SDKは任意Adapterであり、収集器の停止・Exporter失敗・Span欠落を業務Read失敗や0msとしない。AIへ渡すPacketには集計Window、標本数、欠測、実Target/Provider、現在の開示資格を添える。実Traceの完全受領が未確認なら「提案候補」に留める。

### 20.4 次に実行する独立反例（既存#1180 Scenarioへ追記候補／NOT RUN）

| ID | 故障注入 | 独立Expected |
| --- | --- | --- |
| AT-33 | Hit率は高いがserialization・再検証・GC・egressが増え、p99と総費用が悪化 | 正しさが成立してもCache採用を拒否可能。No Cacheを残す |
| AT-34 | 更新Hookがexecuted、DBはrollbackまたはEffect unknown、Cacheだけ失効/再生成 | Hook成功≠Commit、未確定値をFresh Snapshotにしない |
| AT-35 | DB Commit後、Outbox/CDC失効通知が遅延・重複・欠落、Cache Hitが継続 | 通知受領≠全Cache失効。必要鮮度に応じ再検証/Read fallback |
| AT-36 | 同じQuery Summaryに異なるTyped Query/Provider/Scopeが混在し、Metric labelにParameterを混入 | Summary統計は性能候補の探索に限定。業務同一性や現在Disclosureを推論せず、秘密・高Cardinalityを出さない |

**Owner/順序**：AT-33は#3271/#1180/#565、AT-34/35は#1129/#1174/#3271/#2582、AT-36は#3262/#2516と既存Observer/Provider Contractに接続する。§19のT1/R0成立後にT2でAT-33〜35を評価し、T3/T4でAT-36と費用/実測比較を扱う。新Owner、中央Cache Manager、Telemetry Host、Hard Gateを作らない。

**状態**：本節は現行mainの既存計画への設計追補のみ。AT-33〜36、R0実DB Read、Commit→Cache失効の実End-to-End、Cache性能・費用の実測、外部AI配送・自動最適化、Owner受理は **NOT RUN / NOT ACCEPTED**。
## 21. 最小Performance Experiment Protocol：改善の採否を再現可能にする（2026-11-09追補）

§19のM1b→R0を優先する方針は維持する。本節は実Readが成立した後の**最初の比較実験の具体的な受入票**を定める。Cacheや新しいOptimizerを実装する前に、既存#1180 Scenario/Independent Expected、#2586実Read、#3271 Cache、#2600再利用、#565 Quality Envelopeへ既存の記録を渡す。公開CLI/APIや独立Benchmark Runnerの新設は決定しない。

### 21.1 一つのFunction・一つの変更・二つの比較

初期Subjectは`order_entry.recent_orders`のread-only Query。比較候補は **A = No Cache** と **B = Request-local memo または Memory Cache の一方**。同じSource/DB Dataset/SQL物理実体/Query・Rule版/Provider Instance/Target/Host、Current Disclosure、入力パラメータと実Consumerを使用する。A/Bで異なる条件は意図したPolicy/実装とその不可避な費用だけに限定し、変更点を差分票に残す。

**A/B実験の開始資格**：
- #2586のB-Read+でSource→Function→Query/Relation→実Provider→Rowset/Context→Consumerが同一Attemptで実観測され、B-Disconnected/B-OtherSQL/B-ProviderMismatchの負例を判別済み。
- Providerの実Readを計測する観測点がある。計測なしのResult一致、合成Trace、Mock Providerだけなら性能評価に進めない。
- 独立ExpectedをBaseline/Candidateと別のSourceで固定し、NULL/未取得/部分集合、Query grain、結果順序、Scope、Current Disclosureを確認できる。
- DB・Source・Plugin/Targetの正確な版と選択された実装を記録。測定中の更新/統計reset/Schema driftが不明ならそのRunを比較不適格とする。

### 21.2 再現可能なWorkloadの最小行列

| Workload | 狙い | 最小の反証 |
| --- | --- | --- |
| W1 Cold | 初回実Read、Cache構築費と初期遅延 | Warm Hitの速度をColdの速度として表示しない |
| W2 Warm・同一Scope | 再利用によるDB Read回数・遅延の差 | HitだけでCurrent Grant・入力閉包を省略しない |
| W3 同時Miss | Coalescing/Stampede、OriginとPool負荷 | Cache Hit率改善でもOriginピーク/Queueが悪化 |
| W4 更新と競合 | 更新前Read、Invalidate、旧populateの世代競合 | 古い結果を再格納してFreshと表示 |
| W5 0件・Partial・非単調Query | NOT EXISTS/LEFT JOIN/Top-Nと負依存の再取得 | 不完全な0件をNegative Cacheで不存在確定 |
| W6 異Scope・認可変化 | Tenant/Principal/Context epoch/Grant変化 | 同値Keyを根拠に結果や件数を横断開示 |
| W7 Origin/Cache障害 | Timeout、Backpressure、FallbackとDB負荷 | 無制限Fallback、未観測を0msとみなす |
| W8 Policy/Plugin変化 | Provider版/Instance、Rule/Schema、未選択Plugin、A→B→A | 未選択変更で過剰失効、旧Evidenceの自動復活 |

W1/W2は初期の性能対照、W3〜W8は正しさ・可用性の受入を中心とする。**すべてを一度に自動化することは前提にしない**。W1/W2が成立するまでW3〜W8はFixtureと独立Expectedを先に固定し、実測未了を明記する。

### 21.3 計測票：性能の改善と保証の成立を分離

一つのRunに最低限、Subject/Source/DB Dataset/Workload fingerprint、exact Plugin/Target/Provider Instance、A/B variant、Actual Call、Rowset scope、独立Expected/Actual、Current Grant、観測Window、サンプル数、sampling/drop、測定On/Off費、cold/warm、p50/p95/p99、DB read count、Pool wait、CPU/Memory、Origin request peak、Storage/egress費、Correctness verdict、Performance verdict、Unknownを関連付ける。

p99の標本数が不十分ならp99を比較不能とし、数値を補完しない。計測単位・Histogram境界・実行順・DB統計・他負荷が異なるRunも比較不適格。低CardinalityなQuery summary/Operation別durationを基本とし、Raw SQL、Parameter、Tenant/Principal、PKをMetric labelへ出さない。OpenTelemetryは任意Adapterとし、外部Collectorが停止しても業務Readは維持する。計測のための無制限Trace保存・高Cardinality集約を標準にしない。

**採用判定は二段階**：まず独立Expected、Source/Scope/Grant、Effect境界の正しさを満たす。次に事前指定したSLO/Guardrail（p95/p99、Originピーク、DB write latency、CPU/Memory、運用費、測定費）を満たす場合だけ採用候補とする。SLO数値・許容費用は利用者/案件の要件に基づき、AIが任意の閾値を正本化しない。速度の改善率だけで採用しない。

### 21.4 Cache以外を対照に残す理由

Cache Bが性能改善しても、N+1除去、Query pushdown、Batch、Index、Data Context局所再計算、Target-native実装の方が単純で速い可能性がある。最初のA/Bを成立させた後に、同じWorkload/Expectedで **C = Cacheを使わないQuery/Batch改善** を比較できる余地を残す。ただしCの実SQLやDDLをAIが本番へ無承認適用しない。Native SQLとTEI Queryの同等性はNULL/時点/精度/粒度/順序/部分取得で検証し、DB Index/DDLは#3270の既存権限とMigrationへ委譲する。

### 21.5 実装順・停止条件・責務

- **E0**：#2586のM1b→R0実Readを成立。未成立なら本節の実Benchmarkは開始しない。
- **E1**：#1180/#2514にW1/W2と更新・Scopeの独立Expectedを固定。#3271の既存CS/ATと重複したテスト正本を作らない。
- **E2**：同一ReadでA/Bの実Provider invocationとConsumer結果を照合。実測Windowと正しさのEvidenceを分離。
- **E3**：W3/W4/W7の負荷・失効・障害対照を追加。意味保存や負荷制御が成立しない場合はCacheを不適格とし、再取得またはNo Cacheへ戻す。
- **E4**：必要な場合だけC（Query/Batch）を追加し、採用/撤回を#565の既存品質基準へ引き渡す。

本節は新しいP0やHard Gateを追加せず、§10/§19のT0〜T4の中で**一つの実験を完結させるためのチェックリスト**である。PluginのObserver/Strategy/Cache Providerは既存#951/#3278/#3271のBindingに置き、Targetに不要なRuntime Hopを作らない。AI Tuning Packetは独立Expectedと実測の説明を助けるが、採用/Owner意味受理の権限を持たない。

**状態（2026-11-09）**：E0〜E4、W1〜W8の実Function/実DB比較、Cache/Origin障害負荷、実TargetのSLO達成、AI自動適用、正式Owner受理は **NOT RUN / NOT ACCEPTED**。§19の局所Hook/CI/Binding成果をこれらの実行証拠へ昇格しない。

## 22. DSH等の外部HarnessとAI性能チューニングを接続する最小契約（2026-11-09追補）

§21のW1〜W8は**実行系の正しさと性能を検証する実験**であり、DSH/AI Harness自体の評価ではない。外部Harnessを接続しても、性能改善の正本はTEIのSource/Meaning/Function/Query/Provider/Independent Expectedと実Targetに置く。HarnessはTaskを実行・提案する交換可能なConsumer/Coordinatorであり、TEI Studioや生成Runtimeの必須依存にしない。

### 22.1 連携の入口：選択したUI要素から目的限定のTaskを作る

Studio上でFunction、Query、Relation、Result列、Rule、Lifecycle Hook、Provider Binding、性能Traceのいずれかを選択したとき、**選択対象の安定ID・Source位置・Revision・現在の閲覧資格・周辺Dependencyの取得範囲**を、既存のDesign Context/read-only CLI/APIからTask Contextとして組み立てる。未選択の機密データを便利さのために自動収集しない。

例：`order_entry.recent_orders`のQueryを選択し「遅い理由を調べる」と指示した場合、Harnessへ送る最小情報は、利用者が許可したFunction/Query構造、対象Target、観測済みの低機密な集計値、目的・要求SLO、既存Task/Attempt参照である。Raw SQL/Parameter/Rowset/テナントID/個人情報/秘密Source pathは既定で外部送出しない。実測Traceの取得が許可されない場合は不足として示し、AIに推測した実行計画を事実として埋めさせない。

**Studioの選択・説明・提案**と、**TEI側の変更プレビュー・適用・再観測**を別操作とする。選択したUI要素は操作対象の指定であって、書込み権限・本番DB操作権限・外部送出許可を与えない。変更は既存Source編集/Plugin Binding/Target Generator/Host Grantを通し、採用前に独立ExpectedとW1〜W8の該当条件を再検証する。

### 22.2 Harness/モデルを交換しても評価可能な三つの経路

| 経路 | 実行者 | 成立する資格・停止条件 |
| --- | --- | --- |
| **H0 決定論的** | TEI CLI/既存Query・Trace・Rule検査 | AIなしでも実Read・Baseline/Correctness・診断を実行。AI提案がないことを欠落としない |
| **H1 外部Harness** | DSH等の既存Harness＋許可済みTool Adapter | Task Contextの実受領、Tool request/reply全件、モデル入力への配送、外部送出範囲を別記。Tool欠落/切断は比較不適格 |
| **H2 交換Harness/Provider** | 別Harnessまたは別モデルProvider | 同じTask/Source/Workload/Expectedと許可Scopeで比較。モデル・Harness・Skill・専門Pluginを一度に変更せず差分を特定 |

定額契約や外部モデルの提供形態は**接続できること・規約上許されること・技術的に再現できること**を別々に確認する。ChatGPT Plus等の消費者向けUI契約を、TEIから利用可能なAPI資格・自動操作権限・商用再配布権限と同一視しない。DSH固有の認証・通信・Tool/Session互換は実版のAdapterで検証し、DeepSeek由来の名称やOSSライセンスから接続先の利用規約まで推論しない。

### 22.3 外部Harness評価で必須の非推論

- **Transport**：CLI stdout完了、Tool Reply受領、モデルへ送信されたPart/終端、最終回答は別観測。過去H2/H3の不完全Sessionを再利用して「比較適格」としない。
- **Authority**：read-onlyと表示されたSkillでも、実argv/OS権限/外部送出/Session Logの各Sinkを独立確認。OS隔離・Egress制限・機密非流出を未実測のまま宣言しない。
- **Quality**：AIがもっともらしいSQL/Cache変更を提案しても、Product実Read、独立Expected、Baseline/Candidate性能、実適用・Rollback、Owner意味受理は別資格。
- **Cost**：モデル呼出し料金、利用枠、Tool/Retry回数、Context再送、キャッシュ効果、所要時間、失敗率、実装/保守費を同じTaskで記録。定額プランの利用枠・自動化可否は実サービスの現行契約で再確認し、価格や利用可能性を設計前提として固定しない。
- **Availability**：外部Harness・AI Provider・Observerが停止しても、AIを必要としないTEI Function/Query/Targetと手動開発は動作可能にする。自動最適化は任意Plugin。

### 22.4 実装優先順位と撤回条件

**H0**：まず§19/§20のM1b→R0、No Cache Baseline、W1/W2と独立Expectedを成立。**H1**：既存#3262のDesign ContextをUI選択に対応付け、read-onlyの小さなTuning Packetを実CLI/Consumerへ配送。**H2**：DSH一種類で実Tool受領・開示・結果を検証し、H0と同じTaskで候補品質・所要時間・費用を比較。**H3**：必要時だけ別Harness/Providerを追加し、再現可能な差分を測る。

既存#2509/#2515（外部Harness隔離・配送）、#3262（Design Context）、#951/#3278（Plugin実装資格）、#2586（実Read）、#3271（Cache）、#1180/#2514（独立Expected）のOwnerを維持する。新しいAgent Loop/TEI独自Harness/共通AI Gateway/独立Proof Storeは、複数の既存Harnessで再現する重大な未解決欠陥と費用対効果が確認されるまで採用しない。

**状態（2026-11-09）**：本節はUI選択→外部Harness→検証可能な改善提案の設計接続である。Studioの実UI要素選択からDSHへの配送、外部Harness完全受領、利用規約上の各モデル接続、実DBの性能改善、Owner受理は **NOT RUN / NOT ACCEPTED**。


## 23. 固定Selectionと共有Harness Sessionを性能チューニングへ接続する（2026-11-12）

本節は、TEI Studio/SHO/SUI/SEI/CSWにおける「UIで対象を安定固定してAIへ指示する」議論を、§22のHarness連携に接続する。**選択状態はFocus/カーソル位置/画面遷移/プロンプト入力と独立**し、偶発的な解除や暗黙の別対象への付替えを禁止する。共通Harness Sessionは製品をまたいで共有可能とするが、ProductごとのSource・Meaning・Grant・操作権限を継承しない。

### 23.1 固定Selectionの最小表現

固定Selectionは新Canonical Schemaではなく、既存Design Context/Model Locate/Source位置を参照するread-onlyのTask入力とする。

| 項目 | 最小の意味 | 不正な推論 |
| --- | --- | --- |
| Product/Workspace/Subject | 選択したFunction、Query、Rule、UI部品等のQualified IDと要素種別 | 表示Label/行番号が同じなら同一対象 |
| Source/Revision/Location | 原Source、Model/World Revision、Source occurrence、関連する型付き参照 | 移動・A→B→A後に自動で旧Meaningを受理 |
| Selection scope | 明示した単一/複数対象、関連Data、依存範囲、固定/追加/解除の操作履歴 | カーソル移動や他タブFocusで暗黙に変更 |
| Purpose/Audience/Disclosure | 今回の分析目的、送出先、Current Grantと非開示部分 | 共有Sessionが全製品の認可を継承 |
| Evidence/Unknown | 実Provider/Consumer/計測の観測資格、未解決の辺 | 静的Model MapやAI説明を実DB実測へ昇格 |

GUIでは固定Selectionを明示的な「対象・文脈棚」として見せ、CUIではQualified ID/Revisionと選択範囲をread-only inspect/明示的add/remove/clearで扱う。具体的なコマンド名・UIウィジェットは既存#1124/#1158/#1824の設計へ委ね、ここで新APIを確定しない。複数選択時は各対象のRevision/Grantを個別評価し、関連性の推測だけで結合しない。

### 23.2 変更・失効・再開の動作

- Source/Model/Plugin/Provider/Grantが変化したら、旧Selectionを黙って別対象へ付け替えず、**stale/partial/unavailable/permission-denied**等の既存状態で提示し、再解決・再選択を明示する。
- Taskを別製品へ引き継ぐ際は、Activity/Harness Sessionの識別と、Product別Selection/Operation/Resultを分ける。引継ぎ先はSource/Revision/Disclosureを再評価する。AIの説明や候補をTEIのAccepted Meaningへ自動転記しない。
- CLI stdoutの完了、Harness Session Logへの永続化、Tool Reply全Part/終端の受領、モデルContextへの配送、最終回答を独立観測する。4,096B等の既存Transport上限により後続Partが切れた事例を、単なる保存成功で合格にしない。
- PreviewとApplyは別操作。UI選択・自然言語指示・共有Sessionの存在はSource変更、DB write、認可変更、Build/Effect実行の権限を意味しない。既存TEI Application API/Hostの認可と独立Expectedで再検証する。

### 23.3 性能改善の一つの縦断シナリオ

**S0** Studioで`order_entry.recent_orders`のQuery/結果列を固定選択し、Source/Revision/Scopeをinspect。**S1** #2586の実Read B-Read+の成立範囲だけを§21のA/B実験へ関連付ける。**S2** §22 H0（AIなし）のBaseline/独立Expectedを表示。**S3** 許可したread-only Contextのみを外部Harnessへ送り、完全受領を確認。**S4** AIのQuery/Batch/Cache改善候補をPreviewし、変更対象と影響範囲を表示。**S5** Ownerが適切な変更を受理した場合のみSource/Plugin/Targetへ適用し、同じIndependent ExpectedとW1〜W8の該当条件で再測定。**S6** 別製品の共有Sessionへ引き継ぐ場合はProduct別Revision/Grantを再確認する。

失敗例は、F1 Focus移動でSelection解除、F2 Source更新後の別対象誤選択、F3 権限変更後の旧Context送出、F4 Tool Reply欠落を完全受領扱い、F5 AI候補を未承認Apply、F6 合成Traceを実Readと誤認、F7 Session共有をDB権限共有と誤認。これらは既存#3262/#2586/#1180/#1452/#1824のScenarioへ接続し、新Case正本・Harness Host・Registry・認可Engineを作らない。

### 23.4 優先順位と実装資格

**P0** は#2586の隔離PG Product Seedと実Function→実Provider→Rowset→Consumerを一Attemptで接続すること。**P1** は既存Model Locate/Design ContextとGUI/CUIの固定Selectionの同一性・失効を確認すること。**P2** はDSH等一Harnessへの限定送出・Tool全量受領を確認すること。**P3** は同じWorkload/ExpectedでAIなし/AIありの提案品質・性能・費用を比較すること。P1〜P3をP0の前提条件へ逆転させない。

既存の#2586（実Read）、#3262（Context/Consumer）、#1452/#1124/#1158/#1824（対象指定と変更）、#951/#1251（Provider/Plugin）、#3271（Cache）、#1180/#2514（独立Expected）、#565（Quality）へ責務を戻す。SHO/SUI/SEI/CSW/KONは必要時のConsumer/連携先であり、TEIの初回実ReadやCache比較の必須Host/Storageにしない。

**状態（2026-11-12）**：固定Selectionの実GUI/CUI一貫性、製品横断Sessionの実配送、DSH完全受領、実DB A/B、AI改善の適用、Owner受理は **NOT RUN / NOT ACCEPTED**。本節は計画接合と反例整理のみ。


## 24. 実験Evidenceの外部保管とAI Contextへの選択的開示（2026-11-12）

§21のBenchmark、§22のHarness、§23の固定Selectionは同じ一AttemptのSource/Revision/Actual Invocation/Consumerを参照する。ただし**Evidence本体をGitHubへ全量アップロードすることは要件としない**。Gitには小さな宣言・Manifest参照・再現手順・期待値を置き、巨大なTrace/Histogram/Rowset Snapshot/画面記録等は既存Local CASまたは選択可能なNAS/S3 Providerへ保持する。KON Dataspace、外部Evidence Store、クラウドを必須のTEI Runtime依存としない。

### 24.1 保管と受理を別の次元に分ける

| 次元 | 残すもの | 証明しないこと |
| --- | --- | --- |
| Identity | Subject/Revision/Attempt/Source fingerprint、Artifact/Content digest、型・サイズ・Retention分類 | digest一致だけでSource Meaning、現時点のGrant、Consumer受領 |
| Storage resolution | Local/NAS/S3のProvider識別子、Object keyまたはopaque locator、実読取可否、byte照合 | Git commitに参照があるだけでObjectが現在取得可能 |
| Qualification | 実Provider呼出し、Result/Consumer受領、独立Expected、測定Window/欠測、Benchmark verdict | Object保存成功だけで実Read・性能改善・Owner受理 |
| Disclosure/Egress | 送出目的、宛先、最小Projection、現在のGrant、遮蔽と開示記録 | Git参照の公開可否からRaw Evidenceの公開可否 |
| Lifecycle | retain/expire/delete、失効・再計測・復元の記録、offline時のunknown | A→B→Aやdigest一致による旧Evidenceの自動復権 |

この表は既存#2600のWorld/Artifact/Evidence区分と#3262のDesign Context、#3271のCache再利用資格を**参照**するだけで、第二Proof Registryや新しい固定Canonical Schemaを追加しない。CASに同じbyteが残っていても、Toolchain/Provider/DB Dataset/Grantが違えば当該Performance Claimの再利用は独立審査とする。

### 24.2 外部Harnessへ渡すのはEvidence本体ではなく許可されたProjection

AIが「遅いQueryを調べる」と指示された場合、まず固定SelectionのQualified ID/Revisionと、低Cardinalityの集計、測定条件、正しさの判定、unknown、必要なら不透明なEvidence参照を送る。**NAS/S3 URL・認証情報・SQL引数・個別Rowset・内部pathを無条件に送らない**。HarnessがRaw Evidenceを必要とする場合は、既存Host/ProviderのCurrent Disclosureを再評価して、目的・範囲・期限を限定した明示操作を要求する。AIの応答に含まれるLocatorは参照情報であって、Hostの権限を変更しない。

### 24.3 既存Driver/CIへ接続する反例

- **EV-01** Git commit/ManifestはあるがNAS/S3 objectが消失、権限拒否、offline。再現不能はunknown/unavailableとしPASSを流用しない。
- **EV-02** 同digestのbyteが別Toolchain/DB Snapshot/Targetで観測された。保存Objectを再利用できてもPerformance Evidenceを再受理しない。
- **EV-03** Source A→B→A、Provider ManifestまたはDB行/RLSだけ変更。World fingerprint同値で旧Benchmark/Current Grantを復権しない。
- **EV-04** Harnessへ小さな統計Projectionのみ許可したが、Raw SQL/Rowset/NAS URLがTool ReplyやSession Logへ混入。開示違反として拒否し、保存成功と切り離す。
- **EV-05** CLIはEvidenceを出力したがTool Reply part/end欠落、またはAIが未読の外部Objectを読んだと主張。Consumer/Model受領の資格を付けない。
- **EV-06** Cache Hitが増えたがSource失効・p99・Originピーク・DB write latency・計測費が不明。正しさ/性能の採用判定を保留。
- **EV-07** Selection Pinは維持されているがSource occurrence/Grantが変化。古いEvidence参照を含むAI Contextを自動再送しない。
- **EV-08** Local-only設定でGitHubへRaw Evidenceを送らずに独立Expectedと再現手順を保持。クラウド未使用を不合格にしない。

EV-01〜08は新しいCase SSOTではなく#1180/#2514のScenarioに付ける**反例候補**であり、既存#2600/#3262/#3271の試験と重複実装しない。最初の実証はLocal-onlyの一Attempt（Manifest→Local CAS保存→再読取・byte照合→限定Context送出拒否/許可→独立Expected）とする。次にNAS/S3はProvider別に、障害・再取得・Retention/削除を確認する。実DBのM1b→R0をこの外部保管実証の前提にも逆依存にもせず、互いの資格を別記する。

**状態（2026-11-12）**：既存Local CAS機能の存在と本節の計画接合は、Git同期・NAS/S3実運用・選択的AI開示・EV-01〜08実行・実DB性能Benchmark・Owner受理の合格を意味しない。これらは **NOT RUN / NOT ACCEPTED**。
