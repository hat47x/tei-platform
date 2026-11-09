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
| P0 読取り専用の証拠結合 | mainのScript/Function Source、実Go CLI/RunPlan、Effect Commit、受理済みMeaningの同一Revisionを照合。SHを既存Scenario/検証基準へ参照付け | 元Source／実Script入口／独立Expectedのいずれか不在 |
| P1 Headless合成 | Source/Meaning拒否、必須チェック欠落、先行Effect済みblock、Observer有無、ACK-lossの判別を既存Fixtureで実施 | Provider呼出0／Unknown／Partialが期待通り分かれない |
| P2 既存PGの実縦断 | 既存Command Invokerと実PG、Receipt照会と**別接続**独立Readを、同じOperation/Artifact/Rule版へ束縛。Restart/Outboxを後続 | 同一Tx・Owner Scope・現在Grant・実Load版が独立に確認できない |
| P3 UI/外部ジョブ | 同一FunctionをCLI/Web/cron相当・選択Hostへ公開し、必要に応じて1→2 Build比較、EKIはwire+実Worker適合を別評価 | 入口の到達だけでEffect保証、旧Jobの無条件乗換、対話Scriptの無人実行 |

**担当**：起動体験/Preview＝#2502/#2507、Source/Meaning＝#2619/#988/#2595、Mapping/Provider能力＝#2622/#951、Data Rule phase＝#1129、実Effect/Receipt/Current Grant＝#1174/#2499、Run/Step・再起動＝#1130/#2146、Build選択＝#1130/#943/#2788、独立Expected＝#1180/#2514。ホスト認証は[ADR-0044](../adr/0044-authentication-not-a-core-concern.md)に従う。

**完了主張の最低条件**：少なくとも同一Operation/Source/Artifact/Host/Providerが識別された一試行について、(1)実送出前停止を確認、(2)実Commitの確定・Rollback・Unknownの区別、(3)Partialを保持、(4)同IDの安全なLookup/再開と新ID誤再実行の反例、(5)実独立Readによる必要対象の完了判定が存在すること。Synthetic／実PG／実Web／実OS／実EKI／Windows GUI／Owner意味受理は別列のEvidenceであり相互流用しない。

**未決**：実Context/Referenceの鮮度期限、A0で何をどのHostが永続化できるか、Script Code/実ロードbyteの原子的Snapshot、補償のOwner判断、未検証の動的依存、Receipt保持期限と障害後Lookup資格、公開Scriptが対象のサービス認可モデル。欠けた保証を新しい中央Registry/Job Engineで隠さず、既存契約で満たせるか先に反証する。本書の新しいSH試験、製品Script全工程、実再起動/外部Effect、TEI↔EKI実Worker、Owner正式採用は**NOT RUN／未採用**。

## 7. 検査被覆・非同期応答・回復資格の接合（2026-11-06、追補）

