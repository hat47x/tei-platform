# 原子的Traitの実装義務・合成・履行検証（2026-11-07）

- **状態**：横断の設計・実証候補（未採用）。正式なCanonical Schema、Core Trait enum、Plugin API、CLIコマンド、生成器、Provider実装、実Target Conformanceが成立したとの主張ではない。
- **担当**：[#3278](https://github.com/hat47x/tei-platform/issues/3278)。既存[#951](https://github.com/hat47x/tei-platform/issues/951)のContract/Resolver、[#1016](https://github.com/hat47x/tei-platform/issues/1016)の構成選択、[#2495](https://github.com/hat47x/tei-platform/issues/2495)/[#2544](https://github.com/hat47x/tei-platform/issues/2544)の数学的Facet・Law、[#3271](https://github.com/hat47x/tei-platform/issues/3271)の派生データ/CacheはそれぞれのOwnerに留める。
- **関連**：[意味TraitのClaimと観測](SEMANTIC_TRAIT_CLAIMS_AND_CONFORMANCE_2026-11.md)、[Plugin実現計画](PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md)、[Temporalの意味](TEMPORAL_MEANING_DISTINCTION_2026-10.md)、[Derived値](DATA_DERIVATION_REFERENCE_CONTRACT.md)、[#3267](https://github.com/hat47x/tei-platform/issues/3267) BOM、[#2600](https://github.com/hat47x/tei-platform/issues/2600) build reuse、[#2090](https://github.com/hat47x/tei-platform/issues/2090) 手動実装拡張点。
- **非目標**：第二のSemantic正本・Trait Registry・Resolver・Host/認可・CAS/Store・Testing Runnerを作らない。提案だけでOwnerのMeaning受理、Build/Runtime適合、Provider副作用成功を主張しない。

## 1. 結論：Traitは小さな「要求契約」、実装手段とは独立

TEI Traitを、**対象に付与でき、独立して要求・実装・反例検証できる最小の意味的保証**として扱う。単なる属性ラベルではなく、実装の振舞い、必要操作、制約、診断、または検証義務のいずれかに具体的な差を生むことを採用条件とする。

- **Semantic facet / Law**：対象がどのような数学的・業務的性質を持つと主張するか。受理済みか、観測したかの区別は既存[#2495](https://github.com/hat47x/tei-platform/issues/2495)/[#2544](https://github.com/hat47x/tei-platform/issues/2544)へ委譲。
- **Trait obligation**：選択した対象/用途/Scopeに対して、必要な操作・前提・不変条件を何で充足するかを要求する。名称や型の一致だけで充足と見なさない。
- **Implementation / Provider**：自動生成、標準実装、手動実装、外部Host/Provider実装で同じ契約を満たし得る。**実装を強制することと、物理方式を固定することを分ける**。
- **Capability**：特定のProvider/Target/Instanceにおいて実際に使える操作。Traitの宣言・実装メソッドの存在・Providerの自己申告と、検証済み実Capabilityは別。
- **Hint**：最適化や実装選択への参考情報。標準ではTraitのrequired履行と混同しない。何に利用されたか、無視されたかを診断可能にする。

これらは**既存Standard Record/Relation/Typed ReferenceとAssessment/Conformanceの投影候補**とし、別の正本となる永続レコード群を先に採用しない。

### 原子性の試験

Traitは以下を独立に答えられる範囲へ切り分ける。(a) 付与対象と最小の責務、(b) required operations / invariants、(c) Scope・Failure・Effect、(d) 正・負の独立Expected、(e) 失効条件。例えば `ConditionalWrite` は条件付き更新と不一致時の非更新を一体として検査するため、一保証に複数の操作が含まれてもよい。

`Temporal` は初期段階では単一の巨大Traitとせず、`ValidTime`、`RecordedTime`、`HistoricalQuery`、`ChangeTracked` 等の独立責務を比較する。`DerivedData` は `Derivable`、`Rebuildable`、`Disposable`、`Materialized`、`DependencyTracked` の構成例として検討する。これらは**正式Trait IDの採用ではない**。

## 2. 宣言・解決・検証を分離する

```text
Accepted / Proposed Meaning + Source Location/Revision
      │
      ▼
Trait Binding (subject, trait, parameter, scope)
      │
      ▼
Obligation Set (required operation, guarantee, applicability)
      │
      ▼
Existing Contract/Provider Resolver (#951, #1016)
      │
      ├── Generated / Standard
      ├── Manual implementation (#2090)
      └── External Target / Provider
      │
      ▼
Conformance / Independent Expected (#1180, #2514)
      │
      ▼
Fulfillment Assessment (target, version, scope, evidence, unknown)
      │
      ▼
Build/Runtime/Host Gate (独立：#943, #1174, #2600)
```

**Binding** は対象Identity、Source occurrence/Revision、Trait定義Rev、Parameters、必要Scopeを持つ。**Fulfillment** は実Providerの解決済みID/Contract Fingerprint、選択したImplementation、Target/Build、検証Evidenceを持つ。**Assessment** は `required-missing`、`unsupported`、`conflicting`、`unverified`、`satisfied-under-scope` 等を区別する（診断名は候補）。不明を満たしたことにしない。

`required`：成立しなければ対象用途でのBuild/Activation/Operation Offerを拒否するか、利用可能性を未成立と示す。手動実装が必要な場合、必要なOperationとConformanceの不足を開発環境が表示する。`advisory`：構成提案へのヒント。未採用なら無効化理由を表示するが、Runtime保証を要求したとは主張しない。

**Trait付与は権限を生まない**。Host/Providerの現在のGrant、Owner Meaning受理、データ開示、外部EffectとReceiptは従来の独立境界。Pluginの自己宣言をPermissionやContract Publisherの資格へ昇格させない。

## 3. 合成の規則と境界

- 同じTarget/Scopeに要求するTrait集合は原則として **交換的・結合的・冪等**：`A+B=B+A`、`(A+B)+C=A+(B+C)`、`A+A=A`。重複したパラメータは同一値なら統合、異なる値なら契約上の両立性を評価し、後勝ちを禁止する。
- 便利な **Profile** は原子Traitの正規化されたセットへ展開するシンタックスシュガー。Profile自体を実装義務や別のProvider資格として二重発行しない。
- **Requires / Implies / Refines / Conflicts** を関係として検証する。Refinesは最低保証の強化であり黙った弱化は禁止。依存循環・曖昧Provider・Scope不一致は診断する。未選択PluginのTraitは選択済みBuildへ混入しない。
- **実行の順序は別途明示**。Trait宣言順序に `Retry` / `Transaction` / `Audit` の処理順を隠さない。副作用の再送は既存Operation Identity / Effect Fence / Receiptに従う。
- 合成後の保証は個々のTraitのConformance結果の足し算ではない。Provider版×Consumer版×Target×環境の**exact tuple**ごとに[#3267](https://github.com/hat47x/tei-platform/issues/3267)で再検証する。

## 4. 三領域での実用的な合成

| 対象 | 原子Traitの候補 | 宣言によって生じる具体的な差 |
| --- | --- | --- |
| Data/Field/Table | `Versioned`, `ConditionalWrite`, `HistoricalQuery` | CASまたは対応SQL/Providerを束縛、競合を拒否、時点付きReadを提示 |
| Derived dataset / Cache | `Derivable`, `Rebuildable`, `Disposable`, `Materialized` | 再利用資格/鮮度/Source依存失効、失われたMaterializationの安全な再生成 |
| Session / shared state | `Scoped`, `Expiring`, `ConditionalWrite` | Session/User/TenantごとのIdentity/TTL/更新競合検出。認証Sessionや未同期入力を `Disposable` と偽らない |
| Plugin / Resource Instance | `Discover`, `Read`, `ConditionalWrite`, `Validate` | 必須操作の発見・実装欠落診断・Conformance。Resource BindingはStorageの論理用途と実Provider/Instanceを分離 |
| Build / Function | `Pure`, `DependencyTracked`, `Rebuildable` | 真の実消費入力閉包を使う安全なskip/再計算。Cache hit=Evidence再利用にしない |

**DerivedとTemporalの線引き**：単に過去時点から計算した値は破棄可能なMaterializationでよい。しかし「当時観測された」「確定した」という出来事そのものは、正本のTemporal Recordとして別に保全する。削除可能なMaterializationを保持義務付きの同一実体にしない。過去のビジネス時点、観測時点、記録時点、規則版を混同しない。現時点で再生成可能かは入力Source/Rule/実Provider能力の存在に依存する。

**ブラウザー等の制約Storage**：localStorage/IndexedDB/Cache APIのCapacity、同期/非同期、Origin/Storage Partition、永続性、任意削除、現在のGrantをResource Capabilityで評価。保存できなくても正本を取得できればDerived値の計算自体は利用可能。ただし機密Auth TokenのlocalStorage保存、Snapshot保存義務やOffline可用性の過大保証は拒否する。

**データ鮮度**：Retention（保持）/Freshness（利用時の最新性）/Refresh Duration（再構成所要）/Availability（必要時に使えるか）を別契約にする。ユニケージ的な事前集計＋当日差分によるHybrid Readは、Watermark/欠測/遅着/削除/訂正を追跡できる場合に限る。最初はRead-only実行で計測し、独自バッチエンジンは作らない。

## 5. CLI/MCP/Studioからの利用（コマンドは提案のみ）

```text
tei trait inspect <subject> --evidence
tei trait requirements <subject>
tei trait validate <subject>
tei trait explain <subject> --target <target>
```

表示するのは **Declared / Selected / Implemented / Tested / Current Authorized** を混ぜない構造化結果。未取得、partial、permission-denied、not-applicable、stale、unsupported、unknownを区別する。Operationの既存read-only CLI/Studio共通Serviceを再利用し、専用の第二Trait UI/Registryを作らない。開発者HarnessのSkillは同じCLI出力から必要コードと独立Test Packetを作成する。

## 6. 実装順と撤回条件

| 段階 | 実施 | 受入根拠 |
| --- | --- | --- |
| **T0** | 既存Semantic Trait Claim、Contract/Provider、Resource/BOM、Derivedの能力を固定版で棚卸し | 新規共通フィールドを増やさず表現できる範囲を明示 |
| **T1** | `ConditionalWrite`要求を持つ一つのProviderの手動実装欠落→実装→再Validation | 同一Subject/Targetで独立Expected、Source位置、未実装診断、実Providerの整合性 |
| **T2** | 合成 `A+B/B+A/A+A`・Profile展開・矛盾・Refines負例 | 正規化要求、適用範囲、違反と根拠、未知の区別 |
| **T3** | #3271 Memory/Local/tmpのDerivedDataと#3267 2×2 Matrixに実適用 | キャッシュ失効とData/Temporal Record保全、旧Evidence誤復活を防止 |
| **T4** | CLI/Studio同一Operation、Go単一バイナリと後続TargetのConsumer Evidence | 未選択Plugin混入なし、Target/Scope/Grant別資格、手動Trait実装の移植 |

独立Expectedは[#2514](https://github.com/hat47x/tei-platform/issues/2514)の既存管理に従い、TC-01〜10は[#3278](https://github.com/hat47x/tei-platform/issues/3278)の受入票を参照する。実証ではC0/C1/NFRの未カバー領域も列挙し、テストを自動生成できてもTarget実実行を省略しない。

**効果指標**：手動実装漏れ検出率、Trait追加時のCore変更数（目標0）、組合せ順序差、Target別の未観測保証数、CLI/Studio間の差、新しい抽象に伴うコード・設定の増分と開発時間。Traitの採用によって実装・検証・説明が実際に簡単にならない場合はProfileやRoleへ縮小する。

**停止条件**：別Resolverや第二Meaning正本が必要になる、Trait名からGrantが発生する、未テスト保証をPASS扱いする、DerivedDataの削除が業務確定記録を消す、Providerの実特性を無視して一律保証する場合。実験の不足は個別Issueの未充足として記録し、既存[#1119](https://github.com/hat47x/tei-platform/issues/1119)へHard Gateを追加しない。

## 7. 他レーンの成熟差分：Traitを「必要な能力の有向グラフ」に接続する

2026-11-07のPlugin-first D1、Data Source Workbench、Verified Composition BOM、Design Contextとテスト能力の議論を照合した結果、Traitを導入する価値は**実装義務を宣言したこと**だけでなく、**その義務が選択・実呼出・検証まで到達したことを説明できる**点にある。既存[#951](https://github.com/hat47x/tei-platform/issues/951)/[#1016](https://github.com/hat47x/tei-platform/issues/1016)のResolver/Composition、[#3262](https://github.com/hat47x/tei-platform/issues/3262)のDesign Context、[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expectedに投影し、第二のTrait Resolver/Runnerを新設しない。

Trait Bindingを解決するときの**一つの義務に対する最低限の照合列**を以下に固定する（項目名は候補。正式Schemaではない）。

| 照合列 | 目的 | 誤昇格禁止 |
| --- | --- | --- |
| Required Claim | Subject Identity、Source位置・Revision、Trait Contract ID/Revision、Parameters、Scope/Target | 同名Trait・似た項目名から同一義務を推論しない |
| Obligation | 必要なOperation、Pre/Postcondition、Failure/Effect、保証の対象集合・対象時点、明示したRequires/Conflicts | メソッド存在やDoc表示を充足としない |
| Selected Fulfillment | #951で解決したProvider・Implementation・Manifest/Artifact exact identity・選択理由 | 未選択Provider、compatible majorだけ、fallback実装で黙って満たさない |
| Actual Consumer | CLI/Studio/実Applicationの実call-site、Request/ResultのScope、Load/Invoke witness | 選択の成功、共通Go関数直呼び、同じMock Resultを実Plugin消費としない |
| Independent Verification | 独立Expected、Scenario/Mutant、実行Target・Environment、観測範囲・未実施Case | 「1つのProviderがPASS」から合成・全Provider PASSを推論しない |
| Present-time Qualification | 現在のGrant/開示・Provider稼働・Source/Artifact有効性、失効条件 | 過去のテストPASSやCache hitで現在権限/意味受理を復活させない |

義務の完了は「対象となるScope/Target/Use-siteで必要な保証がEvidenceにより支持された」という**範囲限定の結論**。無根拠な完全適合フラグではない。欠落した実装、要求未解決、未観測、否定例、権限不足をそれぞれ区別する。

### 7.1 Default実装とManual実装を同じ入口で扱う

| Binding | Provider選択 | バリデーション結果 |
| --- | --- | --- |
| 明示Trait + 適格な標準実装一つ | 既存Resolverが選択し、実適合を検証 | 選択・実Build/Invoke・Conformanceは別々に記録 |
| 明示Trait + 手動実装指定 | 実装Ref、必要なOperation署名と挙動を検査 | 未提供なら `missing-implementation`。雛形生成は完了ではない |
| 明示Trait + 互換候補二つ | 同Scopeで一意に選べない | `ambiguous-implementation` を明示。列挙順・latestで自動選択しない |
| 明示Trait + 非対応Target/Resource | 同じ名前のProviderがあっても必要能力不成立 | `unsupported`。UI/DSLで黙った縮退をしない |
| Hintのみ | 選択・採否の参考 | `required`義務を発行しない。採否理由を説明可能にする |

「Manual」は独立した実装義務の**解消経路**であり、Traitを別種に増やす理由にはならない。Providerの自己申告・メソッド署名・静的形状・実動作のConformanceは異なる観測。契約適合テストの実行が必要なTargetでは、未実行を「実装あり」としか表現しない。

### 7.2 合成の健全性：要求の集合演算と処理の配線を分ける

`A+B=B+A` は正規化した**要求集合**の同値性であり、実際のイベント処理やMiddleware順序を主張しない。`Retryable` + `Transactional` + `Audited` の例では、Retry全体をTransactionに含めるのか、個々のAttemptをTransactionで囲むのかを別の明示Pipeline/Operation Graphで要求する。要求する配線が未定義なら `composition-needs-wiring`、無条件再試行で副作用が二重化し得るなら `effect-unsafe` として利用資格を与えない。原子Traitの数だけから相互作用の安全性を推測しない。

一つのTraitの保証対象に、生成定義・保存実体・業務確定記録など**別Identityの対象**を含めない。例：`Disposable` はDerivedDataのMaterializationを破棄可能にするが、生成定義、未同期の業務入力、特定時点の観測証拠、Ownerが確定したTemporal Recordを破棄してよいとは意味しない。この分離を持たず `Retained` と `Disposable` の「後勝ち」を許す設計は棄却する。

## 8. クリティカルパスを二経路に分離する（P0での相互ブロック禁止）

既存[Plugin-first移行計画](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)の**D1：read-only Inspector/Model Catalog**は、Trait設計の実Consumer実証に最適だが、書込みを伴う `ConditionalWrite` の完全実装を待つ必要はない。従来のT1を次の**並行する二経路**として具体化する（#1119の既存Gateや優先順は増設・変更しない）。

| 経路 | 最初の実装単位 | 最小の受入・負例 | 所有Issue |
| --- | --- | --- | --- |
| **T1-R Read-only Plugin dogfood** | 同一 `inspect` Requestに必要な `Discover`/`Read` などを候補Traitとして照合。CLI/Studioの二つの実call-siteから**同じ選択済みService Provider**へ到達 | (a) Providerが選択されただけなら不合格、(b) 同じ出力でもAを選んだのにBが実際に呼ばれたなら不合格、(c) 未選択/重複Provider、partial、8,192byte切断、Grant失効で完全性を推定しない | #951/#1016/#1158/#1251/#3262 |
| **T1-M Manual obligation** | `ConditionalWrite`を必要とする一つのData Providerを選び、required operationと不一致時の非更新保証を要求する。まず隔離Fixtureで手動実装欠落を検査 | 未実装・型だけ合致・CAS非原子的を区別。実DBでの効果は隔離Target/独立再観測と結合するまで `unverified`。CLI/Studio D1の完了条件にしない | #3278/#951/#2090/#2622/#1174 |

この分割は `Read` と `ConditionalWrite` をそれぞれ永続的なCore Trait enumへ追加する提案ではない。最初に**既存ContractへのRequirement Projectionで足りるか**を検査し、不足が再現された箇所だけ表現を拡張する。

### 8.1 最初の同一Attemptの受入票

```text
Subject / SourceRev / RequiredScope / TraitContractRev
    → Resolved Provider+Contract Lock+Target
    → actual linked-or-loaded Provider artifact
    → CLI call-site / Studio call-site / actual invocation
    → independently expected Result+Failure+Completeness
    → Conformance witness（Targetごと）
    → present Grant / Disclosure（独立した判定）
```

この列の `selected` / `linked` / `invoked` / `tested` / `currently-authorized` は**互いに代用不可**。同じGoサービス関数をCLI/Studioから直呼びしただけでは D1 のProvider実利用を証明しない。CLIからStudio画面を自動表示できたことだけでも、実Studioの独立Consumer E2Eとは呼ばない。

必要なPluginが無効化・更新・アンインストールされても、Authoritative Source byteは保持し、利用不能なTraitに関する診断は `unknown`/`unavailable` として出す。要求される業務的な保証と、実装が存在するかどうかを混同しない。

## 9. テストへの接続：Traitのカバレッジと実Target資格は別

[#3278](https://github.com/hat47x/tei-platform/issues/3278)のTC-01〜10を[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expectedへ接続するとき、**単なるTrait名の行網羅**ではなく、以下の列をCaseごとに検証する。

1. **要求C0**：Required Trait/Operationごとに少なくとも一つの正例または不正例を特定できるか。表にあるだけでは実行済みでない。
2. **分岐C1**：missing / ambiguous / unsupported / conflict / satisfied-under-scope / unverified の各診断、Manual/Generated/Providerの各解消経路、Profile/原子Trait展開について実際に通った分岐を区別。
3. **変更・失効**：Implementation/Contract/Sourceが変わった場合、具体的にどのTrait Fulfillment Evidenceを無効化し、どのBOM exact tupleを再検証する必要があるか。A→B→Aや同Version別digestの誤復活を拒否。
4. **NFR/Resource**：Cacheの場合は生成時間、容量、鮮度、TTL、offline時の依存元利用不能、Browserの任意削除やStorage書込み失敗。性能悪化と意味誤りを別票にする。
5. **権限・効果**：Traitを実装してもHost/Provider現在Grantは別。例えば `Read` Trait適合とSource開示許可、`ConditionalWrite` 適合と実DB更新許可、`Disposable` と監査記録の削除権限を混同しない。

**追加の独立負例（TC-11〜15、設計・NOT RUN）**：

| ID | 差分入力 | 独立Expected（必要な拒否・表示） |
| --- | --- | --- |
| **TC-11** | D1でProvider Aを選択し、CLI/Studioが同じ値を返すが、実call-siteはBを呼ぶ | 同値ResultでもD1の実Provider履行・Trait適合を拒否。actual invocationのProvider Identityを比較 |
| **TC-12** | 未実装Manual Traitに `@hint(cache_candidate)` が同時付与 | Hintでrequiredを充足したと誤認せず、missing-implementationを維持 |
| **TC-13** | CacheのKey/Byteは同じだが、別User/Tenant・Source Scope・現在Grant失効 | Cache hitから開示不可。TTL内でも拒否/再資格化。認可や秘密の存在を漏らさない |
| **TC-14** | 業務確定済み受注額と、同じ導出式の破棄可能な再計算値を持ち、元価格を遡及訂正 | 再計算値だけ失効・再生成、確定記録は無断変更禁止。時点とSource Revisionを維持 |
| **TC-15** | Trait契約だけが変更、またはImplementationだけが変更して以前と同Byteの値を返す | 同値Resultで全Consumer Conformanceを自動再利用しない。Exact Contract/Artifact/Targetに対するEvidence差を示す |

**テスト能力そのものの判定**：AIがテストケースを提案した／CLIにコマンドがある／MockがPASSした／実Providerに到達した／独立Expectedを満たした／Ownerが意味を受理した、は各々異なるClaim。生のテスト結果から「すべてのTraitにC1保証あり」や「検証済BOMが全組合せを保証する」を導かない。具体的なTest Runnerと結果管理は既存#1180/#2514/#3267の単一正本を使用する。

### 9.1 直近のIssue引継ぎ（文書の成立 ≠ 製品の実証）

- **#3278**：T1-RとT1-Mの分離、TC-11〜15、対象・必要Contract・実Provider・実呼出・独立Expectedの同一Attempt結合を管理。
- **#951/#1016/#1251**：候補Providerと実選択・Link/Load/Invokeの差を機械で観測する。現行CLIの固定Command RegistryへTraitごとの分岐を追加し続けない。
- **#3271/#2600**：DerivedDataのReuse eligibilityとビルド実消費閉包は別管理。Cache hitはArtifact/Evidence/Grant資格のいずれでもない。
- **#3267**：Trait適合はexact BOM tupleの必要条件に過ぎず、複数Trait合成とTarget間の相互作用を独立検証。
- **#3270/#2622**：DataSource/DB schemaのTrait適合はProvider×Target×Schema/Principalで評価。DBA操作、権限未確認、部分観測を黙って一律保証しない。
- **#3262/#1158**：生成AI向けの `inspect→requirements→validate→explain` は将来のOperation Projection候補。開示条件/Source scopeに合わせて情報を返し、未実装コマンドを実在すると説明しない。

**状態**：本追補は横断した契約と受入票の成熟化のみ。T1-R/T1-M、TC-11〜15、実Go Build/CLI/Wails/DB、Conformance、Performance計測、Owner Meaning Acceptanceはすべて未実施。実行上の成果は各OwnerのEvidence/Issueに限って記録する。

## 10. 2026-11-07後半の実証を反映：Traitの「必要集合」とCapabilityの「選択率」は別の指標

新しくmainへ統合された [Plugin能力選択のReference #3337](https://github.com/hat47x/tei-platform/pull/3337) とその[技術名解決の修正](https://github.com/hat47x/tei-platform/commit/641fff18d1a40591543461f043c827d86e090c8b)、[Context部分集合の負例 #3354](https://github.com/hat47x/tei-platform/pull/3354)、[実行経路を持たないHookの検出 #3306](https://github.com/hat47x/tei-platform/pull/3306)から、要求の閉包を証明せずに「Traitが充足された」「未使用」とする危険が具体化した。

**実際に観測された限定事実**：[先行Reference](../evidence/PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08/README.md)は候補能力12件のうち6件を `unselected` と表示した。これは当時の分類規則に `database-provider` の技術名→能力解決が含まれなかったためで、[後続修正の実行記録](../evidence/PLUGIN_CAPABILITY_COVERAGE_TECHNOLOGY_RESOLUTION_2026-11-08/README.md)は**同じ12件中4件**を `unselected` とした。旧Evidenceは当時のSource/Rule Fingerprintに束縛した歴史的な記録として残す。新しい結果が旧Evidenceと矛盾したことを、無条件に「環境が変わった」「利用率が上がった」と説明しない。**分類規則のRevisionが変わった**ためである。

このReferenceはあくまで「宣言されたCapabilityがEnvironmentのBindingへ関連付くか」を報告する**read-only候補観測**。同じPluginがEnvironmentへ明示選択されていても、そのPluginの個々のCapabilityのBindingがなければ選択済みとはならない。同じPluginの一能力が使われても、他能力は使われないことがある。さらに `database-provider` のように提供技術へ複数候補が一致した場合、Referenceは**両方をselectedと表示し得る**。これは一意なProvider解決・実装適合・Invocationの証拠ではない。

### 10.1 分母と対象集合を混同しない

| 問い | 分母／独立根拠 | 結果とClaim上限 |
| --- | --- | --- |
| Candidate Inventory | 指定したSource/Environmentで読めたPlugin `provides` の集合 | 候補の観測に限定。未読込・権限外Pluginの不存在は主張しない |
| Capability Binding Coverage | その候補と環境Bindingの照合（解決規則版も必須） | `selected/unselected` はBindingの一致。実Provider一意性・利用実績を主張しない |
| **Required Trait Coverage** | **独立に固定したFunction/Consumer/Target/ScopeのRequired Trait/Operation集合** | Requirementsが完全に取得できた範囲でのみ `fulfilled/missing/unverified/not-applicable` を判定 |
| Actual Fulfillment | exact Provider・Artifact・Call-site・結果・Independent Expectedと実Target観測 | そのAttempt/Scopeで成立したContractのみ。現在Grant、業務受理、配送受領は独立 |
| Test Coverage | #1180/#2514の独立Scenario/C0/C1/NFR/Negativeの母数 | Test caseの提案・発見・実行・PASS・実Provider資格を分離 |

**未観測のRequired Traitをゼロとしてはいけない。** #3354の実CLI Context検査では、見かけ上正常なJSON/Revisionでも必要なRule/Relation/Sourceが欠け得ることを限定例で検出した。Traitについても、`Required=[]` が自己申告されただけ、CLIが1ページ返しただけ、非開示Contextが省略されただけなら「必要Traitなし」や「充足率100%」にはならない。Required Closureが `partial/unknown/permission-denied` であれば、**母数と充足率を `unknown` とし、限定的に観測済みの事実だけを返す**。秘匿Traitの有無・件数を診断から漏らさない。

### 10.2 既存Referenceを壊さず追加する具体的Pressure

今回の[小さな回帰試験](../../tools/reference-plugin-capability-coverage/coverage_test.go)は**既存read-only Referenceの分類境界**だけを固定する。実Trait API・Resolver・Conformance・実DBを実装しない。

| ID | 入力差分 | 独立Expectedと引継ぎ |
| --- | --- | --- |
| **TC-16** | EnvironmentがPlugin Packageを明示選択するが、そのPluginが宣言する個別CapabilityにBindingなし | Package選択 ≠ Capability選択。Referenceでは当該能力を `unselected` として残し、使用不能／欠陥とは断定しない（#1251/#951） |
| **TC-17** | 1 Pluginが2 Capabilityを宣言し、1つだけEnvironmentがBinding | 同じPlugin内でも能力単位でselected/unselectedを分離。他能力を自動充足しない（#3279） |
| **TC-18** | 1 DB技術を2 Providerがサポート | Coverage上は両候補selectedを取り得るが、一意Provider解決の成功ではない。Resolverが曖昧診断を出せるかは#951の**別実証** |
| **TC-19** | Required Rule/Traitを含むSourceの1つを隠す/途中切断する。残りのJSONとFingerprintは見かけ上正常 | Independent Required Closureを根拠に不足を検出。`absent`や`100%`を返さず`partial/unknown`、秘密情報の開示もなし（#3262/#2514） |
| **TC-20** | Source/Environmentは同じで分類規則だけ変更し、12候補中`unselected`が6→4へ変わる | Rule Fingerprintを比較し、古い結果の資格を現在版に無断移植しない。`selection improved`等の実環境変化として誤報しない（#2600/#3267） |

### 10.3 最短の導入手順と判断境界

1. **P0-A**：現在mainの `reference-plugin-capability-coverage` でTC-16〜18を実行し、実Environment/Source/Rule Fingerprintを同じAttemptへ束縛。これはCapability Coverageの負例であり、Trait Conformanceとは呼ばない。
2. **P0-B**：#3262のread-only Required Closureを利用して、一つのFunction/Inspectorに必要なTrait/Contract集合を**独立Fixtureで固定**し、Declared vs Selected vs Missing vs Unknownを比較。#3338のFunction内配置・Topology Bindingは別Owner／Draftに置き、ここで重複実装しない。
3. **P1**：#951の既存Resolverと#1251の実Consumerへ実際に渡し、`selected`→`linked`→`invoked`→`conformance-observed`の不足辺を測る。#3267 BOMのexact tupleや#1119の既存Gateへ必要な観測だけを受け渡す。

**採用・撤回基準**：Trait固有の巨大Catalogや第二Resolverを作らず、追加したTraitに対して「どの必要操作が、なぜ利用できるか／できないか」がSource/Scope/実証に基づいて説明できること。実Providerへ到達しないのにCoverage%だけ改善する場合は新しい抽象・画面を増やさない。

**状態**：#3337・技術名解決修正・#3354・#3306はリポジトリの限定的な既存実証。今回のTrait Required Closure、TC-19/20の一連の実行、CLI/Studio actual two consumer、正式Trait Schema、Owner意味受理は**NOT RUN / NOT ACCEPTED**。履行資格を既存Issueの権限・証拠へ委譲する。

## 11. Traitの付与先と有効化境界：Source記述から実装義務へ飛躍しない（2026-11-08追補）

[#3368](https://github.com/hat47x/tei-platform/pull/3368)がmainへ統合した限定回帰（Capability宣言・Environment選択の境界）と、他レーンの `*.tei.md` Authoring、Role/Facet、標準データ構造、Function内の表現配置／Topology Binding、Design Context、Browser/Session/DerivedDataを照合した。**新しい成熟点は、Traitの付与先（何に保証を与えるか）と、Traitがいつ実装義務として有効化されるかを独立にすること**である。名称・構文・表示から実装を暗黙導入しない。

### 11.1 一つのTrait名でも付与先の同一性を変えると保証は変わる

`Versioned` を例にすると、`Product.price` の履歴取得、`Product` 行全体の競合制御、`ProductStore` ProviderのCAS能力、`UpdatePrice` Operationの前提Revision検査は**別Subject／別Effect境界**である。Row更新と項目履歴は物理実装を共有し得るが、物理Column/Go Methodが共通だから全てにTrait Bindingが存在するとは判断しない。

| 付与先（例） | 明示的に要求される保証の単位 | 実装選択・観測の責務 |
| --- | --- | --- |
| Data Field | 対象値・業務有効期間・欠損/NULLの意味 | Field→保存Mapping→履歴/Query。行全体の履歴は暗黙に含めない |
| Table / Relation | 行Identity・多重度・書込み・保存期間 | DB Resource/Providerの実制約とMigration結果。1項目の要求からDDL成功を推測しない |
| Function / Operation | 入出力、使用Context、整合性/Effectの前後条件 | Function内の実Read/Write Mapと現在Grant。Reference同名でも異Scopeは別 |
| Plugin / Provider / Resource Instance | Offers/Requires・実Implementation・Target/InstanceのCapability | 選択済Providerと実Load/Invoke。自己申告、Provider技術名の候補一致は適合ではない |
| UI Representation / Session | Functionに配置された表現インスタンスの状態共有、Lifecycle、表示/編集能力 | Data topologyと表示topologyの明示Mapping、User/Tenant/Session権限。表現可能≠編集許可 |

`@Trait` 相当のMarkdownテキストは、ファイルがManagedであるだけではBindingにならない。**Raw Source保全→実Loader Read→構文/Role認識→Typed Subject/Scopeへの束縛→Meaning Intentの採否→必要Contractの解決**を区別する。Provider PluginがdisableされてもRaw Sourceは保持する。未Parsed/未BoundのTraitを「要求がない」「実装済み」に数えない（[#2620](https://github.com/hat47x/tei-platform/issues/2620)、[#3262](https://github.com/hat47x/tei-platform/issues/3262)、[#3030](https://github.com/hat47x/tei-platform/pull/3030)）。

### 11.2 Trait義務の起動条件と診断の非対称性

実行前判定に使う **Required Closure** を独立に固定し、現在のRead Scopeに対する完全性を確認する。取得できた一部の `provides`／Capability Bindingだけを分母にしない。独立に確認できた **known missing** は即診断できる一方、未取得のRequiredが残る場合は「missing 0」とも「complete」とも判定しない。

| Source/Meaning・Required closure | Known Requiredの履行 | 許されるAssessment |
| --- | --- | --- |
| 完全で、対象Intentが受理済み | 必要な全実装と適合証拠がScope内で存在 | そのScope・Targetに限定した `satisfied` 候補。現在Grantは別 |
| 完全 | 実装が一つ以上欠落 | `unfulfilled`。失敗点をBinding/Contract/Implementationで示す |
| 部分的・未Loaded・権限外・途中切断 | 観測済みRequiredの一つが欠落 | **既知の未充足**を示しつつ、全体Assessmentは `partial/unknown`。秘匿分母は非開示 |
| 部分的・未Loaded・権限外・途中切断 | 観測済みRequiredは全て履行 | 全体は `unknown`。100%・Required空集合・利用資格へ昇格しない |
| 全入力が確認されRequired集合が空 | 対応するConsumer/Use-siteには義務がない | `not-required`。未読込による空集合と区別する |

この表は新しいCore status enumや第二Readerの提案ではなく、既存のContract/Design Context/Conformanceを参照する**読み取り側の判定規則**。非開示Requiredの件数・名前・PathはUnauthorized Consumerへ返さない。

### 11.3 実装方式の選択は「どの時点で拒否するか」とも接続する

- **Authoring時**：未BoundのTrait、適用対象の不一致、Profile/Required集合の矛盾、手動実装の必要箇所を診断。候補作業を保存することは許すが正式な利用可能Operationを公開しない。
- **Build/Generation時**：受理されたRequired Contractに対しGenerated/Provided/Manual/Externalのいずれかの実現を解決。選択済みImplementationの実Build/Input Closureを確認。Generatorが出力しただけでは実Target Conformanceでない。
- **Runtime/Operation時**：既存Host/Providerによる現在Grantと実Load/Invokeを確認。Sourceが変化した場合、古いEvidenceを再利用する前にその資格を再評価。認証/監査Sessionの削除をCache Evictionへ委ねない。

既存のD1 read-only Inspector（T1-R）は**Authoring→選択→実Invocation**、ConditionalWrite（T1-M）は**Manual→契約不充足診断→実ProviderのCAS検証**を分担する。両者を一つの新Hard Gateへ統合せず、[#3279](https://github.com/hat47x/tei-platform/issues/3279)のFulfillment Owner、[#951](https://github.com/hat47x/tei-platform/issues/951)のResolverへ引き渡す。

### 11.4 追加の最小反例と独立Expected（全て提案／NOT RUN）

| Case | 操作／改変 | 期待される最初の不足・拒否 |
| --- | --- | --- |
| **TC-21** | Fieldに`HistoricalQuery`を付与。行履歴は存在するが対象Fieldの有効時点を保存していない | Row版のみでField時点Query義務を充足させない。Field→History Mappingが不足 |
| **TC-22** | `*.tei.md`にTrait文字列を置くがLoader未解釈、またはSourceのRole Bindingが曖昧 | Raw textを保持したまま`not-loaded/unbound`。Required 0や自動Default Implementationを生成しない |
| **TC-23** | 独立FixtureがRequired [A,B]。AだけReaderから観測可能、BはSource欠落。AのProviderはselected | Coverageはpartial/unknown。Aの選択とBの不在証明/全Trait適合を混同しない |
| **TC-24** | `ConditionalWrite`をManual実装したが、実体は通常Read→無条件Put。二つの同一Expected Revision更新を同時実行 | 署名/逐次テストPASSではなく、隔離実Targetで**一件だけCommit可、他方はConflict、Post-image一致**を確認できなければConformance未成立。テスト環境では独立Barrier/Receiptが必要 |
| **TC-25** | Browser Cacheの`Disposable`と、同じ値由来の確定Temporal Record/未同期Session入力を格納 | Cache Evictionで正本・時点観測・未同期編集を消さない。Data IdentityとRetention Subjectを別に束縛 |
| **TC-26** | Traitの標準実装Pluginを無効化。旧Sourceが保持され、旧Build Result/Cache byteが一致 | Raw Authoringは可、Required Operation Offer/Runtime資格は保留。旧Direct fallbackで暗黙利用しない |

**直近の実装判断**：新しいTrait構文・Core enum・中央Evaluatorを作る前に、TC-22/23を既存Design Context Consumer、TC-24を既存Provider/Scenario Conformance、TC-26をPlugin-first D1の受入票で再現し、どの不足が既存契約では表現できないかを記録する。受入前はすべて `design-only`、同じCounterexampleを別Issueへ複製して第二Expected正本を作らない。

**採用判断**：Traitが単にメタデータを増やすだけでなく、具体的なImplementation Obligation・Missing Diagnostic・安全な自動構成・独立検証のいずれかを実際に改善する場合に限定して標準Trait候補へ進める。実Provider/CLI/Studio/DB、C0/C1の新規実行やOwner意味受理は本追補では行っていない。

## 12. ADR-0048と実装義務：共通Operationを実際に履行するまで（2026-11-08 横断引継ぎ）

**今回の新しい判断**：TraitをCLI、MCP、Studio、Web、生成アプリごとに実装するのではなく、**採用済み共通Operation契約のRequired部分**として束縛する。一方、共通Operationが実行可能であることと、そのOperationが要求するTraitを**選択されたProviderが実際に履行したこと**は別の主張である。

- [採用済みADR-0048／PR #3439](https://github.com/hat47x/tei-platform/pull/3439)：共通OperationのIdentity／Input／Result／Failure／Effect／Preconditionを定め、CLI argv、MCP、HTTP、WebViewはAdapter。TEI共通HostやすべてHTTP化を新設しない。
- [main統合PR #3458](https://github.com/hat47x/tei-platform/pull/3458)：`validate`についてCLI/MCPの同じResult文書と意味エラー応答を限定的に確認した。**Transport/Result parityであって、Provider selection／actual invocation／Trait ConformanceのEvidenceではない**。
- [main統合PR #3408](https://github.com/hat47x/tei-platform/pull/3408)：`editor.sql_analysis`と`editor.database_workbench`では現在のCapability Validation適用範囲が異なることを限定実測した。CLIが `validate ok` を返しても、全Function/Representationの必要Traitが完備したとは判定しない。
- [main統合PR #3477](https://github.com/hat47x/tei-platform/pull/3477)：Context→Scenarioについて、Sourceのみの改訂をWorldだけでは見逃し得る反例と再照合を限定実証した。**Source Reference／Index RevとTrait Bindingの失効**を保持し、同一結果ByteやWorld FingerprintだけでRequired Closureを復活させない。
- [main統合PR #3440](https://github.com/hat47x/tei-platform/pull/3440)：Stage1実行物とStage2のinvoker/resultが一致する限定Self-build実測を前進させたが、Providerの実Link/Invocation、選択済Plugin Closure、一般的Target/BOM資格には転用しない。
- [main統合PR #3478](https://github.com/hat47x/tei-platform/pull/3478)：HTTP Decision Graphが実Go Hostの分岐、監査順、202の不確定状態を限定実証した。操作の成功・失敗・不確定状態やEffectを**単一のHTTP Status／exit code／Trait名に還元しない**。

上記の追加は[Plugin-first計画§16.6](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)の実証と重複する新機構ではなく、**TraitがOperationのどこで義務を発行・履行・失効するか**というOwner #3278 の不足辺に限定する。

### 12.1 一つのOperationに束縛するTraitの最小照合キー

```text
Source Occurrence / Role / Subject Identity / Revision
              │（独立Required Closure）
              ▼
Operation ID + Variant + Input / Failure / Effect / Scope
              │
              ▼
Required Atomic Traits + Parameters + Contract Revision
              │
              ▼
Existing Composition/Provider Selection (exact Lock)
              │
              ▼
Implementation (Default / Generated / Manual / External)
              │
              ▼
Actual Target Artifact / Provider Invocation / Typed Result
              │
              ▼
Independent Expected / Conformance / Receipt / Current Grant
```

- **Operation Variantの違い**：同じ`view`がpreview時にデータの読取を行い、引数で実出力ファイルを作るなら、`ReadOnly`の宣言を「すべてのFilesystem／通信／Session Log副作用がない」保証へ拡張しない。対象Resource/Effect/Invocation Contextを明示した要求でなければ `effect-unknown` と診断する（[#3182](https://github.com/hat47x/tei-platform/issues/3182)がEffectのOwner）。
- **Semantic Typeとの分離**：Role/Facetの意味上の主張（例：日時、正本、導出）を、直ちに手動実装義務やAdapter提供能力へ昇格しない。`required`が適格なSubject/Operationに束縛されたときだけ実装義務を発行する。便利なProfileは原子Traitへ展開するが、合成後の配線条件は別途検証。
- **Targetの違い**：同一Operation/RequiredをCLI・MCP・Studioで共有しても、実Provider／Artifact／Current Grantは異なり得る。Binding/Invocationの一致はそれぞれのTarget/Attemptで検証する。CLIで一度`validated`となったTraitをWeb/Hostまで自動資格化しない。
- **Manualを強制する場所**：選択された実装にDefaultがなく `manual-required`である場合、Authoringは候補保持＋不足の診断、Build/Offerは使用するRequired Closureの未充足を拒否する。署名一致、Source内の実装らしい文章、未選択Providerによる代用、Advisory Hintは完了証拠にならない。
- **状態が途中で変わる場合**：Source-only drift、Trait Contract改版、Provider/Generator/Artifact変更、Context Scope・Grant失効を区別する。Build成果物が同一ByteでもSource Observation、Trait Fulfillment Evidence、現在の開示/Effect資格を自動再利用しない。

### 12.2 最小の検証パケットと誤ってPASSにしない条件（既存#1180/#2514へ委譲）

| Case | 対照入力 | 独立に期待する結論 |
| --- | --- | --- |
| **TC-27：共通Result** | CLI/MCPの`validate`が同一ResultでもProvider Bindingを与えない | **Result parityだけ**成立。Trait実装・Provider実Invocation・Studio等の資格は`unverified` |
| **TC-28：Variant Effect** | 同じOperationのPreviewとファイル出力Variantを実行 | Invocation VariantとEffect/Sinkを個別判定。read-only文言でFS書込みを隠さない |
| **TC-29：検証非対称** | Capabilityなしの`editor.database_workbench`を使い既存`validate`が成功 | Validator未検査部分を `unknown/not-proven`。Required/Provider満足率100%を出さない |
| **TC-30：Source-only drift** | 同じWorldモデル結果を返すが、Trait宣言を含むSourceだけ変更 | Source Index/Occurrence/Rule Revを見て影響Binding/Expectedを再評価。旧Evidenceの誤復活なし |
| **TC-31：Manual Providerが別Target** | GoではDefault実装あり、Browser/外部Hostは手動実装欠落 | Target別の`unfulfilled`。同名Operationと同じ結果Byteから履行成功を推定しない |
| **TC-32：Composite＋不確定Result** | `Retryable + Audited`の副作用Operationが202/不確定応答 | 不明な業務Effectを成功/未実行に決め付けない。Pipeline/Operation Identity/Receiptが不足なら再実行適格としない |
| **TC-33：BuildとRuntimeを分離** | 限定L3セルフビルドが通り、必要Providerは未Link／未Invoke | Self-buildの観測は維持しつつTrait実Target履行・BOM tupleは`unverified` |

**既存実証への依存関係**：TC-27のResult parity、TC-29の現在Validator差、TC-30のSource-only差検出、TC-33の限定L3相当は各専属PRの観測であり、**TC-27〜33全体のTrait縦断試験が実行済みという主張ではない**。D1 CLI/Studio ProviderのDraft [#3381](https://github.com/hat47x/tei-platform/pull/3381)／[#3371](https://github.com/hat47x/tei-platform/pull/3371)は未統合候補。これらのコードから新たな既定Runtime成立やTrait Conformanceを推論しない。

### 12.3 計画へ戻すP0の一辺（他領域をHard Dependencyにしない）

1. **P0-1：契約照合**。一つのread-only `inspect` Use-siteを固定し、ADR-0048のOperation ID／入力／Result／Failure／Effectと、既存#951/#1016で選択可能なInspector Providerを一つの*read-only*要求票として照合する。`inspect` CLI/StudioはExisting Typed Adapterで接続する。Source Root/Index/Required Closureを#3262と突き合わせ、不完全なら未知を保持する。
2. **P0-2：実呼出し**。CLI Consumer AとStudio Consumer Bが**それぞれ**実Targetから選択ProviderのCallbackに到達することを別Attemptで確認。Provider A/Bの値を意図的に同じにした負例で **B selected / A invoked** を必ず失敗させる。旧Builtin fallbackを黙って利用しない。
3. **P0-3：Traitの不足診断**。同じProvider Contractの必要な一原子TraitにDefaultを持つ経路と `manual-required`経路を用意する。前者は実Provider Targetの適合、後者は未実装診断→独立Expectedによる境界テストの順。T1-RとT1-Mを並行継続し、read-only Plugin-firstをCAS実DBの実証完了待ちにしない。
4. **P1：横断消費**。#3267のexact BOM Matrix、#3271のDerived/Temporal、#3274のHTTP Decision Graphは独立Ownerの同型Contract Consumerとして後続適用する。意味受理/Host Grant/外部送出/実DB効果/実Wails/Qualified BOMを単一成功へ合算しない。

**合格条件**：採用済み共通Operationと既存Contract/Resolver/Scenarioを使用し、原子Traitが「実装か不足診断」という可観測な差を生み、Source/Required/Provider/Actual Invocation/Independent Expected/Effect/Scopeが同一Attempt上で照合できること。**失敗時は、足りない一辺を正確に返し、不要なTrait Core enum／新Host／第二Operation正本／第二Trait Resolver・Test Runnerを作らない。**

**今回の到達範囲**：ADR-0048採用と各main PRの限定成果の静的照合に基づく設計・実証条件の更新。TC-27〜33、D1 CLI/Studio actual two consumer、Trait Manual Conformance、実DB／Web／外部Host、Qualified BOMの新規実行は **NOT RUN／NOT ACCEPTED**。この追補のMarkdown統合と、実Provider適合・Owner意味受理を混同しない。

## 13. 実装を前進させる二つの独立Consumer：Trait義務と実Provider Readを一件ずつ証明する（2026-11-08追加）

**横断で成熟した判断**：共通API/Plugin/Traitの図をさらに増やす段階を終え、**互いに混同できない二つの実Consumerを並行して通す**。先行するCLI/Studio Inspector D1（開発ツール側）と、`order_entry.recent_orders` のFunction/Query/実データReader（生成アプリ側）は、**同じ契約照合方式**を利用するが、同じProvider Instance・Host Grant・Build Target・Source Meaningを強制しない。共通化するのは `Subject + Required Obligation → selected Implementation → actual invocation → independent verification` という証拠の**形**である。

他チャットの[一機能横断受入 #3481](https://github.com/hat47x/tei-platform/pull/3481)と[Design Context 13応答の限定実証 #3482](https://github.com/hat47x/tei-platform/pull/3482)、[Data Workbenchの受入入口 #3498](https://github.com/hat47x/tei-platform/pull/3498)を、#2586（実Function Read）、#1251（Plugin-first D1）、#3279（Trait Fulfillment）の**独立Ownerが衝突しない接点**として整理する。#3482の `sales.order.submit` はSemantic Capabilityの観測対象であり、`order_entry.recent_orders` の実Browse/Queryが選択・実行された証拠ではない。

### 13.1 固定済みSourceから要求を過大生成しない

現行mainの `reference/workspace` に存在する**実Source宣言**は次のとおり。ここで記述されているのはCandidateの接続であり、実ProviderがそのSQLを実行したという観測ではない。

| 既存Source | 確認できる宣言上の事実 | Trait/Operationとして未証明のもの |
| --- | --- | --- |
| `applications/order-entry/patterns/recent-orders.tei` | `pattern order_entry.recent_orders browse` が `data sales.order` と `query order_entry.recent_orders` を指す | そのPatternがProduct Function/実OperationのCall siteで使われた事実 |
| `applications/order-entry/queries/recent_orders.tei` | `datasource sales`、`sql postgres file "recent_orders.sql"`、Column→Semantic Field対応を宣言 | SQL byteと引数の実Loader Read、実Row/Field対応、Result完全性 |
| `applications/order-entry/queries/recent_orders.sql` | `public.orders`、`:from`、`ordered_at DESC` を含む | `:from` の実Binding/Time Zone、実DB取得件数・並び順 |
| `applications/order-entry/application.tei` | Application `order_entry` が論理 `database sales` を `target sales_core` へ要求 | 実Environment Alias/Instance/Grant/Providerの一意選択・現在権限 |
| `semantic/sales/order.tei` | `sales.order` の属性・関係、および別途 `capability sales.order.submit` を宣言 | `submit` に由来するWrite/Transaction Traitを `recent_orders browse` へ自動伝播させること |

**原子Traitの必要性は実Use-site相対**。Browseでは例えばRead/Query Parameter Binding/Typed Result等の義務を独立に照合し、`sales.order.submit` が別途存在するだけでMutation/CAS/Retryが必須だと推測しない。これらの英語名は**Contract要素候補**であり、Trait IDの固定・Core enumへの追加ではない。逆にSourceを全部取得できていない場合、Read義務が十分だという完全性Claimも禁止する。

### 13.2 二Consumerで共有すべきWitnessの最小列

| Witness（論理形のみ） | 開発ツールConsumer：CLI/Studio Inspector | 業務Consumer：`recent_orders` Query Read |
| --- | --- | --- |
| Subject & Source | 対象Kind/ID、Workspace Root、実Source Index/World Revision | Pattern→Query→SQL Source bytes→Semantic Field→Application DB Aliasの出典 |
| Required Obligation | `inspect` 共通OperationのTyped Input/Result/Failure、必要Read/Inspect契約 | Browseが実際に要求するQuery Read/Parameter Mapping/Result Shape/Scope |
| Selected & Built | #951/#1016のexact Tooling Provider/実Binary、CLI/Studio別Call site | `sales_core` のEnvironment Instance、選択Data Provider、実Query/args/Target Artifact |
| Actual Invoke | CLIとStudioが別々に**実選択Provider**を呼んだRecord | SQL/argsを選択Providerが実行してRowset/Field/Scope/Cohortを返したRecord |
| Independent Expected | 同値値でも選択B／実Aの不一致、Provider欠落、古いRoot、未Loadedを拒否 | 既知ID・完全な0件・Provider Error、`:from`境界、未取得/NULL/partial/Join増幅を独立に比較 |
| Claim Ceiling | Provider未Link/未InvokeならD1のPlugin利用は未資格 | 実SQL/実DB/実Rowsetの観測なしではProduct Read未資格。BOM/Grant/Meaningは別 |

**Constraint：同じ出力値であっても、選択Provider以外のHandlerから返されたら不合格**。Read/DerivedDataのCache hitから過去のSQL/Source実消費・現在Grant・業務確定Temporal記録を復元したと主張しない。二Consumerのどちらも、保存先やSession Scope、利用時の鮮度要求を必要な独立契約として検証する。ローカル・BrowserのDisposable Resultと、保持すべきObservation/未同期入力は別Subjectとする。

### 13.3 進行順：既存Issueを完成させるための具体的な不足一辺

**P0-T（Tooling；#1251/#1158/#1824）**：先行CLI typed Inspector D1の[Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)とHost選択Studioの[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)を、main基底の既存Service/Resolver/OperationへFit/Gapした上で、Provider A/Bについて**実Consumer二本のCallback**を証拠化する。Draftの「型付き注入口」だけをProvider実選択完了としない。A/B出力が同一になる負例は既存TC-11・PGD-03に委譲し、第二Test Runnerを追加しない。

**P0-R（業務Read；#2586/#2506/#2514）**：既存`order_entry.recent_orders`のSource/SQL/Parameter/DB Aliasの**最初に欠ける一辺**を1 Function・1 Query・1 Providerについて確定する。[#3482](https://github.com/hat47x/tei-platform/pull/3482)の13 Reply/Capability・Application限定観測はReadinessの一部に限定し、実ReadのOracleにしない。実DBへの投入と実Provider Query実行を同Attemptで観測できるまで、Query結果に由来するTraitを`conformance-observed`へ進めない。詳細は#2586のHR-01/02、#3270のRD-01〜06に委譲。

**P0-F（Fulfillment；#3278/#3279）**：上記二経路を同じ**照合列の構造**へ投影し、一つの原子的Read相当の契約が `default/generated/manual` のどの経路で満たされたか、または`required-missing/unknown`かを表示する。手動`ConditionalWrite`の実CAS試験は既存T1-M/TC-24で並行し、read-only二経路のHard Dependencyにしない。

**P1-C（Cross-target；#3267/#3271/#1124）**：両経路の実Target Witnessがある範囲だけで、Version Matrix、DerivedData/Cache、Function Data Context→Panel/Tableの異質Consumerに拡張する。受注がSQLから取れたことと表現型のEdit/Write能力、Cache再利用資格、正式なOwner意味受理は別資格のまま。SHO/KONを新しいTrait RuntimeやAuthorityの所有者にしない。

### 13.4 停止条件・次の判断

- **実装停止**：新Trait enum、独立Registry/Resolver、専用Trait Runtime、第二Operation Reader/Scenario SSOTを先行導入しない。既存Contract/Relationで不足の具体例が取れた部分だけ拡張する。
- **意味の停止**：`sales.order.submit`の存在だけでBrowseのMutation要求を加えない。Sourceの同名・World Fingerprint一致、Partial Required、`validate ok`で実使用/Source読込/Provider適合を断言しない。
- **権限の停止**：Trait Fulfillmentの正例は現Host Grant、DB Schema Apply、外部Egress、Business Meaning Acceptanceを許可しない。
- **費用対効果**：追加の原子Trait定義／Bindingが、対応する手書き分岐・実装漏れ・再テスト対象・Target間の矛盾をどれだけ減らすかを、P0-T/P0-Rの**同一条件対照**で測る。効果が確認できなければ、Trait定義を増やさず既存Operation/Provider Requirementへ縮小する。

**現在の資格（本追補）**：実Source fileとmain統合[#3481](https://github.com/hat47x/tei-platform/pull/3481)、[#3482](https://github.com/hat47x/tei-platform/pull/3482)、[#3498](https://github.com/hat47x/tei-platform/pull/3498)の限定成果を照合した**設計・受入計画**。上記P0-T/P0-Rの実Provider連結、CLI/Studio actual two consumer、Trait Fulfillment独立Expected、正式Source/Query実Rowset、実Browser/Studio UI、BOM資格、Owner意味受理は**この変更ではNOT RUN／NOT ACCEPTED**。既存各Ownerの実際のCI Evidenceを越えて合格を主張しない。

## 14. Traitと共通Hook Slot：生成・カスタマイズ・状態フローの横断接続（2026-11-08）

**新しい横断要件（設計・未実証）**：API／Web／Desktop／CLI／MCP／VUIの標準動作を、TEIで記述・生成したカスタムロジックで差し替えたり前後に追加できる。ただしTraitの原子性を損なわず、人が把握しやすい**どの操作の・どの段階を・何に置換したか**を説明できること。[共通Operation ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)、[Application Extension #2110](https://github.com/hat47x/tei-platform/issues/2110)、[Table/Data lifecycle #1129](https://github.com/hat47x/tei-platform/issues/1129)、[Plugin-first #1251](https://github.com/hat47x/tei-platform/issues/1251)、[Trait Fulfillment #3279](https://github.com/hat47x/tei-platform/issues/3279)のOwnerは変えない。追加の汎用Hook Runner・独自Host／第二Operation Graph／第二Resolverは前提としない。

### 14.1 TraitとHook Slotは別概念

- **Trait**は、対象が**満たすべき原子的な保証・実装義務**（例：ConditionalWriteの原子性、Readableの必要操作）を宣言する。Required/Hintを区別し、Default/Generated/Manual/Externalのどの経路が履行したかを追う。
- **Hook Slot**は、既存Operation／Build／Representation／Flowの**拡張可能な位置**を示す。Slotの存在・Bindingは、そのSlotの実装がすべてのTraitを満たすことやHost認可を意味しない。
- **Custom Implementation**は、SlotにBindingする具体的実装。TEIで記述したFunction/Rule/Operationから生成する方式を第一の候補とするが、既存外部Provider/手動コードでも適格ならよい。生成済みコードを人が直編集して再生成で消す方式を標準にしない（手動コード境界 #2090）。
- **Hook Order / Effect Wiring**は、複数拡張の適用順・Result変換・再検証・失敗伝播を定める有向関係で、Trait要求集合の可換な合成とは独立。同じSlotに複数のReplaceがある場合、Plugin列挙順やlast-winsで決めず明示選択か競合診断とする。

Slotの最小照合キー候補（**正式Schema/APIではない**）：
`(Operation ID, Variant, phase, Subject/Scope, Required Input/Result/Failure/Effect, allowed mutation, cardinality, ordering constraints, exact Contract Rev, Source occurrence, Implementation Ref, Target)`。Raw SourceやPlugin Capabilityの名前一致だけで実挿入済みにしない。

| 操作・変換の候補 | 具体的な意味 | 守る条件 |
| --- | --- | --- |
| `Observe` | 結果・診断のread-only観測や付随するログ記録 | ログは独立したEffect/Sink。機密開示と用途外送出を許さない |
| `Validate` / `Guard` | 必要条件の追加確認。既存拒否を緩めない | 既存必須Validation/現在Host Grantを削除・迂回しない |
| `Transform` | 型付きRequest/Resultを許可した範囲で変換 | Validate後の入力/Subject/Scope変更なら検証・選択を再実施 |
| `Replace` | 明示された一つの標準Implementationを別実装へ差替え | 同一Operation ContractとRequired Traitの保証を維持。権限・Effect Fenceの置換は不可 |
| `Compose` | 二つ以上の既存実装を明示Pipeline/Flowで接続 | 順序、失敗/取消、重複・部分作用、Receipt、State版を明示 |

これらは構成**操作の候補ラベル**であって、独立したCore Trait一覧ではない。`Validate` と `Guard` の意味の最終分割も、実Provider/Action一例を測ってから決定する。`replace`は一律に許可せず、Host／契約Ownerが明示公開したSlotに限る。特に**Authorize／Commit／Receipt／現在の開示判定の保証は、業務拡張のReplace対象にしない**。実行時に渡すTokenやGrantはPlugin自身が発行できない。

### 14.2 三つのLifecycleを混ぜない

| Lifecycle | 候補Hookの適用点と責務 | 禁止する短絡 |
| --- | --- | --- |
| **Application Operation** | Intent → Prepare → Validate → Authorization/Effect Fence → Execute → Observe → Typed Result。適切なSlotへ追加・差替え | UI表示時のValidateと実Commit時の再検証を同一視しない。副作用の結果不明時に自動再送しない |
| **Interaction / Stateful Flow（VUIを含む）** | Input → Decode/Interpret → State/Context Read → Transition Candidate → Validate/Authorize → Operation Invoke → Persist/Receipt → Respond | VUIを単なる音声Renderer/Transportとせず、**状態を扱うAPI＋フロー制御のConsumer**として検討。Voice/画面/UI入力自体を権限・業務意味の正本にしない |
| **Generator / Build** | Model受理 → 生成計画 → Target Lowering → Template/Implementation Binding → Build → Artifact/Evidence | Generator Hookの存在でRuntime Callback実行・Owner Meaning受理・Artifact再利用資格を保証しない |

共通化するのは **Source／Operation Contract／Trait Obligation／実装選択／Evidenceの関係**。すべてを一つのフック一覧にしない。Build-time Pluginが生成した静的GoバイナリへRuntime Pluginが動的挿入されるとは仮定しない。生成Go・Java/.NET・Browserについて静的合成／外部Process／Target-nativeを個別に実証する（#2390/#943/#1121）。

**VUIの状態契約**：Conversation/Session Identity、Flow/State Revision、業務Event/Transition、Caller/Tenant/Host Grant、Suspend/Resume、Outcome/Receiptを独立に照合する。途中入力や未送信Intentは一般に非破棄のManaged Stateであり、検索候補・再表示用ヒントのようなDerivedDataだけをDisposableにする。発話・認識に曖昧性があれば候補を提示できるが、確定Operation/副作用へ推測で昇格しない。音声に固有のCodec・タイミングはAdapterの能力であって業務Flowの正本ではない。

### 14.3 再利用する既存OwnerとPlugin Dogfoodの位置

- **#1158／ADR-0048**：Operation ID・Input/Output/Failure/Effectの唯一の共通契約。HookはOperationの観測可能なSlotに対する要求であり、別Dispatcher/新API体系ではない。
- **#951/#1016/#1251/#2110**：Plugin/Implementationの候補・exact選択・Build結合・実Call・世代切替。必要ContractにないHookをPluginが自己追加して既存Host Grantを超えることはできない。
- **#3278/#3279**：SlotにBindingされたImplementationが原子Traitをすべて履行しているか。あるTraitのDefaultをReplaceしてもそのTrait保証を暗黙に減らさない。
- **#1129/#1174/#3274**：Validation/Mutation/Side-effect/HTTP unknown/Receipt。Callbackが何回呼ばれてもよいと暗黙に置かず、重複・再入・取消・不確定効果を区別。
- **#2620/#2090/#943/#2600**：TEIモデル由来のカスタムロジック生成・手動拡張の保持・Source/Generator/BOM・Build失効。Plugin ManifestとSourceは独立Revision。
- **#3271/#1043/#361相当の既存Flow Owner**：Session Scope/Temporal/Derived・State Transitionの責任を維持し、独立Workflow RuntimeをTrait追加の前提にしない。

### 14.4 最小反証パケット：新名称を増やすより実動作を測る（全てNOT RUN）

| Case | 圧力入力 | 独立Expected／Owner |
| --- | --- | --- |
| **TC-34 Slot≠Trait** | `inspect`に結果ObserverをBindingしたが、要求TraitのImplementationが欠落 | Observerが呼べてもRequired Traitは`unfulfilled`。#3278/#3279 |
| **TC-35 Custom Generated** | 同じread-only Operationを標準実装→TEI生成Custom Providerへ明示差替え | 同じ型付き契約と独立Expected。選択Providerの実CallとGenerator/Source/Artifact Revisionを比較。#1251/#2090 |
| **TC-36 Hook conflict** | 同一Slotに適格なReplaceを2件、記述順を反転 | 暗黙last-wins禁止。曖昧解決は失敗、または明示選択した一実装のみ。#951/#1016 |
| **TC-37 Revalidation** | Prepare/Validate後のTransformが業務ID/価格/Scopeを変更 | 変更された対象で再Validate/Authorize/Revision照合がなければExecute拒否。#1129/#1174 |
| **TC-38 Unknown effect** | Retryable + Audited + 書込Hook、Timeout/202でEffect不明 | Receipt照合なしに自動再試行不可。監査回数・Operation Identityを独立に検証。#1174/#3274 |
| **TC-39 VUI resume** | 同一Flow/SessionでInterrupt→Resume、旧State Revisionから二重に承認Intentを送信 | CAS/現在Grant/Business Confirmationなしに2 Commitなし。未確定入力・EventはCache Evictionで喪失させない。#1043/#1125/#3271 |
| **TC-40 Cross-surface** | 同一OperationをCLI／MCP／Web／Desktop／VUI Consumerから呼ぶ | 同一Meaning/Failure/Effect要求、Surface別の現在Grant/実Providerを検証。利用できない媒体はunsupported、VUIだけで新業務Dispatcherを作らない。#1158/#1251 |

**現行の新しい基線**：[main統合PR #3378](https://github.com/hat47x/tei-platform/pull/3378) は、既存**直接Operation**経由のCLI／Studio Inspectorについて六対象と不存在の比較を製品Goテストへ追加した。ただし**D1aのShadow互換**であり、実Plugin Provider Selection/Load/Invocation、Trait Hook、実Wails/TargetのD1bを保証しない。既存Draft [#3381](https://github.com/hat47x/tei-platform/pull/3381)／[#3371](https://github.com/hat47x/tei-platform/pull/3371)は実Host注入口候補で、main統合前・適合未資格として扱う。

**P0で着手する順序**：①#3378のD1a基線を維持し、#1251で実選択Inspector ProviderのCLI/Studio D1bに到達 → ②**read-only結果Observerまたは限定Result Transformの一Slotだけ**にDefault/Manual/TEI生成の差を試す（この順をD1bのHard Dependencyにはしない） → ③#2110のread-only Application拡張1件に同じSlot/Contractを適用 → ④VUI stateful Flow・Write／Undo・Cross-target Buildへ展開。途中の失敗時は、最初の未充足（Source／Binding／実Provider／Hook Invocation／Conformance／Grant）を正確に提示する。

**停止条件**：Slot一覧が巨大な固定enumに膨張する、Traitの単一保証が複数無関係責務を要求する、生成カスタムコードが標準生成物の手修正で維持される、HookがHost Authority/Effect Fence/Owner Meaningを迂回する、VUIのStateをDisposable Cacheと混同する、同名Operation・同じResultを実Provider呼出しの証拠にするなら設計を縮小する。新しいHook DSL、正式Slot ABI、実Go Callbacks、VUI Flow Engine、TC-34〜40、新Target/BOMの実試験は **未採用／NOT RUN**。

## 15. Lifecycle Graphを生成入力にするためのTrait証明義務（2026-11-08、他レーンからの差分）

### 15.1 今回解く一つの不足：Graphの編集を実装差分・検証差分に結びつける

ユーザーが明示した「**Lifecycle全体の共通設計／不変と可変の区分／PluginによるLifecycleそのものの拡張／図表をGenerator入力にする**」を、既存Traitの**実装義務を生成前後に保持する一辺**として具体化する。研究・Graph拡張の詳細は[#3503 Interaction Flow](https://github.com/hat47x/tei-platform/issues/3503)／[Draft #3507 横断Hook・Lifecycle Graph研究](https://github.com/hat47x/tei-platform/pull/3507)、全容図とナビゲーションの製品要件は[Draft #3505 RQ-036](https://github.com/hat47x/tei-platform/pull/3505)、B2C適応の圧力は[#3509](https://github.com/hat47x/tei-platform/issues/3509)へ委譲する。**本節は第二Graph Compiler、第二Source Writer、第二Generator、第二Resolverを作らない**。上記Draftは統合前の資料であり、本節によってその採用・実装を先取りしない。

ここで区別するもの：

| 層 | 位置付け | Trait/実装義務と結ぶ点 |
| --- | --- | --- |
| Raw Source + Accepted Meaning | 保存・意味受理の権限ある入力。Markdown/表/専用DSLを含み得る | Subject/Occurrence/Scope/Revision、意味上の必要Trait |
| **Typed Lifecycle Graph** | 上記を参照する構造化モデル。Node、Edge、多重性、Guard、State/Effect、Hook Slot、Trait Binding、実装Refを型付きで保持 | ノード/エッジ/経路が満たす**原子的Obligation**を解決可能にする |
| **Diagram／Editor** | Graphの可視化と変更提案。レイアウト、色、ナビゲーションは独立 | 編集が実Graph Patchになったのか、単なる表示変更かを区別 |
| Expanded Effective Graph | Base Graph＋選択済Plugin Patchを、exact Contract/Scope/Targetで合成したもの | 全経路で実装義務の成立を静的照合。未解決Patch/Slot/Requiredをpartialに保持 |
| Build／Target Code | 受理済みEffective Graphと選択ImplementationをLoweringした成果物 | Source/Graph/Patch/Provider/Generator revision、実組込、Target-native適合 |
| Runtime／Evidence | 実際のCallback、State Transition、Typed Outcome、Effect/Receipt | 静的Graph成立を**実呼出しや現在Grantの証明へ昇格させない** |

「図をGenerator入力にする」は画像のピクセルや座標をコンパイルすることではない。**Diagramが編集可能なTyped Graphを忠実に操作し、そのGraphを生成入力とする**。表示位置だけの変更ではBusiness Meaningと実行コードの再生成を要求しない。ただしDiagram画像・文書など**レイアウトに依存する表示成果物**は再描画が必要となり得る。一方、Node/Edge/Guard/Hook/Target/Scopeが変われば、Semantic Diffと依存関係に沿った再生成・Conformanceを検討する。GraphからSourceへ戻せない表現はread-only Projectionとして表示する。SourceやMeaningをEditorが勝手に受理したことにしない。

### 15.2 不変なのは「Nodeの固定順」ではなく、すべての実効経路に要求される保証

原子Traitの宣言集合 `T` とLifecycle Graph `G` がそれぞれ存在するだけでは保証は成立しない。対象Operation/Target/Scopeに対して、**選択・合成済みGraphのすべての該当実行経路上に、各Traitの要求を満たす実装と順序・効果条件があるか**を別に検証する必要がある。

- **Path obligation**：例えば書込みが発生する実効経路には、適切なRevision/入力/権限の検査が当該Effectより前に存在する。実行時に検査後Subject/金額/Scopeが変化するなら、必要な再検証を経ずにEffectへ到達できない。公開read-only経路へ一律ログイン画面を要求する、という**固定Node列ではない**。
- **Hook obligation**：`Replace`されたProviderは元Implementationが要求したTraitを失わせてはならない。`Add`された観測Hookに外部送出・書込みがあれば、読取り専用と宣言したGraphのEffect契約と再照合する。
- **Branch／Join obligation**：実Targetで未対応Node、未定義の失敗/timeout/cancel/unknown終端、Join後の欠測Contextや二重Effectがあれば、局所TraitだけがPASSでも合成適合は**unverified／failed under scope**。並列化による順序保証や原子性を推測で追加しない。
- **Authority boundary**：Authorization、Owner Meaning Acceptance、Receiptの真正性を、案件PluginのGuard置換やChat/Voiceの確認発話で発行できない。変更提案と実行許可は別。
- **Bounded proof**：グラフの一部が未Loaded、Dynamic Plugin/LLM分岐が未確定、Source/Provider Scopeが非公開なら、列挙した有限経路だけの局所検査に留める。すべての実行を証明したとはしない。複雑な並列/無限ループ解析は必要になったときのModel Checking Provider候補。

この**経路上の不変条件の検証**は、Trait個々の実装の存在検査とは別の`composition-conformance`の責務である。Traitの合成 `A+B=B+A`（要求集合）は維持しつつ、Execution Pipeline／Graphの順序は自由合成ではなく明示選択・検証する。Graph Patch自体はContract/Providerとして既存#951/#1016で解決し、未選択PluginのPatchが実効Graphへ混入しないようにする。

### 15.3 図表のカスタマイズから実装を生成する一往復

```text
Authoritative Source @ R + Accepted Meaning / Scope
           │
           ▼
Typed Lifecycle Graph G(R)
           │      ← Diagram/View（表示変更は意味変更でない）
           │
   Graph Edit → Typed Patch P(SourceRef, expected R, author, intent)
           │
           ▼
Review / Owner Acceptance / Source write-back（許可された範囲のみ）
           │
           ▼
Plugin Selection / Contract + Trait Obligation Resolution
           │
           ▼
Effective Graph G' = compose(G, selected P) → Graph/Path Validation
           │
           ├── fail → 競合/不適合/不完全性を返し、旧成果物を保持
           ▼
Target Lowering / Generator → exact Build Artifact + Revision
           │
           ▼
Target Consumer actual Invoke → Independent Expected / Effect Evidence
```

Source側はScript-first/Everything is Dataを維持する。Graphの`stable node ID`・Edge occurrence、Source span、業務State Identity、Trait Binding/Implementation IDを保存する。Layout DiffとSemantic Diff、Graph Patchと実行中のState Change、Plugin版とGenerator版、Build成果物と現在Grantは別。**実行中Flowは開始時のGraph/Plugin Lockを参照**し、Graph変更時に途中Stateを新Graphへ勝手に再開しない（旧版継続／明示Migration／停止を区別）。SourceのA→B→AをByte一致の理由で「過去に変更なし」としない。

### 15.4 異質なConsumerへ同じ仕組みを当てて破綻しないか

| Consumer | Graphが扱う主な関係 | 標準にしてはいけない仮定 |
| --- | --- | --- |
| **CLI/Studio Inspector** | read-only Operation→Selected Provider→Result/Diagnostics、限定Observe Hook | Direct Shadow parityだけで選択Providerが実呼出しされたとしない（[main #3378](https://github.com/hat47x/tei-platform/pull/3378)） |
| **Application API／DB** | Function Query/Action→Guard/Effect→Data/Receipt | Query宣言から実DB Read/Commitを推測しない。Schema Applyのunknownを済扱いしない |
| **AI Web Chat** | Turn/Run、Token/Event Stream、Tool候補→承認→Typed Operation、Interrupt/Branch/Resume、Client State | Stream delta＝確定Assistant返答／Tool引数／Effectではない。LLMの提案で任意の新Graph/Grantを自動有効化しない |
| **VUI** | ASR候補→確認済みEvent→Flow/State→Operation→TTS、割込・再開・重複 | 音声認識＝業務Intent確定ではない。Sessionの未同期入力をDisposable Cacheへしない |
| **B2C Journey/Experiment** | Route/Touchpoint→Exposure/Variant→Conversion Action→確定Outcome/計測 | UX・SSR・Hydration・CDN・CreativeをTEI標準Rendererに固定しない。購入完了はUI表示や分析Eventでなく権威あるProvider/Receiptで確かめる |

B2Cでは、**TEIのGraph/型付きActionを使うだけで、実フロントエンドは完全Headless・独自React等とする選択**を正規の経路として扱う。例えばExperiment Graph Patchが「商品の表示／販促文言」を変えても、CheckoutのCurrent Price/Inventory/Grant/Effectの保証は自動的に弱めない。TEIを使わないRouteにJS wrapper/追加network hopを要求しない。B2C固有のConversion Engineering Kitと、効果指標CVR/実運用性能の評価は[#3509](https://github.com/hat47x/tei-platform/issues/3509)に委譲する。

### 15.5 最小反証：旧来のread-only経路を先に通す

次は**新しいScenario Runnerや二重Expectedではない**。各Ownerが既存#1180/#2514のScenarioへ追加すべき**独立Expectedの圧力入力候補**である。

| ID | 操作・差分 | 必須観測（未実行） | 主Owner |
| --- | --- | --- | --- |
| **TLG-01 Layout vs meaning** | Graph Nodeの位置だけ変更、次にEdge/Guardを一か所変更 | 前者はSemantic/Runtime Code入力が同一（Diagram表示成果物だけ更新し得る）、後者は型付きGraphと該当依存を変更・再検証。無関係Targetの無条件再Buildや全証拠失効は避ける | #1124/#2497/#2600 |
| **TLG-02 Non-reversible diagram** | 表示図から多重Edge ID/Source Rangeを落とす | 一見同じ図でも書戻し不可を明示。Raw保全、正式Source Patch拒否 | #1124/#2620/#2090 |
| **TLG-03 Bypass path** | Plugin PatchでAuthorize／Revision Guardを避けるEffectへ分岐 | パス保証の不履行を検出して生成/利用候補を拒否。Plugin Trait個別PASSでも合成FAIL | #3278/#1174/#3503 |
| **TLG-04 Wrong provider** | selected B、実Invocation A、結果Bytes同一 | 最初の不足点`actual-provider-mismatch`を示す。GraphでBが描けても合格せず | #1251/#951 |
| **TLG-05 Conflicting graph patches** | 同SlotへP1/P2がReplace、Plugin順序逆転 | 暗黙last-wins禁止、安定した曖昧診断。明示解決前に新Buildを資格化しない | #951/#1016/#3503 |
| **TLG-06 In-flight revision** | チャット/音声でFlow中断→Resume前にGraph/Provider版変更 | 旧Lock、現在Grant、明示State Migrationの要否を別判定。重複Effectなし | #3503/#1174 |
| **TLG-07 AI delta/tool** | Tool引数がpartial streaming中にUIが確定値として扱う、または未承認Toolを呼ぶ | 不完全なdeltaは候補として保持、Operation 0回。確定後も現在Grantが別途必要 | #3503/#1158 |
| **TLG-08 B2C escape** | 専用Frontendの1 Sectionだけ標準UIへ変更し戻す／Experiment Patchを解除 | Operation Meaning同一、変更した対象Target/Artifactのみ再生成・検証。購買のEffectと露出Eventは別 | #3509/#997/#1121 |

**実装順**：**P0**は[main #3378](https://github.com/hat47x/tei-platform/pull/3378)のInspector直接経路を保持し、#1251の**通常CLI/Studio選択済Provider実Call**を先に確立する。**P0+**として上記同一Operationに`Observe`一SlotをBindingし、Graph上の追加差分／Adapter／実呼出しを単一Targetで検証する（D1の新Hard Gateにしない）。**P1**で一つの受理済みread-only FunctionのGraph Node/Edge 1変更→Source Patch→Generator/Conformanceへ接続し、`G`と`G'`のCounterexampleを固定。**P2**のChat/VUI/B2Cはそれぞれ固有Provider/State/UXを持つ後続圧力試験で、最初から全部をCoreに標準実装しない。

**採否基準**：Graphを経由することで、Trait/Hookの履行漏れ・未到達・安全でないバイパス・生成対象の過不足・モデル変更時のSource/Artifact資格を、直接手書き実装より**具体的に早く／正確に**検出できること。図表の充実だけでGenerator・Product実行・Owner Meaning Acceptanceが成立したと報告しない。Graphの書戻しが原Source/Meaningを破壊する、別Runtime/Hostを必要とする、P0の実Provider接続を妨げる、B2Cの自由なFrontendへ負担を強制するなら対象機能をread-only ProjectionまたはPlugin Profileへ縮小する。

**状況・責務**：本節は[#3503](https://github.com/hat47x/tei-platform/issues/3503)／[Draft #3507](https://github.com/hat47x/tei-platform/pull/3507)、[Draft #3505](https://github.com/hat47x/tei-platform/pull/3505)、[#3509](https://github.com/hat47x/tei-platform/issues/3509)の意図と[#3278](https://github.com/hat47x/tei-platform/issues/3278)のTrait履行を接続する設計段階。正式Trait Graph Schema、Source writer、Graph generator、実Host Callback、AI Chat/VUI実Target、B2C計測、TLG-01〜08は**NOT RUN／NOT ACCEPTED**。これらを本節で採用済み製品機能へ昇格させない。

## 16. DerivedData・高度チューニングを原子的Traitで拘束する（2026-11-08横断）

### 16.1 付与したTraitから、実際に何が強制されるか

最新のCache／DerivedData／Function Data Context／高度チューニングの議論では、**性能のヒントと、正しさの保証と、物理保存の保証をTrait一つへ押し込めない**ことが明確になった。既存[#3271 Cache/Snapshot](https://github.com/hat47x/tei-platform/issues/3271)は保存/再利用、[#2582 Data Context Reactive](https://github.com/hat47x/tei-platform/issues/2582)は依存と再評価、[#2387 SQL演算依存](https://github.com/hat47x/tei-platform/issues/2387)は演算意味、[#2600](https://github.com/hat47x/tei-platform/issues/2600)はBuild再利用、[Pluginの安全な最適化計画](PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md)は計測と方式選定のOwnerである。本節は**これらをTraitの原子契約として要求・実装・検査する判断だけ**を扱う。

| 宣言の性質 | 具体的に拘束する実装・検証 | 勝手に含めない保証 |
| --- | --- | --- |
| `Derivable`（原子Trait候補） | 型付きの導出Ruleと必要Input/Operator版の解決、欠落時の明示診断 | 実Sourceがいつでも取得可能、再計算結果の業務意味が受理済み |
| `Rebuildable` | 指定Snapshot/入力閉包・版・Time/Scope・非決定要因を満たしたときの再構成、再構成不可状態の説明 | 失われた外部観測／承認／LLM出力を無条件に再生成できる |
| `Disposable` | **対象をMaterialization identityに限定**し、削除しても正本・Temporal Record/未同期入力を消さない | オフラインでの利用可能性、署名付き確定値の破棄 |
| `Materialized`／`Cached` | 選択Providerで書込み／読取りと失効・不整合検出が可能。**必須永続保存かbest-effort cacheか**をOperation/Policyで明示 | 保存成功＝永続性／Freshness、Store不可＝元計算Operation失敗 |
| `Scoped` | User/Tenant/Session/Principal/取得ScopeとCurrent Grantを実使用条件へBinding | Scope文字列のみで認可、同じBrowser Origin＝同じUser |
| `IncrementalMaintenance`（候補） | **宣言したDelta種別・Query演算・Snapshot前提について**独立Expectedと差分適用が一致するImplementationを要求 | `DependencyTracked`だけで集計・NOT EXISTS・Top-N等を安全に差分適用できる |

ここに挙げた名前は**候補ラベル**であり、新Core固定Enum・正式DSLを確定しない。便利な`DerivedData`などはProfileとして原子Traitへ展開可能だが、Profile一つで`Disposable`／`Retained`の実体を混同しない。Traitの実装方式はDefault／Generated／Provider／Manual Requiredがあり、**手動実装を必要とする場合もRequiredを検証する**。単なる`@hint(cheap_to_rebuild)`はImpl選択に使えてもRequired Obligationを満たした証拠にしない。

特に`Cached`を必須Traitとする際、**キャッシュ利用を試みる義務**と**保存実体が利用中ずっと存在するという保証**を区別する。前者はbest-effort保存＋失敗時計算Fallbackで成立し得るが、後者は対応Resourceの持続性・容量・可用性条件に裏付けられない限りRequiredとして成立しない。ブラウザーのLocalStorage／IndexedDBの強制削除を、保存保証違反でなく通常のcache missとして扱える場合と、オフライン必須データ欠落で機能要求に不適合となる場合を区別する。

### 16.2 増分計算は「依存関係が分かる」より強いTrait義務

`IncrementalMaintenance`を宣言したとき、**何に対して**増分が正しいかを束縛する必要がある。最小適用範囲候補は
`(query/operator revision, result shape, supported delta kinds [insert|update|delete|retract|upsert|late-correction], snapshot/consistency, time/window, source scope, target/provider)`。

- `LEFT JOIN`：右辺が新たに一致すると、従来の未一致`(left,NULL)`は**撤回**され得る。行追加だけのmergeでは正しくない。
- `NOT EXISTS`：否定条件を満たす行が追加されると、**元の結果行が消える**。存在しない依存（negative dependency）の監視・完全取得証拠が必要。
- `Top-N`／`ORDER BY LIMIT`：新しい上位行により既存境界行が押し出され、ソートキー更新でも順位が変わる。必要な候補集合/再評価Windowが不明なら安全なDeltaを宣言できない。
- `SUM`等の集計：削除・訂正・遅着、NULL、重複・JOIN多重度、精度・丸めとSnapshot境界に応じて補償（retract）が必要。
- 時間Window：Event timeとObserved/Recorded time、水位線（watermark）、遅着・訂正を区別。再計算した過去有効時点の値で「当時観測・承認した値」を上書きしない。

**決定規則**：Input ClosureとPredicate/否定依存を含む完全性・Delta適用範囲・実Providerの能力が確認できるときだけ増分更新を選択する。前提が欠けたら、(a) 要求Contractが許す範囲では**全量再読込・安全な再計算**にFallbackして、その方式をResult/Evidenceに記録する、(b) 差分計算そのものが厳格に必須で代替不可なら`unsupported/unfulfilled`として拒否する、(c) 正本も取得できなければ`unknown/unavailable`として保持する。**全量Fallbackを実施した実行を「増分更新が成功」と表示しない**。

### 16.3 性能の最適化と正しさの保証を別の検証列にする

```text
Business/Function freshness + availability + retention requirements
                          │
Required Atomic Trait / Operation Contract
                          │
Existing Provider/Resource Capability + exact Selection
                          │
Complete dependency/negative dependency + snapshot/watermark
                          │
Correctness gate (full recompute / independent Expected)
                          │
Optional optimization selection (incremental / cache / batch / placement)
                          │
Same-target same-workload measurement + current Grant
                          │
Target-specific Adoption/Reject/Rollback record
```

性能改善の提案は`advisory`で構わないが、採用した**実装の正しさ・失効条件・必要Effect**はRequiredで検証する。例えば「推奨：IndexedDB」「推奨：差分更新」と表示されても、実Resource選択/Query Delta/Current Grantを自動的に資格化しない。強いFreshnessを業務要件として宣言した場合、`T_detect + T_queue + T_derive + T_validate + T_publish`のBudgetと実Clock/Source Observationを確認する。**TTLだけでは同時点Snapshot／最新値／SLA充足を証明しない**。過去の結果を「参考として表示」してもよい条件と、その結果を使う業務Actionの許可は別。

Browser Memory／localStorage／IndexedDB／Redis／Local Fileは同じ抽象的`Resource Instance`から利用できても、容量・同期/非同期・TTL/atomic write/CAS・権限・持続性・競合の能力は別。複数Tab/Workerが旧結果を遅延書込みして新しい世代を上書きすることを禁止し、選択ProviderがCASを保証しないならepoch／Revisionを正しく隔離する別実装・Fallbackを要求する。`Cached + Shared`の同一値KeyはTenant/Principal/Source Scope/検索条件/Rule版/現在の開示資格を省略してよい理由にならない。事実の保存義務・退避/削除方針と最適化用Materializationのevictionは別Storage Subjectに束縛する。

**チューニング計測の最小受渡し**（計測主体は既存Optimization Owner）：対象のContract/Target/Provider/Lock・Workload・Source Input／Rule Fingerprint／Snapshot条件・Cold/Warm/Hit/Miss／Rebuild/Refresh/Purge時間・p50/p95／CPU/Memory/容量／Freshness lag／unsupported/fallback件数／誤失効と過剰失効／コスト。原生QueryやSecret/Personal Dataを無制限にログへ記録しない。**契約適合PASS、機能の可用性、性能改善PASSは別票**とし、未計測Budgetは`unknown`。新しい統計収集エンジンはTrait導入の前提にせず、既存Provider/Traceで最低限の計測を実装可能かを先に確認する。

### 16.4 独立ExpectedのPressure：最初の一Providerへ還元する（全て設計・未実行）

下記は[#2582 RC-01〜09](https://github.com/hat47x/tei-platform/issues/2582)、[#3271 CS-01〜10](https://github.com/hat47x/tei-platform/issues/3271)の**Trait Fulfillmentとしての追加観点**であり、第二のScenario/Expected正本を作らない。

| ID | 差分・故障注入 | Trait側で拒否／区別するべきこと |
| --- | --- | --- |
| **TCO-01** | `LEFT JOIN`の未一致結果へ右辺一致行を挿入。Providerはinsert-only deltaしか宣言しない | `IncrementalMaintenance`の要求に必要なretractが欠落。旧未一致行を残したまま`conformant`としない |
| **TCO-02** | `NOT EXISTS`の否定依存を一つ未Loadedにした状態で結果をcache hit | Required Input Closureがpartial。真の0件/不存在、cached freshness、差分更新完了を宣言しない |
| **TCO-03** | Top-Nの境界行変更・集計への削除/遅着訂正 | 実施可能なDeltaの対象範囲だけ資格を与える。Unsupported時は宣言したfull-refresh fallbackまたは拒否 |
| **TCO-04** | Browserで最新世代Bの生成後、遅延した旧世代Aの保存、途中QuotaExceeded/Storage eviction | stale-overwriteなし。保存失敗と正本からの計算成功を別。オフラインAvailability保証を捏造しない |
| **TCO-05** | Tenant/Grant/Source Scope変更、結果ByteとCache Keyが同一 | 現在の開示不可。事実上の存在・件数・KeyをUnauthorizedへ漏らさない |
| **TCO-06** | 原価格に遡及訂正。再計算可能な売上集計と当時確定した受注記録を同じ見た目で保存 | 集計だけ失効・再生成、確定Temporal Record/Owner acceptanceは維持 |
| **TCO-07** | 同Provider/Rule表示Versionで実Rule/Operator/GeneratorのByteだけ変更、結果値同一 | 旧Trait Conformance/Build Evidenceの再資格が必要な範囲を提示。同Byte/Cache hitで旧Qualification復活なし |
| **TCO-08** | `@hint(incremental_candidate)`のみ、Manual Required不足、または実Providerがfull refreshで対応 | HintはRequiredを満たさない。Fallbackは`full-refresh`として観測。増分方式必須なら不適合 |
| **TCO-09** | 同じDatasetを再読込・安全なdelta・高頻度cacheで比較 | 独立Expected一致を先に検査、実Target/負荷のLatency/Memory/Source lag/容量/コストを別評価。改善未計測なら採用しない |

**実施順**：
- **P0を妨げない**：#1251のInspector D1b（CLI/Studio実選択Provider呼出し）と#3279 Manual Required欠落診断を先行する。TCO試験のために第二Cache Engineや第二SQL Query Plannerを構築しない。
- **P0-C（read-only Fixture）**：#2582で`LEFT JOIN`／`NOT EXISTS`／Top-Nの既存演算依存とSource Scopeを一例ずつ確定し、#3278で`required`と`hint`の違い・unsupported/fallbackの診断だけを既存Resolver/CLI read-only結果へ追加できるか評価。
- **P1（実Backend/Storage）**：#3271のMemory/Local/Browser相当の一Provider上で、Source→Derive→Cache→Invalidate→Rebuild／Fallbackと、Temporal保存義務との非干渉を**独立Expected**と実Targetで比較。Browser実媒体とGo headlessは異なる資格。
- **P2（測定・最適化）**：最低限の再取得/full recomputeと最適化候補を同Target/Loadで比較して効果が出た場合にのみ特性・Profileの標準提供範囲を決める。外部Redis/Edge/Streaming/B2Cへの展開をP0の必須条件にしない。

**停止条件**：Traitを増やしても標準実装・手動不足診断・合成適合・最適化効果のいずれにも具体的改善がない、演算の非単調性を依存宣言で過少評価、退避すべき確定値をDisposableとして削除、測定未実施を性能改善と呼ぶ、独立的なCache Authority/Planner/Resolverを導入する場合は、Trait化せず既存Query/Resource/Policyの表現へ戻す。

**今回の資格**：他レーンの既存#2582/#3271/Optimization契約と#3278/#3279の照合・受入候補の追加だけ。正式Incremental Trait ID/DSL、Native Browser Provider、実Cold/Warm Bench、TCO-01〜09、実Go/Studio/SQL/DB、Owner Meaning Acceptanceは**NOT RUN／NOT ACCEPTED**。

## 17. AI性能チューニングの観測を原子的Traitの履行へ接続する（2026-11-08横断・受入案）

### 17.1 今回確認した現行main：合成Referenceの成功と実Providerの資格を分ける

[AI性能チューニング設計§9〜10](AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md)と[main統合PR #3520](https://github.com/hat47x/tei-platform/pull/3520)により、**設計上のHint／合成Call Observation／統計Window／Provider選択・呼出し申告／開示・分割受領を、同じ条件で照合する限定Reference**が存在する。同PRの`synthetic-controls`はGitHub Checkでsuccessと確認できた。Referenceは19件の独立合成Expectedを保持するが、実TEI Source、実DB Query、実Provider、Host現在Grant、実Tool配送、AI改善、実性能計測を実行していない。この範囲を**「input-consistent／調査候補」**と呼び、Runtime Trait Fulfillment／BOM Qualified／Performance Improvementへ昇格させない。#3520の同HEADでは別のroot `govulncheck` checkにfailureがあり、**Repository全体が無条件にPASSしたという主張はしない**。

また[main #3371](https://github.com/hat47x/tei-platform/pull/3371)のStudio Host型付きInspectorに加え、[main #3381](https://github.com/hat47x/tei-platform/pull/3381)では、既存Composition選択BとHost登録Bを結ぶ**局所Go実Callback**、CLI helper／Studio Host APIの**二つの局所Call-siteから六Subject計12回の選択済Provider呼出し**を検査するコードが統合された。ただし通常のCLI DispatcherはBuiltin経路、生成Studio Hostは暫定Builtinを明示登録しており、**公開CLI／実Wails Bridgeの本番入口S2と実Plugin Artifact/BOM資格は未成立**。[main #3516](https://github.com/hat47x/tei-platform/pull/3516)は**実Go HTTP Host→既存Driver→合成Adapter Invoke**の限定経路を実証したが、実業務DB Commit、Trait合成BOM、CLI/Studio D1bを証明しない。[main #3479](https://github.com/hat47x/tei-platform/pull/3479)はImported Provider Source-only driftをConsumer Source Indexとは別に検出する実Go Referenceを加えた。**観測できたものの粒度でClaimを止める**。
また[main #3371](https://github.com/hat47x/tei-platform/pull/3371)はStudio Hostの型付きInspector入口を前進させ、後続の[main #3381](https://github.com/hat47x/tei-platform/pull/3381)はexact selected Provider Bを**CLI HelperとStudio Host APIという二つの局所Go Call-site**から呼ぶ試験コードを統合した。一方、**通常の`tei inspect`ディスパッチと実Wails `StudioBridge`からの同一Provider実Callbackは未資格**のまま（現行状態は§18）。[main #3516](https://github.com/hat47x/tei-platform/pull/3516)は**実Go HTTP Host→既存Driver→合成Adapter Invoke**の限定経路を実証したが、実業務DB Commit、Trait合成BOM、CLI/Studio D1bを証明しない。[main #3479](https://github.com/hat47x/tei-platform/pull/3479)はImported Provider Source-only driftをConsumer Source Indexとは別に検出する実Go Referenceを加えた。**観測できたものの粒度でClaimを止める**。

### 17.2 Performance Profile、Trait、Hint、Instrumentationの役割分担

最適化用の`Interactive`／`ReadHeavy`／`WriteHeavy`等の**Performance Profileは「典型的なWorkload／Budget／実装選択制約」の候補集約**であり、単独で `Fast`、`LowMemory`、`Fresh` という**実測済み保証や原子Traitにはしない**。ProfileはOwnerが定義したNFR、Target/Resource制約、必要な原子Trait、Advisory Hintに展開する。RuntimeやUIの既定実装も無条件に固定しない。

| 宣言層 | 宣言によって必ず起こること | 必ずしも発生しないこと |
| --- | --- | --- |
| **Required atomic Trait** | そのSubject/Scope/Targetでの必要Operation/Invariantと、Default/Generated/Provider/Manualのいずれかによる履行または未実装診断 | 生のTrace、特定Profiler、既定の外部Collector、本番Grant |
| **Required measurement contract（候補）** | 指定Operation/Window/計測範囲で、必要な観測点・単位・Sampling/欠測・収集失敗を実装/検証する。手動実装を要求してもよい | 遅延ゼロ、全件Trace、全環境での同一p99、計測値の正しさまでの自動証明 |
| **Advisory optimization Hint** | 候補生成・選択の際に使われたか／無視されたかを説明できる | Optimizerの自動実行、対象のTrait Requiredを満たすこと |
| **Performance Profile** | 必要なNFR/候補Trait/Hintの解決と、各不足の診断。対象Workload/Targetを明示 | プロファイル名から一律の物理キャッシュ、DB Index、Batch、SSR、Telemetry常駐を強制 |
| **Observed measurement** | 取得Source/Window/Attempt/Scopeと実観測または合成申告の資格を明示 | SLO達成、改善率、Source Meaning Acceptance、今の権限 |

Traitに`Observable`という曖昧な大型保証を追加するよりも、**何を誰がどの範囲で観測する義務か**を既存Operation/Provider/ScenarioのContractに束縛する。**計測機能の未設置でも動く対象**と、SLO検収などのため**計測をRequiredにした対象**を区別する。必須Telemetryが未提供の場合は観測義務の`unfulfilled/unknown`であり、業務Operationを停止すべきかどうかは別の適用Policyで判断する。機密漏洩の危険や独立測定手段がない場合に、Profilerを黙って有効化しない。

### 17.3 AIの最適化提案はGraph PatchやPlugin選択に戻すが、証明を飛ばさない

```text
Typed Source + Required Traits + Scope / Accepted Meaning
             │
 Design Hint / existing Workload Profile
             │
 Real Observation? ─── Synthetic Reference (input-consistent only)
             │                              │
             ▼                              └→ investigate / unknown
 Optimization Proposal（AIが提示する候補）
             │  Target/Effect/Cost/Expected/Hook Slot
             ▼
 Existing Provider Selection / Effective Lifecycle Graph / Generator
             │  Source/Provider/Rule Rev、Graph Patch、Target exact Lock
             ▼
 Independent Correctness & Effect Conformance
             │
 Same Workload Benchmark + Measurement overhead
             │
 Owner/Host-approved apply → Actual Invoke / Receipt
             │
 Post-change Observation / rollback qualification
```

AIはSource内のCardinality、Relation、Query Sort/Limit、Nullable、Negative Dependency、Data Context epoch、Cached/DerivedのRebuildability、Provider/Hookの適用箇所から**設計上のヒント**を抽出できる。一方、実行回数・Rows・Pool滞留・Latency・DB計画は**測定された期間・範囲にのみ帰属する**。`queryReadCalls>=3`のような合成条件は「反復調査候補」であり、N+1と確定したことやBatchで改善した証拠ではない。Agentが生成するOptimization Proposalを直接Provider Override/DB DDL/Runtime Grantとして実行しない。

**Graphとの結合**：Performance Observer、Analyzer、Strategy、Verifierは新Core enumでなく既存Plugin/Functionの実装Role候補。OptimizerがLifecycle Graphの許容Hook SlotへPatchを出す場合は§15の**全経路Trait Obligation／Effect Boundary／再検証／Target Lowering**を経る。性能Observerを追加したことで新たな外部送信・Storage Effectが生じる場合、そのEffectも明示し、既存のread-only保証、秘密保護、現在Grantと衝突していないか評価する。**Hookを入れた＝実観測した、Graphで選んだ＝Providerを呼んだ、前後二つの数値＝改善した**、の三つを拒否する。

**Evidenceの失効を三重に判定**：Consumer managed Source/World変更、Imported Provider raw/Manifest変更（#3479）、Capability Rule/選択実装変更（main #3490）を別々に観測し、実依存閉包に影響するTrait Binding・Benchmark・BOMの行だけ再資格化する。同じ表示Version、出力byte、過去のCache hit、同一World fingerprint、A→B→Aから旧計測やCurrent Disclosureを復活させない。部分的なDesign Context・欠落したTrace・未受領のTool Reply・不一致Histogramから全Workloadのp99/SLOを確定しない。

### 17.4 追加の受入圧力：既存AIチューニングとTraitテストの共通照合列

次表は[AI性能チューニングAT-01〜14](AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md)と§16 TCO-01〜09、[#3279 TR-01〜11](https://github.com/hat47x/tei-platform/issues/3279)の**結合点**であり、独立Expectedの第二正本・新Test Runnerではない。実行はそれぞれ既存#1180/#2514に委譲する。

| ID | 一要因だけ変える | Trait／実装／性能の期待する切り分け |
| --- | --- | --- |
| **TAT-01** | 計測をAdvisory→Requiredに変更、Observerは未配置 | 業務Operationが動いても**計測の実装義務は未充足**。Hintのみなら必須機能の欠落としない |
| **TAT-02** | 合成Referenceで`input-consistent`かつ19件の合成ExpectedをPASS | 「調査候補」のみ。実Provider Invocation/本番Telemetry/SLO/BOM資格には昇格不可 |
| **TAT-03** | 同じOperation/Queryの低Cardinality summary、Sampling/Histogram境界だけ変更 | 両期間のp99・改善率の機械比較は不可。Sample Coverageと欠測範囲を説明 |
| **TAT-04** | 性能ObserverのHookがRaw SQL/Principalを外部送信する、またはTargetで未呼出し | 意図しないEffect／Disclosure違反、Actual Invoke欠落を拒否。Observer選択だけでTrait履行としない |
| **TAT-05** | AI提案によりQuery Pushdown/Batch/Cache Placementを変更、Result値は同じ | NULL/粒度/順序/Grant/Effect/Sourceと独立Expectedを照合し、同条件のp50/p95/費用を別評価。性能未測定なら改善未確認 |
| **TAT-06** | Source Index不変でImported Provider rawのみ変更／Capability Selection Ruleのみ変更 | 影響したProvider/Rule/Trait Evidence・Benchmarkの資格を保留。無関係Scopeの全Artifact失効は避ける |
| **TAT-07** | 計測有効化でCPU/Memory/p95退行、またはCollectorが停止 | 実測した計測費を記録し、必須契約に従った限定縮退/拒否。業務のEffect結果をTelemetry障害により捏造しない |
| **TAT-08** | ReadHeavy ProfileをBrowser/Go static Targetへ合成し、一方はStorage非対応 | 不足する必要Trait／ResourceをTarget別に診断。Profile名だけでCache必須／永続性保証とはしない |

**次の実証一辺（P0維持）**：まず#1251/#3381の**通常CLI Dispatcherとmain #3371 Studio Host**にexact-selected Inspector Providerの実Invocationを接続し、既存#3279のManual Required不足診断を同じ読み取り系で示す。並行して#2586の`order_entry.recent_orders`一Queryを実Provider Readへ接続し、#3520合成Referenceの入力列を**実Call/Scope/Window**から構成できるか検証する。次に#3271/#2582の安全なfull-refresh baselineと一つの最適化候補を独立Expected・同Target/負荷で比較する。#3503のGraph/Chat/VUIや#3509のB2Cは後続Consumerとして圧力検証し、P0のHard Dependencyにしない。

**停止線**：19合成テストの成功を実性能Evidenceと誤表示する、新たな常駐計測Host/Collector／中心Optimizer／第二Resolverを必須化する、Profile名から無条件にCore Traitを追加する、Scope/Targetの欠落をperformance PASSで覆う、Profiler/Telemetryの有害EffectをObserverという名称で隠す、実Provider/独立ExpectedがないのにAI改善を適用する場合は本設計を縮小する。

**現時点の到達範囲**：main #3520の合成Referenceと局所CI、main #3371/#3516/#3479/#3490のそれぞれ限定成果を踏まえた設計・受入案。TAT-01〜08、実DB/Provider計測、実CLI/Studio二Consumer、実Benchmark、AI提案の適用・Rollback、正式Profile／Measurement Trait Schema／Owner意味受理は**NOT RUN / NOT ACCEPTED**。

## 18. D1局所二Consumerのmain実装を、Traitの製品Fulfillmentへつなぐ（2026-11-08：現物に基づく計画収束）

### 18.1 最新の実装基線を限定して認定する

[main統合 #3381](https://github.com/hat47x/tei-platform/pull/3381)（`f8896886c4528920a0d0cb3a2f67181971402987`）と[状況の明示 #3531](https://github.com/hat47x/tei-platform/pull/3531)、[Plugin計画 #3525](https://github.com/hat47x/tei-platform/pull/3525)を踏まえる。従前§12〜17に残る「#3381／#3371はDraft」という文言は**当時の観測**であり、この節を現行資格の優先表示とする。新しいTrait制度を先に定義するのではなく、既にmainへ存在するGo Call-siteの**最初の未接続辺**を優先する。

| 達成した限定事実 | 現物の位置 | 成立していない主張 |
| --- | --- | --- |
| **S0** Typed Inspector Injectionと共通Projection Validation | `cmd/tei/cli_inspect.go`、`internal/operations/inspect_projection.go`、`internal/studiohost/readonly.go` | 既定公開CLIとStudioが選択Pluginを使っている |
| **S1** RegistryでBを選択→Host登録Bの実Go関数を呼び、A誤登録・Epoch/Grant失効・誤Resultを拒否 | `internal/inspectoradapter/selected.go`＋`cmd/tei/cli_inspect_studio_selected_parity_test.go`／対応Go Test | 合成Artifact Digestが実BinaryのByteと独立に結ばれたこと、実外部Pluginのロード |
| **S1の二局所Consumer** | 同じBをCLI helperとStudio Host APIから六Subjectごと二回ずつ呼ぶテスト経路 | **通常起動の`tei inspect`と実`StudioBridge.InspectSubject`の二つの公開入口**をBへ切り替えたこと |
| **S2未成立** | `cmd/tei/main.go`は`inspectWorkspaceJSON`→`builtinCLIInspectionService`、`cmd/tei-studio/host_generated.go`は`selectedBuiltinInspector`を明示登録 | Productの実選択Plugin回路、実Wails Consumer、全Required Trait履行 |
| **S3未成立** | 実Source/Manifest/Rule/Build Artifact/Host Authority/Independent Expected/Consumer Receiptを必要範囲で照合する将来の入口 | Qualified Composition BOM、全Target/現在開示の認可、業務意味受理 |

**CIを一括PASSとしない**：PR #3381の統合HEADのチェックでは多数のビルド／試験ジョブがsuccessだった一方、`repository-health`はfailure、[#3531](https://github.com/hat47x/tei-platform/pull/3531)統合HEADでも同Checkがfailureである。独立のGo Test／Buildの成功は正の局所証拠だが、Repository全体Healthを合格扱いしない。失敗箇所・CI対象版を切り分け、製品S2到達とは別に扱う。

### 18.2 次に実装する一辺は「公開入口→Host選択B→実Entry」であり、Trait拡張ではない

```text
Operation "inspect" (CLI / Studio)
            ↓ 各公開入口でWorkspace Root/Consumer/Targetを決定
Host-owned existing Composition (selected Contract + B + exact Scope/epoch)
            ↓ 実Build/Host登録Entry・Source/Manifest Binding照合
Existing inspectoradapter.Selected / typed Inspector
            ↓ 実Entry B を呼び出したWitnessを取得
Typed Result / classified Failure / Disclosure
            ↓ 独立ExpectedとCurrent Grantを別判定
Required Trait Fulfillment（この範囲のみ）
```

**P0の受入（#1251/#1824/#951、既存D1を使う）**：通常CLI Binaryの`inspect`と実Wails Studio Bridgeの`InspectSubject`から、同じScope/Contractで選ばれたBに**別々の実Callback**を発生させる。選択B→登録A（同じJSONを返す）、選択後Manifest/Rule変更、旧Epoch/Guard撤回、Source未取得/部分取得、Provider Errorに埋めた秘密でfail-closedを確認する。実選択Bが利用不能な時に旧Builtinへ暗黙fallbackしない。局所Go二Consumerの「12呼び出し」だけでは、公開入口の受入を代用しない。既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)のExpectedを使い、第二のScenario正本を増やさない。

**P0-Fの受入（#3278/#3279、P0の補助的・独立課題）**：その一つの`inspect` Operationと選択済Providerに必要な**一原子Trait相当のOperation/Invariant**を既存Contractから抽出し、`provided`と`manual-required`の二履行経路を比較する。手動が未実装ならSource span／対象Subject／必要Operation／Targetを持つ`unfulfilled`を表示する。実装追加後も**署名一致・Registry選択・局所Call・Target Conformance**を別判定する。`Hint`やProfileはRequired不足を埋めない。実装義務は何に付与されたのか（Data、Provider、Operation、Flow）とScopeを保存する。**独立Trait Resolver、Kernel固定Trait ID、第二Test Runnerは実装しない**。

**P1の受入（既存Ownerへ）**：一つのread-only Observer Hookを選択後、`effective Lifecycle Graph→生成/実配線→Observed Path`と比較する。Hookが要求するTraitの充足と、Hookを差し込んだことで増えたログ／送出Effectの資格を分ける。[RQ-036全体マップ #3505](https://github.com/hat47x/tei-platform/pull/3505)には、固定保証、案件可変Slot、実装Requiredの状態、Source/Implementation/Outcomeの有効な参照と**unknown/unverified**をread-only表示する。図上の色や線の成功で、実B呼出しや手動実装適合を証明しない。

**P2の受入（異質Consumer）**：一Function・一Query／実Data Provider→Rowset（#2586）、DerivedData再生成（#3271）、AI Observer統計（#3520）、B2C Headless／Conversation/VUI（#3503/#3509）、BOM 2×2（#3267）は、S2を阻害しない独立の後続実証であり、Scope/Target/Effectごとの証拠に限定する。

### 18.3 「実装不足を検知する」ことの独立Expected

| 引渡し先の既存試験 | 要求する最小対照 | 拒否すべき過大な主張 |
| --- | --- | --- |
| WIT-01/02、TC-11 | 既定公開入口からB選択→B実Invoke、対照としてB選択→A登録（同Result） | 局所テストのB実Callや共通JSON同値＝Product S2 |
| TC-01/02、TRF系 | `manual-required`未実装→実装後Conformance、同TraitのProvidedとの対照 | 型・メソッド・Provider自己申告＝保証の証明 |
| TC-23／BFR-12 | Required Closureがpartial／未Loaded／非開示 | 観測済みRequiredの充足率＝全Requiredの完全性 |
| TC-37／TLG-03 | Hook/Graph PatchでEffect Guardをバイパス | 各Trait単体PASS＝有効Lifecycle全経路の適合 |
| TAT-02/04／WIT-08 | 合成AI Observationが整合するが実ProviderやObserverは未呼出 | 合成Reference success＝実統計／性能改善／Trait計測義務のFulfillment |

**判定点を固定**：新しいTraitの名前・適用箇所が増えたことではなく、(1) 旧実装と比べて必要なImplementation不足を確実に検知、(2) 原子Traitの独立追加にCore変更不要、(3) 複数Traitの要求合成が順序非依存、(4) 選択Providerと実Callの同一性・Source資格を証拠で説明、(5) 異Targetでも同じRequired/Failure/Effect意味を維持、(6) 開発者と生成AIが未充足の**最初の一辺**を取得できる、ことを測る。達成できなければTraitの独立抽象を減らし、既存Contract/OperationのRequired Constraintへ縮小する。

**境界**：§18はGitHubの現在mainの製品実装とCI結果に基づく**計画の現行化**。通常CLI/Wails Product S2実Call、実BinaryのProvider Artifact真正性、Manual Trait連動、P1 Graph/Hook、実Query/Browser性能、独立の横断E2E、Owner Meaning Acceptanceは**NOT RUN／NOT ACCEPTED**。既存#1119のHard Gateは増設せず、Product Scopeとローカル合成テストの成果を区別する。

### 18.4 RQ-036全体マップへの実装義務の投影：可視化で保証の母数を捏造しない（2026-11-08追補）

現行mainの[RQ-036](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)は、利用者にドメインの全容・標準と変更可能箇所・イベント／状態／Function／Providerの関係を示す**採用済み製品要求**である。[Typed Model Map Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)は、`internal/operations/model_map.go`に**read-onlyの部分的Graph Projection**を追加する未統合候補であり、`coverage: partial-projection`が示すように、まだModel全体の存在／Trait Requiredの完全取得／実Provider履行を証明しない。DraftのNode/Edgeは表示可能な観測に限る。**TraitをMapに表示できることと、モデル要素にTraitが正式にBoundされていることを分離**する。

本件で新たに決めるのは、MapのRendererや第二Registryではなく、**既存のRequired/Fulfillment/Source/Conformanceをどの条件でナビゲーションへ投影してよいか**である。

| 投影する関係（実装時の候補） | 必要な識別・資格 | 未資格なら表示すること |
| --- | --- | --- |
| Subject→Source/Role/Facet | Subject Kind+Canonical ID、Source occurrence/Revision、Candidate/Acceptedの区別 | 未読込・部分・意味未受理を明示。単なる名前の一致で意味受理としない |
| Subject/Operation→Required Trait | Operation Variant、Target/Scope、Trait Contract Rev、**Required Closureの取得完全性** | 未Loaded/unknown/permission-deniedを「Traitなし／Requiredゼロ」にしない |
| Required→Selected Implementation | 既存#951/#1016が選択したProvider/実装Ref、exact Contract/Binding/Scope | 未選択、競合、unsupported、manual-requiredを分ける。選択＝実呼出しとはしない |
| Implementation→Actual Consumer/Conformance | CLI/Studio/Runtime別のHost実Call、独立Expected、Target・Artifact・Assessment範囲 | ローカル合成Go試験、登録Function、同じResultだけでは製品Fulfillment未証明 |
| Standard Flow→Patch→Effective Flow | 固定保証／設定可能／明示Hook Slot、選択済みPlugin Patch、受理SourceとのDiff | Layoutだけの差分とGuard/Effect変更を区別。Hook追加で現在権限を生まない |

**Mapの完全性は、表示済みNode数を分母にしない。** ユーザーが辿ろうとしているQuery/Function/Operationに必要なSource/Role/Traitの閉包に対して、取得できた要素、未取得／非開示／未選択／未解決の要素を区別する。閉包の一部が秘匿される場合は、その名称・個数・パス・「何かが存在する」と分かる分類も無権限Consumerへ出さない。閲覧できた範囲の`partial`は表示してよいが、非開示領域の実態について推測しない。Partial Sourceでは既知の`manual-required`不足を報告できても、**全体がFulfilledという結論は保留**する。

**次の最小実証は新しいMap APIではなく既存Consumerへのread-only投影**：
- MTP-01：同じ`sales.order`を既存InspectorとModel Map候補で辿る。Map上の同名ラベル／同じSource FPからRequired Traitを捏造せず、対象Kind+ID+Occurrenceが一致する既存Bindingのみ表示する。
- MTP-02：選択済みInspector Bはin-process Go Callで通過したが通常CLI／実Wailsから未呼出。Mapでは**「局所S1の適合」**と**「製品S2未実証」**を別々に表示し、フローを一本の緑線で確定表現しない。
- MTP-03：Traitを宣言した`*.tei.md`がRaw管理下にあるがLoader未解釈／Roleが曖昧。Source保全はできても、Required 0・Default実装有効・Owner受理としない。
- MTP-04：必要Trait集合の一部が非開示。許されたConsumerへは閲覧範囲の不完全性と権限制限だけを伝え、秘匿対象の存在・件数を返さない。
- MTP-05：Plugin PatchがGuardを迂回、または同じ結果を返す別Providerへ誤配線。Map表示の形式的整合は維持しても、実効Path／Actual Invoke／Trait合成適合を**別の不適合**として示す。

これらは既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expectedに渡す**受入観点・NOT RUN**であり、Draft #3518へ新たな正式Graph Schemaや実行権限を要求しない。実装は#3262（Required Design Context）、#3278/#3279（Trait）、#1251（Inspector S2）、RQ-036/#1124（Map）、#3503（Flow/Graph）の**既存Ownerのread-only成果を再利用する範囲**に限る。

**採否**：開発者・生成AIが「モデル上のどのTraitが未履行か」「そのためにどのProvider/Operation/Targetの何が不足か」「表示情報がどの範囲で不完全か」を追加の正本なしで辿れた場合に、TraitのMap投影を有効な標準機能候補とする。全件解決できないMapで完成図や全体合格を表示するなら、局所Map Projectionへ縮小する。現時点でMTP-01〜05、新たな実Target、全体Model Mapの正式採用、完全なTrait Coverageは**未実証**。

## 19. Trait履行と実行主体の選択：Product S2／外部Host／構成BOMを混同しない（2026-11-09横断）

### 19.1 今回確定する判断の境界

[構成の繋ぎ目に関する判断材料](../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)は、Bindingの宣言・検証・計画と**実行の接続がまだ別**であることを実Sourceと調査で示し、C-1 `routing.Router`、C-2 `servicehost`、C-3 能力ID解決、**C-4：実行を接続せず外部主体に委ねる**選択を併記している。**現時点で採否は未決**であり、Traitの実装義務を理由にC-1/C-2を自動採用しない。C-4も正当な製品構成である。

同時に[Inspector S2接続Draft #3540](https://github.com/hat47x/tei-platform/pull/3540)は通常CLI／通常・生成Studio Hostへ静的な選択済Inspectorを接続する実装候補となった。この節の記述時点では**未統合Draft**だったが、**現在はmain統合PR #3540により通常CLI／通常・生成Studio Hostの静的参照Provider接続が実証済み（S2-reference）**である。配布Plugin、正式Profile選択、Provider固有Link/ロード済Byte、現在Grant、S2-product/S3はなお未証明（現行評価は§21参照）。[RQ-037のBOM要求Draft #3538](https://github.com/hat47x/tei-platform/pull/3538)と[Version Matrix Reference Draft #3541](https://github.com/hat47x/tei-platform/pull/3541)も未統合の候補で、[RQ-036全体Map](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)とは別の要求である。

**今回の設計結論**：Traitの履行と実行主体の所在を独立軸で管理する。既存のContract/Provider Resolver・Operation Contract・Runtime/Host・BOMは一つの巨大な「合格」状態には還元しない。

| 判定軸 | 対象となる質問 | 資格を与えないもの |
| --- | --- | --- |
| **要求** | 誰の、どのSubject／Operation Variant／Scope／Targetに何の原子Traitが必要か。Required集合は完全か | 未Parsed Source、Advisory Hint、モデル図に描かれた同名Node |
| **実現方法** | Default／Generated／Manual／Provider／Externalの何が必要Contractを満たす予定か | 選択だけのProvider、候補Plugin、署名だけの手動実装 |
| **実行責務** | 実際に呼び出すのはTEI製品Hostか、生成された静的成果物か、外部Hostか。参照実装か製品か | 実行入口を持たないC-4に対する「製品の実Invoke完了」、局所Go helperに対する「配布製品S2完了」 |
| **振る舞いの検証** | 指定Target・Consumer・Attemptで、前提／結果／Effect・合成順序と独立Expectedが一致したか | 個別Trait PASSの合算、Synthetic・Mock・自称Receipt、同じResult bytes |
| **現在の資格** | 現在Grant／Source改訂／Artifact／Provider Rule／開示／受理はまだ有効か | 過去のTest PASSやQualified BOMだけによる権限・業務意味の自動付与 |

五軸は**単純な段階順位ではない**。実行せず静的に検査できるTraitも、実行しなければ検証できないEffectもある。「未実行だから契約がない」でも「契約があるから実行済み」でもない。外部Hostが本番実行するC-4では、TEIの宣言・解決・Build/Exportは完了し得るが、**実行・Effectの現在資格は外部の受領・照合Evidenceが来るまで`unknown/delegated`**とする。Externalを選んだだけでRequiredを`satisfied`にしない。

### 19.2 最初に必要な、最小の同一Attempt照合

新しい中央Trait Engine・万能Receipt Schemaを先行導入せず、既存#951/#1016/#1158/#1180/#2514/#2600のデータを同一Attemptで参照できるようにする。対応する既存Keyがなければ**不足箇所を明示して**から薄い参照を追加する。

```text
Subject Kind/ID + accepted Source occurrence/revision + Required Closure
             ↓ (Operation ID/variant/scope/target/effect)
Atomic Trait obligation + exact Contract revision
             ↓ (existing #951 selection)
Provider/Implementation/Host ownership + selection Rule/Manifest revision
             ↓ (product binary or external consumer)
Actual entry/build artifact + invoked provider + attempt identity
             ↓ (existing Scenario & Independent Expected)
Behavior/Effect/Completeness/Failure assessment
             ↓ (checked separately at use time)
Current Grant/Disclosure/Meaning/BOM reuse eligibility
```

最も短い反証は、**選択B・実呼出A・出力値は同じ**である。契約の投影、Mockを介した呼出、同値JSONはProviderの実体が選択と一致したことを保証しない。同様に、Registryの`ArtifactDigest`文字列の一致は実ビルドしたBinaryのSHA／選択済みPlugin byteを証明しない。Provider Source/Manifest・選択Rule・Consumer Source Index・Generator/Artifactをそれぞれ別の失効要因として扱う。

**手動実装必須**：`manual-required`は、対象をLoaded/BoundしRequired Closureが分かる範囲で「未実装」と診断できることが価値である。C-4（実行は外部）でも、TEIが契約を要求した以上、手動実装の宣言・適合結果を受領できなければ「実装済み」としない。実装そのものをTEIで実行することは強制しない。

### 19.3 RQ-037候補への接続：Trait単体適合と合成適合を分ける

[既存BOM設計](VERIFIED_COMPOSITION_BOM_AND_MATRIX_DESIGN_2026-11.md)はComposition Pattern／Resolved／Built／Tested／Qualifiedを区別する。**今回の新たな接続は、各Matrix行のRequired Trait保証が、Provider単体・Consumer適用・Lifecycle実効Pathのどの範囲で観測されたかを説明する**こと。

例：Provider A v1/v2 × Consumer B v1/v2の4組合せで、A v1/v2が`Readable`相当のContractを個別に実装できても、Consumer B v2のResult Shape／Host Scope／Hook Effectに不適合なら、該当行は`unverified/failed`。AとBの個別PASS、Pairwise計画、Manifestの互換Version Rangeを「全組合せQualified」にしない。**外部Host C-4**は、実行のWitnessを外部から照合できる場合だけその行の実行資格を評価し、TEI製品内のPluginロード・Invocationを偽装して記録しない。

同一のTraitが別Targetへ適用されるとき、ReadOnly性・更新原子性・観測契約が要求する**意味**は保つが、Go静的合成、Java/.NET、Browser、外部Hostの**実装・配布形態は一致させない**。1Targetの証拠を他Targetへ黙って複写しない。BOMはデータとしての派生表示であって権限やMeaningの正本ではない。

### 19.4 最小の横断反例（既存Scenarioへの追加観点、全てNOT RUN）

| ID | 一つの変更・圧力 | 期待する不成立点 |
| --- | --- | --- |
| **TEB-01：実行主体C-4** | 同じTrait Requiredを持つFunctionを外部Hostに委ね、TEI自身は実行しない | TEI側は`delegated/not-observed`。外部Receipt／独立Expectedが無ければRuntime Conformance未確認。C-4を自動的に設計失敗としない |
| **TEB-02：S2-reference vs product** | Draft #3540相当の静的Local Inspectorが通るが、製品Targetのactual entry／出典が未検証 | 局所・参照の成果と正式Product S2/BOM資格を分離。通常公開CLI／実Wailsそれぞれの到達を別観測 |
| **TEB-03：合成で保証喪失** | Trait A、Bを個別にPASSさせ、合成Hookで認可Guardを迂回 | 単体2件PASSでも実効Graphの合成適合FAIL。Hook/Effect順序を明示しなければ未資格 |
| **TEB-04：Required外部実装欠落** | C-4 External/Manualを指定しても、必要Operationの実装・検証結果が受領されない | Sourceは保持し、対象/契約/Scope付きの`unfulfilled`または`unknown`。Delegated＝implementedとしない |
| **TEB-05：証拠の古さ** | Provider Manifest／Ruleだけ変更、Consumer Source／実結果は同じ | 影響BOM行とTrait履行Evidenceを再資格化。無関係Targetの全Build無効化は避ける |
| **TEB-06：現在認可と過去BOM** | exact Lockの検証済行を使う直前にHost Grantを撤回 | BOMの過去適合記録は保持、現在の実行・開示は拒否。GrantをTraitやBOMから発行しない |

TEBは[#3278](https://github.com/hat47x/tei-platform/issues/3278)のTrait保証を切り口にした**検証観点のみ**。独立Expectedと実行は[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)、外部実行の認可は[#1174](https://github.com/hat47x/tei-platform/issues/1174)、BOM行は[#3267](https://github.com/hat47x/tei-platform/issues/3267)、Product S2は[#1251](https://github.com/hat47x/tei-platform/issues/1251)へ委譲。第二の受入票正本は作らない。

### 19.5 計画の変更範囲と停止条件

- **P0（優先変更なし）**：現行mainの限定S1を維持し、[Draft #3540](https://github.com/hat47x/tei-platform/pull/3540)を#1251の製品S2へ適合させる。通常CLI dispatchと実Wails Hostの実選択・実Callbackを独立に観測する。S2-referenceに到達してもTargetのBuild provenance／本番AuthorityをS3として別票にする。
- **P0-F（並行）**：一原子Traitで`provided`と`manual-required`の比較、既存Resolverによる正確な`missing`診断と同Target Conformanceを検証する。#3279のOwnerで実施し、C-1/C-2/C-4の採否をここで先取りしない。
- **P1（既存計画に引渡し）**：RQ-037の正式製品要求・BOM ReferenceはDraft #3538/#3541の受理範囲で評価し、最初は**現実に成立した一つのConsumer×Provider構成**のTrait履行Evidenceを追加で読めるか検証する。2×2の新Qualified行を発行するのは実Build/Conformanceが揃ってから。
- **P2**：C-4を選ぶTargetとTEI内実行Targetを同一のRequired/Failure/Effect契約で比較し、Ownerに実行主体を強制せず、受領可能な外部Evidenceの範囲のみで資格化。B2C Headless／VUI／KON等を新Runtimeの必要条件にしない。

**採否・撤回**：原子Traitを追加するたびに、新しいHost、Resolver、固定Enum、認可、Receipt形式、独立Test Runnerを増やす必要があるなら、Traitではなく既存Operation/ContractへのRequired Constraintとして縮小する。「実行を接続しない」選択を受け入れられない場合はPlugin-firstをTEI固有Hostへ密結合させている兆候として再評価する。実装時は、Trait追加で初めて検知できた具体的な不足件数、手動分岐削減、Cross-target再利用、誤PASS/過剰拒否、実性能・Build負担を**同一条件の対照**で測る。

**到達状態**：実mainと現行Issue／Draft候補の静的照合から作成した設計判断・実装引継ぎ。C-1/C-2/C-3/C-4の正式採否、RQ-037の採用、TEB-01〜06、外部Host／実Provider Artifact／Wails、Qualified BOM／実Grant・Owner Meaning Acceptanceは**未決／NOT RUN／NOT ACCEPTED**。

**参照（主張の範囲限定）**：Bao/Rompf, *Type, Ability, and Effect Systems* (OOPSLA 2026, DOI:10.1145/3839518)はType/Ability/Effectを単一機構に還元できないと論じ、Moy/Dimoulas/Felleisen, *Effectful Software Contracts*（POPL 2024）はEffectの契約による制約を扱う。これらは「Trait/能力/Effectを独立に検証する」設計を考える参考であり、TEIでの仕様・実装適合を証明するものではない。

## 20. 実行Carrier・Hook位相・CLI/MCP公開契約をTraitから分離する（2026-11-09、現main実測の横断照合）

### 20.1 §19の実行主体選択を精密化：C-1とC-2は同じ物の二者択一ではない

前節の[構成接続の判断材料](../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)は、C-1 `routing.Router`／C-2 `servicehost`／C-3能力ID／C-4外部実行を選択候補として挙げた。その後、[main PR #3583](https://github.com/hat47x/tei-platform/pull/3583)の[入口比較実測](../evidence/CONNECT_POINT_C1_C2_2026-11-09/README.md)により、**C-1の入力は`executor.Command`で配備動作を実行し、C-2は`requestID+payload`を世代固定のService Carrierへ運ぶ**と分かった。**この二つは必ずしも代替ではなく、異なる責務を持つ**。同じ要求を二重実行させないことは依然として必要だが、同時採用＝重複経路と決め付けない。

[main PR #3588](https://github.com/hat47x/tei-platform/pull/3588)の[限定参照](../evidence/SERVICE_CARRIER_CONNECT_REFERENCE_2026-11-09/README.md)ではC-2を非テストの参照ツールとして組み立て、Routeがあれば世代1を固定して要求を運び、Route/Host入力が無ければ呼出前に拒否する経路を観測した。これは**参照の組立可能性**であり、製品CLI、実Plugin/Service Provider、実配備・現在認可の成功ではない。C-1/C-2の製品採否・接続先・C-4（外部実行）許容は引き続き既存Gate／HostのOwnerの決定に委譲する。

**Trait設計への結論**：`Requires`や`Provides`が対象Operationに束縛されても、`Delivery/Deployment Executor`と`Service Carrier`を同じ必須Traitにまとめない。必要な原子的保証（例：条件付き更新・指定ScopeのReader・世代固定）は契約として表現し、それを**どのCarrierが実現し、どのHostが実行を許可したか**をImplementation/Attempt側で照合する。Go単一バイナリ、外部Host、生成Targetのいずれでも、特定のCarrierをTraitの必須物理実装に固定しない。

### 20.2 Hookの「宣言が見える」を、Hook実装の履行と混同しない

[main PR #3579](https://github.com/hat47x/tei-platform/pull/3579)の[機能定義書の限定試験](../evidence/XTEA_LOWCODE_HOOK_FUNCTION_DEFINITION_2026-11.md)では、`function-definition/v1`に`pattern`ごとの`hook phase reads uses produces on-failure`を、実行側と同じ`hook.PhaseOrder`で出力する処理・試験が追加された。これは、**宣言の投影と位相順がコード上で接続された**という実成果であり、Hookの実Pluginへの登録、実データ読み込み、値提案の採用・保存、最終業務Effectまでを検証したものではない。

| 区分 | 具体的な証拠・不足 | Traitの履行判定 |
| --- | --- | --- |
| **Declared Hook** | Pattern／Phase／`reads`／`uses`／`produces`／`on-failure`をSource・定義書に表示 | Hook Slotと権限／必要契約の**宣言**。表示成功は実装ではない |
| **Selected Hook implementation** | Contract／Selected Provider／Rule／Scope／exact Source・Artifact | 既存Resolverでの選択結果。読み書き権限・効果は未確定 |
| **Invoked and constrained** | 実Phaseで実Callback、許されたRead/Use/Produce／Failureの観測 | その実Attemptの制約適合を独立Expectedで確認 |
| **Applied outcome** | 提案された値の受理・保存・Commit/Receipt／現在Grant | 実保存・業務Effectを実装し確認した範囲でのみ結論 |

**一つの重要な反例**：`hook.PhaseOrder`どおりに機能定義書が整形されても、Runtimeで別位相に呼ばれた場合、または`produces`が単なる提案として返っただけで保存されなかった場合は、実業務Traitを`fulfilled`にしない。Hookの`reads/uses/produces`は、未宣言の値やCapabilityへアクセスする許可ではなく、宣言から実行へ渡す**制約の上限**である。Hookの追加が従来のGuard/Authorization/Effect/Receiptの必須経路を回避することは許さない（§14〜15）。

### 20.3 Tool Discovery・結果の一致と、Requiredの完全性は別

[main PR #3563](https://github.com/hat47x/tei-platform/pull/3563)は、存在しないModel CatalogのnamespaceをCLIだけが`NAMESPACE_NOT_FOUND`とし、MCP参照側が`ok:true, entries:[]`としていた非対称を修正した。[main PR #3582](https://github.com/hat47x/tei-platform/pull/3582)ではCLI/MCP Adapterの一致を追加観測した。一方、[PR #3590](https://github.com/hat47x/tei-platform/pull/3590)は、広告された引数／Tool集合と実際の実行可否の差を調べる**未統合のEvidence候補**である。

**Tool Discovery、公開Operation、TraitのRequired Closure、実ProviderのOfferは四つの異なる集合**である。例えばTool一覧に`tei_inspect`が現れても、未知Subjectへの`SUBJECT_NOT_FOUND`、Root/Namespace不正、Provider不足や非開示Requiredを成功に昇格できない。逆にMCPに載っていないOperationがCLIで存在することは、その機能にRequired Traitが不要であることの証明ではない。

生成AI/外部Harnessへ渡すCLI/MCP/Studio応答は、少なくとも対象Subject/Operation、Current Source・Scope、実用可能な公開面、実行したResult/Failureと未受領状態の違いを説明できる必要がある。ただし**実行不可能なToolの存在や秘匿Requiredの件数を漏らさない**。Advertised Schemaと実Validatorが食い違う場合は、既存#3420/#3419へ不適合として引き渡し、Traitの`satisfied`率に計上しない。機械的なFact出現や同じJSON文面だけでは、業務意味・正答・実Provider履行を証明しない。

### 20.4 新たな最小反証と現行優先順（既存Scenarioへ移管、すべて新規NOT RUN）

| ID | 一要因だけの差分 | 受入上の結論・既存Owner |
| --- | --- | --- |
| **TCR-01 Carrier責務** | C-1の`executor.Command`をC-2のService payloadへ対応づけず同一Traitで扱う | 適用対象の違いを診断。両方必要な場面は二つの実装契約を明示（#951／Kernel Gate） |
| **TCR-02 C-2 Route／世代** | Routeを削除、またはHost世代を変更した後に旧Invokerを使う | 参照でのRoute拒否を保持し、製品Target/Host Grantは未資格。旧世代成功を再利用しない（#1251／#1174） |
| **TCR-03 Hook宣言と実行** | 機能定義書で正しい位相順に見えるが実Runnerが別順・Provider未呼出 | 表示・宣言成功のみ。実Hook/Required Traitは未充足か未観測（#2110／#3279） |
| **TCR-04 Hookの提案と確定** | `produces`が型付き提案を返すが保存は拒否・不明 | 提案取得と確定更新/Receiptを別票にする（#1129／#1174） |
| **TCR-05 CLI/MCP空集合誤判定** | 存在しないnamespaceをMCPで`entries:[]`成功と返す旧実装 | `NAMESPACE_NOT_FOUND`で失敗。同一Sourceの欠落はRequiredゼロではない（#3420／#3262） |
| **TCR-06 Discovery≠Invoke** | Tool宣言にある操作が実Validatorで拒否、またはToolはあるが実Provider未選択 | 広告された操作と実行されたOperationを分け、不足を適切なOwnerへ記録（#3419／#3420／#3279） |

**直近の実装順**：既存#1251のInspector通常CLI・Studio選択済Provider実Call（当該節の記録時点ではDraftだったが、**main統合済み#3540のS2-referenceは成立**。正式Providerを使ったS2-productには未到達）をP0のまま維持。Hook定義書の表示を次のP0必須Gateにせず、**既存Hook Runnerの限定一位相**についてDeclared→Selected→Actual Invoke→Outcomeを独立に一往復する実験を、#2110/#1129の実装Ownerに引き渡す。CLI/MCPのTool説明と実Validatorの差は#3419/#3420へ委譲し、Trait名を増やして補修しない。C-1/C-2の製品利用・C-4の最終採用もここでは判断しない。

**停止・縮小条件**：`Trait = Hook`、`Trait = Carrier`、`Tool Discovery = Required Closure`と固定したくなる、別Resolver/Host/Reader/Runnerが必要になる、参照/定義書/広告の成功をProduct Conformanceへ昇格する場合は、原子的Traitの利用範囲を既存Operation ContractのRequiredへ縮小する。

**状態**：#3579/#3588/#3583/#3563/#3582のmain反映と参照試験記録の静的照合。TCR-01〜06、実Plugin/実Hook Runner＋DB、正式PluginでのProduct S2／S3（静的参照ProviderによるS2-referenceはmain #3540で実証済み）、C-1/C-2/C-4の正式採否、正式Trait Schema、現在Grant／Owner Meaning Acceptanceは**この追補ではNOT RUN／NOT ACCEPTED／未決**。

## 21. 11月9日の実装証拠をTrait履行へ引き上げる最小受入（履行候補と実証済みの差）

### 21.1 現mainで進んだ実装と、まだ主張できないこと

本節は[実装状況](../../docs/status.md)を現行の基線として読む。§18〜20に含まれる「#3540は未統合」「通常CLI/Studioは未接続」という記述は、その節を書いた時点の履歴であり**現在の未達項目ではない**。主な差分：

| Evidence／採用された部分 | 現mainで確認できたこと | 原子Traitの証明にはなお不足すること |
| --- | --- | --- |
| [Inspector #3540](https://github.com/hat47x/tei-platform/pull/3540)／[#3671](https://github.com/hat47x/tei-platform/pull/3671) | 通常CLI dispatch、通常・生成Studio Host、Wails BridgeのGo境界から、共通Contractの**静的ローカル参照Inspector**を実選択・呼出（**S2-reference**）。PR exact-headのHealth・Wails Host等の限定CI記録が存在 | **正式Environment ProfileでのProvider選択→その固有Artifact Entry→二Product Consumer**、Load/Link真性、Current Grant、完全なRole/Source Closure、Owner意味受理（**S2-product/S3**） |
| [Inspector同一性計画 #3739/#3740](https://github.com/hat47x/tei-platform/pull/3740) | Canonical ID・Workspace Manifest・Sourceのrevisionが別観測であり、World/SourceIndex FPだけでは同一対象を証明しないと明確化 | [Draft #3735](https://github.com/hat47x/tei-platform/pull/3735)の実Contract負例IS-01〜04の現main実証、外部ProviderのRoot/Origin真性 |
| [Model Catalog #3728/#3754](https://github.com/hat47x/tei-platform/pull/3728) | `model catalog`は編集中の検証NG Workspaceでも**外側の`ok:true`で読取可能**。内側の`validation.ok/errorCount/warningCount`で検証成否を独立表示 | Catalog読取・検証要約のみで必要Traitの**全Required Closure**や、対象Operationの有効化・Provider実呼出・現在Grantを証明しない |
| [Hook outcome #3761](https://github.com/hat47x/tei-platform/pull/3761)／[Evidence](../evidence/XTEA_LOWCODE_HOOK_OUTCOME_CHECK_2026-11.md) | `internal/hook`の`ValidateOutcome`が**語彙外の不成立名**を`HOOK_OUTCOME_UNDECLARED`で拒否。無名の不成立は許容し、既存Runnerの拒否位置を保持。限定Go試験が記録される | 外部Plugin／プロセス境界、実業務Result/DB Commit・Receipt、対象Hostでの合成適合。**宣言内の結末名であること**だけで業務効果は生まれない |
| [6 Binding調査 #3757](https://github.com/hat47x/tei-platform/pull/3757)／[Evidence](../evidence/BINDING_FORMS_CONSUMER_SURVEY_2026-11-09/README.md) | Environmentの6種（DeploymentExecutors／ObservationProviders／EvidenceProviders／Instances／Bindings／ResourceBindings）にReaderがあり、実行到達は異なる。Deployment実行は**参照経路のみ**、Observation/EvidenceはBinding解決が製品にあるがRouter実組立なし | 6種を`declared`から`executed`へ一括昇格、参照Executionを製品実行と混同、ReaderがあるだけでMissing/Unknownを消すこと |
| [Self-build #3760](https://github.com/hat47x/tei-platform/pull/3760)／[Evidence](../evidence/TEI_GO_SELF_BUILD_INPROCESS_RESOLUTION_2026-11-09.md) | 呼び手指定のcatalogと必須digestから、entry自身がin-processで計画を導き、childと`planFingerprint`を一致させた。`planReceipt.derivedBy`が経路を明示 | `l2bClaimed`／`qualified`はfalse。計画導出一致をProviderの実Link、Artifactの権限・Meaning受理、Build/Runtime全保証に昇格させない |

この表は`required`／`missing`／`unknown`の判定に使う根拠の所在を示す**設計評価表**であり、正式な単一`TraitState` enumや新しい証拠Storeを追加する提案ではない。既存の#951/#1016/#3279/#3262/#1251/#2831/#1180/#2514の責務を変えない。

### 21.2 原子的Traitの履行条件を「読めた・選んだ・呼んだ」の行で明確にする

最小の照合列は次のように理解する。

```text
Subject/Operation + Root/Manifest/Source occurrence/Revision
     → Required contract closure (complete / partial / opaque)
     → Validation status (catalog read != validation pass)
     → Provider binding + exact selection/target + implementation availability
     → Actual consumer call (CLI/Studio/External Host) + generation/attempt
     → Hook path / typed result / failure / effect (if required)
     → Independent Expected + Artifact/Receipt evidence
     → Current disclosure/Grant/Meaning (separate present-time decision)
```

**成功は交差条件**であり、互いの観測結果を埋め合わせない。例えば、`model catalog.ok:true`でも`validation.ok:false`のWorkspaceに対してRequired Traitを「健全に有効」としない。ただし編集中のSource/候補Traitは読めてよい。存在しない名前空間と、合法な空のRequired集合を区別し、非開示Requiredの存在・個数を非許可Consumerへ漏らさない。

`manual-required`にDefaultがない場合は、対象のSourceとRequiredの取得が十分な範囲で`missing-implementation`を診断し、Targetが未受領・未Loaded・外部実行中なら`unknown/not-observed`を保持する。結末名が正当でも実Functionが呼ばれたか、`produces`が提案か保存か、Effectが確定したかは別。どの実装経路でもTrait定義の原子的保証とHookのGuard/Revision/Authorityを交換しない。

**SourceとBuildの同一性**：`Canonical.id`の一致だけで`Kind`や`Workspace Manifest`が同一とはしない。Source位置移動、同Byte別Origin、Provider raw/Manifest更新、Selection Rule変更、Catalogの版変更を別々に観測する。同じ`planFingerprint`／World FP／出力JSONで過去のProvider ConformanceやBOM Qualifiedを復活させない。

### 21.3 旧§18の並行草案を整理した結果と、引き継ぐ試験観点

並行PRの統合によって本書には§18が3件存在し、そのうち2件は**S1→S2未接続という旧時点の計画を繰り返していた**。今回、先頭の正本§18（MTP-01〜05、RQ-036 Map投影）を残し、後置の2件を内容差分と照合して重複整理した。検証観点は既存#1180/#2514へ委譲し、他レーンの検証履歴を消さずGit履歴で辿れる。

後置草案の**TMP-01〜07**は次の最小反証へ継承する（ケースIDは履歴参照、正式Runner/Expectedの第二定義ではない）。

| 継承する負例 | 最新の判定上限とOwner |
| --- | --- |
| **TMP-01/02**：局所Helperでは選択B、通常CLI／Wailsでは別経路 | main #3540の**静的参照S2**で解消した範囲と、正式Profile選択PluginのS2-product不足を分離（#1251） |
| **TMP-03/04**：Manual不足、部分Model Map／非開示Required | 既知のMissingのみ表示し、Required完全性/Origin/Disclosureを推論しない（#3279/#3262/#1124） |
| **TMP-05/06**：Graph PatchでGuard迂回、Provider改版／Source-only漂流 | 個別Provider PASSとEffective Path適合／Current Evidence失効を分ける（#3503/#951/#2600） |
| **TMP-07**：合成計測が整合、実Observerは未呼出 | Synthetic Observationは調査候補。実Measurement/SLO／Trait Conformanceは未資格（#3279/#3271） |

### 21.4 次の実装を一つの不足へ縮める

**P0-A：正式ProviderとのS2-product接続**。既存#951/#1016 Profileが選択した**Bの実Artifact/Entry**を、main #3540の通常CLIとWails Hostに供給し、B selected/B invokedを同一Target/Source/Attemptで独立検証する。B selected/A invoked、別Workspace Manifestで同ID、同Byte別Source、Provider withdraw/epoch変更、失敗Resultと現在Grantを別票で観測。参照Providerを残すなら明示選択であり、失敗時の暗黙Builtin fallback禁止。**新しいResolverや独自Hostは設けない**。

**P0-F：単一Trait義務の実装差**。既存`inspect` Contractの必要な一能力について、標準/Provider実装がある場合と`manual-required`が欠ける場合を、同じRequired/Targetで比較し、完全性が確認できない場合は`unknown`、明示Missingの場合だけ未充足診断とする。実際のOperation/Contractが未対応なら新Trait型を足す前に既存Required形で試し、機能価値を測る。Hook境界は[main #3761](https://github.com/hat47x/tei-platform/pull/3761)の結末語彙拒否を再利用する。

**P1-B：構成と外部実行への展開**。一つの現実に成立したProvider×Consumerのexact Lockから#3267の限定Version Matrixへ進み、C-1/C-2/C-3/C-4の宣言解決・Carrier入場・実Effectを別結果とする。実DB／Function Queryは#2586、三視点Graph／Trait Mapは#1124、Build catalog出所は#2831が所有する。Traitがどの領域にも一律に追加Host／Runnerを要求するならCore化は撤回する。

**新しい独立Expected候補（本変更ではNOT RUN）**：

| ID | 一要因変更 | 必要な判定 |
| --- | --- | --- |
| **TQ-01** | Catalogは`ok:true`で`validation.ok:false` | 読取成功とValidation不合格を両立表示。必要Trait全充足・実行可へ進まない |
| **TQ-02** | Canonical本文IDは同じ、Kind／Workspace Manifestを変更 | 同一Origin/Subject/Requiredへの誤Bindingを拒否。Draft #3735を実検証するまでは未資格 |
| **TQ-03** | CLIとWailsに静的参照B、正式Profileは別Provider Cを選択 | **静的S2-referenceだけ**。正式PluginのS2-product／BOM資格は拒否 |
| **TQ-04** | Hookが未宣言Outcome、または正しい名を返すが実保存なし | `HOOK_OUTCOME_UNDECLARED`か正当な不成立へ分類。Outcome名の許可を業務Commitへ昇格しない |
| **TQ-05** | Observation/EvidenceのBinding解決あり、Router実Invokeなし | `resolved/not-observed`を保持。環境の6宣言に同じ`executed`を配布しない |
| **TQ-06** | self-build entryとchildが同Fingerprintだがcatalog別digest／Claim未承認 | 実行元の申告とinput digest／`l2bClaimed:false`を保持。BOM/権限を自動認定しない |
| **TQ-07** | 原子的Traitの手動実装未提供、Required closureは一部未Loaded | 確認済Missingと全体`unknown`を区別、秘匿項目を開示しない |
| **TQ-08** | Provider/Role改版後もResult Bytesは同じ | 対象Scopeに影響したConformance/Evidenceだけ再資格化、無関係Artifactは保持可能 |

**本節の到達状態**：main #3540/#3728/#3754/#3757/#3761/#3760などの現物と限定証拠を照合し、§18の重複を編集上整理して受入計画を更新した。**正式PluginのS2-product／S3、実External Provider・実DB/Browser、TQ-01〜08、正式Trait Schema、Owner意味受理は、この変更ではNOT RUN/NOT ACCEPTED**。主張の上限は元Evidenceの実行対象・Source/Target/時点に限定する。


### 21.5 実行に届いたHookのusesと、まだ届かないTrait保証（2026-11-09横断追補：同一Attemptの必要な観測だけを統合）

§21までの`Declared Hook→Selected Implementation→Actual Invoke→Outcome→Effect/Receipt`は**設計候補の直列化**だった。[main #3828](https://github.com/hat47x/tei-platform/pull/3828)と[main #3843](https://github.com/hat47x/tei-platform/pull/3843)で、少なくとも**参照CLI／StudioのHook `uses`からCommand境界へ到達する配線**が生まれた。これは前節までの「Hookの宣言は見えるが呼ぶ配線自体は無い」という旧状態を解消した。実装と[限定実行記録](../evidence/XTEA_LOWCODE_HOOK_DECLARED_CALL_WIRING_2026-11.md)の一致を認める一方、**外部Plugin、実業務DB、現在Grant、成果物実Link、Owner意味受理**はこの経路では実証されていない。

| 既存の実装済み差分 | Traitの履行判定へ具体的に使えること | まだ保証されないこと |
| --- | --- | --- |
| [#3828](https://github.com/hat47x/tei-platform/pull/3828) `uses`未実行を`Uncalled/NotEvaluated`として返す | 宣言だけある能力を`executed`として偽装しない。Manual/Provided不足の**検出候補**として利用する | 「未呼出しなら必ず実装未提供」とは限らない。入力未束縛・未到達・現在権限拒否を区別する |
| [#3843](https://github.com/hat47x/tei-platform/pull/3843) `RunDeclaredCalls`／`CommandCaller`／`PhaseReport.Calls` | 実際に**参照Command境界へ行った能力名・呼出し結果/失敗**と、未束縛のため呼ばなかった能力を別に記録。CLI/Studioの参照実行・Go試験の範囲で成立 | `CallOutcome.Called`は**宣言制限付きCaller境界への試行**であり、正式Providerの実Entry到達・成功・Commitの証明ではない。`calls.go`では`Declared.Call`前に`Called=true`を立てるため、内側が利用不能でも`Called=true + Error`が成立しうる |
| [#3802](https://github.com/hat47x/tei-platform/pull/3802) `produces`値をCLI/Studioで候補表示 | 提案値・未採用をUIから区別し、作業者が検証・採用へ進む位置を見つけられる | Proposal表示を保存・承認・Effectへ昇格できない |
| [#2784](https://github.com/hat47x/tei-platform/pull/2784) 期間の現在状態をdispatch直前に読み直す | 過去のAction Offer時点のGuard判定を現在時点へ無断再利用しない**時間条件の反例**が限定Go試験として存在 | 実DBでのAtomic CAS／Commit直前の競合検査、現在のAuthorizationを省略してよいこと |
| [#2812](https://github.com/hat47x/tei-platform/pull/2812) 同じProvider Loadから選択Namespaceのparsed原文・Manifestを内部保全 | 「今ファイルを開き直して得たbyte」を「そのConsumerが解析したbyte」と取り違えず、選択SourceのOccurrence/RevisionにTraitを束縛する材料ができた | `FullInputClosure=false`。Provider全体の負依存・multi-file atomicity・開示許可は未保証。内部保全RawをCatalog/Map/生成AIに無許可で投影しない |
| [#3839](https://github.com/hat47x/tei-platform/pull/3839) `cmd/tei`の実Goビルドと依存Source閉包の限定束縛 | 選択TargetのSource/Module/Toolchain/Build入力と実Artifactの独立参照試験を進められる。Nested Studio module欠落や未確定BuildをPASSにしない | 実Pluginのリンク元・実Providerが当該Artifactに入ったか・LoadしたByte/Trust/Host Grantは別。1Target成功から全BOMをQualifiedとしない |
| [#3848](https://github.com/hat47x/tei-platform/pull/3848) Observation/Evidence参照の`attemptedCount + complete` | 1件/1件成功と1件/2件成功の差、ゼロ試行を`complete=false`として扱える。**呼出し結果の部分性**を機械可読にした | `complete=true`は「試した分は参照契約上、全件成功した」の意味。**必要集合そのものの取得完全性**、実世界との一致、unknown/stale、Provider正式資格を証明しない |
| [#3829](https://github.com/hat47x/tei-platform/pull/3829)、[#3847](https://github.com/hat47x/tei-platform/pull/3847) | MCPの引数一覧を宣言`inputSchema`から導出し、RQ-036全体Mapを初学者と生成AIの「現在地・近傍・実装未充足」へつなぐ設計を更新。**同じRequiredの別Projection**を再利用する方向と整合 | Tool Schemaの一貫性・Mapの表示から、Required全部解決、実Callback、現Grant、Owner受理を導出しない。MAP-17〜23は未実施 |

### 21.5.1 RequiredなTraitは「試行・応答・結果・実効果」を別に要求する

**契約の原子性は、単一のboolが各層を代表することではない**。例えば`Invokes(capability=sales.submit_order)`のような原子的義務を採用するなら、実装選択時の`uses`と、実Attemptの`Called`だけで自動履行としない。そのTraitの定義が「Command境界に到達すること」までを要求するなら、ScopeとCaller実装を照合した上で局所履行を主張できる。一方「Providerが処理を実行する」「保存まで完了する」を要求するなら、**別の原子義務／Effect Contract**と独立Expectedを必要とする。Command拒否・未知Effectを呼出成功へ繰り入れない。

- **必要集合の被覆**：`requiredCount`は十分にLoaded/AuthorizedされたRequired closureからのみ導出し、`attemptedCount`や描画済みNode数を分母にしない。`complete=true`だけで`Required Fulfillment=complete`にしない。
- **失敗の位置**：未到達・入力未束縛・Caller不在・Callerからの拒否・Result不適合・外部Effect不明は区別する。内側に到達できなかったCallOutcomeを、`Called`欄だけで「正式Provider実行済み」としない。
- **時間と出典**：静的なTrait Bindingが存在することと、Dispatch/Commit時の期間State・Current Grantの適格性は別。別Workspaceでも同Fingerprint/Canonicalが得られる場合、同一Reader/Originや現在のPermissionまで認定しない。Same-Loadの内部Raw証拠も秘匿対象である。
- **表示と自動化**：RQ-036 Map／CLI／MCP／Studio／AI向け説明は同じSource/Required/Conformanceへの参照として、`declared`・`selected`・`attempted`・`refused`・`not-observed`・`effect-confirmed`の**独立の事実**を表現する。これらの語彙は説明用であり、今回新しいCore状態Enumを提案しない。自動で「実装候補」の採用・権限付与・業務確定を進めない。

### 21.5.2 最初の反証：既存Operation/Scenarioで一往復（全件新規NOT RUN）

| ID | 一要因変更 | 独立Expectedとして確認する区別／所有先 |
| --- | --- | --- |
| **TFX-01** | `uses`を宣言、CallInputsを渡さない。一方で別例はCaller境界を呼ぶ | 未呼出`Uncalled/NotEvaluated`と`Calls[phase].Called`を分ける。`Called:true + Error`は`Provider Effect PASS`にならない（#2110/#3279） |
| **TFX-02** | Callは参照Commandで成功し、`produces`は候補表示されるが保存しない | Declared Call／Proposed Value／Applied Effect／Receiptを4票で分ける（#1129/#1174/#3279） |
| **TFX-03** | `Action Offer`後に期間Stateを`open→closing`へ変更 | 直前Guard再読込で拒否。旧Offer時の可否や古いTrait資格を現在の許可として使わない（#1174/#2586） |
| **TFX-04** | Providerがparsed原文を読み込んだ後、物理Sourceを変更。同Byteの別Originも用意 | Same-Load Snapshotを保持しつつ、新Load時に影響Evidenceを失効。RawとManifestは権限なしでは非公開（#3262/#2516） |
| **TFX-05** | 必須2 Providerのうち1件だけ成功、あるいは0件試行 | `attemptedCount/complete`とRequired closureを独立集計。Partial/Unknownを`not-required`へ縮退させない（#951/#3267） |
| **TFX-06** | 同じTraitとResultで実Build依存・ManifestまたはToolchain Byteを変更 | 該当Target ArtifactとConformance/BOMだけ再資格化。未選択Targetの全Build無効化、同Resultによる旧資格復活を禁止（#2600/#2831/#3267） |
| **TFX-07** | Map/Tool SchemaにはTrait/Hookが表示されるが実Caller/Providerは未選択 | 表示の成功・契約書の見通し・履行の欠落を区別してSource/Operationへ戻れる。権限外のTraitや未読込の件数を漏らさない（#1124/#3420/#3279） |

**実装順を変更しない**：P0-Aは正式Profile/Artifactが選択した`Provider B`を通常CLI/Wailsの実Consumerへ注入して、**selected B＝actual B**を独立検査すること（#1251/#951/#1016）。P0-Fは既存Contractを用いた原子Traitの`manual-required`不足を、十分に受領したSource/Requiredの範囲で診断すること（#3278/#3279）。Hookの参照Callは到達済みなので、次は**宣言`reads`→能力パラメータの明示Binding**（推測による埋め合わせは禁止）と実Effectの確認が薄い未接続辺であるが、実Plugin/外部DBをP0の新Hard Gateにしない。受入Ownerは既存#2110/#1129、独立Expectedは#1180/#2514、BOMは#3267、RQ-036の図表からのナビゲーションは#1124が所有する。

**停止条件**：`Called`だけでEffect成立、`complete`だけでRequired closure完全、Source Rawの保持だけでDisclosure許可、Map上の緑線だけで実Provider受理を主張するなら、当該投影を暫定`unknown`へ戻す。原子Traitと既存Command/HookのContractが同じ義務を二重定義してしまうなら、Trait型を追加せず既存Contractの`requires`へ収束する。

**到達範囲**：上記はmain #3828/#3843/#3802/#2784/#2812/#3839/#3848/#3829/#3847の実Source・限定試験記録と、§21の旧計画を照合した**設計・受入差分**である。TFX-01〜07、正式Provider Plugin/実DB Effect、全体BOM、改訂後mainの全対象CI、正式Trait DSL/Schema、Owner意味受理は**今回NOT RUN／NOT ACCEPTED**。
