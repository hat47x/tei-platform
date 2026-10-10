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
