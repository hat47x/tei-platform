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



## 11. 互換性の二次元判定：Application Release × DB State（2026-11-06、他レーン照合による提案）

既存の`application/release.Definition`は個々のApplication Releaseが要求するDB Aliasと`RequiredContracts`を**その版の値としてコピーして保持**する。`database/release.Definition`は`Provides`と`DataReadiness`を保持するが、後者は宣言であって実移行の独立検収ではない。`database/release.StateSnapshot`は`StateFromRelease`と`StateFromObserved`を区別する。外部管理DBの契約状態をread-only投影できても、TEIがMigration履歴を所有・適用したことにはならない。

**Gate検収の二次元評価**は次のとおり。`AppV0`（旧Client）・`AppV1`（新Client）のDB要求を、各時点の`DB State提供契約`・DB Alias/Provider・Read/Writeの実経路と照合する。現行型が契約のRead/Write別提供を直接表現できるとは仮定せず、足りなければ**外部のScenario/観測票に記載**する。新しいDB Release/Contract SSOTは作らない。

| 時点 | AppV0 Read | AppV0 Write | AppV1 Read | AppV1 Write | 判定 |
| --- | --- | --- | --- | --- | --- |
| G1 Expand後 | 旧Contract維持 | 旧形式の安全なWrite維持 | 互換範囲で試験 | Owner認可前は原則無効 | 旧Contractが破損ならG2禁止 |
| G2/G3 並走 | 旧形式と履歴の意味を保持 | Snapshot+CDCに整合する書込み | partial/completeとSnapshot条件を別記録 | Ownerが明示した同期規則が無い限り二重正本不可 | 複数Writerの未解決をG4へ持ち越さない |
| G4 Cutover | read互換が必要なら期限付きで保持 | Fence/Drain、未知Effectをlookup | 現Ownerと新Contractで独立Read | 現Grant/Owner generation/新DB実EffectとReceiptで確認 | 旧Writeが残る／新Write未確認なら切替のqualified PASS不可 |
| G5 Contract | 残す互換Contractの存続条件を明示 | 旧Schemaに依存するWriter=0を母集団・観測窓で証明 | 新DB契約と現Scopeで成立 | 新契約でWrite/履歴保存が成立 | 旧Writeを観測できないことを退役証明にしない |

**必須の互換性述語案**：必要とされた各Client/Release/DB Alias/Operation（Read, Write, Batch, Direct SQL）について、(a)その版の要求ContractとDBの提供Contractが適合、(b)実際のRead/Writeが当該Scope/Providerで成立、(c)必要なデータ移送・履歴と同期の現在資格がある、(d)Owner/Grant条件を満たす。いずれかに必要な観測が欠ければ当該操作は`unknown`。Schema/major版が一致するだけで業務同値としない。古いアプリが動いていない時点の実行ログは「旧アプリ可動保証」にはならない。

### 外部管理DBとMigration History

- **TEI-managed DB**：Canonical Migration Definition→Provider Realization→実行履歴・実DB Schema観測は別。SQL実行の成功だけでは提供契約／Data Readinessの現在性を認定しない。
- **external-observed DB**：`StateSnapshot.Source=observed`、`ObservedAt`、`Provides`を持つが、MigrationをTEIが適用したとは表示しない。外部履歴が取得可能ならFlyway/Liquibase等の**元の履歴・checksum・repair状態・適用時刻・Provider**を来歴付きObservationとして照合する。共通の歴史を書き直さない。履歴不明なら`unknown`、checksumが不一致なら適合確認を止める。
- DB Schemaの観測とMigration Historyが同じように見えても異なるClaim。外部ツールの`repair`や履歴テーブルへの手動変更をTEIが暗黙実行することは無い。
- 同じSchemaが出来上がっていても、元データのBackfill・削除履歴・同期漏れ・追加Constraintの検証段階が違う可能性がある。`StateSnapshot`の提供Contractを、実データ・実Consumer/Ownerの合格へ昇格しない。

## 12. Gate Evidence Packet：必要証拠と現在資格（新Core型ではない）

