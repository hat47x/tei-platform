# Plugin機能の実現・安全な最適化に必要な契約（2026-11-06、設計/要件候補）

- **状態**：製品要求RQ-008/RQ-024/RQ-032を具体化する提案。製品用Optimizer、ABI、実Runtime、Plugin Manifestの新schemaを採用済みとしない。
- **固定調査基準**：tei-platform main `3b57a3a05a32be44f2cfead38f015a66cdc15f70`。本稿はGitHub上のIssue/ADR/要求を静的照合して作成した。Go/Node/実Target計測/独立Consumer受入は未実行。
- **採用済み境界**：[ADR-0032](../adr/0032-composition-kernel-replacement.md) Contract/Provider/Scope/Generation、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md) 独自Host非必須、[ADR-0036](../adr/0036-delivery-forms-as-plugin-composition.md) Deliveryの合成、[ADR-0044](../adr/0044-authentication-not-a-core-concern.md) Identity/認証非Core。
- **既存Owner**：[#951](https://github.com/hat47x/tei-platform/issues/951) 契約/Resolver/Conformance、[#1016](https://github.com/hat47x/tei-platform/issues/1016) Profile/Instance選択、[#1121](https://github.com/hat47x/tei-platform/issues/1121) Target Lowering、[#1223](https://github.com/hat47x/tei-platform/issues/1223) 必要TargetのGeneration/Lifecycle、[#1251](https://github.com/hat47x/tei-platform/issues/1251) Cohesive Bundleと未使用実装除外、[#1097](https://github.com/hat47x/tei-platform/issues/1097) 複数論理Plugin→一物理Artifact、[#2600](https://github.com/hat47x/tei-platform/issues/2600) 増分再利用と性能、[#2390](https://github.com/hat47x/tei-platform/issues/2390) 外部Hostの軽量Consumer、[#2504](https://github.com/hat47x/tei-platform/issues/2504) 実Script/Plugin Resource隔離、[#2899](https://github.com/hat47x/tei-platform/issues/2899) Everything-is-Plugin比較研究。
- **対象外**：全Target共通の常駐TEI Host、全Pluginを一Process/一Wasmにする必須規則、新しい中央Cost-based Resolver/Runtime Scheduler/Plugin Store、ContractのMeaning正本・Owner受理・Runtime Authorityの第二実装。

## 1. 問題を「実現」と「最適化」に分割する

同じService Contractでも、静的Goへのリンク、Java/.NETの組込みLibrary、Wasm Component、独立Process、外部API、ビルド時生成コードなど異なる**実現**がある。Plugin Package/Bundleの境界、Component Type、Logical Instance、Provider実装、Builder、物理Artifact、Deployment、Invocation/Effectは一対一である必要はない。

一方、**最適化**は既に選択・適格性が明確な構成に対して、Target条件を守りながら計算/起動/配布/配置/Resourceの費用を変えることである。安価だからProviderを無断採用すること、性能のために業務MeaningやCurrent Grantを緩めることは最適化ではない。

最低限、評価対象を以下の5層に分ける。

| 層 | 入力・参照 | 成立させること | 結果だけでは証明できないこと |
| --- | --- | --- | --- |
| Contract requirement | 契約版、型/Null、Result/Failure、Effect、Scope、必要Context | Consumerの要求が明示される | 候補Providerの適合、実行許可 |
| Implementation/Selection | Candidate Provider/Artifact/Target能力と既存Resolver | 一意・明示的に選択、Unknownの保持 | 実Targetの可用性、性能優位 |
| Realization/Build | 選択済みInstance、Builder/Toolchain、依存閉包、実Artifact member | Targetが実ロードできる形へLowering | 元のMeaningを保持した実行、利用時権限 |
| Runtime/Operation | 必要ならTarget Host、Context、Resource、実Entry、Current Grant | 実在する操作・作用の強制 | 成功応答からの業務効果・原子性 |
| Optimization evidence | Baseline/候補、独立Expected、計測環境/負荷/実Byte | 契約を守り、計測対象で費用が改善したか判定 | 未計測Targetや未来負荷での普遍的性能 |

### 実現の選択順は「適合性→実体→費用」

対象Consumer/Targetが必要とする保証を満たす候補を選び、その中で構成・Artifact・実行境界を確定し、はじめて費用を比較する。単一の`bestProvider`や負荷時の自動Provider差替えを汎用正本にしない。候補が一意でない場合は#951/#1016の既存選択Policyへ返し、Cost順位が暗黙の契約Resolverにならないようにする。

## 2. 要求候補（正式API名ではない）

| ID | 要求 | 最低限の観測・拒否 |
| --- | --- | --- |
| PRO-01 | 同一Contractに対する複数の物理実現と、Target/Host能力の条件付き選択 | static linked / independent processを同じContractで比較。unsupportedなTargetで黙ってfallbackしない |
| PRO-02 | Package/Bundle・論理Instance・物理Build Artifactの独立 | Cohesive Bundleを認め、未選択Providerを実Binaryへ暗黙収録しない。共有ArtifactでAuthorityを共用しない |
| PRO-03 | 型・振舞い・Effect・Scope・失敗/Unknown・整合性を保つ最適化 | 同じ入出力型でもNull、順序、精度、時点、エラー、副作用が異なる変換を拒否 |
| PRO-04 | 静的化・部分評価・Fusion/Pushdownなどの候補は前提条件を明示 | Pure・変更不変・exact pinの証拠がない式やProvider処理を事前計算/移動しない |
| PRO-05 | キャッシュ・増分再利用は依存閉包が証明できる範囲のみ | 未解決依存、Schema/Meaning/Role/Toolchain/Scope変更をfalse hitしない。旧Data/Grantの現在性を再利用しない |
| PRO-06 | 並列化・Batch・Retry・再順序化は効果境界に従う | 書込みの一度限り、順序、競合、Idempotency/Receipt不明なら等価変換と称さない |
| PRO-07 | Instance/Lifecycle/ResourceはTargetに応じた最小構成 | 静的CLI/IoTへWatcher/Routerを強制しない。長寿命TargetのDrain/Generationを無条件省略しない |
| PRO-08 | 性能条件はTarget/Workload/計測環境/施策ごとに固定 | 正しさPASSと性能改善PASSを別記録。数値目標はOwnerが事前設定する |
| PRO-09 | 最適化後もContract→Provider→Build/Artifact→実Invocationへ逆追跡 | 1 Artifactに複数Contractが収録されてもMember/Entry/権限を個別に追う |
| PRO-10 | 開発者・AIによるPlugin追加/最適化は差分と反例で比較可能 | 実装の追加を効果としない。独立二Consumer/二Realizationで採否し、効果不足なら縮小・撤回 |

各要件は本設計の**差分候補**。既存RQ-008/RQ-024/RQ-032、ADR-0032/0043が先に定める禁止事項を置換しない。

## 3. 性能改善で契約を壊しやすい変換

| 変換候補 | 使える可能性がある条件 | 不適合/不明として止める例 |
| --- | --- | --- |
| 未使用実装の除外 / static linking | 正確なSelected/Build-consumed closure、実Link結果 | Go blank import/initやglobal registrationにより実際には利用される実装 |
| Build時部分評価 | Input/Operator/Role/contract fingerprintが固定、Pure、時点非依存 | 現在時刻/Secret/Runtime Tenant/外部Provider応答をCompile時に凍結 |
| Query pushdown | NULL、Collation、型変換、Limit/Order、Snapshot/ACL、FailureがConsumer要求を満たす | 取得範囲・順序・認可がSourceへ移動した結果変わる |
| Transform fusion / streaming | Boundary、入力順、backpressure、flush、部分失敗の帰属が保持される | MFTの取込み/公開Receiptを中間Transform成功で代用 |
| Memoization / Artifact cache | 完全な依存閉包、Target/Scope/Source/Rule/Builder/Artifact pin | Unknown Dependency、旧Source、共有Cache経由のTenant越境 |
| Retry/Parallel map | Pure/Idempotent、処理順の非依存、許可された上限と中断時の結果 | DB書込みや外部配送で未知Outcomeを無条件再送、順序依存の処理を並列化 |
| Shared Process/Artifact co-location | Build/Load版の一致、各Instanceの隔離・Resource Budget | Artifact同梱を理由にPlugin Aの現在GrantをBへ広げる |
| Dynamic hot-swap | 対象が長寿命、Generation pin、drain/切替/旧State処理を実測 | 単一静的Go Binaryへ常駐Routerを追加、再起動だけで無停止と主張 |

**原則**：ある変換の静的な適用可能性は、未知の外部Effectが無いことの証明ではない。Opaque Providerについて自動Pushdown/Fusion/再順序化を許すには、そのProviderのConformance、実Host/OS境界、必要ならTarget固有の独立Oracleを要求する。一般に完全な振舞い同値は自動証明できないため、`proven / conditional / not-qualified`等の説明用区分で範囲を記録し、新しいCore列挙を追加しない。

## 4. 計測と採否

**同じMeaningを実現する二つのConsumer**（例：PrincipalなしのCSV変換、Qualified User Lookupを利用するWeb/外部Host）と、実際に選択した二つの物理方式を最小比較対象とする。要件の違いで両方を同じ方式にする必要はない。さらに1件の副作用付き負例（Document更新または配送Attempt）で危険な最適化を拒否する。

計測票は、(a)固定Contract/Source/Binding/Target、(b)Toolchain/ABI/Builder、(c)独立Expected/Failure/Effect、(d)実Link/Load Byteと権限、(e)入力規模・並列度・cache状態・warmup・測定回数、(f)冷/温起動・初期転送量・memory・CPU・Resolver/Load overhead・p50/p95・総費用、(g)最適化の前後差分、(h)Unknown/NOT RUNと撤回判断、を区別する。

**判定の順序**：
1. **Safety/semantic gate**：同じ契約で対象Bounded Oracleに対する正負/Effect試験を満たし、現在Authorityが強制される。未証明なら性能比較で勝っても採用しない。
2. **Physical gate**：実際のSource closure、Build、ArtifactSet member、Loaded Byte、Targetの配置を固定。ManifestやDiscoverabilityだけで実物混入/除外を主張しない。
3. **Performance gate**：Owner指定のBudget（startup、memory、transfer、latency等）をTarget別に設定。baseline・candidateを同条件で反復し差分と分散を報告。閾値未定なら優劣を「測定値」として報告し、自動採用しない。
4. **Adoption/rollback gate**：負荷やTargetに依存する改善である旨を残し、静的Targetでは再Build/再配布、長寿命Targetでは必要なGeneration切替で導入。永続Effectのrollbackを機能交換と同一視しない。

## 5. 反証可能な最小試験（いずれも提案・未実行）

| Case | 初期状態・変更 | 期待 |
| --- | --- | --- |
| OPT-01 | 未選択ProviderをCohesive Bundleへ追加 | Discovery候補は増えてもSelected Build closureと実Byte/初期副作用は不変（条件付き） |
| OPT-02 | 同Contractをstatic GoとProcessの二方式で構築 | Consumer意味が一致し、Target別コストを別計測 |
| OPT-03 | Pure式をBuild時に畳み込み | 同Input/Role/Operator pinで結果一致。時刻・外部依存は事前評価不可 |
| OPT-04 | PushdownでNULL/Collation/Sortが異なるProviderへ移動 | Targetの意味要求を満たさない場合不適合 |
| OPT-05 | 一つのArtifactにProvider A/Bを収録 | A/Bの論理ID/Lock/Grantが独立。B未許可でAのみ利用 |
| OPT-06 | Invocation時のTenant/Scopeが変わるCache hit | Cross-scope共有を拒否、必要な現在Read/Grantを省略しない |
| OPT-07 | Opaque書込みProviderでRetry/並列化提案 | Idempotency/Receipt/順序保証が未証明なら拒否 |
| OPT-08 | 長寿命Targetだけv1→v2を切替 | 旧in-flight pin/drainとCurrent Authorityを維持。静的TargetへRouterを同梱しない |
| OPT-09 | Candidateのp95は改善するが失敗/Effectが変化 | 性能優位でSafety gateを上書きしない |
| OPT-10 | Sourceを同Byte移動・未選択Role変更・実Builder入力変更 | 意味再計算省略とTrace/Artifact再資格化を別計測 |
| OPT-11 | 実測p50のみ改善、p95や最大MemoryがTarget Budget超過 | 採用Policyを満たさなければ不適合 |
| OPT-12 | 外部Hostから最小Libraryを利用 | TEI Studio/CLI/Registry常駐不要。必要Import/権限/実Loaded Byteを確認 |

## 6. 優先順・作業の分配

| 順位 | 小さな成果 | Owner / 再利用 |
| --- | --- | --- |
| P0-A | 既存Contract＋一つのProviderをstatic/link版とProcess/外部Host版で表現し、Targetごとの必要条件と不足を比較 | #951 #1121 #2390 #1145 |
| P0-B | 未選択Pluginの物理除外、Selected closure、同Bundle/別Artifact/共有Artifactの境界を実測 | #1251 #1097 #1167 #1089 |
| P0-C | Unsafe optimizationの3負例（Pushdown NULL、Cache Scope、effectful retry）を現行Fixtureで拒否 | #951 #2504 #2600 #1174 |
| P1 | 安全なFusion/部分評価/増分再利用を一つずつ実用Consumerへ適用。各TargetのBudgetと実測差を採取 | #2600 #1063 #1130 #2018 |
| P2 | 長寿命TargetのRuntime更新、異種Provider pushdown、分散実行で制約付き最適化を比較 | #1223 #2622 #2827 #2390 |

P0の最初に新しいPlanner、Scheduler、ABI、Optimizer用DSLを作らない。#2899のDSH比較研究は参考にするが、DSH Runtime/Plugin ManagerをTEIへ逐語移植しない。実装者/生成AIの負担削減を主たる採用価値として測り、構造の抽象度が増えても変更量や障害発見能力が改善しなければ採用しない。

## 7. 未決と責任の所在

- **未決**：正式な実現候補のManifest/Wire形式、Resource/Cost model、適用可能な演算子の純粋性証明、Target別Policyの決定主体、最適化施策の自動選択範囲。既存Contract Shape/Lockで不足が確認されたら最小追加とする。
- **採用済みのアーキテクチャ**：Contract中心、Host非必須、Target別Delivery、意味/実現/Authorityの分離。
- **未実装・未実行**：本書に書いたPRO-01〜10とOPT-01〜12の横断試験、複数Target実性能比較、Runtimeの新しい機能。
- **停止線**：Pluginを増やすこと、Bundleを小さくすること、シミュレーションで速いこと、Manifestで「verified」と主張することは製品最適化の受入証拠ではない。exact source/Target/Loaded Byte/独立Oracle/実Budgetと、人間Ownerの採用判断が必要。


## 8. 機能を実装するまでの最小ループ：再利用・合成・生成・特殊化を比較する

既存の[#1067](https://github.com/hat47x/tei-platform/issues/1067)は二つの独立Consumerでの重複が確認されるまで共有Pluginへの昇格を急がない。[#1445](https://github.com/hat47x/tei-platform/issues/1445)は開発工程Plugin、[#2620](https://github.com/hat47x/tei-platform/issues/2620)は標準構造へのRole付与、[#2077](https://github.com/hat47x/tei-platform/issues/2077)は生成器の別言語展開を担当する。この節ではこれらをつなぐ**Plugin機能の最短実現手順**だけを要件候補として明示する。

1. **要求を固定**：必要なService Contractと対象Target/Scope/Effect/Failure/Resource Budgetを定義する。業務Meaningの必要な場合はAccepted Sourceへ戻り、通常の純粋バッチには業務Principal/Modelを強制しない。
2. **既存候補を発見**：標準・案件・外部Providerと、その実装の正確な契約、実現Target、配布・Conformanceの差を比較。Registryの候補順、人気、最新版やAIの説明文だけから自動採用しない。
3. **合成で充足するか検査**：必要な能力を既存の複数Instance/Port/Expression/Adapterで表せるなら、専用Plugin実装を作らない。新しい隠れたHookやGraph外の工程を増やさない。
4. **不足だけを生成/実装**：再利用できない能力についてのみ生成AIや通常開発で小さなProvider/Adapterを作る。必要なHost API・入力・実装Source・Build Target・契約バージョン・生成器を固定し、出力を差分としてレビューする。生成AIや生成器の自己申告をConformanceとみなさない。
5. **独立に適合を検証**：型・振舞い・Scope・Effect・実Target/Load Byte・Permissionを分け、代表の正例だけでなく失敗/Null/Timeout/Unknown/拒否の負例を確認。生成CodeのBuild PASSはContract適合の証拠ではない。
6. **必要なら最適化**：上記Safety Gateを通過した候補だけに部分評価/静的化/融合/Cache等を適用。Target別Budget、利用頻度、冷温起動、運用/保守コストに照らして効果を測る。計測不十分なら元の単純実装を維持する。
7. **導入・更新・撤回**：選択したComposition/Lock/Artifact/Authorityを既存Ownerで固定。実Producer/Consumer差分を観測し、採用後の変更・失効・取戻し方法を明確にする。必要に応じてContractだけを標準へ昇格し、Implementationの公開主体とPackageの所属を同一視しない。

### 容量を増やす「継承」より、実際の能力の合成を優先する

例えば、NAS/Filesystem型の保存機能とS3等のObject Store型の保存機能には、Byte Objectの読み書きという共通の能力があっても、原子的なRename、Directory、部分更新、一覧・整合性、転送結果不明時の再試行の意味が一致するとは限らない。

したがって、`S3Storage extends NASStorage`のような型名の包含だけで全操作を代替可能とせず、`BlobRead`・`BlobWrite`等の共通契約に加え、`AtomicRename`など条件付き保証を必要なときだけ要求する。これは特定の継承表現を禁止するものではなく、**利用者が要求する振舞いを満たした能力だけを代替可能と判定する**ためである。必須Facetを増やす前に二Consumerで再利用の利益を検証する。

| Case | 追加の最小反例 | 期待する判定 |
| --- | --- | --- |
| IMP-01 | 既存の二Providerを組み合わせれば実現できるが、新専用Plugin生成を提案 | 既存Compositionとの宣言量・性能・保守費を比較し、必要性なしなら生成しない |
| IMP-02 | 既存FS利用Consumerが原子的Renameを要求するがObject Store候補はそれを提供しない | Read/Write等の共通能力のみ適合、Renameを含むContract全体への自動代替を拒否 |
| IMP-03 | AI生成Adapterが型検査を通るがNULL/Timeout/Unknownを正常値へ変換する | Semantic/Failure Conformanceを拒否。生成器の成功を製品資格としない |
| IMP-04 | 2 Consumerが同じ抽象を必要とせず、一方だけ固有Pluginを使う | 共通Pluginの昇格を保留し、案件側の小実装を許す |
| IMP-05 | 最適化により起動時間は短縮するがBuild閉包が不明、別ScopeのCache結果を流用 | 速度優位でも採用不可。実Input/Scopeの検査と現在のAuthorityが必要 |

**停止線**：IMP-01〜05は未実行の受入案。実AI生成・Providerの二Target接続、実導入、性能改善、業務Owner受理はNOT RUN。別の『Plugin開発Wizard』『自動Plugin生成器』『汎用Adapter Registry』をP0で新設しない。


## 9. 呼出し粒度の最適化：Transport / Data / BusinessのBatchを同一視しない（2026-11-06 追補）

### 9.1 既存Ownerと追加する問題

[Capability Invocation #1145](https://github.com/hat47x/tei-platform/issues/1145)は意味Request・実行Context・Transportを分離し、[非画面Batch #1130](https://github.com/hat47x/tei-platform/issues/1130)は長時間処理/取消/途中結果/再実行、[SmartDB #2622](https://github.com/hat47x/tei-platform/issues/2622)はProviderごとのbatch/atomicity、[ETL/MFT Draft #2827](https://github.com/hat47x/tei-platform/pull/2827)はMaterialization/Relay/Import Unitと部分結果を担当する。#2504はResource上限、#1174はCurrent Effect/Receipt、#2390は外部Hostの最小Adapterを所有する。**第二Batch Runtime、統一Wire Format、汎用Job Scheduler、全Targetへの常駐TEI Hostは作らない**。

この節は、呼出しオーバーヘッドを減らす手法が**元のService Contractと個々のEffect/Requestの保証を保てる条件**だけを横断要件として扱う。以下は実装APIや固定Core enumの提案ではない。

| 粒度 | 最適化の例 | 変更してはいけない意味 |
| --- | --- | --- |
| Transport Batch | N回のRPCを1回のEnvelopeへまとめる | Request/Result/Scope/Grant/Timeout/Failureの区別、原子性の非推論 |
| Data Batch | N行をまとめたVector/Columnar/Chunk演算 | Row cardinality、重複、NULL、順序と集計粒度、入力Snapshot |
| Streaming | Pull/Pushで順次送受信、Backpressure | 取得/確認済み範囲、終了の完全性、取消後の未確定Item |
| Business Batch | N件の更新を一つの業務操作とする | Commit境界、部分成功、Retry/補償、実Ownerの意味判断 |

**Transportでの一括送信≠Transaction**、**Streamの停止≠外部Effect全件取消**、**Job Completed≠全MemberをConsumerが独立Read済み**。通常の単発Read-only InvokeにBatch/Streaming Contractを必須化しない。

### 9.2 変換適格性の必要条件

**Transport Aggregate**：元の単発Operationを一括送信しても、要素ごとの要求とOutcomeの相関が失われないこと。対象Operation/Instance、Contract/Provider/Artifact/Target/Context、Request/Attempt、Scopeと各要素のOutcome分類を照合可能にする。相関metadataは#1145のDispatch/Receiptと#1174の権限・効果へ委譲する。一つのEnvelopeにしたからといって、異なるTenant/Secret/Grantを合算しない。個別Timeout、部分Failure、返信順序が未知なら同値最適化として採用しない。

**Data Chunk / Stream**：Chunk境界/順序変更、Cardinality・NULL・Precision、Read Snapshot/Watermark、重複/欠落と全件完了をConsumerが要求する意味で検査する。無制限Bufferや無制限Fan-outを低メモリのStreamingと呼ばない。実Host/AdapterがBackpressure/上限を強制できない場合は `unsupported/unknown` とする。生成/送出/受領/独立Consumer読込/業務受理を別区間として扱い、途中取消を「全件取得済み」とせず、部分入力を受け付けないReducerを誤って成功させない。Sort/LIMIT/RLSでの0件をSource全体の不存在としない（#988/#2582）。

**Effectful Batch / Retry / Cancel**：書込みを一括化する場合は実ProviderのCommit境界を明示する。Idempotency keyの宣言と重複効果を実際に防ぐことを混同しない。部分Commit/ACK lost/Unknown後は#1174/#1130の現在Authority・Receipt照合・Recoveryへ接続し、無条件再送しない。取消要求は外部の既Commitを取り消した証拠ではない。元の不可分な業務操作をChunk化の都合で分割確定したり、独立した業務更新をTransport都合で原子的だと宣言したりしない。

### 9.3 性能の測定と閾値

各Target/Workloadの `dispatch/serialization/host-crossing`、`batch setup`、`per-item transform`、`queue/buffer wait`、`transfer`、`independent readback` を可能な範囲で区別する。概念的には個別呼出しの費用を `N × (L + C)`、Batch呼出しの費用を `L_batch + N × C_batch + queue + verification` と見積もれるが、L/CをTarget非依存の定数とは仮定しない。

Batchを大きくするとThroughputが改善しても、最初の結果までの時間、p95、Peak Memory、取消応答、失敗時の再実行費用が悪化し得る。**Throughput、Queue時間、p50/p95、Peak Memory、最大in-flight、部分失敗/取消後の残存Effect**を別途測り、事前のOwner Budgetを守る。Batchの自動閾値決定は実測後の候補とし、単発呼出しが妥当なTargetへ共通Schedulerを追加しない。

### 9.4 追加の反証ケース（全て提案・NOT RUN）

| Case | Given / When | Expected |
| --- | --- | --- |
| INV-01 | Read-onlyの10単発Invokeを一Transport Batchへ置換 | 各Request/Resultの値・失敗・Scopeを独立照合。1 RPC成功≠10件意味成功 |
| INV-02 | 2 TenantのReadを同Envelopeへ集約 | 個別の現行Grant/Contextを検査、Scope混合を拒否 |
| INV-03 | Batch内2件成功・1件Unknown | 全体成功/失敗へ丸めずOutcomeを保存 |
| INV-04 | 100件Streamが60件で途中取消 | 生成・送信・受信・取込み・Commit済みの範囲を別観測 |
| INV-05 | Backpressure不能なProcess Adapterで無制限Buffer | 低メモリ保証付きStreamへの適合を拒否 |
| INV-06 | Pushdownで順序/NULL/Collationが変わる | Consumer要求が不一致なら不適格 |
| INV-07 | 書込み5件のTransport BatchがACKを喪失 | 実Effect Unknownを保持し無条件再送しない |
| INV-08 | 独立Transaction5件を一括atomicと表示 | 実ProviderのCommit境界が未証明なら原子性を宣言しない |
| INV-09 | Single-record Fast Pathは直列の方が速い | Batchを強制せず同条件のLatency/Memory/CPUを比較 |
| INV-10 | Chunk変換Artifact Digest一致、外部ConsumerではMember一部未読 | 内容一致とReaderの全Member受領を分離 |

**検証順**：まず#1145/#2390の単発TransportでINV-01/03/09を比較。#2622の実Providerがある場合だけINV-06/08、#1130/#2827の実Batchがある場合だけINV-04/07/10を接続。Resource上限とEffect Fenceは#2504/#1174の実証範囲を越えない。

**撤回条件**：単発Invokeで既に予算を満たす、Batch効果が小さい/逆効果、正しい相関・部分失敗・取消を維持できない場合、最適化案そのものを撤回する。**新しいBatch Wire Contract、Scheduler、Streaming ABIは独立二Consumerの不足が実証されるまで作らない**。
