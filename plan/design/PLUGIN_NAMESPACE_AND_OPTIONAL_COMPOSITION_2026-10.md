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

ここでいう「信頼されたHost/実行境界」は、独立したTEI専用HostプロセスやWailsを必須にする意味ではない。CLI、Web、Worker、Desktop等のTargetで、実際に外部能力を強制できる責任主体を明示するという意味で使う。

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


---

## 10. 契約互換性：同じ名前・Major・実装宣言でも意味の互換性は証明されない

この節は#951のContract Conformanceへ与える**実Consumer側の検査要件案**であり、新しいCompatibility RegistryやAuthorization Authorityの設計ではない。Namespaceが`tei.std.*`であること、Packageが署名済みであること、Contract Majorが等しいこと、Providerが`implements`を宣言することを、単独で十分条件にしない。

### 10.1 Contractを観測可能な要求の束として捉える

```text
Contract C = (
  identity + exact definition fingerprint,
  input / output / failure / unknown semantics,
  declared effect + scope + context requirements,
  behavioral guarantees,
  conformance / provenance evidence requirements
)
```

具体的な`Lookup<K,V>`のConsumerに対し、Providerが以下を満たせるかを別に照合する。

| 軸 | 例 | 不適合の例 |
| --- | --- | --- |
| 入力 | IDのSemantic Type/Scope、Optional/NULL | `CustomerId`要求へ偶然同じ文字列の`AccountId`を渡す |
| 出力 | 取得件数、欠損・曖昧性、フィールド型 | N件あり得る結果から最初の1件を黙って返す |
| 失敗/不確実性 | `NotFound`/`Rejected`/`Unknown` | 適用後の応答喪失を`Rejected`とする |
| Effect | 読取だけ、外部状態の書込み | read-only Lookupが暗黙にリソース作成する |
| Scope/Context | Workspace限定、必要なContext Slot | 別Application/Tenantへ結果を使い回す |
| 保証 | Snapshot、Staleness、Local Tx等 | `atomic across two stores`をProviderに実力なく宣言 |
| 実現先 | 対象Target・Artifact・現在Authority | Wasm未対応をprocess fallbackで黙って実行 |

型理論上、単純なread-only関数の入力は反変・出力は共変として検討可能だが、可変`ResourceRef<T>`や双方向Portを自動的に共変とはしない。呼出しの不変条件、失敗と副作用、Identity範囲を含む振舞い互換性は単なる構造的Type Shapeより強い要求である。

### 10.2 候補選択と利用許可は異なる判定

```text
contract candidate identified
  -> structural / semantic reference checks
  -> producer publication Authority
  -> provider conformance Evidence evaluated
  -> requested target / binding compatible
  -> current runtime Authority and physical capability check
  -> selected generation pinned on invocation
```

各段階で証拠が不足した結果は`Unknown / NotQualified`として保持し、成功と読み替えない。特に、一つのFixtureへのConformance Test PASSをすべての外部データ・負荷・提供元に関する正しさへ拡張しない。正当なPublisherが後日Contractを追加しても、Consumerの採用済みFingerprintや互換性Evidenceを置き換えずに勝手に再選択しない（#951）。

### 10.3 変更を分解して差分検査する

| 変更 | 候補判定 | 理由 |
| --- | --- | --- |
| 実装Artifactのみ更新 | Contractは保持できる可能性 | Artifact/Build/Runtimeの検証は再実施 |
| 必須Input追加 | 既存Consumerは原則再検証 | 提供できない入力を暗黙既定にしない |
| Optional Field追加 | 条件付きで互換 | 厳密Schema・シリアライズ・Consumer検査による |
| Outputの値域縮小 | 振舞い仕様と契約に依存 | 結果の一部が出なくなるとConsumerの期待を破る場合 |
| Effectの追加（read→write） | 原則自動互換にしない | 実行権限と副作用境界が変わる |
| Strong consistency→eventual | 保証を弱める変更 | 名前・形だけで互換としない |
| Contractの`tei.experimental`→`tei.std` | 新Identityとして比較 | 明示Migration/Adapter/履歴とConsumer採用が必要 |
| 論理Instanceの参照先変更 | Meaningが同じでも再検査 | Scope、Data ownership、Runtime Generation、Source Revisionが変わる |

