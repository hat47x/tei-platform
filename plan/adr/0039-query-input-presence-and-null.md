# ADR-0039: Query入力の不在と明示NULLを契約で分け、SQL実行前に拒否する

- 状態: 提案中
- 日付: 2026-10-20
- 関連ADR: ADR-0007 / ADR-0022 / ADR-0035
- 関連Issue: #2056 / #179 / #1769 / #214
- 関連実装: [`tools/reference-query-input-nullability/`](../../tools/reference-query-input-nullability/README.md)、[設計案](../design/DATABASE_QUERY_INPUT_NULLABILITY_2026-10.md)

## 背景

Database Contract Queryの入力はSemantic型identityを持つが、入力keyの不在（missing）と明示的なJSON nullを区別する契約表現を持たない。現行のmappingはkeyの不在をエラーにする一方、keyが存在するnilを後続へ渡す。このため、次の意味が契約に現れない。

- 必須入力の欠落を拒否するのか、optionalとして省略するのか。
- 明示nullをSQL NULLとして許可するのか、禁止するのか。
- optional欠落を、ProviderがSQL NULLや空文字へ置き換えてよいのか。

missingとnullを同じ「値なし」に潰すと、業務上の必須性、SQLの`IS NULL`条件、Provider差が暗黙になる。

## 判断

### 入力ごとにpresenceとnullを明示する

Query入力は、少なくとも次を明示する。

- `presence`: `required`または`optional`
- `null`: `forbidden`または`allowed`

暗黙の既定値をこのADRでは決めない。既存Queryへの一律適用もしない。既定値・移行は別の受理判断とする。

### 4状態を区別する

keyなし・明示null・値ありを、`presence`と`null`の組合せで次へ分ける。

- required + forbidden: 欠落もnullも拒否。
- required + allowed: 欠落は拒否、nullはnilをbind。
- optional + forbidden: 欠落は省略、nullは拒否。
- optional + allowed: 欠落は省略、nullはnilをbind。

### 拒否はSQL実行前に行う

required欠落、null禁止、未知入力、非文字列値の暗黙変換不能を検出した場合、SQL parameterを一切作らない。呼出し側が拒否済みのparameterを実行へ渡せない形にする。

### 値の型とNULLを混同しない

`type`はSemantic type identityであり、runtime値の暗黙変換規則ではない。JSON stringだけをpresentとして受け付け、number・boolean・object・arrayを暗黙に文字列化しない。空文字はpresentであり、nullでもmissingでもない。

### Providerのbinding差は証拠を分ける

null allowedのnilをProviderがSQL NULLとしてbindするか、optional欠落をSQL文から省略できるかは、Provider・Driver・実DBごとに確認する。契約の状態遷移と、実PostgreSQLのNULL比較・型復元を同じEvidenceにしない。fake binderの成功を実DBの証拠にしない。

### Queryの意味とSQL表現を分ける

`IS NULL`と`= ?`のどちらを使うかはQuery realizationの表現であり、入力のpresence/null契約から自動的に決めない。このADRはSQL組立て・行スコープ認可（#1769）を変更しない。

## 代替案

- **全入力をnullableにする**: 必須性と明示nullの区別を失うため採用しない。
- **missingをSQL NULLとして扱う**: 未指定と明示nullの意味差を潰すため採用しない。
- **Providerが欠落を補完する**: Queryの意味を実装側が変えるため採用しない。
- **Sentinel値でnullを表す**: 値型・比較・表示へ漏れ、SQL NULLと混同するため採用しない。

## 影響

- Query契約に入力ごとのpresence/null表現が必要になる。
- 既存Queryの移行規則（既定値、未指定の扱い、互換期間）を別途決める必要がある。
- mapping/binderは、拒否時にparameterを作らない境界を持つ。
- ProviderごとのNULL binding Evidenceを、実DB・版・Driver付きで記録する必要がある。
- Query realizationのSQL表現（`IS NULL`等）と契約の関係を整理する必要がある。

## 未決

- 既存Queryへ既定値を置くか、presence/nullの明示を必須にするか。
- required + null allowedを許すか。
- optional欠落をSQL文から省略する場合のparameter mapping表現。
- PostgreSQL以外のProvider差をどの実DB Evidenceで確認するか。
- Semantic typeからruntime値への許容変換をどこで行うか。

## 検証

#2056の参照実装で、次を確認する。

- missing・明示null・値の3状態を区別する。
- required欠落と禁止nullをSQL parameter生成前に拒否する。
- optional欠落はparameterを作らない。
- 許可nullはnilとしてbindする。
- 空文字をnullやmissingへ潰さない。
- JSON number等を暗黙に文字列化しない。
- 未知入力を拒否する。
- presence/nullの明示がない契約を拒否する。

この検証は、実PostgreSQLのNULL挙動、Provider間同等性、Query契約全体の受理、本番配備を証明しない。効果が確認できない場合は、presence/nullの表現を参照実装の範囲へ戻す。
