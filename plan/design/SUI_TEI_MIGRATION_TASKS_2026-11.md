# SUI→TEI移行：read-only参照の実装知見と、*.tei.md＋Pluginへの移行タスク分解

- 状態: 評価待ち（未採用）。実装・Reference変更・受入の根拠にしない
- 日付: 2026-11-11（§5を具体化: 2026-11-11）
- 対象: SUI SensemakingのDocument/Card/Island/Tree/Canvas/Relation/Revision/Reviewと、TEIの*.tei.md（Canonical定義）＋Pluginによる設計〜実装への移行
- 正本との関係: [SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md](SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md)（SUI側の分割とscope isolation）と[SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md](SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md)（Stage 0のshadow conformance）を置き換えない。本書は「2026-11-11にSUI側でread-only参照が実装された知見」と、そこから展開した移行タスクの分解だけを扱う
- 根拠: SUI PR #3429（`GET /docs/{doc_id}/tei/sui-reference`）、SUI ADR-0116、SUI `02_Architecture/api.md` §2.18、SUI #3401
- 主張しないこと: TEIのCore・Plugin契約・`CapabilityKind`の変更を求めない。SUIがTEIへ移管されることを保証しない。第二の消費者が出るまでの標準化を主張しない

## 1. 本文書の役割

SUI側で「TEI Referenceのartifactを、型付きのread-only参照として読む」経路が実装されました（2026-11-11）。これは移行のStage 0に当たる実装であり、次の三つを初めて実コードで确定させます。

1. SUIの主要な構造が、TEI側の型へ落としたとき、どこまでが失われずに運べるか
2. 「運べない」が、部分取得・未レビュー・非公開・失効・版とどう区別して現れるか
3. 既存のSUI経路（DocumentV1）と比べて、参照経路にどの実利と代償があるか

本書はこの知見を、*.tei.md＋Pluginという移行先の構成へ翻訳し、今後必要なタスクを**後続の実装者がそのまま作業票にできる具体性**まで分解します。判断の採用は各IssueとADRの仕事で、本書は分解と論点の整理までを行います。

## 2. 実装知見（2026-11-11時点）

### 2.1 動いたもの

SUIは、運用者が置いたTEI Reference artifact（`tai.reference.tei-documentv1-projection/v0`）を、局所I/Oのtransport（ADR-0101 Option A）と文書単位の紐付け（ADR-0102 案B）から読み、`tai.reference.sui-document-reference/v0` へ型付けして返します。返すのは識別子・関係の形・review状態・canvas座標・版の指紋・損失の明示だけで、本文・題名・要約・review attributionは返しません。`acceptedMeaning` は常に偽で、Cardを読めることはTEIのAccepted Meaningではありません。

区別できているものは、同一名称のCard（IDで区別）、複数Islandへの所属（containmentとaffiliationを別のrelationとして保持）、有向Relation、多重Edge（Edge IDごとに保持し、同じfrom/to/typeの`parallel`群を明示）、未レビュー（参照は残し本文を伏せる）、部分取得（宣言されたcompletenessと理由）、entity単位の非公開（IDと件数を出さない）、Revision変更（内容変更とsource revision変更を別の状態として報告）、Source失効（`revoked`/`expired`で内容を返さない）、文書単位の開示制限（種別・ID・件数さえ返さない）です。

### 2.2 損失のinventory

artifactから型へ落とした時点で失われるものは、実装の中で`losses[]`として宣言されています。この一覧は、Canonical定義の設計で「表現できるようにする」か「対象外と宣言する」かの論点そのものです。

