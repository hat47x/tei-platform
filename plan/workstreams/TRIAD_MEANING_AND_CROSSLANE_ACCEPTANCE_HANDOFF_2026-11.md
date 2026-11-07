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
## 12. 11/7横断追補：Receipt再照会を「宣言したData Effectの保存後状態確認」に昇格させない（既存証拠の意味修正案）

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