UI/Graphの座標など表示専用差分は意味的な契約変更としない。他方、見かけ上は同じ項目名でも対応するSource/Relationや有効範囲が変われば、意味的変更として扱う。

### 10.4 最小追加負例と採用判断

- **C1**: 同じMajor/Package Versionでも、Failure SemanticsやEffectのContract Fingerprintが変わった候補を自動採用しない。
- **C2**: `Lookup`が読取として宣言されているのに実際は副作用を伴うProviderのConformance不能/隔離不足を隠さない。
- **C3**: Artifact digest一致だけでPublisher Authorityの失効や現在権限を合格させない。
- **C4**: Read consistencyを強から弱へ変更するAdapter差替えを、単なる`implements`宣言で通さない。
- **C5**: Logical Instanceの配置・設定を変えてもContract定義が変わらない場合、Source/Binding/Runtime readinessだけ適切に再検査する。

C1〜C5はいずれも**受入候補で未実行**。実装が必要になった場合も、#951の既存Resolver・Conformanceと#943の既存Artifact/Lockへ最小の差分を加える方針を守る。振舞い互換性の一般的な完全証明をTEIの目標とせず、可能な静的検査・実Consumer試験・外部Providerの独立観測・残る未確認を区別する。


---

## 11. Coreに残すのは「合成が成立するための機構」まで（2026-10-27 追加仮説）

### 11.1 Kernel minimality の判定原則

ある概念をTEI Coreに組み込むかどうかは、その概念の利用頻度ではなく、**分野固有の型を知らない合成器が、それなしではPort/Instance/Generationを安全かつ決定的に検査できないか**で判断する。

| 横断機構として残す候補 | プラグイン側へ置く候補 | 単独で保証できないこと |
| --- | --- | --- |
| Contract/型参照、入出力要求、Candidate選択と不適合診断 | `tei.std.data`のDatabase、ChangeSet、Schema/Mutation | DBのTransactionと永続データの正本性 |
| 論理Instance、Port配線、依存Scope・Generation/寿命の追跡 | `tei.std.identity`のUser/Principal/Session/Tenant | 実際の本人性やTenant隔離 |
| 宣言上の一時Effect/要求Contextの機構 | 個別プラグインが定義するRead/Write/Publish等の意味 | OS隔離、APIの許可、業務補償 |
| 参照の型・範囲、未解決/曖昧性を保存する検査機構 | Field/Entity/Relation/Expressionの業務意味 | 未受理Sourceの真正性・Entityの唯一性 |
| 必要Target向けLoweringと既存信頼境界への接続 | UI、Batch、Workflow、Migrationの機能 | あらゆるTargetでの同じ実行保証 |

ここで`Effect`は多義的である。ADR-0032の**一時的な逆作用/解放の追跡**は横断機構だが、`db.write`や`network.send`の業務上・物理上の意味は追加の契約と実行境界が判断する。すべての副作用をKernelへenum登録しない。仮に一般的な宣言属性で必要なEffectを表す場合も、静的宣言は実行時の強制ではない。

**反証条件**: 独立した二DomainのConsumerで、利用されないBusiness Typeの登録を新たに強制する、Core enumが増える、第二Resolver/Lock/Authorityが生まれる、あるいはコンパイル後の独立Targetが不要な管理Runtimeを同梱するなら、抽象の境界を再検討する。

### 11.2 意味受理は「すべてのGraphに必須」ではない

`CSV Rows -> Normalize -> File Sink`のような汎用処理は、型と構成の検証だけで成立しうる。**何らかの業務的意味を主張した**Node/Port/Field/Relation/Identity Mappingに限り、その宣言された根拠・Source Revision・受理状態を照合する。

```text
generic composition:
  typed source -> typed transform -> typed sink
  [canonical business meaning is not implicitly invented]

semantic business binding:
  accepted semantic ref (or explicit unresolved candidate)
      + qualified owner/scope/source revision
      + typed relationship/field/expression
      -> candidate wiring, not executable authority
```

