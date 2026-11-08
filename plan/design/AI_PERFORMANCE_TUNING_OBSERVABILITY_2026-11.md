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
