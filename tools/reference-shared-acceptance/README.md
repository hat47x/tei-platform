# reference-shared-acceptance

`ISSUE-000027`（AI Proposal / Acceptance）と`ISSUE-000046`（Studio Shared Acceptance）が共有する、**Shared Acceptance Boundaryの決定論的なReference実装**です。

file edit・CLI・Studio・AIのどの入口から来た変更candidateも、同じ検査を同じ順序で通り、**acceptだけがauthoritative sourceを書き換えます**。AI専用・Studio専用の受入経路はありません。

## 位置づけ

- Reference専用のtoolです。Core primitive、platform共通のProposal schema、Patch DSLではありません。candidateの形式（`tei.reference.change-candidate/v0`）はこのReferenceの実測から決めた運搬形式で、固定を主張しません。
- 差分・依存影響の解析はread-onlyな[`tools/internal/changeanalysis`](../internal/changeanalysis)（`ISSUE-000039`の最初の解析境界）が担当し、このtoolはそれを呼びます。受理・source反映（`ISSUE-000027`）と解析を混ぜず、解析側はsourceを読み書きしません。
- AIによる候補生成、Studio UI、read-only探索（`ISSUE-000047`）は含みません。これらはcandidateの**生成側**であり、このtoolは**受け取る側の境界**だけを実証します。
- Gate B / Gate Cが固定した`reference/workspace`は変更しません。`accept`は、baselineの名前（`.../reference/workspace`）を持つ場所へは`-allow-reference-baseline`が無い限り拒否します。scratch copyへ実行してください。

## 受入境界

```text
decode → normalize → base-revision → resolve → apply(overlay)
       → reload → validate → architecture-policy → semantic-diff
       → impact → acceptance-policy
```

| stage | 確認すること |
| --- | --- |
| decode | 未知fieldを許さず厳密に読む |
| normalize | 正規形へ揃え、意味を持つ部分（base revisionとdelta）だけのfingerprintを作る。Origin / Intent / Evidenceは含めない |
| base-revision | candidateの`baseModelRevision`が現在のsourceと一致する。staleなら黙って適用せず、reject / 再読込して再生成 / 明示rebase / 手動conflict解消へ分岐する |
| resolve | 対象namespace・Attribute・Relationが存在し、追加するidが未使用で、Evidenceがqualified logical keyまたは管理対象sourceへ解決でき、placementが意味を所有するsemantic層のfileである |
| apply | 変更を**overlay（複製）**へ適用する。元のsourceはまだ変えない |
| reload | overlayを共有loaderで読み直す |
| validate | 共有Validationにerrorが一件も無い |
| architecture-policy | 変更されたfileがdeltaに対応するfileだけで、semantic層の中に閉じている |
| semantic-diff | Canonical Worldの実際の差分が、candidateの宣言したdeltaと一致し、semantic層以外のWorld collectionが変わっていない |
| impact | 追加されたsubjectから、exact identityに基づく依存（Attribute → Data → Relation → Application Pattern、namespace利用Application）と、判断できなかった事項を示す |
| acceptance-policy | 変更subject数の上限（`maxSubjectChanges`）を超えない。whole-model再生成を標準経路にしない |

後続stageは、前のstageが失敗すると`skipped`になります。

## 使い方

```bash
go run ./tools/reference-shared-acceptance revision -workspace DIR
go run ./tools/reference-shared-acceptance preview  -workspace DIR -candidate FILE [-out FILE]
go run ./tools/reference-shared-acceptance accept   -workspace DIR -candidate FILE [-derived-out FILE] [-out FILE]
go run ./tools/reference-shared-acceptance reject   -workspace DIR -candidate FILE [-reason TEXT] [-out FILE]
```

- `revision`: workspaceの現在のmodel revision（管理対象source全fileの相対pathと内容fingerprintから導出）を出力します。candidateの`baseModelRevision`にはこれを使います。
- `preview`: sourceを一切書き換えずに、各stageの結果・Semantic Diff・impact・unresolvedを出力します。受理可能でなければ終了コード1です。
- `accept`: 同じ検査を最初から実行し直し、通った場合だけsourceを更新します。更新後はtoolを介さない共有loaderで再読込し、Previewが示した差分どおりか・Validationが通るか・派生物を決定論的に再生成できるかを確認します。一つでも満たさなければ元のsourceへ戻します。`-derived-out`にaccepted sourceから再生成したCanonical World JSONを書き出せます。
- `reject`: sourceを変更しないことを記録します。staleなcandidateも退けられます。

### candidateの例

```json
{
  "schema": "tei.reference.change-candidate/v0",
  "id": "cand-unit-price-positive",
  "baseModelRevision": "sha256:<revisionの出力>",
  "origin": {"kind": "cli", "actor": "developer"},
  "intent": "order lineの単価が正であることを業務ruleとして明示する",
  "evidence": [
    {"subject": "sales.order_line.quantity_positive", "note": "同じ形のruleが数量には既にある"},
    {"source": "semantic/sales/validation.tei"}
  ],
  "delta": [
    {"op": "add-validation", "namespace": "sales", "id": "sales.order_line.unit_price_positive",
     "kind": "positive", "attribute": "sales.order_line.unit_price"}
  ]
}
```

- `origin.kind`は`file` / `cli` / `studio` / `ai`。provenanceだけで、fingerprint・判定・source・Canonical Worldへ影響しません（AI固有のmodel名・prompt digest等は`origin.notes`に置けますが、どこにも保存されません）。
- 対応するoperationは次の5種です。任意のTEI textやfile patchは運ばず、sourceは意味から決定論的に生成します。Relation変更等は、このReferenceで必要になった時点で追加します。

