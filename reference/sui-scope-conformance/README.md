# SUI向け record投影とscope conformance（M3〜M6）

- 状態: 実装前の参照（未採用）。Product API・共通Operation・第二の正本・認可Engineではない
- 日付: 2026-11-12
- 対象: [SUI→TEI移行 実装パケット](../../plan/design/SUI_TEI_MIGRATION_IMPLEMENTATION_PACKET_2026-11.md) の M3・M4・M5・M6
- 根拠: SUI #3401、SUI PR #3428 / #3429、SUI `02_Architecture/api.md` §2.18、SUI ADR-0116、TEI #2497
- 主張しないこと: 本文公開の決定、実Provider・実HTTP・実配備の検証、Owner意味受理、TEI共通型への昇格

## 1. 何を実装したか

| Packet | 実装 | 検査 |
| --- | --- | --- |
| M3 record投影 | `tools/reference-sui-record-projection/` | 入力envelopeから `tai.reference.sui-document-record/v0` を作る |
| M4 開示投影 | 同上の `disclosure` 行列 | 項目×開示先を `present`/`withheld`/`unknown`/`not-applicable` で返す |
| M5 Source権威 | 同上の `source`/`revision` 欄 | envelope宣言をそのまま使い、欠落を`active`にしない |
| M6 scope conformance | `tools/reference-sui-scope-conformance/` と `reference/sui-scope-conformance/matrix.json` | 15行の行列を、言語非依存の契約として検査する |

```bash
go test ./tools/reference-sui-record-projection/... -count=1
go test ./tools/reference-sui-scope-conformance/... -count=1
go run ./tools/reference-sui-record-projection -input tools/reference-sui-record-projection/testdata/small-document.json
go run ./tools/reference-sui-scope-conformance
```

## 2. M3: record投影

出力は次の欄を持ちます。

- `generator`: `version`、`inputSchema`、`inputDocumentDigest`、`inputSourceRevision`、`carriesContent`
- `document.ref`、`source`（`status`/`authority`/`revision`/`observedVia`）
- `revision`（`contentDigest`、`sourceRevision`、`revisionStatus`、`change`、`previousReferenceInvalidated`）
- `cards`（`ref`、`claimType`、`holdState`、`textReview`、`canvas`、`memberships`）
- `islands`（`ref`、`titleReview`、`containment`、`parentIslandRef`）
- `relations`（`ref`、`relationKind`、`fromRef`、`toRef`、`relationType`、`parallel`）
- `memberships`（containmentとaffiliationを別occurrenceとして列挙）
- `disclosure`（項目×開示先）
- `completeness`、`losses`

**本文・題名・要約は含みません。** `generator.carriesContent`は常に`false`で、`true`を名乗る生成物は検査で拒否します。

`documentDigest`と`sourceRevision`は別欄です。同じ値で埋めている場合は不合格にします（M5-T03）。

## 3. M4: 開示投影

| 項目 | SUI UI | TEI参照 | SHO下書き | 外部agent |
| --- | --- | --- | --- | --- |
| identity.and.relationShape | present | present（entity省略時はwithheld） | present | present |
| review.flag | present | present | present | present |
| content.text | present | withheld | unknown | withheld |
| content.title | present | withheld | withheld | withheld |
| review.attribution | unknown | unknown | unknown | unknown |
| canvas.coordinates | present | present（x/yのみ） | not-applicable | unknown |
| revision.and.sourceState | present | present | present | present |

- `withheld`と`unknown`を同じ値で表しません。
- 文書単位の制限（`disclosure.documentRestricted`）は、項目ごとの`withheld`ではなく**投影全体の拒否**（`document-disclosure-restricted`）です。
- entity単位の省略は、ID・件数を出さずに`identity.and.relationShape`の`withheld`へ反映します。
- TEI参照の`present`をSHO・agentへ継承しません。

## 4. M5: Source権威

- `document.status`が`active`/`unknown`なら投影を返し、`revoked`/`expired`なら`source-unavailable`で拒否します。未知の値も拒否します（`active`へ読み替えない）。
- `status`欠落は`unknown`とし、`source.authority`は`artifact-declared`、`observedVia`は`sui-reference-envelope`です。consumerが権威を名乗りません。
- `-expected-source-revision`を渡した場合、`sourceRevision`が一致しないと`stale-reference`、比較不能（`revisionStatus=unknown`や`sourceRevision`欠落）なら`revision-unresolved`です。古い参照を現在として受理しません。

## 5. M6: scope conformance

`reference/sui-scope-conformance/matrix.json` は、`SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md` 付録Aの15行を、`route` / `kind` / `tenantCrossing` / `operationBoundary` / `evidence` / `note` へ写したものです。

- セルは`pass`/`fail`/`unsupported`/`not-run`のいずれか。空欄は不合格です（passへ読み替えない）。
- `knownGaps`に6件（list、export、audit-read、import-operation-boundary、share、cache）を宣言し、それぞれ対応する行が`not-run`であることを検査します。
- `authority`は`reference-matrix`のみ許可します。行列が認可の正本を名乗る場合は拒否します。
- 現在の集計: 15行、既知の不足6件、`not-run` 11セル、`unsupported` 6セル。

**この行列は製品Gateではありません。** SUIの実試験を実行せず、付録Aの静的読取りを写したものです。実PostgreSQLのRLS試験は作業者の申告に基づき、再実行していません。

## 6. 検証

- `go test ./tools/reference-sui-record-projection/... -count=1` → ok（15件）
- `go test ./tools/reference-sui-scope-conformance/... -count=1` → ok（9件）
- `gofmt -l` 空、`go vet` 指摘なし
- CLI実行: record投影はfixtureで`ok`、scope conformanceは`ok`
- 未実施: 実SUI routeの応答を本toolへ直接渡すE2E、実HTTP consumer、実PostgreSQL、Owner意味受理

## 7. 未決（Owner判断）

- O-02（Card本文・題名の業務面公開）: 本実装は本文を返しません。公開を決める場合は別PRで、開示行列とSafeModeの条件を先に固定します。
- O-07（review attribution）: 本実装は`unknown`です。Dataとして持つかは未決です。
- M3の出力型をTEI共通型へ昇格させるか（#2488 / #2585）。
- M6の行列を製品Gateへ昇格させるか（ISSUE-000080）。
- P7consumerと本投影を、共通API Operation（#1158 / #3419）へ登録するか。
