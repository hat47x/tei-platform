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


## 9. 横断統合：性能要求から実装とEvidenceまでの最小契約（2026-11-08追補）

本追補は#3271 Cache/Snapshot、#2582 Data Context、#3278 Trait、#3503 Lifecycle、#3509 B2C、#2600 Build再利用を、既存AI Tuning Packet設計と接続する。**各分野の正本を作り直さず、組込み標準と案件固有実装を同一条件で比較可能にする**ことが目的。以下の名称・Schema/API・Plugin Roleは設計候補であり、採用済みの公開契約ではない。

### 9.1 最適化の判定単位

最適化の単位は Subject × Workload × Requirement/Guardrail × Realization × Evidence とする。

| 軸 | 記録したい内容 | 失敗させる誤認 |
| --- | --- | --- |
| Subject | Function/Operation/Query/Derived ClaimのTyped Ref、Source位置、Meaningと受理範囲 | Query文字列一致を業務的同一性とする |
| Workload | Target/Host/環境、Window、データ量と分布、Read/Write比、同時数、操作Mix、Cold/Warm、Variant、Sampling/欠測 | 観測した一部を全利用者・全時点とみなす |
| Requirement | SLO、p95/p99、throughput、CPU/Memory/DB/Network/Storage費、鮮度、整合性、可用性、現在Authority/Disclosure | 未設定の性能目標をAIが補完して合格扱い |
| Realization | selected/built/invoked/qualifiedの各Provider/Plugin/Target・exact Lock、実DB Plan、Cache Policy | 構成にあるProviderが実際に呼ばれたと推測 |
| Evidence | 独立Expected、結果粒度、実Benchmark、統計母数と誤差、計測負担、独立再観測、Receipt、現在の資格 | Hit率・DB推定Cost・AI予測を実改善と呼ぶ |

判定は**意味・現在の使用資格・正しさを保ち、対象Workloadの事前に定めた制約を満たす**こととする。速度の改善とp99/書込み負荷/費用の退行は別列で示す。品質に新しい正本を作らず、#565のQuality Envelopeと既存Scenario/Evidenceに用途別投影する。

### 9.2 最適化の候補探索と撤回可能性

発見の起点は (1)不要な実行・N+1・過大取得を減らす、(2)SQL/Index/Batch/Stream/並列/Renderingを改善する、(3)Derived/Cache/Snapshotで再利用する、の三種類。ただし適用順と採否は実測と制約で決める。**Cacheなし、TEI共通Runtime非介在、標準UIなしの完全Headlessも正式な比較候補**とする。

| 対象 | 主な候補 | 異質なリスク |
| --- | --- | --- |
| 取得・演算 | Query Pushdown、N+1除去、Batch、Provider native SQL | NULL、粒度、非単調差分、DB書込負荷 |
| Function / Context | 依存Claimごとの局所維持・再計算、再取得Scope | 部分取得、NOT EXISTS、Top-N、JOIN撤回 |
| Materialization | request-local memo、Memory/Local、分散Cache、Snapshot、Edge配信 | 世代・失効・Stampede、Tenant、Grant、Storage/egress |
| Target / UI / Delivery | Pool、Backpressure、Stream、仮想化、SSR/CDN | p99、追加Hop、初期描画・データの開示 |

分散Cache、独立常駐Agent、自動Adaptive制御はP0の前提にしない。最適化のためだけに新たなCache Host、第二Query Planner、第二Resolver、第二認可基盤を設けない。

## 10. Typed Lifecycle Graphを最適化の入力にする

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

## 11. Pluginの実装方式と安全なAdaptive Strategy

### 11.1 三段階の選定

1. 標準Function/Rule/Target-nativeで要件を満たせるかを先に検討する。
2. 不足する場合だけ既存#951 Contract/Provider/CompositionへObserver、Analyzer、Proposer、Strategy、Provider、Verifier等の**役割Binding**を追加する。これらをCore固定enumや独立Resolverへ昇格させない。
3. Go単一バイナリに適した静的合成・コード生成を既定候補にし、既存APM/DB/CacheへのAdapterを許す。独立プロセス・動的ロードは更新・隔離・実測利得が必要な案件に限る。