| 対象 | 現在の扱い | 移行先での論点 |
| --- | --- | --- |
| Tree/Forest | `unsupported`。containmentと`parentIslandId`をTree階層と呼ばない | 閲覧状態（折り畳み・選択・focus）を含むか。含めると第二のApplicationに不要な語彙が流入する |
| Canvas transform / z-order / zoom / gesture | `partial`。Cardのx/yのみ | 空間表現をCanonicalへ入れるか、native assetのままInteraction境界だけを提供するか（#993） |
| Island geometry（polygon/rect） | `partial` | 同上 |
| Review attribution（誰が・いつ） | `unsupported`。真偽フラグのみ | attributionをDataとして持つか、監査・Source側に委ねるか |
| Narrative / relation summary / critique | `unsupported` | 本文系の投影として#2516の開示判定と一体で扱うか |
| Relation rationale | `unsupported` | 根拠をAttributeに載せるか、Evidenceとして別Dataにするか |
| Document / Island / Card の題名・本文 | `unsupported`（内容として返さない） | #3369の判断。開示先ごとの方針（#2516）とreview軸を条件として外から与える形が妥当 |
| Source revision | `unresolved`（欠落時）。`documentDigest`は内容指紋でありSUIの保存revisionではない | 版の同一視を避ける。Source index/provenance（#2743）と条件付き保存（#2595）の対応 |
| Source状態（`active`/`revoked`/`expired`） | `unresolved`（欠落時）。`unknown`を`active`に読み替えない | 失効の権威をconsumer側に置くか、Source index側に置くか |

### 2.3 実測（合成1,000 Card、プロセス内、2026-11-11）

| 観点 | SUI既存方式（DocumentV1） | TEI連携の型付き参照 |
| --- | --- | --- |
| 構築時間 | 39 ms | 127 ms |
| payload | 350 KB | 903 KB |
| 本文バイト | 53,890 | 0 |
| 追加の文書内容DB読み取り | 文書本体を読む | 認可とartifact読み取りのみ |

参照は遅く、大きい。ただし本文を一切運ばず、未レビュー・非公開・失効・多重性を型で区別できます。したがって参照経路の用途は「識別と関係と状態を渡す」ことに限定し、速度を理由に既存経路を置き換えません。

### 2.4 読み取れたこと

- **F-1 型への写像は可能だが、内容の写像は別判断である。** 識別子・関係・状態は落とさずに運べる。本文・題名は「運べる」ではなく「運んでよいか」が未決で、そこを先に分離できた。
- **F-2 occurrence identity が本体である。** 同一名称・複数所属・有向・多重は、すべて「同じものに見える別のもの」を畳らないことで初めて区別できる。Canonical定義でIDをoccurrenceに束縛する設計が要る。
- **F-3 membershipは二種類ある。** containment（見た目の包含）とaffiliation（明示的・可逆・多対多）は同じ「所属」ではない。一つのRelationで畳ると情報が落ちる。
- **F-4 版と内容指紋は別物である。** `documentDigest`は内容の指紋で、保存revisionではない。「変わったか」は別の入力（source revision）と組にして初めて意味を持つ。
- **F-5 失効と欠落は同じに見えてはいけない。** Sourceが失効したとき、参照を返さないことと、artifactが無いことは別の失敗である。consumer側で区別しても、権威はproducer側に欲しい。
- **F-6 開示制限は粒度が違う。** entity単位の省略と文書単位の全面非返却は別で、後者は種別や件数も漏らせない。
- **F-7 未レビューは「無い」ではない。** 参照は残し、内容を伏せ、状態を型で告げる三つを同時に満たす必要がある。
- **F-8 部分取得は「完全」の反対ではなく「不明」の隣である。** 宣言が無い場合は`unknown`であり、`complete`へ読み替えない。
- **F-9 損失は宣言できる。** `unsupported`/`partial`/`unresolved`の三値で宣言する形は、consumerが欠落を推定から切り離せる。ただし損失の語彙はまだconsumer側のもので、共通契約ではない。
- **F-10 同じ構造を二度書くコストが乗る。** occurrenceごとに列挙するためpayloadは大きくなる。DB投影（#2488）と共通API（#1158/#3419）で解消できるかは未検証。
- **F-11 SUIの安全不変条件は、read-only参照でも維持できる。** SafeMode・未レビュー非公開・`human_reviewed`・proposal-onlyは、型を渡す経路では緩まない。逆に、渡す型を増やすほど「何を許すか」の判断が増える。
- **F-12 第二の消費者が無い。** 本PRはSUI一件の実装であり、標準化の根拠にはならない。#2497の判断基準（第二のApplicationで同じ形が再現されるか）は満たしていません。

## 3. SUI構造 → *.tei.md＋Plugin の対応表