| operation | 意味 | source |
| --- | --- | --- |
| `add-validation` | 既存namespaceへValidation（`positive` / `relation-unique`）を追加する | 既存fileの末尾へ追記（`placement`は省略可。省略時は同じnamespaceの既存Validationがあるfile） |
| `add-data` | **新しい**semantic namespaceへData（Attribute・Relation・key）を追加する | 新しいsemantic file（`placement`必須） |
| `add-relation` | **既存の**DataへRelation（key付き / 概念Relation）を一つ追加する | 対象Dataを宣言したfileのdata blockの、閉じ括弧の行の直前へ挿入（`placement`は持たない。閉じ括弧が行頭に無いlayoutは拒否） |
| `add-application` | Environment bindingを含まないApplication（semantic利用・Database requirement）を追加する | 新しい`applications/<dir>/application.tei`（`placement`必須） |
| `add-reference-pattern` | reference Patternを追加する | 新しい`applications/<dir>/patterns/<file>.tei`（`placement`必須） |

- 新しいfileを作るoperationのplacementは、意味を所有する層のprefix（semantic / applications）とloaderが読む形式に合い、まだ存在せず、他のoperationと重ならない必要があります。PatternはそのPatternを所有するApplicationのdirectoryに置きます（loaderはApplication manifestと同じdirectoryの`patterns/`をそのApplicationのものとして読みます）。
- `usesSemantic`や`databases`の順序は、authoredされたsourceの順序としてそのまま保ちます（入れ替えるとsourceとfingerprintが変わります）。
- 変わるべきWorld collectionは、deltaの種類から導きます（semantic層のoperationは`semantics`、Application層のoperationは`applications`）。実際に変わったcollectionの集合とexactに一致しなければ拒否します。
- 受理はfileを新しく作るため、失敗時は作ったfileと、そのために作ったdirectoryを削除して元へ戻します（既存fileへの挿入・追記は元のbyteへ戻します）。
- Relationの追加・変更では、Previewが端点（source / target Data）を実現するPersistence Mappingへの辺と、Relationの保存方法（relation storageが定義済み / keyのsource Attributeが列として実現済みで導ける / 未決定）を返します。未決定なら`RELATION_STORAGE_NOT_AUTHORED`として残し、この変更がPersistence Mapping・migrationを変えないことも示します。

## このtoolが示すこと / 示さないこと

示すこと（このtoolの`boundary_test.go` / `provenance_test.go` / `containment_test.go`と、`changeanalysis`の`analysis_test.go`）:

- Preview / Rejectはauthoritative sourceをbyte単位で変更しない
- 受理した意味はtoolを介さず共有loaderで再読込でき、Validationを満たし、受理前後で評価結果が変わる
- staleなcandidateを黙って適用しない
- 不正なcandidate（未知field / 存在しない参照 / 別layerへのplacement / 数値ではないAttributeへの`positive` / 宣言外の副作用）はsourceを変えずに拒否される
- どの入口のcandidateも同じfingerprint・同じsource・同じCanonical Worldになり、AI固有のEvidenceがCanonicalへ入らない
- accept途中の並行変更・受理後Conformance失敗では、書いたfileを元へ戻す
- 同じ意味を別fileへ移すだけならsemantic差分は出ず、論理的な改名はremoved / addedとして現れる
- 派生物（Canonical World JSON）を消しても、accepted sourceから同じものを再生成できる
- `ISSUE-000046`が最小loopとして名指しするRelation変更（既存Data `sales.order_line`へ`order`を追加）について、Preview（Semantic Diff・exact dependencyのimpact・Persistence Mappingへの辺と保存方法が未決定であること）→ Reject → Accept → toolなしの再読込までを通し、変わるのは一つのfileの挿入した行だけである（`relation_test.go`）
- Second Application `repair_intake`（改訂版S4 Proposal、SHA-256 `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`）を、3つの型付きoperationとして表し、source・payload・candidate identityをS4 Proposalへ結び付けられる（`composition_test.go`）
- そのcompositionをscratch copyへacceptすると、宣言した3 fileだけが作られ、Environmentは変わらず、受理後の再読込・Conformance・Previewとの差分一致が通る。並行変更・受理後Conformance失敗では、3 fileと作ったdirectoryを元へ戻す
- 同じcompositionを実際の`reference/workspace`へPreviewすると全stageがPASSし、`reference/workspace`は変更されない（Previewの結果は`plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_IMPLEMENTATION_PACKET_2026-09.md`「実装記録」）

示さないこと:

- AIがcandidateを生成できること、Studio UI、`ISSUE-000047`のknowledge-first探索との接続
- Driver scenario / Web Target Projectionへの影響の実行（Previewは`DRIVER_SCENARIO_NOT_RUN` / `TARGET_PROJECTION_NOT_EVALUATED`をunresolvedとして明示する）
- Applicationが評価するValidationの選択がCanonicalから導出されること。現行のCanonical modelとDriverは、`uses validation`宣言から評価対象を導出していない（`APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL`）
- Runtime操作のeffect-time authority、file move / renameの自動検出、Derived Experience Knowledge

## 検証

```bash
go test ./tools/reference-shared-acceptance -count=1
```

testはGate B / Gate Cのbaselineを変更せず、`reference/workspace`のscratch copyだけを使います。最初のtestは、scratch copyから得たCanonical Worldが`reference/world.json`とbyte単位で一致すること（実際のReferenceで試していること）を確認します。
