# 高度B2C向けHeadless・Conversion Engineering設計と受入計画（2026-11-08）

- **状態**：設計原則と段階的実証計画。実B2Cサイト、Experiment/Commerce Provider、性能計測、CVR改善、Owner意味受理は **NOT RUN / NOT ACCEPTED**。実装済みの機能を示す文書ではない。
- **親課題**：[#3509](https://github.com/hat47x/tei-platform/issues/3509)。既存UI Composition Kit [#997](https://github.com/hat47x/tei-platform/issues/997)、共通Operation [#1158](https://github.com/hat47x/tei-platform/issues/1158)、Interaction Flow [#3503](https://github.com/hat47x/tei-platform/issues/3503)、原子的Trait [#3278](https://github.com/hat47x/tei-platform/issues/3278)、Plugin解決 [#951](https://github.com/hat47x/tei-platform/issues/951)、生成拡張 [#2090](https://github.com/hat47x/tei-platform/issues/2090)、Effect [#1174](https://github.com/hat47x/tei-platform/issues/1174)、Scenario [#1180](https://github.com/hat47x/tei-platform/issues/1180)、Cache [#3271](https://github.com/hat47x/tei-platform/issues/3271) の責務を変更しない。
- **優先順位**：既存[#1119](https://github.com/hat47x/tei-platform/issues/1119)のP0/E2、実Provider D1bと現在のGateを置換しない。B2C独自の第二Host/Workflow/Resolver/Registry/CMS/決済エンジンを新設せず、既存契約の**異質Consumer試験**として進める。
- **検証の単位**：実際のSource・Generator・選択Provider・Target・Git Revision・Evidenceに束縛する。設計候補と実行済みEvidenceを混同しない。

## 1. 製品方針：高度B2Cで「TEIを使わない箇所」を正規に認める

TEIの標準データ構造・Function・標準表現型・Mapping・Rendererは別概念とする。標準表現型の定義は業務意味を持たず、そのインスタンスはFunctionに配置してAction/Data Contextへ束縛する。これが標準UIによる省力化の土台となる。一方、独自ブランド表現、購買導線、販促、頻繁なUX実験では、標準表現型への強制的なマッピングをしないことが設計上の保証となる。

**不変条件**
1. **完全Headlessを正規利用形態とする**。TEI標準UIを一切使わない独自Frontendでも、型付きData/Action/Failure/Effect/Source traceを任意に活用可能にする。
2. **分離の単位をRoute / Page / Section / Component / Operationにする**。画面の一部だけ標準UI、一部は独自UI、すべて独自UIのどれも妥当。採否はFramework/RendererではなくConsumer/Target条件に応じて決める。
3. **TEIがDOM/CSS/Router/State manager/SSR/Streaming/Hydration/CDN/Edgeを所有しない**。Target固有の実装を第一級とし、Browserに万能TEI Runtimeを強制しない。逆に、既存の標準Rendererも廃止しない。
4. **脱着可能**。UI Kit、計測Plugin、生成Adapterを外しても、独自Frontendと明示的に選択したBackend/Contractの実行が継続する。静的な型生成・検証だけでTEIの価値が成立してよい。
5. **性能上の選択可能性**。Script追加/Server hop/Plugin init/DOM wrapperが不要なUse-siteへ常時介在させない。「TEI導入」自体で初期JSや通信が必ず増える設計は不適合とする。

重要なのは「すべての表現をTEIで描く」ことではなく「独自実装であっても、意味・検証・変更の証拠へ接続できる」ことである。

### 1.1 Surface・Meaning・Executionは別の責務

| 層 | 基本的な責任者 | 変更してよいもの | 上書きしてはいけないもの |
| --- | --- | --- | --- |
| Business Meaning / Data / Operation | 既存TEI Meaning・FunctionおよびOwner | 受理済みRevisionの明示的改訂 | 表示の変更だけで業務意味/計算/確定条件を改変しない |
| Operation Contract / Mapping | #1158, #2506, #2507, #951 | 型付きAdapter、Target別Port | 同じ名前だけで意味・型・NULL・時点・Scopeを同一視しない |
| Interaction Flow | #3503 | 入力、提示、確認、遷移、タイムアウト、再開 | Flow確認をGrantや外部副作用の成功へ昇格しない |
| Presentation / Brand UX | #997と案件Frontend | DOM、CSS、アニメーション、Routing、Native UI、SSR方式 | Clientの成功表示を実注文確定にしない |
| Release / Experiment / Campaign | 案件・外部Provider（Pluginで接続） | 施策・対象・割当・版・露出・迅速撤回 | 価格/在庫/決済/認可の権威ある判定 |
| Effect / Receipt / Independent Read | 既存Host/Provider #1174, #1129 | Provider内の実現 | unknown commitを成功または失敗と断定しない |
| Optimization / Cache / Metrics | #3271, #2600, Plugin実現計画 | 資格確認済みの配置・キャッシュ・計測・候補比較 | Tenant/Consent/現在Grantを性能のために削らない |

## 2. 図表をGeneratorとConformanceの入力にする

個別のフロー図をドキュメントだけに閉じ込めず、**既存標準データ構造の型付きGraph + Role + Reference**による入力として比較する。Graph新Coreや新Workflow Engineを追加しない。固定部はRequirement/Traitで、可変部はExtension Binding/Presentationで宣言する。

~~~mermaid
flowchart TD
  A["施策・顧客体験の要求\n(Source / Revision)"] --> B["Journey候補Graph\n(ページ・露出・Event・Operation)"]
  B --> C["Ownerの意味・要求レビュー\n(Proposal vs Accepted)"]
  C --> D["既存Contract / Provider Resolver\n(適格性・exact pin)"]
  D --> E["Lifecycle Extension合成\n(Trait / Slot / Implementation / Pipeline)"]
  E --> F["Effective Graph検査\n(Required invariant / Gap / Conflict)"]
  F --> G["Target別生成またはNative接続\n(Web / Edge / Server / CLI)"]
  G --> H["Scenario / Browser / Provider Conformance"]
  H --> I["公開と変更の証拠\n(Release / Variant / Exposure)"]
  I --> J["独立の効果観測\n(Action / Order / Receipt / Analytics)"]
  J --> K["評価・修正・採用・撤回"]
  K -. "新Revision" .-> A
~~~

### 2.1 Graphの最低限の識別

- ノード: Journey、Touchpoint、Interaction Event、Offer、Action Intent、Operation、Host/Provider Effect、Experiment Assignment、Exposure、Outcome、Guard、Fail/Unknown、Release。
- エッジ: 表示/導線、Event候補→確定、Action Intent→Operation、Operation→Effect/Receipt、Exposure→観測対象、依存/失効、Transition、Extension Slot。
- すべてに同一名称ではなく **Identity + Source Revision + Scope + Target** の対応を要求。表示名と業務ID、DOM要素ID、外部計測Event IDは区別。
- Flow表示の追加・削除と、業務状態遷移の変更は別のProposal。Flowで「購入完了」と描いたこと自体は、実決済の証拠にならない。
- Pluginで拡張するのは、公開されたSlotと型付きLifecycleであって、既存の不変操作を置換して回避する抜け道ではない。
- AI会話や音声など他媒体にも適用できるよう、媒体固有Action/Promptと共通Operationを分離。Streaming部分出力・ASR候補・曖昧Intentは確定Event/Grantにならない。

### 2.2 Hook/Traitの配置

**Trait = 何を保証するか**、**Hook Slot = どこを拡張できるか**、**Implementation = 実際に何が履行するか**、**Pipeline = どの順序で呼ぶか**を分ける。公開位置、順序、前後条件、Target、Effect、Permissionを型付きContractへ接続する。原子的なRequired Traitは未充足時にFail Closedとし、Advisory Hintは明示的に区別する。HookでAudit・Grant再照合・購入確定時のValidationを短絡しない。

## 3. プラグイン製品候補と、二重基盤を避ける境界

### 3.1 第一候補：Conversion Engineering Kit（名称仮）

**「変更案 → Variant配分 → 実表示（Exposure）→ 行動 → 権威ある業務効果 → 評価 → 安全な採用/撤回」の対応を、型付きOperation/Effect・Provenance・Scenarioに接続して検査する一組のPlugin/Adapter。**

単なる新Analytics/AB SaaSを自作する提案ではない。すでに採用したAnalytics・Flag・CMS・CommerceサービスをConsumer/Provider Adapterとして利用する。独自性は **既存Operationから測定・変更の整合性を機械的に検査できること** に置く。

| 候補 | 提供する機能 | P0/P1 | 外部依存・越えない責任 |
| --- | --- | --- | --- |
| Feature/Experiment Adapter | Provider選択、Variant結果/理由/Revision、配分と配信、ロールバック | P0 | OpenFeature等の標準APIを第一候補。Flag制御面をTEIへ再実装しない |
| Journey/Conversion Inspector | Journeyの欠落Action、露出欠測、重複CV、誤った業務確定イベントの診断 | P0 | 実売上・因果効果は外部Analytics/実験Evidenceで判定 |
| Change Impact & Conformance | UI/SEO/a11y/SSR-Hydration/Cache/Provider Effect差分、再生成保持・Rollbackテスト | P0 | 実Chromium/Targetと独立Expectedを使用 |
| Campaign/CMS/PIM Bridge | 素材・商品情報・公開版・承認と運用差分の接続 | P1 | KON Dataspaceは任意。既存CMS/PIM等と排他にしない |
| Search/Personalization Bridge | 推薦・検索順位・オファーのProvider接続 | P1 | 推定値と確定価格/在庫/許可を区別 |
| Commerce Effect Bridge | Cart/Checkout/Order/PaymentのOperation/Receipt対応 | P1 | 実決済の実装・金銭的確定は外部Commerce/Paymentが権威を持つ |

### 3.2 顧客体験と業務効果を切り離した計測モデル

説明用の観測形（新Canonical Event schemaではない）：

| 観測 | 最小束縛 | 何の証拠か | 証明できないこと |
| --- | --- | --- | --- |
| Assignment | experiment ID/rev、unit kind/key、rule/decision rev、target scope、time | その時点でのVariant決定 | 実際に見たこと |
| Exposure | 対象Unit、Variant、Touchpoint、描画/表示事実、consent・delivery結果 | 特定の体験に接触した観測 | 注文等の業務確定 |
| Intent | Action ID、対象、selection epoch、input rev、client time | 実行要求を発したこと | Host Grant/Provider成功 |
| Operation Result | Op identity、typed result/failure、Provider ref/rev、server time | 選択した実装の応答 | 独立DB状態、決済成立 |
| Effect Observation | Receipt/operation correlation、実Provider authoritative state、別Read | 限定Scopeで観測した業務効果 | 実験の因果的有効性 |
| Experiment Assessment | 母集団、ランダム化単位、実配分、欠測、guardrail、統計法・期間 | 実験対象に限定した解析 | 観測の無い母集団、永久的効果 |

- **Assignment ≠ Exposure ≠ Click ≠ Intent ≠ Authorized Invoke ≠ Provider Effect ≠ Revenue Uplift** を守る。クリックイベントでCVRを代理する場合、その指標を明示。
- 配分比の異常（SRM）、重複、計測遅延、離脱/観測欠損、クロスデバイス/ログイン遷移、匿名と既知Userの結合前提を診断対象とする。Cookie/匿名IDの運用は法的・同意条件に従い、TEI固有の恒久Tracking IDを強制しない。
- 利用者/セッション/端末のどれをランダム化単位にするか、同じUnitに同じVariantを継続する要否、算出指標と評価期間は施策ごとの設計入力。分析結果を自動的なOwner承認に読み替えない。
- Consent未取得、撤回、目的外、開示拒否の状態では計測外部送出を許可しない。認可スコープはRuntimeで確認。匿名化・保持/削除期間もProvider責務と接続する。

## 4. 高頻度変更に耐えるRelease/Build/Cacheの資格

B2CではCreativeとキャンペーンの変更が頻繁であるため、すべてを業務Meaningや全Artifactへ波及させると保守性能を損なう。変更の種類ごとに **失効を最小化しつつ、実際に影響する範囲を漏らさない**。

| 変更 | 再評価する対象 | 変更しないことの検査 |
| --- | --- | --- |
| バナー・文言・画像 | CMS/Asset revision、CDN/HTML/SEO、計測Tag、必要なUI回帰 | 業務Meaning/Commerce priceの自動改訂禁止 |
| CSS/独自Frontend | JS/CSS artifact、Hydration/SSR互換、a11y/CWV、Interaction契約 | 既存Contract/OperationやUI拡張の破壊を診断 |
| Experiment flag / allocation | decision revision、SSR/Client/EdgeのVariant一貫性、Exposure、キャッシュ区分 | 過去版の露出・割当の上書き禁止 |
| 価格・クーポン・在庫 | authority source、計算・時点・Scope、対象表示Cache、Checkout再検証 | stale price/discountを実注文へ確定させない |
| Application Operation / Provider | typed contract、Plugin exact BOM、実呼出し・Scope、独立Expected/Effect | 表示経路で旧実装へ無音fallbackしない |
| Generator/Target | Toolchain、Source dependency closure、Artifact revision、拡張適合・Target試験 | 名前・同じSource byteだけで旧Build Evidenceを再利用しない |

### 4.1 B2CのCache安全条件

- Public HTML / CDN、Experiment-variant HTML、User/Session/Tenant private、Catalog/Inventory/Price、Cart/Checkout、Build Artifactの **所有主体・鮮度・撤回条件を混同しない**。
- Cache Keyまたは資格検査には必要に応じ **Provider ID/version、Source revision/fingerprint、Target、Locale、Currency、Price/Promotion context、Experiment assignment、Scope、Consent/Disclosure、effective time** を含める。すべてを機械的にKeyへ詰め込むのでなく、**漏えいしないことを示す分割・再検証**を要求する。
- 同一URL/Routeでも実験Variantやセッションに応じて表示が違う場合、public cacheの安全な共有範囲を限定。個人化HTMLのpublic cache混入・Variant混線はBlock。
- TTLの満了、Cache hit、ETagが一致したことは **現在のGrant/価格/在庫/注文確定の証拠ではない**。購入時は権威のあるBackendで再照合する。
- DerivedDataの再計算資格、Source更新、A→B→A、部分取得、Stale/Unavailable/Permission deniedは既存[#3271](https://github.com/hat47x/tei-platform/issues/3271)の区別を再利用。TEI独自の第二Cache層を必須化しない。

## 5. AIによる最適化を支える観測・判断の足場

**無条件に自律チューニングする機構より先に、理由を説明できる計測面を設計する。** 既存のPlugin実現・最適化要求（[plan/design/PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md](PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md)）へ接続し、Cost-basedの第二Resolverは作らない。

最低計測票：

| 範囲 | 指標 | 版・取得条件 |
| --- | --- | --- |
| Browser実測 | LCP/INP/CLS p75、long task、script transfer/parse、Hydration、Route変更 | 実デバイス/国・回線/ページ・variant、測定窓、同意 |
| Backend/Provider | TTFB/latency p50/p95/p99、Query/Cache hit/miss、DB read/write、Error/Timeout/unknown | Operation/Provider/Target/Revision、負荷・同時実行数 |
| Journey/Experiment | assignment→exposure欠落、露出重複、sample ratio mismatch、step別離脱 | 実験Unit、対象母集団、Consent、期間、集計母数 |
| Build/Delivery | cold/warm build、生成出力byte、first-party JS、Plugin追加byte、CDN/Edge hit、再生成差分 | selected BOM、Generator/Target/toolchain/source fingerprint |
| Economics | 改修リードタイム、配信リードタイム、障害発見時間、Rollback時間、運用コスト | ベースライン・変更内容・対象Consumer・担当者/AI条件 |

AIに渡すのは **Measure + Evidence + Constraint + Counterexample + Candidate + Expected Impact** の候補セット。AIは「既に観測した事実」と「改善仮説」を区別し、次のように振舞う。

1. 独立Expected、Contract/Effect、現在Grant、Private dataの漏えい防止を先に確認する。
2. Flame graph、Span、Resource timing、Cost、Cache missの証拠を根拠にボトルネック候補を提示する。
3. 選択済みProviderの適格な実現方式（Client/Edge/Server/Build-time）と、追加Script・通信の負担を比較する。
4. 最小のChange Proposalを作り、影響Graphから再検証シナリオを選択する。推測した改善値を「実測PASS」にしない。
5. 正しさPASSと性能PASS、開発生産性PASSとCVR因果効果PASSは**それぞれ別票**にする。実験の停止/採用判断は担当Ownerと外部運用の権限に従う。

最適化候補を勝手に新Providerへ切替えたり、測定に都合のよいユーザーだけを除外したり、CacheのPrivate境界やConsentを緩めたりしない。

## 6. SEO・アクセシビリティ・実ブラウザでの不変条件

- SEOでは、実験のためにGooglebotと人間で意図的に異なる内容を提供するクローキングをしない。実験URLが分かれる場合はcanonical/一時Redirect等の原則を検査。Content/Metadata/Structured Data/robotsの差分も記録する。参照: [Google Search Central - A/B Testing](https://developers.google.com/search/docs/crawling-indexing/website-testing)。
- SSRで決めたVariantとHydration/Client/Edgeで表示するVariantの整合性、公開切替のAtomic性をテストする。強制的な巨大Client SDK/antiflicker Scriptを前提にしない。
- A11y、Keyboard、Focus、Screen reader、モバイルTouch/IME、二重Submit、通信断復旧を標準UIとNative UIの双方で検査する。
- Web Vitals良好範囲（p75 LCP≤2.5s、INP≤200ms、CLS≤0.1）は外部目安。TEI導入差分として測るが、達成済みを意味しない。参照: [Google Search Central - Core Web Vitals](https://developers.google.com/search/docs/appearance/core-web-vitals)。
- OpenFeature [Evaluation](https://openfeature.dev/specification/sections/flag-evaluation/)・[Tracking](https://openfeature.dev/specification/sections/tracking/) のAdapter適合を最初に検討。特定Vendorの制御面、Runtime、Providerの版まで一括して導入することは前提にしない。

## 7. 最初のReference実装と受入シナリオ

**Reference Consumer**：商品詳細(PDP) → カート追加 → Checkout Sandbox → 注文状態の別Read。完全Native/Headless版（H0）と、同一Function内の一部を標準Rendererへ差し替えた混在版（H1）を比較。さらに同じOperationをCLI（H2）から実行し、媒体間の意味差分を検査する。

この例は**検証Fixture**であって、本番決済や法令対応済み商用ECの提供を意味しない。独立Expectedはテストより先に版固定する。各CaseのStep / chosen Provider / loaded / invoked / received / Effect / Target artifactを証拠として記録する。

| Case | 故障・変更 | 固定する独立Expected |
| --- | --- | --- |
| BC-01 | Native PDPで標準TEI UIなし | 型付きOperationを呼び、TEIによるDOM/Router依存と追加Scriptが必須でない |
| BC-02 | 一Sectionを標準Rendererに切替え、戻す | Intent/Subject/Validation/Authority/Result不変。Native専有の自由度を維持 |
| BC-03 | SSR/Client/EdgeのVariant不整合 | 配信・Exposure/Assessmentを不整合として診断。誤った同一実験の集計を拒否 |
| BC-04 | 価格・在庫・Promo改訂中、旧CDN/Cache応答 | Priceは権威Serverで再検証、旧表示を注文確定に使わない |
| BC-05 | Cart成功UI後、Provider拒否/Timeout/Commit unknown | 成功UI・応答・実Effectを別状態で保持。重複送信を防ぐ |
| BC-06 | 文言だけ→Frontendだけ→Meaning/Providerの順に改訂 | 意味受理・Artifact・Cache・Targetテストが変更影響に応じて失効 |
| BC-07 | Analytics script遅延/未同意/重複イベント | 未同意の外部送出0、計測不明は不明、Core UXを不要にブロックしない |
| BC-08 | Variantロールバック/Frontend切替 | Traceの実験・Source・Artifact旧版を保持。更新差分と復旧可能性を観測 |
| BC-09 | canonical/metadata/キーボード/IME/二重Submit | 実ブラウザで回帰を検知。未実施のTargetをPASS扱いしない |
| BC-10 | UI/計測Pluginを除去 | Native Frontendと選択したBackend Contractが独立して成立 |
| BC-11 | User/Tenant/ExperimentのCache key混線 | 他Scopeのprivate HTML/price/consentを一切配信しない |
| BC-12 | 低品質な試験（SRM・欠測・Variant偏り） | 売上優位やCVR改善を自動採用しない。診断/未確定を維持 |
| BC-13 | Trait required未充足、Hook競合、B選択→A実Call | Build/Activation/Invocation適合を拒否。宣言と実効果を区別 |
| BC-14 | Chrome fieldで悪化・単一labで改善 | 実測母集団・期間・環境を区別し、性能改善を無条件認定しない |
| BC-15 | AIがCache/Provider/Script移動を提案 | 許可されたScope/Guard/Effectを保持。独立Expected差分とRollbackを提示 |

**証拠の資格**：表/Graph/DSLの構文PASS、Contract静的適合、Stub/合成Provider、実Browser、実選択Provider、外部Sandbox Effect、独立Read、Field統計、Owner Acceptanceは別の欄に持つ。空結果/部分取得/unknown EffectはPASSへ折り畳まない。

## 8. 実施順と既存Ownerへの還元

| Stage | 作業 | Exit（不足時は止めず「NOT RUN/Gap」を残す） | 既存Owner |
| --- | --- | --- | --- |
| B0: Fit/Gap | #997/#3503/#3278/#951/#3271の実装・採用済ADRとNative B2Cの相違を列挙。Source/Graph/要求・版を固定 | 新Core無しで最低1 Journeyと1 Operationの参照関係が表現可能か | #3509, #997, #3503 |
| B1: Headless/E2E | H0商品詳細→Cart→Sandbox→別Read。H2 CLIと同Operationを比較 | Native Frontend、Provider選択→実Call、Effect、unknownを分離 | #1158, #951, #1180, #1174 |
| B2: 可変UI・Experiment | H1で一部標準UI、OpenFeature等1 Provider、SSR/Client変種、Rollback、Cookie/ConsentとCache負例 | 意味不変、実Variant/Exposure、a11y/SEO/性能、差し戻しが実Targetで観測 | #997, #3278, #3271 |
| B3: AI最適化/収益Kit評価 | 計測データから複数候補、正/負比較、変更速度と測定品質を評価 | Dev Time/故障検出/性能/Costは差分観測。CVR upliftは適正実験のみ | #2600, #951, #1180, #3509 |
| B4: 製品化判断 | B2C固有Pluginを足すか、既存Kit/Provider Adapterに留めるか比較 | 保守面積、導入負担、独立Consumer、撤回費用を示しOwner判断 | #3509→既存Owner |

### 最小受入票（各Stageで記録する）

- Fixed Scenario ID / Independent Expected hash / Source Revision / Accepted Meaning scope。
- Headless/標準Rendererの利用範囲、Framework/Target、実Provider selection/binding/version、実loaded/invoked/received。
- Operation/Effect/Receipt/independent read、Uncertainty、Grant、Consent、Cache qualificationとVersion matrix/BOM。
- 比較条件（Device/Region/Browser/traffic/warm-cold）とベースライン、回帰発生/検出、JS・Latency/CWV、運用変更の所要時間。
- 不変条件・未実証・Owner未受理、Rollbackの実行証拠、採用/保留/棄却理由。

## 9. 判断の保留事項

- Commerce/Experimentの外部Provider実選定、実験の匿名化と保持方針、KONのコンテンツ永続化採否。
- CDN/Edge/Streaming Framework固有のPluginをいくつ標準配布するか。最初は既存Framework一つ・Provider一つだけで実証する。
- 統計解析・レコメンドモデルをTEIの製品責務に含めるか。初期段階は**外部Analytics/ExperimentへのAdapterと契約検証**に限定する。
- 運用上のCVRや売上の向上は未検証。非導入との開発負担・正しさ・性能比較が成立するまでは「キラー機能」と正式認定しない。

**結論**：高度B2Cへの対応を「標準UIの多機能化」と同一視しない。TEIを採用しながらUIの全自由度を維持できることを第一に保証する。そのうえで、TEI独自の意味・操作・Graph・Evidenceを活用して、施策変更と実業務効果を安全かつ迅速に検証できるPlugin群を、実Consumerで選抜する。


<!-- b2c-cross-chat-maturation-20261108 -->
## 10. 横断成果の反映：Experienceの自由と標準Model Mapを両立する

本章は、2026-11-08の全体Model Map/RQ-036、Lifecycle Graph→生成実装、原子的Trait/Hook、AI向けCLI/共通Operation、KON/SHO分担、Cache・性能最適化を、高度B2Cの異質Consumerへ適用した追加設計である。既存の意味正本・Resolver・Host・Graph Writer・Harnessを再構築せず、実装済みの証拠も再発行しない。

### 10.1 Experience Intentは業務意味の正本ではない

ブランド、訴求、導線、体験実験の自由度を保つため、顧客体験の意図を既存の標準Record/Graph/Roleへ **Experience Proposal/Projection** として束縛する可能性を検証する。新Canonical Experience型の採用宣言ではない。

| 視点 | 具体例 | 正本の権威 | 変更した場合 |
| --- | --- | --- | --- |
| Business Meaning | 商品・契約・価格計算・在庫確定・注文状態 | 既存OwnerのAccepted Meaning/Host/Commerce Provider | 変更案とOwner受理・実Effect検証 |
| Experience Intent | 誰に何をどの順序で提示し、何を検証するか | 案件UX/Brand/Experiment Source | 実験・配信の版固定、関連Operationの適合検査 |
| Presentation | 独自Frontend/DOM/CSS/Asset/SSR | 案件コードとFramework/Target | Browser/SEO/a11y/性能・Renderer/Hook適合 |
| Observation | Variant露出、行動、Provider Effect、実測値 | 計測元・外部Backendの限定観測 | 欠測/旧版を保持し意味と権限へ昇格しない |

- 文言・デザインをTEI標準表現型へ強制変換しない。表示の自由と型付きAction契約への接続を両立する。
- 価格・契約条件・同意に関係する表示は「単なるUX」と偽装して業務・法務受理を省略しない。
- 同じBusiness Meaningでも、Flow-oriented / Balanced / Rich/Case-oriented等の異なる体験を採用できる。速度や省操作数だけを普遍的なUX正解としない。

### 10.2 全体MapにB2C導線の位置づけを示す

標準の全体図・固定/可変・Source navigationは[#3505](https://github.com/hat47x/tei-platform/pull/3505)／RQ-036／[三要素図表共通契約](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)へ委譲する。B2C専用の第二Diagram正本を設けない。

~~~text
Journey / Campaign / Experiment (案件Experience投影)
  ├─ Touchpoint / Variant / Exposure → 計測Eventと対象母集団
  ├─ UI Event / Confirm              → Interaction Flow (#3503)
  ├─ Action Intent                   → Typed Operation (#1158)
  ├─ Business Outcome                → Provider Effect / Receipt (#1174)
  ├─ Data Dependency                 → Data/Function/Business / Cache (#3271)
  ├─ Hook / Plugin                   → Trait / Effective Graph (#3278)
  └─ Source / Target / Release       → Generator / Artifact / Conformance
~~~

「全体」は**World/Source/Scope/Consumer相対**である。required、resolved、unresolved、unavailable、permission-denied、out-of-scope（説明用分類）の対象数と理由を持ち、未取得Analyticsや未接続Commerce APIを「イベントなし」と断定しない。Layout変更とEdge/Guard変更を区別し、合成前・Plugin Patch・実効Graphを追跡する。表示Label・DOM ID・外部Event名の一致だけでBusiness Identityや実Effectを結合しない。

## 11. 高頻度変更のための「変更レーン」

すべての販促変更に完全な業務再受理を強制すれば変更速度を損なう。一方、すべてを軽い承認経路に通せば注文・契約・顧客情報を危険にさらす。既存Source/Impact/Acceptance/Build/Release契約上で、**変更の実質的な影響**に応じた検査集合を合成する。新Approval Engineは作らない。

| Lane（評価用） | 例 | 主な再検証 | 変更責任 |
| --- | --- | --- | --- |
| C0 Content | 通常のバナー・画像・文言 | Asset/公開Revision、CDN Cache、SEO・a11y | Content Owner |
| C1 Presentation | CSS、配置、アニメ、UI Kit差替え | Frontend Artifact、Browser/CWV、Actionの結合 | UX/Frontend Owner |
| C2 Interaction | 入力順、確認/申込導線、Event Binding | Flow/Guard/Operation・同意・負例Scenario | Interaction Owner |
| C3 Business/Effect | 価格・割引条件・在庫・決済・契約条件 | Meaning、現在Grant、権威Provider、Effect/別Read | 業務Owner/Host |
| C4 Provider/Release | CDN、Edge、Experiment/Commerce Provider更新 | exact BOM、実Load/Call、Version、Cache失効・Rollback | Platform/Release Owner |

- 一つの変更が複数Laneにまたがる場合、必要な義務を合成する。変更の名称でC3の権限や保証を迂回しない。
- 法的表示や規約を変更する場合、表面上C0であっても必要な受理へ昇格させる。
- 計測タグ追加は純粋な文言変更ではなく、Consent/CSP/外部送信/サプライチェーン/性能の再検査を要求する。
- Experiment停止・Frontend Rollbackは **新規公開/Variant配信の停止** であり、既に確定した注文の取消ではない。既存Effectの補償は別の権限付きOperation/Receiptに従う。
- 生成AIは変更差分からLane・依存・期待する再試験を提案できるが、Ownerの承認と実Provider適格性を代行しない。

### 11.1 B2CをEC一例へ閉じない

| 異質Consumer | 成功と誤りやすいUIイベント | 確定の権威 | 圧力条件 |
| --- | --- | --- | --- |
| EC | 購入ボタン、決済ページ到達 | Order/Payment Providerの最終状態 | 値引き/在庫/返品/二重Effect |
| 予約・席/枠販売 | 入力送信・仮押さえ表示 | 予約Providerの確定枠 | 同時予約、期限、キャンセル、時間帯 |
| 会員・Subscription | 会員登録・無料Trial開始 | 契約/課金/会員権限の実状態 | Trial→課金→解約、期間跨ぎの継続率 |

B1はEC一縦切りで十分だが、B4のKit採用前に**EC以外の少なくとも一Consumerで再利用できる範囲・Gap**を評価する。全Consumerの実Target実装は新P0 Hard Dependencyにしない。

## 12. コンバージョンの観測と事業成果の評価を分ける

- **計画時**：実験ID/Revision、ランダム化単位、対象母集団、primary metric、guardrail、評価期間、停止条件、複数施策への対応を固定する。
- **計測時**：Assignment≠Exposure≠Click≠Intent≠Operation Result≠Effect≠Revenue Uplift。SRM、重複、欠測、Event time/Processing time、同意、クロスデバイス/ログイン移行の結合前提を診断する。
- **効果評価**：CVRだけでなく、粗利益、返品/取消、継続率、取得単価、ユーザー体験の悪化をGuardrail候補として扱う。計測Providerから取得していない指標や定義不明の金額をTEIが創作しない。
- **因果推論**：適正なランダム化、評価母集団、測定窓、欠測・偏りの検査を持つ実験と、単なる相関観測を区別する。サブグループ後付けや途中結果の恣意的採用によって因果効果を断定しない。
- **承認**：Experiment AssessmentもAI提案もOwnerの最終公開権限・Host Grantを生まない。導入効果は実測から初めて判定する。

Performance/Conversion/Checkout/ConsentのGuardrail閾値は案件別のRequired/Advisory契約とし、TEI Coreに普遍的な一律CVR・売上閾値を固定しない。複数の指標が競合した場合、その判断は業務/プロダクトOwnerへ残す。

## 13. AI Harness、テスト、KON/SHOの分担

B2Cで有用なのは、特定のHarnessの常時搭載ではなく、**既存契約とEvidenceを利用して変更案を安全に評価できる足場**である。DSH型のEverything-is-Plugin方式や外部AI Harnessでも、TEI Runtimeに第二Resolver/Host/Source Writer/Test Runnerを導入しない。

~~~text
任意のAI/Harness
  Inspect（Model Map / Source / Performance / Provider Evidence）
    → Propose（UX/Experiment/Cache/Plugin変更案）
    → Generate（案件Native Frontend / Adapter / Scenario候補）
    → Check（独立Expected / 実Browser / 実Provider / CI）
    → Compare（正しさ・性能・計測品質・変更コストを別票）
    → Review & Apply（既存Owner/Hostが許可したReleaseのみ）
           ↓
既存TEI Contract / Operation / Source / Evidence
~~~

- テストはGUI/Browserスクリーンショットだけでなく、構造化CLI/CUI/HTTP/実Providerの**同Scenario・独立Expected**を利用する。見た目の回帰をBusiness Effectの証拠へ昇格しない。
- Plugin・外部Analytics Script・Web取得テキスト・AI生成コードは信頼済み入力ではない。実際の外部送出、Consent/Privacy、必要なCapability、Provider実Callを独立検証する。
- **KON Dataspace**は公開版・永続ID・素材・来歴・権限/公開境界を担う任意候補。TEIのAccepted Meaning、Query/Transaction、Host Grant、第二Plugin Registryにはしない。既存CMS/PIM/Commerceを優先利用してよい。
- **SHO Composer**は自然言語主体の販促テキスト/コンテンツ編集とSource provenanceを担えるが、商品/契約の確定規則を上書きしない。Web UI化する場合も全TEI StudioをB2Cへ常駐させず、共通Application APIとPublish Adapterを使用する。
- **EKI**等の分散転送を計測/通知へ用いる余地はあるがP0必須依存ではない。配信成功と実業務Effect成功を混同しない。

## 14. 追加の独立Expected候補と製品化判断

| Case | 故障・変更 | 固定するExpected | Stage |
| --- | --- | --- | --- |
| BC-16 | 通常バナー/文言だけ変更 | Meaning再受理を不要に保ちつつ配信版/SEO/a11y/Cacheは追跡 | B2 |
| BC-17 | 文言変更に規約/料金条件が混入 | C0扱いを拒否し必要なMeaning/Authorityへ差分を接続 | B2 |
| BC-18 | 図のLayout移動とEdge/Guard編集を混在 | Layoutは実装更新不要、意味Graph変更はSource Proposal/実効Graphへ | B0/B2 |
| BC-19 | Analytics/Commerce外部参照の未取得/Permission denied | 不足件数と理由を保持し、完全Journey/欠落ゼロと主張しない | B0/B1 |
| BC-20 | CVR上昇・粗利益や返品が悪化 | 単独CVRで自動採用せず評価窓/Guardrail/欠測を報告 | B3 |
| BC-21 | Trial登録を課金完了に誤計上 | 契約/課金/解約を異なるEffectと時間条件へ分離 | B4前のGap |
| BC-22 | AIがUI修正後、実Providerを見ずに改善と判定 | UI観測・Backend Effect・統計的因果を別資格に分ける | B2/B3 |
| BC-23 | Variant停止を既存注文のUndoと解釈 | 配信Rollbackと権限付き補償Operationを分離、二重取消を拒否 | B2 |
| BC-24 | Native Frontendを別Frameworkへ交換しTEI UI Plugin除去 | DOM/Router/Stateの自由を維持し、同Operationを利用できる | B1/B4 |

### B4の採否を撤回可能にする

| 評価軸 | 必要な比較と証拠 | 保留・棄却の例 |
| --- | --- | --- |
| 非拘束性 | 同じFrontendでTEI有/無のScript/通信/DOM/SSR差分、Plugin脱着 | 必須Runtime負担、Framework拘束、Headless移行不能 |
| 生産性 | UX変更→審査→公開→Rollbackの時間、生成AIの修正量 | 学習/保守/Plugin追従負担が利益を上回る |
| 計測品質 | 欠落/二重/不明Effect、SRM/Guardrail診断の実検出率 | Contractだけ存在し、実Consumerの誤りを検出できない |
| 安全性 | Checkout/Price/Consent/Cache/unknown commit負例 | 実Targetで権限/Scope/Effectの保証が崩れる |
| 既存ツールとの共存 | 外部Analytics/Flag/CMS/Commerceを置換せず接続 | 新たな中央Runtime/Registryが必須になる |
| 収益性 | 事前に固定した母集団・指標・評価期間・独立Baseline | 因果の根拠なし、後付け指標、欠測 |
| 再利用 | EC以外の予約/Subscriptionで共通契約を再利用できる部分 | 一用途専用ロジックを標準Coreへ昇格させる |

**優先順**：B0のFit/Gapはread-onlyで開始し、既存#1119のP0/E2およびD1b実Provider Callを先行する。B1の実Effectが未実証のままB2のBrowser/Experiment成功を注文確定の証拠にしない。ただしSEO/a11y/性能など独立検査は並行可。B2差分は#997/#3503/#3278/#3271、B3は#2600と既存性能設計、B4のPlugin採否は#951/#1016/#259へ還元する。

以上は**設計・独立Expected候補**。現行TEIの実製品機能、実Browser/Commerce/Experiment Provider適合、CVR向上、Owner Meaning受理は **NOT RUN / NOT ACCEPTED** とする。
