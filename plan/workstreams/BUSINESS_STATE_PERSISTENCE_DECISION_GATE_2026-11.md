# 業務状態の物理保存・時点・権限の横断受入ハンドオフ（2026-11-03）

> **状態：計画差分／未採用／NOT RUN。** 業務状態の新Core型、固定状態テーブル、Workflow Engine、Migration Engine、認可Engine、SQL解析器を設けない。実装事実や受理済み意味の新規宣言ではない。
>
> **目的：** 既存の「SQL静的事実／Meaning／実DB」「Buri→TEI状態移植」「組織の時点関係」「Data-attached Script／Provider」「組織を経由するレコード可視性」を、同一の変更・受入ケースで照合する**実行順と停止条件**だけを担当する。設計本文・各機構は元Ownerに置く。
>
> 関連する進行中Draft：[#2778 SQL/DDL証拠境界](https://github.com/hat47x/tei-platform/pull/2778)、[#2744 Buri→TEI状態](https://github.com/hat47x/tei-platform/pull/2744)、[#2776 Data Script/Workflow Source移植](https://github.com/hat47x/tei-platform/pull/2776)、[#2389 組織Relation/時点](https://github.com/hat47x/tei-platform/pull/2389)。**これらのDraftは未統合であり、本書から採用済みとして扱わない。** 本書は既存のSC-/BL-/RV-/ORG-/WSB-テスト群を複製しない。

## 1. 最初の判断：「同じstatusか」ではなく、何を状態と呼んでいるか

| 業務対象 | Accepted Meaningの単位 | 現在値の自然な正本候補 | 分離が合理的になる条件 |
| --- | --- | --- | --- |
| **受注の確定／取消** | Orderの業務Lifecycle、Data Script上のRule・Action・禁則・副作用 | Order本体のstatus＋version（必要なら独立イベント履歴） | 監査Retention、取消前後の独立観測、別のトランザクションで確定する効果 |
| **組織所属／職務占有** | Employee–Organizationの**時点付きRelation occurrence**。兼務・主所属・役職等の排他は別Rule | Membership/Assignmentという業務Relation行。従業員の単一statusではない | 有効期間の複数行、遡及訂正、兼務／将来異動、別のRelation lifecycle |
| **Workflow進行** | Workflow Instance／attempt／active occurrence（または単純な対象状態） | 単純な1件なら業務本体status／複数軸なら列・JSONB・ARRAY | 複数instance、独立したstep identity/並行更新・承認待ち、独立の処理寿命 |
| **参照専用の横断状態一覧** | 正本ではなく、業務別状態のread projectionとその可視性・時点 | SQL view/UNION/関数的なData Operations | 大規模件数・別Providerの鮮度や権限が統合に影響するなら明示Read Model |

- 「本体行優先」は**状態がその行の単値・同一更新寿命に属するとき**の判断であり、Membershipを従業員本体の状態列へ潰す命令ではない。
- 横断状態の検索要件は中央**物理的Writer**の採用理由にはならない。まず各表からの状態ProjectionとQuery計画、Grant/Scope、読取鮮度を検証する。
- 履歴・監査・Outbox・Receiptはcurrent-state writerと必ずしも同じ保存単位でなくてよい。イベント履歴があることと、Event Sourcingによる完全な現在値再生の保証は別。

## 2. 保存方式の比較で先に問う条件

| 方式 | 検索／時点 | 制約・同時更新の焦点 | 移行時に保全する対象 |
| --- | --- | --- | --- |
| 本体status一列 | index/filter、単純なas-is current。過去履歴は別 | 有効なstatus集合と遷移Guardは別。行version-CAS、他属性との整合 | 状態コードの対応、未確定／NULL、旧履歴の欠損 |
| 本体複数列 | 直交状態軸の複合Predicate | 軸間不変条件、同じ行への並行更新・lost update | 「直交」か「階層」かの意味、旧遷移経路 |
| JSONB／ARRAY等の構造化列 | provider固有path/containment、GIN等の索引候補。無索引時は別評価 | JSON欠損≠null≠空、ARRAYの順序/重複、branch identity、要素更新の原子性・row CAS | 型・版・要素Identity・集合多重度・NULL/空の意味 |
| 業務別Relation／Instance表 | 有効期間・複数step・複数attempt・履歴の自然なSELECT | PK/FK/業務Key/期間重複、独立更新のTx、deleted/closed行 | 関係occurrence、valid/recorded time、履歴、旧キー |
| イベント／監査専用表 | event timeと記録時刻の区別、再生は追加要件 | 順序、重複/冪等、欠落・遡及修正、currentとの結合 | 履歴完全性、イベントschema/version、再生規則 |
| 中央共有Writer／外部Store | 複数Domainの独立運用・監視用途 | 業務型×Tenant×Key×Process instanceの混同、DB間原子性・許可 | writer切替、既存workflow occurrence、二重書込の停止 |

**選択条件**：まず意味的多重度／更新寿命／履歴要求／制約で適否を判定し、その後にQuery性能／Provider能力／移行費用を比較する。JSONBが格納可能だから採用、テーブルを横串で見るから中央Writer採用、とはしない。複数方式にread replicaや履歴を併用しても、**各current logical stateの権威Writerは明示的に一つ**とする（移行期間の特別契約を除く）。

## 3. SQL解釈とAccepted Meaningの境界（既存S/C/O/Aに渡す）

- **S：Source-static** はDDLの型、PK・複合PKの列順、FK、CHECK、JSON path/ARRAY式、トランザクション記述とSQL構文上の制約を把握する。CHECK値域は合法な状態*値*を宣言しても、submit/cancelの遷移可否、取消効果、所属の重複可否、実Host Grantを確定しない。
- **C：Candidate** はstatus列→意味Role、複合Key→Business Subject、所属期間列→valid time、イベント列→history等の対応候補を出典/版とともに提示する。列名一致から受理、またはSQL FROM/JOINから業務Relationを決定しない。
- **A：Accepted Meaning** はOwnerが受理したData/Rule/Relation/State/Scope/時点・権限要求であり、Script IF・明示遷移の**二重正本を作らない**。構文解析不能なIFはOpaque/partialとして保持し、完全遷移図と偽らない。
- **O：Observed** はexact DB Alias/Provider/schema/Tx/current principal/RLS、実行したSQL/更新件数/Result Slice/Commit receipt/独立Readの限定試行。AがあってもOは未達の場合がある。Oが成功してもA、現在Auth、他Provider成功は導けない。

既存[#2778](https://github.com/hat47x/tei-platform/pull/2778)のS/C/O/Aを**本書独自の状態enumやAuthority台帳に再定義しない**。Source byte、schemaの正・負の依存（例：`NATURAL JOIN`が参照してしまう新規同名列）、Dialect、Function Context/Read footprint、Operator版、Consumerの用途・開示範囲が変わった場合、解析Result・生成Query・現在Read Claimの再利用資格を**別々**に判定する。最新のSource静的解析結果から業務意味を自動受理しない。

## 4. 時間・組織・可視性を物理statusへ押し込まない

**組織についての時点を分離する**：`validAt`（業務有効時点）、`knownAt`/recorded time（その時点で記録済みだった事実）、`publishedOrgRevision`（一括改編の公開版）、`decisionAt/effectAt`（実行・確定時点）。四者を単一の`updated_at`または`status`で代用しない。

期間の原則は**半開区間**を比較候補とするが、時刻精度、当日含み、TZ、無期限、遡及訂正、主所属・兼務の排他範囲はOwner判断。独立したMembership occurrenceへの複合Key／業務Key、組織・役職・職務占有・委任の意味を混同しない。スキーマのFK/uniqueやSQL範囲型だけでは、どの重複が不正かを確定できない。

**組織を通じたレコード可視性**：業務Relationの推移的参照から適用候補を広域選択し、必要なら個別例外／参照専用を設定する案を検討する。ただし「全体へ有効化→除外」は設定の**設計候補**であり、検索・表示・編集・実行・行/項目開示の権限が同一になるわけではない。別Provider/HostのRLS・Grantで実際に強制されたRead/Writeと、Previewで提示した候補を分離。時点不明・組織異動直後・部分取得・複数親経路・Tenant越境・deny例外の競合を、見落としたまま許可へ倒さない。Unauthorizedな診断に秘匿対象の存在も漏らさない。

## 5. 同一Scenarioの受入・反証Gate（既存Fixtureを**再利用**）

| 縦切り | 固定する独立Expected／反例 | 受渡し先 |
| --- | --- | --- |
| **BG-01 受注** | `pending`→`submitted`、確定と取消の競合、二重submit、同一DBでのversion-CAS。業務拒否／技術失敗／権限拒否／Commit ACK不明を別結果、予約・outbox・独立Readを別確認 | #1129 WSB-E01, #2564, #1174、既存Lifecycle/Effect Reference |
| **BG-02 組織** | 同人物の兼務重複は許容し、同一ロールの排他所属期間重複だけを拒否候補。2027-04-01異動を04-10に遡及登録して、`validAt`同じ・`knownAt`違いを区別 | #2386 ORG系、#1126 Relation/Temporal Binding、Draft #2389 |
| **BG-03 Workflow** | 同一Orderから二つのattempt/active branchを持つ。単一statusへ投影して同値と誤判定しない。必要なら複数列→JSONB/ARRAY→関連表を比較し、欠損・JSON null・空配列・重複・branch ID／同時更新を観測 | #1129 WSB-E02〜04, #2622、Draft #2744/#2776 |
| **BG-04 横断Read** | 状態一覧を個別業務本体からUNION/viewで構成。current/history・権限/tenant/snapshot・0件完全性・計測した計画を記録。中央共有表なしで成立した範囲と不足を分離 | #2622、#2506/#2586、#2387、#2516 |
| **BG-05 組織→可視性** | 所属・階層変更とrecord-level read-only/非開示が異時点で発生。StudioでAction候補が見えても現在Host/Provider read/write grantを証明しない。Previewは許可を発行しない | #2386、#2507、#1174、#1656 |
| **BG-06 Source/Schema drift** | Ruleのみ変更、JSON path名変更、DB schema追加、A→B→A、Source移動。SQL byte不変でも受理済みData Ruleや実Schemaが変わる負例。静的Derived IR・Build artifact・Read claim・Authorityの失効理由を分離 | #2778 SC/SM-15/16、#2387、#2582、#2600 |
| **BG-07 物理移行** | 本体status→JSONB／業務別関連表／中央Read Modelへの変換を比較。NULL・時点・業務Identity・Workflow attempt・履歴・権威Writerの保存と、read-only indexの遅延／二重Writer停止を判定 | #1126、#2622、#2611、既存Migration Owner |

**この表は新たな同型Fixture群を開発せよという要求ではない。** 既存Fixture/独立Oracleが必要な分類を実際に判別できるかをまず照合し、欠落が確認された一条件だけを各OwnerのReferenceに追加する。合成FixtureのみのPASSを実DB/実Provider成功としない。G0〜G4など既存Gateを改訂しない。

## 6. Ownerへの最小判断票（未受理のまま保持）

1. **受注**：submit/cancel後の再開／再確定、取消の業務効果、予約の同時確定、履歴・訂正に要求する不可逆性は何か。
2. **組織**：兼務・主所属・役職占有のどの組合せを排他にするか。業務有効時点と記録時点を双方保存すべきか。未来日付異動/遡及訂正の公開条件は何か。
3. **Workflow**：業務対象1件につきprocess instance/attempt/active branchが何件存在し得るか。複数status列やJSONの部分更新・履歴保存で足りるか、独立した寿命が必要か。
4. **Provider/セキュリティ**：状態変更は他の業務項目と同一DB Txで原子的か。JSONB/ARRAYの部分更新、行CAS、可視性RLS、複数Provider間のSnapshotは実環境で何を保証するか。
5. **移行/横断**：状態検索の時点・鮮度・範囲・可視性要求は何か。独立中央Writerが必要となる**業務上/運用上の証拠**はあるか。履歴・イベントのRetentionと旧データ欠損をどう扱うか。

Ownerから判断がない意味には**未受理**を明記し、AI/SQL/DSL変換で推測して埋めない。横断Readが必要なだけでは中央Write Storeを提案しない。

## 7. 実行順・完了条件・停止条件

- **P0 既存証拠棚卸し**：#2778（SC/SM-15/16）、#2744（BL）、#2776（RV-S）、#2389（ORG）、#1129（WSB-E）、#2564のSource/Fixture/Expected/exact HEAD/CIを固定。実行済みと未実行、syntheticとreal DBの範囲を分ける。新しいTest Suiteの先行整備はしない。
- **P1 一件の意味→Binding**：BG-01でData Script IF一本をOwner未受理と明記して候補化し、`Order.status`/version＋履歴への**明示**Bindingと静的Query/Update Shapeを出す。SQL解析からScript RuleやTransitionを逆採用しない。ScriptからのDiagramはpartial coverageのみ。
- **P2 一件の実Provider観測**：P1と同じ複合Key/Model revision/Provider/Target条件で、Current Grant、条件付きUPDATE、行数、Tx、Receipt、独立Read、効果Outboxを観測。観測を実行できない場合は`NOT RUN`で停止し、合成adapterの成功へ置換しない。
- **P3 異質な意味の差**：BG-02組織の時間Relation、BG-03 Workflow多重Instanceを同じData/Relation/Rule/Provider契約で表せるか確認。無理な共通State kindを作らず、補助Plugin/Physical Mappingの必要性を反例で判断。
- **P4 横断・移行**：BG-04/05/06/07でRead modelの時間・権限・開示、schema/Rule drift、version移行の損失、生成コードの効率を別々に測る。生成SQL query count/実行計画、cold/warm、編集→利用可能artifactまでを測るが、未計測の改善率は書かない。

**受入できる判断**：業務的同値性、物理保存の適合/非適合、実Providerにおける保証、現在Authority、生成/実行性能を**別列**で説明できること。初回にはstatus一列が本当に十分なケースだけ成立すればよく、同時に全Workflow機構を製品化しない。

**停止**：Owner Meaning未受理、型付きBusiness/Workflow identity不明、read completeness/時点/Grant不明、Provider CAS/部分更新能力不明、別DB Txの原子性未証明、履歴欠落・移行でのIdentity損失があるとき、該当の**適用/確定/移行**を保留。説明やread-only Candidateまで停止させるわけではない。

## 8. 責務と非目標

| Owner | 本書から返す内容 | 作り直さないもの |
| --- | --- | --- |
| #1126 | 状態Field／Key／Relation occurrenceと物理Mapping、期間/複合Key反例 | 新しい共通Data Model／DDL Engine |
| #1129／#361移管計画 | Script-authoritative IF、Guard/Transition、Lifecycle/Effect/保存境界 | 専用Workflow Host・新しい同値Rule正本 |
| #2386 | Membership/Occupancy、valid/recorded time、主所属/兼務の業務意味 | 状態コードからの組織業務意味逆推定 |
| #2622 | 複数Field/JSONB/ARRAY/sidecar Query・更新、Provider能力/実Read | 共通State Store、中央Query Engine |
| #2387／#2506 | S/C SQL Explain、明示Data Contextと実利用Queryの差 | Source SQLからのAccepted Relation |
| #1174／#2564／#2611 | 実Hostのcurrent Grant・DB条件付き更新・複数Alias実対象 | 第二Authority/Tx engine |
| #2497／#2621／#2018／#2600 | 表／図の部分Projection、元DSL書式、保守性・Build品質の測定 | StateMachine専用Core/Editor |

既存設計文書から本書へ機能正本を移さない。採用判断が必要なら対応するOwner/ADRへ提案として戻す。**GitHubへのこの追加は計画整合であり、Source meaning受理、製品実装、実DB/Host試験、Migration成功を主張しない。**

## 9. 第2次深化：生成時Bindingと実行時の正しさを分ける（2026-11-03）

他チャットのX-TEA動的Data Access／TEI生成コード方針と、EKIジョブ／組織／Plugin／Build議論を照合した追加受入条件。**採用済みABI・実装済み性能・新たなEngineの提案ではない。**

### 9.1 何をビルド時に確定できるか

| 生成時に固定するもの | 実行時に再確認するもの | 固定と再確認を混同した反例 |
| --- | --- | --- |
| Data Model／Relation occurrence／Rule/Function/Operation identity、選択Provider/Target/方言、Query template、生成器版 | 現在のDB Alias実対象、schema互換、Tenant/Principal/Role/RLS、入力値、Query結果の完全性 | 正しい静的SQLでも実DB schemaが古い／Aliasが別Tenantを指す |
| 状態変更のGuard/Transition式、Expected-version条件、Typed update path | 現行行version、現在状態、同時更新、Tx隔離、更新行数、Commit結果 | ビルド時に合法なsubmitでも実行時にはcancel済み |
| Membership/Occupancyの意味Role、期間演算子、明示Relation path | `validAt`/`knownAt`、公開組織版、現在Grant、可視範囲 | ビルド時所属に基づく担当候補が異動・委任失効後も実行できる |
| Workflow instance/branchのIdentity・多重度、Provider能力宣言 | 既存instance版、active branch、attempt重複、外部Effect receipt | 一つのOrder status更新成功を、二つの独立branch完了と誤認 |
| 生成物のWorld/Model/Artifact/Generator/Plugin版・依存閉包 | Provider/Schema/Policyの現行資格、実データの状態 | Artifactのハッシュ一致を現在の認可・業務意味受理へ昇格 |

**原則**：X-TEA型の実行時モデル探索・リフレクション・Function→DataAccessの都度解決は、TEIのコード生成で**削減候補**となる。一方、DB round-trip、JOIN fanout、N+1、行復元、索引、競合、RLS、部分取得は残る。生成コードの速度優位を測定前に断定しない。複数Targetに同じRuleを生成しても、Runtime GrantやTransactionの保証を共通化したとは主張しない。

### 9.2 同じActionの5つの時点を保存する

**モデル版**（何を生成したか）／**判断時点**（どの状態・所属・Ruleで候補と判定したか）／**送信時点**（どのProvider/DB AliasへどのExpected条件で書いたか）／**Commit判定時点**（確定／拒否／unknown）／**独立観測時点**（どのRead Cohortで確認したか）を区別する。

- Previewで`allowed`だったActionを、実送信時のGrantやGuard再検査なしで実行しない。
- 同じOperation IDを再送しても「二重副作用なし」を保証するには、Provider/外部Effect側のdurable dedupeまたは照合可能なreceiptが必要。メモリ内の成功フラグだけでは不十分。
- Commit ACK喪失時は**失敗確定でも成功確定でもない**。重複送信を禁止し、receipt／独立Read／専門Providerのreconcileへ進める。未知の結果を別の業務更新で上書きしない。
- DB内のstatus更新と外部通知は同じ原子Txとは限らない。必要なら既存Effect/Outbox契約で扱い、専用Workflow Hostを作らない。
- Job/Batch/CLI/Webから同じ業務Actionを呼ぶ場合、呼出し形態ごとの権限・deadline・retry policyは外側の運用契約。TEIの生成Functionは共通の意味と入力・結果を保持するが、ジョブスケジューラの資源最適化を状態モデルへ埋め込まない。

### 9.3 JSONB/ARRAY/本体列／関連表の公平な比較

比較単位は**同じ業務意味と独立Expected**。候補ごとに変わるのはPhysical Binding/Provider SQLであり、意味上の成功条件は変えない。

1. `Order`単純遷移：同じ`(tenant, orderNo)`、同じversion-CAS、同じcancel競合を、本体列／JSONB pathの両方で比較。JSONBの`missing`・JSON `null`・SQL `NULL`・空配列を分離する。
2. 複数active branch：ARRAY要素の順序・重複・branch ID・局所更新が必要なら、JSONB/ARRAYの配列置換と関連表の行CASを比較。構造化列で原子性が保証できなければProvider非適合とし、意味を弱めない。
3. 時点付きMembership：期間境界、兼務、遡及訂正、valid/recorded二軸、Tenant越境、非開示を、関係行とJSON埋込み候補で比較。索引やQuery速度だけで正本を選ばない。
4. 横断状態一覧：本体行からのView/UNIONと専用Read Modelを比較。read-only replicaの遅延・Snapshotずれ・Grant差分を測定。読み取り投影の作成はWriter移譲ではない。

**最小測定票**：`{same semantic case, expected/result, storage variant, generator+provider+schema revision, target, query count, p50/p95, allocations, read scope/completeness, concurrent update result, receipt status, edit-to-ready, evidence}`。数値の目標・優劣はPoC測定後に判断。実DBと合成Providerを同じ系列に混ぜない。

### 9.4 受入に使う具体的な停止反例（BG追加番号は作らない）

| 既存BGへの追補 | 反例と期待 |
| --- | --- |
| BG-01／BG-03 | Preview時`pending`、実行直前に他者が`cancelled`へ変更。静的に生成された合法ActionでもCAS/Guardが拒否し、業務拒否と技術失敗を分離 |
| BG-01／BG-07 | UPDATE Commit後にACKだけ喪失。Retryが同じ業務Effectを二重に実行しないことをreceiptで観測。証拠がなければ`unknown`で停止 |
| BG-02／BG-05 | 組織版r1の担当者候補がr2公開後に失効。既存候補・保存済みWorklistは現在Grantではない。秘匿行の存在を診断へ漏らさない |
| BG-03 | JSONB pathでactive branchを更新した直後、別branch更新が同じ行を上書き。部分更新能力の宣言だけでなく実同時更新の結果を観測 |
| BG-04／BG-06 | SQL byte・Model IDは同じでもSchema/Policy/DB Aliasが変化。生成Queryの静的整合、実Read Claim、現在Authorityを別々に失効判定 |
| BG-07 | JSONB→関連表の移行で二つのbranch IDが一つに潰れる。値集合が同じでもIdentity保存失敗としてMigration不受理 |

### 9.5 次に着手する最小の縦断検証

**P0.** 既存#2564/#1129の一つの受注更新Fixtureを選び、Owner未受理のRuleや欠けているreceiptを明記した独立Expectedを固定する。

**P1.** 受注本体statusへの静的Bindingを一つ生成できるか確認。生成経路が未実装なら`NOT RUN: generation-unavailable`として止め、手書きSQLの成功を生成成功に代えない。

**P2.** 同じ版のFunctionをCLI/Job/Webのうち実際に存在する入口から呼び、実ProviderのCAS/Guard、Commit/ACK、独立Readを観測。入口が未存在なら当該経路だけ未実施とする。

**P3.** JSONBへの代替Bindingを、同じExpectedと同じDB条件で比較。多重branchやMembershipは、単純受注が成立した後に追加する。SQL生成時間、実行時間、Query数、ビルド時間を別に計測する。

**P4.** Schema/Rule/Plugin/Provider改版のうち一件だけ変更し、どの生成Artifact・Runtime Claim・Consumer公開が失効するかを実際に記録する。

**引継ぎ**：#1129はRule/Guard/Effect、#1126はMeaning/Physical Binding、#2622はProvider/Query/性能、#2564/#1174はCAS/Current Grant/Receipt、#2600は生成物の鮮度と編集待ち時間、#2018は公平な比較基準を担当する。新Issue/ADR・新State Engineは、この一件で不足が観測されるまで作らない。
