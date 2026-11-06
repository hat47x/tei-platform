# Script起動の失敗状態・再実行・回復設計（2026-11）

- **状態：評価待ち（未採用）／文書のみ。** 本文のS0〜S8、F分類、SH fixtureは**説明用の判定票**であり、正式状態enum・公開ABI・中央Runnerの新設ではない。
- **目的**：同一の意味Function/ScriptをUI、CLI、cron、Web、既存Hostから呼ぶ際、Source選択から実消費版、意味診断、実行許可、副作用、独立観測までを誤成功なく接続する。常駐TEI Hostは前提としない（[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)）。
- **既存責務**：[Everything is Data #2619](https://github.com/hat47x/tei-platform/issues/2619)／[Source・DSL #988/#2595](https://github.com/hat47x/tei-platform/issues/2595)／[Data Operations #2622](https://github.com/hat47x/tei-platform/issues/2622)／[Data Rule #1129](https://github.com/hat47x/tei-platform/issues/1129)／[Script製品 #2502](https://github.com/hat47x/tei-platform/issues/2502)／[無人Run #1130](https://github.com/hat47x/tei-platform/issues/1130)／[Effect Fence #1174](https://github.com/hat47x/tei-platform/issues/1174)／[Scenario #1180](https://github.com/hat47x/tei-platform/issues/1180)／[Artifact #943](https://github.com/hat47x/tei-platform/issues/943)。副作用段階・再試行の**詳細正本候補**は[業務副作用の設計](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md)、無人Run/Stepは[RunPlan設計](RUN_PLAN_AND_UNATTENDED_EXECUTION_DESIGN_2026-11.md)、Correlationは[実行相関](EXECUTION_CORRELATION_DESIGN_2026-10.md)。本書は**跨る受渡しの欠落と停止条件だけ**を扱い、内容を複製しない。
- **既知の部分実装と限界**（2026-11-05時点の照合）：mainの`internal/execution/runplan.go`はunattendedでpreconditionsがnilなら当該Stepを止め、`EffectPreconditions`はobserverの**存在だけ**を確認し、実際の`ObserveEffect`を呼ばない。途中でblockされても先行StepのEffectは残り得る。mainに統合済み[#2853](https://github.com/hat47x/tei-platform/pull/2853)は`CommitUncertainError`をCommandの`unknown`へ伝播し、PG限定Fixtureでsame-attempt再送拒否とReceipt Lookupを示す。[#2854](https://github.com/hat47x/tei-platform/pull/2854)では**別Idempotency KeyによるABA後の再適用**が観測された。これは全Providerの回復保証でも実Source→実Script→Host/DBの一気通貫受入でもない。

## 1. 二つの独立した判断：起動資格と効果の確定

通常のScript構文に実行状態機械の言語要素を強制しない。業務の状態（注文・請求・組織の状態）はData/Function/RuleとOwnerの意味定義を正本とし、ここで扱うのは**起動工程と効果確認の診断**である。IFで十分な業務ロジックに専用StateMachine Pluginを強要しない。

```text
S0 Sourceを選択
  → S1 Source/Referenceと実消費対象の版を固定
  → S2 Parse/Link（型・Operator/Plugin・依存・Native資産）
  → S3 Meaning/Context/Ruleを検証（必要なOwner受理も確認）
  → S4 Target/Provider Capabilityと現在実行条件を評価
  → S5 Preview（作用・対象・変更前提を説明）
  → A0 Dispatch Admission（同一要求Identity・版・権限を再確認し記録）
  → S6 実Host/Providerの呼出・Effect Fence
  → S7 Receipt・各Effectの確定/不明を照会
  → S8 独立Read/業務の期待値・必要対象集合を検証
```

S0〜S5でエラーが起きたとき、Provider未呼出しが**検証できる**場合には起動の再試行ではなく編集・参照修復・再Previewを行う。一方、A0以降の接続断/timeout/killでは、**実際にEffectへ到達していないことを独立に証明できない限りunknownを残す**。Preview/表示されたボタン有効/認可UIの成功は、Provider開始・Commitを意味しない。

### 工程ごとの入口／出口条件

| 工程 | 次工程に進める最小条件 | 保存・固定する根拠 | 未達時 |
| --- | --- | --- | --- |
| S0 | Stable Script/Function ID、入口とSourceを一意に選択 | Source locator、origin、選択者／入口／理由 | 曖昧なら選び直し（推測で別名Scriptへfallbackしない） |
| S1 | 必要なSource/外部Referenceを解決し、**実Readerが読む予定のbyte**と範囲を固定／実際の消費と後で照合可能 | selected source + required closure、raw bytes/digest、revision、fresh/stale/partial/unavailable/permission、検証時点 | 未取得を空ファイルや依存なしとしない。A→B→A、Symlink、TOCTOUを区別 |
| S2 | 同じ固定入力で構文・型・Link/Operator/Provider候補が整合 | 出典span、依存/版、Parse診断、Native SQL選択/実SQL候補 | 別方言/Providerへの黙示fallbackなし |
| S3 | Data/Function/Ruleの意味と明示Context Mapping、NULL/欠落/部分取得/粒度を確認 | Meaning ID/revision、Context/対象、必要なOwner Acceptance、Rule effect phase | AI候補・型一致・Parse成功を意味受理にしない |
| S4 | 必須CapabilityとTarget/Host/Providerが適合。現在Principal/Grant/Scopeは独立確認 | 必要能力と保証範囲、Execution Context slot、選択Provider・Artifact | unsupportedとdeniedとunknownを分ける。Token等の秘匿情報は保存しない |
| S5 | 実行対象・効果・非対話性・未確定事項を利用者へ説明 | 入力hash、expected version、read/write/effect set、Preview版・時点・Audience | stale Previewの承認を実行許可としない |
| A0 | **副作用前**に要求ID/入力・効果対象/Grant/実ロード版の一致を評価し、必要な耐久記録が成立 | logical Operation ID、Request digest、Run/Step/Attempt、実Artifact/Source/Target、admission結果 | admissionの永続性・現在資格が必要なProviderで未達なら呼ばない |
| S6 | Providerが現在Grant/対象/Versionを必要なEffect確定境界で強制 | 実dispatch、effect/tx、Provider response、source/byte/契約のidentity | timeout・ack lossの成否を推定せず照会に進む |
| S7 | Provider ReceiptとOperation・対象の一致／対象Effectのunknownを列挙 | Commit/rolled back/rejected/partial/unknown、Receipt lookup、after-commit配送 | 不能ならunknown/partialで停止・回復待ち |
| S8 | **独立Consumer**が指定Snapshot/Scope/期間/完全性でExpectedを検証 | actual read、必要対象集合、観測時点、品質/配送結果、開示資格 | 旧Replica/0件/部分読取の結果を業務完了へ昇格しない |

A0は独立した新しい中央Admission Serviceではなく、既存Host/Command/Providerによる実行境界を**可視化した論理チェックポイント**。S4とA0の両方を経ても、Commit時点で変わり得るGrant/対象Versionは[#1174](https://github.com/hat47x/tei-platform/issues/1174)のFenceで強制する。純粋計算やread-only ScriptにはCommitを要求せず、必要な結果同値とRead Scopeを検証する。

## 2. 単一の成功状態に潰さない

最低限、工程進行（選択済／解析済／dispatch済／終了など）、外部Process（未起動／exit 0／非0／応答喪失）、実Provider Outcome（非到達が確証／拒否／確定／部分確定／unknown）、独立業務検証（未実施／PASS／MISMATCH／unverifiable）を**別軸**で残す。正式enumにはしない。

| 見えた事象 | 何が分かるか | まだ主張できないこと |
| --- | --- | --- |
| Previewに「実行可能」／ボタン有効 | その時点の診断や表示適用条件 | Dispatch承認、現在Grant、DB効果 |
| CLI exit=0／HTTP 200／EKI Task completed | 対象Process/Transportの終了・受付 | 必須入力配送、DB Commit、独立Expected |
| RunPlan Step `resolved`／RunLog `Completed` | 当該Step/Runの記録上の終了 | Providerの実Effect、after-commit配送、業務全体の完全性 |
| Providerのexact Operation ReceiptにCommitted | 当該Providerが保証する範囲のEffect確定 | 別Provider、通知、必要全対象の業務検証 |
| Replicaが古い値、または取得0件 | 観測したSource/Snapshot/Scopeでの読取結果 | Primaryで未Commit、母集団全件、対象不存在 |
| S8でExpected・必要全対象・鮮度が確認済み | **明示した業務完了述語の範囲**でBusinessVerified | 未検証の別Scope/別期間/外部Effect |

結果の資格は単調に上がるとは限らない。Source、Meaning、Provider、Rule、Grant、対象版、Consumerの現在資格は後に失効し得る。過去の結果とReceiptの履歴は保持し、**現在の再利用・表示・再実行資格**だけを再評価する。

## 3. 故障分類と安全な回復

| 故障とcut point | 記録して保持 | 可能な回復／再評価 | 禁止・停止条件 |
| --- | --- | --- | --- |
| F01 S0/S1 Source欠落・同名差・Reference権限失効 | 求めたID/URL・版、失敗区分、元の入力証拠 | 権限のある範囲で再解決→Source/Meaning再検証 | 最新同名へ黙って切替／古いCacheで実行 |
| F02 S1実消費入力差、A→B→A、部分Source closure | expected/actual bytes・reader/tool版、unverified閉包 | 不変bundle/新Loadでやり直し、実使用を再照合 | 前後hash一致だけで実行正当化 |
| F03 S2構文・Link・Dialect/Operator版不整合 | Source位置、選択実装/版、競合・unresolved | 修正Sourceを新revisionとして再Parse/Link | 別Provider/共通SQLに無断fallback |
| F04 S3意味未受理・Context不完全・Rule競合 | Meaning/Context/Rule/Nullability/partial根拠 | 意味Ownerの判断、明示Mappingの修復 | AI生成候補／型一致を正式受理 |
| F05 S4/S5 capability不足・Grant拒否・Preview stale | 必要能力、対象・版、非機密の拒否根拠 | 必要能力ある実行先と再Preview／Grant再取得 | UIでの許可や旧Grantの流用 |
| F06 A0以前の接続失敗・実送出が**証明上なし** | prepared request、送出否定の確実な証拠 | 新Admission後、同じ論理要求を送る | Trace不足なのに「未実行」と断定 |
| F07 A0以降のnetwork timeout／応答喪失 | Operation ID、Request digest、実呼出・Attempt、最終観測 | まず資格付きProvider Receipt Lookup→独立Read | 新Idempotency Key・新Operation IDで盲再送 |
| F08 単一Txの確定Rollback、業務拒否 | Txに属する効果の結果、拒否と別Effect履歴 | 入力修正後に新しい業務要求として再評価 | 先行別Txや外部通知まで巻き戻ったと推測 |
| F09 Step A確定後B失敗／複数Provider一部確定 | **Stepごと**のCommitted/Unknown/Unstarted・Effect IDs | 未実行部分の資格付き再開、必要なら明示補償 | Run全体failed→effect 0と表示／全件再実行 |
| F10 Commit ACK-loss | 同一OperationID、元Request hash、Fence/attempt、commit不明 | 同じ操作IDでReceipt照会・Provider固有の引継ぎ | Unknownをfailed/未実行と変換、無条件replay |
| F11 Commit済み、Outbox/通知失敗 | 確定Receipt、通知Event ID、配送attempt | 通知だけの資格付き再配送／Recipient重複排除 | DBを再Commit、通知の未配送を業務DB rollback扱い |
| F12 crash・再起動・Journal欠落・旧Artifact | durable記録の範囲、実load版、最後のStepとEffect、Owner/Fence | 先に現所有資格取得→既Effect照会→必要部分だけ再開 | record `completed`だけでskip／旧実行を最新版へ自動付替 |
| F13 Observer存在するが該当Effectを未観測 | observer契約、対象effect/Scope/時点、未読範囲 | 実Observerで対象Effectを照会し独立照合 | `observer != nil`だけで「観測済み」 |
| F14 S8 独立Readで旧Replica・0件・部分取得 | 対象・snapshot/watermark・完全性、Expected/Actual | 期間とReplica lagを条件に再Read、結果矛盾は調査 | 空集合＝無効更新、読取一件＝全件完成と推論 |

ここで「処理をやり直す」と「**同じ業務操作の結果を調べ直す**」は異なる。結果unknown時は**再送よりLookup優先**。Receiptの保存/保持期限を越えたら安全な回復を断言しない。Compensationは新たな意味を持つ業務操作であり、RollbackやRetryと同義でなく、補償元Operationと対象Effectが確定しているか確認してから実施する。

### 再起動の決定手順（設計上の処理順）

1. 現在の実行主体が、対象Run/Operationを**所有・照会してよいか**Host/Providerで確認。必要なら既存Lease/Fenceで所有を引き継ぐ。Journal上のOwner文字列は権限でない。
2. 旧Source/Meaning/Rule/Artifact/Provider/Context・Request hashを読取り、現在版との互換を判定。旧版を再ロードできない場合やAdapter差が未確認なら停止する。
3. **効果の無い準備Step**は必要な範囲で再評価できる。Effectに到達した可能性があるStepは、既存Receipt/対象状態/Outboxを照会。Journalに`resolved`があってもEffectの確定証拠がなければskip資格はunknown。
4. 確定済みStepは二重Effectを起こさず保持。未送出が確証されたStepだけ、現在Grant/対象/CASの資格でDispatch可。EffectがunknownのStepは同OperationのLookupを優先し、任意に新Keyを発行しない。
5. 部分確定・外部Effect未配送・補償待ちは別々に残す。業務完了のClaimは、Owner定義の必要対象集合・独立Expected・Source/Snapshot鮮度の確認後に限る。

**Durable証拠がない場合**：「自動再開可能」ではなく`manual-reconciliation / unsupported`相当の診断を返す。永久的な中央Job Registryを新設するのではなく、選択Host/Providerが実際に保持できる記録と責務に合わせて実装する。

## 4. 意味、Script、Data Rule、Job、Buildの境界

- **Script/Function**：型付きの業務操作と分岐・明示Data Contextを表す。純粋IF/高階演算を基本とし、業務の状態遷移定義は保守価値がある場合だけ選択する。起動工程の状態機械を業務テーブルの共通State列へ押し込まない（業務状態は必要なら元のEntity項目／複数属性／JSONB等に保持し、中央共通State tableを必須にしない）。
- **Data Rule (#1129)**：pure derivation／validation／before-write／同一Tx内の効果／after-commit／compensationの**意味とphase**。Hook単体の発火を、Committed Business Eventや実通知完了へ昇格しない。
- **Data Operations/Provider (#2622)**：明示Relation/Field/Scope Mapping、native SQL/Dialectの選択、実Read/Write、Target/CAS/partial/unknown。RDB、OData、他APIで「batch＝atomic」「全置換＝PATCH」と仮定せず、能力不足なら止める。
- **Invocation/Interaction (#2502/#2507)**：ボタン、CLI、cron、Web、EKI要求などの入口から同じFunctionを選ぶ。対話必須Scriptを無人経路で実行するには事前入力または明示Suspend/Resume契約が必要。TxをUIの返答待ちで保持しない。
- **Build/Deployment（[Draft #2788](https://github.com/hat47x/tei-platform/pull/2788)）**：Go単一Binaryを既定とし、必要時だけFunction→Build Unitを明示。含有Function≠外部Entry公開≠現在Grant。crash時に旧Runの実Artifactを新Binaryへ暗黙移行しない。
- **実HostとEKI (#1130)**：認証/資源・サービス主体・排他・実送出は選択Host、EKIではTask/Lease/Workerの担当。EKI P0の`Task completed`はTEI Commit/BusinessVerifiedでない。引数・必要Effect・Result body・Executor版のTransport適合を確認できなければDelegateしない。

## 5. Fixture統合：第二のScenario正本を作らない

初回は合成の受注確定FunctionとOrder A一件、Data Ruleひとつ、Post-commit通知ひとつを使う。同じIndependent ExpectedをScript/CLI/無人Runの**実在する入口**で比較し、実験のための新Host/DSL/Adapterを先行しない。

| Case | 差し込む故障・変更 | 独立Expectedと既存所有 |
| --- | --- | --- |
| SH-01 | 選択Source欠落、外部Reference stale、Operator版不一致 | S1/S2拒否、Provider呼出し0。#2595/#2639/#2619 |
| SH-02 | D1→D2→D1、未読込必須Source、SQL Asset/実読込差 | 全消費閉包の証拠不十分なら実行資格/skip資格unknown。#2595/#2600/#2622 |
| SH-03 | NULL/未取得/partial/0件、誤ったFunction Data Context | 不完全を完全と見なさず意味診断で停止。#2622/#2506/#1180 |
| SH-04 | Preview後に対象ID/expectedVersion/Grant/Script版を変更 | 古いPreviewと承認は無効、拒否時の実Effect=0をProviderで検証。#2499/#1174 |
| SH-05 | 必須Preconditionの**一部だけを渡す** | 組合せ関数のnon-nil＝必須検査網羅ではない。欠けたPeriod/Scopeを検出。#1130 ACU-01 |
| SH-06 | observerオブジェクトは存在するが対象Effectの結果は未取得 | preflightを実観測済みへ昇格させない。#1130/#1174 ACU-02 |
| SH-07 | Step AをCommit後、Step Bでblock | AのEffectを保持、Run全体`no effect`と判定しない。#1130 ACU-04/#1129 |
| SH-08 | Commit ACK-loss／同Attempt再送／引継ぎLookup | Command unknown、無条件再送拒否、同Operation Receiptと別Connection readを照合。既存PG #2853の限定ケース再利用、#1174 |
| SH-09 | 同一論理操作のまま新Key発行／ABA | 新Keyで再適用し得る負例を維持。#2854/#1174 |
| SH-10 | DB Commit後の外部通知が失敗、Process kill→再起動 | DB Receiptは保持、Outbox/通知だけを回復、旧Artifact版と残余Effectを照合。#1129/#1130/#2146 |
| SH-11 | 複数Provider A確定・B不明、または旧Replicaのみ読取 | Partial/unknown、独立ReadのSnapshot/完全性を分ける。#2622 DO-06/#1129/#2514 |
| SH-12 | 1 Binary→Web+Batch 2 Binary、実Entry/Provider版不一致 | 同Function意味・同Expected維持、実ロード版/Grant/Contract適合を再判定。#1130 BU-01〜09、Draft #2788 |

初回の**狭い垂直スライス**はSH-01/03（副作用前の拒否）→SH-07（先行効果保持）→SH-08（ACK-lossとReceipt）→SH-10（再起動と通知）の4系統。SH-08の既存PG合格はこの横断接続の新しいPASSではない。SH-10の実Restart/Outboxは現段階ではNOT RUN。

## 6. 導入順、出口条件、所有境界

| 工程 | 作業と検証根拠 | 不足があれば停止する条件 |
| --- | --- | --- |
| P0 読取り専用の証拠結合 | mainのScript/Function Source、実Go CLI/RunPlan、Effect Commit、受理済みMeaningの同一Revisionを照合。SHを既存Scenario/Oracleへ参照付け | 元Source／実Script入口／独立Expectedのいずれか不在 |
| P1 Headless合成 | Source/Meaning拒否、必須チェック欠落、先行Effect済みblock、Observer有無、ACK-lossの判別を既存Fixtureで実施 | Provider呼出0／Unknown／Partialが期待通り分かれない |
| P2 既存PGの実縦断 | 既存Command Invokerと実PG、Receipt照会と**別接続**独立Readを、同じOperation/Artifact/Rule版へ束縛。Restart/Outboxを後続 | 同一Tx・Owner Scope・現在Grant・実Load版が独立に確認できない |
| P3 UI/外部ジョブ | 同一FunctionをCLI/Web/cron相当・選択Hostへ公開し、必要に応じて1→2 Build比較、EKIはwire+実Worker適合を別評価 | 入口の到達だけでEffect保証、旧Jobの無条件乗換、対話Scriptの無人実行 |

**担当**：起動体験/Preview＝#2502/#2507、Source/Meaning＝#2619/#988/#2595、Mapping/Provider能力＝#2622/#951、Data Rule phase＝#1129、実Effect/Receipt/Current Grant＝#1174/#2499、Run/Step・再起動＝#1130/#2146、Build選択＝#1130/#943/#2788、独立Expected＝#1180/#2514。ホスト認証は[ADR-0044](../adr/0044-authentication-not-a-core-concern.md)に従う。

**完了主張の最低条件**：少なくとも同一Operation/Source/Artifact/Host/Providerが識別された一試行について、(1)実送出前停止を確認、(2)実Commitの確定・Rollback・Unknownの区別、(3)Partialを保持、(4)同IDの安全なLookup/再開と新ID誤再実行の反例、(5)実独立Readによる必要対象の完了判定が存在すること。Synthetic／実PG／実Web／実OS／実EKI／Windows GUI／Owner意味受理は別列のEvidenceであり相互流用しない。

**未決**：実Context/Referenceの鮮度期限、A0で何をどのHostが永続化できるか、Script Code/実ロードbyteの原子的Snapshot、補償のOwner判断、未検証の動的依存、Receipt保持期限と障害後Lookup資格、公開Scriptが対象のサービス認可モデル。欠けた保証を新しい中央Registry/Job Engineで隠さず、既存契約で満たせるか先に反証する。本書の新しいSH試験、製品Script全工程、実再起動/外部Effect、TEI↔EKI実Worker、Owner正式採用は**NOT RUN／未採用**。

## 7. 検査被覆・非同期応答・回復資格の接合（2026-11-06、追補）

**前節との差分**：以前のS0〜S8／A0とF01〜14は、故障が起きたら何を保持するかを定めた。本節は、**停止を解除してよいと誰が何で判断するか**に限定する。新しいEngine/State enum、中央Audit Store、第二Oracleを導入しない。現mainへの到達点は[merged #2882](https://github.com/hat47x/tei-platform/pull/2882)のWorkspace→`DeriveOperationSurface`→`PeriodPreconditions`／`EffectPreconditions`→`CombinePreconditions`→`RunPlan`という**合成Stepの実Goテスト**に更新する。Period closed/open、after-commit Observerなしでblockedを局所確認済み。一方、Guardの網羅性／実ProviderによるEffect強制／Observerが当該Effectを実観測したこと／実永続Run所有は未証明。既存[#1130 RPL-01〜05](https://github.com/hat47x/tei-platform/issues/1130)の検査正本を再利用する。

### 7.1 Required Guard CoverageとGuard Outcomeを二軸にする

各Action/Stepについて、**Required**（Accepted Function/Rule、対象Target/Effect、現在のExecution Context、選択Providerの契約から必要だと分かる条件）、**Supplied**（実Host/CallerがそのStepへ渡した検査）、**Evaluated**（実際に指定されたSnapshot/対象/主体で評価した検査）、**Enforced-at-Effect**（Provider側の更新・Commit境界で拘束された条件）を区別する。これは**既存Contract/Provider/RunPlanへの読取り専用の対照表**であって、Runtimeに新たなGuard Registryを実装することではない。

| 例 | Required | Supplied/Evaluatedの落とし穴 | 判定候補／理由 |
| --- | --- | --- | --- |
| 期間付Job | Period、現在の対象期間、Effectがあれば対象権限/版 | `CombinePreconditions`にEffect検査だけ渡しPeriod未登録、あるいは古いPeriod map | 既知の必須検査が欠ければ**起動停止**。期間を現在読めないならunknownで停止 |
| Post-commit通知 | 対象Effect・観測手段・配送結果の区別 | `EffectObserver != nil`だが対象Effect/Scope/時点の`ObserveEffect`が一度も行われていない | 起動前の観測可能性候補は残せても、**実効果の検証済み**とはしない |
| 更新Action | Resource ID、実Payload、Rule/Context、期待版、現在Grant・Effect Fence | UIで選んだOrder Aと実Adapter PayloadのOrder Bが異なる／Preview後に対象が変化 | 送出前に対象Bindingを拒否。Commit直前のGrant/CASは#1174に委譲 |
| 外部Provider/SQL | Data Source Alias、必要Read/Write/Tx、Native SQL Assetと実Reader | Dialect選択が成立しただけで実SQL byte/対象DBが未確認、全置換しかできないProviderをPATCH扱い | unknown/unsupportedで停止。無断SQL fallbackや全件read-writeへ転換しない |

ここで「Requiredの列挙が0件」は、**必要な検査がないことの証明**と同義でない。未解析Source、未解決のEffect Rule、外部Host固有Policyがある場合は`coverage=unknown`に留める。純粋・read-only Scriptに不要な書込みCASを強制しない。必要なCheckを発見しても、機密のPolicy／Hidden Rule／非公開Row IDを利用者へそのまま開示しない（[#2516](https://github.com/hat47x/tei-platform/issues/2516)）。

**合格条件案**：`Required ⊆ Supplied`だけでなく、検査のSubject/Context/期日/Source revisionが一致し、各`Evaluated`の結果が確認できること。その上で副作用については、既存Providerが`Enforced-at-Effect`を保証した範囲**だけ**を実行資格として採用する。全Requiredを列挙できないなら、欠損なしと推定せず保留する。これは#2882のlocal PASSを否定するのではなく、保証範囲を正確に狭める。

### 7.2 遅延応答・画面終了・再起動とExecution Identity

UI/CLI/Jobの実行要求は、表示/選択の更新世代（**Selection/Context Epoch**、正式APIではない）と、業務の**Operation ID**、Transport **Request ID**、実**Attempt/Fence 世代**を混同しない。UI側の二重クリックを同期ロックで防いでも、別Window・別Process・通信ACK-loss時の二重Effectを防げたことにはならない。逆に、画面を閉じたり通信が切れたりしても、Provider側の業務Operationを自動取消したことにはならない。

| 事象 | 保存すべき参照・状態 | 採用してよい結果 | 禁止すること |
| --- | --- | --- | --- |
| Selection A→B→A、旧非同期Preview到着 | Source/Meaning/Context epoch、Request/対象/Scope、旧Preview時点 | 旧応答は**履歴として保持可**。現在表示の操作可能状態へ採用するには現epochで再評価 | 対象値やSource hashが一致しただけで旧許可を復活 |
| Preview後にGrant/Period/Row version変化 | 操作Intentと当初の根拠、現時点の再検証不足 | Previewをstaleと表示。新A0/Provider Fenceの資格を別取得 | 古いボタン有効状態・旧PolicyでDispatch |
| UI閉鎖/CLI exit/OS再起動 | 元Operation ID・Request digest・送出境界・最後のEffect/Receipt | 履歴／照会待ち画面として復元、必要なら同OperationのLookup | `busy=false`・exit非0から「DB効果なし」を推論 |
| 別WorkerがRunを引継ぐ | Run key/Step、Attempt、Lease/Fence、実ロードArtifact、旧Receipt/Effect | 有効な所有・版・Provider契約の下で確定済みStepをskipし、未確定を照会 | 任意の新Keyで旧Stepを再送、旧binaryのOperationを互換未確認の新版へ差替 |
| Low-privilege Consumerが歴史を見る | 対象Audience、現在Disclosure Grant、許可された縮約Status | 権限内の履歴を提示、非公開部分をwithheldとする | 旧Receipt・内部Source path/Policy/存在件数・Secretを外部へ漏らす |

データを最後まで保持できることと、今そのデータを**再利用／開示／再実行してよいこと**は異なる。過去のEvidenceは原則不変の履歴として扱い、現在のGrantやSource/Context Revisionの失効で削除・上書きしない。古いConsumerへ新しいRaw Source／Provider内部参照を追加開示しない。

### 7.3 回復許可の判定は「未送出を証明」「照会可能」「効果判明」の順

回復時の**操作選択**は、Callerの一律リトライ設定でなく、Provider/Hostが実際に証明できる範囲に従う。以下は選択規則の対照票であり、新State enumではない。

| 観測済みの根拠 | 安全に選べる次動作 | それでも未保証なもの |
| --- | --- | --- |
| 実Dispatch前の拒否、**Provider側へ到達不能が確実** | 必要条件の修復、新Preview/Admission後に初回送出 | 任意の外部Generator/Toolが実行されなかったとは限らない。Tool起動前Admissionを別確認 |
| Dispatch到達の可能性あり、Receipt未取得 | **同じOperation/Request identityで結果照会を先行**。Lookup不能ならunknown維持 | timeout・Client Cancelを「未Commit」に読み替えない |
| Provider Receiptに確定・Replayとあり、対象/版/Hashが一致 | 当該Effectは再更新せず、残余Step/after-commitを分離して検証 | Receiptを別Providerや別対象へ転用しない |
| 既知のRollback／業務拒否、他に独立Effectがないことを確認 | 意味上の新要求か旧操作の再評価かをOwner規則で区別 | 外部通知や先行別TxのRollback |
| A確定、B unknown、C未開始 | Aは保持、BはLookup、CはBの解消・依存・現在Grant確認後に実行可否判断 | A/B/Cを一つのglobal committed/failedへ畳まない |
| Receipt取得期限切れ／Journal欠落／旧Artifact不在 | 手動照合・運用保留、再開不能ならunsupported | 値/Versionが同じだから未実行・安全再送と断定 |

**独立Readだけから元Operationの帰属を一意に逆算できるとは限らない。** 複数Actor、ABA、同じ最終値、read replica lagがあり得るため、Receipt/Idempotency record/Outbox/Operationごとの効果記録をできる限り一致させる。既存[#2853](https://github.com/hat47x/tei-platform/pull/2853)のPostgreSQL限定`CommitUncertainError`→Command Unknown/同Key Lookupは局所実証済み。一方[#2854](https://github.com/hat47x/tei-platform/pull/2854)の新Key+ABA再適用は**安全性未達の反例**。他ProviderやRunPlan再起動へ適用済みとは扱わない。

### 7.4 統合Evidence Packetは「主張ごと」の必要列を持つ

[#1163](https://github.com/hat47x/tei-platform/issues/1163)／[#1119](https://github.com/hat47x/tei-platform/issues/1119)の既存G1/G2受入票を使用し、**新しいScript専用Evidence DBを作らない**。各Caseの入力/Expectedは#1180/#2514を正本とする。各Claimの`Required`、`Observed`、`Undischarged`、`Claim ceiling`を少なくとも照合する。

| Claim | Required/観測列の候補 | ceilingの例 |
| --- | --- | --- |
| Source/Meaning qualified | Source/Reference revision、実Reader consumed bytes/closure、Operator/Plugin版、Accepted Meaning ID/Owner | 入力未取得・未解析なら**構文・意味合格不可** |
| Preview/Admission qualified | Function/Action ID、Context epoch、対象ID、Request hash、Required GuardとSupplied/Evaluated、current Grantの確認範囲 | Preview/表示成功はProvider開始資格でない |
| Execution/Effect qualified | 実ロードArtifact、実Provider+contract、Operation/Attempt/Fence、実Write/Tx/Receipt、具体Effect target | Synthetic step success／process exit 0だけなら**Commit unknown** |
| Business result qualified | 独立Expected、Read Scope/Snapshot、水位、対象集合完全性、after-commit Effect、Consumer actual body/disclosure | Replica0件、Readerエラー、Manifest Complete、ログ1行ではBusinessVerifiedでない |

局所テストの`PASS`は当該Fixture/Source/Tool/環境/HEADのClaimにだけ帰属させる。異なるコミットや合成/実PG/実Browserの結果を合算して`one-revision E2E`にしない。秘密のDSN、Raw Policy、未許可Source path/件数はEvidenceの**外部投影**へ複写しない。

### 7.5 既存Fixtureへ差し込む最小差分と実行順

| 既存Caseへ追加する反証 | 独立Expected（新ID・第二正本を作らない） | 引渡し |
| --- | --- | --- |
| SH-05／#1130 ACU-01：Period Ruleはあるが、`CombinePreconditions`にはEffect checkだけを渡す | checkが正常に返っても**Required Guardの差集合が残る**。追加の起動資格は保留 | #1130、#1180。#2882の実Workspace fixtureをbaselineに使用 |
| SH-06／ACU-02：Observerはnon-nil、別Scopeの古いResultだけ存在 | 事前の観測可能性判断と、当該Effectの実観測/Commit資格を分離 | #1174。実Observer/Scopeは未実施 |
| SH-04：画面A→B→A、旧PreviewのPromiseが遅れて返る | 現epoch以外の旧Previewは**現在の実行可否へ反映されない**。過去Receiptは権限内で参照可 | #2502/#2507/#2516。実React/Windowsは未実施 |
| SH-07/08：Step Aを確定後Bをblockし、さらにAのACK-loss | A=unknownまたはReceipt-confirmedを保ち、Run `blocked`でもno-effectと判定しない | #1130/#1174。既存実PG限定試験を使用 |
| SH-09/10：Process restart、元Operationが残るのに別Idempotency Keyで再送 | 新Keyの再適用危険を検出し、引継ぎ照会と新規要求を区別。旧Artifactの互換未確認なら停止 | #2146/#1130、#2854の負例を再利用 |
| SH-11/12：複数ProviderのA確定+B不明と、旧Batch Binaryを配備したWeb | Transaction/結果/契約/Consumer Bodyを分け、互換未確認ならpartial/unknown維持 | #2622/#2604/#2788、現EKIワイヤは別検証 |

**更新した段階計画**：

- **P0（最優先・既存Go Reference）**：merged #2882を起点に、同一WorkspaceのPeriod/Effect/Operationごとに**Required Guardの被覆表**を作成する。現`CombinePreconditions`が未登録Guardを検出する製品実装を持つとは仮定しない。Owner承認済みの必須Guardをどの契約から列挙できるかをまずread-onlyに確認。
- **P1（合成の強い反証）**：SH-04/05/06/07を既存Runner/Scenarioで故障注入。非呼出なら**対象Step呼出回数0**、先行A確定ならA Effect保持、旧Previewを現在許可に再利用しないことを個別Expectedとする。
- **P2（実PGの一改訂縦断）**：既存#2853/#2859/#1163のCommand Invoker→PG CAS→Receipt/別Connection readを再利用。実Schema gateの起動前停止／Commit時Grant・Target・Periodの再拘束／同Key ACK-loss／別Key ABA／結果Consumerを**同一AttemptとSource revision**で照合する。#2857の暫定E2E監査は`INCOMPLETE / productGate UNQUALIFIED`であり正式G1/G2合格としない。
- **P3（現実の再起動・外部Consumer）**：OS kill→restart、実Outbox/Provider、UI遅延応答・Grant失効、旧Artifact・2 Build・EKI委任を順次接続。別Host/Providerで証明不能な部分はunsupportedとし、新中央Host/Registry/Queueを自動採用しない。

**現時点の新しいSH系一気通貫試験／実UI Epoch／実Service Grant+Period+DB同一Commit／実二Process再起動／二Provider／Owner意味受理はNOT RUN。** 既存#2882局所Goの成立と、#2853実PGの限定Commit Unknownの成果だけを各Source版付きの既存Evidenceとして再利用する。

## 8. 永続回復の反例と一改訂受入の最短経路（2026-11-06、再照合）

### 8.1 既存の実証を追加し、過大な「回復できる」を禁止する

**2026-11-06再照合の新しい事実**：
- [main統合 #2870](https://github.com/hat47x/tei-platform/pull/2870)のRCAS-02は、旧`Lifecycle`に`unknown`があっても、新しい`Lifecycle`（再起動相当）では記録が無く`MayReplay=true`になることをGoテストで固定した。同じIdempotency Keyの実PG再送が`IDEMPOTENT_REPLAY`（適用0・Receipt 1件）となるのは**Provider側の永続記録**による限定的な保護。実OSプロセスの死・実ネットワーク遮断・二Worker/Hostの所有保証をテストしたわけではない。
- [main統合 #2854](https://github.com/hat47x/tei-platform/pull/2854)は、第三者により値が元へ戻ったABA後に**別KeyでAPPLIED・Receipt 2件**となる反例を実PGで固定。同じ値になったこと／新しいAttempt ID／同じ業務目的を自称することは、再実行安全性の証拠にならない。
- [Draft #2889](https://github.com/hat47x/tei-platform/pull/2889)では実Go `internal/execution.RunPlan` にToy Step/Journalを接続し、**Host Grant Guardを渡さなくてもcompleted、Observer存在だけで未観測、途中block前のStepが残る**というOC-RP-01〜03を固定。専用exact-HEAD [Actions #42100472921](https://github.com/hat47x/tei-platform/actions/runs/42100472921)はGo 3/3 PASS。一方、同headのRepository Healthはgofmt driftでFAILと[Draft #2783の再照合](https://github.com/hat47x/tei-platform/pull/2783)に記録されている。**専用試験PASS ≠ repository全体PASS ≠ Host Grant強制済み**。
- [Issue #2823](https://github.com/hat47x/tei-platform/issues/2823)では、宣言Source rootが存在せず／想定ファイルがローダの対象外でも、限定条件で`validate ok`になり得る実CLI反例が記録されている。`ok`だけで「必要なScript/Ruleをすべて解析した」と判定せず、`required / declared / discovered / loaded / parsed / actually consumed`を別確認する。
- [#2845](https://github.com/hat47x/tei-platform/issues/2845)は実運用のReceipt/観測記録の保存先・保持期間・検証・閲覧資格を未決のまま所有する。**digest-only記録は入力の復元ではなく、保存されていない記録の不在は未実行の証拠でない**。

### 8.2 再起動のcut point別に、誰の記録が復旧を許すかを固定

この対照表は状態機械の拡張ではなく、S5〜S8と既存#1130/#1174/#2564の間に配置する**回復許可表**である。

| cut point | 実際の未確定要素 | まず参照する証拠・Authority | 自動で行ってはいけないこと |
| --- | --- | --- | --- |
| C0：Previewまで、実送出なしが実証済み | Previewの現在性・Source/Grant/Target | 実入力版、現在のHost/Provider条件、実送出の不在根拠 | 旧Preview/旧Grantを復帰する |
| C1：HostがDispatch開始、Provider到達は不明 | 任意のWrite/外部送出の有無 | 固定Operation ID/Request digest、Provider側Lookup、実呼出記録 | エラー文字列・プロセス終了からNo Effect推定 |
| C2：DB Commitした可能性、ACK不達 | 確定/失敗の帰属 | **Providerの永続Operation Receiptと同じKey**、対象・Request digest、別ConnectionのRead | in-memory `Lifecycle`が空だから別Keyで再投入 |
| C3：A確定、Bの実行/Commitが不明 | 部分効果・残余Stepと依存 | AのReceipt、Bの同Operation Lookup、Run/Step Journal、実Owner/Fence | Run全体を最初から再実行／Journalの`completed`を業務完了に昇格 |
| C4：Commit確定、Outbox/通知は不明 | 確定後外部Effectの配送・重複 | DB Receipt、通知Event ID、対象Observer/Outbox、受領側の重複排除 | DBを再更新／Observerの存在だけで配送済み |
| C5：Receipt期限超過、記録欠落/破損、旧Artifact不在 | 効果の帰属・Replayability・Current Qualification | #2845の保存/照会資格、Provider/独立Readerの現観測、必要ならOwnerの手動判断 | 欠落を未実行とみなす／値一致・digest一致で自動再送 |

**Operation ID、Idempotency Key、Request Hash、Run Key、Attempt、Fence generationの同一性は推定変換しない**。どのIDをどのContract/Providerへ渡すか、当該Providerが同Key・同Payload・同Targetをどの保持期間まで判定できるかを実際に観測する。異なるIDの文字列が偶然一致しても、異なるScopeやProviderのReceiptへ結合しない。歴史的Receiptと、現在それを読む権限・業務再実行を許可する権限も区別する。

### 8.3 必要条件を候補側の自己申告に依存させない

前節の`Required Guards`には**独立の基準**が必要。Action候補/AI生成Script/実行計画が自分で`required=[]`と宣言すれば全検査完了となる、という循環した合格判定を禁止する。[Draft #2783 §9/10](https://github.com/hat47x/tei-platform/pull/2783)の移植受入と、[#1180](https://github.com/hat47x/tei-platform/issues/1180)のDriver Scenario/独立Expectedを再利用する。

| 必要条件の起点 | 何を独立に特定するか | 未確認時の判定 |
| --- | --- | --- |
| Accepted Function/Rule、Owner/Scenario | 意味ID、業務入力/対象、必要Data Relation、適用Rule/順序/期間/必要Effect | `required=[]`や未読込Ruleを成功にしない |
| Source manifestと実Loader/Builder | required Source closure、実選択・解析・消費したbyte、Native SQL/Operator/Pluginの版 | 「validate ok」や選択Sourceのhashだけでは欠損なしと主張しない |
| Host/Provider Contract | 現在Principal/Grant/Subject、Resource/Version、tx/cas/idempotency、Receipt保管・照会の限界 | Grant未渡し／実Provider非対応ならblocked/unsupported |
| 独立Oracle/Consumer | 正常・拒否・Unknown・partialのExpected、観測が必要な対象集合/Snapshot・開示範囲 | 0件/Reader failure/非公開を業務上の不存在にしない |

各Stepで`Independent Required → Discovered/Supplied → Actually Evaluated → Enforced at Effect → Observed`を辿り、`Undischarged`を残す。必要なSourceやGuard自体を列挙できないときは**unknown**とし、件数0を検査不要の根拠にしない。純粋計算に不要なDB Commit前提を強制しないが、実Providerを呼ぶ場合はその能力/効果分類を確認する。

### 8.4 次の実験はread-only一件から始め、Writeを同じ資料へ接続する

新しい汎用Fixture/Scenarioの正本・承認機構・Script Runtimeを作らない。既存[移植計画 #2783](https://github.com/hat47x/tei-platform/pull/2783)の`RMT-RD1/RD2/WR1/WR2/WR3`、#1130のACU/RPL、#1180のScenarioを使い、**一つのBusiness Functionに対してClaimとAttemptの列を埋める**。

| 順序 | 一回の最小実証 | 合格できる上限／未達を残す欄 |
| --- | --- | --- |
| V0：Source/Meaning | 一つの実Workspace→実Parser/Role/Rule/Function Data Context、Source必須集合と実Parsed集合を独立照合 | Read-onlyの解析・選択資格。未読込Rule/未受理Meaningはunqualified |
| V1：実Read/Output | 上記Functionの一つのQueryを選択Providerへ実Readし、対象/Scope/Grain/NULL/完全性・実結果Bodyを独立Expectedと比較 | Read Resultの範囲だけ。UI/DB更新・新しいReceiptは不要 |
| V2：実Write拒否 | 同じOwner Ruleの保存前拒否を、実Providerへの送出前／実Commit前の正しい境界で検証 | 拒否の段階とEffect=0の保証範囲。実Rollback/外部Effectを推定しない |
| V3：実Write/Recovery | 受理済み対象・Grant・同じOperation ID/Request hashを既存PG CAS/Receipt経路へ接続し、ACK-loss後の同Key Lookup、別Key ABA負例と独立Readを比較 | 当該実PG/同一Attemptの効果資格。実Process Kill/二Worker、本番Retentionは未達 |
| V4：外部Entry/Consumer | その後、実CLI/Web/desktop/Jobで本当に呼んだArtifactとFunction、Current Disclosure、Output Bodyと業務完了述語を追跡 | 入口別・実Consumer別のE2E資格。EKI/多BuildはP0必須でない |

**Evidence票の行粒度は`Case × Claim × Attempt × Environment × Source/Artifact revision`**とする。実旧Driver、実TEI、EKI Worker、別Consumerは同じScenario/Independent Expectedを比較しても**別Attempt**。各行に`observed/not-run/unsupported/unknown`、実行したCIのHEAD・Run・独立Expected、Owner Acceptance、現在Disclosureの可否を別記する。既存G1/G2の[統合責務 #1163](https://github.com/hat47x/tei-platform/issues/1163)/[#1119](https://github.com/hat47x/tei-platform/issues/1119)へ渡し、**局所Go 3/3・実PG限定PASS・暫定Web E2Eを合算した全体PASS**は作らない。

**次の実装者向けの最短作業**：既存#2882 Workspace fixtureを基線とし、(a) Source/Rule/Period/Grantが独立Requiredに揃っているかの読取票、(b) #2889 OC-RP-01〜03のexacting characterizationと現在未成立のGuard enforced/Effect witnessの差、(c) #2870/2854の実PG Restart/ABAの保証範囲、を**同一ScenarioのClaim別列**に配置する。read-only実Provider/Consumerが接続できない段階で次の書込E2Eを「Script完成」と呼ばない。

**受入の停止線**：この節は既存証拠・設計の照合のみであり、新しいScript本体E2E／実Provider Read→Rule→Write一試行／OS kill／二Worker／DB+RunPlan同一Attempt／Owner正式受理は**NOT RUN**。#2870で再起動を模した新インスタンスが`MayReplay=true`となる性質は**未解決の弱点**として記録し、同Key PG replay成功を一般的な復旧済みと扱わない。#2889は**Draft**、専用Go合格とRepository Health失敗を独立のEvidenceとする。

## 9. .tei.md移行・Bootstrap資格・Completion判定の接合（2026-11-06）

### 9.1 方針の更新と実装状況：拡張子の変更は実読込の保証ではない

**本チャットが受け取ったOwnerの最新ファイル方針**：新規のTEI対象Sourceは原則`*.tei.md`へ集約し、`*.tei`は段階的に廃止する。この判断を本書の**移行方針**として扱う。一方、[Issue #2621](https://github.com/hat47x/tei-platform/issues/2621)には「`.md`+OKFタグをAuthoring、`.tei`を厳密定義Source、`.tei.md`は標準化しない」という**従前の比較案**が残っている。Issueの旧案・PRのDraft・Ownerの新方針を決定済みとして混在させない。形式規則・具体的Grammar・互換移行の正本更新は**#2621（Authoring）／#988（Language Plugin）／#2823（Loader）**へ引き渡す。本PRはScript起動への影響だけを所有し、他担当Issueを上書きしない。

**2026-11-06 main実コードの現状**：`internal/workspace/loader.go`には`semantic` Sourceの`strings.HasSuffix(name, ".tei")`、`plugin.tei`／`bundle.tei`／`application.tei`等の固定名、各種`*.tei` globが残っている。`*.tei.md`を既存Workspace Loaderが同等に読むと主張できない。既存の`.tei` Fixtureと正式な実Parser対応が移るまで、**無条件の一括改名・二重読み・黙った無視を禁止する**。旧`.tei`の互換Readerを恒久的に採用すると決めたものでもない。

新しいSourceの機械処理は、`file extension / OKF-or-document metadata / selected TEI region / Language Plugin / parsed structural candidate / accepted Meaning / executable Build entry`を別の境界とする。`*.tei.md`という名前やFrontmatter上の`tags: [tei]`は「TEIを意図した文書」のシグナルになり得るが、**文書全文の実行許可・全表の業務規則化・Owner Meaning Acceptance・Host Grant**ではない。Markdown中のどの範囲を実装可能な構造として読ませるか、Code fenceと表の評価対象／保存Source positionをどう保つかは#988/#2621の正式Grammar・Source mappingを待つ。本文の無関係な文章や引用を黙ってDSL実行へ昇格させない。

| 移行対象 | 必要な段階・保存する証拠 | 起動時の停止条件・禁止推論 |
| --- | --- | --- |
| 文書`order.tei.md`新規作成 | Source path/raw bytes、Intent/profile、選択Source span、Language Plugin/Parser版、読込対象の明示 | 拡張子やタグだけでは`Parse/Meaning/Execute qualified`にならない |
| 既存`order.tei`を新名へ移行 | before/after raw bytes、構造候補・Meaningの対応、独立Expected、生成/実Loader/Consumerの実Byte | 単なるrenameは実行資格の移植でない。移行前後の両方を二重Effectとして走らせない |
| 同名構造が両拡張子にある | Authoritative Source ID・具体的採用範囲・差分・Ownerの選択 | **last-wins**や暗黙の二重定義を認めずambiguousで停止 |
| `workspace-files`で列挙されるが実Readerが読まない | managed inventory／`readByLoader`候補／実Parser成功／実Build入力／必須Source閉包を別列にする | `validate ok`／`readByLoader=true`はParser・Meaning・全閉包資格でない |
| 本文の一部に受理済み意味がある | 原文の保存、use-site/位置/Source revision、受理済みMeaningの参照、未解釈部分 | Draft Saveや部分Parse成功で文書全体を実行可能としない |

[Draft #2935](https://github.com/hat47x/tei-platform/pull/2935)は`workspace-files`に`readByLoader`を追加する限定提案・試験であり、**mainへ未統合**。Loader read事実はParser成功、Role/Meaning受理、Required source closure、実Build consumerを証明しない。[Issue #2823](https://github.com/hat47x/tei-platform/issues/2823)の未知Root／解析対象外Source診断も引き続き未決。

### 9.2 DSL BootstrapとScript Run開始を独立した契約にする

[Draft #3028](https://github.com/hat47x/tei-platform/pull/3028)が提案するK0信頼Seed→K1 Source/Manifest→K2 exact Language選択→K3 Parse/Candidate→K4 Operator staged binding→K5 Required/Owner受理→K6 Build/Consumerの段階を、既存S1〜S5の**成立要件として参照**する（本書でBootstrap実装・第二Registryを作らない）。Candidateが一部登録されたこと、Parserが将来の構文を解釈できたこと、Operatorが見つかったことのどれもActive資格にはしない。初回Seed不整合／版不一致／一部登録・障害後の復帰では、**確実に資格が確認された既存Activeだけを使うか、できなければ拒否**する。

- 同じ`*.tei.md`本文と同じ見かけのRoleでも、Parser/Operatorの実版・Source位置・必要Relation/NULL/不在・選択Targetが変われば、Source Candidate、Accepted Meaning、Build、Action Offerの再資格化を別判断する。
- ScriptのExecutable Definitionが`Source revision`だけ一致していても、実ロードされたPlugin／Operator／Artifact byteが異なるならS4/A0の資格を据え置かない。特に未検証のSource Region・否定依存・選択Dialectがあれば`unknown`で止める。
- Plugin Manifestが部分登録で復帰不能な場合にも、実Providerへの暗黙fallbackや旧Grantを許さない。A0開始前に止めるべき問題と、A0後のCommit unknownの回復手順を混同しない。
- [Draft #2898](https://github.com/hat47x/tei-platform/pull/2898)のRole重複Syntax診断・read-only位置アンカーは局所的なParser改善候補。Role Writer・Owner意味受理・変更後実Consumeの保証は別。

### 9.3 意味受理、実行認可、Effect確定、業務完了は別の主張

[main統合 #2926](https://github.com/hat47x/tei-platform/pull/2926)には`internal/semantic/completion.go`の`EvaluateCompletion`があり、`TargetSet L × EmptyPolicy × Predicate P`から`holds / fails / unverified / contradiction`を判定する**局所的な実装・Go試験**が成立している。Runtime/Provider/Consumerが正しい対象集合・Coverage・空集合の独立証拠・手作業引継ぎを実入力として供給できたことは未実証。新しいScript Completion Evaluatorや独自Completion Statusを作らず、その入力条件へ接続する。

| Owner受理済Meaning M | 現在Host Grant G | 解釈・起動への影響 |
| --- | --- | --- |
| あり | あり | その両者だけでは不十分。Source/Runtime/Target/GuardとEffect Fenceまで満たす範囲だけDispatch候補 |
| あり | なし・失効 | Scriptの意味は受理済みでも実行拒否。過去Receiptは所定の開示権限内で履歴として保持 |
| なし | あり | 実行権限を有するHostがいても当該業務意味は未受理。生成AIの候補・プレビューを実正式処理に昇格しない |
| なし | なし | 意味のReviewとHost認可は別に停止。片方の修復から他方を推定しない |

| 最終観測の例 | Completion評価へ渡せるか | 主張の上限 |
| --- | --- | --- |
| `Process exit=0`／RunLog Completed／Commit Receiptあり | それだけで独立`TargetSet`/Coverage/Pを組み立てない | Process/当該Effectの限定Status |
| 必須対象集合の一部だけReaderで確認、全件Scopeは不明 | `CoveragePartial/Unestablished`と観測できた対象結果のみ渡せる | `unverified`、または観測済み反例があれば`fails` |
| 完全な空集合が独立証明された | 明示`EmptyPolicy`に従い判定 | Policyによって`holds`／`fails`／`unverified`が異なる |
| 手作業確認が必要なProcedure、引継ぎ実受領は無し | `HandoffConfirmed=false`を保持 | DB更新済みでも業務完了の`holds`を自動主張しない |
| 実ReadとReceiptが食い違う／必要観測が相反 | 一致のために片方を破棄しない | `contradiction`を独立観測として残し、Ownerによる再評価 |
| 完全な同一Scope・対象・時点の観測＋全Predicate成立 | 対象列・完全性・空集合証拠・引継ぎ・意味Ownerを明示して接続 | **定義された完了述語に限って**`holds`候補。現在の公開/操作許可はなお別 |

ここで`EvaluateCompletion`が`holds`を返すこと自体と、**その入力証拠が実Target・実Reader・Current Disclosure Grantに由来した**という資格は別。Owner受理済みのCompletion claimではないものを、評価関数が成功しただけで正式な業務完了にしない。Source/Resultが同じでも歴史的Accepted Meaning、現行Grant、Consumerへの開示を毎回混同しない。

### 9.4 追加の独立反証：既存Scenarioへ接続する差分だけ

| 共有するCase/Source | 新しく固定する対照 | Expected / 責務 |
| --- | --- | --- |
| #2823／SH-01：`order.tei`→`order.tei.md`名変更 | 旧Loaderの`.tei` suffix選択と新形式を同一Workspaceで比較 | 宣言されるが実Parsedが無いものを`valid`としない。#2823/#2621/#988で移行実装・仕様判断 |
| #2621 FM-02/03/04／SH-03：OKFタグ付きMarkdown、選択部分にRule候補 | Raw Draft保存／引用部分と実Parsed Role・Operator候補の分離 | タグ追加のみでは実行不可。選択外/未知属性は失わない。正式Role/Writerは#2620/#988 |
| #3028 BS-05/09/12：Parser/Operator途中登録、再起動 | K2実選択／K4部分登録／現Active/Build資格の別観測 | 不完全な候補をActive/実Providerへ通さない。Runtime実障害注入はNOT RUN |
| #1180/#2514／SH-11：TargetSet partial、0件・未知粒度、矛盾 | 同一独立Expectedと`CoverageComplete`／`CoveragePartial`／`EmptyJustified`の差 | #2926既存Evaluatorを用いた条件付き判定。独立Readerの全対象被覆を自作のFixture判定で代用しない |
| #1130/#1174／SH-07/08：RunPlan Completed・Commit Unknown・手作業未完了 | Run/Effect/Completionの3結果軸を同じOperation IDへ結ぶ | DBやRunの成功を`BusinessVerified`へ昇格しない。実同一Attemptは未実施 |
| #2514 #3017：意図的なEvaluator Mutant | Source/Fixture/Expected不変、誤実装時の直接Red判定 | [#3017](https://github.com/hat47x/tei-platform/pull/3017)の限定Product Evaluator mutant検出は既存Oracleの判別力の一例。Script/Completionの新Mutant検出済みとはしない |

**次の最短実証順序（既存V0〜V4への差分）**：V0の最初に`*.tei.md`と旧`*.tei`の**Reader/Parser実消費比較**を挟む。ただし正式Grammarが未決・未実装ならそこで止め、旧Loaderの限定実績を新形式の成功と呼ばない。V1の実Read→独立Body/TargetSet/Coverageを先に整備し、V2の業務拒否とV3の実PG Effect/Receipt/Lookup、V4のConsumer/Business Completionは段階的に追う。比較では**一つの固定Source/Scenario/独立Expected**を用い、旧形式と新形式・実DriverとTEI・Provider別は異なるAttemptとして記録する。実行候補が増えただけで新しい全体E2Eが成立したとしない。

**既存Ownerとの分担**：`*.tei.md`のファイル/Authoring/配置判断＝#2621、Parser/Language Bootstrap＝#988/#3028、Loader/必要Source＝#2823/#2935、Role/Meaning＝#2620/#2619、独立Oracle＝#2514/#1180、Script起動と権限段階＝#2502/#2499、RunPlan/部分効果＝#1130、Effect/Receipt＝#1174/#2564、Completion述語＝#1125/#2926、統合Evidence＝#1163。新しいSource Registry・Script Host・Runner・Completion Engine・第二Oracleを導入しない。

**現状**：`*.tei.md`移行はOwner方針として採用対象だが、現Loader/全Fixture/Studio/CLIの新形式対応および新PackageのParser/Source位置保持はこの追補では**NOT RUN／NOT PROVEN**。#2935/#3028/#2898はDraft、#2926はmainへ統合済みの限定Evaluator。実TargetSet/Coverageの独立Provider証拠、同一Function/実PG/実Consumerの一改訂Product E2E、Ownerの新規意味受理、Windows実行も**NOT RUN**。
