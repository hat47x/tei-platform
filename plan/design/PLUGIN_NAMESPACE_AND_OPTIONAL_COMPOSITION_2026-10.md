# TEI管理名前空間と任意Capability合成の設計候補（2026-10-26）

- **状態**: 提案中。名前の変更、Registry/Runtime/GraphType実装、契約の採用や移行の完了を意味しない。
- **要件正本**: [RQ-005 / RQ-008 / RQ-009](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)。
- **先行の採用判断**: [ADR-0032](../adr/0032-composition-kernel-replacement.md)（Contract中心の合成Kernel）、[ADR-0033](../adr/0033-core-and-plugin-subprojects.md)（Core/Plugin管理境界）。
- **既存担当**: [#951](https://github.com/hat47x/tei-platform/issues/951) Contract identity/definition Authority、[#1016](https://github.com/hat47x/tei-platform/issues/1016) 案件Composition、[#1251](https://github.com/hat47x/tei-platform/issues/1251) サブプロジェクト、[#943](https://github.com/hat47x/tei-platform/issues/943) Build/Artifact/Lock、[#1067](https://github.com/hat47x/tei-platform/issues/1067) 案件間の共有・昇格、[#2091](https://github.com/hat47x/tei-platform/issues/2091) 案件Overlay。
- **関連設計**: [Plugin入出力対応](PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)。データベースのN:Mと複数DB更新はRQ-005を正本とする。

## 1. 背景・現在との差分

既存のRQ-008は標準名前空間を `std.*` と書いていた。一方、TEIが所有する正式なルートを `tei.*` とすれば、`tei.std.*`、`tei.experimental.*`、`tei.contrib.*` を同じ管理領域で区別できる。旧Referenceには `std.message` / `std.localization`（[#1649](https://github.com/hat47x/tei-platform/issues/1649)）や `tei.migration.atlas` のような別のIDも存在する。新方針を理由に既存IDを無言で置換・再発行しない。

ADR-0032は、Service ContractをPackage/物理Artifactから分離し、固定CapabilityKindの将来制約を撤回済み。#951はContract定義Publisher AuthorityとProvider Conformanceを区別し、同一majorだけでは互換性が証明できないことを指摘済み。本設計は**それらの第二Resolver・第二Authority・第二Lockを新設しない**。本設計だけで現行Productionが移行済みになるわけではない。

## 2. 名前空間・識別子の提案

### 2.1 管理領域

| 領域 | 意図 | 留意点 |
| --- | --- | --- |
| `tei.std.*` | 標準の契約・型・任意の標準実装 | 導入必須という意味ではない |
| `tei.experimental.*` | 試行中の契約・実装・検証素材 | 名前だけで安全/不安全とは判定しない |
| `tei.contrib.*` | TEI側が管理・委任を受け入れた寄稿 | 委任単位と公開元の真正性を別途確認 |
| `com.example.*` 等 | 案件・第三者の管理領域 | TEI公式を詐称できないようにする |

`experimental`は安定性の傾向、`contrib`は配布・保守の由来に関する分類であり、直交する。安定性、互換性、保守者、公開Authorityは別のメタデータでも表す。名前空間は型の継承関係でも、実行許可でもない。

### 2.2 五つのIdentityを分離

- **Namespace ID**: 公開名の管理領域。
- **Contract ID**: Capability/型と、その公開契約・互換性世代。
- **Package ID + Version**: 配布・依存解決の単位。
- **Implementation / Artifact Identity**: 契約実現と正確な成果物Digest/Buildの来歴。
- **Logical Instance ID + Configuration/Activation Generation**: 同じ型から独立に生成される個別構成と実行世代。

例えば`tei.std.data/Lookup@1`は提案中の読みやすい**表記例**で、採用済み文法ではない。正本では`package` / `symbol` / `contractVersion`を別項目にした構造化参照を評価する。表示AliasはCanonical IDを変更しない。Contract Version、Package SemVer、Artifact Digest、Instance Generation、Source Revisionを同じ「バージョン」として扱わない。

契約を`tei.std`が所有していても、実装は`tei.contrib.*`や案件のPackageに属してよい。複数のContract Majorは明示的に共存可能とし、名前の近さ、最高Version、Package列挙順、`implements`の自己申告だけで適合を確定しない。

### 2.3 公開Authorityと検証

Namespace Publisher Authority、Contract definition Authority、Provider Conformance issuer、Artifact integrity/publisher、Compositionの選択/Acceptance、現在のExecution Authorityを分離する。名前空間の委任をRuntime権限の委任とみなさない。失効・更新の現在性を、古いLockに記録したDigest一致だけで代替しない。

Registryのオンライン存在をTEI Coreの前提にしない。Local Package、検証済みRegistry、オフラインLock/CASによる取得は候補だが、どの経路でも名前と実Artifactの一致、未解決・競合・未許可発行者を診断する。詳細なTrust root・委任・失効は#951/#943の担当で、今回確定しない。

## 3. Coreと任意プラグインの境界

TEI Coreが担当するのはADR-0032で決めたContract選択、Scope/Effect/Generationの合成上の不変条件と信頼された実行境界への接続であり、次の業務上の概念を**必須の組込みデータ型・構成要素にはしない**。

`Principal`、`Account`、`Tenant`、`Session`、`Database`、`Transaction`、`ChangeSet`、`Migration`、`UserDirectory`。

それぞれが必要なら、`tei.std.identity`、`tei.std.data`等の任意の契約とProvider、または案件独自の型・実装を選択する。CSV変換などPrincipal不要の最小Batchは、Identity Packageを読み込まず構成・実行できる。これを理由にOS権限・Sandbox・Resource許可・Host現在性の検査を省略しない。Runtimeの認可強制は案件業務Principalの存在を前提にしなくてもよい。

論理Instanceの多重生成、参照による共有、Port Cardinality、Configuration BindingとOperation Binding、Resourceの寿命と外部データの保持を別々に扱う。単一の既定Providerは案件のRole Bindingであり、Component TypeをSingletonへ固定する理由にはしない。

## 4. 意味的BindingとGraphの利用例（未採用の構文案）

X-TEA等の個別システムで異なるユーザテーブル名、項目、関係に、再利用可能なUserDirectory Componentを接続する場合、要求を物理`String`名だけで表さない。

| Port例 | 型の候補 | 案件側のBinding |
| --- | --- | --- |
| userEntity | `EntityRef<User>` | 固定Entity参照 |
| userKey | `KeyMapping<User>` | 単一/複合項目 |
| displayName | `Expression<User,String>` | 項目参照・純粋式 |
| organization | `Relation<User,Organization>` | 型付きRelation path |
| currentUser | 案件が選んだContext Contract | 認証を利用する案件のみ |

Data Model上の`FieldRef`と個別レコードの値、UI view-node IDとQualified Semantic Ref、Role/Principalと業務上のUser/Customerを混同しない。同名や同じ局所IDだけから別Scopeの同一性を推測しない。対応がAmbiguous/NotFound/Unknownのときは唯一候補として扱わない。

GraphTypeの型引数、固定値、Provider Instance参照、式は別種のBindingとする。有限なGraph Generatorは固定入力Snapshotから決定的な候補Graphを生成し、展開後にPort/Scope/依存を再検証する。これが成立するかはPoCで反証する。Stage/Execution ContextやGraphTypeの完全な型文法はここで確定しない（#1016/#1121/Plugin入出力対応設計へ接続）。

## 5. 複数データソースは任意のData Capability合成

RQ-005は既にApplicationとDatabaseのN:M、Environment Binding、transaction/failure boundary、複数DB更新の部分成功とUnresolvedを要求する。本設計ではDBをCoreに追加しないことだけを明示する。

- `defaultDataSource`、参照DB、移行元/先などは案件のRole Binding。物理Primary/Secondaryの固定Typeを強制しない。
- RDB、Document DB、APIの`Lookup`/`Query`/`Mutation`能力、Composite Entity、Relation/Field所有元、ChangeSet、同期・移行は必要なプラグインで追加できる。
- Read-only参照とWrite Capabilityを分離する。跨DBの書込みは単一Transactionの原子性を暗黙に保証しない。各参加先のCommitted/Rejected/Unknown、回復・再照合・冪等条件は更新Strategyが明示する。
- 外部保存データの移行はGraphのRename/Rebindとは別であり、必要なSource/Target ID対応、Checkpoint、照合・Cutoverを明示する。
- 単一SQLite等の最小ケースでは高度なCoordination/Identityプラグインがなくても成立する。

Graph Editor/Inspectorは複雑なBindingを投影できるが、表示座標をSemantic identityへ昇格させず、正本のAcceptance/Runtime readinessを別に追跡する（#1124/#267/#1121）。

## 6. Namespace移行・後方互換の境界

既存`std.*`、`tei.migration.*`、`std.message`等について、文字列の全置換や旧IDへの新型の暗黙Aliasはしない。まずUsage/Source/Lock/Consumer/Artifactの棚卸しを行い、旧→新の**明示的MigrationまたはAdapter**と互換性証拠を要求する。既存の採用・受理済み意味、Plugin Lock、Sourceを自動的に書き換えない。旧契約・旧実装の履歴を削除せず、参照時にLegacy/Proposedを区別する。

`tei.experimental`から`tei.std`への昇格はContract単位で判断する。実装PackageとContractの昇格を同一視しない。昇格時もContract Identityの変更と意味上の互換性を別途判定する。案件差分の保存・三者比較は#2091へ委ねる。

## 7. 最小の反証可能なReference / 受入案

| Case | 成功/失敗の期待 | 主担当 |
| --- | --- | --- |
| N1 | `tei.std.*`を第三者Authorityが無断公開しても選択しない。authorized newerとunauthorized newerを区別 | #951 |
| N2 | 同じContractを複数実装が提供する場合、選択Policyなしに先頭候補を採用しない | #951 |
| N3 | Contract Major、Package Version、Artifact Digest、Instance Generationをそれぞれ検査・固定 | #951/#943 |
| N4 | Identity/PrincipalプラグインなしのCSV Batchが構成・実行可能 | #1251/#1016 |
| N5 | 同じComponent Typeの2 Instanceに別々のEntity/Field/Expression Bindingを与えても混線しない | #1016/入出力設計 |
| N6 | 異なるScopeの同名Field、複数候補Relationを唯一参照へ昇格しない | #1120/#1016 |
| N7 | 単一DB、既定DB＋read-only副DB、複数DB ChangeSetの境界と部分成功を区別 | RQ-005既存のデータ担当 |
| N8 | `std.message`等の既存Referenceを自動改名せず、旧Contractを参照できる | #1649/#951 |
| N9 | 実験Contractを標準に移すとき、明示Migration/互換証拠がなければ自動切替しない | #1067/#951 |

これらは**未実装・未実行の候補**。N7の実DB分散原子性、N1の本番Publisher Trustは参照テストだけで証明できない。

### 効果・撤回条件（#1973）

- 効果: 新しい業務Capability追加にCore enum変更が不要、独立Instance/Mappingの再利用、Contract Ownerと実装Ownerの分離、最小BatchにIdentity依存なし、既存の型付きLock/権限の非二重化。
- 比較: 既存機構で同じ二Consumerを構成した場合の追加宣言量、曖昧性診断、変更差分、再現性を測定する。
- 撤回/縮小: 新Manifest/Registry/Graph種別が既存機構の二重正本となる、実Consumerで差異を保存できない、移行で旧IDを安全に保てない、導入コストが再利用効果を上回る場合は新抽象を採用しない。

## 8. 現時点の未決と反映先

- 完全修飾IDの表記文法、文字種、名前正規化、`tei.contrib`委任粒度、発行者Trust root: **未決**。Contract identity/Authorityは#951へ。
- Package VersionとContract Majorの対応、同一Major内のSemantic compatibility、既存`std.*`の移行一覧: **未決**。#951/#943へ。
- GraphTypeの関数的生成、静的/動的Binding、Resource Ownership、Effect型の詳細: **未決**。#1016/#1121/既存入出力設計へ。
- Principalを必須としないPoC、RDBとDocument DBを跨ぐChangeSetやMigrationの実接続: **未実装・未実行**。RQ-005/RQ-009および既存Ownerの受入を利用し、新しい第二実装を作らない。
- リポジトリの`std.*`一括置換、公式Registry運用、業務意味のAcceptance、本番Authority付与、既存PRのmergeは**非目標**。

本文は設計候補であり、人による採用判断と関連OwnerのEvidenceを経るまで、ADR-0032の採用済み境界やProduction仕様を置き換えない。


---

## 9. 深化検討（2026-10-27）：Type / Instance / Bindingを最小の実Consumerで反証する

### 9.1 何を新設しないか（現行Referenceとの差分）

[入出力対応Reference](../../reference/composition-io-mapping/README.md)は既存`composition.Registry.InspectWiringSet`を利用し、契約項目の型・単位・多重度・機密区分・Semantic TypeRef候補・出力順序を限定的に検査する。一方、実Sourceの意味受理、任意式の評価、動的値、実Invocation、正本への適用は未実証である。この差を埋めるために第二Provider Resolverや新しい「全用途対応Graph IR」を先に作らない。

[案件Composition設計](PROJECT_PLUGIN_COMPOSITION_DESIGN.md)の既存Role/Extension/Overlayと、[呼出し文脈](CAPABILITY_INVOCATION_CONTEXT_DESIGN.md)のSemantic Request / Context Slotの分離を再利用する。案件がPrincipalを利用しないことと、実行時のHost-owned context/OS境界が存在しないことを同一視しない。各`requires`が宣言したContext Slotだけが必要であり、`Principal`を全Invocationの暗黙必須項目にしない。

### 9.2 Type引数とInstance参照は異なる

`GraphType<Store : BlobWriter>`の`Store`は**実装可能な契約/型の制約**であり、`storeRef: invoiceStore`は**既に構成された論理Instanceへの参照**。この区別を維持する。

| 観点 | 型の特殊化 | Instance Binding |
| --- | --- | --- |
| 入力 | 契約/Component Type | 特定のLogical Instance |
| 意味 | 型制約を満たす実装群の選択 | 個別の設定・所有者・Namespace・Lifecycleへ接続 |
| 多重度 | 一つの型から多数Instance可能 | 共有/専用は明示 |
| 切替 | 契約/型の互換検査 | 設定・参照範囲・Runtime Generationの再検査 |

`S3Store`を型引数にしたから同じBucketが選ばれるわけではない。逆に、`invoiceStore`と`archiveStore`を同じS3Store Typeから作っても、Bucket、CredentialRef、Data ownership、Scoped Leaseが異なりうる。`singleton`は型の普遍性ではなく、明示的に選択したLifecycle/Provider Policyによる制約候補とする。Shared Instanceを複数Application/Tenantで使う場合、単なる文字列Scope一致やProvider再利用でデータ・権限隔離が成立したとしない（[#2110](https://github.com/hat47x/tei-platform/issues/2110)、[#2020](https://github.com/hat47x/tei-platform/issues/2020)）。

### 9.3 設定Bindingの型と時点を分ける（候補）

`Binding<T>`は以下の概念的な**供給元の分類**を持つが、直ちにKernel固定Enumへしない。正確な型名/構文は未決。

| 供給元 | 例 | 必要な検査 |
| --- | --- | --- |
| Literal | 固定値、設定名 | 許容値と型。機密値をLockへ埋め込まない |
| Semantic reference | 受理済みUser Entity / Field / Relation | Qualified Ref、Source Revision、Semantic Type、Cardinality、所有元 |
| Pure expression | 氏名フィールドを連結して表示名へ写像 | 入力型、欠損/NULL、構文版、演算子の意味、非副作用性 |
| Explicit Provider output | 設定Providerから得る論理識別子 | Provider/Contract選択、結果版、観測時点・失効 |
| Invocation-specific input/context | 要求ごとのCustomer ID、必要な場合だけ認証情報等のSlot | 宣言されたSlotか、実行時の信頼境界、Cache Partition |

評価の時点（Source authoring、Composition/Resolve、Activation、Invocation、Observation）と、値の所有者・改訂・Secret性は別軸とする。ある時点で「提供できる」ことと「後の実行時にも有効」であることを混同しない。Provider outputやCredentialなど未取得の値は`Unknown`であり、空文字やゼロへ変換しない。

**拒否条件**: Activation時の固定ParameterがInvocation単位で変わる式を密かに参照すること、式に任意の更新やネットワーク呼出しを埋めること、`FieldRef<User,String>`の代わりに未検証の文字列列名を意味参照として使うこと、未受理のSource/異Scopeを現在のBindingへ流すこと、欠損/NULL/未確定を既定値で黙って埋めること。

**既存の区別を保存**: 一時的な`Effect Scope`の逆順Cleanup（ADR-0032）と、DB/APIへ適用した**永続的な業務Mutation**は別の意味である。後者を前者の`dispose`で自動取消できるようには扱わない。型検査・静的宣言は実行時の外部能力の許可・隔離に代わらない。

### 9.4 X-TEAの差を使った具体的なGraph Binding候補

同じDirectory Contractを持つ二つの案件で、モデルが異なる例を考える（架空のSchemaでありX-TEA原典の表名・動作の断定ではない）。

```yaml
# 議論用の擬似構文。既存 .tei/parser では受理できない。
componentType: example.identity/UserDirectory
contract: tei.experimental.identity/UserLookup@0
requires:
  entity: EntityRef<User>
  key: KeyMapping<User,UserId>
  display: PureExpression<User,String>
instances:
  - id: backofficeUsers
    type: example.identity/UserDirectory
    bindings:
      entity: {semanticRef: "example.erp/User@1", revision: "r10"}
      key: {semanticRef: "example.erp/User.userNo@1", revision: "r10"}
      display: {expr: "concat(lastName, firstName)", operatorContract: "example.expression/Concat@1"}
  - id: customerUsers
    type: example.identity/UserDirectory
    bindings:
      entity: {semanticRef: "example.customer/CustomerAccount@1", revision: "r21"}
      key: {semanticRef: "example.customer/CustomerAccount.accountId@1", revision: "r21"}
      display: {semanticRef: "example.customer/CustomerProfile.displayName@1", revision: "r21"}
```

`display`へ渡すPureExpressionと、外部DataSourceから`CustomerProfile`を取得する操作は同じではない。後者が別Service/DBを要するなら、`Lookup` Provider/Relation ResolverへのPortを明示し、Read Effect・Consistency・Cache Scope・Failureを契約として分離する。

`UserId`と`CustomerId`は表現が両方文字列でも同一と仮定しない。関係が1:NやAmbiguousなら、`Resolution`の明示的結果を保ち、唯一のUserへ縮退させない。UIのView Node IDから直接Source locator/Authorityへ渡すこともしない。

### 9.5 合成の検証境界を二段階にする

```text
accepted semantic / explicit project bindings
    -> qualified reference + contract/shape/owner inspection
    -> declarative connection candidate (NOT executable)
    -> existing contract/provider/lock resolution
    -> target-specific lowering + current authority + activation readiness
    -> pinned invocation -> per-source effect outcomes / independent observation
```

(1) **Candidate validation**: 型、識別Scope、個数、式の入力、依存循環、Bindingの所有/改訂を判定。未解決は`Unknown`で止める。これだけでRuntime Readyとしない。

(2) **Executable qualification**: 正しいProvider/Artifact、現行Authority、Activationに必要なContext、Target対応、SourceとProviderの鮮度、InvocationごとのGenerationを検査。前段の成功をこの段階の権限としない（#1121/#1118）。

Query・Mutation・Transfer・Migrationは共通Port接続を**再利用**できる一方、Result/Failure/Checkpointや実行保証は独立Contractで扱う。任意Graphの循環と、初期化で値を必要とする依存循環と、業務Relationの相互参照は異なる。計画的なフィードバックループは明示的な遅延/境界なしに許可しない。

### 9.6 複数DB Update / Data Migrationの既存Ownerを再確認する

[RQ-005](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)のN:Mとtransaction/failure boundaryを維持。[#2143](https://github.com/hat47x/tei-platform/issues/2143)にはSnapshot時点とChange Log位置の対、主従の正本切替、逆同期、項目所有者、Collision Queueの不足が、[#2147](https://github.com/hat47x/tei-platform/issues/2147)にはCheckpoint/Reconcilation/Cutover Gateの不足が既に実例とともに記録されている。これらを新しい`tei.std.data`実装のために二重実装しない。

データ操作の`Plan`、`Applied`、`Unknown outcome`を別に表現し、個別更新の原子性・冪等性・再送条件はProvider能力に依存する。`ChangeSet`は意図の表現案であり、ひとつのトランザクションや二相Commitを保証する意味ではない。逆同期や正本切替はデータの所有/権限の変更なので、人のAcceptanceと観測のGateを通す候補とする（#2143/#2147）。異種DBの実動作は未確認である。

### 9.7 既存Referenceと不足を分けた追加受入案

| Case | 反証する仮説 / 負例 | 既存証拠または不足 |
| --- | --- | --- |
| G1 | 同じComponent Typeの二Instanceを別々のEntity/Key/式へ接続し、Runtimeも設定が混線しない | 型・多重度の部分検査は既存Reference、実Binding/Runtimeは未確認 |
| G2 | 固定ConfigへInvocation依存式を入力すると構成時に拒否 | 評価時点と式の規則は未確定 |
| G3 | Optional/NULL/Missing/Unknown/Unavailableを式とMappingで区別 | Referenceの恒等スカラー例は既存、実演算子接続は未確認 |
| G4 | 同じ名前だが異なるScope/型のUserIdを接続しない。Ambiguousは候補のまま | Semantic TypeRef候補検査は既存、発行Authority/Source受理は未確認 |
| G5 | `tei.std.identity`を導入しないBatchのSource→Transform→Sink | 未実装の代表Consumer試験 |
| G6 | 型付きContractにConformantな外部実装でも、不正なRuntime Authority/Artifactを拒否 | Registry試作の宣言・試料と実Runtime権限を区別 |
| G7 | 二DB更新で片側`Applied`・片側`Unknown`を全成功にしない | RQ-005既存要求。異種DB実動作は未確認 |
| G8 | SourceスナップショットとCDC位置不一致、切替前未解決差分で止める | #2143/#2147に既存。試験は既存Ownerへ |
| G9 | `tei.experimental.*`から`tei.std.*`への変更でProviderや保存済みLockを暗黙更新しない | #951/#1920、移行実体と回帰は未確認 |

**最小検証の順**: G5（ゼロIdentity）→ G1/G2/G3/G4（型付き二Instance）→ G6/G9（Contract移行）→ G7/G8（既存DB移行Ownerへ接続）。各ケースでSource Revision、Package/Contract、Provider/Artifact、Target、結果、未検証を独立記録する。新しいKernel TypeやRegistryを追加する前に、既存Referenceの拡張で可能な範囲を実測する。

### 9.8 現段階の採否

- **維持する採用済み判断**: ADR-0032のContract中心/一時Effect/Generation、ADR-0033のCore/Plugin境界、RQ-005の複数DBと保証の区別。
- **今回具体化した候補**: 型パラメータとInstance参照の分離、Bindingの値供給源・評価時点・Source Revision、2段階の合成判定、独立Ownerへのデータ移行の委譲。
- **未採用・未実装・未実証**: `GraphType<T>`文法、`Binding<T>`の型定義・Expressionの文法/評価、Instance再利用の隔離、`tei.*`正式ID構文、Registry移行、異種DBの更新原子性、Runtimeでの実Consumer検証。
