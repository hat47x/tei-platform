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
