# 三視点ローコード実証のクリティカルパスと受入シナリオ

- 状態: **設計・実行計画候補**。この文書の作成はコード実装・実行・製品品質の成立を意味しない。
- 親Issue: [#1119](https://github.com/hat47x/tei-platform/issues/1119)
- 基点: 2026-10-10時点の既存Issue/PR・Repository文書による調査。進行中PRや実行結果は着手時に再取得する。
- 関係: [Product triage #982](https://github.com/hat47x/tei-platform/issues/982)、[三視点業務設計の既存方針](../docs/business-system-design.md)、[現在の実装状況](../docs/status.md)、[Requirement gap audit](PRODUCT_REQUIREMENT_GAP_AUDIT.md)。

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

Consumer: #1126、#1129、#1132。最小条件:

- Datasetは型・参照・Revisionへ結び付く。Snapshotは具体的な状態であり、Model定義そのものではない。
- Stage / Presentation order / 実際のAction execution orderを混同しない。
- 未観測の手作業・外部効果を成功とせず、Scenario fixtureの合成結果をactual Provider Evidenceに格上げしない。

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
4. `tested` / `not-run` / `needs-trusted-checkout` / `actual-target-observed` を明確に区別し、Draft PRのまま未検証統合しない。
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
- `docs/status.md`、`plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md`、`docs/business-system-design.md`を既存の製品判断の基線として参照した。今回確認したsource anchorは `internal/ir/world.go`、`internal/plugin/model.go`、`tools/reference-shared-acceptance/README.md`、`tools/repository-health-check.sh`。pathがあることとAPIの適合・actual PASSは等価ではない。
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
- 本ファイルのrelative link 3件の解決先 `docs/business-system-design.md`、`docs/status.md`、`plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md` は取得できた。

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
