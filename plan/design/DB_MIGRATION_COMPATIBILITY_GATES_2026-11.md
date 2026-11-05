# TEI DB移行互換性ゲート：横断統合と最小受入（2026-11-06）

- **状態：検討案。** 本書によって正式Gate、公開API/DSL、Owner判断を採用しない。今回は文書変更のみでGo/SQL/実DB/本番旧Client/異種DBを実行しない。
- **関係**：[意味変更と既存データの移行設計](SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md)、[#2555](https://github.com/hat47x/tei-platform/issues/2555)（旧システム並走）、[#2147](https://github.com/hat47x/tei-platform/issues/2147)（段階移行参照成果）、[#2545](https://github.com/hat47x/tei-platform/issues/2545)（観測窓などの残余）、[#1126](https://github.com/hat47x/tei-platform/issues/1126)（業務キー/PK/履歴）、[#2622](https://github.com/hat47x/tei-platform/issues/2622)（Provider/Data Operations）、[#2611](https://github.com/hat47x/tei-platform/issues/2611)（同Semanticの異DB Alias）、[#1174](https://github.com/hat47x/tei-platform/issues/1174)/[#2564](https://github.com/hat47x/tei-platform/issues/2564)（Effect/Commit Unknown）、[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1163](https://github.com/hat47x/tei-platform/issues/1163)（独立Expected/同一Attempt）、[#2600](https://github.com/hat47x/tei-platform/issues/2600)（Evidence再利用）、[#2862](https://github.com/hat47x/tei-platform/pull/2862)（DB別Native SQL選択）。

## 1. 実装されている境界と未成立のClaim

- 現行 `internal/database/migration/model.go` に `schema/data/verify` Stepと `forward-fix/explicit-rollback/not-reversible` が存在。`ApplyPostgresWithFreshObservation` は破壊的変更直前のfresh Schema observationと計画済みSQLの一致を確認するが、業務データ完全性、旧新Client共存、正本移管、履歴一致、Consumer受領は証明しない。
- `internal/releaseplan/model.go` に `expand/populate/transition/rollout/contract/verify` と `database-release/application-release/application-retire/database-gate` がある。下記のExpand/Backfill/Validate/Cutover/Contractは**この段階への検収観点**であり、第二のRelease Engine/固定Gate enumではない。
- `tools/reference-order-migration-dual-db` は**合成データ＋同一マシン上のPostgreSQL 3インスタンス**で、計画Digest、Checkpoint、変更記録、旧形式書込みの観測、逆同期・巻戻し訓練等を限定的に実証（#2143/#2147）。本書の5Gate統合、異種DB、実旧System、本番CDC/Volume、正式Owner意味受理は**未実証**。
- 意味の受理／Schema再読取／Data比較／旧新Writer運用／実Commit・Receipt／Consumerの実受領を別々のClaimとする。「Migration SUCCESS」一行からMigration完了を推測しない。

## 2. G0〜G5：段階、独立観測、Owner判断、停止

| Gate | 既存Phase・作用 | 独立観測の必須条件 | Ownerの判断 | 許容停止（次段階禁止） |
| --- | --- | --- | --- | --- |
| **G0 Baseline** | 事前の固定Sourceと影響範囲 | 旧新Meaning/Schema revision、Function/Client母集団、業務キーと物理PK、履歴、Mapping版、SnapshotとCDC境界、Plan Digest、独立Expected | 業務Ownerは変換・例外の意味、運用Ownerは対象と観測窓 | 対象DB Alias/有効時点/独立Expectedが不明なら保留 |
| **G1 Expand** | `expand`：旧Client互換の構造追加 | Migration Executorとは別の接続でActual Schemaを再取得。旧Reader/Writer動作、制約状態、Provider能力・ロックを観測 | DBA/運用OwnerがDDL条件、意味Ownerが契約変更を別受理 | 旧Client不適合、予期せぬ破壊・Schema差・lockで停止。追加列は残して前進修復可 |
| **G2 Backfill** | `populate`：Snapshot/変換/CDC/Chunk | 独立ReadでKey集合/金額/行Digest、Chunk/CheckpointとCommit、CDC position gap/duplicate、未処理Queueを照合 | 業務Ownerが曖昧な変換・保留、運用Ownerが再実行・再Snapshot | 不明なコード多重度、CDC欠落、Plan改版、Commit Unknownを未作用とするなら停止 |
| **G3 Validate** | `verify/database-gate`：`transition`前に検収 | 対象Scope/Rowset completeness・履歴・Relation・NULL/Decimal、旧新Clientの同一Scenarioと独立Expected、CDC catch-upを確認 | 業務Ownerが意味同値・例外、検証Ownerが独立性/範囲を認定 | 1円差、履歴欠落、0件の過剰一般化、未観測Reader、未処理変更なら停止 |
| **G4 Cutover** | `transition/rollout`：書込み正本を新側へ | 旧Writer Drain/Fenceと未完Attempt、CDC収束、Owner/Current Grant、実Command Commit→Receipt lookup→別接続Read、Consumer受領、逆同期試験 | **正本Owner**が移管、運用Ownerが切替窓と切戻し判断 | 二つの正本、未完同期、未知Commit、現Grant欠落、切戻し可否不明なら停止 |
| **G5 Contract** | `contract`：旧構造・互換View削除 | 全旧Writer（Batch/直接SQL/外部連携/Queue含む）母集団と十分な観測窓、保持義務と履歴、直前fresh Schema/データ、削除後独立Read | **業務Owner**が保持/破棄を明示承認、DB運用Ownerが破壊操作を承認 | 月次Batch未観測、外部Writer不明、Retention未決、Rollback窓未終了なら削除しない |

**観測時点の失効**：G3合格後でもG4直前のCDC/Writer/Grant、G5直前のWriter母集団/Schema/Retentionを再確認する。旧アプリ配備ゼロ、旧形式の書込みゼロ、旧Writer全集合が退役済み、は異なるClaim。発行主体を識別できない書込みは「無し」にしない。

## 3. 旧新クライアントの共存とDBの違い

| Actor | Expand | Backfill/Validate | Cutover | Contract |
| --- | --- | --- | --- | --- |
| 旧Reader | 旧Contractで継続 | 値/履歴の意味を保つ | 期限とScopeを明示したread互換経路なら残せる | 旧依存を除去または明示延長 |
| 旧Writer/Batch | 継続可能 | Snapshot+CDC等で更新を捕捉 | Fence/Drainで停止し未完Attemptを照会 | 全対象集合の退役を検証 |
| 新Reader | 部分移送はpartial | Completeness・時点とlagを表示 | 現Ownerのnew DBで独立Read | 新Contractだけで成立 |
| 新Writer | 原則非稼働 | 二重正本禁止・例外は明示 | 新Grant/Owner/Operationで実行 | 新履歴/制約でWriteが成立 |

同一DBのSchema移行と異種DBへのデータ移行は別。PostgreSQL固有のDDL、CDC、Transaction、Snapshotの能力をProviderが宣言・実測し、SQLite/Oracleへ無言で汎用化しない。SQLの選択は#2862のTarget→Provider→Realization境界を使い、専用SQL欠落時の黙示的fallbackを禁止。選択されたSQLを実際に使ったことと結果の適合を別観測する。

## 4. Business Key・Physical PK・履歴を保持する

例：業務ID `(tenant_id, order_no)` は同一だが、旧物理PK 11/12/21と新物理PK 901/903/902は異なる。異なるtenantの `order_no=001` を同じEntityに統合してはいけない。旧PK→新PKの対応、FK、子集合の多重度、業務Keyの有効時点を個別に照合し、物理IDの一致を業務意味としない（#1126/#2611）。

履歴は必要に応じて**有効期間・記録/訂正時点・状態遷移・取消・旧新Source版**を保持する。必ず全DBをbitemporal schemaへ変えるのではない。移行後の最新値一致だけで履歴一致とはしない。時点付きコード変換は、変換方向/版/有効期間/多重度を明記し、1対多や曖昧な履歴解釈をOwnerなしで1件へ絞らない（#2555）。機密の具体値を一般利用者の検収画面へ表示しない。

## 5. 再実行・中断再開・切戻しとUnknown

| Failure | 許容される行動 | 禁止 |
| --- | --- | --- |
| Expand部分DDL成功 | fresh Schema読取り→未実施分のみ修復 | 全DDL rollbackをログだけで推測 |
| Backfill ChunkのCommit後にProcess kill | Plan/Mapping/Source版、Checkpoint、独立Readと同一Operation IDを確認し未完分をResume | Checkpoint応答喪失を未作用と推定 |
| CDC gap/slot loss | 停止、既存SourceとTargetを評価し再Snapshot/再同期の新計画 | Gapを0更新として通過 |
| Cutover後に新DBへ書込済み | 逆同期・書込みFence・時間窓・独立Readが確認できた場合のみOwner承認切戻し | 旧DBが自動的に最新とみなす |
| Commit応答喪失・ABA | Receipt/Effect lookup→独立Read→承認されたResume/補償 | 別Idempotency Keyを発行し盲目的再送。CASの値一致で一回性を主張 |
| Contractで物理削除済み | 基本はForward Fix。復元可能性は変更記録/保持データ・以後更新との整合を個別実証 | Backup存在だけで無損失Rollback可能と断言 |

Operation/Attempt、Plan fingerprint、snapshot/CDC offset、Source/Provider版、Owner generation/Fence、Checkpoint、Effect/Receiptを関連付け、`planned / dispatched / acknowledged / observed / unknown / compensated`を別状態にする。異DBへの二重書込みは一つのACID Transactionとみなさず、Outbox・実Effect照会・補償の境界で扱う（#1129/#1174/#2564）。

## 6. 最小Fixture MIG-COMPAT-01（すべて未実施）

固定の合成データは旧`legacy_orders`／新`orders_v2`、同じBusiness Key `(tenant_id, order_no)`、異なる物理PK、tenant A/Bに重複`order_no=001`、履歴2件、期間別コード変換、旧Writer・月次Batch・新版Reader/Writer。Snapshot直後に旧側更新を注入。Expectedは移行SQL/出力値に依存せず別の固定データと業務定義から作る。

| Case | 故障注入と独立Expected | Owner |
| --- | --- | --- |
| MG-01 | Expand後に旧Clientが読書き可、別接続Schema観測一致 | #2147 / #2622 |
| MG-02 | Snapshot後更新がCDCで漏れ/重複なく反映、未追付なら停止 | #2555 |
| MG-03 | Chunk Commit直後kill/応答欠落でも二重行/履歴/Effect 0 | #2147 / #1174 |
| MG-04 | Plan/変換版変更後の旧Checkpoint再利用を拒否 | #2147 / #2600 |
| MG-05 | 件数一致・金額1円差をValidate FAIL | #2514 |
| MG-06 | tenantなしKey joinによる別Entity衝突を検出 | #1126 / #2611 |
| MG-07 | 履歴1件欠落/期間重複/過去コード誤対応を拒否 | #1126 / #2555 |
| MG-08 | 月次Batchが未実行で観測窓内旧書込み0でもContract拒否 | #2555 |
| MG-09 | Cutover後Commit Unknownで新Key再送せずLookup | #1174 / #2564 |
| MG-10 | CDC gap/Reverse Outbox未到達をCutover禁止 | #2555 |
| MG-11 | 第二DBでNULL/Decimal/日時/Collation差は独立ExpectedでUnknown/Unsupported | #2622 |
| MG-12 | Contract直前にGrant/Schema/Source ABAで旧合格再利用禁止 | #2600 / #1163 |

**実証Packet（新Core正本ではない）**：対象Meaning/OperationとSource Revision、旧新Database Instance/Provider/Schema、Plan fingerprintとData Mapping/History Key、Writer/Reader母集団・観測窓、Snapshot/CDC/Chunk/Checkpoint、現在のGrantとOwner判断、実Effect/Receipt/別ReadとScope、独立Expected・負例、各Gateの`pass/block/unknown`と理由。Log success、Schema一致、Data一致、Client互換、Owner移管、Consumer受領、Rollback可能性は各々別のClaim。

## 7. 実施順と既存Issueへの配置

1. **P0 Read-only**：#2143/#2147参照実証と現mainのReleasePlanをMG-01〜08へ対応付ける。既存PostgreSQL参考結果の再利用範囲を記し、新FixtureのPASSに合算しない。
2. **P1 Key/History**：#1126/#2611/#2555の固定SourceでMG-06/07、Scope/時点と一対多の保留を検査。
3. **P2 Cutover/Effect**：#1174/#2564/#2555の実Invoker/Provider/Receipt経路を、同じSource/Owner/Operation/Attemptで独立Readまで接続。Network kill/製品E2Eは別資格。
4. **P3 Provider横断**：#2622がPostgreSQL＋第二DBで同じFunction/Expectedを照合。Schema/Type/Constraint/TxとDoma型Native SQL差異を分離。
5. **P4 Contract**：実Client/Batch母集団、Retained History、直前観測、Owner承認と不可逆操作の復旧条件を確認してから、Provider所有のDDL適用へ進む。

#2555は旧新並走・Cutoverと残余、#1126/#2611はKey/Mapping、#2622/#2862はDB能力/SQL、#1174/#2564はEffect、#2514/#1163は独立Expected、#2600はEvidence鮮度、#1119は既存製品Gateとの整合を所有する。#2545はclosedの参照結果であり、新実装完了や正式採用ではない。**移行専用新Plugin・CDC Engine・Tx Coordinator・Owner受理エンジン・Gate SSOTは増やさない**。

**明示的に未決**：本番CDC方式、旧Writer帰属と観測窓の下限、正本移管と旧Writer Fenceの実原子性、業務Key変換の解決基準、Retention、異種DB間の回復保証、正式DSL/API表現。採否にはOwner判断・必要なADRと実Provider試験を要する。


## 8. 横断深化：Backfillの集合一貫性・変更記録とGateの鮮度（2026-11-06、未採用／未実行）

**今回の差分**：他レーンの#2506（Result Slice/Completeness）、#1640（Filter/Sort/Page順序）、#2582（更新依存）、#2387（Native SQL/Lineage）、#2862（DB別SQL選択）、#2600（負依存/再資格化）から、移行中に「各SQLの成功」が「全対象集合の移送成功」にはならない反例を移す。ここでは新しいMigration Engine、Query Renderer、Cohort型、Gate State Machineを作らない。

### 8.1 Backfill対象を同じRead Cohortとして固定する

Read Cohortは**検収票上の呼称**で、既存のSource/Context/Query/Provider/Checkpointを横断して参照する。必須のWitness候補は次の通り。

| 証拠 | 必要な対象・失敗時 |
| --- | --- |
| Semantic Data/Relationと複合Business Key、必要対象集合・粒度 | JOINで6行になっても実Entityが6件とは限らない。集計/ページ/0件/未取得を分ける |
| Function Context、Application DB Alias、Provider/Instance、read-purposeと権限 | 同名tableや同じQuery IDからTarget DBを推定しない。異なるAlias/Scopeを黙って結合しない |
| 選択済みRealization、Native SQL Source digest、Role/Operator/Schema版 | Query IDが同じでもページ途中の共通SQL→専用SQL切替は同一Query集合として認定できない |
| Predicate/Sortの全順序、Keyset・Offset/ページ位置、Read Scope/RLS | 読取中の挿入/削除と同値順位でskip/duplicateを生じる。LIMIT完了は全件完了ではない |
| Snapshot境界、CDC開始/到達Position、DB読取一貫性、Source epoch | 別Snapshot/欠落したCDCを一つの完全集合として扱わない |
| ChunkのBusiness Keys/Target Keys、Checkpoint/Receipt、独立Readerと集合完全性 | 処理済み記録や移行RunnerのSUCCESSだけでTargetのEffectを断定しない |

ページ途中にSQL/Provider/Schema/権限/順序が変化したらG2・G3を`block/unknown`へ戻す。安全な再Snapshot/再照合と既処理Operation lookupを行うまで無条件再開しない。0件は「指定Scope・観測時点で可視の条件一致0件」という**限定されたClaim**であり、未取得やRLS外を「不存在」としない。

### 8.2 CDCは削除・取消・履歴訂正も対象

- Snapshot以降の`insert/update/delete`、取消・状態遷移・履歴追加を区別する。Targetの現存行Countが合ってもDelete/Tombstone、過去のHistory、未処理Outboxが欠落していればValidateは不合格。
- Business Keyの合併/分割・改番とPhysical PKの再採番を分ける。Code mapは変換方向・有効時点・記録時点・多重度・Ownerを明示し、曖昧な期間や1対多は`held`を維持する。
- 旧WriterのDeploymentゼロ、観測窓内の旧形式Writeゼロ、対象Writer母集団の退役確認は**異なる証拠**。月次Batch、Queue遅延、直接SQL、旧接続pool、DDL/CDC capture gapはG4/G5の停止条件にする。

### 8.3 Gateの失効行列：再評価は必要な範囲に限定する

| 変化 | 最低限の再判定 | 自動的には失効しないもの／注意 |
| --- | --- | --- |
| SQL Sourceが移動、本文不変・適合した再解決 | Locator/選択Lock、G2/G3の実消費Input証拠 | Business Keyの意味版。現Grantは常に別 |
| SQLの候補が追加／実選択が変更、またはFilter/Scope/Read Schema変更 | G2のCohort、G3の全件性、必要に応じてG4のRead資格 | Query ID同じというだけでは旧実Read結果を再利用不可 |
| 旧Writerが再出現、長周期Batchが未観測 | G4のDrain、G5のContract条件 | G1の過去Schema Observationは履歴として保全 |
| CDC Position欠落、Source A→B→A、Provider/Role/Code Map変更 | 影響するG2/G3/G4の変換・読取と現在資格 | byte一致はOwner受理・現在Grantの自動復活ではない |
| G4後のAck/Commit Unknown、逆同期Receipt欠落 | G4のOperation/Owner/Receipt・独立Read、G5の停止 | ログ成功もRollback完了も推測不可 |

移行処理の進捗、独立Observation、Owner受理、Current Grant、現在のGate資格と失効理由をそれぞれ記録する。`pass/block/unknown`は**提案する診断票の表示値**であり、新Core enumの採用ではない。古いGate PASSを失効させることと実DB処理をRollbackすることを同一視しない。

## 9. 追加反証Fixture MG-13〜19（独立Expected・未実施）

既存MG-01〜12と同じ受注合成Data、Source Snapshot/旧新DB Alias/Code mapを利用する。ExpectedをSQL・移行処理の出力から逆算しない。

| Case | 操作・負例 | 期待される検収判定と既存Owner |
| --- | --- | --- |
| **MG-13** | Backfill 1ページ目は共通SQL、2ページ目でProvider専用SQLへ実選択変更。Query ID/件数は同じ | 同一Cohortで完了扱い不可、G2/G3停止。#2862/#2506 |
| **MG-14** | Offset分割中に先頭付近へ旧Writerが挿入/削除。Sort値が同値 | missing/duplicateを検出しSnapshot/keyset・対象範囲を再確認。#1640/#2506 |
| **MG-15** | Snapshot後のDelete/Cancel/History UpdateをCDCから欠落させ、最終Count/合計は同じ | Validate FAIL、Tombstone/History/CDCで不一致。#2555/#1129 |
| **MG-16** | 旧Code A→新B1/B2が有効期間/記録時点で異なり、改番と訂正が交差 | 勝手に一意変換せず保留、業務Owner判断を要求。#1126/#2555 |
| **MG-17** | 旧Client配備0・直近Write0だが月次Batch/手動SQLの全母集団を未観測 | G5 BLOCK/UNKNOWN。Window・Origin・Ownerの証拠が必要。#2555 |
| **MG-18** | Plan/SQL/SchemaがA→B→Aになり旧G3 PASSとGrantを再利用 | 現在のG4/G5は再資格化が必要。無関係Artifactだけは選択的再利用。#2600/#1163 |
| **MG-19** | Cutover後、旧DB逆同期のAck欠落と新DBの同Logical Command A→B→A・新Key再送 | Operation ID/Receipt/Owner Fence/別ReadのないRollback・再送を拒否。#1174/#2564/#2555 |

**最初の実装圧力**はMG-13/14/15/17を、既存`tools/reference-order-migration-dual-db`のPlan/Checkpoint/CDC観測と、#2506のResult Slice、#2862の選択SQL、#2514のScenario/Expectedで`read-only`に再現できるか確認すること。SQL選択器・第二CDC/Transactionエンジンを先に設計せず、欠けたSource/Scope/Complete Witnessを具体的に列挙する。実Provider・旧実Client・実DBでのMG-13〜19、正式Owner受理は`NOT RUN`。

## 10. 利用者に見える状態と最短の接合

通常のローコード画面は「受注データ：移行準備中／照合待ち／切替可能／旧形式削除は承認待ち」のような業務単位の状態を表示。必要な権限を持つ運用者/開発者だけが、選択SQL、CDC位置、旧Writer、Key Mapping、未解決と再検証理由へ進める。RLS外の件数、内部SQL Sourceや接続秘密を診断から漏らさない（#2516）。

**次の最小垂直スライス**：同一Query/複合業務KeyのChunk Read→選択Source/Provider/Scope・Snapshotの固定→Checkpoint→独立Target Read→CDC catch-up判定。最初はread-onlyの合成反例で偽PASSを防ぎ、実Provider/実旧Client/同一Attemptへの接合は既存#2555/#2622/#1174/#1163 Ownerが各々立証する。全体の製品Gateや意味承認をこの局所PASSへ格上げしない。

