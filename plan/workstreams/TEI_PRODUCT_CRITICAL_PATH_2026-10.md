# 三視点ローコード実証のクリティカルパスと受入シナリオ

- 状態: **設計・実行計画候補**。この文書の作成はコード実装・実行・製品品質の成立を意味しない。
- 親Issue: [#1119](https://github.com/hat47x/tei-platform/issues/1119)
- 基点: 2026-10-10時点の既存Issue/PR・Repository文書による調査。進行中PRや実行結果は着手時に再取得する。
- 関係: [Product triage #982](https://github.com/hat47x/tei-platform/issues/982)、[三視点業務設計の既存方針](../../docs/business-system-design.md)、[現在の実装状況](../../docs/status.md)、[Requirement gap audit](../evidence/PRODUCT_REQUIREMENT_GAP_AUDIT.md)。

## 1. この計画が決めること

TEIで最も重要な製品上の検証を、「設計要素を並べられる」ことではなく、**一つの受理済み業務意味を業務・データ・機能から理解し、不整合を見つけ、正本を変更し、同じ実行契約を使って確かめられること**と定める。

対象は `order_entry` の既存の受注・明細Referenceから始める。既存のApplication Pattern、Driver、最初のWeb Targetを再実装しない。既存Gate A/B/CのReady判定も遡及的に変更しない。

この計画における「クリティカルパス」は工期の実測から求めた最長経路ではない。**下流の複数担当を同時に停止させ得る共通契約の依存経路**である。工期や実績が未取得の段階で工数や完了日を捏造しない。

### 成立させたい最短の製品体験

```text
受理済みの業務意味（正本）
  ├── データ視点：受注・明細・参照先・識別・制約
  ├── 機能視点：一覧・単票・明細・登録・確定
  └── 業務視点：担当者・契機・手作業・例外
                ↓ 同一のtyped reference / source trace
        三視点から不備を発見（read-only）
                ↓ 明示的な変更案
     Preview / Validation / Impact / Acceptance
                ↓ authoritative sourceへの変更
     再ロード・再投影・Driver / Webの実行
                ↓
     実行結果と別のObservationを検証
```

- `read-only` 投影は受理済みSourceを変更しない。
- 図の配置、スライド順、表示ラベルは業務意味そのものではない。
- 生成AI、旧資産の解析結果、推測から作った関係は受理前に正本へ昇格しない。
- Driverの成功応答は、更新後の外部状態を観測した証拠ではない。

## 2. 既存能力の評価と未成立の製品契約

| 境界 | 既存の足場 | 今回の未成立点 | 主担当 |
| --- | --- | --- | --- |
| 意味識別・Source | Canonical ID、Source Index、差分/影響解析のReference | 三視点に共通するtyped edgeと、その採否・変化の意味を小契約に限定して実証する | [#1120](https://github.com/hat47x/tei-platform/issues/1120) |
| 業務モデル | Business Procedure / Handoffの設計とReference | 一階層の業務図でActor・trigger・manual/external・Functionを編集・再確認する | [#1125](https://github.com/hat47x/tei-platform/issues/1125) |
| データモデル | Semantic Data/Relation、Schema/Migration、Data Design Context | 論理ER図、定義表、業務識別・関数従属性・物理Mappingを矛盾なく往復 | [#1126](https://github.com/hat47x/tei-platform/issues/1126) |
| 機能モデル | 5種のApplication Pattern、第一Web Targetの実行Evidence | Data-firstから標準Function/保守画面を直ちに試し、案件差分に耐える | [#1127](https://github.com/hat47x/tei-platform/issues/1127) |
| 三視点の人間向け編集 | read-onlyなCross-view Source観測、Workbench等の候補 | 同一参照先のnavigate、編集提案、Source Acceptance、再投影が一続きになる | [#1124](https://github.com/hat47x/tei-platform/issues/1124) |
| 横断検査 | Source観測、部分的なValidation / Impact | Accepted World内のFunction/Data/Procedure関係とSubsystem CUDの実用的な診断 | [#1132](https://github.com/hat47x/tei-platform/issues/1132) |
| 意味→実行 | Driver、Capability Binding、Target/Plugin/Build各契約 | 段階ごとの入力/出力/evidenceとfail-closed境界を明確に連結 | [#1121](https://github.com/hat47x/tei-platform/issues/1121) |
| 業務効果 | Command/Validation/Observation、状態遷移の検討 | pure derivedと更新前後・commit後の副作用の分離、取消/競合の安全性 | [#1129](https://github.com/hat47x/tei-platform/issues/1129) |
| 具体例 | Golden/Reference、旧Instance/Slide調査 | 型付きCase、Snapshot、説明順、Scenario、Expectationの実行接続 | [#1131](https://github.com/hat47x/tei-platform/issues/1131) |
| 非画面機能 | Capability、既存Runtime、元sourceのPanel/Spool/Batch観測 | 帳票1件・非画面Batch1件の実行/Artifact/取消Evidence | [#1130](https://github.com/hat47x/tei-platform/issues/1130) |

「足場」があることは、この表に記した製品動作の完了判定ではない。PRがDraftの場合は、試作コードが存在してもcurrent mainでの実装済みEvidenceと区別する。

## 3. 変更される可能性が高い最小共有契約

### C0 — Subject / Reference / Evidence（#1120）

Consumer: #1124、#1125、#1126、#1132、#1131。最小条件:

- Subjectは `kind + logical identity + scope` により参照可能。正確な直列化方式・型階層は実装担当が既存Semantic境界と照合して決める。
- Source locator、display name、physical PK、business occurrence identifier、runtime object IDをlogical identityと混同しない。
- `source-backed` / `derived` / `proposed` / `accepted` / `unresolved` の権限差を保つ。すべてを同列の単一「事実状態」として強制する要求ではない。
- Canonical IDの変更を根拠なしに単なるrenameとして確定しない。表示ラベル変更と論理識別子変更は異なる。
- 削除やsplitに追随できない参照は診断にし、存在しない参照先を暗黙に作らない。

**非Blocking**: 全Model kind / 全関係種別 / 全期間のidentityをCoreに固定すること。

### C1 — Read-only Projection / Proposed Edit（#1124）

Consumer: 三視点の全Editor。最小条件:

- 導出Projectionの行・図形を直接正本へ書き込まない。編集要求は、編集可能なSource subjectを示すChange Proposalに変換する。
- Proposalの基準revision、対象subject、期待元状態、変更field、由来、検証結果、未解決事項を追跡する。
- Preview / Source Validation / Impactを通し、明示的なAcceptanceのみが正本を変更する。
- concurrent edit / stale revision / unauthorized editは安全側に失敗し、旧Sourceが保たれる。
- 画面上の「変更済み」という表示だけをAcceptance済みとみなさない。

**非Blocking**: Wails、全図編集機能、全画面の完成。CLI/JSONを使った最初の縦切りでもよい。

### C2 — Cross-view Review（#1132）

Consumer: #1124、#1125、#1126、#1127。最小条件:

- 原Sourceに明記されたFunction-level CRUDと、Business Actionが**実際に行った更新**を分離する。
- manual/external taskがFunctionへ対応しなくても許容する。
- Subsystem/ownershipの解釈が未確定ならWarning/Questionに留め、強制的にinvalidと決めない。
- source-backed edgeと導出edgeに由来・構成規則・coverageを添える。

**非Blocking**: 既存資産の全転写、静的解析による実行時副作用の完全推測。

### C3 — Lowering / Capability / Execution Authority（#1121）

Consumer: #1127、#1129、#1130、#259。最小条件:

- 同じAccepted meaningを異なるTarget/Rendererへ渡しても権限・入力・Validation・Outcomeの意味は変わらない。
- 生成機能の存在は、実際のDatabase/Providerへのアクセス権限の付与を意味しない。
- 不足Capability、Target非対応、Binding衝突、認可不足を明示的に失敗またはunresolvedとする。
- Business Result、Execution Error、後続のObservation、Physical Artifactを分ける。

**非Blocking**: 汎用プラグインセルフビルドの完了、第二Targetの全面実装、すべてを事前バイナリ生成すること。

### C4 — Dataset / Scenario / Observation（#1131）

Consumer: #1126、#1129、#1132、#2144（期間）。最小条件:

- Datasetは型・参照・Revisionへ結び付く。Snapshotは具体的な状態であり、Model定義そのものではない。
- Stage / Presentation order / 実際のAction execution orderを混同しない。
- 未観測の手作業・外部効果を成功とせず、Scenario fixtureの合成結果をactual Provider Evidenceに格上げしない。
- 期間（period）は状態と時点の特殊形としてC4で扱う。開始・終了・締め・再開・訂正をDataset / Scenarioの状態と時点へ接続し、締め後の訂正を未承認の成功にしない。基盤の`persist`へ期間列を先に固定せず、実装は#2144、採用判断は#2484 D-2で扱う。

**非Blocking**: 大規模な業務シミュレーター、旧Instance/Slideの完全互換。

## 4. 共通最小受入シナリオ（P0）

既存 `order_entry` のSourceを変更することから始めず、**read-only観測から始める**。実Sourceへ変更を適用するフェーズは安全な合成fixtureまたは独立Workspaceで行う。

| ID | Given | When | Then / Evidence | 担当 |
| --- | --- | --- | --- | --- |
| V00 | 既存の受注、明細、得意先Relationと5つのPattern | 3視点のread-only Projectionを作る | 同じlogical subjectへ戻れる。存在しないBusiness意味を補完しない | #1120, #1124 |
| V01 | 受注明細Relation、Function/Data参照 | ER図/定義表でRelationを選択 | Source anchor、参照するFunction、影響するProcedureを一覧できる。推測辺は区別する | #1126, #1132 |
| V02 | Business Procedureに担当者の確認作業がある | そのmanual stepを閲覧 | Application Functionが無くても欠落エラーにならない | #1125 |
| V03 | Relationのcardinality変更案を保持 | Editorが変更をPreview | 影響箇所とValidationを表示。正本のrevision/digestは変わらない | #1124, #1120 |
| V04 | ユーザーが有効な変更案を明示的に採用 | Acceptance→再ロード | authoritative sourceの差分が限定され、各Projectionが同じ意味に再同期する | #1124 |
| V05 | Data+Relation+必要なProvider/Authorityがある | 標準CRUD/関連保守UIを生成し呼ぶ | Driver既存境界でValidation/Auth/Outcome/Observationを分けて処理する | #1127, #1121 |
| V06 | FunctionにCUD宣言とSubsystem ownerがある | 所有外Dataへの更新を検査 | 由来付きの診断またはReview Questionを出し、定義のみで実効果を断定しない | #1132 |
| V07 | 同じAccepted業務定義がある | 標準/案件UIを差し替える | 業務識別子、Validation、認可、Outcomeの意味が同一 | #1127, #997 |
| V08 | 実行されたCommandが成功応答を返す | separate Observationを取得する | Business Resultの成功とObservationの結果を区別して表示する | #1121, #1127 |

V04/V05は実際に動くContractが揃って初めて達成とする。UIのモックとDesign-only文書でP0 Gateを完了にしない。

### 必須の否定試験

| ID | 入力・状況 | 期待動作 | 担当 |
| --- | --- | --- | --- |
| N01 | 参照先の存在しないRelation / duplicate logical ID | source-aware diagnostic。暗黙の成功は禁止 | #1120, #1126 |
| N02 | Logical ID変更を表示名変更と同一視 | semantic changeまたはunresolved。根拠のないrename推測は禁止 | #1120 |
| N03 | 物理PK / UI row keyから業務識別を自動確定 | proposalまたはunresolvedに留める | #1126 |
| N04 | Functionの潜在CUDをBusiness Actionのactual effectに置換 | 異なるauthorityとして保持し、強い結論を拒否 | #1132 |
| N05 | Derivedセル/図形を直接編集 | readonly拒否、または正本の参照先を示したProposalへ案内 | #1124 |
| N06 | Preview後にSourceが他の主体に変更 | stale conflict。黙って上書きしない | #1124 |
| N07 | 業務手順のmanual stepに対応Functionが無い | valid manual/externalとして残す | #1125 |
| N08 | 必要なCapability/権限/Target bindingがない | fail-closed。生成されたUIから強制実行できない | #1121, #1127 |
| N09 | スライド表示順の変更 | Dataset/Actionの実時系列が変化しない | #1131 |
| N10 | Commit前の更新とCommit後の通知 | Rollback時に通知を済ませたと報告しない | #1129 |
| N11 | 結果応答は成功だがObservation未取得 | 外部状態をVerifiedと扱わない | #1121 |
| N12 | ER図の座標変更、フィールド表示の切替え | Accepted Semantic diffがない | #1124, #1126 |

N10は既存Runtimeで同等試験が作れない場合、`unresolved`としてGateを保留する。試験不能をPASSに置き換えない。

## 5. 並列レーンと境界の優先順

| Lane | Owner | 最初の確定事項 | 他Laneの進行を止めない範囲 |
| --- | --- | --- | --- |
| A — 意味参照 | #1120 | V00/N01/N02のtyped ID/edge、Source link | 全拡張Kindの確定は後続 |
| B — Business/Dataの読み取り | #1125, #1126 | read-only投影と元Sourceの対応 | GUI双方向編集完成前に進行可 |
| C — 実行 | #1121, #1127 | 既存Driver/Web経由のV05/V08 | self-build/第二Target完成不要 |
| D — 横断診断 | #1132 | V06/N04/N07のdiagnostic | 旧Source側#1031の全機能完了不要 |
| E — Studio連携 | #1124 | V03/V04/N05/N06のProposal/Acceptance | 最初はCLI/JSON consumerで可 |
| F — 次段Pressure | #1129, #1130, #1131 | 更新効果/帳票/Scenarioの一例 | P0最小の業務一往復をブロックしない |

### 依存として「待つもの」と「待たないもの」

- **Hard**: #1124のSource書込みには既存のShared Acceptance境界（#267）とsubject参照の最小契約が必要。#1127の実書込みにはDriverのAuthorityと明示Provider接続が必要。
- **Soft**: #1124のview skeletonは#1126の全ER表記実装を待たない。#1126のread-only ER図は#1124の編集UIを待たない。#1132の静的診断は#1129の副作用Runtime完成を待たない。
- **Evidence-dependent**: 旧Runtimeの実効果は[#940](https://github.com/hat47x/tei-platform/issues/940)の実操作Evidenceが必要。未観測のScript結果を断定しない。
- **外部PRとの分離**: [PR #1128](https://github.com/hat47x/tei-platform/pull/1128)は具体例→ER図スライドの合成・read-only試作。#1131はDataset/ScenarioとDriver実行の照合、#1126は論理ER/定義編集の契約を担当する。同じデータモデルや描画器を重複実装せず、PRの採用判断を待って既存素材を再利用する。

## 6. Gate と必要証跡

| Gate | Exit evidence | 不足時の状態 |
| --- | --- | --- |
| G0 — Inventory | 現在のowner Issue、producer/consumer、関連PR、具体的実装箇所と未成立条件を一覧 | `design-inventory-incomplete` |
| G1 — Minimal contracts | C0/C1/C2/C3の実行例と負例、version/Sourceの再参照。既存Contractとの衝突を説明 | `contract-unverified` |
| G2 — P0 product loop | V00〜V08のうち採用したP0必須セットとN01〜N12の該当負例を、明示的なR0→R1の受理系譜（Preview/失敗時はR0不変）と、実行に使ったAccepted revisionに結び付けて再現 | `product-loop-unverified` |
| G3 — Cross-target / mutation pressure | 第二Target差、仮想状態/取消、副作用、帳票・Batch・Scenario等を能力ごとに独立検証 | `capability-partial` |
| G4 — Release qualification | exact headとclean pre/post状態のtrusted full checkout、focused tests、Repository Health、actual Target Evidence、独立review | `qualification-pending` |

既存Gate A/B/CのReady記録は歴史上の固有Evidenceであり、上の新しい製品上の検証に自動的に読み替えない。また、本表のG0〜G4は#1119の**作業管理用Gate**であり、現在の正式な製品Gateを改名・置換するものではない。

## 7. 担当チャットの同期・引継ぎ

各チャットは着手時に次を明記して#1119へ報告する。

1. Current mainと自分のPR head、他のopen PRの変更path。担当が重なるときは既存branch/PRの所有者を先に確認する。
2. 新しい契約を導入するならproducer/consumer（Issue番号、API/Schemaの位置）、破壊的変更と負例。
3. 既存契約を借りるだけならそのSource path、必要な不足点だけを提示。
4. `tested` / `not-run` / `needs-trusted-checkout` / `actual-target-observed` を明確に区別し、`not-run` の範囲を制約として記録して統合する。Draftは解除して統合する。
5. 各milestoneが資格確認されたらmainに統合し作業branchを削除する。他チャットのbranchを無断で書き換えない。

初回の全レーン一斉実装は行わず、**V00→V03→V04→V05→V08** の縦断を最初の到達目標とする。その間に#1125/#1126/#1132のread-only診断を並行して準備し、製品の実働ループが成立してから必要に応じて#1129/#1130/#1131を深める。

## 8. 現段階での非主張

- 現時点ではG0〜G4のどれにもこの文書だけで合格したとは主張しない。
- GitHub PRのDiff/Issue本文を読むことは、信頼できるfull checkoutで実行して検証したことではない。
- X-TEA由来の機能の「観測」「移植候補」「TEIでの実装完了」は異なる。
- 関連元モデルの著作物や原データを、公開可能性の確認なしにfixtureへ複製しない。

## 9. G0 横断依存台帳（2026-10-10・暫定）

### 監査の基点と証拠の区分

- GitHubの `main` と本PRの比較基点: `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4`。本PR #1135は `work/issue-1119-critical-path-product-gate` からの**文書だけのDraft**。次回の着手時にはこのSHAを流用せず、改めてmainを取得する。
- 現状区分は `baseline-reported`（既存のstatus / Gateに記録）、`source-located`（今回main上のpathを読んだ）、`draft-candidate`（未統合）、`not-observed`（今回actual実行していない）を併記する。「Referenceあり」と「このGateのPASS」は別。
- `docs/status.md`、`plan/evidence/PRODUCT_REQUIREMENT_GAP_AUDIT.md`、`docs/business-system-design.md`を既存の製品判断の基線として参照した。今回確認したsource anchorは `internal/ir/world.go`、`internal/plugin/model.go`、`tools/reference-shared-acceptance/README.md`、`tools/repository-health-check.sh`。pathがあることとAPIの適合・actual PASSは等価ではない。
- 既存Gate B/Cの過去のReady判定は変更しない。今回のP0 product loopについては全ケース**未資格確認**であり、以下の実装場所・テストは各ownerの申告待ち。

| 契約・CP | Producer / 最初のconsumer | 再利用する既存境界と現状 | 下流を止める最小blocker | Evidence / Ownerが提出するもの |
| --- | --- | --- | --- | --- |
| CP-1 / C0: typed subject・edge・provenance | #1120 → #1124/#1126/#1132（後続 #1125/#1131） | Canonical/Source Indexと意味差分。既存基線あり、今回の三視点契約は未確認 | 論理ID、Source locator、表示名、physical keyを混同せず、無効edgeとrenameを検出できること | V00/V01、N01/N02の正負fixture・JSON差分・破壊的変更規則 |
| CP-2: Capability/Plugin選択・依存 | #951/#1016/#1019 → #1121/#1127 | `internal/plugin/model.go`に固定CapabilityKindあり。#951のgeneric resolverはReference/Draftのためproduction採用を未認定 | exact contract / scope / version / provider・Conformance / Authorityの欠落をfail-closedにすること | 合致・不一致2 Provider、wrong scope・ambiguous provider・unauthorized higher versionの結果 |
| CP-3 / C3: lowering・execution handoff | #1121 → #1127/#1129/#1130 | DriverとWeb Targetは既存Gateの基線。新しい共通巨大IRは不要 | Accepted revision → Binding/Lock → Driver/Targetのtrace、required Context Slot宣言を落とさないこと | V05/V08、N08/N11。Hostの実Principal/sessionをCanonicalへ入れない負例 |
| CP-4: Target/Authority束縛 | #976/#1043/#259・#1118 → #1121/#1127 | 実行時のPermission/Host/Target既存境界を流用。第二Targetによる意味同等性は未実証 | declarative permissionとcurrent execution authorization、potential CUDとactual effectを区別 | Authority拒否・Target Unsupported・別read Observationを別証跡として添付 |
| CP-5: Build / physical Artifact | #943/#979/#1080/#1101/#1103 → #1121、後続G3/G4 | Build/Plugin Lockは既存の別責務。self-build Stage3はP0のHard blockerにしない | exact artifact/target profileの追跡は保つが、Runtime Lockへbuild-only依存を入れない | Stage別の再現・digest照合はG3/G4で評価。P0では既存実行物を用いる |
| CP-6 / C1: Projection→Accepted Source | #1124 + 既存#267 → #1125/#1126のview consumer | `tools/reference-shared-acceptance/`はfile/Studio/AIの同一受理境界を持つReference。人向け三視点UIのactual運用は未確認 | proposalのbase revision、対象Source、preview非書換え、stale拒否、accept後再投影 | V03/V04、N05/N06/N12のbefore/after source digest |
| CP-6 / C2: Cross-view diagnostic | #1132 → #1124/#1125/#1126/#1127 | #1031はlegacy source read-only観測でAccepted Action effectではない | Function CUDのdeclareとOwner Actionのactual effectを別の関係型にする | V06、N04/N07のsource-backed / derived / unresolved診断 |
| CP-6 / C4: typed example / Scenario | #1131 → #1126/#1129/#1132 | PR #1128はread-only ER slide試作。Dataset→実行→差分照合は別 | Stage、presentation order、actual step timeを混同しない | N09、2例のexpected/actual差分。P1寄りでP0初回loopを停止させない |
| CP-6: Business DFD・Data ER・standard UI | #1125/#1126/#1127 → #1124/#1132 | それぞれPlugin Projection・既存Pattern/Driverに接続 | 手作業の欠落扱い禁止、logical/physical key分離、UI由来の認可昇格禁止 | V01/V02/V05/V07、N03/N07/N08/N12をそれぞれ提出 |
| CP-6: Consequence / report / batch | #1129/#1130 → P1製品圧力 | 既存Command/Observationを再利用、actual consequenceは未認定 | rollback、二重effect、cancel、artifact outputを静的なFunction宣言から推定しない | N10および帳票・Batch・再実行の各actual証跡 |

**評価の優先度**: P0の先行安定化はC0（参照）・C1（受理境界）・C3（実行Authority）。これらは下流consumerが多く、片方の都合で変えると三視点とDriverに同時手戻りが生じる。C2はC0に依存する薄いRule側で先行可能、C4とself-buildはP0最短loopの阻害条件にしない。変更頻度・影響先件数の実測が未取得なので、定量スコアや完了期日は付与しない。

### G0からG1へ渡す契約ハンドシェイク

各Producerは実装を他レーンへ配布する前に、以下の**最小互換packet**を#1119へ返す。packetは新たなCore型を強制するものではなく、既存APIを採用する場合はそのpathとversionで足りる。

1. `owner issue / provider path / consumer issue / exact revision / contract version / schema or sample`。採用済みSource、Derived projection、候補、未解決、実観測のAuthorityを個別に記す。
2. 正例1件・負例1件以上の実行方法と入力digest、期待されるdiagnostic code、出力のSource anchor。少なくとも旧consumerの互換fixtureを含める。
3. **破壊変更の分類**: read-only field追加、semantic kind/edge追加、existing fieldの意味変更、ID再割当、serialization変更、required capability/contextの追加を区別。重大な変更はproducerがconsumerを列挙し、consumer側がversionと失敗時動作を確認するまで共有契約を凍結しない。
4. `tested | not-run | needs-trusted-checkout | actual-target-observed`と、実施時のmain/head SHA・Go version・pre/post clean状態を明記。GitHubに文書やテストソースがあるだけではPASSにしない。
5. C0→C1では「subject/source locator」と「acceptの対象/expected revision」の対応を、C0→C2では「source-backed/derived」と「Action-specific effectではないこと」を、C3→#1127では「required invocation context/Host authority」が落ちないことを**consumer側の負例**で確認する。

**具体的な契約衝突の停止条件**:

- C0の新しいedge typeがSource Adapterの観測factをAccepted meaningへ暗黙昇格させる場合、#1120と#1031/#1132でreviewするまで採用しない。
- C1のUI独自accept経路が `tools/reference-shared-acceptance` のstale/current Source検査を迂回する場合、#1124/#267で止める。
- C3のArtifact/Lockにcurrent Principal/credentialを固定したり、required Context Slotをoptionalとして欠落させたりする場合、#1121/#1118で止める。
- C2の「Subsystem外のCUD」を静的なFunction-level declared operationだけで実更新/認可違反と断定する場合、#1132へ差し戻す。
- P0のRuntime選択をself-build Stage3 / 全Plugin generic化 / 第二Targetの完成まで延期する場合、#1119でHard blocker指定の根拠を見直す。

### PR競合、推奨統合順序、Gate判定

- **今回確認した変更path**: PR #1135は本計画ファイルのみ。PR #1128（調査時head `62f0c98b3f230b11ba09716747ec5e5b4a4dc053`）は `plan/design/EXAMPLE_ER_SLIDE_M1_2026-10.md` と `tools/reference-example-slide/` の10ファイル、PR #1136（head `ec22e36ed3207de6884ca9e0387396f57de8dd3e`）は `tools/reference-subsystem-cud/` の5ファイル。現時点で#1135との直接path重複なし。#1128はDraftであり、#1126/#1131がその描画器を先取り・複製しない。
- **境界の近い別PR**: #1040はShared Acceptance周辺、#1115はAcceptance source safety、#1114はWeb Host/Auth、#1123はInvocation Contextであり、#1124/#1121/#1127が同じ契約やファイルを変更する前に変更pathとmain採用状況を再照合する。この列挙はpath一致を確認した一覧ではなく衝突候補。
- **統合wave 0**: #1119の計画候補を独立review・文書検査して資格確認。ここでG0の全Gate合格を宣言しない。
- **wave 1（並列契約確認）**: #1120はC0最小fixture、#1121は既存Capability/Driver上のC3最小fixtureを別PRで提出。同じ`internal/ir/world.go`や`internal/plugin/model.go`を両者が同時編集しない。必要ならC0の薄いproducer patchを先に採用し、C3はそのmain revisionへ追随。
- **wave 2（独立consumer）**: #1125/#1126のread-only Projection、#1132のsource-aware cross-view diagnostic、#1124のPreview/Stale契約をそれぞれ小さく試験。Coreへ新Domain enumを足す前にExtension/Projection適合性を確認。
- **wave 3（P0製品一往復）**: #1124がC0/C1を使ってV03→V04、#1127がC3と既存Driverを使ってV05→V08へ連結する。Producerが互いに無関係なfixture revisionを検証しただけでは合格としない。V03のR0→V04のR1をDecisionで結び、V05/V08のconsumerがR1または根拠付き等価revisionを使用したことを検査する。
- **wave 4（P1拡張）**: #1129/#1131/#1130、および第二Target・self-buildとの差分確認を個別に進める。新しいHard dependencyはP0の失敗例を提示してから採用する。
- 各waveの終わりに、対応ownerはPR head・main SHA・実際に走らせた試験・未実行理由・次consumerをIssueへ記録する。資格確認済みmilestoneのみmainへmergeし、不要な作業branchを削除する。他レーンのPRを横断担当が無断統合しない。
- **現在のGate状態**: G0 = inventory candidate（具体的API/API testのowner照合待ち）、G1 = contract-unverified、G2 = product-loop-unverified、G3 = capability-partial/actual未確認、G4 = qualification-pending。今回の作業環境にtrustedなRepository full checkoutは無く、`tools/repository-health-check.sh`・actual Targetの実行は**not-run**。GitHub上の差分確認のみでG0/G1をPASSとしない。

## 10. G0証跡受付の最小フォーマット

#1119への各laneの次回報告は、長文の進捗説明より次の1行単位のcoverageを優先する。

```text
case=V03/N06; owner=#1124; producer=C1(#1124); consumer=#1126;
main=<sha>; head=<sha>; source=<path:anchor>; test=<command/test>;
fixture=<digest>; outcome=PASS|FAIL|NOT_RUN|UNRESOLVED;
evidence=source-located|synthetic-tested|actual-target-observed|not-observed;
blocked_by=<issue/pr/revision or none>; next=<consumer action>
```

すべてのV00〜V08/N01〜N12は**設計上の担当あり・このプログラムとしての実行証跡なし**から始める。Issueごとの実証結果を受領した時点で該当caseのみ更新し、別RevisionでのPASSを横滑りさせない。G2の採否判定には、Preview時R0、Accept後R1、生成・実行側のR1（または適合性の証跡）、Provider/Target identity、否定試験結果、別readによるObservationを束ねた受入packetが必要。

## 11. 最新PRを用いた契約の実地監査（2026-10-10）

この節はGitHub上のexact blob / changed paths / Issue申告を照合した**横断レビュー結果**であり、trusted checkoutの検証や採用決定ではない。新規PRの状態は着手時に取り直す。

### #1132 のC2候補: PR #1136

- PR head: `ec22e36ed3207de6884ca9e0387396f57de8dd3e`、base main: `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4`、Draft。変更先は `tools/reference-subsystem-cud/` の5ファイルであり、#1135/#1128や他の現時点で確認したPRの編集pathとは一致しない。
- 提供契約（**Candidate E0のみ**）: `subsystem-cud-candidate-v1`、read-onlyな `tei world` の `semantics[].data[].id` / `applications[].patterns[].id` に加えて、caller-supplied `owners[]`・`functionIO[]`・`procedureSteps[]` を入力にする。outputは `Diagnostic`（code/severity/subject/source）と明示的な `authorityLimit`。実TEI `reference/world.json` に同じJSON shapeが存在することはsource上で確認したが、実Worldをこのtoolでロードして一致を試験したわけではない。
- 所有者の**隔離環境での報告**: Go 1.23.2、8件のunit test PASS、`go vet ./...` PASS、合成CLI正例PASS。GitHub blob照合済みとIssue #1132へ報告。これは`tested-isolated`であり`tested-trusted-TEI-checkout`でも`actual-target-observed`でもない。
- **横断ケースに対応する既存テスト名（出典: PR #1136 のpatch）**:
  - V06 / foreign write review候補: `TestCrossSubsystemWriteRequiresReview`。synthetic factから `CROSS_SUBSYSTEM_WRITE_INTENT` を出す。
  - N07 / manual step保持候補: `TestOwnerSideDelegationAndCrossSubsystemReadsAllowed`（manual/external stepを含む正例）と `TestNoPatternInferenceOrManualFabrication`（誤ったFunction関係を拒否）。
  - N04 / actual effect非主張の一部: output `authorityLimit` があることを確認する `TestOwnerSideDelegationAndCrossSubsystemReadsAllowed`。ただしCUD宣言を実際のAction effectと分離する**Accepted Sourceを用いた結合試験は未実施**。
- **未取得のG0〜G2証跡**: Source Indexで検証された `source` locator、Accepted meaningへのtyped join、同一revisionに対するowner/Function/Procedureのcoverage分母、EditorへのSource navigation、実Runtime effectの観測、trusted full checkout。caller-supplied `source` をVerified Sourceへ自動昇格しない。
- **軽量レビューで見つかった改善候補（owner判断）**: 空の`facts`で診断0件となっても「診断対象の定義が足りている」とは言えない。G2では`source-backed count / candidate count / unjoined count / omitted-unknown count`等のcoverageを別に要求する。また `FunctionIO`が未知のPatternを参照した場合、現行 `Inspect` は`IO_PATTERN_UNRESOLVED`を出しながら、同じ入力に `CROSS_SUBSYSTEM_WRITE_INTENT`を追加し得る。前提不成立のとき依存診断を抑制するか、その「未解決前提からの候補」性を明示する負例を#1132に委譲する。
- **判断**: #1136の現行status `no-findings-in-declared-scope` を `G1 PASS` / `G2 PASS` や「権限違反なし」に変換しない。#1120のC0のtyped Source subjectが確定するまでは、C2固有ルールをCore Graphに昇格させない。

### PRごとの衝突・依存の粒度（GitHub変更pathの調査）

| Group | 対象PR / 主な変更先 | 共有契約上の注意とhandoff |
| --- | --- | --- |
| Source / Acceptance | #1115 `tools/reference-shared-acceptance/`、#1111 `internal/workspace/` | #1124のSource editは既存baseline alias防御、Loader bytes / Source Index同期を維持。両者の採用済みmain revisionを取り直す |
| Business / cross-view | #1040 `tools/reference-business-procedure-check/`、#1077 `internal/xeadsource/`、#1136 `tools/reference-subsystem-cud/` | Source/legacy/TEI Accepted WorldはAuthorityが異なる。#1132は新たなGraph正本やAction effectを作らない |
| ER / Example | #1128 `tools/reference-example-slide/`、#1136 `tools/reference-subsystem-cud/` | 現段階ではpath重複なし。Dataset示例と論理ER編集を別責務にする |
| Runtime / Invocation | #1114 `reference/web-composition/`・`tools/reference-first-web-target/`、#1123 `tools/reference-capability-invocation-context/`、#1116 `tools/reference-plugin-capability-contract/` | #1121/#1127はProvider resolution、current Host session、Business Resultを一つに圧縮しない。PR間の共通Interface変更時だけ明示的にmerge順を指定 |
| Semantic/Workspace Pressure | #1049 `reference/dsl-language-plugin/`、#1051 `reference/application-composition/`、#1112 `internal/knowledgecontext/` | #1120/#1124は新DSL/CompositionをCoreへ先行固定せず、必要な既存契約を読み取りで借りる |

ここでの「path重複なし」は**調査したPR headに対する直接ファイル名一致がないことだけ**を意味し、go testの統合互換、semantic interfaceの互換、rebase後の競合不在を示さない。全PRの相互patchを解決していない。各担当はmain追随後に再照合する。

### 第1回G0静的セルフチェック（Repository Healthの代替ではない）

#1135 head `b6d0b1f2eaa16b430a7d95a4bc4f3a9c7fddd106` の文書をGitHubから取得し、次を静的に確認した。

- V00〜V08、N01〜N12の全21 IDが表に各1回現れる（欠番・表内の重複なし）。
- Markdownのfence開始/終了は各2個で偶数。
- 本ファイルのrelative link 3件の解決先 `docs/business-system-design.md`、`docs/status.md`、`plan/evidence/PRODUCT_REQUIREMENT_GAP_AUDIT.md` は取得できた。

これらは`static-doc-check-only`。その他のリンク、Go build、plan guard、repository-health-check、actual executionは未実行。対象headが変更されたので、統合判定では本静的確認のheadと新headを混同しない。

### 決定すべき一つの「到達条件」

G2は単なる21ケースのチェック数ではなく、**一つの受注明細業務について、R0→R1のAcceptance系譜と、R1から派生したProjection・Action binding・Driver request・separate Observationの連結を追えるか**で判断する。最初のハッピーパスは `V00→V03→V04→V05→V08`、最少否定経路は `N05/N06`（Source）、`N08`（Capability/Authorization）、`N11`（Observation）とし、現時点の具体的な変更対象・実装可能なvalidation変更は#1124/#1127が既存Schemaと照合して確定させる。合成例に一回だけ一致したことは、実Targetの独立した成功観測ではない。

## 12. G1→G2接合部：設計の受理と実DBへの適用を分離する

### 問題：V04が成功してもV05は実行可能とは限らない

V04の「設計変更を正本へ受理」と、V05の「標準CRUDが保存できる」は別の責務である。ER上のRelation、cardinality、必須項目などを変更して正本を再投影できても、実DB、Provider binding、Target artifactがその意味を扱えるとは限らない。ここを一続きの成功として扱うと、**新しい設計意味を古い物理構造へ誤適用する**失敗が隠れる。

特にTEIではApplicationとDatabaseはN:Mであり、Database schemaにはApplicationと独立したライフサイクルがある。Application revisionとDB schema versionの番号一致は要求しない。必要なのは、利用する実体ごとの**互換性と権限を検査した根拠**である。

これは新しい全体Core型やmigration engineを要求するものではない。既存のSource Acceptance、Schema Desired/Actual/Diff/Migration、Capability/Plugin/Driver、Target/Artifact、Execution/Observationから一つの接合証跡を作る。

### 実行可否の判定を行う場所

1. **Source Acceptance**（#1124 / 既存#267）: proposalの基準revisionを検査し、受理済みSourceを作る。DB操作は行わない。PreviewとAcceptedを混ぜない。
2. **Realization assessment**（#1126 / 既存Schema・Migration）: 変更後のData/Relationから、選択したDatasource/Providerに必要な実現差分と互換性を判定する。「差分なし」「既存schemaで互換」「migration必要」「未対応・未確認」を分ける。暗黙のDDL実行は禁止。
3. **Invocation readiness**（#1121 / #1127）: 実行直前にAccepted meaning、生成Function/Action、Target profile、選ばれたDatasource/Providerの観測状態、Capability/Authorityが**互換**であることを判定する。build-time Lockとcurrent Principal/sessionは別。
4. **Actual command と independent observation**（既存Driver / #1127）: 権限付きCommandの結果と保存後の別readを別証跡にする。Command成功だけでphysical stateをVerifiedとしない。

段階ごとに別のrevisionやfingerprintを記録できること。Schemaのphysical migrationを必要としない画面文言変更まで書込停止にしてはならない。一方、危険なSchema差分を「自動生成可能だから」と無検査で流してはならない。判定時点と実Invocationの間にbinding/physical stateが変わった場合は、その情報が安全性条件に関わる限り再検証またはfail-closedとする。具体的な監視/排他方式は各Providerに委譲する。

### G2提出packet（提案、既存契約を優先）

次のキーは**監査用packetに必要な情報**の提案であり、同名のCore fieldを新設する指示ではない。既存Evidenceへのリンクで代用できる。

| 欄 | 意味・条件 |
| --- | --- |
| `accepted_source` | Source locator、Accepted meaning revision、digest。acceptedでないproposalは不可 |
| `change_class` | `presentation-only` / `semantic-no-physical-change` / `physical-change-required` / `unsupported-or-unresolved` |
| `realization` | Application/DBの各authorityとmapping、選択したDatasource、対象physical schemaの観測・差分・互換性根拠。schema versionの同値は不要 |
| `function_target` | 導出Function/Action、Capability binding、Target profileと実行対象artifact/Runtime revision（該当する場合） |
| `execution_guard` | invocation時のcurrent authorization、必要なHost Context、required capability、schema互換性の判定結果 |
| `outcome_observation` | Command result、独立したread Observation、対象record識別と時点。未観測をunknownとする |
| `evidence_status` | 各段階ごとに`NOT_RUN`/`PASS`/`FAIL`/`UNRESOLVED`、exact head/fixture/provider/toolchain条件 |

### 接合部の追加正・負試験

| ID | 前提・操作 | 期待 | 主担当 |
| --- | --- | --- | --- |
| S01 | 画面ラベルのみの変更をAccepted | Semantic revisionは追跡し、不要なDB migrationを要求しない | #1124/#1121 |
| S02 | Relation/cardinality変更をAcceptedし、旧physical schemaのまま更新 | 相互互換性の未確認ならCommand前に停止し、Source受理成功をDB更新成功と言わない | #1126/#1127 |
| S03 | 変更済みphysical schemaを別Datasourceへ切り替えて呼び出し | Datasourceごとのrealization/Authorityを再評価。別DBの観測結果を流用しない | #1121/#1127 |
| S04 | 安全な独立fixtureで必要な変更を計画・適用し、schemaを再観測 | 既存Migration安全境界の実行記録と、Function保存→別readを結び付ける | #1126/#1127 |
| S05 | 有効だったbinding/required capability/current authorizationが実行時に失われる | 旧判定・preview・生成artifactだけを根拠に実行せずfail-closed | #1121/#1127 |
| S06 | 同一DB schemaを複数Applicationが参照し、片方のみmeaning revision変更 | 各ApplicationとDatasourceのmapping互換性を独立に評価し、無関係なApplicationを自動改版しない | #1120/#1126 |

- S02/S05は初期P0の**必須負例**とする。S01/S03/S04/S06はG2の範囲内で実装難度を確認し、未取得なら未確認と記録する。G3以降へ送る場合も理由と担当を固定する。
- V03→V04の成立条件はS04の成功を前提としない。逆に、V05の「保存」はrealization/Authorityの確認なしにPASSとしない。
- 合成fixtureでのS04は合成Evidence。実ProviderでのS04と同一のclaimにしない。Target・DB・権限・独立readの実操作証跡は別途必要。

### 責任境界と次の受け渡し

- **#1120（C0）**: Accepted subjectとsource revisionの追跡を担う。DB schemaや物理record keyへのidentity同一視を要求しない。
- **#1124（C1）**: Preview→Acceptance→再投影まで。Accept時点でmigrationや新Target配備を暗黙に行わない。
- **#1126（Data realization）**: 既存Schema Diff/Migrationの再利用箇所、上記4分類の判定と未対応条件をowner packetに記す。
- **#1121（C3）**: lowering段階と実行直前でどのreadiness情報を伝え、何をその場で再検査するかをC3 packetで示す。current credentialsを不変artifactへ入れない。
- **#1127（Product consumer）**: S02/S05の実行前拒否とV05/V08のCommand+Observationを最小Web referenceで実証する。
- **#1119（Program）**: 上記packetのproducer/consumer確認と同一シナリオへの証跡の束ね直しを担当する。旧schema向けの旧Function実行結果を、変更後Sourceの実証へ横滑りさせない。

**現状**: 2026-10-10の設計上の漏れを検出し、連携契約と否定試験を追加した段階。S01〜S06のテスト実行、actual DB/Target操作、各担当のAPI適合確認、trusted full Repository Healthはいずれも**未実施**。この節の追加だけでG1/G2をPASSにしない。

## 13. 初期consumer・lowering候補の横断照合（2026-10-10）

### C1: Studio Candidate/Projection — #1124 / PR #1137

- 正確な比較時点: base main `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4`、head `3bede1f48a0e373e3f444b7e63dab324037dee07`、Draft。変更は `reference/studio-authoring/` の4ファイルのみ。#1135/#1136/#1141との直接path一致なし。
- Producer側の約束は `tei.reference.studio-edit/v0` の `add-relation` / `add-validation/positive` を既存 `tei.reference.change-candidate/v0` へ渡すこと。受理済み意味やSource改変、Validationを独自実装せず、`tools/reference-shared-acceptance`を再利用する。
- `projectWorld` は `Data / ER / Function` を元のCanonical IDで投影するが、Business viewは `unresolved`、Source Indexのanchorも `unresolved`。これがV00の**三視点完全達成ではない**ことを残す。
- PRに記載された**9件の合成Nodeテスト**は、テストコードが存在しても `node --test ...` actual実行済みではない。報告されているV8 smoke 9件PASSは別の限定的な合成証拠。
- **consumer handoff**: #1120はC0 exact source anchor/typed edge、#1125はBusiness Procedure/Actor/triggerのread-only投影、#1124はscratch Workspaceで `V03→V04→reload` と `N05/N06` をexact revisionで実際に走らせる。#1137のProposal JSONを単に生成できるだけでは `V04 PASS` にしない。

### C3: Lowering Witness — #1121 / PR #1141

- 比較時点: base main `5fc5a0e1cb4fb42897c76dd3334292e2ad4a66e4`、head `afe8a648f0c09d361d21f0f2433a3d13c4ce9fa4`、Draft。新規は `plan/design/CRITICAL_LOWERING_CONTRACT_DESIGN.md` および `tools/reference-critical-lowering/` の3ファイル、合計4ファイル。
- E0 `tei.lowering-witness/v0` は、synthetic Candidate→Acceptance link→neutral intent→wiring→Provider witnessを扱う**候補**。Trust callbackやProvider/Lock/Artifactの具体IDは合成であり、実#951 Resolver・Shared Acceptance・Driver・Targetへまだ連結していない。
- `requiredContextSlots` の**宣言**をAccepted ContractとProvider選択に残し、actual verified Principal/session値はcurrent Host invocationで扱う設計が維持されている。これを理由に `C3 G1 PASS` としない。
- **consumer handoff**: #1121は#951/#1016の実Resolverや#1118 Host Context Gateへ繋ぎ、#1127は既存Driver/Web Targetで `N08/N11` と `S02/S05` を実行する。synthetic `authorization-denied` が出たことをcurrent Runtime Authorizationの実証に読み替えない。

### 合流時の最小ルール

1. **C0（#1120）**のsource subjectはStudio（#1137）とCUD（#1136）で同じIDを指せても、前者は編集候補、後者はcaller-supplied factであってAuthorityが違う。typed ID一致だけで後者のowner/effectをAcceptedへ昇格しない。
2. **C1（#1137）とC3（#1141）**の間には、既存Shared Acceptanceで生成されたexact Accepted Source revisionが必要。合成Acceptance stubやbridge precheckは橋渡し証跡にしない。
3. **C3（#1141）とDriver/Web（#1127）**の間には実Resolver/Capability/Host Permission/DB readinessが必要。§12のS02/S05を通るまで、設計受理の成功はphysical write可能の根拠ではない。
4. 並列PRはそれぞれ独立のDraftとして保持し、Source joinが必要になった段階でowner/consumerの相互Reviewを実施する。プロダクトG2の共通packetはR0→R1の受理系譜、R1に整合するactual Provider/Targetと独立Observationを束ねて初めて評価する。

**未確認**: この§13はGitHub変更path・patch・PR本文の静的照合であり、今回の環境でNode/Goの試験を実行したわけではない。各PRが報告した限定的なsyntheticテストと、trusted exact-head checkoutでの資格確認を区別する。

## 14. G2のRevision系譜とEnd-to-end受入packet（候補）

### 訂正: 変更前と変更後に「同一Source revision」を要求しない

V03 PreviewとV04 Acceptanceが有効な**意味変更**であるなら、一般に変更前のAccepted Source revision `R0` と変更後のrevision `R1` は異なる。従って「V00〜V08を同一Source revisionで実行する」解釈は不適切。G2で求めるのは**変更前後の不可逆な同一視をせず、accepted DecisionによりR0からR1へ辿れ、後続のDerived/Runtime consumerがR1に対応すること**である。

ここで`R0`/`R1`は説明上の記号。Git commit SHAではなく、既存Shared Acceptanceの`revision -workspace`で得るmanaged-source `modelRevision`に相当する。Git head、Source Indexのversion、Derived Worldのdigest、Database schemaの観測ID、Runtime/Build artifact digestも互いに独立した値であり、単純な文字列同値を要求しない。各値のproducerを明記し、同一性/互換性を**適切な辺で検証**する。

```text
trusted scratch workspace + existing accepted source
    │ R0 = Shared Acceptance revision -workspace
    ▼
Studio / AI / file candidate P(baseModelRevision=R0)
    │ preview/review: R0 unchanged, fingerprint=FP
    ├── reject, stale, validation failure: stays R0
    ▼ explicit acceptance with expected-review-fingerprint FP
accepted Decision D (R0, P, FP, decision evidence, resulting R1)
    │ source reload: obtain R1 independently
    ├── R0 -> R1 trace must be explicit
    ▼
World/Projections W1 from accepted Source R1
    │ build/derivation dependency witness, not ID equality
    ▼
Function/Binding F1 for R1
    │ schema/Provider/Target compatibility and current Host Authority
    ▼
Invocation I1 (R1-compatible realization; exact target/binding)
    │ Command outcome O_cmd
    ▼
Independent read O_read (record/provider/time correlated with I1)
    │
    ▼
G2 acceptance packet: complete path or explicit UNRESOLVED
```

### Producer/consumerの最小必須証跡

| 接合辺 | 必要な比較 | 失敗時の扱い | Owner |
| --- | --- | --- | --- |
| `R0→P→Preview` | Shared Acceptanceのcurrent `modelRevision`、candidate `baseModelRevision`、meaning delta、preview前後のmanaged-source digest | stale/invalidなら**R0不変**、Preview成功をAccept済みにしない | #1124/#267 |
| `Preview→D→R1` | exact candidateとreview fingerprint、accept decision、実Sourceの再読込revision、許可されたsemantic diff | accept失敗ならR1不成立。Decision記録だけでSource変更を推測しない | #1124/#267 |
| `R1→World/Studio` | 新R1からの再投影、logical ID/typed refs、Source Indexのcurrent anchor、未解決Business view | 旧Worldの再利用、Source anchorの捏造を拒否する | #1120/#1124/#1125/#1126 |
| `R1→Function/Lowering` | Accepted meaning→Function/Capabilityのproducer、依存fingerprint・Target profile・versioned witness。cache hitは意味上の同値性で説明 | 合成Trustや無根拠なR0のartifactをR1実行の証拠にしない | #1121/#1127 |
| `Function→Provider/DB` | 対象Datasourceごとのschema互換性、binding、current Host Context/Authority、invocation時の条件 | S02/S05に従いfail-closed。Previewやschema version番号だけで許可しない | #1121/#1126/#1127 |
| `Invocation→Observation` | Command outcome、別readのProvider/record identity・取得時点・freshness、可視性と権限 | read未実施/不一致/競合・遅延で確定不可なら`UNRESOLVED`、成功応答で代用しない | #1127/#1129 |

- `R1`が`R0`と同じ場合、意図したsource editが実際には起きていない可能性を診断する。No-opを意味変更済みのV04 PASSにはしない。表示・配置だけの操作はS01/N12として別扱いにする。
- `O_read` は `O_cmd` と**独立に取得**するが、強いtransaction snapshotを必須としない。read時点の競合更新、非同期反映、閲覧権限の不足がある場合、CommandとObservationの不一致理由を断定せず`UNRESOLVED`へ分ける。
- V00のbaseline R0とV05実行R1を独立した「同一版PASS」と主張しない。業務モデルの物理DBとの互換性はR1から個別に判定する。旧Physical schemaでも意味的に互換なら、**根拠のある互換**は認め、不必要なMigrationをブロッカーにしない。
- 既存`reviewFingerprint`はShared AcceptanceのAPI値であり、モデル全体のrevisionそのものではない。候補識別用`candidateFingerprint`、Evidence fingerprint、review fingerprint、Acceptance後revisionを混同しない。

### Ownerごとの次の実装可能な一本

初回のEnd-to-end対象は、既存`reference/workspace`をscratchへコピーし、#1124の`add-relation`または`add-validation/positive`候補で`R0→Preview→Accept→R1→reload`を先に**actual scratch操作**で固定する。PR #1137のREADMEにある手順はまだ実行Evidenceではない。Shared Acceptanceの`reviewFingerprint`を用いて受理した時だけ`R1`を記録する。

その後、#1127がR1に対応する既存Function/Driverが何を実際に実行できるかを確認する。**意味変更が既存TargetのObservable Effectに結び付かない場合は無理に同一画面の成功演出を作らず、次に必要な最小Capability/Generator adapterを実装する。** #1121が合成lowering witnessだけでTarget実行と称していないか、#1126が未確認schemaに暗黙対応させていないかを合同レビューする。

### 追加の必須検査（既存V/N/Sの組合せ）

1. **正常系**: V03（R0不変）→V04（R1成立）→V05/V08（R1に整合するFunctionと実行、独立read）。R0→R1のDecision参照がない独立の成功結果はG2 PASS不可。
2. **stale**: N06（R0で提案した後Sourceを変更してcurrent R0'へ）→accept拒否→current R0'のbytes不変。旧candidateを暗黙に再適用しない。
3. **権限**: N08/S05（R1で生成済でもHost/Capability現在値に問題）→Driver呼出し前拒否。事前生成物があるだけでは実行可能と扱わない。
4. **realization**: S02（R1のData変更に旧schemaの互換根拠がない）→readiness不成立。Source editの成功をrollbackしない一方、DB writeへ進まない。
5. **観測**: N11（Command成功でread未実施）→`Observed`や`Verified`を立てない。独立readに必要な権限が無い場合も`UNRESOLVED`。

**G1/G2の状態**: 今回作ったのは接合・判定の規約候補であって新たなCore schemaではない。#1137のStudio Bridge・#1141のlowering・#1136のCUDは依然Draft candidate。既存AcceptanceとRuntimeを同じ実Source変更上で実行したEvidence、full trusted checkoutのqualification、実DB/Target Observationは未取得。各Ownerのpacket検証が済むまでG1/G2とも未合格とする。

## 15. G1実consumer間のスコープ・Kind・Authority接合（2026-10-10）

### 観測した現物と導かれる問題

新たに成立した**未統合の候補PR**をGitHub source / diff単位で再確認した。#1120 [PR #1139](https://github.com/hat47x/tei-platform/pull/1139) は `internal/modelref.Key{Scope,Subject{Kind,ID}}` / `Ref{Scope,Subject,Version}` を提供する。一方、#1124 [PR #1137](https://github.com/hat47x/tei-platform/pull/1137) のStudio selectionは `"data:<ID>"` / `"pattern:<ID>"` というScopeなしのローカル表記。#1125 [PR #1138](https://github.com/hat47x/tei-platform/pull/1138) は `BP-xx` を非CanonicalなBusiness Procedure Reference stepとして投影する。#1121 [PR #1141](https://github.com/hat47x/tei-platform/pull/1141) の`Subject.Kind`は合成の `business/data/function`、#1132 [PR #1136](https://github.com/hat47x/tei-platform/pull/1136) のCUD入力はcaller-suppliedな`pattern/data/source`候補である。

**同じ文字列を参照していることと、同じ業務意味・Scope・Authorityを表していることは異なる。** C0の`Scope`をC1/C3で落としたり、PatternとFunction・Business Stepを名前だけで同一視したりすれば、Referenceの単体試験がそれぞれ通っても、製品のcross-view接合は成立しない。

| 接合先 | 初期Producer / Consumer | G1での合流条件（既存契約を優先） | 必須の負例 |
| --- | --- | --- | --- |
| `C0→C1` | #1120 scoped typed Ref → #1124 Studio selection/source navigation | Project/Workspace等の**実際に確認されたScope**を引き回す。UIのflat tokenを使う場合も、C0境界でexact scopeを復元できることを確認する。暗黙の唯一候補選択をしない | 二つのScopeに同じKind+IDがあるとき、誤ったSource anchorを返さずambiguous/unresolved |
| `C0→Business` | #1125 DFD / #1138 → #1120 / #1124 | `BP-xx`はReference局所ID。explicit identity mapping/採用なしにAccepted Business Stepへ昇格しない。手作業・外部作業はFunction不在でも保持する | `BP-00`と同名のCanonical IDを勝手に作らず、手作業がUIから消えない |
| `Business/Function→C3` | #1124 Pattern / #1125 Step / #1120 typed edge → #1121 Lowering | `pattern` ≠ `function` ≠ `business task`。`uses/implements`等の明示・由来付き関係がない場合はunresolved。型変換をCoreに固定せず、必要なExtension契約で扱う | 同名ID・同versionでも異Kindに暗黙解決しない |
| `C1→Accepted R1→C3` | #1124 Shared Acceptance → #1121 / #1127 | §14の`R0→P→D→R1`系譜と、R1に対応する実Function/Capability/Targetの適合根拠を結ぶ | 合成Acceptance receiptまたは旧R0のBuild artifactだけでR1の実行成功を宣言しない |
| `C2→Cross-view review` | #1132 CUD候補 → #1124/#1126 | `source` locator、owner/FunctionIOは未検証の候補であり、C0のidentityや実Runtime effectにはならない | 空coverage・未知Pattern・旧source evidenceから「所有権違反なし」「実更新あり」を断定しない |
| `C3→DB/Host→Observation` | #1121/#1126 → #1127 | §12のDatasourceごとのschema互換性、実行直前Host Authority、§14の独立Observationを利用 | S02/S05/N11をfail-closedまたはUNRESOLVED。生成物の存在で権限を補わない |

### G0証跡の最新差分と次の手順

- #1139はGoの隔離ミラーでfocused race test / vetを実行済みと報告。ただしexact-head trusted full checkout未実施。Source Indexにはscopeがないため、`sourcebridge.Locate`は同一Kind+IDが複数scopeにあるときに`unresolved`で止める候補である。これは望ましい安全側の挙動だが、Studioや実Worldからの正式接続は未実証。
- #1138はBusiness DFDのJSON/DOTとSource aware diagnostic、human/event/schedule、manual/external、情報フローと責任分担のread-only Reference。unit test sourceが存在するが、PR時点でactual test未実行。Studio #1137側には**任意DFDをreference-onlyで読むAdapter**が追加された。両者の共通JSON shapeとSource freshness、Scopeはtrusted checkoutでConsumer統合を実行するまで未確認。
- #1136は先のレビューで指摘した「candidate factsが空でもclean」の誤解を、head `06a24821881756dd468f9ee27eb7daf81bb5d526` にて `EMPTY_CANDIDATE_SCOPE` とcoverage accountingの追加で緩和。隔離環境9テストPASSとの報告を取得済みだが、実Source-based coverageは依然未検証。
- 横断担当#1119は#1120/#1124/#1121に、ScopeとKindのconsumer-negative-testをIssueコメントで引き渡した。Producer/Consumerの両側が同じexact fixture、revision、Authorityで実行した証跡が出るまで**G1 contract-unverified**を維持する。

Gate G2の最初の実行は、既存Shared Acceptanceを使ったscratch `R0→R1` と、#1127による**R1互換性を検証した実Driver/Web実行**の接続を優先する。DFDからのBusiness Acceptance、全種類のPlugin解決やself-buildをその一往復の前提へ追加しない。ただし三視点の完成を主張する際は、Business viewが`reference-only`のままでは製品受入不可とする。

**非主張**: この監査は現PRのGitHub source/diffとIssueの報告に基づく。今回の#1119担当からはGo/Nodeのfocused testもfull trusted checkoutも実行していない。scope/kindの橋渡しは設計上の未解決項目であり、製品のG1/G2はPASSしていない。

## 16. P0最短実演の実装済み接点を特定する（2026-10-10）

### 静的照合の結論：新規Validation単独ではWeb Commandの挙動は変わらない

最初の一往復に、**受注明細の単価が正であることのValidationを追加し、`order_entry`が評価対象として採用する**変更を選ぶ。これなら物理DBスキーマ改版を無条件に必要とするRelation追加ではなく、業務上の検証結果というobservableな差を最小実装で確かめやすい。

設計上の注意として、#1124のStudio Bridge（Draft PR #1137）の初期版では`applicationUse`が指定されない限り、`table / add-validation / positive`から **`add-validation` 1操作だけ**を生成した。更新後のPRには、明示的な利用選択を伴う二操作候補が追加されている（§20）。一方、現在のmain上の`tools/reference-first-web-target/command_scenario.go`は`driver.Resolver.ResolveApplicationValidations("order_entry")`で**Applicationが明示的に選択した規則だけ**をCommand前に評価する。現在の`reference/world.json`の`order_entry.validations`は次の3件だけであり、新しい`sales.order_line.unit_price_positive`は含まれない。

- `sales.order_line.quantity_positive`
- `sales.order.lines.line_number_unique`
- `sales.order_line.line_number_positive`

**従ってStudioから規則だけを追加してV04が成立しても、V05でその規則の評価まで確認したことにはならない。** この差はWeb側の欠陥ではなく、Data上の規則定義とApplicationでの規則利用を分ける既存ADR-0029の意図である。自動で全Validationを選択する実装はしない。

幸い、**既存main**の`tools/reference-shared-acceptance`には`add-application-validation-use`がある。`practical_validation_loop_test.go`では、`add-validation`と`add-application-validation-use`を**同じcandidate内で受理**し、Previewで`semantics`・`applications`の両方の差分を出し、受理後のDriver resolverで4件の規則を宣言順に取得する試験がすでにある。新しいAcceptanceエンジンや新しいCore Rule選択を作らず、この経路を再利用する。

### 最初のaccepted change候補（Review前提の参考形）

```json
{
  "schema": "tei.reference.change-candidate/v0",
  "id": "critical-order-unit-price-guard",
  "baseModelRevision": "sha256:<R0: revision -workspace の実値>",
  "origin": {"kind": "studio"},
  "intent": "受注明細の単価に正値制約を設け、受注確定前に評価する",
  "delta": [
    {
      "op": "add-validation",
      "namespace": "sales",
      "id": "sales.order_line.unit_price_positive",
      "kind": "positive",
      "attribute": "sales.order_line.unit_price"
    },
    {
      "op": "add-application-validation-use",
      "placement": "applications/order-entry/application.tei",
      "applicationValidationUse": {
        "application": "order_entry",
        "validation": "sales.order_line.unit_price_positive"
      }
    }
  ]
}
```

この例は**候補の構造**を示す。プレースホルダーのRevisionでは実行できず、Workspaceのcurrent R0を取得してReview/Source変更範囲・未解決事項を確認し、人間が明示的に採用する必要がある。元Workspaceを直接変更せずscratch copyで実施する。二つの操作を分離した二回のacceptは、規則だけ存在してCommandで未使用になる中間状態を作るので、最初の縦断には使わない。

### 実装/検証を再利用する短い順序

| Step | 既存の足場 | 提出する証跡と注意 |
| --- | --- | --- |
| A | `reference/studio-authoring/bridge.mjs`（#1137） | PR #1137の更新版は`applicationUse`指定で2操作候補を生成できる。次はscratch E2Eのactual試験・Review/Source再投影・G1 consumer証跡を取得。単独ルール追加はRuntime利用と混同しない |
| B | `tools/reference-shared-acceptance/practical_validation_loop_test.go`（main） | scratch R0、Preview時のsource byte不変、Review fingerprint、人間のAccepted Decision、R1再読込。既存testの成立はテスト対象時点の証拠であり、#1137 exact headのactual実行を代用しない |
| C | `driver.Resolver.ResolveApplicationValidations`（main） | R1の`order_entry`に4件のrule IDが宣言順で現れること。UIの見かけの検証一覧ではなく、Driverの解決結果で確認 |
| D | `tools/reference-first-web-target/command_scenario.go`（main） | R1由来Worldを読み、`sales.order_line.unit_price`の正/非正値を隔離fixtureで変え、`satisfied`と`validation-not-satisfied`の差・Command未実行を検証。未評価/欠損/不正数値を成功に変えない |
| E | 同Web Targetの`commandScenarioRunner`とObservation | Context/current Authorization、Command、別readを区別。synthetic storeの成功はactual external DB writeやS02 schema互換を証明しない |

**既存のactual実装に関する重要な制約**: `command_scenario.go`はOrderLineの各行を個別に`validationeval.Evaluate`し、値が無い場合や評価不能なときに失敗を返す。mainのread fixtureは`sales.order_line.unit_price`を保持する。ただしWeb Commandの送信処理は**synthetic store**を利用し、Observationも`driver.ObservationEvidenceSynthetic`を用いる。従って、Webサーバを実際に起動して操作しても、外部実DBの更新が成立したことにはならない。

### 製品Gateへ変換できるEvidenceの上限

- **E0 — Contract only**: 同じcandidate内の2操作とDriver選択の既存単体試験を確認。契約形の整合まで。StudioやWebの横断動作を証明しない。
- **E1 — Actual scratch + Web / synthetic Provider**: R0→R1のAcceptanceをtrusted checkoutで実行し、R1 WorldのWeb Commandを正例/負例で操作したEvidence。これで**設計→選択→検証→Command境界**の一往復を示せる。ただしDBは合成、実データ永続化/外部観測の製品Gateは未達。
- **E2 — Actual DB Provider + independent read**: #1126/#1127が実ProviderへR1互換のData contractを結び、write resultと独立したDB readを取得、S02/S05とN11を同じ実行packetへ付ける。E1からE2への自動昇格は禁止。E2は現在のWeb synthetic fixtureを本番DBと呼び替えることでは得られない。

**現在の実装Owner判断（初期提案から更新）**: #1124 PR #1137は明示的`applicationUse`付きで2操作候補を既に生成可能となった。次は同PRのNode/Go scratch E2Eをtrusted exact-headで実行し、その受理系譜とSource/Driver結果を提出する。#1127はE1のWeb統合試験と、E2に必要なactual Providerを接続する実機構を区別する。#1121はR1のAccepted Source系譜を合成`trustedAcceptanceReceipt`で置換しない。#1119はE0/E1/E2を別欄で管理し、PR #1137/#1141のDraftを合格と数えない。

**今回の実測範囲**: main上の`reference/world.json`、`tools/reference-shared-acceptance`のREADME/試験ソース、`tools/reference-first-web-target`のCommand/Read実装、PR #1137のBridgeソースをGitHubから読み取って比較した。ここに記した`R0→R1→Web`操作を、この作業環境で実行したわけではない。trusted exact-head checkout / Repository Health / current real-DB Evidenceは未取得、G2未合格。

## 17. E1の「追加した単価ルール」を正負の両方で観測するためのfixture境界

### 静的に確認できる実装上の制約

§16で選んだ `sales.order_line.unit_price_positive` は、単なるRule宣言だけでなく、`order_entry` の `uses validation` に追加されて初めてWeb Commandの対象になる。既存`practical_validation_loop_test.go`は同じcandidateで受理したとき、`ResolveApplicationValidations`が**既存3件の後に単価ルールを4件目として返す**ことを検査している。

一方、mainの`tools/reference-first-web-target/read_scenario.go`のsynthetic Contract Query Executorは、すべての受注明細で`unit_price`に `4000.00` / `4500.00` を返す。受注1002にある既存の負例は**単価ではなく数量0**であり、Command側は評価が不満足になった最初のValidationで停止する。従って、現在のWeb標準fixtureだけで`/scenario/submit`の失敗応答を得ても、**追加した単価ルールが拒否を発生させた証拠にはならない**。

これは実DBを導入して解決すべき課題ではない。既存`staticReadScenarioRunner`によるテスト用read差替えを使い、**新Ruleだけが不満足**となる独立ケースを評価する。現mainにはその横断テストがすでに存在する（§18）。

### E1の二段階証跡（#1127へのconsumer契約）

| ケース | setup/操作 | 必ず確認すること | 証跡の上限 |
| --- | --- | --- | --- |
| E1-A: R1に届く正常系 | 独立scratchで`add-validation`+`add-application-validation-use`を同一Acceptance→R1再読込。既存Web Targetを`-workspace <scratch>`で起動し、受注1001（既存単価4,000/4,500）をsubmit | R1のRule解決は4件で、**4件目の単価も`satisfied`**。Command結果とsynthetic Observationは別欄。R1 Source revisionと実際のWeb起動Workspaceを記録 | actual Web / synthetic Provider。外部DB保存の証拠にはならない |
| E1-B: 追加Ruleの負例 | **同じR1由来World**で`referenceCommandScenarioRunner`のreadだけを試験用に差替え、受注1001の一明細`unit_price=0`とし、数量/番号/一意性は満足させる。既存Web HTTP handlerへsubmit | 先行3件は`satisfied`、4件目の**`sales.order_line.unit_price_positive`だけ`not-satisfied`**、Stageは`validation-not-satisfied`、Authorization/Command/Observationは未実行・未生成 | テスト用fixtureを使った実Go HTTP handler。通常Web実行fixtureや実DBの改変証拠ではない |
| E1-C: Rule未選択の負例 | 同じ新Rule定義があっても`uses validation`のない隔離Worldを読む | Webで解決されるRuleは既存3件のまま。Rule定義の存在だけを「新Rule適用済み」と判定しない | schema/選択のnegative contract test。未選択をRuntime不具合と断定しない |

E1-Bでは、既存の数量0が単価の不満足を**隠さない**よう、既存成功例1001を基点とする。Command送信入力へ単価を任意注入する**本番用API**は追加しない。テスト専用`staticReadScenarioRunner`を、`httptest`と既存の`newWebTargetServer`で利用する案を優先する。HTTP statusやJSON envelopeの形式は既存Web handlerの仕様に照合し、推測で固定しない。

### 最小のhandoffと安全側の判定

- **#1124**: R0→R1の2操作が、既存Shared AcceptanceのPreview/Review/Accepted Decisionを通じて一体として変更された証跡を出す。Source変更はscratch内のSemantic ValidationとApplication定義に限定する。Studio独自Acceptanceは作らない。
- **#1127**: E1-A（actual target起動）とE1-B（合成read差替えによるHTTP handler負例）、E1-C（明示的Use欠落）を別のテストとして記録する。既存Driver/Validation/Authorization/Outcomeを差替えない。E1-Bだけで「通常Targetの実データから単価不正を観測した」とは主張しない。
- **#1121**: R1に紐づくRuleの選択とCapability/Target readinessを独立証跡で繋ぐ。合成ReceiptだけではE1-Aの実行結果を証明しない。
- **#1119**: Evidence packetには`accepted_revision`、`application_rule_set`、`web_workspace_source`、`fixture_provenance`、`rule_results`、`command_status`、`observation_nature`の根拠をリンクする。E1-A/B/Cは異なる観測条件として区別する。§18の既存テスト結果を再利用し、二重実装・二重計上を避ける。E1→E2（実DB Provider+独立read）の自動昇格は禁止する。

**現状**: この節は観測条件を整理した設計上の区分。E1-B相当の既存Go横断テストとその限界は§18に記録されている。この横断担当では当該Goテストのactual実行、trusted full checkout、Repository Health、実DB操作は**NOT RUN**。G1/G2のGate状態は変更しない。


## 18. 既存のGo横断テストの再利用と残るG2接合点

§16の最短実演のため、新しい単価Validation専用Runnerをもう一つ作る必要はない。現在のmainに既に次のテストソースが存在することを確認した。

- `tools/reference-shared-acceptance/practical_validation_loop_test.go`: `TestPracticalRuleChangePreviewsAndAcceptsBothMeaningAndApplicationSelection`。一つのcandidateで「Validation追加＋Application利用宣言」を受理し、Driverが4規則を解決する。
- `tools/reference-first-web-target/practical_business_rule_test.go`: `TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList`。scratch WorkspaceでShared Acceptance CLIの`revision→preview→review→accept→reload`を経て、旧3規則と新4規則のCommand動作を比べる。2行目の単価をテスト内で0に変え、旧規則は満たすが新規規則だけが拒否する。さらに`httptest`で`POST /scenario/submit`の拒否とCommand/Observation未実行を確認し、単価を正に戻せば成功する。
- 同じGoファイルの`TestReferenceCommandScenarioRejectsMissingValuesAcrossAllLines`は追加規則の参照値が欠落したケースで、成功扱い・通常の業務Validation拒否に丸めずExecution Errorにする。

このソースは未実行の静的確認であり、現時点のE1 PASSではない。各既存テストは合成read/Command、test-only accept、HTTP handlerの`httptest`を用いる。実ブラウザや実DB、実際の人間の承認を証明しない。なお通常の`/scenario/submit`要求に単価の任意指定は無い。負例のためにWeb APIへ任意の価格上書き機能を追加しない。既存order 1002は数量が0で旧規則に先に失敗するため、今回の新規単価規則の負例には使わない。

### 主担当への最小追加要求

| Owner | 既存テストで足りること | 新たに必要な接合 |
| --- | --- | --- |
| #1124 | Shared Acceptance側の2操作受理 | PR #1137の更新版は明示的Application useの2操作候補に対応。次は既存scratch E2Eのactual試験を実施し、R0→R1とSource/Driver連結を示す |
| #1127 | scratch R1由来のWeb handlerで正/負/欠損のValidation実行 | 同じR1のProvider/Target readinessとreal Observationを後続で実証。別のunit-price用Runnerは作らない |
| #1121 | lower段階のversion/capability候補 | R1の真正なAccepted Sourceを受け取り、実行時ContextやAuthorityを合成のTrustへ置き換えない |
| #1119 | 既存試験のSourceとP0 Caseへの対応 | trusted exact HEADでの実テストreceipt、R0/R1/target/observationの証跡を収集し、E1/E2を分ける |

trustedなactual full checkoutでの最初の対象コマンド（今回の作業では**実行していない**）:

```bash
go test -count=1 ./tools/reference-shared-acceptance -run '^TestPracticalRuleChangePreviewsAndAcceptsBothMeaningAndApplicationSelection$'
go test -count=1 ./tools/reference-first-web-target -run '^(TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList|TestReferenceCommandScenarioRejectsMissingValuesAcrossAllLines)$'
bash tools/repository-health-check.sh
```

G2に残るのはStudio #1137の実consumer経路、accepted R1からのLowering / Authority接続、およびactual Provider・独立Observationである。上記test sourceの存在をG1/G2のPASSへ昇格しない。

## 19. G2証跡packetの静的Preflightを実装（2026-10-10）

§16〜18の初回業務シナリオに対し、#1119が所有する軽量Reference `tools/reference-critical-path-evidence-audit/` を追加した。これは**Gate合格判定器ではなく、各レーンからのEvidence packetの欠損・合成証拠の混入を機械的に弾く前置検査**である。Core、Shared Acceptance、Driver、既存Gate C Evidence Auditのいずれも置換しない。

| Scope | 静的に検査する内容 | 上限・根拠 |
| --- | --- | --- |
| E0契約 | producer/consumerのsource path、正負テスト名、`source-only/synthetic-tested/actual-observed`の報告種別 | IDやテスト名は単なる申告。テスト実行/契約互換性は別のactual Evidenceを要す |
| 資格確認 | exact PR headとreviewしたmainの一致、full Repository Health PASS・前後clean・receiptの入力欄 | 文字列とflagの一致に過ぎず、実際のtrusted checkout、current remote mainは本toolで証明しない |
| E1提出 | `R0≠R1`、candidate/review/Decision/reload、`add-validation`＋`add-application-validation-use`の**同一**変更、R1上の`order_entry`明示選択、Web正負・N11、`running-server`の申告 | `httptest`/handler-onlyはE1の実Webサーバ操作に自動昇格しない。Synthetic ProviderはE1の範囲内で許容、外部DB実証ではない |
| E2提出 | actual-db、Datasource/schema compatible、current Host Authority、独立Command/read receipt/record、S02/S05拒否試験 | Web synthetic/共通receiptではE2不可。read独立性とphysical stateの真偽は独立Reviewが必要 |

入力schemaはReference固有の `tei.critical-path-evidence/v0`。コマンドは `go run ./tools/reference-critical-path-evidence-audit -packet <file> -head <exact-HEAD> -main <reviewed-main-SHA>`。schema/type不正や必須証跡不足は `INCOMPLETE` とdiagnosticで返す。整合する提出物も `REVIEW_REQUIRED` で止まり、**`productGate` は必ず `UNQUALIFIED`**。exit code 0をProduct Gate PASS / merge権限に変換してはならない。詳細と未取得Evidenceは同toolのREADMEに固定。

### この回の実検証と限界

- `main.go` と `main_test.go` は Go 1.23.2 の**単独コピー**で `GO111MODULE=off go test -race -count=1 -v main.go main_test.go` を実行し、11件PASS。さらに `GO111MODULE=off go vet main.go main_test.go` PASS、`gofmt -l`空を確認した。GitHubの該当Blobはこの試験済みファイルのgit object SHAと一致：`58643a702479db12219b317281c975687b743363`（main.go）、`30ec224d5722d4584e54e9fc2cb5544ac4f1ccef`（main_test.go）。
- 否定試験はsynthetic WebでE2申告を拒否、Validation-onlyの候補を拒否、R0=R1やstale head/mainを拒否、readとCommand同一receiptを拒否、S02/S05/N11証跡欠落を拒否、健康試験`NOT_RUN`を拒否、producer/consumer空のE0を拒否、handler-onlyのE1昇格を拒否、未知field・余分なJSON値を拒否する。
- この限定的Goテストは**trusted full TEI checkoutのPackageテストや`tools/repository-health-check.sh`の代替ではない**。このコード自身を含むPR headのfull checkout資格確認、実E1 `R0→R1→Web`、外部DBの実E2・別read、独立監査はいずれも未実施。
- `REVIEW_REQUIRED`になったpacketであっても**未取得な証拠の空欄を「PASS」で埋めることは禁止**。#1124/#1127が実験と根拠を別々に提出し、#1119は同一R0→R1系譜とScope/Authorityの実consumer確認を担当する。

**現状判定**: 横断レーンの証拠申告を検査する仕組みが増えたに過ぎず、G0はconsumer合意待ち、G1/G2/G3/G4の合格はなお未取得。このツールが合格を出せない設計は、意図的な製品安全境界である。

## 20. Studio 2操作接続の進捗反映とWeb負例のE1境界（2026-10-10）

### Studioの設計ギャップは一段解消したが、E1実行証拠とは別

§16/§18の「Studio Bridgeが1操作しか出せない」という記載は、その時点の候補についての監査履歴である。#1124のDraft PR #1137はその後更新され、現在の `candidateFromEdit` は `applicationUse:{application,placement}` が明示されたとき、既存Shared Acceptance向けの `add-validation` と `add-application-validation-use` を**同一candidateの二つのdelta**として生成する。対象Applicationが該当Semantic Namespaceを使うこと、既存重複選択がないこと、危険な配置pathでないことをReferenceで確認し、実際のSource更新権限は既存Shared Acceptanceへ委譲する。

さらにPR #1137の `e2e.mjs` はscratchで同じ受理系譜を確認し、再投影でApplicationの利用宣言と変更対象Source二件を確認するテストを持つ。ただし **trusted exact-head Node+Go実行は未取得**。Sourceの2操作を表現できるようになったことだけで、実動作のV04/ G1 PASSとしない。#1124へは新機構の重複開発ではなく**既存E2Eスクリプトを資格実行してreceiptを提出すること**を次の作業として渡す。

### E1: running-serverとHTTP handlerの負例を同一視しない

mainの通常Web fixtureでは新規`unit_price_positive`を失敗させる単価0が用意されておらず、受注1002の失敗原因は数量0である。一方、既存`TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList`はGo test内の `staticReadScenarioRunner` で単価0を注入し、新規4番目のValidationによる拒否を確かめる。**これは有益なhandler-level証拠だが、実行中のWebサーバが新規単価規則を拒否したことはまだ示さない**。

横断E1の「actual running Web正例・負例」という要件に対し、従来preflightは正負receipt名と共通 `web.runMode=running-server` だけを確認しており、**負例が数量検証なのか新規単価検証なのか**、**negativeだけhandler-onlyなのか**を形式的に分離できなかった。そこで`tools/reference-critical-path-evidence-audit/` のE1申告欄に次の二項目を追加した。

- `web.negativeValidation`: 実Web負例として申告する**失敗規則**。新規単価規則と一致しないと `E1_NEGATIVE_SELECTED_RULE_RUNNING_WEB_EVIDENCE_REQUIRED`。
- `web.negativeEvidenceMode`: **負例側個別**の実行形態。`handler-only`を`running-server`と申告しない。E1提出としては`running-server`を要求し、異なれば同じ診断を出す。

これらはあくまで**申告の整合性を検査する欄**である。値を`running-server`と記述しただけでは実行・証跡の真正性を証明できず、ProductGateは従来どおり常に`UNQUALIFIED`。人が実Web receiptとProvider/Ruleの根拠を検査する必要がある。

#1127への次の実装要求は、通常Web APIへテスト専用の単価上書きparameterを足すこと**ではない**。隔離されたReference起動構成または環境内のtest-only fixture selectionを用いて、R1の同一World・Ruleのまま、actual running Webで単価0を観測可能にする**最小手段**を調べる。実Provider・Production Targetへこの試験機構を混入させず、合成Providerだと明示する。実現までE1のrunning-Web負例は`NOT_RUN`であり、既存httptestをE1実サーバ証拠に繰り上げない。

### 次に取得すべき証跡（作業順）

1. #1124 PR #1137: exact-head `node --test reference/studio-authoring/bridge.test.mjs` と `TEI_STUDIO_E2E=1 node reference/studio-authoring/e2e.mjs`、R0→R1とN06、source digest。mainへ無資格統合しない。
2. #1127: 既存Go `TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList` をtrusted current mainで実行し、Webの正例と単価0のhandler負例を分けて報告。
3. #1127/#1121: R1との互換根拠つき`running-server`正例を取得。新規単価負例もrunning-serverで実施できる安全なfixture接続を確立してから、負例receiptを記録。
4. #1126/#1127: E2は別途actual DB、同じR1に対するDB互換性・current Host Authority・Commandと独立read、S02/S05/N11を実証する。

**現状**: §20は更新後のGitHub source/PRに基づく設計・実装照合およびpreflight code/testの変更。新規のGoテストはこの横断担当でtrusted full TEI checkout実行していない。Node E2E、running Web、actual DB、Repository Healthも未実行。#1119 G1/G2は未合格、親PR #1135はDraft継続。

## 21. G2証跡監査の敵対的入力対策と現状負例（2026-10-10）

§19のpreflightを単なる申告欄の検査として利用する際、次の二つの**偽の形式合格経路**を発見し、Reference側へ修正した。

1. `E1/E2`を申告しながら`contract.evidenceClass=source-only`または`synthetic-tested`のまま、他の文字列だけ埋めると、先行実装では`REVIEW_REQUIRED`になり得た。現在は **`E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED`** として`INCOMPLETE`にする。Web Providerがsyntheticでも、E1に必要なrunning Webの**実行**の申告まで不要になるわけではない。
2. Goの標準JSON decoderは重複したfieldで最後の値を採用する。さらに`io.LimitReader(...,1 MiB)`は先頭1 MiBが有効なJSONなら**残りの入力を読み落としてEOFとする**ため、hidden trailing contentを検出できない。修正後は入力サイズを**1 MiBを超えて拒否**し、UTF-8、未知field、重複key（入れ子を含む）、`claim/Claim`のようなcase-folded alias、過深nest、追記JSONを拒否する。Source/Target/Authorityの実体をこのtoolで検証できるという意味ではない。

現状の機械可読負例は `tools/reference-critical-path-evidence-audit/testdata/p0-order-unit-price-incomplete.json`。既存Goテストソース名と、Validation追加＋Application利用宣言の二つのdeltaを明示する一方、**実行済みではないものを埋めていない**。現在の`main`と候補PR headを引数に指定してReference CLIを**隔離Go環境**で実走査し、次を確認した。

- 結果 `status=INCOMPLETE`、`productGate=UNQUALIFIED`、終了非ゼロ。
- `ACCEPTED_R0_R1_LINEAGE_MISSING`、`ACCEPTANCE_REVIEW_AND_RELOAD_EVIDENCE_MISSING`、`E1_RUNNING_WEB_TARGET_RECEIPT_REQUIRED`、`E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED`、`ACTUAL_WEB_POSITIVE_AND_NEGATIVE_RECEIPTS_REQUIRED`、`N11_UNOBSERVED_CONTROL_MISSING`、`REVISION_QUALIFICATION_STALE_OR_MISSING`、`TRUSTED_FULL_HEALTH_NOT_RECORDED` を出力。後続の更新で「未実行」を確定PASSへ改ざんしていないか、fixtureの回帰テストでも確認する。
- 既存の#1124 E2Eと#1127 Web/Driverの受注明細Validationテストを新規に重複実装するのではなく、**そのactual実行Evidenceを受け取る**ための検査に限定する。

### 試験の所在と限界

Go 1.23.2の**独立コピー**で `GO111MODULE=off go test -race -count=1 -v main.go main_test.go` は20件PASS、`GO111MODULE=off go vet main.go main_test.go`はPASS、`gofmt -l`は空。これらの対象コードは、その時点のGitHub blobと一致している：`main.go=84cf10a53fe560cd3595e058e1476a1f6055571c`、`main_test.go=b8e113e63135b6bb203a541afd63cc4fa53bf1e1`、fixture=`7a499e246489b3ef4b8e18263c6c8deb253a39e9`。

**未取得**: trusted full TEI checkoutのexact-head Package test、`tools/repository-health-check.sh`、Web running-server上の単価負例、実DBのCommand＋独立Read、実Human Acceptance。Code/testの孤立実行はこれらに代わらない。E1/E2の形式上の入力に`actual-observed`と記入しただけでは、actual観測の真偽を証明したことにはならず、preflightは常に`productGate=UNQUALIFIED`とする。

## 22. G1 Studio consumerの実Source照合とPreflightコスト上限（2026-10-10）

### #1124 の実装候補を `reference/world.json` で再照合

#1124 の[Draft PR #1137](https://github.com/hat47x/tei-platform/pull/1137) head `1e724dd78d3373ae7ecb3d707df284f00185cdae` の `reference/studio-authoring/bridge.mjs`（Git blob `a3105c8b3feb9defd3e7e1f3b46c8697be4a92d4`）と、main の `reference/world.json`（blob `2e5cb6dbbea562babd82487e32e5b65fa2b5a62b`）をGitHubから取得し、**実際のソース文字列を隔離V8上で評価**して6ケースを照合した。Revisionには形だけの `sha256:aaaa...` を渡したので、この実験をAccepted revision検証に使わない。

| 探索・候補操作 | 観測結果 | 現時点の製品上の意味 |
| --- | --- | --- |
| `data:sales.order_line`をread-only投影 | Table/ER/Functionは`available`、BusinessとSourceNavigationは`unresolved` | 現Worldで三つのビュー候補を生成できたが、四視点一致・Source anchor・Acceptanceの証拠ではない |
| 正値Validationの追加のみ | `add-validation`1操作 | Webの受注確定に新Ruleは採用されない。§16の初期不備は再現可能 |
| `applicationUse:{application:order_entry,placement:applications/order-entry/application.tei}`を明示 | `add-validation`＋`add-application-validation-use`の同一Candidate | #1124の2操作Proposal拡張を実Worldで確認。**Shared Acceptanceで受理・実Sourceを更新したわけではない** |
| 危険な`../../escape`配置 | `UNSAFE_APPLICATION_SOURCE_PLACEMENT`で拒否 | Candidate側の局所的な安全検査が存在することを確認 |
| `repair_intake`に`sales` Ruleを勝手に選択 | `APPLICATION_DOES_NOT_DECLARE_SEMANTIC_USE`で拒否 | ApplicationとSemanticの関係を拡大解釈せず拒否 |
| `canonicalProcedureUsed=true`を僭称したDFD | `UNSUPPORTED_BUSINESS_DFD_AUTHORITY_OR_SCHEMA`で拒否 | Reference-local Business情報をAccepted Procedureへ自動昇格しない |

これはPR #1137の**静的に取得したソースを使う in-memory/isolated smoke**であり、`node --test`や `TEI_STUDIO_E2E=1`、trusted current checkout、Go Workspace Loader、実行Webには接続していない。Scope付きC0 `modelref.Key`とのconsumer統合、Source Index locator検証、R0→R1の受理実走は未解決。#1124 ownerに重複実装を求めず、既存 E2E のexact-head 実行と、出力packetの提出を優先する。

### #1119 の既存Go Evidence preflightを敵対的な大量キー入力に耐えさせる

§19〜21で導入した`rejectDuplicateJSONKeys`は、最初の実装では重複keyの判定に`strings.EqualFold`で既出key**すべて**を順次照合していた。入力1 MiBという制限があっても、互いに異なる大量のkeyを与えられると照合数が**二乗的に増加**する。プロダクト横断の証跡処理で不必要なCPU消費を発生させないため、Go標準`unicode.SimpleFold`の等価クラスを正規キーにし、map検索へ置き換えた。Locale依存の推測や文字列の人為的な正規名寄せはしない。

- 既存の`claim`/`Claim`拒否に加え、Unicode `Key`/`key`を拒否する回帰テストを追加。
- **異なる10,000 key**の入力を拒否・誤検出なく解析する回帰テストを追加。これは本CLIの1 MiB上限内の入力。
- **独立 Go 1.23.2**で `GO111MODULE=off go test -race -count=1 main.go main_test.go` を実行して**22件PASS**、`GO111MODULE=off go vet main.go main_test.go` PASS、`gofmt -l`空。GitHubに書き込んだ対象blobは同じローカル実行ソースと一致（`main.go=59dfc5c7b48a98891309b40fd9e11577bc9b6128`、`main_test.go=14b107b12556388c7dfc4c6b6570541422cc7ccf`）。
- これは証拠検査tool単体の挙動であって、**受注業務の実行・保存・読み直しを実証するものではない**。GitHubへの直接git networkアクセスは現在の隔離環境では利用できず、trusted full checkoutの`tools/repository-health-check.sh`は未実施。

### 残るGate通過への一本

#1124の2-op Proposalを同一scratch Source `R0→R1`で受理し、#1121/#1127の**実Web Target**がR1で選択された4規則を参照することを、正値・新単価0の負値両方で確認する。ここで`httptest`や`handler-only`の負例を`running-server`へ無条件に読み替えない。#1126はDB Schema readiness/actual Providerの独立条件を判定する。どちらも既存Referenceを再利用し、新しい正本・Runner・Synthetic Backendは増やさない。

**Gate**: G0のconsumer交換は増えたが正式接合未取得。G1/G2未合格、G3/G4未合格。§22を含むPRは引き続きDraftとする。


## 23. WebのSource revisionとB/R/Tは異なる証拠である（2026-10-10）

### 実行中WebのHTTP応答だけでは受理済みR1を証明できない

mainの `tools/reference-first-web-target/main.go` は `-workspace` を読み取り、`projection.go` と `server.go` はWeb応答へ `handoverBaselineId`、`baselineRepositoryRevision`、`targetBuildRevision` を出す。一方、**Shared Acceptanceが算出するmanaged-source `modelRevision` は現manifest/Command envelopeに含まれない**。

従って、ブラウザ/HTTPでCommandとValidationの結果を観測しても、`R0→R1`の**どのaccepted SourceをWeb Targetが実際にロードしたか**は応答だけでは立証できない。Git SHAの `T` とSource revision `R1`を文字列一致させるのも誤りである。

PR #1135のE1 preflight `tei.critical-path-evidence/v0` に、以下の**独立した申告欄**を追加した。

| 追加欄 | 最小必要内容 | 証跡上の限界 |
| --- | --- | --- |
| `web.workspaceRevisionReceipt` | 実際にWebへ渡したscratch `-workspace`を指定してShared Acceptance `revision -workspace` を観測したR1、Source locatorと観測時点。少なくとも起動前後の一致・変化有無を提示 | Webがロードした正確な瞬間や競合不在を文字列だけでは証明しない |
| `web.targetLaunchReceipt` | 起動した**実Target**のコマンド引数（credential除外）、使用Workspace、Target artifact/revision `T`、process・loopback endpoint・稼働確認・停止時点 | 起動申告は現行Host Authorityや実DB対応を保証しない |

これらはWeb正負HTTP応答receiptと別物とし、未記入・同一receiptを使い回した場合は `E1_WEB_WORKSPACE_R1_AND_TARGET_LAUNCH_PROVENANCE_MISSING` で `INCOMPLETE`。既存 `web.acceptedRevision` と `acceptance.r1` の一致という**申告だけ**では、Web実行時のR1使用を証明できない。CLIは依然としてファイルの実体や真偽を検査しないため、`productGate` は常に `UNQUALIFIED`。提出された生証拠を別reviewで突合する。

### 並走#1121のWeb HTTP smoke testをE1へ誤転用しない

#1121の[Draft PR #1141](https://github.com/hat47x/tei-platform/pull/1141) には `TEI_LOWERING_WEB_SMOKE=1` で、既存Web実行バイナリを`go build`し、loopback HTTPを起動して3件の既存Validation・Command・合成Observationを観測するopt-in Go試験が追加された。**これは既存Web Targetのlive process接点として有用**だが、変更後scratch R1の4番目の単価Validationを評価しておらず、E1のSource変更→Web効果を実証するものではない。#1127がlive serverの起動やHTTP client処理を重複実装する前に、#1141の実装上の足場と既存Gate C runnerを比較・再利用する。

さらに#1141の現在の `web_target_smoke_test.go` には、横断証跡として確認が必要な二点がある。

1. Web起動で `-baseline-id B -baseline-revision <head>` を渡している。既存Gate B handoverの固定基準 `R=e5231839a5dcc62cce1181a2731fcb7c70d16736` と、今回実行する候補Targetの `T=<head>` は**別identity**。単にsmoke用placeholderとしてB/Rを埋めない。実基準Rを使う、またはBとしての証拠と称さない方法へ改める。
2. 事前事後clean判定は `git status --porcelain --untracked-files=all` のみ。ignored path、`assume-unchanged`/`skip-worktree` 等の非標準index状態を見落とし得る。trusted qualificationでは既存の検査手順を再利用し、clean判定の高さを別途明示する。

このsmoke testは本レーンで**ソースを静的照合したのみ**であり、追加試験や実Web serverをここで動かしたという意味ではない。

### 最短の実証手順

1. #1124: accepted scratch SourceのR0→R1、review/decision/source reloadを実際のE2Eで証明。
2. #1121/#1127: 同一scratchを使用するWeb TargetがR1を読んだことを、independent revisionと起動証拠で対応付け、正値4規則と単価0（テスト限定）の拒否を**live Web**で取得。通常のWeb APIへテスト専用mutation機能を混入させない。
3. #1126/#1127: physical DB/Host Authorization/independent readはE2で別途取得。外部DB効果をsynthetic Observationの結果から推論しない。

**状態**: E1証拠の受渡し欠落を検査するGoコード、負例Goテスト、README、現状fixtureを更新済み。今回の更新分のtrusted full checkout Go test / Repository Health / live Web / actual DBは**NOT RUN**。G1/G2もPASSにしない。


## 24. trusted full checkoutの実行手順を固定する（2026-10-10）

### source qualificationと製品E1/E2の資格を分ける

§19〜23の横断Evidence preflightは、候補PR内Go/READMEの隔離試験と、実製品のrunning-Web/DB証跡を区別する。しかし**当該候補の最新HEADと最新mainで、既存Go packageを実行してよいか**を判断する再現可能な手順が不足していた。

そこで親担当の `tools/reference-critical-path-evidence-audit/qualify.sh` を追加した。これはスコープを**#1119のDraft PR自身の資格試験**に限定する。#1120/#1124/#1121/#1127の実装を複製しない。

- 実行前: `TEI_CP_EXPECTED_HEAD` と `TEI_CP_EXPECTED_MAIN`（ともにexact SHA40）を必須とし、remote fetch後のmain・work branch一致、mainがcandidateの祖先、変更pathが親レーン内に限られることを検査する。ignored/未追跡を含むclean状態と非標準Git indexも検査する。
- focused: Evidence preflightのGo race test/vet/gofmt、既存Shared Acceptanceの正例・stale拒否、既存Web Driver/HTTP handlerの単価Validation正負・欠損を**既存testコードで**実行する。
- full: `tools/repository-health-check.sh`を実行後、HEAD/clean/current remote main・branchが動いていないことを再fetchして確認する。
- 出力は`CRITICAL_PATH_LOCAL_SOURCE_QUALIFIED_ONLY`とexact revision/runner blobのみ。G1/G2/実Web/実DB/独立readのPASS、他Draft PRの資格、独立Review、mainへの自動mergeを意味しない。

起動例（**ここでは実行していない**）:

```bash
TEI_CP_EXPECTED_HEAD=<PR-1135の最新SHA40> \
TEI_CP_EXPECTED_MAIN=<origin/mainの最新SHA40> \
bash tools/reference-critical-path-evidence-audit/qualify.sh
```

この手順はcurrent mainが更新された後で古い親ブランチをそのまま合格扱いにしない。mainの新コミットを通常の安全な方法で取り込み、exact HEADを再指定したtrusted完全checkoutで再実行する必要がある。差分の形式上の非競合は資格試験の代わりにならない。

**未取得Evidence**: 今回はGitHubに資格スクリプトとrunbookを追加した段階。直接gitアクセスできない現セッションでは、このスクリプトの実full checkoutは走らせていない。親PR #1135はDraft/Gate未合格を維持し、完了時にのみmain統合とIssue完了を判定する。


## 25. 合成worktreeでStudio→R1→Webの実接続を圧力試験する（2026-10-10）

### 無関係なDraft owner branchを改変せず横断試験する方法

並走#1120/#1124/#1121の単体Contract候補が増えても、各レーンが**同一Workspace・同一Source Revision・同じDriver**で動く証跡がないままではG1/G2を宣言できない。4本のDraftをmainへ先行mergeしてから不整合を発見する方法は避ける。

親#1119の`tools/reference-critical-path-evidence-audit/compose-smoke.sh` を追加した。GitHub正規remoteの **current main + #1119/#1120/#1124/#1121のexact tip SHA** を引数に、`git worktree add --detach`で一時的なworktreeを作り、その中だけで4つの候補Commitをmergeする。競合時はそこで終了し、常設branchのref/Source/mainには書き込まない。テスト完了・失敗時の双方で一時worktreeを回収する。合成したHEAD/TREEと元SHAを区別する。

合成環境で実行する既存の検査は、Studio Nodeテストとscratch E2E、typed modelref/Workspace Goテスト、Lowering候補、既存Web/Shared Acceptanceの正負、親Evidence preflight、full Repository Healthとする。この手順は**mainへ統合するPRの資格検査そのものではない**。各Ownerがexact-head/mainで改めて資格を通す必要がある。

### E1-Aの不足を埋めるtest-only live Web正常系

`tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs` を追加し、上記合成環境に閉じた一つの実行経路を実装した。

1. 基準Source `reference/workspace` をsymlink拒否後に**一時scratchへコピー**。Shared Acceptanceの`revision`で変更前のR0を取得する。
2. #1124 Studio CLI `propose`で`unit_price_positive`の`add-validation`と`order_entry`への明示`add-application-validation-use`を一つの候補にまとめる。Preview/ReviewはSource bytes不変。
3. Shared Acceptanceを**テスト用判断**でAcceptし、Source2ファイルだけが変更されたこと、再読込R1がR0と異なること、`tei world`のApplication Rule選択が4件であることを検査する。**人間の受理ではない**。
4. 既存 `tools/reference-first-web-target` を同じ合成HEADでビルドし、scratch R1の`-workspace`で**実際のloopback Web process**を起動する。固定B/Rと合成Target Tを独立値として渡す。
5. manifestのApplication/Baseline/Targetを照合し、`POST /scenario/submit`で受注1001のCommandを実行。**4件のValidation ID/順序/結果**、Command stage、synthetic Observation、credential非漏出を検査する。実行後もSource revision R1と管理対象Source2件だけの差分を再確認する。

成功時の表示は **`E1_A_LIVE_HTTP_SYNTHETIC_PROVIDER_ONLY`**。これまでの`httptest`でなく実プロセスのHTTP正常系を対象にできるが、実行時のSnapshot同一性・current Host Authorityの絶対証明ではなく、**E1の単価0のlive Web負例は未実装**。普通のWeb APIへ任意の価格注入パラメータは追加しない。実DB Provider・独立再読込・S02/S05・Business DFDのAccepted接合も別Gateである。

起動は**full checkoutでの手動実行のみ**。4つのDraft SHAとcurrent main SHAをその都度取得し、以下を実行する。詳細はtoolのREADME参照。

```bash
TEI_CP_COMPOSE_MAIN=<current-main-SHA40> \
TEI_CP_COMPOSE_1119=<PR-1135-HEAD-SHA40> \
TEI_CP_COMPOSE_1120=<PR-1139-HEAD-SHA40> \
TEI_CP_COMPOSE_1124=<PR-1137-HEAD-SHA40> \
TEI_CP_COMPOSE_1121=<PR-1141-HEAD-SHA40> \
bash tools/reference-critical-path-evidence-audit/compose-smoke.sh
```

### 未解消の真のクリティカルパス

**G1**では#1120 scoped RefとStudioのLocal IDを暗黙に等価扱いせず、Producer/Consumerの正負検証とBusiness DFDのAccepted Authorityを後続で接続する。**G2**ではE1-B（live Webで新規単価規則が拒否）、current Provider/HostのS02/S05、実DBへの変更と独立readのE2を別に取得する。合成worktreeのSource-to-Web正常系を得てもこれらのGateのPASSには昇格しない。

### Source R1/Target/HTTPの限定receipt（提出前検査）

実行後にすべてのscratchを破棄していた最初のE1-A試験では、報告者が同じworktreeの**Source R1、起動Target T、実HTTP結果**を別々に再検査するためのファイルを提出できなかった。これは§23で定めた証拠連結の不足であり、`web.acceptedRevision=R1`との自己申告だけでは埋められない。

修正後の`live-r1-smoke.mjs`は任意の`TEI_CP_EVIDENCE_DIR`（**リポジトリ・一時scratch外、存在しない絶対path**）に、実験成功後だけ`workspace-r1.json`、`target-launch.json`、`web-positive.json`を新規作成する。Sourceでは変更2ファイルの**受理前後のSHA256**、Source R0/R1とfingerprintを保存。Targetは起動バイナリSHA256・基準R・Build T・loopback実起動情報、Webは全4件のValidation ID/結果とCommand/合成Observationのみを保存する。Secret/credentialや応答原文は出力しない。保存はオプションであり、デフォルトのCI相当試験はファイルを残さない。

これらは**同一のテスト実行が記録した相異なる段階のreceipt**である。外部署名、Human Approval、Host権限、DB実永続化、第三者の独立Readを意味しない。途中失敗で残った不完全なファイル、または3件のreceiptを使い回した申告はE1合格に数えない。さらにE1-B（実Webで新たな単価0を拒否）が依然欠落し、G2は未合格。

### 親ブランチへのmain同期（今回の実操作）

Issue #1119の現Draft作業branchだけに、current main `e084b89f19d4ce53047906c03e28746af56afa4d` を**非force・2親の通常merge commit**として取り込んだ。事前にmainの変更2 path（`plan/design/GENERIC_CAPABILITY_MIGRATION_DESIGN.md`とISSUE-001067-cross-project-plugin-promotion.md）が親PRの8変更pathと重ならないことをGitHubで照合し、mainの2 blobを既存branchのtreeへ載せた。作業branch head `1c8cd60699d5eaff98df452ee651c7f63f7bb3c5`は第1親が旧head、第2親がmainとなり、GitHub compareは**behind 0**。正本mainや他Ownerのbranchを変更していない。

これは**Gitの系譜/変更pathが整合したことの確認のみ**である。統合後の最新HEADでtrusted full checkout/Go/Node/full Repository Healthは実行できていないため、資格PASSやmainへのPR mergeを意味しない。mainが再び進めば最新pinと統合結果を再確認する。

**実際の資格と証跡**: この節の更新はGitHub上の新しいtest-only Node/Bashスクリプトとsource契約の静的照合による。V8ではESM importを除いたNode本体の構文が解析可能なことを確認したが、これは`node --check`の代わりではない。ネットワークが使用できない現セッションで合成worktreeを作って実Go/Node/Web/Repository Healthを実行したわけではない。親PRはDraft/未統合、G1〜G4未合格。

## 26. 最新main統合後の横断再判定（2026-10-12）

### 実装の「存在」と製品受入を分離する

旧§1〜25は、2026-10-10時点に作成した作業計画と、その時々の調査記録である。当時の「PR #1135はDraft／未統合」「#1120・#1124・#1125・#1121はDraftのみ」という表記を**現在の状態として読まない**。2026-10-12にGitHub APIから確認した事実は次のとおり。

| 対象 | mainへの統合状況 | この統合だけでは成立しないもの |
| --- | --- | --- |
| #1119 / PR #1135 | 2026-10-11 08:00 UTCにmerge（統合commit `963de43f2abe761a583736fb890c5a6768dfbdb8`） | G0〜G4の製品受入、trusted full Repository Health |
| #1120 / PR #1139 | 2026-10-11 07:57 UTCにmerge | scope付き意味参照から実行機能への実接合 |
| #1125 / PR #1138 | 2026-10-11 08:00 UTCにmerge | 参照用DFDを受理済み業務手順へ昇格する権限 |
| #1121 / PR #1141 | 2026-10-11 08:02 UTCにmerge | 実Host/Provider・下流効果の適合性、第二Target |
| #1124 / PR #1137 | 2026-10-11 08:02 UTCにmerge | 同じSource R1を用いた一往復の実行、利用者による受入 |

今回の点検で参照したmainは`b29eaf89348bbed5b59816dff4292c4bf678d4f5`（検査時点の値であり、将来のmain固定値ではない）。計画書の現行配置は`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md`で、旧`plan/`直下のパスは使わない。**Issue #1119は現在もopen**。PR #1135のmerged状態を本Issueのcompleted判定へ読み替えない。

### 先に解消すべき横断契約の不整合

main上の`tools/reference-critical-path-evidence-audit/main.go`には、`acceptance.candidateFingerprint`と`reviewFingerprint`の**同値**を要求する条件がある。しかしShared AcceptanceのCandidate指紋とReview指紋は別の内容から生成される。正しい契約は、Review投影が保持する**対象Candidateの指紋**とCandidate指紋の対応を確かめ、Review自身の指紋は別値として保持すること。現在の実装では正常な受理を誤拒否し得る。

この修正は[#1163](https://github.com/hat47x/tei-platform/issues/1163)が所有し、[Draft PR #1262](https://github.com/hat47x/tei-platform/pull/1262)に正負の試験候補があるため、**#1119で同じファイルや第二の証跡検査器を作り直さない**。PR #1262はGo/Node実行とexact-head full Healthの提出が未済。未統合の修正案をmainに適用済みとして扱わない。

### 現在の実行手順の使用可能範囲

- `tools/reference-critical-path-evidence-audit/qualify.sh`は**PR #1135の作業ブランチ名を固定**しており、当該PRはmerge済みで旧branchはGitHubの検索上現存しない。現mainの事後資格や別PRの資格として、そのまま起動するのは適切でない。
- `compose-smoke.sh`も**#1119を含む未統合の4本のbranch tip**がremoteにあることを前提とする。PR #1135・#1139・#1137・#1141のmerge後には、その前提を満たさない。過去の合成試験の設計記録として保持するが、現在のmainの試験に成功したと称しない。
- 今後の全体の実行環境・資格は[公開後のCI整備 #1189](https://github.com/hat47x/tei-platform/issues/1189)を担当とし、既存`tools/repository-health-check.sh`を省略しない。#1119専用の並列CI/資格エンジンを追加しない。
- 独立のSource revision→Observation連結は#1163、受注の実WebとData-backed Function/保存は#1127、実DB SchemaとHost Authorityは#1126/#1121の既存責務を使う。独立した別readが無い限り、synthetic Command後のObservationを外部永続化の実証に数えない。

### 次に提出する最小の実験と判定

1. **E0の再資格**：current mainのclean full checkoutで、Goの既存`TestPracticalRuleChangePreviewsAndAcceptsBothMeaningAndApplicationSelection`、`TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList`、StudioのNode scratch E2E、および全`tools/repository-health-check.sh`を実行し、実行直前・直後のmain SHAとclean状態を記録する。旧branch前提の`qualify.sh`は使わない。
2. **E1の実接続**：Studioの2操作→Shared AcceptanceのReview／Decision→Source R0≠R1→reloaded Worldの4つのValidation→**実起動**Web Targetで正値と新規単価0の拒否（Command未実行）を、同じR1と起動したTarget Tへ結びつける。通常のAPIにテスト専用単価変更機構を混ぜず、隔離されたデータを使う。
3. **E2の独立結果**：R1からの現行Provider/Schema/Host認可適合を確認し、実DBへの効果と独立したread、非互換Schemaと権限失効の拒否を取得。E1 synthetic Webの成功から推論しない。
4. **G3/G4**：第二Targetの同じ意味定義に対する実差分と、採用したPlugin・Build/self-buildの安全な接続を別途確認する。先行するE1/E2の未達を帳消しにしない。

**判定**：G0は台帳整備が進んだがConsumer間の正式な確認待ち。G1〜G4は引き続き未合格。mainに実装が増えたことは有益だが、独立した現物証拠と正式な受入を得るまでIssue #1119をcloseしない。実行できなかった試験にPASSと書かない。

## 27. 現行mainで横断試験を実行する入口（2026-10-12）

### 統合前の合成試験を再利用し、現行mainを検査する

§25の`compose-smoke.sh`は、当初#1119/#1120/#1124/#1121の未統合ブランチを合成する方式であった。これらは2026-10-11にmainへ統合されており、旧ブランチを必須にしたままでは**最新mainを検証できない**。§26でその使用不能を指摘したため、既存の実行入口を**二つの明示モード**に分けた。

- `integrated-main`（今回追加）: **mainのexact commitだけ**を入力し、一時的なdetached worktreeで既存のStudio/Shared Acceptance/typed ref/Lowering/Web/Go証跡検査と全Repository Healthを実行する。旧Draft SHAは指定してはならない。
- `draft-tips`（従来互換）: current mainに未統合の4つの確定SHAを一時合成する。いまの#1119には基本的に不要。過去の旧branchが削除されている場合はfail-closedし、存在しないrefを補って合格したように扱わない。

現行mainへの実行例（**本セッションでは実行していない**）:

```bash
# canonical GitHub origin/mainのSHAを、実行環境で取得して指定する。
TEI_CP_COMPOSE_MODE=integrated-main \
TEI_CP_COMPOSE_MAIN="$(git rev-parse refs/remotes/origin/main)" \
bash tools/reference-critical-path-evidence-audit/compose-smoke.sh
```

両モードは正規のGitHub remoteを`fetch`し、mainのSHA固定、クリーン状態と非標準index検出、一時worktreeのHEAD/TREE、実行後のmain再照合、元checkoutの不変を要求する。**mainへmerge/pushしたり各Ownerのbranchを書き換えたりしない**。`integrated-main`は未存在の旧branchを参照せず、旧Draft pinが混入すれば拒否する。 実行入口となるスクリプト自身のGit blobを開始・終了時に照合し、成功表示には対象main／一時HEAD／tree／スクリプトと全体健全性検査スクリプトのblob／Go・Node版を併記する。**これらは後日の照合に使う来歴であって、第三者署名や改ざん防止の証明ではない**。

### 検証できる範囲、検証できない範囲

現在mainにある`live-r1-smoke.mjs`は、新規単価Validationをscratch R1へ受理し、4規則の正常系を実際に起動したWebへHTTP送信する**試験ソース**である。合成Providerの負例や実DBの永続化を自動的に満たしたわけではない。#1163の[Draft PR #1262](https://github.com/hat47x/tei-platform/pull/1262)には、**別バイナリ**とGo Overlayの隔離fixtureによる単価0のrunning-server負例を試すコードが追加されているが、これはmainに未統合でactual Web実行も未取得。今回のmain-modeは未統合PR #1262のコードを暗黙に試験へ混ぜない。

- 成功時の識別子は`CURRENT_MAIN_CROSS_LANE_SMOKE_ONLY_NO_PRODUCT_GATE`。**特定mainの横断回帰試験**であり、各PRのexact-head統合資格やG0〜G4の合格ではない。
- 実験はscratchでテスト用Decisionを生成する。**Human Acceptanceの代替ではない**。
- Command後の合成Observationは**独立した実Database readではない**。E2の保存・Host権限・Schema互換、第二Targetの合格には用いない。
- **変更範囲の独立試験（実施済み）**: GitHubの`compose-smoke.sh` blob `8173c4bd29001db812e3e140a79daa03d24b75c8`とローカルの試験対象を`git hash-object`で一致確認し、実際の`bash -n`はPASS。隔離Gitリポジトリで、不明モード、main SHA未指定・不正、旧Draft SHA混入、従来モードの必須pin未指定、正しい両モードの入力検査通過、不正remote拒否の**8ケースが全件PASS**。さらに隔離した実Gitの一時worktreeで`integrated-main`の開始から後片付けまで通し、試験用Go/Node処理を呼び出すこと、main HEAD・clean不変、一時worktree消去、runner/Health blob・Go/Node版の出力を確認した（実Gitは使用したが、Go・Studio・Web・Healthの中身は**テスト用代替処理**）。
- **製品側の試験（未実施）**: TEIの実Go/Node E2E、全`tools/repository-health-check.sh`、実Web HTTP、外部DB接続は、この環境で完全checkoutを入手できずNOT RUN。隔離試験の成功をこれらのPASSとして転記しない。
- 2026-10-12に更新された現行`AGENTS.md`は、全体健全性検査をすべてのPRの統合条件とはせず、変更範囲に応じた試験・静的解析・境界値・既存動作との照合、および未実施範囲・危険性の明示を要求する。`compose-smoke.sh`が**起動された場合に**全体健全性検査まで実施する設計であることと、**本PRの統合前に一律の全体健全性検査PASSを義務付けること**は区別する。Source/Target/DBの受入は別Gateであり、全て未達のままとする。

**次の作業の入口**: 新しい資格検査器を追加するのではなく、既存`compose-smoke.sh`の`integrated-main`でE0/E1-Aを実行し、実測結果を#1119と#1163へ提出する。その後、#1262の修正を独立に資格確認・統合してから必要なら同じ入口を再実行する。未実施のE1-B/E2/G3/G4を推定で埋めない。

## 28. GitHub Actionsで得られた実行証拠と次の停止条件（2026-10-12）

### E1-Aはmainで実行済み、E1-BとN11は別PRで実行済み

本節は§26〜27の**当時の「未実行」という記載を訂正する現時点の判定**である。古い手順・試験候補を消すのではなく、実行した版と合格範囲を一対一で明らかにする。

| 実験 | 検査対象の版 | 実行結果と根拠 | 受入上の限界 |
| --- | --- | --- | --- |
| E1-A（正の単価） | main `4b8bf628a91654ff19ab267c900699ac2b4b1e9f` | [Actions #38224540835](https://github.com/hat47x/tei-platform/actions/runs/38224540835) **SUCCESS**。実Goビルド・Webプロセスを使い、Source R0≠R1、4つのValidation、HTTP 200、Commandを確認 | Sourceの受理はtest harness、Observationは合成、単価0は未実施 |
| E1-A＋E1-B＋N11 | [#1163 / Draft PR #1262](https://github.com/hat47x/tei-platform/pull/1262) head `20f8453b451d5621d3ed6c37cc06e96b47295de9`、GitHub仮統合版 `b24216402b9e21f477e2c12c72df7a7ebc67319d` | [E1 Actions #38225770264](https://github.com/hat47x/tei-platform/actions/runs/38225770264) **SUCCESS**。正常注文、単価0の規則拒否とCommand/Observation未実行、N11のHTTP 422・Command結果あり・Observationなし。合わせて[標準Health #38225770258](https://github.com/hat47x/tei-platform/actions/runs/38225770258) **SUCCESS** | **mainに未統合**。正常系と二つの異常系は異なる試験用Goバイナリ。E2の実DBや人間受理は未実証 |
| 最新mainの全体CI | main `d324fd9127859c431ae53c78a90203474fa91850` | [Actions #38235176913](https://github.com/hat47x/tei-platform/actions/runs/38235176913) **FAIL**。Go vet、全Go tests、Workspaceは**PASS**。Go 1.23.2の`gofmt`で`tools/reference-capability-mutation-protocol/main.go`が不一致 | 同じパスの整形PR [#1311](https://github.com/hat47x/tei-platform/pull/1311)は一度merge済みだが、現行mainで再発。担当領域の原因確認が必要 |

E1-Aの実測Sourceは、`R0=sha256:f414e2b09d2f6ea5d0f5f282aea55612985c0e8155877ad8d310250633cf3fd2`、`R1=sha256:b78d5a186b1227f2bc4863ba2063c4f7edb5640c1d38faebaf2864c0423e3c07`。E1-Aのmain上Target binary SHA-256は`fc3efaf3aafdfbfa9a9a22ea85eef964e47ce587d448d681f91d81aa9fd96a8a`。いずれも**その試験実行で生じたexact値**であり、後のmainへ転用しない。

E1-Bの`validation-not-satisfied`は**業務上の拒否**でありHTTP 200となる。対してN11のHTTP 422は**実行後の観測失敗**で、Commandが取り消されたという意味ではない。三つのケースを同じ「エラー」へまとめない。N11を実DBによる独立したreadの成功・不成功の証拠へ昇格しない。

### 既存の実PostgreSQL証拠をE2へ誤流用しない

既存`tools/reference-postgres-header-detail-runtime`は、別の空の専用PostgreSQLに一時fixtureを作り、読み出し、後片付けを行い、Sourceと実DBの安全境界を記録できる。これは実Provider／SQLの**先行する有効な証拠**だが、E1の`sales.order_line.unit_price_positive`を追加して受理した**同じR1**から業務Commandを発火させた証拠ではない。`tools/reference-gate-b-final-evidence`の同一Repository改訂への収束も、**同一Accepted Source R1からのCommandと独立Observation**とは別である。

次に提出するE2の最小証拠は、#1127を実行owner、#1163を一改訂Evidence監査ownerとして次をつなぐ。

1. **Source**：明示受理されたR1（変更後のSemantic Validation＋Application採用）と、ターゲットをビルドした同じ意味契約の版／受理指紋。単なるGit SHAの一致だけでは代用しない。
2. **実行前**：既存Provider・Schema実態・契約版・権限のreadinessが`current`であること。空の専用DB／独立接続を使用し、既存DBを清掃して流用しない。権限失効S05、Schema非互換S02などはCommandを**発行する前**に拒否する。
3. **副作用**：**通常の製品Web／Driver**から選択された4規則に対する正常Commandを起動し、処理IDと実Providerのwrite結果を示す。テスト専用Go overlayバイナリで観測した単価0を通常Web実装の成功と呼ばない。
4. **独立Observation**：別のread invocation／DB connectionで、実DBの書込み後の状態とその対象（同じscope、key、時点、Database instance）を読んで検証。Command responseの再表示、同一メモリcache、合成Observationは不可。
5. **否定・未観測**：単価0はCommand非発行、S02/S05は事前拒否、N11ではCommand成功とObservation失敗を明確に区別し、永続性の「確認済み」を返さない。read未取得・Source相違・Provider相違は常に`not-observed`。
6. **記録と安全**：Repository/Source/Target/Provider/Schema/Authority各revision、Source→Command→readの相関ID、実行範囲、cleanupの責任者と結果、stdoutの機密漏洩防止を残す。DB関連SecretやDSN本文を証跡へ複製しない。

このE2は新しいDB Provider・第二のSchema Authority・第二の受理機構を作る指示ではない。既存のDB runtime、Web Driver、Shared Acceptance、#1163の検査を**一つの実験でつなぐ**。既存toolの個別成功を足して一往復の成功へ換算しない。

**親Gate判定**：E1-Aの技術的実証は**済**。E1-B＋N11の隔離実Web実証も**PR #1262内では済**だがmainに未反映であり、異なる合成バイナリの範囲に限定。G1の正式なAuthority接続、G2のE2実DB＋独立read、G3第二Target、G4 self-buildの横断的な資格はなお未合格。#1119のIssueはopen。

## 29. E1保存結果とSource版拘束の横断判定（2026-10-12、追補）

この節は§28の後に実測された版別結果を補う。**当時の成功・失敗を後のmainや別のPRに遡及適用しない**。合成fixtureのCI成功、受理済みSourceとの真正な結合、製品Gate通過はそれぞれ異なる。

| 確認事項 | 実際に確認した版・証拠 | ここからは言えないこと |
| --- | --- | --- |
| E1-Aのmain事後検査と証拠保存 | [PR #1331](https://github.com/hat47x/tei-platform/pull/1331) merge `4b390208acbea4c22b2fb688174f6f41c22d6b0d`、[main E1 run #38239334884](https://github.com/hat47x/tei-platform/actions/runs/38239334884)／[Health #38239334982](https://github.com/hat47x/tei-platform/actions/runs/38239334982)はSUCCESS。実Artifact `11741926281`のZIPと3 JSONのSHA-256照合を確認 | これだけでは単価0、N11、R2、実DB、人的受理は未検証 |
| E1-A／E1-B／N11とSource R0・R1・R2の観測 | [#1163 Draft PR #1262](https://github.com/hat47x/tei-platform/pull/1262) head `ef0ea673a988536558970cae86f00672f0599b8a`、仮統合 `dd8f7524062dcff18c15c9c356c03746461a1913`、[E1 run #38243337087](https://github.com/hat47x/tei-platform/actions/runs/38243337087)・[Health #38243337100](https://github.com/hat47x/tei-platform/actions/runs/38243337100)は双方SUCCESS。Artifact `11744141694`の8 JSON原本が目録内SHA-256と全件一致 | PRはmain未統合。受理は試験用、正常・負例は異なる合成Webバイナリ、Host Authorityと実DB書込みは未実証 |
| 8件の証拠契約へSource R2差異の検査を組込む | [PR #1395](https://github.com/hat47x/tei-platform/pull/1395) head `0954be7d5e49d26572dcb73b1b54c6d9cf1c81b3`、[E1 #38246643033](https://github.com/hat47x/tei-platform/actions/runs/38246643033)・[Health #38246643056](https://github.com/hat47x/tei-platform/actions/runs/38246643056)はSUCCESS、merge `9b385961ab0c7744184e742d95aff91931985091` | この二つのCIは**正常系3件モード**。#1395の変更後のワークフローで**8件モードを実行したPASSは未取得** |

### 製品上の反例は二つあり、同じ試験ではない

- **旧R0の意味不一致**：受理済みR1というラベルを与えながら実際にはR0をWebへ読み込ませると、新単価規則の欠落とManifestの意味差分を外部ハーネスが見つける。ただし実Webは自ら拒否していない。
- **R2の意味同値・Source不一致**：R1のSourceへ空行を加えたR2は、元のSource bytesとrevisionが異なるにもかかわらず、派生Worldと実HTTP Manifestの意味SHAがR1と同一。**意味指紋一致・B/R/Tのラベル一致は、実際に読み込んだAccepted Source bytesの同一性を保証しない**。

両者をE1で記録する際、外部照合で差を検知できた事実をTarget内部の事前拒否へ読み替えない。R1の実バイトを呼出し・起動・Command発行の正しい時点で拘束する実装責任は[#1121](https://github.com/hat47x/tei-platform/issues/1121)とSource Readerの既存契約にあり、#1119が第二のSource正本や認可判定器を作らない。

### 次に取るべき実測と役割

1. **#1163／#1119のE1資格照合**：#1395の修正を含むワークフローを用い、#1262の8件生成を**新しいGitHub仮統合版**で実行する。新旧のrun ID、PR head、仮統合SHA、Artifact IDを固定する。ZIP実体を再取得し、GitHub記載のZIP SHA-256、目録、全8 JSONのraw byte SHA-256、R0≠R1≠R2とR1/R2のWorld・Manifest同値、Runtime未拒否という否定フラグを照合する。旧run #38243337087の成功を新workflowの成功へ転記しない。
2. **#1121の実拘束**：Web/Driver自身が、実際に開いたManaged Source bytesを明示受理されたexact R1と照合し、不一致ならCommand前にfail closedする。R0の意味差分とR2の意味同値を双方負例に含める。受理した当事者・権限・改訂の真正性は別途証明する。
3. **#1127の実DB、#1163の一改訂Evidence**：[Draft #1399](https://github.com/hat47x/tei-platform/pull/1399)の手書きoverlay→PostgreSQL write／別接続readは独立した先行試験であり、#1119のAccepted R1から通常Web Commandへつながった証拠ではない。[#1406](https://github.com/hat47x/tei-platform/pull/1406)の保存宣言の静的照合も書込み権限・実Providerの成功ではない。これらを既存契約内で収束させ、実DB write→別invocation read、S02/S05拒否、N11の未観測を区別する。

**資格判定**：G0はOwner間の正式なConsumer受入を一部未確認。G1はRuntime側の受理済みSource版拘束が欠け、G2は同じR1からの実Web→実DB→独立readが未取得。G3第二Target、G4自己構築・横断受入も未完了。E1の合成証拠は進展したが、製品Gateは **UNQUALIFIED**、[#1119](https://github.com/hat47x/tei-platform/issues/1119)はopenを維持する。短期Artifactは7日で失効するため、期限後に現物の再取得が必要なら同一版の証拠を再生成し、再実行版を記録する。

## 30. 変更集中と影響範囲による優先順位の更新（2026-10-12）

§9のP0/P1仮説を、現在確認できる変更履歴と製品上の失敗例で再評価する。ここでの優先順位は**契約が下流を止める危険性**であり、担当者の成果や実装行数の順位ではない。§29の版別実証が根拠であり、既に閉じた#1120や#1132を再起票しない。

### 観測に用いた変更頻度と限界

GitHubの既定ブランチについて、`2026-10-10T00:00:00Z`〜`2026-10-12T14:33:00Z`の`commits?path=...`結果を数えた。作業時点のmain参照値は`9478b5fac91b24d4620f8fcddb933052d5c44ad2`。ディレクトリの値は配下全体であり、**コミット数は契約の破壊件数でも、変更が失敗した回数でもない**。同一コミットは複数行に重複する。Draft PRの未統合変更はここに入らない。マージ後のmainが進めば、これらは固定時点の参考値となる。

| 観測した領域 | 対象期間のmain上の該当コミット数 | 読み方 |
| --- | ---: | --- |
| `internal/modelref/`（C0） | 69 | 下流の意味参照に波及し得るが、現行#1120の完了判定を取り消さない |
| `tools/reference-shared-acceptance/`・`reference/studio-authoring/`（C1） | 65・58 | 受理と編集の接合面は変更が多く、実Consumerでの照合が必要 |
| `tools/reference-first-web-target/`（C3/Web） | 182 | もっとも変更が集中。Source版を実Webが拒否しないという実測が優先度の主根拠 |
| `tools/reference-critical-path-evidence-audit/`・`tei-critical-path-e1-ci.yml`（E1証拠） | 52・6 | 検査の変更と製品本体の修正を同一視しない |
| `internal/plugin/`・`tools/reference-standard-function-candidates/`（構成・Data-first） | 43・3 | Plugin変更は広く波及し得るが、標準機能の現行不足は静的候補だけでは埋まらない |
| `tools/reference-data-first-persistence/`（実DBの先行試験） | 0 | **Draft #1399の実装不存在という意味ではない**。main未統合のため、計数対象外 |
| `tools/reference-subsystem-cud/`・`internal/sourceindex/`（C2・出所索引） | 36・3 | 独立した出所照合が進展。正式Accepted Businessの根拠とは区別 |

`reference-first-web-target`はAPIの100件上限を越えたため2ページ目82件を加算した。ほかは初回ページで100件未満。対象範囲はGitHubのpath条件に依存し、開発者の実作業時間、変更の重大度、並行する作業ブランチの進捗は測っていない。

### 優先順位：変更頻度よりも、依存と安全な代替可否を重く見る

| 順位 | 重点課題と担当 | 下流・逆方向依存の評価 | 代替可能性と停止条件 |
| --- | --- | --- | --- |
| **P0-1** | **受理済みSource R1を実Web自身が拘束**。#1121・#1017、検証#1163 | C1受理→C3実行の境界。#1124・#1127・#1163が下流。B/R/TやWorldの値を呼出元が指定するだけでは受理の根拠が逆流する | **安全な代替なし**。R0と意味同値R2の両方で実行前に拒否し、R1正常系を維持するまでG1未合格 |
| **P0-2** | **同じ受理済みR1から実Web Command→実DB write→別接続・別invocation read**。#1127・#1163 | C3、Provider/DB、Host Authority、Observationを接ぐG2の最終阻害。静的Persistence候補だけで実行権限を逆流させてはならない | **安全な代替なし**。実DB先行試験や合成Observationは部分証拠。S02/S05とN11を含む一連の実験までG2未合格 |
| **P0-3** | **#1395後の8件E1証拠保存を現行PR仮統合版で再実測**。#1163、保存契約#1119 | E1監査はC1/C3のConsumer検査であり、製品本体のAuthorityではない。#1262のDraftを直接変更しない | 旧Artifactは回帰の参考にはなるが**新workflow版の実行証拠の代替にならない**。PR head、仮統合版、原本のハッシュを一対一で固定 |
| **P0-4** | **三視点のAccepted Sourceへの接合・正式Consumer確認**。#1124・#1125・#1126、参照#1120、C2 #1132 | C0の識別・出所からC1の変更受理、業務・Data・FunctionのConsumerへ扇状に依存。Scope/Kindや候補と正式受理の混同を避ける | Read-only図や合成CUD候補は部分代替。業務手順の正式受理、再投影、権限のConsumer負例を取得するまで「三視点完成」は未認定 |
| **P0-維持** | **Plugin/Capabilityの構成・禁止依存**。#951・#1016・#980・#1019 | CP-2は多数Targetへの影響を持つ。ただし具体的な禁止importは既存#980の検査が主担当で、#1119から第二検査器を作らない | 当面は既存Capability/Driver契約でP0業務一往復を進める。新たな逆方向依存が実際に検出されたら上記P0との衝突を再判定 |
| **P1（並走）** | 第二Target、自己構築、取消・通知・帳票・シナリオ（#1129〜#1131ほか） | G3/G4と製品拡張には必要だが、最初の一つの実Web/DB一往復を直列に止めない | 代替できるという意味ではない。別のGate・ケースで固有の実証を維持 |

優先順位を上げる条件は、**同一版・同一主体・同一実行対象**で矛盾が再現すること、または既存の禁止依存検査が具体的な違反を報告すること。件数だけでCore化、Enum追加、全体再構築、第二のOwnerによる重複実装を認めない。

### 次の最小受入と責務

1. **最初に#1163**は、#1395統合後のworkflowと#1262の8件生成を同じPR仮統合版で動かした実行ID・Artifactを報告する。未取得なら`NOT_RUN`を維持。合成試験の合格をG1/G2へ昇格しない。
2. **#1121**は、受理Decisionの実revisionと実際に開いたManaged Sourceを突合し、R0/R2のCommand前拒否とR1成功の正負試験を担当する。#1119ではWeb Loaderを改造しない。
3. **#1127**は、Provider/DB/Hostの実態と権限を読み戻したうえで、正常Command→独立readの同一R1相関を実証する。#1163が証拠の版と由来を監査する。
4. **#980/#1019**が既存の依存検査と拡張シナリオを維持し、#1119は具体的な違反・改修がP0製品ループへ及ぼす影響だけを確認する。

**判定**：最初の作業一覧にあったP0/P1評価について、変更集中・下流・逆依存・代替可否を実測可能な範囲で整理した。これは優先順位の評価を進めたという意味であり、正式Consumer資格、G1〜G4の製品GateやCIの新たなPASSではない。

## 31. G0の実Consumer検証と実PostgreSQL先行実証を踏まえた接合順（2026-10-13）

この節は§30以後の**実行済み個別証拠**と、まだ未取得の**一つの製品横断証拠**を分離する。旧節の時点別判定は遡及変更しない。#1119のG0〜G4、#1163の一改訂Evidence、Shared Acceptanceの判断権限、#1127の実行責務は既存のまま。

### 統合・検証の現在地点

| 接合点 | 実測と根拠 | 残る未保証 |
| --- | --- | --- |
| Studio → Shared Acceptance（G0 Consumer） | 現行mainだけではRelation編集候補のflatな`add-relation`がGoの`addRelation`構造に合わず`decode=fail`。修正を含む[Draft #1273](https://github.com/hat47x/tei-platform/pull/1273)を組み込んだ[検証専用PR #1558](https://github.com/hat47x/tei-platform/pull/1558)の[実Go/Node横断試験 #38315568392](https://github.com/hat47x/tei-platform/actions/runs/38315568392)は成功 | #1273本体のmain統合、正式な人的受理、Businessを含む三視点の同一Accepted版 |
| Web R1正常系・R0/R2負例 | [E1合成Web原本と来歴](https://github.com/hat47x/tei-platform/issues/1119#issuecomment-6133338346)を保存。R0では意味が異なり、R2ではSource bytesが異なる一方でWorld/Manifestの意味は同値 | 差異を検出したのは外部ハーネス。Web自身による起動／dispatch前拒否は未実証 |
| 実DB保存・別接続読取 | 手書き`intake.request` overlay/SQLによるPostgreSQL16のwrite→同一プロセス別Poolのreadが[実CI #38311557882](https://github.com/hat47x/tei-platform/actions/runs/38311557882)で成功。原本は[PR #1549](https://github.com/hat47x/tei-platform/pull/1549)でmainに保全 | 正式Accepted `order_entry` R1・通常Web・実Host権限・別invocation read・Data-first生成CRUD |
| 証拠誤結合の拒否 | [PR #1555](https://github.com/hat47x/tei-platform/pull/1555)は上記二つの旧原本を`UNJOINABLE_HISTORICAL_EVIDENCE`と判定する | 新たな同一改訂・同一業務操作によるG2のProduct証拠は未取得 |

G0に関して、**仮統合版の正常系成功**を**main単独の統合完了**と表記しない。現時点のG1〜G4は`UNQUALIFIED`。

### 最小依存関係と並列実装の境界

1. **G0: #1124**は、Draft #1273の候補→Shared Acceptance Consumerの変換を、最新mainとの合流と正負テストで確認して統合判断する。#1119はその修正を複製しない。Scratch用自動受理と人間の権限ある受理は別の証拠。
2. **G1: #1121（#1017と連携）**は、`tools/reference-first-web-target/main.go`の`workspace.Load`で**実際に開いた**管理対象Sourceのmembership・相対path・fingerprintを、信頼できる既存Shared Acceptanceが承認した**exact R1**と起動／Command前に照合する。呼出元のB/R/Tラベル、Decision JSONの`accepted`文字列、World/Manifestの意味DigestからAuthorityを生成しない。R1を正常に通し、R0と意味同値R2をWeb自身が拒否する。既存Source Indexと受理側の版算法を再利用し、第二の版正本を作らない。Source読込のTOCTOU、起動後の変更、実Host権限は別の安全境界。[具体的な受渡し条件](https://github.com/hat47x/tei-platform/issues/1121#issuecomment-6133639725)を参照。
3. **G2: #1127**は、#1406の保存先静的照合（`executable=false`）と#1399の手書きDB先行試験を、そのまま書込み権限や生成済みCRUDとみなさない。既存Web/DriverとProvider/Schema/Host認可の実態を、同じ正式Accepted `order_entry` R1から一本の保存Operationへ接続し、**別の実行プロセス／invocation**から実業務rowを読んで結果を確認する。
4. **一改訂Packet: #1163**は、Source membershipと版、受理根拠、業務SubjectとOperation、実Web build、Host認可主体／policy版、DB instance／Schema、Command ID／保存key、別invocation Observationを**同じ新しい試験**から記録する。#1555は旧原本の保全照合であり、将来の正式なG2判定器に再用途化しない。

G0/G1/G2は**全面的な直列開発を要求しない**。DB fixtureとProvider事前検証はStudio/Source拘束と並列に進められる。ただし正式Accepted R1→通常Web→実PostgreSQL→別invocation読取を製品資格とするのは、同じ実行でG1とHost権限を確認した後に限る。第二Target・完全なPlugin dependency solver・self-buildをこの一本の前提へ追加しない。

### #1163へ引き渡す最小受入マトリクス

| ケース | 確認すべき事実 | 判定の境界 |
| --- | --- | --- |
| R1正常保存 | Accepted managed Source集合、Webの実読取版、現在のHost認可、Command、実DB row、別invocation readの相関 | 同じApplication/Subject/Operation/key/revisionを実証できた範囲のみ`observed` |
| R0未受理 | CallerのR1ラベルに対しWeb LoaderがR0を読んだ | Web自身がdispatch前拒否。外部ハーネスだけの不一致検出は未合格 |
| R2意味同値 | Source bytesと版だけR1と異なり、World/Manifest意味Digestは同じ | exact R1を要求したWeb自身がdispatch前拒否 |
| S02 | Source revision不一致と物理Schema/Contract不一致を別々に記録 | 原因を混同せず、該当する不一致を実行前に拒否 |
| S05 | Caller/subject/policy/current effective authority、失効時dispatch0 | Hostの現在の認可で拒否。モデル宣言は実権限でない |
| N11 | Commandが実DBを更新しても後続readが失敗・不一致 | Command完了とObservation成功を分離し、`observed`を返さない |

この行列は第二のEvidence Schemaではなく#1163既存Packetの実験入力。Source/Accepted/Target/Provider/Authority/Command/Observationの個別証拠と、実行版・run URL・原本checksum・未実行理由を保持する。実行していない列は`unresolved`／`not-executed`、Secret・生の接続文字列は保存しない。

**停止線**：この節の追加は実装・CI・正式人的受理・G1〜G4の新規PASSを意味しない。対象変更の範囲とexact headに応じて現行`AGENTS.md`の検証規則を適用し、全PRへの全体健全性確認の一律義務化を避ける。

## 32. X-TEA包含 #8：初回read-onlyと更新系の証拠・依存を分離する（2026-10-25）

本節は[#1119の#8統合記録](https://github.com/hat47x/tei-platform/issues/1119#issuecomment-6386491091)と[#1802の既存機能対応表](../design/AUTHORING_TOOL_FEATURE_TRANSFER_MAP_2026-10.md)への**最小追補候補**。新Gate・Reader・Host・二重の機能台帳を定義しない。既存のG0〜G4や§31の受入条件を書き換えず、**参照モデル転写の最初の実行slice**を[#1142](https://github.com/hat47x/tei-platform/issues/1142)の自作synthetic read-onlyに限定する。未受理の原典更新を初回成立条件に戻さない。

### Source / Candidate / Accepted と証拠段階

- **Source**：X-TEA原典の固定版・構造・宣言・旧Driver静的コードと、別に取得した原Runtime観測。原典の静的事実と実行観測は別証拠。非公開XML/Script本文/業務データ/個別ID/画面は[#941](https://github.com/hat47x/tei-platform/issues/941)に従いpublicへ出さない。
- **Candidate**：Readerによる出典付きfact、Data/Function/Business一モデルの対応案、UI/Script/Driverの実現候補、独立loss ledger・candidate receipt。名称・Key一致・宣言CRUDから業務identityや副作用を導出しない。
- **Accepted**：対象範囲、根拠版、責任ある判断者、受理内容を固定した意味の受理。**試験用scratch Acceptance、CI成功、旧GUI操作成功、Source Factは原典意味のOwner Acceptanceではない**。
- 証拠は code-located / CI-success-at-exact-head / synthetic-executed / original-GUI-observed / real-DB-observed / owner-accepted / independently-compared を各々記録する。一つのPASSは他段階へ昇格しない。未読担当はunconfirmed。

### S0〜S5：最小引渡しと着手条件（全面直列化しない）

| 段階 | Producer → Consumer | 最小成果・着手条件 | 未成立の境界 |
| --- | --- | --- | --- |
| S0 Source／包含範囲 | #1 → #3 → #2；#1802参照 | 元版/digest、node・attribute・text・関係の処遇、loss ledgerとcandidate bytes receipt。Windows不要 | inventory PASS≠意味移植 |
| S1 **初回synthetic read-only** | #2/#3 → #7；#5はUI/Interaction観測 | 既存merged [#1144](https://github.com/hat47x/tei-platform/pull/1144)のsynthetic testを再利用。Candidate→試験用Decision→reload→Driver browse→mapped read、正負Validation、store不変 | 原典の人的受理、原実機、実DB、等価性を要求・主張しない |
| S2 原Driverの独立実動 | #4/#5 → #W/#940 → #3 | 固定配布版と隔離モデル/DBでUI、Script/SQL、例外、DB前後、独立readを取得 | read-only表示でもScript副作用が未観測なら副作用なしは未確認 |
| S3 原典意味の受理 | #2/#6/#3 → Owner → #7 | identity、業務Action、拒否/取消、Script効果/例外を対象版ごとに受理または保留 | 未受理の原典更新系を正式実装しない。低位機構の合成調査は並走可能 |
| S4 更新・Commit・結果伝播 | #6/#7 + #4 → #1129/#1174/#1127 | **受理した対象のみ**、既存Target環境の認可/Provider/Transactionへ接続。必要時#1708で鮮度確認。Commit unknown、Observation unknown、呼出前/後取消を分離 | 新TEI独自Host、Runner、Capability文法を開始条件にしない |
| S5 全件処遇・比較 | #3/#8 → Owner | 全Source要素と機能種別の処遇、旧/TEI別run、権利・未観測・差異を照合 | 一件の合成PASS≠全機能移植・原典動作等価 |

**必須否定例**：宣言CRUDからOwner Actionの実効果を推定しない／旧実動証拠と前提一致なしでequivalentとしない／pre-dispatch取消を呼出後rollbackと混同しない／Commit不明と正常Commit後Observation失敗を混同して再送しない／実行までにSource・Binding・Schema・Authorityが安全性に関係して変化したら再検査またはfail-closed。実行責務は既存#267/#1121/#1129/#1174/#1708/#1142へ返す。

### 同時進行Draftの合流前提と証拠の範囲

| Draft PR | 当該担当の修正 | 合流で検証する反例 |
| --- | --- | --- |
| [#2219](https://github.com/hat47x/tei-platform/pull/2219) (#1) | Source Relationship種別をsource factへ保持 | 種別文字列をTEI Relation意味/Acceptedに自動昇格しない |
| [#2220](https://github.com/hat47x/tei-platform/pull/2220) (#4) | **正常Submit後のObservation失敗**を結果未確認とする | read失敗を書込失敗確定としない |
| [#2236](https://github.com/hat47x/tei-platform/pull/2236) (#7) | **Commit応答喪失**のAdapter/HTTP結果分類。#2220と同じcommand_scenario.goの別hunk | Commit不明を失敗確定・自動再送可能としない。両PR合流後に正負試験 |
| [#2251](https://github.com/hat47x/tei-platform/pull/2251) (#6) | Capability呼出直前の取消済みContextでdispatchを止める | dispatch 0回は呼出後副作用不在やrollbackを保証しない |

各PRはDraft/Open。PR exact-headでのRepository Health成功はその版に対するコード・合成試験の資格であって、4本の統合版、実DB、Windows原典Runtime、Owner受理の証明ではない。main進行・head更新後はrunを再取得する。#8から他担当の差分を変更・マージしない。

### #Wへの確認仕様とOwner未決

- **対象/操作**：既存[#2152](https://github.com/hat47x/tei-platform/issues/2152)の制御機構を使い、固定Modeler/Editor/Driverでread-only一覧・詳細をまず操作する。Menu/Option、Script発火・例外、SQL、UI結果、DB before/after、別接続readを記録。更新/取消/失敗は別の明示ケース。Editor追加削除の制御面は[#2199](https://github.com/hat47x/tei-platform/issues/2199)に委譲。
- **隔離/復旧**：原本はread-only、モデル/DBは使い捨て複製、loopback接続・試験主体を使用。接続停止→プロセス終了→残接続閉鎖→ベースラインから複製を再作成→digest再照合。確定不明を成功・失敗のいずれにも丸めない。
- **未決Owner**：原典のidentity・業務Action・Scriptの効果と失敗、拒否・取消と認可、権利・公開許諾。既存ADR-0020/0029と#1129/#1174で表せない具体的反例が得られる場合のみ代案・根拠・影響先をProposed ADR候補へ送り、Acceptedとしない。

**検証状態**：本節は固定main・既存ソース・Issue/PR・GitHub Actionsを読み取った計画追補。Go試験、trusted full health、実原典Candidateの全件検収、Windows GUI、実DB、原Driver/TEI実行比較、人的Owner受理を本節の作業で実施していない。

## 34. 2026-11-02 横断追補：読める業務設計からClaim・操作候補・成果物へつなぐ最短一往復（計画候補）

**位置づけ**：§1〜32の当時の記録とG0〜G4を変更せず、現在のM3/M4と既存C0〜C4の**検収対象を具体化する調整票**。新しいGate、第二の意味正本、共通Host/Runner/Registry、Proof Engineを作らない。照合mainは `9c9ef4d160640908e06ad13afb6c2b45c664c748`（2026-11-02）。Issueの進行に伴う本節の「現在」は固定版に限定する。

### 最新の成立事項・未成立事項

- [#2484](https://github.com/hat47x/tei-platform/issues/2484)のD-1〜D-3の採用判断は[merged #2588](https://github.com/hat47x/tei-platform/pull/2588)で既存RQ-019/020/030、M3/M4/M9、C4へ反映済み。標準操作の導出・束縛実行は到達条件とし、**無人実行は宣言と拒否試験まで**。D-4の#982「締め」追加は保留。#2144の期間意味モデルA4採用や実Provider稼働を#2588の文書統合から推定しない。
- [#2578](https://github.com/hat47x/tei-platform/pull/2578)／[#2580](https://github.com/hat47x/tei-platform/pull/2580)／[#2581](https://github.com/hat47x/tei-platform/pull/2581)はmerged。ただし投影のJSON化で十進値の`float64`経由の精度喪失、optionalのmissing/null同一省略などがあり得る。実消費値の忠実度は[#2585](https://github.com/hat47x/tei-platform/issues/2585)の別検査であり、生成成功≠型付き意味保持。
- [#1803](https://github.com/hat47x/tei-platform/issues/1803)の既存設計出力はclose済みの履歴を再利用する。[PR #2587](https://github.com/hat47x/tei-platform/pull/2587)の複数設計成果物は**指定時点でOpenの候補**。ファイル一式の現在の公開可能性とConsumerが読んだ版の一致は[#2604](https://github.com/hat47x/tei-platform/issues/2604)の独立課題。manifestの`complete`や作成成功だけで公開資格にしない。
- 原Driverの限定観測（merged [#2154](https://github.com/hat47x/tei-platform/pull/2154)）・一改訂実PG観測（[#1163](https://github.com/hat47x/tei-platform/issues/1163#issuecomment-6415151414)）・test-only Driver→Runner（merged [#2273](https://github.com/hat47x/tei-platform/pull/2273)）は別条件・別実行。合成Acceptance/Grant、業務Owner意味、Target実許可、正式Operation Receiptを一本の製品G2証拠へ合成しない。

### 最小の「設計が読める」一往復（既存fixtureとOwnerを再利用）

次の一つの**架空・合成**Fixtureを各担当で共用する。親H、独立した明細L1=100・L2=100、配送履歴3行。Data/Function/Businessの一モデルを正本とし、説明文・Query計算結果・CUI表示・SHO文書・Operation候補は**別Projection／派生成果物**として扱う。受注確定条件や通貨・単位・全明細規則は未受理なら仮定であり、原典SourceやOwner Acceptedとは表示しない。

| 順序・既存Owner | この一往復で実際に照合するもの | 成功と取り違えないもの |
| --- | --- | --- |
| P0-A：Source/意味・[#1125](https://github.com/hat47x/tei-platform/issues/1125)／[#1063](https://github.com/hat47x/tei-platform/issues/1063) | 設計書に対象Identity、行集合、規則適用範囲（先頭／編集中／全件）、条件、実行時点、未評価箇所を読める形で記す。現行三規則の**合成正常系**は存在するが、同経路の負例・全明細被覆・Owner意味を別欄にする | 二規則Gateの旧記録を「三規則不存在」へ戻すこと、試験用Commandを正式Actionと呼ぶこと |
| P0-B：Query→Context→Claim・[#2506](https://github.com/hat47x/tei-platform/issues/2506)／[#2586](https://github.com/hat47x/tei-platform/issues/2586)／[#2593](https://github.com/hat47x/tei-platform/issues/2593) | 明細Identityは2件、JOIN結果は6行、明細の正しいSUMは200、素朴JOIN SUMは600、SUM(DISTINCT amount)は100。取得済Scope/Grain/Completenessと、全称・存在・部分集計の**Claimごとの片側結論**を独立Expectedで比較。閉じたfixtureの0件は通常結果、既知の部分取得から母集団全体の不存在は導かない | Query列と業務Entity、集計行と更新対象の同一視、全Provider一律の完全性証明やLeanの必須化 |
| P0-C：投影→表示・[#2497](https://github.com/hat47x/tei-platform/issues/2497)／[#2585](https://github.com/hat47x/tei-platform/issues/2585)／[#2516](https://github.com/hat47x/tei-platform/issues/2516) | Missing／明示NULL／値0／未取得、精度が異なる数値、同値に見える集計値について**actual Consumer byte／適用Claim／不明な情報**を並べる。Read-only表示可能、編集可、外部開示可を用途別に分ける | `exit=0`・JSON妥当・図が読めることを意味保存やOwner受理と見なすこと |
| P1-A：編集・Request・[#2597](https://github.com/hat47x/tei-platform/issues/2597)／[#2598](https://github.com/hat47x/tei-platform/issues/2598)／[#1174](https://github.com/hat47x/tei-platform/issues/1174) | 同一Functionの複数Instance、選択epoch、Dirty Patch、保存Intent、正式Requestの対象／時点／Version／OperationIDを区別。Read時点とCommit時点で対象集合が変化する反例、確定不明後の無条件再送拒否を既存経路に照合 | 同じUI「保存」から全件atomicity、候補IntentからHost実行許可、状態再読取から同一OperationのReceipt回収を推定すること |
| P1-B：設計成果物→SHO／公開・[#1803](https://github.com/hat47x/tei-platform/issues/1803)／[#2516](https://github.com/hat47x/tei-platform/issues/2516)／[#2604](https://github.com/hat47x/tei-platform/issues/2604) | モデル正本、文書Artifact、Document Occurrence、表示時依存、説明・Review、実公開ファイル集合を区別。生成時の内部整合と、Consumerから見た**一式の世代一致**を別証拠にする。MMIの保存後Source再解決C-01/T2は別レーンで未完と扱う | 文書byte一致から説明妥当性、manifest完整から公開成功、SHO説明修正からTEI意味変更を推論すること |

**評価単位**：上表の各対象について、実際に固定したSource/Meaning版、Query/Context/Snapshot/Consumer版、入力と出力の実byte、独立Expected、Ownerが受理した意味、実行した／しない手順、残余Unknownを**既存Issue・試験票の参照**として追う。単一の新しい全体receipt schemaに転記せず、必要な接続辺がない場合はそのClaimのみ`unverified`とする。グラフ・一覧の閲覧可能性は全体のOwner意味受理を要求しないが、操作・公開・実効果へ進む資格は用途別に別確認する。

### 実施順と停止線

1. **最初の一作業はP0-Bのread-only縦断**：[#2586](https://github.com/hat47x/tei-platform/issues/2586)がHJ-01〜07を同一fixture／版で観測し、#982は利用価値と受入比較だけを担当。Query・Data Context・Claim・Patternを第二の正本へ再実装しない。必要な未接続辺を1件特定できれば成果とする。
2. 次にP0-A/Cで対象範囲と投影の非可逆性を検査し、業務設計書の読者が「評価済み／条件付き／未評価」を見分けられるかを比較する。テキスト可読性を満たす前にGUI、Lean、専用Runtimeを開始条件にしない。
3. P1-A/Bは独立並列。編集Intentの実行と文書一式の安全な公開は**異なる副作用**なので各Ownerが対象環境の現行権限と独立観測を検証。SHO Composerは独立執筆のDocument/Reviewを所有し、MMIは抽出・Source/Provenanceを所有する。TEIの意味正本を複製しない。
4. **保留・Owner判断**：全明細と編集対象の範囲、NULL／0の業務用途、Measureの再集計、受注確定／取消／電話確認の業務意味、期間のA4意味採用、外部開示と実行時Authority。[#2484](https://github.com/hat47x/tei-platform/issues/2484) D-4は現行の保留を尊重。#2226のADR-0041番号衝突の採否・改番はこの追補で変更しない。

**証拠状態**：本節は固定GitHub版と既存Issueの静的照合に基づく**調整提案**。Go/SQL/PG/Windows/GUI/外部Harness/SHOの新規実行はしておらず、実証済み個別結果を本節自身の全経路PASSとしない。原典の非公開本文・ID・locatorは含めない。
## 33. Studio Consumer P0とHeadless-first E0〜E4の接合（2026-11-02）

この節は[#1124](https://github.com/hat47x/tei-platform/issues/1124)の**Studio固有のP0〜P2**と、[#1180](https://github.com/hat47x/tei-platform/issues/1180)・[#2391](https://github.com/hat47x/tei-platform/issues/2391)の**全Target向けE0〜E4**を別軸の計画として接続する。既存G0〜G4・S01/S02やOwnerの受理基準を変更せず、本文の過去の時点別判定も遡及変更しない。

- **方針**：TEI独自共通Hostは必須ではなく、StudioはTEIのAuthoring / Observation Workbenchである。Wails/WebView・Theiaは必須アーキテクチャではなく、過去のPoC A/B/Cは固定Targetでの限定証拠。Host非必須化を文書化する[Draft #2226](https://github.com/hat47x/tei-platform/pull/2226)、ADR改番の[Draft #2426](https://github.com/hat47x/tei-platform/pull/2426)と[#2341](https://github.com/hat47x/tei-platform/issues/2341)は**別の文書・Owner手続きレーン**。これらのmergeをStudio P0の先行条件としない。
- **照合版**：main [`3784a4432a28502ca04057325689e409d14d9140`](https://github.com/hat47x/tei-platform/commit/3784a4432a28502ca04057325689e409d14d9140)、[PR1 Draft #2583](https://github.com/hat47x/tei-platform/pull/2583)の当初報告head `dc7b44750279d8f9e4c4e4b797c054b865021c1a`（後続更新あり）。PRとmainの資格を混ぜない。新たな実行・Owner採用は本節では**NOT RUN**。

### Studio内の依存順（PR番号は分割単位であって既存PRの番号ではない）

| 順序・Owner | 実装単位と現在地 | Product/Consumer受入に残す条件 |
| --- | --- | --- |
| **P0 / PR1 — #1124** | Catalogのglobal `ambiguous`を保持し、共有`inspect()`の入口でscope未確定のunqualified Inspectを拒否する。Draft #2583は3 source pathの変更とNode 18件・型検査・buildの限定PASSを報告 | #2583の埋込みWorkbench `main.js`が生成bundleと不一致、`connect-workbench.mjs`の旧exact source pinで`workbench_revision_mismatch`。正規bundle同期と新snapshotの**資格付きpin更新**が必要。pinを無条件に緩めない。PR1成功を4入口のPASSと呼ばない |
| **P0 / PR2 — #1124** | Graph／一覧／初期自動選択／Commandの**4入口**を共通guardに結線する。Go Catalogの曖昧性、候補の表示、診断保持を利用する | 実イベントによる同kind/ID・異scope、global ambiguity、旧応答を負例として`InspectSubject`の誤呼出し0件を各入口で観測。共有関数へのguard配置だけでは未達 |
| **P0 / PR3 — #1124** | Graphの描画snapshot／view ID→qualified Key対応とselection epochを結ぶ。[Draft #2457](https://github.com/hat47x/tei-platform/pull/2457)は独立したread-only参考実装で、PR1・PR2の代替ではない | G2更新拒否でG1の旧DOMが残っても、R10→R11後の旧選択からInspect 0件、新map M2をG1に公開しない。実DOMとmapの**同一snapshot**、描画成功・失敗双方の結合証拠が必要 |
| **P1 — #1124** | [merged #2293](https://github.com/hat47x/tei-platform/pull/2293)・[#2294](https://github.com/hat47x/tei-platform/pull/2294)・[#2348](https://github.com/hat47x/tei-platform/pull/2348)のConsumer freshness設計を実接続 | Source変更で旧schema/Inspect/readinessを失効。非ゼロ終了でも`accepted + sourceChanged + DECISION_RECORD_FAILED`ならSourceとledgerを別々に再読取、再Accept自動実行0件。S01は不要migrationなし、S02は互換性証拠なしの誤Readyなし |
| **P2 — #1124 / #267等再利用** | Data/Function/Businessの編集案→Preview/Validation/Impact→既存Shared Acceptance→再投影とObservationを一往復 | StudioはSource writer・Owner意味受理・Runtime Authorityの第二正本にならない。選択S／実Invocation対象T／確定Effect対象E、Actual Observationを別々に示す。実Provider/Targetの資格は独立 |

### #1180 Scenario SSOTと#2391の後段UI制御

| 全Target軸 | 先行／後続の意味 | 未成立の証拠 |
| --- | --- | --- |
| **E0→E2** | 外部要求の机上Pressure→最小基盤→CUIと標準Pluginを並行検証。AI-first計画の[Draft #2492](https://github.com/hat47x/tei-platform/pull/2492)は**未統合の設計候補**。Studio PR1のread-only guardはNode/Headlessな限定試験で先行可能（PR2/PR3の実UI結合は別受入） | 机上適合表の全件保証・本番Contract採用 |
| **E3 — #1180** | Accepted Canonicalは業務意味SSOT、保存Driver Scenarioは**テストSSOT**。CLI/headlessのOperation・Outcome・Receipt・独立Observationを同一Scenarioから派生し、Target Binding/Profileの適合・版を検査。既存[merged #1867](https://github.com/hat47x/tei-platform/pull/1867)は保存Scenario→限定CLI 6 Step生成の証拠 | 実Command＋独立Provider Observation、全Targetへ全Step導出、Version/Binding不一致・未対応Stepのfail-closed、実認可・効果の横断E2E |
| **E4 — #2391（opt-in）** | 同一Scenarioから対象UI操作を派生。外部Runnerを第一候補とし、必要なアプリ内InstrumentationのみTest/Developmentで任意追加。Wails内蔵MCPは特定Targetの比較候補 | 実UI E2Eと独立Effect、Test／Productionそれぞれの**検査対象Artifactと実配布・起動Artifactの同一性**照合、Production専用制御面の物理除去、通常UI/a11y保持。表示・クリック・MCP Tool一覧はOwner意味受理や権限の証拠でない |

**停止・非重複原則**：PR1のsource test PASS、PR3の独立qualified選択PASS、E3のCLI PASS、E4のUI PASS、ADR改番CI PASSをそれぞれ別の証拠とする。#1120のtyped Identity、#267のAcceptance、#1174のEffect/Receipt、#1180のScenario SSOTを再実装しない。#2391は独立したHost/Authority/MCP ABIを新設しない。実行していないConsumer/GUI/Production除去・Owner受理を達成と表記しない。

### PR1/PR2/PR3の反証優先試験とexact-head引渡し（2026-11-02追補）

[PR1 #2583](https://github.com/hat47x/tei-platform/pull/2583)の11月2日確認headは `c71d1b8ca56dc366e4581563c4de0ab5c74f861d`（**Draft・main未統合**）。差分は当初のWorkbench 3 source pathに加え `.github/workflows/tei-studio-workbench-embed-ci.yml` の計4 path。旧head `dc7b447...` のNode 18/18・型検査・build PASSを現headの全ジョブPASSとして転用しない。現headのWorkflow確認時はWorkbench embedded bundleとPoC Aが **failure**、Studio #8 independent consumerと依存監査が **success**、repository-healthは **in_progress**。Workflowでの生成artifact採取は更新候補であり、埋込みbyte一致・新Consumer snapshot資格はまだ証明していない。

[PR3参考 #2457](https://github.com/hat47x/tei-platform/pull/2457)の11月2日確認headは `b90afac103f60f222c5c8397741a2c8978947d77`（**Draft・main未統合**）。当初の一般 `publish(token, render)` に加え、`publishGraph(token, graphView, graphInput)` は `normalizeGraph` 後の可視node集合にBinding対象の全 `viewId` が含まれることを確認し、`binding_not_rendered` を拒否する。未結合の装飾nodeは `unbound` として残せる。Graph更新失敗・非同期renderもfail-closedにする局所Referenceだが、実DOM／Workbenchのイベント結合・Source readback・Inspect RPCの0回証拠ではない。

| 反証fixture・順序 | 直接のassertionと証拠 | 未成立なら |
| --- | --- | --- |
| **PR1-A：namespaceで1件に絞ったglobal ambiguous** | `ambiguous=true` 候補は一覧に保持し、同じ `{kind,id}` のunqualified `InspectSubject` transport **0回**。理由はConsumer側の `SUBJECT_AMBIGUOUS` とし、未送信のRPC結果を捏造しない | PR1未達。単体predicate PASSだけではtransport 0回を保証しない |
| **PR1-B：同一kind/IDの複数Occurrenceと一意対照** | 全Occurrenceがambiguousなら候補を落とさず選択拒否。矛盾するflagはCatalog不正。一意で有効なSubjectは**実transport 1回**。選択変更前の旧Inspect成功／失敗が表示を汚さない | 候補の誤一意化・重複消去・正常系の過拒否を失敗として分ける |
| **PR2：Graph / list / initial / command** | **各入口の実イベント**で曖昧0回、一意1回（意図した各選択ごと）、保留理由・候補保持・選択epochを観測。初期自動Inspect失敗でも独立Validationを妨げない。旧selectedをCommandが再使用しない | `inspect()`共通関数の静的guard確認だけではPR2未達 |
| **PR3-A：成功したG1→G2、同じview ID Kの再利用** | 新描画のnode ID集合とBindingが同じsnapshot。旧DOMクリックからInspect 0回、新DOMはqualified Key一意＋Catalog全体でunqualified APIの対象一意と現在版が確認された場合に限り1回 | view ID、label、rowIndex、旧mapをSource locatorやAuthorityと同一視しない |
| **PR3-B：R10/G1→R11、G2 `missing_endpoint` 拒否** | 可視G1が残ってもM2を公開せず、旧G1クリックはInspect 0回／Source locator確定0回／stale診断維持。Sourceが不変でも描画失敗後のmap適用資格を別途確認 | Renderer単体の旧SVG保持・`publishGraph`単体の拒否はConsumer結合PASSに転用不可 |
| **PR3-C：Source ABA・描画途中更新・binding_not_rendered** | 旧stage tokenは失効、未描画nodeへのBindingを公開しない。意図した未結合nodeは `unbound`。旧成功表示から `ready`／authorizedを推定しない | Source revision観測だけで原子的readback・現在Authorityは証明できない |
| **P1接続：Accept部分成功＋旧応答後着** | `decision=accepted, sourceChanged=true, DECISION_RECORD_FAILED` とR10旧Inspectの合流で、旧schema/readiness失効、Source/ledger別readback（R12ならR12優先）、自動再Accept 0回 | Source受理・ledger記録・Decision witness・Owner意味受理・Runtime Effectを単一successに圧縮しない |

**引渡し単位**：各fixtureのSource/Catalog/Graph観測revision、UI入口、selection epoch、qualified Ref／候補集合、Inspect spy、表示された診断と公開可否、実行asset/build SHA、exact-head run IDを残す。UI selection S、Invocation T、確定Effect EはOwnerにより受理された対応規則がない限り等号で結ばない。Decision witness、Source鮮度、独立Effect/Observation、Ownerの意味受理も別のread-only表示レーン候補であり、証拠が欠けた場合は `unknown/not-verified` とする。四レーンの追加表示は#1124の別段階で評価し、PR1の変更対象に含めない。

**移行停止条件**：#2583の埋込みasset／生成bundle byte一致と、新しいexact Workbench sourceの依存集合・資格付きpin追加を確認する。旧snapshot pinを削除・緩和しない。#2457は汎用`publish`の成功をGraph表示ID対応の証拠として扱わず、可能なら`publishGraph`の実Graph.update経由で受入する。両Draftの現headに対してはfocused/full CI、ブラウザー・WebView2、Owner採用をこの文書作業で実行していない。


<!-- tei-2831-self-build-fitgap-p0-20261105 -->
## 35. 2026-11-05 TEI全体セルフビルド Fit/Gap再評価：G4のP0並行縦切り

**判定・正本**：横断責務は[#2831](https://github.com/hat47x/tei-platform/issues/2831)、Gateは[#1119](https://github.com/hat47x/tei-platform/issues/1119)。照合mainは `36bda76271db4db72ddd41a8566d281f02764b8a`。本節は**優先順位・テスト手順の候補**であり、Go/Node実行、trusted full checkout、最新mainのセルフビルドCI PASS、Owner受入の新証拠ではない。G4は `UNQUALIFIED` のまま維持する。§29/30の「self-buildは業務P0のHard blockerではない」は引き続き有効だが、**セルフビルド自身の最初の実証はP1から独立P0並行レーンへ上げる**。

### 既存Fitと、現在明確なGap

| 層 | 利用する実装・Reference | 未実証の境界（本節のGap） |
| --- | --- | --- |
| Build authoring / exact plan | #1109、`tools/reference-build-authoring-intent` | 宣言Source→実TEI自身のBuild全工程・実Tool消費・結果を単一witnessにする経路 |
| Build-only graph / transform | #1103、`tools/reference-generation-graph`（後者はread-only構造検査のみ） | 宣言されたNode/Edgeとactual実行・生成中間物・失敗/blockedを同一版で突合。構造fingerprintを実Build成功とみなさない |
| Plugin build / exact Runtime Engine / stage | #943、`.github/workflows/tei-self-build-qualification.yml`、`tools/reference-plugin-self-build` | Reference Plugin SB-3の収束と、非Pluginを含むTEI全体の自己構築は別。CI定義があるだけでcurrent mainの実PASSを主張しない |
| Physical artifact / subject binding | `internal/plugin/build.go` のPlugin専用 `BuildPlan`に加え、既存`tools/reference-build-execution-binding`の`BuildExecutionPlan`/`ArtifactSet`/`PluginSubjectBinding`/`RuntimeEngineSubjectBinding`（read-only Reference） | **共通分離の設計自体はFit**。未達は、実TEI CLI/他Targetのactual build attempt・実消費閉包・typed subject bindingを同じ版で接続する製品縦断証拠。偽Plugin IDを作らない |
| Source closure / Toolchain / Target | #1088/#1090/#1096/#2595/#2600と#1251 PB-10 | root Go moduleのTest成功とnested `cmd/tei-studio`の実Build成功は別。選択Targetごとの `go.mod/go.sum`・リンクされた依存・生成Assetの実消費closureが未接合 |
| 標準データ構造・投影 | #2620/#1063/#1124 | Build/Transform/Target/ArtifactをTEI標準構造に写像し、図表から実Graphの全工程を説明する消費側の検証。専用UIを先行させない |

### P0-SB（G4並行）の実施順と最小の完了条件

| 順序 | 具体的な実装・反証 | 最小Evidence／Owner |
| --- | --- | --- |
| **SB-A / P0 即着手** | 実際のSB-3 Referenceを一次baselineに、BuildIntent/Source closure/actual builder execution/produced artifactを観測。宣言Graphと実行Graphの差を read-only ledger にし、Source/Node/Edge/Target/Toolchain/Output/失敗を結ぶ。まず構造表示ではなく「欠落実工程を検出」するテスト | #2831、#1109、#1103、#2620。未宣言pre/post処理、ambient input、undeclared generated inputを `unknown/reject`。Schema・第二Runner・手作りGraphicを先行実装しない |
| **SB-B / P0 並走** | **実TEI `cmd/tei` binaryを第一候補**として既存Go build adapterから実構築する。現行Plugin必須 `BuildPlan` を非Pluginへ偽装転用せず、#1080/#1097のgeneric executionとsubject bindingの最小Referenceを先にpressureする。既存Runtime Engineのexact buildも比較対照とする | #1080/#1097/#1096/#943。実Source/toolchain/Target/Build artifact digestと消費Assetの一式。selected nested moduleのclosure欠落は失敗。非PluginをPlugin Lockへ入れない |
| **SB-C / P0 接合** | stage0 trusted Go/toolchain/Runtime seed→stage1→stage2を実行。Buildとexecution identityを区別し、同Source・同build-only closure・exact engineに対しdigest収束を比較 | #979/#973/#1083/#1095/#1163。stage1/2 artifact digest一致、不一致時 fail-closed、engine差替え・source差・target差の負例。seed trustとHost Grantは別の未証明資格 |
| **SB-D / P1** | CLI＋Runtime＋標準Pluginのmulti-subject ArtifactSetとTarget別output/必要memberを結ぶ。実消費差・未選択Plugin差と再利用資格・待ち時間を測る | #1097/#1251/#2600、Draft #2636。意味/Build/Artifact/Evidence/Current Authorityの再利用を分離。GUIはread-only Projection |
| **SB-E / P2** | Java/.NET/JS/Wasm等の別adapterと異種Targetを、既存上位契約で差分実証 | #1101/#1016/#1121。初回P0のHard dependencyにはしない |

### P0反証を先に固定する（全て今回の新規実行は NOT RUN）

- **SB-N1 hidden step**：図表に無い実build step、未宣言generator input、ambient environmentの影響を識別。実際に未検査なら `unknown` に留め、実行していないstepを「存在しない」と主張しない。
- **SB-N2 selected nested module**：rootのGo testは成功しつつ、選択したnested `cmd/tei-studio` moduleの`go.sum`が欠け実binary build失敗。#1251記録の[#2868](https://github.com/hat47x/tei-platform/pull/2868)→[#2872](https://github.com/hat47x/tei-platform/pull/2872)/[#2873](https://github.com/hat47x/tei-platform/pull/2873)/[#2874](https://github.com/hat47x/tei-platform/pull/2874)を既存反例として引用するだけで、現mainへ再発中とは断定しない。選択moduleの依存閉包と結果はrootから独立に検査。
- **SB-N3 build execution ≠ Plugin subject**：CLI/engine artifactへfake Plugin identityを付けない。shared binaryから2論理Pluginへbindする場合もactual Build Attemptは一回、各Runtime Authorityは独立。
- **SB-N4 stage identity**：同じinterpreted realization bytesでもexact Runtime Engine変更で旧stage execution資格を拒否。stage0のseed署名/Trustとstage1/2収束を混同しない。
- **SB-N5 partial/stale**：一部memberのみ成功、欠落edge、旧Toolchain/Target/Lock、同byte artifactでも不適切な現在資格をfail-closed。
- **SB-N6 unselected vs actual consumed**：未選択PluginのSource追加が必要成果物へ影響しない正例と、Go init/import/embed等で実際に混入した負例を区別。#1251/#2600の既存試験計画を所有元のまま再利用。

### G4への提出方式と独立性

各試行は `tested main SHA / actual test checkout / Source+closure / build intent+resolved plan / builder+engine+toolchain exact identity / selected Target+all module roots / actual consumed bytes / produced artifact members+digests / graph declared-vs-observed / stage receipts / negative results / clean pre-post / tests run and NOT RUN` を固定。対象revisionが違う成功記録を加算せず、read-only Referenceとproduction build、build reproducibilityとHost/Owner Acceptanceを別欄にする。

**P0のExit**：一つの実TEI targetで宣言Graph↔実行Graphを閉じ、非Plugin実artifactを構築して逆追跡し、Stage比較と少なくともSB-N1/N2/N3/N4の拒否を記録すること。これが成立しても全Target/ReleaseのG4製品資格には追加の独立Gateが必要。#1127/#1163の業務Web/DB縦切り、#1124のStudio M0/M1、#1180のScenarioはセルフビルド完成を待たず並走する。

<!-- tei-2831-20261106-other-chats-evidence-closure -->
### 35.1 2026-11-06 他レーンの新成果との照合：既存ReferenceをつないでG4に到達する

**差分と修正点**：2026-11-06 main `bf2e7b0ffe21747a308cd2b37328eb440535f696`の実ソース・関連Issue/PRをread-only再照合。`tools/reference-build-execution-binding`には既にdomain-neutral `BuildExecutionPlan`、`ArtifactSet`、Plugin/Runtime Engine/BuilderのSubject Binding、build-only Graph/Seed/Stageの構造検査、`actual_go_attempt_test.go`の**実Go buildを二回行う限定試験**がある。したがって「非Plugin用Build Execution型のReferenceが存在しない」というGapではない。一方、同試験は独立一時`main.go`を対象とし、`os.Environ()`を引き継ぐため、それだけでは**TEI CLI本体の完全Source closureやhermeticityを証明しない**。#1088/#1096の環境・Go build資産と同一試行で結ぶ必要がある。

#### P0の順序と受入をさらに狭める

| 優先・実装境界 | Fitの利用 | 次に要求する独立Evidence・負例 |
| --- | --- | --- |
| **SB-A0 最先行（Reference接合）** | #1109 `BuildIntent`→Plugin専用resolved Plan、#1103 build-only DAG、#1080/#1097 domain-neutral Execution Reference、#1063 phase付きOperator Graph | **一つの対象Build**について `Intent/source revision`→`resolved selection`→`execution plan`→`actual attempt`→`ArtifactSet`の因果を同じTarget・Toolchainで突合。未実装の非Plugin Intent→Execution接合は`unsupported`、別fixtureの局所PASSの合算は不可 |
| **SB-A1 最優先（範囲付き完全性）** | #2620自己記述のRequired/Declared/Resolved-Consumed/Observed-Disclosed区分、#2600の否定依存・Source ABA、#2881の合成bounded absence | `Required / Declared / Selected / ActuallyConsumed / Observed / Undischarged`をClaim・Target・Scope・Attemptごとに区別。宣言外のpre/post実行、外部tool I/O、生成中間物、path-sensitive Go package/embed、暗黙envを一つ以上反証。**実行可能な外部Toolを完全観測していない範囲は`unknown`**であり、図に無いことから不存在を結論しない |
| **SB-B P0（TEI実binary）** | `cmd/tei`、既存Go Builder・`tools/reference-transform-runtime-engine-build`・`tools/reference-build-execution-binding`、#1251 PB-10 | exact `cmd/tei` Source closure/toolchain/Build recipe/実ファイルdigest/Target/Artifact member/Subjectを同一Attemptで照合。root Go moduleと選択したnested moduleの`go.mod/go.sum`を別々に確認。**既存Plugin専用BuildPlanへfake Pluginを割当てない** |
| **SB-C P0（再帰的Builder）** | #979/#973と既存SB-3、BuilderSubjectBinding/RuntimeEngineSubjectBinding、#1104 Trust境界 | Stage0 external seed→Stage1 builder→Stage2 builderが**同じSource・exact実Builder/Engine実体**を消費したことを確認。Stage1/2 digest収束・engine差替え・seed差・不収束・partial output/Lock拒否を別試行で記録。独立Oracle、Seed Trust、Current Authorityはそれぞれ別のClaim |
| **SB-D P1（追加先）** | #1251 Bundle/未選択module、#2933 Plugin実現・最適化Draft、#2600増分・Draft #2636、#2788単一Build→複数Build Unit | `available` ≠ `selected` ≠ `compiled/linked` ≠ `invoked`、1 physical Artifact:N logical Pluginと現在のAuthorityを分離。同意味でTarget別ビルドをA/B実測、性能改善と安全・Effect保持は別判定。P0のHard blockerにしない |
| **SB-E P1/P2（二Consumer圧力）** | #2880 Typed I/O、#2827 ETL/ELT/MFT、#1130非画面Job、#1121 Lowering | build/ETL双方を型付きNode/Edge/Artifactへ表現し得るか確認。ただしContent bytes、配送Attempt、Import、業務受理は別receipt。File ACK/Build successからCurrent Host/Owner Acceptanceへ自動昇格しない |

**P0反証更新（受入ケース追加、実行はNOT RUN）**：
- **SB-N7 / actual Go fixture限定**：`actual_go_attempt_test.go`の一時`main.go`二回build PASSを`cmd/tei`のビルドやhermeticity PASSと扱ったら失敗。Host ambient envを混入させた場合の影響・拒否条件は#1088/#1096に既存実装を再利用して調べる。
- **SB-N8 / 同digest別Attempt**：実bytesが等しくてもBuilder/recipe/path-sensitive input/actual Attemptが変化したら同一Execution Receiptにしない。Source location変更でもartifact-affecting package pathが変わる場合は安易にdigest同値扱いしない。
- **SB-N9 / 全工程不在の主張**：実行後GraphにNodeが無くても、Tool/Processの観測閉包が非完全なら`not observed / unknown`。閉じたTarget・Scope・Reader/Tool・期間についてのみ`absent in verified scope`を検証。
- **SB-N10 / Plugin最適化は後段**：未選択PluginをCatalogに追加しても実消費へ混入しなければArtifactが無関係に失効しない。実`init/import/embed`・linkによる混入、Provider選択変更・Current Grant変更なら再評価。Fusion/Batch最適化は実Effect・部分失敗・Unknownの意味を保持する別試験とし#2933/#1251/#2600へ委譲。

**PR統合の重複停止線**：同一計画ファイル§35を追加する[Draft #2876](https://github.com/hat47x/tei-platform/pull/2876)と[Draft #2877](https://github.com/hat47x/tei-platform/pull/2877)が競合する。**どちらか一方に内容を収束してからのみ統合**し、両方を別々にmergeしない。#2876には既存domain-neutral execution Referenceの明示があり、本節でそのFitを反映した。#2831がG4の横断追跡Owner、#1109/#1097/#1251/#2600/#2620/#2933等の実装・受入Ownerを代替しない。

**資格の上限**：本節はGitHubのソース/Issue/PR静的照合と計画更新のみ。`P0-A0/A1/B/C`の実TEI CLI/Stage Go Run、fresh full checkout、CI・独立Review/Owner承認は未取得。現在のG4=`UNQUALIFIED`。業務G1/G2・Studio M0/M1・ETL実配送に新Hard dependencyを追加しない。

<!-- tei-2831-20261106-module-closure-recursive-selfbuild -->
### 35.2 2026-11-06 最優先Gapの実ソース追跡：外部Go Module閉包と本当の再帰的セルフビルド

**確認main**：`2d0b09b13b16c7f2d7183dbc995d3c62208c79ef`。以下はGo実装・Issue [#1090](https://github.com/hat47x/tei-platform/issues/1090)/[#1096](https://github.com/hat47x/tei-platform/issues/1096)/[#1088](https://github.com/hat47x/tei-platform/issues/1088)/[#1104](https://github.com/hat47x/tei-platform/issues/1104)に基づく**新しい静的Fit/Gapおよび受入候補**。対象mainでの実Go/Node Build、外部依存解決、CI、Stageの実行は今回**NOT RUN**。G4も`UNQUALIFIED`。

#### Factual Fitと阻害条件

- **Fit / 固定環境**：`tools/internal/gobuilder/build.go`には`BuildPackage`（`go build -trimpath -buildvcs=false -mod=readonly`）、`environment.go`には`GOPROXY=off`/`GOSUMDB=off`/`GOWORK=off`、Host環境deny-by-default、試行ごとに分離した`GOMODCACHE`がある。別エンジンを追加しない。
- **Fit / 実入力列挙**：`source_closure.go`の`ResolveSourceClosure`は`go list -deps -json -mod=readonly`を使い、Go/C/assembly/embed等の選択ファイル・`go.mod`を列挙、Git-tracked外・symlink・repository外moduleを拒否する。既存仕様[#1090](https://github.com/hat47x/tei-platform/issues/1090)の安全な停止線であり、単なる不具合として削除しない。
- **Gap / 外部依存**：root `go.mod`には`github.com/jackc/pgx/v5`、`modernc.org/sqlite`、`golang.org/x/text`等がある。特定Targetの`go list`で外部moduleが**実選択**されれば、現在の`ResolveSourceClosure`は`moduleDir != root`で明示拒否する。さらに隔離`GOMODCACHE`と`GOPROXY=off`のまま外部moduleを事前供給しなければ、Build/Resolve自体が停止し得る。どのmoduleが`cmd/tei`で選択されるか、どの入口が最初に停止するかは**実試験前には未確定**。
- **Gap / 一時実Go試験との違い**：`reference-build-execution-binding/actual_go_attempt_test.go`は、`GO111MODULE=off`の一時的な単一`main.go`をビルドし、`os.Environ()`から環境を派生する。これをTEI本体・外部依存付きBuildのHermetic証拠へ昇格させない。対照として`gobuilder.BuildPackage/ResolveSourceClosure/HashToolchain`を使う。

#### 優先順位をP0-SB-0として前倒し（A0/Bの**preflight**、最初の業務P0の新Hard blockerではない）

| チェック | 再利用する入口 | 必要な観測と停止線 |
| --- | --- | --- |
| **SB-0a / 最初** | `cmd/tei`、`go list -deps -json`、`gobuilder.ResolveSourceClosure` | fixed `main SHA`・GOOS/GOARCH・toolchain・module roots・選択Source・`go.mod/go.sum`・実runner環境を確認し、`supported / external-unresolved / missing-module / unknown`を分ける。外部moduleの存在を、他Targetの`go.mod`宣言から推定しない |
| **SB-0b / 外部が選択された場合だけ** | #1088/#1090/#1096のdeny-by-default環境・#1104のTrust分離 | exact module version・物理入力bytes/検証済みcontent identity・`go.sum`と実選択sourceの一致・外部供給元と隔離cacheへのpreloadを**明示的に**記録してoffline再解決できる方式を検討。暗黙ネットワーク解決・Hostの共有cache・手書き`complete=true`は禁止。`go.sum`存在だけで実消費bytes/署名Trustを証明したとしない |
| **SB-0c / 再結線** | `gobuilder.BuildPackage` + `ResolveSourceClosure` + exact `HashToolchain` + `BuildExecutionPlan/ArtifactSet/Subject Binding` | 一つの`cmd/tei`実Attemptについて実Tool/入力/成果物digestとTyped Graphが対応。不明なmoduleやTool入力があれば`G4=UNQUALIFIED`と保持。前処理・生成器の実入力は別edgeで明示 |

**最小負例**：X1 build成功だがclosureは外部module拒否→**artifact produced**と**qualified closure**を別列にし、G4不可。X2 事前供給無し・GOPROXY=off→失敗を成功へ丸めない。X3 cache内source差替え・module replace・sum/実bytes不一致→受理せず。X4 rootのみ合格／nested module欠落→Target別Fail（#1251）。X5 docs-only変更→無関係artifactを過剰失効させず、`go:embed`やpath-sensitive import変更は失効。これは**テスト候補であって実行済みではない**。

#### さらに分ける二種類のセルフビルドClaim

1. **TEI-defined Build**：TEIの標準構造・BuildIntent/Resolved Graphに基づき、既存Go adapterが実TEI CLI artifactを出力し、閉包と出力がexactに一致。これはP0-SB-Bの最低限のExit。
2. **TEI-driven Recursive Build**：stage0に外部seed（Go toolchain等）を明示した上で、**stage1で生成したTEI Builder/Engineの実ファイルがstage2のBuild処理に実際に使用された**ことを、Process/Builder executable digest・Execution Attempt・Input/Output edgeで検証。同じバイナリをGoコマンドで二回構築しただけではこのClaimに届かない。stage1/2のbit一致は再現性の観測であり、Seed/Issuer Trust・Owner/Runtime Authorityではない（#973/#979/#1104）。

**G4のread-only実証図（提案、図外の未知入力はunknown）**：

```text
version-pinned BuildIntent / Source revision
           |
           v
 resolved typed BuildGraph -----> Required / Declared / Selected
           |                                   |
           v                                   v
 exact Go toolchain + offline module closure + actual consumed bytes
           |
           v
 stage0 external seed --[exact Builder input]--> stage1 TEI build artifact
                                                    |
                                 [actual executed stage1 binary digest]
                                                    |
                                                    v
                                                stage2 artifact
                                                    |
                                                    v
                              ArtifactSet + actual attempts + comparison
                                                    |
                                                    v
                         Build Evidence (G4 Candidate, not Authority)
```

**他チャットとの接合**：[#2620](https://github.com/hat47x/tei-platform/issues/2620)はRequired/Declared/Resolved-Consumed/Observed-Disclosedを別Claimへ分離し、[Draft #2966](https://github.com/hat47x/tei-platform/pull/2966)はtyped queryに相対的なStructural/Mapping/Coverage/Currentの有界な図表完全性を提案。いずれもBuild Processの未観測I/Oを「存在しない」とする証拠ではない。[Draft #2827](https://github.com/hat47x/tei-platform/pull/2827)/[#2880](https://github.com/hat47x/tei-platform/pull/2880)のETL/MFTは同じ型付きグラフを別Consumerへ当てる圧力試験であり、配送ACK・Import・Business EffectはBuild artifactの保証と混ぜない。

**独立検証の責務**：実Go Preflight/外部module閉包=#1090/#1096/#2831、subject binding=#1080/#1097、Stage/Trust=#973/#979/#1104、図表投影=#2620/#2966、過剰・誤再利用=#2600、Target別nested module=#1251。新Core Schema、万能Builder、外部Dependency Store、Host、第二Runnerを先行採用しない。#2876と#2877の§35重複も統合前に一方へ収束する。

<!-- tei-2831-20261106-actual-tei-reentry-interface -->
### 35.3 2026-11-06 TEI自身の実行入口（Re-entry）を追加Gapとして特定する

**静的確認main**: `3572dcedfa50c3de2903dd4d8ca4ad45ccfa1417`。`cmd/tei/main.go:71-124`の公開UsageとCommand Catalogでは`build`/`self-build`を提供するコマンドは掲載されていない。`tools/reference-plugin-self-build/main.go`にはPlugin構築・stage2実行の**専用Reference**が存在する。したがって「TEI本体の汎用Build entrypointが実際に稼働している」とは言えない。ただし、公開CLI一覧は**すべての内部/Plugin経路の不存在証明ではない**。既存#1130非画面Job/#1121 Lowering/#943/#979、`tools/reference-build-execution-binding`に別の再入可能な実行経路があるか先に調べ、新Runnerを作らない。

#### TEIセルフビルドの3段階のClaimを分離

| Level | 主語・証明したいこと | 合格の必要Evidence |
| --- | --- | --- |
| **SB-L1 / reproducible Go build** | 固定したGo Builderと正しい閉包を使いTEI CLI等のバイナリを生成 | exact Go compiler/toolchain、external module closure、`BuildExecutionPlan`、two attempts、output bytes/ArtifactSet digest。TEI自身がbuildを操作したとは言わない |
| **SB-L2 / TEI-defined build** | TEI標準構造の`BuildIntent`・型付きBuild Graphで宣言した入力・処理・成果物が実buildと一致 | 原本Source/解析版、resolved Graph、実Builder選択、宣言vs実消費、入出力とAttempt、Undischarged。まだGoのReference launcherが実buildを実行してもよいが、その制御主体を記録 |
| **SB-L3 / TEI-driven recursive build** | **stage1で出力されたTEI自身のバイナリ**がstage2のBuild Plan解決・実Build起動/委譲を駆動 | stage1 artifact digestに一致する**実プロセス実行ファイル**、stage2が消費したSource/BuildIntent・Toolchain・Builder・ArtifactSet、operation/attempt/exit/resultとStage1/2 digest比較。外部Go compilerはseed/toolchainとして明示し、TEI自身によるGo compiler self-hostingとは主張しない |

#### 最優先の実装順（前節§35.2と矛盾しない）

1. **SB-0a/0b**: `cmd/tei`の実依存module、オフライン供給、sum/実物照合、Go toolchainと実入力のpreflight。解決不能な場合は`unsupported/unknown`として記録。
2. **SB-A0/A1 + SB-L1/2**: 既存Build Execution Referenceと実Go adapterを同一AttemptのTEI CLI buildへ結ぶ。Graphを見せるだけでなく、その実消費入力・生成/実行stepの`Required/Declared/Resolved/Consumed/Observed/Undischarged`を独立Expectedと突合。
3. **SB-C0 / 新P0 Gap**: **Re-entry contract**として、stage1 TEI CLIが同一BuildIntentからBuild Graphを解決し、既存Adapterへ実行要求できる既存入口を探す。存在しなければ必要なコマンド/Operation**一個の薄い接続**を候補にし、`cmd/tei`の公開command list/受入契約・CLIエラー・read-only Previewと明示Executionを分離する。TEI DSLのBuild文法・汎用Scheduler・権限Engineを前提にしない。
4. **SB-C1 / 真の再入実証**: stage1の**生成済みTEIバイナリを起動**してstage2を駆動。実Process image digestとGraph/Execution receiptの`Builder/Invoker`参照を突合する。別Go processがこっそり同じstage2 artifactを作った場合は不合格。stage1/stage2が同bytesでも`invoker`が一致しなければ不合格。

**独立負例（未実行）**：
- **SB-N11**: Stage2のTEI Binaryはbit-identicalだが、実際の命令はStage0 Go launcher/外部Scriptが発行した→**SB-L1だけ**。SB-L3不可。
- **SB-N12**: 読取PreviewだけのTEIコマンドを実Build/権限付与の証拠に転用→不合格。
- **SB-N13**: Stage1 TEI executable digestをmanifestに記載するのみで、実Processが別SHAのバイナリを起動→不合格。
- **SB-N14**: プラグイン/外部Toolに任せた実Command・Input/OutputがGraphの型付きedgeに未記載→`Undischarged`。図表で隠しただけで「不存在」とは主張しない。
- **SB-N15**: 外部Moduleは`go.sum`に記録されているが、隔離cacheへの固定bytesの供給・実選択module rootの照合が無い→full closureと見なさない。
- **SB-N16**: `tei`のBuild Resultが同じでも、Source ABA、Build Attemptの異なる実行主体、Current Authorityの失効を同一identityに圧縮→不合格。（`tei`はテキスト上の仮名であり具体コマンド名を採用したものではない。）

**他チャットの整合・優先範囲**：#2620 Everything is DataのRole/Required閉包、[Draft #2881](https://github.com/hat47x/tei-platform/pull/2881)の合成typed-query bounded absence、[#2966](https://github.com/hat47x/tei-platform/pull/2966)の独立Source/認知研究、[#2880](https://github.com/hat47x/tei-platform/pull/2880)/[#2827](https://github.com/hat47x/tei-platform/pull/2827)のETL二Consumer、[#2933](https://github.com/hat47x/tei-platform/pull/2933)のPlugin最適化、[#2815](https://github.com/hat47x/tei-platform/pull/2815)の柔軟なHash由来IDを参照。**可変表現の派生IDは現行Artifactの完全digest/署名/Lockや実Process同一性を置換しない**。TEI self-buildのP0はL1→L2→L3の**同一Subject・同一Revision・同一Toolchain・独立Expected**の最短経路に限定し、他Target/ETL/Studioの製品資格は独立のまま。

**提出/受理の上限**：この追補はGitHub上のコード/Issue/PRの静的調査にもとづく設計候補。G4=`UNQUALIFIED`。実TEI CLIのL1/L2/L3、真の実Process image照合、実Go/CI/全Health、採用Ownerレビュー、Phaseごとの完全Source/Host観測は**NOT RUN**。#2876と#2877の同一ファイル§35重複を解消するまでどちらもmainへ独立統合しない。

<!-- tei-self-build-fitgap-p0-20261105 -->
## 35. 2026-11-05 TEI全体セルフビルド Fit/Gap：G4の最短実証をP0並行に昇格（提案）

**本節が最新の作業優先順**。§9/§30の「自己構築はP1並走」「wave4」は当時の判断として保持するが、[Issue #1119](https://github.com/hat47x/tei-platform/issues/1119)の2026-11-04追補に従い、**セルフビルドの最短縦切りだけをP0並行**へ引き上げる。既存G0〜G4の定義と業務Web/DB P0のHard dependencyは変更しない。担当・受入の正本は[Issue #2831](https://github.com/hat47x/tei-platform/issues/2831)に置き、#943（Plugin self-build Referenceとしてclosed）の二重起票・再実装をしない。

**照合版** main `36bda76271db4db72ddd41a8566d281f02764b8a`。既存`tools/reference-build-execution-binding/`はBuildExecutionPlan/ArtifactSet/Result、Build-only DAG、seed/stage edge、失敗時pin拒否をReferenceとして表現する。`tools/reference-build-authoring-intent/`はPlugin主体のBuildIntent→BuildPlan導出（実ビルド実行・非Plugin主題・.tei構文は対象外）。`.github/workflows/tei-self-build-qualification.yml`はPlugin SB-3＋exact Engineの参照CIを定義するが、TEI全体の標準構造Graphを正本としてTEI CLI等の実成果物を再生成したという証拠ではない。

| 順位 | Fitを再利用して解く具体Gap | 担当・最初の独立したExit |
| --- | --- | --- |
| **P0-SB-A / 最優先** | **標準データ構造に記述した有効Build graphと実行済み工程が一致するか**。既存BuildIntent/BuildGraph、型付きSource/Operation/Dependency/Resultを連結。実行器が実際に開いたSource bytes、toolchain、環境、生成入力、Node/Edgeと出力memberを記録し、宣言外工程・missing edge・undeclared input・Graphにだけある未実行工程をそれぞれ未証明または拒否。Diagramはこの同じGraphからのread-only投影で、図中にない隠れフックを許さない。 | #2831、#2620/#1063、#1109/#1103。まず既存Plugin SB-3 Referenceの**実工程**をGraphと照合する一つのHeadless fixture。独立したGraph Model/Runner/Core DSLを先に新設しない |
| **P0-SB-B / Aと並行可能** | **非PluginのTEI自身の実成果物**（最初はCLIかRuntime Engineの一つ）へ、正確なBuild subject、BuildIntent→解決済みPlan、言語Adapter、Source closure、toolchain、Artifact Set/Result、digest、Lock、runtime engine bindingをつなぐ。Plugin identityを偽装しない。 | #2831、#1080/#1097/#1096/#973。実バイトを二度構築してdigest比較、Source改版で失効・docs-only非干渉・wrong engine/target拒否を一件ずつ |
| **P0-SB-C / A+B接合後** | **Stage0→Stage1→Stage2の実行・収束**。stage0 seedの外部性とexact実行identity、同じSource closureとbuilder/program、stageごとのartifact digestを別々に記録。stage1/stage2同値と、seed/engine/Source/Builder変更時の失効・不収束を検証する。 | #2831、#979/#973/#1095/#1083/#1163。収束Evidence、失敗Evidence、exact-head/clean pre-post、実バイナリに結びつくProofの未取得を分離。収束はseed trustや正式Acceptanceを意味しない |
| **P1-SB** | 複数logical subject→一physical Artifact、複数output/Target、一式の公開・増分Build省略・selective Evidence再資格化・Studio図表Projection | #1097、#2600（Draft #2636を再実装しない）、#1016/#1124 |
| **P2-SB** | Go以外のbuild adapter、Web/Java/.NET/Wasmや書籍等の異種Target、全TEI成果物のrelease-wide self-hosting | #1101/#1121/#1016 |

**受入・Claimの停止線**：(a) GraphのfingerprintやRenderer表示のみで実工程を見たとしない、(b) Runtime PluginのSB-3同値をTEI全体のself-hostingと呼ばない、(c) Go外部seed・現行Toolchainの事実を隠さない、(d) exact Source/Builder/Target/Artifact/Build Evidenceが一致してもOwner意味受理・現在のRuntime権限・本番配備を主張しない、(e) 旧G4は**UNQUALIFIED**を維持する。最初の業務P0（#1127/#1163）やStudio M0/M1（#1124）には新たなHard blockerを設けない。

**この追補の実施区分**：main上の静的Source/Issue/Workflow照合にもとづく**優先順位・Fit/Gapの設計提案**。PR branchでの実Go/Node Build、CI run、exact-head全Repository Health、Stage1/2実収束、人的Owner受理は、この追補で新規に実施していない。確認できた既存Referenceの有無と、新しいProduct E2E資格を混同しない。
## 35. #2583/#2457統合後のStudio・Headless優先順（2026-11-03）

**時点と優先関係**：main [`0e18b916aa15bc7768e9abfb34318b98ee497cb3`](https://github.com/hat47x/tei-platform/commit/0e18b916aa15bc7768e9abfb34318b98ee497cb3)の読取りに基づく**後続差分**。§33で「Draft／未統合」「bundle/pin修復待ち」と記した記録は、当時のexact-head経緯として保持し、現在の停止条件としない。製品のG0〜G4やS01/S02、既存Owner受理を新規定義・達成判定しない。

- **統合済みの実装・文書**：[merged #2583](https://github.com/hat47x/tei-platform/pull/2583)（merge `681a6c87`）のCatalog `ambiguous`保持＋共有`inspectEligibility` guardはmainの`reference/tei-studio-workbench/{presentation.ts,main.ts}`へ入り、PR1のbundle生成一致と限定CIが報告済み。[merged #2457](https://github.com/hat47x/tei-platform/pull/2457)（`87cbe945`）の`qualified-selection.mjs`は`publishGraph`によるnode集合の照合、stale/unbound/描画失敗のReferenceを提供。[merged #2590](https://github.com/hat47x/tei-platform/pull/2590)（`3cd97c07`）は§33の受入案を文書統合した。ただし、`main.ts`からqualified-selection Referenceへの実配線と、4入口／残存DOMのConsumer受入は未確認・未完。
- **Host方針の処理済み事項**：[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)が採用・main統合済み（[#2426](https://github.com/hat47x/tei-platform/pull/2426)、merge `b6e6cdf4`）。既存ADR-0041/0042は維持、[#2341](https://github.com/hat47x/tei-platform/issues/2341)はClosed。[#2226](https://github.com/hat47x/tei-platform/pull/2226)も同mergeでMergedと表示されるが、別個の採用試験とはみなさない。TEI独自共通Hostは**非必須**、Wails/Theiaも**非必須**。既存PoC WailsのHost/bridge語彙はそのTargetの実装・歴史的Evidenceであり、全製品の前提条件ではない。

| 優先・既存Owner | 次に実証する最小範囲 | 完了と誤認しない条件 |
| --- | --- | --- |
| **1 / #1124 PR2（直接ブロッカー）** | Graph・list・initial・commandの**実イベント**から統合済みCatalog guardへ到達させる。二scope同kind/IDの曖昧候補・絞込み後1件・正常一意を同一Fixtureで比較し、曖昧なunqualified Inspect transport **0回**、一意対象は意図した1回、診断と候補保持をassert。Source移動でMeaning同一でもSourceIndex/locatorの鮮度を再確認 | 共通`inspect()`での静的拒否とNode単体PASSはPR2の実4入口PASSではない。既存ReadOnly PortとSource Ownerを再利用し、第二Resolverを作らない |
| **2 / #1124 PR3** | mainへ統合済み`createQualifiedSelectionGuard/publishGraph`を実Graph・Workbench側へ接続。view ID→qualified Refと描画snapshotの同一性、正常G1→G2、R10→R11＋G2 `missing_endpoint`拒否で旧G1 DOMクリック後も**Inspect 0・locator確定0**。ABA・別Window・既描画DOM/ARIA/診断の失効も必要範囲で測る | `publishGraph`の単体PASS、旧SVG保持、表示ラベル一致、Source`modelRevision`観測だけで、現在Source・開示資格・Authorityを保証しない |
| **3 / #1124 P1 Freshness / Readiness** | merged #2293/#2294/#2348の設計に従い、Source R10→R11で旧schema/Inspect/readyを失効。`decision=accepted`＋`sourceChanged=true`＋`DECISION_RECORD_FAILED`の場合、SourceとDecision ledgerを独立readbackし、自動再Accept **0回**。S01は不要なmigrationなし、S02は実Target互換性証拠なしの誤Readyなし | Source受理、ledger永続化、Decision witness、Owner意味受理、Target実現準備、表示の公開資格を一つの成功Statusへ圧縮しない |
| **4 / #1124 P2 Authoring / Observation** | 三視点のtyped提案→Preview/meaning diff/Impact→既存#267 Shared Acceptance→再投影を同一revisionで一往復。元Markdown等のSource Fidelity（[#2621](https://github.com/hat47x/tei-platform/issues/2621)）とNo-op Save byte不変を検査。[merged #2747](https://github.com/hat47x/tei-platform/pull/2747)のStudio `PlanCommand/ExecuteCommand`は**ReferenceExecutor**であり、実DB永続化/正式Receiptではないと表示 | `Committed=true`やUI応答成功を実Provider Commit・Business Completion・Action Owner受理へ昇格しない |
| **並行 / #1180・全Target E0→E3** | 要件Pressure→最小基盤→CUI/標準Plugin→同じAccepted Scenario由来のHeadless Driver/Provider/独立Observationを先行。[#2492](https://github.com/hat47x/tei-platform/pull/2492)は確認時Open・main未統合の計画候補。CLI限定6-Step生成（merged #1867）は再利用 | StudioのPR2/PR3はNode/実DOMの限定fixtureで独立に進行可能。Headless E3成功を実GUI/実認可/実業務Effectの全Target PASSとしない |
| **後続opt-in / #2391 E4** | #1180 ScenarioをTarget UI操作へ派生。外部Runnerを基本とし、アプリ内Instrumentationは必要なBuildのみ追加。Productionでは**専用制御面を物理除去**し、検査対象Artifactと実配布・起動Artifactを照合。通常UI・a11yは維持 | Wails内蔵MCP、Playwright等は対象固有の選択肢であり、TEI Host/Wails/Theiaの必須化・第二Scenario/Authorityを意味しない |

**着手判断**：PR1の再実装・旧bundle mismatchの再起票・ADRの再採番は不要。最初の新しい実装PRは**PR2の4入口Consumer統合とInspect spy**に限定し、PR3のGraph snapshot接続を次に扱う。PR2/PR3に共通するSource現在性と開示資格の情報が不足する場合は、無根拠に`resolved/ready`へ昇格せず`unresolved/not-verified`に留める。全Target Headless E3は並列進行、E4は後段であり、Studio自体の成立要件にしない。

**証拠資格**：この節はGitHubのPR/Issue、mainコードとADR索引の**静的照合**。新しい4入口UI、実DOM、Accept+旧応答の結合、実Provider/DB、Production物理除去、ユーザー試験、Owner意味受理を本節で実行・達成した主張ではない。

## 36. 2026-11-05 Studio M0〜M4 / Headless E0〜E4 / EV-R0 現行証拠による優先順位更新

<!-- tei-studio-m0-m4-e0-e4-evr0-roadmap-20261105 -->

**時点**：main [`45a4bb0308359a026c1700c112d7b6671e3a908b`](https://github.com/hat47x/tei-platform/commit/45a4bb0308359a026c1700c112d7b6671e3a908b)（2026-11-05 01:46:01Z）を固定。§33/§35は各時点の履歴として保持し、この節のみ現況を優先して読む。既存#1119 G0〜G4、S01/S02、Ownerによる意味・Authority受理、#1180 Scenario SSOTを変更しない。[#1124現行本文](https://github.com/hat47x/tei-platform/issues/1124)の冒頭は11/04に**Script-first / Everything is Dataへ統合済み**。Graph/ER編集を一律必須とはせず、authoritative Script/DSL・Markdown・表からgeneric Property/Table/Tree/Graph/式へ投影し、Source span+Revisionへ可逆なもののみPreview/既存Acceptanceを通す。unknown Role・多対一/非可逆Graphはread-only/拒否+Source navigate。専用Studio Editor正本や第二Policy Engineを作らない。

### A. 既存の確定した成果と、完了と扱わないもの

| Evidence / 完了した機構 | 限定成立事項 | 達成していないProduct条件 |
| --- | --- | --- |
| [merged #2583](https://github.com/hat47x/tei-platform/pull/2583) + [merged #2457](https://github.com/hat47x/tei-platform/pull/2457) | `presentation.ts`のglobal ambiguity保持・`inspectEligibility`、`main.ts`の共通Inspect入口guard。別途`qualified-selection.mjs`の`publishGraph`・scope/ref snapshot拒否。 | 現行`main.ts`にqualified-selection import/実Graph接続はなく、4入口実イベントのInspect 0/1 spy、R10→R11+G2失敗後の旧DOMクリック0-Inspectは未実証。 |
| [merged #2747](https://github.com/hat47x/tei-platform/pull/2747) | Studio`PlanCommand/ExecuteCommand`・UI/Wails bridge・ReferenceExecutor。 | `Mode=reference`の`Committed=true`は外部DB Commit・Receipt・実Authorityでない。 |
| [merged #2626](https://github.com/hat47x/tei-platform/pull/2626) | 実Go CLIによるSource移動でWorldFP不変/SourceIndexFP・Trace locator変更、人/AI二素材の限定生成・再読取。 | 同一Studio Consumerの実CLI byte受理、現在Policy、Source引用/DOM再資格化、独立SHO/AI受領は未実証。 |
| [merged #2853](https://github.com/hat47x/tei-platform/pull/2853) + [merged #2854](https://github.com/hat47x/tei-platform/pull/2854) | #2853の実PG14.24 opt-in Product Invoker→Lifecycle→Adapter→Runner→PG一Attemptでcommitted/独立read、conflict、commit応答不確実→unknown/再送拒否、同一OperationIDのreceipt照会。#2854は別ReferenceのA→B→A＋新key再適用とreceipt二件の実PG反例。 | Studio操作→そのPGへの結線、実断線/プロセス再起動、durable Owner fencing、本番Grant、CASのABA防止は未証明。#2854は防止実装でない。 |
| [merged #2857](https://github.com/hat47x/tei-platform/pull/2857) | 暫定Web→実PG一改訂E2Eの実走と監査が記録済み。 | 監査は`INCOMPLETE / UNQUALIFIED`。暫定D8〜D11をG1/G2正式資格へ昇格しない。 |
| [merged #2848](https://github.com/hat47x/tei-platform/pull/2848) / [#2849](https://github.com/hat47x/tei-platform/pull/2849) / [#2856](https://github.com/hat47x/tei-platform/pull/2856) | run-plan出力別read-only RowReader/Error、Action `emptyPolicy`宣言、期間+Effect前提合成。 | 表示0件≠Provider完全空≠業務完了。実Provider全件Read・Completion Claim・Schedulerは別。 |

### B. #1124 M0〜M4 — 最小Consumer受入順（別Gateではない）

| 順序 | 既存受入と次の最小実証 | 状態／停止条件 |
| --- | --- | --- |
| **M0 実Source** | 同一checkoutの実`tei model catalog/inspect/trace`から実行物・Workspace・Subject/Scope・Source revision・WorldFP/SourceIndexFP・実読取byteを採取し、**同一read-only Consumer**へ入力。取得していない診断はunobserved。 | CLI・Source移動の限定証拠は再利用可能。製品Consumer縦断は**NOT RUN**。 |
| **M1 4入口 PR2→Graph PR3** | Graph/List/initial/commandの**実イベント**を、既配線の共通`inspect()` guardへ通す。異scope同kind/ID・絞込後1候補でもambiguousならunqualified Inspect 0、合法一意なら1回、候補/独立Validationを保持。次に`publishGraph`を同一render snapshotへ実接続し、R11観測後の旧G1 DOM/ARIA/Graphクリック・G2拒否更新でInspect 0／locator確定0。 | 既存ソース上の4ルート・guard配置を「4入口実イベントPASS」と呼ばない。Source変更未観測期間の即時撤回も主張しない。**PR2/PR3本体未実証**。 |
| **M2 Source/Decision・失効** | R10後着の新規表示拒否と、既表示情報の失効を別assert。SourceとDecision ledgerを独立readbackし、`accepted+sourceChanged=true+DECISION_RECORD_FAILED`なら自動再Accept 0。Source不変でもPolicy/Grant/Role Pin失効は独立判定。 | #2853の実PG Attemptは別Producerの証拠。Studio Source watcher・Accept UI・Policy現在性・Readiness実接続は**NOT RUN**。 |
| **M3 Claim・開示・Evidence Presenter** | 同一用途のeffective Query/read-set/Scope/Policy、Manifest Entry→実Path→byte/digest→Reader generationをOwnerの既存資格で再照合。Decision Review/witness/現在性、Command/Effect/Delivery、Owner意味、Source locatorを独立列にする。公開可能なsliceだけを4レーンへ渡し、二世界DOM/ARIA/diagnostics/AI payload非干渉を確認。 | [Draft #2833](https://github.com/hat47x/tei-platform/pull/2833) head `eee5ac743fad5868edc0534db128ba1ffa31b712`：`projectEvidenceView` EV-R0-01〜08 **Node 8/8はPR報告の公開合成試験のみ**。mainへ未統合、現在main比ahead 3/behind 94。実Policy/Authority判定器・real CLI/ledger・実Web/Desktop/SHO/AI受信・旧DOM消去は**NOT RUN**。 |
| **M4 Script-first往復** | authoritative Script/Markdown→generic Projection→Source span/Revisionでtyped change候補→Preview/Impact→既存#267 Acceptance→Source byte再parse/reproject。No-op Save byte不変、源表記保持、unknown Role/plain-data fallback、可逆Graphまたは非可逆Graph拒否を確認。 | #2747 ReferenceExecutor・Draft #2617/#2631の局所合成を製品Editor/正式Source writer/外部Effectへ昇格しない。実Studio/Acceptance一往復は**NOT RUN**。 |

### C. #2391 E0〜E4 — 全Target軸は並走し、E4をStudioの先行条件にしない

| 段階 | 接続先・優先作業 | 現在の証拠資格 |
| --- | --- | --- |
| **E0** | 外部要件PressureとRole/Binding/Scope・Targetの網羅を時点付きで固定。 | [Draft #2492](https://github.com/hat47x/tei-platform/pull/2492)は未統合計画。要件集合の全網羅・Owner採用は未証明。 |
| **E1** | 既存型付きOperation・Plugin/Role閉包・Authority/Effect/Receiptを再利用し、第二Host/Engineを置かない。 | Reference基盤は存在するが、実主体/Host/Providerの全組合せは未実証。 |
| **E2** | AI向けCUI・標準Pluginを並行。明示Role pin、Capability候補、選択Binding、Grant/Target・CLI versionを現行に再資格化。 | CLI/操作導出の局所実装・試験あり。外部Harnessと同一実Targetの意味・実行資格は別。 |
| **E3** | [#1180](https://github.com/hat47x/tei-platform/issues/1180)保存ScenarioをテストSSOTとし、Scenario→Operation/Driver→Provider→Receipt→独立Readを同じAttemptで結ぶ。 | #2853実PG/ #2857暫定Web実PGは**限定実績**であり、同じ保存Scenarioから全Stepを生成・独立Observationした#1180製品E3 PASSではない。Draft #2830の`Verified=true`もstatic/evidence scope限定。 |
| **E4 opt-in** | 実UIのLocate→Act→Observe→独立業務Read、Test/Production二Build。Test専用Instrument・MCP制御面をProduction実配布bytes/dependency/entrypoint/起動実物から**物理除去**、通常UI/a11yは維持。#2515 OS/Network別証拠。 | 実GUI/OS/Network二世界、Production除去・Artifact identityは**NOT RUN**。[#2391のA-13注記](https://github.com/hat47x/tei-platform/issues/2391#issuecomment-6619117946)の通り、テスト用制御MCPと[#2093](https://github.com/hat47x/tei-platform/issues/2093)業務Target MCPを区別（注記提案であり採用/除去実証ではない）。 |

**次の直接作業と並列可能範囲**：Studio実装は **M0実CLI→同一Consumer入力とPR2の4入口イベントSpyを最小縦切り**にし、PR3を次のGraph/旧DOM分離PRとする。M2 Source/Policy失効は実Producer観測に応じて限定assertし、M3のEV-R0はDraft #2833側で独立に開示済み合成表示を成熟させる。M4の可逆Source編集は#267/#2621の契約へ接続。#1180 E3はOwner別に並列、E4は後続opt-inでありHost/Wails/Theiaを必須化しない。PR2/PR3/E3/E4の既存担当branchを触らず、#2775は**roadmap文書のみ**を保つ。

**確認/未実行の区別**：この節は固定main/Issue/PR/既存実装の**静的照合**と、担当PRに記録された限定試験報告の再分類。新しいGo/Node/Browser/WebView2/PG/Scenario実行、実Disclosure、Repo full health、Owner意味受理は**この節ではNOT RUN**。PR #2775は過去のbaseから分岐したDraftのままで、現行main比較は更新前 ahead 1/behind 131（Roadmap以外のmain変更を併合・mergeしていない）。統合前に新headのCI・競合・Reviewを再検証する。

## 37. 2026-11-05 #1/#4/#8/W1分担：M0〜M4・E0〜E4 一枚ロードマップ

<!-- tei-studio-20261105-assigned-owners-single-roadmap -->

**固定版**：main [`36bda76271db4db72ddd41a8566d281f02764b8a`](https://github.com/hat47x/tei-platform/commit/36bda76271db4db72ddd41a8566d281f02764b8a)（2026-11-05 12:48:57Z）。本節は§36の**後続の担当・成果物・依存の差分**であり、旧節の固定SHAを事後に書き換えない。ユーザーが#1実装着手、#4/#8/W1独立レビューを**依頼済み**という担当計画を記録する。依頼・役割の存在は、担当PRの作成、レビュー完了、実行PASSの証明ではない。#1124は[現行H0〜H4の一Consumer一往復](https://github.com/hat47x/tei-platform/issues/1124)とScript-first Source正本を採用済み。#2391のE0→E4、#1180保存Scenario正本、#1119 Gate/Owner意味受理を再定義しない。

| 優先・範囲 | 成果物・担当（依頼済み／既存Owner） | 依存・再利用する実装済み証拠 | この時点の受入資格・未実証 |
| --- | --- | --- | --- |
| **P0 M0→M1 / #1 実装着手** | **#1が唯一の小実装Owner**：同一fixture/checkoutの**実CLI model catalog/inspect/trace→同一Workbench Consumer**を、既存Harnessへ結ぶ。最初は**Initial/List実イベント**から始め、実CLI出力の起源、SourceRevision/WorldFP/SourceIndexFP/Scope/ambiguous保持、Inspection RPC **0/1回**をassertする。差分は小さなテスト/接続に限定し、PR2/PR3完成を先取りしない。 | [merged #2626](https://github.com/hat47x/tei-platform/pull/2626)実CLI/Source移動、[merged #2583](https://github.com/hat47x/tei-platform/pull/2583)共通`inspectEligibility`、#1824 `cli_catalog_parity.py`/Browser smoke、#2033 reference基盤。 | **着手依頼済み・専用実装PR/PASSは未確認**。別scope同kind/id、絞込後一行でもambiguous→Inspect 0、一意な対象→Inspect 1を実イベントと実RPC/transportで証明するまではM0/M1製品受入**NOT RUN**。 |
| **P0 #4 独立レビュー** | #4は**Scope/Source/Authorityと受入負例**を担当。曖昧性、同名異scope、実CLI Input origin、旧source locator、正規Consumerへの到達、Inspect 0/1回を#1実装とは**独立に査読**し、失敗なら再現条件と観測不足を返す。 | #1124 M0/M1/M2受入、#2583 guard、#2457 Binding、#267部分成功、#2516 Policy Owner。 | **レビュー依頼済み／独立レビュー結果未取得**。Auth/Grantの現在性をUI guard・CLI成功から推定しない。 |
| **P0 #8 独立レビュー** | #8は**PoC/Browser資産再利用と証拠資格**を査読。`cli_catalog_parity.py`（実CLI↔Bridge）、`browser_smoke.py`（実DOMイベント）、`browser_isolation_smoke.py`、RPCRecorder/Test adapterの差を切り分け、#1の入力から**本物のtransport RPC増分0/1**が測れたか確認する。 | #1824/#2033既存CLI/Browser/WebView2実測を再利用。新Browser Harnessや第二Consumerを再実装しない。 | **レビュー依頼済み／実Target別の再実行は未確認**。単独CLI、Test adapter、synthetic event/Browser smokeのPASSを同一CLI→Consumer E2Eへ転用しない。 |
| **P0 M1 PR2（#1後続）** | Graph/List/Initial/Commandの**4入口実イベント**を共通Inspectに通し、ambiguous/合法な候補/別ScopeのRPC 0/1回を統一ケースで検証。#1の初回Initial/List証拠を拡張。 | 現main `reference/tei-studio-workbench/main.ts`の4入口は**ソース上で既に`inspect()`へ配線**、同`presentation.ts`のguardはmain統合済み。 | **構造配線は実装済み、4入口実イベントとScopeを含む製品Consumer受入は未実証**。重複して4入口を再実装しない。 |
| **P0 M1 PR3（PR2の次）** | `createQualifiedSelectionGuard`/`publishGraph`を**実Graph/Workbenchの現在render snapshot**へ結線。R10→R11 Source変更、G2描画失敗後の残存G1 DOM/ARIA/クリックを0-Inspect・0-誤Source locateで検査。 | [merged #2457](https://github.com/hat47x/tei-platform/pull/2457)は独立Referenceで実装済み。ただし現main `main.ts`にはqualified-selection import/`publishGraph`実接続なし。 | **PR3実配線・実DOM負例NOT RUN**。Renderer generationのみをSource/Policy epochの保証にしない。 |
| **P1 M2 / Source・ledger・鮮度** | Source観測R10→R11で**既表示の資格失効**と**旧応答reject**を別assert。部分成功後のSourceとDecision ledgerの別readback、自動再Accept 0、Grant/Review/Query SQL候補切替の独立資格をConsumerへ接続。 | #1124 H0/H1/H2、#267既存Acceptance、#2586 Query HJ-08、#2829 Review ABA Draft。 | Workbench Accept/Source watcher/current Grant、旧DOM消去、#2586実Query選択/実SQLを伴う同一Consumerは**NOT RUN**。新Dialect Editorを先行しない。 |
| **P1 M3 / W1独立レビュー** | **W1が[Draft #2833](https://github.com/hat47x/tei-platform/pull/2833)を独立レビュー**。EV-R0 4レーン（Operation、Declared Action、Command/Effect/Delivery、Owner Meaning）の**公開合成read-only投影のみ**を査読。偽造可能な`prepared-public-fixture`、二世界共通fallback、strict input、same ID≠causality、reference Committed≠DBを重点確認。 | #2833 HEAD `eee5ac743fad5868edc0534db128ba1ffa31b712`、3新規ファイル。PR報告は局所Node **8/8 PASS**。当該HEADのGitHub Actions check-runs **10/10 success**（Repository Health等；専用EV-R0 Node8件のCI実行と同義ではない）。 | **W1レビュー依頼済み／独立所見未取得／PRはDraft未統合**。実Policy/Grant/Disclosure、実Producer/Source/ledger、DOM/ARIA/Tool JSON二世界、Recipient受領、Owner意味受理は**NOT RUN**。 |
| **P2 M4 Script-first** | H0→H4で正確なSource span/Revisionが確定してからtyped change→Preview→既存#267 Acceptance→実byte再parse/reproject。No-op Save byte不変、未知Role/非可逆Graph拒否＋Source navigate。 | #1124 Script-first受入、#2619/#2620、#2621 Fidelity、#2747 Studio reference Command/bridgeは統合済み。 | 参照`Committed=true`を実DB/Receipt/Authority/業務完了としない。実Studio Source編集一往復は**NOT RUN**。 |
| **並列 E0→E2 / #2391** | E0外部要件圧力→E1既存Operation/Role/Plugin/Authority→E2 CUI＋標準Pluginの意味・Binding・Grantと実CLI入出力照合。 | [#2391](https://github.com/hat47x/tei-platform/issues/2391)現行順序、#2605 CLI、#1180 Scenario SSOT、[Draft #2492](https://github.com/hat47x/tei-platform/pull/2492)は未統合の計画。 | E0全網羅/Owner受理、同一TargetのRole→Binding→Authority/Provider起動は**NOT RUN**。 |
| **並列 E3 / Headless Driver** | 保存Scenario→実Operation/Driver→Provider→Receipt/独立Readを**同一Attempt・固定Artifact**で連結し、Scenario`Verified`の局所静的意味と外部Effectを別欄にする。 | [merged #2853](https://github.com/hat47x/tei-platform/pull/2853)実PG Product Invokerのcommit/unknown/replay、[merged #2854](https://github.com/hat47x/tei-platform/pull/2854)別Reference ABA反例、[merged #2857](https://github.com/hat47x/tei-platform/pull/2857)暫定Web→PG一改訂。 | **実PG限定E2Eは実績あり**だが#2857監査は`INCOMPLETE/UNQUALIFIED`。#1180保存Scenarioからの製品全Step同一E3、正式Grant/Owner受理は**NOT RUN**。 |
| **後続 E4 / opt-in UI** | 実Web/Native/GUI locate→act→observe→独立業務Read、外部Runner/実OS送出を検証。Test専用制御面のProduction物理除去は**実配布byte/closure/起動Artifact**で通常UI/a11y維持と別照合。 | #2391 A/B/C・W0、#2515 OS/egress、#2093の業務用MCP Targetとテスト用制御MCPを区別。 | 実UI E2E、実OS/Network、Production物理除去、Artifact同一性は**NOT RUN**。**Host/Wails v3/独自MCPを必須化しない**。 |

**短期の依存線**：`#1 実CLI→同一Consumer Initial/List` → `#4 Scope/Authority負例レビュー + #8 Browser/transport証拠レビュー（並行）` → `PR2 残りGraph/Commandの実イベント` → `PR3 qualified Graphと旧DOM失効` → `M2 Source/Policy/ledger` → `M4 typed Source編集`。**W1→#2833 M3合成表示レビューはPR2/PR3と独立並列**、実Producer/Disclosure接続のみM0/H0と現在Policyの証拠に依存。#2391 E0〜E3は別Owner並列、E4は後続。レビューにより失敗を発見した場合は#1または既存Ownerへ差し戻し、#4/#8/W1が別Consumer・別Guard・別Authorityを実装して重複しない。

**確認の境界**：この§37はmain・Issue・PR・WorkbenchソースとGitHub Actionsの**実記録照合／担当計画の更新**。#1/#4/#8/W1の新規実装・独立レビュー完了を確認した文書ではない。**今回未実行**：実CLI→実Workbench、Node/Goの新規ローカルtest、実DOM/WebView2/DB/Source edit/OS/Disclosure、Owner承認、full trusted integration。既存Draft #2775のroadmapブランチのみを更新し、#1/#4/#8/W1の作業branch・#2833・#1124/#2391本文には手を入れない。


## 38. 2026-11-07 #2775統合後：M0→M1 / #4・#8独立レビュー / #2833の新しい証拠資格

<!-- tei-studio-post2775-m0m1-evidence-20261107 -->

**固定時点**：[main `9fec9fa38f13e08be20183d0ac6e53766ef1ffcf`](https://github.com/hat47x/tei-platform/commit/9fec9fa38f13e08be20183d0ac6e53766ef1ffcf)（2026-11-07 14:39:42 UTC）をGitHubから照合。旧[PR #2775](https://github.com/hat47x/tei-platform/pull/2775)は**merged**（head `225617b845b9758380b090a6ae695b1eeca5e6d0`、merge `f2c5e042274100cf778f3def065453a76e6501ee`、11/07 04:24:41 UTC）。RESTの`draft=true`履歴より`merged=true`/merge commitを優先する。§35〜37の「Draft/未統合」は**過去の固定時点の記述**であり現況ではない。§37はmainの当該Markdownに存在することを再読取済み。

### A. 一枚の担当・成果物・次の証拠（実装/レビューの進捗を推測しない）

| 順序・責務 | 固定できた成果／最新状態 | 残る最小成果物・実証境界 |
| --- | --- | --- |
| **P0 #1：#1124 M0→M1実装** | [merged #2626](https://github.com/hat47x/tei-platform/pull/2626)の実CLI Source観測、[merged #2583](https://github.com/hat47x/tei-platform/pull/2583)のCatalog ambiguity保持/共通Inspect guard、[merged #2457](https://github.com/hat47x/tei-platform/pull/2457)の独立qualified-selectionを再利用可能。現main `reference/tei-studio-workbench/main.ts` blob `9582c85db4c27c1f89d8594f5b212984d755c9c0`でguard・4入口のコード上配線を確認。 | #1担当の最初の**同一CLI実出力→同一Workbench Initial/List実イベント**の最小Harness/PRと固定HEAD・試験ログは、今回のPR探索・引継ぎ資料で**未確認**。実CLIから曖昧A（異Scope同kind/id・絞込後一行でもambiguous）の全Inspect transport **0件**、独立unique Jの**1件かつSubject引数一致**を同一Attempt/Source snapshotで観測。初期選択順はfixtureで明示。CLI出力/Bridge Adapter/表示の対応付けが欠ければM0→M1 PASSとしない。 |
| **P0 #4：Scope・Source・Authority独立レビュー** | #4へ依頼済み。現mainの共通GuardはCatalog内一意性の判定でありSource locator/現行Grant/実Authを付与しない。 | 実CLI A/J負例、異Scope同名、List/初期選択順、旧応答/Source移動、actual Inspect 0/1回・同一Attempt identityの欠落箇所を**#1の実装と独立に**指摘する。今回#4の固定版レビュー完了記録は未確認。Scope無しInspectが合法Targetを特定したとみなさない。 |
| **P0 #8：CLI/Browser/Recorder証拠の独立レビュー** | #1824 `cli_catalog_parity.py`、#2033 `browser_smoke.py`、`browser_isolation_smoke.py`のRPCRecorder、PoC A同一Serverの順次測定は再利用候補。 | 実CLI snapshot起源、同じWorkbench/BrowserContext、実DOMイベント、**全要求数**・Inspect呼出し引数/時点/件数を対応付ける。Test adapterによるシミュレーションの0/1と**実Wails bridge RPC**の0/1を混同しない。今回#8の固定版レビュー完了記録は未確認。 |
| **P0 PR2→PR3：4入口と旧Graphの二段** | Graph/List/Initial/Commandから共通`inspect()`への**構造配線は既存**。独立Reference `qualified-selection.mjs`の`publishGraph`はmerged #2457。現main `main.ts`にはqualified-selection import/実`publishGraph`接続なし。 | **PR2**：#1のInitial/Listを足場にGraph/Commandも実イベント・同一guard・Inspect 0/1件で検証。**PR3**：実renderのSource/scope/epochとqualified mapを同期、R10→R11/G2更新拒否後の残存G1 DOM/ARIAクリックを**0 Inspect/0誤Source navigate**で拒否する。renderer generationだけでSource/Grant失効を保証しない。製品E2Eは未証明。 |
| **P1 M2：現在性・Decision ledger** | #1124 H0〜H4、既存#267 Acceptanceの部分成功負例、#2626実Source移動を別Ownerの限定証拠として再利用。 | 旧表示の失効と旧応答の拒否を別assert。CLI非ゼロでも`accepted+sourceChanged=true`ならSource/Decision ledgerを別readbackし、自動再Accept 0。SourceとGrant/Review/Query bindingの独立epochは未接続。 |
| **並列M3 W1 / [#2833](https://github.com/hat47x/tei-platform/pull/2833)** | **Open Draft**、更新HEAD `b99dd33dd85752eea757feb3cd8ec7d540491d9b`（旧`eee5ac7`から進行）。W1は**旧HEADを独立取得してNode 24.21.0で合成8/8 PASS、範囲内の明確な欠陥なし**とPR本文が報告。新HEADに専用`tei-studio-evidence-view-r0-ci.yml`が追加され、[EV-R0専用CI #42055355800](https://github.com/hat47x/tei-platform/actions/runs/42055355800)の`ev-r0-node`が**success**。GitHub当該HEADのcheck-runs 11件もすべてcompleted/success。 | **局所合成Projectionの試験成功**と**実Policy/Grant/Producer/Consumer/DOM/ARIA/Tool/Network非干渉/Owner意味受理**は別。W1の独立技術レビューと専用CI成功は証拠があるが**採否・main統合は未確認**。`prepared-public-fixture`は偽造可能で認可ではない。現`project.mjs`は`EffectUnknown`相当の`unknown` dispositionを扱わないため、実Provider結果を`failed`/`unobserved`へ丸めず接続前に契約調整（#2564/#1174等のOwner）。 |
| **P2 M4：Script-first編集・図面** | #1124のauthoritative Script/Markdown/表→generic標準Projectionは現行方針。[11/07三視点・CRUD提案](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6704500042)は、可逆な図上操作1件をPreview→Source→reprojectへ往復させる**P0製品受入追加案（未採用・NOT RUN）**。[同日複数Canvas提案](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6707668340)も設計/局所Pressureである。 | 旧Script-first冒頭受入を無断上書きせず、Owner採用後に三視点Diagram/CRUDとView Occurrence/同じSemantic Subjectの検収へ編入。初期M0→M1小HarnessのDone条件へ押し込まない。No-op Save byte不変・Source span typed edit/Acceptance/reparseの実Studio一往復は未証明。 |
| **並列#2391 E0→E3 / 後続E4** | E0要件→E1既存基盤→E2 CUI/Plugin→E3 #1180 Scenario/Driver/Provider/独立Read→E4 opt-in実UIの順。既存[#2853](https://github.com/hat47x/tei-platform/pull/2853)実PG Product Invoker、[#2854](https://github.com/hat47x/tei-platform/pull/2854)別ABA反例、[#2857](https://github.com/hat47x/tei-platform/pull/2857)暫定Web→PG E2Eは**限定実績**。 | #1180保存Scenarioから同一Attemptの製品E3正式受入は未証明。#2857監査は`INCOMPLETE/UNQUALIFIED`。E4実UI/OS/Network・Test/Prod二Build・Production専用外部制御面の物理除去/起動artifact identityは未実証。業務Target MCP（#2093）と専用UI制御MCPを分け、**TEI独自Host/Wails/MCPを基盤の必須条件にしない**。 |

### B. 先に解消する実証ギャップ（Owner/境界を変えない）

1. **#1同一Attempt接続**：実CLI origin（実行物/Workspace/Source Revision・WorldFP/SourceIndexFP・Scope/Subject）→Adapter/Consumer受領→Initial/List操作→RPCRecorder全要求とInspect 0/1→表示Subjectを、同一fixtureと時点のEvidenceへ結ぶ。既存PoC Aの「CLI parity後にbrowser smokeを順次PASS」だけでは両者が同一CLI由来A/Jを操作した証明ではない。
2. **#4/#8独立反証**：#4はAとJ/異Scope/旧Sourceでの誤実行・guard抜け、#8は実Transportとtest adapter・初期Auto selection・RPC全件・BrowserContext差をレビューし、#1候補HEADと独立Expectedを照合。未レビューの場合はunverifiedのまま。
3. **PR2/PR3を分離**：PR2の4入口実イベントが通ってからPR3でGraph render/qualified-selectionと旧DOM/Source epochを結線。CLI Scopeのない`InspectSubject(workspace,kind,id)`呼出しをCatalog内一意Guardだけで一般のSource/Policy/Authority保証へ昇格しない。
4. **#2833の合成CIを正しく反映**：旧§37「専用8件CI実行未証明」は現在は訂正され、**新HEAD dedicated CI success**が確認済み。一方実Disclosure/Effect causal join/Recipient受領とOwner受理は引き続き未実証。実DB `unknown`を合成Presenterへ誤変換しない。
5. **他レーンの競合回避**：[Draft #3117](https://github.com/hat47x/tei-platform/pull/3117)はStudio製品Bootstrap `cmd/tei-studio/assets/main.js`のD0後着防止候補であり、`reference/tei-studio-workbench/main.ts`のM0/M1検証そのものではない。[Draft #3303](https://github.com/hat47x/tei-platform/pull/3303)の三視点図面・Representation計画、[Draft #3301](https://github.com/hat47x/tei-platform/pull/3301)のPlugin-first inspector D1も独立Owner。Studio/Headless間の同じCLI・Provider意味を実験では共有しても、第二Consumer/Policy/Gateを新設しない。

**資格・実行宣言**：この§38は2026-11-07固定main/PR/Issue/Workbenchソース/Actionsの**GitHub照合による計画更新**であり、今回この節のためにGo/Node/Browser/WebView2/実PGを実行したものではない。#1のM0→M1固有PR、#4/#8固定レビュー結果、PR2/PR3実E2EはGitHub検索・既知の他チャット引継ぎでは確認できない。欠落を失敗確定や不存在とは断定しない。実Provider効果・Authorization・Owner意味受理は既存Ownerの証拠と判断を要する。


## 39. 2026-11-08 #1環境未確認・#3117 CI変更経路・#2833 W1/W2の資格更新

<!-- tei-studio-m0-m4-e0-e4-3117-evr0-w2-20261108 -->

**照合時点**：main [`7cd70eaee39e608fd0401649bb4c68a7b1d7b1e4`](https://github.com/hat47x/tei-platform/commit/7cd70eaee39e608fd0401649bb4c68a7b1d7b1e4)（2026-11-08 01:48:27 UTC）。§38は11/07固定版の履歴として保存し、本節の主張と混同しない。既存[Draft #3317](https://github.com/hat47x/tei-platform/pull/3317)のroadmapファイル**のみ**を追記。#1124はScript-first / Everything is Dataの同一Consumer一往復、#2391はE0→E3 Headless先行・E4後続opt-in、ADR-0043による**Host非必須**を維持。

### A. 更新版 M0〜M4 / E0〜E4の実績・担当・未実証

| Gate／担当 | 実装済み・実行記録（限定） | 提案・担当成果物と残る未実証 |
| --- | --- | --- |
| **M0→M1 P0：#1** | [merged #2626](https://github.com/hat47x/tei-platform/pull/2626)実CLI Source/Trace、[merged #2583](https://github.com/hat47x/tei-platform/pull/2583)Catalog global ambiguity保持と共通Guard、[merged #2457](https://github.com/hat47x/tei-platform/pull/2457)qualified-selection独立Reference。既存PoC A `cli_catalog_parity.py` / `browser_smoke.py` / `browser_isolation_smoke.py` は再利用候補。 | **#1の環境が稼働・利用可能か、#1が実装を開始/完了したか、専用branch/PR/実行ログは未確認**。環境不在/timeoutを「実装失敗」と断定しない。最小成果は実CLI由来A(異Scope同kind/idの曖昧)・J(独立unique)の**same-attempt input→同一Workbench Initial/List実イベント→Inspect全要求数0/1と引数→表示**。初期選択順をfixtureで固定し、実CLI出力の正真性/読取version/SourceIndexとTest adapterを別記する。 |
| **M1 PR2 / #4・#8独立レビュー** | 現main `reference/tei-studio-workbench/main.ts` blob `9582c85db4c27c1f89d8594f5b212984d755c9c0`にはGraph/List/Initial/Command→共通`inspect()`の構造配線と`inspectEligibility`がある。 | #4はA/J異Scope・Source locator・Authority/Policy・初期選択・後着Inspectの負例を、#8はCLI/Browser同一Target・RPCRecorderの**全要求数**・実Wails RPC vs Test adapterを独立点検する。**#4/#8の固定HEAD査読完了・実GUI上での4入口0/1は未確認**。構造配線を実イベントPASSとしない。 |
| **M1 PR3 / qualified Graph** | #2457独立`publishGraph`は統合済み。現mainの`main.ts`は`graph.update(related.input)`であり、そのReferenceの実import/結線はない。 | Source/scope/render epoch一致、G2拒否後のG1旧DOM/ARIA click拒否、R10→R11 Source移動・後着を**0 Inspect/0誤Source navigate**で確認。別表示世代・選択epoch・Source現在性・Grant現在性を混同しない。**NOT RUN**。 |
| **M2 / Source・Decision・資格失効** | #267既存Shared Acceptance、#2626 Source移動、[Draft #2829](https://github.com/hat47x/tei-platform/pull/2829)合成Review ABAは別Ownerの証拠。 | 既表示失効、後着抑止、Source/Decision ledger別readback、部分成功`accepted+sourceChanged=true`後の自動再Accept 0、Policy/Grant/Review/Source epochを別測定。Studio製品での縦断は**未実証**。 |
| **M3 / W1・W2と[Draft #2833](https://github.com/hat47x/tei-platform/pull/2833)** | #2833は**Open Draft / HEAD `b99dd33dd85752eea757feb3cd8ec7d540491d9b`**。W1の旧HEAD独立Node 24.21.0 **8/8 PASS** はPR本文の検証記録。新HEAD [専用EV-R0 CI #42055355800](https://github.com/hat47x/tei-platform/actions/runs/42055355800)の`ev-r0-node`を含む11 check-runsは**completed/success**（合成8件、exact-HEAD）。 | **W2独立担当・実施・結果の記録は今回のPR会話/API/他チャット文脈では未確認**。W1をW2二者レビュー完了へ読み替えない。合成`prepared-public-fixture`はGrant/Policyではない。実Source/Decision/Artifact Reader/実Provider、二世界DOM/ARIA/Tool JSON/Network、Recipient受領、Owner受理は**未実証**。実Provider `EffectUnknown`を現Presenterで`failed`/`unobserved`へ丸めない。 |
| **M4 / Script-first編集・三視点** | #1124のSource/standard Projection/typed Preview/Shared Acceptanceと、11/07の三視点Graph/CRUD/Canvas追加案は区分済み。 | 正確なSource span/Revisionに結び付くtyped change→Preview→#267 Acceptance→実Source再parse/reproject、No-op Save byte不変は**NOT RUN**。図上編集/複数Canvas P0追加は[11/07 Issue追補](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6704500042)時点の**設計提案・未採用**で、#1の最小Harnessには追加しない。 |
| **#2391 E0→E2（別Owner並列）** | 既存CLI/標準Operation/Role/Plugin/Scenario一式の局所Referenceあり。 | 外部要件圧力→CUI＋標準Plugin→Provider/Grant/Bindingの同一実Target適合、Owner採用は未実証。Draft #2492を採用済みmainとは呼ばない。 |
| **#2391 E3（別Owner並列）** | [merged #2853](https://github.com/hat47x/tei-platform/pull/2853)製品Invoker→実PG、[merged #2854](https://github.com/hat47x/tei-platform/pull/2854)別Reference ABA、[merged #2857](https://github.com/hat47x/tei-platform/pull/2857)暫定Web→実PG一改訂E2Eの**対象限定結果**。 | #1180保存Scenario→Operation/Driver→Provider/Effect/Receipt→独立Readの同一Attempt／正式Grant・Owner受理は未実証。#2857監査は`INCOMPLETE/UNQUALIFIED`。 |
| **#2391 E4（後続・opt-in）** | Web/Desktop及び#1824の局所Browser/Bridge smokeは存在。 | 外部UI act/observe→独立業務Read、実OS/Network開示、Test/Production配布実byteと起動Artifact同一性、**Test専用制御面のProduction物理除去**は未実証。通常UI/a11yは維持。業務用MCP Target #2093とUIテスト用制御MCPを分離し、Host/Wails/MCPを必須としない。 |

### B. #3117のCI重複についての固定事実・割当

- [Draft #3117](https://github.com/hat47x/tei-platform/pull/3117) **open、HEAD `b75df7a54a27a540e51c57778fd173213752491f`、現main比較 ahead 5 / behind 728**。旧#3036は**closed・not merged**。変更5パス：`.github/workflows/tei-studio-async-lanes-ci.yml`、`.github/workflows/tei-studio-wails-poc-a-ci.yml`、`cmd/tei-studio/assets/main.js`、`cmd/tei-studio/test/bootstrap_async_smoke.py`、`cmd/tei-studio/test/main-async-scope.test.mjs`。
- #3117 exact HEAD のGitHub check-runs **13/13 completed/success**：専用Node、Server/Chromium Bootstrap、Windows WebView2/生成Workbench smoke、Repo Health等。これらは**Bootstrap `/`のD0非同期失効**という限定機能の試験であり、Desktop `/workbench/`にある**別`reference/tei-studio-workbench/main.ts`**のM0→M1 A/J + Inspect Spy E2Eではない。UI response discardは既Effectの取消し/本番Grantでもない。
- **変更パスの重複リスク**：#1が`.github/workflows/tei-studio-wails-poc-a-ci.yml`や`cmd/tei-studio/test/**`に追記するなら、#3117の変更範囲と作業分担を先に明示する。現時点、`.github/workflows/tei-studio-wails-poc-a-ci.yml`の**current main blobと#3117 head blobは双方 `a7e2498ab8a1eaf7afbe4e763561dbb5115e9e94` で同内容**。従って**現在の同ファイル内容競合は確認されず**、単にpathが重なる可能性と現在の差分を混同しない。
- #1がまず既存の`reference/tei-studio-workbench/**`と独立の小テスト/adapter/Recorder拡張で受入できるなら、**#3117のBootstrapコード/CIに干渉しない**のが望ましい。Wails PoC A既存CIへ同一Workbench実DOM受入を追加する必要がある場合だけ、現行mainの正規Workflowを基線に#3117 Ownerと差分を一本化し、同じJob/Pathを別PRで二重実装しない。#3117のCI Greenを#1の実CLI由来A/J同一試行PASSへ流用しない。

### C. 判定条件と未実証宣言

- **最初の次の成果**：#1の環境・担当branch/PR・固定SHAの所在が確認できること。未稼働なら実装可否の具体的障害を明記し、代替の別Harnessを無断で作らない。次に同一CLI出力A/J→Workbench Initial/List→Inspect 0/1→表示と#4/#8の独立負例を**同じAttempt**に束ねる。
- **レビュー条件**：#4/#8は当該#1 HEAD、W1は#2833旧HEAD独立結果、W2は**未確認**をそれぞれ区別する。未報告を不合格とせず、レビュー結果や修正PRが確認されるまで正式な受入済みとはしない。
- **この追補の作業資格**：GitHub Issues/PR/mainソース/CIチェックの再照合とロードマップ文書変更。**今回新規にGo/Node/Browser/Wails/WebView2/実PG/Source編集・OS開示・実Policyを実行していない**。既存Owner/Authority/#1180 Scenario・#1119 Gate、TargetごとのHost optionalを維持し、実装PR/Reviewerを重複させない。


## 40. 2026-11-08 #1/#3/#9 idle・#6 systemError後の担当再整理とP0 Blocker

<!-- tei-studio-20261108-idle-owners-blocker-and-draft-ci-roadmap -->

**固定時点**：GitHub main [`7cd70eaee39e608fd0401649bb4c68a7b1d7b1e4`](https://github.com/hat47x/tei-platform/commit/7cd70eaee39e608fd0401649bb4c68a7b1d7b1e4)（2026-11-08 01:48:27 UTC）。以下の「#1/#3/#9 idle・直近回答なし、#6 systemError」は**依頼者からのチャット作業状態の報告**であり、GitHubで検証可能なbranchやCI状態ではない。**未回答≠未着手・失敗・担当解除**。#6の既存失敗チャットから未確認の作業成果を引き継いだと主張しない。§38/§39を過去の時点付き記録として維持し、現況は本節を優先する。

| 優先・レーン | 現況：確認済み実装・CIと確認できないこと | 担当維持／次の成果・停止線 |
| --- | --- | --- |
| **P0 / #1124 M0→M1** | mainに[merged #2583](https://github.com/hat47x/tei-platform/pull/2583) ambiguity guard、[merged #2457](https://github.com/hat47x/tei-platform/pull/2457) qualified-selection Reference、[merged #2626](https://github.com/hat47x/tei-platform/pull/2626)実CLI Source移動の限定証拠。現`reference/tei-studio-workbench/main.ts`はblob `9582c85db4c27c1f89d8594f5b212984d755c9c0`、共通`inspectEligibility`と4入口コード配線あり。**#1の実装環境の利用可否、branch/PR、同じCLI実出力A/J→同一Workbench操作の完了報告は確認できず**。 | **#1を実装Ownerとして保持（連絡未取得）**。最小受入は同一CLI attemptのSource/Scope/snapshotから曖昧A→Initial/List Inspect **0**、無関係unique J→Inspect **1**・引数/表示一致、RPCRecorder全要求数・初期選択順・実Transport vs Test adapterを明記した固定HEAD証拠。環境が不明なら状況確認を第一作業とし、重複Harnessを別担当が無断作成しない。 |
| **独立レビュー / #4・#8** | 既存#1824 CLI parity/Browser smoke/二BrowserContext/RPCRecorder等の局所証拠は再利用可能。#1の新候補HEADへ紐付くレビュー記録は未確認。 | **#4**＝Scope/Source identity/Authority、異Scope同kind/idとA/J・遅延応答の負例。**#8**＝実CLI・実Workbench・実RPC vs Test adapter、Recorder全要求とfixture初期選択の証拠。両者は#1のbranch/固定HEADが分かるまで**独立暫定Expected作成まで**、#1実装完了・PASSやOwner受理を代行宣言しない。 |
| **待機の#3/#9、障害の#6** | 依頼者報告は **#3/#9 idle・直近回答なし／#6 systemError**。各チャットの最新実行環境・担当branch/PRをGitHub状態から断定できない。 | **#3/#9を#1の代替実装Ownerに自動変更しない**。応答再開時は既存HEAD/担当と未実証の差分を確認して、#4/#8レビュー、PR2/PR3、M2/M3等の既存非重複小作業に限る。**#6のsystemErrorをクリティカルパスの成果に算入しない**；再依頼/引継ぎには固定証拠が必要。 |
| **M1 PR2→PR3** | 4入口の構造配線は既存。独立Graph `publishGraph`はmainのWorkbenchへ未接続。 | PR2＝Graph/List/Initial/Commandの**実イベント・同一Guard・Inspect 0/1**、PR3＝qualified rendering epoch・旧G1 DOM/ARIAクリック拒否とSource/Policy別鮮度。どちらも**実Workbench製品E2E未実証**。 |
| **近接Draft：[#3117](https://github.com/hat47x/tei-platform/pull/3117)** | **Open Draft HEAD `b75df7a54a27a540e51c57778fd173213752491f`**、main比 **ahead5/behind728**、5パス変更（`cmd/tei-studio/assets/main.js`とBootstrap試験、2 workflow）。exact-HEAD GitHub **13/13 check-runs success**。`.github/workflows/tei-studio-wails-poc-a-ci.yml`は**mainと#3117 headで同一blob `a7e2498ab8a1eaf7afbe4e763561dbb5115e9e94`**。旧#3036はclosed/not merged。 | #3117は**Bootstrap `/`のD0旧Promise/finally抑止**担当。Desktop `/workbench/`の独立ConsumerとM0→M1 A/Jではない。#1が同じ`cmd/tei-studio/test/**`・PoC A workflowを触る場合は**CI job/編集Ownerを先に一本化**。同一path潜在重複≠現時点のworkflow内容コンフリクト。#3117のCI成功を#1の受入へ移転しない。 |
| **近接Draft：[#3327](https://github.com/hat47x/tei-platform/pull/3327) / [#3378](https://github.com/hat47x/tei-platform/pull/3378) / [#3381](https://github.com/hat47x/tei-platform/pull/3381)** | #3327は同Workspace A→B→Aで旧Inspect成功/失敗の採用を拒否する**Responses Scope局所テスト**。HEAD `dd2c782d4127405e86c04ebca71e88b7db5873be`のチェック12件中**`build-and-match-embedded-bundle`がfailure**（他11件success）；作成者報告の局所2/2を当該HEADの全CI PASSとは呼ばない。#3378はCLI/Studio直結`operations.InspectSubject`のshadow parity、#3381はPlugin Inspector typed注入口のDraft。 | M2への局所再利用は可能だが、#3327のCI失敗を先に切り分ける。#3378のdirect-Go Service一致は実Browser/CLI→Workbench同一attemptでなく、#3381は実選択Providerの起動証拠ではない。**これらを#1のM0→M1完了の代替にしない**。各既存Owner/PRを尊重。 |
| **M2→M4** | M2の#267 Shared Acceptance/Source ledger別readback、M3のReview/Policy/Disclosure、M4のScript-first Source span→typed change→Preview→Accepted→reparseは独立契約として存在。#1124直近[三視点/CRUD/同型Relation・複数Canvas追加](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6710684306)は**設計・局所Referenceであり製品受入未実証**。 | M2＝Source/Decision ledger・Grant/Source epochと後着別失効、部分成功の自動再Accept **0**。M3＝合法public sliceの実Consumer表示と現在Disclosure。M4＝可逆なSource逆編集・Source fidelity/Graph＋CRUDへの再Projection。**実Studio／実Owner受理はNOT RUN**。 |
| **M3 / [#2833](https://github.com/hat47x/tei-platform/pull/2833) W1/W2** | **Open Draft HEAD `b99dd33dd85752eea757feb3cd8ec7d540491d9b`**、main比 **ahead5/behind1610**、4パス変更。W1旧HEAD独立Node24.21.0 **8/8 PASS**はPR本文報告、新HEADの[専用EV-R0 CI #42055355800](https://github.com/hat47x/tei-platform/actions/runs/42055355800)を含む**11/11 checks success**をAPIで確認。PR上の独立**W2査読結果は未確認**。 | W1/専用CIは**合成read-only四レーン表示**限定で、`prepared-public-fixture`は認可でない。W2にはPR現HEADの4ファイル・専用CI・二世界generic fallback・`unknown` Effectの未対応を別途評価してもらう余地があるが、**W2に指示/完了したとは主張しない**。mainへ自動mergeしない。実Policy/Grant/SHO/AI/Recipient/Owner meaningは未実証。 |
| **#2391 E0→E4 / Headless-first並行** | E0設計圧力、E1標準構造/Operation/Role、E2 CUI/Plugin、E3 [merged #2853](https://github.com/hat47x/tei-platform/pull/2853)実PG Invoker・[#2854](https://github.com/hat47x/tei-platform/pull/2854)ABA別Reference・[#2857](https://github.com/hat47x/tei-platform/pull/2857)暫定Web→PGの**限定実測**あり。 | E3の#1180保存Scenario→Driver→実Provider/Receipt→独立Read同一Attemptは**未実証**、#2857は`INCOMPLETE/UNQUALIFIED`。E4実UI/OS/Networkの業務効果・Test/Production artifact同一性、Test用制御面**物理除去**は**NOT RUN**。E4をE0〜E3の前提にせず、Targetに固有Host/Wails/MCPの必須化なし。 |

**最短の担当・blocker線**：**B0＝#1の稼働環境・branch/固定HEAD/進捗返信が不明** → **B1＝実CLI A/J→同じWorkbench Initial/List→RPC Recorder 0/1の同一Attempt Evidenceが無い** → **B2＝#4/#8独立反証・PR2実イベント** → **B3＝PR3旧Graph DOM拒否**。これらは**Studio直接レーン**。#3117のBootstrap、#3327の局所ABA、#2833 W1/W2のM3合成表示、#2391 E0〜E3は**独立並列**（各成果をM0→M1 PASSに加算しない）。#3/#9のidleと#6 systemErrorは**作業調整リスク**だが、具体的な未実行結果が確定するまでは新たな技術的FAILとは扱わない。

**確認の性質**：GitHub現行main/Issue/PR/CLI・Workbench関連ソースとexact-HEAD CI status、既存ロードマップのみを照合。**今回Go/Node/Browser/WebView2/実PG/外部Tool/Disclosureの新規実行なし**。#1/#3/#9/#6の各チャットの内部環境は直接観測していないため、依頼者報告とGitHubで確認できた事実を区別。正式Owner意味/Authority/実Effectsの採用を変更せず、**第二Host/Policy/Scenario/Rendererを新設しない**。

---

<!-- tei-cp-cross-chat-convergence-20261113 -->
## 2026-11-13 横断収束：Studio自己再構築を中心にした作業順と検収

**位置づけ**：他チャットのDSL、Plugin、X-TEA移管、DB、GUI/CUI、生成AI協業、テストの検討を、既存Issue間の合流条件として整理する。新しい契約・実行基盤・Gate・意味受理の仕組みを採用するものではない。G0〜G4と各Ownerは従来どおりとし、[Desktop Self-build #4583](https://github.com/hat47x/tei-platform/issues/4583)、[Plugin役割分離 Draft #4601](https://github.com/hat47x/tei-platform/pull/4601)、[Hook設計 #4589](https://github.com/hat47x/tei-platform/pull/4589)などの個別仕様を複写しない。

### 優先順位と並行作業

1. **P0-A／本線：Studio Desktopの自己再構築**（#2831／#4583／#1824）。N0 Inspector生成、N1 Go/TS生成、N2 Node/esbuild/embed、N3 nested Go/Wails Windows exeの各工程で、宣言されたBuild Intentと実際に消費したSource、実Invoker、生成物を照合する。既存Build Execution/ArtifactSetを使い、Desktop exeをPlugin Subjectと偽装しない。Stage1のTEI実ProcessがStage2 Desktopを作ったこと、そのStage2を実WebView2が使用したことを別々に確認する。
2. **P0-B／並行：PluginとX-TEA由来の三要素分析法モデル**（#1251／#951／#1016／#1124）。通常Studioと生成Studioで、選択したProvider、実Link・実呼出し、成果物への同梱、未選択Providerの非混入を検証する。Build時GeneratorとRuntime Providerは別の役割とする。[Draft #4570](https://github.com/hat47x/tei-platform/pull/4570)のread-only三要素Pluginを初例とし、実X-TEA原典、Modeler編集、DB更新は後続の独立受入に置く。
3. **P0-C／並行：人間と生成AIの協業**（#3262／#4515／#1120／#1124／#1180／#2514）。GUI/CUIは同じ型付きOperationとread-only設計Queryの異なる入口とする。Subjectの論理Identity、Source内の出現位置、Revision、固定選択、表示Projection、取得できた範囲を混同しない。AIの提案はSourceへ戻して通常の開発レビュー、Validation、再投影で扱う。形式的な多重承認は増やさない。OKF等で文書を根拠として参照可能にしても、推測を業務意味へ自動昇格させない。
4. **P1／合流後の拡張**（#4359／#2620／#2586／#3270／#1129／#1174／#4198／#2600）。DSL糖衣・自己記述、実Provider Read、DB方言とMigration、Hook・Receipt、可変Lifecycle、増分再利用は、それぞれ既存契約の小さな縦断例で検証する。全機能の完成をP0-Aの前提としない。

### 共通の反証条件

| 境界 | 検収する実物 | 合格としない例 | Owner |
| --- | --- | --- | --- |
| Build → Desktop | N0〜N3の実入力・実Builder・Artifact、Stage1→Stage2、実WebView2 | 別HEADのGreen合算、Plan外esbuild、Stage0による代行生成、古いJS | #4583／#2831／#1824 |
| Plugin → 実Studio | 選択・実呼出し・実収録、通常版／生成版の両方 | B選択でA呼出し、未選択Provider混入、静的画面のみ | #1251／#951／#1016 |
| 人間／AI → Source | Required情報・対象Identity・Revision・pinの再解決 | 参照0件を対象不存在と誤認、重複IDを勝手に接続、A→B→Aで旧選択を再利用 | #3262／#4515／#1120 |
| X-TEA → TEI | 三要素分析法の原典忠実性、宣言CRUDと業務Actionの粒度差、手作業 | Function-level CRUDを業務実作用と断定、汎用Graphへ勝手に意味を追加 | #950／#1031／#1124／#1132 |
| Hook → Effect | Guardの前後、実Call回数、Commit不明・Receipt・独立Read | Guard拒否前の副作用、二重Call、lookup-first消失 | #1129／#1174／#4198／#2514 |
| Provider → Consumer | Source、selected/actual Provider、Rowset、Result、完全受領 | SQL生成だけで実Readを主張、0行と未取得を混同、途中切断をcomplete | #2586／#2509／#2515 |

検収票ではSource/HEAD、必要な入力の閉包、実Invoker、Artifact、実Consumer、独立Expectedを分けて記録する。Stage1とStage2は別の実行でも、因果参照と対象の版を追跡できること。PASS、FAIL、UNKNOWN、NOT RUN、unsupportedを混同しない。生成器やSourceが変わったときも、確認できた依存範囲だけを再検証し、無関係な対象まで自動失効させない。

### 現在の証拠と最短の次工程

[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)の固定HEAD 708ca27e、[Windows CI #43385876034](https://github.com/hat47x/tei-platform/actions/runs/43385876034)では、N0 Inspectorの実再生成、使用TSのbyte一致、通常版・生成版WebView2 Jobの成功を限定的に確認できる。ただし、**N0〜N3の全工程をTEI Build Intentが駆動したB1、Stage1実TEIによるStage2 Desktopの構築B2、そのStage2自身の実Consumer B3は未成立**。別のPR HEADへCI成功を継承しない。

着手順は #4583 B1（実入力閉包）→ B2（Stage1→Stage2）→ B3（実WebView2 Consumer）。その後に非Builtin Providerの通常／生成Studio両系統、固定対象の再解決、三要素モデルの実原典適合を段階的に合流する。並行担当は独立Expectedと負例を先に準備してよい。ここで行ったのは既存GitHub記録の静的整理のみ。新Go/Node/Windows/DB試験とOwner意味受理は行っておらず、製品全体G4は未資格。
---

<!-- tei-cp-crosschat-second-pass-20261113 -->
## 2026-11-13 第2次横断レビュー：実消費の証拠、Pin非干渉、Providerの実同梱

前節のP0-A/B/Cの優先順位を変えず、他チャットの実コードレビューから具体化した**最初の失敗例と実装順**を補う。新しいBuild Manager、Source Registry、承認Engine、Hostを作らない。各担当Issueで受入を管理し、この節は接合点だけを扱う。

### A. Build入力：事後ハッシュと実際の消費を区別する

[#4583](https://github.com/hat47x/tei-platform/issues/4583)のB1-Bで、[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)の `b1-stage-input-observation.json` は、ビルド後に指定ファイルを再読取した目録であり、esbuildやGo Builderがそのbyteを実際に消費した証明ではないと確認された。現状の `qualifiedBuildIntent=false`、`dependencyClosure=unknown` を維持する。

最初の一本は **N0生成Inspector TS → N2 esbuild → N3 Go embed** に限定する。各工程の直前に入力・Toolchain・実Invokerを固定し、出力と次工程の入力を同じAttemptに対応づける。工程後の変更も照合するが、pre/post一致だけでTOCTOUやA→B→Aが解消したとは扱わない。実消費の完全な証拠が必要な場合は不変SnapshotやOS読取境界の実証を別資格とする。

独立Expectedは、(1)esbuild直前のTS差替え、(2)Go build直前のJS/Go差替え、(3)ビルド後差替えによる事後目録の偽装、(4)未宣言の推移的import、(5)選択Builder Bと実Invoker Aの不一致。実験できない境界はunsupported/UNKNOWNと記録する。Desktop exeをPlugin Subjectと偽装せず、既存Execution→ArtifactSet→Subject Bindingの適合から始める。Ownerは#4583/#2831、増分失効は#2600。

### B. Pin再確認：画面のInspectと独立した読取経路

[#4515](https://github.com/hat47x/tei-platform/issues/4515)で確認された現行 `inspect(subject)` は、応答と同時にselected/Editor/Graph/Noticeを更新する。Pin Aの資格再確認に同じ入口を用いると、利用者がBを閲覧中でもAへ表示が戻る。**Pinの再確認は画面選択を変更しない**ことを受入条件にする。

既存 `client.inspectSubject`/`readInspection` を再利用し、`pinned-inspection` laneでPin Observationだけを更新する。Pin/Unpin/Undo/Workspace切替でローカル世代を更新し、応答はlaneの最新性、captured generation/epoch、qualified Subjectの一致で採用する。同じSubjectをUnpin→再PinしたA→Aでも旧応答を拒否する。ローカル世代はSource Revision/Grant/Disclosureの現在性を証明しないため、製品側の再資格化を別に行う。

[Draft #4597](https://github.com/hat47x/tei-platform/pull/4597)はPinの純粋状態遷移に限定される。最初に既存responsesテスト、次にPin状態テスト、最後に実WorkbenchでEditor/Graph非干渉と生成Inspector使用を検証する。Pin状態テストだけでDesktop/Source/Grantの受入へ昇格させない。Ownerは#4515/#1824。

### C. Plugin構成：選択と実収録を両方向から確認する

[#4583のX-TEA横断引渡し](https://github.com/hat47x/tei-platform/issues/4583)と[Draft #4601](https://github.com/hat47x/tei-platform/pull/4601)に従い、Build-time GeneratorとStage2 Runtime Providerを区別する。最初の非Builtin例は[Draft #4570](https://github.com/hat47x/tei-platform/pull/4570)の読取り専用三要素モデルとする。

必要なProviderが**選択されたのにStage2から欠落**した場合と、**未選択なのにimport/init/embedから混入**した場合を対で検査する。Build Profile、依存Go Source、nested Studio module/tags、実Wails Artifact、実Bridge呼出しを同じ選択に結び付ける。通常Studioの静的importやB0/N0のWebView2成功だけでは、生成Stage2がX-TEA Providerを実行したことにはならない。X-TEA全Function・実原典・SHO/KON/他Targetはこの最小試験の前提にしない。Ownerは#1251/#951/#1016/#1124、Stage2への結合は#4583。

### D. 最小の協業・dogfoodingシナリオ

開発者と生成AIがCLI/CUIでSubjectを特定し、Source位置と参照・被参照を確認する。Studioでは人間がBを表示したままAをPinし、AIがAの根拠を再確認してもBの表示を変更しない。AIの提案はSourceに戻し、通常のレビュー、Validation、再読込、再Projectionで確認する。参照0件・対象不存在・partial/unavailableを区別する。OKFによる文書管理は根拠の可用性を高める手段であり、業務意味の自動承認にはしない。

この協業シナリオとStage2のBuild/Provider検証は**別のAttempt/別の資格**であり、相互の成功を合算しない。実装順は B1の実消費観測 → Pin非干渉の実Workbench接続 → B2/B3の実Stage2 → Optional Provider実呼出し。CLIの実PostgreSQL Read (#2586/#1251) は並行して進め、Desktop完成を待つHard Dependencyとしない。

**検証状況**：本追補は既存GitHub Issue/PRのレビュー内容の計画反映のみ。新しいGo/Node/Windows/PostgreSQL実行はNOT RUN。既存Draftの最新HEAD CIを本追補で追試したものではなく、B1〜B3、全体G4、Ownerによる意味受理は未資格。
