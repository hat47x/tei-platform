# reference-scoped-query-preflight（#1769）

実行前に、認可済みの行スコープを物理SQLへ適用できるかを判定する読取り専用の参照です。Scope条件は信頼済みの認可根拠から取り、クライアント指定のTenant・Scope・Roleを採用しません。Scope列・Query/Data・Provider・Field・Operationが明示的に対応付けられていない場合はSQLを発行せず拒否します。Sourceを書き換えず、SQLも発行しません。

**この参照は、実データベースに対する行スコープ強制を実証しません。** 実行前の契約だけを検査します。実DB fixtureや実接続による越境不能・安定Top・拒否時SQL未実行の実証は未達として残します。

~~~text
go run ./tools/reference-scoped-query-preflight/cmd/tei-scoped-query-preflight -input request.json
~~~

`-input`は判定の依頼を書いたJSONです。`-`で標準入力から読みます（既定）。

## 入力

- `authorization`: 信頼済みの認可根拠。`principal`、`tenant`、`subject`、`purpose`、`policyRevision`、`expired`、`scopes`
- `realization`: 明示的なScoped Query実現。`application`、`pattern`、`query`、`data`、`datasource`、`provider`、`scopeColumn`、`fields`、`operations`
- `request`: クライアント要求。`tenant` / `scope` / `role`は権威ではない。`query`、`data`、`filter`、`sort`、`limit`、`fields`、`operations`、`postFilterRows`
- `known`: ホストが許可する明示対応。`fields`、`providers`、`operations`

## 出力

- `decision.execute`: SQLを発行してよいか
- `decision.clauses`: 実行順の句（`scope` → `filter` → `order` → `limit`）
- `decision.boundParams` / `decision.identifiers`: バインド値と、明示対応から取った識別子
- `decision.postFilter` / `decision.unscopedFallback`: 常に`false`
- `decision.refusals`: 拒否の理由
- 指紋

## 守っている境界

- Scope条件は認可根拠から取ります。クライアント指定のTenant・Scope・Roleを採用せず、異なれば拒否します（`client-scope-not-authoritative`）。
- 認可失効、Scope不在は拒否します（`authorization-expired`、`authorization-scope-missing`）。
- 適切なScope列が無ければ対象を拒否します（`scope-column-missing`）。既存native SQLから物理Tenant列を推測しません。
- Query/Dataの不一致は拒否します（`query-data-mismatch`）。
- Provider、Field、Operationは明示対応にある場合だけ許可します（`provider-not-supported`、`unknown-field`、`operation-not-supported`）。Count/Aggregate/Expandは実証完了まで許可しません。
- 無スコープのProvider fallbackを作りません（`unscopedFallback`は常に`false`）。
- 無スコープ結果を後処理で削除しません（`post-filter-forbidden`）。
- 拒否時は句もバインド値も作らず、SQLを発行しません。

## 未実装・未実証

- 実PostgreSQLへのQuery実行と、2つのTenant/Scopeでの越境不能・安定Top・拒否時SQL未実行の実証
- SQL文字列とbind argsの実テスト（DB未接続のモックと区別すること）
- Canonical Result Mapping後の公開項目の絞り込みと、Raw SQL列・Provider Payloadの非漏洩の実証
- 既存DriverのQuery/Runtime、Authorization、Provider境界への接続（#1043、#1127、#1158、#179）