「表現できる」「条件付き」「対象外」の三つで示します。対象外は、移行先でも持たないと宣言するものを意味します。

### 3.1 ドメイン

| SUI | TEI側の表現候補 | 状態 | 担当 |
| --- | --- | --- | --- |
| Document | `sui.document`（Data）と、record/string投影（#2488） | 定義は可能、投影は未実装 | #2488 |
| Card | `sui.card`（Data）＋Attribute（text, claimType, holdState, x, y） | 表現できる。本文の開示は別軸 | #2516 |
| Island | `sui.island`（Data）＋containment Relation＋親島Relation | 表現できる。geometryは対象外 | #993 |
| Tree | hierarchyはRelationで書ける。閲覧状態（折り畳み・focus）は対象 | 条件付き | #2497 |
| Canvas | x/yはAttribute。polygon・placard・z-order・gestureはnative asset | 条件付き（Interaction境界のみ） | #993 |
| Relation | directed・多重を保つRelation。型はAttributeまたは別Data | 表現できる。rationaleは未 | #2497 |
| Revision | 内容指紋＋source revisionの組。保存は条件付き保存（#2595） | 条件付き | #2743 / #2595 |
| Review | Lifecycle（状態機械）＋Capability（人間のみ設定）＋開示投影 | 表現できる。attributionは未 | #1174 / #2516 |

### 3.2 横断

| SUIの関心 | TEI側の表現候補 | 状態 | 担当 |
| --- | --- | --- | --- |
| tenant / scope | Semantic Extension（宣言）＋Database Provider（RLS等）＋Runtime Capability＋conformance | 設計済み・未採用 | [SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md](SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md) §5 |
| SafeMode（redact/preserve/omit） | Attribute注釈＋Projection。review状態は条件として外から与える | 設計候補 | #2516 |
| Source失効・鮮度 | Source index / provenance、Import原文の失効 | 設計済み | #2743 / #3479 |
| 部分取得・coverage | read footprint / slice（未取得・0・部分・完全） | 設計済み | #2506 / #2611 |
| 版と条件付き保存 | 照合済みbyteだけを渡す、ConditionalWrite | 設計済み | #2595 |
| 外部agent | 許可リスト方式のread-only投影 | SUI側で実装済み | #3368 |
| 共通操作 | 共通Application API（CLI/MCP/HTTP/Web/Desktop） | 方針採用済み | #1158 / #3419 |

## 4. Plugin分解の案（既存`CapabilityKind`のまま）

新しい種別は増やしません。SUIの単位（A〜G）を、既存の capability と投影へ割り当てます。

| SUIの単位 | 移行先の姿 | 既存のcapability / 仕組み |
| --- | --- | --- |
| A 中核ドメイン（DocumentV1、検証、KJ語彙） | `.tei.md`のCanonical定義（`sui.*`）＋Validation。SUIは最後まで正本 | Canonical / Validation。`CapabilityKind`は増やさない |
| B canvas / geometry | native assetのまま。TEIはInteraction境界とHost Grantの検査だけを提供 | #993（Reference）。標準UI外部制御（#2391）は用途を分ける |
| C AI層 | 任意の依存。SEIとの境界を整理してから判断 | editor-intelligenceはSEI側。#2497のProjectionはAI向け説明にも使う |
| D SaaS基盤（tenant・認証） | scope-isolation BundleのL0〜L3。Python realizationとして明示 | Database Provider＋Runtime Capability＋conformance |
| E 交換（import/export/bundle） | Disclosure Projection（項目×開示先）とSafe Artifact Intake | #2516、#943 |
| F MCP | 共通APIのAdapter。Aの契約だけに依存 | #1158 / #3419 |
| G 永続化 | Database Providerの利用者側 | #2622 |

SUI固有の語彙（`Card`、`Island`、`human_reviewed`）が共通部品の契約へ流入しないことは、[SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md](SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md) §3の防ぎたい失敗として既に扱われています。本書の対応表も同じ線引きに従います。

## 5. タスク分解（M0〜M6）と作業票

### 5.0 後続の実装者向けの約束事

**読む順番**

