# reference-derived-field-projection（#1707）

導出項目をモデラー・業務画面へ投影するときの**意味状態**を扱う読取り専用の参照です。意味の宣言、未保存の一時プレビュー、実行時の導出値、業務上確定した保存値を混同せず、導出値への直接書込をAction／保存境界でも拒否します。

Widgetの選択・見た目はUI側（#1124／#997／#1127）の責務です。この参照は、各表示先へ渡す共通の意味状態だけを持ちます。

## 実行

```bash
go run ./tools/reference-derived-field-projection/cmd/tei-derived-field-projection \
  -input tools/reference-derived-field-projection/testdata/preview.json
```

結果はJSONで標準出力へ出します。

## 入力

```json
{
  "field": {"id": "order-line.amount", "kind": "derived"},
  "phase": "preview",
  "operation": "display",
  "derivedState": "present",
  "derivedValue": "900",
  "savedValue": "600",
  "inputsUnsaved": true,
  "renderers": ["web", "cli"]
}
```

- `kind`は`physical`（物理保存値）・`derived`（導出値）・`confirmed`（確定保存値）
- `phase`は`declaration`・`preview`・`runtime`
- `operation`は`display`・`write`・`edit-rule`
- `derivedState`は`present`・`input-unavailable`・`authorization-denied`・`computation-failed`

## 結果

| 条件 | コード |
| --- | --- |
| 導出値の宣言 | `PROJECTION_DECLARED`（`editable=false`） |
| 物理保存値の読取り／書込み | `PHYSICAL_FIELD_READ`／`PHYSICAL_FIELD_WRITABLE` |
| 導出値への直接書込（preview・runtime のどちらでも） | `DERIVED_FIELD_READ_ONLY` |
| Previewで評価できた | `PREVIEW_PRESENT`（`savedValueUnchanged=true`） |
| Previewで入力未取得／認可拒否／計算不能 | `PREVIEW_INPUT_UNAVAILABLE`／`PREVIEW_AUTHORIZATION_DENIED`／`PREVIEW_COMPUTATION_FAILED`（`value`は空） |
| 実行時の導出値 | `RUNTIME_PRESENT` ほか `RUNTIME_*` |
| 確定保存値の読取り | `CONFIRMED_SAVED`（再計算しない） |
| 確定保存値への直接書込 | `CONFIRMED_WRITE_REQUIRES_ACTION` |
| 規則の編集 | `RULE_EDIT_REQUIRES_ACCEPTANCE`（#1124／#267の経路へ） |

## 守っている境界

- 導出値は、UIがセルを無効にしているかどうかに関係なく、`write`として拒否します。Action／保存境界からの書込みも同じ判定です
- Previewの結果は`PREVIEW_*`として返し、確定保存値（`savedValue`）は変えません。キャンセル後も`CONFIRMED_SAVED`は保存済みの値を返します
- 入力未取得・認可拒否・計算不能のときは、古い表示値を現在の結果として返しません（`value`は空）
- 確定保存値は、現在の導出値が変わっても再計算しません
- 同じ意味状態を`renderers`の各表示先へ同じ値で渡します。Web／CLI／専用UIで別の規則を持たせません

## 合成Fixture

受注明細の数量2・単価300 → 導出金額600。確定保存値600。数量を3へ未保存変更するとPreviewは900。現在の単価が350になると実行時の導出は700。

| 要求 | 結果 |
| --- | --- |
| 宣言 | `PROJECTION_DECLARED`、`editable=false` |
| Preview（数量3） | `PREVIEW_PRESENT`、`value=900`、`savedValue=600`、`savedValueUnchanged=true` |
| 導出金額セルへ`999`を書込み | `DERIVED_FIELD_READ_ONLY`（preview・runtimeとも） |
| 確定保存値の読取り | `CONFIRMED_SAVED`、`value=600`（実行時700でも変えない） |
| 単価未取得 | `PREVIEW_INPUT_UNAVAILABLE`、`value=""` |
| 規則の編集 | `RULE_EDIT_REQUIRES_ACCEPTANCE` |

## 範囲と非主張

- 合成入力に対する静的な意味の判定です。実UI・実Action・実DBの保存を観測した証拠ではありません
- Widget・画面遷移・Editor・Command／Transactionを実装しません。Source編集パイプライン・標準CRUD生成器・永続化実行器を再実装しません
- AI提案と受理（#267）、モデラー編集（#1124）、標準Function（#1127）、UI契約（#997）の既存経路を置き換えません
- 実Rendererでの表示検証は対象外で、未検証として残します
- `-race`はcgoを伴わない可搬Goでは実行していません