G0〜G5の判定は、既存`releaseplan database-gate`、Scenario/Oracle、Source/Artifact/Execution Receipt等を**参照する検収票**とする。人間に提示する最小の列案は以下。

| 列 | 意味／必須の区別 |
| --- | --- |
| `obligation / claim / gate` | Schema一致、Data集合完全性、旧Writer互換、Owner切替、履歴、復旧可能性を**別の主張**として扱う |
| `required subject set / scope` | DB Alias/Instance、tenant・業務キー集合・有効時点、Client version/periodic Batch母集団、Provider read/write能力 |
| `source / model / schema / plan revision` | World・Meaning・Actual Schema・SQL Asset/Realization・Migration Plan Digestを別固定 |
| `observer / method / environment` | 実行者と独立のDB接続、Source/Target Reader、CDC/Outbox、Receipt照会、外部履歴の取得元・資格/権限 |
| `snapshot / CDC / attempt` | Snapshot時点とPosition・Result Slice completeness、個別EffectのOperation/Attempt ID、Owner/Fence generationを混同しない |
| `expected / observation` | 事前固定の独立Expectedと観測値。SQL/移行器の出力から期待値を作らない |
| `state / reason / freshness` | `pass / block / unknown`、不一致/観測欠落/時間失効/Owner保留/Unsupportedを区別 |
| `owner decision / authority` | 業務意味、例外/データ保持、切替・破壊的操作の承認と、Hostの現在Grantは別 |
| `remedy / resume boundary` | 修正後に何を再検証し、Checkpointをいつ再使用できるか。実RollbackとGate資格の失効は別 |

**判定規則案**：当該Gateで必要なobligationのすべてが、適切な範囲・現在版・必要な独立性で`pass`、かつOwnerの意思決定とHostの実行権限が別途成立した場合だけ進行候補。明白な不一致は`block`、観測不足・Provider不対応・部分Dataset・不明な正本/CDCなどは`unknown`。未適用のobligationはOwnerが理由と適用範囲を明示して記録した場合のみ`not-applicable`とできる（新enumの採用ではない）。**unknownをPASSとして集計しない**。

全部のGate証拠が物理的に同時刻に採取されることは要求しない。Schema、CDC、Client実行、Owner承認は異なる観測窓で得られるため、**Plan/対象Scope/世代の整合と各証拠の鮮度条件**を照合する。一方、**同一の書込みEffect**についてはOperation/Request hash/Host Grant/Provider/Receipt/別Readを同一論理操作として相関させる。違う実行のPASSを寄せ集めて一つのEffect成功とは主張しない（#1163/#1174）。

### Cutover境界の限定シーケンス（実装ではない）

`Prepare（基線固定）→ Fence旧Writer → Drain/未知Operation照会 → CDC catch-up/独立検証 → Owner承認と新Writer権限の適用 → 新側の実Write/Receipt/別Read → Post-cutover観測`。

旧/新DBをまたぐこの順序を、**分散した単一ACID Transactionだと主張しない**。移管の途中で旧WriterのFence応答が失われた場合、進行を止めて実際のOwner/Grant/旧Write効果を照会する。新Writerが既に有効化された後は「Cutover中止＝自動Rollback」ではなく、実Commitと逆同期の可否を再評価する（#2555/#1174/#2564）。Cutover合格はG5の不可逆なContractを許可するものでもない。

## 13. 追加反例と次の最小実証（MG-20〜23、未実施）

