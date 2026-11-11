# TEI受理済みモデル → SHO文書Draft：最小統合機構の設計

## この文書の役割と状態

- **状態**：`実装済みの最小統合の基線`（2026-11-11）。`internal/documentdraft`・`tei model document-draft`・`tei model document-reconcile` と、SHO側の `src/document-draft.cjs`・`bin/sho-doc.cjs` を対象とする。実装済みの `documentKind` は `element-explanation`・`crud-table`・`specification`・`three-element-diagram` の4つ。`-frozen` による固定納品版のpinも実装済み。後続段階は[TEI_SHO_DOCUMENT_DRAFT_NEXT_STAGES_2026-11.md](TEI_SHO_DOCUMENT_DRAFT_NEXT_STAGES_2026-11.md)が正本。
- **正本との関係**：意味の正本はTEIのWorkspace SourceとOwner受理（[#267](https://github.com/hat47x/tei-platform/issues/267)）にあり、この文書は**派生Projectionの契約**を置くだけである。採用した設計判断は[plan/adr/](../adr/README.md)、作業の進行状況はIssueが正本。SHO側の対応は[SHO #19](https://github.com/hat47x/sho-composer/issues/19)・[SHO #4](https://github.com/hat47x/sho-composer/issues/4)・[SHO #6](https://github.com/hat47x/sho-composer/issues/6)で管理し、契約本文をSHO repositoryへ複製しない（URL参照）。
- **対象範囲**：受理済みモデルから**一つの要素と一つの関係**を読み取り、SourceとRevisionを保持した説明文書のDraftをSHOへ渡し、SHOで編集・保存・再開し、TEI側でモデル改版との差分と再確認が必要な部分を提示するまでの一経路。
- **この文書が主張しないこと**：SHO保存成功がTEIモデルへの取込み成功やAccepted Meaningの更新成功であることは示さない。図面の製品編集、Studioへの組込み実証、`*.tei.md`の正式Parser、Owner意味受理、旧Runtimeとの同値はすべて別資格である。

## 1. 責務の分離

| 責務 | 所有者 | すること | しないこと |
| --- | --- | --- | --- |
| 文書編集の共通コア | SHO（`src/g2-document.cjs`） | Markdown raw byteのopen・編集・保存・再開。raw byte忠実性、BOM/EOL/未知構文の保存、競合検出時の上書き拒否 | 意味解釈、モデル参照の解決、再生成、整形 |
| TEIモデル参照 | TEI（`internal/documentdraft`） | 受理済みWorldとSource Indexからの読取り。Workspace identity・`worldFingerprint`・`sourceIndexFingerprint`・`modelRevision`・Source（path/fingerprint/role/range）の保持 | Source書換え、意味受理、Host権限 |
| 文書Projection | TEI（`internal/documentdraft`） | 一要素＋一関係から説明・関係・表・引用の各Occurrenceを生成し、生成条件を付与 | 著者文の生成・上書き、Occurrence IDの永続保証 |
| 保存 | SHO（`src/document-draft.cjs`＋共通コア） | 著者原文の保存、base digestによる競合検出、readback検証、生成領域と著者領域の分離保持 | モデル側への書戻し、引用の自動再解決 |
| 引用 | SHO（記録）／TEI（源泉） | 引用OccurrenceはSource identity・Revision・rangeを**記録として保持**するだけ | 取得・検証・relink。未確認は未確認のまま保存する |
| 引用Register（SHO既存） | SHO（`src/g2-references.cjs`） | 意図したSource・名目解決先・取得byte・宣言/観測fingerprint・Review現在性を別欄で持つ。取得は明示かつbounded | このDraft契約の代替にはならない。Draft引用は `recorded-not-resolved` の記録モード |
| 意味受理・モデル変更 | TEI Owner（既存経路） | 変更計画・検証・受理は[#267](https://github.com/hat47x/tei-platform/issues/267)と既存Shared Acceptance経路 | SHO文書からの逆書込み、REA/再解析からの暗黙昇格 |

SHOはTEIの意味正本も専用Hostも持たない。TEI StudioはSHOの文書編集コンポーネントを**同じ契約の部品**として利用し、製品別の二重実装を作らない。組込み形態（子プロセス／静的組込み／Wails資産）は未測定で、この設計は**JSON契約とCLI境界**を最小統合機構として固定する。

## 2. Draft契約（`tei.document-draft/v0`）

TEIが生成し、SHOが消費する。SHOは语义を解釈せず、identity・revision・生成条件・anchorを**そのまま保持**する。CLIはこのDraftを `tei.cli.model-document-draft/v0` の封筒に入れて返すので、Consumerは封筒を開いて `result` を採る（SHO側は両方の形を受付ける）。

`documentKind` は文書の種類を選ぶ。実装済みは `element-explanation`（一要素＋一関係）・`crud-table`（一ApplicationのCRUD行列）・`specification`（複数要素の章立て）・`three-element-diagram`（三要素分析図）。**未実装の種類は `UNSUPPORTED_DOCUMENT_KIND` で拒否する**——後続段階を途中まで実装して完成文書のように見せないため。CLIの入力は種類ごとに排他で、`crud-table` は `-application` を、`specification` は `-section`（repeatable・章順）を、`three-element-diagram` は `-model`（三要素分析モデルJSON・**Workspaceは読まない**）を取り、`element-explanation` はそのいずれも拒否する。

`three-element-diagram` の入力は**TEI Workspaceとは別の分析モデル**である。2つを混ぜない。要素対応（何がData/Function/Businessか）はモデルファイルの宣言でありTEIの判断ではない——Owner判断事項で、この投影は対応を決けたように見せない。宣言されていないIDを端点にする辺は落とさず `unresolved` として名前を残す。

`-frozen` は**種類ではなくpin**。Draftを「読出した時点のmodelRevision」に固定し、固定納品版として使う。pin後は `frozen: true` と `frozenRevision` をDraftが持ち、reconcileはpinが現在のrevisionと違えば `stale`（`frozenRevision` diff）として**失効**を報告する。**Draftも文書も書換わない**。`frozen-delivery` を種類として指定した場合は拒否する（pinの付け忘れを種類の誤指定で誤魔化さない）。`frozen: true` で `frozenRevision` が無いDraftは `INVALID_DRAFT`（手編集で期限を不能にできない）。

```jsonc
{
  "schemaVersion": "tei.document-draft/v0",
  "effect": "read-only-projection",
  "draftId": "sha256:...",                 // schema+subject+relation+modelRevisionから決まる
  "documentKind": "element-explanation",   // 後述の段階的拡張の種別
  "title": "sales.order — 要素説明",
  "model": {
    "workspace": { "name": "...", "formatVersion": 1, "manifest": { "path": "...", "sha256": "..." } },
    "worldFingerprint": "sha256:...",
    "sourceIndexFingerprint": "sha256:...",
    "modelRevision": "sha256:..."
  },
  "subjects": [                            // 参照した要素と関係の対象
    { "subject": { "kind": "semantic-data", "id": "sales.order" },
      "symbol": "order",
      "sources": [ { "artifactPath": "semantic/sales/order.tei", "format": "tei",
                      "fingerprint": "sha256:...", "role": "definition",
                      "ranges": [ { "start": {...}, "end": {...} } ] } ] }
  ],
  "relations": [                           // 読取った一関係。未解決ならunknownを保持
    { "relation": { "kind": "semantic-relation", "id": "sales.order.customer" },
      "from": { "kind": "semantic-data", "id": "sales.order" },
      "to":   { "kind": "semantic-data", "id": "customer.customer" },
      "resolution": "resolved",             // resolved | unresolved | ambiguous
      "cardinality": "one", "keys": [ ... ] }
  ],
  "occurrences": [                         // 追跡単位
    { "occurrenceId": "occ-explanation-1", "kind": "explanation",
      "subjectRefs": [ { "kind": "semantic-data", "id": "sales.order" } ],
      "generationConditions": {
        "generator": "tei.internal.documentdraft", "generatorVersion": "v0",
        "inputDigests": { "worldFingerprint": "...", "sourceIndexFingerprint": "...", "modelRevision": "..." },
        "rule": "element-summary-v0"
      },
      "content": { "text": "..." },
      "contentDigest": "sha256:..." }
  ],
  "authorityLimits": [ ... ]
}
```

`occurrences[].kind` は `explanation`・`relation`・`table`・`citation`・`diagram`。生成物（説明・図面・表・引用）はすべてOccurrenceであり、**元モデルのIdentity・Revision・生成条件・文書内Occurrence**を失わずに保持する。図面は`diagram`種別で**構造記述**のみを持ち、レイアウトは持たない（[THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)と同じ扱い）。

`crud-table` では表の各行が `rowSubjects`（対象データ＋Application）を持ち、行単位で元Identityへ戻れる。表の根拠は `scope.crudRule` が命名する（`internal/operations` の operation surface）。未導出・矛盾は行列に混ぜず別Occurrenceにする。

## 3. 経路（初回実証）

1. **Draft生成（TEI, read-only）**
   `tei model document-draft -workspace <root> -subject-kind semantic-data -subject-id sales.order -relation-id sales.order.customer`
   受理済みWorldとSource Indexを一つのloadで読み、Draft JSONを標準出力に書く。Source書換え・意味受理は行わない。
2. **Draft受領・編集（SHO）**
   `node bin/sho-doc.cjs open <draft.json> <document.md>` で作業文書を新規作成（既存は上書きしない）。生成Occurrenceは`<!-- sho:occurrence ... -->`マーカーで囲み、著者領域と区別する。文書冒頭のマーカーはDraft ID・`modelRevision`・Occurrence一覧を持ち、Occurrence全体の削除を検出できる。
3. **保存（SHO）**
   `node bin/sho-doc.cjs save <document.md> [--note "著者のメモ"]`。保存はG2文書セッションを通り、公開前にtargetを読み直して競合を検出する。競合時は**書かない**。no-op保存は物理書込み0。
4. **再開（SHO）**
   `node bin/sho-doc.cjs resume <document.md>`。保存済みraw byteのdigest・Occurrenceの状態（`intact`・`authorEditedGeneratedBlock`・`unterminated-occurrence`・欠落）・著者領域のbyte数を返す。モデル参照は**記録として報告**し、検証したとは言わない。
5. **モデル改版時の差分と再確認（TEI, read-only）**
   `tei model document-reconcile -workspace <root> -draft <draft.json>`。model参照の失効（どのdigestが動いたか）・Occurrence単位の差分・**再確認が必要なOccurrence一覧**を返す。**著者文は決して上書きしない**。更新は新しい候補Draftとして提示し、著者が採用する。
6. **逆方向（TEIモデル変更）**
   SHO文書の文章修正はTEIモデル変更の根拠にならない。モデル変更は既存の変更計画・検証・Owner受理の経路（[#267](https://github.com/hat47x/tei-platform/issues/267)）のみを通る。

### 相互実証（2026-11-11、合成Workspace・CI同一Attempt）

- **CI**: SHO repository の [sho-tei-draft-roundtrip.yml](https://github.com/hat47x/sho-composer/blob/feat/tei-document-draft-core-20261111/.github/workflows/sho-tei-draft-roundtrip.yml) が **success**（run 42995269509）。pin した TEI SHA を public clone し、実CLIの draft 生成 → 実CLIの open/save/resume → 実CLIの reconcile を一回のAttemptで実行。receipt は artifact に保存。
- **ローカル**: 同一runnerを手元で実行し、全check true・全claim false を確認。誤pin・誤headでは assertion が exit 1。
- いずれも**合成データによる局部経路の実証**である。TEI Studioからの実組込み、Owner意味受理、旧Runtimeとの同値は含まない。

## 4. `*.tei.md` の扱う場合の分離

- SHOは`*.tei.md`を**通常のMarkdown原文**として編集・保存できる。構文解析・意味検証・モデル反映はTEI側の責務で、現mainのLoaderは`TEI_MD_PARSER_UNBOUND`として未束縛のMarkdown Sourceを**拒否**する（`internal/workspace/tei_md_source_guard.go`）。
- したがって**SHOで`*.tei.md`を保存できたことは、TEIモデルへの取込み成功でもAccepted Meaningの更新成功でもない**。取込みは正式Parser/Role/Source Indexが成立した時だけ別途判定する。
- Draft契約の`authorityLimits`は`consumer-save-is-not-tei-model-ingestion`と`tei-md-authoring-parser-is-unbound-on-current-main`を常設で持ち、SHOのReceiptも「原文保存」のみを主張する。

## 5. 独立Expected（負例を含む）と実テストの対応

| ID | 圧力条件 | 期待される区別 | 実テスト |
| --- | --- | --- | --- |
| RD-01 | Draft生成後にWorkspaceのSourceを改版し、旧Draftでreconcile | `stale`（どのdigestが動いたか）・Occurrence差分・再確認リスト。自動更新しない | `TestReconcileNamesTheOccurrencesThatNeedReconfirmationAfterAModelChange`（TEI） |
| RD-02 | 同名・別内容のSource（同じID・別byte／同じbyte・別所在） | path/occurrenceと内容identityを分離。同名一致で同一意味と推定しない | `TestBuildReportsAnAmbiguousTargetSeparatelyFromAnUnresolvedOne`・`TestReconcileSeparatesASourceMoveFromAMeaningChange`（TEI） |
| RD-03 | SHO保存前に他Editorが文書を改版 | `conflict`で書込み0。base digestとfile identityの両方を観測する | `a concurrent external change is a conflict and nothing is written`（SHO） |
| RD-04 | 生成条件（generator/rule/input digest）を変えて再生成 | 同一Occurrence IDでも生成結果差異を報告。無変化に潰さない | `TestReconcileReportsAnUnknownRuleInsteadOfGuessing`・`TestReconcileDetectsAGenerationDifferenceWithTheSameRuleName`（TEI） |
| RD-05 | 著者が生成Occurrence内の文を編集し、reconcile後も保存 | 著者文を自動上書きしない。`authorEditedGeneratedBlock`を保持し再確認を要求 | `an edit inside a generated block is reported, not repaired`（SHO）＋`reprojection-never-overwrites-author-text`（TEI） |
| RD-06 | Draft JSONを手編集・改変 | schemaVersion/effect/必須identityの欠落を拒否。部分的なDraftを推測で補完しない | `TestReconcileRefusesADraftThatIsNotThisContract`（TEI）・`a draft that is not this contract is refused`（SHO） |
| RD-07 | CRUD表でApplicationを削除・書き込み権限を撤回・update intentを追加 | Occurrence単位で `subject-missing`・`content-changed`・`occurrence-removed`・`occurrence-added` を区別。空表を「このApplicationは何もしない」と読ませない | `TestCRUDTableReconcileReportsARemovedApplication`・`TestCRUDTableReconcileNamesChangedRowsAfterAModelChange`・`TestCRUDTableReconcileReportsARemovedGapTable`・`TestCRUDTableReconcileReportsAnOccurrenceTheModelStartedDeriving`（TEI） |
| RD-08 | 固定納品版（`-frozen`）のpin後・pinなし・種類としての指定・pinの手編集 | pinはmodelRevision。改版で`stale`＋`frozenRevision` diff。`frozen`を種類として指定は拒否。`frozen`でpin無しは`INVALID_DRAFT`。pinは書換わない | `TestFrozenDeliveryPinsTheModelRevision`・`TestFrozenDeliveryReportsExpiryAfterAModelChange`・`TestFrozenDeliveryIsCurrentWhenTheModelDidNotMove`・`TestFrozenDeliveryIsNotADocumentKind`・`TestFrozenDeliveryAppliesToACRUDTable`・`TestFrozenDraftWithoutAPinIsRefused`（TEI） |
| RD-09 | 仕様書（`specification`）で未宣言Subject・未宣言関係・混合入力を指定 | 未宣言Subjectは章を作らず`skippedSubjects`に理由。未宣言関係は章残り関係節のみスキップ。入力形状の混在は拒否 | `TestSpecificationSkipsAnUndeclaredChapterInsteadOfWritingOne`・`TestSpecificationSkipsARelationTheElementDoesNotDeclare`・`TestSpecificationRejectsMixedInputs`・`TestElementAndCRUDKindsRejectSections`（TEI） |
| RD-10 | 仕様書で一章の要素を改版・要素を削除 | 改版はその章のみ`content-changed`、他章は`generation-conditions-changed`（内容は同一）。削除は`occurrence-removed` | `TestSpecificationReconcileReportsTheChaptersThatMoved`・`TestSpecificationReconcileReportsARemovedChapter`（TEI） |
| RD-11 | 三要素分析図で未宣言端点・混在入力・Workspace指定 | 未宣言端点の辺は落とさず`unresolved`。入力形状の混在は拒否。**Workspaceを読まない**（`model.workspace` は記録されない） | `TestThreeElementDiagramKeepsAnUnresolvedEdge`・`TestThreeElementDiagramRefusesMixedInputs`・`TestOtherKindsRefuseAModelPath`・`TestThreeElementDiagramNeverReadsAWorkspace`（TEI） |
| RD-12 | 三要素分析図でモデル改変・不改変 | 改変は`stale`＋`modelFingerprint` diff＋`content-changed`。不改変は`current`・再確認0件 | `TestThreeElementDiagramReconcileReportsAMovedModel`（TEI） |

## 6. 段階的拡張（単一モデル→単一文書の実証後）

| 段階 | documentKind | 追加で必要になるもの | 状態 |
| --- | --- | --- | --- |
| 1 | `element-explanation` | 一要素＋一関係の説明・属性表・引用 | 実装済み（本PR） |
| 2 | `crud-table` | `internal/threeelement`のモデルJSONを第2入力としたCRUD表Projection（[#3263](https://github.com/hat47x/tei-platform/issues/3263)の受入条件を再利用） | 未実装 |
| 3 | `three-element-diagram` | Data/Function/Businessの構造記述（node/edge identity・出典）。レイアウトは持たない | 未実装 |
| 4 | `specification` | 複数Occurrenceの束ねと章構成。意味入力の版範囲の明示 | 未実装 |
| 5 | `frozen-delivery` | 固定Revisionのpin。以後のモデル改版は失効として報告し、自動更新しない | 未実装 |

段階2以降は、それぞれ独立した検証とIssue上の受入条件を要する。1段階のPASSを後段のPASSに流用しない。未実装の`documentKind`は`UNSUPPORTED_DOCUMENT_KIND`で拒否し、実装済みとして扱わない。

## 7. 未解決事項

- SHO文書の引用Occurrenceに対する**解決・検証**（取得・raw byte照合）は未実装。未確認は未確認として保存するのみ。SHO mainには引用Register（`src/g2-references.cjs`）があるが、これは**別のモード**（意図したSource・名目解決先・取得byte・宣言/観測fingerprint・Review現在性を別欄で持つ）であり、Draft引用の `recorded-not-resolved` を置き換えない。二本の引用経路の関係はOwnerの判断を要する。
- TEI Studioからの**実組込み**（子プロセス起動・Wails資産・同一Attempt）は未実証。
- Draft IDの永続性（A→B→A、Workspace差し替え）は`modelRevision`と`worldFingerprint`の範囲でしか保証しない。
- `document-reconcile`は**読取り専用**であり、変更計画・検証・Owner受理の代替ではない。
- 保存済み文書に対する**再投影のマージ方針**（著者編集と新Draftの突合せ）は未実装。現状は新Draftを別文書として提示する。
- SHO側の保存の中核（`src/g2-document.cjs`）は[SHO PR #30](https://github.com/hat47x/sho-composer/pull/30)でmainに統合されたものを再利用する。本契約はそれを前提としない（ Consumerは自分の編集核を持てる）が、現状の参照実装は它に依存する。
