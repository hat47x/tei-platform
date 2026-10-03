# reference-shared-acceptance

`ISSUE-000027`（AI Proposal / Acceptance）と`ISSUE-000046`（Studio Shared Acceptance）が共有する、**Shared Acceptance Boundaryの決定論的なReference実装**です。

file edit・CLI・Studio・AIのどの入口から来た変更candidateも、同じ検査を同じ順序で通り、**acceptだけがauthoritative sourceを書き換えます**。AI専用・Studio専用の受入経路はありません。

## 位置づけ

- Reference専用のtoolです。Core primitive、platform共通のProposal schema、Patch DSLではありません。candidateの形式（`tei.reference.change-candidate/v0`）はこのReferenceの実測から決めた運搬形式で、固定を主張しません。
- 差分・依存影響の解析はread-onlyな[`tools/internal/changeanalysis`](../internal/changeanalysis)（`ISSUE-000039`の最初の解析境界）が担当し、このtoolはそれを呼びます。受理・source反映（`ISSUE-000027`）と解析を混ぜず、解析側はsourceを読み書きしません。
- AIによる候補生成、Studio UI、read-only探索そのものは含みません。これらはcandidateの生成側です。一方、`ISSUE-000047`で`applicable`になったqualification結果は、exactなsubject・source・fingerprint・requirementsをEvidenceとしてcandidateへ付与できます。requirementsが明示的なdependencyを含む場合は、そのdependencyのauthoritative sourceとfingerprintも必要な範囲だけ保持します。このtoolはPreview時に同じrequirementsで再qualificationし、現在も適用可能かを確認します。
- Gate B / Gate CのBaseline Evidenceは、その判定時のrevisionに固定されたhistorical Evidenceです。current `reference/workspace`を永久凍結する意味ではありませんが、baseline名のworkspaceへの誤書込みを避けるため、`accept`は`-allow-reference-baseline`が無い限り拒否します。通常はscratch copyで検証し、post-Gate-CのintentionalなReference更新だけをexplicit decision + flagで行います。Second Applicationの別生成AIレビューは2026-09-21に完了し、`review-clear-with-conditions`となりました。実際の`reference/workspace`へのacceptには、なおexact Proposalに対するexplicit human Acceptance decisionが必要です（`ISSUE-000040`）。

## 受入境界

```text
decode → normalize → base-revision → resolve → apply(overlay)
       → reload → validate → architecture-policy → semantic-diff
       → impact → acceptance-policy
```

