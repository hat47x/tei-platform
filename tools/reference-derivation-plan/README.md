# Data Derivation D2：導出値を含む問い合わせ計画 Reference（#1640）

本ディレクトリは **物理SQL実行の前に、導出と集合演算の依存順を守る問い合わせ計画を合成Fixtureで検査する読取り専用の小さな参照** であり、TEIの正式なSQL最適化器、方言書換器、分散Query実行器、認可判定器、受入済みAPIではありません。実装主体は `derivationplan.go`、受入は `derivationplan_test.go`、起動点は `cmd/tei-derivation-plan/` です。`正本`を書き換える経路はありません。

## 責務と分界

- 規則の意味契約は #1637（D1）、Operatorは #988、Loweringは #1121、標準CRUD・検索は #1127、ScenarioのテストSSOTは #1180 が所有します。
- 既存 `queryresult.MapColumns` は生の列から意味属性への明示mappingとして保持します。暗黙の仮想raw列を差し込みません。
- native `.sql` はAuthoritative Sourceのまま扱います。本参照はSQLの解析情報に相当する依存順のみを判定し、SQLを根拠なく自動書換えしません。
- 項目レベル権限の結果開示とProvider実行権限は既存Authorityへ委譲します。

## 計画の依存順（守るもの）

| 順序 | 規則 |
| --- | --- |
| Derive → Filter／Sort | 導出項目は導出ステップの後にだけ使える（`DERIVED_ATTR_UNRESOLVED`） |
| Filter／Sort → Limit／Offset | 取得後に付け足した値でRow集合を作り替えない（`ILLEGAL_PLAN_ORDER`） |
| 安定ページ分割 | 方向と同値順（tie-breaker）を宣言するまで保証しない（`NO_STABLE_ORDER`） |
| Count | その時点の行集合に対して取る |

## 合成Fixture（在庫・引当済・引当可能数）

| 商品ID | 在庫数 | 引当済数 | 引当可能数（導出） |
| --- | ---: | ---: | ---: |
| A | 120 | 100 | 20 |
| B | 100 | 10 | 90 |
| C | 80 | 30 | 50 |
| D | 70 | 20 | 50 |
| E | 40 | 70 | -30 |

| 計画 | 本参照の結果 |
| --- | --- |
| Filter `>=50` → Sort `desc, id asc` | `RESULT_ROWS` `[B,C,D]` |
| … `limit 1` | `[B]` |
| … `offset 1 limit 2` | `[C,D]` |
| Filter 後 `count` | `RESULT_COUNT` `3` |
| 「在庫数降順で1件→ その後導出してSort」 | `ILLEGAL_PLAN_ORDER`（A を返せない不正な計画） |
| 同値50の二次順序を宣言しないページ分割 | `NO_STABLE_ORDER`（C/D を保証しないと診断） |
| Derive 前に導出項目を Filter | `DERIVED_ATTR_UNRESOLVED` |
| 未解決の項目・式 | `UNKNOWN_FIELD`／`COMPUTE_UNAVAILABLE` |
| 入力未取得 | `INCOMPLETE_INPUT` |
| NULL の比較 | 一致しない扱い。件数は `nullInputCount` で開示、0や未取得と混同しない |

### SQL移譲と取得後評価の適用条件（受入3）

`-mode applicability` は、計画をSQLへ委譲できるか、取得後に評価できるか、どちらも保証できないかを判定します。保証できない方式へ黙って落としません。

| 条件 | 結果 |
| --- | --- |
| NULL比較・十進精度・関連多重度・実行権限がすべて確認でき、`limit-before-derive`でない | `DELEGATE_TO_SQL` |
| SQLでは意味を保てないが、必要な行を完全かつ有界に取得できる | `EVALUATE_AFTER_FETCH`（不足条件を`reasons`に併記） |
| 完全取得ができない、取得が有界でない、`limit-before-derive`、式がSQLでも取得後でも評価不能 | `DELEGATION_UNSUPPORTED`（全不足条件を`reasons`に列挙） |

- `limit-before-derive`は、行を先に切っているため取得後評価でも修復できません
- `provider-*`の未確認は「未対応」ではなく「未証明」として扱います
- 未対応の式を別方式で黙って計算しません

多重一致の参照先の縮約禁止は #1706（D9）が所有します。本参照は多重参照の意味判定をしません。

## 実行方法

```sh
go run ./tools/reference-derivation-plan/cmd/tei-derivation-plan -input request.json

# SQL移譲と取得後評価の適用条件
go run ./tools/reference-derivation-plan/cmd/tei-derivation-plan -mode applicability \
  -input tools/reference-derivation-plan/testdata/applicability-sql.json
```

`-input -`（既定）は標準入力から読みます。入力例（`limit 1`）：

```json
{
  "rows": [
    { "id": "A", "stock": "120", "reserved": "100" },
    { "id": "B", "stock": "100", "reserved": "10" },
    { "id": "C", "stock": "80", "reserved": "30" },
    { "id": "D", "stock": "70", "reserved": "20" },
    { "id": "E", "stock": "40", "reserved": "70" }
  ],
  "derivedRules": [ { "field": "availableCount", "expression": "stock - reserved" } ],
  "steps": [
    { "op": "derive", "field": "availableCount" },
    { "op": "filter", "field": "availableCount", "comparator": ">=", "operand": "50" },
    { "op": "sort", "field": "availableCount", "direction": "desc", "tieBreaker": "id" },
    { "op": "count" }
  ]
}
```

## 限界

- 式の対応範囲は `<field> - <field>` のみ。その他の式は `COMPUTE_UNAVAILABLE` として計算不能を明示し、黙って別方式へ落としません。
- 合成入力に対する静的な判定です。実SQL方言・実Providerでの実行や、SQL／Goの実測比較の証拠ではありません。二方式が成立する範囲での同値比較は未実証として残します。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- M2（既存Driver・Query本体への接続と同値比較）は本PRの範囲外です。
