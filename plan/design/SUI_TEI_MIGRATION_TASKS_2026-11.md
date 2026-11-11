# SUI→TEI移行：read-only参照の実装知見と、*.tei.md＋Pluginへの移行タスク分解

- 状態: 評価待ち（未採用）。実装・Reference変更・受入の根拠にしない
- 日付: 2026-11-11
- 対象: SUI SensemakingのDocument/Card/Island/Tree/Canvas/Relation/Revision/Reviewと、TEIの*.tei.md（Canonical定義）＋Pluginによる設計〜実装への移行
- 正本との関係: [SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md](SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md)（SUI側の分割とscope isolation）と[SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md](SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md)（Stage 0のshadow conformance）を置き換えない。本書は「2026-11-11にSUI側でread-only参照が実装された知見」と、そこから展開した移行タスクの分解だけを扱う
- 根拠: SUI PR #3429（`GET /docs/{doc_id}/tei/sui-reference`）、SUI ADR-0116、SUI `02_Architecture/api.md` §2.18、SUI #3401
- 主張しないこと: TEIのCore・Plugin契約・`CapabilityKind`の変更を求めない。SUIがTEIへ移管されることを保証しない。第二の消費者が出るまでの標準化を主張しない

## 1. 本文書の役割

SUI側で「TEI Referenceのartifactを、型付きのread-only参照として読む」経路が実装されました（2026-11-11）。これは移行のStage 0に当たる実装であり、次の三つを初めて実コードで确定させます。

1. SUIの主要な構造が、TEI側の型へ落としたとき、どこまでが失われずに運べるか
2. 「運べない」が、部分取得・未レビュー・非公開・失効・版とどう区別して現れるか
3. 既存のSUI経路（DocumentV1）と比べて、参照経路にどの実利と代償があるか

本書はこの知見を、*.tei.md＋Pluginという移行先の構成へ翻訳し、今後必要なタスクを依存関係と受入条件付きで分解します。判断の採用は各IssueとADRの仕事で、本書は分解と論点の整理までを行います。

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

## 5. タスク分解（M0〜M6）

M0は完了です。M1以降は未実施で、依存と受入条件を付けます。各タスクは「変更範囲」「依存」「受入」「未実施・不明」を明示し、合成FixtureのPASSを実ProviderやOwner意味受理の証拠として扱いません。

### M0 read-only参照の実装（完了: SUI PR #3429）

- 変更範囲: SUIのconsumer（`tei_sui_reference.py`、route、api.md、ADR-0116）。TEI側の変更なし
- 受入: 参照・変更・再参照・Source失効・開示制限の試験（28件）と関連8ファイル（82件）が通り、CI 7ジョブが成功
- 未実施: 実ブラウザ、実PostgreSQL、TEI側producer

### M1 producer契約の判断（`source.status`と文書単位の開示宣言）

- 目的: consumerが推定している失効と開示を、producerが宣言する契約にする
- 変更範囲: TEI側のartifact生成側。未決の間はSUIは`unknown`として動く（後方互換）
- 依存: M0。#2743（Source鮮度）、#3479（Import原文の失効）
- 受入: 「宣言がない場合の既定」と「宣言がある場合の検証」が別に試験される。`unknown`が`active`に読み替わらない負例が在る
- 未実施: producer側の実装。契約の採否はOwner判断

### M2 SUIドメインのCanonical定義（Reference Workspace）

- 目的: `sui.document` / `sui.card` / `sui.island` とrelationを、`.tei.md`で宣言する。instanceは昇格しない
- 変更範囲: Reference Workspace。Product runtime・Canonical sourceの変更なし
- 依存: M0の対応表。#2497（構造保存写像）
- 受入: 定義が一意に解決すること。containmentとaffiliationが別のrelationとして解決すること。Tree階層・canvas geometryが「未宣言」として観測できること
- 未実施: 定義の採否。instanceのCanonical object化は扱わない

### M3 record/string投影（#2488の欠落の充足）