**前節との差分**：以前のS0〜S8／A0とF01〜14は、故障が起きたら何を保持するかを定めた。本節は、**停止を解除してよいと誰が何で判断するか**に限定する。新しいEngine/State enum、中央Audit Store、第二の検証基準エンジンを導入しない。現mainへの到達点は[merged #2882](https://github.com/hat47x/tei-platform/pull/2882)のWorkspace→`DeriveOperationSurface`→`PeriodPreconditions`／`EffectPreconditions`→`CombinePreconditions`→`RunPlan`という**合成Stepの実Goテスト**に更新する。Period closed/open、after-commit Observerなしでblockedを局所確認済み。一方、Guardの網羅性／実ProviderによるEffect強制／Observerが当該Effectを実観測したこと／実永続Run所有は未証明。既存[#1130 RPL-01〜05](https://github.com/hat47x/tei-platform/issues/1130)の検査正本を再利用する。

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

**更新されたUI実証範囲**：旧[Draft #3036](https://github.com/hat47x/tei-platform/pull/3036)はmainとの乖離・競合のため**未統合のままClose**され、現mainから再作成した[後継Draft #3117](https://github.com/hat47x/tei-platform/pull/3117)のみがD0 Bootstrapの進行候補。新HEAD `b75df7a54a27a540e51c57778fd173213752491f`では[専用Node #42309517633](https://github.com/hat47x/tei-platform/actions/runs/42309517633)と[実Wails Server＋Chromium #42309517587](https://github.com/hat47x/tei-platform/actions/runs/42309517587)が局所成功（旧RPC応答・旧`finally`抑止）。Repo Healthの完了結果は本照合では未確認。**既定Windows Desktopは`/workbench/`という別Consumer**であり、Server Bootstrap `/` のPASSをDesktop Command/View・Host Grant・Provider Effect/Receiptへ転用しない。旧版の成功ログを新HEADの証拠に使わない。

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

**このScript設計の同一Revision・同一AttemptでのSH系一気通貫試験／実UI Epoch／実Service Grant+Period+DB同一Commit／再起動後の実Provider Receipt接合／二Provider／Owner意味受理はNOT RUN。** main #2942/#3002にはCommand Stage単体と製品CLIの二Process再起動試験が既にある。これを本Script/実PG統合成功へ昇格しない。既存#2882局所Go、#2853実PG Commit Unknownも、それぞれのSource/HEADで成立した限定Evidenceとして再利用する。

## 8. 永続回復の反例と一改訂受入の最短経路（2026-11-06、再照合）

### 8.1 既存の実証を追加し、過大な「回復できる」を禁止する

**2026-11-06再照合の新しい事実**：
- [main統合 #2870](https://github.com/hat47x/tei-platform/pull/2870)のRCAS-02は、旧`Lifecycle`に`unknown`があっても、新しい`Lifecycle`（再起動相当）では記録が無く`MayReplay=true`になることをGoテストで固定した。同じIdempotency Keyの実PG再送が`IDEMPOTENT_REPLAY`（適用0・Receipt 1件）となるのは**Provider側の永続記録**による限定的な保護。実OSプロセスの死・実ネットワーク遮断・二Worker/Hostの所有保証をテストしたわけではない。
- [main統合 #2942](https://github.com/hat47x/tei-platform/pull/2942)は`internal/command.DurableLifecycle`（単一ホストのファイル永続Stage）と**実サブプロセス再起動テスト**を追加。未知/観測済みStageを跨いで保持し、破損Stateを空として扱わず拒否する。一方、同時Process間競合・分散Fence・実Provider Receiptは未検証。従前#2870の「再起動でStageを失う」は**in-memory Lifecycle限定の反例**へ再分類する。
- [main統合 #3002](https://github.com/hat47x/tei-platform/pull/3002)では製品CLIの`view -execute`に**opt-in SQLite Lifecycle Store**（`-lifecycle-state`／安定`-lifecycle-command`）を結線。実CLIバイナリを2プロセスで起動し、同じCommandの観測済み再送を`COMMAND_LIFECYCLE_REFUSED`で拒否する局所試験が成立。ただしExecutorはReferenceで実Provider効果なし、Store未指定時は従来in-memory。
- [main統合 #3008](https://github.com/hat47x/tei-platform/pull/3008)では`studiohost.ExecuteCommandWithLifecycle`を追加し、合成Store利用時に**ホスト再生成相当**の観測済みCommand再送を拒否。既存`ExecuteCommand/WithPolicy`への強制はなく、Wails製品HostのStore合成・実WebView/Provider/複数Hostは未実証。
- [main統合 #2854](https://github.com/hat47x/tei-platform/pull/2854)は、第三者により値が元へ戻ったABA後に**別KeyでAPPLIED・Receipt 2件**となる反例を実PGで固定。同じ値になったこと／新しいAttempt ID／同じ業務目的を自称することは、再実行安全性の証拠にならない。
- [Draft #2889](https://github.com/hat47x/tei-platform/pull/2889)では実Go `internal/execution.RunPlan` にToy Step/Journalを接続し、**Host Grant Guardを渡さなくてもcompleted、Observer存在だけで未観測、途中block前のStepが残る**というOC-RP-01〜03を固定。専用exact-HEAD [Actions #42100472921](https://github.com/hat47x/tei-platform/actions/runs/42100472921)はGo 3/3 PASS。一方、同headのRepository Healthはgofmt driftでFAILと[Draft #2783の再照合](https://github.com/hat47x/tei-platform/pull/2783)に記録されている。**専用試験PASS ≠ repository全体PASS ≠ Host Grant強制済み**。
- [Issue #2823](https://github.com/hat47x/tei-platform/issues/2823)では、宣言Source rootが存在せず／想定ファイルがローダの対象外でも、限定条件で`validate ok`になり得る実CLI反例が記録されている。`ok`だけで「必要なScript/Ruleをすべて解析した」と判定せず、`required / declared / discovered / loaded / parsed / actually consumed`を別確認する。
- [#2845](https://github.com/hat47x/tei-platform/issues/2845)は実運用のReceipt/観測記録の保存先・保持期間・検証・閲覧資格を未決のまま所有する。**digest-only記録は入力の復元ではなく、保存されていない記録の不在は未実行の証拠でない**。

**現在のStage経路の実装差**：

| 入口／Stage実装 | 再起動後の拒否が局所実証された範囲 | 残る失効・効果の証拠 |
| --- | --- | --- |
| in-memory `Lifecycle` | #2870では再生成時にunknown消失（負例） | 同Key PG Receiptが別に二重適用を防ぐことは一般保証でない |
| file-backed `DurableLifecycle` | #2942のサブプロセス再起動、unknown/observedと破損拒否 | 同時Writer、Process間Fencing、Provider実Effectとの原子的接合 |
| CLI opt-in SQLite Stage | #3002の別CLIプロセス再起動で観測済み再送拒否 | 未設定CLI・実DB Write・Provider Receipt／対象Grant |
| StudioHost composed Stage | #3008のHost再生成時の再送拒否 | Wails実起動でのStore合成、実Provider、他Hostとの所有 |

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
| 独立検証基準／Consumer | 正常・拒否・Unknown・partialのExpected、観測が必要な対象集合/Snapshot・開示範囲 | 0件/Reader failure/非公開を業務上の不存在にしない |

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

**受入の停止線**：この節は既存証拠・設計の照合のみであり、新しいScript本体E2E／実Provider Read→Rule→Write一試行／OS kill／二Worker／DB+RunPlan同一Attempt／Owner正式受理は**NOT RUN**。#2870で新インスタンスが`MayReplay=true`となる性質は**in-memory方式に限定された弱点**として記録する。#2942のDurableLifecycleと#3002のCLI接続でStageの一部対策は実装・局所実証済み。ただし実Provider Receiptと永続Stageの同一Attempt接合、複数Writer/Fence、本番Host認可は依然未解決であり、同Key PG replay成功を一般的な復旧済みと扱わない。#2889は**Draft**、専用Go合格とRepository Health失敗を独立のEvidenceとする。

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
| #2514 #3017：意図的なEvaluator Mutant | Source/Fixture/Expected不変、誤実装時の直接Red判定 | [#3017](https://github.com/hat47x/tei-platform/pull/3017)の限定Product Evaluator mutant検出は既存検証基準の判別力の一例。Script/Completionの新Mutant検出済みとはしない |

**次の最短実証順序（既存V0〜V4への差分）**：V0の最初に`*.tei.md`と旧`*.tei`の**Reader/Parser実消費比較**を挟む。ただし正式Grammarが未決・未実装ならそこで止め、旧Loaderの限定実績を新形式の成功と呼ばない。V1の実Read→独立Body/TargetSet/Coverageを先に整備し、V2の業務拒否とV3の実PG Effect/Receipt/Lookup、V4のConsumer/Business Completionは段階的に追う。比較では**一つの固定Source/Scenario/独立Expected**を用い、旧形式と新形式・実DriverとTEI・Provider別は異なるAttemptとして記録する。実行候補が増えただけで新しい全体E2Eが成立したとしない。

**既存Ownerとの分担**：`*.tei.md`のファイル/Authoring/配置判断＝#2621、Parser/Language Bootstrap＝#988/#3028、Loader/必要Source＝#2823/#2935、Role/Meaning＝#2620/#2619、独立検証基準＝#2514/#1180、Script起動と権限段階＝#2502/#2499、RunPlan/部分効果＝#1130、Effect/Receipt＝#1174/#2564、Completion述語＝#1125/#2926、統合Evidence＝#1163。新しいSource Registry・Script Host・Runner・Completion Engine・第二検証基準を導入しない。

**現状**：`*.tei.md`移行はOwner方針として採用対象だが、現Loader/全Fixture/Studio/CLIの新形式対応および新PackageのParser/Source位置保持はこの追補では**NOT RUN／NOT PROVEN**。#2935/#3028/#2898はDraft、#2926はmainへ統合済みの限定Evaluator。実TargetSet/Coverageの独立Provider証拠、同一Function/実PG/実Consumerの一改訂Product E2E、Ownerの新規意味受理、Windows実行も**NOT RUN**。

## 10. 二経路移行・Script公開/Queue・実行権限の再資格化（2026-11-07）

### 10.1 形式移行を既存実行E2EのHard Dependencyにしない

最新のAuthoring Owner方針は[Issue #2621](https://github.com/hat47x/tei-platform/issues/2621)の2026-11-06追補、および[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)の**M0〜M6**を参照する。新規TEI Authoring Sourceは原則`*.tei.md`、旧`*.tei`は段階廃止。`plan/design/TEI_MD_UNIFIED_AUTHORING_MIGRATION_2026-11.md`が移行の責務を所有する。前節§9「V0でまず`*.tei.md`と旧`*.tei`を比較」は、**新形式Parser未成立なら既存legacy実行経路全体を止める**という意味ではない。本節の二経路で補正する。

| 並行Lane | 実際のInput・実行先 | 正当に主張できる範囲 | 横断してはいけない結論 |
| --- | --- | --- | --- |
| L：現行legacy機構の縦切り | **実際に現行Loaderが読み、Parse/Linkした`*.tei`**を固定。既存RunPlan/Function→実Provider Read/Expected、後段Command/PG/Receipt | そのSource/Parser/実Artifact/実Providerに限る局所実行・障害回復の資格 | 旧形式のPASSは新`*.tei.md`のParser・Source位置・意味受理を保証しない |
| M：Authoring新形式の段階移行 | 新`*.tei.md`、Markdown container、Language Plugin、選択した構造span、実Loader/Build Input。#3030 M0〜M6の段階成果を再使用 | 新形式の発見／解析／意味候補／旧定義との比較など**成立した段階だけ** | Read/ParseのPASSはRuntime Effect、Script公開、現在Grantを保証しない |
| J：将来の結合受入 | LとMの**同じ業務Scenario／独立Expected**、別Source Revision/Attemptを突合して移行後の実Targetを検証 | M側が実Parser/Link/Build/Providerを消費した時点で初めて新形式End-to-EndのClaimを検証可能 | 値だけ一致・Rename・旧CIを新規受理に転記しない |

同名の旧`*.tei`と新`*.tei.md`が混在する場合、候補を暗黙にどちらか選択して読まない／last-winsしない。暫定[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)は管理Rootに新形式が一件でもあるとWorkspace Load全体を拒否し得るため、SHOの未完成`*.tei.md`草稿を**管理外Draft領域に保持すること**と、管理対象の必須Sourceとして選択・解析することを分離する。暗黙に新形式を無視して`validate ok`へ戻すのも禁止する。明示的なAuthoritative Source・Version・移行時点・対象範囲と、**実Reader consumed bytes**を束縛する。旧形式のread-only/Import経路をいつ削除するかは#3030のM6とOwner判断による。`.tei/decisions/`はSource拡張子ではなく固定履歴ディレクトリであり変更しない。新形式Markdown中の通常文章・引用・表が無条件にExecutable Ruleになるわけではない。

[Draft #2935](https://github.com/hat47x/tei-platform/pull/2935)の`readByLoader`（Load snapshot）と[Draft #3028](https://github.com/hat47x/tei-platform/pull/3028)のBootstrap K0〜K6は、それぞれ局所的な**読込証拠**、**Language/Operator資格の先行案**として参照するだけ。両PRはmain未統合。`readByLoader=true`はSource解析・必要閉包の完全性を示さず、K2/K4のCandidateはOwner受理済みMeaningでも現在の実行資格でもない。

### 10.2 Scriptの作成・公開・予約・起動を同一承認にしない

[#2502](https://github.com/hat47x/tei-platform/issues/2502)が所有するTrusted Developer Script／Delegated User Macro／組織公開のProduct Contractを、S0〜S8の**入口境界**へ接続する。高権限Scriptを一律に禁止しない。一方、Macroへ公開された能力を越えるraw SQL／管理操作を暗黙に与えず、PublisherがScriptを公開した事実をデータ変更のGrantと混同しない。新Macro EngineやTEI独自Hostを設けず、既存Host/Provider/Pluginの実契約で判定する。

| 段階 | 確認する不変条件 | その段階だけでは与えられない資格 |
| --- | --- | --- |
| Author/Save | Source raw bytes、作者・候補Meaning/Function、Draft/Attachmentの実保存版。未受理・Parse失敗でもRaw Draft保持は可能 | 公開・起動・書込みGrant |
| Review/Publish | 公開対象のScript Artifact/Contract、必要Capability/Effectと受理範囲、Publisher/対象Organization／Tenant・Audience、依存版 | その時点のInvoker/Runtime Principalの実Grant、他Tenant利用資格 |
| Bind/Queue | 起動契機、実Invoker、対象・Context、Script公開版、Application/Target、**業務Operation Identity/Request Hashの予約時の記録** | Queue時のGrantだけで将来の実DispatchやCommitを許可しない |
| Start/Admission | **実ロードScript/Artifact**と公開対象版・必要能力の照合、現在のHost Principal/Grant/Target/Context、Required Guardの被覆、Effect計画、Attempt/Fence | プレビュー承認・Queue受付・Task起動成功からDB Commitを推定しない |
| Provider Effect/Recover | Provider固有のCurrent Grant/CAS/Transaction/Idempotency/Receipt、部分効果・unknown・Lookup権限 | 実CommitとAfter-commit配送、業務完了・外部開示の同一視 |
| Observe/Present | 独立ReadのTargetSet/Coverage/Expected、Completion判定、**現在のDisclosure Grantと用途** | 過去の公開/保存/Runの成功が現在の開示権限を発行することはない |

**Identityの最小参照集合**（新公開封筒Schemaではない）：`Script Definition/Source revision`、`Published Artifact/Contract revision`、`Invocation binding/Target`、`Business Operation ID`、`Request digest`、`Run/Step/Attempt/Fence`、`runtime Principal/Grant判定の非機密参照`、`Provider Effect/Receipt`。これらは一つに潰さず、実際の既存Contract/Hostの所有者へ参照として渡す。Raw token/DSN、秘密値/非公開Row一覧を結果や監査・AI表示へ複写しない。Invocation Contextは#1118、公開/委任のGrantは#2499、実Commit Fenceは#1174、組織/Tenantの時間意味は#2386、Receipt保存は#2845の責務。

**起動取り消し/タイムアウト**：予約を取り消しても既にDispatchされたEffectが巻き戻るとは限らない。`cancelled`はQueue/Processの実行予定取消の証拠として扱い、Providerへ到達した可能性がある場合は同じOperation IDのLookup・部分効果保全を先行する。二法人兼務の同一人物、同Script別Tenant、公開後のGrant撤回を、本人ID一致や同Artifact bytesを理由に権限流用しない。対象/Operationを跨いで同じIdempotency Keyを無条件に共有しない。

### 10.3 既存SM/SH受入へ足すべき差分と、着手順

[#2502 SM-1〜6](https://github.com/hat47x/tei-platform/issues/2502)および本書SH-04/07〜12、[#1180](https://github.com/hat47x/tei-platform/issues/1180) Driver Scenario/独立Expectedを再利用し、Case ID・第二Fixture正本を作らない。旧Driver/TEI／Targetごとに別Attemptを記録する。

| Fixture接続 | 独立に比較する二状態 | 期待結果・Claim ceiling |
| --- | --- | --- |
| SM-1 vs SM-2 | 同一Script/型付きContract、Trusted DeveloperとDelegated Macro、raw SQL可否・対象Order A/B | Adminに許されたraw pathは標準Data Rule/Receipt保証を**自動主張しない**。Macroの非許可能力はHostで拒否し、当該対象Effect=0を実証するまでは未資格 |
| SM-3：二法人／同Plugin別Tenant | Script publish済み、AだけにGrantあり。Bで起動／他Tenantの歴史を照会 | Publish≠Grant、同Artifact≠共有Authority。拒否時は非公開Source/対象の存在を出さない |
| SM-4：改版済みの同名Macro | `M@digest1`で公開後、`M@digest2`が新Capabilityを要求／旧予約が残る | 旧承認を新版へ移さず、旧予約も現実のArtifact/Contract/Grant確認なしに新Versionへ付替えない |
| SM-5＋SH-04：Queue時許可・開始直前撤回 | Queue登録と起動、及びCommit前にGrant/Target/Period変化 | AdmissionとEffect前強制を別確認。拒否時はProviderで到達/Effect=0の証拠を取る |
| SM-5＋SH-08/09：Run cancel/Restart/ACK-loss | 旧Operationはunknown、Processが新Attemptを作る。同Key/新KeyとABAを比較 | `Process cancelled/exit`からEffect=0を推論しない。同Operation Receipt照会優先、新Key盲再送禁止 |
| SM-6＋#3030 M0〜M6 | Lの旧`.tei`、Mの新`.tei.md`、CLI→Web/Studioの実Consumer | 新形式の実Loader/Parser/Artifact/Body受領と別Attemptの独立Expectedを確認するまで新形式E2Eはnot-run |

**段階計画への更新**：
- **P0-A（既存機構を止めずに試験）**：現行`*.tei` Workspaceで**一つのread-only Function/Query→実Provider Read→独立Expected/Body**を固定し、現行Host/ProviderとRuntimeが実際に消費したInputを記録。#1163/#2586の既存統合票を利用。
- **P0-B（移行専門レーン）**：#3030 M0→M2で`*.tei.md` Source列挙、Markdown→実TEI span、Parser選択、Raw Draft保存を別途実証。M2が未達なら新形式のScript起動を`unsupported/unknown`とし、P0-Aのlegacy局所成果は保全。
- **P1（Script Surface）**：#2502 SM-1/SM-2から同一Function/契約を異なるPrincipal/Capabilityで呼ぶread-only/拒否試験を優先。SM-3/4の公開・改版・Queueを段階的に追加。
- **P2（実Commit/回復）**：選択HostのCurrent Grant/Target/Periodを、既存#1174/#2564の実PG CAS/Unknown/Receipt・RunPlanへ同一Attemptで束縛。先行Stepの効果/After-commit通知、Process restartを分離。
- **P3（Target別Consumers）**：CLI/Web/Studio/Jobの各実Entry・配備Artifact/権限/独立Bodyを照合し、新形式移行と公開Macroの消費者互換を検証。EKI/複数Build/Marketplaceは最初のread-only E2Eの必須条件にしない。

**現在の証拠**：#3030は**Draft docs-only**、#2935は**Draft限定Loader read観測**、#3028は**Draft部分Go bridge**、#2926のCompletion Evaluatorは**main統合済み・Runtime未接続**。本書の二経路L/Mの実比較、#2502 SM-1〜6の実Product/Host/Provider試験、実OS/Network ACK loss、全Script形式移行、Ownerの業務意味受理は**NOT RUN／NOT QUALIFIED**。#1119 G1/G2やOwner承認をこの計画から自動成立させない。

## 11. Script Admissionの実装接続点と一改訂検証の停止線（2026-11-07）

### 11.1 既存コードで成立した局所防御を、新しいScript E2E合格と取り違えない

- **TEI Markdown未接続**：[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031) HEAD `63953ccb5c12ead26ec1268e7c0afdba4eff66ae`は、Domain Loaderの前でManaged File中の`*.tei.md`を`TEI_MD_PARSER_UNBOUND`＋Source pathにより拒否する**暫定fail-closed**を実Go試験で確認。専用[CI #42217379093](https://github.com/hat47x/tei-platform/actions/runs/42217379093) success、4 focused testとWorkspace package test/vetの限定証拠。**Draft/main未統合**、Markdown AST/Source span/TEI Grammar/実Parseは未実装。旧`*.tei`と通常`.md`の処理は維持するが、従来Loaderの`validate ok`を新Sourceの成功に転用しない。
- **Command認可の入口**：[main統合 #3034](https://github.com/hat47x/tei-platform/pull/3034)は`internal/security/legacyauthorization/command_authorizer.go`の`CommandAuthorizer`を`command.Invoker.Authorizer`へ接続。既存`security.authorize`ルールがallowならExecutor到達、**規則上denyなら`ErrNoGrant`＋`COMMAND_UNAUTHORIZED`かつExecutor非呼出**、規則評価不能ならdenyと混同せずerror、と実Goテストが区別。**権限付与SSOT/Host Credential/current Grantの供給とCommit直前強制は未接続**。
- **宣言効果・報告・実適用の観測を分離**：[main統合 #3068](https://github.com/hat47x/tei-platform/pull/3068)は`AppliedUnobserved`、[main #3104](https://github.com/hat47x/tei-platform/pull/3104)は効果非宣言時の「比較対象なし」を未観測と誤表示しない補正、[main #3112](https://github.com/hat47x/tei-platform/pull/3112)は`ReportedEffectMissing`を追加。現`internal/command/invoke.go`で`ReportedEffectMissing = (plan.Effect != nil && result.Effect == nil)`、`AppliedUnobserved = (plan.Effect != nil && len(result.Applied) == 0)`を確認。後者のfalseは**比較対象なし**と**適用を観測した**の双方を含み得る。どちらもProviderのCommit/Receipt・業務意味の確認ではない。
- **意図的な非拒否**：#3112は`effectcommit` ExecutorがCommit Receiptを返しても`result.Effect`を返さない局面を考慮し、報告欠落だけを`MISSING_COMMAND_EFFECT`や`EffectUnknown`へ一律変換しない。この設計を「報告欠落でもEffect成立」と読み替えない。Provider固有のReceipt/実DB独立Read/Expectedとの照合、未報告を拒否すべきExecutor契約の採用は#2996/#2998/#1174の残件。
- **実DBチェーンのEvidence訂正**：[main統合 #3025](https://github.com/hat47x/tei-platform/pull/3025)の固定HEAD `f887ab979b104b2593561bde3f9e11e8fc7083c7`／[Health #42195019838](https://github.com/hat47x/tei-platform/actions/runs/42195019838)で暫定PG一改訂Packetの`TRUSTED_FULL_HEALTH_NOT_RECORDED`は解消。しかし監査Resultは**`INCOMPLETE / productGate=UNQUALIFIED`**。残る`PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE`、`PROVISIONAL_GRANT_NOT_G2_EVIDENCE`、`PROVISIONAL_GUARANTEE_NOT_G2_EVIDENCE`の3診断は正式受理・現在認可・外部Provider保証の不足を示す。局所Go`#3034`認可成功と暫定PG Chain Health成功を**異なるHEAD/Attemptのまま合算しない**。
- **移行Ownerの整理**：[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)の実文書`plan/design/TEI_MD_UNIFIED_AUTHORING_MIGRATION_2026-11.md`冒頭では、移行検収の主担当を`M0〜M6/TM-01〜14`へ一本化し、[Draft #2667](https://github.com/hat47x/tei-platform/pull/2667)の`M0〜M5/TMD-01〜12`を**履歴・参照**と規定（#2667当該文書の冒頭も同じ）。両方のCase IDを別々の受入Gateへ増設しない。#2621のOwner判断・具体的な新Grammar受理はなお別。§10の二経路L/M/Jは、正式移行計画の重複ではなく**Script固有の実行資格比較**に限定する。

### 11.2 原因別の停止と診断を混ぜない

下表は新たな汎用Status enumの提案ではなく、**既存の各Componentが返す観測**と、S0〜S8／A0で主張できる上限を対応づけるもの。

| Source / 認可 / 効果の条件 | 当該既存実装が示す限定観測 | Script起動で追加確認すべきこと／禁止推論 |
| --- | --- | --- |
| 現Loaderへ新`*.tei.md`、選択Parserなし | #3031 DraftではWorkspace Loadが`TEI_MD_PARSER_UNBOUND`、特定Source pathで停止 | Source manifestの全Requiredを把握し、実未Parseを明記。**Mainでは現時点で無診断の読飛ばしがあり得る**。拒否だけで新Parser実装完了とはしない |
| 旧`*.tei`が実Parserで成功 | 現Loader/構文の局所資格。RunPlanや実DBを意味しない | 同一Source Revision・実ロードArtifact・必要Rule集合・実Providerまで追う |
| PatternにCommandあり、既存Ruleが当該主体・操作・資源でdeny | #3034では`COMMAND_UNAUTHORIZED`、Executor呼出0の正負対照 | このCommandの拒否を「Runtimeの全Effectが0」とは言わない。前Step/外部Providerの先行効果を別記 |
| Rule Store失敗・主体やAction/Resource不明 | #3034は評価不能errorを`ErrNoGrant`と区別（主体/Action/Resource欠落もerror） | **拒否の確定**と**認可が評価できない**を違う投影にする。再試行は実要求・現在資格を再確認し、未送出証明範囲を保つ |
| Ruleがallow | #3034局所TestではToy Executorへ到達 | 実Providerの現在Grant、対象行/時点/Period/CAS/Receipt、業務Meaning/Script受理まで合格とはしない |
| 宣言Effectあり／Executorが`Effect`を未報告 | main #3112で`ReportedEffectMissing=true`を記録（**自動拒否はしない**） | Reportの欠落とCommit失敗を同一視しない。Provider Receipt/別接続Readで実効果を独立確認するまで業務効果はunverified |
| 宣言Effectあり／`Applied`の列挙なし | main #3068/#3104で`AppliedUnobserved=true`。効果非宣言ならfalse | 操作の列挙欠落を「差なし」としない。falseも効果非宣言の場合は照合完了でない |
| 実Commit Receipt成功だが宣言対象／対象集合の実Read未成立 | 既存#2853/#3025のPG側Receiptは当該Provider効果だけの証人 | 宣言Effect・適用Row・業務Completionへの照合を別列で`unverified`にする |
| 実Commit後ACK-loss | #2853実PGでは同Operation ID+同Key Receipt lookupの限定成功 | RunPlan/Script/Host全工程と同じAttemptの証明でない。新Idempotency Key/ABAを許可しない |
| Read完全性・手作業Handoff未証明 | #2926`EvaluateCompletion`は完全性や空集合証拠を呼出側に要求 | Process exit/Command commitに加え、独立TargetSet・Coverage・Predicate・Handoffを確認するまで`holds`としない |

**現在の`CommandAuthorizer`の具体的な拘束不足**：`Subject`はHostが渡す文字列で、認証済み本人との一致は呼出側の責務。`Action`を省略するとCapability ID、`Resource`を省略すると**Application ID**が使われる。受注`ORD-1`等の**行・業務Target固有の許可**が必要なのにResourceをApplicationに丸めた場合、Ruleのallowから対象行の許可を導けない。対象行のID/Scopeへ写像する明示`Resource` mapper、Hostの実Principal、評価時点とCommit直前Fenceが必要。これらは#2499/#1174へ委ね、#3034の既存Adapterを二重実装しない。`Auth evaluation failed`を`no grant`へ丸めると監査と運用の意味が変わるため、UI/CLI/AI投影でも区別する（内部Rule/存在件数/秘密情報は開示しない）。

### 11.3 同じ固定業務Caseで必要な“次の一試行”を定義する

**外部ハーネス接続の確認範囲**：[PR #3113](https://github.com/hat47x/tei-platform/pull/3113)は2026-11-07に、固定Windows/Go実行物からDSH経由で`commands`/`version`/`model catalog`/`inspect`/`source-index`/`validate`を実行した**read-only実測Evidence**を提出（56 tool call／うち実TEI CLI起動32、`validate ok`・警告3件、9試行中8試行の停止理由も記録）。しかしこれは`Function→Query→実Provider Reader`ではなく、外部Harness側の独立Expected採点・Skill H3・書込みも未実施。Sandbox `read-only`でWorkspace外の可読パスへのアクセスが可能だったため、**read-only=参加者Source読取範囲の隔離**と推論しない。#2509/TAI #168へ実行環境・公開するSource/ログ送出・隔離の受入を引渡す。本PRに第二HarnessやSession保存正本を作らない。

**候補Scenarioは2段階**：最初のread-only P0-Rでは[#2586](https://github.com/hat47x/tei-platform/issues/2586)の既存Function/Query Contextから**実際に解決・実消費できる一つのQuery/Provider Reader**を選び、Source/Scope/Result Bodyを独立Expectedで検査する。実Reader/Bindingが未接続なら`UNBOUND_DATASOURCE`／`product-read-not-connected`を止まる理由として残し、合成MemoryStoreを実DB資格へ昇格しない。その後の書込みP0-Wでは、#3034既存Referenceの`order_entry.submit_order`／Capability `sales.submit_order`、`order=ORD-1`、Rule Resource `sales.order`をベースにする。業務Ownerによる意味受理や本番Resource mappingが済んでいると仮定しない。既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)のScenario／[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expectedへ対応付け、別のScript用Test SSOTを作らない。

| 一因子の違い | 実証したいExpected（今回のGo/PG統合試験ではNOT RUN） | 根拠/Owner |
| --- | --- | --- |
| 旧`*.tei`のSource、実Loaded/Parsedと正常Script入力 | 必須Source/Role/Ruleが実Consumed、同一Function/Target/実Read Bodyを照合できる | #2586/#2823、現Loader |
| 新`*.tei.md`にだけ入替、選択Parserなし | **新形式を未解析のままacceptしない**。#3031導入済み環境なら`TEI_MD_PARSER_UNBOUND`で停止、Commandは呼ばない | #3031 Draft。これを新形式E2E PASSとしない |
| 同じCommand/Rule、`Subject/Action/Resource`のGrantだけfalse | `COMMAND_UNAUTHORIZED`、当該Executor 0回。許可のある正対照はExecutor 1回 | #3034 merged局所Referenceを再利用 |
| Host Resourceを意図したOrder IDからApplication IDへ落とす | 行単位Grantが必要なScenarioでは、要求資源に対する資格を主張**不可**。明示Resource mapで正負対照 | #2499/#1174、現Adapterのfallback事実 |
| 同じ有効なSource/CommandでRule Storeを失効/故障させる | Rule評価errorで保留。確定した拒否`ErrNoGrant`と異なる、当該Actionを無条件再Dispatchしない | #3034評価不能の既存Go試験 |
| Commit後の応答だけ失わせる | `unknown`→同Operation/同Key Receipt lookupと独立Read。別Key+ABAは安全とみなさない | #2853/#2854/#2870既存実PG負例、#1174 |
| Readの対象集合をpartialにし、外部手作業確認も欠ける | Provider ReceiptだけでCompletion `holds`へ昇格せず`unverified`の根拠を表示 | #2926実Evaluator、#1125/#2514 |

**次の順序・採用を要しない仮説**：
1. **既存コードを再利用する読取専用縦切り**：旧`*.tei`の一つのFunction→Query→選択Datasource/Provider→実Reader Bodyを**同じ実Loader World・Source Revision・Scope**で固定する。Command AuthorizerやDB Commitをread-only初回の必須前提にしない。結合前に、現APIに渡すRequired Guard/実Row Targetが独立Expectedから揃うか調べ、不足はunsupportedとして停止。
2. **新形式の停止検証は別Lane**：#3031のDraft HEADが固定した`TEI_MD_PARSER_UNBOUND`の正負対照を再利用し、#3030のParser/Source-span実装までは新形式を無条件に有効化しない。mainに実装済みとは記載しない。
3. **許可を持つ本当のWriteだけ実PGへ**：#3034の`CommandAuthorizer`（Subject/Resource明示）・#2882のRequired Guard・#3002のopt-in Durable Stageを、既存#2853実PG Command/Receipt/別Connection Readに**同じCommand ID/Operation ID/Attemptの明示対応**で接続できるか検証する。Stage保存の成功やRef Executorの`Applied`をProvider Receiptにしない。Provider Commit時の再照会・事前/事後のGrant失効、Crash/ACK-lossは別試験。新しいRuntime Host/Policy Engine/Receipt DBは作らない。
4. **受入パケットのClaim ceiling**：一回の試行に`Required Obligation / Actually Read Source / Subject+Resource mapping / Evaluated Grant / Actual Dispatch / Effect Receipt / Independent Read / Completion / Actual Consumer body`を並べ、欠落を`not-run/unknown/unsupported`で明示。既存#1163のG1/G2判定を変更しない。

**残余と資格**：#3031はDraft限定Goテスト、#3034はmain実認可Adapterの局所試験、#3112はmainの効果報告欠落記録、#3113は外部DSHのread-only CLI実測、#3025は実PG暫定Chain Health。**Script Source→Auth→RunPlan→PG→Independent Read→Completion→Consumerの一改訂同一Attempt**、正式Host Grant、Provider Commit時の最新認可、新`*.tei.md`の実Parser/Read、**当該Scriptと実Providerを跨ぐOS再起動**、Owner正式意味受理は**NOT RUN / UNQUALIFIED**。main #2942/#3002のCommand単体・CLI段階storeのOS再起動試験は既に存在するが、その証拠をScript/Provider一改訂の合格へ移さない。
### 11.4 最小実装・検証チケット：結果の「未観測」を合格へ変えない

**一つ目の検証作業はP0-Rの「欠落した一辺」**に固定する。現main `examples/xtea-lowcode`の`UNBOUND_DATASOURCE`と、#2586のQuery/Function Data Contextを起点として、`Function/Query Source→明示DataSource alias/instance→実選択Provider→実際のSQL/params（存在時）→実Reader Rowset/Scope/時点→Consumer Body`のうち**最初に未接続の一辺をSource/既存コードで特定**する。製品Readerに接続できない場合は`product-read-not-connected`として止め、MemoryStore、DSH`validate ok`、外部Harnessの成功数を代替E2成功にしない。

**同一Source/Fixtureを固定する正負対照**（新Scenario IDではなく既存#1180/#2586/#2514のCaseに対応付ける）：

| 1因子だけ変更する条件 | 独立Expectedの判別力 | 実装Owner／合格上限 |
| --- | --- | --- |
| P0-R：存在するProvider bindingと実Reader、正しいTarget Scope | Source・Query・選択Alias・Provider実体・実行SQL/params・Result byte・Snapshot・Consumer Bodyが一改訂で追跡可能 | #2586/#2622。**現時点未接続／NOT RUN** |
| P0-R：同一QueryでProvider未束縛 | 実Provider Readが呼ばれない。未取得を空集合・0行・完了と表示しない | #2586/#2823。合成Sourceの`validate ok`だけはfail |
| P0-R：DataSource A/Bの選択を変更 | 実際に使われたProviderとSQL/paramsが選択に一致し、未選択Aが呼ばれていないことを追う | #951/#2622。Provider Selection証拠なしならunverified |
| P0-R：rowset 0、部分ページ、NULL、非開示を差替え | 完全な0行と未取得/partial/非開示は異なる。必要なOutput Grain/TargetSet/CoverageによりClaimが変化 | #2586/#2516/#2926の既存契約を利用 |
| P0-W：同一Command/同一TargetでEffect reportだけ欠落 | `ReportedEffectMissing`を観測するがCommit/業務効果の否定に直結させない。Receipt/独立Read別 | #3112/#2996。効果報告とProvider Receiptを別Claim |
| P0-W：適用操作の列挙だけ欠落 | `AppliedUnobserved`を保持し、列挙0件を「差なし」としない。効果非宣言なら比較対象なし | #3068/#3104/#2998。実Provider行単位確認は未実施 |
| P0-W：Commit応答喪失、同ID再試行、異Key/ABA | unknown時に新Key盲再送しない。同OperationID/Provider Receipt＋別Connection Readで限定確定 | #1174/#2564。実同一Attempt Script連結はNOT RUN |

**既存Evidence Packetへの入力列**：`Case / Claim / Source+Actual Binary digest / required-discovered-loaded-parsed-consumed / selected Query+Provider+parameter digest / subject+resource+current grant / Attempt+Operation ID / declared Effect / reported Effect present / Applied operations observed / Receipt+independent read / TargetSet+Coverage+Completion / Consumer disclosure / Result+unknown reason`。これは新たな公開Schemaではなく#1163の**一試行受渡し表**。機微パラメータのbyte・DSN/secret、権限外Row/内部Ruleを外部Harnessへ渡さない。digestのみなら元byteの再現性/Record真正性を主張できない（#2845）。

**Exit条件**：最初はread-only Queryの`実Source＋実Provider Selection＋実Reader結果＋独立Expected`を一つのAttemptに結ぶことだけを受け入れる。次に書込み側の`実Auth/Guard＋Durable Stage＋Provider Receipt＋独立Read`を同じOperationへ束縛する。外部DSHのSkill比較/実Sandbox隔離は#2509/TAI #168、`*.tei.md`移行は#3030、履歴の改変防止とDisclosureは#2845/#2516へ渡す。**以上は実証計画であり、新テストやOwner受理はこのPRで実行していない。**

## 12. 宣言効果・Receipt・永続化観測の分離と最短受入（2026-11-07）

本節は旧Draft [#2884](https://github.com/hat47x/tei-platform/pull/2884) を**最新mainベースへ再構成する際の新しい成熟差分**。11章までの旧設計は過去の観測時点を保持する。**本節の現状表と各PRの最新状態が優先**し、古い「未実装」「未実証」の記述を現在の全経路へ一律に適用しない。

### 12.1 時点別のContract差分（現在の実装資格は§13を優先）

| Mechanismと出典 | 実装・実証済み範囲（またはDraft候補） | Script受入で主張してはいけないこと |
| --- | --- | --- |
| [main #3118](https://github.com/hat47x/tei-platform/pull/3118) `commandadapter.Executor.Effect` | Callerが**宣言した**EffectをAdapterが`Committed`／`Replayed`の結果へ載せ、`ReportedEffectMissing`を解消し得る。Effect未指定時はreceipt-onlyの後方互換 | Caller提供の宣言を返しただけでは、**DBの対象Row/Fieldが実際に宣言どおり変わった**ことを独立に証明しない。Replayは現在値の観測ではない |
| [main #3124](https://github.com/hat47x/tei-platform/pull/3124) `PersistenceObserver`／`PersistenceState` | `not-applicable`（非永続／宣言Effectなし）、`not-observed`（観測未構成・失敗）、当時の`observed`を分離。現mainは#3155により`receipt-observed`と`resource-observed`へ分化。Commit後Observerエラーで正常Commit自体を取り消した扱いにしない | `DurableStore=true`はCompositionが供給する主張であり、InvokerがStore実体を検証した証拠ではない。Observedの**何を見たか**を記録せず`persisted=true`を汎用業務効果確認としない |
| [main統合 #3127](https://github.com/hat47x/tei-platform/pull/3127)／[main #3131](https://github.com/hat47x/tei-platform/pull/3131) `commandadapter.Executor.Lookup` | 同一Request/Grant/OperationID/payloadで`Runner.Lookup`を使ってReceiptを観測し、実PostgreSQLの局所試験も成立 | **LookupはReceiptの確認でありResource現在値ではない**。現main #3155は`receipt-observed`と`resource-observed`を区別し、`persisted:true`は後者のみ |
| [main #3123](https://github.com/hat47x/tei-platform/pull/3123) `access.FileSystem` | Root外へのアクセス試行を拒否時に記録（記録失敗でも拒否は継続） | Access GuardのRoot境界と**外部DSH Harness SandboxのOS読取隔離**は別。拒否ログの欠落から「試行なし」を導かず、ログの真正性・完全性も未保証 |
| [外部実測PR #3113](https://github.com/hat47x/tei-platform/pull/3113) | 外部Harness→固定TEI CLIのread-only実起動を証拠化 | Tool/CLI成功・validate ok・world fingerprint一致から、実Provider Read、参加者隔離、Source Meaning受理、業務Effectを推論しない |

**5段階の確認順**：①`declared Effect`（受理Contractの主張）→②`Executor reported Effect/Applied operations`（実装側報告）→③`Provider Commit Receipt/Idempotency Lookup`（実処理記録）→④`Providerの対象Row/Fieldを別接続・同ScopeでRead`（現在Data状態）→⑤`Completion(TargetSet, Coverage, Predicate, Hand-off)`（業務条件）。**③で②が返っても④が成立するとは限らず、③のReceipt保存と④の現在値が異なることも正常な履歴としてあり得る**。同一Operation IDとRef/Revisionの対応を保持し、異なるClaimを同一のBooleanやUI「成功」へ畳まない。

### 12.2 追加の独立Expected（第二のFixture/判定エンジンなし）

以下は[#1180](https://github.com/hat47x/tei-platform/issues/1180)の既存Scenario、[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立検証基準、[#1163](https://github.com/hat47x/tei-platform/issues/1163)の一改訂Evidence Packetへ**追加入力として対応付ける設計**。新Case IDを正本として増やさない。

| 一因子変更 | 期待する独立判定（この一体試験はNOT RUN） |
| --- | --- |
| 実Commit成功、Caller宣言EffectをAdapter結果へ返す（#3118） | `ReportedEffectMissing=false`になり得る。ただしCallerとAdapterの同じ値を照合するだけで`AppliedRowConfirmed`ではなく、Receipt・独立Read資格は別 |
| 実Commit成功、Executor Effectだけ欠落 | `ReportedEffectMissing=true`でもCommit失敗と断定しない。Receipt/RowReadは独立に実施 |
| `DurableStore=false`、または比較するEffectがない | `PersistenceState=not-applicable`。「観測済み」と「観測不要」を混同しない |
| Durable Store主張あり、Observer無し/失敗 | `PersistenceState=not-observed`。Commit済みなら再送判断はReceipt LookupとOperation Identityに基づく。Observer失敗で二重実行しない |
| 同KeyのProvider Receiptは`StatusReplayed`、対象Rowは他者に更新/削除された | **Receiptはobservedでも現在値は未確認/異なる**。現在Rowの読み直しなしに、宣言Data EffectやBusiness Completion `holds`へ進まない |
| 同ID/KeyでもHost Grant・Tenant/Target Scopeが変化 | 古いReceipt/報告を別Targetへの認可・副作用資格に流用しない。Host Grantの現在資格・Disclosureを再評価 |
| read-only外部AgentがRoot外のファイル読取を試行 | TEI Access Guard通過時の拒否記録と、DSH Sandbox自体の隔離を別観測。前者が実装済みでも後者は未知として扱う |
| 実Readが0行／partial／RLS等で非開示 | 証拠の完全性・TargetSet Scope/Versionが無ければ「不存在」も`Completion holds`も推論しない |

ここで`AppliedUnobserved`と`ReportedEffectMissing`の二つは、**Resultの列挙・報告能力の制約を示す**欄であり、Commitの実証でも実Data状態の否定でもない。現mainの`PersistenceState`では`receipt-observed`と`resource-observed`の確認対象が異なり、`persisted:true`は後者のみ。Scope・時点・Versionと業務完了はさらに別に検証する。Receipt lookup用ObserverをData state Observerへ名前だけで流用しない。

### 12.3 統合の順序と停止線を、現行の実機構へ束縛

- **P0-R（先行・read-only）**：#2586と`examples/xtea-lowcode`の`UNBOUND_DATASOURCE`を起点に、既存Product `Function/Query→Datasource Alias/Instance→選択実Provider→実Read→Rowset/Scope/Body`の**最初の未接続の一辺を確定**。可能なら1件の独立Expectedとの対照を実試験。未接続なら`product-read-not-connected`を明記し、CLI/外部Harness成功に代えない。
- **P0-W（その後・write）**：選択した同一Business Operationを、既存`CommandAuthorizer`・Required Guard・`DurableLifecycle`・`effectcommit.Runner`・#3127/#3131 Receipt lookup・#3155 `ResourceRead`（別poolの現在値読取）へ接続する。`Lookup=true`は`receipt-observed`まで、`ResourceRead=true`はpayload一致を実観測したときだけ`resource-observed`。実PG局所試験をScriptの一改訂E2Eと取り違えず、skipをPASSに数えない。Consent/OwnerのMeaning/Grantは別責務。
- **Consumer/AI**：CLI/Studio/DSHへは実Target/Scopeに対するDisclosure・raw secret非開示・判定不能の理由を投影する。Root拒否が記録されてもDSHサンドボックスの読取隔離証明が無い限り非公開SourceをParticipantへ渡さない。
- **移行レーン**：`*.tei.md`の正式Reader/Writerは[#3030](https://github.com/hat47x/tei-platform/pull/3030)、暫定fail-closedは未統合[#3031](https://github.com/hat47x/tei-platform/pull/3031)。旧`*.tei`での局所Read/Effect資格を新形式へ転記しない。

**統合資格**：本書は設計と実装Fit/Gapを示すDraft記録。#3131/#3155のReceipt・Resource Readの局所実PostgreSQL試験は既に成立。一方、Script→Function/Query/実Provider→Completion/Consumerの**同一Source・同一Attempt E2E**、二Host/正式Owner Meaning/本番Grantは**NOT RUN／NOT QUALIFIED**。#1119 G1/G2 Gateや業務受理を代行しない。古いDraftの変更履歴は[#2884](https://github.com/hat47x/tei-platform/pull/2884)に保存し、本書を**最新main起点の後継候補**として扱う。旧PRは後継PRが作成・差分検証できた後に重複を解消する。

## 13. Product Browse→Query→実Provider Readの未接続辺と回復資格（2026-11-07）

本節は、未統合Closeされた[後継Draft #3129](https://github.com/hat47x/tei-platform/pull/3129)の実装接続検討を現mainへ回収し、実コード・新しい隔離PostgreSQL実証に照らして絞り込んだ成果。§1〜12の当時の「未実装」記述より、**現在の実装状態と検証の資格**について本節を優先する。既存の[#1180](https://github.com/hat47x/tei-platform/issues/1180)のScenario、[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expected、[#1163](https://github.com/hat47x/tei-platform/issues/1163)の一改訂Packetを再利用する。

### 13.1 次の製品実Readで最初に接続すべき一辺

現mainの`internal/applicationview/browse.go:BuildBrowse`は、Browse PatternのQuery IDを`world.Queries`へ照合し、Application一致、ResultColumn→Semantic Attribute、Filter/Sort/PageSizeなどを表示用`BrowseView`へ解決する。**Pattern→Queryの表示選択は実装済み**。

現mainの`cmd/tei/cli_view.go:runViewCommand`は、`PatternBrowse`分岐で`BuildBrowse(loaded.World, applicationID, patternID)`を呼び、表示定義を`result=view`として返す。この**製品CLI入口には実SQL/paramsやProvider呼出し・Rowset受領がない**。別Product Callerの不在を全探索したわけではないため、実Reader全体の不存在とは断定しない。

**P0-Rの最初の不足辺**は、既存`reference/workspace`の`order_entry.recent_orders`を一つ選び、実Function/Patternから選択した`Query ID/Source revision`と、`Runtime DataSource Alias/Instance→実Provider→actual SQL+params→受領Rowset/Scope/時点`を*同じAttempt*で結ぶ**Caller側の接続**。`examples/xtea-lowcode`の`UNBOUND_DATASOURCE`は読取未束縛の負例として参照するが、別Workspaceの合格実績へ混ぜない。

[main #3195](https://github.com/hat47x/tei-platform/pull/3195)・[main #3204](https://github.com/hat47x/tei-platform/pull/3204)のOperation/Pattern Crosswalkは、Data/Capability/Relationの限定照合と衝突・非対応を観測する参照用。W3 Product Operation Binding/自動Invoker経路の採用ではない。**O（Function/Pattern→Operation/Invoker）とR（Function/Query→実Provider Read）は別の接続辺**。read-only P0-RにW3新実装を必須前提として持ち込まない。

### 13.2 実PGの新しい前進を、過不足なく受入する

[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238) fixed HEAD`469cdbadb1fcf23d65526ec3c5899f4d94e8390b`の[専用PG CI #42346176936](https://github.com/hat47x/tei-platform/actions/runs/42346176936)はcompleted/success。隔離PostgreSQL16.15・SKIPなしで`ScopedPostgresOrdersReference→ExecuteScopedCanonicalRead`を実行し、Tenant Aの事前固定Expected L={3,9}に対して`$top=2→[[3],[9]]`、`$top=1→[[3]]`の実Rowsetを確認した。Q05〜07の局所Completion反例も限定PASS。

これは**既存Reference経路での実物理Reader/有限Result Sliceと独立Expected**に限る。TOP1に誤って`CoverageComplete`を与えれば局所`holds`になり得るため、`EvaluateCompletion`がCallerのCoverage・TargetSet・P/非出力依存を自己認証するわけではない。#3238は未統合であり、**製品Functionが選択したQuery→実Provider/SQL→Consumer**の証拠でも、全Repo Health資格でもない。

### 13.3 最小実装・一因子負例とExit条件

| 段階 | 既存経路へ加える最小確認 | Exit条件／不合格で止める範囲 |
| --- | --- | --- |
| R0：Source/Query選択 | `BuildBrowse`のApplication/Pattern/QueryとResultColumn、実Loader/World/Source版 | **表示定義だけならR0止まり**。`modelRevision`は管理Sourceの版であってWorld版ではない（main #3163）。宣言0件/preludeと管理Sourceの実読込を分離 |
| R1：Provider Binding | R0から既存DataSource/Provider実行IFへ接続。選択Alias・Instance・dialect/SQL+params・Grant/Tenant/Scope・Reader実呼出しの実結果 | 初回は**呼出し側で未接続の1箇所**を確定し、同一Function/Fixtureで実Binding正例と未束縛負例。未束縛は**0行でなく未取得**。別Reference Readerの成功でProductの資格を補わない |
| R2：Result/Claim | #2506の実Result Slice（occurrence,grain,rows,dedup,limit,page,actual field presence）と#2514のQ05〜07を接続 | HeaderページのTOP1、重複、NULL/missing/withheld、非投影Predicate評価と全対象集合L/Completionは別。実PG Reference L={3,9}を別Sourceの製品Lに流用しない |
| W0：Script/Effect（後続） | OのCapability/Operation・現在Grant/Target、#3127/#3131 Receipt lookup、#3155の別接続`ResourceRead`を同一Operationへ束縛 | `receipt-observed`と`resource-observed`、業務完了を別評価。`ResourceRead`のpayload bytes一致はValue A→B→AのVersion因果・現在Grant・明細集合全件を証明しない |

**最初の実装対象をR1のCallerの未接続部分に限定**する。今ある`BuildBrowse`/ReadPlanを破棄せず、既存Providerでの最小実Readerを利用する。技術的接続が不足ならSource位置と不足IFを記録し、独立Expectedの緑を偽装しない。新Query Engine／Provider Registry／別Host／第二Test Fixtureを作らない。

### 13.4 回復・外部Agent・表現型の非推論

- **RC-03/05は残る**：現mainの`internal/command/invoke.go`ではExecutor後の`Recorder.Record`エラーを`COMMAND_RECORD_FAILED`として返すが、そこで`EffectUnknown`が付かない。`internal/command/lifecycle.go`の`EffectMayBeUnknown(err)`による分類で、Executorの別Txが先にCommitしたときClean Rejected扱いになる懸念がある。これは[#1174](https://github.com/hat47x/tei-platform/issues/1174)の**実Sourceで確認できた故障仮説**であり、実故障注入Go/実PG確認はNOT RUN。[main #3197](https://github.com/hat47x/tei-platform/pull/3197)によるExecutorの同一Tx参加は未契約。Rollback成功=外部Effectなしとしない。次のW0にはCommit後Recorder/Stage記録失敗→Receipt/別Readを一因子負例として加える。
- **Function内の表現**：[Draft #3236](https://github.com/hat47x/tei-platform/pull/3236)の標準Data Shape・機能非依存Presentation Type・Function所有Instanceと、Data→View→Consumerの写像は、**実R1 Rowset/Scope/Presence/Disclosure**の上に置く別証拠。Selection同期と表示上の0件から実Dataの不存在・更新Target・Host Grantへ昇格させない。Ownerは#2507/#997/SHO/SUI側。
- **公開と自律**：[main #3210](https://github.com/hat47x/tei-platform/pull/3210)で`tei operations`はCLIから利用できても`tei commands`の選択公開外。[Draft #3231](https://github.com/hat47x/tei-platform/pull/3231)の`unattended`はread-only由来に留まり、実Service Subject/現在Grantは未接続。`view -execute`の操作ログ/Lifecycleローカル書込み（Draft #3162）も、Business read-onlyや外部AI送出とは別属性。DSHの実Tool読取/Session Log/開示隔離は#2509/#2515/TAI #168へ移し、Client表示の権限をHostの正式権限としない。
- **Evidence境界**：Source/Query/実バイナリ/Provider/actual SQL+params（機密値非公開）/Result Slice/Owner L/P/OperationID/Receipt/Resource post-image/Consumer実受領を、#1163の既存PacketでClaim別に紐付ける。OKF/SHOのAttested記述は検査内容・根拠の来歴であり、Meaning/Grant/実Effectの権限ではない。

### 13.5 Operationの陳腐化・記録障害・外部Session復帰を同一境界で誤認しない

[main #3254](https://github.com/hat47x/tei-platform/pull/3254)の実測では、`reference/standard-crud/workspace`の属性を1つ追加するだけで`worldFingerprint`／`modelRevision`が変化し、Operation 5件の`basis`は**すべて不変**だった。現`tei operations`には`-expected-fingerprint`が無く、実行時`period_guard`も操作面の版は照合しない。このため、**Operation ID/Basisの一致は、Source/World/Operationが現在有効な証拠ではない**。`basis`は依存根拠のFingerprintとして保持し、世界全体の版であるかのように再定義しない。

**Script起動・回復の独立Expected**は、既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)／[#1163](https://github.com/hat47x/tei-platform/issues/1163)と[#1174](https://github.com/hat47x/tei-platform/issues/1174) RC-03/05に対応付け、以下の一因子対照を優先する。

| 一因子を変えた条件 | Admission／Recoveryで求める区別（本Script経路の新実走は未実施） |
| --- | --- |
| 同Operation IDと`basis`のまま管理対象Source属性だけ変更 | **実Source・World・選択Targetの現在性を再検証**。改版前の許可/Planを自動再利用しない。`worldFingerprint`不一致の扱い（再導出/停止/再受理）はOwnerの実行Contract選択に委ねる。新エラーコードを採用済みとして書かない |
| 同じSource/WorldでTenant・Target Resource・Actor/Grantだけ変更 | Operationが同一でも**現在Host Grant/Target**は別。対象集合を実Providerへ渡す直前に比較し、過去の認可/別TenantのReceiptを再利用しない |
| Executor別Tx Commit済み→`Recorder.Record`だけ失敗（RC-03） | `COMMAND_RECORD_FAILED`を*Clean Rejected/Effect 0*へ昇格させない。Stageを可能な限りunknownとして保持し、同Operation IDのReceipt Lookup＋別接続現在値Readで限定確定。実コード`EffectUnknown`伝播の修正要否は#1174が所有 |
| Executor別Tx Commit済み→`Lifecycle.Observe`失敗、または`MarkUnknown`自体失敗（RC-05） | 記録失敗がある状態で**安全に再送可能**と表現しない。Stageの実耐久性を観測し、Receipt/Target/Operation Identity一致まで再試行資格を保留。Ack喪失と補償は別のRun/判断 |
| 同OperationのReceipt Lookup一致、ResourceのValueがA→B→A | `receipt-observed`と現在値byte一致だけで**Version/因果的Postimage**を確認したとしない。`ResourceRead`のVersion・Read時点・同一Target/Grantを別欄に残す。今のObserverがVersion比較を保証するとは推定しない |
| 非公開Rawを過去Sessionで読み、Sessionを再開する | [未統合 #3237](https://github.com/hat47x/tei-platform/pull/3237)の**合成DSH実測**では既送出Session Logの水位は進み再送が止まっても、モデル入力に以前のCanaryが再登場した。**Session Logの非再送≠モデル入力からの消去**。再開時は現在Disclosure/Task/Session資格を再判定し、過去のRawが新要求に載らないと推測しない |
| 保存されたJournalの鎖・pinは一致、別Provider実Commit証拠は欠落 | [main #3230](https://github.com/hat47x/tei-platform/pull/3230)の`PinnedJournal`は接続した固定World Buildの**末尾削除・追記時検査**を限定観測する。署名でも全Journalへの適用でも実Provider Receiptでもない。Source/Run/Effect/ConsumerそれぞれのReceiptを代用しない |

**同一Attemptへ追加する実観測列（新公開Schemaではなく既存Evidence Packetへの手渡し）**：`Source path+bytes/manifest identity（観測時）/World FP/Source Index FP/modelRevision/Operation ID+basis/selected Function+Pattern+Query/actual Binary/Actor+Tenant+Target+current Grant/Provider SQL+args digest+scope/Stage before+after/Recorder+Lifecycle outcome/OperationID+RequestHash/Receipt lookup/Resource value+version+asOf/Consumer delivery+Disclosure decision`。実Providerとの接続がない列は`unverified`とする。実SQLパラメータやGrant、非開示Row、Session Rawを外部AIの記録へ丸ごと転載しない。

**実装順と責務**：P0-Rの最初の作業は引き続き§13.1〜13.3のProduct`BuildBrowse→selected Query→実Provider Reader`接続であり、Operation SurfaceのW3全実装や新Snapshot/Authority基盤を先に要求しない。次にP0-Sとして**実行直前のSource/World/OperationとTarget/Grantの照合口**を既存RunPlan/Host Admissionに探し、なければ最小の不足を#2502/#1130へ引渡す。最後にP0-WでRC-03/05を#1174の既存Stage/Receipt/ResourceReadに繋ぐ。外部SessionのRaw再露出とEgressは#2509/#2515・TAI #232、Journal pinの適用範囲は#1174/#2845、Product Readは#2586、Ownerの業務Meaningと対象集合は#2506/#2514が受理する。第二のPolicy/Query/Recovery Engine・Fixture正本は追加しない。

### 13.6 実行境界の切断点：UnitOfWork参加と記録失敗の区別

現main`internal/command/invoke.go`の実コードでは、`BuildCommandPlan→Authorize→DataSource/Period Check→任意のUnitOfWork.Begin→Executor.Execute→Effect/Applied/Outcome Check→Recorder.Record→任意のUnitOfWork.Commit→Persistence Observe`の順で処理する。`LifecycleInvoker.Dispatch`はInvokerの前、`Lifecycle.Observe/MarkUnknown`は後である。**Recorderは外側UnitOfWorkのCommitより前だが、独立TxのExecutorが既Commitした後にはなり得る**。`Executor`は同一UoWへの参加を契約上要求していない（[main #3197](https://github.com/hat47x/tei-platform/pull/3197)）。したがって、`UnitOfWork != nil`から参加やRollbackの有効性を推測しない。

[#1174](https://github.com/hat47x/tei-platform/issues/1174)の既存RC-03/05へ接続する**実装担当向け故障注入仕様**（新しいScenario正本を作らない）を次のように固定する。

| 切断点（一因子） | 事前に固定する独立Expected | 現mainに関する確認・未実証 |
| --- | --- | --- |
| A：事前認可・期間拒否で`Executor`未呼出 | 実Executor呼出0、Providerへの書込み0を証拠で確認。`Dispatch`の記録は既にあり得るためStageと効果を混同しない | 実コード上の順序は確認済み。実Providerの拒否時readbackは未試験 |
| B：Executorが型付きEffectUnknown errorを返す | 元のUnknownを保持し、同一Owner/Generationの盲再送を拒否。外側Rollbackだけで独立Txの効果0を断定しない | 既存Command/Lifecycle局所試験がある。実外部Commit/別接続Readの同一Attemptは未試験 |
| C：独立TxでCommit後、`Recorder.Record`だけ失敗（RC-03） | `COMMAND_RECORD_FAILED`だけではClean Rejectedを証明せず、実Receipt・別接続Postimageまで`effect unknown`に止める | 現`invoke.go`は`EffectUnknown`なしで返し、`LifecycleInvoker`が通常`OutcomeRejected`へ進み得る。実故障注入は未試験 |
| D：Cと同じ失敗だがExecutorが外側UoWの**同一Tx参加を証明**し、Rollback完了 | 対象Providerのpost-image/Receiptが更新されなかったことを別途確認してから、**当該Effectに限り**非適用の候補 | 現`Executor`型から参加は分からない。参加未証明ならCと同じUnknownとして保留 |
| E：外側UoW Commit済み→`Lifecycle.Observe`だけ失敗（RC-05） | CommitとStage永続記録を別々に照合。記録失敗で効果0や再送可能としない | 現`lifecycleRecordError`の外側エラーにはUnknown flagがなく、保存Stageの成功は保証しない |
| F：元のUnknownに続く`MarkUnknown`も失敗（RC-05） | 元ErrorのUnknown、Stage記録失敗、Provider結果を別記録し、Error包装でUnknownを消失させない | 現`EffectMayBeUnknown`は最初に一致した`ExecutionError`のflagを読むため、外側falseが内側trueを遮り得る |
| G：同Operation ID／Basisのまま実行直前Source/World改版 | 検査時と実消費時のSource Closure/World/Target/Grantを比較し、相違なら既存Admission Ownerへ再受理要否を返す | [main #3254](https://github.com/hat47x/tei-platform/pull/3254)は属性1つでWorld/Model版が変わりBasisは不変と実測。実行時照合の製品実装は未資格 |

**最小の実施順**：①A/C/E/Fの既存Go型/`LifecycleInvoker`を使う**Characterization**を同じRequest/OperationIDで行い、失敗コード・`EffectMayBeUnknown`・Stageを独立Expectedに突合。②C/Dを**別のExecutor取引参加条件**で差分検証し、Commit前/後の独立Resultを区別。③本番DBを使わず、許可された隔離PostgreSQLでC/EのReceipt＋別接続Readの故障注入を行う。④Gは既存`Operation Surface`の出力と実行直前Loaded Sourceを一Revisionで比較するAdmission試験に分ける。実Providerを再実装せず、既存#1180/#2514/#1163に結果を引き渡す。

**同じRunに束縛する最小欄**：`Source/World/OperationID+basis/actual Binary/Actor+Tenant+Target+current Grant/Provider Instance/RequestHash/ExecutorTx participation（proved, unproved）/UoW begin+rollback+commit/Recorder error/Stage durability before+after/EffectUnknown provenance/Receipt Lookup+resource version+Read epoch/Result Slice+Coverage+Consumer Disclosure`。不明な欄は`unverified`、秘匿Input/Grant/非開示RowのrawをTool/AI Sessionに転載しない。**新公開SchemaやRun Registryの採用ではない**。

**到達点**：Reference物理PG ReadとCommand Receipt/Resource Read、Operation driftの合成実測、PinnedJournalの限定保護は個別の有限証拠。Product`Browse→Query→実Provider→Function/Consumer→Script Effect/Recovery`一改訂統合、RC-03/05実故障注入、Admission現在Grant/World照合、正式Owner Meaning、外部AI秘匿隔離は**NOT RUN／NOT QUALIFIED**。

### 13.7 回復の最小実装修正とDesign Context入力資格の再分離（2026-11-08）

本節は、§13.6のRC-03/05を**計画から局所Code+Test候補へ進めた**ことと、他レーンで実測されたP0-R前段のDesign Context Readinessを現状態に更新する。旧記述の「未実装」は当時のmainの観測であり、本節以降の判断では**main統合済み／Draft変更あり／実試験未確認**を分ける。

**RC-03/05の実装候補**：[Draft #3328](https://github.com/hat47x/tei-platform/pull/3328)では、`Invoker.execute`でExecutor成功後の`Recorder.Record`エラーを`COMMAND_RECORD_FAILED + EffectUnknown=true`として返し、`lifecycleRecordError`も`COMMAND_LIFECYCLE_RECORD_FAILED + EffectUnknown=true`を返す。加えて、`StageDispatched`なのに`MayReplay=true`だった参照Lifecycleの矛盾を修正し、**存在するAttempt（dispatched/unknown/observed）は同Owner/Generationの自動再送候補にしない**。専用Goテストは、①Recorder障害後のUnknown/再送拒否、②実行成功後Observe障害、③既Unknown後MarkUnknown障害、④事前Authorization不備はExecutor0/Rejected、⑤DispatchedとMayReplay/Dispatchの一致を固定する。**2026-11-08記録時点はPR Draft・未統合。Go CIは完了結果未確認**。旧§13.6 C/E/Fの静的「現main」指摘を、すでにProduct修正済みと読み替えず、候補PRとベースmainとの差として残す。

**このGo修正が保証しないこと**：`EffectUnknown=true`はCallerへ警告する情報であって、`Observe/MarkUnknown`失敗時の**Stageの耐久保存を修復しない**。保存された`dispatched`は二重Dispatchを拒否できても、Process crash/Journal失効/別Hostからの同Operation再送の完全Fenceを証明しない。ResourceReadの`value`一致はVersion/ABA因果、全対象集合、現在Grantを証明しない。次段は#1174/#2564の**既存**隔離PG Commit→Recorder/Stage障害→同Operation Receipt/別接続Resource Readを実試行で比較する。C（別Tx）とD（同Tx参加保証）を同じFakeのテスト成功から混同しない。

**P0-R前段の実証資格と停止位置**：

| 先行証拠 | 成立範囲 | このScript計画で認めない飛躍 |
| --- | --- | --- |
| [main #3284](https://github.com/hat47x/tei-platform/pull/3284)／[main #3296](https://github.com/hat47x/tei-platform/pull/3296) | CLI実9応答を受けた`context_gate.py`のReadinessは`partial`/`exploration-only`で、7＋追加Source Root 2＋受領byte 7の負例を判別 | Design Contextが完全・Meaning accepted・テスト生成/Provider実Readがqualifiedとはしない。別Packet/第二Parserを追加しない |
| [main #3299](https://github.com/hat47x/tei-platform/pull/3299) | 実Source型改変・Plugin Lock改変後の旧Pinned Scenarioを拒否する限定CI。byteがA→B→Aで戻ればContent Pin自体は再成立し得る | Pin一致から改版履歴・Owner意味受理・現在Grant・全Plugin実Loadを推論しない |
| [main #3304](https://github.com/hat47x/tei-platform/pull/3304) | 既知の正数Ruleをもとに、一時第7Stepの正負と偽Expected/Fixture/旧Pinの実CLI判別を実行 | 未知RuleをAIが盲検発見した／C1/NFR網羅・実Product Provider/Hostが適格とはしない |
| [main #3254](https://github.com/hat47x/tei-platform/pull/3254) | 属性変更でWorld/Model版は変わるがOperation`basis`が同一の合成実測 | Basisだけで実行時Source driftを検出したり、Operation ID一致からAdmissionを保証したりしない |
| [main #3310](https://github.com/hat47x/tei-platform/pull/3310) | `access.ChainFile`は判断レコード全欄のJSONとchainVersion`tei.access.chain/v2`をハッシュ入力へ含め、版不一致を明示 | 完全な防改竄、実Provider Commit Receipt、公開署名、全Journal/全Hostへの配備としない |
| [未統合 #3311](https://github.com/hat47x/tei-platform/pull/3311)／[未統合 #3302](https://github.com/hat47x/tei-platform/pull/3302) | Workspace名/manifest digestのCLI投影と、`view -execute`の名前付きOutput/`read-only`適用範囲の補正候補 | 両PRをmain実装済みとしない。`effect: read-only`と実argvのFile Write/Network Egress、Host Grantを区別 |

**一改訂で実験する順番**：P0-LでRequired/Discovered/Loaded/Parsed/Meaning-bound/Build-consumedを別列にし、Contextが`partial`なら**生成資格は保留したまま**P0-R1の実Source解析・実Reader Call Path調査を実施する。P0-R1のExitは**Product Pattern→selected Query Source→Provider Instance+actual SQL/params→Result Slice/Scope→Bound Function Consumer**の一Attempt束縛。DB Reader取得成功だけでも宣言済みTargetSet全件の`Completion holds`には進めない。P0-Wで#3328の限定Go Testと実PG故障注入を独立した段に置く。P0-Hで外部Harness受領・Disclosure・Session Log再送は#2509/#2515側に委譲する。新Runtime/Host/Second Expected Registryを作らない。

**所有と現状の上限**：RC-03/05 Code＝#1174/#2564（Draft #3328）、Function/Query Product Reader＝#2586、Design Context/AI Test Ability＝#3262/#2514、Source/BOM＝#2822/#3267、Operation Drift＝#3254/#2502、External Read/Egress＝#2509/#2515/TAI #168、Evidence Packet＝#1163。§13.7の記述は**新しい計画と実装候補の照合**であり、今回のScript Product E2E・実PG故障注入・正式Host/Owner Meaning・C1測定・実H3隔離は**NOT RUN／UNQUALIFIED**。
