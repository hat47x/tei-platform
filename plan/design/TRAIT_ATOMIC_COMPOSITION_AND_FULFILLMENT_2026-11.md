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
