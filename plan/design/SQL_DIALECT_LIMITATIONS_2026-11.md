# SQL Dialect Abstraction の制限事項

- **状態**: 実装済みの範囲と未対応の記録（2026-11-11 時点）
- **対象**: `internal/database/sqlgen` と各 dialect adapter
- **関連文書**: [プラグインと配布一式](../../docs/plugins.md)、[全体構成](../../docs/architecture.md)

## 1. この文書の役割

SQL方言の共通契約（`internal/database/sqlgen`）が何を吸収し、何を吸収しないかを一か所にまとめる。契約の正本は実装と、実装と表が一致することを確認する試験である。この文書はその記録であり、契約そのものの正本にはならない。

## 2. 共通契約が運ぶもの

次の構成要素は `SelectOptions` が運び、各 adapter が生成する。

| 構成要素 | 生成位置 | PostgreSQL | SQLite | MySQL |
| --- | --- | --- | --- | --- |
| CTE（`WITH`） | 文の先頭 | 対応 | 対応（`WITH RECURSIVE` も可） | 対応 |
| 名前付きウィンドウ（`WINDOW`） | `GROUP BY` / `HAVING` の後、`ORDER BY` の前 | 対応 | 対応（3.25.0 以降） | 対応 |
| 集合操作（`UNION` / `INTERSECT` / `EXCEPT`） | 文の末尾 | 対応 | 対応 | 対応 |
| `UNION ALL` 等の重複保持 | 同上（`All: true`） | 対応 | 対応 | 対応 |

ウィンドウの位置は、select list の直後に置くと構文誤りになるため、`HAVING` の後とする。SQLite の in-memory データベースで CTE・ウィンドウ・集合操作を実行する試験（`TestSQLiteExtendedFeaturesExecute`）で確認している。

集合操作は文の末尾にkeywordを出すだけであり、もう一方の select は呼出し側が続ける。このため、呼出し側が `ORDER BY` や `LIMIT` を第一select側に置いた場合、それが集合結果ではなく第一selectに束縛される。集合結果全体に対する並び順が必要なときは、共通契約ではなく native SQL realization を使う。

## 3. 未対応の構成要素

| 構成要素 | 状態 | 回避策 |
| --- | --- | --- |
| `INSERT ... ON CONFLICT`（upsert） | 未対応。共通表現がない | native SQL realization |
| 配列型と配列演算子（`ANY` / `ALL` / `ARRAY[...]`） | 未対応 | native SQL |
| JSONB 演算子（`->` / `->>` / `@>` / `?`） | 未対応。`StorageJSON` は型映射のみ | native SQL |
| `SELECT DISTINCT ON (...)` | 未対応。汎用等価物がない | native SQL |
| `TABLESAMPLE` | 未対応 | native SQL |
| `IS DISTINCT FROM` / `IS NOT DISTINCT FROM` | 未対応。NULL を比較可能な値として扱う | `IS NULL` / `IS NOT NULL` の組み合わせ |
| `LATERAL` 結合 | 未対応。PostgreSQL 固有 | native SQL |
| 再帰クエリ（`WITH RECURSIVE` の意味表現） | 再帰の構造表現は未対応。CTE 文字列として渡せる | native SQL |
| 全文検索 | 対象外。DB 固有 | provider ネイティブ機能 |
| 空間型・空間演算子 | 対象外。DB 固有 | native SQL |
| ストアドプロシージャ | 対象外 | native SQL |

## 4. DB ごとの差

| 項目 | PostgreSQL | SQLite | MySQL |
| --- | --- | --- | --- |
| プレースホルダ | `$1`, `$2`, ... | `?` | `?` |
| `RETURNING` | 対応 | 3.35.0 以降（古い版では失敗） | 非対応（生成しない） |
| `FOR UPDATE` / `FOR SHARE` | 対応 | 非対応（生成しない） | 非対応（生成しない） |
| 日時型 | `timestamptz` | `TEXT` | `DATETIME(6)` |
| 小数型 | `numeric(18,2)` | `TEXT` | `DECIMAL(18,2)` |
| JSON 型 | `jsonb` | `TEXT` | `JSON` |
| 制約違反の判定 | SQLSTATE（`pgconn.PgError`） | 結果コード | メッセージ一致（コード未公開） |

`RETURNING` は adapter が版を判定せずに生成する。SQLite 3.35.0 未満を扱う場合は、呼出し側が版を確認するか native SQL realization を使う。

## 5. 既知の制限（呼出し側の契約）

- **識別子は呼出し側が検証済みのものを渡す。** 表名・列名・別名はそのまま文字列へ埋め込む。この契約はSQL本文の検証境界ではなく、`Condition.Expression` が生のSQL fragmentを受け取る設計とも一致する。未知の入力から組み立てる経路では、native SQL realization を使う。
- **名前付きパラメータ（`:name`）は未配線。** `SelectOptions.Parameters` 等は将来のnamed bindingのための予約であり、現状の adapter は対応するプレースホルダを生成せず、値も束縛しない。構造化した `Condition` を使う。
- **`UPDATE` の代入順序は map 走査に従う。** 生成SQLの文字列順は実行ごとに変わらない保証はない（束縛値との対応は保たれる）。文字列比較に依存する試験は両順序を受ける。

## 6. 失敗の分類

データベースエラーは `sqlgen.ToDatabaseFailure` が `failure.Failure` へ変換する。ドライバーがコードを公開する場合（PostgreSQL の SQLSTATE、SQLite の結果コード）はコードを優先し、公開しない場合はメッセージ一致を fallback とする。未知のエラーは技術障害として扱い、効果不明のときは `lookup_first` に昇格する（同一 identity で照和してから再送）。
