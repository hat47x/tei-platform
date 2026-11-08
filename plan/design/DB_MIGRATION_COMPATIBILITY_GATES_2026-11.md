# TEI DB移行互換性ゲート：横断統合と最小受入（2026-11-06）

- **状態：検討案。** 本書によって正式Gate、公開API/DSL、Owner判断を採用しない。本書の追補は設計・受入条件である。Draft PR全体には既存Release契約の構造的なGo特性テスト（MG-21）も含むが、実SQL/実DB Migration、実旧新Client共存、異種DB・本番環境、Owner受理は未実施。
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

**P0の順序**：まず§13のMG-21/22で旧新RequiredContractsと部分Readerをread-onlyに確認し、続いてMG-24を既存#2555のOwner/Writer観測・#2147のCheckpoint/CDC・#1174の現Grantの**合成read-only**検収票で表現可能か確かめる。`12のGate Evidence Packetで `old-fenced=true / cdc-caught-up=true / new-grant=unknown` に対してG4を `unknown`、新旧とも新規Write資格なしとする。新状態型やRunnerを追加しない。

**P1**：MG-25/26は実Providerと外部Effect Consumerで旧世代拒否・別Read/Receipt/補償の状態を実測するまで未成立とする。**P2**：MG-27で過去時点Reader・Retention、MG-28で必要Claimのみ再資格化。各FixtureのExpectedは実Migrator/生成SQLの出力ではなく、固定業務Key・時点・操作履歴から独立に作る。

**Ownerへの判断事項**：①旧Writer Fence後の許容停止時間、②保留中に許せるReadの時点/対象範囲、③切戻し完了をDB行だけか外部効果も含めるか、④旧コード表・過去履歴の保存期間・閲覧権限、⑤旧Writer/Offline Queueの捕捉不能時にどこまで停止を継続するか。これらは既存Issue Ownerに残し、正式Gate/ADR・製品API化は別の受理後に判断する。

**資格の上限**：本節は独立Expectedと判断の候補。旧#2143/#2147の3 PostgreSQL参照、#2564実PG局所CAS、#2516の人工FixtureをMG-24〜28の実Provider PASSに合算しない。新しい実DB/本番Client/SHO Reader試験・正式Owner受理は**NOT RUN**。



## 15. 既存実装のP0特性試験と残存する偽PASSの経路（2026-11-06、Source照合・追加テスト・まだGate未合格）

**実装済みの既存構造を再利用し、移行製品Gateとして何が不足しているかを静的コード・狭い特性テストで確定する。** 新しいCompatibility Evaluator、Provider/Host/Migration Engineは追加しない。

### 15.1 現行コードで証明できる範囲

| 既存実装 | 確認した性質 | G0〜G5に残る判定不能 |
| --- | --- | --- |
| `internal/application/release/model.go` | `Definition`がDatabase RequirementをRelease版内に固定、`AsApplication()`で既存Compatibilityへ投影 | Active Client/Batch/Direct SQLの**完全な母集団**を列挙できたか、実Clientがその版で稼働したか |
| `internal/database/release/compatibility.go` | `CheckApplication`はContract有無、Major family一致、宣言済み`DataReadiness`を判断。Major 4≠5 | `DatabaseRequirement.Access`のRead/Write別**実能力・実効果を検査していない**。実Schema Read、RLS、History、Current Grant、Source freshnessも不明 |
| `internal/database/release/model.go` | DB Release由来とexternal-observed由来を`StateSnapshot.Source`で区別 | `StateFromObserved`の`ObservedAt`が古い／欠ける場合の鮮度判断とDataReadyの独立裏付け。宣言を実データ観測とみなさない |
| `internal/database/migration/history.go` | `HistorySnapshot`と`CompareHistory`で`applied/pending/failed/diverged/orphan-applied`を出力 | 取得が失敗/部分のHistoryを完全な空集合と区別する必要がある。**現在の`AppliedUnknown`がchecksum一致等で`RepoApplied`へ分類され得る**。現行`CompareHistory`はHistoryのCapturedAt/取得完全性を検証しない |
| `internal/database/migration/postgres_apply.go` | 適用直前にActual Schemaを観測し、planned SQLとfresh renderingを照合 | SQL実行後の別接続Schema/Data、migration history、Client Read/Write、Owner/Cutover/Receiptを確認しない |