- 目的: 受理した意味から、DocumentV1相当のpayloadを生成できるようにする。consumerが二度書くコスト（F-10）を避ける
- 変更範囲: generator contractとfixture。#2488が観測した二つの拒否（数値契約の前提、textだけの契約の拒否）を解消する
- 依存: M2。#2585（値忠実度・Presence・多値型）
- 受入: 生成物のsource/binary digestとprovenanceが記録される。欠損・未確定の扱いが契約に載る
- 未実施: generator familyの設計判断は#2488のOwner

### M4 Disclosure Projection（項目×開示先）

- 目的: `redact` / `preserve` / `omit` と、SUI固有のreview軸を条件として分離し、export・共有・AI入力を一つの宣言から導く
- 変更範囲: Attribute注釈とProjection。#2516がOwner
- 依存: M2。#3369（本文公開の判断）
- 受入: 「項目が伏せられた」ことを示す印が、`withheld`（開示しない）と`unknown`（不明・無い）を区別する。網羅性が型または検査で縛られる
- 未実施: 印の形の決定。#3369はSUI側の判断

### M5 Source状態と版の権威をSource index側へ移す

- 目的: `revoked` / `expired` / 鮮度を、consumerの推定ではなくSource indexが権威として扱う
- 変更範囲: Source index / provenance。#2743・#3479がOwner
- 依存: M1。#2595（照合済みbyteだけを渡す）
- 受入: 失効が次のreadで反映されること。古い参照が無効化されたことがconsumerから観測できること
- 未実施: 実Providerでの観測

### M6 scope isolationのconformanceを言語非依存で書き出す

- 目的: SUIの越境試験を、Python実装から切り離した契約としてTEI側へ置く。実装の書き換えを急がない
- 変更範囲: 試験と言語非依存の記述。SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md §8.4 Stage 1
- 依存: 既存設計。ISSUE-000080（Capability Contract）の結果
- 受入: 14経路の行列が「該当機能があれば必須」の形で書ける。list・export-audit・監査読み取り・import権限非継承の欠落（同書 付録A）が埋まる
- 未実施: ISSUE-000080の解消。第二のApplicationでの再現

### 横断で見るべき依存

```text
M0（参照consumer） ── M1（producer契約） ── M5（Source権威）
        └── M2（Canonical定義） ── M3（record投影） ── M4（開示投影）
                                     └── M6（scope conformance、独立）
```

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
| record/string投影 | [#2488](https://github.com/hat47x/tei-platform/issues/2488) | M3 |
| 開示投影 | [#2516](https://github.com/hat47x/tei-platform/issues/2516) | M4 |
| Source鮮度・失効 | [#2743](https://github.com/hat47x/tei-platform/issues/2743) / [#3479](https://github.com/hat47x/tei-platform/pull/3479) | M1・M5 |
| 共通Application API | [#1158](https://github.com/hat47x/tei-platform/issues/1158) / [#3419](https://github.com/hat47x/tei-platform/issues/3419) | M3・M4の出口 |
| 実Read・coverage | [#2586](https://github.com/hat47x/tei-platform/issues/2586) / [#2506](https://github.com/hat47x/tei-platform/issues/2506) | M5 |
| canvas / Interaction | [#993](https://github.com/hat47x/tei-platform/issues/993) | §3.1 |
| 外部agent投影 | SUI [#3368](https://github.com/hat47x/sui-sensemaking/issues/3368) | §3.2 |
| 本文公開の判断 | SUI [#3369](https://github.com/hat47x/sui-sensemaking/issues/3369) | M4 |
| データ統合の段階 | SUI [#3401](https://github.com/hat47x/sui-sensemaking/issues/3401) | 全体 |
| SUI側の分割・scope | [SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md](SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md) | M6 |

## 9. 本文書が主張しないこと

- SUIがTEIへ移管されること、どのPlugin化も実施されることを保証しません
- 実測値は合成Documentのプロセス内計測で、配備後のlatencyでも実Providerのreadでもありません
- M1〜M6は未実施です。合成FixtureのPASSやconsumer側の試験を、Ownerの意味受理や実Providerの資格として扱いません
- TEIのCore、Plugin契約、`CapabilityKind`の変更を要求しません
- SUI一件の実装から、共通部品としての標準化が十分な証拠を持つことは主張しません（F-12）