| stage | 確認すること |
| --- | --- |
| decode | 未知fieldを許さず厳密に読む |
| normalize | 正規形へ揃え、意味を持つ部分（base revisionとdelta）だけの`candidateFingerprint`を作る。Origin / Intent / Evidenceは含めない。qualification付きEvidenceはrequirementsとrequired dependency Evidenceの対応を検証し、Evidence集合をdedup・sortして入口差を除く。正規化済みEvidenceには別の`evidenceFingerprint`を付け、意味identityと根拠identityを混ぜない |
| base-revision | candidateの`baseModelRevision`が現在のsourceと一致する。staleなら黙って適用せず、reject / 再読込して再生成 / 明示rebase / 手動conflict解消へ分岐する |
| resolve | 対象namespace・Attribute・Relationが存在し、追加するidが未使用で、Evidenceがcurrent Source Indexへ解決できることを確認する。qualification付きEvidenceはsubject kind・source・fingerprintを照合し、同じrequirementsで再qualificationする。明示required dependencyがある場合は、そのauthoritative sourceとfingerprintもcurrent qualificationと一致する必要がある。placementは意味を所有する層に限る |
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
go run ./tools/reference-shared-acceptance attach-qualification -candidate FILE -qualification FILE [-out FILE]
go run ./tools/reference-shared-acceptance revision -workspace DIR
go run ./tools/reference-shared-acceptance preview  -workspace DIR -candidate FILE [-out FILE]
go run ./tools/reference-shared-acceptance review   -workspace DIR -candidate FILE [-format text|json] [-out FILE]
go run ./tools/reference-shared-acceptance accept   -workspace DIR -candidate FILE [-derived-out FILE] [-out FILE]
go run ./tools/reference-shared-acceptance reject   -workspace DIR -candidate FILE [-reason TEXT] [-out FILE]
go run ./tools/reference-shared-acceptance decisions -workspace DIR [-out FILE]
```

- `attach-qualification`: `knowledge-qualify`が返した`applicable`な結果だけをcandidateのEvidenceへ付与します。candidateのdeltaやfingerprintは変えません。`incompatible`と`unknown`は付与できません。requirementsは添付時に正規化・検証し、`expectedKind`とEvidenceの`subjectKind`が一致しない場合もPreview前に拒否します。required dependencyごとのauthoritative Evidenceも必須で、requirementsに無いdependency Evidenceは拒否します。同値なEvidenceは正規化後にdedup・sortされるため、再添付は冪等です。
- `revision`: workspaceの現在のmodel revision（管理対象source全fileの相対pathと内容fingerprintから導出）を出力します。candidateの`baseModelRevision`にはこれを使います。
- `preview`: sourceを一切書き換えずに、各stageの結果・Semantic Diff・impact・unresolvedを出力します。受理可能でなければ終了コード1です。
- `review`: `preview`と同じread-only評価を使い、承認時に確認する内容をA1〜A8の順で表示します。既定は人が読むtextで、`-format json`を指定すると同じ内容を機械可読で出力します。権限差分とnative codeの残存箇所は現時点で専用評価がないため、推測で「変更なし」とせず`未評価`と表示します。
- `accept`: 同じ検査を最初から実行し直し、通った場合だけsourceを更新します。更新後はtoolを介さない共有loaderで再読込し、Previewが示した差分どおりか・Validationが通るか・派生物を決定論的に再生成できるかを確認します。一つでも満たさなければ元のsourceへ戻します。`-derived-out`にaccepted sourceから再生成したCanonical World JSONを書き出せます。
- `reject`: sourceを変更しないことを記録します。staleなcandidateも退けられます。workspaceへは何も書きません。
- `decisions`: 受理した決定の記録を、現在のworkspaceと照合して出力します。workspace全体のrevision、受理時に書いたsource、受理理由に使ったEvidenceの現在性を別々に確認します。

### 人が読むReview Projection

`review`は、AIの要約を新しい根拠にせず、Shared Acceptance Previewが持つ差分・影響・診断・未解決点・Evidence・source editをA1〜A8へ並べ直します。`candidateFingerprint`は意味のidentity、`evidenceFingerprint`は正規化済み根拠のidentityとして別々に表示します。A7ではEvidenceと変更予定sourceへ戻れる情報を出します。

A5の権限変更とA6のnative code残存箇所は、現在のPreviewだけでは完全に評価できません。`review`は、関連するunresolvedや`customGoBindings` collectionの状態を補助情報として示しますが、専用評価の代わりにはしません。情報がない状態を「変更なし」と読み替えないため、coverageは`未評価`のまま表示します。

このProjectionはread-onlyであり、承認そのものでもAcceptance authorityでもありません。A1〜A8が承認条件として十分かはISSUE-000071のowner reviewが必要で、Reference toolとしての統合にはexact-headのfull Repository-health PASSが必要です。

### 受理の記録 — なぜそのsourceがあるのか

`accept`は、authoritative sourceを更新したあと、その受理を`.tei/decisions/<内容のSHA-256>.json`へ残します。数か月後や別sessionのAIが、会話履歴を持たずに「このsourceはなぜここにあるのか」を機械的に読めるようにするためです（`RQ-027`）。

記録には、fingerprintへ入らない`intent`と`evidence`、候補の同一性（`candidateFingerprint`）、基準と受理後のrevision、書いたfileとその前後のfingerprint、Semantic / Application差分、受理後Conformance、残ったunresolvedが入ります。

```bash
go run ./tools/reference-shared-acceptance decisions -workspace DIR
```

`decisions`は記録の内容を信用して意味を組み立て直さず、現在のworkspaceと照合した結果だけを付けます。

- `status`: `current`（記録時のrevisionが現在と一致）か`superseded`（workspaceが先へ進んだ）
- `sourceEdits`: `intact`（書いたfileが今も同じ内容）、`diverged`（変わった。`divergedPaths`に列挙）、`none`
- `evidenceStatus`: `current`（受理理由に使ったEvidenceがcurrent Source Indexとqualificationに一致）、`diverged`（根拠sourceや明示dependencyが変わった、または保存済みEvidence本文が`evidenceFingerprint`と一致しない）、`none`
- `evidenceDiagnostics`: `diverged`の理由。過去のAcceptanceを現在のapplicabilityとして読み替えないための照合結果
- 読めない記録があれば、黙って飛ばさずerrorにします

この記録はauthorityではありません。`.tei/`はmanaged sourceではないため、記録を置いてもmodel revisionは変わらず、全部消してもaccepted meaningは変わりません。偽の記録を置いても意味は動かず、照合結果が食い違いとして出ます（`record_test.go`）。`reject`は記録を残しません。`RQ-027`が機械到達を求めるのはaccepted Decisionだからです。

### knowledge qualificationからcandidateへの接続

`tei knowledge-qualify`で候補を`applicable`と判定した場合、結果にはauthoritativeなsubject kind、Source Index identity、source path、source fingerprintが含まれます。required dependencyを明示した場合は、そのdependencyについてもDefinition / Authoring sourceとfingerprintを`dependencyEvidence`へ含めます。`attach-qualification`は、そのEvidenceと判定時のrequirementsを既存candidateへ付与します。requirementsは同じ共有正規化を通すため、空の`expectedKind`、不正なdependency指定、重複dependency、`expectedKind`と`subjectKind`の不一致をgrounded candidateとして出力する前にfail closedで拒否します。required dependencyのEvidenceが欠ける場合や、requirementsに無いdependency Evidenceが混入した場合も拒否します。

```bash
go run ./cmd/tei knowledge-qualify \
  -workspace DIR \
  -subject-kind application-pattern \
  -subject-id order_entry.customer_reference \
  -qualification-requirements requirements.json \
  > qualification.json

