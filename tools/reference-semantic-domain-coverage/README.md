# reference-semantic-domain-coverage（#2095）

意味の定義と物理項目・要素の対応について、次の3点を機械で確かめる読取り専用の参照です。

- 要素から意味の定義へ遡る道が一意か。複数の受理済み定義に一致する要素を警告にする
- 定義が無い要素を、AIの推測で埋めずに一覧へ出す
- 実装から分かる「何であるか（What）」と、人が決めた「なぜそうか（Why）」を、要素ごとに区別して記録する

Issue [#2095](https://github.com/hat47x/tei-platform/issues/2095)の最小PoCに沿った参照です。受理済みの意味を書き換えず、識別子も変更しません。

## 実行

```bash
go run ./tools/reference-semantic-domain-coverage/cmd/tei-semantic-domain-coverage \
  -input tools/reference-semantic-domain-coverage/testdata/input.json
```

`-input`は必須です。結果はJSONで標準出力へ出します。入力が矛盾しているとき（識別子の重複、定義が存在しない要素を指す、schema違い）だけ終了コード1です。それ以外は`passed`または`warning`で終了コード0です。

## 入力

```json
{
  "schema": "tei.reference.semantic-domain-coverage-input/v0",
  "elements": [
    {"id": "sales.order.total", "name": "total", "what": "sum(line.amount)", "why": "受注金額の合計"}
  ],
  "domains": [
    {"id": "domain.money", "name": "金額", "elements": ["sales.order.total"], "accepted": true}
  ]
}
```

- `elements[].id`は正本の識別子です。検査は変更しません
- `what`は実装から分かる説明、`why`は人が決めた意味です。空は未記入として扱います
- `domains[].accepted`が偽の定義は**提案**であり、定義として数えません（#267、P11）

## 報告

| 項目 | 内容 |
| --- | --- |
| `status` | `passed`（一意）／`warning`（複数の定義に一致する要素がある）／`rejected`（入力が矛盾） |
| `elements[].status` | `typed`（受理済み定義が1つ）／`untyped`（無い）／`ambiguous`（2つ以上） |
| `elements[].domains` | 一致した受理済み定義のID（`ambiguous`の根拠） |
| `whatWhy[]` | 要素ごとの`what`・`why`の有無 |
| `problems[]` | 入力の矛盾（`rejected`の理由） |

`untyped`と`why`の欠落は失敗ではありません。次に書くべきものとして返します。

## 範囲と非主張

- `testdata/input.json`は合成の12要素です。実在のReferenceのDataを写したものではありません
- 意味の同一性、定義の正しさ、AIが提案した定義の妥当性は判定しません。件数と対応関係だけを返します
- 受理済み定義を追加したときの影響は、要素ごとの結果の差分で確かめます（試験`TestAddingDomainOnlyAffectsItsElements`）。定義の採用そのものはOwnerの判断です
- 実画面、Studio、実DB、実Providerの意味は対象外です
- 第二の意味正本、Core enum、定義の自動補完は作りません
