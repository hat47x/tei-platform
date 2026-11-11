# SUI参照のread-only consumer（P7）

- 状態: 実装前の参照（未採用）。Product API・共通Operation・第二の正本ではない
- 日付: 2026-11-11
- 対象: [SUI→TEI移行 実装パケット](../../plan/design/SUI_TEI_MIGRATION_IMPLEMENTATION_PACKET_2026-11.md) の P7
- 根拠: SUI #3401、SUI PR #3428 / #3429、SUI `02_Architecture/api.md` §2.18、SUI ADR-0116
- 主張しないこと: TEIの共通APIへOperationを追加すること、実SUIへのHTTP接続、Ownerの意味受理

## 1. 何をするか

SUIが返す `tai.reference.sui-document-reference/v0` のenvelopeを読み、**consumerが緩めてはいけない主張**を検査します。SUIのrouteやartifactを読むのではなく、受け取ったJSONだけを見ます。

```bash
go run ./tools/reference-sui-reference-consumer -input <reference.json>
go test ./tools/reference-sui-reference-consumer/... -count=1
```

## 2. 検査する内容

| ID | 検査 | 失敗コード |
| --- | --- | --- |
| P7-01 | `schema`が固定値 | `unsupported-schema` |
| P7-02 | 未知のtop-level keyを拒否（consumer compatibility guard） | `unsupported-schema` |
| P7-03 | `acceptedMeaning`が`false`、`semanticStatus`が`reference-only`、`authority`が`sui-document` | `accepted-meaning-claimed` |
| P7-04 | `disclosure.textIncluded`が`false` | `content-in-reference` |
| P7-05 | `disclosure.unreviewedContent`が`withheld` | `invalid-reference` |
| P7-06 | 本文・題名・要約・narrative・attribution・reviewerRefのキーが現れない | `invalid-reference` |
| P7-07 | 各`ref`が{kind,id}を持ち、Card/RelationのIDが重複しない | `invalid-reference` |
| P7-08 | `content.status`が`not-carried`/`withheld-unreviewed`、`review.text`が既知の3状態 | `invalid-reference` |
| P7-09 | `withheld-unreviewed`のCardが`reviewed`を名乗らない | `invalid-reference` |
| P7-10 | Islandの`containmentCardRefs`が既知のCardを指す | `invalid-reference` |
| P7-11 | `relationKind`が`directed`/`containment`/`affiliation`のいずれか | `invalid-reference` |
| P7-12 | directed Relationだけが`parallel`を持ち、`index`が1〜`count`、`groupRef`が空でない | `invalid-reference` |
| P7-13 | 同じ`groupRef`のoccurrence数が宣言`count`と一致し、全occurrenceの`count`が同じ | `invalid-reference` |
| P7-14 | 入力がJSON objectで、空でない | `invalid-reference` |

失敗メッセージは入力のID・本文・件数を含みません。`problems`は固定文言だけです。

## 3. 何を検査しないか

- 実SUIのHTTP応答・認可・artifactの読み取り（このtoolはファイル入力だけを扱う）。
- `revision`の正しさ、`losses`の網羅性、`completeness`の意味。
- TEIの意味受理、OwnerのAccepted Meaning、Hostの現在Grant。
- 実Provider・実DB・配備環境。

いずれも別の資格であり、本toolの`ok`から推論しません。

## 4. 試験

`main_test.go`の11件で、正常1件と、schema不正・未知key・acceptedMeaning=true・semanticStatus/authority不正・textIncluded=true・unreviewedContent不正・reviewedとwithheldの矛盾・parallel count不整合・directedのparallel欠落・未知のcontainment card・check-in済みfixtureを確認します。

- `go test ./tools/reference-sui-reference-consumer/... -count=1` → `ok`
- `gofmt -l tools/reference-sui-reference-consumer` → 空
- `go vet ./tools/reference-sui-reference-consumer/...` → 指摘なし
- 未実施: 実SUI routeの応答をこのtoolへ直接渡すE2E、実HTTP consumer、Owner受理。

## 5. 未決

- この検査をTEIの共通API Operation（#1158/#3419）へ登録するか。現状はReference toolに留めています。
- `losses`の`feature`/`status`語彙を共通契約にするか（TEI #2497）。
- 実SUI応答をファイルへ書き出す運用を誰が担うか。ADR-0114のhandoffは逆方向（SUI→外部）であり、この経路の受け渡しは未決です。
