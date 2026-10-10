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
| G2 — P0 product loop | V00〜V08のうち採用したP0必須セットとN01〜N12の該当負例を、同一のmeaning revisionに結び付けて再現 | `product-loop-unverified` |
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
- **wave 3（P0製品一往復）**: #1124がC0/C1を使ってV03→V04、#1127がC3と既存Driverを使ってV05→V08へ連結する。Producerが異なるfixture revisionを検証しただけでは合格としない。同じAccepted meaning revisionを通す。
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

すべてのV00〜V08/N01〜N12は**設計上の担当あり・このプログラムとしての実行証跡なし**から始める。Issueごとの実証結果を受領した時点で該当caseのみ更新し、別RevisionでのPASSを横滑りさせない。G2の採否判定には、同じSource revision、Provider/Target identity、否定試験結果、別readによるObservationを束ねた受入packetが必要。

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

G2は単なる21ケースのチェック数ではなく、**一つの受注明細業務について、同一Accepted meaning revisionをSource・Projection・Action binding・Driver request・separate Observationまで追えるか**で判断する。最初のハッピーパスは `V00→V03→V04→V05→V08`、最少否定経路は `N05/N06`（Source）、`N08`（Capability/Authorization）、`N11`（Observation）とし、現時点の具体的な変更対象・実装可能なvalidation変更は#1124/#1127が既存Schemaと照合して確定させる。合成例に一回だけ一致したことは、実Targetの独立した成功観測ではない。