1. 本設計 §2（何が実証されたか）→ §3（何が表現できるか）→ §4（どこに載るか）
2. 担当タスクの作業票（§5.1〜§5.7）
3. 既存設計の該当章: scopeは[SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md](SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md) §5・§8、Stage 0は[SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md](SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md)
4. 委譲先Issueの現在の内容（本書は各Issueの代わりになりません）

**作業票の8項目**。各タスクはIssueへ起こすとき、この8項目をそのまま書きます。項目が埋まらない場合は、未決として明示し、推測で埋めません。

| 項目 | 意味 |
| --- | --- |
| 目的 | 減らす不確実性（RQ・Issue） |
| 変更境界 | 触るファイル・触らないファイル。第二の正本を作らない |
| 依存 | 着手前に成立しているべきタスクと版 |
| 受入 | 機械で確認できる条件。合成FixtureのPASSは実ProviderやOwner意味受理の資格にしない |
| 負例 | 落とすべき入力（未知・欠落・曖昧・失効・非公開） |
| 未実施 | 実行しなかった検証と残る危険 |
| 委譲先 | レビュー・統合の戻り先（ロール） |
| 撤退条件 | 効果が確認できないときに戻る先 |

**環境の注意（2026-11-11時点）**

- `TEI Cache Full vs Incremental Verification` ワークフローの `cache-verification` ジョブは、`actions/upload-artifact` のSHA解決に失敗して全PRで赤くなります（同時刻の無関係ブランチでも同一失敗を確認）。**自分の変更の失敗ではありません**。追跡しないでください。
- `TEI PR・main 健全性確認` の `repository-health` は、GitHub-hostedランナーが割り当てられず保留になることがあります（既知のinfra Issue）。ローカルで `go test ./tools/repository-plan-check/... -count=1` を実行し、その結果をPR本文に書きます。
- このリポジトリでは同時に複数のセッションが動くことがあります。作業は専用のworktreeで分離し、`main`を直接変更しません。

### 5.1 M0 read-only参照の実装（**完了**: SUI PR #3429）

- 目的: SUI #3401 I0。契約とProviderのread-only照合
- 変更範囲: SUIのconsumer（`tei_sui_reference.py`、`routes/tei_epistemic.py`、`api.md` §2.18、ADR-0116）。TEI側の変更なし
- 受入: 参照・変更・再参照・Source失効・開示制限の試験（28件）と関連8ファイル（82件）が通り、CI 7ジョブが成功
- 未実施: 実ブラウザ、実PostgreSQL、TEI側producer
- 委譲先: `reviewer`

### 5.2 M1 producer契約（`source.status`と文書単位の開示宣言）

- 目的: consumerが推定している失効と開示を、producerが宣言する契約にする（F-5・F-6）
- 変更境界: TEI側のartifact生成側のみ。consumer（SUI）は変更しない（後方互換を保つ）
- 依存: M0。#2743（Source鮮度）、#3479（Import原文の失効）
- 契約案（**案。採択前**）。artifactへの追加はすべて省略可能とする:

```json
{
  "source": { "teiCommit": "…", "documentRevision": "rev-1", "status": "active" },
  "disclosure": { "scope": "internal" }
}
```

  - `source.status`: `active` / `revoked` / `expired`。**省略と未知の文字列は`unknown`**。`unknown`を`active`に読み替えない
  - `disclosure.scope`: `internal` / `restricted`。省略は制限なし。`restricted`は文書全体を非返却（種別・ID・件数も出さない）
- 受入:
  1. 省略時にconsumerが`unknown`として損失へ出すこと（SUI PR #3429の`source-status: unresolved`）
  2. `revoked` / `expired`でconsumerが409を返し、IDも件数も出さないこと
  3. `restricted`でconsumerが403を返し、種別・ID・件数を出さないこと
  4. 未知の文字列値（`"Active"`、`"gone"`）が`unknown`に倒れること
  5. producer側で、宣言の有無がconsumerの挙動を変えることを試験で示すこと
- 負例: `status`に数値を入れる、`disclosure`を文字列（`"restricted"`）で渡す（扱いを決めてから実装する。案: 文字列も許す）、`active`を既定値としてproducerが埋める
- 未実施: producer側の実装。契約の採否はOwner判断（§7-1）
- 委譲先: `architect`（契約の形）→ `implementer`
- 撤退条件: 採択されない場合、consumerは`unknown`のまま動かし、失効の判定はconsumer側の限定した読みに留める

