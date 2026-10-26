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