go run ./tools/reference-shared-acceptance attach-qualification \
  -candidate candidate.json \
  -qualification qualification.json \
  > grounded-candidate.json

go run ./tools/reference-shared-acceptance preview \
  -workspace DIR \
  -candidate grounded-candidate.json
```

Previewでは、添付された結果をそのまま信用しません。current Source Indexに対してexact subjectとsourceを解決し、source fingerprintを照合したうえで、同じrequirementsを使ってqualificationを再実行します。required dependencyがある場合は、そのauthoritative sourceとfingerprintの集合もcurrent qualificationと照合します。したがって、dependencyのIDが同じままsource内容だけが変わった場合もstaleとして拒否できます。一方、requirementsが参照していないsourceの変更はqualification freshnessへ持ち込まず、repository-wideなglobal revisionは要求しません。

このEvidenceはcandidateの意味を変更しないため、`candidateFingerprint`には入りません。一方、Preview / Decisionは正規化済みEvidenceの`evidenceFingerprint`を別に持つため、「同じ意味だが根拠が違う」candidateを機械的に区別できます。Accepted DecisionにはEvidence本文とそのidentityがprovenanceとして残りますが、Acceptance、write authority、Runtime authorityを与えません。 Decision Indexは新しい記録について保存済み`evidenceFingerprint`とEvidence本文も照合し、記録だけを書き換えた場合を`EVIDENCE_FINGERPRINT_MISMATCH`として検出します。
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
- どの入口のcandidateも同じ`candidateFingerprint`・同じsource・同じCanonical Worldになり、AI固有のEvidenceがCanonicalへ入らない。同値なEvidenceは同じ`evidenceFingerprint`へ正規化され、Evidenceだけを変えてもsemantic fingerprintは変わらない
- accept途中の並行変更・受理後Conformance失敗では、書いたfileを元へ戻す
- 同じ意味を別fileへ移すだけならsemantic差分は出ず、論理的な改名はremoved / addedとして現れる
- 派生物（Canonical World JSON）を消しても、accepted sourceから同じものを再生成できる
- Environment binding: 既存のEnvironment fileへ`bind <application>.<alias> -> <instance>`を一行挿入します。Environment blockの閉じ括弧の直前へ入れ、内側のblock（`database-instance { … }`等）と取り違えません。挿入先は、そのEnvironmentを宣言しているfileに限ります。
- `ISSUE-000046`が最小loopとして名指しするRelation変更（既存Data `sales.order_line`へ`order`を追加）について、Preview（Semantic Diff・exact dependencyのimpact・Persistence Mappingへの辺と保存方法が未決定であること）→ Reject → Accept → toolなしの再読込までを通し、変わるのは一つのfileの挿入した行だけである（`relation_test.go`）
- Second Application `repair_intake`（改訂版S4 Proposal、SHA-256 `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`）を、3つの型付きoperationとして表し、source・payload・candidate identityをS4 Proposalへ結び付けられる（`composition_test.go`）
- そのcompositionをscratch copyへacceptすると、宣言した3 fileだけが作られ、Environmentは変わらず、受理後の再読込・Conformance・Previewとの差分一致が通る。並行変更・受理後Conformance失敗では、3 fileと作ったdirectoryを元へ戻す
- 同じcompositionを実際の`reference/workspace`へPreviewすると全stageがPASSし、`reference/workspace`は変更されない（Previewの結果は`plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_IMPLEMENTATION_PACKET_2026-09.md`「実装記録」）
- `ISSUE-000047`の`applicable`なqualificationをexact Evidenceとしてcandidateへ付与し、Preview時にcurrent Source Indexと同じrequirementsへ戻って再qualificationできる。required dependencyのauthoritative source / fingerprintもscoped Evidenceとして照合し、同じIDのdependency sourceが変わったcaseをfail closedにできる。requirements外のdependency Evidenceはhandoffしない（`qualification_handoff_test.go`）
- 受理した決定が、会話履歴なしでworkspaceから読める。Decision Indexはworkspace全体の`current / superseded`、そのDecisionが書いたsourceの`intact / diverged`、受理理由に使ったEvidenceの`current / diverged`を分けて返す。記録はauthorityではなく、消してもmodel revisionもaccepted meaningも変わらない（`record_test.go`）
- Environment binding（`add-environment-binding`）をApplication意味の変更と分けて扱える。差分は`environmentDiff`へ出て意味の差分は空のまま、変わるcollectionは`environments`だけ、impactはEnvironment / Application / database instanceへexact identityで辿り、権限が付かないことを`RUNTIME_AUTHORITY_NOT_GRANTED`として残す（`environment_test.go`）
- そのbindingでも、層をまたぐ正しさ（unknown alias / unknown instance / target mismatch）は境界が再実装せず、共有Validationのdiagnosticとして`validate` stageに現れる

示さないこと:

- AIやStudioがcandidateのdeltaを生成できること、knowledge-firstのretrievalやranking自体
- Driver scenario / Web Target Projectionへの影響の実行（Previewは`DRIVER_SCENARIO_NOT_RUN` / `TARGET_PROJECTION_NOT_EVALUATED`をunresolvedとして明示する）
- Applicationが評価するValidationの選択がCanonicalから導出されること。現行のCanonical modelとDriverは、`uses validation`宣言から評価対象を導出していない（`APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL`）
- Runtime操作のeffect-time authority、file move / renameの自動検出、Derived Experience Knowledge
- 決定記録が`RQ-027`を閉じること。この境界を通らずに変更したsourceは記録を残さず、`Variation`はまだ表現を持たず、Gate Evidence / Release locatorは索引化されていない。Coreにも同等の仕組みはない

## 検証

```bash
go test ./tools/reference-shared-acceptance -count=1
```

testはGate B / Gate Cのbaselineを変更せず、`reference/workspace`のscratch copyだけを使います。最初のtestは、scratch copyから得たCanonical Worldが`reference/world.json`とbyte単位で一致すること（実際のReferenceで試していること）を確認します。
