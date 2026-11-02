# reference-provider-override-update（#2091）

提供元（標準プラグインなど）の定義が新しい版を出したとき、案件側の上書きと追加を**知らせないまま失わない**ための読取り専用の参照です。旧版・新版・案件側の3つを正本識別子ごとに比べ、`unchanged` / `applicable` / `conflict` / `unresolved` に分類します（[`KNOWLEDGE_VARIATION_DESIGN_2026-10.md`](../../plan/design/KNOWLEDGE_VARIATION_DESIGN_2026-10.md) 4章の語に合わせます）。

定義を書き換えず、更新を自動適用しません。`applicable`でも、提案を作るところまでです。

## 実行

```bash
go run ./tools/reference-provider-override-update/cmd/tei-provider-override-update \
  -input tools/reference-provider-override-update/testdata/input-applicable.json
```

結果はJSONで標準出力へ出します。`conflict`または`unresolved`のときは終了コード1、`unchanged`または`applicable`のときは0です。

## 入力

```json
{
  "schema": "tei.reference.provider-override-update-input/v0",
  "provider": {
    "id": "std.message",
    "fromVersion": "1.0.0",
    "toVersion": "1.1.0",
    "base": [{"id": "std.message.order.submitted", "kind": "message", "fields": {"text": "...", "description": "..."}}],
    "next": [{"id": "std.message.order.submitted", "kind": "message", "fields": {"text": "...", "description": "..."}}]
  },
  "case": {
    "baseVersion": "1.0.0",
    "overrides": [{"element": "std.message.order.submitted", "fields": {"text": "案件の文言"}}],
    "additions": [{"id": "std.message.case.received", "kind": "message", "fields": {"text": "案件で追加"}}]
  },
  "displayOnlyFields": ["x", "y"]
}
```

- `case.baseVersion`は`provider.fromVersion`と一致する必要があります。一致しない場合は`unresolved`です
- `displayOnlyFields`は、座標・色など意味の衝突に数えないフィールドです

## 分類

| 分類 | 条件 | 例 |
| --- | --- | --- |
| `unchanged` | 案件に関係する意味の変更が無い | 案件が上書きしていない要素の変更、表示だけの変更、案件だけの追加 |
| `applicable` | 提供元の変更が案件の上書きに触れない、または案件が上書きしない要素の変更 | 案件が`text`を上書きし、提供元が`description`を変えた |
| `conflict` | 提供元が案件の上書きした同じフィールドを変えた、上書き対象を削除した、案件の追加と名前が衝突した | 提供元が`text`を変え、案件も`text`を上書きしている |
| `unresolved` | 比較に必要な情報が足りない | 版の不一致、上書き対象やフィールドが無い、識別子の重複 |

全体の`status`は、要素の分類のうち最も重いものです（`unresolved` > `conflict` > `applicable` > `unchanged`）。

## 報告

- `elements[]` — 要素ごとの分類、提供元の変更種別（`unchanged` / `changed` / `added` / `removed` / `missing`）、変わったフィールド、案件が上書きしたフィールド、理由
- `nameCollisions[]` — 案件の追加が提供元の追加・既存要素と同じ識別子を使った場合（`case-addition-vs-provider-addition` / `case-addition-vs-provider-element`）
- `blocked` — `conflict`または`unresolved`で真。真のとき`after`を返しません
- `before` / `after` — 更新前後の状態のスナップショット（正規化して要素ID順）。`after`は`applicable`以下のときだけ

`Rollback(plan)`は`before`を返します。更新を取り消して元の状態に戻せることを、状態の比較で確かめられます。

## 規則

- `applicable`でも自動適用しません。受理は人が行います
- 削除された要素への上書きを、知らせないまま捨てません。`conflict`として報告します
- 識別子の変更を名前変更と推測しません。別の識別子として「削除と追加」で扱います
- 表示だけの変更は、意味の衝突に数えません

## 範囲と非主張

- 入力は合成の例です。実在の標準プラグインの定義を写したものではありません
- 定義を書き換えず、更新を適用しません。受理・権限・配備の判断もしません
- Provider・Host・Authorityとの接続、実データの移行、生成物の再生成は対象外です
- `applicable`は「提案できる」ことを示すだけで、意味の受理や現在の権限を示しません
