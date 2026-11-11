# 2waySQL-like記述との整合性

## 現行TEIの設計

TEIでは、`query.EmbeddedSQL` と `realization.ContractQuerySQLRealization` の組み合わせで、
SQLとパラメータの対応を管理している。これは2waySQLの「SQLとパラメータの分離」
に相当する役割を果たす。

### query.EmbeddedSQL
- `Source`: 実際のSQL
- `Inputs`: パラメータ定義（名前、型）
- `ResultColumns`: 結果列のマッピング
- `Dialect`: 対象DB種別

### ContractQuerySQLRealization
- `Target`: Queryのidentity
- `Provider`: DB provider
- `Source`: native SQL
- `Inputs`: input名→物理parameter名の明示的マッピング
- `ResultColumns`: raw列→Semantic attributeのマッピング

## 2waySQLとの対応

| 2waySQL概念 | TEI相当 | 備考 |
|---|---|---|
| SQLファイル | `EmbeddedSQL.Source` | テキストとしてのSQL |
| パラメータ定義 | `EmbeddedSQL.Inputs` | 型付きパラメータ |
| 実行時バインド | `ContractQuerySQLRealization.Inputs` | input→parameter変換 |
| 結果マッピング | `ResultColumns` | 列→属性 |
| DB依存SQL | `Dialect` + `Provider` | 切替可能 |

## 既存設計との整合

- Native SQLを強制せず、共通契約からSQL生成する経路を追加した
- `EmbeddedSQL` は既存の入力元索引・解析経路を維持
- `ContractQuerySQLRealization` は明示マッピングにより暗黙の補完を禁止
- 新しいDB追加時は、既存の `EmbeddedSQL`/`ContractQuery` モデルを変更不要

## 残る検討事項

- 2waySQLの「ファイル外のパラメータ」に相当する機能は未実装
- 現在は `map[string]any` でパラメータを渡す方式
- 将来的に `.sql` ファイルの直接読み込みや、外部パラメータ定義の対応を検討
