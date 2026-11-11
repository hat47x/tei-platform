# SUI参照のCanonical定義（M2）

- 状態: 実装前の提案（未採用）。Product runtime・Canonical意味の変更ではない
- 日付: 2026-11-11
- 対象: [SUI→TEI移行 実装パケット](../design/SUI_TEI_MIGRATION_IMPLEMENTATION_PACKET_2026-11.md) の M2
- 根拠: SUI #3401、SUI PR #3428 / #3429、SUI `02_Architecture/api.md` §2.18、SUI ADR-0116
- 主張しないこと: SUIがTEIへ移管されること、この定義が製品の正式意味契約になること、instanceをCanonical objectへ昇格させること

## 1. 何を再現するか

SUIのread-only参照（`tai.reference.sui-document-reference/v0`）が運ぶ構造を、`.tei`の型宣言として参照workspaceに置きます。目的は、**consumerが型で区別しているものを、定義側でも同じ区別で宣言できるか**を確かめることです。instanceの取込や保存は扱いません。

## 2. 宣言したもの

`reference/sui-reference/workspace/semantic/sui/` に4ファイルを置きます。

| 定義 | 内容 |
| --- | --- |
| `sui.document` | `documentId`、`revisionState`、`disclosureState` |
| `sui.card` | `cardId`、`claimType`、`holdState`、`x`、`y`、`textReviewState`、`containedIn`（`one`）、`affiliatedWith`（`many`） |
| `sui.island` | `islandId`、`titleReviewState`、`parent`（`one`） |
| `sui.relation` | `relationId`、`relationType`、`fromKind`、`fromId`、`toKind`、`toId` |

型は`std.identifier`・`std.code`・`std.decimal`・`std.text`と、`sui.*`の別名だけで構成します。第二の型語彙は作りません。

## 3. 区別して宣言したこと

- **containmentとaffiliationは別Relation**。`containedIn`（`one`）と`affiliatedWith`（`many`）を別IDで宣言し、一つの`belongsTo`へ畳んでいません。`many`側だけが多対多を表します。
- **Treeは宣言しない**。`sui.island.parent`は単一の親リンクとして置き、`sui.tree`・`sui.forest`・`branch`・`child`のいずれも宣言しません。containmentをTree階層と呼ばない制約を、定義側で観測できる形にしています。
- **本文・題名はAttributeにしない**。`text`・`title`・`body`・`content`・`summary`・`narrative`を`sui.card`・`sui.island`・`sui.document`へ宣言していません。必須Attributeもありません。本文の有無はM4の開示判断の対象であり、型の必須項目ではありません。
- **x/yは`std.decimal`**。`std.number`は標準型に無いため、実在する`std.decimal`を使います。小数座標の丸め規則は決めていません。

## 4. 検証

```bash
go run ./cmd/tei validate -workspace reference/sui-reference/workspace
go test ./reference/sui-reference/... -count=1
```

- `validate`は`ok`を返しました（2026-11-11、Windows、Goツールチェーン固定）。
- `go test ./reference/sui-reference/... -count=1`は`ok`。4件の試験で、containment/affiliationの分離、Tree未宣言、本文Attribute不在、型の解決を確認しています。
- `gofmt -l reference/sui-reference`は空、`go vet ./reference/sui-reference/...`は指摘なし。
- 未実施: 実Product runtimeでの読込み、instance fixtureのCanonical化、`world.json`固定、Owner意味受理。

## 5. 未決

- 型名・namespaceをTEIの共通語彙へ昇格させるか（パケットO-08）。
- `sui.relation`を、`fromKind`/`fromId`の組合せではなく型付きRelationとして宣言できるか。現行の`relation ... to <data>`は静的な対象型を要求し、端点半区別（card/island）を一つのRelationで表せません。今回は属性で表し、この制約を記録します。
- `parent`のroot許容（親なし）は、現行Relationの`one`だけでは表現できません。SUI側の`parentIslandId: null`意味は未解決のままです（`reference/sui-adoption/README.md`の`parentPresenceSemantics = unresolved`と同じ扱い）。
