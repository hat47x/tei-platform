# `desired-schema` は「マッピングが無い」と「表が無い」を区別しない（2026-11-18、第105ラウンド）

- 起票: **#5508 を新規に起票**（本ラウンドで検出）
- 前段: [第104ラウンド](../TEI_DOGFOODING_ROUND104_ISSUE5228_CRITERIA_2026-11-18/README.md)
- 状態: **未使用の面 `desired-schema` を使い、設計者が次に書くべきものを答えられない箇所を見つけた。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。未使用だった面 **`tei desired-schema`** を使い、**自分の equip 設計に足りないものを答えられるか**を問うた。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `5b0a4779f`（`origin/main`） |
| CLI実行物 | `sha256:19c22c577790505a69b690e04c3da8fb8404650f00cf1d3faeb97042d4792bff` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 動くことは確認した

参照ワークスペースの `sales_core` では、**2テーブル・列・主キーまで含む JSON** が返る。`equip` に正しい永続化マッピングを書けば、**自分のテーブルも同じ形で出る**:

```json
{"database":"equip_core","tables":[{"schema":"public","name":"loans","kind":"table",
 "columns":[{"name":"loan_id","physicalType":"pg_catalog.varchar","nullable":false,"ordinal":1}, ...],
 "primaryKey":["loan_id"]}]}
```

**`-format sql` も同じ定義から作られる**（`#2586` の設計）。

## 4. 見つけた欠落

**マッピングを1つも書いていない `equip_core`** では:

```text
$ tei desired-schema -workspace W -database equip_core -provider postgres
{
  "database": "equip_core"
}
exit=0
```

**`tables` すら現れない**（`omitempty`）。呼び出し側は次を区別できない:

| 実際の状態 | 出力 |
| --- | --- |
| まだ `persist` を1つも書いていない | `{"database": "equip_core"}` |
| 書いたが表が0になる設計 | **同じ** |

`validate` も**これを警告しない**（`UNBOUND_DATABASE_REQUIREMENT` は環境束縛の話で、マッピングの有無とは別）。

**「次に何を書くべきか」を問うたのに、答えは「空です」だけ**であり、**空である理由**はどこにも無い。

## 5. 途中で2回、自分の書き方が違った

| 試行 | 結果 | 学び |
| --- | --- | --- |
| `persist equip.loan { table equip_core.loans { ... } }` | `expected database, semantic, table, column, ...` | **persist の本体は `database`/`semantic`/`table`/`column` の並び** |
| `persist equip.loan { database equip_core ... }` | `PERSISTENCE_MAPPING_DATABASE_NAMESPACE_MISMATCH` + `PERSISTENCE_UNDECLARED_SEMANTIC_DEPENDENCY` | **マッピングIDはデータベース名前空間に入る**（`equip_core.loans`）。**database 側に `uses semantic equip` が要る** |
| `persist equip_core.loans { ... }` + `uses semantic equip` | **ok** | 上記2件を直した |

**この2件のエラーは、どちらも直し方を名指ししていた。** 「outside its namespace」と「declare uses semantic equip」— **私が次に何をすべきか分かった。**

## 6. 起票

**#5508** として起票した。内容:

- `desired-schema -format json` が、**そのデータベースに永続化マッピングが1件も無い場合**、`{"database": ...}` のみを返し `exit 0`。
- これは「表が0件の設計」と**同じ出力**であり、**「まだ書いていない」と「書いたが空」を区別できない**。
- 提案（案を3つ、実装は次ラウンド以降）: (A) `mappingCount` を追加、(B) `tables` を常に出す、(C) 0件のとき `validate` が警告。

**設計判断を伴う**（`desired-schema` は公開契約の envelope を持たない raw 出力であり、欄の追加が読み手に与える影響を決める必要がある）ため、**本ラウンドでは起票までとし、実装はしない。**

## 7. NOT RUN

- 実 PostgreSQL に対する DDL 適用（`-format sql` の文は生成したが**実行していない**）。
- `-format sql` が空スキーマで何を出すか（今回 `equip_core` では**何も出なかった**。これは起票の対象に含めた）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 8. 位置付け

**「設計者が次に書くべきものを答えられるか」を問うた回**である。**答えは「半分」**だった — 正しく書けば自分のスキーマが出るが、**書いていない状態を「空のスキーマ」としか言わない**。**直前の2エラーは直し方を名指ししていた**のに、**0件のときだけ何も言わない**という非対称を見つけた。
