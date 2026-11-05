# Data・Function・Businessの意味整合と移植・変更管理の受入引渡し（2026-11-05〜06）

- **状態**：横断監査の差分を既存Ownerへ渡す計画候補。業務意味の採用、Requirement Probe・Definition/Instanceの正式仕様決定、製品Gate合格ではない。
- **対象**：#1125 Business Procedure、#1126 Data/Relation/Key、#1129 Data Rule/Lifecycle、#2619 Everything is Data。移植・DB Migration・組織Relation・Pluginは既存担当へ接続する。
- **照合の基点**：main `8aa18ca150f8689e9875d2e0319fb20e704086f5`（2026-11-06取得）。以降のmain/PR変更は再照合する。実装状態は[docs/status.md](../../docs/status.md)、製品全体の資格は[#1119](https://github.com/hat47x/tei-platform/issues/1119)が判断する。
- **位置づけ**：四Issueの要求を複製しない。先行する[#1125 BP-05](ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md)、[#2619横断](https://github.com/hat47x/tei-platform/issues/2619)、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)の**意味がずれ得る継ぎ目**と、最小反証・実証の順序だけを記録する。

## 1. 根拠の強さと原典帰属

| 区分 | 確認した範囲 | 使ってよい主張／禁止する昇格 |
| --- | --- | --- |
| **A：過去会話のユーザー発言を検索で直接確認** | 2026-03-08：三要素の業務モデルは業務間の単純な実行順序でなく、業務とデータソースのDFD・起動契機・疎結合を重視 | TEIへの設計要求として扱う。外部方法論の著作者による正式仕様・文面と断定しない |
| **B：既存会話要約とリポジトリの確認済み記録** | 2026-08-23の生成ID・PK方針、および2026-10-24のHost方針は既存の会話要約を参照（原文の逐語再読は今回未実施）。#1125/#1126/#1129/#2619、[EmptyPolicy統合PR #2849](https://github.com/hat47x/tei-platform/pull/2849)、[実PG限定Command PR #2853](https://github.com/hat47x/tei-platform/pull/2853)、[ABA負例PR #2854](https://github.com/hat47x/tei-platform/pull/2854)、[組織E1候補 #2681](https://github.com/hat47x/tei-platform/pull/2681) | 各固定版の実装・限定試験・計画の状態を区別する。PR mergeや合成PASSを旧業務意味・現在のOwner受入へ転記しない |
| **C：今回の整合性推論** | 下記の分離原則、横断受入順、各Ownerへの引渡し | 監査上の補正候補。正式なDefinition/Instance共通型、Requirement Probe製品機構、Capability付与、原設計者の主張として採用しない |

**出典限界**：過去会話の検索でユーザーの明示的な説明を確認したが、凍結履歴の全本文や外部原著者の一次資料を逐語確認したものではない。「Requirement Probe」「Definition/Instance」はTEI研究での候補語彙として扱い、外部方法論の原著者が同一用語で公刊したと主張しない。非公開X-TEA/XEAD原本を転記しない（#941）。

## 2. 意味がずれやすい七つの境界

| 対象 | 維持する最小の区別 | 不適合となる例 | 主担当 |
| --- | --- | --- | --- |
| Requirement Probe | 疑問・判断候補・回答根拠・影響あり・明示受理を分ける。前提変更でAnsweredからImpactedへ再確認を促す | 影響検出をOwnerの回答撤回・新Meaningの自動受理に変える。第二Decision Registryを作る | #2619横断、#1125/#1126/#1129が個別Decisionを消費、#267 Acceptance |
| Definition / Instance | Relation/Rule/Procedureの意味定義と、個別の所属Occurrence・評価Attempt・業務実施を分ける | 全DefinitionへOperationID/Receipt/物理Hostを必須付帯し、通常の専門家記述に実行ノイズを露出 | #2619（標準構造）、#1126/#1129（適用粒度） |
| Data / Function / Business | 共通データ表現で記述できても、Data＝業務識別・関係、Function＝目的・入出力、Business＝Actor・DFD・契機・手作業/引継ぎの責務を保つ | Businessを自動実行Workflowへ、FunctionをCRUDへ、Dataを物理Tableへ一律縮約 | #1125/#1126/#2619 |
| Business Key / physical PK | 業務Keyの構成・Role・Tenant/時点、Occurrence、物理PK/順序、UI row keyを分ける | FK/DDLの一致、単一Generated ID、Column同名から業務同一性・所有Relationを自動採用 | #1126、#2622 Provider |
| Procedure / Workflow | DFD/Task/契機・手作業と、選択されたWorkflow/Script IF/状態遷移の実行を分ける | 手作業・外部確認を自動Transitionへ変換。StateMachine/共通State Storeを全案件に強制 | #1125、#1129 |
| History | モデル/Decision、業務値・状態、組織Membership valid/recorded time、Workflow Attempt、Effect Receiptを分ける | 現在値A→B→Aを「何も変更なし」とみなす。後日のReadで歴史的Commitを取消扱い | #1126/#1129、#1174 |
| Host Capability / Plugin | 宣言したCapabilityとProvider実装、選択されたBundle/Module、Build利用、現在Host GrantとEffectを分ける | Plugin selected＝実行認可、未使用Bundle module＝必須Runtime、AIの候補＝実Grant | #951/#1016/#1251、#2619が境界参照 |

**過剰抽象の棄却条件**：専用State Engine／Meaning Registry／Requirement Probe Engine／Identity Registry／Workflow中央表／新Hostを導入しないと表現できないと主張する前に、既存Data/Role/Relation/Rule/Context/Operation/Projectionで最小例を記述する。機構追加による新しい実用能力と独立反例を示せない場合は提案を撤回する。

## 3. 移植ゲートはP・A・Eと実行資格を分ける

[既存の一試行移植計画 Draft #2783](https://github.com/hat47x/tei-platform/pull/2783)の責務を変更せず、BP-05を最小判別例として用いる。

| 観測の継ぎ目 | 必要な実証・停止条件 | 既存Owner |
| --- | --- | --- |
| **P：Parity**（原Sourceと旧実際） | 旧Data Script/Hook/Function/DFD/Relationと、旧Driverで本当に実行された範囲、独立Expectedを別列に置く。旧Sourceだけでは旧Runtime発火はunknown | #950/#940/#1125 |
| **A：TEIへの安全な適合**（定義→Instance） | 受理済みOrder O1の対象明細集合L、空集合Policy、Rule P、保存済み値と未保存Patchのpost-image、実Provider読取のRow/Field completenessを結ぶ。データ取得不足をRule違反や成功に変えない | #1126/#1129/#2506/#2622、BP-05 |
| **Function / Command**（Instance→操作） | 同じAttemptの選択O1、編集Session、typed Intent、実Request byte/Payload、実Target/DB Alias、current Grant、ExpectedVersion、子Relation membershipを束縛する。O1検証PASS→Payload O2は不合格 | #2597/#2598/#1174 |
| **Effect / Recovery** | Commit前RollbackとCommit応答不明を区別。同一OperationIDのReceipt Lookup→実Targetに束縛した独立Readを確認。Replayを新しいWriteと表示しない。別OperationIDでのABA再適用も検証が必要 | #1174/#1129/#1119 |
| **Business / Consumer** | Receipt・Readが一致しても、外部Handoff/手作業、Ownerの業務完了、現在の開示資格、Windows GUIでの理解・操作受入は別 | #1125/#267/#2516/#2391 |
| **E：Extension** | 他Target・非同期Job・高度Workflowなどは必要性が出たとき独立検証。P/Aの前提へ強制しない | #2619/#1119 |

**版別の実証上限**：#2849のAction emptyPolicyは統合済みの構文/意味表現であり、BP-05全対象Lの実Read保証や業務完了を自動受理しない。#2853は製品Invoker→実PG CAS/模擬ACK喪失/Lookup/独立Readの限定証拠、#2854は新冪等KeyでABA後に再作用できる実PG反例。いずれも旧Driver同義、二Function共有Rule、組織E2→E4、Windows実GUI、Owner意味受理を証明しない。

**最初の縦切り**：BP-05第3規則のみ不成立→認可/Command非呼出を既存S1で試し、保存済みA(1)と未保存B(0)の提案後状態を対照。次に同じRequest/Provider/Receipt/Readを繋ぐ。実PG局所成功を合算して製品Gate G0〜G4合格としない。

## 4. DB移行：物理変更と業務同一性の維持を別資格にする

旧/新ClientとRead/Writeの共存を、次の既存Migration Provider・検証経路で扱う**計画ゲート**として整理する（新Migration Engine/共通Schemaは作らない）。

| 段階 | 独立に確かめること | 停止条件とOwner判断 |
| --- | --- | --- |
| **Expand** | 旧ClientのRead/Write保持、新定義と実Schema/catalog/Constraint・接続設定の差 | 旧Clientの書込不能・NULL/DEFAULT/制約の意味破壊、未受理Targetへの切替を停止 |
| **Backfill** | 業務Key（例：tenant＋orderNo）、物理PK、Relation occurrence、valid/recorded history、再開Checkpoint/冪等性・独立集計 | キー衝突/重複・欠落、履歴破壊、不明な中途再実行なら停止 |
| **Validate** | 独立Expectedによる旧新Clientの同じOperation意味、取得集合/NULL/空、CAS/Receipt、対象・時点付き独立再読取 | 移行成功ログ・件数一致・Schema Diffだけでは成功にしない |
| **Cutover** | Reader/Writer route、Host/Grant、旧新同時稼働、Query/SQL Realizationの固定、現在Target/rollback window | 固定済みDB専用SQL不在から共通SQLへ無音fallback禁止。Commit不明は同ID照会が先 |
| **Contract** | 旧Client停止・排出、負の依存/DDL削除、History/外部Consumer再読込、戻し不能域 | 旧Clientまたは読取Consumerが残る／過去Historyを失う場合、破壊的削除を保留 |

**MIG-COMPAT-01（既存議論の最小Fixture名、NOT RUN）**：同一業務Key `(tenant=T1, orderNo=O1)` に対して、旧Clientと新ClientがExpand以降共存する間に更新・再読取を交互実行する。Backfillを中断→同Checkpointで再開→二重実行しても、物理PK変換・Relation・履歴・金額/NULLの意味が独立Expectedと一致するかを検査する。Cutover直後のCommit応答喪失では**同じOperationIDのLookup**と別Invocation Readを優先し、旧Clientの書込資格消失・SQL Realizationの固定失効・戻し窓終了を別々に記録する。旧/新のどちらかが一件の書込みに成功したことを5段階完了へ昇格させない。過去のMG-01〜10案はこのケースの検査項目候補にとどめ、新しい製品Gate/テスト正本をこの文書で作らない。

実DBでの実行／切戻しは**本計画ではNOT RUN**。再実行はSource/Schema/Mapping/Checkpoint同一性を証明できる場合だけ。逆Migrationが安全でない場合は停止し、Owner承認のForward-fix候補を提示。DDL適用と業務移行完了は別。既存[#2862 DB別Query](https://github.com/hat47x/tei-platform/pull/2862)、[#2622 Data Operations](https://github.com/hat47x/tei-platform/issues/2622)、[#1126 Data/Mapping](https://github.com/hat47x/tei-platform/issues/1126)、[#1174 Effect](https://github.com/hat47x/tei-platform/issues/1174)へ分担する。

## 5. 組織RelationとPlugin Bundleの独立反例

**組織Caseの読取**：Person Pに二つのMembership occurrenceがあり、D1/D2とvalid/recorded期間が異なる例を使う。所属Role、職位Occupancy、Case責任、見せてよいScopeを同じRelationへ統合しない。#2681のGo/CLI限定E1・[#2389組織計画](https://github.com/hat47x/tei-platform/pull/2389)を参照し、実ProviderのRelation Traversal、対象集合完全性、RLSと現在GrantはE2以降として独立検証する。Case C1のD1→D2変更は同じTransaction/Version/Operation/Receiptと独立post-imageの検証が必要。組織E1、合成what-if、実PG一般Commandの成功を組織固有のE2→E4として加算しない。

**Plugin選択と利用**：高凝集なModule群をBundleにまとめることと、全Moduleを選択・コンパイル・実行することは別。 `available / selected / resolved / function-used / build-included / invoked / authorized`を区別する。#2681のManifest外の未宣言Source root非干渉テストは、その範囲の限定実測にとどまる。Catalogにある未選択、選択済Bundle内の未使用、Go単一Binaryへの実取込み、現在Host Grantの不変はそれぞれ未実証。純粋Operatorは標準ライブラリ/静的実装で足りる場合を残し、Data Providerは境界が必要な場合、Target Generatorは配布差分がある場合にPluginを検討。Owner業務規則をPlugin化すること自体を目的化しない。

## 6. Issueの受渡し、証拠の記録と終了条件

| Owner | 最小の追補・反証先 | 引渡しだけで済ませるもの |
| --- | --- | --- |
| **#1125** Business | DFDのTask/Actor/Event/manual/handoffを実行Workflowから分離。BP-05のL/EmptyPolicy/Pと、提出・手作業・外部Handoffの完了を別判定 | #1129のScript/Command実装、#1174のReceipt |
| **#1126** Data | 複合業務Key/physical PK/Role/FD/期間付きRelation、組織Occurrence、旧新SchemaによるMeaning/Historyの維持。Graphの縮約表示から更新対象を自動確定しない | #2622実DB ProviderとMigration apply |
| **#1129** Rule/Lifecycle | Rule Definition・Attachment・評価時点/Phaseと対象Instanceを分離。旧Script PとTEI適合A、Go/PG効果とBusiness完了を別資格にする | #940旧Runtime、#1174実Txn、#1125業務意味 |
| **#2619** 横断 | Everything is Dataは表現/変換の共通化とし三要素の役割を保持。Requirement Probe/Definition-Instanceは候補語彙、Owner Decision/実Capabilityの第二正本を増やさない | #1119既存四最小縦切り、#951/#1016 Plugin合成 |

**最小検収パケット**：同じSubject/Source revision/Operation Attemptの(a)原Source＋独立Expected、(b)Data Key/Relation/時点、(c)Business Procedure/契機/手作業、(d)Function Data Context/Rule input/未保存Patch、(e)Provider実Read/Row・Field completeness、(f)typed Request/Payload/Grant/Target/ExpectedVersion、(g)Commit/unknown/Receipt/別invocation read、(h)現在Consumer開示・Owner業務判断を**独立列**で保持する。DB移行と組織の別試行の成功をこの一Attemptへ接ぎ木しない。

**次の最小作業順**：①read-onlyで三要素/Key/Role/時点の意味ずれ反例を固定→②既存BP-05/S1および組織E1の局所実経路を再利用→③#1119のB(Command/DB)で一Operationの実Provider/Receipt/独立Read→④旧新Client・DB移行段階の実観測とOwner受理→⑤Windows GUI/SHOなどのConsumer。全面的な直列依存にはしない。

**停止線**：出典確認・静的解析・synthetic Go/SQLite・実PostgreSQL・Windows GUI・Owner意味受理を分ける。Requirement Probe/Definition-Instanceの外部原典帰属、旧Driver互換、実組織DB、移行5段階の実適用、Plugin未使用Moduleの実Binary非干渉、二Provider同一業務Function、Owner受理は**この文書では未証明**。Docs-onlyであり、新Rule/Host/Registry/Gate/DB Migrationの実装変更はない。