### 5.3 M2 SUIドメインのCanonical定義（Reference Workspace）

- 目的: `sui.*`を`.tei.md`で宣言する。instanceは昇格しない（Stage 0の規則を維持）
- 変更範囲: Reference Workspaceの定義ファイル。Product runtime・Canonical source・fixtureのinstanceは変更しない
- 依存: M0の対応表（§3.1）。#2497（構造保存写像）
- 定義案（**案**）:

```text
namespace sui

data sui.document { id DocumentID }
data sui.card     { id CardID text Text? x Number y Number
                    claimType ClaimType? holdState HoldState? }
data sui.island   { id IslandID }

relation sui.containment { one sui.island many sui.card }
relation sui.affiliation { many sui.card many sui.island }
relation sui.islandParent { zero-or-one sui.island zero-or-one sui.island }
```

  - `containment`と`affiliation`は**別のrelation**（F-3）。一つのrelationに畳かない
  - Cardは`text`で合一しない（F-2）。同一名称は別occurrence
  - **宣言しない**: Tree閲覧状態、canvas transform / z-order / zoom / gesture、island geometry、review attribution、Relation rationale。宣言しないことを「未宣言」として観測できるようにする
- 受入:
  1. `go run ./cmd/tei validate -workspace <workspace>` が通ること
  2. `sui.containment`と`sui.affiliation`が別のrelationとして一意に解決できること
  3. fixtureのinstanceがCanonical objectへ昇格していないこと（read-only probeの規則）
  4. Tree階層・canvas geometryが「未宣言」として報告されること
- 負例: containmentとaffiliationを一つのrelationにする案、`text`をcardの自然キーにする案、fixtureのinstanceをCanonicalへ写す案
- 未実施: 定義の採否。instanceのCanonical object化は扱わない
- 委譲先: `architect`（語彙の線引き）→ `implementer`
- 撤退条件: 第二の消費者が現れなければ、`sui.*`はSUIのnative assetのままにし、共通部品へ昇格させない（#2497の基準）

### 5.4 M3 record/string投影（#2488の欠落の充足）

- 目的: 受理した意味からDocumentV1相当のpayloadを生成する。consumerが二度書くコスト（F-10）を避ける
- 変更範囲: generator contractと適合fixture。#2488が観測した二つの拒否（数値契約の前提、textだけの契約の拒否）を解消する
- 依存: M2。#2585（値忠実度・Presence・多値型）
- 契約案（**案**）:

```json
{
  "strategy": "template",
  "inputs": [{ "attribute": "sui.card.text" }, { "attribute": "sui.card.x" }],
  "output": { "kind": "record" },
  "missing": "unknown",
  "digest": { "record": "source-and-binary" }
}
```

  - `missing`: 欠損・未確定を`unknown`として扱うか、項目を落とすか。**空文字や0で埋めない**
  - `digest`: 生成物のsource/binary digestとprovenanceを残す
- 受入:
  1. #2488が観測した2つの拒否が、この契約では起きないこと
  2. 欠損入力で生成が失敗または`unknown`になり、空文字で埋まらないこと
  3. 生成物のdigestとprovenanceが記録され、再生成で一致すること
- 負例: 数値契約（quantity/rounding前提）をそのまま渡す、欠損を空文字で埋める、digestを記録しない
- 未実施: generator familyの設計判断は#2488のOwner。実DB・実Providerでの生成は未実施
- 委譲先: #2488のOwner → `implementer`
- 撤退条件: 成立しなければ、二度書きはconsumer（SUI側）のままに残す。性能の主張はしない

### 5.5 M4 Disclosure Projection（項目×開示先）

- 目的: `redact` / `preserve` / `omit` と、SUI固有のreview軸を条件として分離し、export・共有・AI入力を一つの宣言から導く
- 変更範囲: Attribute注釈とProjection。#2516がOwner
- 依存: M2。#3369（本文公開の判断）
- 宣言案（**案**）:

