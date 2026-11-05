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