業務意味が無いという理由で未検証の物理操作を許可するわけではない。例えば`File Sink`のOS書込みは実行Adapter/環境の強制範囲であり、業務Principalの有無と独立している。一方、`CurrentPrincipal`を要求する機能は、任意の文字列から認証済みの主体を生成できない。必要な信頼済みContext Slotを契約で宣言する（[#1118](https://github.com/hat47x/tei-platform/issues/1118)）。

### 11.3 複数のグラフを一つの辺の意味へ還元しない

| Graph | 辺の意味 | 主な検査 |
| --- | --- | --- |
| Contract/Service dependency | 提供・要求/実装 | Contract適合、Provider選択、Scope |
| Instance configuration | 既存Instanceへの参照・固定パラメータ | 所有者、Cardinality、Binding stage |
| Semantic relation / mapping | 業務上の参照と項目の対応 | Semantic ID、Relation多重度、Source Revision |
| Value/Data flow | 実際の値の変換・配送 | 値型、欠損/NULL/Unknown、順序/縮約 |
| Execution/Workflow | 処理の前後関係・分岐・反復 | 依存循環、失敗・再試行/補償 |
| Resource/Data ownership | 利用可能な実体、正本と派生先 | Lease、保持/削除、移行と整合性 |

同じUIに描画しても、辺種別によって意味と責務が異なる。相互参照する業務EntityのRelationが存在しても、起動順序の循環として自動的に拒否しない。逆に、値の生成前にその値を必要とする初期化循環を、Relationの循環として見逃さない。画面表示用Node IDは永続的な契約RefやSource locatorへ昇格しない（#1124/#1120）。

### 11.4 決定的な候補検証と、後段の動的証拠

次の記法は**検査の役割を説明する疑似形式**であり、新しいKernel APIの採用ではない。

```text
Given:
  G        = application-owned composition selection
  S        = accepted semantic snapshot where semantic claims exist
  R        = existing provider Registry snapshot
  L        = pinned package / contract / artifact references
  B        = instance-scoped typed bindings

Inspect(G,S,R,L,B) -> { candidate | unresolved | rejected, diagnostics, dependencies }
```

構成上は`candidate`でも、実行可能な判定ではない。Runtimeで要求された外部能力と現在Authorityを有効化時・操作時の必要箇所で確認し、旧Generationのin-flight参照を保持する（ADR-0032、#1118/#1223）。同じ入力Snapshot/選択規則の下で同じ候補/診断/依存集合を返すことと、外部DB・Network応答が毎回同じになることを混同しない。

異なるTarget向けの静的/動的Loweringが同じ意味/契約の観測結果を維持することは受入目標だが、配布物、Protocol、常駐機構、更新保証が同じとは限らない。Targetで存在しない保証を暗黙に補わず、必要なら不適合として返す。これは既存[#1121](https://github.com/hat47x/tei-platform/issues/1121)の担当。

## 12. 実証の優先度と比較可能な効果

### 12.1 一つのKernel候補を三つの独立Consumerへ適用

| Consumer | 必須な型・能力 | 持ち込まない前提 | 独立に観測する結果 |
| --- | --- | --- | --- |
| **A. CSV整形バッチ** | Stream/File Source、Pure Transform、File Sink | Principal、Tenant、RDB、業務Semantic Acceptance | File/Recordの期待値、追加依存、静的成果物のサイズ |
| **B. 顧客Web表示** | User/CustomerのQualified Model Mapping、Lookup、表示Projection | 固定のUser物理表・DB一種類・固定IdP | 同じConsumerで異なるSchema 2件、別Instanceの混線なし、曖昧Mapping拒否 |
| **C. 異種DB移送** | Source/Target、型変換、ID Mapping、Checkpoint、個別Effect Outcome | 暗黙の二相CommitやRollback、固定Primary DB | 件数/照合、途中停止・再開、Applied/Unknown差、正本切替の停止条件 |

AとBが同じRegistry/Bindingを利用しても、Bの業務上のUserとAのファイル処理を一つのSemantic Classへ合併しない。Bは既定のUser Directoryを一つ指しても、Component TypeのSingletonを要求しない。CでDB固有のAdapterを使用しても、変更の意図と各Sourceの実保証を区別し、移行の正式受入は#2143/#2147に委譲する。

**検証の階層**：設計候補、Source/Definition静的検査、合成Reference、独立Consumer、実Provider/DB、対象Targetでの権限/副作用、GUI/運用受入を別のEvidenceとして記録する。ReferenceがPASSでも上位をPASSへ伝播させない。

### 12.2 二つの正本を作らないための計測

追加のGraphType/Binding DSLを採用する前に、既存`composition.Registry`と`reference/composition-io-mapping`の差分でA/Bを表せるかを先に調べる。新しい抽象を導入する場合は以下を旧方式と対比する。

- 2つ目の独立Consumerへ移す際に追加するSource/Mapping/手書きコードの量、同じ意味の二重定義数。
- 未定義/曖昧/型不一致/旧Revisionのそれぞれについて、原因が失われない診断の数。
- 単純バッチに読み込まれる不要Package・サービス・起動処理と、静的実行成果物の大きさ。
- Instanceの共有/個別構成、機密情報の伝播、更新時の再資格確認について、明示的に再検証が必要となる対象数。
- DB移行でのCheckpoint再開の差分と、独立に観測できた不明結果・部分失敗の件数。

**撤回条件**：同じ利用者価値を既存機構で表現できる、型付きGraph導入の記述量が比較対象より増える一方で安全性・再利用性・追跡可能性に観測可能な改善がない、Scope/Authorityを重複定義する、独立Consumer間で共通契約が成立しない場合は抽象を撤回/縮小する（[#1973](https://github.com/hat47x/tei-platform/issues/1973)、[#1067](https://github.com/hat47x/tei-platform/issues/1067)）。

### 12.3 追加する負例（いずれも提案・未実行）

| Case | 条件 | 期待 |
| --- | --- | --- |
| K1 | Principal未導入で純粋CSV変換 | 不要Identity解決なしで候補を構成。外部File出力の実行権限は別 |
| K2 | 顧客の意味を主張しないGeneric Scalar Transform | Business Acceptanceを暗黙要求/偽装しない |
| K3 | UserIdとCustomerIdの文字列表現が一致 | Qualified Semantic Mappingの明示がなければ同一視しない |
| K4 | Semantic Relationには循環、Instance activationには循環なし | Relationの循環を一律にActivation失敗としない |
| K5 | Init Value dependencyに閉路 | 明示的Delay/不変条件なしには未適格 |
| K6 | 同一Typeを二Instanceで異なるDataSource/固定式へ接続 | 個別のBinding、秘密参照、結果/Cacheが混ざらない |
| K7 | Runtime選択Generationは固定、Authorityは途中で失効 | 固定Generationを理由に失効後の効果Commitを許さない |
| K8 | SQLite静的TargetへDynamic hot-swap-only保証を要求 | Targetにその能力がなければ、不足を診断して拒否 |
| K9 | Data MigrationでTarget成功応答が喪失 | 実際の`Unknown`を安全な再照合なしで`Rejected`や`Applied`へ変えない |

本節は**新製品仕様の確定や実Provider受入を宣言しない**。最初の実証はA→Bの小さい二Consumerを選び、既存Referenceで表現できないものが固定できたときだけ追加形を検討する。Cは既存Ownerへ接続する。

---

## 13. 読取Projectionからの更新は自動的に導けない（2026-10-29 追加候補）

### 13.1 何が新しい問題か

複数Sourceから一つの`Customer`や`User`を組み立て、一覧/単票で**読める**ことは、その論理オブジェクトを編集して**書ける**ことの証明ではない。例えば次の区別がある。

| 論理項目 | 読取りの実現 | 編集時に不足する情報 |
| --- | --- | --- |
| `customer.displayName` | RDBの`familyName + givenName` | 変更後の文字列を姓・名のどちらへ戻すか |
| `customer.preferences` | Document DBのJSON path | 当該pathの正本・Version・Patch保証 |
| `customer.orderCount` | Order APIの集約結果 | Countへの逆変換は存在しない |
| `customer.organization` | 複数Relationの探索 | どのMembershipを選択・更新するか |
| `customer.enabled` | Account Store + 外部IdPの派生 | どのOwnerへのCommandが状態を確定するか |

これは、読み取り側の`CompositeEntity`や`Projection`を更新可能なDB行と同じものに見せる際の**view-update問題**である。各Sourceが書込みCapabilityを持つだけでは解けず、どの変更を誰の正本へどう配分するかの業務判断と、外部副作用の保証が必要である。X-TEAの「主DB＋参照用副DB」型の簡便さは既定の**Read Projection**として提供できるが、複数DB更新へ拡張するときはこの境界を越える契約が必要。

DBのView更新には基底関係への変更伝播が一意に決まらない場合があるという研究上の背景がある（[Franconi & Guagliardo, The View Update Problem Revisited](https://arxiv.org/abs/1211.3016)）。また、[双方向モデル変換のLens法則を扱う研究](https://pmc.ncbi.nlm.nih.gov/articles/PMC8645541/)は純粋な更新写像の整合条件を与えるが、TEIの外部DBの原子性や認可を保証するものではない。

### 13.2 読取りと書込みを別契約へ分ける

提案する**概念上の**段階は次の通り。新しい固定Kernel enumや別Resolver/第二Source正本は導入しない。

```text
sources + qualified bindings
   -> ReadProjection<V> + provenance/field ownership
       -> explicit user/domain update intent
           -> optional WriteMapping / owner-routed Commands
               -> Prepare/Validate/Authorize/Commit via existing owners
                   -> per-target receipt and independent readback
```

`ReadProjection<V>`は**既定でread-only**。保存ボタン等を生成するときには、(a) 正本Ownerへの単一Command、(b) 明示的な書戻しMapping、(c) 明示的な多Source ChangeSet/Operation のいずれを使うかを案件側が選ぶ。どれもなければ編集可能UIを生成せず、read-only/unsupportedを表示する。

**単一DBの通常CRUD**は簡単な(a)のTemplateで成立してよい。これは一律に複数DB用のChangeSet/Journal/Sagaを要求する意味ではない。逆に、`readable`というPortだけから自動的に`writable`を推論しない。

### 13.3 更新を伴うMappingには部分的な逆方向契約が要る

純粋な双方向写像を考えると、`get: S -> V`と、元の状態を参照する`put: (S,V) -> Result<S,Conflict>`が候補になる。これは**考察用のモデル**で、実行用のDB一括Mutation APIを定義するものではない。

更新可能な範囲では、少なくとも以下の性質を検査対象にできる。

- **Get–Put**：`get(s)`を変更なしで戻す際、外部正本に不要な更新を発生させない。
- **Put–Get**：`put(s,v)`が成功した後の独立した読取りは、保証された整合性条件の下で利用者が要求した値を再現する。
- **Put–Put（追加の候補法則）**：純粋な写像上で、定義可能な範囲について `put(put(s,v1),v2) = put(s,v2)`。先の変更が後の変更で上書きされるという性質であり、外部DBの再試行や冪等性を保証しない。

ただし、これらは単純な関数Lensの法則を**外部更新の契約候補**へ持ち込む際の比較指標に過ぎない。実DBの更新にはRevision競合、権限、遅延、部分Commit、通信断がある。`put`を宣言しただけで上記が証明済みになるわけではない。永続DB操作の確定は[#1157](https://github.com/hat47x/tei-platform/issues/1157)・[#1174](https://github.com/hat47x/tei-platform/issues/1174)・[#1129](https://github.com/hat47x/tei-platform/issues/1129)の既存Ownerに接続し、下位実行を再実装しない。

例えば、`displayName = familyName + " " + givenName`は読取Projectionとして容易でも、同じ表示名から元の姓名を一意に戻せないため、明示的な分割規則・独立した姓名編集UI・業務Commandなしにwrite-backの候補とはしない。`orderCount`のような集約値は原則として直接更新Portを持たない。

### 13.4 フィールド所有元と更新先の選択

```yaml
# 提案中の擬似構文。読み取りモデルの所有/更新を表す例。
view: example.customer/CustomerSummary
fields:
  displayName:
    read:
      from: profileStore
      expression: displayName
    update:
      mode: owner-command
      command: profileService/UpdateDisplayName
  orderCount:
    read:
      from: orderService
      expression: count(orders)
    update:
      mode: read-only
  newsletterPreference:
    read:
      from: preferencesDocument
      path: $.newsletter
    update:
      mode: explicit-patch
      owner: preferencesService
      expectedRevision: required
```

`profileStore`と`preferencesDocument`が別DataSourceでも、`CustomerSummary`は単一の読取契約として使える。複数の正本に書く操作は**一つのProjectionの逆演算**ではなく、Targetを明示する複数Command/ChangeSetとして扱い、各TargetのCommit/Unknownを保持する。JSON pathが分かっても、フィールドの正本・配列Indexの安定性・削除の意味・Document Revisionが分からなければ更新は認めない。

Read Projectionの取得時点、Source Revision、Projection Definition Revision、Field Ownership、データのStaleness/Cacheの条件は、後続のWrite Intent検証に必要な場合だけ参照する。これらを全部一つの「Version」に潰さない。Cacheから読んだ値のRevisionが更新先で保証されないときは、適切な条件付き更新・再読取り・競合提示が必要になる。

### 13.5 Data migrationと更新Mappingの共通点・非共通点

通常編集とMigrationは、Source/Target Mapping、型変換、Qualified Identity、対象Revision、Read/Write Capabilityの検査を再利用できる。ただしMigrationは、大量データを横断するCheckpoint、履歴・削除、再開、正本所有者の切替、Source/Targetの差分照合が必要なため、通常の単行`put`契約とは別である。

移行先での単純なupsertを「読取Projectionの逆写像」と呼ばない。ID生成を伴う移行では対応表の永続性が必要であり、削除の伝播も案件で決める。#2143/#2147の既存Ownerの切替/照合条件を保持する。外部の二DBを1つのトランザクションとして扱うことをTEI Coreで保証しない。

### 13.6 受入候補（NOT RUN）

| Case | Given / When | Expected |
| --- | --- | --- |
| W1 | 姓＋名の表示だけを定義し、結合後の表示名を編集 | 逆写像やOwner Commandなしではread-onlyとして拒否 |
| W2 | 注文件数の集約Projectionを編集 | 更新先を推測せずunsupported |
| W3 | Document JSON pathを変更するが期待Revisionが古い | Providerの条件付き更新でConflictを返し、成功としない |
| W4 | 2つのSourceが`displayName`を返しField Ownerが未決 | 暗黙の先勝ち/後勝ち更新を拒否 |
| W5 | 単一SQLiteで編集 | Local Owner Commandのみで動作し分散Coordinationを必須にしない |
| W6 | 複数DBへ変更後片側がApplied/片側がUnknown | 部分結果と再照合要件を維持、無条件再送しない |
| W7 | 別Tenant/Semantic Scopeの同名CustomerIdを表示 | 自動結合・更新・Cache共有しない |
| W8 | 同じProjectionのSource Revisionが更新直前に変わる | 現在の前提に再資格確認し、古いRead ReceiptをCommit権限にしない |
| W9 | Source→Target移行でID採番を含むCheckpoint後の再実行 | 対応IDを安定させ、同じ旧レコードから新IDを二重生成しない |

**比較する効果**：単一DBの標準CRUDを複雑化せず、RDB＋Document DBのRead Projectionを作れるか。書戻し先の曖昧性を拒否できるか。既存のQuery/Mutation/Source Mappingを重複実装せず、2 Consumerの宣言量、追加コード、拒否診断、実データでの独立確認を比較する。既存`#1127`のCRUD/生成、`#1129`の業務副作用、`#2143/#2147`の移行、`#951`のProvider適合を独立Ownerとして維持する。

**撤回条件**：既存の明示Command＋Read Projectionだけで必要な編集操作を自然に実現できる場合、一般的な双方向`Lens`型の新設は撤回する。逆写像を自動生成しても実業務の曖昧さを解決できない場合は、型としての「書戻し可能性」を固定Coreに導入しない。実装/実DB/GUIの検証は未実施。