```text
attribute sui.card.text {
    disclosure { ui = preserve, share = redact, review-pack = preserve,
                 diff = omit, ai-input = derive(reviewState) }
}
```

  - 判定は**項目 × 開示先**の表。文脈は一つではない（付録B.2-3）
  - review状態（`human_reviewed`）は**条件として外から与える**。SUI語彙を契約へ入れない
  - `withheld`（開示しない）と`unknown`（不明・無い）を区別する印を持つ（F-7）
  - `omit`の結果に「伏せられた」ことを示す印を残すかを決める（付録B.2-6）
- 受入:
  1. 網羅性が型または検査で縛られること（項目追加時に判断を忘れない）
  2. 印が`withheld`と`unknown`を区別すること
  3. #3369の判断なしに、本文が`share`や`ai-input`へ出ないこと
  4. 同じ宣言からexport・share・AI入力の三経路が導けること
- 負例: 拒否リスト式の開示（既定で出る）、`omit`を「無い」として扱う、review状態を契約に埋め込む
- 未実施: 印の形の決定、第二のApplicationでの再現
- 委譲先: #2516のOwner → `implementer`
- 撤退条件: 採択されない場合、開示判断は各経路のまま（SUIの第四の形を含む）に留め、一本化は主張しない

### 5.6 M5 Source状態と版の権威をSource index側へ移す

- 目的: `revoked` / `expired` / 鮮度を、consumerの推定ではなくSource indexが権威として扱う（F-4・F-5）
- 変更範囲: Source index / provenance。#2743・#3479がOwner
- 依存: M1。#2595（照合済みbyteだけを渡す）
- 受入:
  1. 失効が次のreadで反映されること（キャッシュで遅らせない）
  2. 古い参照が無効化されたことがconsumerから観測できること
  3. `unknown`（未宣言）が`active`として扱われないこと
  4. 内容変更とsource revision変更が別の状態として観測できること
- 負例: 失効をキャッシュで遅らせる、`unknown`を`active`に読み替える、digestとrevisionを同一視する
- 未実施: 実Providerでの観測。replica構成での「次のread」の意味
- 委譲先: #2743のOwner → `implementer`
- 撤退条件: Source index側に載せられない場合、consumer側の限定した読みに留め、権威の所在をIssueへ記録する

### 5.7 M6 scope isolationのconformanceを言語非依存で書き出す

- 目的: SUIの越境試験を、Python実装から切り離した契約としてTEI側へ置く。実装の書き換えを急がない
- 変更範囲: 試験と言語非依存の記述。SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md §8.4 Stage 1
- 依存: 既存設計。ISSUE-000080（Capability Contract）の結果
- 受入:
  1. 14経路の行列が「該当機能があれば必須」の形で機械可読に書けること
  2. 欠落4経路（list / export-audit / 監査読み取り / import権限非継承。同書 付録A）を埋める試験がSUI側で通ること
  3. Python実装がその試験を通ること
- 負例: 試験を通すために契約を緩める、実装を通すだけで契約を変える
- 未実施: ISSUE-000080の解消。第二のApplicationでの再現
- 委譲先: `planner`（行列の言葉）→ `implementer`
- 撤退条件: ISSUE-000080が解けない間は、L0（宣言）とL3（試験）だけを進め、L1・L2のPlugin化は後に回す

### 5.8 順序とブロッカー

```text
M0（参照consumer、完了）
  ├── M1（producer契約） ── M5（Source権威）
  └── M2（Canonical定義） ── M3（record投影） ── M4（開示投影）
M6（scope conformance）は独立
```

| タスク | ブロッカー | 解除の条件 |
| --- | --- | --- |
| M1・M4 | SUI #3369（本文公開の判断） | Owner判断。判断が無い間は本文を出さない経路のまま |
| M3 | TEI #2488（generator familyの設計判断） | #2488のOwner判断 |
| M5 | #2743 / #3479 の現在の設計 | Source index側の責務の明確化 |
| M6 | ISSUE-000080（Capability Contract） | 解消後。それまでL0とL3のみ |
| M2 | #2497（語彙の線引き） | 第二の消費者が出るまで標準化しない |

## 6. リスクと、成立しない場合の撤退

