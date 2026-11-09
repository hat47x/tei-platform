# Composite Function の改版・状態移行・実行世代の交差検証（2026-11）

- **状態**：設計・Pressure Fixture案／未採用（2026-11-02）
- **性格**：既存契約に対するread-only比較・検証計画。新しいCanonical Schema、State Migration Engine、Host、Authority、Resolver、共有Transaction Engineを採用する文書ではない。
- **主要Owner**：[#1223](https://github.com/hat47x/tei-platform/issues/1223)（実行世代/切替）、[#2507](https://github.com/hat47x/tei-platform/issues/2507)（Function Definition/Placement/Composition）、[#2597](https://github.com/hat47x/tei-platform/issues/2597)（編集Session/Save Intent）、[#988](https://github.com/hat47x/tei-platform/issues/988)（DSL/Operatorの意味）、[#1121](https://github.com/hat47x/tei-platform/issues/1121)（Lowering/Target）、[#1174](https://github.com/hat47x/tei-platform/issues/1174)（実EffectとCommit時の権限）、[#2582](https://github.com/hat47x/tei-platform/issues/2582)（変更の影響/再評価）。
- **関連**：#951 Provider Contract/Conformance、#1016案件構成、#943 Artifact、#1131 Scenario、#2514独立Oracle、#2598 Intent→実Request、#2497情報保存、#2502 Script/Macro改版、#2504 Script Execution Policy、#2493圧力Portfolios、[ADR-0032](../adr/0032-composition-kernel-replacement.md)。

## 1. 今回の固有の問い

Composite Functionの「Definitionが適合」「Providerが交換可能」「切替中もin-flightを保持」は、**進行中の複合Operation、状態付きFunction Instance、未保存Patch、新しい業務Rule、外部Effectに対して、そのまま継承できる保証ではない**。

改版を受ける主体が異なる。

1. 受理済みの**意味・Rule**の版（Source CandidateとAcceptanceを区別）。
2. Function Definition／Composition Bindingと、配置した**Function Instance**の版・状態。
3. Plugin Contract／Provider／Runtime Engine／Artifact／**Host-local Generation** の実体と選択。
4. Function Data ContextのResult Slice、選択epoch、取得Scope、未保存Dirty Patchとbase revision。
5. Operation Identity、対象、Payload、実Effect/Receipt、読み直したObservation、返却Resultの生成版。
6. Domain有効時点（例：組織の在任・有効日）と、実行/取得/Commit/表示時点。

単一の `currentVersion`、`sameRevision`、`sameType` では上記を表現しない。「過去に何が有効だったか」と「現在何を実行してよいか」も別命題である。

## 2. 版・時点の交差（新共通Clock APIではない）

| 観点 | 評価時に区別すべき値 | 決定責務／既存Owner |
| --- | --- | --- |
| Meaning revision | Accepted meaning R1/R2、Rule/Operator binding、受理Decision | #988/#1124／Domain Owner |
| Domain-effective time | 例：組織の異動発効日、契約適用開始、業務対象as-of | Domain Plugin/組織RuleのOwner（暦日と実処理日時を暗黙に同一視しない） |
| Data/observation time | Record/Snapshot revision、event time／read time、範囲・欠損 | #2506/#1708/#2582 |
| Composition/placement revision | Definition、Instance ID、port/connector binding、内部State version | #2507／各Domain |
| Runtime generation | Provider/Engine/Artifact exact版とAcquire lease、publish/drain | #1223/#951 |
| Request/effect time | Intent形成、Host認可、request dispatch、commit/ACK、独立read | #2598/#1174 |
| Presentation time | 過去Resultの現在表示、閲覧資格・開示・説明の再投影 | #2516/#2502/#1803 |

**注意**：Versionを束ねるread-onlyの「同一操作比較票」は既存Evidenceを参照するビューであり、もう一つのCanonical、全Domain共通Snapshot、唯一の実行時刻を設けない。

## 3. 状態移行を型変換でなく「条件付き意味写像」として評価

旧State空間 `S1` から新State空間 `S2` への移行候補 `M: S1 ⇀ S2` は**部分写像**でよい。少なくとも次を用途に応じて確認する。

- **Scope / identity**：どのInstance/Subject/Relation occurrenceの状態か。新Definitionでも旧Instanceの意味上の対象が残るか。
- **State invariants**：旧状態の有効条件と新状態の制約。失われる履歴・Observation・乱数/時刻/solver内部状態。
- **Behavior obligation**：同じ許容入力群・外部Contextに関する、観測者の必要Claim `Q` について、移行後のTraceが新契約で許容されるか。全Domainの厳密同値を自動主張しない。
- **Rollback / reversibility**：移行前Snapshotが保存されるか、移行が逆写像可能か、実TargetがRestore/Checkpointを許容するか。逆写像がない場合は旧版に戻しても旧Stateが復元できるとは言えない。
- **Side effects**：状態Snapshot/Simulation rollbackと、実DB/通知/物理作用の取消を混同しない。
- **Availability**：安全な切替点を認定できない、Snapshot取得不可、Domain意味が変わるなら、旧世代維持・ドレイン・再起動・明示再初期化等をTargetに応じて選択。安易に無停止を主張しない。

保守的な初期判定例：`semantics-preserved-under-Q`／`loss-disclosed`／`unknown`／`incompatible`。これは公開ステータス仕様の採用ではなく、検証票の表現である。Observer相対の情報保存（#2497）と独立Oracle（#2514）で具体化する。

## 4. 既存Generation KernelとComposite Stateの境界

現mainの [`internal/composition/coordinator.go`](../../internal/composition/coordinator.go) は、Host局所で新世代をPrepare/Readyし、切替後に旧in-flightを保持・DrainするReferenceを持つ。[`internal/composition/composition_test.go`](../../internal/composition/composition_test.go)には旧Lease保持中の早期Cleanup拒否等のテストがある。これらは**Function InstanceのDomain State移行、Dirty Patch migration、実Provider/DB Effect、跨Host一斉切替の保証ではない**。

後者は既存#2507/#2597/#1129/#1174の契約・実Target責務として必要なときだけ確認する。

更新の3つの異なる帰結：

- **Pin & Drain**：実行中Operationは旧Generationを使用、新規Operationは新Generation。意味やStateの互換条件を満たすとは限らない。
- **State Migration**：旧Instance状態を、Domainが認めた変換と安全な境界で新Instanceへ移す。移行できる範囲を選んだClaim/Targetに限定。
- **New Meaning Adoption**：受理済みR1からR2への移行。Domain Ownerの意味判断・発効条件が必要であり、Provider交換やコード生成だけで暗黙採用しない。

複数プロセス/Host/デバイスの同時切替をHost局所のCAS Cutoverから一般化しない。

## 5. 業務上の発効時点と進行中Operation

ユーザーの組織DSL方針では、組織異動・在任などの有効日ロジックは**Organization Domain Plugin**が持ち、DSLの文法/意味は標準Plugin側が提供する。Data物理層や共通Coreへ暦日判定を固定しない。

架空ケース：R1が`2026-11-02 23:59`まで、R2が`2026-11-03 00:00`から適用。OperationがR1の時点に開始され、CommitはR2発効後となる。

ここで業務上の処理基準が「申請時点」「対象の有効日」「Commit時点」「前提再評価時点」のいずれかはDomain Rule/Ownerが定める。**実行途中の旧Provider Pinと、採用すべき業務Ruleの選択は直交**する。

現在のHost認可/lease/fencingは永続作用の確定点で#1174の実Providerに従い、旧Ruleや古い資格の保持を許可の根拠としない。再評価が必要でも同じOperationを二重発行してよいとは限らず、Operation Identity/Idempotency/Effect Readを照合する。

## 6. 反証Fixture候補（テスト未実行）

| ID | 交差させる版・状態 | 独立に検出したい誤成功 | 主なOwner |
| --- | --- | --- | --- |
| VE-01 Read-only Cutover | v1のin-flight Leaseを保持しv2を公開 | 新規/旧呼出しの途中すり替え、旧Cleanup早期実行。既存Kernel試験と重複しないよう基準は#1223から再利用 | #1223 |
| VE-02 Stateful Instance | Cognition/集計BlockのState schema v1→v2、定義同一 | Stateのloss、初期値/seed/clockの変化をShape互換で隠す | #2507/#988 |
| VE-03 Partial State Map | 旧Stateの一部だけ新Stateへ対応 | 未移行Occurrenceを勝手にdefault/zeroへ置換し、元の履歴を消す | #2497/#2507 |
| VE-04 Dirty Session | v1 Functionで未保存Patch、v2へ差替え | base revision/親Anchor/選択epochを失い別Subjectへ誤更新 | #2597 |
| VE-05 Effective-date Boundary | 業務Rule発効時点と旧Provider lease・Commit時点が交差 | Provider版=Rule版とみなして誤った規則採用／旧GrantでEffect許可 | Domain Owner/#1174 |
| VE-06 ACK Unknown Replay | v1 CommitのACK不明後にv2配備、同一Operationの再試行 | Effectを二重化する、v2投影を過去の原BusinessResultと呼ぶ | #1174 |
| VE-07 Contract Shape Same | Provider v2でNULL/丸め/イベント順を変更 | バイト/Shape互換から下流Claimの互換性を推定 | #951/#1121 |
| VE-08 Mixed-Generation | A=v1、B=v2が同じDBや共有Stateへ操作 | Host局所切替がグローバルSnapshot/分散Txのように扱われる | #1223/#1708/#1174 |
| VE-09 Physics Tick | Engineのtick n直後にCognition Stateの版変更 | tick n+1の観測/運動が異世代Stateを黙って使用 | #988/#2507／外部Game Target |
| VE-10 Continuous Solver | early-return中・Solver checkpoint不可、version変更 | 内部Solver State不明なのにHot swap後を同一軌道と認定 | #988/#1121／外部Solver |
| VE-11 Static Target | 静的生成Wasm/Java/.NET Libraryの改版 | 新artifactを生成しただけで実Hostの稼働中置換が成立したと主張 | #1121/#259 |
| VE-12 History/Disclosure | v1のReceipt、v2のResult Projection、現在閲覧資格 | 古いEffect receipt→同じ画面値/開示資格を自動継承 | #1174/#2502/#2516 |

VRクマやFMI/Godotは**異分野Pressureの例**。実Game/Physics実装の現在仕様、成功、固定tick能力を本書から断定しない。

## 7. 実証順序（新Gateを設けない）

**E0 静的read-only票**：#1223 Kernelの既存Lease/Generation/Drain、#2507 Function Definition/Instance、#2597 Editing Session、#1174 Operation/Receipt、#2582 Claim invalidationを一枚の`Version/Instance/Assumption/Actual Evidence`表へ対応。既存テストの実行報告と本文の新試験を混同しない。

**E1 純粋合成**：VE-02〜05/07の小さな2-version、2-instance、2-Ruleケースを独立Expectedで検証。PythonまたはGo局所fixtureはRunner/期待値の実byteと試験結果を結び、今あるGo Kernelの振る舞いを仮定で補わない。合成PASSだけで製品受理としない。

**E2 Runtime local**：既存Go Composition API/Router/Coordinatorに対するread-only引渡しを優先。実呼出し中のv1-v2世代Pin、State migration候補の可否、Drain失敗を比較。#1223 Ownerの既存Reference/試験へ追補するときは別PR/担当確認。

**E3 Effect/Target**：TEI正式Function→Action→実Host/Provider→Effect→独立Observationの連鎖でVE-06/08を実証。既存#2598/#1174/#1121/#1119の本番資格条件を緩和しない。

**E4 異種Consumer**：業務/組織 effective-rule／ゲーム cognition-physics／連続Solver／外部Host Libraryを比較し、どこに安全な移行点を置けるかTarget別に説明。未実施TargetはNOT RUNのまま保持。

**E5 UX/文書**：Modeler/SHO/CUIで「作業版の変更」「意味受理」「稼働中Provider更新」「古い編集Session」「履歴の説明」「現在権限」を別の言葉で表示。GUIはHeadlessの同版Evidence成立後に検証。

## 8. Owner判断待ちと定量評価

1. **業務Ruleの実行時点**：発効時点とOperation開始/CommitのどちらがBusiness Meaningに適切か。案件/Domainにより異なる。
2. **State移行手法**：Version跨ぎでState/Dirtyを残す、編集を保留して再レビュー、旧Instance保持、強制再初期化など、対象Consumer/Targetごとの選択。
3. **意味変更の再承認**：Script/Macro公開者や組織固有Workflowの受理。TEIは特定の承認順序を標準業務意味として固定しない。
4. **可用性要件**：無停止を必須とするHostか、再起動を許す静的Targetか。実Targetの構成・技術制約に従う。
5. **Loss許容**：Q相対のState/Observation保存、履歴開示、精度許容。推測でAcceptedにしない。

測定：正常切替比率（不正な成功率を除外）、旧Lease未完了数とDrain時間、必要State mapping件数/損失箇所、Dirty保存/衝突件数、実副作用重複件数、Targetの可用性/P99、必要手作業レビューとAdapterコード量。#2018と#2493へフィードバックする。

## 9. 状態判定

- **既存として確認**：ADR-0032の目標アーキテクチャ採用、現mainのHost局所Composition Reference（`coordinator.go`等）、#2597の隔離した合成Fixture報告、既存#1174の実Provider／Receipt境界設計。
- **本書が追加**：跨Version State/Dirty/Domain effective-timeとin-flight/Effectを一度に扱う、VE-01〜12の対照とE0〜E5受渡し票。
- **未実証**：TEI製品でのFunction State Migration、Domain Rule実発効切替、跨Host/Targetの無停止更新、実DB/認可/Receiptとの同版E2E、Provider差替えのObservation同値、実Game/Simulation移行。Owner受理・実装資格は未取得。

受理済みReferenceの範囲を縮小せず、提案を既存合成Kernelの製品適用完了へ昇格させない。
