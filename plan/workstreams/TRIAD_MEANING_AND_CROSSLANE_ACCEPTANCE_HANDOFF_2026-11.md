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

**認可の限定的な実証更新（2026-11-07）**：[main統合PR #3037](https://github.com/hat47x/tei-platform/pull/3037)／[実試験記録](https://github.com/hat47x/tei-platform/blob/main/plan/evidence/XTEA_LOWCODE_COMMAND_AUTHORIZATION_2026-11.md) により、`legacyauthorization.CommandAuthorizer` の付与ありExecutor到達／未付与の`COMMAND_UNAUTHORIZED`・Executor非到達／規則評価不能を拒否と区別、のGo 3件が限定PASS。**Pattern/メニュー宣言はGrantではない**ことを一つのReferenceで実証した。一方、`EpochReader`による実版照合、権限の正本・付与管理、対象Resourceの解像、実Provider/DB・本番Host、Owner業務意味受理は未検証。[未統合PR #3038](https://github.com/hat47x/tei-platform/pull/3038) は付与Readerと宣言の照合候補であり、現参照ツールに必要な能力宣言・動的Resource解像まで完了した証拠ではない。#1125のBusiness Actor/承認判断、#267のMeaning受理と、ここでのCommand認可を同一状態へ縮約しない。

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

<!-- tei-triad-derived-approval-cut-20261106 -->
## 7. 11/6横断追補：派生値を業務の確定事実へ昇格させる「時点」を明示する（計画・未受理）

本節は §2 の Data/Function/Business 境界を、**見積試算→承認→受注確定→取消**という用途別の値の意味へ適用する最小差分。業務段階と Owner 完了の既存詳細は [#1125の11/6受渡し](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6681840353)、DFD階層の境界は [同Issueの別追補](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6681764711)に譲り、第二の業務Scenario正本を作らない。

### 7.1 値の4つの用途を保存方式に直結させない

| 値の用途 | 言えること | 自動的に言えないこと | 既存Owner |
| --- | --- | --- | --- |
| 計算／Preview | 特定Source・Rule/Operator版・入力集合・時点で計算した値 | 保存済み、提示済み、承認済み、全明細対象の確定値 | #2622/#2506/#2514 |
| Provider Cache／Materialization | あるProvider・Snapshot・再計算契約に従い保持された値 | Ownerが承認した業務事実、永続化が正しい、最新である | #2622/#2600 |
| 業務承認・確定に引用した値 | Owner受理済みの業務目的に対して、特定対象・版の値を判断根拠に使用した記録（正式Meaning/実Decisionの適用は別） | 全変更で自動的に再承認、DBの特定列へ必ず保存、Host Grantを取得済み | #1125/#267/#1126 |
| 実DB Write/外部Effectの観測 | 当該OperationID・対象・時点・Providerで効果が生じたという限定的な事実 | 見積の承認・手作業完了・Owner業務完了・現在開示可能 | #1129/#1174/#2516 |

この4行は **派生値の新しいenum／4つの保存戦略ではない**。同じ数量×単価の計算結果でも、用途・対象・値の採用時点・Rule版・業務責任が異なれば意味同値ではない。承認された値は、採用された業務Meaningによって参照先Revisionを固定している場合、価格表の改版後も「過去に承認した内容」として履歴に残る。一方、その承認値を確定Requestへ再利用できるかは**変更種類・有効期限・再承認規則**のOwner判断であり、自動推測しない。保存方式（導出、キャッシュ、実列、外部証拠参照）は別の設計選択。

### 7.2 独立Expectedを持つ最小の判別対照（既存#1125/#2500 RT/#2514へ割付）

以下は正式な見積業務Ruleではなく、同形の値でも意味を分けられるかを試す**仮定上の対照**。入力・ExpectedをGenerator/Provider結果から生成しない。

| 対照 | 入力／観測 | 必要なExpected・保留条件 |
| --- | --- | --- |
| Previewのみ | 明細1件：数量2、単価300、表示金額600 | 数学的な純粋演算として600。正式な見積発行・承認・DB保存・Business完了は未確立 |
| 承認後の価格改版 | r1で600を判断材料として承認、価格表変更後に再計算すると700 | 過去の承認対象r1と現在の再計算結果を分離。確定時に600/700/再承認必須のいずれかはOwner未決として停止・質問候補 |
| 値は同一だが根拠変更 | 価格/数量/Ruleまたは対象がA→B→A、表示金額は再び600 | 同じ数値／Digestを、依存未変更・旧受理自動復活・現在有効な確定資格としない。変更窓と用途相対の再資格化 |
| 見積明細の部分取得 | 先頭ページだけで金額600、未取得明細あり | その行の局所Previewは可能でも全明細見積額・不存在の証明はunverified。隠れた行やRLS内容を露出しない |
| 取消後 | 元確定Commit/Receiptと、後続のCancel Request/Receiptが共に存在 | 現在の派生状態「未確定」や金額0から過去の承認／Commit履歴を消去しない。返金・通知・手作業補償は別の必要Effect |
| 別受注で同じ値 | O1とO2のPreview金額が両方600 | O1の承認／Handoff実施証拠をO2へ流用しない。Business Definition一致と個別Instance完了は別 |

### 7.3 手順と意思決定を接続する停止条件

- **業務間DFD**はデータ授受と起動契機の記述。分岐・例外・人の確認は**各業務内部Procedure**へ置き、Workflowの実行順序／Host Grantへ自動変換しない（#1125）。Dataflowの描画配置は実行・意味の正本にしない。
- **Question/Requirement Probe**は「承認r1後に価格が変更された場合どう扱うか」を既存Reviewへ提示する用途で十分。Source/Rule変更で判断への影響を検出しても、過去の回答を自動撤回・新Meaningを自動採用せず、履歴を保持し再質問の濫発を防ぐ（#2619/#267）。
- **実行接続**は Function Data Context → typed Intent → 実Request/Target/version/current Grant → Receipt → 因果的に有効な独立Read の既存境界（#2586/#2598/#1174）。Preview成功、Decision記録、DB CAS成功、Manual Handoff完了を相互に代用しない。
- **参照実装の現況**：main統合 [#2926](https://github.com/hat47x/tei-platform/pull/2926) は L・EmptyPolicy・P の局所評価、main統合 [#2671](https://github.com/hat47x/tei-platform/pull/2671) は U/K/V/O/B/D の証拠区分。いずれも「承認値の事業上の確定規則」や実Providerが完全なL/Coverage/空証明を供給した実証ではない。#2619暫定PGチェーンにも productGate=UNQUALIFIED の診断が残る。

**作業順**：①Ownerに「見積の正式発行・承認対象Revision・再見積条件・確定時の採用値・取消の効力」を問う未決項目の棚卸し（判断代行なし）→②既存#1125／#2500 RT／#2514の独立Expectedへ上の対照を割付け、重複Fixtureを作らない→③#2586の実Read入口で必要な入力閉包を確認→④#2598/#1174の同一対象Request・実Readへ接続→⑤#267のMeaning/Consumer正式判断。純粋PreviewだけのConsumerには書込Grant・永続Receiptを要求しない。

**証拠限界**：本節は設計文書のみ。新たなSource/Grammar/Schema/Meaning Engine/Host/DB/GUI/Workflow/テストを追加しない。本対照のGo/PG/E2E、旧X-TEA実動、Ownerの見積・承認意味受理は **NOT RUN／NOT ACCEPTED**。外部原著者の仕様と主張せず、過去ユーザー発言・旧Chat解釈・TEI提案の区分は §1 を適用する。

<!-- tei-triad-teimd-source-subject-bridge-20261106 -->
## 8. 11/6横断追補：*.tei.md統一時も三視点のSubject/Meaning/実行資格を保存する（方針はOwner決定、実移行未実施）

**方針変更の取り込み**：同日後続のAuthoring議論で TEI向けSourceの標準を `*.tei.md` とし、従来 `*.tei` の新規作成を原則停止・段階廃止する方向が明示された。旧「.mdは草稿、.teiが最終意味」という二層ファイル構成を**必要条件としない**。**移行工程と独立Expectedの唯一の計画正本**は [Draft #3030のM0〜M6/TM-01〜14](https://github.com/hat47x/tei-platform/pull/3030) とし、[Draft #2667のTMD-01〜12](https://github.com/hat47x/tei-platform/pull/2667) は横断履歴と対応表に限定する（#3030 PR本文の旧逆向き記述は[後続の訂正段落](https://github.com/hat47x/tei-platform/pull/3030)で明示修正済み。歴史的記述との混同を避ける）。Authoringの意図・原文保持は#2621、Language Parserは#988、実Loader消費は#2823、形式変換は#1920のOwnerが持つ。本書ではData/Function/Business・Requirement Probe・Derived Valueに対する**影響だけ**を保持する。

| 三視点／用途 | `*.tei.md` で保持するもの | 自動昇格してはいけないもの |
| --- | --- | --- |
| **Business** | 原文/見出し/表/手作業/業務間データ授受、内部Procedure分岐、質問と出所span、選択済み意味Role | 文面の曖昧な「高額」→自動閾値、業務間DFD Edge→Workflow実行、質問回答→正式Meaning/Host Grant |
| **Data** | 業務Identity/Relation/時間意味、Instance例、複合Key、NULL・未取得・非開示・0件、実Data Contextの参照 | Markdown表の行順→DB PK、例示行→現在実DBの全件/完全集合、同じ数値→同じ確定業務事実 |
| **Function** | Preview演算とContext Binding、Expectedの独立出典、typed Intentの元となる定義と履歴 | 自由な式/コードフェンス→自動実行、Roleタグ→Plugin実装選択/Authority、Preview600→保存/承認/確定 |
| **Authoring/Build/Consumer** | Raw Draft byte、Markup/Code fence、Source locator/Revision、Required/Parsed/Actually Consumed、Consumerごとの実読取 | ファイル名一致→Parser適合、validate ok→必須Source読込、A→B→A→旧Build/Meaning/Review資格復活 |

**保存先一つ、表示は複数**：同じ `order.tei.md` に説明・Data/Function/Business定義・表形式Expectedを近接させられるとしても、複数の受理済み意味正本を追加しない。どのspan/RoleがAcceptedかと、どの段落がRaw Draftかは用途相対。Graph/表/Studio/SHOはSourceに対するProjectionであり、逆写像が曖昧ならPreview/Reviewに停止する。無効な途中原稿はRaw Draftとして保存・再開できるが、実Runtime入力へ自動合格させない。

**例と停止線**：
- `semantic/sales/order.tei` をそのまま `order.tei.md` にRenameしても、現mainのSemantic Loaderは `strings.HasSuffix(name, ".tei")` なので実読込できない。この事実を移行後の「Sourceを全部検査した」と言い換えない。Plugin/Bundle/Applicationの固定名等も別Consumerで調べる（#2621/#2823）。
- 文書に「50万円以上は上長承認」と書いてあっても、税/通貨/金額の採用時点/上長の所属/例外を勝手に埋めない。#1440の必要な質問を持ち、#1125のBusiness内部Procedure候補を部分受理できても、正式出荷・Host Grantは別資格。
- `*.tei.md` の同一表で数量2・単価300=600を計算し、価格改版後700となった場合、§7のPreview/承認Revision/Provider Cache/実Writeの区分を保持。文書の`status: approved`やOKF `verified`等のメタ情報だけで、受注O1のOwner業務承認や実DB確定を証明しない。
- 移行期間に旧`.tei`と新`.tei.md`が同じ意味IDを持つときは二重定義・Parser差/Source出典を表示し、一方を無音優先しない。ソース移動と意味同値を同一としない一方、パス移動だけで全意味変更とも断定しない。

**受入順の既存Owner対応**：#2621=Authoringの意図・原文保持／#2823=Discoveryと実Loader消費／#988/#2620=Language Role・Parse境界／#1440=文章→意味候補／#1125=DFD・Procedure／#1126=Data Identity／#2506/#2622=実Read／#2500 RT=編集逆写像／#267=Owner受理／#2514=独立検証基準／#1174=Host効果。既存#2586のread-only一往復、#1119 Gateを増設しない。

**証拠・決定範囲**：Ownerの`*.tei.md`方向性と、現main Loaderに残る`.tei`条件は別事実。本節は横断設計への取り込みのみ。既存Workspaceファイル一括改名、Parser移植、OKF規約のTEIへの全面強制、Go/DB/GUI実行、正式Owner意味受理は**NOT RUN**。過去の外部設計者の原著仕様と帰属させない。

<!-- tei-triad-claim-relative-promotion-20261106 -->
## 9. 11/6横断追補：混在するMarkdownの一部をBusiness Meaningへ昇格する際の「主張相対」の閉包（設計・NOT RUN）

**追加の理由と非重複**：[#3030移行計画§9](https://github.com/hat47x/tei-platform/blob/docs/tei-md-unified-source-migration-20261106/plan/design/TEI_MD_UNIFIED_AUTHORING_MIGRATION_2026-11.md) は、同一Markdown内の受理済み`total`／未受理`discount`・部分Parse・Claim単位の鮮度を既に扱う。またM0〜M6とTM-01〜14の移行工程/独立Expectedを所有する。[#2667横断移行メモ§6](https://github.com/hat47x/tei-platform/blob/docs/everything-data-real-evidence-cut-20261103/plan/design/TEI_MARKDOWN_SOURCE_MIGRATION_2026-11.md) は歴史・対応表に限定し、TMDを第二受入正本にしない。[#3031暫定ガード](https://github.com/hat47x/tei-platform/pull/3031) の限定Go/CIは未BindingのTEI Markdown Sourceの黙殺を拒否しただけで、新Parser接続ではない。形式移行と一般的な部分Meaning受理はこれらへ譲る。[#1440 POC-4/5](https://github.com/hat47x/tei-platform/issues/1440) はAs-Is/To-Be/仮説と業務有効期間を区別済み、[#1125の受注・承認縦切り](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6681840353) はBusiness完了の個別対象・承認Revisionを区別済み。本節の固有差分は、**1つのTEI Markdown文書中に混在する複数の主張に、同一の「文書が読めた／文書が承認された」資格を一括付与しない**こと。

**本節のBusiness固有の評価単位**：一つのAuthoringファイル内で **As-Is／To-Be／Preview／個別O1** の各Claimについて、(i) Ownerの規範的Meaningとして受理する必要があるか、(ii) どのビジネス時点・Subject・金額/承認Revisionを固定すべきか、(iii) 実Function/Providerの現在観測・Grantを必要とするかを別々に決める。**文書全体でRole/ParserがPASSしたか**はこの表の代用にならない。独立Expectedは本節X1〜X6を既存#1440/#1125/#2514のCaseへ対応付ける比較列にとどめ、新番号の主受入票を増やさない。

### 9.1 一つの原文にある四種類の主張を別々に扱う

例として `sales/order_policy.tei.md` に次の断片が並ぶケースを想定する（**自作の検証入力候補であり、受理済み業務規則ではない**）。

1. 「現状は担当者が電話で確認している」：As-Isの業務説明。手作業Procedureの候補であり、自動Workflow Actionではない。
2. 「今後は50万円以上の注文を部門長が承認する」：To-Beの規範的候補。対象金額の税・通貨・改訂時点、承認する組織Role、適用開始時点、差戻し条件が未決なら、依存する業務Actionだけを保留する。
3. 「数量2×単価300なら600」：単位・数値型・丸めRule・入力が確定した範囲で純粋PreviewのExpectedになり得る。ただし見積発行、特定受注の承認、Provider実Write、Business完了のExpectedにはならない。
4. 「承認済み受注O1を確定する」：Workflowとみなす前に、DefinitionでなくInstance O1、承認対象Revision、有効な業務Meaning、Function Contextの全必要対象、実Target/Grant/Effectを別に要する。

同じファイル内で(1)の説明を保持でき、(3)の純粋計算が局所的に検証できても、(2)の未決条件や(4)の実行不可が解消されたことにはならない。逆に未解決の(2)のためにすべてのRaw Draftの編集・保存を禁止しない。

### 9.2 処理入口と主張の資格を二軸で評価する

| 利用目的 | Source側で必要な根拠 | 成立しない推論 |
| --- | --- | --- |
| Raw Draft Save / Restart | 原byte・版・保存先・Write競合／権限 | Lexer不成立なので原稿を破棄する、保存できたから意味受理 |
| 選択範囲のRead/Preview | 実選択Frontend/Parserと対象Role/span、部分構造の妥当性、必要なInput/Expected | 同一ファイルの全段落が解析・意味採用された |
| Meaning Promote / Procedure Decision | 候補が依拠するSource部分集合、外部参照・Role/Operator版、As-Is/To-Be・適用時間/Subject、独立Expected、Ownerの明示受理 | タグ/ファイル名/構文PASS、旧回答、表示金額から自動採用 |
| Build / 実Action / Consumer | 対象Claimに必要な実消費Source閉包、最新Meaning/Rule、実Context・Host Grant/Provider・Receipt・独立Read/開示資格 | ManagedFiles列挙、validate ok、局所Frontend成功、旧Artifactから全資格成立 |

`*.tei.md` を実Workspaceに置いた際、現mainではNative Loaderが未読込となる。#3031 Draftは `TEI_MD_PARSER_UNBOUND` で**黙った成功を止める**が、これはMarkdown本文のParser/Partial Acceptanceではない。現形式未対応のProduct Validationと、Raw Draftを保存・再開できるべきAuthoring要件は**別入口・別結果**として検証する。新Reader完成までガードを無効化して無音スキップへ戻さない。

### 9.3 最小の独立反証（各Ownerの既存Scenarioに付け足す比較列、Fixture正本は増やさない）

| 比較条件 | 独立Expectedと停止条件 |
| --- | --- |
| X1: 文書にAs-IsとTo-Beが併記 | 現行運用の記述は現行Runtime Ruleではない。To-Be候補は採否とbusiness effective intervalを別に持ち、曖昧な範囲だけ保留 |
| X2: 50万円閾値の質問に回答があるが、税込/税抜と適用開始日が未決 | 他の独立Stepを記述/審査できても、当該承認Branchの正式Meaningと実行可否は未確立。動的な注文金額を意味質問で埋めない |
| X3: Preview=600と、価格改版後700、承認した過去のr1 | 数値の再計算・承認したRevision・実受注の現在値を別列とし、旧Business Approvalの自動再有効化／取消をしない（§7再利用） |
| X4: 最初のSnippetだけTEIとして構文解析でき、別段落は未完 | Raw Draftは保持。明示された対象spanと依存閉包が実際に確定しない限り、文書全体のMeaning Promoteを主張しない。構文境界が曖昧なら局所解析も保留 |
| X5: .tei.md存在・ManagedFiles列挙・Loader未消費 | 本番相当validateは合格扱いしない。#3031の一時的拒否（Draft）と#2935のreadByLoader（Draft）を実資格と別記 |
| X6: 旧.teiの受注定義と新.tei.mdのProcedure候補が同じ意味IDを引用 | Source Location・Authoritative use-site・受理Rev・Semantic SubjectとInstanceを別に再照合。重複正本・旧Receipt/旧Reviewの無条件継承を拒否 |

**次の最小実証順**：#3031で「未接続の失敗を隠さない」を維持→#3030/#988/#2621が明示Parserとbyte spanの閉包を限定PoC→#1440が一段落の現状/将来/仮説と回答履歴を候補化→#1125/#267が影響するBusiness Claimだけを受理→#2586/#2514が実Readerと独立Expectedを照合→#2598/#1174が実書込Target/Grant/Receiptを別検収。#1119の既存Gateと、#2620 Role・#2600 Build再資格化のOwnerを置換しない。

**状態と限界**：#3031の固定HEAD Go/CI成功は暫定ガードのみの局所証拠で、Markdown Parse/Role Lowering/Business Accepted Meaningの成功ではない。上記X1〜X6の接続Go/CLI/DB/GUI/SHO/実業務試験、Owner正式Meaning Acceptanceは**NOT RUN／NOT ACCEPTED**。本節はdocs-only、既存規約の新enum・第二Registry/Parser/Acceptance/Host/Workflow Engine追加は求めない。

<!-- tei-triad-completion-effect-nontransitivity-20261107 -->
## 10. 11/7横断追補：業務完了の未検証とCommand作用の不明を混同しない（実装済みの差分を既存受入へ接続）

**今回の追加判断**：[#1125 CDE-01〜04](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6687996326)が指摘した`観測0件≠業務対象Lが空`と、[#1174 F-Pre/Post/NoReport/SameKind](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6690399598)が指摘した`Commandの失敗報告≠Provider効果なし`を、**同一受注Actionの二つの独立した判定**として接続する。既存CompletionState／Effect状態とScenarioを再利用し、新しい総合成功State/Proof Engineを追加しない。

### 10.1 先行実装と証拠の上限（mainとDraftを明示）

- [main統合 #3051](https://github.com/hat47x/tei-platform/pull/3051)（merge `b30ce3708a8385dcf219aed9f53e594bc0e95d12`）の現 `internal/semantic/completion.go`（取得Blob `e42610e9159e02ba3b5c7b84c46742a846331e64`）は、`EmptyPolicyDeny && Targets=[]` でも `CoverageComplete` が無ければ `CompletionUnverified` を返す。既知の述語違反があればpartialでも `CompletionFails` は維持。固定HEADの[Repository Health #42298116044](https://github.com/hat47x/tei-platform/actions/runs/42298116044)はcompleted/success。これは**Callerが申告するCoverageの出典／Tenant／時点／Scopeを実検証した証拠ではない**。
- [main統合 #3042](https://github.com/hat47x/tei-platform/pull/3042)（[Health #42251024310](https://github.com/hat47x/tei-platform/actions/runs/42251024310) completed/success）は、`Authorizer == nil` と宣言された期間の実状態不足を**実行前に拒否**し、**Executor実行後**の宣言外効果／Outcome／Applied不一致を `EffectUnknown` とLifecycle `unknown` へ記録。同owner/generationの盲再送を拒否する。これは実Providerで作用0やrollback成功を証明するものではない。参照用`ReferenceAuthorizer`は実主体・実Grantの検証を代替しない。
- [main統合 #3047](https://github.com/hat47x/tei-platform/pull/3047)の `GrantAuthorizer` は宣言と発行済み許可の包含を比較する限定Reference。許可の発行/保存、CLI・Studioの実Command経路、実Providerの最終対象検証に結合済みとは扱わない。
- [Draft #3030](https://github.com/hat47x/tei-platform/pull/3030) は`*.tei.md`のM0〜M6/TM-01〜14という移行主担当のまま。旧`*.tei`に対するCDE/Command Go成功を、新形式の実Source/Readerが消費した成功へ転用しない。

### 10.2 同じ受注O1のClaimにおける「停止理由」の直交対照

下表のビジネス規則・Source/Row/Grantは**合成例の検査用仮定**であり、Ownerが正式に受理した受注規則ではない。Condition判定・Dispatch・Effect・Owner完了を同じ一つの合否に縮約しない。

| 既存検収Caseに対応させる状況 | 業務条件の判定 | 実行・実効果の別判定 | 誤った昇格 |
| --- | --- | --- | --- |
| **CDE-02**：Lを0件観測、取得完全性未確立、空集合deny | `unverified`（現main修正済み） | 未取得なのでBusiness上の成立・不成立は言えない。**未Dispatch**ならその事実を別記 | `0 observed→fails`、または空集合allowへ勝手に変更 |
| **CDE-01**：同一Scope/RevisionでLが真に空、deny | `fails` | Condition不成立から過去のEffectの不存在は導けない | `fails→DB rollback済み` |
| **CDE-04**：対象内の違反L1を一件観測、他はpartial | `fails`（反例に必要な対象束縛は独立検証） | 他明細／他Tenantの不存在は証明していない | Partialだから常に`unverified`にする過剰防御 |
| **F-Pre**：条件が`holds`でもAuthorizer/period/required guard不足 | 業務条件は局所成立し得る | **実行前拒否**。実Executor非到達、変更なしの対象を必要に応じ独立Readで確認 | `Business holds→Authorized/Committed` |
| **F-Post**：実Executorが返したApplied/Outcomeが宣言と不一致 | 完了条件は別評価 | `EffectUnknown`、Lifecycle `unknown`。同OperationIDでReceipt照会→同Target別経路Readが先 | 上位Error/unknownを`Effect=0`として別ID再送 |
| **F-NoReport/F-SameKind**：Applied空／同Kind異Dataの追加更新 | 観測済み条件を超えるBusiness完了は未確定 | `Applied=[]`は実作用ゼロの証明ではない。追加更新は関係先の正当なcascadeか対象外作用かを別判定 | Result成功や報告件数だけでProvider効果と許可範囲が完全と扱う |
| **ORG-TB**：同じCase IDを別Tenantが持つ／承認Resourceと実更新Rowが異なる | 正しいSubject・承認Revisionに限定して評価 | Grantの論理名、実Resource、Provider更新Targetの同一性を別照合。Case ID文字列一致は不十分 | 他Tenant Caseへの書込み・承認実施を同一と扱う |

**互いに代用しない証拠**：Businessでの`CompletionFails`／`CompletionUnverified`、Commandの事前拒否／事後`EffectUnknown`、ProviderのCommit/Receipt・独立Postimage、現在Grant、手作業Handoffは**同一Attemptで並立し得る**。例えば違反明細が実在して業務条件が`fails`でも、別の既往CommandのEffectが`unknown`ならその効果を消したと解釈してはならない。逆にCompleteな空の不成立と、0件しか観測できなかった不明状態は、その後のデータ取得・是正操作の判断が違う。

### 10.3 次の一回の実受入へ渡す最小単位（新Fixture・Gateなし）

1. **#3030/#2823/#988**：対象に実消費されたSource/Role/Operator/Revisionを固定。新`*.tei.md`未Bindingなら実業務の検証を開始せずRaw DraftとProduct Validateの状態を分離。
2. **#2506/#2586/#2514/#1125**：受注O1の業務上必要な集合L、実ReadのScope/Tenant/Field/Row/Cohort、独立に根拠づけられた`CoverageComplete`、Rule版とExpectedを同じSourceに結ぶ。新修正#3051のCDE-01〜04を含む現mainのGo結果は**局所Evaluator合格**であり、実Readerの完全性は未資格。
3. **#2598/#1174/#1163**：同一O1のAuthorizer/Period/Current Grant/Resource/Payload/ExpectedVersion、Executor呼出有無、適用操作報告、Commit/unknown/Receipt/別接続Readを一Attemptで実測。#3042で事後unknownは既に表現されるが、`Applied=nil`や同Kind追加Scopeを実際に観測したTargetとして見なさない。
4. **#2389 §82/82.5／#1125/#267**：承認したBusiness Actor/職位/委任の有効時点、Hostで許可されたResource、実ProviderのRowの同一性と、手作業/外部Handoff/Ownerによる正式意味受理を最後まで独立列に保持。既存[ORG-TB-01〜10](https://github.com/hat47x/tei-platform/pull/2389)と[#1174 Fケース](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6690399598)へ割り当て、新しい組織・Command検証正本を作らない。

**結論と未実証範囲**：局所`Completion`修正と局所`EffectUnknown`対応はmainへ統合済み。一方、両者を**同一TEI Source・同一受注O1・実Providerの同一Attempt・Owner受理**で通した製品検収は未実施。PR/CIごとの限定成功を寄せ集めてG1/G2合格としない。実PGでのCommit・別Connection読取、GrantとProvider実更新Rowの比較、業務受理、GUI/Consumer現在性は**NOT RUN／NOT ACCEPTED**。追加の新Status enum、Universal Authorization/Workflow/Effect Engine、独立検証基準の第二正本は作らない。

<!-- tei-triad-business-projection-claim-evidence-20261107 -->
## 11. 11/7横断追補：「読んだ・計算した・実行した・保存した・業務上成立した」を同じConsumer上で区別する（既存契約の接合のみ）

### 11.1 追加するのは新たな状態機械ではなく、一つの業務結果の説明責務

§7〜10では受注のPreview、Owner受理済みMeaning、Completion、Commandの認可、Provider Effect/Receiptを別々に整理した。その後のmain実装には、**外部Client/Studio/AI Harnessが受け取る結果そのものが、旧い慣習的な「成功」の意味を保証しない**という新しい具体例が増えた。基盤が報告している各事実をBusiness Procedureの状態やUI表示へ翻訳するとき、**入力元の実消費と結果の証拠上限を失わないこと**を本計画固有の接続点とする。CLI JSONの形式/互換の正本は[横断#2667 §59](https://github.com/hat47x/tei-platform/pull/2667)／[#2824](https://github.com/hat47x/tei-platform/issues/2824)へ、Commandの正式結果契約は[#1174](https://github.com/hat47x/tei-platform/issues/1174)へ委譲する。

| 受注O1の利用者に説明する区分 | 現行Source/CLIで実際に観測したい値 | 当該値だけでは言えないこと |
| --- | --- | --- |
| **業務の定義が存在する** | Requested Source/Root、World内のSubject、Source identity、Parser/Role/Meaning Decision | `*.tei.md`の存在・管理列挙・`validate ok`・空Model Catalogで正式定義がある／ない |
| **定義を読んで検査した** | `readByLoader`、実Parsed Occurrence/依存/対象数、必要Sourceの実消費、独立期待値の適用 | `readByLoader=true`だけで当該Function、Rule、Business Meaningを検査・実Builderが消費した |
| **計算・業務条件を評価した** | 数量/単価/税等の入力版、Function Context/TargetSet/Coverage、`EvaluateCompletion`の理由 | Preview600や`CompletionHolds`だけで見積を発行・承認・保存・業務完了 |
| **実行を許し、呼び出した** | 現在の主体/Resource/Grant、Required/Supplied/Enforced Guard、Executor到達、Attempt/Operation ID | Pattern/メニュー宣言、古いGrant、Command応答成功から実作用が存在した |
| **作業単位をCommitした** | `committed`・実際のUnitOfWork種類 | `committed=true`から永続保存済み・DB Tx確定 |
| **作用と保存を確認した** | `persisted`/`durability`、`Applied`/`differences`/`EffectUnknown`、実Provider Tx/Receipt/別Reader Postimage | `persisted=false`をDB保存と扱う、差分なし/報告空を実作用0・対象完全集合とみなす |
| **業務上成立し公開できる** | 引継ぎ・手作業の確認、Owner受理した業務Meaning/期間・対象、Consumer現在権限・開示資格 | Resultの`ok`・Commit/ReceiptやDB Postimageだけで業務承認/完了/公開可 |

### 11.2 最新mainの限定成果を同じ業務Scenarioに束縛する

- [merged #3052](https://github.com/hat47x/tei-platform/pull/3052) は`model catalog -namespace`の未知/非namespace IDを`NAMESPACE_NOT_FOUND`（終了1）として分ける。存在するが構成要素が0件のnamespaceは正常空に留まる。したがってAI/Harnessによる文字列指定間違いをBusiness定義「存在しない」の証明へ転用しない。応答互換は#2824の別判定。
- [merged #3078](https://github.com/hat47x/tei-platform/pull/3078) は`workspace-files`の管理対象に`readByLoader`、Root診断を追加し、旧`result`配列から新Objectへ変更した。これは**Byte読取りまで**の証拠で、Role/Meaning/Build消費とは違う。[merged #3092](https://github.com/hat47x/tei-platform/pull/3092)／[merged #3095](https://github.com/hat47x/tei-platform/pull/3095) のschemaVersion広告対実応答の検査がPASSしても、`v0`のまま形を変えた旧Consumerとの構造互換は自動成立しない。旧新Clientの4方向比較は#2667 §59／#2824で所有。
- [merged #3054](https://github.com/hat47x/tei-platform/pull/3054) により参照Commandの`committed=true`は`MemoryUnitOfWork.Commit`呼出し成功を指す一方、`persisted=false`／`durability=none`である。CLI/Studioの結果を「受注を保存しました」と翻訳しない。実Provider時の`persisted`資格や本番のTransaction→独立Read対応は未採用。
- [merged #3055](https://github.com/hat47x/tei-platform/pull/3055)／[merged #3057](https://github.com/hat47x/tei-platform/pull/3057) は、同Kind別対象への適用を`Difference`としてOperation記録・CLI/Studio結果に表示する。差分を記録しながらCommandが成功し得ることは**業務上の追加対象が正式に許可・説明されたという意味ではない**。合法なCascadeと無関係C9への更新を同じKindで自動同等に扱わない。報告`Applied=[]`の完全性とProviderの実際の作用範囲は[#1174](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6690399598)へ。
- [merged #3042](https://github.com/hat47x/tei-platform/pull/3042)／[merged #3053](https://github.com/hat47x/tei-platform/pull/3053) は実行後の不一致や効果不明を`EffectUnknown`／`COMMAND_EFFECT_UNKNOWN`で区別する。**不明は永続化済みでも未作用でもない**。Operation IDと実Provider Receipt/再読取による回復判断を必要とし、別keyでの盲再送を許さない。

### 11.3 受注O1／無関係対象C9の最小の正負対照（追加Fixture IDなし）

**前提**：次は公開合成Scenarioの独立Expected候補であり、受注承認・Cascadeの業務規則として正式にOwnerが受理したものではない。旧`*.tei`の同一Revision／同一Subject／同一独立期待値を基準に、将来新`*.tei.md`を実Readerへ移した後は形式ごとに実消費を証明して比較する。

| 対照 | 期待される停止・表示／許される結論 |
| --- | --- |
| 正しい`sales`を選択、Subject/Rule/必要入力が実消費 | 文書・Reader・Meaningの各資格を示して局所Previewを許す。ただし承認・実Effectはまだ不明 |
| `salex`と誤記、結果0件／Unknown Root／Managed未読 | 未知指定・未消費・既知の真の空を区別し、AIが`0=契約上対象なし`を勝手に選ばない |
| `committed=true,persisted=false,durability=none` | 「参照用作業単位のCommit成功、永続保存は未確認ではなく**この参照実行は保存していない**」と表示。業務確定済みとは表示しない |
| 宣言したO1の更新に付随し、承認済みRelation/Ruleに従う明細のみ追加更新 | 差分の存在/対象を表示。許容範囲と実Providerの更新対象が一致すると確認できるまでは業務的な追加作用の受理を保留 |
| O1の更新と無関係C9への同Kind更新、または`Applied`空 | `differences`／未観測として表示し、単純成功に畳まない。C9を作用させたなら実Receipt/再読取を別に記録。既存の事後報告だけでCommit取消しを推論しない |
| Responseが`COMMAND_EFFECT_UNKNOWN`／再送拒否 | 旧Operation/Attemptを固定したReceipt Lookup・独立Postimageと必要な権限を確認。別key再送・自動承認・自動Rollbackをしない |
| 同じ`tei.cli.workspace-files/v0`で旧配列Readerが新Objectを受信 | 当該Consumerは構造不適合として明示診断。schemaVersion文字列の一致や正常終了を業務証拠の取り込み成功にしない（実互換テストは#2824） |

**業務利用者に見せる最小区分**：説明上は少なくとも「定義/入力を確認した」「試算・条件判定」「実行済みかは未確認」「保存先への作用確認」「業務承認・完了」の区別を保つ。これらは上表の**表示・検証の観点**であって新しい公開Status enumや一つの総合ゲートではない。Studio、CLI、Harness、SHOの出力形が違っても、同じ根拠を各Consumerの現在資格に基づいて説明する。業務意味・実行主体・実効果・開示権限が欠ける場合に、もっともらしい肯定文へ変換しない。

### 11.4 既存Ownerへ割り付ける次の実証

1. **#3030/#988/#2823/#2586**：同一Sourceに対するRequested/Managed/`readByLoader`/Parsed/Build/Function実消費の差を、旧Sourceの実Go Readから採る。新`*.tei.md`はReader未対応の間は別途fail-closedのまま。形式切替と業務Readを同一の「成功」にしない。
2. **#2824/#1163**：受注O1の実Consumer側（CLI/Studio/AI Harness）で`schemaVersion`と実JSON形/必須Fieldを読み分け、旧新Readerの構造互換を独立Expectedで確認。Producerが`ok:true`を出しただけのProvider/evidence Claimを禁止。
3. **#2500/#2598/#1174/#2514**：O1更新・関係明細・無関係C9を固定し、Intentに対するApplied差分のSource/Rule/Target資格と、`committed`/`persisted`/実Receipt/独立Readを同じAttemptで対照する。事前拒否と事後Unknownを分け、真のProvider Row/Operationの同一性を確認。
4. **#1125/#267／#2389**：Business Procedureの手作業・承認対象Revision、組織上のActor/委任と現Grant、業務的なCascade許可Scopeを判断する。仮規則で合成PASSしても、人間の正式な意味受理・本番Grant/Business完了の資格へ昇格させない。

**証拠上限**：各main PRの限定Go/CLI試験は既存の結果であり、本節の同一O1/C9／同一Revision/Provider/Consumerでの統合試験ではない。新Reader・旧新Clientの実互換・DB実作用と別Connection Read・本番Grant・Owner正式Business受理・新しいStudio/SHO/Harness E2Eは**NOT RUN／NOT ACCEPTED**。新たなSource/Oracle/RunPlan Registry、第二JSON Parser、万能Audit Engine、Host、業務状態表を作らない。

<!-- tei-triad-receipt-postimage-separation-20261107 -->
## 12. 11/7横断追補：Receipt再照会を「宣言したData Effectの保存後状態確認」に昇格させない（#3127当時の履歴と#3155以降の実装再資格化）

**現行版の読み方**：§12.1〜12.4は#3127当時の単一`observed`問題を分析した履歴であり、現在mainの未修正欠陥を主張するものではない。最新の修正状況・残余リスクは§12.5を優先する。

### 12.1 main統合された3段階と、最新実コードで残る意味のずれ

[#3054](https://github.com/hat47x/tei-platform/pull/3054)で参照実行の`committed=true`と`persisted=false`を分離した後、[main統合 #3124](https://github.com/hat47x/tei-platform/pull/3124)で`command.Result.Persistence`に`not-applicable / not-observed / observed`を追加した。[main統合 #3127](https://github.com/hat47x/tei-platform/pull/3127) は`commandadapter.Executor.Lookup`から`effectcommit.Runner.Lookup`を呼び、`StatusReplayed`なら`PersistenceObserver.Observe`を成功終了させる。

**Source-staticの具体的な非推論**：
- 現main `internal/command/persistence.go`（確認Blob `be114c633dd7b41815ab713c18a33e0efd878da0`）では、Observerがエラーを返さなければ`PersistenceObserved`となる。コメント上の意味は「宣言したEffectが保存先から独立読取で確認された」であり、CLI/Studioの`persisted`はこの状態から真になり得る。
- 現main `internal/security/effectcommit/commandadapter/executor.go`（確認Blob `6b854d7240a4401593c8b693bdea657aee8b5ff2`）の`Observe`は**同じOperationID/Request/payload/GrantのReceiptの再照会**だけを行う。対象となる業務行の保存後値、Tenant付きRow Identity、適用されたField/Relation/Versionを再読取しない。引数`declared semantic.DataEffect`はReceipt検査対象の粒度を実質的に拡張しない。同ファイルのコメント自体も「receipt observation, not record observation」と明記する。
- [main統合 #3118](https://github.com/hat47x/tei-platform/pull/3118)では、Executorが返す`Result.Effect`を**呼出し側が与えた宣言Effectの再報告**で補える。報告欠落`ReportedEffectMissing=false`であっても、実Providerがその対象Effectを作用させたことは証明できない。類似して`Applied`空や`Difference`無しは実更新の全件証明ではない（§10〜11、#1174）。
- よって、`Lookup=true`、`DurableStore=true`、該当Observerを実配線した構成では、**Receiptしか確認していなくても汎用`PersistenceObserved`へ達し得る**。これをStudio/CLI/Harness側で「受注O1の保存を独立確認した」と受け取る場合に、契約上の過大主張が成立する。**当該構成が本番業務で使われ、誤った保存確認が実際に発生したとは確認していない**。

### 12.2 一つの受注O1で独立させる証拠：履歴Receiptと保存後の現在値

| 観測 | 独立に言える範囲 | その観測だけでは言えない範囲 |
| --- | --- | --- |
| `UnitOfWork.Commit`成功 | 当該構成のCommit呼出しが成功した | 独立な対象確認、業務上の受注確定 |
| `Runner.Lookup=StatusReplayed` | 指定OperationID＋RequestHash等に対応するReceiptが記録され、照会できた（当該Lookup契約の範囲） | O1の現在値/明細全件・許可外C9の不在・今の業務完了 |
| Caller由来の`Result.Effect` | 呼出し側が宣言Effectを特定していた | 実Data Effectがそのとおり作用した |
| `Applied`と`Differences` | Executorが報告した操作と宣言との差 | 報告漏れ/同Kind別対象の未検証・実Provider行の完全性 |
| 別接続・適切なSnapshotでのO1 PostimageとC9不変 | 読み取った時点/ScopeでのO1/C9の実状態 | 過去Operationの因果的寄与（ReceiptやVersionの束縛なし）・将来の不変性 |
| Receipt＋Postimage＋Operation/Target/Version/Grant/Meaningの同一試行照合 | **証拠が揃った範囲**で「指定した受注Operationの効果を確認」する候補 | 手作業引継ぎ、業務承認、OwnerのMeaning受理、Consumer開示権限 |

`persistence=not-observed`も「DBに保存されなかった」ではなく、**現在のObserverに必要な観測が無い／失敗した**という意味を守る。Receiptのみ確認できてもBusiness完了・確定Postimageは保留する。また`StatusReplayed`は既往のOperationを指すので、現在のGrantや再承認・更新後の版が有効なままとは推測しない。

### 12.3 既存Scenarioへ重ねる最小4対照（新Fixture正本・新状態機械なし）

| 同一O1、OperationIDを固定した負例／対照 | 独立期待値・診断の要求 |
| --- | --- |
| **R+ / O?**：Receipt Lookup成功、O1の実状態は未読・Reader unavailable・Replica遅延 | 「操作記録を確認」とは言えるが**対象状態の確認済みとは言えない**。現在の`persistence=observed`だけから「O1保存済み」の表示・Owner完了Claimを発行しない |
| **R+ / O≠**：Receiptは存在するが、その後O1が更新/取消された、または別Tenantの同表示IDを読んだ | 履歴Receiptは残す。現在O1の値/対象/版の不一致を提示し、過去Effectの取消・無作用や現時点の条件達成を推測しない |
| **R? / O+**：Receipt Lookup不可、O1の状態値は期待と一致 | 一致したPostimageの帰属を当該Operationへ**無条件に確定しない**（別Attempt/同値業務/旧更新の可能性）。Lookup失敗で別ID盲再送しない |
| **R+ / O+、Target mismatch**：Receipt/Readが双方成功、だがGrantはO1・PayloadはO2、あるいは余分にC9更新 | 両方成功でもBusinessの`Target/Relation/Scope`の矛盾が解消していない。必要な現在Grant、実更新行、承認Revisionの同一性を照合し、過剰な成功にしない |

`R`は**Receipt照会**、`O`は**保存後の対象状態の独立Read**。二つは「どちらか一方の成功」ではなく、用途ごとに必要性が異なる**別の検証根拠**。この4対照は既存[#1174 F-Post/F-NoReport](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6690399598)・[#2671 PO-01〜15](https://github.com/hat47x/tei-platform/pull/2671)・[#2514独立検証基準](https://github.com/hat47x/tei-platform/issues/2514)へ判別列として追加する想定で、第二の固定Scenario/Oracle/Runnerを設けない。

### 12.4 既存Ownerへ渡す最小の契約補正と実装順

- **#2998 / #2824（Command結果・Consumer契約）**：`PersistenceObserved`と`persisted=true`が**Receipt観測を示すのか、宣言Effectの保存後Row確認を示すのか**を、現在の公開説明/実行UIで一致させる。後者を意味するならReceipt-only Observerはその資格を満たさない。別途、観測根拠の種類（ReceiptかPostimageか）が利用側へ伝わる最小表現と旧Client互換を判断する。新enumや版上げの採用はOwner判断であり、本書で決めない。暫定表示は「操作記録を確認」と「対象データを確認」を言い分ける。
- **#1174 / #2598 / #1163（Provider・効果・証拠）**：同一OperationID/RequestHash/Scope/Grant/Targetの`Runner.Lookup`が確かめたReceiptと、独立した対象O1 Postimage/C9不変・Snapshot/期間を別Witnessにする。実PGと別Readerによる観測後に初めて効果・永続化のQualificationを見直す。2026-11-07時点で #3127 は**mainへ統合されたが、実PG結合でLookup/Row Postimageを同時に通した試験は未実証**。
- **#1125 / #267 / #2506（Business/Meaning）**：Owner受理済みの受注確定規則と、作業担当者の承認・個別O1の成立判断をこれらの技術的Receipt/Readと区別し、今の`Completion`結果・Scope/TargetSetを再照合する。過去の承認・非公開の資料から新しい承認/開示を自動導かない。
- **#3030 / #988 / #2823（Authoringと実Read）**：旧`*.tei`でのReceipt/Lookup成功と、未実装の新`*.tei.md`定義の実消費を別に扱う。同一Meaning/Source/Revision・Operationでつながっていない別々のCI成功を製品一往復へ合算しない。

**状態**：main #3124/#3127/#3118の静的実Sourceと既存テスト/PR記録、計画への引継ぎに基づく**設計上の矛盾候補と検収の補強**。今回のR/O対照を実Go/PG/Windows/GUIで新たに走らせたものではない。Ownerが公開`persisted`契約の意味を修正採用したわけでも、実Provider・Resource Postimage・正式Business受理が成功したわけでもない。第二Provider/Host/Runtime/Meaning Registry・新業務状態表は作らない。


<!-- tei-triad-receipt-resource-current-qualification-20261107 -->
### 12.5 #3155以降はReceipt・現在値の観測を実装上分離済み、残る因果・同一性を別検収にする

[main統合 #3155](https://github.com/hat47x/tei-platform/pull/3155)では、`PersistenceState`を`not-applicable`／`not-observed`／`receipt-observed`／`resource-observed`に分離し、**`persisted:true`を`resource-observed`に限定**した。これは前掲のReceipt確認をResource確認へ誤昇格させる問題への実装済み対処。[main #3157](https://github.com/hat47x/tei-platform/pull/3157)の成立状況には実PostgreSQLでのReceipt Lookupと資源読取試験の結果が反映されている。[PR #3155の認可・Effect PGワークフロー](https://github.com/hat47x/tei-platform/actions/runs/42328707393)もcompleted/success。**ただし同一Business O1/C9・承認・全Provider経路の製品完了実証ではない**。

main実コードを照合した残余の証拠境界：
- `internal/command/persistence.go`（blob `5e8fcec8eca81ded38c9305adba054f71ce168e9`）は`DetailedPersistenceObserver`の`receipt`/ `resource`を分離する。一方、旧`PersistenceObserver`はエラー無しをresource観測と扱う後方互換経路があるため、**そのObserver実装が本当に独立Readした証拠**は外側で確認する。
- `internal/security/effectcommit/commandadapter/executor.go`（blob `7d1ccedd7dca5f8df8f5c84da0fca722ef8e9198`）の`ResourceRead`は`Found`と**`bytes.Equal(state.Payload,payload)`**を確認する。返却された`ResourceState.Version`は、同コードではExpected/Commit版と照合していない。**同値を別Attemptが書いた場合、現在値一致だけで当該Operationの因果を証明しない**。
- PostgreSQL `ReadResource`（blob `31c0f03a60fbabfec2d0163553bc9f95e54e1b20`）は別poolで`objects`の`version,value`を`application,resource`条件で再読取する。関数のコメントどおり**ここではRead Authorityを再検査しない**。Host側の読取権限と開示、Tenant/業務Keyと物理Resource/RowのBinding、関係する子集合・無関係C9の追加更新は別に検証する。

| 同一O1／Operationの追加対照 | 独立期待値 |
| --- | --- |
| Receipt R+のみ、O1は未読 | `receipt-observed`。資源観測/業務確定へ昇格しない |
| O1保存後値は一致、別Attempt・旧Attemptも同値 | `resource-observed`は**観測時点の局所値一致**。Receipt/Version/RequestHashと対象を再照合し、因果と業務完了を別検証 |
| O1は一致、必須子明細がpartial／C9が余分に更新 | 一資源の観測を全Effect/全件・Scope内作用に拡張しない |
| 観測データが正しくても現在のReader GrantやConsumer開示権限が無い | 実値は非開示。Authorizerが他境界で許可したことをReader権限へ推定しない |
| Receipt保持、取消やABAによって現在値が再一致 | 履歴Receiptは残るが旧Business承認/Host Grant・現在Effect資格は自動復活しない |

**既存Ownerへの依頼**：#1174/#2598/#1163は同じOperationID/Target/Version/GrantでReceipt・独立Postimage・追加作用を同時に確認し、#2998/#2824はCLI/Studio/Harnessの新旧Consumerが`receipt-observed`と`resource-observed`を区別できるか検証する。#2514/#2506/#1125は同じShape/Business対象の部分集合・意味受理と切り分ける。新しい状態機械、Receipt Store、検証基準の第二正本は作らない。**今回の新Go/実PG/本番IdP/Business一体試験はNOT RUN**。

## 13. 11/7横断追補：標準データ構造と標準表現型を独立に定義し、具体的表現はFunction内へ配置する（設計・未受理）

### 13.1 三視点に対する新しい整理と既存Ownerの境界

X-TEAではDBテーブルの1:1・1:NトポロジーをXFxxxの単票/明細機能種別に結び付け、機能の連携で全体の画面を覆う構成が中心だった。TEIでは以下を**別の記述単位**として取り扱う設計案を採る。標準表現型の候補語彙そのものをTEI Coreの固定UI enumに直ちに採用するものではない。

| 記述単位 | 意味・責務 | ここへ置かないもの |
| --- | --- | --- |
| **標準データ構造（Data Shape）** | Scalar/Record/Collection/Relation/Tree/Graph/時系列/空間等の形と、Identity・粒度・多重性・順序・NULL・時間・完全性の制約。DBテーブルから独立 | XF型・UI widget名・画面レイアウト・Host Grant |
| **標準表現型（Representation Kind）** | Panel/Tab/Text Input/単一・複数選択/Toggle/Slider/Table/Tree/File-like/各種Graph等の**再利用可能な表示・入力方式の契約**。必要な値域/形・表示/操作適合・情報損失を記述 | 特定業務のEntity、業務Rule、利用者の現在権限、単独で成立する画面/機能Instance |
| **Function Data Context** | Functionが使うData Shapeの対象・Relation path・Field・Record grain/Result grain・必要集合/取得Scope・選択時点を特定 | Data Modelに関係があるというだけの暗黙取得、Renderer固有の意味 |
| **Function内のRepresentation Instance/Slot** | 選択した表現型をFunctionのData Context/Roleへ**具体的に束縛**して配置。表示/入力/選択/Navigation/任意のAction Offerを合成 | 新たなAccepted Meaning/権限正本、SQLやDB更新を直接確定するWidget |
| **Business Procedure・Action / Host Effect** | 手順・業務上の判断・受理対象の版を定義し、ViewChangeを既存型付きIntent/Command/Providerへ渡す | Tab移動・Slider操作・Button表示だけから推論した業務責任、承認・実Commit |

**所有規則の提案**：再利用可能な*表現型の定義*は独立して存在してよいが、**データへ束縛された具体的な表現Instanceは必ず何らかのFunction/Tool FunctionのContext内に所属**する。Functionは表現を0個（Headless/Batch）、1個、複数個持てる。StudioのData InspectorやSHOのPreviewも、業務機能の第二正本ではなく「Inspect/PreviewというTool Functionが生成した派生表示」として説明できる。画面構成・Slotの入れ子構造をData Model上の1:Nや業務フローの階層へ自動転写しない。

この境界は既存[#2507](https://github.com/hat47x/tei-platform/issues/2507)のRequired Data Shape/Required Operation/Presentation Bindingに**Instance所有規則を追加する差分**である。データ形状の正本は#1126/#2495、形状間ProjectionとLossは#2497、Function Context/Patternは#2506/#2507、標準CRUD生成は#1127、Renderer/Framework/案件Slotは#997、Studio編集逆写像は#1124、表示編集からActionへの型付きIntentは#2500、実Host/Providerの資格は#1174が所有。ここで第二のRepresentation Registry/Parser/Owner Meaning/Hostを設けない。

### 13.2 型の独立性とFunction内での適合を混同しない

| 表現型の類型と候補 | 必要な束縛・適合の例 | 保持・非推論 |
| --- | --- | --- |
| **配置容器**：Panel/Tab/Split | 同一Function内の子Slot・表示順・Selectionを配置。容器自身には業務値の粒度を要求しない場合がある | Tab数・順序≠DB Relation/Cardinality/Workflow Step |
| **単一値**：Text Box・Checkbox/Toggle・Slider | Value Role/型、nullable/未取得/非開示、単位・精度、Sliderなら有効範囲・刻み・値変換 | NULLをfalse/0へ潰さない。閾値やstepを画面側から業務Ruleとして採用しない |
| **選択**：Radio/Select・Checkbox Group | 選択肢のIdentity・単一/複数性・値域・現在の候補集合と取得完全性、Source revision | 画面に出た選択肢が全対象とは限らない。選択だけでEntityやRelationを自動生成しない |
| **集合**：Table/List・Header/Detail | Row/Entity/Result grain、Identity、多重性、ソート、ページ・Filter・取得範囲。Header/DetailのRelation pathを明示 | JOIN増幅/集計表示/ページ分割後の行を、実Entity全件と誤認しない |
| **構造**：Tree/File System風/Graph | Node/Edge identity、方向・複数親・循環・辺属性、Selection、表示に使った展開規則 | Graph→Treeで隠れた辺を削除しない。「ファイル表示」≠実Filesystemへの読書き権限 |
| **解析・可視化**：各種Chart/時系列/地図等 | Aggregate grain、Window/Snapshot、欠損・NULL・単位・CRS、非可逆な写像を明示 | Chart点・棒/折れ線を、業務Rowへの編集可能な1:1逆写像とみなさない |

一つのData Shapeから複数の表現候補を選べる一方、一つの表現型は異なるData Shapeへ適用可能である。ただし**何を保持・失うか、Data/Value/Roleの必要条件、Function Actionとの関係**をFunction InstanceのBindingで検証する。名前やWidget種別の一致だけで自動適合としない。Displayが可能でもEdit/Select/Action Offerが可能とは限らず、実行可能性と開示可否は現在の権限・Provider・Consumerで独立判定する。

### 13.3 現行mainの観測と実装方針を混ぜない

[main統合 #3145](https://github.com/hat47x/tei-platform/pull/3145)の[固定Source実測](https://github.com/hat47x/tei-platform/blob/main/plan/evidence/OPERATION_SURFACE_DECLARED_PATTERN_GAP_2026-11-07/README.md)では、同じ参照Workspaceへ見積候補を重ねて`function-definition`にPattern 6件が現れWorld/Model revisionも変化したが、`tei operations`は依然**derived 8件・declared 0件・gaps 0件**だった。現main `internal/operations/surface.go`（確認Blob `ad414e82a19fb7572f02a2ea7c1e8bcd12434bbe`）では生成側で`Origin="declared"`が付く経路を確認できず、設計W3未着手という記録と整合する。

この実測から言えるのは、**宣言したFunction PatternのActionが現行の導出Operation Surfaceへ自動反映されていない**ことまで。だからといってPatternを無条件にCRUDへ変換する/一覧へ追加する判断をしない（Pattern `command`とOperation `list/read/create/update`の意味適合・効果が未決）。新しいRepresentation Slot/画面側の操作Offerが必要なActionを解決できるのか、`function-definition`のSourceと現行Operation Surface/Service Carrierを**明示的に接続する受入**が先となる。

[main統合 #3120](https://github.com/hat47x/tei-platform/pull/3120)でStudioの検証画面にSource版付き・`Accepted=false`の修正案表示が加わったことも、**表現と正式な採用・実行の分離**を裏付ける限定Referenceであり、表現型体系の採用/実GUI統合を証明しない。

### 13.4 一つの受注Function/同一Data Contextで既存独立検証基準へ加える反例

以下は既存#2586の親H・明細D1(2行)・独立配送D2(3行)の**同一公開合成Fixtureと独立期待値**へ追加する比較観点。別のFixture正本・番号体系を作らない。

1. **同じD1、Table↔Tree↔Chart/CUI**：表示を切替えてもL1/L2のEntity identity、必要ReadのScopeとOwner意味は不変。JOIN 2×3=6は6業務Entityにならず、集計値1本は1レコードへ無根拠にWriteBackできない。
2. **Panel/Tabだけ追加・並べ替え**：DataのRelation、Business ProcedureのStep順、Action Effectは変化しない。別Functionへの遷移をLayoutのTab構造から推論しない。
3. **Graphの多重親をTreeへ投影**：第二親/循環/Edge identityは投影損失を明示。Tree上のMove/Deleteを元Graphへ曖昧に逆写像しない。
4. **Single/Multi/ToggleのNULL・partial**：未取得/非開示/NULLをfalseや未選択と断定しない。候補集合が部分取得なら「他の候補なし」と言わず、現在Selection epochを固定。
5. **Sliderの数値範囲・単位**：例えば税率/金額/時間を同じNumber Sliderへ表示できても、Role・scale・丸め・業務閾値・承認条件の同一性は成立しない。UI min/max/stepはOwner受理済みBusiness Constraintに昇格しない。
6. **Function Patternの宣言追加**：#3145のとおり現行`tei operations`が不変でも、それを「Actionが存在しない」「Actionが選択済み」「全Offer検証済み」のいずれにも転用しない。必要なPattern/Capability/Source/Operation identityを未接続として報告。
7. **同じFunction Contextに複数Widgetを配置**：一方のButtonやChartの表示可否を、他Slotが参照する非公開Dataset/Grantの存在へ連動させない。Action Offerを出す場合は現在Disclosure/Target/Host Grantと実Provider Routeを再照合。
8. **古い選択を経由した編集**：表の行L1選択後に選択ContextがL2へ変わり、旧Graph/Sliderの操作が遅延到着した場合、旧epoch/Source/RowVersionから新対象へのActionを生成しない。

**実施順**：P0は#2586で同一Read-only Context→表/木/グラフ/CUIのCandidate適合・Identity保持/損失・未対応を比較すること（#2497/#2507）。次に#997/#1124でFunction内の複数SlotとRendererの差替え/編集逆写像を確認し、#3145のOperation Surface Gapを#1127/#2507のOwnerへ渡す。最後に必要なActionだけ#2500/#2598/#1174で現在資格・実Receipt/保存後独立Readを確認する。**Headless Functionが画面なしで成立すること**も正例に含める。

**証拠資格**：ここまでの表現型分類・Function所有規則・具体的Binding契約は**設計案／Owner未受理**。#3145の既存CLI実測と#3120の局所試験は確認できるが、今回のTable/Tree/Chart/Widget比較、`*.tei.md` Reader、Web/Studio GUI、実Provider・Host・Owner Meaning/Business Acceptanceの実行は**NOT RUN**。新UI Core、独自Host、二重意味正本、万能表現型Editorを導入しない。

<!-- tei-triad-binding-topologies-media-axes-20261107 -->
### 13.5 データ構造／Function内表現／Bindingを三つのトポロジーとして接続する

**既存Ownerを優先する**：[Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)の`STANDARD_REPRESENTATION_TOPOLOGY_AND_MEDIA_AXES_2026-11.md`は、標準表現型の構造、時間・音声・動画・文書の軸適合、RP-01〜08の独立反例を**詳細設計の正本候補として所有**する。本節ではその反例・Grammar・共通型を再定義せず、Data/Function/Businessの**意味・作用・Consumerへの接点だけ**を記述する。#2497（写像と損失）、#2506（Context）、#2507（Pattern）、#997（Renderer）も各々のOwnerを維持する。

| 層 | 当該Functionで束縛すべきもの | 誤って昇格させないこと |
| --- | --- | --- |
| **Data topology** | Entity/Edge/OccurrenceのIdentity、関係向き/多重性、時間/単位/CRS、粒度と取得完全性 | JOIN行や画面Slotを元のEntity/Relationと同一視 |
| **Representation topology** | Function ID内のSlot ID、Tab/Panel/Tree/Chart/Media Track/文書ページの配置・選択・軸 | Slot順序をBusiness ProcedureのStep、表示ページをSource段落、表示GraphをデータGraphの完全表現とみなす |
| **Binding topology** | `Source Role/Occurrence/Path/Grain → Function Slot/Dimension`を明示対応、Revision/Operator、情報損失/取得Scope、逆写像可否を検査 | 表示可能性から編集・Action可用性/Host Grant/実Effectを推定 |

**Triad計画の最小の検収境界**：同じH/D1/D2をTable/Tree/Chart/CUIに配置しても、D1実体2件・JOIN投影6行・集計棒1件の粒度とBusiness Claimの必要集合は不変。Tab並べ替えや媒体再生位置の変更はBusiness/Sourceの更新ではない。異なる単位・時間軸・文書ページ・Graph多重親は#3192のRP-01〜08と#2497のLoss条件に委譲する。表現上の編集から一意なData Targetへ逆解決できず、現在Version/Scope/権限も欠けるなら、`ViewChange→Intent`は未資格としてread-onlyに留める（#2500/#2598/#1174）。

**複数Instance間の同期は別の責務**：[Draft #3222](https://github.com/hat47x/tei-platform/pull/3222)の`REPRESENTATION_CROSSVIEW_COORDINATION_2026-11.md`がSelection/Focus/Filter/Seekの同期、Subject・Path・集計group・View Occurrence・Media timeのIdentity差、MV-01〜08の独立対照を所有する。本計画では、**表示上の選択が他Slotへ同期されたこと ≠ 一意なBusiness Entity/更新Targetが選ばれたこと ≠ 現在のAction Offer/Grant取得**というTriadの受入境界のみを引き受ける。元Sourceや業務データを変えるのは明示的な型付きIntent以降であり、表示同期をDomain Mutationとして自動保存しない。部分取得・古いepoch・非開示には同期結果の不確実性を保持する。

**導入順**：#3192でトポロジー分類と独立Expected→#2586の一Function/ContextでBinding/Lossを観測→#997/#1124でRenderer/Authoring→#2500で逆写像→#1174/#1163でHost/Provider Effectを別検証。新Core Widget enum、別Mapping/Meaning Registry、第二Fixture正本は作らない。媒体間同期・ページ/音声化の製品E2E、正式Owner受理は本PRではNOT RUN。

### 13.6 実Patternと導出Operationは同じDataを指すだけでは結合しない

[main #3167](https://github.com/hat47x/tei-platform/pull/3167)は同じWorld内の宣言Patternと`tei operations`を比較し、Data IDの一致候補はあってもlist/read等のKindを決められず、Capability ID/Relation Targetの一致は得られないと実測した。読み取り専用の`preview_order_quote` `command`には対応する導出Operationが無く、`binding=null`は副作用を宣言しない能力を勝手に書込みへ昇格しない結果である。

[main #3172](https://github.com/hat47x/tei-platform/pull/3172)はW3のKind対応表/Relation変換/Command一致/複数Kindの優先順位を**採用前の設計候補**として追加したが、**W3自体は未実装**。Function内Representation SlotがそのDataを表示できても、正しいActionが発見される・導出CRUDに置換してよい・現在Grantがある、のいずれも推論しない。#1127/#2507の同一受注例で「Data IDだけ一致」「Capability不一致」「effectなしCommand」「明示Accepted Action Binding」を別対照とする。Actionの意味をRendererへ重複定義せず、実Invoker/Provider資格は#2598/#1174が担当する。

<!-- tei-triad-autonomy-offer-qualification-20261107 -->
### 13.7 選択同期・操作発見・自律区分・現在権限は、同一Functionでも別の資格

[main #3226](https://github.com/hat47x/tei-platform/pull/3226)で、Interactionを持たないPattern Viewの処理が統合された。実Source `internal/applicationview/view.go`（確認Blob `e5faa4631129ae35ba20da42056a3322c9672325`）では`pattern.Interaction == nil`なら空の`application.Interaction{}`を使ってDataのViewを組み立てる。**Function/Patternの閲覧と、操作やCommandの存在は同義でない**。従って標準表現型Slotを表示できるだけではAction Offerを製品が提供したことにはならない。

[調査PR #3231](https://github.com/hat47x/tei-platform/pull/3231)は現行`tei operations`の自律区分を、参照・CRUD合成Workspaceで実測した**未統合の記録**。実Source `internal/operations/surface.go`（確認Blob `ad414e82a19fb7572f02a2ea7c1e8bcd12434bbe`）の生成箇所はList/Readへ`Autonomy="unattended"`、Create/Updateへ`"attended"`を付け、`human-gated`生成は確認できない。設計`OPERATION_SURFACE_DESIGN_2026-11.md` §8は自律の判定候補としてサービス主体、冪等性、人的判断の有無を記すが、**現行Derived Operationの自律ラベルはそれらを検証した結果ではない**。これを実装欠陥・正式自律Policy違反と断定するのではなく、表示ラベルをどこまで業務/AIが根拠にできるかという**限定事項**として扱う。自律判定W4の受入・採否は#1127/Operations Surface Ownerが担当する。

| 同一Functionの利用段階 | 必要な観測 | そこからは言えないこと |
| --- | --- | --- |
| **Representation Display** | Function/Context/Role/SlotとReader/Disclosureの表示資格 | PatternにInteraction/Commandがある、操作可 |
| **Cross-view Coordination** | Context Instance、Source/Result/Path/View Occurrence、Selection epoch、非開示の切り分け（[Draft #3222](https://github.com/hat47x/tei-platform/pull/3222)） | 選択されたView位置が一意なBusiness Entity・更新Targetである |
| **Operation Discovery / Action Offer** | Pattern/Capabilityの実宣言とOrigin、Data IDに加えKind/Target/Binding適合。#3145/#3167の未接続を検査 | `tei operations`の数・`gaps=0`だけで宣言Actionが全件ある／実行可能 |
| **Autonomy classification** | 採用された区分規則、その判定に実消費したサービス主体・冪等/人判断等の証拠、区分の対象Scope/版 | `unattended`だけで現在のGrant・外部送信許可・ビジネス承認がある |
| **Actual invocation / Effect** | 現在Host Grant/Resource、Intent→Request Target、Provider Effect/Receipt/資源独立Read、必要なBusiness Meaning | read-only、画面enabled、HTTP成功、Receiptのどれか一つでBusiness完了 |

**既存固定H/D1/D2・O1/C9での非重複対照**：

1. **Viewのみ**：Patternに`interaction`が無くてもDataは表示可能。Operation数0/Command欄空を壊れたDataと見なさず、表現Instanceへ架空のWrite Actionを補完しない。
2. **表示間Selection**：L1をTableからChartへ伝えたがChart集計groupはL1と1:1でない。選択・Focus・Filter・Seekは表示状態であり、Business Targetの編集・Action Offer/Grantを暗黙に生成しない（#3222 MV）。
3. **`unattended`を持つread**：Data Contextはpartial、現在Consumer Disclosureが許さない、または外部送信先が未承認。**自律区分とRead/Disclosure/Delivery資格が別**であることを示し、非公開Dataを画面/AI出力に出さない。読み取り専用は「外部へ一切送らない」ことも意味しない（#2515/#2516）。
4. **`attended`を持つwrite／人的承認候補**：UIが操作可能でも、実Host Grant、必要な現在Business Approval、Operation Scope、Rollback/Unknown Effect判断を再取得する。設計上の`human-gated`を実装が出力しなかったことだけで、人による判断が業務上不要だとは推定しない。
5. **Patternを追加してもOperation不変**：Data ID一致に留まり、Kind/Capability未一致なら未接続と報告。Operationが`unattended`でも新Patternの宣言読込やCapability Targetの適合は別に検証する（#3167/#3172）。

**Owner/実証の割付**：PatternとRepresentation適合は#2507、表現間Coordinationは#3222、Derived Operation/W3/W4は#1127と[PR #3231](https://github.com/hat47x/tei-platform/pull/3231)、Current Context/Selectionは#2506/#2582、Action変換は#2500/#2598、DisclosureとDeliveryは#2515/#2516、実Host/Effectは#1174。新Autonomy Policy/Prompt Approval Engine・状態Enum・第二Action Registryを本計画に新設しない。**本節は既存Go Sourceと#3231合成CLI観測の引渡しであり、今回この5対照の製品E2E、実権限・実媒体/GUI・Owner正式受理はNOT RUN**。

<!-- tei-triad-domain-operator-role-binding-time-unit-20261107 -->
### 13.8 時間・金額のRepresentation適合と、期間に依存する業務Operatorの入力閉包を分ける

標準表現型のDate/Time、Number、Money、Chart/Timeline等への表示適合と、Functionの業務計算に必要な**適用期間・有効時点・単位・Role・演算規則**は別資格である。この差については、X-TEA継承の参照実装が2026-11-07 mainへ進んだため、抽象論ではなく次の3つの**限定実装**を既存のFunction Data Context受入へ接続する。

| main統合済みの参照実装 | 確認済みの振舞い | Function Context→実業務へ残る境界 |
| --- | --- | --- |
| [#3151 taxrate](https://github.com/hat47x/tei-platform/pull/3151) | RateのCategoryと両端を含む適用期間、宣言された端数処理を使用。該当Rate無し／複数Rate競合は`ErrInvalidRate` | 受注O1の税区分・計算対象日付・金額のRole、税率Dataの実Provider行、Source/Rate版、Owner Ruleの受理。Chartの日付やSliderの数値だけから税率を選ばない |
| [#3154 currencyrate](https://github.com/hat47x/tei-platform/pull/3154) | 通貨・年度/月・率の種類、掛け/割り方向と端数処理を宣言。ゼロ率・行欠落・重複を正当な率と扱わない | 金額通貨と基準通貨、適用年度/月、方向/Rateの取得Scope、選択元Source/版。表示通貨記号だけで換算済みのBusiness値を断定しない |
| [#3232 businesscalendar](https://github.com/hat47x/tei-platform/pull/3232) | 暦が名指すHoliday Dataの`holiday-date` Roleを解決し、日付の不正・時間帯未指定を診断。**Holiday Dataを宣言しない暦**は「休日なし」と解釈 | **宣言が存在しない**場合と、**宣言された休日Dataをまだ読んでいない／部分取得／権限で隠された**場合を分ける。後者を空の休日集合に自動変換しない。暦別の複数Holiday Data識別や実Provider E2Eは未接続 |

**横断の反例を一つに束ねる**：同一受注O1の見積Previewの数値600がTable/Chartと同じ表示であっても、税額の端数処理・Rateの適用時点・換算方向・休日条件が異なるとBusiness Claimの内容は異なり得る。表現側のDate/Decimal/通貨ラベルが等しいだけでは、Functionの入力行集合と業務Operatorの意味は同じにならない。率が欠けたときの近傍日付への推測Fallback、休日Dataを宣言したのに未取得なのを「休日なし」、同じ表の別暦行を混入、許可されたScopeより広い行の計算使用、過去の承認に現時点のRateを無言適用、を**既存業務Operatorの負例**とする。復元された過去値が一致しても承認Revision・Operator版・実Read資格は自動復活しない。

**着手順と非重複**：表示軸・媒体適合は#3192/#2497、税・換算・休日の演算自体とDomain Referenceは各PRの既存Owner、Function内Role/Scope/Result Sliceは#2506、Source→Actual Readerは#2586、Business Rule/確定値は#1125、独立期待値は#2514に委譲。最初はRead-onlyの一Functionを用いて`Role＋期間＋実行されたOperator＋Actual Input Rows＋結果のUnit/Precision/Source`を同じAttemptで確認し、Provider実Readまで接続してからBusiness Claimへ進む。**今回の三つの参照実装を受注O1の実Function・*.tei.md・実PG・GUIに接続した新試験はNOT RUN**。第二Tax/FX/Calendar Engineや新State Enumは作らない。

<!-- tei-triad-real-function-witness-matrix-20261107 -->
## 14. 11/7横断追補：同一Function・同一Actual Rowsetから表現・業務Claim・Client反応までをつなぐ最小Witness計画

### 14.1 最新の実行証拠が閉じた範囲と、残る一辺

main統合済み[ #3264 ](https://github.com/hat47x/tei-platform/pull/3264)は、対象Issue→read-only CLI 9種→Source Index/版照合→必要SourceのLoader読込（及び欠落/異版/未読込負例）を**限定CI**で実行した。しかし対象Domain/FunctionのRequired Role/Rule/Procedure/ContextがすべてParser/Meaningで解決されたとは言えない。[ #3266 ](https://github.com/hat47x/tei-platform/pull/3266)は同じ実バイナリでpinned Scenario 6 Step、実CLIで4種の負例、独立義務の2負例を検出した。限定的なstatement coverageは約10.1%の**局所C0相当**であり、C1・未カバー機能要件/非機能要件・生成AIが新規に作る負例の妥当性を証明しない。[ #3268 ](https://github.com/hat47x/tei-platform/pull/3268)は関連Source/Scenario/Skill変更による再検証起動範囲を拡張したが、完全な依存追跡ではない。

[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238) の**exact head `d3b67d5809478fad828052ce01bdafe576b2f934`**では、実隔離PostgreSQLから同じ固定FixtureのTenant A Rowset `{3,9}`と`TOP1={3}`を取得し、別の非出力列`ordered_at`のSQL Predicateで`{9}`が得られることを独立Expectedで照合した。[実PG CI #42351796330](https://github.com/hat47x/tei-platform/actions/runs/42351796330)・[Health #42351796383](https://github.com/hat47x/tei-platform/actions/runs/42351796383)はともにcompleted/success。これは**Web ReferenceのScoped Reader**の実証であり、TEIの受理済み**Function→選択Query→実Product Provider→Context/Claim/Consumer**を同じAttemptで通した証拠ではない。#3238は**Draft/未統合**。

残るP0は[#2586](https://github.com/hat47x/tei-platform/issues/2586)の**一Function・一Query・一Actual Provider Read**である。機能の宣言やPanel/Table描画、Source `readByLoader:true`、別ReferenceのPG Greenを足し算して、この一辺の成立にしない。

### 14.2 同じ受注Functionの「必要な証拠」と判別点を縦にそろえる

| 段階（同じSource/Function/Target/Attemptに限定） | 揃える観測根拠・最小判別点 | 未充足時に禁止する主張 | 既存Owner |
| --- | --- | --- | --- |
| **D0：Design Context取得** | Subject/FunctionとRequired Data・Rule・Business Procedure・Context・Source Identity/Revision・Loaded/Parsed/Accepted/Unknownを**scope相対**に一覧。欠落/部分/非開示を区別 | catalog件数/CLI `ok`だけで必要設計を全取得・受理済み | #3262 / #2822 / #2823 |
| **R1：Functionから実Read** | Function ID→Pattern/Query/Provider/Actual SQLまたはnative selection→Tenant/Scope/Predicate/Projection→**実Rowset**/Field/Result Grain/Read Cohortと物理取得時点を連結 | 別Reference PG PASSをそのFunctionの実取得に読み替える | #2586 / #2387 / #2506 / #2622 |
| **R2：Claim相対の完全性** | 事前固定L={3,9}とActualを独立比較。`TOP1={3}`はPartial、同件数重複やPredicate未取得はUnverified、完全な母集団だけが対象Predicateに必要な範囲でComplete | 誤Caller申告`CoverageComplete`をEvaluator戻り`holds`で自己検証 | #2514 / #1180 / #2506 / #1163 |
| **R3：非出力Predicateと開示** | `WHERE ordered_at>=...`等の非表示列依存をQuery/Lineageに残す。二つの許可WorldでRow Membership・件数/順序/診断/Preview/Tool Logの開示差を調べる | $selectで列を伏せたことを、Predicate依存の推知やTool出力の非干渉の証明に使う | #2516 / #2515 / #2506 / #3238 |
| **P1：Function内の複数表現** | 同じContextのTable/Form/Chart/TreeでEntity/Occurrence/Grain/Revision/Selectionを束縛。集計・JOINの非単射写像はread-onlyまたは明示Inverseのみ | Renderer選択や同期した選択位置から一意な業務Target/Actionを生成 | #2497 / #2507 / #997 / #2500 / #3236 |
| **A1：操作・権限・作用** | Patternと導出OperationのKind/Capability適合、現在Grant/Target/Version、Intent→Command→Outcome/Effect、ReceiptとResource readの区別 | `unattended`やenabled、`persisted:true`だけで全Entity/業務完了/承認済みと説明 | #1127 / #2598 / #1174 / #2998 |
| **C1：TransportとClient Reaction** | typed Business Outcome/EffectUnknown/Receipt/ReadをHTTPやDesktop IPCへ**輸送手段別に投影**し、Client側のRetry/Refresh/Reconcileと表示・非開示を別観測 | HTTP 200/202・Problem text・タイムアウトから「業務完了／確実に無作用」と推測 | #3274 / #1656 / #1158 / #1174 |

本表は**既存Issueへの受入引渡しを同一Attemptで見渡すための比較票**であり、製品の新しい`D0/R1...` Enum・Graph/Proof Engine・第二Scenario SSOTではない。初回は**Read-onlyのR1とR2/R3まで**をP0とし、UI/Action/EffectをP0の必須前提に昇格させない。Source・Read・Consumerが同一の実証に閉じていない行は`not-connected/unverified`として明記する。

### 14.3 既存テストへの追加判別：結果が一致しても実施範囲と開示は別

同じ公開固定Fixtureを利用し、#3238の専用Scoped Readerと#2586の製品Function入口を**別Implementation/Witness**として対照する。新しいRow FixtureやExpectedの正本は作らない。

1. **実Readの足場とQuery Originの不一致**：独立期待L={A/3,A/9}。隔離PGで`TOP2=[3,9]`、`TOP1=[3]`を取れた既存結果は、製品Functionの接続の**必要な参考**だが代替ではない。main実Source `reference/workspace/applications/order-entry/queries/recent_orders.tei`（blob `a570bceddc7d61aae62ac315a886cc9aedcac082`）は`sql postgres file "recent_orders.sql"`／`datasource sales`を宣言し、SQL（blob `ecc4fb7daab99bd8f03d490f56f7af47c9ebd97a`）は`FROM public.orders`、`WHERE ordered_at>=:from`、`ORDER BY ordered_at DESC`。一方、[隔離PG Reference](https://github.com/hat47x/tei-platform/blob/main/reference/web-api-contract/scoped_postgres_reference.go)（blob `b3f823e2e621e2cbbda0ea8c83c1ffdc3a166211`）は`public.tei_web_scoped_orders_reference`という**別テーブル・Tenant条件・表示列/順序/Top契約**を用いる。**同じ論理Query IDや同じ行IDが現れても、実消費SQL・対象母集団・Order/Scopeが一致した証拠にはならない**。最初のProduct Function ReadではFunction/Pattern実選択→Query Source/SQL byte→`:from`値と時点→`sales`の物理Provider→実Statement/Args→Rowset/Field/Cohort→Contextを一Attemptで観測する（既存[#2586のHR-01/02](https://github.com/hat47x/tei-platform/issues/2586#issuecomment-6704012671)がOwner）。
2. **完全性と判定**：`TOP1`で`CoveragePartial`なら局所Completionは`unverified`。誤って`CoverageComplete`を申告して`holds`になったときは評価器の**入力信頼境界の反例**として保持。必要なField `status`を取得していないなら`orderId`のみから状態述語のHoldsを作らない。
**Field Presenceの追加注意（既存CDE-Q08）**：実Scoped Readerは非選択の属性を生のSQL結果で`NULL AS status`として返し得る。Raw Slotの`nil`は格納済み値がNULLである証明ではない（Fixture上の`status`はNOT NULLの場合がある）。`Selected`・Physical SQL投影・Column Provenanceと`stored-null / not-selected / withheld / unknown`を別に扱い、UIのNull表示・Business Predicateの偽を捏造しない。#2506と[#2586の既存引渡し](https://github.com/hat47x/tei-platform/issues/2586#issuecomment-6703995604)にOwnerを置く。新しいResult Schemaや第二のQ08 Fixtureは作らない。

3. **非出力Predicateは閲覧秘密を保証しない**：`$select=orderId`とし、SQL内で`ordered_at`の条件が実評価されれば、返却する行IDや件数が条件を間接的に伝え得る。正当なQuery実行権と、そのResult Membership/Count/Tool Logを現在Consumerへ開示する資格は別。#2516の二World差異試験と連結する。
4. **同じActual Contextから二表現**：TableでL1/L2を表示し、Chartで合計を表示する。L1/L2の2実体、JOINによる6行、集計棒1件の異なる粒度を保つ。部分D2や古いSelection epochから全件不存在Claim・一意Targetの書戻しへ昇格しない。表現合成の非単射/CXケースは[統合候補 #3236](https://github.com/hat47x/tei-platform/pull/3236)がOwner（旧#3192/#3222は未統合Draftのまま）。
5. **Clientが副作用を知らない**：リードだけの初回P0と混同せず、将来同じ受注のCommandがHTTPでタイムアウトしたらEffectUnknown/Receipt Lookup/Current Grant/Resource観測に沿ったClient Reactionを検査。`202 Accepted`で業務完了としない。[main統合 #3275](https://github.com/hat47x/tei-platform/pull/3275)の[Decision Graph設計](https://github.com/hat47x/tei-platform/blob/main/plan/design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)は**設計/NOT RUN**であり、実HTTP/Client実装の資格ではない。

### 14.4 テスト生成・Provider/Plugin/BOM/Cacheへの依存を増やさない順序

**テスト能力（11/7後続補正）**：T0はmain #3264/#3266/#3268の限定実証で進み、[main統合 #3304](https://github.com/hat47x/tei-platform/pull/3304)では#3273で**既知の行番号0 Rule候補（T1-A）**を、生成済Pinned実TEI CLI／既存6 Step＋一時第7 Step／偽Expected・偽Fixture・古いPinの3負例で実行した（[限定CI #42378033965](https://github.com/hat47x/tei-platform/actions/runs/42378033965)は当該旧実行HEADでsuccess）。しかしT1-Bの**未発見Ruleの盲検探索／AIによる新規Case独立生成**やC1/要件・NFR Coverageは未実証。T1「#3262 Design Contextから未カバー義務を発見し、生成AIが新しいCaseを提案、独立Expected/Mutation/実CLIにより判別」の全面資格は[Draft #3273](https://github.com/hat47x/tei-platform/pull/3273)の計画段階を出ない。C0局所測定とC1/要件・NFR Coverageは別。Scenario正本は#1180、独立期待値は#2514、Coverage不足提案は#3223/#3273。**Product Function→Provider R1未接続を、T0のGreenやAIが生成したScenario数で相殺しない**。

**拡張範囲**：[main #3276](https://github.com/hat47x/tei-platform/pull/3276)がData Source Workbench RQ-035（Schema CRUDは標準、DBA運用は非標準）、[main #3277](https://github.com/hat47x/tei-platform/pull/3277)がCore外Plugin化とCLI/Studio自身を使うdogfood計画を追加済み。対象の実Provider能力・Reader Scope・Action Grantを解決して初めてこのR1入口を選択する。Plugin/DB/Cloud実装を全件P0としない。#3267（Verified Composition BOM）はExact Source/Contract/Target/組合せ試験の再利用資格、#3271（Cache/Snapshot）は観測時点・依存Closure・失効資格の再利用を所有する。**単体Provider/Plugin個別PASS、Cache hit、BOMの構成一覧は、実Functionの同一Attempt R1/R2成立の代替にならない**。

**順序**：既存#1119のGatesを増やさず、(i) #3262で対象設計Required閉包の不足を可視化 → (ii) #2586の一Function・一Query・一実Provider R1 → (iii) #2506/#2514のCoverage/非出力依存と#2516の開示 → (iv) #2507/#997の同Context複数表現 → (v) 必要な実Action/Host/HTTP Decision Graph → (vi) #3267/#3271で結合済み実証の適切な再利用とT1新規負例、という順。Read-only R1に実HTTP/本番認可/媒体Rendererを**前提要件として過剰に課さない**。本書はOwnerの採用・本番許可やProduct E2/G2合格を決定しない。

**証拠資格（2026-11-07時点）**：この追補は既存main Source/PR exact-head CI/Issueに基づく**文書更新**。#3238の実隔離PG・#3264等の実CLI/Scenario実行は各担当の既存限定証拠であり、**今回**新たなGo/DB/Harness/Studio/Clientを実行したものではない。実Product Function→実Reader→Context/Claim→Form/Table→Client、Owner業務意味受理、実Host Grant/実Provider完了の同一試行は**NOT RUN / NOT ACCEPTED**。

<!-- tei-triad-parallel-modeler-read-and-context-readiness-20261107 -->
## 15. 11/7横断追補：三視点Modelerと実Function Readを並行させ、結合資格だけを後段で検証する

### 15.1 一つの直列P0で塞がず、「意味からの図面」「実Read」「二者の照合」を分離

[Draft #3303（旧#3236の現main向け置換候補）](https://github.com/hat47x/tei-platform/pull/3303)は標準構造/表現型の合成写像の統合候補を更新し、**P0-M（三視点Modeler read-only）とP0-R（実Function→Provider Read）を互いに独立した最小実証として並行実施**し、次に**P0-C（cross-view結合）**を行う順を示した。これは§14が最初に強調したP0-Rを撤回するものではなく、#1124の旧X-TEA三視点図面／CRUD cross-referenceという**モデリング製品固有の主要価値をProvider実装待ちにしない**ための境界修正である。

| 並行レーン | 最小の入出力・独立期待値 | まだ主張しない資格とOwner |
| --- | --- | --- |
| **P0-M：三視点のread-only導出** | 同じ受理Source/RevisionのData Entity/Relation、Function/Pattern、Business Procedure/Step/Actorから、Data・Function・Businessの3図とCRUD Matrixを**Headless CLI/JSONの投影**として説明。Node/Edge/Source Location、宣言と派生、欠落/非開示/部分集合を保持。 | Product実Row Read、実Effect、画面上編集・逆Patch、業務Ownerの新規意味受理は未実施。#1124/#1125/#1126、[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)が設計・検証Owner |
| **P0-R：Functionの実Read** | 実Function/Pattern→実Query Source/SQL byte/parameters→Physical Provider/Scope→Actual Rowset/Field Presence/Read Cohort→Contextを**一Attempt**で接続（§14）。 | CRUD表のC/U/Dの実行、図面の編集資格、全Function一般化を主張しない。#2586/#2506/#2387/#2514 |
| **P0-C：Model/Readの照合** | その**同じSource Subject/Workspace/Revision・Function/Query Binding・Entity identity/grain・Scope/Time**を使って、P0-M側のData/Function/CRUD RowとP0-R側のActual Read Slice/Consumerを照合する。未一致は解決せず差分として返す。 | Nodeのラベル一致だけで意味・取得対象/Host Grant一致を確定しない。合成写像#3303（旧#3236）、正本/投影#2497、同一試行#1163、設計取得#3262 |

**非推論**：CRUD表の「R」はFunctionが対象データを**参照すると宣言/派生された**ことを表す分類であり、物理RowsetをそのFunctionが取得したことのReceiptではない。C/U/Dも実業務更新の記録やAuthorityではない。またP0-Rの実PostgreSQL成功を三視点図面の正本・意味受理へ逆流させない。**P0-M単独とP0-R単独の独立合格を尊重**し、双方が揃っていないことだけで「図面導出不可」としない。

### 15.2 Design Contextの「探索利用可」と「検証済みモデル/テスト生成」を区別する

main統合[#3284](https://github.com/hat47x/tei-platform/pull/3284)は`tools/reference-issue-test-ability/context_gate.py`を導入し、続く[main統合 #3296](https://github.com/hat47x/tei-platform/pull/3296)が**同じGate**でread-only CLI Reply 9種、既存7負例＋Source Root/Manifest 2負例＋Tool ReplyのByte改変7負例、計16対照へ強化した（[限定CI #42371310577](https://github.com/hat47x/tei-platform/actions/runs/42371310577) success）。実Sourceでは**正常な限定Referenceでも**`boundedReference=partial`、`testExecutionAdmission=exploration-only`、`qualifiedTestGeneration=false`。初版#3284と拡張#3296の**個別実行証拠**を区別し、Tool Reply Byte負例は受領済みJSONの一時コピーの合成改変であって実外部Harness通信の完全性保証ではない。**AIが新規テストを設計できる製品資格が成立したわけではない**。

この実装を三視点Modelerへ**転用する際の判断はscope相対**である：

- **P0-M探索・候補図面**：必要なData/Function/Business Sourceが部分なら、観測できた部分を`candidate/partial`として表示できる。ただし未知のRelation/Function/Actorや非開示Edgeを「存在しない」と描かず、Matrixの空セルでCRUDが無いと主張しない。
- **P0-Mの完成Claim**：必要な三視点のSubject/Source/Meaning/Relation/Function/Business閉包、版・Scope・Source Locationが受理可能な形で揃った場合に限り、対象範囲内の完成を判定する。既存Gateの9 Replyはこの全面閉包の証明ではない。
- **テスト生成候補**：探索用に負例を提案できても`qualifiedTestGeneration=false`を変更しない。**独立Required/Owner RuleとExpected、Source version、Mutant検出力、C0/C1およびNFR coverage gap**を既存#3223/#3273/#2514/#1180で検査した後に用途別に認める。テスト生成資格の正本は#3262/#3284のOwnerとし、本計画に第二Gateを作らない。
- **Consumer受領**：CLI JSONが正しくてもMCP/外部Skill/Studio/SHOが全Replyを受け、権限内で同じ結果を再構成できたとは言えない。古いFP、途中切断/欠落Tool Reply、非公開行の件数等による漏洩を別Consumer別Witnessにする。

### 15.3 既存Fixtureに対する独立の負例と最小進行順

| 対照（新Fixtureを作らず既存Sourceを再利用） | 期待する判定と配置 |
| --- | --- |
| **M+ / R?**：受理Sourceから三図とCRUD Matrixを導出し、物理Providerは未接続 | 三視点read-only投影は対象Sourceの範囲で評価可能。Rセルは実Read成功を意味せず、P0-R/P0-Cは`not-observed`（#3263/#2586） |
| **M? / R+**：#3238の隔離PGでRowsetは読めたがFunction選択・Model Source/Query byteは未照合 | 実参照Readerの局所Readは保持。TEIの受注FunctionのCRUD R/Business Meaningへ自動帰属させない（§14.3、#2586） |
| **M+ / R+ / Revision違い**：同名のEntity/Function、旧Sourceや別Workspace/Queryへ束縛 | P0-Cはunresolved。Labelや行ID一致を同一Subject/同一SQL/同一Attemptとみなさない（#1124/#1163） |
| **図の同名2 Node/多重EdgeとJOIN 2×3** | Entity/Relation Edge IdentityとOccurrence、多重性を保存。集計棒・表の6行・Graph表示Node数をCRUD対象Entityの6件とみなさない（#3263/#2497） |
| **業務Actorの手作業StepとCUDの宣言** | 手作業・潜在的Function参照と実Command/Grant/Receiptを区別。Actor≠認可Principal、導出Potential Path≠実Effect（#1125/#1174） |
| **未読込Business Source／非開示Relation** | 図・Matrixには`unknown/partial/withheld`の資格を残し、空セル・空集合を「業務上不存在」へ昇格させない（#3262/#2516） |
| **Source/Rule変更後に図だけ再生成** | P0-Mの差分は表示可でも旧P0-R/旧Test T0/旧Artifactの適合を自動再利用しない。Source+Query+Context/Mapping+Consumer/Targetの失効範囲を比較（#2582/#2600） |

**重要な独立Expectedの上限**：[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)には三図/CRUDの合成入力と期待JSONが別ファイルで存在し、内部整合15項目の静的確認が記録されているが、**両方ともAI草案**。出力Expectedがファイルとして分離されているだけでは、#2514が要求する意味上独立した検証基準・Mutation検出力・Owner受理にならない。P0-Mを実Go/CLI実行し、既存Mutationを誤成功として通さない追加試験が必要。#3303が継承する旧#3236のRP/MV/CXケースをこの表で再定義・複製しない。

**実装順**：P0-Mは既存Source/CLI/Modeler ProjectionからData→Function→Business→CRUDをread-onlyで独立実行し、P0-Rは§14のSource→Actual SQL→Provider行の接合を独立実行する。P0-Cは双方の**受理Source/版・Subject/Role・Function Binding・実Data同一性・Scope/時点**が揃った後だけ実施する。図上編集、SHO文書Consumer、Studio GUI、Host Write、HTTP Decision GraphをP0-Mの開始条件にしない。P0-C成功後も現在Grant/Business確定とは別資格。#1119の既存Gate・Critical Pathを変更せず、担当Issueの実証を受入可能な形で横断照合する。

**証拠資格**：main #3284の限定read-only Consumerテストや#3238の隔離PG、Draft #3263の合成Artifactを参照した**設計計画の更新**であり、この§15の三図・CRUD生成/Mutation/Function Product実Read/P0-Cを新たに実行したわけではない。人間/OwnerのMeaning採用や実Web/Studio/SHO/本番Host・Providerの一往復は**NOT RUN/NOT ACCEPTED**。新Source Registry/Renderer/Acceptance Engine/独立Expected正本/テストRunnerは設けない。

<!-- tei-triad-crosscanvas-two-identity-20261108 -->
## 16. 11/7〜8横断追補：多Canvas共有と三視点Graph/CRUDを「意味Subject・表示Occurrence・実データInstance」の三つの識別単位で検収する

### 16.1 同じ「受注」を指しても、Modeler Node・Canvas上のカード・実業務Rowは別

[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)の現行合成入力`reference/diagram-crossref/pressure-input-v0.json`（確認Blob `26bd66d01bd97fc0647fda0c511faf3d62af75f9`）は、**Entity 9 / Relation 7 / Function 4 / Business Action 3 / Canvas 2**と、Canvas上の**9つの表示Occurrence**を区別する。別指定Expected `expected-v0.json`（確認Blob `b9ab9c2a4fe559db6afd48e841eb2ef178d83bb0`）も、複数Canvasで同じSubjectを共有しつつOccurrenceは異なる、という対照を明示。ただし**両JSONはAI草案のPressure/Expectedであり、TEI Parserが読んだAccepted Sourceでも製品Graph/CRUDの実出力でもない**。

- **Semantic Definition/Subject**：例 `data.order`（受注というData *型・定義*）、`rel.order.lines`（Relation定義）、`fn.order.register`（Function定義）。安定したSubject ID/Source Position/Revisionを共有する。表示ラベル「状況」が同じでも`data.order.status`と`data.flow.status`は別Subject。
- **Representation Occurrence**：`canvas.sales-overview`の`view.sales.order`と`canvas.fulfillment`の`view.fulfillment.order`は**同じ`data.order`への二つの配置**。同様にRelation `rel.order.lines`にも`view.sales.edge.order-lines`・`view.fulfillment.edge.order-lines`という別の表示辺がある。表示位置、選択、展開・折畳み、レイヤー/Canvas所属はPresentation状態であり、Semantic Subjectの複製・削除・変更を意味しない。
- **業務データInstance/実対象**：受注O1や明細L1の実Row/Key、Tenant/Scope/Versionは、`data.order`という**モデル型**そのものではない。P0-Mの選択した設計ノード/Function×Data CRUDセルから、具体的受注O1へのAction Target/Host Grantを直接生成しない。P0-CではFunction Data Context→Query/Result Slice→Row/Occurrenceへ**別の束縛Witness**が必要（§14/15、#2586/#2506/#2598/#1174）。

この三つはIdentityだけでなく**失効粒度**も別。共通Semantic Definitionの意味・Relation Rule改訂は双方のCanvas由来Projection/依存Buildへ影響し得るが、一つのCanvasで座標・折畳みだけを変えた場合、他CanvasやSource Meaningの受理を黙って失効させない。表示上の削除は特定Occurrenceの削除であり、Data/Relation/業務イベントの削除と推定しない（旧#3222のMV、現在は#3303の合成写像Owner）。

### 16.2 同じ合成Fixtureからだけ派生させるData/Function/Business反例

| 対照（#3263 Input/Expected・#3303既存ケースへ判別列を追加） | 独立に確認する意味 | 禁止する誤保証 |
| --- | --- | --- |
| **二Canvas共有**：`data.order`が二つの`view.*.order`へ表示される | 受注定義は1 Subject。二つのOccurrence ID、Canvas所属、Selection/Source版は識別できる | 2表示＝2業務Entity／2受注Row、片方の配置削除＝元受注定義の削除 |
| **Relationの二重出現**：`rel.order.lines`が二つのCanvasに投影 | 同じRelation定義を異なる表示辺が参照。表示辺とSemantic EdgeのIDを保持 | 片方の表示辺の非表示＝Semantic Relation不存在／Edge Rule変更 |
| **同名・異ID**：「状況」の2 Subject、同端点を持つ`rel.line.sku.primary/alt` | ラベル・端点が同じでもSource IDとRole/Edgeが違う | 表示ラベル/端点キーだけで統合、並列Edgeを一つへ縮約 |
| **Context境界**：同じSubjectが別Canvas/Functionの別Scope・Disclosureで見える | 表示利用者の現在Scope/Selection epochに応じ、開示・Action候補を別々に再評価 | 片方で表示可・操作可→他方で同じGrantがある |
| **CRUD列の意味**：`fn.order.register×data.order=C` | FunctionからData *定義*への**宣言操作**であり、個別O1の実更新Receiptではない | Canvas出現数×CRUD、`C`セル＝O1作成済み、`R`セル＝Providerの実Read成功 |
| **未完備Scope**：`fn.external.dispatch`の`declaredCrudComplete=false` | 未列挙のセルは`unknown-not-enumerated`で、`none`にしてよいのは列挙範囲がcompleteな場合のみ | 表に空欄→実作用がない、外部配送にGrantやEffectが無いと確定 |
| **手作業Step**：`action.phone.confirm`はFunction無しで手順に存在 | Business Actionの説明・手作業と自動Function/Commandを独立に表示 | Function未接続→手順不存在、Step＝実自動発火Event |
| **編集の逆写像**：Canvas Aでカードを移動、同じSubjectを表すCanvas Bはそのまま | Presentation Occurrenceだけの変更と、Role/Relation等のSourceパッチ・両Canvas再Projectionを独立判定 | ドラッグやSelection同期をDomain Mutation・Business Approval/Effectへ変換 |

**Itemの分類**についても将来の汎用Canvas/Documentへ接続する際、`Task/Action/Event`を現在の`Item`と排他的なCore型の列挙として先決しない。情報断片に付く**Role/Traitの候補**と、実発生Event・実行Attempt・実業務Actionの**別Subject/履歴/Receipt**を混同しない。詳細設計のOwnerは#2495/#2497/#2620、Function/Patternは#2507、実Action/Receiptは#2598/#1174。これは正式採用済みのType/Role Registryという主張ではない。

### 16.3 Source・CLI Tool・生成AIテストの現在資格と失効

**Source/Consumerの現実**：P0-Mの合成JSONをProduct Sourceと見なさず、まずTEIが実読したSourceの`Workspace Identity/Manifest Source`・`worldFingerprint`・Model Revision・Source Span・Semantic Subject/Relation ID・Read Completenessを一致させる。[main #3307](https://github.com/hat47x/tei-platform/pull/3307)は`tei commands`に引数集合/型/既定値を追加し、実CLI受理値の局所試験を固定した。ただし**Diagram/CRUD専用Operationの公表／実Consumer到達を実装したものではない**。[PR #3311](https://github.com/hat47x/tei-platform/pull/3311)は`workspace`名・Manifest版/照合値のCLI応答追加の候補だが現時点**未統合**。二つのWorkspaceで同じModel名・同じByte/World FPでも、同じ業務Contextと断定できない。#2822/#2824/#3262が公開・同一性の既存Ownerを保つ。

**テスト能力の最新上限**：[main #3296](https://github.com/hat47x/tei-platform/pull/3296)の9 CLI Reply・16合成負例は`exploration-only / qualifiedTestGeneration=false`のSource/Tool Responseを機械的に切り分ける限定資格。[main #3304](https://github.com/hat47x/tei-platform/pull/3304)で行番号0の**既知**Rule Candidateを生成済Pinned CLIで実行し、偽Expected/偽Fixture/古いPinの3負例まで区別できるようになったが、AIによる**未知のMulti-Canvas/CRUDの不変条件発見（T1-B）**は未検証。#3263のExpectedが別ファイルであることだけで独立Meaning保証は成立しない。#2514/#1180/#3273の同一Scenario/Expected/Mutation体制に、上表の最小反例を差分として取り込む。

**図面・テスト・実データの失効を一緒にしない**：Source Subjectの意味変更なら対応するP0-M/Binding/Consumerを再資格化、CanvasだけのLayout変更なら対応Occurrence/Renderer差分だけを評価。Function Query/SQL/Provider/RowVersion変更ならP0-R/Business Claimを独立再実証し、P0-Mの内容が同じでも古い実Readを使い回さない。Plugin/Target/BOM/Operator版の変更はSelected実装/実利用・Conformance Witnessを再点検し、単に旧Artifactが存在するだけで適格にしない。これらは既存#1163/#2582/#2600/#3262/#3297と[main #3298](https://github.com/hat47x/tei-platform/pull/3298)のTrait/Provider実利用の責務に委ねる。

### 16.4 既存Ownerの実装順と停止線

1. **P0-M（#1124/#3263、統合候補#3303）**：まず実TEI Sourceを読んだHeadless Data/Function/Business図＋CRUD表、Source SpanへのRead-only逆参照。Canvas2/Occurrence9、同名2 Subject、同端点多重辺、`unknown`と`none`の差を**既存合成Pressureと独立期待値**で検査する。最初に重要なのは製品Parser→Graph/CRUDの実出力であり、Canvas描画自体は後段。合成Fixture用Go auditのPASSだけで製品投影の受理としない。
2. **P0-R（#2586/#2506）**：一Function/一Query/一実ProviderのProduct Rowset/Selected・Predicate・Scope・Timeを実観測。参照PGの限定成功と製品選択SQL/Providerの成功を区別する（§14）。
3. **P0-C（#2497/#1163）**：P0-Mの**Model型/Function-Data定義参照**とP0-Rの**実行時Entity Instance/Rowset**を同じSource/Workspace/Revision/Function Bindingで照合。照合可能なものだけ連結し、未接続や同名別Contextは未解決として返す。
4. **編集/Consumer/Effect（#1124/#997/#2500/#2598/#1174）**：対象OccurrenceのLayout/Selection状態、可逆なSource編集、Actual RowのBusiness Commandを別手段で検証。SHO/Studio/Webの画面試験・現在Grant・Receipt・実資源読取・Business AcceptanceはP0-M/R単体の必要条件にしない。

**担当境界**：三図/CRUDのFixtureやCanvas2/MV/RP/CXの正本は#3263/#3303、汎用Role/Trait/Itemの正式設計は#2495/#2620、実Provider R1とContextは#2586/#2506、独立期待値/Mutationは#2514/#1180、実Target/Grant/Effectは#1174が所有。本計画は受入横断の**誤帰属と失効の照合条件のみ**。**現時点の証拠は既存合成JSON・Go Fixture候補・限定CLI/PG/CIと本設計計画であり、今回新たな製品三図/CRUD/複数Canvas実GUI/製品Function→Provider→Business実Effectを試験していない。正式なOwner Meaning受理もNOT RUN**。第二Scenario SSOT、Canvas Meaning Store、Graph Editor、Policy Engine、Hostは作らない。