- **payloadと遅延が解消しない場合**: 参照経路は「識別・関係・状態」の用途に限定し、本文を要する用途はDocumentV1のままにします（ADR-0116と同じ立場）。M3が成立しなければ、二度書きはconsumerのままに残します。
- **producer契約が採択されない場合**: `unknown`のまま動かし、失効の判定はconsumer側の限定した読みに留めます。契約の強制はしません。
- **scope isolationが標準化されない場合**: L0（宣言）とL3（試験）だけを進め、L1・L2のPlugin化はISSUE-000080の後に回します（既存設計と同じ）。
- **第二の消費者が現れない場合**: `sui.*`語彙はSUIのnative assetのままにし、共通部品へ昇格させません（#2497の基準）。
- **撤回条件**: 参照経路が「第二の正本」として使われ始めた場合（artifactを保存・編集する経路が生まれた場合）、あるいは本文が不用意に載り始めた場合、経路をread-onlyのまま閉じ、判断をIssueへ戻します。

## 7. Owner判断が必要な論点

1. `source.status`と文書単位の開示宣言をproducer契約にするか（M1）
2. カード本文・題名を業務面へ出してよいか（#3369）。参照経路は出していません
3. `revoked`を「参照の無効」と「Sourceの削除」のどちらとして扱うか
4. `documentDigest`（内容指紋）とSUIの保存revisionを同一視しないことを、どう契約に載せるか
5. Treeの閲覧状態（折り畳み・focus）をCanonicalへ入れるか、対象外とするか
6. Canvasのpolygon・placardをCanonicalへ入れるか、Interaction境界だけにするか（#993）
7. review attributionをDataとして持つか、監査・Source側に委ねるか
8. SUI固有の語彙（`Card`・`Island`・`human_reviewed`）を、どこまで一般化の候補として外へ出すか

## 8. 既存Issueへの委譲

| 論点 | 委譲先 | 本書の扱い |
| --- | --- | --- |
| 構造保存写像・損失の語彙 | [#2497](https://github.com/hat47x/tei-platform/issues/2497) | §2.2のinventoryを論点として渡す |
| record/string投影 | [#2488](https://github.com/hat47x/tei-platform/issues/2488) | §5.4 |
| 開示投影 | [#2516](https://github.com/hat47x/tei-platform/issues/2516) | §5.5 |
| Source鮮度・失効 | [#2743](https://github.com/hat47x/tei-platform/issues/2743) / [#3479](https://github.com/hat47x/tei-platform/pull/3479) | §5.2・§5.6 |
| 共通Application API | [#1158](https://github.com/hat47x/tei-platform/issues/1158) / [#3419](https://github.com/hat47x/tei-platform/issues/3419) | §5.4・§5.5の出口 |
| 実Read・coverage | [#2586](https://github.com/hat47x/tei-platform/issues/2586) / [#2506](https://github.com/hat47x/tei-platform/issues/2506) | §5.6 |
| canvas / Interaction | [#993](https://github.com/hat47x/tei-platform/issues/993) | §3.1 |
| 外部agent投影 | SUI [#3368](https://github.com/hat47x/sui-sensemaking/issues/3368) | §3.2 |
| 本文公開の判断 | SUI [#3369](https://github.com/hat47x/sui-sensemaking/issues/3369) | §5.5 |
| データ統合の段階 | SUI [#3401](https://github.com/hat47x/sui-sensemaking/issues/3401) | 全体 |
| SUI側の分割・scope | [SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md](SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md) | §5.7 |

## 9. 本文書が主張しないこと

- SUIがTEIへ移管されること、どのPlugin化も実施されることを保証しません
- 実測値は合成Documentのプロセス内計測で、配備後のlatencyでも実Providerのreadでもありません
- M1〜M6は未実施です。合成FixtureのPASSやconsumer側の試験を、Ownerの意味受理や実Providerの資格として扱いません
- 契約案・定義案・宣言案は**案**であり、採択された契約ではありません
- TEIのCore、Plugin契約、`CapabilityKind`の変更を要求しません
- SUI一件の実装から、共通部品としての標準化が十分な証拠を持つことは主張しません（F-12）
