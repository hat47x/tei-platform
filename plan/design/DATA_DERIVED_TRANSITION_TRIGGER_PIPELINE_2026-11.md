# Data Modelの仮想項目からFlow／Batch／Pipelineを起動する意味契約（設計案）

- 日付：2026-11-03
- 状態：**設計提案／反証計画**。新しいCore kind、Data Script構文、Event Broker、Scheduler、Job Engine、Execution Hostの採用や実装済みを意味しない。
- 発端：Data Script上の通常項目の更新や仮想項目の導出状態の変化から、バッチ／パイプライン／大規模Jobを起動したいという要求。状態列は他の属性・関連データの状況を代表して二重管理されやすいため、**意味の正本をもつ事実から状態を導出**し、必要なEffectだけを別の履歴／Receiptへ保存する。
- 関連：[Buriからの継承](BURI_TO_TEI_LIFECYCLE_TRANSFER_2026-11.md)、[非業務State Machine・三形態](STATE_MACHINE_CROSS_DOMAIN_DESIGN_PRESSURE_2026-11.md)、[Decision Workbook](DECISION_WORKBOOK_INTAKE_REVIEW_2026-11.md)。
- Issue Owner：[#2622](https://github.com/hat47x/tei-platform/issues/2622) Data/Derived、[#1129](https://github.com/hat47x/tei-platform/issues/1129) Rule Phase/Effect、[#1130](https://github.com/hat47x/tei-platform/issues/1130) Non-UI Batch、[#2582](https://github.com/hat47x/tei-platform/issues/2582) Dependency/Refresh、[#1708](https://github.com/hat47x/tei-platform/issues/1708) joint Read、[#1174](https://github.com/hat47x/tei-platform/issues/1174) Effect Fence、[#2564](https://github.com/hat47x/tei-platform/issues/2564) actual CAS、[#2514](https://github.com/hat47x/tei-platform/issues/2514) Independent Oracle、[#2600](https://github.com/hat47x/tei-platform/issues/2600) Build/Invalidation。新Issueを作らない。

## 1. 三つの責務と三種類の「状態」を混同しない

### 1.1 Data Modelの仮想項目（意味宣言）

**Data Modelが、属性を「導出される仮想属性」として明示し、型、対象、書込み可否、基礎データの所有者を定義する**ことを要件とする。同じData ID/Attribute IDを他のField、UI、Rule、Flowから参照できる。

たとえば `order.billing_ready` は `boolean / virtual / read-only` としてData Modelに宣言する。`order.status` を物理列に必ず追加する設計ではない。仮想属性が物理的に未保存でも、UI/Query/Rule/Flowはその同じ意味Attributeを参照できるようにする。Data Modelで「仮想」だと宣言されていない項目を、Scriptに関数があるというだけで暗黙に仮想化しない。

**まだ正式なGo Struct／DSL grammarは採用しない**。属性の型、Virtual／Derived Role、可観測なScope、保存方式（read-time/on-write/materialized/generated DB）を一つのCore enumに押し込まない。意味上の`virtual`と物理実現の`computed in DB`／`cached`／`generated column`を分離する。

### 1.2 Data Scriptが導出規則と依存を定義する

Data Script（名称・Syntaxは未確定）は、`order.billing_ready`の**純粋な導出規則**を定義する責務を持つ。入力はData ModelのAttribute/Relation IDと明示Context、必要なGrain、Filter、欠損/NULL、取得完全性、評価時点、Query/Operator/Plugin版。例は配送明細すべて完了／受注承認済み／請求レコード未生成などの事実から導出する。

- `all(lines, delivered)`には`lines`の**完全集合**の根拠が必要。`0行`と`未取得`、部分Read、権限により非開示の行を区別する。`all(empty)`の真偽はそのDomainの受理済み規則に従い、勝手に空集合を請求可能へしない。
- `NOT EXISTS invoice`は「取得できない」「Providerが完全性を証明できない」と同義でない。Queryの非出力Predicate、集約、JOIN、Scope、Timeを依存閉包へ含める。
- 状態を変更する目的で`billing_ready`へのwriteを許すと、元の事実との二重管理が再発する。仮想Attributeへの明示writeは原則unsupportedとし、受理された**元データへのOperation**へ誘導する。手動上書きが業務要件なら「明示Overrideという新しい元データ」をモデル化し、導出式の入力とし履歴化する。
- 純粋導出と、write前Validation、before-write mutation、同一Tx Effect、post-commit外部Effectは#1129の既存Phaseで分離する。Data Script内の計算が外部Jobを暗黙起動することは禁止する。

### 1.2.1 Data Script上の「項目変更→Flow」Bindingを第一級の設計入力とする

**ユーザー要求として、Data Scriptの中で、通常Field／仮想Attributeの変化を契機にする宣言を置ける構成を検討する。** これはData ScriptがJobやQueueを直接操作する方式ではない。例えば`billing_ready`の導出式と、その`false→true`変化を`invoice_pipeline`へつなぐ**Trigger参照Binding**を近接記述できるようにする。

- **Data Model**：仮想項目の宣言・型・対象Data Identity・read-only（例：`order.billing_ready`）。
- **Data Script**：導出規則、その依存・時点・取得完全性、さらに変更時に起動すべきFlow ID／Event Contractを**宣言として関連付ける**。Source位置とstable Attribute/Rule/Trigger IDを保持する。
- **Flow Definition**：起動契機のedge/level等の意味、前提・受理済Rule／必要なData/Job Target・失敗／再実行Policyを保持。Data ScriptにFlowの全Stepを重複定義しない。
- **Execution Binding**：Changed Event/Job Intentを、そのHost/Providerで実現できるかを検証。実行権限、現在のData/Grant、Durable effect/Receiptはこの段階の責務。

UIで当該項目を選んだ際、**Data Modelの宣言→Data Scriptの式とTrigger参照→Flow図／状態表→Batchジョブ**が相互に辿れ、いずれの箇所から修正しても同じReference IDへ差分候補を提示できることを受入項目にする。これならデータ中心・フロー中心の双方から機能を設計できるが、三つの宣言をそれぞれ「業務状態の別正本」とするわけではない。

### 1.2.2 現行Semantic実装との距離（静的照合）

[現行main `internal/semantic/model.go`](https://github.com/hat47x/tei-platform/blob/main/internal/semantic/model.go)の`Data`には`Script string`フィールドが既にある。一方、`Attribute`には`ID/Symbol/TypeRef/Required/Immutable/Role/Value`等はあるが、**専用の`virtual`判別子／Derived Rule ID／Dependency集合／Trigger Bindingを持つ正式な契約はない**。したがって、「DataにScript欄がある」ことと「仮想属性の型付き導出・変化イベント・Batch起動が実装されている」ことを混同しない。

この差分を埋める最短案は、既存Data／Attributeの型付き意味IDとRoleの解釈境界、Data Script Source参照、既存`derivationref`・Data Rule/Effect Phaseを用いた**非破壊のReference**から開始し、正式な`virtual` Syntax/Coreフィールドを採るべきかは具体的二Domainの読取・書込・再資格化反例で判断すること。物理保存に`Generated Column`／Cacheを使うかはProvider Realizationに属する。

### 1.2.3 ValidationとDerivationの配置規約（X-TEAからの継承方針／2026-11-03）

ユーザーの追加方針に従い、**Data Modelで表現できる項目属性・拡張定義はData Modelへ、そこから外れる独自ValidationはData Scriptへ**置く。**Derivationは主にDataの更新契機でData Scriptが評価する**。これは今後のTEIの設計方針であって、旧X-TEAの各イベントコードやフックをそのまま移植する確定案ではない。

| 対象／契約 | 記述場所（Authoring） | 何を表すか | 正本・実行との境界 |
| --- | --- | --- | --- |
| 基礎的Validation | **Data ModelのAttribute拡張定義** | 型、必須、許容範囲、長さ、集合、形式など**その属性に局所的な宣言で記述できる規則**。型依存で実装・受理済みのものだけ | モデル上の宣言が業務規則の意味元。現在実装の`required`等と、未実装`min/max`等を区別。DB CHECKと同じ保証ではない |
| 関係・横断・独自Validation | **Data ScriptのValidation Rule** | 条件付き必須、他項目との比較、複数行の重複、親子件数・状態、時点付き規則、専門Domainの判定など | 型付きID・依存Read・Context／Snapshot／Provider完全性を要求。独自ロジックでも`true`を返しただけで実行認可にしない |
| Derivation | **Data Scriptの純粋Derivation Rule** | Data Modelが`virtual/derived`を宣言したAttributeへの計算、他Attribute／Relationからの値・状態の導出 | 主な評価契機はinsert/update/delete、Relation link/unlink等の**実Data変化**。Scriptが同じ項目に勝手に別正本を生成しない |
| Trigger／Flow | **Data ScriptのTrigger参照宣言＋Flowの実行契約** | データ項目更新や導出値のedge/levelの成立をJob/Capabilityへ接続 | 導出やValidationの評価はEffectなし。Jobの発火／Commit／認可は既存#1129/#1130/#1174の契約 |

**配置の単純さと実行の正確さを両立**：
- **Model→Scriptへの明示参照**：同じ規則をData Model拡張とScriptに重複記述しない。Data Modelに属する制約をScriptで独自実装する必要がある場合、ScriptのRule IDを項目へ参照Bindingし、表示上は項目拡張の一部として見せられるかを評価する。型・対象・Rule/Operator版・Source spanを追う。
- **派生値はData Modelで宣言、Data Scriptで定義**：型と`virtual`指定だけでは値を導出できない。Script側のTarget Attribute IDと依存閉包が特定できない場合は`unresolved`。物理列を不要にすることと、検索性能や障害回復のためのMaterialized Cache／実Job Receiptの保存は両立する。
- **Data更新イベントは更新のIntentではない**：`field edited`／`pending update`／`provider committed`／`replica observed`を区別。Validationは**書込み拒否に必要な段階**で評価し、純粋Derivationは依存順に評価する。Flow/外部Jobの起動は原則Commit確認後のIntentへ結び、before-write評価中に外部効果を起こさない。
- **同じ導出に複数用途**：保存前の整合判定、read-time表示、変更時キャッシュ失効、commit後のTrigger比較に、同じ純粋Ruleを適用してよい。ただしread-timeだけで観測した値から厳密な「変化が発生した」というEventを推測しない。強い一貫性が必要ならProviderの同一Snapshot/Tx/Conditional Write能力を検証する。
- **更新集合の局所性**：単一Field更新のほか、子集合の追加・削除、Lookup値の改版、時計の経過で派生値が変わり得る。Source更新時の評価対象とRule Dependencyの逆引きが不明なら、局所再計算／確定Eventを主張しない。周期再調停やread-time再評価が必要なProfileもある。
- **業務判定と技術失敗**：Validationは`satisfied`／`not-satisfied`／`unknown`／`error`を区別する候補契約。Script例外、partial read、Providerの権限不足を「Validation不成立」と同一視しない。X-TEAのhistoricalなcheck-only時にScript副作用が抑止されない懸念は、TEIでは純粋判定の実行フェンスで切り離す。

#### 一つの更新からの処理順（提案でありAPI／Runtime実装済みを意味しない）

~~~text
変更要求 / Operation Intent（Data ID・対象Record Key・期待Revision）
    ↓
対象の現在Read／必要なRelation・入力Scope／権限の確認
    ↓
Data Modelの項目制約 → Data Scriptの独自Validation（純粋）
    ↓
Data ScriptのDerivation（純粋・依存順、未確定なら停止）
    ↓
必要なValidation/Derivation再評価・予測差分と独立Expected
    ↓
Owner Action / Providerの条件付きWrite・Transaction
    ↓
Commit outcome（success / rejected / unknown）とReceipt
    ↓
Data Script Triggerの前後比較（確定した対象・時点に限定）
    ↓
Flow / Batchの起動Intent → Host/Provider認可 → Jobの実Effect/Receipt
~~~

上の順は**意味上の関門**。書込み前後で同じ入力Snapshotが必要か、導出値をWrite Setへ含めるか、post-commitに再取得するかはProvider能力とRule依存により異なり、実装の固定フック順ではない。特に、`Commit unknown`で成功イベントを捏造しないこと、必要なConstraintをpost-commitに遅らせて誤書込みを許さないことを必須とする。

**最小負例の拡張（既存#1129／#2514に相乗り、試験番号の新設なし）**：`quantity=-1`は項目制約で拒否、`quantity=4 / available=2`はData Scriptの独自Validationで拒否、`quantity=2 / price=100`でDerived amount=200、子明細の一部未取得ならDerived全件完了はunknown、同じ値に更新してもDerived-edgeは発生しない、Commit rollbackならJob発火しない、同じCommitの再通知なら重複Jobを防ぐ。これらのExpectedはRuleの式をコピーせず独立根拠で固定する。

**判断待ち**：項目拡張の具体的な最小セット（required/length/range等）、Data Scriptに置くPredicate/FunctionのDSL表面、評価フェーズと同じRuleの実行可能Target範囲、Relation集合・非同期Providerの整合要件、Scriptのトリガー宣言とFlow定義の参照方向。既存#2620/#2622/#1129/#1130/#988/#2514に割り当て、現行Coreへ無条件に追加しない。

### 1.3 Flowは「評価された状態・変化」と「副作用Intent」を結ぶ

Flowには`when physical attribute changed`、`when derived value transitions`、`when state condition holds`、`when domain event committed`等の契機の**意味上の違い**を明示できることを要件とする。これらは仮の記法であって正式Grammarではない。

| 契機 | 意味 | 追加の証拠・制約 |
| --- | --- | --- |
| Source-write Event | 基礎Field/Relationの変更が確定した | どのWriteが、どのSubject・Txnに確定したか。Derivedの値が変わった証拠にはならない |
| Derived-edge | 同じSubjectの計算値が、承認されたbefore/afterの観測境界で`false→true`等へ変化 | 比較するBefore/Afterの完全性・整合Snapshot、規則版、Event Identity、再検出/重複抑止。毎回の再計算成功＝edgeではない |
| Level predicate | `billing_ready=true`等が一定Scope/時点で成立する | 初回・再観測・定期再走査でも成立し得る。ジョブ重複抑止・期限・現在Authが別途必要。Edgeとは異なる |
| Accepted domain event | `order.confirmed`等、Ownerが意味受理したEventが確定した | 対象・Version・Receipt／Commit・発行契約。見えた状態やUI ACKから自動生成しない |
| Timer / external event | 時間の経過、Webhook、Queue、センサなど | 外部Clock／送信者Authority／重複・遅着・配送順の契約。データ更新がないことを無変化とみなさない |

**データ導出状態をイベント正本へ格上げしない**。Derived claimは同じ入力・Operator版・適切なRead上では計算できる一方、`T1`時点で`false`だったという履歴、`T2`で外部通知したという事実、Job Stepが完了したというReceiptは別の永続事実。必要ならOutbox／Job journal／Outcomeは**実行記録**として保存するが、同じ業務状態を別のTruthとして保存しない。

## 2. 図・表・Scriptを結ぶ設計Surface

**同じModel IDを起点に、次を連結**できることをTEI Studio/SHOの設計圧力とする。

| Surface | 設計内容／編集責務 | 別の意味正本にしないもの |
| --- | --- | --- |
| Data Model Table | `order.billing_ready`: type bool、`derived/virtual`、read-only、対象Data、説明とOwner | Data Script式の複製、DBにstatus列を強制 |
| Data Script／Derivation Table | 依存Field/Relation、演算、集約集合の完全性、missing/null、Time、純粋計算 | Script hookの内部でJobを送る |
| Flow／Trigger Table | 対象Attribute/Event、edge/level/write/commitの違い、Guard、Action/Capability参照、Re-trigger/replay policy | 表を描くだけでEffect許可になること |
| Pipeline／Job Graph | Task ID、入出力Artifact、依存、順序/並行、分割・checkpoint、容量・取消/回復、Ownerと実行場所 | 全作業の基盤に一律の永続Workflow Instanceを要求 |
| Test／Scenario／Trace Matrix | Source/Before/After、Signal、Expected Event/Job Intent、実Observation/Receipt、失効とRevision | Derived Expectedを独立Oracleと称する |
| SHO設計資料 | これらの意味／未決事項を読者向けに相互参照・レビュー／版管理 | 承認済みドキュメントと意味受理を混同 |

**例（概念・正式TEI DSLではない）**：
~~~text
Data Model
  order.billing_ready : bool [virtual, read-only]

Data Script
  billing_ready <- approved(order)
                   AND all(order.lines, delivery_confirmed)
                   AND not_exists(order.invoices)

Flow
  when order.billing_ready changes false -> true [after commit, scoped by order ID]
    request Capability invoice.create_for_order

Batch / Pipeline
  invoice.create_for_order -> validate current facts -> create invoice intent
                           -> provider effect -> receipt -> independent read
~~~

これは「同じ意味をModel、Script、Flowに三重保存する」ということではない。Modelは対象の宣言、Scriptは評価規則、Flowは変化を利用するTrigger/Action Bindingをそれぞれ**参照で接続**する。`delivery_confirmed`と`approved`の業務意味はOwner決定、`invoice.create_for_order`のTx/冪等性はProvider/既存Hostの実行責務。Script自体にJob実行権限はない。

## 3. 導出値の変化を実Eventへ変換するための証拠境界

**安易な`onChange(virtual_status)`は危険。** 無保存の純粋式は、値を再計算できることを示すだけで、過去に何度・いつ変化したかを示さない。以下を区別する。

1. **入力の変化を捕捉**：Committed source mutation、CDC/Queue通知、明示Operation/Receipt、またはPeriodic reconciliationのどれか。Providerによっては変化通知不可。通知の有無から値の不変を推定しない。
2. **対象Dependency/Subjectを特定**：更新された子行がどの`order`に影響するか、Join/NOT EXISTS/全件集約、Rule/Plugin版、Source移動、外部Provider更新、Clock/timerを含む依存範囲を評価。依存不明なら局所最適化できない。
3. **一貫した前後観測**：Before/Afterが同じ論理Subject・Scope・許可されたRule版で比較できるか。#1708のsame-record/snapshot/version-aligned/best-effort能力に照合し、単に近い時刻・同じ数字では証明しない。比較に必要な旧値がない場合、Edgeは**unknown**。初回評価を勝手に`false→true`としない。
4. **Trigger result**：`edge-known`／`no-edge`／`unknown`／`unsupported`の区別を検討。Edge発生が確定しても、当該EventからJobが起動できる資格は別。外部状態がstale/unavailable/partialなら「発火せず保留」やrecheckへ進み、false扱いしない。
5. **Effect fenceと回復**：対応する`Capability`の最新Owner／Resource権限を実行時確認。DBで原子的な`source update + outbox`を提供できれば採用候補だが、複数Provider跨ぎには同一Tx保証なし。`Event ID→Job ID/Attempt ID→Step Effects/Receipt`をつなぎ、at-least-once送信時は重複を検出。Exactly-onceの一般保証はしない。
6. **定期再調停**：イベント欠落、DB/APIの遅延、Crash/ACK loss、A→B→A（途中の遷移が端点から消える）、時計の進行による条件成立を別ケースにする。厳密Edge要件を満たさないProviderは`level-triggered reconciliation`＋一意業務Action/既存Receipt等の方式を比較。ただし`level`が`edge`の意味的な代替になれるかはOwner判断。

**バッチの実務上の複雑さ**：数十万対象のJobでは「statusが変わった」だけで全件走査すると重複・取りこぼし・過負荷を招き得る。対象粒度（Subject/Subset/Window）、処理対象の基準Snapshot/Watermark、順序／並列度、処理再開点、Step結果、Backpressure・Rate limit、再送／取消／部分成功、失敗の隔離、再実行した場合のEffect一致をJob/Provider契約へ渡す。これらは`#1130`の既存Batch Ownerが担当するものであり、汎用Task Engineの実装を本稿で確定しない。

**再帰と循環**：FlowのActionが`billing_ready`の入力`invoices`を変更して`true→false`へ戻し、次の再計算が走ることは起こり得る。これは正常なFeedbackの場合もあり、無限起動ループの場合もある。Dependency DAGに加えてEffect→Inputの関係を表示し、業務上の終了条件、再入禁止／最大試行、冪等性を明示する。`false→true`の再入は**新しい業務Eventか重複か**を、単に値が同じという理由で決めない。

## 4. 仮想状態は二重管理を減らすが、全てを仮想化しない

| 状態の例 | 正本と望ましい扱い | 仮想で代替できないもの |
| --- | --- | --- |
| `ready_to_invoice` | 承認・納品・請求不存在の事実から**計算可能な状態**。仮想属性が候補 | 元データが外部で完全に得られない場合の「現在確定」 |
| `all_lines_delivered` | 受注明細・納品のRelationsと完全性から計算できる | 子集合の一部だけ見た集計、非開示行、正本のない履歴 |
| `currently_overdue` | 期限・現在時刻から計算可能。Clockを宣言した導出 | 時刻だけで閾値を越える場合の発火にはScheduler/Time Eventが必要 |
| `job_running`／`job_step_done` | 実Jobのjournal/Attempt/Checkpoint／Receiptに根拠を持つ**実行状態** | 外部Side Effectの成功や送信済みを業務データから推測すること |
| `notice_sent` | 送信要求と実Delivery Receiptが区別された保存済み事実 | 未確認のメール・Queue配信を`sent`へ自動昇格 |
| `order_cancelled` | Ownerが決める業務取消の独立した意思決定・履歴が必要なら**保存された明示事実** | 他の値が条件を満たしただけで取消の意思が発生したとは言えない |

パフォーマンスのために物理側へ保存する`materialized billing_ready`は、規則版、入力Snapshot/Read completeness、再計算時点と無効化条件をもつ**キャッシュ／派生Projection**であり、新しいAccepted Meaningの正本ではない。Materializedを読むConsumerが`fresh`と認められる条件を別に宣言する。分散環境では「読み込みたび最新」を要件とするかを先にOwnerが判断する。

## 5. 最小のConformanceと独立Expected（すべてNOT RUN）

| 既存Owner | 反例／検査Fixture | 最低限許される結論／禁止推論 |
| --- | --- | --- |
| #2622/#1129 | Modelにvirtual宣言、Scriptに純粋導出。Script出力へ直接writeしようとする | 不正writeを拒否。Hookが登録できただけで実Computed Column採用とは言わない |
| #1708/#2582 | 受注明細3件中2件だけ取得・未着1件を隠す。集計`all delivered`と`NOT EXISTS invoice` | Result`unknown`、**false/true・emptyにしない**。影響範囲不明の局所skip不可 |
| #1129/#1174 | 行更新がRollback、ACK消失、同一Event重複、commit後にGrant撤回 | rollbackからJob確定を推測しない。Unknownを成功にしない。現在GrantでEffectを再検査 |
| #2582/#2600 | A→B→A、ルール版更新、Script source移動、Time threshold超過 | 終点同値でもEdge継続保証なし。Source/Meaning/Artifact/Triggerの失効を別々に表示 |
| #1130 | 10万件対象の分割、途中中断・再開・取消・並列更新、同一要求再送 | Job Attempt/Checkpoint/Receiptを観測。重複Effectを合格にしない。throughputは測定後 |
| #2514/#1131 | 同じSource/Ruleから生成した期待と独立手書きExpectedが異なる | 生成ロジック自己採点を独立Oracleと呼ばない。誤Guard・漏れEventをMutationで検知 |
| #2497/#2621 | Model/Script/Flow図表の1箇所だけ変更し、他Projection/SHOとの差分 | Source/ID/Versionが同じ範囲のみ整合と結論。図に見えるだけで実行資格を付与しない |
| #2564/#1174 | 2 Workerが同じ`ready=true`に対し同時請求Jobを起動 | 条件付きWrite/一意Operation IDと実DB/Providerの防止を確認するまで本番安全性未取得 |

最小縦断を一つに絞るなら、**受注の一部納品→全明細納品→請求準備完了のDerived Attribute→Batch起動Intent→同じ請求の二重作成防止→Receipt/別Read**。受注状態列を手で更新して二重管理する案と、Derived＋業務Factを使う案を比較し、規則修正件数、同期更新箇所、誤検出、性能、復旧難度を計測する。どちらが実際に有利かは試験で判断する。

## 6. 設計上の段階とPlan Ownership

- **F0（今回）**：Model virtual＋Data Script derivation＋Flow trigger＋Batch Actionの責務分離と、既存Issueへの引渡しを文書化。**採用済みのDSL syntaxやEngineを作ったと主張しない**。
- **F1（read-only）**：既存Derived/Rule ReferenceからModel/Script/FlowのLinkとDependencyをExplain可能にし、前後値・未知・完全性・型を独立Expectedで検証。既存Canonical/IRの新規正本を増やさない。
- **F2（Trigger Intent）**：単一ProviderでのCommitted Source Update／Virtual Edgeの比較可能性、outbox/claim correlation、重複・Rollback・A→B→AとClock変化を実Provider要件へ分離。Event Broker構築を先行しない。
- **F3（Batch E2E）**：#1130のUIなしJob/進捗/再試行/大量対象に対象ID/Chunk/Checkpoint、#1129/#1174のEffect／Receiptを接続。実ホスト/Providerの一度限りの実Effectを独立に観測する。
- **F4（図表・設計・生成）**：同一Accepted MeaningのData Model/Derivation/Flow/Job DAG/SHOへ編集・Source逆写像・選択的再生成。Runtime安全性とAuthoring価値を別軸で検収。
- **採用Gate**：仮想Fieldで減らせた物理status二重管理と、増えた導出コスト／Watch/Job記録/Recoveryの複雑さを比較。Domain全体で統一Status表・共通Scheduler・Workflow Engineを採ることを前提にしない。

**既存Issue責務**：意味属性・Relation/Context=#2622/#2620/#2506、Rule/仮想/phase=#1129、Dependency・Read Freshness=#2582/#1708/#1641、Query集合完全性=#2387、Action Binding=#2507、Batch=#1130、Effect/Grant/Receipt=#1174/#2564、Diagram↔Table/Source=#2497/#2621/#1803、独立Expected=#2514、Build差分=#2600、全体Gate=#1119。新しいJob Registry/TEI Host/第二Meaning正本を作らない。

## 7. X-TEA既存機構の小さな移植を先行させる（2026-11-03、優先順位の修正）

**ユーザー指摘**：Data Modelの項目拡張Validation、Data Script側の独自Validation／Derivation、データ変更時のScript評価・状態連動はX-TEAに既にある。TEIでは、これらのために新しい抽象アーキテクチャを作るのではなく、**既存実装の構造と業務意味を保持して移植すること**を第一候補にする。前節までのTrigger／Job／Edgeの一般論は、すべての移植Caseに強制する事前要件ではない。

ただし「X-TEAで実装された」「Source上にhookがある」「TEIでも同じ結果を実行できる」は別の証拠段階。既存[#950](https://github.com/hat47x/tei-platform/issues/950)（RMT-2 Source-faithful adapter）には、実Benchmarkで**DERIVABLE Field 52件**を分類した所見がある。[#1637](https://github.com/hat47x/tei-platform/issues/1637)（closed）はX-TEAのTable Script／Virtual Fieldの**read-time導出**を、DerivationDefinitionとBindingへ分けた合成Reference。一方[#940](https://github.com/hat47x/tei-platform/issues/940)（open）は旧Driverの実Runtime Effect/Transaction観測を扱う。これらを合算して製品移植完了と判定しない。

### 7.1 移植可能性を三種類に切り分ける

| X-TEAの既存機構と想定入力 | TEI側の受け皿 | 移植クラス／確認すべき差分 |
| --- | --- | --- |
| Data ModelのField定義・属性拡張（型、必須、DERIVABLE等） | 既存Data/Attribute、Validation意味Role、Source-faithful adapter | **P：意味・構造をほぼ直接対応付ける候補**。X-TEAのField拡張がTEI現行Coreにない場合は、既存属性に紛れ込ませずRole/Extension候補として保持。実構文採用は別 |
| Data Scriptの独自Validation | 既存#1129のRule Definition/Attachment、#1637／現行Validation Eval Reference | **P：既存hookの判定ロジックを移植候補へ分類**。Scriptでread/write/APIが起きる部分は判定から分離し、`check-only`でもEffectがないと保証されるかを負例で確認 |
| Data Scriptによる仮想Field導出 | Data Modelのvirtual対象ID ＋ #1637のDerivationDefinition/Binding | **P：同じInput/Outputを移植する候補**。Dependency、NULL、時点、Relation全件性、Operator版を失わない。現TEIに未実装Operatorがあれば外部Script Asset Binding/unsupportedで止める |
| BR／AR等のデータ更新イベント、`oldValue`／`valueChanged`を使う条件 | #1129のwrite lifecycle phase＋Rule Attachment、#2582の依存/再取得 | **A：既存Script hook→TEI Operation phaseへのAdapter移植**。旧Driverの順序・Tx・commit前後・例外・再入抑止・依存更新先をSource観察と独立Expectedで確かめる |
| Script連鎖での関連レコード更新／状態の代表値再計算 | #1129 Action/Effect、#2622 Derived、Provider Binding | **A：同一Provider内の意味は継承候補**だが、旧Driverの`true`返却や見かけの状態値だけでCommit確定・原子性を推測しない。元データとDerived表示／保存結果を分ける |
| 更新後の別プロセスJob／大規模Pipeline／Queue・再開・高並列実行 | #1130のBatch/Non-UI、#1174 Effect Fence、#2564 実DB CAS/Receipt | **E：X-TEAのScriptで呼び出し得ることと、独立Runtime保証は別**。UIなし起動・Durable Event・多Worker・部分失敗回復が必要なときだけTEI追加拡張として評価 |

P=まず意味忠実な移植、A=既存契約への適合Adapter、E=追加要件の実装／独立受入が必要、という**計画上の分類**。ライブラリの機械的コピー・互換API・公式移植完了を意味しない。X-TEA上で読取時算定と更新時Script実行が併存していた可能性も残し、「Derivationは主に更新契機」を唯一の実行時点へ固定しない。

### 7.2 「単純移植」縦切りFixtureの優先順

1. **属性拡張Validation**：XEAF/XEADの型・必須・値域候補を原Source/ID/版つきで読み、同じ値を良否判定する最小ケースを作る。現行TEIの`positive`／`relation-unique`で表せない範囲を**不足として示す**。安易に自由式言語を増やさない。
2. **仮想Fieldの一計算**：数量×単価など、元Field→Data Script→Virtual Attribute値を同じ型で出し、read-time／update-timeの観測時点を分ける。`DERIVABLE`のX-TEA定義とScript算定のLocatorを紐付け、Script本体の公開Repo転載を避ける。
3. **フィールド変更に伴う純粋判定・算定**：`oldValue/valueChanged`を使い、0→1／同値更新／NULL→値／空文字↔設定／複数Field同時更新を比較。旧Driver内で実行されたScript順序と、TEIの提案Phaseを別欄に記録する。Rollback・Script例外後に`true`が残る負例を併記し、`boolean true`を処理成功の根拠にしない。
4. **同一Data更新に伴うFlow候補**：上記の導出値／状態条件を、TEIのFlow table/CapabilityへBindingする。旧モデルにFlow ID・Effect資格が存在しない場合は**新Owner判断の意味**として保留。Data Scriptに隠れたJob実行を持ち込まない。
5. **実CommitとBatchへの接続（別Gate）**：Rollback、ACK喪失、再起動、同一Event再送、異時点Read、Grant撤回を実Provider/Hostへ引き渡す。多Provider／外部Queue／大規模Jobの全保証まで待って1〜3の単純移植を止めない。

最小成果物は「X-TEAの元要素／Rule event／Script・Field Locator／時点／Input→Output／TEI候補ID／対応可否／差分／独立Expected」の**1行1根拠の対応票**。RMT-2のTranscription／loss-ledgerを使い、第二Canonicalや巨大な再実装調査票を作らない。原Sourceの内容・秘匿IDやScript本文は必要に応じ非公開で管理し、GitHubの公開PRへ複製しない。

**Doneの境界**：P1〜3はSource-faithful candidateと限定Fixtureの同値まで。Aの既存Adapter経路が通るなら新Core/DSL/Pluginを提案しない。実Provider／正式Targetで同じ拒否・更新・回復が独立観測できたときだけ**実行互換**へ進める。Owner受理はこのどの技術PASSとも別。X-TEAの既存設計を過度に複雑な新基盤で置き換えない。

### 7.3 EKIにJob制御を委譲する場合の実行境界（2026-11-03）

[TEI→EKI委任の共通契約／Plugin拡張／Tx・Callback設計](TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)へ、**Data Scriptで宣言した通常Field／Virtual Attributeの変更→Flow→Batch/大規模Job**という要求を接続した。TEIがData/Rule/Flow/Ownerの意味を持ち、EKIがTask／Attempt／Lease／資源配置／Result検証を担当する。EKI Console/Policy Review等はTEIローコード生成の候補だが、EKI KernelはTEI製品稼働へ依存しない。

新たな先行決定をせず、次を先に反証する：
- **発火時点**：Data Scriptの更新前純粋判定と実ProviderのCommitを分離し、厳格なEdge起動には同一Scopeのbefore/afterと同一Tx Outboxなどの証拠を要する。Txn外のEKI SubmitをRollback可能なWrite中に呼ばない。
- **インターフェース**：EKIは言語中立なTask/Result/Lease/Authorityの共通契約を一意に所有し、TEI PluginはTrigger/Rule→EKI申請・Callback配送方式・Provider差分を吸収する。任意PluginにEKIの権限／再試行／結果受理を上書きさせない。
- **Callback**：通知はAck/Progress/ResultのObservationで、Business Commit／Accepted Meaningでない。Pull照会を初期基準、Pushは耐久受領・重複・順不同・再取得を含む追加のCapability候補。EKIにWebhookの実装済みを仮定しない。
- **大規模Job**：TEI Flow GraphとEKI Task Group／Workflow DAGを同一視しない。依存Jobの業務意味はTEI側、資源／予算／AttemptはEKI側。現EKIのTask Group等が実E2Eで利用できるかは別の受入Gate。

JD-01〜09の相関・Rollback・重複通知・Effect Unknown等の独立Expectedは新資料へ集約し、本資料では第二のFixture/Engine/Runtimeを作らない。EKIの設計主権とTEIのMeaning Ownerを維持する。

### 7.3 ステートに基づくJob/Flow管理はTEIが基本、実行継続だけEKIへ部分委譲（Owner方針、2026-11-03）

[TEI→EKIの限定委譲・状態機械の役割比較](TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)§8に従い、**前版の「ジョブ管理の所有者はEKI」という一般化を撤回**する。Data Model/ScriptのVirtual State・Validation・Derivation、状態変化から起動するFlowの**意味・Step依存・次工程の条件・再実行の業務妥当性**はTEI側の責務。TEI生成Applicationが既存Hostとして常駐していれば、この契約の実際のJob/Flow進行もHost側で管理できる。

一方、**Data Scriptそのものは常駐サービスではない**。Go CLIや他Hostに組み込まれるライブラリだけではClock/Queue/CDC/Retry/JobRunを維持できない。この場合、明示的なOS Scheduler、既存Host、EKI、他のJob Provider等の実行管理主体を選ぶ。EKIはJobのTask/Attempt/Lease、資源配置・中断/回復・結果検証を担う**任意の専門実行先**であり、TEIのJob意味を置換しない。永続Timerや限定Step進行まで委譲する場合は、採用済みFlow Revision・Executor／Source・Grantの照合と、対応するEKI外部Contract/Conformanceを要求する。**現EKIでStep DAG・Trigger Registryが実装済みという意味ではない**。

Data Providerがcommit/outboxや必要なRead completenessを提供できない場合、strictなVirtual-edgeの永続Event発生を推論しない。systemd/cronは明示した実行主体またはEKIサービスのbootstrapとしてのみ扱う。同一Trigger/Stepを二つのCoordinatorが独立して実行しない。TEIの純粋計算・単発実行がEKI導入を要求することもない。


### 7.4 仮想Fieldの変化が「業務変更」か「Schema/Query解釈変更」かを区別する（2026-11-03 横断反証）

<!-- DERIVED_TRIGGER_SCHEMA_COHORT_20261103 -->

**接合理由**：本計画§3/§7.3は `Derived Value change`→`Flow candidate`→`Provider Commit/Outbox`→`EKI Task` を分離している。別レーンの[SQL意味境界 Draft #2778](https://github.com/hat47x/tei-platform/pull/2778)における**SM-15実SQLite 14件**では、**同じSQL byte**でも `NATURAL JOIN` の暗黙列集合がSchema変更によって変わり、結果が2行→1行となった。`SELECT *`の出力形状も3列→4列となる。独立した[SM-16実Go DerivedIR characterization](https://github.com/hat47x/tei-platform/blob/docs/sql-static-semantic-evidence-handoff-20261103/internal/query/derived_sm16_schema_drift_characterization_test.go)は合成解析入力に関する別試行であり、実SQLGlot→Go→Provider→Data ScriptのEnd-to-end成功ではない。

**新しい圧力**：SQLがData Modelの正本にならないTEIでも、Data ScriptがData Operations/Queryを使用して仮想項目・状態を導出すれば、**Script Source不変／業務行不変／Query Source不変でも、解決済みSchemaやRelation/Providerの変更で導出値が変わり得る**。このような再計算差分をそのまま `oldValue != newValue` の業務Eventと解釈すると、取消・通知・料金計算・EKIの大規模Jobを根拠なく起動しかねない。

**固定公開合成例（SM-15と同じ作例を使用／正式TEI DSLではない）**：
- `orders(tenant_id,order_id,status)` にO1=draft、O2=submitted、`payments(tenant_id,order_id,amount)` に両注文が存在。SQL `SELECT order_id FROM orders NATURAL JOIN payments ORDER BY order_id` の結果はO1/O2。
- paymentsへ `status TEXT DEFAULT 'draft'` を追加。元注文・支払データの識別子とSQL Source byteは変更しないが、共有列が増えた結果はO1のみとなる（SM-15実SQLiteで確認済み）。
- 仮想述語 `payment_match(O2)` は前=true、後=falseに見える。しかしこの差は、**元の支払取引の取消が観測された証拠ではなく、SQLの解釈・Schema Mappingの変更に起因し得る**。仮想属性の現在資格を再評価する契機にはなるが、`true→false`をBusiness Cancel Eventとして自動発行してはならない。移行に伴う再処理をOwnerが明示受理する場合は、**移行・再評価を起点とする別の業務Action/Intent**として扱い、元のBusiness Commitと区別する。
- 対照：同じ条件で `JOIN ... ON o.tenant_id=p.tenant_id AND o.order_id=p.order_id` と明示したSQLはSM-15ではO1/O2を維持した。よって**Schema全変更で全Queryを一律失効させる**のではなく、解決できた意味依存の閉包に沿って再検証範囲を限定する。閉包の完全性が不明なら「無影響」は主張しない。

**実行に用いてよい差分の必要条件（採用前の設計ゲートであり共通Runtime APIではない）**：
1. **Meaning**：Stable Subject、Owner受理されたRule/Operator版、Virtual Field/Flow条件、Scope/Grain/NULL/missingの意味が固定されている。
2. **Query Binding**：原Query/SQL bytesだけでなく、Dialect、解決時点のSchema Revision、暗黙JOIN/Star/View/Path、Provider Mapping、非出力Predicate/Relation依存と解析Coverageが現在資格を持つ。
3. **Before/Afterの比較可能性**：同じSubject/Business Period、互換なMeaning/Schema・Read Cohort・必要Field completeness、実際の取得値が確認できる。異なるQuery MappingやRead epochから得たtrue/falseは、同じ業務StateのEdge比較としない。
4. **Eventの帰属**：`business write committed`、`schema/mapping migration`、`source rewrite`、`cache replay/recompute`、`late/partial read`を分類する。Schema revision変化だけでBusiness Commit/Outbox発生を推定しない。業務Eventが必要ならProviderの実Commit/Receiptを用いる。
5. **Effect資格**：Business Triggerを実行する場合は#1129/#1174の現在Host認可、Tx/CAS、Idempotency、Effect Journal/Receiptと、再起動後の独立Readを別に確認。EKIに委任するJobは同一Event/Job/Attemptの相関、予算/Deadline、結果Bodyと実業務Effectを区別する。[期限・予算・Recovery計画](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)の再試行提案も未知Effectを自動再起動する理由にはならない。

| ID | 追加する独立Expected／停止条件 | 既存Owner |
| --- | --- | --- |
| **DS-01 暗黙JOIN** | `NATURAL JOIN`のSchema変更でO2が結果から消えても、同一Business Commitを観測していない限りBusiness Cancel Event/Jobを作らず、Derived Claimは要再資格化 | #2387/#2622、#1129 |
| **DS-02 明示JOIN対照** | 明示的なON条件のQueryとその必要依存が不変なら、無関係列追加のみで「意味変更確定」にしない。ただしProvider実Readの現在資格は別 | #2387/#2582 |
| **DS-03 Starの形状差** | `SELECT *`の3→4列を以前のColumn PositionやDTO Mappingで黙って解釈しない。旧生成/Consumer資格を保持せず再照合 | #2387/#2600、#2497 |
| **DS-04 0件／欠測** | `NOT EXISTS`や0件、partial/permission-denied/unknownは、Business falseや「不存在の全件証拠」へ自動昇格しない | #2622/#2514 |
| **DS-05 Before/After非比較** | Replica t0とPrimary t1、Schema s1/s2を跨いだtrue→falseをEdge Eventにしない。各時点のQuery Resultとして別記 | #2582/#1129 |
| **DS-06 Schema ABA** | Schema s1→s2→s1または結果true→false→trueで前後が同値でも、途中無変更・旧Approvalの現在有効性を推定しない | #2582/#2600 |
| **DS-07 Commit→Outbox** | 同Tx内の業務Writeなしの再計算では永続Business Eventを作らない。Migration起点再処理は独立Intent/Owner判断と相関IDを要する | #1129/#2564/#1174 |
| **DS-08 EKI重複・unknown** | Schema変更でDerived値が揺れ、EKI JobがACK timeoutでも、未知Effect照会前に新Job/Attemptを自動生成しない。Budget/DeadlineはAuthorityを代替しない | #1130、EKI #57/#112 |

**優先実証手順**：①SM-15の正負2本を参照する**read-only Data Script/Virtual Outputの合成fixture**を作る（実SQL結果とDerived値は別Observerとして記録）、②同じRevision/CohortでのみBusiness Edgeが成立するか、DS-01〜06を独立Oracleへ接続、③OwnerがMigration起点再処理を要求した場合だけDS-07へ、④実Provider Commit/Outbox/ReceiptとEKI実TaskをDS-08で検証。新Query Engine・State Store・二重Job Schedulerを作らない。まずX-TEA直接移植のP1〜3を妨げず、実Model/Provider実接続が必要な段階だけ追加資格とする。

**Evidence/未決**：SM-15の隔離SQLite実行とSM-16の合成解析入力Go特性試験は別Repo Draftの記録であり、本節で新たな実行はしていない。DS-01〜08のTEI Data Script/Studio/Provider/EKI実統合、Schema migration後の実Observer、Ownerの「Migration起点で再処理してよい」受理は**すべてNOT RUN/NOT ACCEPTED**。Source/SQL/Schemaの意味同値を既存Proofから推測しない。#2778はSQL解析依存閉包のOwner、本Draft #2744はVirtual/Trigger ConsumerのOwner、#1129/#1174はBusiness Effect、EKIはTask/Attempt/予算・回復を保持する。

## 8. 不明／Ownerが決める条件と現在のEvidence

**Ownerの具体判断が必要**：`billing_ready`は明細の完了だけか、出荷実績確定・検収・時間を含むか。訂正／返品・取消で何が起こるか。再計算時の粒度と許容遅延、Strict EdgeかLevel Reconciliationか、同一Subject二回目の正当なJob起動、依存先が別Providerなら整合Snapshot不可能時に停止するか、履歴・監査と重複通知の許容範囲。必要なときだけOwnerが意味とEffect Contractを受理する。

**既存Evidenceの上限**：TEIに`Derived Attribute`／`Data Rule`の設計と実験Reference、`Period`の限定契約、`reference-expected-state-update`の合成CASがある。ただし実`virtual state change→committed event→batch invocation→provider effect receipt`の通しE2Eは確認していない。**本資料はPROPOSAL、設計FixtureはNOT RUN、実DB/Host/Batchの保証はNOT ACCEPTED**。旧Buriの静的実装からこのイベント基盤を実装済みと推定しない。