#3278原子的Traitは、Requiredなら実装義務とConformanceを要求し、Advisoryはヒントとしてのみ扱う。Cacheable等の名称だけでは保証にならない。Capability自己宣言、実現、実Target Evidence、現在の実行権限を区別する。Plugin Bundle配布と選択した構成、実Build依存の閉包を分離し、未選択Pluginが結果・ビルド・Cache Keyへ混入しないことを確認する。

### 11.2 安全な最適化の競合とFallback

- **同時Missの合流**：Request Coalescingは純粋なReadでQuery/parameter、Source revision、Context epoch、鮮度条件、Tenant/Disclosure scopeが両立する単位に限定。異なる現在権限や不確定な結果を一つのFutureへ束ねない。書込みの重複防止はCacheでなく既存Operation ID/Effect契約が所有する。
- **旧値復活**：Aが旧版を読みBが更新・失効後、Aが旧版を書き戻す競合では、Sourceの世代・条件付き書込み等が有効な場合にだけ防止保証を主張。Provider能力が未知なら強い鮮度を表明せず、条件に応じ再取得・利用不可・unknownへ戻す。
- **キーと資格の分離**：Key hitでもInput Closure、Rule/Operator/Toolchain/Schema、Snapshot Scope/Completeness、現在Grantを再照合する。Cache evictionは業務確定保存値を上書き・削除する理由にならない。
- **Query意味保存**：Native SQL/Pushdown/Batchの同等性はNULL/missing/非開示、集合粒度、時点、Collation、精度、ORDER、部分取得と非単調なRetractまで検証する。
- **Adaptive**：P0/P1は自由な本番自動書換えを行わず、事前に適合性を検証したStrategyだけを切替候補とする。後続の適応制御ではヒステリシス、最小保持時間、切替数上限、欠測/Exporter障害時の安全なFallback、変更履歴とRollbackを検査する。採用前の提案はSource・業務意味・権限を変更しない。

## 12. 検証に追加する反例（AT-15〜24、すべてNOT RUN）

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

## 13. 段階的な統合先・停止条件

| 段階 | 具体的な縦断 | 既存の正本と完了条件 |
| --- | --- | --- |
| T0：Fit/Gap | 実CLI/Query/Design Context/Provider/Telemetry/Scenarioの現行能力を照合 | #951/#3262/#2387、存在しないCLIを標準化せずunknownを保持 |
| T1：Read-only | 既存order_entryの実ReadでFunction→Query/SQL→DB Provider→Context ResultとSpan/Metricを同一Attemptへ結合 | #2586/#3270/#3262。実Providerと結果Scope/欠測・権限を照合 |
| T2：異質な2方式 | N+1除去またはSQL Pushdown、Cacheなし→Memory/Localの比較 | #2582/#3271/#1180。独立Expected、cold/warm、同時負荷、p99/費用・更新競合 |
| T3：AI Candidate | Design Context read-only投影→説明可能な改善候補→Plan/独立Benchmark→採否・事後観測 | #3262/#2516/#1180。AI推定は未受理、DDL/Effectは既存Grant/Receipt |
| T4：複数実Consumer | Go、Web/Desktop/完全Headless、後続Java/.NETのうち実際に実行したものだけ比較 | #1121/#3503/#3509/#3267。別TargetのPASSを流用しない |

#2600は実消費入力・Artifact再利用・Build省略・Evidence再資格化の所有先、#3271はCache/SnapshotとBackend、#2582はContext依存、#3278はTraitの実装義務、#3503/#1124は型付きFlow・図表とGeneratorの所有先とし、本書は**横断の最適化判定と受渡し**だけを所有する。#1119の既存G0〜G4へ追加Hard Gateを設けない。プロセスやPluginを増やしたのに実際の価値・正しさ・待ち時間改善を示せない場合は、その抽象を縮小/撤回する。

**現況**：これは設計と計画の成熟化。AT-15〜24、T1〜T4、実OTel/実DB、AI実適用、実Target性能改善と正式Owner受理は**NOT RUN / NOT ACCEPTED**。既存の限定Reference PASSや過去のGate B/Cを新性能の合格証拠と読み替えない。