| Case | 故障注入／独立Expected | 停止先・既存Owner |
| --- | --- | --- |
| **MG-20 外部履歴差** | external-observed DBが目的Schema/ProvidedContractを返すが、外部Migration履歴のchecksum不一致・未取得。TEI Migrationを実行済み扱いにしない。履歴不一致時block、取得不可で安全性判断が必要ならunknown | #2622（Provider観測）/#1163（Evidence） |
| **MG-21 新旧Release非対称** | AppV0とAppV1が異なるRequiredContracts。DB Stateは新majorのみ、AppV0 Readが通るがWriteが失敗する。Contract一致表示やDataReadiness宣言だけでG1/G4/G5をPASSにしない | #2555（Client移行）/#2622（Contract） |
| **MG-22 独立観測欠落** | Runner完了・Target 0件・schema一致、Readerはtenant Aのpartialだけ、tenant B未観測。0件を全件0件にせずG3 unknown、外部の必要対象が欠けていることを報告 | #2506/#2514/#1163 |
| **MG-23 Fence応答喪失** | G4で旧Writer停止Ackが消失したまま新Writerが有効化される。旧/新両方の別Read、Owner generationと実Effectを照会するまで切戻し/継続を止める。Cutoverを原子的成功と記録しない | #2555/#1174/#2564 |

**P0実証順**：既存の`application.release.Definition`×`database.release.StateSnapshot`でMG-21を**read-onlyに比較**→同じ合成FixtureにMG-22の部分Readerを追加→MG-20の外部履歴・観測のない状態を`unknown`で表現→MG-23のFence/Unknownは既存実PostgreSQL参照経路に接続するまで**未検証**。実SQLや新Providerは追加せず、どのobligationが既存Scenario/Evidenceで記録可能か、まだ存在しない観測を列挙する。

**採用を保留**：正式なMigration Evidence Schema、新しい二者間同期Tx API、外部Migration Historyの正規化型、DB別共通互換性ポリシー、業務Ownerの切替承認を自動化する仕組み。本書は現行コードに存在する契約を**証明可能な範囲で再利用する設計候補**に限る。

## 14. Cutoverの安全停止・不可逆効果・保存義務の最小ゲート（2026-11-06）

> **追加の決定候補**：Cutoverを一つの成功/失敗ではなく、旧Writer停止後も**書込みを保留して実状態を独立照会できる**検収手順として扱う。中間状態名は検収票の説明用で、新しいReleasePlan Phase、Migration Coordinator、固定Core enumではない。既存#2555 MIG-H01〜07、#1174、#2516・SHOの責務に従う。

### 14.1 Safe pauseの条件

| 旧Writer側 | CDC・未完Operation | 新Writer側 | Gateの期待 |
| --- | --- | --- | --- |
| 停止/Fenceが未確認 | 任意 | 任意 | 新Writer有効化による成功判定は禁止。旧Owner・実Effectを照会しG4保留 |
| 停止/Fenceを実確認 | gap、遅延、unknown | 新側未開始 | 書込み保留を候補にする。必要ReadのScope/鮮度/Grantも不明ならRead停止 |
| 停止/Fenceを実確認 | 収束を独立確認 | 新Grantまたは実Provider資格が不明 | 旧Writerを勝手に再開せず、新Writerも起動しない。G4 unknown |
| 停止/Fenceを実確認 | 収束を独立確認 | 同じOwner世代・現Grant・実Commit/Receipt/別Readを確認 | G4の必要Claimを検収する候補。Consumer・業務Owner承認は別 |
| 旧Queue/Offline Writerの再流入、Fence期限切れ | 過去の収束証拠あり | 新側稼働中 | split-brain疑い。両側の実Effect・Owner世代の再照会とG4再資格化が必要 |

「書込み保留」は停止しても業務が許容する期間と連絡方法について**業務/運用Ownerの判断**が必要。Read-onlyに縮退したことを自動的な安全性の証拠にしない。旧Writerを再有効化する前には新側の未知Commit、逆同期、権限、Consumerへの公表済みEffectを評価する。旧DBと新DB間でこの順序が原子的とは主張しない（#2146/#2555/#1174）。

### 14.2 Rollbackの上限はDBのRowset差分とは異なる

新DBから外部API・通知・在庫引当・請求、Outbox公開、SHO固定文書・他Consumerへ展開された結果は、旧新DBの行数・金額が一致しても自動的には取り消されない。