**重要なソース静的負例**：現行`CompareHistory`の分岐は、`AppliedFailed`を特別扱いし、checksum違いを`diverged`とする一方、`AppliedUnknown`を`applied`から明示的に排除していない。実Historyの結果不明なのに`RepoApplied`が出ればG1/G3/Contractの偽PASSにつながり得る。この境界は正式な`RepositoryState`の挙動／不明時の診断方法についてOwner判断を要する。この偽PASSは[#2892](https://github.com/hat47x/tei-platform/issues/2892)に独立した再現負例・修正方針として切り出した。**本PRでは製品型/比較関数の仕様を無断で変更しない**。

別の負例は、Historyを取得できなかった際に空の`HistorySnapshot`を構築すると、予定されている全Migrationが`RepoPending`に見えること。正しい「取得成功・適用件数ゼロ」と「観測失敗」を区別できる取得資格が必要。単一`CompareHistory`の出力をHistory全体の完全性証拠にしない。

### 15.2 今回追加した実Go特性テスト（構造のみ）

`internal/database/release/migration_gate_characterization_test.go`に以下の2件を追加した。`go test ./internal/database/release`のexact-head CI完了が確認されるまでは**テスト実行の成功未判定**。

- `TestMG21OldAndNewApplicationReleasesDoNotSilentlyCrossMajor`：同じ業務Applicationに`v1/major4`と`v2/major5`のRelease要求を固定し、DB提供major4/major5の両状態で`Compatible/Blocked`の4通りを独立期待で比較する。**1契約IDのmajor切替では旧新が自動的に共存するわけではない**ことを示す。
- `TestMG21CompatibilityDoesNotDistinguishReadFromWriteEffect`：`AccessRead`と`AccessWrite`を指定した場合に、現行`CheckApplication`の構造判定が双方`Compatible`を返すことを特徴付ける。これは**実Read/Writeが成功した証拠ではなく、互換性判定器がAccess別Effectを証明していないという制約の証拠**。

旧新を本当に同時提供する実現（別Contract ID、互換View、Writer Adapter、段階的変換等）は案ごとに対応するSchema/Role/Provider実能力と独立Expectedを要求する。Major互換の偽装、単一DB Snapshotに複数Majorを重複登録して暗黙の最後勝ちを利用する方式は禁止候補。実装必要性を#2555/#2622のOwnerで比較する。

### 15.3 Gateへの引渡しと追加の停止条件（Case IDを増殖させない）

- **MG-21**は上記構造特性まで部分的に実証可能。実AppV0/V1とDBの同一期間共存、Read vs Writeの実効果、現在のGrant、旧Versionの実利用者母集団は未実証のまま保持する。
- **MG-22**は`CheckApplication`だけで検証不能。Result Slice（未取得／部分／ページ／全件）とtenant・時点別の必要集合は#2506/1706、実Provider ReaderとScenario Expectedは#2622/#2514に引き継ぎ、観測できないときはG3 `unknown`。
- **MG-20**は`CompareHistory`で`unknown→applied`になり得ることをソース上の検証停止条件として記録。まず既存#2555/#2622の責務と調整し、`AppliedUnknown`・空checksum・空/部分History Snapshot・diverged/orphan/failedの負例をfocused Go Testへ分解する。対策候補（エラー停止か`RepoUnknown`の明示）は製品型・呼び出し側影響を比較してOwnerが決める。**実装済み修正とは扱わない**。
- **G5 Contract**では、`CheckApplication`が個々のAppでCompatibleでも、**把握していない旧App/Batches/Direct SQLの母集団が残る限りunknown**。現在の提供契約とRelease Requirementsだけでは未稼働Clientが無いことを証明できない。古いReleaseを消す前に全Active/潜伏Consumerの退役証拠が必要。
- **G4 Cutover**では、上流のstatic compatibility PASSを、現在Owner/Grant・旧Writer Fence・実Update/Receipt・別Readへ格上げしない。`DataReadiness=ready`は移行完了の宣言ではなく、独立に裏付けられる対象範囲・時点を持つ必要がある。

### 15.4 最短の次の実証

1. 追加した`MG-21`構造テストのexact-head CIを確認し、failなら既存契約/Goモデルに沿って修正、passでも限定的結果として記録。
2. `MG-20`の現在の偽PASS境界に、独立Expected`unknown never qualified as applied`のfocused負例を用意する。修正/正式API追加は別の小DraftとOwner判断に委ねる。
3. `MG-22`の部分Scope/0件/未取得を`Result Slice`で分離し、旧新Applicationの同じ要求集合と同じSource/DB Snapshotで比較する。SQL/Provider実行のないfixtureではG3をPASSにしない。
4. #2555の旧新Client母集団と#1174の書込みEffect/Current Grantを接続し、Owner承認のG4/G5資格はその後の実DB/実Consumer観測で判定。

**採用保留**：Migration Historyの新enum/外部Adapter契約、同一Contract IDの複数Major同時提供、実Client Registryの制度化、Owner承認自動化。必要な具体的な失敗ケースで追加面を決める。


## 16. Gate観測の一意性・必要検査の完全性（2026-11-06）

> 状態：現行コードの静的確認および小さなGo**特性テスト**で判別する、採用前の修正対象。公開Schema/Runtime契約・Migration Gate・Owner承認を新設しない。構造特性テストは**望ましい修正後のPASSを保証するものではない**。

### 16.1 同じDB IDのStateSnapshotは、入力順で上書きされる

現行 `internal/database/release/compatibility.go` の `CheckApplication` は、入力 `[]StateSnapshot` を `map[DatabaseID]StateSnapshot` に取り込み、同じ `Database` が複数含まれると**後の項目で上書き**する。`ObservedAt` や `Source` を用いて現在の観測世代・実Database Instance・SnapshotのAuthorityを決めていない。

このため、旧DB状態（契約Major=4）と新DB状態（Major=5）を同じDatabase IDで受け渡した場合、旧→新の順と新→旧の順で `Compatible/Blocked` が変わり得る。入力の順番は「どの観測が現在の正本か」の根拠にならない。さらに `StateFromObserved` の `ObservedAt` が空欄/古い値でも、提供契約と `DataReadiness=ready` が一致すれば**構造的には `Compatible` と判定**され得る。

**提案する停止基準**：
1. 入力に同じDatabase IDの複数Snapshotがあり、どのEnvironment Binding/Instance・世代・Source・観測区間を選ぶか明示されていない場合、Gateは `unknown`。暗黙のlast-wins/最も新しそうな時刻を採用しない。明示選択と観測資格の検査責務は#2622/#2611の既存Binding/Providerへ返す。
2. 履歴管理DBと `external-observed` DBは、移行履歴の取得完全性と独立Sourceを別に検査。`StateSnapshot` の `Provides/DataReadiness` を、#2892の `CompareHistory` unknown/applied未修正問題の代替証拠にしない。
3. `CheckApplication Compatible` の意味は**宣言されたMajorとReadinessに関する局所的な構造適合**。現行Provider read/write、全Read Cohort、CDC/Checkpoint、Owner/Grant/Fence、旧新Client実行を保証しない。

#### MG-29（現状の不安全な挙動を特徴付けるGo試験、実DBなし）

`internal/database/release/migration_gate_observation_order_characterization_test.go` の2テストで、(a)同じDB IDの旧新Snapshotを入れ替えるだけで互換判定が反転する、(b) `external-observed` の観測日時が欠ける/古くても構造適合結果が同じ、という**現在のギャップ**を再現する。

**望ましい受入**：明示した1つのDB/世代/Sourceに解決できる場合だけ構造互換をGate Evidence Packetへ渡し、競合・Scope不足・古い観測は `unknown` として次の段階へ進めない。現状を固定する特性テストと、将来の適合修正に対するExpectedは**別の証拠**。修正後は古いlast-winsのExpectedを残さず、拒否・診断Expectedへ変更する。Release機能そのものの改修には#2622とRelease Ownerの承認を要する。

### 16.2 必要検査の完全性と実際の実施を分ける

他レーンのRunPlan/Guard実証（#2882/ #1174）は、**必要なGuardがすべて供給されたこと**と、渡されたGuardが成功したことが別である点を示す。この考え方をMigration `database-gate` の検収票へ再利用する。

| 観測欄 | 独立させる理由 |
| --- | --- |
| `required obligations` | 対象Client/Writer・Read Scope・CDC/History・Provider/Current Grant・Retentionのうち、そのGateに必要な全対象集合。Owner/契約から導き、実行された観測の件数から逆算しない |
| `supplied evidence` | 取得できたSchema、旧新Client、History/CDC、現在Grant・Receipt・Consumerの証拠。供給されていない検査は `pass` ではない |
| `evaluated/observed` | 検査器の起動/Goテスト成功と、実Provider・Clientが観測されたことを区別。合成Runnerの返した `resolved` は実Effectではない |
| `independent qualification` | Expectedとの比較、実対象・Scope/時点、取得の完全性、Source/Provider版、Owner判断、現在のHost Grantを別資格化 |
| `undischarged / remedy` | 必須対象の欠落、結果不明、失効、止まった箇所と、再Snapshot/旧Writer Fence/Effect Lookup/Owner受理などの再開条件 |

**ゲート進行の最低条件案**：`required` の全要素について、適切な独立Expected・現在のScope/Source/Providerに基づく適格な `observed` を得るか、Ownerが非適用を明示承認していること。`supplied⊂required` や `evaluated=pass` だけでは進行不可。部分実行が失敗/停止した場合も、**それ以前に完了したChunk/DB Commit/外部Effectは残り得る**ため、Gateの `block` を「全操作no-effect」へ翻訳しない（#1174/#1129）。

G1〜G5で各観測は取得時点が異なる。要求するのは**同一瞬間**ではなく、各Claimの必要な鮮度・Source/Plan/Provider/Read Cohortの整合。個々のEffectだけは同一Operation ID/Attemptに基づき、実Commit/Receipt/独立Readを相関させる（#1163）。

### 16.3 実施順とIssueへの引継ぎ

- **先行**：現MG-21の旧新Release互換Goテストと、今回のMG-29で `CheckApplication` の構造的限界を可視化。MG-20外部履歴は[#2892](https://github.com/hat47x/tei-platform/issues/2892)で修正責務を持つため、本計画で第二History Engineを作らない。
- **P0 read-only**：同じDB IDで異なるStateSnapshot、観測年月が欠ける状態、旧新Provider/DB Aliasが競合する状態を使い、MG-21/22/29の **required/supplied/evaluated/observed/qualified** 表を作る。見かけの `Compatible` がGate PASSにならないことを確認。
- **P1 Provider**：#2622/#2611が正しい実Target Instance/Read/Write資格と明示Snapshot選択/重複診断を決める。局所的な契約検査の成功を実DBの現在資格へ拡大しない。
- **P2 Cutover**：#2555/#1174のMG-24 Safe Pauseに必要な**旧Writer Fence/CDC/新Grant/Unknown Lookupの全必須集合**を列挙し、供給不足は書込み保留（両側への無条件再送禁止）。#1163/#2514が独立Expectedと同一Attempt証拠の確認。
- **P3 Contract**：旧Writer/Batch/SHO・外部Consumer/History/Retentionの全要件をOwnerと再確認。値が一致しても必要Reader/保存義務が残る限り物理削除不可。

**状態**：MG-29のGoテストは追加したが、実DB Migration/旧新Client/Fence/CDC/外部History・正式Owner受理は**NOT RUN**。現挙動を示すテストの成功は、望ましい安全性の成立ではない。#2892の修正は未完了であり、本書で修正済みとしない。


## 17. 互換性検査の二重last-wins防止と修正順（2026-11-06）

**新たに確認したコード上の事実**：`CheckApplication` は（a）`[]StateSnapshot`→`map[DatabaseID]StateSnapshot`と（b）`StateSnapshot.Provides`→`map[ContractID]ProvidedContract`という**2箇所で重複値を後勝ち**にする。つまり同じDB IDの旧新Snapshotだけでなく、一つのDB状態の中で同じContract IDに異なるMajorが2つあれば、入力順だけで `Compatible/Blocked` が反転し得る。`MG-29`はこの**二重の曖昧性**を検収対象にする。

- **修正Owner／Issue**：[**#2897**](https://github.com/hat47x/tei-platform/issues/2897) はRelease互換検査の同一DB・同一Provided Contract重複をfail-closedにする。既存 `CheckApplication` の構造判定の責務を維持し、Source/Provider/Instance/時点・現在Grantの自動推測を追加しない。
- **区別する別Issue**：[**#2892**](https://github.com/hat47x/tei-platform/issues/2892) は `migration.CompareHistory` の `AppliedUnknown`→偽`RepoApplied`／履歴取得不能→偽`Pending` 問題を受け持つ。二つを混ぜて新しいHistory/Gate SSOTを作らない。
- **一時的な上限**：`MG-29a/b`のGo特性テストは現行の順序依存という**望ましくない現状**を固定している。これらがGo CIで成功しても、安全修正は未完成。修正用の小PRが同じGitツリーへ入る際、特性テストのExpectedを望ましい`Unknown/入力拒否`へ**必ず更新**し、後勝ち期待のまま残さない。`MG-29c`の古い`ObservedAt`での構造 `Compatible` も、**現在Gate資格が得られた**との主張にはしない。
- **Provider差の扱い**：一つのDBが2 Majorを同時に提供する仕様は現行 `map[ContractID]` では表現されず、値をまとめて一つのMajorにしてはならない。併存が必要なら、Provider/Release Ownerが新しい明示的契約を別途設計・受理するまでは`Unknown/不正入力`とする。旧新Applicationが別Contract Majorを必要とする場合、現在のStateSnapshotを単純に繋いで両方成立とは推測しない。

### 17.1 修正優先順位と独立Expected

| 順序 | 最小実装／確認 | 採用に必要な証拠 |
| --- | --- | --- |
| P0-1 | `#2897`：DB Snapshot重複／ProvidedContract重複を検出して非黙示・順序不変の`Unknown`またはエラーにする | MG-29a/bの順序を入替えても安全な同一結果。既存の単一DB State/複数別DBの正常動作は退行しない |
| P0-2 | `#2892`：History unknown/partialと真正の成功/空を分離 | HIST-01/03（偽`RepoApplied/Pending`拒否）、HIST-04の既存分類維持。実Provider Adapterは別証拠 |
| P0-3 | `MG-21/22/29`：現在のDB Snapshot/Read Scopeと必要Obligationを紐付ける | 同一Query/Stateの構造互換成功、実Reader/Writerが未観測ならG3/G4に`unknown`を維持 |
| P1 | `#2555/#1174`：MG-24 Safe Pauseと実Provider Effectの対象集合を結ぶ | old Writer Fence、CDC、new Grant、Commit/Receipt/別Readを**同一Logical Operation**・必要な観測区間で確認。非観測は中止・保留 |
| P2 | `#1126/#2555`：MG-27の履歴Retentionと旧形式退役/物理削除のOwner判断 | 実過去時点Reader/記録/Code Map再現、外部Consumer契約の保存。旧WriterゼロだけではContract通過不可 |

**採用留保**：重複入力をエラーにするか `Unknown`にするか、重複した全く同一Snapshotの重複排除を許すか、複数Major同時提供の新正式構文、現在のSource/GrantのAPI表現。いずれも#2897と必要なOwner審査で定める。データが**複数観測できたこと自体**は現在の正本を選ぶ根拠ではない。

**現状**：MG-29a/b/cのうち二つの順序依存と古い観測日時のGapをGoで特徴付け（同一DB、同一Contract、観測日時の計3テスト関数）。修正後の拒否判定、History Adapter、実DB、旧Client、Cutover、Owner受理は未実行・未受理。`#2897`の課題登録は実施済み。



## 18. 移行Gateの採否を決める最小の証拠閉包と修正合流順（2026-11-06、跨るIssueの受入条件／未採用）

これまでのMG-01〜29は故障注入の**候補集合**であり、この先はCaseを増やすことより**何を必要証拠と認め、どの既存Issueが修正したら次の段階へ進めるか**を厳密化する。

### 18.1 検収の出発点は「列挙された観測」ではなく「必要な対象全集合」

移行の対象としてOwnerが宣言・受理した`Client Release / Batch / Direct SQL / Function / Database Alias / Operation`の集合を基準に、対象ごとの`Required Contract / Business Key / Temporal Scope / Current Grant / Result Completeness / CDC・Checkpoint / Effect Receipt / Retention`を**必要Obligation**として導出する。実行器のログや取得できた観測結果からRequiredの母集団を逆算しない。

`CheckApplication`が空の`app.Databases`に対して`Overall=Compatible`かつ`Checks=[]`を返すことは、当該Applicationに申告された契約の**空集合への構造的照合**が完了しただけである。「影響を受ける旧Clientや外部Writerが存在しない」という**母集団の完結性**を証明しない。業務上本当にDB利用者がいない場合は、Ownerが閉じた対象母集団・必要な非適用条件を示せば合法な空集合を扱える。**未知の集合と正当に空である集合を区別**する。

| Proof item | 最低限の判定 | 別の証明として残るもの |
| --- | --- | --- |
| Required closure | 対象Client/Writer/Contract/Scopeと対象範囲が、事前固定版として全件列挙できたか | Source/Schema/DB観測の実行結果 |
| Input/selection uniqueness | 同一DB State/Contractの重複拒否、DB Alias→Instance・Provider・Role/SQL Sourceの明示解決 | 現在のProvider Grant/実DB Read |
| Observation completeness | Requiredごとに独立Reader/History/CDC/Client実利用/Receiptの観測が供給され、欠けていないか | 観測した値がExpectedへ一致するか |
| Meaning and freshness | Independent Expected・業務Key/履歴時点・Source/Plan/Provider版・Scope/水位が適合するか | Ownerが業務意味・移管・Retain/Destroyを受理したか |
| Effect qualification | 実際のLogical Operation・Target/Owner・Receipt/別Readが矛盾しないか | 旧Writer全件停止、外部API/通知などの別EffectとConsumer受領 |

**Gate進行の候補述語**：`required closure known ∧ requiredごとのqualified evidence complete ∧ independent expected consistent ∧ freshness/selection valid ∧ relevant Owner decision accepted ∧ current Host authority qualified`。どれかが未知なら`unknown`、明確な競合/不適合は`block`、すべて揃ったときに初めて`pass候補`。`CheckApplication=Compatible`、Migration History=applied、DB Schema一致は**個別obligationの局所的な根拠**であり、Gateの結論にはならない。実操作の実行許可はこの検収票とは別のHost enforcementを必須とする。

### 18.2 最小の統合Expected（既存MGを再利用）

| 既存Case | 入力の変化 | 最低限のExpected | Owner |
| --- | --- | --- | --- |
| **MG-29a/b** | DBまたは提供Contractの重複を順序交換 | 双方とも`Unknown/入力拒否`で順序不変。別の正常なDBについて不要な偽失敗を起こさない | #2897 |
| **MG-20 / HIST-01/03** | `AppliedUnknown`／履歴取得失敗・部分のままchecksum/Schemaが一致 | 履歴成功に昇格しない。空履歴の「正常取得0件」と取得失敗を分離 | #2892 |
| **MG-21** | 旧/新Appが異なるMajorを要求し、同時にWriterも必要 | 構造互換の一致・不一致は判定可能。ただし実Read/Write・旧Client継続は別の`unknown` | #2555/#2622 |
| **MG-22** | Tenant Aだけのpartial Read／Tenant B未観測で実Row Count=0 | 必要集合のclosure不足でG3通過不可。0件を全件不存在と推測しない | #2506/#2514 |
| **MG-24 / MIG-H02** | 旧WriterのFence確認後、新Grantが未供給／旧Batchが再流入 | Safe PauseでG4 unknown、両側への無条件新Write禁止。実Provider/Owner再照会 | #2555/#1174 |
| **MG-27** | 旧Writerゼロだが過去時点のReader・Retention要件が残る | 論理形式の退役は候補、旧物理データの破棄はG5 block/unknown | #1126/#2555 |

注：`unknown`は技術的に不可能であることを意味しない。証拠が未供給・未観測・失効・対象不明であれば`unknown`、確定した不適合／Ownerによる拒否等は`block`。旧Client互換チェックだけが`pass`でも他Requiredが不明ならGate全体は未通過。既存テストで**危険なlast-winsをそのままExpectedにした特性試験**は修正前の障害再現であり、安全仕様の受入試験ではない。

### 18.3 変更を衝突させない最短実装・合流順

1. **#2897のRelease局所修正**：既存`CheckApplication`の同一DB Snapshot／同一ProvidedContract重複をfail-closedにし、現行`migration_gate_observation_order_characterization_test.go`にある危険なlast-winsのExpectedを**修正と同じ統合単位で安全Expectedへ置換**。入力順、別DBの独立結果、Major切替、空/欠落の正常ケースに対する回帰を重点にする。観測のSource/時間/Grant選択を比較器に勝手に追加しない。
2. **#2892のHistory局所修正**：`AppliedUnknown`・適用記録のchecksum欠落・取得失敗/部分と正常空履歴を独立Expectedで比較。`RepoUnknown`を新設するかfail-closed ErrorかはOwner決定。Provider History Adapterの取得資格が未確認なら履歴の`RepoApplied`だけでGateを進めない。
3. **#2555＋#2506のRead closure**：MG-21/22のRequired集合に旧Reader/Writer/Batchと移行対象の業務キー・時点・Snapshot/CDCを束縛。各Readerの返値が正しくても対象集合を網羅しなければG3 unknown。宣言済みClientの集合を全稼働者集合とみなさない。
4. **#1174＋#1163のEffect closure**：MG-24の旧Writer Fence/CDC、新Grant、同一Operation ID・Commit/Receipt/別Readと外部Effectを別Claimsで照合。安全停止時の`block/unknown`は、**すでに起きたDB Commit/通知/外部Effectを取消したことではない**。
5. **#1119と各Ownerの最終Gate判断**：個別Unit/Fixture→Provider実DB→旧新Client/Batch→Consumer/SHO→Owner Meaning/Authorityの証拠を区別して採用判定。既存PostgreSQL参照合格を新しい異種DB/本番受理へ流用しない。

**PR/変更境界**：Draft #2887は計画と現実の不安全な挙動の**特性化**を所有する。安全修正は#2897と#2892の別責務としてレビューし、両者の安全修正が合流した際に既存characterization testのExpectedを同時更新する。#2887に新Gate Engine/Owner受理機構を導入しない。GoのCI成功は、該当テストが**どの期待（不安全な現挙動か、安全仕様か）**を確かめたのかまで明記する。

**検証状態**：本節の統合Proof/実Reader/実DB/実Client/外部EffectテストはNOT RUN。既存Go特性テストの存在は安全仕様の実装/受理を意味しない。

### 18.4 #2897の局所修正Draftを検収順へ接続（2026-11-06）

[Draft **#2900**](https://github.com/hat47x/tei-platform/pull/2900)を、[#2897](https://github.com/hat47x/tei-platform/issues/2897)の**最小コード修正**として別branchから作成。現 `CheckApplication` の同一DB StateSnapshotと、同一StateのProvidedContractの2つのlast-winsを、重複時に `Unknown` と理由を返すfail-closed判定へ変更する。従来の `CompatibilityState` の公開値は増やさず、上流のInstance/ObservedAt/Sourceや現Grantの暗黙選択は行わない。

#2900には安全なExpectedを持つ4つのGoテスト関数を追加：逆順/同一値の重複DB、逆順/同一値の重複Contract、別DB正常結果への非干渉、単一DB正常/異Major/状態欠落/未readyの退行防止。**このPRが存在することを修正完了や現mainの採用済み仕様とみなさない**。exact-head CIを確認し、Owner/Review判断後に統合可否を決定する。

**合流停止線**：計画Draft #2887の `migration_gate_observation_order_characterization_test.go` は、旧実装のlast-winsという危険な挙動をExpectedにしている。安全修正#2900が先に統合された場合、#2887をmergeする前に**その3関数の期待値をUnknown/順序不変に更新**し、更新後のexact head Goテスト・基準Source/Planに沿った回帰を再実施する。#2900の局所PASSと#2887の歴史的特性PASSを一つの安全性合格へ加算しない。

履歴の偽RepoApplied問題は独立の[#2892](https://github.com/hat47x/tei-platform/issues/2892)に残し、DB Releaseの構造的Compatibleが Migration History/実DB Data/Client compatibility/現在のOwner Grant/Effect/Consumerの合格を意味しない境界は変えない。**本件の実DB Migration、異種DB、旧新実Client、Cutover/Contract実操作とOwner受理はNOT RUN**。



## 19. 局所修正PRの合流ゲートと偽PASS再発防止（2026-11-06、#2900/#2892/#2887の最新分担）

### 19.1 実装進捗を分離する

- [Draft #2900](https://github.com/hat47x/tei-platform/pull/2900) は[#2897](https://github.com/hat47x/tei-platform/issues/2897)の**重複Database State・重複ProvidedContractのlast-wins**を、既存`CheckApplication`の`Unknown`へfail-closedさせる局所Go修正および4つの受入テストを追加した。修正は同PR内の実装で、**main採用・統合した実DB互換・Owner受理には未到達**。
- [Draft #2887](https://github.com/hat47x/tei-platform/pull/2887) は業務/Client/Provider/Cutoverを含むG0〜G5の計画と、MG-21/29の**現行挙動を把握する特性テスト**を所有する。`migration_gate_observation_order_characterization_test.go`のMG-29a/bは意図的に**危険な後勝ち**を期待している。#2900の安全修正と両立しないため、修正採用と同じ統合段階で**安全な独立Expectedへ置換**し、現行の危険挙動のテストを恒久仕様にしない。
- [#2892](https://github.com/hat47x/tei-platform/issues/2892) は`CompareHistory`の`AppliedUnknown`→偽`RepoApplied`、History取得失敗/部分→偽`RepoPending`の修正Owner。**現時点で修正を検証したとは主張しない**。DB Releaseの構造判定修正がMigration Historyの資格を自動修正することはない。

### 19.2 推奨する合流・修正順（正式なmerge実行はOwner判断）

| 手順 | 必要な変更／確認 | 避けるべき誤り |
| --- | --- | --- |
| **A. #2900単独** | MG-29a/bの安全Expected、重複を拒否する`Unknown`診断、正常な単一Stateと他DB結果の非退行を同一PR exact-headで確認 | #2900の局所CI成功を#2887のG0〜G5や現在DB/Client互換の合格に数えない |
| **B. #2887への統合差分** | #2900の修正が取り込まれる統合版で、MG-29の現状固定テストを順序不変な`Unknown`期待へ更新。MG-29c（古いObservedAtでも構造Compatible）は**現在Gate資格の否定制御**として維持し、必要ならテスト名・説明を正す | merge後に古い後勝ちExpectがテスト失敗するのを修正コードの退行と誤判定しない／逆に古いExpectのため安全修正を取り消さない |
| **C. #2892の局所修正** | HIST-01/03の独立Expectedを先に固め、Unknown/partial/取得不能を`RepoApplied/Pending`へ昇格しない。failed/diverged/orphanと真正empty historyの回帰を確認 | 新History Registry/外部History修復/DB実作用の実装を混入させない |
| **D. 統合head** | #2900相当の局所修正・#2892安全履歴・#2887側の期待修正が同一Git treeで存在する場合、**そのexact-head**のGo/Workspace/Repo健全性を別に検証。申告済み旧/新Release要件・必要集合の充足を独立照合 | 互いに異なるPRのgreen CIを足し、合成された新headのPASSだと主張しない |
| **E. 製品Gate** | 実ProviderでのSchema/CDC/Writer/Commit・Receipt/別Read、旧新Clientの同一Scenario、SHO/外部Consumer、Owner受理の各obligationを証明 | A〜Dの成功を現行Host Grant/正本移管/Contract物理破棄の許可に変換しない |

**合流の選択肢**：本書では原則として#2900の修正を先にレビューし、#2887の特性テストを次の統合headで修正することを推奨。ただしOwnerが単一統合PRを選ぶ場合も、**安全修正と旧期待値の更新を同一統合単位**として扱う。#2887の`MG-29`を削除して論点を隠すのではなく、危険入力を安全に拒否する再現可能なテストへ転換する。

### 19.3 横断的な「入力曖昧性／必要証拠欠落」を取り違えない

ほかのレーンでも、DSL Role定義の重複、JSON duplicate key、Studioで古い要求の後着応答、RunPlanの必要Host Grant未供給など、**「一つを勝手に選んだ」「与えられた分だけで全部と見なした」**ことが偽PASSにつながる反例が存在する。ただし一つの汎用レジストリ・万能Validatorを追加しない。

- **重複の拒否（uniqueness）**：#2897の`StateSnapshot/Provides`だけで解決。正しいDB Instance/Source/Observation世代は#2611/#2622で**明示的に選択**する。配列順や最大`ObservedAt`はAuthorityではない。
- **既知状態の不確実性（status）**：#2892のHistory`AppliedUnknown`は「観測値が不明」。明確な不一致`diverged`、観測成功の`pending`、適用成功`applied`と混同しない。
- **観測集合の未完結（closure）**：必要Writer/Batch/Read Scope/Sourceを事前に確定し、`required≠supplied`または`required≠qualified`ならG2〜G5の該当Gate`unknown`。`CheckApplication`の`Checks=[]/Compatible`から「旧Client不存在」を導かない。
- **時間と効果の失効（freshness/effect）**：同じ入力版・DB Row値に戻るABAであっても、Owner/Grant/Fence/CDC/External Effectが以前の資格へ自動復活するわけではない。停止したGateの前にCommit済みの効果は残り得る。

### 19.4 実装成果を何と呼ぶか

**局所特性テストPASS**＝現行または修正後の判定器の特定条件を検査したこと。**統合版Conformance PASS**＝同じhead上の必要な構造・負例を検査したこと。**実DB/Client観測PASS**＝指定Provider/対象/時点の実作用を独立に観測したこと。**Owner-accepted migration**＝実意味、データ履歴、正本移管、物理破棄の必要判断が揃ったこと。段階を混ぜない。#2900の成果によって#2892やMG-24〜28、外部Consumerまで完成したとは判断しない。

**本節は設計/合流計画のみ。今回#2900の修正コード、#2892本体、実DB/Owner状態は変更しない。**


### 19.5 #2892の局所的なHistory fail-closed Draftと観測元の残余（2026-11-06）

[**Draft PR #2902**](https://github.com/hat47x/tei-platform/pull/2902)は[#2892](https://github.com/hat47x/tei-platform/issues/2892)の**HIST-01 P0**を既存`CompareHistory`内で限定修正。履歴Entryの`AppliedUnknown`／不正な`AppliedState`を、Definitionとchecksumが一致していても**比較エラーでfail-closed**にし、Repositoryに存在しないunknown Entryも`RepoOrphanApplied`として安全と呼ばない。既存`RepositoryState`のenumやHistory Engineを増やさない。HIST-01、unknown orphan、空/不正status、真正empty history、既知success/failedのGoテスト5関数を追加した。

**残す区分・次の試験**：

| 対象 | #2902で成立する可能性がある局所契約 | 成立しない現在のMigration Gate資格 |
| --- | --- | --- |
| HIST-01 applied unknown | 観測Entryの不明な適用状態を`RepoApplied`へ昇格させない | 実DBのSchema/Effect/現在Grantや対象Clientの互換性 |
| unknown orphan／無効state | Repository内外を問わず、状態不明・未対応を`orphan-applied`等の成功にしない | Provider/外部Toolの実用歴史の取込能力・別接続の独立Read |
| HIST-03 真正empty vs 取得失敗 | **完全取得された**空のHistoryは従来どおり`RepoPending`。比較器の入力だけでは取得失敗/permission-denied/partialを識別**できない** | History Reader/Adapterでsource、取得資格、Completeness、CapturedAt/時点を別に証明。失敗を空Entriesに変換しない |
| HIST-02 checksum欠落 | 既存動作を変更しない（適用記録の報告と同一Definition byte証明は別Claim） | checksum欠落を黙って「契約一致・現在適用資格」へ格上げしない。Provider/Ownerが許容条件を定義 |
| HIST-04 failed/diverged/orphan の既存分類 | 現行比較の退行防止。History Adapter/Tool本体は更新しない | 既存のPostgreSQL参照結果やMigrationログをG3/G4/G5の実証へ自動加算しない |

**レビュー・合流順**：#2902のexact-head局所Goテストを先に確認し、#2900のDuplicate State/Provided Contract修正とは別責務で受け入れを判断。移行計画の#2887は安全修正両PRと**同じ統合Git tree**でMG-29特性テストの旧last-wins Expectedを安全な`Unknown`へ変更し、HIST-01とMG-20の資格上限を再照合する。異なるbranchのgreen CIを足して一つの新しいE2Eの成功としない。

**状態**：#2902はGoコード＋合成Unit TestのDraftであり、Provider実History reader/失敗注入・実DB Migration・本番旧Client/SHO Consumer・Owner受理はNOT RUN。History sourceの完全性と鮮度を検証する公的契約は未決。`CompareHistory`の返却エラー方針の既存利用先への影響もレビュー対象。

### 19.6 同一HEADの統合検証専用DraftとClaim ceiling（2026-11-06）

[**Draft PR #3018（検証専用・merge禁止）**](https://github.com/hat47x/tei-platform/pull/3018)を作成し、#2887のMG-01〜29計画・Go特性テストを基線に、[#2900](https://github.com/hat47x/tei-platform/pull/2900)（Releaseの二重last-wins拒否）と[#2902](https://github.com/hat47x/tei-platform/pull/2902)（Migration Historyのunknown拒否）を**実際の同一Git tree**へ取り込んだ。#2887の旧挙動を再現するMG-29a/bは**順序に依存せず`Unknown`へ拒否する安全Expected**に変更。MG-29cの`ObservedAt`が古くても構造的に`Compatible`という局所特性は残し、現在Gate資格の別観測を引き続き必要とする。

| 資格 | #3018で確認する対象 | **対象外・禁止する昇格** |
| --- | --- | --- |
| **Source composition** | #2887 `321bbfea` を基点に、#2900 `dcb21e9`（2ファイル）と#2902 `adfdd2d`（2ファイル）を組み合わせ、MG-29a/bの期待値を安全仕様へ更新 | 異なるPRのCI成功の合算、後から変更された各枝を同一版と偽ること |
| **Go・Workspace conformance** | この**exact head**のGo test/vet/gofmt/参照Workspace。重複DB/Contract、unknown History、旧新Releaseの既存Expectedが同時に成立すること | 合成Go fixtureのsuccessを実Provider・実Client/CDC・業務意味Acceptedへ昇格しない |
| **Runtime・Owner** | 未観測/未実施の項目をRequired/Undischargedに残し、実DB独立Read/Writer Fence/Receipt/Retention等の停止条件を明記 | Sourceの構造互換・RepoAppliedの一部成功をCutover/Contract完了へ変換しない |

**合流選択**：#3018は製品PRではなく**統合適合診断専用**。#2900/#2902の個別レビュー・必要な修正と#2887の期待値改訂が受理された後、Ownerがmainへの正式な統合経路と順序を決める。#3018を先にmergeしたり、他ブランチをforce/rebaseしたりしない。**2026-11-06検証結果**：#3018の固定HEAD `983b9ed0f212fe6c6c227a2d71587ccdd84dfe09`は GitHub Actions 6件すべて **completed/success**（TEI PR・main健全性確認、セルフビルド、Studio参照編集、Desktop WebView smoke、Wails Host、依存脆弱性確認）。#2900の `dcb21e9`、#2902の `adfdd2d` も各exact HEADで6/6 success。ただしこれは同一Git treeの**Go/Workspace/CI適合**であり、実DB/旧新Client/CDC/外部履歴取得/Owner受理の資格ではない。#3018は依然Draft・統合禁止。#2887は修正前last-winsを期待するMG-29特徴付けを保持しており、#2900を取り込む正式な統合単位で安全Expectedに更新する。

**残余**：HIST-03のHistory Adapterからの取得失敗→偽空入力、HIST-02 checksum欠落のProvider別許容、旧新Client・月次Batch・CDC/Sourceの網羅、MG-24の安全停止と現Grant/Effect/Receipt、過去時点History/SHO参照、物理削除Owner受理。新Gate Engine/Registry/Hostは作らない。


### 19.7 History Readerの閉包確認：真正な空履歴と取得失敗を区別する（#2892、未採用の最小Fixture）

**現行コードで確認できる限界**：`HistorySnapshot` は `Database / Tool / CapturedAt / Entries` を持つが、取得の成功/失敗、Pagination完結、許されたScope、Readerの実接続先、SourceのRevision/Watermarkを保証するフィールドはない。`CompareHistory` は `Entries=[]` を「履歴なし」として `RepoPending` に分類するため、**Readerが失敗・権限不足・途中応答を空Snapshotへ潰すことを禁止**する。#2902の `AppliedUnknown`拒否は、既に取得した各Entryの適用状態を限定的に検査するもので、Readerの完全性は別のClaimである。

**検収対象**：History Readerが既存のConnection/Providerを利用してHistorySnapshotを作る境界。独立Expectedは、比較処理より前に固定したリポジトリMigration定義 `m001`、Readerの合成fixture、要求Scope、取得成功/失敗・ページ数の**事前に定めた正解表**から作る。Reader/比較関数の出力をExpectedへ逆算しない。実Providerの確認では別Readerまたは別の証拠経路で事実を照合する。外部Migration ToolのHistory TableをTEIが修復・上書きする機能は実装しない。

| 既存HIST-03を分割した最小負例 | Readerの事実 | 比較器へ渡してよいもの／期待 |
| --- | --- | --- |
| **HIST-03a 真正な空履歴** | DB/Tool/Scope確定、Reader成功、Pagination完結、CapturedAtが要求窓内、実行Grantも適切 | `Entries=[]` を渡し `RepoPending` を評価可能。ただし未適用の業務的意味までは確定しない |
| **HIST-03b 接続切れ・拒否** | Network error／PermissionDenied／Reader例外 | 空の `HistorySnapshot` を合成しない。取得は `unknown`、比較実行は保留 |
| **HIST-03c 部分ページ** | 1/2ページだけ取得、Cursor失効、Readerが上限で打切り | 欠落Entriesを「不存在」とみなさない。必要範囲の完全性が証明できるまで `unknown` |
| **HIST-03d 別Target・古い観測** | 履歴IDは同一だが実DB Instance/Provider・Source時点が異なる、CapturedAt期限切れ | 履歴内容一致でも現在Targetに対する `RepoApplied`資格を付与しない。既存Binding・現行Grantを確認 |
| **HIST-03e ABAと欠落** | Source/Historyが A→B→A と表面値に戻る、または履歴の特定IDの検索自体を実施できない | 旧Receipt/Old Observationの再利用資格を別再評価。取得済み分だけを全集合としない |

**最小受入票（既存Evidenceへ記録、正本型新設ではない）**：`target database instance + provider + tool + source/reader revision + observed interval + capturedAt + query/filter + expected scope/entry set + pagination completion + current grant + error/partial/permission-denied + independent read/checksum`。Sourceに資格がない場合、`CompareHistory` を実行できてもG1〜G5の必要なHistory Claimは `unknown` に留める。正しい空履歴、適用済み、不明、拒否、部分の区別を消さない。

**導入順**：
1. #2892のReader/Adapter所有者が、実際にHistorySnapshotを供給する既存経路を列挙し、取得失敗/部分取得時に空のEntriesへ変換していないかを確認（Readerがまだ存在しないProviderは未実装として明示）。
2. 実接続を伴わないReaderのsynthetic fail/partial対照でHIST-03a/b/cを確認。その後に必要なら対象Providerの**実History接続・別Reader**でHIST-03d/eを検証する。
3. #2902の局所 `AppliedState` fail-closedと今回のReader完結性を**同じSource/Provider/Attempt**で照合してからG3/G4のRequired Historyへ供給。Provider未検証、未接続、権限不足なら合格にしない。

**採用・実証の上限**：上記は#2892の受入条件案であり、HistorySnapshotの新しいフィールドや全Provider共通Adapterを採用したものではない。実Readerの完全性・異種DB・Cutover・Owner承認のE2E試験はNOT RUN。#3018の6/6 successをHIST-03の実証へ昇格しない。


### 19.8 履歴の内容照合と取得元資格の独立性（HIST-03dの特性試験）（2026-11-06）

**新しく直接確認したSource上の境界**：`CompareHistory` は `HistorySnapshot.Database` と `AppliedMigration.Database` が**空欄のときも比較を許し**、`Tool` と `CapturedAt` は結果の分類時に資格条件として検査しない。従って、Repository定義とEntry ID・checksumが一致する場合、**実際の読取先や時点の証拠が欠落していても`RepoApplied`になり得る**。これは比較器の設計上の限界であり、現時点で「現在DBにMigration済み」というGate Claimではない。

`internal/database/migration/history_origin_characterization_test.go` に、既存HIST-03dのための**現在の非認証挙動を特徴付けるGoテスト**を追加した。固定`m001`と一致するchecksumについて（1）Snapshot/EntryのDB・Tool・CapturedAtが空、（2）CapturedAtだけ空、（3）Entry.Toolが空、を変更し、`CompareHistory` が`RepoApplied`に分類し得ることを検査する。**このGoテストが成功した場合でも、求める移行Gateの正解は「Source/Reader/Instance/取得完結・現在時点が未確認なのでunknown」**である。構造比較結果とGateの独立Expectedを分離する。

**追加採用を提案する最小手順（Provider/Reader実装は未決）**：

1. 外部Migration Tool／Source Readerは既存の`HistorySnapshot`へ正規化する前に、**実Connection Target + DB Instance/Provider + Tool Source + Query Scope/Filters + 取得完結性 + CapturedAt/Revision + Grant**を個別観測する。欠けた場合は`HistorySnapshot{Entries:nil}`を「空履歴の成功」として供給しない。
2. 内容比較`CompareHistory`を呼ぶ資格があることと、結果の`RepoApplied`が現在Gateへ使えることを分離し、後者は独立のActual Schema／Provider Effect／必要ならReader再観測を要する。`Database` の単純な文字列一致、履歴のchecksum、`CapturedAt`の新しさだけでAuthority/現在性を推測しない。
3. **第一候補は上流Reader/既存Bindingでの資格確認**とする。実Connection/Providerの観測を持たない`CompareHistory`へ`CapturedAt`の必須化やTool文字列照合だけを追加しても、真の取得元・ページ完結性や現在Grantは証明できないため。比較器は引き続き**取得済みEntryとRepo定義の内容比較**を責務とし、必要なら不整合な入力値の診断だけ局所追加する。Reader/Bindingで確定した観測資格をGateへ渡せない場合は、比較直前の**スコープ限定の受入チェック**を次点として評価する。ただしどちらも新しい正本Registry・History Engine・Migration Coordinatorを導入せず、#2892/#2622/#2611の既存責務でOwner判断・必要な実Provider実証を行う。
4. 合格の実証では、正常取得／拒否／部分取得／別Instance・異なるProvider／古い観測を同一`m001`の独立Expectedで比較し、`required/supplied/observed/qualified`を個別に示す。実接続のないProviderでは「未実装・未実施」と記録する。

**合流時の注意**：本節のGoテストは**現行比較器の限界のcharacterization**であり、安全なReader admissionの実装完了を示すものではない。将来`CompareHistory`自体に厳格なprovenance検査を入れる場合は、旧`RepoApplied` Expectedを拒否/不明のExpectedに**同じ統合HEADで改める**。#3018の固定HEAD`983b9ed`は今回の追加テストを**含まない**ため、過去の6/6合格をこの新テストへ流用しない。

**状態**：試験のGo Sourceを追加し、新CIの結果は別確認。実DB、History Reader/Adapter、旧新Client、CDC/Owner Grant/Receipt/独立Read/Contract受理は**NOT RUN**。


### 19.9 History Readerの二段階閉包：ページ完了≠対象全集合（2026-11-06）

**新しい独立反例（既存HIST-03cの深化）**：History Sourceに適用済み`m001`（Repositoryにも定義あり）と`m999`（Repositoryには定義なし、orphan）が実在する。Readerが `WHERE id IN (repo IDs)` で`m001`だけを返し、**全ページ完了**と報告した場合、`CompareHistory`は`m001=RepoApplied`のみを返し、未定義の`m999`が隠れる。**全ページ取得と対象集合の完全性は別であり、ページ完了だけをG2/G3の履歴Gate合格にしてはならない**。

`internal/database/migration/history_origin_characterization_test.go`に `TestHIST03cFilteredHistoryHidesOrphanDespiteCompletePages` を追加し、(a)固定Source真値`{m001,m999}`に対して両ID取得なら`RepoOrphanApplied`を検出、(b)同じSourceから`{m001}`しか供給しない場合は`RepoApplied`のみと分類される、という**現行比較器の情報限界**を特徴付ける。独立Gate Expectedは(b)を`unknown / unqualified-history`とする。Readerでフィルタをかけた場合でも「全ページ完了」の真偽は範囲内に限られ、Scope変更・RLS/Permissionによる非開示・Replica遅延・未定義のHistory IDがないことを推認できない。

| 既存HIST-03との照合 | 固定した読取条件・World真値 | 技術上の`CompareHistory`結果 | **G2/G3の独立Expected** |
| --- | --- | --- | --- |
| **03a 真正な空** | 権限を確認したFull History Scope、対象DBを固定、必要な全頁・一致Watermark、空であることを別観測 | `RepoPending`候補 | 技術上の未適用を判定可能。ただし業務移行完了ではない |
| **03c-1 全取得** | Source `{m001,m999}`、History Scopeを絞らず全頁・権限／可視性を確認 | `m001=RepoApplied; m999=RepoOrphanApplied` | 未定義適用履歴があるためCutover停止・Origin/Owner照会 |
| **03c-2 限定フィルタ** | 同じSource、`WHERE id IN (m001)`、結果`{m001}`、フィルタ内では全頁完了 | `m001=RepoApplied`のみ | **unknown**。Orphan不存在の証拠にはならない |
| **03c-3 隠れた行** | Full queryでもDBのRLS/権限・未開示Scopeで`m999`非表示 | `m001=RepoApplied`のみになり得る | **unknown/permission-limited**。サーバー成功と全可視性は別 |
| **03d/e stale／ABA** | 同じ論理DB ID/History ID・見かけの値へ復帰、別Instance/古いWatermark | `RepoApplied`になり得る | 現在のOwner/Source/Read Scope・Writer/CDCの資格が必要 |

**最小Reader受入契約（新しい共通Core型を要求しない）**：
1. **Target identity**：実Connection/Instance/Provider、Tenant/Organization Scope、read replicaかprimaryか、対応するSource/World・Read epoch。`CanonicalID`だけでは実Targetが一意にならない（#2611/#2622）。
2. **Transport closure**：Read成功/失敗、Cursor/全ページ、サーバー読み取り一貫性、Watermark/Replica lag、関係するSource版・時点。失敗・部分は空履歴に変換しない。
3. **Semantic scope closure**：適用済み・失敗・unknown・Repository定義にないOrphanを検出できる**History全範囲**と可視性。Repository内のIDで先にフィルタしてから「すべて取得」と判定しない。権限で見えない可能性がある場合は正当な存在否定に使わない。
4. **Qualification and disclosure**：現在のHost Grant、必要Source/Provider版、独立Expectedとの照合をGateごとの`Required / Supplied / Observed / Qualified / Undischarged`に記録。低権限ConsumerへDB接続情報・不可視行の個数・SQL全文を漏らさない。履歴の内容比較が成功しても実Schema/業務Key/Old Writer/外部Effectの成功とはしない。

**実行順**：#2892で既存Reader/Adapterの有無・投入されたSource/Scope・フィルタ・失敗の伝搬を特定 → `03c-1/2/3`のSynthetic同一Source対照を先行 → Provider付き別Reader（必要なら実PG）で`03a/b/d/e`の取得資格を確かめる。既存[#2555](https://github.com/hat47x/tei-platform/issues/2555)に記載された`tei dbcheck`は、DSNを指定した場合**TEMPORARY TABLEのCREATE/DROPを行う接続能力試験**であり、Read-only History Readerでも完全取得試験でもない。`dbcheck`の成功をHIST-03/Cutoverの資格へ昇格させない。秘密DSNを証跡に記録しない。

**資格の上限**：このGoテストは`CompareHistory`の構造的限界の特徴付けであり、適切なReaderの存在・保護・全件取得を立証しない。#3018の旧HEADの6/6成功は**この新ケースを含まない**。Reader/Provider負例の実実行、実DB History接続、業務Owner受理はNOT RUN。新History Registry/Engine/独立Gateを追加しない。


### 19.10 Migration checksumの証明領域：適用申告・定義同一性・生成SQL・実効果（HIST-02／他チャットのSource-Artifact-Effect分離、2026-11-06）

**直接確認した現コードの二つの境界**：`migration.CompareHistory` は`AppliedSucceeded`でも`AppliedMigration.Checksum=""`なら不一致検出をせず`RepoApplied`に分類する。また`migration.Fingerprint`は正規化した`Definition`から`Realizations`（生成されたTarget SQL文群）を**意図的に除外**してハッシュ化する。前者は「checksumなしの適用申告」、後者は「同一Definitionで生成SQLが異なり得る」問題であり、**checksum有無だけでは両方を一度に解決できない**。

`internal/database/migration/history_checksum_characterization_test.go`に**現在の限界を固定するGoテスト3関数**を追加した。

- `TestHIST02MissingAppliedChecksumCannotDetectDefinitionDrift`：固定した同一IDの`original/changed`（Step内容相違）に対し、履歴の`AppliedSucceeded`・checksum欠落で両方`RepoApplied`となる。独立にold checksumを与えた場合だけ、変更後の定義は`RepoDiverged`になる。
- `TestHIST02CanonicalChecksumDoesNotCertifyDerivedRealizationBytes`：同じCanonical Definitionの下で生成`Realizations`（SQLの列型）が変わっても`Fingerprint`が一致し、そのCanonical checksumをHistoryに与えれば両方`RepoApplied`になる。**実Toolがこのchecksumを供給すること、どちらかのSQLを実行したことを示す試験ではない**。
- `TestHIST02DifferentDigestBasesLookLikeDefinitionDivergence`：**同一Canonical Definition**に対し、Repository側はCanonical JSON、仮のTool側は同じ定義から導出したSQL本文をSHA-256にかけると、ハッシュの対象バイトが違うため`CompareHistory`は`RepoDiverged`となる。これは**外部Toolの実アルゴリズムを再現したテストではなく**、比較対象の契約を確定せずに異なるハッシュを突き合わせた際の誤った「定義変更」推認を固定する合成負例。GateのH1は`unknown`を保持する。

#### 19.10.1 Evidenceの4つの主張を区別する

| Claim | 確認する内容 | 合格にできない推論 |
| --- | --- | --- |
| **H0 Applied report** | 対象DB・Tool・Reader/Instance・Scope・取得完結性が資格化されたHistory Entryは`AppliedSucceeded`と報告 | Providerが書き込んだ申告だけで定義バイト同一・実Schema一致・現在のHost Grantが確定するとはしない |
| **H1 Canonical Definition identity** | Repositoryの版に固定した`Definition` byte列と、対応する履歴上の検証可能な定義ダイジェストの**同じ算法/同じ対象**を独立照合 | Provider checksumとTEI`Fingerprint`の文字列を、ハッシュの対象/アルゴリズム確認なしに比較しない。checksum空を一致としない |
| **H2 Realized SQL identity** | Dialect、Target Generator/Plugin版、実際に投入したSQL（Provider/Toolが記録したときのみそのbyte digest）、適用順/Tx境界を相関 | `Fingerprint`一致や`RepoApplied`を、`Realizations`や実際に実行されたSQLバイトの一致に昇格しない。生成SQLの変更だけを業務意味変更と自動判定もしない |
| **H3 Actual Schema/Data/Effect** | 実Target/時点を固定したActual Schemaと必要なデータ/キー/履歴/別Connection Read・Effect/Receiptを**独立に観測** | SQL実行成功やHistoryログだけで業務データ全件・CDC・旧新Client・外部Consumer・Owner受理は確定しない |

**現在の具体的な責務配置**：H0取得完全性とTargetは#2892/#2611/#2622、H1は既存`CompareHistory`とTool別Checksumの意味の検証、H2は既存Generator/Provider/Build Evidence（#2600等）、H3は#2555/#1174/#2514/#1163の実観測とOwner判断。新しいHistory Registry、全Provider共通SQL Receipt、Gate EngineやCore enumはこの検討だけでは導入しない。

#### 19.10.2 未決定と最小受入

- **HIST-02a：適用記録にchecksumがない**。`RepoApplied`は現行比較器の**局所分類**として維持しても、Gateへ提出する`H1 Definition identity`は`unknown`。代替証拠を使う場合、同じSource/Definition版・Target・実Reader時点へ独立に結び付く原本Migrationファイル/外部Toolの来歴などを明示し、Owner受理まで無音昇格しない。
- **HIST-02b：Canonical checksumはあるが生成SQLが違う**。`Fingerprint`の`Realizations`除外は意図的な契約として維持。Target SQLの固定版・生成器版/Provider能力・実投入記録がない限り`H2 Realized SQL identity`は`unknown`。`H1=pass`を`H2=pass`としない。
- **HIST-02c：Tool checksumとCanonical checksumの定義が異なる**。同じ`sha256:`形式でも、Toolの対象がSQL文列、TEIが正規化Definition JSONなら、**定義に差がない合成例でも`RepoDiverged`と出る**。元の対象バイト列・正規化・改行/補間・実行したTool/Generator版・Source revisionの契約が確認できなければ、**不一致を業務上の定義変更や適用失敗と断定せず`H1=unknown`に止める**。同じ算法名・文字列形式だけでは資格にならない。
- **HIST-02d：Historyと実DBの時点が不整合**。履歴の取得元とActual Schema/データの観測区間やReplica水位が整合しなければ、H0〜H2の一部が成功しても`H3=unknown`。SourceのABAでも古いReceiptの資格は復活しない。

**検証の独立Expected**：同一Migration IDに対し、Definitionの旧新・Checksum欠落/一致・Realizations差・Provider/Target/Owner Scope違いを明示した**事前固定表**を作り、H0/H1/H2/H3をそれぞれ`pass / blocked / unknown / not-run`で記録。各段で`Required/Observed/Undischarged`を別々に示す。今回のGo特性テストはchecksum欠落・生成SQL差・ハッシュ対象不一致という**3件の判別限界**を再現したのみ。実SQL投入、Tool checksum algorithm照合、実DB Query/Schema/旧新Client、Owner受理は**NOT RUN**。

**次の最小順序**：#2892がHistory Reader/Toolの実際の取得元・checksum意味/有無を1 Providerで記録 → #2622/#2611がTarget・Scope・資格の一意性を照合 → #2555のG2/G3で、H0〜H3のうち何を必須とするかOwnerが用途別に明示。checksumのないProviderを一律排除する製品政策や新しいProof Storeは、現時点では採用しない。


### 19.11 H0/H1/H2/H3とRelease互換性の同一Target・同一時点結合（既存#1163 Evidence Packetに接続、2026-11-07）

**確認したSource上の非結合**：`release.StateSnapshot` は`Database/Source/Release/Provides/DataReadiness/ObservedAt`を持ち、`CheckApplication`はContract Majorと`DataReadiness`を照合する。`migration.HistorySnapshot` は`Database/Tool/CapturedAt/Entries`を持ち、`CompareHistory`はRepository ID/checksum/Stateを照合する。**どちらも、相手のObservation ID・実Connection/DB Instance・Provider・同一Read Epochを入力に持たず、相互の一致を検査しない**。両結果が局所`Compatible`と`RepoApplied`でも、それらが同じデータベース世代・同じ事象の証拠とは確定しない。

| 既存MG-22/MG-27＋HIST-02/03の圧力ケース | 固定した独立Source/実対象 | 局所チェックで起こり得ること | **統合Gateの独立Expected** |
| --- | --- | --- | --- |
| **XH-01 同名別Instance** | 論理ID`sales_core`は同じ。Stateは新主DB、Historyは旧DB/異Tenantから採取 | `Compatible` ＋ `RepoApplied` | **unknown**：実接続先・Tenant/Provider/Authorityの同一性未証明 |
| **XH-02 観測時刻ずれ** | Stateの`ObservedAt`は切替後、Historyの`CapturedAt`は切替前。途中にDDL/CDC/権限変更 | 両局所判定は時点の整合を検査しない | **unknown**：Read Epoch / Source revision / Commit・CDC境界を再取得 |
| **XH-03 同じ値への復帰** | Source/定義 A→B→A、旧Receiptのchecksumは再び見かけ上同じ | Canonical/Schemaの文字列一致だけでは過去Observationの有効性は復活しない | **unknown**：依存関係・Generator版・再観測を再資格化（#2506/#2600） |
| **XH-04 空/部分の別Reader** | 必要History全範囲でない検索結果、別時点のReplica、RLSが隠したorphan | 内容比較で`RepoApplied`、Release互換でも`Compatible`になり得る | **unknown / blocked by known orphan**：ScopeとReader可視性・水位を独立確認 |
| **XH-05 完全な同一Attempt** | 同じ実DB Target、必要なRead ClosureとSource/Provider版を固定し、Release契約/History/SQL投入/Actual Schemaをそれぞれ別観測 | 局所値の整合を**必要条件の一部**として使える | **それでもG4/G5は未合格**：現Grant/旧新Client Writer/外部Effect/独立Read/Owner受理を要する |

**最小の合成受入票（新Core型ではなく#1163既存Packetへ追加するRequired欄）**：

`target logical ID + actual instance/connection identity + provider/tool identity + tenant/org scope + selected DB alias + Source/World revision + operation/attempt ID + snapshot/read epoch + observed/captured interval + complete Reader scope + checksum basis/algorithm + generator/toolchain version + SQL input digest (if H2 is required) + current Grant + independent Actual Schema/Data read`。

値を持たない場合は対応するClaimを`unknown / not-run`として残す。実DSN、不可視Row件数、秘密SQL/資格情報の開示は必要権限で制御し、通常UIでは業務操作・保留理由を優先する（#2516/#2622）。**異なるGit HEADのCI合格や別AttemptのRead成功を結合の証拠へ合算しない**。

**実施順と責務**：①#2892が1 Reader/ToolのRead Scope・Checksum Basis・取得時点を固定、②#2611/#2622が実Target・Alias・Providerの同一性を確認、③#2600がGenerator/Artifactから実SQL投入への根拠を対応付け、④#1163/#1119がH0〜H3・`CheckApplication`・独立Schema Readの**同一Source/Attempt結合**を判定、⑤#2555/#1174が旧新Client/CDC/Fence/Effect/Receipt/Owner受理を追加。**この横断表の作成は実Reader、実DB、同一Attemptを実行した証拠ではない**。新Gate Engine、正本Registry、分散Transaction Managerは追加しない。

<!-- tei-db-migration-apply-admission-20261107 -->
## 20. Migration Apply Admission：同じSQL・同じTarget・同じ移行前提は別の主張（2026-11-07、設計差分／未受理）

### 20.1 既存実装の境界（Source-static）
* `internal/database/migration/model.go` の `Definition` は `Database`、`BasedOnActualFingerprint`、`Steps`（schema/data/verify）、`Realizations`を保持し、`plan.go` は計画時Actual fingerprintを定義へ写す。これらが記録されていても、適用時の資格検査が全件実行されたとは限らない。
* `internal/database/migration/postgres_apply.go` の `ApplyPostgresWithFreshObservation` はSchema StepsからDiffを復元し、提供されたfresh PostgreSQL Observationで再Renderし、計画済み文列と`slices.Equal`比較した後にExecutor callbackを**一回**呼ぶ。しかしこの関数には`BasedOnActualFingerprint`との比較、実Connection/Instance・Tenant・Aliasの一致、Schema+Data+Verify全Stepの実行確認、Provider Tx/Lockによる観測から作用までの原子的保護はない。これは関数**単体**の責務境界であり、製品全体の経路が必ず無検査だという主張ではない。実Callerの検査所在を先に特定する。
* `postgres.SchemaObservation.AssessDestructiveReadiness` はDropTableがなければ、`AssessConstraintDataReadiness` は既存表への制約追加がなければ、対象DB文字列を比較する経路まで到達しない。普通のAddColumn等に対し、`ApplyPostgresWithFreshObservation` の任意callbackだけで実Target一致まで保証することはできない。
* `PostgresStatementExecutor` は`func(context.Context, []string) error`。**callback一回呼出し≠SQL文列が単一Tx・全成功・全失敗・再試行安全**。成功応答も別接続Actual Schema／Data・History・Consumer受領を含まない。

### 20.2 提案する三つの検収面（新Core Gate・Migration Engineではない）

| 面 | 必要な確認 | 資格の上限／既存Owner |
| --- | --- | --- |
| **P: Plan Provenance** | Source/World Revision・受理Meaning、Logical Database・Migration Definition/Parents、計画時Actual fingerprintと対象Scope、Generator/Provider/Realization版、実SQL Assetの選択 | 正しい計画候補であることまで。#1126/#2611/#2622/#2600、既存§19.10 H1/H2 |
| **A: Apply Admission** | 実Connection/Instance・Alias・Tenant/Scopeの固定、必要な現Actual Schema・Data・依存条件の再観測、current Host/Provider権限、計画との整合、実行中のLock/Tx/Fence能力 | 作用前の許可候補まで。実行中の変化を防げない場合はその保証を`unsupported/unknown`。#2622/#1174/#2555、既存G1/G4/G5 |
| **E: Effect & Verification** | 実際の文列・Statement単位結果・Tx境界・partial/unknown、History H0〜H2、別接続Actual Schema/Data再読取 H3、Client/CDC/外部Effect/Owner判断 | Schema SQL成功を全Migration完了に昇格しない。#2892/#2555/#1163/#2514、既存G1〜G5 |

**重要な選択**：`BasedOnActualFingerprint`の変化を**検出すること**と、無関係なDDL変更を含めて全件を恒久的に拒否することは別。初期P0では未評価の差分はfail-closedとし、再Plan／明示的な限定再受理を認める場合だけ、変更された依存集合が移行の安全性・意味・結果に影響しないことを**独立Expected**で検証する。Fingerprint値の一致だけで旧Reader/Source/Grantの現在資格は復活しない。

**Observation→Executorの保証**：観測callbackと実行callbackを別に渡す現APIのままでは、fresh観測の後に別Sessionが変更できる。PostgreSQL 16の`ALTER TABLE`は操作別にLock levelが異なり、`CREATE INDEX CONCURRENTLY`は通常のTx block内に置けない（公式資料： https://www.postgresql.org/docs/16/sql-altertable.html 、https://www.postgresql.org/docs/16/sql-createindex.html ）。したがって「全Migrationを一つのTxにすればよい」と一般化せず、**実行する各Statementのatomicity可否・Lock/Barrier・失敗後の実状態・Recovery Strategy**をProvider側で検証する。保証できないPhaseをatomicとして宣言しない。

### 20.3 既存MA-01〜04と最小追加反例（独立Expected、未実行）

| Case | 同じSourceで固定する圧力 | 独立Expected／実施場所 |
| --- | --- | --- |
| **MA-01 Base drift** | `BasedOnActualFingerprint=A`で`ADD COLUMN`計画後、別DDLでActual Bへ変化。ただし再RenderのSQL文列は同一 | **SQL文列一致だけで現在のApply資格を与えない**。A/B差分・依存を説明し、再Planまたは明示再受理まで実行停止。上位Caller/Provider admission |
| **MA-02 Wrong target** | Definitionは`sales_core`だがObservation/Executor callbackが別Instanceを使用。DropTable/Constraint Additionのない通常のAddColumn | 実Instance・Alias・Tenant/Provider・権限を明示し、異Targetなら`executeCalls=0`。単なる同名Logical DB IDでは不十分。#2611/#2622 |
| **MA-03 Phase incomplete** | DefinitionにSchema/Data/Verifyを宣言、Schema Statementsだけ成功 | `schema-SQL-submitted`と`Data/Verify not-run`を別記。migration-completedは不可。G1〜G3 |
| **MA-04 TOCTOU** | Fresh Observation後、別SessionでSchema/データ変更を挿入してからExecutorへ進む | Lock/Tx/Version fenceの下で実対象を安定化できることを実証、または`unknown/unsupported`。独立再読取を付す。#1174/DB Provider |
| **MA-05 Partial executor** | 2文のExecutorが1文目適用後に失敗、またはCommit ACK消失。Executor callback呼出しは1回 | `error`を「全DDL no-effect」と推定しない。Tx保証が観測できなければ`partial/unknown`、現Schema/History再取得後に明示Recovery。新key盲再送禁止。#1174/#2892 |
| **MA-06 Authoring revision** | 旧`*.tei`計画を新標準`*.tei.md`に表現、SQL本文は既存`*.sql`のまま。表層・Reader・Parserが未受理 | ファイル名/SQL byte一致だけでMigration Definition/Meaning/Realization資格を再利用しない。旧経路の実PG試験を移行待ちにせず、新Reader/Parser・Source span・実消費Artifactで別検証。#2621/#3030/#3031/#2600 |

**MA-01/02の最初のテストは二段構成**：①現`ApplyPostgresWithFreshObservation`へ誤Target/変更済みActualの合成Observationを渡した際の**現実装の出力を特徴付ける**（安全な成功を意味しない）。②既存CallerのPlan Base・実Target資格を通して、同じ入力を**実Executorへ送らず拒否**できることを、Sourceと独立のExpectedで判定。②の実経路が未特定なら、新APIを決めず`unresolved`のまま停止する。①だけのGo PASSを②の資格としない。

### 20.4 実装・合流順（既存の修正Ownerを維持）

1. **P0 Source/Caller inventory**：`ApplyPostgresWithFreshObservation`の実呼出元・`BasedOnActualFingerprint`消費・実Target/Session・Tx・DDL対象を調べ、既存安全保証がどこにあるかを記録。①のMA-01/02特性試験と②の不足を分離。#2887に第二Provider/Runnerは増設しない。
2. **P1 正確な修正順**：#2902のHistory unknown修正、#2900の重複Release状態修正を、#2887の**旧挙動を固定するMG-29特性試験のExpected変更**と同じ統合Treeで調整。検証専用#3018はmergeしない。HIST-03 Reader完全性は別。
3. **P2 絞った実PostgreSQL Gate**：資格を持つ既存Test DB/DSNで、MA-01/02の停止、MA-04の実並行障害、MA-05のTx/非Tx別失敗、別Connection Actual Schema・Data readを測る。実DB接続・変更対象の同一性・Lock・権限・History取得範囲を記録し、DBcheckのTEMP DDL成功や別PRのPG CAS成功から推論しない。
4. **P3 Cutover/Consumer**：既存#2555の旧新Writer/CDC/書込み正本、#1174の現Grant/Commit/Receipt、#1163の一改訂Packetで必要なObligationを照合。G0〜G5を変更せず、Owner受理と実Target効果を別に評価する。
5. **P4 Authoringの並走**：旧`*.tei`の取得・PG実証を妨げず、選択された新`*.tei.md` Sourceの実Reader/Meaning/Buildを#3030側で再資格化。移行が未実装の段階で新形式を旧ParserのPASSとみなさない。

**状態**：本節は現main・既存Draft・PostgreSQL公式資料の静的照合に基づく設計提案。MA-01〜06の新Go/実PostgreSQL並行DDL/異種DB/Writer/Owner受理は**NOT RUN／NOT ACCEPTED**。Issue #2892/#2897/#2555/#1174/#2622 の担当者を置換せず、既存Plan/Release/Migration/HistoryのSourceを唯一の実装境界として再利用する。

<!-- tei-db-migration-reporting-failure-20261107 -->
### 20.5 実Effectと記録・通知の失敗：Command #1174のRC-03/RC-05を移行に移す条件（2026-11-07、静的検証・未採用）

**新たに確認した他レーンの反例**：[Issue #1174の11/7追記](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6696983123) は、`internal/command/invoke.go` の `Executor.Execute`成功後に `Recorder.Record` が失敗すると、`COMMAND_RECORD_FAILED` が `EffectUnknown=false` のまま返り得ること、`internal/command/lifecycle.go` の `Observe`・`MarkUnknown`自体の失敗がUnknown資格を外側へ確実に保持しない経路を示した。実mainのコードとの静的照合であり、実Providerによる故障注入成功の主張ではない。#3042で宣言外EffectがUnknownになったこと、#3053でUnknown表示が分離されたことは**この記録失敗経路を解消した証拠ではない**。

**DB Migrationへの転用可能な原則**：`migration.ApplyPostgresWithFreshObservation`の`PostgresStatementExecutor`は単なる`error`を返すCallbackであり、Commandの`Invoker/Recorder/Lifecycle`を実際に通るとは現時点で確認できていない。したがって「MigrationのRecorderに実Bugあり」とは主張しない。**適用Callbackの後で任意のHistory Writer・検収票Recorder・CLI/API返却・外部通知が失敗した場合を想定し、移行EffectとControl-plane Reportを別々に観測**する。実書込み成功と報告失敗をClean Rejectedへ変換しない。

| 新しい負例／対照 | 独立Expected（事前固定） | Claimの上限／担当 |
| --- | --- | --- |
| **MA-07a 健全な移行報告** | 同一Migration ID/Plan・Actual Target/Attemptで、Execute/History記録/別接続Schema・Data再読取が対応。Schema Step完了とG1〜G5を別主張にする | 実際に観測したStepの資格のみ。#1163/#2892 |
| **MA-07b 適用後History記録失敗** | 既に変更されたActual Schemaに対し、History記録だけNetwork/Permission error。Entry未取得・RepoPendingでもEffect 0とは推定しない。再読取まで`effect unknown`、観測で作用が判明すれば`effect observed; report missing`として修復計画へ | Record失敗時に`apply not-run`へ戻さず、同じMigration/AttemptでLookup。#1174/#2892 |
| **MA-07c 成功後CLI/検収票通知失敗** | 実EffectとHistoryは残るが、Callerの受領やEvidence保存が失敗。正常終了と誤って扱わず、保存済みSchema/History・Operation identityを取得し直す。新しいMigration IDで盲再実行しない | Consumerの受領とDB Effectは別。#1163/#2555 |
| **MA-07d Unknownの二次記録失敗** | 一次Executor/Commitの成否不明に続いてUnknownを記録する処理も失敗。二番目の障害により元のUnknownを消さず、再送可否は保留 | Unknownの伝播契約は既存#1174を先行検収。DB Migrationは実経路確認後に適用 |

**実装判断順**：①Migrationを実行する**実Caller**がHistory Reader/Writer・Outcome Recorder・CLIをどの順に呼ぶか特定。現状呼ばない機構を組み込んだことにしない。②Executorの`error`が「作用なし」「作用不明」「作用の一部確認」を表現できるか、既存Provider/Recovery経路の振る舞いを独立Fixtureで特徴付ける。③単体FakeでMA-07b〜dの分類を検証しても、実PostgreSQL Commit→記録失敗→別Connection Read→同一IDでの安全回復をE2Eで確認するまで完了扱いしない。**新Tx Coordinator・履歴Registry・Recorder正本を提案しない**。

**CI・合流に関する同一HEADの診断（2026-11-07）**：前段の計画更新commit `acfce089b6ec64797123291867813d93eac799f1` の [repository-health run #42297752845](https://github.com/hat47x/tei-platform/actions/runs/42297752845) は**failure**。job logから `internal/database/migration/history_checksum_characterization_test.go` の`gofmt drift`を特定。同じjobで `go vet ./...`、`go test -count=1 -timeout 60m ./...`、`tei validate -workspace reference/workspace` はPASS。他5件のWorkflowはcompleted/success。新規§20のDocs追補はGoの書式不一致を修正しないため、**PR #2887はHealth未合格**。該当Goファイルの書式だけを既存Draft Ownerが修正し、修正後の**新しいexact HEAD**でCIを再実行・再確認する。旧headのテストPASSを新headの合格として継承しない。

<!-- tei-db-migration-cli-closure-20261107 -->
### 20.6 Caller調査の確定範囲：CLIの計画・比較と実Provider Applyの分離（2026-11-07、main静的照合）

前節の「適用Callerを特定する」を一段進め、**少なくとも現行CLIのDB操作群は実Applyを接続していない**ことをSourceで確認した。これは全リポジトリに適用Callerが存在しないことの網羅的証明ではない。

| 確認済み入口／Source | 実際に入力・呼出しているもの | **ここでは成立しないClaim** |
| --- | --- | --- |
| `cmd/tei/main.go` `plan-migration`（約632–673行） | `-actual`で渡した**JSONファイル**を`SchemaSnapshot`へDecodeし、desiredを生成、`schemadiff.Compare`後、`migration.Plan(id, dbid, actual.Fingerprint, diff, provider)`を呼ぶ | CLIが現DBへ接続してScope/Instanceを再観測、`BasedOnActualFingerprint`を実Current DBへ照合、またはApplyしたという証拠にはならない。ここでのPlanは**入力JSONに基づく候補** |
| `cmd/tei/main.go` `database-reconcile`（約693–729行） | `-actual`と`-history`の**JSONファイル**をDecodeし、`migration.CompareHistory`と`reconcile.Build`で比較結果を返す | このCLI経路は実History Readerの取得成功/Completeness/接続先真正性、現在DBのApply事実を実測しない。局所比較に留まる |
| `internal/database/migration/history_provider.go` `ReadHistory` | `HistoryReader`のエラー・Top-level Logical DB/Toolを照合し、欠けた**個別EntryのDatabase/ToolをSnapshotの値で補完**する | Entry属性の補完は**観測された実Connection/DB Instance/Toolの出典証明ではない**。既存HIST-03dのReader資格、Scope/ページ完結性/権限は別 |
| `cmd/tei/dbcheck.go` | DSNでPGへ接続し、`SELECT version()`と同一SessionのTemporary Tableの作成・削除を試す | Targetに接続し簡易DDLが可能だという限定証拠。Migration DDL、History、別Connection Read、Owner Cutoverを試していない |
| `internal/database/migration/postgres_apply.go` | 任意の`observe`/`execute` Callbackを受け、再Renderが一致すれば1回Executorへ渡す | 実CLI、Host Grant、単一実Connection、履歴記録・独立readと**自動接続されているわけではない** |

**第一の独立反例（既存MA-01/02に統合）**：同じ`sales_core`の固定`actual.json`から`plan-migration`が成功した後、実DBのSchema/Instanceが変わる。CLIのPlan成功は正常な**offline plan claim**。そのJSONを直接`Apply Eligible`へ昇格するのは誤り。適用Callerが未確定なら`A: Apply Admission`は`not-run/unknown`、実Executorの呼出証拠なし。実DB readはSource/connection/Scope/epochを固定して別試験で評価する。

**第二の独立反例（既存HIST-03d/XH-01に統合）**：ReaderがTop-level `HistorySnapshot.Database=sales_core`を返す一方、EntryのDatabase/Toolは空。正規化後にEntryの文字列が一致しても、実DB Instance/Reader Sourceが同じだとは結論しない。`database-reconcile`のFile Inputでは、そもそもReaderの成功・取得完結性を示すものではない。実History Readerの誤接続/部分RLS/欠落を独立に検証する。

**次の境界決定（#2622/#2611/#2555既存Owner）**：`plan-migration`を実行入口に拡張するのではなく、①正式Migrationを実行するHost/Providerの責任者・接続先/Grant、②現Plan/History/Actualを同一Attemptへ結合する方法、③破壊的/非Tx/部分適用の停止/復旧保証を先に確認する。製品の`apply-migration`コマンドや新Migration Host/Storeは、この調査だけで新設しない。#1163の既存Evidence Packetへ`input-kind=file/live`・`input-scope/target/reader`・`actual-execution-attempt`を**診断項目**として渡し、新しいCore Schemaの採用は求めない。

**CI修正の事実と上限**：旧§20.5に記録した`acfce089`/ `8166c655`のHealth失敗（原因：`history_checksum_characterization_test.go`の`gofmt`）に対し、同じDraft branchで**空白1か所のみ**を整列するcommit `05304eab133627dca22bec4730343023c4749526`を作成、Sourceを再取得して変更位置を確認した。過去HeadのGo test/vet/validate PASSは現Headへ自動継承しない。新Headのexact-head CI完了を別途確認するまでHealthの成功を主張しない。新しい実Go/PGテスト、正式Caller接続、Owner受理はNOT RUN。

<!-- tei-db-migration-snapshot-trust-20261107 -->
### 20.7 Actual Schemaの入力真正性とClosed-World資格：`Fingerprint`値・表一覧を「実DBの現在事実」と誤認しない（2026-11-07、静的照合／提案）

**§20.6のSource照合を一段進めた新事実**：`physical.SchemaSnapshot`の主要属性は`Database/CapturedAt/Fingerprint/Tables`であり、取得した実Connection・Provider/Tenant/Schema scope、読取のページ完結性や権限・取得全体の成否をこの型自体は保持しない。`SchemaSnapshot.Normalize()`は表/列/Constraint/Indexの順序を整えるが、`Fingerprint`を計算・照合しない。`schemadiff.Compare(desired, actual)`はLogical Databaseを比較して表一覧を集合差分する。`cmd/tei/main.go`の`plan-migration`は`-actual`の**外部JSON**をDecodeし、`schemadiff.Compare(desired.Normalize(), actual.Normalize())`でDiffを作り、**ファイル内の値をそのまま**`migration.Plan(..., actual.Fingerprint, ...)`へ渡す。`plan.go`も文字列の真偽やSchema byteとの整合を検証せず、`Definition.BasedOnActualFingerprint`へ保存する。

従って、**現CLIが返すMigration Definitionの`BasedOnActualFingerprint`は「入力ファイルの申告値」であって、その時点で実DBから検証・束縛されたSchema fingerprintを表す保証はない**。値が空欄・古い・別Source由来でも、別の検査なしに`Plan`の入力へ渡る。ここでの`Fingerprint`が何をハッシュ／世代表示するかは現段階で一律確定していないため、既存フィールドを無条件で「正規化Snapshot JSONのSHA-256」とみなして上書きする設計は採らない。

**より重要なClosed-World反例**：`schemadiff.Compare`は`actual.Tables`に存在しない表を「実DBに存在しない表」と扱い、desiredにだけある場合は`CreateTable`候補を作る。しかしActual一覧がPermission/Filter/接続間違い/途中打切りで**部分取得された場合**、その差分は「表が存在しない」ではなく「表の有無を判定できない」。全ページ完了・Scope/権限が未確認のまま、構造Diffが解決済みであることを**G1のDDL適用資格に昇格させない**。データ行の0件/非開示とSchema一覧の欠落は別観測だが、どちらも部分集合を対象全集合と推認しない原則は共通（#2506/#2600）。

| 既存MA-01/02を補う新Fixture | 固定した同一Source・入出力 | **独立Expected（移行適用の資格）** |
| --- | --- | --- |
| **MA-08a Claimed fingerprint collision** | `actual-A.json`と内容を変えた`actual-B.json`へ同じ申告`Fingerprint=fp1`を与える。`BasedOnActualFingerprint`は両方同じになり得る | Diff/Planの生成が成功しても、FingerprintがSchema内容と対応する**証明なし**。`Apply Admission=unknown`、Provider Origin/アルゴリズム/対象集合・版を検証するまで実行しない |
| **MA-08b Blank/stale fingerprint** | 取得時点・Schema内容が異なるのに`Fingerprint=""`、または旧`fp1`を再提出 | 空欄を検証済み最新値として扱わない。**計画候補の生成**と**変更前提が現在も成立すること**を区別。申告値一致をABA/Grant現在性の証拠にしない |
| **MA-09 Partial schema looks absent** | 独立Expectedに既存表`public.audit_log`を固定。Readerの部分集合`Tables=[public.orders]`を`-actual`へ渡す。desiredには両表あり | `Compare`が`CreateTable(public.audit_log)`候補を作り得ても、**実DBに表がないという負の事実は未確定**。全集合を証明できるReader/Scopeで再観測するまでG1適用停止 |
| **MA-10 Same input, different Origin** | `Database=sales_core`、`Tables`、`Fingerprint`が同じJSONを、異なる実Instance/Tenantやoffline cacheから提出 | ファイル内容の同一性・Logical DB名一致は実Target束縛を証明しない。Provider/Connection/Scope/Read epoch/Authorityと最終消費者を明示するまで`unknown`（既存XH-01/03の深化） |

**同一Fixtureでの観測列（新Core型ではない）**：
- `declared`：要求したLogical Database、実行目的/要求Scope、計画Source/Mapping、`BasedOnActualFingerprint`の申告値とその**定義済み意味**。
- `observed`：実際に読んだConnection/Instance/Provider、Readerと版、取得権限・Scope・全件性、`CapturedAt`とWatermark、保存した入力ファイルの**実消費byte digest**。読み取りの成功・不成功を別記。
- `computed`：Providerが定義した算法/対象領域から再算出または証明できる場合のみ検証済みSnapshot fingerprint。**入力ファイルのhash**と**Actual Schemaの完全・現在のhash**を同一Claimにしない。
- `accepted`：G0の計画用入力として使用可か、G1のApply Admissionとして十分か、Owner/Host Grantが現在有効かの判断を**別列**にする。`unknown`や不完全読取はPlan候補の存在で解消しない。

これはOKFの`Attested Computation`で強調する**実入力・計算内容・観測可能な結果・検証範囲の分離**を、#1163既存Evidence Packetと#2600の入力鮮度に適用する案である。新しい署名基盤、Attestation Registry、全Provider共通Fingerprint算法を設けない。計算記録があっても業務MeaningのOwner受理・実DB権限・現在Grantは得られない。また、[merged #3060](https://github.com/hat47x/tei-platform/pull/3060)の宣言/実行`Differences`保持は**X-TEA参照実装の限定証拠**であり、Migrationの実Provider適用を合格にする根拠ではない。

**最小の試験順（新Runnerを作らない）**：①`physical.SchemaSnapshot.Normalize`/ `schemadiff.Compare`/ `migration.Plan`の既存APIでMA-08aとMA-09の**現挙動**を特性化（そのGo PASSは安全な適用の証明ではない）→②同じExpectedをProvider/Reader/実Targetの上位Admissionへ渡し、不適格なら`executeCalls=0`とする→③使い捨てPostgreSQLの完全/部分・別Instance・実変更による独立ReadをG1〜G3へ追加する。第一段のGo fixtureで実Reader完全性・実Provider効果を実証したと主張しない。

**担当と採否**：#2622は実Provider観測のScope/Capability、#2611はDB Alias/実Instance、#2600/#1163は実消費入力/証拠束縛、#2555は移行進行とCutover、#2514は独立期待値を既存の責任範囲として保持する。すでに存在するMA-01/02、HIST-03、XH-01/03へ統合し、新Gate番号や独立Registryは設けない。本節はDocs提案・Source-static。MA-08〜10のGo/PG/Owner試験は**NOT RUN**。

<!-- tei-db-migration-KOB-binding-20261107 -->
### 20.8 適用前拒否の実装位置とReceipt／現在状態／業務検収の非合成（2026-11-07、横断成熟差分）

**§20.3 MA-01/02のソース上の判別をGo特性試験へ移した**。`internal/database/migration/postgres_apply_test.go`の`TestMA01MA02NonDestructiveApplyCharacterization`は、単一`ADD COLUMN`の固定`Definition`について、①`BasedOnActualFingerprint`とfresh `SchemaObservation.Snapshot.Fingerprint`が違う、②`Definition.Database`と`SchemaObservation.Snapshot.Database`が違う、という2例を`ApplyPostgresWithFreshObservation`へ供給し、**現関数ではExecutor Callbackが一度呼ばれ得る**ことを特徴付ける。パーティション削除/制約追加がないとProviderの局所Readiness関数はDB ID確認より前で返り、fresh SQL文列は一致する、というソース上の経路。**現在の望ましくない挙動を固定するテストであり、安全合格ではない**。Go実行と実DBは別の検証。

**最終の拒否条件を置く場所**：単なる`schemaDiffFromDefinition`や`RenderMigrationSQLWithObservation`の検査に任せず、実Connectionを所有するHost/ProviderのApply入口で`Plan Source/actual base/Target Scope/Provider/Current Grant`を相関させ、受入不能なら**実Executor呼出しゼロ**を独立Expectedとする。既存の`PostgresObservationFunc`と`PostgresStatementExecutor`が**同一Session・同一物理Instance・同じロック対象を指す証拠は現署名にはない**ため、双方の入力文字列を検査するだけでも不十分。現行CLIがJSONから作るoffline Planを正式Applyへ直接接続しない（§20.6）。Provider保証なしに一般化した`migration.Apply` API/RunnerやCapabilityを追加しない。

**他チャットの新しいK/O/B分離をMigrationへ限定して接続する**：[merged #3127](https://github.com/hat47x/tei-platform/pull/3127)が提供した`effectcommit`のReceipt Lookupは**同じOperationに対する確定記録の存在 K**を照会する。`ResourceReader.ReadResource`は**資源の現在値 O**を別接続で読む別Capabilityであり、`Business Owner`によるデータ変換・履歴・使用可能性の判定 **B**ではない。これらをMigrationへ流用する場合も、同じResourceでないSchema/Rowset、旧新DB、CDC、外部Consumerへ一律に移植できるわけではない。`effectcommit`の単一DB Receiptがあることを、別のMigration DDLやBackfillが成功した証拠としない。[Draft #3129](https://github.com/hat47x/tei-platform/pull/3129)のFunction→実Provider Read／Command→Receipt→独立Row Readの段階分離も同じ上限で参照する。

| 既存G0〜G5のClaim | 要求する独立観測 | 禁止する短絡 |
| --- | --- | --- |
| **K: History/Receipt recorded**（G1/G2の記録、G4のCommand） | Migration ID/Definition・実Target・Toolの履歴取得資格、または同一OperationのProvider Receiptと真の対象Scope | `RepoApplied`・Receipt存在だけで**現在Schema/Data一致**、Reader完全性、別DBの移送済みとしない。履歴欠落を作用なしとも推定しない |
| **O: Actual State observed**（G1〜G3・G4/G5の再検証） | 別Connectionで実Schema／Data/Key/Historyを取得し、必要集合の完全性・RLS/Privilege・観測時点/Watermark・Replica lag/Writer epochを資格化 | 一度観測したt1の一致を変更後t2の**現在資格**へ再利用しない。Replicaが遅延している場合、直後の不一致をRollbackや効果ゼロと断定しない |
| **B: Business meaning accepted**（G3〜G5） | 必要なTenant・業務Key/物理PK・変換版/有効期間・金額精度・旧新Client/Writer/外部ConsumerとOwner判断を同一Caseへ対応付け | SQL一致／行数一致／K+Oだけで業務上の同値性やCutover・破棄承認を推認しない（既存MG-05/06/07/24〜28） |

**既存ケースの再利用（新しいケース台帳を増やさない）**：
1. **MA-01/02**を`actual base/target mismatch → executeCalls=0`という上位Provider適用Gateの独立Expectedへ、現在の関数単体の特性試験と**別実装**で接続。
2. **MA-07b/c**でCommit/DDLの後にHistory WriterやCLI ACKだけが失敗した場合、K未取得/unknownでもOで実際の結果を再確認し、同一Migration/AttemptでRecovery。無根拠に新IDで再送しない。Oが未確認なら`unknown`維持。
3. **MG-05/06/07・XH-01/02**で同じReceipt・同じ件数だが別Tenant/別Epoch/履歴欠落、またはRead後にWriterが更新した反例を同一Expectedで再利用。`K historical pass`、`O at t1 pass`、`O current unknown`、`B pending`を別欄に記録。
4. **§20.7 MA-08/09**でFingerprintの申告値や部分Schema観測を入力にした場合、`K/O`以前に適用前`A: Apply Admission`が`unknown`。後段のReceiptなどで**誤Targetへの実行前承認があった**ことに後付けしない。

**実装・検収の次の一手**：#2611/#2622で実接続先の束縛と単一Callerの責務を固定し、`MA-01/02`の現在挙動のGo PASSと**期待されるfail-closed上位Adapterテスト**を明確に分離。#1163は同一Attempt/Source/実消費SQL/Receipt/Reader/Expectedを収容する既存Packet、#1174は実Effect/Unknown、#2892は履歴の取得・欠落、#2555は旧新Writer/CDC・Cutoverの各Ownerを維持する。新しいReceipt Store、Business Validator、Attestation Registryを作らない。

**資格の上限**：今回のGo特性試験はソースを追加・保存した段階。実行したGo/Provider/DBの結果、正式Migration Apply Host、旧新Client/CDC、Owner受理は**NOT RUN/NOT ACCEPTED**。#3127のmain採用と局所Unit成功は、Migrationの統合実証には合算しない。

<!-- tei-db-migration-workbench-plugin-qualification-20261107 -->
### 20.9 RQ-035 Data Source Workbench・Plugin-first・テスト設計への合流（2026-11-07、横断決定の適用範囲）

[main統合済み #3276](https://github.com/hat47x/tei-platform/pull/3276)の[RQ-035/Workbench](DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)は、一般開発者によるSQL/NoSQL/ログ・クラウドデータソースの構造/データ管理に**Schema CREATE/ALTER/DROPを標準機能として含める**。DBA領域（DBサーバー/クラスタ/バックアップ/アカウント運用）を標準へ広げない。これに従い、Migration G0〜G5はWorkbenchの**Schema Apply経路の安全な部分**として再利用するが、RDB向けのトランザクション・履歴・正本切替条件を全Providerへ一律適用しない。SQL/DDL未対応Provider、サンプリング/ページングのみのSourceは`unsupported`または`unknown`のまま、読取可能な範囲を正確に返す。既存のWorkbench WB-01〜25とは二重の受入台帳を作らない。

[main統合済み #3277](https://github.com/hat47x/tei-platform/pull/3277)はCLI・Studioを含むCore以外のPlugin-first実装を製品要求とした。ただし**要求/設計の統合でありDB Migration RunnerのPlugin化は未実装**。実Schema操作については、CLI/MCP/Studio/生成Appが別のSchema意味・Authority・Provider Resolverを持たず、**同じData Operation ContractとProvider実能力**に到達することを目指す。CLIの`plan-migration`が現状offline JSONを読むこと（§20.6）を維持し、実適用をCLI手書き分岐だけへ追加しない。

| 既存能力・段階 | 必須入力/検証（§20.2 P/A/Eを再利用） | 統合時の独立Expected |
| --- | --- | --- |
| **Read/Reverse（Workbench W0/W1、G0）** | Readerが要求Scope/実取得Scope/開示Scope・全件性/時点/権限を報告、構造の確定/候補/未取得を分離。PostgreSQL実読取→Data Context→標準表現型のread-only縦切りは#2586を先行 | `SchemaSnapshot`の`Tables`欠落を実不存在としない（MA-09）。CLI/Studio/MCPの同じSource/Scopeは同じ候補と診断。未観測/非開示を空集合へ変えない |
| **Plan/Preview（G0、Workbench W1/W2）** | Accepted Data Structureと実Snapshot、Provider/Realization・SQL bytes、変更対象・破壊性/費用、差分`Unresolved`、Context/Source revision | MA-08の申告Fingerprint一致をActual資格へ使わず、モデルと観測の時点・能力が異なれば`not-admitted`。Previewは権限ではない |
| **Admit→Apply（G1/G2/G5、Workbench W2）** | 実Instance/Alias/Tenant・Environment/Grant/Provider Trait（DDL/Tx/Lock/NonTx/partial operation）の**現在資格**、同一対象への再観測と適用の束縛 | MA-01/02で不一致・未取得・未対応なら`executeCalls=0`。UIからもAI/CLIからも同じProvider拒否、ログやNoSQLにDDLを捏造しない |
| **Observe→Recover（G1〜G5）** | 実発行文/Statement単位Result、History/Receipt(K)、別接続Actual Schema/Data(O)、業務Meaning/Retention/Old Client(B)を別主張、部分効果・ACK lossを記録 | MA-05/07でEffect unknownをClean Rejected/自動再送へ落とさず、実ReadとOperation identityでRecovery。#3155の`persisted:true`の意味は**Resource観測に限定**し、Migrationへ自動昇格させない |

**Data Scope/Plugin/BOM/Cacheの横断境界**：Session/User/Tenantは保存先の物理共有状態と異なり、`Scope + Authority + Lifecycle + Consistency`が操作ごとに要る。Traitは「能力の宣言」と「現Targetで実証した能力」を分離し、特定DB製品名・Feature名のみで適用権限を付与しない。Version組合せBOM（#3269/#3267）、Source/Artifact Cache（#2600/#3271）は**検証済み組合せ・生成物の再利用資格**であり、現在のTarget Schema、Grant、Writer EpochやDDL許可を保証しない。変更後の適用可否は必ず再判定する。Pluginの版違いでSQL実現が変わる場合は§19.10のH2を再評価し、Meaning H1とは別票で示す。

**テスト・ハーネスへの実装接続**：既存#3273のDesign Context→未カバー要件→独立Expected→CLI実行と#2514/#1180のScenarioを、WorkbenchのWB-15〜25と本計画MA-01/02/05/07/08/09に**同じFixture**で接続する。C0は`inspect→plan→admit→apply→observe→recover`の各Node実到達、C1は`stale/partial/unsupported/denied/unknown/target-mismatch/ack-lost`分岐で、必要母集団が分からなければ率を出さない。Provider read-only契約と正式Apply契約は分離し、実Connectionのない合成FakeからG1 PASSを作らない。生成AIへのCLI/MCP応答にはExpectedとObservedと未解決の必要項目を明示するが、別のScenario Engine/証拠Storeは設けない。

**短期の実装順**：①#2586でPostgreSQL実Reader→Data Context→read-only表現の最短縦切りを先行、②#2611/#2622で対象Instance/Capabilityの現在資格とMA-01/02の`executeCalls=0`を実Adapterで検証、③使い捨てDBのSchema CREATE/ALTER/DROP→別接続Read/History・非Tx/partial負例、④実行できた契約をCLI/MCP/Studioから同一Providerへ投影、⑤旧新Writer・CDC/Retentionを伴うG3〜G5は#2555の独立Cutover・Owner承認に留める。既存WB計画の段階を置換せず、本PRでのPlugin化・データソース拡充・実DDL・Owner受理は**NOT RUN**。

<!-- tei-db-migration-catalog-resolver-matrix-20261108 -->
### 20.10 Plugin選択→実Catalog→三視点候補→Schema変更の受入閉包（2026-11-08、他レーン差分の採否）

前節のWorkbench/Plugin-first要求を既存コードに照らし、**「対応技術の宣言が選ばれた」ことを「実Providerを使える」「現在のDDL実行権限がある」と誤昇格させない**ことを新しいP0検収項目にする。[merged #3350](https://github.com/hat47x/tei-platform/pull/3350)はPlugin能力とDatabase Instanceの技術名を`SupportsDatabaseProvider`で対応付ける参照測定を改善し、`selected`/ `unselected`の集計を正した。ただし実Providerの実装、実Instanceへの接続、権限、操作・結果はこの測定に含まない。既存`internal/database/providers/postgres/provider.go`の`Capabilities`は`MetadataRead/SchemaChange/Transaction`等を返すが、**対象テーブルへの現在Grant・ロック/非Tx DDL・実Snapshot完全性・処理済みEffectの証明ではない**。Resolverを増設せず、既存#951/#2622の責務で区別する。

[Draft #3372](https://github.com/hat47x/tei-platform/pull/3372)は、物理Catalogの`FK/UNIQUE/Index`、`CapturedAt/Fingerprint`をread-only Workbench `SchemaView`へ落とさず投影し、`Physical Observation → Structural Candidate → Accepted Data/Function/Business → Diagram/CRUD/SHO`をつなぐ**未統合**変更。実SchemaのFK/UNIQUEは業務Relation/Keyの受理ではなく、逆にFKが観測されないことは業務Relationの不在でもない。このSource ObservationをそのままMigrationのDiff/Apply Admissionへ渡す場合も、**表示Projectionのフィールドが増えても、Instance/Scope/Readerの取得完全性・権限・世代が保証されたことにはならない**（MA-08/09、XH-01）。業務意味から宣言した希望Schemaと、物理Catalogから観測したActual Schemaを混同しない。

**既存Fixtureに追加する最小の「同じInputで2つのConsumer」検収**：

| 入力・変更 | Workbench/Reverse側（#3270/#3372） | Migration側（#2887/#2611/#2622） | 必須の非推論 |
| --- | --- | --- | --- |
| **MA-09a：同一Logical DBでpartial catalog**、FK/UNIQUE・表の一部がReaderから欠落 | 取得済み物理制約だけをSource Candidateとして表示、欠落は`unknown/not-observed` | DesiredにあってActualから欠けた表が`CreateTable`候補になり得ても適用しない（`executeCalls=0`） | 表・関係が無い、または自由にCREATEできるとは言わない |
| **MA-02a：`selected` Provider技術は一致、実Instance/Aliasが違う** | 宣言上のBinding解決と、実CatalogのConnection/Scopeを別に示す | 実ProviderとCurrent Grant/Actual Observationが計画のTargetに束縛されない限りAdmit拒否 | Plugin選択、Capabilities=true、Plan成功を実行許可へ昇格しない |
| **MA-01a：同じPlan/SQL、Catalog改訂世代・制約・Indexが変わる** | Observation Revisionと候補の失効を表示し、Accepted MeaningはOwner判断 | 変更後の依存影響・生成SQLを再審査し、未受理なら実行を止める | 申告Fingerprint一致や新Projectionの表示を現在Target保証へ使わない |
| **MG-05/07：物理FKあり、業務多重度・履歴・時点Relation未受理** | 物理関係と業務候補を区別して三視点/CRUDへ引き渡す | Schema変更が成功してもB（業務受理）とG3/G4はpending | FKと行数一致だけで業務上の整合性・Cutover完了にしない |

**テストの成立条件**：既存の合成Catalog→Introspect→Workbench Projectionの試験で確認できるのは、フィールド・ID/向き/多重度・出典時刻が**投影で消えないこと**だけ。実PostgreSQL独立Connectionの`Catalog Reader`→同じSnapshot/Scope→Migration Plan→不一致時拒否→DDL（隔離DBに限定）→別Connection再読取と、旧新Client・Owner確認は別経路。テストハーネスにはこの2 Consumerの必要項目（実取得列集合、欠落、Resolver selected、実Capability、Current Grant、Apply Attempt、Observed Diff）を**既存#1163 Packet**の別欄で返す。C0/C1の母集団が不明な場合は未評価とし、偽のカバレッジ率を出さない。

**短期の責務と停止線**：#951/#3350はResolver宣言選択、#3270/#3372は物理情報の非損失投影、#2611/#2622は実DB Target/Provider能力・Apply Admission、#2892はHistory Reader、#1163/#2514はExpectedと実入力の照合、#2555はCutoverのOwner。新しいPhysical Relation正本、Migration向けPlugin Resolver、Project-level History Registryは作らない。§20.8のMA-01/02特性試験は現挙動を特徴付けた段階であり、安全なAdapter拒否・実PG・SHO/Studio実利用・Owner受理は**NOT RUN**。

**統合に関する検証上限**：前HEAD `0675d019c9a3f19ff6edbc4538b229d37e52a2a4` のRepository Healthは**failure**（ほか6つのWorkflowはsuccess）。Jobログでは`internal/database/migration/postgres_apply_test.go`のgofmtと、`RQ-035`の要求マッピング2箇所、設計索引4箇所の不一致を特定。gofmt 1箇所は本PRの後続commit `882e99d5b52141953a01e25400a8c25c330be1b7`で整列し再読取済み。**RQ-035マッピングと索引4箇所は、その後の最新mainに記録が存在することを確認した**ためPR側で重複編集せず、最新mainの内容を損失なく合成する。新exact-headでCI完了するまでGoテスト合格は未認定。

<!-- tei-db-migration-apply-fence-rea-source-handoff-20261108 -->
### 20.11 観測直後の競合窓とReverse→Meaning→Migrationの証拠引継ぎ（2026-11-08、判別負例追加）

**現行Codeで分かったこと**：`internal/database/migration/postgres_apply.go`の`ApplyPostgresWithFreshObservation`は、`observe(ctx)`の結果でPostgreSQL SQL文列を再Renderし、その直後に別の`execute(ctx, statements)` Callbackを呼ぶ。両Callbackの署名は**同一DB Instance/Session/Transaction/Lock/Grant Epochを束縛しない**。従って、事前のSnapshotが正しく、`BasedOnActualFingerprint`と一致し、再RenderしたSQLが同一でも、**観測後・実Effect直前の別ActorによるSchema/Grant変更（TOCTOU）を防止した証拠にはならない**。一般のDDLにトランザクションがあることだけを根拠に、別接続のReadとExecutorが同一隔離境界にあるとは推論しない。

**新しい特性試験 MA-11（Source保存済み、実PG未検証）**：`internal/database/migration/postgres_apply_test.go`へ`TestMA11ObservationCallbackDoesNotFenceExecution`を追加。合成Callbackが最初に`fp-before`を返し、後続Executor Callback内でSchema世代を`fp-after`へ変更する。現関数が観測Callbackを一度呼び、Executor Callbackを一度呼び、エラーなく戻り得ることを**現挙動として固定する**。これは実DBの並行トランザクションを再現しないため、競合発生率・DDL実効果・ロックの安全性を証明しない。**独立Expected**：正式Apply経路ではTargetとGuardを固定した同一保護境界を検査し、保護できないなら`executeCalls=0`（または具体的なProvider保証条件を満たすまで停止）。Lockの粒度/取得順、接続Pool/DDLの非Tx条件、実Host Grantは#2611/#2622/#2555の実装責務。

**他チャットのReverse/SHO・REA議論の限定的な接続**：既存システムからのReverse成果は`Raw Source → Source Observation → Structural Candidate → Accepted Meaning → Desired Schema/Function/Business → Provider Realization → DB Effect → Independent Observation`の**異なる資格**を持つ。REAから出力した設計資料、観測PK/FK、テーブル同名一致、SHO図面やCRUD生成は、**物理スキーマ変更のOwner受理・移行前提・Target Grant・Cutoverの根拠にはならない**。Source位置/Revision/Scope/欠損・非開示・部分取得、候補採用のOwner/適用範囲/版、Generated SQLのTarget/Generator版を既存#1163 Evidence Packetへ**別欄**で結ぶ。原本変更やA→B→A、参照先移動・取得不能時は、候補/意味/Artifact/Applyの各資格を別に失効させる。外部のReverse連携が存在しても、実Targetへ適用されたことや旧新Client同値性は証明しない。**他プロダクトのリポジトリや独立データ基盤へここから設計判断を書き出さない**。

| 既存Case/工程 | 入力・対立条件 | この段階での独立Expected |
| --- | --- | --- |
| **MA-11／G1 Apply Admission** | 直前Readは`fp-before`、別ActorがRead後〜Effect前に`fp-after`へ変更。観測CallbackとExecutorのConnection/Lock/Grant束縛は未提示 | `not-admitted/unknown`のまま、実Execを許容した証明を出さない。使い捨てPGの2接続・同期Barrier・同一/別Tx・DDL/Lock負例で`executeCalls=0`または実競合拒否を独立に実証する |
| **MA-09・RE-13/14/15／G0** | REA/WorkbenchのCatalog Readerが部分集合・途中切断・異ProviderのSourceを返し、同名テーブル/FKが一部だけ存在 | Rawの部分性、ObservationのScope、Candidateのunknownを保持し、空/不存在を推定しない。Schema DiffのPlan候補はApplyの許可ではない |
| **MG-05/07・RE-16/17/18／G2〜G5** | Reverse原本やAccepted Meaningが改版・失効し、古いProjection/Artifact/Cacheまたは異時点のSchema Readが混入 | Source/Meaning/Generator/Provider/Actualの版を独立に拘束し、旧成果物一致/SQL一致だけではCurrent Schemaと業務意味の合格にしない。Owner受理・旧新Client/CDCは別 |
| **MA-07／G1〜G3** | DDL Effectの後History Writer/CLI ACKに失敗。Receipt Kが不明、観測Oも未完了 | 実Effect unknownを維持し同Attemptを再読。誤った新ID再送/重複DDLを禁止。Receiptの有無を現在Schema Oや業務Bへ昇格しない |

**実装に進むための狭い契約**（新Engine/Registryを作らない）：①Providerを**宣言Selection→実接続Capability→Current Target/Grant→Execution**の順で確認し、②`Observation`と`Execute`の間に成立した保証（同接続のLock/Tx/DDL制約/対象Fingerprint再読取/Target Epoch）を**実Host側で観測できる証拠**として残し、③非Tx・部分副作用・Reader errorでfail-closedにする。単なる`fingerprint==`ではABAやCatalog完全性を証明できない。④同じFixtureをWorkbench ReverseとMigrationの**二Consumer**で走らせ、読取Projection/候補とApply Admissionの結果を別Expectとする。テスト・C0/C1・BOMは#1180/#2514/#3267の既存仕組みを使う。

**今回のexact-head CI根拠**：前HEAD `2f7e2a11cf7b37e6741fa63710dd18e8ada5a709` のGitHub Actionsは7件中6件success、Repository Healthのみfailure。Jobログで`go vet ./...`、`go test -count=1 -timeout 60m ./...`、`tei validate -workspace reference/workspace`は**PASS**。失敗要因は`tools/reference-plugin-capability-coverage/coverage_test.go`の`gofmt`のみで、Migration Test failureではない。このGo fileはその後の`main`のBlobが異なり、**当時のHEADと新mainのCIを混同せず、最新main合流後のexact-head CIを再確認**する。追加MA-11単体・現行本番Target・PG実Read/Apply・Meaning/Ownerは**NOT RUN/NOT ACCEPTED**。
