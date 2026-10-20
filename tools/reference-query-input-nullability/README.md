# reference-query-input-nullability（#2056）

Database Contract Queryの入力について、**未指定（missing）**と**明示的なJSON null**を区別し、NULL可否を契約で明示してSQL実行前に拒否する参照です。

これは採用済みの製品API・schemaではありません。#179のQuery契約、#1769のSQL bind境界を置き換えず、入力の有無とNULLの扱いだけを切り出して反証可能にします。

```text
go run ./tools/reference-query-input-nullability/cmd/tei-query-nullability \
  -contract tools/reference-query-input-nullability/testdata/contract.json \
  -request tools/reference-query-input-nullability/testdata/request.json
```

## 契約

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

- `presence: required`は入力keyが必須です。欠落は`MISSING_REQUIRED_INPUT`としてSQL実行前に拒否します。
- `presence: optional`は入力keyを省略できます。省略時はparameterを作りません。
- `null: forbidden`はkeyが存在しても値がJSON nullなら`NULL_NOT_ALLOWED`として拒否します。
- `null: allowed`は明示nullをSQL NULLとしてbindします。missingとは別の状態です。
- `presence`と`null`は入力ごとに明示します。暗黙の既定値や全Queryへの一律適用は、この参照では決めません。
- `type`はSemantic type identityです。runtime値の型変換・coercionは行いません。JSON stringだけをpresentとして受け付け、number・boolean・object・arrayは`UNSUPPORTED_VALUE`として拒否します。
- 空文字`""`はpresentな値であり、nullでもmissingでもありません。
- 契約にない入力keyは`UNKNOWN_INPUT`として拒否します。

## 状態

| 状態 | 意味 | SQL parameter |
| --- | --- | --- |
| `present` | 文字列値がある | 値をbind |
| `null` | 明示nullで許可されている | nilをbind |
| `missing` | keyがない（optionalのみ） | 作らない |
| 拒否 | required欠落、禁止null、型不一致、未知入力 | 一切作らない |

拒否時は`passed: false`、`queryNotExecuted: true`、`sqlParameters: {}`を返します。呼出し側が拒否済みのparameterを実行へ渡せない形にしています。

## 証拠の境界

- この参照は、SQL文の組立て・実行、PostgreSQL接続、実DBのNULL比較・型復元・行順序を検証しません。
- ProviderごとのNULL binding差（PostgreSQL、Oracle、SQLite等）は未検証です。
- fake binderの結果を、実PostgreSQLのNULL挙動のEvidenceとして扱いません。
- #1769の行スコープ認可・SQL組立境界を再実装しません。
- #179のQuery契約全体の型体系・必須/任意の既定値を確定しません。

## 検証

`go test ./tools/reference-query-input-nullability/... -count=1`で、次を確認します。

- missing・明示null・値の3状態を区別する。
- required欠落と禁止nullをSQL parameter生成前に拒否する。
- optional欠落はparameterを作らない。
- 許可nullはnilとしてbindする。
- 空文字をnullやmissingへ潰さない。
- JSON number等を暗黙に文字列化しない。
- 未知入力を拒否する。
- presence/nullの明示がない契約を拒否する。
- 末尾に別のJSON値がある入力を拒否する。

## 未実装・未決

- 実PostgreSQLでのNULL比較・bind・型復元のEvidence。
- PostgreSQL以外のProviderのNULL binding差。
- Query全体の必須/任意・nullableの既定値をどうするか。
- Semantic typeからruntime値への許容変換規則（必要な場合）。
- SQL組立て・認可・Runnerへの接続。