- **DB recovery**：旧新Owner/Fence、Business Key/物理PK/履歴、Reverse Outbox・CDC、Operation ID/Receiptと別Readを照合。
- **External effect recovery**：業務取消/補償・配送済み/unknown・外部受取Receiptと現在Grantを確認。Rollback SQL＝外部Effect取消とはしない（#1129/#1174）。
- **Consumer qualification**：旧時点の引用・納品物は歴史的根拠として保持し、現在の正本を説明する利用ではSource/Read Cohort/Grantを再確認（#2516／SHO #4/#6）。
- **Retention**：過去時点再現、監査、訂正履歴、保存期間・開示許可のOwner判断が必要。旧Readerが退役したことから旧物理値の破棄可能性は導かない。

したがってContractは①**論理的な旧形式退役**と②**物理的な旧列・表・Code Map削除**を独立した判断とし、①の成立を②への自動許可にしない。保管先は旧DBでなくてもよいが、独立Readerで過去の業務Claimを再構成でき、保存/閲覧資格を維持できる必要がある。

### 14.3 最小反証MG-24〜28（新しいテストの実行なし）

| Case | 故障注入 | 独立Expected／担当 |
| --- | --- | --- |
| **MG-24 Safe Pause** | 旧Writer Fence成功・CDC収束、しかし新Writer Grantが未確認 | 旧再開・新書込みを両方禁止。許されたReadだけを限定し、G4 unknown。#2555/#1174 |
| **MG-25 Offline Writer再流入** | Cutover後に古いクライアントのQueueが送信、Fence世代/接続Poolは旧版 | Provider側で効果ゼロを別接続観測できるまでG4再資格化。アプリ停止ログだけは不足。#2555/#2146/#1174 |
| **MG-26 外部Effect後の切戻し** | 新DB Commitと外部通知済み、逆同期でDB行は一致 | DB切戻し成功≠業務効果の巻戻し。外部Receipt/補償/Consumerの確認まで未完。#1129/#1174/#2516 |
| **MG-27 過去時点Readerと破棄** | 旧Writerゼロでも過去受注・固定SHO引用が旧Schema/変換表を必要とする | 物理削除の停止、または同等の履歴再構成/Retention資格を別Readerで確認。#1126/#2555/SHO #6 |
| **MG-28 PASS後のContract変更** | G3合格後に旧Application RequiredContractsかDB Provider版/Read Policyを変更 | 影響のあるG3/G4/G5のみ再検証。旧World FP/CI成功を現在資格へ復活させない。#2600/#1163/#2622 |

### 14.4 実証の優先順と採用を保留する判断

**P0**：MG-24を既存#2555のOwner/Writer観測・#2147のCheckpoint/CDC・#1174の現Grantの**合成read-only**検収票で表現可能か確かめる。`12のGate Evidence Packetで `old-fenced=true / cdc-caught-up=true / new-grant=unknown` に対してG4を `unknown`、新旧とも新規Write資格なしとする。新状態型やRunnerを追加しない。

**P1**：MG-25/26は実Providerと外部Effect Consumerで旧世代拒否・別Read/Receipt/補償の状態を実測するまで未成立とする。**P2**：MG-27で過去時点Reader・Retention、MG-28で必要Claimのみ再資格化。各FixtureのExpectedは実Migrator/生成SQLの出力ではなく、固定業務Key・時点・操作履歴から独立に作る。

**Ownerへの判断事項**：①旧Writer Fence後の許容停止時間、②保留中に許せるReadの時点/対象範囲、③切戻し完了をDB行だけか外部効果も含めるか、④旧コード表・過去履歴の保存期間・閲覧権限、⑤旧Writer/Offline Queueの捕捉不能時にどこまで停止を継続するか。これらは既存Issue Ownerに残し、正式Gate/ADR・製品API化は別の受理後に判断する。

**資格の上限**：本節は独立Expectedと判断の候補。旧#2143/#2147の3 PostgreSQL参照、#2564実PG局所CAS、#2516の人工FixtureをMG-24〜28の実Provider PASSに合算しない。新しい実DB/本番Client/SHO Reader試験・正式Owner受理は**NOT RUN**。

