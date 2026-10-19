# Database Contract Queryの入力不在と明示NULL（2026-10）

- 状態: **評価待ち（未採用）**。製品API・schema・受理規則の決定ではない。
- 日付: 2026-10-20
- 担当: [Issue #2056](https://github.com/hat47x/tei-platform/issues/2056)
- 関連: #179（Query契約） / #1769（SQL bind・行スコープ） / #214（外部master_data） / ADR-0007 / ADR-0039
- 参照実装: [`tools/reference-query-input-nullability/`](../../tools/reference-query-input-nullability/README.md)

## 1. 問題

Database Contract Queryの入力は、`QueryInput{Name, Type}`としてSemantic型identityを持ちますが、次の区別を持っていません。

- 入力keyが存在しない（missing）。
- 入力keyが存在し、値が明示的なJSON nullである。
- 入力keyが存在し、値がある（present。空文字を含む）。

現行の`MapRuntimeParameters`とPostgreSQL named-parameter binderは、map keyの不在をエラーにしますが、keyが存在して値がnilの場合はそのまま後続へ渡します。このため、missingと明示nullの意味差が契約に現れず、ProviderやSQLの`IS NULL`・`= NULL`・`IS NOT NULL`の扱いへ暗黙に押し込まれます。

missingを「値がないのでoptionalとして省略」と解釈するのか、「required入力の不足として拒否」と解釈するのか、明示nullを「SQL NULLとして許可」と解釈するのかは、Queryの意味に属します。Providerの都合で補完・変換してはいけません。

## 2. 検証仮説

入力ごとに次の2つを明示すれば、missingと明示nullをSQL実行前に区別できる。

- `presence`: `required`または`optional`
- `null`: `forbidden`または`allowed`

この組合せは、少なくとも次の4状態を作る。

| presence | null | keyなし | 明示null | 値あり |
| --- | --- | --- | --- | --- |
| required | forbidden | 拒否（missing） | 拒否（null禁止） | bind |
| required | allowed | 拒否（missing） | nilをbind | bind |
| optional | forbidden | 省略 | 拒否（null禁止） | bind |
| optional | allowed | 省略 | nilをbind | bind |

`presence`と`null`の既定値は、この文書では決めない。既存Queryへ一律に適用せず、契約ごと・入力ごとに明示する案として評価する。

## 3. 候補契約

参照実装のschemaは`tei.reference.query-input-nullability-contract/v0`であり、製品schemaではない。

```json
{
  "schema": "tei.reference.query-input-nullability-contract/v0",
  "id": "customer.find",
  "version": "1",
  "inputs": [
    {
      "name": "customer_id",
      "type": "customer.customer_id",
      "presence": "required",
      "null": "forbidden"
    },
    {
      "name": "nickname",
      "type": "std.text",
      "presence": "optional",
      "null": "allowed"
    }
  ]
}
```

- `type`はSemantic type identityであり、物理型・Provider型ではない。
- `presence`と`null`は入力ごとに必須とし、暗黙の既定値を作らない。
- 値はruntime入力として受け取る。JSON stringだけをpresentとして受け付け、number・boolean・object・arrayの暗黙変換は行わない。空文字はpresentである。
- 未知の入力keyは拒否する。

## 4. 実行前検査とProvider binding

参照実装は、拒否が1件でもあればSQL parameterを一切作らない。これにより、呼出し側が拒否済みのparameterを実行へ渡す経路を減らす。

- requiredのkey欠落: `MISSING_REQUIRED_INPUT`で拒否。
- null forbiddenの明示null: `NULL_NOT_ALLOWED`で拒否。
- optionalのkey欠落: parameterを作らず、SQL文側で入力自体を省略する。
- null allowedの明示null: Providerへnil（SQL NULL）として渡す。
- present: 値をそのまま渡す。型変換・coercionは行わない。

Providerごとの差は別に確認する。

- PostgreSQLのnamed parameterへnilを渡したとき、`IS NULL`条件が成立するか。
- optional欠落を、SQL NULLや空文字へ暗黙に置き換えないか。
- Driverがnilをunsupportedとして拒否しないか。
- SQLの`= NULL`ではなく`IS NULL`を使うのはQuery realizationの責務であり、この契約だけでは決めない。

## 5. 証拠の境界

参照実装が示すのは、契約とbinderの状態遷移だけである。

- SQL文の組立て・実行、PostgreSQL接続、実DBのNULL比較・型復元・行順序は未検証。
- fake binderの結果を、実PostgreSQLのNULL挙動のEvidenceにしない。
- PostgreSQL以外のProviderについて、実装済み・未確認を区別する。
- #1769の行スコープ認可・SQL組立てを再実装しない。
- #179のQuery契約全体の型体系・必須/任意の既定値を確定しない。

## 6. 代替案

- **全入力を一律nullableにする**: 必須入力の欠落と明示nullを区別できず、業務上の必須性を失うため採用しない。
- **missingを常にSQL NULLとして扱う**: 「未指定」と「NULLを指定」の意味差を潰すため採用しない。
- **Providerがmissingを補完する**: Queryの意味を実装側が変えるため採用しない。
- **Sentinel値でnullを表す**: 値型・比較・表示へ漏れ、SQL NULLと混同するため採用しない。
- **入力ごとの`presence`/`null`を明示する**: 今回の参照で状態遷移を検証する候補。既定値と全Query適用は未決。

## 7. 受入条件との対応

| #2056の確認条件 | この参照で確認できること |
| --- | --- |
| missingと明示NULLの違いを契約・実行前検査で説明 | `presence`/`null`と状態遷移で区別 |
| nullableの可否を明示し、禁止NULLをSQL実行前に拒否 | `null: forbidden`をparameter生成前に拒否 |
| 許可NULLのbinding規則を明示し、暗黙変換と区別 | nil bindと`UNSUPPORTED_VALUE`を分離 |
| missing/allowed NULL/forbidden NULLのmapping tests | 参照実装の試験 |
| Runner境界で引数または拒否を確認 | receiptの`sqlParameters`と`queryNotExecuted` |
| PostgreSQL以外の実装済み・未確認を区別 | 未検証と明記 |

## 8. 未決

- 既存Queryの`presence`/`null`をどう移行するか。既定値を置くか、未指定を拒否するか。
- 必須入力で明示nullを許可する組合せを許すか。
- null allowedの入力を、SQLで`IS NULL`と`= ?`のどちらへ実現するか。Query realizationの表現と契約の関係。
- PostgreSQL以外のProvider差を、どの実DB Evidenceで確認するか。
- Semantic typeからruntime値への許容変換を、どこで・谁が行うか。
- optional欠落をSQL文から省略する場合の、Query realizationとparameter mappingの表現。

## 9. 非目標

- この文書だけで全入力をnullable/non-nullableと決めること。
- ORM・汎用coercion・Query全体の型体系の設計。
- #1769の行スコープ認可・SQL組立ての再実装。
- 実PostgreSQLでの受理・本番配備。
